extends Control
class_name RuntimeHud

const SETTINGS_MENU_SCENE: PackedScene = preload("res://scenes/ui/common/action_menu_popup.tscn")

## 单条业务提示在移动端 HUD 中保留的秒数。
const NOTICE_DISPLAY_DURATION_SEC: float = 3.0
const SETTINGS_ACTIONS: Array[Dictionary] = [
	{"key": "return_login", "label": "返回登录页"},
	{"key": "quit_game", "label": "退出游戏"},
]

## 左上角组合状态 HUD，内部包含人物 HUD 与宠物 HUD。
@onready var status_hud_group: CombinedStatusHud = %StatusHudGroup
## 右上角当前场景名称标签。
@onready var scene_name_label: Label = %SceneNameLabel
## 右下角常驻背包按钮，移动端玩家可以直接点击打开背包。
@onready var bag_button: Button = %BagButton
## 背包按钮旁边的设置按钮。
@onready var settings_button: Button = %SettingsButton
## 设置按钮旁边的任务按钮，移动端玩家可以直接打开任务面板。
@onready var task_button: Button = %TaskButton
## 设置按钮旁边的挂机按钮；仅暗雷地图展示。
@onready var auto_encounter_button: Button = %AutoEncounterButton
## 底部单条业务提示正文。
@onready var log_output: RichTextLabel = %LogOutput
## 单条业务提示使用的既有底部面板；默认隐藏，不恢复历史调试日志区域。
@onready var notice_panel: PanelContainer = $BottomPanel

## 头像被点击时向外转发，供主场景打开人物面板。
signal avatar_pressed
## 背包按钮被点击时向外转发，供主场景复用现有背包打开链路。
signal bag_pressed
## 任务按钮被点击时向外转发，供主场景复用任务面板打开链路。
signal task_pressed
## 宠物 HUD 被点击时向外转发，供主场景打开宠物状态面板。
signal pet_pressed
## 玩家选择返回登录页时向外转发。
signal return_to_login_pressed
## 玩家选择退出游戏时向外转发。
signal quit_game_pressed
## 玩家点击挂机按钮时向外转发。
signal auto_encounter_pressed

## 右下角设置动作菜单。
var _settings_menu: ActionMenuPopup = null
## 当前业务提示的显示代次；新提示会让旧提示的延迟隐藏失效。
var _notice_generation: int = 0


## 初始化 HUD 常驻按钮与头像事件转发。
func _ready() -> void:
	if status_hud_group != null and status_hud_group.has_signal("avatar_pressed"):
		status_hud_group.connect("avatar_pressed", Callable(self, "_on_player_status_avatar_pressed"))
	if status_hud_group != null and status_hud_group.has_signal("pet_pressed"):
		status_hud_group.connect("pet_pressed", Callable(self, "_on_pet_status_pressed"))
	if bag_button != null:
		bag_button.pressed.connect(_on_bag_button_pressed)
	if settings_button != null:
		settings_button.pressed.connect(_on_settings_button_pressed)
	if task_button != null:
		task_button.pressed.connect(_on_task_button_pressed)
	if auto_encounter_button != null:
		auto_encounter_button.pressed.connect(_on_auto_encounter_button_pressed)
	_ensure_settings_menu()


## 刷新头像、血条、蓝条与经验条。
func refresh_player_status() -> void:
	if status_hud_group != null and status_hud_group.has_method("refresh_from_game_state"):
		status_hud_group.call("refresh_from_game_state")


## 更新场景内局部坐标展示。
func set_local_coordinates(local_position: Vector2) -> void:
	if status_hud_group != null and status_hud_group.has_method("set_local_coordinates"):
		status_hud_group.call("set_local_coordinates", local_position)


## 控制左上角玩家状态 HUD 显隐；进入战斗时隐藏，回到世界后恢复。
func set_player_status_visible(should_show: bool) -> void:
	if status_hud_group != null and status_hud_group.has_method("set_hud_enabled"):
		status_hud_group.call("set_hud_enabled", should_show)
	elif status_hud_group != null:
		status_hud_group.visible = should_show


