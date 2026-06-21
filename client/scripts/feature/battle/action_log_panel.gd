extends PanelContainer
class_name ActionLogPanel

const MAX_VISIBLE_LINES: int = 4
const LINE_DURATION_SEC: float = 3.0
const LINE_HEIGHT: float = 12.0

@onready var _lines_root: VBoxContainer = %LogLinesRoot

@export var log_font_size: int = 16
@export var line_height: float = 18.0  # 建议略大于字号

var _line_timers: Dictionary = {}
var _line_labels_by_id: Dictionary = {}

## 清空当前战斗动作日志，并释放仍在显示的日志行。
func clear_logs() -> void:
	for child: Node in _lines_root.get_children():
		if child is Label:
			_dismiss_line(child as Label)
	_line_timers.clear()
	_line_labels_by_id.clear()

## 追加一条会自动消失的战斗动作日志。
func append_log(text: String) -> void:
	if text.is_empty():
		return
	while _lines_root.get_child_count() >= MAX_VISIBLE_LINES:
		var oldest: Label = _lines_root.get_child(0) as Label
		_dismiss_line(oldest)
	var label: Label = _create_line_label(text)
	_lines_root.add_child(label)
	var label_id: int = label.get_instance_id()
	var timer: SceneTreeTimer = get_tree().create_timer(LINE_DURATION_SEC)
	_line_timers[label] = timer
	_line_labels_by_id[label_id] = label
	timer.timeout.connect(_on_line_timeout.bind(label_id))

## 追加包含行动单位名称的战斗动作日志。
func append_action_log(actor_name: String, text: String) -> void:
	var line_text: String = text
	if not actor_name.is_empty():
		line_text = "%s：%s" % [actor_name, text]
	append_log(line_text)

## 日志行定时到期后按实例 ID 安全回收，避免已释放 Label 被绑定参数强转时报错。
func _on_line_timeout(label_id: int) -> void:
	if not _line_labels_by_id.has(label_id):
		return
	var label_variant: Variant = _line_labels_by_id.get(label_id)
	if not label_variant is Label:
		_line_labels_by_id.erase(label_id)
		return
	var label: Label = label_variant as Label
	_dismiss_line(label)

## 移除并释放指定日志行，同时清理它对应的定时器索引。
func _dismiss_line(label: Label) -> void:
	if label == null or not is_instance_valid(label):
		return
	var label_id: int = label.get_instance_id()
	if _line_labels_by_id.has(label_id):
		_line_labels_by_id.erase(label_id)
	if _line_timers.has(label):
		_line_timers.erase(label)
	if label.get_parent() == _lines_root:
		_lines_root.remove_child(label)
	label.queue_free()

## 创建一条移动端战斗日志 Label。
func _create_line_label(text: String) -> Label:
	var label: Label = Label.new()
	label.text = text
	label.autowrap_mode = TextServer.AUTOWRAP_OFF
	label.text_overrun_behavior = TextServer.OVERRUN_TRIM_ELLIPSIS
	label.clip_text = false
	label.add_theme_font_size_override("font_size", log_font_size)
	label.custom_minimum_size = Vector2(0.0, LINE_HEIGHT)
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	return label
