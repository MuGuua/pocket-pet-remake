extends Node

## 处理玩家成长相关协议，并把服务端权威快照写回 GameState。

# 处理属性点分配响应，并刷新本地玩家快照。
func handle_allocate_attr_response(payload: Dictionary) -> void:
	var player_variant: Variant = payload.get("player", {})
	if player_variant is Dictionary:
		GameState.merge_player_snapshot(player_variant)
