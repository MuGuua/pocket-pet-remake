extends RefCounted
class_name BattleFormationMapper

## 战斗站位布局：围绕 battle_scene.tscn 中 MagicCircle 的中心点左右对称分布。

const ALLY_SLOT_PREFIXES: Array[String] = ["left_front", "left_back"]
const ENEMY_SLOT_PREFIXES: Array[String] = ["right_front", "right_back"]
## 前排站位到对称中心点的默认水平距离；BattleScene 可导出覆盖，方便按美术布局微调。
const DEFAULT_FRONT_COLUMN_DISTANCE_FROM_CENTER: float = 105.0
## 后排比前排额外远离中心点的默认水平距离；BattleScene 可导出覆盖。
const DEFAULT_BACK_COLUMN_EXTRA_DISTANCE: float = 60.0

## 四方及以下全部上下站位；超过四方则按组排布。
const COMPACT_FORMATION_MAX: int = 4
## 单列模式下相邻单位默认 Y 间距（像素）；BattleScene 可导出覆盖。
const DEFAULT_VERTICAL_UNIT_SPACING: float = 126.0
## 单列模式下人物相对列 X 的补偿；为了围绕 MagicCircle 中心严格对称，当前不额外偏移。
const ALLY_PLAYER_COMPACT_X_OFFSET: float = 0.0
## 分组模式下相邻小组默认 Y 间距（像素）；通常略大于单列间距，避免前后排视觉挤在一起。
const DEFAULT_GROUP_VERTICAL_SPACING: float = 144.0
## 分组模式相比单列模式额外增加的默认 Y 间距。
const GROUP_VERTICAL_SPACING_EXTRA: float = DEFAULT_GROUP_VERTICAL_SPACING - DEFAULT_VERTICAL_UNIT_SPACING

## 与 battle_scene.tscn 中 Bg/TextureRect2 的 MagicCircle 可视区域保持一致：战斗弹窗为 780x1440，魔法阵 600x300 居中显示。
const MAGIC_CIRCLE_LEFT_X: float = 90.0
const MAGIC_CIRCLE_RIGHT_X: float = 690.0
const MAGIC_CIRCLE_TOP_Y: float = 430.0
const MAGIC_CIRCLE_BOTTOM_Y: float = 730.0
const MAGIC_CIRCLE_CENTER_X: float = (MAGIC_CIRCLE_LEFT_X + MAGIC_CIRCLE_RIGHT_X) * 0.5
const MAGIC_CIRCLE_CENTER_Y: float = (MAGIC_CIRCLE_TOP_Y + MAGIC_CIRCLE_BOTTOM_Y) * 0.5

## 运行时可调整的战斗站位对称中心点；默认等于 MagicCircle 中心。
static var formation_center: Vector2 = Vector2(MAGIC_CIRCLE_CENTER_X, MAGIC_CIRCLE_CENTER_Y)
## 运行时可调整的单位到对称中心点主间距；左右两侧始终以这个值保持对称。
static var side_distance_from_center: float = DEFAULT_FRONT_COLUMN_DISTANCE_FROM_CENTER
## 运行时可调整的上下分布单位 Y 间距；左右两侧都使用同一数值保持对称。
static var vertical_unit_spacing: float = DEFAULT_VERTICAL_UNIT_SPACING
## 运行时可调整的两列水平间距；后列在各自阵营方向上额外远离中心。
static var back_column_extra_distance: float = DEFAULT_BACK_COLUMN_EXTRA_DISTANCE
## 运行时可调整的第二列 Y 偏移；正数向下，负数向上。
static var back_column_y_offset: float = 0.0


## 配置战斗站位对称中心点；由 battle_scene.tscn 的导出变量驱动，方便在 Inspector 里微调。
static func configure_formation_center(center: Vector2) -> void:
    formation_center = center


## 配置战斗单位左右两侧到对称中心点的主间距；数值越大双方站得越开。
static func configure_side_distance(distance: float) -> void:
    side_distance_from_center = maxf(0.0, distance)


## 配置同侧出现两列时第二列与第一列的水平间距。
static func configure_back_column_extra_distance(distance: float) -> void:
    back_column_extra_distance = maxf(0.0, distance)


