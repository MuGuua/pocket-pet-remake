extends PanelContainer

signal transfer_requested(source_container_type: String, slot_index: int, quantity: int)
signal container_switch_requested(target_container_type: String)

const INVENTORY_CAPACITY: int = 30
const GRID_PAGE_SIZE: int = 32
const RequestLoadingOverlay = preload("res://scripts/ui/request_loading_overlay.gd")

@onready var title_label: Label = $VBoxContainer/HBoxContainer/ColorRect/Label
@onready var capacity_label: Label = $VBoxContainer/HBoxContainer2/PanelContainer/HBoxContainer/Label
@onready var gold_label: Label = $VBoxContainer/HBoxContainer2/PanelContainer2/HBoxContainer/Label
@onready var distinct_count_label: Label = $VBoxContainer/HBoxContainer2/PanelContainer2/HBoxContainer/Label3
@onready var page_label: Label = $VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer/HBoxContainer/PanelContainer/Label
@onready var grid_container: GridContainer = $VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer/GridContainer

var _items_summary_label: Label
var _action_container: VBoxContainer
var _switch_button: Button
var _quantity_dialog: ConfirmationDialog
var _quantity_spin_box: SpinBox
var _pending_transfer_container_type: String = "bag"
var _pending_transfer_slot_index: int = 0
var _pending_transfer_max_quantity: int = 0
var _container_type: String = "bag"
var _panel_title: String = "背包"
var _warehouse_available: bool = false
## 通用 loading 遮罩。
var _request_loading: RequestLoadingOverlay = null
## 正在等待服务端回包的请求序列号。
var _loading_request_seq: int = 0
## 宠物选择对话框。
var _pet_pick_dialog: AcceptDialog = null
## 宠物选择下拉框。
var _pet_pick_option: OptionButton = null
## 待使用物品的背包槽位。
var _pending_use_slot_index: int = 0
## 法宝装备对话框。
var _artifact_equip_dialog: AcceptDialog = null
## 法宝装备时的宠物选择下拉框。
var _artifact_pet_option: OptionButton = null
## 法宝装备时的槽位选择下拉框。
var _artifact_slot_option: OptionButton = null
## 待装备的背包槽位（法宝）。
var _pending_equip_bag_slot_index: int = 0


func _ready() -> void:
	# 追加一个运行时文本区域，用最小改动把服务端背包列表摘要展示出来。
	_items_summary_label = Label.new()
	_items_summary_label.name = "ItemsSummaryLabel"
	_items_summary_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	_items_summary_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_LEFT
	$VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer.add_child(_items_summary_label)
	$VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer.move_child(_items_summary_label, 1)

	_action_container = VBoxContainer.new()
	_action_container.name = "ActionContainer"
	_action_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	$VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer.add_child(_action_container)

	_switch_button = Button.new()
	_switch_button.name = "SwitchContainerButton"
	_switch_button.custom_minimum_size = Vector2(0, 28)
	_switch_button.pressed.connect(_on_switch_button_pressed)
	_action_container.add_child(_switch_button)

	_build_quantity_dialog()
	_build_pet_pick_dialog()
	_build_artifact_equip_dialog()
	_build_request_loading()

	if not GameState.bag_changed.is_connected(refresh_panel_data):
		GameState.bag_changed.connect(refresh_panel_data)
	if not GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.connect(refresh_panel_data)
	refresh_panel_data()


func set_container_context(container_type: String, panel_title: String = "") -> void:
	_container_type = container_type if not container_type.is_empty() else "bag"
	_panel_title = panel_title if not panel_title.is_empty() else ("仓库" if _container_type == "warehouse" else "背包")
	if title_label != null:
		title_label.text = _panel_title
	refresh_panel_data()


func set_warehouse_available(available: bool) -> void:
	_warehouse_available = available
	refresh_panel_data()


