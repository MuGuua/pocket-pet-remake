class_name NetworkDoorLevelBase
extends Node2D

## INVALID_PORTAL_SPAWN_SCENE_POSITION 表示当前场景没有为指定 portal_id 配置客户端场景坐标出生点。
const INVALID_PORTAL_SPAWN_SCENE_POSITION: Vector2 = Vector2.INF

signal scene_change_requested(change_request: Dictionary)

func _ready() -> void:
	for door_name in _get_door_configs().keys():
		_bind_door(door_name)

func _get_door_configs() -> Dictionary:
	return {}

func _get_default_facing_direction() -> Vector2:
	return Vector2.DOWN

## 返回从指定 portal_id 传入当前场景时的客户端场景坐标；未配置时返回 INVALID_PORTAL_SPAWN_SCENE_POSITION。
func get_portal_spawn_scene_position(_portal_id: int) -> Vector2:
	return INVALID_PORTAL_SPAWN_SCENE_POSITION

func _bind_door(door_name: String) -> void:
	var door := _find_door_area(door_name)
	if door == null:
		push_warning("Door area not found: %s" % door_name)
		return

	# 场景层级和全局碰撞层改动后，门区域仍保持可检测。
	door.monitoring = true
	door.monitorable = true
	for layer_index in range(1, 33):
		door.set_collision_mask_value(layer_index, true)

	var callback := _on_door_body_entered.bind(door_name)
	if not door.body_entered.is_connected(callback):
		door.body_entered.connect(callback)

func _on_door_body_entered(body: Node2D, door_name: String) -> void:
	if body == null or body.name != "Player":
		return

	var door_config: Dictionary = _get_door_configs().get(door_name, {})
	var portal_id := int(door_config.get("portal_id", 0))
	var target_scene_id := int(door_config.get("target_scene_id", 0))
	if portal_id <= 0 or target_scene_id <= 0:
		return

	var change_request := {
		"portal_id": portal_id,
		"target_scene_id": target_scene_id,
		"facing_direction": _resolve_facing_direction(body, door_config),
	}
	scene_change_requested.emit(change_request)

func _resolve_facing_direction(body: Node2D, door_config: Dictionary) -> Vector2:
	if door_config.has("facing_direction"):
		var facing_direction: Variant = door_config.get("facing_direction")
		if facing_direction is Vector2 and facing_direction != Vector2.ZERO:
			return facing_direction

	var body_facing: Variant = body.get("cardinal_direction")
	if body_facing is Vector2 and body_facing != Vector2.ZERO:
		return body_facing
	return _get_default_facing_direction()


func _find_door_area(door_name: String) -> Area2D:
	var direct_door := get_node_or_null(door_name) as Area2D
	if direct_door != null:
		return direct_door
	return find_child(door_name, true, false) as Area2D
