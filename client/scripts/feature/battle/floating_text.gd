extends Label
class_name FloatingText

const DISPLAY_DURATION: float = 0.6

func play(text_value: String, text_color: Color) -> void:
	text = text_value
	modulate = text_color
	scale = Vector2.ONE
	var tween: Tween = create_tween()
	tween.set_parallel(true)
	tween.tween_property(self, "position:y", position.y - 18.0, DISPLAY_DURATION).set_trans(Tween.TRANS_SINE).set_ease(Tween.EASE_OUT)
	tween.tween_property(self, "modulate:a", 0.0, DISPLAY_DURATION)
	tween.tween_property(self, "scale", Vector2(1.08, 1.08), minf(DISPLAY_DURATION * 0.3, DISPLAY_DURATION))
	await tween.finished
	queue_free()
