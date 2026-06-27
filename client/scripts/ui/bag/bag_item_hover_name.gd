extends Control
class_name BagItemHoverName

## 名称标签底边与锚点控件顶边之间的垂直间距（像素）。
const ANCHOR_GAP_Y: float = 2.0

## 悬停名称标签；右对齐显示，便于固定在物品右上方。
@onready var _name_label: Label = %NameLabel


## 初始化时忽略鼠标，避免遮挡格子点击。
func _ready() -> void:
    mouse_filter = Control.MOUSE_FILTER_IGNORE
    hide()


## 在锚点控件右上方显示物品名称；位置相对锚点固定，不跟随鼠标。
func show_for_anchor(anchor: Control, item_name: String) -> void:
    if anchor == null:
        hide_name()
        return
    var normalized_name: String = item_name.strip_edges()
    if normalized_name.is_empty():
        hide_name()
        return
    _name_label.text = normalized_name
    top_level = true
    z_index = 32
    show()
    call_deferred("_apply_position", anchor)


## 隐藏悬停名称并恢复为普通子节点，避免残留 top_level 浮层。
func hide_name() -> void:
    top_level = false
    hide()


## 将标签右缘对齐锚点右缘，并放在锚点顶边上方。
func _apply_position(anchor: Control) -> void:
    if anchor == null or not is_inside_tree() or _name_label == null:
        return
    var label_size: Vector2 = _name_label.get_minimum_size()
    if label_size.x <= 0.0 or label_size.y <= 0.0:
        label_size = _name_label.size
    size = label_size
    _name_label.size = label_size
    _name_label.position = Vector2.ZERO
    var anchor_rect: Rect2 = anchor.get_global_rect()
    var global_x: float = anchor_rect.position.x + anchor_rect.size.x - label_size.x
    var global_y: float = anchor_rect.position.y - label_size.y - ANCHOR_GAP_Y
    var viewport_size: Vector2 = get_viewport().get_visible_rect().size
    global_x = clampf(global_x, 0.0, maxf(viewport_size.x - label_size.x, 0.0))
    global_y = clampf(global_y, 0.0, maxf(viewport_size.y - label_size.y, 0.0))
    global_position = Vector2(global_x, global_y)
