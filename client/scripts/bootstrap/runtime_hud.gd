extends Control
class_name RuntimeHud

# 数据卡标题使用的字号。
const CARD_TITLE_FONT_SIZE := 13
# 数据卡正文使用的字号。
const CARD_TEXT_FONT_SIZE := 11
# 数据卡按钮使用的字号。
const CARD_BUTTON_FONT_SIZE := 11
# 数据卡按钮的统一高度。
const CARD_BUTTON_HEIGHT := 26.0
# 数据卡边角使用的圆角半径。
const CARD_CORNER_RADIUS := 6
# 数据卡内部左右留白。
const CARD_MARGIN_X := 6
# 数据卡内部上下留白。
const CARD_MARGIN_Y := 5
# 数据卡描边的统一宽度。
const CARD_BORDER_WIDTH := 1

# 请求挑战附近目标时向外发出的信号。
signal challenge_requested
# 请求刷新宠物列表时向外发出的信号。
signal pet_requested
# 请求刷新编队数据时向外发出的信号。
signal lineup_requested
# 请求刷新背包摘要时向外发出的信号。
signal bag_requested
# 请求刷新任务列表时向外发出的信号。
signal quest_requested
# 请求切换当前追踪任务时向外发出的信号。
signal quest_track_requested(quest_id: int)
# 请求接取任务时向外发出的信号。
signal quest_accept_requested(quest_id: int, npc_id: int)
# 请求提交任务时向外发出的信号。
signal quest_submit_requested(quest_id: int, npc_id: int)
# 提交当前待保存编队时向外发出的信号。
signal lineup_submit_requested(pet_uids: Array[int])
signal npc_menu_refresh_requested(entity_id: int)
signal npc_menu_entry_selected(entity_id: int, entry_id: String, entry_type: String, quest_id: int, quest_state: String)

# 顶部状态栏中显示连接状态的标签。
@onready var status_label: Label = %StatusLabel
# 顶部状态栏中显示当前场景的标签。
@onready var scene_label: Label = %SceneLabel
# 顶部状态栏中显示玩家信息的标签。
@onready var player_label: Label = %PlayerLabel
# 操作区模式提示标签。
@onready var mode_label: Label = %ModeLabel
# 操作区摘要说明标签。
@onready var summary_label: Label = %SummaryLabel
# 操作区任务追踪标签。
@onready var tracking_label: Label = %TrackingLabel
# 世界态下发起挑战的按钮。
@onready var challenge_button: Button = %ChallengeButton
# 打开宠物面板的按钮。
@onready var pet_button: Button = %PetButton
# 打开编队面板的按钮。
@onready var lineup_button: Button = %LineupButton
# 打开任务面板的按钮。
@onready var quest_button: Button = %QuestButton
# 打开背包面板的按钮。
@onready var bag_button: Button = %BagButton
# 承载弹出数据面板的根容器。
@onready var data_panel: PanelContainer = %DataPanel
# 数据面板标题标签。
@onready var data_title_label: Label = %DataTitleLabel
# 数据面板刷新按钮。
@onready var data_refresh_button: Button = %DataRefreshButton
# 数据面板关闭按钮。
@onready var data_close_button: Button = %DataCloseButton
# 数据面板提示文案标签。
@onready var data_hint_label: Label = %DataHintLabel
# 数据面板滚动内容列表容器。
@onready var data_list: VBoxContainer = %DataList
# 数据面板底部操作栏。
@onready var data_footer: HBoxContainer = %DataFooter
# 数据面板中重置待编辑编队的按钮。
@onready var data_reset_button: Button = %DataResetButton
# 数据面板中提交待编辑编队的按钮。
@onready var data_apply_button: Button = %DataApplyButton
# 底部日志输出区域。
@onready var log_output: RichTextLabel = %LogOutput

# 记录当前处于打开状态的数据面板类型。
var _active_panel_key: String = ""
# 缓存当前面板里待提交的编队宠物唯一标识列表。
var _pending_lineup: Array[int] = []
var _npc_menu_payload: Dictionary = {}

