extends PanelContainer
class_name ActionPanel

const CHOICE_PAGE_SIZE: int = 10

signal action_selected(action_type: String)
signal list_choice_selected(list_type: String, choice_index: int)
signal list_choice_cancelled(list_type: String)
signal target_selection_cancelled()
signal target_selection_confirmed()

@onready var _buttons: Array[Button] = [
	%AttackButton,
	%SkillButton,
	%ItemButton,
	%AutoButton,
	%EscapeButton
]
@onready var _choice_backdrop: ColorRect = %ChoiceBackdrop
@onready var _choice_modal_center: CenterContainer = %ChoiceModalCenter
@onready var _choice_list_vbox: VBoxContainer = %ChoiceListVBox
@onready var _choice_page_bar: HBoxContainer = %ChoicePageBar
@onready var _choice_prev_button: Button = %ChoicePrevButton
@onready var _choice_page_label: Label = %ChoicePageLabel
@onready var _choice_next_button: Button = %ChoiceNextButton
@onready var _choice_cancel_button: Button = %ChoiceCancelButton
@onready var _target_action_bar: HBoxContainer = %TargetCancelBar
@onready var _target_confirm_button: Button = %TargetConfirmButton
@onready var _target_cancel_button: Button = %TargetCancelButton

var _default_button_texts: Array[String] = []
var _choice_list_open: bool = false
var _active_list_type: String = ""
var _choice_entries: Array[Dictionary] = []
var _choice_page: int = 0
var _target_selection_active: bool = false

func _ready() -> void:
	for button: Button in _buttons:
		_default_button_texts.append(button.text)
	_reconnect_action_buttons()
	_choice_cancel_button.pressed.connect(_on_choice_cancel_pressed)
	_choice_prev_button.pressed.connect(_on_choice_prev_pressed)
	_choice_next_button.pressed.connect(_on_choice_next_pressed)
	_target_confirm_button.pressed.connect(_on_target_confirm_pressed)
	_target_cancel_button.pressed.connect(_on_target_cancel_pressed)
	close_choice_list()
	set_target_selection_mode(false)

func set_buttons_disabled(disabled: bool) -> void:
	for button: Button in _buttons:
		button.disabled = disabled
		button.modulate = Color(1, 1, 1, 0.55) if disabled else Color.WHITE
	if disabled and not _target_selection_active:
		close_choice_list()


## 自动战斗开启后仅保留“自动”按钮可点，便于再次点击关闭托管。
func set_auto_mode_active(active: bool) -> void:
	if not active:
		return
	close_choice_list()
	set_target_selection_mode(false)
	for button: Button in _buttons:
		var is_auto_button: bool = button == %AutoButton
		button.disabled = not is_auto_button
		button.modulate = Color.WHITE if is_auto_button else Color(1, 1, 1, 0.55)

func set_target_selection_mode(enabled: bool) -> void:
	_target_selection_active = enabled
	set_buttons_disabled(enabled)
	_target_action_bar.visible = enabled
	set_target_confirm_enabled(false)
	close_choice_list()

func set_target_confirm_enabled(enabled: bool) -> void:
	if _target_confirm_button == null:
		return
	_target_confirm_button.disabled = not enabled
	_target_confirm_button.modulate = Color.WHITE if enabled else Color(1, 1, 1, 0.55)

func begin_selection_phase() -> void:
	close_choice_list()
	set_target_selection_mode(false)
	_restore_action_buttons()
	_reconnect_action_buttons()
	set_buttons_disabled(false)

func open_choice_list(list_type: String, entries: Array[Dictionary]) -> void:
	if entries.is_empty():
		return
	_active_list_type = list_type
	_choice_entries = entries.duplicate()
	_choice_page = 0
	_refresh_choice_page()
	_choice_backdrop.visible = true
	_choice_modal_center.visible = true
	_choice_list_open = true

func close_choice_list() -> void:
	_clear_choice_list_buttons()
	_choice_entries.clear()
	_choice_page = 0
	if _choice_backdrop != null:
		_choice_backdrop.visible = false
	if _choice_modal_center != null:
		_choice_modal_center.visible = false
	if _choice_page_bar != null:
		_choice_page_bar.visible = false
	_choice_list_open = false
	_active_list_type = ""

