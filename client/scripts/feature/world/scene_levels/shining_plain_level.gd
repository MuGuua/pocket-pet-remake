extends NetworkDoorLevelBase

## 闪光平原通用地图脚本，复用服务端权威传送门切图和移动端地图展示能力。

## scene_display_name 是右上角 HUD 展示的当前场景名称，由各地图场景资源配置。
@export var scene_display_name: String = "闪光平原"
## level_scale_factor 是当前地图在移动端视口中的展示缩放倍数。
@export var level_scale_factor: float = 1.0
## login_spawn_position 是没有服务端权威位置时的默认场景坐标。
@export var login_spawn_position: Vector2 = Vector2.ZERO
## door_configs 只保存门节点对应的请求参数；目标合法性和出生格始终由服务端判定。
@export var door_configs: Dictionary = {}


## 返回当前场景用于 HUD 展示的名称。
func get_scene_display_name() -> String:
    return scene_display_name


## 返回当前场景门节点与服务端传送门编号的映射。
func _get_door_configs() -> Dictionary:
    return door_configs


## 出生位置统一使用服务端世界快照；通用脚本不覆盖任何传送门落点。
func get_portal_spawn_scene_position(_portal_id: int) -> Vector2:
    return INVALID_PORTAL_SPAWN_SCENE_POSITION


## 返回当前地图已使用区域的中心点，供移动端视口居中展示。
func get_level_center_position() -> Vector2:
    var map_layer: TileMapLayer = _resolve_map_layer()
    if map_layer == null:
        return Vector2.ZERO

    var used_rect: Rect2i = map_layer.get_used_rect()
    if not used_rect.has_area():
        return Vector2.ZERO

    var top_left: Vector2 = map_layer.map_to_local(used_rect.position)
    var bottom_right: Vector2 = map_layer.map_to_local(used_rect.position + used_rect.size)
    return (top_left + bottom_right) * 0.5


## 返回当前地图在移动端视口中的展示缩放倍数。
func get_level_scale_factor() -> float:
    return level_scale_factor


## 返回没有服务端权威位置时的默认场景坐标。
func get_login_spawn_position() -> Vector2:
    return login_spawn_position


## 查找闪光平原地图用于中心点和边界计算的主 TileMapLayer。
func _resolve_map_layer() -> TileMapLayer:
    for layer_name: String in ["地图", "Map", "TileMapLayer"]:
        var layer: TileMapLayer = get_node_or_null(layer_name) as TileMapLayer
        if layer != null:
            return layer
    return null
