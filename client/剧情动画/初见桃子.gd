extends WorldPlayerCinematic

## 桃子在固定过场画面中的初始位置。
const TAOZI_START_POSITION: Vector2 = Vector2(147.0, 135.0)
## 七色羽在固定过场画面中的初始位置。
const QISEYU_START_POSITION: Vector2 = Vector2(115.0, 130.0)
## 玩家在固定过场画面中的初始位置。
const PLAYER_START_POSITION: Vector2 = Vector2(212.0, 135.0)
## 桃子和七色羽资源中配置的左向待机动画名。
const IDLE_LEFT_ANIMATION: StringName = &"待机左"
## 对话半身像优先固定使用的正面待机动画名，避免角色移动时头像跟着转向。
const IDLE_DOWN_ANIMATION: StringName = &"待机下"
## 桃子资源中配置的右向待机动画名。
const IDLE_RIGHT_ANIMATION: StringName = &"待机右"
## 桃子资源中配置的上向待机动画名。
const IDLE_UP_ANIMATION: StringName = &"待机上"
## 桃子和七色羽资源中配置的左向行走动画名。
const WALK_LEFT_ANIMATION: StringName = &"向左走"
## 桃子资源中配置的右向行走动画名。
const WALK_RIGHT_ANIMATION: StringName = &"向右走"
## 桃子资源中配置的上向行走动画名。
const WALK_UP_ANIMATION: StringName = &"向上走"
## 玩家旧版 AnimationPlayer 和动态皮肤共同使用的左向行走动画名。
const PLAYER_WALK_LEFT_ANIMATION: String = "walk_left"
## 固定过场角色的像素移动速度，用于根据距离统一计算 Tween 时长。
const CINEMATIC_MOVE_SPEED: float = 100.0
## 冲击波三个序列帧节点共同使用的动画名。
const SHOCKWAVE_ANIMATION: StringName = &"default"
## 冲击波相对七色羽终点位置左移 30px、上移 40px 的初始偏移。
const SHOCKWAVE_START_OFFSET: Vector2 = Vector2(-30.0, -40.0)
## 冲击波播放期间整体向左移动的距离。
const SHOCKWAVE_TRAVEL_OFFSET: Vector2 = Vector2(-100.0, 0.0)
## 冲击波从出现到消失的固定持续秒数。
const SHOCKWAVE_DURATION_SECONDS: float = 2.0
## 两名角色转身后到冲击波出现前的固定等待秒数。
const SHOCKWAVE_START_DELAY_SECONDS: float = 1.0
## 冲击波序列帧固定循环播放帧率。
const SHOCKWAVE_FPS: float = 6.0
## 技能特效出现时全黑画面保持的秒数。
const IMPACT_BLACK_HOLD_SECONDS: float = 0.06
## 黑屏退回透明的秒数。
const IMPACT_BLACK_FADE_SECONDS: float = 0.08
## 每段相机震动的秒数，短促反馈避免影响移动端观看对白。
const IMPACT_SHAKE_STEP_SECONDS: float = 0.045
## 相机震动偏移序列，最后必须回到零偏移以恢复原有构图。
const IMPACT_SHAKE_OFFSETS: Array[Vector2] = [
	Vector2(-6.0, 3.0),
	Vector2(8.0, -4.0),
	Vector2(-5.0, 2.0),
	Vector2.ZERO,
]

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
## 当前固定过场复用的闪光镇东路地图节点。
@onready var _scene_level: Node2D = $EastRoadOfShanguangTown
## 单独运行当前场景时使用的备用对话面板；正式客户端仍复用主场景对话面板。
@onready var _standalone_dialogue_panel: NPCDialoguePanel = $StandaloneDialoguePanel
## 剧情开始后显示的图片提示层，内部脚本只负责闪烁图片本身。
@onready var _plot_image: CanvasLayer = $PlotImage
## 冲击波整体移动与显隐使用的根节点。
@onready var _shockwave_root: Node2D = $冲击波
## 冲击波上层序列帧节点。
@onready var _shockwave_top: AnimatedSprite2D = $冲击波/冲击波上
## 冲击波中层序列帧节点。
@onready var _shockwave_middle: AnimatedSprite2D = $冲击波/冲击波中
## 冲击波下层序列帧节点。
@onready var _shockwave_bottom: AnimatedSprite2D = $冲击波/冲击波下
## 技能出现瞬间覆盖整个过场画面的黑闪层。
@onready var _impact_flash: ColorRect = $ImpactOverlay/ImpactFlash

