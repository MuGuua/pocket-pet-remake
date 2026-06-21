package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pocket-pet-remake/server/internal/module/progression"
)

// ProgressionRepository 负责玩家成长配置与成长状态持久化。
type ProgressionRepository struct {
	db DBTX
}

// NewProgressionRepository 构造玩家成长仓储。
func NewProgressionRepository(db DBTX) *ProgressionRepository {
	return &ProgressionRepository{db: db}
}

const listPlayerLevelConfigsQuery = `
SELECT level, exp_required, attr_points, bonus_atk, bonus_hp_max, bonus_spd, bonus_mana, status
FROM player_level_config
ORDER BY level ASC
`

const listPlayerAttrConvertConfigsQuery = `
SELECT id, source_attr, target_attr, convert_rate, status
FROM player_attr_convert_config
ORDER BY id ASC
`

const upsertPlayerLevelConfigQuery = `
INSERT INTO player_level_config (
  level,
  exp_required,
  attr_points,
  bonus_atk,
  bonus_hp_max,
  bonus_spd,
  bonus_mana,
  status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (level) DO UPDATE SET
  exp_required = EXCLUDED.exp_required,
  attr_points = EXCLUDED.attr_points,
  bonus_atk = EXCLUDED.bonus_atk,
  bonus_hp_max = EXCLUDED.bonus_hp_max,
  bonus_spd = EXCLUDED.bonus_spd,
  bonus_mana = EXCLUDED.bonus_mana,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP
RETURNING level, exp_required, attr_points, bonus_atk, bonus_hp_max, bonus_spd, bonus_mana, status
`

const upsertPlayerAttrConvertConfigQuery = `
UPDATE player_attr_convert_config
SET source_attr = $2,
    target_attr = $3,
    convert_rate = $4,
    status = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, source_attr, target_attr, convert_rate, status
`

const loadPlayerProgressionStateQuery = `
SELECT
  level,
  exp,
  free_attr_points,
  strength,
  vitality,
  agility,
  mind,
  base_hp_max,
  base_atk,
  base_def,
  base_spd,
  base_mana,
  base_hit_pct,
  base_dodge_pct
FROM player
WHERE id = $1 AND status = 1
LIMIT 1
`

const savePlayerExpProgressionQuery = `
UPDATE player
SET level = $2,
    exp = $3,
    free_attr_points = $4,
    base_hp_max = $5,
    base_atk = $6,
    base_spd = $7,
    base_mana = $8,
    hp_max = $9,
    atk = $10,
    def = $11,
    spd = $12,
    mana = $13,
    hit_pct = $14,
    dodge_pct = $15,
    hp = CASE WHEN $16 > 0 THEN $9 ELSE hp END,
    vigor = CASE WHEN $16 > 0 THEN vigor_max ELSE vigor END
WHERE id = $1 AND status = 1
`

const savePlayerAttrAllocationQuery = `
UPDATE player
SET free_attr_points = $2,
    strength = $3,
    vitality = $4,
    agility = $5,
    mind = $6,
    hp_max = $7,
    atk = $8,
    def = $9,
    spd = $10,
    mana = $11,
    hit_pct = $12,
    dodge_pct = $13
WHERE id = $1 AND status = 1
`

const insertPlayerAttrAllocateLogQuery = `
INSERT INTO player_attr_allocate_log (
  player_id,
  delta_strength,
  delta_vitality,
  delta_agility,
  delta_mind,
  free_before,
  free_after,
  reason_type,
  operator_type,
  operator_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`

