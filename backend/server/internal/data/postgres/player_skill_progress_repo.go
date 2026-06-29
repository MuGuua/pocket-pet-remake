package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pocket-pet-remake/server/internal/module/playerskill"
)

// PlayerSkillProgressRepository 映射 player_skill_progress 表。
type PlayerSkillProgressRepository struct {
	db DBTX
}

// NewPlayerSkillProgressRepository 构造玩家技能进度仓储。
func NewPlayerSkillProgressRepository(db DBTX) *PlayerSkillProgressRepository {
	return &PlayerSkillProgressRepository{db: db}
}

const listPlayerSkillProgressQuery = `
SELECT player_id, skill_id, skill_exp, skill_level, is_learned, learned_at
FROM player_skill_progress
WHERE player_id = $1
ORDER BY skill_id ASC
`

func (r *PlayerSkillProgressRepository) ListByPlayerID(ctx context.Context, playerID uint64) ([]playerskill.Progress, error) {
	if playerID == 0 {
		return []playerskill.Progress{}, nil
	}
	rows, err := r.db.QueryContext(ctx, listPlayerSkillProgressQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]playerskill.Progress, 0, 8)
	for rows.Next() {
		item, scanErr := scanPlayerSkillProgressRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PlayerSkillProgressRepository) UpsertBattleUpdates(ctx context.Context, playerID uint64, updates []playerskill.BattleUseUpdate) error {
	if playerID == 0 || len(updates) == 0 {
		return nil
	}
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)
	for _, update := range updates {
		if update.SkillID == 0 || update.ExpGained == 0 {
			return playerskill.ErrInvalidSkillProgressInput
		}
		finalLevel := update.FinalLevel
		if finalLevel == 0 {
			finalLevel = 1
		}
		if update.NewlyLearned {
			_, execErr := tx.ExecContext(ctx, upsertLearnedPlayerSkillProgressQuery,
				playerID, update.SkillID, update.FinalExp, finalLevel, time.Now().UTC(),
			)
			if execErr != nil {
				return execErr
			}
			continue
		}
		_, execErr := tx.ExecContext(ctx, upsertLearningPlayerSkillProgressQuery,
			playerID, update.SkillID, update.FinalExp, finalLevel,
		)
		if execErr != nil {
			return execErr
		}
	}
	return tx.Commit()
}

const upsertLearningPlayerSkillProgressQuery = `
INSERT INTO player_skill_progress (
  player_id, skill_id, skill_exp, skill_level, is_learned
) VALUES (
  $1, $2, $3, $4, FALSE
)
ON CONFLICT (player_id, skill_id) DO UPDATE SET
  skill_exp = EXCLUDED.skill_exp,
  skill_level = GREATEST(player_skill_progress.skill_level, EXCLUDED.skill_level),
  updated_at = CURRENT_TIMESTAMP
`

const upsertLearnedPlayerSkillProgressQuery = `
INSERT INTO player_skill_progress (
  player_id, skill_id, skill_exp, skill_level, is_learned, learned_at
) VALUES (
  $1, $2, $3, $4, TRUE, $5
)
ON CONFLICT (player_id, skill_id) DO UPDATE SET
  skill_exp = EXCLUDED.skill_exp,
  skill_level = GREATEST(player_skill_progress.skill_level, EXCLUDED.skill_level),
  is_learned = TRUE,
  learned_at = COALESCE(player_skill_progress.learned_at, EXCLUDED.learned_at),
  updated_at = CURRENT_TIMESTAMP
`

func scanPlayerSkillProgressRow(scanner interface {
	Scan(dest ...any) error
}) (playerskill.Progress, error) {
	var item playerskill.Progress
	var skillID int64
	var skillExp int64
	var skillLevel int64
	var isLearned bool
	var learnedAt sql.NullTime
	if err := scanner.Scan(&item.PlayerID, &skillID, &skillExp, &skillLevel, &isLearned, &learnedAt); err != nil {
		return playerskill.Progress{}, err
	}
	item.SkillID = uint32(skillID)
	item.SkillExp = uint32(skillExp)
	item.SkillLevel = uint32(skillLevel)
	item.IsLearned = isLearned
	if learnedAt.Valid {
		ts := learnedAt.Time
		item.LearnedAt = &ts
	}
	return item, nil
}
