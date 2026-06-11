package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"pocket-pet-remake/server/internal/module/quest"
)

type QuestRepository struct {
	db DBTX
}

func NewQuestRepository(db DBTX) *QuestRepository {
	return &QuestRepository{db: db}
}

const listQuestTemplatesQuery = `
SELECT
  quest_id,
  quest_type,
  title,
  description,
  accept_mode,
  submit_mode,
  start_npc_id,
  submit_npc_id,
  auto_track,
  pre_quest_ids,
  objectives_json
FROM quest_template
WHERE status = 1
ORDER BY chapter ASC, sort_order ASC, quest_id ASC
`

const findQuestTemplateByIDQuery = `
SELECT
  quest_id,
  quest_type,
  title,
  description,
  accept_mode,
  submit_mode,
  start_npc_id,
  submit_npc_id,
  auto_track,
  pre_quest_ids,
  objectives_json
FROM quest_template
WHERE quest_id = $1 AND status = 1
LIMIT 1
`

const listPlayerQuestsQuery = `
SELECT player_id, quest_id, state, tracked
FROM player_quest
WHERE player_id = $1
`

const listPlayerObjectivesQuery = `
SELECT player_id, quest_id, objective_id, description, current_value, target_value, completed
FROM player_quest_objective
WHERE player_id = $1
ORDER BY quest_id ASC, objective_id ASC
`

const upsertPlayerQuestQuery = `
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
  COALESCE((SELECT quest_type FROM quest_template WHERE quest_id = $2), 'MAIN'),
  $3::VARCHAR(32),
  $4,
  CASE WHEN $3::VARCHAR(32) = 'ACCEPTED' THEN CURRENT_TIMESTAMP ELSE NULL END,
  CASE WHEN $3::VARCHAR(32) = 'COMPLETED' THEN CURRENT_TIMESTAMP ELSE NULL END,
  CASE WHEN $3::VARCHAR(32) = 'COMPLETED' THEN CURRENT_TIMESTAMP ELSE NULL END,
  CASE WHEN $3::VARCHAR(32) = 'COMPLETED' THEN TRUE ELSE FALSE END
)
ON CONFLICT (player_id, quest_id) DO UPDATE SET
  state = EXCLUDED.state,
  tracked = EXCLUDED.tracked,
  accepted_at = CASE
    WHEN EXCLUDED.state = 'ACCEPTED' AND player_quest.accepted_at IS NULL THEN CURRENT_TIMESTAMP
    ELSE player_quest.accepted_at
  END,
  completed_at = CASE
    WHEN EXCLUDED.state = 'COMPLETED' THEN CURRENT_TIMESTAMP
    ELSE player_quest.completed_at
  END,
  submitted_at = CASE
    WHEN EXCLUDED.state = 'COMPLETED' THEN CURRENT_TIMESTAMP
    ELSE player_quest.submitted_at
  END,
  reward_claimed = CASE
    WHEN EXCLUDED.state = 'COMPLETED' THEN TRUE
    ELSE player_quest.reward_claimed
  END
`

const deletePlayerObjectivesQuery = `
DELETE FROM player_quest_objective
WHERE player_id = $1 AND quest_id = $2
`

const insertPlayerObjectiveQuery = `
INSERT INTO player_quest_objective (
  player_id,
  quest_id,
  objective_id,
  event_type,
  description,
  current_value,
  target_value,
  completed
) VALUES ($1, $2, $3, '', $4, $5, $6, $7)
`

const clearTrackedQuestQuery = `
UPDATE player_quest
SET tracked = FALSE
WHERE player_id = $1
`

const markTrackedQuestQuery = `
UPDATE player_quest
SET tracked = TRUE
WHERE player_id = $1 AND quest_id = $2
`

