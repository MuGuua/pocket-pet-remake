class_name NPCShopPanel
extends CanvasLayer

## 玩家点击购买按钮时广播商品信息，主场景据此向服务端发起购买请求。
signal buy_requested(item_id: int, price_copper: int)
## 面板关闭时通知主场景解除输入锁定。
signal panel_closed

@onready var _root: Control = $Root
@onready var _title_label: Label = $Root/Panel/Margin/Layout/TitleLabel
@onready var _wallet_label: Label = $Root/Panel/Margin/Layout/WalletLabel
@onready var _goods_container: VBoxContainer = $Root/Panel/Margin/Layout/GoodsContainer
@onready var _status_label: Label = $Root/Panel/Margin/Layout/StatusLabel
@onready var _close_button: Button = $Root/Panel/Margin/Layout/CloseButton

## 当前商店 NPC 实体 ID，购买请求必须原样带回服务端做 proximity 校验。
var _shop_entity_id: int = 0

## 绑定按钮事件并在启动时保持隐藏。
func _ready() -> void:
	if _close_button != null:
		_close_button.pressed.connect(_on_close_button_pressed)
	hide_panel(false)

## 用服务端返回的商店载荷刷新面板；每次打开都直接展示最新商品与钱包快照。
func show_shop(npc_name: String, shop_entity_id: int, shop_payload: Dictionary) -> void:
	_shop_entity_id = shop_entity_id
	if _title_label != null:
		_title_label.text = "%s 的商店" % npc_name
	_update_wallet_label(shop_payload.get("wallet", {}))
	_rebuild_goods(shop_payload.get("goods", []))
	if _status_label != null:
		_status_label.text = ""
	if _root != null:
		_root.show()
	visible = true

## 购买请求等待服务端回包时锁定按钮，避免重复提交。
func show_waiting_state(status_text: String) -> void:
	if _status_label != null:
		_status_label.text = status_text
	for child_variant: Variant in _goods_container.get_children():
		if child_variant is Button:
			var buy_button: Button = child_variant as Button
			buy_button.disabled = true
	if _close_button != null:
		_close_button.disabled = true

## 购买成功后刷新钱包展示并恢复按钮可点状态。
func update_wallet(wallet_payload: Dictionary) -> void:
	_update_wallet_label(wallet_payload)
	if _status_label != null:
		_status_label.text = "购买成功"
	for child_variant: Variant in _goods_container.get_children():
		if child_variant is Button:
			var buy_button: Button = child_variant as Button
			buy_button.disabled = false
	if _close_button != null:
		_close_button.disabled = false

## 购买失败时展示原因并恢复按钮。
func show_error_message(message: String) -> void:
	if _status_label != null:
		_status_label.text = message
	for child_variant: Variant in _goods_container.get_children():
		if child_variant is Button:
			var buy_button: Button = child_variant as Button
			buy_button.disabled = false
	if _close_button != null:
		_close_button.disabled = false

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
