extends PanelContainer

const StatusPanelDataProvider = preload("res://scripts/data/status_panel_data_provider.gd")

var _default_data: Dictionary = {}

@onready var stage_value: Label = get_node_or_null("TransferMargin/TransferVBox/TransferCurrentRow/TransferCurrentValue")
@onready var progress_value: Label = get_node_or_null("TransferMargin/TransferVBox/TransferProgressRow/TransferProgressValue")
@onready var progress_rate: Label = get_node_or_null("TransferMargin/TransferVBox/TransferProgressRow/TransferProgressRate")
@onready var need_value: Label = get_node_or_null("TransferMargin/TransferVBox/TransferNeedRow/TransferNeedValue")
@onready var attack_bonus_value: Label = get_node_or_null("TransferMargin/TransferVBox/TransferBonusGrid/TransferBonusAtkBox/TransferBonusAtkValue")
@onready var hp_bonus_value: Label = get_node_or_null("TransferMargin/TransferVBox/TransferBonusGrid/TransferBonusHpBox/TransferBonusHpValue")
@onready var next_stage_value: Label = get_node_or_null("TransferMargin/TransferVBox/TransferNextRow/TransferNextValue")
@onready var desc_text: Label = get_node_or_null("TransferMargin/TransferVBox/TransferDescRow/TransferDescText")


func _ready() -> void:
	_default_data = StatusPanelDataProvider.get_section("transfer")
	apply_data(_default_data)


func apply_data(data: Dictionary) -> void:
	var resolved := _default_data.duplicate(true)
	for key in data.keys():
		resolved[key] = UiFormat.value_to_text(data[key])

	_set_label_text(stage_value, UiFormat.value_to_text(resolved.get("stage", "")))
	_set_label_text(progress_value, UiFormat.value_to_text(resolved.get("progress_value", "")))
	_set_label_text(progress_rate, UiFormat.value_to_text(resolved.get("progress_rate", "")))
	_set_label_text(need_value, UiFormat.value_to_text(resolved.get("need", "")))
	_set_label_text(attack_bonus_value, UiFormat.value_to_text(resolved.get("attack_bonus", "")))
	_set_label_text(hp_bonus_value, UiFormat.value_to_text(resolved.get("hp_bonus", "")))
	_set_label_text(next_stage_value, UiFormat.value_to_text(resolved.get("next_stage", "")))
	_set_label_text(desc_text, UiFormat.value_to_text(resolved.get("desc", "")))


func _set_label_text(label: Label, value: String) -> void:
	if label == null:
		return
	label.text = value
