package pet

// CombatStatCapKey 标识宠物单项战斗属性的封顶配置键，与 pet_combat_stat_cap.stat_key 对齐。
type CombatStatCapKey string

const (
	CombatStatCapHPMax                 CombatStatCapKey = "hp_max"
	CombatStatCapSpirit                CombatStatCapKey = "spirit"
	CombatStatCapSpiritMax             CombatStatCapKey = "spirit_max"
	CombatStatCapATK                   CombatStatCapKey = "atk"
	CombatStatCapDEF                   CombatStatCapKey = "def"
	CombatStatCapSPD                   CombatStatCapKey = "spd"
	CombatStatCapMANA                  CombatStatCapKey = "mana"
	CombatStatCapHitPct                CombatStatCapKey = "hit_pct"
	CombatStatCapDodgePct              CombatStatCapKey = "dodge_pct"
	CombatStatCapCritRatePct           CombatStatCapKey = "crit_rate_pct"
	CombatStatCapCritDmgPct            CombatStatCapKey = "crit_dmg_pct"
	CombatStatCapPhysicalResistPct     CombatStatCapKey = "physical_resist_pct"
	CombatStatCapReversePhysicalResist CombatStatCapKey = "reverse_physical_resist_pct"
	CombatStatCapSkillResistPct        CombatStatCapKey = "skill_resist_pct"
	CombatStatCapReverseSkillResist    CombatStatCapKey = "reverse_skill_resist_pct"
	CombatStatCapConfusionResistPct    CombatStatCapKey = "confusion_resist_pct"
	CombatStatCapSleepResistPct        CombatStatCapKey = "sleep_resist_pct"
	CombatStatCapParalysisResistPct   CombatStatCapKey = "paralysis_resist_pct"
	CombatStatCapSealResistPct         CombatStatCapKey = "seal_resist_pct"
	CombatStatCapCurseResistPct        CombatStatCapKey = "curse_resist_pct"
	CombatStatCapCritDmgResistPct      CombatStatCapKey = "crit_dmg_resist_pct"
	CombatStatCapCritResistPct         CombatStatCapKey = "crit_resist_pct"
	CombatStatCapCharacterResistPct    CombatStatCapKey = "character_resist_pct"
	CombatStatCapPetResistPct          CombatStatCapKey = "pet_resist_pct"
	CombatStatCapGuard                 CombatStatCapKey = "guard"
	CombatStatCapTalentDmgPct          CombatStatCapKey = "talent_dmg_pct"
	CombatStatCapTalentReducePct       CombatStatCapKey = "talent_reduce_pct"
	CombatStatCapElementAdvPct         CombatStatCapKey = "element_adv_pct"
	CombatStatCapElementPenaltyPct     CombatStatCapKey = "element_penalty_pct"
)

// CombatStatCaps 保存宠物各项数值属性的服务端封顶，来源为运营参考与玩法约定。
type CombatStatCaps struct {
	values map[CombatStatCapKey]uint32
}

// DefaultCombatStatCaps 返回默认封顶表；与迁移 053 种子一致。
func DefaultCombatStatCaps() CombatStatCaps {
	return CombatStatCaps{values: map[CombatStatCapKey]uint32{
		CombatStatCapHPMax:                 1_500_000,
		CombatStatCapSpirit:                1_000,
		CombatStatCapSpiritMax:             1_000,
		CombatStatCapATK:                   250_000,
		CombatStatCapDEF:                   250_000,
		CombatStatCapSPD:                   30_000,
		CombatStatCapMANA:                  50_000,
		CombatStatCapHitPct:                250,
		CombatStatCapDodgePct:              200,
		CombatStatCapCritRatePct:           150,
		CombatStatCapCritDmgPct:            2_000,
		CombatStatCapPhysicalResistPct:     150,
		CombatStatCapReversePhysicalResist: 100,
		CombatStatCapSkillResistPct:        150,
		CombatStatCapReverseSkillResist:    100,
		CombatStatCapConfusionResistPct:    700,
		CombatStatCapSleepResistPct:        700,
		CombatStatCapParalysisResistPct:    700,
		CombatStatCapSealResistPct:         700,
		CombatStatCapCurseResistPct:        700,
		CombatStatCapCritDmgResistPct:      1_000,
		CombatStatCapCritResistPct:         100,
		CombatStatCapCharacterResistPct:    100,
		CombatStatCapPetResistPct:          100,
		CombatStatCapGuard:                 250_000,
		CombatStatCapTalentDmgPct:          200,
		CombatStatCapTalentReducePct:       100,
		CombatStatCapElementAdvPct:         100,
		CombatStatCapElementPenaltyPct:     100,
	}}
}

