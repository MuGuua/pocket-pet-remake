extends Node2D

signal scene_loaded(scene_id: String)
signal player_position_changed(position: Vector2)
signal scene_transition_requested(from_scene_id: int, to_scene_id: int)
signal scene_transition_failed(reason: String)

const DEFAULT_RENDER_FRAME_SIZE: Vector2 = Vector2(360.0, 480.0)
const PORTAL_ACTIVATION_COOLDOWN_MS: int = 350
const SCENE_GRID_TO_PIXELS: float = 24.0
const FIXED_VIEW_MARGIN: float = 24.0
const SCENE_CONFIGS: Dictionary = {
	1: {
		"scene_path": "res://scenes/maps/fashtown/roxus_house.tscn",
		"spawn": Vector2(8.0, 6.0),
		"fixed_view": true,
		"view_offset": Vector2.ZERO,
		"view_scale": 1.0,
	},
}

@onready var camera: Camera2D = $Camera2D
@onready var map_mount: Node2D = %MapMount
@onready var local_player_anchor: Node2D = %LocalPlayerAnchor
@onready var local_player: CharacterBody2D = local_player_anchor.get_node("player") as CharacterBody2D
@onready var remote_entities_root: Node2D = %RemoteEntities
@onready var map_loading_overlay: ColorRect = %MapLoadingOverlay

var _next_op_id: int = 1
var _next_move_seq: int = 1
var _pending_target_scene_id: int = 0
var _pending_portal_id: int = 0
var _last_loaded_scene_id: int = 0
var _loaded_map_scene_id: int = 0
var _current_map_node: Node
var _portal_cooldown_until_ms: int = 0
var _render_frame_size: Vector2 = DEFAULT_RENDER_FRAME_SIZE

func _ready() -> void:
	_apply_render_frame_size()
	_apply_scene_layout(_current_scene_id())
	GameState.battle_changed.connect(_sync_local_player_battle_state)
	_sync_local_player_battle_state()

func handle_enter_world(payload: Dictionary) -> void:
	GameState.set_world_snapshot(payload)
	_apply_authoritative_snapshot()
	_emit_scene_loaded_if_changed(true)

func handle_entity_enter(payload: Dictionary) -> void:
	var entity_variant: Variant = payload.get("entity", payload)
	var entity: Dictionary = entity_variant if entity_variant is Dictionary else {}
	GameState.add_entity(entity)

func handle_entity_leave(payload: Dictionary) -> void:
	GameState.remove_entity(int(payload.get("entity_id", 0)))

func handle_entity_move(payload: Dictionary) -> void:
	GameState.apply_entity_move(payload)

func handle_move_intent_response(payload: Dictionary) -> void:
	var accepted: bool = bool(payload.get("accepted", false))
	var scene_id: int = int(payload.get("scene_id", _current_scene_id()))
	if accepted and scene_id == _current_scene_id():
		_pending_target_scene_id = 0
		_pending_portal_id = 0
		_set_transition_loading(false)
		_unlock_local_player()
		return

	if accepted:
		return

	_pending_target_scene_id = 0
	_pending_portal_id = 0
	_set_transition_loading(false)
	_unlock_local_player()
	scene_transition_failed.emit(str(payload.get("reason", "scene transfer rejected")))

func handle_world_resync(payload: Dictionary) -> void:
	GameState.set_world_snapshot(payload)
	_apply_authoritative_snapshot()
	_emit_scene_loaded_if_changed(false)

func request_scene_transition(target_scene_id: int, portal_id: int = 0) -> void:
	var current_scene_id := _current_scene_id()
	if current_scene_id <= 0:
		_unlock_local_player()
		scene_transition_failed.emit("scene not initialized")
		return
	if target_scene_id <= 0 or target_scene_id == current_scene_id:
		_unlock_local_player()
		return
	if _pending_target_scene_id != 0:
		return

	_pending_target_scene_id = target_scene_id
	_pending_portal_id = portal_id
	_lock_local_player()
	_set_transition_loading(true)
	scene_transition_requested.emit(current_scene_id, target_scene_id)
	NetClient.send_command(
		CommandIds.MOVE_INTENT_REQ,
		{
			"op_id": _take_next_op_id(),
			"move_seq": _take_next_move_seq(),
			"scene_id": current_scene_id,
			"target_scene_id": target_scene_id,
			"portal_id": portal_id,
		}
	)

