extends Resource
class_name SkillVisualConfig

## 技能表现资源：一个 .tres 对应一套完整技能视觉效果。
## 服务端下发 skill_visual_id，客户端按 ID 加载本资源，同时获得角色动作键与内嵌特效帧。

@export_group("标识")
## 技能表现唯一 ID，需与服务端字段一致（例如 "刃光斩_一级"）
@export var skill_visual_id: String = ""

@export_group("界面")
## 技能按钮和技能详情使用的本地图标；服务端只需下发 skill_visual_id。
@export var icon: Texture2D = null

@export_group("动作")
## 角色侧动画映射键，对应 UnitSkin.animation_map 的键名
@export var animation_key: String = ""

## 动作类型，如 skill / attack / buff
@export var action_type: String = ""

## 调试或 JSON 回退用的特效名称
@export var effect_id: String = ""

@export_group("特效")
## 内嵌于本资源的特效序列帧；优先于 effect_texture
@export var effect_frames: SpriteFrames

## 单帧静态特效贴图，仅在 effect_frames 为空时使用
@export var effect_texture: Texture2D

## effect_frames 中要播放的动画名
@export var effect_animation_key: String = "默认"

## 特效相对角色锚点的偏移
@export var effect_offset: Vector2 = Vector2.ZERO

## 特效整体缩放，作用于序列帧节点；冲击/爆发等内置动画只叠加在 MotionRoot 上。
@export var effect_scale: Vector2 = Vector2.ONE

## 特效染色
@export var effect_tint: Color = Color.WHITE

## BattleEffect 节点上 AnimationPlayer 播放的内置动画名
@export var effect_player_animation: String = "effect_default"

func resolve_effect_frames() -> SpriteFrames:
	if effect_frames != null:
		return effect_frames
	if effect_texture == null:
		return null
	var frames: SpriteFrames = SpriteFrames.new()
	frames.add_animation("默认")
	frames.set_animation_loop("默认", false)
	frames.set_animation_speed("默认", 8.0)
	frames.add_frame("默认", effect_texture)
	return frames

func resolve_effect_animation_name() -> String:
	var resolved_animation_key: String = effect_animation_key.strip_edges()
	if resolved_animation_key.is_empty():
		resolved_animation_key = "默认"
	var frames: SpriteFrames = resolve_effect_frames()
	if frames != null and frames.has_animation(resolved_animation_key):
		return resolved_animation_key
	return "默认"
