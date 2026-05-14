extends Node

signal interact_responded(accepted: bool, reason: String)
signal action_responded(accepted: bool, reason: String)
signal battle_started(state: Dictionary)
signal battle_updated(state: Dictionary)
signal battle_finished(state: Dictionary)

func handle_interact_response(payload: Dictionary) -> void:
    interact_responded.emit(bool(payload.get("accepted", false)), str(payload.get("reason", "")))

func handle_battle_action_response(payload: Dictionary) -> void:
    action_responded.emit(bool(payload.get("accepted", false)), str(payload.get("reason", "")))

func handle_battle_start(payload: Dictionary) -> void:
    GameState.clear_battle_state()
    GameState.set_battle_state(payload, true)
    battle_started.emit(GameState.battle_state)
    battle_updated.emit(GameState.battle_state)

func handle_battle_state(payload: Dictionary) -> void:
    GameState.set_battle_state(payload, true)
    battle_updated.emit(GameState.battle_state)

func handle_battle_result(payload: Dictionary) -> void:
    GameState.set_battle_state(payload, false)
    battle_updated.emit(GameState.battle_state)
    battle_finished.emit(GameState.battle_state)
