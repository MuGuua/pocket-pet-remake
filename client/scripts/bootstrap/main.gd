extends Node

const UiFormat = preload("res://scripts/common/ui_format.gd")

# 世界场景资源的预加载引用。
const WORLD_SCENE := preload("res://scenes/world/world_scene.tscn")
const BATTLE_SCENE := preload("res://scenes/battle/battle_scene.tscn")
const MAIN_MENU_SCENE := preload("res://scenes/ui/main_menu.tscn")
const PLAYER_PANEL_SCENE := preload("res://scenes/ui/player_panel.tscn")
const NPC_MENU_SCENE := preload("res://scenes/ui/npc_menu.tscn")
const NPC_LIST_MENU_SCENE := preload("res://scenes/ui/npc_list_menu.tscn")
const LIMENG_DIALOGUE_PATH := "res://dialogue/untitled.dialogue"
const GENERIC_NPC_DIALOGUE_PATH := "res://dialogue/测试1.dialogue"
# 返回登录页时使用的场景路径。
const LOGIN_SCENE_PATH := "res://scenes/auth/login_scene.tscn"
# 场景切换遮罩淡入淡出的持续时间。
const TRANSITION_DURATION := 0.18
# 进入战斗时黑色遮罩的渐入渐出时长，略长于普通场景切换以掩盖加载过程。
const BATTLE_TRANSITION_DURATION := 0.28
# 当前客户端默认只允许从附近玩家列表中发起 PVP 挑战，避免没有明确目标时误发请求。
const PLAYER_ENTITY_TYPE: int = 1

# 上部游戏显示区域的根节点。
@onready var gameplay_area: Control = %GameplayArea
# 世界场景实例的挂载节点。
@onready var world_mount: Control = $GameplayArea/WorldMount
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
# 战斗表现场景实例；非战斗态时为 null。
var _battle_scene: Control = null
# 标记当前是否正在跳回登录页，避免重复切场景。
var _redirecting_to_login: bool = false
var _main_menu: CanvasLayer
var _player_panel: CanvasLayer
var _npc_menu: CanvasLayer
var _npc_list_menu: CanvasLayer
var _pvp_target_menu: CanvasLayer
var _pvp_invite_dialog: ConfirmationDialog
var _opening_npc_menu_from_list: bool = false
var _runtime_data_requested: bool = false
# 缓存最近一次收到但尚未处理的 PVP 邀请载荷，供确认框按钮回调读取。
var _pending_pvp_invite: Dictionary = {}

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
	_create_runtime_ui()
	_sync_world_render_frame()
	_append_log("主场景已就绪。")
	_append_log("正在请求进入世界。")
	App.enter_world()
	_refresh_view()

