extends PanelContainer

const StatusPanelDataProvider = preload("res://scripts/data/status_panel_data_provider.gd")
const RequestLoadingOverlay = preload("res://scripts/ui/request_loading_overlay.gd")
const PetSkillPanel = preload("res://scripts/ui/pet_skill_panel.gd")

var _default_data: Dictionary = {}
## 当前选中的宠物实例唯一标识。
var _selected_pet_uid: int = 0
## 正在等待服务端回包的请求序列号。
var _loading_request_seq: int = 0
## 通用 loading 遮罩。
var _request_loading: RequestLoadingOverlay = null
## 宠物技能详情浮层。
var _pet_skill_panel: PetSkillPanel = null
## 宠物切换下拉框。
var _pet_selector: OptionButton = null
## 五项 +1 按钮引用。
var _allocate_buttons: Array[Button] = []

@onready var pet_name: Label = $PetMargin/PetVBox/PetHeaderRow/PetName
@onready var pet_mood: Label = $PetMargin/PetVBox/PetHeaderRow/PetMood
@onready var pet_level: Label = $PetMargin/PetVBox/PetHeaderRow/PetLevel
@onready var pet_vbox: VBoxContainer = $PetMargin/PetVBox
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
@onready var hp_row: HBoxContainer = $PetMargin/PetVBox/PetHpRow
@onready var mp_row: HBoxContainer = $PetMargin/PetVBox/PetMpRow
@onready var atk_box: HBoxContainer = $PetMargin/PetVBox/PetGrid1/PetAtkBox
@onready var def_box: HBoxContainer = $PetMargin/PetVBox/PetGrid1/PetDefBox
@onready var speed_box: HBoxContainer = $PetMargin/PetVBox/PetGrid2/PetSpeedBox

var _free_points_value: Label = null
var _exp_value: Label = null


func _ready() -> void:
	_default_data = StatusPanelDataProvider.get_section("pet")
	_build_pet_selector()
	_build_skill_button()
	_build_progress_rows()
	_build_request_loading()
	_build_allocate_buttons()
	if not GameState.pets_changed.is_connected(_on_pets_changed):
		GameState.pets_changed.connect(_on_pets_changed)
	apply_data(_default_data)
	refresh_from_game_state()


func _exit_tree() -> void:
	if GameState.pets_changed.is_connected(_on_pets_changed):
		GameState.pets_changed.disconnect(_on_pets_changed)


## 处理 GameState 宠物快照变化通知；pets_changed 当前不携带参数。
func _on_pets_changed() -> void:
	refresh_from_game_state()


func apply_data(data: Dictionary) -> void:
	var resolved: Dictionary = _default_data.duplicate(true)
	for key: Variant in data.keys():
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


func refresh_from_game_state() -> void:
	_rebuild_pet_selector_options()
	var pet: Dictionary = _selected_pet_snapshot()
	if pet.is_empty():
		hint_text.text = "暂无宠物数据，请先打开队伍页同步服务端列表。"
		_set_allocate_buttons_disabled(true)
		return

	var pet_id: int = int(pet.get("pet_id", 0))
	pet_name.text = "宠物 %d" % pet_id
	pet_mood.text = "出战" if bool(pet.get("in_lineup", false)) else ""
	var level: int = int(pet.get("level", 1))
	pet_level.text = "Lv.%s" % UiFormat.value_to_text(level)

	var exp_value: int = int(pet.get("exp", 0))
	var exp_to_next: int = int(pet.get("exp_to_next", 0))
	if _exp_value != null:
		if exp_to_next > 0:
			_exp_value.text = "%s / %s" % [UiFormat.value_to_text(exp_value), UiFormat.value_to_text(exp_to_next)]
		elif level >= 100:
			_exp_value.text = "满级"
		else:
			_exp_value.text = UiFormat.value_to_text(exp_value)

	var free_points: int = int(pet.get("free_attr_points", 0))
	if _free_points_value != null:
		_free_points_value.text = UiFormat.value_to_text(free_points)

	hp_value.text = "%s / %s" % [
		UiFormat.value_to_text(pet.get("hp", 0)),
		UiFormat.value_to_text(pet.get("hp_max", 0)),
	]
	mp_value.text = UiFormat.value_to_text(pet.get("mana", 0))
	attack_value.text = _format_stat_with_alloc(int(pet.get("atk", 0)), int(pet.get("alloc_atk_points", 0)))
	defense_value.text = _format_stat_with_alloc(int(pet.get("def", 0)), int(pet.get("alloc_def_points", 0)))
	speed_value.text = _format_stat_with_alloc(int(pet.get("spd", 0)), int(pet.get("alloc_spd_points", 0)))
	affinity_value.text = _format_stat_with_alloc(int(pet.get("mana", 0)), int(pet.get("alloc_mana_points", 0)))
	hit_value.text = "生命 +%s" % UiFormat.value_to_text(pet.get("alloc_hp_points", 0))
	dodge_value.text = "系统点 ATK %s" % UiFormat.value_to_text(pet.get("auto_atk_points", 0))
	element_value.text = _format_growth_aptitudes(pet)
	hint_text.text = "点击 +1 分配宠物自由属性点，战斗属性由服务端重算。"
	_set_allocate_buttons_disabled(free_points <= 0 or _loading_request_seq != 0)


