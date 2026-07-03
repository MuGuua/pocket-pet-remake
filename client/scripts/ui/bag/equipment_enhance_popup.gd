class_name EquipmentEnhancePopup
extends "res://scripts/ui/common/modal_popup_layer.gd"

## 装备强化弹窗场景路径。
const SCENE_PATH: String = "res://scenes/ui/bag/equipment_enhance_popup.tscn"
## 通用物品格子选择面板场景。
const ITEM_SLOT_PICKER_SCENE: PackedScene = preload("res://scenes/ui/common/item_slot_picker.tscn")
## 预览区属性行高亮色。
const PREVIEW_NEXT_COLOR: String = "#63c64d"
## 成功率文案高亮色。
const SUCCESS_RATE_COLOR: String = "#63c64d"
## 材料数量不足时的文案色。
const MATERIAL_SHORT_COLOR: String = "#ff6b6b"
## 预览箭头纹理资源路径。
const PREVIEW_ARROW_TEXTURE_PATH: String = "res://asset/分类/ui/战斗箭头.png"
## 预览箭头图集区域。
const PREVIEW_ARROW_REGION: Rect2 = Rect2(15.0, 0.0, 15.0, 11.0)
## 强化进度条动画时长（秒）。
const ENHANCE_PROGRESS_DURATION_SEC: float = 3.0
## 强化成功结果文案色。
const ENHANCE_RESULT_SUCCESS_COLOR: String = "#63c64d"
## 强化失败结果文案色。
const ENHANCE_RESULT_FAILURE_COLOR: String = "#ff6b6b"
## 强化进度条运行中填充色（黄）。
const ENHANCE_PROGRESS_RUNNING_COLOR: String = "#e6b422"
## 强化进度条成功填充色（绿）。
const ENHANCE_PROGRESS_SUCCESS_COLOR: String = "#63c64d"
## 强化进度条失败填充色（红）。
const ENHANCE_PROGRESS_FAILURE_COLOR: String = "#ff6b6b"
## 满级预览右侧占位文案。
const ENHANCE_PREVIEW_MAX_LABEL: String = "max"
## 满级预览右侧占位文案色。
const ENHANCE_PREVIEW_MAX_COLOR: String = "#9494b8"
## 材料按钮悬停名称浮层场景。
const BAG_ITEM_HOVER_NAME_SCENE: PackedScene = preload("res://scenes/ui/common/bag_item_hover_name.tscn")
## 未选中材料时，悬停材料按钮展示的提示文案。
const MATERIAL_BUTTON_HOVER_HINT: String = "点击选择强化材料"
## 材料选择面板相对默认定位向上偏移（像素）。
const MATERIAL_PICKER_OFFSET_Y: int = -30

signal enhance_requested(item: Dictionary, times: int, continuous: bool, cost_item_id: int)
## 强化进度演出结束且结果文案已展示。
signal enhance_presentation_finished

@onready var _item_icon: TextureRect = %ItemIcon
@onready var _item_enhance_badge: Label = %ItemEnhanceBadge
@onready var _item_name_label: Label = %ItemNameLabel
@onready var _item_level_label: Label = %ItemLevelLabel
@onready var _item_slot_label: Label = %ItemSlotLabel
@onready var _preview_rows: VBoxContainer = %PreviewRows
@onready var _success_rate_label: Label = %SuccessRateLabel
@onready var _material_icon: TextureRect = %MaterialIcon
@onready var _material_select_button: Button = %MaterialSelectButton
@onready var _material_count_label: RichTextLabel = %MaterialCountLabel
@onready var _cost_copper_amount_label: Label = %CostGoldAmountLabel
@onready var _wallet_gold_amount_label: Label = %WalletGoldAmountLabel
@onready var _wallet_silver_amount_label: Label = %WalletSilverAmountLabel
@onready var _wallet_copper_amount_label: Label = %WalletCopperAmountLabel
@onready var _enhance_button: RuntimeActionButton = %EnhanceButton
@onready var _enhance_progress_bar: ProgressBar = %EnhanceProgressBar
@onready var _enhance_result_label: Label = %EnhanceResultLabel

## 当前要强化的背包装备快照。
var _item: Dictionary = {}
## 本次计划强化的次数。
var _enhance_times: int = 1
## 当前选中的强化材料 item_id，默认来自服务端 cost_item_id。
var _selected_cost_item_id: int = 0
## 强化材料选择浮层。
var _material_picker: ItemSlotPicker = null
## 材料按钮悬停时展示名称/提示的浮层。
var _material_button_hover_name: BagItemHoverName = null
## 预览行背景样式，与场景编辑器模板保持一致。
var _preview_row_panel_style: StyleBoxFlat = null
## 预览箭头纹理缓存。
var _preview_arrow_texture: Texture2D = null
## 是否正在播放强化进度演出。
var _enhance_presentation_active: bool = false
## 是否禁止关闭弹窗（强化演出期间）。
var _block_dismiss: bool = false
## 强化进度条动画是否已结束。
var _enhance_progress_finished: bool = false
## 强化请求是否已收到回包。
var _enhance_response_ready: bool = false
## 强化请求回包是否成功（协议层）。
var _enhance_response_ok: bool = false
## 强化玩法是否成功（业务层 success 字段）。
var _enhance_response_success: bool = false
## 强化失败时服务端返回的惩罚类型：damage / level_down / none。
var _enhance_failure_penalty: String = "damage"
## 强化进度条 tween。
var _enhance_progress_tween: Tween = null
## 强化进度条 fill 样式（运行时改色）。
var _enhance_progress_fill_style: StyleBoxFlat = null
## 点击强化后乐观扣除的铜币总量，服务端快照到达后清零。
var _optimistic_copper_spent: int = 0


