extends CanvasLayer
## A basic dialogue balloon for use with Dialogue Manager.

const NAME_PANEL_PADDING := 20.0
const NAME_ICON_SIZE := 28
## 对话框底边距屏幕底部的固定像素距离，保证“地板”y 坐标不变。
const DIALOGUE_BOX_BOTTOM_OFFSET := 20.0
## 对话框左右半宽（总宽 = 2 * 该值），与 DialogueVBox 宽度保持一致。
const DIALOGUE_PANEL_HALF_WIDTH := 160.0
## 单行或短句时的最小对话框高度。
const DIALOGUE_BOX_MIN_HEIGHT := 72.0
## 对话框内边距、进度指示行等额外高度，避免文本贴边。
const DIALOGUE_BOX_EXTRA_PADDING := 36.0


## The dialogue resource
@export var dialogue_resource: DialogueResource

## Start from a given title when using balloon as a [Node] in a scene.
@export var start_from_title: String = ""

## Optional portrait shown beside the speaker name.
@export var npc_portrait: Texture2D

## Optional portrait shown when the player is speaking.
@export var player_portrait: Texture2D

## If running as a [Node] in a scene then auto start the dialogue.
@export var auto_start: bool = false

## If all other input is blocked as long as dialogue is shown.
@export var will_block_other_input: bool = true

## The action to use for advancing the dialogue
@export var next_action: StringName = &"ui_accept"

## The action to use to skip typing the dialogue
@export var skip_action: StringName = &"ui_cancel"

## A sound player for voice lines (if they exist).
@onready var audio_stream_player: AudioStreamPlayer = %AudioStreamPlayer

## Temporary game states
var temporary_game_states: Array = []

## See if we are waiting for the player
var is_waiting_for_input: bool = false

## See if we are running a long mutation and should hide the balloon
var will_hide_balloon: bool = false

## A dictionary to store any ephemeral variables
var locals: Dictionary = {}

var _locale: String = TranslationServer.get_locale()

## The current line
var dialogue_line: DialogueLine:
    set(value):
        if value:
            dialogue_line = value
            apply_dialogue_line()
        else:
            # The dialogue has finished so close the balloon
            if owner == null:
                queue_free()
            else:
                hide()
    get:
        return dialogue_line

## A cooldown timer for delaying the balloon hide when encountering a mutation.
var mutation_cooldown: Timer = Timer.new()

## The base balloon anchor
@onready var balloon: Control = %Balloon

## 整组对话 UI 的外层容器，包含说话人角标行与正文框。
@onready var dialogue_vbox: VBoxContainer = %DialogueVBox

## NPC 说话时的左上角名字立绘框
@onready var npc_name_panel: PanelContainer = %NpcName

## NPC 名字标签
@onready var character_label: RichTextLabel = %CharacterLabel

## 玩家说话时的右上角名字立绘框
@onready var player_name_panel: PanelContainer = %PlayerName

## 玩家名字标签
@onready var player_character_label: RichTextLabel = %PlayerCharacterLabel

## The portrait shown alongside the speaking character's name
@onready var portrait_rect: TextureRect = %NpcPortrait

## 底部对话框外层容器
@onready var dialogue_box_container: MarginContainer = %DialogueBox

## The label showing the currently spoken dialogue
@onready var dialogue_label: DialogueLabel = %DialogueLabel

## The menu of responses
@onready var responses_menu: DialogueResponsesMenu = %ResponsesMenu

## Indicator to show that player can progress dialogue.
@onready var progress: Polygon2D = %Progress


func _ready() -> void:
    balloon.hide()
    balloon.resized.connect(_on_balloon_resized)
    Engine.get_singleton("DialogueManager").mutated.connect(_on_mutated)

    # If the responses menu doesn't have a next action set, use this one
    if responses_menu.next_action.is_empty():
        responses_menu.next_action = next_action

    mutation_cooldown.timeout.connect(_on_mutation_cooldown_timeout)
    add_child(mutation_cooldown)

    if auto_start:
        if not is_instance_valid(dialogue_resource):
            assert(false, DMConstants.get_error_message(DMConstants.ERR_MISSING_RESOURCE_FOR_AUTOSTART))
        start()


func _process(delta: float) -> void:
    if is_instance_valid(dialogue_line):
        progress.visible = not dialogue_label.is_typing and dialogue_line.responses.size() == 0 and not dialogue_line.has_tag("voice")


func _unhandled_input(_event: InputEvent) -> void:
    # Only the balloon is allowed to handle input while it's showing
    if will_block_other_input:
        get_viewport().set_input_as_handled()


