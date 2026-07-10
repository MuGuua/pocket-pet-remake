extends RuntimeRootPanel
class_name PetStatusPanel

## 通用技能说明弹窗；宠物技能按钮悬停/按下时复用它渲染服务端 BBCode 描述。
const ConfirmPromptPopupScene: PackedScene = preload(ConfirmPromptPopup.SCENE_PATH)
## 等待 PET_LIST_RESP 的最大帧数，避免网络异常时面板打开流程永久挂起。
const OPEN_REQUEST_TIMEOUT_FRAMES: int = 300
## 宠物基础属性页在场景中的相对路径，集中管理便于后续 UI 调整。
const BASIC_PANEL_PATH: NodePath = NodePath("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据面板/基础属性")
## 宠物状态抗性页在场景中的相对路径，必须和场景节点名保持一致。
const STATUS_PANEL_PATH: NodePath = NodePath("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据面板/状态抗性")
## 宠物资质技能页在场景中的相对路径，集中管理便于后续 UI 调整。
const SKILL_PANEL_PATH: NodePath = NodePath("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据面板/技能资质")
## 宠物法宝装备页在场景中的相对路径。
const ARTIFACT_PANEL_PATH: NodePath = NodePath("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据面板/法宝装备")
## 宠物状态页默认展示的分页键。
const DEFAULT_TAB_KEY: String = "basic"

## DisplayPanel 中宠物待机形象的本地坐标；可在 Inspector 调整来微调展示位置。
@export var preview_sprite_position: Vector2 = Vector2(207.77693, 106.18534)
## 宠物技能描述弹窗主体位置；对应 confirm_prompt_popup.tscn 中 VBoxContainer 的 position，可在 Inspector 自行微调。
@export var skill_description_popup_position: Vector2 = Vector2(-280.0, -74.5)
@export_group("技能品质边框")
## 普通技能按钮主题。
@export var normal_skill_button_theme: Theme = null
## 神技按钮主题。
@export var divine_skill_button_theme: Theme = null
## 魂技按钮主题。
@export var soul_skill_button_theme: Theme = null
## 圣技按钮主题。
@export var sacred_skill_button_theme: Theme = null
## 绝世技能按钮主题。
@export var peerless_skill_button_theme: Theme = null

## 标题栏关闭按钮。
@onready var _close_button: BaseButton = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/Title/HBoxContainer/Button") as BaseButton
## 左侧宠物列表容器，复用场景里已有的宠物按钮组件。
@onready var _pet_grid: GridContainer = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/宠物列表/MarginContainer/GridContainer") as GridContainer
## 顶部展示区的宠物名称。
@onready var _pet_name_label: Label = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DisplayPanel/PanelContainer/VBoxContainer/PanelContainer/HBoxContainer/PetName") as Label
## 顶部展示区的宠物等级。
@onready var _pet_level_label: Label = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DisplayPanel/PanelContainer/VBoxContainer/PanelContainer/HBoxContainer/PetLevel") as Label
## 顶部展示区的宠物动画节点；没有服务端形象时隐藏，避免展示场景模板假数据。
@onready var _preview_sprite: AnimatedSprite2D = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DisplayPanel/PanelContainer/VBoxContainer/PanelContainer2/Control/HBoxContainer/Control/HBoxContainer/Pet") as AnimatedSprite2D
## 基础属性数据页。
@onready var _basic_panel: Control = get_node_or_null(BASIC_PANEL_PATH) as Control
## 状态抗性数据页，当前展示服务端下发的状态抗性类数值。
@onready var _status_panel: Control = get_node_or_null(STATUS_PANEL_PATH) as Control
## 技能资质数据页。
@onready var _skill_panel: Control = get_node_or_null(SKILL_PANEL_PATH) as Control
## 法宝装备数据页。
@onready var _artifact_panel: Control = get_node_or_null(ARTIFACT_PANEL_PATH) as Control
## 基础属性分页按钮。
@onready var _basic_tab_button: BaseButton = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据按钮/基础属性") as BaseButton
## 状态抗性分页按钮。
@onready var _status_tab_button: BaseButton = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据按钮/状态抗性") as BaseButton
## 技能资质分页按钮。
@onready var _skill_tab_button: BaseButton = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据按钮/技能资质") as BaseButton
## 法宝装备分页按钮。
@onready var _artifact_tab_button: BaseButton = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据按钮/法宝装备") as BaseButton
## 其他功能按钮容器；这些按钮不是分页，不应保持按下状态。
@onready var _operation_button_box: HBoxContainer = get_node_or_null("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/其他按钮") as HBoxContainer
## 切换宠物时使用的通用 Loading；短请求会延迟 1 秒显示，避免闪屏。
@onready var _request_loading: GenericLoadingScene = get_node_or_null("RequestLoadingOverlay") as GenericLoadingScene
## 技能表现资源注册表；按服务端下发的 skill_visual_id 解析本地技能图标。
@onready var _skill_visual_registry: BattleContentRegistry = get_node_or_null("BattleContentRegistry") as BattleContentRegistry

## 当前宠物列表按钮集合。
var _pet_buttons: Array[BaseButton] = []
## 宠物列表按钮的场景默认贴图；服务端暂缺外观时用于兜底，避免槽位空白。
var _default_pet_choice_textures: Dictionary = {}
## 技能资质页里的固定技能按钮集合。
var _skill_buttons: Array[BaseButton] = []
## 技能按钮场景自带的默认图标；服务端未下发 icon 时用作“已有技能”的兜底图标。
var _default_skill_icon: Texture2D = null
## 技能按钮实例 ID 到服务端技能槽数据的映射，悬停/按下时读取技能名和富文本描述。
var _skill_entry_by_button_id: Dictionary = {}
## 技能详情通用弹窗实例，按需懒创建，避免每个按钮重复实例化弹窗。
var _skill_description_popup: ConfirmPromptPopup = null
## 当前由鼠标悬停打开技能详情的按钮；鼠标离开该按钮区域后自动收起弹窗。
var _skill_hover_source_button: BaseButton = null
## 非分页功能按钮集合，点击后统一清掉按下态。
var _operation_buttons: Array[BaseButton] = []
## 当前打开的数据分页。
var _active_tab_key: String = DEFAULT_TAB_KEY
## 当前选中的宠物 uid。
var _selected_pet_uid: int = 0
## 面板打开请求代次，用于关闭或重复打开时取消旧 await。
var _open_load_generation: int = 0
## 宠物列表点击切换代次，用于快速连点时丢弃较早的详情回包刷新。
var _switch_detail_generation: int = 0
## 当前等待的 PET_LIST_REQ 序列号。
var _pending_pet_list_seq: int = 0
## PET_LIST_REQ 是否已有回调结果。
var _pending_pet_list_ready: bool = false
## PET_LIST_REQ 是否成功。
var _pending_pet_list_ok: bool = false
## 当前等待的 PET_SKILL_DETAIL_REQ 序列号。
var _pending_skill_detail_seq: int = 0
## PET_SKILL_DETAIL_REQ 是否已有回调结果。
var _pending_skill_detail_ready: bool = false


