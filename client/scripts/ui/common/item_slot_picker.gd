extends AnchoredPopupBase
class_name ItemSlotPicker

## 通用物品格子选择浮层场景路径，可在强化材料、消耗品选择等流程复用。
const SCENE_PATH: String = "res://scenes/ui/common/item_slot_picker.tscn"
## 默认标题文案。
const DEFAULT_TITLE: String = "选择物品"

## 玩家选中某个选项后向外广播完整物品快照。
signal item_selected(item: Dictionary)
## 选择面板关闭时通知父级同步交互状态。
signal picker_closed

## 与背包格子一致的槽位场景。
const SLOT_SCENE: PackedScene = preload("res://scenes/ui/slot.tscn")

## 默认网格列数，可被 open_picker_near 的 config 覆盖。
@export var grid_columns: int = 5
## 超过该行数后固定可视高度并启用纵向滚动。
@export var max_visible_rows: int = 6
## 选中条目后是否自动关闭面板。
@export var close_on_select: bool = true

## 新版场景最外层滚动容器，用于把标题栏和内容区作为一个整体定位。
@onready var _popup_scroll: ScrollContainer = get_node_or_null("ScrollContainer") as ScrollContainer
## 新版场景内容根节点，包含顶部标题栏和下方物品网格面板。
@onready var _content_root: VBoxContainer = get_node_or_null("ScrollContainer/VBoxContainer") as VBoxContainer
## 面板根节点，用于刷新下方物品网格区域布局。
@onready var _panel_root: PanelContainer = get_node_or_null("ScrollContainer/VBoxContainer/PanelRoot") as PanelContainer
## 物品网格滚动容器。
@onready var _grid_scroll: ScrollContainer = get_node_or_null("ScrollContainer/VBoxContainer/PanelRoot/MainVBox/GridScroll") as ScrollContainer
## 物品网格容器。
@onready var _item_grid: GridContainer = get_node_or_null("ScrollContainer/VBoxContainer/PanelRoot/MainVBox/GridScroll/ItemGrid") as GridContainer
## 顶部标题栏标签，新版布局中位于独立 Title/HeaderRow 下。
@onready var _title_label: Label = get_node_or_null("ScrollContainer/VBoxContainer/Title/HeaderRow/TitleLabel") as Label
## 右上角关闭按钮。
@onready var _top_close_button: BaseButton = get_node_or_null("ScrollContainer/VBoxContainer/Title/HeaderRow/TopCloseButton") as BaseButton

## 当前可选物品快照列表。
var _item_options: Array[Dictionary] = []
## 当前选中项的 item_id，用于高亮格子背景。
var _selected_item_id: int = 0
## 本次打开时使用的网格列数。
var _active_grid_columns: int = 5
## 本次打开时允许的最大可视行数。
var _active_max_visible_rows: int = 6
## 本次打开时选中后是否自动关闭。
var _active_close_on_select: bool = true


## 绑定网格参数；默认隐藏，由父级按需打开。
func _ready() -> void:
	super._ready()
	z_index = 18
	_active_grid_columns = grid_columns
	_active_max_visible_rows = max_visible_rows
	_active_close_on_select = close_on_select
	if _item_grid != null:
		_item_grid.columns = _active_grid_columns
	if _title_label != null:
		_title_label.text = DEFAULT_TITLE
	if _top_close_button != null and not _top_close_button.pressed.is_connected(_on_top_close_button_pressed):
		_top_close_button.pressed.connect(_on_top_close_button_pressed)
	if not popup_closed.is_connected(_on_popup_closed):
		popup_closed.connect(_on_popup_closed)


