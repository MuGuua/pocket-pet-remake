extends RuntimeRootPanel
class_name WarehousePanel

## 仓库面板：上半部分为仓库格子，下半部分沿用背包的格子、筛选与翻页布局。
## 所有存取都走服务端权威协议（5041 存入 / 5051 取出），面板只负责展示与发起请求；
## 存取成功后服务端会推送两个容器的最新快照（5011），面板随 GameState.bag_changed 刷新。

## 仓库区每页格子数，对应场景中 WarehouseGrid 的 21 个 Slot（3 行 × 7 列）。
const WAREHOUSE_SLOTS_PER_PAGE: int = 21
## 背包区每页格子数，与背包面板保持一致（4 行 × 7 列）。
const BAG_SLOTS_PER_PAGE: int = 28
## 背包区筛选键，与背包面板含义一致：全部 / 装备 / 其他。
const FILTER_ALL: String = "all"
const FILTER_EQUIPMENT: String = "equipment"
const FILTER_OTHER: String = "other"
## 格子动作菜单键：存入仓库 / 取出到背包。
const ACTION_DEPOSIT: String = "deposit"
const ACTION_WITHDRAW: String = "withdraw"
## 格子旁弹出的动作菜单场景，复用背包「更多」所用的通用组件。
const ACTION_MENU_SCENE: PackedScene = preload(ActionMenuPopup.SCENE_PATH)
## 通用全屏 loading；所有存取请求等待期间都必须展示它。
const LOADING_SCENE: PackedScene = preload(GenericLoadingScene.SCENE_PATH)

## 标题栏关闭按钮。
@onready var _close_button: Button = $RootPanel/MarginContainer/VBoxContainer/Title/HBoxContainer/Button
## 仓库容量文案，例如「仓库 12/60」。
@onready var _warehouse_capacity_label: Label = %WarehouseCapacityLabel
## 仓库格子网格。
@onready var _warehouse_grid: GridContainer = %WarehouseGrid
## 仓库上一页 / 下一页按钮与页码文案。
@onready var _warehouse_prev_button: Button = %WarehousePrevButton
@onready var _warehouse_next_button: Button = %WarehouseNextButton
@onready var _warehouse_page_label: Label = %WarehousePageLabel
## 背包容量文案，节点路径与背包面板一致。
@onready var _bag_capacity_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/PanelContainer/HBoxContainer/Label
## 金银铜展示，与背包面板一致。
@onready var _gold_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/GoldCoinAmount
@onready var _silver_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/SilverCoinAmount
@onready var _copper_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/CopperCoinAmount
## 背包格子网格与底部筛选、翻页控件，节点路径与背包面板一致。
@onready var _bag_grid: GridContainer = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer/GridContainer
@onready var _filter_all_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/Button
@onready var _filter_equipment_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/Button2
@onready var _filter_other_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/Button3
@onready var _bag_prev_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/Button4
@onready var _bag_next_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/Button5
@onready var _bag_page_input: LineEdit = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/PanelContainer/PageInput

## 仓库 NPC 实体 ID；存取请求必须带回服务端做距离与菜单校验。
var _warehouse_entity_id: int = 0
## 仓库区与背包区格子实例缓存。
var _warehouse_slots: Array[BagSlot] = []
var _bag_slots: Array[BagSlot] = []
## 两个区域当前页码（从 0 开始）。
var _warehouse_page: int = 0
var _bag_page: int = 0
## 背包区当前筛选。
var _bag_filter: String = FILTER_ALL
## 当前点中的物品与其所在容器（"warehouse" / "bag"），用于动作菜单回调。
var _selected_item: Dictionary = {}
var _selected_container: String = ""
## 存取请求在途标记，防止连点重复提交。
var _request_in_flight: bool = false
## 打开流程代次；关闭或重复打开时使旧的等待结果失效。
var _open_generation: int = 0
## 懒创建的动作菜单与 loading 实例。
var _action_menu: ActionMenuPopup = null
var _loading: GenericLoadingScene = null


