extends CanvasLayer

signal menu_closed

## 默认打开的分页索引，保持面板每次展示时先进入战斗属性。
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

## 当前可切换的按钮集合。
var _tab_buttons: Array[Button] = []
## 当前可切换的内容面板集合。
var _tab_panels: Array[Control] = []
## 当前选中的分页索引。
var _current_tab_index: int = DEFAULT_TAB_INDEX


## 初始化按钮事件、订阅权威快照变化，并默认隐藏弹窗。
func _ready() -> void:
    hide()
    _tab_buttons.append(battle_attribute_button)
    _tab_buttons.append(status_resistance_button)
    _tab_buttons.append(social_attribute_button)
    _tab_panels.append(battle_attributes_panel)
    _tab_panels.append(status_resistance_panel)
    _tab_panels.append(social_attribute_panel)

    battle_attribute_button.pressed.connect(_on_tab_pressed.bind(0))
    status_resistance_button.pressed.connect(_on_tab_pressed.bind(1))
    social_attribute_button.pressed.connect(_on_tab_pressed.bind(2))
    if not GameState.session_changed.is_connected(refresh_panel_data):
        GameState.session_changed.connect(refresh_panel_data)
    if not GameState.world_snapshot_changed.is_connected(refresh_panel_data):
        GameState.world_snapshot_changed.connect(refresh_panel_data)
    if not GameState.bag_changed.is_connected(refresh_panel_data):
        GameState.bag_changed.connect(refresh_panel_data)
    if not GameState.wallet_changed.is_connected(refresh_panel_data):
        GameState.wallet_changed.connect(refresh_panel_data)

    reset_to_default()
    refresh_panel_data()


## 断开全局状态信号，避免面板销毁后继续收到回调。
func _exit_tree() -> void:
    if GameState.session_changed.is_connected(refresh_panel_data):
        GameState.session_changed.disconnect(refresh_panel_data)
    if GameState.world_snapshot_changed.is_connected(refresh_panel_data):
        GameState.world_snapshot_changed.disconnect(refresh_panel_data)
    if GameState.bag_changed.is_connected(refresh_panel_data):
        GameState.bag_changed.disconnect(refresh_panel_data)
    if GameState.wallet_changed.is_connected(refresh_panel_data):
        GameState.wallet_changed.disconnect(refresh_panel_data)


## 打开人物状态弹窗，并刷新为当前服务端权威快照。
func open_menu() -> void:
    show()
    reset_to_default()
    refresh_panel_data()


## 关闭人物状态弹窗，并通知主场景解除菜单锁定。
func close_menu() -> void:
    var was_visible: bool = visible
    hide()
    if was_visible:
        menu_closed.emit()


## 恢复默认选中项，供外部打开面板时主动调用。
func reset_to_default() -> void:
    _select_tab(DEFAULT_TAB_INDEX)


## 根据当前 GameState 中的服务端权威快照刷新所有展示字段。
func refresh_panel_data() -> void:
    var player: Dictionary = GameState.player_snapshot
    _refresh_battle_attributes(player)
    _refresh_status_resistance(player)
    _refresh_social_attribute(player)


## 响应玩家点击，切换到目标分页。
func _on_tab_pressed(index: int) -> void:
    _select_tab(index)


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


