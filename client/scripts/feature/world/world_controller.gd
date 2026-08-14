extends Control

signal scene_loaded(scene_id: String)
signal player_position_changed(local_position: Vector2, global_position: Vector2)
signal scene_transition_requested(from_scene_id: int, to_scene_id: int)
signal scene_transition_failed(reason: String)
signal npc_interaction_requested(entity_id: int, npc_name: String)
signal wild_encounter_responded(accepted: bool, reason: String)

const DEFAULT_RENDER_FRAME_SIZE: Vector2 = Vector2(780.0, 1440.0)
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
## 远端玩家复用现有角色场景，确保人物形象、动画和地图层级口径一致。
const REMOTE_PLAYER_SCENE: PackedScene = preload("res://scenes/world/player.tscn")
## 服务端场景实体中代表玩家的类型值。
const PLAYER_ENTITY_TYPE: int = 1
## 实时表现坐标使用千分之一场景格定点整数，与服务端协议保持一致。
const NETWORK_POSITION_FIXED_SCALE: int = 1000
## 移动中的高精度表现坐标最多每 100 毫秒发送一次，兼顾移动端流量与视觉同步精度。
const NETWORK_MOVEMENT_REPORT_INTERVAL_MS: int = 100
@export_group("点击移动反馈")
## 点击地面时播放的精灵帧动画；在检查器拖入 SpriteFrames 后优先于 Line2D 圆环。
@export var click_marker_sprite_frames: SpriteFrames = null
## 精灵帧动画名，需存在于 click_marker_sprite_frames 中。
@export var click_marker_animation: String = "default"
## 落点特效整体缩放。
@export var click_marker_scale: Vector2 = Vector2(1.0, 1.0)

@onready var game_shell: Control = %GameShell
@onready var game_viewport_container: SubViewportContainer = %GameViewportContainer
@onready var game_viewport: SubViewport = %GameViewport
@onready var background_fill: Sprite2D = %BackgroundFill
@onready var game_root: Node2D = %GameRoot
@onready var player_node: CharacterBody2D = %Player
## 人物场景预置的传送特效；位置和缩放直接在 player.tscn 中调整。
@onready var map_teleport_effect: SpaceTimeTeleportEffect = player_node.get_node_or_null("MapTeleportEffect") as SpaceTimeTeleportEffect
@onready var map_loading_overlay: ColorRect = %MapLoadingOverlay

var _next_op_id: int = 1
var _next_move_seq: int = 1
var _pending_target_scene_id: int = 0
var _pending_portal_id: int = 0
## 尚未收到服务端确认的普通移动序号；限制同时只有一个移动请求在途，避免慢数据库链路堆积并阻塞切图请求。
var _pending_position_move_seq: int = 0
## 当前切图请求使用的移动序号；只有相同序号的回包才能改变转场状态。
var _pending_transition_move_seq: int = 0
## 当前切图是否来自世界地图快速传送，用于在目标地图加载后读取本地统一出生点。
var _pending_map_teleport: bool = false
## 是否正在应用正式登录后的首次地图；该流程使用地图脚本本地出生点，断线重连不设置此标记。
var _pending_login_spawn_requested: bool = false
## 地图传送特效是否已经到达人物消失点并启动主场景黑屏，避免信号重复进入转场。
var _map_teleport_transition_started: bool = false
## 人物是否由地图传送演出主动隐藏；只恢复本流程隐藏的节点，避免覆盖其他系统可见性。
var _map_teleport_player_hidden: bool = false
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
var _last_reported_player_position: Vector2 = Vector2.INF
var _current_level: Node2D
## 当前地图左上角在地图根节点本地坐标中的位置，用于把服务端场景坐标映射到地图实际像素。
var _current_level_scene_origin_pixels: Vector2 = Vector2.ZERO
var _current_interactable_entity_id: int = 0
var _current_interactable_npc_name: String = ""
var _current_interactable_requested: bool = false
var _runtime_input_locked: bool = false
var _navigation_grid: AStarGrid2D
var _navigation_layer: TileMapLayer
var _navigation_region: Rect2i = Rect2i()
var _click_destination_marker_root: Node2D
var _click_destination_marker_sprite: AnimatedSprite2D
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
## 主角上一次记录宠物路径时的方向；转向时重置锚点，避免残余距离被记到错误轴上。
var _leader_path_direction: Vector2 = Vector2.ZERO
## 当前地图已创建的远端玩家节点，键为服务端 entity_id。
var _remote_player_nodes: Dictionary = {}
## 当前地图已创建的远端玩家宠物跟随节点，键与远端玩家 entity_id 一致。
var _remote_pet_follower_nodes: Dictionary = {}
## 远端玩家最近一次权威目标坐标，用于把其移动路径同步给宠物跟随节点。
var _remote_player_target_positions: Dictionary = {}
## 每个远端宠物对应的玩家路径记录锚点，键为远端玩家 entity_id。
var _remote_pet_path_anchors: Dictionary = {}
## 每个远端宠物上一次记录路径时的玩家朝向，键为远端玩家 entity_id。
var _remote_pet_path_directions: Dictionary = {}
## 最近一次发给服务端的整数场景坐标，用于限制移动同步为每格一次。
var _last_network_position: Vector2i = Vector2i.ZERO
## 标记最近网络坐标是否有效，切图后会重新上报新地图落点。
var _has_last_network_position: bool = false
## 最近一次发送的千分之一格高精度位置。
var _last_network_precise_position: Vector2i = Vector2i.ZERO
## 最近一次发送的四方向朝向。
var _last_network_facing: Vector2i = Vector2i.DOWN
## 最近一次发送时玩家是否处于移动状态。
var _last_network_moving: bool = false
## 最近一次移动表现上报的本地单调时钟毫秒数。
var _last_network_report_msec: int = 0
func _process(delta: float) -> void:
    _update_pet_follow(delta)
    _update_remote_pet_followers(delta)
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
    if map_teleport_effect != null and not map_teleport_effect.vanish_started.is_connected(_on_map_teleport_vanish_started):
        map_teleport_effect.vanish_started.connect(_on_map_teleport_vanish_started)
    GameState.battle_changed.connect(_sync_local_player_battle_state)
    GameState.battle_changed.connect(_on_battle_state_changed)
    GameState.pets_changed.connect(_sync_pet_follower_lineup)
    _ensure_pet_follower()
    _ensure_click_destination_marker()
    call_deferred("_refresh_game_layout")
    _sync_local_player_battle_state()

## 登录页已在主场景挂载前完成 ENTER_WORLD；主场景就绪后加载地图，并直接读取地图脚本的本地统一出生点。
func apply_prepared_world_entry() -> void:
    _pending_login_spawn_requested = true
    _pending_player_facing_requested = true
    _pending_player_facing_direction = Vector2.DOWN
    _apply_authoritative_snapshot()
    _emit_scene_loaded_if_changed(true)

func handle_enter_world(payload: Dictionary) -> void:
    var already_in_world: bool = int(GameState.scene_snapshot.get("scene_id", 0)) > 0
    var preserved_scene_position: Vector2 = Vector2.INF
    if already_in_world:
        preserved_scene_position = _current_player_scene_position()
    if not already_in_world:
        _pending_login_spawn_requested = true
        _pending_player_facing_requested = true
        _pending_player_facing_direction = Vector2.DOWN
    GameState.set_world_snapshot(payload)
    if already_in_world:
        # 人物面板等场景只会刷新属性，不应重置当前地图站位。
        if preserved_scene_position != Vector2.INF:
            GameState.sync_player_scene_position(preserved_scene_position)
        return
    _apply_authoritative_snapshot()
    _emit_scene_loaded_if_changed(true)

func handle_entity_enter(payload: Dictionary) -> void:
    if payload.has("scene_id") and int(payload.get("scene_id", 0)) != _current_scene_id():
        return
    var entity_variant: Variant = payload.get("entity", payload)
    var entity: Dictionary = entity_variant if entity_variant is Dictionary else {}
    GameState.add_entity(entity)
    _sync_remote_players()

