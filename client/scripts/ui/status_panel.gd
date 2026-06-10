extends PanelContainer

const TAB_NORMAL_COLOR := Color(0.5019608, 0.62352943, 0.654902, 1)
const TAB_HOVER_COLOR := Color(0.65882355, 0.7607843, 0.7529412, 1)
const TAB_ACTIVE_COLOR := Color(0.8352941, 0.91764706, 0.8509804, 1)

@onready var character_tab_button: Button = $RootVBox/SubTabRow/CharacterTabButton
@onready var pet_tab_button: Button = $RootVBox/SubTabRow/PetTabButton
@onready var points_tab_button: Button = $RootVBox/SubTabRow/PointsTabButton
@onready var transfer_tab_button: Button = $RootVBox/SubTabRow/TransferTabButton

@onready var character_panel: Control = $RootVBox/PanelHost/CharacterStatusPanel
@onready var pet_panel: Control = $RootVBox/PanelHost/PetStatusPanel
@onready var points_panel: Control = $RootVBox/PanelHost/PointsStatusPanel
@onready var transfer_panel: Control = $RootVBox/PanelHost/TransferStatusPanel

@onready var name_label: Label = $RootVBox/HeaderPanel/HeaderMargin/HeaderRow/HeaderLeft/NameLabel
@onready var badge_label: Label = $RootVBox/HeaderPanel/HeaderMargin/HeaderRow/HeaderLeft/BadgeLabel
@onready var level_label: Label = $RootVBox/HeaderPanel/HeaderMargin/HeaderRow/HeaderLeft/LevelLabel
@onready var fly_label: Label = $RootVBox/HeaderPanel/HeaderMargin/HeaderRow/FlyLabel
@onready var tier_label: Label = $RootVBox/HeaderPanel/HeaderMargin/HeaderRow/TierLabel

var _tab_buttons: Array[Button] = []
var _tab_panels: Array[Control] = []
var _current_tab_index: int = 0
var _hovered_tab_index: int = -1
var _player_source: Node = null


func _ready() -> void:
	_tab_buttons = [character_tab_button, pet_tab_button, points_tab_button, transfer_tab_button]
	_tab_panels = [character_panel, pet_panel, points_panel, transfer_panel]

	character_tab_button.pressed.connect(_on_tab_pressed.bind(0))
	pet_tab_button.pressed.connect(_on_tab_pressed.bind(1))
	points_tab_button.pressed.connect(_on_tab_pressed.bind(2))
	transfer_tab_button.pressed.connect(_on_tab_pressed.bind(3))
	character_tab_button.mouse_entered.connect(_on_tab_hovered.bind(0))
	pet_tab_button.mouse_entered.connect(_on_tab_hovered.bind(1))
	points_tab_button.mouse_entered.connect(_on_tab_hovered.bind(2))
	transfer_tab_button.mouse_entered.connect(_on_tab_hovered.bind(3))
	character_tab_button.mouse_exited.connect(_on_tab_hover_exited.bind(0))
	pet_tab_button.mouse_exited.connect(_on_tab_hover_exited.bind(1))
	points_tab_button.mouse_exited.connect(_on_tab_hover_exited.bind(2))
	transfer_tab_button.mouse_exited.connect(_on_tab_hover_exited.bind(3))

	_select_tab(0)


func reset_to_default() -> void:
	_select_tab(0)


func set_player_source(player_source: Node) -> void:
	_player_source = player_source
	refresh_panel_data()


func refresh_panel_data() -> void:
	var data: Dictionary = {}
	if _player_source != null and _player_source.has_method("get_status_panel_data"):
		var panel_data: Variant = _player_source.call("get_status_panel_data")
		if typeof(panel_data) == TYPE_DICTIONARY:
			data = panel_data
	if data.is_empty():
		data = _build_runtime_panel_data()
	_apply_header_data(data.get("header", {}))
	_apply_panel_data(character_panel, data.get("character", {}))
	_apply_panel_data(pet_panel, data.get("pet", {}))
	_apply_panel_data(points_panel, data.get("points", {}))
	_apply_panel_data(transfer_panel, data.get("transfer", {}))


func _on_tab_pressed(index: int) -> void:
	_select_tab(index)


func _on_tab_hovered(index: int) -> void:
	_hovered_tab_index = index
	_refresh_tabs()


func _on_tab_hover_exited(index: int) -> void:
	if _hovered_tab_index == index:
		_hovered_tab_index = -1
	_refresh_tabs()


func _select_tab(index: int) -> void:
	if index < 0 or index >= _tab_buttons.size():
		return

	_current_tab_index = index
	_refresh_tabs()
	for i in range(_tab_panels.size()):
		var panel := _tab_panels[i]
		if panel != null:
			panel.visible = i == _current_tab_index


func _refresh_tabs() -> void:
	for i in range(_tab_buttons.size()):
		var is_current := i == _current_tab_index
		var button := _tab_buttons[i]
		if button != null:
			button.button_pressed = is_current
			if is_current:
				button.modulate = TAB_ACTIVE_COLOR
			elif i == _hovered_tab_index:
				button.modulate = TAB_HOVER_COLOR
			else:
				button.modulate = TAB_NORMAL_COLOR


func _apply_header_data(data: Dictionary) -> void:
	if data.is_empty():
		return
	name_label.text = str(data.get("name", name_label.text))
	badge_label.text = str(data.get("title", badge_label.text))
	level_label.text = str(data.get("level", level_label.text))
	fly_label.text = str(data.get("fly", fly_label.text))
	tier_label.text = str(data.get("tier", tier_label.text))


func _apply_panel_data(panel: Control, data: Dictionary) -> void:
	if panel == null or data.is_empty():
		return
	if panel.has_method("apply_data"):
		panel.call("apply_data", data)


func _build_runtime_panel_data() -> Dictionary:
	var data: Dictionary = StatusPanelDataProvider.get_panel_data()
	var header: Dictionary = data.get("header", {}).duplicate(true)
	var character: Dictionary = data.get("character", {}).duplicate(true)
	var pet: Dictionary = data.get("pet", {}).duplicate(true)
	var default_title: String = str(header.get("title", "称号"))
	var player_title: String = default_title
	if SomeGlobal != null:
		player_title = str(SomeGlobal.player_title)

	header["name"] = str(GameState.player_snapshot.get("name", header.get("name", "玩家名")))
	header["title"] = str(GameState.player_snapshot.get("title", player_title))
	if GameState.player_snapshot.has("level"):
		header["level"] = str(GameState.player_snapshot.get("level", header.get("level", "")))

	if GameState.player_snapshot.has("hp") and GameState.player_snapshot.has("hp_max"):
		character["hp"] = "%s / %s" % [
			str(GameState.player_snapshot.get("hp", 0)),
			str(GameState.player_snapshot.get("hp_max", 0)),
		]

	if not GameState.pets.is_empty():
		var pet_variant: Variant = GameState.pets[0]
		if pet_variant is Dictionary:
			var first_pet: Dictionary = pet_variant
			pet["name"] = str(first_pet.get("name", pet.get("name", "宠物")))
			pet["level"] = "Lv.%s" % str(first_pet.get("level", pet.get("level", "1")))
			pet["hp"] = "%s / %s" % [
				str(first_pet.get("hp", 0)),
				str(first_pet.get("hp_max", 0)),
			]
			if first_pet.has("mp") or first_pet.has("mp_max"):
				pet["mp"] = "%s / %s" % [
					str(first_pet.get("mp", 0)),
					str(first_pet.get("mp_max", 0)),
				]

	data["header"] = header
	data["character"] = character
	data["pet"] = pet
	return data