# 退出主场景时注销当前注册的业务路由。
func _exit_tree() -> void:
	_unregister_routes()
	if App.notice_received.is_connected(_on_notice_received):
		App.notice_received.disconnect(_on_notice_received)
	if App.kicked.is_connected(_on_kicked):
		App.kicked.disconnect(_on_kicked)
	if App.server_result_logged.is_connected(_on_server_result_logged):
		App.server_result_logged.disconnect(_on_server_result_logged)
	if gameplay_area != null and gameplay_area.resized.is_connected(_sync_world_render_frame):
		gameplay_area.resized.disconnect(_sync_world_render_frame)
	if GameState.session_changed.is_connected(_refresh_view):
		GameState.session_changed.disconnect(_refresh_view)
	if GameState.world_snapshot_changed.is_connected(_refresh_view):
		GameState.world_snapshot_changed.disconnect(_refresh_view)
	if GameState.battle_changed.is_connected(_refresh_view):
		GameState.battle_changed.disconnect(_refresh_view)
	if GameState.quests_changed.is_connected(_refresh_view):
		GameState.quests_changed.disconnect(_refresh_view)
	if NetClient.connection_state_changed.is_connected(_on_connection_state_changed):
		NetClient.connection_state_changed.disconnect(_on_connection_state_changed)
	if NetClient.websocket_closed.is_connected(_on_websocket_closed):
		NetClient.websocket_closed.disconnect(_on_websocket_closed)

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
	if _world_controller.has_signal("wild_encounter_responded"):
		_world_controller.connect("wild_encounter_responded", Callable(self, "_on_wild_encounter_responded"))
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
	MessageRouter.register_handler(CommandIds.WILD_ENCOUNTER_RESP, Callable(_world_controller, "handle_wild_encounter_response"))
	MessageRouter.register_handler(CommandIds.MOVE_INTENT_RESP, Callable(_world_controller, "handle_move_intent_response"))
	MessageRouter.register_handler(CommandIds.INTERACT_RESP, Callable(battle_controller, "handle_interact_response"))
	MessageRouter.register_handler(CommandIds.NPC_ACTION_RESP, Callable(battle_controller, "handle_npc_action_response"))
	MessageRouter.register_handler(CommandIds.PVP_CHALLENGE_RESP, Callable(battle_controller, "handle_pvp_challenge_response"))
	MessageRouter.register_handler(CommandIds.PVP_CHALLENGE_PUSH, Callable(battle_controller, "handle_pvp_challenge_push"))
	MessageRouter.register_handler(CommandIds.PVP_CHALLENGE_REPLY_RESP, Callable(battle_controller, "handle_pvp_challenge_reply_response"))

	MessageRouter.register_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.register_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.register_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))

	MessageRouter.register_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.register_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.register_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.register_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))

	MessageRouter.register_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.register_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))
	MessageRouter.register_handler(CommandIds.USE_ITEM_RESP, Callable(bag_controller, "handle_use_item_response"))
	MessageRouter.register_handler(CommandIds.CONTAINER_LIST_RESP, Callable(bag_controller, "handle_container_list"))
	MessageRouter.register_handler(CommandIds.WALLET_QUERY_RESP, Callable(bag_controller, "handle_wallet_query"))
	MessageRouter.register_handler(CommandIds.WALLET_UPDATE_PUSH, Callable(bag_controller, "handle_wallet_update"))
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
	MessageRouter.unregister_handler(CommandIds.WILD_ENCOUNTER_RESP, Callable(_world_controller, "handle_wild_encounter_response"))
	MessageRouter.unregister_handler(CommandIds.MOVE_INTENT_RESP, Callable(_world_controller, "handle_move_intent_response"))
	MessageRouter.unregister_handler(CommandIds.INTERACT_RESP, Callable(battle_controller, "handle_interact_response"))
	MessageRouter.unregister_handler(CommandIds.NPC_ACTION_RESP, Callable(battle_controller, "handle_npc_action_response"))
	MessageRouter.unregister_handler(CommandIds.PVP_CHALLENGE_RESP, Callable(battle_controller, "handle_pvp_challenge_response"))
	MessageRouter.unregister_handler(CommandIds.PVP_CHALLENGE_PUSH, Callable(battle_controller, "handle_pvp_challenge_push"))
	MessageRouter.unregister_handler(CommandIds.PVP_CHALLENGE_REPLY_RESP, Callable(battle_controller, "handle_pvp_challenge_reply_response"))
	MessageRouter.unregister_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.unregister_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.unregister_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))
	MessageRouter.unregister_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.unregister_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))
	MessageRouter.unregister_handler(CommandIds.USE_ITEM_RESP, Callable(bag_controller, "handle_use_item_response"))
	MessageRouter.unregister_handler(CommandIds.CONTAINER_LIST_RESP, Callable(bag_controller, "handle_container_list"))
	MessageRouter.unregister_handler(CommandIds.WALLET_QUERY_RESP, Callable(bag_controller, "handle_wallet_query"))
	MessageRouter.unregister_handler(CommandIds.WALLET_UPDATE_PUSH, Callable(bag_controller, "handle_wallet_update"))
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
	App.server_result_logged.connect(_on_server_result_logged)
	gameplay_area.resized.connect(_sync_world_render_frame)

	if battle_controller.has_signal("interact_responded"):
		battle_controller.connect("interact_responded", Callable(self, "_on_interact_responded"))
	if battle_controller.has_signal("action_responded"):
		battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
	if battle_controller.has_signal("interact_payload_received"):
		battle_controller.connect("interact_payload_received", Callable(self, "_on_interact_payload_received"))
	if battle_controller.has_signal("npc_action_payload_received"):
		battle_controller.connect("npc_action_payload_received", Callable(self, "_on_npc_action_payload_received"))
	if battle_controller.has_signal("pvp_challenge_responded"):
		battle_controller.connect("pvp_challenge_responded", Callable(self, "_on_pvp_challenge_responded"))
	if battle_controller.has_signal("pvp_challenge_received"):
		battle_controller.connect("pvp_challenge_received", Callable(self, "_on_pvp_challenge_received"))
	if battle_controller.has_signal("pvp_challenge_reply_responded"):
		battle_controller.connect("pvp_challenge_reply_responded", Callable(self, "_on_pvp_challenge_reply_responded"))
	if battle_controller.has_signal("battle_started"):
		battle_controller.connect("battle_started", Callable(self, "_on_battle_started"))
	if battle_controller.has_signal("battle_finished"):
		battle_controller.connect("battle_finished", Callable(self, "_on_battle_finished"))

	GameState.session_changed.connect(_refresh_view)
	GameState.world_snapshot_changed.connect(_refresh_view)
	GameState.battle_changed.connect(_refresh_view)
	GameState.quests_changed.connect(_refresh_view)
	NetClient.connection_state_changed.connect(_on_connection_state_changed)
	NetClient.websocket_closed.connect(_on_websocket_closed)