func handle_entity_leave(payload: Dictionary) -> void:
    if payload.has("scene_id") and int(payload.get("scene_id", 0)) != _current_scene_id():
        return
    GameState.remove_entity(int(payload.get("entity_id", 0)))
    _sync_remote_players()

func handle_entity_move(payload: Dictionary) -> void:
    if payload.has("scene_id") and int(payload.get("scene_id", 0)) != _current_scene_id():
        return
    GameState.apply_entity_move(payload)
    _sync_remote_players()

func handle_move_intent_response(payload: Dictionary) -> void:
    var accepted: bool = bool(payload.get("accepted", false))
    var scene_id: int = int(payload.get("scene_id", _current_scene_id()))
    var response_move_seq: int = int(payload.get("move_seq", 0))
    # 普通移动严格保持单请求在途；确认到达后，下一帧会把角色最新状态而不是中间历史位置补发给服务端。
    if response_move_seq == _pending_position_move_seq:
        _pending_position_move_seq = 0
    # 普通移动回包不参与切图状态机；特别是延迟回包不能清除随后创建的切图 pending 状态。
    if _pending_target_scene_id == 0:
        return
    if response_move_seq != _pending_transition_move_seq:
        _debug_scene_transition(
            "MOVE_INTENT_RESP ignored stale move_seq=%d pending_move_seq=%d response_scene=%d reason=%s" % [
                response_move_seq,
                _pending_transition_move_seq,
                scene_id,
                str(payload.get("reason", "")),
            ]
        )
        return

    _debug_scene_transition(
        "MOVE_INTENT_RESP accepted=%s move_seq=%d current_scene=%d response_scene=%d pending_target=%d portal=%d reason=%s" % [
            str(accepted),
            response_move_seq,
            _current_scene_id(),
            scene_id,
            _pending_target_scene_id,
            _pending_portal_id,
            str(payload.get("reason", "")),
        ]
    )
    if accepted:
        # 服务端接受后继续等待 WORLD_RESYNC_PUSH；地图加载成功后，客户端再按转场类型读取当前场景脚本的本地出生点。
        return

    _pending_target_scene_id = 0
    _pending_portal_id = 0
    _pending_position_move_seq = 0
    _pending_transition_move_seq = 0
    _pending_map_teleport = false
    _pending_player_facing_requested = false
    _set_transition_loading(false)
    _finish_map_teleport_visual()
    cancel_scene_visual_apply_lock()
    _unlock_local_player()
    scene_transition_failed.emit(str(payload.get("reason", "scene transfer rejected")))

func handle_world_resync(payload: Dictionary) -> void:
    # 权威快照已经覆盖此前的普通移动请求；解除背压，以便新场景从最新位置重新开始同步。
    _pending_position_move_seq = 0
    var force_scene_loaded: bool = _pending_map_teleport
    _debug_scene_transition(
        "WORLD_RESYNC_PUSH scene=%d current_scene=%d pending_target=%d visual_locked=%s" % [
            int(payload.get("scene_id", 0)),
            _current_scene_id(),
            _pending_target_scene_id,
            str(_scene_visual_apply_locked),
        ]
    )
    var normalized_payload: Dictionary = payload.duplicate(true)
    var self_precise_position_variant: Variant = payload.get("self_precise_pos", null)
    if self_precise_position_variant is Dictionary and not (self_precise_position_variant as Dictionary).is_empty():
        # 重连优先把 Redis千分之一格位置转换为场景坐标，再交给统一世界快照入口。
        var self_precise_position: Dictionary = self_precise_position_variant as Dictionary
        normalized_payload["self_pos"] = {
            "x": float(self_precise_position.get("x", 0)) / float(NETWORK_POSITION_FIXED_SCALE),
            "y": float(self_precise_position.get("y", 0)) / float(NETWORK_POSITION_FIXED_SCALE),
        }
    GameState.set_world_snapshot(normalized_payload)
    if _scene_visual_apply_locked:
        _deferred_scene_apply_pending = true
        _debug_scene_transition("WORLD_RESYNC_PUSH deferred until transition midpoint")
        return
    _apply_authoritative_snapshot()
    _emit_scene_loaded_if_changed(force_scene_loaded)

func handle_wild_encounter_response(payload: Dictionary) -> void:
    var accepted: bool = bool(payload.get("accepted", false))
    var reason: String = str(payload.get("reason", ""))
    _wild_encounter_request_pending = false
    _set_transition_loading(false)
    if not accepted:
        set_runtime_input_locked(false)
        _unlock_local_player()
    wild_encounter_responded.emit(accepted, reason)

## 请求世界地图快速传送；请求只提交目标场景，成功加载后直接读取目标地图脚本的本地统一出生点。
## target_scene_id 是当前地图标点配置的目标场景 ID。
func request_map_teleport(target_scene_id: int) -> void:
    request_scene_transition(target_scene_id, 0, Vector2.ZERO, true)


## 请求普通传送门切图或世界地图快速传送；请求本身不携带任何切图落点。
## target_scene_id 是目标场景 ID；portal_id 是普通传送门 ID；facing_direction 是切图后的角色朝向；map_teleport 表示加载后是否读取地图脚本本地统一出生点。
func request_scene_transition(target_scene_id: int, portal_id: int = 0, facing_direction: Vector2 = Vector2.ZERO, map_teleport: bool = false) -> void:
    var current_scene_id: int = _current_scene_id()
    if current_scene_id <= 0:
        _unlock_local_player()
        scene_transition_failed.emit("scene not initialized")
        return
    if target_scene_id <= 0 or (target_scene_id == current_scene_id and not map_teleport):
        _unlock_local_player()
        return
    if _pending_target_scene_id != 0:
        return

    _pending_target_scene_id = target_scene_id
    _pending_portal_id = portal_id
    var transition_move_seq: int = _take_next_move_seq()
    _pending_transition_move_seq = transition_move_seq
    _pending_map_teleport = map_teleport
    _pending_player_facing_requested = facing_direction != Vector2.ZERO
    if _pending_player_facing_requested:
        _pending_player_facing_direction = facing_direction
    _lock_local_player()
    _set_transition_loading(true)
    _debug_scene_transition(
        "request from_scene=%d target_scene=%d portal=%d map_teleport=%s player_pos=%s" % [
            current_scene_id,
            target_scene_id,
            portal_id,
            str(map_teleport),
            str(_current_player_scene_position()),
        ]
    )
    if map_teleport:
        # 在发出请求前锁住视觉应用；服务端较快返回时，权威快照会等到人物消失后的黑屏中点再应用。
        set_scene_visual_apply_locked(true)
        _play_map_teleport_visual()
    else:
        scene_transition_requested.emit(current_scene_id, target_scene_id)
    # 普通门请求只提交门编号和目标场景；目标地图的出生格不进入网络载荷。
    var request_payload: Dictionary = {
        "op_id": _take_next_op_id(),
        "move_seq": transition_move_seq,
        "scene_id": current_scene_id,
        "target_scene_id": target_scene_id,
        "portal_id": portal_id,
        "map_teleport": map_teleport,
    }
    var request_seq: int = NetClient.send_command(CommandIds.MOVE_INTENT_REQ, request_payload)
    _debug_scene_transition("MOVE_INTENT_REQ sent seq=%d move_seq=%d" % [request_seq, transition_move_seq])


## 播放人物场景内预置的传送特效；节点缺失时直接回退到原有黑屏转场。
func _play_map_teleport_visual() -> void:
    _map_teleport_transition_started = false
    _map_teleport_player_hidden = false
    if not is_instance_valid(map_teleport_effect) or not is_instance_valid(player_node):
        _start_pending_map_teleport_transition()
        return
    map_teleport_effect.play_effect()


