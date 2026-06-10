extends CanvasLayer

signal menu_closed

const TAB_NORMAL_COLOR := Color(1, 1, 1, 1)
const TAB_PRESSED_COLOR := Color(0.95, 0.97, 0.84, 1)

@onready var status_button: Button = $RootPanel/ContentRow/TabButtons/StatusTabButton
@onready var bag_button: Button = $RootPanel/ContentRow/TabButtons/BagTabButton
@onready var team_button: Button = $RootPanel/ContentRow/TabButtons/TeamTabButton
@onready var skill_button: Button = $RootPanel/ContentRow/TabButtons/SkillTabButton

@onready var status_panel: Control = $RootPanel/ContentRow/PanelHost/StatusPanel
@onready var bag_panel: Control = $RootPanel/ContentRow/PanelHost/BagPanel
@onready var team_panel: Control = $RootPanel/ContentRow/PanelHost/TeamPanel
@onready var skill_panel: Control = $RootPanel/ContentRow/PanelHost/SkillPanel

var _tab_buttons: Array[Button] = []
var _tab_panels: Array[Control] = []
var _current_tab_index: int = 0


func _ready() -> void:
	hide()
	_tab_buttons = [status_button, bag_button, team_button, skill_button]
	_tab_panels = [status_panel, bag_panel, team_panel, skill_panel]

	status_button.pressed.connect(_on_tab_pressed.bind(0))
	bag_button.pressed.connect(_on_tab_pressed.bind(1))
	team_button.pressed.connect(_on_tab_pressed.bind(2))
	skill_button.pressed.connect(_on_tab_pressed.bind(3))

	_select_tab(0)


func open_menu() -> void:
	show()
	_select_tab(0)
	if status_panel != null and status_panel.has_method("reset_to_default"):
		status_panel.call("reset_to_default")
	if status_panel != null and status_panel.has_method("refresh_panel_data"):
		status_panel.call("refresh_panel_data")


func set_player_source(player_source: Node) -> void:
	if status_panel != null and status_panel.has_method("set_player_source"):
		status_panel.call("set_player_source", player_source)


func close_menu() -> void:
	var was_visible := visible
	hide()
	if was_visible:
		menu_closed.emit()


func _on_tab_pressed(index: int) -> void:
	_select_tab(index)


func _select_tab(index: int) -> void:
	if index < 0 or index >= _tab_buttons.size():
		return

	_current_tab_index = index
	for i in range(_tab_buttons.size()):
		var is_current := i == _current_tab_index
		var button := _tab_buttons[i]
		var panel := _tab_panels[i]
		if button != null:
			button.button_pressed = is_current
			button.modulate = TAB_PRESSED_COLOR if is_current else TAB_NORMAL_COLOR
		if panel != null:
			panel.visible = is_current
