class_name OptionListPanel
extends RuntimeRootPanel

## 通用选项列表面板场景路径；NPC 交互菜单、附近 NPC 列表、PVP 目标列表等流程复用。
const SCENE_PATH: String = "res://scenes/ui/common/option_list_panel.tscn"
## 渲染 NPC 菜单项：数字序号 + 服务端标题。
const RENDER_NPC_ENTRY: String = "npc_entry"
## 渲染头像 + 名称行。
const RENDER_PORTRAIT_TEXT: String = "portrait_text"
## 不可点击的 NPC 菜单项状态。
const DISABLED_STATES: Array[String] = ["locked", "completed"]
## 选项滚动区最大高度占视口高度的比例；超出后启用竖向滚动。
const MAX_OPTIONS_SCROLL_VIEWPORT_RATIO: float = 0.42
## 选项滚动区最小高度（像素）；避免只有 1 项时区域过扁。
const MIN_OPTIONS_SCROLL_HEIGHT: float = 24.0

## 玩家选中某个选项后向外广播完整选项快照。
signal option_selected(option: Dictionary)
## 兼容旧 NPC 列表监听方使用的信号别名。
signal npc_selected(option: Dictionary)

## NPC 菜单标题旁的上半身形象。
@onready var _header_portrait: TextureRect = %HeaderPortrait
## 标题标签。
@onready var _title_label: Label = %TitleLabel
## 标题行右上角关闭按钮。
@onready var _top_close_button: BaseButton = %TopCloseButton
## 选项列表滚动容器。
@onready var _options_scroll: ScrollContainer = %OptionsScroll
## 预置选项行容器。
@onready var _buttons_container: VBoxContainer = %ButtonsContainer
## 当前选项索引到原始数据的映射；按钮点击时用场景节点序号反查服务端选项数据。
var _option_lookup: Dictionary = {}
## 场景中预置的选项按钮；每个按钮都是 option_row.tscn 的场景实例。
var _option_buttons: Array[Button] = []


## 初始化面板节点引用，并绑定标题行右上角关闭按钮。
func _ready() -> void:
	super._ready()
	_collect_option_row_nodes()
	_connect_button_pressed(_top_close_button, close_menu)


## 收集 ButtonsContainer 下预置的 option_row 按钮，并一次性绑定点击信号。
## 直接按节点树顺序收集，不依赖按钮名称，后续在场景中调整命名或顺序时无需同步修改脚本。
func _collect_option_row_nodes() -> void:
	_option_buttons.clear()
	if _buttons_container == null:
		return
	for child: Node in _buttons_container.get_children():
		var button: Button = child as Button
		if button == null:
			continue
		_option_buttons.append(button)
		var slot_index: int = _option_buttons.size() - 1
		_connect_button_pressed(button, _on_preset_option_button_pressed.bind(slot_index))
	_reset_option_rows()


## 安全绑定按钮 pressed 信号；预置节点调整过程中若节点缺失，不会因空实例访问中断面板初始化。
func _connect_button_pressed(button: BaseButton, handler: Callable) -> void:
	if button == null:
		return
	if not button.pressed.is_connected(handler):
		button.pressed.connect(handler)


## 刷新标题与选项列表。
## config 可选键：render_mode、show_close_button、header_portrait；面板尺寸统一由场景节点维护。
func configure(title: String, options: Array, config: Dictionary = {}) -> void:
	var render_mode: String = str(config.get("render_mode", RENDER_NPC_ENTRY))
	var show_close_button: bool = bool(config.get("show_close_button", true))
	var header_portrait_variant: Variant = config.get("header_portrait", null)
	var source_header_portrait: Texture2D = header_portrait_variant as Texture2D if header_portrait_variant is Texture2D else null
	var header_portrait: Texture2D = _build_upper_body_texture(source_header_portrait)
	_option_lookup.clear()
	_reset_option_rows()
	if _header_portrait != null:
		_header_portrait.texture = header_portrait
		_header_portrait.visible = render_mode == RENDER_NPC_ENTRY and header_portrait != null
	if _title_label != null:
		_title_label.text = title
	_set_close_button_visible(show_close_button)
	match render_mode:
		RENDER_PORTRAIT_TEXT:
			_build_portrait_text_options(options)
		_:
			_build_npc_entry_options(options)
	_focus_first_button()
	_schedule_options_scroll_refresh()


## 打开选项列表面板。
func open_menu() -> void:
	super.open_menu()
	_schedule_options_scroll_refresh()


## 关闭选项列表面板。
func close_menu() -> void:
	super.close_menu()


## 重置所有预置选项行；这里只改变显隐和文本，不删除节点，方便编辑器中继续调整布局。
func _reset_option_rows() -> void:
	if _header_portrait != null:
		_header_portrait.texture = null
		_header_portrait.visible = false
	for button: Button in _option_buttons:
		if button == null:
			continue
		button.visible = false
		button.icon = null
		button.text = ""
		button.disabled = true


## 构建 NPC 菜单项按钮列表。
func _build_npc_entry_options(options: Array) -> void:
	var slot_index: int = 0
	for option_variant: Variant in options:
		if option_variant is not Dictionary:
			continue
		if slot_index >= _option_buttons.size():
			push_warning("OptionListPanel: 预置选项按钮不足，已忽略多余 NPC 菜单项。")
			return
		var option_data: Dictionary = (option_variant as Dictionary).duplicate(true)
		var button: Button = _option_buttons[slot_index]
		if button == null:
			continue
		button.visible = true
		button.icon = null
		button.text = _format_npc_entry_label(option_data, slot_index + 1)
		var entry_state: String = str(option_data.get("state", "available"))
		button.disabled = DISABLED_STATES.has(entry_state)
		_option_lookup[str(slot_index)] = option_data
		slot_index += 1


