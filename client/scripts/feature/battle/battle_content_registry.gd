extends Node
class_name BattleContentRegistry

## 技能表现资源目录；目录内的 SkillVisualConfig 会按资源自身 ID 自动注册。
const SKILL_VISUAL_RESOURCE_DIRECTORY: String = "res://resources/battle/skill_visuals"

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
    for resource_path: String in _list_skill_visual_resource_paths():
        _register_resource(resource_path, _skill_visuals, "skill_visual_id", "SkillVisualConfig")
    _is_loaded = true

## 枚举导出包中可识别的技能资源，并排序以保证重复 ID 告警结果稳定。
func _list_skill_visual_resource_paths() -> Array[String]:
    var resource_paths: Array[String] = []
    var resource_names: PackedStringArray = ResourceLoader.list_directory(SKILL_VISUAL_RESOURCE_DIRECTORY)
    resource_names.sort()
    for resource_name: String in resource_names:
        var extension: String = resource_name.get_extension().to_lower()
        if extension != "tres" and extension != "res":
            continue
        var resource_path: String = resource_name
        if not resource_path.begins_with("res://"):
            resource_path = SKILL_VISUAL_RESOURCE_DIRECTORY.path_join(resource_name)
        resource_paths.append(resource_path)
    return resource_paths

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

## 按服务端下发的技能表现 ID 返回技能图标；未注册或未配置时返回 null。
func get_skill_icon(skill_visual_id: String) -> Texture2D:
    var visual: SkillVisualConfig = get_skill_visual(skill_visual_id)
    if visual == null:
        return null
    return visual.icon

## 从自动枚举得到的资源路径注册单个配置，兼容 Web 与桌面导出。
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
