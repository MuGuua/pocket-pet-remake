extends Control

signal scene_loaded(scene_id: String)
signal player_position_changed(local_position: Vector2, global_position: Vector2)
signal scene_transition_requested(from_scene_id: int, to_scene_id: int)
signal scene_transition_failed(reason: String)
signal npc_interaction_requested(entity_id: int, npc_name: String)

const DEFAULT_RENDER_FRAME_SIZE: Vector2 = Vector2(360.0, 480.0)
const PORTAL_ACTIVATION_COOLDOWN_MS: int = 350
const DEFAULT_GRID_TO_PIXELS: float = 24.0
const BACKGROUND_OVERSCAN_SCALE: float = 3.0
const SCENE_CONFIGS: Dictionary = {
	1: {
		"scene_path": "res://scenes/maps/fashtown/roxus_house.tscn",
		"world_anchor": Vector2(8.0, 12.0),
		"local_anchor": Vector2(113.0, 223.0),
		"grid_to_pixels": 24.0,
		"incoming_portal_local_positions": {
			2001: Vector2(113.0, 223.0),
		},
	},
	2: {
		"scene_path": "res://scenes/maps/fashtown/east_road_of_shanguang_town.tscn",
		"world_anchor": Vector2(4.0, 1.0),
		"local_anchor": Vector2(103.0, 37.0),
		"grid_to_pixels": 24.0,
		"incoming_portal_local_positions": {
			1001: Vector2(103.0, 37.0),
			3001: Vector2(24.0, 85.0),
		},
	},
	3: {
		"scene_path": "res://scenes/maps/fashtown/radiant_market.tscn",
		"world_anchor": Vector2(12.0, 10.0),
		"local_anchor": Vector2(296.0, 282.0),
		"grid_to_pixels": 24.0,
		"incoming_portal_local_positions": {
			2002: Vector2(296.0, 282.0),
			4001: Vector2(120.0, 37.0),
			5001: Vector2(96.0, 299.0),
		},
	},
	4: {
		"scene_path": "res://scenes/maps/fashtown/bei_lu.tscn",
		"world_anchor": Vector2(2.0, 8.0),
		"local_anchor": Vector2(85.0, 217.0),
		"grid_to_pixels": 24.0,
		"incoming_portal_local_positions": {
			3002: Vector2(85.0, 217.0),
		},
	},
	5: {
		"scene_path": "res://scenes/maps/fashtown/xue_xiao.tscn",
		"world_anchor": Vector2(11.0, 2.0),
		"local_anchor": Vector2(268.0, 58.0),
		"grid_to_pixels": 24.0,
		"incoming_portal_local_positions": {
			3003: Vector2(268.0, 58.0),
			6001: Vector2(141.0, 263.0),
		},
	},
	6: {
		"scene_path": "res://scenes/maps/fashtown/da_guai_qu.tscn",
		"world_anchor": Vector2(6.0, 10.0),
		"local_anchor": Vector2(144.0, 41.0),
		"grid_to_pixels": 24.0,
		"incoming_portal_local_positions": {
			5002: Vector2(144.0, 41.0),
		},
	},
}

@onready var game_shell: PanelContainer = %GameShell
@onready var game_viewport_container: SubViewportContainer = %GameViewportContainer
@onready var game_viewport: SubViewport = %GameViewport
@onready var background_fill: Sprite2D = %BackgroundFill
@onready var game_root: Node2D = %GameRoot
@onready var player_node: CharacterBody2D = %Player
@onready var map_loading_overlay: ColorRect = %MapLoadingOverlay

var _next_op_id: int = 1
var _next_move_seq: int = 1
var _pending_target_scene_id: int = 0
var _pending_portal_id: int = 0
var _pending_player_spawn_position: Vector2 = Vector2.ZERO
var _pending_player_spawn_requested: bool = false
var _pending_player_facing_direction: Vector2 = Vector2.ZERO
var _pending_player_facing_requested: bool = false
var _last_loaded_scene_id: int = 0
var _loaded_scene_id: int = 0
var _portal_cooldown_until_ms: int = 0
var _render_frame_size: Vector2 = DEFAULT_RENDER_FRAME_SIZE
var _use_scene_login_spawn_on_next_snapshot: bool = false
var _last_reported_player_position: Vector2 = Vector2.INF
var _current_level: Node2D
var _current_interactable_entity_id: int = 0
var _current_interactable_npc_name: String = ""

func _process(_delta: float) -> void:
	_report_player_position_if_changed()
	_process_npc_interaction_input()

