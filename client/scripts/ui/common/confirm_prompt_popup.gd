class_name ConfirmPromptPopup
extends "res://scripts/ui/common/modal_popup_layer.gd"

## 通用确认提示面板场景路径。
const SCENE_PATH: String = "res://scenes/ui/common/confirm_prompt_popup.tscn"
## 默认标题文案。
const DEFAULT_TITLE: String = "提示"
## 默认确定按钮文案。
const DEFAULT_CONFIRM_LABEL: String = "确定"
## 默认取消按钮文案。
const DEFAULT_CANCEL_LABEL: String = "取消"

## 玩家点击确定后向外广播。
signal confirmed
## 玩家点击取消或点击遮罩关闭后向外广播。
signal cancelled

## 标题标签。
@onready var _title_label: Label = %TitleLabel
## 支持 BBCode 的正文区域。
@onready var _content_label: RichTextLabel = %ContentLabel
## 数量选择行容器。
@onready var _quantity_row: HBoxContainer = %QuantityRow
## 数量减一按钮。
@onready var _quantity_decrease_button: RuntimeActionButton = %QuantityDecreaseButton
## 当前选中数量标签。
@onready var _quantity_value_label: Label = %QuantityValueLabel
## 数量加一按钮。
@onready var _quantity_increase_button: RuntimeActionButton = %QuantityIncreaseButton
## 数量设为最大按钮。
@onready var _quantity_max_button: RuntimeActionButton = %QuantityMaxButton
## 左侧确定按钮。
@onready var _confirm_button: RuntimeActionButton = %ConfirmButton
## 右侧取消按钮。
@onready var _cancel_button: RuntimeActionButton = %CancelButton

## 当前确认弹窗选中的丢弃/操作数量。
var _selected_quantity: int = 1
## 数量选择器允许的最小值。
var _quantity_min: int = 1
## 数量选择器允许的最大值。
var _quantity_max: int = 1


## 绑定按钮信号；默认隐藏，由调用方按需打开。
func _ready() -> void:
	super._ready()
	if _confirm_button != null and not _confirm_button.pressed.is_connected(_on_confirm_pressed):
		_confirm_button.pressed.connect(_on_confirm_pressed)
	if _cancel_button != null and not _cancel_button.pressed.is_connected(_on_cancel_pressed):
		_cancel_button.pressed.connect(_on_cancel_pressed)
	if _quantity_decrease_button != null and not _quantity_decrease_button.pressed.is_connected(_on_quantity_decrease_pressed):
		_quantity_decrease_button.pressed.connect(_on_quantity_decrease_pressed)
	if _quantity_increase_button != null and not _quantity_increase_button.pressed.is_connected(_on_quantity_increase_pressed):
		_quantity_increase_button.pressed.connect(_on_quantity_increase_pressed)
	if _quantity_max_button != null and not _quantity_max_button.pressed.is_connected(_on_quantity_max_pressed):
		_quantity_max_button.pressed.connect(_on_quantity_max_pressed)


## 展示确认提示面板。
## config 可选键：confirm_label、cancel_label、show_cancel、show_quantity_picker、min_quantity、max_quantity、initial_quantity。
func show_prompt(title_text: String, content_bbcode: String, config: Dictionary = {}) -> void:
	var resolved_title: String = title_text.strip_edges()
	if resolved_title.is_empty():
		resolved_title = DEFAULT_TITLE
	if _title_label != null:
		_title_label.text = resolved_title
	if _content_label != null:
		_content_label.clear()
		_content_label.bbcode_enabled = true
		_content_label.text = content_bbcode
	var confirm_label: String = str(config.get("confirm_label", DEFAULT_CONFIRM_LABEL))
	var cancel_label: String = str(config.get("cancel_label", DEFAULT_CANCEL_LABEL))
	var show_cancel: bool = bool(config.get("show_cancel", true))
	if _confirm_button != null:
		_confirm_button.set_button_label(confirm_label)
	if _cancel_button != null:
		_cancel_button.set_button_label(cancel_label)
		_cancel_button.visible = show_cancel
	_apply_quantity_picker_config(config)
	_apply_interactive_nodes()
	_open_modal()


