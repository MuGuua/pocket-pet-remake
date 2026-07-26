extends RuntimeRootPanel

const DEFAULT_TAB_INDEX: int = 0

## 战斗属性切换按钮。
@onready var battle_attribute_button: Button = $Control/VBoxContainer/MarginContainer2/HBoxContainer/Button
## 状态抗性切换按钮。
@onready var status_resistance_button: Button = $Control/VBoxContainer/MarginContainer2/HBoxContainer/Button2
## 社会属性切换按钮。
@onready var social_attribute_button: Button = $Control/VBoxContainer/MarginContainer2/HBoxContainer/Button3
## 战斗属性内容面板。
@onready var battle_attributes_panel: Control = $Control/VBoxContainer/BattleAttributes
## 状态抗性内容面板。
@onready var status_resistance_panel: Control = $Control/VBoxContainer/StatusEsistance
## 社会属性内容面板。
@onready var social_attribute_panel: Control = $Control/VBoxContainer/SocialAttribute
## 场景中配置好的请求 loading；脚本只控制显隐，不创建或覆盖面板布局。
@onready var request_loading: GenericLoadingScene = $RequestLoadingOverlay
## 人物状态面板的关闭按钮；优先使用场景显式节点名，兼容后续 UI 调整。
@onready var close_button: BaseButton = _resolve_close_button()

## 当前可切换的按钮集合。
var _tab_buttons: Array[Button] = []
## 当前可切换的内容面板集合。
var _tab_panels: Array[Control] = []
## 当前选中的分页索引。
var _current_tab_index: int = DEFAULT_TAB_INDEX
## 当前仍在等待回包的请求序列号集合，面板已打开后的二次刷新流程使用。
var _pending_request_seqs: Dictionary = {}
## 打开面板加载代次；关闭面板或重复打开时递增，用于取消进行中的 await。
var _open_load_generation: int = 0


## 初始化按钮事件、订阅权威快照变化，并默认隐藏弹窗。
func _ready() -> void:
	super._ready()
	_tab_buttons.append(battle_attribute_button)
	_tab_buttons.append(status_resistance_button)
	_tab_buttons.append(social_attribute_button)
	_tab_panels.append(battle_attributes_panel)
	_tab_panels.append(status_resistance_panel)
	_tab_panels.append(social_attribute_panel)

	battle_attribute_button.pressed.connect(_on_tab_pressed.bind(0))
	status_resistance_button.pressed.connect(_on_tab_pressed.bind(1))
	social_attribute_button.pressed.connect(_on_tab_pressed.bind(2))
	if close_button != null:
		_apply_close_button_static_style(close_button)
		close_button.button_down.connect(_on_close_button_pressed)
	if not GameState.session_changed.is_connected(refresh_panel_data):
		GameState.session_changed.connect(refresh_panel_data)
	if not GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.connect(refresh_panel_data)
	if not App.request_finished.is_connected(_on_request_finished):
		App.request_finished.connect(_on_request_finished)

	reset_to_default()
	refresh_panel_data()


## 断开全局状态信号，避免面板销毁后继续收到回调。
func _exit_tree() -> void:
	if GameState.session_changed.is_connected(refresh_panel_data):
		GameState.session_changed.disconnect(refresh_panel_data)
	if GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.disconnect(refresh_panel_data)
	if App.request_finished.is_connected(_on_request_finished):
		App.request_finished.disconnect(_on_request_finished)


## 面板展示前只拉取人物权威属性；背包内容必须等玩家点击背包入口后再查询。
func prepare_open_data() -> bool:
	_open_load_generation += 1
	var load_id: int = _open_load_generation
	_pending_request_seqs.clear()
	reset_to_default()
	if not GameState.is_ws_authenticated:
		return load_id == _open_load_generation
	var status_seq: int = App.refresh_player_status()
	var wait_result: Dictionary = await _wait_player_panel_open_request(status_seq)
	if load_id != _open_load_generation:
		return false
	return bool(wait_result.get("all_succeeded", false))


