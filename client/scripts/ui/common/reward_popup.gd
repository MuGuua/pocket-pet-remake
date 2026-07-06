class_name RewardPopup
extends "res://scripts/ui/common/modal_popup_layer.gd"

## 通用奖励弹窗场景路径，供战斗结算、礼包开启等流程复用。
const SCENE_PATH: String = "res://scenes/ui/common/reward_popup.tscn"
## 默认标题文案。
const DEFAULT_TITLE: String = "您获得了："
## 数值高亮色，对应 RGB(130, 213, 99)。
const REWARD_NUMBER_COLOR: String = "#82D563"
## 奖励面板根容器；当前场景结构为 Control/VBoxContainer/...。
const ROOT_PANEL_PATH: NodePath = ^"Control"
## 模糊背景节点路径；当前使用通用模糊背景子场景，不再强依赖旧 DimLayer。
const BLUR_BACKDROP_PATH: NodePath = ^"Control/ColorRect"
## 标题面板路径；结构调整后标题被放到 ScrollContainer 内。
const TITLE_PANEL_PATH: NodePath = ^"Control/VBoxContainer/ScrollContainer/Title"
## 奖励弹窗正文容器的真实路径；用于恢复正文区点击交互。
const CENTER_CONTAINER_PATH: NodePath = ^"Control/VBoxContainer/CenterContainer"
## 奖励弹窗正文面板的真实路径；用于让面板区域重新接管鼠标事件。
const PANEL_PATH: NodePath = ^"Control/VBoxContainer/CenterContainer/PanelContainer"

## 奖励列表容器。
@onready var _content_list: VBoxContainer = %ContentList
## 普通文本行模板；具体字号与自动换行等布局以场景节点为准。
@onready var _plain_line_template: Label = %ContentList.get_node_or_null("PlainLineTemplate") as Label
## 富文本行模板；用于经验、铜币、宠物经验与技能进度等高亮文本。
@onready var _rich_line_template: RichTextLabel = %ContentList.get_node_or_null("RichLineTemplate") as RichTextLabel
## 物品奖励行模板；图标尺寸、行距与对齐方式统一以场景节点为准。
@onready var _item_reward_template: HBoxContainer = %ContentList.get_node_or_null("ItemRewardTemplate") as HBoxContainer
## 奖励弹窗右上角关闭按钮；单独显式绑定，避免场景重构后仍依赖基类查找链路。
@onready var _top_close_button_direct: BaseButton = %TopCloseButton
## 奖励弹窗底部继续按钮；点击、5 键和回车键都会关闭弹窗。
@onready var _continue_button: BaseButton = %ContinueButton


## 初始化奖励弹窗；沿用模态弹窗基类，并补齐正文区域交互节点配置。
func _ready() -> void:
	super._ready()
	_bind_reward_popup_dim_layer()
	_bind_direct_close_button()
	_bind_continue_button()
	_set_content_mouse_ignore(true)
	_apply_interactive_nodes()


## 展示奖励列表；无有效奖励时不弹窗。
## title_text 为空时使用 DEFAULT_TITLE；items_header_text 已废弃，保留参数仅为兼容旧调用。
## skill_progress_rewards 为战斗结算中的武器技能学习进度行。
func show_rewards(
	title_text: String,
	rewards: Array,
	pet_rewards: Array = [],
	items_header_text: String = "",
	skill_progress_rewards: Array = []
) -> void:
	var parsed: Dictionary = _parse_rewards(rewards, pet_rewards)
	if not _has_visible_reward(parsed, skill_progress_rewards):
		return
	_clear_reward_rows()
	var resolved_title: String = title_text if not title_text.is_empty() else DEFAULT_TITLE
	_append_plain_line(resolved_title)
	var player_exp: int = int(parsed.get("player_exp", 0))
	if player_exp > 0:
		_append_labeled_value_line("经验：", player_exp)
	var gold_copper: int = int(parsed.get("gold_copper", 0))
	if gold_copper > 0:
		_append_labeled_value_line("铜币：", gold_copper)
	var attr_rewards: Array = parsed.get("attrs", []) as Array
	for attr_reward_variant: Variant in attr_rewards:
		if attr_reward_variant is not Dictionary:
			continue
		var attr_reward: Dictionary = attr_reward_variant as Dictionary
		_append_labeled_value_line("%s：" % _format_attr_reward_label(str(attr_reward.get("attr_key", ""))), int(attr_reward.get("value", 0)))
	var pet_exp: int = int(parsed.get("pet_exp", 0))
	if pet_exp > 0:
		_append_labeled_value_line("宠物经验：", pet_exp)
	for skill_progress_variant: Variant in skill_progress_rewards:
		if skill_progress_variant is not Dictionary:
			continue
		_append_skill_progress_line(skill_progress_variant as Dictionary)
	var item_rewards: Array = parsed.get("items", []) as Array
	for item_reward_variant: Variant in item_rewards:
		if item_reward_variant is not Dictionary:
			continue
		_append_item_reward_line(item_reward_variant as Dictionary)
	var granted_pets: Array = parsed.get("pets", []) as Array
	for pet_reward_variant: Variant in granted_pets:
		if pet_reward_variant is not Dictionary:
			continue
		_append_pet_reward_line(pet_reward_variant as Dictionary)
	_apply_interactive_nodes()
	_move_continue_button_to_bottom()
	_open_modal()


