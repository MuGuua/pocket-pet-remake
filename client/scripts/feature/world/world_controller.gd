extends Node2D

signal scene_loaded(scene_id: String)
signal player_position_changed(position: Vector2)

@onready var local_player_anchor: Node2D = %LocalPlayerAnchor
@onready var remote_entities_root: Node2D = %RemoteEntities

func handle_enter_world(payload: Dictionary) -> void:
    GameState.set_world_snapshot(payload)
    _sync_local_player_from_state()
    scene_loaded.emit(str(GameState.scene_snapshot.get("scene_id", "unknown")))

func handle_entity_enter(payload: Dictionary) -> void:
    var entity_variant: Variant = payload.get("entity", payload)
    var entity: Dictionary = entity_variant if entity_variant is Dictionary else {}
    GameState.add_entity(entity)

func handle_entity_leave(payload: Dictionary) -> void:
    GameState.remove_entity(int(payload.get("entity_id", 0)))

func handle_entity_move(payload: Dictionary) -> void:
    GameState.apply_entity_move(payload)
    _sync_local_player_from_state()

func handle_world_resync(payload: Dictionary) -> void:
    GameState.set_world_snapshot(payload)
    _sync_local_player_from_state()

func request_move(target: Vector2) -> void:
    NetClient.send_command(
        CommandIds.MOVE_INTENT_REQ,
        {
            "x": target.x,
            "y": target.y,
        }
    )

func _sync_local_player_from_state() -> void:
    var fallback := local_player_anchor.position
    var position := _extract_player_position(GameState.player_snapshot, fallback)
    local_player_anchor.position = position
    player_position_changed.emit(position)

func _extract_player_position(player: Dictionary, fallback: Vector2) -> Vector2:
    var x := fallback.x
    var y := fallback.y

    if player.has("x"):
        x = float(player.get("x", x))
    if player.has("y"):
        y = float(player.get("y", y))

    var position_variant: Variant = player.get("position", {})
    if position_variant is Dictionary:
        x = float(position_variant.get("x", x))
        y = float(position_variant.get("y", y))

    return Vector2(x, y)
