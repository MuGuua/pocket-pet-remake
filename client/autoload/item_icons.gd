extends Node

## 物品图标资源目录；每个物品单独一个 .tres，按 item_id 索引。
const ICONS_DIR: String = "res://resources/item_icons/"
## 未配置 item_id 或加载失败时使用的默认图标资源。
const DEFAULT_ICON_PATH: String = "res://resources/item_icons/_default.tres"
const FALLBACK_ATLAS_PATH: String = "res://asset/分类/武器/pixel items0.png"
const FALLBACK_ICON_SIZE: int = 32

var _definitions: Dictionary = {}
var _default_definition: ItemIconDefinition = null
var _fallback_texture: Texture2D = null
var _scanned: bool = false


func _ready() -> void:
    _scan_icon_directory()


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


## 扫描 item_icons 目录，构建 item_id 到 ItemIconDefinition 的索引。
func _scan_icon_directory() -> void:
    _definitions.clear()
    _default_definition = null
    _scanned = true
    _load_default_definition()
    var dir: DirAccess = DirAccess.open(ICONS_DIR)
    if dir == null:
        push_warning("ItemIcons: 无法打开目录 %s" % ICONS_DIR)
        return
    dir.list_dir_begin()
    var entry_name: String = dir.get_next()
    while entry_name != "":
        if not dir.current_is_dir() and entry_name.ends_with(".tres"):
            var resource_path: String = ICONS_DIR + entry_name
            if resource_path == DEFAULT_ICON_PATH:
                entry_name = dir.get_next()
                continue
            _register_icon_resource(resource_path)
        entry_name = dir.get_next()
    dir.list_dir_end()


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
        _scan_icon_directory()


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
