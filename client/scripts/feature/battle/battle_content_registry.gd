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
	_load_resources_from_dir(UNIT_SKIN_DIR, _unit_skins, "skin_id")
	_load_resources_from_dir(SKILL_VISUAL_DIR, _skill_visuals, "skill_visual_id")
	_is_loaded = true

func get_unit_skin(skin_id: String) -> UnitSkin:
	load_content()
	if not _unit_skins.has(skin_id):
		return null
	return _unit_skins[skin_id] as UnitSkin

func get_skill_visual(skill_visual_id: String) -> SkillVisualConfig:
	load_content()
	if not _skill_visuals.has(skill_visual_id):
		return null
	return _skill_visuals[skill_visual_id] as SkillVisualConfig

func _load_resources_from_dir(dir_path: String, target_map: Dictionary, id_field: String) -> void:
	var dir: DirAccess = DirAccess.open(dir_path)
	if dir == null:
		push_warning("资源目录不存在: %s" % dir_path)
		return
	dir.list_dir_begin()
	while true:
		var file_name: String = dir.get_next()
		if file_name.is_empty():
			break
		if dir.current_is_dir() or not file_name.ends_with(".tres"):
			continue
		var resource_path: String = "%s/%s" % [dir_path, file_name]
		var resource: Resource = load(resource_path) as Resource
		if resource == null:
			push_warning("资源加载失败: %s" % resource_path)
			continue
		if dir_path == UNIT_SKIN_DIR and not resource is UnitSkin:
			continue
		if dir_path == SKILL_VISUAL_DIR and not resource is SkillVisualConfig:
			continue
		var resource_id_value: Variant = resource.get(id_field)
		var resource_id: String = str(resource_id_value)
		if resource_id.is_empty():
			push_warning("资源缺少标识字段 %s: %s" % [id_field, resource_path])
			continue
		target_map[resource_id] = resource
	dir.list_dir_end()