func (r *ProgressionRepository) ListLevelConfigs(ctx context.Context) ([]progression.LevelConfig, error) {
	rows, err := r.db.QueryContext(ctx, listPlayerLevelConfigsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]progression.LevelConfig, 0, 100)
	for rows.Next() {
		var (
			level, attrPoints, status                      int64
			bonusATK, bonusHPMax, bonusSPD, bonusMANA      int64
			expRequired                                    int64
			item                                           progression.LevelConfig
		)
		if err := rows.Scan(
			&level,
			&expRequired,
			&attrPoints,
			&bonusATK,
			&bonusHPMax,
			&bonusSPD,
			&bonusMANA,
			&status,
		); err != nil {
			return nil, err
		}
		item.Level = uint32(level)
		item.ExpRequired = uint64(expRequired)
		item.AttrPoints = uint32(attrPoints)
		item.BonusATK = uint32(bonusATK)
		item.BonusHPMax = uint32(bonusHPMax)
		item.BonusSPD = uint32(bonusSPD)
		item.BonusMANA = uint32(bonusMANA)
		item.Status = uint32(status)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ProgressionRepository) ListAttrConvertConfigs(ctx context.Context) ([]progression.AttrConvertConfig, error) {
	rows, err := r.db.QueryContext(ctx, listPlayerAttrConvertConfigsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]progression.AttrConvertConfig, 0, 16)
	for rows.Next() {
		var (
			id, convertRate, status int64
			sourceAttr, targetAttr  string
			item                    progression.AttrConvertConfig
		)
		if err := rows.Scan(&id, &sourceAttr, &targetAttr, &convertRate, &status); err != nil {
			return nil, err
		}
		item.ID = uint64(id)
		item.SourceAttr = sourceAttr
		item.TargetAttr = targetAttr
		item.ConvertRate = uint32(convertRate)
		item.Status = uint32(status)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ProgressionRepository) GetLevelConfig(ctx context.Context, level uint32) (*progression.LevelConfig, error) {
	rows, err := r.ListLevelConfigs(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		if item.Level == level {
			copied := item
			return &copied, nil
		}
	}
	return nil, progression.ErrLevelConfigNotFound
}

func (r *ProgressionRepository) UpsertLevelConfig(ctx context.Context, level uint32, input progression.AdminUpsertLevelConfigInput) (*progression.LevelConfig, error) {
	var (
		resultLevel, attrPoints, status             int64
		bonusATK, bonusHPMax, bonusSPD, bonusMANA int64
		expRequired                                 int64
		item                                        progression.LevelConfig
	)
	err := r.db.QueryRowContext(
		ctx,
		upsertPlayerLevelConfigQuery,
		level,
		input.ExpRequired,
		input.AttrPoints,
		input.BonusATK,
		input.BonusHPMax,
		input.BonusSPD,
		input.BonusMANA,
		input.Status,
	).Scan(
		&resultLevel,
		&expRequired,
		&attrPoints,
		&bonusATK,
		&bonusHPMax,
		&bonusSPD,
		&bonusMANA,
		&status,
	)
	if err != nil {
		return nil, err
	}
	item.Level = uint32(resultLevel)
	item.ExpRequired = uint64(expRequired)
	item.AttrPoints = uint32(attrPoints)
	item.BonusATK = uint32(bonusATK)
	item.BonusHPMax = uint32(bonusHPMax)
	item.BonusSPD = uint32(bonusSPD)
	item.BonusMANA = uint32(bonusMANA)
	item.Status = uint32(status)
	return &item, nil
}

