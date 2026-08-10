extends Control
class_name EllipseCarouselDemo

## 基础椭圆被均分的点位数量；上、下半圆会从这些基础点中按不同间隔抽样。
@export_range(12, 64, 4) var ellipse_point_count: int = 32
## 上半圆抽样间隔；数值越小，上半圆保留的图标越密集。
@export_range(1, 8, 1) var upper_sample_step: int = 2
## 下半圆抽样间隔；数值越大，下半圆保留的图标越稀疏。
@export_range(1, 8, 1) var lower_sample_step: int = 4
## 图标围绕该局部坐标中心排列，便于直接在场景中调整整体构图。
@export var ellipse_center: Vector2 = Vector2(390.0, 600.0)
## 椭圆横向半径，适配项目 780 像素宽的移动端设计视口。
@export_range(120.0, 360.0, 1.0) var ellipse_radius_x: float = 292.0
## 椭圆纵向半径；较扁的纵深让轮播更接近正视角，并允许前后图标自然遮挡。
@export_range(30.0, 120.0, 1.0) var ellipse_radius_y: float = 50.0
## 单次向左或向右轮转的补间时长。
@export_range(0.1, 1.5, 0.05) var rotate_duration_seconds: float = 0.42
## 上半圆最远位置的最小缩放值。
@export_range(0.2, 1.0, 0.05) var minimum_icon_scale: float = 0.52
## 下半圆最近位置的最大缩放值。
@export_range(0.5, 1.6, 0.05) var maximum_icon_scale: float = 1.22
## 上半圆最远位置的最小透明度。
@export_range(0.1, 1.0, 0.05) var minimum_icon_alpha: float = 0.34
## 下半圆最近位置的最大透明度。
@export_range(0.5, 1.0, 0.05) var maximum_icon_alpha: float = 1.0

## 承载全部图标节点的 Y 排序父节点。
@onready var _icon_sort_root: Node2D = $CarouselViewport/IconSortRoot
## 显示当前位于最前方的图标名称。
@onready var _selection_label: Label = $SelectionLabel
## 向左轮转的移动端触摸按钮。
@onready var _left_button: Button = $LeftButton
## 向右轮转的移动端触摸按钮。
@onready var _right_button: Button = $RightButton

## 按椭圆前后顺序保存的抽样点位；第一个点固定为下半圆正中央。
var _sampled_positions: Array[Vector2] = []
## 按当前点位顺序保存的图标节点；轮转后该数组会同步循环移动。
var _ordered_icons: Array[Node2D] = []
## 当前正在播放的轮转补间；用于阻止连续输入造成动画互相覆盖。
var _rotation_tween: Tween = null
## 当前是否正在轮转，动画完成前暂时锁定按钮和方向键输入。
var _is_rotating: bool = false


## 初始化抽样点、收集场景中预置的图标，并绑定桌面端与移动端按钮。
func _ready() -> void:
    _sampled_positions = _build_sampled_positions()
    _ordered_icons = _collect_icon_nodes()
    _validate_icon_count()
    _apply_layout_immediately()
    _left_button.pressed.connect(_on_left_button_pressed)
    _right_button.pressed.connect(_on_right_button_pressed)
    _update_selection_label()


## 支持键盘左右方向键快速检查轮转，也保留场景按钮供移动端触摸操作。
## event 是 Godot 分发且尚未被其他控件消费的输入事件。
func _unhandled_input(event: InputEvent) -> void:
    if _is_rotating:
        return
    if not event is InputEventKey:
        return

    var key_event: InputEventKey = event as InputEventKey
    if not key_event.pressed or key_event.echo:
        return
    if key_event.is_action_pressed("ui_left"):
        _rotate_icons(-1)
        get_viewport().set_input_as_handled()
    elif key_event.is_action_pressed("ui_right"):
        _rotate_icons(1)
        get_viewport().set_input_as_handled()


