extends Resource
class_name TaskIconDefinition

## 客户端任务图标 ID；服务端 QuestSummary.client_icon_id 只引用该本地 ID。
@export var icon_id: int = 0

## 任务卡片静态图标，通常为 AtlasTexture 或独立贴图。
@export var static_texture: Texture2D = null


## 返回任务卡片展示用的静态贴图。
func resolve_preview_texture() -> Texture2D:
    return static_texture
