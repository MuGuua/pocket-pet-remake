extends RuntimeRootPanel
class_name BagPanel

const SLOTS_PER_PAGE: int = 28
const DEFAULT_CAPACITY: int = 300
const FILTER_ALL: String = "all"
const FILTER_EQUIPMENT: String = "equipment"
const FILTER_OTHER: String = "other"
const BAG_ITEM_DETAIL_SCENE: PackedScene = preload("res://scenes/ui/bag/bag_item_detail.tscn")
const RUNTIME_PROGRESS_OVERLAY_SCENE: PackedScene = preload("res://scenes/ui/common/runtime_progress_overlay.tscn")
const RUNTIME_PROGRESS_BAR_OVERLAY_SCENE: PackedScene = preload(RuntimeProgressBarOverlay.SCENE_PATH)
const REWARD_POPUP_SCENE: PackedScene = preload("res://scenes/ui/common/reward_popup.tscn")
const EQUIPMENT_ENHANCE_POPUP_SCENE: PackedScene = preload("res://scenes/ui/bag/equipment_enhance_popup.tscn")
const DROP_ITEM_POPUP_SCENE: PackedScene = preload("res://scenes/ui/bag/drop_item_popup.tscn")
const REPAIR_EQUIPMENT_POPUP_SCENE: PackedScene = preload("res://scenes/ui/bag/repair_equipment_popup.tscn")
const USE_ITEM_TARGET_PICKER_SCENE: PackedScene = preload("res://scenes/ui/bag/use_item_target_picker.tscn")
## 礼包打开进度条播放时长（秒）。
const BOX_OPEN_PROGRESS_DURATION_SEC: float = 3.0
## 礼包打开进度条提示文案。
const BOX_OPEN_PROGRESS_STATUS_TEXT: String = "打开中..."
## 礼包开启成功后奖励弹窗标题。
## 等待 USE_ITEM 回包的最长帧数（约 15 秒），防止界面永久卡住。
const BOX_OPEN_RESPONSE_TIMEOUT_FRAMES: int = 900
## 普通消耗品使用进度条时长（秒）。
const USE_ITEM_PROGRESS_DURATION_SEC: float = 1.0
## 普通消耗品使用进度条提示文案。
const USE_ITEM_PROGRESS_STATUS_TEXT: String = "使用中..."
## 拉取目标列表请求超时帧数。
const USE_TARGET_FETCH_TIMEOUT_FRAMES: int = 300
## 详情面板与锚点按钮之间的垂直间距（像素）。
const DETAIL_ANCHOR_GAP_Y: float = 6.0
## 锚点横向位置低于该比例时，面板出现在右上方。
const DETAIL_ZONE_LEFT_THRESHOLD: float = 0.33
## 锚点横向位置高于该比例时，面板出现在左上方。
const DETAIL_ZONE_RIGHT_THRESHOLD: float = 0.66

enum DetailAnchorZone {
	LEFT,
	CENTER,
	RIGHT,
}

@onready var _close_button: Button = $RootPanel/MarginContainer/VBoxContainer/Title/HBoxContainer/Button
@onready var _root_panel: PanelContainer = $RootPanel
@onready var _equipment_root: Node = $RootPanel/MarginContainer/VBoxContainer/EquipmentSlot
@onready var _capacity_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/PanelContainer/HBoxContainer/Label
@onready var _gold_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/GoldCoinAmount
@onready var _silver_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/SilverCoinAmount
@onready var _copper_label: Label = $RootPanel/MarginContainer/VBoxContainer/Property/HBoxContainer/MarginContainer/PanelContainer2/HBoxContainer2/CopperCoinAmount
@onready var _slot_grid: GridContainer = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer/GridContainer
@onready var _filter_all_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/Button
@onready var _filter_equipment_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/Button2
@onready var _filter_other_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/Button3
@onready var _prev_page_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/Button4
@onready var _next_page_button: Button = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/Button5
@onready var _page_input: LineEdit = $RootPanel/MarginContainer/VBoxContainer/MarginContainer/PanelContainer/VBoxContainer/MarginContainer2/PanelContainer/MarginContainer/HBoxContainer/MarginContainer/HBoxContainer/PanelContainer/PageInput

## 当前页内 28 个背包格子实例缓存。
var _slots: Array[BagSlot] = []
## 当前可卸装的装备槽节点缓存。
var _equipment_slots: Array[EquipmentSlot] = []
## 详情 overlay 根节点；覆盖背包面板并带独立遮罩，点空白只关详情不关背包。
var _detail_overlay: Control = null
## 详情 overlay 的全屏遮罩。
var _detail_dim: ColorRect = null
## 详情弹层里的内容面板。
var _detail_panel: BagItemDetail = null
## 当前页索引，从 0 开始。
var _current_page: int = 0
## 当前背包筛选分类：all / equipment / other。
var _current_filter: String = FILTER_ALL
## 当前选中的背包格子编号，使用服务端 slot_index。
var _selected_slot_index: int = 0
## 当前选中的服务端物品快照，供详情弹层和操作按钮复用。
var _selected_item: Dictionary = {}
## 当前选中的已穿戴装备槽位标识；用于详情弹层卸装和刷新。
var _selected_equip_slot_key: String = ""
## 当前详情面板相对定位所使用的锚点控件（背包格子或装备槽）。
var _detail_anchor: Control = null
## 通用请求 loading 遮罩；等待回包等流程复用 GenericLoadingScene。
var _progress_overlay: RuntimeProgressOverlay = null
## 开礼包专用线性进度条遮罩；保留固定 3 秒进度条演出，不走 GenericLoadingScene。
var _box_open_progress_overlay: RuntimeProgressBarOverlay = null
## 通用奖励弹窗实例。
var _reward_popup: RewardPopup = null
## 丢弃物品专用确认弹窗。
var _drop_popup: DropItemPopup = null
## 装备强化弹窗实例。
var _enhance_popup: EquipmentEnhancePopup = null
## 当前强化目标 item_uid，用于背包刷新后回写弹窗。
var _enhance_target_item_uid: String = ""
## 强化回包监听 Callable。
var _enhance_request_handler: Callable = Callable()
## 装备修复确认弹窗。
var _repair_popup: RepairEquipmentPopup = null
## 是否正在执行装备修复请求。
var _repair_in_flight: bool = false
## 当前修复请求 seq。
var _repair_request_seq: int = 0
## 修复成功回包后，是否正在等待最新背包快照落地再结束 loading。
var _repair_waiting_bag_refresh: bool = false
## 修复成功回包缓存；等背包快照写入后再统一提示结果。
var _repair_response_payload: Dictionary = {}
## 修复回包监听 Callable。
var _repair_request_handler: Callable = Callable()
## 是否正在执行礼包打开演出，避免重复点击。
var _box_open_in_flight: bool = false
## 当前礼包打开请求对应的 seq，用于匹配 USE_ITEM 回包。
var _box_open_request_seq: int = 0
## 礼包打开演出代次；取消或新一次打开时递增，避免旧 await 误弹奖励。
var _box_open_presentation_id: int = 0
## 是否已收到当前礼包打开请求的回包。
var _box_open_response_ready: bool = false
## 当前礼包打开请求是否成功。
var _box_open_response_ok: bool = false
## 当前礼包打开请求的回包载荷。
var _box_open_response_payload: Dictionary = {}
## USE_ITEM 回包监听 Callable，便于 disconnect。
var _box_open_request_handler: Callable = Callable()
## 礼包打开演出期间暂缓刷新背包格子/钱包展示，避免与成功弹窗同时跳变。
var _defer_bag_visual_refresh: bool = false
## 丢弃请求是否进行中，避免重复提交。
var _drop_in_flight: bool = false
## 当前丢弃请求 seq，用于匹配 App.request_finished。
var _drop_request_seq: int = 0
## 丢弃回包监听 Callable。
var _drop_request_handler: Callable = Callable()
## 当前丢弃流程缓存的物品名，用于回包字段缺失时兜底提示。
var _pending_drop_item_name: String = ""
## 丢弃成功回包载荷；等待背包完整快照刷新后再用于提示玩家。
var _drop_response_payload: Dictionary = {}
## 是否已收到丢弃成功回包，正在等待服务端最新背包快照落地。
var _drop_waiting_bag_refresh: bool = false
## 打开背包加载代次；关闭面板或重复打开时递增，用于取消进行中的 await。
var _open_bag_load_generation: int = 0
## 丢弃成功后主动补拉当前页的请求 seq，用于避免旧 BAG_LIST 回包提前结束 loading。
var _drop_refresh_request_seq: int = 0
## 当前丢弃后的 BAG_LIST 回包是否已经到达；完整快照信号需在该标记之后才可信。
var _drop_refresh_response_received: bool = false
## 丢弃刷新阶段是否已收到 BAG_LIST 或 BAG_UPDATE_PUSH 写入 GameState 的快照信号。
var _drop_snapshot_applied_received: bool = false
## 消耗品目标选择弹窗实例。
var _use_target_picker: UseItemTargetPicker = null
## 目标选择结果：0 等待中，1 已选中，2 已取消。
var _use_target_choice_state: int = 0
## 目标选择结果缓存。
var _use_target_choice_result: Dictionary = {}
## 是否正在等待 USE_ITEM 回包。
var _use_item_in_flight: bool = false
## 当前 USE_ITEM 请求 seq。
var _use_item_request_seq: int = 0
## USE_ITEM 成功回包后，是否正在等待最新背包快照落地再结束 loading。
var _use_item_waiting_bag_refresh: bool = false
## USE_ITEM 成功回包缓存；等背包快照写入后再统一提示结果。
var _use_item_response_payload: Dictionary = {}
## USE_ITEM 回包监听 Callable。
var _use_item_request_handler: Callable = Callable()
## 拉取宠物/装备目标列表时的请求 seq。
var _use_target_fetch_request_seq: int = 0
## 拉取目标列表回包是否就绪。
var _use_target_fetch_ready: bool = false
## 拉取目标列表回包是否成功。
var _use_target_fetch_ok: bool = false
## 拉取目标列表回包监听 Callable。
var _use_target_fetch_handler: Callable = Callable()


