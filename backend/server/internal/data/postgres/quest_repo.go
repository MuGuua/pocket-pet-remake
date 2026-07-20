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
  client_icon_id,
  title,
  description,
  completion_prompt_text,
  accept_mode,
  submit_mode,
  start_npc_id,
  submit_npc_id,
  accept_animation_key,
  submit_animation_key,
  auto_track,
  pre_quest_ids,
  objectives_json,
  rewards_json
FROM quest_template
WHERE status = 1
ORDER BY chapter ASC, sort_order ASC, quest_id ASC
`

const findQuestTemplateByIDQuery = `
SELECT
  quest_id,
  quest_type,
  client_icon_id,
  title,
  description,
  completion_prompt_text,
  accept_mode,
  submit_mode,
  start_npc_id,
  submit_npc_id,
  accept_animation_key,
  submit_animation_key,
  auto_track,
  pre_quest_ids,
  objectives_json,
  rewards_json
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

const revertCompletedQuestToReadyQuery = `
UPDATE player_quest
SET state = 'READY_TO_SUBMIT',
    completed_at = NULL,
    submitted_at = NULL,
    reward_claimed = FALSE
WHERE player_id = $1 AND quest_id = $2 AND state = 'COMPLETED'
`

const findQuestGuideNPCNameQuery = `
SELECT display_name
FROM world_entity_definition
WHERE entity_id = $1 AND status = 1
LIMIT 1
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
		if err := r.enrichQuestGuideNPCNames(ctx, &value); err != nil {
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
	if err := r.enrichQuestGuideNPCNames(ctx, &value); err != nil {
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

func (r *QuestRepository) RevertCompletedQuestToReady(ctx context.Context, playerID uint64, questID uint64) error {
	_, err := r.db.ExecContext(ctx, revertCompletedQuestToReadyQuery, playerID, questID)
	return err
}

func (r *QuestRepository) enrichQuestGuideNPCNames(ctx context.Context, template *quest.Template) error {
	if template == nil {
		return nil
	}
	npcNames := map[uint64]string{}
	for objectiveIndex := range template.Objectives {
		guide := template.Objectives[objectiveIndex].Guide
		if guide == nil || guide.NPCID == 0 || guide.NPCName != "" {
			continue
		}
		name, ok, err := r.findQuestGuideNPCName(ctx, guide.NPCID, npcNames)
		if err != nil {
			return err
		}
		if ok {
			guide.NPCName = name
		}
	}
	return nil
}

func (r *QuestRepository) findQuestGuideNPCName(ctx context.Context, npcID uint64, cache map[uint64]string) (string, bool, error) {
	if npcID == 0 {
		return "", false, nil
	}
	if value, ok := cache[npcID]; ok {
		return value, value != "", nil
	}
	var name string
	err := r.db.QueryRowContext(ctx, findQuestGuideNPCNameQuery, npcID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		cache[npcID] = ""
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	cache[npcID] = name
	return name, name != "", nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuestTemplate(scanner rowScanner) (quest.Template, error) {
	var (
		value          quest.Template
		preQuestRaw    []byte
		objectivesRaw  []byte
		rewardsRaw     []byte
		objectivesJSON []struct {
			ObjectiveID    uint64                     `json:"objective_id"`
			EventType      string                     `json:"event_type"`
			Description    string                     `json:"description"`
			Target         uint32                     `json:"target"`
			TargetSelector map[string]any             `json:"target_selector"`
			Guide          *quest.ObjectiveGuideInput `json:"guide"`
		}
		rewardsJSON []quest.Reward
	)

	err := scanner.Scan(
		&value.QuestID,
		&value.QuestType,
		&value.ClientIconID,
		&value.Title,
		&value.Description,
		&value.CompletionPromptText,
		&value.AcceptMode,
		&value.SubmitMode,
		&value.StartNPCID,
		&value.SubmitNPCID,
		&value.AcceptAnimationKey,
		&value.SubmitAnimationKey,
		&value.AutoTrack,
		&preQuestRaw,
		&objectivesRaw,
		&rewardsRaw,
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
	if len(rewardsRaw) > 0 {
		if err := json.Unmarshal(rewardsRaw, &rewardsJSON); err != nil {
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
			Guide:          objective.Guide,
		})
	}
	value.Rewards = append([]quest.Reward{}, rewardsJSON...)
	return value, nil
}
