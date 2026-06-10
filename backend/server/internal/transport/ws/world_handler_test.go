package wstransport

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
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
	if payload.Gold != 100 {
		t.Fatalf("payload.Gold = %d, want 100", payload.Gold)
	}
	if len(payload.Lineup) != 2 {
		t.Fatalf("len(payload.Lineup) = %d, want 2", len(payload.Lineup))
	}
	if len(payload.NearbyEntities) != 2 {
		t.Fatalf("len(payload.NearbyEntities) = %d, want 2", len(payload.NearbyEntities))
	}
}

func TestRouterRejectUnauthenticatedEnterWorld(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, nil, nil, questService, nil)
	petHandler := NewPetHandler(sessionService, nil)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleHandler := NewBattleHandler(sessionService, nil, nil, nil, questService, npcService, battle.NewService(), teststub.NewBattleRepository())
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, battleHandler, NewQuestHandler(questService, sessionService), sessionService)

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
	if len(conn.packets) != 4 {
		t.Fatalf("len(conn.packets) = %d, want 4", len(conn.packets))
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
	if got := questUpdates[1001].State; got != quest.StateCompleted {
		t.Fatalf("quest 1001 state = %q, want %q", got, quest.StateCompleted)
	}
	if got := questUpdates[1002].State; got != quest.StateAvailable {
		t.Fatalf("quest 1002 state = %q, want %q", got, quest.StateAvailable)
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
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 201, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93001})
	clearPackets(conn)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}
	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCActionReq, 202, protocol.NPCActionReq{EntityID: 93001, EntryID: "dialog_market_news"})

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
	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 211, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93001})
	clearPackets(conn)

	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}
	mustHandleJSONPacket(t, router, conn, protocol.CmdNPCActionReq, 212, protocol.NPCActionReq{EntityID: 93001, EntryID: "dialog_market_news"})
	clearPackets(conn)

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 213, protocol.QuestSubmitReq{QuestID: 1002, NPCID: 93002})
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

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestSubmitReq, 214, protocol.QuestSubmitReq{QuestID: 1002, NPCID: 93001})
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) after correct submit = %d, want 3", len(conn.packets))
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

	questUpdates := collectQuestUpdatesByID(t, conn.packets[1:])
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

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 216, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93002})
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

	mustHandleJSONPacket(t, router, conn, protocol.CmdQuestAcceptReq, 217, protocol.QuestAcceptReq{QuestID: 1002, NPCID: 93001})
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
	if len(conn.packets) != 2 {
		t.Fatalf("len(conn.packets) = %d, want 2", len(conn.packets))
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

func TestRouterHandleInteractMenu(t *testing.T) {
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
	if !resp.Accepted {
		t.Fatalf("resp.Accepted = false, want true")
	}
	if resp.ResponseType != "menu" {
		t.Fatalf("resp.ResponseType = %q, want menu", resp.ResponseType)
	}
	if resp.EntityID != 93001 {
		t.Fatalf("resp.EntityID = %d, want 93001", resp.EntityID)
	}
	if len(resp.MenuEntries) != 1 {
		t.Fatalf("len(resp.MenuEntries) = %d, want 1", len(resp.MenuEntries))
	}
	if resp.MenuEntries[0].EntryType != "dialog" {
		t.Fatalf("resp.MenuEntries[0].EntryType = %q, want dialog", resp.MenuEntries[0].EntryType)
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
	if err := playerService.UpdatePosition(context.Background(), 10001, 3, 12, 10); err != nil {
		t.Fatalf("UpdatePosition() error = %v", err)
	}

	mustHandleJSONPacket(t, router, conn, protocol.CmdInteractReq, 163, protocol.InteractReq{EntityID: 93001})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	var resp protocol.InteractResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if len(resp.MenuEntries) != 2 {
		t.Fatalf("len(resp.MenuEntries) = %d, want 2", len(resp.MenuEntries))
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
	if resp.ResultType != "notice" {
		t.Fatalf("resp.ResultType = %q, want notice", resp.ResultType)
	}
	if resp.EntityID != 93002 {
		t.Fatalf("resp.EntityID = %d, want 93002", resp.EntityID)
	}
	if len(resp.MenuEntries) != 2 {
		t.Fatalf("len(resp.MenuEntries) = %d, want 2", len(resp.MenuEntries))
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
	if len(start.Allies) != 2 || len(start.Enemies) != 1 {
		t.Fatalf("unexpected actor counts allies=%d enemies=%d", len(start.Allies), len(start.Enemies))
	}
	if start.ActiveActorID != start.Allies[0].ActorID {
		t.Fatalf("start.ActiveActorID = %d, want %d", start.ActiveActorID, start.Allies[0].ActorID)
	}
	if start.ActivePetUID != start.Allies[0].PetUID {
		t.Fatalf("start.ActivePetUID = %d, want %d", start.ActivePetUID, start.Allies[0].PetUID)
	}
	if start.Allies[0].LineupIndex != 0 {
		t.Fatalf("start.Allies[0].LineupIndex = %d, want 0", start.Allies[0].LineupIndex)
	}
	if len(start.Allies[0].SkillIDs) != 2 {
		t.Fatalf("len(start.Allies[0].SkillIDs) = %d, want 2", len(start.Allies[0].SkillIDs))
	}
	if len(start.Allies[0].Skills) != 2 {
		t.Fatalf("len(start.Allies[0].Skills) = %d, want 2", len(start.Allies[0].Skills))
	}
	if start.Allies[0].Skills[0].TargetType != "enemy_single" {
		t.Fatalf("start.Allies[0].Skills[0].TargetType = %q, want %q", start.Allies[0].Skills[0].TargetType, "enemy_single")
	}
	if len(start.Allies[1].Skills) != 2 {
		t.Fatalf("len(start.Allies[1].Skills) = %d, want 2", len(start.Allies[1].Skills))
	}
	if start.Allies[1].Skills[1].TargetType != "ally_single" {
		t.Fatalf("start.Allies[1].Skills[1].TargetType = %q, want %q", start.Allies[1].Skills[1].TargetType, "ally_single")
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
	if state.Round != 1 {
		t.Fatalf("state.Round = %d, want 1", state.Round)
	}
	if state.ActiveActorID != start.Allies[1].ActorID {
		t.Fatalf("state.ActiveActorID = %d, want %d", state.ActiveActorID, start.Allies[1].ActorID)
	}
	if state.ActivePetUID != start.Allies[1].PetUID {
		t.Fatalf("state.ActivePetUID = %d, want %d", state.ActivePetUID, start.Allies[1].PetUID)
	}
	if len(state.PendingActorIDs) != 1 || state.PendingActorIDs[0] != start.Allies[1].ActorID {
		t.Fatalf("unexpected pending actor ids: %#v", state.PendingActorIDs)
	}
	if len(state.Events) != 0 {
		t.Fatalf("len(state.Events) = %d, want 0 while still collecting commands", len(state.Events))
	}

	secondAction, err := protocol.NewJSONPacket(protocol.CmdBattleActionReq, 18, 0, protocol.BattleActionReq{
		OpID:       2,
		BattleID:   start.BattleID,
		Round:      state.Round,
		ActionType: battle.ActionTypeSkill,
		ActorID:    start.Allies[1].ActorID,
		SkillID:    start.Allies[1].SkillIDs[0],
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
	if len(conn.packets) != 9 {
		t.Fatalf("len(conn.packets) after second action = %d, want 9", len(conn.packets))
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
	if result.RewardGold == 0 {
		t.Fatal("result.RewardGold = 0, want positive battle gold reward")
	}
	if result.RewardPlayerExp == 0 {
		t.Fatal("result.RewardPlayerExp = 0, want positive player exp reward")
	}
	if result.PlayerGold <= 100 {
		t.Fatalf("result.PlayerGold = %d, want greater than initial 100", result.PlayerGold)
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

	var petUpdate protocol.PetUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[7].Body, &petUpdate); err != nil {
		t.Fatalf("UnmarshalBody(petUpdate) error = %v", err)
	}
	if petUpdate.Pet.PetUID != start.ActivePetUID {
		t.Fatalf("petUpdate.Pet.PetUID = %d, want %d", petUpdate.Pet.PetUID, start.ActivePetUID)
	}
	if petUpdate.Pet.Exp <= 120 {
		t.Fatalf("petUpdate.Pet.Exp = %d, want greater than starter exp 120", petUpdate.Pet.Exp)
	}
	allyHPAfterBattle := petUpdate.Pet.HP
	if conn.packets[8].Cmd != protocol.CmdPetUpdatePush {
		t.Fatalf("conn.packets[8].Cmd = %d, want %d", conn.packets[8].Cmd, protocol.CmdPetUpdatePush)
	}
	if conn.packets[5].Cmd != protocol.CmdBattleStatePush {
		t.Fatalf("conn.packets[5].Cmd = %d, want %d", conn.packets[5].Cmd, protocol.CmdBattleStatePush)
	}
	if allyHPAfterBattle == 0 {
		t.Fatalf("allyHPAfterBattle = 0, want non-zero")
	}

	petListPacket, err := protocol.NewJSONPacket(protocol.CmdPetListReq, 19, 0, protocol.PetListReq{})
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
	if len(conn.packets) != 10 {
		t.Fatalf("len(conn.packets) after pet list = %d, want 10", len(conn.packets))
	}

	var petList protocol.PetListResp
	if err := protocol.UnmarshalBody(conn.packets[9].Body, &petList); err != nil {
		t.Fatalf("UnmarshalBody(petList) error = %v", err)
	}
	if len(petList.Pets) == 0 {
		t.Fatalf("len(petList.Pets) = 0, want non-zero")
	}
	if petList.Pets[0].PetUID != start.ActivePetUID {
		t.Fatalf("petList.Pets[0].PetUID = %d, want %d", petList.Pets[0].PetUID, start.ActivePetUID)
	}
	if petList.Pets[0].HP != allyHPAfterBattle {
		t.Fatalf("petList.Pets[0].HP = %d, want %d", petList.Pets[0].HP, allyHPAfterBattle)
	}
	if petList.Pets[0].Exp <= 120 {
		t.Fatalf("petList.Pets[0].Exp = %d, want persisted reward exp above 120", petList.Pets[0].Exp)
	}
	if len(petList.Lineup) == 0 {
		t.Fatalf("len(petList.Lineup) = 0, want non-zero")
	}
	if petList.Lineup[0].HP != allyHPAfterBattle {
		t.Fatalf("petList.Lineup[0].HP = %d, want %d", petList.Lineup[0].HP, allyHPAfterBattle)
	}
}

func TestBattleCustodySweepAfterDisconnectPersistsResult(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := player.NewService(teststub.NewPlayerRepository())
	petService := pet.NewService(teststub.NewPetRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleService := battle.NewService()
	battleHandler := NewBattleHandler(sessionService, playerService, petService, worldService, questService, npcService, battleService, teststub.NewBattleRepository())
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

	start, err := battleService.StartPVE(ctx, profile, lineup, target)
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
		t.Fatalf("SubmitAction(after custody finish) error = %v, want ErrBattleNotFound", err)
	}

	pets, err := petService.ListPets(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("ListPets() error = %v", err)
	}
	if len(pets) < 2 {
		t.Fatalf("len(pets) = %d, want at least 2", len(pets))
	}
	var sawPersistedDamage bool
	for _, item := range pets {
		if item.HP < item.HPMax {
			sawPersistedDamage = true
			break
		}
	}
	if !sawPersistedDamage {
		t.Fatal("expected at least one pet hp change to be persisted after disconnected custody battle")
	}
}

func TestRouterHandleReconnectRestoresWorldAndBattleSnapshots(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := player.NewService(teststub.NewPlayerRepository())
	petService := pet.NewService(teststub.NewPetRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, worldService)
	petHandler := NewPetHandler(sessionService, petService)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleService := battle.NewService()
	battleHandler := NewBattleHandler(sessionService, playerService, petService, worldService, questService, npcService, battleService, teststub.NewBattleRepository())
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, battleHandler, NewQuestHandler(questService, sessionService), sessionService)

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
	start, err := battleService.StartPVE(ctx, profile, lineup, target)
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
	if len(payload.BattleState.PendingActorIDs) != 1 {
		t.Fatalf("len(payload.BattleState.PendingActorIDs) = %d, want 1", len(payload.BattleState.PendingActorIDs))
	}
	if len(payload.BattleReplayStates) != 1 {
		t.Fatalf("len(payload.BattleReplayStates) = %d, want 1", len(payload.BattleReplayStates))
	}
	if payload.BattleReplayStates[0].Frame <= start.BattleVersion {
		t.Fatalf("payload.BattleReplayStates[0].Frame = %d, want > %d", payload.BattleReplayStates[0].Frame, start.BattleVersion)
	}
	if payload.ReconnectToken == originalReconnectToken {
		t.Fatal("reconnect token was not rotated after reconnect")
	}
}

func TestRouterHandleReconnectReturnsBattleResultAfterCustodyFinish(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := player.NewService(teststub.NewPlayerRepository())
	petService := pet.NewService(teststub.NewPetRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, worldService)
	petHandler := NewPetHandler(sessionService, petService)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleService := battle.NewService()
	battleHandler := NewBattleHandler(sessionService, playerService, petService, worldService, questService, npcService, battleService, teststub.NewBattleRepository())
	sessionService.SetDisconnectHandler(battleHandler.HandleSessionDisconnect)
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, battleHandler, NewQuestHandler(questService, sessionService), sessionService)

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
	start, err := battleService.StartPVE(ctx, profile, lineup, target)
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
		t.Fatalf("payload battle snapshot = start:%#v state:%#v, want nil because custody already finished", payload.BattleStart, payload.BattleState)
	}
	if payload.BattleResult == nil {
		t.Fatal("payload.BattleResult = nil, want cached result for reconnect")
	}
	if payload.BattleResult.BattleID != start.BattleID {
		t.Fatalf("payload.BattleResult.BattleID = %d, want %d", payload.BattleResult.BattleID, start.BattleID)
	}
}

func TestRouterHandlePVPChallengeAcceptStartsBattleForBothPlayers(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	sessionService := session.NewService(logger, 10*time.Second, 30*time.Second)
	playerService := player.NewService(teststub.NewPlayerRepository())
	petService := pet.NewService(teststub.NewPetRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, worldService)
	petHandler := NewPetHandler(sessionService, petService)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleHandler := NewBattleHandler(sessionService, playerService, petService, worldService, questService, npcService, battle.NewService(), teststub.NewBattleRepository())
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, battleHandler, NewQuestHandler(questService, sessionService), sessionService)

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
	playerService := player.NewService(teststub.NewPlayerRepository())
	petService := pet.NewService(teststub.NewPetRepository())
	worldService := world.NewService(teststub.NewWorldRepository())
	questService := quest.NewService(teststub.NewQuestRepository())
	worldHandler := NewWorldHandler(sessionService, playerService, petService, questService, worldService)
	petHandler := NewPetHandler(sessionService, petService)
	npcService := npc.NewService(teststub.NewNPCRepository())
	battleHandler := NewBattleHandler(sessionService, playerService, petService, worldService, questService, npcService, battle.NewService(), teststub.NewBattleRepository())
	router := NewRouter(&AuthHandler{sessionService: sessionService}, worldHandler, petHandler, battleHandler, NewQuestHandler(questService, sessionService), sessionService)

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