# 绑定按钮与全局状态信号，并在首次显示前完成一次界面刷新。
func _ready() -> void:
	challenge_button.pressed.connect(func() -> void: challenge_requested.emit())
	pet_button.pressed.connect(_on_pet_button_pressed)
	lineup_button.pressed.connect(_on_lineup_button_pressed)
	quest_button.pressed.connect(_on_quest_button_pressed)
	bag_button.pressed.connect(_on_bag_button_pressed)
	data_refresh_button.pressed.connect(_on_data_refresh_pressed)
	data_close_button.pressed.connect(_close_data_panel)
	data_reset_button.pressed.connect(_on_data_reset_pressed)
	data_apply_button.pressed.connect(_on_data_apply_pressed)

	GameState.session_changed.connect(_refresh_runtime_view)
	GameState.world_snapshot_changed.connect(_refresh_runtime_view)
	GameState.pets_changed.connect(_refresh_runtime_view)
	GameState.bag_changed.connect(_refresh_runtime_view)
	GameState.quests_changed.connect(_refresh_runtime_view)
	GameState.battle_changed.connect(_refresh_runtime_view)
	_refresh_runtime_view()

# 更新顶部状态栏显示文案。
func set_header_texts(status_text: String, scene_text: String, player_text: String) -> void:
	status_label.text = status_text
	scene_label.text = scene_text
	player_label.text = player_text

# 追加一条运行日志到输出区域末尾。
func append_log(message: String) -> void:
	log_output.append_text(message + "\n")

# 根据当前全局状态刷新按钮可用性、摘要文本和数据面板显示。
func _refresh_runtime_view() -> void:
	# 标记当前是否处于战斗态，用于切换操作区文案和交互权限。
	var in_battle := GameState.is_in_battle
	mode_label.text = "战斗操作区" if in_battle else "世界操作区"
	if in_battle:
		# 读取当前战斗的唯一标识用于摘要展示。
		var battle_id := str(GameState.battle_state.get("battle_id", "未分配"))
		# 读取当前战斗回合数用于摘要展示。
		var round_text := str(GameState.battle_state.get("round", 0))
		# 读取当前出战宠物唯一标识用于摘要展示。
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
	quest_button.text = "任务 %d" % GameState.quests.size()
	bag_button.text = "背包 %d" % GameState.bag_items.size()
	tracking_label.text = _build_tracking_text()
	if in_battle and _active_panel_key != "":
		_close_data_panel()
	else:
		_refresh_data_panel()

# 处理宠物按钮点击，并切换到宠物面板。
func _on_pet_button_pressed() -> void:
	pet_requested.emit()
	_toggle_data_panel("pets")

# 处理编队按钮点击，并切换到编队面板。
func _on_lineup_button_pressed() -> void:
	lineup_requested.emit()
	_toggle_data_panel("lineup")

# 处理背包按钮点击，并切换到背包面板。
func _on_bag_button_pressed() -> void:
	bag_requested.emit()
	_toggle_data_panel("bag")

# 处理任务按钮点击，并切换到任务面板。
func _on_quest_button_pressed() -> void:
	quest_requested.emit()
	_toggle_data_panel("quests")

# 按面板类型切换可见状态，并在需要时同步待编辑编队。
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

# 关闭当前数据面板并清空激活标记。
func _close_data_panel() -> void:
	if _active_panel_key == "npc_menu":
		_npc_menu_payload = {}
	_active_panel_key = ""
	data_panel.visible = false

# 按当前面板类型重建内容列表和底部操作区。
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
		"quests":
			data_title_label.text = "任务列表"
			data_hint_label.text = "查看当前任务状态，支持接取、提交和追踪。"
			_render_quests_panel()
		"npc_menu":
			_render_npc_menu_panel()
		_:
			data_title_label.text = "面板"
			data_hint_label.text = "暂无数据"
			_append_empty_card("暂无数据")

# 根据当前打开的面板触发对应的数据刷新请求。
func _on_data_refresh_pressed() -> void:
	match _active_panel_key:
		"pets":
			pet_requested.emit()
		"lineup":
			lineup_requested.emit()
		"bag":
			bag_requested.emit()
		"quests":
			quest_requested.emit()
		"npc_menu":
			var entity_id := int(_npc_menu_payload.get("entity_id", 0))
			if entity_id > 0:
				npc_menu_refresh_requested.emit(entity_id)

# 放弃本地临时编队改动并恢复为全局状态中的编队顺序。
func _on_data_reset_pressed() -> void:
	if _active_panel_key != "lineup":
		return
	_sync_pending_lineup_from_state()
	_refresh_data_panel()

# 把当前临时编队提交给外层主流程。
func _on_data_apply_pressed() -> void:
	if _active_panel_key != "lineup" or _pending_lineup.is_empty():
		return
	lineup_submit_requested.emit(_pending_lineup.duplicate())

