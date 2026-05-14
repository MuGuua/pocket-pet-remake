extends Node

signal bootstrapped
signal login_succeeded(response: Dictionary)
signal login_failed(message: String)
signal session_authenticated(payload: Dictionary)
signal notice_received(message: String)
signal kicked(reason: String)

const DEFAULT_ACCOUNT: String = "demo"
const DEFAULT_PASSWORD: String = "demo123"
const DEFAULT_BATTLE_SKILL_ID: int = 1001

var _bootstrapped: bool = false
var _next_battle_op_id: int = 1

func _ready() -> void:
    process_mode = Node.PROCESS_MODE_ALWAYS

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

func login(account: String, password: String) -> Dictionary:
    var response: Dictionary = await HttpClient.login(account, password)
    var code: int = int(response.get("code", 0))
    if code != 200:
        login_failed.emit(str(response.get("msg", "login failed")))
        return response

    var data_variant: Variant = response.get("data", {})
    var data: Dictionary = data_variant if data_variant is Dictionary else {}
    GameState.store_login_result(data)
    login_succeeded.emit(response)
    return response

func login_with_demo_account() -> Dictionary:
    return await login(DEFAULT_ACCOUNT, DEFAULT_PASSWORD)

func connect_ws() -> int:
    return NetClient.connect_to_server()

func authenticate_ws() -> void:
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

func enter_world() -> void:
    NetClient.send_command(CommandIds.ENTER_WORLD_REQ, {})

func request_pet_list() -> void:
    NetClient.send_command(CommandIds.PET_LIST_REQ, {})

func request_bag_list() -> void:
    NetClient.send_command(CommandIds.BAG_LIST_REQ, {})

func request_interact(entity_id: int) -> void:
    NetClient.send_command(
        CommandIds.INTERACT_REQ,
        {
            "entity_id": entity_id,
        }
    )

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

func _on_dev_message_received(cmd: int, payload: Dictionary) -> void:
    MessageRouter.route_message(cmd, payload)

func _on_websocket_opened() -> void:
    authenticate_ws()

func _on_websocket_closed(_code: int, _reason: String) -> void:
    NetClient.set_authenticated(false)
    GameState.set_ws_authenticated(false)

func _on_ws_auth_response(payload: Dictionary) -> void:
    GameState.store_ws_session(payload)
    NetClient.set_authenticated(true)
    NetClient.configure_heartbeat(GameState.heartbeat_sec)
    session_authenticated.emit(payload)

func _on_heartbeat_response(_payload: Dictionary) -> void:
    pass

func _on_force_offline_push(payload: Dictionary) -> void:
    var reason := str(payload.get("reason", "account logged in elsewhere"))
    kicked.emit(reason)

func _on_error_push(payload: Dictionary) -> void:
    notice_received.emit(str(payload.get("msg", "server returned an error push")))

func _on_notice_push(payload: Dictionary) -> void:
    notice_received.emit(str(payload.get("message", payload.get("msg", ""))))

func _on_kickout_push(payload: Dictionary) -> void:
    var reason := str(payload.get("reason", payload.get("msg", "kicked by server")))
    kicked.emit(reason)

func _take_battle_op_id() -> int:
    var op_id := _next_battle_op_id
    _next_battle_op_id += 1
    if _next_battle_op_id > 0x7FFFFFFF:
        _next_battle_op_id = 1
    return op_id
