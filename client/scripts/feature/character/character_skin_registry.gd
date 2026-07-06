extends RefCounted
class_name CharacterSkinRegistry

## Web 导出后无法使用 DirAccess 扫描 res:// 目录，因此这里显式列出全部 UnitSkin 资源路径。
## 新增形象时：在 resources/battle/unit_skins/ 下创建 .tres 后，把路径补进本数组。
const UNIT_SKIN_RESOURCE_PATHS: Array[String] = [
	"res://resources/battle/unit_skins/人物/初始形象男_001.tres",
	"res://resources/battle/unit_skins/人物/初始形象女_002.tres",
	"res://resources/battle/unit_skins/人物/法师男.tres",
	"res://resources/battle/unit_skins/宠物/狮子王chj_486.tres",
	"res://resources/battle/unit_skins/宠物/白色幻影chj_830.tres",
	"res://resources/battle/unit_skins/宠物/魅惑猫妖chj_834.tres",
	"res://resources/battle/unit_skins/怪物/螳螂怪_003.tres",
	"res://resources/battle/unit_skins/其他/战斗待机_004.tres",
	"res://resources/battle/unit_skins/其他/CHJ测试_2057.tres",
]

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
	for resource_path: String in UNIT_SKIN_RESOURCE_PATHS:
		_register_unit_skin_resource(resource_path)
	_register_legacy_skin_aliases()
	_is_loaded = true


## 从单个 .tres 注册 UnitSkin，并同步文件名别名。
static func _register_unit_skin_resource(resource_path: String) -> void:
	var resource: Resource = load(resource_path) as Resource
	if resource == null or not resource is UnitSkin:
		push_warning("UnitSkin 资源加载失败: %s" % resource_path)
		return
	var skin: UnitSkin = resource as UnitSkin
	var resource_id: String = skin.skin_id.strip_edges()
	if resource_id.is_empty():
		push_warning("UnitSkin 缺少 skin_id: %s" % resource_path)
		return
	if _unit_skins.has(resource_id):
		push_warning(
			"UnitSkin skin_id 重复，已跳过后者: id=%s 已有=%s 冲突=%s"
			% [resource_id, str(_unit_skins[resource_id]), resource_path]
		)
		return
	_unit_skins[resource_id] = skin
	var file_name: String = resource_path.get_file()
	_register_skin_alias_from_filename(skin, file_name)


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
