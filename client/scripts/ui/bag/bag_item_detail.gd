extends PanelContainer
class_name BagItemDetail

signal action_requested(action_key: String, item: Dictionary)

enum DetailContext {
	BAG_ITEM,
	EQUIPPED_ITEM,
}

const ACTION_MENU_SCENE: PackedScene = preload(ActionMenuPopup.SCENE_PATH)
## 物品名称高亮色，与背包悬停名称保持一致。
const NAME_COLOR: Color = Color(1, 0.952941, 0.745098, 1)
## 空态名称与引导文案色。
const EMPTY_TEXT_COLOR: Color = Color(0.580392, 0.580392, 0.721569, 1)
## 元信息标签前缀色（等级、类型、部位等）。
const META_LABEL_COLOR_HEX: String = "#9494b8"
## 元信息数值色。
const META_VALUE_COLOR_HEX: String = "#f0d5b1"
## 强化等级数值色。
const ENHANCE_VALUE_COLOR_HEX: String = "#82d563"

## 物品名称标签；布局与样式在 bag_item_detail.tscn 中编辑。
@onready var _name_label: Label = %NameLabel
## 佩戴等级标签。
@onready var _level_label: RichTextLabel = %LevelLabel
## 强化等级标签。
@onready var _reinforcement_label: RichTextLabel = %ReinforcementLabel
## 等级与强化信息行容器。
@onready var _meta_row2: HBoxContainer = %MetaRow2
## 物品类型标签。
@onready var _type_label: RichTextLabel = %TypeLabel
## 装备部位标签。
@onready var _equip_slot_label: RichTextLabel = %EquipSlotLabel
## 物品描述区；支持 {item:ID} 占位符内联展示 icon。
@onready var _description_label: ItemDescriptionView = %DescriptionLabel
## 左侧主操作按钮：打开 / 使用 / 装备 / 卸下。
@onready var _primary_button: RuntimeActionButton = %PrimaryButton
## 中间强化按钮占位容器；非装备物品仍保留宽度，避免左右按钮错位。
@onready var _enhance_slot: Control = %EnhanceSlot
## 中间强化按钮：仅背包装备时显示，占位由 EnhanceSlot 承担。
@onready var _enhance_button: RuntimeActionButton = %EnhanceButton
## 右侧次操作按钮：已穿戴显示分享，背包物品显示更多。
@onready var _secondary_button: RuntimeActionButton = %SecondaryButton

## 当前详情展示上下文：背包格子物品或已穿戴装备。
var _context: DetailContext = DetailContext.BAG_ITEM
## 当前选中的服务端物品快照。
var _item: Dictionary = {}
## 独立浮层更多菜单；运行时创建，不参与详情面板布局与尺寸。
var _more_menu: ActionMenuPopup = null


## 绑定场景内按钮信号，并初始化空态文案。
func _ready() -> void:
	if _primary_button != null and not _primary_button.pressed.is_connected(_on_primary_pressed):
		_primary_button.pressed.connect(_on_primary_pressed)
	if _enhance_button != null and not _enhance_button.pressed.is_connected(_on_enhance_pressed):
		_enhance_button.pressed.connect(_on_enhance_pressed)
	if _secondary_button != null and not _secondary_button.pressed.is_connected(_on_secondary_pressed):
		_secondary_button.pressed.connect(_on_secondary_pressed)
	_ensure_more_menu()
	_apply_static_text_theme()
	clear_item()


## 初始化详情面板固定文本样式。
func _apply_static_text_theme() -> void:
	if _name_label != null:
		_name_label.add_theme_color_override("font_color", NAME_COLOR)
		_name_label.add_theme_color_override("font_outline_color", Color(0, 0, 0, 1))
		_name_label.add_theme_constant_override("outline_size", 1)


## 懒创建更多菜单浮层，并挂到详情面板根节点下。
func _ensure_more_menu() -> void:
	if _more_menu != null:
		return
	_more_menu = ACTION_MENU_SCENE.instantiate() as ActionMenuPopup
	if _more_menu == null:
		return
	_more_menu.configure_actions(ActionMenuPopup.BAG_ITEM_ACTIONS)
	add_child(_more_menu)
	if not _more_menu.action_selected.is_connected(_on_more_menu_action_selected):
		_more_menu.action_selected.connect(_on_more_menu_action_selected)
	if not _more_menu.menu_closed.is_connected(_on_more_menu_closed):
		_more_menu.menu_closed.connect(_on_more_menu_closed)


