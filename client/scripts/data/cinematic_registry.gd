class_name CinematicRegistry
extends RefCounted

## 客户端剧情场景目录；按顺序兼容通用目录与项目中文剧情动画目录。
const CINEMATIC_DIRECTORIES: Array[String] = [
    "res://scenes/cinematics",
    "res://剧情动画"
]

## 根据服务端下发的剧情动画键返回本地可实例化的场景路径。
static func get_path_by_key(animation_key: String) -> String:
    var normalized_key: String = animation_key.strip_edges()
    if normalized_key.is_empty() or normalized_key.get_file() != normalized_key or normalized_key.get_basename() != normalized_key:
        return ""
    for cinematic_directory: String in CINEMATIC_DIRECTORIES:
        var scene_path: String = cinematic_directory.path_join(normalized_key + ".tscn")
        if ResourceLoader.exists(scene_path, "PackedScene"):
            return scene_path
    return ""
