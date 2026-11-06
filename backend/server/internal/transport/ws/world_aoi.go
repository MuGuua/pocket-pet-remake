package wstransport

import (
	"context"
	"sync"

	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/platform/telemetry"
	"pocket-pet-remake/server/internal/protocol"
)

// 网格只索引当前进程已进入世界的连接；正式位置仍经原服务端移动/持久化链路确认。
type aoiCell struct {
	scene uint32
	x, y  int64
}
type aoiPlayer struct {
	id      uint64
	cell    aoiCell
	brief   protocol.EntityBrief
	last    protocol.EntityMovePush
	visible map[uint64]*aoiPlayer
}

// 可见图变化与进入/离开/移动入队由同一个短锁串行化，避免跨格并发导致先移动后创建。
// 锁内不读 SQL/Redis，不等待生产连接网络写入；配置在进程启动时固定。
type worldAOI struct {
	mu      sync.Mutex
	config  world.AOIConfig
	handler *WorldHandler
	players map[uint64]*aoiPlayer
	cells   map[aoiCell]map[uint64]*aoiPlayer
}

type aoiPacket struct {
	packet   *protocol.Packet
	encoded  []byte
	movement *protocol.EntityMovePush
}

// 使用数据库已校验配置构造临时连接索引，不引入新的业务数据来源。
func newWorldAOI(config world.AOIConfig, h *WorldHandler) *worldAOI {
	return &worldAOI{config: config, handler: h, players: make(map[uint64]*aoiPlayer), cells: make(map[aoiCell]map[uint64]*aoiPlayer)}
}

// 负坐标采用向下取整，不能用向零截断把地图左/上边界错误合并到零网格。
func (a *worldAOI) cellFor(scene uint32, pos protocol.Vec2i) aoiCell {
	size := int64(a.config.CellSizeMilli)
	floor := func(value int32) int64 {
		n := int64(value)
		if n < 0 {
			return (n - size + 1) / size
		}
		return n / size
	}
	return aoiCell{scene: scene, x: floor(pos.X), y: floor(pos.Y)}
}

// 只扫描固定邻域的桶，成本取决于邻域实体数，不遍历整个场景。
func (a *worldAOI) neighbors(cell aoiCell, self uint64) map[uint64]*aoiPlayer {
	result := make(map[uint64]*aoiPlayer)
	radius := int64(a.config.Radius)
	for x := cell.x - radius; x <= cell.x+radius; x++ {
		for y := cell.y - radius; y <= cell.y+radius; y++ {
			for id, player := range a.cells[aoiCell{scene: cell.scene, x: x, y: y}] {
				if id != self {
					result[id] = player
				}
			}
		}
	}
	return result
}

// 构造一份不可变编码供一组接收者共享；只有移动包携带可合并元信息。
func buildAOIPacket(cmd uint16, payload any) aoiPacket {
	packet, err := protocol.NewJSONPacket(cmd, 0, errcode.WSCodeSuccess, payload)
	if err != nil {
		return aoiPacket{}
	}
	encoded, err := protocol.EncodePacket(packet)
	if err != nil {
		return aoiPacket{}
	}
	result := aoiPacket{packet: packet, encoded: encoded}
	if state, ok := payload.(protocol.EntityMovePush); ok {
		result.movement = &state
	}
	return result
}

// 生产连接仅非阻塞入队；旧替代连接保留同步接口，正式队列满仍按原背压策略断开恢复。
func (a *worldAOI) deliver(id uint64, packet aoiPacket) {
	if packet.packet == nil || a.handler.sessionService == nil {
		return
	}
	conn := a.handler.sessionService.ConnectionByPlayerID(id)
	if conn == nil {
		return
	}
	if packet.movement != nil {
		if conn, ok := conn.(interface {
			TryQueueMovementPacket([]byte, protocol.EntityMovePush) bool
		}); ok {
			conn.TryQueueMovementPacket(packet.encoded, *packet.movement)
			return
		}
	}
	if conn, ok := conn.(interface{ TryQueueEncodedPacket([]byte) bool }); ok {
		conn.TryQueueEncodedPacket(packet.encoded)
		return
	}
	_ = conn.SendPacket(packet.packet)
}

// 用已确认的最近移动覆盖展示坐标和朝向；字段指针引用独立副本，不暴露可写共享状态。
func (p *aoiPlayer) currentBrief(base protocol.EntityBrief) protocol.EntityBrief {
	position, facing, moving := p.last.PrecisePos, p.last.Facing, p.last.Moving
	base.Pos = p.last.ToPos
	base.PrecisePos = &position
	base.Facing = &facing
	base.Moving = &moving
	base.ClientTick = p.last.ClientTick
	base.Speed = p.last.Speed
	return base
}

