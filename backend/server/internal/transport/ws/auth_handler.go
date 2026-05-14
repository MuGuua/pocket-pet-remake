package wstransport

import (
	"context"
	"errors"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type AuthHandler struct {
	authService    *auth.Service
	sessionService *session.Service
}

func NewAuthHandler(authService *auth.Service, sessionService *session.Service) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		sessionService: sessionService,
	}
}

func (h *AuthHandler) Handle(conn packetSender, packet *protocol.Packet) error {
	if h.sessionService.IsAuthenticated(conn.ID()) {
		return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "connection already authenticated")
	}

	var request protocol.WsAuthReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid auth body")
	}

	principal, err := h.authService.ConsumeWSToken(context.Background(), request.WSToken, request.DeviceID)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidWSToken) {
			return sendError(conn, packet.Seq, errcode.WSCodeTokenInvalid, "invalid ws token")
		}
		return err
	}

	sess, err := h.sessionService.Bind(principal.PlayerID, conn)
	if err != nil {
		return err
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdWSAuthResp, packet.Seq, errcode.WSCodeSuccess, protocol.WsAuthResp{
		PlayerID:       principal.PlayerID,
		SessionID:      sess.ID,
		ReconnectToken: sess.ReconnectToken,
		HeartbeatSec:   uint32(h.sessionService.HeartbeatInterval() / time.Second),
		ServerTimeMS:   time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}
