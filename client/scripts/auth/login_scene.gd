extends Node

# 登录成功后要切入的主运行态场景路径。
const MAIN_SCENE_PATH := "res://scenes/bootstrap/main.tscn"
# 登录页默认填充的演示账号名。
const DEMO_ACCOUNT := "demo"
# 登录页默认填充的演示账号密码。
const DEMO_PASSWORD := "demo123"
# 登录场景淡入淡出过渡的持续时间。
const TRANSITION_DURATION := 0.18

# 账号输入框节点。
@onready var account_input: LineEdit = %AccountInput
# 密码输入框节点。
@onready var password_input: LineEdit = %PasswordInput
# 触发登录流程的主按钮。
@onready var login_button: Button = %LoginButton
# 顶部提示文案标签。
@onready var hint_label: Label = %HintLabel
# 连接状态摘要标签。
@onready var status_label: Label = %StatusLabel
# 当前场景摘要标签。
@onready var scene_label: Label = %SceneLabel
# 当前玩家摘要标签。
@onready var player_label: Label = %PlayerLabel
# 登录页日志输出区域。
@onready var log_output: RichTextLabel = %LogOutput
# 登录切场景使用的全屏过渡遮罩。
@onready var transition_overlay: ColorRect = %TransitionOverlay

# 标记当前是否正在执行登录流程。
var _login_flow_running: bool = false
# 标记当前是否正在切换到主运行态场景。
var _switching_scene: bool = false

# 初始化登录页并绑定登录链路所需信号。
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

# 绑定登录页依赖的按钮、输入框、应用信号和全局状态信号。
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

# 在输入框为空时自动填充演示账号和密码。
func _fill_demo_credentials() -> void:
	if account_input.text.is_empty():
		account_input.text = DEMO_ACCOUNT
	if password_input.text.is_empty():
		password_input.text = DEMO_PASSWORD

# 处理输入框回车提交事件，并复用主登录入口。
func _on_credentials_submitted(_value: String) -> void:
	_on_login_button_pressed()

# 处理登录按钮点击事件，串行执行 HTTP 登录和实时连接建立。
func _on_login_button_pressed() -> void:
	if _login_flow_running:
		return

	# 读取并裁剪账号输入内容。
	var account := account_input.text.strip_edges()
	# 读取并裁剪密码输入内容。
	var password := password_input.text.strip_edges()
	if account.is_empty() or password.is_empty():
		_append_log("请输入账号和密码。")
		return

	GameState.reset_session_state()
	NetClient.disconnect_from_server()
	_set_login_busy(true)
	_append_log("开始登录账号 %s。" % account)

	# 等待应用层完成 HTTP 登录请求。
	var response: Dictionary = await App.login(account, password)
	# 读取 HTTP 登录返回的状态码。
	var code: int = int(response.get("code", 0))
	if code != 200:
		_set_login_busy(false)
		return

	_append_log("HTTP 登录成功，开始建立实时连接。")
	# 发起 WebSocket 连接并等待后续自动鉴权。
	var err := App.connect_ws()
	if err != OK:
		_append_log("WebSocket 连接失败: %s" % error_string(err))
		_set_login_busy(false)
		return
	_refresh_view()

# 记录 HTTP 登录成功后的角色与令牌获取结果。
func _on_login_succeeded(response: Dictionary) -> void:
	var data_variant: Variant = response.get("data", {})
	# 规范化登录结果数据体为字典结构。
	var data: Dictionary = data_variant if data_variant is Dictionary else {}
	_append_log("HTTP 登录成功，角色 %s 已获取会话令牌。" % str(data.get("player_id", "unknown")))

# 记录 HTTP 登录失败原因。
func _on_login_failed(message: String) -> void:
	_append_log("登录失败: %s" % message)

# 处理实时连接鉴权成功事件，并切换到主运行态场景。
func _on_session_authenticated(_payload: Dictionary) -> void:
	_append_log("实时连接鉴权成功，正在进入主场景。")
	_enter_main_scene()

# 处理应用层普通提示消息，并在必要时结束登录忙碌态。
func _on_notice_received(message: String) -> void:
	_append_log("提示: %s" % message)
	if _login_flow_running and not GameState.is_ws_authenticated:
		_set_login_busy(false)

# 处理被服务端踢下线事件。
func _on_kicked(reason: String) -> void:
	_append_log("连接已被服务端断开: %s" % reason)
	_set_login_busy(false)

# 处理网络连接状态变化，并在错误态时恢复登录按钮。
func _on_connection_state_changed(state: String) -> void:
	_refresh_view()
	_append_log("WebSocket 状态 -> %s" % state)
	if state == "error":
		_set_login_busy(false)

# 处理底层 WebSocket 关闭事件，并在登录中断时恢复可输入状态。
func _on_websocket_closed(code: int, reason: String) -> void:
	if _login_flow_running and not GameState.is_ws_authenticated:
		_set_login_busy(false)
	if code == -1 and reason.is_empty():
		return
	_append_log("WebSocket 已关闭: %d %s" % [code, reason])

# 按当前会话状态刷新登录页文案和输入控件状态。
func _refresh_view() -> void:
	status_label.text = "连接状态: %s | HTTP: %s | WS: %s" % [
		NetClient.get_connection_state(),
		_short_token(GameState.access_jwt),
		"ok" if GameState.is_ws_authenticated else "pending",
	]

	# 读取当前世界快照中的场景标识。
	var scene_id := str(GameState.scene_snapshot.get("scene_id", "未进入"))
	scene_label.text = "场景: %s | 附近实体: %d" % [scene_id, GameState.nearby_entities.size()]

	# 读取当前玩家名称用于拼接角色文案。
	var player_name := str(GameState.player_snapshot.get("name", "未登录"))
	# 生成登录页顶部显示的玩家摘要文本。
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

# 切换登录流程忙碌状态，并同步刷新界面。
func _set_login_busy(busy: bool) -> void:
	_login_flow_running = busy
	_refresh_view()

# 向登录页日志区域追加一条文本。
func _append_log(message: String) -> void:
	log_output.append_text(message + "\n")

# 切换到主运行态场景，并在切换前播放淡出动画。
func _enter_main_scene() -> void:
	if _switching_scene:
		return
	_switching_scene = true
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(1.0)
	get_tree().change_scene_to_file(MAIN_SCENE_PATH)

# 播放登录页进入时的淡入动画。
func _play_fade_in() -> void:
	transition_overlay.color.a = 1.0
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
	await _fade_overlay(0.0)
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

# 把过渡遮罩透明度补间到目标值。
func _fade_overlay(target_alpha: float) -> void:
	# 创建当前使用的过渡补间动画。
	var tween := create_tween()
	tween.tween_property(transition_overlay, "color:a", target_alpha, TRANSITION_DURATION)
	await tween.finished

# 对令牌做简化显示，避免完整内容直接出现在登录页。
func _short_token(token: String) -> String:
	if token.is_empty():
		return "none"
	if token.length() <= 12:
		return token
	return "%s...%s" % [token.substr(0, 6), token.substr(token.length() - 4, 4)]