## 过场相机同步世界相机后的基础偏移，震动结束或剧情中断时必须恢复。
var _scene_camera_base_offset: Vector2 = Vector2.ZERO


## 场景退出时关闭剧情图片，避免剧情被打断后图片残留。
func _exit_tree() -> void:
	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	if _impact_flash != null:
		_impact_flash.hide()
	if _scene_camera != null:
		_scene_camera.offset = _scene_camera_base_offset
	super._exit_tree()


## 剧情动画脚本，动画介绍。
func _run_sequence() -> void:
	if _plot_image != null:
		_plot_image.call("show_plot_image")
	normalize_cinematic_level_origin(_scene_level)
	sync_cinematic_camera_with_world(_scene_camera, _scene_level)
	_scene_camera_base_offset = _scene_camera.offset

	# 人物位置摆放：桃子(147,135)，播放待机左动画；七色羽(115,130)，播放水平镜像后的待机左；玩家(212,135)，播放待机左动画。
	_taozi.position = TAOZI_START_POSITION
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(IDLE_LEFT_ANIMATION)

	_qiseyu.position = QISEYU_START_POSITION
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)

	_scene_player.position = PLAYER_START_POSITION
	_scene_player.call("set_scene_transition_locked", true)
	_scene_player.call("set_facing_direction", Vector2.LEFT)

	await get_tree().create_timer(SHOCKWAVE_START_DELAY_SECONDS).timeout

	# 七色羽：啾啾～～啾～
	await _show_actor_dialogue("七色羽", "啾啾～～啾～", _get_animated_sprite_portrait(_qiseyu_sprite))

	# 桃子转身，向右走到玩家前：桃子向右移动 50px，同时播放向右走动画。
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(40.0, 0.0),
		WALK_RIGHT_ANIMATION,
		false
	)
	_taozi_sprite.play(IDLE_RIGHT_ANIMATION)

	# 桃子：你怎么这么慢慢吞吞的啊？是不是想人[color=yellow]七色羽[/color]给你点厉害尝尝～～
	await _show_actor_dialogue("桃子", "你怎么这么慢慢吞吞的啊？是不是想人[color=yellow]七色羽[/color]给你点厉害尝尝～～", _get_animated_sprite_portrait(_taozi_sprite))

	# 玩家：别逗了！我会被[color=yellow]七色羽[/color]的火焰烤熟的！
	await _show_actor_dialogue("玩家", "别逗了！我会被[color=yellow]七色羽[/color]的火焰烤熟的！", _get_scene_player_portrait(), true)

	# 玩家：明天你就要去[color=orange]比武大会[/color]了，你们准备的怎么样了？有可能拿到名次不？
	await _show_actor_dialogue("玩家", "明天你就要去[color=orange]比武大会[/color]了，你们准备的怎么样了？有可能拿到名次不？", _get_scene_player_portrait(), true)

	# 桃子：挨挨.......你也太小看我了吧？让你看看我新训练的[color=yellow]七色羽[/color]的大招吧，[color=red]凤凰神炎[/color]！
	await _show_actor_dialogue("桃子", "挨挨.......你也太小看我了吧？让你看看我新训练的[color=yellow]七色羽[/color]的大招吧，[color=red]凤凰神炎[/color]！", _get_animated_sprite_portrait(_taozi_sprite))

	# 桃子转身，走回到初始位置：桃子向左移动 50px，同时播放向左走动画。
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(-40.0, 0.0),
		WALK_LEFT_ANIMATION,
		false
	)
	_taozi_sprite.play(IDLE_LEFT_ANIMATION)

	# 七色羽：啾啾～～啾～啾～
	await _show_actor_dialogue("七色羽", "啾啾～～啾～啾～", _get_animated_sprite_portrait(_qiseyu_sprite))

	# 桃子转向上，向上移动 18px。
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(0.0, -18.0),
		WALK_UP_ANIMATION,
		false
	)
	_taozi_sprite.play(IDLE_UP_ANIMATION)

	# 桃子和七色羽同时向右移动 75px；七色羽使用水平镜像后的向左走动画表现向右移动。
	await _move_taozi_and_qiseyu_right()

	# 移动完成后桃子和七色羽同时转身，恢复左向待机。
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(IDLE_LEFT_ANIMATION)
	_qiseyu_sprite.flip_h = false
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)

	# 转身完成后播放冲击波：初始位置在七色羽左侧 50px，播放期间整体向左移动 100px，结束后消失。
	await _play_shockwave()

	# 七色羽向上移动 18px；资源没有独立右向动画，因此镜像播放左向行走，视觉上保持向右。
	await _move_actor(
		_qiseyu,
		_qiseyu_sprite,
		Vector2(0.0, -18.0),
		WALK_LEFT_ANIMATION,
		true
	)
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)

	# 七色羽：啾啾～～啾～啾～
	await _show_actor_dialogue(
		"七色羽",
		"啾啾～～啾～啾～",
		_get_animated_sprite_portrait(_qiseyu_sprite)
	)

	# 桃子向左移动 10px，到达后保持左向待机。
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(-10.0, 0.0),
		WALK_LEFT_ANIMATION,
		false
	)
	_taozi_sprite.play(IDLE_LEFT_ANIMATION)

	# 桃子：真棒！我就知道[color=yellow]七色羽[/color]你绝对是这片大陆上最强的宠物精灵！咱们一定会再[color=orange]比武大会[/color]上获得冠军的
	await _show_actor_dialogue(
		"桃子",
		"真棒！我就知道[color=yellow]七色羽[/color]你绝对是这片大陆上最强的宠物精灵！咱们一定会再[color=orange]比武大会[/color]上获得冠军的",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	# 玩家向左移动 10px 的同时，桃子面朝下；玩家移动结束后面朝上。
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(IDLE_DOWN_ANIMATION)
	await _move_scene_player(
		Vector2(-10.0, 0.0),
		PLAYER_WALK_LEFT_ANIMATION,
		Vector2.UP
	)

	# 玩家：原来你们已经达到这种级别的实力了！太厉害了，[color=orange]比武大会[/color]的时候我会去给你加油的！
	await _show_actor_dialogue(
		"玩家",
		"原来你们已经达到这种级别的实力了！太厉害了，[color=orange]比武大会[/color]的时候我会去给你加油的！",
		_get_scene_player_portrait(),
		true
	)

	# 桃子：嗯？加油？
	await _show_actor_dialogue(
		"桃子",
		"嗯？加油？",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	# 桃子：什么加油？你在说什么啊？你也得和我一起去参加[color=orange]比武大会[/color]。你也得参加，[color=green]和我一起参赛[/color]！明白吗？
	await _show_actor_dialogue(
		"桃子",
		"什么加油？你在说什么啊？你也得和我一起去参加[color=orange]比武大会[/color]。你也得参加，[color=green]和我一起参赛[/color]！明白吗？",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	# 玩家：啊！？我......我也要参赛吗？你是和我开玩笑吧......
	await _show_actor_dialogue(
		"玩家",
		"啊！？我......我也要参赛吗？你是和我开玩笑吧......",
		_get_scene_player_portrait(),
		true
	)
	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	complete_cinematic()


## 展示一条固定对白；正式客户端走 CinematicPlayer，单独运行时改用场景内备用面板。
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


## 使用场景角色当前帧展示固定对白，避免回退到全局头像注册表中的默认形象。
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


## 从 AnimatedSprite2D 提取固定正面待机半身像；没有待机下时才回退当前动画帧。
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
	var frame_index: int = 0
	if animation_name == sprite.animation:
		frame_index = clampi(sprite.frame, 0, frame_count - 1)
	var frame_texture: Texture2D = sprite.sprite_frames.get_frame_texture(animation_name, frame_index)
	return _crop_dialogue_upper_body(frame_texture)


## 获取剧情场景中玩家固定正面待机半身像；动态皮肤优先，旧版精灵图仅做兜底。
func _get_scene_player_portrait() -> Texture2D:
	var character_visual: Node2D = _scene_player.get_node_or_null("CharacterVisual") as Node2D
	if character_visual != null and character_visual.visible:
		var animated_sprite: AnimatedSprite2D = character_visual.get_node_or_null("AnimatedSprite2D") as AnimatedSprite2D
		if animated_sprite != null and animated_sprite.visible:
			return _get_animated_sprite_portrait(animated_sprite)
		var chj_sprite: Sprite2D = character_visual.get_node_or_null("ChjSprite2D") as Sprite2D
		if chj_sprite != null and chj_sprite.visible and chj_sprite.texture != null:
			var registered_portrait: Texture2D = PortraitRegistry.load_player_dialogue_portrait()
			if registered_portrait != null:
				return registered_portrait
			return _crop_dialogue_upper_body(chj_sprite.texture)
	var player_portrait: Texture2D = PortraitRegistry.load_player_dialogue_portrait()
	if player_portrait != null:
		return player_portrait
	var legacy_sprite: Sprite2D = _scene_player.get_node_or_null("Sprite2D") as Sprite2D
	return _get_sprite_frame_portrait(legacy_sprite)


## 将普通 Sprite2D 图集的当前 frame 裁成独立 AtlasTexture，供对话角标按头像尺寸渲染。
func _get_sprite_frame_portrait(sprite: Sprite2D) -> Texture2D:
	if sprite == null or sprite.texture == null:
		return null
	var horizontal_frames: int = maxi(sprite.hframes, 1)
	var vertical_frames: int = maxi(sprite.vframes, 1)
	if horizontal_frames == 1 and vertical_frames == 1:
		return sprite.texture
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


## 统一把固定过场对白头像裁成上半身，避免对话角标显示完整小人形象。
func _crop_dialogue_upper_body(source_texture: Texture2D) -> Texture2D:
	if source_texture == null:
		return null
	return PortraitRegistry.crop_upper_body_portrait(source_texture)


## 让单个本地剧情角色按固定偏移线性移动，并在移动期间播放指定动画。
func _move_actor(
	actor: Node2D,
	actor_sprite: AnimatedSprite2D,
	relative_offset: Vector2,
	walk_animation: StringName,
	flip_h: bool
) -> void:
	actor_sprite.flip_h = flip_h
	actor_sprite.play(walk_animation)
	var move_distance: float = relative_offset.length()
	var duration_seconds: float = maxf(move_distance / CINEMATIC_MOVE_SPEED, 0.01)
	var target_position: Vector2 = actor.position + relative_offset
	var move_tween: Tween = create_tween()
	move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	move_tween.set_trans(Tween.TRANS_LINEAR)
	move_tween.tween_property(actor, "position", target_position, duration_seconds)
	await move_tween.finished


## 让剧情场景中的玩家按固定偏移移动，移动结束后切换到指定朝向的待机状态。
func _move_scene_player(
	relative_offset: Vector2,
	walk_animation: String,
	final_facing_direction: Vector2
) -> void:
	_scene_player.call("set_facing_direction", relative_offset)
	_scene_player.call("play_cinematic_animation", walk_animation, -1, 12.0)
	var move_distance: float = relative_offset.length()
	var duration_seconds: float = maxf(move_distance / CINEMATIC_MOVE_SPEED, 0.01)
	var target_position: Vector2 = _scene_player.position + relative_offset
	var move_tween: Tween = create_tween()
	move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	move_tween.set_trans(Tween.TRANS_LINEAR)
	move_tween.tween_property(
		_scene_player,
		"position",
		target_position,
		duration_seconds
	)
	await move_tween.finished
	_scene_player.call("set_facing_direction", final_facing_direction)


## 让桃子和七色羽以相同速度并行向右移动，保证两者同时到达终点。
func _move_taozi_and_qiseyu_right() -> void:
	var relative_offset: Vector2 = Vector2(60.0, 0.0)
	var duration_seconds: float = relative_offset.length() / CINEMATIC_MOVE_SPEED
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(WALK_RIGHT_ANIMATION)
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(WALK_LEFT_ANIMATION)
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


## 重置并播放三层冲击波动画，同时让冲击波根节点向左移动，完成后统一隐藏。
func _play_shockwave() -> void:
	await get_tree().create_timer(SHOCKWAVE_START_DELAY_SECONDS).timeout
	var shockwave_sprites: Array[AnimatedSprite2D] = [
		_shockwave_top,
		_shockwave_middle,
		_shockwave_bottom
	]
	_shockwave_root.position = _qiseyu.position + SHOCKWAVE_START_OFFSET
	_shockwave_root.visible = true
	for shockwave_sprite: AnimatedSprite2D in shockwave_sprites:
		shockwave_sprite.stop()
		shockwave_sprite.frame = 0
		shockwave_sprite.frame_progress = 0.0
		if shockwave_sprite.sprite_frames != null:
			shockwave_sprite.sprite_frames.set_animation_speed(
				SHOCKWAVE_ANIMATION,
				SHOCKWAVE_FPS
			)
			shockwave_sprite.sprite_frames.set_animation_loop(
				SHOCKWAVE_ANIMATION,
				true
			)
		shockwave_sprite.play(SHOCKWAVE_ANIMATION)
	var target_position: Vector2 = _shockwave_root.position + SHOCKWAVE_TRAVEL_OFFSET
	var shockwave_tween: Tween = create_tween()
	shockwave_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	shockwave_tween.set_trans(Tween.TRANS_LINEAR)
	shockwave_tween.tween_property(
		_shockwave_root,
		"position",
		target_position,
		SHOCKWAVE_DURATION_SECONDS
	)
	await _play_shockwave_impact_feedback()
	await shockwave_tween.finished
	for shockwave_sprite: AnimatedSprite2D in shockwave_sprites:
		shockwave_sprite.stop()
	_shockwave_root.visible = false


## 冲击波第一帧出现后先短暂闪黑，再震动剧情相机，并恢复原始画面状态。
func _play_shockwave_impact_feedback() -> void:
	if _impact_flash != null:
		_impact_flash.modulate = Color.WHITE
		_impact_flash.show()
		await get_tree().create_timer(IMPACT_BLACK_HOLD_SECONDS).timeout
		var flash_tween: Tween = create_tween()
		flash_tween.tween_property(
			_impact_flash,
			"modulate:a",
			0.0,
			IMPACT_BLACK_FADE_SECONDS
		)
		await flash_tween.finished
		_impact_flash.hide()
		_impact_flash.modulate = Color.WHITE
	if _scene_camera == null:
		return
	for shake_offset: Vector2 in IMPACT_SHAKE_OFFSETS:
		var shake_tween: Tween = create_tween()
		shake_tween.tween_property(
			_scene_camera,
			"offset",
			_scene_camera_base_offset + shake_offset,
			IMPACT_SHAKE_STEP_SECONDS
		)
		await shake_tween.finished
	_scene_camera.offset = _scene_camera_base_offset
