package wstransport

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/protocol"
)

// 固定配置仅供测试，正式配置由迁移后的数据库提供。
type aoiConfigRepoStub struct {
	world.Repository
	value world.AOIConfig
}

func (r *aoiConfigRepoStub) GetAOIConfig(context.Context) (world.AOIConfig, error) {
	return r.value, nil
}
func (r *aoiConfigRepoStub) UpdateAOIConfig(_ context.Context, input world.UpdateAOIConfigInput) (world.AOIConfig, error) {
	r.value.Enabled = input.Enabled
	r.value.CellSizeMilli = input.CellSizeMilli
	r.value.Radius = input.Radius
	return r.value, nil
}

// 多实体轻量源只提供正式摘要能力，不允许 AOI 回退成完整人物/战斗查询。
type aoiSummaryRepo struct {
	player.Repository
	values   map[uint64]player.WorldSummary
	requests [][]uint64
}

func (r *aoiSummaryRepo) ListWorldSummariesByPlayerIDs(_ context.Context, ids []uint64) ([]player.WorldSummary, error) {
	r.requests = append(r.requests, append([]uint64(nil), ids...))
	var result []player.WorldSummary
	for _, id := range ids {
		if value, ok := r.values[id]; ok {
			result = append(result, value)
		}
	}
	return result, nil
}

// 外观刷新沿原完整档案入口读取，测试只投影已有源摘要，不修改正式仓储策略。
func (r *aoiSummaryRepo) FindByPlayerID(_ context.Context, id uint64) (*player.Profile, error) {
	value, ok := r.values[id]
	if !ok {
		return nil, nil
	}
	return &player.Profile{PlayerID: id, SceneID: value.SceneID, PosX: value.PosX, PosY: value.PosY, Name: value.Name, SkinID: value.SkinID}, nil
}

// 构造器通过真实领域配置快照启用 AOI，连接仍使用现有可观察发送桩。
func aoiHandlerFixture(t *testing.T) (*WorldHandler, map[uint64]*fakeConn, *aoiSummaryRepo) {
	t.Helper()
	repo := &aoiConfigRepoStub{value: world.AOIConfig{Enabled: true, CellSizeMilli: 1000, Radius: 1}}
	worldService := world.NewService(repo)
	if err := worldService.RefreshAOIConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	summaries := &aoiSummaryRepo{values: make(map[uint64]player.WorldSummary)}
	sessions := session.NewService(log.New(io.Discard, "", 0), time.Second, time.Minute)
	handler := NewWorldHandler(sessions, player.NewService(summaries, nil, nil, nil), nil, nil, nil, worldService, nil, nil)
	peers := make(map[uint64]*fakeConn)
	for id := uint64(1); id <= 4; id++ {
		peers[id] = &fakeConn{id: fmt.Sprintf("aoi_%d", id)}
		if _, err := sessions.Bind(id, peers[id]); err != nil {
			t.Fatal(err)
		}
		scene := uint32(7)
		x := int32(0)
		if id == 2 {
			x = 1
		}
		if id == 3 {
			x = 4
		}
		if id == 4 {
			scene = 8
		}
		summaries.values[id] = player.WorldSummary{PlayerID: id, SceneID: scene, PosX: x, Name: fmt.Sprintf("player%d", id), SkinID: "skin"}
		handler.enterPlayerScene(context.Background(), id, scene)
	}
	for _, peer := range peers {
		peer.packets = nil
	}
	return handler, peers, summaries
}

