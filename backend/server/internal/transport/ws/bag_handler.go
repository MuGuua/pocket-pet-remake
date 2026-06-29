package wstransport

import (
	"context"
	"errors"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

const (
	defaultBagListPage     uint32 = 1
	defaultBagListPageSize uint32 = 28
	bagListCategoryAll             string = "all"
	bagListCategoryEquip           string = "equipment"
	bagListCategoryOther           string = "other"
	bagListCategoryEnhanceMaterial string = "enhance_material"
)

// BagHandler 负责玩家端背包、仓库与钱包查询协议。
// 当前先打通查询链路，后续容器移动、扩容和使用道具都可以继续挂在这里扩展。
type BagHandler struct {
	sessionService   *session.Service
	bagService       *bag.Service
	itemService      *item.Service
	walletService    *wallet.Service
	playerService    *player.Service
	equipmentService *equipment.Service
	worldService     *world.Service
	npcService       *npc.Service
	rewardService    *reward.Service
	petService       *pet.Service
}

// NewBagHandler 构造运行时背包处理器。
func NewBagHandler(sessionService *session.Service, bagService *bag.Service, itemService *item.Service, walletService *wallet.Service, playerService *player.Service, petService *pet.Service, equipmentService *equipment.Service, worldService *world.Service, npcService *npc.Service) *BagHandler {
	return &BagHandler{
		sessionService:   sessionService,
		bagService:       bagService,
		itemService:      itemService,
		walletService:    walletService,
		playerService:    playerService,
		petService:       petService,
		equipmentService: equipmentService,
		worldService:     worldService,
		npcService:       npcService,
		rewardService:    reward.NewService(bagService, petService, playerService, nil, walletService),
	}
}

// HandleBagList 返回玩家随身背包与钱包快照。
// BAG_LIST_REQ 兼容省略 container_type 的旧写法，默认回包 bag 容器。
func (h *BagHandler) HandleBagList(conn packetSender, packet *protocol.Packet) error {
	var request protocol.BagListReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid bag list body")
	}
	return h.sendContainerList(conn, packet.Seq, request, protocol.CmdBagListResp)
}

// HandleContainerList 返回玩家指定容器与钱包快照。
func (h *BagHandler) HandleContainerList(conn packetSender, packet *protocol.Packet) error {
	var request protocol.ContainerListReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid container list body")
	}
	return h.sendContainerList(
		conn,
		packet.Seq,
		protocol.BagListReq{
			ContainerType: request.ContainerType,
		},
		protocol.CmdContainerListResp,
	)
}

