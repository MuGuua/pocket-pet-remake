extends Node

signal pets_updated(count: int)

func handle_pet_list(payload: Dictionary) -> void:
    var pets_variant: Variant = payload.get("pets", [])
    var lineup_variant: Variant = payload.get("lineup", [])
    var pets: Array = pets_variant if pets_variant is Array else []
    var lineup: Array = lineup_variant if lineup_variant is Array else []
    GameState.set_pets(pets, lineup)
    pets_updated.emit(GameState.pets.size())

func handle_pet_update(payload: Dictionary) -> void:
    var pet_variant: Variant = payload.get("pet", payload)
    var pet: Dictionary = pet_variant if pet_variant is Dictionary else {}
    GameState.upsert_pet(pet)
    pets_updated.emit(GameState.pets.size())

func handle_lineup_set_response(payload: Dictionary) -> void:
    if not bool(payload.get("accepted", false)):
        return
    var lineup_variant: Variant = payload.get("lineup", [])
    var lineup: Array = lineup_variant if lineup_variant is Array else []
    GameState.set_lineup(lineup)
    pets_updated.emit(GameState.pets.size())
