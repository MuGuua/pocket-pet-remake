extends Node
class_name BattleDirector

const BattleUnitScene: PackedScene = preload("res://scenes/battle/battle_unit.tscn")
const BattleEffectScene: PackedScene = preload("res://scenes/battle/battle_effect.tscn")
const FloatingTextScript: GDScript = preload("res://scripts/feature/battle/floating_text.gd")
const BattleDigitFloatScript: GDScript = preload("res://scripts/feature/battle/battle_digit_float.gd")

const STATE_INIT: String = "init"
const STATE_WAITING_INPUT: String = "waiting_input"
const STATE_SELECTING_TARGET: String = "selecting_target"
const STATE_SUBMITTING: String = "submitting_action"
const STATE_PLAYING: String = "playing_round"
const STATE_BATTLE_END: String = "battle_end"

var _state: String = STATE_INIT
var _current_round: int = 1
var _units: Dictionary = {}
var _slot_positions: Dictionary = {}
var _actions_by_id: Dictionary = {}
var _combos_by_id: Dictionary = {}
var _parallel_remaining: int = 0
var _selection_order: Array[int] = []
var _selection_index: int = 0
var _ally_selections: Dictionary = {}
var _planning_unit: BattleUnit = null
var _pending_selection: Dictionary = {}
var _pending_skill_choices: Array[Dictionary] = []
var _pending_item_choices: Array[Dictionary] = []
var _target_arrow_unit_id: int = 0

@onready var _network: BattleNetworkProvider = %BattleNetworkProvider
@onready var _content_registry: BattleContentRegistry = %BattleContentRegistry
@onready var _unit_layer: Node2D = %UnitLayer
@onready var _effect_layer: Node2D = %EffectLayer
@onready var _floating_layer: Node2D = %FloatingTextLayer
@onready var _battlefield_root: Control = %BattlefieldRoot
@onready var _action_panel: ActionPanel = %ActionPanel
@onready var _action_log_panel: ActionLogPanel = %ActionLogPanel
@onready var _command_status: PanelContainer = %CommandStatusBar

signal parallel_group_finished
signal action_requested(actor_id: int, action_type: int, skill_id: int, target_id: int)
signal interaction_locked_changed(locked: bool, tip: String)

var _initialized: bool = false
var _is_playing_events: bool = false
var _last_played_frame: int = 0
var _interaction_locked: bool = false

func _ready() -> void:
	_slot_positions = BattleFormationMapper.build_slot_positions()
	_content_registry.load_content()
	_action_panel.action_selected.connect(_on_action_selected)
	_action_panel.list_choice_selected.connect(_on_list_choice_selected)
	_action_panel.list_choice_cancelled.connect(_on_list_choice_cancelled)
	_action_panel.target_selection_cancelled.connect(_on_target_selection_cancelled)
	_action_panel.target_selection_confirmed.connect(_on_target_selection_confirmed)
	if _action_panel.has_node("%ItemButton"):
		_action_panel.get_node("%ItemButton").visible = false
	if _command_status != null and _command_status.has_signal("request_auto_timeout"):
		_command_status.request_auto_timeout.connect(_on_command_timeout_auto_request)

## 战斗场景挂载后，根据 4011 快照初始化单位与指令阶段。
func initialize_battle() -> void:
	if _initialized:
		return
	_initialized = true
	_last_played_frame = _network.get_frame()
	_current_round = _network.get_round()
	_bootstrap_units()
	_action_log_panel.clear_logs()
	_action_log_panel.append_log("战斗开始，等待指令。")
	_refresh_command_phase()

## 收到 4012/4013 后由 battle_scene 转发。
func handle_battle_state_update() -> void:
	_current_round = _network.get_round()
	var frame: int = _network.get_frame()
	var events: Array[Dictionary] = _network.get_events()
	if not events.is_empty() and not _is_playing_events and frame > _last_played_frame:
		print("[BattleScene][PLAY_EVENTS] frame=%d events=%d phase=%s" % [frame, events.size(), _network.get_phase()])
		_lock_interaction("正在播放战斗演出")
		await _play_state_events()
		_last_played_frame = frame
	_sync_runtime_actors()
	if _network.get_phase() == "command":
		var pending_actor_ids: Array[int] = _network.get_pending_actor_ids()
		if not pending_actor_ids.is_empty():
			_unlock_interaction()
			_refresh_command_phase()
		else:
			_lock_interaction("正在等待回合结算")
	elif _network.get_phase() == "finished":
		_lock_interaction("战斗结算中")

## 服务端已受理动作，但尚未收到带演出事件的 4012，此时继续保持锁定。
func mark_action_accepted() -> void:
	if _state == STATE_SUBMITTING:
		_state = STATE_WAITING_INPUT
	_lock_interaction("正在等待服务端结算")

## 动作被拒绝时回滚当前选择并恢复可操作状态。
func handle_action_rejected(reason: String) -> void:
	_selection_index = maxi(0, _selection_index - 1)
	_state = STATE_WAITING_INPUT
	_unlock_interaction()
	_refresh_command_phase()
	if not reason.is_empty():
		_action_log_panel.append_log(reason)