// HandleWalletQuery 单独返回玩家钱包快照，供后续商城、任务奖励面板等复用。
func (h *BagHandler) HandleWalletQuery(conn packetSender, packet *protocol.Packet) error {
	var request protocol.WalletQueryReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid wallet query body")
	}

	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	walletSnapshot, err := h.walletService.GetRuntimeWallet(context.Background(), sess.PlayerID)
	if err != nil {
		if errors.Is(err, wallet.ErrWalletNotFound) {
			return sendError(conn, packet.Seq, errcode.WSCodeWalletQueryFailed, "wallet not found", err)
		}
		return sendError(conn, packet.Seq, errcode.WSCodeWalletQueryFailed, "load wallet snapshot failed", err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdWalletQueryResp, packet.Seq, errcode.WSCodeSuccess, protocol.WalletQueryResp{
		Wallet: toProtocolWalletSnapshot(*walletSnapshot),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleBuyItem executes the minimal server-authoritative shop purchase flow.
// The current MVP treats `shop_id` as the nearby shop NPC entity ID so the
// server can reuse the same world proximity validation chain as warehouse.
func (h *BagHandler) HandleBuyItem(conn packetSender, packet *protocol.Packet) error {
	var request protocol.BuyItemReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid buy item body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if request.ShopID == 0 || request.ItemID == 0 || request.Quantity == 0 {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "invalid buy item request")
	}
	if request.GoodsID != 0 && request.GoodsID != request.ItemID {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "shop goods mismatch")
	}
	if err := h.ensureShopAccess(context.Background(), sess.PlayerID, request.ShopID); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "shop access denied", err)
	}
	if h.npcService != nil {
		allowed, shopErr := h.npcService.ShopGoodExists(context.Background(), request.ShopID, request.ItemID)
		if shopErr != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "shop goods lookup failed", shopErr)
		}
		if !allowed {
			return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "shop goods unavailable")
		}
	}
	if h.itemService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopBuyFailed, "item service unavailable")
	}
	itemDetail, err := h.itemService.GetRuntimeItemDetail(context.Background(), request.ItemID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "item definition not found", err)
	}
	if !itemDetail.IsEnabled || itemDetail.BuyPriceCopper == 0 || !strings.EqualFold(itemDetail.PriceType, "base_coin") {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "item cannot be bought")
	}
	totalCost := itemDetail.BuyPriceCopper * request.Quantity
	walletSnapshot, err := h.walletService.GetRuntimeWallet(context.Background(), sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWalletQueryFailed, "wallet not found", err)
	}
	if walletSnapshot.TotalCopper < totalCost {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "insufficient wallet balance")
	}
	grantedItem, err := h.bagService.GrantRuntimeItem(context.Background(), sess.PlayerID, bag.ContainerTypeBag, request.ItemID, request.Quantity, "shop_buy", request.ShopID, "player", sess.PlayerID)
	if err != nil {
		switch {
		case errors.Is(err, bag.ErrContainerCapacityFull):
			return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "bag capacity full", err)
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeShopBuyFailed, "grant purchased item failed", err)
		}
	}
	adjustResult, err := h.walletService.AdjustRuntimeWallet(context.Background(), sess.PlayerID, wallet.RuntimeAdjustInput{
		ChangeTotalCopper: -int64(totalCost),
		ReasonType:        "shop_buy",
		ReasonRefID:       request.ShopID,
		OperatorType:      "player",
		OperatorID:        sess.PlayerID,
	})
	if err != nil {
		_, _ = h.bagService.ConsumeRuntimeItemStack(context.Background(), sess.PlayerID, bag.ContainerTypeBag, grantedItem.SlotIndex, grantedItem.GrantedQty, "shop_buy_rollback", request.ShopID)
		if snapshot, snapshotErr := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, bag.ContainerTypeBag); snapshotErr == nil && snapshot != nil {
			_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
		}
		switch {
		case errors.Is(err, wallet.ErrInvalidRuntimeAdjustInput):
			return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "insufficient wallet balance", err)
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeShopBuyFailed, "deduct wallet failed", err)
		}
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdBuyItemResp, packet.Seq, errcode.WSCodeSuccess, protocol.BuyItemResp{
		ShopID:   request.ShopID,
		GoodsID:  normalizeShopGoodsID(request),
		ItemID:   request.ItemID,
		Quantity: request.Quantity,
		Cost:     toProtocolCurrencyCost("base_coin", totalCost),
		Wallet:   toProtocolWalletSnapshot(adjustResult.Wallet),
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	if snapshot, snapshotErr := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, bag.ContainerTypeBag); snapshotErr == nil && snapshot != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
	}
	_ = pushWalletUpdatePacket(conn, adjustResult.Wallet, "shop_buy", request.ShopID)
	return nil
}

