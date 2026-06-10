extends CanvasLayer

signal menu_closed

const MENU_ICON_1: Texture2D = preload("res://asset/口袋所有形象/菜单图标1.png")
const MENU_ICON_2: Texture2D = preload("res://asset/口袋所有形象/菜单图标2.png")
const ICON_SIZE := 16
const TAB_TEXT_COLOR := Color(0.82, 0.9, 0.98, 1.0)
const TAB_ACTIVE_COLOR := Color(1.0, 0.88, 0.35, 1.0)
const ITEM_TEXT_COLOR := Color(0.86, 0.92, 0.96, 1.0)
const ITEM_ACTIVE_COLOR := Color(1.0, 0.93, 0.52, 1.0)
const ITEM_HOVER_COLOR := Color(0.96, 0.96, 0.78, 1.0)
const ITEM_SHADOW_COLOR := Color(0.0, 0.0, 0.0, 0.85)
const ITEM_ACTIVE_BACKGROUND := Color(0.22, 0.32, 0.44, 0.55)
const ITEM_HOVER_BACKGROUND := Color(0.18, 0.24, 0.34, 0.45)
const ITEM_IDLE_BACKGROUND := Color(0.0, 0.0, 0.0, 0.0)

const MENU_DATA: Array[Dictionary] = [
	{
		"title": "常用",
		"items": [
			{"label": "物品行囊", "icon": "2.8"},
			{"label": "个人状态", "icon": "2.10"},
			{"label": "宠物指令", "icon": "1.9"},
			{"label": "队伍指令", "icon": "1.13"},
			{"label": "神秘商店", "icon": "2.4"},
			{"label": "佣兵指令", "icon": "1.9"},
			{"label": "宠物之魂", "icon": "2.12"},
			{"label": "龙魂系统", "icon": "1.49"},
			{"label": "全服竞技场", "icon": "1.49"},
			{"label": "仙气修炼", "icon": "1.49"},
		],
	},
	{
		"title": "辅助",
		"items": [
			{"label": "快速购物", "icon": "1.16"},
			{"label": "生产装备", "icon": "2.18"},
			{"label": "快速补血", "icon": "1.14"},
			{"label": "查看成就", "icon": "2.6"},
			{"label": "原地挂机", "icon": "1.47"},
			{"label": "生活技能", "icon": "1.41"},
			{"label": "灵魂粉末", "icon": "1.11"},
		],
	},
	{
		"title": "聊天",
		"items": [
			{"label": "同屏聊天", "icon": "2.7"},
			{"label": "本服广播", "icon": "2.28"},
			{"label": "本线广播", "icon": "2.28"},
			{"label": "队伍聊天", "icon": "2.7"},
			{"label": "发悄悄话", "icon": "2.7"},
			{"label": "口袋好友", "icon": "1.42"},
			{"label": "家族指令", "icon": "1.42"},
		],
	},
	{
		"title": "任务",
		"items": [
			{"label": "查看任务", "icon": "2.21"},
			{"label": "区域地图", "icon": "2.11"},
			{"label": "世界地图", "icon": "2.11"},
			{"label": "每日活动", "icon": "2.20"},
			{"label": "口袋攻略", "icon": "1.12"},
		],
	},
	{
		"title": "系统",
		"items": [
			{"label": "流量设置", "icon": "1.13"},
			{"label": "游戏帮助", "icon": "1.13"},
			{"label": "细节设置", "icon": "1.13"},
			{"label": "退出游戏", "icon": "1.13"},
			{"label": "免费领取", "icon": "1.13"},
			{"label": "账号功能", "icon": "1.13"},
		],
	},
	{
		"title": "商城",
		"items": [
			{"label": "游戏充值", "icon": "2.4"},
			{"label": "元宝道具", "icon": "2.4"},
			{"label": "金币道具", "icon": "2.4"},
			{"label": "当前等级特惠商品", "icon": "2.4"},
			{"label": "团包", "emoji": "📦"},
		],
	},
]

@onready var tabs_container: HBoxContainer = $Root/TabsFrame/Content/TabsRow
@onready var items_container: VBoxContainer = $Root/ItemsFrame/Content/ItemsList

var _current_tab_index: int = 0
var _current_item_index: int = 0
var _hovered_tab_index: int = -1
var _hovered_item_index: int = -1
var _tab_labels: Array[Label] = []
var _item_rows: Array[PanelContainer] = []


func _ready() -> void:
	hide()
	call_deferred("_initialize_menu_ui")


func _initialize_menu_ui() -> void:
	if tabs_container == null or items_container == null:
		push_warning("MainMenu UI 节点未就绪，跳过初始化。")
		return
	_build_tabs()
	_refresh_items()