## 在锚点控件附近打开面板，并刷新可选物品列表。
## items 支持背包物品快照、强化材料选项（owned_quantity）或仅含 item_id 的最小字典。
## config 可选键：title、grid_columns、max_visible_rows、close_on_select、placement、anchor_gap、position_offset_x、position_offset_y。
func open_picker_near(anchor: Control, items: Array, selected_item_id: int = 0, config: Dictionary = {}) -> void:
	if anchor == null:
		return
	_item_options.clear()
	for item_variant: Variant in items:
		if item_variant is not Dictionary:
			continue
		var normalized_item: Dictionary = normalize_item_option(item_variant as Dictionary)
		if normalized_item.is_empty():
			continue
		_item_options.append(normalized_item)
	_selected_item_id = selected_item_id
	_rebuild_item_grid()
	_prepare_open(config)
	_start_anchored_open(anchor)
	call_deferred("_refresh_panel_layout_and_position")


## 关闭物品选择面板。
func hide_picker() -> void:
	close_popup()


## 响应右上角关闭按钮。
func _on_top_close_button_pressed() -> void:
	hide_picker()


## 判断全局坐标是否落在选择面板整体区域内（含标题与边框）。
func is_global_point_over_panel(global_point: Vector2) -> bool:
	if not is_open():
		return false
	var layout_root: Control = _get_layout_root()
	if layout_root == null:
		return false
	return layout_root.get_global_rect().has_point(global_point)


## 判断全局坐标是否落在物品格子可视区域内。
func is_global_point_over_item_cells(global_point: Vector2) -> bool:
	if _item_grid == null or not is_open():
		return false
	if _grid_scroll != null:
		if not _grid_scroll.get_global_rect().has_point(global_point):
			return false
	return _item_grid.get_global_rect().has_point(global_point)


## 将外部传入的选项字典规范化为 BagSlot 可消费的物品快照。
static func normalize_item_option(option: Dictionary) -> Dictionary:
	var item_id: int = BagUiMapper.item_id(option)
	if item_id <= 0:
		return {}
	var snapshot: Dictionary = option.duplicate(true)
	if not snapshot.has("quantity"):
		if snapshot.has("owned_quantity"):
			snapshot["quantity"] = int(snapshot.get("owned_quantity", 0))
		else:
			snapshot["quantity"] = BagUiMapper.quantity(option)
	if str(snapshot.get("item_name", "")).is_empty():
		var fallback_name: String = str(snapshot.get("name", ""))
		if not fallback_name.is_empty():
			snapshot["item_name"] = fallback_name
	if not snapshot.has("is_stackable"):
		snapshot["is_stackable"] = BagUiMapper.is_stackable(snapshot)
	return snapshot


## 返回用于定位的面板根节点。
func _get_layout_root() -> Control:
	if _content_root != null:
		return _content_root
	if _popup_scroll != null:
		return _popup_scroll
	return self


## 解析物品选择面板专属 config，并交给基类处理锚点参数。
func _prepare_open(config: Dictionary) -> void:
	super._prepare_open(config)
	_apply_picker_config(config)


## 解析标题、网格列数、最大可视行数等运行时配置。
func _apply_picker_config(config: Dictionary) -> void:
	var title_text: String = str(config.get("title", DEFAULT_TITLE))
	if title_text.is_empty():
		title_text = DEFAULT_TITLE
	if _title_label != null:
		_title_label.text = title_text
	_active_grid_columns = int(config.get("grid_columns", grid_columns))
	if _active_grid_columns <= 0:
		_active_grid_columns = grid_columns
	_active_max_visible_rows = int(config.get("max_visible_rows", max_visible_rows))
	if _active_max_visible_rows <= 0:
		_active_max_visible_rows = max_visible_rows
	_active_close_on_select = bool(config.get("close_on_select", close_on_select))
	if _item_grid != null:
		_item_grid.columns = _active_grid_columns


## 清空并重建物品格子。
func _rebuild_item_grid() -> void:
	if _item_grid == null:
		return
	for child: Node in _item_grid.get_children():
		child.queue_free()
	for option_index: int in range(_item_options.size()):
		var item: Dictionary = _item_options[option_index]
		var item_id: int = BagUiMapper.item_id(item)
		if item_id <= 0:
			continue
		var slot: BagSlot = _build_item_cell_slot(option_index)
		_item_grid.add_child(slot)
		slot.set_item(item)
		slot.set_selected(item_id == _selected_item_id)


