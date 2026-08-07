extends Node


# 世界场景资源的预加载引用。
const WORLD_SCENE := preload("res://scenes/world/world_scene.tscn")
const BATTLE_SCENE := preload("res://scenes/battle/battle_scene.tscn")
const MAIN_MENU_SCENE := preload("res://scenes/ui/main_menu.tscn")
const PLAYER_PANEL_SCENE := preload("res://scenes/ui/status_panels/player_status_panel.tscn")
const PET_PANEL_SCENE := preload("res://scenes/ui/pet/pet_status_panel.tscn")
const BAG_PANEL_SCENE := preload("res://scenes/ui/bag/bag_panel.tscn")
const TASK_PANEL_SCENE := preload("res://scenes/ui/task/task_panel.tscn")
const OPTION_LIST_PANEL_SCENE := preload(OptionListPanel.SCENE_PATH)
const RUNTIME_PROGRESS_OVERLAY_SCENE := preload(RuntimeProgressOverlay.SCENE_PATH)
const NPC_DIALOGUE_PANEL_SCENE := preload("res://scenes/ui/npc_dialogue_panel.tscn")
const NPC_SHOP_PANEL_SCENE := preload("res://scenes/ui/npc_shop_panel.tscn")
const RewardPopupScene := preload("res://scenes/ui/common/reward_popup.tscn")
const ConfirmPromptPopupScene := preload(ConfirmPromptPopup.SCENE_PATH)
const GRID_SPREAD_SCENE := preload("res://scenes/ui/grid_spread.tscn")
# 返回登录页时使用的场景路径。
const LOGIN_SCENE_PATH := "res://scenes/auth/login_scene.tscn"
# 场景切换遮罩淡入淡出的持续时间。
const TRANSITION_DURATION := 0.18
# 挂机暗雷轮询间隔（秒）；玩家开启挂机后每轮结算结束都会重新倒计时。
const AUTO_WILD_ENCOUNTER_INTERVAL_SEC := 5.0
# 世界地图切换整段过渡时长（秒）；前半渐入、中点换图、后半渐出各占一半。
const SCENE_MAP_TRANSITION_DURATION := 0.37
## 等待服务端世界快照和地图挂载完成的最长时间；超时后必须解除黑屏，避免客户端永久卡在转场。
const SCENE_MAP_LOAD_TIMEOUT_MSEC: int = 15000
## 单个 NPC 菜单预加载的最长等待时间；超时后跳过该 NPC，避免地图输入永久锁定。
# 过渡遮罩 CanvasLayer 层级，需高于战斗弹窗层以便盖住切换过程。
const TRANSITION_LAYER := 20
# 世界场景内战斗弹窗的固定尺寸；与当前全局设计分辨率保持一致，
# 保证 UI、地图、角色与战斗继续共用同一套分辨率口径。
const BATTLE_MODAL_SIZE := Vector2(780.0, 1440.0)
# 战斗弹窗 CanvasLayer 层级，需高于运行时菜单与 HUD。
const BATTLE_MODAL_LAYER := 10
# 当前客户端默认只允许从附近玩家列表中发起 PVP 挑战，避免没有明确目标时误发请求。
const PLAYER_ENTITY_TYPE: int = 1
# 附近实体中的 NPC 类型；挂机目标列表需要显式排除这类运营/交互单位。
const NPC_ENTITY_TYPE: int = 2

# 上部游戏显示区域的根节点。
@onready var gameplay_area: Control = %GameplayArea
# 世界场景实例的挂载节点。
@onready var world_mount: Control = $GameplayArea/WorldMount
# 宠物相关消息处理控制器。
@onready var pet_controller: Node = %PetController
# 玩家成长相关消息处理控制器。
@onready var player_controller: Node = %PlayerController
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

# 奖励弹窗实例，战斗与任务结算后复用。
var _reward_popup: RewardPopup = null
## 流程确认弹窗；进入地图、任务前置、动画结束与任务完成提示共用同一实例。
var _flow_confirm_popup: ConfirmPromptPopup = null
# 退出游戏前使用的二次确认弹窗，避免误触设置菜单直接关闭客户端。
var _quit_confirm_popup: ConfirmPromptPopup = null
# 当前挂载的世界控制器实例引用。
var _world_controller: Node
# 战斗表现场景实例；非战斗态时为 null。
var _battle_scene: Control = null
# 战斗弹窗所在的 CanvasLayer，包含半透明遮罩与居中战斗面板。
var _battle_modal_layer: CanvasLayer = null
# 战斗弹窗半透明遮罩，用于阻断世界与 HUD 的点击。
var _battle_modal_dim: ColorRect = null
# 战斗弹窗居中容器，承载战斗场景实例。
var _battle_modal_host: CenterContainer = null
# 全屏过渡遮罩专用层，保证战斗渐入渐出盖在战斗弹窗之上。
var _transition_canvas_layer: CanvasLayer = null
# 进入战斗时使用的网格铺展转场实例。
var _grid_spread_transition: GridSpreadTransition = null
# 标记当前是否正在跳回登录页，避免重复切场景。
var _redirecting_to_login: bool = false
var _main_menu: CanvasLayer
var _player_panel: CanvasLayer
var _pet_panel: CanvasLayer
var _bag_panel: CanvasLayer
var _task_panel: CanvasLayer
var _npc_menu: OptionListPanel = null
var _npc_list_menu: OptionListPanel = null
var _pvp_target_menu: OptionListPanel = null
var _auto_encounter_target_menu: OptionListPanel = null
var _pvp_invite_dialog: ConfirmationDialog
var _opening_npc_menu_from_list: bool = false
var _runtime_data_requested: bool = false
# 缓存最近一次收到但尚未处理的 PVP 邀请载荷，供确认框按钮回调读取。
var _pending_pvp_invite: Dictionary = {}
# 结算弹窗关闭当帧继续吞掉输入，避免同一次按键触发移动或开菜单。
var _suppress_settlement_input_until_frame: int = -1
# 结算弹窗流程代次；新战斗开始时会递增，用于打断未关闭的升级/奖励弹窗链路。
var _popup_flow_generation: int = 0
# 串行处理 battle_finished，避免 4012 兜底与 4013 结算包并发时重复卸载或抢跑空结算。
var _battle_finish_handler_running: bool = false
# 若结算 handler 正在执行，暂存后续到达的权威 4013 结算包。
var _pending_battle_finish_payload: Dictionary = {}
# 当前运行中的 NPC 结构化剧情面板实例。
var _npc_dialogue_panel: NPCDialoguePanel = null
# 当前运行中的 NPC 商店面板实例。
var _npc_shop_panel: NPCShopPanel = null
# 当前运行中的客户端剧情动画播放器。
var _cinematic_player: CinematicPlayer = null
# 当前等待客户端剧情动画播完后继续推进的 NPC 实体标识。
var _active_dialogue_entity_id: int = 0
# 当前等待客户端剧情动画播完后继续推进的剧情实例标识。
var _active_dialogue_id: int = 0
# 当前等待客户端剧情动画播完后继续推进的剧情节点标识。
var _active_dialogue_node_id: String = ""
# 当前剧情所属 NPC 名称，供动作节点播完后恢复对话面板标题使用。
var _active_dialogue_npc_name: String = ""
## 当前 action 节点是否要求等待动画完成后再推进，避免非阻塞动画结束时重复发送继续请求。
var _waiting_blocking_cinematic: bool = false
## 当前等待播放完成 Ack 的服务端场景剧情触发器编号。
var _pending_scene_trigger_code: String = ""
## 当前场景剧情触发器是否要求播放期间锁住运行时菜单与玩家输入。
var _pending_scene_trigger_block_movement: bool = false
## 当前等待场景剧情完成确认回包的请求序列号。
var _pending_scene_trigger_ack_seq: int = 0
## 当前场景剧情动画结束后需要展示的一次性提示正文。
var _pending_scene_trigger_prompt_text: String = ""
## 当前由任务领取或交付触发的动画类型；空字符串表示不是任务动画。
var _active_quest_animation_action: String = ""
## 等待当前剧情结束后按顺序播放的任务动画。
var _queued_quest_animations: Array[Dictionary] = []
## 任务动画播放期间到达的服务端场景剧情触发载荷。
var _queued_scene_trigger_payload: Dictionary = {}
# NPC 交互/剧情请求的通用 loading 遮罩。
var _npc_request_loading: RuntimeProgressOverlay = null
# 背包面板是否正在执行「先 loading 再打开」流程，避免重复点击。
var _bag_panel_open_in_flight: bool = false
# 任务面板是否正在执行「先 loading 再打开」流程，避免重复点击。
var _task_panel_open_in_flight: bool = false
# 人物状态面板是否正在执行「先 loading 再打开」流程，避免重复点击。
var _player_panel_open_in_flight: bool = false
# 宠物状态面板是否正在执行「先 loading 再打开」流程，避免重复点击。
var _pet_panel_open_in_flight: bool = false
# 当前等待 NPC_MENU_RESP 的请求序列号。
var _pending_npc_menu_seq: int = 0
# 与 _pending_npc_menu_seq 对应的菜单回包缓存。
var _pending_npc_menu_payload: Dictionary = {}
## 当前 NPC 菜单请求对应的实体 ID，用于避免并发回包写入错误目标。
var _pending_npc_menu_entity_id: int = 0
## 当前 NPC 菜单请求发起时的场景 ID，用于隔离切图后的迟到响应与超时标记。
var _pending_npc_menu_scene_id: int = 0
## 当前 NPC 菜单请求完成后是否立即打开面板；地图预加载请求只写缓存。
var _pending_npc_menu_should_open: bool = false
## 当前地图已由服务端计算完成的 NPC 菜单缓存，键为 NPC 实体 ID。
var _scene_npc_menu_cache: Dictionary = {}
## NPC 菜单缓存所属的场景 ID，切图后旧场景数据不能继续使用。
var _scene_npc_menu_cache_scene_id: int = 0
## 标记当前是否正在后台请求当前地图全部 NPC 菜单；该状态不会锁定玩家输入。
var _npc_menu_preload_active: bool = false
## 当前地图 NPC 批量菜单请求序号，用于忽略切图后到达的旧请求结果。
var _pending_npc_menu_batch_seq: int = 0
## 当前地图 NPC 批量菜单请求所属场景 ID。
var _pending_npc_menu_batch_scene_id: int = 0
## 防止同一帧的多次世界快照更新重复排队 NPC 菜单补载任务。
var _npc_menu_refresh_scheduled: bool = false
# 当前等待 NPC_ACTION_RESP 的请求序列号。
var _pending_npc_action_seq: int = 0
# 与 _pending_npc_action_seq 对应的动作回包缓存。
var _pending_npc_action_payload: Dictionary = {}
# 当前等待 NPC_DIALOGUE_RESP 的请求序列号。
var _pending_dialogue_request_seq: int = 0
# 与 _pending_dialogue_request_seq 对应的剧情回包缓存。
var _pending_dialogue_payload: Dictionary = {}
# 当前打开的商店 NPC 实体 ID。
var _active_shop_entity_id: int = 0
# 当前等待 BUY_ITEM_RESP 的请求序列号。
var _pending_buy_item_seq: int = 0
# 与 _pending_buy_item_seq 对应的购买回包缓存。
var _pending_buy_item_payload: Dictionary = {}
# 标记是否正在播放地图切换的黑色遮罩过渡。
var _scene_map_transition_active: bool = false
# 新地图是否已在过渡中点之前加载完成，供中点换图后渐出。
var _scene_map_new_scene_ready: bool = false
# 地图切换是否被服务端拒绝，中点后会直接渐出回到旧场景。
var _scene_map_transition_failed: bool = false
# 地图过渡协程代次，避免连续切图时旧协程干扰新过渡。
var _scene_map_transition_generation: int = 0
# 当前遮罩透明度补间，切换前先 kill 避免并发动画冲突。
var _overlay_tween: Tween = null
# 玩家是否已开启暗雷挂机；只在当前地图支持暗雷时允许切换。
var _auto_wild_encounter_enabled: bool = false
# 当前挂机选择的目标实体标识；当前阶段仅用于客户端展示与日志，不改变服务端暗雷权威选怪逻辑。
var _auto_wild_encounter_target_entity_id: int = 0
# 当前挂机选择的目标名称；用于让玩家确认自己正在刷哪种单位。
var _auto_wild_encounter_target_name: String = ""
# 挂机倒计时代次；每次重新排程或关闭挂机都会递增，用于取消旧倒计时。
var _auto_wild_encounter_schedule_generation: int = 0
# 初始化主运行态，挂载世界场景并注册主链路消息与信号。
func _ready() -> void:
	App.bootstrap()
	if not GameState.is_ws_authenticated:
		call_deferred("_return_to_login_scene")
		return

	_ensure_transition_canvas_layer()
	transition_overlay.color.a = 1.0
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	_mount_world_scene()
	_register_routes()
	_connect_signals()
	_create_runtime_ui()
	_ensure_battle_modal_layer()
	_sync_world_render_frame()
	_append_log("主场景已就绪。")
	if GameState.world_entry_prepared:
		GameState.world_entry_prepared = false
		_append_log("正在应用登录页已拉取的世界快照。")
		if _world_controller != null and _world_controller.has_method("apply_prepared_world_entry"):
			_world_controller.call("apply_prepared_world_entry")
		else:
			_append_log("世界控制器未就绪，改为重新请求进入世界。")
			App.enter_world()
		await _play_fade_in()
	else:
		_append_log("正在请求进入世界。")
		App.enter_world()
		_play_fade_in()
	_refresh_view()

