package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
)

var configuredPlayerNumericColumns = map[string]struct{}{
	"level": {}, "exp": {}, "free_attr_points": {},
	"strength": {}, "vitality": {}, "agility": {}, "mind": {},
	"gold": {},
	"hp": {}, "hp_max": {}, "vigor": {}, "vigor_max": {}, "spirit": {}, "spirit_max": {},
	"atk": {}, "def": {}, "spd": {}, "mana": {},
	"hit_pct": {}, "dodge_pct": {}, "crit_rate_pct": {}, "crit_dmg_pct": {},
	"physical_resist_pct": {}, "skill_resist_pct": {},
	"confusion_resist_pct": {}, "sleep_resist_pct": {}, "paralysis_resist_pct": {},
	"seal_resist_pct": {}, "curse_resist_pct": {}, "crit_resist_pct": {},
	"crit_dmg_resist_pct": {}, "character_resist_pct": {}, "pet_resist_pct": {},
	"mercenary_resist_pct": {}, "generic_shield_pct": {},
	"scene_id": {}, "pos_x": {}, "pos_y": {}, "status": {},
}

var configuredPetNumericColumns = map[string]struct{}{
	"level": {}, "exp": {}, "quality": {},
	"hp": {}, "hp_max": {}, "atk": {}, "def": {}, "spd": {}, "mana": {},
	"spirit": {}, "spirit_max": {},
	"hit_pct": {}, "dodge_pct": {}, "crit_rate_pct": {}, "crit_dmg_pct": {},
	"physical_resist_pct": {}, "reverse_physical_resist_pct": {},
	"skill_resist_pct": {}, "reverse_skill_resist_pct": {},
	"confusion_resist_pct": {}, "sleep_resist_pct": {}, "paralysis_resist_pct": {},
	"seal_resist_pct": {}, "curse_resist_pct": {}, "crit_dmg_resist_pct": {},
	"crit_resist_pct": {}, "character_resist_pct": {}, "pet_resist_pct": {},
	"guard": {}, "talent_dmg_pct": {}, "talent_reduce_pct": {},
	"element_adv_pct": {}, "element_penalty_pct": {},
}

var configuredEquipmentNumericColumns = map[string]struct{}{
	"enhance_level": {},
}

// applyRuntimeConfiguredUseEffects 按 effect_params_json.use_effects 权威执行消耗品效果。
func applyRuntimeConfiguredUseEffects(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	sourceRow *useItemSourceRow,
	quantity uint64,
	targetPetUID uint64,
	targetPlayerID uint64,
	targetItemUID string,
	effects []bag.ConfiguredUseEffect,
) (bag.RuntimeUseEffect, error) {
	if sourceRow == nil || len(effects) == 0 || quantity == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUnsupportedItemEffect
	}
	if targetPlayerID != 0 && targetPlayerID != playerID {
		return bag.RuntimeUseEffect{}, bag.ErrUseTargetNotFound
	}
	if bag.RequiresPetTarget(effects) && targetPetUID == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrUseTargetRequired
	}
	if bag.RequiresEquipmentTarget(effects) && strings.TrimSpace(targetItemUID) == "" {
		return bag.RuntimeUseEffect{}, bag.ErrUseTargetRequired
	}

	result := bag.RuntimeUseEffect{
		EffectType:     "use_effects",
		AppliedEffects: make([]bag.RuntimeAppliedUseEffect, 0, len(effects)),
	}
	appliedCount := 0

	for _, effect := range effects {
		applied, partial, err := applyConfiguredUseEffectEntry(
			ctx,
			tx,
			playerID,
			sourceRow,
			quantity,
			targetPetUID,
			targetPlayerID,
			targetItemUID,
			effect,
			len(effects),
		)
		if err != nil {
			return bag.RuntimeUseEffect{}, err
		}
		if partial != nil {
			mergeConfiguredUseEffectResult(&result, *partial)
		}
		if applied {
			appliedCount++
			result.AppliedEffects = append(result.AppliedEffects, toRuntimeAppliedUseEffect(effect))
		}
	}
	if appliedCount == 0 {
		return bag.RuntimeUseEffect{}, bag.ErrItemUseNoEffect
	}
	if result.UpdatedPet == nil && targetPetUID > 0 && hasPetCategoryEffect(effects) {
		updatedPet, err := buildRuntimePetSnapshotAfterUse(ctx, tx, playerID, targetPetUID)
		if err != nil {
			return bag.RuntimeUseEffect{}, err
		}
		result.UpdatedPet = updatedPet
		result.TargetPetUID = targetPetUID
	}
	return result, nil
}

