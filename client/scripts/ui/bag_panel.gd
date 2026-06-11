extends PanelContainer

# 当前背包面板沿用旧 UI 外框，但内部数据改为直接消费 GameState 中
# 的服务端权威背包快照，避免继续展示静态占位数字。
const INVENTORY_CAPACITY: int = 30
const GRID_PAGE_SIZE: int = 32
const UiFormat = preload("res://scripts/common/ui_format.gd")

@onready var capacity_label: Label = $VBoxContainer/HBoxContainer2/PanelContainer/HBoxContainer/Label
@onready var gold_label: Label = $VBoxContainer/HBoxContainer2/PanelContainer2/HBoxContainer/Label
@onready var distinct_count_label: Label = $VBoxContainer/HBoxContainer2/PanelContainer2/HBoxContainer/Label3
@onready var page_label: Label = $VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer/HBoxContainer/PanelContainer/Label
@onready var grid_container: GridContainer = $VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer/GridContainer

var _items_summary_label: Label


func _ready() -> void:
	# 追加一个运行时文本区域，用最小改动把服务端背包列表摘要展示出来。
	_items_summary_label = Label.new()
	_items_summary_label.name = "ItemsSummaryLabel"
	_items_summary_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	_items_summary_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_LEFT
	$VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer.add_child(_items_summary_label)
	$VBoxContainer/PanelContainer2/MarginContainer/HBoxContainer.move_child(_items_summary_label, 1)

	if not GameState.bag_changed.is_connected(refresh_panel_data):
		GameState.bag_changed.connect(refresh_panel_data)
	if not GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.connect(refresh_panel_data)
	refresh_panel_data()


func _exit_tree() -> void:
	# 面板销毁时及时断开全局状态信号，避免悬空回调。
	if GameState.bag_changed.is_connected(refresh_panel_data):
		GameState.bag_changed.disconnect(refresh_panel_data)
	if GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.disconnect(refresh_panel_data)


func refresh_panel_data() -> void:
	var items: Array = GameState.bag_items
	var total_stack_count: int = 0
	for item_variant in items:
		if item_variant is Dictionary:
			total_stack_count += int(item_variant.get("count", 0))

	capacity_label.text = "%d/%d" % [items.size(), INVENTORY_CAPACITY]
	gold_label.text = UiFormat.value_to_text(GameState.player_snapshot.get("gold", 0))
	distinct_count_label.text = UiFormat.value_to_text(total_stack_count)
	page_label.text = UiFormat.normalize_text("1/%d" % maxi(1, int(ceil(float(max(items.size(), 1)) / float(GRID_PAGE_SIZE)))))
	_items_summary_label.text = _build_items_summary(items)
	_apply_grid_tooltips(items)


func _build_items_summary(items: Array) -> String:
	if items.is_empty():
		return "服务端背包为空，等待后续获得物品。"

	var lines: Array[String] = []
	var max_lines := mini(items.size(), 6)
	for index in range(max_lines):
		var item_variant: Variant = items[index]
		if item_variant is Dictionary:
			var item: Dictionary = item_variant
			lines.append(
				"槽位%d 物品ID %d x%d" % [
					index + 1,
					int(item.get("item_id", 0)),
					int(item.get("count", 0)),
				]
			)
	if items.size() > max_lines:
		lines.append("...... 其余 %d 个物品槽请以后续背包交互页为准" % (items.size() - max_lines))
	return UiFormat.normalize_text("\n".join(lines))


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
			cell.tooltip_text = UiFormat.normalize_text("item_id=%d\ncount=%d\nitem_uid=%d" % [
				int(item.get("item_id", 0)),
				int(item.get("count", 0)),
				int(item.get("item_uid", 0)),
			])
		else:
			cell.tooltip_text = "槽位数据格式异常"
