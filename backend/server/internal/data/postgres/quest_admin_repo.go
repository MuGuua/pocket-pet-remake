package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/quest"
)

const adminQuestTemplateListBaseQuery = `
SELECT
  quest_id,
  name,
  quest_type,
  title,
  chapter,
  sort_order,
  accept_mode,
  submit_mode,
  auto_track,
  client_icon_id,
  min_player_level,
  status,
  created_at,
  updated_at
FROM quest_template
`

const adminQuestTemplateDetailQuery = `
SELECT
  quest_id,
  name,
  quest_type,
  title,
  description,
  chapter,
  sort_order,
  accept_mode,
  submit_mode,
  auto_track,
  client_icon_id,
  start_npc_id,
  submit_npc_id,
  accept_animation_key,
  submit_animation_key,
  min_player_level,
  status,
  pre_quest_ids,
  objectives_json,
  rewards_json,
  created_at,
  updated_at
FROM quest_template
WHERE quest_id = $1
LIMIT 1
`

const insertAdminQuestTemplateQuery = `
INSERT INTO quest_template (
  quest_id,
  quest_type,
  name,
  title,
  description,
  chapter,
  sort_order,
  accept_mode,
  submit_mode,
  auto_track,
  client_icon_id,
  start_npc_id,
  submit_npc_id,
  accept_animation_key,
  submit_animation_key,
  min_player_level,
  pre_quest_ids,
  objectives_json,
  rewards_json,
  status
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
)
`

const adminQuestTemplateAutoIDLockQuery = `SELECT pg_advisory_xact_lock($1)`
const adminQuestTemplateNextIDQuery = `SELECT COALESCE(MAX(quest_id), 1000) + 1 FROM quest_template`

const updateAdminQuestTemplateQuery = `
UPDATE quest_template
SET quest_type = $2,
    name = $3,
    title = $4,
    description = $5,
    chapter = $6,
    sort_order = $7,
    accept_mode = $8,
    submit_mode = $9,
    auto_track = $10,
    client_icon_id = $11,
    start_npc_id = $12,
    submit_npc_id = $13,
    accept_animation_key = $14,
    submit_animation_key = $15,
    min_player_level = $16,
    pre_quest_ids = $17,
    objectives_json = $18,
    rewards_json = $19,
    status = $20
WHERE quest_id = $1
`

const softDeleteAdminQuestTemplateQuery = `
UPDATE quest_template
SET status = 0,
    updated_at = CURRENT_TIMESTAMP
WHERE quest_id = $1
`

const adminPlayerQuestListBaseQuery = `
SELECT
  pq.id,
  pq.player_id,
  p.name,
  pq.quest_id,
  COALESCE(qt.title, ''),
  pq.quest_type,
  pq.state,
  pq.tracked,
  pq.reward_claimed,
  pq.accepted_at,
  pq.completed_at,
  pq.created_at,
  pq.updated_at
FROM player_quest pq
JOIN player p ON p.id = pq.player_id
LEFT JOIN quest_template qt ON qt.quest_id = pq.quest_id
`

const adminPlayerQuestDetailQuery = `
SELECT
  pq.id,
  pq.player_id,
  p.name,
  pq.quest_id,
  COALESCE(qt.title, ''),
  pq.quest_type,
  pq.state,
  pq.tracked,
  pq.reward_claimed,
  pq.accepted_at,
  pq.completed_at,
  pq.submitted_at,
  pq.created_at,
  pq.updated_at
FROM player_quest pq
JOIN player p ON p.id = pq.player_id
LEFT JOIN quest_template qt ON qt.quest_id = pq.quest_id
WHERE pq.id = $1
LIMIT 1
`

const adminPlayerQuestObjectivesQuery = `
SELECT
  objective_id,
  description,
  current_value,
  target_value,
  completed
FROM player_quest_objective
WHERE player_id = $1 AND quest_id = $2
ORDER BY objective_id ASC
`

