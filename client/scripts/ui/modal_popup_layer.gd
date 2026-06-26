extends CanvasLayer

## 运行时模态弹窗基类：加入 runtime_modal_popup 组，新战斗开始时会由主场景统一 force_close_popup。
signal popup_closed()

## 全屏遮罩，负责拦截点击并触发关闭。
var _dim_layer: ColorRect = null
## 是否已安排延迟解锁，避免关闭弹窗的同一次输入穿透到底层。
var _deferred_unlock_pending: bool = false


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


## 在输入链路最前端吞掉事件；只有“按下类”输入会关闭弹窗。
func _consume_modal_input(event: InputEvent) -> void:
	if not visible or not _is_topmost_runtime_modal():
		return
	get_viewport().set_input_as_handled()
	if _is_dismiss_event(event):
		_dismiss_modal()


func _shortcut_input(event: InputEvent) -> void:
	_consume_modal_input(event)


func _input(event: InputEvent) -> void:
	_consume_modal_input(event)


func _unhandled_input(event: InputEvent) -> void:
	_consume_modal_input(event)


func _on_dim_layer_gui_input(event: InputEvent) -> void:
	if not visible or not _is_topmost_runtime_modal():
		return
	if _dim_layer != null:
		_dim_layer.accept_event()
	if _is_dismiss_event(event):
		_dismiss_modal()


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
	_notify_host_suppress_input_leak()
	_close_modal()
	popup_closed.emit()


## 外部强制关闭（例如新战斗开始）；会立即解锁输入并发出 popup_closed。
func force_close_popup() -> void:
	if not visible:
		return
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


func _set_runtime_input_locked(locked: bool) -> void:
	var host: Node = get_parent()
	if host != null and host.has_method("_set_runtime_menu_locked"):
		host.call("_set_runtime_menu_locked", locked)


func _notify_host_suppress_input_leak() -> void:
	var host: Node = get_parent()
	if host != null and host.has_method("_suppress_settlement_input_leak"):
		host.call("_suppress_settlement_input_leak")
