extends NetworkDoorLevelBase

@export var level_center_position: Vector2 = Vector2.ZERO
@export var level_scale_factor: float = 1.0
@export var login_spawn_position: Vector2 = Vector2(120.0, 136.0)

func _get_door_configs() -> Dictionary:
	return {
		"DownDoor": {
			"portal_id": 4001,
			"target_scene_id": 3,
		}
	}

func _get_default_facing_direction() -> Vector2:
	return Vector2.DOWN

func get_level_center_position() -> Vector2:
	if level_center_position != Vector2.ZERO:
		return level_center_position

	var map_layer := _resolve_map_layer()
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

func _resolve_map_layer() -> TileMapLayer:
	for layer_name in ["Map", "TileMapLayer"]:
		var layer := get_node_or_null(layer_name) as TileMapLayer
		if layer != null:
			return layer
	return null
