extends Node

const WORLD_SCENE := preload("res://scenes/world/world_scene.tscn")

@onready var world_mount: Node2D = %WorldMount
@onready var pet_controller: Node = %PetController
@onready var battle_controller: Node = %BattleController
@onready var bag_controller: Node = %BagController
@onready var status_label: Label = %StatusLabel
@onready var scene_label: Label = %SceneLabel
@onready var player_label: Label = %PlayerLabel
@onready var demo_login_button: Button = %DemoLoginButton
@onready var connect_button: Button = %ConnectButton
@onready var enter_world_button: Button = %EnterWorldButton
@onready var log_output: RichTextLabel = %LogOutput

var _world_controller: Node

func _ready() -> void:
    App.bootstrap()
    _mount_world_scene()
    _register_routes()
    _connect_signals()
    _append_log("Client skeleton ready.")
    _append_log("Next step: implement protobuf packet codec in NetClient and bind live WS auth.")
    _refresh_view()

func _mount_world_scene() -> void:
    _world_controller = WORLD_SCENE.instantiate()
    world_mount.add_child(_world_controller)
    if _world_controller.has_signal("scene_loaded"):
        _world_controller.connect("scene_loaded", Callable(self, "_on_world_scene_loaded"))
    if _world_controller.has_signal("player_position_changed"):
        _world_controller.connect("player_position_changed", Callable(self, "_on_player_position_changed"))
    _append_log("Mounted world scene skeleton.")

func _register_routes() -> void:
    if _world_controller == null:
        return

    MessageRouter.register_handler(CommandIds.ENTER_WORLD_RESP, Callable(_world_controller, "handle_enter_world"))
    MessageRouter.register_handler(CommandIds.ENTITY_ENTER_PUSH, Callable(_world_controller, "handle_entity_enter"))
    MessageRouter.register_handler(CommandIds.ENTITY_LEAVE_PUSH, Callable(_world_controller, "handle_entity_leave"))
    MessageRouter.register_handler(CommandIds.ENTITY_MOVE_PUSH, Callable(_world_controller, "handle_entity_move"))
    MessageRouter.register_handler(CommandIds.WORLD_RESYNC_PUSH, Callable(_world_controller, "handle_world_resync"))

    MessageRouter.register_handler(CommandIds.PET_LIST_RESP, Callable(pet_controller, "handle_pet_list"))
    MessageRouter.register_handler(CommandIds.PET_UPDATE_PUSH, Callable(pet_controller, "handle_pet_update"))
    MessageRouter.register_handler(CommandIds.PET_LINEUP_SET_RESP, Callable(pet_controller, "handle_lineup_set_response"))

    MessageRouter.register_handler(CommandIds.BATTLE_START_PUSH, Callable(battle_controller, "handle_battle_start"))
    MessageRouter.register_handler(CommandIds.BATTLE_STATE_PUSH, Callable(battle_controller, "handle_battle_state"))
    MessageRouter.register_handler(CommandIds.BATTLE_RESULT_PUSH, Callable(battle_controller, "handle_battle_result"))

    MessageRouter.register_handler(CommandIds.BAG_LIST_RESP, Callable(bag_controller, "handle_bag_list"))
    MessageRouter.register_handler(CommandIds.BAG_UPDATE_PUSH, Callable(bag_controller, "handle_bag_update"))

func _connect_signals() -> void:
    demo_login_button.pressed.connect(_on_demo_login_pressed)
    connect_button.pressed.connect(_on_connect_ws_pressed)
    enter_world_button.pressed.connect(_on_enter_world_pressed)

    App.login_succeeded.connect(_on_login_succeeded)
    App.login_failed.connect(_on_login_failed)
    App.notice_received.connect(_on_notice_received)
    App.kicked.connect(_on_kicked)

    GameState.session_changed.connect(_refresh_view)
    GameState.world_snapshot_changed.connect(_refresh_view)
    NetClient.connection_state_changed.connect(_on_connection_state_changed)

func _on_demo_login_pressed() -> void:
    _append_log("POST /api/v1/auth/login using the demo account.")
    var response: Dictionary = await App.login_with_demo_account()
    var code: int = int(response.get("code", 0))
    if code == 200:
        _append_log("Login succeeded. ws_token cached in GameState.")
    else:
        _append_log("Login failed: %s" % str(response.get("msg", "unknown error")))
    _refresh_view()

func _on_connect_ws_pressed() -> void:
    var err := App.connect_ws()
    if err != OK:
        _append_log("WebSocket connect failed: %s" % error_string(err))
        return

    _append_log("WebSocket connecting to ws://127.0.0.1:8080/ws")
    if not GameState.ws_token.is_empty():
        _append_log("Next milestone after open: encode WS_AUTH_REQ in NetClient.authenticate_ws().")

func _on_enter_world_pressed() -> void:
    App.enter_world()
    _append_log("ENTER_WORLD_REQ dispatched through the transport skeleton.")

func _on_login_succeeded(response: Dictionary) -> void:
    var data_variant: Variant = response.get("data", {})
    var data: Dictionary = data_variant if data_variant is Dictionary else {}
    _append_log("Player %s is ready for websocket auth." % str(data.get("player_id", "unknown")))

func _on_login_failed(message: String) -> void:
    _append_log("Login failed: %s" % message)

func _on_notice_received(message: String) -> void:
    _append_log("Notice: %s" % message)

func _on_kicked(reason: String) -> void:
    _append_log("Disconnected by server: %s" % reason)

func _on_connection_state_changed(_state: String) -> void:
    _refresh_view()
    _append_log("WebSocket state -> %s" % NetClient.get_connection_state())

func _on_world_scene_loaded(scene_id: String) -> void:
    _append_log("Scene loaded: %s" % scene_id)
    _refresh_view()

func _on_player_position_changed(position: Vector2) -> void:
    _append_log("Local player anchor moved to (%.0f, %.0f)." % [position.x, position.y])

func _refresh_view() -> void:
    status_label.text = "Connection: %s | HTTP token: %s" % [
        NetClient.get_connection_state(),
        _short_token(GameState.access_jwt),
    ]

    var scene_id := str(GameState.scene_snapshot.get("scene_id", "not entered"))
    scene_label.text = "Scene: %s | Nearby entities: %d" % [scene_id, GameState.nearby_entities.size()]

    var player_name := str(GameState.player_snapshot.get("name", "not logged in"))
    var player_text := "%s" % player_name
    if GameState.player_id > 0:
        player_text += " (#%d)" % GameState.player_id
    player_label.text = "Player: %s" % player_text

func _append_log(message: String) -> void:
    log_output.append_text(message + "\n")

func _short_token(token: String) -> String:
    if token.is_empty():
        return "none"
    if token.length() <= 12:
        return token
    return "%s...%s" % [token.substr(0, 6), token.substr(token.length() - 4, 4)]
