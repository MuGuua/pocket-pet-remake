package pet

// AdminPetCombatStats 描述运营后台可编辑的宠物次要战斗属性，与 player_pet 扩展列对齐。
type AdminPetCombatStats struct {
	Spirit                   uint32 `json:"spirit"`
	SpiritMax                uint32 `json:"spirit_max"`
	HitPct                   uint32 `json:"hit_pct"`
	DodgePct                 uint32 `json:"dodge_pct"`
	CritRatePct              uint32 `json:"crit_rate_pct"`
	CritDmgPct               uint32 `json:"crit_dmg_pct"`
	PhysicalResistPct        uint32 `json:"physical_resist_pct"`
	ReversePhysicalResistPct uint32 `json:"reverse_physical_resist_pct"`
	SkillResistPct           uint32 `json:"skill_resist_pct"`
	ReverseSkillResistPct    uint32 `json:"reverse_skill_resist_pct"`
	ConfusionResistPct       uint32 `json:"confusion_resist_pct"`
	SleepResistPct           uint32 `json:"sleep_resist_pct"`
	ParalysisResistPct       uint32 `json:"paralysis_resist_pct"`
	SealResistPct            uint32 `json:"seal_resist_pct"`
	CurseResistPct           uint32 `json:"curse_resist_pct"`
	CritDmgResistPct         uint32 `json:"crit_dmg_resist_pct"`
	CritResistPct            uint32 `json:"crit_resist_pct"`
	CharacterResistPct       uint32 `json:"character_resist_pct"`
	PetResistPct             uint32 `json:"pet_resist_pct"`
	Guard                    uint32 `json:"guard"`
	TalentDmgPct             uint32 `json:"talent_dmg_pct"`
	TalentReducePct          uint32 `json:"talent_reduce_pct"`
	ElementAdvPct            uint32 `json:"element_adv_pct"`
	ElementPenaltyPct        uint32 `json:"element_penalty_pct"`
}

// applyAdminCombatCaps 把后台提交的战斗字段限制在封顶表内，并写回 input。
func applyAdminCombatCaps(
	hp, hpMax, atk, def, spd, mana *uint32,
	stats *AdminPetCombatStats,
	caps CombatStatCaps,
) {
	if hp == nil || hpMax == nil || atk == nil || def == nil || spd == nil || mana == nil || stats == nil {
		return
	}
	if caps.values == nil {
		caps = DefaultCombatStatCaps()
	}
	snapshot := &Pet{
		HP:                       *hp,
		HPMax:                    *hpMax,
		ATK:                      *atk,
		DEF:                      *def,
		SPD:                      *spd,
		MANA:                     *mana,
		Spirit:                   stats.Spirit,
		SpiritMax:                stats.SpiritMax,
		HitPct:                   stats.HitPct,
		DodgePct:                 stats.DodgePct,
		CritRatePct:              stats.CritRatePct,
		CritDmgPct:               stats.CritDmgPct,
		PhysicalResistPct:        stats.PhysicalResistPct,
		ReversePhysicalResistPct: stats.ReversePhysicalResistPct,
		SkillResistPct:           stats.SkillResistPct,
		ReverseSkillResistPct:    stats.ReverseSkillResistPct,
		ConfusionResistPct:       stats.ConfusionResistPct,
		SleepResistPct:           stats.SleepResistPct,
		ParalysisResistPct:       stats.ParalysisResistPct,
		SealResistPct:            stats.SealResistPct,
		CurseResistPct:           stats.CurseResistPct,
		CritDmgResistPct:         stats.CritDmgResistPct,
		CritResistPct:            stats.CritResistPct,
		CharacterResistPct:       stats.CharacterResistPct,
		PetResistPct:             stats.PetResistPct,
		Guard:                    stats.Guard,
		TalentDmgPct:             stats.TalentDmgPct,
		TalentReducePct:          stats.TalentReducePct,
		ElementAdvPct:            stats.ElementAdvPct,
		ElementPenaltyPct:        stats.ElementPenaltyPct,
	}
	ClampPetCombatStats(snapshot, caps)
	*hp = snapshot.HP
	*hpMax = snapshot.HPMax
	*atk = snapshot.ATK
	*def = snapshot.DEF
	*spd = snapshot.SPD
	*mana = snapshot.MANA
	stats.Spirit = snapshot.Spirit
	stats.SpiritMax = snapshot.SpiritMax
	stats.HitPct = snapshot.HitPct
	stats.DodgePct = snapshot.DodgePct
	stats.CritRatePct = snapshot.CritRatePct
	stats.CritDmgPct = snapshot.CritDmgPct
	stats.PhysicalResistPct = snapshot.PhysicalResistPct
	stats.ReversePhysicalResistPct = snapshot.ReversePhysicalResistPct
	stats.SkillResistPct = snapshot.SkillResistPct
	stats.ReverseSkillResistPct = snapshot.ReverseSkillResistPct
	stats.ConfusionResistPct = snapshot.ConfusionResistPct
	stats.SleepResistPct = snapshot.SleepResistPct
	stats.ParalysisResistPct = snapshot.ParalysisResistPct
	stats.SealResistPct = snapshot.SealResistPct
	stats.CurseResistPct = snapshot.CurseResistPct
	stats.CritDmgResistPct = snapshot.CritDmgResistPct
	stats.CritResistPct = snapshot.CritResistPct
	stats.CharacterResistPct = snapshot.CharacterResistPct
	stats.PetResistPct = snapshot.PetResistPct
	stats.Guard = snapshot.Guard
	stats.TalentDmgPct = snapshot.TalentDmgPct
	stats.TalentReducePct = snapshot.TalentReducePct
	stats.ElementAdvPct = snapshot.ElementAdvPct
	stats.ElementPenaltyPct = snapshot.ElementPenaltyPct
}

// applyCreateAdminCombatCaps 对创建请求做战斗属性封顶。
func (input AdminCreatePetInput) applyCreateAdminCombatCaps(caps CombatStatCaps) AdminCreatePetInput {
	applyAdminCombatCaps(
		&input.HP,
		&input.HPMax,
		&input.ATK,
		&input.DEF,
		&input.SPD,
		&input.MANA,
		&input.AdminPetCombatStats,
		caps,
	)
	return input
}

// applyUpdateAdminCombatCaps 对更新请求做战斗属性封顶。
func (input AdminUpdatePetInput) applyUpdateAdminCombatCaps(caps CombatStatCaps) AdminUpdatePetInput {
	applyAdminCombatCaps(
		&input.HP,
		&input.HPMax,
		&input.ATK,
		&input.DEF,
		&input.SPD,
		&input.MANA,
		&input.AdminPetCombatStats,
		caps,
	)
	return input
}

// fillAdminCombatStatsFromPet 把领域宠物快照映射到后台战斗属性结构。
func fillAdminCombatStatsFromPet(item Pet) AdminPetCombatStats {
	return AdminPetCombatStats{
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
}
