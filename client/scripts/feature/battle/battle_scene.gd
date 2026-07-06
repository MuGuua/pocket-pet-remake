extends Control

## 战斗单位整体等比缩放；使用单个 float 避免宠物宽高比被误调歪。
@export_range(0.1, 4.0, 0.01) var battle_unit_scale: float = 1.0:
	set(value):
		battle_unit_scale = clampf(value, 0.1, 4.0)
		if is_inside_tree():
			_apply_exported_battle_layout_settings()
## 当前出手单位高亮倍率；最终高亮缩放 = battle_unit_scale * battle_planning_highlight_multiplier。
@export_range(1.0, 2.0, 0.01) var battle_planning_highlight_multiplier: float = 1.06:
	set(value):
		battle_planning_highlight_multiplier = clampf(value, 1.0, 2.0)
		if is_inside_tree():
			_apply_exported_battle_layout_settings()
## 战斗单位左右对称分布的中心点；默认对齐 MagicCircle 中心。
@export var battle_formation_center: Vector2 = Vector2(390.0, 580.0):
	set(value):
		battle_formation_center = value
		if is_inside_tree():
			_apply_exported_battle_layout_settings()
## 战斗单位左右两侧到对称中心点的主间距；数值越大双方离中心越远。
@export_range(0.0, 360.0, 1.0) var battle_unit_side_distance: float = 105.0:
	set(value):
		battle_unit_side_distance = clampf(value, 0.0, 360.0)
		if is_inside_tree():
			_apply_exported_battle_layout_settings()
## 同侧出现前后两列时，第二列相对第一列额外拉开的水平间距。
@export_range(0.0, 240.0, 1.0) var battle_back_column_extra_distance: float = 60.0:
	set(value):
		battle_back_column_extra_distance = clampf(value, 0.0, 240.0)
		if is_inside_tree():
			_apply_exported_battle_layout_settings()
## 同侧出现前后两列时，第二列相对第一列的 Y 坐标偏移；正数向下，负数向上。
@export_range(-200.0, 200.0, 1.0) var battle_back_column_y_offset: float = 0.0:
	set(value):
		battle_back_column_y_offset = clampf(value, -200.0, 200.0)
		if is_inside_tree():
			_apply_exported_battle_layout_settings()
## 技能、普攻等战斗命中特效的整体倍率；只影响表现层，不参与服务端战斗计算。
@export_range(0.1, 4.0, 0.01) var battle_effect_scale: float = 1.0:
	set(value):
		battle_effect_scale = clampf(value, 0.1, 4.0)
		if is_inside_tree():
			_apply_exported_battle_layout_settings()
## 同侧存在多个单位上下分布时的 Y 间距；左右两边都会用这个值保持视觉一致。
@export_range(0.0, 260.0, 1.0) var battle_unit_vertical_spacing: float = 126.0:
	set(value):
		battle_unit_vertical_spacing = clampf(value, 0.0, 260.0)
		if is_inside_tree():
			_apply_exported_battle_layout_settings()

@onready var _director: BattleDirector = $BattleDirector
@onready var _network_provider: BattleNetworkProvider = %BattleNetworkProvider

var _battle_controller: Node = null
var _initialized_battle_id: int = 0
var _state_update_running: bool = false
var _state_update_queued: bool = false
var _request_loading: RuntimeProgressOverlay = null

## 进入场景树时提前同步导出配置，确保子节点初始化前能拿到最新战斗布局参数。
func _enter_tree() -> void:
	_apply_exported_battle_layout_settings()


## 初始化战斗请求 loading，并再次同步 Inspector 中的战斗布局参数。
func _ready() -> void:
	_apply_exported_battle_layout_settings()
	_request_loading = RuntimeProgressOverlay.new()
	_request_loading.name = "BattleRequestLoading"
	add_child(_request_loading)


## 把场景导出参数同步到战斗站位与单位缩放的运行时配置。
func _apply_exported_battle_layout_settings() -> void:
	var resolved_unit_scale: Vector2 = Vector2(battle_unit_scale, battle_unit_scale)
	BattleUnit.default_unit_scale = resolved_unit_scale
	BattleUnit.planning_highlight_scale = resolved_unit_scale * battle_planning_highlight_multiplier
	BattleFormationMapper.configure_formation_center(battle_formation_center)
	BattleFormationMapper.configure_side_distance(battle_unit_side_distance)
	BattleFormationMapper.configure_back_column_extra_distance(battle_back_column_extra_distance)
	BattleFormationMapper.configure_back_column_y_offset(battle_back_column_y_offset)
	BattleFormationMapper.configure_vertical_spacing(battle_unit_vertical_spacing)
	BattleEffect.configure_effect_scale_multiplier(battle_effect_scale)
	var director: BattleDirector = get_node_or_null("BattleDirector") as BattleDirector
	if director != null:
		director.apply_current_unit_scale_to_spawned_units()
		director.apply_current_formation_to_spawned_units()


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
	var summary: String = _resolve_battle_finish_summary(payload)
	_director.handle_battle_finished(summary)
	await get_tree().create_timer(0.8).timeout
	if _request_loading != null:
		_request_loading.hide_waiting()
	_initialized_battle_id = 0

## 根据服务端结算原因转换战斗日志文案；逃跑属于主动脱离，不展示为战斗失败。
func _resolve_battle_finish_summary(payload: Dictionary) -> String:
	var reason: String = str(payload.get("reason", ""))
	if reason == "player escaped battle":
		return "逃跑成功"
	if bool(payload.get("win", false)):
		return "战斗胜利"
	return "战斗失败"

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