const insertAdminPlayerQuestQuery = `
INSERT INTO player_quest (
  player_id,
  quest_id,
  quest_type,
  state,
  tracked,
  accepted_at,
  completed_at,
  submitted_at,
  reward_claimed
) VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  CASE WHEN $4 IN ('ACCEPTED', 'READY_TO_SUBMIT', 'COMPLETED') THEN CURRENT_TIMESTAMP ELSE NULL END,
  CASE WHEN $4 = 'COMPLETED' THEN CURRENT_TIMESTAMP ELSE NULL END,
  CASE WHEN $4 = 'COMPLETED' THEN CURRENT_TIMESTAMP ELSE NULL END,
  $6
)
RETURNING id
`

const updateAdminPlayerQuestQuery = `
UPDATE player_quest
SET player_id = $2,
    quest_id = $3,
    quest_type = $4,
    state = $5,
    tracked = $6,
    reward_claimed = $7,
    accepted_at = CASE WHEN $5 IN ('ACCEPTED', 'READY_TO_SUBMIT', 'COMPLETED') THEN COALESCE(accepted_at, CURRENT_TIMESTAMP) ELSE NULL END,
    completed_at = CASE WHEN $5 = 'COMPLETED' THEN COALESCE(completed_at, CURRENT_TIMESTAMP) ELSE NULL END,
    submitted_at = CASE WHEN $5 = 'COMPLETED' THEN COALESCE(submitted_at, CURRENT_TIMESTAMP) ELSE NULL END
WHERE id = $1
`

const deleteAdminPlayerQuestQuery = `
DELETE FROM player_quest
WHERE id = $1
`

const deleteAdminPlayerQuestObjectivesQuery = `
DELETE FROM player_quest_objective
WHERE player_id = $1 AND quest_id = $2
`

const insertAdminPlayerQuestObjectiveQuery = `
INSERT INTO player_quest_objective (
  player_id,
  quest_id,
  objective_id,
  event_type,
  description,
  current_value,
  target_value,
  completed,
  target_selector_json
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '{}'::jsonb)
`

const adminQuestTemplateExistsQuery = `SELECT COUNT(1) FROM quest_template WHERE quest_id = $1`
const adminQuestPlayerExistsQuery = `SELECT COUNT(1) FROM player WHERE id = $1 AND status = 1`

