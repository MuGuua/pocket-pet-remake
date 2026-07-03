extends Button
class_name RuntimeActionButton

## 按钮中心展示文案；实例化场景时在检查器里改这个字段即可。
@export var button_label: String = "按钮"
## 鼠标悬停时的缩放比例。
@export var hover_scale: Vector2 = Vector2(1.1, 1.1)
## 按下瞬间的缩放比例。
@export var pressed_scale: Vector2 = Vector2(0.9, 0.9)

## 当前正在播放的缩放动画，避免快速移入移出时 tween 互相打架。
var _scale_tween: Tween = null


## 启动时同步文案、绑定悬停/按下动画，并延迟初始化缩放中心点。
func _ready() -> void:
    _sync_button_label()
    if not mouse_entered.is_connected(_button_enter):
        mouse_entered.connect(_button_enter)
    if not mouse_exited.is_connected(_button_exit):
        mouse_exited.connect(_button_exit)
    if not pressed.is_connected(_button_pressed):
        pressed.connect(_button_pressed)
    call_deferred("_init_pivot")


## 运行时更新按钮中心文案。
func set_button_label(label: String) -> void:
    button_label = label
    _sync_button_label()
    call_deferred("_init_pivot")


## 将 button_label 写入 Button.text，供布局与无障碍读取。
func _sync_button_label() -> void:
    text = button_label


## 将缩放中心设为按钮几何中心，保证放大/缩小视觉对称。
func _init_pivot() -> void:
    pivot_offset = size * 0.5


## 鼠标移入时放大到 hover_scale。
func _button_enter() -> void:
    _play_scale_tween(hover_scale, 0.1)


## 鼠标移出时恢复到原始大小。
func _button_exit() -> void:
    _play_scale_tween(Vector2.ONE, 0.1)


## 点击时在 pressed_scale 与 hover_scale 之间做一次短促缩放反馈。
func _button_pressed() -> void:
    if _scale_tween != null and _scale_tween.is_valid():
        _scale_tween.kill()
    var button_press_tween: Tween = create_tween()
    _scale_tween = button_press_tween
    button_press_tween.tween_property(self, "scale", pressed_scale, 0.06).set_trans(Tween.TRANS_SINE)
    button_press_tween.tween_property(self, "scale", hover_scale, 0.12).set_trans(Tween.TRANS_SINE)


## 播放单段缩放 tween，并在开始前终止上一次动画。
func _play_scale_tween(target_scale: Vector2, duration: float) -> void:
    if _scale_tween != null and _scale_tween.is_valid():
        _scale_tween.kill()
    _scale_tween = create_tween()
    _scale_tween.tween_property(self, "scale", target_scale, duration).set_trans(Tween.TRANS_SINE)
