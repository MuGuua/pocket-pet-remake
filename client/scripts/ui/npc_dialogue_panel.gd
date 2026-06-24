class_name NPCDialoguePanel
extends CanvasLayer

## 玩家点击继续时广播当前节点标识，主场景据此向服务端请求下一个剧情节点。
signal continue_requested(dialogue_id: int, node_id: String)
## 玩家点击选项时广播当前节点与选项标识，主场景据此走服务端权威分支。
signal option_selected(dialogue_id: int, node_id: String, option_id: String)
## 面板关闭时通知主场景解除输入锁定。
signal panel_closed

const _TEXT_PHASE_TYPING: int = 0
const _TEXT_PHASE_READY: int = 1
const _CHAR_INTERVAL_SEC: float = 0.035
## 物品提及图标的移动端展示尺寸；这里压到接近单个文字大小，避免喧宾夺主。
const _ITEM_ICON_SIZE: int = 12
## 物品图集兜底路径；优先使用服务端 item_definition.icon。
const _ITEM_ATLAS_TEXTURE_PATH: String = "res://asset/分类/武器/pixel items0.png"
## 说话人角标内边距，用于动态计算面板宽度。
const _SPEAKER_PANEL_PADDING: float = 20.0
## 说话人角标头像尺寸。
const _SPEAKER_ICON_SIZE: int = 28

@onready var _root: Control = $Root
@onready var _dim_layer: ColorRect = $Root/DimLayer
@onready var _panel: PanelContainer = $Root/DialogueVBox/Panel
@onready var _npc_speaker_panel: PanelContainer = %NpcSpeakerPanel
@onready var _npc_speaker_label: RichTextLabel = %NpcSpeakerLabel
@onready var _player_speaker_panel: PanelContainer = %PlayerSpeakerPanel
@onready var _player_speaker_label: RichTextLabel = %PlayerSpeakerLabel
@onready var _content_label: RichTextLabel = $Root/DialogueVBox/Panel/Margin/Layout/ContentLabel
@onready var _item_mention_container: HBoxContainer = $Root/DialogueVBox/Panel/Margin/Layout/ItemMentionContainer
@onready var _options_container: VBoxContainer = $Root/DialogueVBox/Panel/Margin/Layout/OptionsContainer
@onready var _continue_button: Button = $Root/DialogueVBox/Panel/Margin/Layout/ContinueButton
@onready var _status_label: Label = $Root/DialogueVBox/Panel/Margin/Layout/StatusLabel

## 当前剧情节点 ID，供按钮回调原样带回服务端做权威校验。
var _current_node_id: String = ""
## 当前剧情实例 ID，供按钮回调原样带回服务端做权威校验。
var _current_dialogue_id: int = 0
## 当前节点是否包含分支选项；有选项时点击空白区域不会自动推进。
var _has_options: bool = false
## 当前文本展示阶段：打字中 / 已展示完整文本。
var _text_phase: int = _TEXT_PHASE_READY
## 当前节点完整文本，供打字机效果逐字显示。
var _full_content_text: String = ""
## 当前内容格式，plain 或 bbcode。
var _content_format: String = "plain"
## 当前节点里的物品提及；正文仍由服务端控制，这里只在正文尾部补 icon。
var _current_item_mentions: Array[Dictionary] = []
## 打字机计时器，累计到间隔阈值后推进一个可见字符。
var _typing_elapsed: float = 0.0
## 当前已显示的字符数。
var _visible_char_count: int = 0
## 服务端下发的提示文案，打字结束后再展示。
var _pending_effect_notice: String = ""
## 等待服务端回包或演出完成时锁住点击。
var _input_locked: bool = false
## 懒加载的物品图集兜底纹理。
var _item_atlas_texture: Texture2D = null

## 绑定按钮与点击区域，并在启动时保持隐藏。
func _ready() -> void:
	if _continue_button != null:
		DialogueActionButtonTheme.apply(_continue_button, false)
		_continue_button.visible = false
		_continue_button.pressed.connect(_on_continue_button_pressed)
	if _dim_layer != null:
		_dim_layer.gui_input.connect(_on_advance_gui_input)
	if _panel != null:
		_panel.gui_input.connect(_on_advance_gui_input)
	set_process(false)
	hide_panel(false)

