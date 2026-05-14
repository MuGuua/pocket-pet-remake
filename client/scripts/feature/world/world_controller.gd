extends Node2D

signal scene_loaded(scene_id: String)
signal player_position_changed(position: Vector2)
signal scene_transition_requested(from_scene_id: int, to_scene_id: int)
signal scene_transition_failed(reason: String)

const PLAYER_ANCHOR_POSITION: Vector2 = Vector2(640.0, 360.0)
const SCENE_GRID_TO_PIXELS: float = 24.0
const SCENE_CONFIGS: Dictionary = {
	1: {
		"spawn": Vector2(8.0, 6.0),
		"neighbors": {
			"right": 2,
		},
	},
	2: {
		"spawn": Vector2(2.0, 4.0),
		"neighbors": {
			"left": 1,
			"right": 3,
		},
	},
	3: {
		"spawn": Vector2(3.0, 9.0),
		"neighbors": {
			"left": 2,
		},
	},
}

@onready var local_player_anchor: Node2D = %LocalPlayerAnchor
@onready var local_player: CharacterBody2D = local_player_anchor.get_node("player") as CharacterBody2D
@onready var remote_entities_root: Node2D = %RemoteEntities

var _next_op_id: int = 1
var _next_move_seq: int = 1
var _pending_target_scene_id: int = 0
var _last_loaded_scene_id: int = 0

func _ready() -> void:
	local_player_anchor.position = PLAYER_ANCHOR_POSITION
	if local_player != null and local_player.has_signal("scene_exit_requested"):
		local_player.connect("scene_exit_requested", Callable(self, "_on_local_player_scene_exit_requested"))
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
		_unlock_local_player()
		return

	if accepted:
		return

	_pending_target_scene_id = 0
	_unlock_local_player()
	scene_transition_failed.emit(str(payload.get("reason", "scene transfer rejected")))

func handle_world_resync(payload: Dictionary) -> void:
	GameState.set_world_snapshot(payload)
	_apply_authoritative_snapshot()
	_emit_scene_loaded_if_changed(false)

func request_scene_transition(target_scene_id: int) -> void:
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
	scene_transition_requested.emit(current_scene_id, target_scene_id)
	NetClient.send_command(
		CommandIds.MOVE_INTENT_REQ,
		{
			"op_id": _take_next_op_id(),
			"move_seq": _take_next_move_seq(),
			"scene_id": current_scene_id,
			"target_scene_id": target_scene_id,
		}
	)

func _on_local_player_scene_exit_requested(direction: String) -> void:
	var target_scene_id := _neighbor_scene_id(_current_scene_id(), direction)
	if target_scene_id <= 0:
		_unlock_local_player()
		if local_player != null and local_player.has_method("snap_inside_bounds"):
			local_player.call("snap_inside_bounds", direction)
		scene_transition_failed.emit("no neighboring scene on %s edge" % direction)
		return

	request_scene_transition(target_scene_id)

func _apply_authoritative_snapshot() -> void:
	var scene_id := _current_scene_id()
	local_player_anchor.position = PLAYER_ANCHOR_POSITION

	var self_pos := _extract_self_position(GameState.player_snapshot)
	var local_position := _server_to_local_position(scene_id, self_pos)
	if local_player != null and local_player.has_method("apply_authoritative_position"):
		local_player.call("apply_authoritative_position", local_position)
		if local_player.has_method("set_battle_active"):
			local_player.call("set_battle_active", GameState.is_in_battle)

	_pending_target_scene_id = 0
	_unlock_local_player()
	player_position_changed.emit(local_player_anchor.position + local_position)

func _emit_scene_loaded_if_changed(force_emit: bool) -> void:
	var scene_id := _current_scene_id()
	if force_emit or scene_id != _last_loaded_scene_id:
		_last_loaded_scene_id = scene_id
		scene_loaded.emit(str(scene_id))

func _current_scene_id() -> int:
	return int(GameState.scene_snapshot.get("scene_id", 0))

func _neighbor_scene_id(scene_id: int, direction: String) -> int:
	var scene_config_variant: Variant = SCENE_CONFIGS.get(scene_id, {})
	if scene_config_variant is not Dictionary:
		return 0

	var neighbors_variant: Variant = scene_config_variant.get("neighbors", {})
	if neighbors_variant is not Dictionary:
		return 0
	return int(neighbors_variant.get(direction, 0))

func _extract_self_position(player_snapshot: Dictionary) -> Vector2:
	var x: float = float(player_snapshot.get("x", 0.0))
	var y: float = float(player_snapshot.get("y", 0.0))
	return Vector2(x, y)

func _server_to_local_position(scene_id: int, server_position: Vector2) -> Vector2:
	var scene_config_variant: Variant = SCENE_CONFIGS.get(scene_id, {})
	if scene_config_variant is not Dictionary:
		return Vector2.ZERO

	var spawn_variant: Variant = scene_config_variant.get("spawn", Vector2.ZERO)
	var spawn: Vector2 = spawn_variant if spawn_variant is Vector2 else Vector2.ZERO
	return (server_position - spawn) * SCENE_GRID_TO_PIXELS

func _unlock_local_player() -> void:
	if local_player != null and local_player.has_method("set_scene_transition_locked"):
		local_player.call("set_scene_transition_locked", false)

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
