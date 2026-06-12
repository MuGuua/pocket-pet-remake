package item

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrItemDefinitionNotFound 表示后台请求的物品模板不存在。
	ErrItemDefinitionNotFound = errors.New("item definition not found")
	// ErrInvalidAdminItemInput 表示后台提交的物品模板缺少必要字段或字段值非法。
	ErrInvalidAdminItemInput = errors.New("invalid admin item input")
	// ErrItemDefinitionConflict 表示 item_id 或 item_code 等唯一键冲突。
	ErrItemDefinitionConflict = errors.New("item definition conflict")
)

// AdminListQuery 定义后台物品模板列表页的筛选参数。
// 模板列表会被商店、掉落、任务奖励与背包展示同时复用，因此分页与筛选必须由服务端统一处理。
type AdminListQuery struct {
	ItemID   uint64
	ItemType string
	Keyword  string
	Enabled  *bool
	Page     uint32
	PageSize uint32
}

// Normalize 收口默认分页和筛选字符串，避免每个 handler 重复做同样的边界处理。
func (q AdminListQuery) Normalize() AdminListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.ItemType = strings.TrimSpace(q.ItemType)
	q.Keyword = strings.TrimSpace(q.Keyword)
	return q
}

// AdminItemSummary 描述后台物品模板列表页需要展示的摘要字段。
// 列表优先突出模板主键、编码、分类、价格与启用状态，方便运营快速定位正式配置。
type AdminItemSummary struct {
	ItemID          uint64    `json:"item_id"`
	ItemCode        string    `json:"item_code"`
	ItemName        string    `json:"item_name"`
	ItemType        string    `json:"item_type"`
	ItemSubType     string    `json:"item_sub_type"`
	Quality         uint32    `json:"quality"`
	MaxStack        uint32    `json:"max_stack"`
	BuyPriceCopper  uint64    `json:"buy_price_copper"`
	SellPriceCopper uint64    `json:"sell_price_copper"`
	Usable          bool      `json:"usable"`
	CanSell         bool      `json:"can_sell"`
	CanStore        bool      `json:"can_store"`
	IsEnabled       bool      `json:"is_enabled"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// AdminItemList 是后台物品模板列表的标准分页响应。
type AdminItemList struct {
	Items    []AdminItemSummary `json:"items"`
	Total    uint64             `json:"total"`
	Page     uint32             `json:"page"`
	PageSize uint32             `json:"page_size"`
}

// AdminItemDetail 描述后台物品模板详情与编辑弹窗所需的完整字段。
// 这里先覆盖主表里的核心通用字段，后续若补扩展表可继续往结构里追加子对象。
type AdminItemDetail struct {
	ItemID             uint64    `json:"item_id"`
	ItemCode           string    `json:"item_code"`
	ItemName           string    `json:"item_name"`
	ItemType           string    `json:"item_type"`
	ItemSubType        string    `json:"item_sub_type"`
	Quality            uint32    `json:"quality"`
	Rarity             uint32    `json:"rarity"`
	Icon               string    `json:"icon"`
	Desc               string    `json:"desc"`
	MaxStack           uint32    `json:"max_stack"`
	OccupySlots        uint32    `json:"occupy_slots"`
	AutoMerge          bool      `json:"auto_merge"`
	SortWeight         int32     `json:"sort_weight"`
	Usable             bool      `json:"usable"`
	UseScope           string    `json:"use_scope"`
	TargetType         string    `json:"target_type"`
	RequiredLevel      uint32    `json:"required_level"`
	RequiredSceneID    uint64    `json:"required_scene_id"`
	BindType           string    `json:"bind_type"`
	CanSell            bool      `json:"can_sell"`
	CanDrop            bool      `json:"can_drop"`
	CanStore           bool      `json:"can_store"`
	CanTrade           bool      `json:"can_trade"`
	ExpireAtRule       string    `json:"expire_at_rule"`
	EffectType         string    `json:"effect_type"`
	EffectValue        int64     `json:"effect_value"`
	EffectParamsJSON   string    `json:"effect_params_json"`
	BuyPriceCopper     uint64    `json:"buy_price_copper"`
	SellPriceCopper    uint64    `json:"sell_price_copper"`
	RecyclePriceCopper uint64    `json:"recycle_price_copper"`
	PriceType          string    `json:"price_type"`
	IsEnabled          bool      `json:"is_enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AdminUpsertItemInput 描述后台新增或编辑物品模板时允许提交的通用字段。
// 输入会映射到统一模板主表，保证正式配置只能通过数据库事实来源生效。
type AdminUpsertItemInput struct {
	ItemID             uint64 `json:"item_id"`
	ItemCode           string `json:"item_code"`
	ItemName           string `json:"item_name"`
	ItemType           string `json:"item_type"`
	ItemSubType        string `json:"item_sub_type"`
	Quality            uint32 `json:"quality"`
	Rarity             uint32 `json:"rarity"`
	Icon               string `json:"icon"`
	Desc               string `json:"desc"`
	MaxStack           uint32 `json:"max_stack"`
	OccupySlots        uint32 `json:"occupy_slots"`
	AutoMerge          bool   `json:"auto_merge"`
	SortWeight         int32  `json:"sort_weight"`
	Usable             bool   `json:"usable"`
	UseScope           string `json:"use_scope"`
	TargetType         string `json:"target_type"`
	RequiredLevel      uint32 `json:"required_level"`
	RequiredSceneID    uint64 `json:"required_scene_id"`
	BindType           string `json:"bind_type"`
	CanSell            bool   `json:"can_sell"`
	CanDrop            bool   `json:"can_drop"`
	CanStore           bool   `json:"can_store"`
	CanTrade           bool   `json:"can_trade"`
	ExpireAtRule       string `json:"expire_at_rule"`
	EffectType         string `json:"effect_type"`
	EffectValue        int64  `json:"effect_value"`
	EffectParamsJSON   string `json:"effect_params_json"`
	BuyPriceCopper     uint64 `json:"buy_price_copper"`
	SellPriceCopper    uint64 `json:"sell_price_copper"`
	RecyclePriceCopper uint64 `json:"recycle_price_copper"`
	PriceType          string `json:"price_type"`
	IsEnabled          bool   `json:"is_enabled"`
}

// Normalize 收口后台表单默认值，避免空字符串和 0 值把数据库写成无意义的正式配置。
func (input AdminUpsertItemInput) Normalize() AdminUpsertItemInput {
	input.ItemCode = strings.TrimSpace(input.ItemCode)
	input.ItemName = strings.TrimSpace(input.ItemName)
	input.ItemType = strings.TrimSpace(input.ItemType)
	input.ItemSubType = strings.TrimSpace(input.ItemSubType)
	input.Icon = strings.TrimSpace(input.Icon)
	input.Desc = strings.TrimSpace(input.Desc)
	input.UseScope = strings.TrimSpace(input.UseScope)
	input.TargetType = strings.TrimSpace(input.TargetType)
	input.BindType = strings.TrimSpace(input.BindType)
	input.ExpireAtRule = strings.TrimSpace(input.ExpireAtRule)
	input.EffectType = strings.TrimSpace(input.EffectType)
	input.EffectParamsJSON = strings.TrimSpace(input.EffectParamsJSON)
	input.PriceType = strings.TrimSpace(input.PriceType)
	if input.Quality == 0 {
		input.Quality = 1
	}
	if input.Rarity == 0 {
		input.Rarity = 1
	}
	if input.MaxStack == 0 {
		input.MaxStack = 1
	}
	if input.OccupySlots == 0 {
		input.OccupySlots = 1
	}
	if input.BindType == "" {
		input.BindType = "none"
	}
	if input.PriceType == "" {
		input.PriceType = "base_coin"
	}
	if input.EffectParamsJSON == "" {
		input.EffectParamsJSON = "{}"
	}
	return input
}
