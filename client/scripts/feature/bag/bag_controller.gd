extends Node

# 背包数据刷新后向外广播当前物品总数。
signal bag_updated(count: int)

# 处理背包列表响应，并把物品数组写入全局状态。
func handle_bag_list(payload: Dictionary) -> void:
    # 读取服务端返回的背包物品列表载荷。
    var items_variant: Variant = payload.get("items", [])
    # 规范化背包物品列表为数组结构。
    var items: Array = items_variant if items_variant is Array else []
    GameState.set_bag_items(items)
    bag_updated.emit(GameState.bag_items.size())

# 处理单个背包物品更新推送，并把结果合并进全局状态。
func handle_bag_update(payload: Dictionary) -> void:
    # 兼容 item 字段和直接物品结构两种载荷格式。
    var item_variant: Variant = payload.get("item", payload)
    # 规范化单个物品数据为字典结构。
    var item: Dictionary = item_variant if item_variant is Dictionary else {}
    GameState.upsert_bag_item(item)
    bag_updated.emit(GameState.bag_items.size())