## 是否正在播放事件时间轴。
func is_playing_events() -> bool:
	return _is_playing_events

## 战斗结束时锁定面板。
func handle_battle_finished(summary: String = "战斗结束") -> void:
	_lock_interaction("战斗结束")
	_finish_battle(summary)

## 锁定战斗交互并禁用操作按钮；自动战斗开启时仍保留自动按钮可点。
func _lock_interaction(tip: String) -> void:
	if _interaction_locked and tip.is_empty():
		return
	_interaction_locked = true
	if _network.is_auto_battle_enabled():
		_action_panel.set_auto_mode_active(true)
	else:
		_action_panel.set_buttons_disabled(true)
	interaction_locked_changed.emit(true, tip)

## 解除战斗交互锁定，允许继续选招或点选目标。
func _unlock_interaction() -> void:
	if not _interaction_locked:
		return
	_interaction_locked = false
	interaction_locked_changed.emit(false, "")

func is_interaction_locked() -> bool:
	return _interaction_locked

func _refresh_command_phase() -> void:
	if _interaction_locked:
		return
	if _network.is_auto_battle_enabled():
		_action_panel.set_auto_mode_active(true)
		return
	_selection_order = _network.get_pending_actor_ids()
	_selection_index = 0
	if _selection_order.is_empty():
		_action_panel.set_buttons_disabled(true)
		return
	_state = STATE_WAITING_INPUT
	_action_panel.begin_selection_phase()
	_begin_next_ally_selection()

func _input(event: InputEvent) -> void:
	if _interaction_locked:
		return
	if _state != STATE_SELECTING_TARGET:
		return
	if event is InputEventMouseButton:
		var mouse_event: InputEventMouseButton = event as InputEventMouseButton
		if mouse_event.pressed and mouse_event.button_index == MOUSE_BUTTON_LEFT:
			var picked_unit: BattleUnit = _pick_target_unit_at_mouse()
			if picked_unit != null:
				_handle_target_unit_pressed(picked_unit)
				get_viewport().set_input_as_handled()

func _bootstrap_units() -> void:
	var initial_units: Array[Dictionary] = _network.get_initial_units()
	for unit_data: Dictionary in initial_units:
		var actor_id: int = int(unit_data.get("actor_id", 0))
		if actor_id <= 0 or _units.has(actor_id):
			continue
		_spawn_unit(unit_data)

func _spawn_unit(unit_data: Dictionary) -> BattleUnit:
	var unit: BattleUnit = BattleUnitScene.instantiate() as BattleUnit
	var position_key: String = str(unit_data.get("position", "left_front"))
	var slot_position: Vector2 = _get_slot_position(position_key)
	var resolved_skin: UnitSkin = _content_registry.get_unit_skin(str(unit_data.get("skin_id", "")))
	var tween: Tween = create_tween()
	_unit_layer.add_child(unit)
	unit.setup(unit_data, slot_position, resolved_skin)
	unit.position = slot_position + Vector2(0.0, 10.0)
	unit.modulate.a = 0.0
	tween.set_parallel(true)
	tween.tween_property(unit, "position", slot_position, 0.28).set_trans(Tween.TRANS_BACK).set_ease(Tween.EASE_OUT)
	tween.tween_property(unit, "modulate:a", 1.0, 0.2)
	_units[unit.actor_id] = unit
	return unit

func _is_ally_unit(unit: BattleUnit) -> bool:
	return unit.unit_type == "player" or unit.unit_type == "pet"

func _begin_next_ally_selection() -> void:
	_clear_planning_highlight()
	if _selection_index >= _selection_order.size():
		_action_panel.set_buttons_disabled(true)
		return
	var actor_id: int = _selection_order[_selection_index]
	var unit: BattleUnit = _get_unit(actor_id)
	if unit == null or unit.is_dead:
		_selection_index += 1
		_begin_next_ally_selection()
		return
	_planning_unit = unit
	unit.set_planning_highlight(true)
	_action_panel.set_buttons_disabled(false)

func _on_action_selected(action_type: String) -> void:
	if action_type == "auto":
		if _network.is_auto_battle_enabled():
			_submit_auto_toggle(false)
		elif _state == STATE_WAITING_INPUT:
			_submit_auto_toggle(true)
		return
	if _state == STATE_SELECTING_TARGET and action_type in ["escape", "cancel_target"]:
		_cancel_target_selection()
		return
	if _state != STATE_WAITING_INPUT:
		return
	if _selection_index >= _selection_order.size():
		return
	var actor_id: int = _selection_order[_selection_index]
	var unit: BattleUnit = _get_unit(actor_id)
	if unit == null:
		return
	match action_type:
		"attack":
			_resolve_action_selection(_build_attack_selection())
		"skill":
			_begin_skill_selection(unit)
		"item":
			pass
		"escape":
			_resolve_action_selection(_build_escape_selection())
		_:
			_resolve_action_selection(_build_simple_selection(action_type))