## 更新右上角当前场景名称；名称由当前地图场景脚本导出配置。
func set_scene_name(scene_name: String) -> void:
	if scene_name_label == null:
		return
	var display_name: String = scene_name.strip_edges()
	scene_name_label.text = UiFormat.normalize_text(display_name)
	scene_name_label.visible = not display_name.is_empty()


## 按当前可用性与开关态刷新挂机按钮；不可用时直接隐藏，避免非暗雷地图误触。
func set_auto_encounter_button_state(available: bool, active: bool) -> void:
	if auto_encounter_button == null:
		return
	auto_encounter_button.visible = available
	auto_encounter_button.disabled = not available
	auto_encounter_button.text = "挂机中" if active else "挂机"


func _on_player_status_avatar_pressed() -> void:
	avatar_pressed.emit()


## 点击宠物 HUD 时，只广播打开宠物状态面板的意图。
func _on_pet_status_pressed() -> void:
	pet_pressed.emit()


## 点击右下角背包按钮时，只广播意图，不在 HUD 内直接操作背包面板。
func _on_bag_button_pressed() -> void:
	_hide_settings_menu()
	bag_pressed.emit()


## 点击右下角任务按钮时，只广播意图，不在 HUD 内直接操作任务面板。
func _on_task_button_pressed() -> void:
	_hide_settings_menu()
	task_pressed.emit()


## 点击挂机按钮时只广播意图，由主场景根据地图暗雷状态决定是否真正切换。
func _on_auto_encounter_button_pressed() -> void:
	_hide_settings_menu()
	auto_encounter_pressed.emit()


## 点击设置按钮时打开或关闭锚点菜单。
func _on_settings_button_pressed() -> void:
	if settings_button == null:
		return
	_ensure_settings_menu()
	if _settings_menu == null:
		return
	if _settings_menu.is_open():
		_settings_menu.hide_menu()
		return
	_settings_menu.configure_actions(SETTINGS_ACTIONS)
	_settings_menu.open_near(settings_button, {
		"placement": "above",
		"anchor_gap": 6.0,
	})


## 客户端已关闭运行时日志；保留空入口避免业务流程为日志产生额外分支。
func append_log(_message: String) -> void:
	pass


## 展示一条由业务流程产生的玩家提示，并在固定时间后自动隐藏。
## message 是服务端或客户端业务校验生成的提示正文。
func show_notice(message: String) -> void:
	var normalized_message: String = UiFormat.normalize_text(message).strip_edges()
	if normalized_message.is_empty() or log_output == null or notice_panel == null:
		return
	_notice_generation += 1
	var generation: int = _notice_generation
	log_output.text = normalized_message
	notice_panel.show()
	var notice_timer: SceneTreeTimer = get_tree().create_timer(NOTICE_DISPLAY_DURATION_SEC)
	await notice_timer.timeout
	if generation == _notice_generation and log_output != null:
		log_output.text = ""
		if notice_panel != null:
			notice_panel.hide()


## 懒创建设置动作菜单并挂到 HUD 根节点下，复用通用锚点弹层样式。
func _ensure_settings_menu() -> void:
	if _settings_menu != null:
		return
	_settings_menu = SETTINGS_MENU_SCENE.instantiate() as ActionMenuPopup
	if _settings_menu == null:
		return
	add_child(_settings_menu)
	_settings_menu.configure_actions(SETTINGS_ACTIONS)
	if not _settings_menu.action_selected.is_connected(_on_settings_action_selected):
		_settings_menu.action_selected.connect(_on_settings_action_selected)


## 主动隐藏设置菜单，供其它入口打开时复用。
func _hide_settings_menu() -> void:
	if _settings_menu != null and _settings_menu.is_open():
		_settings_menu.hide_menu()


## 根据玩家选择向主场景转发具体设置动作。
func _on_settings_action_selected(action_key: String) -> void:
	match action_key:
		"return_login":
			return_to_login_pressed.emit()
		"quit_game":
			quit_game_pressed.emit()
