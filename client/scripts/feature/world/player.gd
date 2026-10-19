class_name player
extends CharacterBody2D

## 剧情或自动寻路完整走完当前路径后发出；主动取消路径不会触发。
signal auto_move_finished

const CharacterVisualScene: PackedScene = preload("res://scenes/character/character_visual.tscn")

# 角色待机状态标识。
const STATE_IDLE := "idle"
# 角色行走状态标识。
const STATE_WALK := "walk"
# 角色战斗锁定状态标识。
const STATE_BATTLE := "battle"
# 无 skin_id 时使用的默认战斗待机动画名，对应 player.tscn 中的 battle_pose（第 12 帧）。
const LEGACY_BATTLE_ANIMATION := "battle_pose"
# 世界场景进入战斗后统一替换的形象资源 ID，对应 resources/battle/unit_skins/其他/战斗待机_004.tres。
const WORLD_BATTLE_SKIN_ID: String = "战斗待机_004"
## 远端角色的插值渲染延迟：观察端固定落后发送方这么多毫秒，用已收到的两个快照之间插值。
## 需要大于发送间隔（100ms）再加上网络抖动余量，抖动在此范围内时远端移动完全匀速。
const REMOTE_INTERPOLATION_DELAY_MS: float = 150.0
## 缓冲耗尽（丢包或长时间卡顿）时允许沿最后朝向外推的最长毫秒数，超过后原地等待新包。
const REMOTE_MAX_EXTRAPOLATION_MS: float = 200.0
## 每个远端角色最多缓存的快照数量，防止后台挂起后堆积。
const REMOTE_SNAPSHOT_BUFFER_LIMIT: int = 32
## 新包到达导致画面目标点突变时，视觉误差按此速率指数衰减，避免肉眼可见的瞬移。
const REMOTE_ERROR_DECAY_PER_SECOND: float = 12.0
## 超过该距离的突变视为传送，直接瞬移，不做平滑。
const REMOTE_SNAP_DISTANCE_PX: float = 96.0
## 时钟偏移估计向较大样本缓慢靠拢的系数，用于适应网络延迟整体上升。
const REMOTE_CLOCK_OFFSET_RISE_RATE: float = 0.02

# 本地角色移动速度。
@export var move_speed: float = 90.0
# 世界相机统一缩放；大于 1 会放大画面，小于 1 会缩小画面。
@export var camera_zoom_scale: float = 2.0
# 世界相机相对玩家的垂直偏移；负值表示画面整体上移。
@export var camera_vertical_offset: float = 150.0
## 标记该实例是否代表其他在线玩家；远端角色不读取输入、不启用相机和碰撞。
@export var is_remote_avatar: bool = false

# 当前角色朝向的四方向单位向量。
var cardinal_direction: Vector2 = Vector2.DOWN
# 当前输入驱动得到的移动方向向量。
var direction: Vector2 = Vector2.ZERO
# 当前角色状态机所在状态。
var state: String = STATE_IDLE
# 标记切图期间是否锁定移动。
var _scene_transition_locked: bool = false
# 标记战斗期间是否锁定移动。
var _battle_locked: bool = false
# 当前自动寻路剩余的本地路径点列表。
var _auto_move_path: Array[Vector2] = []
# 自动寻路判定到达路径点时使用的像素容差。
var _auto_move_stop_tolerance: float = 3.0
## 当前自动路径是否需要在自然完成后发出完成信号。
var _auto_move_active: bool = false
## 路径请求代次，避免旧的延迟完成回调误结束新路径。
var _auto_move_generation: int = 0
## 剧情控制期间屏蔽方向键，但仍允许脚本自动路径驱动移动。
var _cinematic_input_locked: bool = false
## 进入剧情控制前的场景锁状态，演出结束时原样恢复。
var _cinematic_previous_scene_lock: bool = false
# 负责播放角色动画的动画播放器节点。
@onready var animation_player: AnimationPlayer = $AnimationPlayer
@onready var legacy_sprite: Sprite2D = $Sprite2D
@onready var camera_node: Camera2D = $Camera2D
@onready var body_collision_shape: CollisionShape2D = $CollisionShape2D
@onready var click_collision_shape: CollisionShape2D = $Area2D/CollisionShape2D

