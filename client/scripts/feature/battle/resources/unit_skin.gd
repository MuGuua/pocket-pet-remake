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

@export_group("CHJ 世界渲染")
## 显式 CHJ 路径；留空则尝试 res://asset/chj/{skin_id}.chj。
@export_file("*.chj") var chj_path: String = ""
## CHJ 显示缩放。
@export var chj_display_scale: Vector2 = Vector2(2.0, 2.0)
## CHJ 行走动画速度（越大切换越快）。
@export var chj_walk_anim_speed: float = 120.0
## CHJ 待机动画速度。
@export var chj_idle_anim_speed: float = 60.0
## CHJ 取帧除数，对应 floor(tick / divisor)。
@export var chj_frame_divisor: float = 8.0
## 技能/普攻专用 CHJ；留空则尝试 res://asset/chj/{skin_id}_skill.chj。
@export_file("*.chj") var chj_skill_path: String = ""

@export_group("世界动画")
## 世界场景状态+朝向到 SpriteFrames 动画名的映射，例如 {"idle_down": "idle_down"}。
@export var world_animation_map: Dictionary = {}

## 进入战斗后在世界场景播放的专用动画名，例如 "战斗待机"；留空时回退到 battle_朝向 或同朝向 idle。
@export var world_battle_animation: String = ""

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

## 解析最终 CHJ 路径：显式 chj_path 优先，其次 res://asset/chj/{skin_id}.chj。
func resolve_chj_path() -> String:
	var explicit_path: String = chj_path.strip_edges()
	if not explicit_path.is_empty() and FileAccess.file_exists(explicit_path):
		return explicit_path
	if skin_id.is_empty():
		return ""
	var conventional_path: String = "res://asset/chj/%s.chj" % skin_id.strip_edges()
	if FileAccess.file_exists(conventional_path):
		return conventional_path
	return ""


## 是否已配置 PNG SpriteFrames，可用于按动画名局部覆盖 CHJ。
func has_configured_sprite_frames() -> bool:
	return sprite_frames != null


## 是否以 CHJ 作为世界行走/待机的基础渲染（有 chj 路径即可，可与 sprite_frames 局部覆盖并存）。
func uses_chj_world_render() -> bool:
	return not resolve_chj_path().is_empty()


## 若当前世界状态在 sprite_frames 中有对应覆盖动画，返回动画名；否则返回空字符串。
func get_world_png_override_animation(state: String, direction_suffix: String) -> String:
	if sprite_frames == null:
		return ""
	if state == "battle":
		var battle_animation: String = resolve_world_battle_animation(direction_suffix)
		if has_animation(battle_animation):
			return battle_animation
		return ""
	var composed_key: String = "%s_%s" % [state, direction_suffix]
	if not world_animation_map.has(composed_key):
		return ""
	var mapped_name: String = str(world_animation_map[composed_key])
	if has_animation(mapped_name):
		return mapped_name
	return ""


## 战斗场景待机：sprite_frames 中若配置了 default_animation 或 world_battle_animation 则覆盖 CHJ 战斗待机。
func get_battle_idle_png_override() -> String:
	if sprite_frames == null:
		return ""
	if not world_battle_animation.is_empty() and has_animation(world_battle_animation):
		return world_battle_animation
	if not default_animation.is_empty() and has_animation(default_animation):
		return default_animation
	return ""


## 战斗语义动画：仅当 animation_map 或逻辑名在 sprite_frames 中有精确匹配时返回 PNG 覆盖名。
## 不会回退到 default_animation，避免「战斗待机」误替换普攻/技能。
func get_battle_action_png_override(logical_name: String) -> String:
	if sprite_frames == null or logical_name.is_empty():
		return ""
	if animation_map.has(logical_name):
		var mapped_from_logical: String = str(animation_map[logical_name])
		if has_animation(mapped_from_logical):
			return mapped_from_logical
	if has_animation(logical_name):
		return logical_name
	return ""


## 解析技能 CHJ 路径：显式 chj_skill_path 优先，其次 res://asset/chj/{skin_id}_skill.chj。
func resolve_chj_skill_path() -> String:
	var explicit_path: String = chj_skill_path.strip_edges()
	if not explicit_path.is_empty() and FileAccess.file_exists(explicit_path):
		return explicit_path
	if skin_id.is_empty():
		return ""
	var conventional_path: String = "res://asset/chj/%s_skill.chj" % skin_id.strip_edges()
	if FileAccess.file_exists(conventional_path):
		return conventional_path
	return ""


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

## 解析世界场景战斗态动画名；优先 battle_朝向 映射，其次 world_battle_animation。
func resolve_world_battle_animation(direction_suffix: String) -> String:
	var direction_key: String = "battle_%s" % direction_suffix
	if world_animation_map.has(direction_key):
		var mapped_name: String = str(world_animation_map[direction_key])
		if has_animation(mapped_name):
			return mapped_name
	if not world_battle_animation.is_empty() and has_animation(world_battle_animation):
		return world_battle_animation
	var idle_key: String = "idle_%s" % direction_suffix
	if world_animation_map.has(idle_key):
		var idle_animation: String = str(world_animation_map[idle_key])
		if has_animation(idle_animation):
			return idle_animation
	return resolve_world_bootstrap_animation()

## 解析世界场景动画名；缺少专用行走帧时回退到同朝向 idle，不再使用战斗 default_animation。
func resolve_world_animation(state: String, direction_suffix: String) -> String:
	if state == "battle":
		return resolve_world_battle_animation(direction_suffix)
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


## 解析 HUD 头像等预览位使用的首帧动画；仅使用世界四方向待机，不回退战斗待机。
## CHJ 形象若无 PNG 待机映射，请改用 resolve_avatar_preview_texture()。
func resolve_avatar_preview_animation() -> String:
	var world_animation: String = resolve_world_bootstrap_animation()
	if not world_animation.is_empty():
		return world_animation
	var preferred_world_idle_names: Array[String] = ["待机下", "待机上", "待机左", "待机右"]
	for animation_name: String in preferred_world_idle_names:
		if has_animation(animation_name) and sprite_frames.get_frame_count(animation_name) > 0:
			return animation_name
	return ""


## 解析 HUD 头像贴图；优先 PNG 世界待机下，纯 CHJ 形象回退主文件 down idle 首帧。
func resolve_avatar_preview_texture() -> Texture2D:
	var animation_name: String = resolve_avatar_preview_animation()
	if not animation_name.is_empty() and sprite_frames != null:
		if sprite_frames.get_frame_count(animation_name) > 0:
			var png_texture: Texture2D = sprite_frames.get_frame_texture(animation_name, 0)
			if png_texture != null:
				return png_texture
	return _resolve_chj_avatar_preview_texture()


## 从 CHJ 主文件截取 down idle 首帧，供未配置 PNG 待机下的形象展示头像。
func _resolve_chj_avatar_preview_texture() -> Texture2D:
	var chj_resolved_path: String = resolve_chj_path()
	if chj_resolved_path.is_empty():
		return null
	var chj_sprite: ChjSprite = ChjSprite.load_from_path(chj_resolved_path)
	if chj_sprite == null:
		return null
	return chj_sprite.create_world_idle_preview_texture("down", 0)
