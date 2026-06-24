extends CanvasLayer

signal option_selected(option: Dictionary)
signal menu_closed

const ENTRY_TYPE_LABELS: Dictionary = {
	"quest": "[任务]",
	"dialog": "[对话]",
	"shop": "[商店]",
	"battle": "[战斗]",
	"warehouse": "[仓库]",
	"teleport": "[传送]",
	"craft": "[制作]",
	"event": "[事件]",
}

const STATE_LABELS: Dictionary = {
	"locked": "未解锁",
	"completed": "已完成",
	"in_progress": "进行中",
	"available": "",
}

const DISABLED_STATES: Array[String] = ["locked", "completed"]

@onready var panel: PanelContainer = $MarginContainer/PanelContainer
@onready var title_label: Label = $MarginContainer/PanelContainer/VBoxContainer/TitleLabel
@onready var buttons_container: VBoxContainer = $MarginContainer/PanelContainer/VBoxContainer/Buttons


func _ready() -> void:
	hide()


## 根据服务端返回的菜单项列表刷新按钮文案与可点击状态。
func configure(title: String, options: Array[Dictionary]) -> void:
	title_label.text = title

	for child in buttons_container.get_children():
		buttons_container.remove_child(child)
		child.queue_free()

	for option in options:
		var option_data: Dictionary = option.duplicate(true)
		var button: Button = Button.new()
		button.text = _format_option_label(option_data)
		button.alignment = HORIZONTAL_ALIGNMENT_LEFT
		DialogueActionButtonTheme.apply(button, true)
		var entry_state: String = str(option_data.get("state", "available"))
		if DISABLED_STATES.has(entry_state):
			button.disabled = true
		button.pressed.connect(_on_button_pressed.bind(option_data))
		buttons_container.add_child(button)

	var close_button: Button = Button.new()
	close_button.text = "关闭"
	close_button.alignment = HORIZONTAL_ALIGNMENT_LEFT
	DialogueActionButtonTheme.apply(close_button, true)
	close_button.pressed.connect(close_menu)
	buttons_container.add_child(close_button)

	if buttons_container.get_child_count() > 0:
		var first_button := buttons_container.get_child(0) as Button
		if first_button != null:
			first_button.grab_focus()


func open_menu() -> void:
	show()


func close_menu() -> void:
	var was_visible := visible
	hide()
	if was_visible:
		menu_closed.emit()


func _on_button_pressed(option: Dictionary) -> void:
	option_selected.emit(option)


## 把类型前缀、标题、副标题和状态标签拼成移动端可读的按钮文案。
func _format_option_label(option: Dictionary) -> String:
	var entry_type: String = str(option.get("entry_type", ""))
	var title_text: String = str(option.get("title", option.get("label", option.get("id", ""))))
	var subtitle_text: String = str(option.get("subtitle", "")).strip_edges()
	var state_text: String = str(STATE_LABELS.get(str(option.get("state", "available")), ""))
	var type_prefix: String = str(ENTRY_TYPE_LABELS.get(entry_type, "[交互]"))
	var label_text: String = "%s %s" % [type_prefix, title_text]
	if not subtitle_text.is_empty():
		label_text += "\n%s" % subtitle_text
	if not state_text.is_empty():
		label_text += " (%s)" % state_text
	return label_text
