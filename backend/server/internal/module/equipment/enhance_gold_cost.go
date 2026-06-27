package equipment

import (
	"errors"
	"strings"
	"time"
)

const (
	// EnhanceGoldIncrementModeFixed 表示每升 1 级在基础上固定增加 increment_fixed 铜币。
	EnhanceGoldIncrementModeFixed = "fixed"
	// EnhanceGoldIncrementModePercent 表示每升 1 级在上一级消耗上乘以 (1 + increment_percent/100)。
	EnhanceGoldIncrementModePercent = "percent"
)

var (
	// ErrEnhanceGoldCostConfigNotFound 表示全局强化铜币公式尚未初始化。
	ErrEnhanceGoldCostConfigNotFound = errors.New("enhance gold cost config not found")
	// ErrInvalidEnhanceGoldCostConfig 表示后台提交的铜币公式字段非法。
	ErrInvalidEnhanceGoldCostConfig = errors.New("invalid enhance gold cost config")
)

// EnhanceGoldCostConfig 描述强化至目标等级的铜币消耗计算公式参数。
type EnhanceGoldCostConfig struct {
	IsEnabled        bool      `json:"is_enabled"`
	BaseCopper       uint64    `json:"base_copper"`
	IncrementMode    string    `json:"increment_mode"`
	IncrementFixed   uint64    `json:"increment_fixed"`
	IncrementPercent uint32    `json:"increment_percent"`
	Description      string    `json:"description"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AdminEnhanceGoldCostPreviewRow 供后台展示各目标等级的计算结果。
type AdminEnhanceGoldCostPreviewRow struct {
	TargetLevel uint32 `json:"target_level"`
	CostCopper  uint64 `json:"cost_copper"`
}

// AdminEnhanceGoldCostConfigDetail 返回后台编辑页使用的完整配置与预览表。
type AdminEnhanceGoldCostConfigDetail struct {
	Config  EnhanceGoldCostConfig            `json:"config"`
	Preview []AdminEnhanceGoldCostPreviewRow `json:"preview"`
}

// AdminUpsertEnhanceGoldCostConfigInput 描述后台保存铜币公式时可提交的字段。
type AdminUpsertEnhanceGoldCostConfigInput struct {
	IsEnabled        bool   `json:"is_enabled"`
	BaseCopper       uint64 `json:"base_copper"`
	IncrementMode    string `json:"increment_mode"`
	IncrementFixed   uint64 `json:"increment_fixed"`
	IncrementPercent uint32 `json:"increment_percent"`
	Description      string `json:"description"`
}

// Normalize 清洗后台输入并统一递增模式枚举。
func (input AdminUpsertEnhanceGoldCostConfigInput) Normalize() AdminUpsertEnhanceGoldCostConfigInput {
	input.IncrementMode = strings.ToLower(strings.TrimSpace(input.IncrementMode))
	input.Description = strings.TrimSpace(input.Description)
	if input.IncrementMode != EnhanceGoldIncrementModePercent {
		input.IncrementMode = EnhanceGoldIncrementModeFixed
	}
	return input
}

// Validate 校验铜币公式参数是否在允许范围内。
func (input AdminUpsertEnhanceGoldCostConfigInput) Validate() error {
	input = input.Normalize()
	if input.IncrementMode != EnhanceGoldIncrementModeFixed && input.IncrementMode != EnhanceGoldIncrementModePercent {
		return ErrInvalidEnhanceGoldCostConfig
	}
	if input.IncrementPercent > 1000 {
		return ErrInvalidEnhanceGoldCostConfig
	}
	return nil
}

// ToConfig 把后台输入转换为运行时配置结构。
func (input AdminUpsertEnhanceGoldCostConfigInput) ToConfig(updatedAt time.Time) EnhanceGoldCostConfig {
	input = input.Normalize()
	return EnhanceGoldCostConfig{
		IsEnabled:        input.IsEnabled,
		BaseCopper:       input.BaseCopper,
		IncrementMode:    input.IncrementMode,
		IncrementFixed:   input.IncrementFixed,
		IncrementPercent: input.IncrementPercent,
		Description:      input.Description,
		UpdatedAt:        updatedAt,
	}
}

// CalculateEnhanceGoldCost 按全局公式计算强化至 targetLevel 所需铜币总量。
// targetLevel 为强化后的目标等级，例如从 +0 强化到 +1 时 targetLevel=1。
func CalculateEnhanceGoldCost(targetLevel uint32, config EnhanceGoldCostConfig) uint64 {
	if !config.IsEnabled || targetLevel == 0 {
		return 0
	}
	if targetLevel == 1 {
		return config.BaseCopper
	}
	switch config.IncrementMode {
	case EnhanceGoldIncrementModeFixed:
		return config.BaseCopper + uint64(targetLevel-1)*config.IncrementFixed
	case EnhanceGoldIncrementModePercent:
		if config.IncrementPercent == 0 {
			return config.BaseCopper
		}
		cost := config.BaseCopper
		multiplierNumerator := uint64(100 + config.IncrementPercent)
		for level := uint32(2); level <= targetLevel; level++ {
			cost = cost * multiplierNumerator / 100
		}
		return cost
	default:
		return config.BaseCopper
	}
}

// BuildEnhanceGoldCostPreview 生成 +1~+maxLevel 的铜币消耗预览表，供后台展示。
func BuildEnhanceGoldCostPreview(config EnhanceGoldCostConfig, maxLevel uint32) []AdminEnhanceGoldCostPreviewRow {
	if maxLevel == 0 {
		maxLevel = 15
	}
	rows := make([]AdminEnhanceGoldCostPreviewRow, 0, maxLevel)
	for level := uint32(1); level <= maxLevel; level++ {
		rows = append(rows, AdminEnhanceGoldCostPreviewRow{
			TargetLevel: level,
			CostCopper:  CalculateEnhanceGoldCost(level, config),
		})
	}
	return rows
}