## 等待布局完成后刷新面板尺寸，并在需要时启用滚动。
func _refresh_panel_layout_and_position() -> void:
	if not is_inside_tree():
		return
	_refresh_panel_layout()
	await get_tree().process_frame
	if _anchor_control != null:
		_apply_position_near(_anchor_control)


## 按物品数量动态计算面板宽高；超过 max_visible_rows 时锁定高度并显示滚动条。
func _refresh_panel_layout() -> void:
	if _panel_root == null or _grid_scroll == null or _item_grid == null:
		return
	var item_count: int = _item_grid.get_child_count()
	var row_count: int = _resolve_item_row_count(item_count)
	var visible_rows: int = mini(row_count, _active_max_visible_rows)
	var grid_width: float = _grid_width_for_columns(_active_grid_columns)
	var grid_viewport_height: float = _grid_height_for_rows(visible_rows)
	# GridScroll 位于容器布局中，只声明最小尺寸即可。父级容器会在下一布局帧逐级刷新，
	# 直接写 size 或对全矩形锚定的 ScrollContainer 调用 reset_size() 会与 anchors 的拉伸结果冲突。
	_grid_scroll.custom_minimum_size = Vector2(grid_width, grid_viewport_height)
	if row_count > _active_max_visible_rows:
		_grid_scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_AUTO
	else:
		_grid_scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
		_grid_scroll.scroll_vertical = 0


## 计算物品条目对应的网格行数。
func _resolve_item_row_count(item_count: int) -> int:
	if item_count <= 0:
		return 1
	return int(ceil(float(item_count) / float(_active_grid_columns)))


## 计算指定列数下的网格宽度。
func _grid_width_for_columns(column_count: int) -> float:
	if column_count <= 0:
		return 0.0
	var slot_size: Vector2 = _resolve_slot_scene_size()
	var separation: int = _item_grid.get_theme_constant("h_separation") if _item_grid != null else 0
	return float(column_count) * slot_size.x + float(column_count - 1) * float(separation)


## 计算指定行数下的网格高度。
func _grid_height_for_rows(row_count: int) -> float:
	if row_count <= 0:
		return _resolve_slot_scene_size().y
	var slot_size: Vector2 = _resolve_slot_scene_size()
	var separation: int = _item_grid.get_theme_constant("v_separation") if _item_grid != null else 0
	return float(row_count) * slot_size.y + float(row_count - 1) * float(separation)


## 读取 slot.tscn 自身配置的最小尺寸，避免脚本内硬编码放大后的格子边长。
func _resolve_slot_scene_size() -> Vector2:
	if _item_grid != null:
		for child: Node in _item_grid.get_children():
			if child is Control:
				var child_control: Control = child as Control
				var child_size: Vector2 = child_control.get_combined_minimum_size()
				if child_size.x > 0.0 and child_size.y > 0.0:
					return child_size
	var preview_slot: Control = SLOT_SCENE.instantiate() as Control
	if preview_slot == null:
		return Vector2.ZERO
	var scene_size: Vector2 = preview_slot.get_combined_minimum_size()
	preview_slot.free()
	return scene_size


## 构建单个物品格子，复用背包 BagSlot 样式。
func _build_item_cell_slot(option_index: int) -> BagSlot:
	var slot: BagSlot = SLOT_SCENE.instantiate() as BagSlot
	if not slot.item_selected.is_connected(_on_item_cell_pressed.bind(option_index)):
		slot.item_selected.connect(_on_item_cell_pressed.bind(option_index))
	return slot


## 转发物品格子点击并在需要时关闭面板。
func _on_item_cell_pressed(_item: Dictionary, option_index: int) -> void:
	if option_index < 0 or option_index >= _item_options.size():
		return
	var selected_item: Dictionary = _item_options[option_index].duplicate(true)
	if BagUiMapper.item_id(selected_item) <= 0:
		return
	if _active_close_on_select:
		hide_picker()
	item_selected.emit(selected_item)


## 同步 picker_closed 信号，兼容旧监听方。
func _on_popup_closed() -> void:
	picker_closed.emit()