## 从均分椭圆的基础点中抽样：下半圆稀疏、上半圆密集，并从最前方开始排序。
## 返回值是最终用于图标轮转的局部坐标数组。
func _build_sampled_positions() -> Array[Vector2]:
    var positions: Array[Vector2] = []
    var half_point_count: int = ellipse_point_count / 2

    # Godot 的 Y 轴向下，因此 0 到 PI 是下半圆，PI 到 TAU 是上半圆。
    for lower_index: int in range(0, half_point_count, lower_sample_step):
        var lower_angle: float = TAU * float(lower_index) / float(ellipse_point_count)
        positions.append(_calculate_ellipse_position(lower_angle))

    for upper_index: int in range(half_point_count, ellipse_point_count, upper_sample_step):
        var upper_angle: float = TAU * float(upper_index) / float(ellipse_point_count)
        positions.append(_calculate_ellipse_position(upper_angle))

    _move_front_position_to_first(positions)
    return positions


## 把 Y 坐标最大的下半圆点移到数组首位，让首个点稳定代表最前方选中位置。
## positions 是已按椭圆顺时针顺序抽样的点位数组，方法只循环顺序，不改变点位本身。
func _move_front_position_to_first(positions: Array[Vector2]) -> void:
    if positions.is_empty():
        return

    var front_index: int = 0
    for point_index: int in range(1, positions.size()):
        if positions[point_index].y > positions[front_index].y:
            front_index = point_index

    for _move_index: int in range(front_index):
        var first_position: Vector2 = positions.pop_front()
        positions.append(first_position)


## 根据弧度计算椭圆上的局部坐标。
## angle_radians 是 Godot 二维坐标系下用于标准三角函数计算的弧度。
## 返回值是相对于 CarouselViewport 左上角的图标中心点坐标。
func _calculate_ellipse_position(angle_radians: float) -> Vector2:
    return ellipse_center + Vector2(
        cos(angle_radians) * ellipse_radius_x,
        sin(angle_radians) * ellipse_radius_y
    )


## 收集场景中已经预置好的图标节点，不在脚本中动态创建 UI。
## 返回值按场景树顺序保存，初始顺序同时作为首次点位顺序。
func _collect_icon_nodes() -> Array[Node2D]:
    var icons: Array[Node2D] = []
    for child: Node in _icon_sort_root.get_children():
        if child is Node2D:
            icons.append(child as Node2D)
    return icons


## 检查场景图标数量是否与抽样点数量一致，避免遗漏节点时出现越界或空点位。
func _validate_icon_count() -> void:
    if _ordered_icons.size() == _sampled_positions.size():
        return
    push_error(
        "椭圆轮播图标数与抽样点数不一致：图标 %d 个，点位 %d 个。" % [
            _ordered_icons.size(),
            _sampled_positions.size(),
        ]
    )


## 首次进入场景时直接把图标放到对应点位，并同步缩放、透明度与 Y 排序。
func _apply_layout_immediately() -> void:
    var layout_count: int = mini(_ordered_icons.size(), _sampled_positions.size())
    for point_index: int in range(layout_count):
        var icon: Node2D = _ordered_icons[point_index]
        var target_position: Vector2 = _sampled_positions[point_index]
        icon.position = target_position
        icon.scale = _calculate_icon_scale(target_position)
        icon.modulate.a = _calculate_icon_alpha(target_position)
        icon.y_sort_enabled = true


## 点击左按钮时让每个图标补间到左侧相邻对象的旧点位。
func _on_left_button_pressed() -> void:
    _rotate_icons(-1)


## 点击右按钮时让每个图标补间到右侧相邻对象的旧点位。
func _on_right_button_pressed() -> void:
    _rotate_icons(1)


