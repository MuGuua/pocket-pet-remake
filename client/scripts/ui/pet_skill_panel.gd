class_name PetSkillPanel
extends CanvasLayer

const RequestLoadingOverlay = preload("res://scripts/ui/request_loading_overlay.gd")

## 当前展示的宠物实例唯一标识。
var _pet_uid: int = 0
## 正在等待服务端回包的请求序列号。
var _loading_request_seq: int = 0
## 通用 loading 遮罩。
var _request_loading: RequestLoadingOverlay = null
## 技能内容滚动区。
var _content_vbox: VBoxContainer = null
## 标题标签。
var _title_label: Label = null
## 提示标签。
var _hint_label: Label = null
## 从背包选择法宝的对话框。
var _artifact_bag_pick_dialog: AcceptDialog = null
## 背包法宝物品下拉框。
var _artifact_bag_option: OptionButton = null
## 待装备的目标法宝槽位索引。
var _pending_equip_artifact_slot_index: int = -1


## 初始化面板 UI 并默认隐藏。
func _ready() -> void:
    layer = 45
    _build_ui()
    hide_panel()


## 打开指定宠物的技能详情面板，并向服务端请求完整 skill_slots。
func open_for_pet(pet_uid: int) -> void:
    if pet_uid <= 0:
        return
    _pet_uid = pet_uid
    show()
    _refresh_title()
    _request_skill_detail()


## 关闭面板并清理等待状态。
func hide_panel() -> void:
    _loading_request_seq = 0
    if _request_loading != null:
        _request_loading.hide_overlay()
    hide()


## 构建运行时 UI 结构。
func _build_ui() -> void:
    var root: Control = Control.new()
    root.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
    root.mouse_filter = Control.MOUSE_FILTER_STOP
    add_child(root)

    var dim: ColorRect = ColorRect.new()
    dim.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
    dim.color = Color(0.04, 0.08, 0.14, 0.72)
    dim.mouse_filter = Control.MOUSE_FILTER_STOP
    root.add_child(dim)

    var panel: PanelContainer = PanelContainer.new()
    panel.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
    panel.custom_minimum_size = Vector2(360.0, 480.0)
    panel.position = Vector2(-180.0, -240.0)
    root.add_child(panel)

    var outer_vbox: VBoxContainer = VBoxContainer.new()
    outer_vbox.add_theme_constant_override("separation", 8)
    panel.add_child(outer_vbox)

    var header_row: HBoxContainer = HBoxContainer.new()
    outer_vbox.add_child(header_row)

    _title_label = Label.new()
    _title_label.text = "宠物技能"
    _title_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
    header_row.add_child(_title_label)

    var close_button: Button = Button.new()
    close_button.text = "关闭"
    close_button.pressed.connect(_on_close_pressed)
    header_row.add_child(close_button)

    _hint_label = Label.new()
    _hint_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
    _hint_label.text = "正在同步服务端技能分槽..."
    outer_vbox.add_child(_hint_label)

    var scroll: ScrollContainer = ScrollContainer.new()
    scroll.custom_minimum_size = Vector2(340.0, 380.0)
    scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
    outer_vbox.add_child(scroll)

    _content_vbox = VBoxContainer.new()
    _content_vbox.size_flags_horizontal = Control.SIZE_EXPAND_FILL
    _content_vbox.add_theme_constant_override("separation", 6)
    scroll.add_child(_content_vbox)

    _request_loading = RequestLoadingOverlay.new()
    _request_loading.name = "PetSkillLoadingOverlay"
    add_child(_request_loading)
    _build_artifact_bag_pick_dialog()


## 刷新标题文案。
func _refresh_title() -> void:
    if _title_label == null:
        return
    for pet_variant: Variant in GameState.pets:
        if pet_variant is Dictionary and int((pet_variant as Dictionary).get("pet_uid", 0)) == _pet_uid:
            var pet: Dictionary = pet_variant as Dictionary
            _title_label.text = "宠物 %d 技能" % int(pet.get("pet_id", 0))
            return
    _title_label.text = "宠物技能"


