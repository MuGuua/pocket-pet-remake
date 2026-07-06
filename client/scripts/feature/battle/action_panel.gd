extends PanelContainer
class_name ActionPanel

const CHOICE_PAGE_SIZE: int = 10

signal action_selected(action_type: String)
signal list_choice_selected(list_type: String, choice_index: int)
signal list_choice_cancelled(list_type: String)
signal target_selection_cancelled()
signal target_selection_confirmed()

@onready var _buttons: Array[Button] = [
	get_node_or_null("%AttackButton") as Button,
	get_node_or_null("%SkillButton") as Button,
	get_node_or_null("%ItemButton") as Button,
	get_node_or_null("%AutoButton") as Button,
	get_node_or_null("%EscapeButton") as Button
]
@onready var _choice_backdrop: ColorRect = get_node_or_null("%ChoiceBackdrop") as ColorRect
@onready var _choice_modal_center: CenterContainer = get_node_or_null("%ChoiceModalCenter") as CenterContainer
@onready var _choice_list_vbox: VBoxContainer = get_node_or_null("%ChoiceListVBox") as VBoxContainer
## 选择弹窗里的文字按钮全部来自 battle_scene.tscn，便于在编辑器中直接调整尺寸和样式。
@onready var _choice_buttons: Array[Button] = [
	get_node_or_null("%ChoiceButton1") as Button,
	get_node_or_null("%ChoiceButton2") as Button,
	get_node_or_null("%ChoiceButton3") as Button,
	get_node_or_null("%ChoiceButton4") as Button,
	get_node_or_null("%ChoiceButton5") as Button,
	get_node_or_null("%ChoiceButton6") as Button,
	get_node_or_null("%ChoiceButton7") as Button,
	get_node_or_null("%ChoiceButton8") as Button,
	get_node_or_null("%ChoiceButton9") as Button,
	get_node_or_null("%ChoiceButton10") as Button
]
@onready var _choice_page_bar: HBoxContainer = get_node_or_null("%ChoicePageBar") as HBoxContainer
@onready var _choice_prev_button: Button = get_node_or_null("%ChoicePrevButton") as Button
@onready var _choice_page_label: Label = get_node_or_null("%ChoicePageLabel") as Label
@onready var _choice_next_button: Button = get_node_or_null("%ChoiceNextButton") as Button
@onready var _choice_cancel_button: Button = get_node_or_null("%ChoiceCancelButton") as Button
@onready var _target_action_bar: HBoxContainer = get_node_or_null("%TargetCancelBar") as HBoxContainer
@onready var _target_confirm_button: Button = get_node_or_null("%TargetConfirmButton") as Button
@onready var _target_cancel_button: Button = get_node_or_null("%TargetCancelButton") as Button
## 主操作按钮下方的选中底图；正常状态隐藏，悬停/聚焦/按下时显示。
@onready var _button_highlight_images: Dictionary = {}

var _default_button_texts: Array[String] = []
var _choice_list_open: bool = false
var _active_list_type: String = ""
var _choice_entries: Array[Dictionary] = []
var _choice_page: int = 0
var _target_selection_active: bool = false
## 记录每个主操作按钮是否正处于鼠标按下态，避免 button_up 前底图闪烁。
var _button_pressing_states: Dictionary = {}
## 记录刚完成选择的按钮，直到鼠标离开或再次按下前不再因为悬停残留显示底图。
var _button_highlight_suppressed: Dictionary = {}

func _ready() -> void:
	for button: Button in _buttons:
		if button == null:
			_default_button_texts.append("")
			continue
		_default_button_texts.append(button.text)
	_cache_action_button_highlight_images()
	_connect_action_button_state_signals()
	_reconnect_action_buttons()
	_connect_choice_buttons()
	_connect_button_pressed(_choice_cancel_button, _on_choice_cancel_pressed)
	_connect_button_pressed(_choice_prev_button, _on_choice_prev_pressed)
	_connect_button_pressed(_choice_next_button, _on_choice_next_pressed)
	_connect_button_pressed(_target_confirm_button, _on_target_confirm_pressed)
	_connect_button_pressed(_target_cancel_button, _on_target_cancel_pressed)
	_hide_all_action_button_highlights(false)
	close_choice_list()
	set_target_selection_mode(false)

func set_buttons_disabled(disabled: bool) -> void:
	for button: Button in _buttons:
		if button == null:
			continue
		button.disabled = disabled
		button.modulate = Color(1, 1, 1, 0.55) if disabled else Color.WHITE
		_sync_action_button_highlight(button)
	if disabled and not _target_selection_active:
		close_choice_list()


