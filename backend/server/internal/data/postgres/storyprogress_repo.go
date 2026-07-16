package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"pocket-pet-remake/server/internal/module/storyprogress"
)

// StoryProgressRepository 负责玩家剧情标记和场景触发器持久化。
// 这里所有判断都直接依赖数据库，避免客户端本地缓存决定剧情是否已经播放。
type StoryProgressRepository struct {
	db DBTX
}

// NewStoryProgressRepository 构造 PostgreSQL 版剧情进度仓储。
func NewStoryProgressRepository(db DBTX) *StoryProgressRepository {
	return &StoryProgressRepository{db: db}
}

const findPendingSceneTriggerQuery = `
SELECT
  trigger_code,
  scene_id,
  client_animation_key,
  prompt_text,
  block_movement,
  effect_accept_quest_id,
  effect_set_flags,
  once_flag_key
FROM scene_entry_trigger t
WHERE t.scene_id = $2
  AND t.status = 1
  AND NOT EXISTS (
    SELECT 1 FROM player_story_flag f
    WHERE f.player_id = $1 AND f.flag_key = t.once_flag_key
  )
  AND (
    t.required_quest_id = 0
    OR EXISTS (
      SELECT 1 FROM player_quest pq
      WHERE pq.player_id = $1
        AND pq.quest_id = t.required_quest_id
        AND (t.required_quest_state = '' OR pq.state = t.required_quest_state)
    )
  )
  AND (
    t.forbidden_quest_id = 0
    OR NOT EXISTS (
      SELECT 1 FROM player_quest pq
      WHERE pq.player_id = $1
        AND pq.quest_id = t.forbidden_quest_id
        AND (t.forbidden_quest_state = '' OR pq.state = t.forbidden_quest_state)
    )
  )
ORDER BY t.priority DESC, t.id ASC
LIMIT 1
`

const findSceneTriggerByCodeQuery = `
SELECT
  trigger_code,
  scene_id,
  client_animation_key,
  prompt_text,
  block_movement,
  effect_accept_quest_id,
  effect_set_flags,
  once_flag_key
FROM scene_entry_trigger
WHERE trigger_code = $1 AND status = 1
LIMIT 1
`

const upsertPlayerStoryFlagQuery = `
INSERT INTO player_story_flag (player_id, flag_key, flag_value)
VALUES ($1, $2, '1')
ON CONFLICT (player_id, flag_key) DO UPDATE SET
  flag_value = EXCLUDED.flag_value,
  updated_at = CURRENT_TIMESTAMP
`

// FindPendingSceneTrigger 返回玩家进入场景时尚未消费的一次性触发器。
func (r *StoryProgressRepository) FindPendingSceneTrigger(ctx context.Context, playerID uint64, sceneID uint32) (*storyprogress.SceneTrigger, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx, findPendingSceneTriggerQuery, playerID, sceneID)
	trigger, _, err := scanSceneTrigger(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return trigger, nil
}

// CompleteSceneTrigger 写入 once flag 和效果 flag，保证同一剧情对同一玩家只播放一次。
func (r *StoryProgressRepository) CompleteSceneTrigger(ctx context.Context, playerID uint64, triggerCode string) (*storyprogress.SceneTrigger, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx, findSceneTriggerByCodeQuery, triggerCode)
	trigger, onceFlagKey, err := scanSceneTrigger(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	flags := make([]string, 0, len(trigger.EffectSetFlags)+1)
	if onceFlagKey != "" {
		flags = append(flags, onceFlagKey)
	}
	flags = append(flags, trigger.EffectSetFlags...)
	for _, flagKey := range flags {
		if flagKey == "" {
			continue
		}
		if _, err := r.db.ExecContext(ctx, upsertPlayerStoryFlagQuery, playerID, flagKey); err != nil {
			return nil, err
		}
	}
	return trigger, nil
}

type sceneTriggerScanner interface {
	Scan(dest ...any) error
}

func scanSceneTrigger(scanner sceneTriggerScanner) (*storyprogress.SceneTrigger, string, error) {
	var (
		trigger             storyprogress.SceneTrigger
		sceneID             int64
		effectAcceptQuestID int64
		effectFlagsJSON     []byte
		onceFlagKey         string
	)
	if err := scanner.Scan(
		&trigger.TriggerCode,
		&sceneID,
		&trigger.ClientAnimationKey,
		&trigger.PromptText,
		&trigger.BlockMovement,
		&effectAcceptQuestID,
		&effectFlagsJSON,
		&onceFlagKey,
	); err != nil {
		return nil, "", err
	}
	trigger.SceneID = uint32(sceneID)
	trigger.EffectAcceptQuestID = uint64(effectAcceptQuestID)
	trigger.EffectSetFlags = parseStoryFlagList(effectFlagsJSON)
	return &trigger, onceFlagKey, nil
}

func parseStoryFlagList(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	values := []string{}
	if err := json.Unmarshal(raw, &values); err != nil {
		return []string{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}
