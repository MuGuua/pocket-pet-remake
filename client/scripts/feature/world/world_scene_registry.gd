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
    8: {
        "display_name": "闪光镇传送区",
        "scene_path": "res://scenes/maps/fashtown/闪光镇传送区.tscn",
        "grid_to_pixels": 16.0,
    },
    9: {
        "display_name": "闪耀广场",
        "scene_path": "res://scenes/maps/闪光平原/闪耀广场.tscn",
        "grid_to_pixels": 16.0,
    },
    10: {
        "display_name": "闪光平原宠物学校",
        "scene_path": "res://scenes/maps/闪光平原/闪光平原宠物学校.tscn",
        "grid_to_pixels": 16.0,
    },
    11: {
        "display_name": "冰雪梦境",
        "scene_path": "res://scenes/maps/闪光平原/冰雪梦境.tscn",
        "grid_to_pixels": 16.0,
    },
    12: {
        "display_name": "灰烬梦境",
        "scene_path": "res://scenes/maps/闪光平原/灰烬梦境.tscn",
        "grid_to_pixels": 16.0,
    },
    13: {
        "display_name": "翡翠梦境",
        "scene_path": "res://scenes/maps/闪光平原/翡翠梦境.tscn",
        "grid_to_pixels": 16.0,
    },
    14: {
        "display_name": "阿尔的房间",
        "scene_path": "res://scenes/maps/闪光平原/阿尔的房间.tscn",
        "grid_to_pixels": 16.0,
    },
    15: {
        "display_name": "办公区",
        "scene_path": "res://scenes/maps/闪光平原/办公区.tscn",
        "grid_to_pixels": 16.0,
    },
    16: {
        "display_name": "商业区",
        "scene_path": "res://scenes/maps/闪光平原/商业区.tscn",
        "grid_to_pixels": 16.0,
    },
    17: {
        "display_name": "报名区",
        "scene_path": "res://scenes/maps/闪光平原/报名区.tscn",
        "grid_to_pixels": 16.0,
    },
    18: {
        "display_name": "准备区",
        "scene_path": "res://scenes/maps/闪光平原/准备区.tscn",
        "grid_to_pixels": 16.0,
    },
    19: {
        "display_name": "家族会馆",
        "scene_path": "res://scenes/maps/闪光平原/家族会馆.tscn",
        "grid_to_pixels": 16.0,
    },
    20: {
        "display_name": "闪光南路",
        "scene_path": "res://scenes/maps/闪光平原/闪光南路.tscn",
        "grid_to_pixels": 16.0,
    },
    21: {
        "display_name": "五彩湖",
        "scene_path": "res://scenes/maps/闪光平原/五彩胡.tscn",
        "grid_to_pixels": 16.0,
    },
    22: {
        "display_name": "沼泽地",
        "scene_path": "res://scenes/maps/闪光平原/沼泽地.tscn",
        "grid_to_pixels": 16.0,
    },
    23: {
        "display_name": "闪光海岸",
        "scene_path": "res://scenes/maps/闪光平原/闪光海岸.tscn",
        "grid_to_pixels": 16.0,
    },
    24: {
        "display_name": "尘泥之地",
        "scene_path": "res://scenes/maps/闪光平原/尘泥之地.tscn",
        "grid_to_pixels": 16.0,
    },
    25: {
        "display_name": "精灵大厅",
        "scene_path": "res://scenes/maps/闪光平原/精灵大厅.tscn",
        "grid_to_pixels": 16.0,
    },
    26: {
        "display_name": "海道",
        "scene_path": "res://scenes/maps/闪光平原/海道.tscn",
        "grid_to_pixels": 16.0,
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
