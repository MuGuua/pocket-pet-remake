package wstransport

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/npcdialogue"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/storyprogress"
	"pocket-pet-remake/server/internal/module/unlock"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
)

func TestIsPromptOnlySceneTrigger(t *testing.T) {
	tests := []struct {
		name    string
		trigger *storyprogress.SceneTrigger
		want    bool
	}{
		{
			name: "plain map prompt",
			trigger: &storyprogress.SceneTrigger{
				TriggerCode: "first_enter_market",
				PromptText:  "首次进入市场提示",
			},
			want: true,
		},
		{
			name: "cinematic prompt",
			trigger: &storyprogress.SceneTrigger{
				PromptText:         "剧情结束提示",
				ClientAnimationKey: "初见桃子",
			},
			want: false,
		},
		{
			name: "quest side effect",
			trigger: &storyprogress.SceneTrigger{
				PromptText:          "接取任务提示",
				EffectAcceptQuestID: 1101,
			},
			want: false,
		},
		{
			name: "story flag side effect",
			trigger: &storyprogress.SceneTrigger{
				PromptText:     "剧情解锁提示",
				EffectSetFlags: []string{"npc_unlocked"},
			},
			want: false,
		},
		{
			name:    "missing trigger",
			trigger: nil,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isPromptOnlySceneTrigger(test.trigger); got != test.want {
				t.Fatalf("isPromptOnlySceneTrigger() = %t, want %t", got, test.want)
			}
		})
	}
}

// fakeScenePromptRepository 模拟数据库中的一次性地图提示标记，供世界推送测试验证首次消费行为。
type fakeScenePromptRepository struct {
	trigger  *storyprogress.SceneTrigger
	consumed bool
}

func (r *fakeScenePromptRepository) FindPendingSceneTrigger(_ context.Context, _ uint64, sceneID uint32) (*storyprogress.SceneTrigger, error) {
	if r.consumed || r.trigger == nil || r.trigger.SceneID != sceneID {
		return nil, nil
	}
	return r.trigger, nil
}

func (r *fakeScenePromptRepository) CompleteSceneTrigger(_ context.Context, _ uint64, triggerCode string) (*storyprogress.SceneTrigger, error) {
	if r.trigger == nil || r.trigger.TriggerCode != triggerCode {
		return nil, nil
	}
	r.consumed = true
	return r.trigger, nil
}

