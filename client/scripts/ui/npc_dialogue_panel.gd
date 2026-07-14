class_name NPCDialoguePanel
extends CanvasLayer

## 玩家点击继续时广播当前节点标识，主场景据此向服务端请求下一个剧情节点。
signal continue_requested(dialogue_id: int, node_id: String)
## 玩家点击选项时广播当前节点与选项标识，主场景据此走服务端权威分支。
signal option_selected(dialogue_id: int, node_id: String, option_id: String)
## 面板关闭时通知主场景解除输入锁定。
signal panel_closed
## 客户端固定过场对白点击继续时广播，不携带服务端剧情节点标识。
signal local_continue_requested

const _TEXT_PHASE_TYPING: int = 0
const _TEXT_PHASE_READY: int = 1
const _CHAR_INTERVAL_SEC: float = 0.035
## 物品提及图标的移动端展示尺寸；这里压到接近单个文字大小，避免喧宾夺主。
const _ITEM_ICON_SIZE: int = 12
## 物品图集兜底路径；优先使用服务端 item_definition.icon。
const _ITEM_ATLAS_TEXTURE_PATH: String = "res://asset/分类/武器/pixel items0.png"
## 服务端把 {item:ID} 渲染成该自定义富文本标签，客户端在原位置替换成物品图标和名称。
const _INLINE_ITEM_TAG_PATTERN: String = "\\[item\\s+id=(\\d+)\\](.*?)\\[/item\\]"
## 说话人角标左侧内边距，用于动态计算面板宽度并让头像更贴近左缘。
const _SPEAKER_PANEL_PADDING_LEFT: float = 4.0
## 说话人角标右侧内边距，用于动态计算面板宽度。
const _SPEAKER_PANEL_PADDING_RIGHT: float = 8.0
## 说话人角标垂直留白，包含背景纹理上下各 14px 的安全边距。
const _SPEAKER_PANEL_PADDING_Y: float = 32.0
## 说话人角标头像尺寸。
const _SPEAKER_ICON_SIZE: int = 28

@onready var _root: Control = get_node_or_null("Root") as Control
@onready var _dim_layer: ColorRect = get_node_or_null("Root/DimLayer") as ColorRect
@onready var _panel: PanelContainer = get_node_or_null("Root/DialogueVBox/Panel") as PanelContainer
@onready var _npc_speaker_panel: PanelContainer = get_node_or_null("Root/DialogueVBox/SpeakerRow/NpcSpeakerPanel") as PanelContainer
@onready var _npc_speaker_label: RichTextLabel = get_node_or_null("Root/DialogueVBox/SpeakerRow/NpcSpeakerPanel/NpcSpeakerLabel") as RichTextLabel
@onready var _player_speaker_panel: PanelContainer = get_node_or_null("Root/DialogueVBox/SpeakerRow/PlayerSpeakerPanel") as PanelContainer
@onready var _player_speaker_label: RichTextLabel = get_node_or_null("Root/DialogueVBox/SpeakerRow/PlayerSpeakerPanel/PlayerSpeakerLabel") as RichTextLabel
@onready var _content_label: RichTextLabel = get_node_or_null("Root/DialogueVBox/Panel/Layout/ContentLabel") as RichTextLabel
@onready var _item_mention_container: HBoxContainer = get_node_or_null("Root/DialogueVBox/Panel/Layout/ItemMentionContainer") as HBoxContainer
@onready var _options_container: VBoxContainer = get_node_or_null("Root/DialogueVBox/Panel/Layout/OptionsContainer") as VBoxContainer
@onready var _continue_button: BaseButton = _resolve_base_button([
	NodePath("Root/DialogueVBox/Panel/Layout/ContinueButton"),
	NodePath("Root/DialogueVBox/Panel/Layout/Button"),
])
@onready var _status_label: Label = get_node_or_null("Root/DialogueVBox/Panel/Layout/StatusLabel") as Label
@onready var _top_close_button: BaseButton = _resolve_base_button([
	NodePath("Root/TopCloseButton"),
	NodePath("Root/DialogueVBox/TopCloseButton"),
	NodePath("Root/DialogueVBox/SpeakerRow/TopCloseButton"),
])

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
## 当前节点里的物品提及；客户端仅用它解析图标，物品名仍以服务端渲染后的正文为准。
var _current_item_mentions: Array[Dictionary] = []
## 打字机计时器，累计到间隔阈值后推进一个可见字符。
var _typing_elapsed: float = 0.0
## 当前已显示的字符数。
var _visible_char_count: int = 0
## 服务端下发的提示文案，打字结束后再展示。
var _pending_effect_notice: String = ""
## 等待服务端回包或演出完成时锁住点击。
var _input_locked: bool = false
## 当前是否展示由客户端过场脚本提供的本地对白。
var _local_dialogue_active: bool = false
## 懒加载的物品图集兜底纹理。
var _item_atlas_texture: Texture2D = null

