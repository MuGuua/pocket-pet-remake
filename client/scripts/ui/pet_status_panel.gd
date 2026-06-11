extends PanelContainer

const StatusPanelDataProvider = preload("res://scripts/data/status_panel_data_provider.gd")
const UiFormat = preload("res://scripts/common/ui_format.gd")

var _default_data: Dictionary = {}

@onready var pet_name: Label = $PetMargin/PetVBox/PetHeaderRow/PetName
@onready var pet_mood: Label = $PetMargin/PetVBox/PetHeaderRow/PetMood
@onready var pet_level: Label = $PetMargin/PetVBox/PetHeaderRow/PetLevel
@onready var hp_value: Label = $PetMargin/PetVBox/PetHpRow/PetHpValue
@onready var mp_value: Label = $PetMargin/PetVBox/PetMpRow/PetMpValue
@onready var element_value: Label = $PetMargin/PetVBox/PetElementRow/PetElementValue
@onready var hint_text: Label = $PetMargin/PetVBox/PetHintRow/PetHintText
@onready var attack_value: Label = $PetMargin/PetVBox/PetGrid1/PetAtkBox/PetAtkValue
@onready var defense_value: Label = $PetMargin/PetVBox/PetGrid1/PetDefBox/PetDefValue
@onready var speed_value: Label = $PetMargin/PetVBox/PetGrid2/PetSpeedBox/PetSpeedValue
@onready var affinity_value: Label = $PetMargin/PetVBox/PetGrid2/PetAffinityBox/PetAffinityValue
@onready var hit_value: Label = $PetMargin/PetVBox/PetGrid3/PetHitBox/PetHitValue
@onready var dodge_value: Label = $PetMargin/PetVBox/PetGrid3/PetDodgeBox/PetDodgeValue


func _ready() -> void:
	_default_data = StatusPanelDataProvider.get_section("pet")
	apply_data(_default_data)


func apply_data(data: Dictionary) -> void:
	var resolved := _default_data.duplicate(true)
	for key in data.keys():
		resolved[key] = UiFormat.value_to_text(data[key])

	pet_name.text = UiFormat.value_to_text(resolved.get("name", ""))
	pet_mood.text = UiFormat.value_to_text(resolved.get("mood", ""))
	pet_level.text = UiFormat.value_to_text(resolved.get("level", ""))
	hp_value.text = UiFormat.value_to_text(resolved.get("hp", ""))
	mp_value.text = UiFormat.value_to_text(resolved.get("mp", ""))
	element_value.text = UiFormat.value_to_text(resolved.get("element", ""))
	hint_text.text = UiFormat.value_to_text(resolved.get("hint", ""))
	attack_value.text = UiFormat.value_to_text(resolved.get("attack", ""))
	defense_value.text = UiFormat.value_to_text(resolved.get("defense", ""))
	speed_value.text = UiFormat.value_to_text(resolved.get("speed", ""))
	affinity_value.text = UiFormat.value_to_text(resolved.get("affinity", ""))
	hit_value.text = UiFormat.value_to_text(resolved.get("hit", ""))
	dodge_value.text = UiFormat.value_to_text(resolved.get("dodge", ""))
