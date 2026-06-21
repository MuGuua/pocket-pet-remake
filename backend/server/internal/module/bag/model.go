package bag

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrBagItemNotFound 表示后台指定的容器记录不存在。
	ErrBagItemNotFound = errors.New("bag item not found")
	// ErrInvalidAdminBagInput 表示后台提交的容器记录缺少必要字段或字段值非法。
	ErrInvalidAdminBagInput = errors.New("invalid admin bag input")
	// ErrBagItemConflict 表示同一容器格子或实例引用出现唯一键冲突。
	ErrBagItemConflict = errors.New("bag item conflict")
	// ErrInvalidContainerType 表示运行时请求传入了当前系统不支持的容器类型。
	ErrInvalidContainerType = errors.New("invalid container type")
	// ErrContainerNotFound 表示玩家目标容器尚未初始化或数据库中不存在。
	ErrContainerNotFound = errors.New("container not found")
	// ErrContainerItemNotFound 表示指定格子没有对应物品。
	ErrContainerItemNotFound = errors.New("container item not found")
	// ErrInvalidTransferQuantity 表示跨容器移动请求的数量非法。
	ErrInvalidTransferQuantity = errors.New("invalid transfer quantity")
	// ErrContainerCapacityFull 表示目标容器无可用格子且无法继续叠堆。
	ErrContainerCapacityFull = errors.New("container capacity full")
	// ErrItemCannotStore 表示该物品模板当前不允许进入仓库。
	ErrItemCannotStore = errors.New("item cannot store")
	// ErrWarehouseAccessDenied 表示玩家当前不满足仓库存取前置条件。
	ErrWarehouseAccessDenied = errors.New("warehouse access denied")
	// ErrInvalidContainerMove 表示同容器移动请求参数或当前状态不合法。
	ErrInvalidContainerMove = errors.New("invalid container move")
	// ErrItemNotUsable 表示玩家尝试使用一个未声明可使用或当前不支持主动使用的物品。
	ErrItemNotUsable = errors.New("item not usable")
	// ErrContainerCapacityLimit 表示扩容类道具命中了容器最大上限，不能再继续增加容量。
	ErrContainerCapacityLimit = errors.New("container capacity limit reached")
	// ErrUnsupportedItemEffect 表示模板虽然允许使用，但服务端尚未接入该效果类型的正式处理逻辑。
	ErrUnsupportedItemEffect = errors.New("unsupported item effect")
	// ErrUseTargetRequired 表示当前物品效果需要明确的目标，但请求里没有带上目标信息。
	ErrUseTargetRequired = errors.New("use target required")
	// ErrUseTargetNotFound 表示请求里的目标对象不属于当前玩家，或数据库中不存在。
	ErrUseTargetNotFound = errors.New("use target not found")
	// ErrItemUseNoEffect 表示目标当前状态已经是满值，继续使用不会产生正式效果。
	ErrItemUseNoEffect = errors.New("item use has no effect")
)

const (
	// ContainerTypeBag 表示玩家随身背包。
	ContainerTypeBag = "bag"
	// ContainerTypeWarehouse 表示玩家个人仓库。
	ContainerTypeWarehouse = "warehouse"
)

// AdminListQuery 描述后台容器列表的筛选与分页参数。
// 列表页直接按数据库记录分页，避免前端拼接假分页造成与真实格子状态不一致。
type AdminListQuery struct {
	RecordID      uint64
	PlayerID      uint64
	ContainerType string
	ItemID        uint64
	ItemUID       string
	Page          uint32
	PageSize      uint32
}

// Normalize 统一收口后台列表的默认分页与字符串筛选值。
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
	q.ContainerType = strings.TrimSpace(q.ContainerType)
	q.ItemUID = strings.TrimSpace(q.ItemUID)
	return q
}

// AdminCreateItemInput 描述后台新增容器记录时允许提交的字段。
// 当前后台默认直接落 player_container_item，保证玩家下次上线看到的就是数据库真值。
type AdminCreateItemInput struct {
	PlayerID      uint64 `json:"player_id"`
	ContainerType string `json:"container_type"`
	SlotIndex     uint32 `json:"slot_index"`
	ItemID        uint64 `json:"item_id"`
	ItemUID       string `json:"item_uid"`
	Quantity      uint64 `json:"quantity"`
	IsBound       bool   `json:"is_bound"`
}

// Normalize 补齐后台最小默认值，避免容器类型和数量为空时落成无意义记录。
func (input AdminCreateItemInput) Normalize() AdminCreateItemInput {
	input.ContainerType = strings.TrimSpace(input.ContainerType)
	input.ItemUID = strings.TrimSpace(input.ItemUID)
	if input.ContainerType == "" {
		input.ContainerType = "bag"
	}
	if input.Quantity == 0 {
		input.Quantity = 1
	}
	return input
}

