package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pocket-pet-remake/server/internal/module/petprogression"
)

// PetProgressionRepository 负责宠物成长配置与实例成长状态持久化。
type PetProgressionRepository struct {
	db DBTX
}

// NewPetProgressionRepository 构造宠物成长仓储。
func NewPetProgressionRepository(db DBTX) *PetProgressionRepository {
	return &PetProgressionRepository{db: db}
}

const listPetLevelConfigsQuery = `
SELECT level, exp_required, attr_points, status
FROM pet_level_config
ORDER BY level ASC
`

const listPetConvertConfigsQuery = `
SELECT attr_type, convert_rate, status
FROM pet_attr_convert_config
ORDER BY attr_type ASC
`

const loadPetProgressionStateQuery = `
SELECT
  pp.id,
  pp.pet_id,
  pp.level,
  pp.exp,
  pp.free_attr_points,
  pp.alloc_hp_points,
  pp.alloc_atk_points,
  pp.alloc_spd_points,
  pp.alloc_mana_points,
  pp.alloc_def_points,
  pp.evolution_level,
  pp.rebirth_level,
  pp.base_hp_apt,
  pp.base_atk_apt,
  pp.base_def_apt,
  pp.base_spd_apt,
  pp.base_mana_apt,
  pp.extra_hp_apt,
  pp.extra_atk_apt,
  pp.extra_def_apt,
  pp.extra_spd_apt,
  pp.extra_mana_apt,
  pp.hp,
  COALESCE(pd.aptitude_profile, 'normal') AS aptitude_profile
FROM player_pet pp
JOIN pet_definition pd ON pd.pet_id = pp.pet_id
WHERE pp.player_id = $1 AND pp.id = $2
LIMIT 1
`

const savePetExpProgressionQuery = `
UPDATE player_pet
SET level = $3,
    exp = $4,
    free_attr_points = $5,
    hp_max = $6,
    atk = $7,
    def = $8,
    spd = $9,
    mana = $10,
    hp = LEAST($11, $6)
WHERE player_id = $1 AND id = $2
`

const savePetAttrAllocationQuery = `
UPDATE player_pet
SET free_attr_points = $3,
    alloc_hp_points = alloc_hp_points + $4,
    alloc_atk_points = alloc_atk_points + $5,
    alloc_spd_points = alloc_spd_points + $6,
    alloc_mana_points = alloc_mana_points + $7,
    alloc_def_points = alloc_def_points + $8,
    hp_max = $9,
    atk = $10,
    def = $11,
    spd = $12,
    mana = $13,
    hp = LEAST($14, $9)
WHERE player_id = $1 AND id = $2
`

const insertPetAttrAllocateLogQuery = `
INSERT INTO pet_attr_allocate_log (
  pet_uid,
  player_id,
  delta_hp,
  delta_atk,
  delta_spd,
  delta_mana,
  delta_def,
  free_before,
  free_after,
  reason_type
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'manual_allocate')
`