## 聚能到人物消失点后隐藏本地人物，并通知主场景开始原有黑屏换图。
func _on_map_teleport_vanish_started() -> void:
    if not _pending_map_teleport or _pending_target_scene_id <= 0:
        return
    if player_node != null and player_node.visible:
        player_node.visible = false
        _map_teleport_player_hidden = true
    _start_pending_map_teleport_transition()


## 只启动一次地图传送黑屏；目标场景仍使用点击时记录的服务端权威请求参数。
func _start_pending_map_teleport_transition() -> void:
    if _map_teleport_transition_started or not _pending_map_teleport:
        return
    _map_teleport_transition_started = true
    scene_transition_requested.emit(_current_scene_id(), _pending_target_scene_id)


## 停止地图传送特效并恢复由本流程隐藏的人物，供成功、拒绝和超时路径统一调用。
func _finish_map_teleport_visual() -> void:
    if is_instance_valid(map_teleport_effect):
        map_teleport_effect.stop_effect()
    if _map_teleport_player_hidden and is_instance_valid(player_node):
        player_node.visible = true
    _map_teleport_player_hidden = false
    _map_teleport_transition_started = false

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


## 主动中止尚未完成的地图切换，并恢复玩家输入。
## reason 是用于定向日志定位超时或上层取消原因的简短说明。
func abort_scene_transition(reason: String) -> void:
    _debug_scene_transition(
        "transition aborted pending_target=%d portal=%d reason=%s" % [
            _pending_target_scene_id,
            _pending_portal_id,
            reason,
        ]
    )
    _pending_target_scene_id = 0
    _pending_portal_id = 0
    _pending_position_move_seq = 0
    _pending_transition_move_seq = 0
    _pending_map_teleport = false
    _pending_player_facing_requested = false
    _pending_player_spawn_requested = false
    _set_transition_loading(false)
    _finish_map_teleport_visual()
    cancel_scene_visual_apply_lock()
    _unlock_local_player()


func _flush_deferred_scene_apply() -> void:
    if not _deferred_scene_apply_pending:
        _debug_scene_transition("transition midpoint has no deferred world snapshot yet")
        return
    _deferred_scene_apply_pending = false
    _debug_scene_transition("applying deferred world snapshot at transition midpoint")
    var force_scene_loaded: bool = _pending_map_teleport
    _apply_authoritative_snapshot()
    _emit_scene_loaded_if_changed(force_scene_loaded)

func set_render_frame_size(frame_size: Vector2) -> void:
    if frame_size.x <= 0.0 or frame_size.y <= 0.0:
        return
    var normalized_size: Vector2 = frame_size.floor()
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
        _debug_scene_transition("load_level rejected: empty scene path")
        push_warning("Level scene path is empty.")
        return

    var load_started_msec: int = Time.get_ticks_msec()
    _debug_scene_transition("load_level begin path=%s" % scene_path)
    var level_scene: PackedScene = load(scene_path) as PackedScene
    if level_scene == null:
        _debug_scene_transition("load_level failed path=%s" % scene_path)
        push_warning("Failed to load level scene: %s" % scene_path)
        return

    mount_level(level_scene)
    _debug_scene_transition(
        "load_level end path=%s elapsed_ms=%d mounted=%s" % [
            scene_path,
            Time.get_ticks_msec() - load_started_msec,
            str(is_instance_valid(_current_level)),
        ]
    )

func mount_level(level_scene: PackedScene) -> void:
    unmount_current_level()

    var level_instance := level_scene.instantiate()
    if level_instance is not Node2D:
        push_warning("Mounted level must inherit Node2D.")
        level_instance.queue_free()
        return

    _current_level = level_instance as Node2D
    game_root.add_child(_current_level)
    _normalize_current_level_origin()
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
    _sync_remote_players()

func unmount_current_level() -> void:
    if _current_level == null:
        return

    _clear_current_interactable_npc()
    _clear_remote_players()
    _clear_navigation_grid()
    _detach_player_from_current_level()
    _detach_pet_follower_from_current_level()
    var level_to_free := _current_level
    _current_level = null
    _current_level_scene_origin_pixels = Vector2.ZERO
    if level_to_free.get_parent() == game_root:
        game_root.remove_child(level_to_free)
    level_to_free.queue_free()

func _on_viewport_size_changed() -> void:
    _sync_render_frame_size_from_shell()
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

    var cell_path: Array[Vector2i] = _build_low_turn_navigation_path(start_cell, target_cell)
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
    _sync_render_frame_size_from_shell()
    _resize_game_viewport()
    _layout_game_viewport_container()
    _refresh_background_fill()
    if map_loading_overlay != null:
        map_loading_overlay.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)

## Web 临时调试页容易把容器尺寸改成浏览器当前可视区域，因此这里保留一套固定设计尺寸口径，
## 让内部世界渲染始终按项目设计分辨率执行，而不是被 tmp_js_export.html 当前尺寸带偏。
func _should_force_fixed_render_frame_size() -> bool:
    return OS.has_feature("web")

## 从 GameShell 同步当前可用渲染区域；Web 调试时强制保持设计尺寸，避免内部视口随壳层漂移。
func _sync_render_frame_size_from_shell() -> void:
    if _should_force_fixed_render_frame_size():
        _render_frame_size = DEFAULT_RENDER_FRAME_SIZE
        return
    if game_shell == null:
        return
    var shell_size: Vector2 = game_shell.size.floor()
    if shell_size.x <= 0.0 or shell_size.y <= 0.0:
        return
    if shell_size == _render_frame_size.floor():
        return
    _render_frame_size = shell_size

## 解析世界 SubViewport 内部渲染尺寸；Web 调试时固定返回设计分辨率，其他平台沿用壳层尺寸。
func _resolve_internal_render_frame_size() -> Vector2i:
    if _should_force_fixed_render_frame_size():
        return Vector2i(int(DEFAULT_RENDER_FRAME_SIZE.x), int(DEFAULT_RENDER_FRAME_SIZE.y))
    var frame_size: Vector2 = _render_frame_size.floor()
    if frame_size.x <= 0.0 or frame_size.y <= 0.0:
        frame_size = DEFAULT_RENDER_FRAME_SIZE
    return Vector2i(maxi(1, int(frame_size.x)), maxi(1, int(frame_size.y)))

func _resize_game_viewport() -> void:
    if game_viewport == null:
        return
    var target_size: Vector2i = _resolve_internal_render_frame_size()
    # 世界先按当前视口尺寸渲染，再交给外层做整数倍放大，保证像素边缘稳定。
    if game_viewport.size == target_size:
        return
    game_viewport.size = target_size

