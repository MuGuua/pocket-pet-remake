extends Resource
class_name UnitSkin

## 单位形象资源：一个 .tres 对应一个可显示形象。
## 服务端下发 skin_id，客户端按 ID 加载本资源，同时获得动画帧与表现配置。

@export_group("标识")
## 形象唯一 ID，需与服务端字段一致（例如 "决斗者_001"）
@export var skin_id: String = ""

@export_group("动画")
## 待机、行走、攻击等序列帧，直接内嵌在本资源中编辑
@export var sprite_frames: SpriteFrames

@export_group("战斗动画")
## 战斗场景加载后的默认待机动画，仅 BattleUnit 使用，不影响主世界。
@export var default_animation: String = "待机"

## 逻辑动作名到具体动画名的映射，例如 {"普攻": "slash_01"}
@export var animation_map: Dictionary = {}

## 按动画名控制水平翻转，未列出的动画保持当前朝向不变。
@export var animation_flip_h: Dictionary = {}

@export_group("世界动画")
## 世界场景状态+朝向到 SpriteFrames 动画名的映射，例如 {"idle_down": "idle_down"}。
@export var world_animation_map: Dictionary = {}

## 世界场景碰撞圆相对脚底锚点的微调偏移。
@export var world_collision_offset: Vector2 = Vector2.ZERO

@export_group("外观")
## 脚底自动对齐后的微调偏移（一般保持 0）
@export var sprite_offset: Vector2 = Vector2.ZERO

## 精灵缩放
@export var sprite_scale: Vector2 = Vector2(1.6, 1.6)

## 状态图标/血条相对单位的偏移
@export var status_offset: Vector2 = Vector2(-90.0, -118.0)

## 整体染色，Color.WHITE 表示无染色
@export var tint: Color = Color.WHITE

func has_animation(animation_name: String) -> bool:
	if sprite_frames == null or animation_name.is_empty():
		return false
	return sprite_frames.has_animation(animation_name)

func resolve_animation(logical_name: String) -> String:
	var mapped_name: String = str(animation_map.get(logical_name, logical_name))
	if has_animation(mapped_name):
		return mapped_name
	if has_animation(logical_name):
		return logical_name
	if has_animation(default_animation):
		return default_animation
	return logical_name

## 解析世界场景动画名；缺少专用行走帧时回退到同朝向 idle，不再使用战斗 default_animation。
func resolve_world_animation(state: String, direction_suffix: String) -> String:
	var composed_key: String = "%s_%s" % [state, direction_suffix]
	if world_animation_map.has(composed_key):
		return str(world_animation_map[composed_key])
	if state == "walk":
		var idle_key: String = "idle_%s" % direction_suffix
		if world_animation_map.has(idle_key):
			return str(world_animation_map[idle_key])
	return resolve_world_bootstrap_animation()

## 主世界换肤后用于脚底对齐与首帧展示的动画，优先 idle_down。
func resolve_world_bootstrap_animation() -> String:
	var preferred_keys: Array[String] = ["idle_down", "idle_up", "idle_left", "idle_right"]
	for key: String in preferred_keys:
		if not world_animation_map.has(key):
			continue
		var animation_name: String = str(world_animation_map[key])
		if has_animation(animation_name):
			return animation_name
	return ""
