extends Node
class_name BattleNetworkProvider

## 从 GameState.battle_state 读取权威战斗快照，替代 Demo JSON 数据源。

## 返回当前战斗快照副本。
func get_battle_state() -> Dictionary:
	if GameState.battle_state.is_empty():
		return {}
	return GameState.battle_state.duplicate(true)

## 返回战斗 ID。
func get_battle_id() -> int:
	return int(get_battle_state().get("battle_id", 0))

## 返回当前回合数。
func get_round() -> int:
	return int(get_battle_state().get("round", 1))

## 返回当前阶段字符串。
func get_phase() -> String:
	return str(get_battle_state().get("phase", ""))

## 是否开启自动战斗。
func is_auto_battle_enabled() -> bool:
	return bool(get_battle_state().get("auto_battle_enabled", false))

## 返回待提交动作的己方 actor_id 列表。
func get_pending_actor_ids() -> Array[int]:
	var result: Array[int] = []
	var pending_variant: Variant = get_battle_state().get("pending_actor_ids", [])
	if pending_variant is Array:
		for actor_id_variant: Variant in pending_variant:
			var actor_id: int = int(actor_id_variant)
			if actor_id > 0:
				result.append(actor_id)
	return result

## 把 4011 的 allies/enemies 转成 Director 可消费的单位初始化数组。
func get_initial_units() -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	var state: Dictionary = get_battle_state()
	var allies_variant: Variant = state.get("allies", [])
	if allies_variant is Array:
		for actor_variant: Variant in allies_variant:
			if actor_variant is Dictionary:
				result.append(_build_unit_data(actor_variant as Dictionary, true))
	var enemies_variant: Variant = state.get("enemies", [])
	if enemies_variant is Array:
		for actor_variant: Variant in enemies_variant:
			if actor_variant is Dictionary:
				result.append(_build_unit_data(actor_variant as Dictionary, false))
	return result

## 按 actor_id 在 allies/enemies 中查找完整单位快照。
func find_actor_snapshot(actor_id: int) -> Dictionary:
	if actor_id <= 0:
		return {}
	var state: Dictionary = get_battle_state()
	for group_key: String in ["allies", "enemies"]:
		var actors_variant: Variant = state.get(group_key, [])
		if actors_variant is Array:
			for actor_variant: Variant in actors_variant:
				if actor_variant is Dictionary:
					var actor: Dictionary = actor_variant as Dictionary
					if int(actor.get("actor_id", 0)) == actor_id:
						return actor.duplicate(true)
	return {}

## 返回 4012 actors 精简快照列表。
func get_runtime_actors() -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	var actors_variant: Variant = get_battle_state().get("actors", [])
	if actors_variant is Array:
		for actor_variant: Variant in actors_variant:
			if actor_variant is Dictionary:
				result.append((actor_variant as Dictionary).duplicate(true))
	return result

## 返回当前状态推送中的事件列表。
func get_events() -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	var events_variant: Variant = get_battle_state().get("events", [])
	if events_variant is Array:
		for event_variant: Variant in events_variant:
			if event_variant is Dictionary:
				result.append((event_variant as Dictionary).duplicate(true))
	return result

## 判断 actor 是否属于己方。
func is_ally_actor(actor_id: int) -> bool:
	var allies_variant: Variant = get_battle_state().get("allies", [])
	if allies_variant is Array:
		for actor_variant: Variant in allies_variant:
			if actor_variant is Dictionary:
				if int((actor_variant as Dictionary).get("actor_id", 0)) == actor_id:
					return true
	return false

func _build_unit_data(actor: Dictionary, is_ally: bool) -> Dictionary:
	var lineup_index: int = int(actor.get("lineup_index", 0))
	var unit_class: int = int(actor.get("unit_class", 0))
	return {
		"actor_id": int(actor.get("actor_id", 0)),
		"name": str(actor.get("name", "")),
		"type": _resolve_unit_type(actor, is_ally, unit_class),
		"hp": int(actor.get("hp", 0)),
		"max_hp": int(actor.get("hp_max", 0)),
		"skin_id": str(actor.get("skin_id", "")),
		"position": BattleFormationMapper.resolve_position_key(is_ally, lineup_index),
		"skills": _normalize_skills(actor.get("skills", [])),
		"items": [],
	}

func _resolve_unit_type(actor: Dictionary, is_ally: bool, unit_class: int) -> String:
	if not is_ally:
		return "enemy"
	match unit_class:
		1:
			return "player"
		2:
			return "pet"
		_:
			return "pet"

func _normalize_skills(skills_variant: Variant) -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	if not skills_variant is Array:
		return result
	for skill_variant: Variant in skills_variant:
		if not skill_variant is Dictionary:
			continue
		var skill: Dictionary = skill_variant as Dictionary
		result.append({
			"skill_id": int(skill.get("skill_id", 0)),
			"display_name": str(skill.get("name", skill.get("display_name", ""))),
			"skill_visual_id": str(skill.get("skill_visual_id", "")),
			"animation_key": str(skill.get("animation_key", "")),
			"target_count": int(skill.get("target_count", 1)),
			"target_side": _target_side_from_type(str(skill.get("target_type", "enemy_single"))),
		})
	return result

func _target_side_from_type(target_type: String) -> String:
	match target_type:
		"ally_single", "ally_all", "ally_multi", "self":
			return "ally"
		"enemy_all", "enemy_multi", "enemy_single":
			return "enemy"
		_:
			return "enemy"
