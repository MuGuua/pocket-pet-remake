extends Node
class_name WorldPlayerCinematic

## 当前动作场景完成后通知剧情播放器推进服务端节点。
signal finished
## 请求主场景使用现有对话面板展示一句客户端固定对白。
signal local_dialogue_requested(
    speaker_name: String,
    content: String,
    portrait_key: String,
    is_player_speaking: bool,
    content_format: String,
    portrait_texture: Texture2D
)
## 玩家点击本地对白继续后唤醒当前过场脚本。
signal local_dialogue_advanced

@export_group("路径")
## 玩家依次经过的统一场景坐标；运行时会通过当前地图导航网格生成碰撞路径。
@export var scene_waypoints: Array[Vector2] = []
## 路径完成后的朝向；零向量表示保持移动结束时的朝向。
@export var final_facing_direction: Vector2 = Vector2.ZERO
## 路径最长等待秒数；动态障碍持续阻挡时自动结束，避免剧情永久卡死。
@export_range(1.0, 120.0, 1.0) var path_timeout_seconds: float = 20.0

@export_group("动作")
## 路径完成后播放的角色动画名；留空表示只移动。
@export var animation_name: String = ""
## 大于等于零时暂停在该动画帧；负数表示正常播放动画。
@export var animation_frame: int = -1
## 旧 AnimationPlayer 将帧号换算为时间时使用的帧率。
@export_range(1.0, 60.0, 1.0) var animation_frame_fps: float = 12.0
## 动作或指定帧保持时长，结束后恢复世界待机状态。
@export_range(0.0, 30.0, 0.1) var animation_hold_seconds: float = 0.8

## 主场景注入的世界控制器。
var _world_controller: Node = null
## 当前被剧情控制的真实玩家节点。
var _player_node: CharacterBody2D = null
## 防止退出树和正常结束重复恢复玩家状态。
var _control_active: bool = false
## 保证完成信号最多发出一次。
var _finished_emitted: bool = false
## 当前由固定过场创建的移动 Tween，中断过场时必须同步停止。
var _active_move_tween: Tween = null
## 当前是否正在等待玩家推进客户端固定对白。
var _waiting_local_dialogue: bool = false

## 在节点入树前接收当前世界控制器，避免剧情场景自行查找全局节点路径。
func configure_world_context(world_controller: Node) -> void:
    _world_controller = world_controller

## 入树后延迟一帧启动，确保当前世界玩家与导航网格已经就绪。
func _ready() -> void:
    call_deferred("_run_sequence")

## 场景被外部提前释放时恢复玩家控制状态。
func _exit_tree() -> void:
    if _active_move_tween != null and _active_move_tween.is_valid():
        _active_move_tween.kill()
    _active_move_tween = null
    _restore_player_control()

## 获取玩家并进入过场控制；自定义动画 Key 场景应在执行 Tween 前调用。
func begin_player_cinematic() -> bool:
    if _world_controller == null or not _world_controller.has_method("get_cinematic_player_node"):
        return false
    _player_node = _world_controller.call("get_cinematic_player_node") as CharacterBody2D
    if _player_node == null:
        return false
    _player_node.call("begin_cinematic_control")
    _control_active = true
    return true

## 使用线性 Tween 让玩家按固定像素偏移移动，并在移动期间播放指定方向动画。
## 该方法直接修改 position，不处理墙体或动态障碍，只应用于已确认畅通的固定过场路线。
func tween_player_offset(
    relative_offset: Vector2,
    duration_seconds: float,
    walk_animation_name: String
) -> bool:
    if not _control_active or _player_node == null:
        return false
    if _active_move_tween != null and _active_move_tween.is_valid():
        _active_move_tween.kill()
    var normalized_animation_name: String = walk_animation_name.strip_edges()
    if not normalized_animation_name.is_empty():
        _player_node.call("play_cinematic_animation", normalized_animation_name, -1, 12.0)
    var safe_duration: float = maxf(duration_seconds, 0.01)
    var target_position: Vector2 = _player_node.position + relative_offset
    _active_move_tween = create_tween()
    _active_move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
    _active_move_tween.set_trans(Tween.TRANS_LINEAR)
    _active_move_tween.tween_property(_player_node, "position", target_position, safe_duration)
    await _active_move_tween.finished
    _active_move_tween = null
    return _control_active and _player_node != null and is_instance_valid(_player_node)

## 设置固定过场结束或转段时的玩家朝向。
func set_cinematic_player_facing(facing_direction: Vector2) -> void:
    if _player_node == null:
        return
    _player_node.call("set_facing_direction", facing_direction)

## 按世界场景的相同算法，把过场地图可用矩形左上角归一到世界原点。
func normalize_cinematic_level_origin(cinematic_level: Node2D) -> void:
    if cinematic_level == null:
        return
    var level_rect: Rect2 = _resolve_cinematic_level_rect(cinematic_level)
    if not level_rect.has_area() or level_rect.position.is_equal_approx(Vector2.ZERO):
        return
    cinematic_level.global_position -= level_rect.position

## 将固定过场相机的居中、缩放和地图边界同步为当前世界玩家相机的实际参数。
func sync_cinematic_camera_with_world(
    cinematic_camera: Camera2D,
    cinematic_level: Node2D = null
) -> void:
    if cinematic_camera == null:
        return
    if _world_controller != null and _world_controller.has_method("get_cinematic_player_node"):
        var world_player: CharacterBody2D = _world_controller.call("get_cinematic_player_node") as CharacterBody2D
        if world_player != null:
            var world_camera: Camera2D = world_player.get_node_or_null("Camera2D") as Camera2D
            if world_camera != null:
                cinematic_camera.position = world_camera.position
                cinematic_camera.offset = world_camera.offset
                cinematic_camera.zoom = world_camera.zoom
                cinematic_camera.anchor_mode = world_camera.anchor_mode
    _apply_cinematic_camera_limits(cinematic_camera, cinematic_level)
    cinematic_camera.make_current()

