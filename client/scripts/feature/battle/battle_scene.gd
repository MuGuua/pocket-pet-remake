extends Control

@onready var _director: BattleDirector = $BattleDirector
@onready var _network_provider: BattleNetworkProvider = %BattleNetworkProvider

var _battle_controller: Node = null
var _initialized_battle_id: int = 0
var _state_update_running: bool = false
var _state_update_queued: bool = false
var _request_loading: BattleRequestLoading = null

func _ready() -> void:
	_request_loading = BattleRequestLoading.new()
	_request_loading.name = "BattleRequestLoading"
	_request_loading.z_index = 120
	add_child(_request_loading)

## 绑定主场景里的战斗控制器并订阅信号。
func bind_battle_controller(battle_controller: Node) -> void:
	if _battle_controller != null:
		_disconnect_battle_controller()
	_battle_controller = battle_controller
	if _battle_controller == null:
		return
	if _battle_controller.has_signal("battle_started"):
		_battle_controller.connect("battle_started", Callable(self, "_on_battle_started"))
	if _battle_controller.has_signal("battle_updated"):
		_battle_controller.connect("battle_updated", Callable(self, "_on_battle_updated"))
	if _battle_controller.has_signal("battle_finished"):
		_battle_controller.connect("battle_finished", Callable(self, "_on_battle_finished"))
	if _battle_controller.has_signal("action_responded"):
		_battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
	if _director != null and _director.has_signal("action_requested"):
		_director.connect("action_requested", Callable(self, "_on_action_requested"))

func _exit_tree() -> void:
	_disconnect_battle_controller()
	if _director != null and _director.has_signal("action_requested") and _director.action_requested.is_connected(_on_action_requested):
		_director.action_requested.disconnect(_on_action_requested)

func _disconnect_battle_controller() -> void:
	if _battle_controller == null:
		return
	if _battle_controller.has_signal("battle_started") and _battle_controller.battle_started.is_connected(_on_battle_started):
		_battle_controller.battle_started.disconnect(_on_battle_started)
	if _battle_controller.has_signal("battle_updated") and _battle_controller.battle_updated.is_connected(_on_battle_updated):
		_battle_controller.battle_updated.disconnect(_on_battle_updated)
	if _battle_controller.has_signal("battle_finished") and _battle_controller.battle_finished.is_connected(_on_battle_finished):
		_battle_controller.battle_finished.disconnect(_on_battle_finished)
	if _battle_controller.has_signal("action_responded") and _battle_controller.action_responded.is_connected(_on_action_responded):
		_battle_controller.action_responded.disconnect(_on_action_responded)
	_battle_controller = null

func _on_battle_started(payload: Dictionary) -> void:
	var battle_id: int = int(payload.get("battle_id", 0))
	if battle_id <= 0:
		return
	if _initialized_battle_id != battle_id:
		_initialized_battle_id = battle_id
		_director.initialize_battle()
	else:
		_director.handle_battle_state_update()

func _on_battle_updated(payload: Dictionary) -> void:
	if payload.is_empty():
		return
	if _request_loading != null:
		_request_loading.hide_waiting()
	_director.ingest_state_push(payload)
	call_deferred("_run_state_update")

func _on_battle_finished(_payload: Dictionary) -> void:
	# 结算演出与卸载由 main 调用 wait_for_presentation_complete 统一收尾。
	pass

## 播放剩余事件并短暂停留结算文案，供主场景在卸载前等待。
func wait_for_presentation_complete(payload: Dictionary) -> void:
	await _wait_for_presentation_idle()
	var summary: String = "战斗胜利" if bool(payload.get("win", false)) else "战斗失败"
	_director.handle_battle_finished(summary)
	await get_tree().create_timer(0.8).timeout
	if _request_loading != null:
		_request_loading.hide_waiting()
	_initialized_battle_id = 0

## 等待所有排队的状态更新与事件演出结束，避免结算包先到导致提前卸载战斗场景。
func _wait_for_presentation_idle() -> void:
	while true:
		while (
			_state_update_running
			or _state_update_queued
			or _director.has_pending_presentations()
		):
			if not _state_update_running:
				await _run_state_update()
			await get_tree().process_frame
		await _director.wait_for_post_presentation_settle()
		if (
			not _state_update_running
			and not _state_update_queued
			and not _director.has_pending_presentations()
		):
			break
		await get_tree().process_frame

func _on_action_responded(accepted: bool, reason: String) -> void:
	if not accepted and _request_loading != null:
		_request_loading.hide_waiting()
	if accepted:
		_director.mark_action_accepted()
	else:
		_director.handle_action_rejected(reason)

func _on_action_requested(actor_id: int, action_type: int, skill_id: int, target_id: int) -> void:
	if _request_loading != null:
		_request_loading.show_waiting()
	App.submit_battle_action(
		_network_provider.get_battle_id(),
		_network_provider.get_round(),
		actor_id,
		target_id,
		action_type,
		skill_id
	)

func _run_state_update() -> void:
	if _state_update_running:
		_state_update_queued = true
		return
	await _run_state_update_loop()

## 串行消费 battle_updated 触发的状态更新，直到当前队列排空。
func _run_state_update_loop() -> void:
	_state_update_running = true
	while true:
		_state_update_queued = false
		await _director.handle_battle_state_update()
		if not _state_update_queued:
			break
	_state_update_running = false
