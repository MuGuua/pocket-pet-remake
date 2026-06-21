package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/pet"
)

const upsertPetArtifactEquipmentQuery = `
INSERT INTO pet_artifact_equipment (pet_uid, player_id, slot_index, skill_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (pet_uid, slot_index)
DO UPDATE SET skill_id = EXCLUDED.skill_id, updated_at = CURRENT_TIMESTAMP
`

const deletePetArtifactEquipmentQuery = `
DELETE FROM pet_artifact_equipment
WHERE pet_uid = $1 AND player_id = $2 AND slot_index = $3
`

const updatePlayerPetBattleSkillIDsQuery = `
UPDATE player_pet
SET skill_ids = $3::jsonb
WHERE player_id = $1 AND id = $2
`

type runtimeArtifactItemParams struct {
	SkillID uint32 `json:"skill_id"`
}

// EquipArtifactFromBagSlot 从背包扣除 1 个法宝道具并写入宠物法宝槽。
func (r *PetRepository) EquipArtifactFromBagSlot(ctx context.Context, playerID uint64, petUID uint64, slotIndex uint32, containerType string, bagSlotIndex uint32) (pet.Pet, error) {
	if slotIndex >= pet.MaxArtifactSkillSlots {
		return pet.Pet{}, pet.ErrInvalidArtifactSlot
	}
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return pet.Pet{}, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return pet.Pet{}, err
	}
	defer rollbackTx(tx)

	sourceRow, err := loadUseItemSourceRow(ctx, tx, playerID, containerType, bagSlotIndex)
	if err != nil {
		return pet.Pet{}, err
	}
	if sourceRow == nil {
		return pet.Pet{}, bag.ErrContainerItemNotFound
	}
	skillID, err := resolveArtifactSkillID(sourceRow)
	if err != nil {
		return pet.Pet{}, err
	}
	if _, err := tx.ExecContext(ctx, upsertPetArtifactEquipmentQuery, petUID, playerID, slotIndex, skillID); err != nil {
		return pet.Pet{}, err
	}
	if err := consumeOneBagSlotItem(ctx, tx, playerID, containerType, bagSlotIndex, sourceRow); err != nil {
		return pet.Pet{}, err
	}
	if err := refreshPlayerPetBattleSkillIDsInTx(ctx, tx, playerID, petUID); err != nil {
		return pet.Pet{}, err
	}
	if err := tx.Commit(); err != nil {
		return pet.Pet{}, err
	}
	return r.FindPetByUID(ctx, playerID, petUID)
}

// UnequipArtifact 卸下宠物指定法宝槽技能。
func (r *PetRepository) UnequipArtifact(ctx context.Context, playerID uint64, petUID uint64, slotIndex uint32) (pet.Pet, error) {
	if slotIndex >= pet.MaxArtifactSkillSlots {
		return pet.Pet{}, pet.ErrInvalidArtifactSlot
	}
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return pet.Pet{}, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return pet.Pet{}, err
	}
	defer rollbackTx(tx)

	result, err := tx.ExecContext(ctx, deletePetArtifactEquipmentQuery, petUID, playerID, slotIndex)
	if err != nil {
		return pet.Pet{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pet.Pet{}, err
	}
	if rowsAffected == 0 {
		return pet.Pet{}, pet.ErrArtifactSlotEmpty
	}
	if err := refreshPlayerPetBattleSkillIDsInTx(ctx, tx, playerID, petUID); err != nil {
		return pet.Pet{}, err
	}
	if err := tx.Commit(); err != nil {
		return pet.Pet{}, err
	}
	return r.FindPetByUID(ctx, playerID, petUID)
}

func resolveArtifactSkillID(sourceRow *useItemSourceRow) (uint32, error) {
	if sourceRow == nil {
		return 0, pet.ErrInvalidArtifactItem
	}
	normalizedEffectType := strings.TrimSpace(sourceRow.EffectType)
	if normalizedEffectType != "pet_artifact" {
		return 0, pet.ErrInvalidArtifactItem
	}
	if len(sourceRow.EffectParams) > 0 {
		var params runtimeArtifactItemParams
		if err := json.Unmarshal(sourceRow.EffectParams, &params); err == nil && params.SkillID > 0 {
			return params.SkillID, nil
		}
	}
	if sourceRow.EffectValue > 0 {
		return uint32(sourceRow.EffectValue), nil
	}
	return 0, pet.ErrInvalidArtifactItem
}

func consumeOneBagSlotItem(ctx context.Context, tx *sql.Tx, playerID uint64, containerType string, slotIndex uint32, sourceRow *useItemSourceRow) error {
	if sourceRow.Quantity <= 1 {
		if _, err := tx.ExecContext(ctx, runtimeTransferDeleteItemQuery, sourceRow.RecordID); err != nil {
			return err
		}
		return insertItemChangeLog(ctx, tx, itemChangeLogEntry{
			PlayerID:      playerID,
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ChangeType:    "artifact_equip_remove",
			ItemID:        sourceRow.ItemID,
			ItemUID:       sourceRow.ItemUID,
			BeforeQty:     sourceRow.Quantity,
			ChangeQty:     -1,
			AfterQty:      0,
			ReasonType:    "pet_artifact_equip",
			OperatorType:  "player",
			OperatorID:    playerID,
		})
	}
	if err := updateTransferItemQuantity(ctx, tx, sourceRow.RecordID, sourceRow.Quantity-1); err != nil {
		return err
	}
	return insertItemChangeLog(ctx, tx, itemChangeLogEntry{
		PlayerID:      playerID,
		ContainerType: containerType,
		SlotIndex:     slotIndex,
		ChangeType:    "artifact_equip_reduce",
		ItemID:        sourceRow.ItemID,
		ItemUID:       sourceRow.ItemUID,
		BeforeQty:     sourceRow.Quantity,
		ChangeQty:     -1,
		AfterQty:      sourceRow.Quantity - 1,
		ReasonType:    "pet_artifact_equip",
		OperatorType:  "player",
		OperatorID:    playerID,
	})
}

