extends PanelContainer
class_name PetStatusHud

## 点击宠物 HUD 时触发，由主场景打开宠物状态面板。
signal pet_pressed

## 与战斗表现层一致：unit_class 2 表示己方宠物单位。
const PET_UNIT_CLASS: int = 2

## 可点击的宠物头像按钮。
@onready var _pet_button: TextureButton = %PetButton
## 宠物头像贴图展示节点。
@onready var _pet_texture: TextureRect = %PetTexture
## 宠物生命值进度条。
@onready var _hp_bar: ProgressBar = %HpBar
## 宠物法力值进度条，颜色与人物 HUD 的蓝色法力条保持一致。
@onready var _mp_bar: ProgressBar = %MpBar
## 宠物经验值进度条，颜色与人物 HUD 的绿色经验条保持一致。
@onready var _exp_bar: ProgressBar = %ExpBar
## 宠物当前生命与生命上限文本。
@onready var _hp_value_label: Label = %HpValueLabel
## 宠物当前法力与法力上限文本。
@onready var _mp_value_label: Label = %MpValueLabel
## 宠物当前经验与本级升级所需总经验文本。
@onready var _exp_value_label: Label = %ExpValueLabel
## 宠物名称与等级摘要。
@onready var _name_label: RichTextLabel = %NameLabel

## 当前 HUD 展示的宠物唯一标识，用于战斗态匹配实时生命。
var _current_pet_uid: int = 0
## 外部 HUD 总控设置的显隐开关；战斗弹窗期间为 false，避免状态刷新把 HUD 重新显示出来。
var _hud_enabled: bool = true


## 初始化宠物 HUD：订阅宠物/战斗快照变化，并绑定点击事件。
func _ready() -> void:
    if _pet_button != null and not _pet_button.pressed.is_connected(_on_pet_button_pressed):
        _pet_button.pressed.connect(_on_pet_button_pressed)
    if not GameState.pets_changed.is_connected(refresh_from_game_state):
        GameState.pets_changed.connect(refresh_from_game_state)
    if not GameState.battle_changed.is_connected(refresh_from_game_state):
        GameState.battle_changed.connect(refresh_from_game_state)
    refresh_from_game_state()


## 退出场景时断开全局信号，避免热重载或切场景后重复回调。
func _exit_tree() -> void:
    if GameState.pets_changed.is_connected(refresh_from_game_state):
        GameState.pets_changed.disconnect(refresh_from_game_state)
    if GameState.battle_changed.is_connected(refresh_from_game_state):
        GameState.battle_changed.disconnect(refresh_from_game_state)


## 按当前服务端权威宠物快照刷新头像、名称、生命条、法力条与经验条。
func refresh_from_game_state() -> void:
    var pet: Dictionary = _resolve_display_pet()
    _current_pet_uid = int(pet.get("pet_uid", 0))
    _update_name_label(pet)
    _update_pet_texture(pet)
    _update_hp_bar(pet)
    _update_mp_bar(pet)
    _update_exp_bar(pet)
    visible = _hud_enabled and not pet.is_empty()


## 由 RuntimeHud 控制 HUD 整体显隐，避免战斗中刷新宠物数据时重新露出。
func set_hud_enabled(enabled: bool) -> void:
    _hud_enabled = enabled
    refresh_from_game_state()


## 处理宠物 HUD 点击，打开宠物状态面板。
func _on_pet_button_pressed() -> void:
    pet_pressed.emit()


## 优先展示首只出战宠物；没有编队时退回第一只拥有的宠物。
func _resolve_display_pet() -> Dictionary:
    if not GameState.lineup.is_empty():
        var lineup_variant: Variant = GameState.lineup[0]
        if lineup_variant is Dictionary:
            var lineup_pet: Dictionary = (lineup_variant as Dictionary).duplicate(true)
            var full_pet: Dictionary = _resolve_full_pet_by_uid(int(lineup_pet.get("pet_uid", 0)))
            if not full_pet.is_empty():
                full_pet.merge(lineup_pet, true)
                return full_pet
            return lineup_pet
    if not GameState.pets.is_empty():
        var pet_variant: Variant = GameState.pets[0]
        if pet_variant is Dictionary:
            return (pet_variant as Dictionary).duplicate(true)
    return {}


## 按宠物 uid 从完整宠物列表里查找服务端快照，补齐编队摘要里没有的生命和属性字段。
func _resolve_full_pet_by_uid(pet_uid: int) -> Dictionary:
    if pet_uid <= 0:
        return {}
    for pet_variant: Variant in GameState.pets:
        if pet_variant is not Dictionary:
            continue
        var pet: Dictionary = pet_variant as Dictionary
        if int(pet.get("pet_uid", 0)) == pet_uid:
            return pet.duplicate(true)
    return {}


## 刷新宠物 HUD 名称；名称缺失时用 pet_id/pet_uid 兜底展示。
func _update_name_label(pet: Dictionary) -> void:
    if _name_label == null:
        return
    if pet.is_empty():
        _name_label.text = "宠物"
        return
    var custom_name: String = str(pet.get("custom_name", "")).strip_edges()
    var pet_name: String = custom_name
    if pet_name.is_empty():
        pet_name = str(pet.get("pet_name", pet.get("system_pet_name", pet.get("name", "")))).strip_edges()
    if pet_name.is_empty():
        var pet_id: int = int(pet.get("pet_id", 0))
        var pet_uid: int = int(pet.get("pet_uid", 0))
        if pet_id > 0:
            pet_name = "宠物%s" % UiFormat.value_to_text(pet_id)
        elif pet_uid > 0:
            pet_name = "宠物#%s" % UiFormat.value_to_text(pet_uid)
        else:
            pet_name = "宠物"
    var pet_level: int = int(pet.get("level", 0))
    var display_text: String = pet_name
    if pet_level > 0:
        display_text = "%s Lv.%s" % [pet_name, UiFormat.value_to_text(pet_level)]
    if custom_name.is_empty():
        RichTextContent.apply_system_name(_name_label, display_text)
    else:
        RichTextContent.apply_plain_name(_name_label, display_text)


