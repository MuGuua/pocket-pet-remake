extends Control

const UiFormat = preload("res://scripts/common/ui_format.gd")

# 战斗场景中技能标识到展示文案的映射表。
const SKILL_LABELS := {
	1001: "普通攻击",
	1002: "火花冲击",
	1003: "活力治愈",
	1004: "弧光连射",
	90001: "野性撞击",
	90002: "利爪突袭",
	90003: "野性回春",
}

# 战斗状态标识到简短中文名的映射表。
const STATUS_LABELS := {
	1: "流血",
	2: "封印",
	3: "眩晕",
}

# 本地仅把服务端命令阶段截止时间展示给玩家；真正超时补行动由服务端负责。
const COMMAND_TIMEOUT_SEC: float = 15.0

# 战斗标题标签。
@onready var title_label: Label = %TitleLabel
# 战斗摘要标签。
@onready var summary_label: Label = %SummaryLabel
# 当前逐条播放的事件横幅标签。
@onready var event_banner_label: Label = %EventBannerLabel
# 我方阵营标题标签。
@onready var ally_title_label: Label = %AllyTitleLabel
# 我方单位摘要标签。
@onready var ally_label: Label = %AllyLabel
# 敌方阵营标题标签。
@onready var enemy_title_label: Label = %EnemyTitleLabel
# 敌方单位摘要标签。
@onready var enemy_label: Label = %EnemyLabel
# 战斗详情标签。
@onready var detail_label: Label = %DetailLabel
# 提示文案标签。
@onready var hint_label: Label = %HintLabel
# 当前动作提交状态标签。
@onready var action_status_label: Label = %ActionStatusLabel
# 第一技能按钮。
@onready var primary_skill_button: Button = %PrimarySkillButton
# 第二技能按钮。
@onready var secondary_skill_button: Button = %SecondarySkillButton
# 切换目标按钮。
@onready var target_button: Button = %TargetButton
# 自动战斗按钮。
@onready var auto_battle_button: Button = %AutoBattleButton
# 逃跑按钮。
@onready var escape_button: Button = %EscapeButton

# 标记当前是否已有待服务端结算的动作。
var _is_action_pending: bool = false
# 缓存战斗控制器节点引用。
var _battle_controller: Node
# 缓存当前手动选中的敌方目标索引。
var _selected_enemy_index: int = 0
# 缓存当前手动选中的己方目标索引。
var _selected_ally_index: int = 0
# 记录当前玩家在技能栏中高亮的技能索引，用于切换目标时区分敌方目标和己方目标。
var _selected_skill_index: int = 0
# 标记当前是否开启服务端自动战斗；当前界面只做显示与开关，不再本地代投动作。
var _auto_battle_enabled: bool = false
# 记录当前命令阶段的唯一键，用于在切换到下一个可控单位时重置倒计时。
var _command_state_key: String = ""
# 记录当前手动选将阶段的本地截止时间（秒）。
var _command_deadline_sec: float = 0.0
# 记录当前尚未提交前的本地提示，用来表达“已选中技能，等待二次确认”。
var _selection_notice: String = ""
# 记录最近一次已经接入播放队列的事件批次键，避免同一批回合事件重复排队。
var _last_event_batch_key: String = ""
# 保存等待逐条播放的事件文案队列。
var _event_playback_queue: Array[String] = []
# 保存当前正在横幅区域展示的事件文案。
var _current_event_banner: String = ""
# 记录当前横幅文案的自动切换截止时间（秒）。
var _event_banner_until_sec: float = 0.0

# 初始化战斗场景显示，并绑定全局战斗状态变化信号。
func _ready() -> void:
	_battle_controller = get_node_or_null("../../BattleController")
	GameState.battle_changed.connect(_refresh_view)
	if _battle_controller != null and _battle_controller.has_signal("action_responded"):
		_battle_controller.connect("action_responded", Callable(self, "_on_action_responded"))
	_bind_buttons()
	set_process(true)
	_refresh_view()

