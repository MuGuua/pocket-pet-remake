extends Control

# 战斗场景中技能标识到展示文案的映射表。
const SKILL_LABELS := {
    1001: "普通攻击",
    1002: "火花冲击",
    90001: "野性撞击",
    90002: "利爪突袭",
}

# 战斗标题标签。
@onready var title_label: Label = %TitleLabel
# 战斗摘要标签。
@onready var summary_label: Label = %SummaryLabel
# 我方单位摘要标签。
@onready var ally_label: Label = %AllyLabel
# 敌方单位摘要标签。
@onready var enemy_label: Label = %EnemyLabel
# 战斗详情标签。
@onready var detail_label: Label = %DetailLabel
# 提示文案标签。
@onready var hint_label: Label = %HintLabel
# 当前动作提交状态标签。
@onready var action_status_label: Label = %ActionStatusLabel
# 第一技能按钮。
@onready var primary_skill_button: Button = %PrimarySkillButton
# 第二技能按钮。
@onready var secondary_skill_button: Button = %SecondarySkillButton

# 标记当前是否已有待服务端结算的动作。
var _is_action_pending: bool = false
# 缓存战斗控制器节点引用。
var _battle_controller: Node

# 初始化战斗场景显示，并绑定全局战斗状态变化信号。
func _ready() -> void:
    _battle_controller = get_node_or_null("../../BattleController")
    GameState.battle_changed.connect(_refresh_view)
    if _battle_controller != null and _battle_controller.has_signal("action_responded"):
        _battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
    _bind_skill_buttons()
    _refresh_view()

# 退出战斗场景时注销已绑定的信号。
func _exit_tree() -> void:
    if GameState.battle_changed.is_connected(_refresh_view):
        GameState.battle_changed.disconnect(_refresh_view)
    if _battle_controller != null and _battle_controller.has_signal("action_responded") and _battle_controller.is_connected("action_responded", Callable(self, "_on_action_responded")):
        _battle_controller.disconnect("action_responded", Callable(self, "_on_action_responded"))

# 按当前战斗快照刷新战斗界面全部显示内容。
func _refresh_view() -> void:
    _is_action_pending = false
    title_label.text = "战斗中" if GameState.is_in_battle else "战斗结算"

    # 读取当前战斗唯一标识用于摘要展示。
    var battle_id := str(GameState.battle_state.get("battle_id", "未分配"))
    # 读取当前战斗回合数用于摘要展示。
    var round_text := str(GameState.battle_state.get("round", GameState.battle_state.get("turn", 0)))
    # 读取当前出战宠物唯一标识用于摘要展示。
    var active_pet_uid := int(GameState.battle_state.get("active_pet_uid", 0))
    summary_label.text = "战斗ID: %s | 回合: %s | 出战宠: %s" % [battle_id, round_text, str(active_pet_uid)]
    # 读取当前轮到行动的我方单位快照。
    var active_ally := GameState.active_battle_actor("allies")
    ally_label.text = "我方: " + _build_actor_text(active_ally, _actor_state(active_ally))
    enemy_label.text = "敌方: " + _build_actor_text(_first_actor("enemies"), _actor_state(_first_actor("enemies")))

    # 保存要展示在详情区域中的战斗事件文本。
    var detail_parts: Array[String] = []
    # 读取当前战斗事件列表。
    var events_variant: Variant = GameState.battle_state.get("events", [])
    if events_variant is Array and not events_variant.is_empty():
        # 读取最近一次战斗事件快照。
        var last_event_variant: Variant = events_variant[events_variant.size() - 1]
        if last_event_variant is Dictionary:
            detail_parts.append(_format_event(last_event_variant))
    for key in ["enemy_name", "enemy_id", "result", "winner", "reason"]:
        if GameState.battle_state.has(key):
            detail_parts.append("%s=%s" % [key, str(GameState.battle_state.get(key, ""))])
    if detail_parts.is_empty():
        detail_label.text = "等待服务端同步战斗详情。"
    else:
        detail_label.text = " | ".join(detail_parts)

    hint_label.text = "战斗场景已接管显示，等待服务端继续推进。"
    action_status_label.text = "操作将提交给服务端处理。"
    if not GameState.is_in_battle:
        hint_label.text = "收到战斗结果，正在返回世界场景。"
        action_status_label.text = "战斗已结束。"
    _refresh_skill_buttons()

# 处理技能按钮点击，并向服务端提交动作意图。
func _on_skill_button_pressed(skill_id: int) -> void:
    # 读取当前可行动的我方单位快照。
    var ally := GameState.active_battle_actor("allies")
    # 读取当前默认攻击目标的敌方单位快照。
    var enemy := _first_actor("enemies")
    if ally.is_empty() or enemy.is_empty() or _is_action_pending:
        action_status_label.text = "缺少可用战斗目标。"
        return

    _is_action_pending = true
    action_status_label.text = "已提交 `%s` 指令，等待服务端结算。" % _skill_label(skill_id)
    _refresh_skill_buttons()
    App.submit_battle_action(
        int(GameState.battle_state.get("battle_id", 0)),
        int(GameState.battle_state.get("round", 1)),
        int(ally.get("actor_id", 0)),
        int(enemy.get("actor_id", 0)),
        1,
        skill_id
    )

