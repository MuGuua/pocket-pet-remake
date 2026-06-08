package wstransport

import (
	"context"
	"errors"
	"time"

	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type QuestHandler struct {
	questService   *quest.Service
	sessionService *session.Service
}

func NewQuestHandler(questService *quest.Service, sessionService *session.Service) *QuestHandler {
	return &QuestHandler{
		questService:   questService,
		sessionService: sessionService,
	}
}

func (h *QuestHandler) HandleQuestList(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestListReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest list body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	summaries, trackedQuestID, err := h.questService.List(context.Background(), playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest list failed")
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestListResp, packet.Seq, errcode.WSCodeSuccess, protocol.QuestListResp{
		Quests:         toProtocolQuestSummaries(summaries),
		TrackedQuestID: trackedQuestID,
		ServerTimeMS:   time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

func (h *QuestHandler) HandleQuestAccept(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestAcceptReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest accept body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	ctx := context.Background()
	before, err := listQuestSummaries(ctx, h.questService, playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest state failed")
	}
	summary, err := h.questService.Accept(ctx, playerID, request.QuestID, request.NPCID)
	if err != nil {
		return h.sendQuestDomainError(conn, packet.Seq, err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestAcceptResp, packet.Seq, errcode.WSCodeSuccess, protocol.QuestAcceptResp{
		Accepted: true,
		Reason:   "quest accepted",
		Quest:    toProtocolQuestSummary(summary),
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, before)
}

func (h *QuestHandler) HandleQuestSubmit(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestSubmitReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest submit body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	ctx := context.Background()
	before, err := listQuestSummaries(ctx, h.questService, playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest state failed")
	}
	summary, err := h.questService.Submit(ctx, playerID, request.QuestID, request.NPCID)
	if err != nil {
		return h.sendQuestDomainError(conn, packet.Seq, err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestSubmitResp, packet.Seq, errcode.WSCodeSuccess, protocol.QuestSubmitResp{
		Accepted: true,
		Reason:   "quest submitted",
		Quest:    toProtocolQuestSummary(summary),
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, before)
}

func (h *QuestHandler) HandleQuestTrack(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestTrackReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest track body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	ctx := context.Background()
	before, err := listQuestSummaries(ctx, h.questService, playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest state failed")
	}
	if err := h.questService.Track(ctx, playerID, request.QuestID); err != nil {
		return h.sendQuestDomainError(conn, packet.Seq, err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestTrackResp, packet.Seq, errcode.WSCodeSuccess, protocol.QuestTrackResp{
		Accepted: true,
		Reason:   "track updated",
		QuestID:  request.QuestID,
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, before)
}

func (h *QuestHandler) requirePlayerID(connID string) (uint64, error) {
	sess, err := h.sessionService.GetByConnID(connID)
	if err != nil {
		return 0, err
	}
	return sess.PlayerID, nil
}

func (h *QuestHandler) sendQuestDomainError(conn packetSender, seq uint32, err error) error {
	reason := "quest request failed"
	switch {
	case errors.Is(err, quest.ErrQuestTemplateNotFound):
		reason = "quest not found"
	case errors.Is(err, quest.ErrQuestLocked):
		reason = "quest locked"
	case errors.Is(err, quest.ErrQuestNotAvailable):
		reason = "quest not available"
	case errors.Is(err, quest.ErrQuestAcceptNPCMismatch):
		reason = "quest accept npc mismatch"
	case errors.Is(err, quest.ErrQuestSubmitNPCMismatch):
		reason = "quest submit npc mismatch"
	}
	packet, packetErr := protocol.NewJSONPacket(protocol.CmdErrorPush, seq, errcode.WSCodeSuccess, protocol.ErrorPush{
		Code: errcode.WSCodeSuccess,
		Msg:  reason,
	})
	if packetErr != nil {
		return packetErr
	}
	return conn.SendPacket(packet)
}
