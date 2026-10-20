class_name NPCShopPanel
extends CanvasLayer

## 玩家点击购买按钮时广播商品信息，主场景据此向服务端发起购买请求。
signal buy_requested(item_id: int, price_copper: int)
## 玩家点击出售按钮时广播背包格子信息，主场景据此向服务端发起 5111 出售请求。
## slot_index 与 item_id 一起提交，服务端会校验格子实际物品，避免旧快照误卖。
signal sell_requested(slot_index: int, item_id: int, quantity: int)
## 面板关闭时通知主场景解除输入锁定。
signal panel_closed

@onready var _root: Control = $Root
@onready var _title_label: Label = get_node_or_null("Root/Panel/Layout/TitleLabel") as Label
@onready var _wallet_label: Label = get_node_or_null("Root/Panel/Layout/WalletLabel") as Label
## 购买页商品列表容器。
@onready var _goods_container: VBoxContainer = get_node_or_null("Root/Panel/Layout/ListScroll/ListBox/GoodsContainer") as VBoxContainer
## 出售页背包物品列表容器。
@onready var _sell_container: VBoxContainer = get_node_or_null("Root/Panel/Layout/ListScroll/ListBox/SellContainer") as VBoxContainer
## 购买页签按钮。
@onready var _buy_tab_button: BaseButton = %BuyTabButton
## 出售页签按钮。
@onready var _sell_tab_button: BaseButton = %SellTabButton
@onready var _status_label: Label = get_node_or_null("Root/Panel/Layout/StatusLabel") as Label
@onready var _top_close_button: BaseButton = %TopCloseButton
@onready var _close_button: BaseButton = get_node_or_null("Root/Panel/Layout/RuntimeActionButton") as BaseButton

## 当前商店 NPC 实体 ID，购买请求必须原样带回服务端做 proximity 校验。
var _shop_entity_id: int = 0
## 当前是否处于出售页签。
var _is_sell_tab: bool = false
## 请求进行中时锁住所有列表按钮，防止重复提交。
var _request_pending: bool = false

## 绑定按钮事件并在启动时保持隐藏。
func _ready() -> void:
	if _top_close_button != null and not _top_close_button.pressed.is_connected(_on_close_button_pressed):
		_top_close_button.pressed.connect(_on_close_button_pressed)
	if _close_button != null:
		_close_button.pressed.connect(_on_close_button_pressed)
	if _buy_tab_button != null and not _buy_tab_button.pressed.is_connected(_on_buy_tab_pressed):
		_buy_tab_button.pressed.connect(_on_buy_tab_pressed)
	if _sell_tab_button != null and not _sell_tab_button.pressed.is_connected(_on_sell_tab_pressed):
		_sell_tab_button.pressed.connect(_on_sell_tab_pressed)
	# 背包快照来自服务端 5002 / 5011 推送；出售后服务端推送新背包，这里随之重建出售列表。
	if not GameState.bag_changed.is_connected(_on_bag_changed):
		GameState.bag_changed.connect(_on_bag_changed)
	hide_panel(false)


## 离开场景树时断开全局信号，避免已释放面板继续接收背包刷新。
func _exit_tree() -> void:
	if GameState.bag_changed.is_connected(_on_bag_changed):
		GameState.bag_changed.disconnect(_on_bag_changed)

## 用服务端返回的商店载荷刷新面板；每次打开都直接展示最新商品与钱包快照。
func show_shop(npc_name: String, shop_entity_id: int, shop_payload: Dictionary) -> void:
	_shop_entity_id = shop_entity_id
	if _title_label != null:
		_title_label.text = "%s 的商店" % npc_name
	_update_wallet_label(shop_payload.get("wallet", {}))
	_rebuild_goods(shop_payload.get("goods", []))
	_rebuild_sell_list()
	_request_pending = false
	_select_tab(false)
	if _status_label != null:
		_status_label.text = ""
	if _root != null:
		_root.show()
	visible = true

## 购买请求等待服务端回包时锁定按钮，避免重复提交。
func show_waiting_state(status_text: String) -> void:
	if _status_label != null:
		_status_label.text = status_text
	_request_pending = true
	_set_list_buttons_disabled(true)

## 购买成功后刷新钱包展示并恢复按钮可点状态。
func update_wallet(wallet_payload: Dictionary) -> void:
	update_wallet_with_status(wallet_payload, "购买成功")


## 买卖成功后刷新钱包展示、显示结果文案并恢复按钮；status_text 为展示给玩家的结果提示。
func update_wallet_with_status(wallet_payload: Dictionary, status_text: String) -> void:
	_update_wallet_label(wallet_payload)
	if _status_label != null:
		_status_label.text = status_text
	_request_pending = false
	_set_list_buttons_disabled(false)

## 购买失败时展示原因并恢复按钮。
func show_error_message(message: String) -> void:
	if _status_label != null:
		_status_label.text = message
	_request_pending = false
	_set_list_buttons_disabled(false)

## 隐藏商店面板并清理当前上下文。
func hide_panel(emit_closed_signal: bool = true) -> void:
	_shop_entity_id = 0
	if _root != null:
		_root.hide()
	visible = false
	if emit_closed_signal:
		panel_closed.emit()

