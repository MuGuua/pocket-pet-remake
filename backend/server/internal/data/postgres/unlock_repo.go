package postgres

import (
	"context"
	"database/sql"
	"errors"

	"pocket-pet-remake/server/internal/module/unlock"
)

// UnlockRepository 把正式玩法里的功能解锁标记映射到 player_feature_unlock。
// 第一版只要求可靠幂等落库，后续客户端需要读取时再补查询接口。
type UnlockRepository struct {
	db DBTX
}

func NewUnlockRepository(db DBTX) *UnlockRepository {
	return &UnlockRepository{db: db}
}

const grantRuntimeFeatureQuery = `
INSERT INTO player_feature_unlock (
  player_id,
  feature_id,
  reason_type,
  reason_ref_id,
  operator_type,
  operator_id
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (player_id, feature_id) DO NOTHING
RETURNING unlocked_at
`

const findRuntimeFeatureQuery = `
SELECT unlocked_at
FROM player_feature_unlock
WHERE player_id = $1
  AND feature_id = $2
LIMIT 1
`

func (r *UnlockRepository) GrantRuntimeFeature(ctx context.Context, playerID uint64, featureID uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*unlock.RuntimeGrantResult, error) {
	var unlockedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, grantRuntimeFeatureQuery, playerID, featureID, reasonType, reasonRefID, operatorType, operatorID).Scan(&unlockedAt)
	granted := true
	if errors.Is(err, sql.ErrNoRows) {
		granted = false
		if err = r.db.QueryRowContext(ctx, findRuntimeFeatureQuery, playerID, featureID).Scan(&unlockedAt); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &unlock.RuntimeGrantResult{
		Feature: unlock.FeatureRecord{
			PlayerID:   playerID,
			FeatureID:  featureID,
			UnlockedAt: unlockedAt.Time,
		},
		Granted: granted,
	}, nil
}
