extends WorldPlayerCinematic

## 桃子在固定过场画面中的初始位置。
const TAOZI_START_POSITION: Vector2 = Vector2(86.0, 125.0)
## 七色羽在固定过场画面中的初始位置。
const QISEYU_START_POSITION: Vector2 = Vector2(114.0, 125.0)
## 玩家在固定过场画面中的初始位置。
const PLAYER_START_POSITION: Vector2 = Vector2(90.0, 240.0)
## NPC 资源共同使用的正面待机动画名。
const IDLE_DOWN_ANIMATION: StringName = &"待机下"
## 桃子资源中配置的上向待机动画名。
const IDLE_UP_ANIMATION: StringName = &"待机上"
## 桃子和七色羽资源中配置的左向待机动画名。
const IDLE_LEFT_ANIMATION: StringName = &"待机左"
## 桃子资源中配置的向下行走动画名。
const WALK_DOWN_ANIMATION: StringName = &"向下走"
## 桃子资源中配置的向右行走动画名。
const WALK_RIGHT_ANIMATION: StringName = &"向右走"
## 桃子资源中配置的向左行走动画名。
const WALK_LEFT_ANIMATION: StringName = &"向左走"
## 七色羽唯一可复用的行走动画名，通过镜像表现右向状态。
const QISEYU_WALK_ANIMATION: StringName = &"向左走"
## 玩家旧版动画与动态皮肤共同使用的向上行走动画名。
const PLAYER_WALK_UP_ANIMATION: String = "walk_up"
## 固定过场角色的像素移动速度。
const CINEMATIC_MOVE_SPEED: float = 100.0
## 结尾画面渐变为全黑所需的秒数。
const FADE_TO_BLACK_SECONDS: float = 1.2

## 当前固定过场复用的闪光镇北路地图节点。
@onready var _scene_level: Node2D = $BeiLu
## 桃子场景节点。
@onready var _taozi: Node2D = $桃子
## 桃子的序列帧动画节点。
@onready var _taozi_sprite: AnimatedSprite2D = $桃子/AnimatedSprite2D
## 七色羽场景节点。
@onready var _qiseyu: Node2D = $七色羽
## 七色羽的序列帧动画节点。
@onready var _qiseyu_sprite: AnimatedSprite2D = $七色羽/AnimatedSprite2D
## 当前固定过场使用的本地玩家节点。
@onready var _scene_player: CharacterBody2D = $Player
## 当前固定过场使用的本地相机节点。
@onready var _scene_camera: Camera2D = $Player/Camera2D
## 单独运行当前场景时使用的备用对话面板。
@onready var _standalone_dialogue_panel: NPCDialoguePanel = $StandaloneDialoguePanel
## 剧情播放期间显示在屏幕顶部的闪烁“剧情”图片层。
@onready var _plot_image: CanvasLayer = $PlotImage
## 结尾覆盖世界画面的黑色渐变节点；层级低于对话面板，保证内心独白可见。
@onready var _fade_to_black: ColorRect = $FadeOverlay/FadeToBlack


## 场景退出时清理剧情图片和黑屏，避免剧情中断后视觉状态残留。
func _exit_tree() -> void:
	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	if _fade_to_black != null:
		_fade_to_black.hide()
		_fade_to_black.modulate.a = 0.0
	super._exit_tree()


