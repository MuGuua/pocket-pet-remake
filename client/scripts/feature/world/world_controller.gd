extends Node2D

signal scene_loaded(scene_id: String)
signal player_position_changed(position: Vector2)
signal scene_transition_requested(from_scene_id: int, to_scene_id: int)
signal scene_transition_failed(reason: String)

const PLAYER_ANCHOR_POSITION: Vector2 = Vector2(640.0, 360.0)
const PORTAL_ACTIVATION_COOLDOWN_MS: int = 350
const SCENE_GRID_TO_PIXELS: float = 24.0
const SCENE_CONFIGS: Dictionary = {
	1: {
		"scene_path": "res://scenes/maps/scene_1.tscn",
		"spawn": Vector2(8.0, 6.0),
	},
	2: {
		"scene_path": "res://scenes/maps/scene_2.tscn",
		"spawn": Vector2(2.0, 4.0),
	},
	3: {
		"scene_path": "res://scenes/maps/scene_3.tscn",
		"spawn": Vector2(3.0, 9.0),
	},
}

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

func _ready() -> void:
	map_mount.position = PLAYER_ANCHOR_POSITION
	local_player_anchor.position = PLAYER_ANCHOR_POSITION
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

	local_player_anchor.position = PLAYER_ANCHOR_POSITION

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
	player_position_changed.emit(local_player_anchor.position + local_position)

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
	return (server_position - spawn) * SCENE_GRID_TO_PIXELS

func _scene_config(scene_id: int) -> Dictionary:
	var scene_config_variant: Variant = SCENE_CONFIGS.get(scene_id, {})
	return scene_config_variant if scene_config_variant is Dictionary else {}

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