// AdminUpdateItemInput 描述后台编辑容器记录时允许修改的字段。
// 更新接口按整条记录覆写，避免部分字段漏传后残留旧格子状态。
type AdminUpdateItemInput struct {
	PlayerID      uint64 `json:"player_id"`
	ContainerType string `json:"container_type"`
	SlotIndex     uint32 `json:"slot_index"`
	ItemID        uint64 `json:"item_id"`
	ItemUID       string `json:"item_uid"`
	Quantity      uint64 `json:"quantity"`
	IsBound       bool   `json:"is_bound"`
}

// Normalize 保证编辑请求同样遵循新增时的默认规则。
func (input AdminUpdateItemInput) Normalize() AdminUpdateItemInput {
	return AdminCreateItemInput(input).Normalize().ToUpdate()
}

// ToUpdate 把创建输入转换回更新输入，避免重复写相同字段赋值逻辑。
func (input AdminCreateItemInput) ToUpdate() AdminUpdateItemInput {
	return AdminUpdateItemInput{
		PlayerID:      input.PlayerID,
		ContainerType: input.ContainerType,
		SlotIndex:     input.SlotIndex,
		ItemID:        input.ItemID,
		ItemUID:       input.ItemUID,
		Quantity:      input.Quantity,
		IsBound:       input.IsBound,
	}
}