func _notification(what: int) -> void:
    ## Detect a change of locale and update the current dialogue line to show the new language
    if what == NOTIFICATION_TRANSLATION_CHANGED and _locale != TranslationServer.get_locale() and is_instance_valid(dialogue_label):
        _locale = TranslationServer.get_locale()
        var visible_ratio: float = dialogue_label.visible_ratio
        dialogue_line = await dialogue_resource.get_next_dialogue_line(dialogue_line.id)
        if visible_ratio < 1:
            dialogue_label.skip_typing()


## Start some dialogue
func start(with_dialogue_resource: DialogueResource = null, title: String = "", extra_game_states: Array = []) -> void:
    temporary_game_states = [self] + extra_game_states
    is_waiting_for_input = false
    if is_instance_valid(with_dialogue_resource):
        dialogue_resource = with_dialogue_resource
    if not title.is_empty():
        start_from_title = title
    dialogue_line = await dialogue_resource.get_next_dialogue_line(start_from_title, temporary_game_states)
    show()


## Apply any changes to the balloon given a new [DialogueLine].
func apply_dialogue_line() -> void:
    mutation_cooldown.stop()

    progress.hide()
    is_waiting_for_input = false
    balloon.focus_mode = Control.FOCUS_ALL
    balloon.grab_focus()

    var speaker_name: String = tr(dialogue_line.character, "dialogue")
    var is_player_speaking: bool = _is_player_speaker(speaker_name)
    var resolved_player_portrait: Texture2D = PortraitRegistry.load_player_dialogue_portrait()
    var npc_speaker_portrait: Texture2D = PortraitRegistry.load_npc_dialogue_portrait("default")
    if npc_portrait != null:
        var cropped_npc_portrait: Texture2D = PortraitRegistry.crop_upper_body_portrait(npc_portrait)
        if cropped_npc_portrait != null:
            npc_speaker_portrait = cropped_npc_portrait
    var player_speaker_portrait: Texture2D = resolved_player_portrait

    portrait_rect.texture = npc_speaker_portrait if not is_player_speaking else player_speaker_portrait
    portrait_rect.hide()

    balloon.show()
    will_hide_balloon = false

    await _configure_speaker_name_panel(
        npc_name_panel,
        character_label,
        speaker_name,
        npc_speaker_portrait,
        false,
        not is_player_speaking and not dialogue_line.character.is_empty()
    )
    await _configure_speaker_name_panel(
        player_name_panel,
        player_character_label,
        speaker_name,
        player_speaker_portrait,
        true,
        is_player_speaking and not dialogue_line.character.is_empty()
    )

    dialogue_label.hide()
    dialogue_label.dialogue_line = dialogue_line
    dialogue_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART

    responses_menu.hide()
    responses_menu.responses = dialogue_line.responses

    await _layout_dialogue_box()

    dialogue_label.show()
    if not dialogue_line.text.is_empty():
        dialogue_label.type_out()
        await dialogue_label.finished_typing

    # Wait for next line
    if dialogue_line.has_tag("voice"):
        audio_stream_player.stream = load(dialogue_line.get_tag_value("voice"))
        audio_stream_player.play()
        await audio_stream_player.finished
        next(dialogue_line.next_id)
    elif dialogue_line.responses.size() > 0:
        balloon.focus_mode = Control.FOCUS_NONE
        responses_menu.show()
        _layout_responses_menu()
    elif dialogue_line.time != "":
        var time: float = dialogue_line.text.length() * 0.02 if dialogue_line.time == "auto" else dialogue_line.time.to_float()
        await get_tree().create_timer(time).timeout
        next(dialogue_line.next_id)
    else:
        is_waiting_for_input = true
        balloon.focus_mode = Control.FOCUS_ALL
        balloon.grab_focus()


## 刷新单个说话人角标（左上 NPC / 右上玩家），仅当前说话者可见。
func _configure_speaker_name_panel(
    panel: PanelContainer,
    label: RichTextLabel,
    speaker_name: String,
    speaker_portrait: Texture2D,
    is_player: bool,
    should_show: bool
) -> void:
    if not should_show:
        panel.custom_minimum_size = Vector2.ZERO
        panel.hide()
        return

    label.visible = true
    label.clear()
    label.fit_content = true
    label.autowrap_mode = TextServer.AUTOWRAP_OFF
    if is_player:
        label.append_text(speaker_name)
        if speaker_portrait != null:
            label.append_text(" ")
            label.add_image(speaker_portrait, NAME_ICON_SIZE, NAME_ICON_SIZE)
    else:
        if speaker_portrait != null:
            label.add_image(speaker_portrait, NAME_ICON_SIZE, NAME_ICON_SIZE)
            label.append_text(" ")
        label.append_text(speaker_name)

    panel.show()
    await get_tree().process_frame
    var panel_width: float = label.get_content_width() + NAME_PANEL_PADDING
    panel.custom_minimum_size = Vector2(panel_width, 0.0)


