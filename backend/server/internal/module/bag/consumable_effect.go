package bag

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ConfiguredUseEffectCategory 与后台 consumableEffect.ts 的大类枚举保持一致。
type ConfiguredUseEffectCategory string

const (
	ConfiguredUseEffectCategoryPlayer    ConfiguredUseEffectCategory = "player"
	ConfiguredUseEffectCategoryPet       ConfiguredUseEffectCategory = "pet"
	ConfiguredUseEffectCategoryEquipment ConfiguredUseEffectCategory = "equipment"
	ConfiguredUseEffectCategorySystem    ConfiguredUseEffectCategory = "system"
	ConfiguredUseEffectCategoryOther     ConfiguredUseEffectCategory = "other"
)

// ConfiguredUseEffectOperation 描述数值类效果是增加、减少还是设值。
type ConfiguredUseEffectOperation string

const (
	ConfiguredUseEffectOperationAdd      ConfiguredUseEffectOperation = "add"
	ConfiguredUseEffectOperationSubtract ConfiguredUseEffectOperation = "subtract"
	ConfiguredUseEffectOperationSet      ConfiguredUseEffectOperation = "set"
)

// ConfiguredUseEffect 描述 effect_params_json.use_effects 中的单条配置。
type ConfiguredUseEffect struct {
	Category  ConfiguredUseEffectCategory  `json:"category"`
	FieldKey  string                       `json:"field_key"`
	Operation ConfiguredUseEffectOperation `json:"operation"`
	Value     int64                        `json:"value"`
	BoolValue bool                         `json:"bool_value"`
	IsBoolean bool                         `json:"-"`
}

// RuntimeAppliedUseEffect 描述服务端已落地的一条使用效果，供客户端展示与排查。
type RuntimeAppliedUseEffect struct {
	Category  string `json:"category"`
	FieldKey  string `json:"field_key"`
	Operation string `json:"operation"`
	Value     int64  `json:"value,omitempty"`
	BoolValue bool   `json:"bool_value,omitempty"`
}

// ParseConfiguredUseEffects 从 effect_params_json 解析 use_effects 列表。
func ParseConfiguredUseEffects(effectParamsJSON []byte) ([]ConfiguredUseEffect, error) {
	if len(effectParamsJSON) == 0 {
		return nil, nil
	}
	var payload struct {
		UseEffects []json.RawMessage `json:"use_effects"`
	}
	if err := json.Unmarshal(effectParamsJSON, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal use_effects payload: %w", err)
	}
	if len(payload.UseEffects) == 0 {
		return nil, nil
	}
	result := make([]ConfiguredUseEffect, 0, len(payload.UseEffects))
	for _, rawEntry := range payload.UseEffects {
		entry, err := normalizeConfiguredUseEffect(rawEntry)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			continue
		}
		result = append(result, *entry)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("use_effects is empty after normalization")
	}
	return result, nil
}

func normalizeConfiguredUseEffect(rawEntry json.RawMessage) (*ConfiguredUseEffect, error) {
	var record map[string]any
	if err := json.Unmarshal(rawEntry, &record); err != nil {
		return nil, fmt.Errorf("unmarshal use effect entry: %w", err)
	}
	category := ConfiguredUseEffectCategory(strings.TrimSpace(fmt.Sprint(record["category"])))
	if !isConfiguredUseEffectCategory(category) {
		return nil, fmt.Errorf("unsupported use effect category %q", category)
	}
	fieldKey := strings.TrimSpace(fmt.Sprint(record["field_key"]))
	if fieldKey == "" {
		return nil, fmt.Errorf("use effect field_key is required")
	}
	operation := ConfiguredUseEffectOperation(strings.TrimSpace(fmt.Sprint(record["operation"])))
	if operation == "" {
		operation = ConfiguredUseEffectOperationAdd
	}
	if operation != ConfiguredUseEffectOperationAdd && operation != ConfiguredUseEffectOperationSubtract && operation != ConfiguredUseEffectOperationSet {
		return nil, fmt.Errorf("unsupported use effect operation %q", operation)
	}
	if boolValue, ok := record["value"].(bool); ok {
		return &ConfiguredUseEffect{
			Category:  category,
			FieldKey:  fieldKey,
			Operation: ConfiguredUseEffectOperationSet,
			BoolValue: boolValue,
			IsBoolean: true,
		}, nil
	}
	numericValue, err := coerceConfiguredUseEffectNumber(record["value"])
	if err != nil {
		return nil, err
	}
	if (operation == ConfiguredUseEffectOperationAdd || operation == ConfiguredUseEffectOperationSubtract) && numericValue == 0 {
		return nil, fmt.Errorf("use effect numeric value must not be zero")
	}
	return &ConfiguredUseEffect{
		Category:  category,
		FieldKey:  fieldKey,
		Operation: operation,
		Value:     numericValue,
	}, nil
}

func coerceConfiguredUseEffectNumber(rawValue any) (int64, error) {
	switch typed := rawValue.(type) {
	case nil:
		return 0, fmt.Errorf("use effect numeric value is required")
	case float64:
		return int64(typed), nil
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid use effect numeric value: %w", err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported use effect numeric value type %T", rawValue)
	}
}

func isConfiguredUseEffectCategory(category ConfiguredUseEffectCategory) bool {
	switch category {
	case ConfiguredUseEffectCategoryPlayer,
		ConfiguredUseEffectCategoryPet,
		ConfiguredUseEffectCategoryEquipment,
		ConfiguredUseEffectCategorySystem,
		ConfiguredUseEffectCategoryOther:
		return true
	default:
		return false
	}
}

// RequiresPetTarget 判断效果列表是否需要 target_pet_uid。
func RequiresPetTarget(effects []ConfiguredUseEffect) bool {
	for _, effect := range effects {
		if effect.Category == ConfiguredUseEffectCategoryPet {
			return true
		}
		if effect.Category == ConfiguredUseEffectCategorySystem && effect.FieldKey == "pet_talisman_slot_unlock" {
			return true
		}
	}
	return false
}

// RequiresEquipmentTarget 判断效果列表是否需要 target_item_uid。
func RequiresEquipmentTarget(effects []ConfiguredUseEffect) bool {
	for _, effect := range effects {
		if effect.Category == ConfiguredUseEffectCategoryEquipment {
			return true
		}
	}
	return false
}