## 初始化面板事件、按钮缓存与权威状态订阅。
func _ready() -> void:
    super._ready()
    set_process(false)
    _collect_pet_buttons()
    _collect_skill_buttons()
    _connect_data_tabs()
    _collect_operation_buttons()
    _apply_preview_sprite_position()
    if _close_button != null and not _close_button.button_down.is_connected(_on_close_button_pressed):
        _close_button.button_down.connect(_on_close_button_pressed)
    if not GameState.pets_changed.is_connected(refresh_panel_data):
        GameState.pets_changed.connect(refresh_panel_data)
    if not App.request_finished.is_connected(_on_request_finished):
        App.request_finished.connect(_on_request_finished)
    _show_data_tab(DEFAULT_TAB_KEY)
    refresh_panel_data()


## 断开全局信号，避免面板销毁后继续刷新。
func _exit_tree() -> void:
    if GameState.pets_changed.is_connected(refresh_panel_data):
        GameState.pets_changed.disconnect(refresh_panel_data)
    if App.request_finished.is_connected(_on_request_finished):
        App.request_finished.disconnect(_on_request_finished)
    if _skill_description_popup != null:
        _skill_description_popup.queue_free()
        _skill_description_popup = null


## 打开前通过服务端权威 PET_LIST_REQ 刷新宠物数据，确保面板首次展示就是最新数据。
func prepare_open_data() -> bool:
    _open_load_generation += 1
    var load_id: int = _open_load_generation
    if not GameState.is_ws_authenticated:
        return false
    _pending_pet_list_seq = App.request_pet_list()
    if _pending_pet_list_seq <= 0:
        return false
    _pending_pet_list_ready = false
    _pending_pet_list_ok = false
    var waited_frames: int = 0
    while not _pending_pet_list_ready and waited_frames < OPEN_REQUEST_TIMEOUT_FRAMES:
        await get_tree().process_frame
        waited_frames += 1
    if load_id != _open_load_generation:
        return false
    if _pending_pet_list_ok:
        _ensure_selected_pet()
        await _request_selected_pet_skill_detail_for_open(load_id)
    _pending_pet_list_seq = 0
    return _pending_pet_list_ok


## 数据准备完成后打开宠物面板，并刷新展示。
func open_menu() -> void:
    super.open_menu()
    refresh_panel_data()


## 关闭宠物面板并取消仍在等待的打开请求。
func close_menu() -> void:
    _open_load_generation += 1
    _switch_detail_generation += 1
    _pending_pet_list_seq = 0
    _pending_skill_detail_seq = 0
    _close_skill_description_popup()
    _hide_switch_loading_overlay()
    super.close_menu()


## 按当前 GameState 中的服务端宠物快照刷新列表、摘要和基础属性。
func refresh_panel_data() -> void:
    _ensure_selected_pet()
    _refresh_pet_buttons()
    var selected_pet: Dictionary = _resolve_selected_pet()
    _refresh_summary(selected_pet)
    _refresh_basic_attributes(selected_pet)
    _refresh_status_resistance(selected_pet)
    _refresh_qualification_and_skills(selected_pet)


## 收集左侧宠物按钮，并绑定点击切换选中宠物。
func _collect_pet_buttons() -> void:
    _pet_buttons.clear()
    _default_pet_choice_textures.clear()
    if _pet_grid == null:
        return
    var button_index: int = 0
    for child: Node in _pet_grid.get_children():
        if child is BaseButton:
            var button: BaseButton = child as BaseButton
            button.toggle_mode = true
            _pet_buttons.append(button)
            var image_node: TextureRect = button.get_node_or_null("PetImage") as TextureRect
            if image_node != null:
                _default_pet_choice_textures[button.get_instance_id()] = image_node.texture
            if not button.button_down.is_connected(_on_pet_button_pressed.bind(button_index)):
                button.button_down.connect(_on_pet_button_pressed.bind(button_index))
            button_index += 1


## 收集技能资质页中的技能槽按钮，并缓存场景默认图标。
func _collect_skill_buttons() -> void:
    _skill_buttons.clear()
    _skill_entry_by_button_id.clear()
    if _skill_panel == null:
        return
    var grid: GridContainer = _skill_panel.get_node_or_null("资质与技能/VBoxContainer/技能面板/PanelContainer/MarginContainer/VBoxContainer/GridContainer") as GridContainer
    if grid == null:
        return
    for child: Node in grid.get_children():
        if child is BaseButton:
            var button: BaseButton = child as BaseButton
            button.toggle_mode = false
            _skill_buttons.append(button)
            if _default_skill_icon == null:
                var icon_node: TextureRect = _resolve_skill_button_icon(button)
                if icon_node != null:
                    _default_skill_icon = icon_node.texture
            if not button.mouse_entered.is_connected(_on_skill_button_hovered.bind(button)):
                button.mouse_entered.connect(_on_skill_button_hovered.bind(button))
            if not button.mouse_exited.is_connected(_on_skill_button_unhovered.bind(button)):
                button.mouse_exited.connect(_on_skill_button_unhovered.bind(button))
            if not button.button_down.is_connected(_on_skill_button_down.bind(button)):
                button.button_down.connect(_on_skill_button_down.bind(button))
    _sort_skill_buttons_by_visual_slot()


## 按场景按钮命名排序技能槽：Button/Button2...Button6 为上排 1~6，Button7...Button12 为下排 7~12。
func _sort_skill_buttons_by_visual_slot() -> void:
    _skill_buttons.sort_custom(_compare_skill_buttons_by_visual_slot)


