class_name ConfirmPromptPopup
extends "res://scripts/ui/common/modal_popup_layer.gd"

## 通用确认提示面板场景路径；调用方通过该路径实例化，避免散落硬编码路径。
const SCENE_PATH: String = "res://scenes/ui/common/confirm_prompt_popup.tscn"
## 默认标题文案；调用方传空标题时使用它，保证面板不会出现空标题栏。
const DEFAULT_TITLE: String = "提示"
## 默认确认按钮文案；当前重构场景底部只有一个继续按钮，语义等同确认。
const DEFAULT_CONFIRM_LABEL: String = "确定"
## 默认正文行字号；动态复制 RichTextLabel 时沿用模板节点，也允许业务通过 config 覆盖。
const DEFAULT_CONTENT_FONT_SIZE: int = 24

## 玩家点击底部继续按钮后广播。
signal confirmed
## 玩家点击右上角关闭或遮罩取消后广播。
signal cancelled

## 标题标签，来自重构后的标题栏真实节点。
@onready var _title_label: Label = get_node_or_null("VBoxContainer/ScrollContainer/Title/HeaderRow/Label") as Label
## 弹窗主体容器；外部可用 set_popup_position 微调整个提示框在屏幕上的位置。
@onready var _panel_root: Control = get_node_or_null("VBoxContainer") as Control
## 正文行容器；每一行都会复制模板 RichTextLabel，支持后端 BBCode 富文本。
@onready var _content_list: VBoxContainer = get_node_or_null("VBoxContainer/CenterContainer/PanelContainer/MarginContainer/RootVBox/ContentList") as VBoxContainer
## 正文富文本模板；运行时隐藏并复制，避免脚本生成完整 UI 结构。
@onready var _rich_line_template: RichTextLabel = get_node_or_null("VBoxContainer/CenterContainer/PanelContainer/MarginContainer/RootVBox/ContentList/MarginContainer/RichLineTemplate") as RichTextLabel
## 模板所在的边距容器；复制它可以保留场景里调好的内边距。
var _rich_line_template_container: Control = null
## 底部继续按钮；点击后按确认处理。
@onready var _continue_button: BaseButton = get_node_or_null("%ContinueButton") as BaseButton

## 绑定底部按钮；弹窗默认隐藏，等待 show_prompt 正式打开。
func _ready() -> void:
    super._ready()
    if _rich_line_template != null:
        _rich_line_template_container = _rich_line_template.get_parent() as Control
    if _rich_line_template_container != null:
        _rich_line_template_container.visible = false
    if _continue_button != null and not _continue_button.pressed.is_connected(_on_continue_pressed):
        _continue_button.pressed.connect(_on_continue_pressed)


## 展示确认提示面板。
## title_text 为标题；content_bbcode 为服务端或业务传入的 BBCode 正文；config 可选 confirm_label、content_font_size。
func show_prompt(title_text: String, content_bbcode: String, config: Dictionary = {}) -> void:
    var resolved_title: String = title_text.strip_edges()
    if resolved_title.is_empty():
        resolved_title = DEFAULT_TITLE
    if _title_label != null:
        _title_label.text = resolved_title
    var content_font_size: int = int(config.get("content_font_size", DEFAULT_CONTENT_FONT_SIZE))
    _rebuild_content_lines(content_bbcode, content_font_size)
    _set_continue_button_label(str(config.get("confirm_label", DEFAULT_CONFIRM_LABEL)))
    _apply_interactive_nodes()
    _open_modal()


## 关闭提示面板，不广播确认；通常由外部强制收起时调用。
func close_prompt() -> void:
    if not visible:
        return
    get_viewport().set_input_as_handled()
    _notify_host_suppress_input_leak()
    _close_modal()


## 设置提示框主体位置；坐标对应场景中 VBoxContainer 的 Control.position，默认是相对屏幕中心锚点的偏移。
func set_popup_position(popup_position: Vector2) -> void:
    if _panel_root == null:
        return
    _panel_root.position = popup_position


## 点击遮罩或右上角关闭时按取消处理，避免误触被当成确认。
func _dismiss_modal() -> void:
    _close_with_result(false)


## 启用确认弹窗专用输入监听，统一处理继续键和移动端按钮热区。
func _enable_modal_input_listeners() -> void:
    set_process_input(true)


## 关闭弹窗后停止接收全局输入，避免影响世界场景和其他面板。
func _disable_modal_input_listeners() -> void:
    set_process_input(false)


## 优先处理 5 键、回车和两个按钮的点击，其余输入只阻断穿透到底层场景。
## event 是 Godot 分发的键盘、鼠标或触摸输入事件。
func _input(event: InputEvent) -> void:
    if not visible or not _is_topmost_runtime_modal():
        return
    if _is_continue_key_event(event):
        get_viewport().set_input_as_handled()
        _close_with_result(true)
        return
    var click_position: Vector2 = _extract_pressed_position(event)
    if click_position == Vector2.INF:
        get_viewport().set_input_as_handled()
        return
    if _is_position_inside_button(_top_close_button, click_position, 24.0):
        get_viewport().set_input_as_handled()
        _close_with_result(false)
        return
    if _is_position_inside_button(_continue_button, click_position, 12.0):
        get_viewport().set_input_as_handled()
        _close_with_result(true)
        return
    get_viewport().set_input_as_handled()


## 判断输入是否为确认继续键，同时兼容主键盘、小键盘和物理键位。
## event 是待判断的输入事件。
func _is_continue_key_event(event: InputEvent) -> bool:
    if not (event is InputEventKey):
        return false
    var key_event: InputEventKey = event as InputEventKey
    if not key_event.pressed or key_event.echo:
        return false
    return (
        key_event.keycode == KEY_5
        or key_event.keycode == KEY_KP_5
        or key_event.keycode == KEY_ENTER
        or key_event.keycode == KEY_KP_ENTER
        or key_event.physical_keycode == KEY_5
        or key_event.physical_keycode == KEY_KP_5
        or key_event.physical_keycode == KEY_ENTER
        or key_event.physical_keycode == KEY_KP_ENTER
    )