func _ready() -> void:
	_refresh_game_layout()
	if get_viewport() != null and not get_viewport().size_changed.is_connected(_on_viewport_size_changed):
		get_viewport().size_changed.connect(_on_viewport_size_changed)
	if game_viewport_container != null and not game_viewport_container.resized.is_connected(_refresh_game_layout):
		game_viewport_container.resized.connect(_refresh_game_layout)
	GameState.battle_changed.connect(_sync_local_player_battle_state)
	call_deferred("_refresh_game_layout")
	_sync_local_player_battle_state()

func handle_enter_world(payload: Dictionary) -> void:
	_use_scene_login_spawn_on_next_snapshot = true
	_pending_player_facing_requested = true
	_pending_player_facing_direction = Vector2.DOWN
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
		_pending_player_facing_requested = false
		_set_transition_loading(false)
		_unlock_local_player()
		return

	if accepted:
		return

	_pending_target_scene_id = 0
	_pending_portal_id = 0
	_pending_player_facing_requested = false
	_set_transition_loading(false)
	_unlock_local_player()
	scene_transition_failed.emit(str(payload.get("reason", "scene transfer rejected")))

func handle_world_resync(payload: Dictionary) -> void:
	GameState.set_world_snapshot(payload)
	_apply_authoritative_snapshot()
	_emit_scene_loaded_if_changed(false)

func request_scene_transition(target_scene_id: int, portal_id: int = 0, facing_direction: Vector2 = Vector2.ZERO) -> void:
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
	_pending_player_facing_requested = facing_direction != Vector2.ZERO
	if _pending_player_facing_requested:
		_pending_player_facing_direction = facing_direction
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

func set_render_frame_size(size: Vector2) -> void:
	if size.x <= 0.0 or size.y <= 0.0:
		return
	_render_frame_size = size
	_refresh_game_layout()

func load_level(scene_path: String) -> void:
	if scene_path.is_empty():
		push_warning("Level scene path is empty.")
		return

	var level_scene := load(scene_path) as PackedScene
	if level_scene == null:
		push_warning("Failed to load level scene: %s" % scene_path)
		return

	mount_level(level_scene)

func mount_level(level_scene: PackedScene) -> void:
	unmount_current_level()

	var level_instance := level_scene.instantiate()
	if level_instance is not Node2D:
		push_warning("Mounted level must inherit Node2D.")
		level_instance.queue_free()
		return

	_current_level = level_instance as Node2D
	game_root.add_child(_current_level)
	_bind_level_signals(_current_level)
	_bind_interactive_npcs(_current_level)
	_refresh_background_fill()
	_attach_player_to_current_level()
	_apply_pending_player_transition()
	_apply_level_camera_limits()
	_keep_player_on_top()

func unmount_current_level() -> void:
	if _current_level == null:
		return

	_clear_current_interactable_npc()
	_detach_player_from_current_level()
	var level_to_free := _current_level
	_current_level = null
	if level_to_free.get_parent() == game_root:
		game_root.remove_child(level_to_free)
	level_to_free.queue_free()

func _on_viewport_size_changed() -> void:
	_refresh_game_layout()

func _refresh_game_layout() -> void:
	_resize_game_viewport()
	_refresh_background_fill()
	if map_loading_overlay != null:
		map_loading_overlay.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)

func _resize_game_viewport() -> void:
	if game_viewport == null:
		return
	var viewport_size: Vector2 = game_viewport_container.size
	if viewport_size.x <= 0.0 or viewport_size.y <= 0.0:
		viewport_size = size
	if viewport_size.x <= 0.0 or viewport_size.y <= 0.0:
		viewport_size = _render_frame_size
	game_viewport.size = viewport_size.floor()

func _refresh_background_fill() -> void:
	if background_fill == null:
		return

	var background_texture := background_fill.texture
	if background_texture == null:
		return

	var texture_size := background_texture.get_size()
	if texture_size.x <= 0.0 or texture_size.y <= 0.0:
		return

	var level_rect := _resolve_level_world_rect(_current_level)
	if not level_rect.has_area():
		var viewport_size := Vector2(game_viewport.size)
		if viewport_size.x <= 0.0 or viewport_size.y <= 0.0:
			viewport_size = _render_frame_size
		background_fill.position = Vector2.ZERO
		background_fill.scale = Vector2(
			viewport_size.x * BACKGROUND_OVERSCAN_SCALE / texture_size.x,
			viewport_size.y * BACKGROUND_OVERSCAN_SCALE / texture_size.y
		)
		return

	var background_size := level_rect.size * BACKGROUND_OVERSCAN_SCALE
	background_fill.position = level_rect.get_center() - background_size * 0.5
	background_fill.scale = Vector2(
		background_size.x / texture_size.x,
		background_size.y / texture_size.y
	)

