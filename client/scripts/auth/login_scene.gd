extends Node


# 登录成功后要切入的主运行态场景路径。
const MAIN_SCENE_PATH := "res://scenes/bootstrap/main.tscn"
# 登录页默认填充的演示账号名。
const DEMO_ACCOUNT := "demo"
# 登录页默认填充的演示账号密码。
const DEMO_PASSWORD := "demo123"
# 登录场景淡入淡出过渡的持续时间。
const TRANSITION_DURATION := 0.18
const REGISTER_GENDER_MALE: String = "male"
const REGISTER_GENDER_FEMALE: String = "female"

# 账号输入框节点。
@onready var account_input: LineEdit = %AccountInput
# 密码输入框节点。
@onready var password_input: LineEdit = %PasswordInput
# 注册确认密码输入框节点。
@onready var confirm_password_input: LineEdit = %ConfirmPasswordInput
# 注册性别选择节点。
@onready var gender_option_button: OptionButton = %GenderOptionButton
# 触发登录流程的主按钮。
@onready var login_button: Button = %LoginButton
# 触发注册流程的次按钮。
@onready var register_button: Button = %RegisterButton
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
# 登录页内嵌的开发切服面板。
@onready var dev_server_switcher: DevServerSwitcher = %DevServerSwitcher
# 登录切场景使用的全屏过渡遮罩。
@onready var transition_overlay: ColorRect = %TransitionOverlay

# 标记当前是否正在执行登录流程。
var _login_flow_running: bool = false
# 标记当前是否正在切换到主运行态场景。
var _switching_scene: bool = false
# 标记当前是否正在登录页等待世界与地图就绪。
var _world_entry_flow_running: bool = false
# 登录页全流程使用的 loading 遮罩（点击登录后立即展示）。
var _login_loading: GenericLoadingScene = null

# 初始化登录页并绑定登录链路所需信号。
func _ready() -> void:
	App.bootstrap()
	_connect_signals()
	_fill_demo_credentials()
	_play_fade_in()
	_append_log("登录页已就绪。")
	_append_log("点击“登录并进入世界”后会自动完成 HTTP 登录和实时连接。")
	_append_log("新账号可先选择男女形象后注册，注册成功后再登录。")
	_refresh_view()
	if GameState.is_ws_authenticated:
		call_deferred("_prepare_world_entry_and_enter_main")

# 绑定登录页依赖的按钮、输入框、应用信号和全局状态信号。
func _connect_signals() -> void:
	login_button.pressed.connect(_on_login_button_pressed)
	register_button.pressed.connect(_on_register_button_pressed)
	account_input.text_submitted.connect(_on_credentials_submitted)
	password_input.text_submitted.connect(_on_credentials_submitted)
	confirm_password_input.text_submitted.connect(_on_register_credentials_submitted)
	_setup_gender_options()

	App.login_succeeded.connect(_on_login_succeeded)
	App.login_failed.connect(_on_login_failed)
	App.session_authenticated.connect(_on_session_authenticated)
	App.notice_received.connect(_on_notice_received)
	App.kicked.connect(_on_kicked)
	App.server_result_logged.connect(_on_server_result_logged)

	GameState.session_changed.connect(_refresh_view)
	GameState.world_snapshot_changed.connect(_refresh_view)
	NetClient.connection_state_changed.connect(_on_connection_state_changed)
	NetClient.websocket_closed.connect(_on_websocket_closed)
	dev_server_switcher.config_applied.connect(_on_dev_server_config_applied)
	dev_server_switcher.config_cleared.connect(_on_dev_server_config_cleared)