## 关闭奖励弹窗。
func close_popup() -> void:
	print("[RewardPopup] close_popup requested visible=%s topmost=%s frame=%d" % [visible, _is_topmost_runtime_modal(), Engine.get_process_frames()])
	if not visible:
		print("[RewardPopup] close_popup ignored because popup is not visible")
		return
	_dismiss_modal()
	print("[RewardPopup] close_popup dismissed visible=%s" % visible)


## 奖励弹窗单独绑定右上角关闭按钮，避免场景层级调整后基类绑定失效或被事件链吞掉。
func _bind_direct_close_button() -> void:
	if _top_close_button_direct == null:
		print("[RewardPopup] TopCloseButton bind failed: node is null")
		return
	_top_close_button_direct.mouse_filter = Control.MOUSE_FILTER_STOP
	print("[RewardPopup] TopCloseButton bound disabled=%s visible=%s mouse_filter=%s" % [_top_close_button_direct.disabled, _top_close_button_direct.visible, _top_close_button_direct.mouse_filter])
	if not _top_close_button_direct.gui_input.is_connected(_on_direct_top_close_button_gui_input):
		_top_close_button_direct.gui_input.connect(_on_direct_top_close_button_gui_input)
	if not _top_close_button_direct.button_down.is_connected(_on_direct_top_close_button_down):
		_top_close_button_direct.button_down.connect(_on_direct_top_close_button_down)
	if not _top_close_button_direct.pressed.is_connected(_on_direct_top_close_button_pressed):
		_top_close_button_direct.pressed.connect(_on_direct_top_close_button_pressed)


## 记录关闭按钮原始 GUI 输入，用于确认点击是否到达按钮节点。
func _on_direct_top_close_button_gui_input(event: InputEvent) -> void:
	if event is InputEventMouseButton:
		var mouse_event: InputEventMouseButton = event as InputEventMouseButton
		print("[RewardPopup] TopCloseButton gui_input mouse button=%s pressed=%s visible=%s disabled=%s" % [mouse_event.button_index, mouse_event.pressed, visible, _top_close_button_direct.disabled])
	elif event is InputEventScreenTouch:
		var touch_event: InputEventScreenTouch = event as InputEventScreenTouch
		print("[RewardPopup] TopCloseButton gui_input touch index=%s pressed=%s visible=%s disabled=%s" % [touch_event.index, touch_event.pressed, visible, _top_close_button_direct.disabled])


## 记录关闭按钮按下态，用于区分 GUI 输入到达但 Button 未触发 pressed 的情况。
func _on_direct_top_close_button_down() -> void:
	print("[RewardPopup] TopCloseButton button_down visible=%s topmost=%s disabled=%s" % [visible, _is_topmost_runtime_modal(), _top_close_button_direct.disabled])


## 直接关闭奖励弹窗；不再额外依赖基类的按钮冷却判断。
func _on_direct_top_close_button_pressed() -> void:
	print("[RewardPopup] TopCloseButton pressed visible=%s topmost=%s disabled=%s" % [visible, _is_topmost_runtime_modal(), _top_close_button_direct.disabled])
	close_popup()


## 绑定底部继续按钮，作为移动端和键盘之外的显式关闭入口。
func _bind_continue_button() -> void:
	if _continue_button == null:
		return
	_continue_button.mouse_filter = Control.MOUSE_FILTER_STOP
	_set_button_children_mouse_ignore(_continue_button)
	if not _continue_button.pressed.is_connected(_on_continue_button_pressed):
		_continue_button.pressed.connect(_on_continue_button_pressed)


