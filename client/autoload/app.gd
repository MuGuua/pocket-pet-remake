extends Node

# 应用启动编排完成后向外广播。
signal bootstrapped
# HTTP 登录成功后向外广播完整响应。
signal login_succeeded(response: Dictionary)
# HTTP 登录失败后向外广播错误信息。
signal login_failed(message: String)
# WebSocket 鉴权成功后向外广播会话载荷。
signal session_authenticated(payload: Dictionary)
# 收到通用提示信息后向外广播提示文本。
signal notice_received(message: String)
# 收到服务端 ERROR_PUSH 后向外广播，供主场景弹出信息模态。
signal server_error_received(error_code: int, message: String)
# 收到强制下线信息后向外广播原因。
signal kicked(reason: String)
# 断线重连状态发生变化后向外广播。
signal reconnect_state_changed(in_progress: bool)
# 收到服务端请求结果或关键推送后向外广播统一日志文本。
signal server_result_logged(message: String)
# 由 App 发起的请求收到成功/失败回包后向外广播，供带 loading 的 UI 等待完成。
signal request_finished(request_cmd: int, seq: int, succeeded: bool, response_cmd: int, payload: Dictionary)

# 默认演示账号名。
const DEFAULT_ACCOUNT: String = "demo"
# 默认演示账号密码。
const DEFAULT_PASSWORD: String = "demo123"
# 默认战斗动作使用的技能标识。
const DEFAULT_BATTLE_SKILL_ID: int = 1001
# 普攻固定绑定的 slash.tres 视觉资源标识。
const DEFAULT_BATTLE_SKILL_VISUAL_ID: String = "slash"
## 是否打印背包完整 JSON 载荷；默认关闭，避免 BAG_LIST_RESP 物品较多时刷爆 Godot 输出窗口。
const ENABLE_VERBOSE_BAG_PAYLOAD_LOGS: bool = false

# 标记当前单例是否已经完成初始化绑定。
var _bootstrapped: bool = false
# 生成战斗类操作标识时使用的自增计数器。
var _next_battle_op_id: int = 1
# 标记当前是否正在尝试断线重连。
var _reconnect_in_progress: bool = false
# 记录当前仍在等待回包的请求序列号与命令号映射，供加载态 UI 做精确匹配。
var _pending_request_cmds_by_seq: Dictionary = {}
# 同一个请求命令可能并发发送，因此还要保留按命令分组的序列队列。
var _pending_request_seqs_by_cmd: Dictionary = {}

# 让应用单例在运行期间持续参与帧循环。
func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS

# 初始化应用层事件编排，并注册全局协议入口。
func bootstrap() -> void:
	if _bootstrapped:
		return

	_bootstrapped = true
	NetClient.dev_message_received.connect(_on_dev_message_received)
	NetClient.websocket_opened.connect(_on_websocket_opened)
	NetClient.websocket_closed.connect(_on_websocket_closed)
	MessageRouter.register_handler(CommandIds.WS_AUTH_RESP, Callable(self, "_on_ws_auth_response"))
	MessageRouter.register_handler(CommandIds.RECONNECT_RESP, Callable(self, "_on_reconnect_response"))
	MessageRouter.register_handler(CommandIds.HEARTBEAT_RESP, Callable(self, "_on_heartbeat_response"))
	MessageRouter.register_handler(CommandIds.FORCE_OFFLINE_PUSH, Callable(self, "_on_force_offline_push"))
	MessageRouter.register_handler(CommandIds.ERROR_PUSH, Callable(self, "_on_error_push"))
	MessageRouter.register_handler(CommandIds.NOTICE_PUSH, Callable(self, "_on_notice_push"))
	MessageRouter.register_handler(CommandIds.KICKOUT_PUSH, Callable(self, "_on_kickout_push"))
	bootstrapped.emit()

# 执行 HTTP 登录流程，并把成功结果写入全局会话状态。
func login(account: String, password: String) -> Dictionary:
	# 等待 HTTP 登录接口返回完整响应。
	var response: Dictionary = await HttpClient.login(account, password)
	_emit_server_result_log(0, _format_http_result_log("/api/v1/auth/login", response))
	# 读取当前 HTTP 响应状态码。
	var code: int = int(response.get("code", 0))
	if code != 200:
		login_failed.emit(str(response.get("msg", "login failed")))
		return response

	var data_variant: Variant = response.get("data", {})
	# 提取登录成功返回的数据体。
	var data: Dictionary = data_variant if data_variant is Dictionary else {}
	# 规范化登录结果数据体为字典结构。
	GameState.store_login_result(data)
	login_succeeded.emit(response)
	return response

# 执行 HTTP 注册流程，并返回服务端创建的新角色摘要。
func register_account(account: String, password: String, gender: String) -> Dictionary:
	var response: Dictionary = await HttpClient.register_account(account, password, gender)
	_emit_server_result_log(0, _format_http_result_log("/api/v1/auth/register", response))
	return response

func login_with_demo_account() -> Dictionary:
# 使用预设演示账号执行登录流程。
	return await login(DEFAULT_ACCOUNT, DEFAULT_PASSWORD)

func connect_ws() -> int:
# 发起到服务端的 WebSocket 连接。
	return NetClient.connect_to_server()

