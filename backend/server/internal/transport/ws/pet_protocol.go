package wstransport

import (
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/petprogression"
	"pocket-pet-remake/server/internal/protocol"
)

// toProtocolPetDetail 把领域宠物快照转换为协议结构，并附带系统自动分配点。
func toProtocolPetDetail(item pet.Pet) protocol.PetDetail {
	return toProtocolPetDetailWithOptions(item, true)
}

// toProtocolPetDetailForList 列表页不下发法宝技 skill_id，仅保留空槽结构。
func toProtocolPetDetailForList(item pet.Pet) protocol.PetDetail {
	return toProtocolPetDetailWithOptions(item, false)
}

func toProtocolPetDetailWithOptions(item pet.Pet, includeArtifactSkills bool) protocol.PetDetail {
	pet.ResolvePetBattleSkills(&item)
	skills := make([]uint32, 0, len(item.SkillIDs))
	skills = append(skills, item.SkillIDs...)
	slotView := pet.BuildSkillSlotView(item.SkillLoadout, includeArtifactSkills)
	skillSlots := toProtocolPetSkillSlots(slotView)
	autoPoints := petprogression.AutoPointsForState(petprogression.ProgressionState{
		Level:          item.Level,
		EvolutionLevel: item.EvolutionLevel,
		RebirthLevel:   item.RebirthLevel,
	})
	return protocol.PetDetail{
		PetUID:          item.PetUID,
		PetID:           item.PetID,
		Level:           item.Level,
		Exp:             item.Exp,
		Quality:         item.Quality,
		HP:              item.HP,
		HPMax:           item.HPMax,
		ATK:             item.ATK,
		DEF:             item.DEF,
		SPD:             item.SPD,
		MANA:            item.MANA,
		SkillIDs:        skills,
		SkillSlots:      &skillSlots,
		InLineup:        item.InLineup,
		IsUsable:        item.IsUsable,
		ExpToNext:       item.ExpToNext,
		FreeAttrPoints:  item.FreeAttrPoints,
		AllocHPPoints:   item.AllocHPPoints,
		AllocATKPoints:  item.AllocATKPoints,
		AllocSPDPoints:  item.AllocSPDPoints,
		AllocMANAPoints: item.AllocMANAPoints,
		AllocDEFPoints:  item.AllocDEFPoints,
		BaseHPApt:       item.BaseHPApt,
		BaseATKApt:      item.BaseATKApt,
		BaseDEFApt:      item.BaseDEFApt,
		BaseSPDApt:      item.BaseSPDApt,
		BaseMANAApt:     item.BaseMANAApt,
		ExtraHPApt:      item.ExtraHPApt,
		ExtraATKApt:     item.ExtraATKApt,
		ExtraDEFApt:     item.ExtraDEFApt,
		ExtraSPDApt:     item.ExtraSPDApt,
		ExtraMANAApt:    item.ExtraMANAApt,
		GrowthAptitudes: protocol.PetGrowthAptitudes{
			HPApt:   item.GrowthAptitudes.HPApt,
			ATKApt:  item.GrowthAptitudes.ATKApt,
			DEFApt:  item.GrowthAptitudes.DEFApt,
			SPDApt:  item.GrowthAptitudes.SPDApt,
			MANAApt: item.GrowthAptitudes.MANAApt,
		},
		AutoHPPoints:   autoPoints.HP,
		AutoATKPoints:  autoPoints.ATK,
		AutoSPDPoints:  autoPoints.SPD,
		AutoMANAPoints: autoPoints.MANA,
		AutoDEFPoints:  autoPoints.DEF,
		Spirit:         item.Spirit,
		SpiritMax:      item.SpiritMax,
		HitPct:         item.HitPct,
		DodgePct:       item.DodgePct,
		CritRatePct:    item.CritRatePct,
		CritDmgPct:     item.CritDmgPct,
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
	}
}

func toProtocolPetSkillSlots(view pet.SkillSlotView) protocol.PetSkillSlots {
	return protocol.PetSkillSlots{
		Innate:         toProtocolSkillSlotEntries(view.Innate),
		ActiveTalisman: toProtocolSkillSlotEntry(view.ActiveTalisman),
		TalismanHero:   toProtocolSkillSlotEntry(view.TalismanHero),
		Talisman1:      toProtocolSkillSlotEntry(view.Talisman1),
		Talisman2:      toProtocolSkillSlotEntry(view.Talisman2),
		Talisman3:      toProtocolSkillSlotEntry(view.Talisman3),
		Normal:         toProtocolSkillSlotEntries(view.Normal),
		Artifact:       toProtocolSkillSlotEntries(view.Artifact),
	}
}

func toProtocolSkillSlotEntries(entries []pet.SkillSlotEntry) []protocol.PetSkillSlotEntry {
	if len(entries) == 0 {
		return []protocol.PetSkillSlotEntry{}
	}
	result := make([]protocol.PetSkillSlotEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, toProtocolSkillSlotEntry(entry))
	}
	return result
}

func toProtocolSkillSlotEntry(entry pet.SkillSlotEntry) protocol.PetSkillSlotEntry {
	return protocol.PetSkillSlotEntry{
		SlotIndex: entry.SlotIndex,
		SkillID:   entry.SkillID,
		Enabled:   entry.Enabled,
	}
}
