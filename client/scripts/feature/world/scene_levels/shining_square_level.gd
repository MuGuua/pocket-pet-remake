extends NetworkDoorLevelBase

## 闪耀广场地图脚本，负责把场景中的门节点转换为服务端权威切图请求。

## scene_display_name 是右上角 HUD 展示的当前场景名称。
@export var scene_display_name: String = "闪耀广场"
## level_scale_factor 是当前地图在移动端视口中的展示缩放倍数。
@export var level_scale_factor: float = 1.0
## login_spawn_position 是没有服务端权威位置时的默认场景坐标。
@export var login_spawn_position: Vector2 = Vector2(14.0, 8.0)

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


## 创建客户端切图意图；出生格不在客户端配置，由服务端 portal 拓扑返回。
func _create_door_config(portal_id: int, target_scene_id: int) -> Dictionary:
    return {
        "portal_id": portal_id,
        "target_scene_id": target_scene_id,
    }


## 返回角色没有有效朝向时使用的默认朝向。
func _get_default_facing_direction() -> Vector2:
    return Vector2.DOWN


## 出生位置统一使用服务端世界快照，闪耀广场不提供客户端覆盖值。
func get_portal_spawn_scene_position(_portal_id: int) -> Vector2:
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


## 返回没有服务端权威位置时的默认场景坐标。
func get_login_spawn_position() -> Vector2:
    return login_spawn_position
