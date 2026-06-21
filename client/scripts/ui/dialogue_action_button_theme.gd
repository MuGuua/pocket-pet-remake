class_name DialogueActionButtonTheme
extends RefCounted

## 剧情/菜单共用操作按钮 normal 样式。
const STYLE_NORMAL: StyleBox = preload("res://resources/ui/dialogue_action_button_normal.tres")
## 剧情/菜单共用操作按钮 hover 样式。
const STYLE_HOVER: StyleBox = preload("res://resources/ui/dialogue_action_button_hover.tres")
## 剧情/菜单共用操作按钮 pressed 样式。
const STYLE_PRESSED: StyleBox = preload("res://resources/ui/dialogue_action_button_pressed.tres")
## 剧情/菜单共用操作按钮 disabled 样式。
const STYLE_DISABLED: StyleBox = preload("res://resources/ui/dialogue_action_button_disabled.tres")
## 按钮统一字号。
const FONT_SIZE: int = 12
## 全宽选项按钮默认高度。
const OPTION_BUTTON_HEIGHT: float = 34.0
## 居中继续按钮默认宽度。
const CONTINUE_BUTTON_WIDTH: float = 112.0


## 将共用按钮主题应用到目标 Button；full_width 为 false 时使用继续按钮的居中窄宽布局。
static func apply(target_button: Button, full_width: bool = true) -> void:
	if STYLE_NORMAL != null:
		target_button.add_theme_stylebox_override("normal", STYLE_NORMAL)
		target_button.add_theme_stylebox_override("focus", STYLE_NORMAL)
	if STYLE_HOVER != null:
		target_button.add_theme_stylebox_override("hover", STYLE_HOVER)
	if STYLE_PRESSED != null:
		target_button.add_theme_stylebox_override("pressed", STYLE_PRESSED)
	if STYLE_DISABLED != null:
		target_button.add_theme_stylebox_override("disabled", STYLE_DISABLED)
	target_button.add_theme_color_override("font_color", Color(1.0, 0.9529412, 0.74509805, 1.0))
	target_button.add_theme_color_override("font_hover_color", Color(1.0, 1.0, 0.8627451, 1.0))
	target_button.add_theme_color_override("font_pressed_color", Color(0.8627451, 0.78431374, 0.5882353, 1.0))
	target_button.add_theme_color_override("font_disabled_color", Color(0.7058824, 0.7058824, 0.7058824, 0.8))
	target_button.add_theme_font_size_override("font_size", FONT_SIZE)
	target_button.focus_mode = Control.FOCUS_NONE
	if full_width:
		target_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		target_button.custom_minimum_size = Vector2(0.0, OPTION_BUTTON_HEIGHT)
	else:
		target_button.size_flags_horizontal = Control.SIZE_SHRINK_CENTER
		target_button.custom_minimum_size = Vector2(CONTINUE_BUTTON_WIDTH, OPTION_BUTTON_HEIGHT)
