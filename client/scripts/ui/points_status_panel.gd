extends PanelContainer

const StatusPanelDataProvider = preload("res://scripts/data/status_panel_data_provider.gd")

const LOADING_DISPLAY_DELAY_SEC: float = 1.0
const LOADING_FRAME_INTERVAL_SEC: float = 0.2
const LOADING_TEXT_FRAMES: Array[String] = ["加载中   ", "加载中.  ", "加载中.. ", "加载中..."]

var _default_data: Dictionary = {}
var _loading_request_seq: int = 0
var _loading_text_index: int = 0
var _loading_overlay: Control = null
var _loading_label: Label = null
var _loading_delay_timer: Timer = null
var _loading_timer: Timer = null
var _allocate_buttons: Array[Button] = []

@onready var free_points_value: Label = get_node_or_null("PointsMargin/PointsVBox/FreePointsRow/FreePointsValue")
@onready var hint_text: Label = get_node_or_null("PointsMargin/PointsVBox/PointHint")
@onready var strength_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid1/StrengthBox/StrengthValue")
@onready var vitality_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid1/VitalityBox/VitalityValue")
@onready var agility_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid2/AgilityBox/AgilityValue")
@onready var mind_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid2/MindBox/MindValue")
@onready var advice_text: Label = get_node_or_null("PointsMargin/PointsVBox/PointAdviceRow/PointAdviceText")
@onready var plan_text: Label = get_node_or_null("PointsMargin/PointsVBox/PointPlanRow/PointPlanText")
@onready var strength_box: HBoxContainer = get_node_or_null("PointsMargin/PointsVBox/PointsGrid1/StrengthBox")
@onready var vitality_box: HBoxContainer = get_node_or_null("PointsMargin/PointsVBox/PointsGrid1/VitalityBox")
@onready var agility_box: HBoxContainer = get_node_or_null("PointsMargin/PointsVBox/PointsGrid2/AgilityBox")
@onready var mind_box: HBoxContainer = get_node_or_null("PointsMargin/PointsVBox/PointsGrid2/MindBox")


func _ready() -> void:
	_default_data = StatusPanelDataProvider.get_section("points")
	_build_loading_overlay()
	_build_allocate_buttons()
	apply_data(_default_data)


func apply_data(data: Dictionary) -> void:
	var resolved := _default_data.duplicate(true)
	for key in data.keys():
		resolved[key] = UiFormat.value_to_text(data[key])

	_set_label_text(free_points_value, UiFormat.value_to_text(resolved.get("free_points", "")))
	_set_label_text(hint_text, UiFormat.value_to_text(resolved.get("hint", "")))
	_set_label_text(strength_value, UiFormat.value_to_text(resolved.get("strength", "")))
	_set_label_text(vitality_value, UiFormat.value_to_text(resolved.get("vitality", "")))
	_set_label_text(agility_value, UiFormat.value_to_text(resolved.get("agility", "")))
	_set_label_text(mind_value, UiFormat.value_to_text(resolved.get("mind", "")))
	_set_label_text(advice_text, UiFormat.value_to_text(resolved.get("advice", "")))
	_set_label_text(plan_text, UiFormat.value_to_text(resolved.get("plan", "")))


func _build_allocate_buttons() -> void:
	_attach_allocate_button(strength_box, Callable(self, "_on_allocate_strength_pressed"))
	_attach_allocate_button(vitality_box, Callable(self, "_on_allocate_vitality_pressed"))
	_attach_allocate_button(agility_box, Callable(self, "_on_allocate_agility_pressed"))
	_attach_allocate_button(mind_box, Callable(self, "_on_allocate_mind_pressed"))


func _attach_allocate_button(container: HBoxContainer, callback: Callable) -> void:
	if container == null:
		return
	var existing: Node = container.get_node_or_null("AllocateButton")
	if existing != null:
		return
	var button: Button = Button.new()
	button.name = "AllocateButton"
	button.text = "+1"
	button.custom_minimum_size = Vector2(44, 28)
	button.pressed.connect(callback)
	container.add_child(button)
	_allocate_buttons.append(button)


func _on_allocate_strength_pressed() -> void:
	_request_allocate_attr_points(1, 0, 0, 0)


func _on_allocate_vitality_pressed() -> void:
	_request_allocate_attr_points(0, 1, 0, 0)


func _on_allocate_agility_pressed() -> void:
	_request_allocate_attr_points(0, 0, 1, 0)


func _on_allocate_mind_pressed() -> void:
	_request_allocate_attr_points(0, 0, 0, 1)


func _request_allocate_attr_points(strength: int, vitality: int, agility: int, mind: int) -> void:
	if _loading_request_seq != 0:
		return
	if not GameState.is_ws_authenticated:
		return
	var request_seq: int = App.request_allocate_attr_points(strength, vitality, agility, mind)
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	_set_allocate_buttons_disabled(true)
	_show_loading_overlay()
	call_deferred("_wait_allocate_request", request_seq)


func _wait_allocate_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PLAYER_ALLOCATE_ATTR_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	_hide_loading_overlay()


func _build_loading_overlay() -> void:
	_loading_overlay = Control.new()
	_loading_overlay.name = "AllocateLoadingOverlay"
	_loading_overlay.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_loading_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(_loading_overlay)

	var dim_layer: ColorRect = ColorRect.new()
	dim_layer.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	dim_layer.color = Color(0.06, 0.1, 0.16, 0.82)
	_loading_overlay.add_child(dim_layer)

	_loading_label = Label.new()
	_loading_label.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
	_loading_label.position = Vector2(-60, -12)
	_loading_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_loading_label.add_theme_font_size_override("font_size", 18)
	_loading_label.text = LOADING_TEXT_FRAMES[0]
	_loading_overlay.add_child(_loading_label)

	_loading_delay_timer = Timer.new()
	_loading_delay_timer.wait_time = LOADING_DISPLAY_DELAY_SEC
	_loading_delay_timer.one_shot = true
	_loading_delay_timer.timeout.connect(_on_loading_delay_timeout)
	add_child(_loading_delay_timer)

	_loading_timer = Timer.new()
	_loading_timer.wait_time = LOADING_FRAME_INTERVAL_SEC
	_loading_timer.timeout.connect(_on_loading_timer_timeout)
	add_child(_loading_timer)

	_loading_overlay.hide()


func _show_loading_overlay() -> void:
	_loading_text_index = 0
	if _loading_label != null:
		_loading_label.text = LOADING_TEXT_FRAMES[_loading_text_index]
	if _loading_overlay != null:
		_loading_overlay.hide()
	if _loading_delay_timer != null:
		_loading_delay_timer.start()


func _hide_loading_overlay() -> void:
	if _loading_delay_timer != null:
		_loading_delay_timer.stop()
	if _loading_timer != null:
		_loading_timer.stop()
	if _loading_overlay != null:
		_loading_overlay.hide()
	_set_allocate_buttons_disabled(false)


func _on_loading_delay_timeout() -> void:
	if _loading_request_seq == 0:
		return
	if _loading_overlay != null:
		_loading_overlay.show()
	if _loading_timer != null:
		_loading_timer.start()


func _on_loading_timer_timeout() -> void:
	_loading_text_index = (_loading_text_index + 1) % LOADING_TEXT_FRAMES.size()
	if _loading_label != null:
		_loading_label.text = LOADING_TEXT_FRAMES[_loading_text_index]


func _set_allocate_buttons_disabled(disabled: bool) -> void:
	for button in _allocate_buttons:
		if button != null:
			button.disabled = disabled


func _set_label_text(label: Label, value: String) -> void:
	if label == null:
		return
	label.text = value
