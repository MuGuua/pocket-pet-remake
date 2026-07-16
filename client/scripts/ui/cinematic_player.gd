class_name CinematicPlayer
extends Node

## 剧情动画播放完成后向外广播，主流程收到后再向服务端请求下一个剧情节点。
signal cinematic_finished(animation_key: String)
## 当前客户端过场请求展示一句本地对白。
signal local_dialogue_requested(
    speaker_name: String,
    content: String,
    portrait_key: String,
    is_player_speaking: bool,
    content_format: String,
    portrait_texture: Texture2D
)

## 当前世界控制器；剧情场景只能通过其公开接口控制真实玩家。
var _world_controller: Node = null
## 主场景提供的全屏黑色遮罩；所有剧情统一复用该场景节点执行进出过渡。
var _transition_overlay: ColorRect = null
## 单次渐黑或渐亮的持续秒数，由主场景统一注入。
var _transition_duration: float = 0.18
## 当前剧情过渡补间；取消剧情时需要同步停止，避免遮罩残留。
var _transition_tween: Tween = null
## 当前正在播放的剧情场景实例；同一时间只允许一个演出控制真实玩家。
var _active_cinematic_node: Node = null
## 播放代次；用于忽略已取消演出的延迟回调和完成信号。
var _playback_generation: int = 0

## 绑定当前世界控制器，新建剧情实例时会在入树前注入。
func bind_world_controller(world_controller: Node) -> void:
    _world_controller = world_controller

## 绑定主场景已有的全屏过渡遮罩，避免每个独立剧情场景重复创建转场 UI。
func bind_transition_overlay(transition_overlay: ColorRect, duration_seconds: float) -> void:
    _transition_overlay = transition_overlay
    _transition_duration = maxf(duration_seconds, 0.01)

## 播放指定的客户端内置剧情动画；当前阶段如果资源不存在，则直接视为播放完成。
func play_cinematic(animation_key: String) -> void:
    _playback_generation += 1
    var playback_generation: int = _playback_generation
    _release_active_cinematic()
    var scene_path: String = CinematicRegistry.get_path_by_key(animation_key)
    if scene_path.is_empty():
        call_deferred("_emit_finished", animation_key, playback_generation)
        return
    var resource_variant: Variant = load(scene_path)
    var packed_scene: PackedScene = resource_variant as PackedScene
    if packed_scene == null:
        call_deferred("_emit_finished", animation_key, playback_generation)
        return
    call_deferred("_enter_cinematic", animation_key, packed_scene, playback_generation)

## 先把世界画面渐黑，在全黑状态挂载独立剧情场景，再渐亮开始演出。
func _enter_cinematic(animation_key: String, packed_scene: PackedScene, playback_generation: int) -> void:
    await _fade_transition_overlay(1.0)
    if playback_generation != _playback_generation:
        return
    var cinematic_node: Node = packed_scene.instantiate()
    _active_cinematic_node = cinematic_node
    if cinematic_node.has_method("configure_world_context"):
        cinematic_node.call("configure_world_context", _world_controller)
    if cinematic_node.has_signal("local_dialogue_requested"):
        cinematic_node.connect(
            "local_dialogue_requested",
            Callable(self, "_on_local_dialogue_requested").bind(cinematic_node, playback_generation)
        )
    var cinematic_parent: Node = _resolve_cinematic_mount_parent()
    cinematic_parent.add_child(cinematic_node)
    if cinematic_node.has_signal("finished"):
        cinematic_node.connect("finished", Callable(self, "_on_cinematic_finished").bind(animation_key, cinematic_node, playback_generation))
    else:
        call_deferred("_finish_without_signal", animation_key, cinematic_node, playback_generation)
    await get_tree().process_frame
    if playback_generation != _playback_generation:
        return
    await _fade_transition_overlay(0.0)

## 取消当前演出；动作场景退出树时会自行恢复玩家控制，且不会推进剧情节点。
func cancel_active_cinematic() -> void:
    _playback_generation += 1
    _stop_transition_tween()
    _release_active_cinematic()
    _reset_transition_overlay()

## 把玩家的本地对白继续操作转交给当前过场场景。
func advance_local_dialogue() -> void:
    if _active_cinematic_node == null or not is_instance_valid(_active_cinematic_node):
        return
    if _active_cinematic_node.has_method("advance_local_dialogue"):
        _active_cinematic_node.call("advance_local_dialogue")

