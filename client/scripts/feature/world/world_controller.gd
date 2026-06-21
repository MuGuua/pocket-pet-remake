extends Control

signal scene_loaded(scene_id: String)
signal player_position_changed(local_position: Vector2, global_position: Vector2)
signal scene_transition_requested(from_scene_id: int, to_scene_id: int)
signal scene_transition_failed(reason: String)
signal npc_interaction_requested(entity_id: int, npc_name: String)
signal wild_encounter_responded(accepted: bool, reason: String)

const DEFAULT_RENDER_FRAME_SIZE: Vector2 = Vector2(360.0, 480.0)
const PORTAL_ACTIVATION_COOLDOWN_MS: int = 350
const DEFAULT_GRID_TO_PIXELS: float = 24.0
const WILD_ENCOUNTER_RATE_DENOMINATOR: int = 10000
const BACKGROUND_OVERSCAN_SCALE: float = 3.0
const INVALID_NAVIGATION_CELL: Vector2i = Vector2i(-2147483648, -2147483648)
const CLICK_MARKER_RING_RADIUS: float = 10.0
const CLICK_MARKER_CROSS_HALF_SIZE: float = 4.0
const CLICK_MARKER_WIDTH: float = 2.0
const CLICK_MARKER_ANIMATION_DURATION: float = 0.28
## 每帧最多记录的宠物跟随路径步数，防止主角锚点异常时在单帧 while 中卡死。
const PET_FOLLOW_MAX_LEADER_STEPS_PER_FRAME: int = 6
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
## 地图转场视觉锁：过渡前半段为 true，推迟实际换图直到中点。
var _scene_visual_apply_locked: bool = false
## 视觉锁期间收到的 WORLD_RESYNC 是否待在中点应用。
var _deferred_scene_apply_pending: bool = false
var _last_loaded_scene_id: int = 0
var _loaded_scene_id: int = 0
var _portal_cooldown_until_ms: int = 0
var _render_frame_size: Vector2 = DEFAULT_RENDER_FRAME_SIZE
var _use_scene_login_spawn_on_next_snapshot: bool = false
var _last_reported_player_position: Vector2 = Vector2.INF
var _current_level: Node2D
var _current_interactable_entity_id: int = 0
var _current_interactable_npc_name: String = ""
var _current_interactable_requested: bool = false
var _runtime_input_locked: bool = false
var _navigation_grid: AStarGrid2D
var _navigation_layer: TileMapLayer
var _navigation_region: Rect2i = Rect2i()
var _click_destination_marker_root: Node2D
var _click_destination_marker_ring: Line2D
var _click_destination_marker_cross: Line2D
var _click_destination_marker_tween: Tween
var _last_wild_encounter_nav_cell: Vector2i = INVALID_NAVIGATION_CELL
var _wild_encounter_request_pending: bool = false
# 战斗弹窗仍可见时由主场景置位，避免结算演出阶段提前切回 idle 动画。
var _force_battle_pose_active: bool = false
## 世界场景宠物跟随节点。
var _pet_follower: WorldPetFollower = null
## 主角路径记录锚点，用于按 24px 格距压入跟随路径。
var _leader_path_anchor: Vector2 = Vector2.INF

func _process(delta: float) -> void:
	_update_pet_follow(delta)
	_sync_local_actor_y_sort()
	_report_player_position_if_changed()
	_process_npc_interaction_input()
	_check_wild_encounter_step()

func _ready() -> void:
	_refresh_game_layout()
	if get_viewport() != null and not get_viewport().size_changed.is_connected(_on_viewport_size_changed):
		get_viewport().size_changed.connect(_on_viewport_size_changed)
	if game_viewport_container != null and not game_viewport_container.resized.is_connected(_refresh_game_layout):
		game_viewport_container.resized.connect(_refresh_game_layout)
	if game_viewport_container != null and not game_viewport_container.gui_input.is_connected(_on_game_viewport_gui_input):
		game_viewport_container.gui_input.connect(_on_game_viewport_gui_input)
	GameState.battle_changed.connect(_sync_local_player_battle_state)
	GameState.battle_changed.connect(_on_battle_state_changed)
	GameState.pets_changed.connect(_sync_pet_follower_lineup)
	_ensure_pet_follower()
	_ensure_click_destination_marker()
	call_deferred("_refresh_game_layout")
	_sync_local_player_battle_state()