func _on_list_choice_selected(list_type: String, choice_index: int) -> void:
	if _state != STATE_WAITING_INPUT:
		return
	match list_type:
		"skill":
			if choice_index < 0 or choice_index >= _pending_skill_choices.size():
				_pending_skill_choices.clear()
				return
			var skill_data: Dictionary = _pending_skill_choices[choice_index]
			_pending_skill_choices.clear()
			_resolve_action_selection(_build_skill_selection(skill_data))
		"item":
			if choice_index < 0 or choice_index >= _pending_item_choices.size():
				_pending_item_choices.clear()
				return
			var item_data: Dictionary = _pending_item_choices[choice_index]
			_pending_item_choices.clear()
			_resolve_action_selection(_build_item_selection(item_data))
		_:
			pass

func _on_list_choice_cancelled(list_type: String) -> void:
	if _state != STATE_WAITING_INPUT:
		return
	if list_type == "skill":
		_pending_skill_choices.clear()
	elif list_type == "item":
		_pending_item_choices.clear()

func _on_target_selection_cancelled() -> void:
	if _state != STATE_SELECTING_TARGET:
		return
	_cancel_target_selection()

func _on_target_selection_confirmed() -> void:
	if _state != STATE_SELECTING_TARGET:
		return
	_confirm_target_selection()

func _get_round_available_skills(unit: BattleUnit) -> Array[Dictionary]:
	return unit.skills.duplicate()

func _handle_target_unit_pressed(unit: BattleUnit) -> void:
	var target_side: String = str(_pending_selection.get("target_side", "enemy"))
	if not _is_target_candidate(unit, target_side):
		return
	if _target_arrow_unit_id == unit.actor_id:
		_confirm_target_selection()
		return
	_target_arrow_unit_id = unit.actor_id
	_update_target_selection_visuals()

func _confirm_target_selection() -> void:
	if _target_arrow_unit_id <= 0:
		return
	var target: BattleUnit = _get_unit(_target_arrow_unit_id)
	if target == null or not _is_target_candidate(target, str(_pending_selection.get("target_side", "enemy"))):
		return
	_pending_selection["target_ids"] = [_target_arrow_unit_id]
	_clear_target_selection_visuals()
	_action_panel.set_target_selection_mode(false)
	_state = STATE_WAITING_INPUT
	_commit_unit_selection(_pending_selection)
	_pending_selection.clear()

func _begin_skill_selection(unit: BattleUnit) -> void:
	var available_skills: Array[Dictionary] = _get_round_available_skills(unit)
	if available_skills.is_empty():
		_action_log_panel.append_log("%s 没有可用技能" % unit.unit_name)
		return
	if available_skills.size() == 1:
		_resolve_action_selection(_build_skill_selection(available_skills[0]))
		return
	_pending_skill_choices = available_skills.duplicate()
	_action_panel.open_choice_list("skill", available_skills)

func _begin_item_selection(unit: BattleUnit) -> void:
	if unit.items.is_empty():
		_action_log_panel.append_log("%s 没有可用物品" % unit.unit_name)
		return
	if unit.items.size() == 1:
		_resolve_action_selection(_build_item_selection(unit.items[0]))
		return
	_pending_item_choices = unit.items.duplicate()
	_action_panel.open_choice_list("item", unit.items)

func _build_attack_selection() -> Dictionary:
	return {
		"action_type": "attack",
		"skill_id": App.DEFAULT_BATTLE_SKILL_ID,
		"display_name": "普通攻击",
		"skill_visual_id": "",
		"target_count": 1,
		"target_side": "enemy",
		"target_ids": [],
	}

func _build_escape_selection() -> Dictionary:
	return {
		"action_type": "escape",
		"skill_id": 0,
		"display_name": "逃跑",
		"skill_visual_id": "",
		"target_count": 0,
		"target_side": "none",
		"target_ids": [],
	}

func _build_simple_selection(action_type: String) -> Dictionary:
	return {
		"action_type": action_type,
		"skill_id": "",
		"display_name": _display_action_name(action_type),
		"skill_visual_id": "",
		"target_count": 0,
		"target_side": "none",
		"target_ids": [],
	}

func _build_item_selection(item_data: Dictionary) -> Dictionary:
	return {
		"action_type": "item",
		"skill_id": "",
		"item_id": str(item_data.get("item_id", "")),
		"display_name": str(item_data.get("display_name", item_data.get("item_id", "物品"))),
		"skill_visual_id": str(item_data.get("skill_visual_id", "")),
		"target_count": int(item_data.get("target_count", 1)),
		"target_side": str(item_data.get("target_side", "ally")),
		"target_ids": [],
	}

func _build_skill_selection(skill_data: Dictionary) -> Dictionary:
	return {
		"action_type": "skill",
		"skill_id": int(skill_data.get("skill_id", 0)),
		"display_name": str(skill_data.get("display_name", skill_data.get("skill_id", "技能"))),
		"skill_visual_id": str(skill_data.get("skill_visual_id", "")),
		"target_count": int(skill_data.get("target_count", 1)),
		"target_side": str(skill_data.get("target_side", "enemy")),
		"target_ids": [],
	}