func _exit_tree() -> void:
	# 面板销毁时及时断开全局状态信号，避免悬空回调。
	if GameState.bag_changed.is_connected(refresh_panel_data):
		GameState.bag_changed.disconnect(refresh_panel_data)
	if GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.disconnect(refresh_panel_data)


func refresh_panel_data() -> void:
	var container: Dictionary = GameState.warehouse_container if _container_type == "warehouse" else GameState.bag_container
	var items_variant: Variant = container.get("items", GameState.bag_items if _container_type == "bag" else [])
	var items: Array = items_variant if items_variant is Array else []
	var total_stack_count: int = 0
	for item_variant in items:
		if item_variant is Dictionary:
			total_stack_count += int(item_variant.get("quantity", item_variant.get("count", 0)))

	if title_label != null:
		title_label.text = _panel_title
	var capacity: int = int(container.get("capacity", INVENTORY_CAPACITY))
	capacity_label.text = "%d/%d" % [items.size(), capacity]
	gold_label.text = _format_wallet_text(GameState.wallet_snapshot)
	distinct_count_label.text = UiFormat.value_to_text(total_stack_count)
	page_label.text = UiFormat.normalize_text("1/%d" % maxi(1, int(ceil(float(max(items.size(), 1)) / float(GRID_PAGE_SIZE)))))
	_items_summary_label.text = _build_items_summary(items)
	_refresh_action_buttons(items)
	_apply_grid_tooltips(items)


func _build_items_summary(items: Array) -> String:
	if items.is_empty():
		return "服务端%s为空，等待后续获得物品。" % _panel_title

	var lines: Array[String] = []
	var max_lines := mini(items.size(), 6)
	for index in range(max_lines):
		var item_variant: Variant = items[index]
		if item_variant is Dictionary:
			var item: Dictionary = item_variant
			lines.append(
				"槽位%d %s x%d" % [
					int(item.get("slot_index", index + 1)),
					str(item.get("item_name", "物品ID %d" % int(item.get("item_id", 0)))),
					int(item.get("quantity", item.get("count", 0))),
				]
			)
	if items.size() > max_lines:
		lines.append("...... 其余 %d 个物品槽请以后续背包交互页为准" % (items.size() - max_lines))
	return UiFormat.normalize_text("\n".join(lines))


func _format_wallet_text(snapshot: Dictionary) -> String:
	if snapshot.is_empty():
		return UiFormat.value_to_text(GameState.player_snapshot.get("gold", 0))
	return UiFormat.normalize_text("%d金 %d银 %d铜" % [
		int(snapshot.get("gold", 0)),
		int(snapshot.get("silver", 0)),
		int(snapshot.get("copper", 0)),
	])


func _apply_grid_tooltips(items: Array) -> void:
	# 旧格子资源没有完整的物品渲染链路，这里先把真实物品信息写入 tooltip，
	# 至少能确认每个格子对应的是服务端返回的哪条数据。
	for index in range(grid_container.get_child_count()):
		var cell := grid_container.get_child(index) as Control
		if cell == null:
			continue
		if index >= items.size():
			cell.tooltip_text = "空槽位"
			continue
		var item_variant: Variant = items[index]
		if item_variant is Dictionary:
			var item: Dictionary = item_variant
			cell.tooltip_text = UiFormat.normalize_text("slot=%d\nitem_id=%d\nquantity=%d\nitem_uid=%s" % [
				int(item.get("slot_index", index + 1)),
					int(item.get("item_id", 0)),
					int(item.get("quantity", item.get("count", 0))),
					str(item.get("item_uid", "")),
				]
			)
		else:
			cell.tooltip_text = "槽位数据格式异常"