func handle_enter_world(payload: Dictionary) -> void:
	var already_in_world: bool = int(GameState.scene_snapshot.get("scene_id", 0)) > 0
	if not already_in_world:
		_use_scene_login_spawn_on_next_snapshot = true
		_pending_player_facing_requested = true
		_pending_player_facing_direction = Vector2.DOWN
	GameState.set_world_snapshot(payload)
	if already_in_world:
		# 人物面板等场景只会刷新属性，不应重置当前地图站位。
		return
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
	if _scene_visual_apply_locked:
		_deferred_scene_apply_pending = true
		return
	_apply_authoritative_snapshot()
	_emit_scene_loaded_if_changed(false)

func handle_wild_encounter_response(payload: Dictionary) -> void:
	var accepted: bool = bool(payload.get("accepted", false))
	var reason: String = str(payload.get("reason", ""))
	_wild_encounter_request_pending = false
	_set_transition_loading(false)
	if not accepted:
		set_runtime_input_locked(false)
		_unlock_local_player()
	wild_encounter_responded.emit(accepted, reason)

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

## 主场景在地图过渡开始时加锁，避免新地图在渐入前半段就渲染到视口。
func set_scene_visual_apply_locked(locked: bool) -> void:
	_scene_visual_apply_locked = locked
	if not locked:
		_flush_deferred_scene_apply()


## 过渡动画到达中点时应用已缓存的世界快照，并触发 scene_loaded。
func flush_deferred_scene_apply_at_midpoint() -> void:
	_scene_visual_apply_locked = false
	_flush_deferred_scene_apply()


## 切图失败或过渡中断时清理视觉锁，丢弃尚未应用的延迟快照。
func cancel_scene_visual_apply_lock() -> void:
	_scene_visual_apply_locked = false
	_deferred_scene_apply_pending = false


func _flush_deferred_scene_apply() -> void:
	if not _deferred_scene_apply_pending:
		return
	_deferred_scene_apply_pending = false
	_apply_authoritative_snapshot()
	_emit_scene_loaded_if_changed(false)

func set_render_frame_size(size: Vector2) -> void:
	if size.x <= 0.0 or size.y <= 0.0:
		return
	var normalized_size: Vector2 = size.floor()
	if normalized_size == _render_frame_size.floor():
		return
	_render_frame_size = normalized_size
	_refresh_game_layout()

## 截取当前世界地图视口画面，供战斗场景作为背景使用。
func capture_current_map_snapshot() -> Texture2D:
	if game_viewport == null:
		return null
	var viewport_texture: ViewportTexture = game_viewport.get_texture()
	if viewport_texture == null:
		return null
	var snapshot_image: Image = viewport_texture.get_image()
	if snapshot_image == null or snapshot_image.is_empty():
		return null
	return ImageTexture.create_from_image(snapshot_image)

## 异步截取世界地图：先隐藏玩家与点击标记，等待一帧渲染后再读取视口。
func capture_current_map_snapshot_async() -> Texture2D:
	if game_viewport == null:
		return null
	var hidden_nodes: Array[CanvasItem] = []
	if player_node != null and player_node.visible:
		player_node.visible = false
		hidden_nodes.append(player_node)
	if _click_destination_marker_root != null and _click_destination_marker_root.visible:
		_click_destination_marker_root.visible = false
		hidden_nodes.append(_click_destination_marker_root)
	await get_tree().process_frame
	var snapshot: Texture2D = capture_current_map_snapshot()
	for node: CanvasItem in hidden_nodes:
		node.visible = true
	return snapshot

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
	_attach_pet_follower_to_current_level()
	_apply_pending_player_transition()
	_reset_pet_follow_near_player()
	_apply_level_camera_limits()
	_rebuild_navigation_grid()
	_sync_local_actor_y_sort()
	_sync_pet_follower_lineup()

func unmount_current_level() -> void:
	if _current_level == null:
		return

	_clear_current_interactable_npc()
	_clear_navigation_grid()
	_detach_player_from_current_level()
	_detach_pet_follower_from_current_level()
	var level_to_free := _current_level
	_current_level = null
	if level_to_free.get_parent() == game_root:
		game_root.remove_child(level_to_free)
	level_to_free.queue_free()

func _on_viewport_size_changed() -> void:
	_refresh_game_layout()

