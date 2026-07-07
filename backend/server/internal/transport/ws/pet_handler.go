package wstransport

import (
	"context"
	"errors"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/petprogression"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type PetHandler struct {
	sessionService *session.Service
	petService     *pet.Service
}

func NewPetHandler(sessionService *session.Service, petService *pet.Service, _ *petprogression.Service) *PetHandler {
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
	pets, err := h.petService.ListPetSummaries(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePetListFailed, "load pet list failed", err)
	}
	lineup, err := h.petService.ListLineupSummaries(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePetListFailed, "load pet lineup failed", err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPetListResp, packet.Seq, errcode.WSCodeSuccess, protocol.PetListResp{
		Pets:   toProtocolPetDetails(pets),
		Lineup: toProtocolLineup(lineup, h.petService.ResolveSkinID),
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
		if errors.Is(err, pet.ErrPetNotFound) || errors.Is(err, pet.ErrInvalidLineup) || errors.Is(err, pet.ErrDuplicateLineup) || errors.Is(err, pet.ErrPetUnusable) {
			return h.sendLineupSetResponse(conn, packet.Seq, false, nil, err.Error())
		}
		return sendError(conn, packet.Seq, errcode.WSCodePetLineupInvalid, "set pet lineup failed", err)
	}
	return h.sendLineupSetResponse(conn, packet.Seq, true, lineup, "lineup updated")
}

// HandleAllocateAttr 处理宠物主动分配自由属性点。
func (h *PetHandler) HandleAllocateAttr(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PetAllocateAttrReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pet allocate attr body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.petService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "pet service unavailable")
	}

	delta := petprogression.ManualAllocatedPoints{
		HP:   request.HP,
		ATK:  request.ATK,
		SPD:  request.SPD,
		MANA: request.MANA,
		DEF:  request.DEF,
	}
	updatedPet, err := h.petService.AllocateAttrPoints(context.Background(), sess.PlayerID, request.PetUID, delta)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrPetNotFound):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "pet not found")
		case errors.Is(err, petprogression.ErrInsufficientAttrPoints):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "insufficient attr points")
		case errors.Is(err, petprogression.ErrInvalidAllocateInput):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid allocate attr input")
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "allocate pet attr failed")
		}
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPetAllocateAttrResp, packet.Seq, errcode.WSCodeSuccess, protocol.PetAllocateAttrResp{
		Pet: toProtocolPetDetail(updatedPet),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleArtifactEquip 处理宠物法宝装备。
func (h *PetHandler) HandleArtifactEquip(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PetArtifactEquipReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pet artifact equip body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	updatedPet, err := h.petService.EquipArtifactFromBagSlot(
		context.Background(),
		sess.PlayerID,
		request.PetUID,
		request.SlotIndex,
		request.ContainerType,
		request.BagSlotIndex,
	)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrPetNotFound):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "pet not found")
		case errors.Is(err, pet.ErrInvalidArtifactSlot), errors.Is(err, pet.ErrInvalidArtifactItem):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid artifact equip request")
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "equip pet artifact failed")
		}
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPetArtifactEquipResp, packet.Seq, errcode.WSCodeSuccess, protocol.PetArtifactEquipResp{
		Pet: toProtocolPetDetail(updatedPet),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleArtifactUnequip 处理宠物法宝卸下。
func (h *PetHandler) HandleArtifactUnequip(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PetArtifactUnequipReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pet artifact unequip body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	updatedPet, err := h.petService.UnequipArtifact(context.Background(), sess.PlayerID, request.PetUID, request.SlotIndex)
	if err != nil {
		switch {
		case errors.Is(err, pet.ErrPetNotFound):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "pet not found")
		case errors.Is(err, pet.ErrInvalidArtifactSlot), errors.Is(err, pet.ErrArtifactSlotEmpty):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid artifact unequip request")
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "unequip pet artifact failed")
		}
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPetArtifactUnequipResp, packet.Seq, errcode.WSCodeSuccess, protocol.PetArtifactUnequipResp{
		Pet: toProtocolPetDetail(updatedPet),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleSkillDetail 返回单只宠物完整 skill_slots（含法宝技）。
func (h *PetHandler) HandleSkillDetail(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PetSkillDetailReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pet skill detail body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	item, err := h.petService.GetPetDetail(context.Background(), sess.PlayerID, request.PetUID)
	if err != nil {
		if errors.Is(err, pet.ErrPetNotFound) {
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "pet not found")
		}
		return sendError(conn, packet.Seq, errcode.WSCodePetListFailed, "load pet skill detail failed", err)
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPetSkillDetailResp, packet.Seq, errcode.WSCodeSuccess, protocol.PetSkillDetailResp{
		Pet: toProtocolPetDetail(item),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

func (h *PetHandler) sendLineupSetResponse(conn packetSender, seq uint32, accepted bool, lineup []pet.LineupPet, reason string) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdPetLineupSetResp, seq, errcode.WSCodeSuccess, protocol.PetLineupSetResp{
		Accepted: accepted,
		Lineup:   toProtocolLineup(lineup, h.petService.ResolveSkinID),
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
		result = append(result, toProtocolPetDetailForList(item))
	}
	return result
}
