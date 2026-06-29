extends Node

# 背包数据刷新后向外广播当前物品总数。
signal bag_updated(count: int)
# 容器完整快照写入 GameState 后广播，供带 loading 的 UI 等待新数据落地。
signal container_snapshot_applied(container_type: String)
# 商店购买回包到达后向外广播完整载荷。
signal buy_item_responded(payload: Dictionary)
# 丢弃物品成功回包到达后向外广播完整载荷；UI 仍需等待后续容器快照刷新后再结束 loading。
signal drop_item_responded(payload: Dictionary)

# 处理背包列表响应，并把物品数组写入全局状态。
func handle_bag_list(payload: Dictionary) -> void:
    _apply_container_payload(payload)
    _apply_equipped_payload(payload)


# 处理单个背包物品更新推送，并把结果合并进全局状态。
func handle_bag_update(payload: Dictionary) -> void:
    var container_type: String = str(payload.get("container_type", "bag"))
    var updates_variant: Variant = payload.get("updates", [])
    if updates_variant is Array:
        var updates: Array = updates_variant
        GameState.apply_container_updates(
            container_type,
            updates,
            int(payload.get("capacity", 0)),
            int(payload.get("max_capacity", 0)),
            int(payload.get("used_slots", -1))
        )
        _refresh_paginated_bag_page_if_needed(container_type)
        _emit_bag_updated()
        if not updates.is_empty():
            container_snapshot_applied.emit(container_type)
    else:
        # 兼容 item 字段和直接物品结构两种旧载荷格式。
        var item_variant: Variant = payload.get("item", payload)
        var item: Dictionary = item_variant if item_variant is Dictionary else {}
        GameState.upsert_bag_item(item)
        _emit_bag_updated()


# 处理使用物品响应。
# 当前真正的容器变化仍以后续 `5011 BAG_UPDATE_PUSH` 为准，这里先保留入口方便 UI 后续接结果提示。
func handle_use_item_response(_payload: Dictionary) -> void:
    _emit_bag_updated()


# 处理丢弃物品成功响应；背包刷新与结束 loading 由 BagPanel 统一编排。
func handle_drop_item_response(payload: Dictionary) -> void:
    drop_item_responded.emit(payload if payload is Dictionary else {})


# 处理仓库等其他容器的完整快照响应。
func handle_container_list(payload: Dictionary) -> void:
    _apply_container_payload(payload)


# 处理钱包独立查询响应。
func handle_wallet_query(payload: Dictionary) -> void:
    _apply_wallet_payload(payload)


# 处理钱包增量更新推送。
func handle_wallet_update(payload: Dictionary) -> void:
    _apply_wallet_payload(payload)


# 处理购买物品响应，并同步钱包快照。
func handle_buy_item_response(payload: Dictionary) -> void:
    _apply_wallet_payload(payload)
    buy_item_responded.emit(payload)


func _apply_container_payload(payload: Dictionary) -> void:
    var container_variant: Variant = payload.get("container", {})
    var applied_container_type: String = "bag"
    if container_variant is Dictionary and not container_variant.is_empty():
        applied_container_type = str(container_variant.get("container_type", "bag"))
        GameState.set_container_snapshot(container_variant)
    else:
        # 兼容旧协议仍然直接返回 items 数组的情况。
        var items_variant: Variant = payload.get("items", [])
        var items: Array = items_variant if items_variant is Array else []
        GameState.set_bag_items(items)
    _apply_wallet_payload(payload)
    _emit_bag_updated()
    container_snapshot_applied.emit(applied_container_type)


func _apply_equipped_payload(payload: Dictionary) -> void:
    if not payload.has("equipped_items"):
        return
    var items_variant: Variant = payload.get("equipped_items", [])
    if items_variant is Array:
        GameState.set_equipped_items(items_variant)


func _apply_wallet_payload(payload: Dictionary) -> void:
    var wallet_variant: Variant = payload.get("wallet", {})
    if wallet_variant is Dictionary and not wallet_variant.is_empty():
        GameState.set_wallet_snapshot(wallet_variant)


## 若当前背包面板正处于服务端分页模式，收到增量推送后主动补拉当前页，避免局部推送把当前分页结果污染成“半页 + 其他页混入”。
func _refresh_paginated_bag_page_if_needed(container_type: String) -> void:
    if container_type != "bag" or not GameState.is_ws_authenticated:
        return
    if GameState.bag_container.is_empty() or not GameState.bag_container.has("page_size"):
        return
    var page: int = int(GameState.bag_container.get("page", 1))
    var page_size: int = int(GameState.bag_container.get("page_size", 28))
    var category: String = str(GameState.bag_container.get("category", "all"))
    App.request_bag_list(page, page_size, category)


## 统一按服务端 used_slots 广播背包占用格数，避免新版分页只返回当前页后把总数误判成 28 以内。
func _emit_bag_updated() -> void:
    var bag_count: int = int(GameState.bag_container.get("used_slots", GameState.bag_items.size()))
    bag_updated.emit(bag_count)