## 初始化按钮信号，并启用模态遮罩的空白区域关闭能力。
func _ready() -> void:
    super._ready()
    _init_preview_row_style()
    _init_preview_arrow_texture()
    _init_enhance_progress_styles()
    _ensure_material_picker()
    _ensure_material_button_hover_name()
    if _enhance_button != null and not _enhance_button.pressed.is_connected(_on_enhance_button_pressed):
        _enhance_button.pressed.connect(_on_enhance_button_pressed)
    if _material_select_button != null and not _material_select_button.pressed.is_connected(_on_material_select_button_pressed):
        _material_select_button.pressed.connect(_on_material_select_button_pressed)
        _material_select_button.mouse_filter = Control.MOUSE_FILTER_STOP
        if not _material_select_button.mouse_entered.is_connected(_on_material_select_button_mouse_entered):
            _material_select_button.mouse_entered.connect(_on_material_select_button_mouse_entered)
        if not _material_select_button.mouse_exited.is_connected(_on_material_select_button_mouse_exited):
            _material_select_button.mouse_exited.connect(_on_material_select_button_mouse_exited)
    _reset_enhance_presentation_ui()


## 打开强化弹窗并刷新展示。
func show_equipment(item: Dictionary) -> void:
    _cancel_enhance_presentation(false)
    _reset_enhance_presentation_ui()
    _item = item.duplicate(true)
    _enhance_times = 1
    _selected_cost_item_id = int(_resolve_enhance_preview().get("cost_item_id", 0))
    _refresh_all()
    _set_content_mouse_ignore(true)
    _open_modal()


## 外部在背包/钱包刷新后同步弹窗内容；演出进行中也会刷新消耗与预览，但不重置进度条。
func refresh_current_item(item: Dictionary) -> void:
    if not visible or item.is_empty():
        return
    var was_presentation_active: bool = _enhance_presentation_active
    var previous_selected: int = _selected_cost_item_id
    _item = item.duplicate(true)
    _optimistic_copper_spent = 0
    var preview: Dictionary = _resolve_enhance_preview()
    if previous_selected > 0 and _find_material_option(previous_selected) != null:
        _selected_cost_item_id = previous_selected
    else:
        _selected_cost_item_id = int(preview.get("cost_item_id", 0))
    _apply_selected_material_to_preview()
    if was_presentation_active:
        # 演出进行中只更新快照；预览数值与结果文案在进度结束后一并刷新。
        return
    _refresh_all()


## 读取当前选中的强化材料 item_id，供强化请求使用。
func get_selected_cost_item_id() -> int:
    return _selected_cost_item_id


## 强化请求回包到达后写入结果；进度条与回包都就绪时再同步刷新数值并展示文案。
func notify_enhance_response(ok: bool, success: bool, failure_penalty: String = "damage", payload: Dictionary = {}) -> void:
    if not _enhance_presentation_active:
        return
    if not ok:
        _optimistic_copper_spent = 0
    else:
        _apply_enhance_response_payload(payload, success, failure_penalty)
    _enhance_response_ready = true
    _enhance_response_ok = ok
    _enhance_response_success = success
    _enhance_failure_penalty = failure_penalty.strip_edges()
    if _enhance_failure_penalty.is_empty():
        _enhance_failure_penalty = "damage"
    _try_finish_enhance_presentation()


## 强化演出是否进行中。
func is_enhance_presentation_active() -> bool:
    return _enhance_presentation_active


## 刷新弹窗全部文案与预览行。
func _refresh_all() -> void:
    _refresh_item_header()
    _refresh_preview_rows()
    _refresh_success_rate()
    _refresh_material_and_cost()
    _refresh_enhance_button_state()
    _set_content_mouse_ignore(true)


## 刷新顶部装备信息区。
func _refresh_item_header() -> void:
    if _item_name_label != null:
        _item_name_label.text = BagUiMapper.item_name(_item)
    if _item_icon != null:
        _item_icon.texture = BagUiMapper.icon_texture(_item)
    if _item_enhance_badge != null:
        var enhance_level: int = BagUiMapper.enhance_level(_item)
        _item_enhance_badge.text = "+%s" % UiFormat.value_to_text(enhance_level)
        _item_enhance_badge.visible = true
    var level_text: String = BagUiMapper.required_level_text(_item)
    if _item_level_label != null:
        if level_text.is_empty():
            _item_level_label.text = "等级：1"
        else:
            _item_level_label.text = level_text.replace("等级：", "等级: ")
    if _item_slot_label != null:
        var slot_text: String = BagUiMapper.equip_slot_text(_item)
        if slot_text.is_empty():
            _item_slot_label.text = "部位: -"
        else:
            _item_slot_label.text = slot_text.replace("部位：", "部位: ")


## 根据服务端 enhance_preview.rows 重建预览表格；满级时右侧列显示 max。
func _refresh_preview_rows() -> void:
    if _preview_rows == null:
        return
    for child: Node in _preview_rows.get_children():
        child.queue_free()
    var preview_variant: Variant = _item.get("enhance_preview", {})
    if preview_variant is not Dictionary:
        return
    var preview: Dictionary = preview_variant as Dictionary
    var rows_variant: Variant = preview.get("rows", [])
    if rows_variant is not Array:
        return
    for row_variant: Variant in rows_variant as Array:
        if row_variant is not Dictionary:
            continue
        _preview_rows.add_child(_build_preview_row(row_variant as Dictionary))


