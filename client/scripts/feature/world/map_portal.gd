class_name MapPortal
extends Area2D

signal activated(portal_id: int, target_scene_id: int)

@export var portal_id: int = 0
@export var target_scene_id: int = 0

func _ready() -> void:
	body_entered.connect(_on_body_entered)

func _on_body_entered(body: Node) -> void:
	if portal_id <= 0 or target_scene_id <= 0:
		return
	if body == null or body.name != "player":
		return
	activated.emit(portal_id, target_scene_id)