func refreshPlayerPetBattleSkillIDsInTx(ctx context.Context, tx *sql.Tx, playerID uint64, petUID uint64) error {
	loadout, legacySkillIDs, err := loadPetSkillLoadoutInTx(ctx, tx, playerID, petUID)
	if err != nil {
		return err
	}
	if loadout == nil {
		return pet.ErrPetNotFound
	}
	pet.ApplyLegacySkillIDs(loadout, legacySkillIDs)
	artifactSlots, err := loadPetArtifactSlotsInTx(ctx, tx, playerID, petUID)
	if err != nil {
		return err
	}
	loadout.ArtifactSkillIDs = artifactSlots
	battleSkillIDs, err := json.Marshal(pet.BuildBattleSkillIDs(*loadout))
	if err != nil {
		return fmt.Errorf("marshal refreshed pet battle skill ids: %w", err)
	}
	result, err := tx.ExecContext(ctx, updatePlayerPetBattleSkillIDsQuery, playerID, petUID, battleSkillIDs)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return pet.ErrPetNotFound
	}
	return nil
}

func loadPetArtifactSlotsInTx(ctx context.Context, tx *sql.Tx, playerID uint64, petUID uint64) ([pet.MaxArtifactSkillSlots]uint32, error) {
	var slots [pet.MaxArtifactSkillSlots]uint32
	rows, err := tx.QueryContext(ctx, `
SELECT slot_index, skill_id
FROM pet_artifact_equipment
WHERE player_id = $1 AND pet_uid = $2
ORDER BY slot_index ASC
`, playerID, petUID)
	if err != nil {
		return slots, err
	}
	defer rows.Close()
	for rows.Next() {
		var slotIndex int64
		var skillID int64
		if err := rows.Scan(&slotIndex, &skillID); err != nil {
			return slots, err
		}
		if slotIndex < 0 || slotIndex >= pet.MaxArtifactSkillSlots {
			continue
		}
		slots[slotIndex] = uint32(skillID)
	}
	return slots, rows.Err()
}

func loadPetSkillLoadoutInTx(ctx context.Context, tx *sql.Tx, playerID uint64, petUID uint64) (*pet.SkillLoadout, []uint32, error) {
	var (
		innateSkillIDsJSON []byte
		normalSkillIDsJSON []byte
		skillIDsJSON       []byte
		activeTalismanSkillID int64
		talismanHeroSkillID int64
		talismanSlot1SkillID int64
		talismanSlot2SkillID int64
		talismanSlot3SkillID int64
		activeTalismanEnabled bool
		talismanHeroEnabled bool
		talismanSlot1Enabled bool
		talismanSlot2Enabled bool
		talismanSlot3Enabled bool
	)
	err := tx.QueryRowContext(ctx, `
SELECT
  innate_skill_ids,
  normal_skill_ids,
  skill_ids,
  active_talisman_skill_id,
  talisman_hero_skill_id,
  talisman_slot_1_skill_id,
  talisman_slot_2_skill_id,
  talisman_slot_3_skill_id,
  active_talisman_enabled,
  talisman_hero_enabled,
  talisman_slot_1_enabled,
  talisman_slot_2_enabled,
  talisman_slot_3_enabled
FROM player_pet
WHERE player_id = $1 AND id = $2
LIMIT 1
`, playerID, petUID).Scan(
		&innateSkillIDsJSON,
		&normalSkillIDsJSON,
		&skillIDsJSON,
		&activeTalismanSkillID,
		&talismanHeroSkillID,
		&talismanSlot1SkillID,
		&talismanSlot2SkillID,
		&talismanSlot3SkillID,
		&activeTalismanEnabled,
		&talismanHeroEnabled,
		&talismanSlot1Enabled,
		&talismanSlot2Enabled,
		&talismanSlot3Enabled,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	loadout := decodeSkillLoadout(
		innateSkillIDsJSON,
		normalSkillIDsJSON,
		uint32(activeTalismanSkillID),
		uint32(talismanHeroSkillID),
		uint32(talismanSlot1SkillID),
		uint32(talismanSlot2SkillID),
		uint32(talismanSlot3SkillID),
		activeTalismanEnabled,
		talismanHeroEnabled,
		talismanSlot1Enabled,
		talismanSlot2Enabled,
		talismanSlot3Enabled,
	)
	legacySkillIDs := decodeSkillIDJSONArray(skillIDsJSON)
	return &loadout, legacySkillIDs, nil
}