func _unhandled_input(event: InputEvent) -> void:
	if event.is_action_pressed("open_main_menu"):
		if _has_blocking_ui_open("main_menu"):
			return
		if _main_menu == null:
			return
		if _main_menu.visible:
			_main_menu.call("close_menu")
		else:
			_main_menu.call("open_menu")
			_set_runtime_menu_locked(true)
		get_viewport().set_input_as_handled()
		return

	if event.is_action_pressed("open_player_panel"):
		if _has_blocking_ui_open("player_panel"):
			return
		if _player_panel == null:
			return
		if _player_panel.visible:
			_player_panel.call("close_menu")
		else:
			_player_panel.call("open_menu")
			_set_runtime_menu_locked(true)
		get_viewport().set_input_as_handled()
		return

	if not event.is_action_pressed("open_scene_npc_list"):
		return
	if GameState.is_in_battle or _npc_list_menu == null or _has_blocking_ui_open("npc_list"):
		return
	if _npc_list_menu.visible:
		_npc_list_menu.call("close_menu")
		get_viewport().set_input_as_handled()
		return

	var nearby_npcs: Array[Dictionary] = _collect_nearby_npc_entries()
	if nearby_npcs.is_empty():
		return
	_npc_list_menu.call("configure", "周围 NPC", nearby_npcs)
	_npc_list_menu.call("open_menu")
	_set_runtime_menu_locked(true)
	get_viewport().set_input_as_handled()

# 处理服务端下发的普通提示信息。
func _on_notice_received(message: String) -> void:
	_append_log("提示: %s" % message)

# 把统一格式的服务端请求结果写入运行态 HUD，便于边操作边核对服务端回包。
func _on_server_result_logged(message: String) -> void:
	_append_log("服务端结果: %s" % message)

# 处理被服务端踢下线的事件，并返回登录页。
func _on_kicked(reason: String) -> void:
	_append_log("连接已被服务端断开: %s" % reason)
	_return_to_login_scene()

# 处理网络连接状态变化，并在断开时跳回登录页。
func _on_connection_state_changed(state: String) -> void:
	_refresh_view()
	_append_log("WebSocket 状态 -> %s" % state)
	if state == "closed" and not _redirecting_to_login and not App.is_reconnecting():
		_return_to_login_scene()

# 处理世界场景装载完成事件，并在首次进入世界后拉取基础摘要数据。
func _on_world_scene_loaded(scene_id: String) -> void:
	_append_log("已进入场景: %s" % scene_id)
	if not _runtime_data_requested:
		_runtime_data_requested = true
		App.request_pet_list()
		App.request_bag_list()
		App.request_quest_list()
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
	player_text = UiFormat.normalize_text(player_text)
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

func _on_wild_encounter_responded(accepted: bool, reason: String) -> void:
	_append_log("暗雷遭遇: %s (%s)" % ["accepted" if accepted else "rejected", reason])
	_refresh_view()