# 退出主场景时注销当前注册的业务路由。
func _exit_tree() -> void:
	# 使仍在等待下一帧或地图回包的异步流程立即失效，避免节点离树后继续访问空 SceneTree。
	_scene_map_transition_generation += 1
	_scene_map_transition_active = false
	_npc_menu_preload_active = false
	_pending_npc_menu_batch_seq = 0
	_pending_npc_menu_batch_scene_id = 0
	_unregister_routes()
	if App.notice_received.is_connected(_on_notice_received):
		App.notice_received.disconnect(_on_notice_received)
	if App.server_error_received.is_connected(_on_server_error_received):
		App.server_error_received.disconnect(_on_server_error_received)
	if App.kicked.is_connected(_on_kicked):
		App.kicked.disconnect(_on_kicked)
	if App.server_result_logged.is_connected(_on_server_result_logged):
		App.server_result_logged.disconnect(_on_server_result_logged)
	if gameplay_area != null and gameplay_area.resized.is_connected(_sync_world_render_frame):
		gameplay_area.resized.disconnect(_sync_world_render_frame)
	if GameState.session_changed.is_connected(_refresh_view):
		GameState.session_changed.disconnect(_refresh_view)
	if GameState.world_snapshot_changed.is_connected(_on_world_snapshot_changed):
		GameState.world_snapshot_changed.disconnect(_on_world_snapshot_changed)
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
	MessageRouter.register_handler(CommandIds.NPC_MENU_RESP, Callable(battle_controller, "handle_npc_menu_response"))
	MessageRouter.register_handler(CommandIds.NPC_MENU_BATCH_RESP, Callable(battle_controller, "handle_npc_menu_batch_response"))
	MessageRouter.register_handler(CommandIds.NPC_ACTION_RESP, Callable(battle_controller, "handle_npc_action_response"))
	MessageRouter.register_handler(CommandIds.NPC_DIALOGUE_RESP, Callable(battle_controller, "handle_npc_dialogue_response"))
	MessageRouter.register_handler(CommandIds.SCENE_TRIGGER_PUSH, Callable(self, "_on_scene_trigger_push"))
	MessageRouter.register_handler(CommandIds.PVP_CHALLENGE_RESP, Callable(battle_controller, "handle_pvp_challenge_response"))
	MessageRouter.register_handler(CommandIds.PVP_CHALLENGE_PUSH, Callable(battle_controller, "handle_pvp_challenge_push"))
	MessageRouter.register_handler(CommandIds.PVP_CHALLENGE_REPLY_RESP, Callable(battle_controller, "handle_pvp_challenge_reply_response"))

	MessageRouter.register_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.register_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.register_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))
	MessageRouter.register_handler(CommandIds.PET_ALLOCATE_ATTR_RESP, Callable(pet_controller, "handle_allocate_attr_response"))
	MessageRouter.register_handler(CommandIds.PET_SKILL_DETAIL_RESP, Callable(pet_controller, "handle_pet_skill_detail_response"))
	MessageRouter.register_handler(CommandIds.PET_ARTIFACT_EQUIP_RESP, Callable(pet_controller, "handle_pet_artifact_response"))
	MessageRouter.register_handler(CommandIds.PET_ARTIFACT_UNEQUIP_RESP, Callable(pet_controller, "handle_pet_artifact_response"))

	MessageRouter.register_handler(CommandIds.PLAYER_ALLOCATE_ATTR_RESP, Callable(player_controller, "handle_allocate_attr_response"))
	MessageRouter.register_handler(CommandIds.PLAYER_PROFILE_RESP, Callable(player_controller, "handle_profile_response"))
	MessageRouter.register_handler(CommandIds.PLAYER_EQUIPMENT_LIST_RESP, Callable(player_controller, "handle_equipment_list_response"))
	MessageRouter.register_handler(CommandIds.PLAYER_EQUIP_RESP, Callable(player_controller, "handle_equip_response"))
	MessageRouter.register_handler(CommandIds.PLAYER_UNEQUIP_RESP, Callable(player_controller, "handle_unequip_response"))
	MessageRouter.register_handler(CommandIds.PLAYER_EQUIPMENT_ENHANCE_RESP, Callable(player_controller, "handle_enhance_response"))
	MessageRouter.register_handler(CommandIds.PLAYER_EQUIPMENT_REPAIR_RESP, Callable(player_controller, "handle_repair_response"))

	MessageRouter.register_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.register_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.register_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.register_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))

	MessageRouter.register_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.register_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))
	MessageRouter.register_handler(CommandIds.USE_ITEM_RESP, Callable(bag_controller, "handle_use_item_response"))
	MessageRouter.register_handler(CommandIds.DROP_ITEM_RESP, Callable(bag_controller, "handle_drop_item_response"))
	MessageRouter.register_handler(CommandIds.CONTAINER_LIST_RESP, Callable(bag_controller, "handle_container_list"))
	MessageRouter.register_handler(CommandIds.WALLET_QUERY_RESP, Callable(bag_controller, "handle_wallet_query"))
	MessageRouter.register_handler(CommandIds.WALLET_UPDATE_PUSH, Callable(bag_controller, "handle_wallet_update"))
	MessageRouter.register_handler(CommandIds.BUY_ITEM_RESP, Callable(bag_controller, "handle_buy_item_response"))
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
	MessageRouter.unregister_handler(CommandIds.NPC_MENU_RESP, Callable(battle_controller, "handle_npc_menu_response"))
	MessageRouter.unregister_handler(CommandIds.NPC_MENU_BATCH_RESP, Callable(battle_controller, "handle_npc_menu_batch_response"))
	MessageRouter.unregister_handler(CommandIds.NPC_ACTION_RESP, Callable(battle_controller, "handle_npc_action_response"))
	MessageRouter.unregister_handler(CommandIds.NPC_DIALOGUE_RESP, Callable(battle_controller, "handle_npc_dialogue_response"))
	MessageRouter.unregister_handler(CommandIds.SCENE_TRIGGER_PUSH, Callable(self, "_on_scene_trigger_push"))
	MessageRouter.unregister_handler(CommandIds.PVP_CHALLENGE_RESP, Callable(battle_controller, "handle_pvp_challenge_response"))
	MessageRouter.unregister_handler(CommandIds.PVP_CHALLENGE_PUSH, Callable(battle_controller, "handle_pvp_challenge_push"))
	MessageRouter.unregister_handler(CommandIds.PVP_CHALLENGE_REPLY_RESP, Callable(battle_controller, "handle_pvp_challenge_reply_response"))
	MessageRouter.unregister_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
	MessageRouter.unregister_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
	MessageRouter.unregister_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))
	MessageRouter.unregister_handler(CommandIds.PET_ALLOCATE_ATTR_RESP, Callable(pet_controller, "handle_allocate_attr_response"))
	MessageRouter.unregister_handler(CommandIds.PET_SKILL_DETAIL_RESP, Callable(pet_controller, "handle_pet_skill_detail_response"))
	MessageRouter.unregister_handler(CommandIds.PET_ARTIFACT_EQUIP_RESP, Callable(pet_controller, "handle_pet_artifact_response"))
	MessageRouter.unregister_handler(CommandIds.PET_ARTIFACT_UNEQUIP_RESP, Callable(pet_controller, "handle_pet_artifact_response"))
	MessageRouter.unregister_handler(CommandIds.PLAYER_ALLOCATE_ATTR_RESP, Callable(player_controller, "handle_allocate_attr_response"))
	MessageRouter.unregister_handler(CommandIds.PLAYER_PROFILE_RESP, Callable(player_controller, "handle_profile_response"))
	MessageRouter.unregister_handler(CommandIds.PLAYER_EQUIPMENT_LIST_RESP, Callable(player_controller, "handle_equipment_list_response"))
	MessageRouter.unregister_handler(CommandIds.PLAYER_EQUIP_RESP, Callable(player_controller, "handle_equip_response"))
	MessageRouter.unregister_handler(CommandIds.PLAYER_UNEQUIP_RESP, Callable(player_controller, "handle_unequip_response"))
	MessageRouter.unregister_handler(CommandIds.PLAYER_EQUIPMENT_ENHANCE_RESP, Callable(player_controller, "handle_enhance_response"))
	MessageRouter.unregister_handler(CommandIds.PLAYER_EQUIPMENT_REPAIR_RESP, Callable(player_controller, "handle_repair_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_ACTION_RESP, Callable(battle_controller, "handle_battle_action_response"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
	MessageRouter.unregister_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))
	MessageRouter.unregister_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
	MessageRouter.unregister_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))
	MessageRouter.unregister_handler(CommandIds.USE_ITEM_RESP, Callable(bag_controller, "handle_use_item_response"))
	MessageRouter.unregister_handler(CommandIds.DROP_ITEM_RESP, Callable(bag_controller, "handle_drop_item_response"))
	MessageRouter.unregister_handler(CommandIds.CONTAINER_LIST_RESP, Callable(bag_controller, "handle_container_list"))
	MessageRouter.unregister_handler(CommandIds.WALLET_QUERY_RESP, Callable(bag_controller, "handle_wallet_query"))
	MessageRouter.unregister_handler(CommandIds.WALLET_UPDATE_PUSH, Callable(bag_controller, "handle_wallet_update"))
	MessageRouter.unregister_handler(CommandIds.BUY_ITEM_RESP, Callable(bag_controller, "handle_buy_item_response"))
	MessageRouter.unregister_handler(CommandIds.QUEST_LIST_RESP, Callable(quest_controller, "handle_quest_list"))
	MessageRouter.unregister_handler(CommandIds.QUEST_UPDATE_PUSH, Callable(quest_controller, "handle_quest_update"))
	MessageRouter.unregister_handler(CommandIds.QUEST_REMOVE_PUSH, Callable(quest_controller, "handle_quest_remove"))
	MessageRouter.unregister_handler(CommandIds.QUEST_ACCEPT_RESP, Callable(quest_controller, "handle_quest_accept_response"))
	MessageRouter.unregister_handler(CommandIds.QUEST_SUBMIT_RESP, Callable(quest_controller, "handle_quest_submit_response"))
	MessageRouter.unregister_handler(CommandIds.QUEST_TRACK_RESP, Callable(quest_controller, "handle_quest_track_response"))

# 绑定主运行态依赖的应用信号、HUD 交互信号和全局状态信号。
func _connect_signals() -> void:
	App.notice_received.connect(_on_notice_received)
	App.server_error_received.connect(_on_server_error_received)
	App.kicked.connect(_on_kicked)
	App.server_result_logged.connect(_on_server_result_logged)
	App.session_authenticated.connect(_on_session_authenticated)
	gameplay_area.resized.connect(_sync_world_render_frame)

	if battle_controller.has_signal("interact_responded"):
		battle_controller.connect("interact_responded", Callable(self, "_on_interact_responded"))
	if battle_controller.has_signal("action_responded"):
		battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
	if battle_controller.has_signal("interact_payload_received"):
		battle_controller.connect("interact_payload_received", Callable(self, "_on_interact_payload_received"))
	if battle_controller.has_signal("npc_menu_payload_received"):
		battle_controller.connect("npc_menu_payload_received", Callable(self, "_on_npc_menu_payload_received"))
	if battle_controller.has_signal("npc_menu_batch_payload_received"):
		battle_controller.connect("npc_menu_batch_payload_received", Callable(self, "_on_npc_menu_batch_payload_received"))
	if battle_controller.has_signal("npc_action_payload_received"):
		battle_controller.connect("npc_action_payload_received", Callable(self, "_on_npc_action_payload_received"))
	if battle_controller.has_signal("npc_dialogue_payload_received"):
		battle_controller.connect("npc_dialogue_payload_received", Callable(self, "_on_npc_dialogue_payload_received"))
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
	if quest_controller.has_signal("quest_settlement_popup_requested"):
		quest_controller.connect("quest_settlement_popup_requested", Callable(self, "_on_quest_settlement_popup_requested"))
	if quest_controller.has_signal("quest_animation_requested"):
		quest_controller.connect("quest_animation_requested", Callable(self, "_on_quest_animation_requested"))
	if bag_controller.has_signal("buy_item_responded"):
		bag_controller.connect("buy_item_responded", Callable(self, "_on_buy_item_responded"))

	GameState.session_changed.connect(_refresh_view)
	GameState.world_snapshot_changed.connect(_on_world_snapshot_changed)
	GameState.battle_changed.connect(_refresh_view)
	GameState.quests_changed.connect(_refresh_view)
	NetClient.connection_state_changed.connect(_on_connection_state_changed)
	NetClient.websocket_closed.connect(_on_websocket_closed)
	if hud_root.has_signal("avatar_pressed"):
		hud_root.connect("avatar_pressed", Callable(self, "_on_hud_avatar_pressed"))
	if hud_root.has_signal("pet_pressed"):
		hud_root.connect("pet_pressed", Callable(self, "_on_hud_pet_pressed"))
	if hud_root.has_signal("bag_pressed"):
		hud_root.connect("bag_pressed", Callable(self, "_on_hud_bag_pressed"))
	if hud_root.has_signal("task_pressed"):
		hud_root.connect("task_pressed", Callable(self, "_on_hud_task_pressed"))
	if hud_root.has_signal("return_to_login_pressed"):
		hud_root.connect("return_to_login_pressed", Callable(self, "_on_hud_return_to_login_pressed"))
	if hud_root.has_signal("quit_game_pressed"):
		hud_root.connect("quit_game_pressed", Callable(self, "_on_hud_quit_game_pressed"))
	if hud_root.has_signal("auto_encounter_pressed"):
		hud_root.connect("auto_encounter_pressed", Callable(self, "_on_hud_auto_encounter_pressed"))

func _unhandled_input(event: InputEvent) -> void:
	# 升级/奖励弹窗展示或刚关闭当帧，吞掉快捷键，避免误开其它菜单。
	if _is_settlement_input_blocked():
		get_viewport().set_input_as_handled()
		return

	# 战斗弹窗可见期间仅允许弹窗内交互，拦截主菜单、玩家面板与周围 NPC 列表快捷键。
	if _is_battle_modal_active():
		if event.is_action_pressed("open_main_menu") or event.is_action_pressed("open_player_panel") or event.is_action_pressed("open_scene_npc_list"):
			get_viewport().set_input_as_handled()
		return

	if event.is_action_pressed("open_main_menu"):
		if _main_menu == null or _is_battle_modal_active() or _is_settlement_input_blocked():
			return
		_toggle_root_runtime_panel(_main_menu, "main_menu")
		get_viewport().set_input_as_handled()
		return

	if event.is_action_pressed("open_player_panel"):
		if _player_panel == null or _is_battle_modal_active() or _is_settlement_input_blocked():
			return
		_toggle_root_runtime_panel(_player_panel, "player_panel")
		get_viewport().set_input_as_handled()
		return

	if not event.is_action_pressed("open_scene_npc_list"):
		return
	if GameState.is_in_battle or _is_battle_modal_active() or _npc_list_menu == null or _is_settlement_input_blocked():
		return
	if _npc_list_menu.visible:
		_npc_list_menu.call("close_menu")
		get_viewport().set_input_as_handled()
		return

	var nearby_npcs: Array[Dictionary] = _collect_nearby_npc_entries()
	if nearby_npcs.is_empty():
		return
	_close_other_root_panels("npc_list")
	_npc_list_menu.configure("周围 NPC", nearby_npcs, {
		"render_mode": OptionListPanel.RENDER_PORTRAIT_TEXT,
	})
	_npc_list_menu.call("open_menu")
	_set_runtime_menu_locked(true)
	get_viewport().set_input_as_handled()

# 处理服务端下发的普通提示信息。
func _on_notice_received(message: String) -> void:
	hud_root.show_notice(message)


# 处理服务端 ERROR_PUSH，统一使用可点击关闭的流程确认弹窗。
func _on_server_error_received(_error_code: int, message: String) -> void:
	var display_message: String = message.strip_edges()
	if display_message.is_empty():
		return
	await _show_flow_confirm_prompt_and_wait("提示", display_message, 24)

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

# 处理世界场景装载完成事件；地图就绪立即结束转场，其他数据在玩家进入后并发返回。
func _on_world_scene_loaded(scene_id: String) -> void:
	var scene_display_name: String = scene_id
	if _world_controller != null and _world_controller.has_method("get_current_scene_display_name"):
		scene_display_name = str(_world_controller.call("get_current_scene_display_name"))
	hud_root.set_scene_name(scene_display_name)
	_append_log("已进入场景: %s" % scene_display_name)
	if _scene_map_transition_active:
		_debug_scene_transition("world scene ready scene=%s" % scene_id)
		_scene_map_new_scene_ready = true
	_preload_scene_npc_menus(int(scene_id))
	if not _runtime_data_requested:
		_runtime_data_requested = true
		App.request_pet_list()
		App.request_bag_list()
		App.request_quest_list()
	_refresh_view()

# 处理本地玩家坐标变化，并刷新左上角 HUD 的场景内坐标。
func _on_player_position_changed(local_position: Vector2, _global_position: Vector2) -> void:
	hud_root.set_local_coordinates(local_position)

