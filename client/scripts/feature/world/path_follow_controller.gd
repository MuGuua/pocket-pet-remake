extends RefCounted
class_name PathFollowController

## 路径跟随控制器：记录主角走过的格点路径，并按延迟与等速规则驱动跟随者移动。
## 参数口径参考 docs/pet_follow_logic.md，格步长与世界导航网格一致（24px）。

## 单格路径步长（像素），与世界 grid_to_pixels 保持一致。
const PATH_STEP_SIZE: float = 24.0
## 开始跟随所需的最小路径点数（2 格 ≈ 48px，比 demo 的 3 格更贴近角色）。
const MIN_PATH_TO_START: int = 2
## 已在移动中继续取下一步所需的最小路径点数。
const MIN_PATH_TO_CONTINUE: int = 1
## 首次开始跟随时累计的等待毫秒数。
const START_WAIT_MS: float = 100.0
## 路径队列最大长度，超出时丢弃最旧点。
const MAX_PATH_LENGTH: int = 24
## 跟随者停步且路径过长时触发跳点的阈值。
const PATH_OVERFLOW_THRESHOLD: int = 8
## 路径跳点处理后保留的最新路径点数。
const PATH_OVERFLOW_KEEP: int = 4

## 跟随者当前世界坐标（相对父节点）。
var position: Vector2 = Vector2.ZERO
## 当前一步的起点坐标。
var from_position: Vector2 = Vector2.ZERO
## 当前一步的终点坐标。
var target_position: Vector2 = Vector2.ZERO
## 当前一步的插值进度，范围 0..1。
var step_progress: float = 1.0
## 当前四方向朝向。
var cardinal_direction: Vector2 = Vector2.DOWN
## 是否正在移动一步。
var moving: bool = false
## 首次开始跟随前的等待累计（毫秒）。
var wait_ms: float = 0.0
## 主角走过的路径队列，每项为 { "position": Vector2, "direction": Vector2 }。
var path: Array[Dictionary] = []


## 把主角离开的一个格点压入路径队列。
func push_leader_step(leader_from: Vector2, direction: Vector2) -> void:
	if direction == Vector2.ZERO:
		return
	path.append({
		"position": leader_from,
		"direction": direction,
	})
	if path.size() > MAX_PATH_LENGTH:
		path.pop_front()


## 按帧推进跟随者移动；move_speed 与主角 move_speed 保持一致以实现等速跟随。
func update(delta: float, move_speed: float) -> void:
	if not moving:
		_try_start_follow_step(delta)
		_apply_path_overflow_guard()
		return

	var step_speed: float = 0.0
	if PATH_STEP_SIZE > 0.0 and move_speed > 0.0:
		step_speed = move_speed * delta / PATH_STEP_SIZE
	step_progress = minf(1.0, step_progress + step_speed)
	position = from_position.lerp(target_position, step_progress)
	moving = step_progress < 1.0

	if step_progress >= 1.0 and path.size() >= MIN_PATH_TO_CONTINUE:
		_begin_follow_step(MIN_PATH_TO_CONTINUE)
		moving = step_progress < 1.0


## 清空路径并把跟随者放到主角附近的偏移格上。
func reset_near_leader(leader_position: Vector2, offset: Vector2) -> void:
	position = leader_position + offset
	from_position = position
	target_position = position
	step_progress = 1.0
	cardinal_direction = Vector2.DOWN
	moving = false
	wait_ms = 0.0
	path.clear()


## 尝试在停步状态下启动第一步跟随。
func _try_start_follow_step(delta: float) -> void:
	if path.size() < MIN_PATH_TO_START:
		wait_ms = 0.0
		return
	wait_ms += delta * 1000.0
	if wait_ms < START_WAIT_MS:
		return
	wait_ms = START_WAIT_MS
	_begin_follow_step(MIN_PATH_TO_START)


## 从路径队列取下一个目标格并开始一步移动。
func _begin_follow_step(min_path: int) -> void:
	if step_progress < 1.0 or path.size() < min_path:
		return

	var next_entry: Dictionary = path.pop_front()
	var next_position: Vector2 = next_entry.get("position", position) as Vector2
	if position.distance_to(next_position) < 1.0:
		moving = false
		return

	var direction: Vector2 = _direction_between(position, next_position)
	if direction == Vector2.ZERO:
		moving = false
		return

	from_position = position
	target_position = next_position
	cardinal_direction = direction
	step_progress = 0.0
	moving = true


## 路径堆积过多且跟随者停步时，跳到较新的路径点避免长时间追赶。
func _apply_path_overflow_guard() -> void:
	if moving or path.size() <= PATH_OVERFLOW_THRESHOLD:
		return

	var remove_count: int = path.size() - PATH_OVERFLOW_KEEP
	if remove_count <= 0:
		return
	var skipped: Array[Dictionary] = []
	for index in range(remove_count):
		skipped.append(path[index])
	for index in range(remove_count):
		path.pop_front()
	if skipped.is_empty():
		return
	var anchor: Dictionary = skipped[skipped.size() - 1]
	var anchor_position: Vector2 = anchor.get("position", position) as Vector2
	position = anchor_position
	from_position = anchor_position
	target_position = anchor_position
	step_progress = 1.0
	moving = false


## 根据起点与终点计算四方向单位向量。
func _direction_between(from_position_value: Vector2, to_position_value: Vector2) -> Vector2:
	if to_position_value.x > from_position_value.x:
		return Vector2.RIGHT
	if to_position_value.x < from_position_value.x:
		return Vector2.LEFT
	if to_position_value.y > from_position_value.y:
		return Vector2.DOWN
	if to_position_value.y < from_position_value.y:
		return Vector2.UP
	return Vector2.ZERO
