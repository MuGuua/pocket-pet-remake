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
}