// AdminItemSummary 是后台容器列表页消费的摘要记录。
// 这里额外带上玩家名、模板名和格子信息，减少运营在多个页面间来回查主键。
type AdminItemSummary struct {
	RecordID      uint64    `json:"record_id"`
	PlayerID      uint64    `json:"player_id"`
	PlayerName    string    `json:"player_name"`
	ContainerType string    `json:"container_type"`
	SlotIndex     uint32    `json:"slot_index"`
	ItemID        uint64    `json:"item_id"`
	ItemUID       string    `json:"item_uid"`
	ItemName      string    `json:"item_name"`
	ItemType      string    `json:"item_type"`
	Quantity      uint64    `json:"quantity"`
	IsBound       bool      `json:"is_bound"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AdminItemList 封装后台容器列表响应，分页器必须以服务端 total 为准。
type AdminItemList struct {
	Items    []AdminItemSummary `json:"items"`
	Total    uint64             `json:"total"`
	Page     uint32             `json:"page"`
	PageSize uint32             `json:"page_size"`
}

// AdminItemDetail 是后台详情抽屉读取的完整容器记录快照。
type AdminItemDetail struct {
	RecordID      uint64     `json:"record_id"`
	PlayerID      uint64     `json:"player_id"`
	PlayerName    string     `json:"player_name"`
	ContainerType string     `json:"container_type"`
	SlotIndex     uint32     `json:"slot_index"`
	ItemID        uint64     `json:"item_id"`
	ItemUID       string     `json:"item_uid"`
	ItemName      string     `json:"item_name"`
	ItemType      string     `json:"item_type"`
	Quantity      uint64     `json:"quantity"`
	IsBound       bool       `json:"is_bound"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// RuntimeItemSnapshot 描述玩家端运行时直接消费的格子物品快照。
// 这里保留模板与实例的混合字段，确保客户端打开背包时无需再拼装第二次请求。
type RuntimeItemSnapshot struct {
	SlotIndex    uint32     `json:"slot_index"`
	ItemID       uint64     `json:"item_id"`
	ItemUID      string     `json:"item_uid"`
	Quantity     uint64     `json:"quantity"`
	IsBound      bool       `json:"is_bound"`
	ExpireAt     *time.Time `json:"expire_at,omitempty"`
	ItemName     string     `json:"item_name"`
	ItemType     string     `json:"item_type"`
	ItemSubType  string     `json:"item_sub_type"`
	Quality      uint32     `json:"quality"`
	Icon         string     `json:"icon"`
	EnhanceLevel uint32     `json:"enhance_level"`
	Usable       bool       `json:"usable"`
	TargetType   string     `json:"target_type"`
	EffectType   string     `json:"effect_type"`
}

// RuntimeContainerSnapshot 描述一个容器的完整权威快照。
// 玩家端每次打开背包或仓库面板时，都应以该结构整体刷新本地展示状态。
type RuntimeContainerSnapshot struct {
	ContainerType string                `json:"container_type"`
	Capacity      uint32                `json:"capacity"`
	MaxCapacity   uint32                `json:"max_capacity"`
	UsedSlots     uint32                `json:"used_slots"`
	Items         []RuntimeItemSnapshot `json:"items"`
}

// RuntimeTransferResult 描述一次跨容器移动操作的核心结果。
// WebSocket 响应会直接基于该结构转成玩家端可消费的确认回包。
type RuntimeTransferResult struct {
	MovedItemID       uint64 `json:"moved_item_id"`
	MovedItemUID      string `json:"moved_item_uid"`
	MovedQuantity     uint64 `json:"moved_quantity"`
	FromContainerType string `json:"from_container_type"`
	ToContainerType   string `json:"to_container_type"`
	FromSlotIndex     uint32 `json:"from_slot_index"`
	ToSlotIndex       uint32 `json:"to_slot_index"`
}

// RuntimeSortResult 描述一次整理请求的结果。
type RuntimeSortResult struct {
	ContainerType string `json:"container_type"`
	Sorted        bool   `json:"sorted"`
}

// RuntimeMoveResult 描述同容器换位或拆分移动结果。
type RuntimeMoveResult struct {
	ContainerType string `json:"container_type"`
	FromSlotIndex uint32 `json:"from_slot_index"`
	ToSlotIndex   uint32 `json:"to_slot_index"`
	Moved         bool   `json:"moved"`
}

// RuntimeGrantResult 描述一次系统侧发放道具到背包后的结果。
// 当前主要供任务奖励、邮件附件和活动补偿复用，统一要求走服务端权威落库与日志链路。
type RuntimeGrantResult struct {
	ContainerType string `json:"container_type"`
	ItemID        uint64 `json:"item_id"`
	ItemName      string `json:"item_name"`
	ItemUID       string `json:"item_uid"`
	GrantedQty    uint64 `json:"granted_qty"`
	SlotIndex     uint32 `json:"slot_index"`
}

// RuntimeUseResult 描述一次主动使用物品后的权威结果。
// 当前第一版先覆盖扩容类功能道具，后续可继续往 Result 中追加宠物治疗、角色恢复等效果字段。
type RuntimeUseResult struct {
	ContainerType string           `json:"container_type"`
	SlotIndex     uint32           `json:"slot_index"`
	ItemID        uint64           `json:"item_id"`
	UsedQuantity  uint64           `json:"used_quantity"`
	Result        RuntimeUseEffect `json:"result"`
}

// RuntimeUseEffect 描述服务端已经生效的物品效果。
// 客户端只能展示这里返回的最终结果，不能自行推导扩容或恢复后的新状态。
type RuntimeUseEffect struct {
	EffectType   string              `json:"effect_type"`
	ExpandTarget string              `json:"expand_target"`
	ExpandSlots  uint32              `json:"expand_slots"`
	NewCapacity  uint32              `json:"new_capacity"`
	TargetPetUID uint64              `json:"target_pet_uid"`
	RestoredHP   uint32              `json:"restored_hp"`
	NewPetHP     uint32              `json:"new_pet_hp"`
	// UnlockedTalismanSlot 描述神符类道具解锁的槽位键。
	UnlockedTalismanSlot string              `json:"unlocked_talisman_slot,omitempty"`
	Rewards              []RuntimeRewardItem `json:"rewards,omitempty"`
	UpdatedPet   *RuntimePetSnapshot `json:"updated_pet,omitempty"`
}

// RuntimeRewardItem 描述一个可被统一发奖服务消费的奖励条目。
// 这里放在 bag 域里，是为了让礼包开启类道具可以先通过 use item 链路权威计算出奖励摘要，
// 再由上层 handler 决定是否继续走 reward 服务发放。
type RuntimeRewardItem struct {
	Type     string `json:"type"`
	Value    uint64 `json:"value"`
	ItemID   uint64 `json:"item_id"`
	ItemName string `json:"item_name"`
	Count    uint64 `json:"count"`
	PetID    uint64 `json:"pet_id"`
}

// RuntimePetSnapshot 描述背包使用宠物治疗类道具后需要同步给客户端的最新宠物状态。
// 这里避免直接依赖 pet 模块，保持背包领域结果结构可独立复用。
type RuntimePetSnapshot struct {
	PetUID   uint64   `json:"pet_uid"`
	PetID    uint32   `json:"pet_id"`
	Level    uint32   `json:"level"`
	Exp      uint64   `json:"exp"`
	Quality  uint32   `json:"quality"`
	HP       uint32   `json:"hp"`
	HPMax    uint32   `json:"hp_max"`
	ATK      uint32   `json:"atk"`
	DEF      uint32   `json:"def"`
	SPD      uint32   `json:"spd"`
	SkillIDs []uint32 `json:"skill_ids"`
	InLineup bool     `json:"in_lineup"`
}

// NormalizeRuntimeContainerType 统一容器类型默认值和合法性校验。
func NormalizeRuntimeContainerType(containerType string) (string, error) {
	normalized := strings.TrimSpace(containerType)
	if normalized == "" {
		return ContainerTypeBag, nil
	}
	switch normalized {
	case ContainerTypeBag, ContainerTypeWarehouse:
		return normalized, nil
	default:
		return "", ErrInvalidContainerType
	}
}
