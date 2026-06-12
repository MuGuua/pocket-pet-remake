extends CanvasLayer

signal menu_closed

const TAB_NORMAL_COLOR := Color(1, 1, 1, 1)
const TAB_PRESSED_COLOR := Color(0.95, 0.97, 0.84, 1)
const LOADING_DISPLAY_DELAY_SEC := 1.0
const LOADING_FRAME_INTERVAL_SEC := 0.2
const LOADING_TEXT_FRAMES := ["加载中   ", "加载中.  ", "加载中.. ", "加载中..."]
const TAB_STATUS_INDEX := 0
const TAB_BAG_INDEX := 1
const TAB_TEAM_INDEX := 2
const TAB_SKILL_INDEX := 3
const BAG_CONTEXT_BAG := "bag"
const BAG_CONTEXT_WAREHOUSE := "warehouse"

@onready var status_button: Button = $RootPanel/ContentRow/TabButtons/StatusTabButton
@onready var bag_button: Button = $RootPanel/ContentRow/TabButtons/BagTabButton
@onready var team_button: Button = $RootPanel/ContentRow/TabButtons/TeamTabButton
@onready var skill_button: Button = $RootPanel/ContentRow/TabButtons/SkillTabButton
@onready var root_panel: PanelContainer = $RootPanel

@onready var status_panel: Control = $RootPanel/ContentRow/PanelHost/StatusPanel
@onready var bag_panel: Control = $RootPanel/ContentRow/PanelHost/BagPanel
@onready var team_panel: Control = $RootPanel/ContentRow/PanelHost/TeamPanel
@onready var skill_panel: Control = $RootPanel/ContentRow/PanelHost/SkillPanel

var _tab_buttons: Array[Button] = []
var _tab_panels: Array[Control] = []
var _current_tab_index: int = 0
var _loading_request_seq: int = 0
var _loading_request_cmd: int = 0
var _loading_target_tab_index: int = TAB_STATUS_INDEX
var _loading_text_index: int = 0
var _loading_overlay: Control
var _loading_label: Label
var _loading_tip_label: Label
var _loading_delay_timer: Timer
var _loading_timer: Timer
var _bag_context: String = BAG_CONTEXT_BAG
var _bag_button_default_text: String = "背包"
var _warehouse_entity_id: int = 0


func _ready() -> void:
	hide()
	_tab_buttons = [status_button, bag_button, team_button, skill_button]
	_tab_panels = [status_panel, bag_panel, team_panel, skill_panel]

	status_button.pressed.connect(_on_tab_pressed.bind(0))
	bag_button.pressed.connect(_on_tab_pressed.bind(1))
	team_button.pressed.connect(_on_tab_pressed.bind(2))
	skill_button.pressed.connect(_on_tab_pressed.bind(3))

	_build_loading_overlay()
	_bag_button_default_text = bag_button.text
	if bag_panel != null:
		if bag_panel.has_signal("transfer_requested"):
			bag_panel.connect("transfer_requested", Callable(self, "_on_bag_transfer_requested"))
		if bag_panel.has_signal("container_switch_requested"):
			bag_panel.connect("container_switch_requested", Callable(self, "_on_bag_container_switch_requested"))
	_select_tab(0)


func open_menu() -> void:
	_warehouse_entity_id = 0
	_set_bag_context(BAG_CONTEXT_BAG)
	show()
	_request_tab_open(TAB_STATUS_INDEX)


func open_warehouse_menu(entity_id: int = 0) -> void:
	_warehouse_entity_id = entity_id
	_set_bag_context(BAG_CONTEXT_WAREHOUSE)
	show()
	_request_tab_open(TAB_BAG_INDEX)


func set_player_source(player_source: Node) -> void:
	if status_panel != null and status_panel.has_method("set_player_source"):
		status_panel.call("set_player_source", player_source)


func close_menu() -> void:
	var was_visible := visible
	_loading_request_seq = 0
	_loading_request_cmd = 0
	_hide_loading_overlay()
	hide()
	_warehouse_entity_id = 0
	_set_bag_context(BAG_CONTEXT_BAG)
	if was_visible:
		menu_closed.emit()


func _on_tab_pressed(index: int) -> void:
	_request_tab_open(index)


