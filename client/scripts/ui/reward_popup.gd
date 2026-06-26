extends "res://scripts/ui/modal_popup_layer.gd"


## 奖励列表容器。
@onready var _content_list: VBoxContainer = %ContentList

## 正文与数字字号。
const CONTENT_FONT_SIZE: int = 10
## 数字高亮色，对应 RGB(130, 213, 99)。
const REWARD_NUMBER_COLOR: String = "#82D563"
## 与服务端钱包拆分一致：1000 铜 = 1 银，100 万铜 = 1 金。
const COPPER_PER_SILVER: int = 1000
const COPPER_PER_GOLD: int = 1000000
const ITEM_ICON_SIZE: int = 32


## 展示奖励列表；无有效奖励时不弹窗。
func show_rewards(title_text: String, rewards: Array, pet_rewards: Array = []) -> void:
	var parsed: Dictionary = _parse_rewards(rewards, pet_rewards)
	if not _has_visible_reward(parsed, title_text):
		return
	_clear_reward_rows()
	if not title_text.is_empty():
		_append_plain_line(title_text)
	var player_exp: int = int(parsed.get("player_exp", 0))
	if player_exp > 0:
		_append_rich_line(_format_player_exp_line(player_exp))
	var gold_copper: int = int(parsed.get("gold_copper", 0))
	if gold_copper > 0:
		var gold_line: String = _format_gold_reward_line(gold_copper)
		if not gold_line.is_empty():
			_append_rich_line(gold_line)
	var pet_exp: int = int(parsed.get("pet_exp", 0))
	if pet_exp > 0:
		_append_rich_line(_format_pet_exp_line(pet_exp))
	var item_rewards: Array = parsed.get("items", []) as Array
	if not item_rewards.is_empty():
		_append_plain_line("你得到了以下物品：")
		_append_item_icon_row(item_rewards)
	_open_modal()


## 关闭奖励弹窗。
func close_popup() -> void:
	_close_modal()


## 解析奖励数组，拆出经验、货币与物品三类展示数据。
func _parse_rewards(rewards: Array, pet_rewards: Array) -> Dictionary:
	var player_exp: int = 0
	var gold_copper: int = 0
	var item_rewards: Array[Dictionary] = []
	for reward_variant: Variant in rewards:
		if reward_variant is not Dictionary:
			continue
		var reward: Dictionary = reward_variant as Dictionary
		var reward_type: String = str(reward.get("type", ""))
		match reward_type:
			"exp":
				player_exp += int(reward.get("value", 0))
			"gold":
				gold_copper += int(reward.get("value", 0))
			"item":
				var item_count: int = int(reward.get("count", 0))
				if item_count <= 0:
					continue
				item_rewards.append(reward.duplicate(true))
	return {
		"player_exp": player_exp,
		"gold_copper": gold_copper,
		"pet_exp": _sum_pet_exp(pet_rewards),
		"items": item_rewards,
	}


## 汇总所有参战宠物的经验奖励。
func _sum_pet_exp(pet_rewards: Array) -> int:
	var total_exp: int = 0
	for pet_reward_variant: Variant in pet_rewards:
		if pet_reward_variant is not Dictionary:
			continue
		var pet_reward: Dictionary = pet_reward_variant as Dictionary
		total_exp += int(pet_reward.get("exp_gained", pet_reward.get("exp", 0)))
	return total_exp


## 判断是否至少有一条可展示内容。
func _has_visible_reward(parsed: Dictionary, title_text: String) -> bool:
	if not title_text.is_empty():
		return true
	if int(parsed.get("player_exp", 0)) > 0:
		return true
	if int(parsed.get("gold_copper", 0)) > 0:
		return true
	if int(parsed.get("pet_exp", 0)) > 0:
		return true
	var item_rewards: Array = parsed.get("items", []) as Array
	return not item_rewards.is_empty()


func _clear_reward_rows() -> void:
	if _content_list == null:
		return
	for child: Node in _content_list.get_children():
		child.queue_free()


## 追加一行左对齐纯文本。
func _append_plain_line(text: String) -> void:
	if _content_list == null:
		return
	var row_label: Label = Label.new()
	row_label.text = UiFormat.normalize_text(text)
	row_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_LEFT
	row_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	row_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row_label.add_theme_font_size_override("font_size", CONTENT_FONT_SIZE)
	row_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_content_list.add_child(row_label)