## 刷新宠物头像贴图；skin_id 缺失时保持为空，不在客户端硬编码假宠物形象。
func _update_pet_texture(pet: Dictionary) -> void:
    if _pet_texture == null:
        return
    var skin_id: String = str(pet.get("skin_id", "")).strip_edges()
    if GameState.is_in_battle:
        var battle_actor: Dictionary = _resolve_pet_battle_actor()
        var battle_skin_id: String = str(battle_actor.get("skin_id", "")).strip_edges()
        if not battle_skin_id.is_empty():
            skin_id = battle_skin_id
    var avatar_texture: Texture2D = _resolve_avatar_texture(skin_id)
    _pet_texture.texture = avatar_texture


## 通过形象注册表解析 HUD 头像预览贴图。
func _resolve_avatar_texture(skin_id: String) -> Texture2D:
    var normalized_skin_id: String = skin_id.strip_edges()
    if normalized_skin_id.is_empty():
        return null
    var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(normalized_skin_id)
    if skin == null:
        return null
    return skin.resolve_avatar_preview_texture()


## 刷新生命条；战斗态优先使用战斗快照里的实时 HP。
func _update_hp_bar(pet: Dictionary) -> void:
    if _hp_bar == null:
        return
    var hp_current: int = int(pet.get("hp", 0))
    var hp_max: int = max(1, int(pet.get("hp_max", hp_current)))
    if GameState.is_in_battle:
        var battle_actor: Dictionary = _resolve_pet_battle_actor()
        if not battle_actor.is_empty():
            hp_current = int(battle_actor.get("hp", hp_current))
            hp_max = max(1, int(battle_actor.get("hp_max", hp_max)))
    _hp_bar.max_value = float(hp_max)
    _hp_bar.value = float(clampi(hp_current, 0, hp_max))
    if _hp_value_label != null:
        _hp_value_label.text = _build_status_value_text("生命", hp_current, hp_max)


## 刷新法力条；宠物当前协议只有 mana 面板值，所以按人物 HUD 的现有口径用 mana 作为上限。
func _update_mp_bar(pet: Dictionary) -> void:
    if _mp_bar == null:
        return
    var mp_current: int = int(pet.get("mana", 0))
    var mp_max: int = max(1, int(pet.get("mana_max", mp_current)))
    if GameState.is_in_battle:
        var battle_actor: Dictionary = _resolve_pet_battle_actor()
        if not battle_actor.is_empty():
            mp_current = int(battle_actor.get("mana", mp_current))
            mp_max = max(1, int(battle_actor.get("mana_max", mp_current)))
    _mp_bar.max_value = float(mp_max)
    _mp_bar.value = float(clampi(mp_current, 0, mp_max))
    if _mp_value_label != null:
        _mp_value_label.text = _build_status_value_text("法力", mp_current, mp_max)


## 刷新经验条；exp_to_next 是服务端下发的“距离下级还差多少经验”。
func _update_exp_bar(pet: Dictionary) -> void:
    if _exp_bar == null:
        return
    var exp_current: int = int(pet.get("exp", 0))
    var exp_to_next: int = int(pet.get("exp_to_next", 0))
    var pet_level: int = int(pet.get("level", 0))
    # 进度条总需求 = 当前等级已获得经验 + 距离下级所需经验；满级或缺配置时用满条兜底。
    var exp_max: int = exp_current + exp_to_next
    if exp_max <= 0:
        if pet_level >= 100:
            exp_current = 1
            exp_max = 1
        else:
            exp_max = 1
    _exp_bar.max_value = float(exp_max)
    _exp_bar.value = float(clampi(exp_current, 0, exp_max))
    if _exp_value_label != null:
        if pet_level >= 100 and exp_to_next <= 0:
            _exp_value_label.text = "经验 满级"
        else:
            _exp_value_label.text = _build_status_value_text("经验", exp_current, exp_max)


## 构建状态条整数文本，避免移动端 HUD 展示浮点值。
## status_name 是生命、法力或经验等服务端权威数值名称。
## current_value 是当前值，maximum_value 是对应上限或本级升级所需总值。
func _build_status_value_text(status_name: String, current_value: int, maximum_value: int) -> String:
    return "%s %s/%s" % [
        status_name,
        UiFormat.value_to_text(current_value),
        UiFormat.value_to_text(maximum_value),
    ]


## 从战斗 allies 列表中定位当前 HUD 展示的出战宠物。
func _resolve_pet_battle_actor() -> Dictionary:
    var allies_variant: Variant = GameState.battle_state.get("allies", [])
    if allies_variant is not Array:
        return {}
    for actor_variant: Variant in allies_variant as Array:
        if actor_variant is not Dictionary:
            continue
        var actor: Dictionary = actor_variant as Dictionary
        if int(actor.get("unit_class", 0)) != PET_UNIT_CLASS:
            continue
        if _current_pet_uid > 0 and int(actor.get("pet_uid", 0)) != _current_pet_uid:
            continue
        return actor.duplicate(true)
    return {}
