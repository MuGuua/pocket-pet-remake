extends NetworkDoorLevelBase

@export var level_center_position: Vector2 = Vector2.ZERO
@export var level_scale_factor: float = 2.25
@export var inbound_from_roxus_position: Vector2 = Vector2(101.0, 235.0)
@export var inbound_from_market_position: Vector2 = Vector2(26.0, 85.0)
@export var login_spawn_position: Vector2 = Vector2(72.0, 55.0)

func _get_door_configs() -> Dictionary:
	return {
		"UpPortal": {
			"portal_id": 2001,
			"target_scene_id": 1,
		},
		"LeftPortal": {
			"portal_id": 2002,
			"target_scene_id": 3,
		},
	}

func _get_default_facing_direction() -> Vector2:
	return Vector2.UP

func get_level_center_position() -> Vector2:
	if level_center_position != Vector2.ZERO:
		return level_center_position

	var map_layer := get_node_or_null("Map") as TileMapLayer
	if map_layer == null:
		return Vector2.ZERO

	var used_rect := map_layer.get_used_rect()
	if not used_rect.has_area():
		return Vector2.ZERO

	var top_left := map_layer.map_to_local(used_rect.position)
	var bottom_right := map_layer.map_to_local(used_rect.position + used_rect.size)
	return (top_left + bottom_right) * 0.5

func get_level_scale_factor() -> float:
	return level_scale_factor

func get_login_spawn_position() -> Vector2:
	return login_spawn_position