## 刷新战斗属性分页，所有数值均来自服务端玩家快照。
func _refresh_battle_attributes(player: Dictionary) -> void:
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/生命/HBoxContainer/Label2", _snapshot_text(player, ["hp"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/生命/HBoxContainer/Label4", _snapshot_text(player, ["hp_max"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/精力/HBoxContainer/Label2", _snapshot_text(player, ["vigor"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/精力/HBoxContainer/Label4", _snapshot_text(player, ["vigor_max"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/元素属性/HBoxContainer/Label2", _snapshot_text(player, ["element", "element_type", "element_name"], "无"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/经验/HBoxContainer/Label2", _snapshot_text(player, ["exp"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/经验/HBoxContainer/Label3", _build_exp_suffix(player))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/飞升/HBoxContainer/Label2", _snapshot_text(player, ["fly", "fly_value", "ascension"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/飞升/HBoxContainer/Label3", _snapshot_percent_suffix(player, ["fly_rate", "ascension_rate_pct"], "0%"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/转职/HBoxContainer/Label2", _snapshot_text(player, ["transfer", "transfer_value", "job_level"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/转职/HBoxContainer/Label3", _snapshot_text(player, ["transfer_state", "job_state"], ""))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/攻击和防御/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["atk", "attack"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/攻击和防御/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player, ["def", "defense"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/速度和法力/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["spd", "speed"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/速度和法力/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player, ["mana", "mp"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/命中和闪避/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["hit_pct", "hit"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/命中和闪避/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player, ["dodge_pct", "dodge"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/致命和爆伤/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["crit_rate_pct", "crit"], "0"))
    _set_label_text(battle_attributes_panel, "PanelContainer/VBoxContainer/致命和爆伤/HBoxContainer/HBoxContainer2/Label2", _snapshot_percent_text(player, ["crit_dmg_pct", "crit_damage"], "0%"))


## 刷新状态抗性分页，字段缺失时显示 0，避免继续展示旧假数据。
func _refresh_status_resistance(player: Dictionary) -> void:
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/物理攻击抗性/HBoxContainer/Label2", _snapshot_text(player, ["physical_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/物理攻击抗性/HBoxContainer/Label4", _snapshot_text(player, ["reverse_physical_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/技能攻击抗性/HBoxContainer/Label2", _snapshot_text(player, ["skill_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/技能攻击抗性/HBoxContainer/Label4", _snapshot_text(player, ["reverse_skill_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/混乱和昏睡/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["confusion_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/混乱和昏睡/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player, ["sleep_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/麻痹和封印/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["paralysis_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/麻痹和封印/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player, ["seal_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/诅咒/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["curse_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗致命和抗爆伤/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["crit_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗致命和抗爆伤/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player, ["crit_dmg_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗人物和抗宠物/HBoxContainer/HBoxContainer/Label2", _snapshot_text(player, ["character_resist_pct"], "0"))
    _set_label_text(status_resistance_panel, "PanelContainer/VBoxContainer/抗人物和抗宠物/HBoxContainer/HBoxContainer2/Label2", _snapshot_text(player, ["pet_resist_pct"], "0"))


## 刷新社会属性分页，优先读取玩家快照，背包数量来自当前背包容器快照。
func _refresh_social_attribute(player: Dictionary) -> void:
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/家族/HBoxContainer/Label2", _snapshot_text(player, ["guild_name", "family_name", "family"], "无"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/背包数量/HBoxContainer/Label2", _build_bag_count_text())
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/人品值/HBoxContainer/Label2", _snapshot_text(player, ["luck", "luck_value", "morality"], "无"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/活力值/HBoxContainer/Label2", _snapshot_text(player, ["spirit", "vitality_value"], "0"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/战神积分/HBoxContainer/Label2", _snapshot_text(player, ["arena_points", "war_god_points"], "0"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/战神高级积分/HBoxContainer/Label2", _snapshot_text(player, ["arena_advanced_points", "war_god_advanced_points"], "无"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/爱心值/HBoxContainer/Label2", _snapshot_text(player, ["love_points", "heart_points"], "无"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/人缘值/HBoxContainer/Label2", _snapshot_text(player, ["popularity", "relationship_points"], "无"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/摆摊时长/HBoxContainer/Label2", _snapshot_text(player, ["stall_duration", "stall_seconds"], "无"))
    _set_label_text(social_attribute_panel, "PanelContainer/VBoxContainer/掌握配方/HBoxContainer/Label2", _snapshot_text(player, ["recipe_count", "known_recipe_count"], "无"))


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


## 构建经验后缀，优先展示距离下一级所需经验，满级时显示满级。
func _build_exp_suffix(snapshot: Dictionary) -> String:
    var level: int = int(snapshot.get("level", 0))
    var exp_to_next: int = int(snapshot.get("exp_to_next", 0))
    if exp_to_next > 0:
        return "/%s" % _format_value(exp_to_next)
    if level >= 100:
        return "（满级）"
    return ""


## 构建背包容量文案，优先使用服务端容器容量，缺失时退回当前物品数量。
func _build_bag_count_text() -> String:
    var used_count: int = GameState.bag_items.size()
    var capacity: int = int(GameState.bag_container.get("capacity", GameState.bag_container.get("max_slots", 0)))
    if capacity > 0:
        return "%d/%d" % [used_count, capacity]
    return _format_value(used_count)


## 将服务端字段格式化为 UI 文案；浮点数进入 UI 前统一转成整数。
func _format_value(value: Variant) -> String:
    if typeof(value) == TYPE_FLOAT:
        return str(int(value))
    if typeof(value) == TYPE_INT:
        return str(int(value))
    return UiFormat.value_to_text(value)
