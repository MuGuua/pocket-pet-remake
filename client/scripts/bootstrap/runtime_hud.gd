extends Control
class_name RuntimeHud

const CARD_TITLE_FONT_SIZE := 14
const CARD_TEXT_FONT_SIZE := 12
const CARD_BUTTON_FONT_SIZE := 12
const CARD_BUTTON_HEIGHT := 28.0
const CARD_CORNER_RADIUS := 6
const CARD_MARGIN_X := 8
const CARD_MARGIN_Y := 6
const CARD_BORDER_WIDTH := 1

signal challenge_requested
signal pet_requested
signal lineup_requested
signal bag_requested
signal lineup_submit_requested(pet_uids: Array[int])

@onready var status_label: Label = %StatusLabel
@onready var scene_label: Label = %SceneLabel
@onready var player_label: Label = %PlayerLabel
@onready var mode_label: Label = %ModeLabel
@onready var summary_label: Label = %SummaryLabel
@onready var challenge_button: Button = %ChallengeButton
@onready var pet_button: Button = %PetButton
@onready var lineup_button: Button = %LineupButton
@onready var bag_button: Button = %BagButton
@onready var data_panel: PanelContainer = %DataPanel
@onready var data_title_label: Label = %DataTitleLabel
@onready var data_refresh_button: Button = %DataRefreshButton
@onready var data_close_button: Button = %DataCloseButton
@onready var data_hint_label: Label = %DataHintLabel
@onready var data_list: VBoxContainer = %DataList
@onready var data_footer: HBoxContainer = %DataFooter
@onready var data_reset_button: Button = %DataResetButton
@onready var data_apply_button: Button = %DataApplyButton
@onready var log_output: RichTextLabel = %LogOutput

var _active_panel_key: String = ""
var _pending_lineup: Array[int] = []

func _ready() -> void:
	challenge_button.pressed.connect(func() -> void: challenge_requested.emit())
	pet_button.pressed.connect(_on_pet_button_pressed)
	lineup_button.pressed.connect(_on_lineup_button_pressed)
	bag_button.pressed.connect(_on_bag_button_pressed)
	data_refresh_button.pressed.connect(_on_data_refresh_pressed)
	data_close_button.pressed.connect(_close_data_panel)
	data_reset_button.pressed.connect(_on_data_reset_pressed)
	data_apply_button.pressed.connect(_on_data_apply_pressed)

	GameState.session_changed.connect(_refresh_runtime_view)
	GameState.world_snapshot_changed.connect(_refresh_runtime_view)
	GameState.pets_changed.connect(_refresh_runtime_view)
	GameState.bag_changed.connect(_refresh_runtime_view)
	GameState.battle_changed.connect(_refresh_runtime_view)
	_refresh_runtime_view()

func set_header_texts(status_text: String, scene_text: String, player_text: String) -> void:
	status_label.text = status_text
	scene_label.text = scene_text
	player_label.text = player_text

func append_log(message: String) -> void:
	log_output.append_text(message + "\n")

func _refresh_runtime_view() -> void:
	var in_battle := GameState.is_in_battle
	mode_label.text = "战斗操作区" if in_battle else "世界操作区"
	if in_battle:
		var battle_id := str(GameState.battle_state.get("battle_id", "未分配"))
		var round_text := str(GameState.battle_state.get("round", 0))
		var active_pet_uid := str(GameState.battle_state.get("active_pet_uid", 0))
		summary_label.text = "战斗ID: %s | 回合: %s | 出战宠: %s" % [battle_id, round_text, active_pet_uid]
	else:
		summary_label.text = "附近实体: %d | 宠物: %d | 编队: %d | 背包: %d" % [
			GameState.nearby_entities.size(),
			GameState.pets.size(),
			GameState.lineup.size(),
			GameState.bag_items.size(),
		]

	challenge_button.visible = not in_battle
	challenge_button.disabled = in_battle or GameState.nearby_entities.is_empty()
	pet_button.text = "宠物 %d" % GameState.pets.size()
	lineup_button.text = "编队 %d" % GameState.lineup.size()
	bag_button.text = "背包 %d" % GameState.bag_items.size()
	if in_battle and _active_panel_key != "":
		_close_data_panel()
	else:
		_refresh_data_panel()