// HandleUseItem 执行背包物品主动使用。
// 当前服务端先接入扩容类功能道具，其他效果仍然要继续通过这里扩展，避免客户端绕过权威链路。
func (h *BagHandler) HandleUseItem(conn packetSender, packet *protocol.Packet) error {
	var request protocol.UseItemReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid use item body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	result, err := h.bagService.UseRuntimeItem(context.Background(), sess.PlayerID, request.ContainerType, request.SlotIndex, request.Quantity, request.TargetPetUID, request.TargetPlayerID, request.TargetItemUID)
	if err != nil {
		switch {
		case errors.Is(err, bag.ErrInvalidContainerType), errors.Is(err, bag.ErrInvalidTransferQuantity):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "invalid use item request", err)
		case errors.Is(err, bag.ErrContainerItemNotFound):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "item slot is empty", err)
		case errors.Is(err, bag.ErrItemNotUsable), errors.Is(err, bag.ErrUnsupportedItemEffect):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "item cannot be used", err)
		case errors.Is(err, bag.ErrUseTargetRequired), errors.Is(err, bag.ErrUseTargetNotFound):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "item target invalid", err)
		case errors.Is(err, bag.ErrItemUseNoEffect):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "item has no effect on current target", err)
		case errors.Is(err, bag.ErrContainerCapacityLimit):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "target container capacity already reached limit", err)
		case errors.Is(err, bag.ErrUseItemTooFast):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "use item too fast", err)
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeBagListFailed, "use item failed", err)
		}
	}
	var (
		walletSnapshot *wallet.Snapshot
		grantedPets    []pet.Pet
	)
	if h.rewardService != nil && len(result.Result.Rewards) > 0 {
		grantResult, grantErr := h.rewardService.GrantRuntimeRewards(context.Background(), reward.GrantInput{
			PlayerID:     sess.PlayerID,
			ReasonType:   "item_use_reward",
			ReasonRefID:  result.ItemID,
			OperatorType: "player",
			OperatorID:   sess.PlayerID,
			Rewards:      toRewardEntriesFromBagUse(result.Result.Rewards),
		})
		if grantErr != nil {
			return h.handleUseItemRewardGrantFailure(conn, packet.Seq, sess.PlayerID, result, grantErr)
		}
		result.Result.Rewards = toBagRuntimeRewards(grantResult.Granted)
		walletSnapshot = grantResult.Wallet
		grantedPets = append([]pet.Pet{}, grantResult.GrantedPets...)
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdUseItemResp, packet.Seq, errcode.WSCodeSuccess, protocol.UseItemResp{
		ContainerType: result.ContainerType,
		SlotIndex:     result.SlotIndex,
		ItemID:        result.ItemID,
		UsedQuantity:  result.UsedQuantity,
		Result: protocol.UseItemResult{
			EffectType:           result.Result.EffectType,
			ExpandTarget:         result.Result.ExpandTarget,
			ExpandSlots:          result.Result.ExpandSlots,
			NewCapacity:          result.Result.NewCapacity,
			TargetPetUID:         result.Result.TargetPetUID,
			RestoredHP:           result.Result.RestoredHP,
			NewPetHP:             result.Result.NewPetHP,
			UnlockedTalismanSlot: result.Result.UnlockedTalismanSlot,
			Rewards:              toProtocolUseItemRewards(result.Result.Rewards),
			AppliedEffects:       toProtocolUseItemAppliedEffects(result.Result.AppliedEffects),
			NeedsWalletPush:      result.Result.NeedsWalletPush,
		},
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	if snapshot, snapshotErr := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, result.ContainerType); snapshotErr == nil && snapshot != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
	}
	targetContainerType := strings.TrimSpace(result.Result.ExpandTarget)
	if targetContainerType != "" && targetContainerType != result.ContainerType {
		if snapshot, snapshotErr := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, targetContainerType); snapshotErr == nil && snapshot != nil {
			_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
		}
	}
	if result.Result.UpdatedPet != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
			Pet: toProtocolPetDetailFromBagSnapshot(*result.Result.UpdatedPet),
		}))
	}
	if result.Result.UnlockedTalismanSlot != "" && h.petService != nil && result.Result.TargetPetUID > 0 {
		if pets, listErr := h.petService.ListPets(context.Background(), sess.PlayerID); listErr == nil {
			for _, currentPet := range pets {
				if currentPet.PetUID != result.Result.TargetPetUID {
					continue
				}
				_ = conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
					Pet: toProtocolPetDetail(currentPet),
				}))
				break
			}
		}
	}
	for _, grantedPet := range grantedPets {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
			Pet: toProtocolPetDetail(grantedPet),
		}))
	}
	if walletSnapshot != nil {
		_ = pushWalletUpdatePacket(conn, *walletSnapshot, "item_use_reward", result.ItemID)
	} else if result.Result.NeedsWalletPush {
		if runtimeWallet, walletErr := h.walletService.GetRuntimeWallet(context.Background(), sess.PlayerID); walletErr == nil && runtimeWallet != nil {
			_ = pushWalletUpdatePacket(conn, *runtimeWallet, "item_use", result.ItemID)
		}
	}
	return nil
}

// HandleDropItem 执行背包物品主动丢弃，并在成功后推送最新容器快照。
func (h *BagHandler) HandleDropItem(conn packetSender, packet *protocol.Packet) error {
	var request protocol.DropItemReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid drop item body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	logBagEquipPacket("req", conn.ID(), sess.PlayerID, protocol.CmdDropItemReq, packet.Seq, request)
	result, err := h.bagService.DropRuntimeItem(context.Background(), sess.PlayerID, request.ContainerType, request.SlotIndex, request.ItemUID, request.Quantity)
	if err != nil {
		logBagEquipPacket("error", conn.ID(), sess.PlayerID, protocol.CmdDropItemReq, packet.Seq, map[string]any{
			"request": request,
			"error":   err.Error(),
		})
		switch {
		case errors.Is(err, bag.ErrInvalidContainerType), errors.Is(err, bag.ErrInvalidTransferQuantity):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "invalid drop item request", err)
		case errors.Is(err, bag.ErrContainerItemNotFound):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "item slot is empty", err)
		case errors.Is(err, bag.ErrItemNotDroppable):
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "item cannot be dropped", err)
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeBagListFailed, "drop item failed", err)
		}
	}
	response := protocol.DropItemResp{
		ContainerType: result.ContainerType,
		SlotIndex:     result.SlotIndex,
		ItemUID:       result.ItemUID,
		ItemID:        result.ItemID,
		ItemName:      result.ItemName,
		DroppedQty:    result.DroppedQty,
	}
	logBagEquipPacket("resp", conn.ID(), sess.PlayerID, protocol.CmdDropItemResp, packet.Seq, response)
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdDropItemResp, packet.Seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	if snapshot, snapshotErr := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, result.ContainerType); snapshotErr == nil && snapshot != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
	}
	return nil
}

