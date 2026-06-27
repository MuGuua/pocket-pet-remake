package equipment

import (
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
)

// EquippedPieceTemplate 描述单件已佩戴装备用于属性聚合的模板快照。
type EquippedPieceTemplate struct {
	EquipSlot            EquipSlot
	AppearanceOnly       bool
	AppearanceSkinID     string
	BaseHP               uint32
	BaseMana             uint32
	BaseATK              uint32
	BaseDEF              uint32
	BaseSPD              uint32
	CombatStats          AdminCombatStats
	EnhancePerLevelStats map[string]uint32
	RequiredLevel        uint32
	EnhanceLevel         uint32
}

// RecalcContext 携带重算所需的成长层与封顶配置。
type RecalcContext struct {
	Progression progression.ProgressionState
	CombatBonus progression.CombatBonus
	Caps        pet.CombatStatCaps
}

// RecalcResult 是装备重算后应写回 player 表的战斗字段快照。
type RecalcResult struct {
	HPMax              uint32
	ATK                uint32
	DEF                uint32
	SPD                uint32
	MANA               uint32
	HitPct             uint32
	DodgePct           uint32
	Spirit             uint32
	SpiritMax          uint32
	CritRatePct        uint32
	CritDmgPct         uint32
	PhysicalResistPct  uint32
	SkillResistPct     uint32
	ConfusionResistPct uint32
	SleepResistPct     uint32
	ParalysisResistPct uint32
	SealResistPct      uint32
	CurseResistPct     uint32
	CritResistPct      uint32
	CritDmgResistPct   uint32
	CharacterResistPct uint32
	PetResistPct       uint32
	SkinID             string
}

// ComputePieceBonus 计算单件装备在当前强化等级下的固定数值加成。
func ComputePieceBonus(template EquippedPieceTemplate) BonusAggregate {
	if template.AppearanceOnly || SlotSkipsCombatStats(template.EquipSlot) {
		return BonusAggregate{}
	}
	enhancedSecondary := scaleEnhanceStatsJSON(template.EnhancePerLevelStats, template.EnhanceLevel)
	bonus := BonusAggregate{
		HPMax: template.BaseHP + enhancedSecondary["hp_max"],
		MANA:  template.BaseMana + enhancedSecondary["mana"],
		ATK:   template.BaseATK + enhancedSecondary["atk"],
		DEF:   template.BaseDEF + enhancedSecondary["def"],
		SPD:   template.BaseSPD + enhancedSecondary["spd"],
	}
	secondary := template.CombatStats
	secondary = addScaledEnhanceToCombatStats(secondary, enhancedSecondary)
	bonus.Spirit = secondary.Spirit
	bonus.SpiritMax = secondary.SpiritMax
	bonus.HitPct = secondary.HitPct
	bonus.DodgePct = secondary.DodgePct
	bonus.CritRatePct = secondary.CritRatePct
	bonus.CritDmgPct = secondary.CritDmgPct
	bonus.PhysicalResistPct = secondary.PhysicalResistPct
	bonus.ReversePhysicalResistPct = secondary.ReversePhysicalResistPct
	bonus.SkillResistPct = secondary.SkillResistPct
	bonus.ReverseSkillResistPct = secondary.ReverseSkillResistPct
	bonus.ConfusionResistPct = secondary.ConfusionResistPct
	bonus.SleepResistPct = secondary.SleepResistPct
	bonus.ParalysisResistPct = secondary.ParalysisResistPct
	bonus.SealResistPct = secondary.SealResistPct
	bonus.CurseResistPct = secondary.CurseResistPct
	bonus.CritDmgResistPct = secondary.CritDmgResistPct
	bonus.CritResistPct = secondary.CritResistPct
	bonus.CharacterResistPct = secondary.CharacterResistPct
	bonus.PetResistPct = secondary.PetResistPct
	return bonus
}

// SumEquippedBonus 聚合全身装备加成（时装与药囊不参与战斗属性）。
func SumEquippedBonus(templates []EquippedPieceTemplate) BonusAggregate {
	total := BonusAggregate{}
	for _, template := range templates {
		total.Add(ComputePieceBonus(template))
	}
	return total
}

