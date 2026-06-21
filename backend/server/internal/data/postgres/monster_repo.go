package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/monster"
)

type MonsterRepository struct {
	db DBTX
}

func NewMonsterRepository(db DBTX) *MonsterRepository {
	return &MonsterRepository{db: db}
}

const adminMonsterDefinitionListQuery = `
SELECT monster_id, monster_name, level, quality, status, skin_id, updated_at, created_at
FROM monster_definition
`

const adminMonsterDefinitionDetailQuery = `
SELECT monster_id, monster_name, description, level, quality, hp, hp_max, atk, def, spd, mana,
       guard, talent_dmg_pct, talent_reduce_pct, element_adv_pct, element_penalty_pct,
       skill_ids,
       is_capturable, capture_pet_id, capture_rate_base, capture_min_hp_pct, capture_item_ids,
       status, skin_id, created_at, updated_at
FROM monster_definition
WHERE monster_id = $1
LIMIT 1
`

const insertMonsterDefinitionQuery = `
INSERT INTO monster_definition (
  monster_id, monster_name, description, level, quality, hp, hp_max, atk, def, spd, mana,
  guard, talent_dmg_pct, talent_reduce_pct, element_adv_pct, element_penalty_pct,
  skill_ids,
  is_capturable, capture_pet_id, capture_rate_base, capture_min_hp_pct, capture_item_ids, status, skin_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb,$18,$19,$20,$21,$22::jsonb,$23,$24)
`

const updateMonsterDefinitionQuery = `
UPDATE monster_definition
SET monster_name = $2, description = $3, level = $4, quality = $5, hp = $6, hp_max = $7,
    atk = $8, def = $9, spd = $10, mana = $11,
    guard = $12, talent_dmg_pct = $13, talent_reduce_pct = $14, element_adv_pct = $15, element_penalty_pct = $16,
    skill_ids = $17::jsonb,
    is_capturable = $18, capture_pet_id = $19, capture_rate_base = $20, capture_min_hp_pct = $21,
    capture_item_ids = $22::jsonb, status = $23, skin_id = $24
WHERE monster_id = $1
`

const deleteMonsterDefinitionQuery = `DELETE FROM monster_definition WHERE monster_id = $1`

const adminMonsterEncounterListQuery = `
SELECT entity_id, encounter_name, spawn_monster_ids, status, updated_at, created_at
FROM monster_encounter
`

const adminMonsterEncounterDetailQuery = `
SELECT entity_id, encounter_name, description, spawn_monster_ids, status, created_at, updated_at
FROM monster_encounter
WHERE entity_id = $1
LIMIT 1
`

const insertMonsterEncounterQuery = `
INSERT INTO monster_encounter (entity_id, encounter_name, description, spawn_monster_ids, status)
VALUES ($1,$2,$3,$4::jsonb,$5)
`

const updateMonsterEncounterQuery = `
UPDATE monster_encounter
SET encounter_name = $2, description = $3, spawn_monster_ids = $4::jsonb, status = $5
WHERE entity_id = $1
`

const deleteMonsterEncounterQuery = `DELETE FROM monster_encounter WHERE entity_id = $1`

const runtimeMonsterDefinitionQuery = `
SELECT monster_id, monster_name, level, quality, hp, hp_max, atk, def, spd, mana,
       guard, talent_dmg_pct, talent_reduce_pct, element_adv_pct, element_penalty_pct,
       skill_ids, skin_id
FROM monster_definition
WHERE monster_id = $1 AND status = 1
LIMIT 1
`

const runtimeMonsterCaptureConfigQuery = `
SELECT monster_id, is_capturable, capture_pet_id, capture_rate_base, capture_min_hp_pct, capture_item_ids
FROM monster_definition
WHERE monster_id = $1 AND status = 1
LIMIT 1
`

const runtimeMonsterEncounterQuery = `
SELECT entity_id, encounter_name, spawn_monster_ids
FROM monster_encounter
WHERE entity_id = $1 AND status = 1
LIMIT 1
`

const runtimeWildEncounterConfigQuery = `
SELECT scene_id, encounter_name, encounter_rate, spawn_monster_ids
FROM scene_wild_encounter
WHERE scene_id = $1 AND status = 1
LIMIT 1
`

const runtimeWildEncounterQuery = `
SELECT scene_id, encounter_name, spawn_monster_ids
FROM scene_wild_encounter
WHERE scene_id = $1 AND status = 1
LIMIT 1
`

