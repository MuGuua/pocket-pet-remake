extends Resource
class_name ItemIconEntry

## 服务端下发的图标键，只用于客户端查找本地贴图。
@export var icon_key: String = ""

## 本地图标贴图，随客户端包体发布，不从网络下载。
@export var texture: Texture2D = null
