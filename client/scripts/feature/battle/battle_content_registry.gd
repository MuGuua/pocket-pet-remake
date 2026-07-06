extends Node
class_name BattleContentRegistry

## 技能表现资源路径清单；Web 导出后不能依赖 DirAccess 扫描目录。
const SKILL_VISUAL_RESOURCE_PATHS: Array[String] = [
	"res://resources/battle/skill_visuals/slash.tres",
	"res://resources/battle/skill_visuals/裂空斩.tres",
	"res://resources/battle/skill_visuals/酸液喷射_一级.tres",
]

var _unit_skins: Dictionary = {}
var _skill_visuals: Dictionary = {}
var _is_loaded: bool = false

func _ready() -> void:
	load_content()

func load_content() -> void:
	if _is_loaded:
		return
	_unit_skins.clear()
	_skill_visuals.clear()
	for resource_path: String in CharacterSkinRegistry.UNIT_SKIN_RESOURCE_PATHS:
		_register_resource(resource_path, _unit_skins, "skin_id", "UnitSkin")
	_register_unit_skin_aliases()
	for resource_path: String in SKILL_VISUAL_RESOURCE_PATHS:
		_register_resource(resource_path, _skill_visuals, "skill_visual_id", "SkillVisualConfig")
	_is_loaded = true

func get_unit_skin(skin_id: String) -> UnitSkin:
	load_content()
	var normalized_skin_id: String = skin_id.strip_edges()
	if normalized_skin_id.is_empty():
		return null
	if not _unit_skins.has(normalized_skin_id):
		return null
	return _unit_skins[normalized_skin_id] as UnitSkin

func get_skill_visual(skill_visual_id: String) -> SkillVisualConfig:
	load_content()
	var normalized_skill_visual_id: String = skill_visual_id.strip_edges()
	if normalized_skill_visual_id.is_empty():
		return null
	if not _skill_visuals.has(normalized_skill_visual_id):
		return null
	return _skill_visuals[normalized_skill_visual_id] as SkillVisualConfig

## 从显式资源路径注册单个 .tres，兼容 Web 与桌面导出。
func _register_resource(
		resource_path: String,
		target_map: Dictionary,
		id_field: String,
		expected_class: String
) -> void:
	var resource: Resource = load(resource_path) as Resource
	if resource == null:
		push_warning("资源加载失败: %s" % resource_path)
		return
	if expected_class == "UnitSkin" and not resource is UnitSkin:
		return
	if expected_class == "SkillVisualConfig" and not resource is SkillVisualConfig:
		return
	var resource_id_value: Variant = resource.get(id_field)
	var resource_id: String = str(resource_id_value).strip_edges()
	if resource_id.is_empty():
		push_warning("资源缺少标识字段 %s: %s" % [id_field, resource_path])
		return
	if target_map.has(resource_id):
		push_warning(
			"资源标识重复，已跳过后者: id=%s 冲突=%s"
			% [resource_id, resource_path]
		)
		return
	target_map[resource_id] = resource
	if expected_class == "UnitSkin" and resource is UnitSkin:
		_register_unit_skin_filename_alias(target_map, resource as UnitSkin, resource_path.get_file())

func _register_unit_skin_aliases() -> void:
	_register_unit_skin_alias_if_missing("决斗者_001", "初始形象女_002")

func _register_unit_skin_alias_if_missing(alias_id: String, target_id: String) -> void:
	var normalized_alias: String = alias_id.strip_edges()
	var normalized_target: String = target_id.strip_edges()
	if normalized_alias.is_empty() or normalized_target.is_empty():
		return
	if _unit_skins.has(normalized_alias):
		return
	if not _unit_skins.has(normalized_target):
		return
	_unit_skins[normalized_alias] = _unit_skins[normalized_target]

func _register_unit_skin_filename_alias(target_map: Dictionary, skin: UnitSkin, file_name: String) -> void:
	var file_stem: String = file_name.get_basename()
	if file_stem.is_empty():
		return
	var resource_id: String = skin.skin_id.strip_edges()
	if file_stem == resource_id:
		return
	if target_map.has(file_stem):
		return
	target_map[file_stem] = skin
