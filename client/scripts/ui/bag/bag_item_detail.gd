extends PanelContainer
class_name BagItemDetail

signal action_requested(action_key: String, item: Dictionary)

enum DetailContext {
    BAG_ITEM,
    EQUIPPED_ITEM,
}

const MORE_MENU_ACTION_KEYS: Array[String] = [
    "drop",
    "give",
    "share",
]

## 物品名称标签；布局与样式在 bag_item_detail.tscn 中编辑。
@onready var _name_label: Label = %NameLabel
## 物品类型标签。
@onready var _type_label: Label = %TypeLabel
## 装备部位标签。
@onready var _equip_slot_label: Label = %EquipSlotLabel
## 堆叠数量标签。
@onready var _quantity_label: Label = %QuantityLabel
## 物品描述标签。
@onready var _description_label: Label = %DescriptionLabel
## 左侧主操作按钮：打开 / 使用 / 装备 / 卸下。
@onready var _primary_button: Button = %PrimaryButton
## 右侧次操作按钮：已穿戴显示分享，背包物品显示更多。
@onready var _secondary_button: Button = %SecondaryButton
## 点击「更多」后在主按钮上方展开的次级菜单容器。
@onready var _more_menu_row: HBoxContainer = %MoreMenuRow
## 更多菜单中的丢弃按钮。
@onready var _drop_button: Button = %DropButton
## 更多菜单中的给人按钮。
@onready var _give_button: Button = %GiveButton
## 更多菜单中的分享按钮。
@onready var _share_button: Button = %ShareButton

## 当前详情展示上下文：背包格子物品或已穿戴装备。
var _context: DetailContext = DetailContext.BAG_ITEM
## 当前选中的服务端物品快照。
var _item: Dictionary = {}
## 更多菜单是否处于展开状态。
var _more_menu_open: bool = false
## 更多菜单按钮索引，key 与协议动作一致。
var _more_menu_buttons: Dictionary = {}


## 绑定场景内按钮信号，并初始化空态文案。
func _ready() -> void:
    _more_menu_buttons = {
        "drop": _drop_button,
        "give": _give_button,
        "share": _share_button,
    }
    if _primary_button != null and not _primary_button.pressed.is_connected(_on_primary_pressed):
        _primary_button.pressed.connect(_on_primary_pressed)
    if _secondary_button != null and not _secondary_button.pressed.is_connected(_on_secondary_pressed):
        _secondary_button.pressed.connect(_on_secondary_pressed)
    for action_key: String in MORE_MENU_ACTION_KEYS:
        var button: Button = _more_menu_buttons.get(action_key, null) as Button
        if button == null:
            continue
        if not button.pressed.is_connected(_on_more_menu_action_pressed.bind(action_key)):
            button.pressed.connect(_on_more_menu_action_pressed.bind(action_key))
    clear_item()


## 展示背包格子里的物品详情。
func set_item(item: Dictionary) -> void:
    _context = DetailContext.BAG_ITEM
    _apply_item_snapshot(item)


## 展示人物装备槽上已穿戴的装备详情。
func set_equipped_item(item: Dictionary) -> void:
    _context = DetailContext.EQUIPPED_ITEM
    _apply_item_snapshot(item)


## 写入物品快照并刷新详情文案与操作按钮。
func _apply_item_snapshot(item: Dictionary) -> void:
    _item = item.duplicate(true)
    _more_menu_open = false
    if _name_label != null:
        _name_label.text = BagUiMapper.item_name(_item)
    if _type_label != null:
        if _context == DetailContext.EQUIPPED_ITEM:
            _type_label.text = "类型：装备"
        else:
            _type_label.text = BagUiMapper.item_type_text(_item)
    if _equip_slot_label != null:
        var equip_slot_text: String = BagUiMapper.equip_slot_text(_item)
        _equip_slot_label.text = equip_slot_text
        _equip_slot_label.visible = not equip_slot_text.is_empty()
    if _quantity_label != null:
        _quantity_label.text = "数量：%s" % UiFormat.value_to_text(BagUiMapper.quantity(_item))
        _quantity_label.visible = _context == DetailContext.BAG_ITEM and BagUiMapper.is_stackable(_item)
    if _description_label != null:
        var description_text: String = BagUiMapper.description(_item)
        if description_text.is_empty():
            _description_label.text = "暂无描述。"
        else:
            _description_label.text = description_text
    _refresh_actions()


