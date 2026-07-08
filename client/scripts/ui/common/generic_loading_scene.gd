class_name GenericLoadingScene
extends CanvasLayer

## 通用全屏 Loading 场景路径；可在编辑器中直接打开本场景调整布局与样式。
const SCENE_PATH: String = "res://scenes/ui/common/generic_loading_scene.tscn"
## 默认主文案（已不在 UI 展示，仅保留接口兼容）。
const DEFAULT_MAIN_TEXT: String = ""
## 顶部 HBox 从右向左滚动速度（像素/秒）。
const SCROLL_SPEED_PX_PER_SEC: float = 150.0
## 精灵默认动画名；与场景中 AnimatedSprite2D 的 SpriteFrames 一致。
const LOADING_SPRITE_ANIMATION: StringName = &"default"
## 精灵帧原始尺寸（与 SpriteFrames  atlas 切片一致）。
const SPRITE_FRAME_SIZE: Vector2 = Vector2(26.0, 42.0)
## 等待服务端回包时，超过该秒数才展示遮罩，避免短请求闪屏。
const WAITING_DISPLAY_DELAY_SEC: float = 1.0
## 固定时长进度动画默认时长（秒）。
const DEFAULT_PROGRESS_DURATION_SEC: float = 3.0

## 全屏根节点，负责拦截点击。
@onready var _root: Control = $Root
## 顶部滚动裁剪区域。
@onready var _scroll_clip: Control = %ScrollClip
## 从右向左滚动的内容容器。
@onready var _scroll_hbox: HBoxContainer = %ScrollHBox
## 精灵帧动画节点。
@onready var _loading_sprite: AnimatedSprite2D = %LoadingSprite
## 尾部状态文案（默认省略号）。
@onready var _status_label: Label = %StatusLabel
## 「读取中」图片字节点。
@onready var _loading_text_rect: TextureRect = %LoadingTextRect

## 当前是否处于展示状态。
var _is_visible: bool = false
## 滚动视口宽度（像素）。
var _scroll_viewport_width: float = 0.0
## 滚动内容总宽度（像素）。
var _scroll_content_width: float = 0.0
## 是否已完成滚动尺寸测量。
var _scroll_metrics_ready: bool = false
## 等待模式延迟展示计时器。
var _delay_timer: Timer = null
## 是否仍处于等待服务端回包、尚未展示遮罩的状态。
var _waiting_for_display: bool = false
## 进行中的进度等待计时器。
var _progress_wait_timer: SceneTreeTimer = null


## 初始化时保持隐藏；单独「运行当前场景」时自动进入预览。
func _ready() -> void:
	visible = false
	if _root != null:
		_root.hide()
	set_process(false)
	if _scroll_clip != null and not _scroll_clip.resized.is_connected(_on_scroll_clip_resized):
		_scroll_clip.resized.connect(_on_scroll_clip_resized)
	_ensure_delay_timer()
	_apply_no_status_text()
	if _is_standalone_preview():
		call_deferred("_start_standalone_preview")


## 判断是否为编辑器/F6 单独运行本场景，便于调试动画。
func _is_standalone_preview() -> bool:
	var tree: SceneTree = get_tree()
	if tree == null:
		return false
	return tree.current_scene == self


## 单独运行场景时自动展示 Loading，方便在编辑器里调样式。
func _start_standalone_preview() -> void:
	show_loading()


## 展示全屏 Loading；main_text / tip_text 参数保留兼容，不在 UI 展示说明文案。
func show_loading(_main_text: String = "", _tip_text: String = "") -> void:
	_cancel_waiting_delay()
	_stop_progress_wait()
	_is_visible = true
	_apply_no_status_text()
	visible = true
	if _root != null:
		_root.show()
	_start_sprite_animation()
	_sync_row_layout()
	call_deferred("_refresh_scroll_metrics_after_layout")
	set_process(true)


## 等待服务端回包；短于 WAITING_DISPLAY_DELAY_SEC 的请求不会展示遮罩。
func show_waiting(_tip_text: String = "", skip_delay: bool = false) -> void:
	_cancel_waiting_delay()
	_stop_progress_wait()
	if skip_delay:
		show_loading()
		return
	_waiting_for_display = true
	if _delay_timer != null:
		_delay_timer.start()


## 展示 Loading 并在固定时长后结束；供普通消耗品使用等短进度流程复用。
func play_progress(duration_sec: float = DEFAULT_PROGRESS_DURATION_SEC, _status_text: String = "") -> void:
	if not is_node_ready():
		await ready
	show_loading()
	if duration_sec <= 0.0:
		return
	_stop_progress_wait()
	_progress_wait_timer = get_tree().create_timer(duration_sec)
	await _progress_wait_timer.timeout
	_progress_wait_timer = null


## 关闭等待模式遮罩；兼容旧 RuntimeProgressOverlay 调用。
func hide_overlay() -> void:
	hide_loading()


## 关闭进度模式遮罩；兼容旧 RuntimeProgressOverlay 调用。
func hide_progress() -> void:
	hide_loading()


## 关闭战斗场景旧接口别名。
func hide_waiting() -> void:
	hide_loading()


