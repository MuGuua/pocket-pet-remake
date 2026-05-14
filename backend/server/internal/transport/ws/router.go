package wstransport

import (
	"time"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type Router struct {
	authHandler    *AuthHandler
	worldHandler   *WorldHandler
	battleHandler  *BattleHandler
	sessionService *session.Service
}

func NewRouter(authHandler *AuthHandler, worldHandler *WorldHandler, battleHandler *BattleHandler, sessionService *session.Service) *Router {
	return &Router{
		authHandler:    authHandler,
		worldHandler:   worldHandler,
		battleHandler:  battleHandler,
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
		return conn.SendPacket(responsePacket)
	case protocol.CmdEnterWorldReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.worldHandler.HandleEnterWorld(conn, packet)
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
	case protocol.CmdBattleActionReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleBattleAction(conn, packet)
	default:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "unsupported command")
	}
}

func sendError(conn packetSender, seq uint32, code uint32, message string) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdErrorPush, seq, code, protocol.ErrorPush{
		Code: code,
		Msg:  message,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}
