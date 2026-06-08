extends Node

# 世界场景资源的预加载引用。
const WORLD_SCENE := preload("res://scenes/world/world_scene.tscn")
# 战斗场景资源的预加载引用。
const BATTLE_SCENE := preload("res://scenes/battle/battle_scene.tscn")
# 返回登录页时使用的场景路径。
const LOGIN_SCENE_PATH := "res://scenes/auth/login_scene.tscn"
# 场景切换遮罩淡入淡出的持续时间。
const TRANSITION_DURATION := 0.18

# 上部游戏显示区域的根节点。
@onready var gameplay_area: Control = %GameplayArea
# 世界场景实例的挂载节点。
@onready var world_mount: Control = $GameplayArea/WorldMount
# 战斗场景实例的挂载节点。
@onready var battle_mount: Control = $GameplayArea/BattleMount
# 宠物相关消息处理控制器。
@onready var pet_controller: Node = %PetController
# 战斗相关消息处理控制器。
@onready var battle_controller: Node = %BattleController
# 背包相关消息处理控制器。
@onready var bag_controller: Node = %BagController
# 任务相关消息处理控制器。
@onready var quest_controller: Node = %QuestController
# 底部常驻 HUD 组件实例。
@onready var hud_root: RuntimeHud = %HudRoot
# 场景切换期间使用的全屏遮罩层。
@onready var transition_overlay: ColorRect = %TransitionOverlay

# 当前挂载的世界控制器实例引用。
var _world_controller: Node
# 当前挂载的战斗场景实例引用。
var _battle_scene: Control
# 标记当前是否正在跳回登录页，避免重复切场景。
var _redirecting_to_login: bool = false
# 标记当前是否已经请求过运行态基础摘要数据。
var _runtime_data_requested: bool = false

# 初始化主运行态，挂载世界场景并注册主链路消息与信号。
func _ready() -> void:
	App.bootstrap()
	if not GameState.is_ws_authenticated:
		call_deferred("_return_to_login_scene")
		return

	_play_fade_in()
	_mount_world_scene()
	_register_routes()
	_connect_signals()
	_sync_world_render_frame()
	_append_log("主场景已就绪。")
	_append_log("正在请求进入世界。")
	_sync_battle_visibility()
	App.enter_world()
	_refresh_view()

# 退出主场景时注销当前注册的业务路由。
func _exit_tree() -> void:
	_unregister_routes()

# 挂载世界场景实例，并绑定世界控制器对外广播的信号。
func _mount_world_scene() -> void:
	_world_controller = WORLD_SCENE.instantiate()
	world_mount.add_child(_world_controller)
	if _world_controller is Control:
		var world_control := _world_controller as Control
		world_control.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_sync_world_render_frame()
	if _world_controller.has_signal("scene_loaded"):
		_world_controller.connect("scene_loaded", Callable(self, "_on_world_scene_loaded"))
	if _world_controller.has_signal("player_position_changed"):
		_world_controller.connect("player_position_changed", Callable(self, "_on_player_position_changed"))
	if _world_controller.has_signal("scene_transition_requested"):
		_world_controller.connect("scene_transition_requested", Callable(self, "_on_scene_transition_requested"))
	if _world_controller.has_signal("scene_transition_failed"):
		_world_controller.connect("scene_transition_failed", Callable(self, "_on_scene_transition_failed"))
	if _world_controller.has_signal("npc_interaction_requested"):
		_world_controller.connect("npc_interaction_requested", Callable(self, "_on_npc_interaction_requested"))
	_append_log("世界场景已挂载。")

