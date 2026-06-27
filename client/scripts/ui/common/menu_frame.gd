extends Control
class_name MenuFrame

## 通用九宫格菜单边框场景路径。
const SCENE_PATH: String = "res://scenes/ui/common/menu_frame.tscn"

const HORIZONTAL_BORDER: Texture2D = preload("res://asset/分类/菜单表情/菜单ui边框横.png")
const VERTICAL_BORDER: Texture2D = preload("res://asset/分类/菜单表情/菜单ui边框竖.png")
const CORNER_BORDER: Texture2D = preload("res://asset/分类/菜单表情/菜单ui边框角落(左上角).png")

@export var background_color: Color = Color(0.06, 0.09, 0.14, 0.88)
@export var content_margin_left: int = 12
@export var content_margin_top: int = 10
@export var content_margin_right: int = 12
@export var content_margin_bottom: int = 10

@onready var background_rect: ColorRect = $Background
@onready var top_edge: TextureRect = $Top
@onready var bottom_edge: TextureRect = $Bottom
@onready var left_edge: TextureRect = $Left
@onready var right_edge: TextureRect = $Right
@onready var top_left_corner: TextureRect = $TopLeft
@onready var top_right_corner: TextureRect = $TopRight
@onready var bottom_left_corner: TextureRect = $BottomLeft
@onready var bottom_right_corner: TextureRect = $BottomRight
@onready var content_container: MarginContainer = $Content


## 初始化边框纹理与布局。
func _ready() -> void:
    background_rect.color = background_color
    _apply_textures()
    _update_layout()


## 尺寸变化时重新计算九宫格布局。
func _notification(what: int) -> void:
    if what == NOTIFICATION_RESIZED:
        _update_layout()


## 应用边框图集到各边与四角。
func _apply_textures() -> void:
    top_edge.texture = HORIZONTAL_BORDER
    bottom_edge.texture = HORIZONTAL_BORDER
    left_edge.texture = VERTICAL_BORDER
    right_edge.texture = VERTICAL_BORDER
    top_left_corner.texture = CORNER_BORDER
    top_right_corner.texture = CORNER_BORDER
    bottom_left_corner.texture = CORNER_BORDER
    bottom_right_corner.texture = CORNER_BORDER

    top_right_corner.flip_h = true
    bottom_left_corner.flip_v = true
    bottom_right_corner.flip_h = true
    bottom_right_corner.flip_v = true

    top_edge.stretch_mode = TextureRect.STRETCH_TILE
    bottom_edge.stretch_mode = TextureRect.STRETCH_TILE
    left_edge.stretch_mode = TextureRect.STRETCH_TILE
    right_edge.stretch_mode = TextureRect.STRETCH_TILE


## 按当前控件尺寸更新背景、边框与内容留白。
func _update_layout() -> void:
    if not is_node_ready():
        return

    var corner_size: Vector2 = Vector2(7.0, 7.0)
    var border_thickness: float = 3.0
    var control_size: Vector2 = size

    top_left_corner.position = Vector2.ZERO
    top_right_corner.position = Vector2(control_size.x - corner_size.x, 0.0)
    bottom_left_corner.position = Vector2(0.0, control_size.y - corner_size.y)
    bottom_right_corner.position = Vector2(control_size.x - corner_size.x, control_size.y - corner_size.y)

    top_edge.position = Vector2(corner_size.x, 0.0)
    top_edge.size = Vector2(max(control_size.x - corner_size.x * 2.0, 0.0), border_thickness)

    bottom_edge.position = Vector2(corner_size.x, control_size.y - border_thickness)
    bottom_edge.size = Vector2(max(control_size.x - corner_size.x * 2.0, 0.0), border_thickness)

    left_edge.position = Vector2(0.0, corner_size.y)
    left_edge.size = Vector2(border_thickness, max(control_size.y - corner_size.y * 2.0, 0.0))

    right_edge.position = Vector2(control_size.x - border_thickness, corner_size.y)
    right_edge.size = Vector2(border_thickness, max(control_size.y - corner_size.y * 2.0, 0.0))

    background_rect.position = Vector2(border_thickness, border_thickness)
    background_rect.size = Vector2(
        max(control_size.x - border_thickness * 2.0, 0.0),
        max(control_size.y - border_thickness * 2.0, 0.0)
    )

    content_container.offset_left = content_margin_left
    content_container.offset_top = content_margin_top
    content_container.offset_right = -content_margin_right
    content_container.offset_bottom = -content_margin_bottom
