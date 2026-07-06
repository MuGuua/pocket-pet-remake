extends PanelContainer

## 命令阶段倒计时归零后通知 BattleDirector 在客户端生成自动战斗意图。
signal request_auto_timeout()

## 260 宽视口下压缩顶部提示文案，避免自动战斗提示超出状态条。
const AUTO_HINT_TEXT: String = "自动战斗中，点自动关闭"
const SELECTOR_HIGHLIGHT_COLOR: String = "#FFE566"
const PLAYER_UNIT_CLASS: int = 1
const PET_UNIT_CLASS: int = 2

@onready var _status_label: RichTextLabel = %StatusLabel
@onready var _network: BattleNetworkProvider = %BattleNetworkProvider

## 战斗导演，用于判断是否在演出/提交锁定中。
var _director: BattleDirector = null
## 顶部倒计时/操作提示字体大小；导出后可在 battle_scene.tscn 的 CommandStatusBar 上继续微调。
@export var status_font_size: int = 36
## 当前命令阶段是否已发起过超时自动托管请求，避免重复发包。
var _timeout_auto_requested: bool = false
## 用于识别新一轮命令阶段，重置超时请求标记。
var _last_command_signature: String = ""


func _ready() -> void:
    hide()
    _status_label.bbcode_enabled = true
    _status_label.scroll_active = false
    _status_label.add_theme_font_size_override("normal_font_size", status_font_size)


## 由 BattleDirector 注入，用于同步「可操作」与「演出中」的显示时机。
func bind_director(director: BattleDirector) -> void:
    _director = director


func _process(_delta: float) -> void:
    _refresh_status()


## 提交指令、等待结算或播放回合动画时不显示选操作提示。
func _should_hide_for_presentation() -> bool:
    if _director == null:
        return false
    if _director.is_interaction_locked():
        return true
    if _director.has_pending_presentations():
        return true
    return false


## 根据客户端本地倒计时与自动战斗状态刷新顶部提示。
func _refresh_status() -> void:
    if _network == null:
        hide()
        return
    if _director == null:
        hide()
        return
    # 自动战斗开启后提示常驻（含动画播放、等待结算阶段），直到玩家再次关闭。
    if _director.is_client_auto_battle_enabled():
        visible = true
        _status_label.text = _format_plain_text(AUTO_HINT_TEXT)
        return
    if _should_hide_for_presentation():
        hide()
        return
    if _network.get_phase() != "command":
        hide()
        _reset_timeout_flag("")
        return
    var pending_actor_ids: Array[int] = _network.get_pending_actor_ids()
    if pending_actor_ids.is_empty():
        hide()
        _reset_timeout_flag("")
        return
    var command_signature: String = "%d_%d_%s" % [
        _network.get_battle_id(),
        _network.get_round(),
        str(pending_actor_ids),
    ]
    if command_signature != _last_command_signature:
        _reset_timeout_flag(command_signature)
    var remain_sec: int = _director.get_client_command_remaining_seconds()
    visible = true
    _status_label.text = _format_selection_prompt(remain_sec)
    if remain_sec <= 0 and not _timeout_auto_requested:
        _timeout_auto_requested = true
        request_auto_timeout.emit()


func _reset_timeout_flag(signature: String) -> void:
    _last_command_signature = signature
    _timeout_auto_requested = false


## 根据当前待选单位生成「请选择人物/宠物操作」提示文案。
func _format_selection_prompt(remain_sec: int) -> String:
    var role_label: String = _resolve_selector_role_label()
    return "请选择[color=%s]%s[/color]操作：%d秒" % [
        SELECTOR_HIGHLIGHT_COLOR,
        role_label,
        remain_sec,
    ]


## 解析当前应选单位是人物还是宠物。
func _resolve_selector_role_label() -> String:
    var actor_id: int = int(_network.get_battle_state().get("active_actor_id", 0))
    if actor_id <= 0:
        var pending_actor_ids: Array[int] = _network.get_pending_actor_ids()
        if pending_actor_ids.is_empty():
            return "操作"
        actor_id = pending_actor_ids[0]
    var snapshot: Dictionary = _network.find_actor_snapshot(actor_id)
    var unit_class: int = int(snapshot.get("unit_class", 0))
    if unit_class == PLAYER_UNIT_CLASS:
        return "人物"
    if unit_class == PET_UNIT_CLASS:
        return "宠物"
    return "操作"


## 自动战斗等无需高亮时用纯文本。
func _format_plain_text(text: String) -> String:
    return text
