extends Control

@onready var _director: BattleDirector = $BattleDirector
@onready var _network_provider: BattleNetworkProvider = %BattleNetworkProvider
@onready var _bg_texture: TextureRect = $Bg/TextureRect

var _battle_controller: Node = null
var _initialized_battle_id: int = 0

## 绑定主场景里的战斗控制器并订阅信号。
func bind_battle_controller(battle_controller: Node) -> void:
	if _battle_controller != null:
		_disconnect_battle_controller()
	_battle_controller = battle_controller
	if _battle_controller == null:
		return
	if _battle_controller.has_signal("battle_started"):
		_battle_controller.connect("battle_started", Callable(self, "_on_battle_started"))
	if _battle_controller.has_signal("battle_updated"):
		_battle_controller.connect("battle_updated", Callable(self, "_on_battle_updated"))
	if _battle_controller.has_signal("battle_finished"):
		_battle_controller.connect("battle_finished", Callable(self, "_on_battle_finished"))
	if _battle_controller.has_signal("action_responded"):
		_battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
	if _director != null and _director.has_signal("action_requested"):
		_director.connect("action_requested", Callable(self, "_on_action_requested"))

func _exit_tree() -> void:
	_disconnect_battle_controller()
	if _director != null and _director.has_signal("action_requested") and _director.action_requested.is_connected(_on_action_requested):
		_director.action_requested.disconnect(_on_action_requested)

func _disconnect_battle_controller() -> void:
	if _battle_controller == null:
		return
	if _battle_controller.has_signal("battle_started") and _battle_controller.battle_started.is_connected(_on_battle_started):
		_battle_controller.battle_started.disconnect(_on_battle_started)
	if _battle_controller.has_signal("battle_updated") and _battle_controller.battle_updated.is_connected(_on_battle_updated):
		_battle_controller.battle_updated.disconnect(_on_battle_updated)
	if _battle_controller.has_signal("battle_finished") and _battle_controller.battle_finished.is_connected(_on_battle_finished):
		_battle_controller.battle_finished.disconnect(_on_battle_finished)
	if _battle_controller.has_signal("action_responded") and _battle_controller.action_responded.is_connected(_on_action_responded):
		_battle_controller.action_responded.disconnect(_on_action_responded)
	_battle_controller = null

## 用世界地图截图替换战斗背景；为空时保留场景内默认贴图。
func apply_background_texture(texture: Texture2D) -> void:
	if _bg_texture == null:
		return
	if texture == null:
		return
	_bg_texture.texture = texture
	_bg_texture.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	_bg_texture.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_COVERED

func _on_battle_started(payload: Dictionary) -> void:
	var battle_id: int = int(payload.get("battle_id", 0))
	if battle_id <= 0:
		return
	if _initialized_battle_id != battle_id:
		_initialized_battle_id = battle_id
		_director.initialize_battle()
	else:
		_director.handle_battle_state_update()

func _on_battle_updated(payload: Dictionary) -> void:
	if payload.is_empty():
		return
	call_deferred("_run_state_update")

func _on_battle_finished(payload: Dictionary) -> void:
	var summary: String = "战斗结束"
	if bool(payload.get("win", false)):
		summary = "战斗胜利"
	else:
		summary = "战斗失败"
	_director.handle_battle_finished(summary)
	_initialized_battle_id = 0

func _on_action_responded(accepted: bool, _reason: String) -> void:
	if accepted:
		_director.handle_action_accepted()

func _on_action_requested(actor_id: int, action_type: int, skill_id: int, target_id: int) -> void:
	App.submit_battle_action(
		_network_provider.get_battle_id(),
		_network_provider.get_round(),
		actor_id,
		target_id,
		action_type,
		skill_id
	)

func _run_state_update() -> void:
	await _director.handle_battle_state_update()
