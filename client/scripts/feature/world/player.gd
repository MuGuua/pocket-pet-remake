class_name player
extends CharacterBody2D

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

# 本地角色移动速度。
@export var move_speed: float = 100.0
# 世界相机统一缩放；小于 1 会放大画面。
@export var camera_zoom_scale: float = 2.0
# 世界相机相对玩家的垂直偏移；负值表示画面整体上移。
@export var camera_vertical_offset: float = 150.0

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

func _ready() -> void:
	if camera_node != null:
		camera_node.make_current()
	_apply_camera_zoom()
	_apply_camera_offset()
	_setup_character_visual()
	if not GameState.world_snapshot_changed.is_connected(_on_world_snapshot_changed):
		GameState.world_snapshot_changed.connect(_on_world_snapshot_changed)
	_sync_skin_from_snapshot()

func _exit_tree() -> void:
	if GameState.world_snapshot_changed.is_connected(_on_world_snapshot_changed):
		GameState.world_snapshot_changed.disconnect(_on_world_snapshot_changed)

# 每帧读取输入并更新角色状态与动画。
func _process(_delta: float) -> void:
	if _is_movement_locked():
		# 锁定移动时同时清空自动寻路，避免切图或战斗结束后继续沿旧路径前进。
		_clear_auto_move_path()
		direction = Vector2.ZERO
		velocity = Vector2.ZERO
		if _update_state():
			_update_animation()
		return

	# 读取水平输入并计算横向移动分量。
	direction.x = Input.get_action_strength("ui_right") - Input.get_action_strength("ui_left")
	# 读取垂直输入并计算纵向移动分量。
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
	move_and_slide()
	# 像素风世界要求角色始终落在整数像素上，避免相机跟随时把整张地图带成半像素采样。
	position = position.round()

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
	if _update_state():
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
	_auto_move_path.clear()
	for path_point in path_points:
		# 跳过距离当前点过近的路径点，避免角色在首点原地抖动。
		if position.distance_to(path_point) <= _auto_move_stop_tolerance:
			continue
		_auto_move_path.append(path_point)

# 主动清空当前自动寻路状态，供世界控制器或锁定逻辑调用。
func clear_auto_move_path() -> void:
	_clear_auto_move_path()
	direction = Vector2.ZERO
	velocity = Vector2.ZERO
	if _update_state():
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
	_setup_character_visual()
	if _character_visual == null:
		if legacy_sprite != null:
			legacy_sprite.visible = true
		return
	var skin_id: String = str(GameState.player_snapshot.get("skin_id", ""))
	if skin_id.is_empty():
		_uses_character_visual = false
		_character_visual.visible = false
		if legacy_sprite != null:
			legacy_sprite.visible = true
		_sync_collision_anchor()
		_update_animation()
		return
	if _character_visual.apply_skin_id(skin_id):
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

# 内部统一使用的自动寻路清理入口。
func _clear_auto_move_path() -> void:
	_auto_move_path.clear()