func _resolve_action_selection(selection: Dictionary) -> void:
	if _selection_index >= _selection_order.size():
		return
	var actor_id: int = _selection_order[_selection_index]
	selection["actor_id"] = actor_id
	var mode: String = _get_target_selection_mode(selection)
	match mode:
		"manual":
			_pending_selection = selection.duplicate(true)
			_begin_target_selection()
		"random":
			selection["target_ids"] = _pick_random_target_ids(selection)
			_commit_unit_selection(selection)
		"all":
			selection["target_ids"] = _pick_all_target_ids(selection)
			_commit_unit_selection(selection)
		_:
			_commit_unit_selection(selection)

func _get_target_selection_mode(selection: Dictionary) -> String:
	var action_type: String = str(selection.get("action_type", ""))
	if action_type in ["auto", "escape"]:
		return "none"
	var target_count: int = int(selection.get("target_count", 1))
	if target_count <= 0:
		return "all"
	if target_count == 1:
		return "manual"
	return "random"

func _begin_target_selection() -> void:
	_state = STATE_SELECTING_TARGET
	_target_arrow_unit_id = 0
	var default_target: BattleUnit = _pick_default_target_unit(_pending_selection)
	if default_target != null:
		_target_arrow_unit_id = default_target.actor_id
	_action_panel.set_target_selection_mode(true)
	_update_target_selection_visuals()

func _pick_default_target_unit(selection: Dictionary) -> BattleUnit:
	var target_side: String = str(selection.get("target_side", "enemy"))
	var candidates: Array[BattleUnit] = _get_target_candidates(target_side)
	if candidates.is_empty():
		return null
	_sort_target_candidates(candidates, target_side)
	return candidates[0]

func _sort_target_candidates(candidates: Array[BattleUnit], target_side: String) -> void:
	match target_side:
		"ally":
			candidates.sort_custom(func(a: BattleUnit, b: BattleUnit) -> bool:
				return a.base_position.x < b.base_position.x
			)
		"enemy":
			candidates.sort_custom(func(a: BattleUnit, b: BattleUnit) -> bool:
				return a.base_position.x > b.base_position.x
			)
		_:
			candidates.sort_custom(func(a: BattleUnit, b: BattleUnit) -> bool:
				return a.actor_id < b.actor_id
			)

func _cancel_target_selection() -> void:
	_clear_target_selection_visuals()
	_action_panel.set_target_selection_mode(false)
	_pending_selection.clear()
	_state = STATE_WAITING_INPUT
	if _planning_unit != null and not _interaction_locked:
		_action_panel.set_buttons_disabled(false)

func _commit_unit_selection(selection: Dictionary) -> void:
	var actor_id: int = int(selection.get("actor_id", 0))
	var action_type_name: String = str(selection.get("action_type", ""))
	var skill_id: int = int(selection.get("skill_id", 0))
	var target_ids: Array = selection.get("target_ids", []) as Array
	var target_id: int = 0
	if not target_ids.is_empty():
		target_id = int(target_ids[0])
	var action_type: int = _map_action_type(action_type_name)
	_state = STATE_SUBMITTING
	_lock_interaction("正在提交战斗指令")
	action_requested.emit(actor_id, action_type, skill_id, target_id)
	_selection_index += 1

func _submit_auto_toggle(enabled: bool) -> void:
	_lock_interaction("正在切换自动战斗")
	App.set_battle_auto(_network.get_battle_id(), _network.get_round(), enabled)


func _on_command_timeout_auto_request() -> void:
	if _network.is_auto_battle_enabled():
		return
	if _network.get_phase() != "command":
		return
	if _network.get_pending_actor_ids().is_empty():
		return
	_submit_auto_toggle(true)

func _map_action_type(action_type_name: String) -> int:
	match action_type_name:
		"attack", "skill":
			return 1
		"item":
			return 2
		"escape":
			return 4
		"auto":
			return 5
		_:
			return 1

func _pick_random_target_ids(selection: Dictionary) -> Array[int]:
	var target_count: int = max(1, int(selection.get("target_count", 1)))
	var candidates: Array[BattleUnit] = _get_target_candidates(str(selection.get("target_side", "enemy")))
	var result: Array[int] = []
	var pool: Array[BattleUnit] = candidates.duplicate()
	pool.shuffle()
	for index: int in range(min(target_count, pool.size())):
		result.append(pool[index].actor_id)
	return result

func _pick_all_target_ids(selection: Dictionary) -> Array[int]:
	var candidates: Array[BattleUnit] = _get_target_candidates(str(selection.get("target_side", "enemy")))
	var result: Array[int] = []
	for candidate: BattleUnit in candidates:
		result.append(candidate.actor_id)
	return result

