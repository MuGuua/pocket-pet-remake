extends RuntimeRootPanel
class_name PetStatusPanel

## 形象注册表用于把服务端下发的 skin_id 转成客户端头像贴图。
const CharacterSkinRegistry = preload("res://scripts/feature/character/character_skin_registry.gd")
## 等待 PET_LIST_RESP 的最大帧数，避免网络异常时面板打开流程永久挂起。
const OPEN_REQUEST_TIMEOUT_FRAMES: int = 300
## 宠物基础属性页在场景中的相对路径，集中管理便于后续 UI 调整。
const BASIC_PANEL_PATH: NodePath = NodePath("PanelContainer/MarginContainer/VBoxContainer/DataPanel/HBoxContainer/数据/VBoxContainer/数据面板/基础属性")

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

## 当前宠物列表按钮集合。
var _pet_buttons: Array[BaseButton] = []
## 当前选中的宠物 uid。
var _selected_pet_uid: int = 0
## 面板打开请求代次，用于关闭或重复打开时取消旧 await。
var _open_load_generation: int = 0
## 当前等待的 PET_LIST_REQ 序列号。
var _pending_pet_list_seq: int = 0
## PET_LIST_REQ 是否已有回调结果。
var _pending_pet_list_ready: bool = false
## PET_LIST_REQ 是否成功。
var _pending_pet_list_ok: bool = false


## 初始化面板事件、按钮缓存与权威状态订阅。
func _ready() -> void:
    super._ready()
    _collect_pet_buttons()
    if _close_button != null and not _close_button.button_down.is_connected(_on_close_button_pressed):
        _close_button.button_down.connect(_on_close_button_pressed)
    if not GameState.pets_changed.is_connected(refresh_panel_data):
        GameState.pets_changed.connect(refresh_panel_data)
    if not App.request_finished.is_connected(_on_request_finished):
        App.request_finished.connect(_on_request_finished)
    refresh_panel_data()


## 断开全局信号，避免面板销毁后继续刷新。
func _exit_tree() -> void:
    if GameState.pets_changed.is_connected(refresh_panel_data):
        GameState.pets_changed.disconnect(refresh_panel_data)
    if App.request_finished.is_connected(_on_request_finished):
        App.request_finished.disconnect(_on_request_finished)


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
    _pending_pet_list_seq = 0
    return _pending_pet_list_ok


## 数据准备完成后打开宠物面板，并刷新展示。
func open_menu() -> void:
    super.open_menu()
    refresh_panel_data()


## 关闭宠物面板并取消仍在等待的打开请求。
func close_menu() -> void:
    _open_load_generation += 1
    _pending_pet_list_seq = 0
    super.close_menu()


## 按当前 GameState 中的服务端宠物快照刷新列表、摘要和基础属性。
func refresh_panel_data() -> void:
    _ensure_selected_pet()
    _refresh_pet_buttons()
    var selected_pet: Dictionary = _resolve_selected_pet()
    _refresh_summary(selected_pet)
    _refresh_basic_attributes(selected_pet)


## 收集左侧宠物按钮，并绑定点击切换选中宠物。
func _collect_pet_buttons() -> void:
    _pet_buttons.clear()
    if _pet_grid == null:
        return
    var button_index: int = 0
    for child: Node in _pet_grid.get_children():
        if child is BaseButton:
            var button: BaseButton = child as BaseButton
            button.toggle_mode = true
            _pet_buttons.append(button)
            if not button.pressed.is_connected(_on_pet_button_pressed.bind(button_index)):
                button.pressed.connect(_on_pet_button_pressed.bind(button_index))
            button_index += 1


## 响应左侧宠物按钮点击，切换当前查看的宠物。
func _on_pet_button_pressed(button_index: int) -> void:
    if button_index < 0 or button_index >= GameState.pets.size():
        return
    var pet_variant: Variant = GameState.pets[button_index]
    if pet_variant is not Dictionary:
        return
    var pet: Dictionary = pet_variant as Dictionary
    _selected_pet_uid = int(pet.get("pet_uid", 0))
    refresh_panel_data()


## 响应标题栏关闭按钮。
func _on_close_button_pressed() -> void:
    close_menu()


## 处理 PET_LIST_REQ 回包结果，只接收当前打开流程的同一个 seq。
func _on_request_finished(request_cmd: int, seq: int, succeeded: bool, _response_cmd: int, _payload: Dictionary) -> void:
    if request_cmd != CommandIds.PET_LIST_REQ:
        return
    if _pending_pet_list_seq <= 0 or seq != _pending_pet_list_seq:
        return
    _pending_pet_list_ready = true
    _pending_pet_list_ok = succeeded


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
        _set_pet_choice_texture(button, _resolve_avatar_texture(str(pet.get("skin_id", ""))))
        _set_pet_choice_deploy_visible(button, bool(pet.get("in_lineup", false)))


