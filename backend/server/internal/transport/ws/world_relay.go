package wstransport

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/idgen"
	"pocket-pet-remake/server/internal/platform/telemetry"
	"pocket-pet-remake/server/internal/protocol"
)

// 只保存远端在线展示节点，源数据/位置仍归属数据库和 Redis；连接属于当前实例会话服务。
type remoteScenePlayer struct {
	owner string
	scene uint32
}
type worldRelay struct {
	// 故障后的旧远端坐标不能被重新连接快照复用，下一次租约核对先重建远端图。
	invalid atomic.Bool
	origin  string
	bus     world.SceneEventBus
	queue   chan world.SceneEvent
	mu      sync.Mutex
	remote  map[uint64]remoteScenePlayer
	handler *WorldHandler
}

// 启动配置关闭时完全沿用原单实例路径；开关启用但缺少正式 Redis Stream 能力则明确失败。
func (h *WorldHandler) ConfigureWorldRelay(bus world.SceneEventBus) error {
	if h.worldService == nil || !h.worldService.AOIConfigSnapshot().RelayEnabled {
		return nil
	}
	if bus == nil {
		return errors.New("world relay requires Redis Streams")
	}
	origin, err := idgen.RandomHex(16)
	if err != nil {
		return err
	}
	h.relay = &worldRelay{origin: origin, bus: bus, queue: make(chan world.SceneEvent, 256), remote: make(map[uint64]remoteScenePlayer), handler: h}
	return nil
}

// 消费、发布和租约校对共同随应用退出，不在每个包创建 goroutine，不无限积压事件。
func (h *WorldHandler) RunWorldRelay(ctx context.Context) {
	if h.relay == nil {
		return
	}
	r := h.relay
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for ctx.Err() == nil {
			err := r.bus.Consume(ctx, func(e world.SceneEvent) { r.receive(ctx, e) }, r.reset)
			if ctx.Err() != nil {
				return
			}
			log.Printf("world relay consume failed: %v", err)
			r.reset()
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	go func() {
		defer workers.Done()
		ticker := time.NewTicker(h.sessionService.HeartbeatInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				scenes := make(map[uint32]struct{})
				h.presenceMu.RLock()
				for id, scene := range h.playerScenes {
					if h.sessionService.ConnectionByPlayerID(id) != nil {
						scenes[scene] = struct{}{}
					}
				}
				h.presenceMu.RUnlock()
				for scene := range scenes {
					h.syncRelayScene(ctx, scene)
				}
				r.mu.Lock()
				for id, current := range r.remote {
					if _, ok := scenes[current.scene]; !ok {
						r.removeRemote(id, current)
					}
				}
				r.mu.Unlock()
			}
		}
	}()
	defer workers.Wait()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-r.queue:
			publishCtx, cancel := context.WithTimeout(ctx, time.Second)
			err := r.bus.Publish(publishCtx, event)
			cancel()
			if err != nil && ctx.Err() == nil {
				r.reset()
			}
		}
	}
}

// 本机先正常投递，跨实例仅进有界 FIFO；事件丢失/队列溢出通过原重连快照恢复，不能假装成功。
func (h *WorldHandler) publishSceneEvent(scene uint32, id uint64, packet *protocol.Packet, ownerOverride ...string) {
	if h.relay == nil || packet == nil || id == 0 {
		return
	}
	h.presenceMu.RLock()
	owner := h.playerOwners[id]
	h.presenceMu.RUnlock()
	if len(ownerOverride) > 0 {
		owner = ownerOverride[0]
	}
	if owner == "" {
		return
	}
	event := world.SceneEvent{Origin: h.relay.origin, Owner: owner, SceneID: scene, PlayerID: id, Command: packet.Cmd, Body: append([]byte(nil), packet.Body...)}
	select {
	case h.relay.queue <- event:
	default:
		h.relay.reset()
	}
}

// 漏事件必须关闭本机观察连接，让既有 reconnect/world snapshot 清除残留并读取最新源数据。
func (r *worldRelay) reset() {
	if r.invalid.CompareAndSwap(false, true) {
		telemetry.Default.Record("world.relay_failure", "events", 1, true)
	}
	for _, current := range r.handler.sessionService.ListConnected() {
		if current.Conn != nil {
			if conn, ok := current.Conn.(interface{ startClose(string) }); ok {
				conn.startClose("")
			} else {
				_ = current.Conn.Close()
			}
		}
	}
}