func (r *QuestRepository) ListTemplatesForAdmin(ctx context.Context, query quest.AdminTemplateListQuery) (*quest.AdminTemplateList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 6)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.QuestID > 0 {
		conditions = append(conditions, "quest_id = "+nextArg(query.QuestID))
	}
	if query.QuestType != "" {
		conditions = append(conditions, "quest_type = "+nextArg(query.QuestType))
	}
	if query.Title != "" {
		conditions = append(conditions, "title ILIKE "+nextArg("%"+query.Title+"%"))
	}
	if query.Status != nil {
		conditions = append(conditions, "status = "+nextArg(*query.Status))
	}
	whereClause := joinWhere(conditions)
	countQuery := `SELECT COUNT(1) FROM quest_template ` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminQuestTemplateListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY chapter ASC, sort_order ASC, quest_id ASC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]quest.AdminTemplateSummary, 0)
	for rows.Next() {
		item, err := scanAdminQuestTemplateSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &quest.AdminTemplateList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *QuestRepository) FindAdminTemplateDetailByQuestID(ctx context.Context, questID uint64) (*quest.AdminTemplateDetail, error) {
	item, err := scanAdminQuestTemplateDetail(r.db.QueryRowContext(ctx, adminQuestTemplateDetailQuery, questID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *QuestRepository) CreateTemplateForAdmin(ctx context.Context, input quest.AdminCreateTemplateInput) (*quest.AdminTemplateDetail, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("quest repository transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if input.QuestID == 0 {
		if _, err = tx.ExecContext(ctx, adminQuestTemplateAutoIDLockQuery, int64(2026070801)); err != nil {
			return nil, err
		}
		if err = tx.QueryRowContext(ctx, adminQuestTemplateNextIDQuery).Scan(&input.QuestID); err != nil {
			return nil, err
		}
	}

	preQuestIDsJSON, err := json.Marshal(input.PreQuestIDs)
	if err != nil {
		return nil, err
	}
	objectivesJSON, err := marshalAdminQuestObjectives(input.Objectives)
	if err != nil {
		return nil, err
	}
	rewardsJSON, err := marshalAdminQuestRewards(input.Rewards)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, insertAdminQuestTemplateQuery,
		input.QuestID, input.QuestType, input.Name, input.Title, input.Description,
		input.Chapter, input.SortOrder, input.AcceptMode, input.SubmitMode, input.AutoTrack,
		input.ClientIconID, input.StartNPCID, input.SubmitNPCID, input.AcceptAnimationKey, input.SubmitAnimationKey,
		input.MinPlayerLevel, preQuestIDsJSON, objectivesJSON, rewardsJSON, input.Status,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, quest.ErrAdminQuestConflict
		}
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.FindAdminTemplateDetailByQuestID(ctx, input.QuestID)
}

func (r *QuestRepository) UpdateTemplateForAdmin(ctx context.Context, questID uint64, input quest.AdminUpdateTemplateInput) (*quest.AdminTemplateDetail, error) {
	preQuestIDsJSON, err := json.Marshal(input.PreQuestIDs)
	if err != nil {
		return nil, err
	}
	objectivesJSON, err := marshalAdminQuestObjectives(input.Objectives)
	if err != nil {
		return nil, err
	}
	rewardsJSON, err := marshalAdminQuestRewards(input.Rewards)
	if err != nil {
		return nil, err
	}
	result, err := r.db.ExecContext(ctx, updateAdminQuestTemplateQuery,
		questID, input.QuestType, input.Name, input.Title, input.Description,
		input.Chapter, input.SortOrder, input.AcceptMode, input.SubmitMode, input.AutoTrack,
		input.ClientIconID, input.StartNPCID, input.SubmitNPCID, input.AcceptAnimationKey, input.SubmitAnimationKey,
		input.MinPlayerLevel, preQuestIDsJSON, objectivesJSON, rewardsJSON, input.Status,
	)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, quest.ErrAdminQuestTemplateNotFound
	}
	return r.FindAdminTemplateDetailByQuestID(ctx, questID)
}

func (r *QuestRepository) DeleteTemplateForAdmin(ctx context.Context, questID uint64) error {
	result, err := r.db.ExecContext(ctx, softDeleteAdminQuestTemplateQuery, questID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return quest.ErrAdminQuestTemplateNotFound
	}
	return nil
}