func (r *ProgressionRepository) UpsertAttrConvertConfig(ctx context.Context, id uint64, input progression.AdminUpsertAttrConvertInput) (*progression.AttrConvertConfig, error) {
	var (
		resultID, convertRate, status int64
		sourceAttr, targetAttr        string
		item                          progression.AttrConvertConfig
	)
	err := r.db.QueryRowContext(
		ctx,
		upsertPlayerAttrConvertConfigQuery,
		id,
		input.SourceAttr,
		input.TargetAttr,
		input.ConvertRate,
		input.Status,
	).Scan(&resultID, &sourceAttr, &targetAttr, &convertRate, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, progression.ErrConvertConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	item.ID = uint64(resultID)
	item.SourceAttr = sourceAttr
	item.TargetAttr = targetAttr
	item.ConvertRate = uint32(convertRate)
	item.Status = uint32(status)
	return &item, nil
}

func (r *ProgressionRepository) LoadProgressionState(ctx context.Context, playerID uint64) (*progression.ProgressionState, error) {
	var (
		level, freeAttrPoints                         int64
		strength, vitality, agility, mind             int64
		exp                                           int64
		baseHPMax, baseATK, baseDEF, baseSPD          int64
		baseMANA, baseHitPct, baseDodgePct            int64
		state                                         progression.ProgressionState
	)
	err := r.db.QueryRowContext(ctx, loadPlayerProgressionStateQuery, playerID).Scan(
		&level,
		&exp,
		&freeAttrPoints,
		&strength,
		&vitality,
		&agility,
		&mind,
		&baseHPMax,
		&baseATK,
		&baseDEF,
		&baseSPD,
		&baseMANA,
		&baseHitPct,
		&baseDodgePct,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("player progression state not found")
	}
	if err != nil {
		return nil, err
	}
	state.Level = uint32(level)
	state.Exp = uint64(exp)
	state.FreeAttrPoints = uint32(freeAttrPoints)
	state.Allocated = progression.AllocatedAttrs{
		Strength: uint32(strength),
		Vitality: uint32(vitality),
		Agility:  uint32(agility),
		Mind:     uint32(mind),
	}
	state.BaseCombat = progression.BaseCombatStats{
		HPMax:    uint32(baseHPMax),
		ATK:      uint32(baseATK),
		DEF:      uint32(baseDEF),
		SPD:      uint32(baseSPD),
		MANA:     uint32(baseMANA),
		HitPct:   uint32(baseHitPct),
		DodgePct: uint32(baseDodgePct),
	}
	return &state, nil
}

// SaveExpProgression 持久化经验结算结果；若本次发生升级，则同步补满 hp 与 vigor。
func (r *ProgressionRepository) SaveExpProgression(
	ctx context.Context,
	playerID uint64,
	result progression.ExpApplyResult,
	baseCombat progression.BaseCombatStats,
	combatBonus progression.CombatBonus,
) error {
	finalHPMax := progression.FinalCombatValue(baseCombat.HPMax, combatBonus.HPMax)
	finalATK := progression.FinalCombatValue(baseCombat.ATK, combatBonus.ATK)
	finalDEF := progression.FinalCombatValue(baseCombat.DEF, combatBonus.DEF)
	finalSPD := progression.FinalCombatValue(baseCombat.SPD, combatBonus.SPD)
	finalMANA := progression.FinalCombatValue(baseCombat.MANA, combatBonus.MANA)
	finalHitPct := progression.FinalCombatValue(baseCombat.HitPct, combatBonus.HitPct)
	finalDodgePct := progression.FinalCombatValue(baseCombat.DodgePct, combatBonus.DodgePct)

	execResult, err := r.db.ExecContext(
		ctx,
		savePlayerExpProgressionQuery,
		playerID,
		result.Level,
		result.Exp,
		result.FreeAttrPoints,
		baseCombat.HPMax,
		baseCombat.ATK,
		baseCombat.SPD,
		baseCombat.MANA,
		finalHPMax,
		finalATK,
		finalDEF,
		finalSPD,
		finalMANA,
		finalHitPct,
		finalDodgePct,
		result.LevelUpCount,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := execResult.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("player progression state not found")
	}
	return nil
}

func (r *ProgressionRepository) SaveAttrAllocation(
	ctx context.Context,
	playerID uint64,
	delta progression.AttrAllocationDelta,
	freeBefore uint32,
	freeAfter uint32,
	combatBonus progression.CombatBonus,
) error {
	baseState, err := r.LoadProgressionState(ctx, playerID)
	if err != nil {
		return err
	}
	nextAllocated := progression.AllocatedAttrs{
		Strength: baseState.Allocated.Strength + delta.Strength,
		Vitality: baseState.Allocated.Vitality + delta.Vitality,
		Agility:  baseState.Allocated.Agility + delta.Agility,
		Mind:     baseState.Allocated.Mind + delta.Mind,
	}
	finalHPMax := progression.FinalCombatValue(baseState.BaseCombat.HPMax, combatBonus.HPMax)
	finalATK := progression.FinalCombatValue(baseState.BaseCombat.ATK, combatBonus.ATK)
	finalDEF := progression.FinalCombatValue(baseState.BaseCombat.DEF, combatBonus.DEF)
	finalSPD := progression.FinalCombatValue(baseState.BaseCombat.SPD, combatBonus.SPD)
	finalMANA := progression.FinalCombatValue(baseState.BaseCombat.MANA, combatBonus.MANA)
	finalHitPct := progression.FinalCombatValue(baseState.BaseCombat.HitPct, combatBonus.HitPct)
	finalDodgePct := progression.FinalCombatValue(baseState.BaseCombat.DodgePct, combatBonus.DodgePct)

	tx, err := r.beginTx(ctx)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	execResult, err := tx.ExecContext(
		ctx,
		savePlayerAttrAllocationQuery,
		playerID,
		freeAfter,
		nextAllocated.Strength,
		nextAllocated.Vitality,
		nextAllocated.Agility,
		nextAllocated.Mind,
		finalHPMax,
		finalATK,
		finalDEF,
		finalSPD,
		finalMANA,
		finalHitPct,
		finalDodgePct,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := execResult.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("player progression state not found")
	}
	_, err = tx.ExecContext(
		ctx,
		insertPlayerAttrAllocateLogQuery,
		playerID,
		delta.Strength,
		delta.Vitality,
		delta.Agility,
		delta.Mind,
		freeBefore,
		freeAfter,
		"manual_allocate",
		"player",
		playerID,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ProgressionRepository) beginTx(ctx context.Context) (*sql.Tx, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	return beginner.BeginTx(ctx, nil)
}
