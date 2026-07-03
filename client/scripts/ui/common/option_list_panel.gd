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
## NPC 菜单项按钮高度（像素）；比 DialogueActionButtonTheme 默认 34 更紧凑。
const NPC_ENTRY_BUTTON_HEIGHT: float = 16.0
## NPC 菜单项按钮字号（像素）。
const NPC_ENTRY_FONT_SIZE: int = 8
## 头像行按钮默认高度（像素）。
const PORTRAIT_ROW_HEIGHT: float = 22.0
## 头像展示尺寸（像素）。
const PORTRAIT_SIZE: float = 32.0

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
## 动态选项按钮容器。
@onready var _buttons_container: VBoxContainer = %ButtonsContainer
## 右下角关闭按钮所在行。
@onready var _close_row: HBoxContainer = %CloseRow
## 通用样式关闭按钮。
@onready var _close_button: RuntimeActionButton = %CloseButton

## 当前选项索引到原始数据的映射；主要用于 portrait_text 模式。
var _option_lookup: Dictionary = {}


## 初始化面板节点引用，并绑定右下角关闭按钮。
func _ready() -> void:
    super._ready()
    if _close_button != null:
        _close_button.set_button_label("关闭")
        if not _close_button.pressed.is_connected(close_menu):
            _close_button.pressed.connect(close_menu)
    if _top_close_button != null and not _top_close_button.pressed.is_connected(close_menu):
        _top_close_button.pressed.connect(close_menu)


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
    _clear_buttons()
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


## 清空选项按钮容器。
func _clear_buttons() -> void:
    if _buttons_container == null:
        return
    for child: Node in _buttons_container.get_children():
        _buttons_container.remove_child(child)
        child.queue_free()


## 构建 NPC 菜单项按钮列表。
func _build_npc_entry_options(options: Array) -> void:
    for option_variant: Variant in options:
        if option_variant is not Dictionary:
            continue
        var option_data: Dictionary = (option_variant as Dictionary).duplicate(true)
        var button: Button = Button.new()
        button.text = _format_npc_entry_label(option_data)
        button.alignment = HORIZONTAL_ALIGNMENT_LEFT
        DialogueActionButtonTheme.apply(button, true)
        _apply_npc_entry_button_style(button)
        var entry_state: String = str(option_data.get("state", "available"))
        if DISABLED_STATES.has(entry_state):
            button.disabled = true
        if not button.pressed.is_connected(_on_option_button_pressed.bind(option_data)):
            button.pressed.connect(_on_option_button_pressed.bind(option_data))
        _buttons_container.add_child(button)


## 构建头像 + 名称选项行。
func _build_portrait_text_options(options: Array) -> void:
    for index: int in range(options.size()):
        var option_variant: Variant = options[index]
        if option_variant is not Dictionary:
            continue
        var option_data: Dictionary = option_variant as Dictionary
        var option_id: String = str(index)
        var item_container: HBoxContainer = HBoxContainer.new()
        item_container.alignment = BoxContainer.ALIGNMENT_CENTER
        item_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
        var portrait_rect: TextureRect = TextureRect.new()
        portrait_rect.custom_minimum_size = Vector2(PORTRAIT_SIZE, PORTRAIT_SIZE)
        portrait_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
        portrait_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
        portrait_rect.texture = _resolve_portrait_texture(option_data)
        item_container.add_child(portrait_rect)
        var button: Button = Button.new()
        button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
        button.alignment = HORIZONTAL_ALIGNMENT_LEFT
        button.text = _resolve_display_name(option_data)
        button.focus_mode = Control.FOCUS_ALL
        button.custom_minimum_size = Vector2(0.0, PORTRAIT_ROW_HEIGHT)
        button.add_theme_font_size_override("font_size", NPC_ENTRY_FONT_SIZE)
        if not button.pressed.is_connected(_on_portrait_option_pressed.bind(option_id)):
            button.pressed.connect(_on_portrait_option_pressed.bind(option_id))
        item_container.add_child(button)
        _buttons_container.add_child(item_container)
        _option_lookup[option_id] = option_data.duplicate(true)


## 把 NPC 菜单项按钮压到更矮的字号布局；覆盖共用主题里的默认高度与字号。
func _apply_npc_entry_button_style(button: Button) -> void:
    button.add_theme_font_size_override("font_size", NPC_ENTRY_FONT_SIZE)
    button.custom_minimum_size = Vector2(0.0, NPC_ENTRY_BUTTON_HEIGHT)


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
    if _buttons_container == null or _buttons_container.get_child_count() <= 0:
        return
    var first_item: Node = _buttons_container.get_child(0)
    if first_item is Button:
        (first_item as Button).grab_focus()
        return
    if first_item is HBoxContainer and first_item.get_child_count() > 1:
        var first_button: Button = first_item.get_child(1) as Button
        if first_button != null:
            first_button.grab_focus()


## 转发 NPC 菜单项点击。
func _on_option_button_pressed(option: Dictionary) -> void:
    _emit_option_selected(option)


## 转发头像列表项点击。
func _on_portrait_option_pressed(option_id: String) -> void:
    var option_variant: Variant = _option_lookup.get(option_id, {})
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
