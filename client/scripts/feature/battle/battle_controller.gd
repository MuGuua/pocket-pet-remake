extends Node

signal battle_updated(state: Dictionary)

func handle_battle_start(payload: Dictionary) -> void:
    GameState.set_battle_state(payload)
    battle_updated.emit(GameState.battle_state)

func handle_battle_state(payload: Dictionary) -> void:
    GameState.set_battle_state(payload)
    battle_updated.emit(GameState.battle_state)

func handle_battle_result(payload: Dictionary) -> void:
    GameState.set_battle_state(payload)
    battle_updated.emit(GameState.battle_state)