## 循环移动图标顺序，并把每个图标补间到相邻对象旋转前占用的旧点位。
## direction 必须为 -1 或 1，分别表示向左和向右轮转一个点位。
func _rotate_icons(direction: int) -> void:
    if _is_rotating or _ordered_icons.is_empty():
        return

    var normalized_direction: int = clampi(direction, -1, 1)
    if normalized_direction == 0:
        return

    _is_rotating = true
    _set_navigation_enabled(false)
    if _rotation_tween != null and _rotation_tween.is_valid():
        _rotation_tween.kill()

    var old_positions: Array[Vector2] = []
    for icon: Node2D in _ordered_icons:
        old_positions.append(icon.position)

    _rotate_ordered_icons(normalized_direction)
    _rotation_tween = create_tween()
    _rotation_tween.set_parallel(true)

    for icon_index: int in range(_ordered_icons.size()):
        var current_icon: Node2D = _ordered_icons[icon_index]
        var target_position: Vector2 = old_positions[icon_index]
        var target_scale: Vector2 = _calculate_icon_scale(target_position)
        var target_alpha: float = _calculate_icon_alpha(target_position)
        _rotation_tween.tween_property(
            current_icon,
            "position",
            target_position,
            rotate_duration_seconds
        ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN_OUT)
        _rotation_tween.tween_property(
            current_icon,
            "scale",
            target_scale,
            rotate_duration_seconds
        ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN_OUT)
        _rotation_tween.tween_property(
            current_icon,
            "modulate:a",
            target_alpha,
            rotate_duration_seconds
        ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN_OUT)

    _rotation_tween.finished.connect(_on_rotation_finished)


## 根据方向循环移动图标数组，使数组索引继续与目标点位索引保持一致。
## direction 为负数时首个图标移到末尾，为正数时末尾图标移到开头。
func _rotate_ordered_icons(direction: int) -> void:
    if direction < 0:
        var first_icon: Node2D = _ordered_icons.pop_front()
        _ordered_icons.append(first_icon)
        return

    var last_icon: Node2D = _ordered_icons.pop_back()
    _ordered_icons.push_front(last_icon)


## 按点位纵向深度计算缩放；越靠近下半圆前方，图标越大。
## target_position 是图标即将到达的椭圆点位。
## 返回值是可直接写入 Node2D.scale 的等比二维缩放。
func _calculate_icon_scale(target_position: Vector2) -> Vector2:
    var depth_ratio: float = _calculate_depth_ratio(target_position)
    var scale_value: float = lerpf(minimum_icon_scale, maximum_icon_scale, depth_ratio)
    return Vector2(scale_value, scale_value)


## 按点位纵向深度计算透明度；越靠近上半圆后方，图标越透明。
## target_position 是图标即将到达的椭圆点位。
## 返回值范围由 minimum_icon_alpha 与 maximum_icon_alpha 控制。
func _calculate_icon_alpha(target_position: Vector2) -> float:
    var depth_ratio: float = _calculate_depth_ratio(target_position)
    return lerpf(minimum_icon_alpha, maximum_icon_alpha, depth_ratio)


## 把点位 Y 坐标转换为 0 到 1 的椭圆纵向深度比例。
## target_position 是需要计算前后景深的椭圆点位。
## 返回值 0 表示上半圆最远点，1 表示下半圆最近点。
func _calculate_depth_ratio(target_position: Vector2) -> float:
    var top_y: float = ellipse_center.y - ellipse_radius_y
    var diameter_y: float = ellipse_radius_y * 2.0
    if is_zero_approx(diameter_y):
        return 1.0
    return clampf((target_position.y - top_y) / diameter_y, 0.0, 1.0)


## 轮转结束后解锁输入，并刷新当前位于下半圆正中央的图标名称。
func _on_rotation_finished() -> void:
    _is_rotating = false
    _set_navigation_enabled(true)
    _update_selection_label()


## 同步左右按钮可用状态，防止动画过程中重复触发补间。
## is_enabled 表示按钮当前是否允许交互。
func _set_navigation_enabled(is_enabled: bool) -> void:
    _left_button.disabled = not is_enabled
    _right_button.disabled = not is_enabled


## 读取第一个点位上的图标标题，更新 Demo 当前选择提示。
func _update_selection_label() -> void:
    if _ordered_icons.is_empty():
        _selection_label.text = "当前选择：无"
        return

    var selected_icon: Node2D = _ordered_icons[0]
    var display_name_value: Variant = selected_icon.get_meta("display_name", selected_icon.name)
    var display_name: String = str(display_name_value)
    _selection_label.text = "当前选择：%s" % display_name
