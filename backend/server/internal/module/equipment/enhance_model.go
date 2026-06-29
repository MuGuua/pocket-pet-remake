package equipment

import (
	"errors"

	"pocket-pet-remake/server/internal/module/wallet"
)

var (
	ErrEquipmentEnhanceNotAllowed           = errors.New("equipment cannot be enhanced")
	ErrEquipmentEnhanceMaxLevel             = errors.New("equipment enhance level already max")
	ErrEquipmentEnhanceMaterialInsufficient = errors.New("insufficient enhance materials")
	ErrEquipmentEnhanceMaterialInvalid      = errors.New("invalid enhance material item")
	ErrEquipmentEnhanceWalletInsufficient   = errors.New("insufficient wallet balance for enhance")
	ErrEquipmentEnhanceConfigMissing        = errors.New("equipment enhance config missing")
	ErrEquipmentEnhanceEquipped             = errors.New("equipment must be unequipped to enhance")
	ErrEquipmentEnhanceDamaged              = errors.New("equipment is damaged and cannot be enhanced")
	ErrEquipmentRepairNotDamaged            = errors.New("equipment is not damaged")
	ErrEquipmentRepairMaterialInsufficient  = errors.New("insufficient repair materials")
	ErrEquipmentRepairMaterialInvalid       = errors.New("invalid repair material item")
	ErrEquipmentRepairConfigMissing         = errors.New("equipment repair config missing")
	ErrEquipmentRepairEquipped              = errors.New("equipment must be unequipped to repair")
)

// EnhanceCost 描述单次强化到目标等级所需的材料与铜币消耗。
type EnhanceCost struct {
	TargetLevel    uint32
	CostItemID     uint64
	CostQuantity   uint64
	CostGoldCopper uint64
}

// EnhanceSuccessConfig 描述强化到目标等级的成功概率。
type EnhanceSuccessConfig struct {
	TargetLevel    uint32
	SuccessRatePct uint32
}

// RepairCost 描述修复损坏装备所需的材料消耗。
type RepairCost struct {
	CostItemID   uint64
	CostQuantity uint64
}

// RepairResult 是一次装备修复的服务端权威结果。
type RepairResult struct {
	Item        RuntimeEquippedItem   `json:"item"`
	AllEquipped []RuntimeEquippedItem `json:"all_equipped"`
}

// EnhanceResult 是一次强化尝试的服务端权威结果。
type EnhanceResult struct {
	Success        bool                  `json:"success"`
	OldLevel       uint32                `json:"old_level"`
	NewLevel       uint32                `json:"new_level"`
	RatePct        uint32                `json:"rate_pct"`
	RollPct        uint32                `json:"roll_pct"`
	FailurePenalty string                `json:"failure_penalty,omitempty"`
	Item           RuntimeEquippedItem   `json:"item"`
	AllEquipped    []RuntimeEquippedItem `json:"all_equipped"`
	// Wallet 在扣减强化铜币后返回最新钱包快照，供传输层推送客户端。
	Wallet *wallet.Snapshot `json:"wallet,omitempty"`
}

// DefaultEnhanceSuccessRate 返回迁移种子默认成功率；数据库缺失时兜底。
func DefaultEnhanceSuccessRate(targetLevel uint32) uint32 {
	switch targetLevel {
	case 1, 2, 3:
		return 100
	case 4, 5, 6:
		return 90
	case 7, 8:
		return 75
	case 9:
		return 65
	case 10:
		return 55
	case 11:
		return 45
	case 12:
		return 35
	case 13:
		return 25
	case 14:
		return 15
	case 15:
		return 10
	default:
		return 0
	}
}
