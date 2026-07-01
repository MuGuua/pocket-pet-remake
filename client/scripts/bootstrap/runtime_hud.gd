extends Control
class_name RuntimeHud

## HUD 日志区最多保留的行数，防止 RichTextLabel 无限增长拖慢主线程。
const MAX_LOG_LINES: int = 120

## 左上角玩家头像与属性条 HUD。
@onready var player_status_hud: PanelContainer = %PlayerStatusHud
## 人物 HUD 下方的首只出战宠物 HUD。
@onready var pet_status_hud: PanelContainer = %PetStatusHud
## 右上角当前场景名称标签。
@onready var scene_name_label: Label = %SceneNameLabel
## 右下角常驻背包按钮，移动端玩家可以直接点击打开背包。
@onready var bag_button: Button = %BagButton
## 底部调试日志输出控件。
@onready var log_output: RichTextLabel = %LogOutput

## 头像被点击时向外转发，供主场景打开人物面板。
signal avatar_pressed
## 背包按钮被点击时向外转发，供主场景复用现有背包打开链路。
signal bag_pressed
## 宠物 HUD 被点击时向外转发，供主场景打开宠物状态面板。
signal pet_pressed


## 初始化 HUD 常驻按钮与头像事件转发。
func _ready() -> void:
	if player_status_hud != null and player_status_hud.has_signal("avatar_pressed"):
		player_status_hud.connect("avatar_pressed", Callable(self, "_on_player_status_avatar_pressed"))
	if pet_status_hud != null and pet_status_hud.has_signal("pet_pressed"):
		pet_status_hud.connect("pet_pressed", Callable(self, "_on_pet_status_pressed"))
	if bag_button != null:
		bag_button.pressed.connect(_on_bag_button_pressed)


## 刷新头像、血条、蓝条与经验条。
func refresh_player_status() -> void:
	if player_status_hud != null and player_status_hud.has_method("refresh_from_game_state"):
		player_status_hud.call("refresh_from_game_state")
	if pet_status_hud != null and pet_status_hud.has_method("refresh_from_game_state"):
		pet_status_hud.call("refresh_from_game_state")


## 更新场景内局部坐标展示。
func set_local_coordinates(local_position: Vector2) -> void:
	if player_status_hud != null and player_status_hud.has_method("set_local_coordinates"):
		player_status_hud.call("set_local_coordinates", local_position)


## 控制左上角玩家状态 HUD 显隐；进入战斗时隐藏，回到世界后恢复。
func set_player_status_visible(visible: bool) -> void:
	if player_status_hud != null:
		player_status_hud.visible = visible
	if pet_status_hud != null:
		if pet_status_hud.has_method("set_hud_enabled"):
			pet_status_hud.call("set_hud_enabled", visible)
		else:
			pet_status_hud.visible = visible and not GameState.pets.is_empty()


## 更新右上角当前场景名称；名称由当前地图场景脚本导出配置。
func set_scene_name(scene_name: String) -> void:
	if scene_name_label == null:
		return
	var display_name: String = scene_name.strip_edges()
	scene_name_label.text = UiFormat.normalize_text(display_name)
	scene_name_label.visible = not display_name.is_empty()


func _on_player_status_avatar_pressed() -> void:
	avatar_pressed.emit()


## 点击宠物 HUD 时，只广播打开宠物状态面板的意图。
func _on_pet_status_pressed() -> void:
	pet_pressed.emit()


## 点击右下角背包按钮时，只广播意图，不在 HUD 内直接操作背包面板。
func _on_bag_button_pressed() -> void:
	bag_pressed.emit()


## 追加一条运行时日志。
func append_log(message: String) -> void:
	if log_output == null:
		return
	log_output.append_text(UiFormat.normalize_text(message) + "\n")
	_trim_log_lines()


## 超出上限时丢弃最旧行，避免长时间运行后日志控件占用过多内存。
func _trim_log_lines() -> void:
	if log_output == null:
		return
	var current_text: String = log_output.text
	if current_text.is_empty():
		return
	var lines: PackedStringArray = current_text.split("\n", false)
	if lines.size() <= MAX_LOG_LINES:
		return
	log_output.text = "\n".join(lines.slice(lines.size() - MAX_LOG_LINES))