func applyConfiguredUseEffectEntry(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	sourceRow *useItemSourceRow,
	quantity uint64,
	targetPetUID uint64,
	targetPlayerID uint64,
	targetItemUID string,
	effect bag.ConfiguredUseEffect,
	totalEffectCount int,
) (applied bool, partial *bag.RuntimeUseEffect, err error) {
	switch effect.Category {
	case bag.ConfiguredUseEffectCategoryPlayer:
		if effect.FieldKey == "total_copper" {
			return applyConfiguredPlayerWalletEffect(ctx, tx, playerID, sourceRow, quantity, effect)
		}
		return applyConfiguredPlayerNumericEffect(ctx, tx, playerID, quantity, effect)
	case bag.ConfiguredUseEffectCategoryPet:
		return applyConfiguredPetNumericEffect(ctx, tx, playerID, targetPetUID, quantity, effect, totalEffectCount)
	case bag.ConfiguredUseEffectCategoryEquipment:
		return applyConfiguredEquipmentEffect(ctx, tx, playerID, targetItemUID, effect)
	case bag.ConfiguredUseEffectCategorySystem:
		return applyConfiguredSystemEffect(ctx, tx, playerID, sourceRow, quantity, targetPetUID, targetPlayerID, effect)
	case bag.ConfiguredUseEffectCategoryOther:
		return false, nil, bag.ErrUnsupportedItemEffect
	default:
		return false, nil, bag.ErrUnsupportedItemEffect
	}
}

func applyConfiguredPlayerWalletEffect(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	sourceRow *useItemSourceRow,
	quantity uint64,
	effect bag.ConfiguredUseEffect,
) (bool, *bag.RuntimeUseEffect, error) {
	delta, err := configuredNumericDelta(effect, quantity)
	if err != nil {
		return false, nil, err
	}
	beforeTotal, err := loadRuntimeWalletTotalCopper(ctx, tx, playerID)
	if err != nil {
		return false, nil, err
	}
	var afterTotal uint64
	if err := tx.QueryRowContext(ctx, adjustWalletTotalQuery, playerID, delta).Scan(&afterTotal, new(uint64), new(sql.NullTime), new(sql.NullTime)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, bag.ErrUnsupportedItemEffect
		}
		return false, nil, err
	}
	if _, err := tx.ExecContext(ctx, insertCurrencyChangeLogQuery, playerID, beforeTotal, delta, afterTotal, "item_use", sourceRow.ItemID, "player", playerID); err != nil {
		return false, nil, err
	}
	return true, &bag.RuntimeUseEffect{NeedsWalletPush: true}, nil
}

func applyConfiguredPlayerNumericEffect(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	quantity uint64,
	effect bag.ConfiguredUseEffect,
) (bool, *bag.RuntimeUseEffect, error) {
	if _, ok := configuredPlayerNumericColumns[effect.FieldKey]; !ok {
		return false, nil, bag.ErrUnsupportedItemEffect
	}
	delta, err := configuredNumericDelta(effect, quantity)
	if err != nil {
		return false, nil, err
	}
	query, args := buildConfiguredNumericUpdateQuery("player", effect.FieldKey, effect.Operation, "id = $1 AND status = 1", playerID, delta)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return false, nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, nil, err
	}
	if rowsAffected == 0 {
		return false, nil, bag.ErrUseTargetNotFound
	}
	return true, nil, nil
}