func authenticate_ws() -> void:
# 使用当前保存的 `ws_token` 发起实时连接鉴权。
	if _reconnect_in_progress and not GameState.reconnect_token.is_empty():
		_send_command(
			CommandIds.RECONNECT_REQ,
			{
				"reconnect_token": GameState.reconnect_token,
				"battle_id": int(GameState.battle_state.get("battle_id", 0)),
				"last_frame": int(GameState.battle_state.get("frame", GameState.battle_state.get("battle_version", 0))),
			}
		)
		return
	if GameState.ws_token.is_empty():
		push_warning("Missing ws_token. Login before websocket auth.")
		return


	_send_command(
		CommandIds.WS_AUTH_REQ,
		{
			"ws_token": GameState.ws_token,
			"client_version": "godot-4.5-dev",
		}
	)

# 请求进入当前服务端权威世界场景。
func enter_world() -> int:
	return _send_command(CommandIds.ENTER_WORLD_REQ, {})

## 人物状态面板打开时只拉取人物权威属性，不重新进入世界，也不加载背包、钱包、宠物或任务。
func refresh_player_status() -> int:
	return _send_command(CommandIds.PLAYER_PROFILE_REQ, {})

# 请求刷新当前玩家的宠物列表摘要；完整属性由单只宠物详情请求拉取。
func request_pet_list() -> int:
	return _send_command(CommandIds.PET_LIST_REQ, {})

# 提交玩家属性点分配请求。
func request_allocate_attr_points(strength: int = 0, vitality: int = 0, agility: int = 0, mind: int = 0) -> int:
	return _send_command(
		CommandIds.PLAYER_ALLOCATE_ATTR_REQ,
		{
			"strength": strength,
			"vitality": vitality,
			"agility": agility,
			"mind": mind,
		}
	)

# 提交单只宠物的自由属性点分配请求。
func request_pet_allocate_attr_points(
	pet_uid: int,
	hp: int = 0,
	atk: int = 0,
	spd: int = 0,
	mana: int = 0,
	def_value: int = 0
) -> int:
	return _send_command(
		CommandIds.PET_ALLOCATE_ATTR_REQ,
		{
			"pet_uid": pet_uid,
			"hp": hp,
			"atk": atk,
			"spd": spd,
			"mana": mana,
			"def": def_value,
		}
	)

# 提交完整编队宠物唯一标识列表。
func set_pet_lineup(pet_uids: Array[int]) -> void:
	_send_command(
		CommandIds.PET_LINEUP_SET_REQ,
		{
			"op_id": _take_battle_op_id(),
			"pet_uids": pet_uids,
		}
	)

# 拉取单只宠物完整属性和技能分槽（含法宝技 skill_id）。
func request_pet_skill_detail(pet_uid: int) -> int:
	return _send_command(
		CommandIds.PET_SKILL_DETAIL_REQ,
		{
			"pet_uid": pet_uid,
		}
	)

# 从背包装备法宝到宠物指定法宝槽。
func request_pet_artifact_equip(pet_uid: int, slot_index: int, bag_slot_index: int) -> int:
	return _send_command(
		CommandIds.PET_ARTIFACT_EQUIP_REQ,
		{
			"pet_uid": pet_uid,
			"slot_index": slot_index,
			"container_type": "bag",
			"bag_slot_index": bag_slot_index,
		}
	)

# 卸下宠物指定法宝槽上的技能。
func request_pet_artifact_unequip(pet_uid: int, slot_index: int) -> int:
	return _send_command(
		CommandIds.PET_ARTIFACT_UNEQUIP_REQ,
		{
			"pet_uid": pet_uid,
			"slot_index": slot_index,
		}
	)

# 拉取当前人物已佩戴装备列表。
func request_player_equipment_list() -> int:
	return _send_command(CommandIds.PLAYER_EQUIPMENT_LIST_REQ, {})

# 从背包指定格子佩戴人物装备。
func request_player_equip(bag_slot_index: int, container_type: String = "bag") -> int:
	return _send_command(
		CommandIds.PLAYER_EQUIP_REQ,
		{
			"container_type": container_type,
			"bag_slot_index": bag_slot_index,
		}
	)

# 卸下人物指定槽位装备。
func request_player_unequip(equip_slot: String, container_type: String = "bag") -> int:
	return _send_command(
		CommandIds.PLAYER_UNEQUIP_REQ,
		{
			"equip_slot": equip_slot,
			"container_type": container_type,
		}
	)

# 强化指定 item_uid 的人物装备实例；cost_item_id 为玩家选中的强化材料。
func request_player_equipment_enhance(item_uid: String, cost_item_id: int = 0) -> int:
	var payload: Dictionary = {
		"item_uid": item_uid,
	}
	if cost_item_id > 0:
		payload["cost_item_id"] = cost_item_id
	return _send_command(
		CommandIds.PLAYER_EQUIPMENT_ENHANCE_REQ,
		payload
	)

# 修复指定 item_uid 的损坏人物装备实例。
func request_player_equipment_repair(item_uid: String) -> int:
	return _send_command(
		CommandIds.PLAYER_EQUIPMENT_REPAIR_REQ,
		{
			"item_uid": item_uid,
		}
	)