# 记录切图请求发起时的来源与目标场景，并启动「前半渐入 → 中点换图 → 后半渐出」过渡。
func _on_scene_transition_requested(from_scene_id: int, to_scene_id: int) -> void:
	_append_log("请求切换地图: %d -> %d" % [from_scene_id, to_scene_id])
	_scene_map_transition_generation += 1
	var generation: int = _scene_map_transition_generation
	_scene_map_new_scene_ready = false
	_scene_map_transition_failed = false
	_debug_scene_transition(
		"transition requested generation=%d from_scene=%d to_scene=%d" % [generation, from_scene_id, to_scene_id]
	)
	_lock_scene_visual_apply_for_transition()
	_run_scene_map_transition(generation)

# 记录切图失败原因；若遮罩仍在过渡中，标记失败供中点后渐出恢复。
## 处理服务端拒绝切图的结果；reason 为服务端权威提示文案。
func _on_scene_transition_failed(reason: String) -> void:
	_append_log("地图切换失败: %s" % reason)
	_debug_scene_transition("transition failed reason=%s active=%s" % [reason, str(_scene_map_transition_active)])
	# 复用全局移动端短提示，不在客户端重复判断地图等级或拼装准入文案。
	if not reason.is_empty():
		App.notice_received.emit(reason)
	if _scene_map_transition_active:
		_scene_map_transition_failed = true
	_unlock_scene_visual_apply_for_transition()

## 解析当前地图中指定 NPC 的展示纹理；菜单仅使用本地场景当前帧，不改变服务端菜单数据权威。
## entity_id 是服务端 NPC 实体 ID；无法解析时返回空纹理。
func _resolve_npc_header_portrait(entity_id: int) -> Texture2D:
	if entity_id <= 0 or _world_controller == null:
		return null
	if not _world_controller.has_method("get_npc_portrait_texture"):
		return null
	var portrait_variant: Variant = _world_controller.call("get_npc_portrait_texture", entity_id)
	if portrait_variant is Texture2D:
		return portrait_variant as Texture2D
	return null


func _on_npc_interaction_requested(entity_id: int, npc_name: String) -> void:
	_append_log("尝试与 NPC 交互: %s (%d)" % [npc_name, entity_id])
	_open_cached_scene_npc_menu(entity_id, npc_name)

## 使用进图阶段准备好的当前地图菜单打开 NPC 面板；该方法不会发送网络请求。
func _open_cached_scene_npc_menu(entity_id: int, npc_name: String) -> bool:
	var current_scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	if _scene_npc_menu_cache_scene_id != current_scene_id:
		_append_log("NPC 菜单缓存与当前地图不一致: %s (%d)" % [npc_name, entity_id])
		return false
	var payload_variant: Variant = _scene_npc_menu_cache.get(entity_id, {})
	if payload_variant is not Dictionary or (payload_variant as Dictionary).is_empty():
		_append_log("NPC 菜单尚未在进图阶段加载完成: %s (%d)" % [npc_name, entity_id])
		return false
	_open_npc_menu_from_payload((payload_variant as Dictionary).duplicate(true))
	return true

func _on_wild_encounter_responded(accepted: bool, reason: String) -> void:
	_append_log("暗雷遭遇: %s (%s)" % ["accepted" if accepted else "rejected", reason])
	if _auto_wild_encounter_enabled and not accepted:
		_schedule_next_auto_wild_encounter()
	_refresh_view()

func _on_interact_payload_received(_payload: Dictionary) -> void:
	pass

func _on_npc_menu_payload_received(payload: Dictionary) -> void:
	var response_entity_id: int = int(payload.get("entity_id", 0))
	if _pending_npc_menu_seq > 0:
		if response_entity_id == _pending_npc_menu_entity_id:
			_pending_npc_menu_payload = payload.duplicate(true)
		return
	# 超时后的迟到响应没有可靠的场景归属，直接忽略，避免污染新地图缓存。
	return

## 接收当前地图 NPC 批量菜单；旧地图迟到回包只结束旧请求，不写入新地图缓存。
func _on_npc_menu_batch_payload_received(payload: Dictionary) -> void:
	var response_scene_id: int = int(payload.get("scene_id", 0))
	var current_scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	if response_scene_id != _pending_npc_menu_batch_scene_id:
		return
	_pending_npc_menu_batch_seq = 0
	_pending_npc_menu_batch_scene_id = 0
	_npc_menu_preload_active = false
	if response_scene_id <= 0 or response_scene_id != current_scene_id:
		return
	if not bool(payload.get("accepted", false)):
		return
	var menus_variant: Variant = payload.get("menus", [])
	if menus_variant is not Array:
		return
	for menu_variant: Variant in menus_variant:
		if menu_variant is not Dictionary:
			continue
		var menu_payload: Dictionary = menu_variant as Dictionary
		if bool(menu_payload.get("accepted", false)):
			_cache_scene_npc_menu(menu_payload)
		else:
			_mark_scene_npc_menu_attempted(response_scene_id, int(menu_payload.get("entity_id", 0)))
	_schedule_missing_scene_npc_menu_preload()

func _on_npc_action_payload_received(payload: Dictionary) -> void:
	if _pending_npc_action_seq > 0:
		_pending_npc_action_payload = payload.duplicate(true)
		return
	_handle_npc_action_payload(payload)

## 处理 NPC 剧情继续/选项选择后的统一节点回包。
func _on_npc_dialogue_payload_received(payload: Dictionary) -> void:
	if _pending_dialogue_request_seq > 0:
		_pending_dialogue_payload = payload.duplicate(true)
		return
	_handle_npc_dialogue_payload(payload)