## 用服务端返回的剧情节点刷新整个面板，并启动打字机效果。
func show_dialogue(npc_name: String, node_payload: Dictionary) -> void:
	_input_locked = false
	_current_dialogue_id = int(node_payload.get("dialogue_id", 0))
	_current_node_id = str(node_payload.get("node_id", ""))
	var raw_speaker: String = str(node_payload.get("speaker", ""))
	var portrait_key: String = str(node_payload.get("portrait_key", ""))
	var speaker_name: String = PortraitRegistry.resolve_speaker_display(raw_speaker, npc_name)
	var resolved_portrait_key: String = PortraitRegistry.resolve_portrait_key(
		portrait_key,
		raw_speaker,
		npc_name
	)
	var is_player_speaking: bool = bool(node_payload.get("is_player_speaker", false))
	if not is_player_speaking:
		is_player_speaking = _is_player_speaker(raw_speaker, resolved_portrait_key)
	var speaker_portrait: Texture2D = PortraitRegistry.load_dialogue_portrait(
		resolved_portrait_key,
		is_player_speaking
	)
	_full_content_text = str(node_payload.get("content", ""))
	_content_format = str(node_payload.get("content_format", "plain"))
	_pending_effect_notice = str(node_payload.get("effect_notice", ""))
	var options_variant: Variant = node_payload.get("options", [])
	await _configure_speaker_panels(speaker_name, speaker_portrait, is_player_speaking)
	_render_item_mentions(node_payload.get("mentioned_items", []))
	_clear_option_buttons()
	_has_options = false
	if options_variant is Array:
		for option_variant: Variant in options_variant:
			if option_variant is not Dictionary:
				continue
			_has_options = true
			_add_option_button(option_variant as Dictionary)
	if _options_container != null:
		_options_container.visible = false
	if _continue_button != null:
		_continue_button.visible = false
		_continue_button.disabled = false
	_begin_typewriter()
	if _root != null:
		_root.show()
	visible = true

## 在等待服务端回包或等待客户端剧情动画完成时锁住输入并展示状态文案。
func show_waiting_state(status_text: String) -> void:
	_input_locked = true
	_stop_typewriter()
	if _continue_button != null:
		_continue_button.visible = false
		_continue_button.disabled = true
	for child_variant: Variant in _options_container.get_children():
		if child_variant is Button:
			var option_button: Button = child_variant as Button
			option_button.disabled = true
	if _status_label != null:
		_status_label.text = status_text

## 隐藏对话面板并清理当前节点引用，避免旧节点被误复用。
func hide_panel(emit_closed: bool = true) -> void:
	_input_locked = false
	_stop_typewriter()
	_current_dialogue_id = 0
	_current_node_id = ""
	_full_content_text = ""
	_current_item_mentions.clear()
	_pending_effect_notice = ""
	_clear_item_mentions()
	if _npc_speaker_panel != null:
		_npc_speaker_panel.hide()
	if _player_speaker_panel != null:
		_player_speaker_panel.hide()
	if _continue_button != null:
		_continue_button.visible = false
	if _root != null:
		_root.hide()
	visible = false
	if emit_closed:
		panel_closed.emit()

## 逐字推进打字机效果。
func _process(delta: float) -> void:
	if _text_phase != _TEXT_PHASE_TYPING or _input_locked:
		return
	var total_chars: int = _get_total_char_count()
	if total_chars <= 0:
		_finish_typewriter()
		return
	_typing_elapsed += delta
	while _typing_elapsed >= _CHAR_INTERVAL_SEC and _visible_char_count < total_chars:
		_typing_elapsed -= _CHAR_INTERVAL_SEC
		_visible_char_count += 1
		_apply_visible_characters(_visible_char_count)
	if _visible_char_count >= total_chars:
		_finish_typewriter()