func _on_pet_button_pressed() -> void:
	pet_requested.emit()
	_toggle_data_panel("pets")

func _on_lineup_button_pressed() -> void:
	lineup_requested.emit()
	_toggle_data_panel("lineup")

func _on_bag_button_pressed() -> void:
	bag_requested.emit()
	_toggle_data_panel("bag")

func _toggle_data_panel(panel_key: String) -> void:
	if GameState.is_in_battle:
		return
	if _active_panel_key == panel_key and data_panel.visible:
		_close_data_panel()
		return
	_active_panel_key = panel_key
	if panel_key == "lineup":
		_sync_pending_lineup_from_state()
	data_panel.visible = true
	_refresh_data_panel()

func _close_data_panel() -> void:
	_active_panel_key = ""
	data_panel.visible = false

func _refresh_data_panel() -> void:
	if _active_panel_key.is_empty() or not data_panel.visible:
		return
	_clear_data_list()
	_refresh_panel_actions()
	match _active_panel_key:
		"pets":
			data_title_label.text = "宠物列表"
			data_hint_label.text = "查看当前宠物实例、HP 与是否已在编队中。"
			_render_pets_panel()
		"lineup":
			data_title_label.text = "当前编队"
			data_hint_label.text = "点击加入或移除宠物，使用上移/下移调整顺序，提交完整编队。"
			_render_lineup_panel()
		"bag":
			data_title_label.text = "背包摘要"
			data_hint_label.text = "查看当前背包中的基础物品数量摘要。"
			_render_bag_panel()
		_:
			data_title_label.text = "面板"
			data_hint_label.text = "暂无数据"
			_append_empty_card("暂无数据")

func _on_data_refresh_pressed() -> void:
	match _active_panel_key:
		"pets":
			pet_requested.emit()
		"lineup":
			lineup_requested.emit()
		"bag":
			bag_requested.emit()

func _on_data_reset_pressed() -> void:
	if _active_panel_key != "lineup":
		return
	_sync_pending_lineup_from_state()
	_refresh_data_panel()

func _on_data_apply_pressed() -> void:
	if _active_panel_key != "lineup" or _pending_lineup.is_empty():
		return
	lineup_submit_requested.emit(_pending_lineup.duplicate())

func _refresh_panel_actions() -> void:
	var is_lineup_panel := _active_panel_key == "lineup"
	data_footer.visible = is_lineup_panel
	data_reset_button.visible = is_lineup_panel
	data_apply_button.visible = is_lineup_panel
	data_apply_button.disabled = _pending_lineup.is_empty() or _pending_lineup == _current_lineup_uids()

func _render_pets_panel() -> void:
	if GameState.pets.is_empty():
		_append_empty_card("暂无宠物数据")
		return
	for pet_variant in GameState.pets:
		if pet_variant is Dictionary:
			var pet: Dictionary = pet_variant
			var detail_lines: Array[String] = [
				"等级 Lv.%s" % str(pet.get("level", 1)),
				"HP %s/%s" % [str(pet.get("hp", 0)), str(pet.get("hp_max", 0))],
			]
			var badge := "已在编队" if bool(pet.get("in_lineup", false)) else "待命"
			_append_info_card(
				"宠物 %s" % str(pet.get("pet_uid", 0)),
				"模板 %s | %s" % [str(pet.get("pet_id", 0)), badge],
				detail_lines,
				Color(0.22, 0.30, 0.42, 0.92),
				Color(0.46, 0.70, 0.98, 0.95)
			)

func _render_bag_panel() -> void:
	if GameState.bag_items.is_empty():
		_append_empty_card("背包暂无物品")
		return
	for item_variant in GameState.bag_items:
		if item_variant is Dictionary:
			var item: Dictionary = item_variant
			var quantity := int(item.get("count", item.get("quantity", item.get("num", 0))))
			var detail_lines: Array[String] = [
				"数量 x %d" % quantity,
			]
			_append_info_card(
				"物品 %s" % str(item.get("item_id", 0)),
				"背包物品摘要",
				detail_lines,
				Color(0.24, 0.28, 0.24, 0.92),
				Color(0.58, 0.78, 0.52, 0.95)
			)