func _update_wallet_label(wallet_variant: Variant) -> void:
	if _wallet_label == null or wallet_variant is not Dictionary:
		return
	var wallet_payload: Dictionary = wallet_variant as Dictionary
	var total_copper: int = int(wallet_payload.get("total_copper", 0))
	var gold: int = int(wallet_payload.get("gold", 0))
	var silver: int = int(wallet_payload.get("silver", 0))
	var copper: int = int(wallet_payload.get("copper", 0))
	_wallet_label.text = "持有货币：%d金 %d银 %d铜（共 %d 铜）" % [gold, silver, copper, total_copper]

func _rebuild_goods(goods_variant: Variant) -> void:
	for child_variant: Variant in _goods_container.get_children():
		_goods_container.remove_child(child_variant)
		(child_variant as Node).queue_free()
	if goods_variant is not Array:
		return
	for good_variant: Variant in goods_variant:
		if good_variant is not Dictionary:
			continue
		var good_payload: Dictionary = good_variant as Dictionary
		var item_id: int = int(good_payload.get("item_id", 0))
		var item_name: String = str(good_payload.get("item_name", "未知物品"))
		var price_copper: int = int(good_payload.get("price_copper", 0))
		var buy_button: Button = Button.new()
		buy_button.text = "%s - %d 铜" % [item_name, price_copper]
		buy_button.pressed.connect(_on_buy_button_pressed.bind(item_id, price_copper))
		_goods_container.add_child(buy_button)

func _on_buy_button_pressed(item_id: int, price_copper: int) -> void:
	if _shop_entity_id <= 0 or item_id <= 0:
		return
	show_waiting_state("购买处理中...")
	buy_requested.emit(item_id, price_copper)

func _on_close_button_pressed() -> void:
	hide_panel()


## 统一切换列表按钮与底部关闭按钮的可点状态；disabled 为 true 时锁定。
func _set_list_buttons_disabled(disabled: bool) -> void:
	for container: VBoxContainer in [_goods_container, _sell_container]:
		if container == null:
			continue
		for child_variant: Variant in container.get_children():
			if child_variant is Button:
				(child_variant as Button).disabled = disabled
	if _close_button != null:
		_close_button.disabled = disabled


## 切换购买/出售页签；sell 为 true 时显示出售列表。
func _select_tab(sell: bool) -> void:
	_is_sell_tab = sell
	if _goods_container != null:
		_goods_container.visible = not sell
	if _sell_container != null:
		_sell_container.visible = sell
	if _buy_tab_button != null:
		_buy_tab_button.set_pressed_no_signal(not sell)
	if _sell_tab_button != null:
		_sell_tab_button.set_pressed_no_signal(sell)


## 点击「购买」页签。
func _on_buy_tab_pressed() -> void:
	_select_tab(false)


## 点击「出售」页签。
func _on_sell_tab_pressed() -> void:
	_select_tab(true)


## 背包快照变化时重建出售列表；面板隐藏时跳过，避免无意义的节点重建。
func _on_bag_changed() -> void:
	if not visible:
		return
	_rebuild_sell_list()
	if _request_pending:
		_set_list_buttons_disabled(true)


## 用 GameState 中的服务端背包快照重建出售列表。
## 只列出模板可卖、售价大于 0 且未绑定的物品；这只是展示过滤，最终可否出售由 5111 服务端判定。
func _rebuild_sell_list() -> void:
	if _sell_container == null:
		return
	for child_variant: Variant in _sell_container.get_children():
		_sell_container.remove_child(child_variant)
		(child_variant as Node).queue_free()
	var listed_count: int = 0
	for item_variant: Variant in GameState.bag_items:
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		var item_id: int = int(item.get("item_id", 0))
		var slot_index: int = int(item.get("slot_index", 0))
		var quantity: int = int(item.get("quantity", 0))
		var unit_price: int = int(item.get("sell_price_copper", 0))
		if item_id <= 0 or slot_index <= 0 or quantity <= 0:
			continue
		if not bool(item.get("can_sell", false)) or unit_price <= 0 or bool(item.get("is_bound", false)):
			continue
		var sell_button: Button = Button.new()
		sell_button.text = "%s ×%d  单价 %d 铜" % [str(item.get("item_name", "未知物品")), quantity, unit_price]
		sell_button.pressed.connect(_on_sell_button_pressed.bind(slot_index, item_id))
		_sell_container.add_child(sell_button)
		listed_count += 1
	if listed_count == 0:
		var empty_label: Label = Label.new()
		empty_label.text = "背包中没有可出售的物品"
		empty_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		_sell_container.add_child(empty_label)


## 点击出售某个背包格子；当前每次出售 1 个，堆叠物品可多次点击。
func _on_sell_button_pressed(slot_index: int, item_id: int) -> void:
	if _shop_entity_id <= 0 or slot_index <= 0 or item_id <= 0 or _request_pending:
		return
	show_waiting_state("出售处理中...")
	sell_requested.emit(slot_index, item_id, 1)