## 构建头像 + 名称选项行。
func _build_portrait_text_options(options: Array) -> void:
	var slot_index: int = 0
	for option_variant: Variant in options:
		if option_variant is not Dictionary:
			continue
		if slot_index >= _option_buttons.size():
			push_warning("OptionListPanel: 预置选项按钮不足，已忽略多余头像列表项。")
			return
		var option_data: Dictionary = option_variant as Dictionary
		var button: Button = _option_buttons[slot_index]
		if button == null:
			continue
		button.visible = true
		button.icon = _resolve_portrait_texture(option_data)
		button.text = _resolve_display_name(option_data)
		button.disabled = false
		_option_lookup[str(slot_index)] = option_data.duplicate(true)
		slot_index += 1


## 等待布局稳定后再测量选项高度，避免滚动区尺寸计算不准。
func _schedule_options_scroll_refresh() -> void:
	call_deferred("_refresh_options_scroll_layout_deferred")


## 延迟一帧刷新滚动区，确保按钮最小尺寸已参与布局。
func _refresh_options_scroll_layout_deferred() -> void:
	if not is_inside_tree():
		return
	await get_tree().process_frame
	_refresh_options_scroll_layout()


## 按选项内容高度与视口比例刷新滚动区；超出最大高度时启用竖向滚动。
func _refresh_options_scroll_layout() -> void:
	if _options_scroll == null or _buttons_container == null:
		return
	var content_height: float = _buttons_container.get_combined_minimum_size().y
	if content_height <= 0.0:
		content_height = _buttons_container.size.y
	if content_height <= 0.0:
		_options_scroll.custom_minimum_size = Vector2.ZERO
		_options_scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
		_options_scroll.scroll_vertical = 0
		return
	var viewport_height: float = get_viewport().get_visible_rect().size.y
	var max_scroll_height: float = viewport_height * MAX_OPTIONS_SCROLL_VIEWPORT_RATIO
	var scroll_height: float = minf(content_height, max_scroll_height)
	scroll_height = maxf(scroll_height, MIN_OPTIONS_SCROLL_HEIGHT)
	_options_scroll.custom_minimum_size = Vector2(0.0, scroll_height)
	if content_height > scroll_height + 1.0:
		_options_scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_AUTO
	else:
		_options_scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
		_options_scroll.scroll_vertical = 0


## 控制标题行右上角关闭按钮的显隐。
func _set_close_button_visible(should_show: bool) -> void:
	if _top_close_button != null:
		_top_close_button.visible = should_show


## 打开面板后聚焦第一个可交互按钮。
func _focus_first_button() -> void:
	for button: Button in _option_buttons:
		if button != null and button.visible and not button.disabled:
			button.grab_focus()
			return


## 转发预置按钮点击。
func _on_preset_option_button_pressed(slot_index: int) -> void:
	var option_variant: Variant = _option_lookup.get(str(slot_index), {})
	if option_variant is not Dictionary:
		return
	_emit_option_selected(option_variant as Dictionary)


## 同时广播 option_selected 与 npc_selected，兼容旧监听方。
func _emit_option_selected(option: Dictionary) -> void:
	var payload: Dictionary = option.duplicate(true)
	option_selected.emit(payload)
	npc_selected.emit(payload)


## 只使用服务端返回的标题生成带数字序号的按钮文案，不展示简单描述或状态文案。
## option 是单个服务端菜单项，display_index 是从 1 开始的展示序号。
func _format_npc_entry_label(option: Dictionary, display_index: int) -> String:
	var title_text: String = str(option.get("title", option.get("label", option.get("id", ""))))
	return "%d. %s" % [display_index, title_text]


## 从 NPC 场景当前帧生成上半身预览纹理，标题区只展示头部和上半身。
## source_texture 是当前场景 NPC 的完整待机帧；纹理为空时返回空值。
func _build_upper_body_texture(source_texture: Texture2D) -> Texture2D:
	if source_texture == null:
		return null
	var source_image: Image = source_texture.get_image()
	if source_image == null or source_image.is_empty():
		return null
	var source_width: int = source_image.get_width()
	var source_height: int = source_image.get_height()
	var upper_body_height: int = maxi(1, ceili(float(source_height) * 0.68))
	var upper_body_image: Image = source_image.get_region(Rect2i(0, 0, source_width, upper_body_height))
	return ImageTexture.create_from_image(upper_body_image)


## 解析选项展示名称。
func _resolve_display_name(option: Dictionary) -> String:
	return str(option.get("npc_name", option.get("name", "未知")))


## 优先按服务端 skin_id 解析形象首帧，再兼容旧 portrait_path 纹理。
func _resolve_portrait_texture(option: Dictionary) -> Texture2D:
	var skin_id: String = str(option.get("skin_id", "")).strip_edges()
	if not skin_id.is_empty():
		var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(skin_id)
		if skin != null:
			var preview_texture: Texture2D = skin.resolve_avatar_preview_texture()
			if preview_texture != null:
				return preview_texture
	var portrait_path: String = str(option.get("portrait_path", ""))
	if portrait_path.is_empty():
		return null
	var portrait: Variant = load(portrait_path)
	if portrait is Texture2D:
		return portrait as Texture2D
	return null