func _render_lineup_panel() -> void:
	_append_section_label("当前编队")
	if _pending_lineup.is_empty():
		_append_empty_card("当前未选择编队，请至少加入一只宠物。")
	else:
		for index in _pending_lineup.size():
			var pet := _find_pet_by_uid(_pending_lineup[index])
			_append_lineup_card(pet, index, _pending_lineup.size())

	_append_section_label("可加入宠物")
	var added_candidate := false
	for pet_variant in GameState.pets:
		if pet_variant is Dictionary:
			var pet: Dictionary = pet_variant
			var pet_uid := int(pet.get("pet_uid", 0))
			if pet_uid == 0 or _pending_lineup.has(pet_uid):
				continue
			_append_candidate_pet_card(pet)
			added_candidate = true
	if not added_candidate:
		_append_empty_card("当前没有可加入的其他宠物。")

func _append_lineup_card(pet: Dictionary, index: int, total: int) -> void:
	var title := "%d. 宠物 %s" % [index + 1, str(pet.get("pet_uid", 0))]
	var subtitle := "模板 %s | Lv.%s | HP %s/%s" % [
		str(pet.get("pet_id", 0)),
		str(pet.get("level", 1)),
		str(pet.get("hp", 0)),
		str(pet.get("hp_max", 0)),
	]
	var panel := _create_card_panel(Color(0.32, 0.26, 0.18, 0.96), Color(0.93, 0.78, 0.42, 0.95))
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 3
	panel.add_child(root)

	var title_label := Label.new()
	title_label.text = title
	title_label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	root.add_child(title_label)

	var subtitle_label := Label.new()
	subtitle_label.text = subtitle
	subtitle_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	subtitle_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
	root.add_child(subtitle_label)

	var actions := HBoxContainer.new()
	actions.theme_override_constants.separation = 4
	root.add_child(actions)

	var up_button := Button.new()
	up_button.text = "上移"
	up_button.custom_minimum_size = Vector2(0, CARD_BUTTON_HEIGHT)
	up_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	up_button.disabled = index == 0
	up_button.add_theme_font_size_override("font_size", CARD_BUTTON_FONT_SIZE)
	up_button.pressed.connect(func() -> void:
		_move_pending_lineup(index, -1)
	)
	actions.add_child(up_button)

	var down_button := Button.new()
	down_button.text = "下移"
	down_button.custom_minimum_size = Vector2(0, CARD_BUTTON_HEIGHT)
	down_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	down_button.disabled = index >= total - 1
	down_button.add_theme_font_size_override("font_size", CARD_BUTTON_FONT_SIZE)
	down_button.pressed.connect(func() -> void:
		_move_pending_lineup(index, 1)
	)
	actions.add_child(down_button)

	var remove_button := Button.new()
	remove_button.text = "移除"
	remove_button.custom_minimum_size = Vector2(0, CARD_BUTTON_HEIGHT)
	remove_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	remove_button.add_theme_font_size_override("font_size", CARD_BUTTON_FONT_SIZE)
	remove_button.pressed.connect(func() -> void:
		_remove_from_pending_lineup(int(pet.get("pet_uid", 0)))
	)
	actions.add_child(remove_button)

	data_list.add_child(panel)

func _append_candidate_pet_card(pet: Dictionary) -> void:
	var pet_uid := int(pet.get("pet_uid", 0))
	var panel := _create_card_panel(Color(0.18, 0.22, 0.31, 0.94), Color(0.52, 0.72, 0.94, 0.95))
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 3
	panel.add_child(root)

	var title_label := Label.new()
	title_label.text = "宠物 %s" % str(pet.get("pet_uid", 0))
	title_label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	root.add_child(title_label)

	var subtitle_label := Label.new()
	subtitle_label.text = "模板 %s | Lv.%s | HP %s/%s" % [
		str(pet.get("pet_id", 0)),
		str(pet.get("level", 1)),
		str(pet.get("hp", 0)),
		str(pet.get("hp_max", 0)),
	]
	subtitle_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	subtitle_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
	root.add_child(subtitle_label)

	var add_button := Button.new()
	add_button.text = "加入编队"
	add_button.custom_minimum_size = Vector2(0, CARD_BUTTON_HEIGHT)
	add_button.add_theme_font_size_override("font_size", CARD_BUTTON_FONT_SIZE)
	add_button.pressed.connect(func() -> void:
		_add_to_pending_lineup(pet_uid)
	)
	root.add_child(add_button)

	data_list.add_child(panel)

