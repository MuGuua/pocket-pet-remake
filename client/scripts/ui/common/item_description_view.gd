class_name ItemDescriptionView
extends RichTextLabel

## 与服务端 item.ExtractMentionItemIDs 对齐的占位符正则。
const ITEM_TOKEN_REGEX: String = "\\{item:(\\d+)\\}"
## 与服务端 item.ExtractMentionPetIDs 对齐的宠物占位符正则。
const PET_TOKEN_REGEX: String = "\\{pet:(\\d+)\\}"
## 同时匹配物品与宠物占位符，按出现顺序渲染。
const MENTION_TOKEN_REGEX: String = "(\\{item:(\\d+)\\}|\\{pet:(\\d+)\\})"
## 内联物品 icon 的展示边长，与对话面板保持一致。
const ITEM_ICON_SIZE: int = 12
## 介绍原文颜色。
const INTRO_COLOR_HEX: String = "#c7bbb0"
## 装备属性加成行颜色。
const STAT_COLOR_HEX: String = "#82d563"
## 空态引导文案颜色。
const EMPTY_HINT_COLOR_HEX: String = "#9494b8"
## 内联提及物品名称颜色。
const MENTION_NAME_COLOR_HEX: String = "#f0d5b1"


## 根据服务端物品快照刷新介绍区：占位符处内联 icon + 名称，属性行追加在下方。
func apply_item_snapshot(item: Dictionary) -> void:
    clear()
    bbcode_enabled = true
    if item.is_empty():
        _append_colored_text("点击物品或已装备槽位查看详情。", EMPTY_HINT_COLOR_HEX)
        return
    var intro_text: String = BagUiMapper.description(item)
    var item_mention_map: Dictionary = {}
    var pet_mention_map: Dictionary = {}
    _split_mention_maps(item.get("description_mentions", []), item_mention_map, pet_mention_map)
    var wrote_content: bool = false
    if not intro_text.is_empty():
        _append_intro_with_inline_mentions(intro_text, item_mention_map, pet_mention_map)
        wrote_content = true
    wrote_content = _append_bonus_lines(item, wrote_content)
    if not wrote_content:
        _append_colored_text("暂无描述。", EMPTY_HINT_COLOR_HEX)


## 展示空态引导文案。
func apply_empty_hint(hint_text: String) -> void:
    clear()
    bbcode_enabled = true
    _append_colored_text(hint_text, EMPTY_HINT_COLOR_HEX)


## 将服务端 description_mentions 拆成 item_id / pet_id 两个字典。
func _split_mention_maps(
    mentions_variant: Variant,
    item_mention_map: Dictionary,
    pet_mention_map: Dictionary,
) -> void:
    if mentions_variant is not Array:
        return
    var mentions: Array = mentions_variant as Array
    for mention_variant: Variant in mentions:
        if mention_variant is not Dictionary:
            continue
        var mention: Dictionary = mention_variant as Dictionary
        var mentioned_item_id: int = int(mention.get("item_id", 0))
        var mentioned_pet_id: int = int(mention.get("pet_id", 0))
        if mentioned_item_id > 0:
            item_mention_map[mentioned_item_id] = mention
        elif mentioned_pet_id > 0:
            pet_mention_map[mentioned_pet_id] = mention


## 按占位符顺序拆分介绍原文，并在每个 {item:ID}/{pet:ID} 位置插入展示内容。
func _append_intro_with_inline_mentions(
    intro_text: String,
    item_mention_map: Dictionary,
    pet_mention_map: Dictionary,
) -> void:
    var regex: RegEx = RegEx.new()
    regex.compile(MENTION_TOKEN_REGEX)
    var search_offset: int = 0
    while true:
        var match: RegExMatch = regex.search(intro_text, search_offset)
        if match == null:
            if search_offset < intro_text.length():
                _append_colored_text(intro_text.substr(search_offset), INTRO_COLOR_HEX)
            break
        var match_start: int = match.get_start()
        var match_end: int = match.get_end()
        if match_start > search_offset:
            _append_colored_text(intro_text.substr(search_offset, match_start - search_offset), INTRO_COLOR_HEX)
        var mentioned_item_id: int = int(match.get_string(2))
        var mentioned_pet_id: int = int(match.get_string(3))
        if mentioned_item_id > 0:
            _append_inline_item_mention(mentioned_item_id, item_mention_map)
        elif mentioned_pet_id > 0:
            _append_inline_pet_mention(mentioned_pet_id, pet_mention_map)
        search_offset = match_end


## 在占位符处插入单个物品 icon 与名称；名称优先使用服务端权威 item_name。
func _append_inline_item_mention(item_id: int, mention_map: Dictionary) -> void:
    var mention_variant: Variant = mention_map.get(item_id, {})
    var mention: Dictionary = mention_variant if mention_variant is Dictionary else {}
    var item_name: String = str(
        mention.get("item_name", "物品%s" % UiFormat.value_to_text(item_id))
    )
    var icon_texture: Texture2D = ItemIcons.resolve_texture(item_id)
    if icon_texture != null:
        add_image(icon_texture, ITEM_ICON_SIZE, ITEM_ICON_SIZE, Color.WHITE, INLINE_ALIGNMENT_CENTER)
    _append_colored_text(item_name, MENTION_NAME_COLOR_HEX)


## 在 {pet:ID} 占位符处插入宠物名称；宠物暂无独立 icon 资源。
func _append_inline_pet_mention(pet_id: int, mention_map: Dictionary) -> void:
    var mention_variant: Variant = mention_map.get(pet_id, {})
    var mention: Dictionary = mention_variant if mention_variant is Dictionary else {}
    var pet_name: String = str(
        mention.get("item_name", mention.get("pet_name", "宠物%s" % UiFormat.value_to_text(pet_id)))
    )
    _append_colored_text(pet_name, MENTION_NAME_COLOR_HEX)


## 追加装备属性行；若已有介绍内容则先换行。
func _append_bonus_lines(item: Dictionary, wrote_content: bool) -> bool:
    var bonus_lines: PackedStringArray = BagUiMapper.bonus_stat_lines(item)
    if bonus_lines.is_empty():
        return wrote_content
    for line_index: int in bonus_lines.size():
        if wrote_content:
            add_text("\n")
        _append_colored_text(bonus_lines[line_index], STAT_COLOR_HEX)
        wrote_content = true
    return wrote_content


## 追加带颜色的 BBCode 文本片段；若原文已含 BBCode 则不再套默认色。
func _append_colored_text(text_content: String, color_hex: String) -> void:
    if text_content.is_empty():
        return
    RichTextContent.append_text_segment(self, text_content, color_hex)
