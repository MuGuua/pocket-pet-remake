package wstransport

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/memory"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type fakeConn struct {
	id      string
	packets []*protocol.Packet
	closed  bool
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
	cfg, router, _, conn := buildWorldRouterForTest(t)

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
	if payload.Self.PlayerID != cfg.DemoPlayerID {
		t.Fatalf("payload.Self.PlayerID = %d, want %d", payload.Self.PlayerID, cfg.DemoPlayerID)
	}
	if payload.SceneID != 1 {
		t.Fatalf("payload.SceneID = %d, want 1", payload.SceneID)
	}
	if payload.Gold != 100 {
		t.Fatalf("payload.Gold = %d, want 100", payload.Gold)
	}
	if len(payload.Lineup) != 2 {
		t.Fatalf("len(payload.Lineup) = %d, want 2", len(payload.Lineup))
	}
	if len(payload.NearbyEntities) != 1 {
		t.Fatalf("len(payload.NearbyEntities) = %d, want 1", len(payload.NearbyEntities))
	}
}

func TestRouterRejectUnauthenticatedEnterWorld(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	worldHandler := NewWorldHandler(sessionService, nil, nil, nil)
	battleHandler := NewBattleHandler(sessionService, nil, nil, nil, battle.NewService())
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, battleHandler, sessionService)

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
	cfg, router, playerService, conn := buildWorldRouterForTest(t)

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

	profile, err := playerService.GetProfile(context.Background(), cfg.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.PosX != 8 || profile.PosY != 6 {
		t.Fatalf("profile position = (%d,%d), want (8,6)", profile.PosX, profile.PosY)
	}
}

func TestRouterHandleMoveIntentSceneTransfer(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 14, 0, protocol.MoveIntentReq{
		OpID:          2,
		MoveSeq:       4,
		SceneID:       1,
		TargetSceneID: 2,
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
	if resync.SelfPos.X != 2 || resync.SelfPos.Y != 4 {
		t.Fatalf("resync.SelfPos = (%d,%d), want (2,4)", resync.SelfPos.X, resync.SelfPos.Y)
	}

	profile, err := playerService.GetProfile(context.Background(), 10001)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.SceneID != 2 {
		t.Fatalf("profile.SceneID = %d, want 2", profile.SceneID)
	}
	if profile.PosX != 2 || profile.PosY != 4 {
		t.Fatalf("profile position = (%d,%d), want (2,4)", profile.PosX, profile.PosY)
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
	if len(start.Allies) != 1 || len(start.Enemies) != 1 {
		t.Fatalf("unexpected actor counts allies=%d enemies=%d", len(start.Allies), len(start.Enemies))
	}
	if len(start.Allies[0].SkillIDs) != 2 {
		t.Fatalf("len(start.Allies[0].SkillIDs) = %d, want 2", len(start.Allies[0].SkillIDs))
	}

	firstAction, err := protocol.NewJSONPacket(protocol.CmdBattleActionReq, 17, 0, protocol.BattleActionReq{
		OpID:       1,
		BattleID:   start.BattleID,
		Round:      start.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[0],
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
	if len(conn.packets) != 4 {
		t.Fatalf("len(conn.packets) after first action = %d, want 4", len(conn.packets))
	}

	var state protocol.BattleStatePush
	if err := protocol.UnmarshalBody(conn.packets[3].Body, &state); err != nil {
		t.Fatalf("UnmarshalBody(state) error = %v", err)
	}
	if state.Round != 2 {
		t.Fatalf("state.Round = %d, want 2", state.Round)
	}

	secondAction, err := protocol.NewJSONPacket(protocol.CmdBattleActionReq, 18, 0, protocol.BattleActionReq{
		OpID:       2,
		BattleID:   start.BattleID,
		Round:      state.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[0].ActorID,
		SkillID:    start.Allies[0].SkillIDs[1],
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
	if len(conn.packets) != 7 {
		t.Fatalf("len(conn.packets) after second action = %d, want 7", len(conn.packets))
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
}

func buildWorldRouterForTest(t *testing.T) (config.Config, *Router, *player.Service, *fakeConn) {
	t.Helper()

	cfg := config.Config{
		DemoPlayerName: "DemoTrainer",
		DemoPlayerID:   10001,
	}
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := player.NewService(memory.NewPlayerRepository(cfg))
	petService := pet.NewService(memory.NewPetRepository(cfg))
	worldService := world.NewService(memory.NewWorldRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, worldService)
	battleHandler := NewBattleHandler(sessionService, playerService, petService, worldService, battle.NewService())
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, battleHandler, sessionService)

	conn := &fakeConn{id: "conn-1"}
	if _, err := sessionService.Bind(cfg.DemoPlayerID, conn); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	return cfg, router, playerService, conn
}