const LEGACY_COLLISION_OFFSET := Vector2(13.0, -7.0)

var _character_visual: CharacterVisual = null
var _uses_character_visual: bool = false
# 标记当前是否正在使用战斗专用形象覆盖正常 skin_id。
var _battle_skin_override_active: bool = false
## 远端角色按发送方采样时间排序的移动快照，每项为 { "t": float, "pos": Vector2, "facing": Vector2, "moving": bool }。
## t 位于发送方时钟域（毫秒）；旧服务未转发 client_tick 时用本地接收时间换算到同一时钟域。
var _remote_snapshots: Array[Dictionary] = []
## 标记远端角色是否已经取得有效初始位置。
var _remote_target_initialized: bool = false
## 本地时钟减发送方时钟的估计值（毫秒），取最快到达的包作为基线，从而为慢包留出抖动余量。
var _remote_clock_offset_ms: float = 0.0
## 时钟偏移是否已经初始化。
var _remote_clock_offset_valid: bool = false
## 当前画面位置与插值理想位置之间的视觉误差，新包修正目标时逐帧衰减到零。
var _remote_render_error: Vector2 = Vector2.ZERO
## 远端角色最近一次成功应用的服务端人物形象 ID，用于跳过重复的形象刷新。
var _remote_applied_skin_id: String = ""
## 相机在人物本地坐标中的固定跟随锚点；像素吸附时使用原始值，避免逐帧累计舍入误差。
var _camera_follow_local_position: Vector2 = Vector2.ZERO

func _ready() -> void:
	if camera_node != null:
		_camera_follow_local_position = camera_node.position
	if camera_node != null and not is_remote_avatar:
		camera_node.make_current()
	if is_remote_avatar:
		collision_layer = 0
		collision_mask = 0
		if body_collision_shape != null:
			body_collision_shape.disabled = true
		if click_collision_shape != null:
			click_collision_shape.disabled = true
		if camera_node != null:
			camera_node.enabled = false
	_apply_camera_zoom()
	_apply_camera_offset()
	_sync_pixel_camera_position()
	_setup_character_visual()
	if is_remote_avatar:
		return
	if not GameState.world_snapshot_changed.is_connected(_on_world_snapshot_changed):
		GameState.world_snapshot_changed.connect(_on_world_snapshot_changed)
	_sync_skin_from_snapshot()

func _exit_tree() -> void:
	if GameState.world_snapshot_changed.is_connected(_on_world_snapshot_changed):
		GameState.world_snapshot_changed.disconnect(_on_world_snapshot_changed)

# 每帧读取输入并更新角色状态与动画。
func _process(_delta: float) -> void:
	if is_remote_avatar:
		_update_remote_avatar(_delta)
		return
	if _is_movement_locked():
		# 锁定移动时同时清空自动寻路，避免切图或战斗结束后继续沿旧路径前进。
		_clear_auto_move_path()
		direction = Vector2.ZERO
		velocity = Vector2.ZERO
		if _update_state():
			_update_animation()
		return

	# 剧情演出屏蔽手动方向输入，但保留自动路径驱动。
	if _cinematic_input_locked:
		direction = Vector2.ZERO
	else:
		direction.x = Input.get_action_strength("ui_right") - Input.get_action_strength("ui_left")
		direction.y = Input.get_action_strength("ui_down") - Input.get_action_strength("ui_up")
	if direction != Vector2.ZERO:
		# 玩家手动输入优先级最高，按下方向键后立即取消点击寻路。
		_clear_auto_move_path()
	else:
		# 没有手动输入时，尝试继续沿自动寻路路径前进。
		_update_auto_move_direction()
	if direction.x != 0.0 and direction.y != 0.0:
		# 斜向输入时只保留绝对值更大的主方向，保持四方向移动口径。
		if abs(direction.x) >= abs(direction.y):
			direction.y = 0.0
		else:
			direction.x = 0.0

	velocity = direction * move_speed

	if _update_state() or _set_direction():
		_update_animation()