// ListLevelConfigs 读取全部宠物等级配置。
func (r *PetProgressionRepository) ListLevelConfigs(ctx context.Context) ([]petprogression.LevelConfig, error) {
	rows, err := r.db.QueryContext(ctx, listPetLevelConfigsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]petprogression.LevelConfig, 0, 100)
	for rows.Next() {
		var (
			item       petprogression.LevelConfig
			level      int64
			expRequired int64
			attrPoints int64
			status     int64
		)
		if err := rows.Scan(&level, &expRequired, &attrPoints, &status); err != nil {
			return nil, err
		}
		item.Level = uint32(level)
		item.ExpRequired = uint64(expRequired)
		item.AttrPoints = uint32(attrPoints)
		item.Status = uint32(status)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ListConvertConfigs 读取全部宠物属性转化率配置。
func (r *PetProgressionRepository) ListConvertConfigs(ctx context.Context) ([]petprogression.ConvertConfig, error) {
	rows, err := r.db.QueryContext(ctx, listPetConvertConfigsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]petprogression.ConvertConfig, 0, 8)
	for rows.Next() {
		var (
			item        petprogression.ConvertConfig
			convertRate float64
			status      int64
		)
		if err := rows.Scan(&item.AttrType, &convertRate, &status); err != nil {
			return nil, err
		}
		item.ConvertRate = convertRate
		item.Status = uint32(status)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// LoadProgressionState 加载单只宠物的成长快照。
func (r *PetProgressionRepository) LoadProgressionState(ctx context.Context, playerID uint64, petUID uint64) (*petprogression.ProgressionState, error) {
	var (
		state           petprogression.ProgressionState
		petID           int64
		level           int64
		exp             int64
		freeAttrPoints  int64
		allocHP         int64
		allocATK        int64
		allocSPD        int64
		allocMANA       int64
		allocDEF        int64
		evolutionLevel  int64
		rebirthLevel    int64
		baseHPApt       int64
		baseATKApt      int64
		baseDEFApt      int64
		baseSPDApt      int64
		baseMANAApt     int64
		extraHPApt      int64
		extraATKApt     int64
		extraDEFApt     int64
		extraSPDApt     int64
		extraMANAApt    int64
		hp              int64
	)
	err := r.db.QueryRowContext(ctx, loadPetProgressionStateQuery, playerID, petUID).Scan(
		&state.PetUID,
		&petID,
		&level,
		&exp,
		&freeAttrPoints,
		&allocHP,
		&allocATK,
		&allocSPD,
		&allocMANA,
		&allocDEF,
		&evolutionLevel,
		&rebirthLevel,
		&baseHPApt,
		&baseATKApt,
		&baseDEFApt,
		&baseSPDApt,
		&baseMANAApt,
		&extraHPApt,
		&extraATKApt,
		&extraDEFApt,
		&extraSPDApt,
		&extraMANAApt,
		&hp,
		&state.AptitudeProfile,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state.PlayerID = playerID
	state.PetID = uint32(petID)
	state.Level = uint32(level)
	state.Exp = uint64(exp)
	state.FreeAttrPoints = uint32(freeAttrPoints)
	state.ManualPoints = petprogression.ManualAllocatedPoints{
		HP:   uint32(allocHP),
		ATK:  uint32(allocATK),
		SPD:  uint32(allocSPD),
		MANA: uint32(allocMANA),
		DEF:  uint32(allocDEF),
	}
	state.EvolutionLevel = uint32(evolutionLevel)
	state.RebirthLevel = uint32(rebirthLevel)
	state.Aptitudes = petprogression.GrowthAptitudes{
		BaseHPApt:    uint32(baseHPApt),
		BaseATKApt:   uint32(baseATKApt),
		BaseDEFApt:   uint32(baseDEFApt),
		BaseSPDApt:   uint32(baseSPDApt),
		BaseMANAApt:  uint32(baseMANAApt),
		ExtraHPApt:   uint32(extraHPApt),
		ExtraATKApt:  uint32(extraATKApt),
		ExtraDEFApt:  uint32(extraDEFApt),
		ExtraSPDApt:  uint32(extraSPDApt),
		ExtraMANAApt: uint32(extraMANAApt),
	}
	state.HP = uint32(hp)
	return &state, nil
}

// SaveExpProgression 持久化经验结算后的等级、经验、自由点与战斗属性。
func (r *PetProgressionRepository) SaveExpProgression(
	ctx context.Context,
	playerID uint64,
	petUID uint64,
	result petprogression.ExpApplyResult,
	combat petprogression.CombatStats,
	hp uint32,
) error {
	execResult, err := r.db.ExecContext(
		ctx,
		savePetExpProgressionQuery,
		playerID,
		petUID,
		result.Level,
		result.Exp,
		result.FreeAttrPoints,
		combat.HPMax,
		combat.ATK,
		combat.DEF,
		combat.SPD,
		combat.MANA,
		hp,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := execResult.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return petprogression.ErrPetProgressionNotFound
	}
	return nil
}

// SaveAttrAllocation 持久化主动加点并写入审计日志。
func (r *PetProgressionRepository) SaveAttrAllocation(
	ctx context.Context,
	playerID uint64,
	petUID uint64,
	delta petprogression.ManualAllocatedPoints,
	freeBefore uint32,
	freeAfter uint32,
	combat petprogression.CombatStats,
) error {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	state, err := r.LoadProgressionState(ctx, playerID, petUID)
	if err != nil {
		return err
	}
	hp := state.HP
	if hp == 0 || hp > combat.HPMax {
		hp = combat.HPMax
	}

	result, err := tx.ExecContext(
		ctx,
		savePetAttrAllocationQuery,
		playerID,
		petUID,
		freeAfter,
		delta.HP,
		delta.ATK,
		delta.SPD,
		delta.MANA,
		delta.DEF,
		combat.HPMax,
		combat.ATK,
		combat.DEF,
		combat.SPD,
		combat.MANA,
		hp,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return petprogression.ErrPetProgressionNotFound
	}
	if _, err := tx.ExecContext(
		ctx,
		insertPetAttrAllocateLogQuery,
		petUID,
		playerID,
		delta.HP,
		delta.ATK,
		delta.SPD,
		delta.MANA,
		delta.DEF,
		freeBefore,
		freeAfter,
	); err != nil {
		return err
	}
	return tx.Commit()
}

const upsertPetLevelConfigQuery = `
INSERT INTO pet_level_config (level, exp_required, attr_points, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (level) DO UPDATE SET
  exp_required = EXCLUDED.exp_required,
  attr_points = EXCLUDED.attr_points,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP
RETURNING level, exp_required, attr_points, status
`

const upsertPetConvertConfigQuery = `
UPDATE pet_attr_convert_config
SET convert_rate = $2,
    status = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE attr_type = $1
RETURNING attr_type, convert_rate, status
`

// UpsertLevelConfig 更新或插入单条宠物等级配置。
func (r *PetProgressionRepository) UpsertLevelConfig(ctx context.Context, level uint32, input petprogression.AdminUpsertLevelConfigInput) (*petprogression.LevelConfig, error) {
	var (
		item        petprogression.LevelConfig
		levelValue  int64
		expRequired int64
		attrPoints  int64
		status      int64
	)
	err := r.db.QueryRowContext(
		ctx,
		upsertPetLevelConfigQuery,
		level,
		input.ExpRequired,
		input.AttrPoints,
		input.Status,
	).Scan(&levelValue, &expRequired, &attrPoints, &status)
	if err != nil {
		return nil, err
	}
	item.Level = uint32(levelValue)
	item.ExpRequired = uint64(expRequired)
	item.AttrPoints = uint32(attrPoints)
	item.Status = uint32(status)
	return &item, nil
}

// UpsertConvertConfig 更新单项宠物资质转化率配置。
func (r *PetProgressionRepository) UpsertConvertConfig(ctx context.Context, attrType string, input petprogression.AdminUpsertConvertConfigInput) (*petprogression.ConvertConfig, error) {
	var (
		item        petprogression.ConvertConfig
		convertRate float64
		status      int64
	)
	err := r.db.QueryRowContext(
		ctx,
		upsertPetConvertConfigQuery,
		attrType,
		input.ConvertRate,
		input.Status,
	).Scan(&item.AttrType, &convertRate, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, petprogression.ErrConvertConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	item.ConvertRate = convertRate
	item.Status = uint32(status)
	return &item, nil
}

const listPetProgressionTargetsQuery = `
SELECT player_id, id
FROM player_pet
WHERE ($1 = 0 OR player_id = $1)
  AND ($2 = 0 OR id = $2)
ORDER BY player_id ASC, id ASC
`

const saveRecalculatedPetCombatStatsQuery = `
UPDATE player_pet
SET hp_max = $3,
    atk = $4,
    def = $5,
    spd = $6,
    mana = $7,
    hp = LEAST($8, $3)
WHERE player_id = $1 AND id = $2
`

// ListProgressionTargets 列出待重算战斗属性的宠物实例；playerID/petUID 为 0 时不限制。
func (r *PetProgressionRepository) ListProgressionTargets(ctx context.Context, playerID uint64, petUID uint64) ([]petprogression.PetProgressionTarget, error) {
	rows, err := r.db.QueryContext(ctx, listPetProgressionTargetsQuery, playerID, petUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]petprogression.PetProgressionTarget, 0, 32)
	for rows.Next() {
		var (
			target      petprogression.PetProgressionTarget
			playerValue int64
			petValue    int64
		)
		if err := rows.Scan(&playerValue, &petValue); err != nil {
			return nil, err
		}
		target.PlayerID = uint64(playerValue)
		target.PetUID = uint64(petValue)
		items = append(items, target)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// SaveRecalculatedCombatStats 仅更新战斗属性与当前血量上限约束，不改动等级与经验。
func (r *PetProgressionRepository) SaveRecalculatedCombatStats(
	ctx context.Context,
	playerID uint64,
	petUID uint64,
	combat petprogression.CombatStats,
	hp uint32,
) error {
	result, err := r.db.ExecContext(
		ctx,
		saveRecalculatedPetCombatStatsQuery,
		playerID,
		petUID,
		combat.HPMax,
		combat.ATK,
		combat.DEF,
		combat.SPD,
		combat.MANA,
		hp,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return petprogression.ErrPetProgressionNotFound
	}
	return nil
}
