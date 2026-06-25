extends Resource
class_name ItemIconRegistry

## 默认图标，item_id 与 icon_key 均未命中时使用。
@export var default_icon: Texture2D = null

## 图标映射列表，由编辑器或资源文件维护。
@export var entries: Array[ItemIconEntry] = []

## 运行时缓存，避免每次查找都遍历列表。
var _icon_key_map: Dictionary = {}
var _item_id_map: Dictionary = {}


## 按 item_id 查找本地图标，未命中时回退默认图标。
func resolve_icon_for_item(item_id: int) -> Texture2D:
	if _item_id_map.is_empty() and _icon_key_map.is_empty():
		_build_entry_maps()
	if item_id > 0 and _item_id_map.has(item_id):
		return _item_id_map[item_id] as Texture2D
	return default_icon


## 按 icon_key 查找本地图标，未命中时回退默认图标。
func resolve_icon(icon_key: String) -> Texture2D:
	if _item_id_map.is_empty() and _icon_key_map.is_empty():
		_build_entry_maps()
	if _icon_key_map.has(icon_key):
		return _icon_key_map[icon_key] as Texture2D
	return default_icon


## 构建 item_id 与 icon_key 到 Texture2D 的查询表。
func _build_entry_maps() -> void:
	_icon_key_map.clear()
	_item_id_map.clear()
	for entry: ItemIconEntry in entries:
		if entry == null or entry.texture == null:
			continue
		if entry.item_id > 0:
			_item_id_map[entry.item_id] = entry.texture
		if not entry.icon_key.is_empty():
			_icon_key_map[entry.icon_key] = entry.texture
