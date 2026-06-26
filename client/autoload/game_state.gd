extends Node

# 会话相关字段变化后向外广播。
signal session_changed
# 世界快照变化后向外广播。
signal world_snapshot_changed
# 宠物与编队数据变化后向外广播。
signal pets_changed
# 背包数据变化后向外广播。
signal bag_changed
# 钱包数据变化后向外广播。
signal wallet_changed
# 任务快照变化后向外广播。
signal quests_changed
# 战斗状态变化后向外广播。
signal battle_changed
# 人物已佩戴装备变化后向外广播。
signal equipment_changed

# 当前登录态持有的访问令牌。
var access_jwt: String = ""
# 当前登录态持有的 WebSocket 鉴权令牌。
var ws_token: String = ""
# 当前 WebSocket 鉴权令牌的过期时间戳。
var ws_expire_at: int = 0
# 当前实时连接对应的会话标识。
var session_id: String = ""
# 当前实时连接断线重连使用的令牌。
var reconnect_token: String = ""
# 当前会话约定的心跳间隔秒数。
var heartbeat_sec: int = 0
# 标记当前实时连接是否已完成鉴权。
var is_ws_authenticated: bool = false
# 当前登录角色的玩家标识。
var player_id: int = 0
# 当前玩家的本地快照数据。
var player_snapshot: Dictionary = {}
# 当前场景的本地快照数据。
var scene_snapshot: Dictionary = {}
# 当前附近实体的本地快照索引表。
var nearby_entities: Dictionary = {}
# 当前拥有的宠物实例列表。
var pets: Array = []
# 当前编队中的宠物摘要列表。
var lineup: Array = []
# 当前背包物品列表。
var bag_items: Array = []
# 当前随身背包完整快照。
var bag_container: Dictionary = {}
# 当前仓库完整快照。
var warehouse_container: Dictionary = {}
# 当前钱包完整快照。
var wallet_snapshot: Dictionary = {}
# 当前任务快照列表。
var quests: Array = []
# 当前追踪任务的任务标识。
var tracked_quest_id: int = 0
# 当前战斗态完整快照数据。
var battle_state: Dictionary = {}
# 标记当前是否处于战斗中。
var is_in_battle: bool = false
# 当前地图暗雷遭遇配置，由进图/切图响应下发。
var wild_encounter_config: Dictionary = {}
# 当前人物已佩戴装备摘要列表。
var equipped_items: Array = []

# 清空当前会话和运行态数据，并广播会话变化。
func reset_session_state() -> void:
	access_jwt = ""
	ws_token = ""
	ws_expire_at = 0
	session_id = ""
	reconnect_token = ""
	heartbeat_sec = 0
	is_ws_authenticated = false
	player_id = 0
	reset_runtime_state()
	session_changed.emit()

# 清空世界、宠物、背包和战斗等运行态数据。
func reset_runtime_state() -> void:
	player_snapshot = {}
	scene_snapshot = {}
	nearby_entities = {}
	pets = []
	lineup = []
	bag_items = []
	bag_container = {}
	warehouse_container = {}
	wallet_snapshot = {}
	quests = []
	tracked_quest_id = 0
	battle_state = {}
	is_in_battle = false
	wild_encounter_config = {}
	equipped_items = []
	world_snapshot_changed.emit()
	pets_changed.emit()
	bag_changed.emit()
	wallet_changed.emit()
	quests_changed.emit()
	battle_changed.emit()
	equipment_changed.emit()

# 写入 HTTP 登录结果返回的会话基础信息。
func store_login_result(data: Dictionary) -> void:
	player_id = int(data.get("player_id", 0))
	access_jwt = str(data.get("access_jwt", ""))
	ws_token = str(data.get("ws_token", ""))
	ws_expire_at = int(data.get("ws_expire_at", 0))
	is_ws_authenticated = false

	# 提取登录结果中的玩家名称用于初始化角色快照。
	var player_name := str(data.get("player_name", ""))
	player_snapshot = {
		"player_id": player_id,
		"name": player_name,
	}
	session_changed.emit()

# 写入 WebSocket 鉴权成功后返回的实时会话信息。
func store_ws_session(data: Dictionary) -> void:
	session_id = str(data.get("session_id", ""))
	reconnect_token = str(data.get("reconnect_token", ""))
	heartbeat_sec = int(data.get("heartbeat_sec", 0))
	is_ws_authenticated = true
	session_changed.emit()

