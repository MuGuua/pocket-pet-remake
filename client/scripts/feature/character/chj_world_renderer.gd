extends Sprite2D
class_name ChjWorldRenderer

signal skill_animation_finished

## 四方向到 CHJ 动画组索引的映射；右方向复用左侧面并 force_flip。
const ACTION_MAP: Dictionary = {
	"down": {"idle": 0, "walk": 1, "force_flip": false},
	"up": {"idle": 2, "walk": 3, "force_flip": false},
	"left": {"idle": 4, "walk": 5, "force_flip": false},
	"right": {"idle": 4, "walk": 5, "force_flip": true},
}

var _chj_sprite: ChjSprite = null
var _atlas: AtlasTexture = null
var _walk_anim_speed: float = 120.0
var _idle_anim_speed: float = 60.0
var _frame_divisor: float = 8.0
var _world_state: String = "idle"
var _direction_suffix: String = "down"
var _frame_tick: float = 0.0
var _idle_tick: float = 0.0
var _sprite_offset: Vector2 = Vector2.ZERO
var _is_skill_playing: bool = false
var _skill_chj_sprite: ChjSprite = null
var _skill_frames: PackedInt32Array = PackedInt32Array()
var _skill_cursor: int = 0
var _skill_tick: float = 0.0


## 加载 CHJ 并按 UnitSkin 配置初始化显示参数；成功返回 true。
func apply_chj(path: String, skin: UnitSkin) -> bool:
	var loaded: ChjSprite = ChjSprite.load_from_path(path)
	if loaded == null:
		return false

	_chj_sprite = loaded
	if _atlas == null:
		_atlas = AtlasTexture.new()
	_atlas.atlas = _chj_sprite.texture
	texture = _atlas

	if skin != null:
		scale = skin.chj_display_scale
		_walk_anim_speed = skin.chj_walk_anim_speed
		_idle_anim_speed = skin.chj_idle_anim_speed
		_frame_divisor = maxf(skin.chj_frame_divisor, 0.001)
		_sprite_offset = skin.sprite_offset

	_is_skill_playing = false
	_skill_chj_sprite = null
	_frame_tick = 0.0
	_idle_tick = 0.0
	_world_state = "idle"
	_direction_suffix = "down"
	_align_to_ground()
	_apply_frame_info(_resolve_world_frame(false), _chj_sprite)
	return true


## 更新世界状态与朝向；state 为 idle/walk/battle，direction_suffix 为 up/down/left/right。
func set_world_pose(state: String, direction_suffix: String) -> void:
	if _is_skill_playing:
		return
	var direction_changed: bool = direction_suffix != _direction_suffix
	var state_changed: bool = state != _world_state
	_world_state = state
	_direction_suffix = direction_suffix
	if state_changed and _world_state == "walk":
		_frame_tick = 0.0
	if state_changed and (_world_state == "idle" or _world_state == "battle"):
		_idle_tick = 0.0
	if direction_changed:
		_frame_tick = 0.0
		_idle_tick = 0.0
	_apply_frame_info(_resolve_world_frame(false), _chj_sprite)
	if state_changed or direction_changed or _world_state == "battle":
		_align_to_ground()


## 每帧推进动画计时并刷新当前帧；由 CharacterVisual / BattleUnit 在 CHJ 模式下调用。
func tick_world(state: String, direction_suffix: String, delta: float) -> void:
	if _is_skill_playing:
		_tick_skill(delta)
		return
	var was_walking: bool = _world_state == "walk"
	set_world_pose(state, direction_suffix)
	var is_walking: bool = _world_state == "walk"
	if is_walking:
		_frame_tick += delta * _walk_anim_speed
		_idle_tick = 0.0
	elif was_walking and not is_walking:
		_frame_tick = 0.0
		_idle_tick = 0.0
	else:
		_frame_tick = 0.0
		_idle_tick += delta * _idle_anim_speed
	_apply_frame_info(_resolve_world_frame(true), _chj_sprite)


## 播放独立技能 CHJ 的一次性动画；默认读取第 0 组，成功返回 true。
func start_skill_animation(skill_path: String, action_index: int = 0) -> bool:
	var normalized_path: String = skill_path.strip_edges()
	if normalized_path.is_empty():
		return false
	var loaded_skill: ChjSprite = ChjSprite.load_from_path(normalized_path)
	if loaded_skill == null:
		return false
	var frames: PackedInt32Array = loaded_skill.get_animation_frames(action_index)
	if frames.is_empty():
		return false
	_skill_chj_sprite = loaded_skill
	_skill_frames = frames
	_skill_cursor = 0
	_skill_tick = 0.0
	_is_skill_playing = true
	visible = true
	_atlas.atlas = _skill_chj_sprite.texture
	_apply_frame_info(_resolve_skill_frame(0), _skill_chj_sprite)
	return true


