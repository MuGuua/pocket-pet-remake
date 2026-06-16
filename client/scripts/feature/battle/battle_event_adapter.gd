extends RefCounted
class_name BattleEventAdapter

## 把 4012 events[] 转成 Director 可播放的伪剧本结构。

const EVENT_USE_SKILL: int = 1
const EVENT_DAMAGE: int = 2
const EVENT_HEAL: int = 3
const EVENT_APPLY_STATUS: int = 4
const EVENT_DEFEAT: int = 7
const EVENT_COUNTER: int = 9
const EVENT_COMBO: int = 11

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
			EVENT_DAMAGE, EVENT_HEAL, EVENT_DEFEAT:
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
	return {
		"id": "action_%03d" % action_index,
		"actor_id": source_id,
		"action_type": "skill",
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
	runtime_actors: Array[Dictionary],
	event_type: int
) -> void:
	var target_id: int = int(event.get("target_id", 0))
	var value: int = int(event.get("value", 0))
	var hp_after: int = _resolve_hp_after(target_id, runtime_actors, value, event_type)
	var result_type: String = "heal" if event_type == EVENT_HEAL else "damage"
	if event_type == EVENT_DEFEAT:
		hp_after = 0
		result_type = "defeat"
	var label_text: String = str(event.get("label", ""))
	var is_crit: bool = label_text.find("暴击") != -1
	var targets: Array = action.get("targets", []) as Array
	targets.append({
		"target_id": target_id,
		"result_type": result_type,
		"value": value,
		"hp_after": hp_after,
		"log_text": label_text,
		"is_crit": is_crit,
		"hit_type": "crit" if is_crit else "normal",
	})
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
	return {
		"id": "combo_%d_%d" % [source_id, target_id],
		"actor_id": source_id,
		"target_id": target_id,
		"display_name": str(skill_meta.get("display_name", "连击")),
		"skill_id": skill_id,
		"skill_visual_id": str(skill_meta.get("skill_visual_id", "")),
		"result_type": "damage",
		"value": int(event.get("value", 0)),
		"hp_after": 0,
		"log_text": str(event.get("label", "")),
	}

static func _find_skill_meta(actor_id: int, skill_id: int, network_provider: BattleNetworkProvider) -> Dictionary:
	var actor: Dictionary = network_provider.find_actor_snapshot(actor_id)
	var skills_variant: Variant = actor.get("skills", [])
	if skills_variant is Array:
		for skill_variant: Variant in skills_variant:
			if skill_variant is Dictionary:
				var skill: Dictionary = skill_variant as Dictionary
				if int(skill.get("skill_id", 0)) == skill_id:
					return {
						"display_name": str(skill.get("name", "")),
						"skill_visual_id": str(skill.get("skill_visual_id", "")),
						"animation_key": str(skill.get("animation_key", "")),
					}
	return {"display_name": "技能", "skill_visual_id": "", "animation_key": ""}

static func _resolve_hp_after(target_id: int, runtime_actors: Array[Dictionary], value: int, event_type: int) -> int:
	for actor: Dictionary in runtime_actors:
		if int(actor.get("actor_id", 0)) == target_id:
			return int(actor.get("hp", 0))
	if event_type == EVENT_HEAL:
		return value
	return 0