func _get_target_candidates(target_side: String) -> Array[BattleUnit]:
	var result: Array[BattleUnit] = []
	for unit_id_value: Variant in _units.keys():
		var unit: BattleUnit = _get_unit(int(unit_id_value))
		if unit == null or unit.is_dead:
			continue
		if _is_target_candidate(unit, target_side):
			result.append(unit)
	return result

func _is_target_candidate(unit: BattleUnit, target_side: String) -> bool:
	if unit.is_dead:
		return false
	match target_side:
		"ally":
			return _is_ally_unit(unit)
		"enemy":
			return not _is_ally_unit(unit)
		"all":
			return true
		_:
			return false

func _pick_target_unit_at_mouse() -> BattleUnit:
	var mouse_pos: Vector2 = _get_mouse_position_in_unit_layer()
	var target_side: String = str(_pending_selection.get("target_side", "enemy"))
	var best_unit: BattleUnit = null
	var best_distance: float = INF
	for unit_id_value: Variant in _units.keys():
		var unit: BattleUnit = _get_unit(int(unit_id_value))
		if unit == null or not _is_target_candidate(unit, target_side):
			continue
		if not unit.contains_click_point(mouse_pos):
			continue
		var distance: float = mouse_pos.distance_squared_to(unit.position)
		if distance < best_distance:
			best_distance = distance
			best_unit = unit
	return best_unit

func _get_mouse_position_in_unit_layer() -> Vector2:
	var canvas_pos: Vector2 = get_viewport().get_canvas_transform().affine_inverse() * get_viewport().get_mouse_position()
	return _unit_layer.get_global_transform().affine_inverse() * canvas_pos

func _resolve_selection_arrow_type(target_side: String) -> String:
	match target_side:
		"ally":
			return BattleUnit.SELECTION_ARROW_ALLY
		"enemy":
			return BattleUnit.SELECTION_ARROW_ENEMY
		_:
			return BattleUnit.SELECTION_ARROW_ENEMY

func _update_target_selection_visuals() -> void:
	var target_side: String = str(_pending_selection.get("target_side", "enemy"))
	var arrow_type: String = _resolve_selection_arrow_type(target_side)
	for unit_id_value: Variant in _units.keys():
		var unit: BattleUnit = _get_unit(int(unit_id_value))
		if unit == null:
			continue
		var selectable: bool = _is_target_candidate(unit, target_side)
		unit.set_target_highlight(selectable, selectable)
		if unit.actor_id == _target_arrow_unit_id:
			unit.set_selection_arrow(arrow_type)
		else:
			unit.set_selection_arrow(BattleUnit.SELECTION_ARROW_NONE)
	_action_panel.set_target_confirm_enabled(_target_arrow_unit_id > 0)

func _clear_target_selection_visuals() -> void:
	_target_arrow_unit_id = 0
	for unit_id_value: Variant in _units.keys():
		var unit: BattleUnit = _get_unit(int(unit_id_value))
		if unit != null:
			unit.set_target_highlight(false)
			unit.clear_selection_arrows()

func _clear_target_highlights() -> void:
	_clear_target_selection_visuals()

func _format_target_names(target_ids: Array) -> String:
	var names: PackedStringArray = PackedStringArray()
	for target_id_value: Variant in target_ids:
		var target: BattleUnit = _get_unit(int(target_id_value))
		if target != null:
			names.append(target.unit_name)
		elif not str(target_id_value).is_empty():
			names.append(str(target_id_value))
	return "、".join(names)

func _format_selection_summary(selection: Dictionary) -> String:
	var action_name: String = str(selection.get("display_name", _display_action_name(str(selection.get("action_type", "")))))
	var target_text: String = _format_target_names(selection.get("target_ids", []) as Array)
	if target_text.is_empty():
		return action_name
	return "%s → %s" % [action_name, target_text]

func _play_state_events() -> void:
	var events: Array[Dictionary] = _network.get_events()
	if events.is_empty():
		return
	_is_playing_events = true
	_state = STATE_PLAYING
	_clear_planning_highlight()
	_actions_by_id.clear()
	_combos_by_id.clear()
	var round_data: Dictionary = BattleEventAdapter.build_round_data(
		events,
		_network.get_runtime_actors(),
		_network
	)
	var actions: Array = round_data.get("actions", []) as Array
	for action_value: Variant in actions:
		if action_value is Dictionary:
			var action: Dictionary = action_value as Dictionary
			_actions_by_id[str(action.get("id", ""))] = action
	var combos: Array = round_data.get("combo", []) as Array
	for combo_value: Variant in combos:
		if combo_value is Dictionary:
			var combo_entry: Dictionary = combo_value as Dictionary
			_combos_by_id[str(combo_entry.get("id", ""))] = combo_entry
	var timeline: Array = round_data.get("timeline", []) as Array
	for timeline_step_value: Variant in timeline:
		if timeline_step_value is Dictionary:
			await _play_timeline_step(timeline_step_value as Dictionary)
	_is_playing_events = false

