extends RefCounted
class_name BattleFormationMapper

## 把服务端 lineup_index 映射为客户端站位 key，position 仅作本地布局使用。

const ALLY_SLOT_PREFIXES: Array[String] = ["left_front", "left_back"]
const ENEMY_SLOT_PREFIXES: Array[String] = ["right_front", "right_back"]

# 与 battle_scene.tscn -> Bg/PanelContainer/TextureRect2 魔法阵贴图区域保持一致。
const MAGIC_CIRCLE_TOP_Y: float = 165.0
const MAGIC_CIRCLE_BOTTOM_Y: float = 315.0
const MAGIC_CIRCLE_CENTER_Y: float = (MAGIC_CIRCLE_TOP_Y + MAGIC_CIRCLE_BOTTOM_Y) * 0.5

## 返回 slot_key -> Vector2 的站位表，坐标按移动端战斗区域预置。
static func build_slot_positions() -> Dictionary:
	var result: Dictionary = {}
	var battle_area: Dictionary = {
		"top": MAGIC_CIRCLE_TOP_Y,
		"bottom": MAGIC_CIRCLE_BOTTOM_Y,
		"anchor_y": MAGIC_CIRCLE_CENTER_Y,
		"ally_back_x": 42.0,
		"ally_front_x": 88.0,
		"enemy_front_x": 272.0,
		"enemy_back_x": 314.0,
		"ally_rows": 4,
		"enemy_rows": 6,
		"ally_row_spacing": 38.0,
		"enemy_row_spacing": 28.0,
		"unit_footprint": 32.0,
	}
	_apply_battle_area_layout(result, battle_area)
	result["left_front"] = result.get("left_front_1", Vector2(88.0, MAGIC_CIRCLE_CENTER_Y))
	result["right_front"] = result.get("right_front_1", Vector2(272.0, MAGIC_CIRCLE_CENTER_Y))
	return result

## 根据阵营与 lineup_index 计算站位 key。
static func resolve_position_key(is_ally: bool, lineup_index: int) -> String:
	var row_index: int = max(0, lineup_index)
	var prefixes: Array[String] = ALLY_SLOT_PREFIXES if is_ally else ENEMY_SLOT_PREFIXES
	var prefix: String = prefixes[min(row_index, prefixes.size() - 1)]
	return "%s_%d" % [prefix, row_index + 1]

static func _apply_battle_area_layout(target: Dictionary, battle_area: Dictionary) -> void:
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
		ally_spacing = (usable_height - unit_footprint) / float(max(ally_rows - 1, 1))
	if enemy_spacing <= 0.0:
		enemy_spacing = (usable_height - unit_footprint) / float(max(enemy_rows - 1, 1))
	var anchor_y: float = float(battle_area.get("anchor_y", -1.0))
	if anchor_y >= 0.0:
		# 前排脚底锚定在魔法阵中心 Y，后排逐级向画面上方排开。
		for row_index: int in range(ally_rows):
			var slot_y: float = anchor_y - float(row_index) * ally_spacing
			target["left_back_%d" % (row_index + 1)] = Vector2(ally_back_x, slot_y)
			target["left_front_%d" % (row_index + 1)] = Vector2(ally_front_x, slot_y)
		for row_index: int in range(enemy_rows):
			var slot_y: float = anchor_y - float(row_index) * enemy_spacing
			target["right_front_%d" % (row_index + 1)] = Vector2(enemy_front_x, slot_y)
			target["right_back_%d" % (row_index + 1)] = Vector2(enemy_back_x, slot_y)
		return
	var vertical_anchor: float = clampf(float(battle_area.get("vertical_anchor", 0.5)), 0.0, 1.0)
	var ally_block_height: float = unit_footprint + ally_spacing * float(ally_rows - 1)
	var enemy_block_height: float = unit_footprint + enemy_spacing * float(enemy_rows - 1)
	var ally_start_y: float = top + (usable_height - ally_block_height) * vertical_anchor
	var enemy_start_y: float = top + (usable_height - enemy_block_height) * vertical_anchor
	for row_index: int in range(ally_rows):
		var slot_y: float = ally_start_y + float(row_index) * ally_spacing
		target["left_back_%d" % (row_index + 1)] = Vector2(ally_back_x, slot_y)
		target["left_front_%d" % (row_index + 1)] = Vector2(ally_front_x, slot_y)
	for row_index: int in range(enemy_rows):
		var slot_y: float = enemy_start_y + float(row_index) * enemy_spacing
		target["right_front_%d" % (row_index + 1)] = Vector2(enemy_front_x, slot_y)
		target["right_back_%d" % (row_index + 1)] = Vector2(enemy_back_x, slot_y)