func _on_game_viewport_gui_input(event: InputEvent) -> void:
	if _runtime_input_locked or GameState.is_in_battle or _pending_target_scene_id != 0:
		return
	if _current_level == null or player_node == null:
		return

	if event is InputEventMouseButton:
		var mouse_event := event as InputEventMouseButton
		if mouse_event.button_index != MOUSE_BUTTON_LEFT or not mouse_event.pressed:
			return
		_request_click_to_move(mouse_event.position)
		return

	if event is InputEventScreenTouch:
		var touch_event := event as InputEventScreenTouch
		if not touch_event.pressed:
			return
		_request_click_to_move(touch_event.position)

func _request_click_to_move(container_position: Vector2) -> void:
	var target_world_position: Vector2 = _container_to_world_position(container_position)
	if target_world_position == Vector2.INF:
		return
	_request_auto_move_to_world(target_world_position)

func _container_to_world_position(container_position: Vector2) -> Vector2:
	if game_viewport_container == null or game_viewport == null:
		return Vector2.INF
	if game_viewport_container.size.x <= 0.0 or game_viewport_container.size.y <= 0.0:
		return Vector2.INF

	# 把容器坐标按实际显示比例换算回 SubViewport 坐标，再通过画布变换还原到世界坐标。
	var viewport_scale := Vector2(
		float(game_viewport.size.x) / game_viewport_container.size.x,
		float(game_viewport.size.y) / game_viewport_container.size.y
	)
	var viewport_position := container_position * viewport_scale
	return game_viewport.get_canvas_transform().affine_inverse() * viewport_position

func _request_auto_move_to_world(target_world_position: Vector2) -> void:
	if player_node == null or _navigation_grid == null or _navigation_layer == null:
		return

	var start_cell: Vector2i = _resolve_walkable_navigation_cell(_world_to_navigation_cell(player_node.global_position))
	var target_cell: Vector2i = _resolve_walkable_navigation_cell(_world_to_navigation_cell(target_world_position))
	if not _is_navigation_cell_valid(start_cell) or not _is_navigation_cell_valid(target_cell):
		return

	var cell_path: Array[Vector2i] = _navigation_grid.get_id_path(start_cell, target_cell)
	if cell_path.is_empty():
		if player_node.has_method("clear_auto_move_path"):
			player_node.call("clear_auto_move_path")
		return

	var local_path: Array[Vector2] = []
	for path_cell in cell_path:
		local_path.append(_navigation_cell_to_player_local_position(path_cell))

	if player_node.has_method("set_auto_move_path"):
		player_node.call("set_auto_move_path", local_path)
	_show_click_destination_marker(_navigation_cell_to_world_position(target_cell))

func _refresh_game_layout() -> void:
	_resize_game_viewport()
	_refresh_background_fill()
	if map_loading_overlay != null:
		map_loading_overlay.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)

func _resize_game_viewport() -> void:
	if game_viewport == null:
		return
	# SubViewportContainer 开启 stretch 时会自动同步子视口尺寸，手动赋值会触发 Godot 警告并造成重复布局。
	if game_viewport_container != null and game_viewport_container.stretch:
		return
	var viewport_size: Vector2 = game_viewport_container.size if game_viewport_container != null else Vector2.ZERO
	if viewport_size.x <= 0.0 or viewport_size.y <= 0.0:
		viewport_size = size
	if viewport_size.x <= 0.0 or viewport_size.y <= 0.0:
		viewport_size = _render_frame_size
	var normalized_size: Vector2i = Vector2i(viewport_size.floor())
	if game_viewport.size == normalized_size:
		return
	game_viewport.size = normalized_size

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
	_attach_pet_follower_to_current_level()
	_reset_pet_follow_near_player()
	_sync_pet_follower_lineup()
	_apply_level_camera_limits()
	_sync_local_actor_y_sort()
	_refresh_background_fill()

	_pending_target_scene_id = 0
	_pending_portal_id = 0
	_pending_player_facing_requested = false
	_portal_cooldown_until_ms = Time.get_ticks_msec() + PORTAL_ACTIVATION_COOLDOWN_MS
	_set_transition_loading(false)
	_unlock_local_player()
	player_position_changed.emit(_current_player_scene_position(), _current_player_global_position())
	_reset_wild_encounter_step_tracking()

func _reset_wild_encounter_step_tracking() -> void:
	_last_wild_encounter_nav_cell = INVALID_NAVIGATION_CELL
	if player_node == null or _navigation_layer == null:
		return
	var current_cell: Vector2i = _world_to_navigation_cell(player_node.global_position)
	if _is_navigation_cell_valid(current_cell):
		_last_wild_encounter_nav_cell = current_cell

