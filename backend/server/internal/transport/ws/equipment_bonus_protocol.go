package wstransport

import (
	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/protocol"
)

// toProtocolEquipmentBonus 把领域层装备加成映射为客户端消费的协议结构。
func toProtocolEquipmentBonusFromAggregate(bonus equipment.BonusAggregate) protocol.PlayerEquipmentBonusSnapshot {
	return protocol.PlayerEquipmentBonusSnapshot{
		HPMax:                    bonus.HPMax,
		MANA:                     bonus.MANA,
		ATK:                      bonus.ATK,
		DEF:                      bonus.DEF,
		SPD:                      bonus.SPD,
		Spirit:                   bonus.Spirit,
		SpiritMax:                bonus.SpiritMax,
		HitPct:                   bonus.HitPct,
		DodgePct:                 bonus.DodgePct,
		CritRatePct:              bonus.CritRatePct,
		CritDmgPct:               bonus.CritDmgPct,
		PhysicalResistPct:        bonus.PhysicalResistPct,
		ReversePhysicalResistPct: bonus.ReversePhysicalResistPct,
		SkillResistPct:           bonus.SkillResistPct,
		ReverseSkillResistPct:    bonus.ReverseSkillResistPct,
		ConfusionResistPct:       bonus.ConfusionResistPct,
		SleepResistPct:           bonus.SleepResistPct,
		ParalysisResistPct:       bonus.ParalysisResistPct,
		SealResistPct:            bonus.SealResistPct,
		CurseResistPct:           bonus.CurseResistPct,
		CritDmgResistPct:         bonus.CritDmgResistPct,
		CritResistPct:            bonus.CritResistPct,
		CharacterResistPct:       bonus.CharacterResistPct,
		PetResistPct:             bonus.PetResistPct,
	}
}

// toProtocolEquipmentBonusFromRuntimeItem 把背包快照中的属性加成映射为协议结构。
func toProtocolEquipmentBonusFromRuntimeItem(bonus bag.RuntimeItemBonus) protocol.PlayerEquipmentBonusSnapshot {
	return protocol.PlayerEquipmentBonusSnapshot{
		HPMax:                    bonus.HPMax,
		MANA:                     bonus.MANA,
		ATK:                      bonus.ATK,
		DEF:                      bonus.DEF,
		SPD:                      bonus.SPD,
		Spirit:                   bonus.Spirit,
		SpiritMax:                bonus.SpiritMax,
		HitPct:                   bonus.HitPct,
		DodgePct:                 bonus.DodgePct,
		CritRatePct:              bonus.CritRatePct,
		CritDmgPct:               bonus.CritDmgPct,
		PhysicalResistPct:        bonus.PhysicalResistPct,
		ReversePhysicalResistPct: bonus.ReversePhysicalResistPct,
		SkillResistPct:           bonus.SkillResistPct,
		ReverseSkillResistPct:    bonus.ReverseSkillResistPct,
		ConfusionResistPct:       bonus.ConfusionResistPct,
		SleepResistPct:           bonus.SleepResistPct,
		ParalysisResistPct:       bonus.ParalysisResistPct,
		SealResistPct:            bonus.SealResistPct,
		CurseResistPct:           bonus.CurseResistPct,
		CritDmgResistPct:         bonus.CritDmgResistPct,
		CritResistPct:            bonus.CritResistPct,
		CharacterResistPct:       bonus.CharacterResistPct,
		PetResistPct:             bonus.PetResistPct,
	}
}

