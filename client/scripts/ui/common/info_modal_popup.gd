class_name InfoModalPopup
extends "res://scripts/ui/common/modal_popup_layer.gd"

## 通用信息模态面板场景路径；玩家/宠物升级等单行摘要弹窗复用此资源。
const SCENE_PATH: String = "res://scenes/ui/common/info_modal_popup.tscn"
## 默认确定按钮文案。
const DEFAULT_CONFIRM_LABEL: String = "确定"
## 默认标题字号。
const DEFAULT_TITLE_FONT_SIZE: int = 10
## 默认正文行字号。
const DEFAULT_CONTENT_FONT_SIZE: int = 10
## 标题标签颜色，与背包物品名称高亮保持一致。
const TITLE_COLOR: Color = Color(1, 0.952941, 0.745098, 1)

## 可选标题标签；无标题时隐藏。
@onready var _title_label: Label = %TitleLabel
## 动态正文行容器。
@onready var _content_list: VBoxContainer = %ContentList
## 底部确定按钮。
@onready var _confirm_button: RuntimeActionButton = %ConfirmButton


## 绑定确定按钮；默认隐藏，由调用方按需打开。
func _ready() -> void:
    super._ready()
    if _confirm_button != null and not _confirm_button.pressed.is_connected(_on_confirm_pressed):
        _confirm_button.pressed.connect(_on_confirm_pressed)


## 展示标题与多行纯文本摘要。
## config 可选键：title_font_size、content_font_size、confirm_label。
## 返回是否已成功调度打开弹窗。
func show_info(title_text: String, lines: Array, config: Dictionary = {}) -> bool:
    var title_font_size: int = int(config.get("title_font_size", DEFAULT_TITLE_FONT_SIZE))
    var content_font_size: int = int(config.get("content_font_size", DEFAULT_CONTENT_FONT_SIZE))
    var confirm_label: String = str(config.get("confirm_label", DEFAULT_CONFIRM_LABEL))
    _apply_title(title_text, title_font_size)
    _rebuild_content_lines(lines, content_font_size)
    if _confirm_button != null:
        _confirm_button.set_button_label(confirm_label)
    _apply_interactive_nodes()
    call_deferred("_open_modal")
    return true


## 展示玩家升级结果；level 为升级后的当前等级，bonus 为服务端属性加成摘要。
## 等级缺失时回退到 GameState 权威快照；无法解析等级时返回 false。
func show_player_level_up(level: int, bonus: Dictionary) -> bool:
    var resolved_level: int = level
    if resolved_level <= 0:
        resolved_level = int(GameState.player_snapshot.get("level", 0))
    if resolved_level <= 0:
        return false
    var hp_gain: int = int(bonus.get("hp_max", 0))
    var atk_gain: int = int(bonus.get("atk", 0))
    var mana_gain: int = int(bonus.get("mana", 0))
    var spd_gain: int = int(bonus.get("spd", 0))
    var lines: Array[String] = [
        "恭喜你升到了%d级" % resolved_level,
        "最大生命值增加：%d" % hp_gain,
        "攻击力增加：%d" % atk_gain,
        "法力增加：%d" % mana_gain,
        "速度增加：%d" % spd_gain,
    ]
    return show_info("", lines, {"content_font_size": 10})


## 展示单只宠物升级结果；pet_name 为展示名，level 为升级后的当前等级。
## 等级缺失时不弹窗并返回 false。
func show_pet_level_up(pet_name: String, level: int, attr_points_gained: int, free_attr_points: int) -> bool:
    if level <= 0:
        return false
    var resolved_name: String = pet_name.strip_edges()
    if resolved_name.is_empty():
        resolved_name = "你的宠物"
    var lines: Array[String] = [
        "升到了 %d 级" % level,
        "获得自由属性点：%d" % attr_points_gained,
        "当前可用自由点：%d" % free_attr_points,
    ]
    return show_info(resolved_name, lines, {
        "title_font_size": 18,
        "content_font_size": 18,
    })


## 关闭信息模态面板。
func close_popup() -> void:
    if not visible:
        return
    _notify_host_suppress_input_leak()
    _close_modal()


## 让面板与确定按钮可交互，其余区域仍沿用基类“点空白关闭”。
func _apply_interactive_nodes() -> void:
    var panel: Control = get_node_or_null("CenterContainer/PanelContainer") as Control
    if panel != null:
        panel.mouse_filter = Control.MOUSE_FILTER_STOP
    if _confirm_button != null:
        _confirm_button.mouse_filter = Control.MOUSE_FILTER_STOP


## 处理确定按钮点击。
func _on_confirm_pressed() -> void:
    _dismiss_modal()


## 刷新标题标签；空标题时不占布局高度。
func _apply_title(title_text: String, title_font_size: int) -> void:
    if _title_label == null:
        return
    var resolved_title: String = title_text.strip_edges()
    _title_label.visible = not resolved_title.is_empty()
    _title_label.text = resolved_title
    _title_label.add_theme_font_size_override("font_size", title_font_size)
    _title_label.add_theme_color_override("font_color", TITLE_COLOR)
    _title_label.add_theme_color_override("font_outline_color", Color(0, 0, 0, 1))
    _title_label.add_theme_constant_override("outline_size", 1)


## 清空并重建正文行。
func _rebuild_content_lines(lines: Array, content_font_size: int) -> void:
    _clear_content_lines()
    for line_variant: Variant in lines:
        var line_text: String = str(line_variant).strip_edges()
        if line_text.is_empty():
            continue
        _append_content_line(line_text, content_font_size)


## 移除旧的正文 Label 节点。
func _clear_content_lines() -> void:
    if _content_list == null:
        return
    for child: Node in _content_list.get_children():
        child.queue_free()


## 追加一行居中正文；含 BBCode 时用 RichTextLabel，否则沿用 Label。
func _append_content_line(text: String, font_size: int) -> void:
    if _content_list == null:
        return
    if RichTextContent.contains_bbcode(text):
        var rich_label: RichTextLabel = RichTextLabel.new()
        rich_label.bbcode_enabled = true
        rich_label.text = text
        rich_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
        rich_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
        rich_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
        rich_label.add_theme_font_size_override("normal_font_size", font_size)
        rich_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
        rich_label.scroll_active = false
        rich_label.fit_content = true
        _content_list.add_child(rich_label)
        return
    var row_label: Label = Label.new()
    row_label.text = text
    row_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
    row_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
    row_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
    row_label.add_theme_font_size_override("font_size", font_size)
    row_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
    _content_list.add_child(row_label)