func TestPushPendingSceneTriggerConsumesPlainMapPromptBeforeSend(t *testing.T) {
	repo := &fakeScenePromptRepository{
		trigger: &storyprogress.SceneTrigger{
			TriggerCode: "first_enter_market",
			SceneID:     3,
			PromptText:  "首次进入市场提示",
		},
	}
	handler := &WorldHandler{storyService: storyprogress.NewService(repo)}
	conn := &fakeConn{id: "scene-prompt-conn"}

	if err := handler.pushPendingSceneTrigger(context.Background(), conn, 10001, 3); err != nil {
		t.Fatalf("first pushPendingSceneTrigger() error = %v", err)
	}
	if err := handler.pushPendingSceneTrigger(context.Background(), conn, 10001, 3); err != nil {
		t.Fatalf("second pushPendingSceneTrigger() error = %v", err)
	}
	if !repo.consumed {
		t.Fatal("plain map prompt was not persisted as consumed")
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdSceneTriggerPush {
		t.Fatalf("packet.Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdSceneTriggerPush)
	}
}

type fakeConn struct {
	id      string
	packets []*protocol.Packet
	closed  bool
}

func mustFindProtocolBattleActorByUnitClass(t *testing.T, actors []protocol.BattleActorSnapshot, unitClass uint32) protocol.BattleActorSnapshot {
	t.Helper()
	for _, actor := range actors {
		if actor.UnitClass == unitClass {
			return actor
		}
	}
	t.Fatalf("missing protocol battle actor for unit class %d", unitClass)
	return protocol.BattleActorSnapshot{}
}

func (c *fakeConn) ID() string {
	return c.id
}

func (c *fakeConn) SendPacket(packet *protocol.Packet) error {
	c.packets = append(c.packets, packet)
	return nil
}

func (c *fakeConn) Close() error {
	c.closed = true
	return nil
}

func TestRouterHandleEnterWorld(t *testing.T) {
	demoPlayerID, router, _, conn := buildWorldRouterForTest(t)

	packet := protocol.NewPacket(protocol.CmdEnterWorldReq, 11, 0, nil)
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	response := conn.packets[0]
	if response.Cmd != protocol.CmdEnterWorldResp {
		t.Fatalf("response.Cmd = %d, want %d", response.Cmd, protocol.CmdEnterWorldResp)
	}
	if response.Seq != 11 {
		t.Fatalf("response.Seq = %d, want 11", response.Seq)
	}

	var payload protocol.EnterWorldResp
	if err := protocol.UnmarshalBody(response.Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if payload.Self.PlayerID != demoPlayerID {
		t.Fatalf("payload.Self.PlayerID = %d, want %d", payload.Self.PlayerID, demoPlayerID)
	}
	if payload.SceneID != 1 {
		t.Fatalf("payload.SceneID = %d, want 1", payload.SceneID)
	}
	if payload.Player.PlayerID != demoPlayerID {
		t.Fatalf("payload.Player.PlayerID = %d, want %d", payload.Player.PlayerID, demoPlayerID)
	}
	if len(payload.Player.SkillIDs) != 2 {
		t.Fatalf("len(payload.Player.SkillIDs) = %d, want 2", len(payload.Player.SkillIDs))
	}
	if payload.Gold != 2 {
		t.Fatalf("payload.Gold = %d, want 2", payload.Gold)
	}
	if len(payload.Lineup) != 2 {
		t.Fatalf("len(payload.Lineup) = %d, want 2", len(payload.Lineup))
	}
	if len(payload.NearbyEntities) != 3 {
		t.Fatalf("len(payload.NearbyEntities) = %d, want 3", len(payload.NearbyEntities))
	}
	var sawRivalPlayer bool
	for _, entity := range payload.NearbyEntities {
		if entity.PlayerID == teststub.RivalPlayerID {
			sawRivalPlayer = true
			if entity.FollowingPet == nil {
				t.Fatal("rival player entity missing following_pet")
			}
			if entity.FollowingPet.PetUID != 21001 || entity.FollowingPet.SkinID != "嫩叶犬_001" {
				t.Fatalf("rival following pet = %+v, want pet_uid 21001 skin 嫩叶犬_001", entity.FollowingPet)
			}
			break
		}
	}
	if !sawRivalPlayer {
		t.Fatal("payload.NearbyEntities missing rival player_id entry")
	}
}

// TestWorldSummaryEntityIncludesLightweightStatus 验证切图后增量实体包含移动端展示所需字段，无需加载完整玩家或背包档案。
func TestWorldSummaryEntityIncludesLightweightStatus(t *testing.T) {
	entity := worldSummaryEntity(player.WorldSummary{
		PlayerID:  10002,
		Name:      "RivalTrainer",
		Level:     12,
		Exp:       3456,
		SceneID:   3,
		PosX:      7,
		PosY:      8,
		HP:        88,
		HPMax:     120,
		Vigor:     70,
		VigorMax:  100,
		Spirit:    25,
		SpiritMax: 40,
		SkinID:    player.DefaultPlayerSkinID,
	})
	if entity.Name != "RivalTrainer" || entity.Level != 12 || entity.Exp != 3456 {
		t.Fatalf("player identity summary = %+v", entity)
	}
	if entity.HP != 88 || entity.HPMax != 120 || entity.Vigor != 70 || entity.VigorMax != 100 || entity.Spirit != 25 || entity.SpiritMax != 40 {
		t.Fatalf("player status summary = %+v", entity)
	}
	if entity.SkinID != player.DefaultPlayerSkinID || entity.Pos != (protocol.Vec2i{X: 7, Y: 8}) {
		t.Fatalf("player visual summary = %+v", entity)
	}
}

// TestProtocolWorldFollowingPetIncludesLightweightStatus 验证跟随宠物增量实体携带名字、经验、血量、精力和形象。
func TestProtocolWorldFollowingPetIncludesLightweightStatus(t *testing.T) {
	brief := toProtocolWorldFollowingPet(pet.WorldFollowingPet{
		PlayerID:  10002,
		PetUID:    21001,
		PetID:     101,
		Name:      "小火龙",
		SkinID:    "嫩叶犬_001",
		Level:     5,
		Exp:       110,
		HP:        31,
		HPMax:     32,
		Spirit:    12,
		SpiritMax: 20,
	})
	if brief.Name != "小火龙" || brief.Level != 5 || brief.Exp != 110 || brief.SkinID != "嫩叶犬_001" {
		t.Fatalf("pet identity summary = %+v", brief)
	}
	if brief.HP != 31 || brief.HPMax != 32 || brief.Spirit != 12 || brief.SpiritMax != 20 {
		t.Fatalf("pet status summary = %+v", brief)
	}
}

// TestRouterHandlePlayerProfile 验证人物面板通过独立请求获取权威属性，不再依赖完整进入世界响应。
func TestRouterHandlePlayerProfile(t *testing.T) {
	demoPlayerID, router, _, conn := buildWorldRouterForTest(t)

	mustHandleJSONPacket(t, router, conn, protocol.CmdPlayerProfileReq, 1201, protocol.PlayerProfileReq{})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1 player profile response", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdPlayerProfileResp {
		t.Fatalf("response cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdPlayerProfileResp)
	}
	var response protocol.PlayerProfileResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &response); err != nil {
		t.Fatalf("UnmarshalBody(player profile) error = %v", err)
	}
	if response.Player.PlayerID != demoPlayerID || response.Player.Name != "DemoTrainer" {
		t.Fatalf("player profile identity = %+v", response.Player)
	}
	if response.Player.HP == 0 || response.Player.HPMax == 0 || response.Player.SkinID == "" {
		t.Fatalf("player profile status = %+v", response.Player)
	}
}

func TestRouterHandleEnterWorldFallsBackToMarketWhenSceneMissing(t *testing.T) {
	demoPlayerID, router, playerService, conn := buildWorldRouterForTest(t)

	// 先把玩家档案中的非法场景写进去，模拟历史脏数据或旧配置导致的 scene_id 失效。
	if err := playerService.UpdatePosition(context.Background(), demoPlayerID, 999, 0, 0); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	packet := protocol.NewPacket(protocol.CmdEnterWorldReq, 111, 0, nil)
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	var payload protocol.EnterWorldResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if payload.SceneID != world.FallbackSceneID {
		t.Fatalf("payload.SceneID = %d, want %d", payload.SceneID, world.FallbackSceneID)
	}
	if payload.SelfPos != (protocol.Vec2i{X: 12, Y: 10}) {
		t.Fatalf("payload.SelfPos = %#v, want market fallback spawn", payload.SelfPos)
	}

	profile, err := playerService.GetProfile(context.Background(), demoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.SceneID != world.FallbackSceneID || profile.PosX != 12 || profile.PosY != 10 {
		t.Fatalf("profile fallback = (%d,%d,%d), want (%d,12,10)", profile.SceneID, profile.PosX, profile.PosY, world.FallbackSceneID)
	}
}

func TestRouterRejectUnauthenticatedEnterWorld(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, nil, nil, questService, nil, nil, nil, nil)
	petHandler := NewPetHandler(sessionService, nil, nil)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleHandler := NewBattleHandler(sessionService, nil, nil, nil, nil, nil, questService, npcService, nil, battle.NewService(nil), teststub.NewBattleRepository(), nil, nil)
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, NewPlayerHandler(sessionService, nil), nil, battleHandler, nil, NewQuestHandler(questService, sessionService, nil, nil, nil, nil, nil), sessionService)

	conn := &fakeConn{id: "conn-2"}
	packet := protocol.NewPacket(protocol.CmdEnterWorldReq, 12, 0, nil)
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	response := conn.packets[0]
	if response.Cmd != protocol.CmdErrorPush {
		t.Fatalf("response.Cmd = %d, want %d", response.Cmd, protocol.CmdErrorPush)
	}
	if response.Code != errcode.WSCodeUnauthorized {
		t.Fatalf("response.Code = %d, want %d", response.Code, errcode.WSCodeUnauthorized)
	}

	var payload protocol.ErrorPush
	if err := protocol.UnmarshalBody(response.Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if payload.Code != errcode.WSCodeUnauthorized {
		t.Fatalf("payload.Code = %d, want %d", payload.Code, errcode.WSCodeUnauthorized)
	}
	if payload.Msg != "unauthorized" {
		t.Fatalf("payload.Msg = %q, want unauthorized", payload.Msg)
	}
}

func TestRouterHandleMoveIntentLocalOnly(t *testing.T) {
	demoPlayerID, router, playerService, conn := buildWorldRouterForTest(t)

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 13, 0, protocol.MoveIntentReq{
		OpID:    1,
		MoveSeq: 3,
		SceneID: 1,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}

	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	respPacket := conn.packets[0]
	if respPacket.Cmd != protocol.CmdMoveIntentResp {
		t.Fatalf("respPacket.Cmd = %d, want %d", respPacket.Cmd, protocol.CmdMoveIntentResp)
	}

	var resp protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(respPacket.Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("resp.Accepted = false, want true")
	}
	if resp.MoveSeq != 3 {
		t.Fatalf("resp.MoveSeq = %d, want 3", resp.MoveSeq)
	}
	if resp.SceneID != 1 {
		t.Fatalf("resp.SceneID = %d, want 1", resp.SceneID)
	}
	if resp.Reason != "local movement handled by client" {
		t.Fatalf("resp.Reason = %q, want local movement handled by client", resp.Reason)
	}

	profile, err := playerService.GetProfile(context.Background(), demoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.PosX != 8 || profile.PosY != 6 {
		t.Fatalf("profile position = (%d,%d), want (8,6)", profile.PosX, profile.PosY)
	}
}

// TestRouterHandleMoveIntentSynchronizesPosition 验证新客户端上报坐标后，服务端先持久化再返回权威坐标。
func TestRouterHandleMoveIntentSynchronizesPosition(t *testing.T) {
	demoPlayerID, router, playerService, conn := buildWorldRouterForTest(t)
	targetPos := protocol.Vec2i{X: 9, Y: 7}

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 131, 0, protocol.MoveIntentReq{
		OpID:      11,
		MoveSeq:   31,
		SceneID:   1,
		TargetPos: &targetPos,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	var resp protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted || resp.CorrectedPos != targetPos {
		t.Fatalf("response = %+v, want accepted target position %+v", resp, targetPos)
	}
	if resp.Reason != "position synchronized" {
		t.Fatalf("resp.Reason = %q, want position synchronized", resp.Reason)
	}

	profile, err := playerService.GetProfile(context.Background(), demoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.PosX != targetPos.X || profile.PosY != targetPos.Y {
		t.Fatalf("profile position = (%d,%d), want (%d,%d)", profile.PosX, profile.PosY, targetPos.X, targetPos.Y)
	}
}

// TestRouterHandleMapTeleportToAnotherScene 验证快速传送可以跨越普通出口关系，并使用服务端配置的地图中心格。
func TestRouterHandleMapTeleportToAnotherScene(t *testing.T) {
	demoPlayerID, router, playerService, conn := buildWorldRouterForTest(t)

	mustHandleJSONPacket(t, router, conn, protocol.CmdMoveIntentReq, 1321, protocol.MoveIntentReq{
		OpID:          121,
		MoveSeq:       321,
		SceneID:       1,
		TargetSceneID: 6,
		MapTeleport:   true,
	})
	if len(conn.packets) < 2 {
		t.Fatalf("len(conn.packets) = %d, want at least move response and world resync", len(conn.packets))
	}

	var response protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &response); err != nil {
		t.Fatalf("UnmarshalBody(move response) error = %v", err)
	}
	if !response.Accepted || response.SceneID != 6 || response.CorrectedPos != (protocol.Vec2i{X: 5, Y: 5}) {
		t.Fatalf("map teleport response = %+v, want accepted scene 6 center (5,5)", response)
	}

	profile, err := playerService.GetProfile(context.Background(), demoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.SceneID != 6 || profile.PosX != 5 || profile.PosY != 5 {
		t.Fatalf("profile scene/position = %d/(%d,%d), want 6/(5,5)", profile.SceneID, profile.PosX, profile.PosY)
	}
}

// TestRouterHandleMapTeleportWithinCurrentScene 验证再次点击当前地图也会移动到中心，但不会误走普通坐标上报分支。
func TestRouterHandleMapTeleportWithinCurrentScene(t *testing.T) {
	demoPlayerID, router, playerService, conn := buildWorldRouterForTest(t)

	mustHandleJSONPacket(t, router, conn, protocol.CmdMoveIntentReq, 1322, protocol.MoveIntentReq{
		OpID:          122,
		MoveSeq:       322,
		SceneID:       1,
		TargetSceneID: 1,
		MapTeleport:   true,
	})
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want move response and world resync only", len(conn.packets))
	}

	var response protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &response); err != nil {
		t.Fatalf("UnmarshalBody(move response) error = %v", err)
	}
	if !response.Accepted || response.SceneID != 1 || response.CorrectedPos != (protocol.Vec2i{X: 5, Y: 7}) {
		t.Fatalf("same-scene map teleport response = %+v, want accepted scene 1 center (5,7)", response)
	}

	profile, err := playerService.GetProfile(context.Background(), demoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.SceneID != 1 || profile.PosX != 5 || profile.PosY != 7 {
		t.Fatalf("profile scene/position = %d/(%d,%d), want 1/(5,7)", profile.SceneID, profile.PosX, profile.PosY)
	}
}

// TestRouterBroadcastsMovementOnlyToPlayersInSameScene 验证移动推送会到达同场景玩家，且不会回推给移动者自身。
func TestRouterBroadcastsMovementOnlyToPlayersInSameScene(t *testing.T) {
	_, router, _, firstConn := buildWorldRouterForTest(t)
	secondConn := &fakeConn{id: "conn-2"}
	if _, err := router.sessionService.Bind(teststub.RivalPlayerID, secondConn); err != nil {
		t.Fatalf("Bind(rival) error = %v", err)
	}

	mustHandleJSONPacket(t, router, firstConn, protocol.CmdEnterWorldReq, 132, protocol.EnterWorldReq{})
	mustHandleJSONPacket(t, router, secondConn, protocol.CmdEnterWorldReq, 133, protocol.EnterWorldReq{})
	clearPackets(firstConn)
	clearPackets(secondConn)

	targetPos := protocol.Vec2i{X: 10, Y: 7}
	precisePos := protocol.Vec2i{X: 10250, Y: 7000}
	facing := protocol.Vec2i{X: 1}
	moving := true
	mustHandleJSONPacket(t, router, firstConn, protocol.CmdMoveIntentReq, 134, protocol.MoveIntentReq{
		OpID:       12,
		MoveSeq:    32,
		SceneID:    1,
		TargetPos:  &targetPos,
		PrecisePos: &precisePos,
		Facing:     &facing,
		Moving:     &moving,
	})
	if len(firstConn.packets) != 1 || firstConn.packets[0].Cmd != protocol.CmdMoveIntentResp {
		t.Fatalf("moving player packets = %+v, want only move response", firstConn.packets)
	}
	if len(secondConn.packets) != 1 || secondConn.packets[0].Cmd != protocol.CmdEntityMovePush {
		t.Fatalf("nearby player packets = %+v, want one entity move push", secondConn.packets)
	}

	var push protocol.EntityMovePush
	if err := protocol.UnmarshalBody(secondConn.packets[0].Body, &push); err != nil {
		t.Fatalf("UnmarshalBody(entity move) error = %v", err)
	}
	if push.SceneID != 1 || push.EntityID != teststub.DemoPlayerID || push.ToPos != targetPos {
		t.Fatalf("entity move push = %+v, want demo player target %+v in scene 1", push, targetPos)
	}
	if push.PrecisePos != precisePos || push.Facing != facing || !push.Moving {
		t.Fatalf("entity movement presentation = %+v, want precise=%+v facing=%+v moving=true", push, precisePos, facing)
	}
}

// TestNormalizeMovementPresentation 验证服务端会限制高精度表现位置并拒绝斜向朝向。
func TestNormalizeMovementPresentation(t *testing.T) {
	targetPos := world.Vec2i{X: 10, Y: 7}
	outOfRange := protocol.Vec2i{X: 12000, Y: 5000}
	precisePos := normalizePreciseMovementPosition(targetPos, &outOfRange)
	if precisePos.X != 10500 || precisePos.Y != 6500 {
		t.Fatalf("precise position = %+v, want clamped (10500,6500)", precisePos)
	}

	diagonalFacing := protocol.Vec2i{X: 1, Y: 1}
	facing := normalizeMovementFacing(world.Vec2i{X: 9, Y: 7}, targetPos, &diagonalFacing)
	if facing != (protocol.Vec2i{X: 1}) {
		t.Fatalf("facing = %+v, want inferred right direction", facing)
	}
	if !normalizeMovementState(world.Vec2i{X: 9, Y: 7}, targetPos, nil) {
		t.Fatal("legacy movement state = false, want inferred moving=true")
	}
	stopped := false
	if normalizeMovementState(world.Vec2i{X: 9, Y: 7}, targetPos, &stopped) {
		t.Fatal("explicit movement state = true, want stopped=false")
	}
}

// TestRouterBroadcastsFollowingPetAfterLineupChange 验证玩家更换首只出战宠物后同场景客户端立即刷新跟随表现。
func TestRouterBroadcastsFollowingPetAfterLineupChange(t *testing.T) {
	_, router, _, firstConn := buildWorldRouterForTest(t)
	secondConn := &fakeConn{id: "conn-2"}
	if _, err := router.sessionService.Bind(teststub.RivalPlayerID, secondConn); err != nil {
		t.Fatalf("Bind(rival) error = %v", err)
	}

	mustHandleJSONPacket(t, router, firstConn, protocol.CmdEnterWorldReq, 135, protocol.EnterWorldReq{})
	mustHandleJSONPacket(t, router, secondConn, protocol.CmdEnterWorldReq, 136, protocol.EnterWorldReq{})
	clearPackets(firstConn)
	clearPackets(secondConn)

	mustHandleJSONPacket(t, router, firstConn, protocol.CmdPetLineupSetReq, 137, protocol.PetLineupSetReq{
		PetUIDs: []uint64{20002},
	})
	if len(firstConn.packets) != 1 || firstConn.packets[0].Cmd != protocol.CmdPetLineupSetResp {
		t.Fatalf("lineup owner packets = %+v, want one lineup response", firstConn.packets)
	}
	if len(secondConn.packets) != 1 || secondConn.packets[0].Cmd != protocol.CmdEntityEnterPush {
		t.Fatalf("nearby player packets = %+v, want one entity refresh push", secondConn.packets)
	}

	var push protocol.EntityEnterPush
	if err := protocol.UnmarshalBody(secondConn.packets[0].Body, &push); err != nil {
		t.Fatalf("UnmarshalBody(entity refresh) error = %v", err)
	}
	if push.Entity.PlayerID != teststub.DemoPlayerID || push.Entity.FollowingPet == nil {
		t.Fatalf("entity refresh = %+v, want demo player with following pet", push.Entity)
	}
	if push.Entity.FollowingPet.PetUID != 20002 || push.Entity.FollowingPet.SkinID != "潮汐狐_001" {
		t.Fatalf("following pet = %+v, want pet_uid 20002 skin 潮汐狐_001", push.Entity.FollowingPet)
	}

	clearPackets(firstConn)
	clearPackets(secondConn)
	mustHandleJSONPacket(t, router, firstConn, protocol.CmdPetLineupSetReq, 138, protocol.PetLineupSetReq{
		PetUIDs: []uint64{},
	})
	if len(secondConn.packets) != 1 || secondConn.packets[0].Cmd != protocol.CmdEntityEnterPush {
		t.Fatalf("empty lineup nearby packets = %+v, want one entity refresh push", secondConn.packets)
	}
	push = protocol.EntityEnterPush{}
	if err := protocol.UnmarshalBody(secondConn.packets[0].Body, &push); err != nil {
		t.Fatalf("UnmarshalBody(empty lineup entity refresh) error = %v", err)
	}
	if push.Entity.FollowingPet != nil {
		t.Fatalf("empty lineup following pet = %+v, want nil", push.Entity.FollowingPet)
	}
}

func TestRouterHandleMoveIntentSceneTransfer(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 14, 0, protocol.MoveIntentReq{
		OpID:          2,
		MoveSeq:       4,
		SceneID:       1,
		TargetSceneID: 2,
		PortalID:      1001,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}

	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) = %d, want 3", len(conn.packets))
	}

	respPacket := conn.packets[0]
	if respPacket.Cmd != protocol.CmdMoveIntentResp {
		t.Fatalf("respPacket.Cmd = %d, want %d", respPacket.Cmd, protocol.CmdMoveIntentResp)
	}

	var resp protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(respPacket.Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("resp.Accepted = false, want true")
	}
	if resp.SceneID != 2 {
		t.Fatalf("resp.SceneID = %d, want 2", resp.SceneID)
	}

	resyncPacket := conn.packets[1]
	if resyncPacket.Cmd != protocol.CmdWorldResyncPush {
		t.Fatalf("resyncPacket.Cmd = %d, want %d", resyncPacket.Cmd, protocol.CmdWorldResyncPush)
	}

	var resync protocol.WorldResyncPush
	if err := protocol.UnmarshalBody(resyncPacket.Body, &resync); err != nil {
		t.Fatalf("UnmarshalBody(resync) error = %v", err)
	}
	if resync.SceneID != 2 {
		t.Fatalf("resync.SceneID = %d, want 2", resync.SceneID)
	}
	if resync.SelfPos.X != 4 || resync.SelfPos.Y != 1 {
		t.Fatalf("resync.SelfPos = (%d,%d), want (4,1)", resync.SelfPos.X, resync.SelfPos.Y)
	}

	questUpdates := collectQuestUpdatesByID(t, conn.packets[2:])
	if got := questUpdates[1001].State; got != quest.StateReadyToSubmit {
		t.Fatalf("quest 1001 state = %q, want %q", got, quest.StateReadyToSubmit)
	}
	if _, exists := questUpdates[1002]; exists {
		t.Fatalf("quest 1002 should stay locked before claiming quest 1001 reward")
	}

	profile, err := playerService.GetProfile(context.Background(), 10001)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.SceneID != 2 {
		t.Fatalf("profile.SceneID = %d, want 2", profile.SceneID)
	}
	if profile.PosX != 4 || profile.PosY != 1 {
		t.Fatalf("profile position = (%d,%d), want (4,1)", profile.PosX, profile.PosY)
	}
}

func TestRouterHandleUseItemExpandsBagAndPushesContainerUpdate(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdUseItemReq, 301, protocol.UseItemReq{
		ContainerType: bag.ContainerTypeBag,
		SlotIndex:     3,
		Quantity:      1,
	})
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdUseItemResp {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdUseItemResp)
	}
	if conn.packets[1].Cmd != protocol.CmdBagUpdatePush {
		t.Fatalf("conn.packets[1].Cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdBagUpdatePush)
	}

	var useResp protocol.UseItemResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &useResp); err != nil {
		t.Fatalf("UnmarshalBody(useResp) error = %v", err)
	}
	if useResp.ItemID != 3001 {
		t.Fatalf("useResp.ItemID = %d, want 3001", useResp.ItemID)
	}
	if useResp.Result.EffectType != "bag_expand" {
		t.Fatalf("useResp.Result.EffectType = %q, want bag_expand", useResp.Result.EffectType)
	}
	if useResp.Result.NewCapacity != 35 {
		t.Fatalf("useResp.Result.NewCapacity = %d, want 35", useResp.Result.NewCapacity)
	}

	var update protocol.BagUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &update); err != nil {
		t.Fatalf("UnmarshalBody(update) error = %v", err)
	}
	if update.ContainerType != bag.ContainerTypeBag {
		t.Fatalf("update.ContainerType = %q, want %q", update.ContainerType, bag.ContainerTypeBag)
	}
	if update.Capacity != 35 {
		t.Fatalf("update.Capacity = %d, want 35", update.Capacity)
	}
}

