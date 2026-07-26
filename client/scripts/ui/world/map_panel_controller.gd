extends Node
class_name MapPanelController

## 主运行态右下角地图入口按钮。
@onready var map_button: Button = get_node_or_null("../UiLayer/HudRoot/MapButton") as Button
## 主场景中预先实例化的地图节点面板。
@onready var map_panel: MapTeleportPanel = get_node_or_null("../MapTeleportPanel") as MapTeleportPanel
## 世界场景实例所在的固定挂载节点。
@onready var world_mount: Control = get_node_or_null("../GameplayArea/WorldMount") as Control
## 运行态 HUD 提供统一的移动端业务提示入口。
@onready var hud_root: Control = get_node_or_null("../UiLayer/HudRoot") as Control

## 绑定地图入口和战斗状态；所有 UI 都由主场景节点预先提供。
func _ready() -> void:
    if map_button != null and not map_button.pressed.is_connected(_on_map_button_pressed):
        map_button.pressed.connect(_on_map_button_pressed)
    if map_panel != null and not map_panel.menu_closed.is_connected(_on_map_panel_closed):
        map_panel.menu_closed.connect(_on_map_panel_closed)
    if map_panel != null and not map_panel.teleport_requested.is_connected(_on_teleport_requested):
        map_panel.teleport_requested.connect(_on_teleport_requested)
    if map_panel != null and not map_panel.notice_requested.is_connected(_show_notice):
        map_panel.notice_requested.connect(_show_notice)
    if not GameState.battle_changed.is_connected(_on_battle_changed):
        GameState.battle_changed.connect(_on_battle_changed)


## 离开主运行态时断开全局信号，避免已释放控制器继续接收回调。
func _exit_tree() -> void:
    if GameState.battle_changed.is_connected(_on_battle_changed):
        GameState.battle_changed.disconnect(_on_battle_changed)


## 点击地图入口时切换面板；战斗期间不允许打开世界地图。
func _on_map_button_pressed() -> void:
    if map_panel == null or GameState.is_in_battle:
        return
    if map_panel.visible:
        map_panel.close_menu()
        return
    _close_other_runtime_panels()
    map_panel.open_menu()
    _set_world_input_locked(true)


## 地图面板关闭后恢复世界输入。
func _on_map_panel_closed() -> void:
    if not GameState.is_in_battle:
        _set_world_input_locked(false)


## 关闭地图面板并把目标场景交给当前世界控制器；出生格不在客户端计算或提交。
## target_scene_id 是玩家再次点击标点后选择的目标场景 ID；point_name 用于本地失败提示。
func _on_teleport_requested(target_scene_id: int, point_name: String) -> void:
    var world_controller: Node = _find_world_controller()
    if world_controller == null or not world_controller.has_method("request_map_teleport"):
        _show_notice("暂时无法前往%s，请稍后重试。" % point_name)
        return
    # 关闭信号与随后发起请求发生在同一调用栈，不会产生可消费世界输入的中间帧。
    map_panel.close_menu()
    world_controller.call("request_map_teleport", target_scene_id)


## 通过既有运行态 HUD 展示地图业务提示，避免新增并行弹窗。
## message 是地图面板或本地可用性校验产生的提示正文。
func _show_notice(message: String) -> void:
    if hud_root != null and hud_root.has_method("show_notice"):
        hud_root.call("show_notice", message)


## 进入战斗时主动关闭地图面板，避免世界 UI 覆盖战斗表现。
func _on_battle_changed() -> void:
    if not GameState.is_in_battle or map_panel == null or not map_panel.visible:
        return
    map_panel.close_menu()


## 打开地图前关闭其它根面板，保持移动端同一时间只显示一个主面板。
func _close_other_runtime_panels() -> void:
    for panel_node: Node in get_tree().get_nodes_in_group("runtime_root_panel"):
        var panel_layer: CanvasLayer = panel_node as CanvasLayer
        if panel_layer == null or panel_layer == map_panel or not panel_layer.visible:
            continue
        if panel_layer.has_method("close_menu"):
            panel_layer.call("close_menu")


## 设置当前世界控制器的输入锁；地图面板只控制交互，不修改任何权威状态。
## locked 表示是否阻止地图移动和 NPC 交互。
func _set_world_input_locked(locked: bool) -> void:
    var world_controller: Node = _find_world_controller()
    if world_controller != null and world_controller.has_method("set_runtime_input_locked"):
        world_controller.call("set_runtime_input_locked", locked)


## 从固定 WorldMount 中查找当前动态挂载的世界控制器。
func _find_world_controller() -> Node:
    if world_mount == null:
        return null
    for child: Node in world_mount.get_children():
        if child.has_method("set_runtime_input_locked"):
            return child
    return null
