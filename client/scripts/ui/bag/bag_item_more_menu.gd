extends Control
class_name BagItemMoreMenu

## 玩家在更多菜单里选中某个操作后向外广播动作 key。
signal action_selected(action_key: String)
## 更多菜单关闭时通知详情面板同步状态。
signal menu_closed

const ACTION_KEYS: Array[String] = [
	"drop",
	"give",
	"share",
]
## 菜单底边与「更多」按钮顶边之间的间距（像素）。
const ANCHOR_GAP_Y: float = 4.0
## 相对默认对齐位置的额外右移偏移（像素）。
const POSITION_OFFSET_X: float = 0.0
## 相对默认对齐位置的额外下移偏移（像素）。
const POSITION_OFFSET_Y: float = 0.0

## 内容根节点，用于读取场景里固定的菜单尺寸。
@onready var _content_root: MarginContainer = $MarginContainer
## 丢弃按钮。
@onready var _drop_button: RuntimeActionButton = %DropButton
## 给人按钮。
@onready var _give_button: RuntimeActionButton = %GiveButton
## 分享按钮。
@onready var _share_button: RuntimeActionButton = %ShareButton

## 动作按钮索引，key 与协议动作一致。
var _action_buttons: Dictionary = {}
## 更多菜单是否处于打开状态；不依赖 visible，避免 top_level 节点状态不同步。
var _more_menu_tracking_open: bool = false


## 绑定按钮信号；打开时使用 top_level 浮层定位，避免 PanelContainer 参与布局。
func _ready() -> void:
	z_index = 16
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	_action_buttons = {
		"drop": _drop_button,
		"give": _give_button,
		"share": _share_button,
	}
	for action_key: String in ACTION_KEYS:
		var button: RuntimeActionButton = _action_buttons.get(action_key, null) as RuntimeActionButton
		if button == null:
			continue
		if not button.pressed.is_connected(_on_action_button_pressed.bind(action_key)):
			button.pressed.connect(_on_action_button_pressed.bind(action_key))
	hide()


## 更多菜单当前是否处于展开态。
func is_open() -> bool:
	return _more_menu_tracking_open


## 在「更多」按钮正上方打开菜单；只改位置，不改场景里预设的按钮尺寸。
func open_near(anchor: Control) -> void:
	if anchor == null:
		return
	_refresh_button_states()
	_more_menu_tracking_open = true
	top_level = true
	show()
	if get_parent() != null:
		get_parent().move_child(self, get_parent().get_child_count() - 1)
	call_deferred("_apply_position_near", anchor)


## 根据锚点按钮的全局矩形计算并应用菜单位置。
func _apply_position_near(anchor: Control) -> void:
	if anchor == null or not is_inside_tree():
		return
	var menu_size: Vector2 = _content_root.size
	if menu_size.x <= 0.0 or menu_size.y <= 0.0:
		menu_size = _content_root.get_combined_minimum_size()
	if menu_size.x <= 0.0 or menu_size.y <= 0.0:
		menu_size = size
	var anchor_rect: Rect2 = anchor.get_global_rect()
	var global_x: float = anchor_rect.position.x + (anchor_rect.size.x - menu_size.x) * 0.5 + POSITION_OFFSET_X
	var global_y: float = anchor_rect.position.y - menu_size.y - ANCHOR_GAP_Y + POSITION_OFFSET_Y
	var viewport_size: Vector2 = get_viewport().get_visible_rect().size
	global_x = clampf(global_x, 0.0, maxf(viewport_size.x - menu_size.x, 0.0))
	global_y = clampf(global_y, 0.0, maxf(viewport_size.y - menu_size.y, 0.0))
	size = menu_size
	global_position = Vector2(global_x, global_y)


## 关闭更多菜单。
func hide_menu() -> void:
	if not _more_menu_tracking_open:
		return
	_more_menu_tracking_open = false
	top_level = false
	hide()
	menu_closed.emit()


## 判断全局坐标是否落在三个动作按钮的可点击区域内。
func is_global_point_over_action_buttons(global_point: Vector2) -> bool:
	for action_key: String in ACTION_KEYS:
		var button: RuntimeActionButton = _action_buttons.get(action_key, null) as RuntimeActionButton
		if button == null or not button.visible:
			continue
		if button.get_global_rect().has_point(global_point):
			return true
	return false


## 刷新三个动作按钮的启用态。
func _refresh_button_states() -> void:
	for action_key: String in ACTION_KEYS:
		var button: RuntimeActionButton = _action_buttons.get(action_key, null) as RuntimeActionButton
		if button == null:
			continue
		button.disabled = false


## 转发菜单内按钮点击并在触发后关闭弹窗。
func _on_action_button_pressed(action_key: String) -> void:
	hide_menu()
	action_selected.emit(action_key)
