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
## 抽奖开始阶段每次移动一个点位的补间时长。
@export_range(0.03, 0.4, 0.01) var draw_step_duration_seconds: float = 0.06
## 抽奖结束阶段每次移动一个点位的最长补间时长。
@export_range(0.2, 1.2, 0.01) var draw_deceleration_seconds: float = 0.42
## 进入减速阶段前至少完整转动的圈数，避免结果刚好在当前位置时看不出轮盘转动。
@export_range(2, 10, 1) var draw_full_rounds: int = 5
## 减速阶段占总步数的比例；比例越大，停轮过程越从容。
@export_range(0.2, 0.65, 0.01) var draw_deceleration_ratio: float = 0.36
## 每个稀有形象的中奖概率；两个稀有形象各为 5%。
@export_range(0.01, 0.2, 0.01) var rare_draw_probability: float = 0.05

## 承载全部图标节点的 Y 排序父节点。
@onready var _icon_sort_root: Node2D = $CarouselViewport/IconSortRoot
## 显示当前位于最前方或本次抽奖结果的图标名称。
@onready var _selection_label: Label = $SelectionLabel
## 开始轮盘抽奖的移动端触摸按钮。
@onready var _draw_button: Button = $DrawButton

## 按椭圆前后顺序保存的抽样点位；第一个点固定为下半圆正中央。
var _sampled_positions: Array[Vector2] = []
## 按当前点位顺序保存的图标节点；轮转动画中会循环移动数组顺序。
var _ordered_icons: Array[Node2D] = []
## 当前正在播放的单步轮转补间。
var _rotation_tween: Tween = null
## 用于产生每次抽奖结果的独立随机数生成器。
var _random_generator: RandomNumberGenerator = RandomNumberGenerator.new()
## 当前是否正在抽奖，动画完成前会锁定抽奖按钮和键盘输入。
var _is_drawing: bool = false
## 本次抽奖还需要移动的点位步数。
var _remaining_draw_steps: int = 0
## 本次抽奖的总步数，用于计算逐步减速比例。
var _total_draw_steps: int = 0
## 本次抽奖的轮转方向，-1 表示向左，1 表示向右。
var _draw_direction: int = -1
## 本次抽奖随机选出的目标图标。
var _draw_target_icon: Node2D = null


## 初始化抽样点、收集场景中预置的图标，并绑定抽奖按钮。
func _ready() -> void:
    _random_generator.randomize()
    _sampled_positions = _build_sampled_positions()
    _ordered_icons = _collect_icon_nodes()
    _validate_icon_count()
    _validate_draw_configuration()
    _apply_layout_immediately()
    _draw_button.pressed.connect(_on_draw_button_pressed)
    _update_selection_label()


## 支持键盘空格或确认键开始抽奖，移动端仍使用场景中的抽奖按钮。
## event 是 Godot 分发且尚未被其他控件消费的输入事件。
func _unhandled_input(event: InputEvent) -> void:
    if _is_drawing:
        return
    if not event is InputEventKey:
        return

    var key_event: InputEventKey = event as InputEventKey
    if not key_event.pressed or key_event.echo:
        return
    if key_event.is_action_pressed("ui_accept"):
        _start_draw()
        get_viewport().set_input_as_handled()


## 从均分椭圆的基础点中抽样：下半圆稀疏、上半圆密集，并从最前方开始排序。
## 返回值是最终用于轮盘转动的局部坐标数组。
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
        "椭圆轮盘图标数与点位数不一致：图标 %d 个，点位 %d 个。" % [
            _ordered_icons.size(),
            _sampled_positions.size(),
        ]
    )


## 检查场景中是否配置了两个稀有形象；稀有概率和普通概率由运行时统一计算。
func _validate_draw_configuration() -> void:
    var rare_count: int = _get_rare_icons().size()
    if rare_count == 2:
        return
    push_error("轮盘抽奖需要配置两个稀有形象，当前配置为 %d 个。" % rare_count)


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