func _sync_runtime_actors() -> void:
	for runtime_actor: Dictionary in _network.get_runtime_actors():
		var actor_id: int = int(runtime_actor.get("actor_id", 0))
		var unit: BattleUnit = _get_unit(actor_id)
		if unit == null:
			continue
		unit.apply_runtime_snapshot(runtime_actor)

func _clear_planning_highlight() -> void:
	_clear_target_highlights()
	_pending_selection.clear()
	_pending_skill_choices.clear()
	_pending_item_choices.clear()
	if _planning_unit != null:
		_planning_unit.set_planning_highlight(false)
		_planning_unit = null
	for unit_id_value: Variant in _units.keys():
		var unit: BattleUnit = _units[unit_id_value] as BattleUnit
		if unit != null:
			unit.set_planning_highlight(false)

func _play_timeline_step(timeline_step: Dictionary) -> void:
	var mode: String = str(timeline_step.get("mode", "serial"))
	var refs: Array[String] = []
	if timeline_step.has("action_refs"):
		var action_refs: Array = timeline_step.get("action_refs", []) as Array
		for ref_value: Variant in action_refs:
			refs.append(str(ref_value))
	elif timeline_step.has("action_ref"):
		refs.append(str(timeline_step.get("action_ref", "")))

	match mode:
		"parallel":
			await _play_parallel_refs(refs)
		_:
			for ref in refs:
				await _play_ref(ref)

	var wait_ms: int = int(timeline_step.get("wait_ms", 0))
	if wait_ms > 0:
		await get_tree().create_timer(float(wait_ms) / 1000.0).timeout

func _play_parallel_refs(refs: Array[String]) -> void:
	if refs.is_empty():
		return
	_parallel_remaining = refs.size()
	for ref in refs:
		_play_ref_in_parallel(ref)
	await parallel_group_finished

func _play_ref_in_parallel(ref: String) -> void:
	await _play_ref(ref)
	_parallel_remaining -= 1
	if _parallel_remaining <= 0:
		parallel_group_finished.emit()

func _play_ref(ref: String) -> void:
	if ref.is_empty():
		return
	if _actions_by_id.has(ref):
		await _play_action(_actions_by_id[ref])
		return
	if _combos_by_id.has(ref):
		await _play_combo(_combos_by_id[ref])
		return
	_action_log_panel.append_log("未找到表现引用：%s" % ref)

func _play_action(action: Dictionary) -> void:
	var actor: BattleUnit = _get_unit(int(action.get("actor_id", "")))
	var target_position: Vector2 = _resolve_primary_target_position(action)
	var display_name: String = str(action.get("display_name", "未命名动作"))
	var skill_visual: SkillVisualConfig = _resolve_skill_visual(action)
	var effect_name: String = _resolve_effect_name(action, skill_visual)
	var actor_name: String = actor.unit_name if actor != null else str(action.get("actor_id", ""))
	_action_log_panel.append_action_log(actor_name, str(action.get("log_text", display_name)))
	if actor != null:
		_show_caster_action_name(actor, _resolve_caster_action_label(action))
		await actor.play_attack(
			target_position,
			skill_visual,
			str(action.get("skill_id", "")),
			display_name,
			_resolve_action_type(action, skill_visual),
			str(action.get("animation", ""))
		)
	await _show_effect(
		skill_visual,
		effect_name,
		target_position,
		_should_mirror_skill_effect_at_target(actor, target_position)
	)
	var target_results: Array = action.get("targets", []) as Array
	for target_result_value: Variant in target_results:
		if target_result_value is Dictionary:
			var target_result: Dictionary = target_result_value
			await _apply_target_result(target_result)
	var buff_changes: Array = action.get("buff_changes", []) as Array
	for buff_change_value: Variant in buff_changes:
		if buff_change_value is Dictionary:
			var buff_change: Dictionary = buff_change_value
			_apply_buff_change(buff_change)
	var summon_changes: Array = action.get("summon_changes", []) as Array
	for summon_change_value: Variant in summon_changes:
		if summon_change_value is Dictionary:
			var summon_change: Dictionary = summon_change_value
			_apply_summon_change(summon_change)
	var extra_logs: Array = action.get("extra_logs", []) as Array
	for extra_log_value: Variant in extra_logs:
		_action_log_panel.append_log(str(extra_log_value))