# 注册主运行态依赖的全部协议路由。
func _register_routes() -> void:
	if _world_controller == null:
		return

	MessageRouter.register_handler(CommandIds.ENTER_WORLD_RESP, Callable(_world_controller, "handle_enter_world"))
	MessageRouter.register_handler(CommandIds.ENTITY_ENTER_PUSH, Callable(_world_controller, "handle_entity_enter"))
	MessageRouter.register_handler(CommandIds.ENTITY_LEAVE_PUSH, Callable(_world_controller, "handle_entity_leave"))
	MessageRouter.register_handler(CommandIds.ENTITY_MOVE_PUSH, Callable(_world_controller, "handle_entity_move"))
	MessageRouter.register_handler(CommandIds.WORLD_RESYNC_PUSH, Callable(_world_controller, "handle_world_resync"))
	MessageRouter.register_handler(CommandIds.MOVE_INTENT_RESP, Callable(_world_controller, "handle_move_intent_response"))
	MessageRouter.register_handler(CommandIds.INTERACT_RESP, Callable(battle_controller, "handle_interact_response"))
	MessageRouter.register_handler(CommandIds.NPC_ACTION_RESP, Callable(battle_controller, "handle_npc_action_response"))

	MessageRouter.register_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.register_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.register_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))

	MessageRouter.register_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.register_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.register_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.register_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))

	MessageRouter.register_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.register_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))
	MessageRouter.register_handler(CommandIds.QUEST_LIST_RESP, Callable(quest_controller, "handle_quest_list"))
	MessageRouter.register_handler(CommandIds.QUEST_UPDATE_PUSH, Callable(quest_controller, "handle_quest_update"))
	MessageRouter.register_handler(CommandIds.QUEST_REMOVE_PUSH, Callable(quest_controller, "handle_quest_remove"))
	MessageRouter.register_handler(CommandIds.QUEST_ACCEPT_RESP, Callable(quest_controller, "handle_quest_accept_response"))
	MessageRouter.register_handler(CommandIds.QUEST_SUBMIT_RESP, Callable(quest_controller, "handle_quest_submit_response"))
	MessageRouter.register_handler(CommandIds.QUEST_TRACK_RESP, Callable(quest_controller, "handle_quest_track_response"))

# 注销主运行态注册过的全部协议路由。
func _unregister_routes() -> void:
	if _world_controller == null:
		return

	MessageRouter.unregister_handler(CommandIds.ENTER_WORLD_RESP, Callable(_world_controller, "handle_enter_world"))
	MessageRouter.unregister_handler(CommandIds.ENTITY_ENTER_PUSH, Callable(_world_controller, "handle_entity_enter"))
	MessageRouter.unregister_handler(CommandIds.ENTITY_LEAVE_PUSH, Callable(_world_controller, "handle_entity_leave"))
	MessageRouter.unregister_handler(CommandIds.ENTITY_MOVE_PUSH, Callable(_world_controller, "handle_entity_move"))
	MessageRouter.unregister_handler(CommandIds.WORLD_RESYNC_PUSH, Callable(_world_controller, "handle_world_resync"))
	MessageRouter.unregister_handler(CommandIds.MOVE_INTENT_RESP, Callable(_world_controller, "handle_move_intent_response"))
	MessageRouter.unregister_handler(CommandIds.INTERACT_RESP, Callable(battle_controller, "handle_interact_response"))
	MessageRouter.unregister_handler(CommandIds.NPC_ACTION_RESP, Callable(battle_controller, "handle_npc_action_response"))
	MessageRouter.unregister_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.unregister_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.unregister_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))
	MessageRouter.unregister_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.unregister_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))
	MessageRouter.unregister_handler(CommandIds.QUEST_LIST_RESP, Callable(quest_controller, "handle_quest_list"))
	MessageRouter.unregister_handler(CommandIds.QUEST_UPDATE_PUSH, Callable(quest_controller, "handle_quest_update"))
	MessageRouter.unregister_handler(CommandIds.QUEST_REMOVE_PUSH, Callable(quest_controller, "handle_quest_remove"))
	MessageRouter.unregister_handler(CommandIds.QUEST_ACCEPT_RESP, Callable(quest_controller, "handle_quest_accept_response"))
	MessageRouter.unregister_handler(CommandIds.QUEST_SUBMIT_RESP, Callable(quest_controller, "handle_quest_submit_response"))
	MessageRouter.unregister_handler(CommandIds.QUEST_TRACK_RESP, Callable(quest_controller, "handle_quest_track_response"))