func _on_interact_payload_received(payload: Dictionary) -> void:
	if str(payload.get("response_type", "")) != "menu":
		return
	_append_log("收到 NPC 菜单数据: %s" % str(payload.get("npc_name", "未知 NPC")))
	if _npc_menu != null:
		var menu_options: Array[Dictionary] = _build_npc_menu_options(payload)
		_npc_menu.call("configure", str(payload.get("npc_name", "NPC")), menu_options)
		_npc_menu.call("open_menu")
		_set_runtime_menu_locked(true)

func _on_npc_action_payload_received(payload: Dictionary) -> void:
	if str(payload.get("result_type", "")) == "battle":
		_close_runtime_menus()
		return
	var dialogue_binding := _resolve_npc_dialogue_binding({
		"entity_id": int(payload.get("entity_id", 0)),
		"id": str(payload.get("entry_id", "")),
		"entry_type": "dialog",
		"npc_name": str(payload.get("npc_name", "NPC")),
	})
	if not dialogue_binding.is_empty():
		_show_npc_dialogue(dialogue_binding, str(payload.get("npc_name", "NPC")))
		return
	var notice := str(payload.get("notice", payload.get("reason", "")))
	if not notice.is_empty():
		_append_log("NPC 操作: %s" % notice)
	if payload.has("menu_entries") and _npc_menu != null:
		var menu_options: Array[Dictionary] = _build_npc_menu_options(payload)
		_npc_menu.call("configure", str(payload.get("npc_name", "NPC")), menu_options)
		_npc_menu.call("open_menu")
		_set_runtime_menu_locked(true)