## 估算当前技能 CHJ 播放时长（秒），供战斗层做超时兜底。
func estimate_skill_playback_seconds() -> float:
	if _skill_frames.is_empty():
		return 0.5
	return float(_skill_frames.size()) * _frame_divisor / maxf(_walk_anim_speed, 1.0)


## 强制结束技能 CHJ 并恢复主 CHJ 显示。
func cancel_skill_animation() -> void:
	if not _is_skill_playing:
		return
	_finish_skill_animation()


## 技能 CHJ 是否正在播放。
func is_skill_playing() -> bool:
	return _is_skill_playing


## 返回当前帧高度（含缩放），供脚底对齐计算。
func get_frame_display_height() -> float:
	var active_sprite: ChjSprite = _skill_chj_sprite if _is_skill_playing and _skill_chj_sprite != null else _chj_sprite
	if active_sprite == null:
		return 0.0
	return float(active_sprite.frame_height) * absf(scale.y)


func _resolve_world_frame(use_tick: bool) -> Dictionary:
	if _chj_sprite == null:
		return {"frame_index": 0, "flip": false}

	if _world_state == "battle":
		return _resolve_battle_idle_frame(use_tick)

	var mode: String = "walk" if _world_state == "walk" else "idle"
	var direction_key: String = _direction_suffix
	if not ACTION_MAP.has(direction_key):
		direction_key = "down"
	var map: Dictionary = ACTION_MAP[direction_key]
	var action_index: int = int(map.get(mode, 0))
	var frames: PackedInt32Array = _chj_sprite.get_animation_frames(action_index)
	var tick: float = _frame_tick if _world_state == "walk" else _idle_tick
	var cursor: int = 0
	if use_tick:
		cursor = int(floor(tick / _frame_divisor)) % maxi(frames.size(), 1)
	var raw: int = int(frames[cursor])
	var force_flip: bool = bool(map.get("force_flip", false))
	var flip: bool = raw >= 128 or force_flip
	var frame_index: int = raw - 128 if raw >= 128 else raw
	return {"frame_index": frame_index, "flip": flip}


func _resolve_battle_idle_frame(use_tick: bool) -> Dictionary:
	var frames: PackedInt32Array = _chj_sprite.get_battle_idle_frames()
	var cursor: int = 0
	if use_tick:
		cursor = int(floor(_idle_tick / _frame_divisor)) % maxi(frames.size(), 1)
	var raw: int = int(frames[cursor])
	var flip: bool = raw >= 128
	var frame_index: int = raw - 128 if raw >= 128 else raw
	return {"frame_index": frame_index, "flip": flip}


func _resolve_skill_frame(cursor: int) -> Dictionary:
	if _skill_frames.is_empty():
		return {"frame_index": 0, "flip": false}
	var safe_cursor: int = clampi(cursor, 0, _skill_frames.size() - 1)
	var raw: int = int(_skill_frames[safe_cursor])
	var flip: bool = raw >= 128
	var frame_index: int = raw - 128 if raw >= 128 else raw
	return {"frame_index": frame_index, "flip": flip}


func _tick_skill(delta: float) -> void:
	if _skill_chj_sprite == null or _skill_frames.is_empty():
		_finish_skill_animation()
		return
	_skill_tick += delta * _walk_anim_speed
	var cursor: int = int(floor(_skill_tick / _frame_divisor))
	if cursor >= _skill_frames.size():
		_finish_skill_animation()
		return
	if cursor != _skill_cursor:
		_skill_cursor = cursor
	_apply_frame_info(_resolve_skill_frame(_skill_cursor), _skill_chj_sprite)


func _finish_skill_animation() -> void:
	_is_skill_playing = false
	_skill_chj_sprite = null
	_skill_frames = PackedInt32Array()
	_skill_cursor = 0
	_skill_tick = 0.0
	if _chj_sprite != null:
		_atlas.atlas = _chj_sprite.texture
		_apply_frame_info(_resolve_world_frame(true), _chj_sprite)
	skill_animation_finished.emit()


func _apply_frame_info(frame_info: Dictionary, chj_source: ChjSprite) -> void:
	if chj_source == null or _atlas == null:
		return
	var frame_index: int = int(frame_info.get("frame_index", 0))
	_atlas.region = Rect2(
		float(frame_index * chj_source.frame_width),
		0.0,
		float(chj_source.frame_width),
		float(chj_source.frame_height)
	)
	flip_h = bool(frame_info.get("flip", false))


func _align_to_ground() -> void:
	if _chj_sprite == null:
		return
	var frame_height: float = get_frame_display_height()
	if centered:
		position = Vector2(_sprite_offset.x, -frame_height * 0.5 + _sprite_offset.y)
	else:
		position = Vector2(_sprite_offset.x, -frame_height + _sprite_offset.y)
