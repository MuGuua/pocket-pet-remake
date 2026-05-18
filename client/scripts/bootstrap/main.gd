extends Node

const WORLD_SCENE := preload("res://scenes/world/world_scene.tscn")
const BATTLE_SCENE := preload("res://scenes/battle/battle_scene.tscn")
const LOGIN_SCENE_PATH := "res://scenes/auth/login_scene.tscn"
const TRANSITION_DURATION := 0.18

@onready var gameplay_area: Control = %GameplayArea
@onready var gameplay_viewport: SubViewport = $GameplayArea/GameplayViewportContainer/GameplayViewport
@onready var world_mount: Node2D = $GameplayArea/GameplayViewportContainer/GameplayViewport/WorldMount
@onready var battle_mount: Control = $GameplayArea/GameplayViewportContainer/GameplayViewport/BattleMount
@onready var pet_controller: Node = %PetController
@onready var battle_controller: Node = %BattleController
@onready var bag_controller: Node = %BagController
@onready var hud_root: RuntimeHud = %HudRoot
@onready var transition_overlay: ColorRect = %TransitionOverlay

var _world_controller: Node
var _battle_scene: Control
var _redirecting_to_login: bool = false
var _runtime_data_requested: bool = false

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

func _exit_tree() -> void:
	_unregister_routes()

func _mount_world_scene() -> void:
	_world_controller = WORLD_SCENE.instantiate()
	world_mount.add_child(_world_controller)
	_sync_world_render_frame()
	if _world_controller.has_signal("scene_loaded"):
		_world_controller.connect("scene_loaded", Callable(self, "_on_world_scene_loaded"))
	if _world_controller.has_signal("player_position_changed"):
		_world_controller.connect("player_position_changed", Callable(self, "_on_player_position_changed"))
	if _world_controller.has_signal("scene_transition_requested"):
		_world_controller.connect("scene_transition_requested", Callable(self, "_on_scene_transition_requested"))
	if _world_controller.has_signal("scene_transition_failed"):
		_world_controller.connect("scene_transition_failed", Callable(self, "_on_scene_transition_failed"))
	_append_log("世界场景已挂载。")

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

	MessageRouter.register_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.register_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.register_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))

	MessageRouter.register_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.register_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.register_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.register_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))

	MessageRouter.register_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.register_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))

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
	MessageRouter.unregister_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.unregister_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.unregister_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))
	MessageRouter.unregister_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.unregister_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))

func _connect_signals() -> void:
	App.notice_received.connect(_on_notice_received)
	App.kicked.connect(_on_kicked)
	gameplay_area.resized.connect(_sync_world_render_frame)
	hud_root.challenge_requested.connect(_on_challenge_requested)
	hud_root.pet_requested.connect(_on_pet_requested)
	hud_root.lineup_requested.connect(_on_lineup_requested)
	hud_root.bag_requested.connect(_on_bag_requested)
	hud_root.lineup_submit_requested.connect(_on_lineup_submit_requested)

	if battle_controller.has_signal("interact_responded"):
		battle_controller.connect("interact_responded", Callable(self, "_on_interact_responded"))
	if battle_controller.has_signal("action_responded"):
		battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
	if battle_controller.has_signal("battle_started"):
		battle_controller.connect("battle_started", Callable(self, "_on_battle_started"))
	if battle_controller.has_signal("battle_finished"):
		battle_controller.connect("battle_finished", Callable(self, "_on_battle_finished"))

	GameState.session_changed.connect(_refresh_view)
	GameState.world_snapshot_changed.connect(_refresh_view)
	GameState.battle_changed.connect(_sync_battle_visibility)
	GameState.battle_changed.connect(_refresh_view)
	NetClient.connection_state_changed.connect(_on_connection_state_changed)
	NetClient.websocket_closed.connect(_on_websocket_closed)

func _on_notice_received(message: String) -> void:
	_append_log("提示: %s" % message)

func _on_kicked(reason: String) -> void:
	_append_log("连接已被服务端断开: %s" % reason)
	_return_to_login_scene()

func _on_connection_state_changed(state: String) -> void:
	_refresh_view()
	_append_log("WebSocket 状态 -> %s" % state)
	if state == "closed" and not _redirecting_to_login:
		_return_to_login_scene()

func _on_world_scene_loaded(scene_id: String) -> void:
	_append_log("已进入场景: %s" % scene_id)
	if not _runtime_data_requested:
		_runtime_data_requested = true
		App.request_pet_list()
		App.request_bag_list()
		_append_log("正在同步宠物与背包摘要。")
	_refresh_view()

func _on_player_position_changed(position: Vector2) -> void:
	var player_text := "玩家: %s @ (%.0f, %.0f)" % [
		str(GameState.player_snapshot.get("name", "未登录")),
		position.x,
		position.y,
	]
	hud_root.set_header_texts(
		str(hud_root.status_label.text),
		str(hud_root.scene_label.text),
		player_text
	)

func _on_scene_transition_requested(from_scene_id: int, to_scene_id: int) -> void:
	_append_log("请求切换地图: %d -> %d" % [from_scene_id, to_scene_id])

func _on_scene_transition_failed(reason: String) -> void:
	_append_log("地图切换失败: %s" % reason)

