extends RuntimeRootPanel
class_name BagPanel

const SLOTS_PER_PAGE: int = 28
const DEFAULT_CAPACITY: int = 300
const FILTER_ALL: String = "all"
const FILTER_EQUIPMENT: String = "equipment"
const FILTER_OTHER: String = "other"
const BAG_ITEM_DETAIL_SCENE: PackedScene = preload("res://scenes/ui/bag/bag_item_detail.tscn")

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


## 打开背包时向服务端拉取最新背包快照，确保展示使用权威数据。
func open_menu() -> void:
	super.open_menu()
	_current_filter = FILTER_ALL
	_current_page = 0
	_sync_filter_buttons()
	_request_current_bag_page()
	## 装备槽展示依赖独立的人物装备列表协议；仅刷新背包不会自动带回已穿戴装备。
	## 因此每次打开背包时都补拉一次服务端权威装备列表，避免后台已穿戴但客户端槽位仍显示为空。
	if GameState.is_ws_authenticated:
		App.request_player_equipment_list()
	_refresh_panel()


## 关闭背包并同步关闭详情弹层，避免旧选中状态残留到下一次打开。
func close_menu() -> void:
	_hide_detail_popup()
	super.close_menu()


## 关闭背包内所有 overlay（详情层）。
func _close_all_overlays() -> void:
	_hide_detail_popup()


## 点根面板遮罩时若详情已打开，仅关闭详情层。
func _dismiss_top_overlay() -> bool:
	if _detail_overlay != null and _detail_overlay.visible:
		_hide_detail_popup()
		return true
	return false


## 背包或钱包快照变化后，只在面板可见时刷新当前页内容。
func _on_bag_data_changed() -> void:
	if visible:
		_refresh_panel()


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
	_show_item_detail(_selected_item)


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
	_show_equipped_item_detail(slot.get_equipment())


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

	var detail_center: CenterContainer = CenterContainer.new()
	detail_center.name = "DetailCenter"
	detail_center.set_anchors_preset(Control.PRESET_FULL_RECT)
	detail_center.offset_left = 0.0
	detail_center.offset_top = 0.0
	detail_center.offset_right = 0.0
	detail_center.offset_bottom = 0.0
	detail_center.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_detail_overlay.add_child(detail_center)

    _detail_panel = BAG_ITEM_DETAIL_SCENE.instantiate() as BagItemDetail
    if _detail_panel == null:
        return
    _detail_panel.mouse_filter = Control.MOUSE_FILTER_STOP
	detail_center.add_child(_detail_panel)
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


## 显示当前选中物品详情，并把弹层尽量放在背包面板中心附近，适配移动端单手操作。
func _show_item_detail(item: Dictionary) -> void:
	if item.is_empty():
		return
	_selected_equip_slot_key = ""
	_ensure_detail_overlay()
	if _detail_overlay == null or _detail_panel == null:
		return
	_detail_panel.set_item(item)
	_open_detail_popup()


## 显示已穿戴装备详情，并提供卸下/分享操作入口。
func _show_equipped_item_detail(item: Dictionary) -> void:
	if item.is_empty():
		return
	_ensure_detail_overlay()
	if _detail_overlay == null or _detail_panel == null:
		return
	_detail_panel.set_equipped_item(item)
	_open_detail_popup()


## 打开详情 overlay 并置于当前面板最顶层。
func _open_detail_popup() -> void:
	if _detail_overlay == null:
		return
	_detail_overlay.show()
	move_child(_detail_overlay, get_child_count() - 1)


## 关闭详情 overlay 并清空当前选中物品，避免旧高亮残留到下一次交互。
func _hide_detail_popup() -> void:
	if _detail_overlay != null:
		_detail_overlay.hide()
	if _detail_panel != null:
		_detail_panel.clear_item()
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
	if not GameState.is_ws_authenticated:
		return
	App.request_bag_list(_current_page + 1, SLOTS_PER_PAGE, _current_filter)


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
			App.notice_received.emit("丢弃功能尚未接入新版背包。")
		"give":
			App.notice_received.emit("给人功能尚未接入新版背包。")
		"share":
			App.notice_received.emit("分享功能尚未接入新版背包。")
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
	App.notice_received.emit("已发送卸下请求，请等待服务端返回最新装备数据。")
	_hide_detail_popup()


## 按物品类型把“主操作”分发到对应服务端权威接口。
func _execute_primary_item_action(item: Dictionary) -> void:
	var slot_index: int = BagUiMapper.slot_index(item)
	if slot_index <= 0:
		App.notice_received.emit("物品格子编号无效，暂时无法操作。")
		return
	if BagUiMapper.requires_pet_target(item):
		App.notice_received.emit("该道具需要先选择目标宠物，新版背包暂未接入该流程。")
		return
	if BagUiMapper.is_equipment(item):
		App.request_player_equip(slot_index, "bag")
		App.notice_received.emit("已发送装备请求，请等待服务端返回最新装备数据。")
	else:
		App.request_use_item("bag", slot_index, 1)
	_hide_detail_popup()