# 登录页销毁时断开全局信号，避免返回登录页多次后重复打印同一条服务端日志。
func _exit_tree() -> void:
	if App.login_succeeded.is_connected(_on_login_succeeded):
		App.login_succeeded.disconnect(_on_login_succeeded)
	if App.login_failed.is_connected(_on_login_failed):
		App.login_failed.disconnect(_on_login_failed)
	if App.session_authenticated.is_connected(_on_session_authenticated):
		App.session_authenticated.disconnect(_on_session_authenticated)
	if App.notice_received.is_connected(_on_notice_received):
		App.notice_received.disconnect(_on_notice_received)
	if App.kicked.is_connected(_on_kicked):
		App.kicked.disconnect(_on_kicked)
	if App.server_result_logged.is_connected(_on_server_result_logged):
		App.server_result_logged.disconnect(_on_server_result_logged)
	if GameState.session_changed.is_connected(_refresh_view):
		GameState.session_changed.disconnect(_refresh_view)
	if GameState.world_snapshot_changed.is_connected(_refresh_view):
		GameState.world_snapshot_changed.disconnect(_refresh_view)
	if NetClient.connection_state_changed.is_connected(_on_connection_state_changed):
		NetClient.connection_state_changed.disconnect(_on_connection_state_changed)
	if NetClient.websocket_closed.is_connected(_on_websocket_closed):
		NetClient.websocket_closed.disconnect(_on_websocket_closed)
	if dev_server_switcher != null and dev_server_switcher.config_applied.is_connected(_on_dev_server_config_applied):
		dev_server_switcher.config_applied.disconnect(_on_dev_server_config_applied)
	if dev_server_switcher != null and dev_server_switcher.config_cleared.is_connected(_on_dev_server_config_cleared):
		dev_server_switcher.config_cleared.disconnect(_on_dev_server_config_cleared)

# 在输入框为空时自动填充演示账号和密码。
func _fill_demo_credentials() -> void:
	if account_input.text.is_empty():
		account_input.text = DEMO_ACCOUNT
	if password_input.text.is_empty():
		password_input.text = DEMO_PASSWORD
	if confirm_password_input.text.is_empty():
		confirm_password_input.text = DEMO_PASSWORD

# 初始化注册性别下拉选项，默认选中男性初始形象。
func _setup_gender_options() -> void:
	gender_option_button.clear()
	gender_option_button.add_item("男 - 初始形象男", 0)
	gender_option_button.set_item_metadata(0, REGISTER_GENDER_MALE)
	gender_option_button.add_item("女 - 初始形象女", 1)
	gender_option_button.set_item_metadata(1, REGISTER_GENDER_FEMALE)
	gender_option_button.selected = 0

# 处理输入框回车提交事件，并复用主登录入口。
func _on_credentials_submitted(_value: String) -> void:
	_on_login_button_pressed()

# 注册确认密码框回车时，直接复用注册入口。
func _on_register_credentials_submitted(_value: String) -> void:
	_on_register_button_pressed()

# 处理登录按钮点击事件，串行执行 HTTP 登录和实时连接建立。
func _on_login_button_pressed() -> void:
	if _login_flow_running:
		return

	# 读取并裁剪账号输入内容。
	var account: String = account_input.text.strip_edges()
	# 读取并裁剪密码输入内容。
	var password: String = password_input.text.strip_edges()
	if account.is_empty() or password.is_empty():
		_append_log("请输入账号和密码。")
		return

	await _start_login_flow(account, password)

# 处理注册按钮点击事件，创建账号与初始角色后自动进入登录流程。
func _on_register_button_pressed() -> void:
	if _login_flow_running:
		return

	var account: String = account_input.text.strip_edges()
	var password: String = password_input.text.strip_edges()
	var confirm_password: String = confirm_password_input.text.strip_edges()
	if account.is_empty() or password.is_empty():
		_append_log("注册前请输入账号和密码。")
		return
	if confirm_password.is_empty():
		_append_log("注册前请输入确认密码。")
		return
	if password != confirm_password:
		_append_log("两次输入的密码不一致，请重新确认。")
		return

	var selected_gender: String = _selected_register_gender()
	_set_login_busy(true)
	_show_login_loading()
	_append_log("开始注册账号 %s，形象=%s。" % [account, "男" if selected_gender == REGISTER_GENDER_MALE else "女"])

	var response: Dictionary = await App.register_account(account, password, selected_gender)

	var code: int = int(response.get("code", 0))
	if code != 200:
		_set_login_busy(false)
		var error_message: String = str(response.get("msg", "register failed"))
		if error_message == "account already exists":
			error_message = "账号已存在，请更换后重试。"
		elif error_message == "account and password are required":
			error_message = "账号和密码不能为空。"
		_append_log("注册失败: %s" % error_message)
		return

	var data_variant: Variant = response.get("data", {})
	var data: Dictionary = data_variant if data_variant is Dictionary else {}
	_append_log("注册成功，角色 %s 已创建，正在自动登录。" % str(data.get("player_name", account)))
	await _start_login_flow(account, password, false)