# 退出战斗场景时注销已绑定的信号。
func _exit_tree() -> void:
	if GameState.battle_changed.is_connected(_refresh_view):
		GameState.battle_changed.disconnect(_refresh_view)
	if _battle_controller != null and _battle_controller.has_signal("action_responded") and _battle_controller.is_connected("action_responded", Callable(self, "_on_action_responded")):
		_battle_controller.disconnect("action_responded", Callable(self, "_on_action_responded"))

# 在每帧中维护事件横幅轮播；动作超时与自动托管都由服务端负责。
func _process(_delta: float) -> void:
	_update_event_banner()

# 按当前战斗快照刷新战斗界面全部显示内容。
func _refresh_view() -> void:
	_is_action_pending = false
	_auto_battle_enabled = bool(GameState.battle_state.get("auto_battle_enabled", false))
	_normalize_target_selection()
	_sync_command_deadline()
	_sync_event_playback()
	title_label.text = "战斗中" if GameState.is_in_battle else "战斗结算"

	# 读取当前战斗唯一标识、回合数和阶段用于顶部摘要展示。
	var battle_id := str(GameState.battle_state.get("battle_id", "未分配"))
	var round_text := str(GameState.battle_state.get("round", 0))
	var phase_text := str(GameState.battle_state.get("phase", "command"))
	summary_label.text = UiFormat.normalize_text("战斗ID: %s | 回合: %s | 阶段: %s" % [battle_id, round_text, phase_text])
	event_banner_label.text = UiFormat.normalize_text(_build_event_banner_text())
	_refresh_board_headers()

	ally_label.text = UiFormat.normalize_text(_build_group_text("allies"))
	enemy_label.text = UiFormat.normalize_text(_build_group_text("enemies"))
	detail_label.text = UiFormat.normalize_text(_build_event_log_text())
	hint_label.text = UiFormat.normalize_text(_build_hint_text())
	action_status_label.text = UiFormat.normalize_text(_build_action_status_text())
	_refresh_skill_buttons()
	_refresh_utility_buttons()

# 处理技能按钮点击；第一次点击用于切换当前高亮技能，重复点击当前技能时才真正提交动作。
func _on_skill_button_pressed(skill_index: int, skill_id: int) -> void:
	if _selected_skill_index != skill_index:
		_selected_skill_index = skill_index
		_selection_notice = "已选中 `%s`，请确认目标后再次点击技能提交。" % _skill_label(skill_id)
		_refresh_view()
		return
	_submit_skill_action(skill_id, "已提交 `%s` 指令，等待服务端确认。")

# 处理切换目标按钮点击，在存活敌人之间轮换默认目标。
func _on_target_button_pressed() -> void:
	_selection_notice = ""
	if _current_skill_target_type() == "ally_single":
		var allies := _living_allies()
		if allies.size() <= 1:
			return
		_selected_ally_index = (_selected_ally_index + 1) % allies.size()
	else:
		var enemies := _living_enemies()
		if enemies.size() <= 1:
			return
		_selected_enemy_index = (_selected_enemy_index + 1) % enemies.size()
	_refresh_view()

# 切换自动战斗开关；当前由服务端托管剩余动作，不再由客户端本地代投。
func _on_auto_battle_button_pressed() -> void:
	var next_enabled := not _auto_battle_enabled
	_selection_notice = ""
	App.set_battle_auto(
		int(GameState.battle_state.get("battle_id", 0)),
		int(GameState.battle_state.get("round", 1)),
		next_enabled
	)
	action_status_label.text = "已请求%s服务端自动战斗。" % ("开启" if next_enabled else "关闭")

# 处理逃跑按钮点击，交给服务端执行 PVE 逃跑判定。
func _on_escape_button_pressed() -> void:
	var ally := GameState.active_battle_actor("allies")
	if ally.is_empty() or _is_action_pending:
		return
	_is_action_pending = true
	_selection_notice = ""
	action_status_label.text = "已提交逃跑请求，等待服务端处理。"
	_refresh_skill_buttons()
	_refresh_utility_buttons()
	App.submit_battle_action(
		int(GameState.battle_state.get("battle_id", 0)),
		int(GameState.battle_state.get("round", 1)),
		int(ally.get("actor_id", 0)),
		0,
		4,
		0
	)