# 处理世界交互回执，并刷新主视图显示。
func _on_interact_responded(accepted: bool, reason: String) -> void:
	_append_log("交互结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])
	_refresh_view()

# 处理战斗动作回执，并记录服务端接受或拒绝结果。
func _on_action_responded(accepted: bool, reason: String) -> void:
	_append_log("战斗动作结果: %s (%s)" % ["accepted" if accepted else "rejected", reason])

# 处理战斗开始事件，在世界场景上方弹出战斗面板。
func _on_battle_started(payload: Dictionary) -> void:
	_append_log("收到战斗开始事件。")
	_cancel_auto_wild_encounter_schedule()
	hud_root.set_player_status_visible(false)
	_close_all_blocking_popups_for_battle()
	if payload.has("battle_id"):
		_append_log("战斗ID: %s" % str(payload.get("battle_id", "")))
	await _enter_battle_with_transition(payload)
	_sync_world_player_battle_pose()
	_set_runtime_menu_locked(true)
	_refresh_view()

## 网格铺展转场：先盖住世界 → 挂载战斗弹窗 → 再揭开露出战斗界面。
func _enter_battle_with_transition(payload: Dictionary) -> void:
	_ensure_grid_spread_transition()
	if _grid_spread_transition == null:
		_mount_battle_popup(payload)
		return
	_grid_spread_transition.prepare_grid()
	await _grid_spread_transition.play_cover()
	_mount_battle_popup(payload)
	if not await _wait_for_next_process_frame():
		return
	await _grid_spread_transition.play_reveal()

# 处理战斗结束事件：4012 兜底只负责退出战斗界面，4013 结算包单独驱动升级/奖励弹窗。
func _on_battle_finished(payload: Dictionary) -> void:
	if payload.is_empty():
		return
	if _battle_finish_handler_running:
		if not bool(payload.get("fallback_result", false)):
			_pending_battle_finish_payload = payload.duplicate(true)
		return
	_battle_finish_handler_running = true
	await _process_battle_finished(payload)
	while not _pending_battle_finish_payload.is_empty():
		var pending_payload: Dictionary = _pending_battle_finish_payload
		_pending_battle_finish_payload = {}
		await _process_battle_finished(pending_payload)
	_battle_finish_handler_running = false


## 4012 兜底包只退出战斗弹窗；4013 权威结算包才展示升级与奖励弹窗。
func _process_battle_finished(payload: Dictionary) -> void:
	if bool(payload.get("fallback_result", false)):
		await _dismiss_battle_modal(payload)
		return
	if _is_battle_modal_active():
		await _dismiss_battle_modal(payload)
	else:
		_sync_world_player_battle_pose()
	await _present_battle_settlement(payload)
	_schedule_next_auto_wild_encounter()


## 等待战斗演出结束并卸载战斗弹窗，恢复世界 HUD。
func _dismiss_battle_modal(payload: Dictionary) -> void:
	_set_runtime_menu_locked(true)
	_sync_world_player_battle_pose()
	if _battle_scene != null and _battle_scene.has_method("wait_for_presentation_complete"):
		await _battle_scene.wait_for_presentation_complete(payload)
	_unmount_battle_scene()


## 展示战斗胜利后的升级弹窗、宠物升级弹窗与奖励弹窗，并刷新权威数据。
func _present_battle_settlement(payload: Dictionary) -> void:
	var flow_id: int = _popup_flow_generation
	var auto_settlement_mode: bool = _auto_wild_encounter_enabled
	if _should_show_player_level_up_popup(payload):
		if auto_settlement_mode:
			_append_log("挂机中：本轮跳过玩家升级确认弹窗，5 秒后自动继续遇怪。")
		else:
			var player_level: int = _resolve_player_level_for_popup(payload)
			var bonus_variant: Variant = payload.get("level_up_bonus", {})
			var bonus: Dictionary = bonus_variant if bonus_variant is Dictionary else {}
			await _show_level_up_popup_and_wait(player_level, bonus)
			if flow_id != _popup_flow_generation:
				return
	var pet_rewards_variant: Variant = payload.get("pet_rewards", [])
	var pet_rewards: Array = pet_rewards_variant if pet_rewards_variant is Array else []
	if not auto_settlement_mode:
		await _show_pet_level_up_popups_and_wait(pet_rewards)
		if flow_id != _popup_flow_generation:
			return
	var popup_rewards: Array = _collect_battle_popup_rewards(payload)
	if _should_show_battle_reward_popup(payload, popup_rewards, pet_rewards):
		if flow_id != _popup_flow_generation:
			return
		if auto_settlement_mode:
			_show_reward_popup("", popup_rewards, pet_rewards, _collect_battle_skill_progress(payload))
		else:
			await _show_reward_popup_and_wait("", popup_rewards, pet_rewards, _collect_battle_skill_progress(payload))
	var reward_gold: int = int(payload.get("reward_gold", 0))
	var reward_player_exp: int = int(payload.get("reward_player_exp", 0))
	var drop_texts_variant: Variant = payload.get("drop_texts", [])
	var drop_texts: Array = drop_texts_variant if drop_texts_variant is Array else []
	if reward_gold > 0 or reward_player_exp > 0:
		_append_log("战斗结束，获得 %d 金币 / %d 角色经验。" % [reward_gold, reward_player_exp])
	else:
		_append_log("战斗结束，返回世界场景。")
	for drop_text_variant: Variant in drop_texts:
		var drop_text: String = str(drop_text_variant)
		if not drop_text.is_empty():
			_append_log(drop_text)
	_log_pet_level_up_rewards(pet_rewards)
	App.request_quest_list()
	if bool(payload.get("win", false)):
		App.refresh_player_status()
		App.request_pet_list()
	_refresh_view()

## 初始化战斗弹窗层：全屏半透明遮罩 + 居中战斗面板，世界场景保持可见。
func _ensure_battle_modal_layer() -> void:
	if _battle_modal_layer != null:
		return
	_battle_modal_layer = CanvasLayer.new()
	_battle_modal_layer.name = "BattleModalLayer"
	_battle_modal_layer.layer = BATTLE_MODAL_LAYER
	_battle_modal_layer.visible = false
	add_child(_battle_modal_layer)

	_battle_modal_dim = ColorRect.new()
	_battle_modal_dim.name = "DimOverlay"
	_battle_modal_dim.mouse_filter = Control.MOUSE_FILTER_STOP
	_battle_modal_dim.color = Color(0.0, 0.0, 0.0, 0.55)
	_battle_modal_layer.add_child(_battle_modal_dim)

	_battle_modal_host = CenterContainer.new()
	_battle_modal_host.name = "BattleModalHost"
	_battle_modal_host.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_battle_modal_layer.add_child(_battle_modal_host)

	if not get_viewport().size_changed.is_connected(_sync_battle_modal_layout):
		get_viewport().size_changed.connect(_sync_battle_modal_layout)
	_sync_battle_modal_layout()

## 让战斗弹窗遮罩与居中容器始终铺满当前视口。
func _sync_battle_modal_layout() -> void:
	if _battle_modal_layer == null:
		return
	var viewport_size: Vector2 = get_viewport().get_visible_rect().size
	if _battle_modal_dim != null:
		_battle_modal_dim.position = Vector2.ZERO
		_battle_modal_dim.size = viewport_size
	if _battle_modal_host != null:
		_battle_modal_host.position = Vector2.ZERO
		_battle_modal_host.size = viewport_size

## 在世界场景上方挂载战斗弹窗，不再隐藏世界或截取地图背景。
func _mount_battle_popup(payload: Dictionary) -> void:
	_ensure_battle_modal_layer()
	if _battle_modal_layer == null or _battle_modal_host == null:
		return
	if _battle_scene == null:
		_battle_scene = BATTLE_SCENE.instantiate() as Control
		if _battle_scene == null:
			return
		_battle_scene.custom_minimum_size = BATTLE_MODAL_SIZE
		_battle_scene.set_deferred("size", BATTLE_MODAL_SIZE)
		_battle_modal_host.add_child(_battle_scene)
		if _battle_scene.has_method("bind_battle_controller"):
			_battle_scene.call("bind_battle_controller", battle_controller)
	_battle_modal_layer.visible = true
	_sync_battle_modal_layout()
	if _battle_scene.has_method("_on_battle_started"):
		_battle_scene.call("_on_battle_started", payload)
	_sync_world_player_battle_pose()

func _unmount_battle_scene() -> void:
	if _battle_scene != null:
		_battle_scene.queue_free()
		_battle_scene = null
	if _battle_modal_layer != null:
		_battle_modal_layer.visible = false
	_sync_world_player_battle_pose()
	_refresh_world_pet_follower_after_battle()
	_set_runtime_menu_locked(false)
	hud_root.set_player_status_visible(true)


## 战斗弹窗卸载后恢复世界场景出战宠物展示。
func _refresh_world_pet_follower_after_battle() -> void:
	if _world_controller != null and _world_controller.has_method("refresh_pet_follower_after_battle"):
		_world_controller.call("refresh_pet_follower_after_battle")

## 同步世界玩家战斗待机动画，与战斗弹窗可见态保持一致。
func _sync_world_player_battle_pose() -> void:
	if _world_controller == null:
		return
	var battle_pose_active: bool = _is_battle_modal_active()
	if _world_controller.has_method("set_local_player_battle_pose_active"):
		_world_controller.call("set_local_player_battle_pose_active", battle_pose_active)

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

# 按当前全局状态刷新左上角玩家状态 HUD。
func _refresh_view() -> void:
	hud_root.refresh_player_status()
	_refresh_auto_wild_encounter_button_state()

## 世界快照更新后刷新 HUD，并为剧情解锁等同地图新增 NPC 补载菜单缓存。
func _on_world_snapshot_changed() -> void:
	_refresh_view()
	_schedule_missing_scene_npc_menu_preload()

## 点击左上角头像时打开人物状态面板。
func _on_hud_avatar_pressed() -> void:
	if _is_battle_modal_active() or _is_settlement_input_blocked():
		return
	_toggle_root_runtime_panel(_player_panel, "player_panel")

## 点击宠物 HUD 时打开宠物状态面板，打开前会先拉取服务端最新宠物列表。
func _on_hud_pet_pressed() -> void:
	if _is_battle_modal_active() or _is_settlement_input_blocked():
		return
	await _open_pet_panel_prepared()

## 点击右下角背包常驻按钮时打开背包，沿用主菜单中的背包面板与服务端刷新逻辑。
func _on_hud_bag_pressed() -> void:
	if _is_battle_modal_active() or _is_settlement_input_blocked():
		return
	await _open_bag_panel_prepared()

## 点击右下角任务常驻按钮时打开任务面板，打开前会先拉取服务端最新任务列表。
func _on_hud_task_pressed() -> void:
	if _is_battle_modal_active() or _is_settlement_input_blocked():
		return
	await _open_task_panel_prepared()

## 点击挂机按钮时切换暗雷挂机状态；仅当前地图支持暗雷时允许开启。
func _on_hud_auto_encounter_pressed() -> void:
	if _is_battle_modal_active() or _is_settlement_input_blocked():
		return
	if _auto_wild_encounter_enabled:
		_set_auto_wild_encounter_enabled(false)
		return
	if not _is_auto_wild_encounter_available():
		_append_log("当前地图不支持暗雷挂机。")
		_refresh_auto_wild_encounter_button_state()
		return
	_open_auto_encounter_target_menu()

## 点击设置菜单中的“返回登录页”时，沿用现有切场景清理链路。
func _on_hud_return_to_login_pressed() -> void:
	call_deferred("_return_to_login_scene")

## 点击设置菜单中的“退出游戏”时，先弹出确认框，避免误触直接关闭客户端。
func _on_hud_quit_game_pressed() -> void:
	_ensure_quit_confirm_popup()
	if _quit_confirm_popup == null:
		return
	_quit_confirm_popup.show_prompt("退出游戏", "确定要退出当前游戏吗？", {
		"confirm_label": "退出",
		"cancel_label": "取消",
	})


## 按当前场景暗雷配置刷新挂机按钮；离开可挂机地图时自动关闭挂机，避免玩家误以为仍在轮询。
func _refresh_auto_wild_encounter_button_state() -> void:
	if hud_root == null:
		return
	var available: bool = _is_auto_wild_encounter_available()
	if _auto_wild_encounter_enabled and not available:
		_set_auto_wild_encounter_enabled(false, false)
		_append_log("已离开暗雷地图，自动关闭挂机。")
		available = false
	hud_root.set_auto_encounter_button_state(available, _auto_wild_encounter_enabled)


## 判断当前地图是否允许开启暗雷挂机；只依赖服务端权威暗雷配置，不在客户端硬编码地图白名单。
func _is_auto_wild_encounter_available() -> bool:
	var config: Dictionary = GameState.wild_encounter_config
	if not bool(config.get("enabled", false)):
		return false
	var config_scene_id: int = int(config.get("scene_id", 0))
	var current_scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	if config_scene_id <= 0 or current_scene_id <= 0:
		return false
	return config_scene_id == current_scene_id


## 切换挂机状态，并同步按钮文案与轮询倒计时。
func _set_auto_wild_encounter_enabled(enabled: bool, append_log: bool = true) -> void:
	if enabled and not _is_auto_wild_encounter_available():
		enabled = false
	if enabled and _auto_wild_encounter_target_name.is_empty():
		enabled = false
	if _auto_wild_encounter_enabled == enabled:
		_refresh_auto_wild_encounter_button_state()
		return
	_auto_wild_encounter_enabled = enabled
	_cancel_auto_wild_encounter_schedule()
	if _auto_wild_encounter_enabled:
		if append_log:
			_append_log("已开启暗雷挂机，目标：%s。" % _auto_wild_encounter_target_name)
		_schedule_next_auto_wild_encounter()
	else:
		_clear_auto_wild_encounter_target()
		if append_log:
			_append_log("已关闭暗雷挂机。")
	_refresh_auto_wild_encounter_button_state()


## 取消当前倒计时；通过代次失效旧的 await 定时器，避免重复自动开战。
func _cancel_auto_wild_encounter_schedule() -> void:
	_auto_wild_encounter_schedule_generation += 1


## 根据当前状态安排下一轮挂机暗雷；战斗中或地图不可用时不排程。
func _schedule_next_auto_wild_encounter() -> void:
	if not _auto_wild_encounter_enabled:
		return
	if not _is_auto_wild_encounter_available():
		return
	if GameState.is_in_battle or _is_battle_modal_active():
		return
	_auto_wild_encounter_schedule_generation += 1
	var generation: int = _auto_wild_encounter_schedule_generation
	call_deferred("_run_auto_wild_encounter_countdown", generation)


## 挂机倒计时协程；5 秒到期后若状态仍有效，则关闭所有弹窗并请求新的暗雷战斗。
func _run_auto_wild_encounter_countdown(generation: int) -> void:
	await get_tree().create_timer(AUTO_WILD_ENCOUNTER_INTERVAL_SEC).timeout
	if generation != _auto_wild_encounter_schedule_generation:
		return
	if not _auto_wild_encounter_enabled:
		return
	if not _is_auto_wild_encounter_available():
		_set_auto_wild_encounter_enabled(false, false)
		return
	if GameState.is_in_battle or _is_battle_modal_active():
		return
	_trigger_auto_wild_encounter()


## 真正执行一轮挂机暗雷请求；进战前统一关闭运行时面板，满足“弹窗未手动关闭也能继续挂机”的需求。
func _trigger_auto_wild_encounter() -> void:
	if _auto_wild_encounter_target_name.is_empty():
		_set_auto_wild_encounter_enabled(false, false)
		_append_log("挂机目标已失效，请重新选择。")
		return
	if _world_controller == null:
		return
	if not _world_controller.has_method("request_auto_wild_encounter_battle"):
		return
	_close_all_blocking_popups_for_battle()
	var can_request: bool = true
	if _world_controller.has_method("can_request_auto_wild_encounter_battle"):
		can_request = bool(_world_controller.call("can_request_auto_wild_encounter_battle"))
	if not can_request:
		_schedule_next_auto_wild_encounter()
		return
	var accepted: bool = bool(_world_controller.call("request_auto_wild_encounter_battle"))
	if not accepted:
		_schedule_next_auto_wild_encounter()


## 懒创建退出确认弹窗，并把确认动作绑定到真正的退出链路。
func _ensure_quit_confirm_popup() -> void:
	if _quit_confirm_popup != null:
		return
	_quit_confirm_popup = ConfirmPromptPopupScene.instantiate() as ConfirmPromptPopup
	if _quit_confirm_popup == null:
		return
	_quit_confirm_popup.name = "QuitConfirmPopup"
	add_child(_quit_confirm_popup)
	if not _quit_confirm_popup.confirmed.is_connected(_confirm_quit_game):
		_quit_confirm_popup.confirmed.connect(_confirm_quit_game)


## 玩家确认退出后，桌面端直接关闭客户端；Web 端尝试关闭当前页面。
func _confirm_quit_game() -> void:
	NetClient.disconnect_from_server()
	GameState.reset_session_state()
	if OS.has_feature("web"):
		_append_log("正在尝试关闭当前页面。")
		JavaScriptBridge.eval("window.open('', '_self'); window.close();", true)
		return
	get_tree().quit()

# 向底部 HUD 日志区域追加一条文本。
func _append_log(message: String) -> void:
	hud_root.append_log(message)

func _create_runtime_ui() -> void:
	_create_main_menu()
	_create_player_panel()
	_create_pet_panel()
	_create_bag_panel()
	_create_task_panel()
	_create_npc_menu()
	_create_npc_list_menu()
	_create_pvp_target_menu()
	_create_auto_encounter_target_menu()
	_create_npc_dialogue_panel()
	_create_npc_shop_panel()
	_create_pvp_invite_dialog()
	_create_reward_popup()
	_create_flow_confirm_popup()
	_create_cinematic_player()
	_create_npc_request_loading()


## 懒创建流程确认弹窗，避免与已经绑定退出动作的确认弹窗共享实例。
func _create_flow_confirm_popup() -> void:
	if _flow_confirm_popup != null:
		return
	_flow_confirm_popup = ConfirmPromptPopupScene.instantiate() as ConfirmPromptPopup
	if _flow_confirm_popup == null:
		return
	_flow_confirm_popup.name = "FlowConfirmPopup"
	add_child(_flow_confirm_popup)


## 展示流程确认提示并等待玩家关闭；正文直接渲染服务端提供的 BBCode。
## title_text 是弹窗标题。
## content_bbcode 是支持 RichTextLabel BBCode 的提示正文。
## content_font_size 是移动端正文显示字号。
func _show_flow_confirm_prompt_and_wait(title_text: String, content_bbcode: String, content_font_size: int = 20) -> void:
	_create_flow_confirm_popup()
	if _flow_confirm_popup == null:
		return
	_flow_confirm_popup.show_prompt(title_text, content_bbcode, {
		"confirm_label": "确定",
		"content_font_size": content_font_size,
	})
	if not await _wait_for_next_process_frame():
		return
	if _flow_confirm_popup.visible:
		await _flow_confirm_popup.popup_closed


## 展示玩家升级结果；等级缺失时读取权威人物快照，仍无法解析则不展示无效内容。
func _show_level_up_popup_and_wait(level: int, bonus: Dictionary) -> void:
	var resolved_level: int = level
	if resolved_level <= 0:
		resolved_level = int(GameState.player_snapshot.get("level", 0))
	if resolved_level <= 0:
		return
	var content_lines: PackedStringArray = PackedStringArray([
		"恭喜你升到了%d级" % resolved_level,
		"最大生命值增加：%d" % int(bonus.get("hp_max", 0)),
		"攻击力增加：%d" % int(bonus.get("atk", 0)),
		"法力增加：%d" % int(bonus.get("mana", 0)),
		"速度增加：%d" % int(bonus.get("spd", 0)),
	])
	await _show_flow_confirm_prompt_and_wait("升级", "\n".join(content_lines), 20)


## 逐只展示战斗结算里升级过的宠物摘要；玩家点按关闭后再进入下一项。
func _show_pet_level_up_popups_and_wait(pet_rewards: Array) -> void:
	for pet_reward_variant: Variant in pet_rewards:
		if pet_reward_variant is not Dictionary:
			continue
		var pet_reward: Dictionary = pet_reward_variant as Dictionary
		var level_up_count: int = int(pet_reward.get("level_up_count", 0))
		if level_up_count <= 0:
			continue
		var pet_uid: int = int(pet_reward.get("pet_uid", 0))
		var pet_id: int = int(pet_reward.get("pet_id", 0))
		var level: int = int(pet_reward.get("level", 0))
		var attr_points_gained: int = int(pet_reward.get("attr_points_gained", 0))
		var free_attr_points: int = int(pet_reward.get("free_attr_points", 0))
		if level <= 0:
			continue
		var pet_name: String = _resolve_pet_display_name(pet_uid, pet_id)
		var content_lines: PackedStringArray = PackedStringArray([
			"升到了 %d 级" % level,
			"获得自由属性点：%d" % attr_points_gained,
			"当前可用自由点：%d" % free_attr_points,
		])
		await _show_flow_confirm_prompt_and_wait(pet_name, "\n".join(content_lines), 36)


## 根据本地宠物列表解析展示名；缺失时回退到 pet_id 或 pet_uid。
func _resolve_pet_display_name(pet_uid: int, pet_id: int = 0) -> String:
	if pet_uid > 0:
		for pet_variant: Variant in GameState.pets:
			if pet_variant is not Dictionary:
				continue
			var pet: Dictionary = pet_variant as Dictionary
			if int(pet.get("pet_uid", 0)) != pet_uid:
				continue
			var resolved_pet_id: int = int(pet.get("pet_id", 0))
			if resolved_pet_id > 0:
				return "宠物 %d" % resolved_pet_id
	if pet_id > 0:
		return "宠物 %d" % pet_id
	if pet_uid > 0:
		return "宠物 #%d" % pet_uid
	return "你的宠物"


## 把战斗结算中的宠物升级摘要写入运行日志，便于玩家确认获得自由点。
func _log_pet_level_up_rewards(pet_rewards: Array) -> void:
	for pet_reward_variant: Variant in pet_rewards:
		if pet_reward_variant is not Dictionary:
			continue
		var pet_reward: Dictionary = pet_reward_variant as Dictionary
		var level_up_count: int = int(pet_reward.get("level_up_count", 0))
		if level_up_count <= 0:
			continue
		var pet_uid: int = int(pet_reward.get("pet_uid", 0))
		var attr_points_gained: int = int(pet_reward.get("attr_points_gained", 0))
		_append_log("宠物 #%d 升了 %d 级，获得 %d 自由属性点。" % [pet_uid, level_up_count, attr_points_gained])


func _create_reward_popup() -> void:
	if _reward_popup != null:
		return
	_reward_popup = RewardPopupScene.instantiate() as RewardPopup
	if _reward_popup == null:
		return
	_reward_popup.name = "RewardPopup"
	add_child(_reward_popup)


func _show_reward_popup(title_text: String, rewards: Array, pet_rewards: Array = [], skill_progress_rewards: Array = []) -> void:
	_create_reward_popup()
	if _reward_popup != null:
		_reward_popup.show_rewards(title_text, rewards, pet_rewards, "", skill_progress_rewards)


## 展示奖励弹窗并等待玩家关闭，避免后续刷新请求抢跑导致弹窗一闪而过。
func _show_reward_popup_and_wait(title_text: String, rewards: Array, pet_rewards: Array = [], skill_progress_rewards: Array = []) -> void:
	_create_reward_popup()
	if _reward_popup == null:
		return
	_reward_popup.show_rewards(title_text, rewards, pet_rewards, "", skill_progress_rewards)
	if _reward_popup.visible:
		await _reward_popup.popup_closed


## 从战斗结算包整理弹窗奖励；优先使用服务端 rewards 列表，缺失时回退到顶层数值字段。
func _collect_battle_popup_rewards(payload: Dictionary) -> Array:
	var rewards_variant: Variant = payload.get("rewards", [])
	var normalized: Array = []
	if rewards_variant is Array:
		for reward_variant in rewards_variant:
			if reward_variant is Dictionary:
				normalized.append(reward_variant)
	if not normalized.is_empty():
		return normalized
	var player_exp: int = int(payload.get("reward_player_exp", 0))
	if player_exp > 0:
		normalized.append({"type": "exp", "value": player_exp})
	var reward_gold: int = int(payload.get("reward_gold", 0))
	if reward_gold > 0:
		normalized.append({"type": "gold", "value": reward_gold})
	return normalized


## 从战斗结算包提取武器技能学习进度，供奖励弹窗展示。
func _collect_battle_skill_progress(payload: Dictionary) -> Array:
	var progress_variant: Variant = payload.get("skill_progress", [])
	if progress_variant is not Array:
		return []
	var normalized: Array = []
	for item_variant: Variant in progress_variant as Array:
		if item_variant is Dictionary:
			normalized.append(item_variant)
	return normalized


## 判断战斗胜利后是否应弹出奖励面板；需与 RewardPopup 的可展示规则保持一致。
func _should_show_battle_reward_popup(payload: Dictionary, popup_rewards: Array, pet_rewards: Array) -> bool:
	if not bool(payload.get("win", false)):
		return false
	if not popup_rewards.is_empty():
		return true
	if not _collect_battle_skill_progress(payload).is_empty():
		return true
	if int(payload.get("reward_player_exp", 0)) > 0 or int(payload.get("reward_gold", 0)) > 0:
		return true
	for pet_reward_variant: Variant in pet_rewards:
		if pet_reward_variant is not Dictionary:
			continue
		var pet_reward: Dictionary = pet_reward_variant as Dictionary
		if int(pet_reward.get("exp_gained", pet_reward.get("exp", 0))) > 0:
			return true
	return false


func _on_quest_settlement_popup_requested(payload: Dictionary) -> void:
	var flow_id: int = _popup_flow_generation
	var completion_prompt_text: String = _resolve_quest_completion_prompt_text(payload)
	if not completion_prompt_text.is_empty():
		await _show_quest_completion_prompt_and_wait(completion_prompt_text)
		if flow_id != _popup_flow_generation:
			return
	if _should_show_player_level_up_popup(payload):
		var player_level: int = _resolve_player_level_for_popup(payload)
		var bonus_variant: Variant = payload.get("level_up_bonus", {})
		var bonus: Dictionary = bonus_variant if bonus_variant is Dictionary else {}
		await _show_level_up_popup_and_wait(player_level, bonus)
		if flow_id != _popup_flow_generation:
			return
	var rewards_variant: Variant = payload.get("rewards", [])
	var rewards: Array = rewards_variant if rewards_variant is Array else []
	if not rewards.is_empty():
		if flow_id != _popup_flow_generation:
			return
		_show_reward_popup("", rewards, [])
	if _should_show_player_level_up_popup(payload):
		App.refresh_player_status()


## 从任务提交响应中解析服务端配置的完成提示文案；优先读取顶层字段，兼容任务摘要内字段。
func _resolve_quest_completion_prompt_text(payload: Dictionary) -> String:
	var prompt_text: String = str(payload.get("completion_prompt_text", "")).strip_edges()
	if not prompt_text.is_empty():
		return prompt_text
	var quest_variant: Variant = payload.get("quest", {})
	if quest_variant is not Dictionary:
		return ""
	var quest: Dictionary = quest_variant as Dictionary
	return str(quest.get("completion_prompt_text", "")).strip_edges()


## 展示任务完成提示；正文支持服务端下发的 RichTextLabel BBCode，关闭后才继续奖励弹窗。
func _show_quest_completion_prompt_and_wait(prompt_text: String) -> void:
	await _show_flow_confirm_prompt_and_wait("任务完成", prompt_text, 20)


## 判断结算包是否应展示玩家升级弹窗；优先看 level_up_count，其次看是否发放了升级属性点。
func _should_show_player_level_up_popup(payload: Dictionary) -> bool:
	if int(payload.get("level_up_count", 0)) > 0:
		return true
	if int(payload.get("attr_points_gained", 0)) > 0:
		return true
	return false


## 从结算包与本地权威快照解析升级后的玩家等级，供升级弹窗展示。
func _resolve_player_level_for_popup(payload: Dictionary) -> int:
	var player_level: int = int(payload.get("player_level", 0))
	if player_level > 0:
		return player_level
	var snapshot_level: int = int(GameState.player_snapshot.get("level", 0))
	if snapshot_level > 0:
		return snapshot_level
	return 0


func _on_reward_popup_requested(title_text: String, rewards: Array, pet_rewards: Array = []) -> void:
	_show_reward_popup(title_text, rewards, pet_rewards)


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

## 创建宠物状态面板，并接入运行时根面板关闭信号。
func _create_pet_panel() -> void:
	_pet_panel = PET_PANEL_SCENE.instantiate() as CanvasLayer
	if _pet_panel == null:
		return
	add_child(_pet_panel)
	if _pet_panel.has_signal("menu_closed"):
		_pet_panel.connect("menu_closed", Callable(self, "_on_runtime_menu_closed"))

func _create_bag_panel() -> void:
	_bag_panel = BAG_PANEL_SCENE.instantiate() as CanvasLayer
	if _bag_panel == null:
		return
	add_child(_bag_panel)
	if _bag_panel.has_signal("menu_closed"):
		_bag_panel.connect("menu_closed", Callable(self, "_on_runtime_menu_closed"))
	if bag_controller.has_signal("drop_item_responded") and _bag_panel.has_method("on_drop_item_responded"):
		var drop_responded_handler: Callable = Callable(_bag_panel, "on_drop_item_responded")
		if not bag_controller.drop_item_responded.is_connected(drop_responded_handler):
			bag_controller.drop_item_responded.connect(drop_responded_handler)
	if bag_controller.has_signal("container_snapshot_applied") and _bag_panel.has_method("on_bag_snapshot_applied"):
		var snapshot_applied_handler: Callable = Callable(_bag_panel, "on_bag_snapshot_applied")
		if not bag_controller.container_snapshot_applied.is_connected(snapshot_applied_handler):
			bag_controller.container_snapshot_applied.connect(snapshot_applied_handler)

## 创建任务面板，并接入运行时根面板关闭信号。
func _create_task_panel() -> void:
	_task_panel = TASK_PANEL_SCENE.instantiate() as CanvasLayer
	if _task_panel == null:
		return
	add_child(_task_panel)
	if _task_panel.has_signal("menu_closed"):
		_task_panel.connect("menu_closed", Callable(self, "_on_runtime_menu_closed"))
	if _task_panel.has_signal("quest_prompt_requested"):
		_task_panel.connect("quest_prompt_requested", Callable(self, "_on_quest_prompt_requested"))


## 使用通用确认弹窗展示任务面板发起的前置或目标引导提示。
## message 是任务面板根据服务端任务快照生成的 BBCode 提示正文。
func _on_quest_prompt_requested(message: String) -> void:
	await _show_flow_confirm_prompt_and_wait("任务提示", message, 20)
	# 模态基类会在关闭后延迟解锁；下一帧恢复仍处于打开状态的任务面板锁定。
	if not await _wait_for_next_process_frame():
		return
	if _task_panel != null and _task_panel.visible:
		_set_runtime_menu_locked(true)

func _create_npc_menu() -> void:
	_npc_menu = OPTION_LIST_PANEL_SCENE.instantiate() as OptionListPanel
	if _npc_menu == null:
		return
	_npc_menu.name = "NpcMenu"
	add_child(_npc_menu)
	if not _npc_menu.option_selected.is_connected(_on_npc_menu_option_selected):
		_npc_menu.option_selected.connect(_on_npc_menu_option_selected)
	if not _npc_menu.menu_closed.is_connected(_on_runtime_menu_closed):
		_npc_menu.menu_closed.connect(_on_runtime_menu_closed)


func _create_npc_list_menu() -> void:
	_npc_list_menu = OPTION_LIST_PANEL_SCENE.instantiate() as OptionListPanel
	if _npc_list_menu == null:
		return
	_npc_list_menu.name = "NpcListMenu"
	add_child(_npc_list_menu)
	if not _npc_list_menu.option_selected.is_connected(_on_npc_selected):
		_npc_list_menu.option_selected.connect(_on_npc_selected)
	if not _npc_list_menu.menu_closed.is_connected(_on_npc_list_closed):
		_npc_list_menu.menu_closed.connect(_on_npc_list_closed)


func _create_pvp_target_menu() -> void:
	_pvp_target_menu = OPTION_LIST_PANEL_SCENE.instantiate() as OptionListPanel
	if _pvp_target_menu == null:
		return
	_pvp_target_menu.name = "PvpTargetMenu"
	add_child(_pvp_target_menu)
	if not _pvp_target_menu.option_selected.is_connected(_on_pvp_target_selected):
		_pvp_target_menu.option_selected.connect(_on_pvp_target_selected)
	if not _pvp_target_menu.menu_closed.is_connected(_on_runtime_menu_closed):
		_pvp_target_menu.menu_closed.connect(_on_runtime_menu_closed)

## 创建挂机目标选择面板，复用通用头像列表面板样式。
func _create_auto_encounter_target_menu() -> void:
	_auto_encounter_target_menu = OPTION_LIST_PANEL_SCENE.instantiate() as OptionListPanel
	if _auto_encounter_target_menu == null:
		return
	_auto_encounter_target_menu.name = "AutoEncounterTargetMenu"
	add_child(_auto_encounter_target_menu)
	if not _auto_encounter_target_menu.option_selected.is_connected(_on_auto_encounter_target_selected):
		_auto_encounter_target_menu.option_selected.connect(_on_auto_encounter_target_selected)
	if not _auto_encounter_target_menu.menu_closed.is_connected(_on_runtime_menu_closed):
		_auto_encounter_target_menu.menu_closed.connect(_on_runtime_menu_closed)

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

## 创建服务端结构化剧情专用的对话面板，并绑定继续/分支按钮回调。
func _create_npc_dialogue_panel() -> void:
	if _npc_dialogue_panel != null:
		return
	_npc_dialogue_panel = NPC_DIALOGUE_PANEL_SCENE.instantiate() as NPCDialoguePanel
	if _npc_dialogue_panel == null:
		return
	add_child(_npc_dialogue_panel)
	if _npc_dialogue_panel.has_signal("continue_requested"):
		_npc_dialogue_panel.connect("continue_requested", Callable(self, "_on_dialogue_continue_requested"))
	if _npc_dialogue_panel.has_signal("local_continue_requested"):
		_npc_dialogue_panel.connect("local_continue_requested", Callable(self, "_on_local_dialogue_continue_requested"))
	if _npc_dialogue_panel.has_signal("option_selected"):
		_npc_dialogue_panel.connect("option_selected", Callable(self, "_on_dialogue_option_selected"))
	if _npc_dialogue_panel.has_signal("panel_closed"):
		_npc_dialogue_panel.connect("panel_closed", Callable(self, "_on_runtime_menu_closed"))

## 创建 NPC 商店面板，并绑定购买/关闭回调。
func _create_npc_shop_panel() -> void:
	if _npc_shop_panel != null:
		return
	_npc_shop_panel = NPC_SHOP_PANEL_SCENE.instantiate() as NPCShopPanel
	if _npc_shop_panel == null:
		return
	add_child(_npc_shop_panel)
	if _npc_shop_panel.has_signal("buy_requested"):
		_npc_shop_panel.connect("buy_requested", Callable(self, "_on_shop_buy_requested"))
	if _npc_shop_panel.has_signal("panel_closed"):
		_npc_shop_panel.connect("panel_closed", Callable(self, "_on_runtime_menu_closed"))

## 创建客户端内置剧情动画播放器，供 action 节点按照 animation_key 拉起本地演出。
func _create_cinematic_player() -> void:
	if _cinematic_player != null:
		return
	_cinematic_player = CinematicPlayer.new()
	_cinematic_player.name = "CinematicPlayer"
	add_child(_cinematic_player)
	_cinematic_player.bind_world_controller(_world_controller)
	_cinematic_player.bind_transition_overlay(transition_overlay, TRANSITION_DURATION)
	if _cinematic_player.has_signal("cinematic_finished"):
		_cinematic_player.connect("cinematic_finished", Callable(self, "_on_cinematic_finished"))
	if _cinematic_player.has_signal("local_dialogue_requested"):
		_cinematic_player.connect("local_dialogue_requested", Callable(self, "_on_local_cinematic_dialogue_requested"))

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
	var npc_name: String = str(npc_data.get("npc_name", "未知 NPC"))
	_append_log("通过列表选择 NPC: %s (%d)" % [npc_name, entity_id])
	if not _open_cached_scene_npc_menu(entity_id, npc_name):
		_opening_npc_menu_from_list = false
		_set_runtime_menu_locked(false)

func _on_pvp_target_selected(target_data: Dictionary) -> void:
	var target_player_id: int = int(target_data.get("target_player_id", target_data.get("entity_id", 0)))
	if target_player_id <= 0:
		_append_log("PVP 目标数据无效，无法发起挑战。")
		return
	if _pvp_target_menu != null and _pvp_target_menu.has_method("close_menu"):
		_pvp_target_menu.call("close_menu")
	_append_log("发起 PVP 挑战: %s (%d)" % [str(target_data.get("npc_name", target_data.get("name", "未知玩家"))), target_player_id])
	App.request_pvp_challenge(target_player_id)

## 玩家确认挂机目标后，记录目标并正式开启挂机。
func _on_auto_encounter_target_selected(target_data: Dictionary) -> void:
	var target_entity_id: int = int(target_data.get("entity_id", 0))
	var target_name: String = str(target_data.get("npc_name", target_data.get("name", ""))).strip_edges()
	if target_entity_id <= 0 or target_name.is_empty():
		_append_log("挂机目标无效，无法开启挂机。")
		return
	if _auto_encounter_target_menu != null and _auto_encounter_target_menu.has_method("close_menu"):
		_auto_encounter_target_menu.call("close_menu")
	_auto_wild_encounter_target_entity_id = target_entity_id
	_auto_wild_encounter_target_name = target_name
	_append_log("已选择挂机目标：%s。" % target_name)
	_set_auto_wild_encounter_enabled(true)

func _on_main_menu_item_selected(item: Dictionary) -> void:
	if _is_battle_modal_active():
		return
	var label: String = str(item.get("label", ""))
	if label == "物品行囊":
		if _main_menu != null and _main_menu.has_method("close_menu"):
			_main_menu.call("close_menu")
		call_deferred("_open_bag_panel_prepared")
		return
	if label == "个人状态":
		if _main_menu != null and _main_menu.has_method("close_menu"):
			_main_menu.call("close_menu")
		call_deferred("_open_player_panel_prepared")
		return
	if label == "宠物指令":
		if _main_menu != null and _main_menu.has_method("close_menu"):
			_main_menu.call("close_menu")
		call_deferred("_open_pet_panel_prepared")
		return
	if label != "全服竞技场":
		return
	if _main_menu != null and _main_menu.has_method("close_menu"):
		_main_menu.call("close_menu")
	_open_pvp_target_menu()

func _on_npc_menu_option_selected(option: Dictionary) -> void:
	if _is_battle_modal_active():
		return
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
		elif quest_state == "READY_TO_SUBMIT":
			App.submit_quest(quest_id, entity_id)
		_begin_npc_menu_request(entity_id)
		return
	if entry_type == "warehouse":
		_append_log("仓库旧面板已移除，等待新版仓库界面接入。")
		return
	_begin_npc_action_request(entity_id, entry_id)

func _on_pvp_challenge_responded(payload: Dictionary) -> void:
	var accepted: bool = bool(payload.get("accepted", false))
	var reason: String = str(payload.get("reason", ""))
	var target_player_id: int = int(payload.get("target_player_id", 0))
	if accepted:
		_append_log("PVP 挑战已发送，目标玩家: %d。" % target_player_id)
		return
	_append_log("PVP 挑战发送失败: %s" % reason)

func _on_pvp_challenge_received(payload: Dictionary) -> void:
	if _is_battle_modal_active():
		return
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
		if int(entity.get("entity_type", 0)) != NPC_ENTITY_TYPE:
			continue
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
		var entry: Dictionary = {
			"entity_id": int(entity.get("entity_id", entity_id_variant)),
			"target_player_id": target_player_id,
			"npc_name": str(entity.get("name", "附近玩家")),
			"portrait_path": "res://asset/口袋所有形象/imgs/51.png",
		}
		entries.append(entry)
	return entries

## 从服务端权威暗雷配置组装当前地图的挂机目标列表。
func _collect_auto_encounter_target_entries() -> Array[Dictionary]:
	var entries: Array[Dictionary] = []
	var targets_variant: Variant = GameState.wild_encounter_config.get("targets", [])
	if targets_variant is not Array:
		return entries
	var targets: Array = targets_variant as Array
	var seen_monster_ids: Dictionary = {}
	for target_variant: Variant in targets:
		if target_variant is not Dictionary:
			continue
		var target: Dictionary = target_variant
		var monster_id: int = int(target.get("monster_id", 0))
		var monster_name: String = str(target.get("monster_name", "")).strip_edges()
		if monster_id <= 0 or monster_name.is_empty() or seen_monster_ids.has(monster_id):
			continue
		seen_monster_ids[monster_id] = true
		var entry: Dictionary = {
			"entity_id": monster_id,
			"npc_name": monster_name,
			"skin_id": str(target.get("skin_id", "")).strip_edges(),
		}
		entries.append(entry)
	return entries

## 打开挂机目标选择面板；先让玩家选定附近单位，再开启自动暗雷。
func _open_auto_encounter_target_menu() -> void:
	if _auto_encounter_target_menu == null:
		return
	var target_entries: Array[Dictionary] = _collect_auto_encounter_target_entries()
	if target_entries.is_empty():
		App.notice_received.emit("当前地图未配置可挂机怪物。")
		return
	_close_other_root_panels("auto_encounter_target")
	_auto_encounter_target_menu.configure("选择挂机目标", target_entries, {
		"render_mode": OptionListPanel.RENDER_PORTRAIT_TEXT,
	})
	_auto_encounter_target_menu.call("open_menu")
	_set_runtime_menu_locked(true)

## 清空当前挂机目标，避免关闭挂机后沿用旧选择。
func _clear_auto_wild_encounter_target() -> void:
	_auto_wild_encounter_target_entity_id = 0
	_auto_wild_encounter_target_name = ""

func _open_pvp_target_menu() -> void:
	if _is_battle_modal_active():
		_append_log("战斗中无法发起新的 PVP 挑战。")
		return
	if _pvp_target_menu == null:
		return
	var nearby_players: Array[Dictionary] = _collect_nearby_player_entries()
	if nearby_players.is_empty():
		_append_log("附近没有可挑战的玩家。")
		return
	_close_other_root_panels("pvp_list")
	_pvp_target_menu.configure("选择挑战玩家", nearby_players, {
		"render_mode": OptionListPanel.RENDER_PORTRAIT_TEXT,
	})
	_pvp_target_menu.call("open_menu")
	_set_runtime_menu_locked(true)

## 创建 NPC 请求 loading 遮罩，统一覆盖交互、菜单执行与剧情推进等待态。
func _create_npc_request_loading() -> void:
	if _npc_request_loading != null:
		return
	_npc_request_loading = RUNTIME_PROGRESS_OVERLAY_SCENE.instantiate() as RuntimeProgressOverlay
	if _npc_request_loading == null:
		return
	_npc_request_loading.name = "NpcRequestLoading"
	add_child(_npc_request_loading)

## 展示 NPC 相关请求的 loading 遮罩。
func _show_npc_request_loading() -> void:
	if _npc_request_loading != null:
		_npc_request_loading.show_waiting()


## 立即展示全屏 loading（无 1 秒延迟），用于背包打开等需要即时反馈的场景。
func _show_npc_request_loading_immediate() -> void:
	if _npc_request_loading != null:
		_npc_request_loading.show_loading()


## 隐藏 NPC 相关请求的 loading 遮罩。
func _hide_npc_request_loading() -> void:
	if _npc_request_loading != null:
		_npc_request_loading.hide_overlay()

## 发起 NPC_MENU_REQ；open_when_ready 决定回包后打开菜单还是仅写入当前地图缓存。
func _begin_npc_menu_request(entity_id: int, open_when_ready: bool = true) -> bool:
	if _pending_npc_menu_seq > 0:
		return false
	_pending_npc_menu_payload.clear()
	var request_seq: int = App.request_npc_menu(entity_id)
	if request_seq <= 0:
		return false
	_pending_npc_menu_seq = request_seq
	_pending_npc_menu_entity_id = entity_id
	_pending_npc_menu_scene_id = int(GameState.scene_snapshot.get("scene_id", 0))
	_pending_npc_menu_should_open = open_when_ready
	if open_when_ready:
		_show_npc_request_loading()
	call_deferred("_wait_npc_menu_request", request_seq)
	return true

## NPC 菜单回包到达后写入当前地图缓存，并按请求用途决定是否打开面板。
func _finish_npc_menu_request(expected_seq: int, succeeded: bool) -> void:
	if expected_seq != _pending_npc_menu_seq:
		return
	var payload: Dictionary = _pending_npc_menu_payload.duplicate(true)
	var request_entity_id: int = _pending_npc_menu_entity_id
	var request_scene_id: int = _pending_npc_menu_scene_id
	var should_open: bool = _pending_npc_menu_should_open
	if should_open:
		_hide_npc_request_loading()
	_pending_npc_menu_seq = 0
	_pending_npc_menu_entity_id = 0
	_pending_npc_menu_scene_id = 0
	_pending_npc_menu_should_open = false
	_pending_npc_menu_payload.clear()
	if not succeeded or payload.is_empty():
		_mark_scene_npc_menu_attempted(request_scene_id, request_entity_id)
		return
	_cache_scene_npc_menu(payload)
	if should_open:
		_open_npc_menu_from_payload(payload)

## 进入地图后异步批量请求全部 NPC 动态菜单；该请求不参与转场等待，也不锁定玩家输入。
func _preload_scene_npc_menus(scene_id: int, reset_cache: bool = true) -> void:
	if scene_id <= 0:
		return
	if reset_cache:
		_scene_npc_menu_cache.clear()
		_scene_npc_menu_cache_scene_id = scene_id
	elif _scene_npc_menu_cache_scene_id != scene_id:
		return
	var npc_entries: Array[Dictionary] = _collect_nearby_npc_entries()
	if npc_entries.is_empty():
		_npc_menu_preload_active = false
		_pending_npc_menu_batch_seq = 0
		_pending_npc_menu_batch_scene_id = 0
		return
	if _npc_menu_preload_active and _pending_npc_menu_batch_scene_id == scene_id:
		return
	var request_seq: int = App.request_npc_menu_batch(scene_id)
	if request_seq <= 0:
		return
	_npc_menu_preload_active = true
	_pending_npc_menu_batch_seq = request_seq
	_pending_npc_menu_batch_scene_id = scene_id
	call_deferred("_wait_npc_menu_batch_request", request_seq, scene_id)

## 在同地图世界快照新增 NPC 后排队补载菜单，例如剧情确认后刚解锁的 NPC。
func _schedule_missing_scene_npc_menu_preload() -> void:
	if _npc_menu_refresh_scheduled or _npc_menu_preload_active:
		return
	var scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	if scene_id <= 0 or scene_id != _scene_npc_menu_cache_scene_id:
		return
	for entry: Dictionary in _collect_nearby_npc_entries():
		var entity_id: int = int(entry.get("entity_id", 0))
		if entity_id > 0 and not _scene_npc_menu_cache.has(entity_id):
			_npc_menu_refresh_scheduled = true
			call_deferred("_preload_missing_scene_npc_menus", scene_id)
			return

## 执行已排队的同地图 NPC 菜单补载，并在开始前重新核对当前场景。
func _preload_missing_scene_npc_menus(scene_id: int) -> void:
	_npc_menu_refresh_scheduled = false
	if _npc_menu_preload_active:
		return
	if int(GameState.scene_snapshot.get("scene_id", 0)) != scene_id:
		return
	_preload_scene_npc_menus(scene_id, false)

## 只处理批量菜单请求的失败完成事件；成功响应由业务路由先写入缓存。
func _wait_npc_menu_batch_request(expected_seq: int, expected_scene_id: int) -> void:
	while expected_seq > 0:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		var succeeded: bool = bool(result[2])
		if request_cmd != CommandIds.NPC_MENU_BATCH_REQ or seq != expected_seq:
			continue
		if _pending_npc_menu_batch_seq != expected_seq or _pending_npc_menu_batch_scene_id != expected_scene_id:
			return
		if not succeeded:
			_pending_npc_menu_batch_seq = 0
			_pending_npc_menu_batch_scene_id = 0
			_npc_menu_preload_active = false
		return

## 取消超时的菜单预加载等待；迟到回包会因请求序号不匹配而被忽略。
func _cancel_pending_npc_menu_request() -> void:
	_mark_scene_npc_menu_attempted(_pending_npc_menu_scene_id, _pending_npc_menu_entity_id)
	_pending_npc_menu_seq = 0
	_pending_npc_menu_entity_id = 0
	_pending_npc_menu_scene_id = 0
	_pending_npc_menu_should_open = false
	_pending_npc_menu_payload.clear()

## 等待指定 NPC 菜单请求完成，并把请求序号传给完成函数校验迟到回包。
func _wait_npc_menu_request(expected_seq: int) -> void:
	while expected_seq > 0:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		var succeeded: bool = bool(result[2])
		if request_cmd != CommandIds.NPC_MENU_REQ or seq != expected_seq:
			continue
		_finish_npc_menu_request(expected_seq, succeeded)
		return

## 缓存服务端返回的 NPC 菜单；拒绝结果也会标记为已加载，避免进图阶段无限重试。
func _cache_scene_npc_menu(payload: Dictionary) -> void:
	var entity_id: int = int(payload.get("entity_id", 0))
	if entity_id <= 0:
		return
	var current_scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	if current_scene_id <= 0 or current_scene_id != _scene_npc_menu_cache_scene_id:
		return
	_scene_npc_menu_cache[entity_id] = payload.duplicate(true)

## 为无回包或空回包的 NPC 保存已尝试占位，防止补载调度持续重复请求。
func _mark_scene_npc_menu_attempted(scene_id: int, entity_id: int) -> void:
	if scene_id <= 0 or entity_id <= 0:
		return
	if scene_id != int(GameState.scene_snapshot.get("scene_id", 0)):
		return
	if scene_id != _scene_npc_menu_cache_scene_id:
		return
	if not _scene_npc_menu_cache.has(entity_id):
		_scene_npc_menu_cache[entity_id] = {}

## 发起 NPC_ACTION_REQ，并在回包到达后再执行后续 UI 逻辑。
func _begin_npc_action_request(entity_id: int, entry_id: String) -> void:
	if _pending_npc_action_seq > 0:
		return
	_pending_npc_action_payload.clear()
	var request_seq: int = App.request_npc_action(entity_id, entry_id)
	if request_seq <= 0:
		return
	_pending_npc_action_seq = request_seq
	_show_npc_request_loading()
	call_deferred("_wait_npc_request", request_seq, CommandIds.NPC_ACTION_REQ, "_finish_npc_action_request")

## 等待指定 NPC 相关请求完成，再回调 finish_method 处理 UI。
func _wait_npc_request(expected_seq: int, expected_cmd: int, finish_method: StringName) -> void:
	while expected_seq > 0:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		var succeeded: bool = bool(result[2])
		if request_cmd != expected_cmd or seq != expected_seq:
			continue
		call(finish_method, succeeded)
		return

## NPC_ACTION 回包到达后关闭 loading，并分发战斗/剧情/提示结果。
func _finish_npc_action_request(succeeded: bool) -> void:
	_hide_npc_request_loading()
	var payload: Dictionary = _pending_npc_action_payload.duplicate(true)
	_pending_npc_action_seq = 0
	_pending_npc_action_payload.clear()
	if not succeeded or payload.is_empty():
		return
	_handle_npc_action_payload(payload)

## 剧情推进回包到达后关闭 loading，并刷新剧情面板。
func _finish_dialogue_request(succeeded: bool) -> void:
	_hide_npc_request_loading()
	var payload: Dictionary = _pending_dialogue_payload.duplicate(true)
	_pending_dialogue_request_seq = 0
	_pending_dialogue_payload.clear()
	if not succeeded or payload.is_empty():
		if _npc_dialogue_panel != null:
			_npc_dialogue_panel.show_waiting_state("剧情同步失败")
		return
	_handle_npc_dialogue_payload(payload)

## 用 NPC_MENU_RESP 菜单数据打开 NPC 菜单面板。
func _open_npc_menu_from_payload(payload: Dictionary) -> void:
	if not bool(payload.get("accepted", false)):
		var failed_reason: String = str(payload.get("reason", "npc menu request failed"))
		_append_log("NPC 菜单拉取失败: %s" % failed_reason)
		return
	_append_log("收到 NPC 菜单数据: %s" % str(payload.get("npc_name", "未知 NPC")))
	if _npc_menu == null:
		return
	var menu_options: Array[Dictionary] = _build_npc_menu_options(payload)
	_close_other_root_panels("npc_menu")
	_npc_menu.configure(str(payload.get("npc_name", "NPC")), menu_options, {
		"render_mode": OptionListPanel.RENDER_NPC_ENTRY,
		"header_portrait": _resolve_npc_header_portrait(int(payload.get("entity_id", 0))),
	})
	_npc_menu.call("open_menu")
	_set_runtime_menu_locked(true)

## 处理 NPC 菜单项执行后的服务端结果。
func _handle_npc_action_payload(payload: Dictionary) -> void:
	if bool(payload.get("accepted", false)) and payload.has("menu_entries"):
		_cache_scene_npc_menu(payload)
	var animation_key: String = str(payload.get("client_animation_key", "")).strip_edges()
	var entry_id: String = str(payload.get("entry_id", ""))
	if bool(payload.get("accepted", false)) and not animation_key.is_empty() and entry_id in ["quest_accept", "quest_submit"]:
		_close_runtime_menus(true)
		_on_quest_animation_requested(animation_key, "accept" if entry_id == "quest_accept" else "submit")
		return
	if str(payload.get("result_type", "")) == "battle":
		_close_runtime_menus(true)
		return
	if str(payload.get("result_type", "")) == "dialogue":
		_present_dialogue_node(int(payload.get("entity_id", 0)), str(payload.get("npc_name", "NPC")), payload.get("dialogue", {}))
		return
	if str(payload.get("result_type", "")) == "shop":
		_present_shop_payload(int(payload.get("entity_id", 0)), str(payload.get("npc_name", "NPC")), payload.get("shop", {}))
		return
	var notice: String = str(payload.get("notice", payload.get("reason", "")))
	if not notice.is_empty():
		_append_log("NPC 操作: %s" % notice)
	if payload.has("menu_entries") and _npc_menu != null:
		var menu_options: Array[Dictionary] = _build_npc_menu_options(payload)
		_close_other_root_panels("npc_menu")
		_npc_menu.configure(str(payload.get("npc_name", "NPC")), menu_options, {
			"render_mode": OptionListPanel.RENDER_NPC_ENTRY,
			"header_portrait": _resolve_npc_header_portrait(int(payload.get("entity_id", 0))),
		})
		_npc_menu.call("open_menu")
		_set_runtime_menu_locked(true)

## 处理 NPC 剧情继续/分支选择后的节点回包。
func _handle_npc_dialogue_payload(payload: Dictionary) -> void:
	if not bool(payload.get("accepted", false)):
		var failed_reason: String = str(payload.get("reason", "dialogue request failed"))
		_append_log("剧情推进失败: %s" % failed_reason)
		if _npc_dialogue_panel != null:
			_npc_dialogue_panel.show_waiting_state(failed_reason)
		return
	var effect_notice: String = ""
	var node_variant: Variant = payload.get("node", {})
	if node_variant is Dictionary:
		effect_notice = str((node_variant as Dictionary).get("effect_notice", ""))
	if not effect_notice.is_empty():
		_append_log("剧情提示: %s" % effect_notice)
		call_deferred("_show_dialogue_effect_notice", effect_notice)
	_present_dialogue_node(int(payload.get("entity_id", 0)), _active_dialogue_npc_name, payload.get("node", {}))

## 使用通用确认弹窗展示剧情节点阶段提示，不影响普通系统通知。
func _show_dialogue_effect_notice(message: String) -> void:
	_set_runtime_menu_locked(true)
	await _show_flow_confirm_prompt_and_wait("提示", message, 20)
	_set_runtime_menu_locked(false)

## 根据服务端返回的结构化剧情节点，决定是展示对话、播放客户端动画还是直接结束剧情。
func _present_dialogue_node(entity_id: int, npc_name: String, node_variant: Variant) -> void:
	if node_variant is not Dictionary:
		return
	var node_payload: Dictionary = node_variant as Dictionary
	_active_dialogue_entity_id = entity_id
	_active_dialogue_id = int(node_payload.get("dialogue_id", 0))
	_active_dialogue_node_id = str(node_payload.get("node_id", ""))
	_active_dialogue_npc_name = npc_name
	if bool(node_payload.get("is_end", false)) or str(node_payload.get("node_type", "")) == "end":
		_append_log("剧情结束: %s" % npc_name)
		_active_dialogue_entity_id = 0
		_active_dialogue_id = 0
		_active_dialogue_node_id = ""
		_active_dialogue_npc_name = ""
		if _npc_dialogue_panel != null:
			_npc_dialogue_panel.hide_panel()
		return
	if str(node_payload.get("node_type", "")) == "action":
		_play_dialogue_action_node(node_payload)
		return
	if _npc_dialogue_panel != null:
		_npc_dialogue_panel.show_dialogue(npc_name, node_payload)
	_set_runtime_menu_locked(true)
	_append_log("打开剧情对话: %s" % npc_name)

## 用服务端返回的商店载荷打开 NPC 商店面板。
func _present_shop_payload(entity_id: int, npc_name: String, shop_variant: Variant) -> void:
	if shop_variant is not Dictionary:
		return
	var shop_payload: Dictionary = shop_variant as Dictionary
	_active_shop_entity_id = entity_id
	if _npc_shop_panel != null:
		_npc_shop_panel.show_shop(npc_name, entity_id, shop_payload)
	_set_runtime_menu_locked(true)
	_append_log("打开商店: %s" % npc_name)

## 断线重连成功后恢复未结束的剧情或刷新全局摘要。
func _on_session_authenticated(payload: Dictionary) -> void:
	var active_dialogue_variant: Variant = payload.get("active_dialogue", {})
	if active_dialogue_variant is Dictionary:
		var active_dialogue: Dictionary = active_dialogue_variant as Dictionary
		if not active_dialogue.is_empty():
			_present_dialogue_node(
				int(active_dialogue.get("entity_id", 0)),
				str(active_dialogue.get("npc_name", "NPC")),
				active_dialogue.get("node", {})
			)

## 发起 BUY_ITEM 请求，并在回包到达后刷新商店钱包展示。
func _on_shop_buy_requested(item_id: int, _price_copper: int) -> void:
	if _active_shop_entity_id <= 0 or item_id <= 0 or _pending_buy_item_seq > 0:
		return
	_pending_buy_item_payload.clear()
	var request_seq: int = App.request_buy_item(_active_shop_entity_id, item_id, 1)
	if request_seq <= 0:
		if _npc_shop_panel != null:
			_npc_shop_panel.show_error_message("购买请求发送失败")
		return
	_pending_buy_item_seq = request_seq
	_show_npc_request_loading()
	call_deferred("_wait_npc_request", request_seq, CommandIds.BUY_ITEM_REQ, "_finish_buy_item_request")

## BUY_ITEM 回包到达后关闭 loading，并更新商店面板状态。
func _finish_buy_item_request(succeeded: bool) -> void:
	_hide_npc_request_loading()
	var payload: Dictionary = _pending_buy_item_payload.duplicate(true)
	_pending_buy_item_seq = 0
	_pending_buy_item_payload.clear()
	if _npc_shop_panel == null:
		return
	if not succeeded or payload.is_empty():
		_npc_shop_panel.show_error_message("购买失败")
		return
	_npc_shop_panel.update_wallet(payload.get("wallet", {}))

## 缓存 BUY_ITEM 回包，供 loading 等待完成后消费。
func _on_buy_item_responded(payload: Dictionary) -> void:
	if _pending_buy_item_seq > 0:
		_pending_buy_item_payload = payload.duplicate(true)

## 处理服务端权威场景剧情触发；客户端只负责播放本地动画，完成后再 Ack 服务端落库。
func _on_scene_trigger_push(payload: Dictionary) -> void:
	if _scene_map_transition_active:
		_queued_scene_trigger_payload = payload.duplicate(true)
		return
	if not _active_quest_animation_action.is_empty():
		_queued_scene_trigger_payload = payload.duplicate(true)
		return
	var trigger_code: String = str(payload.get("trigger_code", "")).strip_edges()
	var animation_key: String = str(payload.get("client_animation_key", "")).strip_edges()
	var prompt_text: String = str(payload.get("prompt_text", "")).strip_edges()
	if trigger_code.is_empty():
		return
	_pending_scene_trigger_code = trigger_code
	_pending_scene_trigger_block_movement = bool(payload.get("block_movement", true))
	if _pending_scene_trigger_block_movement:
		_set_runtime_menu_locked(true)
	_pending_scene_trigger_prompt_text = prompt_text
	if not prompt_text.is_empty() and animation_key.is_empty():
		call_deferred("_show_scene_trigger_prompt", prompt_text, animation_key)
		return
	if _cinematic_player == null or animation_key.is_empty():
		_ack_pending_scene_trigger()
		return
	_cinematic_player.play_cinematic(animation_key)

## 展示服务端配置的一次性场景提示，确认后再播放动画或完成触发器。
func _show_scene_trigger_prompt(prompt_text: String, animation_key: String) -> void:
	await _show_flow_confirm_prompt_and_wait("提示", prompt_text, 20)
	if _cinematic_player == null or animation_key.is_empty():
		_ack_pending_scene_trigger()
		return
	_cinematic_player.play_cinematic(animation_key)

## 向服务端确认当前场景剧情已播放完毕，由服务端解锁 NPC 并接取后续主线任务。
func _ack_pending_scene_trigger() -> void:
	if _pending_scene_trigger_code.is_empty() or _pending_scene_trigger_ack_seq > 0:
		return
	var trigger_code: String = _pending_scene_trigger_code
	_pending_scene_trigger_code = ""
	_pending_scene_trigger_prompt_text = ""
	_pending_scene_trigger_block_movement = false
	_set_runtime_menu_locked(false)
	var request_seq: int = App.ack_scene_trigger(trigger_code)
	if request_seq <= 0:
		return
	_pending_scene_trigger_ack_seq = request_seq
	_show_npc_request_loading()
	call_deferred("_wait_npc_request", request_seq, CommandIds.SCENE_TRIGGER_ACK_REQ, "_finish_scene_trigger_ack_request")

## 场景剧情 Ack 回包到达后关闭 loading；世界快照和任务变化由服务端后续 push 更新。
func _finish_scene_trigger_ack_request(_succeeded: bool) -> void:
	_hide_npc_request_loading()
	_pending_scene_trigger_ack_seq = 0
	_play_next_queued_quest_animation()

## 播放服务端指定的客户端内置剧情动画；阻塞型动画播完后再向服务端请求下一节点。
func _play_dialogue_action_node(node_payload: Dictionary) -> void:
	if _npc_dialogue_panel != null:
		_npc_dialogue_panel.show_waiting_state("剧情演出中...")
	var animation_key: String = str(node_payload.get("client_animation_key", ""))
	var must_wait: bool = bool(node_payload.get("client_animation_block", false))
	_waiting_blocking_cinematic = must_wait
	if _cinematic_player == null:
		_waiting_blocking_cinematic = false
		_request_next_dialogue_node()
		return
	_cinematic_player.play_cinematic(animation_key)
	if not must_wait:
		_request_next_dialogue_node()

## 使用当前缓存的剧情上下文向服务端请求下一节点，保证推进顺序始终由服务端权威决定。
func _request_next_dialogue_node() -> void:
	if _active_dialogue_entity_id <= 0 or _active_dialogue_id <= 0 or _active_dialogue_node_id.is_empty():
		return
	if _pending_dialogue_request_seq > 0:
		return
	var request_seq: int = App.request_npc_dialogue_next(_active_dialogue_entity_id, _active_dialogue_id, _active_dialogue_node_id)
	_begin_dialogue_request(CommandIds.NPC_DIALOGUE_NEXT_REQ, request_seq)

## 发起剧情推进请求并进入 loading 等待态。
func _begin_dialogue_request(request_cmd: int, request_seq: int) -> void:
	if request_seq <= 0:
		return
	_pending_dialogue_request_seq = request_seq
	_pending_dialogue_payload.clear()
	_show_npc_request_loading()
	call_deferred("_wait_npc_request", request_seq, request_cmd, "_finish_dialogue_request")

## 响应对话面板继续按钮点击，先锁住面板输入，再向服务端请求下一节点。
func _on_dialogue_continue_requested(dialogue_id: int, node_id: String) -> void:
	_active_dialogue_id = dialogue_id
	_active_dialogue_node_id = node_id
	_request_next_dialogue_node()

## 展示客户端固定过场脚本中的对白；该内容不会创建或推进服务端剧情节点。
func _on_local_cinematic_dialogue_requested(
	speaker_name: String,
	content: String,
	portrait_key: String,
	is_player_speaking: bool,
	content_format: String,
	portrait_texture: Texture2D
) -> void:
	if _npc_dialogue_panel == null:
		if _cinematic_player != null:
			_cinematic_player.call_deferred("advance_local_dialogue")
		return
	_npc_dialogue_panel.show_local_dialogue(
		speaker_name,
		content,
		portrait_key,
		is_player_speaking,
		content_format,
		portrait_texture
	)
	_set_runtime_menu_locked(true)

## 本地对白点击继续后只推进当前客户端过场，不发送 NPC_DIALOGUE_NEXT_REQ。
func _on_local_dialogue_continue_requested() -> void:
	if _npc_dialogue_panel != null:
		_npc_dialogue_panel.hide_panel(false)
	if _cinematic_player != null:
		_cinematic_player.advance_local_dialogue()

## 响应对话面板选项按钮点击，分支结果由服务端校验后返回下一个节点。
func _on_dialogue_option_selected(dialogue_id: int, node_id: String, option_id: String) -> void:
	if _active_dialogue_entity_id <= 0:
		return
	if _pending_dialogue_request_seq > 0:
		return
	_active_dialogue_id = dialogue_id
	_active_dialogue_node_id = node_id
	var request_seq: int = App.request_npc_dialogue_choose(_active_dialogue_entity_id, dialogue_id, node_id, option_id)
	_begin_dialogue_request(CommandIds.NPC_DIALOGUE_CHOOSE_REQ, request_seq)

## 客户端内置剧情动画播放完成后，继续向服务端请求下一节点。
func _on_cinematic_finished(_animation_key: String) -> void:
	if _npc_dialogue_panel != null and _npc_dialogue_panel.is_local_dialogue_active():
		_npc_dialogue_panel.hide_panel(false)
	if not _pending_scene_trigger_code.is_empty():
		if not _pending_scene_trigger_prompt_text.is_empty():
			var prompt_text: String = _pending_scene_trigger_prompt_text
			_pending_scene_trigger_prompt_text = ""
			call_deferred("_show_scene_trigger_prompt", prompt_text, "")
			return
		_ack_pending_scene_trigger()
		return
	if not _active_quest_animation_action.is_empty():
		var completed_action: String = _active_quest_animation_action
		_active_quest_animation_action = ""
		_set_runtime_menu_locked(false)
		quest_controller.finish_quest_animation(completed_action)
		if not _queued_scene_trigger_payload.is_empty():
			var queued_scene_trigger: Dictionary = _queued_scene_trigger_payload.duplicate(true)
			_queued_scene_trigger_payload.clear()
			_on_scene_trigger_push(queued_scene_trigger)
			return
		_play_next_queued_quest_animation()
		return
	if not _waiting_blocking_cinematic:
		return
	_waiting_blocking_cinematic = false
	_request_next_dialogue_node()

## 锁定运行时交互并复用通用剧情播放器播放任务领取或交付动画。
func _on_quest_animation_requested(animation_key: String, action: String) -> void:
	if animation_key.is_empty() or _cinematic_player == null:
		quest_controller.finish_quest_animation(action)
		return
	if not _active_quest_animation_action.is_empty() or not _pending_scene_trigger_code.is_empty() or _waiting_blocking_cinematic:
		_queued_quest_animations.append({"animation_key": animation_key, "action": action})
		return
	_active_quest_animation_action = action
	_set_runtime_menu_locked(true)
	_cinematic_player.play_cinematic(animation_key)

## 当前没有更高优先级剧情时，播放队列中的下一条任务动画。
func _play_next_queued_quest_animation() -> void:
	if _queued_quest_animations.is_empty():
		return
	var next_animation: Dictionary = _queued_quest_animations.pop_front() as Dictionary
	_on_quest_animation_requested(
		str(next_animation.get("animation_key", "")),
		str(next_animation.get("action", ""))
	)

## 新战斗开始时关闭所有会挡住战斗的弹窗页面。
func _close_all_blocking_popups_for_battle() -> void:
	_popup_flow_generation += 1
	_pending_npc_menu_seq = 0
	_pending_npc_menu_entity_id = 0
	_pending_npc_menu_scene_id = 0
	_pending_npc_menu_should_open = false
	_pending_npc_action_seq = 0
	_pending_dialogue_request_seq = 0
	_pending_scene_trigger_ack_seq = 0
	_pending_npc_menu_payload.clear()
	_pending_npc_action_payload.clear()
	_pending_dialogue_payload.clear()
	_hide_npc_request_loading()
	_close_runtime_menus(true)
	_force_close_runtime_modal_popups()
	_force_close_pvp_invite_dialog()
	_force_close_dialogue_balloon()
	_suppress_settlement_input_leak()

## 关闭所有继承模态弹窗基类的运行时弹窗（升级、奖励及后续同类页面）。
func _force_close_runtime_modal_popups() -> void:
	for node_variant: Variant in get_tree().get_nodes_in_group("runtime_modal_popup"):
		if not node_variant is Node:
			continue
		var popup_node: Node = node_variant as Node
		if popup_node.has_method("force_close_popup"):
			popup_node.call("force_close_popup")

func _force_close_pvp_invite_dialog() -> void:
	if _pvp_invite_dialog != null and _pvp_invite_dialog.visible:
		_pvp_invite_dialog.hide()

func _force_close_dialogue_balloon() -> void:
	_waiting_blocking_cinematic = false
	if _cinematic_player != null:
		_cinematic_player.cancel_active_cinematic()
	_active_dialogue_entity_id = 0
	_active_dialogue_id = 0
	_active_dialogue_node_id = ""
	_active_dialogue_npc_name = ""
	if _npc_dialogue_panel != null:
		_npc_dialogue_panel.hide_panel(false)

func _set_runtime_menu_locked(locked: bool) -> void:
	if _world_controller != null and _world_controller.has_method("set_runtime_input_locked"):
		_world_controller.call("set_runtime_input_locked", locked)

## 结算弹窗关闭时通知主场景：当前帧剩余输入全部丢弃。
func _suppress_settlement_input_leak() -> void:
	_suppress_settlement_input_until_frame = Engine.get_process_frames()

## 判断流程确认或奖励结算弹窗是否正在展示。
func _is_settlement_popup_active() -> bool:
	var flow_prompt_visible: bool = _flow_confirm_popup != null and _flow_confirm_popup.visible
	var reward_visible: bool = _reward_popup != null and _reward_popup.visible
	return flow_prompt_visible or reward_visible

## 判断结算弹窗是否应继续阻断快捷键与底层交互。
func _is_settlement_input_blocked() -> bool:
	if _is_settlement_popup_active():
		return true
	return Engine.get_process_frames() <= _suppress_settlement_input_until_frame

## 关闭所有运行时菜单；战斗中传入 keep_world_locked=true 避免误解锁世界输入。
func _close_runtime_menus(keep_world_locked: bool = false) -> void:
	for layer in [_main_menu, _player_panel, _pet_panel, _bag_panel, _task_panel, _npc_menu, _npc_list_menu, _pvp_target_menu, _auto_encounter_target_menu]:
		if layer != null and layer.has_method("close_menu"):
			layer.call("close_menu")
	if _pvp_invite_dialog != null:
		_pvp_invite_dialog.hide()
	if not keep_world_locked:
		_set_runtime_menu_locked(false)


## 打开某个根面板前关闭其它互斥根面板（不解除世界锁定）。
func _close_other_root_panels(keep_key: String) -> void:
	var panel_entries: Array[Array] = [
		["main_menu", _main_menu],
		["player_panel", _player_panel],
		["pet_panel", _pet_panel],
		["bag_panel", _bag_panel],
		["task_panel", _task_panel],
		["npc_menu", _npc_menu],
		["npc_list", _npc_list_menu],
		["pvp_list", _pvp_target_menu],
		["auto_encounter_target", _auto_encounter_target_menu],
	]
	for entry: Array in panel_entries:
		var panel_key: String = str(entry[0])
		if panel_key == keep_key:
			continue
		var panel_layer: CanvasLayer = entry[1] as CanvasLayer
		if panel_layer != null and panel_layer.visible and panel_layer.has_method("close_menu"):
			panel_layer.call("close_menu")


## 切换根面板：已打开则关闭；否则先关其它根面板再打开目标面板。
func _toggle_root_runtime_panel(panel: CanvasLayer, panel_key: String) -> void:
	if panel == null:
		return
	if panel.visible:
		panel.call("close_menu")
		return
	if panel_key == "bag_panel":
		call_deferred("_open_bag_panel_prepared")
		return
	if panel_key == "task_panel":
		call_deferred("_open_task_panel_prepared")
		return
	if panel_key == "player_panel":
		call_deferred("_open_player_panel_prepared")
		return
	if panel_key == "pet_panel":
		call_deferred("_open_pet_panel_prepared")
		return
	_close_other_root_panels(panel_key)
	panel.call("open_menu")
	_set_runtime_menu_locked(true)


## 先展示全屏 loading 并拉取人物/背包/钱包权威快照，全部就绪后再打开人物状态面板。
func _open_player_panel_prepared() -> void:
	if _player_panel == null:
		return
	if _player_panel.visible:
		_player_panel.call("close_menu")
		return
	if _player_panel_open_in_flight:
		return
	_player_panel_open_in_flight = true
	_close_other_root_panels("player_panel")
	_create_npc_request_loading()
	_show_npc_request_loading_immediate()
	var prepared: bool = false
	var player_panel: CanvasLayer = _player_panel
	if player_panel != null and player_panel.has_method("prepare_open_data"):
		prepared = bool(await player_panel.call("prepare_open_data"))
	elif _player_panel.has_method("prepare_open_data"):
		var prepare_result: Variant = await _player_panel.call("prepare_open_data")
		prepared = bool(prepare_result)
	_hide_npc_request_loading()
	_player_panel_open_in_flight = false
	if not prepared:
		return
	if player_panel != null and player_panel.has_method("open_menu"):
		player_panel.call("open_menu")
	elif _player_panel.has_method("open_menu"):
		_player_panel.call("open_menu")
	_set_runtime_menu_locked(true)


## 先展示全屏 loading 并拉取背包/装备数据，全部就绪后再打开背包面板。
func _open_bag_panel_prepared() -> void:
	if _bag_panel == null:
		return
	if _bag_panel.visible:
		_bag_panel.call("close_menu")
		return
	if _bag_panel_open_in_flight:
		return
	_bag_panel_open_in_flight = true
	_close_other_root_panels("bag_panel")
	_create_npc_request_loading()
	_show_npc_request_loading_immediate()
	var prepared: bool = false
	var bag_panel: BagPanel = _bag_panel as BagPanel
	if bag_panel != null:
		prepared = await bag_panel.prepare_open_data()
	elif _bag_panel.has_method("prepare_open_data"):
		var prepare_result: Variant = await _bag_panel.call("prepare_open_data")
		prepared = bool(prepare_result)
	_hide_npc_request_loading()
	_bag_panel_open_in_flight = false
	if not prepared:
		return
	if bag_panel != null:
		bag_panel.open_menu()
	elif _bag_panel.has_method("open_menu"):
		_bag_panel.call("open_menu")
	_set_runtime_menu_locked(true)

## 先展示全屏 loading 并拉取任务列表，权威快照就绪后再打开任务面板。
func _open_task_panel_prepared() -> void:
	if _task_panel == null:
		return
	if _task_panel.visible:
		_task_panel.call("close_menu")
		return
	if _task_panel_open_in_flight:
		return
	_task_panel_open_in_flight = true
	_close_other_root_panels("task_panel")
	_create_npc_request_loading()
	_show_npc_request_loading_immediate()
	var prepared: bool = false
	var task_panel: CanvasLayer = _task_panel
	if task_panel != null and task_panel.has_method("prepare_open_data"):
		var prepare_result: Variant = await task_panel.call("prepare_open_data")
		prepared = bool(prepare_result)
	_hide_npc_request_loading()
	_task_panel_open_in_flight = false
	if not prepared:
		return
	if task_panel != null and task_panel.has_method("open_menu"):
		task_panel.call("open_menu")
	_set_runtime_menu_locked(true)

## 先展示全屏 loading 并拉取宠物列表，权威快照就绪后再打开宠物状态面板。
func _open_pet_panel_prepared() -> void:
	if _pet_panel == null:
		return
	if _pet_panel.visible:
		_pet_panel.call("close_menu")
		return
	if _pet_panel_open_in_flight:
		return
	_pet_panel_open_in_flight = true
	_close_other_root_panels("pet_panel")
	_create_npc_request_loading()
	_show_npc_request_loading_immediate()
	var prepared: bool = false
	var pet_panel: CanvasLayer = _pet_panel
	if pet_panel != null and pet_panel.has_method("prepare_open_data"):
		var prepare_result: Variant = await pet_panel.call("prepare_open_data")
		prepared = bool(prepare_result)
	_hide_npc_request_loading()
	_pet_panel_open_in_flight = false
	if not prepared:
		return
	if pet_panel != null and pet_panel.has_method("open_menu"):
		pet_panel.call("open_menu")
	_set_runtime_menu_locked(true)

func _has_blocking_ui_open(except: String = "") -> bool:
	if except != "main_menu" and _main_menu != null and _main_menu.visible:
		return true
	if except != "player_panel" and _player_panel != null and _player_panel.visible:
		return true
	if except != "pet_panel" and _pet_panel != null and _pet_panel.visible:
		return true
	if except != "bag_panel" and _bag_panel != null and _bag_panel.visible:
		return true
	if except != "task_panel" and _task_panel != null and _task_panel.visible:
		return true
	if except != "npc_menu" and _npc_menu != null and _npc_menu.visible:
		return true
	if except != "npc_list" and _npc_list_menu != null and _npc_list_menu.visible:
		return true
	if except != "pvp_list" and _pvp_target_menu != null and _pvp_target_menu.visible:
		return true
	if except != "pvp_invite" and _pvp_invite_dialog != null and _pvp_invite_dialog.visible:
		return true
	if except != "npc_shop" and _npc_shop_panel != null and _npc_shop_panel.visible:
		return true
	if except != "npc_dialogue" and _npc_dialogue_panel != null and _npc_dialogue_panel.visible:
		return true
	if _is_settlement_input_blocked():
		return true
	return false

## 判断战斗弹窗是否仍处于展示态（含结算演出尚未卸载的阶段）。
func _is_battle_modal_active() -> bool:
	return _battle_modal_layer != null and _battle_modal_layer.visible

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

## 把全屏过渡遮罩挂到独立高层 CanvasLayer，避免被战斗弹窗挡住。
func _ensure_transition_canvas_layer() -> void:
	if _transition_canvas_layer != null:
		return
	_transition_canvas_layer = CanvasLayer.new()
	_transition_canvas_layer.name = "TransitionLayer"
	_transition_canvas_layer.layer = TRANSITION_LAYER
	add_child(_transition_canvas_layer)
	if transition_overlay == null:
		return
	var overlay_parent: Node = transition_overlay.get_parent()
	if overlay_parent != null:
		overlay_parent.remove_child(transition_overlay)
	_transition_canvas_layer.add_child(transition_overlay)
	transition_overlay.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

## 懒加载网格铺展转场，并挂到高层过渡 CanvasLayer 上。
func _ensure_grid_spread_transition() -> void:
	_ensure_transition_canvas_layer()
	if _grid_spread_transition != null or _transition_canvas_layer == null:
		return
	_grid_spread_transition = GRID_SPREAD_SCENE.instantiate() as GridSpreadTransition
	if _grid_spread_transition == null:
		return
	_transition_canvas_layer.add_child(_grid_spread_transition)
	_grid_spread_transition.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)