## 根据当前可用区域，把 SubViewportContainer 以整数倍居中显示，避免 1.3x / 1.87x 这类非整数缩放带来的发糊。
func _layout_game_viewport_container() -> void:
    if game_viewport_container == null:
        return

    var internal_frame_size: Vector2i = _resolve_internal_render_frame_size()
    var available_size: Vector2 = game_shell.size if game_shell != null else Vector2.ZERO
    if available_size.x <= 0.0 or available_size.y <= 0.0:
        available_size = Vector2(float(internal_frame_size.x), float(internal_frame_size.y))

    var scale_x: float = available_size.x / float(internal_frame_size.x)
    var scale_y: float = available_size.y / float(internal_frame_size.y)
    var integer_scale: int = maxi(1, int(floor(minf(scale_x, scale_y))))
    var target_size: Vector2 = Vector2(
        float(internal_frame_size.x * integer_scale),
        float(internal_frame_size.y * integer_scale)
    )
    var target_position: Vector2 = ((available_size - target_size) * 0.5).floor()

    game_viewport_container.position = target_position
    game_viewport_container.size = target_size

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
        _pending_position_move_seq = 0
        _pending_transition_move_seq = 0
        _pending_map_teleport = false
        _pending_login_spawn_requested = false
        _pending_player_facing_requested = false
        _set_transition_loading(false)
        _finish_map_teleport_visual()
        _unlock_local_player()
        scene_transition_failed.emit("failed to load scene map: %d" % scene_id)
        return

    # 登录、普通传送门与世界地图快速传送都使用服务端快照中的权威落点。
    # 客户端地图资源只负责渲染，不能再次覆盖已持久化并广播给其他玩家的位置。
    var self_pos: Vector2 = _extract_self_position(GameState.player_snapshot)
    _stage_pending_player_transition({"spawn_position": _server_to_local_position(scene_id, self_pos)})
    _apply_pending_player_transition()
    _attach_pet_follower_to_current_level()
    _reset_pet_follow_near_player()
    _sync_pet_follower_lineup()
    _apply_level_camera_limits()
    _sync_local_actor_y_sort()
    _sync_remote_players()
    _refresh_background_fill()

    _pending_target_scene_id = 0
    _pending_portal_id = 0
    _pending_position_move_seq = 0
    _pending_transition_move_seq = 0
    _pending_map_teleport = false
    _pending_login_spawn_requested = false
    _pending_player_facing_requested = false
    _portal_cooldown_until_ms = Time.get_ticks_msec() + PORTAL_ACTIVATION_COOLDOWN_MS
    _set_transition_loading(false)
    _finish_map_teleport_visual()
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


## 判断当前世界状态是否允许客户端主动发起一轮暗雷挂机请求。
## 返回值为 true 表示当前场景、输入锁和战斗状态都满足条件。
func can_request_auto_wild_encounter_battle() -> bool:
    if _wild_encounter_request_pending or GameState.is_in_battle:
        return false
    if _pending_target_scene_id != 0 or _runtime_input_locked:
        return false
    if not bool(GameState.wild_encounter_config.get("enabled", false)):
        return false
    var config_scene_id: int = int(GameState.wild_encounter_config.get("scene_id", 0))
    var current_scene_id: int = _current_scene_id()
    if config_scene_id <= 0 or current_scene_id <= 0:
        return false
    return config_scene_id == current_scene_id


## 供主场景挂机逻辑主动请求一轮暗雷战斗；成功发起后返回 true。
## 这里复用现有暗雷请求链路，确保仍由服务端权威决定是否真正进入战斗。
func request_auto_wild_encounter_battle() -> bool:
    if not can_request_auto_wild_encounter_battle():
        return false
    _request_wild_encounter_battle(_current_scene_id())
    return true

func _request_wild_encounter_battle(scene_id: int) -> void:
    _wild_encounter_request_pending = true
    _lock_local_player()
    set_runtime_input_locked(true)
    if player_node != null and player_node.has_method("clear_auto_move_path"):
        player_node.call("clear_auto_move_path")
    _set_transition_loading(true)
    var scene_position: Vector2 = _current_player_scene_position()
    GameState.sync_player_scene_position(scene_position)
    App.request_wild_encounter(scene_id, _take_next_move_seq())

func _ensure_scene_loaded(scene_id: int) -> bool:
    if scene_id <= 0:
        _debug_scene_transition("ensure_scene_loaded rejected invalid scene=%d" % scene_id)
        return false
    if _loaded_scene_id == scene_id and is_instance_valid(_current_level):
        _debug_scene_transition("ensure_scene_loaded reused scene=%d" % scene_id)
        _attach_player_to_current_level()
        return true

    var scene_config := _scene_config(scene_id)
    var scene_path := str(scene_config.get("scene_path", ""))
    if scene_path.is_empty():
        _debug_scene_transition("ensure_scene_loaded missing registry path scene=%d" % scene_id)
        return false

    load_level(scene_path)
    if not is_instance_valid(_current_level):
        _debug_scene_transition("ensure_scene_loaded mount failed scene=%d path=%s" % [scene_id, scene_path])
        return false
    _loaded_scene_id = scene_id
    _refresh_game_layout()
    return true

func _emit_scene_loaded_if_changed(force_emit: bool) -> void:
    var scene_id: int = _current_scene_id()
    if force_emit or scene_id != _last_loaded_scene_id:
        _last_loaded_scene_id = scene_id
        _debug_scene_transition("scene_loaded emitted scene=%d force=%s" % [scene_id, str(force_emit)])
        scene_loaded.emit(str(scene_id))


## 输出地图切换专用调试日志；其他客户端业务日志继续保持关闭。
## message 是当前切图阶段及其关键权威参数。
func _debug_scene_transition(message: String) -> void:
    print("[SceneTransition][Client][World] %s" % message)

## 返回当前已加载场景导出的展示名称；没有配置时回退为 scene_id，避免 HUD 空白。
func get_current_scene_display_name() -> String:
    if _current_level != null and _current_level.has_method("get_scene_display_name"):
        var scene_name_value: Variant = _current_level.call("get_scene_display_name")
        var scene_name: String = str(scene_name_value).strip_edges()
        if not scene_name.is_empty():
            return scene_name
    return str(_current_scene_id())

func _current_scene_id() -> int:
    return int(GameState.scene_snapshot.get("scene_id", 0))

func _extract_self_position(player_snapshot: Dictionary) -> Vector2:
    return Vector2(float(player_snapshot.get("x", 0.0)), float(player_snapshot.get("y", 0.0)))

func _scene_config(scene_id: int) -> Dictionary:
    return WorldSceneRegistry.get_scene_config(scene_id)

## 把服务端权威场景坐标转换成 Godot 渲染像素坐标。
## 参数 scene_id 表示当前场景 ID；server_position 是以地图左上角为 (0,0) 的场景坐标；返回值是玩家父节点内的像素坐标。
func _server_to_local_position(scene_id: int, server_position: Vector2) -> Vector2:
    return _scene_coordinate_to_local_pixels(scene_id, server_position)


## 读取当前场景的单格像素大小，统一服务端格子坐标与客户端渲染坐标的换算倍率。
## 参数 scene_id 表示当前场景 ID；返回值是每 1 个服务端坐标单位对应的像素长度。
func _grid_to_pixels_for_scene(scene_id: int) -> float:
    var scene_config: Dictionary = _scene_config(scene_id)
    var grid_to_pixels: float = float(scene_config.get("grid_to_pixels", DEFAULT_GRID_TO_PIXELS))
    return maxf(grid_to_pixels, 1.0)

## 把统一场景坐标换算为渲染像素坐标；地图左上角 (0,0) 对应像素原点。
## 参数 scene_id 表示当前场景 ID；scene_position 是服务端/场景统一坐标；返回值是 Godot 像素坐标。
func _scene_coordinate_to_local_pixels(scene_id: int, scene_position: Vector2) -> Vector2:
    var grid_to_pixels: float = _grid_to_pixels_for_scene(scene_id)
    # 地图和角色统一吸附到整数像素，避免服务端浮点坐标换算后出现半像素渲染。
    return (_current_level_scene_origin_pixels + scene_position * grid_to_pixels).round()

## 把玩家当前像素坐标换算回统一场景坐标，供 HUD 和调试日志展示。
## 参数 scene_id 表示当前场景 ID；local_pixels 是地图内像素坐标；返回值与服务端 self_pos 使用同一坐标系。
func _local_pixels_to_scene_coordinate(scene_id: int, local_pixels: Vector2) -> Vector2:
    var grid_to_pixels: float = _grid_to_pixels_for_scene(scene_id)
    return (local_pixels - _current_level_scene_origin_pixels) / grid_to_pixels


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
            player_node.position = _pending_player_spawn_position.round()
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


