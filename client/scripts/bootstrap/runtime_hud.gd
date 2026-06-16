extends Control
class_name RuntimeHud

const UiFormat = preload("res://scripts/common/ui_format.gd")

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
	if log_output != null:
		log_output.append_text(UiFormat.normalize_text(message) + "\n")
