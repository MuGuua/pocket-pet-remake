extends Node

signal bootstrapped
signal login_succeeded(response: Dictionary)
signal login_failed(message: String)
signal notice_received(message: String)
signal kicked(reason: String)

const DEFAULT_ACCOUNT: String = "demo"
const DEFAULT_PASSWORD: String = "demo123"

var _bootstrapped: bool = false

func _ready() -> void:
    process_mode = Node.PROCESS_MODE_ALWAYS

func bootstrap() -> void:
    if _bootstrapped:
        return

    _bootstrapped = true
    NetClient.dev_message_received.connect(_on_dev_message_received)
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

    NetClient.send_command(CommandIds.WS_AUTH_REQ, {"ws_token": GameState.ws_token})

func enter_world() -> void:
    NetClient.send_command(CommandIds.ENTER_WORLD_REQ, {})

func request_pet_list() -> void:
    NetClient.send_command(CommandIds.PET_LIST_REQ, {})

func request_bag_list() -> void:
    NetClient.send_command(CommandIds.BAG_LIST_REQ, {})

func _on_dev_message_received(cmd: int, payload: Dictionary) -> void:
    MessageRouter.route_message(cmd, payload)

func _on_error_push(payload: Dictionary) -> void:
    notice_received.emit(str(payload.get("msg", "server returned an error push")))

func _on_notice_push(payload: Dictionary) -> void:
    notice_received.emit(str(payload.get("message", payload.get("msg", ""))))

func _on_kickout_push(payload: Dictionary) -> void:
    var reason := str(payload.get("reason", payload.get("msg", "kicked by server")))
    kicked.emit(reason)