func TestRouterHandleUseItemRestoresPetHPAndPushesPetUpdate(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdUseItemReq, 302, protocol.UseItemReq{
		ContainerType: bag.ContainerTypeBag,
		SlotIndex:     1,
		Quantity:      1,
		TargetPetUID:  20002,
	})
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) = %d, want 3", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdUseItemResp {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdUseItemResp)
	}
	if conn.packets[1].Cmd != protocol.CmdBagUpdatePush {
		t.Fatalf("conn.packets[1].Cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdBagUpdatePush)
	}
	if conn.packets[2].Cmd != protocol.CmdPetUpdatePush {
		t.Fatalf("conn.packets[2].Cmd = %d, want %d", conn.packets[2].Cmd, protocol.CmdPetUpdatePush)
	}

	var useResp protocol.UseItemResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &useResp); err != nil {
		t.Fatalf("UnmarshalBody(useResp) error = %v", err)
	}
	if useResp.ItemID != 3003 {
		t.Fatalf("useResp.ItemID = %d, want 3003", useResp.ItemID)
	}
	if useResp.Result.EffectType != "pet_hp_restore" {
		t.Fatalf("useResp.Result.EffectType = %q, want pet_hp_restore", useResp.Result.EffectType)
	}
	if useResp.Result.TargetPetUID != 20002 {
		t.Fatalf("useResp.Result.TargetPetUID = %d, want 20002", useResp.Result.TargetPetUID)
	}
	if useResp.Result.RestoredHP != 2 {
		t.Fatalf("useResp.Result.RestoredHP = %d, want 2", useResp.Result.RestoredHP)
	}
	if useResp.Result.NewPetHP != 30 {
		t.Fatalf("useResp.Result.NewPetHP = %d, want 30", useResp.Result.NewPetHP)
	}

	var bagUpdate protocol.BagUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &bagUpdate); err != nil {
		t.Fatalf("UnmarshalBody(bagUpdate) error = %v", err)
	}
	if bagUpdate.ContainerType != bag.ContainerTypeBag {
		t.Fatalf("bagUpdate.ContainerType = %q, want %q", bagUpdate.ContainerType, bag.ContainerTypeBag)
	}
	if len(bagUpdate.Updates) != int(bagUpdate.Capacity) {
		t.Fatalf("len(bagUpdate.Updates) = %d, want %d", len(bagUpdate.Updates), bagUpdate.Capacity)
	}
	if bagUpdate.Updates[0].SlotIndex != 1 || bagUpdate.Updates[0].Item == nil || bagUpdate.Updates[0].Item.Quantity != 2 {
		t.Fatalf("bagUpdate.Updates[0] = %#v, want slot 1 quantity 2", bagUpdate.Updates[0])
	}

	var petUpdate protocol.PetUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[2].Body, &petUpdate); err != nil {
		t.Fatalf("UnmarshalBody(petUpdate) error = %v", err)
	}
	if petUpdate.Pet.PetUID != 20002 {
		t.Fatalf("petUpdate.Pet.PetUID = %d, want 20002", petUpdate.Pet.PetUID)
	}
	if petUpdate.Pet.HP != 30 || petUpdate.Pet.HPMax != 30 {
		t.Fatalf("petUpdate.Pet HP = %d/%d, want 30/30", petUpdate.Pet.HP, petUpdate.Pet.HPMax)
	}

	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdPetListReq, 303, protocol.PetListReq{})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}
	var petList protocol.PetListResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &petList); err != nil {
		t.Fatalf("UnmarshalBody(petList) error = %v", err)
	}
	var found bool
	for _, currentPet := range petList.Pets {
		if currentPet.PetUID != 20002 {
			continue
		}
		found = true
		if currentPet.HP != 30 {
			t.Fatalf("currentPet.HP = %d, want 30", currentPet.HP)
		}
	}
	if !found {
		t.Fatal("pet list missing updated target pet")
	}
}