// handleUseItemRewardGrantFailure compensates the already-consumed source item
// when a later reward grant step fails, so gift box usage does not silently eat
// the box and leave the player with only a partial result.
func (h *BagHandler) handleUseItemRewardGrantFailure(conn packetSender, seq uint32, playerID uint64, useResult *bag.RuntimeUseResult, grantErr error) error {
	if h.bagService != nil && useResult != nil && useResult.ItemID != 0 && useResult.UsedQuantity > 0 {
		_, _ = h.bagService.GrantRuntimeItem(
			context.Background(),
			playerID,
			useResult.ContainerType,
			useResult.ItemID,
			useResult.UsedQuantity,
			"item_use_rollback",
			useResult.ItemID,
			"system",
			0,
		)
		if snapshot, snapshotErr := h.bagService.ListRuntimeContainer(context.Background(), playerID, useResult.ContainerType); snapshotErr == nil && snapshot != nil {
			_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
		}
	}
	return sendError(conn, seq, errcode.WSCodeBagListFailed, "grant use item rewards failed", grantErr)
}

func toRewardEntriesFromBagUse(values []bag.RuntimeRewardItem) []reward.Entry {
	result := make([]reward.Entry, 0, len(values))
	for _, value := range values {
		result = append(result, reward.Entry{
			Type:     value.Type,
			Value:    value.Value,
			ItemID:   value.ItemID,
			ItemName: value.ItemName,
			Count:    value.Count,
			PetID:    value.PetID,
		})
	}
	return result
}

func toBagRuntimeRewards(values []reward.Entry) []bag.RuntimeRewardItem {
	result := make([]bag.RuntimeRewardItem, 0, len(values))
	for _, value := range values {
		result = append(result, bag.RuntimeRewardItem{
			Type:     value.Type,
			Value:    value.Value,
			ItemID:   value.ItemID,
			ItemName: value.ItemName,
			Count:    value.Count,
			PetID:    value.PetID,
		})
	}
	return result
}

func toProtocolUseItemRewards(values []bag.RuntimeRewardItem) []protocol.QuestReward {
	result := make([]protocol.QuestReward, 0, len(values))
	for _, value := range values {
		result = append(result, protocol.QuestReward{
			Type:     value.Type,
			Value:    value.Value,
			ItemID:   value.ItemID,
			ItemName: value.ItemName,
			Count:    value.Count,
			PetID:    value.PetID,
		})
	}
	return result
}

func toProtocolUseItemAppliedEffects(values []bag.RuntimeAppliedUseEffect) []protocol.UseItemAppliedEffect {
	result := make([]protocol.UseItemAppliedEffect, 0, len(values))
	for _, value := range values {
		result = append(result, protocol.UseItemAppliedEffect{
			Category:  value.Category,
			FieldKey:  value.FieldKey,
			Operation: value.Operation,
			Value:     value.Value,
			BoolValue: value.BoolValue,
		})
	}
	return result
}

func toProtocolPetDetailFromBagSnapshot(item bag.RuntimePetSnapshot) protocol.PetDetail {
	skills := make([]uint32, 0, len(item.SkillIDs))
	skills = append(skills, item.SkillIDs...)
	return protocol.PetDetail{
		PetUID:   item.PetUID,
		PetID:    item.PetID,
		Level:    item.Level,
		Exp:      item.Exp,
		Quality:  item.Quality,
		HP:       item.HP,
		HPMax:    item.HPMax,
		ATK:      item.ATK,
		DEF:      item.DEF,
		SPD:      item.SPD,
		SkillIDs: skills,
		InLineup: item.InLineup,
	}
}

func toProtocolCurrencyCost(currencyType string, totalCopper uint64) protocol.CurrencyCost {
	return protocol.CurrencyCost{
		CurrencyType: currencyType,
		TotalCopper:  totalCopper,
		Gold:         totalCopper / wallet.CopperPerGold,
		Silver:       (totalCopper % wallet.CopperPerGold) / wallet.CopperPerSilver,
		Copper:       totalCopper % wallet.CopperPerSilver,
	}
}

