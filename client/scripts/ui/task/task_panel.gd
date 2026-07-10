extends RuntimeRootPanel
class_name TaskPanel

## 通用任务卡片场景；列表按服务端任务数量逐个实例化该场景。
const TASK_CARD_SCENE: PackedScene = preload("res://scenes/ui/task/task_list.tscn")

## 面板打开前拉取任务列表的流程代次，避免快速重复点击时旧请求打开新面板。
var _open_load_generation: int = 0
## 卡片实例 ID 到当前绑定任务快照的映射。
var _quest_by_card_id: Dictionary = {}
## 当前正在等待的任务操作请求序列号。
var _pending_action_seq: int = 0

## 关闭按钮，玩家点击后关闭整个任务面板。
@onready var close_button: BaseButton = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/Title/HBoxContainer/PanelCloseButton") as BaseButton
## 主线任务标签按钮。
@onready var main_task_button: BaseButton = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/任务分类/PanelContainer/HBoxContainer/Button") as BaseButton
## 支线任务标签按钮。
@onready var side_task_button: BaseButton = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/任务分类/PanelContainer/HBoxContainer/Button2") as BaseButton
## 日常任务标签按钮。
@onready var daily_task_button: BaseButton = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/任务分类/PanelContainer/HBoxContainer/Button3") as BaseButton
## 主线任务列表容器。
@onready var main_task_list: Control = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/主线任务列表") as Control
## 支线任务列表容器。
@onready var side_task_list: Control = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/支线任务列表") as Control
## 日常任务列表容器。
@onready var daily_task_list: Control = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/日常任务列表") as Control
## 主线任务滚动列表内容容器。
@onready var main_task_card_box: VBoxContainer = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/主线任务列表/PanelContainer/VBoxContainer/任务列表/ScrollContainer/TaskCardVBox") as VBoxContainer
## 支线任务滚动列表内容容器。
@onready var side_task_card_box: VBoxContainer = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/支线任务列表/PanelContainer/VBoxContainer/任务列表/ScrollContainer/TaskCardVBox") as VBoxContainer
## 日常任务滚动列表内容容器。
@onready var daily_task_card_box: VBoxContainer = get_node_or_null("RootPanel/MarginContainer/VBoxContainer/日常任务列表/PanelContainer/VBoxContainer/任务列表/ScrollContainer/TaskCardVBox") as VBoxContainer


## 初始化任务面板按钮事件，并默认展示主线任务页。
func _ready() -> void:
    super._ready()
    if close_button != null and not close_button.pressed.is_connected(close_menu):
        close_button.pressed.connect(close_menu)
    if main_task_button != null and not main_task_button.pressed.is_connected(_show_main_tasks):
        main_task_button.pressed.connect(_show_main_tasks)
    if side_task_button != null and not side_task_button.pressed.is_connected(_show_side_tasks):
        side_task_button.pressed.connect(_show_side_tasks)
    if daily_task_button != null and not daily_task_button.pressed.is_connected(_show_daily_tasks):
        daily_task_button.pressed.connect(_show_daily_tasks)
    if not GameState.quests_changed.is_connected(_refresh_task_lists):
        GameState.quests_changed.connect(_refresh_task_lists)
    _show_main_tasks()
    _refresh_task_lists()


## 退出场景树时断开全局任务刷新信号，避免已释放面板继续被回调。
func _exit_tree() -> void:
    if GameState.quests_changed.is_connected(_refresh_task_lists):
        GameState.quests_changed.disconnect(_refresh_task_lists)


## 面板展示前拉取服务端最新任务列表；自身保持隐藏，由主场景负责 loading。
func prepare_open_data() -> bool:
    _open_load_generation += 1
    var load_id: int = _open_load_generation
    if not GameState.is_ws_authenticated:
        return load_id == _open_load_generation
    var quest_seq: int = App.request_quest_list()
    if quest_seq <= 0:
        return false
    var succeeded: bool = await _wait_quest_list_request(quest_seq)
    return load_id == _open_load_generation and succeeded


## 数据就绪后打开任务面板，并保证默认停留在主线任务页。
func open_menu() -> void:
    super.open_menu()
    _show_main_tasks()
    _refresh_task_lists()


## 等待指定任务列表请求回包，避免打开时先展示旧任务数据。
func _wait_quest_list_request(expected_seq: int) -> bool:
    while expected_seq > 0:
        var result: Array = await App.request_finished
        if result.size() < 5:
            continue
        var request_cmd: int = int(result[0])
        var seq: int = int(result[1])
        if request_cmd != CommandIds.QUEST_LIST_REQ or seq != expected_seq:
            continue
        return bool(result[2])
    return false


