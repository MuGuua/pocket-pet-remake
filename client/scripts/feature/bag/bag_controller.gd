extends Node

# 背包数据刷新后向外广播当前物品总数。
signal bag_updated(count: int)

# 处理背包列表响应，并把物品数组写入全局状态。
func handle_bag_list(payload: Dictionary) -> void:
    _apply_container_payload(payload)

# 处理单个背包物品更新推送，并把结果合并进全局状态。
func handle_bag_update(payload: Dictionary) -> void:
    var updates_variant: Variant = payload.get("updates", [])
    if updates_variant is Array:
        GameState.apply_container_updates(
            str(payload.get("container_type", "bag")),
            updates_variant,
            int(payload.get("capacity", 0)),
            int(payload.get("max_capacity", 0)),
            int(payload.get("used_slots", -1))
        )
    else:
        # 兼容 item 字段和直接物品结构两种旧载荷格式。
        var item_variant: Variant = payload.get("item", payload)
        var item: Dictionary = item_variant if item_variant is Dictionary else {}
        GameState.upsert_bag_item(item)
    bag_updated.emit(GameState.bag_items.size())

# 处理使用物品响应。
# 当前真正的容器变化仍以后续 `5011 BAG_UPDATE_PUSH` 为准，这里先保留入口方便 UI 后续接结果提示。
func handle_use_item_response(_payload: Dictionary) -> void:
    bag_updated.emit(GameState.bag_items.size())

# 处理仓库等其他容器的完整快照响应。
func handle_container_list(payload: Dictionary) -> void:
    _apply_container_payload(payload)

# 处理钱包独立查询响应。
func handle_wallet_query(payload: Dictionary) -> void:
    _apply_wallet_payload(payload)

# 处理钱包增量更新推送。
func handle_wallet_update(payload: Dictionary) -> void:
    _apply_wallet_payload(payload)

func _apply_container_payload(payload: Dictionary) -> void:
    var container_variant: Variant = payload.get("container", {})
    if container_variant is Dictionary and not container_variant.is_empty():
        GameState.set_container_snapshot(container_variant)
    else:
        # 兼容旧协议仍然直接返回 items 数组的情况。
        var items_variant: Variant = payload.get("items", [])
        var items: Array = items_variant if items_variant is Array else []
        GameState.set_bag_items(items)
    _apply_wallet_payload(payload)
    bag_updated.emit(GameState.bag_items.size())

func _apply_wallet_payload(payload: Dictionary) -> void:
    var wallet_variant: Variant = payload.get("wallet", {})
    if wallet_variant is Dictionary and not wallet_variant.is_empty():
        GameState.set_wallet_snapshot(wallet_variant)
