extends Node

const MAIN_SCENE_PATH := "res://scenes/bootstrap/main.tscn"
const DEMO_ACCOUNT := "demo"
const DEMO_PASSWORD := "demo123"
const TRANSITION_DURATION := 0.18

@onready var account_input: LineEdit = %AccountInput
@onready var password_input: LineEdit = %PasswordInput
@onready var login_button: Button = %LoginButton
@onready var hint_label: Label = %HintLabel
@onready var status_label: Label = %StatusLabel
@onready var scene_label: Label = %SceneLabel
@onready var player_label: Label = %PlayerLabel
@onready var log_output: RichTextLabel = %LogOutput
@onready var transition_overlay: ColorRect = %TransitionOverlay

var _login_flow_running: bool = false
var _switching_scene: bool = false

func _ready() -> void:
	App.bootstrap()
	_connect_signals()
	_fill_demo_credentials()
	_play_fade_in()
	_append_log("登录页已就绪。")
	_append_log("点击“登录并进入世界”后会自动完成 HTTP 登录和实时连接。")
	_refresh_view()
	if GameState.is_ws_authenticated:
		call_deferred("_enter_main_scene")

func _connect_signals() -> void:
	login_button.pressed.connect(_on_login_button_pressed)
	account_input.text_submitted.connect(_on_credentials_submitted)
	password_input.text_submitted.connect(_on_credentials_submitted)

	App.login_succeeded.connect(_on_login_succeeded)
	App.login_failed.connect(_on_login_failed)
	App.session_authenticated.connect(_on_session_authenticated)
	App.notice_received.connect(_on_notice_received)
	App.kicked.connect(_on_kicked)

	GameState.session_changed.connect(_refresh_view)
	GameState.world_snapshot_changed.connect(_refresh_view)
	NetClient.connection_state_changed.connect(_on_connection_state_changed)
	NetClient.websocket_closed.connect(_on_websocket_closed)

func _fill_demo_credentials() -> void:
	if account_input.text.is_empty():
		account_input.text = DEMO_ACCOUNT
	if password_input.text.is_empty():
		password_input.text = DEMO_PASSWORD

func _on_credentials_submitted(_value: String) -> void:
	_on_login_button_pressed()

func _on_login_button_pressed() -> void:
	if _login_flow_running:
		return

	var account := account_input.text.strip_edges()
	var password := password_input.text.strip_edges()
	if account.is_empty() or password.is_empty():
		_append_log("请输入账号和密码。")
		return

	GameState.reset_session_state()
	NetClient.disconnect_from_server()
	_set_login_busy(true)
	_append_log("开始登录账号 %s。" % account)

	var response: Dictionary = await App.login(account, password)
	var code: int = int(response.get("code", 0))
	if code != 200:
		_set_login_busy(false)
		return

	_append_log("HTTP 登录成功，开始建立实时连接。")
	var err := App.connect_ws()
	if err != OK:
		_append_log("WebSocket 连接失败: %s" % error_string(err))
		_set_login_busy(false)
		return
	_refresh_view()

func _on_login_succeeded(response: Dictionary) -> void:
	var data_variant: Variant = response.get("data", {})
	var data: Dictionary = data_variant if data_variant is Dictionary else {}
	_append_log("HTTP 登录成功，角色 %s 已获取会话令牌。" % str(data.get("player_id", "unknown")))

func _on_login_failed(message: String) -> void:
	_append_log("登录失败: %s" % message)

func _on_session_authenticated(_payload: Dictionary) -> void:
	_append_log("实时连接鉴权成功，正在进入主场景。")
	_enter_main_scene()

func _on_notice_received(message: String) -> void:
	_append_log("提示: %s" % message)
	if _login_flow_running and not GameState.is_ws_authenticated:
		_set_login_busy(false)

func _on_kicked(reason: String) -> void:
	_append_log("连接已被服务端断开: %s" % reason)
	_set_login_busy(false)

func _on_connection_state_changed(state: String) -> void:
	_refresh_view()
	_append_log("WebSocket 状态 -> %s" % state)
	if state == "error":
		_set_login_busy(false)

func _on_websocket_closed(code: int, reason: String) -> void:
	if _login_flow_running and not GameState.is_ws_authenticated:
		_set_login_busy(false)
	if code == -1 and reason.is_empty():
		return
	_append_log("WebSocket 已关闭: %d %s" % [code, reason])

func _refresh_view() -> void:
	status_label.text = "连接状态: %s | HTTP: %s | WS: %s" % [
		NetClient.get_connection_state(),
		_short_token(GameState.access_jwt),
		"ok" if GameState.is_ws_authenticated else "pending",
	]

	var scene_id := str(GameState.scene_snapshot.get("scene_id", "未进入"))
	scene_label.text = "场景: %s | 附近实体: %d" % [scene_id, GameState.nearby_entities.size()]

	var player_name := str(GameState.player_snapshot.get("name", "未登录"))
	var player_text := "%s" % player_name
	if GameState.player_id > 0:
		player_text += " (#%d)" % GameState.player_id
	player_label.text = "玩家: %s" % player_text

	hint_label.text = "演示账号: %s  演示密码: %s" % [DEMO_ACCOUNT, DEMO_PASSWORD]
	if GameState.is_ws_authenticated:
		hint_label.text = "实时会话已建立，正在切换主场景。"

	login_button.text = "登录中..." if _login_flow_running else "登录并进入世界"
	login_button.disabled = _login_flow_running
	account_input.editable = not _login_flow_running
	password_input.editable = not _login_flow_running

func _set_login_busy(busy: bool) -> void:
	_login_flow_running = busy
	_refresh_view()

func _append_log(message: String) -> void:
	log_output.append_text(message + "\n")

func _enter_main_scene() -> void:
	if _switching_scene:
		return
	_switching_scene = true
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(1.0)
	get_tree().change_scene_to_file(MAIN_SCENE_PATH)

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