# 请求刷新当前玩家的背包摘要。
# page/category/page_size 仅对新版背包 UI 生效；旧调用不传时默认仍返回第 1 页、28 条、全部分类。
func request_bag_list(page: int = 1, page_size: int = 28, category: String = "all") -> int:
	return _send_command(
		CommandIds.BAG_LIST_REQ,
		{
			"container_type": "bag",
			"page": page,
			"page_size": page_size,
			"category": category,
		}
	)

# 请求刷新指定容器的完整快照；当前主要用于仓库等非随身容器。
func request_container_list(container_type: String) -> int:
	return _send_command(
		CommandIds.CONTAINER_LIST_REQ,
		{
			"container_type": container_type,
		}
	)

# 请求刷新当前玩家的钱包快照。
func request_wallet() -> int:
	return _send_command(CommandIds.WALLET_QUERY_REQ, {})

# 请求主动使用当前容器格子中的物品。
# target_pet_uid 在 target_type=pet_single 的道具（治疗、神符解锁等）上必填。
# target_item_uid 在 target_type=equipment_single 的道具（修理、强化相关）上必填。
func request_use_item(container_type: String, slot_index: int, quantity: int = 1, target_pet_uid: int = 0, target_item_uid: String = "") -> int:
	var payload: Dictionary = {
		"container_type": container_type,
		"slot_index": slot_index,
		"quantity": quantity,
	}
	if target_pet_uid > 0:
		payload["target_pet_uid"] = target_pet_uid
	var normalized_item_uid: String = target_item_uid.strip_edges()
	if not normalized_item_uid.is_empty():
		payload["target_item_uid"] = normalized_item_uid
	return _send_command(CommandIds.USE_ITEM_REQ, payload)

# 请求丢弃当前容器格子中的物品；实例化物品应传 item_uid，可堆叠物品可传部分 quantity。
func request_drop_item(container_type: String, slot_index: int, quantity: int = 1, item_uid: String = "") -> int:
	var payload: Dictionary = {
		"container_type": container_type,
		"slot_index": slot_index,
		"quantity": quantity,
	}
	var normalized_uid: String = item_uid.strip_edges()
	if not normalized_uid.is_empty():
		payload["item_uid"] = normalized_uid
	return _send_command(CommandIds.DROP_ITEM_REQ, payload)

# 请求把背包指定格子的部分或全部物品存入仓库。
func request_bag_to_warehouse(entity_id: int, from_slot_index: int, quantity: int) -> int:
	return _send_command(
		CommandIds.BAG_TO_WAREHOUSE_REQ,
		{
			"entity_id": entity_id,
			"from_slot_index": from_slot_index,
			"quantity": quantity,
		}
	)

# 请求把仓库指定格子的部分或全部物品取回背包。
func request_warehouse_to_bag(entity_id: int, from_slot_index: int, quantity: int) -> int:
	return _send_command(
		CommandIds.WAREHOUSE_TO_BAG_REQ,
		{
			"entity_id": entity_id,
			"from_slot_index": from_slot_index,
			"quantity": quantity,
		}
	)

# 请求刷新当前玩家的任务列表。
func request_quest_list() -> int:
	return _send_command(CommandIds.QUEST_LIST_REQ, {})

# 请求接取指定任务。
func accept_quest(quest_id: int, npc_id: int = 0) -> int:
	return _send_command(CommandIds.QUEST_ACCEPT_REQ, {"quest_id": quest_id, "npc_id": npc_id})

# 请求提交指定任务；NPC 交付任务时同时带上提交 NPC。
func submit_quest(quest_id: int, npc_id: int = 0) -> int:
	return _send_command(CommandIds.QUEST_SUBMIT_REQ, {"quest_id": quest_id, "npc_id": npc_id})

# 请求切换当前追踪任务。
func track_quest(quest_id: int) -> int:
	return _send_command(CommandIds.QUEST_TRACK_REQ, {"quest_id": quest_id})

## 通知服务端指定场景剧情已经播放完毕，由服务端落库剧情 flag 并执行任务/NPC 解锁。
func ack_scene_trigger(trigger_code: String) -> int:
	return _send_command(CommandIds.SCENE_TRIGGER_ACK_REQ, {"trigger_code": trigger_code})

# 向服务端提交与指定实体的交互意图。
func request_interact(entity_id: int) -> int:
	return _send_command(
		CommandIds.INTERACT_REQ,
		{
			"entity_id": entity_id,
			"self_pos": _build_battle_self_pos_payload(),
		}
	)

# 客户端本地判定暗雷触发后，向服务端上报并请求权威开战。
func request_wild_encounter(scene_id: int, move_seq: int) -> void:
	_send_command(
		CommandIds.WILD_ENCOUNTER_REQ,
		{
			"scene_id": scene_id,
			"move_seq": move_seq,
			"self_pos": _build_battle_self_pos_payload(),
		}
	)

# 向服务端提交 NPC 菜单拉取请求。
func request_npc_menu(entity_id: int) -> int:
	return _send_command(
		CommandIds.NPC_MENU_REQ,
		{
			"entity_id": entity_id,
		}
	)

