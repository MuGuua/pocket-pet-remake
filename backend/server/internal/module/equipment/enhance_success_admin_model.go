package equipment

import (
	"errors"
	"strings"
	"time"
)

const (
	// MaxEnhanceTargetLevel 与 equipment_enhance_success_config.target_level 上限一致。
	MaxEnhanceTargetLevel uint32 = 15
)

var (
	// ErrInvalidEnhanceSuccessConfig 表示后台提交的全局强化成功率配置非法。
	ErrInvalidEnhanceSuccessConfig = errors.New("invalid enhance success config")
	// ErrEnhanceSuccessConfigNotFound 表示指定目标等级没有配置行。
	ErrEnhanceSuccessConfigNotFound = errors.New("enhance success config not found")
)

// AdminEnhanceSuccessConfig 描述后台维护的单条全局强化成功率。
type AdminEnhanceSuccessConfig struct {
	TargetLevel          uint32    `json:"target_level"`
	RequiredLevelMin     uint32    `json:"required_level_min"`
	RequiredLevelBandMax uint32    `json:"required_level_band_max"`
	RequiredLevelBand    string    `json:"required_level_band_label"`
	SuccessRatePct       uint32    `json:"success_rate_pct"`
	Description          string    `json:"description"`
	Status               uint8     `json:"status"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// FillRequiredLevelBandDisplay 补全穿戴等级段展示字段。
func (cfg *AdminEnhanceSuccessConfig) FillRequiredLevelBandDisplay() {
	if cfg == nil {
		return
	}
	if cfg.RequiredLevelMin == 0 {
		cfg.RequiredLevelMin = 1
	}
	cfg.RequiredLevelBandMax = ResolveRequiredLevelBandMax(cfg.RequiredLevelMin)
	cfg.RequiredLevelBand = FormatRequiredLevelBandLabel(cfg.RequiredLevelMin)
}

// AdminEnhanceSuccessConfigList 是后台列表响应。
type AdminEnhanceSuccessConfigList struct {
	Items []AdminEnhanceSuccessConfig `json:"items"`
}

// AdminUpsertEnhanceSuccessConfigInput 描述后台更新单条全局强化成功率的请求体。
type AdminUpsertEnhanceSuccessConfigInput struct {
	SuccessRatePct uint32 `json:"success_rate_pct"`
	Description    string `json:"description"`
	Status         uint8  `json:"status"`
}

// Normalize 收口后台表单默认值。
func (input AdminUpsertEnhanceSuccessConfigInput) Normalize() AdminUpsertEnhanceSuccessConfigInput {
	input.Description = strings.TrimSpace(input.Description)
	if input.Status == 0 {
		input.Status = 1
	}
	return input
}

// Validate 校验后台提交的全局强化成功率配置。
func (input AdminUpsertEnhanceSuccessConfigInput) Validate(targetLevel uint32, requiredLevelMin uint32) error {
	input = input.Normalize()
	if targetLevel == 0 || targetLevel > MaxEnhanceTargetLevel {
		return ErrInvalidEnhanceSuccessConfig
	}
	if !IsValidEnhanceRequiredLevelBandMin(requiredLevelMin) {
		return ErrInvalidEnhanceSuccessConfig
	}
	if input.SuccessRatePct > 100 {
		return ErrInvalidEnhanceSuccessConfig
	}
	if input.Status != 1 && input.Status != 0 {
		return ErrInvalidEnhanceSuccessConfig
	}
	return nil
}