func applyConfiguredPetNumericEffect(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	targetPetUID uint64,
	quantity uint64,
	effect bag.ConfiguredUseEffect,
	totalEffectCount int,
) (bool, *bag.RuntimeUseEffect, error) {
	if _, ok := configuredPetNumericColumns[effect.FieldKey]; !ok {
		return false, nil, bag.ErrUnsupportedItemEffect
	}
	if effect.FieldKey == "hp" && effect.Operation == bag.ConfiguredUseEffectOperationAdd {
		targetPet, err := loadRuntimeUseTargetPetRow(ctx, tx, playerID, targetPetUID)
		if err != nil {
			return false, nil, err
		}
		if targetPet == nil {
			return false, nil, bag.ErrUseTargetNotFound
		}
		if targetPet.HP >= targetPet.HPMax {
			if totalEffectCount == 1 {
				return false, nil, bag.ErrItemUseNoEffect
			}
			return false, nil, nil
		}
	}
	var beforeHP uint32
	if effect.FieldKey == "hp" {
		targetPet, err := loadRuntimeUseTargetPetRow(ctx, tx, playerID, targetPetUID)
		if err != nil {
			return false, nil, err
		}
		if targetPet == nil {
			return false, nil, bag.ErrUseTargetNotFound
		}
		beforeHP = targetPet.HP
	}
	delta, err := configuredNumericDelta(effect, quantity)
	if err != nil {
		return false, nil, err
	}
	query, args := buildConfiguredPetNumericUpdateQuery(effect.FieldKey, effect.Operation, playerID, targetPetUID, delta)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return false, nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, nil, err
	}
	if rowsAffected == 0 {
		return false, nil, bag.ErrUseTargetNotFound
	}
	partial := &bag.RuntimeUseEffect{TargetPetUID: targetPetUID}
	if effect.FieldKey == "hp" {
		updatedPet, err := buildRuntimePetSnapshotAfterUse(ctx, tx, playerID, targetPetUID)
		if err != nil {
			return false, nil, err
		}
		if updatedPet != nil {
			if effect.FieldKey == "hp" {
				if updatedPet.HP > beforeHP {
					partial.RestoredHP = updatedPet.HP - beforeHP
				}
				partial.NewPetHP = updatedPet.HP
			}
			partial.UpdatedPet = updatedPet
		}
	}
	return true, partial, nil
}

func applyConfiguredEquipmentEffect(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	targetItemUID string,
	effect bag.ConfiguredUseEffect,
) (bool, *bag.RuntimeUseEffect, error) {
	targetItemUID = strings.TrimSpace(targetItemUID)
	if targetItemUID == "" {
		return false, nil, bag.ErrUseTargetRequired
	}
	if effect.FieldKey == "max_enhance_level" {
		return false, nil, bag.ErrUnsupportedItemEffect
	}
	if _, ok := configuredEquipmentNumericColumns[effect.FieldKey]; !ok {
		return false, nil, bag.ErrUnsupportedItemEffect
	}
	delta := effect.Value
	if effect.Operation == bag.ConfiguredUseEffectOperationAdd || effect.Operation == bag.ConfiguredUseEffectOperationSubtract {
		if delta == 0 {
			return false, nil, bag.ErrUnsupportedItemEffect
		}
	} else if delta < 0 {
		return false, nil, bag.ErrUnsupportedItemEffect
	}
	query := fmt.Sprintf(`
UPDATE equipment_instance
SET %s = GREATEST(0, CASE
    WHEN $3 = 'add' THEN %s + $4
    WHEN $3 = 'subtract' THEN %s - $4
    ELSE $4
END),
    updated_at = CURRENT_TIMESTAMP
WHERE player_id = $1
  AND item_uid = $2
  AND state IN ('bag', 'warehouse', 'equipped')
`, effect.FieldKey, effect.FieldKey, effect.FieldKey)
	result, err := tx.ExecContext(ctx, query, playerID, targetItemUID, string(effect.Operation), delta)
	if err != nil {
		return false, nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, nil, err
	}
	if rowsAffected == 0 {
		return false, nil, bag.ErrUseTargetNotFound
	}
	return true, nil, nil
}

func applyConfiguredSystemEffect(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	sourceRow *useItemSourceRow,
	quantity uint64,
	targetPetUID uint64,
	targetPlayerID uint64,
	effect bag.ConfiguredUseEffect,
) (bool, *bag.RuntimeUseEffect, error) {
	switch effect.FieldKey {
	case "bag_capacity_expand":
		return applyConfiguredExpandEffect(ctx, tx, playerID, sourceRow, quantity, "bag", effect)
	case "warehouse_capacity_expand":
		return applyConfiguredExpandEffect(ctx, tx, playerID, sourceRow, quantity, "warehouse", effect)
	case "pet_talisman_slot_unlock":
		if !effect.IsBoolean || !effect.BoolValue {
			return false, nil, bag.ErrUnsupportedItemEffect
		}
		partial, err := applyRuntimePetTalismanSlotUnlockEffect(ctx, tx, playerID, sourceRow, targetPetUID, targetPlayerID)
		if err != nil {
			return false, nil, err
		}
		return true, &partial, nil
	default:
		return false, nil, bag.ErrUnsupportedItemEffect
	}
}

