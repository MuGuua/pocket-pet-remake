class_name GridSpreadTransition
extends ColorRect

## 网格铺展动画播放完毕时触发，供外部 await 或监听。
signal animation_completed

## 网格单元之间的间距（像素）。
@export var spacing: int = 32
## 对角线错峰延迟系数，越大铺展越慢。
@export var stagger_delay: float = 0.008
## 单个网格单元缩放动画时长（秒）。
@export var scale_duration: float = 0.16

## 承载铺展小球的 MultiMesh 渲染节点。
@onready var _balls_multimesh: MultiMeshInstance2D = %MultiMeshInstance2D

## 当前视口横向网格数量。
var _grid_width: int = 0
## 当前视口纵向网格数量。
var _grid_height: int = 0


func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	color = Color(1.0, 1.0, 1.0, 0.0)
	hide()


## 按当前视口尺寸重新计算网格，并把全部实例缩放到 0，供铺展转场复用。
func prepare_grid() -> void:
	_prepare_grid_layout()
	_reset_all_instances(0.0)


## 从透明铺展到全屏遮挡，用于进入战斗前盖住世界画面。
func play_cover() -> void:
	_prepare_grid_layout()
	_reset_all_instances(0.0)
	show()
	mouse_filter = Control.MOUSE_FILTER_STOP
	await _animate_all(0.0, 1.0)
	animation_completed.emit()


## 从全屏遮挡收缩到透明，用于露出已挂载的战斗弹窗。
func play_reveal() -> void:
	await _animate_all(1.0, 0.0)
	_reset_all_instances(0.0)
	hide()
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	animation_completed.emit()


## 根据视口大小计算网格行列数，并同步 MultiMesh 实例总量。
func _prepare_grid_layout() -> void:
	var viewport_size: Vector2 = get_viewport_rect().size
	_grid_width = int(ceil(viewport_size.x / float(spacing))) + 2
	_grid_height = int(ceil(viewport_size.y / float(spacing))) + 2
	var multimesh: MultiMesh = _balls_multimesh.multimesh
	multimesh.instance_count = _grid_width * _grid_height


## 把所有实例统一设置为指定缩放，常用于动画开始或结束后的复位。
func _reset_all_instances(scale_value: float) -> void:
	var multimesh: MultiMesh = _balls_multimesh.multimesh
	var index: int = 0
	for i in range(_grid_width):
		for j in range(_grid_height):
			var origin: Vector2 = Vector2(float(i), float(j)) * float(spacing)
			_apply_instance_scale(multimesh, index, origin, scale_value)
			index += 1


## 按对角线顺序播放缩放动画，from_scale 与 to_scale 决定铺展或收回方向。
func _animate_all(from_scale: float, to_scale: float) -> void:
	var multimesh: MultiMesh = _balls_multimesh.multimesh
	var index: int = 0
	for i in range(_grid_width):
		for j in range(_grid_height):
			var instance_index: int = index
			var origin: Vector2 = Vector2(float(i), float(j)) * float(spacing)
			var delay: float = float(i + j) * stagger_delay
			var tween: Tween = create_tween()
			tween.tween_interval(delay)
			# tween_method 始终把插值 float 作为回调的第一个参数，不能直接用 bind 绑定 multimesh。
			tween.tween_method(
				func(scale_value: float) -> void:
					_apply_instance_scale(multimesh, instance_index, origin, scale_value),
				from_scale,
				to_scale,
				scale_duration
			)
			index += 1
	var max_delay: float = float(_grid_width + _grid_height - 2) * stagger_delay
	var total_time: float = max_delay + scale_duration
	await get_tree().create_timer(total_time).timeout


## 写入单个 MultiMesh 实例的缩放与位置。
func _apply_instance_scale(multimesh: MultiMesh, instance_index: int, origin: Vector2, scale_value: float) -> void:
	var transform: Transform2D = Transform2D()
	transform.x = Vector2(scale_value, 0.0)
	transform.y = Vector2(0.0, scale_value)
	transform.origin = origin
	multimesh.set_instance_transform_2d(instance_index, transform)
