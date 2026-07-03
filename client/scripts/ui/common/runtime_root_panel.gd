extends CanvasLayer
class_name RuntimeRootPanel

## 运行时根面板基类：全屏遮罩只负责拦截空白点击；面板关闭统一交给显式按钮或业务按钮。
signal menu_closed

## 全屏半透明遮罩，点击面板外空白区域时触发关闭或关闭顶层 overlay。
var _backdrop: ColorRect = null


## 注册根面板分组并创建遮罩层。
func _ready() -> void:
    add_to_group("runtime_root_panel")
    _ensure_backdrop()
    hide()


## 懒创建全屏遮罩，始终置于子节点最底层，避免挡住面板正文。
func _ensure_backdrop() -> void:
    if _backdrop != null:
        return
    _backdrop = ColorRect.new()
    _backdrop.name = "BackdropDim"
    _backdrop.set_anchors_preset(Control.PRESET_FULL_RECT)
    _backdrop.offset_left = 0.0
    _backdrop.offset_top = 0.0
    _backdrop.offset_right = 0.0
    _backdrop.offset_bottom = 0.0
    _backdrop.color = Color(0.0, 0.0, 0.0, 0.28)
    _backdrop.mouse_filter = Control.MOUSE_FILTER_STOP
    _backdrop.visible = false
    _backdrop.gui_input.connect(_on_backdrop_gui_input)
    add_child(_backdrop)
    move_child(_backdrop, 0)


## 打开根面板并显示遮罩。
func open_menu() -> void:
    show()
    if _backdrop != null:
        _backdrop.show()


## 关闭根面板：先清 overlay，再隐藏遮罩并广播 menu_closed。
func close_menu() -> void:
    var was_visible: bool = visible
    _close_all_overlays()
    hide()
    if _backdrop != null:
        _backdrop.hide()
    if was_visible:
        menu_closed.emit()


## 子类覆写：关闭面板内所有 overlay（例如背包详情）。
func _close_all_overlays() -> void:
    pass


## 子类覆写：尝试只关闭最顶层 overlay；成功时返回 true，根面板保持打开。
func _dismiss_top_overlay() -> bool:
    return false


## 点击遮罩时只吞掉输入，不再执行空白区域关闭。
func _on_backdrop_gui_input(event: InputEvent) -> void:
    if not visible:
        return
    if not _is_dismiss_event(event):
        return
    get_viewport().set_input_as_handled()
    if _backdrop != null:
        _backdrop.accept_event()


## 判断是否为“按下类”输入，用于空白区域关闭。
func _is_dismiss_event(event: InputEvent) -> bool:
    if event is InputEventScreenTouch:
        return (event as InputEventScreenTouch).pressed
    if event is InputEventMouseButton:
        return (event as InputEventMouseButton).pressed
    if event is InputEventKey:
        var key_event: InputEventKey = event as InputEventKey
        return key_event.pressed and not key_event.echo
    if event is InputEventJoypadButton:
        return (event as InputEventJoypadButton).pressed
    return false
