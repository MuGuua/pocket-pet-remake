class_name DropItemPopup
extends CanvasLayer

## 丢弃物品专用确认弹窗场景路径。
const SCENE_PATH: String = "res://scenes/ui/bag/drop_item_popup.tscn"

## 玩家完成选择（确认或取消）后向外广播；载荷含 confirmed、quantity。
signal prompt_finished(result: Dictionary)

@onready var _item_icon: TextureRect = %ItemIcon
@onready var _item_name_label: RichTextLabel = %ItemNameLabel
@onready var _item_meta_label: Label = %ItemMetaLabel
@onready var _warning_label: RichTextLabel = %WarningLabel
@onready var _quantity_row: HBoxContainer = %QuantityRow
@onready var _quantity_decrease_button: RuntimeActionButton = %QuantityDecreaseButton
@onready var _quantity_value_label: Label = %QuantityValueLabel
@onready var _quantity_increase_button: RuntimeActionButton = %QuantityIncreaseButton
@onready var _quantity_max_button: RuntimeActionButton = %QuantityMaxButton
@onready var _confirm_button: RuntimeActionButton = %ConfirmButton
@onready var _cancel_button: RuntimeActionButton = %CancelButton
@onready var _top_close_button: BaseButton = %TopCloseButton
@onready var _dim_layer: ColorRect = $DimLayer

## 当前待丢弃物品快照。
var _item: Dictionary = {}
## 玩家选中的丢弃数量。
var _selected_quantity: int = 1
## 数量选择下限。
var _quantity_min: int = 1
## 数量选择上限。
var _quantity_max: int = 1
## 是否正在等待玩家确认，防止重复 finish。
var _prompt_active: bool = false


## 绑定按钮；默认隐藏，遮罩仅拦截点击不自动关闭。
func _ready() -> void:
	visible = false
	add_to_group("runtime_modal_popup")
	if _dim_layer != null and not _dim_layer.gui_input.is_connected(_on_dim_layer_gui_input):
		_dim_layer.gui_input.connect(_on_dim_layer_gui_input)
	if _confirm_button != null and not _confirm_button.pressed.is_connected(_on_confirm_pressed):
		_confirm_button.pressed.connect(_on_confirm_pressed)
	if _cancel_button != null and not _cancel_button.pressed.is_connected(_on_cancel_pressed):
		_cancel_button.pressed.connect(_on_cancel_pressed)
	if _top_close_button != null and not _top_close_button.pressed.is_connected(_on_top_close_button_pressed):
		_top_close_button.pressed.connect(_on_top_close_button_pressed)
	if _quantity_decrease_button != null and not _quantity_decrease_button.pressed.is_connected(_on_quantity_decrease_pressed):
		_quantity_decrease_button.pressed.connect(_on_quantity_decrease_pressed)
	if _quantity_increase_button != null and not _quantity_increase_button.pressed.is_connected(_on_quantity_increase_pressed):
		_quantity_increase_button.pressed.connect(_on_quantity_increase_pressed)
	if _quantity_max_button != null and not _quantity_max_button.pressed.is_connected(_on_quantity_max_pressed):
		_quantity_max_button.pressed.connect(_on_quantity_max_pressed)


## 打开丢弃确认弹窗并阻塞到玩家确认或取消；返回 confirmed 与 quantity。
func prompt_drop(item: Dictionary) -> Dictionary:
	_item = item.duplicate(true)
	_apply_item_view()
	_prompt_active = true
	_set_runtime_input_locked(true)
	show()
	_raise_above_sibling_popups()
	var result: Dictionary = await prompt_finished
	return result


## 将弹窗移到父节点子树末尾，保证盖住背包内其它 overlay。
func _raise_above_sibling_popups() -> void:
	var parent_node: Node = get_parent()
	if parent_node == null:
		return
	parent_node.move_child(self, parent_node.get_child_count() - 1)


## 外部强制关闭（例如切战斗）；视为取消。
func force_close_popup() -> void:
	if not visible:
		return
	_finish_prompt(false, 0)


