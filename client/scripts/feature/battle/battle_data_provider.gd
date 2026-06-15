extends Node
class_name BattleDataProvider

const DATA_PATH: String = "res://data/battle_demo.json"

var _battle_data: Dictionary = {}

func load_data() -> void:
	var file: FileAccess = FileAccess.open(DATA_PATH, FileAccess.READ)
	if file == null:
		push_error("无法打开战斗数据: %s" % DATA_PATH)
		_battle_data = {}
		return

	var parsed: Variant = JSON.parse_string(file.get_as_text())
	if typeof(parsed) != TYPE_DICTIONARY:
		push_error("战斗数据格式错误: %s" % DATA_PATH)
		_battle_data = {}
		return

	_battle_data = parsed

func get_battle_name() -> String:
	return str(_battle_data.get("battle_name", "战斗演示"))

func get_initial_units() -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	var units: Array = _battle_data.get("units", []) as Array
	for entry_value: Variant in units:
		if entry_value is Dictionary:
			var entry: Dictionary = entry_value
			result.append(entry)
	return result

func get_round_count() -> int:
	return int((_battle_data.get("rounds", []) as Array).size())

func get_round_data(round_number: int) -> Dictionary:
	var rounds: Array = _battle_data.get("rounds", []) as Array
	for round_entry_value: Variant in rounds:
		if round_entry_value is Dictionary:
			var round_entry: Dictionary = round_entry_value
			if int(round_entry.get("round", 0)) == round_number:
				return round_entry
	return {}

func get_round_input_phase(round_number: int) -> Dictionary:
	var round_data: Dictionary = get_round_data(round_number)
	var input_phase_value: Variant = round_data.get("input_phase", {})
	if input_phase_value is Dictionary:
		return input_phase_value as Dictionary
	return {}

func get_formation_limits() -> Dictionary:
	var formation: Dictionary = _battle_data.get("formation", {}) as Dictionary
	var limits: Dictionary = formation.get("limits", {}) as Dictionary
	return {
		"ally_max": int(limits.get("ally_max", 8)),
		"enemy_max": int(limits.get("enemy_max", 12))
	}

func get_slot_positions() -> Dictionary:
	var result: Dictionary = {}
	var formation: Dictionary = _battle_data.get("formation", {}) as Dictionary
	if formation.is_empty():
		push_warning("战斗数据缺少 formation 配置，无法解析站位。")
		return result

	var explicit_slots: Dictionary = formation.get("slots", {}) as Dictionary
	for slot_key_value: Variant in explicit_slots.keys():
		var slot_key: String = str(slot_key_value)
		var slot_value: Variant = explicit_slots[slot_key]
		if _try_parse_slot_coordinate(slot_value, result, slot_key):
			continue
		push_warning("站位坐标格式无效: %s" % slot_key)

	var layout: Dictionary = formation.get("layout", {}) as Dictionary
	var battle_area: Dictionary = formation.get("battle_area", {}) as Dictionary
	if not battle_area.is_empty():
		_apply_battle_area_layout(result, battle_area)
	for layout_group_value: Variant in layout.values():
		if layout_group_value is Dictionary:
			_apply_layout_group(result, layout_group_value as Dictionary)

	var aliases: Dictionary = formation.get("aliases", {}) as Dictionary
	for alias_key_value: Variant in aliases.keys():
		var alias_key: String = str(alias_key_value)
		var target_key: String = str(aliases[alias_key_value])
		if result.has(target_key):
			result[alias_key] = result[target_key]
		else:
			push_warning("站位别名目标不存在: %s -> %s" % [alias_key, target_key])

	return result

func validate_unit_formation(units: Array[Dictionary]) -> void:
	var limits: Dictionary = get_formation_limits()
	var ally_max: int = int(limits.get("ally_max", 8))
	var enemy_max: int = int(limits.get("enemy_max", 12))
	var ally_count: int = 0
	var enemy_count: int = 0
	for unit_data: Dictionary in units:
		var unit_type: String = str(unit_data.get("type", ""))
		if unit_type == "enemy":
			enemy_count += 1
		elif unit_type == "player" or unit_type == "pet":
			ally_count += 1
	if ally_count > ally_max:
		push_warning("我方单位数量 %d 超过 formation.limits.ally_max=%d" % [ally_count, ally_max])
	if enemy_count > enemy_max:
		push_warning("敌方单位数量 %d 超过 formation.limits.enemy_max=%d" % [enemy_count, enemy_max])