## 按剧情注释依次播放角色站位、移动、对白和结尾渐黑演出。
func _run_sequence() -> void:
	if _plot_image != null:
		_plot_image.call("show_plot_image")
	normalize_cinematic_level_origin(_scene_level)
	sync_cinematic_camera_with_world(_scene_camera, _scene_level)
	_prepare_scene_actors()

	await _show_actor_dialogue(
		"桃子",
		"春天，你拉着我看月亮圆圆的脸，将衣服轻轻披在我的肩上。我知道，这是我一生最快乐的一年。",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _show_actor_dialogue(
		"桃子",
		"夏天，你送我一句最美的誓言，把他写在沙滩上面，让浪花把我们最美的回忆带到海的心底。",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	await _move_scene_player(
		Vector2(0.0, -30.0),
		PLAYER_WALK_UP_ANIMATION,
		Vector2.UP
	)
	await _show_actor_dialogue(
		"玩家",
		"桃子，我已经拿到自己的战斗精灵了。我们回去吧？🙂",
		_get_scene_player_portrait(),
		true
	)

	await _move_taozi_toward_player()
	await _move_qiseyu(
		Vector2(0.0, 20.0),
		false,
		false
	)
	await _show_actor_dialogue(
		"桃子",
		"秋冬，你和我安静从容地捧着[color=green]比武大会[/color]冠军奖杯，为属于我们的一年画上一个完美的句号🙂",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _show_actor_dialogue(
		"玩家",
		"傻桃子，你在说什么啊？听得我一头雾水的……😳",
		_get_scene_player_portrait(),
		true
	)
	await _show_actor_dialogue(
		"桃子",
		"不读书的家伙，这是我们九泪公主写的散文《最快乐的一年》啊。嘿嘿，这也是我自己期望能够经历的浪漫岁月💗",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _show_actor_dialogue(
		"玩家",
		"散文啊，我的确读得不多……啊好啦，不说这个啦🤣明天我们还要出发去[color=blue]闪光平原[/color]呢，你不去收拾收拾东西吗？",
		_get_scene_player_portrait(),
		true
	)

	await _move_taozi_down(25.0)
	await _move_qiseyu(
		Vector2(-30.0, 0.0),
		false,
		false
	)
	await _move_qiseyu(
		Vector2(0.0, 25.0),
		true,
		true
	)
	await _show_actor_dialogue(
		"桃子",
		"知道啦……嘿嘿……\n🙂[color=orange]你听，风是不是在笑啊？[/color]💗",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(-20.0, 0.0),
		WALK_LEFT_ANIMATION,
		false
	)
	await _move_taozi_and_qiseyu_down()
	await _show_actor_dialogue(
		"桃子",
		"谢谢……谢谢你能陪我🙂[color=#F26BD5]我想……[/color][color=red]我……[/color]",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	await _fade_screen_to_black()
	_hide_cinematic_actors()
	await _show_actor_dialogue(
		"桃子",
		"（想）[color=#F26BD5]如果我想的一切都能顺利的实现……[/color]",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	_clear_black_screen()
	complete_cinematic()


## 重置所有剧情角色的初始站位、朝向和可见状态。
func _prepare_scene_actors() -> void:
	_taozi.position = TAOZI_START_POSITION
	_taozi.show()
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(IDLE_UP_ANIMATION)

	_qiseyu.position = QISEYU_START_POSITION
	_qiseyu.show()
	_qiseyu_sprite.flip_h = false
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)

	_scene_player.position = PLAYER_START_POSITION
	_scene_player.call("set_scene_transition_locked", true)
	_scene_player.call("set_facing_direction", Vector2.UP)

	_fade_to_black.modulate.a = 0.0
	_fade_to_black.hide()


## 让桃子先向下移动 40px，再向右移动 10px并切换为向下待机。
func _move_taozi_toward_player() -> void:
	await _move_taozi_down(40.0)
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(17.0, 0.0),
		WALK_RIGHT_ANIMATION,
		false
	)
	_taozi_sprite.play(IDLE_DOWN_ANIMATION)


## 让桃子向下移动指定距离并在结束后保持向下待机。
func _move_taozi_down(distance: float) -> void:
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(0.0, distance),
		WALK_DOWN_ANIMATION,
		false
	)
	_taozi_sprite.play(IDLE_DOWN_ANIMATION)


## 使用七色羽现有左向动画移动；mirror_right 为 true 时镜像表现右向状态。
func _move_qiseyu(
	relative_offset: Vector2,
	mirror_right: bool,
	finish_idle_right: bool
) -> void:
	await _move_actor(
		_qiseyu,
		_qiseyu_sprite,
		relative_offset,
		QISEYU_WALK_ANIMATION,
		mirror_right
	)
	_qiseyu_sprite.flip_h = finish_idle_right
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)


## 让桃子和七色羽同时向下移动 50px，并在结束后保持各自最终待机方向。
func _move_taozi_and_qiseyu_down() -> void:
	var relative_offset: Vector2 = Vector2(0.0, 60.0)
	var duration_seconds: float = relative_offset.length() / CINEMATIC_MOVE_SPEED
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(WALK_DOWN_ANIMATION)
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(QISEYU_WALK_ANIMATION)
	var parallel_tween: Tween = create_tween()
	parallel_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	parallel_tween.set_trans(Tween.TRANS_LINEAR)
	parallel_tween.set_parallel(true)
	parallel_tween.tween_property(
		_taozi,
		"position",
		_taozi.position + relative_offset,
		duration_seconds
	)
	parallel_tween.tween_property(
		_qiseyu,
		"position",
		_qiseyu.position + relative_offset,
		duration_seconds
	)
	await parallel_tween.finished
	_taozi_sprite.play(IDLE_DOWN_ANIMATION)
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)


## 逐渐提高黑色遮罩透明度，完成剧情结尾的全屏渐黑演出。
func _fade_screen_to_black() -> void:
	_fade_to_black.modulate.a = 0.0
	_fade_to_black.show()
	var fade_tween: Tween = create_tween()
	fade_tween.tween_property(
		_fade_to_black,
		"modulate:a",
		1.0,
		FADE_TO_BLACK_SECONDS
	)
	await fade_tween.finished


## 屏幕完全变黑后关闭桃子和七色羽的显示，避免取消黑屏时角色短暂闪现。
func _hide_cinematic_actors() -> void:
	_taozi.hide()
	_qiseyu.hide()


## 内心独白结束后立即取消黑色遮罩，恢复剧情场景的正常画面。
func _clear_black_screen() -> void:
	_fade_to_black.hide()
	_fade_to_black.modulate.a = 0.0


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


## 使用场景角色的上半身形象展示固定对白。
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