# 按当前面板类型控制底部按钮栏是否显示以及提交按钮是否可用。
func _refresh_panel_actions() -> void:
	# 标记当前是否处于编队编辑面板。
	var is_lineup_panel := _active_panel_key == "lineup"
	data_footer.visible = is_lineup_panel
	data_reset_button.visible = is_lineup_panel
	data_apply_button.visible = is_lineup_panel
	data_apply_button.disabled = _pending_lineup.is_empty() or _pending_lineup == _current_lineup_uids()

# 渲染宠物列表卡片内容。
func _render_pets_panel() -> void:
	if GameState.pets.is_empty():
		_append_empty_card("暂无宠物数据")
		return
	for pet_variant in GameState.pets:
		if pet_variant is Dictionary:
			# 保存当前遍历到的宠物实例数据。
			var pet: Dictionary = pet_variant
			# 组织当前宠物卡片要显示的详情行文本。
			var detail_lines: Array[String] = [
				"等级 Lv.%s" % str(pet.get("level", 1)),
				"HP %s/%s" % [str(pet.get("hp", 0)), str(pet.get("hp_max", 0))],
			]
			# 生成当前宠物的编队状态标记文案。
			var badge := "已在编队" if bool(pet.get("in_lineup", false)) else "待命"
			_append_info_card(
				"宠物 %s" % str(pet.get("pet_uid", 0)),
				"模板 %s | %s" % [str(pet.get("pet_id", 0)), badge],
				detail_lines,
				Color(0.22, 0.30, 0.42, 0.92),
				Color(0.46, 0.70, 0.98, 0.95)
			)

# 渲染背包摘要卡片内容。
func _render_bag_panel() -> void:
	if GameState.bag_items.is_empty():
		_append_empty_card("背包暂无物品")
		return
	for item_variant in GameState.bag_items:
		if item_variant is Dictionary:
			# 保存当前遍历到的背包物品数据。
			var item: Dictionary = item_variant
			# 兼容不同字段名后得到当前物品数量。
			var quantity := int(item.get("count", item.get("quantity", item.get("num", 0))))
			# 组织当前物品卡片要显示的详情行文本。
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

# 渲染任务列表卡片内容。
func _render_quests_panel() -> void:
	if GameState.quests.is_empty():
		_append_empty_card("暂无任务数据")
		return
	for quest_variant in GameState.quests:
		if quest_variant is Dictionary:
			var quest: Dictionary = quest_variant
			_append_quest_card(quest)

func _append_quest_card(quest: Dictionary) -> void:
	var quest_id := int(quest.get("quest_id", 0))
	var state := str(quest.get("state", "UNKNOWN"))
	var subtitle := "状态 %s | 类型 %s" % [state, str(quest.get("quest_type", "UNKNOWN"))]
	var detail_lines: Array[String] = []
	detail_lines.append(str(quest.get("description", "")))
	var objective_text := _first_incomplete_objective_text(quest)
	if not objective_text.is_empty():
		detail_lines.append("目标: %s" % objective_text)
	if bool(quest.get("tracked", false)):
		detail_lines.append("当前追踪中")
	_append_action_card(
		"任务 %s - %s" % [str(quest_id), str(quest.get("title", "未命名任务"))],
		subtitle,
		detail_lines,
		_build_quest_actions(quest),
		Color(0.19, 0.26, 0.36, 0.94),
		Color(0.97, 0.84, 0.43, 0.96)
	)

func _build_tracking_text() -> String:
	if GameState.is_in_battle:
		return "任务追踪: 战斗中暂停显示"
	var tracked := GameState.tracked_quest()
	if tracked.is_empty():
		return "任务追踪: 暂无"
	var title := str(tracked.get("title", "未命名任务"))
	var objective_text := _first_incomplete_objective_text(tracked)
	if objective_text.is_empty():
		objective_text = str(tracked.get("state", "UNKNOWN"))
	return "任务追踪: %s | %s" % [title, objective_text]

func _first_incomplete_objective_text(quest: Dictionary) -> String:
	var objectives_variant: Variant = quest.get("objectives", [])
	if objectives_variant is not Array:
		return ""
	for objective_variant in objectives_variant:
		if objective_variant is Dictionary and not bool(objective_variant.get("completed", false)):
			var current := int(objective_variant.get("current", 0))
			var target := int(objective_variant.get("target", 0))
			return "%s (%d/%d)" % [str(objective_variant.get("description", "未命名目标")), current, target]
	return ""

