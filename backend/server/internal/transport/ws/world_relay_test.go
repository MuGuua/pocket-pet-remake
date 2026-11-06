package wstransport

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/protocol"
)

// 共享权威状态替代器模拟两个独立会话进程；所有修改都先更新权威，再调用正式广播入口。
type relayMovementRepo struct {
	movementStateRepoForHandlerTest
	owners map[uint64]string
}

func (r *relayMovementRepo) MarkSceneOnline(_ context.Context, id uint64, _ uint32, _ string, owner string, _ time.Duration) error {
	r.owners[id] = owner
	return nil
}
func (r *relayMovementRepo) UnmarkSceneOnline(_ context.Context, id uint64, owner string) error {
	if r.owners[id] == owner {
		delete(r.owners, id)
	}
	return nil
}
func (r *relayMovementRepo) SceneOnlineOwner(_ context.Context, id uint64) (string, error) {
	return r.owners[id], nil
}
func (r *relayMovementRepo) ListSceneOnlinePlayerIDs(_ context.Context, scene uint32) ([]uint64, error) {
	var ids []uint64
	for id, owner := range r.owners {
		if owner != "" && r.states[id].SceneID == scene {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

type relayBusStub struct{}

func (*relayBusStub) Publish(context.Context, world.SceneEvent) error { return nil }
func (*relayBusStub) Consume(ctx context.Context, _ func(world.SceneEvent), _ func()) error {
	<-ctx.Done()
	return ctx.Err()
}

// 验证两套正式 handler 在全场景与 AOI 两种模式下的进入、移动、停步、旧 owner 和断开。
func TestWorldRelayTwoInstances(t *testing.T) {
	for _, aoi := range []bool{false, true} {
		t.Run(fmt.Sprint(aoi), func(t *testing.T) {
			ctx := context.Background()
			stateRepo := &relayMovementRepo{movementStateRepoForHandlerTest: movementStateRepoForHandlerTest{states: make(map[uint64]world.MovementState)}, owners: make(map[uint64]string)}
			source := &aoiSummaryRepo{values: map[uint64]player.WorldSummary{1: {PlayerID: 1, Name: "one", SceneID: 7}, 2: {PlayerID: 2, Name: "two", SceneID: 7}}}
			handlers := make([]*WorldHandler, 2)
			peers := make([]*fakeConn, 2)
			for i := 0; i < 2; i++ {
				id := uint64(i + 1)
				cfg := &aoiConfigRepoStub{value: world.AOIConfig{Enabled: aoi, RelayEnabled: true, CellSizeMilli: 1000, Radius: 1}}
				svc := world.NewService(cfg)
				if err := svc.RefreshAOIConfig(ctx); err != nil {
					t.Fatal(err)
				}
				svc.SetMovementStateRepository(stateRepo)
				sessions := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
				peers[i] = &fakeConn{id: fmt.Sprint("node-", i)}
				bound, err := sessions.Bind(id, peers[i])
				if err != nil {
					t.Fatal(err)
				}
				stateRepo.states[id] = world.MovementState{PlayerID: id, SceneID: 7, SceneVersion: 1, PositionVersion: 1, SessionID: bound.ID, Speed: 1000}
				handlers[i] = NewWorldHandler(sessions, player.NewService(source, nil, nil, nil), nil, nil, nil, svc, nil, nil)
				if err := handlers[i].ConfigureWorldRelay(&relayBusStub{}); err != nil {
					t.Fatal(err)
				}
				handlers[i].enterPlayerScene(ctx, id, 7)
			}
			flush := func(from, to *WorldHandler) {
				for {
					select {
					case event := <-from.relay.queue:
						to.relay.receive(ctx, event)
					default:
						return
					}
				}
			}
			flush(handlers[0], handlers[1])
			flush(handlers[1], handlers[0])
			for _, peer := range peers {
				peer.packets = nil
			}
			handlers[0].broadcastEntityMove(7, 1, 1, 1, world.Vec2i{}, world.Vec2i{}, protocol.Vec2i{X: 500}, protocol.Vec2i{X: 1}, true, 1000, 100, 100)
			flush(handlers[0], handlers[1])
			if len(peers[1].packets) != 1 || peers[1].packets[0].Cmd != protocol.CmdEntityMovePush || len(peers[0].packets) != 0 {
				t.Fatal("cross-instance move missing/echoed")
			}
			// 新连接代次替换后，迟到旧离开不得删除新远端节点。
			old := handlers[1].relay.remote[1]
			stateRepo.owners[1] = "replacement"
			enter, _ := protocol.NewJSONPacket(protocol.CmdEntityEnterPush, 0, 0, protocol.EntityEnterPush{SceneID: 7, Entity: protocol.EntityBrief{PlayerID: 1, EntityID: 1, EntityType: 1}})
			handlers[1].relay.receive(ctx, world.SceneEvent{Origin: "new-node", Owner: "replacement", PlayerID: 1, SceneID: 7, Command: enter.Cmd, Body: enter.Body})
			leave, _ := protocol.NewJSONPacket(protocol.CmdEntityLeavePush, 0, 0, protocol.EntityLeavePush{SceneID: 7, EntityID: 1})
			handlers[1].relay.receive(ctx, world.SceneEvent{Origin: "old-node", Owner: old.owner, PlayerID: 1, SceneID: 7, Command: leave.Cmd, Body: leave.Body})
			if handlers[1].relay.remote[1].owner != "replacement" {
				t.Fatal("old leave removed replacement")
			}
			// 无离开事件的进程故障由租约核对清理。
			delete(stateRepo.owners, 1)
			handlers[1].syncRelayScene(ctx, 7)
			if _, ok := handlers[1].relay.remote[1]; ok {
				t.Fatal("expired lease retained ghost")
			}
			// 本机同 ID 重新登录后，旧远端清理不可删除本机索引/可见节点。
			handlers[1].relay.remote[1] = old
			replacement := &fakeConn{id: "local-new"}
			if _, err := handlers[1].sessionService.Bind(1, replacement); err != nil {
				t.Fatal(err)
			}
			handlers[1].setPlayerScenePresence(1, 7, replacement.ID())
			handlers[1].relay.removeRemote(1, old)
			if handlers[1].playerOwners[1] != replacement.ID() {
				t.Fatal("remote cleanup removed local owner")
			}
		})
	}
}

// 发布队列溢出不能无限占内存或静默忽略，已有本机连接必须沿重连路径重建世界。
func TestWorldRelayBoundedQueueRecovery(t *testing.T) {
	sessions := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	peer := &fakeConn{id: "bounded"}
	if _, err := sessions.Bind(1, peer); err != nil {
		t.Fatal(err)
	}
	handler := NewWorldHandler(sessions, nil, nil, nil, nil, nil, nil, nil)
	handler.setPlayerScenePresence(1, 7, peer.ID())
	handler.relay = &worldRelay{origin: "test", handler: handler, queue: make(chan world.SceneEvent, 2), remote: make(map[uint64]remoteScenePlayer)}
	packet, _ := protocol.NewJSONPacket(protocol.CmdEntityMovePush, 0, 0, protocol.EntityMovePush{EntityID: 1, SceneID: 7})
	handler.publishSceneEvent(7, 1, packet)
	handler.publishSceneEvent(7, 1, packet)
	handler.publishSceneEvent(7, 1, packet)
	if !peer.closed || !handler.relay.invalid.Load() || len(handler.relay.queue) != 2 {
		t.Fatal("overflow did not request bounded recovery")
	}
}