func normalizeShopGoodsID(request protocol.BuyItemReq) uint64 {
	if request.GoodsID != 0 {
		return request.GoodsID
	}
	return request.ItemID
}

// HandleBagToWarehouse 在校验仓库 NPC 可交互的前提下，把背包格子物品转移到仓库。
func (h *BagHandler) HandleBagToWarehouse(conn packetSender, packet *protocol.Packet) error {
	var request protocol.BagToWarehouseReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid bag to warehouse body")
	}
	return h.handleContainerTransfer(conn, packet.Seq, request.EntityID, bag.ContainerTypeBag, bag.ContainerTypeWarehouse, request.FromSlotIndex, request.Quantity, protocol.CmdBagToWarehouseResp)
}

// HandleWarehouseToBag 在校验仓库 NPC 可交互的前提下，把仓库格子物品取回背包。
func (h *BagHandler) HandleWarehouseToBag(conn packetSender, packet *protocol.Packet) error {
	var request protocol.WarehouseToBagReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid warehouse to bag body")
	}
	return h.handleContainerTransfer(conn, packet.Seq, request.EntityID, bag.ContainerTypeWarehouse, bag.ContainerTypeBag, request.FromSlotIndex, request.Quantity, protocol.CmdWarehouseToBagResp)
}

// HandleContainerSort 按服务端规则整理指定容器。
func (h *BagHandler) HandleContainerSort(conn packetSender, packet *protocol.Packet) error {
	var request protocol.ContainerSortReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid container sort body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	result, err := h.bagService.SortRuntimeContainer(context.Background(), sess.PlayerID, request.ContainerType)
	if err != nil {
		if errors.Is(err, bag.ErrInvalidContainerType) || errors.Is(err, bag.ErrContainerNotFound) {
			return sendError(conn, packet.Seq, errcode.WSCodeBagRequestInvalid, "invalid container sort request", err)
		}
		return sendError(conn, packet.Seq, errcode.WSCodeBagListFailed, "sort container failed", err)
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdContainerSortResp, packet.Seq, errcode.WSCodeSuccess, protocol.ContainerSortResp{
		ContainerType: result.ContainerType,
		Sorted:        result.Sorted,
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	snapshot, err := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, result.ContainerType)
	if err == nil && snapshot != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
	}
	return nil
}

// HandleContainerMove 执行同容器换位或拆分移动。
func (h *BagHandler) HandleContainerMove(conn packetSender, packet *protocol.Packet) error {
	var request protocol.ContainerMoveReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid container move body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	result, err := h.bagService.MoveRuntimeItem(context.Background(), sess.PlayerID, request.ContainerType, request.FromSlotIndex, request.ToSlotIndex, request.Quantity)
	if err != nil {
		switch {
		case errors.Is(err, bag.ErrInvalidContainerType), errors.Is(err, bag.ErrInvalidContainerMove):
			return h.sendContainerMoveResponse(conn, packet.Seq, false, "invalid move request", request)
		case errors.Is(err, bag.ErrContainerNotFound):
			return h.sendContainerMoveResponse(conn, packet.Seq, false, "container not found", request)
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeBagListFailed, "move container item failed", err)
		}
	}
	if err := h.sendContainerMoveResult(conn, packet.Seq, *result, "success"); err != nil {
		return err
	}
	snapshot, err := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, result.ContainerType)
	if err == nil && snapshot != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*snapshot)))
	}
	return nil
}

