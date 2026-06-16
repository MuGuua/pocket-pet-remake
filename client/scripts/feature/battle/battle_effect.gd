extends Node2D
class_name BattleEffect

@onready var _pivot: Node2D = %Pivot
@onready var _effect_sprite: AnimatedSprite2D = %EffectSprite
@onready var _debug_label: Label = %DebugLabel
@onready var _animation_player: AnimationPlayer = %AnimationPlayer

var _motion_root: Node2D = null

func _ready() -> void:
	_ensure_motion_root()
	_ensure_builtin_animations()

func _ensure_motion_root() -> void:
	if _motion_root != null:
		return
	_motion_root = Node2D.new()
	_motion_root.name = "MotionRoot"
	_pivot.add_child(_motion_root)
	if _effect_sprite.get_parent() == _pivot:
		_pivot.remove_child(_effect_sprite)
		_motion_root.add_child(_effect_sprite)

func play_from_config(skill_visual: SkillVisualConfig, fallback_name: String = "", flip_horizontal: bool = false) -> void:
	if not is_node_ready():
		await ready
	var sprite_animation_name: String = "默认"
	if skill_visual != null:
		_apply_visual(skill_visual)
		sprite_animation_name = _resolve_sprite_animation(skill_visual)
	else:
		_apply_fallback_visual(fallback_name)

	scale.x = -1.0 if flip_horizontal else 1.0
	_debug_label.visible = false
	var player_animation_name: String = _resolve_player_animation(skill_visual)
	var sprite_duration: float = _play_sprite_animation(sprite_animation_name)
	var player_duration: float = _play_player_animation(player_animation_name)
	var playback_duration: float = max(sprite_duration, player_duration)
	if playback_duration <= 0.0:
		playback_duration = 0.25
	await get_tree().create_timer(playback_duration).timeout
	queue_free()

func _apply_visual(skill_visual: SkillVisualConfig) -> void:
	_ensure_motion_root()
	_pivot.position = skill_visual.effect_offset
	_pivot.scale = Vector2.ONE
	_effect_sprite.sprite_frames = _resolve_effect_frames(skill_visual)
	_effect_sprite.frame = 0
	_effect_sprite.animation = &"默认"
	_effect_sprite.modulate = skill_visual.effect_tint
	var has_sprite_frames: bool = _effect_sprite.sprite_frames != null
	_effect_sprite.visible = has_sprite_frames
	if has_sprite_frames:
		_effect_sprite.scale = skill_visual.effect_scale
		_motion_root.scale = Vector2.ONE
	else:
		_effect_sprite.scale = Vector2.ONE
		_motion_root.scale = skill_visual.effect_scale
	_debug_label.visible = false

func _apply_fallback_visual(_effect_name: String) -> void:
	_ensure_motion_root()
	_pivot.position = Vector2.ZERO
	_pivot.scale = Vector2.ONE
	_motion_root.scale = Vector2.ONE
	_effect_sprite.scale = Vector2.ONE
	_effect_sprite.sprite_frames = null
	_effect_sprite.modulate = Color.WHITE
	_effect_sprite.visible = false
	_debug_label.visible = false

func _resolve_effect_frames(skill_visual: SkillVisualConfig) -> SpriteFrames:
	if skill_visual == null:
		return null
	return skill_visual.resolve_effect_frames()

func _resolve_sprite_animation(skill_visual: SkillVisualConfig) -> String:
	if skill_visual == null:
		return "默认"
	return skill_visual.resolve_effect_animation_name()

func _play_sprite_animation(animation_name: String) -> float:
	if _effect_sprite.sprite_frames == null or not _effect_sprite.sprite_frames.has_animation(animation_name):
		return 0.0
	_effect_sprite.play(animation_name)
	var frame_count: int = _effect_sprite.sprite_frames.get_frame_count(animation_name)
	var animation_speed: float = _effect_sprite.sprite_frames.get_animation_speed(animation_name)
	if frame_count <= 1 or animation_speed <= 0.0:
		return 0.12
	return float(frame_count) / animation_speed

func _play_player_animation(animation_name: String) -> float:
	if not _animation_player.has_animation(animation_name):
		return 0.0
	_animation_player.play(animation_name)
	return _animation_player.get_animation(animation_name).length

func _resolve_player_animation(skill_visual: SkillVisualConfig) -> String:
	if skill_visual != null:
		var direct_name: String = skill_visual.effect_player_animation.strip_edges()
		if not direct_name.is_empty() and _animation_player.has_animation(direct_name):
			return direct_name
	return "特效默认"

func _ensure_builtin_animations() -> void:
	_ensure_animation("特效默认", _build_default_animation())
	_ensure_animation("冲击", _build_impact_animation())
	_ensure_animation("治疗特效", _build_heal_animation())
	_ensure_animation("状态特效", _build_status_animation())
	_ensure_animation("爆发", _build_burst_animation())

func _ensure_animation(animation_name: String, animation: Animation) -> void:
	if _animation_player.has_animation(animation_name):
		return
	_get_default_animation_library().add_animation(animation_name, animation)

func _get_default_animation_library() -> AnimationLibrary:
	if not _animation_player.has_animation_library(""):
		_animation_player.add_animation_library("", AnimationLibrary.new())
	return _animation_player.get_animation_library("")