## 自动战斗开启后仅保留“自动”按钮可点，便于再次点击关闭托管。
func set_auto_mode_active(active: bool) -> void:
	if not active:
		return
	close_choice_list()
	set_target_selection_mode(false)
	for button: Button in _buttons:
		if button == null:
			continue
		var auto_button: Button = _buttons[3] if _buttons.size() > 3 else null
		var is_auto_button: bool = button == auto_button
		button.disabled = not is_auto_button
		button.modulate = Color.WHITE if is_auto_button else Color(1, 1, 1, 0.55)
		_sync_action_button_highlight(button)

func set_target_selection_mode(enabled: bool) -> void:
	_target_selection_active = enabled
	set_buttons_disabled(enabled)
	if _target_action_bar != null:
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
	_hide_choice_buttons()
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
	_hide_choice_buttons()
	var page_count: int = _get_choice_page_count()
	if page_count <= 0:
		return
	_choice_page = clampi(_choice_page, 0, page_count - 1)
	var start_index: int = _choice_page * CHOICE_PAGE_SIZE
	var end_index: int = min(start_index + CHOICE_PAGE_SIZE, _choice_entries.size())
	for index: int in range(start_index, end_index):
		var entry: Dictionary = _choice_entries[index]
		var slot_index: int = index - start_index
		if slot_index < 0 or slot_index >= _choice_buttons.size():
			continue
		var button: Button = _choice_buttons[slot_index]
		button.text = str(entry.get("display_name", "选项"))
		button.visible = true
		button.disabled = false
	_update_choice_page_controls(page_count)

func _get_choice_page_count() -> int:
	if _choice_entries.is_empty():
		return 0
	return int(ceil(float(_choice_entries.size()) / float(CHOICE_PAGE_SIZE)))

func _update_choice_page_controls(page_count: int) -> void:
	var show_pager: bool = page_count > 1
	if _choice_page_bar == null:
		return
	_choice_page_bar.visible = show_pager
	if not show_pager:
		return
	if _choice_page_label != null:
		_choice_page_label.text = "%d / %d" % [_choice_page + 1, page_count]
	if _choice_prev_button != null:
		_choice_prev_button.disabled = _choice_page <= 0
	if _choice_next_button != null:
		_choice_next_button.disabled = _choice_page >= page_count - 1

## 绑定场景里预先放好的选择按钮；按钮顺序就是每页展示顺序。
func _connect_choice_buttons() -> void:
	for slot_index: int in range(_choice_buttons.size()):
		var button: Button = _choice_buttons[slot_index]
		if button == null:
			continue
		if not button.pressed.is_connected(_on_choice_button_pressed.bind(slot_index)):
			button.pressed.connect(_on_choice_button_pressed.bind(slot_index))


## 隐藏所有预置选择按钮，后续刷新页面时只显示有数据的按钮。
func _hide_choice_buttons() -> void:
	for button: Button in _choice_buttons:
		if button == null:
			continue
		button.visible = false
		button.disabled = true


## 处理预置按钮点击，并按当前页码换算回完整选项列表下标。
func _on_choice_button_pressed(slot_index: int) -> void:
	var choice_index: int = _choice_page * CHOICE_PAGE_SIZE + slot_index
	_on_choice_entry_pressed(choice_index)

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
	if not _target_selection_active:
		return
	if _target_confirm_button != null and _target_confirm_button.disabled:
		return
	target_selection_confirmed.emit()

func _emit_action(action_type: String) -> void:
	if not _target_selection_active:
		close_choice_list()
	_hide_all_action_button_highlights()
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
		if button == null:
			continue
		for connection: Dictionary in button.pressed.get_connections():
			var callable_value: Callable = connection["callable"] as Callable
			if button.pressed.is_connected(callable_value):
				button.pressed.disconnect(callable_value)

func _reconnect_action_buttons() -> void:
	_disconnect_action_buttons()
	if _buttons.size() > 0:
		_connect_button_pressed(_buttons[0], _on_attack_pressed)
	if _buttons.size() > 1:
		_connect_button_pressed(_buttons[1], _on_skill_pressed)
	if _buttons.size() > 2:
		_connect_button_pressed(_buttons[2], _on_item_pressed)
	if _buttons.size() > 3:
		_connect_button_pressed(_buttons[3], _on_auto_pressed)
	if _buttons.size() > 4:
		_connect_button_pressed(_buttons[4], _on_escape_pressed)