## 展示主线任务列表。
func _show_main_tasks() -> void:
    _set_active_task_page(main_task_button, main_task_list)


## 展示支线任务列表。
func _show_side_tasks() -> void:
    _set_active_task_page(side_task_button, side_task_list)


## 展示日常任务列表。
func _show_daily_tasks() -> void:
    _set_active_task_page(daily_task_button, daily_task_list)


## 切换当前显示的任务分页，并同步按钮按下状态。
func _set_active_task_page(active_button: BaseButton, active_list: Control) -> void:
    var buttons: Array = [main_task_button, side_task_button, daily_task_button]
    for button_variant: Variant in buttons:
        var button: BaseButton = button_variant as BaseButton
        if button != null:
            button.set_pressed_no_signal(button == active_button)
    var lists: Array = [main_task_list, side_task_list, daily_task_list]
    for list_variant: Variant in lists:
        var task_list: Control = list_variant as Control
        if task_list != null:
            task_list.visible = task_list == active_list


## 按当前 GameState 中的服务端任务快照重建三类任务列表。
func _refresh_task_lists() -> void:
    _rebuild_task_cards(main_task_card_box, _quests_by_type("MAIN"))
    _rebuild_task_cards(side_task_card_box, _quests_by_type("SIDE"))
    _rebuild_task_cards(daily_task_card_box, _quests_by_type("DAILY"))


## 从本地权威快照中筛出指定类型且需要展示的任务。
func _quests_by_type(quest_type: String) -> Array[Dictionary]:
    var result: Array[Dictionary] = []
    for quest_variant: Variant in GameState.quests:
        if quest_variant is not Dictionary:
            continue
        var quest: Dictionary = (quest_variant as Dictionary).duplicate(true)
        if str(quest.get("quest_type", "")).to_upper() != quest_type:
            continue
        var state: String = str(quest.get("state", ""))
        if state == "LOCKED" or state == "COMPLETED":
            continue
        result.append(quest)
    return result


## 清空并按任务数量实例化卡片；一个服务端任务对应一个卡片。
func _rebuild_task_cards(container: VBoxContainer, quests: Array[Dictionary]) -> void:
    if container == null:
        return
    for child: Node in container.get_children():
        _quest_by_card_id.erase(child.get_instance_id())
        container.remove_child(child)
        child.queue_free()
    for quest: Dictionary in quests:
        var card: Control = TASK_CARD_SCENE.instantiate() as Control
        if card == null:
            continue
        container.add_child(card)
        _quest_by_card_id[card.get_instance_id()] = quest.duplicate(true)
        _connect_task_card_buttons(card)
        _apply_quest_to_card(card, quest)


## 绑定卡片操作按钮，所有任务操作统一回到服务端权威协议。
func _connect_task_card_buttons(card: Control) -> void:
    var action_button: BaseButton = _task_action_button(card)
    if action_button != null and not action_button.pressed.is_connected(_on_task_card_action_pressed.bind(card)):
        action_button.pressed.connect(_on_task_card_action_pressed.bind(card))
    var detail_button: BaseButton = _task_detail_button(card)
    if detail_button != null and not detail_button.pressed.is_connected(_on_task_card_detail_pressed.bind(card)):
        detail_button.pressed.connect(_on_task_card_detail_pressed.bind(card))


## 把单个任务快照写入卡片的标题、描述、进度和按钮文案。
func _apply_quest_to_card(card: Control, quest: Dictionary) -> void:
    var icon_node: TextureRect = _task_icon(card)
    if icon_node != null:
        icon_node.texture = TaskIcons.resolve_texture(_task_client_icon_id(quest))
        icon_node.visible = true
    var title_label: Label = _task_title_label(card)
    if title_label != null:
        title_label.text = UiFormat.normalize_text(str(quest.get("title", "任务")))
    var description_label: RichTextLabel = _task_description_label(card)
    if description_label != null:
        RichTextContent.apply_bbcode_text(description_label, _build_task_description(quest))
    var progress: Dictionary = _resolve_task_progress(quest)
    var progress_bar: ProgressBar = _task_progress_bar(card)
    if progress_bar != null:
        progress_bar.min_value = 0.0
        progress_bar.max_value = float(maxi(1, int(progress.get("target", 1))))
        progress_bar.value = float(int(progress.get("current", 0)))
        progress_bar.step = 1.0
    var progress_label: Label = _task_progress_label(card)
    if progress_label != null:
        progress_label.text = "%s/%s" % [
            UiFormat.value_to_text(int(progress.get("current", 0))),
            UiFormat.value_to_text(int(progress.get("target", 1))),
        ]
    var action_button: BaseButton = _task_action_button(card)
    if action_button != null:
        var action_label: String = _action_label_for_quest(quest)
        action_button.text = action_label
        action_button.disabled = action_label.is_empty() or _pending_action_seq > 0
    var detail_button: BaseButton = _task_detail_button(card)
    if detail_button != null:
        detail_button.disabled = false