// BuildRecalcResult 根据成长层、装备加成与封顶表生成最终战斗属性。
func BuildRecalcResult(ctx RecalcContext, templates []EquippedPieceTemplate, currentProfile *player.Profile) RecalcResult {
	starter := player.DefaultStarterProfile()
	equipmentBonus := SumEquippedBonus(templates)

	progressionHPMax := progression.FinalCombatValue(ctx.Progression.BaseCombat.HPMax, ctx.CombatBonus.HPMax)
	progressionATK := progression.FinalCombatValue(ctx.Progression.BaseCombat.ATK, ctx.CombatBonus.ATK)
	progressionDEF := progression.FinalCombatValue(ctx.Progression.BaseCombat.DEF, ctx.CombatBonus.DEF)
	progressionSPD := progression.FinalCombatValue(ctx.Progression.BaseCombat.SPD, ctx.CombatBonus.SPD)
	progressionMANA := progression.FinalCombatValue(ctx.Progression.BaseCombat.MANA, ctx.CombatBonus.MANA)
	progressionHitPct := progression.FinalCombatValue(ctx.Progression.BaseCombat.HitPct, ctx.CombatBonus.HitPct)
	progressionDodgePct := progression.FinalCombatValue(ctx.Progression.BaseCombat.DodgePct, ctx.CombatBonus.DodgePct)

	result := RecalcResult{
		HPMax:              progressionHPMax + equipmentBonus.HPMax,
		ATK:                progressionATK + equipmentBonus.ATK,
		DEF:                progressionDEF + equipmentBonus.DEF,
		SPD:                progressionSPD + equipmentBonus.SPD,
		MANA:               progressionMANA + equipmentBonus.MANA,
		HitPct:             progressionHitPct + equipmentBonus.HitPct,
		DodgePct:           progressionDodgePct + equipmentBonus.DodgePct,
		SpiritMax:          starter.SpiritMax + equipmentBonus.SpiritMax,
		CritRatePct:        starter.CritRatePct + equipmentBonus.CritRatePct,
		CritDmgPct:         starter.CritDmgPct + equipmentBonus.CritDmgPct,
		PhysicalResistPct:  equipmentBonus.PhysicalResistPct,
		SkillResistPct:     equipmentBonus.SkillResistPct,
		ConfusionResistPct: equipmentBonus.ConfusionResistPct,
		SleepResistPct:     equipmentBonus.SleepResistPct,
		ParalysisResistPct: equipmentBonus.ParalysisResistPct,
		SealResistPct:      equipmentBonus.SealResistPct,
		CurseResistPct:     equipmentBonus.CurseResistPct,
		CritResistPct:      equipmentBonus.CritResistPct,
		CritDmgResistPct:   equipmentBonus.CritDmgResistPct,
		CharacterResistPct: equipmentBonus.CharacterResistPct,
		PetResistPct:       equipmentBonus.PetResistPct,
	}

	result.Spirit = starter.Spirit + equipmentBonus.Spirit
	if result.SpiritMax > 0 && result.Spirit > result.SpiritMax {
		result.Spirit = result.SpiritMax
	}

	result.SkinID = resolveEquippedSkinID(templates, currentProfile)
	ClampRecalcResult(&result, ctx.Caps)
	return result
}

// ClampRecalcResult 把重算结果限制在人物/宠物共用封顶表内。
func ClampRecalcResult(result *RecalcResult, caps pet.CombatStatCaps) {
	if result == nil {
		return
	}
	result.HPMax = clampToCap(result.HPMax, caps.Cap(pet.CombatStatCapHPMax))
	result.ATK = clampToCap(result.ATK, caps.Cap(pet.CombatStatCapATK))
	result.DEF = clampToCap(result.DEF, caps.Cap(pet.CombatStatCapDEF))
	result.SPD = clampToCap(result.SPD, caps.Cap(pet.CombatStatCapSPD))
	result.MANA = clampToCap(result.MANA, caps.Cap(pet.CombatStatCapMANA))
	result.Spirit = clampToCap(result.Spirit, caps.Cap(pet.CombatStatCapSpirit))
	result.SpiritMax = clampToCap(result.SpiritMax, caps.Cap(pet.CombatStatCapSpiritMax))
	if result.SpiritMax > 0 && result.Spirit > result.SpiritMax {
		result.Spirit = result.SpiritMax
	}
	result.HitPct = clampToCap(result.HitPct, caps.Cap(pet.CombatStatCapHitPct))
	result.DodgePct = clampToCap(result.DodgePct, caps.Cap(pet.CombatStatCapDodgePct))
	result.CritRatePct = clampToCap(result.CritRatePct, caps.Cap(pet.CombatStatCapCritRatePct))
	result.CritDmgPct = clampToCap(result.CritDmgPct, caps.Cap(pet.CombatStatCapCritDmgPct))
	result.PhysicalResistPct = clampToCap(result.PhysicalResistPct, caps.Cap(pet.CombatStatCapPhysicalResistPct))
	result.SkillResistPct = clampToCap(result.SkillResistPct, caps.Cap(pet.CombatStatCapSkillResistPct))
	result.ConfusionResistPct = clampToCap(result.ConfusionResistPct, caps.Cap(pet.CombatStatCapConfusionResistPct))
	result.SleepResistPct = clampToCap(result.SleepResistPct, caps.Cap(pet.CombatStatCapSleepResistPct))
	result.ParalysisResistPct = clampToCap(result.ParalysisResistPct, caps.Cap(pet.CombatStatCapParalysisResistPct))
	result.SealResistPct = clampToCap(result.SealResistPct, caps.Cap(pet.CombatStatCapSealResistPct))
	result.CurseResistPct = clampToCap(result.CurseResistPct, caps.Cap(pet.CombatStatCapCurseResistPct))
	result.CritDmgResistPct = clampToCap(result.CritDmgResistPct, caps.Cap(pet.CombatStatCapCritDmgResistPct))
	result.CritResistPct = clampToCap(result.CritResistPct, caps.Cap(pet.CombatStatCapCritResistPct))
	result.CharacterResistPct = clampToCap(result.CharacterResistPct, caps.Cap(pet.CombatStatCapCharacterResistPct))
	result.PetResistPct = clampToCap(result.PetResistPct, caps.Cap(pet.CombatStatCapPetResistPct))
}