# 绑定主运行态依赖的应用信号、HUD 交互信号和全局状态信号。
func _connect_signals() -> void:
	App.notice_received.connect(_on_notice_received)
	App.kicked.connect(_on_kicked)
	gameplay_area.resized.connect(_sync_world_render_frame)
	hud_root.challenge_requested.connect(_on_challenge_requested)
	hud_root.pet_requested.connect(_on_pet_requested)
	hud_root.lineup_requested.connect(_on_lineup_requested)
	hud_root.bag_requested.connect(_on_bag_requested)
	hud_root.quest_requested.connect(_on_quest_requested)
	hud_root.quest_track_requested.connect(_on_quest_track_requested)
	hud_root.quest_accept_requested.connect(_on_quest_accept_requested)
	hud_root.quest_submit_requested.connect(_on_quest_submit_requested)
	hud_root.lineup_submit_requested.connect(_on_lineup_submit_requested)
	hud_root.npc_menu_refresh_requested.connect(_on_npc_menu_refresh_requested)
	hud_root.npc_menu_entry_selected.connect(_on_npc_menu_entry_selected)

	if battle_controller.has_signal("interact_responded"):
		battle_controller.connect("interact_responded", Callable(self, "_on_interact_responded"))
	if battle_controller.has_signal("action_responded"):
		battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
	if battle_controller.has_signal("interact_payload_received"):
		battle_controller.connect("interact_payload_received", Callable(self, "_on_interact_payload_received"))
	if battle_controller.has_signal("npc_action_payload_received"):
		battle_controller.connect("npc_action_payload_received", Callable(self, "_on_npc_action_payload_received"))
	if battle_controller.has_signal("battle_started"):
		battle_controller.connect("battle_started", Callable(self, "_on_battle_started"))
	if battle_controller.has_signal("battle_finished"):
		battle_controller.connect("battle_finished", Callable(self, "_on_battle_finished"))

	GameState.session_changed.connect(_refresh_view)
	GameState.world_snapshot_changed.connect(_refresh_view)
	GameState.battle_changed.connect(_sync_battle_visibility)
	GameState.battle_changed.connect(_refresh_view)
	GameState.quests_changed.connect(_refresh_view)
	NetClient.connection_state_changed.connect(_on_connection_state_changed)
	NetClient.websocket_closed.connect(_on_websocket_closed)

# 处理服务端下发的普通提示信息。
func _on_notice_received(message: String) -> void:
	_append_log("提示: %s" % message)

# 处理被服务端踢下线的事件，并返回登录页。
func _on_kicked(reason: String) -> void:
	_append_log("连接已被服务端断开: %s" % reason)
	_return_to_login_scene()

# 处理网络连接状态变化，并在断开时跳回登录页。
func _on_connection_state_changed(state: String) -> void:
	_refresh_view()
	_append_log("WebSocket 状态 -> %s" % state)
	if state == "closed" and not _redirecting_to_login:
		_return_to_login_scene()

# 处理世界场景装载完成事件，并在首次进入世界后拉取基础摘要数据。
func _on_world_scene_loaded(scene_id: String) -> void:
	_append_log("已进入场景: %s" % scene_id)
	if not _runtime_data_requested:
		_runtime_data_requested = true
		App.request_pet_list()
		App.request_bag_list()
		App.request_quest_list()
		_append_log("正在同步宠物、背包与任务摘要。")
	_refresh_view()

# 处理本地玩家坐标变化，并把最新位置写回 HUD 头部文案。
func _on_player_position_changed(local_position: Vector2, global_position: Vector2) -> void:
	var player_name := str(GameState.player_snapshot.get("name", "未登录"))
	var player_text := "玩家 %s | 局部(%.0f,%.0f) 全局(%.0f,%.0f)" % [
		player_name,
		local_position.x,
		local_position.y,
		global_position.x,
		global_position.y,
	]
	hud_root.set_header_texts(str(hud_root.status_label.text), str(hud_root.scene_label.text), player_text)

# 记录切图请求发起时的来源与目标场景。
func _on_scene_transition_requested(from_scene_id: int, to_scene_id: int) -> void:
	_append_log("请求切换地图: %d -> %d" % [from_scene_id, to_scene_id])

# 记录切图失败原因。
func _on_scene_transition_failed(reason: String) -> void:
	_append_log("地图切换失败: %s" % reason)

func _on_npc_interaction_requested(entity_id: int, npc_name: String) -> void:
	_append_log("尝试与 NPC 交互: %s (%d)" % [npc_name, entity_id])
	App.request_interact(entity_id)