## 点击空白遮罩时只吞掉输入，不再自动关闭弹窗。
func _on_dim_layer_gui_input(event: InputEvent) -> void:
	if not visible or not _prompt_active:
		return
	if not _is_dismiss_event(event):
		return
	get_viewport().set_input_as_handled()
	if _dim_layer != null:
		_dim_layer.accept_event()


## 判断是否为“按下类”关闭输入，移动端触摸与桌面鼠标统一处理。
func _is_dismiss_event(event: InputEvent) -> bool:
	if event is InputEventScreenTouch:
		return (event as InputEventScreenTouch).pressed
	if event is InputEventMouseButton:
		return (event as InputEventMouseButton).pressed
	if event is InputEventKey:
		var key_event: InputEventKey = event as InputEventKey
		return key_event.pressed and not key_event.echo
	if event is InputEventJoypadButton:
		return (event as InputEventJoypadButton).pressed
	return false


## 刷新物品图标、名称与数量选择器。
func _apply_item_view() -> void:
	var item_name: String = BagUiMapper.item_name(_item)
	if _item_name_label != null:
		RichTextContent.apply_system_name(_item_name_label, item_name)
	if _item_icon != null:
		_item_icon.texture = BagUiMapper.icon_texture(_item)
	if _item_meta_label != null:
		var meta_parts: Array[String] = []
		var type_text: String = BagUiMapper.item_type_text(_item)
		if not type_text.is_empty():
			meta_parts.append(type_text)
		var slot_text: String = BagUiMapper.equip_slot_text(_item)
		if not slot_text.is_empty():
			meta_parts.append(slot_text)
		var qty: int = BagUiMapper.quantity(_item)
		if qty > 1:
			meta_parts.append("数量 %d" % qty)
		_item_meta_label.text = " · ".join(meta_parts) if not meta_parts.is_empty() else ""
	if _warning_label != null:
		_warning_label.text = "确定要丢弃 %s 吗？\n[color=#c9a227]此操作不可撤销。[/color]" % item_name
	var supports_partial: bool = BagUiMapper.supports_partial_drop(_item)
	_quantity_min = 1
	_quantity_max = maxi(1, BagUiMapper.quantity(_item)) if supports_partial else 1
	_selected_quantity = _quantity_max if supports_partial else BagUiMapper.default_drop_quantity(_item)
	if _quantity_row != null:
		_quantity_row.visible = supports_partial
	_refresh_quantity_label()


## 刷新数量标签。
func _refresh_quantity_label() -> void:
	if _quantity_value_label != null:
		_quantity_value_label.text = str(_selected_quantity)


## 玩家点击确认丢弃。
func _on_confirm_pressed() -> void:
	if not _prompt_active:
		return
	_finish_prompt(true, _selected_quantity)


## 玩家点击取消。
func _on_cancel_pressed() -> void:
	if not _prompt_active:
		return
	_finish_prompt(false, 0)


## 玩家点击右上角关闭按钮时视为取消。
func _on_top_close_button_pressed() -> void:
	if not _prompt_active:
		return
	_finish_prompt(false, 0)


## 数量减一。
func _on_quantity_decrease_pressed() -> void:
	_selected_quantity = clampi(_selected_quantity - 1, _quantity_min, _quantity_max)
	_refresh_quantity_label()


## 数量加一。
func _on_quantity_increase_pressed() -> void:
	_selected_quantity = clampi(_selected_quantity + 1, _quantity_min, _quantity_max)
	_refresh_quantity_label()


## 数量设为最大。
func _on_quantity_max_pressed() -> void:
	_selected_quantity = _quantity_max
	_refresh_quantity_label()


## 结束一次 prompt 并广播结果；遮罩点击不会走到这里。
func _finish_prompt(confirmed: bool, quantity: int) -> void:
	if not _prompt_active:
		return
	get_viewport().set_input_as_handled()
	_prompt_active = false
	hide()
	_set_runtime_input_locked(false)
	prompt_finished.emit({
		"confirmed": confirmed,
		"quantity": quantity,
	})


## 向上查找主场景并锁定/解锁世界交互。
func _set_runtime_input_locked(locked: bool) -> void:
	var host: Node = self
	while host != null:
		if host.has_method("_set_runtime_menu_locked"):
			host.call("_set_runtime_menu_locked", locked)
			return
		host = host.get_parent()