func (r *QuestRepository) ListPlayerQuestsForAdmin(ctx context.Context, query quest.AdminPlayerQuestListQuery) (*quest.AdminPlayerQuestList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 6)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.RecordID > 0 {
		conditions = append(conditions, "pq.id = "+nextArg(query.RecordID))
	}
	if query.PlayerID > 0 {
		conditions = append(conditions, "pq.player_id = "+nextArg(query.PlayerID))
	}
	if query.QuestID > 0 {
		conditions = append(conditions, "pq.quest_id = "+nextArg(query.QuestID))
	}
	if query.State != "" {
		conditions = append(conditions, "pq.state = "+nextArg(query.State))
	}
	if query.Tracked != nil {
		conditions = append(conditions, "pq.tracked = "+nextArg(*query.Tracked))
	}
	whereClause := joinWhere(conditions)
	countQuery := `
SELECT COUNT(1)
FROM player_quest pq
JOIN player p ON p.id = pq.player_id
LEFT JOIN quest_template qt ON qt.quest_id = pq.quest_id
` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminPlayerQuestListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY pq.id DESC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]quest.AdminPlayerQuestSummary, 0)
	for rows.Next() {
		item, err := scanAdminPlayerQuestSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &quest.AdminPlayerQuestList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *QuestRepository) FindAdminPlayerQuestDetailByRecordID(ctx context.Context, recordID uint64) (*quest.AdminPlayerQuestDetail, error) {
	item, err := scanAdminPlayerQuestDetail(r.db.QueryRowContext(ctx, adminPlayerQuestDetailQuery, recordID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	objectives, err := r.listAdminPlayerQuestObjectives(ctx, item.PlayerID, item.QuestID)
	if err != nil {
		return nil, err
	}
	item.Objectives = objectives
	return item, nil
}

func (r *QuestRepository) CreatePlayerQuestForAdmin(ctx context.Context, input quest.AdminCreatePlayerQuestInput) (*quest.AdminPlayerQuestDetail, error) {
	return r.withTransaction(ctx, func(tx *sql.Tx) (*quest.AdminPlayerQuestDetail, error) {
		if err := r.ensureAdminPlayerQuestReferences(ctx, tx, input.PlayerID, input.QuestID); err != nil {
			return nil, err
		}
		template, err := r.FindTemplateByQuestID(ctx, input.QuestID)
		if err != nil {
			return nil, err
		}
		var recordID int64
		err = tx.QueryRowContext(ctx, insertAdminPlayerQuestQuery, input.PlayerID, input.QuestID, template.QuestType, input.State, input.Tracked, input.RewardClaimed).Scan(&recordID)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, quest.ErrAdminQuestConflict
			}
			return nil, err
		}
		objectives := buildAdminPlayerQuestObjectives(template, input.Objectives, input.PlayerID, input.QuestID)
		if err := replaceAdminPlayerQuestObjectives(ctx, tx, input.PlayerID, input.QuestID, objectives); err != nil {
			return nil, err
		}
		if input.Tracked {
			if _, err := tx.ExecContext(ctx, clearTrackedQuestQuery, input.PlayerID); err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx, markTrackedQuestQuery, input.PlayerID, input.QuestID); err != nil {
				return nil, err
			}
		}
		return r.findAdminPlayerQuestDetailByRecordIDWithQuerier(ctx, tx, uint64(recordID))
	})
}

func (r *QuestRepository) UpdatePlayerQuestForAdmin(ctx context.Context, recordID uint64, input quest.AdminUpdatePlayerQuestInput) (*quest.AdminPlayerQuestDetail, error) {
	return r.withTransaction(ctx, func(tx *sql.Tx) (*quest.AdminPlayerQuestDetail, error) {
		current, err := scanAdminPlayerQuestDetail(tx.QueryRowContext(ctx, adminPlayerQuestDetailQuery, recordID))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, quest.ErrAdminPlayerQuestNotFound
		}
		if err != nil {
			return nil, err
		}
		if err := r.ensureAdminPlayerQuestReferences(ctx, tx, input.PlayerID, input.QuestID); err != nil {
			return nil, err
		}
		template, err := r.FindTemplateByQuestID(ctx, input.QuestID)
		if err != nil {
			return nil, err
		}
		result, err := tx.ExecContext(ctx, updateAdminPlayerQuestQuery, recordID, input.PlayerID, input.QuestID, template.QuestType, input.State, input.Tracked, input.RewardClaimed)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, quest.ErrAdminQuestConflict
			}
			return nil, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rowsAffected == 0 {
			return nil, quest.ErrAdminPlayerQuestNotFound
		}
		if _, err := tx.ExecContext(ctx, deleteAdminPlayerQuestObjectivesQuery, current.PlayerID, current.QuestID); err != nil {
			return nil, err
		}
		objectives := buildAdminPlayerQuestObjectives(template, input.Objectives, input.PlayerID, input.QuestID)
		if err := replaceAdminPlayerQuestObjectives(ctx, tx, input.PlayerID, input.QuestID, objectives); err != nil {
			return nil, err
		}
		if input.Tracked {
			if _, err := tx.ExecContext(ctx, clearTrackedQuestQuery, input.PlayerID); err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx, markTrackedQuestQuery, input.PlayerID, input.QuestID); err != nil {
				return nil, err
			}
		}
		return r.findAdminPlayerQuestDetailByRecordIDWithQuerier(ctx, tx, recordID)
	})
}

