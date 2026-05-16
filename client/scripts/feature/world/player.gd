class_name player
extends CharacterBody2D

const STATE_IDLE := "idle"
const STATE_WALK := "walk"
const STATE_BATTLE := "battle"

@export var move_speed: float = 100.0

var cardinal_direction: Vector2 = Vector2.DOWN
var direction: Vector2 = Vector2.ZERO
var state: String = STATE_IDLE
var _scene_transition_locked: bool = false
var _battle_locked: bool = false

@onready var animation_player: AnimationPlayer = $AnimationPlayer

func _process(_delta: float) -> void:
	if _is_movement_locked():
		direction = Vector2.ZERO
		velocity = Vector2.ZERO
		if _update_state():
			_update_animation()
		return

	direction.x = Input.get_action_strength("ui_right") - Input.get_action_strength("ui_left")
	direction.y = Input.get_action_strength("ui_down") - Input.get_action_strength("ui_up")
	if direction.x != 0.0 and direction.y != 0.0:
		if abs(direction.x) >= abs(direction.y):
			direction.y = 0.0
		else:
			direction.x = 0.0

	velocity = direction * move_speed

	if _update_state() or _set_direction():
		_update_animation()

func _physics_process(_delta: float) -> void:
	move_and_slide()

func apply_authoritative_position(local_position: Vector2) -> void:
	position = local_position
	velocity = Vector2.ZERO
	direction = Vector2.ZERO
	_scene_transition_locked = false
	if _update_state():
		_update_animation()

func set_scene_transition_locked(locked: bool) -> void:
	_scene_transition_locked = locked
	if locked:
		velocity = Vector2.ZERO
		direction = Vector2.ZERO
		if _update_state():
			_update_animation()

func set_battle_active(active: bool) -> void:
	_battle_locked = active
	if active:
		velocity = Vector2.ZERO
		direction = Vector2.ZERO
	if _update_state():
		_update_animation()

func _update_state() -> bool:
	var new_state := _resolve_state()
	if new_state == state:
		return false
	state = new_state
	return true

func _resolve_state() -> String:
	if _battle_locked:
		return STATE_BATTLE
	if direction == Vector2.ZERO:
		return STATE_IDLE
	return STATE_WALK

func _update_animation() -> void:
	if animation_player == null:
		return

	var animation_name := state + "_" + _direction_suffix()
	if animation_player.has_animation(animation_name):
		animation_player.play(animation_name)
	elif state == STATE_BATTLE:
		var fallback_animation := STATE_IDLE + "_" + _direction_suffix()
		if animation_player.has_animation(fallback_animation):
			animation_player.play(fallback_animation)
		elif animation_player.has_animation(STATE_IDLE):
			animation_player.play(STATE_IDLE)
	elif animation_player.has_animation(state):
		animation_player.play(state)

func _set_direction() -> bool:
	var new_dir: Vector2 = cardinal_direction
	if direction == Vector2.ZERO:
		return false

	if direction.y == 0:
		new_dir = Vector2.LEFT if direction.x < 0.0 else Vector2.RIGHT
	elif direction.x == 0:
		new_dir = Vector2.UP if direction.y < 0.0 else Vector2.DOWN

	if new_dir == cardinal_direction:
		return false

	cardinal_direction = new_dir
	return true

func _direction_suffix() -> String:
	if cardinal_direction == Vector2.UP:
		return "up"
	if cardinal_direction == Vector2.DOWN:
		return "down"
	if cardinal_direction == Vector2.LEFT:
		return "left"
	return "right"

func _is_movement_locked() -> bool:
	return _scene_transition_locked or _battle_locked
