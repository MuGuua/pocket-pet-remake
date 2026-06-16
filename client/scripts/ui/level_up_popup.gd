extends "res://scripts/ui/modal_popup_layer.gd"

## 展示升级结果；level 为升级后的当前等级，bonus 为服务端下发的属性加成摘要。
func show_level_up(level: int, bonus: Dictionary) -> void:
	if level <= 0:
		return
	var hp_gain: int = int(bonus.get("hp_max", 0))
	var atk_gain: int = int(bonus.get("atk", 0))
	var mana_gain: int = int(bonus.get("mana", 0))
	var spd_gain: int = int(bonus.get("spd", 0))
	_set_label_text("LevelLabel", "恭喜你升到了%d级" % level)
	_set_label_text("HpGainLabel", "最大生命值增加：%d" % hp_gain)
	_set_label_text("AtkGainLabel", "攻击力增加：%d" % atk_gain)
	_set_label_text("ManaGainLabel", "法力增加：%d" % mana_gain)
	_set_label_text("SpdGainLabel", "速度增加：%d" % spd_gain)
	_open_modal()


## 关闭升级弹窗。
func close_popup() -> void:
	_close_modal()


func _set_label_text(unique_name: String, text: String) -> void:
	var label: Label = get_node_or_null("%" + unique_name) as Label
	if label == null:
		return
	label.text = text
