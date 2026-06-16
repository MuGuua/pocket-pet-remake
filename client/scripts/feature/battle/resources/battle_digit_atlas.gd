extends Resource
class_name BattleDigitAtlas

## 战斗飘字用的数字图集配置；默认读取 0123456789+- 横排贴图。

const DEFAULT_TEXTURE_PATH: String = "res://asset/分类/ui/战斗数字.png"
const DEFAULT_CHAR_ORDER: String = "0123456789+-"

@export var texture: Texture2D
## 单个字符在图集里的宽高，默认 13x16 对应战斗数字.png。
@export var glyph_size: Vector2i = Vector2i(13, 16)
@export var char_order: String = DEFAULT_CHAR_ORDER

## 返回指定字符在图集里的裁剪区域；找不到时返回空区域。
func get_glyph_region(char_text: String) -> Rect2:
	if char_text.is_empty() or texture == null:
		return Rect2()
	var index: int = char_order.find(char_text.substr(0, 1))
	if index < 0:
		return Rect2()
	return Rect2(
		float(index * glyph_size.x),
		0.0,
		float(glyph_size.x),
		float(glyph_size.y)
	)

## 加载默认战斗数字图集，供飘字组件复用。
static func load_default() -> BattleDigitAtlas:
	var atlas: BattleDigitAtlas = BattleDigitAtlas.new()
	atlas.texture = load(DEFAULT_TEXTURE_PATH) as Texture2D
	atlas.glyph_size = Vector2i(13, 16)
	atlas.char_order = DEFAULT_CHAR_ORDER
	return atlas