func _build_quest_actions(quest: Dictionary) -> Array[Dictionary]:
	var actions: Array[Dictionary] = []
	var quest_id := int(quest.get("quest_id", 0))
	var state := str(quest.get("state", ""))
	actions.append({
		"label": "追踪",
		"disabled": GameState.tracked_quest_id == quest_id,
		"callback": func() -> void: quest_track_requested.emit(quest_id),
	})
	if state == "AVAILABLE":
		var start_npc_id := int(quest.get("start_npc_id", 0))
		if start_npc_id == 0:
			actions.append({
				"label": "接取",
				"disabled": false,
				"callback": func() -> void: quest_accept_requested.emit(quest_id, 0),
			})
	elif state == "READY_TO_SUBMIT":
		var submit_npc_id := int(quest.get("submit_npc_id", 0))
		if submit_npc_id == 0:
			actions.append({
				"label": "提交",
				"disabled": false,
				"callback": func() -> void: quest_submit_requested.emit(quest_id, 0),
			})
	return actions

# 渲染编队编辑面板，分为当前编队和待加入候选两部分。
func _render_lineup_panel() -> void:
	_append_section_label("当前编队")
	if _pending_lineup.is_empty():
		_append_empty_card("当前未选择编队，请至少加入一只宠物。")
	else:
		for index in _pending_lineup.size():
			var pet := _find_pet_by_uid(_pending_lineup[index])
			_append_lineup_card(pet, index, _pending_lineup.size())

	_append_section_label("可加入宠物")
	# 标记当前是否已渲染至少一个候选宠物卡片。
	var added_candidate := false
	for pet_variant in GameState.pets:
		if pet_variant is Dictionary:
			# 保存当前遍历到的候选宠物数据。
			var pet: Dictionary = pet_variant
			# 读取候选宠物唯一标识，用于判定是否已在待编辑编队中。
			var pet_uid := int(pet.get("pet_uid", 0))
			if pet_uid == 0 or _pending_lineup.has(pet_uid):
				continue
			_append_candidate_pet_card(pet)
			added_candidate = true
	if not added_candidate:
		_append_empty_card("当前没有可加入的其他宠物。")

# 追加一张可调整顺序和移除的编队卡片。
func _append_lineup_card(pet: Dictionary, index: int, total: int) -> void:
	# 生成编队卡片标题文案。
	var title := "%d. 宠物 %s" % [index + 1, str(pet.get("pet_uid", 0))]
	# 生成编队卡片副标题文案。
	var subtitle := "模板 %s | Lv.%s | HP %s/%s" % [
		str(pet.get("pet_id", 0)),
		str(pet.get("level", 1)),
		str(pet.get("hp", 0)),
		str(pet.get("hp_max", 0)),
	]
	# 创建承载当前卡片内容的带边框面板。
	var panel := _create_card_panel(Color(0.32, 0.26, 0.18, 0.96), Color(0.93, 0.78, 0.42, 0.95))
	# 创建卡片内部使用的纵向布局容器。
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 3
	panel.add_child(root)

	# 创建显示卡片标题的文本控件。
	var title_label := Label.new()
	title_label.text = title
	title_label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	root.add_child(title_label)

	# 创建显示卡片副标题的文本控件。
	var subtitle_label := Label.new()
	subtitle_label.text = subtitle
	subtitle_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	subtitle_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
	root.add_child(subtitle_label)

	# 创建承载排序与移除按钮的横向布局容器。
	var actions := HBoxContainer.new()
	actions.theme_override_constants.separation = 4
	root.add_child(actions)

	# 创建把当前卡片向前移动的按钮。
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

	# 创建把当前卡片向后移动的按钮。
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

	# 创建把当前宠物移出待编辑编队的按钮。
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

# 追加一张可加入编队的候选宠物卡片。
func _append_candidate_pet_card(pet: Dictionary) -> void:
	# 读取候选宠物唯一标识，供按钮回调复用。
	var pet_uid := int(pet.get("pet_uid", 0))
	# 创建承载候选宠物内容的带边框面板。
	var panel := _create_card_panel(Color(0.18, 0.22, 0.31, 0.94), Color(0.52, 0.72, 0.94, 0.95))
	# 创建候选卡片内部使用的纵向布局容器。
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 3
	panel.add_child(root)

	# 创建显示宠物标题的文本控件。
	var title_label := Label.new()
	title_label.text = "宠物 %s" % str(pet.get("pet_uid", 0))
	title_label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	root.add_child(title_label)

	# 创建显示宠物详情的副标题控件。
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

	# 创建把当前候选宠物加入待编辑编队的按钮。
	var add_button := Button.new()
	add_button.text = "加入编队"
	add_button.custom_minimum_size = Vector2(0, CARD_BUTTON_HEIGHT)
	add_button.add_theme_font_size_override("font_size", CARD_BUTTON_FONT_SIZE)
	add_button.pressed.connect(func() -> void:
		_add_to_pending_lineup(pet_uid)
	)
	root.add_child(add_button)

	data_list.add_child(panel)