const MOTION_SCALE_TRACK_PATH: String = "Pivot/MotionRoot:scale"

func _build_default_animation() -> Animation:
	var animation: Animation = Animation.new()
	animation.length = 0.38
	var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(scale_track, NodePath(MOTION_SCALE_TRACK_PATH))
	animation.track_insert_key(scale_track, 0.0, Vector2(0.74, 0.74))
	animation.track_insert_key(scale_track, 0.16, Vector2(1.1, 1.1))
	animation.track_insert_key(scale_track, 0.38, Vector2(1.18, 1.18))
	var alpha_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(alpha_track, NodePath(":modulate:a"))
	animation.track_insert_key(alpha_track, 0.0, 0.0)
	animation.track_insert_key(alpha_track, 0.08, 1.0)
	animation.track_insert_key(alpha_track, 0.38, 0.0)
	var offset_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(offset_track, NodePath("Pivot:position"))
	animation.track_insert_key(offset_track, 0.0, Vector2(0.0, 12.0))
	animation.track_insert_key(offset_track, 0.38, Vector2(0.0, -16.0))
	return animation

func _build_impact_animation() -> Animation:
	var animation: Animation = Animation.new()
	animation.length = 0.26
	var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(scale_track, NodePath(MOTION_SCALE_TRACK_PATH))
	animation.track_insert_key(scale_track, 0.0, Vector2(0.42, 0.42))
	animation.track_insert_key(scale_track, 0.08, Vector2(1.26, 1.26))
	animation.track_insert_key(scale_track, 0.26, Vector2(0.92, 0.92))
	var rotation_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(rotation_track, NodePath(":rotation"))
	animation.track_insert_key(rotation_track, 0.0, -0.16)
	animation.track_insert_key(rotation_track, 0.08, 0.12)
	animation.track_insert_key(rotation_track, 0.26, 0.0)
	var alpha_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(alpha_track, NodePath(":modulate:a"))
	animation.track_insert_key(alpha_track, 0.0, 0.0)
	animation.track_insert_key(alpha_track, 0.04, 1.0)
	animation.track_insert_key(alpha_track, 0.26, 0.0)
	return animation

func _build_heal_animation() -> Animation:
	var animation: Animation = Animation.new()
	animation.length = 0.5
	var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(scale_track, NodePath(MOTION_SCALE_TRACK_PATH))
	animation.track_insert_key(scale_track, 0.0, Vector2(0.6, 0.6))
	animation.track_insert_key(scale_track, 0.18, Vector2(1.0, 1.0))
	animation.track_insert_key(scale_track, 0.5, Vector2(1.2, 1.2))
	var alpha_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(alpha_track, NodePath(":modulate:a"))
	animation.track_insert_key(alpha_track, 0.0, 0.0)
	animation.track_insert_key(alpha_track, 0.08, 1.0)
	animation.track_insert_key(alpha_track, 0.5, 0.0)
	var offset_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(offset_track, NodePath("Pivot:position"))
	animation.track_insert_key(offset_track, 0.0, Vector2(0.0, 18.0))
	animation.track_insert_key(offset_track, 0.5, Vector2(0.0, -34.0))
	return animation

func _build_status_animation() -> Animation:
	var animation: Animation = Animation.new()
	animation.length = 0.42
	var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(scale_track, NodePath(MOTION_SCALE_TRACK_PATH))
	animation.track_insert_key(scale_track, 0.0, Vector2(0.84, 0.84))
	animation.track_insert_key(scale_track, 0.2, Vector2(1.08, 1.08))
	animation.track_insert_key(scale_track, 0.42, Vector2(1.0, 1.0))
	var offset_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(offset_track, NodePath("Pivot:position"))
	animation.track_insert_key(offset_track, 0.0, Vector2(0.0, 4.0))
	animation.track_insert_key(offset_track, 0.21, Vector2(0.0, -18.0))
	animation.track_insert_key(offset_track, 0.42, Vector2(0.0, -8.0))
	var alpha_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(alpha_track, NodePath(":modulate:a"))
	animation.track_insert_key(alpha_track, 0.0, 0.0)
	animation.track_insert_key(alpha_track, 0.08, 1.0)
	animation.track_insert_key(alpha_track, 0.42, 0.0)
	return animation

func _build_burst_animation() -> Animation:
	var animation: Animation = Animation.new()
	animation.length = 0.32
	var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(scale_track, NodePath(MOTION_SCALE_TRACK_PATH))
	animation.track_insert_key(scale_track, 0.0, Vector2(0.58, 0.58))
	animation.track_insert_key(scale_track, 0.12, Vector2(1.34, 1.0))
	animation.track_insert_key(scale_track, 0.32, Vector2(1.0, 0.92))
	var alpha_track: int = animation.add_track(Animation.TYPE_VALUE)
	animation.track_set_path(alpha_track, NodePath(":modulate:a"))
	animation.track_insert_key(alpha_track, 0.0, 0.0)
	animation.track_insert_key(alpha_track, 0.04, 1.0)
	animation.track_insert_key(alpha_track, 0.32, 0.0)
	return animation