const adminSceneWildEncounterListQuery = `
SELECT scene_id, encounter_name, encounter_rate, spawn_monster_ids, status, updated_at, created_at
FROM scene_wild_encounter
`

const adminSceneWildEncounterDetailQuery = `
SELECT scene_id, encounter_name, description, encounter_rate, spawn_monster_ids, status, created_at, updated_at
FROM scene_wild_encounter
WHERE scene_id = $1
LIMIT 1
`

const insertSceneWildEncounterQuery = `
INSERT INTO scene_wild_encounter (scene_id, encounter_name, description, encounter_rate, spawn_monster_ids, status)
VALUES ($1,$2,$3,$4,$5::jsonb,$6)
`

const updateSceneWildEncounterQuery = `
UPDATE scene_wild_encounter
SET encounter_name = $2, description = $3, encounter_rate = $4, spawn_monster_ids = $5::jsonb, status = $6
WHERE scene_id = $1
`

const deleteSceneWildEncounterQuery = `DELETE FROM scene_wild_encounter WHERE scene_id = $1`

func (r *MonsterRepository) ListDefinitionsForAdmin(ctx context.Context, query monster.AdminDefinitionListQuery) (*monster.AdminDefinitionList, error) {
	query = query.Normalize()
	conditions := []string{}
	args := make([]any, 0, 4)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.MonsterID > 0 {
		conditions = append(conditions, "monster_id = "+nextArg(query.MonsterID))
	}
	if query.Name != "" {
		conditions = append(conditions, "monster_name ILIKE "+nextArg("%"+query.Name+"%"))
	}
	if query.Enabled != nil {
		status := int64(0)
		if *query.Enabled {
			status = 1
		}
		conditions = append(conditions, "status = "+nextArg(status))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM monster_definition `+whereClause, args...).Scan(&total); err != nil {
		return nil, err
	}
	listQuery := adminMonsterDefinitionListQuery + whereClause + ` ORDER BY monster_id ASC LIMIT ` + nextArg(query.PageSize) + ` OFFSET ` + nextArg((query.Page-1)*query.PageSize)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]monster.AdminDefinitionSummary, 0, query.PageSize)
	for rows.Next() {
		item, err := scanAdminMonsterDefinitionSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &monster.AdminDefinitionList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, rows.Err()
}

func (r *MonsterRepository) FindDefinitionForAdmin(ctx context.Context, monsterID uint32) (*monster.AdminDefinitionDetail, error) {
	row := r.db.QueryRowContext(ctx, adminMonsterDefinitionDetailQuery, monsterID)
	return scanAdminMonsterDefinitionDetail(row)
}

func (r *MonsterRepository) CreateDefinitionForAdmin(ctx context.Context, input monster.AdminUpsertDefinitionInput) (*monster.AdminDefinitionDetail, error) {
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, err
	}
	captureItemIDsJSON, err := json.Marshal(input.CaptureItemIDs)
	if err != nil {
		return nil, err
	}
	status := statusFromEnabled(input.IsEnabled)
	if _, err := r.db.ExecContext(ctx, insertMonsterDefinitionQuery, input.MonsterID, input.MonsterName, input.Description, input.Level, input.Quality, input.HP, input.HPMax, input.ATK, input.DEF, input.SPD, input.MANA, input.Guard, input.TalentDmgPct, input.TalentReducePct, input.ElementAdvPct, input.ElementPenaltyPct, skillIDsJSON, boolToInt(input.IsCapturable), input.CapturePetID, input.CaptureRateBase, input.CaptureMinHPPct, captureItemIDsJSON, status, input.SkinID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, monster.ErrMonsterDefinitionConflict
		}
		return nil, err
	}
	return r.FindDefinitionForAdmin(ctx, input.MonsterID)
}

func (r *MonsterRepository) UpdateDefinitionForAdmin(ctx context.Context, monsterID uint32, input monster.AdminUpsertDefinitionInput) (*monster.AdminDefinitionDetail, error) {
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, err
	}
	captureItemIDsJSON, err := json.Marshal(input.CaptureItemIDs)
	if err != nil {
		return nil, err
	}
	result, err := r.db.ExecContext(ctx, updateMonsterDefinitionQuery, monsterID, input.MonsterName, input.Description, input.Level, input.Quality, input.HP, input.HPMax, input.ATK, input.DEF, input.SPD, input.MANA, input.Guard, input.TalentDmgPct, input.TalentReducePct, input.ElementAdvPct, input.ElementPenaltyPct, skillIDsJSON, boolToInt(input.IsCapturable), input.CapturePetID, input.CaptureRateBase, input.CaptureMinHPPct, captureItemIDsJSON, statusFromEnabled(input.IsEnabled), input.SkinID)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, nil
	}
	return r.FindDefinitionForAdmin(ctx, monsterID)
}

func (r *MonsterRepository) DeleteDefinitionForAdmin(ctx context.Context, monsterID uint32) error {
	result, err := r.db.ExecContext(ctx, deleteMonsterDefinitionQuery, monsterID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return monster.ErrMonsterDefinitionNotFound
	}
	return nil
}

func (r *MonsterRepository) MapUsableMonsterIDs(ctx context.Context, monsterIDs []uint32) (map[uint32]bool, error) {
	result := make(map[uint32]bool, len(monsterIDs))
	unique := uniqueUint32(monsterIDs)
	if len(unique) == 0 {
		return result, nil
	}
	args := make([]any, len(unique))
	placeholders := make([]string, len(unique))
	for index, monsterID := range unique {
		args[index] = monsterID
		placeholders[index] = fmt.Sprintf("$%d", index+1)
	}
	query := fmt.Sprintf(`SELECT monster_id FROM monster_definition WHERE status = 1 AND monster_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var monsterID int64
		if err := rows.Scan(&monsterID); err != nil {
			return nil, err
		}
		result[uint32(monsterID)] = true
	}
	return result, rows.Err()
}

