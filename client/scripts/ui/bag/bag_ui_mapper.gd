extends RefCounted
class_name BagUiMapper


## 读取服务端物品名称。
static func item_name(item: Dictionary) -> String:
    return str(item.get("item_name", item.get("name", "未知物品")))


## 读取服务端物品描述。
static func description(item: Dictionary) -> String:
    return str(item.get("description", item.get("desc", "暂无描述。")))


## 读取服务端权威物品模板 id。
static func item_id(item: Dictionary) -> int:
    return int(item.get("item_id", 0))


## 按 item_id 从本地注册表解析图标贴图。
static func icon_texture(item: Dictionary) -> Texture2D:
    return ItemIcons.resolve_texture(item_id(item))


## 读取服务端权威格子编号。
static func slot_index(item: Dictionary) -> int:
    return int(item.get("slot_index", 0))


## 读取服务端权威数量。
static func quantity(item: Dictionary) -> int:
    return int(item.get("quantity", item.get("count", 1)))


## 判断是否显示右下角计数，优先使用服务端 is_stackable 字段。
static func is_stackable(item: Dictionary) -> bool:
    if item.has("is_stackable"):
        return bool(item.get("is_stackable", false))
    return quantity(item) > 1


## 判断指定行为是否由服务端标记为可用。
static func has_action(item: Dictionary, action_key: String) -> bool:
    var actions_variant: Variant = item.get("available_actions", [])
    if actions_variant is Array:
        return (actions_variant as Array).has(action_key)
    if action_key == "use":
        return bool(item.get("usable", false))
    return false


## 格式化物品类型文案。
static func item_type_text(item: Dictionary) -> String:
    var item_type: String = str(item.get("item_type", ""))
    if item_type.is_empty():
        return "类型：未知"
    return "类型：%s" % item_type