func _bind_level_signals(level: Node2D) -> void:
	_connect_level_signal(level, "scene_change_requested", Callable(self, "_on_level_scene_change_requested"))

func _connect_level_signal(level: Node2D, signal_name: StringName, callable: Callable) -> void:
	if not level.has_signal(signal_name):
		return
	if level.is_connected(signal_name, callable):
		return
	level.connect(signal_name, callable)

func _on_level_scene_change_requested(change_request: Variant) -> void:
	if Time.get_ticks_msec() < _portal_cooldown_until_ms:
		return
	if _pending_target_scene_id != 0:
		return
	if change_request is not Dictionary:
		return
	var request := change_request as Dictionary
	request_scene_transition(
		int(request.get("target_scene_id", 0)),
		int(request.get("portal_id", 0)),
		request.get("facing_direction", Vector2.ZERO) if request.get("facing_direction", Vector2.ZERO) is Vector2 else Vector2.ZERO
	)

func _apply_authoritative_snapshot() -> void:
	var scene_id := _current_scene_id()
	if not _ensure_scene_loaded(scene_id):
		_pending_target_scene_id = 0
		_pending_portal_id = 0
		_pending_player_facing_requested = false
		_set_transition_loading(false)
		_unlock_local_player()
		scene_transition_failed.emit("failed to load scene map: %d" % scene_id)
		return

	var self_pos := _extract_self_position(GameState.player_snapshot)
	var spawn_position: Vector2 = _resolve_snapshot_spawn_position(scene_id, self_pos)
	_stage_pending_player_transition({"spawn_position": spawn_position})
	_apply_pending_player_transition()
	_apply_level_camera_limits()
	_keep_player_on_top()
	_refresh_background_fill()

	_pending_target_scene_id = 0
	_pending_portal_id = 0
	_pending_player_facing_requested = false
	_portal_cooldown_until_ms = Time.get_ticks_msec() + PORTAL_ACTIVATION_COOLDOWN_MS
	_set_transition_loading(false)
	_unlock_local_player()
	player_position_changed.emit(_current_player_scene_position(), _current_player_global_position())

func _ensure_scene_loaded(scene_id: int) -> bool:
	if scene_id <= 0:
		return false
	if _loaded_scene_id == scene_id and is_instance_valid(_current_level):
		_attach_player_to_current_level()
		return true

	var scene_config := _scene_config(scene_id)
	var scene_path := str(scene_config.get("scene_path", ""))
	if scene_path.is_empty():
		return false

	load_level(scene_path)
	if not is_instance_valid(_current_level):
		return false
	_loaded_scene_id = scene_id
	_refresh_game_layout()
	return true

func _emit_scene_loaded_if_changed(force_emit: bool) -> void:
	var scene_id := _current_scene_id()
	if force_emit or scene_id != _last_loaded_scene_id:
		_last_loaded_scene_id = scene_id
		scene_loaded.emit(str(scene_id))

func _current_scene_id() -> int:
	return int(GameState.scene_snapshot.get("scene_id", 0))

func _extract_self_position(player_snapshot: Dictionary) -> Vector2:
	return Vector2(float(player_snapshot.get("x", 0.0)), float(player_snapshot.get("y", 0.0)))

func _scene_config(scene_id: int) -> Dictionary:
	var scene_config_variant: Variant = SCENE_CONFIGS.get(scene_id, {})
	return scene_config_variant if scene_config_variant is Dictionary else {}

func _server_to_local_position(scene_id: int, server_position: Vector2) -> Vector2:
	var scene_config := _scene_config(scene_id)
	if scene_config.is_empty():
		return Vector2.ZERO
	var world_anchor: Vector2 = scene_config.get("world_anchor", Vector2.ZERO)
	var local_anchor: Vector2 = scene_config.get("local_anchor", Vector2.ZERO)
	var grid_to_pixels: float = float(scene_config.get("grid_to_pixels", DEFAULT_GRID_TO_PIXELS))
	return local_anchor + (server_position - world_anchor) * grid_to_pixels

func _resolve_snapshot_spawn_position(scene_id: int, server_position: Vector2) -> Vector2:
	var portal_spawn_position: Vector2 = _resolve_pending_portal_spawn_position(scene_id)
	if portal_spawn_position != Vector2.ZERO:
		return portal_spawn_position
	if _use_scene_login_spawn_on_next_snapshot:
		_use_scene_login_spawn_on_next_snapshot = false
		var level_spawn_position: Vector2 = _resolve_level_login_spawn_position()
		if level_spawn_position != Vector2.ZERO:
			return level_spawn_position
	return _server_to_local_position(scene_id, server_position)

