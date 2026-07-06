extends Control
class_name CombinedStatusHud

## 点击玩家头像时触发，由 RuntimeHud 转发给主场景打开人物状态面板。
signal avatar_pressed
## 点击宠物 HUD 时触发，由 RuntimeHud 转发给主场景打开宠物状态面板。
signal pet_pressed

## 人物状态 HUD，负责人物头像、生命、法力、经验和坐标展示。
@onready var _player_status_hud: PanelContainer = %PlayerStatusHud
## 宠物状态 HUD，负责首只出战宠物头像、生命、法力和经验展示。
@onready var _pet_status_hud: PanelContainer = %PetStatusHud


## 初始化组合 HUD：只负责组合与转发，不在这里重复维护服务端状态。
func _ready() -> void:
    if _player_status_hud != null and _player_status_hud.has_signal("avatar_pressed"):
        _player_status_hud.connect("avatar_pressed", Callable(self, "_on_player_status_avatar_pressed"))
    if _pet_status_hud != null and _pet_status_hud.has_signal("pet_pressed"):
        _pet_status_hud.connect("pet_pressed", Callable(self, "_on_pet_status_pressed"))


## 刷新人物和宠物 HUD；数据仍由各自子 HUD 从 GameState 读取。
func refresh_from_game_state() -> void:
    if _player_status_hud != null and _player_status_hud.has_method("refresh_from_game_state"):
        _player_status_hud.call("refresh_from_game_state")
    if _pet_status_hud != null and _pet_status_hud.has_method("refresh_from_game_state"):
        _pet_status_hud.call("refresh_from_game_state")


## 更新人物 HUD 上的场景局部坐标。
func set_local_coordinates(local_position: Vector2) -> void:
    if _player_status_hud != null and _player_status_hud.has_method("set_local_coordinates"):
        _player_status_hud.call("set_local_coordinates", local_position)


## 控制组合 HUD 显隐；宠物 HUD 仍保留自身“无宠物时隐藏”的逻辑。
func set_hud_enabled(enabled: bool) -> void:
    visible = enabled
    if _player_status_hud != null:
        _player_status_hud.visible = enabled
    if _pet_status_hud != null:
        if _pet_status_hud.has_method("set_hud_enabled"):
            _pet_status_hud.call("set_hud_enabled", enabled)
        else:
            _pet_status_hud.visible = enabled and not GameState.pets.is_empty()


## 转发人物头像点击事件。
func _on_player_status_avatar_pressed() -> void:
    avatar_pressed.emit()


## 转发宠物 HUD 点击事件。
func _on_pet_status_pressed() -> void:
    pet_pressed.emit()