func (r *QuestRepository) DeletePlayerQuestForAdmin(ctx context.Context, recordID uint64) error {
	_, err := r.withTransaction(ctx, func(tx *sql.Tx) (*quest.AdminPlayerQuestDetail, error) {
		current, err := scanAdminPlayerQuestDetail(tx.QueryRowContext(ctx, adminPlayerQuestDetailQuery, recordID))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, quest.ErrAdminPlayerQuestNotFound
		}
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, deleteAdminPlayerQuestObjectivesQuery, current.PlayerID, current.QuestID); err != nil {
			return nil, err
		}
		result, err := tx.ExecContext(ctx, deleteAdminPlayerQuestQuery, recordID)
		if err != nil {
			return nil, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rowsAffected == 0 {
			return nil, quest.ErrAdminPlayerQuestNotFound
		}
		return nil, nil
	})
	return err
}

type queryRowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type adminQuestQuerier interface {
	queryRowQuerier
	queryContextQuerier
}

func (r *QuestRepository) findAdminPlayerQuestDetailByRecordIDWithQuerier(ctx context.Context, querier adminQuestQuerier, recordID uint64) (*quest.AdminPlayerQuestDetail, error) {
	item, err := scanAdminPlayerQuestDetail(querier.QueryRowContext(ctx, adminPlayerQuestDetailQuery, recordID))
	if err != nil {
		return nil, err
	}
	objectives, err := r.listAdminPlayerQuestObjectivesWithQuerier(ctx, querier, item.PlayerID, item.QuestID)
	if err != nil {
		return nil, err
	}
	item.Objectives = objectives
	return item, nil
}

func (r *QuestRepository) listAdminPlayerQuestObjectives(ctx context.Context, playerID uint64, questID uint64) ([]quest.AdminPlayerObjectiveInput, error) {
	return r.listAdminPlayerQuestObjectivesWithQuerier(ctx, r.db, playerID, questID)
}

type queryContextQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (r *QuestRepository) listAdminPlayerQuestObjectivesWithQuerier(ctx context.Context, querier queryContextQuerier, playerID uint64, questID uint64) ([]quest.AdminPlayerObjectiveInput, error) {
	rows, err := querier.QueryContext(ctx, adminPlayerQuestObjectivesQuery, playerID, questID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]quest.AdminPlayerObjectiveInput, 0)
	for rows.Next() {
		var (
			item         quest.AdminPlayerObjectiveInput
			currentValue int64
			targetValue  int64
		)
		if err := rows.Scan(&item.ObjectiveID, &item.Description, &currentValue, &targetValue, &item.Completed); err != nil {
			return nil, err
		}
		item.CurrentValue = uint32(currentValue)
		item.TargetValue = uint32(targetValue)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *QuestRepository) ensureAdminPlayerQuestReferences(ctx context.Context, querier queryRowQuerier, playerID uint64, questID uint64) error {
	var playerCount int64
	if err := querier.QueryRowContext(ctx, adminQuestPlayerExistsQuery, playerID).Scan(&playerCount); err != nil {
		return err
	}
	if playerCount == 0 {
		return quest.ErrAdminPlayerQuestNotFound
	}
	var questCount int64
	if err := querier.QueryRowContext(ctx, adminQuestTemplateExistsQuery, questID).Scan(&questCount); err != nil {
		return err
	}
	if questCount == 0 {
		return quest.ErrAdminQuestTemplateNotFound
	}
	return nil
}