# 记录 HTTP 登录成功后的角色与令牌获取结果。
func _on_login_succeeded(response: Dictionary) -> void:
	var data_variant: Variant = response.get("data", {})
	# 规范化登录结果数据体为字典结构。
	var data: Dictionary = data_variant if data_variant is Dictionary else {}
	_append_log("HTTP 登录成功，角色 %s 已获取会话令牌。" % str(data.get("player_id", "unknown")))

# 记录 HTTP 登录失败原因。
func _on_login_failed(message: String) -> void:
	_append_log("登录失败: %s" % message)


# 统一展示服务端请求结果，方便在登录页直接看到 HTTP 登录和首次 WS 回包。
func _on_server_result_logged(message: String) -> void:
	_append_log("服务端结果: %s" % message)

# 处理实时连接鉴权成功事件，并在登录页等待世界与地图就绪后再切主场景。
func _on_session_authenticated(_payload: Dictionary) -> void:
	_append_log("实时连接鉴权成功，正在加载世界数据。")
	await _prepare_world_entry_and_enter_main()

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
	status_label.text = UiFormat.normalize_text(status_label.text)

	# 读取当前世界快照中的场景标识。
	var scene_id := str(GameState.scene_snapshot.get("scene_id", "未进入"))
	scene_label.text = "场景: %s | 附近实体: %d" % [scene_id, GameState.nearby_entities.size()]
	scene_label.text = UiFormat.normalize_text(scene_label.text)

	# 读取当前玩家名称用于拼接角色文案。
	var player_name := str(GameState.player_snapshot.get("name", "未登录"))
	# 生成登录页顶部显示的玩家摘要文本。
	var player_text := "%s" % player_name
	if GameState.player_id > 0:
		player_text += " (#%d)" % GameState.player_id
	player_label.text = "玩家: %s" % player_text
	player_label.text = UiFormat.normalize_text(player_label.text)

	hint_label.text = "演示账号: %s  演示密码: %s" % [DEMO_ACCOUNT, DEMO_PASSWORD]
	if dev_server_switcher != null:
		dev_server_switcher.refresh_from_active_config()

	login_button.text = "登录中..." if _login_flow_running else "登录并进入世界"
	login_button.disabled = _login_flow_running
	register_button.disabled = _login_flow_running
	account_input.editable = not _login_flow_running
	password_input.editable = not _login_flow_running
	confirm_password_input.editable = not _login_flow_running
	gender_option_button.disabled = _login_flow_running

# 切换登录流程忙碌状态，并同步刷新界面。
func _set_login_busy(busy: bool) -> void:
	_login_flow_running = busy
	if not busy and not _world_entry_flow_running:
		_hide_login_loading()
	_refresh_view()

# 向登录页日志区域追加一条文本。
func _append_log(message: String) -> void:
	log_output.append_text(UiFormat.normalize_text(message) + "\n")

# 返回当前注册表单选中的规范性别值。
func _selected_register_gender() -> String:
	var selected_index: int = gender_option_button.selected
	if selected_index < 0:
		return REGISTER_GENDER_MALE
	var metadata: Variant = gender_option_button.get_item_metadata(selected_index)
	var gender: String = str(metadata).strip_edges()
	if gender.is_empty():
		return REGISTER_GENDER_MALE
	return gender

# 统一封装登录主流程，供登录按钮与注册成功后的自动登录复用。
func _start_login_flow(account: String, password: String, reset_session: bool = true) -> void:
	if reset_session:
		GameState.reset_session_state()
		NetClient.disconnect_from_server()
		_set_login_busy(true)
		_show_login_loading()
	else:
		GameState.reset_session_state()
		NetClient.disconnect_from_server()

	_append_log("开始登录账号 %s。" % account)

	# 等待应用层完成 HTTP 登录请求。
	var response: Dictionary = await App.login(account, password)
	# 读取 HTTP 登录返回的状态码。
	var code: int = int(response.get("code", 0))
	if code != 200:
		_set_login_busy(false)
		return

	_append_log("HTTP 登录成功，开始建立实时连接。")
	_show_login_loading()
	# 发起 WebSocket 连接并等待后续自动鉴权。
	var err: int = App.connect_ws()
	if err != OK:
		_append_log("WebSocket 连接失败: %s" % error_string(err))
		_set_login_busy(false)
		return
	_refresh_view()

