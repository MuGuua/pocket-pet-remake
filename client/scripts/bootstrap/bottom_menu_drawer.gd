extends Node
class_name BottomMenuDrawer

## 单个菜单按钮滑入或滑出抽屉的动画时长。
const BUTTON_MOVE_DURATION_SEC: float = 0.22
## 相邻按钮开始动画的间隔，用于形成从右到左收起、从左到右展开的顺序感。
const BUTTON_STAGGER_DURATION_SEC: float = 0.04
## 按钮收进抽屉时的轻微缩放比例，配合位移让收纳反馈更清晰。
const COLLAPSED_BUTTON_SCALE: Vector2 = Vector2(0.86, 0.86)
## 十字按钮每次切换抽屉状态时旋转的角度。
const TOGGLE_ROTATION_RADIANS: float = PI * 0.5

## 抽屉最左侧的地图按钮。
@onready var map_button: Button = %MapButton
## 仅暗雷地图可见的挂机按钮。
@onready var auto_encounter_button: Button = %AutoEncounterButton
## 任务入口按钮。
@onready var task_button: Button = %TaskButton
## 设置入口按钮。
@onready var settings_button: Button = %SettingsButton
## 抽屉最右侧的背包入口按钮。
@onready var bag_button: Button = %BagButton
## 固定在屏幕右下角、负责切换抽屉状态的十字按钮。
@onready var toggle_button: Button = %BottomMenuToggleButton

## 每个业务按钮完全展开时的场景坐标；坐标只从场景节点读取，不在代码中重复维护布局值。
var _expanded_positions: Dictionary[Button, Vector2] = {}
## 每个业务按钮当前是否应该出现在展开后的抽屉中；挂机按钮会跟随地图能力动态变化。
var _button_availability: Dictionary[Button, bool] = {}
## 当前抽屉是否处于展开状态。
var _drawer_expanded: bool = true
## 当前是否正在播放抽屉动画，避免连续点击产生互相覆盖的 Tween。
var _drawer_animating: bool = false
## 当前抽屉 Tween；切换场景释放节点前可安全终止。
var _drawer_tween: Tween = null
## 场景坐标和可用性是否已经完成初始化。
var _layout_initialized: bool = false


## 绑定十字按钮与挂机按钮可见性，并在首帧布局完成后记录场景配置坐标。
func _ready() -> void:
    if toggle_button != null and not toggle_button.pressed.is_connected(_on_toggle_button_pressed):
        toggle_button.pressed.connect(_on_toggle_button_pressed)
    call_deferred("_initialize_drawer_layout")


## 节点退出场景树时终止未完成动画，避免 Tween 回调访问已释放的 HUD。
func _exit_tree() -> void:
    if _drawer_tween != null and _drawer_tween.is_valid():
        _drawer_tween.kill()


## 记录所有按钮在场景中配置的展开位置，并设置旋转与缩放中心。
func _initialize_drawer_layout() -> void:
    if _layout_initialized or toggle_button == null:
        return
    for button: Button in _menu_buttons_left_to_right():
        if button == null:
            continue
        _expanded_positions[button] = button.position
        _button_availability[button] = button.visible
        button.pivot_offset = button.size * 0.5
    toggle_button.pivot_offset = toggle_button.size * 0.5
    _layout_initialized = true
    _apply_expanded_layout()


## 点击十字按钮时在展开和收起状态之间切换。
func _on_toggle_button_pressed() -> void:
    if not _layout_initialized or _drawer_animating:
        return
    if _drawer_expanded:
        _collapse_drawer()
    else:
        _expand_drawer()