# 处理战斗动作回执，并在服务端拒绝时恢复按钮可用性。
func _on_action_responded(accepted: bool, reason: String) -> void:
	if accepted:
		return
	_is_action_pending = false
	_selection_notice = ""
	action_status_label.text = "服务端拒绝动作: %s" % reason
	_refresh_skill_buttons()
	_refresh_utility_buttons()

# 读取当前阵营下全部存活敌人，供默认目标和切换目标逻辑使用。
func _living_enemies() -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	var enemies_variant: Variant = GameState.battle_state.get("enemies", [])
	if enemies_variant is not Array:
		return result
	for enemy_variant in enemies_variant:
		if enemy_variant is Dictionary:
			var enemy: Dictionary = enemy_variant
			var state := _actor_state(enemy)
			if not bool(state.get("dead", false)):
				result.append(enemy)
	return result

# 读取当前阵营下全部存活己方单位，供治疗类技能的手动目标选择逻辑使用。
func _living_allies() -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	var allies_variant: Variant = GameState.battle_state.get("allies", [])
	if allies_variant is not Array:
		return result
	for ally_variant in allies_variant:
		if ally_variant is Dictionary:
			var ally: Dictionary = ally_variant
			var state := _actor_state(ally)
			if not bool(state.get("dead", false)):
				result.append(ally)
	return result

# 返回当前选中的敌方目标快照。
func _selected_enemy_actor() -> Dictionary:
	var enemies := _living_enemies()
	if enemies.is_empty():
		return {}
	if _selected_enemy_index >= enemies.size():
		_selected_enemy_index = 0
	return enemies[_selected_enemy_index]

# 返回当前选中的己方目标快照；治疗技能默认锁定血量比例最低的己方单位。
func _selected_ally_actor() -> Dictionary:
	var allies := _living_allies()
	if allies.is_empty():
		return {}
	if _selected_ally_index >= allies.size():
		_selected_ally_index = 0
	return allies[_selected_ally_index]

# 计算当前存活己方单位里血量比例最低的索引，用于每次进入新的选将阶段时重置治疗默认目标。
func _lowest_hp_ally_index() -> int:
	var allies := _living_allies()
	if allies.is_empty():
		return 0
	var best_index := 0
	var best_ratio := 2.0
	for index in range(allies.size()):
		var ally := allies[index]
		var state := _actor_state(ally)
		var hp := float(int(state.get("hp", ally.get("hp", 0))))
		var hp_max := maxf(float(int(state.get("hp_max", ally.get("hp_max", 1)))), 1.0)
		var ratio := hp / hp_max
		if ratio < best_ratio:
			best_ratio = ratio
			best_index = index
	return best_index

# 把当前目标索引限制在仍然有效的敌方范围内，避免目标死亡后越界。
func _normalize_target_selection() -> void:
	var enemies := _living_enemies()
	if enemies.is_empty():
		_selected_enemy_index = 0
		return
	if _selected_enemy_index >= enemies.size():
		_selected_enemy_index = 0
	var allies := _living_allies()
	if allies.is_empty():
		_selected_ally_index = 0
		return
	if _selected_ally_index >= allies.size():
		_selected_ally_index = 0

# 根据单位标识从战斗状态列表中查找实时属性快照。
func _actor_state(actor: Dictionary) -> Dictionary:
	if actor.is_empty():
		return {}
	var actor_id := int(actor.get("actor_id", 0))
	var states_variant: Variant = GameState.battle_state.get("actors", [])
	if states_variant is Array:
		for state_variant in states_variant:
			if state_variant is Dictionary and int(state_variant.get("actor_id", 0)) == actor_id:
				return state_variant
	return {}