## 构建一行带 Margin 与 Panel 的预览行，样式对齐场景模板。
func _build_preview_row(row: Dictionary) -> MarginContainer:
    var row_margin: MarginContainer = MarginContainer.new()
    row_margin.add_theme_constant_override("margin_left", 3)
    row_margin.add_theme_constant_override("margin_right", 3)
    var row_panel: PanelContainer = PanelContainer.new()
    row_panel.custom_minimum_size = Vector2(0, 10)
    row_panel.custom_maximum_size = Vector2(-1, 11)
    if _preview_row_panel_style != null:
        row_panel.add_theme_stylebox_override("panel", _preview_row_panel_style)
    var row_hbox: HBoxContainer = HBoxContainer.new()
    var stat_label: Label = Label.new()
    stat_label.custom_minimum_size = Vector2(30, 0)
    stat_label.text = str(row.get("label", ""))
    stat_label.add_theme_font_size_override("font_size", 6)
    stat_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
    row_hbox.add_child(stat_label)
    var current_label: Label = Label.new()
    current_label.custom_minimum_size = Vector2(40, 0)
    current_label.text = str(row.get("current", "+0"))
    current_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
    current_label.add_theme_font_size_override("font_size", 6)
    current_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
    row_hbox.add_child(current_label)
    var arrow_icon: TextureRect = TextureRect.new()
    arrow_icon.custom_minimum_size = Vector2(12, 0)
    arrow_icon.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
    arrow_icon.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
    arrow_icon.texture = _preview_arrow_texture
    row_hbox.add_child(arrow_icon)
    var next_text: String = _resolve_preview_row_next_text(row)
    var next_label: Label = Label.new()
    next_label.custom_minimum_size = Vector2(74, 0)
    next_label.text = next_text
    next_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
    next_label.add_theme_font_size_override("font_size", 6)
    if _is_enhance_preview_at_max_level():
        next_label.add_theme_color_override("font_color", Color(ENHANCE_PREVIEW_MAX_COLOR))
    else:
        next_label.add_theme_color_override("font_color", Color(PREVIEW_NEXT_COLOR))
    next_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
    row_hbox.add_child(next_label)
    row_panel.add_child(row_hbox)
    row_margin.add_child(row_panel)
    return row_margin


## 解析预览行右侧文案；满级时统一显示 max。
func _resolve_preview_row_next_text(row: Dictionary) -> String:
    if _is_enhance_preview_at_max_level():
        return ENHANCE_PREVIEW_MAX_LABEL
    var next_min: String = str(row.get("next_min", "+0"))
    var next_max: String = str(row.get("next_max", next_min))
    if next_max != next_min and not next_max.is_empty():
        return "%s ~ %s" % [next_min, next_max]
    return next_min


## 当前装备是否已达模板最高强化等级。
func _is_enhance_preview_at_max_level() -> bool:
    var preview: Dictionary = _resolve_enhance_preview()
    var max_level: int = int(preview.get("max_enhance_level", 0))
    if max_level <= 0:
        return false
    return BagUiMapper.enhance_level(_item) >= max_level


## 刷新成功率文案；满级时不展示成功率。
func _refresh_success_rate() -> void:
    if _success_rate_label == null:
        return
    if _is_enhance_preview_at_max_level():
        _success_rate_label.text = "已达最高强化等级"
        _success_rate_label.add_theme_color_override("font_color", Color(SUCCESS_RATE_COLOR))
        return
    var preview: Dictionary = _resolve_enhance_preview()
    var success_rate: int = int(preview.get("success_rate_pct", 0))
    # var band_label: String = str(preview.get("required_level_band_label", "")).strip_edges()
    _success_rate_label.text = "强化成功率: %s%%" % UiFormat.value_to_text(success_rate)
    # if band_label.is_empty():
    #   _success_rate_label.text = "强化成功率: %s%%" % UiFormat.value_to_text(success_rate)
    # else:
    #   _success_rate_label.text = "强化成功率(%s): %s%%" % [band_label, UiFormat.value_to_text(success_rate)]
    _success_rate_label.add_theme_color_override("font_color", Color(SUCCESS_RATE_COLOR))


## 刷新材料与铜币消耗展示。
func _refresh_material_and_cost() -> void:
    var preview: Dictionary = _resolve_enhance_preview()
    var cost_item_id: int = _selected_cost_item_id
    if cost_item_id <= 0:
        cost_item_id = int(preview.get("cost_item_id", 0))
    var owned_count: int = int(preview.get("owned_cost_quantity", 0))
    var need_count: int = int(preview.get("cost_quantity", 0)) * _enhance_times
    if _material_icon != null:
        if cost_item_id <= 0:
            _material_icon.texture = null
            _material_icon.hide()
        else:
            _material_icon.texture = ItemIcons.resolve_texture(cost_item_id)
            _material_icon.show()
    if _material_select_button != null:
        _material_select_button.disabled = _enhance_presentation_active
        _material_select_button.tooltip_text = ""
    if _material_count_label != null:
        if cost_item_id <= 0:
            _material_count_label.text = ""
        else:
            var count_color: String = PREVIEW_NEXT_COLOR if owned_count >= need_count and need_count > 0 else MATERIAL_SHORT_COLOR
            _material_count_label.text = "[color=%s]%s[/color]/%s" % [
                count_color,
                UiFormat.value_to_text(owned_count),
                UiFormat.value_to_text(maxi(need_count, int(preview.get("cost_quantity", 0)))),
            ]
    var cost_copper: int = int(preview.get("cost_gold_copper", 0)) * _enhance_times
    var wallet_components: Dictionary = _resolve_display_wallet_components()
    var wallet_total_copper: int = int(wallet_components.get("total_copper", 0))
    var cost_color: String = PREVIEW_NEXT_COLOR if cost_copper <= 0 or wallet_total_copper >= cost_copper else MATERIAL_SHORT_COLOR
    if _cost_copper_amount_label != null:
        _cost_copper_amount_label.text = UiFormat.value_to_text(cost_copper)
        _cost_copper_amount_label.add_theme_color_override("font_color", Color(cost_color))
    if _wallet_gold_amount_label != null:
        _wallet_gold_amount_label.text = UiFormat.value_to_text(wallet_components.get("gold", 0))
    if _wallet_silver_amount_label != null:
        _wallet_silver_amount_label.text = UiFormat.value_to_text(wallet_components.get("silver", 0))
    if _wallet_copper_amount_label != null:
        _wallet_copper_amount_label.text = UiFormat.value_to_text(wallet_components.get("copper", 0))


