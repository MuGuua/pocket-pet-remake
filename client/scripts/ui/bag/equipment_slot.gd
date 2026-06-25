extends Button
class_name EquipmentSlot

const BAG_UI2_ATLAS_PATH: String = "res://asset/分类/ui/bagUI2.png"
const REGION_EMPTY: Rect2 = Rect2(485.0, 504.0, 16.0, 16.0)
const REGION_EQUIPPED: Rect2 = Rect2(506.0, 504.0, 16.0, 16.0)

## 对应服务端 equip_slot，例如 weapon、hat。
@export var equip_slot_key: String = ""
@export var texture_empty: Texture2D
@export var texture_equipped: Texture2D

## 当前槽位已佩戴装备快照。
var _equipment: Dictionary = {}
## 槽位背景。
var _bg_rect: TextureRect = null
## 空槽占位图标，通常由父场景配置武器/防具类型剪影。
var _placeholder_rect: TextureRect = null
## 已佩戴装备的图标。
var _item_icon_rect: TextureRect = null


func _ready() -> void:
	focus_mode = Control.FOCUS_NONE
	mouse_filter = Control.MOUSE_FILTER_STOP
	_resolve_node_refs()
	_ensure_slot_textures()
	_apply_visual_state()


## 写入一件已佩戴装备并刷新展示。
func set_equipment(item: Dictionary) -> void:
	_equipment = item.duplicate(true)
	tooltip_text = str(_equipment.get("item_name", ""))
	if _item_icon_rect != null:
		_item_icon_rect.texture = ItemIcons.resolve_texture(int(_equipment.get("item_id", 0)))
		_item_icon_rect.show()
	_apply_visual_state()


## 清空槽位，恢复 NullEquipmentSlot 样式。
func clear_equipment() -> void:
	_equipment.clear()
	tooltip_text = ""
	if _item_icon_rect != null:
		_item_icon_rect.texture = null
		_item_icon_rect.hide()
	_apply_visual_state()


## 根据 GameState 中对应 equip_slot 刷新当前槽位。
func refresh_from_game_state() -> void:
	if equip_slot_key.is_empty():
		return
	var matched_item: Dictionary = _find_equipped_item_by_slot(equip_slot_key)
	if matched_item.is_empty():
		clear_equipment()
	else:
		set_equipment(matched_item)


## 按空槽/已装备切换背景与占位图标。
func _apply_visual_state() -> void:
	if _bg_rect == null:
		return
	var has_equipment: bool = not _equipment.is_empty()
	_bg_rect.texture = texture_equipped if has_equipment else texture_empty
	if _placeholder_rect != null:
		_placeholder_rect.visible = not has_equipment


func _resolve_node_refs() -> void:
	_bg_rect = get_node_or_null("TextureRect") as TextureRect
	_placeholder_rect = get_node_or_null("TextureRect2") as TextureRect
	_item_icon_rect = get_node_or_null("CenterContainer/Control/ItemIcon") as TextureRect
	if _bg_rect != null:
		_bg_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	if _placeholder_rect != null:
		_placeholder_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	if _item_icon_rect != null:
		_item_icon_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE


func _ensure_slot_textures() -> void:
	if texture_empty == null:
		texture_empty = _make_atlas_texture(BAG_UI2_ATLAS_PATH, REGION_EMPTY)
	if texture_equipped == null:
		texture_equipped = _make_atlas_texture(BAG_UI2_ATLAS_PATH, REGION_EQUIPPED)


func _find_equipped_item_by_slot(slot_key: String) -> Dictionary:
	for item_variant: Variant in GameState.equipped_items:
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		if str(item.get("equip_slot", "")) == slot_key:
			return item.duplicate(true)
	return {}


func _make_atlas_texture(atlas_path: String, region: Rect2) -> Texture2D:
	var atlas_source: Resource = load(atlas_path)
	if atlas_source is not Texture2D:
		return null
	var atlas_texture: AtlasTexture = AtlasTexture.new()
	atlas_texture.atlas = atlas_source as Texture2D
	atlas_texture.region = region
	return atlas_texture