func (r *QuestRepository) withTransaction(ctx context.Context, fn func(tx *sql.Tx) (*quest.AdminPlayerQuestDetail, error)) (*quest.AdminPlayerQuestDetail, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("quest repository transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	result, err := fn(tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func replaceAdminPlayerQuestObjectives(ctx context.Context, execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, playerID uint64, questID uint64, objectives []playerQuestObjectiveRow) error {
	if _, err := execer.ExecContext(ctx, deleteAdminPlayerQuestObjectivesQuery, playerID, questID); err != nil {
		return err
	}
	for _, objective := range objectives {
		if _, err := execer.ExecContext(ctx, insertAdminPlayerQuestObjectiveQuery,
			playerID, questID, objective.ObjectiveID, objective.EventType, objective.Description,
			objective.CurrentValue, objective.TargetValue, objective.Completed,
		); err != nil {
			return err
		}
	}
	return nil
}

type playerQuestObjectiveRow struct {
	ObjectiveID  uint64
	EventType    string
	Description  string
	CurrentValue uint32
	TargetValue  uint32
	Completed    bool
}

func buildAdminPlayerQuestObjectives(template *quest.Template, input []quest.AdminPlayerObjectiveInput, playerID uint64, questID uint64) []playerQuestObjectiveRow {
	templateObjectiveMap := make(map[uint64]quest.ObjectiveTemplate)
	if template != nil {
		for _, objective := range template.Objectives {
			templateObjectiveMap[objective.ObjectiveID] = objective
		}
	}
	if len(input) == 0 && template != nil {
		rows := make([]playerQuestObjectiveRow, 0, len(template.Objectives))
		for _, objective := range template.Objectives {
			rows = append(rows, playerQuestObjectiveRow{
				ObjectiveID:  objective.ObjectiveID,
				EventType:    objective.EventType,
				Description:  objective.Description,
				CurrentValue: 0,
				TargetValue:  objective.TargetValue,
				Completed:    false,
			})
		}
		return rows
	}
	rows := make([]playerQuestObjectiveRow, 0, len(input))
	for _, objective := range input {
		definition := templateObjectiveMap[objective.ObjectiveID]
		description := strings.TrimSpace(objective.Description)
		if description == "" {
			description = definition.Description
		}
		targetValue := objective.TargetValue
		if targetValue == 0 && definition.TargetValue > 0 {
			targetValue = definition.TargetValue
		}
		rows = append(rows, playerQuestObjectiveRow{
			ObjectiveID:  objective.ObjectiveID,
			EventType:    definition.EventType,
			Description:  description,
			CurrentValue: objective.CurrentValue,
			TargetValue:  targetValue,
			Completed:    objective.Completed,
		})
	}
	return rows
}

func scanAdminQuestTemplateSummary(rows *sql.Rows) (quest.AdminTemplateSummary, error) {
	var (
		item      quest.AdminTemplateSummary
		chapter   int64
		sortOrder int64
		minLevel  int64
		status    int64
	)
	if err := rows.Scan(&item.QuestID, &item.Name, &item.QuestType, &item.Title, &chapter, &sortOrder, &item.AcceptMode, &item.SubmitMode, &item.AutoTrack, &item.ClientIconID, &minLevel, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return quest.AdminTemplateSummary{}, err
	}
	item.Chapter = uint32(chapter)
	item.SortOrder = uint32(sortOrder)
	item.MinPlayerLevel = uint32(minLevel)
	item.Status = uint32(status)
	item.StatusText = quest.AdminQuestTemplateStatusText(item.Status)
	return item, nil
}

func scanAdminQuestTemplateDetail(row *sql.Row) (*quest.AdminTemplateDetail, error) {
	var (
		item          quest.AdminTemplateDetail
		chapter       int64
		sortOrder     int64
		minLevel      int64
		status        int64
		preQuestRaw   []byte
		objectivesRaw []byte
		rewardsRaw    []byte
	)
	if err := row.Scan(&item.QuestID, &item.Name, &item.QuestType, &item.Title, &item.Description, &chapter, &sortOrder, &item.AcceptMode, &item.SubmitMode, &item.AutoTrack, &item.ClientIconID, &item.StartNPCID, &item.SubmitNPCID, &item.AcceptAnimationKey, &item.SubmitAnimationKey, &minLevel, &status, &preQuestRaw, &objectivesRaw, &rewardsRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.Chapter = uint32(chapter)
	item.SortOrder = uint32(sortOrder)
	item.MinPlayerLevel = uint32(minLevel)
	item.Status = uint32(status)
	item.StatusText = quest.AdminQuestTemplateStatusText(item.Status)
	if len(preQuestRaw) > 0 {
		if err := json.Unmarshal(preQuestRaw, &item.PreQuestIDs); err != nil {
			return nil, err
		}
	}
	objectives, err := unmarshalAdminQuestObjectives(objectivesRaw)
	if err != nil {
		return nil, err
	}
	item.Objectives = objectives
	rewards, err := unmarshalAdminQuestRewards(rewardsRaw)
	if err != nil {
		return nil, err
	}
	item.Rewards = rewards
	return &item, nil
}

func scanAdminPlayerQuestSummary(rows *sql.Rows) (quest.AdminPlayerQuestSummary, error) {
	var item quest.AdminPlayerQuestSummary
	if err := rows.Scan(&item.RecordID, &item.PlayerID, &item.PlayerName, &item.QuestID, &item.QuestTitle, &item.QuestType, &item.State, &item.Tracked, &item.RewardClaimed, &item.AcceptedAt, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return quest.AdminPlayerQuestSummary{}, err
	}
	return item, nil
}

func scanAdminPlayerQuestDetail(row *sql.Row) (*quest.AdminPlayerQuestDetail, error) {
	var item quest.AdminPlayerQuestDetail
	if err := row.Scan(&item.RecordID, &item.PlayerID, &item.PlayerName, &item.QuestID, &item.QuestTitle, &item.QuestType, &item.State, &item.Tracked, &item.RewardClaimed, &item.AcceptedAt, &item.CompletedAt, &item.SubmittedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

type adminObjectiveJSON struct {
	ObjectiveID    uint64                     `json:"objective_id"`
	EventType      string                     `json:"event_type"`
	Description    string                     `json:"description"`
	Target         uint32                     `json:"target"`
	TargetSelector map[string]any             `json:"target_selector,omitempty"`
	Guide          *quest.ObjectiveGuideInput `json:"guide,omitempty"`
}

func marshalAdminQuestObjectives(input []quest.AdminObjectiveInput) ([]byte, error) {
	payload := make([]adminObjectiveJSON, 0, len(input))
	for _, item := range input {
		payload = append(payload, adminObjectiveJSON{
			ObjectiveID:    item.ObjectiveID,
			EventType:      strings.TrimSpace(item.EventType),
			Description:    strings.TrimSpace(item.Description),
			Target:         item.TargetValue,
			TargetSelector: item.TargetSelector,
			Guide:          item.Guide,
		})
	}
	return json.Marshal(payload)
}

func unmarshalAdminQuestObjectives(raw []byte) ([]quest.AdminObjectiveInput, error) {
	if len(raw) == 0 {
		return []quest.AdminObjectiveInput{}, nil
	}
	var payload []adminObjectiveJSON
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	result := make([]quest.AdminObjectiveInput, 0, len(payload))
	for _, item := range payload {
		result = append(result, quest.AdminObjectiveInput{
			ObjectiveID:    item.ObjectiveID,
			EventType:      item.EventType,
			Description:    item.Description,
			TargetValue:    item.Target,
			TargetSelector: item.TargetSelector,
			Guide:          item.Guide,
		})
	}
	return result, nil
}

func marshalAdminQuestRewards(input []quest.AdminRewardInput) ([]byte, error) {
	payload := make([]quest.AdminRewardInput, 0, len(input))
	for _, item := range input {
		normalized := item.Normalize()
		if normalized.Type != "exp" && normalized.Type != "item" && normalized.Type != "gold" {
			continue
		}
		if normalized.Type == "exp" && normalized.Value == 0 {
			continue
		}
		if normalized.Type == "gold" && normalized.Value == 0 {
			continue
		}
		if normalized.Type == "item" && (normalized.ItemID == 0 || normalized.Count == 0) {
			continue
		}
		payload = append(payload, normalized)
	}
	return json.Marshal(payload)
}

func unmarshalAdminQuestRewards(raw []byte) ([]quest.AdminRewardInput, error) {
	if len(raw) == 0 {
		return []quest.AdminRewardInput{}, nil
	}
	var payload []quest.AdminRewardInput
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	result := make([]quest.AdminRewardInput, 0, len(payload))
	for _, item := range payload {
		normalized := item.Normalize()
		if normalized.Type != "exp" && normalized.Type != "item" && normalized.Type != "gold" {
			continue
		}
		result = append(result, normalized)
	}
	return result, nil
}

func joinWhere(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(conditions, " AND ")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