// NewCombatStatCaps 用给定键值构造封顶表。
func NewCombatStatCaps(values map[CombatStatCapKey]uint32) CombatStatCaps {
	copied := make(map[CombatStatCapKey]uint32, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return CombatStatCaps{values: copied}
}

// MergeCombatStatCaps 用数据库覆盖值合并默认封顶表，缺失键保留默认种子。
func MergeCombatStatCaps(overrides map[CombatStatCapKey]uint32) CombatStatCaps {
	defaults := DefaultCombatStatCaps()
	merged := make(map[CombatStatCapKey]uint32, len(defaults.values))
	for key, capValue := range defaults.values {
		merged[key] = capValue
	}
	for key, capValue := range overrides {
		merged[key] = capValue
	}
	return NewCombatStatCaps(merged)
}

// Cap 读取单项封顶；未知键返回 0 表示不限制。
func (caps CombatStatCaps) Cap(key CombatStatCapKey) uint32 {
	if caps.values == nil {
		caps = DefaultCombatStatCaps()
	}
	return caps.values[key]
}

func clampToCap(value uint32, cap uint32) uint32 {
	if cap > 0 && value > cap {
		return cap
	}
	return value
}

// ClampPetCombatStats 把宠物持久化战斗字段限制在封顶表内，并保证当前 hp 不超过 hp_max。
func ClampPetCombatStats(item *Pet, caps CombatStatCaps) {
	if item == nil {
		return
	}
	if caps.values == nil {
		caps = DefaultCombatStatCaps()
	}
	item.HPMax = clampToCap(item.HPMax, caps.Cap(CombatStatCapHPMax))
	item.ATK = clampToCap(item.ATK, caps.Cap(CombatStatCapATK))
	item.DEF = clampToCap(item.DEF, caps.Cap(CombatStatCapDEF))
	item.SPD = clampToCap(item.SPD, caps.Cap(CombatStatCapSPD))
	item.MANA = clampToCap(item.MANA, caps.Cap(CombatStatCapMANA))
	item.Spirit = clampToCap(item.Spirit, caps.Cap(CombatStatCapSpirit))
	item.SpiritMax = clampToCap(item.SpiritMax, caps.Cap(CombatStatCapSpiritMax))
	if item.SpiritMax > 0 && item.Spirit > item.SpiritMax {
		item.Spirit = item.SpiritMax
	}
	item.HitPct = clampToCap(item.HitPct, caps.Cap(CombatStatCapHitPct))
	item.DodgePct = clampToCap(item.DodgePct, caps.Cap(CombatStatCapDodgePct))
	item.CritRatePct = clampToCap(item.CritRatePct, caps.Cap(CombatStatCapCritRatePct))
	item.CritDmgPct = clampToCap(item.CritDmgPct, caps.Cap(CombatStatCapCritDmgPct))
	item.PhysicalResistPct = clampToCap(item.PhysicalResistPct, caps.Cap(CombatStatCapPhysicalResistPct))
	item.ReversePhysicalResistPct = clampToCap(item.ReversePhysicalResistPct, caps.Cap(CombatStatCapReversePhysicalResist))
	item.SkillResistPct = clampToCap(item.SkillResistPct, caps.Cap(CombatStatCapSkillResistPct))
	item.ReverseSkillResistPct = clampToCap(item.ReverseSkillResistPct, caps.Cap(CombatStatCapReverseSkillResist))
	item.ConfusionResistPct = clampToCap(item.ConfusionResistPct, caps.Cap(CombatStatCapConfusionResistPct))
	item.SleepResistPct = clampToCap(item.SleepResistPct, caps.Cap(CombatStatCapSleepResistPct))
	item.ParalysisResistPct = clampToCap(item.ParalysisResistPct, caps.Cap(CombatStatCapParalysisResistPct))
	item.SealResistPct = clampToCap(item.SealResistPct, caps.Cap(CombatStatCapSealResistPct))
	item.CurseResistPct = clampToCap(item.CurseResistPct, caps.Cap(CombatStatCapCurseResistPct))
	item.CritDmgResistPct = clampToCap(item.CritDmgResistPct, caps.Cap(CombatStatCapCritDmgResistPct))
	item.CritResistPct = clampToCap(item.CritResistPct, caps.Cap(CombatStatCapCritResistPct))
	item.CharacterResistPct = clampToCap(item.CharacterResistPct, caps.Cap(CombatStatCapCharacterResistPct))
	item.PetResistPct = clampToCap(item.PetResistPct, caps.Cap(CombatStatCapPetResistPct))
	item.Guard = clampToCap(item.Guard, caps.Cap(CombatStatCapGuard))
	item.TalentDmgPct = clampToCap(item.TalentDmgPct, caps.Cap(CombatStatCapTalentDmgPct))
	item.TalentReducePct = clampToCap(item.TalentReducePct, caps.Cap(CombatStatCapTalentReducePct))
	item.ElementAdvPct = clampToCap(item.ElementAdvPct, caps.Cap(CombatStatCapElementAdvPct))
	item.ElementPenaltyPct = clampToCap(item.ElementPenaltyPct, caps.Cap(CombatStatCapElementPenaltyPct))
	if item.HPMax > 0 && item.HP > item.HPMax {
		item.HP = item.HPMax
	}
}

// ClampLineupPetCombatStats 把编队快照中的战斗字段限制在封顶表内。
func ClampLineupPetCombatStats(item *LineupPet, caps CombatStatCaps) {
	if item == nil {
		return
	}
	petSnapshot := &Pet{
		HP:                       item.HP,
		HPMax:                    item.HPMax,
		ATK:                      item.ATK,
		DEF:                      item.DEF,
		SPD:                      item.SPD,
		MANA:                     item.MANA,
		Spirit:                   item.Spirit,
		SpiritMax:                item.SpiritMax,
		HitPct:                   item.HitPct,
		DodgePct:                 item.DodgePct,
		CritRatePct:              item.CritRatePct,
		CritDmgPct:               item.CritDmgPct,
		PhysicalResistPct:        item.PhysicalResistPct,
		ReversePhysicalResistPct: item.ReversePhysicalResistPct,
		SkillResistPct:           item.SkillResistPct,
		ReverseSkillResistPct:    item.ReverseSkillResistPct,
		ConfusionResistPct:       item.ConfusionResistPct,
		SleepResistPct:           item.SleepResistPct,
		ParalysisResistPct:       item.ParalysisResistPct,
		SealResistPct:            item.SealResistPct,
		CurseResistPct:           item.CurseResistPct,
		CritDmgResistPct:         item.CritDmgResistPct,
		CritResistPct:            item.CritResistPct,
		CharacterResistPct:       item.CharacterResistPct,
		PetResistPct:             item.PetResistPct,
		Guard:                    item.Guard,
		TalentDmgPct:             item.TalentDmgPct,
		TalentReducePct:          item.TalentReducePct,
		ElementAdvPct:            item.ElementAdvPct,
		ElementPenaltyPct:        item.ElementPenaltyPct,
	}
	ClampPetCombatStats(petSnapshot, caps)
	item.HP = petSnapshot.HP
	item.HPMax = petSnapshot.HPMax
	item.ATK = petSnapshot.ATK
	item.DEF = petSnapshot.DEF
	item.SPD = petSnapshot.SPD
	item.MANA = petSnapshot.MANA
	item.Spirit = petSnapshot.Spirit
	item.SpiritMax = petSnapshot.SpiritMax
	item.HitPct = petSnapshot.HitPct
	item.DodgePct = petSnapshot.DodgePct
	item.CritRatePct = petSnapshot.CritRatePct
	item.CritDmgPct = petSnapshot.CritDmgPct
	item.PhysicalResistPct = petSnapshot.PhysicalResistPct
	item.ReversePhysicalResistPct = petSnapshot.ReversePhysicalResistPct
	item.SkillResistPct = petSnapshot.SkillResistPct
	item.ReverseSkillResistPct = petSnapshot.ReverseSkillResistPct
	item.ConfusionResistPct = petSnapshot.ConfusionResistPct
	item.SleepResistPct = petSnapshot.SleepResistPct
	item.ParalysisResistPct = petSnapshot.ParalysisResistPct
	item.SealResistPct = petSnapshot.SealResistPct
	item.CurseResistPct = petSnapshot.CurseResistPct
	item.CritDmgResistPct = petSnapshot.CritDmgResistPct
	item.CritResistPct = petSnapshot.CritResistPct
	item.CharacterResistPct = petSnapshot.CharacterResistPct
	item.PetResistPct = petSnapshot.PetResistPct
	item.Guard = petSnapshot.Guard
	item.TalentDmgPct = petSnapshot.TalentDmgPct
	item.TalentReducePct = petSnapshot.TalentReducePct
	item.ElementAdvPct = petSnapshot.ElementAdvPct
	item.ElementPenaltyPct = petSnapshot.ElementPenaltyPct
}