func _on_interact_payload_received(payload: Dictionary) -> void:
	if str(payload.get("response_type", "")) != "menu":
		return
	hud_root.show_npc_menu(payload)
	_append_log("打开 NPC 菜单: %s" % str(payload.get("npc_name", "未知 NPC")))

func _on_npc_menu_refresh_requested(entity_id: int) -> void:
	App.request_interact(entity_id)

func _on_npc_menu_entry_selected(entity_id: int, entry_id: String, entry_type: String, quest_id: int, quest_state: String) -> void:
	if entity_id <= 0 or entry_id.is_empty():
		return
	if entry_type == "quest" and quest_id > 0:
		if quest_state == "AVAILABLE":
			App.accept_quest(quest_id, entity_id)
		elif quest_state == "READY_TO_SUBMIT":
			App.submit_quest(quest_id, entity_id)
		App.request_interact(entity_id)
		return
	App.request_npc_action(entity_id, entry_id)

func _on_npc_action_payload_received(payload: Dictionary) -> void:
	var notice := str(payload.get("notice", payload.get("reason", "")))
	if not notice.is_empty():
		_append_log("NPC 操作: %s" % notice)
	App.request_quest_list()
	if payload.has("menu_entries"):
		var menu_entries_value: Variant = payload.get("menu_entries", [])
		if menu_entries_value is Array and not (menu_entries_value as Array).is_empty():
			hud_root.show_npc_menu({
				"entity_id": int(payload.get("entity_id", 0)),
				"npc_name": str(payload.get("npc_name", "可交互 NPC")),
				"menu_entries": menu_entries_value,
			})
			return
	hud_root.hide_npc_menu()

func _on_quest_requested() -> void:
	App.request_quest_list()

func _on_quest_track_requested(quest_id: int) -> void:
	App.track_quest(quest_id)

func _on_quest_accept_requested(quest_id: int, npc_id: int) -> void:
	App.accept_quest(quest_id, npc_id)
	if npc_id > 0:
		App.request_interact(npc_id)

func _on_quest_submit_requested(quest_id: int, npc_id: int) -> void:
	App.submit_quest(quest_id, npc_id)
	if npc_id > 0:
		App.request_interact(npc_id)

