extends Control
class_name BattleRequestLoading

## 战斗场景顶部等待提示：请求超过阈值后才显示一行文字，不遮挡回合演出。

const DISPLAY_DELAY_SEC: float = 2.0
const DEFAULT_TIP_TEXT: String = "加载中......"

var _tip_label: Label = null
var _delay_timer: Timer = null
var _is_waiting: bool = false

func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	anchor_right = 1.0
	offset_bottom = 36.0
	_build_ui()
	hide()

## 开始等待服务端回包；短于 DISPLAY_DELAY_SEC 的请求不会显示提示。
func show_waiting(tip_text: String = DEFAULT_TIP_TEXT) -> void:
	_is_waiting = true
	if _tip_label != null:
		_tip_label.text = tip_text
	hide()
	if _delay_timer != null:
		_delay_timer.start()

## 收到服务端数据后立即隐藏提示。
func hide_waiting() -> void:
	_is_waiting = false
	if _delay_timer != null:
		_delay_timer.stop()
	hide()

func _build_ui() -> void:
	var top_bar: ColorRect = ColorRect.new()
	top_bar.name = "TopBar"
	top_bar.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	top_bar.color = Color(0.04, 0.08, 0.14, 0.55)
	top_bar.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(top_bar)

	_tip_label = Label.new()
	_tip_label.name = "TipLabel"
	_tip_label.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_tip_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_tip_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	_tip_label.add_theme_font_size_override("font_size", 16)
	_tip_label.text = DEFAULT_TIP_TEXT
	_tip_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(_tip_label)

	_delay_timer = Timer.new()
	_delay_timer.wait_time = DISPLAY_DELAY_SEC
	_delay_timer.one_shot = true
	_delay_timer.timeout.connect(_on_delay_timeout)
	add_child(_delay_timer)

func _on_delay_timeout() -> void:
	if not _is_waiting:
		return
	show()