# 更新实时连接鉴权状态，并在失效时清空会话字段。
func set_ws_authenticated(authenticated: bool, preserve_reconnect_token: bool = false) -> void:
	is_ws_authenticated = authenticated
	if not authenticated:
		session_id = ""
		if not preserve_reconnect_token:
			reconnect_token = ""
		heartbeat_sec = 0
	session_changed.emit()

# 用服务端下发的世界快照重建本地场景、玩家和实体状态。
func set_world_snapshot(payload: Dictionary) -> void:
	# 提取世界快照中的场景结构。
	var scene_data: Variant = payload.get("scene", {})
	scene_snapshot = scene_data.duplicate(true) if scene_data is Dictionary else {}
	if payload.has("scene_id"):
		scene_snapshot["scene_id"] = payload.get("scene_id")
	if payload.has("scene_version"):
		scene_snapshot["scene_version"] = payload.get("scene_version")

	# 以现有玩家快照为基础合并新的权威玩家数据。
	var next_player := player_snapshot.duplicate(true)
	# 兼容 player/self 两种字段格式读取玩家结构。
	var player_data: Variant = payload.get("player", payload.get("self", {}))
	if player_data is Dictionary:
		# 合并玩家字段时保留本地已结算的经验，避免 ENTER_WORLD 用 0 覆盖刚战斗完的增长。
		var incoming_player: Dictionary = player_data.duplicate(true)
		var current_exp: int = int(next_player.get("exp", 0))
		if incoming_player.has("exp"):
			var incoming_exp: int = int(incoming_player.get("exp", 0))
			if incoming_exp <= 0 and current_exp > 0:
				incoming_player.erase("exp")
		next_player.merge(incoming_player, true)
	if player_id > 0 and not player_snapshot.has("player_id"):
		next_player["player_id"] = player_id
	if payload.has("self_pos"):
		# 提取世界快照中的角色权威坐标。
		var self_pos_variant: Variant = payload.get("self_pos", {})
		if self_pos_variant is Dictionary:
			next_player["x"] = float(self_pos_variant.get("x", next_player.get("x", 0.0)))
			next_player["y"] = float(self_pos_variant.get("y", next_player.get("y", 0.0)))
	player_snapshot = next_player

	if player_data is Dictionary and player_data.has("equipped_items"):
		var equipped_variant: Variant = player_data.get("equipped_items", [])
		if equipped_variant is Array:
			set_equipped_items(equipped_variant)

	nearby_entities = {}
	# 兼容 entities/nearby_entities 两种字段格式读取实体列表。
	var entities_variant: Variant = payload.get("entities", payload.get("nearby_entities", []))
	if entities_variant is Array:
		for entity_variant in entities_variant:
			if entity_variant is Dictionary and entity_variant.has("entity_id"):
				nearby_entities[int(entity_variant["entity_id"])] = entity_variant.duplicate(true)

	# 仅当快照显式携带 lineup 时才覆盖本地编队，避免切图 resync 把出战宠物清掉。
	if payload.has("lineup"):
		var lineup_variant: Variant = payload.get("lineup", [])
		lineup = lineup_variant.duplicate(true) if lineup_variant is Array else []

	var wild_encounter_variant: Variant = payload.get("wild_encounter", {})
	wild_encounter_config = wild_encounter_variant.duplicate(true) if wild_encounter_variant is Dictionary else {}
	if not wild_encounter_config.is_empty():
		scene_snapshot["wild_encounter"] = wild_encounter_config.duplicate(true)

	world_snapshot_changed.emit()
	pets_changed.emit()

# 把客户端当前场景坐标写入玩家快照，供开战上报与服务端 return_pos 对齐。
func sync_player_scene_position(scene_position: Vector2) -> void:
	var next_player: Dictionary = player_snapshot.duplicate(true)
	next_player["x"] = scene_position.x
	next_player["y"] = scene_position.y
	player_snapshot = next_player

# 向附近实体表中写入一个新实体。
func add_entity(entity: Dictionary) -> void:
	if not entity.has("entity_id"):
		return

	nearby_entities[int(entity["entity_id"])] = entity.duplicate(true)
	world_snapshot_changed.emit()

# 从附近实体表中删除指定实体。
func remove_entity(entity_id: int) -> void:
	nearby_entities.erase(entity_id)
	world_snapshot_changed.emit()

