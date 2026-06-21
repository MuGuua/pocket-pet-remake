extends Node2D
class_name WorldPetFollower

## 世界宠物跟随节点；position 为脚底锚点，与主角一致，供 ActorRoot Y-Sort 遮挡排序。
const CharacterVisualScene: PackedScene = preload("res://scenes/character/character_visual.tscn")

## 世界场景 idle 状态标识。
const STATE_IDLE: String = "idle"
## 世界场景 walk 状态标识。
const STATE_WALK: String = "walk"

## 路径跟随逻辑实例。
var _path_controller: PathFollowController = PathFollowController.new()
## 当前绑定的出战宠物唯一标识。
var _bound_pet_uid: int = 0
## 当前绑定的形象资源 ID。
var _bound_skin_id: String = ""
## 是否允许显示并更新跟随者。
var _follow_enabled: bool = false
## 角色视觉节点。
var _character_visual: CharacterVisual = null


func _ready() -> void:
	_setup_character_visual()


## 根据编队首只宠物同步绑定信息；无有效宠物或 skin_id 时隐藏跟随者。
func sync_lineup_pet(lineup_pet: Dictionary) -> void:
	var pet_uid: int = int(lineup_pet.get("pet_uid", 0))
	var skin_id: String = str(lineup_pet.get("skin_id", "")).strip_edges()
	if pet_uid <= 0 or skin_id.is_empty():
		_clear_binding()
		return
	if pet_uid == _bound_pet_uid and skin_id == _bound_skin_id:
		_follow_enabled = true
		visible = true
		return
	_bound_pet_uid = pet_uid
	_bound_skin_id = skin_id
	_follow_enabled = true
	visible = true
	_apply_skin(skin_id)


## 清空绑定并隐藏跟随者（战斗、无编队等场景）。
func clear_binding() -> void:
	_clear_binding()


## 在主角附近重置跟随位置并清空路径（切图、权威坐标同步时调用）。
func reset_near_leader(leader_position: Vector2, offset: Vector2 = Vector2(0.0, 12.0)) -> void:
	if not _follow_enabled:
		return
	_path_controller.reset_near_leader(leader_position, offset)
	position = _path_controller.position
	visible = true
	_update_animation()


## 记录主角刚离开的一个格点。
func push_leader_step(leader_from: Vector2, direction: Vector2) -> void:
	_path_controller.push_leader_step(leader_from, direction)


## 按帧更新跟随移动与动画。
func update_follow(delta: float, move_speed: float) -> void:
	if not _follow_enabled or not visible:
		return
	_path_controller.update(delta, move_speed)
	position = _path_controller.position
	_update_animation()


## 返回当前跟随者是否正在移动。
func is_moving() -> bool:
	return _path_controller.moving


## 返回当前四方向朝向。
func get_cardinal_direction() -> Vector2:
	return _path_controller.cardinal_direction


func _clear_binding() -> void:
	_bound_pet_uid = 0
	_bound_skin_id = ""
	_follow_enabled = false
	visible = false


func _setup_character_visual() -> void:
	if _character_visual != null:
		return
	_character_visual = CharacterVisualScene.instantiate() as CharacterVisual
	if _character_visual == null:
		return
	_character_visual.position = Vector2.ZERO
	add_child(_character_visual)


func _apply_skin(skin_id: String) -> void:
	_setup_character_visual()
	if _character_visual == null:
		return
	if not _character_visual.apply_skin_id(skin_id):
		push_warning("世界宠物跟随找不到形象: %s" % skin_id)
		_clear_binding()


func _update_animation() -> void:
	if _character_visual == null:
		return
	var state: String = STATE_WALK if _path_controller.moving else STATE_IDLE
	var direction_suffix: String = _direction_suffix(_path_controller.cardinal_direction)
	_character_visual.play_world(state, direction_suffix)


func _direction_suffix(direction: Vector2) -> String:
	if direction == Vector2.UP:
		return "up"
	if direction == Vector2.LEFT:
		return "left"
	if direction == Vector2.RIGHT:
		return "right"
	return "down"