## 响应底部继续按钮点击。
func _on_continue_button_pressed() -> void:
	close_popup()


## 让按钮内的贴图、文字和动画只负责显示，不拦截按钮点击。
func _set_button_children_mouse_ignore(button: BaseButton) -> void:
	for child: Node in button.get_children():
		_apply_mouse_filter_recursive(child, true)


## 奖励弹窗保留全局输入兜底：当 GUI 没有命中按钮时，仍可按按钮屏幕矩形关闭。
func _enable_modal_input_listeners() -> void:
	set_process_input(true)


func _disable_modal_input_listeners() -> void:
	set_process_input(false)


## 兜底捕获右上角关闭按钮点击；用于处理深层 UI 布局导致 Button 收不到 gui_input 的情况。
func _input(event: InputEvent) -> void:
	if not visible or not _is_topmost_runtime_modal():
		return
	if _is_continue_key_event(event):
		get_viewport().set_input_as_handled()
		close_popup()
		return
	var click_position: Vector2 = _extract_pressed_position(event)
	if click_position == Vector2.INF:
		get_viewport().set_input_as_handled()
		return
	if _is_position_inside_close_button(click_position):
		print("[RewardPopup] TopCloseButton fallback_input position=%s visible=%s topmost=%s" % [click_position, visible, _is_topmost_runtime_modal()])
		get_viewport().set_input_as_handled()
		close_popup()
		return
	if _is_position_inside_continue_button(click_position):
		get_viewport().set_input_as_handled()
		close_popup()
		return
	get_viewport().set_input_as_handled()


## 判断是否按下奖励弹窗继续键；移动端外接键盘和桌面调试都复用这套入口。
func _is_continue_key_event(event: InputEvent) -> bool:
	if not (event is InputEventKey):
		return false
	var key_event: InputEventKey = event as InputEventKey
	if not key_event.pressed or key_event.echo:
		return false
	return (
		key_event.keycode == KEY_5
		or key_event.physical_keycode == KEY_5
		or key_event.keycode == KEY_ENTER
		or key_event.physical_keycode == KEY_ENTER
		or key_event.keycode == KEY_KP_ENTER
		or key_event.physical_keycode == KEY_KP_ENTER
	)


## 从鼠标或触摸输入中提取按下位置；非按下事件返回 Vector2.INF。
func _extract_pressed_position(event: InputEvent) -> Vector2:
	if event is InputEventMouseButton:
		var mouse_event: InputEventMouseButton = event as InputEventMouseButton
		if mouse_event.pressed:
			return mouse_event.position
	if event is InputEventScreenTouch:
		var touch_event: InputEventScreenTouch = event as InputEventScreenTouch
		if touch_event.pressed:
			return touch_event.position
	return Vector2.INF


## 判断输入坐标是否落在关闭按钮热区内；额外扩 24 像素方便移动端点击。
func _is_position_inside_close_button(click_position: Vector2) -> bool:
	if _top_close_button_direct == null:
		return false
	var close_rect: Rect2 = _resolve_button_screen_rect(_top_close_button_direct).grow(24.0)
	print("[RewardPopup] close button hit test click=%s rect=%s" % [click_position, close_rect])
	return close_rect.has_point(click_position)


## 判断输入坐标是否落在底部继续按钮热区内；用于兜底处理 GUI 未触发 pressed 的情况。
func _is_position_inside_continue_button(click_position: Vector2) -> bool:
	if _continue_button == null:
		return false
	var continue_rect: Rect2 = _resolve_button_screen_rect(_continue_button).grow(12.0)
	return continue_rect.has_point(click_position)


## 计算按钮真实屏幕矩形；额外叠加父级 offset_transform_position，避免视觉偏移后热区停在旧位置。
func _resolve_button_screen_rect(button: BaseButton) -> Rect2:
	var button_rect: Rect2 = button.get_global_rect()
	var visual_offset: Vector2 = _collect_offset_transform_position(button)
	return Rect2(button_rect.position + visual_offset, button_rect.size)


