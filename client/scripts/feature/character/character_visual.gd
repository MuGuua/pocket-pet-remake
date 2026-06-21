extends Node2D
class_name CharacterVisual

const CharacterVisualScene: PackedScene = preload("res://scenes/character/character_visual.tscn")
const RENDER_MODE_CHJ: String = "chj"
const RENDER_MODE_PNG: String = "png"

@onready var _sprite: AnimatedSprite2D = %AnimatedSprite2D
@onready var _chj_renderer: ChjWorldRenderer = %ChjSprite2D

var _skin: UnitSkin = null
var _current_skin_id: String = ""
var _render_mode: String = ""
var _world_state: String = "idle"
var _world_direction: String = "down"
var _using_png_override: bool = false

## 根据服务端下发的 skin_id 切换形象；成功时返回 true。
func apply_skin_id(skin_id: String) -> bool:
	var normalized_skin_id: String = skin_id.strip_edges()
	if normalized_skin_id.is_empty():
		return false
	if normalized_skin_id == _current_skin_id and _skin != null and not _render_mode.is_empty():
		return true
	var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(normalized_skin_id)
	if skin == null:
		push_warning("找不到玩家形象资源: %s" % normalized_skin_id)
		return false
	if not _apply_skin(skin):
		push_warning("形象资源无法渲染: %s" % normalized_skin_id)
		return false
	_current_skin_id = normalized_skin_id
	return true


## 是否当前以 CHJ 为基础渲染（含局部 PNG 覆盖场景）。
func uses_chj_render() -> bool:
	return _render_mode == RENDER_MODE_CHJ


## 播放世界场景四方向动画；CHJ 模式下若 sprite_frames 有对应覆盖则只替换该动画。
func play_world(state: String, direction_suffix: String) -> void:
	if _skin == null:
		return
	_world_state = state
	_world_direction = direction_suffix
	if _render_mode == RENDER_MODE_CHJ:
		var override_animation: String = _skin.get_world_png_override_animation(state, direction_suffix)
		if not override_animation.is_empty():
			_show_png_override(override_animation)
			return
		_show_chj_world()
		if _chj_renderer != null:
			_chj_renderer.set_world_pose(state, direction_suffix)
		return
	if _sprite == null:
		return
	var animation_name: String = _skin.resolve_world_animation(state, direction_suffix)
	_align_sprite_to_ground_for_animation(animation_name)
	_play_animation(animation_name)


## 播放战斗语义动画；sprite_frames 有对应动画则 PNG，否则走 chj_skill_path。
func play_battle(animation_name: String) -> void:
	if _skin == null:
		return
	if _render_mode == RENDER_MODE_CHJ:
		var png_override: String = _skin.get_battle_action_png_override(animation_name)
		if not png_override.is_empty():
			_show_png_override(png_override)
			return
		if _chj_renderer != null:
			var skill_path: String = _skin.resolve_chj_skill_path()
			if skill_path.is_empty():
				push_warning("CHJ 形象缺少技能 CHJ 且 sprite_frames 无覆盖: %s" % _skin.skin_id)
				return
			_show_chj_world()
			_chj_renderer.start_skill_animation(skill_path, 0)
		return
	if _sprite == null or _render_mode != RENDER_MODE_PNG:
		return
	var resolved_name: String = _skin.resolve_animation(animation_name)
	_play_animation(resolved_name)


## 返回脚底锚点在 CharacterVisual 本地坐标中的位置，默认对齐到节点原点。
func get_feet_local_position() -> Vector2:
	return Vector2.ZERO


## 返回世界碰撞圆相对 Player 根节点的建议偏移。
func get_world_collision_offset() -> Vector2:
	if _skin == null:
		return Vector2.ZERO
	return _skin.world_collision_offset


func _process(delta: float) -> void:
	if _render_mode != RENDER_MODE_CHJ or _chj_renderer == null or _using_png_override:
		return
	_chj_renderer.tick_world(_world_state, _world_direction, delta)


func _apply_skin(skin: UnitSkin) -> bool:
	_skin = skin
	var chj_resolved_path: String = skin.resolve_chj_path()
	if not chj_resolved_path.is_empty() and _chj_renderer != null:
		if _chj_renderer.apply_chj(chj_resolved_path, skin):
			_render_mode = RENDER_MODE_CHJ
			_using_png_override = false
			if skin.has_configured_sprite_frames() and _sprite != null:
				_setup_png_sprite(skin)
				_sprite.visible = false
			_chj_renderer.visible = true
			_world_state = "idle"
			_world_direction = "down"
			_chj_renderer.set_world_pose("idle", "down")
			return true
		push_warning("CHJ 加载失败，尝试纯 PNG: %s" % chj_resolved_path)
	if skin.has_configured_sprite_frames():
		return _apply_png_skin(skin)
	return false


func _apply_png_skin(skin: UnitSkin) -> bool:
	if skin.sprite_frames == null:
		return false
	_render_mode = RENDER_MODE_PNG
	_using_png_override = false
	if _chj_renderer != null:
		_chj_renderer.visible = false
	_setup_png_sprite(skin)
	_sprite.visible = true
	_align_sprite_to_ground()
	var bootstrap_animation: String = skin.resolve_avatar_preview_animation()
	if not bootstrap_animation.is_empty():
		_play_animation(bootstrap_animation)
	return true


func _setup_png_sprite(skin: UnitSkin) -> void:
	if _sprite == null:
		return
	_sprite.sprite_frames = skin.sprite_frames
	_sprite.position = skin.sprite_offset
	_sprite.scale = skin.sprite_scale
	_sprite.modulate = skin.tint
	_sprite.flip_h = false


func _show_png_override(animation_name: String) -> void:
	if _sprite == null or _skin == null or not _skin.has_animation(animation_name):
		return
	_using_png_override = true
	if _chj_renderer != null:
		_chj_renderer.visible = false
	_sprite.visible = true
	_align_sprite_to_ground_for_animation(animation_name)
	_play_animation(animation_name)


func _show_chj_world() -> void:
	_using_png_override = false
	if _sprite != null:
		_sprite.visible = false
	if _chj_renderer != null:
		_chj_renderer.visible = true


func _align_sprite_to_ground() -> void:
	if _skin == null:
		return
	var animation_name: String = _skin.resolve_avatar_preview_animation()
	if animation_name.is_empty():
		return
	_align_sprite_to_ground_for_animation(animation_name)


func _align_sprite_to_ground_for_animation(animation_name: String) -> void:
	if _sprite == null or _sprite.sprite_frames == null or _skin == null:
		return
	if not _sprite.sprite_frames.has_animation(animation_name):
		return
	var frame_texture: Texture2D = _sprite.sprite_frames.get_frame_texture(animation_name, 0)
	if frame_texture == null:
		return
	var frame_height: float = float(frame_texture.get_height()) * absf(_sprite.scale.y)
	if _sprite.centered:
		_sprite.position = Vector2(_skin.sprite_offset.x, -frame_height * 0.5 + _skin.sprite_offset.y)
	else:
		_sprite.position = Vector2(_skin.sprite_offset.x, -frame_height + _skin.sprite_offset.y)


func _play_animation(animation_name: String) -> void:
	if _sprite == null or _sprite.sprite_frames == null:
		return
	if not _sprite.sprite_frames.has_animation(animation_name):
		return
	_sprite.flip_h = false
	if _skin != null:
		var flip_h_value: Variant = _skin.animation_flip_h.get(animation_name, null)
		if flip_h_value != null:
			_sprite.flip_h = bool(flip_h_value)
	_align_sprite_to_ground_for_animation(animation_name)
	_sprite.play(animation_name)