## 根据材料与铜币是否足够决定强化按钮可用态。
func _refresh_enhance_button_state() -> void:
    if _enhance_button == null:
        return
    if _enhance_presentation_active:
        _enhance_button.disabled = true
        return
    if _is_enhance_preview_at_max_level():
        _enhance_button.disabled = true
        return
    var preview: Dictionary = _resolve_enhance_preview()
    var can_enhance: bool = bool(preview.get("can_enhance", false))
    var owned_count: int = int(preview.get("owned_cost_quantity", 0))
    var need_count: int = int(preview.get("cost_quantity", 0)) * _enhance_times
    var cost_copper: int = int(preview.get("cost_gold_copper", 0)) * _enhance_times
    var wallet_total_copper: int = int(GameState.wallet_snapshot.get("total_copper", 0))
    var wallet_enough: bool = cost_copper <= 0 or wallet_total_copper >= cost_copper
    _enhance_button.disabled = not can_enhance or need_count <= 0 or owned_count < need_count or not wallet_enough


## 解析当前装备的服务端强化预览块。
func _resolve_enhance_preview() -> Dictionary:
    var preview_variant: Variant = _item.get("enhance_preview", {})
    if preview_variant is Dictionary:
        return preview_variant as Dictionary
    return {}


## 懒创建材料按钮悬停名称浮层。
func _ensure_material_button_hover_name() -> void:
    if _material_button_hover_name != null:
        return
    _material_button_hover_name = BAG_ITEM_HOVER_NAME_SCENE.instantiate() as BagItemHoverName
    if _material_button_hover_name == null:
        return
    add_child(_material_button_hover_name)


## 鼠标移入材料按钮时，在右上方展示与背包格子一致的悬停名称样式。
func _on_material_select_button_mouse_entered() -> void:
    if _material_select_button == null or _material_button_hover_name == null:
        return
    if _material_select_button.disabled or _enhance_presentation_active:
        return
    var hover_text: String = _resolve_material_button_hover_text()
    if hover_text.is_empty():
        return
    _material_button_hover_name.show_for_anchor(_material_select_button, hover_text)


## 鼠标移出材料按钮时隐藏悬停名称。
func _on_material_select_button_mouse_exited() -> void:
    _hide_material_button_hover_name()


## 解析材料按钮悬停文案：优先展示当前材料名，否则展示操作提示。
func _resolve_material_button_hover_text() -> String:
    var preview: Dictionary = _resolve_enhance_preview()
    var cost_item_id: int = _selected_cost_item_id
    if cost_item_id <= 0:
        cost_item_id = int(preview.get("cost_item_id", 0))
    if cost_item_id > 0:
        var material: Dictionary = _find_material_option(cost_item_id)
        if not material.is_empty():
            return str(material.get("item_name", ""))
        var preview_name: String = str(preview.get("cost_item_name", ""))
        if not preview_name.is_empty():
            return preview_name
    return MATERIAL_BUTTON_HOVER_HINT


## 关闭材料按钮悬停名称浮层。
func _hide_material_button_hover_name() -> void:
    if _material_button_hover_name == null:
        return
    _material_button_hover_name.hide_name()


## 懒创建材料选择浮层并绑定选中事件。
func _ensure_material_picker() -> void:
    if _material_picker != null:
        return
    _material_picker = ITEM_SLOT_PICKER_SCENE.instantiate() as ItemSlotPicker
    if _material_picker == null:
        return
    add_child(_material_picker)
    if not _material_picker.item_selected.is_connected(_on_material_picker_selected):
        _material_picker.item_selected.connect(_on_material_picker_selected)
    if not _material_picker.picker_closed.is_connected(_on_material_picker_closed):
        _material_picker.picker_closed.connect(_on_material_picker_closed)


## 初始化预览行 Panel 背景样式。
func _init_preview_row_style() -> void:
    _preview_row_panel_style = StyleBoxFlat.new()
    _preview_row_panel_style.bg_color = Color(0.6, 0.6, 0.6, 0.078431375)


## 初始化预览箭头图集纹理。
func _init_preview_arrow_texture() -> void:
    var atlas_texture: Texture2D = load(PREVIEW_ARROW_TEXTURE_PATH) as Texture2D
    if atlas_texture == null:
        return
    var arrow_atlas: AtlasTexture = AtlasTexture.new()
    arrow_atlas.atlas = atlas_texture
    arrow_atlas.region = PREVIEW_ARROW_REGION
    _preview_arrow_texture = arrow_atlas


## 在材料列表中查找指定 item_id 的条目。
func _find_material_option(item_id: int) -> Dictionary:
    var preview: Dictionary = _resolve_enhance_preview()
    var materials: Array = _resolve_picker_materials(preview)
    for material_variant: Variant in materials:
        if material_variant is not Dictionary:
            continue
        var material: Dictionary = material_variant as Dictionary
        if int(material.get("item_id", 0)) == item_id:
            return material
    return {}


## 解析材料选择面板可选列表；preview 缺失时回退到背包内强化材料。
func _resolve_picker_materials(preview: Dictionary) -> Array:
    var materials_variant: Variant = preview.get("materials", [])
    if materials_variant is Array:
        var preview_materials: Array = materials_variant as Array
        if not preview_materials.is_empty():
            return preview_materials
    return _collect_enhance_material_options_from_bag()


