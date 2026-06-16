extends PanelContainer

## 命令阶段倒计时归零后请求服务端切入自动战斗。
signal request_auto_timeout()

const AUTO_HINT_TEXT: String = "已开启自动战斗，再次点击按钮关闭"

@onready var _status_label: Label = %StatusLabel
@onready var _network: BattleNetworkProvider = %BattleNetworkProvider

## 当前命令阶段是否已发起过超时自动托管请求，避免重复发包。
var _timeout_auto_requested: bool = false
## 用于识别新一轮命令阶段，重置超时请求标记。
var _last_command_signature: String = ""


func _ready() -> void:
	hide()


func _process(_delta: float) -> void:
	_refresh_status()


## 根据服务端权威倒计时与自动战斗状态刷新顶部提示。
func _refresh_status() -> void:
	if _network == null:
		hide()
		return
	# 自动战斗开启后提示常驻，直到玩家再次点击自动按钮关闭。
	if _network.is_auto_battle_enabled():
		visible = true
		_status_label.text = AUTO_HINT_TEXT
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
	var command_signature: String = "%d_%d_%d" % [
		_network.get_battle_id(),
		_network.get_round(),
		_network.get_command_deadline_ms(),
	]
	if command_signature != _last_command_signature:
		_reset_timeout_flag(command_signature)
	var remain_sec: int = _network.get_command_remaining_seconds()
	visible = true
	_status_label.text = "请选择操作：%d秒" % remain_sec
	if remain_sec <= 0 and not _timeout_auto_requested:
		_timeout_auto_requested = true
		request_auto_timeout.emit()


func _reset_timeout_flag(signature: String) -> void:
	_last_command_signature = signature
	_timeout_auto_requested = false
