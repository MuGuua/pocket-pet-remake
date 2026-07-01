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
    return str(item.get("description", item.get("desc", "")))


## 装备属性 bonus 字段与中文展示标签的固定顺序映射。
const BONUS_STAT_LINES: Array[Dictionary] = [
    {"key": "hp_max", "label": "生命"},
    {"key": "mana", "label": "法力"},
    {"key": "atk", "label": "攻击"},
    {"key": "def", "label": "防御"},
    {"key": "spd", "label": "速度"},
    {"key": "spirit", "label": "精力"},
    {"key": "spirit_max", "label": "精力上限"},
    {"key": "hit_pct", "label": "命中"},
    {"key": "dodge_pct", "label": "闪避"},
    {"key": "crit_rate_pct", "label": "致命"},
    {"key": "crit_dmg_pct", "label": "爆伤"},
    {"key": "physical_resist_pct", "label": "物抗"},
    {"key": "reverse_physical_resist_pct", "label": "逆物抗"},
    {"key": "skill_resist_pct", "label": "技抗"},
    {"key": "reverse_skill_resist_pct", "label": "逆技抗"},
    {"key": "confusion_resist_pct", "label": "混乱抗性"},
    {"key": "sleep_resist_pct", "label": "昏睡抗性"},
    {"key": "paralysis_resist_pct", "label": "麻痹抗性"},
    {"key": "seal_resist_pct", "label": "封印抗性"},
    {"key": "curse_resist_pct", "label": "诅咒抗性"},
    {"key": "crit_dmg_resist_pct", "label": "抗爆伤"},
    {"key": "crit_resist_pct", "label": "抗致命"},
    {"key": "character_resist_pct", "label": "抗人物"},
    {"key": "pet_resist_pct", "label": "抗宠物"},
]


## 组装详情面板描述区：后台介绍原文 + 换行 + 装备属性行（基础与强化已合并）。
static func detail_description_text(item: Dictionary) -> String:
    var lines: PackedStringArray = PackedStringArray()
    var intro_text: String = description(item)
    if not intro_text.is_empty():
        lines.append(intro_text)
    var bonus_lines: PackedStringArray = bonus_stat_lines(item)
    for line_index: int in bonus_lines.size():
        lines.append(bonus_lines[line_index])
    if lines.is_empty():
        return "暂无描述。"
    return "\n".join(lines)


## 读取装备属性 bonus 的中文展示行；无有效属性时返回空数组。
static func bonus_stat_lines(item: Dictionary) -> PackedStringArray:
    var lines: PackedStringArray = PackedStringArray()
    var bonus_variant: Variant = item.get("bonus", {})
    if bonus_variant is not Dictionary:
        return lines
    var bonus: Dictionary = bonus_variant as Dictionary
    for entry: Dictionary in BONUS_STAT_LINES:
        var stat_key: String = str(entry.get("key", ""))
        var stat_label: String = str(entry.get("label", ""))
        if stat_key.is_empty() or stat_label.is_empty():
            continue
        var stat_value: int = int(bonus.get(stat_key, 0))
        if stat_value <= 0:
            continue
        lines.append("%s+%s" % [stat_label, UiFormat.value_to_text(stat_value)])
    return lines


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


## 读取背包实例唯一标识；实例化物品丢弃时应优先传该字段。
static func item_uid(item: Dictionary) -> String:
    return str(item.get("item_uid", "")).strip_edges()


## 判断是否为带 item_uid 的实例化物品。
static func is_instanced_item(item: Dictionary) -> bool:
    return not item_uid(item).is_empty()


## 判断是否支持在丢弃时选择部分数量（可堆叠且无 item_uid）。
static func supports_partial_drop(item: Dictionary) -> bool:
    if is_instanced_item(item):
        return false
    return is_stackable(item) and quantity(item) > 1


## 计算实例化物品或整格丢弃时的默认数量。
static func default_drop_quantity(item: Dictionary) -> int:
    return maxi(1, quantity(item))


## 读取服务端权威佩戴等级要求。
static func required_level(item: Dictionary) -> int:
    return int(item.get("required_level", 0))


## 读取服务端权威装备强化等级。
static func enhance_level(item: Dictionary) -> int:
    return int(item.get("enhance_level", 0))


## 读取服务端权威装备损坏标记。
static func is_damaged(item: Dictionary) -> bool:
    return bool(item.get("is_damaged", false))