# 把实体移动推送合并到本地实体快照中。
func apply_entity_move(payload: Dictionary) -> void:
	# 读取本次移动对应的实体标识。
	var entity_id: int = int(payload.get("entity_id", 0))
	if entity_id == 0:
		return

	# 取出本地缓存中的实体快照。
	var entity: Dictionary = nearby_entities.get(entity_id, {})
	# 兼容 to_pos/position 两种字段格式读取目标坐标。
	var position_variant: Variant = payload.get("to_pos", payload.get("position", {}))
	if position_variant is Dictionary:
		entity["pos"] = position_variant.duplicate(true)
		entity["x"] = float(position_variant.get("x", entity.get("x", 0.0)))
		entity["y"] = float(position_variant.get("y", entity.get("y", 0.0)))
	else:
		entity["x"] = float(payload.get("x", entity.get("x", 0.0)))
		entity["y"] = float(payload.get("y", entity.get("y", 0.0)))
	nearby_entities[entity_id] = entity

	if entity_id == player_id:
		player_snapshot["x"] = entity["x"]
		player_snapshot["y"] = entity["y"]

	world_snapshot_changed.emit()

# 整体替换宠物列表和编队摘要，并刷新编队标记。
func set_pets(next_pets: Array, next_lineup: Array = []) -> void:
	pets = next_pets.duplicate(true)
	lineup = next_lineup.duplicate(true)
	_sync_pet_lineup_flags()
	pets_changed.emit()

# 仅替换编队摘要，并刷新宠物上的编队标记。
func set_lineup(next_lineup: Array) -> void:
	lineup = next_lineup.duplicate(true)
	_sync_pet_lineup_flags()
	pets_changed.emit()

# 把单只宠物的最新权威状态合并进本地宠物列表。
func upsert_pet(pet: Dictionary) -> void:
	# 读取当前宠物实例的唯一标识。
	var pet_uid: int = int(pet.get("pet_uid", 0))
	if pet_uid == 0:
		return

	# 复制当前宠物数据，避免外部引用直接共享。
	var next_pet := pet.duplicate(true)
	for index in pets.size():
		# 读取当前遍历到的本地宠物数据。
		var current: Variant = pets[index]
		if current is Dictionary and int(current.get("pet_uid", 0)) == pet_uid:
			pets[index] = next_pet
			_sync_pet_lineup_flags()
			pets_changed.emit()
			return

	pets.append(next_pet)
	_sync_pet_lineup_flags()
	pets_changed.emit()

# 按当前编队摘要重写宠物列表中的 in_lineup 标记。
func _sync_pet_lineup_flags() -> void:
	if pets.is_empty():
		return

	# 保存当前编队中全部宠物唯一标识的查找表。
	var lineup_pet_uids := {}
	for lineup_item_variant in lineup:
		if lineup_item_variant is Dictionary:
			# 读取当前编队项对应的宠物唯一标识。
			var lineup_pet_uid: int = int(lineup_item_variant.get("pet_uid", 0))
			if lineup_pet_uid != 0:
				lineup_pet_uids[lineup_pet_uid] = true

	for index in pets.size():
		# 读取当前遍历到的本地宠物数据。
		var current: Variant = pets[index]
		if current is Dictionary:
			# 复制当前宠物数据以便安全写回编队标记。
			var next_pet: Dictionary = current.duplicate(true)
			# 读取当前宠物实例的唯一标识。
			var pet_uid: int = int(next_pet.get("pet_uid", 0))
			next_pet["in_lineup"] = lineup_pet_uids.has(pet_uid)
			pets[index] = next_pet

# 整体替换背包物品列表。
func set_bag_items(next_items: Array) -> void:
	var normalized_items := next_items.duplicate(true)
	bag_items = normalized_items
	if bag_container.is_empty():
		bag_container = {
			"container_type": "bag",
			"capacity": maxi(normalized_items.size(), 30),
			"max_capacity": 300,
			"used_slots": normalized_items.size(),
			"items": normalized_items,
		}
	else:
		bag_container["items"] = normalized_items
		bag_container["used_slots"] = normalized_items.size()
	bag_changed.emit()

# 用服务端权威容器快照整体刷新随身背包或仓库。
func set_container_snapshot(container: Dictionary) -> void:
	var normalized_container := container.duplicate(true)
	var container_type := str(normalized_container.get("container_type", "bag"))
	var items_variant: Variant = normalized_container.get("items", [])
	var normalized_items: Array = items_variant.duplicate(true) if items_variant is Array else []
	normalized_container["items"] = normalized_items
	normalized_container["used_slots"] = int(normalized_container.get("used_slots", normalized_items.size()))

	if container_type == "warehouse":
		warehouse_container = normalized_container
	else:
		bag_container = normalized_container
		bag_items = normalized_items
	bag_changed.emit()