func _format_stat_with_alloc(stat_value: int, alloc_points: int) -> String:
	if alloc_points > 0:
		return "%s (+%s)" % [UiFormat.value_to_text(stat_value), UiFormat.value_to_text(alloc_points)]
	return UiFormat.value_to_text(stat_value)


func _format_growth_aptitudes(pet: Dictionary) -> String:
	var growth_variant: Variant = pet.get("growth_aptitudes", {})
	if growth_variant is Dictionary:
		var growth: Dictionary = growth_variant as Dictionary
		return "攻资 %s / 血资 %s" % [
			UiFormat.value_to_text(growth.get("atk_apt", 0)),
			UiFormat.value_to_text(growth.get("hp_apt", 0)),
		]
	return "攻资 %s / 血资 %s" % [
		UiFormat.value_to_text(int(pet.get("base_atk_apt", 0)) + int(pet.get("extra_atk_apt", 0))),
		UiFormat.value_to_text(int(pet.get("base_hp_apt", 0)) + int(pet.get("extra_hp_apt", 0))),
	]


func _build_skill_button() -> void:
	if pet_vbox == null:
		return
	var skill_button: Button = Button.new()
	skill_button.name = "PetSkillButton"
	skill_button.text = "查看技能"
	skill_button.custom_minimum_size = Vector2(0, 32)
	skill_button.pressed.connect(_on_open_skill_panel_pressed)
	pet_vbox.add_child(skill_button)
	if _pet_selector != null:
		pet_vbox.move_child(skill_button, _pet_selector.get_index() + 1)


func _on_open_skill_panel_pressed() -> void:
	if _selected_pet_uid <= 0:
		return
	if _pet_skill_panel == null:
		_pet_skill_panel = PetSkillPanel.new()
		add_child(_pet_skill_panel)
	_pet_skill_panel.open_for_pet(_selected_pet_uid)


func _build_pet_selector() -> void:
	if pet_vbox == null:
		return
	_pet_selector = OptionButton.new()
	_pet_selector.name = "PetSelector"
	_pet_selector.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_pet_selector.item_selected.connect(_on_pet_selector_changed)
	pet_vbox.add_child(_pet_selector)
	pet_vbox.move_child(_pet_selector, 0)


func _build_progress_rows() -> void:
	if pet_vbox == null:
		return
	var divider_index: int = pet_vbox.get_node("PetDivider").get_index() if pet_vbox.has_node("PetDivider") else 1

	var free_row: HBoxContainer = HBoxContainer.new()
	free_row.name = "PetFreePointsRow"
	free_row.add_theme_constant_override("separation", 6)
	var free_icon: Label = Label.new()
	free_icon.text = "✨"
	free_row.add_child(free_icon)
	var free_key: Label = Label.new()
	free_key.text = "自由点数:"
	free_row.add_child(free_key)
	_free_points_value = Label.new()
	_free_points_value.name = "PetFreePointsValue"
	_free_points_value.text = "0"
	free_row.add_child(_free_points_value)
	pet_vbox.add_child(free_row)
	pet_vbox.move_child(free_row, divider_index)

	var exp_row: HBoxContainer = HBoxContainer.new()
	exp_row.name = "PetExpRow"
	exp_row.add_theme_constant_override("separation", 6)
	var exp_icon: Label = Label.new()
	exp_icon.text = "⭐"
	exp_row.add_child(exp_icon)
	var exp_key: Label = Label.new()
	exp_key.text = "经验:"
	exp_row.add_child(exp_key)
	_exp_value = Label.new()
	_exp_value.name = "PetExpValue"
	_exp_value.text = "0"
	exp_row.add_child(_exp_value)
	pet_vbox.add_child(exp_row)
	pet_vbox.move_child(exp_row, divider_index + 1)


