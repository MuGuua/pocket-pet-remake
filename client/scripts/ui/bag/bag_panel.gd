extends CanvasLayer
class_name BagPanel

signal menu_closed

const ITEM_ICON_REGISTRY: ItemIconRegistry = preload("res://resources/ui/item_icon_registry.tres")
const SLOT_COLUMNS: int = 5
const DEFAULT_SLOT_COUNT: int = 30

## 背包主容器。
var _root_panel: PanelContainer = null
## 容量展示标签。
var _capacity_label: Label = null
## 格子网格。
var _slot_grid: GridContainer = null
## 物品详情面板。
var _detail_panel: BagItemDetail = null
## 提示信息标签。
var _message_label: Label = null
## 通用服务端请求 loading 遮罩。
var _request_loading: RequestLoadingOverlay = null
## 当前等待的背包请求序列号。
var _loading_request_seq: int = 0
## 当前创建出的格子列表。
var _slots: Array[BagSlot] = []
## 当前选中格子编号。
var _selected_slot_index: int = 0


## 构建背包 UI，订阅服务端快照变化。
func _ready() -> void:
    hide()
    _build_ui()
    _build_loading_overlay()
    if not GameState.bag_changed.is_connected(_on_bag_changed):
        GameState.bag_changed.connect(_on_bag_changed)
    if not App.request_finished.is_connected(_on_request_finished):
        App.request_finished.connect(_on_request_finished)
    _refresh_from_game_state()


## 断开全局信号，避免面板销毁后继续收到回调。
func _exit_tree() -> void:
    if GameState.bag_changed.is_connected(_on_bag_changed):
        GameState.bag_changed.disconnect(_on_bag_changed)
    if App.request_finished.is_connected(_on_request_finished):
        App.request_finished.disconnect(_on_request_finished)


## 打开背包面板，并向服务端请求最新背包快照。
func open_menu() -> void:
    show()
    _set_message("正在同步服务端背包数据。")
    _request_bag_snapshot()


## 关闭背包面板。
func close_menu() -> void:
    var was_visible: bool = visible
    _loading_request_seq = 0
    _hide_loading_overlay()
    hide()
    if was_visible:
        menu_closed.emit()


## 构建移动端竖屏背包布局。
func _build_ui() -> void:
    _root_panel = PanelContainer.new()
    _root_panel.name = "RootPanel"
    _root_panel.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
    _root_panel.custom_minimum_size = Vector2(330, 560)
    _root_panel.position = Vector2(-165, -280)
    add_child(_root_panel)

    var margin: MarginContainer = MarginContainer.new()
    margin.add_theme_constant_override("margin_left", 10)
    margin.add_theme_constant_override("margin_top", 10)
    margin.add_theme_constant_override("margin_right", 10)
    margin.add_theme_constant_override("margin_bottom", 10)
    _root_panel.add_child(margin)

    var root: VBoxContainer = VBoxContainer.new()
    root.add_theme_constant_override("separation", 8)
    margin.add_child(root)

    var header: HBoxContainer = HBoxContainer.new()
    root.add_child(header)

    var title_label: Label = Label.new()
    title_label.text = "物品行囊"
    title_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
    title_label.add_theme_font_size_override("font_size", 18)
    header.add_child(title_label)

    _capacity_label = Label.new()
    _capacity_label.text = "0/0"
    _capacity_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
    header.add_child(_capacity_label)

    var close_button: Button = Button.new()
    close_button.text = "关闭"
    close_button.pressed.connect(close_menu)
    header.add_child(close_button)

    var scroll: ScrollContainer = ScrollContainer.new()
    scroll.custom_minimum_size = Vector2(0, 310)
    scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
    root.add_child(scroll)

    _slot_grid = GridContainer.new()
    _slot_grid.columns = SLOT_COLUMNS
    _slot_grid.add_theme_constant_override("h_separation", 6)
    _slot_grid.add_theme_constant_override("v_separation", 6)
    scroll.add_child(_slot_grid)

    _detail_panel = BagItemDetail.new()
    _detail_panel.action_requested.connect(_on_detail_action_requested)
    root.add_child(_detail_panel)

    _message_label = Label.new()
    _message_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
    _message_label.custom_minimum_size = Vector2(0, 34)
    root.add_child(_message_label)


## 构建服务端请求通用 loading 遮罩。
func _build_loading_overlay() -> void:
    _request_loading = RequestLoadingOverlay.new()
    _request_loading.name = "BagRequestLoadingOverlay"
    add_child(_request_loading)

## 请求服务端最新背包快照。
func _request_bag_snapshot() -> void:
    if not GameState.is_ws_authenticated:
        _set_message("实时连接未就绪，无法同步背包。")
        _refresh_from_game_state()
        return
    var request_seq: int = App.request_bag_list()
    if request_seq <= 0:
        _set_message("背包请求发送失败。")
        _refresh_from_game_state()
        return
    _loading_request_seq = request_seq
    _show_loading_overlay()