func _play_combo(combo_entry: Dictionary) -> void:
	var actor: BattleUnit = _get_unit(int(combo_entry.get("actor_id", "")))
	var target: BattleUnit = _get_unit(int(combo_entry.get("target_id", "")))
	var display_name: String = str(combo_entry.get("display_name", "追加表现"))
	var skill_visual: SkillVisualConfig = _resolve_skill_visual(combo_entry)
	var effect_name: String = _resolve_effect_name(combo_entry, skill_visual)
	var actor_name: String = actor.unit_name if actor != null else str(combo_entry.get("actor_id", ""))
	_action_log_panel.append_action_log(actor_name, str(combo_entry.get("log_text", display_name)))
	if actor != null and target != null:
		_show_caster_action_name(actor, _resolve_caster_action_label(combo_entry))
		await actor.play_attack(
			target.base_position,
			skill_visual,
			str(combo_entry.get("skill_id", combo_entry.get("id", ""))),
			display_name,
			_resolve_action_type(combo_entry, skill_visual),
			str(combo_entry.get("animation", ""))
		)
	if target != null:
		await _show_effect(
			skill_visual,
			effect_name,
			target.base_position,
			_should_mirror_skill_effect_at_target(actor, target.base_position)
		)
		var result: Dictionary = {
			"target_id": int(combo_entry.get("target_id", 0)),
			"result_type": str(combo_entry.get("result_type", "damage")),
			"value": int(combo_entry.get("value", 0)),
			"hp_after": int(combo_entry.get("hp_after", 0)),
			"floating_text": str(combo_entry.get("floating_text", "")),
			"log_text": ""
		}
		await _apply_target_result(result)

func _apply_target_result(target_result: Dictionary) -> void:
	var target: BattleUnit = _get_unit(int(target_result.get("target_id", "")))
	if target == null:
		return
	var result_type: String = str(target_result.get("result_type", "damage"))
	var floating_text: String = str(target_result.get("floating_text", ""))
	if floating_text.is_empty() and result_type != "defeat":
		var damage_value: int = int(target_result.get("value", 0))
		if result_type == "heal" or damage_value > 0:
			var sign: String = "-"
			if result_type == "heal":
				sign = "+"
			floating_text = "%s%d" % [sign, damage_value]
	if not floating_text.is_empty():
		_start_floating_number(target.base_position + Vector2(20.0, -20.0), floating_text, result_type)
	if _should_shake_for_crit(target_result):
		await _shake_battlefield()
	await target.play_result(target_result)
	var target_log: String = str(target_result.get("log_text", ""))
	if not target_log.is_empty():
		_action_log_panel.append_log(target_log)

func _apply_buff_change(buff_change: Dictionary) -> void:
	var target: BattleUnit = _get_unit(int(buff_change.get("target_id", "")))
	if target == null:
		return
	target.apply_buff_change(buff_change)
	var log_text: String = str(buff_change.get("log_text", ""))
	if not log_text.is_empty():
		_action_log_panel.append_log(log_text)
	_show_floating_text(target.base_position + Vector2(-10.0, -60.0), str(buff_change.get("buff_id", "状态")), "buff")

func _apply_summon_change(summon_change: Dictionary) -> void:
	var change_type: String = str(summon_change.get("change_type", "add"))
	match change_type:
		"add":
			var unit_data: Dictionary = summon_change.get("unit", {}) as Dictionary
			if not unit_data.is_empty() and not _units.has(str(unit_data.get("id", ""))):
				_spawn_unit(unit_data)
		"remove":
			var actor_id: int = int(summon_change.get("unit_id", summon_change.get("actor_id", 0)))
			var unit: BattleUnit = _get_unit(actor_id)
			if unit != null:
				unit.queue_free()
				_units.erase(actor_id)
	var log_text: String = str(summon_change.get("log_text", ""))
	if not log_text.is_empty():
		_action_log_panel.append_log(log_text)

func _should_shake_for_crit(target_result: Dictionary) -> bool:
	if str(target_result.get("result_type", "")) != "damage":
		return false
	if bool(target_result.get("is_crit", false)):
		return true
	return str(target_result.get("hit_type", "")) == "crit"

func _shake_battlefield() -> void:
	var base_position: Vector2 = _battlefield_root.position
	var shake_offsets: Array[Vector2] = [Vector2(-8.0, 4.0), Vector2(10.0, -6.0), Vector2.ZERO]
	for offset: Vector2 in shake_offsets:
		var tween: Tween = create_tween()
		tween.tween_property(_battlefield_root, "position", base_position + offset, 0.05)
		await tween.finished

func _show_effect(
	skill_visual: SkillVisualConfig,
	effect_name: String,
	world_position: Vector2,
	flip_horizontal: bool = false
) -> void:
	if skill_visual == null and effect_name.is_empty():
		return
	var effect: BattleEffect = BattleEffectScene.instantiate() as BattleEffect
	effect.position = world_position + Vector2(0.0, 0.0)
	_effect_layer.add_child(effect)
	await effect.play_from_config(skill_visual, effect_name, flip_horizontal)

## 右侧站位攻击左侧目标时镜像命中特效，适配当前左右对阵布局。
func _should_mirror_skill_effect_at_target(actor: BattleUnit, target_position: Vector2) -> bool:
	if actor == null:
		return false
	return (
		actor.base_position.x > BattleFormationMapper.BATTLEFIELD_SPLIT_X
		and target_position.x < BattleFormationMapper.BATTLEFIELD_SPLIT_X
	)