func _unhandled_input(event: InputEvent) -> void:
	if not visible or not (event is InputEventKey):
		return

	var key_event := event as InputEventKey
	if not key_event.pressed or key_event.echo:
		return

	if event.is_action_pressed("ui_left"):
		_switch_tab(-1)
	elif event.is_action_pressed("ui_right"):
		_switch_tab(1)
	elif event.is_action_pressed("ui_up"):
		_move_selection(-1)
	elif event.is_action_pressed("ui_down"):
		_move_selection(1)
	else:
		return

	get_viewport().set_input_as_handled()


func open_menu() -> void:
	show()
	if tabs_container == null or items_container == null:
		return
	_refresh_tabs()
	_refresh_item_selection()


func close_menu() -> void:
	var was_visible := visible
	hide()
	if was_visible:
		menu_closed.emit()


func _build_tabs() -> void:
	if tabs_container == null:
		return

	for child in tabs_container.get_children():
		tabs_container.remove_child(child)
		child.queue_free()

	_tab_labels.clear()
	for index in range(MENU_DATA.size()):
		var category: Dictionary = MENU_DATA[index]
		var tab_label := Label.new()
		tab_label.text = String(category.get("title", ""))
		tab_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		tab_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		tab_label.mouse_filter = Control.MOUSE_FILTER_PASS
		tab_label.add_theme_color_override("font_color", TAB_TEXT_COLOR)
		tab_label.add_theme_color_override("font_outline_color", ITEM_SHADOW_COLOR)
		tab_label.add_theme_constant_override("outline_size", 3)
		tab_label.add_theme_font_size_override("font_size", 12)
		tab_label.gui_input.connect(_on_tab_gui_input.bind(index))
		tab_label.mouse_entered.connect(_on_tab_mouse_entered.bind(index))
		tab_label.mouse_exited.connect(_on_tab_mouse_exited.bind(index))
		tabs_container.add_child(tab_label)
		_tab_labels.append(tab_label)

	_refresh_tabs()


func _refresh_tabs() -> void:
	for index in range(_tab_labels.size()):
		var tab_label := _tab_labels[index]
		if tab_label == null:
			continue
		var title := String(MENU_DATA[index].get("title", ""))
		var is_current := index == _current_tab_index
		var is_hovered := index == _hovered_tab_index
		var font_color := TAB_ACTIVE_COLOR if is_current else TAB_TEXT_COLOR
		if is_hovered and not is_current:
			font_color = ITEM_HOVER_COLOR
		tab_label.add_theme_color_override("font_color", font_color)
		tab_label.text = "【%s】" % title if index == _current_tab_index else title


func _refresh_items() -> void:
	if items_container == null:
		return

	for child in items_container.get_children():
		items_container.remove_child(child)
		child.queue_free()

	_item_rows.clear()
	var items: Array = MENU_DATA[_current_tab_index].get("items", [])
	for item in items:
		var row_panel := PanelContainer.new()
		row_panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row_panel.mouse_filter = Control.MOUSE_FILTER_STOP
		row_panel.add_theme_stylebox_override("panel", _make_row_stylebox(ITEM_IDLE_BACKGROUND))

		var row_box := HBoxContainer.new()
		row_box.alignment = BoxContainer.ALIGNMENT_BEGIN
		row_box.add_theme_constant_override("separation", 6)
		row_panel.add_child(row_box)

		row_box.add_child(_create_item_icon(item))

		var text_label := Label.new()
		text_label.text = String(item.get("label", ""))
		text_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_LEFT
		text_label.size_flags_horizontal = Control.SIZE_SHRINK_BEGIN
		text_label.add_theme_color_override("font_color", ITEM_TEXT_COLOR)
		text_label.add_theme_color_override("font_outline_color", ITEM_SHADOW_COLOR)
		text_label.add_theme_constant_override("outline_size", 3)
		text_label.add_theme_font_size_override("font_size", 12)
		row_box.add_child(text_label)

		row_box.add_child(_create_item_icon(item))

		row_panel.set_meta("text_label", text_label)
		row_panel.mouse_entered.connect(_on_item_mouse_entered.bind(_item_rows.size()))
		row_panel.mouse_exited.connect(_on_item_mouse_exited.bind(_item_rows.size()))
		row_panel.gui_input.connect(_on_item_gui_input.bind(_item_rows.size()))
		items_container.add_child(row_panel)
		_item_rows.append(row_panel)

	_current_item_index = clampi(_current_item_index, 0, max(_item_rows.size() - 1, 0))
	_hovered_item_index = -1
	_refresh_item_selection()


