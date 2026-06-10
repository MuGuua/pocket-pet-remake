extends Node

# 世界交互回执返回后向外广播接受结果与原因。
signal interact_responded(accepted: bool, reason: String)
signal interact_payload_received(payload: Dictionary)
signal npc_action_payload_received(payload: Dictionary)
# 战斗动作回执返回后向外广播接受结果与原因。
signal action_responded(accepted: bool, reason: String)
# 战斗开始并写入全局状态后向外广播当前战斗快照。
signal battle_started(state: Dictionary)
# 战斗状态变化后向外广播当前战斗快照。
signal battle_updated(state: Dictionary)
# 战斗结束后向外广播最终战斗快照。
signal battle_finished(state: Dictionary)

# 处理世界交互响应，并转为统一信号事件。
func handle_interact_response(payload: Dictionary) -> void:
    interact_responded.emit(bool(payload.get("accepted", false)), str(payload.get("reason", "")))
    interact_payload_received.emit(payload)

func handle_npc_action_response(payload: Dictionary) -> void:
    npc_action_payload_received.emit(payload)

# 处理战斗动作响应，并转为统一信号事件。
func handle_battle_action_response(payload: Dictionary) -> void:
    action_responded.emit(bool(payload.get("accepted", false)), str(payload.get("reason", "")))

# 处理战斗开始推送，并初始化全局战斗状态。
func handle_battle_start(payload: Dictionary) -> void:
    GameState.clear_battle_state()
    GameState.set_battle_state(payload, true)
    battle_started.emit(GameState.battle_state)
    battle_updated.emit(GameState.battle_state)

# 处理战斗中状态推送，并更新全局战斗快照。
func handle_battle_state(payload: Dictionary) -> void:
    GameState.set_battle_state(payload, true)
    battle_updated.emit(GameState.battle_state)

# 处理战斗结果推送，并把全局状态切换为非战斗态。
func handle_battle_result(payload: Dictionary) -> void:
    GameState.apply_battle_player_rewards(payload)
    GameState.set_battle_state(payload, false)
    battle_updated.emit(GameState.battle_state)
    battle_finished.emit(GameState.battle_state)