## 一次请求当前场景全部 NPC 的动态菜单；响应在后台写入地图缓存，不阻塞玩家进入地图。
func request_npc_menu_batch(scene_id: int) -> int:
	return _send_command(
		CommandIds.NPC_MENU_BATCH_REQ,
		{
			"scene_id": scene_id,
		}
	)

# 向服务端提交 NPC 菜单项执行请求。
func request_npc_action(entity_id: int, entry_id: String) -> int:
	return _send_command(
		CommandIds.NPC_ACTION_REQ,
		{
			"entity_id": entity_id,
			"entry_id": entry_id,
			"self_pos": _build_battle_self_pos_payload(),
		}
	)

# 向服务端提交“继续当前 NPC 剧情”的请求。
func request_npc_dialogue_next(entity_id: int, dialogue_id: int, node_id: String) -> int:
	return _send_command(
		CommandIds.NPC_DIALOGUE_NEXT_REQ,
		{
			"entity_id": entity_id,
			"dialogue_id": dialogue_id,
			"node_id": node_id,
		}
	)

# 向服务端提交“选择当前 NPC 剧情分支”的请求。
func request_npc_dialogue_choose(entity_id: int, dialogue_id: int, node_id: String, option_id: String) -> int:
	return _send_command(
		CommandIds.NPC_DIALOGUE_CHOOSE_REQ,
		{
			"entity_id": entity_id,
			"dialogue_id": dialogue_id,
			"node_id": node_id,
			"option_id": option_id,
		}
	)

# 向服务端提交商店购买请求；shop_id 当前等同于商店 NPC 的 entity_id。
func request_buy_item(shop_id: int, item_id: int, quantity: int = 1) -> int:
	return _send_command(
		CommandIds.BUY_ITEM_REQ,
		{
			"shop_id": shop_id,
			"goods_id": item_id,
			"item_id": item_id,
			"quantity": quantity,
		}
	)

# 向服务端发起一次单人 PVP 挑战请求。
func request_pvp_challenge(target_player_id: int) -> void:
	_send_command(
		CommandIds.PVP_CHALLENGE_REQ,
		{
			"op_id": _take_battle_op_id(),
			"target_player_id": target_player_id,
		}
	)

# 对收到的单人 PVP 邀请提交接受或拒绝结果。
func reply_pvp_challenge(challenge_id: int, accept: bool) -> void:
	_send_command(
		CommandIds.PVP_CHALLENGE_REPLY_REQ,
		{
			"challenge_id": challenge_id,
			"accept": accept,
		}
	)

# 向服务端提交一条战斗动作意图。
func submit_battle_action(
	battle_id: int,
	battle_round: int,
	actor_id: int,
	target_id: int,
	action_type: int = 1,
	skill_id: int = DEFAULT_BATTLE_SKILL_ID
) -> void:
	_send_command(
		CommandIds.BATTLE_ACTION_REQ,
		{
			"op_id": _take_battle_op_id(),
			"battle_id": battle_id,
			"round": battle_round,
			"action_type": action_type,
			"actor_id": actor_id,
			"skill_id": skill_id,
			"target_id": target_id,
		}
	)

# 向服务端切换当前战斗是否进入自动托管。
func set_battle_auto(battle_id: int, battle_round: int, enabled: bool) -> void:
	_send_command(
		CommandIds.BATTLE_ACTION_REQ,
		{
			"op_id": _take_battle_op_id(),
			"battle_id": battle_id,
			"round": battle_round,
			"action_type": 5,
			"actor_id": 0,
			"skill_id": 0,
			"target_id": 0,
			"auto_battle_enabled": enabled,
		}
	)

# 统一接收开发态消息并交给消息路由器分发。
func _on_dev_message_received(cmd: int, seq: int, _code: int, payload: Dictionary) -> void:
	if CommandIds.should_log_result(cmd):
		_emit_server_result_log(cmd, _format_ws_result_log(cmd, payload))
		_log_battle_payload_debug(cmd, payload)
		_log_bag_payload_debug(cmd, payload)
	# 先路由业务处理器写入 GameState，再结束 request_finished，避免 UI 读到旧快照。
	# 登录页发起的 ENTER_WORLD_RESP 只需要唤醒 request_finished，主场景尚未注册世界处理器时不应刷 warning。
	if MessageRouter.has_handler(cmd) or _request_cmd_for_response(cmd) == 0:
		MessageRouter.route_message(cmd, payload)
	_resolve_request_completion(cmd, seq, _code, payload)

# 底层 WebSocket 建连成功后自动发起业务鉴权。
func _on_websocket_opened() -> void:
	authenticate_ws()

# 底层 WebSocket 关闭后同步清空鉴权状态。
func _on_websocket_closed(_code: int, _reason: String) -> void:
	NetClient.set_authenticated(false)
	if _should_try_reconnect():
		_set_reconnect_in_progress(true)
		GameState.set_ws_authenticated(false, true)
		call_deferred("_reconnect_ws")
		return
	GameState.set_ws_authenticated(false)

