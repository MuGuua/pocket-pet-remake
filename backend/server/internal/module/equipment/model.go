package equipment

// EquipSlot 标识人物装备部位，与 item_equipment_extra.equip_slot 及 player_equipment_slot 对齐。
type EquipSlot string

const (
	EquipSlotWeapon          EquipSlot = "weapon"
	EquipSlotClassWeapon     EquipSlot = "class_weapon"
	EquipSlotHat             EquipSlot = "hat"
	EquipSlotClothes         EquipSlot = "clothes"
	EquipSlotPants           EquipSlot = "pants"
	EquipSlotShoes           EquipSlot = "shoes"
	EquipSlotNecklace        EquipSlot = "necklace"
	EquipSlotRing            EquipSlot = "ring"
	EquipSlotHeroRing        EquipSlot = "hero_ring"
	EquipSlotBadge           EquipSlot = "badge"
	EquipSlotCharm           EquipSlot = "charm"
	EquipSlotMedicinePouch   EquipSlot = "medicine_pouch"
	EquipSlotGuardianRing    EquipSlot = "guardian_ring"
	EquipSlotClassBadge      EquipSlot = "class_badge"
	EquipSlotCostume         EquipSlot = "costume"
	EquipSlotElementBracelet EquipSlot = "element_bracelet"
	EquipSlotRebirthStone    EquipSlot = "rebirth_stone"
)

// AllEquipSlots 返回全部合法人物装备槽位，供 Admin 下拉与校验复用。
func AllEquipSlots() []EquipSlot {
	return []EquipSlot{
		EquipSlotWeapon,
		EquipSlotClassWeapon,
		EquipSlotHat,
		EquipSlotClothes,
		EquipSlotPants,
		EquipSlotShoes,
		EquipSlotNecklace,
		EquipSlotRing,
		EquipSlotHeroRing,
		EquipSlotBadge,
		EquipSlotCharm,
		EquipSlotMedicinePouch,
		EquipSlotGuardianRing,
		EquipSlotClassBadge,
		EquipSlotCostume,
		EquipSlotElementBracelet,
		EquipSlotRebirthStone,
	}
}

// IsValidEquipSlot 判断槽位字符串是否为已定义的人物装备部位。
func IsValidEquipSlot(value string) bool {
	for _, slot := range AllEquipSlots() {
		if string(slot) == value {
			return true
		}
	}
	return false
}

// EquipSlotLabel 返回槽位中文展示名。
func EquipSlotLabel(slot EquipSlot) string {
	switch slot {
	case EquipSlotWeapon:
		return "武器"
	case EquipSlotClassWeapon:
		return "职业武器"
	case EquipSlotHat:
		return "帽子"
	case EquipSlotClothes:
		return "衣服"
	case EquipSlotPants:
		return "裤子"
	case EquipSlotShoes:
		return "鞋子"
	case EquipSlotNecklace:
		return "项链"
	case EquipSlotRing:
		return "戒指"
	case EquipSlotHeroRing:
		return "英雄之戒"
	case EquipSlotBadge:
		return "徽章"
	case EquipSlotCharm:
		return "护符"
	case EquipSlotMedicinePouch:
		return "药囊"
	case EquipSlotGuardianRing:
		return "守护之戒"
	case EquipSlotClassBadge:
		return "职业徽章"
	case EquipSlotCostume:
		return "时装"
	case EquipSlotElementBracelet:
		return "元素手镯"
	case EquipSlotRebirthStone:
		return "转生之石"
	default:
		return string(slot)
	}
}

// SlotSkipsCombatStats 判断该槽位是否不参与战斗属性聚合。
func SlotSkipsCombatStats(slot EquipSlot) bool {
	return slot == EquipSlotCostume || slot == EquipSlotMedicinePouch
}