## 初始化背包面板：绑定按钮、订阅 GameState，并懒创建详情弹层。
func _ready() -> void:
	super._ready()
	if _close_button != null:
		_close_button.pressed.connect(close_menu)
	if _filter_all_button != null:
		_filter_all_button.pressed.connect(_on_filter_button_pressed.bind(FILTER_ALL))
	if _filter_equipment_button != null:
		_filter_equipment_button.pressed.connect(_on_filter_button_pressed.bind(FILTER_EQUIPMENT))
	if _filter_other_button != null:
		_filter_other_button.pressed.connect(_on_filter_button_pressed.bind(FILTER_OTHER))
	if _prev_page_button != null:
		_prev_page_button.pressed.connect(_on_prev_page_pressed)
	if _next_page_button != null:
		_next_page_button.pressed.connect(_on_next_page_pressed)
	if _page_input != null:
		_page_input.text_submitted.connect(_on_page_input_submitted)
		_page_input.focus_exited.connect(_on_page_input_focus_exited)
	if not GameState.bag_changed.is_connected(_on_bag_data_changed):
		GameState.bag_changed.connect(_on_bag_data_changed)
	if not GameState.wallet_changed.is_connected(_on_bag_data_changed):
		GameState.wallet_changed.connect(_on_bag_data_changed)
	if not GameState.equipment_changed.is_connected(_refresh_equipment_slots):
		GameState.equipment_changed.connect(_refresh_equipment_slots)
	_collect_bag_slots()
	_collect_equipment_slots(_equipment_root, _equipment_slots)
	_bind_equipment_slot_actions()
	_ensure_detail_overlay()
	_sync_filter_buttons()
	_refresh_panel()


## 退出场景时注销 GameState 订阅，避免二次打开时重复回调。
func _exit_tree() -> void:
	if GameState.bag_changed.is_connected(_on_bag_data_changed):
		GameState.bag_changed.disconnect(_on_bag_data_changed)
	if GameState.wallet_changed.is_connected(_on_bag_data_changed):
		GameState.wallet_changed.disconnect(_on_bag_data_changed)
	if GameState.equipment_changed.is_connected(_refresh_equipment_slots):
		GameState.equipment_changed.disconnect(_refresh_equipment_slots)


## 打开背包前先重置分页/筛选状态，供主场景在 loading 期间调用。
func _reset_open_state() -> void:
	_current_filter = FILTER_ALL
	_current_page = 0
	_selected_slot_index = 0
	_selected_item = {}
	_defer_bag_visual_refresh = false
	_sync_filter_buttons()


## 面板展示前拉取背包页与已穿戴装备；自身保持隐藏，由主场景负责 loading。
func prepare_open_data() -> bool:
	_open_bag_load_generation += 1
	var load_id: int = _open_bag_load_generation
	_reset_open_state()
	if not GameState.is_ws_authenticated:
		return load_id == _open_bag_load_generation
	var bag_seq: int = _send_current_bag_page_request()
	if bag_seq <= 0:
		return false
	var equip_seq: int = App.request_player_equipment_list()
	var wait_result: Dictionary = await _wait_open_bag_requests(bag_seq, equip_seq)
	if load_id != _open_bag_load_generation:
		return false
	return bool(wait_result.get("all_succeeded", false))


## 数据就绪后打开背包；不再在此处重复请求服务端。
func open_menu() -> void:
	super.open_menu()
	_refresh_panel()


## 并行等待背包列表与已穿戴装备列表回包；打开背包前缩短总等待时间。
func _wait_open_bag_requests(bag_seq: int, equip_seq: int) -> Dictionary:
	var bag_done: bool = bag_seq <= 0
	var equip_done: bool = equip_seq <= 0
	var bag_succeeded: bool = bag_seq <= 0
	var equip_succeeded: bool = equip_seq <= 0
	while not bag_done or not equip_done:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if not bag_done and request_cmd == CommandIds.BAG_LIST_REQ and seq == bag_seq:
			bag_done = true
			bag_succeeded = bool(result[2])
		elif not equip_done and request_cmd == CommandIds.PLAYER_EQUIPMENT_LIST_REQ and seq == equip_seq:
			equip_done = true
			equip_succeeded = bool(result[2])
	return {
		"bag_succeeded": bag_succeeded,
		"equip_succeeded": equip_succeeded,
		"all_succeeded": bag_succeeded and equip_succeeded,
	}


## 等待指定 seq 的 BAG_LIST 请求结束。
func _wait_bag_list_request(expected_seq: int) -> Dictionary:
	while expected_seq > 0:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.BAG_LIST_REQ or seq != expected_seq:
			continue
		return {
			"succeeded": bool(result[2]),
			"response_cmd": int(result[3]),
			"payload": result[4],
		}
	return {
		"succeeded": false,
		"response_cmd": 0,
		"payload": {},
	}


## 等待指定 seq 的人物装备列表请求结束。
func _wait_equipment_list_request(expected_seq: int) -> Dictionary:
	while expected_seq > 0:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PLAYER_EQUIPMENT_LIST_REQ or seq != expected_seq:
			continue
		return {
			"succeeded": bool(result[2]),
			"response_cmd": int(result[3]),
			"payload": result[4],
		}
	return {
		"succeeded": false,
		"response_cmd": 0,
		"payload": {},
	}


## 关闭背包并同步关闭详情弹层，避免旧选中状态残留到下一次打开。
func close_menu() -> void:
	_open_bag_load_generation += 1
	if _progress_overlay != null:
		_progress_overlay.hide_overlay()
	_use_item_waiting_bag_refresh = false
	_use_item_response_payload = {}
	_repair_waiting_bag_refresh = false
	_repair_response_payload = {}
	_cancel_box_open_presentation()
	_cancel_use_item_presentation()
	_force_close_reward_popup()
	_hide_enhance_popup()
	_hide_drop_popup()
	_hide_repair_popup()
	_end_box_open_deferred_refresh(false)
	_hide_detail_popup()
	super.close_menu()


## 关闭背包内所有 overlay（详情层与礼包打开进度）。
func _close_all_overlays() -> void:
	_cancel_box_open_presentation()
	_cancel_use_item_presentation()
	_hide_detail_popup()


## 点根面板遮罩时若详情已打开，仅关闭详情层。
func _dismiss_top_overlay() -> bool:
	if _detail_overlay != null and _detail_overlay.visible:
		_hide_detail_popup()
		return true
	return false


## 背包或钱包快照变化后，只在面板可见且未处于礼包演出暂缓期时刷新。
func _on_bag_data_changed() -> void:
	if not visible or _defer_bag_visual_refresh:
		return
	_refresh_panel()
	_refresh_enhance_popup_if_open()


## 用最新的服务端快照重绘容量、格子、分页和已装备物品。
func _refresh_panel() -> void:
	_refresh_capacity_and_wallet()
	_refresh_bag_slots()
	_update_page_controls()
	_refresh_equipment_slots()
	_refresh_selected_item_state()


## 刷新容量和钱包文案；所有数值都统一转成整数显示文本。
func _refresh_capacity_and_wallet() -> void:
	var capacity: int = _resolve_capacity()
	var used_slots: int = int(GameState.bag_container.get("used_slots", _count_used_bag_slots()))
	if _capacity_label != null:
		_capacity_label.text = "%d/%d" % [used_slots, capacity]

	var wallet: Dictionary = GameState.wallet_snapshot
	if _gold_label != null:
		_gold_label.text = UiFormat.value_to_text(wallet.get("gold", 0))
	if _silver_label != null:
		_silver_label.text = UiFormat.value_to_text(wallet.get("silver", 0))
	if _copper_label != null:
		_copper_label.text = UiFormat.value_to_text(wallet.get("copper", 0))


## 根据当前服务端返回的分页列表刷新这一页 28 个格子；列表中的第 1 个元素渲染到 1 号格，不允许中间留空。
func _refresh_bag_slots() -> void:
	if _slots.is_empty():
		return

	var page_items: Array = _collect_bag_items()
	var page_count: int = _get_page_count()
	if page_count <= 0:
		_current_page = 0
	else:
		_current_page = clampi(_current_page, 0, page_count - 1)

	for index: int in range(_slots.size()):
		var slot: BagSlot = _slots[index]
		if index < page_items.size():
			var item: Dictionary = page_items[index] as Dictionary
			slot.set_item(item)
			slot.set_selected(BagUiMapper.slot_index(item) == _selected_slot_index)
		else:
			slot.clear_item()


## 刷新上一页/下一页按钮和页码输入框状态。
func _update_page_controls() -> void:
	var page_count: int = _get_page_count()
	if _prev_page_button != null:
		_prev_page_button.disabled = _current_page <= 0 or page_count <= 1
	if _next_page_button != null:
		_next_page_button.disabled = _current_page >= page_count - 1 or page_count <= 1
	_sync_page_input()


