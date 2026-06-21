extends Control
class_name RuntimeHud

## HUD 日志区最多保留的行数，防止 RichTextLabel 无限增长拖慢主线程。
const MAX_LOG_LINES: int = 120

## 左上角玩家头像与属性条 HUD。
@onready var player_status_hud: PanelContainer = %PlayerStatusHud
@onready var log_output: RichTextLabel = %LogOutput

## 头像被点击时向外转发，供主场景打开人物面板。
signal avatar_pressed


func _ready() -> void:
	if player_status_hud != null and player_status_hud.has_signal("avatar_pressed"):
		player_status_hud.connect("avatar_pressed", Callable(self, "_on_player_status_avatar_pressed"))


## 刷新头像、血条、蓝条与经验条。
func refresh_player_status() -> void:
	if player_status_hud != null and player_status_hud.has_method("refresh_from_game_state"):
		player_status_hud.call("refresh_from_game_state")


## 更新场景内局部坐标展示。
func set_local_coordinates(local_position: Vector2) -> void:
	if player_status_hud != null and player_status_hud.has_method("set_local_coordinates"):
		player_status_hud.call("set_local_coordinates", local_position)


## 控制左上角玩家状态 HUD 显隐；进入战斗时隐藏，回到世界后恢复。
func set_player_status_visible(visible: bool) -> void:
	if player_status_hud != null:
		player_status_hud.visible = visible


func _on_player_status_avatar_pressed() -> void:
	avatar_pressed.emit()


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