# 组装当前阵营中全部单位的多行摘要文案。
func _build_group_text(group_key: String) -> String:
	var actors_variant: Variant = GameState.battle_state.get(group_key, [])
	if actors_variant is not Array or actors_variant.is_empty():
		return "未同步"

	var selected_enemy := _selected_enemy_actor()
	var selected_ally := _selected_ally_actor()
	var rows: Array[String] = []
	for actor_variant in actors_variant:
		if actor_variant is Dictionary:
			var actor: Dictionary = actor_variant
			var is_selected_target := false
			if group_key == "enemies" and not selected_enemy.is_empty() and int(actor.get("actor_id", 0)) == int(selected_enemy.get("actor_id", 0)):
				is_selected_target = true
			elif group_key == "allies" and _current_skill_target_type() == "ally_single" and not selected_ally.is_empty() and int(actor.get("actor_id", 0)) == int(selected_ally.get("actor_id", 0)):
				is_selected_target = true
			rows.append(_build_actor_text(actor, _actor_state(actor), is_selected_target, group_key))
	return "\n".join(rows)

# 组装单个战斗单位展示文案，并在当前行动者或目标上附加标记。
func _build_actor_text(actor: Dictionary, state: Dictionary, selected_target: bool = false, group_key: String = "allies") -> String:
	if actor.is_empty():
		return "未同步"
	var hp := int(state.get("hp", actor.get("hp", 0)))
	var hp_max := int(state.get("hp_max", actor.get("hp_max", 0)))
	var tags: Array[String] = []
	if int(actor.get("actor_id", 0)) == int(GameState.battle_state.get("active_actor_id", 0)) and GameState.is_in_battle:
		tags.append("当前选择")
	if selected_target:
		tags.append("目标")
	if bool(state.get("dead", false)):
		tags.append("倒下")
	var status_text := _format_status_list(state.get("status_ids", actor.get("status_ids", [])))
	if not status_text.is_empty():
		tags.append(status_text)
	var suffix := ""
	if not tags.is_empty():
		suffix = " [%s]" % ", ".join(tags)
	return "%s %s%s\nHP %d/%d %s%s" % [_actor_marker(actor, selected_target, group_key), _actor_kind_label(actor), str(actor.get("name", "未知")), hp, hp_max, _build_hp_bar(hp, hp_max), suffix]

# 刷新左右阵营面板标题，让当前轮到选择的阵营标题更醒目。
func _refresh_board_headers() -> void:
	if ally_title_label != null:
		ally_title_label.text = "我方阵营 <-"
	if enemy_title_label != null:
		enemy_title_label.text = "-> 敌方阵营"

# 返回当前单位在战斗面板中的简洁标记，帮助玩家快速识别当前选择和目标。
func _actor_marker(actor: Dictionary, selected_target: bool, group_key: String) -> String:
	var actor_id := int(actor.get("actor_id", 0))
	if actor_id == int(GameState.battle_state.get("active_actor_id", 0)) and group_key == "allies" and GameState.is_in_battle:
		return ">>"
	if selected_target:
		return "!!"
	if bool(_actor_state(actor).get("dead", false)):
		return "xx"
	return "--"

# 给人物 / 宠物 / 怪物补一个紧凑前缀，方便移动端在同一列表里快速识别当前行动主体。
func _actor_kind_label(actor: Dictionary) -> String:
	match int(actor.get("unit_class", 0)):
		1:
			return "[人] "
		2:
			return "[宠] "
		4:
			return "[怪] "
		_:
			return ""