## sort_custom 比较器；返回 true 表示 left 应排在 right 前面。
func _compare_skill_buttons_by_visual_slot(left_button: BaseButton, right_button: BaseButton) -> bool:
    return _resolve_skill_button_visual_slot(left_button) < _resolve_skill_button_visual_slot(right_button)


## 从按钮节点名解析技能槽显示序号，确保服务端第 N 个技能稳定渲染到第 N 个按钮。
func _resolve_skill_button_visual_slot(button: BaseButton) -> int:
    if button == null:
        return 9999
    var button_name: String = str(button.name)
    if button_name == "Button":
        return 1
    if button_name.begins_with("Button"):
        var suffix: String = button_name.substr("Button".length()).strip_edges()
        if suffix.is_valid_int():
            return int(suffix)
    return button.get_index() + 1


## 每帧检查悬停触发的技能说明是否仍停留在原按钮上；弹窗遮罩会接管鼠标事件，所以不能只依赖 mouse_exited。
func _process(_delta: float) -> void:
    if _skill_hover_source_button == null or not is_instance_valid(_skill_hover_source_button):
        _skill_hover_source_button = null
        set_process(false)
        return
    if _skill_description_popup == null or not _skill_description_popup.visible:
        _skill_hover_source_button = null
        set_process(false)
        return
    var mouse_position: Vector2 = get_viewport().get_mouse_position()
    if not _skill_hover_source_button.get_global_rect().has_point(mouse_position):
        _close_skill_description_popup()


## 绑定数据页签按钮，四个顶部按钮保持单选 Tab 行为。
func _connect_data_tabs() -> void:
    _connect_data_tab_button(_basic_tab_button, "basic")
    _connect_data_tab_button(_status_tab_button, "status")
    _connect_data_tab_button(_skill_tab_button, "skill")
    _connect_data_tab_button(_artifact_tab_button, "artifact")


## 绑定单个分页按钮，并确保它使用 toggle_mode 展示当前页签状态。
func _connect_data_tab_button(button: BaseButton, tab_key: String) -> void:
    if button == null:
        return
    button.toggle_mode = true
    var callback: Callable = _on_data_tab_pressed.bind(tab_key)
    if not button.pressed.is_connected(callback):
        button.pressed.connect(callback)


## 收集下方功能按钮，这些按钮只触发功能入口，不参与分页选中态。
func _collect_operation_buttons() -> void:
    _operation_buttons.clear()
    if _operation_button_box == null:
        return
    for child: Node in _operation_button_box.get_children():
        if child is BaseButton:
            var button: BaseButton = child as BaseButton
            button.toggle_mode = false
            _operation_buttons.append(button)
            if not button.button_down.is_connected(_on_operation_button_down.bind(button)):
                button.button_down.connect(_on_operation_button_down.bind(button))


## 应用导出的宠物待机形象坐标，方便策划/美术直接在 Inspector 调整。
func _apply_preview_sprite_position() -> void:
    if _preview_sprite != null:
        _preview_sprite.position = preview_sprite_position


## 响应左侧宠物按钮点击，切换当前查看的宠物并按单只宠物请求完整属性。
func _on_pet_button_pressed(button_index: int) -> void:
    if button_index < 0 or button_index >= GameState.pets.size():
        return
    var pet_variant: Variant = GameState.pets[button_index]
    if pet_variant is not Dictionary:
        return
    var pet: Dictionary = pet_variant as Dictionary
    _selected_pet_uid = int(pet.get("pet_uid", 0))
    _switch_detail_generation += 1
    var switch_id: int = _switch_detail_generation
    _refresh_pet_buttons()
    _refresh_summary(pet)
    await _request_selected_pet_detail_for_switch(_selected_pet_uid, switch_id)
    refresh_panel_data()


## 响应标题栏关闭按钮。
func _on_close_button_pressed() -> void:
    close_menu()


## 处理 PET_LIST_REQ 回包结果，只接收当前打开流程的同一个 seq。
func _on_request_finished(request_cmd: int, seq: int, succeeded: bool, _response_cmd: int, _payload: Dictionary) -> void:
    if request_cmd == CommandIds.PET_LIST_REQ:
        if _pending_pet_list_seq <= 0 or seq != _pending_pet_list_seq:
            return
        _pending_pet_list_ready = true
        _pending_pet_list_ok = succeeded
        return
    if request_cmd == CommandIds.PET_SKILL_DETAIL_REQ:
        if _pending_skill_detail_seq <= 0 or seq != _pending_skill_detail_seq:
            return
        _pending_skill_detail_ready = true


## 确保当前选中宠物仍存在；列表为空时清空选中态。
func _ensure_selected_pet() -> void:
    if GameState.pets.is_empty():
        _selected_pet_uid = 0
        return
    if _selected_pet_uid > 0 and not _resolve_selected_pet().is_empty():
        return
    var first_pet_variant: Variant = GameState.pets[0]
    if first_pet_variant is Dictionary:
        var first_pet: Dictionary = first_pet_variant as Dictionary
        _selected_pet_uid = int(first_pet.get("pet_uid", 0))


## 从全局宠物列表中取出当前选中的宠物快照。
func _resolve_selected_pet() -> Dictionary:
    if _selected_pet_uid > 0:
        for pet_variant: Variant in GameState.pets:
            if pet_variant is not Dictionary:
                continue
            var pet: Dictionary = pet_variant as Dictionary
            if int(pet.get("pet_uid", 0)) == _selected_pet_uid:
                return pet.duplicate(true)
    if not GameState.pets.is_empty() and GameState.pets[0] is Dictionary:
        return (GameState.pets[0] as Dictionary).duplicate(true)
    return {}


## 切换到指定数据页；技能资质页会额外拉取当前宠物完整技能分槽。
func _on_data_tab_pressed(tab_key: String) -> void:
    _show_data_tab(tab_key)
    if tab_key == "skill":
        _request_selected_pet_skill_detail()


## 功能按钮按下后不保持选中态，避免和上方 Tab 状态混淆。
func _on_operation_button_down(button: BaseButton) -> void:
    if button != null:
        button.set_pressed_no_signal(false)


