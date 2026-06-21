class_name CinematicPlayer
extends Node

## 剧情动画播放完成后向外广播，主流程收到后再向服务端请求下一个剧情节点。
signal cinematic_finished(animation_key: String)

## 播放指定的客户端内置剧情动画；当前阶段如果资源不存在，则直接视为播放完成。
func play_cinematic(animation_key: String) -> void:
	var scene_path: String = CinematicRegistry.get_path_by_key(animation_key)
	if scene_path.is_empty():
		call_deferred("_emit_finished", animation_key)
		return
	var resource_variant: Variant = load(scene_path)
	var packed_scene: PackedScene = resource_variant as PackedScene
	if packed_scene == null:
		call_deferred("_emit_finished", animation_key)
		return
	var cinematic_node: Node = packed_scene.instantiate()
	add_child(cinematic_node)
	if cinematic_node.has_signal("finished"):
		cinematic_node.connect("finished", Callable(self, "_on_cinematic_finished").bind(animation_key, cinematic_node))
		return
	call_deferred("_finish_without_signal", animation_key, cinematic_node)

## 处理带有 finished 信号的剧情动画资源，释放实例后再恢复剧情推进。
func _on_cinematic_finished(animation_key: String, cinematic_node: Node) -> void:
	if is_instance_valid(cinematic_node):
		cinematic_node.queue_free()
	_emit_finished(animation_key)

## 处理没有 finished 信号的简单剧情资源，避免节点残留在树上。
func _finish_without_signal(animation_key: String, cinematic_node: Node) -> void:
	if is_instance_valid(cinematic_node):
		cinematic_node.queue_free()
	_emit_finished(animation_key)

## 统一发出剧情动画播放完成事件。
func _emit_finished(animation_key: String) -> void:
	cinematic_finished.emit(animation_key)