## 向服务端请求完整技能详情。
func _request_skill_detail() -> void:
    if _loading_request_seq != 0:
        return
    if not GameState.is_ws_authenticated:
        return
    var request_seq: int = App.request_pet_skill_detail(_pet_uid)
    if request_seq <= 0:
        return
    _loading_request_seq = request_seq
    if _request_loading != null:
        _request_loading.show_waiting("正在获取宠物技能详情")
    call_deferred("_wait_skill_detail_request", request_seq)


## 等待技能详情回包后刷新展示。
func _wait_skill_detail_request(expected_seq: int) -> void:
    while expected_seq != 0 and _loading_request_seq == expected_seq:
        var result: Array = await App.request_finished
        if result.size() < 5:
            continue
        var request_cmd: int = int(result[0])
        var seq: int = int(result[1])
        if request_cmd != CommandIds.PET_SKILL_DETAIL_REQ or seq != expected_seq:
            continue
        break
    if _loading_request_seq != expected_seq:
        return
    _loading_request_seq = 0
    if _request_loading != null:
        _request_loading.hide_overlay()
    call_deferred("_refresh_skill_slots_from_state")


## 延迟一帧后从 GameState 刷新技能展示，确保 pet_controller 已合并回包。
func _refresh_skill_slots_from_state() -> void:
    _render_skill_slots(_selected_pet_snapshot())


## 获取当前宠物的最新快照。
func _selected_pet_snapshot() -> Dictionary:
    for pet_variant: Variant in GameState.pets:
        if pet_variant is Dictionary and int((pet_variant as Dictionary).get("pet_uid", 0)) == _pet_uid:
            return (pet_variant as Dictionary).duplicate(true)
    return {}


## 根据 skill_slots 渲染各分类技能槽。
func _render_skill_slots(pet: Dictionary) -> void:
    if _content_vbox == null:
        return
    for child: Node in _content_vbox.get_children():
        _content_vbox.remove_child(child)
        child.queue_free()

    if pet.is_empty():
        if _hint_label != null:
            _hint_label.text = "未找到宠物数据。"
        return

    var skill_slots_variant: Variant = pet.get("skill_slots", {})
    if skill_slots_variant is not Dictionary:
        if _hint_label != null:
            _hint_label.text = "服务端尚未返回 skill_slots 字段。"
        return

    var skill_slots: Dictionary = skill_slots_variant as Dictionary
    if _hint_label != null:
        _hint_label.text = "空法宝槽可点「装备」从背包选择法宝；已镶嵌可「卸下」。"

    _append_slot_section("天生技", skill_slots.get("innate", []))
    _append_single_slot_section("主动神符技", skill_slots.get("active_talisman", {}))
    _append_single_slot_section("神符·英雄", skill_slots.get("talisman_hero", {}))
    _append_single_slot_section("神符【1】", skill_slots.get("talisman_1", {}))
    _append_single_slot_section("神符【2】", skill_slots.get("talisman_2", {}))
    _append_single_slot_section("神符【3】", skill_slots.get("talisman_3", {}))
    _append_slot_section("普通技", skill_slots.get("normal", []))
    _append_artifact_section(skill_slots.get("artifact", []))


## 渲染一组多槽技能区。
func _append_slot_section(title: String, entries_variant: Variant) -> void:
    var section_title: Label = Label.new()
    section_title.text = title
    section_title.add_theme_font_size_override("font_size", 16)
    _content_vbox.add_child(section_title)

    if entries_variant is not Array or (entries_variant as Array).is_empty():
        var empty_label: Label = Label.new()
        empty_label.text = "（空）"
        _content_vbox.add_child(empty_label)
        return

    for entry_variant: Variant in entries_variant as Array:
        if entry_variant is not Dictionary:
            continue
        _content_vbox.add_child(_build_slot_row(entry_variant as Dictionary, false))


