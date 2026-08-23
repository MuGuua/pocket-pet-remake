extends Node

# 世界交互回执返回后向外广播接受结果与原因。
signal interact_responded(accepted: bool, reason: String)
signal interact_payload_received(payload: Dictionary)
signal npc_action_payload_received(payload: Dictionary)
signal npc_menu_payload_received(payload: Dictionary)
## 当前地图全部 NPC 菜单批量回包到达后广播完整载荷。
signal npc_menu_batch_payload_received(payload: Dictionary)
signal npc_dialogue_payload_received(payload: Dictionary)
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

## 当前战斗是否已经向主运行态发出过结束通知，用于避免 4012 finished 与 4013 结算包重复卸载场景。
var _battle_finish_emitted: bool = false
## 当前已发出的结束通知是否只是 4012 finished 兜底结果；正式 4013 到达时仍允许再同步奖励。
var _battle_finish_was_fallback: bool = false
## 4012 兜底调度序号；新战斗开始或正式 4013 到达后递增，用于取消过期的兜底 emit。
var _battle_fallback_emit_generation: int = 0


## 延迟一帧再尝试发出 4012 兜底结束，优先让同帧或下一帧到达的 4013 结算包接管奖励弹窗。
func _schedule_battle_finished_fallback(state_snapshot: Dictionary) -> void:
	var generation: int = _battle_fallback_emit_generation
	call_deferred("_try_emit_battle_finished_fallback", state_snapshot, generation)


func _try_emit_battle_finished_fallback(state_snapshot: Dictionary, generation: int) -> void:
	if generation != _battle_fallback_emit_generation:
		return
	if _battle_finish_emitted and not _battle_finish_was_fallback:
		return
	_emit_battle_finished_from_state(state_snapshot)


## 客户端不再格式化或输出战斗协议日志，避免高频状态包产生额外序列化开销。
func _log_battle_scene(_tag: String, _payload: Dictionary) -> void:
	pass


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

## 处理 NPC 菜单拉取响应，供主运行态打开统一菜单面板。
func handle_npc_menu_response(payload: Dictionary) -> void:
	npc_menu_payload_received.emit(payload)

## 转发当前地图 NPC 批量菜单响应，主运行态会按场景 ID 校验后异步写入缓存。
func handle_npc_menu_batch_response(payload: Dictionary) -> void:
	npc_menu_batch_payload_received.emit(payload)

## 处理 NPC 结构化剧情节点回包，供主运行态驱动客户端面板或本地剧情动画。
func handle_npc_dialogue_response(payload: Dictionary) -> void:
	npc_dialogue_payload_received.emit(payload)

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
	_battle_finish_emitted = false
	_battle_finish_was_fallback = false
	_battle_fallback_emit_generation += 1
	GameState.clear_battle_state()
	GameState.set_battle_state(payload, true)
	## 开战时补拉一次已佩戴装备，确保人物普攻能展示武器名。
	if GameState.equipped_items.is_empty():
		App.request_player_equipment_list()
	battle_started.emit(GameState.battle_state)
	battle_updated.emit(GameState.battle_state)

# 处理战斗中状态推送，并更新全局战斗快照。
func handle_battle_state(payload: Dictionary) -> void:
	_log_battle_scene("BATTLE_STATE_PUSH", payload)
	GameState.set_battle_state(payload, true)
	battle_updated.emit(GameState.battle_state)
	if str(payload.get("phase", "")) == "finished":
		call_deferred("_schedule_battle_finished_fallback", payload.duplicate(true))

# 处理战斗结果推送，并把全局状态切换为非战斗态。
func handle_battle_result(payload: Dictionary) -> void:
	_log_battle_scene("BATTLE_RESULT_PUSH", payload)
	_battle_fallback_emit_generation += 1
	## 结算包单独保留一份副本，避免后续 battle_state 合并时被战斗过程字段干扰弹窗展示。
	var result_snapshot: Dictionary = payload.duplicate(true)
	GameState.apply_battle_player_rewards(payload)
	GameState.apply_battle_pet_rewards(payload)
	GameState.set_battle_state(payload, false)
	battle_updated.emit(GameState.battle_state)
	## 延迟一帧再通知主场景结束，确保 battle_updated 已 ingest 并排入演出队列。
	call_deferred("_emit_battle_finished", result_snapshot)

## 当服务端只下发 4012 finished、4013 结算包因奖励落库异常等原因未到达时，兜底退出战斗界面。
func _emit_battle_finished_from_state(state_snapshot: Dictionary) -> void:
	if _battle_finish_emitted:
		return
	var merged_state: Dictionary = GameState.battle_state.duplicate(true)
	if merged_state.is_empty():
		merged_state = state_snapshot.duplicate(true)
	# 4012 finished 已经是服务端权威战斗终态；即使 4013 结算包稍后才到，
	# 客户端也必须先退出本地战斗态，避免任务、世界移动和 HUD 继续被“战斗中”锁住。
	GameState.set_battle_state(merged_state, false)
	## 4012 不携带奖励结算字段，因此这里只构造最小结束载荷；奖励、经验和掉落仍以 4013 为准。
	var fallback_result: Dictionary = {
		"battle_id": int(merged_state.get("battle_id", 0)),
		"win": _infer_player_win_from_state(merged_state),
		"reason": "battle state finished",
		"fallback_result": true,
	}
	_battle_finish_was_fallback = true
	_emit_battle_finished(fallback_result)

## 在 battle_updated 处理完成后再发出 battle_finished，避免抢在演出前卸载战斗场景。
func _emit_battle_finished(result_snapshot: Dictionary) -> void:
	if _battle_finish_emitted:
		if not _battle_finish_was_fallback or bool(result_snapshot.get("fallback_result", false)):
			return
		_battle_finish_was_fallback = false
	else:
		_battle_finish_was_fallback = bool(result_snapshot.get("fallback_result", false))
	_battle_finish_emitted = true
	battle_finished.emit(result_snapshot)

## 根据 4012 finished 中的权威单位血量推断胜负；仅用于 UI 兜底退出，不用于奖励结算。
func _infer_player_win_from_state(state_snapshot: Dictionary) -> bool:
	var enemy_actor_ids: Dictionary = {}
	var enemies_variant: Variant = state_snapshot.get("enemies", [])
	if enemies_variant is Array:
		for enemy_variant: Variant in enemies_variant:
			if enemy_variant is not Dictionary:
				continue
			var enemy: Dictionary = enemy_variant as Dictionary
			var enemy_actor_id: int = int(enemy.get("actor_id", 0))
			if enemy_actor_id > 0:
				enemy_actor_ids[enemy_actor_id] = true
	if enemy_actor_ids.is_empty():
		return false
	var actors_variant: Variant = state_snapshot.get("actors", [])
	if actors_variant is not Array:
		return false
	var has_living_enemy: bool = false
	for actor_variant: Variant in actors_variant:
		if actor_variant is not Dictionary:
			continue
		var actor: Dictionary = actor_variant as Dictionary
		var actor_id: int = int(actor.get("actor_id", 0))
		if not enemy_actor_ids.has(actor_id):
			continue
		if int(actor.get("hp", 0)) > 0:
			has_living_enemy = true
			break
	return not has_living_enemy
