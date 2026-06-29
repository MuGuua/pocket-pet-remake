package item

import (
	"errors"
	"strings"
)

var errInvalidEnhanceMaterialConfig = errors.New("invalid enhance material config")

const (
	EnhanceMaterialSuccessModeBase     = "base"
	EnhanceMaterialSuccessModeBonus    = "bonus"
	EnhanceMaterialSuccessModeOverride = "override"

	EnhanceMaterialFailurePenaltyDamage    = "damage"
	EnhanceMaterialFailurePenaltyNone      = "none"
	EnhanceMaterialFailurePenaltyLevelDown = "level_down"
)

// AdminEnhanceMaterialConfig 描述后台强化材料锻造效果配置。
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
		FailureLevelDelta: 1,
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
		return errInvalidEnhanceMaterialConfig
	}
	switch cfg.FailurePenalty {
	case EnhanceMaterialFailurePenaltyDamage, EnhanceMaterialFailurePenaltyNone, EnhanceMaterialFailurePenaltyLevelDown:
	default:
		return errInvalidEnhanceMaterialConfig
	}
	if cfg.SuccessRateBonusPct > 100 || cfg.SuccessRateOverridePct > 100 {
		return errInvalidEnhanceMaterialConfig
	}
	if cfg.FailurePenalty == EnhanceMaterialFailurePenaltyLevelDown && cfg.FailureLevelDelta == 0 {
		return errInvalidEnhanceMaterialConfig
	}
	return nil
}