func _resolve_level_login_spawn_position() -> Vector2:
	if _current_level != null and _current_level.has_method("get_login_spawn_position"):
		var spawn_position_value: Variant = _current_level.call("get_login_spawn_position")
		if spawn_position_value is Vector2:
			return spawn_position_value as Vector2
	return Vector2.ZERO

func _resolve_pending_portal_spawn_position(scene_id: int) -> Vector2:
	if _pending_portal_id == 0:
		return Vector2.ZERO
	var scene_config := _scene_config(scene_id)
	var portal_positions_value: Variant = scene_config.get("incoming_portal_local_positions", {})
	if portal_positions_value is Dictionary:
		var portal_positions := portal_positions_value as Dictionary
		var spawn_position_value: Variant = portal_positions.get(_pending_portal_id, Vector2.ZERO)
		if spawn_position_value is Vector2:
			return spawn_position_value as Vector2
	return Vector2.ZERO


func _stage_pending_player_transition(change_request: Dictionary) -> void:
	var spawn_position_value: Variant = change_request.get("spawn_position", null)
	_pending_player_spawn_requested = spawn_position_value is Vector2
	if _pending_player_spawn_requested:
		_pending_player_spawn_position = spawn_position_value as Vector2

	if _pending_player_facing_requested:
		return
	var facing_direction_value: Variant = change_request.get("facing_direction", null)
	_pending_player_facing_requested = facing_direction_value is Vector2 and facing_direction_value != Vector2.ZERO
	if _pending_player_facing_requested:
		_pending_player_facing_direction = facing_direction_value as Vector2

func _apply_pending_player_transition() -> bool:
	if player_node == null:
		return false

	var applied := false
	if _pending_player_spawn_requested:
		if player_node.has_method("apply_authoritative_position"):
			player_node.call("apply_authoritative_position", _pending_player_spawn_position)
		else:
			player_node.position = _pending_player_spawn_position
		_pending_player_spawn_requested = false
		applied = true

	if _pending_player_facing_requested:
		if player_node.has_method("set_facing_direction"):
			player_node.call("set_facing_direction", _pending_player_facing_direction)
		elif "cardinal_direction" in player_node:
			player_node.set("cardinal_direction", _pending_player_facing_direction)
		_pending_player_facing_requested = false
		applied = true

	return applied


func _bind_interactive_npcs(level: Node) -> void:
	if level == null:
		return
	for child in level.find_children("*", "", true, false):
		if child.has_signal("interaction_entered"):
			var entered := Callable(self, "_on_npc_interaction_entered")
			if not child.is_connected("interaction_entered", entered):
				child.connect("interaction_entered", entered)
		if child.has_signal("interaction_exited"):
			var exited := Callable(self, "_on_npc_interaction_exited")
			if not child.is_connected("interaction_exited", exited):
				child.connect("interaction_exited", exited)

func _on_npc_interaction_entered(entity_id: int, npc_name: String) -> void:
	_current_interactable_entity_id = entity_id
	_current_interactable_npc_name = npc_name

func _on_npc_interaction_exited(entity_id: int, _npc_name: String) -> void:
	if _current_interactable_entity_id != entity_id:
		return
	_clear_current_interactable_npc()

func _clear_current_interactable_npc() -> void:
	_current_interactable_entity_id = 0
	_current_interactable_npc_name = ""

func _process_npc_interaction_input() -> void:
	if GameState.is_in_battle or _pending_target_scene_id != 0:
		return
	if _current_interactable_entity_id <= 0:
		return
	if not Input.is_action_just_pressed("ui_accept"):
		return
	npc_interaction_requested.emit(_current_interactable_entity_id, _current_interactable_npc_name)

func _attach_player_to_current_level() -> void:
	if _current_level == null or player_node == null:
		return

	var target_parent := _get_player_host(_current_level)
	if player_node.get_parent() == target_parent:
		return
	player_node.reparent(target_parent, true)

func _detach_player_from_current_level() -> void:
	if player_node == null:
		return
	if player_node.get_parent() == game_viewport:
		return
	player_node.reparent(game_viewport, true)

func _get_player_host(level: Node2D) -> Node:
	var actor_root := level.get_node_or_null("ActorRoot")
	if actor_root != null:
		return actor_root
	return level

func _keep_player_on_top() -> void:
	if player_node == null:
		return
	var player_parent := player_node.get_parent()
	if player_parent == null:
		return
	player_parent.move_child(player_node, player_parent.get_child_count() - 1)

