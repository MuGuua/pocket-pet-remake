extends CanvasLayer
class_name NewLoginPanel

## 玩家在账号密码弹窗中点击确认后触发；上层登录场景可接入真正登录流程。
signal login_credentials_submitted(account: String, password: String)
## 玩家在注册模式下点击“注册并登录”后触发；上层登录场景可先注册再登录。
signal register_credentials_submitted(account: String, password: String)

## 箭头从当前选项移动到目标选项的补间时长。
const ARROW_MOVE_DURATION: float = 0.18
## 箭头与按钮左侧之间保留的横向间距，避免贴住按钮边缘。
const ARROW_BUTTON_GAP: float = 4.0
## 没有找到 ArrowIndicator 初始位置时，箭头放在按钮左侧的兜底距离。
const FALLBACK_ARROW_LEFT_OFFSET: float = 22.0
## “登录游戏”面板绑定到第二个主菜单按钮。
const LOGIN_GAME_OPTION_INDEX: int = 1

## 登录选项所在的全屏内容根节点。
@onready var _content_root: Control = get_node_or_null("Control") as Control
## 登录选项按钮纵向容器；优先使用整理后的 OptionsVBox，兼容旧 VBoxContainer 名称。
@onready var _option_container: VBoxContainer = _resolve_option_container()
## 菜单当前选项指示箭头；优先使用整理后的 ArrowIndicator，兼容旧按钮内 TextureRect。
@onready var _arrow: TextureRect = _resolve_arrow_indicator()
## 账号密码弹窗遮罩层，来自场景真实节点，方便在编辑器里调整。
@onready var _credentials_backdrop: ColorRect = %CredentialsBackdrop
## 账号密码弹窗标题；登录/注册模式会切换文案。
@onready var _credentials_title_label: Label = %CredentialsTitle
## 账号输入框，来自场景真实节点。
@onready var _account_input: LineEdit = %AccountInput
## 密码输入框，来自场景真实节点。
@onready var _password_input: LineEdit = %PasswordInput
## 确认密码标签，只有注册模式展示。
@onready var _confirm_password_label: Label = %ConfirmPassword
## 确认密码输入框，只有注册模式展示。
@onready var _confirm_password_input: LineEdit = %ConfirmPasswordInput
## 弹窗底部提示文本，来自场景真实节点。
@onready var _credentials_hint_label: Label = %HintLabel
## 弹窗内注册模式切换按钮。
@onready var _register_button: Button = %RegisterButton
## 账号密码弹窗确认按钮。
@onready var _credentials_confirm_button: Button = %ConfirmButton
## 账号密码弹窗取消按钮。
@onready var _credentials_cancel_button: Button = %CancelButton

## 当前负责平滑移动箭头的补间实例。
var _arrow_tween: Tween = null
## 箭头固定使用的全局横坐标，切换选项时只跟随按钮纵向移动。
var _arrow_global_x: float = 0.0
## 已绑定交互事件的按钮列表。
var _option_buttons: Array[Button] = []
## 当前由箭头指向的按钮索引。
var _selected_index: int = 0
## 账号密码弹窗当前是否为注册模式。
var _credentials_register_mode: bool = false


## 初始化选项按钮事件、清理按钮视觉样式，并将箭头对齐到第一个选项。
func _ready() -> void:
	await get_tree().process_frame
	_collect_option_buttons()
	_prepare_arrow_node()
	_bind_credentials_popup()
	_select_option(0, true)


## 捕获上下键与确认键：上下移动箭头，确认触发当前按钮 pressed。
func _unhandled_input(event: InputEvent) -> void:
	if _is_credentials_popup_visible():
		if event.is_action_pressed("ui_cancel"):
			_hide_credentials_popup()
			get_viewport().set_input_as_handled()
		return
	if not visible or _option_buttons.is_empty():
		return
	if event.is_action_pressed("ui_up"):
		_select_option(_selected_index - 1, false)
		get_viewport().set_input_as_handled()
		return
	if event.is_action_pressed("ui_down"):
		_select_option(_selected_index + 1, false)
		get_viewport().set_input_as_handled()
		return
	if event.is_action_pressed("ui_accept"):
		_confirm_selected_option()
		get_viewport().set_input_as_handled()
		return


