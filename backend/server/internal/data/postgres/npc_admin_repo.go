package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/npcdialogue"
)

const adminNPCEntityListBaseQuery = `
SELECT
  e.entity_id,
  e.entity_code,
  e.display_name,
  e.entity_type,
  e.scene_id,
  COALESCE(s.scene_name, '') AS scene_name,
  e.status,
  e.created_at,
  e.updated_at
FROM world_entity_definition e
LEFT JOIN world_scene_definition s ON s.scene_id = e.scene_id
`

const adminNPCEntityDetailQuery = `
SELECT
  e.entity_id,
  e.entity_code,
  e.display_name,
  e.entity_type,
  e.scene_id,
  COALESCE(s.scene_name, '') AS scene_name,
  e.status,
  e.created_at,
  e.updated_at
FROM world_entity_definition e
LEFT JOIN world_scene_definition s ON s.scene_id = e.scene_id
WHERE e.entity_id = $1
LIMIT 1
`

const insertAdminNPCEntityQuery = `
WITH generated AS (
  SELECT nextval('world_entity_definition_entity_id_seq')::BIGINT AS entity_id
)
INSERT INTO world_entity_definition (
  entity_id,
  entity_code,
  display_name,
  entity_type,
  scene_id,
  status
) SELECT
  generated.entity_id,
  'npc_' || generated.entity_id::TEXT,
  $1,
  $2,
  $3,
  $4
FROM generated
RETURNING entity_id
`

const updateAdminNPCEntityQuery = `
UPDATE world_entity_definition
SET display_name = $2,
    entity_type = $3,
    scene_id = $4,
    status = $5
WHERE entity_id = $1
`

const listAdminWorldScenesQuery = `
SELECT scene_id, scene_code, scene_name, required_level, status
FROM world_scene_definition
WHERE status = 1
ORDER BY scene_id ASC
`

const updateAdminWorldSceneRequiredLevelQuery = `
UPDATE world_scene_definition
SET required_level = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE scene_id = $1
RETURNING scene_id, scene_code, scene_name, required_level, status
`

const deleteAdminNPCEntityQuery = `
DELETE FROM world_entity_definition
WHERE entity_id = $1
`

const adminNPCMenuEntryListBaseQuery = `
SELECT
  entity_id,
  entry_id,
  entry_type,
  title,
  subtitle,
  state,
  priority,
  sort_order,
  action_result_type,
  battle_encounter_entity_id,
  linked_quest_id,
  status,
  created_at,
  updated_at
FROM npc_menu_entry
`

const adminNPCMenuEntryDetailQuery = `
SELECT
  entity_id,
  entry_id,
  entry_type,
  title,
  subtitle,
  state,
  priority,
  sort_order,
  action_result_type,
  action_notice,
  battle_encounter_entity_id,
  linked_quest_id,
  conditions_json,
  status,
  created_at,
  updated_at
FROM npc_menu_entry
WHERE entity_id = $1 AND entry_id = $2
LIMIT 1
`

const insertAdminNPCMenuEntryQuery = `
INSERT INTO npc_menu_entry (
  entity_id,
  entry_id,
  entry_type,
  title,
  subtitle,
  state,
  priority,
  sort_order,
  action_result_type,
  action_notice,
  battle_encounter_entity_id,
  linked_quest_id,
  conditions_json,
  status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`

const updateAdminNPCMenuEntryQuery = `
UPDATE npc_menu_entry
SET entity_id = $3,
    entry_type = $4,
    title = $5,
    subtitle = $6,
    state = $7,
    priority = $8,
    sort_order = $9,
    action_result_type = $10,
    action_notice = $11,
    battle_encounter_entity_id = $12,
    linked_quest_id = $13,
    conditions_json = $14,
    status = $15
WHERE entity_id = $1 AND entry_id = $2
`

const deleteAdminNPCMenuEntryQuery = `
DELETE FROM npc_menu_entry
WHERE entity_id = $1 AND entry_id = $2
`

const adminNPCEntityExistsQuery = `
SELECT COUNT(1)
FROM world_entity_definition
WHERE entity_id = $1
`