func _apply_level_camera_limits() -> void:
	if _current_level == null or player_node == null:
		return

	var limits := _resolve_level_camera_limits(_current_level)
	if limits.is_empty():
		return

	if player_node.has_method("set_camera_limits"):
		player_node.call(
			"set_camera_limits",
			int(limits.get("left", -10000000)),
			int(limits.get("top", -10000000)),
			int(limits.get("right", 10000000)),
			int(limits.get("bottom", 10000000))
		)

func _resolve_level_camera_limits(level: Node2D) -> Dictionary:
	if level.has_method("get_camera_limits"):
		var custom_limits: Variant = level.call("get_camera_limits")
		if custom_limits is Dictionary and not custom_limits.is_empty():
			return custom_limits

	var limit_layer := _get_camera_limit_layer(level)
	if limit_layer == null:
		return {}

	var used_rect: Rect2i = limit_layer.get_used_rect()
	if not used_rect.has_area():
		return {}

	var tile_set := limit_layer.tile_set
	if tile_set == null:
		return {}

	var tile_size := Vector2(tile_set.tile_size)
	var top_left_local := limit_layer.map_to_local(used_rect.position) - tile_size * 0.5
	var bottom_right_cell := used_rect.position + used_rect.size - Vector2i.ONE
	var bottom_right_local := limit_layer.map_to_local(bottom_right_cell) + tile_size * 0.5
	var top_left_global := limit_layer.to_global(top_left_local)
	var bottom_right_global := limit_layer.to_global(bottom_right_local)

	return {
		"left": int(round(top_left_global.x)),
		"top": int(round(top_left_global.y)),
		"right": int(round(bottom_right_global.x)),
		"bottom": int(round(bottom_right_global.y)),
	}

func _get_camera_limit_layer(level: Node2D) -> TileMapLayer:
	for layer_name in ["Collision", "Bottom", "Map", "TileMapLayer"]:
		var layer := level.get_node_or_null(layer_name) as TileMapLayer
		if layer != null:
			return layer
	return null

func _resolve_level_world_rect(level: Node2D) -> Rect2:
	if level == null:
		return Rect2()

	var limit_layer := _get_camera_limit_layer(level)
	if limit_layer == null:
		return Rect2()

	var used_rect: Rect2i = limit_layer.get_used_rect()
	if not used_rect.has_area():
		return Rect2()

	var tile_set := limit_layer.tile_set
	if tile_set == null:
		return Rect2()

	var tile_size := Vector2(tile_set.tile_size)
	var top_left_local := limit_layer.map_to_local(used_rect.position) - tile_size * 0.5
	var bottom_right_cell := used_rect.position + used_rect.size - Vector2i.ONE
	var bottom_right_local := limit_layer.map_to_local(bottom_right_cell) + tile_size * 0.5
	var top_left_global := limit_layer.to_global(top_left_local)
	var bottom_right_global := limit_layer.to_global(bottom_right_local)

	return Rect2(top_left_global, bottom_right_global - top_left_global)

func _sync_local_player_battle_state() -> void:
	if player_node != null and player_node.has_method("set_battle_active"):
		player_node.call("set_battle_active", GameState.is_in_battle)

func _lock_local_player() -> void:
	if player_node != null and player_node.has_method("set_scene_transition_locked"):
		player_node.call("set_scene_transition_locked", true)

func _unlock_local_player() -> void:
	if player_node != null and player_node.has_method("set_scene_transition_locked"):
		player_node.call("set_scene_transition_locked", false)

func _set_transition_loading(active: bool) -> void:
	if map_loading_overlay != null:
		map_loading_overlay.visible = active

func _take_next_op_id() -> int:
	var op_id := _next_op_id
	_next_op_id += 1
	return op_id

func _take_next_move_seq() -> int:
	var move_seq := _next_move_seq
	_next_move_seq += 1
	return move_seq

func _current_player_global_position() -> Vector2:
	if player_node == null or not is_instance_valid(player_node):
		return Vector2.ZERO
	return player_node.global_position

func _current_player_scene_position() -> Vector2:
	if player_node == null or not is_instance_valid(player_node):
		return Vector2.ZERO
	if _current_level != null and is_instance_valid(_current_level):
		return _current_level.to_local(player_node.global_position)
	return player_node.position

func _report_player_position_if_changed() -> void:
	if player_node == null or not is_instance_valid(player_node):
		return
	var current_position: Vector2 = _current_player_scene_position()
	if current_position.is_equal_approx(_last_reported_player_position):
		return
	_last_reported_player_position = current_position
	player_position_changed.emit(current_position, _current_player_global_position())