func (h *BagHandler) sendContainerList(conn packetSender, seq uint32, request protocol.BagListReq, responseCmd uint16) error {
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	ctx := context.Background()
	containerSnapshot, err := h.bagService.ListRuntimeContainer(ctx, sess.PlayerID, request.ContainerType)
	if err != nil {
		if errors.Is(err, bag.ErrInvalidContainerType) {
			return sendError(conn, seq, errcode.WSCodeBagRequestInvalid, "invalid container type", err)
		}
		if errors.Is(err, bag.ErrContainerNotFound) {
			return sendError(conn, seq, errcode.WSCodeBagListFailed, "container not found", err)
		}
		return sendError(conn, seq, errcode.WSCodeBagListFailed, "load container snapshot failed", err)
	}
	paginatedSnapshot, page, pageSize, category, totalItems := buildPaginatedContainerSnapshot(
		*containerSnapshot,
		request,
		responseCmd == protocol.CmdBagListResp,
	)

	walletSnapshot, err := h.walletService.GetRuntimeWallet(ctx, sess.PlayerID)
	if err != nil {
		if errors.Is(err, wallet.ErrWalletNotFound) {
			return sendError(conn, seq, errcode.WSCodeWalletQueryFailed, "wallet not found", err)
		}
		return sendError(conn, seq, errcode.WSCodeWalletQueryFailed, "load wallet snapshot failed", err)
	}

	responseBody := protocol.ContainerListResp{
		Container: toProtocolContainerSnapshot(paginatedSnapshot, page, pageSize, category, totalItems),
		Wallet:    toProtocolWalletSnapshot(*walletSnapshot),
	}
	if responseCmd == protocol.CmdBagListResp {
		equippedItems := []protocol.PlayerEquippedItemSnapshot{}
		if h.equipmentService != nil {
			items, listErr := h.equipmentService.ListEquipped(ctx, sess.PlayerID)
			if listErr != nil {
				return sendError(conn, seq, errcode.WSCodeInteractFailed, "load player equipment failed", listErr)
			}
			equippedItems = toProtocolEquippedItems(items)
		}
		packet, err := protocol.NewJSONPacket(responseCmd, seq, errcode.WSCodeSuccess, protocol.BagListResp{
			Container:     responseBody.Container,
			Wallet:        responseBody.Wallet,
			EquippedItems: equippedItems,
		})
		if err != nil {
			return err
		}
		return conn.SendPacket(packet)
	}

	packet, err := protocol.NewJSONPacket(responseCmd, seq, errcode.WSCodeSuccess, responseBody)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func toProtocolContainerSnapshot(
	snapshot bag.RuntimeContainerSnapshot,
	page uint32,
	pageSize uint32,
	category string,
	totalItems uint32,
) protocol.ContainerSnapshot {
	items := make([]protocol.ContainerItemSnapshot, 0, len(snapshot.Items))
	for _, itemValue := range snapshot.Items {
		items = append(items, toProtocolContainerItemSnapshot(itemValue))
	}
	return protocol.ContainerSnapshot{
		ContainerType: snapshot.ContainerType,
		Capacity:      snapshot.Capacity,
		MaxCapacity:   snapshot.MaxCapacity,
		UsedSlots:     snapshot.UsedSlots,
		Page:          page,
		PageSize:      pageSize,
		TotalItems:    totalItems,
		Category:      category,
		Items:         items,
	}
}

// buildPaginatedContainerSnapshot 按请求分类和分页裁切容器快照。
// 仅 BAG_LIST_REQ 当前会走服务端分页；其他容器查询仍保持原始完整列表契约。
func buildPaginatedContainerSnapshot(
	snapshot bag.RuntimeContainerSnapshot,
	request protocol.BagListReq,
	useBagPagination bool,
) (bag.RuntimeContainerSnapshot, uint32, uint32, string, uint32) {
	if !useBagPagination || !strings.EqualFold(snapshot.ContainerType, bag.ContainerTypeBag) {
		return snapshot, 1, uint32(len(snapshot.Items)), bagListCategoryAll, uint32(len(snapshot.Items))
	}
	page, pageSize, category := normalizeBagListQuery(request)
	filteredItems := filterBagSnapshotItems(snapshot.Items, category)
	pageItems := paginateBagSnapshotItems(filteredItems, page, pageSize)
	snapshot.Items = pageItems
	return snapshot, page, pageSize, category, uint32(len(filteredItems))
}

// normalizeBagListQuery 为背包列表分页提供稳定默认值，兼容旧客户端空请求。
func normalizeBagListQuery(request protocol.BagListReq) (uint32, uint32, string) {
	page := request.Page
	if page == 0 {
		page = defaultBagListPage
	}
	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = defaultBagListPageSize
	}
	category := strings.ToLower(strings.TrimSpace(request.Category))
	switch category {
	case "", bagListCategoryAll:
		category = bagListCategoryAll
	case bagListCategoryEquip:
		category = bagListCategoryEquip
	case bagListCategoryOther:
		category = bagListCategoryOther
	case bagListCategoryEnhanceMaterial:
		category = bagListCategoryEnhanceMaterial
	default:
		category = bagListCategoryAll
	}
	return page, pageSize, category
}

