package bag

import (
	"errors"
	"time"
)

var (
	// ErrBagItemNotFound 表示后台指定的背包记录不存在。
	ErrBagItemNotFound = errors.New("bag item not found")
	// ErrInvalidAdminBagInput 表示后台提交的背包请求体缺少必要字段。
	ErrInvalidAdminBagInput = errors.New("invalid admin bag input")
	// ErrBagItemConflict 表示同一玩家的同一种道具出现唯一键冲突。
	ErrBagItemConflict = errors.New("bag item conflict")
)

// AdminListQuery 描述后台背包列表的筛选与分页参数。
// 背包页按数据库记录分页，避免前端自己拼假分页数据。
type AdminListQuery struct {
	RecordID uint64
	PlayerID uint64
	ItemID   uint32
	Page     uint32
	PageSize uint32
}

// Normalize 统一收口后台列表的默认分页，避免每个 handler 重复硬编码默认值。
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
	return q
}

// AdminCreateItemInput 描述后台新增背包记录时允许提交的字段。
// 这里直接落 player_item 持久化表，保证玩家下次上线仍能看到相同数据。
type AdminCreateItemInput struct {
	PlayerID uint64 `json:"player_id"`
	ItemID   uint32 `json:"item_id"`
	Count    uint32 `json:"count"`
}

// Normalize 为后台创建背包记录补齐最小默认值，避免 count 为空时落成 0。
func (input AdminCreateItemInput) Normalize() AdminCreateItemInput {
	if input.Count == 0 {
		input.Count = 1
	}
	return input
}

// AdminUpdateItemInput 描述后台编辑背包记录时可修改的字段。
// 更新接口按整条记录覆写，避免部分字段漏传后产生意料之外的旧值残留。
type AdminUpdateItemInput struct {
	PlayerID uint64 `json:"player_id"`
	ItemID   uint32 `json:"item_id"`
	Count    uint32 `json:"count"`
}

// Normalize 保证后台编辑至少会把数量收敛到合法的正整数。
func (input AdminUpdateItemInput) Normalize() AdminUpdateItemInput {
	if input.Count == 0 {
		input.Count = 1
	}
	return input
}

// AdminItemSummary 是后台列表页消费的背包摘要记录。
// 这里额外带上玩家名，减少运营同学在多页面间来回跳转确认归属。
type AdminItemSummary struct {
	RecordID   uint64    `json:"record_id"`
	PlayerID   uint64    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	ItemID     uint32    `json:"item_id"`
	Count      uint32    `json:"count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AdminItemList 封装背包列表响应，分页器必须以服务端 total 为准。
type AdminItemList struct {
	Items    []AdminItemSummary `json:"items"`
	Total    uint64             `json:"total"`
	Page     uint32             `json:"page"`
	PageSize uint32             `json:"page_size"`
}

// AdminItemDetail 是后台详情弹窗读取的完整背包记录快照。
type AdminItemDetail struct {
	RecordID   uint64     `json:"record_id"`
	PlayerID   uint64     `json:"player_id"`
	PlayerName string     `json:"player_name"`
	ItemID     uint32     `json:"item_id"`
	Count      uint32     `json:"count"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}