func _apply_authoritative_snapshot() -> void:
	var scene_id := _current_scene_id()
	if not _ensure_scene_map_loaded(scene_id):
		_pending_target_scene_id = 0
		_pending_portal_id = 0
		_set_transition_loading(false)
		_unlock_local_player()
		scene_transition_failed.emit("failed to load scene map: %d" % scene_id)
		return

	var self_pos := _extract_self_position(GameState.player_snapshot)
	var local_position := _server_to_local_position(scene_id, self_pos)
	if local_player != null and local_player.has_method("apply_authoritative_position"):
		local_player.call("apply_authoritative_position", local_position)
		if local_player.has_method("set_battle_active"):
			local_player.call("set_battle_active", GameState.is_in_battle)

	_pending_target_scene_id = 0
	_pending_portal_id = 0
	_portal_cooldown_until_ms = Time.get_ticks_msec() + PORTAL_ACTIVATION_COOLDOWN_MS
	_set_transition_loading(false)
	_unlock_local_player()
	player_position_changed.emit(local_player.global_position if local_player != null else local_player_anchor.global_position)

func _emit_scene_loaded_if_changed(force_emit: bool) -> void:
	var scene_id := _current_scene_id()
	if force_emit or scene_id != _last_loaded_scene_id:
		_last_loaded_scene_id = scene_id
		scene_loaded.emit(str(scene_id))

func _current_scene_id() -> int:
	return int(GameState.scene_snapshot.get("scene_id", 0))

func _extract_self_position(player_snapshot: Dictionary) -> Vector2:
	var x: float = float(player_snapshot.get("x", 0.0))
	var y: float = float(player_snapshot.get("y", 0.0))
	return Vector2(x, y)

func _server_to_local_position(scene_id: int, server_position: Vector2) -> Vector2:
	var scene_config_variant: Variant = _scene_config(scene_id)
	if scene_config_variant is not Dictionary:
		return Vector2.ZERO

	var spawn_variant: Variant = scene_config_variant.get("spawn", Vector2.ZERO)
	var spawn: Vector2 = spawn_variant if spawn_variant is Vector2 else Vector2.ZERO
	if _scene_uses_fixed_view(scene_id):
		var spawn_local := _scene_spawn_local_position(scene_id)
		return spawn_local + (server_position - spawn) * SCENE_GRID_TO_PIXELS
	return (server_position - spawn) * SCENE_GRID_TO_PIXELS

func _scene_config(scene_id: int) -> Dictionary:
	var scene_config_variant: Variant = SCENE_CONFIGS.get(scene_id, {})
	return scene_config_variant if scene_config_variant is Dictionary else {}


func _scene_uses_fixed_view(scene_id: int) -> bool:
	return bool(_scene_config(scene_id).get("fixed_view", false))

func _scene_spawn_local_position(scene_id: int) -> Vector2:
	var scene_config := _scene_config(scene_id)
	var spawn_local_variant: Variant = scene_config.get("spawn_local_position", null)
	if spawn_local_variant is Vector2:
		return spawn_local_variant
	if is_instance_valid(_current_map_node):
		# 固定镜头地图默认以地图可见内容中心作为出生显示点。
		var map_rect := _collect_scene_visual_rect(_current_map_node)
		if map_rect.size != Vector2.ZERO:
			return map_rect.get_center()
	return Vector2.ZERO

func set_render_frame_size(size: Vector2) -> void:
	if size.x <= 0.0 or size.y <= 0.0:
		return
	_render_frame_size = size
	_apply_render_frame_size()
	_apply_scene_layout(_current_scene_id())

func _ensure_scene_map_loaded(scene_id: int) -> bool:
	if scene_id <= 0:
		return false
	if _loaded_map_scene_id == scene_id and is_instance_valid(_current_map_node):
		return true

	var scene_config := _scene_config(scene_id)
	var scene_path := str(scene_config.get("scene_path", ""))
	if scene_path.is_empty():
		return false

	var map_scene := load(scene_path) as PackedScene
	if map_scene == null:
		return false

	_unload_scene_map()
	_current_map_node = map_scene.instantiate()
	map_mount.add_child(_current_map_node)
	_apply_scene_layout(scene_id)
	_bind_map_portals()
	_loaded_map_scene_id = scene_id
	return true

func _unload_scene_map() -> void:
	if is_instance_valid(_current_map_node):
		_current_map_node.queue_free()
	_current_map_node = null
	_loaded_map_scene_id = 0

func _bind_map_portals() -> void:
	if not is_instance_valid(_current_map_node):
		return
	for portal_node in _current_map_node.find_children("*", "Area2D", true, false):
		if portal_node.has_signal("activated") and not portal_node.is_connected("activated", Callable(self, "_on_map_portal_activated")):
			portal_node.connect("activated", Callable(self, "_on_map_portal_activated"))

func _on_map_portal_activated(portal_id: int, target_scene_id: int) -> void:
	if Time.get_ticks_msec() < _portal_cooldown_until_ms:
		return
	if _pending_target_scene_id != 0:
		return
	request_scene_transition(target_scene_id, portal_id)

func _lock_local_player() -> void:
	if local_player != null and local_player.has_method("set_scene_transition_locked"):
		local_player.call("set_scene_transition_locked", true)

