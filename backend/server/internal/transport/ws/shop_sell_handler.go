package wstransport

import (
	"context"
	"errors"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/shop"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

// HandleSellItem 执行服务端权威的背包物品出售（5111 -> 5112）。
//
// 流程：协议解析 -> 会话与商店距离校验 -> 读取背包格子与物品模板 -> shop.QuoteSell 定价 ->
// 先扣背包 -> 再加钱包 -> 钱包失败则把物品补回原背包 -> 回包并推送背包/钱包快照。
//
// 先扣物品再加钱的顺序是刻意的：若反过来，钱包已到账但扣物品失败，就会出现「白拿钱」；
// 现在最坏情况只是扣物品后加钱失败，此时把物品补回，玩家资产保持不变。
func (h *BagHandler) HandleSellItem(conn packetSender, packet *protocol.Packet) error {
	var request protocol.SellItemReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid sell item body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if request.ShopID == 0 || request.SlotIndex == 0 || request.ItemID == 0 || request.Quantity == 0 {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "invalid sell item request")
	}
	ctx := context.Background()
	if err := h.ensureShopAccess(ctx, sess.PlayerID, request.ShopID); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "shop access denied", err)
	}
	if h.itemService == nil || h.bagService == nil || h.walletService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopSellFailed, "sell dependencies unavailable")
	}

	// 读取背包格子的权威快照，客户端传入的 item_id 只用于校验「玩家看到的是不是这件」。
	snapshot, err := h.bagService.ListRuntimeContainer(ctx, sess.PlayerID, bag.ContainerTypeBag)
	if err != nil || snapshot == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeBagListFailed, "bag not found", err)
	}
	var stackItem *bag.RuntimeItemSnapshot
	for index := range snapshot.Items {
		if snapshot.Items[index].SlotIndex == request.SlotIndex {
			stackItem = &snapshot.Items[index]
			break
		}
	}
	if stackItem == nil || stackItem.ItemID != request.ItemID {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "bag slot item mismatch")
	}

	itemDetail, err := h.itemService.GetRuntimeItemDetail(ctx, stackItem.ItemID)
	if err != nil || itemDetail == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, "item definition not found", err)
	}
	quote, err := shop.QuoteSell(shop.SellItemTemplate{
		ItemID:          itemDetail.ItemID,
		IsEnabled:       itemDetail.IsEnabled,
		CanSell:         itemDetail.CanSell,
		SellPriceCopper: itemDetail.SellPriceCopper,
	}, shop.SellStack{
		SlotIndex:  stackItem.SlotIndex,
		ItemID:     stackItem.ItemID,
		ItemUID:    stackItem.ItemUID,
		Quantity:   stackItem.Quantity,
		IsBound:    stackItem.IsBound,
		IsEquipped: h.isItemUIDEquipped(ctx, sess.PlayerID, stackItem.ItemUID),
	}, request.Quantity)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopRequestInvalid, sellQuoteErrorReason(err), err)
	}

	// 先扣背包。
	if _, err := h.bagService.ConsumeRuntimeItemStack(ctx, sess.PlayerID, bag.ContainerTypeBag, quote.SlotIndex, quote.Quantity, "shop_sell", request.ShopID); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeShopSellFailed, "consume sold item failed", err)
	}
	// 再加钱包。
	adjustResult, err := h.walletService.AdjustRuntimeWallet(ctx, sess.PlayerID, wallet.RuntimeAdjustInput{
		ChangeTotalCopper: int64(quote.TotalPriceCopper),
		ReasonType:        "shop_sell",
		ReasonRefID:       request.ShopID,
		OperatorType:      "player",
		OperatorID:        sess.PlayerID,
	})
	if err != nil {
		// 钱包失败：把物品补回背包，并推送最新背包，避免客户端显示物品已消失。
		_, _ = h.bagService.GrantRuntimeItem(ctx, sess.PlayerID, bag.ContainerTypeBag, quote.ItemID, quote.Quantity, "shop_sell_rollback", request.ShopID, "system", sess.PlayerID)
		if rollbackSnapshot, snapshotErr := h.bagService.ListRuntimeContainer(ctx, sess.PlayerID, bag.ContainerTypeBag); snapshotErr == nil && rollbackSnapshot != nil {
			_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*rollbackSnapshot)))
		}
		return sendError(conn, packet.Seq, errcode.WSCodeShopSellFailed, "credit wallet failed", err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdSellItemResp, packet.Seq, errcode.WSCodeSuccess, protocol.SellItemResp{
		ShopID:    request.ShopID,
		SlotIndex: quote.SlotIndex,
		ItemID:    quote.ItemID,
		Quantity:  quote.Quantity,
		Income:    toProtocolCurrencyCost("base_coin", quote.TotalPriceCopper),
		Wallet:    toProtocolWalletSnapshot(adjustResult.Wallet),
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	if latest, snapshotErr := h.bagService.ListRuntimeContainer(ctx, sess.PlayerID, bag.ContainerTypeBag); snapshotErr == nil && latest != nil {
		_ = conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*latest)))
	}
	_ = pushWalletUpdatePacket(conn, adjustResult.Wallet, "shop_sell", request.ShopID)
	return nil
}

// isItemUIDEquipped 兜底检查物品实例是否仍处于佩戴状态。
// 正常流程中佩戴会把物品移出背包，这里只防御异常数据；查询失败时按「已佩戴」保守拒绝。
func (h *BagHandler) isItemUIDEquipped(ctx context.Context, playerID uint64, itemUID string) bool {
	if itemUID == "" || h.equipmentService == nil {
		return false
	}
	equipped, err := h.equipmentService.ListEquipped(ctx, playerID)
	if err != nil {
		return true
	}
	for _, item := range equipped {
		if item.ItemUID == itemUID {
			return true
		}
	}
	return false
}

// sellQuoteErrorReason 把领域错误映射为稳定的回包原因，供客户端按原因展示提示。
func sellQuoteErrorReason(err error) string {
	switch {
	case errors.Is(err, shop.ErrSellQuantityInvalid):
		return "sell quantity invalid"
	case errors.Is(err, shop.ErrBoundItemNotSellable):
		return "bound item cannot be sold"
	case errors.Is(err, shop.ErrEquippedItemNotSellable):
		return "equipped item cannot be sold"
	case errors.Is(err, shop.ErrSellPriceOverflow):
		return "sell price overflow"
	default:
		return "item cannot be sold"
	}
}