func _restore_action_buttons() -> void:
	for index: int in range(_buttons.size()):
		var button: Button = _buttons[index]
		if button == null:
			continue
		button.visible = true
		button.disabled = false
		button.modulate = Color.WHITE
		if index < _default_button_texts.size():
			button.text = _default_button_texts[index]
		_sync_action_button_highlight(button)

## 安全绑定按钮 pressed 信号；场景调节点时即使暂时缺节点，也不会因空实例中断初始化。
func _connect_button_pressed(button: BaseButton, handler: Callable) -> void:
	if button == null:
		return
	if not button.pressed.is_connected(handler):
		button.pressed.connect(handler)


## 缓存每个主操作按钮的第一张 TextureRect，作为该按钮的状态底图。
func _cache_action_button_highlight_images() -> void:
	_button_highlight_images.clear()
	_button_pressing_states.clear()
	_button_highlight_suppressed.clear()
	for button: Button in _buttons:
		if button == null:
			continue
		var highlight_image: TextureRect = _resolve_button_highlight_image(button)
		if highlight_image == null:
			continue
		_button_highlight_images[button] = highlight_image
		_button_pressing_states[button] = false
		_button_highlight_suppressed[button] = false
		highlight_image.visible = false


## 找到按钮直属的第一张 TextureRect；它是图标下面的状态底图，子 TextureRect 保持图标常显。
func _resolve_button_highlight_image(button: Button) -> TextureRect:
	if button == null:
		return null
	for child: Node in button.get_children():
		if child is TextureRect:
			return child as TextureRect
	return null


## 连接按钮视觉状态信号，让悬停、聚焦、按下都能刷新底图显隐。
func _connect_action_button_state_signals() -> void:
	for button: Button in _buttons:
		if button == null:
			continue
		var visual_handler: Callable = _on_action_button_visual_state_changed.bind(button)
		if not button.mouse_entered.is_connected(visual_handler):
			button.mouse_entered.connect(visual_handler)
		if not button.mouse_exited.is_connected(visual_handler):
			button.mouse_exited.connect(visual_handler)
		if not button.focus_entered.is_connected(visual_handler):
			button.focus_entered.connect(visual_handler)
		if not button.focus_exited.is_connected(visual_handler):
			button.focus_exited.connect(visual_handler)
		var down_handler: Callable = _on_action_button_down.bind(button)
		if not button.button_down.is_connected(down_handler):
			button.button_down.connect(down_handler)
		var up_handler: Callable = _on_action_button_up.bind(button)
		if not button.button_up.is_connected(up_handler):
			button.button_up.connect(up_handler)


## 鼠标悬停或键盘/手柄聚焦变化时刷新按钮底图。
func _on_action_button_visual_state_changed(button: Button) -> void:
	if button == null:
		return
	if not button.is_hovered() and not button.has_focus():
		_button_highlight_suppressed[button] = false
	_sync_action_button_highlight(button)


## 按下主操作按钮时立即显示底图，给移动端触摸一个明确反馈。
func _on_action_button_down(button: Button) -> void:
	if button == null:
		return
	_button_pressing_states[button] = true
	_button_highlight_suppressed[button] = false
	_sync_action_button_highlight(button)


## 松开主操作按钮后按当前悬停/聚焦状态决定是否继续显示底图。
func _on_action_button_up(button: Button) -> void:
	if button == null:
		return
	_button_pressing_states[button] = false
	_sync_action_button_highlight(button)


## 按按钮当前状态同步底图；正常状态隐藏，悬停/聚焦/按下状态显示。
func _sync_action_button_highlight(button: Button) -> void:
	if button == null:
		return
	if not _button_highlight_images.has(button):
		return
	var highlight_image: TextureRect = _button_highlight_images[button] as TextureRect
	if highlight_image == null:
		return
	var is_pressing: bool = bool(_button_pressing_states.get(button, false))
	var is_suppressed: bool = bool(_button_highlight_suppressed.get(button, false))
	highlight_image.visible = button.visible and not button.disabled and not is_suppressed and (button.is_hovered() or button.has_focus() or is_pressing)


## 清掉所有按钮底图和焦点；选择动作后会临时抑制悬停残留，初始化时则不抑制。
func _hide_all_action_button_highlights(suppress_hover: bool = true) -> void:
	for button: Button in _buttons:
		if button == null:
			continue
		_button_pressing_states[button] = false
		_button_highlight_suppressed[button] = suppress_hover
		if button.has_focus():
			button.release_focus()
		if _button_highlight_images.has(button):
			var highlight_image: TextureRect = _button_highlight_images[button] as TextureRect
			if highlight_image != null:
				highlight_image.visible = false