func _append_info_card(title: String, subtitle: String, detail_lines: Array[String], fill_color: Color, border_color: Color) -> void:
	var panel := _create_card_panel(fill_color, border_color)
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 2
	panel.add_child(root)

	var title_label := Label.new()
	title_label.text = title
	title_label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	root.add_child(title_label)

	var subtitle_label := Label.new()
	subtitle_label.text = subtitle
	subtitle_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
	subtitle_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	root.add_child(subtitle_label)

	for line in detail_lines:
		var line_label := Label.new()
		line_label.text = line
		line_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
		root.add_child(line_label)

	data_list.add_child(panel)

func _append_section_label(text: String) -> void:
	var label := Label.new()
	label.text = text
	label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	data_list.add_child(label)

func _append_empty_card(text: String) -> void:
	var panel := _create_card_panel(Color(0.18, 0.18, 0.18, 0.88), Color(0.42, 0.42, 0.42, 0.95))
	var label := Label.new()
	label.text = text
	label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
	panel.add_child(label)
	data_list.add_child(panel)

func _create_card_panel(fill_color: Color, border_color: Color) -> PanelContainer:
	var panel := PanelContainer.new()
	panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL

	var style := StyleBoxFlat.new()
	style.bg_color = fill_color
	style.border_width_left = CARD_BORDER_WIDTH
	style.border_width_top = CARD_BORDER_WIDTH
	style.border_width_right = CARD_BORDER_WIDTH
	style.border_width_bottom = CARD_BORDER_WIDTH
	style.border_color = border_color
	style.corner_radius_top_left = CARD_CORNER_RADIUS
	style.corner_radius_top_right = CARD_CORNER_RADIUS
	style.corner_radius_bottom_left = CARD_CORNER_RADIUS
	style.corner_radius_bottom_right = CARD_CORNER_RADIUS
	style.content_margin_left = CARD_MARGIN_X
	style.content_margin_top = CARD_MARGIN_Y
	style.content_margin_right = CARD_MARGIN_X
	style.content_margin_bottom = CARD_MARGIN_Y
	panel.add_theme_stylebox_override("panel", style)
	return panel

func _clear_data_list() -> void:
	for child in data_list.get_children():
		child.queue_free()

func _sync_pending_lineup_from_state() -> void:
	_pending_lineup.clear()
	for lineup_variant in GameState.lineup:
		if lineup_variant is Dictionary:
			var pet_uid := int(lineup_variant.get("pet_uid", 0))
			if pet_uid != 0:
				_pending_lineup.append(pet_uid)

func _current_lineup_uids() -> Array[int]:
	var result: Array[int] = []
	for lineup_variant in GameState.lineup:
		if lineup_variant is Dictionary:
			var pet_uid := int(lineup_variant.get("pet_uid", 0))
			if pet_uid != 0:
				result.append(pet_uid)
	return result

func _find_pet_by_uid(pet_uid: int) -> Dictionary:
	for pet_variant in GameState.pets:
		if pet_variant is Dictionary and int(pet_variant.get("pet_uid", 0)) == pet_uid:
			return pet_variant
	for lineup_variant in GameState.lineup:
		if lineup_variant is Dictionary and int(lineup_variant.get("pet_uid", 0)) == pet_uid:
			return lineup_variant
	return {"pet_uid": pet_uid}

func _add_to_pending_lineup(pet_uid: int) -> void:
	if pet_uid == 0 or _pending_lineup.has(pet_uid):
		return
	_pending_lineup.append(pet_uid)
	_refresh_data_panel()

func _remove_from_pending_lineup(pet_uid: int) -> void:
	var index := _pending_lineup.find(pet_uid)
	if index == -1:
		return
	_pending_lineup.remove_at(index)
	_refresh_data_panel()

func _move_pending_lineup(index: int, offset: int) -> void:
	var target_index := index + offset
	if index < 0 or index >= _pending_lineup.size():
		return
	if target_index < 0 or target_index >= _pending_lineup.size():
		return
	var pet_uid := _pending_lineup[index]
	_pending_lineup.remove_at(index)
	_pending_lineup.insert(target_index, pet_uid)
	_refresh_data_panel()
