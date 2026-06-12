package unlock

import (
	"errors"
	"time"
)

var (
	// ErrInvalidRuntimeUnlockInput 表示正式玩法提交的解锁参数缺少必要字段。
	ErrInvalidRuntimeUnlockInput = errors.New("invalid runtime unlock input")
)

// FeatureRecord 描述玩家已经持久化解锁的一项功能。
// 第一版只要求能被服务端可靠记录，客户端后续可据此决定入口显隐。
type FeatureRecord struct {
	PlayerID   uint64    `json:"player_id"`
	FeatureID  uint64    `json:"feature_id"`
	UnlockedAt time.Time `json:"unlocked_at"`
}

// RuntimeGrantResult 返回一次运行时功能解锁的最终结果。
// 如果该功能之前已经解锁，Granted 会为 false，但仍返回当前权威记录。
type RuntimeGrantResult struct {
	Feature FeatureRecord `json:"feature"`
	Granted bool          `json:"granted"`
}
