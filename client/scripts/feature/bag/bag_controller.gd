extends Node

signal bag_updated(count: int)

func handle_bag_list(payload: Dictionary) -> void:
    var items_variant: Variant = payload.get("items", [])
    var items: Array = items_variant if items_variant is Array else []
    GameState.set_bag_items(items)
    bag_updated.emit(GameState.bag_items.size())

func handle_bag_update(payload: Dictionary) -> void:
    var item_variant: Variant = payload.get("item", payload)
    var item: Dictionary = item_variant if item_variant is Dictionary else {}
    GameState.upsert_bag_item(item)
    bag_updated.emit(GameState.bag_items.size())
