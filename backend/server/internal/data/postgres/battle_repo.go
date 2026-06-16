package postgres

import (
	"context"

	"pocket-pet-remake/server/internal/module/battle"
)

type BattleRepository struct {
	db DBTX
}

func NewBattleRepository(db DBTX) *BattleRepository {
	return &BattleRepository{db: db}
}

const insertBattleRewardRecordQuery = `
INSERT INTO battle_record (
  battle_id,
  player_id,
  battle_type,
  result,
  reward_gold,
  reward_exp,
  payload_json
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (battle_id, player_id) DO NOTHING
`

const deleteBattleRewardRecordQuery = `
DELETE FROM battle_record
WHERE battle_id = $1 AND player_id = $2
`

const maxBattleRewardBattleIDQuery = `
SELECT COALESCE(MAX(battle_id), 0)
FROM battle_record
`

func (r *BattleRepository) CreateRewardRecord(ctx context.Context, record battle.RewardRecord) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		insertBattleRewardRecordQuery,
		record.BattleID,
		record.PlayerID,
		record.BattleType,
		record.Result,
		record.RewardGold,
		record.RewardExp,
		record.PayloadJSON,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *BattleRepository) DeleteRewardRecord(ctx context.Context, battleID uint64, playerID uint64) error {
	if battleID == 0 || playerID == 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, deleteBattleRewardRecordQuery, battleID, playerID)
	return err
}

func (r *BattleRepository) MaxRewardBattleID(ctx context.Context) (uint64, error) {
	var maxBattleID int64
	err := r.db.QueryRowContext(ctx, maxBattleRewardBattleIDQuery).Scan(&maxBattleID)
	if err != nil {
		return 0, err
	}
	if maxBattleID < 0 {
		return 0, nil
	}
	return uint64(maxBattleID), nil
}
