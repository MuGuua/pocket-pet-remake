extends Label
class_name FloatingText

const DISPLAY_DURATION: float = 0.6
## 战斗文字飘字统一放大，确保技能名与状态提示在放大后的战场里清晰可读。
const FLOAT_TEXT_SCALE: Vector2 = Vector2(1.35, 1.35)

func play(text_value: String, text_color: Color) -> void:
    text = text_value
    modulate = text_color
    scale = FLOAT_TEXT_SCALE
    var tween: Tween = create_tween()
    tween.set_parallel(true)
    tween.tween_property(self, "position:y", position.y - 32.0, DISPLAY_DURATION).set_trans(Tween.TRANS_SINE).set_ease(Tween.EASE_OUT)
    tween.tween_property(self, "modulate:a", 0.0, DISPLAY_DURATION)
    tween.tween_property(self, "scale", FLOAT_TEXT_SCALE * 1.08, minf(DISPLAY_DURATION * 0.3, DISPLAY_DURATION))
    await tween.finished
    queue_free()