## 返回当前地图中指定 NPC 的当前待机帧，供 NPC 菜单标题展示本地上半身形象。
## entity_id 是服务端权威 NPC 实体 ID；未找到对应场景节点或纹理时返回空值。
func get_npc_portrait_texture(entity_id: int) -> Texture2D:
    if entity_id <= 0 or _current_level == null or not is_instance_valid(_current_level):
        return null
    for child: Node in _current_level.find_children("*", "InteractiveNPCBase", true, false):
        var npc: InteractiveNPCBase = child as InteractiveNPCBase
        if npc == null or npc.entity_id != entity_id:
            continue
        var sprite: AnimatedSprite2D = npc.get_node_or_null("AnimatedSprite2D") as AnimatedSprite2D
        if sprite == null or sprite.sprite_frames == null:
            return null
        var animation_name: String = str(sprite.animation)
        if animation_name.is_empty() or not sprite.sprite_frames.has_animation(animation_name):
            var animation_names: PackedStringArray = sprite.sprite_frames.get_animation_names()
            if animation_names.is_empty():
                return null
            animation_name = str(animation_names[0])
        if sprite.sprite_frames.get_frame_count(animation_name) <= 0:
            return null
        var frame_index: int = clampi(sprite.frame, 0, sprite.sprite_frames.get_frame_count(animation_name) - 1)
        return sprite.sprite_frames.get_frame_texture(animation_name, frame_index)
    return null


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
    for layer_name in ["Collision", "Bottom", "Map", "TileMapLayer", "地图"]:
        var layer := level.get_node_or_null(layer_name) as TileMapLayer
        if layer != null:
            return layer
    return null

## 将加载后的地图整体平移，使可用地图矩形的左上角落在世界原点 (0,0)。
## 该方法只移动当前地图根节点，不改写 .tscn 资源，避免批量重写地图文件。
func _normalize_current_level_origin() -> void:
    _current_level_scene_origin_pixels = Vector2.ZERO
    if _current_level == null:
        return
    var level_rect: Rect2 = _resolve_level_world_rect(_current_level)
    if not level_rect.has_area():
        return
    if not level_rect.position.is_equal_approx(Vector2.ZERO):
        _current_level.global_position -= level_rect.position
    _current_level_scene_origin_pixels = _current_level.to_local(Vector2.ZERO)

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

    _click_destination_marker_root = Node2D.new()
    _click_destination_marker_root.name = "ClickDestinationMarker"
    _click_destination_marker_root.visible = false
    _click_destination_marker_root.z_index = 500
    game_root.add_child(_click_destination_marker_root)

    if _uses_click_marker_sprite():
        _click_destination_marker_sprite = AnimatedSprite2D.new()
        _click_destination_marker_sprite.name = "Sprite"
        _click_destination_marker_sprite.centered = true
        _click_destination_marker_sprite.sprite_frames = click_marker_sprite_frames
        _click_destination_marker_sprite.visible = false
        if not _click_destination_marker_sprite.animation_finished.is_connected(_on_click_marker_animation_finished):
            _click_destination_marker_sprite.animation_finished.connect(_on_click_marker_animation_finished)
        _click_destination_marker_root.add_child(_click_destination_marker_sprite)
        return

    # 未配置 SpriteFrames 时，继续使用程序化 Line2D 圆环 + 十字作为兜底反馈。
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

## 是否已配置精灵帧版点击反馈。
func _uses_click_marker_sprite() -> bool:
    return click_marker_sprite_frames != null

## 解析点击反馈应播放的精灵动画名；配置缺失时回退到资源内首个动画。
func _resolve_click_marker_animation_name() -> String:
    if click_marker_sprite_frames == null:
        return ""
    var configured_name: String = click_marker_animation.strip_edges()
    if not configured_name.is_empty() and click_marker_sprite_frames.has_animation(configured_name):
        return configured_name
    var animation_names: PackedStringArray = click_marker_sprite_frames.get_animation_names()
    if animation_names.is_empty():
        return ""
    return animation_names[0]

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
    if _uses_click_marker_sprite():
        _show_click_destination_marker_sprite(world_position)
        return
    _show_click_destination_marker_legacy(world_position)

## 播放精灵帧版点击落点反馈。
func _show_click_destination_marker_sprite(world_position: Vector2) -> void:
    if _click_destination_marker_sprite == null:
        return
    if _click_destination_marker_tween != null:
        _click_destination_marker_tween.kill()
        _click_destination_marker_tween = null
    var animation_name: String = _resolve_click_marker_animation_name()
    if animation_name.is_empty():
        return
    _click_destination_marker_root.visible = true
    _click_destination_marker_root.position = world_position
    _click_destination_marker_root.scale = click_marker_scale
    _click_destination_marker_root.modulate = Color(1.0, 1.0, 1.0, 1.0)
    _click_destination_marker_sprite.visible = true
    _click_destination_marker_sprite.stop()
    _click_destination_marker_sprite.animation = animation_name
    _click_destination_marker_sprite.frame = 0
    _click_destination_marker_sprite.play(animation_name)

## 播放 Line2D 程序化圆环版点击落点反馈（未配置 SpriteFrames 时使用）。
func _show_click_destination_marker_legacy(world_position: Vector2) -> void:
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

## 精灵帧动画播完后隐藏落点特效。
func _on_click_marker_animation_finished() -> void:
    if _click_destination_marker_sprite != null:
        _click_destination_marker_sprite.stop()
    _hide_click_destination_marker()

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


## 生成优先保持直线的四方向路径；只有直线路段被障碍阻挡时才保留 AStar 绕行节点。
## start_cell 是玩家当前所在的可行走网格。
## target_cell 是本次寻路目标的可行走网格。
func _build_low_turn_navigation_path(start_cell: Vector2i, target_cell: Vector2i) -> Array[Vector2i]:
    var empty_path: Array[Vector2i] = []
    if _navigation_grid == null:
        return empty_path
    var raw_path: Array[Vector2i] = _navigation_grid.get_id_path(start_cell, target_cell)
    if raw_path.size() <= 2:
        return raw_path
    return _reduce_navigation_path_turns(raw_path)


## 从当前节点尝试直达尽可能靠后的 AStar 节点，重建不穿越障碍的低转弯路径。
## raw_path 是 AStarGrid2D 返回的完整可行走网格路径。
func _reduce_navigation_path_turns(raw_path: Array[Vector2i]) -> Array[Vector2i]:
    var reduced_path: Array[Vector2i] = []
    if raw_path.is_empty():
        return reduced_path
    reduced_path.append(raw_path[0])
    var current_index: int = 0
    var incoming_direction: Vector2i = Vector2i.ZERO
    while current_index < raw_path.size() - 1:
        var shortcut_path: Array[Vector2i] = []
        var shortcut_end_index: int = current_index + 1
        for candidate_index: int in range(raw_path.size() - 1, current_index, -1):
            var candidate_path: Array[Vector2i] = _build_clear_axis_path(
                raw_path[current_index],
                raw_path[candidate_index],
                incoming_direction
            )
            if candidate_path.is_empty():
                continue
            shortcut_path = candidate_path
            shortcut_end_index = candidate_index
            break
        if shortcut_path.is_empty():
            shortcut_path = [raw_path[current_index], raw_path[current_index + 1]]
        for path_index: int in range(1, shortcut_path.size()):
            reduced_path.append(shortcut_path[path_index])
        if shortcut_path.size() >= 2:
            incoming_direction = shortcut_path[shortcut_path.size() - 1] - shortcut_path[shortcut_path.size() - 2]
        current_index = shortcut_end_index
    return reduced_path