func applyConfiguredExpandEffect(
	ctx context.Context,
	tx *sql.Tx,
	playerID uint64,
	sourceRow *useItemSourceRow,
	quantity uint64,
	containerType string,
	effect bag.ConfiguredUseEffect,
) (bool, *bag.RuntimeUseEffect, error) {
	if effect.Operation != bag.ConfiguredUseEffectOperationAdd || effect.Value <= 0 {
		return false, nil, bag.ErrUnsupportedItemEffect
	}
	expandSlotsPerUse := uint32(effect.Value)
	if expandSlotsPerUse == 0 {
		return false, nil, bag.ErrUnsupportedItemEffect
	}
	legacySource := *sourceRow
	legacySource.EffectType = containerType + "_expand"
	if containerType == "bag" {
		legacySource.EffectType = "bag_expand"
	}
	legacySource.ExpandTarget = containerType
	legacySource.ExpandSlots = expandSlotsPerUse
	partial, err := applyRuntimeExpandEffect(ctx, tx, playerID, &legacySource, quantity)
	if err != nil {
		return false, nil, err
	}
	return true, &partial, nil
}

func configuredNumericDelta(effect bag.ConfiguredUseEffect, quantity uint64) (int64, error) {
	if effect.IsBoolean {
		return 0, bag.ErrUnsupportedItemEffect
	}
	if effect.Operation == bag.ConfiguredUseEffectOperationSet {
		return effect.Value, nil
	}
	if effect.Value == 0 || quantity == 0 {
		return 0, bag.ErrUnsupportedItemEffect
	}
	magnitude := effect.Value * int64(quantity)
	if effect.Operation == bag.ConfiguredUseEffectOperationSubtract {
		return -magnitude, nil
	}
	return magnitude, nil
}

func buildConfiguredNumericUpdateQuery(
	tableName string,
	column string,
	operation bag.ConfiguredUseEffectOperation,
	whereClause string,
	playerID uint64,
	delta int64,
) (string, []any) {
	switch operation {
	case bag.ConfiguredUseEffectOperationAdd:
		query := fmt.Sprintf(`
UPDATE %s
SET %s = GREATEST(0, %s + $2),
    updated_at = CURRENT_TIMESTAMP
WHERE %s
`, tableName, column, column, whereClause)
		return query, []any{playerID, delta}
	case bag.ConfiguredUseEffectOperationSubtract:
		query := fmt.Sprintf(`
UPDATE %s
SET %s = GREATEST(0, %s - $2),
    updated_at = CURRENT_TIMESTAMP
WHERE %s
`, tableName, column, column, whereClause)
		return query, []any{playerID, -delta}
	}
	query := fmt.Sprintf(`
UPDATE %s
SET %s = GREATEST(0, $2),
    updated_at = CURRENT_TIMESTAMP
WHERE %s
`, tableName, column, whereClause)
	return query, []any{playerID, delta}
}

func buildConfiguredPetNumericUpdateQuery(
	column string,
	operation bag.ConfiguredUseEffectOperation,
	playerID uint64,
	targetPetUID uint64,
	delta int64,
) (string, []any) {
	if column == "hp" {
		switch operation {
		case bag.ConfiguredUseEffectOperationAdd:
			return `
UPDATE player_pet
SET hp = LEAST(hp_max, GREATEST(0, hp + $3))
WHERE player_id = $1 AND id = $2
`, []any{playerID, targetPetUID, delta}
		case bag.ConfiguredUseEffectOperationSubtract:
			return `
UPDATE player_pet
SET hp = LEAST(hp_max, GREATEST(0, hp - $3))
WHERE player_id = $1 AND id = $2
`, []any{playerID, targetPetUID, -delta}
		default:
			return `
UPDATE player_pet
SET hp = LEAST(hp_max, GREATEST(0, $3))
WHERE player_id = $1 AND id = $2
`, []any{playerID, targetPetUID, delta}
		}
	}
	switch operation {
	case bag.ConfiguredUseEffectOperationAdd:
		query := fmt.Sprintf(`
UPDATE player_pet
SET %s = GREATEST(0, %s + $3)
WHERE player_id = $1 AND id = $2
`, column, column)
		return query, []any{playerID, targetPetUID, delta}
	case bag.ConfiguredUseEffectOperationSubtract:
		query := fmt.Sprintf(`
UPDATE player_pet
SET %s = GREATEST(0, %s - $3)
WHERE player_id = $1 AND id = $2
`, column, column)
		return query, []any{playerID, targetPetUID, -delta}
	}
	query := fmt.Sprintf(`
UPDATE player_pet
SET %s = GREATEST(0, $3)
WHERE player_id = $1 AND id = $2
`, column)
	return query, []any{playerID, targetPetUID, delta}
}

