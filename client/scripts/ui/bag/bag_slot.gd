extends Button
class_name BagSlot

signal item_selected(item: Dictionary)

const EMPTY_TEXT: String = ""
const BAG_UI_ATLAS_PATH: String = "res://asset/分类/ui/bagUI.png"
const BAG_UI2_ATLAS_PATH: String = "res://asset/分类/ui/bagUI2.png"
const REGION_EMPTY: Rect2 = Rect2(485.0, 528.0, 16.0, 16.0)
const REGION_FILLED: Rect2 = Rect2(485.0, 528.0, 16.0, 16.0)
const REGION_SELECTED: Rect2 = Rect2(506.0, 528.0, 16.0, 16.0)
const BAG_ITEM_HOVER_NAME_SCENE: PackedScene = preload("res://scenes/ui/bag/bag_item_hover_name.tscn")

@export var texture_empty: Texture2D
@export var texture_filled: Texture2D
@export var texture_selected: Texture2D

## 当前格子的服务端物品快照。
var _item: Dictionary = {}
## 槽位背景节点。
var _bg_rect: TextureRect = null
## 图标展示节点。
var _icon_rect: TextureRect = null
## 右下角数量展示节点。
var _count_label: Label = null
## 右下角强化等级角标（仅装备且 enhance_level > 0 时显示）。
var _enhance_label: Label = null
## 悬停时在格子右上方展示物品名称的浮层。
var _hover_name: BagItemHoverName = null


## 绑定场景节点，并初始化三种槽位样式。
func _ready() -> void:
	custom_minimum_size = Vector2(22, 22)
	toggle_mode = true
	focus_mode = Control.FOCUS_NONE
	mouse_filter = Control.MOUSE_FILTER_STOP
	_resolve_node_refs()
	_ensure_slot_textures()
	if not mouse_entered.is_connected(_on_mouse_entered):
		mouse_entered.connect(_on_mouse_entered)
	if not mouse_exited.is_connected(_on_mouse_exited):
		mouse_exited.connect(_on_mouse_exited)
	toggled.connect(_on_toggled)
	_ensure_hover_name()
	clear_item()


## 填充一个服务端物品快照。
func set_item(item: Dictionary) -> void:
	_item = item.duplicate(true)
	disabled = false
	tooltip_text = ""
	if _icon_rect != null:
		_icon_rect.texture = BagUiMapper.icon_texture(_item)
		_icon_rect.show()
	if _count_label != null:
		var show_count: bool = BagUiMapper.is_stackable(_item)
		_count_label.visible = show_count
		if show_count:
			_count_label.text = UiFormat.value_to_text(BagUiMapper.quantity(_item))
		else:
			_count_label.text = EMPTY_TEXT
	BagUiMapper.apply_enhance_level_badge(_enhance_label, _item)
	_apply_visual_state()


## 清空格子显示。
func clear_item() -> void:
	_item.clear()
	disabled = true
	button_pressed = false
	tooltip_text = ""
	_hide_hover_name()
	if _icon_rect != null:
		_icon_rect.texture = null
		_icon_rect.hide()
	if _count_label != null:
		_count_label.text = EMPTY_TEXT
		_count_label.hide()
	BagUiMapper.apply_enhance_level_badge(_enhance_label, {})
	_apply_visual_state()


## 设置当前格子的选中态。
func set_selected(selected: bool) -> void:
	if _item.is_empty():
		button_pressed = false
	else:
		button_pressed = selected
	_apply_visual_state()


## 根据空槽、有物品、选中三种逻辑态刷新背景。
func _apply_visual_state() -> void:
	if _bg_rect == null:
		return
	if _item.is_empty():
		_bg_rect.texture = texture_empty
		return
	if button_pressed:
		_bg_rect.texture = texture_selected
	else:
		_bg_rect.texture = texture_filled


## 缓存场景内已有节点，避免与 slot.tscn 子节点重复创建。
func _resolve_node_refs() -> void:
	_bg_rect = get_node_or_null("TextureRect") as TextureRect
	_icon_rect = get_node_or_null("CenterContainer/Control/ItemImage") as TextureRect
	_count_label = get_node_or_null("CenterContainer/Control/ItemQuantity") as Label
	_enhance_label = get_node_or_null("CenterContainer/Control/ItemEnhanceLevel") as Label
	if _bg_rect != null:
		_bg_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	if _icon_rect != null:
		_icon_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	if _count_label != null:
		_count_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	if _enhance_label != null:
		_enhance_label.mouse_filter = Control.MOUSE_FILTER_IGNORE


## 懒创建悬停名称浮层，供鼠标移入时在格子右上方展示。
func _ensure_hover_name() -> void:
	if _hover_name != null:
		return
	_hover_name = BAG_ITEM_HOVER_NAME_SCENE.instantiate() as BagItemHoverName
	if _hover_name == null:
		return
	add_child(_hover_name)


## 鼠标移入且格子有物品时，在右上方显示名称。
func _on_mouse_entered() -> void:
	if _item.is_empty() or _hover_name == null:
		return
	_hover_name.show_for_anchor(self, BagUiMapper.item_name(_item))


## 鼠标移出格子时隐藏悬停名称。
func _on_mouse_exited() -> void:
	_hide_hover_name()


## 关闭悬停名称浮层。
func _hide_hover_name() -> void:
	if _hover_name == null:
		return
	_hover_name.hide_name()


## 若编辑器未指定贴图，则按背包图集默认区域生成。
func _ensure_slot_textures() -> void:
	if texture_empty == null:
		texture_empty = _make_atlas_texture(BAG_UI_ATLAS_PATH, REGION_EMPTY)
	if texture_filled == null:
		texture_filled = _make_atlas_texture(BAG_UI_ATLAS_PATH, REGION_FILLED)
	if texture_selected == null:
		texture_selected = _make_atlas_texture(BAG_UI2_ATLAS_PATH, REGION_SELECTED)


## 切换选中态时只刷新背景，并在选中时抛出物品快照。
func _on_toggled(toggled_on: bool) -> void:
	if _item.is_empty():
		button_pressed = false
		_apply_visual_state()
		return
	_apply_visual_state()
	if toggled_on:
		item_selected.emit(_item.duplicate(true))


## 按图集路径与区域构建 AtlasTexture。
func _make_atlas_texture(atlas_path: String, region: Rect2) -> Texture2D:
	var atlas_source: Resource = load(atlas_path)
	if atlas_source is not Texture2D:
		return null
	var atlas_texture: AtlasTexture = AtlasTexture.new()
	atlas_texture.atlas = atlas_source as Texture2D
	atlas_texture.region = region
	return atlas_texture
