extends "res://scripts/ui/modal_popup_layer.gd"

## 奖励列表容器。
@onready var _content_list: VBoxContainer = %ContentList


## 展示奖励列表；无有效奖励时不弹窗。
func show_rewards(title_text: String, rewards: Array, pet_rewards: Array = []) -> void:
	var lines: Array[String] = _build_reward_lines(rewards, pet_rewards)
	if lines.is_empty():
		return
	_clear_reward_rows()
	if not title_text.is_empty():
		_append_content_line(title_text, 12)
	for line in lines:
		_append_content_line(line, 12)
	_open_modal()


## 关闭奖励弹窗。
func close_popup() -> void:
	_close_modal()


func _append_content_line(text: String, font_size: int) -> void:
	if _content_list == null:
		return
	var row_label: Label = Label.new()
	row_label.text = text
	row_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	row_label.add_theme_font_size_override("font_size", font_size)
	row_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_content_list.add_child(row_label)


func _clear_reward_rows() -> void:
	if _content_list == null:
		return
	for child in _content_list.get_children():
		child.queue_free()


func _build_reward_lines(rewards: Array, pet_rewards: Array) -> Array[String]:
	var lines: Array[String] = []
	for reward_variant in rewards:
		if not reward_variant is Dictionary:
			continue
		var reward: Dictionary = reward_variant
		var reward_type: String = str(reward.get("type", ""))
		if reward_type == "exp":
			var exp_value: int = int(reward.get("value", 0))
			if exp_value > 0:
				lines.append("角色经验 +%d" % exp_value)
		elif reward_type == "gold":
			var gold_value: int = int(reward.get("value", 0))
			if gold_value > 0:
				lines.append("铜币 +%d" % gold_value)
		elif reward_type == "item":
			var item_count: int = int(reward.get("count", 0))
			if item_count <= 0:
				continue
			var item_name: String = str(reward.get("item_name", ""))
			if item_name.is_empty():
				item_name = "物品 %d" % int(reward.get("item_id", 0))
			lines.append("%s x%d" % [item_name, item_count])
	for pet_reward_variant in pet_rewards:
		if not pet_reward_variant is Dictionary:
			continue
		var pet_reward: Dictionary = pet_reward_variant
		var pet_exp: int = int(pet_reward.get("exp", 0))
		if pet_exp > 0:
			lines.append("宠物经验 +%d" % pet_exp)
	return lines
