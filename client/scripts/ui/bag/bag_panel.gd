extends CanvasLayer
class_name BagPanel

signal menu_closed

const SLOTS_PER_PAGE: int = 28
const DEFAULT_CAPACITY: int = 300

@onready var _close_button: Button = $RootPanel/MarginContainer/VBoxContainer/Title/HBoxContainer/Button
@onready var _equipment_root: Node = $RootPanel/MarginContainer/VBoxContainer/EquipmentSlot
@onready var _capacity_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/PanelContainer/HBoxContainer/Label
@onready var _gold_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/GoldCoinAmount
@onready var _silver_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/SilverCoinAmount
@onready var _copper_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/CopperCoinAmount
@onready var _slot_grid: GridContainer = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer/GridContainer
@onready var _prev_page_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/Button4
@onready var _next_page_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/Button5
@onready var _page_input: LineEdit = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/PanelContainer/PageInput

var _slots: Array[BagSlot] = []
var _current_page: int = 0
var _selected_slot_index: int = 0


func _ready() -> void:
	hide()
	if _close_button != null:
		_close_button.pressed.connect(close_menu)
	if _prev_page_button != null:
		_prev_page_button.pressed.connect(_on_prev_page_pressed)
	if _next_page_button != null:
		_next_page_button.pressed.connect(_on_next_page_pressed)
	if _page_input != null:
		_page_input.text_submitted.connect(_on_page_input_submitted)
		_page_input.focus_exited.connect(_on_page_input_focus_exited)
	if not GameState.bag_changed.is_connected(_on_bag_data_changed):
		GameState.bag_changed.connect(_on_bag_data_changed)
	if not GameState.wallet_changed.is_connected(_on_bag_data_changed):
		GameState.wallet_changed.connect(_on_bag_data_changed)
	if not GameState.equipment_changed.is_connected(_refresh_equipment_slots):
		GameState.equipment_changed.connect(_refresh_equipment_slots)
	_collect_bag_slots()
	_refresh_panel()


func _exit_tree() -> void:
	if GameState.bag_changed.is_connected(_on_bag_data_changed):
		GameState.bag_changed.disconnect(_on_bag_data_changed)
	if GameState.wallet_changed.is_connected(_on_bag_data_changed):
		GameState.wallet_changed.disconnect(_on_bag_data_changed)
	if GameState.equipment_changed.is_connected(_refresh_equipment_slots):
		GameState.equipment_changed.disconnect(_refresh_equipment_slots)


func open_menu() -> void:
	show()
	if GameState.is_ws_authenticated:
		App.request_bag_list()
	_refresh_panel()


func close_menu() -> void:
	var was_visible: bool = visible
	hide()
	if was_visible:
		menu_closed.emit()


func _on_bag_data_changed() -> void:
	if visible:
		_refresh_panel()


func _refresh_panel() -> void:
	_refresh_capacity_and_wallet()
	_refresh_bag_slots()
	_update_page_controls()
	_refresh_equipment_slots()


func _refresh_capacity_and_wallet() -> void:
	var capacity: int = _resolve_capacity()
	var used_slots: int = int(GameState.bag_container.get("used_slots", _count_used_bag_slots()))
	if _capacity_label != null:
		_capacity_label.text = "%d/%d" % [used_slots, capacity]

	var wallet: Dictionary = GameState.wallet_snapshot
	if _gold_label != null:
		_gold_label.text = UiFormat.value_to_text(wallet.get("gold", 0))
	if _silver_label != null:
		_silver_label.text = UiFormat.value_to_text(wallet.get("silver", 0))
	if _copper_label != null:
		_copper_label.text = UiFormat.value_to_text(wallet.get("copper", 0))


func _refresh_bag_slots() -> void:
	if _slots.is_empty():
		return

	var items_by_slot: Dictionary = _build_items_by_slot()
	var page_count: int = _get_page_count()
	if page_count <= 0:
		_current_page = 0
	else:
		_current_page = clampi(_current_page, 0, page_count - 1)

	var page_start_slot: int = _current_page * SLOTS_PER_PAGE
	for index: int in range(_slots.size()):
		var slot: BagSlot = _slots[index]
		var slot_index: int = page_start_slot + index + 1
		if items_by_slot.has(slot_index):
			slot.set_item(items_by_slot[slot_index] as Dictionary)
		else:
			slot.clear_item()
		slot.set_selected(slot_index == _selected_slot_index)