func loadRuntimeWalletTotalCopper(ctx context.Context, tx *sql.Tx, playerID uint64) (uint64, error) {
	var totalCopper uint64
	if err := tx.QueryRowContext(ctx, runtimeWalletQuery, playerID).Scan(&totalCopper); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, bag.ErrUnsupportedItemEffect
		}
		return 0, err
	}
	return totalCopper, nil
}

func buildRuntimePetSnapshotAfterUse(ctx context.Context, tx *sql.Tx, playerID uint64, targetPetUID uint64) (*bag.RuntimePetSnapshot, error) {
	targetPet, err := loadRuntimeUseTargetPetRow(ctx, tx, playerID, targetPetUID)
	if err != nil {
		return nil, err
	}
	if targetPet == nil {
		return nil, bag.ErrUseTargetNotFound
	}
	return &bag.RuntimePetSnapshot{
		PetUID:   targetPet.PetUID,
		PetID:    targetPet.PetID,
		Level:    targetPet.Level,
		Exp:      targetPet.Exp,
		Quality:  targetPet.Quality,
		HP:       targetPet.HP,
		HPMax:    targetPet.HPMax,
		ATK:      targetPet.ATK,
		DEF:      targetPet.DEF,
		SPD:      targetPet.SPD,
		SkillIDs: append([]uint32{}, targetPet.SkillIDs...),
		InLineup: targetPet.InLineup,
	}, nil
}

func mergeConfiguredUseEffectResult(target *bag.RuntimeUseEffect, partial bag.RuntimeUseEffect) {
	if partial.EffectType != "" && partial.EffectType != "use_effects" {
		target.EffectType = partial.EffectType
	}
	if partial.ExpandTarget != "" {
		target.ExpandTarget = partial.ExpandTarget
	}
	if partial.ExpandSlots > 0 {
		target.ExpandSlots = partial.ExpandSlots
	}
	if partial.NewCapacity > 0 {
		target.NewCapacity = partial.NewCapacity
	}
	if partial.TargetPetUID > 0 {
		target.TargetPetUID = partial.TargetPetUID
	}
	if partial.RestoredHP > 0 {
		target.RestoredHP = partial.RestoredHP
	}
	if partial.NewPetHP > 0 {
		target.NewPetHP = partial.NewPetHP
	}
	if partial.UnlockedTalismanSlot != "" {
		target.UnlockedTalismanSlot = partial.UnlockedTalismanSlot
	}
	if partial.UpdatedPet != nil {
		target.UpdatedPet = partial.UpdatedPet
	}
	if partial.NeedsWalletPush {
		target.NeedsWalletPush = true
	}
}

func toRuntimeAppliedUseEffect(effect bag.ConfiguredUseEffect) bag.RuntimeAppliedUseEffect {
	if effect.IsBoolean {
		return bag.RuntimeAppliedUseEffect{
			Category:  string(effect.Category),
			FieldKey:  effect.FieldKey,
			Operation: string(effect.Operation),
			BoolValue: effect.BoolValue,
		}
	}
	return bag.RuntimeAppliedUseEffect{
		Category:  string(effect.Category),
		FieldKey:  effect.FieldKey,
		Operation: string(effect.Operation),
		Value:     effect.Value,
	}
}

func hasPetCategoryEffect(effects []bag.ConfiguredUseEffect) bool {
	for _, effect := range effects {
		if effect.Category == bag.ConfiguredUseEffectCategoryPet {
			return true
		}
	}
	return false
}

func parseConfiguredUseEffectsFromSourceRow(sourceRow *useItemSourceRow) ([]bag.ConfiguredUseEffect, error) {
	if sourceRow == nil {
		return nil, nil
	}
	if strings.TrimSpace(sourceRow.EffectType) == "use_effects" {
		effects, err := bag.ParseConfiguredUseEffects(sourceRow.EffectParams)
		if err != nil {
			return nil, err
		}
		return effects, nil
	}
	effects, err := bag.ParseConfiguredUseEffects(sourceRow.EffectParams)
	if err != nil {
		return nil, err
	}
	return effects, nil
}
