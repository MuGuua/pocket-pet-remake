extends Node2D
class_name CharacterVisual

const CharacterVisualScene: PackedScene = preload("res://scenes/character/character_visual.tscn")

@onready var _sprite: AnimatedSprite2D = %AnimatedSprite2D

var _skin: UnitSkin = null
var _current_skin_id: String = ""

## 根据服务端下发的 skin_id 切换形象；成功时返回 true。
func apply_skin_id(skin_id: String) -> bool:
	var normalized_skin_id: String = skin_id.strip_edges()
	if normalized_skin_id.is_empty():
		return false
	if normalized_skin_id == _current_skin_id and _skin != null:
		return true
	var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(normalized_skin_id)
	if skin == null:
		push_warning("找不到玩家形象资源: %s" % normalized_skin_id)
		return false
	_apply_skin(skin)
	_current_skin_id = normalized_skin_id
	return true

## 播放世界场景四方向动画，state 为 idle/walk/battle，direction_suffix 为 up/down/left/right。
func play_world(state: String, direction_suffix: String) -> void:
	if _skin == null or _sprite == null:
		return
	var animation_name: String = _skin.resolve_world_animation(state, direction_suffix)
	_play_animation(animation_name)

## 播放战斗语义动画，例如待机、普攻、技能。
func play_battle(animation_name: String) -> void:
	if _skin == null or _sprite == null:
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

func _apply_skin(skin: UnitSkin) -> void:
	_skin = skin
	if _skin.sprite_frames == null:
		push_warning("UnitSkin 缺少 sprite_frames: %s" % _skin.skin_id)
		return
	_sprite.sprite_frames = _skin.sprite_frames
	_sprite.position = _skin.sprite_offset
	_sprite.scale = _skin.sprite_scale
	_sprite.modulate = _skin.tint
	_sprite.flip_h = false
	_align_sprite_to_ground()
	var bootstrap_animation: String = _skin.resolve_world_bootstrap_animation()
	if not bootstrap_animation.is_empty():
		_play_animation(bootstrap_animation)

func _align_sprite_to_ground() -> void:
	if _sprite == null or _sprite.sprite_frames == null or _skin == null:
		return
	var animation_name: String = _skin.resolve_world_bootstrap_animation()
	if animation_name.is_empty():
		return
	var texture: Texture2D = _sprite.sprite_frames.get_frame_texture(animation_name, 0)
	if texture == null:
		return
	var frame_height: float = float(texture.get_height()) * absf(_sprite.scale.y)
	if _sprite.centered:
		_sprite.position = Vector2(_skin.sprite_offset.x, -frame_height * 0.5 + _skin.sprite_offset.y)
	else:
		_sprite.position = Vector2(_skin.sprite_offset.x, -frame_height + _skin.sprite_offset.y)

func _play_animation(animation_name: String) -> void:
	if _sprite == null or _sprite.sprite_frames == null:
		return
	if not _sprite.sprite_frames.has_animation(animation_name):
		return
	# 每次切动画先清零翻转，避免战斗待机 flip_h 残留到日常左右走帧。
	_sprite.flip_h = false
	if _skin != null:
		var flip_h_value: Variant = _skin.animation_flip_h.get(animation_name, null)
		if flip_h_value != null:
			_sprite.flip_h = bool(flip_h_value)
	_sprite.play(animation_name)
