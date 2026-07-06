extends Node2D
class_name BattleDigitFloat

## 使用数字图集拼出 -22 / +15 等战斗飘字，替代 Label 文本。

const FLOAT_DISTANCE: float = 36.0
const DISPLAY_DURATION: float = 1.5
## 位图伤害数字统一放大，让伤害/治疗反馈比战斗单位更醒目。
const DIGIT_FLOAT_SCALE: Vector2 = Vector2(2.0, 2.0)

static var _default_atlas: BattleDigitAtlas = null

## 播放一段数字飘字并在结束后自动销毁。
func play(value_text: String, tint: Color = Color.WHITE, atlas: BattleDigitAtlas = null) -> void:
	var digit_atlas: BattleDigitAtlas = atlas if atlas != null else _get_default_atlas()
	if digit_atlas == null or digit_atlas.texture == null:
		queue_free()
		return
	modulate = Color(tint.r, tint.g, tint.b, 1.0)
	scale = DIGIT_FLOAT_SCALE
	_build_glyphs(value_text, digit_atlas)
	var tween: Tween = create_tween()
	tween.tween_property(self, "position:y", position.y - FLOAT_DISTANCE, DISPLAY_DURATION).set_trans(Tween.TRANS_LINEAR)
	await tween.finished
	queue_free()

func _build_glyphs(value_text: String, atlas: BattleDigitAtlas) -> void:
	var glyph_width: float = float(atlas.glyph_size.x)
	var glyph_height: float = float(atlas.glyph_size.y)
	var char_count: int = value_text.length()
	if char_count <= 0:
		return
	var total_width: float = glyph_width * float(char_count)
	var start_x: float = -total_width * 0.5 + glyph_width * 0.5
	for index: int in range(char_count):
		var char_text: String = value_text.substr(index, 1)
		var region: Rect2 = atlas.get_glyph_region(char_text)
		if region.size == Vector2.ZERO:
			continue
		var glyph_texture: AtlasTexture = AtlasTexture.new()
		glyph_texture.atlas = atlas.texture
		glyph_texture.region = region
		var glyph_sprite: Sprite2D = Sprite2D.new()
		glyph_sprite.name = "Digit_%d" % index
		glyph_sprite.texture = glyph_texture
		glyph_sprite.centered = true
		glyph_sprite.position = Vector2(start_x + float(index) * glyph_width, -glyph_height * 0.5)
		add_child(glyph_sprite)

static func _get_default_atlas() -> BattleDigitAtlas:
	if _default_atlas == null:
		_default_atlas = BattleDigitAtlas.load_default()
	return _default_atlas
