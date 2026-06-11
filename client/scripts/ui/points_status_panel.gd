extends PanelContainer

const StatusPanelDataProvider = preload("res://scripts/data/status_panel_data_provider.gd")
const UiFormat = preload("res://scripts/common/ui_format.gd")

var _default_data: Dictionary = {}

@onready var free_points_value: Label = get_node_or_null("PointsMargin/PointsVBox/FreePointsRow/FreePointsValue")
@onready var hint_text: Label = get_node_or_null("PointsMargin/PointsVBox/PointHint")
@onready var strength_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid1/StrengthBox/StrengthValue")
@onready var vitality_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid1/VitalityBox/VitalityValue")
@onready var agility_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid2/AgilityBox/AgilityValue")
@onready var mind_value: Label = get_node_or_null("PointsMargin/PointsVBox/PointsGrid2/MindBox/MindValue")
@onready var advice_text: Label = get_node_or_null("PointsMargin/PointsVBox/PointAdviceRow/PointAdviceText")
@onready var plan_text: Label = get_node_or_null("PointsMargin/PointsVBox/PointPlanRow/PointPlanText")


func _ready() -> void:
	_default_data = StatusPanelDataProvider.get_section("points")
	apply_data(_default_data)


func apply_data(data: Dictionary) -> void:
	var resolved := _default_data.duplicate(true)
	for key in data.keys():
		resolved[key] = UiFormat.value_to_text(data[key])

	_set_label_text(free_points_value, UiFormat.value_to_text(resolved.get("free_points", "")))
	_set_label_text(hint_text, UiFormat.value_to_text(resolved.get("hint", "")))
	_set_label_text(strength_value, UiFormat.value_to_text(resolved.get("strength", "")))
	_set_label_text(vitality_value, UiFormat.value_to_text(resolved.get("vitality", "")))
	_set_label_text(agility_value, UiFormat.value_to_text(resolved.get("agility", "")))
	_set_label_text(mind_value, UiFormat.value_to_text(resolved.get("mind", "")))
	_set_label_text(advice_text, UiFormat.value_to_text(resolved.get("advice", "")))
	_set_label_text(plan_text, UiFormat.value_to_text(resolved.get("plan", "")))


func _set_label_text(label: Label, value: String) -> void:
	if label == null:
		return
	label.text = value