## 尝试用至多一个拐角连接两个网格，并选择与当前方向衔接转弯更少的方案。
## start_cell 是当前捷径起点。
## target_cell 是候选捷径终点。
## incoming_direction 是进入起点前的移动方向，用于避免连续反复转向。
func _build_clear_axis_path(
    start_cell: Vector2i,
    target_cell: Vector2i,
    incoming_direction: Vector2i
) -> Array[Vector2i]:
    if start_cell == target_cell:
        return [start_cell]
    if start_cell.x == target_cell.x or start_cell.y == target_cell.y:
        return _build_clear_straight_path(start_cell, target_cell)

    var horizontal_corner: Vector2i = Vector2i(target_cell.x, start_cell.y)
    var vertical_corner: Vector2i = Vector2i(start_cell.x, target_cell.y)
    var horizontal_first: Array[Vector2i] = _build_clear_corner_path(start_cell, horizontal_corner, target_cell)
    var vertical_first: Array[Vector2i] = _build_clear_corner_path(start_cell, vertical_corner, target_cell)
    if horizontal_first.is_empty():
        return vertical_first
    if vertical_first.is_empty():
        return horizontal_first

    var horizontal_turns: int = _count_navigation_path_turns(horizontal_first, incoming_direction)
    var vertical_turns: int = _count_navigation_path_turns(vertical_first, incoming_direction)
    if horizontal_turns < vertical_turns:
        return horizontal_first
    if vertical_turns < horizontal_turns:
        return vertical_first
    var horizontal_distance: int = absi(target_cell.x - start_cell.x)
    var vertical_distance: int = absi(target_cell.y - start_cell.y)
    return horizontal_first if horizontal_distance >= vertical_distance else vertical_first


## 构建经过指定拐角的两段直线路径；任一格不可站立时返回空路径。
## start_cell 是第一段起点。
## corner_cell 是两段路径共用的拐角。
## target_cell 是第二段终点。
func _build_clear_corner_path(
    start_cell: Vector2i,
    corner_cell: Vector2i,
    target_cell: Vector2i
) -> Array[Vector2i]:
    var first_segment: Array[Vector2i] = _build_clear_straight_path(start_cell, corner_cell)
    if first_segment.is_empty():
        return []
    var second_segment: Array[Vector2i] = _build_clear_straight_path(corner_cell, target_cell)
    if second_segment.is_empty():
        return []
    for path_index: int in range(1, second_segment.size()):
        first_segment.append(second_segment[path_index])
    return first_segment


## 构建同一行或同一列上的连续路径，并逐格验证边界与障碍。
## start_cell 是直线路段起点。
## target_cell 是直线路段终点。
func _build_clear_straight_path(start_cell: Vector2i, target_cell: Vector2i) -> Array[Vector2i]:
    var result: Array[Vector2i] = []
    if start_cell.x != target_cell.x and start_cell.y != target_cell.y:
        return result
    var step: Vector2i = Vector2i.ZERO
    if start_cell.x != target_cell.x:
        step.x = 1 if target_cell.x > start_cell.x else -1
    elif start_cell.y != target_cell.y:
        step.y = 1 if target_cell.y > start_cell.y else -1
    var current_cell: Vector2i = start_cell
    result.append(current_cell)
    while current_cell != target_cell:
        current_cell += step
        if not _is_navigation_cell_in_bounds(current_cell) or _navigation_grid.is_point_solid(current_cell):
            result.clear()
            return result
        result.append(current_cell)
    return result


## 统计路径方向变化次数，同时把进入路径前的方向纳入计算。
## cell_path 是待比较的连续四方向网格路径。
## incoming_direction 是进入首个路径点前的移动方向。
func _count_navigation_path_turns(cell_path: Array[Vector2i], incoming_direction: Vector2i) -> int:
    var turn_count: int = 0
    var previous_direction: Vector2i = incoming_direction
    for path_index: int in range(1, cell_path.size()):
        var current_direction: Vector2i = cell_path[path_index] - cell_path[path_index - 1]
        if previous_direction != Vector2i.ZERO and current_direction != previous_direction:
            turn_count += 1
        previous_direction = current_direction
    return turn_count

func _sync_local_player_battle_state() -> void:
    _apply_local_player_battle_pose()

## 由主场景在战斗弹窗显示/隐藏时调用，保证结算演出期间仍保持战斗待机动画。
func set_local_player_battle_pose_active(active: bool) -> void:
    _force_battle_pose_active = active
    _apply_local_player_battle_pose()
    if not active and not GameState.is_in_battle:
        refresh_pet_follower_after_battle()


## 战斗弹窗完全关闭且不再处于战斗态时，恢复出战宠物跟随展示。
func refresh_pet_follower_after_battle() -> void:
    _sync_pet_follower_visibility()
    _reset_pet_follow_near_player()


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
    ## 4013 可能先于战斗弹窗关闭把 is_in_battle 置 false；此时仍保持战斗待机，不要提前清宠物。
    if _force_battle_pose_active:
        return
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

## 返回当前世界玩家节点，供受控剧情动作场景调用公开的剧情接口。
func get_cinematic_player_node() -> CharacterBody2D:
    return player_node

## 把统一场景坐标中的剧情途经点转换为遵守地图碰撞的玩家本地路径。
func build_cinematic_player_path(scene_waypoints: Array[Vector2]) -> Array[Vector2]:
    var result: Array[Vector2] = []
    if player_node == null or _current_level == null or _navigation_grid == null or _navigation_layer == null:
        return result
    var start_cell: Vector2i = _resolve_walkable_navigation_cell(_world_to_navigation_cell(player_node.global_position))
    if not _is_navigation_cell_valid(start_cell):
        return result
    var scene_id: int = _current_scene_id()
    for scene_waypoint: Vector2 in scene_waypoints:
        var target_local_pixels: Vector2 = _scene_coordinate_to_local_pixels(scene_id, scene_waypoint)
        var target_world_position: Vector2 = _current_level.to_global(target_local_pixels)
        var target_cell: Vector2i = _resolve_walkable_navigation_cell(_world_to_navigation_cell(target_world_position))
        if not _is_navigation_cell_valid(target_cell):
            push_warning("剧情路径点不可到达: %s" % scene_waypoint)
            result.clear()
            return result
        var cell_path: Array[Vector2i] = _build_low_turn_navigation_path(start_cell, target_cell)
        if cell_path.is_empty():
            push_warning("剧情路径无法生成: %s" % scene_waypoint)
            result.clear()
            return result
        for path_index: int in range(cell_path.size()):
            if path_index == 0 and not result.is_empty():
                continue
            result.append(_navigation_cell_to_player_local_position(cell_path[path_index]))
        start_cell = target_cell
    return result

func _set_transition_loading(_active: bool) -> void:
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
    var scene_id: int = _current_scene_id()
    var local_pixels: Vector2 = player_node.position
    if _current_level != null and is_instance_valid(_current_level):
        local_pixels = _current_level.to_local(player_node.global_position)
    return _local_pixels_to_scene_coordinate(scene_id, local_pixels)

func _report_player_position_if_changed() -> void:
    if player_node == null or not is_instance_valid(player_node):
        return
    var current_position: Vector2 = _current_player_scene_position()
    # 网络表现同步需要在人物停止后补发 moving=false，即使当前坐标与上一帧相同也不能提前返回。
    _report_player_position_to_server(current_position)
    # HUD 和本地权威快照只需要整数场景格；逐像素重排文字和复制字典会造成移动时主线程抖动。
    var display_position: Vector2 = Vector2(roundi(current_position.x), roundi(current_position.y))
    if display_position.is_equal_approx(_last_reported_player_position):
        return
    _last_reported_player_position = display_position
    GameState.sync_player_scene_position(display_position)
    player_position_changed.emit(display_position, _current_player_global_position())

## 将场景脚本导出落点登记为本地移动同步基线，避免“刚落地”本身触发网络坐标上报。
## scene_position 是已经应用到本地人物的场景格坐标；玩家后续真正移动时仍会正常发送位置。
func _prime_local_movement_report_state(scene_position: Vector2) -> void:
    var facing: Vector2 = player_node.get_cardinal_direction() if player_node.has_method("get_cardinal_direction") else Vector2.DOWN
    _last_network_position = Vector2i(roundi(scene_position.x), roundi(scene_position.y))
    _last_network_precise_position = Vector2i(
        roundi(scene_position.x * float(NETWORK_POSITION_FIXED_SCALE)),
        roundi(scene_position.y * float(NETWORK_POSITION_FIXED_SCALE))
    )
    _last_network_facing = Vector2i(roundi(facing.x), roundi(facing.y))
    _last_network_moving = false
    _last_network_report_msec = Time.get_ticks_msec()
    _has_last_network_position = true
    _last_reported_player_position = Vector2(float(_last_network_position.x), float(_last_network_position.y))