func TestRouterHandleUseItemOpensRewardBoxAndPushesBagAndWallet(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdUseItemReq, 304, protocol.UseItemReq{
		ContainerType: bag.ContainerTypeBag,
		SlotIndex:     4,
		Quantity:      1,
	})
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) = %d, want 3", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdUseItemResp {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdUseItemResp)
	}
	if conn.packets[1].Cmd != protocol.CmdBagUpdatePush {
		t.Fatalf("conn.packets[1].Cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdBagUpdatePush)
	}
	if conn.packets[2].Cmd != protocol.CmdWalletUpdatePush {
		t.Fatalf("conn.packets[2].Cmd = %d, want %d", conn.packets[2].Cmd, protocol.CmdWalletUpdatePush)
	}

	var useResp protocol.UseItemResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &useResp); err != nil {
		t.Fatalf("UnmarshalBody(useResp) error = %v", err)
	}
	if useResp.ItemID != 3004 {
		t.Fatalf("useResp.ItemID = %d, want 3004", useResp.ItemID)
	}
	if useResp.Result.EffectType != "reward_box" {
		t.Fatalf("useResp.Result.EffectType = %q, want reward_box", useResp.Result.EffectType)
	}
	if len(useResp.Result.Rewards) != 2 {
		t.Fatalf("len(useResp.Result.Rewards) = %d, want 2", len(useResp.Result.Rewards))
	}
	if useResp.Result.Rewards[0].Type != "gold" || useResp.Result.Rewards[0].Value != 2 {
		t.Fatalf("useResp.Result.Rewards[0] = %#v, want gold 2", useResp.Result.Rewards[0])
	}
	if useResp.Result.Rewards[1].Type != "item" || useResp.Result.Rewards[1].ItemID != 2001 || useResp.Result.Rewards[1].Count != 1 {
		t.Fatalf("useResp.Result.Rewards[1] = %#v, want item 2001 x1", useResp.Result.Rewards[1])
	}
	if useResp.Result.Rewards[1].ItemName != "新手精灵球" {
		t.Fatalf("useResp.Result.Rewards[1].ItemName = %q, want 新手精灵球", useResp.Result.Rewards[1].ItemName)
	}

	var bagUpdate protocol.BagUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &bagUpdate); err != nil {
		t.Fatalf("UnmarshalBody(bagUpdate) error = %v", err)
	}
	if bagUpdate.ContainerType != bag.ContainerTypeBag {
		t.Fatalf("bagUpdate.ContainerType = %q, want %q", bagUpdate.ContainerType, bag.ContainerTypeBag)
	}
	if bagUpdate.Updates[0].SlotIndex != 1 || bagUpdate.Updates[0].Item == nil || bagUpdate.Updates[0].Item.Quantity != 3 {
		t.Fatalf("bagUpdate.Updates[0] = %#v, want slot 1 quantity 3 after reward grant", bagUpdate.Updates[0])
	}
	if bagUpdate.Updates[3].SlotIndex != 4 || bagUpdate.Updates[3].Item != nil {
		t.Fatalf("bagUpdate.Updates[3] = %#v, want consumed reward box slot to be empty", bagUpdate.Updates[3])
	}

	var walletUpdate protocol.WalletUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[2].Body, &walletUpdate); err != nil {
		t.Fatalf("UnmarshalBody(walletUpdate) error = %v", err)
	}
	if walletUpdate.ReasonType != "item_use_reward" || walletUpdate.ReasonRefID != 3004 {
		t.Fatalf("walletUpdate = %#v, want item_use_reward/3004", walletUpdate)
	}
	if walletUpdate.Wallet.TotalCopper != 2345680 {
		t.Fatalf("walletUpdate.Wallet.TotalCopper = %d, want 2345680 after +2 copper reward", walletUpdate.Wallet.TotalCopper)
	}
}

func TestRouterHandleBuyItemConsumesWalletAndPushesBag(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)
	clearPackets(conn)
	mustMovePlayerToScene(t, playerService, 3, 12, 10)

	mustHandleJSONPacket(t, router, conn, protocol.CmdBuyItemReq, 305, protocol.BuyItemReq{
		ShopID:   93002,
		GoodsID:  1001,
		ItemID:   1001,
		Quantity: 2,
	})
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) = %d, want 3", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdBuyItemResp {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdBuyItemResp)
	}
	if conn.packets[1].Cmd != protocol.CmdBagUpdatePush {
		t.Fatalf("conn.packets[1].Cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdBagUpdatePush)
	}
	if conn.packets[2].Cmd != protocol.CmdWalletUpdatePush {
		t.Fatalf("conn.packets[2].Cmd = %d, want %d", conn.packets[2].Cmd, protocol.CmdWalletUpdatePush)
	}

	var buyResp protocol.BuyItemResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &buyResp); err != nil {
		t.Fatalf("UnmarshalBody(buyResp) error = %v", err)
	}
	if buyResp.ShopID != 93002 || buyResp.GoodsID != 1001 || buyResp.ItemID != 1001 {
		t.Fatalf("buyResp identity = %#v, want shop=93002 goods=1001 item=1001", buyResp)
	}
	if buyResp.Quantity != 2 {
		t.Fatalf("buyResp.Quantity = %d, want 2", buyResp.Quantity)
	}
	if buyResp.Cost.CurrencyType != "base_coin" || buyResp.Cost.TotalCopper != 1000 {
		t.Fatalf("buyResp.Cost = %#v, want base_coin/1000", buyResp.Cost)
	}
	if buyResp.Wallet.TotalCopper != 2344678 {
		t.Fatalf("buyResp.Wallet.TotalCopper = %d, want 2344678", buyResp.Wallet.TotalCopper)
	}

	var bagUpdate protocol.BagUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &bagUpdate); err != nil {
		t.Fatalf("UnmarshalBody(bagUpdate) error = %v", err)
	}
	if bagUpdate.ContainerType != bag.ContainerTypeBag {
		t.Fatalf("bagUpdate.ContainerType = %q, want %q", bagUpdate.ContainerType, bag.ContainerTypeBag)
	}
	if bagUpdate.Updates[1].SlotIndex != 2 || bagUpdate.Updates[1].Item == nil {
		t.Fatalf("bagUpdate.Updates[1] = %#v, want purchased item on slot 2", bagUpdate.Updates[1])
	}
	if bagUpdate.Updates[1].Item.ItemID != 1001 || bagUpdate.Updates[1].Item.Quantity != 2 {
		t.Fatalf("bagUpdate.Updates[1].Item = %#v, want item 1001 x2", bagUpdate.Updates[1].Item)
	}

	var walletUpdate protocol.WalletUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[2].Body, &walletUpdate); err != nil {
		t.Fatalf("UnmarshalBody(walletUpdate) error = %v", err)
	}
	if walletUpdate.ReasonType != "shop_buy" || walletUpdate.ReasonRefID != 93002 {
		t.Fatalf("walletUpdate = %#v, want shop_buy/93002", walletUpdate)
	}
	if walletUpdate.Wallet.TotalCopper != 2344678 {
		t.Fatalf("walletUpdate.Wallet.TotalCopper = %d, want 2344678", walletUpdate.Wallet.TotalCopper)
	}
}

func TestRouterHandleNPCActionPushesQuestUpdate(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	mustHandleJSONPacket(t, router, conn, protocol.CmdMoveIntentReq, 200, protocol.MoveIntentReq{
		OpID:          20,
		MoveSeq:       20,
		SceneID:       1,
		TargetSceneID: 2,
		PortalID:      1001,
	})
	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 201, protocol.QuestSubmitReq{QuestID: 1001})
	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 202, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93001})
	clearPackets(conn)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}
	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCActionReq, 203, protocol.NPCActionReq{EntityID: 93001, EntryID: "dialog_market_news"})

	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdNPCActionResp {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdNPCActionResp)
	}

	questUpdates := collectQuestUpdatesByID(t, conn.packets[1:])
	if got := questUpdates[1002].State; got != quest.StateReadyToSubmit {
		t.Fatalf("quest 1002 state = %q, want %q", got, quest.StateReadyToSubmit)
	}
}

func TestRouterHandleQuestSubmitRequiresConfiguredNPC(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	mustHandleJSONPacket(t, router, conn, protocol.CmdMoveIntentReq, 210, protocol.MoveIntentReq{
		OpID:          21,
		MoveSeq:       21,
		SceneID:       1,
		TargetSceneID: 2,
		PortalID:      1001,
	})
	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 211, protocol.QuestSubmitReq{QuestID: 1001})
	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 212, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93001})
	clearPackets(conn)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}
	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCActionReq, 213, protocol.NPCActionReq{EntityID: 93001, EntryID: "dialog_market_news"})
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 214, protocol.QuestSubmitReq{QuestID: 1002, NPCID: 93002})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) after wrong submit = %d, want 1", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdErrorPush {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdErrorPush)
	}
	var errPush protocol.ErrorPush
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &errPush); err != nil {
		t.Fatalf("UnmarshalBody(errPush) error = %v", err)
	}
	if errPush.Msg != "quest submit npc mismatch" {
		t.Fatalf("errPush.Msg = %q, want %q", errPush.Msg, "quest submit npc mismatch")
	}
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 215, protocol.QuestSubmitReq{QuestID: 1002, NPCID: 93001})
	if len(conn.packets) != 6 {
		t.Fatalf("len(conn.packets) after correct submit = %d, want 6", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdQuestSubmitResp {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdQuestSubmitResp)
	}
	var submitResp protocol.QuestSubmitResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &submitResp); err != nil {
		t.Fatalf("UnmarshalBody(submitResp) error = %v", err)
	}
	if !submitResp.Accepted {
		t.Fatalf("submitResp.Accepted = false, want true")
	}
	if submitResp.Quest.State != quest.StateCompleted {
		t.Fatalf("submitResp.Quest.State = %q, want %q", submitResp.Quest.State, quest.StateCompleted)
	}
	if len(submitResp.Rewards) != 2 {
		t.Fatalf("len(submitResp.Rewards) = %d, want 2 popup rewards (gold + item)", len(submitResp.Rewards))
	}
	if submitResp.Rewards[0].Type != "gold" || submitResp.Rewards[0].Value != 150 {
		t.Fatalf("submitResp.Rewards[0] = %#v, want gold 150", submitResp.Rewards[0])
	}
	if submitResp.Rewards[1].Type != "item" || submitResp.Rewards[1].ItemID != 2001 || submitResp.Rewards[1].Count != 2 {
		t.Fatalf("submitResp.Rewards[1] = %#v, want item 2001 x2", submitResp.Rewards[1])
	}
	if conn.packets[1].Cmd != protocol.CmdWalletUpdatePush {
		t.Fatalf("conn.packets[1].Cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdWalletUpdatePush)
	}
	var walletUpdate protocol.WalletUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &walletUpdate); err != nil {
		t.Fatalf("UnmarshalBody(walletUpdate) error = %v", err)
	}
	if walletUpdate.ReasonType != "quest_reward" || walletUpdate.ReasonRefID != 1002 {
		t.Fatalf("walletUpdate = %#v, want quest_reward/1002", walletUpdate)
	}
	if walletUpdate.Wallet.TotalCopper != 2345928 {
		t.Fatalf("walletUpdate.Wallet.TotalCopper = %d, want 2345928 after +100 and +150 copper quest rewards", walletUpdate.Wallet.TotalCopper)
	}
	if conn.packets[2].Cmd != protocol.CmdBagUpdatePush {
		t.Fatalf("conn.packets[2].Cmd = %d, want %d", conn.packets[2].Cmd, protocol.CmdBagUpdatePush)
	}
	var bagUpdate protocol.BagUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[2].Body, &bagUpdate); err != nil {
		t.Fatalf("UnmarshalBody(bagUpdate) error = %v", err)
	}
	if bagUpdate.ContainerType != bag.ContainerTypeBag {
		t.Fatalf("bagUpdate.ContainerType = %q, want %q", bagUpdate.ContainerType, bag.ContainerTypeBag)
	}
	if conn.packets[3].Cmd != protocol.CmdPetUpdatePush {
		t.Fatalf("conn.packets[3].Cmd = %d, want %d", conn.packets[3].Cmd, protocol.CmdPetUpdatePush)
	}
	var petUpdate protocol.PetUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[3].Body, &petUpdate); err != nil {
		t.Fatalf("UnmarshalBody(petUpdate) error = %v", err)
	}
	if petUpdate.Pet.PetID != 102 {
		t.Fatalf("petUpdate.Pet.PetID = %d, want 102", petUpdate.Pet.PetID)
	}

	questUpdates := collectQuestUpdatesByID(t, conn.packets[4:])
	if got := questUpdates[1003].State; got != quest.StateAccepted {
		t.Fatalf("quest 1003 state = %q, want %q", got, quest.StateAccepted)
	}
}