# 在物理帧中执行角色移动。
func _physics_process(_delta: float) -> void:
	if is_remote_avatar:
		return
	# CharacterBody2D 保留连续浮点坐标，避免 90px/s 在 60Hz 下被拆成 1px、2px 的交替跳动；
	# 像素素材的最终采样继续由项目级像素吸附和 nearest 过滤负责，不修改碰撞与服务端坐标口径。
	move_and_slide()
	_sync_pixel_camera_position()

## 返回当前四方向朝向，供宠物跟随等世界表现层读取。
func get_cardinal_direction() -> Vector2:
	return cardinal_direction

## 返回当前是否正在行走（非锁定且有移动输入或自动寻路）。
func is_walking() -> bool:
	return not _is_movement_locked() and direction != Vector2.ZERO

## 返回当前移动速度（像素/秒）。
func get_move_speed() -> float:
	return move_speed

# 把服务端权威坐标直接应用到当前角色显示位置。
func apply_authoritative_position(local_position: Vector2) -> void:
	# 服务端坐标换算成像素后，统一吸附到整数像素，避免世界初次进入时立刻出现发糊。
	position = local_position.round()
	_clear_auto_move_path()
	velocity = Vector2.ZERO
	direction = Vector2.ZERO
	_scene_transition_locked = false
	_sync_pixel_camera_position()
	if _update_state():
		_update_animation()

## 立即放置远端角色，首次创建、切图重建或传送时不播放从旧位置滑过来的插值。
## local_position 是服务端权威场景坐标换算后的本地像素坐标。
func apply_remote_initial_position(local_position: Vector2) -> void:
	# 远端表现保留浮点坐标；项目级像素吸附只影响最终绘制，不应反向破坏插值轨迹。
	_remote_snapshots.clear()
	_remote_target_initialized = true
	_remote_render_error = Vector2.ZERO
	position = local_position
	direction = Vector2.ZERO
	velocity = Vector2.ZERO
	_update_state()
	_update_animation()


## 应用远端玩家实体摘要携带的服务端权威人物形象。
## skin_id 对应 CharacterSkinRegistry 中注册的 UnitSkin；为空或资源无效时回退到旧人物精灵。
func apply_remote_skin_id(skin_id: String) -> void:
	if not is_remote_avatar:
		return
	var normalized_skin_id: String = skin_id.strip_edges()
	# 移动推送每 100ms 会触发一次实体同步；形象未变化时跳过重复应用，避免每包都刷新可见性与动画。
	if normalized_skin_id == _remote_applied_skin_id:
		return
	_remote_applied_skin_id = normalized_skin_id
	_apply_normal_skin_id(normalized_skin_id)


## 写入不带移动状态的远端目标点（兼容旧协议的实体摘要），按静止快照处理。
## local_position 是服务端整数坐标换算后的本地像素坐标。
func set_remote_target_position(local_position: Vector2) -> void:
	if not _remote_target_initialized:
		apply_remote_initial_position(local_position)
		return
	var facing: Vector2 = cardinal_direction
	if not _remote_snapshots.is_empty():
		var last: Dictionary = _remote_snapshots.back()
		var inferred: Vector2 = _resolve_cardinal_direction(local_position - (last["pos"] as Vector2))
		if inferred != Vector2.ZERO:
			facing = inferred
		elif (last["pos"] as Vector2).is_equal_approx(local_position):
			# 摘要刷新（换形象、换编队）重复携带同一坐标时，不应打断正在播放的移动插值。
			return
	_push_remote_snapshot(local_position, facing, false, 0)


