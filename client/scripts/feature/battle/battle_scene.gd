extends Control

@onready var title_label: Label = %TitleLabel
@onready var summary_label: Label = %SummaryLabel
@onready var ally_label: Label = %AllyLabel
@onready var enemy_label: Label = %EnemyLabel
@onready var detail_label: Label = %DetailLabel
@onready var hint_label: Label = %HintLabel
@onready var action_status_label: Label = %ActionStatusLabel
@onready var attack_button: Button = %AttackButton

func _ready() -> void:
    GameState.battle_changed.connect(_refresh_view)
    if attack_button != null:
        attack_button.pressed.connect(_on_attack_button_pressed)
    _refresh_view()

func _exit_tree() -> void:
    if GameState.battle_changed.is_connected(_refresh_view):
        GameState.battle_changed.disconnect(_refresh_view)

func _refresh_view() -> void:
    title_label.text = "战斗中" if GameState.is_in_battle else "战斗结算"

    var battle_id := str(GameState.battle_state.get("battle_id", "未分配"))
    var round_text := str(GameState.battle_state.get("round", GameState.battle_state.get("turn", 0)))
    summary_label.text = "战斗ID: %s | 回合: %s" % [battle_id, round_text]
    ally_label.text = "我方: " + _build_actor_text(_first_actor("allies"), _actor_state(_first_actor("allies")))
    enemy_label.text = "敌方: " + _build_actor_text(_first_actor("enemies"), _actor_state(_first_actor("enemies")))

    var detail_parts: Array[String] = []
    var events_variant: Variant = GameState.battle_state.get("events", [])
    if events_variant is Array and not events_variant.is_empty():
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
    if attack_button != null:
        attack_button.disabled = not GameState.is_in_battle or _first_actor("allies").is_empty() or _first_actor("enemies").is_empty()

func _on_attack_button_pressed() -> void:
    var ally := _first_actor("allies")
    var enemy := _first_actor("enemies")
    if ally.is_empty() or enemy.is_empty():
        action_status_label.text = "缺少可用战斗目标。"
        return

    action_status_label.text = "已提交攻击指令，等待服务端结算。"
    App.submit_battle_action(
        int(GameState.battle_state.get("battle_id", 0)),
        int(GameState.battle_state.get("round", 1)),
        int(ally.get("actor_id", 0)),
        int(enemy.get("actor_id", 0))
    )

func _first_actor(group_key: String) -> Dictionary:
    var actors_variant: Variant = GameState.battle_state.get(group_key, [])
    if actors_variant is Array and not actors_variant.is_empty():
        var actor_variant: Variant = actors_variant[0]
        if actor_variant is Dictionary:
            return actor_variant
    return {}

func _actor_state(actor: Dictionary) -> Dictionary:
    if actor.is_empty():
        return {}
    var actor_id := int(actor.get("actor_id", 0))
    var states_variant: Variant = GameState.battle_state.get("actors", [])
    if states_variant is Array:
        for state_variant in states_variant:
            if state_variant is Dictionary and int(state_variant.get("actor_id", 0)) == actor_id:
                return state_variant
    return {}

func _build_actor_text(actor: Dictionary, state: Dictionary) -> String:
    if actor.is_empty():
        return "未同步"
    var hp := int(state.get("hp", actor.get("hp", 0)))
    var hp_max := int(state.get("hp_max", actor.get("hp_max", 0)))
    return "%s HP %d/%d" % [str(actor.get("name", "未知")), hp, hp_max]

func _format_event(event_payload: Dictionary) -> String:
    var event_type := int(event_payload.get("event_type", 0))
    var value := int(event_payload.get("value", 0))
    match event_type:
        1:
            return "服务端已执行一次技能动作。"
        2:
            return "服务端结算伤害 %d。" % value
        _:
            return "服务端同步了新的战斗事件。"