func TestRouterHandleQuestAcceptRequiresConfiguredNPC(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)

	mustHandleJSONPacket(t, router, conn, protocol.CmdMoveIntentReq, 215, protocol.MoveIntentReq{
		OpID:          23,
		MoveSeq:       23,
		SceneID:       1,
		TargetSceneID: 2,
		PortalID:      1001,
	})
	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 216, protocol.QuestSubmitReq{QuestID: 1001})
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 217, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93002})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) after wrong accept = %d, want 1", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdErrorPush {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdErrorPush)
	}
	var errPush protocol.ErrorPush
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &errPush); err != nil {
		t.Fatalf("UnmarshalBody(errPush) error = %v", err)
	}
	if errPush.Msg != "quest accept npc mismatch" {
		t.Fatalf("errPush.Msg = %q, want %q", errPush.Msg, "quest accept npc mismatch")
	}
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 218, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93001})
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) after correct accept = %d, want 3", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdQuestAcceptResp {
		t.Fatalf("conn.packets[0].Cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdQuestAcceptResp)
	}
	var acceptResp protocol.QuestAcceptResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &acceptResp); err != nil {
		t.Fatalf("UnmarshalBody(acceptResp) error = %v", err)
	}
	if !acceptResp.Accepted {
		t.Fatalf("acceptResp.Accepted = false, want true")
	}
	if acceptResp.Quest.State != quest.StateAccepted {
		t.Fatalf("acceptResp.Quest.State = %q, want %q", acceptResp.Quest.State, quest.StateAccepted)
	}
	questUpdates := collectQuestUpdatesByID(t, conn.packets[1:])
	if got := questUpdates[1002].State; got != quest.StateAccepted {
		t.Fatalf("quest 1002 state = %q, want %q", got, quest.StateAccepted)
	}
}

func TestRouterHandleMoveIntentSceneTransferToBeiLu(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 140, 0, protocol.MoveIntentReq{
		OpID:          20,
		MoveSeq:       8,
		SceneID:       3,
		TargetSceneID: 4,
		PortalID:      3002,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}

	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) < 2 {
		t.Fatalf("len(conn.packets) = %d, want at least move response and world resync", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdMoveIntentResp || conn.packets[1].Cmd != protocol.CmdWorldResyncPush {
		t.Fatalf("packet order = [%d,%d], want MOVE_INTENT_RESP then WORLD_RESYNC_PUSH", conn.packets[0].Cmd, conn.packets[1].Cmd)
	}

	var resp protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted || resp.SceneID != 4 {
		t.Fatalf("resp = %+v, want accepted scene 4", resp)
	}

	var resync protocol.WorldResyncPush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &resync); err != nil {
		t.Fatalf("UnmarshalBody(resync) error = %v", err)
	}
	if resync.SceneID != 4 {
		t.Fatalf("resync.SceneID = %d, want 4", resync.SceneID)
	}
	if resync.SelfPos.X != 2 || resync.SelfPos.Y != 8 {
		t.Fatalf("resync.SelfPos = (%d,%d), want (2,8)", resync.SelfPos.X, resync.SelfPos.Y)
	}
}

func TestRouterHandleMoveIntentRejectUnknownPortal(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 141, 0, protocol.MoveIntentReq{
		OpID:          21,
		MoveSeq:       9,
		SceneID:       1,
		TargetSceneID: 2,
		PortalID:      9999,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}

	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
	}

	var resp protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if resp.Accepted {
		t.Fatalf("resp.Accepted = true, want false")
	}
	if resp.Reason != "portal unavailable" {
		t.Fatalf("resp.Reason = %q, want portal unavailable", resp.Reason)
	}

	var resync protocol.WorldResyncPush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &resync); err != nil {
		t.Fatalf("UnmarshalBody(resync) error = %v", err)
	}
	if resync.SceneID != 1 {
		t.Fatalf("resync.SceneID = %d, want 1", resync.SceneID)
	}

	profile, err := playerService.GetProfile(context.Background(), 10001)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.SceneID != 1 {
		t.Fatalf("profile.SceneID = %d, want 1", profile.SceneID)
	}
}

func TestRouterHandleMoveIntentRejectUnknownScene(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 15, 0, protocol.MoveIntentReq{
		OpID:          3,
		MoveSeq:       5,
		SceneID:       1,
		TargetSceneID: 99,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}

	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
	}

	respPacket := conn.packets[0]
	var resp protocol.MoveIntentResp
	if err := protocol.UnmarshalBody(respPacket.Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if resp.Accepted {
		t.Fatalf("resp.Accepted = true, want false")
	}
	if resp.Reason != "target scene unavailable" {
		t.Fatalf("resp.Reason = %q, want target scene unavailable", resp.Reason)
	}

	resyncPacket := conn.packets[1]
	var resync protocol.WorldResyncPush
	if err := protocol.UnmarshalBody(resyncPacket.Body, &resync); err != nil {
		t.Fatalf("UnmarshalBody(resync) error = %v", err)
	}
	if resync.SceneID != 1 {
		t.Fatalf("resync.SceneID = %d, want 1", resync.SceneID)
	}

	profile, err := playerService.GetProfile(context.Background(), 10001)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.SceneID != 1 {
		t.Fatalf("profile.SceneID = %d, want 1", profile.SceneID)
	}
}

func TestRouterHandleNPCMenu(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	packet, err := protocol.NewJSONPacket(protocol.CmdNPCMenuReq, 160, 0, protocol.NPCMenuReq{EntityID: 93001})
	if err != nil {
		t.Fatalf("NewJSONPacket(npc menu) error = %v", err)
	}
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket(npc menu) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(npc menu) error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	var resp protocol.NPCMenuResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("resp.Accepted = false, want true")
	}
	if resp.EntityID != 93001 {
		t.Fatalf("resp.EntityID = %d, want 93001", resp.EntityID)
	}
	if len(resp.MenuEntries) != 2 {
		t.Fatalf("len(resp.MenuEntries) = %d, want 2", len(resp.MenuEntries))
	}
	var sawDialogueIntro bool
	for _, entry := range resp.MenuEntries {
		if entry.EntryID == "dialog_market_intro" && entry.EntryType == "dialog" {
			sawDialogueIntro = true
			break
		}
	}
	if !sawDialogueIntro {
		t.Fatalf("resp.MenuEntries = %#v, want dialog_market_intro entry", resp.MenuEntries)
	}
}

// TestRouterHandleNPCMenuBatch 验证客户端只需一次请求即可取得当前地图全部 NPC 菜单。
func TestRouterHandleNPCMenuBatch(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCMenuBatchReq, 161, protocol.NPCMenuBatchReq{SceneID: 3})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdNPCMenuBatchResp {
		t.Fatalf("response cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdNPCMenuBatchResp)
	}

	var resp protocol.NPCMenuBatchResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted || resp.SceneID != 3 {
		t.Fatalf("resp = %#v, want accepted scene 3", resp)
	}
	if len(resp.Menus) != 2 {
		t.Fatalf("len(resp.Menus) = %d, want 2", len(resp.Menus))
	}
	menuByEntityID := make(map[uint64]protocol.NPCMenuResp, len(resp.Menus))
	for _, menu := range resp.Menus {
		menuByEntityID[menu.EntityID] = menu
	}
	for _, entityID := range []uint64{93001, 93002} {
		menu, exists := menuByEntityID[entityID]
		if !exists || !menu.Accepted || len(menu.MenuEntries) == 0 {
			t.Fatalf("menuByEntityID[%d] = %#v, want populated menu", entityID, menu)
		}
	}
}

func TestRouterHandleInteractRejectsNPCMenu(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	packet, err := protocol.NewJSONPacket(protocol.CmdInteractReq, 160, 0, protocol.InteractReq{EntityID: 93001})
	if err != nil {
		t.Fatalf("NewJSONPacket(interact) error = %v", err)
	}
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket(interact) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(interact) error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	var resp protocol.InteractResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if resp.Accepted {
		t.Fatalf("resp.Accepted = true, want false")
	}
	if resp.Reason != "use npc menu request" {
		t.Fatalf("resp.Reason = %q, want use npc menu request", resp.Reason)
	}
}

func TestRouterHandleInteractMenuShowsQuestEntryAfterUnlock(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	mustHandleJSONPacket(t, router, conn, protocol.CmdMoveIntentReq, 162, protocol.MoveIntentReq{
		OpID:          22,
		MoveSeq:       22,
		SceneID:       1,
		TargetSceneID: 2,
		PortalID:      1001,
	})
	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 163, protocol.QuestSubmitReq{QuestID: 1001})
	clearPackets(conn)
	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCMenuReq, 163, protocol.NPCMenuReq{EntityID: 93001})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	var resp protocol.NPCMenuResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if len(resp.MenuEntries) != 3 {
		t.Fatalf("len(resp.MenuEntries) = %d, want 3", len(resp.MenuEntries))
	}
	if resp.MenuEntries[0].EntryType != "quest" {
		t.Fatalf("resp.MenuEntries[0].EntryType = %q, want quest", resp.MenuEntries[0].EntryType)
	}
	if resp.MenuEntries[0].QuestID != 1002 {
		t.Fatalf("resp.MenuEntries[0].QuestID = %d, want 1002", resp.MenuEntries[0].QuestID)
	}
	if resp.MenuEntries[0].QuestState != quest.StateAvailable {
		t.Fatalf("resp.MenuEntries[0].QuestState = %q, want %q", resp.MenuEntries[0].QuestState, quest.StateAvailable)
	}
}

func TestRouterHandleNPCAction(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	packet, err := protocol.NewJSONPacket(protocol.CmdNPCActionReq, 161, 0, protocol.NPCActionReq{EntityID: 93002, EntryID: "shop_open_market"})
	if err != nil {
		t.Fatalf("NewJSONPacket(npc action) error = %v", err)
	}
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket(npc action) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(npc action) error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	var resp protocol.NPCActionResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("resp.Accepted = false, want true")
	}
	if resp.ResultType != "shop" {
		t.Fatalf("resp.ResultType = %q, want shop", resp.ResultType)
	}
	if resp.Shop == nil || len(resp.Shop.Goods) == 0 {
		t.Fatalf("resp.Shop = %#v, want non-empty shop payload", resp.Shop)
	}
	if resp.EntityID != 93002 {
		t.Fatalf("resp.EntityID = %d, want 93002", resp.EntityID)
	}
	if len(resp.MenuEntries) != 3 {
		t.Fatalf("len(resp.MenuEntries) = %d, want 3", len(resp.MenuEntries))
	}
}

