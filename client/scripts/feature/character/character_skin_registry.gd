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
	_load_unit_skins_from_dir(UNIT_SKIN_DIR)
	_register_legacy_skin_aliases()
	_is_loaded = true

## 递归扫描 unit_skins 目录，支持 人物/怪物/其他 等子目录结构。
static func _load_unit_skins_from_dir(dir_path: String) -> void:
	var dir: DirAccess = DirAccess.open(dir_path)
	if dir == null:
		if dir_path == UNIT_SKIN_DIR:
			push_warning("形象资源目录不存在: %s" % UNIT_SKIN_DIR)
		return
	dir.list_dir_begin()
	while true:
		var file_name: String = dir.get_next()
		if file_name.is_empty():
			break
		var resource_path: String = "%s/%s" % [dir_path, file_name]
		if dir.current_is_dir():
			_load_unit_skins_from_dir(resource_path)
			continue
		if not file_name.ends_with(".tres"):
			continue
		var resource: Resource = load(resource_path) as Resource
		if resource == null or not resource is UnitSkin:
			continue
		var skin: UnitSkin = resource as UnitSkin
		var resource_id: String = skin.skin_id.strip_edges()
		if resource_id.is_empty():
			push_warning("UnitSkin 缺少 skin_id: %s" % resource_path)
			continue
		if _unit_skins.has(resource_id):
			push_warning(
				"UnitSkin skin_id 重复，已跳过后者: id=%s 已有=%s 冲突=%s"
				% [resource_id, str(_unit_skins[resource_id]), resource_path]
			)
			continue
		_unit_skins[resource_id] = skin
		_register_skin_alias_from_filename(skin, file_name)
	dir.list_dir_end()


## 用 .tres 文件名注册别名，兼容后台填写「初始形象女_002」而资源内 skin_id 为其他值的情况。
static func _register_skin_alias_from_filename(skin: UnitSkin, file_name: String) -> void:
	var file_stem: String = file_name.get_basename()
	if file_stem.is_empty():
		return
	var resource_id: String = skin.skin_id.strip_edges()
	if file_stem == resource_id:
		return
	if _unit_skins.has(file_stem):
		return
	_unit_skins[file_stem] = skin


## 兼容历史 skin_id，避免数据库尚未迁移时找不到形象资源。
static func _register_legacy_skin_aliases() -> void:
	_register_alias_if_missing("决斗者_001", "初始形象女_002")


static func _register_alias_if_missing(alias_id: String, target_id: String) -> void:
	var normalized_alias: String = alias_id.strip_edges()
	var normalized_target: String = target_id.strip_edges()
	if normalized_alias.is_empty() or normalized_target.is_empty():
		return
	if _unit_skins.has(normalized_alias):
		return
	if not _unit_skins.has(normalized_target):
		return
	_unit_skins[normalized_alias] = _unit_skins[normalized_target]