## 写入一条远端移动快照，后续帧在相邻快照之间按发送方时间线性插值。
## local_position 是换算后的地图像素坐标。
## facing_direction 是服务端归一化后的四方向单位向量。
## moving 表示发送方在采样时是否仍在移动。
## sample_tick_ms 是发送方采样时的客户端单调时钟毫秒数；旧服务未转发时传 0，退化为本地接收时间。
func set_remote_motion_target(local_position: Vector2, facing_direction: Vector2, moving: bool, sample_tick_ms: int = 0) -> void:
	var resolved_direction: Vector2 = _resolve_cardinal_direction(facing_direction)
	if resolved_direction == Vector2.ZERO:
		resolved_direction = cardinal_direction if cardinal_direction != Vector2.ZERO else Vector2.DOWN
	if not _remote_target_initialized:
		apply_remote_initial_position(local_position)
		cardinal_direction = resolved_direction
	_push_remote_snapshot(local_position, resolved_direction, moving, sample_tick_ms)


## 把快照按发送方时间插入缓冲区，并维护时钟偏移估计与视觉误差，保证画面连续。
func _push_remote_snapshot(local_position: Vector2, facing: Vector2, moving: bool, sample_tick_ms: int) -> void:
	var now_ms: float = float(Time.get_ticks_msec())
	var tick: float = float(sample_tick_ms)
	if sample_tick_ms <= 0:
		# 无发送方时间戳：把本地接收时刻换算到发送方时钟域，等价于按到达时间插值。
		tick = now_ms - _remote_clock_offset_ms if _remote_clock_offset_valid else now_ms
	if not _remote_snapshots.is_empty():
		var last: Dictionary = _remote_snapshots.back()
		var last_tick: float = float(last["t"])
		if tick <= last_tick:
			if last_tick - tick > 5000.0:
				# 发送方时钟大幅倒退（对方重启客户端），旧缓冲与偏移全部作废。
				_remote_snapshots.clear()
				_remote_clock_offset_valid = false
			else:
				# 同一采样的重复推送或乱序旧包，直接忽略。
				return

	# 时钟偏移取“到达最快”的样本，慢包因此落在插值延迟窗口内；同时缓慢上调以适应延迟整体变大。
	var offset_sample: float = now_ms - tick
	if not _remote_clock_offset_valid or offset_sample < _remote_clock_offset_ms:
		_remote_clock_offset_ms = offset_sample
		_remote_clock_offset_valid = true
	else:
		_remote_clock_offset_ms += (offset_sample - _remote_clock_offset_ms) * REMOTE_CLOCK_OFFSET_RISE_RATE

	var sample_before: Dictionary = _sample_remote_presentation(_remote_render_time_ms())
	if not _remote_snapshots.is_empty():
		var previous: Dictionary = _remote_snapshots.back()
		var elapsed_seconds: float = maxf(tick - float(previous["t"]), 0.0) / 1000.0
		var reachable_px: float = move_speed * elapsed_seconds * 1.5 + REMOTE_SNAP_DISTANCE_PX
		if (previous["pos"] as Vector2).distance_to(local_position) > reachable_px:
			# 两个快照之间的距离超出正常行走能力（传送或重同步），前一段按静止处理，时间线到达后直接瞬移。
			previous["moving"] = false
	_remote_snapshots.append({"t": tick, "pos": local_position, "facing": facing, "moving": moving})
	while _remote_snapshots.size() > REMOTE_SNAPSHOT_BUFFER_LIMIT:
		_remote_snapshots.pop_front()
	var sample_after: Dictionary = _sample_remote_presentation(_remote_render_time_ms())
	# 正常情况下新快照位于渲染时间之后，插值结果不变；只有缓冲耗尽外推后才会产生突变，此时转成可衰减的视觉误差。
	var jump: Vector2 = (sample_before["pos"] as Vector2) - (sample_after["pos"] as Vector2)
	_remote_render_error += jump
	if _remote_render_error.length() > REMOTE_SNAP_DISTANCE_PX:
		_remote_render_error = Vector2.ZERO


## 返回当前帧应该渲染的发送方时间点（毫秒）。
func _remote_render_time_ms() -> float:
	return float(Time.get_ticks_msec()) - _remote_clock_offset_ms - REMOTE_INTERPOLATION_DELAY_MS