func _check_wild_encounter_step() -> void:
	if _wild_encounter_request_pending or GameState.is_in_battle:
		return
	if _pending_target_scene_id != 0 or _runtime_input_locked:
		return
	if player_node == null or _navigation_layer == null or _navigation_grid == null:
		return
	if not bool(GameState.wild_encounter_config.get("enabled", false)):
		return

	var config_scene_id: int = int(GameState.wild_encounter_config.get("scene_id", 0))
	var current_scene_id: int = _current_scene_id()
	if config_scene_id <= 0 or config_scene_id != current_scene_id:
		return

	if not _is_player_moving_for_wild_encounter():
		return

	var current_cell: Vector2i = _world_to_navigation_cell(player_node.global_position)
	if not _is_navigation_cell_valid(current_cell):
		return
	if _last_wild_encounter_nav_cell == INVALID_NAVIGATION_CELL:
		_last_wild_encounter_nav_cell = current_cell
		return
	if current_cell == _last_wild_encounter_nav_cell:
		return

	_last_wild_encounter_nav_cell = current_cell
	_try_roll_wild_encounter(current_scene_id)

func _is_player_moving_for_wild_encounter() -> bool:
	if player_node == null:
		return false
	if player_node.velocity.length_squared() <= 0.0:
		return false
	return true

func _try_roll_wild_encounter(scene_id: int) -> void:
	var encounter_rate: int = int(GameState.wild_encounter_config.get("encounter_rate", 0))
	if encounter_rate <= 0:
		return
	var roll_value: int = randi() % WILD_ENCOUNTER_RATE_DENOMINATOR
	if roll_value >= encounter_rate:
		return
	_request_wild_encounter_battle(scene_id)

func _request_wild_encounter_battle(scene_id: int) -> void:
	_wild_encounter_request_pending = true
	_lock_local_player()
	set_runtime_input_locked(true)
	if player_node != null and player_node.has_method("clear_auto_move_path"):
		player_node.call("clear_auto_move_path")
	_set_transition_loading(true)
	App.request_wild_encounter(scene_id, _take_next_move_seq())

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
	# 进入新的 NPC 检测区后，先刷新当前锁定目标，
	# 再自动发起一次服务端交互请求，用统一的 entity_id 拉取菜单或对话。
	_current_interactable_entity_id = entity_id
	_current_interactable_npc_name = npc_name
	_current_interactable_requested = false
	_request_current_npc_interaction_if_needed()

func _on_npc_interaction_exited(entity_id: int, _npc_name: String) -> void:
	if _current_interactable_entity_id != entity_id:
		return
	_clear_current_interactable_npc()

func _clear_current_interactable_npc() -> void:
	_current_interactable_entity_id = 0
	_current_interactable_npc_name = ""
	_current_interactable_requested = false

func _request_current_npc_interaction_if_needed() -> void:
	if _current_interactable_requested:
		return
	if GameState.is_in_battle or _pending_target_scene_id != 0 or _runtime_input_locked:
		return
	if _current_interactable_entity_id <= 0:
		return

	# 每次进入检测区只请求一次，避免角色在碰撞区内轻微抖动时重复拉取服务端菜单。
	_current_interactable_requested = true
	npc_interaction_requested.emit(_current_interactable_entity_id, _current_interactable_npc_name)

func _process_npc_interaction_input() -> void:
	if GameState.is_in_battle or _pending_target_scene_id != 0 or _runtime_input_locked:
		return
	if _current_interactable_entity_id <= 0:
		return
	if _current_interactable_requested:
		return
	if not Input.is_action_just_pressed("ui_accept"):
		return
	_request_current_npc_interaction_if_needed()

func _attach_player_to_current_level() -> void:
	if _current_level == null or player_node == null:
		return

	var target_parent := _resolve_actor_root(_current_level)
	if player_node.get_parent() == target_parent:
		_attach_pet_follower_to_current_level()
		return
	player_node.reparent(target_parent, true)
	_configure_actor_y_sort(player_node)
	_attach_pet_follower_to_current_level()

func _detach_player_from_current_level() -> void:
	if player_node == null:
		return
	if player_node.get_parent() == game_viewport:
		return
	player_node.reparent(game_viewport, true)

