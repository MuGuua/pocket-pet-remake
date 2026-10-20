// Package shop 承载商店买卖的价格结算与规则校验。
// 背包（bag）只负责容器增删，钱包（wallet）只负责货币真值与流水，
// 「这件物品能不能卖、卖多少钱」这类商店规则统一收口在这里，避免散落在 WebSocket handler 中。
package shop

import "errors"

var (
	// ErrSellQuantityInvalid 出售数量为 0 或超过当前格子持有量。
	ErrSellQuantityInvalid = errors.New("sell quantity invalid")
	// ErrItemNotSellable 物品模板未启用、未开启 can_sell 或售价为 0。
	ErrItemNotSellable = errors.New("item not sellable")
	// ErrBoundItemNotSellable 已绑定物品禁止出售，避免绑定奖励被直接变现。
	ErrBoundItemNotSellable = errors.New("bound item not sellable")
	// ErrEquippedItemNotSellable 正在佩戴的装备必须先卸下才能出售。
	ErrEquippedItemNotSellable = errors.New("equipped item not sellable")
	// ErrSellPriceOverflow 单价 × 数量超出 int64，防止异常配置导致钱包溢出。
	ErrSellPriceOverflow = errors.New("sell price overflow")
)

// SellItemTemplate 是出售校验需要的物品模板字段，由调用方从 item 模块读取后传入。
type SellItemTemplate struct {
	ItemID          uint64
	IsEnabled       bool
	CanSell         bool
	SellPriceCopper uint64
}

// SellStack 是出售校验需要的背包格子字段，由调用方从 bag 模块读取后传入。
type SellStack struct {
	SlotIndex  uint32
	ItemID     uint64
	ItemUID    string
	Quantity   uint64
	IsBound    bool
	IsEquipped bool
}

// SellQuote 是一次出售的权威报价：服务端按它扣背包、加钱包，客户端只做展示。
type SellQuote struct {
	ItemID           uint64
	SlotIndex        uint32
	Quantity         uint64
	UnitPriceCopper  uint64
	TotalPriceCopper uint64
}

// maxSellTotalCopper 是单次出售总价上限，与钱包 int64 的调整入参保持一致。
const maxSellTotalCopper uint64 = 1<<63 - 1

// QuoteSell 校验出售请求并计算总价。价格口径统一按总铜币真值处理。
// 判定顺序：数量 -> 模板可卖 -> 绑定 -> 佩戴 -> 溢出，前面的失败语义优先。
func QuoteSell(template SellItemTemplate, stack SellStack, quantity uint64) (*SellQuote, error) {
	if quantity == 0 || quantity > stack.Quantity || stack.ItemID != template.ItemID {
		return nil, ErrSellQuantityInvalid
	}
	if !template.IsEnabled || !template.CanSell || template.SellPriceCopper == 0 {
		return nil, ErrItemNotSellable
	}
	if stack.IsBound {
		return nil, ErrBoundItemNotSellable
	}
	if stack.IsEquipped {
		return nil, ErrEquippedItemNotSellable
	}
	if template.SellPriceCopper > maxSellTotalCopper/quantity {
		return nil, ErrSellPriceOverflow
	}
	return &SellQuote{
		ItemID:           template.ItemID,
		SlotIndex:        stack.SlotIndex,
		Quantity:         quantity,
		UnitPriceCopper:  template.SellPriceCopper,
		TotalPriceCopper: template.SellPriceCopper * quantity,
	}, nil
}