# 读取并格式化最近几条服务端结算事件，便于在移动端小面板中快速查看回合结果。
func _build_event_log_text() -> String:
	if not GameState.is_in_battle:
		var result_logs: Array[String] = []
		var reward_gold := int(GameState.battle_state.get("reward_gold", 0))
		var reward_player_exp := int(GameState.battle_state.get("reward_player_exp", 0))
		if reward_gold > 0 or reward_player_exp > 0:
			result_logs.append("本场奖励: %d 金币 / %d 角色经验" % [reward_gold, reward_player_exp])
		var drop_texts_variant: Variant = GameState.battle_state.get("drop_texts", [])
		if drop_texts_variant is Array:
			for drop_text_variant in drop_texts_variant:
				var drop_text := str(drop_text_variant)
				if not drop_text.is_empty():
					result_logs.append(drop_text)
		if not result_logs.is_empty():
			return "\n".join(result_logs)
	var events_variant: Variant = GameState.battle_state.get("events", [])
	if events_variant is not Array or events_variant.is_empty():
		return "等待服务端同步战斗详情。"

	var logs: Array[String] = []
	var start_index := maxi(events_variant.size() - 4, 0)
	for index in range(start_index, events_variant.size()):
		var event_variant: Variant = events_variant[index]
		if event_variant is Dictionary:
			logs.append(_format_event(event_variant))
	return "\n".join(logs)

# 返回当前应显示在横幅区域中的事件文案；优先展示逐条播放中的当前事件，没有则退回到阶段提示。
func _build_event_banner_text() -> String:
	if not _current_event_banner.is_empty():
		return ">> " + _current_event_banner
	if not GameState.is_in_battle:
		return "战斗结算已完成。"
	return "等待本回合事件播放。"

# 把服务端战斗事件快照转为战斗详情文案。
func _format_event(event_payload: Dictionary) -> String:
	var label := str(event_payload.get("label", ""))
	if not label.is_empty():
		return label
	var event_type := int(event_payload.get("event_type", 0))
	var value := int(event_payload.get("value", 0))
	var skill_label := _skill_label(int(event_payload.get("skill_id", 0)))
	match event_type:
		1:
			return "服务端已执行 `%s`。" % skill_label
		2:
			return "服务端用 `%s` 结算伤害 %d。" % [skill_label, value]
		3:
			return "服务端用 `%s` 结算治疗 %d。" % [skill_label, value]
		4:
			return "服务端同步了状态命中。"
		5:
			return "服务端结算了持续伤害 %d。" % value
		6:
			return "单位本回合无法行动。"
		7:
			return "有单位被击倒。"
		8:
			return "服务端判定本次攻击被闪避。"
		9:
			return "服务端触发了一次反击。"
		10:
			return "服务端判定单位触发了复活。"
		11:
			return "服务端触发了一次连击。"
		_:
			return "服务端同步了新的战斗事件。"

# 把最新一批服务端事件装入本地播放队列，后续由 `_process()` 逐条轮播。
func _sync_event_playback() -> void:
	var battle_version := int(GameState.battle_state.get("battle_version", 0))
	var events_variant: Variant = GameState.battle_state.get("events", [])
	if events_variant is not Array or events_variant.is_empty():
		if not GameState.is_in_battle:
			_current_event_banner = ""
			_event_playback_queue.clear()
			_last_event_batch_key = ""
		return
	var batch_key := "%d:%d" % [battle_version, events_variant.size()]
	if batch_key == _last_event_batch_key:
		return
	_last_event_batch_key = batch_key
	_event_playback_queue.clear()
	for event_variant in events_variant:
		if event_variant is Dictionary:
			_event_playback_queue.append(_format_event(event_variant))
	_current_event_banner = ""
	_event_banner_until_sec = 0.0

# 在每帧中推动事件横幅轮播，让一批事件能按顺序短暂停留而不是一次性滚过去。
func _update_event_banner() -> void:
	var now_sec := _now_sec()
	if not _current_event_banner.is_empty() and now_sec < _event_banner_until_sec:
		return
	if _event_playback_queue.is_empty():
		if not GameState.is_in_battle:
			_current_event_banner = ""
		return
	_current_event_banner = _event_playback_queue.pop_front()
	_event_banner_until_sec = now_sec + 0.9
	if event_banner_label != null:
		event_banner_label.text = UiFormat.normalize_text(_current_event_banner)

