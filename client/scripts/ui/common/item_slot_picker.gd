extends Control
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
## 单个格子边长，与 slot.tscn 保持一致。
const SLOT_SIZE: int = 22
## 格子间距，与背包 GridContainer 保持一致。
const GRID_SEPARATION: int = 2
## 标题区固定高度（像素）。
const TITLE_HEIGHT: int = 14
## 标题与网格之间的间距（像素）。
const MAIN_VBOX_SEPARATION: int = 3
## 面板 ContentMargin 四边留白（像素）。
const CONTENT_MARGIN: int = 4
## 面板相对锚点的默认间距（像素）。
const ANCHOR_GAP: float = 6.0
## 面板出现在锚点上方（水平居中）。
const PLACEMENT_ABOVE: String = "above"
## 面板出现在锚点右侧（顶部对齐，向下扩展时不遮挡按钮）。
const PLACEMENT_RIGHT: String = "right"

## 默认网格列数，可被 open_near 的 config 覆盖。
@export var grid_columns: int = 5
## 超过该行数后固定可视高度并启用纵向滚动。
@export var max_visible_rows: int = 6
## 选中条目后是否自动关闭面板。
@export var close_on_select: bool = true

## 面板根节点，用于读取尺寸并定位。
@onready var _panel_root: PanelContainer = $PanelRoot
## 物品网格滚动容器。
@onready var _grid_scroll: ScrollContainer = %GridScroll
## 物品网格容器。
@onready var _item_grid: GridContainer = %ItemGrid
## 面板标题标签。
@onready var _title_label: Label = %TitleLabel

## 当前可选物品快照列表。
var _item_options: Array[Dictionary] = []
## 当前选中项的 item_id，用于高亮格子背景。
var _selected_item_id: int = 0
## 面板是否处于打开状态。
var _picker_open: bool = false
## 打开面板时的锚点控件，布局刷新后用于重新定位。
var _anchor_control: Control = null
## 本次打开时使用的网格列数。
var _active_grid_columns: int = 5
## 本次打开时允许的最大可视行数。
var _active_max_visible_rows: int = 6
## 本次打开时选中后是否自动关闭。
var _active_close_on_select: bool = true
## 本次打开时的面板相对锚点方位。
var _active_placement: String = PLACEMENT_ABOVE
## 本次打开时面板与锚点之间的间距（像素）。
var _active_anchor_gap: float = ANCHOR_GAP
## 本次打开时相对默认定位的额外 X 偏移（像素）。
var _active_position_offset_x: float = 0.0
## 本次打开时相对默认定位的额外 Y 偏移（像素）。
var _active_position_offset_y: float = 0.0


## 绑定网格参数；默认隐藏，由父级按需打开。
func _ready() -> void:
    z_index = 18
    mouse_filter = Control.MOUSE_FILTER_IGNORE
    _active_grid_columns = grid_columns
    _active_max_visible_rows = max_visible_rows
    _active_close_on_select = close_on_select
    if _item_grid != null:
        _item_grid.columns = _active_grid_columns
        _item_grid.add_theme_constant_override("h_separation", GRID_SEPARATION)
        _item_grid.add_theme_constant_override("v_separation", GRID_SEPARATION)
    if _title_label != null:
        _title_label.text = DEFAULT_TITLE
    hide()


## 选择面板当前是否展开。
func is_open() -> bool:
    return _picker_open


## 在锚点控件附近打开面板，并刷新可选物品列表。
## items 支持背包物品快照、强化材料选项（owned_quantity）或仅含 item_id 的最小字典。
## config 可选键：title、grid_columns、max_visible_rows、close_on_select、placement、anchor_gap、position_offset_x、position_offset_y。
func open_near(anchor: Control, items: Array, selected_item_id: int = 0, config: Dictionary = {}) -> void:
    if anchor == null:
        return
    _apply_open_config(config)
    _item_options.clear()
    for item_variant: Variant in items:
        if item_variant is not Dictionary:
            continue
        var normalized_item: Dictionary = normalize_item_option(item_variant as Dictionary)
        if normalized_item.is_empty():
            continue
        _item_options.append(normalized_item)
    _selected_item_id = selected_item_id
    _anchor_control = anchor
    _rebuild_item_grid()
    _picker_open = true
    top_level = true
    show()
    if get_parent() != null:
        get_parent().move_child(self, get_parent().get_child_count() - 1)
    call_deferred("_refresh_panel_layout_and_position")


## 关闭物品选择面板。
func hide_picker() -> void:
    if not _picker_open:
        return
    _picker_open = false
    _anchor_control = null
    top_level = false
    hide()
    picker_closed.emit()


## 判断全局坐标是否落在选择面板整体区域内（含标题与边框）。
func is_global_point_over_panel(global_point: Vector2) -> bool:
    if not _picker_open or _panel_root == null:
        return false
    return _panel_root.get_global_rect().has_point(global_point)