// filterBagSnapshotItems 按背包分类筛选运行时物品列表；装备单独归类，强化材料按 item_sub_type 过滤，其余统一归到“其他”。
func filterBagSnapshotItems(items []bag.RuntimeItemSnapshot, category string) []bag.RuntimeItemSnapshot {
	if category == bagListCategoryAll {
		return append([]bag.RuntimeItemSnapshot{}, items...)
	}
	filtered := make([]bag.RuntimeItemSnapshot, 0, len(items))
	for _, itemValue := range items {
		isEquipment := strings.EqualFold(itemValue.ItemType, bagListCategoryEquip)
		isEnhanceMaterial := strings.EqualFold(itemValue.ItemSubType, bag.ItemSubTypeEquipmentEnhance)
		if category == bagListCategoryEquip && isEquipment {
			filtered = append(filtered, itemValue)
			continue
		}
		if category == bagListCategoryEnhanceMaterial && isEnhanceMaterial {
			filtered = append(filtered, itemValue)
			continue
		}
		if category == bagListCategoryOther && !isEquipment && !isEnhanceMaterial {
			filtered = append(filtered, itemValue)
		}
	}
	return filtered
}

// paginateBagSnapshotItems 按页码和页长裁切列表，保持服务端原始顺序不变。
func paginateBagSnapshotItems(items []bag.RuntimeItemSnapshot, page uint32, pageSize uint32) []bag.RuntimeItemSnapshot {
	if len(items) == 0 {
		return []bag.RuntimeItemSnapshot{}
	}
	startIndex := int((page - 1) * pageSize)
	if startIndex >= len(items) {
		return []bag.RuntimeItemSnapshot{}
	}
	endIndex := startIndex + int(pageSize)
	if endIndex > len(items) {
		endIndex = len(items)
	}
	return append([]bag.RuntimeItemSnapshot{}, items[startIndex:endIndex]...)
}

func (h *BagHandler) handleContainerTransfer(conn packetSender, seq uint32, entityID uint64, fromContainerType string, toContainerType string, fromSlotIndex uint32, quantity uint64, responseCmd uint16) error {
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if err := h.ensureWarehouseAccess(context.Background(), sess.PlayerID, entityID); err != nil {
		return h.sendTransferResponse(conn, seq, responseCmd, false, err.Error(), nil)
	}

	result, err := h.bagService.MoveRuntimeItemBetweenContainers(context.Background(), sess.PlayerID, fromContainerType, toContainerType, fromSlotIndex, quantity)
	if err != nil {
		switch {
		case errors.Is(err, bag.ErrWarehouseAccessDenied):
			return h.sendTransferResponse(conn, seq, responseCmd, false, "warehouse access denied", nil)
		case errors.Is(err, bag.ErrContainerItemNotFound):
			return h.sendTransferResponse(conn, seq, responseCmd, false, "source slot is empty", nil)
		case errors.Is(err, bag.ErrInvalidTransferQuantity):
			return h.sendTransferResponse(conn, seq, responseCmd, false, "invalid quantity", nil)
		case errors.Is(err, bag.ErrContainerCapacityFull):
			return h.sendTransferResponse(conn, seq, responseCmd, false, "target container is full", nil)
		case errors.Is(err, bag.ErrItemCannotStore):
			return h.sendTransferResponse(conn, seq, responseCmd, false, "item cannot store", nil)
		default:
			return sendError(conn, seq, errcode.WSCodeBagListFailed, "transfer container item failed", err)
		}
	}

	if err := h.sendTransferResponse(conn, seq, responseCmd, true, "success", result); err != nil {
		return err
	}
	fromSnapshot, err := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, fromContainerType)
	if err == nil && fromSnapshot != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*fromSnapshot)))
	}
	toSnapshot, err := h.bagService.ListRuntimeContainer(context.Background(), sess.PlayerID, toContainerType)
	if err == nil && toSnapshot != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*toSnapshot)))
	}
	return nil
}

func (h *BagHandler) ensureWarehouseAccess(ctx context.Context, playerID uint64, entityID uint64) error {
	if h.playerService == nil || h.worldService == nil || h.npcService == nil {
		return bag.ErrWarehouseAccessDenied
	}
	profile, err := h.playerService.GetProfile(ctx, playerID)
	if err != nil {
		return err
	}
	sceneSnapshot, err := h.worldService.GetSceneSnapshot(ctx, playerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
	if err != nil {
		return err
	}
	for _, entity := range sceneSnapshot.NearbyEntities {
		if entityID != 0 && entity.EntityID != entityID {
			continue
		}
		entries, err := h.npcService.ListMenuEntriesByEntityID(ctx, entity.EntityID)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.EqualFold(entry.EntryType, "warehouse") {
				return nil
			}
		}
	}
	return bag.ErrWarehouseAccessDenied
}