## 让输入框始终显示当前页码，避免服务端快照刷新后页码和实际页面不一致。
func _sync_page_input() -> void:
	if _page_input == null or _page_input.has_focus():
		return
	var page_count: int = max(_get_page_count(), 1)
	_page_input.text = str(_current_page + 1)
	_page_input.tooltip_text = "第 %d / %d 页" % [_current_page + 1, page_count]


## 切到上一页。
func _on_prev_page_pressed() -> void:
	_set_page(_current_page - 1)


## 切到下一页。
func _on_next_page_pressed() -> void:
	_set_page(_current_page + 1)


## 玩家按回车提交页码时，立即按输入值切页。
func _on_page_input_submitted(new_text: String) -> void:
	_apply_page_input(new_text)
	if _page_input != null:
		_page_input.release_focus()


## 页码输入框失焦后也要把输入值应用到当前页，避免移动端关闭键盘后不生效。
func _on_page_input_focus_exited() -> void:
	if _page_input == null:
		return
	_apply_page_input(_page_input.text)


## 解析页码输入；非法输入直接恢复成当前页。
func _apply_page_input(raw_text: String) -> void:
	var trimmed_text: String = raw_text.strip_edges()
	if trimmed_text.is_empty() or not trimmed_text.is_valid_int():
		_sync_page_input()
		return
	_set_page(int(trimmed_text) - 1)


## 切换当前页并刷新该页格子。
func _set_page(page_index: int) -> void:
	var page_count: int = _get_page_count()
	if page_count <= 0:
		_current_page = 0
	else:
		_current_page = clampi(page_index, 0, page_count - 1)
	_hide_detail_popup()
	_request_current_bag_page()


## 根据当前筛选结果和每页格数计算总页数。
func _get_page_count() -> int:
	var total_items: int = int(GameState.bag_container.get("total_items", _collect_bag_items().size()))
	if total_items <= 0:
		return 1
	return int(ceil(float(total_items) / float(SLOTS_PER_PAGE)))


## 解析当前背包总容量；服务端未回包时使用正式默认容量兜底。
func _resolve_capacity() -> int:
	var capacity: int = int(GameState.bag_container.get("capacity", GameState.bag_container.get("max_capacity", 0)))
	if capacity <= 0:
		capacity = max(DEFAULT_CAPACITY, _count_used_bag_slots())
	return capacity


## 统计当前背包中已占用的格子数。
func _count_used_bag_slots() -> int:
	var count: int = 0
	for item_variant: Variant in _collect_bag_items():
		if item_variant is Dictionary:
			count += 1
	return count


## 返回当前服务端权威背包物品列表；优先使用 container.items，兼容旧字段 bag_items。
func _collect_bag_items() -> Array:
	var container_items: Variant = GameState.bag_container.get("items", [])
	if container_items is Array:
		return (container_items as Array).duplicate(true)
	return GameState.bag_items.duplicate(true)


## 收集当前页 28 个 BagSlot，并绑定选中事件。
func _collect_bag_slots() -> void:
	_slots.clear()
	if _slot_grid == null:
		return
	for child: Node in _slot_grid.get_children():
		if child is BagSlot:
			var slot: BagSlot = child as BagSlot
			if not slot.item_selected.is_connected(_on_slot_item_selected):
				slot.item_selected.connect(_on_slot_item_selected)
			_slots.append(slot)


## 点击背包格子后记录服务端 slot_index，并打开详情弹层让玩家走权威操作。
func _on_slot_item_selected(item: Dictionary) -> void:
	_selected_equip_slot_key = ""
	_selected_item = item.duplicate(true)
	_selected_slot_index = BagUiMapper.slot_index(_selected_item)
	_apply_selected_slot_visuals()
	var anchor_slot: BagSlot = _find_visible_slot_by_index(_selected_slot_index)
	_show_item_detail(_selected_item, anchor_slot)


## 响应底部分类按钮切换，并把分页重置到筛选结果的第一页。
func _on_filter_button_pressed(filter_key: String) -> void:
	if filter_key == _current_filter:
		return
	_current_filter = filter_key
	_current_page = 0
	_hide_detail_popup()
	_sync_filter_buttons()
	_request_current_bag_page()


## 根据服务端已佩戴装备快照刷新所有装备槽显示。
func _refresh_equipment_slots() -> void:
	for slot: EquipmentSlot in _equipment_slots:
		slot.refresh_from_game_state()
	_refresh_equipped_detail_if_needed()


## 递归收集装备槽组件，供背包面板统一刷新和绑定卸装事件。
func _collect_equipment_slots(node: Node, result: Array[EquipmentSlot]) -> void:
	if node == null:
		return
	if node is EquipmentSlot:
		result.append(node as EquipmentSlot)
	for child: Node in node.get_children():
		_collect_equipment_slots(child, result)


## 给所有装备槽绑定按下事件；点击已装备槽位时打开详情弹层。
func _bind_equipment_slot_actions() -> void:
	for slot: EquipmentSlot in _equipment_slots:
		if not slot.pressed.is_connected(_on_equipment_slot_pressed.bind(slot)):
			slot.pressed.connect(_on_equipment_slot_pressed.bind(slot))


## 点击已装备槽位后打开详情弹层；空槽点击只给出提示。
func _on_equipment_slot_pressed(slot: EquipmentSlot) -> void:
	if slot == null or not slot.has_equipment():
		App.notice_received.emit("该装备槽当前没有已装备物品。")
		return
	var equip_slot_key: String = slot.get_equip_slot_key()
	if equip_slot_key.is_empty():
		App.notice_received.emit("装备槽标识缺失，暂时无法查看详情。")
		return
	_selected_item.clear()
	_selected_slot_index = 0
	_apply_selected_slot_visuals()
	_selected_equip_slot_key = equip_slot_key
	_show_equipped_item_detail(slot.get_equipment(), slot)


## 刷新页内格子的选中态，确保翻页、筛选切换或服务端回包后高亮状态仍然正确。
func _apply_selected_slot_visuals() -> void:
	var page_items: Array = _collect_bag_items()
	for index: int in range(_slots.size()):
		if index < page_items.size():
			var item: Dictionary = page_items[index] as Dictionary
			_slots[index].set_selected(BagUiMapper.slot_index(item) == _selected_slot_index)
		else:
			_slots[index].set_selected(false)


## 根据最新服务端快照和当前筛选结果同步选中物品；若已不存在或不在当前筛选内则清掉选中和详情弹层。
func _refresh_selected_item_state() -> void:
	if _selected_slot_index <= 0:
		return
	var matched_item: Dictionary = {}
	for item_variant: Variant in _collect_bag_items():
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		if BagUiMapper.slot_index(item) == _selected_slot_index:
			matched_item = item.duplicate(true)
			break
	if matched_item.is_empty():
		_selected_item.clear()
		_selected_slot_index = 0
		_apply_selected_slot_visuals()
		_hide_detail_popup()
		return
	_selected_item = matched_item
	if _detail_panel != null and _detail_overlay != null and _detail_overlay.visible:
		_detail_panel.set_item(_selected_item)


## 懒创建背包详情 overlay：全屏遮罩 + 居中详情，点遮罩只关详情不关背包。
func _ensure_detail_overlay() -> void:
	if _detail_overlay != null:
		return
	_detail_overlay = Control.new()
	_detail_overlay.name = "DetailOverlay"
	_detail_overlay.set_anchors_preset(Control.PRESET_FULL_RECT)
	_detail_overlay.offset_left = 0.0
	_detail_overlay.offset_top = 0.0
	_detail_overlay.offset_right = 0.0
	_detail_overlay.offset_bottom = 0.0
	_detail_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_detail_overlay.hide()
	add_child(_detail_overlay)

	_detail_dim = ColorRect.new()
	_detail_dim.name = "DetailDim"
	_detail_dim.set_anchors_preset(Control.PRESET_FULL_RECT)
	_detail_dim.offset_left = 0.0
	_detail_dim.offset_top = 0.0
	_detail_dim.offset_right = 0.0
	_detail_dim.offset_bottom = 0.0
	_detail_dim.color = Color(0.0, 0.0, 0.0, 0.42)
	_detail_dim.mouse_filter = Control.MOUSE_FILTER_STOP
	_detail_dim.gui_input.connect(_on_detail_dim_gui_input)
	_detail_overlay.add_child(_detail_dim)

	_detail_panel = BAG_ITEM_DETAIL_SCENE.instantiate() as BagItemDetail
	if _detail_panel == null:
		return
	_detail_panel.mouse_filter = Control.MOUSE_FILTER_STOP
	_detail_overlay.add_child(_detail_panel)
	if not _detail_panel.action_requested.is_connected(_on_item_action_requested):
		_detail_panel.action_requested.connect(_on_item_action_requested)


## 详情遮罩收到按下输入时只关闭详情层。
func _on_detail_dim_gui_input(event: InputEvent) -> void:
	if _detail_overlay == null or not _detail_overlay.visible:
		return
	if not _is_dismiss_event(event):
		return
	if _detail_dim != null:
		_detail_dim.accept_event()
	_hide_detail_popup()