# 追加一张通用信息卡片，用于宠物和背包摘要展示。
func _append_info_card(title: String, subtitle: String, detail_lines: Array[String], fill_color: Color, border_color: Color) -> void:
	# 创建当前信息卡片的基础面板。
	var panel := _create_card_panel(fill_color, border_color)
	# 创建信息卡片内部使用的纵向布局容器。
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 2
	panel.add_child(root)

	# 创建信息卡标题控件。
	var title_label := Label.new()
	title_label.text = title
	title_label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	root.add_child(title_label)

	# 创建信息卡副标题控件。
	var subtitle_label := Label.new()
	subtitle_label.text = subtitle
	subtitle_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
	subtitle_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	root.add_child(subtitle_label)

	for line in detail_lines:
		# 为每一行详情创建独立的文本控件。
		var line_label := Label.new()
		line_label.text = line
		line_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
		root.add_child(line_label)

	data_list.add_child(panel)

# 追加一张带操作按钮的数据卡片，用于任务列表等可交互面板。
func _append_action_card(title: String, subtitle: String, detail_lines: Array[String], actions: Array[Dictionary], fill_color: Color, border_color: Color) -> void:
	var panel := _create_card_panel(fill_color, border_color)
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 4
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
		line_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
		root.add_child(line_label)

	if not actions.is_empty():
		var button_row := HBoxContainer.new()
		button_row.theme_override_constants.separation = 6
		root.add_child(button_row)
		for action_variant in actions:
			if action_variant is Dictionary:
				var action: Dictionary = action_variant
				var button := Button.new()
				button.text = str(action.get("label", "操作"))
				button.custom_minimum_size = Vector2(0, CARD_BUTTON_HEIGHT)
				button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
				button.disabled = bool(action.get("disabled", false))
				button.add_theme_font_size_override("font_size", CARD_BUTTON_FONT_SIZE)
				var callback_variant: Variant = action.get("callback")
				if callback_variant is Callable:
					button.pressed.connect(callback_variant)
				button_row.add_child(button)

	data_list.add_child(panel)

# 追加一个用于分隔面板内容区域的小节标题。
func _append_section_label(text: String) -> void:
	# 创建显示小节标题的标签控件。
	var label := Label.new()
	label.text = text
	label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	data_list.add_child(label)

# 追加一张只显示提示文本的空态卡片。
func _append_empty_card(text: String) -> void:
	# 创建空态卡片的基础面板。
	var panel := _create_card_panel(Color(0.18, 0.18, 0.18, 0.88), Color(0.42, 0.42, 0.42, 0.95))
	# 创建显示空态文案的标签控件。
	var label := Label.new()
	label.text = text
	label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
	panel.add_child(label)
	data_list.add_child(panel)

# 创建一张带统一边框、留白和圆角样式的数据卡面板。
func _create_card_panel(fill_color: Color, border_color: Color) -> PanelContainer:
	# 创建当前卡片的面板容器。
	var panel := PanelContainer.new()
	panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL

	# 创建当前卡片使用的扁平样式盒。
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

# 清空当前数据面板里已经渲染的全部子节点。
func _clear_data_list() -> void:
	for child in data_list.get_children():
		child.queue_free()

# 把当前全局编队同步到本地待编辑列表。
func _sync_pending_lineup_from_state() -> void:
	_pending_lineup.clear()
	for lineup_variant in GameState.lineup:
		if lineup_variant is Dictionary:
			# 读取当前编队项对应的宠物唯一标识。
			var pet_uid := int(lineup_variant.get("pet_uid", 0))
			if pet_uid != 0:
				_pending_lineup.append(pet_uid)

# 提取当前全局编队中的宠物唯一标识顺序。
func _current_lineup_uids() -> Array[int]:
	# 保存整理后的编队唯一标识列表。
	var result: Array[int] = []
	for lineup_variant in GameState.lineup:
		if lineup_variant is Dictionary:
			# 读取当前编队项对应的宠物唯一标识。
			var pet_uid := int(lineup_variant.get("pet_uid", 0))
			if pet_uid != 0:
				result.append(pet_uid)
	return result