# 处理战斗动作回执，并在服务端拒绝时恢复按钮可用性。
func _on_action_responded(accepted: bool, reason: String) -> void:
    if accepted:
        return
    _is_action_pending = false
    action_status_label.text = "服务端拒绝动作: %s" % reason
    _refresh_skill_buttons()

# 返回指定阵营下的首个战斗单位快照。
func _first_actor(group_key: String) -> Dictionary:
    # 读取指定阵营的战斗单位列表。
    var actors_variant: Variant = GameState.battle_state.get(group_key, [])
    if actors_variant is Array and not actors_variant.is_empty():
        # 读取当前阵营中的首个单位快照。
        var actor_variant: Variant = actors_variant[0]
        if actor_variant is Dictionary:
            return actor_variant
    return {}

# 根据单位标识从战斗状态列表中查找实时属性快照。
func _actor_state(actor: Dictionary) -> Dictionary:
    if actor.is_empty():
        return {}
    # 读取当前单位的战斗内 actor 标识。
    var actor_id := int(actor.get("actor_id", 0))
    # 读取服务端同步的战斗单位状态列表。
    var states_variant: Variant = GameState.battle_state.get("actors", [])
    if states_variant is Array:
        for state_variant in states_variant:
            if state_variant is Dictionary and int(state_variant.get("actor_id", 0)) == actor_id:
                return state_variant
    return {}

# 组装战斗单位展示文案。
func _build_actor_text(actor: Dictionary, state: Dictionary) -> String:
    if actor.is_empty():
        return "未同步"
    # 读取单位当前 HP，优先采用状态快照中的值。
    var hp := int(state.get("hp", actor.get("hp", 0)))
    # 读取单位最大 HP，优先采用状态快照中的值。
    var hp_max := int(state.get("hp_max", actor.get("hp_max", 0)))
    return "%s HP %d/%d" % [str(actor.get("name", "未知")), hp, hp_max]

# 把服务端战斗事件快照转为战斗详情文案。
func _format_event(event_payload: Dictionary) -> String:
    # 读取事件类型标识。
    var event_type := int(event_payload.get("event_type", 0))
    # 读取事件附带的数值结果。
    var value := int(event_payload.get("value", 0))
    # 读取事件关联的技能文案。
    var skill_label := _skill_label(int(event_payload.get("skill_id", 0)))
    match event_type:
        1:
            return "服务端已执行 `%s`。" % skill_label
        2:
            return "服务端用 `%s` 结算伤害 %d。" % [skill_label, value]
        _:
            return "服务端同步了新的战斗事件。"

# 为技能按钮绑定统一的点击回调。
func _bind_skill_buttons() -> void:
    if primary_skill_button != null:
        primary_skill_button.pressed.connect(func() -> void: _on_skill_button_pressed(int(primary_skill_button.get_meta("skill_id", 0))))
    if secondary_skill_button != null:
        secondary_skill_button.pressed.connect(func() -> void: _on_skill_button_pressed(int(secondary_skill_button.get_meta("skill_id", 0))))

# 按当前出战单位技能列表刷新技能按钮显示。
func _refresh_skill_buttons() -> void:
    # 读取当前可行动的我方单位快照。
    var ally := GameState.active_battle_actor("allies")
    # 读取当前单位拥有的技能标识列表。
    var skills_variant: Variant = ally.get("skill_ids", [])
    # 保存整理后的技能标识数组。
    var skill_ids: Array[int] = []
    if skills_variant is Array:
        for skill_id_variant in skills_variant:
            skill_ids.append(int(skill_id_variant))

    _apply_skill_button(primary_skill_button, skill_ids, 0)
    _apply_skill_button(secondary_skill_button, skill_ids, 1)

# 按索引把指定技能配置应用到某个按钮上。
func _apply_skill_button(button: Button, skill_ids: Array[int], index: int) -> void:
    if button == null:
        return
    if index >= skill_ids.size():
        button.visible = false
        button.disabled = true
        button.set_meta("skill_id", 0)
        return

    # 读取当前索引位置对应的技能标识。
    var skill_id := skill_ids[index]
    button.visible = true
    button.text = _skill_label(skill_id)
    button.set_meta("skill_id", skill_id)
    button.disabled = not GameState.is_in_battle or _first_actor("enemies").is_empty() or _is_action_pending

# 返回指定技能标识对应的展示文案。
func _skill_label(skill_id: int) -> String:
    return str(SKILL_LABELS.get(skill_id, "技能%d" % skill_id))