func _build_request_loading() -> void:
	_request_loading = RequestLoadingOverlay.new()
	_request_loading.name = "PetAllocateLoadingOverlay"
	add_child(_request_loading)


func _build_allocate_buttons() -> void:
	_attach_allocate_button(hp_row, Callable(self, "_on_allocate_hp_pressed"))
	_attach_allocate_button(atk_box, Callable(self, "_on_allocate_atk_pressed"))
	_attach_allocate_button(def_box, Callable(self, "_on_allocate_def_pressed"))
	_attach_allocate_button(speed_box, Callable(self, "_on_allocate_spd_pressed"))
	_attach_allocate_button(mp_row, Callable(self, "_on_allocate_mana_pressed"))


func _attach_allocate_button(container: HBoxContainer, callback: Callable) -> void:
	if container == null:
		return
	if container.get_node_or_null("AllocateButton") != null:
		return
	var button: Button = Button.new()
	button.name = "AllocateButton"
	button.text = "+1"
	button.custom_minimum_size = Vector2(44, 28)
	button.pressed.connect(callback)
	container.add_child(button)
	_allocate_buttons.append(button)


func _rebuild_pet_selector_options() -> void:
	if _pet_selector == null:
		return
	var previous_uid: int = _selected_pet_uid
	_pet_selector.clear()
	var selected_index: int = 0
	for index: int in range(GameState.pets.size()):
		var pet_variant: Variant = GameState.pets[index]
		if pet_variant is not Dictionary:
			continue
		var pet: Dictionary = pet_variant as Dictionary
		var pet_uid: int = int(pet.get("pet_uid", 0))
		if pet_uid == 0:
			continue
		var label: String = "宠物 %d Lv.%d" % [int(pet.get("pet_id", 0)), int(pet.get("level", 1))]
		if bool(pet.get("in_lineup", false)):
			label += " [出战]"
		_pet_selector.add_item(label, pet_uid)
		if previous_uid != 0 and pet_uid == previous_uid:
			selected_index = _pet_selector.item_count - 1
	if _pet_selector.item_count == 0:
		_selected_pet_uid = 0
		return
	if previous_uid == 0:
		selected_index = 0
	_pet_selector.select(selected_index)
	_selected_pet_uid = int(_pet_selector.get_item_id(selected_index))


func _on_pet_selector_changed(index: int) -> void:
	if _pet_selector == null or index < 0:
		return
	_selected_pet_uid = int(_pet_selector.get_item_id(index))
	refresh_from_game_state()


func _selected_pet_snapshot() -> Dictionary:
	if _selected_pet_uid == 0:
		if not GameState.pets.is_empty() and GameState.pets[0] is Dictionary:
			return (GameState.pets[0] as Dictionary).duplicate(true)
		return {}
	for pet_variant: Variant in GameState.pets:
		if pet_variant is Dictionary and int((pet_variant as Dictionary).get("pet_uid", 0)) == _selected_pet_uid:
			return (pet_variant as Dictionary).duplicate(true)
	return {}


func _on_allocate_hp_pressed() -> void:
	_request_allocate_attr_points(1, 0, 0, 0, 0)


func _on_allocate_atk_pressed() -> void:
	_request_allocate_attr_points(0, 1, 0, 0, 0)


func _on_allocate_spd_pressed() -> void:
	_request_allocate_attr_points(0, 0, 1, 0, 0)


func _on_allocate_mana_pressed() -> void:
	_request_allocate_attr_points(0, 0, 0, 1, 0)


func _on_allocate_def_pressed() -> void:
	_request_allocate_attr_points(0, 0, 0, 0, 1)


func _request_allocate_attr_points(hp: int, atk: int, spd: int, mana: int, def_value: int) -> void:
	if _loading_request_seq != 0:
		return
	if not GameState.is_ws_authenticated:
		return
	if _selected_pet_uid == 0:
		return
	var request_seq: int = App.request_pet_allocate_attr_points(_selected_pet_uid, hp, atk, spd, mana, def_value)
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	_set_allocate_buttons_disabled(true)
	if _request_loading != null:
		_request_loading.show_waiting("正在分配宠物属性点")
	call_deferred("_wait_allocate_request", request_seq)


func _wait_allocate_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PET_ALLOCATE_ATTR_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	if _request_loading != null:
		_request_loading.hide_overlay()
	refresh_from_game_state()


func _set_allocate_buttons_disabled(disabled: bool) -> void:
	for button: Button in _allocate_buttons:
		if button != null:
			button.disabled = disabled
