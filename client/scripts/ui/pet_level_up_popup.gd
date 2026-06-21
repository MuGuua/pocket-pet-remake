extends "res://scripts/ui/modal_popup_layer.gd"

## 展示单只宠物的升级结果；pet_name 为展示名，level 为升级后的当前等级。
func show_pet_level_up(pet_name: String, level: int, attr_points_gained: int, free_attr_points: int) -> void:
	if level <= 0:
		return
	var resolved_name: String = pet_name.strip_edges()
	if resolved_name.is_empty():
		resolved_name = "你的宠物"
	_set_label_text("TitleLabel", resolved_name)
	_set_label_text("LevelLabel", "升到了 %d 级" % level)
	_set_label_text("PointsGainLabel", "获得自由属性点：%d" % attr_points_gained)
	_set_label_text("FreePointsLabel", "当前可用自由点：%d" % free_attr_points)
	_open_modal()


## 关闭宠物升级弹窗。
func close_popup() -> void:
	_close_modal()


## 按 unique name 更新标签文案。
func _set_label_text(unique_name: String, text: String) -> void:
	var label: Label = get_node_or_null("%" + unique_name) as Label
	if label == null:
		return
	label.text = text
