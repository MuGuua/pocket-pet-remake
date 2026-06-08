class_name InteractiveNPCBase
extends Node2D

signal interaction_entered(entity_id: int, npc_name: String)
signal interaction_exited(entity_id: int, npc_name: String)

@export var entity_id: int = 0
@export var npc_name: String = "NPC"

@onready var interact_area: Area2D = $Area2D

func _ready() -> void:
    if interact_area == null:
        return
    interact_area.monitoring = true
    interact_area.monitorable = true
    for layer_index in range(1, 33):
        interact_area.set_collision_mask_value(layer_index, true)
    if not interact_area.body_entered.is_connected(_on_body_entered):
        interact_area.body_entered.connect(_on_body_entered)
    if not interact_area.body_exited.is_connected(_on_body_exited):
        interact_area.body_exited.connect(_on_body_exited)

func _on_body_entered(body: Node2D) -> void:
    if body == null or body.name != "Player" or entity_id <= 0:
        return
    interaction_entered.emit(entity_id, npc_name)

func _on_body_exited(body: Node2D) -> void:
    if body == null or body.name != "Player" or entity_id <= 0:
        return
    interaction_exited.emit(entity_id, npc_name)