# 处理 WebSocket 鉴权成功回执，并配置心跳和会话状态。
func _on_ws_auth_response(payload: Dictionary) -> void:
	_set_reconnect_in_progress(false)
	GameState.store_ws_session(payload)
	NetClient.set_authenticated(true)
	NetClient.configure_heartbeat(GameState.heartbeat_sec)
	session_authenticated.emit(payload)

# 处理断线重连成功回执，并把世界/战斗快照重新分发给现有控制器。
func _on_reconnect_response(payload: Dictionary) -> void:
	_set_reconnect_in_progress(false)
	GameState.store_ws_session(payload)
	NetClient.set_authenticated(true)
	NetClient.configure_heartbeat(GameState.heartbeat_sec)
	session_authenticated.emit(payload)

	var world_variant: Variant = payload.get("world", {})
	if world_variant is Dictionary:
		MessageRouter.route_message(CommandIds.WORLD_RESYNC_PUSH, world_variant)

	var battle_start_variant: Variant = payload.get("battle_start", {})
	if battle_start_variant is Dictionary and not battle_start_variant.is_empty():
		MessageRouter.route_message(CommandIds.BATTLE_START_PUSH, battle_start_variant)
		var replay_states_variant: Variant = payload.get("battle_replay_states", [])
		var replay_states: Array = replay_states_variant if replay_states_variant is Array else []
		if not replay_states.is_empty():
			for replay_state_variant in replay_states:
				if replay_state_variant is Dictionary:
					MessageRouter.route_message(CommandIds.BATTLE_STATE_PUSH, replay_state_variant)
		else:
			var battle_state_variant: Variant = payload.get("battle_state", {})
			if battle_state_variant is Dictionary and not battle_state_variant.is_empty():
				MessageRouter.route_message(CommandIds.BATTLE_STATE_PUSH, battle_state_variant)
	else:
		var battle_result_variant: Variant = payload.get("battle_result", {})
		if battle_result_variant is Dictionary and not battle_result_variant.is_empty():
			MessageRouter.route_message(CommandIds.BATTLE_RESULT_PUSH, battle_result_variant)
		else:
			GameState.clear_battle_state()

	# 重连恢复后的任务、宠物和背包摘要统一重新拉一次，避免战斗托管期间的
	# 成长与任务进度只停留在本地旧缓存。
	request_pet_list()
	request_bag_list()
	request_wallet()
	request_quest_list()

# 处理心跳回包，当前仅用于维持链路，无额外业务逻辑。
func _on_heartbeat_response(_payload: Dictionary) -> void:
	pass

# 处理服务端强制下线推送。
func _on_force_offline_push(payload: Dictionary) -> void:
	_set_reconnect_in_progress(false)
	GameState.set_ws_authenticated(false)
	# 提取服务端返回的下线原因文本。
	var reason := str(payload.get("reason", "account logged in elsewhere"))
	kicked.emit(reason)

# 处理服务端错误推送，并转为统一提示事件。
func _on_error_push(payload: Dictionary) -> void:
	var error_code: int = int(payload.get("code", 0))
	var message := str(payload.get("msg", "server returned an error push"))
	var display_message: String = ServerErrorMessages.format(error_code, message)
	_emit_server_result_log(
		CommandIds.ERROR_PUSH,
		"ERROR code=%d msg=%s" % [error_code, message]
	)
	if _reconnect_in_progress:
		_set_reconnect_in_progress(false)
		GameState.set_ws_authenticated(false)
		kicked.emit(display_message)
		return
	server_error_received.emit(error_code, display_message)
	notice_received.emit(display_message)

# 处理服务端普通公告推送，并兼容不同字段名。
func _on_notice_push(payload: Dictionary) -> void:
	notice_received.emit(str(payload.get("message", payload.get("msg", ""))))

# 处理服务端踢下线推送。
func _on_kickout_push(payload: Dictionary) -> void:
	_set_reconnect_in_progress(false)
	GameState.set_ws_authenticated(false)
	# 提取服务端返回的踢下线原因文本。
	var reason := str(payload.get("reason", payload.get("msg", "kicked by server")))
	kicked.emit(reason)

func is_reconnecting() -> bool:
	return _reconnect_in_progress


## 客户端正式运行不再输出协议日志；保留统一入口可避免改动网络请求与回包处理链路。
func _emit_server_result_log(_cmd: int, _message: String) -> void:
	pass


# 包装全部由 App 发起的 WebSocket 请求，统一补充发送日志，便于核对
# 客户端按钮是否真的触发了服务端权威链路。
func _send_command(cmd: int, payload: Dictionary) -> int:
	var seq := NetClient.send_command(cmd, payload)
	if seq > 0:
		_pending_request_cmds_by_seq[seq] = cmd
		var seqs_variant: Variant = _pending_request_seqs_by_cmd.get(cmd, [])
		var seqs: Array = seqs_variant if seqs_variant is Array else []
		seqs.append(seq)
		_pending_request_seqs_by_cmd[cmd] = seqs
	return seq