func TestRouterHandleNPCDialogueFlow(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCActionReq, 164, protocol.NPCActionReq{EntityID: 93001, EntryID: "dialog_market_intro"})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}
	var actionResp protocol.NPCActionResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &actionResp); err != nil {
		t.Fatalf("UnmarshalBody(actionResp) error = %v", err)
	}
	if actionResp.ResultType != "dialogue" {
		t.Fatalf("actionResp.ResultType = %q, want dialogue", actionResp.ResultType)
	}
	if actionResp.Dialogue == nil || actionResp.Dialogue.NodeType != "line" {
		t.Fatalf("actionResp.Dialogue = %#v, want first line node", actionResp.Dialogue)
	}

	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCDialogueNextReq, 165, protocol.NPCDialogueNextReq{
		EntityID:   93001,
		DialogueID: actionResp.Dialogue.DialogueID,
		NodeID:     actionResp.Dialogue.NodeID,
	})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) after next = %d, want 1", len(conn.packets))
	}
	var nextResp protocol.NPCDialogueResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &nextResp); err != nil {
		t.Fatalf("UnmarshalBody(nextResp) error = %v", err)
	}
	if !nextResp.Accepted || nextResp.Node == nil || nextResp.Node.NodeType != "action" {
		t.Fatalf("nextResp = %#v, want accepted action node", nextResp)
	}
	if nextResp.Node.ClientAnimationKey != "market_limeng_step_aside" {
		t.Fatalf("nextResp.Node.ClientAnimationKey = %q, want market_limeng_step_aside", nextResp.Node.ClientAnimationKey)
	}

	clearPackets(conn)
	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCDialogueNextReq, 166, protocol.NPCDialogueNextReq{
		EntityID:   93001,
		DialogueID: nextResp.Node.DialogueID,
		NodeID:     nextResp.Node.NodeID,
	})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) after action next = %d, want 1", len(conn.packets))
	}
	var choiceResp protocol.NPCDialogueResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &choiceResp); err != nil {
		t.Fatalf("UnmarshalBody(choiceResp) error = %v", err)
	}
	if choiceResp.Node == nil || choiceResp.Node.NodeType != "choice" {
		t.Fatalf("choiceResp.Node = %#v, want choice", choiceResp.Node)
	}
	if len(choiceResp.Node.Options) != 2 {
		t.Fatalf("len(choiceResp.Node.Options) = %d, want 2", len(choiceResp.Node.Options))
	}
}

func TestRouterHandleNPCActionBattle(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	packet, err := protocol.NewJSONPacket(protocol.CmdNPCActionReq, 171, 0, protocol.NPCActionReq{EntityID: 93002, EntryID: "battle_market_guard"})
	if err != nil {
		t.Fatalf("NewJSONPacket(npc battle action) error = %v", err)
	}
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket(npc battle action) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(npc battle action) error = %v", err)
	}
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
	}

	var resp protocol.NPCActionResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("resp.Accepted = false, reason=%q", resp.Reason)
	}
	if resp.ResultType != "battle" {
		t.Fatalf("resp.ResultType = %q, want battle", resp.ResultType)
	}
	if conn.packets[1].Cmd != protocol.CmdBattleStartPush {
		t.Fatalf("conn.packets[1].Cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdBattleStartPush)
	}
}