func _select_tab(index: int) -> void:
	if index < 0 or index >= _tab_buttons.size():
		return

	_current_tab_index = index
	for i in range(_tab_buttons.size()):
		var is_current := i == _current_tab_index
		var button := _tab_buttons[i]
		var panel := _tab_panels[i]
		if button != null:
			button.button_pressed = is_current
			button.modulate = TAB_PRESSED_COLOR if is_current else TAB_NORMAL_COLOR
		if panel != null:
			panel.visible = is_current


func _build_loading_overlay() -> void:
	# 运行时构建 loading 遮罩，尽量避免大改 .tscn，同时后续别的请求入口也能复用同样交互。
	_loading_overlay = Control.new()
	_loading_overlay.name = "LoadingOverlay"
	_loading_overlay.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_loading_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(_loading_overlay)

	var dim_layer := ColorRect.new()
	dim_layer.name = "DimLayer"
	dim_layer.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	dim_layer.color = Color(0.06, 0.1, 0.16, 0.82)
	_loading_overlay.add_child(dim_layer)

	var center_box := VBoxContainer.new()
	center_box.name = "CenterBox"
	center_box.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
	center_box.position = Vector2(-90, -24)
	center_box.custom_minimum_size = Vector2(180, 48)
	center_box.alignment = BoxContainer.ALIGNMENT_CENTER
	_loading_overlay.add_child(center_box)

	_loading_label = Label.new()
	_loading_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	# 运行时创建的 Label 需要使用主题覆盖接口，不能直接写场景序列化属性。
	_loading_label.add_theme_font_size_override("font_size", 22)
	_loading_label.text = LOADING_TEXT_FRAMES[0]
	center_box.add_child(_loading_label)

	_loading_tip_label = Label.new()
	_loading_tip_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_loading_tip_label.text = "正在同步服务端人物属性"
	center_box.add_child(_loading_tip_label)

	# 超过 1 秒才显示 loading，避免短请求时 UI 闪烁；在这之前界面保持静默等待。
	_loading_delay_timer = Timer.new()
	_loading_delay_timer.wait_time = LOADING_DISPLAY_DELAY_SEC
	_loading_delay_timer.one_shot = true
	_loading_delay_timer.timeout.connect(_on_loading_delay_timeout)
	add_child(_loading_delay_timer)

	_loading_timer = Timer.new()
	_loading_timer.wait_time = LOADING_FRAME_INTERVAL_SEC
	_loading_timer.one_shot = false
	_loading_timer.timeout.connect(_on_loading_timer_timeout)
	add_child(_loading_timer)

	_loading_overlay.hide()


func _show_loading_overlay() -> void:
	root_panel.hide()
	_set_tab_buttons_disabled(true)
	_loading_text_index = 0
	_loading_label.text = LOADING_TEXT_FRAMES[_loading_text_index]
	_loading_overlay.hide()
	_loading_delay_timer.start()


func _hide_loading_overlay() -> void:
	if _loading_delay_timer != null:
		_loading_delay_timer.stop()
	if _loading_timer != null:
		_loading_timer.stop()
	if _loading_overlay != null:
		_loading_overlay.hide()
	_set_tab_buttons_disabled(false)


func _finish_open_after_loading() -> void:
	_hide_loading_overlay()
	root_panel.show()
	_select_tab(_loading_target_tab_index)
	_refresh_target_tab(_loading_target_tab_index)