## 在快照缓冲中按发送方时间采样远端表现。
## 返回 { "pos": Vector2, "facing": Vector2, "walking": bool }。
func _sample_remote_presentation(render_time_ms: float) -> Dictionary:
	if _remote_snapshots.is_empty():
		return {"pos": position - _remote_render_error, "facing": cardinal_direction, "walking": false}
	var first: Dictionary = _remote_snapshots[0]
	if render_time_ms <= float(first["t"]):
		# 渲染时间早于最早快照：停在最早快照，等待时间线推进。
		return {"pos": first["pos"], "facing": first["facing"], "walking": false}
	for index in range(_remote_snapshots.size() - 1):
		var a: Dictionary = _remote_snapshots[index]
		var b: Dictionary = _remote_snapshots[index + 1]
		var a_t: float = float(a["t"])
		var b_t: float = float(b["t"])
		if render_time_ms > b_t:
			continue
		if not bool(a["moving"]):
			# 发送方在 a 时刻已经停下，直到 b 时刻才重新起步；中间保持静止而不是慢慢滑过去。
			return {"pos": a["pos"], "facing": a["facing"], "walking": false}
		var span: float = maxf(b_t - a_t, 1.0)
		var alpha: float = clampf((render_time_ms - a_t) / span, 0.0, 1.0)
		var a_pos: Vector2 = a["pos"]
		var b_pos: Vector2 = b["pos"]
		return {"pos": a_pos.lerp(b_pos, alpha), "facing": a["facing"], "walking": not a_pos.is_equal_approx(b_pos)}
	# 渲染时间已越过最新快照（缓冲耗尽）：移动中则沿最后朝向短暂外推，静止则停在原地。
	var last: Dictionary = _remote_snapshots.back()
	var last_pos: Vector2 = last["pos"]
	if not bool(last["moving"]):
		return {"pos": last_pos, "facing": last["facing"], "walking": false}
	var overshoot_ms: float = render_time_ms - float(last["t"])
	var clamped_ms: float = minf(overshoot_ms, REMOTE_MAX_EXTRAPOLATION_MS)
	var facing: Vector2 = last["facing"]
	return {
		"pos": last_pos + facing * move_speed * clamped_ms / 1000.0,
		"facing": facing,
		"walking": overshoot_ms < REMOTE_MAX_EXTRAPOLATION_MS,
	}


## 每帧按插值时间线更新远端角色位置、朝向和行走动画。
## delta 是当前渲染帧耗时秒数，用于衰减新包带来的视觉误差。
func _update_remote_avatar(delta: float) -> void:
	if not _remote_target_initialized:
		return
	var render_time_ms: float = _remote_render_time_ms()
	# 丢弃渲染时间之前、已经不再需要的旧快照（保留一个作为插值起点）。
	while _remote_snapshots.size() >= 2 and float(_remote_snapshots[1]["t"]) <= render_time_ms:
		_remote_snapshots.pop_front()
	var sample: Dictionary = _sample_remote_presentation(render_time_ms)
	_remote_render_error *= exp(-REMOTE_ERROR_DECAY_PER_SECOND * delta)
	if _remote_render_error.length() < 0.05:
		_remote_render_error = Vector2.ZERO
	# 不在逻辑层 round()：逐帧取整会把连续速度重新变成走一像素、停一帧的阶梯运动。
	position = (sample["pos"] as Vector2) + _remote_render_error
	var walking: bool = bool(sample["walking"])
	var facing: Vector2 = sample["facing"]
	if facing != Vector2.ZERO:
		cardinal_direction = facing
	direction = facing if walking else Vector2.ZERO
	_set_direction()
	_update_state()
	_update_animation()

func set_facing_direction(facing_direction: Vector2) -> void:
	var resolved_direction := _resolve_cardinal_direction(facing_direction)
	if resolved_direction == Vector2.ZERO:
		return

	cardinal_direction = resolved_direction
	_clear_auto_move_path()
	velocity = Vector2.ZERO
	direction = Vector2.ZERO
	if _update_state():
		_update_animation()
	else:
		_update_animation()

