extends PanelContainer

const StatusPanelDataProvider = preload("res://scripts/data/status_panel_data_provider.gd")

var _default_data: Dictionary = {}

@onready var hp_value: Label = $BodyMargin/BodyVBox/HpRow/HpValue
@onready var element_value: Label = $BodyMargin/BodyVBox/ElementRow/ElementValue
@onready var energy_value: Label = $BodyMargin/BodyVBox/EnergyRow/EnergyValue
@onready var exp_value: Label = $BodyMargin/BodyVBox/ExpGuardGrid/ExpBox/ExpValue
@onready var guard_value: Label = $BodyMargin/BodyVBox/ExpGuardGrid/GuardBox/GuardValue
@onready var fly_value: Label = $BodyMargin/BodyVBox/FlyAscendRow/FlyAscendValue
@onready var fly_rate: Label = $BodyMargin/BodyVBox/FlyAscendRow/FlyAscendRate
@onready var transfer_value: Label = $BodyMargin/BodyVBox/TransferRow/TransferValue
@onready var transfer_rate: Label = $BodyMargin/BodyVBox/TransferRow/TransferRate
@onready var attack_value: Label = $BodyMargin/BodyVBox/CombatGrid1/AtkBox/AtkValue
@onready var defense_value: Label = $BodyMargin/BodyVBox/CombatGrid1/DefBox/DefValue
@onready var speed_value: Label = $BodyMargin/BodyVBox/CombatGrid2/SpeedBox/SpeedValue
@onready var mana_value: Label = $BodyMargin/BodyVBox/CombatGrid2/ManaBox/ManaValue
@onready var hit_value: Label = $BodyMargin/BodyVBox/CombatGrid3/HitBox/HitValue
@onready var dodge_value: Label = $BodyMargin/BodyVBox/CombatGrid3/DodgeBox/DodgeValue
@onready var crit_value: Label = $BodyMargin/BodyVBox/CombatGrid4/CritBox/CritValue
@onready var crit_damage_value: Label = $BodyMargin/BodyVBox/CombatGrid4/CritDmgBox/CritDmgValue


func _ready() -> void:
	_default_data = StatusPanelDataProvider.get_section("character")
	apply_data(_default_data)


func apply_data(data: Dictionary) -> void:
	var resolved := _default_data.duplicate(true)
	for key in data.keys():
		resolved[key] = str(data[key])

	hp_value.text = str(resolved.get("hp", ""))
	element_value.text = str(resolved.get("element", ""))
	energy_value.text = str(resolved.get("energy", ""))
	exp_value.text = str(resolved.get("exp", ""))
	guard_value.text = str(resolved.get("guard_level", ""))
	fly_value.text = str(resolved.get("fly_value", ""))
	fly_rate.text = str(resolved.get("fly_rate", ""))
	transfer_value.text = str(resolved.get("transfer_value", ""))
	transfer_rate.text = str(resolved.get("transfer_rate", ""))
	attack_value.text = str(resolved.get("attack", ""))
	defense_value.text = str(resolved.get("defense", ""))
	speed_value.text = str(resolved.get("speed", ""))
	mana_value.text = str(resolved.get("mana", ""))
	hit_value.text = str(resolved.get("hit", ""))
	dodge_value.text = str(resolved.get("dodge", ""))
	crit_value.text = str(resolved.get("crit", ""))
	crit_damage_value.text = str(resolved.get("crit_damage", ""))