## 点击抽奖按钮后开始一次带随机结果、完整转动和减速停轮的轮盘抽奖。
func _on_draw_button_pressed() -> void:
    _start_draw()


## 生成本次抽奖结果，并计算让目标图标停到最前方所需的轮转步数。
func _start_draw() -> void:
    if _is_drawing or _ordered_icons.is_empty():
        return

    _draw_target_icon = _pick_random_draw_result()
    if _draw_target_icon == null:
        return

    var target_index: int = _ordered_icons.find(_draw_target_icon)
    if target_index < 0:
        push_error("抽奖目标不在当前轮盘图标列表中。")
        return

    _draw_direction = -1 if _random_generator.randi_range(0, 1) == 0 else 1
    var offset_steps: int = target_index if _draw_direction < 0 else _ordered_icons.size() - target_index
    _total_draw_steps = draw_full_rounds * _ordered_icons.size() + offset_steps
    _remaining_draw_steps = _total_draw_steps
    _is_drawing = true
    _set_draw_enabled(false)
    _selection_label.text = "抽奖中……"
    _animate_next_draw_step()


## 按每个稀有形象 5%、普通形象平分剩余 90% 的规则抽取目标图标。
## 返回值是本次抽奖最终要停在最前方的图标节点。
func _pick_random_draw_result() -> Node2D:
    var rare_icons: Array[Node2D] = _get_rare_icons()
    var normal_icons: Array[Node2D] = _get_normal_icons()
    if rare_icons.is_empty() and normal_icons.is_empty():
        return null

    var total_rare_probability: float = rare_draw_probability * float(rare_icons.size())
    var normal_probability: float = 1.0 - total_rare_probability
    var roll: float = _random_generator.randf()
    if not rare_icons.is_empty() and roll < total_rare_probability:
        var rare_index: int = mini(
            int(roll / rare_draw_probability),
            rare_icons.size() - 1
        )
        return rare_icons[rare_index]

    if not normal_icons.is_empty():
        var normal_roll: float = clampf(
            (roll - total_rare_probability) / maxf(normal_probability, 0.0001),
            0.0,
            0.999999
        )
        var normal_index: int = mini(
            int(normal_roll * float(normal_icons.size())),
            normal_icons.size() - 1
        )
        return normal_icons[normal_index]

    var fallback_index: int = _random_generator.randi_range(0, rare_icons.size() - 1)
    return rare_icons[fallback_index]


## 从当前轮盘节点中筛选稀有形象；稀有标记配置在场景节点 metadata 中。
## 返回值是所有设置了 is_rare=true 的图标节点。
func _get_rare_icons() -> Array[Node2D]:
    var rare_icons: Array[Node2D] = []
    for icon: Node2D in _ordered_icons:
        var is_rare_value: Variant = icon.get_meta("is_rare", false)
        if bool(is_rare_value):
            rare_icons.append(icon)
    return rare_icons


## 从当前轮盘节点中筛选普通形象。
## 返回值是不带稀有标记的图标节点。
func _get_normal_icons() -> Array[Node2D]:
    var normal_icons: Array[Node2D] = []
    for icon: Node2D in _ordered_icons:
        var is_rare_value: Variant = icon.get_meta("is_rare", false)
        if not bool(is_rare_value):
            normal_icons.append(icon)
    return normal_icons