## 从最右侧业务按钮开始依次收进十字按钮，并让十字按钮顺时针旋转 90 度。
func _collapse_drawer() -> void:
    _drawer_expanded = false
    _drawer_animating = true
    _close_settings_popup()
    _stop_active_tween()
    var collapsed_position: Vector2 = _get_collapsed_button_position()
    var available_buttons: Array[Button] = _available_buttons_right_to_left()
    _set_buttons_interactive(false)
    _drawer_tween = create_tween()
    _drawer_tween.set_parallel(true)
    _drawer_tween.tween_property(
        toggle_button,
        "rotation",
        TOGGLE_ROTATION_RADIANS,
        BUTTON_MOVE_DURATION_SEC
    ).set_trans(Tween.TRANS_BACK).set_ease(Tween.EASE_OUT)
    for button_index: int in range(available_buttons.size()):
        var button: Button = available_buttons[button_index]
        var start_delay: float = float(button_index) * BUTTON_STAGGER_DURATION_SEC
        _drawer_tween.tween_property(
            button,
            "position",
            collapsed_position,
            BUTTON_MOVE_DURATION_SEC
        ).set_delay(start_delay).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN)
        _drawer_tween.tween_property(
            button,
            "modulate:a",
            0.0,
            BUTTON_MOVE_DURATION_SEC * 0.8
        ).set_delay(start_delay).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN)
        _drawer_tween.tween_property(
            button,
            "scale",
            COLLAPSED_BUTTON_SCALE,
            BUTTON_MOVE_DURATION_SEC
        ).set_delay(start_delay).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN)
    _drawer_tween.chain().tween_callback(_finish_collapse)


## 从最左侧业务按钮开始依次滑出，并让十字按钮逆时针恢复原方向。
func _expand_drawer() -> void:
    _drawer_expanded = true
    _drawer_animating = true
    _stop_active_tween()
    var collapsed_position: Vector2 = _get_collapsed_button_position()
    var available_buttons: Array[Button] = _available_buttons_left_to_right()
    var expanded_targets: Dictionary[Button, Vector2] = _expanded_targets_for_buttons(available_buttons)
    for button: Button in available_buttons:
        _set_button_visible(button, true)
        button.position = collapsed_position
        button.modulate.a = 0.0
        button.scale = COLLAPSED_BUTTON_SCALE
    _set_buttons_interactive(false)
    _drawer_tween = create_tween()
    _drawer_tween.set_parallel(true)
    _drawer_tween.tween_property(
        toggle_button,
        "rotation",
        0.0,
        BUTTON_MOVE_DURATION_SEC
    ).set_trans(Tween.TRANS_BACK).set_ease(Tween.EASE_OUT)
    for button_index: int in range(available_buttons.size()):
        var button: Button = available_buttons[button_index]
        var start_delay: float = float(button_index) * BUTTON_STAGGER_DURATION_SEC
        var expanded_position: Vector2 = expanded_targets.get(button, button.position)
        _drawer_tween.tween_property(
            button,
            "position",
            expanded_position,
            BUTTON_MOVE_DURATION_SEC
        ).set_delay(start_delay).set_trans(Tween.TRANS_BACK).set_ease(Tween.EASE_OUT)
        _drawer_tween.tween_property(
            button,
            "modulate:a",
            1.0,
            BUTTON_MOVE_DURATION_SEC * 0.8
        ).set_delay(start_delay).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_OUT)
        _drawer_tween.tween_property(
            button,
            "scale",
            Vector2.ONE,
            BUTTON_MOVE_DURATION_SEC
        ).set_delay(start_delay).set_trans(Tween.TRANS_BACK).set_ease(Tween.EASE_OUT)
    _drawer_tween.chain().tween_callback(_finish_expand)


## 收起动画结束后隐藏业务按钮，避免透明按钮继续拦截移动端触摸事件。
func _finish_collapse() -> void:
    for button: Button in _menu_buttons_left_to_right():
        if button != null:
            _set_button_visible(button, false)
    _drawer_animating = false
    if toggle_button != null:
        toggle_button.mouse_filter = Control.MOUSE_FILTER_STOP


## 展开动画结束后恢复可用按钮的触摸交互，并重新应用业务可见性。
func _finish_expand() -> void:
    for button: Button in _menu_buttons_left_to_right():
        if button == null:
            continue
        var available: bool = bool(_button_availability.get(button, false))
        _set_button_visible(button, available)
        button.mouse_filter = Control.MOUSE_FILTER_STOP if available else Control.MOUSE_FILTER_IGNORE
    _drawer_animating = false
    if toggle_button != null:
        toggle_button.mouse_filter = Control.MOUSE_FILTER_STOP


