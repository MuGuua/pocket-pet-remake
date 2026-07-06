class_name OptionListPanel
extends RuntimeRootPanel

## 通用选项列表面板场景路径；NPC 交互菜单、附近 NPC 列表、PVP 目标列表等流程复用。
const SCENE_PATH: String = "res://scenes/ui/common/option_list_panel.tscn"
## 渲染 NPC 菜单项：类型前缀 + 标题 + 副标题 + 状态。
const RENDER_NPC_ENTRY: String = "npc_entry"
## 渲染头像 + 名称行。
const RENDER_PORTRAIT_TEXT: String = "portrait_text"
## NPC 菜单项类型前缀文案。
const ENTRY_TYPE_LABELS: Dictionary = {
	"quest": "[任务]",
	"dialog": "[对话]",
	"shop": "[商店]",
	"battle": "[战斗]",
	"warehouse": "[仓库]",
	"teleport": "[传送]",
	"craft": "[制作]",
	"event": "[事件]",
}
## NPC 菜单项状态文案。
const STATE_LABELS: Dictionary = {
	"locked": "未解锁",
	"completed": "已完成",
	"in_progress": "进行中",
	"available": "",
}
## 不可点击的 NPC 菜单项状态。
const DISABLED_STATES: Array[String] = ["locked", "completed"]
## 默认面板最小宽度（像素）。
const DEFAULT_PANEL_MIN_WIDTH: int = 264
## 选项滚动区最大高度占视口高度的比例；超出后启用竖向滚动。
const MAX_OPTIONS_SCROLL_VIEWPORT_RATIO: float = 0.42
## 选项滚动区最小高度（像素）；避免只有 1 项时区域过扁。
const MIN_OPTIONS_SCROLL_HEIGHT: float = 24.0
## 场景中预置的最大选项行数量；需要更多行时直接在 option_list_panel.tscn 里追加节点。
const MAX_PRESET_OPTION_ROWS: int = 30

## 玩家选中某个选项后向外广播完整选项快照。
signal option_selected(option: Dictionary)
## 兼容旧 NPC 列表监听方使用的信号别名。
signal npc_selected(option: Dictionary)

## 面板容器，用于按场景调整最小宽度。
@onready var _panel: PanelContainer = $PanelContainer
## 标题标签。
@onready var _title_label: Label = %TitleLabel
## 标题行右上角关闭按钮。
@onready var _top_close_button: BaseButton = %TopCloseButton
## 选项列表滚动容器。
@onready var _options_scroll: ScrollContainer = %OptionsScroll
## 预置选项行容器。
@onready var _buttons_container: VBoxContainer = %ButtonsContainer
## 右下角关闭按钮所在行。
@onready var _close_row: HBoxContainer = %CloseRow
## 通用样式关闭按钮。
@onready var _close_button: RuntimeActionButton = %CloseButton

## 当前选项索引到原始数据的映射；按钮点击时用场景节点序号反查服务端选项数据。
var _option_lookup: Dictionary = {}
## 场景中预置的选项行；每行由 HBoxContainer + Portrait + OptionButton 组成。
var _option_rows: Array[HBoxContainer] = []
## 场景中预置的头像节点；NPC 菜单模式隐藏，头像列表模式按数据展示。
var _option_portraits: Array[TextureRect] = []
## 场景中预置的文字按钮；只在运行时写入服务端返回的文本，不再动态创建按钮。
var _option_buttons: Array[Button] = []


## 初始化面板节点引用，并绑定右下角关闭按钮。
func _ready() -> void:
	super._ready()
	_collect_option_row_nodes()
	if _close_button != null:
		_close_button.set_button_label("关闭")
	_connect_button_pressed(_close_button, close_menu)
	_connect_button_pressed(_top_close_button, close_menu)