## 绑定按钮与数据刷新信号；面板默认隐藏，由主场景在数据就绪后打开。
func _ready() -> void:
	super._ready()
	hide()
	if _close_button != null:
		_close_button.pressed.connect(close_menu)
	if _warehouse_prev_button != null:
		_warehouse_prev_button.pressed.connect(_on_warehouse_page_step.bind(-1))
	if _warehouse_next_button != null:
		_warehouse_next_button.pressed.connect(_on_warehouse_page_step.bind(1))
	if _bag_prev_button != null:
		_bag_prev_button.pressed.connect(_on_bag_page_step.bind(-1))
	if _bag_next_button != null:
		_bag_next_button.pressed.connect(_on_bag_page_step.bind(1))
	if _bag_page_input != null:
		# 背包页码在仓库面板中只做展示，翻页统一走左右按钮。
		_bag_page_input.editable = false
	if _filter_all_button != null:
		_filter_all_button.pressed.connect(_on_filter_pressed.bind(FILTER_ALL))
	if _filter_equipment_button != null:
		_filter_equipment_button.pressed.connect(_on_filter_pressed.bind(FILTER_EQUIPMENT))
	if _filter_other_button != null:
		_filter_other_button.pressed.connect(_on_filter_pressed.bind(FILTER_OTHER))
	_warehouse_slots = _collect_slots(_warehouse_grid, "warehouse")
	_bag_slots = _collect_slots(_bag_grid, "bag")
	if not GameState.bag_changed.is_connected(_on_data_changed):
		GameState.bag_changed.connect(_on_data_changed)
	if not GameState.wallet_changed.is_connected(_on_data_changed):
		GameState.wallet_changed.connect(_on_data_changed)


## 离开场景树时注销全局信号，避免已释放面板继续接收刷新。
func _exit_tree() -> void:
	if GameState.bag_changed.is_connected(_on_data_changed):
		GameState.bag_changed.disconnect(_on_data_changed)
	if GameState.wallet_changed.is_connected(_on_data_changed):
		GameState.wallet_changed.disconnect(_on_data_changed)


## 打开前拉取仓库与背包的完整权威快照；面板保持隐藏，由主场景负责 loading。
## entity_id 为玩家交互的仓库 NPC 实体 ID。返回 true 表示两个快照都已就绪。
func prepare_open_data(entity_id: int) -> bool:
	_open_generation += 1
	var generation: int = _open_generation
	_warehouse_entity_id = entity_id
	_warehouse_page = 0
	_bag_page = 0
	_bag_filter = FILTER_ALL
	_clear_selection()
	if entity_id <= 0 or not GameState.is_ws_authenticated:
		return false
	var warehouse_seq: int = App.request_container_list("warehouse")
	var bag_seq: int = App.request_container_list("bag")
	if warehouse_seq <= 0 or bag_seq <= 0:
		return false
	var succeeded: bool = await _wait_requests([warehouse_seq, bag_seq], CommandIds.CONTAINER_LIST_REQ)
	return succeeded and generation == _open_generation


## 数据就绪后打开面板，直接展示刚拉取的新快照。
func open_menu() -> void:
	super.open_menu()
	_refresh_all()


## 关闭面板：收起动作菜单与 loading，使在途的打开流程失效。
func close_menu() -> void:
	_open_generation += 1
	_hide_loading()
	super.close_menu()


## 关闭面板内的浮层（动作菜单），供根面板基类在关闭或点空白时调用。
func _close_all_overlays() -> void:
	if _action_menu != null:
		_action_menu.hide_menu()
	_clear_selection()


## 收集网格中的 BagSlot，并把点击事件连同所属容器一起转发。
## container_type 为 "warehouse" 或 "bag"。
func _collect_slots(grid: GridContainer, container_type: String) -> Array[BagSlot]:
	var result: Array[BagSlot] = []
	if grid == null:
		return result
	for child: Node in grid.get_children():
		if child is BagSlot:
			var slot: BagSlot = child as BagSlot
			slot.item_selected.connect(_on_slot_selected.bind(slot, container_type))
			result.append(slot)
	return result


## 任一数据变化（容器推送、钱包推送）时整体重绘；面板隐藏时跳过。
func _on_data_changed() -> void:
	if visible:
		_refresh_all()


## 重绘仓库区、背包区、容量与货币。
func _refresh_all() -> void:
	var warehouse_items: Array = _container_items(GameState.warehouse_container)
	var bag_items: Array = _filter_bag_items(GameState.bag_items)
	_warehouse_page = _render_page(_warehouse_slots, warehouse_items, _warehouse_page, WAREHOUSE_SLOTS_PER_PAGE)
	_bag_page = _render_page(_bag_slots, bag_items, _bag_page, BAG_SLOTS_PER_PAGE)
	var warehouse_pages: int = _page_count(warehouse_items.size(), WAREHOUSE_SLOTS_PER_PAGE)
	var bag_pages: int = _page_count(bag_items.size(), BAG_SLOTS_PER_PAGE)
	if _warehouse_page_label != null:
		_warehouse_page_label.text = str(_warehouse_page + 1)
	if _warehouse_prev_button != null:
		_warehouse_prev_button.disabled = _warehouse_page <= 0
	if _warehouse_next_button != null:
		_warehouse_next_button.disabled = _warehouse_page >= warehouse_pages - 1
	if _bag_page_input != null:
		_bag_page_input.text = str(_bag_page + 1)
	if _bag_prev_button != null:
		_bag_prev_button.disabled = _bag_page <= 0
	if _bag_next_button != null:
		_bag_next_button.disabled = _bag_page >= bag_pages - 1
	if _warehouse_capacity_label != null:
		_warehouse_capacity_label.text = "仓库 %s" % _capacity_text(GameState.warehouse_container)
	if _bag_capacity_label != null:
		_bag_capacity_label.text = _capacity_text(GameState.bag_container)
	_refresh_wallet()
	_sync_filter_buttons()