func _resolve_actor_root(level: Node2D) -> Node2D:
	var actor_root_variant: Node = level.get_node_or_null("ActorRoot")
	if actor_root_variant is Node2D:
		var actor_root: Node2D = actor_root_variant as Node2D
		actor_root.y_sort_enabled = true
		return actor_root
	var created_root: Node2D = Node2D.new()
	created_root.name = "ActorRoot"
	created_root.y_sort_enabled = true
	level.add_child(created_root)
	return created_root


## 配置动态角色参与 ActorRoot Y-Sort：关闭子级 Y-Sort，避免重复排序。
func _configure_actor_y_sort(actor: Node2D) -> void:
	if actor == null:
		return
	actor.y_sort_enabled = false


## 同行/同格时让宠物排在主角之前绘制，作为 Y 值相同时的遮挡 tie-break。
func _apply_pet_player_draw_tiebreak(actor_root: Node2D) -> void:
	if _pet_follower == null or player_node == null or not _pet_follower.visible:
		return
	if _pet_follower.get_parent() != actor_root or player_node.get_parent() != actor_root:
		return
	var player_sort_y: float = player_node.position.y
	var pet_sort_y: float = _pet_follower.position.y
	if absf(player_sort_y - pet_sort_y) >= 1.0:
		return
	var pet_index: int = _pet_follower.get_index()
	var player_index: int = player_node.get_index()
	if pet_index > player_index:
		actor_root.move_child(_pet_follower, player_index)


## 每帧确保 ActorRoot 开启 Y-Sort，使主角、宠物与 NPC 按脚底 Y 值正确遮挡。
func _sync_local_actor_y_sort() -> void:
	if _current_level == null or player_node == null:
		return
	if GameState.is_in_battle or _force_battle_pose_active:
		return
	var actor_root: Node2D = _resolve_actor_root(_current_level)
	if not actor_root.y_sort_enabled:
		actor_root.y_sort_enabled = true
	if player_node.get_parent() == actor_root:
		_configure_actor_y_sort(player_node)
	if _pet_follower != null and _pet_follower.get_parent() == actor_root:
		_configure_actor_y_sort(_pet_follower)
	_apply_pet_player_draw_tiebreak(actor_root)


func _get_player_host(level: Node2D) -> Node:
	return _resolve_actor_root(level)

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

func _ensure_click_destination_marker() -> void:
	if _click_destination_marker_root != null and is_instance_valid(_click_destination_marker_root):
		return
	if game_root == null:
		return

	# 落点特效由一个圆环和十字组成，避免新增贴图资源，便于后续统一换肤。
	_click_destination_marker_root = Node2D.new()
	_click_destination_marker_root.name = "ClickDestinationMarker"
	_click_destination_marker_root.visible = false
	_click_destination_marker_root.z_index = 500
	game_root.add_child(_click_destination_marker_root)

	_click_destination_marker_ring = Line2D.new()
	_click_destination_marker_ring.name = "Ring"
	_click_destination_marker_ring.width = CLICK_MARKER_WIDTH
	_click_destination_marker_ring.default_color = Color(0.98, 0.92, 0.52, 0.95)
	_click_destination_marker_ring.closed = true
	_click_destination_marker_ring.antialiased = true
	_click_destination_marker_ring.points = _build_click_marker_ring_points()
	_click_destination_marker_root.add_child(_click_destination_marker_ring)

	_click_destination_marker_cross = Line2D.new()
	_click_destination_marker_cross.name = "Cross"
	_click_destination_marker_cross.width = CLICK_MARKER_WIDTH
	_click_destination_marker_cross.default_color = Color(1.0, 0.97, 0.78, 0.95)
	_click_destination_marker_cross.antialiased = true
	_click_destination_marker_cross.points = PackedVector2Array([
		Vector2(-CLICK_MARKER_CROSS_HALF_SIZE, 0.0),
		Vector2(CLICK_MARKER_CROSS_HALF_SIZE, 0.0),
		Vector2.ZERO,
		Vector2(0.0, -CLICK_MARKER_CROSS_HALF_SIZE),
		Vector2(0.0, CLICK_MARKER_CROSS_HALF_SIZE),
	])
	_click_destination_marker_root.add_child(_click_destination_marker_cross)