## 根据 GameState 中的服务端权威快照重绘背包。
func _refresh_from_game_state() -> void:
    if _slot_grid == null:
        return

    var items: Array = _collect_bag_items()
    var capacity: int = _resolve_capacity(items.size())
    _capacity_label.text = "%d/%d" % [items.size(), capacity]
    _ensure_slot_count(capacity)

    var items_by_slot: Dictionary = {}
    for item_variant: Variant in items:
        if item_variant is not Dictionary:
            continue
        var item: Dictionary = item_variant as Dictionary
        var slot_index: int = BagUiMapper.slot_index(item)
        if slot_index <= 0:
            slot_index = items_by_slot.size() + 1
        items_by_slot[slot_index] = item

    for index: int in range(_slots.size()):
        var slot: BagSlot = _slots[index]
        var slot_index: int = index + 1
        if items_by_slot.has(slot_index):
            slot.set_item(items_by_slot[slot_index] as Dictionary)
        else:
            slot.clear_item()
        slot.set_selected(slot_index == _selected_slot_index)

    if _selected_slot_index > 0 and items_by_slot.has(_selected_slot_index):
        _detail_panel.set_item(items_by_slot[_selected_slot_index] as Dictionary)
    else:
        _selected_slot_index = 0
        _detail_panel.clear_item()


## 收集当前随身背包物品快照。
func _collect_bag_items() -> Array:
    var container_items: Variant = GameState.bag_container.get("items", [])
    if container_items is Array:
        return (container_items as Array).duplicate(true)
    return GameState.bag_items.duplicate(true)


## 解析服务端容量字段，缺失时使用默认容量兜底。
func _resolve_capacity(item_count: int) -> int:
    var capacity: int = int(GameState.bag_container.get("capacity", GameState.bag_container.get("max_slots", 0)))
    if capacity <= 0:
        capacity = max(DEFAULT_SLOT_COUNT, item_count)
    return capacity


## 确保格子数量与容量一致。
func _ensure_slot_count(capacity: int) -> void:
    while _slots.size() < capacity:
        var slot: BagSlot = BagSlot.new()
        slot.icon_registry = ITEM_ICON_REGISTRY
        slot.item_selected.connect(_on_slot_item_selected)
        _slot_grid.add_child(slot)
        _slots.append(slot)
    while _slots.size() > capacity:
        var slot: BagSlot = _slots.pop_back()
        _slot_grid.remove_child(slot)
        slot.queue_free()


## 点击格子后展示详情。
func _on_slot_item_selected(item: Dictionary) -> void:
    _selected_slot_index = BagUiMapper.slot_index(item)
    if _selected_slot_index <= 0:
        _selected_slot_index = 1
    _detail_panel.set_item(item)
    for index: int in range(_slots.size()):
        _slots[index].set_selected(index + 1 == _selected_slot_index)
    _set_message("已选择：%s" % BagUiMapper.item_name(item))


## 执行详情面板上的物品动作。
func _on_detail_action_requested(action_key: String, item: Dictionary) -> void:
    if action_key != "use":
        _set_message("该功能后端尚未实现，当前仅保留 UI 入口。")
        return
    _request_use_item(item)


## 发起使用物品请求，等待服务端回包与推送刷新。
func _request_use_item(item: Dictionary) -> void:
    if _loading_request_seq != 0:
        return
    if not GameState.is_ws_authenticated:
        _set_message("实时连接未就绪，无法使用物品。")
        return
    var slot_index: int = BagUiMapper.slot_index(item)
    if slot_index <= 0:
        _set_message("物品格子数据无效，无法使用。")
        return
    var request_seq: int = App.request_use_item("bag", slot_index, 1)
    if request_seq <= 0:
        _set_message("使用物品请求发送失败。")
        return
    _loading_request_seq = request_seq
    _set_message("正在请求服务端使用：%s" % BagUiMapper.item_name(item))
    _show_loading_overlay()


## 背包快照变化时刷新当前 UI。
func _on_bag_changed() -> void:
    if visible:
        _refresh_from_game_state()


## 请求完成后关闭 loading；真实数量变化仍以 GameState 推送为准。
func _on_request_finished(_request_cmd: int, seq: int, ok: bool, _response_cmd: int, payload: Dictionary) -> void:
    if _loading_request_seq == 0 or seq != _loading_request_seq:
        return
    _loading_request_seq = 0
    _hide_loading_overlay()
    if ok:
        _set_message("服务端请求已完成。")
        _refresh_from_game_state()
    else:
        _set_message("请求失败：%s" % str(payload.get("message", payload.get("msg", "服务端拒绝请求"))))


## 显示通用请求 loading。
func _show_loading_overlay() -> void:
    if _request_loading != null:
        _request_loading.show_waiting("正在同步服务端背包数据")


## 隐藏通用请求 loading。
func _hide_loading_overlay() -> void:
    if _request_loading != null:
        _request_loading.hide_overlay()

## 设置底部提示文案。
func _set_message(message: String) -> void:
    if _message_label != null:
        _message_label.text = message