## 把一页物品写入格子；返回修正后的页码（物品减少导致当前页越界时回退到最后一页）。
func _render_page(slots: Array[BagSlot], items: Array, page: int, per_page: int) -> int:
	var max_page: int = _page_count(items.size(), per_page) - 1
	var resolved_page: int = clampi(page, 0, max(max_page, 0))
	var start: int = resolved_page * per_page
	for index: int in range(slots.size()):
		var slot: BagSlot = slots[index]
		var item_index: int = start + index
		if item_index < items.size() and items[item_index] is Dictionary:
			slot.set_item(items[item_index] as Dictionary)
		else:
			slot.clear_item()
		slot.set_selected(false)
	return resolved_page


## 读取容器快照中的物品数组，并按服务端格子序号排序，保证展示顺序稳定。
func _container_items(container: Dictionary) -> Array:
	var items_variant: Variant = container.get("items", [])
	var items: Array = (items_variant as Array).duplicate() if items_variant is Array else []
	items.sort_custom(func(left: Variant, right: Variant) -> bool:
		return BagUiMapper.slot_index(left as Dictionary) < BagUiMapper.slot_index(right as Dictionary))
	return items


## 按背包区当前筛选过滤物品；筛选口径与背包面板一致（装备 / 非装备）。
func _filter_bag_items(source: Array) -> Array:
	var result: Array = []
	for item_variant: Variant in _container_items({"items": source}):
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		var is_equipment: bool = BagUiMapper.is_equipment(item)
		if _bag_filter == FILTER_EQUIPMENT and not is_equipment:
			continue
		if _bag_filter == FILTER_OTHER and is_equipment:
			continue
		result.append(item)
	return result


## 生成「已用/容量」文案，容量缺失时回退到 max_capacity。
func _capacity_text(container: Dictionary) -> String:
	var items_variant: Variant = container.get("items", [])
	var used: int = int(container.get("used_slots", (items_variant as Array).size() if items_variant is Array else 0))
	var capacity: int = int(container.get("capacity", container.get("max_capacity", 0)))
	return "%d/%d" % [used, capacity]


## 用服务端钱包快照刷新金银铜，展示口径与背包面板一致。
func _refresh_wallet() -> void:
	var wallet: Dictionary = GameState.wallet_snapshot
	if _gold_label != null:
		_gold_label.text = UiFormat.value_to_text(int(wallet.get("gold", 0)))
	if _silver_label != null:
		_silver_label.text = UiFormat.value_to_text(int(wallet.get("silver", 0)))
	if _copper_label != null:
		_copper_label.text = UiFormat.value_to_text(int(wallet.get("copper", 0)))


## 同步筛选按钮的按下态。
func _sync_filter_buttons() -> void:
	if _filter_all_button != null:
		_filter_all_button.set_pressed_no_signal(_bag_filter == FILTER_ALL)
	if _filter_equipment_button != null:
		_filter_equipment_button.set_pressed_no_signal(_bag_filter == FILTER_EQUIPMENT)
	if _filter_other_button != null:
		_filter_other_button.set_pressed_no_signal(_bag_filter == FILTER_OTHER)


## 计算总页数，至少 1 页。
func _page_count(item_count: int, per_page: int) -> int:
	return max(1, int(ceil(float(item_count) / float(per_page))))


## 仓库区翻页；step 为 -1 或 1。
func _on_warehouse_page_step(step: int) -> void:
	_warehouse_page += step
	_clear_selection()
	_refresh_all()


## 背包区翻页；step 为 -1 或 1。
func _on_bag_page_step(step: int) -> void:
	_bag_page += step
	_clear_selection()
	_refresh_all()


## 切换背包区筛选并回到第一页。
func _on_filter_pressed(filter_key: String) -> void:
	_bag_filter = filter_key
	_bag_page = 0
	_clear_selection()
	_refresh_all()


## 点击格子：记录选中物品，在格子旁弹出「存入仓库」或「取出到背包」。
func _on_slot_selected(item: Dictionary, slot: BagSlot, container_type: String) -> void:
	if _request_in_flight or item.is_empty():
		return
	_clear_selection()
	_selected_item = item.duplicate(true)
	_selected_container = container_type
	slot.set_selected(true)
	_ensure_action_menu()
	if _action_menu == null:
		return
	var is_bag: bool = container_type == "bag"
	_action_menu.configure_actions([
		{
			"key": ACTION_DEPOSIT if is_bag else ACTION_WITHDRAW,
			"label": "存入仓库" if is_bag else "取出到背包",
			"disabled": false,
		},
	])
	_action_menu.open_near(slot)