func (r *MonsterRepository) FindRuntimeDefinition(ctx context.Context, monsterID uint32) (*monster.RuntimeDefinition, error) {
	row := r.db.QueryRowContext(ctx, runtimeMonsterDefinitionQuery, monsterID)
	return scanRuntimeMonsterDefinition(row)
}

func (r *MonsterRepository) FindCaptureConfig(ctx context.Context, monsterID uint32) (*monster.CaptureConfig, error) {
	row := r.db.QueryRowContext(ctx, runtimeMonsterCaptureConfigQuery, monsterID)
	var (
		config         monster.CaptureConfig
		monsterIDValue int64
		isCapturable   int64
		capturePetID   int64
		captureRate    int64
		captureMinHP   int64
		captureItems   []byte
	)
	if err := row.Scan(&monsterIDValue, &isCapturable, &capturePetID, &captureRate, &captureMinHP, &captureItems); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	config.MonsterID = uint32(monsterIDValue)
	config.IsCapturable = isCapturable == 1
	config.CapturePetID = uint32(capturePetID)
	config.CaptureRateBase = uint32(captureRate)
	config.CaptureMinHPPct = uint32(captureMinHP)
	if len(captureItems) > 0 {
		if err := json.Unmarshal(captureItems, &config.CaptureItemIDs); err != nil {
			return nil, err
		}
	}
	if config.CaptureItemIDs == nil {
		config.CaptureItemIDs = []uint32{}
	}
	return &config, nil
}

