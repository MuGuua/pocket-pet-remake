class_name player
extends CharacterBody2D

# 角色待机状态标识。
const STATE_IDLE := "idle"
# 角色行走状态标识。
const STATE_WALK := "walk"
# 角色战斗锁定状态标识。
const STATE_BATTLE := "battle"

# 本地角色移动速度。
@export var move_speed: float = 100.0

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

# 负责播放角色动画的动画播放器节点。
@onready var animation_player: AnimationPlayer = $AnimationPlayer
@onready var camera_node: Camera2D = $Camera2D

# 每帧读取输入并更新角色状态与动画。
func _process(_delta: float) -> void:
	if _is_movement_locked():
		direction = Vector2.ZERO
		velocity = Vector2.ZERO
		if _update_state():
			_update_animation()
		return

	# 读取水平输入并计算横向移动分量。
	direction.x = Input.get_action_strength("ui_right") - Input.get_action_strength("ui_left")
	# 读取垂直输入并计算纵向移动分量。
	direction.y = Input.get_action_strength("ui_down") - Input.get_action_strength("ui_up")
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

# 把服务端权威坐标直接应用到当前角色显示位置。
func apply_authoritative_position(local_position: Vector2) -> void:
	position = local_position
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

# 设置切图期间的移动锁定状态。
func set_scene_transition_locked(locked: bool) -> void:
	_scene_transition_locked = locked
	if locked:
		velocity = Vector2.ZERO
		direction = Vector2.ZERO
		if _update_state():
			_update_animation()

# 设置战斗期间的移动锁定状态。
func set_battle_active(active: bool) -> void:
	_battle_locked = active
	if active:
		velocity = Vector2.ZERO
		direction = Vector2.ZERO
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
	if animation_player == null:
		return

	# 组合当前状态和朝向后缀得到目标动画名。
	var animation_name := state + "_" + _direction_suffix()
	if animation_player.has_animation(animation_name):
		animation_player.play(animation_name)
	elif state == STATE_BATTLE:
		# 战斗态没有独立动画时回退到同朝向待机动画。
		var fallback_animation := STATE_IDLE + "_" + _direction_suffix()
		if animation_player.has_animation(fallback_animation):
			animation_player.play(fallback_animation)
		elif animation_player.has_animation(STATE_IDLE):
			animation_player.play(STATE_IDLE)
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
