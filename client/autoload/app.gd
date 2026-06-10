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
# 收到强制下线信息后向外广播原因。
signal kicked(reason: String)

# 默认演示账号名。
const DEFAULT_ACCOUNT: String = "demo"
# 默认演示账号密码。
const DEFAULT_PASSWORD: String = "demo123"
# 默认战斗动作使用的技能标识。
const DEFAULT_BATTLE_SKILL_ID: int = 1001

# 标记当前单例是否已经完成初始化绑定。
var _bootstrapped: bool = false
# 生成战斗类操作标识时使用的自增计数器。
var _next_battle_op_id: int = 1

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

func login_with_demo_account() -> Dictionary:
# 使用预设演示账号执行登录流程。
    return await login(DEFAULT_ACCOUNT, DEFAULT_PASSWORD)

func connect_ws() -> int:
# 发起到服务端的 WebSocket 连接。
    return NetClient.connect_to_server()

func authenticate_ws() -> void:
# 使用当前保存的 `ws_token` 发起实时连接鉴权。
    if GameState.ws_token.is_empty():
        push_warning("Missing ws_token. Login before websocket auth.")
        return


    NetClient.send_command(
        CommandIds.WS_AUTH_REQ,
        {
            "ws_token": GameState.ws_token,
            "client_version": "godot-4.5-dev",
        }
    )

# 请求进入当前服务端权威世界场景。
func enter_world() -> void:
    NetClient.send_command(CommandIds.ENTER_WORLD_REQ, {})

# 请求刷新当前玩家的宠物列表。
func request_pet_list() -> void:
    NetClient.send_command(CommandIds.PET_LIST_REQ, {})

# 提交完整编队宠物唯一标识列表。
func set_pet_lineup(pet_uids: Array[int]) -> void:
    NetClient.send_command(
        CommandIds.PET_LINEUP_SET_REQ,
        {
            "op_id": _take_battle_op_id(),
            "pet_uids": pet_uids,
        }
    )

# 请求刷新当前玩家的背包摘要。
func request_bag_list() -> void:
    NetClient.send_command(CommandIds.BAG_LIST_REQ, {})

# 请求刷新当前玩家的任务列表。
func request_quest_list() -> void:
    NetClient.send_command(CommandIds.QUEST_LIST_REQ, {})

# 请求接取指定任务。
func accept_quest(quest_id: int, npc_id: int = 0) -> void:
    NetClient.send_command(CommandIds.QUEST_ACCEPT_REQ, {"quest_id": quest_id, "npc_id": npc_id})

# 请求提交指定任务；NPC 交付任务时同时带上提交 NPC。
func submit_quest(quest_id: int, npc_id: int = 0) -> void:
    NetClient.send_command(CommandIds.QUEST_SUBMIT_REQ, {"quest_id": quest_id, "npc_id": npc_id})

# 请求切换当前追踪任务。
func track_quest(quest_id: int) -> void:
    NetClient.send_command(CommandIds.QUEST_TRACK_REQ, {"quest_id": quest_id})

# 向服务端提交与指定实体的交互意图。
func request_interact(entity_id: int) -> void:
    NetClient.send_command(
        CommandIds.INTERACT_REQ,
        {
            "entity_id": entity_id,
        }
    )

# 向服务端提交 NPC 菜单项执行请求。
func request_npc_action(entity_id: int, entry_id: String) -> void:
    NetClient.send_command(
        CommandIds.NPC_ACTION_REQ,
        {
            "entity_id": entity_id,
            "entry_id": entry_id,
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
    NetClient.send_command(
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
    NetClient.send_command(
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
func _on_dev_message_received(cmd: int, payload: Dictionary) -> void:
    MessageRouter.route_message(cmd, payload)

# 底层 WebSocket 建连成功后自动发起业务鉴权。
func _on_websocket_opened() -> void:
    authenticate_ws()

# 底层 WebSocket 关闭后同步清空鉴权状态。
func _on_websocket_closed(_code: int, _reason: String) -> void:
    NetClient.set_authenticated(false)
    GameState.set_ws_authenticated(false)

# 处理 WebSocket 鉴权成功回执，并配置心跳和会话状态。
func _on_ws_auth_response(payload: Dictionary) -> void:
    GameState.store_ws_session(payload)
    NetClient.set_authenticated(true)
    NetClient.configure_heartbeat(GameState.heartbeat_sec)
    session_authenticated.emit(payload)

# 处理心跳回包，当前仅用于维持链路，无额外业务逻辑。
func _on_heartbeat_response(_payload: Dictionary) -> void:
    pass

# 处理服务端强制下线推送。
func _on_force_offline_push(payload: Dictionary) -> void:
    # 提取服务端返回的下线原因文本。
    var reason := str(payload.get("reason", "account logged in elsewhere"))
    kicked.emit(reason)

# 处理服务端错误推送，并转为统一提示事件。
func _on_error_push(payload: Dictionary) -> void:
    notice_received.emit(str(payload.get("msg", "server returned an error push")))

# 处理服务端普通公告推送，并兼容不同字段名。
func _on_notice_push(payload: Dictionary) -> void:
    notice_received.emit(str(payload.get("message", payload.get("msg", ""))))

# 处理服务端踢下线推送。
func _on_kickout_push(payload: Dictionary) -> void:
    # 提取服务端返回的踢下线原因文本。
    var reason := str(payload.get("reason", payload.get("msg", "kicked by server")))
    kicked.emit(reason)

# 返回下一个战斗类请求使用的操作标识，并在上限后回绕。
func _take_battle_op_id() -> int:
    # 暂存当前可返回的操作标识。
    var op_id := _next_battle_op_id
    _next_battle_op_id += 1
    if _next_battle_op_id > 0x7FFFFFFF:
        _next_battle_op_id = 1
    return op_id
