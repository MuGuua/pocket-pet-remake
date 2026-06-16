package battle

import "context"

// RewardRecord stores one already-granted battle reward summary so the server
// can skip duplicate reward grants for the same player and battle id.
type RewardRecord struct {
	BattleID    uint64
	PlayerID    uint64
	BattleType  uint32
	Result      int16
	RewardGold  uint32
	RewardExp   uint64
	PayloadJSON []byte
}

// Repository is intentionally tiny for the current MVP: it only provides
// reward grant dedupe through battle_record persistence.
type Repository interface {
	CreateRewardRecord(ctx context.Context, record RewardRecord) (bool, error)
	// DeleteRewardRecord 在发奖链路失败时回滚占位记录，允许客户端重试同一场战斗结算。
	DeleteRewardRecord(ctx context.Context, battleID uint64, playerID uint64) error
	// MaxRewardBattleID 返回 battle_record 中已持久化的最大 battle_id，供运行时分配新战斗 ID。
	MaxRewardBattleID(ctx context.Context) (uint64, error)
}