## 显示当前选中物品详情，并根据点击格子动态定位到按钮上方。
func _show_item_detail(item: Dictionary, anchor: Control = null) -> void:
	if item.is_empty():
		return
	_selected_equip_slot_key = ""
	_ensure_detail_overlay()
	if _detail_overlay == null or _detail_panel == null:
		return
	_detail_anchor = anchor
	_detail_panel.set_item(item)
	_open_detail_popup(anchor)


## 显示已穿戴装备详情，并根据点击装备槽动态定位到按钮上方。
func _show_equipped_item_detail(item: Dictionary, anchor: Control = null) -> void:
	if item.is_empty():
		return
	_ensure_detail_overlay()
	if _detail_overlay == null or _detail_panel == null:
		return
	_detail_anchor = anchor
	_detail_panel.set_equipped_item(item)
	_open_detail_popup(anchor)


## 打开详情 overlay 并置于当前面板最顶层，随后按锚点控件计算面板位置。
func _open_detail_popup(anchor: Control = null) -> void:
	if _detail_overlay == null:
		return
	_detail_overlay.show()
	move_child(_detail_overlay, get_child_count() - 1)
	call_deferred("_apply_detail_panel_position", anchor)


## 关闭详情 overlay 并清空当前选中物品，避免旧高亮残留到下一次交互。
func _hide_detail_popup() -> void:
	if _detail_overlay != null:
		_detail_overlay.hide()
	if _detail_panel != null:
		_detail_panel.clear_item()
	_detail_anchor = null
	_selected_item.clear()
	_selected_slot_index = 0
	_selected_equip_slot_key = ""
	_apply_selected_slot_visuals()


## 已穿戴装备快照变化后，若详情弹层仍打开则同步最新数据；卸空后自动关闭弹层。
func _refresh_equipped_detail_if_needed() -> void:
	if _selected_equip_slot_key.is_empty() or _detail_overlay == null or not _detail_overlay.visible:
		return
	if _detail_panel == null:
		return
	var matched_item: Dictionary = _find_equipped_item_by_slot(_selected_equip_slot_key)
	if matched_item.is_empty():
		_hide_detail_popup()
		return
	_selected_item.clear()
	_detail_panel.set_equipped_item(matched_item)


## 从 GameState 查找指定槽位当前已穿戴装备。
func _find_equipped_item_by_slot(slot_key: String) -> Dictionary:
	for item_variant: Variant in GameState.equipped_items:
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		if str(item.get("equip_slot", "")) == slot_key:
			return item.duplicate(true)
	return {}


## 在当前页格子列表中查找与 slot_index 对应的 BagSlot，供详情面板定位。
func _find_visible_slot_by_index(slot_index: int) -> BagSlot:
	if slot_index <= 0:
		return null
	var page_items: Array = _collect_bag_items()
	for index: int in range(_slots.size()):
		if index >= page_items.size():
			continue
		var item_variant: Variant = page_items[index]
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		if BagUiMapper.slot_index(item) == slot_index:
			return _slots[index]
	return null


## 根据锚点在 overlay 中的横向位置，判断详情面板应出现在右上方 / 正上方 / 左上方。
func _resolve_detail_anchor_zone(anchor: Control) -> DetailAnchorZone:
	if anchor == null or _detail_overlay == null:
		return DetailAnchorZone.CENTER
	var overlay_rect: Rect2 = _detail_overlay.get_global_rect()
	var anchor_rect: Rect2 = anchor.get_global_rect()
	var anchor_center_x: float = anchor_rect.position.x + anchor_rect.size.x * 0.5
	var relative_x: float = (anchor_center_x - overlay_rect.position.x) / maxf(overlay_rect.size.x, 1.0)
	if relative_x <= DETAIL_ZONE_LEFT_THRESHOLD:
		return DetailAnchorZone.LEFT
	if relative_x >= DETAIL_ZONE_RIGHT_THRESHOLD:
		return DetailAnchorZone.RIGHT
	return DetailAnchorZone.CENTER


## 将详情面板定位到锚点上方：左侧格子偏右、中间居中、右侧格子偏左，且不遮挡锚点。
func _apply_detail_panel_position(anchor: Control) -> void:
	if _detail_panel == null or _detail_overlay == null or not _detail_overlay.visible:
		return
	var panel_size: Vector2 = _detail_panel.size
	if panel_size.x <= 0.0 or panel_size.y <= 0.0:
		panel_size = _detail_panel.get_combined_minimum_size()
	if panel_size.x <= 0.0 or panel_size.y <= 0.0:
		return
	var bounds_rect: Rect2 = _detail_overlay.get_global_rect()
	var global_x: float = bounds_rect.position.x + (bounds_rect.size.x - panel_size.x) * 0.5
	var global_y: float = bounds_rect.position.y + (bounds_rect.size.y - panel_size.y) * 0.5
	if anchor != null and anchor.is_inside_tree():
		var anchor_rect: Rect2 = anchor.get_global_rect()
		var anchor_center_x: float = anchor_rect.position.x + anchor_rect.size.x * 0.5
		global_y = anchor_rect.position.y - panel_size.y - DETAIL_ANCHOR_GAP_Y
		match _resolve_detail_anchor_zone(anchor):
			DetailAnchorZone.LEFT:
				global_x = anchor_center_x
			DetailAnchorZone.CENTER:
				global_x = anchor_center_x - panel_size.x * 0.5
			DetailAnchorZone.RIGHT:
				global_x = anchor_center_x - panel_size.x
	global_x = clampf(global_x, bounds_rect.position.x, bounds_rect.position.x + bounds_rect.size.x - panel_size.x)
	global_y = clampf(global_y, bounds_rect.position.y, bounds_rect.position.y + bounds_rect.size.y - panel_size.y)
	_detail_panel.global_position = Vector2(global_x, global_y)


## 同步底部三个分类按钮的按下状态：左边全部、中间装备、右边其他。
func _sync_filter_buttons() -> void:
	if _filter_all_button != null:
		_filter_all_button.button_pressed = _current_filter == FILTER_ALL
	if _filter_equipment_button != null:
		_filter_equipment_button.button_pressed = _current_filter == FILTER_EQUIPMENT
	if _filter_other_button != null:
		_filter_other_button.button_pressed = _current_filter == FILTER_OTHER

## 向服务端请求当前页和当前分类的背包数据；旧协议默认值也统一在这里收口。
func _request_current_bag_page() -> void:
	_send_current_bag_page_request()


## 发送当前页背包数据请求，并返回请求 seq 供带 loading 的流程精确等待。
func _send_current_bag_page_request() -> int:
	if not GameState.is_ws_authenticated:
		return 0
	return App.request_bag_list(_current_page + 1, SLOTS_PER_PAGE, _current_filter)


## 处理详情弹层的按钮动作：装备/使用/卸下走服务端权威协议，分享暂留占位。
func _on_item_action_requested(action_key: String, item: Dictionary) -> void:
	if item.is_empty():
		return
	match action_key:
		"use", "open":
			_execute_primary_item_action(item)
		"unequip":
			_execute_unequip_action(item)
		"drop":
			_execute_drop_action(item)
		"give":
			App.notice_received.emit("给人功能尚未接入新版背包。")
		"share":
			App.notice_received.emit("分享功能尚未接入新版背包。")
		"enhance":
			_open_enhance_popup(item)
		"repair":
			_execute_repair_action(item)
		_:
			App.notice_received.emit("该操作尚未接入新版背包服务端链路。")


## 按当前已选装备槽向服务端发起卸装请求。
func _execute_unequip_action(item: Dictionary) -> void:
	var equip_slot_key: String = _selected_equip_slot_key
	if equip_slot_key.is_empty():
		equip_slot_key = str(item.get("equip_slot", "")).strip_edges()
	if equip_slot_key.is_empty():
		App.notice_received.emit("装备槽标识缺失，暂时无法卸下。")
		return
	App.request_player_unequip(equip_slot_key, "bag")
	_hide_detail_popup()


## 按物品类型把“主操作”分发到对应服务端权威接口。
func _execute_primary_item_action(item: Dictionary) -> void:
	var slot_index: int = BagUiMapper.slot_index(item)
	if slot_index <= 0:
		App.notice_received.emit("物品格子编号无效，暂时无法操作。")
		return
	if BagUiMapper.is_equipment(item):
		if BagUiMapper.is_damaged(item):
			App.notice_received.emit("该装备已损坏，请先修复后再佩戴。")
			return
		App.request_player_equip(slot_index, "bag")
		_hide_detail_popup()
		return
	if BagUiMapper.is_box_item(item):
		_execute_box_open_action(item)
		return
	if BagUiMapper.requires_pet_target(item):
		_run_use_item_with_pet_target(item)
		return
	if BagUiMapper.requires_equipment_target(item):
		_run_use_item_with_equipment_target(item)
		return
	_run_generic_use_item_action(item)


## 宠物目标类消耗品：选宠后走 USE_ITEM 权威链路。
func _run_use_item_with_pet_target(item: Dictionary) -> void:
	if _use_item_in_flight:
		return
	_hide_detail_popup()
	await _ensure_use_target_picker()
	if _use_target_picker == null:
		App.notice_received.emit("目标选择弹窗初始化失败，请稍后再试。")
		return
	var pets_loaded: bool = await _fetch_pets_for_use_target_picker()
	if not pets_loaded:
		App.notice_received.emit("加载宠物列表失败，请稍后再试。")
		return
	if GameState.pets.is_empty():
		App.notice_received.emit("当前没有可用宠物。")
		return
	var pick_result: Dictionary = await _prompt_use_target_picker(true, GameState.pets)
	if not bool(pick_result.get("confirmed", false)):
		return
	var target_pet_uid: int = int(pick_result.get("pet_uid", 0))
	if target_pet_uid <= 0:
		App.notice_received.emit("请选择有效的目标宠物。")
		return
	await _submit_use_item_request(item, target_pet_uid, "")