// toProtocolContainerItemSnapshot 把背包格子快照映射为协议结构，包含描述与装备属性合计。
func toProtocolContainerItemSnapshot(item bag.RuntimeItemSnapshot) protocol.ContainerItemSnapshot {
	return protocol.ContainerItemSnapshot{
		SlotIndex:           item.SlotIndex,
		ItemID:              item.ItemID,
		ItemUID:             item.ItemUID,
		Quantity:            item.Quantity,
		IsBound:             item.IsBound,
		ItemName:            item.ItemName,
		ItemType:            item.ItemType,
		ItemSubType:         item.ItemSubType,
		Quality:             item.Quality,
		Icon:                item.Icon,
		RequiredLevel:       item.RequiredLevel,
		EnhanceLevel:        item.EnhanceLevel,
		IsDamaged:           item.IsDamaged,
		Usable:              item.Usable,
		CanDrop:             item.CanDrop,
		TargetType:          item.TargetType,
		EffectType:          item.EffectType,
		EquipSlot:           item.EquipSlot,
		Description:         item.Description,
		DescriptionMentions: toProtocolDescriptionMentions(item.DescriptionMentions),
		Bonus:               toProtocolEquipmentBonusFromRuntimeItem(item.Bonus),
		EnhancePreview:      toProtocolEnhancePreview(item.EnhancePreview),
		RepairPreview:       toProtocolRepairPreview(item.RepairPreview),
	}
}

func toProtocolEnhancePreview(preview *bag.RuntimeEnhancePreview) *protocol.EnhancePreviewSnapshot {
	if preview == nil {
		return nil
	}
	materials := make([]protocol.EnhanceMaterialOptionSnapshot, 0, len(preview.Materials))
	for _, material := range preview.Materials {
		materials = append(materials, protocol.EnhanceMaterialOptionSnapshot{
			ItemID:                  material.ItemID,
			ItemName:                material.ItemName,
			OwnedQuantity:           material.OwnedQuantity,
			EffectiveSuccessRatePct: material.EffectiveSuccessRatePct,
			FailurePenalty:          material.FailurePenalty,
			FailurePenaltyLabel:     material.FailurePenaltyLabel,
			Description:             material.Description,
		})
	}
	rows := make([]protocol.EnhancePreviewRowSnapshot, 0, len(preview.Rows))
	for _, row := range preview.Rows {
		rows = append(rows, protocol.EnhancePreviewRowSnapshot{
			Label:   row.Label,
			Current: row.Current,
			NextMin: row.NextMin,
			NextMax: row.NextMax,
		})
	}
	return &protocol.EnhancePreviewSnapshot{
		CanEnhance:             preview.CanEnhance,
		MaxEnhanceLevel:        preview.MaxEnhanceLevel,
		SuccessRatePct:         preview.SuccessRatePct,
		RequiredLevel:          preview.RequiredLevel,
		RequiredLevelBandMin:   preview.RequiredLevelBandMin,
		RequiredLevelBandLabel: preview.RequiredLevelBandLabel,
		CostGoldCopper:         preview.CostGoldCopper,
		CostItemID:              preview.CostItemID,
		CostItemName:            preview.CostItemName,
		CostQuantity:            preview.CostQuantity,
		OwnedCostQuantity:       preview.OwnedCostQuantity,
		EnhanceMaterialCategory: preview.EnhanceMaterialCategory,
		Materials:               materials,
		Rows:                    rows,
	}
}

func toProtocolRepairPreview(preview *bag.RuntimeRepairPreview) *protocol.RepairPreviewSnapshot {
	if preview == nil {
		return nil
	}
	return &protocol.RepairPreviewSnapshot{
		CanRepair:         preview.CanRepair,
		CostItemID:        preview.CostItemID,
		CostItemName:      preview.CostItemName,
		CostQuantity:      preview.CostQuantity,
		OwnedCostQuantity: preview.OwnedCostQuantity,
	}
}

func toProtocolDescriptionMentions(mentions []item.DescriptionMention) []protocol.ItemDescriptionMention {
	if len(mentions) == 0 {
		return nil
	}
	result := make([]protocol.ItemDescriptionMention, 0, len(mentions))
	for _, mention := range mentions {
		result = append(result, protocol.ItemDescriptionMention{
			ItemID:   mention.ItemID,
			ItemName: mention.ItemName,
		})
	}
	return result
}