func _build_click_marker_ring_points() -> PackedVector2Array:
	var ring_points := PackedVector2Array()
	var segment_count: int = 20
	for index in range(segment_count):
		var angle := TAU * float(index) / float(segment_count)
		ring_points.append(Vector2(cos(angle), sin(angle)) * CLICK_MARKER_RING_RADIUS)
	return ring_points

func _show_click_destination_marker(world_position: Vector2) -> void:
	_ensure_click_destination_marker()
	if _click_destination_marker_root == null:
		return

	if _click_destination_marker_tween != null:
		_click_destination_marker_tween.kill()

	# 每次点击都从更大的尺寸淡入，给移动端玩家一个清晰但不过度抢眼的反馈。
	_click_destination_marker_root.visible = true
	_click_destination_marker_root.position = world_position
	_click_destination_marker_root.scale = Vector2(1.45, 1.45)
	_click_destination_marker_root.modulate = Color(1.0, 1.0, 1.0, 0.0)
	_click_destination_marker_tween = create_tween()
	_click_destination_marker_tween.set_parallel(true)
	_click_destination_marker_tween.tween_property(
		_click_destination_marker_root,
		"scale",
		Vector2.ONE,
		CLICK_MARKER_ANIMATION_DURATION
	).set_trans(Tween.TRANS_CUBIC).set_ease(Tween.EASE_OUT)
	_click_destination_marker_tween.tween_property(
		_click_destination_marker_root,
		"modulate",
		Color(1.0, 1.0, 1.0, 1.0),
		CLICK_MARKER_ANIMATION_DURATION * 0.45
	).set_trans(Tween.TRANS_SINE).set_ease(Tween.EASE_OUT)
	_click_destination_marker_tween.chain().tween_property(
		_click_destination_marker_root,
		"modulate",
		Color(1.0, 1.0, 1.0, 0.0),
		CLICK_MARKER_ANIMATION_DURATION * 0.7
	).set_trans(Tween.TRANS_SINE).set_ease(Tween.EASE_IN)
	_click_destination_marker_tween.finished.connect(_hide_click_destination_marker, CONNECT_ONE_SHOT)

func _hide_click_destination_marker() -> void:
	if _click_destination_marker_root == null:
		return
	_click_destination_marker_root.visible = false

func _rebuild_navigation_grid() -> void:
	_clear_navigation_grid()
	if _current_level == null or player_node == null:
		return

	_navigation_layer = _get_camera_limit_layer(_current_level)
	if _navigation_layer == null:
		return

	_navigation_region = _navigation_layer.get_used_rect()
	if not _navigation_region.has_area():
		return

	_navigation_grid = AStarGrid2D.new()
	_navigation_grid.region = _navigation_region
	_navigation_grid.cell_size = Vector2.ONE
	_navigation_grid.diagonal_mode = AStarGrid2D.DIAGONAL_MODE_NEVER
	_navigation_grid.update()

	for cell_y in range(_navigation_region.position.y, _navigation_region.end.y):
		for cell_x in range(_navigation_region.position.x, _navigation_region.end.x):
			var cell := Vector2i(cell_x, cell_y)
			if not _can_player_stand_on_navigation_cell(cell):
				_navigation_grid.set_point_solid(cell, true)

func _clear_navigation_grid() -> void:
	_navigation_grid = null
	_navigation_layer = null
	_navigation_region = Rect2i()
	_hide_click_destination_marker()
	if player_node != null and player_node.has_method("clear_auto_move_path"):
		player_node.call("clear_auto_move_path")

func _can_player_stand_on_navigation_cell(cell: Vector2i) -> bool:
	if _navigation_layer == null or player_node == null:
		return false

	var collision_shape := player_node.get_node_or_null("CollisionShape2D") as CollisionShape2D
	if collision_shape == null or collision_shape.shape == null:
		return true

	var space_state := player_node.get_world_2d().direct_space_state
	var query := PhysicsShapeQueryParameters2D.new()
	query.shape = collision_shape.shape
	query.collide_with_bodies = true
	query.collide_with_areas = false
	query.collision_mask = player_node.collision_mask
	query.exclude = [player_node.get_rid()]
	query.transform = Transform2D(0.0, _navigation_cell_to_world_position(cell)) * collision_shape.transform

	# 没有碰撞结果时，说明玩家碰撞体可以站在该格中心附近。
	return space_state.intersect_shape(query, 1).is_empty()

func _world_to_navigation_cell(world_position: Vector2) -> Vector2i:
	if _navigation_layer == null:
		return Vector2i.ZERO
	return _navigation_layer.local_to_map(_navigation_layer.to_local(world_position))

