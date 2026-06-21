extends PanelContainer

const StatusPanelDataProvider = preload("res://scripts/data/status_panel_data_provider.gd")
const RequestLoadingOverlay = preload("res://scripts/ui/request_loading_overlay.gd")

var _default_data: Dictionary = {}
## 已佩戴装备列表容器。
var _equipment_container: VBoxContainer = null
## 通用 loading 遮罩。
var _request_loading: RequestLoadingOverlay = null
## 正在等待服务端回包的请求序列号。
var _loading_request_seq: int = 0

@onready var hp_value: Label = $BodyMargin/BodyVBox/HpRow/HpValue
@onready var element_value: Label = $BodyMargin/BodyVBox/ElementRow/ElementValue
@onready var vigor_value: Label = $BodyMargin/BodyVBox/EnergyRow/EnergyValue
@onready var spirit_value: Label = $BodyMargin/BodyVBox/SpiritRow/SpiritValue
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
@onready var body_vbox: VBoxContainer = $BodyMargin/BodyVBox


func _ready() -> void:
	_default_data = StatusPanelDataProvider.get_section("character")
	_build_equipment_section()
	_build_request_loading()
	if not GameState.equipment_changed.is_connected(_refresh_equipment_section):
		GameState.equipment_changed.connect(_refresh_equipment_section)
	if not GameState.world_snapshot_changed.is_connected(_on_world_snapshot_changed):
		GameState.world_snapshot_changed.connect(_on_world_snapshot_changed)
	apply_data(_default_data)
	_refresh_equipment_section()


func _exit_tree() -> void:
	if GameState.equipment_changed.is_connected(_refresh_equipment_section):
		GameState.equipment_changed.disconnect(_refresh_equipment_section)
	if GameState.world_snapshot_changed.is_connected(_on_world_snapshot_changed):
		GameState.world_snapshot_changed.disconnect(_on_world_snapshot_changed)


func apply_data(data: Dictionary) -> void:
	var resolved := _default_data.duplicate(true)
	for key in data.keys():
		resolved[key] = UiFormat.value_to_text(data[key])

	hp_value.text = UiFormat.value_to_text(resolved.get("hp", ""))
	element_value.text = UiFormat.value_to_text(resolved.get("element", ""))
	vigor_value.text = UiFormat.value_to_text(resolved.get("vigor", ""))
	spirit_value.text = UiFormat.value_to_text(resolved.get("spirit", ""))
	exp_value.text = UiFormat.value_to_text(resolved.get("exp", ""))
	guard_value.text = UiFormat.value_to_text(resolved.get("guard_level", ""))
	fly_value.text = UiFormat.value_to_text(resolved.get("fly_value", ""))
	fly_rate.text = UiFormat.value_to_text(resolved.get("fly_rate", ""))
	transfer_value.text = UiFormat.value_to_text(resolved.get("transfer_value", ""))
	transfer_rate.text = UiFormat.value_to_text(resolved.get("transfer_rate", ""))
	attack_value.text = UiFormat.value_to_text(resolved.get("attack", ""))
	defense_value.text = UiFormat.value_to_text(resolved.get("defense", ""))
	speed_value.text = UiFormat.value_to_text(resolved.get("speed", ""))
	mana_value.text = UiFormat.value_to_text(resolved.get("mana", ""))
	hit_value.text = UiFormat.value_to_text(resolved.get("hit", ""))
	dodge_value.text = UiFormat.value_to_text(resolved.get("dodge", ""))
	crit_value.text = UiFormat.value_to_text(resolved.get("crit", ""))
	crit_damage_value.text = UiFormat.value_to_text(resolved.get("crit_damage", ""))


## 面板可见时拉取最新已佩戴装备。
func refresh_equipment_from_server() -> void:
	if not visible or not GameState.is_ws_authenticated:
		return
	if _loading_request_seq != 0:
		return
	var request_seq: int = App.request_player_equipment_list()
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	if _request_loading != null:
		_request_loading.show_waiting("正在获取已佩戴装备")
	call_deferred("_wait_equipment_list_request", request_seq)


func _on_world_snapshot_changed() -> void:
	_refresh_equipment_section()


func _build_equipment_section() -> void:
	if body_vbox == null:
		return
	_equipment_container = VBoxContainer.new()
	_equipment_container.name = "EquipmentSection"
	_equipment_container.add_theme_constant_override("separation", 4)
	body_vbox.add_child(_equipment_container)


func _build_request_loading() -> void:
	_request_loading = RequestLoadingOverlay.new()
	_request_loading.name = "CharacterEquipmentLoadingOverlay"
	add_child(_request_loading)


func _refresh_equipment_section() -> void:
	if _equipment_container == null:
		return
	for child_index: int in range(_equipment_container.get_child_count() - 1, -1, -1):
		var child: Node = _equipment_container.get_child(child_index)
		_equipment_container.remove_child(child)
		child.queue_free()

	var title_label: Label = Label.new()
	title_label.text = "已佩戴装备"
	_equipment_container.add_child(title_label)

	if GameState.equipped_items.is_empty():
		var empty_label: Label = Label.new()
		empty_label.text = "当前未佩戴任何装备。"
		_equipment_container.add_child(empty_label)
		return

	for item_variant: Variant in GameState.equipped_items:
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		var row: HBoxContainer = HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)

		var slot_label: String = str(item.get("equip_slot_label", item.get("equip_slot", "")))
		var item_name: String = str(item.get("item_name", "未知装备"))
		var enhance_level: int = int(item.get("enhance_level", 0))
		var summary_label: Label = Label.new()
		summary_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		if enhance_level > 0:
			summary_label.text = "%s · %s +%d" % [slot_label, item_name, enhance_level]
		else:
			summary_label.text = "%s · %s" % [slot_label, item_name]
		row.add_child(summary_label)

		var unequip_button: Button = Button.new()
		unequip_button.text = "卸下"
		unequip_button.custom_minimum_size = Vector2(56, 28)
		var equip_slot: String = str(item.get("equip_slot", ""))
		unequip_button.pressed.connect(_on_unequip_pressed.bind(equip_slot))
		row.add_child(unequip_button)

		_equipment_container.add_child(row)


func _on_unequip_pressed(equip_slot: String) -> void:
	if equip_slot.is_empty() or _loading_request_seq != 0:
		return
	if not GameState.is_ws_authenticated:
		return
	var request_seq: int = App.request_player_unequip(equip_slot, "bag")
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	if _request_loading != null:
		_request_loading.show_waiting("正在卸下装备")
	call_deferred("_wait_unequip_request", request_seq)


func _wait_equipment_list_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PLAYER_EQUIPMENT_LIST_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	if _request_loading != null:
		_request_loading.hide_overlay()
	_refresh_equipment_section()


func _wait_unequip_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PLAYER_UNEQUIP_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	if _request_loading != null:
		_request_loading.hide_overlay()
	_refresh_equipment_section()