## 解析整理后的选项容器；如果场景还没重命名，则退回旧路径。
func _resolve_option_container() -> VBoxContainer:
	var named_node: Node = find_child("OptionsVBox", true, false)
	if named_node is VBoxContainer:
		return named_node as VBoxContainer
	return get_node_or_null("Control/MarginContainer/VBoxContainer") as VBoxContainer


## 解析整理后的箭头节点；如果场景还没重命名，则退回旧按钮内箭头。
func _resolve_arrow_indicator() -> TextureRect:
	var named_node: Node = find_child("ArrowIndicator", true, false)
	if named_node is TextureRect:
		return named_node as TextureRect
	return get_node_or_null("Control/MarginContainer/VBoxContainer/Button/TextureRect") as TextureRect


## 收集当前选项容器下的所有 Button，并去掉按钮自身视觉状态，只保留箭头表示当前选中项。
func _collect_option_buttons() -> void:
	_option_buttons.clear()
	if _option_container == null:
		return
	var option_index: int = 0
	for child: Node in _option_container.get_children():
		if child is not Button:
			continue
		var option_button: Button = child as Button
		_strip_button_visual_style(option_button)
		_option_buttons.append(option_button)
		if not option_button.mouse_entered.is_connected(_on_option_hovered.bind(option_index)):
			option_button.mouse_entered.connect(_on_option_hovered.bind(option_index))
		if not option_button.button_down.is_connected(_on_option_pressed.bind(option_index)):
			option_button.button_down.connect(_on_option_pressed.bind(option_index))
		if not option_button.pressed.is_connected(_on_option_confirmed.bind(option_index)):
			option_button.pressed.connect(_on_option_confirmed.bind(option_index))
		option_index += 1


## 清理按钮默认边框、焦点框和鼠标指针样式，让按钮只负责点击热区。
func _strip_button_visual_style(option_button: Button) -> void:
	option_button.flat = true
	option_button.focus_mode = Control.FOCUS_NONE
	option_button.mouse_default_cursor_shape = Control.CURSOR_POINTING_HAND
	option_button.remove_theme_stylebox_override("normal")
	option_button.remove_theme_stylebox_override("hover")
	option_button.remove_theme_stylebox_override("pressed")
	option_button.remove_theme_stylebox_override("disabled")
	option_button.remove_theme_stylebox_override("focus")
	option_button.remove_theme_color_override("font_color")
	option_button.remove_theme_color_override("font_hover_color")
	option_button.remove_theme_color_override("font_pressed_color")
	option_button.remove_theme_color_override("font_focus_color")


## 准备箭头节点；如果它还在第一个按钮里面，则运行时挪到内容根节点下，方便跨按钮移动。
func _prepare_arrow_node() -> void:
	if _arrow == null or _content_root == null:
		return
	var original_global_position: Vector2 = _arrow.global_position
	var original_size: Vector2 = _arrow.size
	var old_parent: Node = _arrow.get_parent()
	if old_parent != null and old_parent != _content_root:
		old_parent.remove_child(_arrow)
		_content_root.add_child(_arrow)
		_arrow.set_anchors_preset(Control.PRESET_TOP_LEFT, false)
		_arrow.size = original_size
		_arrow.global_position = original_global_position
	_arrow.name = "ArrowIndicator"
	_arrow.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_arrow.z_index = 20
	_arrow_global_x = _arrow.global_position.x


## 鼠标悬停到某一项时，将箭头平滑移动过去。
func _on_option_hovered(option_index: int) -> void:
	_select_option(option_index, false)


## 移动端/鼠标按下时先更新箭头位置，随后按钮自身 pressed 信号继续走原有点击逻辑。
func _on_option_pressed(option_index: int) -> void:
	_select_option(option_index, false)


## 处理选项确认；当前只给第一个“登录游戏”打开账号密码弹窗。
func _on_option_confirmed(option_index: int) -> void:
	_select_option(option_index, false)
	if option_index == LOGIN_GAME_OPTION_INDEX:
		_show_credentials_popup()