func _update_page_controls() -> void:
	var page_count: int = _get_page_count()
	if _prev_page_button != null:
		_prev_page_button.disabled = _current_page <= 0 or page_count <= 1
	if _next_page_button != null:
		_next_page_button.disabled = _current_page >= page_count - 1 or page_count <= 1
	_sync_page_input()


func _sync_page_input() -> void:
	if _page_input == null or _page_input.has_focus():
		return
	var page_count: int = max(_get_page_count(), 1)
	_page_input.text = str(_current_page + 1)
	_page_input.tooltip_text = "第 %d / %d 页" % [_current_page + 1, page_count]


func _on_prev_page_pressed() -> void:
	_set_page(_current_page - 1)


func _on_next_page_pressed() -> void:
	_set_page(_current_page + 1)


func _on_page_input_submitted(new_text: String) -> void:
	_apply_page_input(new_text)
	if _page_input != null:
		_page_input.release_focus()


func _on_page_input_focus_exited() -> void:
	if _page_input == null:
		return
	_apply_page_input(_page_input.text)


func _apply_page_input(raw_text: String) -> void:
	var trimmed_text: String = raw_text.strip_edges()
	if trimmed_text.is_empty() or not trimmed_text.is_valid_int():
		_sync_page_input()
		return
	_set_page(int(trimmed_text) - 1)


func _set_page(page_index: int) -> void:
	var page_count: int = _get_page_count()
	if page_count <= 0:
		_current_page = 0
	else:
		_current_page = clampi(page_index, 0, page_count - 1)
	_refresh_bag_slots()
	_update_page_controls()


func _get_page_count() -> int:
	var capacity: int = _resolve_capacity()
	if capacity <= 0:
		return 1
	return int(ceil(float(capacity) / float(SLOTS_PER_PAGE)))


func _resolve_capacity() -> int:
	var capacity: int = int(GameState.bag_container.get("capacity", GameState.bag_container.get("max_capacity", 0)))
	if capacity <= 0:
		capacity = max(DEFAULT_CAPACITY, _count_used_bag_slots())
	return capacity


func _count_used_bag_slots() -> int:
	var count: int = 0
	for item_variant: Variant in _collect_bag_items():
		if item_variant is Dictionary:
			count += 1
	return count


func _build_items_by_slot() -> Dictionary:
	var items_by_slot: Dictionary = {}
	for item_variant: Variant in _collect_bag_items():
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		var slot_index: int = BagUiMapper.slot_index(item)
		if slot_index <= 0:
			continue
		items_by_slot[slot_index] = item
	return items_by_slot


func _collect_bag_items() -> Array:
	var container_items: Variant = GameState.bag_container.get("items", [])
	if container_items is Array:
		return (container_items as Array).duplicate(true)
	return GameState.bag_items.duplicate(true)


func _collect_bag_slots() -> void:
	_slots.clear()
	if _slot_grid == null:
		return
	for child: Node in _slot_grid.get_children():
		if child is BagSlot:
			var slot: BagSlot = child as BagSlot
			if not slot.item_selected.is_connected(_on_slot_item_selected):
				slot.item_selected.connect(_on_slot_item_selected)
			_slots.append(slot)


func _on_slot_item_selected(item: Dictionary) -> void:
	_selected_slot_index = BagUiMapper.slot_index(item)
	var page_start_slot: int = _current_page * SLOTS_PER_PAGE
	for index: int in range(_slots.size()):
		var slot_index: int = page_start_slot + index + 1
		_slots[index].set_selected(slot_index == _selected_slot_index)


func _refresh_equipment_slots() -> void:
	var slots: Array[EquipmentSlot] = []
	_collect_equipment_slots(_equipment_root, slots)
	for slot: EquipmentSlot in slots:
		slot.refresh_from_game_state()


func _collect_equipment_slots(node: Node, result: Array[EquipmentSlot]) -> void:
	if node is EquipmentSlot:
		result.append(node as EquipmentSlot)
	for child: Node in node.get_children():
		_collect_equipment_slots(child, result)