## 更多菜单关闭后停止监听全局点击，避免无菜单时仍处理输入。
func _on_more_menu_closed() -> void:
	set_process_input(false)


## 菜单展开时监听全局点击：点三个动作按钮之外任意区域即收起菜单。
func _input(event: InputEvent) -> void:
	if _more_menu == null or not _more_menu.is_open():
		return
	if not _is_outside_close_event(event):
		return
	var click_pos: Vector2 = _event_global_position(event)
	if _more_menu.is_global_point_over_action_buttons(click_pos):
		return
	if _secondary_button != null and _secondary_button.get_global_rect().has_point(click_pos):
		return
	_hide_more_menu()


## 判断是否为可用于“点空白关闭菜单”的按下事件（鼠标左键或触屏按下）。
func _is_outside_close_event(event: InputEvent) -> bool:
	if event is InputEventScreenTouch:
		return (event as InputEventScreenTouch).pressed
	if event is InputEventMouseButton:
		var mouse_event: InputEventMouseButton = event as InputEventMouseButton
		return mouse_event.pressed and mouse_event.button_index == MOUSE_BUTTON_LEFT
	return false


## 从鼠标或触屏事件中取出全局坐标，供命中区域判断。
func _event_global_position(event: InputEvent) -> Vector2:
	if event is InputEventScreenTouch:
		return (event as InputEventScreenTouch).position
	if event is InputEventMouseButton:
		return (event as InputEventMouseButton).global_position
	return Vector2.ZERO


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
	_hide_more_menu()
	if _name_label != null:
		_name_label.text = BagUiMapper.item_name(_item)
		_name_label.add_theme_color_override("font_color", NAME_COLOR)
	_refresh_level_labels()
	if _type_label != null:
		if _context == DetailContext.EQUIPPED_ITEM:
			_set_meta_line(_type_label, "类型：", "装备")
		else:
			_apply_meta_line_from_text(_type_label, BagUiMapper.item_type_text(_item), "类型：")
	if _equip_slot_label != null:
		var equip_slot_text: String = BagUiMapper.equip_slot_text(_item)
		_apply_meta_line_from_text(_equip_slot_label, equip_slot_text, "部位：")
		_equip_slot_label.visible = not equip_slot_text.is_empty()
	if _description_label != null:
		_description_label.apply_item_snapshot(_item)
	_refresh_actions()


## 清空详情面板。
func clear_item() -> void:
	_context = DetailContext.BAG_ITEM
	_item.clear()
	_hide_more_menu()
	if _name_label != null:
		_name_label.text = "未选择物品"
		_name_label.add_theme_color_override("font_color", EMPTY_TEXT_COLOR)
	_refresh_level_labels()
	if _type_label != null:
		_set_meta_line(_type_label, "类型：", "-")
	if _equip_slot_label != null:
		_equip_slot_label.text = ""
		_equip_slot_label.hide()
	if _description_label != null:
		_description_label.apply_empty_hint("点击物品或已装备槽位查看详情。")
	_refresh_actions()


## 刷新佩戴等级与强化等级标签；数据来自服务端 required_level / enhance_level。
func _refresh_level_labels() -> void:
	var show_row: bool = not _item.is_empty() and BagUiMapper.should_show_equipment_level_row(_item)
	if _meta_row2 != null:
		_meta_row2.visible = show_row
	if not show_row:
		if _level_label != null:
			_level_label.text = ""
			_level_label.hide()
		if _reinforcement_label != null:
			_reinforcement_label.text = ""
			_reinforcement_label.hide()
		return
	var level_text: String = BagUiMapper.required_level_text(_item)
	if _level_label != null:
		_apply_meta_line_from_text(_level_label, level_text, "等级：")
		_level_label.visible = not level_text.is_empty()
	var enhance_text: String = BagUiMapper.enhance_level_text(_item)
	if _reinforcement_label != null:
		_apply_meta_line_from_text(_reinforcement_label, enhance_text, "强化：", ENHANCE_VALUE_COLOR_HEX)
		_reinforcement_label.visible = not enhance_text.is_empty()