# 根据收到的回包消息，结束对应请求的等待状态。这样 UI 可以在数据真正准备好之后再显示。
func _resolve_request_completion(cmd: int, seq: int, code: int, payload: Dictionary) -> void:
	if cmd == CommandIds.ERROR_PUSH:
		var failed_request_cmd := int(_pending_request_cmds_by_seq.get(seq, 0))
		if failed_request_cmd != 0:
			_consume_pending_request(failed_request_cmd, seq)
			request_finished.emit(failed_request_cmd, seq, false, cmd, payload)
		return

	var request_cmd := _request_cmd_for_response(cmd)
	if cmd == CommandIds.NPC_DIALOGUE_RESP:
		if _pending_request_seqs_by_cmd.has(CommandIds.NPC_DIALOGUE_NEXT_REQ):
			request_cmd = CommandIds.NPC_DIALOGUE_NEXT_REQ
		elif _pending_request_seqs_by_cmd.has(CommandIds.NPC_DIALOGUE_CHOOSE_REQ):
			request_cmd = CommandIds.NPC_DIALOGUE_CHOOSE_REQ
	if request_cmd == 0:
		return

	var matched_seq := _pop_pending_seq_for_cmd(request_cmd)
	if matched_seq == 0:
		return
	_pending_request_cmds_by_seq.erase(matched_seq)
	var succeeded: bool = code == 0
	request_finished.emit(request_cmd, matched_seq, succeeded, cmd, payload)


# 把响应消息号映射回最初发出的请求消息号，供等待中的 UI 找到自己的回包。
func _request_cmd_for_response(response_cmd: int) -> int:
	match response_cmd:
		CommandIds.WS_AUTH_RESP:
			return CommandIds.WS_AUTH_REQ
		CommandIds.RECONNECT_RESP:
			return CommandIds.RECONNECT_REQ
		CommandIds.ENTER_WORLD_RESP:
			return CommandIds.ENTER_WORLD_REQ
		CommandIds.MOVE_INTENT_RESP:
			return CommandIds.MOVE_INTENT_REQ
		CommandIds.INTERACT_RESP:
			return CommandIds.INTERACT_REQ
		CommandIds.WILD_ENCOUNTER_RESP:
			return CommandIds.WILD_ENCOUNTER_REQ
		CommandIds.NPC_ACTION_RESP:
			return CommandIds.NPC_ACTION_REQ
		CommandIds.NPC_MENU_RESP:
			return CommandIds.NPC_MENU_REQ
		CommandIds.NPC_MENU_BATCH_RESP:
			return CommandIds.NPC_MENU_BATCH_REQ
		CommandIds.PET_LIST_RESP:
			return CommandIds.PET_LIST_REQ
		CommandIds.PET_LINEUP_SET_RESP:
			return CommandIds.PET_LINEUP_SET_REQ
		CommandIds.PLAYER_ALLOCATE_ATTR_RESP:
			return CommandIds.PLAYER_ALLOCATE_ATTR_REQ
		CommandIds.PLAYER_PROFILE_RESP:
			return CommandIds.PLAYER_PROFILE_REQ
		CommandIds.PET_ALLOCATE_ATTR_RESP:
			return CommandIds.PET_ALLOCATE_ATTR_REQ
		CommandIds.PLAYER_EQUIPMENT_LIST_RESP:
			return CommandIds.PLAYER_EQUIPMENT_LIST_REQ
		CommandIds.PLAYER_EQUIP_RESP:
			return CommandIds.PLAYER_EQUIP_REQ
		CommandIds.PLAYER_UNEQUIP_RESP:
			return CommandIds.PLAYER_UNEQUIP_REQ
		CommandIds.PLAYER_EQUIPMENT_ENHANCE_RESP:
			return CommandIds.PLAYER_EQUIPMENT_ENHANCE_REQ
		CommandIds.PLAYER_EQUIPMENT_REPAIR_RESP:
			return CommandIds.PLAYER_EQUIPMENT_REPAIR_REQ
		CommandIds.PET_ARTIFACT_EQUIP_RESP:
			return CommandIds.PET_ARTIFACT_EQUIP_REQ
		CommandIds.PET_ARTIFACT_UNEQUIP_RESP:
			return CommandIds.PET_ARTIFACT_UNEQUIP_REQ
		CommandIds.PET_SKILL_DETAIL_RESP:
			return CommandIds.PET_SKILL_DETAIL_REQ
		CommandIds.USE_ITEM_RESP:
			return CommandIds.USE_ITEM_REQ
		CommandIds.DROP_ITEM_RESP:
			return CommandIds.DROP_ITEM_REQ
		CommandIds.BATTLE_ACTION_RESP:
			return CommandIds.BATTLE_ACTION_REQ
		CommandIds.PVP_CHALLENGE_RESP:
			return CommandIds.PVP_CHALLENGE_REQ
		CommandIds.PVP_CHALLENGE_REPLY_RESP:
			return CommandIds.PVP_CHALLENGE_REPLY_REQ
		CommandIds.BAG_LIST_RESP:
			return CommandIds.BAG_LIST_REQ
		CommandIds.CONTAINER_LIST_RESP:
			return CommandIds.CONTAINER_LIST_REQ
		CommandIds.BAG_TO_WAREHOUSE_RESP:
			return CommandIds.BAG_TO_WAREHOUSE_REQ
		CommandIds.WAREHOUSE_TO_BAG_RESP:
			return CommandIds.WAREHOUSE_TO_BAG_REQ
		CommandIds.WALLET_QUERY_RESP:
			return CommandIds.WALLET_QUERY_REQ
		CommandIds.BUY_ITEM_RESP:
			return CommandIds.BUY_ITEM_REQ
		CommandIds.QUEST_LIST_RESP:
			return CommandIds.QUEST_LIST_REQ
		CommandIds.QUEST_ACCEPT_RESP:
			return CommandIds.QUEST_ACCEPT_REQ
		CommandIds.QUEST_SUBMIT_RESP:
			return CommandIds.QUEST_SUBMIT_REQ
		CommandIds.QUEST_TRACK_RESP:
			return CommandIds.QUEST_TRACK_REQ
		CommandIds.SCENE_TRIGGER_ACK_RESP:
			return CommandIds.SCENE_TRIGGER_ACK_REQ
		_:
			return 0