## 显示指定数据页，并同步页签按钮选中态。
func _show_data_tab(tab_key: String) -> void:
    _active_tab_key = tab_key
    if tab_key != "skill":
        _close_skill_description_popup()
    if _basic_panel != null:
        _basic_panel.visible = tab_key == "basic"
    if _status_panel != null:
        _status_panel.visible = tab_key == "status"
    if _skill_panel != null:
        _skill_panel.visible = tab_key == "skill"
    if _artifact_panel != null:
        _artifact_panel.visible = tab_key == "artifact"
    if _basic_tab_button != null:
        _basic_tab_button.button_pressed = tab_key == "basic"
    if _status_tab_button != null:
        _status_tab_button.button_pressed = tab_key == "status"
    if _skill_tab_button != null:
        _skill_tab_button.button_pressed = tab_key == "skill"
    if _artifact_tab_button != null:
        _artifact_tab_button.button_pressed = tab_key == "artifact"
    _clear_operation_button_pressed_state()


## 清理所有非分页功能按钮的按下态。
func _clear_operation_button_pressed_state() -> void:
    for button: BaseButton in _operation_buttons:
        button.set_pressed_no_signal(false)


## 打开面板时等待当前选中宠物的完整属性，避免列表摘要被当成属性详情展示。
func _request_selected_pet_skill_detail_for_open(load_id: int) -> void:
    if _selected_pet_uid <= 0:
        return
    _pending_skill_detail_seq = App.request_pet_skill_detail(_selected_pet_uid)
    if _pending_skill_detail_seq <= 0:
        return
    _pending_skill_detail_ready = false
    var waited_frames: int = 0
    while not _pending_skill_detail_ready and waited_frames < OPEN_REQUEST_TIMEOUT_FRAMES:
        await get_tree().process_frame
        waited_frames += 1
        if load_id != _open_load_generation:
            return
    _pending_skill_detail_seq = 0


## 切换宠物时等待当前宠物完整属性；快速连点时只允许最新一次切换刷新面板。
func _request_selected_pet_detail_for_switch(pet_uid: int, switch_id: int) -> void:
    if pet_uid <= 0 or not GameState.is_ws_authenticated:
        return
    _show_switch_loading_overlay()
    _pending_skill_detail_ready = false
    _pending_skill_detail_seq = App.request_pet_skill_detail(pet_uid)
    if _pending_skill_detail_seq <= 0:
        _hide_switch_loading_overlay()
        return
    var waited_frames: int = 0
    while not _pending_skill_detail_ready and waited_frames < OPEN_REQUEST_TIMEOUT_FRAMES:
        await get_tree().process_frame
        waited_frames += 1
        if switch_id != _switch_detail_generation or pet_uid != _selected_pet_uid:
            return
    _pending_skill_detail_seq = 0
    if switch_id == _switch_detail_generation:
        _hide_switch_loading_overlay()


## 宠物切换请求进入等待态；通用组件内部会在超过 1 秒后才真正显示。
func _show_switch_loading_overlay() -> void:
    if _request_loading != null:
        _request_loading.show_waiting()


## 结束宠物切换等待态；如果 1 秒内返回，会同时取消尚未显示的 loading。
func _hide_switch_loading_overlay() -> void:
    if _request_loading != null:
        _request_loading.hide_loading()


## 请求当前选中宠物的完整属性；用于切到技能页时兜底刷新当前宠物详情。
func _request_selected_pet_skill_detail() -> void:
    if _selected_pet_uid <= 0 or not GameState.is_ws_authenticated:
        return
    _pending_skill_detail_ready = false
    _pending_skill_detail_seq = App.request_pet_skill_detail(_selected_pet_uid)


## 刷新左侧宠物列表按钮的图标、出战标记和选中状态。
func _refresh_pet_buttons() -> void:
    for button_index: int in range(_pet_buttons.size()):
        var button: BaseButton = _pet_buttons[button_index]
        var has_pet: bool = button_index < GameState.pets.size() and GameState.pets[button_index] is Dictionary
        button.visible = has_pet
        button.disabled = not has_pet
        if not has_pet:
            continue
        var pet: Dictionary = GameState.pets[button_index] as Dictionary
        button.button_pressed = int(pet.get("pet_uid", 0)) == _selected_pet_uid
        _set_pet_choice_texture(button, _resolve_pet_slot_idle_texture_from_snapshot(pet))
        _set_pet_choice_deploy_visible(button, bool(pet.get("in_lineup", false)))


## 设置宠物列表按钮头像贴图；没有头像时清空，不显示客户端假数据。
func _set_pet_choice_texture(button: BaseButton, texture: Texture2D) -> void:
    var image_node: TextureRect = button.get_node_or_null("PetImage") as TextureRect
    if image_node == null:
        return
    var resolved_texture: Texture2D = texture
    if resolved_texture == null:
        var button_id: int = button.get_instance_id()
        var default_texture_variant: Variant = _default_pet_choice_textures.get(button_id, null)
        if default_texture_variant is Texture2D:
            resolved_texture = default_texture_variant as Texture2D
    image_node.texture = resolved_texture
    image_node.visible = resolved_texture != null


## 设置宠物列表按钮的出战标记。
func _set_pet_choice_deploy_visible(button: BaseButton, is_deployed: bool) -> void:
    var deploy_node: TextureRect = button.get_node_or_null("MarginContainer/Deploy") as TextureRect
    if deploy_node != null:
        deploy_node.visible = is_deployed


## 刷新顶部名称、等级和预览形象。
func _refresh_summary(pet: Dictionary) -> void:
    if pet.is_empty():
        if _pet_name_label != null:
            _pet_name_label.text = "暂无宠物"
        if _pet_level_label != null:
            _pet_level_label.text = ""
        if _preview_sprite != null:
            _preview_sprite.visible = false
        return
    if _pet_name_label != null:
        _pet_name_label.text = _resolve_pet_name(pet)
    if _pet_level_label != null:
        var level: int = int(pet.get("level", 0))
        _pet_level_label.text = "%s级" % UiFormat.value_to_text(level) if level > 0 else ""
    _refresh_preview_sprite(pet)


