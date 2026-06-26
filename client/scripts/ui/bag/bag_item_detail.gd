extends PanelContainer
class_name BagItemDetail

signal action_requested(action_key: String, item: Dictionary)

enum DetailContext {
    BAG_ITEM,
    EQUIPPED_ITEM,
}

const ACTION_BUTTON_KEYS: Array[String] = [
    "open",
    "use",
    "unequip",
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
## 打开按钮。
@onready var _open_button: Button = %OpenButton
## 使用/装备按钮。
@onready var _use_button: Button = %UseButton
## 卸下按钮。
@onready var _unequip_button: Button = %UnequipButton
## 丢弃按钮。
@onready var _drop_button: Button = %DropButton
## 给人按钮。
@onready var _give_button: Button = %GiveButton
## 分享按钮。
@onready var _share_button: Button = %ShareButton

## 当前详情展示上下文：背包格子物品或已穿戴装备。
var _context: DetailContext = DetailContext.BAG_ITEM
## 当前选中的服务端物品快照。
var _item: Dictionary = {}
## 操作按钮索引，key 与协议动作一致。
var _action_buttons: Dictionary = {}


## 绑定场景内按钮信号，并初始化空态文案。
func _ready() -> void:
    _action_buttons = {
        "open": _open_button,
        "use": _use_button,
        "unequip": _unequip_button,
        "drop": _drop_button,
        "give": _give_button,
        "share": _share_button,
    }
    for action_key: String in ACTION_BUTTON_KEYS:
        var button: Button = _action_buttons.get(action_key, null) as Button
        if button == null:
            continue
        if not button.pressed.is_connected(_on_action_pressed.bind(action_key)):
            button.pressed.connect(_on_action_pressed.bind(action_key))
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
    _refresh_primary_action_text()
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
    _refresh_primary_action_text()
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


## 刷新按钮可见性与启用态；未接入功能仍保留入口但禁用。
func _refresh_actions() -> void:
    for action_key: String in ACTION_BUTTON_KEYS:
        var button: Button = _action_buttons.get(action_key, null) as Button
        if button == null:
            continue
        var should_show: bool = _is_action_visible(action_key)
        button.visible = should_show
        button.disabled = _item.is_empty() or not should_show or not _is_action_enabled(action_key)


## 按当前上下文决定某个操作按钮是否显示。
func _is_action_visible(action_key: String) -> bool:
    if _item.is_empty():
        return false
    match _context:
        DetailContext.EQUIPPED_ITEM:
            return action_key == "unequip" or action_key == "share"
        DetailContext.BAG_ITEM:
            if action_key == "drop" or action_key == "give" or action_key == "share":
                return true
            if BagUiMapper.is_equipment(_item):
                return action_key == "use"
            if action_key == "use":
                return BagUiMapper.supports_primary_action(_item) and not _should_prefer_open_action()
            if action_key == "open":
                return _should_prefer_open_action()
    return false


## 判断当前背包物品是否应优先展示“打开”而不是“使用”。
func _should_prefer_open_action() -> bool:
    if BagUiMapper.is_equipment(_item):
        return false
    return BagUiMapper.has_action(_item, "open") or str(_item.get("effect_type", "")).to_lower() == "open"


## 当前版本开放装备/使用/卸下；丢弃、给人、分享保留入口待后续接入服务端。
func _is_action_enabled(action_key: String) -> bool:
    match action_key:
        "use", "open":
            return BagUiMapper.supports_primary_action(_item)
        "unequip":
            return _context == DetailContext.EQUIPPED_ITEM
        "drop", "give", "share":
            return _context == DetailContext.BAG_ITEM
        _:
            return false


## 根据当前物品类型动态调整主操作按钮文案。
func _refresh_primary_action_text() -> void:
    if _use_button != null:
        if _context == DetailContext.EQUIPPED_ITEM:
            _use_button.text = "使用"
        elif BagUiMapper.is_equipment(_item):
            _use_button.text = "装备"
        else:
            _use_button.text = "使用"
    if _open_button != null:
        _open_button.text = "打开"


## 转发玩家点击的功能按钮。
func _on_action_pressed(action_key: String) -> void:
    if _item.is_empty():
        return
    action_requested.emit(action_key, _item.duplicate(true))