// 进入时双向补齐实体，重复登记不重复发送；调用方已经在锁外完成必要的数据读取。
func (a *worldAOI) join(id uint64, scene uint32, brief protocol.EntityBrief, state *world.MovementState, hydrateSelf bool, knownPeers map[uint64]struct{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.players[id]; exists {
		return
	}
	last := protocol.EntityMovePush{SceneID: scene, EntityID: id, ToPos: brief.Pos, PrecisePos: protocol.Vec2i{X: brief.Pos.X * 1000, Y: brief.Pos.Y * 1000}, Facing: protocol.Vec2i{Y: 1}}
	if state != nil && state.SceneID == scene {
		last.SceneVersion = state.SceneVersion
		last.MoveSeq = state.LastMoveSeq
		last.ToPos = protocol.Vec2i{X: state.PersistedPos.X, Y: state.PersistedPos.Y}
		last.PrecisePos = protocol.Vec2i{X: state.PrecisePos.X, Y: state.PrecisePos.Y}
		last.Facing = protocol.Vec2i{X: state.Facing.X, Y: state.Facing.Y}
		last.Moving = state.Moving
		last.Speed = state.Speed
	}
	player := &aoiPlayer{id: id, cell: a.cellFor(scene, last.PrecisePos), brief: brief, last: last, visible: make(map[uint64]*aoiPlayer)}
	a.players[id] = player
	if a.cells[player.cell] == nil {
		a.cells[player.cell] = make(map[uint64]*aoiPlayer)
	}
	a.cells[player.cell][id] = player
	enter := buildAOIPacket(protocol.CmdEntityEnterPush, protocol.EntityEnterPush{SceneID: scene, Entity: player.currentBrief(player.brief)})
	for peerID, peer := range a.neighbors(player.cell, id) {
		player.visible[peerID] = peer
		peer.visible[id] = player
		a.deliver(peerID, enter)
		if _, known := knownPeers[peerID]; hydrateSelf && !known {
			a.deliver(id, buildAOIPacket(protocol.CmdEntityEnterPush, protocol.EntityEnterPush{SceneID: scene, Entity: peer.currentBrief(peer.brief)}))
		}
	}
}

// 删除空桶，避免长期切图后留下没有连接的历史网格。
func (a *worldAOI) removeCell(player *aoiPlayer) {
	delete(a.cells[player.cell], player.id)
	if len(a.cells[player.cell]) == 0 {
		delete(a.cells, player.cell)
	}
}

// 离开只通知实际可见者，并对称删除关系；已离开的连接不再收到新的邻域通知。
func (a *worldAOI) remove(id uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	player := a.players[id]
	if player == nil {
		return
	}
	leave := buildAOIPacket(protocol.CmdEntityLeavePush, protocol.EntityLeavePush{SceneID: player.cell.scene, EntityID: id})
	for peerID, peer := range player.visible {
		delete(peer.visible, id)
		a.deliver(peerID, leave)
	}
	a.removeCell(player)
	delete(a.players, id)
}

// 更新坐标后在跨格时计算邻域差集；先双向离开，再双向进入，最后广播当前移动。
func (a *worldAOI) move(state protocol.EntityMovePush) {
	a.mu.Lock()
	defer a.mu.Unlock()
	player := a.players[state.EntityID]
	if player == nil || player.cell.scene != state.SceneID {
		return
	}
	if player.last.SceneVersion != 0 && state.SceneVersion != 0 && (state.SceneVersion < player.last.SceneVersion || (state.SceneVersion == player.last.SceneVersion && state.MoveSeq != 0 && (state.MoveSeq < player.last.MoveSeq || (state.MoveSeq == player.last.MoveSeq && !(player.last.ClientTick == 0 && state.ClientTick > 0))))) {
		return
	}
	nextCell := a.cellFor(state.SceneID, state.PrecisePos)
	player.last = state
	if nextCell != player.cell {
		a.removeCell(player)
		player.cell = nextCell
		if a.cells[nextCell] == nil {
			a.cells[nextCell] = make(map[uint64]*aoiPlayer)
		}
		a.cells[nextCell][player.id] = player
		neighbors := a.neighbors(nextCell, player.id)
		leave := buildAOIPacket(protocol.CmdEntityLeavePush, protocol.EntityLeavePush{SceneID: state.SceneID, EntityID: player.id})
		for peerID, peer := range player.visible {
			if _, exists := neighbors[peerID]; !exists {
				delete(peer.visible, player.id)
				delete(player.visible, peerID)
				a.deliver(peerID, leave)
				a.deliver(player.id, buildAOIPacket(protocol.CmdEntityLeavePush, protocol.EntityLeavePush{SceneID: state.SceneID, EntityID: peerID}))
			}
		}
		enter := buildAOIPacket(protocol.CmdEntityEnterPush, protocol.EntityEnterPush{SceneID: state.SceneID, Entity: player.currentBrief(player.brief)})
		for peerID, peer := range neighbors {
			if _, exists := player.visible[peerID]; !exists {
				player.visible[peerID] = peer
				peer.visible[player.id] = player
				a.deliver(peerID, enter)
				a.deliver(player.id, buildAOIPacket(protocol.CmdEntityEnterPush, protocol.EntityEnterPush{SceneID: state.SceneID, Entity: peer.currentBrief(peer.brief)}))
			}
		}
	}
	packet := buildAOIPacket(protocol.CmdEntityMovePush, state)
	telemetry.Default.Record("world.aoi_recipients", "players", int64(len(player.visible)), false)
	for id := range player.visible {
		a.deliver(id, packet)
	}
}

// 外观/编队更新只推送给当前可见者，同时更新后续跨格进入所用的轻量摘要。
func (a *worldAOI) refresh(id uint64, brief protocol.EntityBrief) {
	a.mu.Lock()
	defer a.mu.Unlock()
	player := a.players[id]
	if player == nil {
		return
	}
	player.brief = brief
	packet := buildAOIPacket(protocol.CmdEntityEnterPush, protocol.EntityEnterPush{SceneID: player.cell.scene, Entity: player.currentBrief(brief)})
	for peerID := range player.visible {
		a.deliver(peerID, packet)
	}
}

// 首次进入或重连先完成轻量 DB/Redis 读取，再在短锁内登记，禁止在移动热路径反复读摘要。
func (h *WorldHandler) ensureAOIEntry(ctx context.Context, id uint64, scene uint32, known []protocol.EntityBrief) {
	h.aoi.mu.Lock()
	existing := h.aoi.players[id]
	h.aoi.mu.Unlock()
	if existing != nil {
		return
	}
	brief, exists := h.loadPlayerEntities(ctx, []uint64{id})[id]
	if !exists {
		return
	}
	var state *world.MovementState
	if h.worldService != nil && h.worldService.MovementStateEnabled() {
		loaded, err := h.worldService.LoadMovementState(ctx, id)
		if err != nil {
			return
		}
		state = loaded
	}
	h.presenceMu.RLock()
	defer h.presenceMu.RUnlock()
	if h.playerScenes[id] != scene {
		return
	}
	knownPeers := make(map[uint64]struct{}, len(known))
	for _, entity := range known {
		knownPeers[entity.EntityID] = struct{}{}
	}
	h.aoi.join(id, scene, brief, state, true, knownPeers)
	// 当前仍持有索引读锁，直接使用已经确认的 owner，避免再次获取读锁与待写锁形成互等。
	if h.relay != nil {
		owner := h.playerOwners[id]
		packet, _ := protocol.NewJSONPacket(protocol.CmdEntityEnterPush, 0, 0, protocol.EntityEnterPush{SceneID: scene, Entity: brief})
		event := world.SceneEvent{Origin: h.relay.origin, Owner: owner, PlayerID: id, SceneID: scene, Command: packet.Cmd, Body: packet.Body}
		select {
		case h.relay.queue <- event:
		default:
			h.relay.reset()
		}
	}
}

// 快照只从邻域批量读取最新源摘要，再覆盖实时坐标；NPC/静态实体沿用原场景数据。
func (h *WorldHandler) aoiWorldEntities(ctx context.Context, id uint64, scene uint32, pos world.Vec2i, entities []protocol.EntityBrief) []protocol.EntityBrief {
	h.syncRelayScene(ctx, scene)
	a := h.aoi
	point := protocol.Vec2i{X: pos.X * 1000, Y: pos.Y * 1000}
	a.mu.Lock()
	peers := a.neighbors(a.cellFor(scene, point), id)
	ids := make([]uint64, 0, len(peers))
	for peerID := range peers {
		ids = append(ids, peerID)
	}
	a.mu.Unlock()
	summaries := h.loadPlayerEntities(ctx, ids)
	result := make([]protocol.EntityBrief, 0, len(entities)+len(ids))
	for _, entity := range entities {
		if entity.EntityType != 1 {
			result = append(result, entity)
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	current := a.neighbors(a.cellFor(scene, point), id)
	for peerID, summary := range summaries {
		if peer := current[peerID]; peer != nil {
			result = append(result, peer.currentBrief(summary))
		}
	}
	return result
}
