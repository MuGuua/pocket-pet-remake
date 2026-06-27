class_name RuntimeProgressOverlay
extends CanvasLayer

## 通用进度遮罩场景路径，供其他 UI 懒加载实例化。
const SCENE_PATH: String = "res://scenes/ui/common/runtime_progress_overlay.tscn"
## 默认进度动画时长（秒）。
const DEFAULT_DURATION_SEC: float = 3.0
## 未传入文案时的默认提示。
const DEFAULT_STATUS_TEXT: String = "加载中..."

@onready var _root: Control = $Root
@onready var _status_label: Label = %StatusLabel
@onready var _progress_bar: ProgressBar = %ProgressBar

## 当前进度动画 tween，便于外部取消时停止。
var _progress_tween: Tween = null


## 初始化时保持隐藏，避免编辑器或其他场景预览误显示。
func _ready() -> void:
    visible = false
    if _progress_bar != null:
        _progress_bar.min_value = 0.0
        _progress_bar.max_value = 100.0
        _progress_bar.value = 0.0


## 播放线性进度动画；duration_sec 为填满进度条所需秒数，status_text 为居中提示文案。
func play_progress(duration_sec: float = DEFAULT_DURATION_SEC, status_text: String = DEFAULT_STATUS_TEXT) -> void:
    if not is_node_ready():
        await ready
    _stop_progress_tween()
    visible = true
    if _root != null:
        _root.show()
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


## 立即隐藏进度层并停止动画。
func hide_progress() -> void:
    _stop_progress_tween()
    if _root != null:
        _root.hide()
    visible = false


## 停止进行中的进度 tween，避免取消后仍回调。
func _stop_progress_tween() -> void:
    if _progress_tween != null and _progress_tween.is_valid():
        _progress_tween.kill()
    _progress_tween = null
