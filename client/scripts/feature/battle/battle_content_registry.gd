extends Node
class_name BattleContentRegistry

const UNIT_SKIN_DIR: String = "res://resources/battle/unit_skins"
## 单位形象目录：每个 {skin_id}.tres 即完整形象（动画帧内嵌于 UnitSkin）
const SKILL_VISUAL_DIR: String = "res://resources/battle/skill_visuals"
## 技能表现目录：每个 {skill_visual_id}.tres 即完整技能表现（特效帧内嵌于 SkillVisualConfig）

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
	_load_resources_from_dir(UNIT_SKIN_DIR, _unit_skins, "skin_id", "UnitSkin")
	_register_unit_skin_aliases()
	_load_resources_from_dir(SKILL_VISUAL_DIR, _skill_visuals, "skill_visual_id", "SkillVisualConfig")
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

## 递归扫描资源目录，支持 unit_skins/人物 等子目录。
func _load_resources_from_dir(dir_path: String, target_map: Dictionary, id_field: String, expected_class: String) -> void:
	var dir: DirAccess = DirAccess.open(dir_path)
	if dir == null:
		push_warning("资源目录不存在: %s" % dir_path)
		return
	dir.list_dir_begin()
	while true:
		var file_name: String = dir.get_next()
		if file_name.is_empty():
			break
		var resource_path: String = "%s/%s" % [dir_path, file_name]
		if dir.current_is_dir():
			_load_resources_from_dir(resource_path, target_map, id_field, expected_class)
			continue
		if not file_name.ends_with(".tres"):
			continue
		var resource: Resource = load(resource_path) as Resource
		if resource == null:
			push_warning("资源加载失败: %s" % resource_path)
			continue
		if expected_class == "UnitSkin" and not resource is UnitSkin:
			continue
		if expected_class == "SkillVisualConfig" and not resource is SkillVisualConfig:
			continue
		var resource_id_value: Variant = resource.get(id_field)
		var resource_id: String = str(resource_id_value).strip_edges()
		if resource_id.is_empty():
			push_warning("资源缺少标识字段 %s: %s" % [id_field, resource_path])
			continue
		if target_map.has(resource_id):
			push_warning(
				"UnitSkin skin_id 重复，已跳过后者: id=%s 冲突=%s"
				% [resource_id, resource_path]
			)
			continue
		target_map[resource_id] = resource
		if expected_class == "UnitSkin" and resource is UnitSkin:
			_register_unit_skin_filename_alias(target_map, resource as UnitSkin, file_name)
	dir.list_dir_end()


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
