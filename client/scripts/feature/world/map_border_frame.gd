@tool
class_name MapBorderFrame
extends Node2D

## MapBorderFrame 用九宫格贴图绘制地图外框，并同步生成四面不可穿越的碰撞墙。
## 直接在编辑器中调整 border_size 即可无失真地拉伸边框，四角始终保持原始像素。

## 边框贴图；建议使用四周为装饰边、中心可平铺的图片。
@export var border_texture: Texture2D:
	set(value):
		border_texture = value
		_rebuild()

## 边框覆盖的总尺寸（Godot 像素），左上角对齐本节点原点。
@export var border_size: Vector2 = Vector2(240.0, 208.0):
	set(value):
		border_size = Vector2(maxf(value.x, 1.0), maxf(value.y, 1.0))
		_rebuild()

## 九宫格四边的保护边距（像素），这些区域不会被拉伸。
@export var patch_margin_left: int = 2:
	set(value):
		patch_margin_left = maxi(value, 0)
		_rebuild()
@export var patch_margin_top: int = 2:
	set(value):
		patch_margin_top = maxi(value, 0)
		_rebuild()
@export var patch_margin_right: int = 2:
	set(value):
		patch_margin_right = maxi(value, 0)
		_rebuild()
@export var patch_margin_bottom: int = 2:
	set(value):
		patch_margin_bottom = maxi(value, 0)
		_rebuild()

## 边条的平铺方式：STRETCH 拉伸、TILE 整块重复、TILE_FIT 重复且不裁切。
@export_enum("STRETCH", "TILE", "TILE_FIT") var axis_stretch_mode: int = 1:
	set(value):
		axis_stretch_mode = value
		_rebuild()

## 碰撞墙的厚度（像素）；通常与边框可视厚度一致。
@export var wall_thickness: float = 4.0:
	set(value):
		wall_thickness = maxf(value, 0.5)
		_rebuild()

## 碰撞墙是否向边框内侧收缩；关闭时墙体压在边框外沿。
@export var wall_inside: bool = true:
	set(value):
		wall_inside = value
		_rebuild()

## 碰撞体所在的物理层，需与玩家移动检测的层保持一致。
@export_flags_2d_physics var collision_layer_bits: int = 1:
	set(value):
		collision_layer_bits = value
		_rebuild()

## 是否生成碰撞墙；纯装饰边框可关闭。
@export var generate_collision: bool = true:
	set(value):
		generate_collision = value
		_rebuild()

const _PATCH_NODE_NAME: String = "BorderPatch"
const _BODY_NODE_NAME: String = "BorderWalls"
const _WALL_NAMES: PackedStringArray = ["WallTop", "WallBottom", "WallLeft", "WallRight"]

var _patch: NinePatchRect
var _body: StaticBody2D

func _ready() -> void:
	_rebuild()

## 按当前导出参数重建边框贴图与碰撞墙。
func _rebuild() -> void:
	if not is_inside_tree():
		return
	_ensure_nodes()
	_apply_patch()
	_apply_walls()

## 确保 NinePatchRect 与 StaticBody2D 子节点存在。
func _ensure_nodes() -> void:
	_patch = get_node_or_null(_PATCH_NODE_NAME) as NinePatchRect
	if _patch == null:
		_patch = NinePatchRect.new()
		_patch.name = _PATCH_NODE_NAME
		_patch.mouse_filter = Control.MOUSE_FILTER_IGNORE
		add_child(_patch)
		_adopt(_patch)

	_body = get_node_or_null(_BODY_NODE_NAME) as StaticBody2D
	if _body == null:
		_body = StaticBody2D.new()
		_body.name = _BODY_NODE_NAME
		add_child(_body)
		_adopt(_body)

	for wall_name in _WALL_NAMES:
		if _body.get_node_or_null(wall_name) == null:
			var shape_node := CollisionShape2D.new()
			shape_node.name = wall_name
			shape_node.shape = RectangleShape2D.new()
			_body.add_child(shape_node)
			_adopt(shape_node)

## 在编辑器中把运行时创建的节点挂到当前场景，便于查看结构。
func _adopt(node: Node) -> void:
	if Engine.is_editor_hint():
		var scene_root: Node = get_tree().edited_scene_root
		if scene_root != null:
			node.owner = scene_root

## 刷新九宫格贴图的尺寸与边距。
func _apply_patch() -> void:
	if _patch == null:
		return
	_patch.texture = border_texture
	_patch.position = Vector2.ZERO
	_patch.size = border_size
	_patch.patch_margin_left = patch_margin_left
	_patch.patch_margin_top = patch_margin_top
	_patch.patch_margin_right = patch_margin_right
	_patch.patch_margin_bottom = patch_margin_bottom
	_patch.axis_stretch_horizontal = axis_stretch_mode
	_patch.axis_stretch_vertical = axis_stretch_mode
	_patch.draw_center = false

## 刷新四面碰撞墙的位置与尺寸，使其始终贴合当前边框大小。
func _apply_walls() -> void:
	if _body == null:
		return
	_body.collision_layer = collision_layer_bits
	_body.collision_mask = 0
	_body.process_mode = Node.PROCESS_MODE_INHERIT

	var half: float = wall_thickness * 0.5
	var offset: float = half if wall_inside else -half
	var rects: Dictionary = {
		"WallTop": [Vector2(border_size.x, wall_thickness), Vector2(border_size.x * 0.5, offset)],
		"WallBottom": [Vector2(border_size.x, wall_thickness), Vector2(border_size.x * 0.5, border_size.y - offset)],
		"WallLeft": [Vector2(wall_thickness, border_size.y), Vector2(offset, border_size.y * 0.5)],
		"WallRight": [Vector2(wall_thickness, border_size.y), Vector2(border_size.x - offset, border_size.y * 0.5)],
	}

	for wall_name in _WALL_NAMES:
		var shape_node: CollisionShape2D = _body.get_node_or_null(wall_name) as CollisionShape2D
		if shape_node == null:
			continue
		var rect_shape: RectangleShape2D = shape_node.shape as RectangleShape2D
		if rect_shape == null:
			rect_shape = RectangleShape2D.new()
			shape_node.shape = rect_shape
		var entry: Array = rects[wall_name]
		rect_shape.size = entry[0]
		shape_node.position = entry[1]
		shape_node.disabled = not generate_collision
