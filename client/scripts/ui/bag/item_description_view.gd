class_name ItemDescriptionView
extends RichTextLabel

## 与服务端 item.ExtractMentionItemIDs 对齐的占位符正则。
const ITEM_TOKEN_REGEX: String = "\\{item:(\\d+)\\}"
## 内联物品 icon 的展示边长，与对话面板保持一致。
const ITEM_ICON_SIZE: int = 12


## 根据服务端物品快照刷新介绍区：占位符处内联 icon + 名称，属性行追加在下方。
func apply_item_snapshot(item: Dictionary) -> void:
	clear()
	bbcode_enabled = false
	if item.is_empty():
		add_text("点击物品或已装备槽位查看详情。")
		return
	var intro_text: String = BagUiMapper.description(item)
	var mention_map: Dictionary = _build_mention_map(item.get("description_mentions", []))
	var wrote_content: bool = false
	if not intro_text.is_empty():
		_append_intro_with_inline_items(intro_text, mention_map)
		wrote_content = true
	wrote_content = _append_bonus_lines(item, wrote_content)
	if not wrote_content:
		add_text("暂无描述。")


## 展示空态引导文案。
func apply_empty_hint(hint_text: String) -> void:
	clear()
	bbcode_enabled = false
	add_text(hint_text)


## 将服务端 description_mentions 数组转为 item_id -> mention 字典，便于占位符查找。
func _build_mention_map(mentions_variant: Variant) -> Dictionary:
	var result: Dictionary = {}
	if mentions_variant is not Array:
		return result
	var mentions: Array = mentions_variant as Array
	for mention_variant: Variant in mentions:
		if mention_variant is not Dictionary:
			continue
		var mention: Dictionary = mention_variant as Dictionary
		var mentioned_item_id: int = int(mention.get("item_id", 0))
		if mentioned_item_id <= 0:
			continue
		result[mentioned_item_id] = mention
	return result


## 按占位符顺序拆分介绍原文，并在每个 {item:ID} 位置插入 icon 与物品名。
func _append_intro_with_inline_items(intro_text: String, mention_map: Dictionary) -> void:
	var regex: RegEx = RegEx.new()
	regex.compile(ITEM_TOKEN_REGEX)
	var search_offset: int = 0
	while true:
		var match: RegExMatch = regex.search(intro_text, search_offset)
		if match == null:
			if search_offset < intro_text.length():
				add_text(intro_text.substr(search_offset))
			break
		var match_start: int = match.get_start()
		var match_end: int = match.get_end()
		if match_start > search_offset:
			add_text(intro_text.substr(search_offset, match_start - search_offset))
		var mentioned_item_id: int = int(match.get_string(1))
		_append_inline_item_mention(mentioned_item_id, mention_map)
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
	add_text(item_name)


## 追加装备属性行；若已有介绍内容则先换行。
func _append_bonus_lines(item: Dictionary, wrote_content: bool) -> bool:
	var bonus_lines: PackedStringArray = BagUiMapper.bonus_stat_lines(item)
	if bonus_lines.is_empty():
		return wrote_content
	for line_index: int in bonus_lines.size():
		if wrote_content:
			add_text("\n")
		add_text(bonus_lines[line_index])
		wrote_content = true
	return wrote_content
