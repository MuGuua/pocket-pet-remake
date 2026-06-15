extends RefCounted
class_name CharacterSkinRegistry

const UNIT_SKIN_DIR: String = "res://resources/battle/unit_skins"

static var _unit_skins: Dictionary = {}
static var _is_loaded: bool = false

## 按 skin_id 加载 UnitSkin 资源，世界与战斗表现层共用同一份缓存。
static func get_unit_skin(skin_id: String) -> UnitSkin:
	var normalized_skin_id: String = skin_id.strip_edges()
	if normalized_skin_id.is_empty():
		return null
	_ensure_loaded()
	if not _unit_skins.has(normalized_skin_id):
		return null
	return _unit_skins[normalized_skin_id] as UnitSkin

static func _ensure_loaded() -> void:
	if _is_loaded:
		return
	_unit_skins.clear()
	var dir: DirAccess = DirAccess.open(UNIT_SKIN_DIR)
	if dir == null:
		push_warning("形象资源目录不存在: %s" % UNIT_SKIN_DIR)
		_is_loaded = true
		return
	dir.list_dir_begin()
	while true:
		var file_name: String = dir.get_next()
		if file_name.is_empty():
			break
		if dir.current_is_dir() or not file_name.ends_with(".tres"):
			continue
		var resource_path: String = "%s/%s" % [UNIT_SKIN_DIR, file_name]
		var resource: Resource = load(resource_path) as Resource
		if resource == null or not resource is UnitSkin:
			continue
		var skin: UnitSkin = resource as UnitSkin
		var resource_id: String = skin.skin_id.strip_edges()
		if resource_id.is_empty():
			push_warning("UnitSkin 缺少 skin_id: %s" % resource_path)
			continue
		_unit_skins[resource_id] = skin
	dir.list_dir_end()
	_is_loaded = true