## 地图切换过渡：前半段渐入全黑，中点等待新场景就绪后换图，后半段渐出。
func _run_scene_map_transition(generation: int) -> void:
	_ensure_transition_canvas_layer()
	_scene_map_transition_active = true
	if transition_overlay == null:
		_scene_map_transition_active = false
		return
	var half_duration: float = SCENE_MAP_TRANSITION_DURATION * 0.5
	_debug_scene_transition("generation=%d fade to black begin" % generation)
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(1.0, half_duration)
	if generation != _scene_map_transition_generation:
		_debug_scene_transition("generation=%d cancelled before midpoint" % generation)
		return
	_debug_scene_transition("generation=%d reached midpoint" % generation)
	_apply_scene_map_switch_at_midpoint()
	var load_deadline_msec: int = Time.get_ticks_msec() + SCENE_MAP_LOAD_TIMEOUT_MSEC
	if not _scene_map_transition_failed:
		while not _scene_map_new_scene_ready:
			if generation != _scene_map_transition_generation:
				return
			if _scene_map_transition_failed:
				break
			if Time.get_ticks_msec() >= load_deadline_msec:
				_scene_map_transition_failed = true
				_debug_scene_transition("generation=%d timed out waiting for world snapshot" % generation)
				_abort_world_scene_transition("world snapshot timeout")
				break
			if not await _wait_for_next_process_frame():
				return
	if not _scene_map_transition_failed:
		if not await _wait_for_next_process_frame():
			return
	if generation != _scene_map_transition_generation:
		return
	var has_queued_scene_trigger: bool = not _queued_scene_trigger_payload.is_empty()
	_debug_scene_transition(
		"generation=%d finish failed=%s scene_ready=%s queued_trigger=%s" % [
			generation,
			str(_scene_map_transition_failed),
			str(_scene_map_new_scene_ready),
			str(has_queued_scene_trigger),
		]
	)
	if not has_queued_scene_trigger or _scene_map_transition_failed:
		await _fade_overlay(0.0, half_duration)
		transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_scene_map_transition_active = false
	_scene_map_new_scene_ready = false
	_scene_map_transition_failed = false
	_unlock_scene_visual_apply_for_transition()
	_play_queued_scene_trigger_after_map_transition()


