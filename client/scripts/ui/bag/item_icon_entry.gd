extends Resource
class_name ItemIconEntry

## 物品模板 id，优先用于客户端本地查表。
@export var item_id: int = 0

## 可选图标键，兼容旧配置或跨系统引用。
@export var icon_key: String = ""

## 本地图标贴图，随客户端包体发布，不从网络下载。
@export var texture: Texture2D = null
