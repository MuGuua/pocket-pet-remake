extends RefCounted
class_name BattleEventAdapter

## 把 4012 events[] 转成 Director 可播放的伪剧本结构。

const EVENT_USE_SKILL: int = 1
const EVENT_DAMAGE: int = 2
const EVENT_HEAL: int = 3
const EVENT_APPLY_STATUS: int = 4
const EVENT_STATUS_TICK: int = 5
const EVENT_DEFEAT: int = 7
const EVENT_COUNTER: int = 9
const EVENT_COMBO: int = 11
const EVENT_REVIVE: int = 10

## 一方全灭后截断后续事件，避免客户端继续播放未发生的出手动画。
static func trim_events_after_battle_decided(
    events: Array[Dictionary],
    network_provider: BattleNetworkProvider
) -> Array[Dictionary]:
    if events.is_empty():
        return events
    var living_allies: Dictionary = {}
    var living_enemies: Dictionary = {}
    for unit: Dictionary in network_provider.get_initial_units():
        var actor_id: int = int(unit.get("actor_id", 0))
        if actor_id <= 0:
            continue
        var unit_type: String = str(unit.get("type", ""))
        if unit_type == "enemy":
            living_enemies[actor_id] = true
        else:
            living_allies[actor_id] = true
    # 无法识别双方阵营时不截断，避免误伤终结击杀的完整演出。
    if living_allies.is_empty() or living_enemies.is_empty():
        return events.duplicate()
    var trimmed: Array[Dictionary] = []
    for event: Dictionary in events:
        trimmed.append(event)
        _apply_living_side_update(event, living_allies, living_enemies, network_provider)
        if living_allies.is_empty() or living_enemies.is_empty():
            break
    return trimmed


## 根据 defeat/revive 事件更新存活集合。
static func _apply_living_side_update(
    event: Dictionary,
    living_allies: Dictionary,
    living_enemies: Dictionary,
    network_provider: BattleNetworkProvider
) -> void:
    var event_type: int = int(event.get("event_type", 0))
    var target_id: int = int(event.get("target_id", 0))
    if target_id <= 0:
        return
    match event_type:
        EVENT_DEFEAT:
            if living_allies.has(target_id):
                living_allies.erase(target_id)
            if living_enemies.has(target_id):
                living_enemies.erase(target_id)
        EVENT_REVIVE:
            if network_provider.is_ally_actor(target_id):
                living_allies[target_id] = true
            else:
                living_enemies[target_id] = true
        _:
            pass

## 将事件列表与运行时 actor 快照适配为 round_data 字典。
static func build_round_data(
    events: Array[Dictionary],
    runtime_actors: Array[Dictionary],
    network_provider: BattleNetworkProvider
) -> Dictionary:
    var actions: Array[Dictionary] = []
    var combos: Array[Dictionary] = []
    var timeline: Array[Dictionary] = []
    var current_action: Dictionary = {}
    var action_index: int = 0

    for event: Dictionary in events:
        var event_type: int = int(event.get("event_type", 0))
        match event_type:
            EVENT_USE_SKILL:
                if not current_action.is_empty():
                    actions.append(current_action)
                    timeline.append(_timeline_step(current_action.get("id", "")))
                action_index += 1
                current_action = _begin_action(event, action_index, network_provider)
            EVENT_DAMAGE, EVENT_HEAL, EVENT_DEFEAT, EVENT_STATUS_TICK:
                if current_action.is_empty():
                    action_index += 1
                    current_action = _begin_action_from_effect(event, action_index, network_provider)
                _append_target_result(current_action, event, runtime_actors, event_type)
            EVENT_APPLY_STATUS:
                if current_action.is_empty():
                    action_index += 1
                    current_action = _begin_action_from_effect(event, action_index, network_provider)
                _append_buff_change(current_action, event)
            EVENT_COUNTER, EVENT_COMBO:
                combos.append(_build_combo_entry(event, network_provider))
            _:
                pass

    if not current_action.is_empty():
        actions.append(current_action)
        timeline.append(_timeline_step(current_action.get("id", "")))

    for combo_entry: Dictionary in combos:
        timeline.append(_timeline_step(str(combo_entry.get("id", "")), true))

    return {
        "actions": actions,
        "combo": combos,
        "timeline": timeline,
        "result": {"is_finished": false, "summary": ""},
    }

static func _timeline_step(action_ref: String, is_combo: bool = false) -> Dictionary:
    return {
        "mode": "serial",
        "action_ref": action_ref,
        "wait_ms": 180 if is_combo else 220,
    }

static func _begin_action(event: Dictionary, action_index: int, network_provider: BattleNetworkProvider) -> Dictionary:
    var source_id: int = int(event.get("source_id", 0))
    var skill_id: int = int(event.get("skill_id", 0))
    var skill_meta: Dictionary = _find_skill_meta(source_id, skill_id, network_provider)
    var action_type: String = "attack" if skill_id == App.DEFAULT_BATTLE_SKILL_ID else "skill"
    return {
        "id": "action_%03d" % action_index,
        "actor_id": source_id,
        "action_type": action_type,
        "display_name": str(skill_meta.get("display_name", "技能")),
        "skill_id": skill_id,
        "skill_visual_id": str(skill_meta.get("skill_visual_id", "")),
        "animation_key": str(skill_meta.get("animation_key", "")),
        "log_text": str(event.get("label", "")),
        "targets": [],
        "buff_changes": [],
    }

