extends WorldPlayerCinematic

## 宠物教师薇安在固定过场画面中的初始位置。
const VIANNE_START_POSITION: Vector2 = Vector2(140.0, 122.0)
## 桃子在固定过场画面中的初始位置。
const TAOZI_START_POSITION: Vector2 = Vector2(191.0, 122.0)
## 玩家在固定过场画面中的初始位置。
const PLAYER_START_POSITION: Vector2 = Vector2(270.0, 67.0)
## NPC 资源共同使用的正面待机动画名。
const IDLE_DOWN_ANIMATION: StringName = &"待机下"
## 桃子资源中配置的左向待机动画名。
const IDLE_LEFT_ANIMATION: StringName = &"待机左"
## 桃子资源中配置的右向待机动画名。
const IDLE_RIGHT_ANIMATION: StringName = &"待机右"
## 桃子资源中配置的右向行走动画名。
const WALK_RIGHT_ANIMATION: StringName = &"向右走"
## 桃子资源中配置的上向行走动画名。
const WALK_UP_ANIMATION: StringName = &"向上走"
## 玩家旧版动画与动态皮肤共同使用的向下行走动画名。
const PLAYER_WALK_DOWN_ANIMATION: String = "walk_down"
## 玩家旧版动画与动态皮肤共同使用的向左行走动画名。
const PLAYER_WALK_LEFT_ANIMATION: String = "walk_left"
## 固定过场角色的像素移动速度，用于根据距离统一计算移动时长。
const CINEMATIC_MOVE_SPEED: float = 100.0

## 当前固定过场复用的宠物学校地图节点。
@onready var _scene_level: Node2D = $XueXiao
## 桃子场景节点。
@onready var _taozi: Node2D = $桃子
## 桃子的序列帧动画节点。
@onready var _taozi_sprite: AnimatedSprite2D = $桃子/AnimatedSprite2D
## 当前固定过场使用的本地玩家节点。
@onready var _scene_player: CharacterBody2D = $Player
## 当前固定过场使用的本地相机节点。
@onready var _scene_camera: Camera2D = $Player/Camera2D
## 宠物教师薇安场景节点。
@onready var _vianne: Node2D = $宠物教师薇安
## 宠物教师薇安的序列帧动画节点。
@onready var _vianne_sprite: AnimatedSprite2D = $宠物教师薇安/AnimatedSprite2D
## 单独运行当前场景时使用的备用对话面板；正式客户端仍复用主场景对话面板。
@onready var _standalone_dialogue_panel: NPCDialoguePanel = $StandaloneDialoguePanel
## 剧情播放期间显示在屏幕顶部的闪烁“剧情”图片层。
@onready var _plot_image: CanvasLayer = $PlotImage


## 场景退出时关闭剧情图片，避免剧情被中断后图片残留。
func _exit_tree() -> void:
	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	super._exit_tree()