func set_camera_limits(left: int, top: int, right: int, bottom: int) -> void:
	if camera_node != null:
		camera_node.limit_left = left
		camera_node.limit_top = top
		camera_node.limit_right = right
		camera_node.limit_bottom = bottom

func _apply_camera_zoom() -> void:
	if camera_node == null:
		return
	var zoom_scale: float = maxf(camera_zoom_scale, 0.1)
	camera_node.zoom = Vector2(zoom_scale, zoom_scale)

func _apply_camera_offset() -> void:
	if camera_node == null:
		return
	camera_node.offset = Vector2(0.0, camera_vertical_offset)

## 将相机中心吸附到缩放后的屏幕整数像素，同时保留人物自身的连续浮点移动。
func _sync_pixel_camera_position() -> void:
	if camera_node == null or is_remote_avatar:
		return
	var zoom_x: float = maxf(absf(camera_node.zoom.x), 0.1)
	var zoom_y: float = maxf(absf(camera_node.zoom.y), 0.1)
	var desired_global_position: Vector2 = to_global(_camera_follow_local_position)
	camera_node.global_position = Vector2(
		roundf(desired_global_position.x * zoom_x) / zoom_x,
		roundf(desired_global_position.y * zoom_y) / zoom_y
	)
	camera_node.force_update_scroll()

# 设置切图期间的移动锁定状态。
func set_scene_transition_locked(locked: bool) -> void:
	_scene_transition_locked = locked
	if locked:
		_clear_auto_move_path()
		velocity = Vector2.ZERO
		direction = Vector2.ZERO
		if _update_state():
			_update_animation()

# 设置战斗期间的移动锁定状态，并切换到战斗专用形象。
func set_battle_active(active: bool) -> void:
	_battle_locked = active
	if active:
		_clear_auto_move_path()
		velocity = Vector2.ZERO
		direction = Vector2.ZERO
		_apply_world_battle_skin()
	else:
		_restore_normal_skin()
	_update_state()
	_update_animation()

# 写入一条新的自动寻路路径，路径点使用玩家父节点本地坐标。
func set_auto_move_path(path_points: Array[Vector2]) -> void:
	_clear_auto_move_path()
	_auto_move_generation += 1
	var generation: int = _auto_move_generation
	for path_point in path_points:
		# 跳过距离当前点过近的路径点，避免角色在首点原地抖动。
		if position.distance_to(path_point) <= _auto_move_stop_tolerance:
			continue
		_auto_move_path.append(path_point)
	_auto_move_active = true
	if _auto_move_path.is_empty():
		call_deferred("_complete_auto_move", generation)

# 主动清空当前自动寻路状态，供世界控制器或锁定逻辑调用。
func clear_auto_move_path() -> void:
	_clear_auto_move_path()
	direction = Vector2.ZERO
	velocity = Vector2.ZERO
	if _update_state():
		_update_animation()

## 返回当前是否仍在执行一条等待自然完成的自动路径。
func is_auto_move_active() -> bool:
	return _auto_move_active

## 进入剧情控制：保留原场景锁，临时允许脚本路径移动并屏蔽玩家方向输入。
func begin_cinematic_control() -> void:
	_cinematic_previous_scene_lock = _scene_transition_locked
	_scene_transition_locked = false
	_cinematic_input_locked = true
	_clear_auto_move_path()
	direction = Vector2.ZERO
	velocity = Vector2.ZERO
	_update_state()
	_update_animation()

## 结束剧情控制并恢复进入前的锁定状态与常规世界动画。
func end_cinematic_control() -> void:
	_clear_auto_move_path()
	_cinematic_input_locked = false
	_scene_transition_locked = _cinematic_previous_scene_lock
	direction = Vector2.ZERO
	velocity = Vector2.ZERO
	_update_state()
	_update_animation()