## 输出主场景地图遮罩与等待阶段日志，和世界控制器日志使用同一检索前缀。
## message 是切图代次、遮罩阶段或失败原因。
func _debug_scene_transition(message: String) -> void:
	print("[SceneTransition][Client][Main] %s" % message)

## 地图在全黑状态加载完成后直接播放场景剧情，避免目标世界画面短暂闪现。
func _play_queued_scene_trigger_after_map_transition() -> void:
	if _queued_scene_trigger_payload.is_empty() or not _active_quest_animation_action.is_empty():
		return
	var queued_scene_trigger: Dictionary = _queued_scene_trigger_payload.duplicate(true)
	_queued_scene_trigger_payload.clear()
	call_deferred("_on_scene_trigger_push", queued_scene_trigger)

## 过渡中点：解除世界场景视觉锁并应用已缓存的新地图快照。
func _apply_scene_map_switch_at_midpoint() -> void:
	if _world_controller == null:
		return
	if _world_controller.has_method("flush_deferred_scene_apply_at_midpoint"):
		_world_controller.call("flush_deferred_scene_apply_at_midpoint")


## 地图过渡开始时锁定世界换图，直到中点再真正加载新场景。
func _lock_scene_visual_apply_for_transition() -> void:
	if _world_controller == null:
		return
	if _world_controller.has_method("set_scene_visual_apply_locked"):
		_world_controller.call("set_scene_visual_apply_locked", true)


