package wstransport

import (
	"testing"

	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

// TestRouterHandleSellItemCreditsWalletAndPushesBag 验证出售成功：
// 背包格子 1 的 3 个宠物治疗药剂（售价 160 铜）卖出 2 个，钱包 +320 铜，格子剩 1 个。
func TestRouterHandleSellItemCreditsWalletAndPushesBag(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)
	clearPackets(conn)
	mustMovePlayerToScene(t, playerService, 3, 12, 10)

	mustHandleJSONPacket(t, router, conn, protocol.CmdSellItemReq, 401, protocol.SellItemReq{
		ShopID:    93002,
		SlotIndex: 1,
		ItemID:    3003,
		Quantity:  2,
	})
	if len(conn.packets) != 3 {
		t.Fatalf("len(conn.packets) = %d, want 3 (resp + bag push + wallet push)", len(conn.packets))
	}
	if conn.packets[0].Cmd != protocol.CmdSellItemResp || conn.packets[0].Code != errcode.WSCodeSuccess {
		t.Fatalf("packets[0] = cmd %d code %d, want sell resp success", conn.packets[0].Cmd, conn.packets[0].Code)
	}
	if conn.packets[1].Cmd != protocol.CmdBagUpdatePush || conn.packets[2].Cmd != protocol.CmdWalletUpdatePush {
		t.Fatalf("push order = %d,%d, want bag then wallet", conn.packets[1].Cmd, conn.packets[2].Cmd)
	}

	var resp protocol.SellItemResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &resp); err != nil {
		t.Fatalf("UnmarshalBody(resp) error = %v", err)
	}
	if resp.Income.TotalCopper != 320 || resp.Quantity != 2 || resp.ItemID != 3003 {
		t.Fatalf("resp = %#v, want income 320 for 2 x item 3003", resp)
	}
	// 测试钱包初始 2345678 铜。
	if resp.Wallet.TotalCopper != 2345678+320 {
		t.Fatalf("wallet = %d, want %d", resp.Wallet.TotalCopper, 2345678+320)
	}

	var bagUpdate protocol.BagUpdatePush
	if err := protocol.UnmarshalBody(conn.packets[1].Body, &bagUpdate); err != nil {
		t.Fatalf("UnmarshalBody(bag) error = %v", err)
	}
	var remaining uint64
	for _, update := range bagUpdate.Updates {
		if update.SlotIndex == 1 && update.Item != nil {
			remaining = update.Item.Quantity
		}
	}
	if remaining != 1 {
		t.Fatalf("slot 1 remaining = %d, want 1", remaining)
	}
}

// TestRouterHandleSellItemRejectsUnsellableItem 验证 can_sell=false 的物品被拒绝，且背包与钱包都不变。
func TestRouterHandleSellItemRejectsUnsellableItem(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)
	clearPackets(conn)
	mustMovePlayerToScene(t, playerService, 3, 12, 10)

	// 格子 3 是背包扩容券，模板 can_sell=false。
	mustHandleJSONPacket(t, router, conn, protocol.CmdSellItemReq, 402, protocol.SellItemReq{
		ShopID:    93002,
		SlotIndex: 3,
		ItemID:    3001,
		Quantity:  1,
	})
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want only error response", len(conn.packets))
	}
	if conn.packets[0].Code != errcode.WSCodeShopRequestInvalid {
		t.Fatalf("code = %d, want %d", conn.packets[0].Code, errcode.WSCodeShopRequestInvalid)
	}
}

// TestRouterHandleSellItemRejectsSlotMismatch 验证客户端声明的物品与格子实际物品不一致时拒绝，
// 防止客户端拿旧快照把别的物品卖掉。
func TestRouterHandleSellItemRejectsSlotMismatch(t *testing.T) {
	_, router, playerService, conn := buildWorldRouterForTest(t)
	clearPackets(conn)
	mustMovePlayerToScene(t, playerService, 3, 12, 10)

	mustHandleJSONPacket(t, router, conn, protocol.CmdSellItemReq, 403, protocol.SellItemReq{
		ShopID:    93002,
		SlotIndex: 1,
		ItemID:    1001,
		Quantity:  1,
	})
	if len(conn.packets) != 1 || conn.packets[0].Code != errcode.WSCodeShopRequestInvalid {
		t.Fatalf("packets = %d, code = %d, want single shop-invalid error", len(conn.packets), conn.packets[0].Code)
	}
}
