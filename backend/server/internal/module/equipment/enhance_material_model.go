package equipment

import (
	"errors"
	"fmt"
	"strings"
)

const (
	EnhanceMaterialSuccessModeBase     = "base"
	EnhanceMaterialSuccessModeBonus    = "bonus"
	EnhanceMaterialSuccessModeOverride = "override"

	EnhanceMaterialFailurePenaltyDamage   = "damage"
	EnhanceMaterialFailurePenaltyNone     = "none"
	EnhanceMaterialFailurePenaltyLevelDown = "level_down"
)

var (
	ErrInvalidEnhanceMaterialConfig = errors.New("invalid enhance material config")
)

// EnhanceMaterialConfig 描述单种强化材料对成功率与失败惩罚的修正规则。
type EnhanceMaterialConfig struct {
	ItemID                  uint64 `json:"item_id"`
	SuccessRateMode         string `json:"success_rate_mode"`
	SuccessRateBonusPct     uint32 `json:"success_rate_bonus_pct"`
	SuccessRateOverridePct  uint32 `json:"success_rate_override_pct"`
	GuaranteedSuccess       bool   `json:"guaranteed_success"`
	FailurePenalty          string `json:"failure_penalty"`
	FailureLevelDelta       uint32 `json:"failure_level_delta"`
	Description             string `json:"description"`
	Status                  uint8  `json:"status"`
}

// AdminEnhanceMaterialConfig 供后台物品编辑弹窗读写。
type AdminEnhanceMaterialConfig struct {
	SuccessRateMode        string `json:"success_rate_mode"`
	SuccessRateBonusPct    uint32 `json:"success_rate_bonus_pct"`
	SuccessRateOverridePct uint32 `json:"success_rate_override_pct"`
	GuaranteedSuccess      bool   `json:"guaranteed_success"`
	FailurePenalty         string `json:"failure_penalty"`
	FailureLevelDelta      uint32 `json:"failure_level_delta"`
	Description            string `json:"description"`
}

// DefaultAdminEnhanceMaterialConfig 返回新建强化材料的默认锻造效果。
func DefaultAdminEnhanceMaterialConfig() AdminEnhanceMaterialConfig {
	return AdminEnhanceMaterialConfig{
		SuccessRateMode: EnhanceMaterialSuccessModeBase,
		FailurePenalty:  EnhanceMaterialFailurePenaltyDamage,
	}
}

// Normalize 收口后台表单默认值。
func (cfg AdminEnhanceMaterialConfig) Normalize() AdminEnhanceMaterialConfig {
	cfg.SuccessRateMode = strings.TrimSpace(cfg.SuccessRateMode)
	cfg.FailurePenalty = strings.TrimSpace(cfg.FailurePenalty)
	cfg.Description = strings.TrimSpace(cfg.Description)
	if cfg.SuccessRateMode == "" {
		cfg.SuccessRateMode = EnhanceMaterialSuccessModeBase
	}
	if cfg.FailurePenalty == "" {
		cfg.FailurePenalty = EnhanceMaterialFailurePenaltyDamage
	}
	if cfg.GuaranteedSuccess {
		cfg.SuccessRateMode = EnhanceMaterialSuccessModeOverride
		cfg.SuccessRateOverridePct = 100
	}
	return cfg
}

// Validate 校验后台提交的锻造材料配置。
func (cfg AdminEnhanceMaterialConfig) Validate() error {
	cfg = cfg.Normalize()
	switch cfg.SuccessRateMode {
	case EnhanceMaterialSuccessModeBase, EnhanceMaterialSuccessModeBonus, EnhanceMaterialSuccessModeOverride:
	default:
		return ErrInvalidEnhanceMaterialConfig
	}
	switch cfg.FailurePenalty {
	case EnhanceMaterialFailurePenaltyDamage, EnhanceMaterialFailurePenaltyNone, EnhanceMaterialFailurePenaltyLevelDown:
	default:
		return ErrInvalidEnhanceMaterialConfig
	}
	if cfg.SuccessRateBonusPct > 100 || cfg.SuccessRateOverridePct > 100 {
		return ErrInvalidEnhanceMaterialConfig
	}
	if cfg.FailurePenalty == EnhanceMaterialFailurePenaltyLevelDown && cfg.FailureLevelDelta == 0 {
		return ErrInvalidEnhanceMaterialConfig
	}
	return nil
}

// ToRuntimeConfig 转为运行时强化事务使用的结构。
func (cfg AdminEnhanceMaterialConfig) ToRuntimeConfig(itemID uint64) EnhanceMaterialConfig {
	normalized := cfg.Normalize()
	return EnhanceMaterialConfig{
		ItemID:                 itemID,
		SuccessRateMode:        normalized.SuccessRateMode,
		SuccessRateBonusPct:    normalized.SuccessRateBonusPct,
		SuccessRateOverridePct: normalized.SuccessRateOverridePct,
		GuaranteedSuccess:      normalized.GuaranteedSuccess,
		FailurePenalty:         normalized.FailurePenalty,
		FailureLevelDelta:      normalized.FailureLevelDelta,
		Description:            normalized.Description,
		Status:                 1,
	}
}

// DefaultRuntimeEnhanceMaterialConfig 未配置材料时的运行时兜底：全局成功率 + 失败损坏。
func DefaultRuntimeEnhanceMaterialConfig(itemID uint64) EnhanceMaterialConfig {
	return EnhanceMaterialConfig{
		ItemID:          itemID,
		SuccessRateMode: EnhanceMaterialSuccessModeBase,
		FailurePenalty:  EnhanceMaterialFailurePenaltyDamage,
		Status:          1,
	}
}

// ResolveEffectiveSuccessRatePct 根据全局成功率与材料配置计算最终成功率（1~100）。
func ResolveEffectiveSuccessRatePct(baseRatePct uint32, cfg EnhanceMaterialConfig) uint32 {
	if cfg.GuaranteedSuccess {
		return 100
	}
	switch cfg.SuccessRateMode {
	case EnhanceMaterialSuccessModeOverride:
		if cfg.SuccessRateOverridePct == 0 {
			return clampSuccessRatePct(baseRatePct)
		}
		return clampSuccessRatePct(cfg.SuccessRateOverridePct)
	case EnhanceMaterialSuccessModeBonus:
		return clampSuccessRatePct(baseRatePct + cfg.SuccessRateBonusPct)
	default:
		return clampSuccessRatePct(baseRatePct)
	}
}

func clampSuccessRatePct(value uint32) uint32 {
	if value <= 0 {
		return 1
	}
	if value > 100 {
		return 100
	}
	return value
}

// FormatEnhanceFailurePenaltyLabel 生成客户端/后台可读的失败惩罚说明。
func FormatEnhanceFailurePenaltyLabel(penalty string, levelDelta uint32) string {
	switch strings.TrimSpace(penalty) {
	case EnhanceMaterialFailurePenaltyNone:
		return "失败无惩罚"
	case EnhanceMaterialFailurePenaltyLevelDown:
		if levelDelta <= 1 {
			return "失败降1级，不损坏"
		}
		return fmt.Sprintf("失败降%d级，不损坏", levelDelta)
	default:
		return "失败装备损坏"
	}
}