## 切换当前选项索引，并驱动箭头移动；索引会在首尾循环。
func _select_option(next_index: int, immediate: bool) -> void:
	if _option_buttons.is_empty():
		return
	var normalized_index: int = posmod(next_index, _option_buttons.size())
	_selected_index = normalized_index
	_move_arrow_to_button(_option_buttons[_selected_index], immediate)


## 回车/确认键触发当前选中按钮的 pressed 信号，复用按钮已有业务点击逻辑。
func _confirm_selected_option() -> void:
	if _selected_index < 0 or _selected_index >= _option_buttons.size():
		return
	var option_button: Button = _option_buttons[_selected_index]
	if option_button == null or option_button.disabled:
		return
	option_button.pressed.emit()


## 绑定场景里真实存在的账号密码弹窗节点，脚本只处理交互，不再运行时创建 UI。
func _bind_credentials_popup() -> void:
	if _credentials_backdrop != null:
		_credentials_backdrop.hide()
		if not _credentials_backdrop.gui_input.is_connected(_on_credentials_backdrop_gui_input):
			_credentials_backdrop.gui_input.connect(_on_credentials_backdrop_gui_input)
	if _account_input != null and not _account_input.text_submitted.is_connected(_on_credentials_text_submitted):
		_account_input.text_submitted.connect(_on_credentials_text_submitted)
	if _password_input != null and not _password_input.text_submitted.is_connected(_on_credentials_text_submitted):
		_password_input.text_submitted.connect(_on_credentials_text_submitted)
	if _confirm_password_input != null and not _confirm_password_input.text_submitted.is_connected(_on_credentials_text_submitted):
		_confirm_password_input.text_submitted.connect(_on_credentials_text_submitted)
	if _register_button != null and not _register_button.pressed.is_connected(_on_register_mode_requested):
		_register_button.pressed.connect(_on_register_mode_requested)
	if _credentials_confirm_button != null and not _credentials_confirm_button.pressed.is_connected(_on_credentials_confirmed):
		_credentials_confirm_button.pressed.connect(_on_credentials_confirmed)
	if _credentials_cancel_button != null and not _credentials_cancel_button.pressed.is_connected(_hide_credentials_popup):
		_credentials_cancel_button.pressed.connect(_hide_credentials_popup)
	_set_credentials_register_mode(false)


## 打开账号密码弹窗，并把输入焦点放到账号框。
func _show_credentials_popup() -> void:
	if _credentials_backdrop == null:
		return
	_set_credentials_register_mode(false)
	if _credentials_hint_label != null:
		_credentials_hint_label.text = ""
	_credentials_backdrop.show()
	await get_tree().process_frame
	if _account_input != null:
		_account_input.grab_focus()


## 隐藏账号密码弹窗；不清空输入，方便玩家输错后再次打开修改。
func _hide_credentials_popup() -> void:
	if _credentials_backdrop != null:
		_credentials_backdrop.hide()


## 判断账号密码弹窗是否正在显示，用于阻断底层菜单的上下键和确认键。
func _is_credentials_popup_visible() -> bool:
	return _credentials_backdrop != null and _credentials_backdrop.visible


## 输入框回车提交时复用确认按钮逻辑。
func _on_credentials_text_submitted(_value: String) -> void:
	_on_credentials_confirmed()


## 点击弹窗里的“注册/登录”按钮后，在登录与注册表单之间来回切换。
func _on_register_mode_requested() -> void:
	_set_credentials_register_mode(not _credentials_register_mode)
	if _credentials_register_mode and _confirm_password_input != null:
		_confirm_password_input.grab_focus()
	elif _account_input != null:
		_account_input.grab_focus()