func _navigation_cell_to_world_position(cell: Vector2i) -> Vector2:
	if _navigation_layer == null:
		return Vector2.ZERO
	return _navigation_layer.to_global(_navigation_layer.map_to_local(cell))

func _navigation_cell_to_player_local_position(cell: Vector2i) -> Vector2:
	var player_parent := player_node.get_parent()
	if player_parent is Node2D:
		return (player_parent as Node2D).to_local(_navigation_cell_to_world_position(cell))
	return _current_level.to_local(_navigation_cell_to_world_position(cell))

func _resolve_walkable_navigation_cell(cell: Vector2i) -> Vector2i:
	if not _is_navigation_cell_in_bounds(cell):
		return INVALID_NAVIGATION_CELL
	if not _navigation_grid.is_point_solid(cell):
		return cell

	var max_radius: int = maxi(_navigation_region.size.x, _navigation_region.size.y)
	for radius in range(1, max_radius + 1):
		for offset_y in range(-radius, radius + 1):
			for offset_x in range(-radius, radius + 1):
				if abs(offset_x) + abs(offset_y) != radius:
					continue
				var candidate := cell + Vector2i(offset_x, offset_y)
				if not _is_navigation_cell_in_bounds(candidate):
					continue
				if not _navigation_grid.is_point_solid(candidate):
					return candidate

	return INVALID_NAVIGATION_CELL

func _is_navigation_cell_in_bounds(cell: Vector2i) -> bool:
	return (
		cell.x >= _navigation_region.position.x
		and cell.y >= _navigation_region.position.y
		and cell.x < _navigation_region.end.x
		and cell.y < _navigation_region.end.y
	)

func _is_navigation_cell_valid(cell: Vector2i) -> bool:
	return cell != INVALID_NAVIGATION_CELL and _is_navigation_cell_in_bounds(cell)

func _sync_local_player_battle_state() -> void:
	_apply_local_player_battle_pose()

## 由主场景在战斗弹窗显示/隐藏时调用，保证结算演出期间仍保持战斗待机动画。
func set_local_player_battle_pose_active(active: bool) -> void:
	_force_battle_pose_active = active
	_apply_local_player_battle_pose()

func _apply_local_player_battle_pose() -> void:
	if player_node == null or not player_node.has_method("set_battle_active"):
		return
	var battle_pose_active: bool = GameState.is_in_battle or _force_battle_pose_active
	player_node.call("set_battle_active", battle_pose_active)

func _on_battle_state_changed() -> void:
	if GameState.is_in_battle:
		_wild_encounter_request_pending = false
		_set_transition_loading(false)
		set_runtime_input_locked(true)
		_sync_pet_follower_visibility()
		return
	set_runtime_input_locked(false)
	_sync_pet_follower_visibility()
	_reset_pet_follow_near_player()

func _lock_local_player() -> void:
	if player_node != null and player_node.has_method("set_scene_transition_locked"):
		player_node.call("set_scene_transition_locked", true)

func _unlock_local_player() -> void:
	if player_node != null and player_node.has_method("set_scene_transition_locked"):
		player_node.call("set_scene_transition_locked", false)

func set_runtime_input_locked(locked: bool) -> void:
	_runtime_input_locked = locked
	if locked:
		_lock_local_player()
	else:
		_unlock_local_player()

func _set_transition_loading(active: bool) -> void:
	# 地图切换改由主场景黑色遮罩过渡负责，不再显示本地加载层。
	pass

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


func _ensure_pet_follower() -> void:
	if _pet_follower != null:
		return
	_pet_follower = WorldPetFollower.new()
	_pet_follower.name = "PetFollower"
	game_viewport.add_child(_pet_follower)
	_configure_actor_y_sort(_pet_follower)
	_sync_pet_follower_lineup()


func _attach_pet_follower_to_current_level() -> void:
	_ensure_pet_follower()
	if _pet_follower == null or _current_level == null or player_node == null:
		return
	var target_parent := _get_player_host(_current_level)
	if _pet_follower.get_parent() == target_parent:
		_configure_actor_y_sort(_pet_follower)
		return
	_pet_follower.reparent(target_parent, true)
	_configure_actor_y_sort(_pet_follower)