## 数据就绪后打开人物状态面板；不再在此处重复请求服务端。
func open_menu() -> void:
	super.open_menu()
	refresh_panel_data()


## 关闭人物状态弹窗，并通知主场景解除菜单锁定。
func close_menu() -> void:
	_open_load_generation += 1
	_pending_request_seqs.clear()
	_hide_loading_overlay()
	super.close_menu()


## 恢复默认选中项，供外部打开面板时主动调用。
func reset_to_default() -> void:
	_select_tab(DEFAULT_TAB_INDEX)


## 根据当前 GameState 中的服务端权威快照刷新所有展示字段。
func refresh_panel_data() -> void:
	var player_snapshot: Dictionary = GameState.player_snapshot
	_refresh_battle_attributes(player_snapshot)
	_refresh_status_resistance(player_snapshot)
	_refresh_social_attribute(player_snapshot)


## 响应玩家点击，切换到目标分页。
func _on_tab_pressed(index: int) -> void:
	_select_tab(index)


## 响应面板右上角/标题区关闭按钮，沿用统一关闭逻辑。
func _on_close_button_pressed() -> void:
	close_menu()


## 切换按钮按下状态和内容面板显隐。
func _select_tab(index: int) -> void:
	if index < 0 or index >= _tab_buttons.size():
		return

	_current_tab_index = index
	for button_index: int in range(_tab_buttons.size()):
		var tab_button: Button = _tab_buttons[button_index]
		if tab_button == null:
			continue
		tab_button.button_pressed = button_index == _current_tab_index

	for panel_index: int in range(_tab_panels.size()):
		var tab_panel: Control = _tab_panels[panel_index]
		if tab_panel == null:
			continue
		tab_panel.visible = panel_index == _current_tab_index


## 查找场景中新增的关闭按钮，避免脚本硬编码覆盖面板布局。
func _resolve_close_button() -> BaseButton:
	var candidate_paths: Array[NodePath] = [
		NodePath("Control/CloseButton"),
		NodePath("Control/VBoxContainer/MarginContainer/HBoxContainer/Button"),
		NodePath("Control/VBoxContainer/MarginContainer/CloseButton"),
		NodePath("Control/VBoxContainer/CloseButton"),
		NodePath("Control/Button"),
	]
	for candidate_path: NodePath in candidate_paths:
		var candidate_node: Node = get_node_or_null(candidate_path)
		if candidate_node is BaseButton:
			return candidate_node as BaseButton

	return _find_close_button_in_children(self)


## 递归扫描“关闭”按钮，兼容用户在场景里新增但尚未固定命名的节点。
func _find_close_button_in_children(root: Node) -> BaseButton:
	if root == null:
		return null

	for child: Node in root.get_children():
		if child is BaseButton:
			var button: BaseButton = child as BaseButton
			if button.name == "CloseButton":
				return button
			if button is Button and (button as Button).text == "关闭":
				return button
			if button is Button and not button.toggle_mode and (button as Button).text.is_empty():
				return button

		var nested_button: BaseButton = _find_close_button_in_children(child)
		if nested_button != null:
			return nested_button

	return null


## 关闭按钮只保留 normal 样式，避免 hover/pressed/focus 叠加造成视觉跳动。
func _apply_close_button_static_style(button: BaseButton) -> void:
	if button == null:
		return

	button.toggle_mode = false
	button.focus_mode = Control.FOCUS_NONE
	if button is Control:
		var button_control: Control = button as Control
		var normal_style: StyleBox = button_control.get_theme_stylebox("normal")
		if normal_style != null:
			button_control.add_theme_stylebox_override("hover", normal_style)
			button_control.add_theme_stylebox_override("pressed", normal_style)
			button_control.add_theme_stylebox_override("hover_pressed", normal_style)
			button_control.add_theme_stylebox_override("focus", normal_style)