## 装备目标类消耗品：选装备实例后走 USE_ITEM 权威链路。
func _run_use_item_with_equipment_target(item: Dictionary) -> void:
	if _use_item_in_flight:
		return
	_hide_detail_popup()
	await _ensure_use_target_picker()
	if _use_target_picker == null:
		App.notice_received.emit("目标选择弹窗初始化失败，请稍后再试。")
		return
	var equipment_targets: Array = await _fetch_equipment_targets_for_use_picker()
	if equipment_targets.is_empty():
		App.notice_received.emit("当前没有可对目标生效的装备。")
		return
	var pick_result: Dictionary = await _prompt_use_target_picker(false, equipment_targets)
	if not bool(pick_result.get("confirmed", false)):
		return
	var target_item_uid: String = str(pick_result.get("item_uid", "")).strip_edges()
	if target_item_uid.is_empty():
		App.notice_received.emit("请选择有效的目标装备。")
		return
	await _submit_use_item_request(item, 0, target_item_uid)


## 自身目标类消耗品：直接 USE_ITEM 并展示结果。
func _run_generic_use_item_action(item: Dictionary) -> void:
	if _use_item_in_flight:
		return
	_hide_detail_popup()
	await _submit_use_item_request(item, 0, "")


## 发送 USE_ITEM 请求并等待回包，期间展示通用 loading。
func _submit_use_item_request(item: Dictionary, target_pet_uid: int, target_item_uid: String) -> void:
	var slot_index: int = BagUiMapper.slot_index(item)
	if slot_index <= 0:
		App.notice_received.emit("物品格子编号无效，暂时无法操作。")
		return
	if not GameState.is_ws_authenticated:
		App.notice_received.emit("当前未连接服务端，请重新登录后再使用。")
		return
	await _ensure_runtime_popup_ui_ready()
	_use_item_in_flight = true
	_use_item_request_seq = 0
	_connect_use_item_request_handler()
	if _progress_overlay != null:
		_progress_overlay.show_waiting()
	var request_seq: int = App.request_use_item("bag", slot_index, 1, target_pet_uid, target_item_uid)
	if request_seq <= 0:
		_finish_use_item_action(false, {"msg": "使用请求发送失败，请稍后再试。"})
		return
	_use_item_request_seq = request_seq
	if _progress_overlay != null:
		await _progress_overlay.play_progress(USE_ITEM_PROGRESS_DURATION_SEC)
	var waited_frames: int = 0
	while _use_item_in_flight and waited_frames < BOX_OPEN_RESPONSE_TIMEOUT_FRAMES:
		await get_tree().process_frame
		waited_frames += 1
	_disconnect_use_item_request_handler()
	if _use_item_in_flight:
		_finish_use_item_action(false, {"msg": "使用超时，请稍后再试。"})


## 结束 USE_ITEM 流程并提示结果。
func _finish_use_item_action(response_ok: bool, response_payload: Dictionary) -> void:
	_use_item_in_flight = false
	_use_item_waiting_bag_refresh = false
	_use_item_request_seq = 0
	_use_item_response_payload = {}
	if _progress_overlay != null:
		_progress_overlay.hide_progress()
	if not response_ok:
		var error_message: String = str(
			response_payload.get("msg", response_payload.get("message", "物品使用失败，请稍后再试。"))
		)
		if error_message.is_empty():
			error_message = "物品使用失败，请稍后再试。"
		App.notice_received.emit(error_message)
		return
	var result_variant: Variant = response_payload.get("result", {})
	var result: Dictionary = result_variant if result_variant is Dictionary else {}
	App.notice_received.emit(BagUiMapper.build_use_item_success_notice(result))


## 绑定 USE_ITEM 回包监听。
func _connect_use_item_request_handler() -> void:
	_disconnect_use_item_request_handler()
	_use_item_request_handler = Callable(self, "_handle_use_item_request_finished")
	App.request_finished.connect(_use_item_request_handler)


## 断开 USE_ITEM 回包监听。
func _disconnect_use_item_request_handler() -> void:
	if _use_item_request_handler.is_valid() and App.request_finished.is_connected(_use_item_request_handler):
		App.request_finished.disconnect(_use_item_request_handler)
	_use_item_request_handler = Callable()


## 处理 USE_ITEM 回包；礼包打开流程仍由 box 专用逻辑接管。
func _handle_use_item_request_finished(
	request_cmd: int,
	seq: int,
	ok: bool,
	_response_cmd: int,
	payload: Dictionary
) -> void:
	if request_cmd != CommandIds.USE_ITEM_REQ:
		return
	if _box_open_in_flight:
		return
	if not _use_item_in_flight:
		return
	if _use_item_request_seq > 0 and seq != _use_item_request_seq:
		return
	if not ok:
		_finish_use_item_action(false, payload)
		return
	_use_item_waiting_bag_refresh = true
	_use_item_response_payload = payload.duplicate(true)


## USE_ITEM 成功后等待背包最新快照落地，再结束 loading 与提示结果。
func _try_complete_use_item_refresh() -> void:
	if not _use_item_in_flight or not _use_item_waiting_bag_refresh:
		return
	_finish_use_item_action(true, _use_item_response_payload)


## 懒创建消耗品目标选择弹窗。
func _ensure_use_target_picker() -> void:
	if _use_target_picker != null:
		return
	_use_target_picker = USE_ITEM_TARGET_PICKER_SCENE.instantiate() as UseItemTargetPicker
	if _use_target_picker == null:
		return
	add_child(_use_target_picker)
	move_child(_use_target_picker, get_child_count() - 1)
	if not _use_target_picker.is_node_ready():
		await _use_target_picker.ready
	if not _use_target_picker.target_selected.is_connected(_on_use_target_picker_selected):
		_use_target_picker.target_selected.connect(_on_use_target_picker_selected)
	if not _use_target_picker.target_cancelled.is_connected(_on_use_target_picker_cancelled):
		_use_target_picker.target_cancelled.connect(_on_use_target_picker_cancelled)


## 打开目标选择弹窗并等待玩家选择。
func _prompt_use_target_picker(is_pet_mode: bool, entries: Array) -> Dictionary:
	if _use_target_picker == null:
		return {"confirmed": false}
	_use_target_choice_state = 0
	_use_target_choice_result = {}
	if is_pet_mode:
		_use_target_picker.show_pet_targets(entries)
	else:
		_use_target_picker.show_equipment_targets(entries)
	move_child(_use_target_picker, get_child_count() - 1)
	while _use_target_choice_state == 0:
		await get_tree().process_frame
	if _use_target_choice_state != 1:
		return {"confirmed": false}
	return _use_target_choice_result


## 目标选择弹窗确认回调。
func _on_use_target_picker_selected(result: Dictionary) -> void:
	_use_target_choice_result = result if result is Dictionary else {}
	_use_target_choice_state = 1


## 目标选择弹窗取消回调。
func _on_use_target_picker_cancelled() -> void:
	_use_target_choice_state = 2


## 拉取宠物列表供目标选择使用。
func _fetch_pets_for_use_target_picker() -> bool:
	if not GameState.pets.is_empty():
		return true
	if not GameState.is_ws_authenticated:
		return false
	await _ensure_runtime_popup_ui_ready()
	if _progress_overlay != null:
		_progress_overlay.show_waiting()
	var request_seq: int = App.request_pet_list()
	if request_seq <= 0:
		if _progress_overlay != null:
			_progress_overlay.hide_progress()
		return false
	var fetch_ok: bool = await _wait_use_target_fetch_response(request_seq)
	if _progress_overlay != null:
		_progress_overlay.hide_progress()
	return fetch_ok and not GameState.pets.is_empty()


## 拉取背包装备列表并合并已穿戴装备，供目标选择使用。
func _fetch_equipment_targets_for_use_picker() -> Array:
	var cached_targets: Array = BagUiMapper.collect_equipment_use_targets()
	if not cached_targets.is_empty():
		return cached_targets
	if not GameState.is_ws_authenticated:
		return cached_targets
	await _ensure_runtime_popup_ui_ready()
	if _progress_overlay != null:
		_progress_overlay.show_waiting()
	var request_seq: int = App.request_bag_list(1, 200, FILTER_EQUIPMENT)
	if request_seq <= 0:
		if _progress_overlay != null:
			_progress_overlay.hide_progress()
		return BagUiMapper.collect_equipment_use_targets()
	var fetch_ok: bool = await _wait_use_target_fetch_response(request_seq)
	if _progress_overlay != null:
		_progress_overlay.hide_progress()
	if not fetch_ok:
		return BagUiMapper.collect_equipment_use_targets()
	return BagUiMapper.collect_equipment_use_targets()


## 等待 PET_LIST 或 BAG_LIST 回包，供目标选择前置拉取使用。
func _wait_use_target_fetch_response(request_seq: int) -> bool:
	_use_target_fetch_request_seq = request_seq
	_use_target_fetch_ready = false
	_use_target_fetch_ok = false
	_connect_use_target_fetch_handler()
	var waited_frames: int = 0
	while not _use_target_fetch_ready and waited_frames < USE_TARGET_FETCH_TIMEOUT_FRAMES:
		await get_tree().process_frame
		waited_frames += 1
	_disconnect_use_target_fetch_handler()
	return _use_target_fetch_ok