## 按文本行数向上扩展整组对话 UI，底边固定。
func _layout_dialogue_box() -> void:
    await get_tree().process_frame
    var inner_width: float = (DIALOGUE_PANEL_HALF_WIDTH * 2.0) - 36.0
    if inner_width > 0.0:
        dialogue_label.custom_minimum_size.x = inner_width
    await get_tree().process_frame
    var vbox_height: float = dialogue_vbox.get_minimum_size().y
    if vbox_height < DIALOGUE_BOX_MIN_HEIGHT + DIALOGUE_BOX_EXTRA_PADDING:
        vbox_height = DIALOGUE_BOX_MIN_HEIGHT + DIALOGUE_BOX_EXTRA_PADDING
    var bottom_y: float = balloon.size.y - DIALOGUE_BOX_BOTTOM_OFFSET
    dialogue_vbox.offset_left = -DIALOGUE_PANEL_HALF_WIDTH
    dialogue_vbox.offset_right = DIALOGUE_PANEL_HALF_WIDTH
    dialogue_vbox.offset_bottom = -DIALOGUE_BOX_BOTTOM_OFFSET
    dialogue_vbox.offset_top = bottom_y - vbox_height
    _layout_responses_menu()


## 让分支选项菜单跟随对话框顶部区域，避免与正文重叠。
func _layout_responses_menu() -> void:
    if not responses_menu.visible:
        return
    responses_menu.offset_top = dialogue_vbox.offset_top + 12.0
    responses_menu.offset_bottom = dialogue_vbox.offset_top + 40.0


func _on_balloon_resized() -> void:
    if not balloon.visible or dialogue_line == null:
        return
    _layout_dialogue_box()


func _is_player_speaker(speaker_name: String) -> bool:
    var stripped_name: String = speaker_name.strip_edges()
    if stripped_name == "@player" or stripped_name == "$player":
        return true
    var configured_player_name: String = _get_player_name().strip_edges()
    if not configured_player_name.is_empty() and stripped_name == configured_player_name:
        return true
    var game_state_name: String = str(GameState.player_snapshot.get("name", "")).strip_edges()
    if not game_state_name.is_empty() and stripped_name == game_state_name:
        return true
    return false


func _get_player_name() -> String:
	if _has_some_global():
		var some_global: Node = SomeGlobal
		var configured_name: Variant = some_global.get("some_character_name")
		if configured_name is String and not String(configured_name).is_empty():
			return configured_name
	return "玩家名"


func _has_some_global() -> bool:
    return get_node_or_null("/root/SomeGlobal") != null


## Go to the next line
func next(next_id: String) -> void:
    dialogue_line = await dialogue_resource.get_next_dialogue_line(next_id, temporary_game_states)


#region Signals


func _on_mutation_cooldown_timeout() -> void:
    if will_hide_balloon:
        will_hide_balloon = false
        balloon.hide()


func _on_mutated(mutation: Dictionary) -> void:
    if not mutation.is_inline:
        is_waiting_for_input = false
        will_hide_balloon = true
        mutation_cooldown.start(0.1)


func _on_balloon_gui_input(event: InputEvent) -> void:
    # See if we need to skip typing of the dialogue
    if dialogue_label.is_typing:
        var mouse_was_clicked: bool = event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.is_pressed()
        var skip_button_was_pressed: bool = event.is_action_pressed(skip_action)
        if mouse_was_clicked or skip_button_was_pressed:
            get_viewport().set_input_as_handled()
            dialogue_label.skip_typing()
            return

    if not is_waiting_for_input: return
    if dialogue_line.responses.size() > 0: return

    # When there are no response options the balloon itself is the clickable thing
    get_viewport().set_input_as_handled()

    if event is InputEventMouseButton and event.is_pressed() and event.button_index == MOUSE_BUTTON_LEFT:
        next(dialogue_line.next_id)
    elif event.is_action_pressed(next_action) and get_viewport().gui_get_focus_owner() == balloon:
        next(dialogue_line.next_id)


func _on_responses_menu_response_selected(response: DialogueResponse) -> void:
    next(response.next_id)


#endregion