func _wait_player_status_request(expected_seq: int, expected_cmd: int, target_tab_index: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd := int(result[0])
		var seq := int(result[1])
		if request_cmd != expected_cmd or seq != expected_seq:
			continue
		break

	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	_loading_request_cmd = 0
	_loading_target_tab_index = target_tab_index
	if visible:
		_finish_open_after_loading()


func _on_loading_timer_timeout() -> void:
	_loading_text_index = (_loading_text_index + 1) % LOADING_TEXT_FRAMES.size()
	_loading_label.text = LOADING_TEXT_FRAMES[_loading_text_index]


func _on_loading_delay_timeout() -> void:
	# 只有请求仍未完成时才真正展示 loading，避免小于 1 秒的请求出现闪屏。
	if _loading_request_seq == 0:
		return
	_loading_overlay.show()
	_loading_timer.start()


func _request_tab_open(index: int) -> void:
	if _loading_request_seq != 0:
		return

	if not GameState.is_ws_authenticated:
		_loading_target_tab_index = index
		_finish_open_after_loading()
		return

	var request_cmd := _request_cmd_for_tab(index)
	var request_seq := _send_request_for_tab(index)
	_loading_target_tab_index = index
	if request_cmd == 0 or request_seq <= 0:
		_finish_open_after_loading()
		return

	_loading_request_cmd = request_cmd
	_loading_request_seq = request_seq
	_loading_tip_label.text = _loading_tip_for_tab(index)
	_show_loading_overlay()
	call_deferred("_wait_player_status_request", request_seq, request_cmd, index)


func _request_cmd_for_tab(index: int) -> int:
	match index:
		TAB_STATUS_INDEX:
			return CommandIds.ENTER_WORLD_REQ
		TAB_BAG_INDEX:
			return CommandIds.CONTAINER_LIST_REQ if _bag_context == BAG_CONTEXT_WAREHOUSE else CommandIds.BAG_LIST_REQ
		TAB_TEAM_INDEX:
			return CommandIds.PET_LIST_REQ
		TAB_SKILL_INDEX:
			return CommandIds.ENTER_WORLD_REQ
		_:
			return 0


func _send_request_for_tab(index: int) -> int:
	match index:
		TAB_STATUS_INDEX:
			return App.refresh_player_status()
		TAB_BAG_INDEX:
			return App.request_container_list(BAG_CONTEXT_WAREHOUSE) if _bag_context == BAG_CONTEXT_WAREHOUSE else App.request_bag_list()
		TAB_TEAM_INDEX:
			return App.request_pet_list()
		TAB_SKILL_INDEX:
			return App.refresh_player_status()
		_:
			return 0


func _loading_tip_for_tab(index: int) -> String:
	match index:
		TAB_STATUS_INDEX:
			return "正在同步服务端人物属性"
		TAB_BAG_INDEX:
			return "正在同步服务端仓库数据" if _bag_context == BAG_CONTEXT_WAREHOUSE else "正在同步服务端背包数据"
		TAB_TEAM_INDEX:
			return "正在同步服务端队伍数据"
		TAB_SKILL_INDEX:
			return "正在同步服务端技能数据"
		_:
			return "正在同步服务端数据"


func _refresh_target_tab(index: int) -> void:
	match index:
		TAB_STATUS_INDEX:
			if status_panel != null and status_panel.has_method("reset_to_default"):
				status_panel.call("reset_to_default")
			if status_panel != null and status_panel.has_method("refresh_panel_data"):
				status_panel.call("refresh_panel_data")
		TAB_BAG_INDEX:
			if bag_panel != null and bag_panel.has_method("refresh_panel_data"):
				bag_panel.call("refresh_panel_data")
		TAB_TEAM_INDEX:
			if team_panel != null and team_panel.has_method("refresh_panel_data"):
				team_panel.call("refresh_panel_data")
		TAB_SKILL_INDEX:
			if skill_panel != null and skill_panel.has_method("refresh_panel_data"):
				skill_panel.call("refresh_panel_data")


func _set_tab_buttons_disabled(disabled: bool) -> void:
	for button in _tab_buttons:
		if button != null:
			button.disabled = disabled


func _set_bag_context(context: String) -> void:
	_bag_context = context if context == BAG_CONTEXT_WAREHOUSE else BAG_CONTEXT_BAG
	if bag_button != null:
		bag_button.text = "仓库" if _bag_context == BAG_CONTEXT_WAREHOUSE else _bag_button_default_text
	if bag_panel != null and bag_panel.has_method("set_container_context"):
		var title := "仓库" if _bag_context == BAG_CONTEXT_WAREHOUSE else "背包"
		bag_panel.call("set_container_context", _bag_context, title)
	if bag_panel != null and bag_panel.has_method("set_warehouse_available"):
		bag_panel.call("set_warehouse_available", _warehouse_entity_id > 0)


func _on_bag_container_switch_requested(target_container_type: String) -> void:
	if target_container_type == BAG_CONTEXT_WAREHOUSE:
		if _warehouse_entity_id <= 0:
			return
		_set_bag_context(BAG_CONTEXT_WAREHOUSE)
	else:
		_set_bag_context(BAG_CONTEXT_BAG)
	_request_tab_open(TAB_BAG_INDEX)


func _on_bag_transfer_requested(source_container_type: String, slot_index: int, quantity: int) -> void:
	if slot_index <= 0 or quantity <= 0 or _warehouse_entity_id <= 0:
		return
	if source_container_type == BAG_CONTEXT_WAREHOUSE:
		App.request_warehouse_to_bag(_warehouse_entity_id, slot_index, quantity)
	else:
		App.request_bag_to_warehouse(_warehouse_entity_id, slot_index, quantity)