func clampToCap(value uint32, cap uint32) uint32 {
	if cap > 0 && value > cap {
		return cap
	}
	return value
}

func scaleEnhanceStatsJSON(perLevel map[string]uint32, enhanceLevel uint32) map[string]uint32 {
	scaled := make(map[string]uint32, len(perLevel))
	if enhanceLevel == 0 || len(perLevel) == 0 {
		return scaled
	}
	for key, value := range perLevel {
		scaled[key] = value * enhanceLevel
	}
	return scaled
}

func addScaledEnhanceToCombatStats(base AdminCombatStats, enhanced map[string]uint32) AdminCombatStats {
	base.Spirit += enhanced["spirit"]
	base.SpiritMax += enhanced["spirit_max"]
	base.HitPct += enhanced["hit_pct"]
	base.DodgePct += enhanced["dodge_pct"]
	base.CritRatePct += enhanced["crit_rate_pct"]
	base.CritDmgPct += enhanced["crit_dmg_pct"]
	base.PhysicalResistPct += enhanced["physical_resist_pct"]
	base.ReversePhysicalResistPct += enhanced["reverse_physical_resist_pct"]
	base.SkillResistPct += enhanced["skill_resist_pct"]
	base.ReverseSkillResistPct += enhanced["reverse_skill_resist_pct"]
	base.ConfusionResistPct += enhanced["confusion_resist_pct"]
	base.SleepResistPct += enhanced["sleep_resist_pct"]
	base.ParalysisResistPct += enhanced["paralysis_resist_pct"]
	base.SealResistPct += enhanced["seal_resist_pct"]
	base.CurseResistPct += enhanced["curse_resist_pct"]
	base.CritDmgResistPct += enhanced["crit_dmg_resist_pct"]
	base.CritResistPct += enhanced["crit_resist_pct"]
	base.CharacterResistPct += enhanced["character_resist_pct"]
	base.PetResistPct += enhanced["pet_resist_pct"]
	return base
}

func resolveEquippedSkinID(templates []EquippedPieceTemplate, _ *player.Profile) string {
	for _, template := range templates {
		if template.EquipSlot == EquipSlotCostume && template.AppearanceSkinID != "" {
			return template.AppearanceSkinID
		}
	}
	return player.DefaultPlayerSkinID
}

// ToRuntimeEquippedItem 把模板快照转为协议层已佩戴摘要。
func ToRuntimeEquippedItem(
	template EquippedPieceTemplate,
	itemUID string,
	itemID uint64,
	itemName string,
	icon string,
	description string,
) RuntimeEquippedItem {
	return RuntimeEquippedItem{
		EquipSlot:        string(template.EquipSlot),
		EquipSlotLabel:   EquipSlotLabel(template.EquipSlot),
		ItemUID:          itemUID,
		ItemID:           itemID,
		ItemName:         itemName,
		Icon:             icon,
		EnhanceLevel:     template.EnhanceLevel,
		RequiredLevel:    template.RequiredLevel,
		AppearanceSkinID: template.AppearanceSkinID,
		AppearanceOnly:   template.AppearanceOnly,
		Description:      description,
		Bonus:            ComputePieceBonus(template),
	}
}
