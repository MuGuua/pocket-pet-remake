package wstransport

import (
	"context"
	"log"
	"time"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type Router struct {
	authHandler    *AuthHandler
	worldHandler   *WorldHandler
	petHandler     *PetHandler
	battleHandler  *BattleHandler
	questHandler   *QuestHandler
	sessionService *session.Service
}

func NewRouter(authHandler *AuthHandler, worldHandler *WorldHandler, petHandler *PetHandler, battleHandler *BattleHandler, questHandler *QuestHandler, sessionService *session.Service) *Router {
	return &Router{
		authHandler:    authHandler,
		worldHandler:   worldHandler,
		petHandler:     petHandler,
		battleHandler:  battleHandler,
		questHandler:   questHandler,
		sessionService: sessionService,
	}
}

func (r *Router) Handle(conn packetSender, raw []byte) error {
	packet, err := protocol.DecodePacket(raw)
	if err != nil {
		return sendError(conn, 0, errcode.WSCodeInvalidPacket, "invalid packet")
	}

	switch packet.Cmd {
	case protocol.CmdWSAuthReq:
		return r.authHandler.Handle(conn, packet)
	case protocol.CmdHeartbeatReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		_, err := r.sessionService.Touch(conn.ID())
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
		}
		responsePacket, err := protocol.NewJSONPacket(protocol.CmdHeartbeatResp, packet.Seq, errcode.WSCodeSuccess, protocol.HeartbeatResp{
			ServerTimeMS: time.Now().UnixMilli(),
		})
		if err != nil {
			return err
		}
		if err := conn.SendPacket(responsePacket); err != nil {
			return err
		}
		if r.battleHandler != nil {
			return r.battleHandler.HandleBattleHeartbeat(conn)
		}
		return nil
	case protocol.CmdEnterWorldReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.worldHandler.HandleEnterWorld(conn, packet)
	case protocol.CmdReconnectReq:
		if r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "connection already authenticated")
		}
		return r.handleReconnect(conn, packet)
	case protocol.CmdMoveIntentReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.worldHandler.HandleMoveIntent(conn, packet)
	case protocol.CmdInteractReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleInteract(conn, packet)
	case protocol.CmdNPCActionReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleNPCAction(conn, packet)
	case protocol.CmdPetListReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.petHandler.HandlePetList(conn, packet)
	case protocol.CmdPetLineupSetReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.petHandler.HandleLineupSet(conn, packet)
	case protocol.CmdBattleActionReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleBattleAction(conn, packet)
	case protocol.CmdPVPChallengeReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandlePVPChallenge(conn, packet)
	case protocol.CmdPVPChallengeReplyReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandlePVPChallengeReply(conn, packet)
	case protocol.CmdQuestListReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestList(conn, packet)
	case protocol.CmdQuestAcceptReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestAccept(conn, packet)
	case protocol.CmdQuestSubmitReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestSubmit(conn, packet)
	case protocol.CmdQuestTrackReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestTrack(conn, packet)
	default:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "unsupported command")
	}
}

func (r *Router) handleReconnect(conn packetSender, packet *protocol.Packet) error {
	var request protocol.ReconnectReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid reconnect body")
	}
	sess, err := r.sessionService.Reconnect(request.ReconnectToken, conn)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "reconnect token invalid")
	}

	var worldSnapshot *protocol.EnterWorldResp
	if r.worldHandler != nil {
		worldSnapshot, err = r.worldHandler.BuildWorldSnapshotForPlayer(context.Background(), sess.PlayerID)
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load reconnect world snapshot failed")
		}
	}
	var battleStart *protocol.BattleStartPush
	var battleState *protocol.BattleStatePush
	var battleResult *protocol.BattleResultPush
	var battleReplayStates []protocol.BattleStatePush
	if r.battleHandler != nil {
		battleStart, battleState, battleResult, battleReplayStates, err = r.battleHandler.BuildReconnectSnapshot(context.Background(), sess.PlayerID, request.BattleID, request.LastFrame)
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "load reconnect battle snapshot failed")
		}
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdReconnectResp, packet.Seq, errcode.WSCodeSuccess, protocol.ReconnectResp{
		PlayerID:           sess.PlayerID,
		SessionID:          sess.ID,
		ReconnectToken:     sess.ReconnectToken,
		HeartbeatSec:       uint32(r.sessionService.HeartbeatInterval() / time.Second),
		ServerTimeMS:       time.Now().UnixMilli(),
		World:              worldSnapshot,
		BattleStart:        battleStart,
		BattleState:        battleState,
		BattleResult:       battleResult,
		BattleReplayStates: battleReplayStates,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

func sendError(conn packetSender, seq uint32, code uint32, message string, causes ...error) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdErrorPush, seq, code, protocol.ErrorPush{
		Code: code,
		Msg:  message,
	})
	if err != nil {
		log.Printf("[ws-error] conn_id=%s seq=%d code=%d msg=%s packet_err=%v", conn.ID(), seq, code, message, err)
		return err
	}
	if len(causes) > 0 && causes[0] != nil {
		log.Printf("[ws-error] conn_id=%s seq=%d code=%d msg=%s cause=%v", conn.ID(), seq, code, message, causes[0])
	} else {
		log.Printf("[ws-error] conn_id=%s seq=%d code=%d msg=%s", conn.ID(), seq, code, message)
	}
	return conn.SendPacket(packet)
}