# 用服务端权威钱包快照整体刷新本地货币状态。
func set_wallet_snapshot(snapshot: Dictionary) -> void:
	wallet_snapshot = snapshot.duplicate(true)
	# 旧 UI 仍有部分文案从 player_snapshot.gold 读取，这里同步一个兼容值，
	# 直到全部价格与货币展示都切换到 wallet_snapshot 为止。
	player_snapshot["gold"] = int(wallet_snapshot.get("gold", 0))
	bag_changed.emit()
	wallet_changed.emit()

# 整体替换任务快照列表，并同步当前追踪任务。
func set_quests(next_quests: Array, next_tracked_quest_id: int = 0) -> void:
	quests = next_quests.duplicate(true)
	tracked_quest_id = next_tracked_quest_id if next_tracked_quest_id != 0 else _pick_default_tracked_quest_id()
	quests_changed.emit()

# 把单个任务快照的最新权威状态合并进本地任务列表。
func upsert_quest(quest: Dictionary) -> void:
	var quest_id: int = int(quest.get("quest_id", 0))
	if quest_id == 0:
		return

	var next_quest := quest.duplicate(true)
	for index in quests.size():
		var current: Variant = quests[index]
		if current is Dictionary and int(current.get("quest_id", 0)) == quest_id:
			quests[index] = next_quest
			if bool(next_quest.get("tracked", false)):
				tracked_quest_id = quest_id
			elif tracked_quest_id == 0:
				tracked_quest_id = _pick_default_tracked_quest_id()
			quests_changed.emit()
			return

	quests.append(next_quest)
	if bool(next_quest.get("tracked", false)) or tracked_quest_id == 0:
		tracked_quest_id = quest_id if bool(next_quest.get("tracked", false)) else _pick_default_tracked_quest_id()
	quests_changed.emit()

# 删除指定任务快照，并在需要时重选追踪任务。
func remove_quest(quest_id: int) -> void:
	for index in range(quests.size() - 1, -1, -1):
		var current: Variant = quests[index]
		if current is Dictionary and int(current.get("quest_id", 0)) == quest_id:
			quests.remove_at(index)
	if tracked_quest_id == quest_id:
		tracked_quest_id = _pick_default_tracked_quest_id()
	quests_changed.emit()

# 更新当前追踪任务标识，并同步任务列表中的 tracked 字段。
func set_tracked_quest(quest_id: int) -> void:
	tracked_quest_id = quest_id
	for index in quests.size():
		var current: Variant = quests[index]
		if current is Dictionary:
			var next_quest: Dictionary = current.duplicate(true)
			next_quest["tracked"] = int(next_quest.get("quest_id", 0)) == quest_id
			quests[index] = next_quest
	quests_changed.emit()

# 返回当前追踪任务的快照；若不存在则返回空字典。
func tracked_quest() -> Dictionary:
	for quest_variant in quests:
		if quest_variant is Dictionary and int(quest_variant.get("quest_id", 0)) == tracked_quest_id:
			return quest_variant
	return {}

# 把单个物品的最新权威状态合并进本地背包列表。
func upsert_bag_item(item: Dictionary) -> void:
	# 新版容器格子应优先按 slot_index 合并，避免同模板多堆叠或装备实例互相覆盖。
	var slot_index: int = int(item.get("slot_index", 0))
	var item_id: int = int(item.get("item_id", 0))
	if slot_index == 0 and item_id == 0:
		return

	for index in bag_items.size():
		# 读取当前遍历到的本地物品数据。
		var current: Variant = bag_items[index]
		if current is Dictionary and (
			int(current.get("slot_index", 0)) == slot_index or
			(slot_index == 0 and int(current.get("item_id", 0)) == item_id)
		):
			bag_items[index] = item.duplicate(true)
			if not bag_container.is_empty():
				bag_container["items"] = bag_items.duplicate(true)
				bag_container["used_slots"] = bag_items.size()
			bag_changed.emit()
			return

	bag_items.append(item.duplicate(true))
	if not bag_container.is_empty():
		bag_container["items"] = bag_items.duplicate(true)
		bag_container["used_slots"] = bag_items.size()
	bag_changed.emit()