func _refresh_choice_page() -> void:
	_clear_choice_list_buttons()
	var page_count: int = _get_choice_page_count()
	if page_count <= 0:
		return
	_choice_page = clampi(_choice_page, 0, page_count - 1)
	var start_index: int = _choice_page * CHOICE_PAGE_SIZE
	var end_index: int = min(start_index + CHOICE_PAGE_SIZE, _choice_entries.size())
	for index: int in range(start_index, end_index):
		var entry: Dictionary = _choice_entries[index]
		var button: Button = Button.new()
		button.text = str(entry.get("display_name", "选项"))
		button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		button.pressed.connect(_on_choice_entry_pressed.bind(index))
		_choice_list_vbox.add_child(button)
	_update_choice_page_controls(page_count)

func _get_choice_page_count() -> int:
	if _choice_entries.is_empty():
		return 0
	return int(ceil(float(_choice_entries.size()) / float(CHOICE_PAGE_SIZE)))

func _update_choice_page_controls(page_count: int) -> void:
	var show_pager: bool = page_count > 1
	_choice_page_bar.visible = show_pager
	if not show_pager:
		return
	_choice_page_label.text = "%d / %d" % [_choice_page + 1, page_count]
	_choice_prev_button.disabled = _choice_page <= 0
	_choice_next_button.disabled = _choice_page >= page_count - 1

func _clear_choice_list_buttons() -> void:
	if _choice_list_vbox == null:
		return
	for child: Node in _choice_list_vbox.get_children():
		child.queue_free()

func _on_choice_prev_pressed() -> void:
	if _choice_page <= 0:
		return
	_choice_page -= 1
	_refresh_choice_page()

func _on_choice_next_pressed() -> void:
	if _choice_page >= _get_choice_page_count() - 1:
		return
	_choice_page += 1
	_refresh_choice_page()

func _on_choice_entry_pressed(choice_index: int) -> void:
	if not _choice_list_open or _active_list_type.is_empty():
		return
	if choice_index < 0 or choice_index >= _choice_entries.size():
		return
	var list_type: String = _active_list_type
	close_choice_list()
	list_choice_selected.emit(list_type, choice_index)

func _on_choice_cancel_pressed() -> void:
	if not _choice_list_open or _active_list_type.is_empty():
		return
	var list_type: String = _active_list_type
	close_choice_list()
	list_choice_cancelled.emit(list_type)

func _on_target_cancel_pressed() -> void:
	if not _target_selection_active:
		return
	target_selection_cancelled.emit()

func _on_target_confirm_pressed() -> void:
	if not _target_selection_active or _target_confirm_button.disabled:
		return
	target_selection_confirmed.emit()

func _emit_action(action_type: String) -> void:
	if not _target_selection_active:
		close_choice_list()
	action_selected.emit(action_type)

func _on_attack_pressed() -> void:
	_emit_action("attack")

func _on_skill_pressed() -> void:
	_emit_action("skill")

func _on_item_pressed() -> void:
	_emit_action("item")

func _on_auto_pressed() -> void:
	_emit_action("auto")

func _on_escape_pressed() -> void:
	_emit_action("escape")

func _disconnect_action_buttons() -> void:
	for button: Button in _buttons:
		for connection: Dictionary in button.pressed.get_connections():
			var callable_value: Callable = connection["callable"] as Callable
			if button.pressed.is_connected(callable_value):
				button.pressed.disconnect(callable_value)

func _reconnect_action_buttons() -> void:
	_disconnect_action_buttons()
	%AttackButton.pressed.connect(_on_attack_pressed)
	%SkillButton.pressed.connect(_on_skill_pressed)
	%ItemButton.pressed.connect(_on_item_pressed)
	%AutoButton.pressed.connect(_on_auto_pressed)
	%EscapeButton.pressed.connect(_on_escape_pressed)

func _restore_action_buttons() -> void:
	for index: int in range(_buttons.size()):
		var button: Button = _buttons[index]
		button.visible = true
		button.disabled = false
		button.modulate = Color.WHITE
		button.text = _default_button_texts[index]
