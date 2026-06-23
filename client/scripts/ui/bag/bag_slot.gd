extends Button
class_name BagSlot

signal item_selected(item: Dictionary)

const EMPTY_TEXT: String = ""

## 图标资源索引，仅用于本地表现映射。
var icon_registry: ItemIconRegistry = null
## 当前格子的服务端物品快照。
var _item: Dictionary = {}
## 图标展示节点。
var _icon_rect: TextureRect = null
## 右下角数量展示节点。
var _count_label: Label = null


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
    if _icon_rect != null and icon_registry != null:
        _icon_rect.texture = icon_registry.resolve_icon(BagUiMapper.icon_key(_item))
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