## 汇总祖先 Control 的 offset_transform_position；Godot 的 get_global_rect 不会稳定反映该视觉偏移。
func _collect_offset_transform_position(node: Node) -> Vector2:
	var offset: Vector2 = Vector2.ZERO
	var current: Node = node
	while current != null and current != self:
		if current is Control:
			var control: Control = current as Control
			if control.offset_transform_enabled:
				offset += control.offset_transform_position
		current = current.get_parent()
	return offset


## 奖励弹窗结构调整后没有固定 DimLayer；如果存在可交互背景层，则按遮罩语义绑定。
func _bind_reward_popup_dim_layer() -> void:
	_dim_layer = get_node_or_null(BLUR_BACKDROP_PATH) as ColorRect
	if _dim_layer == null:
		_dim_layer = get_node_or_null("DimLayer") as ColorRect
	if _dim_layer == null:
		return
	_dim_layer.mouse_filter = Control.MOUSE_FILTER_STOP
	if not _dim_layer.gui_input.is_connected(_on_dim_layer_gui_input):
		_dim_layer.gui_input.connect(_on_dim_layer_gui_input)


## 让奖励正文面板与右上角关闭按钮可交互，其余区域仍保持遮罩语义。
func _apply_interactive_nodes() -> void:
	var root_panel: Control = get_node_or_null(ROOT_PANEL_PATH) as Control
	if root_panel != null:
		root_panel.mouse_filter = Control.MOUSE_FILTER_STOP
	var title_panel: Control = get_node_or_null(TITLE_PANEL_PATH) as Control
	if title_panel != null:
		title_panel.mouse_filter = Control.MOUSE_FILTER_STOP
	var panel: Control = get_node_or_null(PANEL_PATH) as Control
	if panel != null:
		panel.mouse_filter = Control.MOUSE_FILTER_STOP
	var close_button: BaseButton = get_node_or_null("%TopCloseButton") as BaseButton
	if close_button != null:
		close_button.mouse_filter = Control.MOUSE_FILTER_STOP
	var continue_button: BaseButton = get_node_or_null("%ContinueButton") as BaseButton
	if continue_button != null:
		continue_button.mouse_filter = Control.MOUSE_FILTER_STOP
		_set_button_children_mouse_ignore(continue_button)


## 奖励弹窗正文容器已被移动到深层节点下，覆写基类查找路径以继续支持“点空白关闭”。
func _set_content_mouse_ignore(ignore: bool) -> void:
	var center: Control = get_node_or_null(CENTER_CONTAINER_PATH) as Control
	if center == null:
		return
	_apply_mouse_filter_recursive(center, ignore)


## 解析奖励数组，拆出经验、货币、物品与宠物展示数据。
func _parse_rewards(rewards: Array, battle_pet_rewards: Array) -> Dictionary:
	var player_exp: int = 0
	var gold_copper: int = 0
	var item_rewards: Array[Dictionary] = []
	var attr_rewards: Array[Dictionary] = []
	var granted_pets: Array[Dictionary] = []
	for reward_variant: Variant in rewards:
		if reward_variant is not Dictionary:
			continue
		var reward: Dictionary = reward_variant as Dictionary
		var reward_type: String = str(reward.get("type", ""))
		match reward_type:
			"exp":
				player_exp += int(reward.get("value", 0))
			"gold":
				gold_copper += int(reward.get("value", 0))
			"item":
				var item_count: int = int(reward.get("count", 0))
				if item_count <= 0:
					continue
				item_rewards.append(reward.duplicate(true))
			"pet":
				var pet_id: int = int(reward.get("pet_id", 0))
				if pet_id <= 0:
					continue
				granted_pets.append(reward.duplicate(true))
			"attr":
				var attr_value: int = int(reward.get("value", 0))
				if attr_value <= 0:
					continue
				attr_rewards.append(reward.duplicate(true))
	return {
		"player_exp": player_exp,
		"gold_copper": gold_copper,
		"pet_exp": _sum_pet_exp(battle_pet_rewards),
		"items": item_rewards,
		"attrs": attr_rewards,
		"pets": granted_pets,
	}


## 将服务端属性字段转为奖励弹窗里的中文展示名。
func _format_attr_reward_label(attr_key: String) -> String:
	match attr_key:
		"free_attr_points":
			return "自由属性点"
		"strength":
			return "力量"
		"vitality":
			return "体质"
		"agility":
			return "敏捷"
		"mind":
			return "灵力"
		"hp_max":
			return "生命上限"
		"atk":
			return "攻击"
		"def":
			return "防御"
		"spd":
			return "速度"
		"mana":
			return "精力"
		_:
			return "属性"