## 等待布局完成后再测量滚动区域，避免首帧宽度为 0。
func _refresh_scroll_metrics_after_layout() -> void:
	if not _is_visible:
		return
	await get_tree().process_frame
	_sync_row_layout()
	_refresh_scroll_metrics()
	if not _scroll_metrics_ready:
		await get_tree().process_frame
		_sync_row_layout()
		_refresh_scroll_metrics()


## 立即关闭 Loading，并停止滚动与精灵动画。
func hide_loading() -> void:
	_cancel_waiting_delay()
	_stop_progress_wait()
	_is_visible = false
	_scroll_metrics_ready = false
	set_process(false)
	_stop_sprite_animation()
	if _root != null:
		_root.hide()
	visible = false


## 兼容旧接口；通用 Loading 不展示尾部说明文案。
func set_main_text(_text: String) -> void:
	_apply_no_status_text()


## 兼容旧接口；tip_text 不在 UI 展示。
func set_tip_text(_text: String) -> void:
	pass


## 隐藏尾部状态 Label，保证滚动条区域只保留动画与「读取中」图字。
func _apply_no_status_text() -> void:
	if _status_label == null:
		return
	_status_label.text = ""
	_status_label.hide()


## 每帧驱动 HBox 自右向左滚动，离开左侧后从右侧重新进入。
func _process(delta: float) -> void:
	if not _is_visible or _scroll_hbox == null or not _scroll_metrics_ready:
		return
	if _scroll_viewport_width <= 0.0 or _scroll_content_width <= 0.0:
		return

	_scroll_hbox.position.x -= SCROLL_SPEED_PX_PER_SEC * delta
	if _scroll_hbox.position.x <= -_scroll_content_width:
		_scroll_hbox.position.x = _scroll_viewport_width


## 重新测量裁剪区与 HBox 宽度，并把内容起点放到视口右侧。
func _refresh_scroll_metrics() -> void:
	if _scroll_clip == null or _scroll_hbox == null:
		return

	_scroll_viewport_width = _scroll_clip.size.x
	_scroll_content_width = _scroll_hbox.get_combined_minimum_size().x
	if _scroll_content_width <= 0.0:
		_scroll_content_width = _scroll_hbox.size.x
	if _scroll_viewport_width <= 0.0:
		_scroll_metrics_ready = false
		return

	_scroll_hbox.position.x = _scroll_viewport_width
	_scroll_metrics_ready = true


## 按精灵实际显示高度统一 HBox 内各元素高度，并垂直居中对齐。
func _sync_row_layout() -> void:
	if _loading_sprite == null or _scroll_hbox == null:
		return

	var row_height: int = int(ceil(SPRITE_FRAME_SIZE.y * _loading_sprite.scale.y))
	var sprite_slot_width: int = int(ceil(SPRITE_FRAME_SIZE.x * _loading_sprite.scale.x))
	if row_height <= 0:
		row_height = 26
	if sprite_slot_width <= 0:
		sprite_slot_width = 22

	_scroll_hbox.alignment = BoxContainer.ALIGNMENT_CENTER

	var sprite_slot: Control = _loading_sprite.get_parent() as Control
	if sprite_slot != null:
		sprite_slot.custom_minimum_size = Vector2(float(sprite_slot_width), float(row_height))
		_loading_sprite.position = Vector2(float(sprite_slot_width) * 0.5, float(row_height) * 0.5)

	if _loading_text_rect != null:
		var text_width: float = _loading_text_rect.custom_minimum_size.x
		if text_width <= 0.0:
			text_width = 48.0
		_loading_text_rect.custom_minimum_size = Vector2(text_width, float(row_height))
		_loading_text_rect.size_flags_vertical = Control.SIZE_SHRINK_CENTER
		_loading_text_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED

	if _status_label != null:
		_status_label.hide()
		_status_label.custom_minimum_size = Vector2.ZERO

	if _scroll_clip != null:
		_scroll_clip.custom_minimum_size.y = float(row_height)


## 裁剪区尺寸变化时重新测量，适配移动端视口变化。
func _on_scroll_clip_resized() -> void:
	if not _is_visible:
		return
	_sync_row_layout()
	_refresh_scroll_metrics()


## 播放精灵帧动画。
func _start_sprite_animation() -> void:
	if _loading_sprite == null:
		return
	if not _loading_sprite.is_playing():
		_loading_sprite.play(LOADING_SPRITE_ANIMATION)


## 停止精灵帧动画。
func _stop_sprite_animation() -> void:
	if _loading_sprite == null:
		return
	_loading_sprite.stop()


## 懒创建等待模式延迟计时器。
func _ensure_delay_timer() -> void:
	if _delay_timer != null:
		return
	_delay_timer = Timer.new()
	_delay_timer.name = "WaitingDelayTimer"
	_delay_timer.wait_time = WAITING_DISPLAY_DELAY_SEC
	_delay_timer.one_shot = true
	_delay_timer.timeout.connect(_on_waiting_delay_timeout)
	add_child(_delay_timer)


## 超过延迟阈值且仍在等待时，才真正展示 Loading。
func _on_waiting_delay_timeout() -> void:
	if not _waiting_for_display:
		return
	_waiting_for_display = false
	show_loading()


## 取消等待模式延迟展示。
func _cancel_waiting_delay() -> void:
	_waiting_for_display = false
	if _delay_timer != null:
		_delay_timer.stop()


## 停止固定时长进度等待。
func _stop_progress_wait() -> void:
	_progress_wait_timer = null