## 等待独立人物资料回包；发送失败时直接结束 loading，避免面板永久等待。
func _wait_player_panel_open_request(status_seq: int) -> Dictionary:
	if status_seq <= 0:
		return {"all_succeeded": false}
	var request_succeeded: bool = false
	while true:
		var result: Array = await App.request_finished
		if result.size() < 5:
			continue
		var request_cmd: int = int(result[0])
		var seq: int = int(result[1])
		var succeeded: bool = bool(result[2])
		if seq != status_seq or request_cmd != CommandIds.PLAYER_PROFILE_REQ:
			continue
		request_succeeded = succeeded
		break
	return {"all_succeeded": request_succeeded}


## 面板已打开时向服务端重新拉取一次完整人物面板数据。
func _request_panel_data() -> void:
	if not GameState.is_ws_authenticated:
		refresh_panel_data()
		return

	var player_status_seq: int = App.refresh_player_status()
	_track_request_seq(player_status_seq)

	if _pending_request_seqs.is_empty():
		refresh_panel_data()
		return
	_show_loading_overlay()


## 记录一次已发出的请求序列号，等待 App.request_finished 后再关闭 loading。
func _track_request_seq(request_seq: int) -> void:
	if request_seq <= 0:
		return
	_pending_request_seqs[request_seq] = true


## 服务端请求完成后移除对应序列号，并在所有请求结束后刷新面板。
func _on_request_finished(_request_cmd: int, seq: int, _ok: bool, _response_cmd: int, _payload: Dictionary) -> void:
	if not _pending_request_seqs.has(seq):
		return
	_pending_request_seqs.erase(seq)
	if not _pending_request_seqs.is_empty():
		return
	_hide_loading_overlay()
	if visible:
		refresh_panel_data()


## 显示人物面板请求 loading，提示用户正在等待服务端权威数据。
func _show_loading_overlay() -> void:
	if request_loading != null:
		request_loading.show_waiting()


## 隐藏人物面板请求 loading。
func _hide_loading_overlay() -> void:
	if request_loading != null:
		request_loading.hide_loading()


