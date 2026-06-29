extends Node

# 任务结算弹窗请求：由主场景按“先升级、后奖励”顺序展示。
signal quest_settlement_popup_requested(payload: Dictionary)

# 任务列表或追踪状态刷新后向外广播当前任务总数。
signal quests_updated(count: int)

# 处理任务列表响应，并把快照整体写入全局状态。
func handle_quest_list(payload: Dictionary) -> void:
	var quests_variant: Variant = payload.get("quests", [])
	var quests: Array = quests_variant if quests_variant is Array else []
	GameState.set_quests(quests, int(payload.get("tracked_quest_id", 0)))
	quests_updated.emit(GameState.quests.size())

# 处理任务增量更新推送，并把单条任务合并进全局状态。
func handle_quest_update(payload: Dictionary) -> void:
	var quest_variant: Variant = payload.get("quest", payload)
	var quest: Dictionary = quest_variant if quest_variant is Dictionary else {}
	GameState.upsert_quest(quest)
	quests_updated.emit(GameState.quests.size())

# 处理任务移除推送。
func handle_quest_remove(payload: Dictionary) -> void:
	GameState.remove_quest(int(payload.get("quest_id", 0)))
	quests_updated.emit(GameState.quests.size())

# 处理任务接取响应；成功时优先合并返回的任务快照。
func handle_quest_accept_response(payload: Dictionary) -> void:
	if not bool(payload.get("accepted", false)):
		return
	if payload.has("quest"):
		handle_quest_update(payload)
	elif payload.has("quests"):
		handle_quest_list(payload)
	App.request_quest_list()

# 处理任务提交响应；成功时优先合并返回的任务快照。
func handle_quest_submit_response(payload: Dictionary) -> void:
	if not bool(payload.get("accepted", false)):
		return
	if payload.has("quest"):
		handle_quest_update(payload)
	elif payload.has("quests"):
		handle_quest_list(payload)
	var rewards_variant: Variant = payload.get("rewards", [])
	var rewards: Array = rewards_variant if rewards_variant is Array else []
	var level_up_count: int = int(payload.get("level_up_count", 0))
	var attr_points_gained: int = int(payload.get("attr_points_gained", 0))
	if level_up_count > 0 or attr_points_gained > 0 or not rewards.is_empty():
		quest_settlement_popup_requested.emit(payload)
	App.request_quest_list()

# 处理任务追踪响应；成功时切换本地追踪状态。
func handle_quest_track_response(payload: Dictionary) -> void:
	if not bool(payload.get("accepted", false)):
		return
	GameState.set_tracked_quest(int(payload.get("quest_id", 0)))
	quests_updated.emit(GameState.quests.size())
	App.request_quest_list()