## 配置同侧出现两列时第二列相对第一列的 Y 偏移。
static func configure_back_column_y_offset(offset_y: float) -> void:
    back_column_y_offset = offset_y


## 配置多单位上下分布时的 Y 间距；数值越大，上下单位站得越开。
static func configure_vertical_spacing(spacing: float) -> void:
    vertical_unit_spacing = maxf(0.0, spacing)


## 返回我方前排 X 坐标。
static func get_ally_front_x() -> float:
    return formation_center.x - side_distance_from_center


## 返回我方后排 X 坐标。
static func get_ally_back_x() -> float:
    return formation_center.x - (side_distance_from_center + back_column_extra_distance)


## 返回敌方前排 X 坐标。
static func get_enemy_front_x() -> float:
    return formation_center.x + side_distance_from_center


## 返回敌方后排 X 坐标。
static func get_enemy_back_x() -> float:
    return formation_center.x + side_distance_from_center + back_column_extra_distance


## 返回左右对阵分界线 X 坐标；右侧单位命中左侧目标时，客户端镜像技能特效。
static func get_battlefield_split_x() -> float:
    return formation_center.x


## 根据单方全部单位计算 slot_position，写入每个 unit 字典。
static func assign_unit_positions(units: Array[Dictionary]) -> void:
    var allies: Array[Dictionary] = []
    var enemies: Array[Dictionary] = []
    for unit: Dictionary in units:
        if str(unit.get("type", "")) == "enemy":
            enemies.append(unit)
        else:
            allies.append(unit)
    _assign_side_positions(allies, true)
    _assign_side_positions(enemies, false)


## 返回 slot_key -> Vector2 的站位表，供旧 demo 与 fallback 使用。
static func build_slot_positions() -> Dictionary:
    var result: Dictionary = {}
    var battle_area: Dictionary = {
        "top": MAGIC_CIRCLE_TOP_Y,
        "bottom": MAGIC_CIRCLE_BOTTOM_Y,
        "anchor_y": formation_center.y,
        "ally_back_x": get_ally_back_x(),
        "ally_front_x": get_ally_front_x(),
        "enemy_front_x": get_enemy_front_x(),
        "enemy_back_x": get_enemy_back_x(),
        "ally_rows": 4,
        "enemy_rows": 6,
        "ally_row_spacing": vertical_unit_spacing,
        "enemy_row_spacing": vertical_unit_spacing,
        "unit_footprint": 96.0,
    }
    _apply_battle_area_layout(result, battle_area)
    result["left_front"] = result.get("left_front_1", Vector2(get_ally_front_x(), formation_center.y))
    result["right_front"] = result.get("right_front_1", Vector2(get_enemy_front_x(), formation_center.y))
    return result


## 兼容旧逻辑：根据 lineup_index 生成站位 key（联机战斗请优先 assign_unit_positions）。
static func resolve_position_key(is_ally: bool, lineup_index: int) -> String:
    var row_index: int = max(0, lineup_index)
    var prefixes: Array[String] = ALLY_SLOT_PREFIXES if is_ally else ENEMY_SLOT_PREFIXES
    var prefix: String = prefixes[min(row_index, prefixes.size() - 1)]
    return "%s_%d" % [prefix, row_index + 1]


static func _assign_side_positions(side_units: Array[Dictionary], is_ally: bool) -> void:
    if side_units.is_empty():
        return
    var front_x: float = get_ally_front_x() if is_ally else get_enemy_front_x()
    var back_x: float = get_ally_back_x() if is_ally else get_enemy_back_x()
    if side_units.size() <= COMPACT_FORMATION_MAX:
        _assign_vertical_column(side_units, front_x)
        return
    var groups: Array[Array] = []
    if is_ally:
        groups = _build_ally_groups(side_units)
    else:
        groups = _build_enemy_pair_groups(side_units)
    _assign_grouped_positions(groups, front_x, back_x)