## 根据服务端选项配置动态创建一个移动端友好的按钮。
func _add_option_button(option_payload: Dictionary) -> void:
	var option_button: Button = Button.new()
	option_button.text = str(option_payload.get("text", option_payload.get("option_text", "继续")))
	DialogueActionButtonTheme.apply(option_button, true)
	option_button.pressed.connect(_on_option_button_pressed.bind(str(option_payload.get("option_id", ""))))
	_options_container.add_child(option_button)

## 清空上一节点残留的所有选项按钮，避免旧分支数据污染新节点展示。
func _clear_option_buttons() -> void:
	for child_variant: Variant in _options_container.get_children():
		if child_variant is Node:
			var child_node: Node = child_variant as Node
			child_node.queue_free()

## 启动当前节点的打字机展示。
func _begin_typewriter() -> void:
	_text_phase = _TEXT_PHASE_TYPING
	_typing_elapsed = 0.0
	_visible_char_count = 0
	_apply_content_text(_full_content_text, _content_format)
	_apply_visible_characters(0)
	if _status_label != null:
		_status_label.text = ""
	if _full_content_text.is_empty() or _get_total_char_count() <= 0:
		_finish_typewriter()
		return
	set_process(true)

## 立即停止打字机并显示完整文本。
func _stop_typewriter() -> void:
	set_process(false)
	_text_phase = _TEXT_PHASE_READY
	_apply_visible_characters(-1)

## 跳过打字机，直接展示完整文本并进入“点击继续”阶段。
func _skip_typewriter() -> void:
	if _text_phase != _TEXT_PHASE_TYPING:
		return
	_stop_typewriter()
	_on_text_ready()

## 自然播放完打字机后进入“点击继续”阶段。
func _finish_typewriter() -> void:
	if _text_phase != _TEXT_PHASE_TYPING:
		return
	_stop_typewriter()
	_on_text_ready()

## 文本已完整展示后的 UI 状态：无分支时可点击推进，有分支时弹出选项。
func _on_text_ready() -> void:
	if _has_options:
		if _options_container != null:
			_options_container.visible = true
		if _continue_button != null:
			_continue_button.visible = false
		if _status_label != null:
			if _pending_effect_notice.is_empty():
				_status_label.text = "请选择"
			else:
				_status_label.text = _pending_effect_notice
		return
	if _continue_button != null:
		_continue_button.visible = true
		_continue_button.disabled = false
	if _status_label != null:
		if _pending_effect_notice.is_empty():
			_status_label.text = ""
		else:
			_status_label.text = _pending_effect_notice

## 把完整文本写入 RichTextLabel，并按格式启用 BBCode。
func _apply_content_text(full_text: String, content_format: String) -> void:
	if _content_label == null:
		return
	_content_label.clear()
	if content_format == "bbcode":
		_content_label.bbcode_enabled = true
	else:
		_content_label.bbcode_enabled = false
	_content_label.text = full_text
	_append_inline_item_icons()

## 更新 RichTextLabel 当前可见字符数；-1 表示显示全部。
func _apply_visible_characters(visible_chars: int) -> void:
	if _content_label == null:
		return
	_content_label.visible_characters = visible_chars

## 根据服务端解析出的物品提及，只缓存 icon 信息并关闭旧的单独一行容器。
func _render_item_mentions(mentions_variant: Variant) -> void:
	_current_item_mentions.clear()
	if mentions_variant is not Array:
		_clear_item_mentions()
		return
	var mentions: Array = mentions_variant as Array
	for mention_variant: Variant in mentions:
		if mention_variant is not Dictionary:
			continue
		_current_item_mentions.append((mention_variant as Dictionary).duplicate(true))
	_clear_item_mentions()

## 清理上一句对白残留的物品提及 UI。
func _clear_item_mentions() -> void:
	if _item_mention_container == null:
		return
	for child_variant: Variant in _item_mention_container.get_children():
		if child_variant is Node:
			var child_node: Node = child_variant as Node
			_item_mention_container.remove_child(child_node)
			child_node.queue_free()
	_item_mention_container.visible = false

