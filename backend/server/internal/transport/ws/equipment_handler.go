package wstransport

import (
	"context"
	"errors"
	"log"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

// EquipmentHandler 处理人物装备佩戴相关的 WebSocket 请求。
type EquipmentHandler struct {
	sessionService   *session.Service
	equipmentService *equipment.Service
}

// NewEquipmentHandler 构造人物装备处理器。
func NewEquipmentHandler(sessionService *session.Service, equipmentService *equipment.Service) *EquipmentHandler {
	return &EquipmentHandler{
		sessionService:   sessionService,
		equipmentService: equipmentService,
	}
}

// HandleList 拉取玩家当前已佩戴装备列表。
func (h *EquipmentHandler) HandleList(conn packetSender, packet *protocol.Packet) error {
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.equipmentService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "equipment service unavailable")
	}
	items, err := h.equipmentService.ListEquipped(context.Background(), sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "list player equipment failed")
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPlayerEquipmentListResp, packet.Seq, errcode.WSCodeSuccess, protocol.PlayerEquipmentListResp{
		Items: toProtocolEquippedItems(items),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleEquip 从背包佩戴装备。
func (h *EquipmentHandler) HandleEquip(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PlayerEquipReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid player equip body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.equipmentService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "equipment service unavailable")
	}
	result, profile, err := h.equipmentService.EquipFromBagSlot(context.Background(), sess.PlayerID, request.ContainerType, request.BagSlotIndex)
	if err != nil {
		return mapEquipmentError(conn, packet.Seq, err)
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPlayerEquipResp, packet.Seq, errcode.WSCodeSuccess, protocol.PlayerEquipResp{
		Equipped:    toProtocolEquippedItem(result.EquippedSlot),
		Unequipped:  toOptionalProtocolEquippedItem(result.Unequipped),
		AllEquipped: toProtocolEquippedItems(result.AllEquipped),
		Player:      toProtocolPlayerSnapshot(profile),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleUnequip 卸下指定槽位装备。
func (h *EquipmentHandler) HandleUnequip(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PlayerUnequipReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid player unequip body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.equipmentService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "equipment service unavailable")
	}
	result, profile, err := h.equipmentService.UnequipSlot(context.Background(), sess.PlayerID, request.EquipSlot, request.ContainerType)
	if err != nil {
		return mapEquipmentError(conn, packet.Seq, err)
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPlayerUnequipResp, packet.Seq, errcode.WSCodeSuccess, protocol.PlayerUnequipResp{
		Unequipped:  toProtocolEquippedItem(result.Unequipped),
		AllEquipped: toProtocolEquippedItems(result.AllEquipped),
		Player:      toProtocolPlayerSnapshot(profile),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleEnhance 消耗材料并尝试强化指定装备实例。
func (h *EquipmentHandler) HandleEnhance(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PlayerEquipmentEnhanceReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid player equipment enhance body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.equipmentService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "equipment service unavailable")
	}
	result, err := h.equipmentService.EnhanceInstance(context.Background(), sess.PlayerID, request.ItemUID)
	if err != nil {
		return mapEquipmentEnhanceError(conn, packet.Seq, err)
	}
	response := protocol.PlayerEquipmentEnhanceResp{
		Success:     result.Success,
		OldLevel:    result.OldLevel,
		NewLevel:    result.NewLevel,
		RatePct:     result.RatePct,
		RollPct:     result.RollPct,
		Item:        toProtocolEquippedItem(result.Item),
		AllEquipped: toProtocolEquippedItems(result.AllEquipped),
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPlayerEquipmentEnhanceResp, packet.Seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

func mapEquipmentEnhanceError(conn packetSender, seq uint32, err error) error {
	switch {
	case errors.Is(err, equipment.ErrEquipmentNotFound):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "equipment not found")
	case errors.Is(err, equipment.ErrEquipmentEnhanceEquipped):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "equipment must be unequipped to enhance")
	case errors.Is(err, equipment.ErrEquipmentEnhanceNotAllowed):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "equipment cannot be enhanced")
	case errors.Is(err, equipment.ErrEquipmentEnhanceMaxLevel):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "equipment enhance level max")
	case errors.Is(err, equipment.ErrEquipmentEnhanceMaterialInsufficient):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "insufficient enhance materials")
	case errors.Is(err, equipment.ErrEquipmentEnhanceConfigMissing):
		return sendError(conn, seq, errcode.WSCodeInteractFailed, "enhance config missing")
	case errors.Is(err, player.ErrPlayerNotFound):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "player not found")
	default:
		return sendError(conn, seq, errcode.WSCodeInteractFailed, "equipment enhance failed")
	}
}