## 上报整数持久化格和限频后的高精度表现状态；朝向、起停或跨格变化会立即发送。
## scene_position 是当前人物以场景格为单位的高精度位置。
func _report_player_position_to_server(scene_position: Vector2) -> void:
    var scene_id: int = _current_scene_id()
    if scene_id <= 0 or not GameState.is_ws_authenticated or _pending_target_scene_id != 0 or _pending_position_move_seq != 0:
        return
    var network_position: Vector2i = Vector2i(roundi(scene_position.x), roundi(scene_position.y))
    var precise_position: Vector2i = Vector2i(
        roundi(scene_position.x * float(NETWORK_POSITION_FIXED_SCALE)),
        roundi(scene_position.y * float(NETWORK_POSITION_FIXED_SCALE))
    )
    var facing: Vector2 = player_node.get_cardinal_direction() if player_node.has_method("get_cardinal_direction") else Vector2.DOWN
    var network_facing: Vector2i = Vector2i(roundi(facing.x), roundi(facing.y))
    var moving: bool = bool(player_node.call("is_walking")) if player_node.has_method("is_walking") else false
    # 当前移动输入只表达玩家意图；停止时必须发送零向量，服务端不会把候选坐标直接视为权威结果。
    var movement_input: Vector2i = network_facing if moving else Vector2i.ZERO
    var movement_state_changed: bool = (
        not _has_last_network_position
        or network_position != _last_network_position
        or network_facing != _last_network_facing
        or moving != _last_network_moving
    )
    var precise_position_changed: bool = precise_position != _last_network_precise_position
    if not movement_state_changed and not precise_position_changed:
        return
    var now_msec: int = Time.get_ticks_msec()
    if not movement_state_changed and now_msec - _last_network_report_msec < NETWORK_MOVEMENT_REPORT_INTERVAL_MS:
        return
    var move_seq: int = _take_next_move_seq()
    var request_seq: int = NetClient.send_command(
        CommandIds.MOVE_INTENT_REQ,
        {
            "op_id": _take_next_op_id(),
            "move_seq": move_seq,
            "scene_id": scene_id,
            "target_pos": {"x": network_position.x, "y": network_position.y},
            "precise_pos": {"x": precise_position.x, "y": precise_position.y},
            "facing": {"x": network_facing.x, "y": network_facing.y},
            "moving": moving,
            "input": {"x": movement_input.x, "y": movement_input.y},
            "client_tick": now_msec,
        }
    )
    if request_seq <= 0:
        return
    _pending_position_move_seq = move_seq
    _last_network_position = network_position
    _last_network_precise_position = precise_position
    _last_network_facing = network_facing
    _last_network_moving = moving
    _last_network_report_msec = now_msec
    _has_last_network_position = true

## 根据 GameState 的附近实体快照创建、更新或移除当前地图中的远端玩家节点。
func _sync_remote_players() -> void:
    if _current_level == null or not is_instance_valid(_current_level):
        return
    var expected_entity_ids: Dictionary = {}
    for entity_id_variant: Variant in GameState.nearby_entities.keys():
        var entity_variant: Variant = GameState.nearby_entities.get(entity_id_variant, {})
        if entity_variant is not Dictionary:
            continue
        var entity: Dictionary = entity_variant
        var entity_id: int = int(entity.get("entity_id", entity_id_variant))
        var remote_player_id: int = int(entity.get("player_id", 0))
        if int(entity.get("entity_type", 0)) != PLAYER_ENTITY_TYPE:
            continue
        if entity_id <= 0 or remote_player_id == GameState.player_id:
            continue
        expected_entity_ids[entity_id] = true
        var remote_node: player = _remote_player_nodes.get(entity_id, null) as player
        var position_variant: Variant = entity.get("pos", {})
        var server_position: Vector2 = Vector2(
            float(position_variant.get("x", entity.get("x", 0.0))) if position_variant is Dictionary else float(entity.get("x", 0.0)),
            float(position_variant.get("y", entity.get("y", 0.0))) if position_variant is Dictionary else float(entity.get("y", 0.0))
        )
        var presentation_scene_position: Vector2 = server_position
        # ENTITY_ENTER_PUSH 与世界快照中的玩家实体没有 precise_pos 字段；
        # 这里必须排除空字典，否则缺省值 {} 会把远端出生点算成 (0,0) 地图左上角。
        var precise_position_variant: Variant = entity.get("precise_pos", null)
        if precise_position_variant is Dictionary and not (precise_position_variant as Dictionary).is_empty():
            var precise_position: Dictionary = precise_position_variant as Dictionary
            presentation_scene_position = Vector2(
                float(precise_position.get("x", 0)) / float(NETWORK_POSITION_FIXED_SCALE),
                float(precise_position.get("y", 0)) / float(NETWORK_POSITION_FIXED_SCALE)
            )
        var local_position: Vector2 = _server_to_local_position(_current_scene_id(), presentation_scene_position)
        var facing_direction: Vector2 = _extract_remote_facing_direction(entity)
        var has_motion_state: bool = entity.has("moving") and entity.has("facing")
        var remote_moving: bool = bool(entity.get("moving", false))
        if remote_node == null or not is_instance_valid(remote_node):
            remote_node = REMOTE_PLAYER_SCENE.instantiate() as player
            if remote_node == null:
                continue
            remote_node.name = "RemotePlayer_%d" % entity_id
            remote_node.is_remote_avatar = true
            _get_player_host(_current_level).add_child(remote_node)
            # 远端实例不能读取本机 GameState.player_snapshot；必须显式消费该实体自己的服务端 skin_id。
            remote_node.apply_remote_skin_id(str(entity.get("skin_id", "")))
            remote_node.apply_remote_initial_position(local_position)
            if has_motion_state:
                remote_node.set_remote_motion_target(local_position, facing_direction, remote_moving)
            _configure_actor_y_sort(remote_node)
            _remote_player_nodes[entity_id] = remote_node
        else:
            # ENTITY_ENTER_PUSH 也用于形象或编队刷新，已有节点同样要重新应用权威人物形象。
            remote_node.apply_remote_skin_id(str(entity.get("skin_id", "")))
            if has_motion_state:
                remote_node.set_remote_motion_target(local_position, facing_direction, remote_moving)
            else:
                remote_node.set_remote_target_position(local_position)
        _sync_remote_player_pet(entity_id, entity, local_position, facing_direction, remote_moving)
    for entity_id_variant: Variant in _remote_player_nodes.keys():
        var entity_id: int = int(entity_id_variant)
        if expected_entity_ids.has(entity_id):
            continue
        _remove_remote_player_entity(entity_id)