## 格式化修复消耗文案，供修复确认弹窗展示。
static func repair_cost_text(item: Dictionary) -> String:
    var preview_variant: Variant = item.get("repair_preview", {})
    if preview_variant is not Dictionary:
        return "修复消耗：修复宝石 ×1"
    var preview: Dictionary = preview_variant as Dictionary
    var cost_item_name: String = str(preview.get("cost_item_name", "修复宝石")).strip_edges()
    if cost_item_name.is_empty():
        cost_item_name = "修复宝石"
    var cost_quantity: int = int(preview.get("cost_quantity", 1))
    if cost_quantity <= 0:
        cost_quantity = 1
    var owned_quantity: int = int(preview.get("owned_cost_quantity", 0))
    return "修复消耗：%s ×%s（拥有 %s）" % [
        cost_item_name,
        UiFormat.value_to_text(cost_quantity),
        UiFormat.value_to_text(owned_quantity),
    ]


## 格式化佩戴等级文案；非装备或等级为 0 时返回空字符串。
static func required_level_text(item: Dictionary) -> String:
    if not is_equipment(item):
        return ""
    var level_value: int = required_level(item)
    if level_value <= 0:
        return ""
    return "等级：%s" % UiFormat.value_to_text(level_value)


## 格式化强化等级文案；非装备或等级为 0 时返回空字符串。
static func enhance_level_text(item: Dictionary) -> String:
    if not is_equipment(item):
        return ""
    var level_value: int = enhance_level(item)
    if level_value <= 0:
        return ""
    return "强化：+%s" % UiFormat.value_to_text(level_value)


## 格式化装备图标右下角强化角标；等级为 0 或非装备时返回空字符串。
static func enhance_level_badge_text(item: Dictionary) -> String:
    if not is_equipment(item):
        return ""
    var level_value: int = enhance_level(item)
    if level_value <= 0:
        return ""
    return "+%s" % UiFormat.value_to_text(level_value)


## 刷新装备图标右下角强化角标；label 为空时不处理。
static func apply_enhance_level_badge(label: Label, item: Dictionary) -> void:
    if label == null:
        return
    var badge_text: String = enhance_level_badge_text(item)
    label.visible = not badge_text.is_empty()
    label.text = badge_text


## 刷新背包格子损坏样式：图标变暗，并在已损坏装备上显示 Damaged 蒙版。
## damaged_badge 为 slot.tscn 根节点下的 Damaged 面板；传入时优先使用该蒙版，并关闭旧版红色遮罩。
static func apply_damaged_slot_style(
        icon_rect: CanvasItem,
        overlay: CanvasItem,
        item: Dictionary,
        damaged_badge: CanvasItem = null
) -> void:
    var damaged: bool = is_damaged(item) and is_equipment(item)
    if icon_rect != null:
        icon_rect.modulate = Color(0.72, 0.52, 0.52, 1.0) if damaged else Color(1, 1, 1, 1)
    if damaged_badge != null:
        damaged_badge.visible = damaged
        if overlay != null:
            overlay.visible = false
    elif overlay != null:
        overlay.visible = damaged


## 判断当前是否允许发起强化；损坏装备不可强化。
static func supports_enhance_action(item: Dictionary) -> bool:
    if not is_equipment(item) or is_damaged(item):
        return false
    var preview_variant: Variant = item.get("enhance_preview", {})
    if preview_variant is not Dictionary:
        return false
    return bool((preview_variant as Dictionary).get("can_enhance", false))


## 判断当前是否允许发起修复；依赖服务端 repair_preview.can_repair。
static func supports_repair_action(item: Dictionary) -> bool:
    if not is_equipment(item) or not is_damaged(item):
        return false
    var preview_variant: Variant = item.get("repair_preview", {})
    if preview_variant is not Dictionary:
        return false
    return bool((preview_variant as Dictionary).get("can_repair", false))


## 判断详情面板是否应展示等级/强化信息行。
static func should_show_equipment_level_row(item: Dictionary) -> bool:
    return is_equipment(item)


## 判断是否显示右下角计数，优先使用服务端 is_stackable 字段。
static func is_stackable(item: Dictionary) -> bool:
    if item.has("is_stackable"):
        return bool(item.get("is_stackable", false))
    return quantity(item) > 1


## 判断当前物品是否允许丢弃，依赖服务端 can_drop 字段。
static func supports_drop_action(item: Dictionary) -> bool:
    if item.is_empty():
        return false
    return bool(item.get("can_drop", false))


## 判断指定行为是否由服务端标记为可用。
static func has_action(item: Dictionary, action_key: String) -> bool:
    if item.has("available_actions"):
        var actions_variant: Variant = item.get("available_actions")
        if actions_variant is Array:
            return (actions_variant as Array).has(action_key)
    if action_key == "use":
        return bool(item.get("usable", false))
    return false


## 判断当前物品是否属于人物装备类型。
static func is_equipment(item: Dictionary) -> bool:
    if str(item.get("item_type", "")).to_lower() == "equipment":
        return true
    return not str(item.get("equip_slot", "")).strip_edges().is_empty()


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