## 四方及以下：同列 X，按 Y 上下排开；人物在上、宠物在下（更靠近镜头）。
static func _assign_vertical_column(units: Array[Dictionary], column_x: float) -> void:
    var sorted_units: Array[Dictionary] = _sort_vertical_units(units)
    var count: int = sorted_units.size()
    var total_span: float = float(max(count - 1, 0)) * vertical_unit_spacing
    var start_y: float = formation_center.y - total_span * 0.5
    for index: int in range(count):
        var slot_y: float = start_y + float(index) * vertical_unit_spacing
        var slot_x: float = column_x
        if str(sorted_units[index].get("type", "")) == "player":
            slot_x += ALLY_PLAYER_COMPACT_X_OFFSET
        sorted_units[index]["slot_position"] = Vector2(slot_x, slot_y)


## 分组模式：组与组上下分布，组内前后站位（宠物前、人物后）。
static func _assign_grouped_positions(
    groups: Array[Array],
    front_x: float,
    back_x: float
) -> void:
    var group_count: int = groups.size()
    if group_count <= 0:
        return
    var group_vertical_spacing: float = vertical_unit_spacing + GROUP_VERTICAL_SPACING_EXTRA
    var total_span: float = float(max(group_count - 1, 0)) * group_vertical_spacing
    var start_y: float = formation_center.y - total_span * 0.5
    for group_index: int in range(group_count):
        var group: Array = groups[group_index]
        var group_y: float = start_y + float(group_index) * group_vertical_spacing
        _assign_single_group_positions(group, front_x, back_x, group_y)


## 组内：宠物走前排 X，人物走后排 X；敌方按顺序前二占位。
static func _assign_single_group_positions(
    group: Array,
    front_x: float,
    back_x: float,
    group_y: float
) -> void:
    var front_used: bool = false
    var back_used: bool = false
    var overflow_index: int = 0
    for member_variant: Variant in group:
        if not member_variant is Dictionary:
            continue
        var member: Dictionary = member_variant as Dictionary
        var unit_type: String = str(member.get("type", ""))
        if unit_type == "pet" and not front_used:
            member["slot_position"] = Vector2(front_x, group_y)
            front_used = true
            continue
        if unit_type == "player" and not back_used:
            member["slot_position"] = Vector2(back_x, group_y + back_column_y_offset)
            back_used = true
            continue
        if not front_used:
            member["slot_position"] = Vector2(front_x, group_y)
            front_used = true
        elif not back_used:
            member["slot_position"] = Vector2(back_x, group_y + back_column_y_offset)
            back_used = true
        else:
            overflow_index += 1
            var overflow_y: float = group_y + float(overflow_index) * 54.0
            member["slot_position"] = Vector2(back_x, overflow_y + back_column_y_offset)


## 按 owner 划分我方小组，保留服务端出现顺序。
static func _build_ally_groups(units: Array[Dictionary]) -> Array[Array]:
    var owner_order: Array[int] = []
    var owner_buckets: Dictionary = {}
    for unit: Dictionary in units:
        var owner_id: int = int(unit.get("owner_player_id", 0))
        if not owner_buckets.has(owner_id):
            owner_buckets[owner_id] = []
            owner_order.append(owner_id)
        var bucket: Array = owner_buckets[owner_id] as Array
        bucket.append(unit)
    var groups: Array[Array] = []
    for owner_id: int in owner_order:
        var members: Array = owner_buckets[owner_id] as Array
        var group: Array[Dictionary] = []
        for member_variant: Variant in members:
            if member_variant is Dictionary and str((member_variant as Dictionary).get("type", "")) == "pet":
                group.append(member_variant as Dictionary)
        for member_variant: Variant in members:
            if member_variant is Dictionary and str((member_variant as Dictionary).get("type", "")) == "player":
                group.append(member_variant as Dictionary)
        for member_variant: Variant in members:
            if member_variant is Dictionary:
                var member: Dictionary = member_variant as Dictionary
                var unit_type: String = str(member.get("type", ""))
                if unit_type != "pet" and unit_type != "player":
                    group.append(member)
        if not group.is_empty():
            groups.append(group)
    return groups


## 敌方按出现顺序两两一组。
static func _build_enemy_pair_groups(units: Array[Dictionary]) -> Array[Array]:
    var groups: Array[Array] = []
    var index: int = 0
    while index < units.size():
        var group: Array[Dictionary] = []
        group.append(units[index])
        if index + 1 < units.size():
            group.append(units[index + 1])
        groups.append(group)
        index += 2
    return groups