## 刷新基础属性页，字段缺失时展示 0 或空，避免继续保留场景模板数字。
func _refresh_basic_attributes(pet: Dictionary) -> void:
    var hp: int = int(pet.get("hp", 0))
    var hp_max: int = int(pet.get("hp_max", hp))
    var spirit: int = _snapshot_int(pet, ["spirit", "energy", "mp"], 0)
    var spirit_max: int = _snapshot_int(pet, ["spirit_max", "energy_max", "mp_max"], spirit)
    var mana: int = _snapshot_int(pet, ["mana", "mp_max"], 0)
    var hit_pct: int = _snapshot_int(pet, ["hit_pct", "hit"], 0)
    var dodge_pct: int = _snapshot_int(pet, ["dodge_pct", "dodge"], 0)
    var crit_rate_pct: int = _snapshot_int(pet, ["crit_rate_pct", "crit"], 0)
    var crit_dmg_pct: int = _snapshot_int(pet, ["crit_dmg_pct", "crit_damage"], 0)
    _set_row_value("基础属性/VBoxContainer/生命", UiFormat.value_to_text(hp), UiFormat.value_to_text(hp_max))
    _set_row_value("基础属性/VBoxContainer/元素属性", UiFormat.normalize_text(str(pet.get("element", pet.get("element_type", "")))))
    _set_row_value("基础属性/VBoxContainer/精力", UiFormat.value_to_text(spirit), UiFormat.value_to_text(spirit_max))
    _set_row_value("基础属性/VBoxContainer/经验", UiFormat.value_to_text(int(pet.get("exp", 0))))
    _set_row_value("基础属性/VBoxContainer/修炼与守护等级/修炼", UiFormat.value_to_text(int(pet.get("practice_level", 0))))
    _set_row_value("基础属性/VBoxContainer/修炼与守护等级/守护等级", UiFormat.value_to_text(int(pet.get("guard_level", 0))))
    _set_row_value("基础属性/VBoxContainer/攻击与防御/攻击", UiFormat.value_to_text(int(pet.get("atk", 0))))
    _set_row_value("基础属性/VBoxContainer/攻击与防御/防御", UiFormat.value_to_text(int(pet.get("def", 0))))
    _set_row_value("基础属性/VBoxContainer/速度与法力/速度", UiFormat.value_to_text(int(pet.get("spd", 0))))
    _set_row_value("基础属性/VBoxContainer/速度与法力/法力", UiFormat.value_to_text(mana))
    _set_row_value("基础属性/VBoxContainer/命中与闪避/命中", UiFormat.value_to_text(hit_pct))
    _set_row_value("基础属性/VBoxContainer/命中与闪避/闪避", UiFormat.value_to_text(dodge_pct))
    _set_row_value("基础属性/VBoxContainer/致命与爆伤/致命", UiFormat.value_to_text(crit_rate_pct))
    _set_row_value("基础属性/VBoxContainer/致命与爆伤/爆伤", "%s%%" % UiFormat.value_to_text(crit_dmg_pct))
    _set_row_value("基础属性/VBoxContainer/里世界潜力", UiFormat.value_to_text(int(pet.get("potential", 0))))