func _refresh_item_selection() -> void:
	for index in range(_item_rows.size()):
		var row_panel := _item_rows[index]
		if row_panel == null:
			continue

		var row_box := row_panel.get_child(0) as HBoxContainer
		var text_label := row_panel.get_meta("text_label", null) as Label
		var is_selected := index == _current_item_index
		var is_hovered := index == _hovered_item_index
		var background_color := ITEM_IDLE_BACKGROUND
		if is_selected:
			background_color = ITEM_ACTIVE_BACKGROUND
		elif is_hovered:
			background_color = ITEM_HOVER_BACKGROUND
		row_panel.add_theme_stylebox_override("panel", _make_row_stylebox(background_color))
		if text_label != null:
			var font_color := ITEM_ACTIVE_COLOR if is_selected else ITEM_TEXT_COLOR
			if is_hovered and not is_selected:
				font_color = ITEM_HOVER_COLOR
			text_label.add_theme_color_override("font_color", font_color)
		if row_box != null:
			for child in row_box.get_children():
				if child is TextureRect:
					var icon_color := ITEM_ACTIVE_COLOR if is_selected else Color.WHITE
					if is_hovered and not is_selected:
						icon_color = ITEM_HOVER_COLOR
					(child as TextureRect).modulate = icon_color
				elif child is Label and child != text_label:
					var icon_label_color := ITEM_ACTIVE_COLOR if is_selected else ITEM_TEXT_COLOR
					if is_hovered and not is_selected:
						icon_label_color = ITEM_HOVER_COLOR
					(child as Label).add_theme_color_override("font_color", icon_label_color)


func _switch_tab(step: int) -> void:
	_current_tab_index = wrapi(_current_tab_index + step, 0, MENU_DATA.size())
	_current_item_index = 0
	_build_tabs()
	_refresh_items()


func _move_selection(step: int) -> void:
	if _item_rows.is_empty():
		return
	_current_item_index = wrapi(_current_item_index + step, 0, _item_rows.size())
	_hovered_item_index = -1
	_refresh_item_selection()


func _create_item_icon(item: Dictionary) -> Control:
	if item.has("emoji"):
		var emoji_label := Label.new()
		emoji_label.text = String(item.get("emoji", ""))
		emoji_label.add_theme_font_size_override("font_size", 14)
		emoji_label.add_theme_color_override("font_outline_color", ITEM_SHADOW_COLOR)
		emoji_label.add_theme_constant_override("outline_size", 2)
		return emoji_label

	var icon_ref := String(item.get("icon", ""))
	var texture := _build_icon_texture(icon_ref)
	var texture_rect := TextureRect.new()
	texture_rect.custom_minimum_size = Vector2(ICON_SIZE, ICON_SIZE)
	texture_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	texture_rect.stretch_mode = TextureRect.STRETCH_KEEP_CENTERED
	texture_rect.texture = texture
	return texture_rect


func _build_icon_texture(icon_ref: String) -> Texture2D:
	var parts := icon_ref.split(".")
	if parts.size() != 2:
		return null

	var sheet_index := int(parts[0])
	var icon_index := int(parts[1]) - 1
	if icon_index < 0:
		return null

	var texture := MENU_ICON_1 if sheet_index == 1 else MENU_ICON_2
	var atlas_texture := AtlasTexture.new()
	atlas_texture.atlas = texture
	atlas_texture.region = Rect2(icon_index * ICON_SIZE, 0, ICON_SIZE, ICON_SIZE)
	return atlas_texture


func _make_row_stylebox(color: Color) -> StyleBoxFlat:
	var style_box := StyleBoxFlat.new()
	style_box.bg_color = color
	style_box.corner_radius_top_left = 4
	style_box.corner_radius_top_right = 4
	style_box.corner_radius_bottom_right = 4
	style_box.corner_radius_bottom_left = 4
	style_box.content_margin_left = 6
	style_box.content_margin_top = 2
	style_box.content_margin_right = 6
	style_box.content_margin_bottom = 2
	return style_box


func _on_tab_gui_input(event: InputEvent, index: int) -> void:
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed:
		_select_tab(index)
		get_viewport().set_input_as_handled()


func _on_tab_mouse_entered(index: int) -> void:
	_hovered_tab_index = index
	_refresh_tabs()


func _on_tab_mouse_exited(_index: int) -> void:
	_hovered_tab_index = -1
	_refresh_tabs()


func _on_item_mouse_entered(index: int) -> void:
	_hovered_item_index = index
	_refresh_item_selection()


func _on_item_mouse_exited(index: int) -> void:
	if _hovered_item_index == index:
		_hovered_item_index = -1
		_refresh_item_selection()


func _on_item_gui_input(event: InputEvent, index: int) -> void:
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed:
		_current_item_index = index
		_refresh_item_selection()
		get_viewport().set_input_as_handled()


func _select_tab(index: int) -> void:
	if index < 0 or index >= MENU_DATA.size():
		return
	_current_tab_index = index
	_current_item_index = 0
	_hovered_item_index = -1
	_build_tabs()
	_refresh_items()
