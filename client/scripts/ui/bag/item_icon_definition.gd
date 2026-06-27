extends Resource
class_name ItemIconDefinition

## 与服务端 item_definition.item_id 对齐；大于 0 时才会被 ItemIcons 扫描注册。
@export var item_id: int = 0

## 静态图标，通常为 AtlasTexture 或独立贴图。
@export var static_texture: Texture2D = null

## 可选帧动画资源；配置后 UI 可切换为 AnimatedSprite2D 等播放方式。
@export var sprite_frames: SpriteFrames = null

## sprite_frames 中默认播放的动画名；为空时使用资源里的第一个动画。
@export var animation: String = ""


## 返回详情/格子等静态展示用的首帧或静态贴图。
func resolve_preview_texture() -> Texture2D:
    if static_texture != null:
        return static_texture
    if sprite_frames == null:
        return null
    var animation_name: String = _resolve_animation_name()
    if animation_name.is_empty():
        return null
    if sprite_frames.get_frame_count(animation_name) <= 0:
        return null
    return sprite_frames.get_frame_texture(animation_name, 0)


## 当前定义是否包含可播放的帧动画。
func is_animated() -> bool:
    if sprite_frames == null:
        return false
    return not sprite_frames.get_animation_names().is_empty()


## 解析实际使用的动画名：优先 export 字段，否则取第一个动画。
func resolve_animation_name() -> String:
    return _resolve_animation_name()


func _resolve_animation_name() -> String:
    if sprite_frames == null:
        return ""
    var trimmed_animation: String = animation.strip_edges()
    if not trimmed_animation.is_empty() and sprite_frames.has_animation(trimmed_animation):
        return trimmed_animation
    var animation_names: PackedStringArray = sprite_frames.get_animation_names()
    if animation_names.is_empty():
        return ""
    return animation_names[0]
