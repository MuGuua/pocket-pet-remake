extends CharacterBody2D

@export var speed: float = 300.0

func _ready() -> void:
	print("Player initialized at ", global_position)

func _physics_process(_delta: float) -> void:
	var direction := Input.get_vector("ui_left", "ui_right", "ui_up", "ui_down")
	if direction != Vector2.ZERO:
		velocity = direction * speed
		move_and_slide()
		
		# 请求服务器同步位置
		var world_controller = get_tree().root.find_child("WorldScene", true, false)
		if world_controller and world_controller.has_method("request_move"):
			world_controller.request_move(global_position)
	else:
		velocity = Vector2.ZERO