func _on_interact_responded(accepted: bool, reason: String) -> void:
	_append_log("交互结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])
	_refresh_view()

func _on_action_responded(accepted: bool, reason: String) -> void:
	_append_log("战斗动作结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])

func _on_battle_started(payload: Dictionary) -> void:
	_append_log("进入战斗场景。")
	_mount_battle_scene()
	_sync_battle_visibility()
	if payload.has("battle_id"):
		_append_log("战斗ID: %s" % str(payload.get("battle_id", "")))

func _on_battle_finished(_payload: Dictionary) -> void:
	_append_log("战斗结束，返回世界场景。")
	_sync_battle_visibility()
	_unmount_battle_scene()
	_refresh_view()

func _on_websocket_closed(code: int, reason: String) -> void:
	if code == -1 and reason.is_empty():
		return
	_append_log("WebSocket 已关闭: %d %s" % [code, reason])
	if not _redirecting_to_login:
		_return_to_login_scene()

func _refresh_view() -> void:
	var status_text: String = ""
	var scene_text: String = ""
	var player_text: String = ""
	if GameState.is_in_battle:
		status_text = "连接状态: %s | 战斗中" % NetClient.get_connection_state()
		scene_text = "场景: Battle"
		player_text = "玩家: %s" % str(GameState.player_snapshot.get("name", "未登录"))
		hud_root.set_header_texts(status_text, scene_text, player_text)
		return

	status_text = "连接状态: %s | HTTP: %s | WS: %s" % [
		NetClient.get_connection_state(),
		_short_token(GameState.access_jwt),
		"ok" if GameState.is_ws_authenticated else "pending",
	]

	var scene_id := str(GameState.scene_snapshot.get("scene_id", "未进入"))
	scene_text = "场景: %s | 附近实体: %d" % [scene_id, GameState.nearby_entities.size()]

	var player_name := str(GameState.player_snapshot.get("name", "未登录"))
	player_text = "%s" % player_name
	if GameState.player_id > 0:
		player_text += " (#%d)" % GameState.player_id
	if GameState.player_snapshot.has("x") and GameState.player_snapshot.has("y"):
		player_text += " @ (%.0f, %.0f)" % [
			float(GameState.player_snapshot.get("x", 0.0)),
			float(GameState.player_snapshot.get("y", 0.0)),
		]
	hud_root.set_header_texts(status_text, scene_text, "玩家: %s" % player_text)

func _append_log(message: String) -> void:
	hud_root.append_log(message)

func _mount_battle_scene() -> void:
	if _battle_scene != null:
		return
	_battle_scene = BATTLE_SCENE.instantiate() as Control
	if _battle_scene == null:
		return
	battle_mount.add_child(_battle_scene)

func _unmount_battle_scene() -> void:
	if _battle_scene == null:
		return
	_battle_scene.queue_free()
	_battle_scene = null

func _sync_battle_visibility() -> void:
	var active := GameState.is_in_battle
	if active:
		_mount_battle_scene()
	world_mount.visible = not active
	battle_mount.visible = active

func _on_challenge_requested() -> void:
	if GameState.is_in_battle:
		return
	var entity_ids := GameState.nearby_entities.keys()
	if entity_ids.is_empty():
		_append_log("附近没有可挑战的NPC。")
		_refresh_view()
		return
	entity_ids.sort()
	var entity_id := int(entity_ids[0])
	_append_log("向服务端发起挑战，目标实体: %d" % entity_id)
	App.request_interact(entity_id)

func _on_pet_requested() -> void:
	_append_log("请求宠物列表。")
	App.request_pet_list()

func _on_lineup_requested() -> void:
	_append_log("请求编队摘要。")
	App.request_pet_list()

func _on_bag_requested() -> void:
	_append_log("请求背包列表。")
	App.request_bag_list()

func _on_lineup_submit_requested(pet_uids: Array[int]) -> void:
	if pet_uids.is_empty():
		_append_log("编队不能为空。")
		return
	_append_log("提交编队: %s" % str(pet_uids))
	App.set_pet_lineup(pet_uids)

func _return_to_login_scene() -> void:
	if _redirecting_to_login:
		return
	_redirecting_to_login = true
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(1.0)
	NetClient.disconnect_from_server()
	GameState.reset_session_state()
	get_tree().change_scene_to_file(LOGIN_SCENE_PATH)

func _play_fade_in() -> void:
	transition_overlay.color.a = 1.0
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(0.0)
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

func _fade_overlay(target_alpha: float) -> void:
	var tween := create_tween()
	tween.tween_property(transition_overlay, "color:a", target_alpha, TRANSITION_DURATION)
	await tween.finished

func _short_token(token: String) -> String:
	if token.is_empty():
		return "none"
	if token.length() <= 12:
		return token
	return "%s...%s" % [token.substr(0, 6), token.substr(token.length() - 4, 4)]

func _sync_world_render_frame() -> void:
	if gameplay_viewport != null:
		var viewport_width := maxi(int(round(gameplay_area.size.x)), 1)
		var viewport_height := maxi(int(round(gameplay_area.size.y)), 1)
		gameplay_viewport.size = Vector2i(viewport_width, viewport_height)
	if _world_controller == null:
		return
	if _world_controller.has_method("set_render_frame_size"):
		_world_controller.call("set_render_frame_size", gameplay_area.size)