# 按服务端增量推送把指定容器的多个格子更新合并进本地状态。
func apply_container_updates(container_type: String, updates: Array, capacity: int = 0, max_capacity: int = 0, used_slots: int = -1) -> void:
	var target_container: Dictionary = warehouse_container.duplicate(true) if container_type == "warehouse" else bag_container.duplicate(true)
	var items_variant: Variant = target_container.get("items", [])
	var target_items: Array = items_variant.duplicate(true) if items_variant is Array else []

	for update_variant in updates:
		if update_variant is not Dictionary:
			continue
		var update: Dictionary = update_variant
		var slot_index: int = int(update.get("slot_index", 0))
		if slot_index == 0:
			continue
		var deleted: bool = bool(update.get("deleted", false))
		var replaced := false
		for index in range(target_items.size() - 1, -1, -1):
			var current: Variant = target_items[index]
			if current is Dictionary and int(current.get("slot_index", 0)) == slot_index:
				if deleted:
					target_items.remove_at(index)
				else:
					var item_variant: Variant = update.get("item", {})
					if item_variant is Dictionary:
						target_items[index] = item_variant.duplicate(true)
				replaced = true
				break
		if not deleted and not replaced:
			var append_item_variant: Variant = update.get("item", {})
			if append_item_variant is Dictionary:
				target_items.append(append_item_variant.duplicate(true))

	target_container["container_type"] = container_type
	if capacity > 0:
		target_container["capacity"] = capacity
	if max_capacity > 0:
		target_container["max_capacity"] = max_capacity
	target_container["items"] = target_items
	target_container["used_slots"] = used_slots if used_slots >= 0 else target_items.size()

	if container_type == "warehouse":
		warehouse_container = target_container
	else:
		bag_container = target_container
		bag_items = target_items
	bag_changed.emit()

# 按当前任务列表优先选择显式追踪任务，否则回落到首个可展示任务。
func _pick_default_tracked_quest_id() -> int:
	for quest_variant in quests:
		if quest_variant is Dictionary and bool(quest_variant.get("tracked", false)):
			return int(quest_variant.get("quest_id", 0))
	for quest_variant in quests:
		if quest_variant is Dictionary:
			var state := str(quest_variant.get("state", ""))
			if state == "AVAILABLE" or state == "ACCEPTED" or state == "READY_TO_SUBMIT":
				return int(quest_variant.get("quest_id", 0))
	for quest_variant in quests:
		if quest_variant is Dictionary:
			return int(quest_variant.get("quest_id", 0))
	return 0

# 写入战斗快照，并按 active 标记当前是否处于战斗态。
# 结算包同样基于上一份快照合并，避免 4013 到达时把 4012 的 events 清掉导致无法演出。
func set_battle_state(next_state: Dictionary, active: bool = true) -> void:
	var merged_state: Dictionary = battle_state.duplicate(true)
	merged_state.merge(next_state, true)
	if next_state.has("actors") and not merged_state.has("actors"):
		merged_state["actors"] = []
	battle_state = merged_state
	is_in_battle = active
	battle_changed.emit()

# 写入服务端下发的人物已佩戴装备摘要。
func set_equipped_items(items: Array) -> void:
	var next_items: Array = []
	for item_variant: Variant in items:
		if item_variant is Dictionary:
			next_items.append((item_variant as Dictionary).duplicate(true))
	equipped_items = next_items
	equipment_changed.emit()

# 合并服务端下发的玩家权威快照字段。
func merge_player_snapshot(player_data: Dictionary) -> void:
	if player_data.is_empty():
		return
	var next_player: Dictionary = player_snapshot.duplicate(true)
	next_player.merge(player_data, true)
	player_snapshot = next_player
	world_snapshot_changed.emit()