func (r *MonsterRepository) ListEncountersForAdmin(ctx context.Context, query monster.AdminEncounterListQuery) (*monster.AdminEncounterList, error) {
	query = query.Normalize()
	conditions := []string{}
	args := make([]any, 0, 4)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.EntityID > 0 {
		conditions = append(conditions, "entity_id = "+nextArg(query.EntityID))
	}
	if query.Name != "" {
		conditions = append(conditions, "encounter_name ILIKE "+nextArg("%"+query.Name+"%"))
	}
	if query.Enabled != nil {
		status := int64(0)
		if *query.Enabled {
			status = 1
		}
		conditions = append(conditions, "status = "+nextArg(status))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM monster_encounter `+whereClause, args...).Scan(&total); err != nil {
		return nil, err
	}
	listQuery := adminMonsterEncounterListQuery + whereClause + ` ORDER BY entity_id ASC LIMIT ` + nextArg(query.PageSize) + ` OFFSET ` + nextArg((query.Page-1)*query.PageSize)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]monster.AdminEncounterSummary, 0, query.PageSize)
	for rows.Next() {
		item, err := scanAdminMonsterEncounterSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &monster.AdminEncounterList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, rows.Err()
}

func (r *MonsterRepository) FindEncounterForAdmin(ctx context.Context, entityID uint64) (*monster.AdminEncounterDetail, error) {
	row := r.db.QueryRowContext(ctx, adminMonsterEncounterDetailQuery, entityID)
	return scanAdminMonsterEncounterDetail(row)
}

func (r *MonsterRepository) CreateEncounterForAdmin(ctx context.Context, input monster.AdminUpsertEncounterInput) (*monster.AdminEncounterDetail, error) {
	spawnJSON, err := json.Marshal(input.SpawnMonsterIDs)
	if err != nil {
		return nil, err
	}
	if _, err := r.db.ExecContext(ctx, insertMonsterEncounterQuery, input.EntityID, input.EncounterName, input.Description, spawnJSON, statusFromEnabled(input.IsEnabled)); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, monster.ErrMonsterEncounterConflict
		}
		return nil, err
	}
	return r.FindEncounterForAdmin(ctx, input.EntityID)
}

func (r *MonsterRepository) UpdateEncounterForAdmin(ctx context.Context, entityID uint64, input monster.AdminUpsertEncounterInput) (*monster.AdminEncounterDetail, error) {
	spawnJSON, err := json.Marshal(input.SpawnMonsterIDs)
	if err != nil {
		return nil, err
	}
	result, err := r.db.ExecContext(ctx, updateMonsterEncounterQuery, entityID, input.EncounterName, input.Description, spawnJSON, statusFromEnabled(input.IsEnabled))
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, nil
	}
	return r.FindEncounterForAdmin(ctx, entityID)
}

func (r *MonsterRepository) DeleteEncounterForAdmin(ctx context.Context, entityID uint64) error {
	result, err := r.db.ExecContext(ctx, deleteMonsterEncounterQuery, entityID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return monster.ErrMonsterEncounterNotFound
	}
	return nil
}

func (r *MonsterRepository) FindRuntimeEncounter(ctx context.Context, entityID uint64) (*monster.RuntimeEncounter, error) {
	row := r.db.QueryRowContext(ctx, runtimeMonsterEncounterQuery, entityID)
	var entityIDValue int64
	var encounterName string
	var spawnJSON []byte
	if err := row.Scan(&entityIDValue, &encounterName, &spawnJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	spawnMonsterIDs, err := parseSpawnMonsterIDsJSON(spawnJSON)
	if err != nil {
		return nil, err
	}
	slots, err := r.buildRuntimeEncounterSlots(ctx, spawnMonsterIDs)
	if err != nil {
		return nil, err
	}
	if len(slots) == 0 {
		return nil, nil
	}
	return &monster.RuntimeEncounter{EntityID: uint64(entityIDValue), EncounterName: encounterName, Slots: slots}, nil
}

func (r *MonsterRepository) FindRuntimeWildEncounterConfig(ctx context.Context, sceneID uint32) (*monster.RuntimeWildEncounterConfig, error) {
	row := r.db.QueryRowContext(ctx, runtimeWildEncounterConfigQuery, sceneID)
	var sceneIDValue int64
	var encounterName string
	var encounterRate int64
	var spawnJSON []byte
	if err := row.Scan(&sceneIDValue, &encounterName, &encounterRate, &spawnJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	spawnMonsterIDs, err := parseSpawnMonsterIDsJSON(spawnJSON)
	if err != nil {
		return nil, err
	}
	if len(spawnMonsterIDs) == 0 || encounterRate <= 0 {
		return &monster.RuntimeWildEncounterConfig{SceneID: uint32(sceneIDValue)}, nil
	}
	return &monster.RuntimeWildEncounterConfig{
		Enabled:         true,
		SceneID:         uint32(sceneIDValue),
		EncounterRate:   uint32(encounterRate),
		SpawnMonsterIDs: spawnMonsterIDs,
	}, nil
}

func (r *MonsterRepository) FindRuntimeWildEncounter(ctx context.Context, sceneID uint32) (*monster.RuntimeWildEncounter, error) {
	row := r.db.QueryRowContext(ctx, runtimeWildEncounterQuery, sceneID)
	var sceneIDValue int64
	var encounterName string
	var spawnJSON []byte
	if err := row.Scan(&sceneIDValue, &encounterName, &spawnJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	spawnMonsterIDs, err := parseSpawnMonsterIDsJSON(spawnJSON)
	if err != nil {
		return nil, err
	}
	slots, err := r.buildRuntimeEncounterSlots(ctx, spawnMonsterIDs)
	if err != nil {
		return nil, err
	}
	if len(slots) == 0 {
		return nil, nil
	}
	return &monster.RuntimeWildEncounter{
		SceneID:       uint32(sceneIDValue),
		EncounterName: encounterName,
		Slots:         slots,
	}, nil
}

func parseSpawnMonsterIDsJSON(spawnJSON []byte) ([]uint32, error) {
	spawnMonsterIDs := []uint32{}
	if len(spawnJSON) > 0 {
		if err := json.Unmarshal(spawnJSON, &spawnMonsterIDs); err != nil {
			return nil, err
		}
	}
	return spawnMonsterIDs, nil
}

func (r *MonsterRepository) buildRuntimeEncounterSlots(ctx context.Context, spawnMonsterIDs []uint32) ([]monster.RuntimeEncounterSlot, error) {
	if len(spawnMonsterIDs) == 0 {
		return nil, nil
	}
	slots := make([]monster.RuntimeEncounterSlot, 0, len(spawnMonsterIDs))
	for _, monsterID := range spawnMonsterIDs {
		definition, err := r.FindRuntimeDefinition(ctx, monsterID)
		if err != nil {
			return nil, err
		}
		if definition == nil {
			continue
		}
		skillIDs := append([]uint32{}, definition.SkillIDs...)
		slots = append(slots, monster.RuntimeEncounterSlot{
			MonsterID: definition.MonsterID, MonsterName: definition.MonsterName,
			Level: definition.Level, HP: definition.HP, HPMax: definition.HPMax,
			ATK: definition.ATK, DEF: definition.DEF, SPD: definition.SPD, MANA: definition.MANA,
			SkillIDs: skillIDs, SkinID: definition.SkinID,
			Guard: definition.Guard, TalentDmgPct: definition.TalentDmgPct,
			TalentReducePct: definition.TalentReducePct, ElementAdvPct: definition.ElementAdvPct,
			ElementPenaltyPct: definition.ElementPenaltyPct,
		})
	}
	return slots, nil
}

func (r *MonsterRepository) ListWildEncountersForAdmin(ctx context.Context, query monster.AdminWildEncounterListQuery) (*monster.AdminWildEncounterList, error) {
	query = query.Normalize()
	conditions := []string{}
	args := make([]any, 0, 4)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.SceneID > 0 {
		conditions = append(conditions, "scene_id = "+nextArg(query.SceneID))
	}
	if query.Name != "" {
		conditions = append(conditions, "encounter_name ILIKE "+nextArg("%"+query.Name+"%"))
	}
	if query.Enabled != nil {
		status := int64(0)
		if *query.Enabled {
			status = 1
		}
		conditions = append(conditions, "status = "+nextArg(status))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM scene_wild_encounter `+whereClause, args...).Scan(&total); err != nil {
		return nil, err
	}
	listQuery := adminSceneWildEncounterListQuery + whereClause + ` ORDER BY scene_id ASC LIMIT ` + nextArg(query.PageSize) + ` OFFSET ` + nextArg((query.Page-1)*query.PageSize)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]monster.AdminWildEncounterSummary, 0, query.PageSize)
	for rows.Next() {
		item, err := scanAdminWildEncounterSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &monster.AdminWildEncounterList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, rows.Err()
}

