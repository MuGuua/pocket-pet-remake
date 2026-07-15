extends WorldPlayerCinematic

## 桃子在东路过场中的初始位置。
const TAOZI_START_POSITION: Vector2 = Vector2(115.0, 130.0)
## 七色羽在东路过场中的初始位置。
const QISEYU_START_POSITION: Vector2 = Vector2(130.0, 140.0)
## 玩家在东路过场中的初始位置。
const PLAYER_START_POSITION: Vector2 = Vector2(17.0, 170.0)
## 机械熊在东路过场中的初始位置。
const BEAR_START_POSITION: Vector2 = Vector2(215.0, 121.0)
## 切换到罗克萨斯家后玩家的显示位置。
const WAREHOUSE_PLAYER_POSITION: Vector2 = Vector2(115.0, 180.0)
## 固定过场角色移动速度，调整为原速度的一半。
const CINEMATIC_MOVE_SPEED: float = 80.0
## NPC 与玩家剧情动画播放倍率，统一调整为原速度的一半。
const CINEMATIC_ANIMATION_SPEED_SCALE: float = 0.5
## 普通渐黑和渐亮所需秒数。
const FADE_SECONDS: float = 0.7
## 单次黑屏闪烁显示或隐藏相位持续秒数。
const FLASH_PHASE_SECONDS: float = 0.07
## 最终能量爆发重复闪烁次数。
const FINAL_FLASH_COUNT: int = 4
## 相机每段震动持续秒数。
const SHAKE_STEP_SECONDS: float = 0.055
## 机械熊与七色羽最终启用的纹理模糊半径。
const FINAL_BLUR_RADIUS: float = 1.0

## 角色动画名称。
const IDLE_LEFT_ANIMATION: StringName = &"待机左"
const IDLE_RIGHT_ANIMATION: StringName = &"待机右"
const IDLE_UP_ANIMATION: StringName = &"待机上"
const WALK_LEFT_ANIMATION: StringName = &"向左走"
const WALK_RIGHT_ANIMATION: StringName = &"向右走"
const SCARED_ANIMATION: StringName = &"受惊"
const DEATH_ANIMATION: StringName = &"死亡"
const BEAR_ATTACK_FIRST_FRAME_ANIMATION: StringName = &"攻击第一帧"
const BEAR_ATTACK_SECOND_FRAME_ANIMATION: StringName = &"攻击第二帧"
## 玩家剧情行走动画名称。
const PLAYER_WALK_LEFT_ANIMATION: String = "walk_left"
const PLAYER_WALK_RIGHT_ANIMATION: String = "walk_right"
const PLAYER_WALK_UP_ANIMATION: String = "walk_up"

## 当前过场使用的东路地图。
@onready var _east_road: Node2D = $EastRoadOfShanguangTown
## 结尾切换后显示的罗克萨斯家地图。
@onready var _warehouse: Node2D = $RoxusHouse
## 桃子节点及动画节点。
@onready var _taozi: Node2D = $桃子
@onready var _taozi_sprite: AnimatedSprite2D = $桃子/AnimatedSprite2D
## 七色羽节点及动画节点。
@onready var _qiseyu: Node2D = $七色羽
@onready var _qiseyu_sprite: AnimatedSprite2D = $七色羽/AnimatedSprite2D
## 机械熊节点及动画节点。
@onready var _bear: Node2D = $机械熊
@onready var _bear_sprite: AnimatedSprite2D = $机械熊/AnimatedSprite2D
## 当前固定过场使用的本地玩家和相机。
@onready var _scene_player: CharacterBody2D = $Player
@onready var _scene_camera: Camera2D = $Player/Camera2D
## 单独运行剧情时使用的备用对话面板。
@onready var _standalone_dialogue_panel: NPCDialoguePanel = $StandaloneDialoguePanel
## 屏幕顶部闪烁的剧情图片。
@onready var _plot_image: CanvasLayer = $PlotImage
## 全屏渐黑遮罩和黑白闪烁遮罩。
@onready var _black_fade: ColorRect = $ScreenFade/BlackFade
@onready var _flash_overlay: ColorRect = $ImpactOverlay/FlashOverlay
## 最终能量爆发时显示的白色发光节点。
@onready var _glow_effect: Control = $发光特效
## 罗克萨斯对白头像来源节点。
@onready var _roxus_portrait_source: Sprite2D = $RoxusPortraitSource

## 相机同步世界参数后的基础偏移。
var _camera_base_offset: Vector2 = Vector2.ZERO