## 判断全局坐标是否落在物品格子可视区域内。
func is_global_point_over_item_cells(global_point: Vector2) -> bool:
    if _item_grid == null or not _picker_open:
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


## 解析 open_near 传入的运行时配置。
func _apply_open_config(config: Dictionary) -> void:
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
    _active_placement = str(config.get("placement", PLACEMENT_ABOVE))
    if _active_placement != PLACEMENT_RIGHT:
        _active_placement = PLACEMENT_ABOVE
    _active_anchor_gap = float(config.get("anchor_gap", ANCHOR_GAP))
    if _active_anchor_gap < 0.0:
        _active_anchor_gap = ANCHOR_GAP
    _active_position_offset_x = float(config.get("position_offset_x", 0.0))
    _active_position_offset_y = float(config.get("position_offset_y", 0.0))
    if _item_grid != null:
        _item_grid.columns = _active_grid_columns


## 根据锚点按钮位置计算并应用面板坐标。
func _apply_position_near(anchor: Control) -> void:
    if anchor == null or not is_inside_tree():
        return
    var panel_size: Vector2 = _panel_root.size
    if panel_size.x <= 0.0 or panel_size.y <= 0.0:
        panel_size = _panel_root.get_combined_minimum_size()
    if panel_size.x <= 0.0 or panel_size.y <= 0.0:
        panel_size = size
    var anchor_rect: Rect2 = anchor.get_global_rect()
    var viewport_size: Vector2 = get_viewport().get_visible_rect().size
    var global_x: float = 0.0
    var global_y: float = 0.0
    if _active_placement == PLACEMENT_RIGHT:
        global_x = anchor_rect.position.x + anchor_rect.size.x + _active_anchor_gap
        global_y = anchor_rect.position.y
        if global_x + panel_size.x > viewport_size.x:
            global_x = anchor_rect.position.x - panel_size.x - _active_anchor_gap
    else:
        global_x = anchor_rect.position.x + (anchor_rect.size.x - panel_size.x) * 0.5
        global_y = anchor_rect.position.y - panel_size.y - _active_anchor_gap
    global_x += _active_position_offset_x
    global_y += _active_position_offset_y
    global_x = clampf(global_x, 0.0, maxf(viewport_size.x - panel_size.x, 0.0))
    global_y = clampf(global_y, 0.0, maxf(viewport_size.y - panel_size.y, 0.0))
    size = panel_size
    global_position = Vector2(global_x, global_y)


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
    _grid_scroll.custom_minimum_size = Vector2(grid_width, grid_viewport_height)
    _grid_scroll.size = Vector2(grid_width, grid_viewport_height)
    if row_count > _active_max_visible_rows:
        _grid_scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_AUTO
    else:
        _grid_scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
        _grid_scroll.scroll_vertical = 0
    var panel_width: float = float(CONTENT_MARGIN * 2) + grid_width
    var panel_height: float = float(CONTENT_MARGIN * 2) + float(TITLE_HEIGHT) + float(MAIN_VBOX_SEPARATION) + grid_viewport_height
    _panel_root.custom_minimum_size = Vector2(panel_width, panel_height)
    _panel_root.size = Vector2(panel_width, panel_height)


## 计算物品条目对应的网格行数。
func _resolve_item_row_count(item_count: int) -> int:
    if item_count <= 0:
        return 1
    return int(ceil(float(item_count) / float(_active_grid_columns)))


## 计算指定列数下的网格宽度。
func _grid_width_for_columns(column_count: int) -> float:
    if column_count <= 0:
        return 0.0
    return float(column_count * SLOT_SIZE + (column_count - 1) * GRID_SEPARATION)


## 计算指定行数下的网格高度。
func _grid_height_for_rows(row_count: int) -> float:
    if row_count <= 0:
        return float(SLOT_SIZE)
    return float(row_count * SLOT_SIZE + (row_count - 1) * GRID_SEPARATION)


## 构建单个物品格子，复用背包 BagSlot 样式。
func _build_item_cell_slot(option_index: int) -> BagSlot:
    var slot: BagSlot = SLOT_SCENE.instantiate() as BagSlot
    if not slot.pressed.is_connected(_on_item_cell_pressed.bind(option_index)):
        slot.pressed.connect(_on_item_cell_pressed.bind(option_index))
    return slot


## 转发物品格子点击并在需要时关闭面板。
func _on_item_cell_pressed(option_index: int) -> void:
    if option_index < 0 or option_index >= _item_options.size():
        return
    var item: Dictionary = _item_options[option_index].duplicate(true)
    if BagUiMapper.item_id(item) <= 0:
        return
    if _active_close_on_select:
        hide_picker()
    item_selected.emit(item)
