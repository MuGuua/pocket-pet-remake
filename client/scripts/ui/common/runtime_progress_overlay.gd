class_name RuntimeProgressOverlay
extends CanvasLayer

## 通用 Loading 遮罩场景路径；等待回包与进度动画两种模式复用同一资源。
const SCENE_PATH: String = "res://scenes/ui/common/runtime_progress_overlay.tscn"
## 默认进度动画时长（秒）。
const DEFAULT_DURATION_SEC: float = 3.0
## 未传入文案时的默认提示。
const DEFAULT_STATUS_TEXT: String = "加载中..."
## 等待模式：超过该秒数后才展示遮罩，避免短请求闪屏。
const WAITING_DISPLAY_DELAY_SEC: float = 1.0
## 等待模式：主文案轮播间隔（秒）。
const WAITING_FRAME_INTERVAL_SEC: float = 0.2
## 等待模式：主文案轮播帧。
const WAITING_TEXT_FRAMES: Array[String] = ["加载中   ", "加载中.  ", "加载中.. ", "加载中..."]

## 全屏遮罩根节点。
@onready var _root: Control = $Root
## 等待模式内容区。
@onready var _waiting_panel: VBoxContainer = %WaitingPanel
## 等待模式主文案。
@onready var _waiting_main_label: Label = %WaitingMainLabel
## 等待模式次级提示文案。
@onready var _waiting_tip_label: Label = %WaitingTipLabel
## 进度模式内容区。
@onready var _progress_panel: PanelContainer = %ProgressPanel
## 进度模式状态文案。
@onready var _status_label: Label = %StatusLabel
## 进度模式进度条。
@onready var _progress_bar: ProgressBar = %ProgressBar

## 当前进度动画 tween，便于外部取消时停止。
var _progress_tween: Tween = null
## 等待模式延迟展示计时器。
var _delay_timer: Timer = null
## 等待模式文案轮播计时器。
var _frame_timer: Timer = null
## 等待模式轮播帧索引。
var _waiting_frame_index: int = 0
## 是否仍处于等待服务端回包状态。
var _is_waiting: bool = false


## 初始化时保持隐藏，并准备进度条与计时器。
func _ready() -> void:
    visible = false
    if _root != null:
        _root.hide()
    if _progress_bar != null:
        _progress_bar.min_value = 0.0
        _progress_bar.max_value = 100.0
        _progress_bar.value = 0.0
    _ensure_timers()
    _hide_all_mode_panels()


## 开始等待服务端回包；tip_text 用来说明当前请求用途。
func show_waiting(tip_text: String) -> void:
    _stop_progress_tween()
    _is_waiting = true
    _waiting_frame_index = 0
    if _waiting_main_label != null:
        _waiting_main_label.text = WAITING_TEXT_FRAMES[_waiting_frame_index]
    if _waiting_tip_label != null:
        _waiting_tip_label.text = tip_text
    _hide_all_mode_panels()
    if _root != null:
        _root.hide()
    visible = true
    if _frame_timer != null:
        _frame_timer.stop()
    if _delay_timer != null:
        _delay_timer.start()


## 播放线性进度动画；duration_sec 为填满进度条所需秒数，status_text 为居中提示文案。
func play_progress(duration_sec: float = DEFAULT_DURATION_SEC, status_text: String = DEFAULT_STATUS_TEXT) -> void:
    if not is_node_ready():
        await ready
    _stop_waiting_timers()
    _is_waiting = false
    _stop_progress_tween()
    visible = true
    if _root != null:
        _root.show()
    _hide_all_mode_panels()
    if _progress_panel != null:
        _progress_panel.show()
    if _status_label != null:
        _status_label.text = status_text
    var progress_bar: ProgressBar = _progress_bar
    if progress_bar == null:
        progress_bar = %ProgressBar
    if progress_bar == null:
        push_warning("RuntimeProgressOverlay: ProgressBar 未就绪，跳过进度动画。")
        return
    progress_bar.min_value = 0.0
    progress_bar.max_value = 100.0
    progress_bar.value = 0.0
    if duration_sec <= 0.0:
        progress_bar.value = 100.0
        return
    _progress_tween = create_tween()
    _progress_tween.tween_property(progress_bar, "value", 100.0, duration_sec).set_trans(Tween.TRANS_LINEAR)
    await _progress_tween.finished
    _progress_tween = null


## 关闭等待模式遮罩；兼容旧 RequestLoadingOverlay 调用。
func hide_overlay() -> void:
    hide_loading()


## 关闭进度模式遮罩；兼容旧 RuntimeProgressOverlay 调用。
func hide_progress() -> void:
    hide_loading()


## 立即关闭 Loading 遮罩并停止所有动画/计时器。
func hide_loading() -> void:
    _is_waiting = false
    _stop_waiting_timers()
    _stop_progress_tween()
    if _root != null:
        _root.hide()
    _hide_all_mode_panels()
    visible = false


## 懒创建等待模式所需计时器。
func _ensure_timers() -> void:
    if _delay_timer == null:
        _delay_timer = Timer.new()
        _delay_timer.name = "WaitingDelayTimer"
        _delay_timer.wait_time = WAITING_DISPLAY_DELAY_SEC
        _delay_timer.one_shot = true
        _delay_timer.timeout.connect(_on_waiting_delay_timeout)
        add_child(_delay_timer)
    if _frame_timer == null:
        _frame_timer = Timer.new()
        _frame_timer.name = "WaitingFrameTimer"
        _frame_timer.wait_time = WAITING_FRAME_INTERVAL_SEC
        _frame_timer.timeout.connect(_on_waiting_frame_timeout)
        add_child(_frame_timer)


## 超过延迟阈值且仍在等待时，才真正展示等待遮罩。
func _on_waiting_delay_timeout() -> void:
    if not _is_waiting or _root == null:
        return
    _hide_all_mode_panels()
    if _waiting_panel != null:
        _waiting_panel.show()
    _root.show()
    if _frame_timer != null:
        _frame_timer.start()


## 轮播等待模式主文案，提升长请求等待感知。
func _on_waiting_frame_timeout() -> void:
    if not _is_waiting or _waiting_main_label == null:
        return
    _waiting_frame_index = (_waiting_frame_index + 1) % WAITING_TEXT_FRAMES.size()
    _waiting_main_label.text = WAITING_TEXT_FRAMES[_waiting_frame_index]


## 停止等待模式计时器。
func _stop_waiting_timers() -> void:
    if _delay_timer != null:
        _delay_timer.stop()
    if _frame_timer != null:
        _frame_timer.stop()


## 停止进行中的进度 tween，避免取消后仍回调。
func _stop_progress_tween() -> void:
    if _progress_tween != null and _progress_tween.is_valid():
        _progress_tween.kill()
    _progress_tween = null


## 隐藏两种模式的内容区，避免切换时残留旧布局。
func _hide_all_mode_panels() -> void:
    if _waiting_panel != null:
        _waiting_panel.hide()
    if _progress_panel != null:
        _progress_panel.hide()