func (r *MonsterRepository) FindWildEncounterForAdmin(ctx context.Context, sceneID uint32) (*monster.AdminWildEncounterDetail, error) {
	row := r.db.QueryRowContext(ctx, adminSceneWildEncounterDetailQuery, sceneID)
	return scanAdminWildEncounterDetail(row)
}

func (r *MonsterRepository) CreateWildEncounterForAdmin(ctx context.Context, input monster.AdminUpsertWildEncounterInput) (*monster.AdminWildEncounterDetail, error) {
	spawnJSON, err := json.Marshal(input.SpawnMonsterIDs)
	if err != nil {
		return nil, err
	}
	if _, err := r.db.ExecContext(ctx, insertSceneWildEncounterQuery, input.SceneID, input.EncounterName, input.Description, input.EncounterRate, spawnJSON, statusFromEnabled(input.IsEnabled)); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, monster.ErrSceneWildEncounterConflict
		}
		return nil, err
	}
	return r.FindWildEncounterForAdmin(ctx, input.SceneID)
}

func (r *MonsterRepository) UpdateWildEncounterForAdmin(ctx context.Context, sceneID uint32, input monster.AdminUpsertWildEncounterInput) (*monster.AdminWildEncounterDetail, error) {
	spawnJSON, err := json.Marshal(input.SpawnMonsterIDs)
	if err != nil {
		return nil, err
	}
	result, err := r.db.ExecContext(ctx, updateSceneWildEncounterQuery, sceneID, input.EncounterName, input.Description, input.EncounterRate, spawnJSON, statusFromEnabled(input.IsEnabled))
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, nil
	}
	return r.FindWildEncounterForAdmin(ctx, sceneID)
}