## 汇总所有参战宠物的经验奖励。
func _sum_pet_exp(pet_rewards: Array) -> int:
	var total_exp: int = 0
	for pet_reward_variant: Variant in pet_rewards:
		if pet_reward_variant is not Dictionary:
			continue
		var pet_reward: Dictionary = pet_reward_variant as Dictionary
		total_exp += int(pet_reward.get("exp_gained", pet_reward.get("exp", 0)))
	return total_exp


## 判断是否至少有一条可展示奖励内容。
func _has_visible_reward(parsed: Dictionary, skill_progress_rewards: Array = []) -> bool:
	if int(parsed.get("player_exp", 0)) > 0:
		return true
	if int(parsed.get("gold_copper", 0)) > 0:
		return true
	var attr_rewards: Array = parsed.get("attrs", []) as Array
	if not attr_rewards.is_empty():
		return true
	if int(parsed.get("pet_exp", 0)) > 0:
		return true
	if not skill_progress_rewards.is_empty():
		return true
	var item_rewards: Array = parsed.get("items", []) as Array
	if not item_rewards.is_empty():
		return true
	var granted_pets: Array = parsed.get("pets", []) as Array
	return not granted_pets.is_empty()


func _clear_reward_rows() -> void:
	if _content_list == null:
		return
	for child: Node in _content_list.get_children():
		if (
			child == _plain_line_template
			or child == _rich_line_template
			or child == _item_reward_template
			or child == _continue_button
		):
			continue
		child.queue_free()


## 把继续按钮保持在奖励列表底部；按钮现在放在 ContentList 内，避免被新追加的奖励行挤到中间。
func _move_continue_button_to_bottom() -> void:
	if _content_list == null or _continue_button == null:
		return
	if _continue_button.get_parent() != _content_list:
		return
	_content_list.move_child(_continue_button, _content_list.get_child_count() - 1)


## 追加一行左对齐纯文本。
func _append_plain_line(text: String) -> void:
	if _content_list == null:
		return
	var row_label: Label = null
	if _plain_line_template != null:
		row_label = _plain_line_template.duplicate() as Label
	if row_label == null:
		row_label = Label.new()
		row_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_LEFT
		row_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	row_label.visible = true
	row_label.add_theme_font_size_override("font_size", 26)
	row_label.text = UiFormat.normalize_text(text)
	row_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_content_list.add_child(row_label)


## 追加一行「标签 + 高亮数值」，例如 铜币：200。
func _append_labeled_value_line(label_prefix: String, value: int) -> void:
	_append_rich_line("%s%s" % [label_prefix, _colored_number(value)])


## 追加武器技能学习进度行，例如「试炼剑技技能经验：2/100」。
func _append_skill_progress_line(skill_progress: Dictionary) -> void:
	var skill_name: String = str(skill_progress.get("skill_name", "")).strip_edges()
	if skill_name.is_empty():
		var skill_id: int = int(skill_progress.get("skill_id", 0))
		if skill_id > 0:
			skill_name = "技能%d" % skill_id
		else:
			skill_name = "武器技能"
	skill_name = UiFormat.normalize_text(skill_name)
	var skill_exp: int = int(skill_progress.get("skill_exp", 0))
	var learn_exp_required: int = int(skill_progress.get("learn_exp_required", 0))
	if learn_exp_required <= 0:
		return
	var learned_suffix: String = ""
	if bool(skill_progress.get("newly_learned", false)):
		learned_suffix = "（已学会）"
	_append_rich_line(
		"%s技能经验：%s/%s%s" % [
			skill_name,
			_colored_number(skill_exp),
			_colored_number(learn_exp_required),
			learned_suffix,
		]
	)


## 追加一行左对齐 BBCode 文本，用于数值高亮。
func _append_rich_line(bbcode_text: String) -> void:
	if _content_list == null:
		return
	var row_label: RichTextLabel = null
	if _rich_line_template != null:
		row_label = _rich_line_template.duplicate() as RichTextLabel
	if row_label == null:
		row_label = RichTextLabel.new()
		row_label.bbcode_enabled = true
		row_label.fit_content = true
		row_label.scroll_active = false
		row_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row_label.visible = true
	row_label.text = bbcode_text
	row_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_content_list.add_child(row_label)