## 在正文尾部补 icon，避免单独一行导致视觉换行；正文文字仍完全使用服务端配置内容。
func _append_inline_item_icons() -> void:
	if _content_label == null:
		return
	for mention: Dictionary in _current_item_mentions:
		var icon_texture: Texture2D = _resolve_item_icon_texture(str(mention.get("icon", "")))
		if icon_texture == null:
			continue
		_content_label.add_image(icon_texture, _ITEM_ICON_SIZE, _ITEM_ICON_SIZE)

## 创建一个仅包含物品 icon 的移动端紧凑标签，避免与服务端正文中的物品名重复显示。
func _create_item_mention_chip(mention: Dictionary) -> Control:
	var icon_rect: TextureRect = TextureRect.new()
	icon_rect.custom_minimum_size = Vector2(float(_ITEM_ICON_SIZE), float(_ITEM_ICON_SIZE))
	icon_rect.texture = _resolve_item_icon_texture(str(mention.get("icon", "")))
	icon_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	icon_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	icon_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	icon_rect.tooltip_text = str(mention.get("item_name", "物品%d" % int(mention.get("item_id", 0))))
	return icon_rect

## 从服务端 icon 字段加载纹理，支持 res:// 路径并提供图集首格兜底。
func _resolve_item_icon_texture(icon_ref: String) -> Texture2D:
	var normalized_ref: String = icon_ref.strip_edges()
	if not normalized_ref.is_empty() and normalized_ref.begins_with("res://"):
		var loaded_resource: Resource = load(normalized_ref)
		if loaded_resource is Texture2D:
			return loaded_resource as Texture2D
	return _build_default_item_atlas_texture()

## 使用统一物品图集首格作为缺省图标，避免配置缺失时没有视觉反馈。
func _build_default_item_atlas_texture() -> Texture2D:
	var atlas_source: Texture2D = _get_item_atlas_texture()
	if atlas_source == null:
		return null
	var atlas_texture: AtlasTexture = AtlasTexture.new()
	atlas_texture.atlas = atlas_source
	atlas_texture.region = Rect2(0.0, 0.0, float(_ITEM_ICON_SIZE), float(_ITEM_ICON_SIZE))
	return atlas_texture

## 懒加载物品图集纹理，避免场景初始化时绑定导入缓存。
func _get_item_atlas_texture() -> Texture2D:
	if _item_atlas_texture != null:
		return _item_atlas_texture
	var loaded_texture: Resource = load(_ITEM_ATLAS_TEXTURE_PATH)
	if loaded_texture is Texture2D:
		_item_atlas_texture = loaded_texture as Texture2D
	return _item_atlas_texture

## 返回当前 RichTextLabel 解析后的总字符数。
func _get_total_char_count() -> int:
	if _content_label == null:
		return _full_content_text.length()
	return _content_label.get_total_character_count()

## 判断当前说话人是否为玩家（占位符、立绘 key 或玩家名）。
func _is_player_speaker(raw_speaker: String, resolved_portrait_key: String) -> bool:
	var normalized_speaker: String = raw_speaker.strip_edges()
	if normalized_speaker == "@player" or normalized_speaker == "$player" or normalized_speaker == "{player_name}" or normalized_speaker == "玩家":
		return true
	if resolved_portrait_key.strip_edges() == "player_default":
		return true
	var player_name: String = str(GameState.player_snapshot.get("name", "")).strip_edges()
	if not player_name.is_empty() and normalized_speaker == player_name:
		return true
	if get_node_or_null("/root/SomeGlobal") != null:
		var global_name: String = str(SomeGlobal.some_character_name).strip_edges()
		if not global_name.is_empty() and normalized_speaker == global_name:
			return true
	return false

