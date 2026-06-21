class_name CinematicRegistry
extends RefCounted

## 剧情动画键到客户端内置场景路径的映射表；服务端只下发稳定 key，不直接暴露资源路径。
static var CINEMATIC_PATH_BY_KEY: Dictionary = {
	"market_limeng_step_aside": "res://scenes/cinematics/market_limeng_step_aside.tscn"
}

## 根据服务端下发的剧情动画键返回本地可实例化的场景路径。
static func get_path_by_key(animation_key: String) -> String:
	if not CINEMATIC_PATH_BY_KEY.has(animation_key):
		return ""
	return str(CINEMATIC_PATH_BY_KEY.get(animation_key, ""))