## 清空详情面板。
func clear_item() -> void:
    _context = DetailContext.BAG_ITEM
    _item.clear()
    _more_menu_open = false
    if _name_label != null:
        _name_label.text = "未选择物品"
    if _type_label != null:
        _type_label.text = "类型：-"
    if _equip_slot_label != null:
        _equip_slot_label.text = ""
        _equip_slot_label.hide()
    if _quantity_label != null:
        _quantity_label.text = "数量：-"
        _quantity_label.hide()
    if _description_label != null:
        _description_label.text = "点击物品或已装备槽位查看详情。"
    _refresh_actions()


## 刷新主按钮、次按钮与更多菜单的文案和可见性。
func _refresh_actions() -> void:
    _refresh_primary_button()
    _refresh_secondary_button()
    _refresh_more_menu()


## 刷新左侧主操作按钮文案与启用态。
func _refresh_primary_button() -> void:
    if _primary_button == null:
        return
    if _item.is_empty():
        _primary_button.text = "操作"
        _primary_button.visible = false
        _primary_button.disabled = true
        return
    _primary_button.visible = true
    _primary_button.text = _resolve_primary_action_label()
    _primary_button.disabled = not _is_primary_action_enabled()


## 刷新右侧次操作按钮文案与启用态。
func _refresh_secondary_button() -> void:
    if _secondary_button == null:
        return
    if _item.is_empty():
        _secondary_button.text = "更多"
        _secondary_button.visible = false
        _secondary_button.disabled = true
        return
    _secondary_button.visible = true
    if _context == DetailContext.EQUIPPED_ITEM:
        _secondary_button.text = "分享"
        _secondary_button.disabled = false
    else:
        _secondary_button.text = "更多"
        _secondary_button.disabled = false


## 刷新更多菜单容器与内部按钮状态。
func _refresh_more_menu() -> void:
    if _more_menu_row != null:
        var should_show_more_menu: bool = (
            not _item.is_empty()
            and _context == DetailContext.BAG_ITEM
            and _more_menu_open
        )
        _more_menu_row.visible = should_show_more_menu
    for action_key: String in MORE_MENU_ACTION_KEYS:
        var button: Button = _more_menu_buttons.get(action_key, null) as Button
        if button == null:
            continue
        var should_enable: bool = (
            not _item.is_empty()
            and _context == DetailContext.BAG_ITEM
            and _is_more_menu_action_enabled(action_key)
        )
        button.disabled = not should_enable


## 根据当前上下文解析左侧主按钮应展示的中文文案。
func _resolve_primary_action_label() -> String:
    if _context == DetailContext.EQUIPPED_ITEM:
        return "卸下"
    if BagUiMapper.is_box_item(_item):
        return "打开"
    if BagUiMapper.is_equipment(_item):
        return "装备"
    return "使用"


## 根据当前上下文解析左侧主按钮应触发的协议动作 key。
func _resolve_primary_action_key() -> String:
    if _context == DetailContext.EQUIPPED_ITEM:
        return "unequip"
    if BagUiMapper.is_box_item(_item):
        return "open"
    return "use"


## 判断左侧主操作当前是否允许点击。
func _is_primary_action_enabled() -> bool:
    if _item.is_empty():
        return false
    if _context == DetailContext.EQUIPPED_ITEM:
        return true
    return BagUiMapper.supports_primary_action(_item)


## 判断更多菜单里的某个动作当前是否允许点击。
func _is_more_menu_action_enabled(action_key: String) -> bool:
    if _item.is_empty() or _context != DetailContext.BAG_ITEM:
        return false
    return action_key == "drop" or action_key == "give" or action_key == "share"


## 处理左侧主操作按钮点击。
func _on_primary_pressed() -> void:
    if _item.is_empty() or _primary_button == null or _primary_button.disabled:
        return
    _more_menu_open = false
    _refresh_more_menu()
    var action_key: String = _resolve_primary_action_key()
    action_requested.emit(action_key, _item.duplicate(true))


## 处理右侧次操作按钮点击：已穿戴直接分享，背包物品切换更多菜单。
func _on_secondary_pressed() -> void:
    if _item.is_empty() or _secondary_button == null or _secondary_button.disabled:
        return
    if _context == DetailContext.EQUIPPED_ITEM:
        action_requested.emit("share", _item.duplicate(true))
        return
    _more_menu_open = not _more_menu_open
    _refresh_more_menu()


## 转发更多菜单内按钮点击，并在触发后收起菜单。
func _on_more_menu_action_pressed(action_key: String) -> void:
    if _item.is_empty():
        return
    _more_menu_open = false
    _refresh_more_menu()
    action_requested.emit(action_key, _item.duplicate(true))