# 处理世界交互回执，并刷新主视图显示。
func _on_interact_responded(accepted: bool, reason: String) -> void:
	_append_log("交互结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])
	_refresh_view()

# 处理战斗动作回执，并记录服务端接受或拒绝结果。
func _on_action_responded(accepted: bool, reason: String) -> void:
	_append_log("战斗动作结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])

# 处理战斗开始事件，挂载战斗表现场景。
func _on_battle_started(payload: Dictionary) -> void:
	_append_log("收到战斗开始事件。")
	_close_runtime_menus()
	if payload.has("battle_id"):
		_append_log("战斗ID: %s" % str(payload.get("battle_id", "")))
	await _enter_battle_with_transition(payload)
	_refresh_view()

## 黑色渐入 → 加载战斗场景与数据 → 渐出，掩盖世界到战斗的切换过程。
func _enter_battle_with_transition(payload: Dictionary) -> void:
	if transition_overlay == null:
		await _mount_battle_scene(payload)
		return
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(1.0, BATTLE_TRANSITION_DURATION)
	await _mount_battle_scene(payload)
	await get_tree().process_frame
	await _fade_overlay(0.0, BATTLE_TRANSITION_DURATION)
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

# 处理战斗结束事件，卸载战斗场景并回到世界 HUD。
func _on_battle_finished(payload: Dictionary) -> void:
	_unmount_battle_scene()
	var reward_gold := int(payload.get("reward_gold", 0))
	var reward_player_exp := int(payload.get("reward_player_exp", 0))
	var drop_texts_variant: Variant = payload.get("drop_texts", [])
	var drop_texts: Array = drop_texts_variant if drop_texts_variant is Array else []
	if reward_gold > 0 or reward_player_exp > 0:
		_append_log("战斗结束，获得 %d 金币 / %d 角色经验。" % [reward_gold, reward_player_exp])
	else:
		_append_log("战斗结束，返回世界场景。")
	for drop_text_variant in drop_texts:
		var drop_text := str(drop_text_variant)
		if not drop_text.is_empty():
			_append_log(drop_text)
	App.request_quest_list()
	_refresh_view()

func _mount_battle_scene(payload: Dictionary) -> void:
	var map_snapshot: Texture2D = await _capture_world_map_snapshot()
	if _battle_scene != null:
		if map_snapshot != null and _battle_scene.has_method("apply_background_texture"):
			_battle_scene.call("apply_background_texture", map_snapshot)
		if _battle_scene.has_method("_on_battle_started"):
			_battle_scene.call("_on_battle_started", payload)
		return
	_battle_scene = BATTLE_SCENE.instantiate() as Control
	if _battle_scene == null:
		return
	gameplay_area.add_child(_battle_scene)
	_battle_scene.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_battle_scene.z_index = 10
	if _battle_scene.has_method("bind_battle_controller"):
		_battle_scene.call("bind_battle_controller", battle_controller)
	if map_snapshot != null and _battle_scene.has_method("apply_background_texture"):
		_battle_scene.call("apply_background_texture", map_snapshot)
	if _battle_scene.has_method("_on_battle_started"):
		_battle_scene.call("_on_battle_started", payload)
	if world_mount != null:
		world_mount.visible = false

func _capture_world_map_snapshot() -> Texture2D:
	if _world_controller == null:
		return null
	if _world_controller.has_method("capture_current_map_snapshot_async"):
		return await _world_controller.capture_current_map_snapshot_async() as Texture2D
	if _world_controller.has_method("capture_current_map_snapshot"):
		return _world_controller.capture_current_map_snapshot() as Texture2D
	return null

func _unmount_battle_scene() -> void:
	if _battle_scene != null:
		_battle_scene.queue_free()
		_battle_scene = null
	if world_mount != null:
		world_mount.visible = true

# 处理底层 WebSocket 关闭事件，并在需要时跳回登录页。
func _on_websocket_closed(code: int, reason: String) -> void:
	if code == -1 and reason.is_empty():
		return
	_append_log("WebSocket 已关闭: %d %s" % [code, reason])
	if App.is_reconnecting():
		_append_log("检测到可恢复会话，正在尝试断线重连。")
		return
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

func _create_runtime_ui() -> void:
	_create_main_menu()
	_create_player_panel()
	_create_npc_menu()
	_create_npc_list_menu()
	_create_pvp_target_menu()
	_create_pvp_invite_dialog()

func _create_main_menu() -> void:
	_main_menu = MAIN_MENU_SCENE.instantiate() as CanvasLayer
	if _main_menu == null:
		return
	add_child(_main_menu)
	if _main_menu.has_signal("menu_item_selected"):
		_main_menu.connect("menu_item_selected", Callable(self, "_on_main_menu_item_selected"))
	if _main_menu.has_signal("menu_closed"):
		_main_menu.connect("menu_closed", Callable(self, "_on_runtime_menu_closed"))

func _create_player_panel() -> void:
	_player_panel = PLAYER_PANEL_SCENE.instantiate() as CanvasLayer
	if _player_panel == null:
		return
	add_child(_player_panel)
	if _player_panel.has_signal("menu_closed"):
		_player_panel.connect("menu_closed", Callable(self, "_on_runtime_menu_closed"))

func _create_npc_menu() -> void:
	_npc_menu = NPC_MENU_SCENE.instantiate() as CanvasLayer
	if _npc_menu == null:
		return
	add_child(_npc_menu)
	if _npc_menu.has_signal("option_selected"):
		_npc_menu.connect("option_selected", Callable(self, "_on_npc_menu_option_selected"))
	if _npc_menu.has_signal("menu_closed"):
		_npc_menu.connect("menu_closed", Callable(self, "_on_runtime_menu_closed"))

func _create_npc_list_menu() -> void:
	_npc_list_menu = NPC_LIST_MENU_SCENE.instantiate() as CanvasLayer
	if _npc_list_menu == null:
		return
	add_child(_npc_list_menu)
	if _npc_list_menu.has_signal("npc_selected"):
		_npc_list_menu.connect("npc_selected", Callable(self, "_on_npc_selected"))
	if _npc_list_menu.has_signal("menu_closed"):
		_npc_list_menu.connect("menu_closed", Callable(self, "_on_npc_list_closed"))

func _create_pvp_target_menu() -> void:
	_pvp_target_menu = NPC_LIST_MENU_SCENE.instantiate() as CanvasLayer
	if _pvp_target_menu == null:
		return
	add_child(_pvp_target_menu)
	if _pvp_target_menu.has_signal("npc_selected"):
		_pvp_target_menu.connect("npc_selected", Callable(self, "_on_pvp_target_selected"))
	if _pvp_target_menu.has_signal("menu_closed"):
		_pvp_target_menu.connect("menu_closed", Callable(self, "_on_runtime_menu_closed"))

func _create_pvp_invite_dialog() -> void:
	_pvp_invite_dialog = ConfirmationDialog.new()
	_pvp_invite_dialog.title = "PVP 邀请"
	_pvp_invite_dialog.dialog_text = "收到新的 PVP 邀请。"
	_pvp_invite_dialog.ok_button_text = "接受"
	_pvp_invite_dialog.cancel_button_text = "拒绝"
	_pvp_invite_dialog.initial_position = Window.WINDOW_INITIAL_POSITION_CENTER_MAIN_WINDOW_SCREEN
	_pvp_invite_dialog.confirmed.connect(_on_pvp_invite_confirmed)
	_pvp_invite_dialog.canceled.connect(_on_pvp_invite_canceled)
	add_child(_pvp_invite_dialog)

func _on_runtime_menu_closed() -> void:
	_set_runtime_menu_locked(false)

func _on_npc_list_closed() -> void:
	if _opening_npc_menu_from_list:
		_opening_npc_menu_from_list = false
		return
	_set_runtime_menu_locked(false)

func _on_npc_selected(npc_data: Dictionary) -> void:
	var entity_id: int = int(npc_data.get("entity_id", 0))
	if entity_id <= 0:
		return
	_opening_npc_menu_from_list = true
	if _npc_list_menu != null:
		_npc_list_menu.call("close_menu")
	_append_log("通过列表选择 NPC: %s (%d)" % [str(npc_data.get("npc_name", "未知 NPC")), entity_id])
	App.request_interact(entity_id)

func _on_pvp_target_selected(target_data: Dictionary) -> void:
	var target_player_id: int = int(target_data.get("target_player_id", target_data.get("entity_id", 0)))
	if target_player_id <= 0:
		_append_log("PVP 目标数据无效，无法发起挑战。")
		return
	if _pvp_target_menu != null and _pvp_target_menu.has_method("close_menu"):
		_pvp_target_menu.call("close_menu")
	_append_log("发起 PVP 挑战: %s (%d)" % [str(target_data.get("npc_name", target_data.get("name", "未知玩家"))), target_player_id])
	App.request_pvp_challenge(target_player_id)

func _on_main_menu_item_selected(item: Dictionary) -> void:
	var label: String = str(item.get("label", ""))
	if label != "全服竞技场":
		return
	if _main_menu != null and _main_menu.has_method("close_menu"):
		_main_menu.call("close_menu")
	_open_pvp_target_menu()

func _on_npc_menu_option_selected(option: Dictionary) -> void:
	if _npc_menu != null and _npc_menu.has_method("close_menu"):
		_npc_menu.call("close_menu")
	var entity_id: int = int(option.get("entity_id", 0))
	var entry_id: String = str(option.get("id", ""))
	var entry_type: String = str(option.get("entry_type", ""))
	var quest_id: int = int(option.get("quest_id", 0))
	var quest_state: String = str(option.get("quest_state", ""))
	if entity_id <= 0 or entry_id.is_empty():
		return
	if entry_type == "quest" and quest_id > 0:
		if quest_state == "AVAILABLE":
			App.accept_quest(quest_id, entity_id)
			var accept_dialogue := _resolve_quest_dialogue_binding(entity_id, "accept")
			if not accept_dialogue.is_empty():
				_show_npc_dialogue(accept_dialogue, str(option.get("npc_name", "NPC")))
		elif quest_state == "READY_TO_SUBMIT":
			App.submit_quest(quest_id, entity_id)
			var submit_dialogue := _resolve_quest_dialogue_binding(entity_id, "submit")
			if not submit_dialogue.is_empty():
				_show_npc_dialogue(submit_dialogue, str(option.get("npc_name", "NPC")))
		App.request_interact(entity_id)
		return
	if entry_type == "warehouse":
		_append_log("打开仓库面板: %s (%d)" % [str(option.get("npc_name", "仓库 NPC")), entity_id])
		if _player_panel != null and _player_panel.has_method("open_warehouse_menu"):
			_player_panel.call("open_warehouse_menu", entity_id)
			_set_runtime_menu_locked(true)
		return
	if entry_type == "battle":
		App.request_npc_action(entity_id, entry_id)
		return
	App.request_npc_action(entity_id, entry_id)

func _on_pvp_challenge_responded(payload: Dictionary) -> void:
	var accepted: bool = bool(payload.get("accepted", false))
	var reason: String = str(payload.get("reason", ""))
	var target_player_id: int = int(payload.get("target_player_id", 0))
	if accepted:
		_append_log("PVP 挑战已发送，目标玩家: %d。" % target_player_id)
		return
	_append_log("PVP 挑战发送失败: %s" % reason)

func _on_pvp_challenge_received(payload: Dictionary) -> void:
	_pending_pvp_invite = payload.duplicate(true)
	var challenger_variant: Variant = payload.get("challenger", {})
	var challenger: Dictionary = challenger_variant if challenger_variant is Dictionary else {}
	var challenger_name: String = str(challenger.get("name", "未知玩家"))
	var challenger_level: int = int(challenger.get("level", 0))
	if _pvp_invite_dialog != null:
		_pvp_invite_dialog.dialog_text = "%s 向你发起了 PVP 挑战。\n等级: %d\n是否接受？" % [challenger_name, challenger_level]
		_pvp_invite_dialog.popup_centered_clamped(Vector2i(320, 180))
	_set_runtime_menu_locked(true)
	_append_log("收到 PVP 邀请: %s" % challenger_name)

func _on_pvp_challenge_reply_responded(payload: Dictionary) -> void:
	var accepted: bool = bool(payload.get("accepted", false))
	var reason: String = str(payload.get("reason", ""))
	if accepted:
		_append_log("PVP 邀请应答已发送: %s" % reason)
		return
	_append_log("PVP 邀请应答失败: %s" % reason)

func _on_pvp_invite_confirmed() -> void:
	var challenge_id: int = int(_pending_pvp_invite.get("challenge_id", 0))
	_set_runtime_menu_locked(false)
	if challenge_id <= 0:
		return
	App.reply_pvp_challenge(challenge_id, true)
	_pending_pvp_invite.clear()

func _on_pvp_invite_canceled() -> void:
	var challenge_id: int = int(_pending_pvp_invite.get("challenge_id", 0))
	_set_runtime_menu_locked(false)
	if challenge_id <= 0:
		return
	App.reply_pvp_challenge(challenge_id, false)
	_pending_pvp_invite.clear()

func _build_npc_menu_options(payload: Dictionary) -> Array[Dictionary]:
	var options: Array[Dictionary] = []
	var entity_id: int = int(payload.get("entity_id", 0))
	var menu_entries_variant: Variant = payload.get("menu_entries", [])
	if menu_entries_variant is not Array:
		return options
	for entry_variant in menu_entries_variant:
		if entry_variant is not Dictionary:
			continue
		var entry: Dictionary = entry_variant
		var option := entry.duplicate(true)
		option["id"] = str(entry.get("entry_id", ""))
		option["label"] = str(entry.get("title", entry.get("entry_id", "")))
		if str(entry.get("subtitle", "")).strip_edges() != "":
			option["label"] += " - %s" % str(entry.get("subtitle", ""))
		option["entity_id"] = entity_id
		option["npc_name"] = str(payload.get("npc_name", "NPC"))
		options.append(option)
	return options

func _collect_nearby_npc_entries() -> Array[Dictionary]:
	var entries: Array[Dictionary] = []
	for entity_id_variant in GameState.nearby_entities.keys():
		var entity_variant: Variant = GameState.nearby_entities.get(entity_id_variant, {})
		if entity_variant is not Dictionary:
			continue
		var entity: Dictionary = entity_variant
		entries.append({
			"entity_id": int(entity.get("entity_id", entity_id_variant)),
			"npc_name": str(entity.get("name", entity.get("npc_name", "NPC"))),
			"portrait_path": "res://asset/口袋所有形象/imgs/51.png",
		})
	return entries

func _collect_nearby_player_entries() -> Array[Dictionary]:
	var entries: Array[Dictionary] = []
	for entity_id_variant in GameState.nearby_entities.keys():
		var entity_variant: Variant = GameState.nearby_entities.get(entity_id_variant, {})
		if entity_variant is not Dictionary:
			continue
		var entity: Dictionary = entity_variant
		var entity_type: int = int(entity.get("entity_type", 0))
		# PVP 挑战目标必须使用服务端显式下发的 player_id，而不是继续依赖
		# entity_id 的偶然重合关系。
		var target_player_id: int = int(entity.get("player_id", 0))
		if entity_type != PLAYER_ENTITY_TYPE:
			continue
		if target_player_id <= 0 or target_player_id == GameState.player_id:
			continue
		var entry := {
			"entity_id": int(entity.get("entity_id", entity_id_variant)),
			"target_player_id": target_player_id,
			"npc_name": str(entity.get("name", "附近玩家")),
			"portrait_path": "res://asset/口袋所有形象/imgs/51.png",
		}
		entries.append(entry)
	return entries

func _open_pvp_target_menu() -> void:
	if GameState.is_in_battle:
		_append_log("战斗中无法发起新的 PVP 挑战。")
		return
	if _pvp_target_menu == null:
		return
	var nearby_players: Array[Dictionary] = _collect_nearby_player_entries()
	if nearby_players.is_empty():
		_append_log("附近没有可挑战的玩家。")
		return
	_pvp_target_menu.call("configure", "选择挑战玩家", nearby_players)
	_pvp_target_menu.call("open_menu")
	_set_runtime_menu_locked(true)

func _resolve_npc_dialogue_binding(option: Dictionary) -> Dictionary:
	var entity_id: int = int(option.get("entity_id", 0))
	var entry_id: String = str(option.get("id", option.get("entry_id", "")))
	var entry_type: String = str(option.get("entry_type", ""))
	if entity_id == 93001:
		if entry_id == "dialog_market_news" or entry_id == "talk" or entry_type == "dialogue":
			return {"resource_path": LIMENG_DIALOGUE_PATH, "title": "talk"}
	if entity_id == 93002:
		if entry_id == "dialog_trade_tip" or entry_id == "talk" or entry_type == "dialog":
			return {"resource_path": GENERIC_NPC_DIALOGUE_PATH, "title": "start"}
	return {}

func _resolve_quest_dialogue_binding(entity_id: int, action: String) -> Dictionary:
	if entity_id != 93001:
		return {}
	match action:
		"accept":
			return {"resource_path": LIMENG_DIALOGUE_PATH, "title": "quest_accepted"}
		"submit":
			return {"resource_path": LIMENG_DIALOGUE_PATH, "title": "quest_completed"}
		_:
			return {}

func _show_npc_dialogue(binding: Dictionary, npc_name: String) -> void:
	var resource_path: String = str(binding.get("resource_path", ""))
	var title: String = str(binding.get("title", ""))
	if resource_path.is_empty():
		return
	var resource_variant: Variant = load(resource_path)
	if resource_variant is not DialogueResource:
		_append_log("对话资源加载失败: %s" % resource_path)
		return
	var balloon_variant: Variant = DialogueManager.show_dialogue_balloon(resource_variant, title)
	if balloon_variant is Node:
		var balloon: Node = balloon_variant
		balloon.set("npc_portrait", load("res://asset/口袋所有形象/imgs/51.png"))
	_append_log("打开对话: %s" % npc_name)

func _set_runtime_menu_locked(locked: bool) -> void:
	if _world_controller != null and _world_controller.has_method("set_runtime_input_locked"):
		_world_controller.call("set_runtime_input_locked", locked)

func _close_runtime_menus() -> void:
	for layer in [_main_menu, _player_panel, _npc_menu, _npc_list_menu, _pvp_target_menu]:
		if layer != null and layer.has_method("close_menu"):
			layer.call("close_menu")
	if _pvp_invite_dialog != null:
		_pvp_invite_dialog.hide()
	_set_runtime_menu_locked(false)

func _has_blocking_ui_open(except: String = "") -> bool:
	if except != "main_menu" and _main_menu != null and _main_menu.visible:
		return true
	if except != "player_panel" and _player_panel != null and _player_panel.visible:
		return true
	if except != "npc_menu" and _npc_menu != null and _npc_menu.visible:
		return true
	if except != "npc_list" and _npc_list_menu != null and _npc_list_menu.visible:
		return true
	if except != "pvp_list" and _pvp_target_menu != null and _pvp_target_menu.visible:
		return true
	if except != "pvp_invite" and _pvp_invite_dialog != null and _pvp_invite_dialog.visible:
		return true
	return false

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
func _fade_overlay(target_alpha: float, duration: float = TRANSITION_DURATION) -> void:
	# 创建当前使用的过渡补间动画。
	var tween := create_tween()
	tween.tween_property(transition_overlay, "color:a", target_alpha, duration)
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
