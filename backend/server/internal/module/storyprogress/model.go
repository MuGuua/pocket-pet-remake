package storyprogress

import "context"

// Repository 定义玩家剧情进度、场景触发器和个人解锁标记的持久化能力。
// 这些数据决定某个玩家是否应该看到一次性剧情、NPC 或后续主线任务入口。
type Repository interface {
	FindPendingSceneTrigger(ctx context.Context, playerID uint64, sceneID uint32) (*SceneTrigger, error)
	CompleteSceneTrigger(ctx context.Context, playerID uint64, triggerCode string) (*SceneTrigger, error)
}

// SceneTrigger 是服务端权威下发给客户端播放的场景剧情触发器。
// 客户端只负责播放 client_animation_key 对应的本地场景，播放完毕后必须 Ack 给服务端。
type SceneTrigger struct {
	TriggerCode         string
	SceneID             uint32
	ClientAnimationKey  string
	PromptText          string
	BlockMovement       bool
	EffectAcceptQuestID uint64
	EffectSetFlags      []string
}