## 生成任务描述：优先展示当前目标，保留任务描述作为补充。
func _build_task_description(quest: Dictionary) -> String:
    var objective: Dictionary = _current_objective(quest)
    var objective_text: String = UiFormat.normalize_text(str(objective.get("description", "")))
    if not objective_text.is_empty():
        return objective_text
    return UiFormat.normalize_text(str(quest.get("description", "")))


## 解析当前任务进度，默认选择第一个未完成目标。
func _resolve_task_progress(quest: Dictionary) -> Dictionary:
    var objective: Dictionary = _current_objective(quest)
    return {
        "current": int(objective.get("current", 0)),
        "target": maxi(1, int(objective.get("target", 1))),
    }


## 返回第一个未完成目标；若全部完成则返回最后一个目标。
func _current_objective(quest: Dictionary) -> Dictionary:
    var objectives_variant: Variant = quest.get("objectives", [])
    if objectives_variant is not Array:
        return {}
    var objectives: Array = objectives_variant as Array
    var fallback: Dictionary = {}
    for objective_variant: Variant in objectives:
        if objective_variant is not Dictionary:
            continue
        var objective: Dictionary = objective_variant as Dictionary
        fallback = objective
        if not bool(objective.get("completed", false)):
            return objective
    return fallback


## 根据任务状态确定卡片主按钮动作文案；NPC 任务只提供前往提示，奖励仍在 NPC 处领取。
func _action_label_for_quest(quest: Dictionary) -> String:
    var state: String = str(quest.get("state", ""))
    match state:
        "AVAILABLE":
            return "前往" if int(quest.get("start_npc_id", 0)) > 0 else "领取"
        "ACCEPTED":
            return "前往"
        "READY_TO_SUBMIT":
            return "前往" if _requires_npc_submit(quest) else "领取"
        _:
            return ""


## 判断任务是否必须去 NPC 处交付并领取奖励。
func _requires_npc_submit(quest: Dictionary) -> bool:
    return int(quest.get("submit_npc_id", 0)) > 0


## 读取服务端下发的客户端任务图标 ID；兼容旧字段便于灰度期间不空图。
func _task_client_icon_id(quest: Dictionary) -> int:
    var client_icon_id: int = int(quest.get("client_icon_id", 0))
    if client_icon_id > 0:
        return client_icon_id
    return int(quest.get("icon_id", 0))


## 点击主按钮时按任务状态发起接取、追踪或面板领奖请求。
func _on_task_card_action_pressed(card: Control) -> void:
    if _pending_action_seq > 0:
        return
    var quest: Dictionary = _quest_for_card(card)
    var quest_id: int = int(quest.get("quest_id", 0))
    if quest_id <= 0:
        return
    var state: String = str(quest.get("state", ""))
    match state:
        "AVAILABLE":
            if int(quest.get("start_npc_id", 0)) > 0:
                _track_and_notice_quest(quest, "请前往任务 NPC 领取。")
                return
            _pending_action_seq = App.accept_quest(quest_id, 0)
        "ACCEPTED":
            _track_and_notice_quest(quest, "已追踪任务，请按目标提示前往。")
            return
        "READY_TO_SUBMIT":
            if _requires_npc_submit(quest):
                _track_and_notice_quest(quest, "请前往任务 NPC 交付并领取奖励。")
                return
            _pending_action_seq = App.submit_quest(quest_id, 0)
        _:
            return
    if _pending_action_seq <= 0:
        return
    _refresh_task_lists()
    call_deferred("_wait_task_action_request", _pending_action_seq)


## 面板不能直接完成的 NPC 任务，点击前往时只切换追踪并给出提示。
func _track_and_notice_quest(quest: Dictionary, notice: String) -> void:
    var quest_id: int = int(quest.get("quest_id", 0))
    if quest_id <= 0:
        return
    _pending_action_seq = App.track_quest(quest_id)
    App.notice_received.emit(notice)
    if _pending_action_seq > 0:
        _refresh_task_lists()
        call_deferred("_wait_task_action_request", _pending_action_seq)


## 点击详情按钮时输出当前任务进度与奖励摘要，后续可替换为专用详情弹窗场景。
func _on_task_card_detail_pressed(card: Control) -> void:
    var quest: Dictionary = _quest_for_card(card)
    if quest.is_empty():
        return
    App.notice_received.emit(_build_task_notice(quest))


