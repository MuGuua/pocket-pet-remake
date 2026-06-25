extends Node

## 本地物品图标解析入口，背包与装备 UI 统一通过 item_id 查表。
const REGISTRY_PATH: String = "res://resources/ui/item_icon_registry.tres"
const FALLBACK_ATLAS_PATH: String = "res://asset/分类/武器/pixel items0.png"
const FALLBACK_ICON_SIZE: int = 32

var _registry: ItemIconRegistry = null
var _fallback_texture: Texture2D = null


func _ready() -> void:
	_load_registry()


## 按 item_id 解析本地图标贴图，未配置时回退默认图标。
func resolve_texture(item_id: int) -> Texture2D:
	if _registry == null:
		_load_registry()
	if _registry != null and item_id > 0:
		var resolved: Texture2D = _registry.resolve_icon_for_item(item_id)
		if resolved != null:
			return resolved
	return _build_fallback_texture()


func _load_registry() -> void:
	var resource: Resource = load(REGISTRY_PATH)
	if resource is ItemIconRegistry:
		_registry = resource as ItemIconRegistry


func _build_fallback_texture() -> Texture2D:
	if _fallback_texture != null:
		return _fallback_texture
	if _registry != null and _registry.default_icon != null:
		_fallback_texture = _registry.default_icon
		return _fallback_texture
	var atlas_source: Resource = load(FALLBACK_ATLAS_PATH)
	if atlas_source is not Texture2D:
		return null
	var atlas_texture: AtlasTexture = AtlasTexture.new()
	atlas_texture.atlas = atlas_source as Texture2D
	atlas_texture.region = Rect2(0.0, 0.0, float(FALLBACK_ICON_SIZE), float(FALLBACK_ICON_SIZE))
	_fallback_texture = atlas_texture
	return _fallback_texture
