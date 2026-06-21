package pet

import "errors"

var (
	// ErrInvalidTalismanSlotKey 表示神符槽配置键不在支持范围内。
	ErrInvalidTalismanSlotKey = errors.New("invalid talisman slot key")
)

const (
	// TalismanSlotKeyActive 主动神符技槽配置键。
	TalismanSlotKeyActive = "active_talisman"
	// TalismanSlotKeyHero 神符技·英雄槽配置键。
	TalismanSlotKeyHero = "talisman_hero"
	// TalismanSlotKey1 神符技【1】槽配置键。
	TalismanSlotKey1 = "talisman_1"
	// TalismanSlotKey2 神符技【2】槽配置键。
	TalismanSlotKey2 = "talisman_2"
	// TalismanSlotKey3 神符技【3】槽配置键。
	TalismanSlotKey3 = "talisman_3"
)

// TalismanSlotColumns 描述神符槽在 player_pet 上对应的 enabled / skill_id 列名。
type TalismanSlotColumns struct {
	EnabledColumn string
	SkillColumn   string
}

// ResolveTalismanSlotColumns 把配置键映射到数据库列名。
func ResolveTalismanSlotColumns(slotKey string) (TalismanSlotColumns, error) {
	switch slotKey {
	case TalismanSlotKeyActive:
		return TalismanSlotColumns{EnabledColumn: "active_talisman_enabled", SkillColumn: "active_talisman_skill_id"}, nil
	case TalismanSlotKeyHero:
		return TalismanSlotColumns{EnabledColumn: "talisman_hero_enabled", SkillColumn: "talisman_hero_skill_id"}, nil
	case TalismanSlotKey1:
		return TalismanSlotColumns{EnabledColumn: "talisman_slot_1_enabled", SkillColumn: "talisman_slot_1_skill_id"}, nil
	case TalismanSlotKey2:
		return TalismanSlotColumns{EnabledColumn: "talisman_slot_2_enabled", SkillColumn: "talisman_slot_2_skill_id"}, nil
	case TalismanSlotKey3:
		return TalismanSlotColumns{EnabledColumn: "talisman_slot_3_enabled", SkillColumn: "talisman_slot_3_skill_id"}, nil
	default:
		return TalismanSlotColumns{}, ErrInvalidTalismanSlotKey
	}
}

// IsTalismanSlotEnabled 读取 loadout 中指定神符槽是否已开启。
func IsTalismanSlotEnabled(loadout SkillLoadout, slotKey string) (bool, error) {
	switch slotKey {
	case TalismanSlotKeyActive:
		return loadout.ActiveTalismanEnabled, nil
	case TalismanSlotKeyHero:
		return loadout.TalismanHeroEnabled, nil
	case TalismanSlotKey1:
		return loadout.TalismanSlot1Enabled, nil
	case TalismanSlotKey2:
		return loadout.TalismanSlot2Enabled, nil
	case TalismanSlotKey3:
		return loadout.TalismanSlot3Enabled, nil
	default:
		return false, ErrInvalidTalismanSlotKey
	}
}

// ApplyTalismanSlotUnlock 在内存 loadout 上标记神符槽已开启，可选写入 skill_id。
func ApplyTalismanSlotUnlock(loadout *SkillLoadout, slotKey string, skillID uint32) error {
	if loadout == nil {
		return ErrInvalidTalismanSlotKey
	}
	switch slotKey {
	case TalismanSlotKeyActive:
		loadout.ActiveTalismanEnabled = true
		if skillID > 0 {
			loadout.ActiveTalismanSkillID = skillID
		}
	case TalismanSlotKeyHero:
		loadout.TalismanHeroEnabled = true
		if skillID > 0 {
			loadout.TalismanHeroSkillID = skillID
		}
	case TalismanSlotKey1:
		loadout.TalismanSlot1Enabled = true
		if skillID > 0 {
			loadout.TalismanSlot1SkillID = skillID
		}
	case TalismanSlotKey2:
		loadout.TalismanSlot2Enabled = true
		if skillID > 0 {
			loadout.TalismanSlot2SkillID = skillID
		}
	case TalismanSlotKey3:
		loadout.TalismanSlot3Enabled = true
		if skillID > 0 {
			loadout.TalismanSlot3SkillID = skillID
		}
	default:
		return ErrInvalidTalismanSlotKey
	}
	return nil
}