## 等待任务操作回包后解锁按钮，任务快照由 QuestController 通过响应或 push 刷新。
func _wait_task_action_request(expected_seq: int) -> void:
    while expected_seq > 0:
        var result: Array = await App.request_finished
        if result.size() < 5:
            continue
        var seq: int = int(result[1])
        if seq != expected_seq:
            continue
        _pending_action_seq = 0
        _refresh_task_lists()
        return


## 根据卡片实例 ID 取回当前绑定的任务快照。
func _quest_for_card(card: Control) -> Dictionary:
    if card == null:
        return {}
    var quest_variant: Variant = _quest_by_card_id.get(card.get_instance_id(), {})
    if quest_variant is Dictionary:
        return (quest_variant as Dictionary).duplicate(true)
    return {}


## 构建轻量详情提示文本。
func _build_task_notice(quest: Dictionary) -> String:
    var progress: Dictionary = _resolve_task_progress(quest)
    var reward_text: String = _format_rewards(quest)
    var parts: Array[String] = [
        UiFormat.normalize_text(str(quest.get("title", "任务"))),
        _build_task_description(quest),
        "进度：%s/%s" % [UiFormat.value_to_text(int(progress.get("current", 0))), UiFormat.value_to_text(int(progress.get("target", 1)))],
    ]
    if not reward_text.is_empty():
        parts.append("奖励：%s" % reward_text)
    return _join_strings(parts, "\n")


## 格式化服务端下发的奖励预览。
func _format_rewards(quest: Dictionary) -> String:
    var rewards_variant: Variant = quest.get("rewards", [])
    if rewards_variant is not Array:
        return ""
    var texts: Array[String] = []
    var rewards: Array = rewards_variant as Array
    for reward_variant: Variant in rewards:
        if reward_variant is not Dictionary:
            continue
        var reward: Dictionary = reward_variant as Dictionary
        var reward_type: String = str(reward.get("type", ""))
        match reward_type:
            "gold":
                texts.append("金币%s" % UiFormat.value_to_text(int(reward.get("value", 0))))
            "exp":
                texts.append("经验%s" % UiFormat.value_to_text(int(reward.get("value", 0))))
            "item":
                texts.append("道具%s x%s" % [UiFormat.value_to_text(int(reward.get("item_id", 0))), UiFormat.value_to_text(int(reward.get("count", 0)))])
            "pet":
                texts.append("宠物%s" % UiFormat.value_to_text(int(reward.get("pet_id", 0))))
            "feature_unlock":
                texts.append("功能解锁")
    return _join_strings(texts, "，")


## 连接字符串数组，避免不同 Godot 小版本对 String.join 入参类型的兼容差异。
func _join_strings(values: Array[String], separator: String) -> String:
    var result: String = ""
    for index: int in range(values.size()):
        if index > 0:
            result += separator
        result += values[index]
    return result


## 通用任务卡片图标节点；当前需求暂时搁置图标展示。
func _task_icon(card: Control) -> TextureRect:
    return card.get_node_or_null("MarginContainer/PanelContainer/HBoxContainer/TextureRect") as TextureRect


## 通用任务卡片标题标签。
func _task_title_label(card: Control) -> Label:
    return card.get_node_or_null("MarginContainer/PanelContainer/HBoxContainer/VBoxContainer/Label") as Label


## 通用任务卡片描述标签。
func _task_description_label(card: Control) -> RichTextLabel:
    return card.get_node_or_null("MarginContainer/PanelContainer/HBoxContainer/VBoxContainer/Label2") as RichTextLabel


## 通用任务卡片进度条。
func _task_progress_bar(card: Control) -> ProgressBar:
    return card.get_node_or_null("MarginContainer/PanelContainer/HBoxContainer/VBoxContainer/MarginContainer/HBoxContainer/ProgressBar") as ProgressBar


## 通用任务卡片进度文本。
func _task_progress_label(card: Control) -> Label:
    return card.get_node_or_null("MarginContainer/PanelContainer/HBoxContainer/VBoxContainer/MarginContainer/HBoxContainer/Label") as Label


## 通用任务卡片主操作按钮。
func _task_action_button(card: Control) -> BaseButton:
    return card.get_node_or_null("MarginContainer/PanelContainer/HBoxContainer/MarginContainer/VBoxContainer/Button") as BaseButton


## 通用任务卡片详情按钮。
func _task_detail_button(card: Control) -> BaseButton:
    return card.get_node_or_null("MarginContainer/PanelContainer/HBoxContainer/MarginContainer/VBoxContainer/Button2") as BaseButton
