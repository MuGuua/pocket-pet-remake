class_name ConfirmPromptPopup
extends "res://scripts/ui/common/modal_popup_layer.gd"

## 通用确认提示面板场景路径。
const SCENE_PATH: String = "res://scenes/ui/common/confirm_prompt_popup.tscn"
## 默认标题文案。
const DEFAULT_TITLE: String = "提示"
## 默认确定按钮文案。
const DEFAULT_CONFIRM_LABEL: String = "确定"
## 默认取消按钮文案。
const DEFAULT_CANCEL_LABEL: String = "取消"

## 玩家点击确定后向外广播。
signal confirmed
## 玩家点击取消或点击遮罩关闭后向外广播。
signal cancelled

## 标题标签。
@onready var _title_label: Label = %TitleLabel
## 支持 BBCode 的正文区域。
@onready var _content_label: RichTextLabel = %ContentLabel
## 左侧确定按钮。
@onready var _confirm_button: RuntimeActionButton = %ConfirmButton
## 右侧取消按钮。
@onready var _cancel_button: RuntimeActionButton = %CancelButton


## 绑定按钮信号；默认隐藏，由调用方按需打开。
func _ready() -> void:
    super._ready()
    if _confirm_button != null and not _confirm_button.pressed.is_connected(_on_confirm_pressed):
        _confirm_button.pressed.connect(_on_confirm_pressed)
    if _cancel_button != null and not _cancel_button.pressed.is_connected(_on_cancel_pressed):
        _cancel_button.pressed.connect(_on_cancel_pressed)


## 展示确认提示面板。
## config 可选键：confirm_label、cancel_label、show_cancel。
func show_prompt(title_text: String, content_bbcode: String, config: Dictionary = {}) -> void:
    var resolved_title: String = title_text.strip_edges()
    if resolved_title.is_empty():
        resolved_title = DEFAULT_TITLE
    if _title_label != null:
        _title_label.text = resolved_title
    if _content_label != null:
        _content_label.clear()
        _content_label.bbcode_enabled = true
        _content_label.text = content_bbcode
    var confirm_label: String = str(config.get("confirm_label", DEFAULT_CONFIRM_LABEL))
    var cancel_label: String = str(config.get("cancel_label", DEFAULT_CANCEL_LABEL))
    var show_cancel: bool = bool(config.get("show_cancel", true))
    if _confirm_button != null:
        _confirm_button.set_button_label(confirm_label)
    if _cancel_button != null:
        _cancel_button.set_button_label(cancel_label)
        _cancel_button.visible = show_cancel
    _apply_interactive_nodes()
    _open_modal()


## 关闭确认提示面板。
func close_prompt() -> void:
    if not visible:
        return
    _notify_host_suppress_input_leak()
    _close_modal()


## 点击遮罩时视为取消，不沿用基类“任意关闭”语义。
func _dismiss_modal() -> void:
    _close_with_result(false)


## 吞掉全局输入，避免点击穿透到底层；仅遮罩点击会触发取消。
func _consume_modal_input(event: InputEvent) -> void:
    if not visible or not _is_topmost_runtime_modal():
        return
    get_viewport().set_input_as_handled()


## 让面板与按钮可交互，同时保留遮罩点击取消能力。
func _apply_interactive_nodes() -> void:
    var panel: Control = get_node_or_null("CenterContainer/PanelContainer") as Control
    if panel != null:
        panel.mouse_filter = Control.MOUSE_FILTER_STOP
    if _confirm_button != null:
        _confirm_button.mouse_filter = Control.MOUSE_FILTER_STOP
    if _cancel_button != null:
        _cancel_button.mouse_filter = Control.MOUSE_FILTER_STOP


## 处理确定按钮点击。
func _on_confirm_pressed() -> void:
    _close_with_result(true)


## 处理取消按钮点击。
func _on_cancel_pressed() -> void:
    _close_with_result(false)


## 根据玩家选择关闭面板并广播对应信号。
func _close_with_result(is_confirmed: bool) -> void:
    if not visible:
        return
    _notify_host_suppress_input_leak()
    _close_modal()
    if is_confirmed:
        confirmed.emit()
    else:
        cancelled.emit()
    popup_closed.emit()