func _refresh_action_buttons(items: Array) -> void:
	if _action_container == null:
		return

	for index in range(_action_container.get_child_count() - 1, 0, -1):
		var child := _action_container.get_child(index)
		_action_container.remove_child(child)
		child.queue_free()

	if _switch_button != null:
		_switch_button.visible = _warehouse_available
		_switch_button.text = "切换到仓库" if _container_type == "bag" else "切换到背包"

	if items.is_empty():
		var empty_label := Label.new()
		empty_label.text = "当前容器为空。"
		_action_container.add_child(empty_label)
		return

	var hint_label := Label.new()
	hint_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	hint_label.text = "点击下方条目可%s整格物品。" % ("存入仓库" if _container_type == "bag" else "取回背包")
	_action_container.add_child(hint_label)

	var max_buttons := mini(items.size(), 8)
	for index in range(max_buttons):
		var item_variant: Variant = items[index]
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant
		var slot_index: int = int(item.get("slot_index", index + 1))
		var item_name: String = str(item.get("item_name", "物品ID %d" % int(item.get("item_id", 0))))
		var quantity: int = int(item.get("quantity", item.get("count", 0)))

		var row: HBoxContainer = HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)

		var transfer_button := Button.new()
		transfer_button.custom_minimum_size = Vector2(0, 28)
		transfer_button.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		transfer_button.alignment = HORIZONTAL_ALIGNMENT_LEFT
		transfer_button.text = "%s 槽位%d %s x%d" % [
			"存入" if _container_type == "bag" else "取回",
			slot_index,
			item_name,
			quantity,
		]
		transfer_button.pressed.connect(_on_transfer_button_pressed.bind(slot_index, quantity))
		row.add_child(transfer_button)

		if _is_artifact_item(item):
			var equip_button := Button.new()
			equip_button.custom_minimum_size = Vector2(56, 28)
			equip_button.text = "装备"
			equip_button.pressed.connect(_on_artifact_equip_button_pressed.bind(slot_index))
			row.add_child(equip_button)
		elif _is_player_equipment_item(item):
			var wear_button := Button.new()
			wear_button.custom_minimum_size = Vector2(56, 28)
			wear_button.text = "穿戴"
			wear_button.pressed.connect(_on_player_equip_button_pressed.bind(slot_index))
			row.add_child(wear_button)
			if _can_enhance_bag_equipment(item):
				var enhance_button := Button.new()
				enhance_button.custom_minimum_size = Vector2(56, 28)
				enhance_button.text = "强化"
				var item_uid: String = str(item.get("item_uid", ""))
				enhance_button.pressed.connect(_on_player_enhance_button_pressed.bind(item_uid))
				row.add_child(enhance_button)
		elif _can_use_item(item):
			var use_button := Button.new()
			use_button.custom_minimum_size = Vector2(56, 28)
			use_button.text = "使用"
			use_button.pressed.connect(_on_use_button_pressed.bind(slot_index, item))
			row.add_child(use_button)

		_action_container.add_child(row)

	if items.size() > max_buttons:
		var more_label := Label.new()
		more_label.text = "仅展示前 %d 个可操作条目。" % max_buttons
		_action_container.add_child(more_label)


func _on_switch_button_pressed() -> void:
	if not _warehouse_available:
		return
	container_switch_requested.emit("warehouse" if _container_type == "bag" else "bag")


func _on_transfer_button_pressed(slot_index: int, quantity: int) -> void:
	if quantity <= 0:
		return
	if quantity == 1:
		transfer_requested.emit(_container_type, slot_index, quantity)
		return

	_pending_transfer_container_type = _container_type
	_pending_transfer_slot_index = slot_index
	_pending_transfer_max_quantity = quantity
	if _quantity_spin_box != null:
		_quantity_spin_box.min_value = 1
		_quantity_spin_box.max_value = quantity
		_quantity_spin_box.step = 1
		_quantity_spin_box.value = quantity
	if _quantity_dialog != null:
		_quantity_dialog.dialog_text = "请选择本次要%s的数量。" % ("存入仓库" if _container_type == "bag" else "取回背包")
		_quantity_dialog.popup_centered(Vector2i(240, 120))