func (r *MonsterRepository) DeleteWildEncounterForAdmin(ctx context.Context, sceneID uint32) error {
	result, err := r.db.ExecContext(ctx, deleteSceneWildEncounterQuery, sceneID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return monster.ErrSceneWildEncounterNotFound
	}
	return nil
}

func statusFromEnabled(enabled bool) int64 {
	if enabled {
		return 1
	}
	return 0
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func uniqueUint32(values []uint32) []uint32 {
	unique := make([]uint32, 0, len(values))
	seen := make(map[uint32]struct{}, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func scanAdminMonsterDefinitionSummary(rows *sql.Rows) (monster.AdminDefinitionSummary, error) {
	var item monster.AdminDefinitionSummary
	var monsterID, level, quality, status int64
	if err := rows.Scan(&monsterID, &item.MonsterName, &level, &quality, &status, &item.SkinID, &item.UpdatedAt, &item.CreatedAt); err != nil {
		return monster.AdminDefinitionSummary{}, err
	}
	item.MonsterID = uint32(monsterID)
	item.Level = uint32(level)
	item.Quality = uint32(quality)
	item.IsEnabled = status == 1
	item.StatusText = statusTextFromEnabled(item.IsEnabled)
	return item, nil
}

func scanAdminMonsterDefinitionDetail(row *sql.Row) (*monster.AdminDefinitionDetail, error) {
	var detail monster.AdminDefinitionDetail
	var monsterID, level, quality, hp, hpMax, atk, def, spd, mana, status int64
	var guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct int64
	var isCapturable int64
	var capturePetID, captureRateBase, captureMinHPPct int64
	var skillIDsJSON, captureItemIDsJSON []byte
	if err := row.Scan(
		&monsterID, &detail.MonsterName, &detail.Description, &level, &quality, &hp, &hpMax, &atk, &def, &spd, &mana,
		&guard, &talentDmgPct, &talentReducePct, &elementAdvPct, &elementPenaltyPct,
		&skillIDsJSON,
		&isCapturable, &capturePetID, &captureRateBase, &captureMinHPPct, &captureItemIDsJSON,
		&status, &detail.SkinID, &detail.CreatedAt, &detail.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	detail.MonsterID = uint32(monsterID)
	detail.IsEnabled = status == 1
	detail.StatusText = statusTextFromEnabled(detail.IsEnabled)
	detail.BaseStats = monster.AdminDefinitionBaseStats{
		Level: uint32(level), Quality: uint32(quality), HP: uint32(hp), HPMax: uint32(hpMax),
		ATK: uint32(atk), DEF: uint32(def), SPD: uint32(spd), MANA: uint32(mana),
		Guard: uint32(guard), TalentDmgPct: uint32(talentDmgPct), TalentReducePct: uint32(talentReducePct),
		ElementAdvPct: uint32(elementAdvPct), ElementPenaltyPct: uint32(elementPenaltyPct),
	}
	detail.IsCapturable = isCapturable == 1
	detail.CapturePetID = uint32(capturePetID)
	detail.CaptureRateBase = uint32(captureRateBase)
	detail.CaptureMinHPPct = uint32(captureMinHPPct)
	if len(skillIDsJSON) > 0 {
		if err := json.Unmarshal(skillIDsJSON, &detail.SkillIDs); err != nil {
			return nil, err
		}
	}
	if len(captureItemIDsJSON) > 0 {
		if err := json.Unmarshal(captureItemIDsJSON, &detail.CaptureItemIDs); err != nil {
			return nil, err
		}
	}
	if detail.SkillIDs == nil {
		detail.SkillIDs = []uint32{}
	}
	if detail.CaptureItemIDs == nil {
		detail.CaptureItemIDs = []uint32{}
	}
	return &detail, nil
}

func scanRuntimeMonsterDefinition(row *sql.Row) (*monster.RuntimeDefinition, error) {
	var definition monster.RuntimeDefinition
	var monsterID, level, quality, hp, hpMax, atk, def, spd, mana int64
	var guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct int64
	var skillIDsJSON []byte
	if err := row.Scan(&monsterID, &definition.MonsterName, &level, &quality, &hp, &hpMax, &atk, &def, &spd, &mana,
		&guard, &talentDmgPct, &talentReducePct, &elementAdvPct, &elementPenaltyPct,
		&skillIDsJSON, &definition.SkinID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	definition.MonsterID = uint32(monsterID)
	definition.Level = uint32(level)
	definition.Quality = uint32(quality)
	definition.HP = uint32(hp)
	definition.HPMax = uint32(hpMax)
	definition.ATK = uint32(atk)
	definition.DEF = uint32(def)
	definition.SPD = uint32(spd)
	definition.MANA = uint32(mana)
	definition.Guard = uint32(guard)
	definition.TalentDmgPct = uint32(talentDmgPct)
	definition.TalentReducePct = uint32(talentReducePct)
	definition.ElementAdvPct = uint32(elementAdvPct)
	definition.ElementPenaltyPct = uint32(elementPenaltyPct)
	if len(skillIDsJSON) > 0 {
		if err := json.Unmarshal(skillIDsJSON, &definition.SkillIDs); err != nil {
			return nil, err
		}
	}
	if definition.SkillIDs == nil {
		definition.SkillIDs = []uint32{}
	}
	return &definition, nil
}

func scanAdminMonsterEncounterSummary(rows *sql.Rows) (monster.AdminEncounterSummary, error) {
	var item monster.AdminEncounterSummary
	var entityID, status int64
	var spawnJSON []byte
	if err := rows.Scan(&entityID, &item.EncounterName, &spawnJSON, &status, &item.UpdatedAt, &item.CreatedAt); err != nil {
		return monster.AdminEncounterSummary{}, err
	}
	item.EntityID = uint64(entityID)
	item.IsEnabled = status == 1
	item.StatusText = statusTextFromEnabled(item.IsEnabled)
	spawnIDs := []uint32{}
	if len(spawnJSON) > 0 {
		_ = json.Unmarshal(spawnJSON, &spawnIDs)
	}
	item.SpawnCount = uint32(len(spawnIDs))
	return item, nil
}

func scanAdminMonsterEncounterDetail(row *sql.Row) (*monster.AdminEncounterDetail, error) {
	var detail monster.AdminEncounterDetail
	var entityID, status int64
	var spawnJSON []byte
	if err := row.Scan(&entityID, &detail.EncounterName, &detail.Description, &spawnJSON, &status, &detail.CreatedAt, &detail.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	detail.EntityID = uint64(entityID)
	detail.IsEnabled = status == 1
	detail.StatusText = statusTextFromEnabled(detail.IsEnabled)
	if len(spawnJSON) > 0 {
		if err := json.Unmarshal(spawnJSON, &detail.SpawnMonsterIDs); err != nil {
			return nil, err
		}
	}
	if detail.SpawnMonsterIDs == nil {
		detail.SpawnMonsterIDs = []uint32{}
	}
	return &detail, nil
}

func scanAdminWildEncounterSummary(rows *sql.Rows) (monster.AdminWildEncounterSummary, error) {
	var item monster.AdminWildEncounterSummary
	var sceneID, encounterRate, status int64
	var spawnJSON []byte
	if err := rows.Scan(&sceneID, &item.EncounterName, &encounterRate, &spawnJSON, &status, &item.UpdatedAt, &item.CreatedAt); err != nil {
		return monster.AdminWildEncounterSummary{}, err
	}
	item.SceneID = uint32(sceneID)
	item.EncounterRate = uint32(encounterRate)
	item.IsEnabled = status == 1
	item.StatusText = statusTextFromEnabled(item.IsEnabled)
	spawnIDs := []uint32{}
	if len(spawnJSON) > 0 {
		_ = json.Unmarshal(spawnJSON, &spawnIDs)
	}
	item.SpawnCount = uint32(len(spawnIDs))
	return item, nil
}

func scanAdminWildEncounterDetail(row *sql.Row) (*monster.AdminWildEncounterDetail, error) {
	var detail monster.AdminWildEncounterDetail
	var sceneID, encounterRate, status int64
	var spawnJSON []byte
	if err := row.Scan(&sceneID, &detail.EncounterName, &detail.Description, &encounterRate, &spawnJSON, &status, &detail.CreatedAt, &detail.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	detail.SceneID = uint32(sceneID)
	detail.EncounterRate = uint32(encounterRate)
	detail.IsEnabled = status == 1
	detail.StatusText = statusTextFromEnabled(detail.IsEnabled)
	if len(spawnJSON) > 0 {
		if err := json.Unmarshal(spawnJSON, &detail.SpawnMonsterIDs); err != nil {
			return nil, err
		}
	}
	if detail.SpawnMonsterIDs == nil {
		detail.SpawnMonsterIDs = []uint32{}
	}
	return &detail, nil
}

func statusTextFromEnabled(enabled bool) string {
	if enabled {
		return "启用"
	}
	return "停用"
}

const listMonsterBattleRewardsQuery = `
SELECT id, monster_id, reward_type, exp_target, item_id, quantity, exp_value, sort_order, status, grant_once
FROM monster_battle_reward
ORDER BY monster_id ASC, sort_order ASC, id ASC
`

const listMonsterBattleRewardsByMonsterIDQuery = `
SELECT id, monster_id, reward_type, exp_target, item_id, quantity, exp_value, sort_order, status, grant_once
FROM monster_battle_reward
WHERE monster_id = $1
ORDER BY sort_order ASC, id ASC
`

const deleteMonsterBattleRewardsByMonsterIDQuery = `
DELETE FROM monster_battle_reward
WHERE monster_id = $1
`

const insertMonsterBattleRewardQuery = `
INSERT INTO monster_battle_reward (
  monster_id, reward_type, exp_target, item_id, quantity, exp_value, sort_order, status, grant_once
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, monster_id, reward_type, exp_target, item_id, quantity, exp_value, sort_order, status, grant_once
`

func (r *MonsterRepository) ListBattleRewards(ctx context.Context) ([]monster.BattleRewardEntry, error) {
	rows, err := r.db.QueryContext(ctx, listMonsterBattleRewardsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMonsterBattleRewardRows(rows)
}

func (r *MonsterRepository) ListBattleRewardsByMonsterID(ctx context.Context, monsterID uint32) ([]monster.BattleRewardEntry, error) {
	rows, err := r.db.QueryContext(ctx, listMonsterBattleRewardsByMonsterIDQuery, monsterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMonsterBattleRewardRows(rows)
}

func (r *MonsterRepository) ReplaceBattleRewardsForMonster(ctx context.Context, monsterID uint32, rewards []monster.AdminBattleRewardInput) ([]monster.BattleRewardEntry, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("monster repository transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx, deleteMonsterBattleRewardsByMonsterIDQuery, monsterID); err != nil {
		return nil, err
	}
	result := make([]monster.BattleRewardEntry, 0, len(rewards))
	for _, reward := range rewards {
		row := tx.QueryRowContext(
			ctx,
			insertMonsterBattleRewardQuery,
			monsterID,
			reward.RewardType,
			reward.ExpTarget,
			reward.ItemID,
			reward.Quantity,
			reward.ExpValue,
			reward.SortOrder,
			reward.Status,
			reward.GrantOnce,
		)
		entry, err := scanMonsterBattleRewardRow(row)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanMonsterBattleRewardRows(rows *sql.Rows) ([]monster.BattleRewardEntry, error) {
	result := make([]monster.BattleRewardEntry, 0)
	for rows.Next() {
		entry, err := scanMonsterBattleRewardFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanMonsterBattleRewardRow(row *sql.Row) (monster.BattleRewardEntry, error) {
	var entry monster.BattleRewardEntry
	var monsterID, sortOrder, status, grantOnce int64
	var itemID, quantity, expValue int64
	if err := row.Scan(&entry.ID, &monsterID, &entry.RewardType, &entry.ExpTarget, &itemID, &quantity, &expValue, &sortOrder, &status, &grantOnce); err != nil {
		return monster.BattleRewardEntry{}, err
	}
	entry.MonsterID = uint32(monsterID)
	entry.ItemID = uint64(itemID)
	entry.Quantity = uint64(quantity)
	entry.ExpValue = uint64(expValue)
	entry.SortOrder = uint32(sortOrder)
	entry.Status = uint32(status)
	entry.GrantOnce = uint32(grantOnce)
	return entry, nil
}

func scanMonsterBattleRewardFromRows(rows *sql.Rows) (monster.BattleRewardEntry, error) {
	var entry monster.BattleRewardEntry
	var monsterID, sortOrder, status, grantOnce int64
	var itemID, quantity, expValue int64
	if err := rows.Scan(&entry.ID, &monsterID, &entry.RewardType, &entry.ExpTarget, &itemID, &quantity, &expValue, &sortOrder, &status, &grantOnce); err != nil {
		return monster.BattleRewardEntry{}, err
	}
	entry.MonsterID = uint32(monsterID)
	entry.ItemID = uint64(itemID)
	entry.Quantity = uint64(quantity)
	entry.ExpValue = uint64(expValue)
	entry.SortOrder = uint32(sortOrder)
	entry.Status = uint32(status)
	entry.GrantOnce = uint32(grantOnce)
	return entry, nil
}