## 按过场地图归一后的实际矩形设置相机边界，与世界控制器的边界计算保持一致。
func _apply_cinematic_camera_limits(cinematic_camera: Camera2D, cinematic_level: Node2D) -> void:
    if cinematic_level == null:
        return
    var level_rect: Rect2 = _resolve_cinematic_level_rect(cinematic_level)
    if not level_rect.has_area():
        return
    cinematic_camera.limit_left = int(round(level_rect.position.x))
    cinematic_camera.limit_top = int(round(level_rect.position.y))
    cinematic_camera.limit_right = int(round(level_rect.end.x))
    cinematic_camera.limit_bottom = int(round(level_rect.end.y))

## 根据世界控制器使用的图层优先级解析过场地图的像素矩形。
func _resolve_cinematic_level_rect(cinematic_level: Node2D) -> Rect2:
    var limit_layer: TileMapLayer = _find_cinematic_limit_layer(cinematic_level)
    if limit_layer == null:
        return Rect2()
    var used_rect: Rect2i = limit_layer.get_used_rect()
    if not used_rect.has_area() or limit_layer.tile_set == null:
        return Rect2()
    var tile_size: Vector2 = Vector2(limit_layer.tile_set.tile_size)
    var top_left_local: Vector2 = limit_layer.map_to_local(used_rect.position) - tile_size * 0.5
    var bottom_right_cell: Vector2i = used_rect.position + used_rect.size - Vector2i.ONE
    var bottom_right_local: Vector2 = limit_layer.map_to_local(bottom_right_cell) + tile_size * 0.5
    var top_left_global: Vector2 = limit_layer.to_global(top_left_local)
    var bottom_right_global: Vector2 = limit_layer.to_global(bottom_right_local)
    return Rect2(top_left_global, bottom_right_global - top_left_global)

## 查找世界地图用于计算原点和相机边界的 TileMapLayer。
func _find_cinematic_limit_layer(cinematic_level: Node2D) -> TileMapLayer:
    if cinematic_level == null:
        return null
    for layer_name: String in ["Collision", "Bottom", "Map", "TileMapLayer"]:
        var layer: TileMapLayer = cinematic_level.get_node_or_null(layer_name) as TileMapLayer
        if layer != null:
            return layer
    return null

## 展示并等待一句完全由客户端过场脚本定义的对白。
func show_local_dialogue(
    speaker_name: String,
    content: String,
    portrait_key: String = "",
    is_player_speaking: bool = false,
    content_format: String = "bbcode",
    portrait_texture: Texture2D = null
) -> void:
    if _finished_emitted:
        return
    _waiting_local_dialogue = true
    local_dialogue_requested.emit(
        speaker_name,
        content,
        portrait_key,
        is_player_speaking,
        content_format,
        portrait_texture
    )
    await local_dialogue_advanced
    _waiting_local_dialogue = false

## 接收对话面板的本地继续操作；服务端剧情继续操作不会进入这里。
func advance_local_dialogue() -> void:
    if not _waiting_local_dialogue:
        return
    local_dialogue_advanced.emit()

## 结束自定义动画 Key 的完整客户端过场，并恢复玩家控制。
func complete_cinematic() -> void:
    _finish_sequence("")

## 串行执行路径、朝向和动作帧，任一步失败都会安全结束并恢复玩家。
func _run_sequence() -> void:
    if not begin_player_cinematic():
        _finish_sequence("剧情找不到世界玩家")
        return
    if not scene_waypoints.is_empty():
        var path_variant: Variant = _world_controller.call("build_cinematic_player_path", scene_waypoints)
        var local_path: Array[Vector2] = []
        if path_variant is Array:
            for point_variant: Variant in path_variant:
                if point_variant is Vector2:
                    local_path.append(point_variant as Vector2)
        if local_path.is_empty():
            _finish_sequence("剧情路径为空或不可到达")
            return
        _player_node.call("set_auto_move_path", local_path)
        var path_deadline_ms: int = Time.get_ticks_msec() + int(path_timeout_seconds * 1000.0)
        while bool(_player_node.call("is_auto_move_active")) and Time.get_ticks_msec() < path_deadline_ms:
            await get_tree().physics_frame
        if bool(_player_node.call("is_auto_move_active")):
            _player_node.call("clear_auto_move_path")
            _finish_sequence("剧情路径执行超时")
            return
    if final_facing_direction != Vector2.ZERO:
        _player_node.call("set_facing_direction", final_facing_direction)
    var normalized_animation_name: String = animation_name.strip_edges()
    if not normalized_animation_name.is_empty():
        var played: bool = bool(_player_node.call("play_cinematic_animation", normalized_animation_name, animation_frame, animation_frame_fps))
        if not played:
            push_warning("剧情角色动画不存在: %s" % normalized_animation_name)
        if animation_hold_seconds > 0.0:
            await get_tree().create_timer(animation_hold_seconds).timeout
    _finish_sequence("")

## 恢复玩家控制并广播完成；reason 非空时同时输出可定位的告警。
func _finish_sequence(reason: String) -> void:
    if not reason.is_empty():
        push_warning(reason)
    _restore_player_control()
    if _finished_emitted:
        return
    _finished_emitted = true
    finished.emit()

## 只恢复一次玩家剧情控制，避免场景退出时覆盖后续状态。
func _restore_player_control() -> void:
    if not _control_active or _player_node == null:
        return
    _control_active = false
    _player_node.call("end_cinematic_control")