func _build_quantity_dialog() -> void:
	_quantity_dialog = ConfirmationDialog.new()
	_quantity_dialog.title = "选择数量"
	_quantity_dialog.ok_button_text = "确认"
	_quantity_dialog.cancel_button_text = "取消"
	_quantity_dialog.confirmed.connect(_on_quantity_dialog_confirmed)
	add_child(_quantity_dialog)

	var dialog_container := VBoxContainer.new()
	dialog_container.custom_minimum_size = Vector2(160, 48)
	_quantity_dialog.add_child(dialog_container)

	var hint_label := Label.new()
	hint_label.text = "数量"
	dialog_container.add_child(hint_label)

	_quantity_spin_box = SpinBox.new()
	_quantity_spin_box.min_value = 1
	_quantity_spin_box.max_value = 1
	_quantity_spin_box.step = 1
	_quantity_spin_box.rounded = true
	_quantity_spin_box.allow_greater = false
	_quantity_spin_box.allow_lesser = false
	dialog_container.add_child(_quantity_spin_box)


func _on_quantity_dialog_confirmed() -> void:
	if _pending_transfer_slot_index <= 0 or _pending_transfer_max_quantity <= 0:
		return
	var quantity: int = clampi(int(_quantity_spin_box.value), 1, _pending_transfer_max_quantity)
	transfer_requested.emit(_pending_transfer_container_type, _pending_transfer_slot_index, quantity)


## 判断是否为人物装备类物品（走 2072 佩戴协议）。
func _is_player_equipment_item(item: Dictionary) -> bool:
	if _container_type != "bag":
		return false
	return str(item.get("item_type", "")) == "equipment"


## 判断背包内人物装备是否可强化（须已有 item_uid 且未佩戴）。
func _can_enhance_bag_equipment(item: Dictionary) -> bool:
	if not _is_player_equipment_item(item):
		return false
	var item_uid: String = str(item.get("item_uid", ""))
	if item_uid.is_empty():
		return false
	return int(item.get("quantity", item.get("count", 0))) == 1


## 判断是否为法宝镶嵌类物品（走 3031 装备协议，不走 5021）。
func _is_artifact_item(item: Dictionary) -> bool:
	if _container_type != "bag":
		return false
	return str(item.get("effect_type", "")) == "pet_artifact"


## 判断背包物品是否可在当前容器中使用。
func _can_use_item(item: Dictionary) -> bool:
	if _container_type != "bag":
		return false
	if _is_artifact_item(item):
		return false
	if not bool(item.get("usable", false)):
		return false
	var target_type: String = str(item.get("target_type", ""))
	if target_type.is_empty():
		return false
	return true


## 构建通用 loading 遮罩。
func _build_request_loading() -> void:
	_request_loading = RequestLoadingOverlay.new()
	_request_loading.name = "BagUseLoadingOverlay"
	add_child(_request_loading)


## 构建选择目标宠物的对话框。
func _build_pet_pick_dialog() -> void:
	_pet_pick_dialog = AcceptDialog.new()
	_pet_pick_dialog.title = "选择目标宠物"
	_pet_pick_dialog.ok_button_text = "确认使用"
	_pet_pick_dialog.confirmed.connect(_on_pet_pick_confirmed)
	add_child(_pet_pick_dialog)

	var dialog_vbox := VBoxContainer.new()
	_pet_pick_dialog.add_child(dialog_vbox)

	var hint_label := Label.new()
	hint_label.text = "请选择要对该道具生效的宠物："
	dialog_vbox.add_child(hint_label)

	_pet_pick_option = OptionButton.new()
	_pet_pick_option.custom_minimum_size = Vector2(220, 32)
	dialog_vbox.add_child(_pet_pick_option)


