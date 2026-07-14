extends NetworkDoorLevelBase

## scene_display_name 是右上角 HUD 展示的当前场景名称，可在场景资源中覆盖。
@export var scene_display_name: String = "时光小屋"
## level_scale_factor 是当前地图在移动端视口中的展示缩放倍数。
@export var level_scale_factor: float = 1.0
## login_spawn_position 是首次进入或没有传送门上下文时的默认出生场景坐标。
@export var login_spawn_position: Vector2 = Vector2.ZERO
## LEFT_DOOR_PORTAL_ID 是时光小屋一次性出口的服务端传送门 ID。
const LEFT_DOOR_PORTAL_ID: int = 7001
## EAST_ROAD_SCENE_ID 是左侧出口唯一允许进入的闪光镇东路场景 ID。
const EAST_ROAD_SCENE_ID: int = 2


## 返回当前场景用于 HUD 展示的名称。
func get_scene_display_name() -> String:
	return scene_display_name


## 返回时光小屋的单向联网出口；东路不会配置返回时光小屋的门。
func _get_door_configs() -> Dictionary:
	return {
		"LeftDoor": {
			"portal_id": LEFT_DOOR_PORTAL_ID,
			"target_scene_id": EAST_ROAD_SCENE_ID,
		},
	}


## 返回没有额外门配置时沿用的角色朝向。
func _get_default_facing_direction() -> Vector2:
	return Vector2.LEFT


## 时光小屋暂未建立传入 portal 映射，交由世界控制器使用服务端坐标兜底。
func get_portal_spawn_scene_position(_portal_id: int) -> Vector2:
	return INVALID_PORTAL_SPAWN_SCENE_POSITION


## 根据场景内全部 TileMapLayer 的合并边界计算居中点，兼容该地图使用中文图层名。
func get_level_center_position() -> Vector2:
	var combined_rect: Rect2 = Rect2()
	var has_used_rect: bool = false
	for child: Node in get_children():
		if child is not TileMapLayer:
			continue
		var map_layer: TileMapLayer = child as TileMapLayer
		var used_rect: Rect2i = map_layer.get_used_rect()
		if not used_rect.has_area():
			continue
		var top_left: Vector2 = map_layer.to_global(map_layer.map_to_local(used_rect.position))
		var bottom_right: Vector2 = map_layer.to_global(map_layer.map_to_local(used_rect.position + used_rect.size))
		var layer_rect: Rect2 = Rect2(top_left, bottom_right - top_left).abs()
		combined_rect = layer_rect if not has_used_rect else combined_rect.merge(layer_rect)
		has_used_rect = true
	if not has_used_rect:
		return Vector2.ZERO
	return to_local(combined_rect.get_center())


## 返回移动端世界画布使用的地图缩放倍率。
func get_level_scale_factor() -> float:
	return level_scale_factor


## 返回没有传送门上下文时使用的默认出生坐标。
func get_login_spawn_position() -> Vector2:
	return login_spawn_position