# 组合当前回合提示文本，帮助玩家理解还需要给多少单位下指令。
func _build_hint_text() -> String:
	if not GameState.is_in_battle:
		return "收到战斗结果，正在返回世界场景。"

	var pending_variant: Variant = GameState.battle_state.get("pending_actor_ids", [])
	var pending_count: int = pending_variant.size() if pending_variant is Array else 0
	var target_name := "未锁定"
	if _current_skill_target_type() == "ally_single":
		var ally_target := _selected_ally_actor()
		if not ally_target.is_empty():
			target_name = str(ally_target.get("name", "未锁定"))
	else:
		var enemy_target := _selected_enemy_actor()
		if not enemy_target.is_empty():
			target_name = str(enemy_target.get("name", "未锁定"))
	var mode_text := "自动" if _auto_battle_enabled else "手动"
	return "模式: %s | 技能: %s | 当前锁定目标: %s | 本回合待选择单位: %d | 服务端剩余 %s 秒" % [
		mode_text,
		_current_selected_skill_name(),
		target_name,
		pending_count,
		UiFormat.value_to_text(_remaining_command_sec()),
	]

# 组合当前操作状态文本，并区分提交中、等待中和已结束三种状态。
func _build_action_status_text() -> String:
	if not GameState.is_in_battle:
		return "战斗已结束。"
	if _is_action_pending:
		return "指令已发送，等待服务端回执。"
	if not _selection_notice.is_empty():
		return _selection_notice
	var active_actor := GameState.active_battle_actor("allies")
	if active_actor.is_empty():
		return "等待服务端推进下一阶段。"
	if _auto_battle_enabled:
		return "服务端自动战斗已开启，将托管 `%s` 的后续动作。" % str(active_actor.get("name", "未知单位"))
	return "请为 `%s` 选择动作。" % str(active_actor.get("name", "未知单位"))

# 为技能与辅助按钮绑定统一回调。
func _bind_buttons() -> void:
	if primary_skill_button != null:
		primary_skill_button.pressed.connect(func() -> void:
			_on_skill_button_pressed(int(primary_skill_button.get_meta("skill_index", 0)), int(primary_skill_button.get_meta("skill_id", 0)))
		)
	if secondary_skill_button != null:
		secondary_skill_button.pressed.connect(func() -> void:
			_on_skill_button_pressed(int(secondary_skill_button.get_meta("skill_index", 1)), int(secondary_skill_button.get_meta("skill_id", 0)))
		)
	if target_button != null:
		target_button.pressed.connect(_on_target_button_pressed)
	if auto_battle_button != null:
		auto_battle_button.pressed.connect(_on_auto_battle_button_pressed)
	if escape_button != null:
		escape_button.pressed.connect(_on_escape_button_pressed)

# 按当前出战单位技能列表刷新技能按钮显示，并把服务端下发的技能目标规则绑定到按钮元数据。
func _refresh_skill_buttons() -> void:
	var ally := GameState.active_battle_actor("allies")
	var skills := _actor_skills(ally)
	if _selected_skill_index >= skills.size():
		_selected_skill_index = 0
	_apply_skill_button(primary_skill_button, skills, 0)
	_apply_skill_button(secondary_skill_button, skills, 1)

# 按索引把指定技能配置应用到某个按钮上。
func _apply_skill_button(button: Button, skills: Array[Dictionary], index: int) -> void:
	if button == null:
		return
	if index >= skills.size():
		button.visible = false
		button.disabled = true
		button.set_meta("skill_id", 0)
		button.set_meta("target_type", "")
		return

	var skill: Dictionary = skills[index]
	var skill_id := int(skill.get("skill_id", 0))
	var target_type := str(skill.get("target_type", "enemy_single"))
	var selected_suffix := " *" if index == _selected_skill_index else ""
	button.visible = true
	button.text = "%s%s" % [_skill_label(skill_id, skill), selected_suffix]
	button.set_meta("skill_id", skill_id)
	button.set_meta("skill_index", index)
	button.set_meta("target_type", target_type)
	button.disabled = not _can_submit_manual_action()