## 把整数包成高亮 BBCode 片段。
func _colored_number(value: int) -> String:
	return "[color=%s]%s[/color]" % [REWARD_NUMBER_COLOR, UiFormat.value_to_text(value)]


## 解析奖励物品展示名：优先服务端 item_name，缺失时尝试从本地背包/已佩戴快照回填。
func _resolve_server_item_name(reward: Dictionary) -> String:
	var item_name: String = str(reward.get("item_name", "")).strip_edges()
	if not item_name.is_empty():
		return UiFormat.normalize_text(item_name)
	var item_id: int = int(reward.get("item_id", 0))
	var fallback_name: String = _resolve_local_item_name(item_id)
	if not fallback_name.is_empty():
		return UiFormat.normalize_text(fallback_name)
	push_warning(
        "RewardPopup: 服务端未返回 item_name，item_id=%s"
		% UiFormat.value_to_text(item_id)
	)
	return "未知物品"


## 从 GameState 已有物品快照按 item_id 查找展示名，仅作服务端字段缺失时的兜底。
func _resolve_local_item_name(item_id: int) -> String:
	if item_id <= 0:
		return ""
	for item_variant: Variant in GameState.bag_items:
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		if BagUiMapper.item_id(item) != item_id:
			continue
		var cached_name: String = BagUiMapper.item_name(item)
		if cached_name != "未知物品":
			return cached_name
	for item_variant: Variant in GameState.equipped_items:
		if item_variant is not Dictionary:
			continue
		var item: Dictionary = item_variant as Dictionary
		if BagUiMapper.item_id(item) != item_id:
			continue
		var cached_name: String = BagUiMapper.item_name(item)
		if cached_name != "未知物品":
			return cached_name
	return ""


## 追加一行物品奖励：图标（与文字同高）+ 服务端物品名：x数量。
func _append_item_reward_line(reward: Dictionary) -> void:
	if _content_list == null:
		return
	var row: HBoxContainer = null
	if _item_reward_template != null:
		row = _item_reward_template.duplicate() as HBoxContainer
	if row == null:
		row = HBoxContainer.new()
		row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.alignment = BoxContainer.ALIGNMENT_CENTER
		row.add_theme_constant_override("separation", 4)
	row.visible = true
	row.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var icon_rect: TextureRect = row.get_node_or_null("ItemIconTemplate") as TextureRect
	if icon_rect == null:
		icon_rect = TextureRect.new()
		icon_rect.custom_minimum_size = Vector2(40, 40)
		icon_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
		icon_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
		row.add_child(icon_rect)
	var icon_texture: Texture2D = _resolve_item_icon_texture(reward)
	icon_rect.texture = icon_texture
	icon_rect.visible = icon_texture != null
	icon_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var item_name: String = _resolve_server_item_name(reward)
	var item_count: int = maxi(int(reward.get("count", 0)), 1)
	var name_label: RichTextLabel = row.get_node_or_null("ItemNameTemplate") as RichTextLabel
	if name_label == null:
		name_label = RichTextLabel.new()
		name_label.bbcode_enabled = true
		name_label.fit_content = true
		name_label.scroll_active = false
		name_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(name_label)
	name_label.text = "%s：x%s" % [item_name, _colored_number(item_count)]
	name_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_content_list.add_child(row)


## 解析宠物奖励展示名：优先服务端 item_name（礼包配置 pet_name 映射），其次 pet_name。
func _resolve_pet_reward_name(reward: Dictionary) -> String:
	var pet_name: String = str(reward.get("item_name", "")).strip_edges()
	if pet_name.is_empty():
		pet_name = str(reward.get("pet_name", "")).strip_edges()
	if not pet_name.is_empty():
		return UiFormat.normalize_text(pet_name)
	var pet_id: int = int(reward.get("pet_id", 0))
	if pet_id > 0:
		return "宠物 %s" % UiFormat.value_to_text(pet_id)
	return "宠物"


## 追加一行宠物奖励：宠物名 ×1。
func _append_pet_reward_line(reward: Dictionary) -> void:
	var pet_name: String = _resolve_pet_reward_name(reward)
	_append_rich_line("宠物：%s ×%s" % [pet_name, _colored_number(1)])


## 按 item_id 从本地注册表解析奖励图标。
func _resolve_item_icon_texture(reward: Dictionary) -> Texture2D:
	return ItemIcons.resolve_texture(int(reward.get("item_id", 0)))