## 渲染单个技能槽区。
func _append_single_slot_section(title: String, entry_variant: Variant) -> void:
    var section_title: Label = Label.new()
    section_title.text = title
    section_title.add_theme_font_size_override("font_size", 16)
    _content_vbox.add_child(section_title)

    if entry_variant is not Dictionary:
        var empty_label: Label = Label.new()
        empty_label.text = "（空）"
        _content_vbox.add_child(empty_label)
        return

    _content_vbox.add_child(_build_slot_row(entry_variant as Dictionary, false))


## 渲染法宝槽区，支持卸下操作。
func _append_artifact_section(entries_variant: Variant) -> void:
    var section_title: Label = Label.new()
    section_title.text = "法宝技"
    section_title.add_theme_font_size_override("font_size", 16)
    _content_vbox.add_child(section_title)

    if entries_variant is not Array or (entries_variant as Array).is_empty():
        var empty_label: Label = Label.new()
        empty_label.text = "（空）"
        _content_vbox.add_child(empty_label)
        return

    for entry_variant: Variant in entries_variant as Array:
        if entry_variant is not Dictionary:
            continue
        _content_vbox.add_child(_build_slot_row(entry_variant as Dictionary, true))


## 构建从背包选择法宝的对话框。
func _build_artifact_bag_pick_dialog() -> void:
    _artifact_bag_pick_dialog = AcceptDialog.new()
    _artifact_bag_pick_dialog.title = "选择背包法宝"
    _artifact_bag_pick_dialog.ok_button_text = "确认装备"
    _artifact_bag_pick_dialog.confirmed.connect(_on_artifact_bag_pick_confirmed)
    add_child(_artifact_bag_pick_dialog)

    var dialog_vbox: VBoxContainer = VBoxContainer.new()
    _artifact_bag_pick_dialog.add_child(dialog_vbox)

    var hint_label: Label = Label.new()
    hint_label.text = "请选择要镶嵌的法宝物品："
    dialog_vbox.add_child(hint_label)

    _artifact_bag_option = OptionButton.new()
    _artifact_bag_option.custom_minimum_size = Vector2(260.0, 32.0)
    dialog_vbox.add_child(_artifact_bag_option)


## 构建单个技能槽行 UI。
func _build_slot_row(entry: Dictionary, is_artifact_slot: bool) -> Control:
    var row: HBoxContainer = HBoxContainer.new()
    row.add_theme_constant_override("separation", 8)

    var slot_index: int = int(entry.get("slot_index", 0))
    var skill_id: int = int(entry.get("skill_id", 0))
    var enabled: bool = bool(entry.get("enabled", true))

    var info_label: Label = Label.new()
    info_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
    info_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
    if skill_id <= 0:
        info_label.text = "槽位 %d：未开启" % slot_index if not enabled else "槽位 %d：空" % slot_index
    else:
        info_label.text = "槽位 %d：技能ID %s%s" % [
            slot_index,
            UiFormat.value_to_text(skill_id),
            "" if enabled else "（未开启）",
        ]
    row.add_child(info_label)

    if is_artifact_slot and skill_id <= 0 and enabled:
        var equip_button: Button = Button.new()
        equip_button.text = "装备"
        equip_button.pressed.connect(_on_equip_artifact_pressed.bind(slot_index))
        row.add_child(equip_button)

    if is_artifact_slot and skill_id > 0:
        var unequip_button: Button = Button.new()
        unequip_button.text = "卸下"
        unequip_button.pressed.connect(_on_unequip_pressed.bind(slot_index))
        row.add_child(unequip_button)

    return row