## 更新挂机按钮的业务可用性；抽屉收起时只记录状态，下一次展开再按该状态显示。
## available 表示当前地图是否允许玩家使用挂机入口。
func set_auto_encounter_available(available: bool) -> void:
    if auto_encounter_button == null:
        return
    _button_availability[auto_encounter_button] = available
    if not _layout_initialized:
        auto_encounter_button.visible = available
        return
    _set_button_visible(auto_encounter_button, available and _drawer_expanded)
    if _drawer_expanded and not _drawer_animating:
        _apply_expanded_layout()
    auto_encounter_button.mouse_filter = (
        Control.MOUSE_FILTER_STOP
        if available and _drawer_expanded and not _drawer_animating
        else Control.MOUSE_FILTER_IGNORE
    )


## 返回场景定义的业务按钮顺序，供展开动画从左到右播放。
func _menu_buttons_left_to_right() -> Array[Button]:
    return [map_button, auto_encounter_button, task_button, settings_button, bag_button]


## 返回当前可用按钮的左到右顺序。
func _available_buttons_left_to_right() -> Array[Button]:
    var available_buttons: Array[Button] = []
    for button: Button in _menu_buttons_left_to_right():
        if button != null and bool(_button_availability.get(button, button.visible)):
            available_buttons.append(button)
    return available_buttons


## 返回当前可用按钮的右到左顺序，供收起动画从右到左播放。
func _available_buttons_right_to_left() -> Array[Button]:
    var available_buttons: Array[Button] = _available_buttons_left_to_right()
    available_buttons.reverse()
    return available_buttons


## 按当前可用按钮数量紧凑计算展开位置；隐藏挂机时不会在地图与任务之间留下空槽。
## available_buttons 是当前需要展示且已经按左到右排列的按钮。
func _expanded_targets_for_buttons(available_buttons: Array[Button]) -> Dictionary[Button, Vector2]:
    var targets: Dictionary[Button, Vector2] = {}
    var all_buttons: Array[Button] = _menu_buttons_left_to_right()
    var first_slot_index: int = maxi(all_buttons.size() - available_buttons.size(), 0)
    for button_index: int in range(available_buttons.size()):
        var button: Button = available_buttons[button_index]
        var slot_button: Button = all_buttons[first_slot_index + button_index]
        targets[button] = _expanded_positions.get(slot_button, button.position)
    return targets


## 立即应用当前业务可用性对应的紧凑展开布局，不参与抽屉动画。
func _apply_expanded_layout() -> void:
    if not _layout_initialized or not _drawer_expanded:
        return
    var available_buttons: Array[Button] = _available_buttons_left_to_right()
    var expanded_targets: Dictionary[Button, Vector2] = _expanded_targets_for_buttons(available_buttons)
    for button: Button in available_buttons:
        button.position = expanded_targets.get(button, button.position)
        button.scale = Vector2.ONE
        button.modulate.a = 1.0


## 计算所有业务按钮收进十字按钮时共用的左上角坐标。
func _get_collapsed_button_position() -> Vector2:
    if toggle_button == null:
        return Vector2.ZERO
    var reference_size: Vector2 = bag_button.size if bag_button != null else Vector2.ZERO
    return toggle_button.position + (toggle_button.size - reference_size) * 0.5


## 统一设置业务按钮是否接收触摸；动画期间全部忽略，结束后再按状态恢复。
func _set_buttons_interactive(interactive: bool) -> void:
    for button: Button in _menu_buttons_left_to_right():
        if button != null:
            button.mouse_filter = Control.MOUSE_FILTER_STOP if interactive else Control.MOUSE_FILTER_IGNORE
    if toggle_button != null:
        toggle_button.mouse_filter = Control.MOUSE_FILTER_IGNORE


## 修改按钮的表现可见性；业务可用状态单独保存在抽屉控制器中。
## button 是需要显示或隐藏的业务按钮。
## should_show 表示按钮当前是否应该参与渲染。
func _set_button_visible(button: Button, should_show: bool) -> void:
    if button == null or button.visible == should_show:
        return
    button.visible = should_show


## 收起抽屉前关闭可能锚定在设置按钮上方的动作菜单，避免按钮消失后弹层悬空。
func _close_settings_popup() -> void:
    var hud_root: Node = get_parent()
    if hud_root != null and hud_root.has_method("_hide_settings_menu"):
        hud_root.call("_hide_settings_menu")


## 终止上一次动画，保证新动画只由当前抽屉状态控制。
func _stop_active_tween() -> void:
    if _drawer_tween != null and _drawer_tween.is_valid():
        _drawer_tween.kill()
    _drawer_tween = null
