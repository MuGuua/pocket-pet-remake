extends Node

signal session_changed
signal world_snapshot_changed
signal pets_changed
signal bag_changed
signal battle_changed

var access_jwt: String = ""
var ws_token: String = ""
var ws_expire_at: int = 0
var session_id: String = ""
var reconnect_token: String = ""
var heartbeat_sec: int = 0
var is_ws_authenticated: bool = false
var player_id: int = 0
var player_snapshot: Dictionary = {}
var scene_snapshot: Dictionary = {}
var nearby_entities: Dictionary = {}
var pets: Array = []
var lineup: Array = []
var bag_items: Array = []
var battle_state: Dictionary = {}
var is_in_battle: bool = false

func reset_session_state() -> void:
    access_jwt = ""
    ws_token = ""
    ws_expire_at = 0
    session_id = ""
    reconnect_token = ""
    heartbeat_sec = 0
    is_ws_authenticated = false
    player_id = 0
    reset_runtime_state()
    session_changed.emit()

func reset_runtime_state() -> void:
    player_snapshot = {}
    scene_snapshot = {}
    nearby_entities = {}
    pets = []
    lineup = []
    bag_items = []
    battle_state = {}
    is_in_battle = false
    world_snapshot_changed.emit()
    pets_changed.emit()
    bag_changed.emit()
    battle_changed.emit()

func store_login_result(data: Dictionary) -> void:
    player_id = int(data.get("player_id", 0))
    access_jwt = str(data.get("access_jwt", ""))
    ws_token = str(data.get("ws_token", ""))
    ws_expire_at = int(data.get("ws_expire_at", 0))
    is_ws_authenticated = false

    var player_name := str(data.get("player_name", ""))
    player_snapshot = {
        "player_id": player_id,
        "name": player_name,
    }
    session_changed.emit()

func store_ws_session(data: Dictionary) -> void:
    session_id = str(data.get("session_id", ""))
    reconnect_token = str(data.get("reconnect_token", ""))
    heartbeat_sec = int(data.get("heartbeat_sec", 0))
    is_ws_authenticated = true
    session_changed.emit()

func set_ws_authenticated(authenticated: bool) -> void:
    is_ws_authenticated = authenticated
    if not authenticated:
        session_id = ""
        reconnect_token = ""
        heartbeat_sec = 0
    session_changed.emit()

func set_world_snapshot(payload: Dictionary) -> void:
    var scene_data: Variant = payload.get("scene", {})
    scene_snapshot = scene_data.duplicate(true) if scene_data is Dictionary else {}
    if payload.has("scene_id"):
        scene_snapshot["scene_id"] = payload.get("scene_id")
    if payload.has("scene_version"):
        scene_snapshot["scene_version"] = payload.get("scene_version")

    var next_player := player_snapshot.duplicate(true)
    var player_data: Variant = payload.get("player", payload.get("self", {}))
    if player_data is Dictionary:
        next_player.merge(player_data, true)
    if player_id > 0 and not player_snapshot.has("player_id"):
        next_player["player_id"] = player_id
    if payload.has("self_pos"):
        var self_pos_variant: Variant = payload.get("self_pos", {})
        if self_pos_variant is Dictionary:
            next_player["x"] = float(self_pos_variant.get("x", next_player.get("x", 0.0)))
            next_player["y"] = float(self_pos_variant.get("y", next_player.get("y", 0.0)))
    player_snapshot = next_player

    nearby_entities = {}
    var entities_variant: Variant = payload.get("entities", payload.get("nearby_entities", []))
    if entities_variant is Array:
        for entity_variant in entities_variant:
            if entity_variant is Dictionary and entity_variant.has("entity_id"):
                nearby_entities[int(entity_variant["entity_id"])] = entity_variant.duplicate(true)

    var lineup_variant: Variant = payload.get("lineup", [])
    lineup = lineup_variant.duplicate(true) if lineup_variant is Array else []
    world_snapshot_changed.emit()
    pets_changed.emit()

func add_entity(entity: Dictionary) -> void:
    if not entity.has("entity_id"):
        return

    nearby_entities[int(entity["entity_id"])] = entity.duplicate(true)
    world_snapshot_changed.emit()

func remove_entity(entity_id: int) -> void:
    nearby_entities.erase(entity_id)
    world_snapshot_changed.emit()

func apply_entity_move(payload: Dictionary) -> void:
    var entity_id: int = int(payload.get("entity_id", 0))
    if entity_id == 0:
        return

    var entity: Dictionary = nearby_entities.get(entity_id, {})
    var position_variant: Variant = payload.get("to_pos", payload.get("position", {}))
    if position_variant is Dictionary:
        entity["pos"] = position_variant.duplicate(true)
        entity["x"] = float(position_variant.get("x", entity.get("x", 0.0)))
        entity["y"] = float(position_variant.get("y", entity.get("y", 0.0)))
    else:
        entity["x"] = float(payload.get("x", entity.get("x", 0.0)))
        entity["y"] = float(payload.get("y", entity.get("y", 0.0)))
    nearby_entities[entity_id] = entity

    if entity_id == player_id:
        player_snapshot["x"] = entity["x"]
        player_snapshot["y"] = entity["y"]

    world_snapshot_changed.emit()

func set_pets(next_pets: Array, next_lineup: Array = []) -> void:
    pets = next_pets.duplicate(true)
    lineup = next_lineup.duplicate(true)
    pets_changed.emit()

func set_lineup(next_lineup: Array) -> void:
    lineup = next_lineup.duplicate(true)
    pets_changed.emit()

func upsert_pet(pet: Dictionary) -> void:
    var pet_id: int = int(pet.get("pet_id", 0))
    if pet_id == 0:
        return

    for index in pets.size():
        var current: Variant = pets[index]
        if current is Dictionary and int(current.get("pet_id", 0)) == pet_id:
            pets[index] = pet.duplicate(true)
            pets_changed.emit()
            return

    pets.append(pet.duplicate(true))
    pets_changed.emit()

func set_bag_items(next_items: Array) -> void:
    bag_items = next_items.duplicate(true)
    bag_changed.emit()

func upsert_bag_item(item: Dictionary) -> void:
    var item_id: int = int(item.get("item_id", 0))
    if item_id == 0:
        return

    for index in bag_items.size():
        var current: Variant = bag_items[index]
        if current is Dictionary and int(current.get("item_id", 0)) == item_id:
            bag_items[index] = item.duplicate(true)
            bag_changed.emit()
            return

    bag_items.append(item.duplicate(true))
    bag_changed.emit()

func set_battle_state(next_state: Dictionary, active: bool = true) -> void:
    var merged_state: Dictionary = battle_state.duplicate(true) if active else {}
    merged_state.merge(next_state, true)
    if next_state.has("actors") and not merged_state.has("actors"):
        merged_state["actors"] = []
    battle_state = merged_state
    is_in_battle = active
    battle_changed.emit()

func clear_battle_state() -> void:
    battle_state = {}
    is_in_battle = false
    battle_changed.emit()
