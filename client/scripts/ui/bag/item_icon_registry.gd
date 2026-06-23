extends Resource
class_name ItemIconRegistry

## 默认图标，服务端传来的 icon_key 不存在时使用。
@export var default_icon: Texture2D = null

## 图标映射列表，由编辑器或资源文件维护。
@export var entries: Array[ItemIconEntry] = []

## 运行时缓存，避免每次查找都遍历列表。
var _entry_map: Dictionary = {}


## 根据服务端 icon_key 查找本地图标。
func resolve_icon(icon_key: String) -> Texture2D:
    if _entry_map.is_empty():
        _build_entry_map()

    if _entry_map.has(icon_key):
        return _entry_map[icon_key] as Texture2D

    return default_icon


## 构建 icon_key 到 Texture2D 的查询表。
func _build_entry_map() -> void:
    _entry_map.clear()
    for entry: ItemIconEntry in entries:
        if entry == null:
            continue
        if entry.icon_key.is_empty():
            continue
        if entry.texture == null:
            continue
        _entry_map[entry.icon_key] = entry.texture