## 上下站位时：人物优先靠上，宠物靠下。
static func _sort_vertical_units(units: Array[Dictionary]) -> Array[Dictionary]:
    var sorted_units: Array[Dictionary] = units.duplicate()
    sorted_units.sort_custom(func(left: Dictionary, right: Dictionary) -> bool:
        var left_is_player: bool = str(left.get("type", "")) == "player"
        var right_is_player: bool = str(right.get("type", "")) == "player"
        if left_is_player != right_is_player:
            return left_is_player
        var left_index: int = int(left.get("lineup_index", 0))
        var right_index: int = int(right.get("lineup_index", 0))
        if left_index != right_index:
            return left_index < right_index
        return int(left.get("actor_id", 0)) < int(right.get("actor_id", 0))
    )
    return sorted_units


static func _apply_battle_area_layout(target: Dictionary, battle_area: Dictionary) -> void:
    var top: float = float(battle_area.get("top", 58.0))
    var bottom: float = float(battle_area.get("bottom", 296.0))
    var ally_rows: int = clampi(int(battle_area.get("ally_rows", 4)), 1, 8)
    var enemy_rows: int = clampi(int(battle_area.get("enemy_rows", 6)), 1, 12)
    var ally_back_x: float = float(battle_area.get("ally_back_x", get_ally_back_x()))
    var ally_front_x: float = float(battle_area.get("ally_front_x", get_ally_front_x()))
    var enemy_front_x: float = float(battle_area.get("enemy_front_x", get_enemy_front_x()))
    var enemy_back_x: float = float(battle_area.get("enemy_back_x", get_enemy_back_x()))
    var unit_footprint: float = float(battle_area.get("unit_footprint", 32.0))
    var usable_height: float = max(bottom - top, unit_footprint)
    var ally_spacing: float = float(battle_area.get("ally_row_spacing", 0.0))
    var enemy_spacing: float = float(battle_area.get("enemy_row_spacing", 0.0))
    if ally_spacing <= 0.0:
        ally_spacing = (usable_height - unit_footprint) / float(max(ally_rows - 1, 1))
    if enemy_spacing <= 0.0:
        enemy_spacing = (usable_height - unit_footprint) / float(max(enemy_rows - 1, 1))
    var anchor_y: float = float(battle_area.get("anchor_y", -1.0))
    if anchor_y >= 0.0:
        for row_index: int in range(ally_rows):
            var slot_y: float = anchor_y - float(row_index) * ally_spacing
            target["left_back_%d" % (row_index + 1)] = Vector2(ally_back_x, slot_y)
            target["left_front_%d" % (row_index + 1)] = Vector2(ally_front_x, slot_y)
        for row_index: int in range(enemy_rows):
            var slot_y: float = anchor_y - float(row_index) * enemy_spacing
            target["right_front_%d" % (row_index + 1)] = Vector2(enemy_front_x, slot_y)
            target["right_back_%d" % (row_index + 1)] = Vector2(enemy_back_x, slot_y)
        return
    var vertical_anchor: float = clampf(float(battle_area.get("vertical_anchor", 0.5)), 0.0, 1.0)
    var ally_block_height: float = unit_footprint + ally_spacing * float(ally_rows - 1)
    var enemy_block_height: float = unit_footprint + enemy_spacing * float(enemy_rows - 1)
    var ally_start_y: float = top + (usable_height - ally_block_height) * vertical_anchor
    var enemy_start_y: float = top + (usable_height - enemy_block_height) * vertical_anchor
    for row_index: int in range(ally_rows):
        var slot_y: float = ally_start_y + float(row_index) * ally_spacing
        target["left_back_%d" % (row_index + 1)] = Vector2(ally_back_x, slot_y)
        target["left_front_%d" % (row_index + 1)] = Vector2(ally_front_x, slot_y)
    for row_index: int in range(enemy_rows):
        var slot_y: float = enemy_start_y + float(row_index) * enemy_spacing
        target["right_front_%d" % (row_index + 1)] = Vector2(enemy_front_x, slot_y)
        target["right_back_%d" % (row_index + 1)] = Vector2(enemy_back_x, slot_y)