## 构建法宝装备对话框（选宠物 + 选法宝槽位）。
func _build_artifact_equip_dialog() -> void:
	_artifact_equip_dialog = AcceptDialog.new()
	_artifact_equip_dialog.title = "装备法宝"
	_artifact_equip_dialog.ok_button_text = "确认装备"
	_artifact_equip_dialog.confirmed.connect(_on_artifact_equip_confirmed)
	add_child(_artifact_equip_dialog)

	var dialog_vbox := VBoxContainer.new()
	_artifact_equip_dialog.add_child(dialog_vbox)

	var pet_hint := Label.new()
	pet_hint.text = "选择目标宠物："
	dialog_vbox.add_child(pet_hint)

	_artifact_pet_option = OptionButton.new()
	_artifact_pet_option.custom_minimum_size = Vector2(220, 32)
	dialog_vbox.add_child(_artifact_pet_option)

	var slot_hint := Label.new()
	slot_hint.text = "选择法宝槽位："
	dialog_vbox.add_child(slot_hint)

	_artifact_slot_option = OptionButton.new()
	_artifact_slot_option.custom_minimum_size = Vector2(220, 32)
	_artifact_slot_option.add_item("法宝槽 1", 0)
	_artifact_slot_option.add_item("法宝槽 2", 1)
	_artifact_slot_option.add_item("法宝槽 3", 2)
	dialog_vbox.add_child(_artifact_slot_option)


## 点击法宝装备按钮，弹出宠物与槽位选择。
func _on_artifact_equip_button_pressed(bag_slot_index: int) -> void:
	if _loading_request_seq != 0 or bag_slot_index <= 0:
		return
	if not GameState.is_ws_authenticated:
		return
	_pending_equip_bag_slot_index = bag_slot_index
	_rebuild_pet_pick_options_for(_artifact_pet_option)
	if _artifact_pet_option == null or _artifact_pet_option.item_count == 0:
		return
	if _artifact_equip_dialog != null:
		_artifact_equip_dialog.popup_centered(Vector2i(300, 180))


## 确认法宝装备后发起 3031 请求。
func _on_artifact_equip_confirmed() -> void:
	if _artifact_pet_option == null or _artifact_slot_option == null:
		return
	if _pending_equip_bag_slot_index <= 0:
		return
	var pet_index: int = _artifact_pet_option.get_selected()
	var slot_index: int = _artifact_slot_option.get_selected()
	if pet_index < 0 or slot_index < 0:
		return
	var target_pet_uid: int = int(_artifact_pet_option.get_item_id(pet_index))
	var artifact_slot_index: int = int(_artifact_slot_option.get_item_id(slot_index))
	_request_artifact_equip(target_pet_uid, artifact_slot_index, _pending_equip_bag_slot_index)
	_pending_equip_bag_slot_index = 0


## 点击人物装备穿戴按钮，直接向服务端发起佩戴请求。
func _on_player_equip_button_pressed(bag_slot_index: int) -> void:
	if _loading_request_seq != 0 or bag_slot_index <= 0:
		return
	if not GameState.is_ws_authenticated:
		return
	_request_player_equip(bag_slot_index)


## 向服务端发送人物装备佩戴请求并等待回包。
func _request_player_equip(bag_slot_index: int) -> void:
	if _loading_request_seq != 0:
		return
	var request_seq: int = App.request_player_equip(bag_slot_index, _container_type)
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	if _request_loading != null:
		_request_loading.show_waiting("正在穿戴装备")
	call_deferred("_wait_player_equip_request", request_seq)


## 点击背包内未佩戴人物装备的强化按钮。
func _on_player_enhance_button_pressed(item_uid: String) -> void:
	if item_uid.is_empty() or _loading_request_seq != 0:
		return
	if not GameState.is_ws_authenticated:
		return
	_request_player_equipment_enhance(item_uid)


## 向服务端发送人物装备强化请求并等待回包。
func _request_player_equipment_enhance(item_uid: String) -> void:
	if _loading_request_seq != 0:
		return
	var request_seq: int = App.request_player_equipment_enhance(item_uid)
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	if _request_loading != null:
		_request_loading.show_waiting("正在强化装备")
	call_deferred("_wait_player_enhance_request", request_seq)


## 等待人物装备强化回包后关闭 loading 并刷新面板。
func _wait_player_enhance_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PLAYER_EQUIPMENT_ENHANCE_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	if _request_loading != null:
		_request_loading.hide_overlay()
	refresh_panel_data()


