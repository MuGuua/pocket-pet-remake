extends Node

## 任务图标资源路径清单；Web 导出后不能依赖 DirAccess 扫描 res:// 目录。
## 新增任务图标时：在 resources/task_icons/ 下创建 .tres 后，把路径补进本数组。
const ICON_RESOURCE_PATHS: Array[String] = [
    "res://resources/task_icons/1主线默认.tres",
    "res://resources/task_icons/2对话任务.tres",
    "res://resources/task_icons/3战斗任务.tres",
]
## 未配置客户端图标 ID 或加载失败时使用的默认图标资源。
const DEFAULT_ICON_PATH: String = "res://resources/task_icons/1主线默认.tres"
const FALLBACK_ATLAS_PATH: String = "res://asset/分类/Free - Raven Fantasy Icons/Free - Raven Fantasy Icons/RPG Maker MV and MZ/IconSet.png"
const FALLBACK_ICON_SIZE: int = 32

## 客户端图标 ID 到任务图标定义的索引。
var _definitions: Dictionary = {}
## 默认任务图标定义。
var _default_definition: TaskIconDefinition = null
## 兜底运行时切片贴图，避免默认资源加载失败时卡片空白。
var _fallback_texture: Texture2D = null
## 是否已完成显式路径清单扫描。
var _scanned: bool = false


## 初始化任务图标注册表。
func _ready() -> void:
    _scan_icon_resources()


## 按服务端 client_icon_id 解析本地任务图标；未命中时回退默认图标。
func resolve_texture(icon_id: int) -> Texture2D:
    _ensure_scanned()
    var definition: TaskIconDefinition = _definitions.get(icon_id, null) as TaskIconDefinition
    if definition != null:
        var resolved: Texture2D = definition.resolve_preview_texture()
        if resolved != null:
            return resolved
    return _resolve_default_texture()


## 按显式路径清单加载任务图标定义，兼容 Web 与桌面导出。
func _scan_icon_resources() -> void:
    _definitions.clear()
    _default_definition = null
    _scanned = true
    _load_default_definition()
    for resource_path: String in ICON_RESOURCE_PATHS:
        _register_icon_resource(resource_path)


## 注册单个任务图标资源。
func _register_icon_resource(resource_path: String) -> void:
    var resource: Resource = load(resource_path)
    if resource == null:
        push_warning("TaskIcons: 无法加载 %s" % resource_path)
        return
    if resource is TaskIconDefinition:
        _register_definition(resource as TaskIconDefinition, resource_path)
        return
    push_warning("TaskIcons: 不支持的资源类型 %s" % resource_path)


## 将任务图标定义写入客户端图标 ID 索引。
func _register_definition(definition: TaskIconDefinition, resource_path: String) -> void:
    if definition == null:
        return
    if definition.icon_id <= 0:
        push_warning("TaskIcons: %s 的 icon_id 无效，已跳过" % resource_path)
        return
    if _definitions.has(definition.icon_id):
        push_warning("TaskIcons: icon_id=%s 重复，%s 覆盖旧配置" % [str(definition.icon_id), resource_path])
    _definitions[definition.icon_id] = definition


## 加载默认任务图标定义。
func _load_default_definition() -> void:
    var resource: Resource = load(DEFAULT_ICON_PATH)
    if resource is TaskIconDefinition:
        _default_definition = resource as TaskIconDefinition


## 解析默认任务图标；默认资源缺失时从 atlas 左上角切出兜底图标。
func _resolve_default_texture() -> Texture2D:
    if _default_definition != null:
        var default_texture: Texture2D = _default_definition.resolve_preview_texture()
        if default_texture != null:
            return default_texture
    return _build_fallback_texture()


## 确保图标注册表已经完成初始化。
func _ensure_scanned() -> void:
    if not _scanned:
        _scan_icon_resources()


## 构建运行时兜底图标。
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