## 从当前背包快照收集强化材料子类物品，供 preview 缺失 materials 时兜底。
func _collect_enhance_material_options_from_bag() -> Array:
    var results: Array = []
    var seen_item_ids: Dictionary = {}
    for item_variant: Variant in GameState.bag_items:
        if item_variant is not Dictionary:
            continue
        var bag_item: Dictionary = item_variant as Dictionary
        if str(bag_item.get("item_sub_type", "")) != "equipment_enhance":
            continue
        var item_id: int = BagUiMapper.item_id(bag_item)
        if item_id <= 0 or seen_item_ids.has(item_id):
            continue
        var owned_quantity: int = BagUiMapper.quantity(bag_item)
        if owned_quantity <= 0:
            continue
        seen_item_ids[item_id] = true
        results.append({
            "item_id": item_id,
            "item_name": BagUiMapper.item_name(bag_item),
            "owned_quantity": owned_quantity,
            "quantity": owned_quantity,
            "is_stackable": BagUiMapper.is_stackable(bag_item),
        })
    return results


## 将当前选中材料写回本地 preview 快照，供 UI 刷新使用。
func _apply_selected_material_to_preview() -> void:
    if _selected_cost_item_id <= 0:
        return
    var preview_variant: Variant = _item.get("enhance_preview", null)
    if preview_variant is not Dictionary:
        return
    var preview: Dictionary = preview_variant as Dictionary
    var material: Dictionary = _find_material_option(_selected_cost_item_id)
    if material.is_empty():
        return
    preview["cost_item_id"] = _selected_cost_item_id
    preview["cost_item_name"] = str(material.get("item_name", preview.get("cost_item_name", "")))
    preview["owned_cost_quantity"] = int(material.get("owned_quantity", 0))
    var effective_rate: int = int(material.get("effective_success_rate_pct", preview.get("success_rate_pct", 0)))
    if effective_rate > 0:
        preview["success_rate_pct"] = effective_rate
    _item["enhance_preview"] = preview


## 点击材料选择按钮：已展开则收起，否则在按钮附近打开材料面板。
func _on_material_select_button_pressed() -> void:
    _ensure_material_picker()
    if _material_picker == null or _material_select_button == null:
        return
    _hide_material_button_hover_name()
    if _material_picker.is_open():
        _hide_material_picker()
        return
    var preview: Dictionary = _resolve_enhance_preview()
    var materials: Array = _resolve_picker_materials(preview)
    if materials.is_empty():
        App.notice_received.emit("背包中没有可用的强化材料。")
        return
    var selected_id: int = _selected_cost_item_id
    if selected_id <= 0:
        selected_id = int(preview.get("cost_item_id", 0))
    _material_picker.open_picker_near(
        _material_select_button,
        materials,
        selected_id,
        {
            "title": "选择强化材料",
            "placement": ItemSlotPicker.PLACEMENT_RIGHT,
            "position_offset_y": MATERIAL_PICKER_OFFSET_Y,
        }
    )


## 关闭材料选择浮层（若当前已展开）。
func _hide_material_picker() -> void:
    if _material_picker == null:
        return
    if _material_picker.is_open():
        _material_picker.hide_picker()
    _hide_material_button_hover_name()


## 材料面板展开时，点击其外部区域只吞掉输入，不再自动关闭。
func _try_close_material_picker_for_event(event: InputEvent) -> bool:
    if _material_picker == null or not _material_picker.is_open():
        return false
    if not _is_dismiss_event(event):
        return false
    var global_pos: Vector2 = _event_global_position(event)
    if global_pos.x < 0.0:
        return false
    if _material_picker.is_global_point_over_panel(global_pos):
        return false
    if _material_select_button != null and _material_select_button.get_global_rect().has_point(global_pos):
        return false
    get_viewport().set_input_as_handled()
    return true


## 处理材料面板选中结果并刷新消耗展示。
func _on_material_picker_selected(item: Dictionary) -> void:
    _selected_cost_item_id = BagUiMapper.item_id(item)
    _apply_selected_material_to_preview()
    _refresh_success_rate()
    _refresh_material_and_cost()
    _refresh_enhance_button_state()


func _on_material_picker_closed() -> void:
    pass


## 关闭弹窗时同步收起材料面板与悬停提示，避免 top_level 浮层残留。
func _close_modal() -> void:
    _cancel_enhance_presentation(true)
    _hide_material_picker()
    super._close_modal()


func _force_close_modal() -> void:
    _cancel_enhance_presentation(true)
    _hide_material_picker()
    super._force_close_modal()


## 拦截模态输入：面板外点击只吞掉事件，不再自动关闭弹窗。
func _input(event: InputEvent) -> void:
    _handle_modal_dismiss_input(event)


func _shortcut_input(event: InputEvent) -> void:
    _handle_modal_dismiss_input(event)


func _unhandled_input(event: InputEvent) -> void:
    _handle_modal_dismiss_input(event)


func _handle_modal_dismiss_input(event: InputEvent) -> void:
    if not visible or not _is_topmost_runtime_modal():
        return
    if _block_dismiss:
        get_viewport().set_input_as_handled()
        return
    if _try_close_material_picker_for_event(event):
        return
    if not _is_dismiss_event(event):
        return
    if _should_keep_modal_open_for_event(event):
        return
    get_viewport().set_input_as_handled()


## 演出锁定期间禁止通过右上角关闭按钮提前关闭，避免中断强化表现与回包同步。
func _on_top_close_button_pressed() -> void:
    if _block_dismiss:
        get_viewport().set_input_as_handled()
        return
    super._on_top_close_button_pressed()