static func _begin_action_from_effect(event: Dictionary, action_index: int, network_provider: BattleNetworkProvider) -> Dictionary:
    var source_id: int = int(event.get("source_id", 0))
    var skill_id: int = int(event.get("skill_id", 0))
    var skill_meta: Dictionary = _find_skill_meta(source_id, skill_id, network_provider)
    return {
        "id": "action_%03d" % action_index,
        "actor_id": source_id,
        "action_type": "attack" if skill_id <= 0 else "skill",
        "display_name": str(skill_meta.get("display_name", "攻击")),
        "skill_id": skill_id,
        "skill_visual_id": str(skill_meta.get("skill_visual_id", "")),
        "animation_key": str(skill_meta.get("animation_key", "")),
        "log_text": str(event.get("label", "")),
        "targets": [],
        "buff_changes": [],
    }

static func _append_target_result(
    action: Dictionary,
    event: Dictionary,
    _runtime_actors: Array[Dictionary],
    event_type: int
) -> void:
    var target_id: int = int(event.get("target_id", 0))
    var value: int = int(event.get("value", 0))
    var result_type: String = "heal" if event_type == EVENT_HEAL else "damage"
    if event_type == EVENT_STATUS_TICK:
        result_type = "damage"
    if event_type == EVENT_DEFEAT:
        result_type = "defeat"
    var label_text: String = str(event.get("label", ""))
    var is_crit: bool = label_text.find("暴击") != -1
    var target_result: Dictionary = {
        "target_id": target_id,
        "result_type": result_type,
        "value": value,
        "log_text": label_text,
        "is_crit": is_crit,
        "hit_type": "crit" if is_crit else "normal",
    }
    # 普通伤害/治疗按当前本地 HP 和事件 value 逐段演算；只有击倒事件才强制归零。
    if event_type == EVENT_DEFEAT:
        target_result["hp_after"] = 0
    var targets: Array = action.get("targets", []) as Array
    targets.append(target_result)
    action["targets"] = targets

static func _append_buff_change(action: Dictionary, event: Dictionary) -> void:
    var buff_changes: Array = action.get("buff_changes", []) as Array
    buff_changes.append({
        "target_id": int(event.get("target_id", 0)),
        "buff_id": str(event.get("state_id", 0)),
        "log_text": str(event.get("label", "")),
    })
    action["buff_changes"] = buff_changes

static func _build_combo_entry(event: Dictionary, network_provider: BattleNetworkProvider) -> Dictionary:
    var source_id: int = int(event.get("source_id", 0))
    var target_id: int = int(event.get("target_id", 0))
    var skill_id: int = int(event.get("skill_id", 0))
    var skill_meta: Dictionary = _find_skill_meta(source_id, skill_id, network_provider)
    var event_type: int = int(event.get("event_type", 0))
    var value: int = int(event.get("value", 0))
    return {
        "id": "combo_%d_%d" % [source_id, target_id],
        "actor_id": source_id,
        "target_id": target_id,
        "display_name": str(skill_meta.get("display_name", "连击")),
        "skill_id": skill_id,
        "skill_visual_id": str(skill_meta.get("skill_visual_id", "")),
        "animation_key": str(skill_meta.get("animation_key", "")),
        # 反击/连击公告只表示追加动作开始；真正伤害由后续 EVENT_DAMAGE 扣血。
        "result_type": "damage" if event_type == EVENT_COMBO and value > 0 else "none",
        "value": value,
        "log_text": str(event.get("label", "")),
    }

static func _find_skill_meta(actor_id: int, skill_id: int, network_provider: BattleNetworkProvider) -> Dictionary:
    if skill_id == App.DEFAULT_BATTLE_SKILL_ID:
        return _basic_attack_meta(actor_id, network_provider)
    var actor: Dictionary = network_provider.find_actor_snapshot(actor_id)
    var skills_variant: Variant = actor.get("skills", [])
    if skills_variant is Array:
        for skill_variant: Variant in skills_variant:
            if skill_variant is Dictionary:
                var skill: Dictionary = skill_variant as Dictionary
                if int(skill.get("skill_id", 0)) == skill_id:
                    if bool(skill.get("is_basic_attack", false)):
                        return _basic_attack_meta(actor_id, network_provider)
                    return {
                        "display_name": str(skill.get("name", "")),
                        "skill_visual_id": str(skill.get("skill_visual_id", "")),
                        "animation_key": str(skill.get("animation_key", "")),
                    }
    return {"display_name": "技能", "skill_visual_id": "", "animation_key": ""}

## 普攻固定使用 slash.tres；人物单位优先展示已佩戴武器名。
static func _basic_attack_meta(actor_id: int, network_provider: BattleNetworkProvider) -> Dictionary:
    var display_name: String = "攻击"
    if network_provider != null:
        display_name = network_provider.get_basic_attack_display_name(actor_id)
    return {
        "display_name": display_name,
        "skill_visual_id": App.DEFAULT_BATTLE_SKILL_VISUAL_ID,
        "animation_key": App.DEFAULT_BATTLE_SKILL_VISUAL_ID,
    }

static func _resolve_hp_after(target_id: int, runtime_actors: Array[Dictionary], value: int, event_type: int) -> int:
    for actor: Dictionary in runtime_actors:
        if int(actor.get("actor_id", 0)) == target_id:
            return int(actor.get("hp", 0))
    if event_type == EVENT_HEAL:
        return value
    return 0
