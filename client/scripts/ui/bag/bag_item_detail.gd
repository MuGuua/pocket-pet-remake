extends PanelContainer
class_name BagItemDetail

signal action_requested(action_key: String, item: Dictionary)

const ACTIONS: Array[Dictionary] = [
    {"key": "open", "label": "打开"},
    {"key": "use", "label": "使用"},
    {"key": "drop", "label": "丢弃"},
    {"key": "give", "label": "给人"},
    {"key": "share", "label": "分享到聊天"},
]

## 当前选中的服务端物品快照。
var _item: Dictionary = {}
## 名称标签。
var _name_label: Label = null
## 类型标签。
var _type_label: Label = null
## 数量标签。
var _quantity_label: Label = null
## 描述标签。
var _description_label: Label = null
## 操作按钮索引。
var _action_buttons: Dictionary = {}


## 构建详情面板。
func _ready() -> void:
    _build_children()
    clear_item()


## 展示选中的物品详情。
func set_item(item: Dictionary) -> void:
    _item = item.duplicate(true)
    if _name_label != null:
        _name_label.text = BagUiMapper.item_name(_item)
    if _type_label != null:
        _type_label.text = BagUiMapper.item_type_text(_item)
    if _quantity_label != null:
        _quantity_label.text = "数量：%s" % UiFormat.value_to_text(BagUiMapper.quantity(_item))
        _quantity_label.visible = BagUiMapper.is_stackable(_item)
    if _description_label != null:
        _description_label.text = BagUiMapper.description(_item)
    _refresh_actions()


## 清空详情面板。
func clear_item() -> void:
    _item.clear()
    if _name_label != null:
        _name_label.text = "未选择物品"
    if _type_label != null:
        _type_label.text = "类型：-"
    if _quantity_label != null:
        _quantity_label.text = "数量：-"
        _quantity_label.hide()
    if _description_label != null:
        _description_label.text = "点击左侧物品查看详情。"
    _refresh_actions()


## 构建详情文本和操作按钮。
func _build_children() -> void:
    custom_minimum_size = Vector2(0, 150)
    var margin: MarginContainer = MarginContainer.new()
    margin.add_theme_constant_override("margin_left", 10)
    margin.add_theme_constant_override("margin_top", 8)
    margin.add_theme_constant_override("margin_right", 10)
    margin.add_theme_constant_override("margin_bottom", 8)
    add_child(margin)

    var root: VBoxContainer = VBoxContainer.new()
    root.add_theme_constant_override("separation", 5)
    margin.add_child(root)

    _name_label = Label.new()
    _name_label.add_theme_font_size_override("font_size", 15)
    root.add_child(_name_label)

    _type_label = Label.new()
    _type_label.add_theme_font_size_override("font_size", 11)
    root.add_child(_type_label)

    _quantity_label = Label.new()
    _quantity_label.add_theme_font_size_override("font_size", 11)
    root.add_child(_quantity_label)

    _description_label = Label.new()
    _description_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
    _description_label.custom_minimum_size = Vector2(0, 40)
    root.add_child(_description_label)

    var action_row: HBoxContainer = HBoxContainer.new()
    action_row.add_theme_constant_override("separation", 4)
    root.add_child(action_row)

    for action: Dictionary in ACTIONS:
        var button: Button = Button.new()
        var action_key: String = str(action.get("key", ""))
        button.text = str(action.get("label", ""))
        button.custom_minimum_size = Vector2(48, 28)
        button.pressed.connect(_on_action_pressed.bind(action_key))
        action_row.add_child(button)
        _action_buttons[action_key] = button


## 刷新按钮启用态；未实现功能仍显示但禁用。
func _refresh_actions() -> void:
    for action: Dictionary in ACTIONS:
        var action_key: String = str(action.get("key", ""))
        var button: Button = _action_buttons.get(action_key, null) as Button
        if button == null:
            continue
        button.disabled = _item.is_empty() or not _is_action_enabled(action_key)


## 当前版本仅使用已有 use 协议，其余按钮作为占位入口展示提示。
func _is_action_enabled(action_key: String) -> bool:
    if action_key == "use":
        return BagUiMapper.has_action(_item, "use")
    return true


## 转发玩家点击的功能按钮。
func _on_action_pressed(action_key: String) -> void:
    if _item.is_empty():
        return
    action_requested.emit(action_key, _item.duplicate(true))