func _unlock_local_player() -> void:
	if local_player != null and local_player.has_method("set_scene_transition_locked"):
		local_player.call("set_scene_transition_locked", false)

func _set_transition_loading(active: bool) -> void:
	if map_loading_overlay != null:
		map_loading_overlay.visible = active

func _sync_local_player_battle_state() -> void:
	if local_player != null and local_player.has_method("set_battle_active"):
		local_player.call("set_battle_active", GameState.is_in_battle)

func _apply_scene_layout(scene_id: int) -> void:
	var scene_config := _scene_config(scene_id)
	var viewport_center := _viewport_center()
	if camera != null:
		camera.position = viewport_center

	map_mount.scale = Vector2.ONE
	local_player_anchor.scale = Vector2.ONE
	remote_entities_root.scale = Vector2.ONE

	if _scene_uses_fixed_view(scene_id) and is_instance_valid(_current_map_node):
		var map_rect := _collect_scene_visual_rect(_current_map_node)
		var available_size := Vector2(
			maxf(_render_frame_size.x - FIXED_VIEW_MARGIN * 2.0, 1.0),
			maxf(_render_frame_size.y - FIXED_VIEW_MARGIN * 2.0, 1.0)
		)
		var scale_factor := 1.0
		if map_rect.size.x > 0.0 and map_rect.size.y > 0.0:
			scale_factor = minf(available_size.x / map_rect.size.x, available_size.y / map_rect.size.y)
		scale_factor *= float(scene_config.get("view_scale", 1.0))
		scale_factor = maxf(scale_factor, 0.01)
		var scaled_center := map_rect.get_center() * scale_factor
		var view_offset_variant: Variant = scene_config.get("view_offset", Vector2.ZERO)
		var view_offset: Vector2 = view_offset_variant if view_offset_variant is Vector2 else Vector2.ZERO
		var layout_origin := viewport_center - scaled_center + view_offset
		map_mount.position = layout_origin
		local_player_anchor.position = layout_origin
		remote_entities_root.position = layout_origin
		var scale_vec := Vector2.ONE * scale_factor
		map_mount.scale = scale_vec
		local_player_anchor.scale = scale_vec
		remote_entities_root.scale = scale_vec
		return

	map_mount.position = viewport_center
	local_player_anchor.position = viewport_center
	remote_entities_root.position = viewport_center

func _viewport_center() -> Vector2:
	if _render_frame_size == Vector2.ZERO:
		return DEFAULT_RENDER_FRAME_SIZE * 0.5
	return _render_frame_size * 0.5

func _collect_scene_visual_rect(root: Node) -> Rect2:
	var has_rect := false
	var combined := Rect2()
	var pending: Array[Node] = [root]
	while not pending.is_empty():
		var current: Node = pending.pop_back()
		var current_rect := _node_visual_rect(current)
		if current_rect.size != Vector2.ZERO:
			if not has_rect:
				combined = current_rect
				has_rect = true
			else:
				combined = combined.merge(current_rect)
		for child in current.get_children():
			pending.append(child)

	if has_rect:
		return combined
	return Rect2(Vector2.ZERO, _render_frame_size)

func _node_visual_rect(node: Node) -> Rect2:
	if node is TileMapLayer:
		var layer := node as TileMapLayer
		var used_rect := layer.get_used_rect()
		if used_rect.size == Vector2i.ZERO:
			return Rect2()
		var tile_size := Vector2(layer.tile_set.tile_size) if layer.tile_set != null else Vector2(16.0, 16.0)
		return Rect2(Vector2(used_rect.position) * tile_size + layer.position, Vector2(used_rect.size) * tile_size)

	if node is Polygon2D:
		var polygon_node := node as Polygon2D
		if polygon_node.polygon.is_empty():
			return Rect2()
		var rect := Rect2(polygon_node.polygon[0], Vector2.ZERO)
		for point in polygon_node.polygon:
			rect = rect.expand(point)
		rect.position += polygon_node.position
		return rect

	return Rect2()

func _apply_render_frame_size() -> void:
	if map_loading_overlay != null:
		map_loading_overlay.offset_left = 0.0
		map_loading_overlay.offset_top = 0.0
		map_loading_overlay.offset_right = _render_frame_size.x
		map_loading_overlay.offset_bottom = _render_frame_size.y

func _take_next_op_id() -> int:
	var next_id := _next_op_id
	_next_op_id += 1
	if _next_op_id > 0x7FFFFFFF:
		_next_op_id = 1
	return next_id

func _take_next_move_seq() -> int:
	var next_seq := _next_move_seq
	_next_move_seq += 1
	if _next_move_seq > 0x7FFFFFFF:
		_next_move_seq = 1
	return next_seq
