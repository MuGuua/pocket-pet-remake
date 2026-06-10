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
