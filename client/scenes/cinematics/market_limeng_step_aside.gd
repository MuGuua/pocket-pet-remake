extends Node2D

## 市场理萌“挪开货箱”剧情演出；播放完成后通过 finished 通知 CinematicPlayer 继续推进服务端剧情。
signal finished

@onready var _sprite: AnimatedSprite2D = $AnimatedSprite2D

## 启动位移动画，模拟 NPC 给玩家让路。
func _ready() -> void:
	var tween: Tween = create_tween()
	tween.tween_property(_sprite, "position:x", _sprite.position.x + 24.0, 0.75)
	tween.tween_callback(_emit_finished)

## 广播剧情动画结束事件。
func _emit_finished() -> void:
	finished.emit()
