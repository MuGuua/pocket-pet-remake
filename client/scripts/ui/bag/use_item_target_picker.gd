class_name UseItemTargetPicker
extends "res://scripts/ui/common/modal_popup_layer.gd"

## 消耗品使用目标选择弹窗场景路径。
const SCENE_PATH: String = "res://scenes/ui/bag/use_item_target_picker.tscn"

enum TargetMode {
    PET,
    EQUIPMENT,
}

signal target_selected(result: Dictionary)
signal target_cancelled

@onready var _title_label: Label = %TitleLabel
@onready var _list_container: VBoxContainer = %ListContainer
@onready var _empty_label: Label = %EmptyLabel
@onready var _cancel_button: RuntimeActionButton = %CancelButton


## 绑定按钮并默认隐藏。
func _ready() -> void:
    super._ready()
    visible = false
    if _cancel_button != null and not _cancel_button.pressed.is_connected(_on_cancel_pressed):
        _cancel_button.pressed.connect(_on_cancel_pressed)


## 展示宠物目标列表；entries 为 GameState.pets 中的字典快照。
func show_pet_targets(pets: Array) -> void:
    _open_target_picker(TargetMode.PET, pets)


## 展示装备目标列表；entries 为带来源标记的装备字典快照。
func show_equipment_targets(items: Array) -> void:
    _open_target_picker(TargetMode.EQUIPMENT, items)


## 清空旧选项并重建目标按钮列表。
func _open_target_picker(mode: TargetMode, entries: Array) -> void:
    if _title_label != null:
        _title_label.text = "选择目标宠物" if mode == TargetMode.PET else "选择目标装备"
    _clear_target_buttons()
    var has_entries: bool = false
    for entry_variant: Variant in entries:
        if entry_variant is not Dictionary:
            continue
        var entry: Dictionary = entry_variant as Dictionary
        if mode == TargetMode.PET:
            var pet_uid: int = int(entry.get("pet_uid", 0))
            if pet_uid <= 0:
                continue
            has_entries = true
            _add_target_button(BagUiMapper.format_pet_use_target_label(entry), Callable(self, "_on_pet_target_pressed").bind(pet_uid))
        else:
            var item_uid: String = BagUiMapper.item_uid(entry)
            if item_uid.is_empty():
                continue
            has_entries = true
            _add_target_button(BagUiMapper.format_equipment_use_target_label(entry), Callable(self, "_on_equipment_target_pressed").bind(item_uid))
    if _empty_label != null:
        _empty_label.visible = not has_entries
        if not has_entries:
            _empty_label.text = "暂无可选目标"
    if _cancel_button != null:
        _cancel_button.visible = true
    _open_modal()


## 移除列表容器中的动态按钮。
func _clear_target_buttons() -> void:
    if _list_container == null:
        return
    for child: Node in _list_container.get_children():
        if child == _empty_label:
            continue
        child.queue_free()


## 在列表中追加一条可点击目标。
func _add_target_button(label_text: String, pressed_callable: Callable) -> void:
    if _list_container == null:
        return
    var button: RuntimeActionButton = preload("res://scenes/ui/common/runtime_action_button.tscn").instantiate() as RuntimeActionButton
    if button == null:
        return
    button.custom_minimum_size = Vector2(360, 56)
    button.set_button_label(label_text)
    if not button.pressed.is_connected(pressed_callable):
        button.pressed.connect(pressed_callable)
    _list_container.add_child(button)
    _list_container.move_child(button, _list_container.get_child_count() - 1)


## 玩家选中宠物目标。
func _on_pet_target_pressed(pet_uid: int) -> void:
    if pet_uid <= 0:
        return
    _emit_target_selected({"confirmed": true, "pet_uid": pet_uid, "item_uid": ""})


## 玩家选中装备目标。
func _on_equipment_target_pressed(item_uid: String) -> void:
    var normalized_uid: String = item_uid.strip_edges()
    if normalized_uid.is_empty():
        return
    _emit_target_selected({"confirmed": true, "pet_uid": 0, "item_uid": normalized_uid})


## 玩家取消选择。
func _on_cancel_pressed() -> void:
    _emit_target_cancelled()


## 点击遮罩关闭时视为取消。
func _dismiss_modal() -> void:
    _emit_target_cancelled()


## 发送选中结果并关闭弹窗。
func _emit_target_selected(result: Dictionary) -> void:
    get_viewport().set_input_as_handled()
    target_selected.emit(result)
    _notify_host_suppress_input_leak()
    _close_modal()
    popup_closed.emit()


## 发送取消并关闭弹窗。
func _emit_target_cancelled() -> void:
    get_viewport().set_input_as_handled()
    target_cancelled.emit()
    _notify_host_suppress_input_leak()
    _close_modal()
    popup_closed.emit()