## 绑定目标列表拉取回包监听。
func _connect_use_target_fetch_handler() -> void:
	_disconnect_use_target_fetch_handler()
	_use_target_fetch_handler = Callable(self, "_handle_use_target_fetch_finished")
	App.request_finished.connect(_use_target_fetch_handler)


## 断开目标列表拉取回包监听。
func _disconnect_use_target_fetch_handler() -> void:
	if _use_target_fetch_handler.is_valid() and App.request_finished.is_connected(_use_target_fetch_handler):
		App.request_finished.disconnect(_use_target_fetch_handler)
	_use_target_fetch_handler = Callable()


## 处理 PET_LIST / BAG_LIST 回包，供目标选择前置拉取使用。
func _handle_use_target_fetch_finished(
	request_cmd: int,
	seq: int,
	ok: bool,
	_response_cmd: int,
	_payload: Dictionary
) -> void:
	if _use_target_fetch_ready:
		return
	if seq != _use_target_fetch_request_seq:
		return
	if request_cmd != CommandIds.PET_LIST_REQ and request_cmd != CommandIds.BAG_LIST_REQ:
		return
	_use_target_fetch_ready = true
	_use_target_fetch_ok = ok


## 丢弃物品：专用确认弹窗 + 服务端 DROP_ITEM 权威链路。
func _execute_drop_action(item: Dictionary) -> void:
	if _drop_in_flight:
		return
	var slot_index: int = BagUiMapper.slot_index(item)
	if slot_index <= 0:
		App.notice_received.emit("物品格子编号无效，暂时无法丢弃。")
		return
	if not GameState.is_ws_authenticated:
		App.notice_received.emit("当前未连接服务端，请重新登录后再丢弃。")
		return
	_hide_detail_popup()
	await get_tree().process_frame
	_ensure_drop_popup()
	if _drop_popup == null:
		App.notice_received.emit("丢弃弹窗初始化失败，请稍后再试。")
		return
	var prompt_result: Dictionary = await _drop_popup.prompt_drop(item)
	if not bool(prompt_result.get("confirmed", false)):
		return
	var drop_quantity: int = int(prompt_result.get("quantity", 1))
	if drop_quantity <= 0:
		App.notice_received.emit("丢弃数量无效，请重试。")
		return
	await _ensure_runtime_popup_ui_ready()
	var item_name: String = BagUiMapper.item_name(item)
	_pending_drop_item_name = item_name
	_drop_in_flight = true
	_drop_request_seq = 0
	_drop_snapshot_applied_received = false
	_drop_refresh_response_received = false
	_connect_drop_request_handler()
	if _progress_overlay != null:
		_progress_overlay.show_waiting()
	var item_uid: String = BagUiMapper.item_uid(item)
	var request_seq: int = App.request_drop_item("bag", slot_index, drop_quantity, item_uid)
	if request_seq <= 0:
		_finish_drop_action(false, {"msg": "丢弃请求发送失败，请稍后再试。"})
		return
	_drop_request_seq = request_seq
	var waited_frames: int = 0
	while _drop_in_flight and waited_frames < BOX_OPEN_RESPONSE_TIMEOUT_FRAMES:
		await get_tree().process_frame
		waited_frames += 1
	if _drop_in_flight:
		_finish_drop_action(false, {"msg": "丢弃超时，请稍后再试。"})


## 懒创建丢弃物品确认弹窗。
func _ensure_drop_popup() -> void:
	if _drop_popup != null:
		return
	_drop_popup = DROP_ITEM_POPUP_SCENE.instantiate() as DropItemPopup
	if _drop_popup == null:
		return
	add_child(_drop_popup)
	move_child(_drop_popup, get_child_count() - 1)
	if not _drop_popup.is_node_ready():
		await _drop_popup.ready


## 关闭丢弃确认弹窗。
func _hide_drop_popup() -> void:
	if _drop_popup != null and _drop_popup.visible:
		_drop_popup.force_close_popup()


## 绑定当前丢弃请求回包监听。
func _connect_drop_request_handler() -> void:
	_disconnect_drop_request_handler()
	_drop_request_handler = Callable(self, "_handle_drop_request_finished")
	App.request_finished.connect(_drop_request_handler)


## 断开丢弃请求回包监听。
func _disconnect_drop_request_handler() -> void:
	if _drop_request_handler.is_valid() and App.request_finished.is_connected(_drop_request_handler):
		App.request_finished.disconnect(_drop_request_handler)
	_drop_request_handler = Callable()


## 处理丢弃回包，仅匹配当前 seq。
func _handle_drop_request_finished(
	request_cmd: int,
	seq: int,
	ok: bool,
	_response_cmd: int,
	payload: Dictionary
) -> void:
	if request_cmd == CommandIds.BAG_LIST_REQ and _drop_waiting_bag_refresh:
		if _drop_refresh_request_seq > 0 and seq != _drop_refresh_request_seq:
			return
		if not ok:
			_finish_drop_action(false, payload)
			return
		_drop_refresh_response_received = true
		_try_complete_drop_refresh()
		return
	if request_cmd != CommandIds.DROP_ITEM_REQ or not _drop_in_flight:
		return
	if _drop_request_seq > 0 and seq != _drop_request_seq:
		return
	if not ok:
		_finish_drop_action(false, payload)
		return
	_handle_drop_success_response(payload)


## 丢弃成功回包到达时由 BagController 回调；这里只进入等待新快照阶段，不提前关闭 loading。
func on_drop_item_responded(payload: Dictionary) -> void:
	_handle_drop_success_response(payload)


## 记录丢弃成功回包，并主动补拉当前页；真正完成要等 BagController 写入最新容器快照。
func _handle_drop_success_response(payload: Dictionary) -> void:
	if not _drop_in_flight:
		return
	if _drop_waiting_bag_refresh:
		if not payload.is_empty():
			_drop_response_payload = payload.duplicate(true)
		return
	_drop_response_payload = payload.duplicate(true)
	_drop_waiting_bag_refresh = true
	_drop_refresh_response_received = false
	_drop_snapshot_applied_received = false
	if _progress_overlay != null:
		_progress_overlay.show_waiting()
	_drop_refresh_request_seq = _send_current_bag_page_request()
	if _drop_refresh_request_seq <= 0:
		_finish_drop_action(false, {"msg": "背包刷新请求发送失败，请稍后再试。"})


## 背包完整快照已经写入 GameState；丢弃流程此时才允许结束 loading 和提示成功。
func on_bag_snapshot_applied(container_type: String) -> void:
	if container_type != "bag":
		return
	_try_complete_use_item_refresh()
	_try_complete_repair_refresh()
	if not _drop_waiting_bag_refresh:
		return
	_drop_snapshot_applied_received = true
	_try_complete_drop_refresh()


## 丢弃刷新阶段需同时满足 BAG_LIST 回包与快照写入，避免时序先后不同导致 loading 卡死。
func _try_complete_drop_refresh() -> void:
	if not _drop_in_flight or not _drop_waiting_bag_refresh:
		return
	if not _drop_refresh_response_received and not _drop_snapshot_applied_received:
		return
	_finish_drop_action(true, _drop_response_payload)


## 结束丢弃流程：隐藏 loading，并根据结果提示玩家。
func _finish_drop_action(response_ok: bool, response_payload: Dictionary) -> void:
	if not _drop_in_flight:
		return
	_drop_in_flight = false
	_drop_waiting_bag_refresh = false
	_drop_refresh_request_seq = 0
	_drop_refresh_response_received = false
	_drop_snapshot_applied_received = false
	_drop_request_seq = 0
	_disconnect_drop_request_handler()
	if _progress_overlay != null:
		_progress_overlay.hide_overlay()
	if not response_ok:
		var error_message: String = _resolve_drop_error_message(response_payload)
		_pending_drop_item_name = ""
		_drop_response_payload = {}
		App.notice_received.emit(error_message)
		return
	var resolved_item_name: String = str(response_payload.get("item_name", _pending_drop_item_name)).strip_edges()
	_pending_drop_item_name = ""
	_drop_response_payload = {}
	if resolved_item_name.is_empty():
		resolved_item_name = "物品"
	App.notice_received.emit("已丢弃：%s" % resolved_item_name)
	_selected_slot_index = 0
	_selected_item = {}
	_refresh_panel()


## 将服务端丢弃失败文案转成更易理解的提示。
func _resolve_drop_error_message(response_payload: Dictionary) -> String:
	var raw_message: String = str(
		response_payload.get("msg", response_payload.get("message", ""))
	).strip_edges()
	if raw_message.find("cannot be dropped") >= 0 or raw_message.find("not droppable") >= 0:
		return "该物品不可丢弃，请先在后台开启「可丢弃」并重新拉取背包。"
	if raw_message.find("item slot is empty") >= 0:
		return "该格子已为空，请刷新背包后重试。"
	if raw_message.is_empty():
		return "丢弃失败，请稍后再试。"
	return raw_message


