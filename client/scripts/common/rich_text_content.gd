class_name RichTextContent
extends RefCounted

## 识别常见 Godot BBCode 标签和项目自定义行内标签，用于判断服务端文案是否已带格式。
const BBCODE_TAG_HINT_REGEX: String = "\\[(?:\\/)?(?:b|i|u|img|item|color(?:=[^\\]]+)?)(?:\\s+[^\\]]+)?\\]"


## 判断文本是否包含 BBCode 标签。
static func contains_bbcode(text: String) -> bool:
    if text.is_empty():
        return false
    var regex: RegEx = RegEx.new()
    if regex.compile(BBCODE_TAG_HINT_REGEX) != OK:
        return false
    return regex.search(text) != null


## 向 RichTextLabel 追加一段文本：纯文本包默认色，已含 BBCode 则原样写入。
static func append_text_segment(label: RichTextLabel, text_content: String, default_color_hex: String) -> void:
    if text_content.is_empty():
        return
    if contains_bbcode(text_content):
        label.append_text(text_content)
        return
    label.append_text("[color=%s]%s[/color]" % [default_color_hex, text_content])


## 将整段 BBCode 写入 RichTextLabel；空文本时不操作。
static func apply_bbcode_text(label: RichTextLabel, bbcode_text: String) -> void:
    if label == null:
        return
    label.clear()
    label.bbcode_enabled = true
    if bbcode_text.strip_edges().is_empty():
        return
    label.text = bbcode_text


## 将数据库配置的系统名称写入名称标签；名称允许使用后台编辑器支持的 BBCode。
static func apply_system_name(label: RichTextLabel, system_name: String) -> void:
    apply_bbcode_text(label, system_name.strip_edges())


## 将玩家自定义名称按纯文本写入富文本标签，避免方括号内容被解析为 BBCode。
static func apply_plain_name(label: RichTextLabel, plain_name: String) -> void:
    if label == null:
        return
    label.clear()
    label.bbcode_enabled = true
    label.text = plain_name.replace("[", "[lb]")