func TestRouterHandleInteractAndBattleAction(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)

	interactPacket, err := protocol.NewJSONPacket(protocol.CmdInteractReq, 16, 0, protocol.InteractReq{
		EntityID: 90001,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket(interact) error = %v", err)
	}
	raw, err := protocol.EncodePacket(interactPacket)
	if err != nil {
		t.Fatalf("EncodePacket(interact) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(interact) error = %v", err)
	}
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
	}

	var start protocol.BattleStartPush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &start); err != nil {
		t.Fatalf("UnmarshalBody(start) error = %v", err)
	}
	if start.BattleID == 0 {
		t.Fatalf("start.BattleID = 0, want non-zero")
	}
	if len(start.Allies) != 3 || len(start.Enemies) != 1 {
		t.Fatalf("unexpected actor counts allies=%d enemies=%d", len(start.Allies), len(start.Enemies))
	}
	character := mustFindProtocolBattleActorByUnitClass(t, start.Allies, battle.ActorUnitClassCharacter)
	firstPet := mustFindProtocolBattleActorByUnitClass(t, start.Allies, battle.ActorUnitClassPet)
	if start.ActiveActorID != character.ActorID {
		t.Fatalf("start.ActiveActorID = %d, want %d", start.ActiveActorID, character.ActorID)
	}
	if start.ActivePetUID != 0 {
		t.Fatalf("start.ActivePetUID = %d, want 0 on the opening character turn", start.ActivePetUID)
	}
	if character.UnitClass != battle.ActorUnitClassCharacter {
		t.Fatalf("character.UnitClass = %d, want %d", character.UnitClass, battle.ActorUnitClassCharacter)
	}
	if len(character.SkillIDs) != 1 {
		t.Fatalf("len(character.SkillIDs) = %d, want 1", len(character.SkillIDs))
	}
	if len(character.Skills) != 1 {
		t.Fatalf("len(character.Skills) = %d, want 1", len(character.Skills))
	}
	if character.Skills[0].TargetType != "enemy_single" {
		t.Fatalf("character.Skills[0].TargetType = %q, want %q", character.Skills[0].TargetType, "enemy_single")
	}
	if character.Skills[0].AnimationKey == "" || character.Skills[0].CastColor == "" || character.Skills[0].ImpactColor == "" {
		t.Fatalf("character.Skills[0] = %#v, want visual metadata from battle skill snapshot", character.Skills[0])
	}
	if firstPet.LineupIndex != 0 {
		t.Fatalf("firstPet.LineupIndex = %d, want 0", firstPet.LineupIndex)
	}
	if len(start.Allies[2].Skills) != 1 {
		t.Fatalf("len(start.Allies[2].Skills) = %d, want 1", len(start.Allies[2].Skills))
	}
	if start.Allies[2].Skills[0].TargetType != "ally_single" {
		t.Fatalf("start.Allies[2].Skills[0].TargetType = %q, want %q", start.Allies[2].Skills[0].TargetType, "ally_single")
	}
	if start.Allies[2].Skills[0].AnimationKey != "heal" || start.Allies[2].Skills[0].Projectile {
		t.Fatalf("start.Allies[2].Skills[0] = %#v, want heal animation without projectile", start.Allies[2].Skills[0])
	}

	firstAction, err := protocol.NewJSONPacket(protocol.CmdBattleActionReq, 17, 0, protocol.BattleActionReq{
		OpID:       1,
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    character.ActorID,
		SkillID:    character.SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket(firstAction) error = %v", err)
	}
	raw, err = protocol.EncodePacket(firstAction)
	if err != nil {
		t.Fatalf("EncodePacket(firstAction) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(firstAction) error = %v", err)
	}
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) after first action = %d, want 3", len(conn.packets))
	}

	secondAction, err := protocol.NewJSONPacket(protocol.CmdBattleActionReq, 18, 0, protocol.BattleActionReq{
		OpID:       2,
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    firstPet.ActorID,
		SkillID:    firstPet.SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket(secondAction) error = %v", err)
	}
	raw, err = protocol.EncodePacket(secondAction)
	if err != nil {
		t.Fatalf("EncodePacket(secondAction) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(secondAction) error = %v", err)
	}
	if len(conn.packets) != 4 {
		t.Fatalf("len(conn.packets) after second action = %d, want 4", len(conn.packets))
	}

	thirdAction, err := protocol.NewJSONPacket(protocol.CmdBattleActionReq, 19, 0, protocol.BattleActionReq{
		OpID:       3,
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[2].ActorID,
		SkillID:    start.Allies[2].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if err != nil {
		t.Fatalf("NewJSONPacket(thirdAction) error = %v", err)
	}
	raw, err = protocol.EncodePacket(thirdAction)
	if err != nil {
		t.Fatalf("EncodePacket(thirdAction) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(thirdAction) error = %v", err)
	}
	if len(conn.packets) != 10 {
		t.Fatalf("len(conn.packets) after third action = %d, want 10", len(conn.packets))
	}
	if conn.packets[6].Cmd != protocol.CmdBattleResultPush {
		t.Fatalf("conn.packets[6].Cmd = %d, want %d", conn.packets[6].Cmd, protocol.CmdBattleResultPush)
	}

	var result protocol.BattleResultPush
	if err := protocol.UnmarshalBody(conn.packets[6].Body, &result); err != nil {
		t.Fatalf("UnmarshalBody(result) error = %v", err)
	}
	if !result.Win {
		t.Fatalf("result.Win = false, want true")
	}
	if result.RewardPlayerExp == 0 {
		t.Fatal("result.RewardPlayerExp = 0, want positive battle player exp reward")
	}
	if len(result.Rewards) == 0 {
		t.Fatal("len(result.Rewards) = 0, want popup rewards from battle settlement")
	}
	if result.PlayerExp == 0 {
		t.Fatal("result.PlayerExp = 0, want persisted player exp")
	}
	if len(result.PetRewards) != 2 {
		t.Fatalf("len(result.PetRewards) = %d, want 2", len(result.PetRewards))
	}
	if len(result.DropTexts) == 0 {
		t.Fatal("len(result.DropTexts) = 0, want text-only drop preview")
	}
	if conn.packets[7].Cmd != protocol.CmdPetUpdatePush {
		t.Fatalf("conn.packets[7].Cmd = %d, want %d", conn.packets[7].Cmd, protocol.CmdPetUpdatePush)
	}

	if conn.packets[8].Cmd != protocol.CmdPetUpdatePush {
		t.Fatalf("conn.packets[8].Cmd = %d, want %d", conn.packets[8].Cmd, protocol.CmdPetUpdatePush)
	}
	var allyHPAfterBattle uint32
	var sawFirstPetExpUpdate bool
	for packetIndex := 7; packetIndex <= 8; packetIndex++ {
		var petUpdate protocol.PetUpdatePush
		if err := protocol.UnmarshalBody(conn.packets[packetIndex].Body, &petUpdate); err != nil {
			t.Fatalf("UnmarshalBody(petUpdate %d) error = %v", packetIndex, err)
		}
		if petUpdate.Pet.PetUID != start.Allies[1].PetUID {
			continue
		}
		allyHPAfterBattle = petUpdate.Pet.HP
		if petUpdate.Pet.Exp > 0 {
			sawFirstPetExpUpdate = true
		}
	}
	if !sawFirstPetExpUpdate {
		t.Fatal("first pet exp update not found or exp did not increase")
	}
	if conn.packets[9].Cmd != protocol.CmdBagUpdatePush {
		t.Fatalf("conn.packets[9].Cmd = %d, want %d", conn.packets[9].Cmd, protocol.CmdBagUpdatePush)
	}
	if conn.packets[5].Cmd != protocol.CmdBattleStatePush {
		t.Fatalf("conn.packets[5].Cmd = %d, want %d", conn.packets[5].Cmd, protocol.CmdBattleStatePush)
	}
	if allyHPAfterBattle == 0 {
		t.Fatalf("allyHPAfterBattle = 0, want non-zero")
	}
	var bagUpdate protocol.BagUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[9].Body, &bagUpdate); err != nil {
		t.Fatalf("UnmarshalBody(bagUpdate) error = %v", err)
	}
	if bagUpdate.ContainerType != bag.ContainerTypeBag {
		t.Fatalf("bagUpdate.ContainerType = %q, want %q", bagUpdate.ContainerType, bag.ContainerTypeBag)
	}

	petListPacket, err := protocol.NewJSONPacket(protocol.CmdPetListReq, 20, 0, protocol.PetListReq{})
	if err != nil {
		t.Fatalf("NewJSONPacket(petList) error = %v", err)
	}
	raw, err = protocol.EncodePacket(petListPacket)
	if err != nil {
		t.Fatalf("EncodePacket(petList) error = %v", err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(petList) error = %v", err)
	}
	if len(conn.packets) != 11 {
		t.Fatalf("len(conn.packets) after pet list = %d, want 11", len(conn.packets))
	}

	var petList protocol.PetListResp
	if err := protocol.UnmarshalBody(conn.packets[10].Body, &petList); err != nil {
		t.Fatalf("UnmarshalBody(petList) error = %v", err)
	}
	if len(petList.Pets) == 0 {
		t.Fatalf("len(petList.Pets) = 0, want non-zero")
	}
	if petList.Pets[0].PetUID != start.Allies[1].PetUID {
		t.Fatalf("petList.Pets[0].PetUID = %d, want %d", petList.Pets[0].PetUID, start.Allies[1].PetUID)
	}
	if petList.Pets[0].HP != allyHPAfterBattle {
		t.Fatalf("petList.Pets[0].HP = %d, want %d", petList.Pets[0].HP, allyHPAfterBattle)
	}
	if len(petList.Lineup) == 0 {
		t.Fatalf("len(petList.Lineup) = 0, want non-zero")
	}
	if petList.Lineup[0].HP != allyHPAfterBattle {
		t.Fatalf("petList.Lineup[0].HP = %d, want %d", petList.Lineup[0].HP, allyHPAfterBattle)
	}
	mustHandleJSONPacket(t, router, conn, protocol.CmdPetSkillDetailReq, 21, protocol.PetSkillDetailReq{PetUID: petList.Pets[0].PetUID})
	if len(conn.packets) != 12 {
		t.Fatalf("len(conn.packets) after pet detail = %d, want 12", len(conn.packets))
	}
	var petDetail protocol.PetSkillDetailResp
	if err := protocol.UnmarshalBody(conn.packets[11].Body, &petDetail); err != nil {
		t.Fatalf("UnmarshalBody(petDetail) error = %v", err)
	}
	if petDetail.Pet.Exp == 0 {
		t.Fatalf("petDetail.Pet.Exp = %d, want persisted reward exp", petDetail.Pet.Exp)
	}
}

func TestSinglePlayerBattleDisconnectFailsWithoutRewards(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := teststub.NewTestPlayerService()
	petService := pet.NewService(teststub.NewPetRepository(), nil, nil, nil)
	walletService := wallet.NewService(teststub.NewWalletRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleService := battle.NewService(nil)
	battleHandler := NewBattleHandler(sessionService, playerService, petService, nil, walletService, worldService, questService, npcService, npcdialogue.NewService(teststub.NewNPCDialogueRepository(), &npcdialogue.QuestServiceAdapter{Service: questService}), battleService, teststub.NewBattleRepository(), nil, nil)
	sessionService.SetDisconnectHandler(battleHandler.HandleSessionDisconnect)

	conn := &fakeConn{id: "disconnect-battle-conn"}
	if _, err := sessionService.Bind(teststub.DemoPlayerID, conn); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	ctx := context.Background()
	profile, err := playerService.GetProfile(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	lineup, err := petService.ListLineup(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("ListLineup() error = %v", err)
	}
	sceneSnapshot, err := worldService.GetSceneSnapshot(ctx, teststub.DemoPlayerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
	if err != nil {
		t.Fatalf("GetSceneSnapshot() error = %v", err)
	}
	target, found := findInteractTarget(sceneSnapshot.NearbyEntities, 90001)
	if !found {
		t.Fatal("findInteractTarget() = false, want nearby npc")
	}

	start, err := battleService.StartPVE(ctx, profile, lineup, target, battle.EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if _, err := battleService.SubmitAction(ctx, profile.PlayerID, battle.ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	}); err != nil {
		t.Fatalf("SubmitAction(first ally) error = %v", err)
	}

	sessionService.Disconnect(conn.ID())
	if err := battleHandler.ProcessAutoCustodyOnce(ctx); err != nil {
		t.Fatalf("ProcessAutoCustodyOnce() error = %v", err)
	}
	if len(conn.packets) != 0 {
		t.Fatalf("len(conn.packets) = %d, want 0 because disconnected players should not receive pushes", len(conn.packets))
	}

	_, err = battleService.SubmitAction(ctx, profile.PlayerID, battle.ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	})
	if !errors.Is(err, battle.ErrBattleNotFound) {
		t.Fatalf("SubmitAction(after disconnect failure) error = %v, want ErrBattleNotFound", err)
	}

	if _, state, ok := battleService.GetActiveSnapshot(ctx, profile.PlayerID); ok || state != nil {
		t.Fatalf("active snapshot ok=%v state=%#v, want battle removed after disconnect failure", ok, state)
	}
}

func TestRouterHandleReconnectRestoresWorldAndBattleSnapshots(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := teststub.NewTestPlayerService()
	petRepo := teststub.NewPetRepository()
	bagRepo := teststub.NewBagRepository()
	bagRepo.BindPetRepository(petRepo)
	petService := pet.NewService(petRepo, nil, nil, nil)
	bagService := bag.NewService(bagRepo)
	walletService := wallet.NewService(teststub.NewWalletRepository())
	unlockService := unlock.NewService(teststub.NewUnlockRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, walletService, worldService, nil, nil)
	petHandler := NewPetHandler(sessionService, petService, nil)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleService := battle.NewService(nil)
	battleHandler := NewBattleHandler(sessionService, playerService, petService, bagService, walletService, worldService, questService, npcService, npcdialogue.NewService(teststub.NewNPCDialogueRepository(), &npcdialogue.QuestServiceAdapter{Service: questService}), battleService, teststub.NewBattleRepository(), nil, nil)
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, NewPlayerHandler(sessionService, playerService), nil, battleHandler, nil, NewQuestHandler(questService, sessionService, bagService, petService, walletService, unlockService, playerService), sessionService)

	firstConn := &fakeConn{id: "reconnect-old-conn"}
	sess, err := sessionService.Bind(teststub.DemoPlayerID, firstConn)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	originalReconnectToken := sess.ReconnectToken

	ctx := context.Background()
	profile, err := playerService.GetProfile(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	lineup, err := petService.ListLineup(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("ListLineup() error = %v", err)
	}
	target := world.Entity{
		EntityID:   90004,
		EntityType: 2,
		Pos:        world.Vec2i{X: profile.PosX + 1, Y: profile.PosY},
		Name:       "ReconnectBattleNPC",
	}
	start, err := battleService.StartPVE(ctx, profile, lineup, target, battle.EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if _, err := battleService.SubmitAction(ctx, profile.PlayerID, battle.ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	}); err != nil {
		t.Fatalf("SubmitAction(first ally) error = %v", err)
	}

	sessionService.Disconnect(firstConn.ID())

	secondConn := &fakeConn{id: "reconnect-new-conn"}
	mustHandleJSONPacket(t, router, secondConn, protocol.CmdReconnectReq, 88, protocol.ReconnectReq{
		ReconnectToken: originalReconnectToken,
		BattleID:       start.BattleID,
		LastFrame:      start.BattleVersion,
	})
	if len(secondConn.packets) != 1 {
		t.Fatalf("len(secondConn.packets) = %d, want 1", len(secondConn.packets))
	}
	if secondConn.packets[0].Cmd != protocol.CmdReconnectResp {
		t.Fatalf("secondConn.packets[0].Cmd = %d, want %d", secondConn.packets[0].Cmd, protocol.CmdReconnectResp)
	}

	var payload protocol.ReconnectResp
	if err := protocol.UnmarshalBody(secondConn.packets[0].Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody(reconnect) error = %v", err)
	}
	if payload.PlayerID != teststub.DemoPlayerID {
		t.Fatalf("payload.PlayerID = %d, want %d", payload.PlayerID, teststub.DemoPlayerID)
	}
	if payload.World == nil || payload.World.SceneID != profile.SceneID {
		t.Fatalf("payload.World = %#v, want scene %d", payload.World, profile.SceneID)
	}
	if payload.BattleStart == nil || payload.BattleState == nil {
		t.Fatalf("payload battle snapshot = start:%#v state:%#v, want both non-nil", payload.BattleStart, payload.BattleState)
	}
	if payload.BattleStart.BattleID != start.BattleID || payload.BattleState.BattleID != start.BattleID {
		t.Fatalf("battle id mismatch start=%d state=%d want %d", payload.BattleStart.BattleID, payload.BattleState.BattleID, start.BattleID)
	}
	if len(payload.BattleState.PendingActorIDs) != 2 {
		t.Fatalf("len(payload.BattleState.PendingActorIDs) = %d, want 2", len(payload.BattleState.PendingActorIDs))
	}
	if len(payload.BattleReplayStates) != 0 {
		t.Fatalf("len(payload.BattleReplayStates) = %d, want 0 because queued intents no longer create replay frames", len(payload.BattleReplayStates))
	}
	if payload.ReconnectToken == originalReconnectToken {
		t.Fatal("reconnect token was not rotated after reconnect")
	}
}

func TestRouterHandleReconnectReturnsFailedBattleResultAfterDisconnect(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := teststub.NewTestPlayerService()
	petRepo := teststub.NewPetRepository()
	bagRepo := teststub.NewBagRepository()
	bagRepo.BindPetRepository(petRepo)
	petService := pet.NewService(petRepo, nil, nil, nil)
	bagService := bag.NewService(bagRepo)
	walletService := wallet.NewService(teststub.NewWalletRepository())
	unlockService := unlock.NewService(teststub.NewUnlockRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, walletService, worldService, nil, nil)
	petHandler := NewPetHandler(sessionService, petService, nil)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleService := battle.NewService(nil)
	battleHandler := NewBattleHandler(sessionService, playerService, petService, bagService, walletService, worldService, questService, npcService, npcdialogue.NewService(teststub.NewNPCDialogueRepository(), &npcdialogue.QuestServiceAdapter{Service: questService}), battleService, teststub.NewBattleRepository(), nil, nil)
	sessionService.SetDisconnectHandler(battleHandler.HandleSessionDisconnect)
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, NewPlayerHandler(sessionService, playerService), nil, battleHandler, nil, NewQuestHandler(questService, sessionService, bagService, petService, walletService, unlockService, playerService), sessionService)

	firstConn := &fakeConn{id: "reconnect-finished-old-conn"}
	sess, err := sessionService.Bind(teststub.DemoPlayerID, firstConn)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	originalReconnectToken := sess.ReconnectToken

	ctx := context.Background()
	profile, err := playerService.GetProfile(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	lineup, err := petService.ListLineup(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("ListLineup() error = %v", err)
	}
	target := world.Entity{
		EntityID:   90001,
		EntityType: 2,
		Pos:        world.Vec2i{X: profile.PosX + 1, Y: profile.PosY},
		Name:       "ReconnectResultNPC",
	}
	start, err := battleService.StartPVE(ctx, profile, lineup, target, battle.EmptyCharacterBattleSkillInput())
	if err != nil {
		t.Fatalf("StartPVE() error = %v", err)
	}
	if _, err := battleService.SubmitAction(ctx, profile.PlayerID, battle.ActionRequest{
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
		TargetID:   start.Enemies[0].ActorID,
	}); err != nil {
		t.Fatalf("SubmitAction(first ally) error = %v", err)
	}

	sessionService.Disconnect(firstConn.ID())
	if err := battleHandler.ProcessAutoCustodyOnce(ctx); err != nil {
		t.Fatalf("ProcessAutoCustodyOnce() error = %v", err)
	}

	secondConn := &fakeConn{id: "reconnect-finished-new-conn"}
	mustHandleJSONPacket(t, router, secondConn, protocol.CmdReconnectReq, 89, protocol.ReconnectReq{
		ReconnectToken: originalReconnectToken,
		BattleID:       start.BattleID,
		LastFrame:      start.BattleVersion,
	})
	if len(secondConn.packets) != 1 {
		t.Fatalf("len(secondConn.packets) = %d, want 1", len(secondConn.packets))
	}

	var payload protocol.ReconnectResp
	if err := protocol.UnmarshalBody(secondConn.packets[0].Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody(reconnect result) error = %v", err)
	}
	if payload.BattleStart != nil || payload.BattleState != nil {
		t.Fatalf("payload battle snapshot = start:%#v state:%#v, want nil because disconnect ended battle", payload.BattleStart, payload.BattleState)
	}
	if payload.BattleResult == nil {
		t.Fatal("payload.BattleResult = nil, want failed battle result after disconnect")
	}
	if payload.BattleResult.Win {
		t.Fatal("payload.BattleResult.Win = true, want false after disconnect")
	}
	if payload.BattleResult.RewardGold != 0 || payload.BattleResult.RewardPlayerExp != 0 || len(payload.BattleResult.Rewards) != 0 || len(payload.BattleResult.DropTexts) != 0 {
		t.Fatalf("payload rewards = gold:%d exp:%d rewards:%d drops:%d, want no rewards", payload.BattleResult.RewardGold, payload.BattleResult.RewardPlayerExp, len(payload.BattleResult.Rewards), len(payload.BattleResult.DropTexts))
	}
}

func TestRouterHandlePVPChallengeAcceptStartsBattleForBothPlayers(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := teststub.NewTestPlayerService()
	petService := pet.NewService(teststub.NewPetRepository(), nil, nil, nil)
	bagService := bag.NewService(teststub.NewBagRepository())
	itemService := item.NewService(teststub.NewItemRepository())
	walletService := wallet.NewService(teststub.NewWalletRepository())
	unlockService := unlock.NewService(teststub.NewUnlockRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, walletService, worldService, nil, nil)
	petHandler := NewPetHandler(sessionService, petService, nil)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleHandler := NewBattleHandler(sessionService, playerService, petService, bagService, walletService, worldService, questService, npcService, npcdialogue.NewService(teststub.NewNPCDialogueRepository(), &npcdialogue.QuestServiceAdapter{Service: questService}), battle.NewService(nil), teststub.NewBattleRepository(), nil, nil)
	bagHandler := NewBagHandler(sessionService, bagService, itemService, walletService, playerService, petService, nil, worldService, npcService)
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, NewPlayerHandler(sessionService, playerService), nil, battleHandler, bagHandler, NewQuestHandler(questService, sessionService, bagService, petService, walletService, unlockService, playerService), sessionService)

	challengerConn := &fakeConn{id: "pvp-challenger-conn"}
	if _, err := sessionService.Bind(teststub.DemoPlayerID, challengerConn); err != nil {
		t.Fatalf("Bind(challenger) error = %v", err)
	}
	defenderConn := &fakeConn{id: "pvp-defender-conn"}
	if _, err := sessionService.Bind(teststub.RivalPlayerID, defenderConn); err != nil {
		t.Fatalf("Bind(defender) error = %v", err)
	}

	mustHandleJSONPacket(t, router, challengerConn, protocol.CmdPVPChallengeReq, 90, protocol.PVPChallengeReq{
		OpID:           1,
		TargetPlayerID: teststub.RivalPlayerID,
	})
	if len(challengerConn.packets) != 1 {
		t.Fatalf("len(challengerConn.packets) = %d, want 1", len(challengerConn.packets))
	}
	if len(defenderConn.packets) != 1 {
		t.Fatalf("len(defenderConn.packets) = %d, want 1 invite push", len(defenderConn.packets))
	}

	var invite protocol.PVPChallengePush
	if err := protocol.UnmarshalBody(defenderConn.packets[0].Body, &invite); err != nil {
		t.Fatalf("UnmarshalBody(invite) error = %v", err)
	}
	if invite.Challenger.PlayerID != teststub.DemoPlayerID {
		t.Fatalf("invite.Challenger.PlayerID = %d, want %d", invite.Challenger.PlayerID, teststub.DemoPlayerID)
	}

	clearPackets(challengerConn)
	clearPackets(defenderConn)
	mustHandleJSONPacket(t, router, defenderConn, protocol.CmdPVPChallengeReplyReq, 91, protocol.PVPChallengeReplyReq{
		ChallengeID: invite.ChallengeID,
		Accept:      true,
	})
	if len(defenderConn.packets) < 2 {
		t.Fatalf("len(defenderConn.packets) = %d, want reply resp + battle start", len(defenderConn.packets))
	}
	if len(challengerConn.packets) != 1 {
		t.Fatalf("len(challengerConn.packets) = %d, want battle start", len(challengerConn.packets))
	}

	var challengerStart protocol.BattleStartPush
	if err := protocol.UnmarshalBody(challengerConn.packets[0].Body, &challengerStart); err != nil {
		t.Fatalf("UnmarshalBody(challenger start) error = %v", err)
	}
	if challengerStart.BattleType != battle.BattleTypePVP {
		t.Fatalf("challengerStart.BattleType = %d, want %d", challengerStart.BattleType, battle.BattleTypePVP)
	}
	if len(challengerStart.ParticipantPlayerIDs) != 2 {
		t.Fatalf("len(challengerStart.ParticipantPlayerIDs) = %d, want 2", len(challengerStart.ParticipantPlayerIDs))
	}
	if len(challengerStart.PendingActorIDs) < 2 {
		t.Fatalf("len(challengerStart.PendingActorIDs) = %d, want at least 2", len(challengerStart.PendingActorIDs))
	}
}

func buildWorldRouterForTest(t *testing.T) (uint64, *Router, *player.Service, *fakeConn) {
	t.Helper()

	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := teststub.NewTestPlayerService()
	petRepo := teststub.NewPetRepository()
	bagRepo := teststub.NewBagRepository()
	bagRepo.BindPetRepository(petRepo)
	petService := pet.NewService(petRepo, nil, nil, nil)
	bagService := bag.NewService(bagRepo)
	itemService := item.NewService(teststub.NewItemRepository())
	walletService := wallet.NewService(teststub.NewWalletRepository())
	unlockService := unlock.NewService(teststub.NewUnlockRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	monsterService := monster.NewService(teststub.NewMonsterRepository(), nil, nil)
	if err := monsterService.RefreshBattleRewardCache(context.Background()); err != nil {
		t.Fatalf("RefreshBattleRewardCache() error = %v", err)
	}
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, walletService, worldService, monsterService, nil)
	petHandler := NewPetHandler(sessionService, petService, nil)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleHandler := NewBattleHandler(sessionService, playerService, petService, bagService, walletService, worldService, questService, npcService, npcdialogue.NewService(teststub.NewNPCDialogueRepository(), &npcdialogue.QuestServiceAdapter{Service: questService}), battle.NewService(monsterService), teststub.NewBattleRepository(), nil, nil)
	bagHandler := NewBagHandler(sessionService, bagService, itemService, walletService, playerService, petService, nil, worldService, npcService)
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, NewPlayerHandler(sessionService, playerService), nil, battleHandler, bagHandler, NewQuestHandler(questService, sessionService, bagService, petService, walletService, unlockService, playerService), sessionService)

	conn := &fakeConn{id: "conn-1"}
	if _, err := sessionService.Bind(teststub.DemoPlayerID, conn); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	return teststub.DemoPlayerID, router, playerService, conn
}

func mustHandleJSONPacket(t *testing.T, router *Router, conn *fakeConn, cmd uint16, seq uint32, payload any) {
	t.Helper()

	packet, err := protocol.NewJSONPacket(cmd, seq, 0, payload)
	if err != nil {
		t.Fatalf("NewJSONPacket(%d) error = %v", cmd, err)
	}
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket(%d) error = %v", cmd, err)
	}
	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle(%d) error = %v", cmd, err)
	}
}

