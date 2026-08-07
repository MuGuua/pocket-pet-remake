extends NetworkDoorLevelBase

## 闪耀广场地图脚本，负责把场景中的门节点转换为服务端权威切图请求。

## scene_display_name 是右上角 HUD 展示的当前场景名称。
@export var scene_display_name: String = "闪耀广场"
## level_scale_factor 是当前地图在移动端视口中的展示缩放倍数。
@export var level_scale_factor: float = 1.0
@export_group("普通门切图落点（场景格）")
## 从闪光镇传送区 portal_id=8002 进入后，出生在闪耀广场西侧入口附近，可在地图根节点检查器中调整。
@export var inbound_from_transfer_area_scene_position: Vector2 = Vector2(20.0, 12.0)
## 从闪光平原宠物学校 portal_id=10001 返回后，出生在宠物学校入口附近。
@export var inbound_from_pet_school_scene_position: Vector2 = Vector2(16.0, 6.0)
## 从办公区 portal_id=15004 返回后，出生在冒险任务小屋入口附近。
@export var inbound_from_office_area_scene_position: Vector2 = Vector2(5.0, 7.0)
## 从商业区 portal_id=16001 返回后，出生在商业区入口附近。
@export var inbound_from_commercial_area_scene_position: Vector2 = Vector2(2.0, 8.0)
## 从报名区 portal_id=17001 返回后，出生在报名区入口附近。
@export var inbound_from_registration_area_scene_position: Vector2 = Vector2(23.0, 8.0)
## 从闪光南路 portal_id=20001 返回后，出生在闪光南路入口附近。
@export var inbound_from_south_road_scene_position: Vector2 = Vector2(12.0, 12.0)
@export_group("默认出生点（场景格）")
## login_and_map_teleport_spawn_position 是登录进入当前地图和世界地图快速传送共用的出生场景坐标。
@export var login_and_map_teleport_spawn_position: Vector2 = Vector2(14.0, 8.0)
@export_group("")

## SHANGUANG_TOWN_TRANSFER_AREA_SCENE_ID 是闪光镇传送区的服务端场景 ID。
const SHANGUANG_TOWN_TRANSFER_AREA_SCENE_ID: int = 8
## SHINING_PLAIN_PET_SCHOOL_SCENE_ID 是闪光平原宠物学校的服务端场景 ID。
const SHINING_PLAIN_PET_SCHOOL_SCENE_ID: int = 10
## OFFICE_AREA_SCENE_ID 是办公区的服务端场景 ID。
const OFFICE_AREA_SCENE_ID: int = 15
## COMMERCIAL_AREA_SCENE_ID 是商业区的服务端场景 ID。
const COMMERCIAL_AREA_SCENE_ID: int = 16
## REGISTRATION_AREA_SCENE_ID 是报名区的服务端场景 ID。
const REGISTRATION_AREA_SCENE_ID: int = 17
## SHINING_SOUTH_ROAD_SCENE_ID 是闪光南路的服务端场景 ID。
const SHINING_SOUTH_ROAD_SCENE_ID: int = 20


## 返回当前场景用于 HUD 展示的名称。
func get_scene_display_name() -> String:
    return scene_display_name


## 返回闪耀广场现有门节点对应的服务端传送门编号与目标场景。
func _get_door_configs() -> Dictionary:
    return {
        "通往闪光镇传送区": _create_door_config(9001, SHANGUANG_TOWN_TRANSFER_AREA_SCENE_ID),
        "通往闪光南路": _create_door_config(9002, SHINING_SOUTH_ROAD_SCENE_ID),
        "通往商业区": _create_door_config(9003, COMMERCIAL_AREA_SCENE_ID),
        "通往冒险任务小屋": _create_door_config(9004, OFFICE_AREA_SCENE_ID),
        "通往宠物学校": _create_door_config(9005, SHINING_PLAIN_PET_SCHOOL_SCENE_ID),
        "通往报名区": _create_door_config(9006, REGISTRATION_AREA_SCENE_ID),
    }


## 创建客户端切图意图；目标场景脚本会根据 portal_id 选择入口，服务端继续验证拓扑。
func _create_door_config(portal_id: int, target_scene_id: int) -> Dictionary:
    return {
        "portal_id": portal_id,
        "target_scene_id": target_scene_id,
    }


## 返回角色没有有效朝向时使用的默认朝向。
func _get_default_facing_direction() -> Vector2:
    return Vector2.DOWN


## 根据来源场景触发的 portal_id 返回闪耀广场本地入口落点；服务端只校验门关系，不接收该坐标。
## portal_id 是来源场景的传送门编号；未配置时返回无效值并回退当前场景脚本的统一出生点。
func get_portal_spawn_scene_position(portal_id: int) -> Vector2:
    match portal_id:
        8002:
            return inbound_from_transfer_area_scene_position
        10001:
            return inbound_from_pet_school_scene_position
        15004:
            return inbound_from_office_area_scene_position
        16001:
            return inbound_from_commercial_area_scene_position
        17001:
            return inbound_from_registration_area_scene_position
        20001:
            return inbound_from_south_road_scene_position
        _:
            return INVALID_PORTAL_SPAWN_SCENE_POSITION


## 返回地图已使用区域的中心点，供移动端视口居中展示。
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


## 返回当前地图在移动端视口中的展示缩放倍数。
func get_level_scale_factor() -> float:
    return level_scale_factor


## 返回登录进入当前地图和世界地图快速传送共用的出生场景坐标。
func get_login_and_map_teleport_spawn_position() -> Vector2:
    return login_and_map_teleport_spawn_position