## 收集 option_list_panel.tscn 中预置的选项行，并一次性绑定按钮点击信号。
func _collect_option_row_nodes() -> void:
	_option_rows.clear()
	_option_portraits.clear()
	_option_buttons.clear()
	if _buttons_container == null:
		return
	for index: int in range(1, MAX_PRESET_OPTION_ROWS + 1):
		var row: HBoxContainer = _buttons_container.get_node_or_null("OptionRow%d" % index) as HBoxContainer
		if row == null:
			continue
		var portrait: TextureRect = row.get_node_or_null("Portrait") as TextureRect
		var button: Button = row.get_node_or_null("OptionButton") as Button
		if button == null:
			continue
		_option_rows.append(row)
		_option_portraits.append(portrait)
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
## config 可选键：render_mode、panel_min_width、show_close_button。
func configure(title: String, options: Array, config: Dictionary = {}) -> void:
	var render_mode: String = str(config.get("render_mode", RENDER_NPC_ENTRY))
	var panel_min_width: int = int(config.get("panel_min_width", DEFAULT_PANEL_MIN_WIDTH))
	var show_close_button: bool = bool(config.get("show_close_button", true))
	if _title_label != null:
		_title_label.text = title
	if _panel != null and panel_min_width > 0:
		_panel.custom_minimum_size = Vector2(float(panel_min_width), 0.0)
	_set_close_button_visible(show_close_button)
	_option_lookup.clear()
	_reset_option_rows()
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
	for index: int in range(_option_rows.size()):
		var row: HBoxContainer = _option_rows[index]
		var portrait: TextureRect = _option_portraits[index]
		var button: Button = _option_buttons[index]
		if row != null:
			row.visible = false
		if portrait != null:
			portrait.visible = false
			portrait.texture = null
		if button != null:
			button.text = ""
			button.disabled = true


## 构建 NPC 菜单项按钮列表。
func _build_npc_entry_options(options: Array) -> void:
	var slot_index: int = 0
	for option_variant: Variant in options:
		if option_variant is not Dictionary:
			continue
		if slot_index >= _option_rows.size():
			push_warning("OptionListPanel: 预置选项行不足，已忽略多余 NPC 菜单项。")
			return
		var option_data: Dictionary = (option_variant as Dictionary).duplicate(true)
		var row: HBoxContainer = _option_rows[slot_index]
		var portrait: TextureRect = _option_portraits[slot_index]
		var button: Button = _option_buttons[slot_index]
		if row == null or button == null:
			continue
		row.visible = true
		if portrait != null:
			portrait.visible = false
			portrait.texture = null
		button.text = _format_npc_entry_label(option_data)
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
		if slot_index >= _option_rows.size():
			push_warning("OptionListPanel: 预置选项行不足，已忽略多余头像列表项。")
			return
		var option_data: Dictionary = option_variant as Dictionary
		var row: HBoxContainer = _option_rows[slot_index]
		var portrait_rect: TextureRect = _option_portraits[slot_index]
		var button: Button = _option_buttons[slot_index]
		if row == null or button == null:
			continue
		row.visible = true
		if portrait_rect != null:
			portrait_rect.texture = _resolve_portrait_texture(option_data)
			portrait_rect.visible = portrait_rect.texture != null
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


## 控制右下角关闭按钮行的显隐。
func _set_close_button_visible(should_show: bool) -> void:
	if _close_row != null:
		_close_row.visible = should_show
		return
	if _close_button != null:
		_close_button.visible = should_show


## 打开面板后聚焦第一个可交互按钮。
func _focus_first_button() -> void:
	for index: int in range(_option_rows.size()):
		var row: HBoxContainer = _option_rows[index]
		var button: Button = _option_buttons[index]
		if row != null and row.visible and button != null and not button.disabled:
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


## 把类型前缀、标题、副标题和状态标签拼成移动端可读按钮文案。
func _format_npc_entry_label(option: Dictionary) -> String:
	var entry_type: String = str(option.get("entry_type", ""))
	var title_text: String = str(option.get("title", option.get("label", option.get("id", ""))))
	var subtitle_text: String = str(option.get("subtitle", "")).strip_edges()
	var state_text: String = str(STATE_LABELS.get(str(option.get("state", "available")), ""))
	var type_prefix: String = str(ENTRY_TYPE_LABELS.get(entry_type, "[交互]"))
	var label_text: String = "%s %s" % [type_prefix, title_text]
	if not subtitle_text.is_empty():
		label_text += "\n%s" % subtitle_text
	if not state_text.is_empty():
		label_text += " (%s)" % state_text
	return label_text


## 解析选项展示名称。
func _resolve_display_name(option: Dictionary) -> String:
	return str(option.get("npc_name", option.get("name", "未知")))


## 按 portrait_path 加载头像纹理。
func _resolve_portrait_texture(option: Dictionary) -> Texture2D:
	var portrait_path: String = str(option.get("portrait_path", ""))
	if portrait_path.is_empty():
		return null
	var portrait: Variant = load(portrait_path)
	if portrait is Texture2D:
		return portrait as Texture2D
	return null