## 根据玩家实体中的权威首只编队宠物创建或更新远端跟随表现。
## entity_id 是远端玩家的场景实体标识。
## entity 是服务端下发的玩家实体摘要。
## leader_target_position 是远端玩家本次收到的高精度本地目标坐标。
## leader_facing 是服务端归一化后的远端玩家朝向。
## leader_moving 表示远端玩家是否仍在移动。
func _sync_remote_player_pet(
    entity_id: int,
    entity: Dictionary,
    leader_target_position: Vector2,
    leader_facing: Vector2,
    leader_moving: bool
) -> void:
    var previous_position_variant: Variant = _remote_player_target_positions.get(entity_id, Vector2.INF)
    var previous_target_position: Vector2 = previous_position_variant as Vector2
    _remote_player_target_positions[entity_id] = leader_target_position
    var following_pet_variant: Variant = entity.get("following_pet", {})
    if following_pet_variant is not Dictionary or (following_pet_variant as Dictionary).is_empty():
        _remove_remote_pet_follower(entity_id)
        return
    var following_pet: Dictionary = following_pet_variant as Dictionary
    var follower: WorldPetFollower = _remote_pet_follower_nodes.get(entity_id, null) as WorldPetFollower
    if follower == null or not is_instance_valid(follower):
        follower = WorldPetFollower.new()
        follower.name = "RemotePetFollower_%d" % entity_id
        _get_player_host(_current_level).add_child(follower)
        _configure_actor_y_sort(follower)
        _remote_pet_follower_nodes[entity_id] = follower
        follower.sync_lineup_pet(following_pet)
        var initial_direction: Vector2 = leader_facing
        if initial_direction == Vector2.ZERO:
            initial_direction = _remote_move_direction(previous_target_position, leader_target_position)
        follower.reset_near_leader(leader_target_position, _remote_pet_reset_offset(initial_direction))
        _remote_pet_path_anchors[entity_id] = leader_target_position
        _remote_pet_path_directions[entity_id] = initial_direction
        return
    follower.sync_lineup_pet(following_pet)
    if not leader_moving or previous_target_position == Vector2.INF or previous_target_position.is_equal_approx(leader_target_position):
        return
    var move_direction: Vector2 = leader_facing
    if move_direction == Vector2.ZERO:
        move_direction = _remote_move_direction(previous_target_position, leader_target_position)
    if move_direction != Vector2.ZERO:
        _record_remote_pet_leader_path(entity_id, follower, leader_target_position, move_direction)


## 从远端玩家高精度轨迹中按 24px 间隔记录宠物路径，避免网络采样点被当成完整格步。
## entity_id 是远端玩家场景实体标识。
## follower 是该玩家对应的宠物跟随节点。
## leader_position 是远端玩家本次权威表现目标位置。
## leader_direction 是服务端明确下发的四方向朝向。
func _record_remote_pet_leader_path(
    entity_id: int,
    follower: WorldPetFollower,
    leader_position: Vector2,
    leader_direction: Vector2
) -> void:
    var anchor_variant: Variant = _remote_pet_path_anchors.get(entity_id, leader_position)
    var anchor: Vector2 = anchor_variant as Vector2
    var previous_direction_variant: Variant = _remote_pet_path_directions.get(entity_id, Vector2.ZERO)
    var previous_direction: Vector2 = previous_direction_variant as Vector2
    if previous_direction != Vector2.ZERO and previous_direction != leader_direction:
        anchor = leader_position
    _remote_pet_path_directions[entity_id] = leader_direction
    var recorded_steps: int = 0
    while leader_position.distance_to(anchor) >= PathFollowController.PATH_STEP_SIZE:
        if recorded_steps >= PET_FOLLOW_MAX_LEADER_STEPS_PER_FRAME:
            anchor = leader_position
            break
        var previous_distance: float = leader_position.distance_to(anchor)
        follower.push_leader_step(anchor, leader_direction)
        anchor += leader_direction * PathFollowController.PATH_STEP_SIZE
        recorded_steps += 1
        if leader_position.distance_to(anchor) >= previous_distance:
            anchor = leader_position
            break
    _remote_pet_path_anchors[entity_id] = anchor


## 按帧推进所有远端宠物沿其所属玩家的权威移动路径跟随。
## delta 是当前渲染帧间隔秒数。
func _update_remote_pet_followers(delta: float) -> void:
    for entity_id_variant: Variant in _remote_pet_follower_nodes.keys():
        var entity_id: int = int(entity_id_variant)
        var follower: WorldPetFollower = _remote_pet_follower_nodes.get(entity_id, null) as WorldPetFollower
        if follower == null or not is_instance_valid(follower) or not follower.visible:
            continue
        var remote_node: player = _remote_player_nodes.get(entity_id, null) as player
        var move_speed: float = remote_node.get_move_speed() if remote_node != null and is_instance_valid(remote_node) else 100.0
        follower.update_follow(delta, move_speed)


## 从远端实体快照读取四方向朝向；字段缺失或非法时返回零向量交给兼容逻辑推导。
## entity 是服务端下发并合并后的远端实体快照。
func _extract_remote_facing_direction(entity: Dictionary) -> Vector2:
    var facing_variant: Variant = entity.get("facing", {})
    if facing_variant is not Dictionary:
        return Vector2.ZERO
    var facing: Dictionary = facing_variant as Dictionary
    var direction: Vector2 = Vector2(float(facing.get("x", 0)), float(facing.get("y", 0)))
    if direction == Vector2.LEFT or direction == Vector2.RIGHT or direction == Vector2.UP or direction == Vector2.DOWN:
        return direction
    return Vector2.ZERO


## 计算远端玩家两次权威坐标之间的四方向移动方向。
## from_position 是上一次目标坐标。
## to_position 是本次目标坐标。
func _remote_move_direction(from_position: Vector2, to_position: Vector2) -> Vector2:
    if from_position == Vector2.INF:
        return Vector2.DOWN
    var offset: Vector2 = to_position - from_position
    if absf(offset.x) >= absf(offset.y) and not is_zero_approx(offset.x):
        return Vector2.RIGHT if offset.x > 0.0 else Vector2.LEFT
    if not is_zero_approx(offset.y):
        return Vector2.DOWN if offset.y > 0.0 else Vector2.UP
    return Vector2.ZERO


## 根据远端玩家移动方向计算宠物初始站位，使宠物出现在人物身后半格。
## move_direction 是远端玩家当前四方向移动方向。
func _remote_pet_reset_offset(move_direction: Vector2) -> Vector2:
    var resolved_direction: Vector2 = move_direction if move_direction != Vector2.ZERO else Vector2.DOWN
    return -resolved_direction * PathFollowController.PATH_STEP_SIZE * 0.5


## 删除指定远端玩家及其宠物跟随表现和路径缓存。
## entity_id 是需要清理的远端玩家场景实体标识。
func _remove_remote_player_entity(entity_id: int) -> void:
    var remote_node: player = _remote_player_nodes.get(entity_id, null) as player
    if remote_node != null and is_instance_valid(remote_node):
        remote_node.queue_free()
    _remote_player_nodes.erase(entity_id)
    _remove_remote_pet_follower(entity_id)
    _remote_player_target_positions.erase(entity_id)


## 删除指定远端玩家的宠物跟随节点；玩家本体仍可继续保留。
## entity_id 是宠物所属远端玩家的场景实体标识。
func _remove_remote_pet_follower(entity_id: int) -> void:
    var follower: WorldPetFollower = _remote_pet_follower_nodes.get(entity_id, null) as WorldPetFollower
    if follower != null and is_instance_valid(follower):
        follower.queue_free()
    _remote_pet_follower_nodes.erase(entity_id)
    _remote_pet_path_anchors.erase(entity_id)
    _remote_pet_path_directions.erase(entity_id)

## 清理旧地图上的远端玩家引用，避免切图后继续更新已经释放的节点。
func _clear_remote_players() -> void:
    for entity_id_variant: Variant in _remote_player_nodes.keys():
        _remove_remote_player_entity(int(entity_id_variant))
    _remote_pet_follower_nodes.clear()
    _remote_player_target_positions.clear()
    _remote_pet_path_anchors.clear()
    _remote_pet_path_directions.clear()
    _has_last_network_position = false
    _last_network_report_msec = 0


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
    _leader_path_direction = Vector2.ZERO
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
        _leader_path_direction = leader_direction
        return
    if _leader_path_direction != Vector2.ZERO and _leader_path_direction != leader_direction:
        # 转向时从当前拐角重新累计格距，避免旧方向不足一格的残余距离被投射到新方向。
        _leader_path_anchor = leader_position
    _leader_path_direction = leader_direction

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
