class_name WorldSceneRegistry
extends RefCounted

## 服务端 scene_id 与客户端地图场景资源的映射表；登录预加载与世界渲染共用同一份配置。
const SCENE_CONFIGS: Dictionary = {
    1: {
        "display_name": "洛克斯小屋",
        "scene_path": "res://scenes/maps/fashtown/roxus_house.tscn",
        "grid_to_pixels": 24.0,
    },
    2: {
        "display_name": "闪光镇东路",
        "scene_path": "res://scenes/maps/fashtown/east_road_of_shanguang_town.tscn",
        "grid_to_pixels": 24.0,
    },
    3: {
        "display_name": "闪光市场",
        "scene_path": "res://scenes/maps/fashtown/radiant_market.tscn",
        "grid_to_pixels": 24.0,
    },
    4: {
        "display_name": "北路",
        "scene_path": "res://scenes/maps/fashtown/bei_lu.tscn",
        "grid_to_pixels": 24.0,
    },
    5: {
        "display_name": "学校",
        "scene_path": "res://scenes/maps/fashtown/xue_xiao.tscn",
        "grid_to_pixels": 24.0,
    },
    6: {
        "display_name": "打怪区",
        "scene_path": "res://scenes/maps/fashtown/da_guai_qu.tscn",
        "grid_to_pixels": 24.0,
    },
    7: {
        "display_name": "时光小屋",
        "scene_path": "res://scenes/maps/fashtown/时光小屋.tscn",
        "grid_to_pixels": 24.0,
    },
}


## 读取指定场景 ID 的配置字典；不存在时返回空字典。
static func get_scene_config(scene_id: int) -> Dictionary:
    var scene_config_variant: Variant = SCENE_CONFIGS.get(scene_id, {})
    return scene_config_variant if scene_config_variant is Dictionary else {}


## 读取指定场景 ID 对应的 Godot 场景路径；无效时返回空字符串。
static func get_scene_path(scene_id: int) -> String:
    return str(get_scene_config(scene_id).get("scene_path", "")).strip_edges()


## 读取指定场景 ID 的玩家可见名称；未配置时返回空字符串，由调用方决定兜底文案。
static func get_scene_display_name(scene_id: int) -> String:
    return str(get_scene_config(scene_id).get("display_name", "")).strip_edges()


## 判断指定场景 ID 的地图资源是否存在且可被加载。
static func can_load_scene_map(scene_id: int) -> bool:
    var scene_path: String = get_scene_path(scene_id)
    if scene_path.is_empty():
        return false
    if not ResourceLoader.exists(scene_path):
        return false
    var resource: Resource = load(scene_path)
    return resource is PackedScene
