package wstransport

import (
	"context"
	"errors"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type PetHandler struct {
	sessionService *session.Service
	petService     *pet.Service
}

func NewPetHandler(sessionService *session.Service, petService *pet.Service) *PetHandler {
	return &PetHandler{
		sessionService: sessionService,
		petService:     petService,
	}
}

func (h *PetHandler) HandlePetList(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PetListReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pet list body")
	}

	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	ctx := context.Background()
	pets, err := h.petService.ListPets(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePetListFailed, "load pet list failed")
	}
	lineup, err := h.petService.ListLineup(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePetListFailed, "load pet lineup failed")
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPetListResp, packet.Seq, errcode.WSCodeSuccess, protocol.PetListResp{
		Pets:   toProtocolPetDetails(pets),
		Lineup: toProtocolLineup(lineup),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

func (h *PetHandler) HandleLineupSet(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PetLineupSetReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pet lineup body")
	}

	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	lineup, err := h.petService.SetLineup(context.Background(), sess.PlayerID, request.PetUIDs)
	if err != nil {
		if errors.Is(err, pet.ErrPetNotFound) || errors.Is(err, pet.ErrInvalidLineup) || errors.Is(err, pet.ErrDuplicateLineup) {
			return h.sendLineupSetResponse(conn, packet.Seq, false, nil, err.Error())
		}
		return sendError(conn, packet.Seq, errcode.WSCodePetLineupInvalid, "set pet lineup failed")
	}
	return h.sendLineupSetResponse(conn, packet.Seq, true, lineup, "lineup updated")
}

func (h *PetHandler) sendLineupSetResponse(conn packetSender, seq uint32, accepted bool, lineup []pet.LineupPet, reason string) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdPetLineupSetResp, seq, errcode.WSCodeSuccess, protocol.PetLineupSetResp{
		Accepted: accepted,
		Lineup:   toProtocolLineup(lineup),
		Reason:   reason,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func toProtocolPetDetails(pets []pet.Pet) []protocol.PetDetail {
	if len(pets) == 0 {
		return []protocol.PetDetail{}
	}
	result := make([]protocol.PetDetail, 0, len(pets))
	for _, item := range pets {
		result = append(result, toProtocolPetDetail(item))
	}
	return result
}