func _detach_pet_follower_from_current_level() -> void:
	if _pet_follower == null:
		return
	if _pet_follower.get_parent() == game_viewport:
		return
	_pet_follower.reparent(game_viewport, true)


func _sync_pet_follower_lineup() -> void:
	_ensure_pet_follower()
	if _pet_follower == null:
		return
	if GameState.lineup.is_empty():
		_pet_follower.clear_binding()
		return
	var first_lineup_variant: Variant = GameState.lineup[0]
	if first_lineup_variant is Dictionary:
		_pet_follower.sync_lineup_pet(first_lineup_variant as Dictionary)
	else:
		_pet_follower.clear_binding()
	_sync_pet_follower_visibility()


func _sync_pet_follower_visibility() -> void:
	if _pet_follower == null:
		return
	if GameState.is_in_battle or _force_battle_pose_active:
		_pet_follower.clear_binding()
		return
	if GameState.lineup.is_empty():
		_pet_follower.clear_binding()
		return
	var first_lineup_variant: Variant = GameState.lineup[0]
	if first_lineup_variant is Dictionary:
		_pet_follower.sync_lineup_pet(first_lineup_variant as Dictionary)


func _reset_pet_follow_near_player() -> void:
	if _pet_follower == null or player_node == null:
		return
	_leader_path_anchor = Vector2.INF
	var follow_offset: Vector2 = _resolve_pet_follow_reset_offset()
	_pet_follower.reset_near_leader(player_node.position, follow_offset)


## 根据主角朝向计算切图/同步后的宠物站位：落在身后约半格，避免初始就隔一整格。
func _resolve_pet_follow_reset_offset() -> Vector2:
	var half_step: float = PathFollowController.PATH_STEP_SIZE * 0.5
	if player_node == null or not player_node.has_method("get_cardinal_direction"):
		return Vector2(0.0, -half_step)
	var facing: Vector2 = player_node.call("get_cardinal_direction") as Vector2
	if facing == Vector2.ZERO or facing == Vector2.DOWN:
		return Vector2(0.0, -half_step)
	if facing == Vector2.UP:
		return Vector2(0.0, half_step)
	if facing == Vector2.LEFT:
		return Vector2(half_step, 0.0)
	if facing == Vector2.RIGHT:
		return Vector2(-half_step, 0.0)
	return Vector2(0.0, -half_step)


func _update_pet_follow(delta: float) -> void:
	if _pet_follower == null or player_node == null:
		return
	if GameState.is_in_battle or _force_battle_pose_active:
		return
	if not _pet_follower.visible:
		return
	_record_leader_path_for_pet_follow()
	var move_speed: float = player_node.get_move_speed() if player_node.has_method("get_move_speed") else 100.0
	_pet_follower.update_follow(delta, move_speed)


func _record_leader_path_for_pet_follow() -> void:
	if _pet_follower == null or player_node == null:
		return
	if not player_node.has_method("is_walking") or not player_node.call("is_walking"):
		return
	var leader_direction: Vector2 = Vector2.ZERO
	if player_node.has_method("get_cardinal_direction"):
		leader_direction = player_node.call("get_cardinal_direction") as Vector2
	if leader_direction == Vector2.ZERO:
		return

	var leader_position: Vector2 = player_node.position
	if _leader_path_anchor == Vector2.INF:
		_leader_path_anchor = leader_position
		return

	var anchor_to_leader: Vector2 = leader_position - _leader_path_anchor
	if anchor_to_leader.dot(leader_direction) <= 0.0:
		# 主角转向、碰撞滑动或服务端校正后，当前朝向可能不再指向旧锚点到主角的位置。
		# 继续沿错误方向推进锚点会让 distance 永远不变甚至变大，导致单帧死循环。
		_leader_path_anchor = leader_position
		return

	var recorded_steps: int = 0
	while leader_position.distance_to(_leader_path_anchor) >= PathFollowController.PATH_STEP_SIZE:
		if recorded_steps >= PET_FOLLOW_MAX_LEADER_STEPS_PER_FRAME:
			_leader_path_anchor = leader_position
			return
		var previous_distance: float = leader_position.distance_to(_leader_path_anchor)
		_pet_follower.push_leader_step(_leader_path_anchor, leader_direction)
		_leader_path_anchor += leader_direction * PathFollowController.PATH_STEP_SIZE
		recorded_steps += 1
		if leader_position.distance_to(_leader_path_anchor) >= previous_distance:
			_leader_path_anchor = leader_position
			return