## 切换登录/注册模式，只改变真实场景节点的显隐与按钮文案。
func _set_credentials_register_mode(enabled: bool) -> void:
	_credentials_register_mode = enabled
	if _credentials_title_label != null:
		_credentials_title_label.text = "注册账号" if _credentials_register_mode else "登录游戏"
	if _confirm_password_label != null:
		_confirm_password_label.visible = _credentials_register_mode
	if _confirm_password_input != null:
		_confirm_password_input.visible = _credentials_register_mode
		if not _credentials_register_mode:
			_confirm_password_input.text = ""
	if _register_button != null:
		var switch_label: String = "登录" if _credentials_register_mode else "注册"
		if _register_button.has_method("set_button_label"):
			_register_button.call("set_button_label", switch_label)
		else:
			_register_button.text = switch_label
	if _credentials_confirm_button != null:
		var confirm_label: String = "注册并登录" if _credentials_register_mode else "登录"
		if _credentials_confirm_button.has_method("set_button_label"):
			_credentials_confirm_button.call("set_button_label", confirm_label)
		else:
			_credentials_confirm_button.text = confirm_label
	if _credentials_hint_label != null:
		_credentials_hint_label.text = ""


## 点击弹窗空白遮罩时关闭账号密码弹窗，不穿透到底层菜单按钮。
func _on_credentials_backdrop_gui_input(event: InputEvent) -> void:
	if not _is_credentials_popup_visible():
		return
	if event is InputEventMouseButton and (event as InputEventMouseButton).pressed:
		get_viewport().set_input_as_handled()
		_credentials_backdrop.accept_event()
		_hide_credentials_popup()
	if event is InputEventScreenTouch and (event as InputEventScreenTouch).pressed:
		get_viewport().set_input_as_handled()
		_credentials_backdrop.accept_event()
		_hide_credentials_popup()


## 校验账号密码输入；通过后广播给上层执行真正登录。
func _on_credentials_confirmed() -> void:
	var account: String = _account_input.text.strip_edges() if _account_input != null else ""
	var password: String = _password_input.text.strip_edges() if _password_input != null else ""
	if account.is_empty() or password.is_empty():
		if _credentials_hint_label != null:
			_credentials_hint_label.text = "请输入账号和密码。"
		return
	if _credentials_register_mode:
		var confirm_password: String = _confirm_password_input.text.strip_edges() if _confirm_password_input != null else ""
		if confirm_password.is_empty():
			if _credentials_hint_label != null:
				_credentials_hint_label.text = "请确认密码。"
			return
		if password != confirm_password:
			if _credentials_hint_label != null:
				_credentials_hint_label.text = "两次密码不一致。"
			return
		register_credentials_submitted.emit(account, password)
	else:
		login_credentials_submitted.emit(account, password)
	_hide_credentials_popup()


## 将箭头平滑移动到目标按钮左侧的垂直居中位置。
func _move_arrow_to_button(option_button: Button, immediate: bool) -> void:
	if _arrow == null or option_button == null:
		return
	var target_position: Vector2 = _calculate_arrow_target_position(option_button)
	if _arrow_tween != null and _arrow_tween.is_valid():
		_arrow_tween.kill()
	if immediate:
		_arrow.global_position = target_position
		return
	_arrow_tween = create_tween()
	_arrow_tween.set_trans(Tween.TRANS_CUBIC)
	_arrow_tween.set_ease(Tween.EASE_OUT)
	_arrow_tween.tween_property(_arrow, "global_position", target_position, ARROW_MOVE_DURATION)


## 根据按钮位置计算箭头目标点，横坐标使用 ArrowIndicator 在场景里的设计稿初始位置。
func _calculate_arrow_target_position(option_button: Button) -> Vector2:
	var arrow_size: Vector2 = _arrow.size
	var button_position: Vector2 = option_button.global_position
	var button_size: Vector2 = option_button.size
	var target_x: float = _arrow_global_x
	if is_zero_approx(target_x) or target_x >= button_position.x:
		target_x = button_position.x - arrow_size.x - ARROW_BUTTON_GAP
	if target_x > button_position.x - ARROW_BUTTON_GAP:
		target_x = button_position.x - arrow_size.x - FALLBACK_ARROW_LEFT_OFFSET
	var target_y: float = button_position.y + (button_size.y - arrow_size.y) * 0.5
	return Vector2(target_x, target_y)