## 播放剧情指定动作；frame_index 大于等于 0 时暂停在对应帧。
func play_cinematic_animation(animation_name: String, frame_index: int = -1, frame_fps: float = 12.0) -> bool:
	var normalized_name: String = animation_name.strip_edges()
	if normalized_name.is_empty():
		return false
	if _uses_character_visual and _character_visual != null:
		return _character_visual.play_cinematic_pose(normalized_name, frame_index)
	if animation_player == null or not animation_player.has_animation(normalized_name):
		return false
	animation_player.play(normalized_name)
	if frame_index >= 0:
		var safe_fps: float = maxf(frame_fps, 1.0)
		animation_player.seek(float(frame_index) / safe_fps, true)
		animation_player.pause()
	return true

## 恢复角色当前状态和朝向对应的世界动画。
func restore_world_animation() -> void:
	_update_animation()

# 刷新角色状态机，并在状态变化时返回 `true`。
func _update_state() -> bool:
	# 解析当前输入和锁定状态下应进入的新状态。
	var new_state := _resolve_state()
	if new_state == state:
		return false
	state = new_state
	return true

# 根据当前移动与锁定状态解析角色状态机结果。
func _resolve_state() -> String:
	if _battle_locked:
		return STATE_BATTLE
	if direction == Vector2.ZERO:
		return STATE_IDLE
	return STATE_WALK

# 按当前状态与朝向播放对应动画。
func _update_animation() -> void:
	if _uses_character_visual and _character_visual != null:
		_character_visual.play_world(state, _direction_suffix())
		return
	if animation_player == null:
		return

	if state == STATE_BATTLE:
		if animation_player.has_animation(LEGACY_BATTLE_ANIMATION):
			animation_player.play(LEGACY_BATTLE_ANIMATION)
			return
		var battle_direction_animation: String = STATE_BATTLE + "_" + _direction_suffix()
		if animation_player.has_animation(battle_direction_animation):
			animation_player.play(battle_direction_animation)
			return

	# 组合当前状态和朝向后缀得到目标动画名。
	var animation_name := state + "_" + _direction_suffix()
	if animation_player.has_animation(animation_name):
		animation_player.play(animation_name)
	elif animation_player.has_animation(state):
		animation_player.play(state)

# 按当前移动向量刷新角色朝向，并在朝向变化时返回 `true`。
func _set_direction() -> bool:
	if direction == Vector2.ZERO:
		return false
	var new_dir := _resolve_cardinal_direction(direction)

	if new_dir == cardinal_direction:
		return false

	cardinal_direction = new_dir
	return true

func _resolve_cardinal_direction(input_direction: Vector2) -> Vector2:
	if input_direction == Vector2.ZERO:
		return Vector2.ZERO
	if abs(input_direction.x) >= abs(input_direction.y):
		return Vector2.LEFT if input_direction.x < 0.0 else Vector2.RIGHT
	return Vector2.UP if input_direction.y < 0.0 else Vector2.DOWN

# 把当前四方向向量转换为动画资源使用的朝向后缀。
func _direction_suffix() -> String:
	if cardinal_direction == Vector2.UP:
		return "up"
	if cardinal_direction == Vector2.DOWN:
		return "down"
	if cardinal_direction == Vector2.LEFT:
		return "left"
	return "right"

# 判断当前角色是否处于不可移动状态。
func _is_movement_locked() -> bool:
	return _scene_transition_locked or _battle_locked

func _setup_character_visual() -> void:
	if _character_visual != null:
		return
	_character_visual = CharacterVisualScene.instantiate() as CharacterVisual
	if _character_visual == null:
		return
	# 新方案约定：Player 根节点即脚底锚点，形象向上绘制，碰撞圆贴在脚下。
	_character_visual.position = Vector2.ZERO
	add_child(_character_visual)
	_character_visual.visible = false

func _on_world_snapshot_changed() -> void:
	# 战斗形象覆盖期间忽略快照刷新，避免把战斗待机形象提前切回日常皮肤。
	if _battle_skin_override_active:
		return
	_sync_skin_from_snapshot()

