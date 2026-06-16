extends PanelContainer

## 点击玩家头像时触发，由主场景打开人物状态面板。
signal avatar_pressed

const UiFormat = preload("res://scripts/common/ui_format.gd")
const CharacterSkinRegistry = preload("res://scripts/feature/character/character_skin_registry.gd")
const DEFAULT_AVATAR_SKIN_ID: String = "初始形象男_001"
## 与战斗表现层一致：unit_class 1 表示玩家角色。
const PLAYER_UNIT_CLASS: int = 1

## 可点击的玩家头像按钮。
@onready var _avatar_button: TextureButton = %AvatarButton
## 头像贴图展示节点。
@onready var _avatar_texture: TextureRect = %AvatarTexture
## 生命值进度条。
@onready var _hp_bar: ProgressBar = %HpBar
## 法力值进度条。
@onready var _mp_bar: ProgressBar = %MpBar
## 经验值进度条。
@onready var _exp_bar: ProgressBar = %ExpBar
## 场景内局部坐标展示标签。
@onready var _coord_label: Label = %CoordLabel

## 最近一次写入的局部坐标，避免重复刷新文本。
var _last_local_position: Vector2 = Vector2.INF


func _ready() -> void:
	_avatar_button.pressed.connect(_on_avatar_button_pressed)
	if not GameState.session_changed.is_connected(refresh_from_game_state):
		GameState.session_changed.connect(refresh_from_game_state)
	if not GameState.world_snapshot_changed.is_connected(refresh_from_game_state):
		GameState.world_snapshot_changed.connect(refresh_from_game_state)
	if not GameState.battle_changed.is_connected(refresh_from_game_state):
		GameState.battle_changed.connect(refresh_from_game_state)
	refresh_from_game_state()


func _exit_tree() -> void:
	if GameState.session_changed.is_connected(refresh_from_game_state):
		GameState.session_changed.disconnect(refresh_from_game_state)
	if GameState.world_snapshot_changed.is_connected(refresh_from_game_state):
		GameState.world_snapshot_changed.disconnect(refresh_from_game_state)
	if GameState.battle_changed.is_connected(refresh_from_game_state):
		GameState.battle_changed.disconnect(refresh_from_game_state)


## 根据 GameState 权威快照刷新头像与三条属性条。
func refresh_from_game_state() -> void:
	_update_avatar_texture()
	_apply_bar_values()
	if _last_local_position.is_equal_approx(Vector2.INF):
		var snapshot_x: float = float(GameState.player_snapshot.get("x", 0.0))
		var snapshot_y: float = float(GameState.player_snapshot.get("y", 0.0))
		set_local_coordinates(Vector2(snapshot_x, snapshot_y))


## 更新左下角场景内局部坐标文案。
func set_local_coordinates(local_position: Vector2) -> void:
	if local_position.is_equal_approx(_last_local_position):
		return
	_last_local_position = local_position
	_coord_label.text = UiFormat.normalize_text(
		"坐标 (%.0f, %.0f)" % [local_position.x, local_position.y]
	)


func _on_avatar_button_pressed() -> void:
	avatar_pressed.emit()


func _update_avatar_texture() -> void:
	var skin_id: String = str(GameState.player_snapshot.get("skin_id", ""))
	var avatar_texture: Texture2D = _resolve_avatar_texture(skin_id)
	if avatar_texture == null:
		avatar_texture = _resolve_avatar_texture(DEFAULT_AVATAR_SKIN_ID)
	if avatar_texture != null:
		_avatar_texture.texture = avatar_texture


func _resolve_avatar_texture(skin_id: String) -> Texture2D:
	var normalized_skin_id: String = skin_id.strip_edges()
	if normalized_skin_id.is_empty():
		return null
	var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(normalized_skin_id)
	if skin == null or skin.sprite_frames == null:
		return null
	var animation_name: String = skin.resolve_world_bootstrap_animation()
	if animation_name.is_empty():
		return null
	if skin.sprite_frames.get_frame_count(animation_name) <= 0:
		return null
	return skin.sprite_frames.get_frame_texture(animation_name, 0)


func _apply_bar_values() -> void:
	var hp_current: int = 0
	var hp_max: int = 1
	var mp_current: int = 0
	var mp_max: int = 1
	var exp_current: int = 0
	var exp_max: int = 1

	if GameState.is_in_battle:
		var battle_actor: Dictionary = _resolve_player_battle_actor()
		if not battle_actor.is_empty():
			hp_current = int(battle_actor.get("hp", 0))
			hp_max = max(1, int(battle_actor.get("hp_max", hp_current)))
			mp_current = int(battle_actor.get("mana", 0))
			mp_max = max(1, mp_current)
		else:
			hp_current = int(GameState.player_snapshot.get("hp", 0))
			hp_max = max(1, int(GameState.player_snapshot.get("hp_max", hp_current)))
			mp_current = int(GameState.player_snapshot.get("mana", 0))
			mp_max = max(1, mp_current)
	else:
		hp_current = int(GameState.player_snapshot.get("hp", 0))
		hp_max = max(1, int(GameState.player_snapshot.get("hp_max", hp_current)))
		mp_current = int(GameState.player_snapshot.get("mana", 0))
		mp_max = max(1, mp_current)

	exp_current = int(GameState.player_snapshot.get("exp", 0))
	exp_max = int(GameState.player_snapshot.get("exp_to_next", 0))
	if exp_max <= 0:
		var player_level: int = int(GameState.player_snapshot.get("level", 0))
		if player_level >= 100:
			exp_current = 1
			exp_max = 1
		else:
			exp_max = 1

	_hp_bar.max_value = float(hp_max)
	_hp_bar.value = float(clampi(hp_current, 0, hp_max))
	_mp_bar.max_value = float(mp_max)
	_mp_bar.value = float(clampi(mp_current, 0, mp_max))
	_exp_bar.max_value = float(exp_max)
	_exp_bar.value = float(clampi(exp_current, 0, exp_max))


## 从战斗 allies 列表中定位本地玩家单位，供战斗态 HUD 读取实时生命。
func _resolve_player_battle_actor() -> Dictionary:
	var allies_variant: Variant = GameState.battle_state.get("allies", [])
	if allies_variant is not Array:
		return {}
	for actor_variant in allies_variant:
		if actor_variant is not Dictionary:
			continue
		var actor: Dictionary = actor_variant
		if int(actor.get("unit_class", 0)) == PLAYER_UNIT_CLASS:
			return actor
	return {}
