extends Node

## 物品图标资源路径清单；Web 导出后不能依赖 DirAccess 扫描 res:// 目录。
## 新增图标时：在 resources/item_icons/ 下创建 .tres 后，把路径补进本数组（_default.tres 除外）。
const ICON_RESOURCE_PATHS: Array[String] = [
    "res://resources/item_icons/10001飞行之羽.tres",
    "res://resources/item_icons/10002轻锻宝石.tres",
    "res://resources/item_icons/10003精炼宝石.tres",
    "res://resources/item_icons/10004神迹三星石.tres",
    "res://resources/item_icons/10005珍奇宠召唤卷轴.tres",
    "res://resources/item_icons/3001精灵腰包.tres",
    "res://resources/item_icons/3004新手礼包.tres",
    "res://resources/item_icons/3201锻造宝石.tres",
    "res://resources/item_icons/3202修复宝石.tres",
    "res://resources/item_icons/4001新手长剑.tres",
    "res://resources/item_icons/4002新手长剑（10）.tres",
    "res://resources/item_icons/4002新手长剑（20）.tres",
    "res://resources/item_icons/金币.tres",
    "res://resources/item_icons/银币.tres",
    "res://resources/item_icons/铜币.tres",
]
## 未配置 item_id 或加载失败时使用的默认图标资源。
const DEFAULT_ICON_PATH: String = "res://resources/item_icons/_default.tres"
const FALLBACK_ATLAS_PATH: String = "res://asset/分类/武器/pixel items0.png"
const FALLBACK_ICON_SIZE: int = 32

var _definitions: Dictionary = {}
var _default_definition: ItemIconDefinition = null
var _fallback_texture: Texture2D = null
var _scanned: bool = false


func _ready() -> void:
    _scan_icon_resources()


## 按 item_id 解析本地静态预览贴图；未命中时回退默认图标。
func resolve_texture(item_id: int) -> Texture2D:
    _ensure_scanned()
    var definition: ItemIconDefinition = _definitions.get(item_id, null) as ItemIconDefinition
    if definition != null:
        var resolved: Texture2D = definition.resolve_preview_texture()
        if resolved != null:
            return resolved
    return _resolve_default_texture()


## 按 item_id 获取完整图标定义，供需要帧动画的 UI 使用。
func resolve_definition(item_id: int) -> ItemIconDefinition:
    _ensure_scanned()
    return _definitions.get(item_id, null) as ItemIconDefinition


## 按显式路径清单加载图标定义，兼容 Web 与桌面导出。
func _scan_icon_resources() -> void:
    _definitions.clear()
    _default_definition = null
    _scanned = true
    _load_default_definition()
    for resource_path: String in ICON_RESOURCE_PATHS:
        _register_icon_resource(resource_path)


## 注册单个 .tres：支持 ItemIconDefinition，以及纯数字文件名的 Texture2D。
func _register_icon_resource(resource_path: String) -> void:
    var resource: Resource = load(resource_path)
    if resource == null:
        push_warning("ItemIcons: 无法加载 %s" % resource_path)
        return
    if resource is ItemIconDefinition:
        _register_definition(resource as ItemIconDefinition, resource_path)
        return
    if resource is Texture2D:
        _register_texture_fallback(resource as Texture2D, resource_path)
        return
    push_warning("ItemIcons: 不支持的资源类型 %s" % resource_path)


## 将 ItemIconDefinition 写入 item_id 索引。
func _register_definition(definition: ItemIconDefinition, resource_path: String) -> void:
    if definition == null:
        return
    if definition.item_id <= 0:
        push_warning("ItemIcons: %s 的 item_id 无效，已跳过" % resource_path)
        return
    if _definitions.has(definition.item_id):
        push_warning(
            "ItemIcons: item_id=%s 重复，%s 覆盖旧配置"
            % [str(definition.item_id), resource_path]
        )
    _definitions[definition.item_id] = definition


## 兼容 item_id.tres 直接存放 Texture2D 的命名方式。
func _register_texture_fallback(texture: Texture2D, resource_path: String) -> void:
    var file_stem: String = resource_path.get_file().get_basename()
    if not file_stem.is_valid_int():
        push_warning("ItemIcons: %s 不是 ItemIconDefinition，且文件名不是 item_id" % resource_path)
        return
    var parsed_item_id: int = int(file_stem)
    if parsed_item_id <= 0:
        return
    var wrapper: ItemIconDefinition = ItemIconDefinition.new()
    wrapper.item_id = parsed_item_id
    wrapper.static_texture = texture
    _register_definition(wrapper, resource_path)


func _load_default_definition() -> void:
    var resource: Resource = load(DEFAULT_ICON_PATH)
    if resource is ItemIconDefinition:
        _default_definition = resource as ItemIconDefinition


func _resolve_default_texture() -> Texture2D:
    if _default_definition != null:
        var default_texture: Texture2D = _default_definition.resolve_preview_texture()
        if default_texture != null:
            return default_texture
    return _build_fallback_texture()


func _ensure_scanned() -> void:
    if not _scanned:
        _scan_icon_resources()


func _build_fallback_texture() -> Texture2D:
    if _fallback_texture != null:
        return _fallback_texture
    var atlas_source: Resource = load(FALLBACK_ATLAS_PATH)
    if atlas_source is not Texture2D:
        return null
    var atlas_texture: AtlasTexture = AtlasTexture.new()
    atlas_texture.atlas = atlas_source as Texture2D
    atlas_texture.region = Rect2(0.0, 0.0, float(FALLBACK_ICON_SIZE), float(FALLBACK_ICON_SIZE))
    _fallback_texture = atlas_texture
    return _fallback_texture
