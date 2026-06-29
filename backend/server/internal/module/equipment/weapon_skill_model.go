package equipment

import (
	"fmt"
	"strconv"
	"strings"
)

// AdminWeaponSkillConfig 描述装备模板上挂载的单条武器技能及其基础等级。
type AdminWeaponSkillConfig struct {
	SkillID   uint32 `json:"skill_id"`
	BaseLevel uint32 `json:"base_level"`
}

// RuntimeWeaponSkill 描述运行时解析后的武器技能及其有效等级。
type RuntimeWeaponSkill struct {
	SkillID uint32 `json:"skill_id"`
	Level   uint32 `json:"level"`
}

// IsWeaponEquipSlot 判断槽位是否允许配置武器附加技能。
func IsWeaponEquipSlot(slot EquipSlot) bool {
	return slot == EquipSlotWeapon || slot == EquipSlotClassWeapon
}

// ComputeWeaponSkillLevel 根据基础等级、每强化一级成长与当前强化等级计算有效技能等级。
func ComputeWeaponSkillLevel(baseLevel uint32, levelPerEnhance uint32, enhanceLevel uint32) uint32 {
	return baseLevel + levelPerEnhance*enhanceLevel
}

// ResolveWeaponSkills 按模板配置与实例强化等级解析武器技能有效等级列表。
func ResolveWeaponSkills(
	configs []AdminWeaponSkillConfig,
	levelPerEnhance map[string]uint32,
	enhanceLevel uint32,
) []RuntimeWeaponSkill {
	if len(configs) == 0 {
		return nil
	}
	result := make([]RuntimeWeaponSkill, 0, len(configs))
	seen := make(map[uint32]struct{}, len(configs))
	for _, config := range configs {
		if config.SkillID == 0 {
			continue
		}
		if _, exists := seen[config.SkillID]; exists {
			continue
		}
		seen[config.SkillID] = struct{}{}
		perEnhance := levelPerEnhance[weaponSkillLevelEnhanceKey(config.SkillID)]
		result = append(result, RuntimeWeaponSkill{
			SkillID: config.SkillID,
			Level:   ComputeWeaponSkillLevel(config.BaseLevel, perEnhance, enhanceLevel),
		})
	}
	return result
}

// CollectWeaponSkillIDs 提取模板配置中的 skill_id 列表，供后台校验引用。
func CollectWeaponSkillIDs(configs []AdminWeaponSkillConfig) []uint32 {
	if len(configs) == 0 {
		return nil
	}
	result := make([]uint32, 0, len(configs))
	seen := make(map[uint32]struct{}, len(configs))
	for _, config := range configs {
		if config.SkillID == 0 {
			continue
		}
		if _, exists := seen[config.SkillID]; exists {
			continue
		}
		seen[config.SkillID] = struct{}{}
		result = append(result, config.SkillID)
	}
	return result
}

// NormalizeEnhancePerLevelWeaponSkillLevels 清洗强化成长 map，键统一为 skill_id 字符串。
func NormalizeEnhancePerLevelWeaponSkillLevels(values map[string]uint32) map[string]uint32 {
	if len(values) == 0 {
		return map[string]uint32{}
	}
	result := make(map[string]uint32, len(values))
	for key, value := range values {
		normalizedKey := normalizeSkillEnhanceKey(key)
		if normalizedKey == "" || value == 0 {
			continue
		}
		result[normalizedKey] = value
	}
	return result
}

func weaponSkillLevelEnhanceKey(skillID uint32) string {
	return strconv.FormatUint(uint64(skillID), 10)
}

func normalizeSkillEnhanceKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || parsed == 0 {
		return ""
	}
	return strconv.FormatUint(parsed, 10)
}

// FormatWeaponSkillEnhanceLabel 生成强化预览中武器技能等级行的展示标签。
func FormatWeaponSkillEnhanceLabel(skillID uint32, skillName string) string {
	if skillName == "" {
		return fmt.Sprintf("武器技能%d等级", skillID)
	}
	return fmt.Sprintf("武器技能·%s等级", skillName)
}