# 处理世界交互回执，并刷新主视图显示。
func _on_interact_responded(accepted: bool, reason: String) -> void:
	_append_log("交互结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])
	_refresh_view()

# 处理战斗动作回执，并记录服务端接受或拒绝结果。
func _on_action_responded(accepted: bool, reason: String) -> void:
	_append_log("战斗动作结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])

# 处理战斗开始事件，挂载战斗场景并同步显示状态。
func _on_battle_started(payload: Dictionary) -> void:
	_append_log("进入战斗场景。")
	_mount_battle_scene()
	_sync_battle_visibility()
	if payload.has("battle_id"):
		_append_log("战斗ID: %s" % str(payload.get("battle_id", "")))

# 处理战斗结束事件，卸载战斗场景并回到世界显示。
func _on_battle_finished(_payload: Dictionary) -> void:
	_append_log("战斗结束，返回世界场景。")
	_sync_battle_visibility()
	_unmount_battle_scene()
	App.request_quest_list()
	_refresh_view()

# 处理底层 WebSocket 关闭事件，并在需要时跳回登录页。
func _on_websocket_closed(code: int, reason: String) -> void:
	if code == -1 and reason.is_empty():
		return
	_append_log("WebSocket 已关闭: %d %s" % [code, reason])
	if not _redirecting_to_login:
		_return_to_login_scene()

# 按当前全局状态刷新主运行态顶部 HUD 文案。
func _refresh_view() -> void:
	# 保存顶部状态行展示的连接文案。
	var status_text: String = ""
	# 保存顶部场景行展示的场景文案。
	var scene_text: String = ""
	# 保存顶部玩家行展示的角色文案。
	var player_text: String = ""
	if GameState.is_in_battle:
		status_text = "WS %s | 战斗中" % NetClient.get_connection_state()
		scene_text = "战斗场景"
		player_text = "玩家 %s" % str(GameState.player_snapshot.get("name", "未登录"))
		hud_root.set_header_texts(status_text, scene_text, player_text)
		return

	status_text = "WS %s | HTTP %s" % [
		NetClient.get_connection_state(),
		"ok" if not GameState.access_jwt.is_empty() else "-",
	]

	# 读取当前世界快照中的场景标识。
	var scene_id := str(GameState.scene_snapshot.get("scene_id", "未进入"))
	scene_text = "场景 %s | 实体 %d" % [scene_id, GameState.nearby_entities.size()]

	# 读取当前玩家名称，用于拼接角色摘要文案。
	var player_name := str(GameState.player_snapshot.get("name", "未登录"))
	player_text = "玩家 %s" % player_name
	hud_root.set_header_texts(status_text, scene_text, player_text)

# 向底部 HUD 日志区域追加一条文本。
func _append_log(message: String) -> void:
	hud_root.append_log(message)

# 挂载战斗场景实例。
func _mount_battle_scene() -> void:
	if _battle_scene != null:
		return
	_battle_scene = BATTLE_SCENE.instantiate() as Control
	if _battle_scene == null:
		return
	battle_mount.add_child(_battle_scene)

# 卸载当前战斗场景实例。
func _unmount_battle_scene() -> void:
	if _battle_scene == null:
		return
	_battle_scene.queue_free()
	_battle_scene = null

# 根据全局战斗状态切换世界层和战斗层的可见性。
func _sync_battle_visibility() -> void:
	# 标记当前是否处于战斗态。
	var active := GameState.is_in_battle
	if active:
		_mount_battle_scene()
	world_mount.visible = not active
	battle_mount.visible = active

# 处理 HUD 发起的挑战请求，并默认选择第一个附近实体作为目标。
func _on_challenge_requested() -> void:
	if GameState.is_in_battle:
		return
	# 读取当前附近实体的全部标识列表。
	var entity_ids := GameState.nearby_entities.keys()
	if entity_ids.is_empty():
		_append_log("附近没有可挑战的NPC。")
		_refresh_view()
		return
	entity_ids.sort()
	# 选取排序后的首个实体作为默认交互目标。
	var entity_id := int(entity_ids[0])
	_append_log("向服务端发起挑战，目标实体: %d" % entity_id)
	App.request_interact(entity_id)

# 处理 HUD 发起的宠物列表刷新请求。
func _on_pet_requested() -> void:
	_append_log("请求宠物列表。")
	App.request_pet_list()

# 处理 HUD 发起的编队摘要刷新请求。
func _on_lineup_requested() -> void:
	_append_log("请求编队摘要。")
	App.request_pet_list()

# 处理 HUD 发起的背包列表刷新请求。
func _on_bag_requested() -> void:
	_append_log("请求背包列表。")
	App.request_bag_list()

# 处理 HUD 提交的编队保存请求。
func _on_lineup_submit_requested(pet_uids: Array[int]) -> void:
	if pet_uids.is_empty():
		_append_log("编队不能为空。")
		return
	_append_log("提交编队: %s" % str(pet_uids))
	App.set_pet_lineup(pet_uids)

# 返回登录页，并在切换前清理当前实时连接和全局状态。
func _return_to_login_scene() -> void:
	if _redirecting_to_login:
		return
	_redirecting_to_login = true
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(1.0)
	NetClient.disconnect_from_server()
	GameState.reset_session_state()
	get_tree().change_scene_to_file(LOGIN_SCENE_PATH)

# 播放主场景进入时的遮罩淡入动画。
func _play_fade_in() -> void:
	transition_overlay.color.a = 1.0
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(0.0)
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

# 把遮罩透明度过渡到指定目标值。
func _fade_overlay(target_alpha: float) -> void:
	# 创建当前使用的过渡补间动画。
	var tween := create_tween()
	tween.tween_property(transition_overlay, "color:a", target_alpha, TRANSITION_DURATION)
	await tween.finished

# 对令牌做简化展示，避免完整内容直接显示在界面上。
func _short_token(token: String) -> String:
	if token.is_empty():
		return "none"
	if token.length() <= 12:
		return token
	return "%s...%s" % [token.substr(0, 6), token.substr(token.length() - 4, 4)]

# 把上部游戏区域尺寸同步到子视口和世界控制器。
func _sync_world_render_frame() -> void:
	if _world_controller == null:
		return
	if _world_controller.has_method("set_render_frame_size"):
		_world_controller.call("set_render_frame_size", gameplay_area.size)
