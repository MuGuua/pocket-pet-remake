package world

import "context"

// 仅供后端实例交换已确认的世界展示事件，不接受客户端直接写入，也不承载战斗或奖励结算。
type SceneEvent struct {
	Origin   string `json:"origin"`
	Owner    string `json:"owner"`
	PlayerID uint64 `json:"player_id"`
	SceneID  uint32 `json:"scene_id"`
	Command  uint16 `json:"command"`
	Body     []byte `json:"body"`
}

// 独立读取游标让每个实例收到同一世界事件；间隙通过重连全量快照恢复，不能当成可靠玩法指令队列。
type SceneEventBus interface {
	Publish(context.Context, SceneEvent) error
	Consume(context.Context, func(SceneEvent), func()) error
}