## 进入战斗时把世界玩家形象替换为战斗待机专用皮肤，并播放其待机动画。
func _apply_world_battle_skin() -> void:
	_setup_character_visual()
	if _character_visual == null:
		return
	if _character_visual.apply_skin_id(WORLD_BATTLE_SKIN_ID):
		_battle_skin_override_active = true
		_uses_character_visual = true
		_character_visual.visible = true
		if legacy_sprite != null:
			legacy_sprite.visible = false
		if animation_player != null:
			animation_player.stop()
		_sync_collision_anchor()
		_update_animation()
		return
	push_warning("找不到世界战斗待机形象: %s" % WORLD_BATTLE_SKIN_ID)

## 退出战斗后恢复服务端下发的正常 skin_id 形象。
func _restore_normal_skin() -> void:
	if not _battle_skin_override_active:
		return
	_battle_skin_override_active = false
	_sync_skin_from_snapshot()

func _sync_skin_from_snapshot() -> void:
	var skin_id: String = str(GameState.player_snapshot.get("skin_id", ""))
	_apply_normal_skin_id(skin_id)


## 应用普通世界人物形象；本地玩家和远端玩家都通过该入口消费服务端 skin_id。
## skin_id 为空或客户端资源尚未注册时，保留 player.tscn 的旧精灵作为兼容展示。
func _apply_normal_skin_id(skin_id: String) -> void:
	_setup_character_visual()
	if _character_visual == null:
		if legacy_sprite != null:
			legacy_sprite.visible = true
		return
	var normalized_skin_id: String = skin_id.strip_edges()
	if normalized_skin_id.is_empty():
		_uses_character_visual = false
		_character_visual.visible = false
		if legacy_sprite != null:
			legacy_sprite.visible = true
		_sync_collision_anchor()
		_update_animation()
		return
	if _character_visual.apply_skin_id(normalized_skin_id):
		_uses_character_visual = true
		_character_visual.visible = true
		if legacy_sprite != null:
			legacy_sprite.visible = false
		if animation_player != null:
			animation_player.stop()
		_sync_collision_anchor()
		_update_animation()
		return
	_uses_character_visual = false
	_character_visual.visible = false
	if legacy_sprite != null:
		legacy_sprite.visible = true
	_sync_collision_anchor()
	_update_animation()

func _sync_collision_anchor() -> void:
	var collision_offset: Vector2 = LEGACY_COLLISION_OFFSET
	if _uses_character_visual and _character_visual != null:
		collision_offset = _character_visual.position + _character_visual.get_feet_local_position()
		collision_offset += _character_visual.get_world_collision_offset()
	if body_collision_shape != null:
		body_collision_shape.position = collision_offset
	if click_collision_shape != null:
		click_collision_shape.position = collision_offset

# 沿当前自动寻路路径计算本帧移动方向，并在到达节点后切换到下一个节点。
func _update_auto_move_direction() -> void:
	while not _auto_move_path.is_empty():
		var next_path_point: Vector2 = _auto_move_path[0]
		var to_next_point: Vector2 = next_path_point - position
		if to_next_point.length() <= _auto_move_stop_tolerance:
			_auto_move_path.remove_at(0)
			continue

		# 自动寻路同样保持四方向移动，优先沿距离更大的轴推进。
		if abs(to_next_point.x) >= abs(to_next_point.y):
			direction = Vector2.LEFT if to_next_point.x < 0.0 else Vector2.RIGHT
		else:
			direction = Vector2.UP if to_next_point.y < 0.0 else Vector2.DOWN
		return

	direction = Vector2.ZERO
	if _auto_move_active:
		_complete_auto_move(_auto_move_generation)

# 内部统一使用的自动寻路清理入口。
func _clear_auto_move_path() -> void:
	_auto_move_generation += 1
	_auto_move_active = false
	_auto_move_path.clear()

## 仅在当前代次路径自然走完时发出完成信号。
func _complete_auto_move(generation: int) -> void:
	if generation != _auto_move_generation or not _auto_move_active:
		return
	_auto_move_active = false
	auto_move_finished.emit()
