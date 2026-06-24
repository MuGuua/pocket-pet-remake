extends Button
class_name BagSlot

signal item_selected(item: Dictionary)

const EMPTY_TEXT: String = ""
const DEFAULT_ICON_TEXTURE_PATH: String = "res://asset/分类/武器/pixel items0.png"
const DEFAULT_ICON_SIZE: int = 32

## 当前格子的服务端物品快照。
var _item: Dictionary = {}
## 图标展示节点。
var _icon_rect: TextureRect = null
## 右下角数量展示节点。
var _count_label: Label = null
## 缺省图标缓存，避免每个格子都重复构建同一个兜底纹理。
var _default_icon_texture: Texture2D = null


## 构建格子内部节点，并绑定点击事件。
func _ready() -> void:
	custom_minimum_size = Vector2(48, 48)
	toggle_mode = true
	focus_mode = Control.FOCUS_NONE
	_build_children()
	pressed.connect(_on_pressed)
	clear_item()


## 填充一个服务端物品快照。
func set_item(item: Dictionary) -> void:
	_item = item.duplicate(true)
	disabled = false
	tooltip_text = BagUiMapper.item_name(_item)
	if _icon_rect != null:
		_icon_rect.texture = _resolve_item_icon_texture(BagUiMapper.icon_ref(_item))
	if _count_label != null:
		var show_count: bool = BagUiMapper.is_stackable(_item)
		_count_label.visible = show_count
		if show_count:
			_count_label.text = UiFormat.value_to_text(BagUiMapper.quantity(_item))


## 清空格子显示。
func clear_item() -> void:
	_item.clear()
	disabled = true
	button_pressed = false
	tooltip_text = ""
	if _icon_rect != null:
		_icon_rect.texture = null
	if _count_label != null:
		_count_label.text = EMPTY_TEXT
		_count_label.hide()


## 设置当前格子的选中态。
func set_selected(selected: bool) -> void:
	button_pressed = selected


## 构建图标和计数标签。
func _build_children() -> void:
	_icon_rect = TextureRect.new()
	_icon_rect.name = "IconRect"
	_icon_rect.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_icon_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	_icon_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	_icon_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(_icon_rect)

	_count_label = Label.new()
	_count_label.name = "CountLabel"
	_count_label.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	_count_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
	_count_label.vertical_alignment = VERTICAL_ALIGNMENT_BOTTOM
	_count_label.add_theme_color_override("font_color", Color(1.0, 0.95, 0.7, 1.0))
	_count_label.add_theme_color_override("font_outline_color", Color(0, 0, 0, 1))
	_count_label.add_theme_constant_override("outline_size", 3)
	_count_label.add_theme_font_size_override("font_size", 11)
	_count_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(_count_label)


## 点击非空格子时抛出当前物品快照。
func _on_pressed() -> void:
	if _item.is_empty():
		return
	item_selected.emit(_item.duplicate(true))


## 直接从服务端 icon 字段加载客户端静态帧贴图，缺失时回退统一兜底图标。
func _resolve_item_icon_texture(icon_ref: String) -> Texture2D:
	var normalized_ref: String = icon_ref.strip_edges()
	if not normalized_ref.is_empty() and normalized_ref.begins_with("res://"):
		var loaded_resource: Resource = load(normalized_ref)
		if loaded_resource is Texture2D:
			return loaded_resource as Texture2D
	return _build_default_item_atlas_texture()


## 使用本地固定图集首格作为兜底图标，避免资源配置漏填时整格空白。
func _build_default_item_atlas_texture() -> Texture2D:
	if _default_icon_texture != null:
		return _default_icon_texture
	var atlas_source: Resource = load(DEFAULT_ICON_TEXTURE_PATH)
	if atlas_source is not Texture2D:
		return null
	var atlas_texture: AtlasTexture = AtlasTexture.new()
	atlas_texture.atlas = atlas_source as Texture2D
	atlas_texture.region = Rect2(0.0, 0.0, float(DEFAULT_ICON_SIZE), float(DEFAULT_ICON_SIZE))
	_default_icon_texture = atlas_texture
	return _default_icon_texture
