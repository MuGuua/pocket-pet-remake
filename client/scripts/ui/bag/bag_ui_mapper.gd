extends RefCounted
class_name BagUiMapper

const ITEM_TYPE_LABELS: Dictionary = {
    "consumable": "消耗品",
    "equipment": "装备",
    "material": "材料",
    "quest": "任务物品",
    "currency": "货币",
    "misc": "杂项",
    "functional": "功能道具",
    "box": "宝箱礼包",
}

## 与服务端 effect_type 对齐的宝箱/礼包开启效果类型集合。
const BOX_EFFECT_TYPES: Array[String] = [
    "reward_box",
    "gift_box",
    "box_open",
    "open",
]

const EQUIP_SLOT_LABELS: Dictionary = {
    "weapon": "武器",
    "class_weapon": "职业武器",
    "hat": "帽子",
    "clothes": "衣服",
    "pants": "裤子",
    "shoes": "鞋子",
    "necklace": "项链",
    "ring": "戒指",
    "hero_ring": "英雄之戒",
    "badge": "徽章",
    "medicine_pouch": "药囊",
    "charm": "护符",
    "class_badge": "职业徽章",
    "element_bracelet": "元素手镯",
    "rebirth_stone": "转生之石",
    "guardian_ring": "守护之戒",
    "costume": "时装",
}

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


## 判断当前物品是否属于人物装备类型。
static func is_equipment(item: Dictionary) -> bool:
    return str(item.get("item_type", "")).to_lower() == "equipment"


## 判断当前物品是否属于服务端定义的宝箱或礼包类型。
static func is_box_item(item: Dictionary) -> bool:
    if item.is_empty():
        return false
    if str(item.get("item_type", "")).to_lower() == "box":
        return true
    var item_sub_type: String = str(item.get("item_sub_type", "")).to_lower()
    if item_sub_type == "reward_box" or item_sub_type == "gift_box":
        return true
    var effect_type: String = str(item.get("effect_type", "")).to_lower()
    return BOX_EFFECT_TYPES.has(effect_type) or has_action(item, "open")


## 判断当前物品是否需要先选择目标宠物，当前新版背包还没有对应目标选择弹窗。
static func requires_pet_target(item: Dictionary) -> bool:
    return str(item.get("target_type", "")).to_lower() == "pet_single"


## 判断新版背包是否已经具备当前物品的“主操作”入口。
## 装备类物品走人物装备协议；普通可用物品走 USE_ITEM 协议；需要宠物目标的道具暂时不开放。
static func supports_primary_action(item: Dictionary) -> bool:
    if item.is_empty():
        return false
    if requires_pet_target(item):
        return false
    if is_equipment(item):
        return true
    if is_box_item(item):
        return has_action(item, "use") or bool(item.get("usable", false))
    return has_action(item, "use")


## 格式化物品类型文案。
static func item_type_text(item: Dictionary) -> String:
    var item_type: String = str(item.get("item_type", ""))
    if item_type.is_empty():
        return "类型：未知"
    var mapped_text: String = str(ITEM_TYPE_LABELS.get(item_type, item_type))
    return "类型：%s" % mapped_text


## 格式化装备部位文案；背包装备与已穿戴装备都支持 equip_slot / equip_slot_label。
static func equip_slot_text(item: Dictionary) -> String:
    var equip_slot: String = str(item.get("equip_slot", "")).strip_edges()
    if equip_slot.is_empty():
        if not is_equipment(item):
            return ""
    var slot_label: String = str(item.get("equip_slot_label", "")).strip_edges()
    if slot_label.is_empty() and not equip_slot.is_empty():
        slot_label = str(EQUIP_SLOT_LABELS.get(equip_slot, equip_slot))
    if slot_label.is_empty():
        return ""
    return "部位：%s" % slot_label