## 按既定站位、移动与对白顺序播放宠物学校固定剧情。
func _run_sequence() -> void:
	if _plot_image != null:
		_plot_image.call("show_plot_image")
	normalize_cinematic_level_origin(_scene_level)
	sync_cinematic_camera_with_world(_scene_camera, _scene_level)
	_prepare_scene_actors()

	await _show_actor_dialogue(
		"桃子",
		"嗯🙂，那就这样啦，拜托您啦。",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	# 玩家先向下进入画面，移动结束后桃子再向右迎接玩家。
	await _move_player_and_taozi_to_meet()

	await _show_actor_dialogue(
		"桃子",
		"啊……你来的好快啊，已经搞定武器和衣服了吗？😜",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _show_actor_dialogue(
		"桃子",
		"我已经和[color=yellow]宠物教师·薇安[/color]说好啦，她会给你一只🐾宠物的，宠物可是战斗的重要助手，一定要对它好点哦！",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _show_actor_dialogue(
		"玩家",
		"😵‍💫……唔，不过我的宠物得训练多久才能赶上你的宠物[color=yellow]七色羽[/color]啊……",
		_get_scene_player_portrait(),
		true
	)
	await _show_actor_dialogue(
		"桃子",
		"嘿嘿……[color=yellow]七色羽[/color]是不可超越的😛那我还是去老地方等你啦～",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	await _move_taozi_out_of_scene()
	await _move_scene_player(
		Vector2(-120.0, 0.0),
		PLAYER_WALK_LEFT_ANIMATION,
		Vector2.LEFT
	)
	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	complete_cinematic()


## 重置所有剧情角色的初始站位、朝向和可见状态。
func _prepare_scene_actors() -> void:
	_vianne.position = VIANNE_START_POSITION
	_vianne.show()
	_vianne_sprite.flip_h = false
	_vianne_sprite.play(IDLE_DOWN_ANIMATION)

	_taozi.position = TAOZI_START_POSITION
	_taozi.show()
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(IDLE_LEFT_ANIMATION)

	_scene_player.position = PLAYER_START_POSITION
	_scene_player.call("set_scene_transition_locked", true)
	_scene_player.call("set_facing_direction", Vector2.DOWN)


## 让玩家先向下移动 60px并面向左侧，完成后桃子再向右移动 50px。
func _move_player_and_taozi_to_meet() -> void:
	await _move_scene_player(
		Vector2(0.0, 55.0),
		PLAYER_WALK_DOWN_ANIMATION,
		Vector2.LEFT
	)
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(50.0, 0.0),
		WALK_RIGHT_ANIMATION,
		false
	)
	_taozi_sprite.play(IDLE_RIGHT_ANIMATION)


## 让桃子先向右移动 20px，再向上移动 70px并离开画面。
func _move_taozi_out_of_scene() -> void:
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(20.0, 0.0),
		WALK_RIGHT_ANIMATION,
		false
	)
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(0.0, -70.0),
		WALK_UP_ANIMATION,
		false
	)
	_taozi.hide()


## 展示一条固定对白；正式客户端走 CinematicPlayer，单独运行时使用场景内备用面板。
func show_local_dialogue(
	speaker_name: String,
	content: String,
	portrait_key: String = "",
	is_player_speaking: bool = false,
	content_format: String = "bbcode",
	portrait_texture: Texture2D = null
) -> void:
	var external_connections: Array[Dictionary] = get_signal_connection_list(&"local_dialogue_requested")
	if not external_connections.is_empty():
		await super.show_local_dialogue(
			speaker_name,
			content,
			portrait_key,
			is_player_speaking,
			content_format,
			portrait_texture
		)
		return
	if _standalone_dialogue_panel == null:
		return
	_standalone_dialogue_panel.show_local_dialogue(
		speaker_name,
		content,
		portrait_key,
		is_player_speaking,
		content_format,
		portrait_texture
	)
	await _standalone_dialogue_panel.local_continue_requested
	_standalone_dialogue_panel.hide_panel(false)


## 使用场景角色的正面上半身形象展示固定对白。
func _show_actor_dialogue(
	speaker_name: String,
	content: String,
	portrait_texture: Texture2D,
	is_player_speaking: bool = false
) -> void:
	await show_local_dialogue(
		speaker_name,
		content,
		"",
		is_player_speaking,
		"bbcode",
		portrait_texture
	)


## 从角色序列帧中提取正面待机帧，并裁切成对话框使用的上半身头像。
func _get_animated_sprite_portrait(sprite: AnimatedSprite2D) -> Texture2D:
	if sprite == null or sprite.sprite_frames == null:
		return null
	var animation_name: StringName = IDLE_DOWN_ANIMATION
	if not sprite.sprite_frames.has_animation(animation_name):
		animation_name = sprite.animation
	if not sprite.sprite_frames.has_animation(animation_name):
		return null
	var frame_count: int = sprite.sprite_frames.get_frame_count(animation_name)
	if frame_count <= 0:
		return null
	var frame_texture: Texture2D = sprite.sprite_frames.get_frame_texture(animation_name, 0)
	return _crop_dialogue_upper_body(frame_texture)


## 获取剧情玩家的正面上半身头像；动态皮肤优先，旧版精灵图作为兜底。
func _get_scene_player_portrait() -> Texture2D:
	var character_visual: Node2D = _scene_player.get_node_or_null("CharacterVisual") as Node2D
	if character_visual != null and character_visual.visible:
		var animated_sprite: AnimatedSprite2D = character_visual.get_node_or_null("AnimatedSprite2D") as AnimatedSprite2D
		if animated_sprite != null and animated_sprite.visible:
			return _get_animated_sprite_portrait(animated_sprite)
	var registered_portrait: Texture2D = PortraitRegistry.load_player_dialogue_portrait()
	if registered_portrait != null:
		return registered_portrait
	var legacy_sprite: Sprite2D = _scene_player.get_node_or_null("Sprite2D") as Sprite2D
	return _get_sprite_frame_portrait(legacy_sprite)


## 从普通 Sprite2D 图集的当前帧创建独立头像纹理。
func _get_sprite_frame_portrait(sprite: Sprite2D) -> Texture2D:
	if sprite == null or sprite.texture == null:
		return null
	var horizontal_frames: int = maxi(sprite.hframes, 1)
	var vertical_frames: int = maxi(sprite.vframes, 1)
	if horizontal_frames == 1 and vertical_frames == 1:
		return _crop_dialogue_upper_body(sprite.texture)
	var frame_size: Vector2 = Vector2(
		float(sprite.texture.get_width()) / float(horizontal_frames),
		float(sprite.texture.get_height()) / float(vertical_frames)
	)
	var frame_column: int = sprite.frame % horizontal_frames
	var frame_row: int = int(sprite.frame / horizontal_frames)
	var atlas_texture: AtlasTexture = AtlasTexture.new()
	atlas_texture.atlas = sprite.texture
	atlas_texture.region = Rect2(
		Vector2(float(frame_column), float(frame_row)) * frame_size,
		frame_size
	)
	return _crop_dialogue_upper_body(atlas_texture)


## 把固定过场对白头像统一裁切成上半身。
func _crop_dialogue_upper_body(source_texture: Texture2D) -> Texture2D:
	if source_texture == null:
		return null
	return PortraitRegistry.crop_upper_body_portrait(source_texture)


## 让单个本地剧情角色按固定偏移线性移动，并播放对应行走动画。
func _move_actor(
	actor: Node2D,
	actor_sprite: AnimatedSprite2D,
	relative_offset: Vector2,
	walk_animation: StringName,
	flip_h: bool
) -> void:
	actor_sprite.flip_h = flip_h
	actor_sprite.play(walk_animation)
	var duration_seconds: float = maxf(relative_offset.length() / CINEMATIC_MOVE_SPEED, 0.01)
	var move_tween: Tween = create_tween()
	move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	move_tween.set_trans(Tween.TRANS_LINEAR)
	move_tween.tween_property(
		actor,
		"position",
		actor.position + relative_offset,
		duration_seconds
	)
	await move_tween.finished


## 让剧情场景玩家按固定偏移移动，并在移动结束后切换到指定朝向。
func _move_scene_player(
	relative_offset: Vector2,
	walk_animation: String,
	final_facing_direction: Vector2
) -> void:
	_scene_player.call("set_facing_direction", relative_offset)
	_scene_player.call("play_cinematic_animation", walk_animation, -1, 12.0)
	var duration_seconds: float = maxf(relative_offset.length() / CINEMATIC_MOVE_SPEED, 0.01)
	var move_tween: Tween = create_tween()
	move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	move_tween.set_trans(Tween.TRANS_LINEAR)
	move_tween.tween_property(
		_scene_player,
		"position",
		_scene_player.position + relative_offset,
		duration_seconds
	)
	await move_tween.finished
	_scene_player.call("set_facing_direction", final_facing_direction)