## 追加一行左对齐 BBCode 文本，用于数字高亮。
func _append_rich_line(bbcode_text: String) -> void:
	if _content_list == null:
		return
	var row_label: RichTextLabel = RichTextLabel.new()
	row_label.bbcode_enabled = true
	row_label.scroll_active = false
	row_label.fit_content = true
	row_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	row_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row_label.add_theme_font_size_override("normal_font_size", CONTENT_FONT_SIZE)
	row_label.text = bbcode_text
	row_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_content_list.add_child(row_label)


## 生成角色经验奖励文案。
func _format_player_exp_line(exp_value: int) -> String:
	return "你得到了%s经验值" % _colored_number(exp_value)


## 生成宠物经验奖励文案。
func _format_pet_exp_line(exp_value: int) -> String:
	return "你的宠物得到了%s经验值" % _colored_number(exp_value)


## 把总铜币拆成金/银/铜后拼成一行展示文案。
func _format_gold_reward_line(total_copper: int) -> String:
	var split: Dictionary = _split_total_copper(total_copper)
	var segments: Array[String] = []
	var gold_amount: int = int(split.get("gold", 0))
	var silver_amount: int = int(split.get("silver", 0))
	var copper_amount: int = int(split.get("copper", 0))
	if gold_amount > 0:
		segments.append("%s金币" % _colored_number(gold_amount))
	if silver_amount > 0:
		segments.append("%s银币" % _colored_number(silver_amount))
	if copper_amount > 0:
		segments.append("%s铜币" % _colored_number(copper_amount))
	if segments.is_empty():
		return ""
	return "你得到了%s" % "".join(segments)


## 与服务端 wallet 拆分规则保持一致。
func _split_total_copper(total_copper: int) -> Dictionary:
	var safe_total: int = maxi(0, total_copper)
	var gold_amount: int = int(safe_total / COPPER_PER_GOLD)
	var remainder: int = safe_total % COPPER_PER_GOLD
	var silver_amount: int = int(remainder / COPPER_PER_SILVER)
	var copper_amount: int = remainder % COPPER_PER_SILVER
	return {
		"gold": gold_amount,
		"silver": silver_amount,
		"copper": copper_amount,
	}


## 把整数包成高亮 BBCode 片段。
func _colored_number(value: int) -> String:
	return "[color=%s]%s[/color]" % [REWARD_NUMBER_COLOR, UiFormat.value_to_text(value)]


## 追加一排物品图标。
func _append_item_icon_row(item_rewards: Array) -> void:
	if _content_list == null:
		return
	var row: HBoxContainer = HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	row.alignment = BoxContainer.ALIGNMENT_BEGIN
	row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	for item_reward_variant: Variant in item_rewards:
		if item_reward_variant is not Dictionary:
			continue
		row.add_child(_create_item_icon_cell(item_reward_variant as Dictionary))
	_content_list.add_child(row)


## 创建单个物品图标格子，右下角显示堆叠数量。
func _create_item_icon_cell(reward: Dictionary) -> Control:
	var cell: Control = Control.new()
	cell.custom_minimum_size = Vector2(ITEM_ICON_SIZE + 4, ITEM_ICON_SIZE + 4)
	cell.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var icon_rect: TextureRect = TextureRect.new()
	icon_rect.custom_minimum_size = Vector2(ITEM_ICON_SIZE, ITEM_ICON_SIZE)
	icon_rect.size = Vector2(ITEM_ICON_SIZE, ITEM_ICON_SIZE)
	var icon_texture: Texture2D = _resolve_item_icon_texture(reward)
	icon_rect.texture = icon_texture
	icon_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	icon_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	icon_rect.visible = icon_texture != null
	icon_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	cell.add_child(icon_rect)
	var item_count: int = int(reward.get("count", 0))
	if item_count > 1:
		var count_label: Label = Label.new()
		count_label.text = UiFormat.value_to_text(item_count)
		count_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
		count_label.vertical_alignment = VERTICAL_ALIGNMENT_BOTTOM
		count_label.set_anchors_preset(Control.PRESET_FULL_RECT)
		count_label.add_theme_font_size_override("font_size", 11)
		count_label.add_theme_color_override("font_outline_color", Color(0.0, 0.0, 0.0, 1.0))
		count_label.add_theme_constant_override("outline_size", 2)
		count_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
		cell.add_child(count_label)
	var item_name: String = str(reward.get("item_name", ""))
	if item_name.is_empty():
		item_name = "物品 %d" % int(reward.get("item_id", 0))
	cell.tooltip_text = UiFormat.normalize_text("%s x%d" % [item_name, maxi(item_count, 1)])
	return cell


## 按 item_id 从本地注册表解析奖励图标。
func _resolve_item_icon_texture(reward: Dictionary) -> Texture2D:
	return ItemIcons.resolve_texture(int(reward.get("item_id", 0)))
