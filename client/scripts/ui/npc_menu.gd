extends CanvasLayer

signal option_selected(option: Dictionary)
signal menu_closed

@onready var panel: PanelContainer = $MarginContainer/PanelContainer
@onready var title_label: Label = $MarginContainer/PanelContainer/VBoxContainer/TitleLabel
@onready var buttons_container: VBoxContainer = $MarginContainer/PanelContainer/VBoxContainer/Buttons


func _ready() -> void:
	hide()


func configure(title: String, options: Array[Dictionary]) -> void:
	title_label.text = title

	for child in buttons_container.get_children():
		buttons_container.remove_child(child)
		child.queue_free()

	for option in options:
		var button := Button.new()
		var option_data: Dictionary = option.duplicate(true)
		button.text = str(option_data.get("label", ""))
		button.focus_mode = Control.FOCUS_ALL
		button.custom_minimum_size = Vector2(0, 28)
		button.add_theme_font_size_override("font_size", 12)
		button.pressed.connect(_on_button_pressed.bind(option_data))
		buttons_container.add_child(button)

	var close_button := Button.new()
	close_button.text = "关闭"
	close_button.focus_mode = Control.FOCUS_ALL
	close_button.custom_minimum_size = Vector2(0, 28)
	close_button.add_theme_font_size_override("font_size", 12)
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
