extends PanelContainer
class_name DevServerSwitcher

## 开发切服面板场景路径，便于登录页和其他调试入口复用。
const SCENE_PATH: String = "res://scenes/ui/common/dev_server_switcher.tscn"

## 当玩家点击“应用配置”后，向外广播当前生效的环境与地址。
signal config_applied(profile_name: String, http_base: String, ws_url: String)
## 当玩家点击“清空覆盖”后，向外广播当前已恢复默认地址。
signal config_cleared(profile_name: String, http_base: String, ws_url: String)

## 统一读取客户端网络配置解析逻辑，避免面板重复维护地址拼接规则。
const NetworkConfigScript = preload("res://autoload/network_config.gd")

## 切换本地 / 远程 / 同源环境的下拉框。
@onready var profile_option_button: OptionButton = %ProfileOptionButton
## 手动覆盖 HTTP 基础地址的输入框。
@onready var http_base_input: LineEdit = %HttpBaseInput
## 手动覆盖 WebSocket 地址的输入框。
@onready var ws_base_input: LineEdit = %WsBaseInput
## 展示当前实际生效地址的摘要文本。
@onready var summary_label: Label = %SummaryLabel
## 应用当前面板配置的按钮。
@onready var apply_button: Button = %ApplyButton
## 清空当前临时覆盖并回到默认解析结果的按钮。
@onready var reset_button: Button = %ResetButton

## 初始化环境选项和当前默认值。
func _ready() -> void:
    _setup_profile_options()
    refresh_from_active_config()
    apply_button.pressed.connect(_on_apply_button_pressed)
    reset_button.pressed.connect(_on_reset_button_pressed)

## 用当前实际生效的网络配置刷新面板显示。
func refresh_from_active_config() -> void:
    _select_profile(NetworkConfigScript.get_active_profile())
    http_base_input.text = NetworkConfigScript.get_manual_http_base_override()
    ws_base_input.text = NetworkConfigScript.get_manual_ws_url_override()
    _refresh_summary()

## 初始化环境下拉框，保持选项值与 NetworkConfig 常量一致。
func _setup_profile_options() -> void:
    profile_option_button.clear()
    profile_option_button.add_item("本地后端", 0)
    profile_option_button.set_item_metadata(0, NetworkConfigScript.PROFILE_LOCAL)
    profile_option_button.add_item("远程服务", 1)
    profile_option_button.set_item_metadata(1, NetworkConfigScript.PROFILE_REMOTE)
    profile_option_button.add_item("浏览器同源", 2)
    profile_option_button.set_item_metadata(2, NetworkConfigScript.PROFILE_BROWSER_ORIGIN)

## 根据当前实际配置选中对应环境项。
func _select_profile(profile_name: String) -> void:
    var item_count: int = profile_option_button.item_count
    var index: int = 0
    while index < item_count:
        var metadata: Variant = profile_option_button.get_item_metadata(index)
        if str(metadata) == profile_name:
            profile_option_button.select(index)
            return
        index += 1
    profile_option_button.select(0)

## 点击应用按钮后，把面板内容写回统一网络配置。
func _on_apply_button_pressed() -> void:
    var profile_name: String = _selected_profile_name()
    var http_base: String = http_base_input.text.strip_edges()
    var ws_url: String = ws_base_input.text.strip_edges()
    NetworkConfigScript.apply_debug_config(profile_name, http_base, ws_url)
    refresh_from_active_config()
    config_applied.emit(
        NetworkConfigScript.get_active_profile(),
        NetworkConfigScript.get_active_http_base_url(),
        NetworkConfigScript.get_active_ws_url()
    )

## 点击清空按钮后，移除手动覆盖并恢复默认环境解析。
func _on_reset_button_pressed() -> void:
    NetworkConfigScript.clear_debug_config()
    refresh_from_active_config()
    config_cleared.emit(
        NetworkConfigScript.get_active_profile(),
        NetworkConfigScript.get_active_http_base_url(),
        NetworkConfigScript.get_active_ws_url()
    )

## 返回当前下拉框选中的环境标识。
func _selected_profile_name() -> String:
    var selected_index: int = profile_option_button.selected
    if selected_index < 0:
        return NetworkConfigScript.PROFILE_LOCAL
    var metadata: Variant = profile_option_button.get_item_metadata(selected_index)
    return str(metadata).strip_edges()

## 重新生成面板里的摘要文案，方便快速确认当前是否已切到目标地址。
func _refresh_summary() -> void:
    var summary_text: String = "当前环境: %s\nHTTP: %s\nWS: %s" % [
        _profile_display_name(NetworkConfigScript.get_active_profile()),
        NetworkConfigScript.get_active_http_base_url(),
        NetworkConfigScript.get_active_ws_url(),
    ]
    summary_label.text = UiFormat.normalize_text(summary_text)

## 把内部环境标识转换为更直观的中文文案。
func _profile_display_name(profile_name: String) -> String:
    match profile_name:
        NetworkConfigScript.PROFILE_LOCAL:
            return "本地后端"
        NetworkConfigScript.PROFILE_REMOTE:
            return "远程服务"
        NetworkConfigScript.PROFILE_BROWSER_ORIGIN:
            return "浏览器同源"
        _:
            return profile_name
