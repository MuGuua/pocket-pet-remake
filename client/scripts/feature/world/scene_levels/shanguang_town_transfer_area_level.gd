extends NetworkDoorLevelBase

## 闪光镇传送区连接闪光镇东路与闪光平原入口。

@export var scene_display_name: String = "闪光镇传送区"
@export var level_scale_factor: float = 1.0
@export_group("普通门切图落点（场景格）")
## 从闪光镇东路 portal_id=2003 进入后，出生在传送区左侧门附近，可在地图根节点检查器中调整。
@export var inbound_from_east_road_scene_position: Vector2 = Vector2(2.0, 12.0)
## 从闪耀广场 portal_id=9001 返回后，出生在传送区通往平原的门附近。
@export var inbound_from_shining_square_scene_position: Vector2 = Vector2(6.0, 9.0)
@export_group("默认出生点（场景格）")
## 首次进入或没有权威位置时的默认场景坐标。
@export var login_spawn_position: Vector2 = Vector2(4.0, 6.0)
@export_group("")

const EAST_ROAD_SCENE_ID: int = 2
const SHINING_SQUARE_SCENE_ID: int = 9
const TO_EAST_ROAD_PORTAL_ID: int = 8001
const TO_SHINING_SQUARE_PORTAL_ID: int = 8002


func get_scene_display_name() -> String:
    return scene_display_name


## 两个 Area2D 可以放在“传送门”容器内，基类会递归查找并绑定 body_entered。
func _get_door_configs() -> Dictionary:
    return {
        "通往闪光镇东路": {
            "portal_id": TO_EAST_ROAD_PORTAL_ID,
            "target_scene_id": EAST_ROAD_SCENE_ID,
            "facing_direction": Vector2.LEFT,
        },
        "通往闪光平原": {
            "portal_id": TO_SHINING_SQUARE_PORTAL_ID,
            "target_scene_id": SHINING_SQUARE_SCENE_ID,
            "facing_direction": Vector2.UP,
        },
    }


func _get_default_facing_direction() -> Vector2:
    return Vector2.UP


## 根据来源场景触发的 portal_id 返回传送区入口落点，供切图请求提交给服务端校验和持久化。
## portal_id 是来源场景的传送门编号；未配置时返回无效值并回退服务端兼容落点。
func get_portal_spawn_scene_position(portal_id: int) -> Vector2:
    match portal_id:
        2003:
            return inbound_from_east_road_scene_position
        9001:
            return inbound_from_shining_square_scene_position
        _:
            return INVALID_PORTAL_SPAWN_SCENE_POSITION


func get_level_center_position() -> Vector2:
    var map_layer: TileMapLayer = get_node_or_null("地图") as TileMapLayer
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


func get_login_spawn_position() -> Vector2:
    return login_spawn_position
