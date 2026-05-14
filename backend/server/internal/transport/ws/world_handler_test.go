package wstransport

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/memory"
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
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, sessionService)

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

func TestRouterHandleMoveIntent(t *testing.T) {
	cfg, router, playerService, conn := buildWorldRouterForTest(t)

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 13, 0, protocol.MoveIntentReq{
		OpID:      1,
		MoveSeq:   3,
		SceneID:   1,
		TargetPos: protocol.Vec2i{X: 12, Y: 8},
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
	if resp.MoveSeq != 3 {
		t.Fatalf("resp.MoveSeq = %d, want 3", resp.MoveSeq)
	}

	movePacket := conn.packets[1]
	if movePacket.Cmd != protocol.CmdEntityMovePush {
		t.Fatalf("movePacket.Cmd = %d, want %d", movePacket.Cmd, protocol.CmdEntityMovePush)
	}

	var movePush protocol.EntityMovePush
	if err := protocol.UnmarshalBody(movePacket.Body, &movePush); err != nil {
		t.Fatalf("UnmarshalBody(movePush) error = %v", err)
	}
	if movePush.EntityID != cfg.DemoPlayerID {
		t.Fatalf("movePush.EntityID = %d, want %d", movePush.EntityID, cfg.DemoPlayerID)
	}
	if movePush.ToPos.X != 12 || movePush.ToPos.Y != 8 {
		t.Fatalf("movePush.ToPos = (%d,%d), want (12,8)", movePush.ToPos.X, movePush.ToPos.Y)
	}

	profile, err := playerService.GetProfile(context.Background(), cfg.DemoPlayerID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.PosX != 12 || profile.PosY != 8 {
		t.Fatalf("profile position = (%d,%d), want (12,8)", profile.PosX, profile.PosY)
	}
}

func TestRouterHandleMoveIntentResync(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)

	packet, err := protocol.NewJSONPacket(protocol.CmdMoveIntentReq, 14, 0, protocol.MoveIntentReq{
		OpID:      2,
		MoveSeq:   4,
		SceneID:   1,
		TargetPos: protocol.Vec2i{X: 99, Y: 99},
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
	if resp.Accepted {
		t.Fatalf("resp.Accepted = true, want false")
	}
	if resp.Reason != "target out of bounds" {
		t.Fatalf("resp.Reason = %q, want target out of bounds", resp.Reason)
	}

	resyncPacket := conn.packets[1]
	if resyncPacket.Cmd != protocol.CmdWorldResyncPush {
		t.Fatalf("resyncPacket.Cmd = %d, want %d", resyncPacket.Cmd, protocol.CmdWorldResyncPush)
	}

	var resync protocol.WorldResyncPush
	if err := protocol.UnmarshalBody(resyncPacket.Body, &resync); err != nil {
		t.Fatalf("UnmarshalBody(resync) error = %v", err)
	}
	if resync.SelfPos.X != 8 || resync.SelfPos.Y != 6 {
		t.Fatalf("resync.SelfPos = (%d,%d), want (8,6)", resync.SelfPos.X, resync.SelfPos.Y)
	}

	profile, err := playerService.GetProfile(context.Background(), 10001)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.PosX != 8 || profile.PosY != 6 {
		t.Fatalf("profile position = (%d,%d), want (8,6)", profile.PosX, profile.PosY)
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
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, sessionService)

	conn := &fakeConn{id: "conn-1"}
	if _, err := sessionService.Bind(cfg.DemoPlayerID, conn); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	return cfg, router, playerService, conn
}