## 绑定按钮与点击区域，并在启动时保持隐藏。
func _ready() -> void:
	if _continue_button != null:
		_set_continue_button_label("继续")
		_continue_button.visible = false
		if not _continue_button.pressed.is_connected(_on_continue_button_pressed):
			_continue_button.pressed.connect(_on_continue_button_pressed)
	if _top_close_button != null and not _top_close_button.pressed.is_connected(_on_top_close_button_pressed):
		_top_close_button.pressed.connect(_on_top_close_button_pressed)
	if _dim_layer != null:
		_dim_layer.gui_input.connect(_on_advance_gui_input)
	if _panel != null:
		_panel.gui_input.connect(_on_advance_gui_input)
	set_process(false)
	hide_panel(false)

## 按候选路径查找按钮节点；用于兼容 npc_dialogue_panel.tscn 重构前后的按钮命名。
func _resolve_base_button(paths: Array[NodePath]) -> BaseButton:
	for path: NodePath in paths:
		var node: Node = get_node_or_null(path)
		if node is BaseButton:
			return node as BaseButton
	return null

## 设置继续按钮文案；兼容旧 RuntimeActionButton 与新通用 continue_button 场景。
func _set_continue_button_label(label: String) -> void:
	if _continue_button == null:
		return
	if _continue_button.has_method("set_button_label"):
		_continue_button.call("set_button_label", label)
		return
	var nested_label: Label = _continue_button.get_node_or_null("TextureRect/Control/Control") as Label
	if nested_label != null:
		if nested_label.text.strip_edges().is_empty():
			nested_label.text = label
		return
	_continue_button.text = label