func (h *BagHandler) ensureShopAccess(ctx context.Context, playerID uint64, shopID uint64) error {
	if h.playerService == nil || h.worldService == nil || h.npcService == nil {
		return bag.ErrWarehouseAccessDenied
	}
	profile, err := h.playerService.GetProfile(ctx, playerID)
	if err != nil {
		return err
	}
	sceneSnapshot, err := h.worldService.GetSceneSnapshot(ctx, playerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
	if err != nil {
		return err
	}
	for _, entity := range sceneSnapshot.NearbyEntities {
		if shopID != 0 && entity.EntityID != shopID {
			continue
		}
		entries, err := h.npcService.ListMenuEntriesByEntityID(ctx, entity.EntityID)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.EqualFold(entry.EntryType, "shop") {
				return nil
			}
		}
	}
	return bag.ErrWarehouseAccessDenied
}

func (h *BagHandler) sendTransferResponse(conn packetSender, seq uint32, responseCmd uint16, accepted bool, reason string, result *bag.RuntimeTransferResult) error {
	if responseCmd == protocol.CmdBagToWarehouseResp {
		payload := protocol.BagToWarehouseResp{Accepted: accepted, Reason: reason}
		if result != nil {
			payload.MovedItemID = result.MovedItemID
			payload.MovedItemUID = result.MovedItemUID
			payload.MovedQuantity = result.MovedQuantity
			payload.FromContainerType = result.FromContainerType
			payload.ToContainerType = result.ToContainerType
			payload.FromSlotIndex = result.FromSlotIndex
			payload.ToSlotIndex = result.ToSlotIndex
		}
		packet, err := protocol.NewJSONPacket(responseCmd, seq, errcode.WSCodeSuccess, payload)
		if err != nil {
			return err
		}
		return conn.SendPacket(packet)
	}
	payload := protocol.WarehouseToBagResp{Accepted: accepted, Reason: reason}
	if result != nil {
		payload.MovedItemID = result.MovedItemID
		payload.MovedItemUID = result.MovedItemUID
		payload.MovedQuantity = result.MovedQuantity
		payload.FromContainerType = result.FromContainerType
		payload.ToContainerType = result.ToContainerType
		payload.FromSlotIndex = result.FromSlotIndex
		payload.ToSlotIndex = result.ToSlotIndex
	}
	packet, err := protocol.NewJSONPacket(responseCmd, seq, errcode.WSCodeSuccess, payload)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func buildContainerUpdatePush(snapshot bag.RuntimeContainerSnapshot) protocol.BagUpdatePush {
	updates := make([]protocol.BagSlotUpdate, 0, snapshot.Capacity)
	occupiedBySlot := make(map[uint32]protocol.ContainerItemSnapshot, len(snapshot.Items))
	for _, itemValue := range snapshot.Items {
		occupiedBySlot[itemValue.SlotIndex] = toProtocolContainerItemSnapshot(itemValue)
	}
	for slotIndex := uint32(1); slotIndex <= snapshot.Capacity; slotIndex++ {
		itemSnapshot, ok := occupiedBySlot[slotIndex]
		if !ok {
			updates = append(updates, protocol.BagSlotUpdate{
				SlotIndex: slotIndex,
				Deleted:   true,
			})
			continue
		}
		itemSnapshotCopy := itemSnapshot
		updates = append(updates, protocol.BagSlotUpdate{
			SlotIndex: slotIndex,
			Deleted:   false,
			Item:      &itemSnapshotCopy,
		})
	}
	return protocol.BagUpdatePush{
		ContainerType: snapshot.ContainerType,
		Capacity:      snapshot.Capacity,
		MaxCapacity:   snapshot.MaxCapacity,
		UsedSlots:     snapshot.UsedSlots,
		Updates:       updates,
	}
}

func (h *BagHandler) sendContainerMoveResponse(conn packetSender, seq uint32, moved bool, reason string, request protocol.ContainerMoveReq) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdContainerMoveResp, seq, errcode.WSCodeSuccess, protocol.ContainerMoveResp{
		ContainerType: request.ContainerType,
		FromSlotIndex: request.FromSlotIndex,
		ToSlotIndex:   request.ToSlotIndex,
		Moved:         moved,
		Reason:        reason,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BagHandler) sendContainerMoveResult(conn packetSender, seq uint32, result bag.RuntimeMoveResult, reason string) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdContainerMoveResp, seq, errcode.WSCodeSuccess, protocol.ContainerMoveResp{
		ContainerType: result.ContainerType,
		FromSlotIndex: result.FromSlotIndex,
		ToSlotIndex:   result.ToSlotIndex,
		Moved:         result.Moved,
		Reason:        reason,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}