## 判断当前输入是否落在弹窗可交互区域内；落在区域内时不应关闭弹窗。
func _should_keep_modal_open_for_event(event: InputEvent) -> bool:
    var global_pos: Vector2 = _event_global_position(event)
    if global_pos.x < 0.0:
        return false
    if _material_picker != null and _material_picker.is_open():
        if _material_picker.is_global_point_over_panel(global_pos):
            return true
    var panel: Control = get_node_or_null("CenterContainer/PopupPanel") as Control
    if panel == null:
        return false
    return panel.get_global_rect().has_point(global_pos)


## 提取鼠标/触摸事件的全局坐标，供命中区域判断使用。
func _event_global_position(event: InputEvent) -> Vector2:
    if event is InputEventMouseButton:
        var mouse_event: InputEventMouseButton = event as InputEventMouseButton
        return mouse_event.global_position
    if event is InputEventScreenTouch:
        var touch_event: InputEventScreenTouch = event as InputEventScreenTouch
        return touch_event.position
    return Vector2(-1.0, -1.0)


func _set_content_mouse_ignore(ignore: bool) -> void:
    var center: Control = get_node_or_null("CenterContainer") as Control
    if center != null:
        center.mouse_filter = Control.MOUSE_FILTER_IGNORE if ignore else Control.MOUSE_FILTER_STOP
    var panel: Control = get_node_or_null("CenterContainer/PopupPanel") as Control
    if panel == null:
        return
    _apply_panel_mouse_filters(panel, ignore)


## 递归设置面板内控件鼠标过滤；交互控件必须保持 STOP，避免点击穿透到遮罩。
func _apply_panel_mouse_filters(node: Node, ignore: bool) -> void:
    if node is Control:
        var control: Control = node as Control
        if control == _material_select_button or control == _enhance_button:
            control.mouse_filter = Control.MOUSE_FILTER_STOP
        else:
            control.mouse_filter = Control.MOUSE_FILTER_IGNORE if ignore else Control.MOUSE_FILTER_STOP
    for child: Node in node.get_children():
        _apply_panel_mouse_filters(child, ignore)


func _on_enhance_button_pressed() -> void:
    if _item.is_empty() or _enhance_button == null or _enhance_button.disabled:
        return
    if _enhance_presentation_active:
        return
    if _is_enhance_preview_at_max_level():
        return
    var cost_item_id: int = _selected_cost_item_id
    if cost_item_id <= 0:
        cost_item_id = int(_resolve_enhance_preview().get("cost_item_id", 0))
    _begin_enhance_presentation()
    enhance_requested.emit(_item.duplicate(true), _enhance_times, false, cost_item_id)


## 点击强化后锁定弹窗交互，并启动 3 秒进度条动画。
func _begin_enhance_presentation() -> void:
    _enhance_presentation_active = true
    _block_dismiss = true
    _enhance_progress_finished = false
    _enhance_response_ready = false
    _enhance_response_ok = false
    _enhance_response_success = false
    _enhance_failure_penalty = "damage"
    if _material_picker != null and _material_picker.is_open():
        _hide_material_picker()
    _reset_enhance_result_label()
    _apply_optimistic_enhance_cost_deduction()
    _refresh_material_and_cost()
    _set_panel_interactive(false)
    _play_enhance_progress_animation()


## 点击强化瞬间乐观扣除材料与铜币展示，与服务端「先扣费再掷骰」一致。
func _apply_optimistic_enhance_cost_deduction() -> void:
    var preview: Dictionary = _resolve_enhance_preview()
    if preview.is_empty():
        return
    var cost_item_id: int = _selected_cost_item_id
    if cost_item_id <= 0:
        cost_item_id = int(preview.get("cost_item_id", 0))
    var cost_quantity: int = int(preview.get("cost_quantity", 0)) * _enhance_times
    var cost_copper: int = int(preview.get("cost_gold_copper", 0)) * _enhance_times
    if cost_quantity > 0:
        var owned_count: int = int(preview.get("owned_cost_quantity", 0))
        preview["owned_cost_quantity"] = maxi(0, owned_count - cost_quantity)
        var materials_variant: Variant = preview.get("materials", [])
        if materials_variant is Array:
            var materials: Array = materials_variant as Array
            for index: int in range(materials.size()):
                var material_variant: Variant = materials[index]
                if material_variant is not Dictionary:
                    continue
                var material: Dictionary = material_variant as Dictionary
                if int(material.get("item_id", 0)) != cost_item_id:
                    continue
                material["owned_quantity"] = maxi(0, int(material.get("owned_quantity", 0)) - cost_quantity)
                materials[index] = material
                break
            preview["materials"] = materials
    _item["enhance_preview"] = preview
    _optimistic_copper_spent = maxi(0, cost_copper)


## 计算弹窗展示用钱包分量；乐观扣费期间在 total_copper 上先行扣除。
func _resolve_display_wallet_components() -> Dictionary:
    var wallet: Dictionary = GameState.wallet_snapshot
    var total_copper: int = int(wallet.get("total_copper", 0)) - _optimistic_copper_spent
    return _wallet_components_from_total_copper(total_copper)


## 将总铜币真值拆成金/银/铜展示分量，规则与服务端 wallet 一致。
func _wallet_components_from_total_copper(total_copper: int) -> Dictionary:
    var normalized_total: int = maxi(0, total_copper)
    return {
        "gold": normalized_total / 1000000,
        "silver": (normalized_total % 1000000) / 1000,
        "copper": normalized_total % 1000,
        "total_copper": normalized_total,
    }


