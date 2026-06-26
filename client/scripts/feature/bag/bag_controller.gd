extends Node

# 背包数据刷新后向外广播当前物品总数。
signal bag_updated(count: int)
# 商店购买回包到达后向外广播完整载荷。
signal buy_item_responded(payload: Dictionary)

# 处理背包列表响应，并把物品数组写入全局状态。
func handle_bag_list(payload: Dictionary) -> void:
    _apply_container_payload(payload)
    _apply_equipped_payload(payload)


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
        _refresh_paginated_bag_page_if_needed(str(payload.get("container_type", "bag")))
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
    if container_variant is Dictionary and not container_variant.is_empty():
        GameState.set_container_snapshot(container_variant)
    else:
        # 兼容旧协议仍然直接返回 items 数组的情况。
        var items_variant: Variant = payload.get("items", [])
        var items: Array = items_variant if items_variant is Array else []
        GameState.set_bag_items(items)
    _apply_wallet_payload(payload)
    _emit_bag_updated()


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