## 点击空法宝槽的装备按钮，从背包选择法宝物品。
func _on_equip_artifact_pressed(artifact_slot_index: int) -> void:
    if _loading_request_seq != 0 or _pet_uid <= 0:
        return
    _pending_equip_artifact_slot_index = artifact_slot_index
    _rebuild_artifact_bag_options()
    if _artifact_bag_option == null or _artifact_bag_option.item_count == 0:
        if _hint_label != null:
            _hint_label.text = "背包中没有可镶嵌的法宝物品（effect_type=pet_artifact）。"
        return
    if _artifact_bag_pick_dialog != null:
        _artifact_bag_pick_dialog.popup_centered(Vector2i(320, 160))


## 填充背包中可用的法宝物品列表。
func _rebuild_artifact_bag_options() -> void:
    if _artifact_bag_option == null:
        return
    _artifact_bag_option.clear()
    var items_variant: Variant = GameState.bag_container.get("items", GameState.bag_items)
    if items_variant is not Array:
        return
    for item_variant: Variant in items_variant as Array:
        if item_variant is not Dictionary:
            continue
        var item: Dictionary = item_variant as Dictionary
        if str(item.get("effect_type", "")) != "pet_artifact":
            continue
        var bag_slot_index: int = int(item.get("slot_index", 0))
        if bag_slot_index <= 0:
            continue
        var item_name: String = str(item.get("item_name", "法宝 %d" % int(item.get("item_id", 0))))
        var quantity: int = int(item.get("quantity", item.get("count", 0)))
        _artifact_bag_option.add_item("%s x%d（槽位%d）" % [item_name, quantity, bag_slot_index], bag_slot_index)


## 确认从背包选择法宝后发起装备请求。
func _on_artifact_bag_pick_confirmed() -> void:
    if _artifact_bag_option == null or _pending_equip_artifact_slot_index < 0:
        return
    var selected_index: int = _artifact_bag_option.get_selected()
    if selected_index < 0:
        return
    var bag_slot_index: int = int(_artifact_bag_option.get_item_id(selected_index))
    _request_artifact_equip(_pending_equip_artifact_slot_index, bag_slot_index)
    _pending_equip_artifact_slot_index = -1


## 发起法宝装备请求。
func _request_artifact_equip(artifact_slot_index: int, bag_slot_index: int) -> void:
    if _loading_request_seq != 0 or _pet_uid <= 0:
        return
    var request_seq: int = App.request_pet_artifact_equip(_pet_uid, artifact_slot_index, bag_slot_index)
    if request_seq <= 0:
        return
    _loading_request_seq = request_seq
    if _request_loading != null:
        _request_loading.show_waiting("正在装备法宝")
    call_deferred("_wait_artifact_request", request_seq, CommandIds.PET_ARTIFACT_EQUIP_REQ)


## 关闭按钮回调。
func _on_close_pressed() -> void:
    hide_panel()


## 发起法宝卸下请求。
func _on_unequip_pressed(slot_index: int) -> void:
    if _loading_request_seq != 0 or _pet_uid <= 0:
        return
    var request_seq: int = App.request_pet_artifact_unequip(_pet_uid, slot_index)
    if request_seq <= 0:
        return
    _loading_request_seq = request_seq
    if _request_loading != null:
        _request_loading.show_waiting("正在卸下法宝")
    call_deferred("_wait_artifact_request", request_seq, CommandIds.PET_ARTIFACT_UNEQUIP_REQ)


## 等待法宝相关回包后刷新展示。
func _wait_artifact_request(expected_seq: int, expected_cmd: int) -> void:
    while expected_seq != 0 and _loading_request_seq == expected_seq:
        var result: Array = await App.request_finished
        if result.size() < 5:
            continue
        var request_cmd: int = int(result[0])
        var seq: int = int(result[1])
        if request_cmd != expected_cmd or seq != expected_seq:
            continue
        break
    if _loading_request_seq != expected_seq:
        return
    _loading_request_seq = 0
    if _request_loading != null:
        _request_loading.hide_overlay()
    call_deferred("_refresh_skill_slots_from_state")