## 刷新状态抗性页，字段缺失时展示 0，避免保留场景模板数字。
func _refresh_status_resistance(pet: Dictionary) -> void:
    _set_status_value("状态抗性/VBoxContainer/物抗/物抗", _snapshot_int(pet, ["physical_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/物抗/逆物", _snapshot_int(pet, ["reverse_physical_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/技抗/技抗", _snapshot_int(pet, ["skill_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/技抗/逆技", _snapshot_int(pet, ["reverse_skill_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/混乱与昏睡/混乱", _snapshot_int(pet, ["confusion_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/混乱与昏睡/昏睡", _snapshot_int(pet, ["sleep_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/麻痹与封印/麻痹", _snapshot_int(pet, ["paralysis_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/麻痹与封印/封印", _snapshot_int(pet, ["seal_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/诅咒", _snapshot_int(pet, ["curse_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/抗致命与抗爆伤/抗致命", _snapshot_int(pet, ["crit_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/抗致命与抗爆伤/抗爆伤", _snapshot_int(pet, ["crit_dmg_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/抗人物与抗宠物/抗人物", _snapshot_int(pet, ["character_resist_pct"], 0))
    _set_status_value("状态抗性/VBoxContainer/抗人物与抗宠物/抗宠物", _snapshot_int(pet, ["pet_resist_pct"], 0))


## 刷新技能资质页，资质和技能槽均来自服务端宠物快照。
func _refresh_qualification_and_skills(pet: Dictionary) -> void:
    _set_qualification_row("攻击资质", _format_aptitude_text(pet, "atk"))
    _set_qualification_row("生命资质", _format_aptitude_text(pet, "hp"))
    _set_qualification_row("速度资质", _format_aptitude_text(pet, "spd"))
    _set_qualification_row("法力资质", _format_aptitude_text(pet, "mana"))
    _set_qualification_row("防御资质", _format_aptitude_text(pet, "def"))
    _refresh_skill_buttons(_collect_skill_entries(pet))


## 设置资质行的数值文案，字段缺失时覆盖为空，避免继续显示场景模板假数据。
func _set_qualification_row(row_name: String, value_text: String) -> void:
    if _skill_panel == null:
        return
    var label: Label = _skill_panel.get_node_or_null("资质与技能/VBoxContainer/%s/Label2" % row_name) as Label
    if label != null:
        label.text = value_text


## 根据服务端 growth_aptitudes 汇总值构建资质展示文案；基础/红色拆分功能未开放，暂不展示括号明细。
func _format_aptitude_text(pet: Dictionary, attr_key: String) -> String:
    if pet.is_empty():
        return ""
    var base_key: String = "base_%s_apt" % attr_key
    var extra_key: String = "extra_%s_apt" % attr_key
    var total_key: String = "%s_apt" % attr_key
    var base_value: int = int(pet.get(base_key, 0))
    var extra_value: int = int(pet.get(extra_key, 0))
    var total_value: int = _resolve_growth_aptitude_total(pet, total_key, base_value + extra_value)
    if total_value <= 0 and base_value <= 0 and extra_value <= 0:
        return "0"
    return UiFormat.value_to_text(total_value)


## 从 growth_aptitudes 读取服务端汇总资质；缺失时回退 base + extra。
func _resolve_growth_aptitude_total(pet: Dictionary, total_key: String, fallback_value: int) -> int:
    var growth_variant: Variant = pet.get("growth_aptitudes", {})
    if growth_variant is Dictionary:
        var growth: Dictionary = growth_variant as Dictionary
        if growth.has(total_key):
            return int(growth.get(total_key, fallback_value))
    return fallback_value


## 从服务端快照中提取技能展示列表。
## skill_ids 是服务端已经整理好的战斗技能顺序，客户端只负责按数组序号渲染到按钮 1~12。
## skill_slots 仅作为技能名、富文本描述等槽位元数据来源，避免分类空槽把图标挤到后面的按钮。
func _collect_skill_entries(pet: Dictionary) -> Array[Dictionary]:
    var slot_entries: Array[Dictionary] = []
    var slots_variant: Variant = pet.get("skill_slots", {})
    if slots_variant is Dictionary:
        var slots: Dictionary = slots_variant as Dictionary
        _append_skill_slot_array(slot_entries, slots.get("innate", []), "innate")
        _append_skill_slot_entry(slot_entries, slots.get("active_talisman", {}), "active_talisman")
        _append_skill_slot_entry(slot_entries, slots.get("talisman_hero", {}), "talisman_hero")
        _append_skill_slot_entry(slot_entries, slots.get("talisman_1", {}), "talisman_1")
        _append_skill_slot_entry(slot_entries, slots.get("talisman_2", {}), "talisman_2")
        _append_skill_slot_entry(slot_entries, slots.get("talisman_3", {}), "talisman_3")
        _append_skill_slot_array(slot_entries, slots.get("normal", []), "normal")
        _append_skill_slot_array(slot_entries, slots.get("artifact", []), "artifact")
    var skill_ids_variant: Variant = pet.get("skill_ids", [])
    if skill_ids_variant is Array:
        var ordered_entries: Array[Dictionary] = _collect_ordered_skill_entries_from_ids(skill_ids_variant as Array, slot_entries)
        if not ordered_entries.is_empty():
            return ordered_entries
    return _collect_non_empty_skill_slot_entries(slot_entries)


## 按服务端 skill_ids 的数组顺序生成技能按钮数据，并从 skill_slots 补齐技能名与描述。
func _collect_ordered_skill_entries_from_ids(skill_ids: Array, slot_entries: Array[Dictionary]) -> Array[Dictionary]:
    var result: Array[Dictionary] = []
    var metadata_by_skill_id: Dictionary = _index_skill_entries_by_id(slot_entries)
    var button_index: int = 0
    for skill_id_variant: Variant in skill_ids:
        var skill_id: int = int(skill_id_variant)
        if skill_id <= 0:
            continue
        var entry: Dictionary = {}
        var metadata_variant: Variant = metadata_by_skill_id.get(skill_id, {})
        if metadata_variant is Dictionary:
            entry = (metadata_variant as Dictionary).duplicate(true)
        entry["slot_index"] = button_index
        entry["slot_type"] = str(entry.get("slot_type", "skill_ids"))
        entry["skill_id"] = skill_id
        entry["enabled"] = true
        result.append(entry)
        button_index += 1
    return result


## 为 skill_slots 建立 skill_id 到元数据的索引；空槽不参与索引，避免覆盖真实技能描述。
func _index_skill_entries_by_id(entries: Array[Dictionary]) -> Dictionary:
    var result: Dictionary = {}
    for entry: Dictionary in entries:
        var skill_id: int = int(entry.get("skill_id", 0))
        if skill_id <= 0 or result.has(skill_id):
            continue
        result[skill_id] = entry.duplicate(true)
    return result


## 没有 skill_ids 时才退回到槽位自身顺序，并过滤空槽让已有技能从第一个按钮开始显示。
func _collect_non_empty_skill_slot_entries(entries: Array[Dictionary]) -> Array[Dictionary]:
    var result: Array[Dictionary] = []
    var button_index: int = 0
    for entry: Dictionary in entries:
        var skill_id: int = int(entry.get("skill_id", 0))
        if skill_id <= 0:
            continue
        var copied_entry: Dictionary = entry.duplicate(true)
        copied_entry["slot_index"] = button_index
        result.append(copied_entry)
        button_index += 1
    return result


## 追加一组技能槽位。
func _append_skill_slot_array(result: Array[Dictionary], value: Variant, slot_type: String) -> void:
    if value is not Array:
        return
    var entries: Array = value as Array
    for entry_variant: Variant in entries:
        _append_skill_slot_entry(result, entry_variant, slot_type)


## 追加单个技能槽位。
func _append_skill_slot_entry(result: Array[Dictionary], value: Variant, slot_type: String) -> void:
    if value is not Dictionary:
        return
    var entry: Dictionary = (value as Dictionary).duplicate(true)
    entry["slot_type"] = slot_type
    result.append(entry)


## 按服务端技能槽刷新固定按钮；空槽保留占位但不显示图标。
func _refresh_skill_buttons(skill_entries: Array[Dictionary]) -> void:
    _skill_entry_by_button_id.clear()
    for button_index: int in range(_skill_buttons.size()):
        var button: BaseButton = _skill_buttons[button_index]
        var entry: Dictionary = skill_entries[button_index] if button_index < skill_entries.size() else {}
        var skill_id: int = int(entry.get("skill_id", 0))
        var has_skill: bool = skill_id > 0
        if has_skill:
            _skill_entry_by_button_id[button.get_instance_id()] = entry.duplicate(true)
        button.visible = true
        button.disabled = not has_skill
        button.button_pressed = false
        button.tooltip_text = _build_skill_tooltip(entry)
        button.theme = _resolve_skill_button_theme(str(entry.get("skill_quality", "normal")))
        _set_skill_button_icon(button, _resolve_skill_icon(entry), has_skill)


## 按服务端技能品质选择场景配置的按钮边框主题，未知值回退普通品质。
func _resolve_skill_button_theme(skill_quality: String) -> Theme:
    match skill_quality.strip_edges().to_lower():
        "divine":
            return divine_skill_button_theme if divine_skill_button_theme != null else normal_skill_button_theme
        "soul":
            return soul_skill_button_theme if soul_skill_button_theme != null else normal_skill_button_theme
        "sacred":
            return sacred_skill_button_theme if sacred_skill_button_theme != null else normal_skill_button_theme
        "peerless":
            return peerless_skill_button_theme if peerless_skill_button_theme != null else normal_skill_button_theme
        _:
            return normal_skill_button_theme


## 生成技能槽提示文案，移动端无 tooltip 时也不影响主显示。
func _build_skill_tooltip(entry: Dictionary) -> String:
    var skill_id: int = int(entry.get("skill_id", 0))
    if skill_id <= 0:
        return "空技能槽"
    return _resolve_skill_display_name(entry)


## 鼠标悬停技能按钮时展示技能详情；桌面调试可直接预览服务端描述。
func _on_skill_button_hovered(button: BaseButton) -> void:
    _show_skill_description_for_button(button, true)


## 鼠标离开技能按钮时关闭由悬停打开的技能详情；触屏按下打开的详情不受该逻辑影响。
func _on_skill_button_unhovered(button: BaseButton) -> void:
    # 弹窗打开后全屏遮罩会盖在按钮上方，Godot 可能立刻触发 mouse_exited。
    # 这里不直接关闭，统一交给 _process 用真实鼠标坐标判断是否已经离开按钮区域。
    if button == null:
        return


## 移动端按下技能按钮时展示技能详情；没有悬停能力的触屏设备通过该入口查看。
func _on_skill_button_down(button: BaseButton) -> void:
    _show_skill_description_for_button(button, false)


## 使用通用确认提示弹窗渲染单个技能槽的服务端技能名和富文本描述；close_on_mouse_exit 表示是否随悬停离开自动关闭。
func _show_skill_description_for_button(button: BaseButton, close_on_mouse_exit: bool) -> void:
    if button == null or button.disabled:
        return
    if close_on_mouse_exit and _skill_hover_source_button == button and _skill_description_popup != null and _skill_description_popup.visible:
        return
    var entry_variant: Variant = _skill_entry_by_button_id.get(button.get_instance_id(), {})
    if entry_variant is not Dictionary:
        return
    var entry: Dictionary = entry_variant as Dictionary
    if int(entry.get("skill_id", 0)) <= 0:
        return
    var popup: ConfirmPromptPopup = _ensure_skill_description_popup()
    if popup == null:
        return
    popup.show_prompt(_resolve_skill_display_name(entry), _resolve_skill_description(entry), {
        "confirm_label": "关闭",
        "content_font_size": 24,
    })
    _apply_skill_description_popup_position()
    _skill_hover_source_button = button if close_on_mouse_exit else null
    set_process(close_on_mouse_exit)


## 懒创建技能说明弹窗，并作为宠物面板子节点统一随面板生命周期清理。
func _ensure_skill_description_popup() -> ConfirmPromptPopup:
    if _skill_description_popup != null:
        return _skill_description_popup
    _skill_description_popup = ConfirmPromptPopupScene.instantiate() as ConfirmPromptPopup
    if _skill_description_popup == null:
        return null
    _skill_description_popup.name = "PetSkillDescriptionPopup"
    add_child(_skill_description_popup)
    _apply_skill_description_popup_position()
    return _skill_description_popup


## 应用 Inspector 中配置的技能说明弹窗位置，方便按实际界面遮挡情况调整。
func _apply_skill_description_popup_position() -> void:
    if _skill_description_popup == null:
        return
    _skill_description_popup.set_popup_position(skill_description_popup_position)


## 面板关闭或切出技能页时收起技能说明，避免旧技能描述残留到其他分页。
func _close_skill_description_popup() -> void:
    _skill_hover_source_button = null
    set_process(false)
    if _skill_description_popup != null:
        _skill_description_popup.close_prompt()


## 解析服务端下发的技能展示名；字段缺失时才回退到技能 ID，避免客户端硬编码技能名。
func _resolve_skill_display_name(entry: Dictionary) -> String:
    var skill_name: String = str(entry.get("skill_name", entry.get("name", entry.get("display_name", "")))).strip_edges()
    if not skill_name.is_empty():
        return UiFormat.normalize_text(skill_name)
    return "技能 %s" % UiFormat.value_to_text(int(entry.get("skill_id", 0)))


## 解析服务端下发的技能富文本描述；字段缺失时给出安全兜底提示。
func _resolve_skill_description(entry: Dictionary) -> String:
    var description: String = str(entry.get("description", entry.get("desc", ""))).strip_edges()
    if not description.is_empty():
        return description
    return "服务端暂未配置该技能描述。"


## 优先按服务端 skill_visual_id 读取本地技能表现图标，并兼容旧版图标路径字段。
func _resolve_skill_icon(entry: Dictionary) -> Texture2D:
    var skill_visual_id: String = str(entry.get("skill_visual_id", "")).strip_edges()
    if _skill_visual_registry != null and not skill_visual_id.is_empty():
        var configured_icon: Texture2D = _skill_visual_registry.get_skill_icon(skill_visual_id)
        if configured_icon != null:
            return configured_icon
    var icon_path: String = str(entry.get("icon", entry.get("skill_icon", entry.get("icon_path", "")))).strip_edges()
    if not icon_path.is_empty() and ResourceLoader.exists(icon_path):
        var resource: Resource = load(icon_path)
        if resource is Texture2D:
            return resource as Texture2D
    return _default_skill_icon


## 设置单个技能按钮图标。
func _set_skill_button_icon(button: BaseButton, texture: Texture2D, should_show: bool) -> void:
    var icon_node: TextureRect = _resolve_skill_button_icon(button)
    if icon_node == null:
        return
    icon_node.texture = texture
    icon_node.visible = should_show and texture != null


## 解析技能按钮中的图标节点；优先使用新增 Icon 节点，兼容旧 TextureRect 名称。
func _resolve_skill_button_icon(button: BaseButton) -> TextureRect:
    if button == null:
        return null
    var icon_node: TextureRect = button.get_node_or_null("Icon") as TextureRect
    if icon_node != null:
        return icon_node
    return button.get_node_or_null("TextureRect") as TextureRect


## 按候选字段读取服务端整数快照；兼容旧字段名，优先使用当前协议字段。
func _snapshot_int(snapshot: Dictionary, keys: Array[String], default_value: int) -> int:
    for key: String in keys:
        if snapshot.has(key):
            return int(snapshot.get(key, default_value))
    return default_value


## 设置基础属性行的主值和可选上限值。
func _set_row_value(row_path: String, value_text: String, max_text: String = "") -> void:
    if _basic_panel == null:
        return
    var row: Node = _basic_panel.get_node_or_null(NodePath(row_path))
    if row == null:
        return
    var value_label: Label = row.get_node_or_null("Label2") as Label
    if value_label != null:
        value_label.text = value_text
    var max_label: Label = row.get_node_or_null("Label4") as Label
    if max_label != null:
        max_label.text = max_text


## 设置状态抗性页单个数值标签。
func _set_status_value(row_path: String, value: int) -> void:
    if _status_panel == null:
        return
    var row: Node = _status_panel.get_node_or_null(NodePath(row_path))
    if row == null:
        return
    var value_label: Label = row.get_node_or_null("Label2") as Label
    if value_label != null:
        value_label.text = UiFormat.value_to_text(value)


## 解析宠物展示名；自定义名优先，没有自定义名时展示系统宠物名。
func _resolve_pet_name(pet: Dictionary) -> String:
    var custom_name: String = str(pet.get("custom_name", "")).strip_edges()
    if not custom_name.is_empty():
        return UiFormat.normalize_text(custom_name)
    var system_name: String = str(pet.get("pet_name", pet.get("system_pet_name", pet.get("definition_name", "")))).strip_edges()
    if not system_name.is_empty():
        return UiFormat.normalize_text(system_name)
    var display_name: String = str(pet.get("name", "")).strip_edges()
    if not display_name.is_empty():
        return UiFormat.normalize_text(display_name)
    var pet_id: int = int(pet.get("pet_id", 0))
    if pet_id > 0:
        return "宠物%s" % UiFormat.value_to_text(pet_id)
    var pet_uid: int = int(pet.get("pet_uid", 0))
    if pet_uid > 0:
        return "宠物#%s" % UiFormat.value_to_text(pet_uid)
    return "宠物"


## 从宠物快照解析宠物槽待机下方向第一帧；优先使用服务端 skin_id，兼容旧快照的名称别名。
func _resolve_pet_slot_idle_texture_from_snapshot(pet: Dictionary) -> Texture2D:
    var candidate_keys: Array[String] = ["skin_id", "appearance_skin_id", "unit_skin_id", "pet_skin_id", "pet_name", "name"]
    for candidate_key: String in candidate_keys:
        var candidate_id: String = str(pet.get(candidate_key, "")).strip_edges()
        if candidate_id.is_empty():
            continue
        var texture: Texture2D = _resolve_pet_slot_idle_texture(candidate_id)
        if texture != null:
            return texture
    return null


## 在 DisplayPanel 的 AnimatedSprite2D 上展示当前宠物待机形象。
func _refresh_preview_sprite(pet: Dictionary) -> void:
    if _preview_sprite == null:
        return
    if pet.is_empty():
        _preview_sprite.visible = false
        _preview_sprite.stop()
        return
    var skin: UnitSkin = _resolve_skin_from_pet_snapshot(pet)
    if skin == null:
        _preview_sprite.visible = false
        _preview_sprite.stop()
        return
    var animation_name: String = _resolve_preview_animation_name(skin)
    if not animation_name.is_empty() and skin.sprite_frames != null:
        _preview_sprite.sprite_frames = skin.sprite_frames
        _preview_sprite.animation = animation_name
        _preview_sprite.visible = true
        _preview_sprite.play(animation_name)
        return
    var idle_texture: Texture2D = skin.resolve_avatar_preview_texture()
    if idle_texture == null:
        _preview_sprite.visible = false
        _preview_sprite.stop()
        return
    _preview_sprite.sprite_frames = _build_single_frame_preview_frames(idle_texture)
    _preview_sprite.animation = "default"
    _preview_sprite.visible = true
    _preview_sprite.play("default")


## 从宠物快照候选字段解析 UnitSkin，确保展示层不按 pet_id 硬编码推断资源。
func _resolve_skin_from_pet_snapshot(pet: Dictionary) -> UnitSkin:
    var candidate_keys: Array[String] = ["skin_id", "appearance_skin_id", "unit_skin_id", "pet_skin_id", "pet_name", "name"]
    for candidate_key: String in candidate_keys:
        var candidate_id: String = str(pet.get(candidate_key, "")).strip_edges()
        if candidate_id.is_empty():
            continue
        var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(candidate_id)
        if skin != null:
            return skin
    return null


## 选择 DisplayPanel 播放的待机动画；优先世界下待机，其次资源声明的头像/战斗待机。
func _resolve_preview_animation_name(skin: UnitSkin) -> String:
    if skin == null or skin.sprite_frames == null:
        return ""
    var down_idle_animation_names: Array[String] = ["下待机", "待机下", "idle_down", "down_idle"]
    for animation_name: String in down_idle_animation_names:
        if skin.sprite_frames.has_animation(animation_name) and skin.sprite_frames.get_frame_count(animation_name) > 0:
            return animation_name
    var avatar_animation: String = skin.resolve_avatar_preview_animation()
    if not avatar_animation.is_empty():
        return avatar_animation
    var battle_idle_animation: String = skin.get_battle_idle_png_override()
    if not battle_idle_animation.is_empty():
        return battle_idle_animation
    return ""


## 把 CHJ 或静态贴图包装成 AnimatedSprite2D 可播放的单帧资源。
func _build_single_frame_preview_frames(texture: Texture2D) -> SpriteFrames:
    var frames: SpriteFrames = SpriteFrames.new()
    frames.add_animation("default")
    frames.set_animation_loop("default", true)
    frames.set_animation_speed("default", 1.0)
    frames.add_frame("default", texture)
    return frames


## 通过服务端下发的 skin_id 或兼容别名解析宠物槽待机下方向第一帧。
func _resolve_pet_slot_idle_texture(skin_id: String) -> Texture2D:
    var normalized_skin_id: String = skin_id.strip_edges()
    if normalized_skin_id.is_empty():
        return null
    var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(normalized_skin_id)
    if skin == null:
        return null
    var down_idle_animation_names: Array[String] = ["下待机", "待机下", "idle_down", "down_idle"]
    var down_idle_texture: Texture2D = _resolve_skin_animation_first_frame(skin, down_idle_animation_names)
    if down_idle_texture != null:
        return down_idle_texture
    return skin.resolve_avatar_preview_texture()


## 从 UnitSkin 的 SpriteFrames 中读取候选动画第一帧，保证宠物槽优先使用“下待机”资源。
func _resolve_skin_animation_first_frame(skin: UnitSkin, animation_names: Array[String]) -> Texture2D:
    if skin == null or skin.sprite_frames == null:
        return null
    for animation_name: String in animation_names:
        if animation_name.is_empty():
            continue
        if not skin.sprite_frames.has_animation(animation_name):
            continue
        if skin.sprite_frames.get_frame_count(animation_name) <= 0:
            continue
        var frame_texture: Texture2D = skin.sprite_frames.get_frame_texture(animation_name, 0)
        if frame_texture != null:
            return frame_texture
    return null