# 按宠物唯一标识查找完整宠物数据，不存在时返回最小占位结构。
func _find_pet_by_uid(pet_uid: int) -> Dictionary:
	for pet_variant in GameState.pets:
		if pet_variant is Dictionary and int(pet_variant.get("pet_uid", 0)) == pet_uid:
			return pet_variant
	for lineup_variant in GameState.lineup:
		if lineup_variant is Dictionary and int(lineup_variant.get("pet_uid", 0)) == pet_uid:
			return lineup_variant
	return {"pet_uid": pet_uid}

# 把指定宠物加入本地待编辑编队。
func _add_to_pending_lineup(pet_uid: int) -> void:
	if pet_uid == 0 or _pending_lineup.has(pet_uid):
		return
	_pending_lineup.append(pet_uid)
	_refresh_data_panel()

# 把指定宠物从本地待编辑编队中移除。
func _remove_from_pending_lineup(pet_uid: int) -> void:
	# 查找当前宠物在待编辑编队中的位置。
	var index := _pending_lineup.find(pet_uid)
	if index == -1:
		return
	_pending_lineup.remove_at(index)
	_refresh_data_panel()

# 按给定偏移量调整待编辑编队中的顺序。
func _move_pending_lineup(index: int, offset: int) -> void:
	# 计算移动后的目标位置。
	var target_index := index + offset
	if index < 0 or index >= _pending_lineup.size():
		return
	if target_index < 0 or target_index >= _pending_lineup.size():
		return
	# 暂存当前要移动的宠物唯一标识。
	var pet_uid := _pending_lineup[index]
	_pending_lineup.remove_at(index)
	_pending_lineup.insert(target_index, pet_uid)
	_refresh_data_panel()


func show_npc_menu(menu_payload: Dictionary) -> void:
	_npc_menu_payload = menu_payload.duplicate(true)
	_active_panel_key = "npc_menu"
	data_panel.visible = true
	_refresh_data_panel()

func hide_npc_menu() -> void:
	if _active_panel_key == "npc_menu":
		_npc_menu_payload = {}
	_close_data_panel()

func _render_npc_menu_panel() -> void:
	var npc_name := str(_npc_menu_payload.get("npc_name", "可交互 NPC"))
	data_title_label.text = npc_name
	data_hint_label.text = "选择一个操作。"
	var entries_value: Variant = _npc_menu_payload.get("menu_entries", [])
	if entries_value is not Array or (entries_value as Array).is_empty():
		_append_empty_card("当前没有可用操作")
		return
	for entry_value in entries_value:
		if entry_value is not Dictionary:
			continue
		var entry: Dictionary = entry_value
		_append_npc_menu_entry(entry)

func _append_npc_menu_entry(entry: Dictionary) -> void:
	var title := "[%s] %s" % [str(entry.get("entry_type", "操作")), str(entry.get("title", "未命名操作"))]
	var subtitle := str(entry.get("subtitle", ""))
	var state := str(entry.get("state", "available"))
	var panel := _create_card_panel(Color(0.18, 0.22, 0.30, 0.96), Color(0.55, 0.74, 0.96, 0.95))
	var root := VBoxContainer.new()
	root.theme_override_constants.separation = 4
	panel.add_child(root)
	var title_label := Label.new()
	title_label.text = title
	title_label.add_theme_font_size_override("font_size", CARD_TITLE_FONT_SIZE)
	root.add_child(title_label)
	if not subtitle.is_empty():
		var subtitle_label := Label.new()
		subtitle_label.text = subtitle
		subtitle_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
		subtitle_label.add_theme_font_size_override("font_size", CARD_TEXT_FONT_SIZE)
		root.add_child(subtitle_label)
	var action_button := Button.new()
	action_button.text = "选择"
	action_button.custom_minimum_size = Vector2(0, CARD_BUTTON_HEIGHT)
	action_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	action_button.add_theme_font_size_override("font_size", CARD_BUTTON_FONT_SIZE)
	action_button.disabled = state == "locked"
	action_button.pressed.connect(func() -> void:
		npc_menu_entry_selected.emit(
			int(_npc_menu_payload.get("entity_id", 0)),
			str(entry.get("entry_id", "")),
			str(entry.get("entry_type", "")),
			int(entry.get("quest_id", 0)),
			str(entry.get("quest_state", ""))
		)
	)
	root.add_child(action_button)
	data_list.add_child(panel)
