class_name AnchoredPopupBase
extends Control

## 锚点浮层关闭时通知父级同步交互状态。
signal popup_closed

## 面板出现在锚点上方（水平居中）。
const PLACEMENT_ABOVE: String = "above"
## 面板出现在锚点右侧（顶部对齐）。
const PLACEMENT_RIGHT: String = "right"
## 面板与锚点之间的默认间距（像素）。
const DEFAULT_ANCHOR_GAP: float = 6.0

## 浮层是否处于打开状态；不依赖 visible，避免 top_level 节点状态不同步。
var _popup_open: bool = false
## 打开时的锚点控件，布局刷新后用于重新定位。
var _anchor_control: Control = null
## 本次打开时的面板相对锚点方位。
var _active_placement: String = PLACEMENT_ABOVE
## 本次打开时面板与锚点之间的间距（像素）。
var _active_anchor_gap: float = DEFAULT_ANCHOR_GAP
## 本次打开时相对默认定位的额外 X 偏移（像素）。
var _active_position_offset_x: float = 0.0
## 本次打开时相对默认定位的额外 Y 偏移（像素）。
var _active_position_offset_y: float = 0.0


## 初始化时忽略鼠标，避免遮挡底层控件点击。
func _ready() -> void:
    mouse_filter = Control.MOUSE_FILTER_IGNORE
    hide()


## 浮层当前是否展开。
func is_open() -> bool:
    return _popup_open


## 在锚点控件附近打开浮层；子类可覆写 _prepare_open 注入额外配置。
func open_near(anchor: Control, config: Dictionary = {}) -> void:
    if anchor == null:
        return
    _prepare_open(config)
    _start_anchored_open(anchor)
    call_deferred("_deferred_position_near")


## 关闭锚点浮层。
func close_popup() -> void:
    _finish_anchored_close()


## 子类覆写：返回用于测量尺寸的定位根节点。
func _get_layout_root() -> Control:
    return self


## 子类覆写：在浮层展示前解析 config 并刷新内容。
func _prepare_open(config: Dictionary) -> void:
    _apply_anchor_config(config)


## 解析锚点定位相关 config 键：placement、anchor_gap、position_offset_x、position_offset_y。
func _apply_anchor_config(config: Dictionary) -> void:
    _active_placement = str(config.get("placement", PLACEMENT_ABOVE))
    if _active_placement != PLACEMENT_RIGHT:
        _active_placement = PLACEMENT_ABOVE
    _active_anchor_gap = float(config.get("anchor_gap", DEFAULT_ANCHOR_GAP))
    if _active_anchor_gap < 0.0:
        _active_anchor_gap = DEFAULT_ANCHOR_GAP
    _active_position_offset_x = float(config.get("position_offset_x", 0.0))
    _active_position_offset_y = float(config.get("position_offset_y", 0.0))


## 进入 top_level 浮层态并置于父节点最上层。
func _start_anchored_open(anchor: Control) -> void:
    _anchor_control = anchor
    _popup_open = true
    top_level = true
    show()
    if get_parent() != null:
        get_parent().move_child(self, get_parent().get_child_count() - 1)


## 退出 top_level 浮层态并广播关闭信号。
func _finish_anchored_close() -> void:
    if not _popup_open:
        return
    _popup_open = false
    _anchor_control = null
    top_level = false
    hide()
    popup_closed.emit()


## 延迟到布局就绪后再定位，避免首次打开尺寸为 0。
func _deferred_position_near() -> void:
    if _anchor_control != null:
        _apply_position_near(_anchor_control)


## 根据锚点与布局根节点尺寸计算全局坐标，并限制在视口内。
func _apply_position_near(anchor: Control) -> void:
    if anchor == null or not is_inside_tree():
        return
    var layout_root: Control = _get_layout_root()
    if layout_root == null:
        return
    var panel_size: Vector2 = _resolve_control_size(layout_root)
    var anchor_rect: Rect2 = anchor.get_global_rect()
    var viewport_size: Vector2 = get_viewport().get_visible_rect().size
    var global_x: float = 0.0
    var global_y: float = 0.0
    if _active_placement == PLACEMENT_RIGHT:
        global_x = anchor_rect.position.x + anchor_rect.size.x + _active_anchor_gap
        global_y = anchor_rect.position.y
        if global_x + panel_size.x > viewport_size.x:
            global_x = anchor_rect.position.x - panel_size.x - _active_anchor_gap
    else:
        global_x = anchor_rect.position.x + (anchor_rect.size.x - panel_size.x) * 0.5
        global_y = anchor_rect.position.y - panel_size.y - _active_anchor_gap
    global_x += _active_position_offset_x
    global_y += _active_position_offset_y
    global_x = clampf(global_x, 0.0, maxf(viewport_size.x - panel_size.x, 0.0))
    global_y = clampf(global_y, 0.0, maxf(viewport_size.y - panel_size.y, 0.0))
    size = panel_size
    global_position = Vector2(global_x, global_y)


## 读取控件当前可用于定位的尺寸。
func _resolve_control_size(control: Control) -> Vector2:
    var panel_size: Vector2 = control.size
    if panel_size.x <= 0.0 or panel_size.y <= 0.0:
        panel_size = control.get_combined_minimum_size()
    if panel_size.x <= 0.0 or panel_size.y <= 0.0:
        panel_size = size
    return panel_size
