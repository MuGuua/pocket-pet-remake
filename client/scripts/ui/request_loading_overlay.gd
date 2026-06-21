class_name RequestLoadingOverlay
extends CanvasLayer

## 通用服务端请求 loading 遮罩：短请求不闪屏，长请求才展示半透明等待层。

const LOADING_DISPLAY_DELAY_SEC: float = 1.0
const LOADING_FRAME_INTERVAL_SEC: float = 0.2
const LOADING_TEXT_FRAMES: Array[String] = ["加载中   ", "加载中.  ", "加载中.. ", "加载中..."]

## 遮罩根节点，用于拦截底层点击。
var _root: Control = null
## 主 loading 文案。
var _loading_label: Label = null
## 次级提示文案，例如“正在获取 NPC 菜单”。
var _tip_label: Label = null
## 延迟展示 loading 的计时器，避免短请求闪屏。
var _delay_timer: Timer = null
## 文案轮播计时器。
var _frame_timer: Timer = null
## 当前轮播帧索引。
var _frame_index: int = 0
## 是否仍处于等待服务端回包状态。
var _is_waiting: bool = false

## 初始化遮罩层并默认隐藏。
func _ready() -> void:
	layer = 40
	_build_ui()
	hide_overlay()

## 开始等待服务端回包；tip_text 用于说明当前请求用途。
func show_waiting(tip_text: String) -> void:
	_is_waiting = true
	_frame_index = 0
	if _loading_label != null:
		_loading_label.text = LOADING_TEXT_FRAMES[_frame_index]
	if _tip_label != null:
		_tip_label.text = tip_text
	if _root != null:
		_root.hide()
	if _frame_timer != null:
		_frame_timer.stop()
	if _delay_timer != null:
		_delay_timer.start()

## 收到服务端数据或请求失败时关闭 loading。
func hide_overlay() -> void:
	_is_waiting = false
	if _delay_timer != null:
		_delay_timer.stop()
	if _frame_timer != null:
		_frame_timer.stop()
	if _root != null:
		_root.hide()

## 构建运行时 loading UI，避免每个面板重复写一套遮罩结构。
func _build_ui() -> void:
	_root = Control.new()
	_root.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_root.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(_root)

	var dim_layer: ColorRect = ColorRect.new()
	dim_layer.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	dim_layer.color = Color(0.06, 0.1, 0.16, 0.82)
	_root.add_child(dim_layer)

	var center_box: VBoxContainer = VBoxContainer.new()
	center_box.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
	center_box.position = Vector2(-100.0, -28.0)
	center_box.custom_minimum_size = Vector2(200.0, 56.0)
	center_box.alignment = BoxContainer.ALIGNMENT_CENTER
	_root.add_child(center_box)

	_loading_label = Label.new()
	_loading_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_loading_label.add_theme_font_size_override("font_size", 22)
	_loading_label.text = LOADING_TEXT_FRAMES[0]
	center_box.add_child(_loading_label)

	_tip_label = Label.new()
	_tip_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_tip_label.text = "正在同步服务端数据"
	center_box.add_child(_tip_label)

	_delay_timer = Timer.new()
	_delay_timer.wait_time = LOADING_DISPLAY_DELAY_SEC
	_delay_timer.one_shot = true
	_delay_timer.timeout.connect(_on_delay_timeout)
	add_child(_delay_timer)

	_frame_timer = Timer.new()
	_frame_timer.wait_time = LOADING_FRAME_INTERVAL_SEC
	_frame_timer.timeout.connect(_on_frame_timeout)
	add_child(_frame_timer)

	_root.hide()

## 超过延迟阈值且仍在等待时，才真正展示 loading 遮罩。
func _on_delay_timeout() -> void:
	if not _is_waiting or _root == null:
		return
	_root.show()
	if _frame_timer != null:
		_frame_timer.start()

## 轮播 loading 文案，提升长请求的等待感知。
func _on_frame_timeout() -> void:
	if not _is_waiting or _loading_label == null:
		return
	_frame_index = (_frame_index + 1) % LOADING_TEXT_FRAMES.size()
	_loading_label.text = LOADING_TEXT_FRAMES[_frame_index]