## 仅转发当前播放代次的本地对白请求，旧场景不能覆盖新过场内容。
func _on_local_dialogue_requested(
    speaker_name: String,
    content: String,
    portrait_key: String,
    is_player_speaking: bool,
    content_format: String,
    portrait_texture: Texture2D,
    cinematic_node: Node,
    playback_generation: int
) -> void:
    if playback_generation != _playback_generation or cinematic_node != _active_cinematic_node:
        return
    local_dialogue_requested.emit(
        speaker_name,
        content,
        portrait_key,
        is_player_speaking,
        content_format,
        portrait_texture
    )

## 处理带有 finished 信号的剧情动画资源，统一渐黑退出后再恢复世界画面和剧情推进。
func _on_cinematic_finished(animation_key: String, cinematic_node: Node, playback_generation: int) -> void:
    if playback_generation != _playback_generation or cinematic_node != _active_cinematic_node:
        return
    await _exit_cinematic(animation_key, cinematic_node, playback_generation)

## 保持剧情场景显示直到屏幕全黑，再释放场景并渐亮恢复原世界场景。
func _exit_cinematic(animation_key: String, cinematic_node: Node, playback_generation: int) -> void:
    await _fade_transition_overlay(1.0)
    if playback_generation != _playback_generation or cinematic_node != _active_cinematic_node:
        return
    _active_cinematic_node = null
    if is_instance_valid(cinematic_node):
        cinematic_node.queue_free()
    await get_tree().process_frame
    if playback_generation != _playback_generation:
        return
    await _fade_transition_overlay(0.0)
    _emit_finished(animation_key, playback_generation)

## 处理没有 finished 信号的简单剧情资源，避免节点残留在树上。
func _finish_without_signal(animation_key: String, cinematic_node: Node, playback_generation: int) -> void:
    if playback_generation != _playback_generation or cinematic_node != _active_cinematic_node:
        return
    await _exit_cinematic(animation_key, cinematic_node, playback_generation)

## 释放当前演出实例；场景的退出树清理负责恢复其占用的世界状态。
func _release_active_cinematic() -> void:
    if _active_cinematic_node != null and is_instance_valid(_active_cinematic_node):
        _active_cinematic_node.queue_free()
    _active_cinematic_node = null

## 渐变主场景提供的黑色遮罩；未绑定遮罩时直接继续，保证播放器仍可独立使用。
func _fade_transition_overlay(target_alpha: float) -> void:
    if _transition_overlay == null or not is_instance_valid(_transition_overlay):
        return
    _stop_transition_tween()
    _transition_overlay.show()
    _transition_overlay.mouse_filter = Control.MOUSE_FILTER_STOP
    var safe_target_alpha: float = clampf(target_alpha, 0.0, 1.0)
    if is_equal_approx(_transition_overlay.color.a, safe_target_alpha):
        if is_zero_approx(safe_target_alpha):
            _transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE
        return
    _transition_tween = create_tween()
    _transition_tween.tween_property(
        _transition_overlay,
        "color:a",
        safe_target_alpha,
        _transition_duration
    )
    await _transition_tween.finished
    _transition_tween = null
    if is_zero_approx(target_alpha):
        _transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

## 停止尚未结束的剧情转场补间，避免新剧情或取消操作继续修改遮罩透明度。
func _stop_transition_tween() -> void:
    if _transition_tween != null and _transition_tween.is_valid():
        _transition_tween.kill()
    _transition_tween = null

## 取消剧情时立即恢复世界可见和可交互状态。
func _reset_transition_overlay() -> void:
    if _transition_overlay == null or not is_instance_valid(_transition_overlay):
        return
    _transition_overlay.color.a = 0.0
    _transition_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE

## 解析剧情场景的挂载父节点；优先放进世界 SubViewport，让地图、背景、相机缩放与正常世界完全一致。
func _resolve_cinematic_mount_parent() -> Node:
    if _world_controller != null:
        var game_viewport_variant: Variant = _world_controller.get("game_viewport")
        if game_viewport_variant is Node:
            var game_viewport_node: Node = game_viewport_variant as Node
            if game_viewport_node.is_inside_tree():
                return game_viewport_node
    return self

## 仅为当前播放代次发出完成事件，避免旧的延迟回调推进新剧情。
func _emit_finished(animation_key: String, playback_generation: int) -> void:
    if playback_generation != _playback_generation:
        return
    cinematic_finished.emit(animation_key)