// 跨格顺序必须是双向离开、双向进入，然后移动；同格、停步、旧序号和断开不破坏可见图。
func TestWorldAOITransitions(t *testing.T) {
	handler, peers, _ := aoiHandlerFixture(t)
	move := func(seq uint32, x int32, moving bool) {
		handler.broadcastEntityMove(7, 1, 1, seq, world.Vec2i{}, world.Vec2i{X: x / 1000}, protocol.Vec2i{X: x}, protocol.Vec2i{X: 1}, moving, 1000, int64(seq)*100, int64(seq)*100)
	}
	move(1, 3000, true)
	if len(peers[2].packets) != 1 || peers[2].packets[0].Cmd != protocol.CmdEntityLeavePush {
		t.Fatal("old neighbor did not leave")
	}
	if len(peers[3].packets) != 2 || peers[3].packets[0].Cmd != protocol.CmdEntityEnterPush || peers[3].packets[1].Cmd != protocol.CmdEntityMovePush {
		t.Fatal("new neighbor received movement before entry")
	}
	if len(peers[1].packets) != 2 || peers[1].packets[0].Cmd != protocol.CmdEntityLeavePush || peers[1].packets[1].Cmd != protocol.CmdEntityEnterPush || len(peers[4].packets) != 0 {
		t.Fatal("self view or scene boundary changed")
	}
	var entry protocol.EntityEnterPush
	if err := protocol.UnmarshalBody(peers[3].packets[0].Body, &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Entity.PrecisePos == nil || entry.Entity.PrecisePos.X != 3000 || entry.Entity.Moving == nil || !*entry.Entity.Moving || entry.Entity.ClientTick != 100 {
		t.Fatal("new visibility used stale DB coordinates or motion")
	}
	for _, peer := range peers {
		peer.packets = nil
	}
	move(2, 3500, true)
	move(3, 3500, false)
	move(2, 3200, true)
	if len(peers[3].packets) != 2 || len(peers[1].packets) != 0 || len(peers[2].packets) != 0 {
		t.Fatal("same-cell or stale sequence handling changed")
	}
	var stopped protocol.EntityMovePush
	if err := protocol.UnmarshalBody(peers[3].packets[1].Body, &stopped); err != nil || stopped.Moving {
		t.Fatal("stop lost")
	}
	handler.HandleSessionDisconnect(1)
	if peers[3].packets[2].Cmd != protocol.CmdEntityLeavePush {
		t.Fatal("disconnect missed visible peer")
	}
	for id := uint64(2); id <= 4; id++ {
		handler.HandleSessionDisconnect(id)
	}
	if len(handler.aoi.players) != 0 || len(handler.aoi.cells) != 0 {
		t.Fatal("AOI retained disconnected players or empty grids")
	}
}

// 负坐标分格与快照一致；快照只读取邻域玩家，静态 NPC 不受玩家 AOI 裁切。
func TestWorldAOISnapshotAndNegativeCells(t *testing.T) {
	handler, _, repo := aoiHandlerFixture(t)
	for value, want := range map[int32]int64{-1001: -2, -1000: -1, -1: -1, 0: 0, 999: 0, 1000: 1} {
		if got := handler.aoi.cellFor(7, protocol.Vec2i{X: value}).x; got != want {
			t.Fatalf("floor(%d)=%d want %d", value, got, want)
		}
	}
	repo.requests = nil
	entities := handler.aoiWorldEntities(context.Background(), 1, 7, world.Vec2i{}, []protocol.EntityBrief{{EntityID: 99, EntityType: 2}, {EntityID: 3, PlayerID: 3, EntityType: 1}})
	if len(repo.requests) != 1 || len(repo.requests[0]) != 1 || repo.requests[0][0] != 2 {
		t.Fatalf("snapshot read outside neighborhood: %v", repo.requests)
	}
	if len(entities) != 2 || entities[0].EntityID != 99 || entities[1].EntityID != 2 {
		t.Fatalf("snapshot included distant player or lost NPC: %+v", entities)
	}
}

// 当前可见图在并发移动、刷新、离开后保持对称，无历史格泄漏；race 覆盖所有可写状态。
func TestWorldAOIConcurrentLifecycle(t *testing.T) {
	handler, _, _ := aoiHandlerFixture(t)
	var group sync.WaitGroup
	for id := uint64(1); id <= 3; id++ {
		group.Add(1)
		go func(id uint64) {
			defer group.Done()
			for seq := uint32(1); seq <= 50; seq++ {
				handler.aoi.move(protocol.EntityMovePush{SceneID: 7, SceneVersion: 1, EntityID: id, MoveSeq: seq, PrecisePos: protocol.Vec2i{X: int32(seq%6) * 1000}, ToPos: protocol.Vec2i{X: int32(seq % 6)}, Moving: true, ClientTick: int64(seq)})
			}
		}(id)
	}
	group.Wait()
	handler.aoi.mu.Lock()
	for id, peer := range handler.aoi.players {
		for neighborID, neighbor := range peer.visible {
			if neighbor.visible[id] != peer || neighborID == id || neighbor.cell.scene != peer.cell.scene {
				t.Fatal("asymmetric AOI visibility")
			}
		}
	}
	handler.aoi.mu.Unlock()
	for id := uint64(1); id <= 4; id++ {
		handler.HandleSessionDisconnect(id)
	}
	if len(handler.aoi.cells) != 0 || len(handler.aoi.players) != 0 {
		t.Fatal("concurrent lifecycle leaked AOI grids")
	}
}

// 同地图传送使用新代次重建邻域，跨图后基础快照之外的玩家仍双向补齐，外观刷新只到可见者。
func TestWorldAOITeleportTransferAndAppearance(t *testing.T) {
	handler, peers, repo := aoiHandlerFixture(t)
	handler.broadcastEntityMove(7, 2, 1, 1, world.Vec2i{}, world.Vec2i{X: 4}, protocol.Vec2i{X: 4000}, protocol.Vec2i{X: 1}, false, 1000, 100, 0)
	for _, peer := range peers {
		peer.packets = nil
	}
	source := repo.values[1]
	source.Name = "新外观"
	source.SkinID = "new_skin"
	repo.values[1] = source
	handler.BroadcastPlayerEntityRefresh(context.Background(), 1)
	if len(peers[3].packets) != 1 || len(peers[2].packets) != 0 || len(peers[1].packets) != 0 {
		t.Fatal("appearance refresh escaped visibility")
	}
	var refreshed protocol.EntityEnterPush
	if err := protocol.UnmarshalBody(peers[3].packets[0].Body, &refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed.Entity.SkinID != "new_skin" || refreshed.Entity.PrecisePos == nil || refreshed.Entity.PrecisePos.X != 4000 {
		t.Fatal("appearance refresh restored stale DB position")
	}
	for _, peer := range peers {
		peer.packets = nil
	}
	handler.movePlayerScenePresence(1, 7, 8)
	source.SceneID = 8
	source.PosX = 0
	repo.values[1] = source
	handler.syncPlayerSceneEntities(context.Background(), peers[1], 1, 8)
	if len(peers[3].packets) != 1 || peers[3].packets[0].Cmd != protocol.CmdEntityLeavePush || len(peers[2].packets) != 0 || len(peers[4].packets) != 1 || peers[4].packets[0].Cmd != protocol.CmdEntityEnterPush || len(peers[1].packets) != 1 {
		t.Fatal("transfer missed old/new visibility lifecycle")
	}
}

// 已在世界快照中出现的邻居不重复补发，快照后才出现的新邻居仍需创建。
func TestWorldAOIJoinOnlyHydratesMissingSnapshotPeers(t *testing.T) {
	handler, peers, repo := aoiHandlerFixture(t)
	handler.aoi.remove(1)
	for _, peer := range peers {
		peer.packets = nil
	}
	source := repo.values[1]
	brief := worldSummaryEntity(source)
	handler.aoi.join(1, 7, brief, nil, true, map[uint64]struct{}{2: {}})
	if len(peers[1].packets) != 0 || len(peers[2].packets) != 1 {
		t.Fatal("known world snapshot peers duplicated")
	}
	handler.aoi.remove(1)
	for _, peer := range peers {
		peer.packets = nil
	}
	handler.aoi.join(1, 7, brief, nil, true, nil)
	if len(peers[1].packets) != 1 {
		t.Fatal("missing peer was not hydrated after snapshot race")
	}
}

// 同会话重连也会更换 ConnID；先离开再进入以重置观察端版本，迟到旧断线不能删除新可见图。
func TestWorldAOIReconnectResetsEntityGeneration(t *testing.T) {
	handler, peers, _ := aoiHandlerFixture(t)
	handler.aoi.players[1].last.SceneVersion = 10
	handler.aoi.players[1].last.MoveSeq = 100
	previous, err := handler.sessionService.GetByPlayerID(1)
	if err != nil {
		t.Fatal(err)
	}
	current := &fakeConn{id: "aoi_reconnected"}
	if _, err := handler.sessionService.Reconnect(previous.ReconnectToken, current); err != nil {
		t.Fatal(err)
	}
	handler.enterPlayerScene(context.Background(), 1, 7, []protocol.EntityBrief{{EntityID: 2, PlayerID: 2, EntityType: 1}})
	handler.broadcastEntityMove(7, 1, 1, 1, world.Vec2i{}, world.Vec2i{}, protocol.Vec2i{}, protocol.Vec2i{X: 1}, true, 1000, 100, 100)
	if len(peers[2].packets) != 3 || peers[2].packets[0].Cmd != protocol.CmdEntityLeavePush || peers[2].packets[1].Cmd != protocol.CmdEntityEnterPush || peers[2].packets[2].Cmd != protocol.CmdEntityMovePush {
		t.Fatal("reconnect did not reset old observer scene/sequence generation")
	}
	handler.HandleOfflineSession(1)
	if handler.aoi.players[1] == nil || handler.playerOwners[1] != current.ID() {
		t.Fatal("late old disconnect deleted reconnected visibility")
	}
}
