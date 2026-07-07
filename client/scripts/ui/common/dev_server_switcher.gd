extends PanelContainer
class_name DevServerSwitcher

## 开发切服面板场景路径，便于登录页和其他调试入口复用。
const SCENE_PATH: String = "res://scenes/ui/common/dev_server_switcher.tscn"

## 当玩家切换环境后，向外广播当前生效的环境与地址。
signal config_changed(profile_name: String, http_base: String, ws_url: String)

## 统一读取客户端网络配置解析逻辑，避免面板重复维护地址拼接规则。
const NetworkConfigScript = preload("res://autoload/network_config.gd")

## 切换本地 / 远程 / 同源环境的下拉框。
@onready var profile_option_button: OptionButton = %ProfileOptionButton

## 初始化环境选项和当前默认值。
func _ready() -> void:
    _setup_profile_options()
    # 登录页调试入口简化后，首次打开统一默认回到本地环境，避免遗留旧覆盖导致误连远程。
    NetworkConfigScript.apply_debug_config(NetworkConfigScript.PROFILE_LOCAL)
    refresh_from_active_config()
    if not profile_option_button.item_selected.is_connected(_on_profile_option_selected):
        profile_option_button.item_selected.connect(_on_profile_option_selected)

## 用当前实际生效的网络配置刷新面板显示。
func refresh_from_active_config() -> void:
    _select_profile(NetworkConfigScript.get_active_profile())

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

## 切换下拉框时立即应用对应环境，避免登录页再维护手动地址与额外操作按钮。
func _on_profile_option_selected(_index: int) -> void:
    var profile_name: String = _selected_profile_name()
    NetworkConfigScript.apply_debug_config(profile_name)
    refresh_from_active_config()
    config_changed.emit(
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