## 进度条与回包都就绪后，同步刷新预览数值与结果文案。
func _try_finish_enhance_presentation() -> void:
    if not _enhance_presentation_active:
        return
    if not _enhance_progress_finished or not _enhance_response_ready:
        return
    _refresh_item_header()
    _refresh_preview_rows()
    _refresh_success_rate()
    _refresh_material_and_cost()
    _show_enhance_result_text()
    _enhance_presentation_active = false
    _block_dismiss = false
    _set_panel_interactive(true)
    _refresh_enhance_button_state()
    enhance_presentation_finished.emit()


## 将强化回包中的等级/损坏/属性同步到本地快照，便于与结果文案同时刷新预览区。
func _apply_enhance_response_payload(payload: Dictionary, success: bool, failure_penalty: String) -> void:
    if payload.is_empty():
        return
    var new_level: int = int(payload.get("new_level", BagUiMapper.enhance_level(_item)))
    _item["enhance_level"] = new_level
    var item_variant: Variant = payload.get("item", null)
    if item_variant is Dictionary:
        var item_snap: Dictionary = item_variant as Dictionary
        if item_snap.has("is_damaged"):
            _item["is_damaged"] = bool(item_snap.get("is_damaged", false))
        var bonus_variant: Variant = item_snap.get("bonus", null)
        if bonus_variant is Dictionary:
            _item["bonus"] = bonus_variant
    if not _preview_already_reflects_level(new_level):
        _sync_enhance_preview_after_result(new_level, success, failure_penalty)
    elif failure_penalty == "damage":
        _mark_enhance_preview_damaged()


## 判断预览区强化等级行是否已与服务端新等级一致。
func _preview_already_reflects_level(level: int) -> bool:
    if level < 0:
        return false
    var preview: Dictionary = _resolve_enhance_preview()
    var rows_variant: Variant = preview.get("rows", [])
    if rows_variant is not Array:
        return false
    var expected_text: String = "+%s" % UiFormat.value_to_text(level)
    for row_variant: Variant in rows_variant as Array:
        if row_variant is not Dictionary:
            continue
        var row: Dictionary = row_variant as Dictionary
        if str(row.get("label", "")) != "强化等级":
            continue
        return str(row.get("current", "")) == expected_text
    return false


## 强化失败且装备损坏时，仅关闭预览区的可强化状态。
func _mark_enhance_preview_damaged() -> void:
    var preview_variant: Variant = _item.get("enhance_preview", null)
    if preview_variant is not Dictionary:
        return
    var preview: Dictionary = (preview_variant as Dictionary).duplicate(true)
    preview["can_enhance"] = false
    _item["enhance_preview"] = preview


## 按强化结果推进预览行：成功/降级时左侧数值切到原右侧预览，满级时右侧显示 max。
func _sync_enhance_preview_after_result(new_level: int, success: bool, failure_penalty: String) -> void:
    var preview_variant: Variant = _item.get("enhance_preview", null)
    if preview_variant is not Dictionary:
        return
    var preview: Dictionary = (preview_variant as Dictionary).duplicate(true)
    var max_level: int = int(preview.get("max_enhance_level", 0))
    if failure_penalty == "damage":
        preview["can_enhance"] = false
        _item["is_damaged"] = true
    var rows_variant: Variant = preview.get("rows", [])
    if rows_variant is not Array:
        _item["enhance_preview"] = preview
        return
    var should_promote_current: bool = success or failure_penalty == "level_down"
    var at_max: bool = max_level > 0 and new_level >= max_level
    var synced_rows: Array = []
    for row_variant: Variant in rows_variant as Array:
        if row_variant is not Dictionary:
            continue
        var row: Dictionary = (row_variant as Dictionary).duplicate(true)
        var label: String = str(row.get("label", ""))
        if label == "强化等级":
            row["current"] = "+%s" % UiFormat.value_to_text(new_level)
            if at_max:
                row["next_min"] = ENHANCE_PREVIEW_MAX_LABEL
                row["next_max"] = ENHANCE_PREVIEW_MAX_LABEL
            else:
                var next_level_text: String = "+%s" % UiFormat.value_to_text(new_level + 1)
                row["next_min"] = next_level_text
                row["next_max"] = next_level_text
        elif should_promote_current:
            var promoted_value: String = str(row.get("next_min", row.get("current", "")))
            if not promoted_value.is_empty() and promoted_value != ENHANCE_PREVIEW_MAX_LABEL:
                row["current"] = promoted_value
            if at_max:
                row["next_min"] = ENHANCE_PREVIEW_MAX_LABEL
                row["next_max"] = ENHANCE_PREVIEW_MAX_LABEL
        synced_rows.append(row)
    preview["rows"] = synced_rows
    if at_max:
        preview["can_enhance"] = false
    _item["enhance_preview"] = preview


## 在进度条右侧展示强化成功/失败文案，并将进度条着色为绿/红。
func _show_enhance_result_text() -> void:
    if _enhance_result_label == null:
        return
    _enhance_result_label.visible = true
    if _enhance_progress_bar != null:
        _enhance_progress_bar.value = 100.0
    if not _enhance_response_ok or not _enhance_response_success:
        if not _enhance_response_ok:
            _enhance_result_label.text = "强化请求失败"
        else:
            _enhance_result_label.text = _resolve_enhance_failure_result_text()
        _enhance_result_label.add_theme_color_override("font_color", Color(ENHANCE_RESULT_FAILURE_COLOR))
        _set_enhance_progress_fill_color(Color(ENHANCE_PROGRESS_FAILURE_COLOR))
        return
    _enhance_result_label.text = "强化成功"
    _enhance_result_label.add_theme_color_override("font_color", Color(ENHANCE_RESULT_SUCCESS_COLOR))
    _set_enhance_progress_fill_color(Color(ENHANCE_PROGRESS_SUCCESS_COLOR))


## 根据服务端 failure_penalty 生成强化失败结果文案。
func _resolve_enhance_failure_result_text() -> String:
    match _enhance_failure_penalty:
        "level_down":
            return "强化失败，等级降低"
        "none":
            return "强化失败"
        _:
            return "强化失败，装备已损坏"


