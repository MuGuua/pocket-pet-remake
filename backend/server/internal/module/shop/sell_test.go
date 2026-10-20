package shop

import (
	"errors"
	"testing"
)

func sellableTemplate() SellItemTemplate {
	return SellItemTemplate{ItemID: 3001, IsEnabled: true, CanSell: true, SellPriceCopper: 25}
}

func sellableStack() SellStack {
	return SellStack{SlotIndex: 2, ItemID: 3001, Quantity: 10}
}

// TestQuoteSellComputesTotalCopper 锁定总价 = 单价 × 数量（总铜币真值）。
func TestQuoteSellComputesTotalCopper(t *testing.T) {
	quote, err := QuoteSell(sellableTemplate(), sellableStack(), 4)
	if err != nil {
		t.Fatalf("QuoteSell() error = %v", err)
	}
	if quote.TotalPriceCopper != 100 || quote.UnitPriceCopper != 25 || quote.SlotIndex != 2 {
		t.Fatalf("quote = %#v, want total 100 / unit 25 / slot 2", quote)
	}
}

// TestQuoteSellRejectsInvalidRequests 覆盖每一种失败语义。
func TestQuoteSellRejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name     string
		template SellItemTemplate
		stack    SellStack
		quantity uint64
		want     error
	}{
		{name: "数量为0", template: sellableTemplate(), stack: sellableStack(), quantity: 0, want: ErrSellQuantityInvalid},
		{name: "超过持有量", template: sellableTemplate(), stack: sellableStack(), quantity: 11, want: ErrSellQuantityInvalid},
		{name: "格子物品与模板不一致", template: sellableTemplate(), stack: SellStack{ItemID: 9, Quantity: 10}, quantity: 1, want: ErrSellQuantityInvalid},
		{name: "未开启可卖", template: SellItemTemplate{ItemID: 3001, IsEnabled: true, SellPriceCopper: 25}, stack: sellableStack(), quantity: 1, want: ErrItemNotSellable},
		{name: "售价为0", template: SellItemTemplate{ItemID: 3001, IsEnabled: true, CanSell: true}, stack: sellableStack(), quantity: 1, want: ErrItemNotSellable},
		{name: "模板停用", template: SellItemTemplate{ItemID: 3001, CanSell: true, SellPriceCopper: 25}, stack: sellableStack(), quantity: 1, want: ErrItemNotSellable},
		{name: "已绑定", template: sellableTemplate(), stack: SellStack{ItemID: 3001, Quantity: 10, IsBound: true}, quantity: 1, want: ErrBoundItemNotSellable},
		{name: "佩戴中", template: sellableTemplate(), stack: SellStack{ItemID: 3001, Quantity: 1, IsEquipped: true}, quantity: 1, want: ErrEquippedItemNotSellable},
		{name: "总价溢出", template: SellItemTemplate{ItemID: 3001, IsEnabled: true, CanSell: true, SellPriceCopper: maxSellTotalCopper}, stack: sellableStack(), quantity: 2, want: ErrSellPriceOverflow},
	}
	for _, tc := range cases {
		if _, err := QuoteSell(tc.template, tc.stack, tc.quantity); !errors.Is(err, tc.want) {
			t.Fatalf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}