## 设置宠物列表按钮头像贴图；没有头像时清空，不显示客户端假数据。
func _set_pet_choice_texture(button: BaseButton, texture: Texture2D) -> void:
    var image_node: TextureRect = button.get_node_or_null("PetImage") as TextureRect
    if image_node == null:
        return
    image_node.texture = texture
    image_node.visible = texture != null


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
    if _preview_sprite != null:
        # 旧场景里预置的是模板演示帧；未接入真实动态资源前先隐藏，避免误展示假宠物。
        _preview_sprite.visible = false


## 刷新基础属性页，字段缺失时展示 0 或空，避免继续保留场景模板数字。
func _refresh_basic_attributes(pet: Dictionary) -> void:
    var hp: int = int(pet.get("hp", 0))
    var hp_max: int = int(pet.get("hp_max", hp))
    var mp: int = int(pet.get("mp", pet.get("energy", 0)))
    var mp_max: int = int(pet.get("mp_max", pet.get("energy_max", mp)))
    _set_row_value("基础属性/VBoxContainer/生命", UiFormat.value_to_text(hp), UiFormat.value_to_text(hp_max))
    _set_row_value("基础属性/VBoxContainer/元素属性", UiFormat.normalize_text(str(pet.get("element", pet.get("element_type", "")))))
    _set_row_value("基础属性/VBoxContainer/精力", UiFormat.value_to_text(mp), UiFormat.value_to_text(mp_max))
    _set_row_value("基础属性/VBoxContainer/经验", UiFormat.value_to_text(int(pet.get("exp", 0))))
    _set_row_value("基础属性/VBoxContainer/修炼与守护等级/修炼", UiFormat.value_to_text(int(pet.get("practice_level", 0))))
    _set_row_value("基础属性/VBoxContainer/修炼与守护等级/守护等级", UiFormat.value_to_text(int(pet.get("guard_level", 0))))
    _set_row_value("基础属性/VBoxContainer/攻击与防御/攻击", UiFormat.value_to_text(int(pet.get("atk", 0))))
    _set_row_value("基础属性/VBoxContainer/攻击与防御/防御", UiFormat.value_to_text(int(pet.get("def", 0))))
    _set_row_value("基础属性/VBoxContainer/速度与法力/速度", UiFormat.value_to_text(int(pet.get("spd", 0))))
    _set_row_value("基础属性/VBoxContainer/速度与法力/法力", UiFormat.value_to_text(int(pet.get("mp_max", 0))))
    _set_row_value("基础属性/VBoxContainer/命中与闪避/命中", UiFormat.value_to_text(int(pet.get("hit", 0))))
    _set_row_value("基础属性/VBoxContainer/命中与闪避/闪避", UiFormat.value_to_text(int(pet.get("dodge", 0))))
    _set_row_value("基础属性/VBoxContainer/致命与爆伤/致命", UiFormat.value_to_text(int(pet.get("crit", 0))))
    _set_row_value("基础属性/VBoxContainer/致命与爆伤/爆伤", "%s%%" % UiFormat.value_to_text(int(pet.get("crit_damage", 0))))
    _set_row_value("基础属性/VBoxContainer/里世界潜力", UiFormat.value_to_text(int(pet.get("potential", 0))))


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


## 解析宠物展示名，优先使用服务端名称，没有名称时回退为编号。
func _resolve_pet_name(pet: Dictionary) -> String:
    var pet_name: String = str(pet.get("name", pet.get("pet_name", ""))).strip_edges()
    if not pet_name.is_empty():
        return UiFormat.normalize_text(pet_name)
    var pet_id: int = int(pet.get("pet_id", 0))
    if pet_id > 0:
        return "宠物%s" % UiFormat.value_to_text(pet_id)
    var pet_uid: int = int(pet.get("pet_uid", 0))
    if pet_uid > 0:
        return "宠物#%s" % UiFormat.value_to_text(pet_uid)
    return "宠物"


## 通过服务端下发的 skin_id 解析头像预览贴图。
func _resolve_avatar_texture(skin_id: String) -> Texture2D:
    var normalized_skin_id: String = skin_id.strip_edges()
    if normalized_skin_id.is_empty():
        return null
    var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(normalized_skin_id)
    if skin == null:
        return null
    return skin.resolve_avatar_preview_texture()