func _apply_battle_area_layout(target: Dictionary, battle_area: Dictionary) -> void:
	var top: float = float(battle_area.get("top", 58.0))
	var bottom: float = float(battle_area.get("bottom", 296.0))
	var ally_rows: int = clampi(int(battle_area.get("ally_rows", 4)), 1, 8)
	var enemy_rows: int = clampi(int(battle_area.get("enemy_rows", 6)), 1, 12)
	var ally_back_x: float = float(battle_area.get("ally_back_x", 42.0))
	var ally_front_x: float = float(battle_area.get("ally_front_x", 88.0))
	var enemy_front_x: float = float(battle_area.get("enemy_front_x", 272.0))
	var enemy_back_x: float = float(battle_area.get("enemy_back_x", 314.0))
	var unit_footprint: float = float(battle_area.get("unit_footprint", 32.0))
	var usable_height: float = max(bottom - top, unit_footprint)
	var ally_spacing: float = float(battle_area.get("ally_row_spacing", 0.0))
	var enemy_spacing: float = float(battle_area.get("enemy_row_spacing", 0.0))
	if ally_spacing <= 0.0:
		if ally_rows <= 1:
			ally_spacing = 0.0
		else:
			ally_spacing = (usable_height - unit_footprint) / float(ally_rows - 1)
	if enemy_spacing <= 0.0:
		if enemy_rows <= 1:
			enemy_spacing = 0.0
		else:
			enemy_spacing = (usable_height - unit_footprint) / float(enemy_rows - 1)
	var ally_block_height: float = unit_footprint + ally_spacing * float(ally_rows - 1)
	var enemy_block_height: float = unit_footprint + enemy_spacing * float(enemy_rows - 1)
	var ally_start_y: float = top + (usable_height - ally_block_height) * 0.5
	var enemy_start_y: float = top + (usable_height - enemy_block_height) * 0.5
	for row_index: int in range(ally_rows):
		var slot_y: float = ally_start_y + float(row_index) * ally_spacing
		target["left_back_%d" % (row_index + 1)] = Vector2(ally_back_x, slot_y)
		target["left_front_%d" % (row_index + 1)] = Vector2(ally_front_x, slot_y)
	for row_index: int in range(enemy_rows):
		var slot_y: float = enemy_start_y + float(row_index) * enemy_spacing
		target["right_front_%d" % (row_index + 1)] = Vector2(enemy_front_x, slot_y)
		target["right_back_%d" % (row_index + 1)] = Vector2(enemy_back_x, slot_y)

func _apply_layout_group(target: Dictionary, layout_group: Dictionary) -> void:
	var prefix: String = str(layout_group.get("prefix", ""))
	if prefix.is_empty():
		push_warning("formation.layout 缺少 prefix 字段。")
		return
	var count: int = clampi(int(layout_group.get("count", 1)), 1, 12)
	var x: float = float(layout_group.get("x", 0.0))
	var start_y: float = float(layout_group.get("start_y", 0.0))
	var row_spacing: float = float(layout_group.get("row_spacing", 80.0))
	for row_index: int in range(count):
		var slot_id: String = "%s_%d" % [prefix, row_index + 1]
		target[slot_id] = Vector2(x, start_y + float(row_index) * row_spacing)

func _try_parse_slot_coordinate(slot_value: Variant, target: Dictionary, slot_key: String) -> bool:
	if slot_value is Dictionary:
		var slot_dict: Dictionary = slot_value as Dictionary
		target[slot_key] = Vector2(float(slot_dict.get("x", 0.0)), float(slot_dict.get("y", 0.0)))
		return true
	if slot_value is Array:
		var slot_array: Array = slot_value as Array
		if slot_array.size() >= 2:
			target[slot_key] = Vector2(float(slot_array[0]), float(slot_array[1]))
			return true
	return false