# 从命令对应的等待队列中弹出最早发出的请求序列号，保证连续点击时仍按顺序完成。
func _pop_pending_seq_for_cmd(request_cmd: int) -> int:
	var seqs_variant: Variant = _pending_request_seqs_by_cmd.get(request_cmd, [])
	if seqs_variant is not Array or seqs_variant.is_empty():
		return 0
	var seqs: Array = seqs_variant
	var matched_seq := int(seqs[0])
	seqs.remove_at(0)
	if seqs.is_empty():
		_pending_request_seqs_by_cmd.erase(request_cmd)
	else:
		_pending_request_seqs_by_cmd[request_cmd] = seqs
	return matched_seq


# 发生错误推送时，需要按原始序列号把等待队列中的条目一并移除，避免后续请求错配。
func _consume_pending_request(request_cmd: int, seq: int) -> void:
	_pending_request_cmds_by_seq.erase(seq)
	var seqs_variant: Variant = _pending_request_seqs_by_cmd.get(request_cmd, [])
	if seqs_variant is not Array:
		return
	var seqs: Array = seqs_variant
	var target_index := seqs.find(seq)
	if target_index == -1:
		return
	seqs.remove_at(target_index)
	if seqs.is_empty():
		_pending_request_seqs_by_cmd.erase(request_cmd)
	else:
		_pending_request_seqs_by_cmd[request_cmd] = seqs


# 组装 HTTP 请求结果日志，优先展示状态码、业务码和消息文本。
func _format_http_result_log(path: String, response: Dictionary) -> String:
	var code := int(response.get("code", 0))
	var http_status := int(response.get("http_status", 0))
	var message := str(response.get("msg", ""))
	var data_variant: Variant = response.get("data", {})
	var data: Dictionary = data_variant if data_variant is Dictionary else {}
	var summary := "HTTP %s -> http=%d code=%d" % [path, http_status, code]
	if not message.is_empty():
		summary += " msg=%s" % message
	var player_id := int(data.get("player_id", 0))
	if player_id > 0:
		summary += " player_id=%d" % player_id
	return summary


# 把战斗相关载荷序列化成 JSON，便于在调试构建下核对字段。
func _format_battle_payload_json(payload: Dictionary) -> String:
	return JSON.stringify(payload, "\t")


# 调试构建下把战斗载荷完整 JSON 输出到控制台；不进入 HUD，避免 RichTextLabel 内存膨胀。
func _log_battle_payload_debug(_cmd: int, _payload: Dictionary) -> void:
	pass


# 调试构建下把背包/装备/丢弃载荷完整 JSON 输出到控制台。
func _log_bag_payload_debug(_cmd: int, _payload: Dictionary) -> void:
	pass


