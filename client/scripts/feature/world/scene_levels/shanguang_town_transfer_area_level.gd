extends NetworkDoorLevelBase

## 闪光镇传送区连接闪光镇东路与闪光平原入口。

@export var scene_display_name: String = "闪光镇传送区"
@export var level_scale_factor: float = 1.0
## 首次进入或没有权威位置时的默认场景坐标。
@export var login_spawn_position: Vector2 = Vector2(4.0, 6.0)

const EAST_ROAD_SCENE_ID: int = 2
const SHINING_SQUARE_SCENE_ID: int = 9
const TO_EAST_ROAD_PORTAL_ID: int = 8001
const TO_SHINING_SQUARE_PORTAL_ID: int = 8002


func get_scene_display_name() -> String:
	return scene_display_name


## 两个 Area2D 可以放在“传送门”容器内，基类会递归查找并绑定 body_entered。
func _get_door_configs() -> Dictionary:
	return {
		"通往闪光镇东路": {
			"portal_id": TO_EAST_ROAD_PORTAL_ID,
			"target_scene_id": EAST_ROAD_SCENE_ID,
			"facing_direction": Vector2.LEFT,
		},
		"通往闪光平原": {
			"portal_id": TO_SHINING_SQUARE_PORTAL_ID,
			"target_scene_id": SHINING_SQUARE_SCENE_ID,
			"facing_direction": Vector2.UP,
		},
	}


func _get_default_facing_direction() -> Vector2:
	return Vector2.UP


## 出生位置以服务端 WORLD_RESYNC 的权威坐标为准，此处不做客户端门点覆盖。
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
