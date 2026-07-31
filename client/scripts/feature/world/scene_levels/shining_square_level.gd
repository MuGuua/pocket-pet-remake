extends NetworkDoorLevelBase

## 闪耀广场当前只接通返回闪光镇传送区的入口；其余门待目标地图落地后再配置。

@export var scene_display_name: String = "闪耀广场"
@export var level_scale_factor: float = 1.0
@export var login_spawn_position: Vector2 = Vector2(14.0, 8.0)

const SHANGUANG_TOWN_TRANSFER_AREA_SCENE_ID: int = 8
const TO_SHANGUANG_TOWN_TRANSFER_AREA_PORTAL_ID: int = 9001


func get_scene_display_name() -> String:
	return scene_display_name


func _get_door_configs() -> Dictionary:
	return {
		"通往闪光镇传送区": {
			"portal_id": TO_SHANGUANG_TOWN_TRANSFER_AREA_PORTAL_ID,
			"target_scene_id": SHANGUANG_TOWN_TRANSFER_AREA_SCENE_ID,
			"facing_direction": Vector2.DOWN,
		},
	}


func _get_default_facing_direction() -> Vector2:
	return Vector2.DOWN


func get_portal_spawn_scene_position(_portal_id: int) -> Vector2:
	return INVALID_PORTAL_SPAWN_SCENE_POSITION


func get_level_center_position() -> Vector2:
	var map_layer: TileMapLayer = get_node_or_null("地图") as TileMapLayer
	if map_layer == null:
		return Vector2.ZERO
	var used_rect: Rect2i = map_layer.get_used_rect()
	if not used_rect.has_area():
		return Vector2.ZERO
	var top_left: Vector2 = map_layer.map_to_local(used_rect.position)
	var bottom_right: Vector2 = map_layer.map_to_local(used_rect.position + used_rect.size)
	return (top_left + bottom_right) * 0.5


func get_level_scale_factor() -> float:
	return level_scale_factor


func get_login_spawn_position() -> Vector2:
	return login_spawn_position
