class_name PlotImage
extends CanvasLayer

## 默认图片闪烁间隔；只切换图片可见性，不修改场景中配置的位置、尺寸或缩放。
const DEFAULT_BLINK_INTERVAL_SECONDS: float = 0.35

## 需要闪烁的图片节点；节点尺寸和位置由 plot_image.tscn 维护。
@onready var _image_rect: CanvasItem = get_node_or_null("Root/TextureRect") as CanvasItem

## 当前图片闪烁循环 Tween；关闭场景时需要主动停止。
var _blink_tween: Tween = null


## 初始化时保持隐藏，避免运行时 UI 场景在未打开前常驻显示。
func _ready() -> void:
    hide_plot_image()


## 场景退出时停止闪烁循环，避免 Tween 回调访问已释放节点。
func _exit_tree() -> void:
    _stop_blink_loop()


## 显示剧情图片并开始闪烁；仅控制 visible，不改变图片大小和位置。
func show_plot_image() -> void:
    visible = true
    if _image_rect == null:
        return
    _image_rect.visible = true
    _start_blink_loop()


## 隐藏剧情图片并停止闪烁，供剧情结束或被打断时调用。
func hide_plot_image() -> void:
    _stop_blink_loop()
    if _image_rect != null:
        _image_rect.visible = false
    visible = false


## 启动循环 Tween，按固定间隔切换图片显示状态。
func _start_blink_loop() -> void:
    _stop_blink_loop()
    _blink_tween = create_tween()
    _blink_tween.set_loops()
    _blink_tween.tween_interval(DEFAULT_BLINK_INTERVAL_SECONDS)
    _blink_tween.tween_callback(Callable(self, "_toggle_image_visible"))


## 停止当前闪烁循环。
func _stop_blink_loop() -> void:
    if _blink_tween != null and _blink_tween.is_valid():
        _blink_tween.kill()
    _blink_tween = null


## 切换图片显示状态；图片节点缺失时直接跳过，避免剧情报错中断。
func _toggle_image_visible() -> void:
    if _image_rect == null:
        return
    _image_rect.visible = not _image_rect.visible