# 懒创建登录页 loading 遮罩。
func _ensure_login_loading() -> GenericLoadingScene:
	if _login_loading == null:
		var loading_scene: PackedScene = preload(GenericLoadingScene.SCENE_PATH)
		_login_loading = loading_scene.instantiate() as GenericLoadingScene
		if _login_loading != null:
			add_child(_login_loading)
	return _login_loading


# 立即展示登录页 loading，并拦截点击。
func _show_login_loading() -> void:
	var loading_overlay: GenericLoadingScene = _ensure_login_loading()
	if loading_overlay != null:
		loading_overlay.show_loading()
	transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP


# 隐藏登录页 loading。
func _hide_login_loading() -> void:
	if _login_loading != null:
		_login_loading.hide_loading()
	if not _world_entry_flow_running and not _switching_scene:
		transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

# 登录页完成 ENTER_WORLD 与地图预加载后再进入主场景，避免先看到空白星空背景。
func _prepare_world_entry_and_enter_main() -> void:
	if _world_entry_flow_running or _switching_scene:
		return
	_world_entry_flow_running = true
	_set_login_busy(true)
	_show_login_loading()

	var request_seq: int = App.enter_world()
	if request_seq <= 0:
		_hide_login_loading()
		_world_entry_flow_running = false
		_set_login_busy(false)
		_append_log("进入世界请求发送失败，请稍后重试。")
		return

	var wait_result: Dictionary = await _wait_app_request(CommandIds.ENTER_WORLD_REQ, request_seq)
	if not bool(wait_result.get("succeeded", false)):
		_hide_login_loading()
		_world_entry_flow_running = false
		_set_login_busy(false)
		_append_log("进入世界失败，请稍后重试。")
		return

	var payload_variant: Variant = wait_result.get("payload", {})
	var payload: Dictionary = payload_variant if payload_variant is Dictionary else {}
	GameState.set_world_snapshot(payload)

	var scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	if scene_id <= 0:
		_hide_login_loading()
		_world_entry_flow_running = false
		_set_login_busy(false)
		_append_log("服务端未返回有效场景信息，暂时无法进入世界。")
		return

	if not WorldSceneRegistry.can_load_scene_map(scene_id):
		_hide_login_loading()
		_world_entry_flow_running = false
		_set_login_busy(false)
		_append_log("场景 %d 的地图资源不可用，请联系管理员检查配置。" % scene_id)
		return

	GameState.world_entry_prepared = true
	_append_log("世界与地图已就绪，正在进入主场景。")
	await _enter_main_scene()
	_world_entry_flow_running = false

# 等待指定 App 请求完成并返回是否成功与载荷。
func _wait_app_request(expected_cmd: int, expected_seq: int) -> Dictionary:
	while expected_seq > 0:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != expected_cmd or seq != expected_seq:
			continue
		return {
			"succeeded": bool(result[2]),
			"response_cmd": int(result[3]),
			"payload": result[4],
		}
	return {
		"succeeded": false,
		"response_cmd": 0,
		"payload": {},
	}

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

## 应用新的开发切服配置后，刷新网络入口并清理旧会话，避免把旧服 token 带到新服。
func _on_dev_server_config_applied(profile_name: String, http_base: String, ws_url: String) -> void:
	HttpClient.reload_base_url_from_config()
	NetClient.disconnect_from_server()
	NetClient.reload_default_ws_url_from_config()
	GameState.reset_session_state()
	_set_login_busy(false)
	_append_log("已应用切服配置：%s | HTTP=%s | WS=%s" % [profile_name, http_base, ws_url])
	_refresh_view()

## 清空切服覆盖后，同样刷新当前网络入口并回到默认环境解析。
func _on_dev_server_config_cleared(profile_name: String, http_base: String, ws_url: String) -> void:
	HttpClient.reload_base_url_from_config()
	NetClient.disconnect_from_server()
	NetClient.reload_default_ws_url_from_config()
	GameState.reset_session_state()
	_set_login_busy(false)
	_append_log("已清空切服覆盖：%s | HTTP=%s | WS=%s" % [profile_name, http_base, ws_url])
	_refresh_view()
