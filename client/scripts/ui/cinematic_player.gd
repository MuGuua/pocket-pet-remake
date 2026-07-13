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
## 当前正在播放的剧情场景实例；同一时间只允许一个演出控制真实玩家。
var _active_cinematic_node: Node = null
## 播放代次；用于忽略已取消演出的延迟回调和完成信号。
var _playback_generation: int = 0

## 绑定当前世界控制器，新建剧情实例时会在入树前注入。
func bind_world_controller(world_controller: Node) -> void:
    _world_controller = world_controller

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
    var cinematic_node: Node = packed_scene.instantiate()
    _active_cinematic_node = cinematic_node
    if cinematic_node.has_method("configure_world_context"):
        cinematic_node.call("configure_world_context", _world_controller)
    if cinematic_node.has_signal("local_dialogue_requested"):
        cinematic_node.connect(
            "local_dialogue_requested",
            Callable(self, "_on_local_dialogue_requested").bind(cinematic_node, playback_generation)
        )
    add_child(cinematic_node)
    if cinematic_node.has_signal("finished"):
        cinematic_node.connect("finished", Callable(self, "_on_cinematic_finished").bind(animation_key, cinematic_node, playback_generation))
        return
    call_deferred("_finish_without_signal", animation_key, cinematic_node, playback_generation)

## 取消当前演出；动作场景退出树时会自行恢复玩家控制，且不会推进剧情节点。
func cancel_active_cinematic() -> void:
    _playback_generation += 1
    _release_active_cinematic()

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

## 处理带有 finished 信号的剧情动画资源，释放实例后再恢复剧情推进。
func _on_cinematic_finished(animation_key: String, cinematic_node: Node, playback_generation: int) -> void:
    if playback_generation != _playback_generation or cinematic_node != _active_cinematic_node:
        return
    _active_cinematic_node = null
    if is_instance_valid(cinematic_node):
        cinematic_node.queue_free()
    _emit_finished(animation_key, playback_generation)

## 处理没有 finished 信号的简单剧情资源，避免节点残留在树上。
func _finish_without_signal(animation_key: String, cinematic_node: Node, playback_generation: int) -> void:
    if playback_generation != _playback_generation or cinematic_node != _active_cinematic_node:
        return
    _active_cinematic_node = null
    if is_instance_valid(cinematic_node):
        cinematic_node.queue_free()
    _emit_finished(animation_key, playback_generation)

## 释放当前演出实例；场景的退出树清理负责恢复其占用的世界状态。
func _release_active_cinematic() -> void:
    if _active_cinematic_node != null and is_instance_valid(_active_cinematic_node):
        _active_cinematic_node.queue_free()
    _active_cinematic_node = null

## 仅为当前播放代次发出完成事件，避免旧的延迟回调推进新剧情。
func _emit_finished(animation_key: String, playback_generation: int) -> void:
    if playback_generation != _playback_generation:
        return
    cinematic_finished.emit(animation_key)