func clearPackets(conn *fakeConn) {
	conn.packets = nil
}

func mustMovePlayerToScene(t *testing.T, playerService *player.Service, sceneID uint32, posX int32, posY int32) {
	t.Helper()

	if err := playerService.UpdatePosition(context.Background(), 10001, sceneID, posX, posY); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}
}

func collectQuestUpdatesByID(t *testing.T, packets []*protocol.Packet) map[uint64]protocol.QuestSummary {
	t.Helper()

	result := map[uint64]protocol.QuestSummary{}
	for _, packet := range packets {
		if packet.Cmd != protocol.CmdQuestUpdatePush {
			continue
		}
		var payload protocol.QuestUpdatePush
		if err := protocol.UnmarshalBody(packet.Body, &payload); err != nil {
			t.Fatalf("UnmarshalBody(quest update) error = %v", err)
		}
		result[payload.Quest.QuestID] = payload.Quest
	}
	return result
}

func TestEnterWorldIncludesWildEncounterConfig(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)
	if err := playerService.UpdatePosition(context.Background(), 10001, 4, 4, 7); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	mustHandleJSONPacket(t, router, conn, protocol.CmdEnterWorldReq, 21, protocol.EnterWorldReq{})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}
	var response protocol.EnterWorldResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &response); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if !response.WildEncounter.Enabled {
		t.Fatalf("WildEncounter.Enabled = false, want true on scene 4")
	}
	if response.WildEncounter.EncounterRate != 800 {
		t.Fatalf("WildEncounter.EncounterRate = %d, want 800", response.WildEncounter.EncounterRate)
	}
	if len(response.WildEncounter.SpawnMonsterIDs) != 1 || response.WildEncounter.SpawnMonsterIDs[0] != 9001 {
		t.Fatalf("WildEncounter.SpawnMonsterIDs = %v, want [9001]", response.WildEncounter.SpawnMonsterIDs)
	}
	if len(response.WildEncounter.Targets) != 1 {
		t.Fatalf("len(WildEncounter.Targets) = %d, want 1", len(response.WildEncounter.Targets))
	}
	if target := response.WildEncounter.Targets[0]; target.MonsterID != 9001 || target.MonsterName != "野生怪物" || target.SkinID != "史莱姆_001" {
		t.Fatalf("WildEncounter.Targets[0] = %+v, want database monster summary", target)
	}
}

func TestHandleWildEncounterStartsBattle(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)
	if err := playerService.UpdatePosition(context.Background(), 10001, 4, 4, 7); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	mustHandleJSONPacket(t, router, conn, protocol.CmdWildEncounterReq, 31, protocol.WildEncounterReq{
		SceneID: 4,
		MoveSeq: 12,
	})
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdWildEncounterResp {
		t.Fatalf("first cmd = %d, want %d", conn.packets[0].Cmd, protocol.CmdWildEncounterResp)
	}
	var wildResp protocol.WildEncounterResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &wildResp); err != nil {
		t.Fatalf("UnmarshalBody(wild resp) error = %v", err)
	}
	if !wildResp.Accepted {
		t.Fatalf("wildResp.Accepted = false, reason=%s", wildResp.Reason)
	}
	if conn.packets[1].Cmd != protocol.CmdBattleStartPush {
		t.Fatalf("second cmd = %d, want %d", conn.packets[1].Cmd, protocol.CmdBattleStartPush)
	}
}