// 远端事件只在共享 owner 匹配时更新；离开还要匹配远端当前场景，不能删除切图后的节点。
func (r *worldRelay) receive(ctx context.Context, event world.SceneEvent) {
	if event.Origin == r.origin || event.PlayerID == 0 || event.Owner == "" || len(event.Body) > 1<<20 {
		return
	}
	h := r.handler
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.Command == protocol.CmdEntityLeavePush {
		if current, ok := r.remote[event.PlayerID]; ok && current.owner == event.Owner && current.scene == event.SceneID {
			r.removeRemote(event.PlayerID, current)
		}
		return
	}
	if event.Command != protocol.CmdEntityEnterPush && event.Command != protocol.CmdEntityMovePush {
		return
	}
	owner, err := h.worldService.SceneOnlineOwner(ctx, event.PlayerID)
	if err != nil || owner != event.Owner {
		return
	}
	if local := h.sessionService.ConnectionByPlayerID(event.PlayerID); local != nil {
		if local.ID() != owner {
			_ = local.Close()
		}
		return
	}
	if current, ok := r.remote[event.PlayerID]; ok && (current.owner != owner || current.scene != event.SceneID) {
		r.removeRemote(event.PlayerID, current)
	}
	relevant := false
	for _, id := range h.playerIDsInScene(event.SceneID, event.PlayerID) {
		if h.sessionService.ConnectionByPlayerID(id) != nil {
			relevant = true
			break
		}
	}
	if !relevant {
		return
	}
	if _, ok := r.remote[event.PlayerID]; !ok {
		if !r.addRemote(ctx, event.PlayerID, event.SceneID, owner) {
			return
		}
		if event.Command == protocol.CmdEntityEnterPush {
			return
		}
	}
	packet := &protocol.Packet{Cmd: event.Command, Body: event.Body}
	switch event.Command {
	case protocol.CmdEntityEnterPush:
		var entered protocol.EntityEnterPush
		if protocol.UnmarshalBody(event.Body, &entered) != nil || entered.SceneID != event.SceneID || entered.Entity.PlayerID != event.PlayerID {
			return
		}
		if h.aoi != nil {
			h.aoi.refresh(event.PlayerID, entered.Entity)
		} else {
			h.deliverScenePacket(event.SceneID, event.PlayerID, packet, nil)
		}
	case protocol.CmdEntityMovePush:
		var moved protocol.EntityMovePush
		if protocol.UnmarshalBody(event.Body, &moved) != nil || moved.SceneID != event.SceneID || moved.EntityID != event.PlayerID {
			return
		}
		if h.aoi != nil {
			h.aoi.move(moved)
		} else {
			h.deliverScenePacket(event.SceneID, event.PlayerID, packet, &moved)
		}
	}
}

// 进入/恢复节点时读一次正式摘要和最新 Redis 精细位置，后续移动事件不重新读取完整档案。
func (r *worldRelay) addRemote(ctx context.Context, id uint64, scene uint32, owner string) bool {
	h := r.handler
	brief, ok := h.loadPlayerEntities(ctx, []uint64{id})[id]
	if !ok {
		return false
	}
	state, err := h.worldService.LoadMovementState(ctx, id)
	if err != nil || state.SceneID != scene {
		return false
	}
	brief = movementStateBrief(brief, state)
	r.remote[id] = remoteScenePlayer{owner: owner, scene: scene}
	h.setPlayerScenePresence(id, scene, owner)
	if h.aoi != nil {
		h.aoi.join(id, scene, brief, state, false, nil)
	} else {
		packet, _ := protocol.NewJSONPacket(protocol.CmdEntityEnterPush, 0, 0, protocol.EntityEnterPush{SceneID: scene, Entity: brief})
		h.deliverScenePacket(scene, id, packet, nil)
	}
	return true
}

// 只清理远端索引与可见关系，不操作其 Redis 租约、会话、持久化或权威数据。
func (r *worldRelay) removeRemote(id uint64, current remoteScenePlayer) {
	h := r.handler
	delete(r.remote, id)
	// 该 ID 可能已重连到本实例，远端旧节点的迟到离开不得删除新本机关系。
	if h.sessionService.ConnectionByPlayerID(id) != nil {
		return
	}
	h.presenceMu.Lock()
	if h.playerOwners[id] != current.owner || h.playerScenes[id] != current.scene {
		h.presenceMu.Unlock()
		return
	}
	delete(h.playerScenes, id)
	delete(h.playerOwners, id)
	h.removeScenePlayerLocked(current.scene, id)
	h.presenceMu.Unlock()
	if h.aoi != nil {
		h.aoi.remove(id)
	} else {
		packet, _ := protocol.NewJSONPacket(protocol.CmdEntityLeavePush, 0, 0, protocol.EntityLeavePush{SceneID: current.scene, EntityID: id})
		h.deliverScenePacket(current.scene, id, packet, nil)
	}
}

// 启动后进入场景及周期租约核对补齐远端节点，实例崩溃没有离开事件时也能清理幽灵实体。
func (h *WorldHandler) syncRelayScene(ctx context.Context, scene uint32) {
	if h.relay == nil {
		return
	}
	ids, err := h.worldService.ListSceneOnlinePlayerIDs(ctx, scene)
	if err != nil {
		h.relay.reset()
		return
	}
	r := h.relay
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.invalid.Swap(false) {
		for id, current := range r.remote {
			r.removeRemote(id, current)
		}
	}
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		seen[id] = struct{}{}
		if h.sessionService.ConnectionByPlayerID(id) != nil {
			continue
		}
		owner, err := h.worldService.SceneOnlineOwner(ctx, id)
		if err != nil {
			continue
		}
		if current, ok := r.remote[id]; ok {
			if current.owner == owner && current.scene == scene {
				continue
			}
			r.removeRemote(id, current)
		}
		r.addRemote(ctx, id, scene, owner)
	}
	for id, current := range r.remote {
		if current.scene == scene {
			if _, ok := seen[id]; !ok {
				r.removeRemote(id, current)
			}
		}
	}
}

// 初始/恢复快照使用 Redis 最新位置，避免等待下一次移动才纠正尚未周期落库的远端坐标。
func movementStateBrief(brief protocol.EntityBrief, state *world.MovementState) protocol.EntityBrief {
	point := protocol.Vec2i{X: state.PrecisePos.X, Y: state.PrecisePos.Y}
	facing := protocol.Vec2i{X: state.Facing.X, Y: state.Facing.Y}
	moving := state.Moving
	brief.Pos = protocol.Vec2i{X: state.PersistedPos.X, Y: state.PersistedPos.Y}
	brief.PrecisePos = &point
	brief.Facing = &facing
	brief.Moving = &moving
	brief.Speed = state.Speed
	return brief
}