func (r *NPCRepository) ListEntitiesForAdmin(ctx context.Context, query npc.AdminEntityListQuery) (*npc.AdminEntityList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 6)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.EntityID > 0 {
		conditions = append(conditions, "e.entity_id = "+nextArg(query.EntityID))
	}
	if query.SceneID > 0 {
		conditions = append(conditions, "e.scene_id = "+nextArg(query.SceneID))
	}
	if query.EntityType != nil {
		conditions = append(conditions, "e.entity_type = "+nextArg(*query.EntityType))
	}
	if query.Status != nil {
		conditions = append(conditions, "e.status = "+nextArg(*query.Status))
	}
	if query.Name != "" {
		conditions = append(conditions, "e.display_name ILIKE "+nextArg("%"+query.Name+"%"))
	}
	whereClause := joinWhere(conditions)
	countQuery := `SELECT COUNT(1) FROM world_entity_definition e ` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminNPCEntityListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY e.scene_id ASC, e.entity_id ASC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]npc.AdminEntitySummary, 0)
	for rows.Next() {
		item, err := scanAdminNPCEntitySummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &npc.AdminEntityList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *NPCRepository) FindAdminEntityDetailByEntityID(ctx context.Context, entityID uint64) (*npc.AdminEntityDetail, error) {
	item, err := scanAdminNPCEntityDetail(r.db.QueryRowContext(ctx, adminNPCEntityDetailQuery, entityID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *NPCRepository) CreateEntityForAdmin(ctx context.Context, input npc.AdminCreateEntityInput) (*npc.AdminEntityDetail, error) {
	var entityID uint64
	err := r.db.QueryRowContext(ctx, insertAdminNPCEntityQuery, input.DisplayName, input.EntityType, input.SceneID, input.Status).Scan(&entityID)
	if err != nil {
		if isNPCUniqueViolation(err) {
			return nil, npc.ErrAdminNPCConflict
		}
		return nil, err
	}
	return r.FindAdminEntityDetailByEntityID(ctx, entityID)
}

func (r *NPCRepository) UpdateEntityForAdmin(ctx context.Context, entityID uint64, input npc.AdminUpdateEntityInput) (*npc.AdminEntityDetail, error) {
	result, err := r.db.ExecContext(ctx, updateAdminNPCEntityQuery, entityID, input.DisplayName, input.EntityType, input.SceneID, input.Status)
	if err != nil {
		if isNPCUniqueViolation(err) {
			return nil, npc.ErrAdminNPCConflict
		}
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, npc.ErrAdminNPCNotFound
	}
	return r.FindAdminEntityDetailByEntityID(ctx, entityID)
}

func (r *NPCRepository) DeleteEntityForAdmin(ctx context.Context, entityID uint64) error {
	result, err := r.db.ExecContext(ctx, deleteAdminNPCEntityQuery, entityID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return npc.ErrAdminNPCNotFound
	}
	return nil
}

func (r *NPCRepository) ListMenuEntriesForAdmin(ctx context.Context, query npc.AdminMenuEntryListQuery) (*npc.AdminMenuEntryList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 5)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.EntityID > 0 {
		conditions = append(conditions, "entity_id = "+nextArg(query.EntityID))
	}
	if query.EntryID != "" {
		conditions = append(conditions, "entry_id = "+nextArg(query.EntryID))
	}
	if query.Status != nil {
		conditions = append(conditions, "status = "+nextArg(*query.Status))
	}
	whereClause := joinWhere(conditions)
	countQuery := `SELECT COUNT(1) FROM npc_menu_entry ` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminNPCMenuEntryListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY entity_id ASC, priority DESC, sort_order ASC, entry_id ASC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]npc.AdminMenuEntrySummary, 0)
	for rows.Next() {
		item, err := scanAdminNPCMenuEntrySummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &npc.AdminMenuEntryList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *NPCRepository) FindAdminMenuEntryDetail(ctx context.Context, entityID uint64, entryID string) (*npc.AdminMenuEntryDetail, error) {
	item, err := scanAdminNPCMenuEntryDetail(r.db.QueryRowContext(ctx, adminNPCMenuEntryDetailQuery, entityID, entryID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *NPCRepository) CreateMenuEntryForAdmin(ctx context.Context, input npc.AdminCreateMenuEntryInput) (*npc.AdminMenuEntryDetail, error) {
	ok, err := r.entityExists(ctx, input.EntityID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, npc.ErrAdminNPCNotFound
	}
	_, err = r.db.ExecContext(ctx, insertAdminNPCMenuEntryQuery, input.EntityID, input.EntryID, input.EntryType, input.Title, input.Subtitle, input.State, input.Priority, input.SortOrder, input.ActionResultType, input.ActionNotice, input.BattleEncounterEntityID, input.LinkedQuestID, npcdialogue.EncodeAdminConditionsJSON(input.Conditions), input.Status)
	if err != nil {
		if isNPCUniqueViolation(err) {
			return nil, npc.ErrAdminNPCConflict
		}
		return nil, err
	}
	return r.FindAdminMenuEntryDetail(ctx, input.EntityID, input.EntryID)
}

func (r *NPCRepository) UpdateMenuEntryForAdmin(ctx context.Context, entityID uint64, entryID string, input npc.AdminUpdateMenuEntryInput) (*npc.AdminMenuEntryDetail, error) {
	ok, err := r.entityExists(ctx, input.EntityID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, npc.ErrAdminNPCNotFound
	}
	result, err := r.db.ExecContext(ctx, updateAdminNPCMenuEntryQuery, entityID, entryID, input.EntityID, input.EntryType, input.Title, input.Subtitle, input.State, input.Priority, input.SortOrder, input.ActionResultType, input.ActionNotice, input.BattleEncounterEntityID, input.LinkedQuestID, npcdialogue.EncodeAdminConditionsJSON(input.Conditions), input.Status)
	if err != nil {
		if isNPCUniqueViolation(err) {
			return nil, npc.ErrAdminNPCConflict
		}
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, npc.ErrAdminNPCMenuEntryNotFound
	}
	return r.FindAdminMenuEntryDetail(ctx, input.EntityID, entryID)
}

func (r *NPCRepository) DeleteMenuEntryForAdmin(ctx context.Context, entityID uint64, entryID string) error {
	result, err := r.db.ExecContext(ctx, deleteAdminNPCMenuEntryQuery, entityID, entryID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return npc.ErrAdminNPCMenuEntryNotFound
	}
	return nil
}

func (r *NPCRepository) entityExists(ctx context.Context, entityID uint64) (bool, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, adminNPCEntityExistsQuery, entityID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *NPCRepository) ListWorldScenesForAdmin(ctx context.Context) ([]npc.AdminWorldSceneSummary, error) {
	rows, err := r.db.QueryContext(ctx, listAdminWorldScenesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]npc.AdminWorldSceneSummary, 0)
	for rows.Next() {
		var (
			item    npc.AdminWorldSceneSummary
			sceneID int64
			status  int64
		)
		if err := rows.Scan(&sceneID, &item.SceneCode, &item.SceneName, &item.RequiredLevel, &status); err != nil {
			return nil, err
		}
		item.SceneID = uint32(sceneID)
		item.Status = uint32(status)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// UpdateWorldSceneRequiredLevelForAdmin 持久化地图最低进入等级并返回最新配置。
func (r *NPCRepository) UpdateWorldSceneRequiredLevelForAdmin(ctx context.Context, sceneID uint32, requiredLevel uint32) (*npc.AdminWorldSceneSummary, error) {
	var item npc.AdminWorldSceneSummary
	var storedSceneID int64
	var status int64
	err := r.db.QueryRowContext(ctx, updateAdminWorldSceneRequiredLevelQuery, sceneID, requiredLevel).Scan(
		&storedSceneID,
		&item.SceneCode,
		&item.SceneName,
		&item.RequiredLevel,
		&status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.SceneID = uint32(storedSceneID)
	item.Status = uint32(status)
	return &item, nil
}

func scanAdminNPCEntitySummary(rows *sql.Rows) (npc.AdminEntitySummary, error) {
	var (
		item       npc.AdminEntitySummary
		entityType int64
		sceneID    int64
		status     int64
	)
	if err := rows.Scan(&item.EntityID, &item.EntityCode, &item.DisplayName, &entityType, &sceneID, &item.SceneName, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return npc.AdminEntitySummary{}, err
	}
	item.EntityType = uint32(entityType)
	item.SceneID = uint32(sceneID)
	item.Status = uint32(status)
	item.StatusText = npc.AdminNPCStatusText(item.Status)
	return item, nil
}

func scanAdminNPCEntityDetail(row *sql.Row) (*npc.AdminEntityDetail, error) {
	var (
		item       npc.AdminEntityDetail
		entityType int64
		sceneID    int64
		status     int64
	)
	if err := row.Scan(&item.EntityID, &item.EntityCode, &item.DisplayName, &entityType, &sceneID, &item.SceneName, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.EntityType = uint32(entityType)
	item.SceneID = uint32(sceneID)
	item.Status = uint32(status)
	item.StatusText = npc.AdminNPCStatusText(item.Status)
	return &item, nil
}

func scanAdminNPCMenuEntrySummary(rows *sql.Rows) (npc.AdminMenuEntrySummary, error) {
	var (
		item          npc.AdminMenuEntrySummary
		priority      int64
		sortOrder     int64
		linkedQuestID int64
		status        int64
	)
	if err := rows.Scan(&item.EntityID, &item.EntryID, &item.EntryType, &item.Title, &item.Subtitle, &item.State, &priority, &sortOrder, &item.ActionResultType, &item.BattleEncounterEntityID, &linkedQuestID, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return npc.AdminMenuEntrySummary{}, err
	}
	item.Priority = uint32(priority)
	item.SortOrder = uint32(sortOrder)
	item.LinkedQuestID = uint64(linkedQuestID)
	item.Status = uint32(status)
	item.StatusText = npc.AdminNPCStatusText(item.Status)
	return item, nil
}

func scanAdminNPCMenuEntryDetail(row *sql.Row) (*npc.AdminMenuEntryDetail, error) {
	var (
		item           npc.AdminMenuEntryDetail
		priority       int64
		sortOrder      int64
		linkedQuestID  int64
		conditionsJSON []byte
		status         int64
	)
	if err := row.Scan(&item.EntityID, &item.EntryID, &item.EntryType, &item.Title, &item.Subtitle, &item.State, &priority, &sortOrder, &item.ActionResultType, &item.ActionNotice, &item.BattleEncounterEntityID, &linkedQuestID, &conditionsJSON, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.Priority = uint32(priority)
	item.SortOrder = uint32(sortOrder)
	item.LinkedQuestID = uint64(linkedQuestID)
	item.Conditions = npcdialogue.DecodeAdminConditionsJSON(conditionsJSON)
	item.Status = uint32(status)
	item.StatusText = npc.AdminNPCStatusText(item.Status)
	return &item, nil
}

func isNPCUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