func _show_caster_action_name(actor: BattleUnit, action_name: String) -> void:
	if actor == null or action_name.is_empty():
		return
	var anchor: Vector2 = actor.base_position + Vector2(0.0, -76.0)
	var label: FloatingText = FloatingTextScript.new() as FloatingText
	label.text = action_name
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	label.add_theme_font_size_override("font_size", 15)
	label.reset_size()
	label.position = anchor - Vector2(label.size.x * 0.5, 0.0)
	_floating_layer.add_child(label)
	label.play(action_name, _resolve_floating_text_color("action_name"))

## 取施法者头顶展示用的技能名；优先 display_name，普攻会显示「普通攻击」。
func _resolve_caster_action_label(action: Dictionary) -> String:
	var display_name: String = str(action.get("display_name", "")).strip_edges()
	if not display_name.is_empty():
		return display_name
	var action_type: String = str(action.get("action_type", ""))
	return _display_action_name(action_type)

func _start_floating_number(world_position: Vector2, value_text: String, result_type: String) -> void:
	var digit_float: BattleDigitFloat = BattleDigitFloatScript.new() as BattleDigitFloat
	digit_float.position = world_position
	_floating_layer.add_child(digit_float)
	digit_float.play(value_text, _resolve_digit_float_color(result_type))

func _resolve_digit_float_color(result_type: String) -> Color:
	# 位图数字本身带描边与渐变，默认保持原色，治疗只做轻微偏绿。
	match result_type:
		"heal":
			return Color(0.92, 1.0, 0.92, 1.0)
		_:
			return Color.WHITE

func _start_floating_text(world_position: Vector2, value: String, result_type: String) -> void:
	var label: FloatingText = _create_floating_text(world_position)
	label.play(value, _resolve_floating_text_color(result_type))

func _create_floating_text(world_position: Vector2) -> FloatingText:
	var label: FloatingText = FloatingTextScript.new() as FloatingText
	label.position = world_position
	_floating_layer.add_child(label)
	return label

func _resolve_floating_text_color(result_type: String) -> Color:
	match result_type:
		"heal":
			return Color("#d7ffd9")
		"buff":
			return Color("#fff59d")
		"action_name":
			return Color("#fff4b8")
		_:
			return Color("#ffede7")

func _show_floating_text(world_position: Vector2, value: String, result_type: String) -> void:
	var label: FloatingText = _create_floating_text(world_position)
	await label.play(value, _resolve_floating_text_color(result_type))

func _resolve_primary_target_position(action: Dictionary) -> Vector2:
	var target_results: Array = action.get("targets", []) as Array
	for target_result_value: Variant in target_results:
		if target_result_value is Dictionary:
			var target_result: Dictionary = target_result_value
			var target: BattleUnit = _get_unit(int(target_result.get("target_id", "")))
			if target != null:
				return target.base_position
	return _get_battlefield_center() + Vector2(0.0, -40.0)

func _get_battlefield_center() -> Vector2:
	return get_viewport().get_visible_rect().size * 0.5

func _get_slot_position(position_key: String) -> Vector2:
	if _slot_positions.has(position_key):
		var slot_position_value: Variant = _slot_positions[position_key]
		if slot_position_value is Vector2:
			return slot_position_value as Vector2
	push_warning("未找到站位 key: %s，请检查 battle_demo.json 的 formation 配置。" % position_key)
	return Vector2.ZERO

func _resolve_skill_visual(data: Dictionary) -> SkillVisualConfig:
	var skill_visual_id: String = str(data.get("skill_visual_id", ""))
	if not skill_visual_id.is_empty():
		var visual: SkillVisualConfig = _content_registry.get_skill_visual(skill_visual_id)
		if visual != null:
			return visual
	var animation_key: String = str(data.get("animation_key", ""))
	if animation_key.is_empty():
		return null
	return _content_registry.get_skill_visual(animation_key)

func _resolve_action_type(data: Dictionary, skill_visual: SkillVisualConfig) -> String:
	if skill_visual != null and not skill_visual.action_type.is_empty():
		return skill_visual.action_type
	var fallback_action_type: String = str(data.get("action_type", ""))
	if fallback_action_type.is_empty():
		fallback_action_type = str(data.get("trigger", "attack"))
	return fallback_action_type

func _resolve_effect_name(data: Dictionary, skill_visual: SkillVisualConfig) -> String:
	if skill_visual != null and not skill_visual.effect_id.is_empty():
		return skill_visual.effect_id
	var effect_name: String = str(data.get("effect", ""))
	if not effect_name.is_empty():
		return effect_name
	return str(data.get("display_name", ""))

func _get_unit(actor_id: int) -> BattleUnit:
	if not _units.has(actor_id):
		return null
	return _units[actor_id] as BattleUnit

func _display_action_name(action_type: String) -> String:
	match action_type:
		"attack":
			return "普通攻击"
		"skill":
			return "技能"
		"item":
			return "物品"
		"auto":
			return "自动战斗"
		"escape":
			return "逃跑"
		_:
			return action_type

func _finish_battle(summary: String) -> void:
	_state = STATE_BATTLE_END
	_action_panel.set_buttons_disabled(true)
	_action_log_panel.append_log(summary)