## 用剧情节点刷新整个面板并启动打字机效果；local_mode 为 true 时继续操作不请求服务端。
func show_dialogue(npc_name: String, node_payload: Dictionary, local_mode: bool = false) -> void:
	_input_locked = false
	_local_dialogue_active = local_mode
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
	var speaker_portrait: Texture2D = null
	var direct_portrait_variant: Variant = node_payload.get("portrait_texture", null)
	if direct_portrait_variant is Texture2D:
		speaker_portrait = direct_portrait_variant as Texture2D
	else:
		speaker_portrait = PortraitRegistry.load_dialogue_portrait(
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
	if _top_close_button != null:
		_top_close_button.visible = not _local_dialogue_active
	_begin_typewriter()
	if _root != null:
		_root.show()
	visible = true

## 展示客户端固定过场脚本提供的一句对白，内容完成后由 local_continue_requested 推进本地演出。
func show_local_dialogue(
	speaker_name: String,
	content: String,
	portrait_key: String = "",
	is_player_speaking: bool = false,
	content_format: String = "bbcode",
	portrait_texture: Texture2D = null
) -> void:
	var raw_speaker: String = "@player" if is_player_speaking else speaker_name
	var node_payload: Dictionary = {
		"speaker": raw_speaker,
		"portrait_key": portrait_key,
		"portrait_texture": portrait_texture,
		"is_player_speaker": is_player_speaking,
		"content": content,
		"content_format": content_format,
		"options": []
	}
	show_dialogue(speaker_name, node_payload, true)

## 返回面板当前是否正在展示客户端固定过场对白。
func is_local_dialogue_active() -> bool:
	return _local_dialogue_active

## 在等待服务端回包或等待客户端剧情动画完成时锁住输入并展示状态文案。
func show_waiting_state(status_text: String) -> void:
	_input_locked = true
	_stop_typewriter()
	if _continue_button != null:
		_continue_button.visible = false
		_continue_button.disabled = true
	if _options_container != null:
		for child_variant: Variant in _options_container.get_children():
			if child_variant is Button:
				var option_button: Button = child_variant as Button
				option_button.disabled = true
	if _status_label != null:
		_status_label.text = status_text

## 隐藏对话面板并清理当前节点引用，避免旧节点被误复用。
func hide_panel(emit_closed: bool = true) -> void:
	_input_locked = false
	_local_dialogue_active = false
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
	if _top_close_button != null:
		_top_close_button.visible = true
	if _root != null:
		_root.hide()
	visible = false
	if emit_closed:
		panel_closed.emit()


## 响应右上角关闭按钮，允许玩家主动结束当前对话面板。
func _on_top_close_button_pressed() -> void:
	if _input_locked or _local_dialogue_active:
		return
	hide_panel()

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

## 键盘输入：打字中任意键只跳过当前段落；本地固定过场对白额外支持继续键推进。
func _input(event: InputEvent) -> void:
	if not visible or _input_locked:
		return
	if event is not InputEventKey:
		return
	var key_event: InputEventKey = event as InputEventKey
	if not key_event.pressed or key_event.echo:
		return
	if _text_phase == _TEXT_PHASE_TYPING:
		get_viewport().set_input_as_handled()
		_skip_typewriter()
		return
	if _local_dialogue_active and _is_local_advance_key(key_event):
		get_viewport().set_input_as_handled()
		_handle_advance_input()

## 判断键盘事件是否为本地过场允许的继续键。
func _is_local_advance_key(key_event: InputEventKey) -> bool:
	var keycode: Key = key_event.keycode
	var physical_keycode: Key = key_event.physical_keycode
	return (
		keycode == KEY_ENTER
		or keycode == KEY_KP_ENTER
		or keycode == KEY_5
		or keycode == KEY_KP_5
		or physical_keycode == KEY_ENTER
		or physical_keycode == KEY_KP_ENTER
		or physical_keycode == KEY_5
		or physical_keycode == KEY_KP_5
	)

## 根据服务端选项配置动态创建一个移动端友好的按钮。
func _add_option_button(option_payload: Dictionary) -> void:
	if _options_container == null:
		return
	var option_button: Button = Button.new()
	option_button.text = str(option_payload.get("text", option_payload.get("option_text", "继续")))
	DialogueActionButtonTheme.apply(option_button, true)
	option_button.pressed.connect(_on_option_button_pressed.bind(str(option_payload.get("option_id", ""))))
	_options_container.add_child(option_button)

## 清空上一节点残留的所有选项按钮，避免旧分支数据污染新节点展示。
func _clear_option_buttons() -> void:
	if _options_container == null:
		return
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
	var resolved_format: String = content_format.strip_edges().to_lower()
	if resolved_format != "bbcode" and RichTextContent.contains_bbcode(full_text):
		resolved_format = "bbcode"
	var use_bbcode: bool = resolved_format == "bbcode"
	_content_label.bbcode_enabled = use_bbcode
	_append_dialogue_text_with_inline_items(full_text, use_bbcode)

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

## 按服务端自定义 [item id=] 标签把图标插入到正文原位置，保持剧情文案和物品引用顺序一致。
func _append_dialogue_text_with_inline_items(full_text: String, use_bbcode: bool) -> void:
	var item_tag_regex: RegEx = _build_inline_item_tag_regex()
	if item_tag_regex == null:
		_append_dialogue_text_segment(full_text, use_bbcode)
		_append_inline_item_icons()
		return
	var matches: Array = item_tag_regex.search_all(full_text)
	if matches.is_empty():
		_append_dialogue_text_segment(full_text, use_bbcode)
		_append_inline_item_icons()
		return
	var cursor: int = 0
	for match_variant: Variant in matches:
		if match_variant is not RegExMatch:
			continue
		var item_match: RegExMatch = match_variant as RegExMatch
		var segment_start: int = item_match.get_start()
		var segment_end: int = item_match.get_end()
		if segment_start > cursor:
			_append_dialogue_text_segment(full_text.substr(cursor, segment_start - cursor), use_bbcode)
		var item_id: int = int(item_match.get_string(1))
		var item_name: String = item_match.get_string(2)
		_append_inline_item_reference(item_id, item_name)
		cursor = segment_end
	if cursor < full_text.length():
		_append_dialogue_text_segment(full_text.substr(cursor), use_bbcode)

## 构建行内物品标签解析器；编译失败时返回 null，避免坏正则中断对白流程。
func _build_inline_item_tag_regex() -> RegEx:
	var item_tag_regex: RegEx = RegEx.new()
	if item_tag_regex.compile(_INLINE_ITEM_TAG_PATTERN) != OK:
		return null
	return item_tag_regex

## 按当前格式追加一段普通正文；富文本正文使用 append_text，纯文本正文使用 add_text。
func _append_dialogue_text_segment(text_segment: String, use_bbcode: bool) -> void:
	if _content_label == null or text_segment.is_empty():
		return
	if use_bbcode:
		# append_text 会强制重建解析缓冲，避免连续两句正文相同时 text 属性赋值被 Godot 视为无变化。
		_content_label.append_text(text_segment)
		return
	_content_label.add_text(text_segment)

## 在正文当前位置追加“物品图标 + 服务端物品名”，实现真正的行内物品引用。
func _append_inline_item_reference(item_id: int, item_name: String) -> void:
	if _content_label == null:
		return
	var mention: Dictionary = _find_item_mention(item_id)
	var icon_texture: Texture2D = _resolve_item_icon_texture(mention)
	if icon_texture != null:
		_content_label.add_image(icon_texture, _ITEM_ICON_SIZE, _ITEM_ICON_SIZE)
	var resolved_item_name: String = str(mention.get("item_name", item_name)).strip_edges()
	if resolved_item_name.is_empty():
		resolved_item_name = item_name.strip_edges()
	if resolved_item_name.is_empty():
		resolved_item_name = "物品%d" % item_id
	_content_label.add_text(resolved_item_name)

## 根据 item_id 查找服务端随正文返回的物品提及信息，找不到时返回可安全解析图标的最小字典。
func _find_item_mention(item_id: int) -> Dictionary:
	for mention: Dictionary in _current_item_mentions:
		if int(mention.get("item_id", 0)) == item_id:
			return mention
	return {
		"item_id": item_id,
		"item_name": "物品%d" % item_id,
	}

## 兼容旧服务端：没有 [item id=] 标签但仍返回 mentioned_items 时，把 icon 追加在正文末尾。
func _append_inline_item_icons() -> void:
	if _content_label == null:
		return
	for mention: Dictionary in _current_item_mentions:
		var icon_texture: Texture2D = _resolve_item_icon_texture(mention)
		if icon_texture == null:
			continue
		_content_label.add_image(icon_texture, _ITEM_ICON_SIZE, _ITEM_ICON_SIZE)

## 创建一个仅包含物品 icon 的移动端紧凑标签，避免与服务端正文中的物品名重复显示。
func _create_item_mention_chip(mention: Dictionary) -> Control:
	var icon_rect: TextureRect = TextureRect.new()
	icon_rect.custom_minimum_size = Vector2(float(_ITEM_ICON_SIZE), float(_ITEM_ICON_SIZE))
	icon_rect.texture = _resolve_item_icon_texture(mention)
	icon_rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	icon_rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	icon_rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	icon_rect.tooltip_text = str(mention.get("item_name", "物品%d" % int(mention.get("item_id", 0))))
	return icon_rect

## 按 item_id 从本地注册表解析图标贴图。
func _resolve_item_icon_texture(mention: Dictionary) -> Texture2D:
	return ItemIcons.resolve_texture(int(mention.get("item_id", 0)))

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

## 布局完成后重新测量角标尺寸，确保 HBox 左右对齐准确且高度紧凑。
func _remeasure_speaker_panel(panel: PanelContainer, label: RichTextLabel) -> void:
	if panel == null or label == null or not panel.visible:
		return
	panel.custom_minimum_size = _resolve_speaker_panel_size(label)

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
	panel.custom_minimum_size = _resolve_speaker_panel_size(label)
	panel.show()


## 根据标签内容计算说话人角标尺寸，并为背景纹理保留完整的上下边距。
func _resolve_speaker_panel_size(label: RichTextLabel) -> Vector2:
	var panel_width: float = (
		label.get_content_width()
		+ _SPEAKER_PANEL_PADDING_LEFT
		+ _SPEAKER_PANEL_PADDING_RIGHT
	)
	var content_height: float = maxf(label.get_content_height(), float(_SPEAKER_ICON_SIZE))
	var panel_height: float = content_height + _SPEAKER_PANEL_PADDING_Y
	return Vector2(panel_width, panel_height)

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
	if _local_dialogue_active:
		_input_locked = true
		local_continue_requested.emit()
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