## 判断当前物品是否需要先选择目标宠物。
static func requires_pet_target(item: Dictionary) -> bool:
    return str(item.get("target_type", "")).to_lower() == "pet_single"


## 判断当前物品是否需要先选择目标装备实例。
static func requires_equipment_target(item: Dictionary) -> bool:
    return str(item.get("target_type", "")).to_lower() == "equipment_single"


## 判断新版背包是否已经具备当前物品的“主操作”入口。
static func supports_primary_action(item: Dictionary) -> bool:
    if item.is_empty():
        return false
    if is_equipment(item) and is_damaged(item):
        return true
    if is_equipment(item):
        return true
    if is_box_item(item):
        return has_action(item, "use") or has_action(item, "open") or bool(item.get("usable", false))
    if requires_pet_target(item) or requires_equipment_target(item):
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


## 汇总背包与已穿戴装备，供 equipment_single 消耗品选择目标。
static func collect_equipment_use_targets() -> Array:
    var results: Array = []
    var seen_uids: Dictionary = {}
    var bag_items_variant: Variant = GameState.bag_container.get("items", [])
    if bag_items_variant is Array:
        _append_equipment_use_targets(bag_items_variant as Array, "", results, seen_uids)
    _append_equipment_use_targets(GameState.equipped_items, "equipped", results, seen_uids)
    return results


## 将一批物品中的装备实例写入目标列表，按 item_uid 去重。
static func _append_equipment_use_targets(
    items: Array,
    source_tag: String,
    results: Array,
    seen_uids: Dictionary
) -> void:
    for item_variant: Variant in items:
        if item_variant is not Dictionary:
            continue
        var item: Dictionary = item_variant as Dictionary
        if not is_equipment(item):
            continue
        var uid: String = item_uid(item)
        if uid.is_empty() or seen_uids.has(uid):
            continue
        seen_uids[uid] = true
        var entry: Dictionary = item.duplicate(true)
        if not source_tag.is_empty():
            entry["use_target_source"] = source_tag
        results.append(entry)


## 格式化宠物目标按钮文案。
static func format_pet_use_target_label(pet: Dictionary) -> String:
    var pet_id: int = int(pet.get("pet_id", 0))
    var level_value: int = int(pet.get("level", 1))
    var hp_value: int = int(pet.get("hp", 0))
    var hp_max_value: int = int(pet.get("hp_max", 0))
    var lineup_suffix: String = " 出战" if bool(pet.get("in_lineup", false)) else ""
    return "宠物#%s Lv.%s HP %s/%s%s" % [
        UiFormat.value_to_text(pet_id),
        UiFormat.value_to_text(level_value),
        UiFormat.value_to_text(hp_value),
        UiFormat.value_to_text(hp_max_value),
        lineup_suffix,
    ]


## 格式化装备目标按钮文案。
static func format_equipment_use_target_label(item: Dictionary) -> String:
    var name_text: String = item_name(item)
    var enhance_value: int = enhance_level(item)
    var enhance_suffix: String = ""
    if enhance_value > 0:
        enhance_suffix = " +%s" % UiFormat.value_to_text(enhance_value)
    var slot_text: String = equip_slot_text(item).replace("部位：", "")
    if not slot_text.is_empty():
        slot_text = " · %s" % slot_text
    var source_suffix: String = ""
    if str(item.get("use_target_source", "")).strip_edges() == "equipped":
        source_suffix = " · 已穿戴"
    return "%s%s%s%s" % [name_text, enhance_suffix, slot_text, source_suffix]


## 根据 USE_ITEM 回包 result 生成玩家可读的成功提示。
static func build_use_item_success_notice(result: Dictionary) -> String:
    if result.is_empty():
        return "物品使用成功。"
    var effect_type: String = str(result.get("effect_type", "")).strip_edges()
    if effect_type == "pet_hp_restore" or int(result.get("restored_hp", 0)) > 0:
        return "已恢复宠物生命 +%s。" % UiFormat.value_to_text(int(result.get("restored_hp", 0)))
    if effect_type == "bag_expand" or effect_type == "warehouse_expand":
        var target_name: String = "背包" if str(result.get("expand_target", "")) == "bag" else "仓库"
        return "%s容量已扩展 %s 格。" % [
            target_name,
            UiFormat.value_to_text(int(result.get("expand_slots", 0))),
        ]
    if not str(result.get("unlocked_talisman_slot", "")).is_empty():
        return "已解锁宠物神符槽。"
    var applied_variant: Variant = result.get("applied_effects", [])
    if applied_variant is Array and not (applied_variant as Array).is_empty():
        return "物品效果已生效（共 %s 条）。" % UiFormat.value_to_text((applied_variant as Array).size())
    return "物品使用成功。"