func mapEquipmentError(conn packetSender, seq uint32, err error) error {
	switch {
	case errors.Is(err, bag.ErrContainerItemNotFound):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "bag item not found")
	case errors.Is(err, equipment.ErrEquipmentBagItemInvalid):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "invalid equipment item")
	case errors.Is(err, equipment.ErrEquipmentLevelTooLow):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "player level too low")
	case errors.Is(err, equipment.ErrEquipmentSlotMismatch):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "equipment slot mismatch")
	case errors.Is(err, equipment.ErrEquipmentSlotEmpty):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "equipment slot empty")
	case errors.Is(err, equipment.ErrEquipmentBagFull):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "bag full")
	case errors.Is(err, player.ErrPlayerNotFound):
		return sendError(conn, seq, errcode.WSCodeInvalidPacket, "player not found")
	default:
		// 兜底分支说明错误没有命中任何已知业务校验；这里必须把真实错误打印出来，
		// 方便排查数据库迁移缺失、事务写入失败或属性重算等服务端权威链路问题。
		log.Printf("[equipment-error] seq=%d err=%v", seq, err)
		return sendError(conn, seq, errcode.WSCodeInteractFailed, "equipment operation failed")
	}
}

func toProtocolEquippedItems(items []equipment.RuntimeEquippedItem) []protocol.PlayerEquippedItemSnapshot {
	if len(items) == 0 {
		return []protocol.PlayerEquippedItemSnapshot{}
	}
	result := make([]protocol.PlayerEquippedItemSnapshot, 0, len(items))
	for _, item := range items {
		result = append(result, toProtocolEquippedItem(item))
	}
	return result
}

func toOptionalProtocolEquippedItem(item *equipment.RuntimeEquippedItem) *protocol.PlayerEquippedItemSnapshot {
	if item == nil {
		return nil
	}
	snapshot := toProtocolEquippedItem(*item)
	return &snapshot
}

func toProtocolEquippedItem(item equipment.RuntimeEquippedItem) protocol.PlayerEquippedItemSnapshot {
	return protocol.PlayerEquippedItemSnapshot{
		EquipSlot:        item.EquipSlot,
		EquipSlotLabel:   item.EquipSlotLabel,
		ItemUID:          item.ItemUID,
		ItemID:           item.ItemID,
		ItemName:         item.ItemName,
		EnhanceLevel:     item.EnhanceLevel,
		AppearanceSkinID: item.AppearanceSkinID,
		AppearanceOnly:   item.AppearanceOnly,
		Bonus: protocol.PlayerEquipmentBonusSnapshot{
			HPMax:                    item.Bonus.HPMax,
			MANA:                     item.Bonus.MANA,
			ATK:                      item.Bonus.ATK,
			DEF:                      item.Bonus.DEF,
			SPD:                      item.Bonus.SPD,
			Spirit:                   item.Bonus.Spirit,
			SpiritMax:                item.Bonus.SpiritMax,
			HitPct:                   item.Bonus.HitPct,
			DodgePct:                 item.Bonus.DodgePct,
			CritRatePct:              item.Bonus.CritRatePct,
			CritDmgPct:               item.Bonus.CritDmgPct,
			PhysicalResistPct:        item.Bonus.PhysicalResistPct,
			ReversePhysicalResistPct: item.Bonus.ReversePhysicalResistPct,
			SkillResistPct:           item.Bonus.SkillResistPct,
			ReverseSkillResistPct:    item.Bonus.ReverseSkillResistPct,
			ConfusionResistPct:       item.Bonus.ConfusionResistPct,
			SleepResistPct:           item.Bonus.SleepResistPct,
			ParalysisResistPct:       item.Bonus.ParalysisResistPct,
			SealResistPct:            item.Bonus.SealResistPct,
			CurseResistPct:           item.Bonus.CurseResistPct,
			CritDmgResistPct:         item.Bonus.CritDmgResistPct,
			CritResistPct:            item.Bonus.CritResistPct,
			CharacterResistPct:       item.Bonus.CharacterResistPct,
			PetResistPct:             item.Bonus.PetResistPct,
		},
	}
}