## 逐步执行轮盘转动；每一步仍复用图标移动到相邻对象旧点位的补间方式。
func _animate_next_draw_step() -> void:
    if _remaining_draw_steps <= 0:
        _finish_draw()
        return

    var old_positions: Array[Vector2] = []
    for icon: Node2D in _ordered_icons:
        old_positions.append(icon.position)

    _rotate_ordered_icons(_draw_direction)
    _remaining_draw_steps -= 1

    var completed_steps: int = _total_draw_steps - _remaining_draw_steps
    var deceleration_start_step: int = maxi(
        1,
        int(float(_total_draw_steps) * (1.0 - draw_deceleration_ratio))
    )
    var deceleration_progress: float = clampf(
        float(completed_steps - deceleration_start_step) / float(
            maxi(_total_draw_steps - deceleration_start_step, 1)
        ),
        0.0,
        1.0
    )
    # 使用 ease-out 曲线，让最后几格的速度变化更连续，避免最后一步突然停住。
    var eased_deceleration: float = 1.0 - pow(1.0 - deceleration_progress, 3.0)
    var step_duration: float = lerpf(
        draw_step_duration_seconds,
        draw_deceleration_seconds,
        eased_deceleration
    )

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
            step_duration
        ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN_OUT)
        _rotation_tween.tween_property(
            current_icon,
            "scale",
            target_scale,
            step_duration
        ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN_OUT)
        _rotation_tween.tween_property(
            current_icon,
            "modulate:a",
            target_alpha,
            step_duration
        ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN_OUT)

    _rotation_tween.finished.connect(_animate_next_draw_step)


## 循环移动图标数组，使数组索引继续与目标点位索引保持一致。
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
    var scale_value: float = lerpf(0.92, 1.22, depth_ratio)
    return Vector2(scale_value, scale_value)


## 按点位纵向深度计算透明度；越靠近上半圆后方，图标越透明。
## target_position 是图标即将到达的椭圆点位。
## 返回值范围由最小和最大透明度固定控制。
func _calculate_icon_alpha(target_position: Vector2) -> float:
    var depth_ratio: float = _calculate_depth_ratio(target_position)
    return lerpf(0.65, 1.0, depth_ratio)


## 把点位 Y 坐标转换为 0 到 1 的椭圆纵向深度比例。
## target_position 是需要计算前后景深的椭圆点位。
## 返回值 0 表示上半圆最远点，1 表示下半圆最近点。
func _calculate_depth_ratio(target_position: Vector2) -> float:
    var top_y: float = ellipse_center.y - ellipse_radius_y
    var diameter_y: float = ellipse_radius_y * 2.0
    if is_zero_approx(diameter_y):
        return 1.0
    return clampf((target_position.y - top_y) / diameter_y, 0.0, 1.0)


## 轮盘动画完成后确认目标图标在最前方，并显示普通或稀有结果。
func _finish_draw() -> void:
    _is_drawing = false
    _set_draw_enabled(true)
    if _ordered_icons.is_empty():
        _selection_label.text = "抽奖结果：无"
        return

    var selected_icon: Node2D = _ordered_icons[0]
    var display_name: String = _get_icon_display_name(selected_icon)
    var is_rare: bool = bool(selected_icon.get_meta("is_rare", false))
    var rarity_text: String = " · 稀有" if is_rare else " · 普通"
    _selection_label.text = "抽奖结果：%s%s" % [display_name, rarity_text]


## 同步抽奖按钮可用状态，防止动画过程中重复触发抽奖。
## is_enabled 表示按钮当前是否允许交互。
func _set_draw_enabled(is_enabled: bool) -> void:
    _draw_button.disabled = not is_enabled


## 读取第一个点位上的图标标题，更新初始选择提示。
func _update_selection_label() -> void:
    if _ordered_icons.is_empty():
        _selection_label.text = "当前选择：无"
        return

    var selected_icon: Node2D = _ordered_icons[0]
    _selection_label.text = "当前选择：%s" % _get_icon_display_name(selected_icon)


## 获取图标在场景 metadata 中配置的展示名称。
## icon 是需要读取名称的轮盘图标节点。
## 返回值是用于 UI 展示的中文名称。
func _get_icon_display_name(icon: Node2D) -> String:
    var display_name_value: Variant = icon.get_meta("display_name", icon.name)
    return str(display_name_value)
