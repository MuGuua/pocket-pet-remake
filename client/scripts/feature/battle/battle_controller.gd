extends Node

# 世界交互回执返回后向外广播接受结果与原因。
signal interact_responded(accepted: bool, reason: String)
signal interact_payload_received(payload: Dictionary)
signal npc_action_payload_received(payload: Dictionary)
# PVP 挑战请求回执返回后向外广播完整载荷。
signal pvp_challenge_responded(payload: Dictionary)
# 收到其他玩家发来的 PVP 邀请后向外广播完整载荷。
signal pvp_challenge_received(payload: Dictionary)
# PVP 邀请接受或拒绝回执返回后向外广播完整载荷。
signal pvp_challenge_reply_responded(payload: Dictionary)
# 战斗动作回执返回后向外广播接受结果与原因。
signal action_responded(accepted: bool, reason: String)
# 战斗开始并写入全局状态后向外广播当前战斗快照。
signal battle_started(state: Dictionary)
# 战斗状态变化后向外广播当前战斗快照。
signal battle_updated(state: Dictionary)
# 战斗结束后向外广播最终战斗快照。
signal battle_finished(state: Dictionary)

# 打印战斗场景消费到的协议载荷，便于对照 App 层网络日志排查字段差异。
func _log_battle_scene(tag: String, payload: Dictionary) -> void:
    var payload_json: String = JSON.stringify(payload, "\t")
    print("[BattleScene][%s] %s" % [tag, payload_json])


# 处理世界交互响应，并转为统一信号事件。
func handle_interact_response(payload: Dictionary) -> void:
    if str(payload.get("response_type", "")) == "battle" or bool(payload.get("accepted", false)):
        _log_battle_scene("INTERACT_RESP", payload)
    interact_responded.emit(bool(payload.get("accepted", false)), str(payload.get("reason", "")))
    interact_payload_received.emit(payload)

func handle_npc_action_response(payload: Dictionary) -> void:
    if str(payload.get("result_type", "")) == "battle":
        _log_battle_scene("NPC_ACTION_RESP", payload)
    npc_action_payload_received.emit(payload)

# 处理单人 PVP 挑战请求回执。
func handle_pvp_challenge_response(payload: Dictionary) -> void:
    _log_battle_scene("PVP_CHALLENGE_RESP", payload)
    pvp_challenge_responded.emit(payload)

# 处理服务端下发的单人 PVP 邀请推送。
func handle_pvp_challenge_push(payload: Dictionary) -> void:
    _log_battle_scene("PVP_CHALLENGE_PUSH", payload)
    pvp_challenge_received.emit(payload)

# 处理单人 PVP 邀请应答回执。
func handle_pvp_challenge_reply_response(payload: Dictionary) -> void:
    _log_battle_scene("PVP_CHALLENGE_REPLY_RESP", payload)
    pvp_challenge_reply_responded.emit(payload)

# 处理战斗动作响应，并转为统一信号事件。
func handle_battle_action_response(payload: Dictionary) -> void:
    _log_battle_scene("BATTLE_ACTION_RESP", payload)
    action_responded.emit(bool(payload.get("accepted", false)), str(payload.get("reason", "")))

# 处理战斗开始推送，并初始化全局战斗状态。
func handle_battle_start(payload: Dictionary) -> void:
    _log_battle_scene("BATTLE_START_PUSH", payload)
    GameState.clear_battle_state()
    GameState.set_battle_state(payload, true)
    battle_started.emit(GameState.battle_state)
    battle_updated.emit(GameState.battle_state)

# 处理战斗中状态推送，并更新全局战斗快照。
func handle_battle_state(payload: Dictionary) -> void:
    _log_battle_scene("BATTLE_STATE_PUSH", payload)
    GameState.set_battle_state(payload, true)
    battle_updated.emit(GameState.battle_state)

# 处理战斗结果推送，并把全局状态切换为非战斗态。
func handle_battle_result(payload: Dictionary) -> void:
    _log_battle_scene("BATTLE_RESULT_PUSH", payload)
    ## 结算包单独保留一份副本，避免后续 battle_state 合并时被战斗过程字段干扰弹窗展示。
    var result_snapshot: Dictionary = payload.duplicate(true)
    GameState.apply_battle_player_rewards(payload)
    GameState.set_battle_state(payload, false)
    battle_updated.emit(GameState.battle_state)
    battle_finished.emit(result_snapshot)