## 播放强化进度条线性动画，结束后尝试展示结果。
func _play_enhance_progress_animation() -> void:
    _stop_enhance_progress_tween()
    _set_enhance_progress_fill_color(Color(ENHANCE_PROGRESS_RUNNING_COLOR))
    if _enhance_progress_bar != null:
        _enhance_progress_bar.min_value = 0.0
        _enhance_progress_bar.max_value = 100.0
        _enhance_progress_bar.value = 0.0
        _enhance_progress_bar.visible = true
    if ENHANCE_PROGRESS_DURATION_SEC <= 0.0:
        if _enhance_progress_bar != null:
            _enhance_progress_bar.value = 100.0
        _on_enhance_progress_animation_finished()
        return
    _enhance_progress_tween = create_tween()
    if _enhance_progress_bar != null:
        _enhance_progress_tween.tween_property(
            _enhance_progress_bar,
            "value",
            100.0,
            ENHANCE_PROGRESS_DURATION_SEC
        ).set_trans(Tween.TRANS_LINEAR)
    else:
        _enhance_progress_tween.tween_interval(ENHANCE_PROGRESS_DURATION_SEC)
    _enhance_progress_tween.finished.connect(_on_enhance_progress_animation_finished, CONNECT_ONE_SHOT)


func _on_dim_layer_gui_input(event: InputEvent) -> void:
    if not visible or not _is_topmost_runtime_modal():
        return
    if not _is_dismiss_event(event):
        return
    get_viewport().set_input_as_handled()
    var dim_layer: ColorRect = get_node_or_null("DimLayer") as ColorRect
    if dim_layer != null:
        dim_layer.accept_event()


func _on_enhance_progress_animation_finished() -> void:
    _stop_enhance_progress_tween()
    _enhance_progress_finished = true
    _try_finish_enhance_presentation()


## 停止进行中的强化进度 tween。
func _stop_enhance_progress_tween() -> void:
    if _enhance_progress_tween != null and _enhance_progress_tween.is_valid():
        _enhance_progress_tween.kill()
    _enhance_progress_tween = null


## 重置为首次打开态：0% 进度、无结果文案、无填充色。
func _reset_enhance_presentation_ui() -> void:
    _stop_enhance_progress_tween()
    _enhance_presentation_active = false
    _block_dismiss = false
    _enhance_progress_finished = false
    _enhance_response_ready = false
    _optimistic_copper_spent = 0
    if _enhance_progress_bar != null:
        _enhance_progress_bar.min_value = 0.0
        _enhance_progress_bar.max_value = 100.0
        _enhance_progress_bar.value = 0.0
        _enhance_progress_bar.visible = true
    _set_enhance_progress_fill_color(Color(0.0, 0.0, 0.0, 0.0))
    _reset_enhance_result_label()


## 初始化进度条 fill 样式，便于运行时切换黄/绿/红。
func _init_enhance_progress_styles() -> void:
    if _enhance_progress_bar == null:
        return
    var fill_variant: Variant = _enhance_progress_bar.get_theme_stylebox("fill")
    if fill_variant is StyleBoxFlat:
        _enhance_progress_fill_style = (fill_variant as StyleBoxFlat).duplicate() as StyleBoxFlat
    else:
        _enhance_progress_fill_style = StyleBoxFlat.new()
        _enhance_progress_fill_style.corner_radius_top_left = 3
        _enhance_progress_fill_style.corner_radius_top_right = 3
        _enhance_progress_fill_style.corner_radius_bottom_right = 3
        _enhance_progress_fill_style.corner_radius_bottom_left = 3
    _set_enhance_progress_fill_color(Color(0.0, 0.0, 0.0, 0.0))


## 更新进度条填充色；透明表示 0% 初始态不显示填充。
func _set_enhance_progress_fill_color(color: Color) -> void:
    if _enhance_progress_fill_style == null or _enhance_progress_bar == null:
        return
    _enhance_progress_fill_style.bg_color = color
    _enhance_progress_bar.add_theme_stylebox_override("fill", _enhance_progress_fill_style)


## 清空右侧强化结果文案，保留固定宽度格子占位。
func _reset_enhance_result_label() -> void:
    if _enhance_result_label == null:
        return
    _enhance_result_label.text = ""
    _enhance_result_label.visible = true


## 取消强化演出；force_unlock 为 true 时立即恢复交互。
func _cancel_enhance_presentation(force_unlock: bool) -> void:
    _stop_enhance_progress_tween()
    _enhance_presentation_active = false
    _block_dismiss = false
    _enhance_progress_finished = false
    _enhance_response_ready = false
    if force_unlock:
        _set_panel_interactive(true)


## 递归禁用/启用弹窗内所有按钮。
func _set_panel_interactive(enabled: bool) -> void:
    var panel: Control = get_node_or_null("CenterContainer/PopupPanel") as Control
    if panel == null:
        return
    _apply_panel_interactive_state(panel, enabled)
    if not enabled:
        if _enhance_button != null:
            _enhance_button.disabled = true
        if _material_select_button != null:
            _material_select_button.disabled = true
    else:
        if _material_select_button != null:
            _material_select_button.disabled = false
        _set_content_mouse_ignore(true)
        _refresh_enhance_button_state()


func _apply_panel_interactive_state(node: Node, enabled: bool) -> void:
    if node is BaseButton:
        var button: BaseButton = node as BaseButton
        button.disabled = not enabled
    elif node is CheckBox:
        var checkbox: CheckBox = node as CheckBox
        checkbox.disabled = not enabled
    for child: Node in node.get_children():
        _apply_panel_interactive_state(child, enabled)
