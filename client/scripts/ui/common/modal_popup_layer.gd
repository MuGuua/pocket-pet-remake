extends CanvasLayer

## 运行时模态弹窗基类：加入 runtime_modal_popup 组，新战斗开始时会由主场景统一 force_close_popup。
signal popup_closed()

## 全屏遮罩，负责拦截点击并触发关闭。
var _dim_layer: ColorRect = null
## 是否已安排延迟解锁，避免关闭弹窗的同一次输入穿透到底层。
var _deferred_unlock_pending: bool = false
## 弹窗打开时的 process 帧号；打开当帧及下一帧忽略空白关闭，避免战斗结算同帧输入把弹窗立刻关掉。
var _modal_open_frame: int = -1
## 已通过空白/全局输入完成关闭的那次物理事件；同一次点击只关闭最顶层一个弹窗。
static var _blank_dismiss_consumed_event: InputEvent = null
## 是否已安排在本帧末尾清空白关闭去重标记。
static var _blank_dismiss_reset_scheduled: bool = false


func _ready() -> void:
	add_to_group("runtime_modal_popup")
	_dim_layer = get_node_or_null("DimLayer") as ColorRect
	if _dim_layer != null:
		_dim_layer.mouse_filter = Control.MOUSE_FILTER_STOP
		_dim_layer.gui_input.connect(_on_dim_layer_gui_input)
	_set_content_mouse_ignore(true)
	hide()
	_disable_modal_input_listeners()


## 打开模态弹窗并锁定世界交互。
func _open_modal() -> void:
	_modal_open_frame = Engine.get_process_frames()
	show()
	_enable_modal_input_listeners()
	_set_runtime_input_locked(true)


## 关闭模态弹窗；世界交互延迟到下一帧再恢复，避免关闭输入穿透。
func _close_modal() -> void:
	hide()
	_disable_modal_input_listeners()
	if _deferred_unlock_pending:
		return
	_deferred_unlock_pending = true
	call_deferred("_finish_modal_close")


func _finish_modal_close() -> void:
	_deferred_unlock_pending = false
	if visible:
		return
	_set_runtime_input_locked(false)


func _enable_modal_input_listeners() -> void:
	set_process_shortcut_input(true)
	set_process_input(true)
	set_process_unhandled_input(true)


func _disable_modal_input_listeners() -> void:
	set_process_shortcut_input(false)
	set_process_input(false)
	set_process_unhandled_input(false)


## 在输入链路最前端吞掉事件；只有“按下类”输入会尝试空白关闭。
func _consume_modal_input(event: InputEvent) -> void:
	if not visible or not _is_topmost_runtime_modal():
		return
	get_viewport().set_input_as_handled()
	_try_blank_dismiss(event)


func _shortcut_input(event: InputEvent) -> void:
	_consume_modal_input(event)


func _input(event: InputEvent) -> void:
	_consume_modal_input(event)


func _unhandled_input(event: InputEvent) -> void:
	_consume_modal_input(event)


func _on_dim_layer_gui_input(event: InputEvent) -> void:
	if not visible or not _is_topmost_runtime_modal():
		return
	get_viewport().set_input_as_handled()
	if _dim_layer != null:
		_dim_layer.accept_event()
	_try_blank_dismiss(event)


## 空白区域关闭：仅最顶层弹窗响应，且同一次物理输入只关闭一层。
func _try_blank_dismiss(event: InputEvent) -> void:
	if not _is_dismiss_event(event):
		return
	if not _can_dismiss_modal_now():
		return
	if not _is_topmost_runtime_modal():
		return
	if _blank_dismiss_consumed_event == event:
		return
	_blank_dismiss_consumed_event = event
	_schedule_blank_dismiss_reset()
	_dismiss_modal()


## 帧末清空白关闭去重，避免影响下一次点击。
func _schedule_blank_dismiss_reset() -> void:
	if _blank_dismiss_reset_scheduled:
		return
	_blank_dismiss_reset_scheduled = true
	call_deferred("_reset_blank_dismiss_consumed_event")


## 重置空白关闭去重状态，供下一帧新的物理点击使用。
static func _reset_blank_dismiss_consumed_event() -> void:
	_blank_dismiss_consumed_event = null
	_blank_dismiss_reset_scheduled = false


## 多个模态弹窗同时可见时，仅最顶层（layer 最高）响应空白关闭。
func _is_topmost_runtime_modal() -> bool:
	var top_modal: Node = null
	var top_order: int = -2147483648
	for node_variant: Variant in get_tree().get_nodes_in_group("runtime_modal_popup"):
		if not (node_variant is CanvasLayer):
			continue
		var modal_layer: CanvasLayer = node_variant as CanvasLayer
		if not modal_layer.visible:
			continue
		var modal_order: int = modal_layer.layer * 1000 + modal_layer.get_index()
		if modal_order >= top_order:
			top_order = modal_order
			top_modal = modal_layer
	return top_modal == self


func _dismiss_modal() -> void:
	get_viewport().set_input_as_handled()
	_notify_host_suppress_input_leak()
	_close_modal()
	popup_closed.emit()


## 外部强制关闭（例如新战斗开始）；会立即解锁输入并发出 popup_closed。
func force_close_popup() -> void:
	if not visible:
		return
	get_viewport().set_input_as_handled()
	_notify_host_suppress_input_leak()
	_force_close_modal()
	popup_closed.emit()


func _force_close_modal() -> void:
	hide()
	_disable_modal_input_listeners()
	_deferred_unlock_pending = false
	_set_runtime_input_locked(false)


## 让面板区域点击也能穿透到遮罩，实现“点屏幕任意区域关闭”。
func _set_content_mouse_ignore(ignore: bool) -> void:
	var center: Control = get_node_or_null("CenterContainer") as Control
	if center == null:
		return
	_apply_mouse_filter_recursive(center, ignore)


func _apply_mouse_filter_recursive(node: Node, ignore: bool) -> void:
	if node is Control:
		var control: Control = node as Control
		control.mouse_filter = Control.MOUSE_FILTER_IGNORE if ignore else Control.MOUSE_FILTER_STOP
	for child in node.get_children():
		_apply_mouse_filter_recursive(child, ignore)


func _is_dismiss_event(event: InputEvent) -> bool:
	if event is InputEventScreenTouch:
		return event.pressed
	if event is InputEventMouseButton:
		return event.pressed
	if event is InputEventKey:
		return event.pressed and not event.echo
	if event is InputEventJoypadButton:
		return event.pressed
	return false


## 弹窗刚打开时忽略空白关闭，避免战斗卸载/结算同帧残留点击把升级弹窗闪关。
func _can_dismiss_modal_now() -> bool:
	if _modal_open_frame < 0:
		return true
	return Engine.get_process_frames() > _modal_open_frame + 1


func _set_runtime_input_locked(locked: bool) -> void:
	var host: Node = get_parent()
	if host != null and host.has_method("_set_runtime_menu_locked"):
		host.call("_set_runtime_menu_locked", locked)


func _notify_host_suppress_input_leak() -> void:
	var host: Node = get_parent()
	if host != null and host.has_method("_suppress_settlement_input_leak"):
		host.call("_suppress_settlement_input_leak")