# 按当前阶段刷新切换目标、自动战斗和逃跑按钮可用性。
func _refresh_utility_buttons() -> void:
	if target_button != null:
		var current_target_type := _current_skill_target_type()
		var target_count := _living_allies().size() if current_target_type == "ally_single" else _living_enemies().size()
		target_button.text = "切友方" if _current_skill_target_type() == "ally_single" else "切换目标"
		target_button.disabled = not _can_submit_manual_action() or current_target_type == "enemy_all" or target_count <= 1
	if auto_battle_button != null:
		auto_battle_button.text = "自动中" if _auto_battle_enabled else "自动战斗"
		auto_battle_button.disabled = not GameState.is_in_battle
	if escape_button != null:
		escape_button.disabled = not _can_submit_manual_action()

# 判断当前是否允许继续提交手动动作。
func _can_submit_manual_action() -> bool:
	return GameState.is_in_battle and not _is_action_pending and str(GameState.battle_state.get("phase", "command")) == "command" and not GameState.active_battle_actor("allies").is_empty()

# 返回当前高亮技能的目标类型，由服务端技能快照驱动。
func _current_skill_target_type() -> String:
	var active_actor := GameState.active_battle_actor("allies")
	var skills := _actor_skills(active_actor)
	if skills.is_empty():
		return "enemy_single"
	if _selected_skill_index >= skills.size():
		_selected_skill_index = 0
	return str(skills[_selected_skill_index].get("target_type", "enemy_single"))

# 统一提交当前单位的技能动作；治疗类技能默认选择血量比例最低的己方单位。
func _submit_skill_action(skill_id: int, status_template: String) -> void:
	var ally := GameState.active_battle_actor("allies")
	if ally.is_empty() or _is_action_pending:
		action_status_label.text = "缺少可行动单位。"
		return

	var target: Dictionary = {}
	if _skill_target_type_by_id(ally, skill_id) == "ally_single":
		target = _selected_ally_actor()
	else:
		target = _selected_enemy_actor()
	if target.is_empty():
		action_status_label.text = "缺少可用战斗目标。"
		return

	_is_action_pending = true
	_selection_notice = ""
	action_status_label.text = status_template % _skill_label(skill_id)
	_refresh_skill_buttons()
	_refresh_utility_buttons()
	App.submit_battle_action(
		int(GameState.battle_state.get("battle_id", 0)),
		int(GameState.battle_state.get("round", 1)),
		int(ally.get("actor_id", 0)),
		int(target.get("actor_id", 0)),
		1,
		skill_id
	)

# 为当前可控单位提交默认动作；优先使用第一个技能，缺失时回退到普通攻击。
func _submit_default_action_for_current_actor(status_message: String) -> void:
	var active_actor := GameState.active_battle_actor("allies")
	if active_actor.is_empty() or _is_action_pending:
		return
	var skill_id := 1001
	var skills := _actor_skills(active_actor)
	if not skills.is_empty():
		skill_id = int(skills[0].get("skill_id", 1001))
	_submit_skill_action(skill_id, status_message + " 使用 `%s`。")

# 根据服务端战斗阶段与当前可控单位，重建本地展示用的倒计时窗口。
func _sync_command_deadline() -> void:
	if not GameState.is_in_battle or str(GameState.battle_state.get("phase", "command")) != "command":
		_command_state_key = ""
		_command_deadline_sec = 0.0
		return

	var next_key := "%s:%s:%s" % [
		str(GameState.battle_state.get("battle_id", 0)),
		str(GameState.battle_state.get("round", 0)),
		str(GameState.battle_state.get("active_actor_id", 0)),
	]
	if next_key == _command_state_key:
		return

	_command_state_key = next_key
	var server_deadline_ms := int(GameState.battle_state.get("command_deadline_ms", 0))
	if server_deadline_ms > 0:
		_command_deadline_sec = float(server_deadline_ms) / 1000.0
	else:
		var now_sec := _now_sec()
		_command_deadline_sec = now_sec + COMMAND_TIMEOUT_SEC
	_selected_ally_index = _lowest_hp_ally_index()
	_selection_notice = ""

