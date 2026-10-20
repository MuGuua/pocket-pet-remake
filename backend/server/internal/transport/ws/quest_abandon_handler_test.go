package wstransport

import (
	"io"
	"log"
	"testing"
	"time"

	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
)

// buildQuestAbandonHandlerForTest 装配只含任务处理器的最小依赖，并注册两条测试任务：
// 9701 可放弃且已接取；9702 模板禁止放弃。
func buildQuestAbandonHandlerForTest(t *testing.T) (*QuestHandler, *fakeConn) {
	t.Helper()
	repo := teststub.NewQuestRepository()
	objectives := []quest.ObjectiveTemplate{{ObjectiveID: 1, EventType: "WIN_BATTLE", Description: "赢得 1 场战斗", TargetValue: 1}}
	repo.PutTemplate(quest.Template{QuestID: 9701, QuestType: "DAILY", Title: "可放弃日常", AcceptMode: "NPC", SubmitMode: "NPC", CanAbandon: true, Objectives: objectives})
	repo.PutTemplate(quest.Template{QuestID: 9702, QuestType: "SIDE", Title: "不可放弃支线", AcceptMode: "NPC", SubmitMode: "NPC", Objectives: objectives})

	service := quest.NewService(repo)
	sessionService := session.NewService(log.New(io.Discard, "", 0), 10*time.Second, 30*time.Second)
	conn := &fakeConn{id: "quest-abandon-conn"}
	if _, err := sessionService.Bind(teststub.DemoPlayerID, conn); err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	handler := NewQuestHandler(service, sessionService, nil, nil, nil, nil, nil)
	for _, questID := range []uint64{9701, 9702} {
		if _, err := service.Accept(t.Context(), teststub.DemoPlayerID, questID, 0); err != nil {
			t.Fatalf("Accept(%d) error = %v", questID, err)
		}
	}
	return handler, conn
}

func mustQuestAbandon(t *testing.T, handler *QuestHandler, conn *fakeConn, seq uint32, questID uint64) {
	t.Helper()
	packet, err := protocol.NewJSONPacket(protocol.CmdQuestAbandonReq, seq, 0, protocol.QuestAbandonReq{QuestID: questID})
	if err != nil {
		t.Fatalf("NewJSONPacket() error = %v", err)
	}
	if err := handler.HandleQuestAbandon(conn, packet); err != nil {
		t.Fatalf("HandleQuestAbandon() error = %v", err)
	}
}

// TestHandleQuestAbandonResetsQuest 验证放弃成功：回包状态为 AVAILABLE，并推送任务差量。
func TestHandleQuestAbandonResetsQuest(t *testing.T) {
	handler, conn := buildQuestAbandonHandlerForTest(t)
	clearPackets(conn)

	mustQuestAbandon(t, handler, conn, 501, 9701)
	if len(conn.packets) < 2 || conn.packets[0].Cmd != protocol.CmdQuestAbandonResp {
		t.Fatalf("packets = %d, first cmd = %d, want abandon resp followed by quest push", len(conn.packets), conn.packets[0].Cmd)
	}
	var resp protocol.QuestAbandonResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if !resp.Accepted || resp.Quest.State != quest.StateAvailable || resp.Quest.CanAbandon {
		t.Fatalf("resp = %#v, want accepted, AVAILABLE, not abandonable", resp)
	}
	if conn.packets[1].Cmd != protocol.CmdQuestUpdatePush {
		t.Fatalf("packets[1].Cmd = %d, want quest update push", conn.packets[1].Cmd)
	}
}

// TestHandleQuestAbandonRejectsDisallowedQuest 验证模板禁止放弃时返回明确原因，且任务保持已接取。
func TestHandleQuestAbandonRejectsDisallowedQuest(t *testing.T) {
	handler, conn := buildQuestAbandonHandlerForTest(t)
	clearPackets(conn)

	mustQuestAbandon(t, handler, conn, 502, 9702)
	if len(conn.packets) != 1 || conn.packets[0].Cmd != protocol.CmdErrorPush {
		t.Fatalf("packets = %d, want single error push", len(conn.packets))
	}
	var push protocol.ErrorPush
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &push); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if push.Msg != "quest cannot be abandoned" {
		t.Fatalf("msg = %q, want quest cannot be abandoned", push.Msg)
	}
}