func (r *QuestRepository) ListTemplates(ctx context.Context) ([]quest.Template, error) {
	rows, err := r.db.QueryContext(ctx, listQuestTemplatesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []quest.Template{}
	for rows.Next() {
		value, err := scanQuestTemplate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *QuestRepository) FindTemplateByQuestID(ctx context.Context, questID uint64) (*quest.Template, error) {
	row := r.db.QueryRowContext(ctx, findQuestTemplateByIDQuery, questID)
	value, err := scanQuestTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *QuestRepository) ListPlayerQuestsByPlayerID(ctx context.Context, playerID uint64) ([]quest.PlayerQuest, error) {
	rows, err := r.db.QueryContext(ctx, listPlayerQuestsQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []quest.PlayerQuest{}
	for rows.Next() {
		var value quest.PlayerQuest
		if err := rows.Scan(&value.PlayerID, &value.QuestID, &value.State, &value.Tracked); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *QuestRepository) ListPlayerObjectivesByPlayerID(ctx context.Context, playerID uint64) ([]quest.PlayerObjective, error) {
	rows, err := r.db.QueryContext(ctx, listPlayerObjectivesQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []quest.PlayerObjective{}
	for rows.Next() {
		var value quest.PlayerObjective
		if err := rows.Scan(
			&value.PlayerID,
			&value.QuestID,
			&value.ObjectiveID,
			&value.Description,
			&value.CurrentValue,
			&value.TargetValue,
			&value.Completed,
		); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *QuestRepository) UpsertPlayerQuest(ctx context.Context, value quest.PlayerQuest) error {
	_, err := r.db.ExecContext(ctx, upsertPlayerQuestQuery, value.PlayerID, value.QuestID, value.State, value.Tracked)
	return err
}

func (r *QuestRepository) ReplacePlayerObjectives(ctx context.Context, playerID uint64, questID uint64, objectives []quest.PlayerObjective) error {
	if _, err := r.db.ExecContext(ctx, deletePlayerObjectivesQuery, playerID, questID); err != nil {
		return err
	}
	for _, objective := range objectives {
		if _, err := r.db.ExecContext(
			ctx,
			insertPlayerObjectiveQuery,
			playerID,
			questID,
			objective.ObjectiveID,
			objective.Description,
			objective.CurrentValue,
			objective.TargetValue,
			objective.Completed,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *QuestRepository) SetTrackedQuest(ctx context.Context, playerID uint64, questID uint64) error {
	if _, err := r.db.ExecContext(ctx, clearTrackedQuestQuery, playerID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, markTrackedQuestQuery, playerID, questID)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuestTemplate(scanner rowScanner) (quest.Template, error) {
	var (
		value          quest.Template
		preQuestRaw    []byte
		objectivesRaw  []byte
		objectivesJSON []struct {
			ObjectiveID    uint64         `json:"objective_id"`
			EventType      string         `json:"event_type"`
			Description    string         `json:"description"`
			Target         uint32         `json:"target"`
			TargetSelector map[string]any `json:"target_selector"`
		}
	)

	err := scanner.Scan(
		&value.QuestID,
		&value.QuestType,
		&value.Title,
		&value.Description,
		&value.AcceptMode,
		&value.SubmitMode,
		&value.StartNPCID,
		&value.SubmitNPCID,
		&value.AutoTrack,
		&preQuestRaw,
		&objectivesRaw,
	)
	if err != nil {
		return quest.Template{}, err
	}
	if len(preQuestRaw) > 0 {
		if err := json.Unmarshal(preQuestRaw, &value.PreQuestIDs); err != nil {
			return quest.Template{}, err
		}
	}
	if len(objectivesRaw) > 0 {
		if err := json.Unmarshal(objectivesRaw, &objectivesJSON); err != nil {
			return quest.Template{}, err
		}
	}
	value.Objectives = make([]quest.ObjectiveTemplate, 0, len(objectivesJSON))
	for _, objective := range objectivesJSON {
		value.Objectives = append(value.Objectives, quest.ObjectiveTemplate{
			ObjectiveID:    objective.ObjectiveID,
			EventType:      objective.EventType,
			Description:    objective.Description,
			TargetValue:    objective.Target,
			TargetSelector: objective.TargetSelector,
		})
	}
	return value, nil
}
