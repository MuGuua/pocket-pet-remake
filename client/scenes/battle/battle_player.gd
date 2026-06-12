extends CharacterBody2D
class_name BattlePlayer

# 约定战斗舞台左侧为我方，右侧为敌方；当前脚本只负责纯表现，不参与任何数值结算。
const SIDE_ALLY: String = "ally"
const SIDE_ENEMY: String = "enemy"

# 我方与敌方默认使用轻度染色区分，方便移动端小舞台快速识别阵营。
const ALLY_BASE_TINT := Color(0.78, 0.91, 1.0, 1.0)
const ENEMY_BASE_TINT := Color(1.0, 0.84, 0.78, 1.0)

@onready var sprite_2d: Sprite2D = $Sprite2D
@onready var animation_player: AnimationPlayer = $AnimatedSprite2D

# 记录战斗角色在舞台中的原始落点，表现播放结束后需要回到这里。
var _home_position: Vector2 = Vector2.ZERO
# 同一时间只保留一条表现 tween，避免多次服务端事件把模型拉飞。
var _active_tween: Tween
# 当前角色所属阵营，仅影响朝向和默认染色。
var _side: String = SIDE_ALLY

func _ready() -> void:
	_home_position = position
	_apply_side_visual()
	play_idle()

# 由外部舞台控制器在实例初始化后设置当前阵营。
func setup_side(side: String) -> void:
	_side = side
	_apply_side_visual()

# 恢复到常驻待机态；战斗 HUD 每次刷新舞台时都会优先把模型收回稳定状态。
func play_idle() -> void:
	_stop_active_tween()
	position = _home_position
	scale = Vector2.ONE
	sprite_2d.modulate = _base_tint()
	if animation_player != null and animation_player.has_animation("idle"):
		animation_player.play("idle")

# 返回当前模型的全局锚点，供战斗场景计算投射物和命中特效的飞行起点。
func anchor_global_position() -> Vector2:
	return sprite_2d.global_position if sprite_2d != null else global_position

# 播放一次由技能类型驱动的施法/出手表现；这里只做视觉动作，真正命中与伤害仍以服务端事件为准。
func play_cast(animation_key: String, target_global_position: Vector2, accent_color: Color) -> void:
	play_idle()
	var move_direction := signf(target_global_position.x - anchor_global_position().x)
	if is_zero_approx(move_direction):
		move_direction = 1.0 if _side == SIDE_ALLY else -1.0

	_active_tween = create_tween()
	match animation_key:
		"heal":
			# 治疗类动作不前冲，而是做一个轻微上抬和发光脉冲，便于和攻击类动作区分。
			_active_tween.tween_property(self, "position:y", _home_position.y - 10.0, 0.10)
			_active_tween.parallel().tween_property(self, "scale", Vector2(1.08, 1.08), 0.10)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", accent_color, 0.10)
			_active_tween.tween_property(self, "position", _home_position, 0.16)
			_active_tween.parallel().tween_property(self, "scale", Vector2.ONE, 0.16)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", _base_tint(), 0.16)
		"volley":
			# 连射类动作使用更短促的两段前压，让玩家能感知这是多段技能而不是普通挥砍。
			_active_tween.tween_property(self, "position:x", _home_position.x + move_direction * 10.0, 0.07)
			_active_tween.parallel().tween_property(self, "scale", Vector2(1.06, 0.95), 0.07)
			_active_tween.tween_property(self, "position:x", _home_position.x + move_direction * 16.0, 0.07)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", accent_color, 0.07)
			_active_tween.tween_property(self, "position", _home_position, 0.14)
			_active_tween.parallel().tween_property(self, "scale", Vector2.ONE, 0.14)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", _base_tint(), 0.14)
		"burst":
			# 法术爆发类动作强调站桩蓄力和放光，不做大幅位移，避免和近战动画混淆。
			_active_tween.tween_property(self, "scale", Vector2(1.10, 1.10), 0.12)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", accent_color, 0.12)
			_active_tween.tween_property(self, "scale", Vector2.ONE, 0.18)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", _base_tint(), 0.18)
		_:
			# 默认按近战处理：短距离前冲后收回，适合作为人物普通攻击与斩击类技能通用模板。
			_active_tween.tween_property(self, "position:x", _home_position.x + move_direction * 16.0, 0.10)
			_active_tween.parallel().tween_property(self, "scale", Vector2(1.05, 0.96), 0.10)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", accent_color, 0.10)
			_active_tween.tween_property(self, "position", _home_position, 0.16)
			_active_tween.parallel().tween_property(self, "scale", Vector2.ONE, 0.16)
			_active_tween.parallel().tween_property(sprite_2d, "modulate", _base_tint(), 0.16)

# 播放一次受击闪白与轻微后仰，帮助玩家在服务端下发伤害事件时快速识别命中目标。
func play_hit_flash(accent_color: Color) -> void:
	_stop_active_tween()
	_active_tween = create_tween()
	var knockback_direction := -1.0 if _side == SIDE_ALLY else 1.0
	_active_tween.tween_property(sprite_2d, "modulate", accent_color, 0.08)
	_active_tween.parallel().tween_property(self, "position:x", _home_position.x + knockback_direction * 8.0, 0.08)
	_active_tween.tween_property(sprite_2d, "modulate", _base_tint(), 0.14)
	_active_tween.parallel().tween_property(self, "position", _home_position, 0.14)

# 治疗命中时使用绿色脉冲，和受击反馈保持视觉区分。
func play_heal_pulse(accent_color: Color) -> void:
	_stop_active_tween()
	_active_tween = create_tween()
	_active_tween.tween_property(self, "scale", Vector2(1.08, 1.08), 0.10)
	_active_tween.parallel().tween_property(sprite_2d, "modulate", accent_color, 0.10)
	_active_tween.tween_property(self, "scale", Vector2.ONE, 0.16)
	_active_tween.parallel().tween_property(sprite_2d, "modulate", _base_tint(), 0.16)

# 当单位不存在或当前舞台槽位不需要显示时，由外层直接调用设置是否可见。
func set_stage_visible(visible_value: bool) -> void:
	visible = visible_value
	if visible_value:
		play_idle()

func _apply_side_visual() -> void:
	if sprite_2d == null:
		return
	sprite_2d.flip_h = _side == SIDE_ALLY
	sprite_2d.modulate = _base_tint()

func _base_tint() -> Color:
	return ALLY_BASE_TINT if _side == SIDE_ALLY else ENEMY_BASE_TINT

func _stop_active_tween() -> void:
	if _active_tween != null:
		_active_tween.kill()
		_active_tween = null
