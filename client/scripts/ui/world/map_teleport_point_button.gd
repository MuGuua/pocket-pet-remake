extends Button
class_name MapTeleportPointButton

## target_scene_id 是该地图标点提交给服务端的目标场景 ID；出生坐标始终由服务端数据库决定。
@export var target_scene_id: int = 0
## teleport_enabled 表示客户端是否允许发起请求；尚无目标场景资源的标点保持可选但不能传送。
@export var teleport_enabled: bool = true
