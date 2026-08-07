extends NetworkDoorLevelBase

## scene_display_name 是右上角 HUD 展示的当前场景名称，可在 .tscn 中按地图覆盖。
@export var scene_display_name: String = "闪光市场"
## level_scale_factor 是当前地图在移动端视口中的展示缩放倍数。
@export var level_scale_factor: float = 1.0
## 从东路 portal_id=2002 进入市场后的玩家场景坐标。
@export var inbound_from_east_road_scene_position: Vector2 = Vector2(12.0, 10.0)
## 从北路 portal_id=4001 回到市场后的玩家场景坐标。
@export var inbound_from_bei_lu_scene_position: Vector2 = Vector2(5.0, 2.0)
## 从学校 portal_id=5001 回到市场后的玩家场景坐标。
@export var inbound_from_xue_xiao_scene_position: Vector2 = Vector2(4.0, 13.0)
## login_and_map_teleport_spawn_position 是登录进入当前地图和世界地图快速传送共用的出生场景坐标。
@export var login_and_map_teleport_spawn_position: Vector2 = Vector2(12.0, 10.0)

## 返回当前场景用于 HUD 展示的名称。
func get_scene_display_name() -> String:
    return scene_display_name

func _get_door_configs() -> Dictionary:
    return {
        "RightDoor": {
            "portal_id": 3001,
            "target_scene_id": 2,
        },
        "UpDoor": {
            "portal_id": 3002,
            "target_scene_id": 4,
        },
        "DownDoor": {
            "portal_id": 3003,
            "target_scene_id": 5,
        },
    }

func _get_default_facing_direction() -> Vector2:
    return Vector2.RIGHT

## 根据传入市场的 portal_id 返回客户端场景坐标；市场连接多个门，所以按来源 portal 分开配置。
func get_portal_spawn_scene_position(portal_id: int) -> Vector2:
    match portal_id:
        2002:
            return inbound_from_east_road_scene_position
        4001:
            return inbound_from_bei_lu_scene_position
        5001:
            return inbound_from_xue_xiao_scene_position
        _:
            return INVALID_PORTAL_SPAWN_SCENE_POSITION


## get_level_center_position 返回缩放居中用的 Godot 像素中心点；该值始终根据 TileMap 自动计算，不使用导出变量覆盖。
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

func get_level_scale_factor() -> float:
    return level_scale_factor

## 返回登录进入当前地图和世界地图快速传送共用的出生场景坐标。
func get_login_and_map_teleport_spawn_position() -> Vector2:
    return login_and_map_teleport_spawn_position

## 查找用于计算缩放居中中心点的 TileMap 图层；不同地图可能使用不同节点名。
func _resolve_map_layer() -> TileMapLayer:
    for layer_name in ["Map", "TileMapLayer"]:
        var layer: TileMapLayer = get_node_or_null(layer_name) as TileMapLayer
        if layer != null:
            return layer
    return null