## 礼包类物品：先关详情，再播 3 秒打开进度，最后展示服务端奖励弹窗。
func _execute_box_open_action(item: Dictionary) -> void:
	if _box_open_in_flight:
		return
	var slot_index: int = BagUiMapper.slot_index(item)
	if slot_index <= 0:
		App.notice_received.emit("物品格子编号无效，暂时无法操作。")
		return
	_hide_detail_popup()
	await _ensure_runtime_popup_ui_ready()
	_defer_bag_visual_refresh = true
	_box_open_in_flight = true
	_box_open_presentation_id += 1
	var presentation_id: int = _box_open_presentation_id
	_run_box_open_presentation(slot_index, presentation_id)


## 懒创建装备强化弹窗，并绑定强化请求信号。
func _ensure_enhance_popup() -> void:
	if _enhance_popup != null:
		return
	_enhance_popup = EQUIPMENT_ENHANCE_POPUP_SCENE.instantiate() as EquipmentEnhancePopup
	if _enhance_popup == null:
		return
	add_child(_enhance_popup)
	if not _enhance_popup.enhance_requested.is_connected(_on_enhance_popup_requested):
		_enhance_popup.enhance_requested.connect(_on_enhance_popup_requested)
	if not _enhance_popup.enhance_presentation_finished.is_connected(_on_enhance_presentation_finished):
		_enhance_popup.enhance_presentation_finished.connect(_on_enhance_presentation_finished)
	if not _enhance_popup.popup_closed.is_connected(_on_enhance_popup_closed):
		_enhance_popup.popup_closed.connect(_on_enhance_popup_closed)


## 关闭详情后打开强化弹窗；优先使用背包快照里带 enhance_preview 的最新物品数据。
func _open_enhance_popup(item: Dictionary) -> void:
	if item.is_empty() or not BagUiMapper.is_equipment(item):
		App.notice_received.emit("仅背包装备可强化。")
		return
	var item_uid: String = str(item.get("item_uid", "")).strip_edges()
	if not item_uid.is_empty():
		var refreshed_item: Dictionary = _find_bag_item_by_uid(item_uid)
		if not refreshed_item.is_empty():
			item = refreshed_item
	_hide_detail_popup()
	_ensure_enhance_popup()
	if _enhance_popup == null:
		return
	_enhance_target_item_uid = str(item.get("item_uid", ""))
	_enhance_popup.show_equipment(item)


## 关闭强化弹窗并清理监听。
func _hide_enhance_popup() -> void:
	_disconnect_enhance_request_handler()
	_enhance_target_item_uid = ""
	if _enhance_popup != null and _enhance_popup.visible:
		_enhance_popup.force_close_popup()


## 强化演出结束后同步背包最新快照到弹窗。
func _on_enhance_presentation_finished() -> void:
	_refresh_enhance_popup_if_open()


## 用户点击空白区域关闭强化弹窗时，同步清理本地追踪状态。
func _on_enhance_popup_closed() -> void:
	_disconnect_enhance_request_handler()
	_enhance_target_item_uid = ""


## 背包刷新后，若强化弹窗仍打开则同步最新物品快照。
func _refresh_enhance_popup_if_open() -> void:
	if _enhance_popup == null or not _enhance_popup.visible:
		return
	if _enhance_target_item_uid.is_empty():
		return
	var refreshed_item: Dictionary = _find_bag_item_by_uid(_enhance_target_item_uid)
	if refreshed_item.is_empty():
		_hide_enhance_popup()
		return
	_enhance_popup.refresh_current_item(refreshed_item)


## 按 item_uid 在当前背包快照中查找物品。
func _find_bag_item_by_uid(item_uid: String) -> Dictionary:
	var normalized_uid: String = item_uid.strip_edges()
	if normalized_uid.is_empty():
		return {}
	for item_variant: Variant in GameState.bag_items:
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		if str(item.get("item_uid", "")) == normalized_uid:
			return item.duplicate(true)
	return {}


## 处理强化弹窗内的强化按钮。
func _on_enhance_popup_requested(item: Dictionary, _times: int, _continuous: bool, cost_item_id: int) -> void:
	var item_uid: String = str(item.get("item_uid", "")).strip_edges()
	if item_uid.is_empty():
		App.notice_received.emit("装备实例数据异常，无法强化。请刷新背包或联系管理员处理。")
		if _enhance_popup != null:
			_enhance_popup.notify_enhance_response(false, false)
			_refresh_enhance_popup_if_open()
		return
	_enhance_target_item_uid = item_uid
	_connect_enhance_request_handler()
	var request_seq: int = App.request_player_equipment_enhance(item_uid, cost_item_id)
	if request_seq <= 0:
		_disconnect_enhance_request_handler()
		if _enhance_popup != null:
			_enhance_popup.notify_enhance_response(false, false)
			_refresh_enhance_popup_if_open()
		App.notice_received.emit("强化请求发送失败，请稍后再试。")
		return


## 绑定强化回包监听。
func _connect_enhance_request_handler() -> void:
	_disconnect_enhance_request_handler()
	_enhance_request_handler = Callable(self, "_handle_enhance_request_finished")
	App.request_finished.connect(_enhance_request_handler)


## 断开强化回包监听。
func _disconnect_enhance_request_handler() -> void:
	if _enhance_request_handler.is_valid() and App.request_finished.is_connected(_enhance_request_handler):
		App.request_finished.disconnect(_enhance_request_handler)
	_enhance_request_handler = Callable()


## 处理强化回包并提示结果；背包快照刷新后会自动回写弹窗。
func _handle_enhance_request_finished(
	request_cmd: int,
	_seq: int,
	ok: bool,
	_response_cmd: int,
	payload: Dictionary
) -> void:
	if request_cmd != CommandIds.PLAYER_EQUIPMENT_ENHANCE_REQ:
		return
	if _enhance_popup != null and _enhance_popup.is_enhance_presentation_active():
		var enhance_success: bool = ok and bool(payload.get("success", false))
		var failure_penalty: String = str(payload.get("failure_penalty", "damage"))
		_enhance_popup.notify_enhance_response(ok, enhance_success, failure_penalty, payload)
	else:
		_refresh_enhance_popup_if_open()
	if not ok:
		var error_message: String = str(payload.get("msg", payload.get("message", "强化请求失败。")))
		if error_message.is_empty():
			error_message = "强化请求失败。"
		if _enhance_popup == null or not _enhance_popup.is_enhance_presentation_active():
			App.notice_received.emit(error_message)
		return
	if _enhance_popup == null or not _enhance_popup.is_enhance_presentation_active():
		if bool(payload.get("success", false)):
			var new_level: int = int(payload.get("new_level", 0))
			App.notice_received.emit("强化成功，当前等级 +%s。" % UiFormat.value_to_text(new_level))
		else:
			var failure_penalty: String = str(payload.get("failure_penalty", "damage"))
			match failure_penalty:
				"level_down":
					App.notice_received.emit("强化失败，等级降低，材料已消耗。")
				"none":
					App.notice_received.emit("强化失败，材料已消耗。")
				_:
					App.notice_received.emit("强化失败，装备已损坏，材料已消耗。")


## 懒创建装备修复确认弹窗。
func _ensure_repair_popup() -> void:
	if _repair_popup != null:
		return
	_repair_popup = REPAIR_EQUIPMENT_POPUP_SCENE.instantiate() as RepairEquipmentPopup
	if _repair_popup == null:
		return
	add_child(_repair_popup)
	move_child(_repair_popup, get_child_count() - 1)
	if not _repair_popup.is_node_ready():
		await _repair_popup.ready


## 关闭装备修复确认弹窗。
func _hide_repair_popup() -> void:
	if _repair_popup != null and _repair_popup.visible:
		_repair_popup.force_close_popup()


## 损坏装备修复：专用确认弹窗 + 服务端 REPAIR 权威链路。
func _execute_repair_action(item: Dictionary) -> void:
	if _repair_in_flight:
		return
	if not BagUiMapper.is_equipment(item) or not BagUiMapper.is_damaged(item):
		return
	var item_uid: String = BagUiMapper.item_uid(item)
	if item_uid.is_empty():
		App.notice_received.emit("装备实例标识缺失，暂时无法修复。")
		return
	if not GameState.is_ws_authenticated:
		App.notice_received.emit("当前未连接服务端，请重新登录后再修复。")
		return
	_hide_detail_popup()
	await get_tree().process_frame
	_ensure_repair_popup()
	if _repair_popup == null:
		App.notice_received.emit("修复弹窗初始化失败，请稍后再试。")
		return
	var prompt_result: Dictionary = await _repair_popup.prompt_repair(item)
	if not bool(prompt_result.get("confirmed", false)):
		return
	if not BagUiMapper.supports_repair_action(item):
		App.notice_received.emit("修复宝石不足，无法修复。")
		return
	await _ensure_runtime_popup_ui_ready()
	_repair_in_flight = true
	_repair_request_seq = 0
	_connect_repair_request_handler()
	if _progress_overlay != null:
		_progress_overlay.show_waiting()
	var request_seq: int = App.request_player_equipment_repair(item_uid)
	if request_seq <= 0:
		_finish_repair_action(false, {"msg": "修复请求发送失败，请稍后再试。"})
		return
	_repair_request_seq = request_seq
	var waited_frames: int = 0
	while _repair_in_flight and waited_frames < BOX_OPEN_RESPONSE_TIMEOUT_FRAMES:
		await get_tree().process_frame
		waited_frames += 1
	if _repair_in_flight:
		_finish_repair_action(false, {"msg": "修复超时，请稍后再试。"})


