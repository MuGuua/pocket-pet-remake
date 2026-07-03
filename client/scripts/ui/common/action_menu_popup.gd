class_name ActionMenuPopup
extends AnchoredPopupBase

## 通用锚点动作菜单场景路径；可用于背包「更多」、上下文操作等流程。
const SCENE_PATH: String = "res://scenes/ui/common/action_menu_popup.tscn"
## 通用动作按钮场景。
const ACTION_BUTTON_SCENE: PackedScene = preload("res://scenes/ui/common/runtime_action_button.tscn")
## 背包物品详情「更多」菜单默认动作列表。
const BAG_ITEM_ACTIONS: Array[Dictionary] = [
    {"key": "drop", "label": "丢弃"},
    {"key": "give", "label": "给人"},
    {"key": "share", "label": "分享"},
]
## 动作菜单默认与锚点之间的垂直间距（像素）。
const DEFAULT_MENU_ANCHOR_GAP: float = 4.0

## 玩家选中某个动作后向外广播动作 key。
signal action_selected(action_key: String)
## 兼容旧调用方的关闭信号别名。
signal menu_closed

## 内容根节点，用于读取菜单整体尺寸。
@onready var _content_root: MarginContainer = $MarginContainer
## 动态动作按钮容器。
@onready var _action_vbox: VBoxContainer = %ActionVBox
## 右上角关闭按钮。
@onready var _top_close_button: BaseButton = %TopCloseButton

## 当前动作定义列表；每项至少包含 key 与 label。
var _action_defs: Array[Dictionary] = []
## 动作 key 到按钮节点的映射。
var _action_buttons: Dictionary = {}


## 绑定默认样式，并在关闭时同步 menu_closed 信号。
func _ready() -> void:
    super._ready()
    z_index = 16
    if _content_root != null:
        _content_root.mouse_filter = Control.MOUSE_FILTER_STOP
    if _top_close_button != null and not _top_close_button.pressed.is_connected(_on_top_close_button_pressed):
        _top_close_button.pressed.connect(_on_top_close_button_pressed)
    if not popup_closed.is_connected(_on_popup_closed):
        popup_closed.connect(_on_popup_closed)


## 配置动作列表；每项支持 key、label、visible、disabled。
func configure_actions(actions: Array) -> void:
    _action_defs.clear()
    for action_variant: Variant in actions:
        if action_variant is not Dictionary:
            continue
        var action: Dictionary = action_variant as Dictionary
        var action_key: String = str(action.get("key", "")).strip_edges()
        if action_key.is_empty():
            continue
        _action_defs.append(action.duplicate(true))
    _rebuild_action_buttons()


## 在锚点上方打开动作菜单。
func open_near(anchor: Control, config: Dictionary = {}) -> void:
    if anchor == null:
        return
    var resolved_config: Dictionary = config.duplicate(true)
    if not resolved_config.has("anchor_gap"):
        resolved_config["anchor_gap"] = DEFAULT_MENU_ANCHOR_GAP
    _refresh_button_states()
    super.open_near(anchor, resolved_config)


## 关闭动作菜单；兼容旧 hide_menu 调用。
func hide_menu() -> void:
    close_popup()


## 判断全局坐标是否落在可见动作按钮区域内。
func is_global_point_over_action_buttons(global_point: Vector2) -> bool:
    for action_key: String in _action_buttons.keys():
        var button: RuntimeActionButton = _action_buttons.get(action_key, null) as RuntimeActionButton
        if button == null or not button.visible:
            continue
        if button.get_global_rect().has_point(global_point):
            return true
    return false


## 判断全局坐标是否落在整个动作菜单区域内，避免点击菜单空白时被误判为“点外部”。
func is_global_point_over_menu(global_point: Vector2) -> bool:
    var layout_root: Control = _get_layout_root()
    if layout_root == null or not is_open():
        return false
    return layout_root.get_global_rect().has_point(global_point)


## 返回用于定位的菜单根节点。
func _get_layout_root() -> Control:
    return _content_root


## 清空并重建动作按钮。
func _rebuild_action_buttons() -> void:
    if _action_vbox == null:
        return
    for child: Node in _action_vbox.get_children():
        child.queue_free()
    _action_buttons.clear()
    for action: Dictionary in _action_defs:
        var action_key: String = str(action.get("key", ""))
        var action_label: String = str(action.get("label", action_key))
        var button: RuntimeActionButton = ACTION_BUTTON_SCENE.instantiate() as RuntimeActionButton
        if button == null:
            continue
        button.set_button_label(action_label)
        button.visible = bool(action.get("visible", true))
        button.disabled = bool(action.get("disabled", false))
        if not button.pressed.is_connected(_on_action_button_pressed.bind(action_key)):
            button.pressed.connect(_on_action_button_pressed.bind(action_key))
        _action_vbox.add_child(button)
        _action_buttons[action_key] = button


## 打开菜单前刷新按钮启用态。
func _refresh_button_states() -> void:
    for action: Dictionary in _action_defs:
        var action_key: String = str(action.get("key", ""))
        var button: RuntimeActionButton = _action_buttons.get(action_key, null) as RuntimeActionButton
        if button == null:
            continue
        button.visible = bool(action.get("visible", true))
        button.disabled = bool(action.get("disabled", false))


## 转发动作按钮点击并在触发后关闭菜单。
func _on_action_button_pressed(action_key: String) -> void:
    close_popup()
    action_selected.emit(action_key)


## 响应右上角关闭按钮。
func _on_top_close_button_pressed() -> void:
    close_popup()


## 同步 menu_closed 信号，兼容旧监听方。
func _on_popup_closed() -> void:
    menu_closed.emit()