## 刷新战斗属性分页，所有数值均来自服务端玩家快照。
func _refresh_battle_attributes(player_snapshot: Dictionary) -> void:
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/等级/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["level"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/生命/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["hp"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/生命/HBoxContainer/Label4", _snapshot_text(player_snapshot, ["hp_max"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/精力/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["vigor"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/精力/HBoxContainer/Label4", _snapshot_text(player_snapshot, ["vigor_max"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/元素属性/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["element", "element_type", "element_name"], "无"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/经验/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["exp"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/经验/HBoxContainer/Label3", _build_exp_suffix(player_snapshot))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/飞升/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["fly", "fly_value", "ascension"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/飞升/HBoxContainer/Label3", _snapshot_percent_suffix(player_snapshot, ["fly_rate", "ascension_rate_pct"], "0%"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/转职/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["transfer", "transfer_value", "job_level"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/转职/HBoxContainer/Label3", _snapshot_text(player_snapshot, ["transfer_state", "job_state"], ""))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/攻击和防御/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["atk", "attack"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/攻击和防御/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player_snapshot, ["def", "defense"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/速度和法力/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["spd", "speed"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/速度和法力/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player_snapshot, ["mana", "mp"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/命中和闪避/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["hit_pct", "hit"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/命中和闪避/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player_snapshot, ["dodge_pct", "dodge"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/致命和爆伤/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["crit_rate_pct", "crit"], "0"))
	_set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/致命和爆伤/HBoxContainer/HBoxContainer2/Label2", _snapshot_percent_text(player_snapshot, ["crit_dmg_pct", "crit_damage"], "0%"))


## 刷新状态抗性分页，字段缺失时显示 0，避免继续展示旧假数据。
func _refresh_status_resistance(player_snapshot: Dictionary) -> void:
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/物理攻击抗性/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["physical_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/物理攻击抗性/HBoxContainer/Label4", _snapshot_text(player_snapshot, ["reverse_physical_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/技能攻击抗性/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["skill_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/技能攻击抗性/HBoxContainer/Label4", _snapshot_text(player_snapshot, ["reverse_skill_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/混乱和昏睡/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["confusion_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/混乱和昏睡/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player_snapshot, ["sleep_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/麻痹和封印/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["paralysis_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/麻痹和封印/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player_snapshot, ["seal_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/诅咒/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["curse_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗致命和抗爆伤/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["crit_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗致命和抗爆伤/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player_snapshot, ["crit_dmg_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗人物和抗宠物/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["character_resist_pct"], "0"))
	_set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗人物和抗宠物/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player_snapshot, ["pet_resist_pct"], "0"))


## 刷新社会属性分页；背包数量不在人物面板查询，避免隐式加载背包物品。
func _refresh_social_attribute(player_snapshot: Dictionary) -> void:
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/家族/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["guild_name", "family_name", "family"], "无"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/背包数量/HBoxContainer/Label2", _build_bag_count_text())
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/人品值/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["luck", "luck_value", "morality"], "无"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/活力值/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["spirit", "vitality_value"], "0"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/战神积分/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["arena_points", "war_god_points"], "0"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/战神高级积分/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["arena_advanced_points", "war_god_advanced_points"], "无"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/爱心值/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["love_points", "heart_points"], "无"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/人缘值/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["popularity", "relationship_points"], "无"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/摆摊时长/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["stall_duration", "stall_seconds"], "无"))
	_set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/掌握配方/HBoxContainer/Label2", _snapshot_text(player_snapshot, ["recipe_count", "known_recipe_count"], "无"))


## 设置指定 Label 文案，路径不存在时跳过，降低场景细节调整带来的脚本崩溃风险。
func _set_label_text(root: Control, label_path: String, label_text: String) -> void:
	if root == null:
		return
	var target: Node = root.get_node_or_null(label_path)
	if target is Label:
		var label: Label = target as Label
		label.text = label_text


## 从快照中按候选字段顺序取第一个有效值，并统一转成 UI 文案。
func _snapshot_text(snapshot: Dictionary, keys: Array[String], default_text: String) -> String:
	for key: String in keys:
		if snapshot.has(key):
			return _format_value(snapshot.get(key, default_text))
	return default_text


## 从快照中读取百分比值，字段值不自带百分号时补上百分号。
func _snapshot_percent_text(snapshot: Dictionary, keys: Array[String], default_text: String) -> String:
	var text: String = _snapshot_text(snapshot, keys, default_text)
	if text.is_empty() or text.ends_with("%"):
		return text
	return "%s%%" % text


## 从快照中读取百分比值，并格式化为括号后缀。
func _snapshot_percent_suffix(snapshot: Dictionary, keys: Array[String], default_text: String) -> String:
	var text: String = _snapshot_percent_text(snapshot, keys, default_text)
	if text.is_empty():
		return ""
	return "(%s)" % text


## 构建经验后缀，展示「当前经验 / 本级升级所需总经验」；满级时显示满级。
func _build_exp_suffix(snapshot: Dictionary) -> String:
	var level: int = int(snapshot.get("level", 0))
	if level >= 100:
		return "（满级）"
	var exp_required: int = _resolve_level_exp_required(snapshot)
	if exp_required > 0:
		return "/%s" % _format_value(exp_required)
	return ""


## 计算当前等级升到下一级所需的总经验；服务端 exp_to_next = exp_required - exp。
func _resolve_level_exp_required(snapshot: Dictionary) -> int:
	var exp_current: int = int(snapshot.get("exp", 0))
	var exp_to_next: int = int(snapshot.get("exp_to_next", 0))
	if exp_to_next > 0:
		return exp_current + exp_to_next
	return exp_current


## 构建背包容量文案，优先使用服务端容器容量，缺失时退回当前物品数量。
func _build_bag_count_text() -> String:
	return "请打开背包查看"


## 将服务端字段格式化为 UI 文案；浮点数进入 UI 前统一转成整数。
func _format_value(value: Variant) -> String:
	if typeof(value) == TYPE_FLOAT:
		return str(int(value))
	if typeof(value) == TYPE_INT:
		return str(int(value))
	return UiFormat.value_to_text(value)
