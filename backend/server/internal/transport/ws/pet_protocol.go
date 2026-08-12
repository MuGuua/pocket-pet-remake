package wstransport

import (
	"strings"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/petprogression"
	"pocket-pet-remake/server/internal/protocol"
)

// toProtocolPetDetail 把领域宠物快照转换为协议结构，并附带系统自动分配点。
func toProtocolPetDetail(item pet.Pet) protocol.PetDetail {
	return toProtocolPetDetailWithOptions(item, true)
}

// toProtocolPetDetailForList 转换宠物列表与主界面宠物 HUD 共用的轻量字段。
// 经验、生命和法力直接来自服务端权威宠物快照；完整属性、资质、技能和法宝槽仍按单只宠物拉取。
func toProtocolPetDetailForList(item pet.Pet) protocol.PetDetail {
	return protocol.PetDetail{
		PetUID:     item.PetUID,
		PetID:      item.PetID,
		PetName:    strings.TrimSpace(item.PetName),
		CustomName: strings.TrimSpace(item.CustomName),
		Name:       resolveProtocolPetDisplayName(item),
		SkinID:     strings.TrimSpace(item.SkinID),
		Level:      item.Level,
		Exp:        item.Exp,
		Quality:    item.Quality,
		HP:         item.HP,
		HPMax:      item.HPMax,
		MANA:       item.MANA,
		SkillIDs:   []uint32{},
		InLineup:   item.InLineup,
		IsUsable:   item.IsUsable,
		ExpToNext:  item.ExpToNext,
	}
}

func toProtocolPetDetailWithOptions(item pet.Pet, includeArtifactSkills bool) protocol.PetDetail {
	pet.ResolvePetBattleSkills(&item)
	skills := make([]uint32, 0, len(item.SkillIDs))
	skills = append(skills, item.SkillIDs...)
	slotView := pet.BuildSkillSlotView(item.SkillLoadout, includeArtifactSkills)
	skillSlots := toProtocolPetSkillSlots(slotView, item.SkillMetadata)
	autoPoints := petprogression.AutoPointsForState(petprogression.ProgressionState{
		Level:          item.Level,
		EvolutionLevel: item.EvolutionLevel,
		RebirthLevel:   item.RebirthLevel,
	})
	return protocol.PetDetail{
		PetUID:          item.PetUID,
		PetID:           item.PetID,
		PetName:         strings.TrimSpace(item.PetName),
		CustomName:      strings.TrimSpace(item.CustomName),
		Name:            resolveProtocolPetDisplayName(item),
		SkinID:          strings.TrimSpace(item.SkinID),
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
		AutoHPPoints:             autoPoints.HP,
		AutoATKPoints:            autoPoints.ATK,
		AutoSPDPoints:            autoPoints.SPD,
		AutoMANAPoints:           autoPoints.MANA,
		AutoDEFPoints:            autoPoints.DEF,
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
	}
}

// resolveProtocolPetDisplayName 统一给旧客户端提供最终展示名，避免前端自行猜测字段含义。
func resolveProtocolPetDisplayName(item pet.Pet) string {
	if customName := strings.TrimSpace(item.CustomName); customName != "" {
		return customName
	}
	return strings.TrimSpace(item.PetName)
}

func toProtocolPetSkillSlots(view pet.SkillSlotView, metadata map[uint32]pet.SkillMetadata) protocol.PetSkillSlots {
	return protocol.PetSkillSlots{
		Innate:         toProtocolSkillSlotEntries(view.Innate, metadata),
		ActiveTalisman: toProtocolSkillSlotEntry(view.ActiveTalisman, metadata),
		TalismanHero:   toProtocolSkillSlotEntry(view.TalismanHero, metadata),
		Talisman1:      toProtocolSkillSlotEntry(view.Talisman1, metadata),
		Talisman2:      toProtocolSkillSlotEntry(view.Talisman2, metadata),
		Talisman3:      toProtocolSkillSlotEntry(view.Talisman3, metadata),
		Normal:         toProtocolSkillSlotEntries(view.Normal, metadata),
		Artifact:       toProtocolSkillSlotEntries(view.Artifact, metadata),
	}
}

func toProtocolSkillSlotEntries(entries []pet.SkillSlotEntry, metadata map[uint32]pet.SkillMetadata) []protocol.PetSkillSlotEntry {
	if len(entries) == 0 {
		return []protocol.PetSkillSlotEntry{}
	}
	result := make([]protocol.PetSkillSlotEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, toProtocolSkillSlotEntry(entry, metadata))
	}
	return result
}

func toProtocolSkillSlotEntry(entry pet.SkillSlotEntry, metadata map[uint32]pet.SkillMetadata) protocol.PetSkillSlotEntry {
	skillMetadata := metadata[entry.SkillID]
	return protocol.PetSkillSlotEntry{
		SlotIndex:     entry.SlotIndex,
		SkillID:       entry.SkillID,
		Enabled:       entry.Enabled,
		SkillName:     strings.TrimSpace(skillMetadata.SkillName),
		Description:   strings.TrimSpace(skillMetadata.Description),
		SkillVisualID: strings.TrimSpace(skillMetadata.SkillVisualID),
		SkillQuality:  strings.TrimSpace(skillMetadata.SkillQuality),
	}
}