## 懒创建动作菜单并绑定回调。
func _ensure_action_menu() -> void:
	if _action_menu != null:
		return
	_action_menu = ACTION_MENU_SCENE.instantiate() as ActionMenuPopup
	if _action_menu == null:
		return
	add_child(_action_menu)
	_action_menu.action_selected.connect(_on_action_selected)
	_action_menu.menu_closed.connect(_on_action_menu_closed)


## 动作菜单被点空白关闭时取消选中，避免格子残留高亮。
func _on_action_menu_closed() -> void:
	if not _request_in_flight:
		_clear_selection()


## 执行存入 / 取出：整格移动，数量取格子当前数量；全程展示通用 loading。
func _on_action_selected(action_key: String) -> void:
	if _request_in_flight or _selected_item.is_empty() or _warehouse_entity_id <= 0:
		return
	var slot_index: int = BagUiMapper.slot_index(_selected_item)
	var quantity: int = max(1, BagUiMapper.quantity(_selected_item))
	var request_seq: int = 0
	var request_cmd: int = 0
	if action_key == ACTION_DEPOSIT:
		request_cmd = CommandIds.BAG_TO_WAREHOUSE_REQ
		request_seq = App.request_bag_to_warehouse(_warehouse_entity_id, slot_index, quantity)
	elif action_key == ACTION_WITHDRAW:
		request_cmd = CommandIds.WAREHOUSE_TO_BAG_REQ
		request_seq = App.request_warehouse_to_bag(_warehouse_entity_id, slot_index, quantity)
	if request_seq <= 0:
		App.notice_received.emit("请求发送失败，请稍后重试。")
		_clear_selection()
		return
	_request_in_flight = true
	_show_loading()
	var result: Dictionary = await _wait_single_request(request_seq, request_cmd)
	_hide_loading()
	_request_in_flight = false
	_clear_selection()
	if not bool(result.get("succeeded", false)):
		App.notice_received.emit(_transfer_failure_text(action_key, result.get("payload", {})))
	# 成功时两个容器的新快照已由 5011 推送写入 GameState，这里补一次重绘兜底。
	_refresh_all()


## 生成存取失败提示：优先展示服务端原因，缺失时给出通用文案。
func _transfer_failure_text(action_key: String, payload_variant: Variant) -> String:
	var payload: Dictionary = payload_variant as Dictionary if payload_variant is Dictionary else {}
	var reason: String = str(payload.get("reason", payload.get("msg", ""))).strip_edges()
	var action_text: String = "存入仓库" if action_key == ACTION_DEPOSIT else "取出到背包"
	if reason.is_empty():
		return "%s失败，请检查空间后重试。" % action_text
	return "%s失败：%s" % [action_text, reason]


## 等待一组同命令请求全部结束；全部成功才返回 true。
func _wait_requests(seqs: Array[int], request_cmd: int) -> bool:
	var pending: Dictionary = {}
	for seq: int in seqs:
		pending[seq] = true
	var all_succeeded: bool = true
	while not pending.is_empty():
		var result: Array = await App.request_finished
		if result.size() < 5 or int(result[0]) != request_cmd:
			continue
		var seq: int = int(result[1])
		if not pending.has(seq):
			continue
		pending.erase(seq)
		if not bool(result[2]):
			all_succeeded = false
	return all_succeeded


## 等待单个请求结束，返回是否成功与回包载荷。
func _wait_single_request(expected_seq: int, request_cmd: int) -> Dictionary:
	while true:
		var result: Array = await App.request_finished
		if result.size() < 5 or int(result[0]) != request_cmd or int(result[1]) != expected_seq:
			continue
		return {"succeeded": bool(result[2]), "payload": result[4]}
	return {"succeeded": false, "payload": {}}


## 清除选中物品与格子高亮。
func _clear_selection() -> void:
	_selected_item = {}
	_selected_container = ""
	for slot: BagSlot in _warehouse_slots:
		slot.set_selected(false)
	for slot: BagSlot in _bag_slots:
		slot.set_selected(false)


## 展示通用全屏 loading（懒创建）。
func _show_loading() -> void:
	if _loading == null:
		_loading = LOADING_SCENE.instantiate() as GenericLoadingScene
		if _loading == null:
			return
		add_child(_loading)
	_loading.show_loading()


## 隐藏通用 loading。
func _hide_loading() -> void:
	if _loading != null:
		_loading.hide_loading()