## 绑定当前修复请求回包监听。
func _connect_repair_request_handler() -> void:
	_disconnect_repair_request_handler()
	_repair_request_handler = Callable(self, "_handle_repair_request_finished")
	App.request_finished.connect(_repair_request_handler)


## 断开修复请求回包监听。
func _disconnect_repair_request_handler() -> void:
	if _repair_request_handler.is_valid() and App.request_finished.is_connected(_repair_request_handler):
		App.request_finished.disconnect(_repair_request_handler)
	_repair_request_handler = Callable()


## 处理修复回包，仅匹配当前 seq。
func _handle_repair_request_finished(
	request_cmd: int,
	seq: int,
	ok: bool,
	_response_cmd: int,
	payload: Dictionary
) -> void:
	if request_cmd != CommandIds.PLAYER_EQUIPMENT_REPAIR_REQ or not _repair_in_flight:
		return
	if _repair_request_seq > 0 and seq != _repair_request_seq:
		return
	if not ok:
		_finish_repair_action(false, payload)
		return
	_repair_waiting_bag_refresh = true
	_repair_response_payload = payload.duplicate(true)


## 结束修复流程：隐藏 loading，并根据结果提示玩家。
func _finish_repair_action(response_ok: bool, response_payload: Dictionary) -> void:
	if not _repair_in_flight:
		return
	_repair_in_flight = false
	_repair_waiting_bag_refresh = false
	_repair_request_seq = 0
	_repair_response_payload = {}
	_disconnect_repair_request_handler()
	if _progress_overlay != null:
		_progress_overlay.hide_overlay()
	if not response_ok:
		var error_message: String = str(
			response_payload.get("msg", response_payload.get("message", "修复失败，请稍后再试。"))
		).strip_edges()
		if error_message.is_empty():
			error_message = "修复失败，请稍后再试。"
		App.notice_received.emit(error_message)
		return
	var item_variant: Variant = response_payload.get("item", {})
	var item_name: String = ""
	if item_variant is Dictionary:
		item_name = str((item_variant as Dictionary).get("item_name", "")).strip_edges()
	if item_name.is_empty():
		item_name = "装备"
	App.notice_received.emit("已修复：%s" % item_name)
	_selected_slot_index = 0
	_selected_item = {}
	_refresh_panel()


## 修复成功后等待背包快照刷新完成，再结束 loading 和提示结果。
func _try_complete_repair_refresh() -> void:
	if not _repair_in_flight or not _repair_waiting_bag_refresh:
		return
	_finish_repair_action(true, _repair_response_payload)


## 懒创建通用进度遮罩与奖励弹窗，并等待 ready。
func _ensure_runtime_popup_ui_ready() -> void:
	_ensure_runtime_popup_ui()
	if _progress_overlay != null and not _progress_overlay.is_node_ready():
		await _progress_overlay.ready
	if _box_open_progress_overlay != null and not _box_open_progress_overlay.is_node_ready():
		await _box_open_progress_overlay.ready
	if _reward_popup != null and not _reward_popup.is_node_ready():
		await _reward_popup.ready


## 懒创建通用 UI 弹层，挂到背包面板下统一管理。
func _ensure_runtime_popup_ui() -> void:
	if _progress_overlay == null:
		_progress_overlay = RUNTIME_PROGRESS_OVERLAY_SCENE.instantiate() as RuntimeProgressOverlay
		if _progress_overlay != null:
			add_child(_progress_overlay)
	if _box_open_progress_overlay == null:
		_box_open_progress_overlay = RUNTIME_PROGRESS_BAR_OVERLAY_SCENE.instantiate() as RuntimeProgressBarOverlay
		if _box_open_progress_overlay != null:
			add_child(_box_open_progress_overlay)
	if _reward_popup == null:
		_reward_popup = REWARD_POPUP_SCENE.instantiate() as RewardPopup
		if _reward_popup != null:
			add_child(_reward_popup)


## 先监听回包再发 USE_ITEM，再播进度条；两者都完成后再弹奖励窗。
func _run_box_open_presentation(slot_index: int, presentation_id: int) -> void:
	_box_open_response_ready = false
	_box_open_response_ok = false
	_box_open_response_payload = {}
	_box_open_request_seq = 0
	_connect_box_open_request_handler()
	var request_seq: int = App.request_use_item("bag", slot_index, 1)
	if request_seq <= 0:
		_disconnect_box_open_request_handler()
		_box_open_in_flight = false
		_end_box_open_deferred_refresh(true)
		App.notice_received.emit("打开请求发送失败，请稍后再试。")
		return
	if not _box_open_response_ready:
		_box_open_request_seq = request_seq
	if _box_open_progress_overlay != null:
		await _box_open_progress_overlay.play_progress(BOX_OPEN_PROGRESS_DURATION_SEC, BOX_OPEN_PROGRESS_STATUS_TEXT)
	var waited_frames: int = 0
	while not _box_open_response_ready and waited_frames < BOX_OPEN_RESPONSE_TIMEOUT_FRAMES:
		await get_tree().process_frame
		waited_frames += 1
	_disconnect_box_open_request_handler()
	if presentation_id != _box_open_presentation_id:
		return
	if not _box_open_response_ready:
		_finish_box_open_presentation(false, {"msg": "打开超时，请稍后再试。"})
		return
	_finish_box_open_presentation(_box_open_response_ok, _box_open_response_payload)


## 绑定当前礼包 USE_ITEM 回包监听。
func _connect_box_open_request_handler() -> void:
	_disconnect_box_open_request_handler()
	_box_open_request_handler = Callable(self, "_handle_box_open_request_finished")
	App.request_finished.connect(_box_open_request_handler)


## 断开礼包 USE_ITEM 回包监听。
func _disconnect_box_open_request_handler() -> void:
	if _box_open_request_handler.is_valid() and App.request_finished.is_connected(_box_open_request_handler):
		App.request_finished.disconnect(_box_open_request_handler)
	_box_open_request_handler = Callable()


## 处理礼包 USE_ITEM 回包，仅匹配当前 seq。
func _handle_box_open_request_finished(
	request_cmd: int,
	seq: int,
	ok: bool,
	_response_cmd: int,
	payload: Dictionary
) -> void:
	if request_cmd != CommandIds.USE_ITEM_REQ or not _box_open_in_flight:
		return
	if _box_open_response_ready:
		return
	if _box_open_request_seq > 0 and seq != _box_open_request_seq:
		return
	_box_open_request_seq = seq
	_box_open_response_ready = true
	_box_open_response_ok = ok
	_box_open_response_payload = payload


## 结束礼包打开演出：隐藏进度层，成功时弹出奖励，失败时提示错误。
func _finish_box_open_presentation(response_ok: bool, response_payload: Dictionary) -> void:
	_box_open_in_flight = false
	_box_open_request_seq = 0
	if _box_open_progress_overlay != null:
		_box_open_progress_overlay.hide_progress()
	if not response_ok:
		var error_message: String = str(
			response_payload.get("msg", response_payload.get("message", "打开失败，请稍后再试。"))
		)
		if error_message.is_empty():
			error_message = "打开失败，请稍后再试。"
		App.notice_received.emit(error_message)
		_end_box_open_deferred_refresh(true)
		return
	var result_variant: Variant = response_payload.get("result", {})
	var result: Dictionary = result_variant if result_variant is Dictionary else {}
	var rewards_variant: Variant = result.get("rewards", [])
	var rewards: Array = rewards_variant if rewards_variant is Array else []
	_show_box_reward_and_refresh_when_closed(rewards)


## 展示礼包成功弹窗，并在玩家关闭弹窗后再刷新背包列表。
func _show_box_reward_and_refresh_when_closed(rewards: Array) -> void:
	if _reward_popup == null:
		_end_box_open_deferred_refresh(true)
		return
	_reward_popup.show_rewards("", rewards, [])
	if not _reward_popup.visible:
		_end_box_open_deferred_refresh(true)
		return
	await _reward_popup.popup_closed
	_end_box_open_deferred_refresh(true)


## 结束礼包演出的背包 UI 暂缓刷新；force_refresh 为 true 时立即重绘当前页。
func _end_box_open_deferred_refresh(force_refresh: bool) -> void:
	_defer_bag_visual_refresh = false
	if force_refresh and visible:
		_refresh_panel()


## 强制关闭奖励弹窗（例如玩家直接关背包时）。
func _force_close_reward_popup() -> void:
	if _reward_popup == null or not _reward_popup.visible:
		return
	_reward_popup.force_close_popup()


## 取消进行中的礼包打开演出（例如关闭背包时）。
func _cancel_box_open_presentation() -> void:
	if not _box_open_in_flight and not _defer_bag_visual_refresh:
		return
	_box_open_presentation_id += 1
	_box_open_in_flight = false
	_box_open_request_seq = 0
	_disconnect_box_open_request_handler()
	if _box_open_progress_overlay != null:
		_box_open_progress_overlay.hide_progress()
	if _defer_bag_visual_refresh:
		_end_box_open_deferred_refresh(true)


## 取消进行中的普通消耗品使用流程（例如关闭背包时）。
func _cancel_use_item_presentation() -> void:
	if not _use_item_in_flight:
		return
	_use_item_in_flight = false
	_use_item_request_seq = 0
	_disconnect_use_item_request_handler()
	if _progress_overlay != null:
		_progress_overlay.hide_progress()
	if _use_target_picker != null and _use_target_picker.visible:
		_use_target_picker.force_close_popup()
		_use_target_choice_state = 2
