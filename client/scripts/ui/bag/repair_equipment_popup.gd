class_name RepairEquipmentPopup
extends CanvasLayer

## 装备修复确认弹窗场景路径。
const SCENE_PATH: String = "res://scenes/ui/bag/repair_equipment_popup.tscn"

## 玩家完成选择（确认或取消）后向外广播；载荷含 confirmed。
signal prompt_finished(result: Dictionary)

@onready var _item_icon: TextureRect = %ItemIcon
@onready var _item_name_label: RichTextLabel = %ItemNameLabel
@onready var _item_meta_label: Label = %ItemMetaLabel
@onready var _warning_label: RichTextLabel = %WarningLabel
@onready var _confirm_button: RuntimeActionButton = %ConfirmButton
@onready var _cancel_button: RuntimeActionButton = %CancelButton
@onready var _top_close_button: BaseButton = %TopCloseButton
@onready var _dim_layer: ColorRect = $DimLayer

## 当前待修复装备快照。
var _item: Dictionary = {}
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


## 打开修复确认弹窗并阻塞到玩家确认或取消。
func prompt_repair(item: Dictionary) -> Dictionary:
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
    _finish_prompt(false)


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


## 刷新物品图标、名称与修复消耗文案。
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
        _item_meta_label.text = " · ".join(meta_parts) if not meta_parts.is_empty() else ""
    if _warning_label != null:
        var cost_text: String = BagUiMapper.repair_cost_text(_item)
        _warning_label.text = "确定要修复吗？\n%s" % cost_text
    if _confirm_button != null:
        _confirm_button.disabled = not BagUiMapper.supports_repair_action(_item)


## 玩家点击确认修复。
func _on_confirm_pressed() -> void:
    if not _prompt_active:
        return
    _finish_prompt(true)


## 玩家点击取消。
func _on_cancel_pressed() -> void:
    if not _prompt_active:
        return
    _finish_prompt(false)


## 玩家点击右上角关闭按钮时视为取消。
func _on_top_close_button_pressed() -> void:
    if not _prompt_active:
        return
    _finish_prompt(false)


## 结束一次 prompt 并广播结果。
func _finish_prompt(confirmed: bool) -> void:
    if not _prompt_active:
        return
    get_viewport().set_input_as_handled()
    _prompt_active = false
    hide()
    _set_runtime_input_locked(false)
    prompt_finished.emit({
        "confirmed": confirmed,
    })


## 向上查找主场景并锁定/解锁世界交互。
func _set_runtime_input_locked(locked: bool) -> void:
    var host: Node = self
    while host != null:
        if host.has_method("_set_runtime_menu_locked"):
            host.call("_set_runtime_menu_locked", locked)
            return
        host = host.get_parent()