## 等待人物装备佩戴回包后关闭 loading 并刷新面板。
func _wait_player_equip_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PLAYER_EQUIP_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	if _request_loading != null:
		_request_loading.hide_overlay()
	refresh_panel_data()


## 填充指定下拉框的宠物列表。
func _rebuild_pet_pick_options_for(option_button: OptionButton) -> void:
	if option_button == null:
		return
	option_button.clear()
	for pet_variant: Variant in GameState.pets:
		if pet_variant is not Dictionary:
			continue
		var pet: Dictionary = pet_variant as Dictionary
		var pet_uid: int = int(pet.get("pet_uid", 0))
		if pet_uid == 0:
			continue
		var label: String = "宠物 %d Lv.%d" % [int(pet.get("pet_id", 0)), int(pet.get("level", 1))]
		option_button.add_item(label, pet_uid)


## 向服务端发送法宝装备请求并等待回包。
func _request_artifact_equip(pet_uid: int, artifact_slot_index: int, bag_slot_index: int) -> void:
	if _loading_request_seq != 0:
		return
	var request_seq: int = App.request_pet_artifact_equip(pet_uid, artifact_slot_index, bag_slot_index)
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	if _request_loading != null:
		_request_loading.show_waiting("正在装备法宝")
	call_deferred("_wait_artifact_equip_request", request_seq)


## 等待法宝装备回包后关闭 loading 并刷新面板。
func _wait_artifact_equip_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.PET_ARTIFACT_EQUIP_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	if _request_loading != null:
		_request_loading.hide_overlay()
	refresh_panel_data()


## 点击使用按钮：需要选宠物时弹窗，否则直接请求。
func _on_use_button_pressed(slot_index: int, item: Dictionary) -> void:
	if _loading_request_seq != 0 or slot_index <= 0:
		return
	if not GameState.is_ws_authenticated:
		return
	var target_type: String = str(item.get("target_type", ""))
	if target_type == "pet_single":
		_pending_use_slot_index = slot_index
		_rebuild_pet_pick_options()
		if _pet_pick_option == null or _pet_pick_option.item_count == 0:
			return
		if _pet_pick_dialog != null:
			_pet_pick_dialog.popup_centered(Vector2i(280, 140))
		return
	_request_use_item(slot_index, 0)


## 填充宠物选择列表。
func _rebuild_pet_pick_options() -> void:
	_rebuild_pet_pick_options_for(_pet_pick_option)


## 确认选择宠物后发起使用请求。
func _on_pet_pick_confirmed() -> void:
	if _pet_pick_option == null or _pending_use_slot_index <= 0:
		return
	var selected_index: int = _pet_pick_option.get_selected()
	if selected_index < 0:
		return
	var target_pet_uid: int = int(_pet_pick_option.get_item_id(selected_index))
	_request_use_item(_pending_use_slot_index, target_pet_uid)
	_pending_use_slot_index = 0


## 向服务端发送使用物品请求并等待回包。
func _request_use_item(slot_index: int, target_pet_uid: int) -> void:
	if _loading_request_seq != 0:
		return
	var request_seq: int = App.request_use_item(_container_type, slot_index, 1, target_pet_uid)
	if request_seq <= 0:
		return
	_loading_request_seq = request_seq
	if _request_loading != null:
		_request_loading.show_waiting("正在使用物品")
	call_deferred("_wait_use_item_request", request_seq)


## 等待使用物品回包后关闭 loading 并刷新面板。
func _wait_use_item_request(expected_seq: int) -> void:
	while expected_seq != 0 and _loading_request_seq == expected_seq:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		if request_cmd != CommandIds.USE_ITEM_REQ or seq != expected_seq:
			continue
		break
	if _loading_request_seq != expected_seq:
		return
	_loading_request_seq = 0
	if _request_loading != null:
		_request_loading.hide_overlay()
	refresh_panel_data()