# 把战斗结算里带回来的玩家成长结果合并到本地玩家快照。
func apply_battle_player_rewards(payload: Dictionary) -> void:
	var changed := false
	## 结算包在胜利时会带回 player_exp / reward_player_exp；这里优先采用权威总值，其次用增量合并。
	var level_up_count: int = int(payload.get("level_up_count", 0))
	var reward_player_exp: int = int(payload.get("reward_player_exp", 0))
	var reward_gold: int = int(payload.get("reward_gold", 0))
	var battle_won: bool = bool(payload.get("win", false))
	if payload.has("player_gold"):
		var next_gold: int = int(payload.get("player_gold", player_snapshot.get("gold", 0)))
		if next_gold > 0:
			player_snapshot["gold"] = next_gold
			changed = true
		elif battle_won and reward_gold > 0:
			var current_gold: int = int(player_snapshot.get("gold", 0))
			player_snapshot["gold"] = current_gold + reward_gold
			changed = true
	if payload.has("player_exp"):
		var next_exp: int = int(payload.get("player_exp", player_snapshot.get("exp", 0)))
		if next_exp > 0:
			player_snapshot["exp"] = next_exp
			changed = true
		elif battle_won and reward_player_exp > 0:
			var current_exp: int = int(player_snapshot.get("exp", 0))
			player_snapshot["exp"] = current_exp + reward_player_exp
			changed = true
	if payload.has("player_level"):
		var next_level: int = int(payload.get("player_level", player_snapshot.get("level", 0)))
		if next_level > 0 or level_up_count > 0:
			player_snapshot["level"] = next_level
			changed = true
	if payload.has("exp_to_next"):
		var next_exp_to_next: int = int(payload.get("exp_to_next", player_snapshot.get("exp_to_next", 0)))
		if next_exp_to_next > 0 or level_up_count > 0:
			player_snapshot["exp_to_next"] = next_exp_to_next
			changed = true
	if payload.has("free_attr_points"):
		var next_free_attr_points: int = int(payload.get("free_attr_points", player_snapshot.get("free_attr_points", 0)))
		if next_free_attr_points > 0 or level_up_count > 0:
			player_snapshot["free_attr_points"] = next_free_attr_points
			changed = true
	elif payload.has("attr_points_gained"):
		var gained: int = int(payload.get("attr_points_gained", 0))
		if gained > 0:
			var current_free: int = int(player_snapshot.get("free_attr_points", 0))
			player_snapshot["free_attr_points"] = current_free + gained
			changed = true
	if changed:
		world_snapshot_changed.emit()

# 把战斗结算里的宠物经验与升级摘要合并进本地宠物列表。
func apply_battle_pet_rewards(payload: Dictionary) -> void:
	var pet_rewards_variant: Variant = payload.get("pet_rewards", [])
	if pet_rewards_variant is not Array:
		return
	var changed: bool = false
	for pet_reward_variant: Variant in pet_rewards_variant:
		if pet_reward_variant is not Dictionary:
			continue
		var pet_reward: Dictionary = pet_reward_variant as Dictionary
		var pet_uid: int = int(pet_reward.get("pet_uid", 0))
		if pet_uid == 0:
			continue
		for index: int in range(pets.size()):
			var current_variant: Variant = pets[index]
			if current_variant is not Dictionary:
				continue
			var current: Dictionary = current_variant as Dictionary
			if int(current.get("pet_uid", 0)) != pet_uid:
				continue
			var next_pet: Dictionary = current.duplicate(true)
			if pet_reward.has("free_attr_points"):
				next_pet["free_attr_points"] = int(pet_reward.get("free_attr_points", 0))
			elif int(pet_reward.get("attr_points_gained", 0)) > 0:
				var current_free: int = int(next_pet.get("free_attr_points", 0))
				next_pet["free_attr_points"] = current_free + int(pet_reward.get("attr_points_gained", 0))
			if pet_reward.has("exp_to_next"):
				next_pet["exp_to_next"] = int(pet_reward.get("exp_to_next", 0))
			pets[index] = next_pet
			changed = true
			break
	if changed:
		pets_changed.emit()

# 返回当前激活战斗单位对应的快照数据。
func active_battle_actor(group_key: String = "allies") -> Dictionary:
	# 读取当前战斗回合轮到行动的单位标识。
	var target_actor_id: int = int(battle_state.get("active_actor_id", 0))
	# 读取当前战斗回合对应的出战宠物唯一标识。
	var target_pet_uid: int = int(battle_state.get("active_pet_uid", 0))
	# 读取指定阵营下的单位列表。
	var actors_variant: Variant = battle_state.get(group_key, [])
	if actors_variant is not Array:
		return {}

	for actor_variant in actors_variant:
		if actor_variant is Dictionary:
			if target_actor_id != 0 and int(actor_variant.get("actor_id", 0)) == target_actor_id:
				return actor_variant
			if target_pet_uid != 0 and int(actor_variant.get("pet_uid", 0)) == target_pet_uid:
				return actor_variant
	if not actors_variant.is_empty() and actors_variant[0] is Dictionary:
		return actors_variant[0]
	return {}

# 清空当前战斗状态并广播退出战斗。
func clear_battle_state() -> void:
	battle_state = {}
	is_in_battle = false
	battle_changed.emit()