# 计算当前命令阶段剩余秒数，供 HUD 提示区显示。
func _remaining_command_sec() -> float:
	if _command_deadline_sec <= 0.0:
		return 0.0
	return maxf(_command_deadline_sec - _now_sec(), 0.0)

# 提取指定战斗单位的技能快照列表；优先使用服务端新下发的 `skills`，再兼容旧版 `skill_ids`。
func _actor_skills(actor: Dictionary) -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	if actor.is_empty():
		return result
	var skills_variant: Variant = actor.get("skills", [])
	if skills_variant is Array and not skills_variant.is_empty():
		for skill_variant in skills_variant:
			if skill_variant is Dictionary:
				result.append(skill_variant)
		return result
	var skill_ids_variant: Variant = actor.get("skill_ids", [])
	if skill_ids_variant is Array:
		for skill_id_variant in skill_ids_variant:
			result.append({
				"skill_id": int(skill_id_variant),
				"name": "",
				"target_type": "enemy_single",
			})
	return result

# 根据技能标识查找当前行动单位的目标类型，便于提交动作时选择正确阵营。
func _skill_target_type_by_id(actor: Dictionary, skill_id: int) -> String:
	for skill in _actor_skills(actor):
		if int(skill.get("skill_id", 0)) == skill_id:
			return str(skill.get("target_type", "enemy_single"))
	return "enemy_single"

# 返回当前高亮技能的名称，便于提示栏展示当前操作上下文。
func _current_selected_skill_name() -> String:
	var active_actor := GameState.active_battle_actor("allies")
	var skills := _actor_skills(active_actor)
	if skills.is_empty():
		return _skill_label(1001)
	if _selected_skill_index >= skills.size():
		_selected_skill_index = 0
	return _skill_label(int(skills[_selected_skill_index].get("skill_id", 1001)), skills[_selected_skill_index])

# 返回指定技能标识对应的展示文案；优先使用服务端下发的技能名，避免客户端维护重复表。
func _skill_label(skill_id: int, skill_payload: Dictionary = {}) -> String:
	var skill_name := str(skill_payload.get("name", ""))
	if not skill_name.is_empty():
		return "%s%s" % [skill_name, _skill_target_badge(str(skill_payload.get("target_type", "enemy_single")), int(skill_payload.get("target_count", 1)))]
	return "%s%s" % [str(SKILL_LABELS.get(skill_id, "技能%d" % skill_id)), _skill_target_badge(str(skill_payload.get("target_type", "enemy_single")), int(skill_payload.get("target_count", 1)))]

# 返回技能目标类型对应的紧凑徽标，方便移动端快速识别这是打敌人还是指向己方。
func _skill_target_badge(target_type: String, target_count: int = 1) -> String:
	if target_type == "ally_single":
		return " [友]"
	if target_type == "enemy_all":
		return " [敌全]"
	if target_type == "enemy_multi":
		return " [敌%d]" % maxi(target_count, 2)
	return " [敌]"

# 把当前生命转换成简洁的文本血条，方便在移动端列表里快速比较残血单位。
func _build_hp_bar(hp: int, hp_max: int) -> String:
	var safe_hp_max := maxi(hp_max, 1)
	var filled_count := int(round(float(hp) / float(safe_hp_max) * 6.0))
	filled_count = clampi(filled_count, 0, 6)
	return "[%s%s]" % ["#".repeat(filled_count), "-".repeat(6 - filled_count)]

# 把状态标识数组转为短文本，避免移动端面板被长句撑开。
func _format_status_list(status_ids_variant: Variant) -> String:
	if status_ids_variant is not Array:
		return ""
	var labels: Array[String] = []
	for status_id_variant in status_ids_variant:
		labels.append(str(STATUS_LABELS.get(int(status_id_variant), "状态%d" % int(status_id_variant))))
	return "/".join(labels)

# 统一读取当前系统时间，避免各处重复书写引擎接口。
func _now_sec() -> float:
	return float(Time.get_ticks_msec()) / 1000.0
