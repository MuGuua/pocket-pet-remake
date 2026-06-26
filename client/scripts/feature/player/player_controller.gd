extends Node

## 处理玩家成长与人物装备相关协议，并把服务端权威快照写回 GameState。

# 人物装备数据刷新后向外广播。
signal equipment_updated()

# 处理属性点分配响应，并刷新本地玩家快照。
func handle_allocate_attr_response(payload: Dictionary) -> void:
	var player_variant: Variant = payload.get("player", {})
	if player_variant is Dictionary:
		GameState.merge_player_snapshot(player_variant as Dictionary)


# 处理已佩戴装备列表响应。
func handle_equipment_list_response(payload: Dictionary) -> void:
	var items_variant: Variant = payload.get("items", [])
	if items_variant is Array:
		GameState.set_equipped_items(items_variant as Array)
	equipment_updated.emit()


# 处理从背包装备人物装备响应。
func handle_equip_response(payload: Dictionary) -> void:
	_apply_equipment_mutation_response(payload)
	var equipped_variant: Variant = payload.get("equipped", {})
	if equipped_variant is Dictionary:
		var equipped_item: Dictionary = equipped_variant as Dictionary
		var item_name: String = str(equipped_item.get("item_name", "装备"))
		App.notice_received.emit("已装备：%s" % item_name)


# 处理卸下人物装备响应。
func handle_unequip_response(payload: Dictionary) -> void:
	_apply_equipment_mutation_response(payload)
	var unequipped_variant: Variant = payload.get("unequipped", {})
	if unequipped_variant is Dictionary:
		var unequipped_item: Dictionary = unequipped_variant as Dictionary
		var item_name: String = str(unequipped_item.get("item_name", "装备"))
		App.notice_received.emit("已卸下：%s" % item_name)


# 处理人物装备强化响应。
func handle_enhance_response(payload: Dictionary) -> void:
	var all_equipped_variant: Variant = payload.get("all_equipped", [])
	if all_equipped_variant is Array:
		GameState.set_equipped_items(all_equipped_variant as Array)
	if GameState.is_ws_authenticated:
		App.request_bag_list()
	equipment_updated.emit()


## 合并佩戴/卸下后的玩家属性与全身装备，并刷新背包快照。
func _apply_equipment_mutation_response(payload: Dictionary) -> void:
	var player_variant: Variant = payload.get("player", {})
	if player_variant is Dictionary:
		GameState.merge_player_snapshot(player_variant as Dictionary)
	var all_equipped_variant: Variant = payload.get("all_equipped", [])
	if all_equipped_variant is Array:
		GameState.set_equipped_items(all_equipped_variant as Array)
	if GameState.is_ws_authenticated:
		App.request_bag_list()
	equipment_updated.emit()