## 场景中断时清理遮罩、发光和相机偏移。
func _exit_tree() -> void:
	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	if _black_fade != null:
		_black_fade.hide()
		_black_fade.modulate.a = 0.0
	if _flash_overlay != null:
		_flash_overlay.hide()
	if _scene_camera != null:
		_scene_camera.offset = _camera_base_offset
	super._exit_tree()


## 按脚本注释顺序播放机械熊完整固定剧情。
func _run_sequence() -> void:
	_prepare_scene()
	await _show_actor_dialogue(
		"桃子",
		"嘿嘿🙂你好慢啊，我等得都快睡着了～",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _fade_from_black()
	await _move_scene_player(Vector2(30.0, 0.0), PLAYER_WALK_RIGHT_ANIMATION, Vector2.RIGHT)
	await _move_scene_player(Vector2(0.0, -39.0), PLAYER_WALK_UP_ANIMATION, Vector2.UP)
	await _move_scene_player(Vector2(10.0, 0.0), PLAYER_WALK_RIGHT_ANIMATION, Vector2.RIGHT)
	await _show_actor_dialogue(
		"玩家",
		"😓嫌我慢，你就先回屋里面等我呗，还在这站着等。",
		_get_scene_player_portrait(),
		true
	)
	await _show_actor_dialogue(
		"玩家",
		"🙂不说这个了，咱们快去收拾一下行李吧，这次比武大会可能需要在[color=blue]闪光平原[/color]住好几天呢，还得过五关斩六将很多次……",
		_get_scene_player_portrait(),
		true
	)
	await _show_actor_dialogue(
		"玩家",
		"不过如果你在初赛就被淘汰了的话，反而可以早点回家来了🤣",
		_get_scene_player_portrait(),
		true
	)
	_taozi_sprite.play(IDLE_RIGHT_ANIMATION)
	await _show_actor_dialogue(
		"桃子",
		"我在初赛就被淘汰😡？可惜啊，那是完全不可能的🤣",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _show_actor_dialogue(
		"桃子",
		"走吧，去收拾东西～😎",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _move_group_right()
	await _play_directional_shake()
	_taozi_sprite.play(IDLE_LEFT_ANIMATION)
	await _show_actor_dialogue(
		"桃子",
		"怎么回事？发生什么了？",
		_get_animated_sprite_portrait(_taozi_sprite)
	)

	await _fade_to_black()
	_bear.show()
	_black_fade.hide()
	_black_fade.modulate.a = 0.0
	await _play_short_shake()
	_taozi_sprite.play(IDLE_RIGHT_ANIMATION)
	await _show_actor_dialogue(
		"桃子",
		"！！这……这是什么？？",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _play_taozi_attack_sequence()
	await _show_actor_dialogue(
		"玩家",
		"[color=yellow]桃子[/color]！！",
		_get_scene_player_portrait(),
		true
	)
	await _move_actor(_bear, _bear_sprite, Vector2(-10.0, 0.0), WALK_LEFT_ANIMATION, false)
	_bear_sprite.play(IDLE_LEFT_ANIMATION)
	await _show_actor_dialogue(
		"机械熊",
		"吼喔喔喔喔喔喔喔！",
		_get_animated_sprite_portrait(_bear_sprite)
	)
	await _show_actor_dialogue(
		"玩家",
		"[color=yellow]桃子[/color]……已经……死了？",
		_get_scene_player_portrait(),
		true
	)
	await _play_short_shake()
	await _show_actor_dialogue("七色羽", "啾。", _get_animated_sprite_portrait(_qiseyu_sprite))
	await _show_actor_dialogue(
		"七色羽",
		"啾！啾啾，啾啾啾！啾啾！啾啾啾啾啾！",
		_get_animated_sprite_portrait(_qiseyu_sprite)
	)
	await _move_actor(_qiseyu, _qiseyu_sprite, Vector2(30.0, 0.0), WALK_LEFT_ANIMATION, true)
	await _move_actor(_qiseyu, _qiseyu_sprite, Vector2(0.0, -10.0), WALK_LEFT_ANIMATION, true)
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)
	await _play_final_energy_burst()

	await _show_actor_dialogue(
		"玩家",
		"……[color=yellow]桃子[/color]……不……这不是真的……",
		_get_scene_player_portrait(),
		true
	)
	await _show_actor_dialogue(
		"玩家",
		"为什么，刚刚我们还在说去[color=blue]闪光平原[/color]参加比武大会的事情，为什么才短短几分钟，就会变成这样？",
		_get_scene_player_portrait(),
		true
	)
	await _show_actor_dialogue(
		"桃子",
		"🙂[color=orange]你听，风是不是在笑啊？[/color]💗",
		_get_animated_sprite_portrait(_taozi_sprite)
	)
	await _show_actor_dialogue(
		"玩家",
		"[color=yellow]桃子[/color]！",
		_get_scene_player_portrait(),
		true
	)
	_switch_to_warehouse()
	await _fade_from_black()
	await _show_actor_dialogue(
		"罗克萨斯",
		"唔，你终于醒来啦？",
		_get_roxus_portrait()
	)
	await _show_actor_dialogue(
		"玩家",
		"[color=yellow]桃子[/color]……",
		_get_scene_player_portrait(),
		true
	)
	if _plot_image != null:
		_plot_image.call("hide_plot_image")
	complete_cinematic()


## 初始化地图、角色状态和所有默认隐藏的演出节点。
func _prepare_scene() -> void:
	if _plot_image != null:
		_plot_image.call("show_plot_image")
	normalize_cinematic_level_origin(_east_road)
	sync_cinematic_camera_with_world(_scene_camera, _east_road)
	_camera_base_offset = _scene_camera.offset
	_east_road.show()
	_warehouse.hide()
	_taozi.position = TAOZI_START_POSITION
	_taozi.show()
	_taozi_sprite.flip_h = false
	_taozi_sprite.play(IDLE_LEFT_ANIMATION)
	_qiseyu.position = QISEYU_START_POSITION
	_qiseyu.show()
	_qiseyu_sprite.flip_h = false
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)
	_bear.position = BEAR_START_POSITION
	_bear.hide()
	_bear_sprite.flip_h = false
	_bear_sprite.play(IDLE_LEFT_ANIMATION)
	_scene_player.position = PLAYER_START_POSITION
	_scene_player.show()
	_scene_player.call("set_scene_transition_locked", true)
	_scene_player.call("set_facing_direction", Vector2.RIGHT)
	_apply_cinematic_animation_speed()
	_set_npc_blur_radius(0.0)
	_glow_effect.hide()
	_flash_overlay.hide()
	_black_fade.modulate.a = 1.0
	_black_fade.show()


## 桃子先向右移动 30px领路，再让三名角色按统一速度各自移动和停止。
func _move_group_right() -> void:
	await _move_actor(
		_taozi,
		_taozi_sprite,
		Vector2(30.0, 0.0),
		WALK_RIGHT_ANIMATION,
		false
	)
	_taozi_sprite.play(WALK_RIGHT_ANIMATION)
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(WALK_LEFT_ANIMATION)
	_scene_player.call("play_cinematic_animation", PLAYER_WALK_RIGHT_ANIMATION, -1, 12.0)
	var taozi_tween: Tween = _create_linear_move_tween(_taozi, Vector2(60.0, 0.0))
	var player_tween: Tween = _create_linear_move_tween(_scene_player, Vector2(60.0, 0.0))
	_create_linear_move_tween(_qiseyu, Vector2(60.0, 0.0))
	await taozi_tween.finished
	_taozi_sprite.play(IDLE_RIGHT_ANIMATION)
	_qiseyu_sprite.flip_h = true
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)
	await player_tween.finished
	_scene_player.call("set_facing_direction", Vector2.RIGHT)


## 按当前统一移动速度创建单个线性 Tween，让不同距离的角色自然先后到达。
func _create_linear_move_tween(actor: Node2D, relative_offset: Vector2) -> Tween:
	var duration: float = maxf(relative_offset.length() / CINEMATIC_MOVE_SPEED, 0.01)
	var move_tween: Tween = create_tween()
	move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	move_tween.set_trans(Tween.TRANS_LINEAR)
	move_tween.tween_property(actor, "position", actor.position + relative_offset, duration)
	return move_tween


## 播放双方后退、机械熊分帧攻击、桃子击飞和其他角色退让动作。
func _play_taozi_attack_sequence() -> void:
	_taozi_sprite.play(WALK_LEFT_ANIMATION)
	_bear_sprite.flip_h = true
	_bear_sprite.play(WALK_LEFT_ANIMATION)
	var retreat_tween: Tween = create_tween().set_parallel(true)
	retreat_tween.tween_property(_bear, "position", _bear.position + Vector2(15.0, 0.0), 0.4)
	retreat_tween.tween_property(_taozi, "position", _taozi.position + Vector2(-15.0, 0.0), 0.4)
	await retreat_tween.finished
	_taozi_sprite.play(IDLE_RIGHT_ANIMATION)
	_bear_sprite.flip_h = false
	await _play_black_flash()
	_bear_sprite.play(BEAR_ATTACK_FIRST_FRAME_ANIMATION)
	await get_tree().create_timer(0.5).timeout
	await _play_black_flash()
	_bear_sprite.play(BEAR_ATTACK_SECOND_FRAME_ANIMATION)
	_taozi_sprite.play(SCARED_ANIMATION)
	var hit_tween: Tween = create_tween().set_parallel(true)
	hit_tween.tween_property(_taozi, "position", _taozi.position + Vector2(-100.0, 0.0), 1.2)
	hit_tween.tween_property(_scene_player, "position", _scene_player.position + Vector2(-20.0, 0.0), 0.4)
	_qiseyu_sprite.flip_h = false
	_qiseyu_sprite.play(WALK_LEFT_ANIMATION)
	hit_tween.tween_property(_qiseyu, "position", _qiseyu.position + Vector2(-100.0, 0.0), 1.0)
	await hit_tween.finished
	_taozi_sprite.play(DEATH_ANIMATION)
	_scene_player.call("set_facing_direction", Vector2.LEFT)
	_qiseyu_sprite.play(IDLE_LEFT_ANIMATION)


## 开启两个 NPC 的局部模糊与白光，并在多次黑白闪烁和震动后保持全黑。
func _play_final_energy_burst() -> void:
	_set_npc_blur_radius(FINAL_BLUR_RADIUS)
	_glow_effect.show()
	for flash_index: int in range(FINAL_FLASH_COUNT):
		await _play_black_flash()
		await _play_short_shake()
	await _fade_to_black()
	_glow_effect.hide()


## 切换到罗克萨斯家地图，并隐藏东路剧情角色和特效。
func _switch_to_warehouse() -> void:
	_east_road.hide()
	_taozi.hide()
	_qiseyu.hide()
	_bear.hide()
	_glow_effect.hide()
	_warehouse.show()
	normalize_cinematic_level_origin(_warehouse)
	_scene_player.position = WAREHOUSE_PLAYER_POSITION
	_scene_player.call("set_facing_direction", Vector2.UP)
	sync_cinematic_camera_with_world(_scene_camera, _warehouse)
	_camera_base_offset = _scene_camera.offset


## 设置机械熊和七色羽实例共用模糊材质的半径。
func _set_npc_blur_radius(radius: float) -> void:
	for sprite: AnimatedSprite2D in [_bear_sprite, _qiseyu_sprite]:
		var blur_material: ShaderMaterial = sprite.material as ShaderMaterial
		if blur_material != null:
			blur_material.set_shader_parameter("blur_radius", radius)


## 画面平滑变为黑色。
func _fade_to_black() -> void:
	_black_fade.modulate.a = 0.0
	_black_fade.show()
	var fade_tween: Tween = create_tween()
	fade_tween.tween_property(_black_fade, "modulate:a", 1.0, FADE_SECONDS)
	await fade_tween.finished


## 从当前黑屏平滑恢复画面。
func _fade_from_black() -> void:
	_black_fade.show()
	var fade_tween: Tween = create_tween()
	fade_tween.tween_property(_black_fade, "modulate:a", 0.0, FADE_SECONDS)
	await fade_tween.finished
	_black_fade.hide()


## 只显示短促黑屏并恢复画面，不再播放可能刺激眼睛的白色闪屏。
func _play_black_flash() -> void:
	_flash_overlay.color = Color.BLACK
	_flash_overlay.show()
	await get_tree().create_timer(FLASH_PHASE_SECONDS).timeout
	_flash_overlay.hide()
	await get_tree().create_timer(FLASH_PHASE_SECONDS).timeout


## 将机械熊、桃子、七色羽和玩家的剧情动画播放倍率统一设为 0.5。
func _apply_cinematic_animation_speed() -> void:
	for actor_sprite: AnimatedSprite2D in [_taozi_sprite, _qiseyu_sprite, _bear_sprite]:
		actor_sprite.speed_scale = CINEMATIC_ANIMATION_SPEED_SCALE
	var player_animation: AnimationPlayer = _scene_player.get_node_or_null("AnimationPlayer") as AnimationPlayer
	if player_animation != null:
		player_animation.speed_scale = CINEMATIC_ANIMATION_SPEED_SCALE
	var player_visual_sprite: AnimatedSprite2D = _scene_player.get_node_or_null(
        "CharacterVisual/AnimatedSprite2D"
	) as AnimatedSprite2D
	if player_visual_sprite != null:
		player_visual_sprite.speed_scale = CINEMATIC_ANIMATION_SPEED_SCALE


## 按上下后左右顺序震动约一秒，并恢复相机基础偏移。
func _play_directional_shake() -> void:
	var offsets: Array[Vector2] = [
		Vector2(0.0, -5.0), Vector2(0.0, 5.0),
		Vector2(0.0, -5.0), Vector2(0.0, 5.0),
		Vector2(-6.0, 0.0), Vector2(6.0, 0.0),
		Vector2(-6.0, 0.0), Vector2(6.0, 0.0),
		Vector2.ZERO,
	]
	for offset: Vector2 in offsets:
		await _tween_camera_offset(offset, 0.11)


## 播放一次短促四段震动。
func _play_short_shake() -> void:
	for offset: Vector2 in [Vector2(-6.0, 3.0), Vector2(7.0, -3.0), Vector2(-4.0, 2.0), Vector2.ZERO]:
		await _tween_camera_offset(offset, SHAKE_STEP_SECONDS)


## 将相机移动到基础偏移叠加指定震动量。
func _tween_camera_offset(offset: Vector2, duration: float) -> void:
	var shake_tween: Tween = create_tween()
	shake_tween.tween_property(_scene_camera, "offset", _camera_base_offset + offset, duration)
	await shake_tween.finished


## 展示一条固定对白；正式客户端走 CinematicPlayer，单独运行时使用备用面板。
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
		await super.show_local_dialogue(speaker_name, content, portrait_key, is_player_speaking, content_format, portrait_texture)
		return
	if _standalone_dialogue_panel == null:
		return
	_standalone_dialogue_panel.show_local_dialogue(speaker_name, content, portrait_key, is_player_speaking, content_format, portrait_texture)
	await _standalone_dialogue_panel.local_continue_requested
	_standalone_dialogue_panel.hide_panel(false)


## 使用指定场景角色头像展示固定对白。
func _show_actor_dialogue(
	speaker_name: String,
	content: String,
	portrait_texture: Texture2D,
	is_player_speaking: bool = false
) -> void:
	await show_local_dialogue(speaker_name, content, "", is_player_speaking, "bbcode", portrait_texture)


## 从角色序列帧提取正面或当前帧上半身头像。
func _get_animated_sprite_portrait(sprite: AnimatedSprite2D) -> Texture2D:
	if sprite == null or sprite.sprite_frames == null:
		return null
	var animation_name: StringName = &"待机下"
	if not sprite.sprite_frames.has_animation(animation_name):
		animation_name = sprite.animation
	if not sprite.sprite_frames.has_animation(animation_name):
		return null
	var frame_count: int = sprite.sprite_frames.get_frame_count(animation_name)
	if frame_count <= 0:
		return null
	return PortraitRegistry.crop_upper_body_portrait(sprite.sprite_frames.get_frame_texture(animation_name, 0))


## 获取当前玩家上半身头像。
func _get_scene_player_portrait() -> Texture2D:
	return PortraitRegistry.load_player_dialogue_portrait()


## 获取罗克萨斯图片的上半身头像。
func _get_roxus_portrait() -> Texture2D:
	if _roxus_portrait_source == null:
		return null
	return PortraitRegistry.crop_upper_body_portrait(_roxus_portrait_source.texture)


## 让单个本地角色按固定偏移移动并播放行走动画。
func _move_actor(
	actor: Node2D,
	actor_sprite: AnimatedSprite2D,
	relative_offset: Vector2,
	walk_animation: StringName,
	flip_h: bool
) -> void:
	actor_sprite.flip_h = flip_h
	actor_sprite.play(walk_animation)
	var duration: float = maxf(relative_offset.length() / CINEMATIC_MOVE_SPEED, 0.01)
	var move_tween: Tween = create_tween()
	move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	move_tween.set_trans(Tween.TRANS_LINEAR)
	move_tween.tween_property(actor, "position", actor.position + relative_offset, duration)
	await move_tween.finished


## 让剧情玩家按固定偏移移动，并在结束后切换指定朝向。
func _move_scene_player(
	relative_offset: Vector2,
	walk_animation: String,
	final_facing_direction: Vector2
) -> void:
	_scene_player.call("set_facing_direction", relative_offset)
	_scene_player.call("play_cinematic_animation", walk_animation, -1, 12.0)
	var duration: float = maxf(relative_offset.length() / CINEMATIC_MOVE_SPEED, 0.01)
	var move_tween: Tween = create_tween()
	move_tween.set_process_mode(Tween.TWEEN_PROCESS_PHYSICS)
	move_tween.set_trans(Tween.TRANS_LINEAR)
	move_tween.tween_property(_scene_player, "position", _scene_player.position + relative_offset, duration)
	await move_tween.finished
	_scene_player.call("set_facing_direction", final_facing_direction)