## 将「标签：数值」格式化为分色 BBCode 并写入元信息 RichTextLabel。
func _apply_meta_line_from_text(
	label: RichTextLabel,
	full_text: String,
	label_prefix: String,
	value_color_hex: String = META_VALUE_COLOR_HEX
) -> void:
	if label == null:
		return
	if full_text.is_empty():
		label.text = ""
		return
	var value_text: String = full_text
	if full_text.begins_with(label_prefix):
		value_text = full_text.substr(label_prefix.length())
	_set_meta_line(label, label_prefix, value_text, value_color_hex)


## 写入带标签前缀与数值分色的元信息行。
func _set_meta_line(
	label: RichTextLabel,
	label_prefix: String,
	value_text: String,
	value_color_hex: String = META_VALUE_COLOR_HEX
) -> void:
	if label == null:
		return
	label.bbcode_enabled = true
	label.text = "[color=%s]%s[/color][color=%s]%s[/color]" % [
		META_LABEL_COLOR_HEX,
		label_prefix,
		value_color_hex,
		value_text,
	]


## 关闭更多菜单弹窗（若已打开）。
func _hide_more_menu() -> void:
	if _more_menu == null:
		return
	if _more_menu.is_open():
		_more_menu.hide_menu()
	set_process_input(false)


## 在「更多」按钮上方展开菜单，并开启全局点击监听以支持点外部关闭。
func _show_more_menu() -> void:
	if _more_menu == null or _secondary_button == null:
		return
	_more_menu.open_near(_secondary_button)
	set_process_input(true)


## 刷新主按钮、强化按钮与次按钮的文案和可见性。
func _refresh_actions() -> void:
	_refresh_primary_button()
	_refresh_enhance_button()
	_refresh_secondary_button()


## 刷新左侧主操作按钮文案与启用态。
func _refresh_primary_button() -> void:
	if _primary_button == null:
		return
	if _item.is_empty():
		_primary_button.set_button_label("操作")
		_primary_button.visible = false
		_primary_button.disabled = true
		return
	_primary_button.visible = true
	_primary_button.set_button_label(_resolve_primary_action_label())
	_primary_button.disabled = not _is_primary_action_enabled()


## 刷新中间强化区：背包装备显示按钮，其他背包物品仅保留占位 Control。
func _refresh_enhance_button() -> void:
	if _enhance_button == null:
		return
	var show_slot: bool = (
		not _item.is_empty()
		and _context == DetailContext.BAG_ITEM
	)
	var show_enhance: bool = show_slot and BagUiMapper.is_equipment(_item)
	if _enhance_slot != null:
		_enhance_slot.visible = show_slot
	_enhance_button.visible = show_enhance
	_enhance_button.disabled = not show_enhance
	if show_enhance:
		_enhance_button.set_button_label("强化")


## 刷新右侧次操作按钮文案与启用态。
func _refresh_secondary_button() -> void:
	if _secondary_button == null:
		return
	if _item.is_empty():
		_secondary_button.set_button_label("更多")
		_secondary_button.visible = false
		_secondary_button.disabled = true
		return
	_secondary_button.visible = true
	if _context == DetailContext.EQUIPPED_ITEM:
		_secondary_button.set_button_label("分享")
		_secondary_button.disabled = false
	else:
		_secondary_button.set_button_label("更多")
		_secondary_button.disabled = false


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


## 处理左侧主操作按钮点击。
func _on_primary_pressed() -> void:
	if _item.is_empty() or _primary_button == null or _primary_button.disabled:
		return
	_hide_more_menu()
	var action_key: String = _resolve_primary_action_key()
	action_requested.emit(action_key, _item.duplicate(true))


## 处理强化按钮点击：关闭详情并由外层打开强化弹窗。
func _on_enhance_pressed() -> void:
	if _item.is_empty() or _enhance_button == null or _enhance_button.disabled:
		return
	_hide_more_menu()
	action_requested.emit("enhance", _item.duplicate(true))


## 处理右侧次操作按钮点击：已穿戴直接分享，背包物品弹出独立更多菜单。
func _on_secondary_pressed() -> void:
	if _item.is_empty() or _secondary_button == null or _secondary_button.disabled:
		return
	if _context == DetailContext.EQUIPPED_ITEM:
		action_requested.emit("share", _item.duplicate(true))
		return
	if _more_menu != null and _more_menu.is_open():
		_hide_more_menu()
		return
	if _more_menu != null:
		_show_more_menu()


## 转发更多菜单内按钮点击。
func _on_more_menu_action_selected(action_key: String) -> void:
	if _item.is_empty():
		return
	action_requested.emit(action_key, _item.duplicate(true))