## 地图过渡结束或失败时清理世界换图锁，避免后续 resync 被误延迟。
func _unlock_scene_visual_apply_for_transition() -> void:
	if _world_controller == null:
		return
	if _world_controller.has_method("cancel_scene_visual_apply_lock"):
		_world_controller.call("cancel_scene_visual_apply_lock")


## 通知世界控制器完整回滚未完成的切图状态，确保超时后玩家可以继续移动和再次触发传送门。
## reason 是写入切图定向日志的中止原因。
func _abort_world_scene_transition(reason: String) -> void:
	if _world_controller == null:
		return
	if _world_controller.has_method("abort_scene_transition"):
		_world_controller.call("abort_scene_transition", reason)

# 播放主场景进入时的遮罩淡入动画。
func _play_fade_in() -> void:
	transition_overlay.color.a = 1.0
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(0.0)
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

# 把遮罩透明度过渡到指定目标值。
func _fade_overlay(target_alpha: float, duration: float = TRANSITION_DURATION) -> void:
	if transition_overlay == null:
		return
	if _overlay_tween != null and _overlay_tween.is_valid():
		_overlay_tween.kill()
	_overlay_tween = create_tween()
	_overlay_tween.tween_property(transition_overlay, "color:a", target_alpha, duration)
	await _overlay_tween.finished


## 安全等待下一渲染帧；切换顶层场景时节点可能已经离树，此时直接终止原异步流程。
func _wait_for_next_process_frame() -> bool:
	var scene_tree: SceneTree = get_tree()
	if scene_tree == null:
		return false
	await scene_tree.process_frame
	return is_inside_tree() and get_tree() != null

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
