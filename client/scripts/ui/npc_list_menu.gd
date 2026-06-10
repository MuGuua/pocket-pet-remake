extends CanvasLayer

signal npc_selected(npc_data: Dictionary)
signal menu_closed

@onready var panel: PanelContainer = $MarginContainer/PanelContainer
@onready var title_label: Label = $MarginContainer/PanelContainer/VBoxContainer/TitleLabel
@onready var buttons_container: VBoxContainer = $MarginContainer/PanelContainer/VBoxContainer/Buttons

var _npc_lookup: Dictionary = {}


func _ready() -> void:
	hide()


func configure(title: String, npcs: Array[Dictionary]) -> void:
	title_label.text = title
	_npc_lookup.clear()

	for child in buttons_container.get_children():
		buttons_container.remove_child(child)
		child.queue_free()

	for index in range(npcs.size()):
		var npc_variant: Variant = npcs[index]
		if npc_variant is not Dictionary:
			continue
		var npc: Dictionary = npc_variant

		var npc_id := str(index)
		var item_container := HBoxContainer.new()
		item_container.alignment = BoxContainer.ALIGNMENT_CENTER
		item_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL

		var portrait_rect := TextureRect.new()
		portrait_rect.custom_minimum_size = Vector2(32.0, 32.0)
		portrait_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
		portrait_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
		portrait_rect.texture = _resolve_npc_portrait(npc)
		item_container.add_child(portrait_rect)

		var button := Button.new()
		button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		button.alignment = HORIZONTAL_ALIGNMENT_LEFT
		button.text = _resolve_npc_name(npc)
		button.focus_mode = Control.FOCUS_ALL
		button.custom_minimum_size = Vector2(0, 30)
		button.add_theme_font_size_override("font_size", 12)
		button.pressed.connect(_on_npc_button_pressed.bind(npc_id))
		item_container.add_child(button)
		buttons_container.add_child(item_container)
		_npc_lookup[npc_id] = npc

	var close_button := Button.new()
	close_button.text = "关闭"
	close_button.focus_mode = Control.FOCUS_ALL
	close_button.custom_minimum_size = Vector2(0, 30)
	close_button.add_theme_font_size_override("font_size", 12)
	close_button.pressed.connect(close_menu)
	buttons_container.add_child(close_button)

	if buttons_container.get_child_count() > 0:
		var first_item := buttons_container.get_child(0)
		if first_item is Button:
			(first_item as Button).grab_focus()
		elif first_item is HBoxContainer and first_item.get_child_count() > 1:
			var first_button := first_item.get_child(1) as Button
			if first_button != null:
				first_button.grab_focus()


func open_menu() -> void:
	show()


func close_menu() -> void:
	var was_visible := visible
	hide()
	if was_visible:
		menu_closed.emit()


func _on_npc_button_pressed(npc_id: String) -> void:
	var npc_variant: Variant = _npc_lookup.get(npc_id, {})
	if npc_variant is not Dictionary:
		return
	npc_selected.emit(npc_variant)


func _resolve_npc_name(npc: Dictionary) -> String:
	return str(npc.get("npc_name", npc.get("name", "未知 NPC")))


func _resolve_npc_portrait(npc: Dictionary) -> Texture2D:
	var portrait_path := str(npc.get("portrait_path", ""))
	if portrait_path.is_empty():
		return null
	var portrait := load(portrait_path)
	return portrait as Texture2D if portrait is Texture2D else null