## 返回玩家最终确认的数量；未启用数量选择器时恒为 1。
func get_selected_quantity() -> int:
	return _selected_quantity


## 关闭确认提示面板。
func close_prompt() -> void:
	if not visible:
		return
	get_viewport().set_input_as_handled()
	_notify_host_suppress_input_leak()
	_close_modal()


## 点击遮罩时视为取消，不沿用基类“任意关闭”语义。
func _dismiss_modal() -> void:
	_close_with_result(false, "backdrop")


## 确认弹窗需要按钮与遮罩走 GUI 点击；基类全局 _input 会先 mark handled，导致确定/取消无效。
func _enable_modal_input_listeners() -> void:
	pass


func _disable_modal_input_listeners() -> void:
	pass


## 丢弃/确认类弹窗需比通用模态更长的空白关闭冷却，避免链式 UI 同一次触屏误关。
func _can_dismiss_modal_now() -> bool:
	if _modal_open_frame < 0:
		return true
	return Engine.get_process_frames() > _modal_open_frame + 4


## 让面板与按钮可交互，同时保留遮罩点击取消能力。
func _apply_interactive_nodes() -> void:
	var panel: Control = get_node_or_null("CenterContainer/PanelContainer") as Control
	if panel != null:
		panel.mouse_filter = Control.MOUSE_FILTER_STOP
	if _confirm_button != null:
		_confirm_button.mouse_filter = Control.MOUSE_FILTER_STOP
	if _cancel_button != null:
		_cancel_button.mouse_filter = Control.MOUSE_FILTER_STOP
	if _quantity_decrease_button != null:
		_quantity_decrease_button.mouse_filter = Control.MOUSE_FILTER_STOP
	if _quantity_increase_button != null:
		_quantity_increase_button.mouse_filter = Control.MOUSE_FILTER_STOP
	if _quantity_max_button != null:
		_quantity_max_button.mouse_filter = Control.MOUSE_FILTER_STOP


## 根据 config 初始化数量选择器显示与边界。
func _apply_quantity_picker_config(config: Dictionary) -> void:
	var show_quantity_picker: bool = bool(config.get("show_quantity_picker", false))
	_quantity_min = maxi(1, int(config.get("min_quantity", 1)))
	_quantity_max = maxi(_quantity_min, int(config.get("max_quantity", _quantity_min)))
	_selected_quantity = clampi(int(config.get("initial_quantity", _quantity_max)), _quantity_min, _quantity_max)
	if _quantity_row != null:
		_quantity_row.visible = show_quantity_picker
	_refresh_quantity_label()


## 刷新数量标签文案。
func _refresh_quantity_label() -> void:
	if _quantity_value_label != null:
		_quantity_value_label.text = str(_selected_quantity)


## 处理确定按钮点击。
func _on_confirm_pressed() -> void:
	_close_with_result(true, "confirm_button")


## 处理取消按钮点击。
func _on_cancel_pressed() -> void:
	_close_with_result(false, "cancel_button")


## 数量减一。
func _on_quantity_decrease_pressed() -> void:
	_selected_quantity = clampi(_selected_quantity - 1, _quantity_min, _quantity_max)
	_refresh_quantity_label()


## 数量加一。
func _on_quantity_increase_pressed() -> void:
	_selected_quantity = clampi(_selected_quantity + 1, _quantity_min, _quantity_max)
	_refresh_quantity_label()


## 数量设为最大。
func _on_quantity_max_pressed() -> void:
	_selected_quantity = _quantity_max
	_refresh_quantity_label()


## 根据玩家选择关闭面板并广播对应信号。
## 先广播结果再 hide，避免调用方用 visible 轮询时在信号写入前误判为取消。
func _close_with_result(is_confirmed: bool, _source: String = "unknown") -> void:
	if not visible:
		return
	get_viewport().set_input_as_handled()
	_notify_host_suppress_input_leak()
	if is_confirmed:
		confirmed.emit()
	else:
		cancelled.emit()
	popup_closed.emit()
	_close_modal()