# 组装 WebSocket 请求结果日志；只保留摘要字段，完整 JSON 见 _log_battle_payload_debug。
func _format_ws_result_log(cmd: int, payload: Dictionary) -> String:
	var summary := "%s" % CommandIds.name_of(cmd)

	if payload.has("accepted"):
		summary += " accepted=%s" % str(bool(payload.get("accepted", false)))
	if payload.has("reason"):
		summary += " reason=%s" % str(payload.get("reason", ""))
	if payload.has("msg"):
		summary += " msg=%s" % str(payload.get("msg", ""))
	if payload.has("battle_id"):
		summary += " battle_id=%s" % str(payload.get("battle_id", 0))
	if payload.has("scene_id"):
		summary += " scene_id=%s" % str(payload.get("scene_id", 0))
	if payload.has("challenge_id"):
		summary += " challenge_id=%s" % str(payload.get("challenge_id", 0))

	# 这些摘要字段能帮助快速判断世界、宠物和战斗结果是否正确返回，
	# 但又不会像整包 JSON 那样在移动端 HUD 里刷出过长文本。
	if payload.has("lineup"):
		var lineup_variant: Variant = payload.get("lineup", [])
		if lineup_variant is Array:
			summary += " lineup=%d" % lineup_variant.size()
	if payload.has("pets"):
		var pets_variant: Variant = payload.get("pets", [])
		if pets_variant is Array:
			summary += " pets=%d" % pets_variant.size()
	if payload.has("allies"):
		var allies_variant: Variant = payload.get("allies", [])
		if allies_variant is Array:
			summary += " allies=%d" % allies_variant.size()
	if payload.has("enemies"):
		var enemies_variant: Variant = payload.get("enemies", [])
		if enemies_variant is Array:
			summary += " enemies=%d" % enemies_variant.size()
	if payload.has("events"):
		var events_variant: Variant = payload.get("events", [])
		if events_variant is Array:
			summary += " events=%d" % events_variant.size()
	if payload.has("frame"):
		summary += " frame=%s" % str(payload.get("frame", 0))
	if payload.has("round"):
		summary += " round=%s" % str(payload.get("round", 0))
	if payload.has("phase"):
		summary += " phase=%s" % str(payload.get("phase", ""))
	if payload.has("code"):
		summary += " code=%s" % str(payload.get("code", 0))
	if payload.has("item_id"):
		summary += " item_id=%s" % str(payload.get("item_id", 0))
	if payload.has("item_name"):
		summary += " item_name=%s" % str(payload.get("item_name", ""))
	if payload.has("item_uid"):
		summary += " item_uid=%s" % str(payload.get("item_uid", ""))
	if payload.has("slot_index"):
		summary += " slot_index=%s" % str(payload.get("slot_index", 0))
	if payload.has("quantity"):
		summary += " quantity=%s" % str(payload.get("quantity", 0))
	if payload.has("dropped_quantity"):
		summary += " dropped_quantity=%s" % str(payload.get("dropped_quantity", 0))
	if payload.has("used_quantity"):
		summary += " used_quantity=%s" % str(payload.get("used_quantity", 0))
	if payload.has("container_type"):
		summary += " container_type=%s" % str(payload.get("container_type", ""))
	if payload.has("items"):
		var items_variant: Variant = payload.get("items", [])
		if items_variant is Array:
			summary += " items=%d" % items_variant.size()
	if payload.has("success"):
		summary += " success=%s" % str(bool(payload.get("success", false)))
	return summary


# 组装客户端发出的请求日志；只保留关键标识，完整 JSON 见 _log_battle_payload_debug。
func _format_ws_request_log(cmd: int, payload: Dictionary) -> String:
	var summary := "REQ %s" % CommandIds.name_of(cmd)
	if payload.has("entity_id"):
		summary += " entity_id=%s" % str(payload.get("entity_id", 0))
	if payload.has("target_player_id"):
		summary += " target_player_id=%s" % str(payload.get("target_player_id", 0))
	if payload.has("challenge_id"):
		summary += " challenge_id=%s" % str(payload.get("challenge_id", 0))
	if payload.has("quest_id"):
		summary += " quest_id=%s" % str(payload.get("quest_id", 0))
	if payload.has("battle_id"):
		summary += " battle_id=%s" % str(payload.get("battle_id", 0))
	if payload.has("container_type"):
		summary += " container_type=%s" % str(payload.get("container_type", ""))
	if payload.has("slot_index"):
		summary += " slot_index=%s" % str(payload.get("slot_index", 0))
	if payload.has("bag_slot_index"):
		summary += " bag_slot_index=%s" % str(payload.get("bag_slot_index", 0))
	if payload.has("item_uid"):
		summary += " item_uid=%s" % str(payload.get("item_uid", ""))
	if payload.has("item_id"):
		summary += " item_id=%s" % str(payload.get("item_id", 0))
	if payload.has("quantity"):
		summary += " quantity=%s" % str(payload.get("quantity", 0))
	if payload.has("equip_slot"):
		summary += " equip_slot=%s" % str(payload.get("equip_slot", ""))
	if payload.has("cost_item_id"):
		summary += " cost_item_id=%s" % str(payload.get("cost_item_id", 0))
	if payload.has("pet_uids"):
		var pet_uids_variant: Variant = payload.get("pet_uids", [])
		if pet_uids_variant is Array:
			summary += " pet_uids=%d" % pet_uids_variant.size()
	return summary

func _should_try_reconnect() -> bool:
	return GameState.player_id > 0 and not GameState.reconnect_token.is_empty()

func _reconnect_ws() -> void:
	if not _reconnect_in_progress:
		return
	var err := NetClient.connect_to_server()
	if err != OK:
		_set_reconnect_in_progress(false)
		GameState.set_ws_authenticated(false)
		notice_received.emit("断线重连失败，请重新登录。")

func _set_reconnect_in_progress(in_progress: bool) -> void:
	if _reconnect_in_progress == in_progress:
		return
	_reconnect_in_progress = in_progress
	reconnect_state_changed.emit(_reconnect_in_progress)

## 构造开战请求附带的客户端场景坐标，与服务端 return_pos 使用同一坐标系。
func _build_battle_self_pos_payload() -> Dictionary:
	var scene_x: int = int(round(float(GameState.player_snapshot.get("x", 0.0))))
	var scene_y: int = int(round(float(GameState.player_snapshot.get("y", 0.0))))
	return {
		"x": scene_x,
		"y": scene_y,
	}

# 返回下一个战斗类请求使用的操作标识，并在上限后回绕。
func _take_battle_op_id() -> int:
	# 暂存当前可返回的操作标识。
	var op_id := _next_battle_op_id
	_next_battle_op_id += 1
	if _next_battle_op_id > 0x7FFFFFFF:
		_next_battle_op_id = 1
	return op_id