## 提取鼠标或触摸按下时的屏幕坐标，其他事件返回无效坐标。
## event 是待解析的输入事件。
func _extract_pressed_position(event: InputEvent) -> Vector2:
    if event is InputEventMouseButton:
        var mouse_event: InputEventMouseButton = event as InputEventMouseButton
        if mouse_event.pressed:
            return mouse_event.position
    if event is InputEventScreenTouch:
        var touch_event: InputEventScreenTouch = event as InputEventScreenTouch
        if touch_event.pressed:
            return touch_event.position
    return Vector2.INF


## 判断点击位置是否落在按钮热区内，并为移动端提供额外可点击边距。
## button 是需要检测的按钮节点。
## click_position 是当前鼠标或触摸的屏幕坐标。
## touch_margin 是按钮矩形向外扩展的像素数。
func _is_position_inside_button(button: BaseButton, click_position: Vector2, touch_margin: float) -> bool:
    if button == null:
        return false
    return button.get_global_rect().grow(touch_margin).has_point(click_position)


## 弹窗打开后短暂忽略关闭，避免移动端同一次触屏穿透导致刚打开就取消。
func _can_dismiss_modal_now() -> bool:
    if _modal_open_frame < 0:
        return true
    return Engine.get_process_frames() > _modal_open_frame + 4


## 让面板和按钮可接收点击，遮罩仍负责空白区域取消。
func _apply_interactive_nodes() -> void:
    var panel: Control = get_node_or_null("VBoxContainer") as Control
    if panel != null:
        panel.mouse_filter = Control.MOUSE_FILTER_STOP
    if _top_close_button != null:
        _top_close_button.mouse_filter = Control.MOUSE_FILTER_STOP
    if _continue_button != null:
        _continue_button.mouse_filter = Control.MOUSE_FILTER_STOP
        _set_button_children_mouse_ignore(_continue_button)


## 让按钮内部贴图、文字和动画只负责显示，不抢占父按钮的点击事件。
## button 是需要整理子节点鼠标过滤规则的按钮。
func _set_button_children_mouse_ignore(button: BaseButton) -> void:
    for child: Node in button.get_children():
        _apply_mouse_filter_recursive(child, true)


## 清空旧正文并按换行重建富文本行。
func _rebuild_content_lines(content_bbcode: String, font_size: int) -> void:
    if _content_list == null or _rich_line_template == null or _rich_line_template_container == null:
        return
    _clear_content_lines()
    var normalized_content: String = content_bbcode.strip_edges()
    if normalized_content.is_empty():
        normalized_content = " "
    var lines: PackedStringArray = normalized_content.split("\n", true)
    for line_index: int in range(lines.size()):
        var line_text: String = lines[line_index]
        if line_index == 0:
            _rich_line_template_container.visible = true
            _apply_content_line_label(_rich_line_template, line_text, font_size)
        else:
            _append_content_line(line_text, font_size)


## 移除上一次打开时复制出的正文行，保留场景模板节点。
func _clear_content_lines() -> void:
    if _content_list == null:
        return
    for child: Node in _content_list.get_children():
        if child == _rich_line_template_container or child == _continue_button:
            continue
        _content_list.remove_child(child)
        child.queue_free()


## 复制模板追加一行富文本，确保技能描述里的 BBCode 由 RichTextLabel 渲染。
func _append_content_line(line_text: String, font_size: int) -> void:
    if _content_list == null or _rich_line_template_container == null:
        return
    var line_container: Control = _rich_line_template_container.duplicate() as Control
    if line_container == null:
        return
    var line_label: RichTextLabel = line_container.get_node_or_null("RichLineTemplate") as RichTextLabel
    if line_label == null:
        return
    line_container.visible = true
    _apply_content_line_label(line_label, line_text, font_size)
    _content_list.add_child(line_container)
    if _continue_button != null:
        _content_list.move_child(_continue_button, _content_list.get_child_count() - 1)


## 将一行服务端 BBCode 写入 RichLineTemplate 样式的富文本节点。
func _apply_content_line_label(line_label: RichTextLabel, line_text: String, font_size: int) -> void:
    if line_label == null:
        return
    line_label.visible = true
    line_label.bbcode_enabled = true
    line_label.text = line_text if not line_text.is_empty() else " "
    line_label.fit_content = true
    line_label.scroll_active = false
    line_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
    line_label.add_theme_font_size_override("normal_font_size", font_size)


## 更新继续按钮里的展示文案；兼容 continue_button.tscn 当前内部 Label 结构。
func _set_continue_button_label(label_text: String) -> void:
    if _continue_button == null:
        return
    var resolved_label: String = label_text.strip_edges()
    if resolved_label.is_empty():
        resolved_label = DEFAULT_CONFIRM_LABEL
    _continue_button.text = resolved_label
    var nested_label: Label = _continue_button.get_node_or_null("TextureRect/Control/Control") as Label
    if nested_label != null:
        nested_label.text = resolved_label


## 处理底部继续按钮点击。
func _on_continue_pressed() -> void:
    _close_with_result(true)


## 根据玩家选择关闭面板并广播结果；先发信号再隐藏，方便调用方立即读取结果。
func _close_with_result(is_confirmed: bool) -> void:
    if not visible:
        return
    get_viewport().set_input_as_handled()
    _notify_host_suppress_input_leak()
    if is_confirmed:
        confirmed.emit()
    else:
        cancelled.emit()
    popup_closed.emit()
    _close_modal()