## 刷新左上 NPC / 右上玩家角标，仅当前说话者可见。
## HBox 结构为 [NpcPanel | Spacer(expand) | PlayerPanel]，左右边线与下方对话框对齐。
func _configure_speaker_panels(
	speaker_name: String,
	speaker_portrait: Texture2D,
	is_player_speaking: bool
) -> void:
	_configure_single_speaker_panel(
		_npc_speaker_panel,
		_npc_speaker_label,
		speaker_name,
		speaker_portrait,
		false,
		not is_player_speaking
	)
	_configure_single_speaker_panel(
		_player_speaker_panel,
		_player_speaker_label,
		speaker_name,
		speaker_portrait,
		true,
		is_player_speaking
	)
	await get_tree().process_frame
	_remeasure_speaker_panel(_npc_speaker_panel, _npc_speaker_label)
	_remeasure_speaker_panel(_player_speaker_panel, _player_speaker_label)

## 布局完成后重新测量角标宽度，确保 HBox 左右对齐准确。
func _remeasure_speaker_panel(panel: PanelContainer, label: RichTextLabel) -> void:
	if panel == null or label == null or not panel.visible:
		return
	var panel_width: float = label.get_content_width() + _SPEAKER_PANEL_PADDING
	panel.custom_minimum_size = Vector2(panel_width, 0.0)

## 配置单个说话人角标：只改内容宽度，位置交给 HBoxContainer + Spacer 处理。
func _configure_single_speaker_panel(
	panel: PanelContainer,
	label: RichTextLabel,
	speaker_name: String,
	speaker_portrait: Texture2D,
	is_player: bool,
	should_show: bool
) -> void:
	if panel == null or label == null:
		return
	if not should_show or speaker_name.is_empty():
		panel.custom_minimum_size = Vector2.ZERO
		panel.hide()
		return
	label.clear()
	label.fit_content = true
	label.autowrap_mode = TextServer.AUTOWRAP_OFF
	if is_player:
		label.append_text(speaker_name)
		if speaker_portrait != null:
			label.append_text(" ")
			label.add_image(speaker_portrait, _SPEAKER_ICON_SIZE, _SPEAKER_ICON_SIZE)
	else:
		if speaker_portrait != null:
			label.add_image(speaker_portrait, _SPEAKER_ICON_SIZE, _SPEAKER_ICON_SIZE)
			label.append_text(" ")
		label.append_text(speaker_name)
	var panel_width: float = label.get_content_width() + _SPEAKER_PANEL_PADDING
	panel.custom_minimum_size = Vector2(panel_width, 0.0)
	panel.show()

## 处理面板与遮罩层的点击：第一次跳过打字，第二次请求下一节点。
func _on_advance_gui_input(event: InputEvent) -> void:
	if _input_locked or not visible:
		return
	if not _is_advance_tap(event):
		return
	get_viewport().set_input_as_handled()
	_handle_advance_input()

## 判断当前输入事件是否为一次有效的“继续/跳过”点击。
func _is_advance_tap(event: InputEvent) -> bool:
	if event is InputEventScreenTouch:
		var touch_event: InputEventScreenTouch = event as InputEventScreenTouch
		return touch_event.pressed
	if event is InputEventMouseButton:
		var mouse_event: InputEventMouseButton = event as InputEventMouseButton
		return mouse_event.pressed and mouse_event.button_index == MOUSE_BUTTON_LEFT
	return false

## 根据当前文本阶段决定是跳过打字还是向主场景请求下一节点。
func _handle_advance_input() -> void:
	if _text_phase == _TEXT_PHASE_TYPING:
		_skip_typewriter()
		return
	if _text_phase != _TEXT_PHASE_READY or _has_options:
		return
	continue_requested.emit(_current_dialogue_id, _current_node_id)

## 响应继续按钮点击，与点击遮罩层保持同一套推进逻辑。
func _on_continue_button_pressed() -> void:
	_handle_advance_input()

## 响应选项按钮点击，并把当前节点与选项标识原样交回主场景。
func _on_option_button_pressed(option_id: String) -> void:
	if _input_locked or _text_phase == _TEXT_PHASE_TYPING:
		return
	option_selected.emit(_current_dialogue_id, _current_node_id, option_id)
