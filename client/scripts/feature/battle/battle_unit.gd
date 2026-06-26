extends CharacterBody2D
class_name BattleUnit

const ATTACK_MOVE_DISTANCE: float = 36.0
## 战斗单位在 260 宽视口下的统一默认缩放，避免人物与宠物占位过满。
const DEFAULT_UNIT_SCALE: Vector2 = Vector2(0.7, 0.7)
## 规划中的当前出手单位略微放大一点，仍基于默认缩放计算，避免高亮时突然过大。
const PLANNING_HIGHLIGHT_SCALE: Vector2 = Vector2(0.742, 0.742)

const ACTION_TYPE_ANIMATIONS: Dictionary = {
    "attack": "普攻",
    "skill": "技能",
    "item": "物品",
    "auto": "自动",
    "escape": "逃跑",
    "buff_tick": "持续伤害",
    "follow_attack": "追击",
    "counter_attack": "反击",
}

const DEFAULT_ANIMATIONS: Array[String] = [
    "待机",
    "普攻",
    "技能",
    "自动",
    "物品",
    "逃跑",
    "追击",
    "反击",
    "持续伤害",
    "受击",
    "治疗",
    "死亡"
]

var actor_id: int = 0
var unit_name: String = ""
var unit_type: String = ""
var skin_id: String = ""
var max_hp: int = 1
var current_hp: int = 1
var buffs: Array[String] = []
var skills: Array[Dictionary] = []
var items: Array[Dictionary] = []
var base_position: Vector2 = Vector2.ZERO
var is_dead: bool = false
var _sprite_tint: Color = Color.WHITE
var _skin: UnitSkin

@onready var _character_sprite: AnimatedSprite2D = %CharacterSprite
@onready var _animation_player: AnimationPlayer = %AnimationPlayer
@onready var _status_root: Control = %StatusRoot
@onready var _hp_bar: ProgressBar = %HpBar
@onready var _target_arrow: Sprite2D = %TargetArrow
@onready var _ally_target_arrow: Sprite2D = %AllyTargetArrow
## 右侧敌方单位头顶名称标签，运行时创建以避免大范围重写 .tscn。
var _name_label: Label = null

const SELECTION_ARROW_NONE: String = ""
const SELECTION_ARROW_ENEMY: String = "enemy"
const SELECTION_ARROW_ALLY: String = "ally"

var _hp_bar_tween: Tween = null
## 血条延迟隐藏令牌，避免连续受击时旧定时器误隐藏新显示的血条。
var _hp_bar_hide_token: int = 0
## 受击/治疗血条动画结束后，再保留一小段时间便于阅读。
const HP_BAR_HIDE_DELAY_AFTER_ANIM_SEC: float = 0.35
var _target_arrow_texture: Texture2D = null
var _chj_renderer: ChjWorldRenderer = null
var _uses_chj_render: bool = false
var _using_png_override: bool = false

const CHJ_SKILL_ANIMATIONS: Array[String] = [
    "普攻",
    "技能",
    "自动",
    "追击",
    "反击",
]

## 仅用于 CHJ 战斗待机/反馈类动画，不应走 chj_skill_path。
const CHJ_REACTION_ANIMATIONS: Array[String] = [
    "待机",
    "战斗待机",
    "受击",
    "治疗",
    "死亡",
    "物品",
    "逃跑",
    "持续伤害",
]

## CHJ 技能演出最长等待秒数，防止资源异常时战斗卡死。
const CHJ_SKILL_MAX_WAIT_SEC: float = 6.0

signal target_clicked(unit: BattleUnit)

func _ready() -> void:
    _ensure_builtin_animations()
    _bind_click_area()


func _process(delta: float) -> void:
    if not _uses_chj_render or _chj_renderer == null:
        return
    # 技能 CHJ 播放期间必须持续 tick，不能被 PNG 战斗待机覆盖阻断。
    if _chj_renderer.is_skill_playing():
        _chj_renderer.tick_world("battle", "down", delta)
        return
    if _using_png_override:
        return
    _chj_renderer.tick_world("battle", "down", delta)


func setup(data: Dictionary, slot_position: Vector2, skin: UnitSkin) -> void:
    _ensure_scene_nodes()
    actor_id = int(data.get("actor_id", data.get("id", 0)))
    unit_name = str(data.get("name", str(actor_id)))
    unit_type = str(data.get("type", "unit"))
    skin_id = str(data.get("skin_id", ""))
    max_hp = max(1, int(data.get("max_hp", 1)))
    current_hp = clamp(int(data.get("hp", max_hp)), 0, max_hp)
    base_position = slot_position
    position = slot_position
    scale = DEFAULT_UNIT_SCALE
    modulate = Color.WHITE
    rotation = 0.0
    is_dead = current_hp <= 0
    buffs.clear()
    skills.clear()
    items.clear()
    var skills_data: Array = data.get("skills", []) as Array
    for skill_value: Variant in skills_data:
        if skill_value is Dictionary:
            skills.append(skill_value as Dictionary)
    var items_data: Array = data.get("items", []) as Array
    for item_value: Variant in items_data:
        if item_value is Dictionary:
            items.append(item_value as Dictionary)
    _apply_skin(skin)
    _refresh_name_label()
    _refresh_hp_display()
    _set_hp_bar_visible(false)
    _update_dead_visual()
    set_selection_arrow(SELECTION_ARROW_NONE)
    _play_idle_animation()

func play_attack(target_position: Vector2, skill_visual: SkillVisualConfig, fallback_skill_id: String = "", fallback_skill_name: String = "", action_type: String = "attack", animation_hint: String = "") -> void:
    if is_dead:
        return
    var direction: Vector2 = (target_position - base_position).normalized()
    if direction.length() < 0.01:
        direction = Vector2(1.0, 0.0)
    var attack_offset: Vector2 = direction * ATTACK_MOVE_DISTANCE
    var move_tween: Tween = create_tween()
    move_tween.tween_property(self, "position", base_position + attack_offset, 0.2).set_trans(Tween.TRANS_SINE).set_ease(Tween.EASE_OUT)
    var animation_name: String = _resolve_action_animation(skill_visual, fallback_skill_id, fallback_skill_name, action_type, animation_hint)
    await _play_animation(animation_name)
    var move_back_tween: Tween = create_tween()
    move_back_tween.tween_property(self, "position", base_position, 0.22).set_trans(Tween.TRANS_SINE).set_ease(Tween.EASE_IN)
    await move_back_tween.finished
    _character_sprite.modulate = _sprite_tint
    _play_idle_animation()

func play_result(result: Dictionary) -> void:
    var result_type: String = str(result.get("result_type", "damage"))
    var previous_hp: int = current_hp
    var target_hp: int = _resolve_target_hp_after(result, result_type, previous_hp)
    current_hp = target_hp
    var should_die: bool = current_hp <= 0 and not is_dead
    if result_type == "heal":
        _show_hp_bar_for_hit()
        _prepare_hp_bar(previous_hp)
        _play_animation("治疗")
        await _animate_hp_bar_to(current_hp, FloatingText.DISPLAY_DURATION)
        _schedule_hp_bar_hide()
    elif result_type == "damage":
        _show_hp_bar_for_hit()
        _prepare_hp_bar(previous_hp)
        await _play_animation("受击")
        await _animate_hp_bar_to(current_hp, FloatingText.DISPLAY_DURATION)
        _schedule_hp_bar_hide()
    elif result_type == "defeat":
        if not is_dead:
            _show_hp_bar_for_hit()
            _prepare_hp_bar(previous_hp)
            await _play_animation("受击")
            await _animate_hp_bar_to(0, FloatingText.DISPLAY_DURATION)
            current_hp = 0
            await play_death()
            _set_hp_bar_visible(false)
        return
    else:
        var hp_changed: bool = target_hp != previous_hp
        if hp_changed:
            _show_hp_bar_for_hit()
            _prepare_hp_bar(previous_hp)
        await _play_animation("受击")
        if hp_changed:
            await _animate_hp_bar_to(current_hp, FloatingText.DISPLAY_DURATION)
            _schedule_hp_bar_hide()
        else:
            _refresh_hp_display()
    if should_die:
        await play_death()
    else:
        _character_sprite.modulate = _sprite_tint
        _update_dead_visual()
        _play_idle_animation()

func apply_buff_change(change: Dictionary) -> void:
    var buff_id: String = str(change.get("buff_id", ""))
    var change_type: String = str(change.get("change_type", "add"))
    if buff_id.is_empty():
        return
    match change_type:
        "add", "refresh":
            if not buffs.has(buff_id):
                buffs.append(buff_id)
        "remove":
            buffs.erase(buff_id)

func play_death() -> void:
    if is_dead:
        return
    is_dead = true
    await _play_animation("死亡")
    _update_dead_visual()

func reset_focus() -> void:
    position = base_position
    scale = DEFAULT_UNIT_SCALE
    rotation = 0.0
    _character_sprite.rotation = 0.0
    if not is_dead:
        _play_idle_animation()

func set_planning_highlight(enabled: bool) -> void:
    _ensure_scene_nodes()
    if is_dead:
        return
    if enabled:
        scale = PLANNING_HIGHLIGHT_SCALE
        _character_sprite.modulate = _sprite_tint.lightened(0.15)
    else:
        scale = DEFAULT_UNIT_SCALE
        _character_sprite.modulate = _sprite_tint

func set_target_highlight(enabled: bool, selectable: bool = true) -> void:
    if is_dead:
        return
    if enabled and selectable:
        modulate = Color(1.08, 1.06, 0.92, 1.0)
    elif enabled:
        modulate = Color(0.75, 0.75, 0.75, 0.85)
    else:
        modulate = Color.WHITE
        _character_sprite.modulate = _sprite_tint

func set_selection_arrow(arrow_type: String) -> void:
    _ensure_scene_nodes()
    if _target_arrow != null:
        _target_arrow.visible = arrow_type == SELECTION_ARROW_ENEMY
    if _ally_target_arrow != null:
        _ally_target_arrow.visible = arrow_type == SELECTION_ARROW_ALLY

func set_target_arrow_visible(visible: bool, arrow_type: String = SELECTION_ARROW_ENEMY) -> void:
    if visible:
        set_selection_arrow(arrow_type)
    else:
        set_selection_arrow(SELECTION_ARROW_NONE)

func clear_selection_arrows() -> void:
    set_selection_arrow(SELECTION_ARROW_NONE)

func contains_click_point(point_in_layer: Vector2) -> bool:
    if is_dead:
        return false
    var local_point: Vector2 = point_in_layer - position
    var click_center: Vector2 = Vector2(0.0, -6.0)
    var half_size: Vector2 = Vector2(40.0, 50.0)
    return Rect2(click_center - half_size, half_size * 2.0).has_point(local_point)

func _ensure_target_arrow_texture() -> void:
    if _target_arrow == null:
        return
    if _target_arrow.texture != null:
        return
    if _target_arrow_texture == null:
        _target_arrow_texture = _build_target_arrow_texture()
    _target_arrow.texture = _target_arrow_texture
    _target_arrow.centered = true
    _target_arrow.offset = Vector2(0.0, 0.0)

func _build_target_arrow_texture() -> ImageTexture:
    var width: int = 18
    var height: int = 22
    var image: Image = Image.create(width, height, false, Image.FORMAT_RGBA8)
    image.fill(Color(0.0, 0.0, 0.0, 0.0))
    var tip_y: int = height - 1
    var top_y: int = 2
    var center_x: int = width / 2
    for y: int in range(top_y, tip_y + 1):
        var progress: float = float(y - top_y) / float(max(tip_y - top_y, 1))
        var half_width: float = lerpf(1.0, float(center_x - 1), progress)
        for x: int in range(width):
            if abs(float(x) - float(center_x)) <= half_width:
                image.set_pixel(x, y, Color(1.0, 0.84, 0.2, 1.0))
    return ImageTexture.create_from_image(image)

func _bind_click_area() -> void:
    var click_area: Area2D = get_node_or_null("ClickArea") as Area2D
    if click_area == null:
        return
    if not click_area.input_event.is_connected(_on_click_area_input):
        click_area.input_event.connect(_on_click_area_input)

func _on_click_area_input(_viewport: Node, event: InputEvent, _shape_idx: int) -> void:
    if event is InputEventMouseButton:
        var mouse_event: InputEventMouseButton = event as InputEventMouseButton
        if mouse_event.pressed and mouse_event.button_index == MOUSE_BUTTON_LEFT:
            target_clicked.emit(self)
            get_viewport().set_input_as_handled()

func _ensure_scene_nodes() -> void:
    if _character_sprite == null:
        _character_sprite = get_node_or_null("CharacterSprite") as AnimatedSprite2D
    if _animation_player == null:
        _animation_player = get_node_or_null("AnimationPlayer") as AnimationPlayer
    if _status_root == null:
        _status_root = get_node_or_null("StatusRoot") as Control
    if _hp_bar == null:
        _hp_bar = get_node_or_null("StatusRoot/HpBar") as ProgressBar
    if _target_arrow == null:
        _target_arrow = get_node_or_null("TargetArrow") as Sprite2D
    if _ally_target_arrow == null:
        _ally_target_arrow = get_node_or_null("AllyTargetArrow") as Sprite2D
    _ensure_name_label()


## 确保敌方名称标签存在；标签只展示服务端下发的单位名称，不参与任何战斗计算。
func _ensure_name_label() -> void:
    if _name_label != null:
        return
    _name_label = get_node_or_null("NameLabel") as Label
    if _name_label == null:
        _name_label = Label.new()
        _name_label.name = "NameLabel"
        _name_label.z_index = 5
        _name_label.position = Vector2(-54.0, -56.0)
        _name_label.size = Vector2(108.0, 18.0)
        _name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
        _name_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
        _name_label.autowrap_mode = TextServer.AUTOWRAP_OFF
        _name_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
        _name_label.add_theme_font_size_override("font_size", 12)
        _name_label.add_theme_color_override("font_color", Color(1.0, 0.94, 0.78, 1.0))
        _name_label.add_theme_color_override("font_shadow_color", Color(0.0, 0.0, 0.0, 0.85))
        _name_label.add_theme_constant_override("shadow_offset_x", 1)
        _name_label.add_theme_constant_override("shadow_offset_y", 1)
        add_child(_name_label)


## 刷新敌方名称标签文案与可见性；当前需求只展示右侧敌方单位名称。
func _refresh_name_label() -> void:
    _ensure_name_label()
    if _name_label == null:
        return
    _name_label.text = unit_name
    _name_label.visible = unit_type == "enemy" and not unit_name.is_empty()


func _ensure_chj_renderer() -> void:
    if _chj_renderer != null:
        return
    _chj_renderer = ChjWorldRenderer.new()
    _chj_renderer.name = "ChjSprite2D"
    _chj_renderer.centered = false
    _chj_renderer.visible = false
    _chj_renderer.position = Vector2(0.0, -6.0)
    add_child(_chj_renderer)


func _apply_skin(skin: UnitSkin) -> void:
    _skin = skin
    _uses_chj_render = false
    if _skin == null:
        push_warning("单位缺少 UnitSkin: %d" % actor_id)
        _character_sprite.sprite_frames = _build_fallback_sprite_frames(null)
        _character_sprite.position = Vector2(0.0, -6.0)
        _character_sprite.scale = Vector2(1.0, 1.0)
        _character_sprite.flip_h = false
        _character_sprite.visible = true
        if _chj_renderer != null:
            _chj_renderer.visible = false
        return
    if _skin.uses_chj_world_render():
        _ensure_chj_renderer()
        if _chj_renderer.apply_chj(_skin.resolve_chj_path(), _skin):
            _uses_chj_render = true
            _using_png_override = false
            if _skin.has_configured_sprite_frames():
                _character_sprite.sprite_frames = _skin.sprite_frames
                _character_sprite.position = _skin.sprite_offset
                _character_sprite.scale = _skin.sprite_scale
            _character_sprite.visible = false
            _chj_renderer.visible = true
            _sprite_tint = _skin.tint
            _chj_renderer.modulate = _sprite_tint
            return
    if _chj_renderer != null:
        _chj_renderer.visible = false
    _uses_chj_render = false
    _using_png_override = false
    _character_sprite.visible = true
    if _skin.has_configured_sprite_frames():
        _character_sprite.sprite_frames = _skin.sprite_frames
    else:
        push_warning("UnitSkin 缺少 sprite_frames 与可用 CHJ: %s" % _skin.skin_id)
        _character_sprite.sprite_frames = _build_fallback_sprite_frames(null)
    _character_sprite.position = _skin.sprite_offset
    _character_sprite.scale = _skin.sprite_scale
    _sprite_tint = _skin.tint
    _character_sprite.modulate = _sprite_tint
    _align_character_sprite_to_ground()
    _apply_sprite_flip_for_animation(_skin.default_animation if not _skin.default_animation.is_empty() else "待机")

func _align_character_sprite_to_ground() -> void:
    if _character_sprite == null or _character_sprite.sprite_frames == null:
        return
    var animation_name: String = "待机"
    if _skin != null and not _skin.default_animation.is_empty():
        animation_name = _skin.default_animation
    if not _character_sprite.sprite_frames.has_animation(animation_name):
        return
    var texture: Texture2D = _character_sprite.sprite_frames.get_frame_texture(animation_name, 0)
    if texture == null:
        return
    var manual_offset: Vector2 = _skin.sprite_offset if _skin != null else Vector2.ZERO
    var frame_height: float = float(texture.get_height()) * absf(_character_sprite.scale.y)
    if _character_sprite.centered:
        _character_sprite.position = Vector2(
            manual_offset.x,
            -frame_height * 0.5 + manual_offset.y
        )
    else:
        _character_sprite.position = Vector2(
            manual_offset.x,
            -frame_height + manual_offset.y
        )

func _build_fallback_sprite_frames(texture: Texture2D) -> SpriteFrames:
    var frames: SpriteFrames = SpriteFrames.new()
    for animation_name in DEFAULT_ANIMATIONS:
        frames.add_animation(animation_name)
        frames.set_animation_speed(animation_name, 8.0)
        if texture != null:
            frames.add_frame(animation_name, texture)
    return frames

func _set_hp_bar_visible(visible: bool) -> void:
    _ensure_scene_nodes()
    if _status_root != null:
        _status_root.visible = visible
    if _hp_bar != null:
        _hp_bar.visible = visible

## 按事件 value 推算受击后的 HP，保证血条能按段落下动画。
func _resolve_target_hp_after(result: Dictionary, result_type: String, previous_hp: int) -> int:
    match result_type:
        "damage":
            if result.has("hp_after"):
                return clampi(int(result.get("hp_after", previous_hp)), 0, max_hp)
            return maxi(0, previous_hp - int(result.get("value", 0)))
        "heal":
            return mini(max_hp, previous_hp + int(result.get("value", 0)))
        "defeat":
            return 0
        _:
            return clampi(int(result.get("hp_after", previous_hp)), 0, max_hp)

## 用服务端 actors 快照校准本地演出 HP。
func apply_runtime_snapshot(runtime_actor: Dictionary) -> void:
    max_hp = max(1, int(runtime_actor.get("hp_max", max_hp)))
    current_hp = clamp(int(runtime_actor.get("hp", current_hp)), 0, max_hp)
    is_dead = bool(runtime_actor.get("dead", current_hp <= 0))
    _refresh_hp_display()
    _set_hp_bar_visible(false)
    _update_dead_visual()

## 受击或治疗时显示血条，并取消尚未执行的隐藏定时。
func _show_hp_bar_for_hit() -> void:
    _hp_bar_hide_token += 1
    _set_hp_bar_visible(true)

## 血条动画结束后延迟隐藏，平时不常驻显示。
func _schedule_hp_bar_hide() -> void:
    var token: int = _hp_bar_hide_token
    await get_tree().create_timer(HP_BAR_HIDE_DELAY_AFTER_ANIM_SEC).timeout
    if token != _hp_bar_hide_token:
        return
    _set_hp_bar_visible(false)

func _refresh_hp_display() -> void:
    _ensure_scene_nodes()
    if _hp_bar == null:
        return
    _hp_bar.max_value = max_hp
    _hp_bar.value = current_hp

func _prepare_hp_bar(display_hp: int) -> void:
    _ensure_scene_nodes()
    if _hp_bar == null:
        return
    if _hp_bar_tween != null and _hp_bar_tween.is_valid():
        _hp_bar_tween.kill()
    _hp_bar.max_value = max_hp
    _hp_bar.value = clamp(display_hp, 0, max_hp)

func _animate_hp_bar_to(target_hp: int, duration: float) -> void:
    _ensure_scene_nodes()
    if _hp_bar == null:
        return
    if _hp_bar_tween != null and _hp_bar_tween.is_valid():
        _hp_bar_tween.kill()
    var clamped_target: float = clamp(float(target_hp), 0.0, float(max_hp))
    if duration <= 0.0 or is_equal_approx(_hp_bar.value, clamped_target):
        _hp_bar.value = clamped_target
        return
    _hp_bar_tween = create_tween()
    _hp_bar_tween.tween_property(
        _hp_bar,
        "value",
        clamped_target,
        duration
    ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_OUT)
    await _hp_bar_tween.finished

func _update_dead_visual() -> void:
    if current_hp <= 0 or is_dead:
        is_dead = true
        _character_sprite.modulate = _sprite_tint.darkened(0.45)
        _status_root.modulate = Color(1, 1, 1, 0.82)
    else:
        modulate = Color.WHITE
        scale = DEFAULT_UNIT_SCALE
        _character_sprite.modulate = _sprite_tint
        _status_root.modulate = Color.WHITE

func _resolve_action_animation(skill_visual: SkillVisualConfig, fallback_skill_id: String, fallback_skill_name: String, action_type: String, animation_hint: String = "") -> String:
    var candidates: Array[String] = []
    var normalized_animation_hint: String = _normalize_animation_key(animation_hint)
    var normalized_skill_id: String = _normalize_animation_key(fallback_skill_id)
    var normalized_skill_name: String = _normalize_animation_key(fallback_skill_name)
    var normalized_action_type: String = _normalize_animation_key(action_type)
    if skill_visual != null and not skill_visual.animation_key.is_empty():
        candidates.append(_normalize_animation_key(skill_visual.animation_key))
    if not normalized_animation_hint.is_empty():
        candidates.append(normalized_animation_hint)
    if not normalized_skill_id.is_empty():
        candidates.append(normalized_skill_id)
    if not normalized_skill_name.is_empty():
        candidates.append(normalized_skill_name)
    if not normalized_action_type.is_empty() and ACTION_TYPE_ANIMATIONS.has(normalized_action_type):
        candidates.append(ACTION_TYPE_ANIMATIONS[normalized_action_type])
    candidates.append("普攻")

    for candidate in candidates:
        if _skin != null and _skin.animation_map.has(candidate):
            var mapped_name: String = str(_skin.animation_map[candidate])
            if not _uses_chj_render or _skin.has_animation(mapped_name):
                return mapped_name
        if _uses_chj_render:
            if _skin != null:
                var png_override: String = _skin.get_battle_action_png_override(candidate)
                if not png_override.is_empty():
                    return png_override
            continue
        if _character_sprite.sprite_frames != null and _character_sprite.sprite_frames.has_animation(candidate):
            return candidate
        if _animation_player.has_animation(candidate):
            return candidate
    if not normalized_action_type.is_empty() and ACTION_TYPE_ANIMATIONS.has(normalized_action_type):
        return ACTION_TYPE_ANIMATIONS[normalized_action_type]
    return "普攻"

func _normalize_animation_key(raw_name: String) -> String:
    var key: String = raw_name.strip_edges().to_lower()
    key = key.replace(" ", "_")
    key = key.replace("-", "_")
    return key

func _apply_sprite_flip_for_animation(animation_name: String) -> void:
    if _character_sprite == null:
        return
    if _skin != null and _skin.animation_flip_h.has(animation_name):
        _character_sprite.flip_h = bool(_skin.animation_flip_h[animation_name])


func _show_battle_png_override(animation_name: String) -> void:
    if _character_sprite == null or _skin == null or not _skin.has_animation(animation_name):
        return
    _using_png_override = true
    if _chj_renderer != null:
        _chj_renderer.visible = false
    _character_sprite.visible = true
    _apply_sprite_flip_for_animation(animation_name)
    _align_character_sprite_to_ground_for_animation(animation_name)
    _character_sprite.play(animation_name)


func _hide_battle_png_override() -> void:
    _using_png_override = false
    if not _uses_chj_render or _chj_renderer == null:
        return
    _character_sprite.visible = false
    _chj_renderer.visible = true


func _align_character_sprite_to_ground_for_animation(animation_name: String) -> void:
    if _character_sprite == null or _character_sprite.sprite_frames == null:
        return
    if not _character_sprite.sprite_frames.has_animation(animation_name):
        return
    var frame_texture: Texture2D = _character_sprite.sprite_frames.get_frame_texture(animation_name, 0)
    if frame_texture == null:
        return
    var manual_offset: Vector2 = _skin.sprite_offset if _skin != null else Vector2.ZERO
    var frame_height: float = float(frame_texture.get_height()) * absf(_character_sprite.scale.y)
    if _character_sprite.centered:
        _character_sprite.position = Vector2(
            manual_offset.x,
            -frame_height * 0.5 + manual_offset.y
        )
    else:
        _character_sprite.position = Vector2(
            manual_offset.x,
            -frame_height + manual_offset.y
        )


func _play_animation(animation_name: String) -> void:
    if _uses_chj_render and _chj_renderer != null and _skin != null:
        var png_override: String = _skin.get_battle_action_png_override(animation_name)
        if not png_override.is_empty():
            _show_battle_png_override(png_override)
            var animation_speed: float = _character_sprite.sprite_frames.get_animation_speed(png_override)
            var frame_count: int = _character_sprite.sprite_frames.get_frame_count(png_override)
            if frame_count <= 1 or animation_speed <= 0.0:
                await get_tree().process_frame
            else:
                await get_tree().create_timer(float(frame_count) / animation_speed).timeout
            _play_idle_animation()
            return
        if _should_use_chj_skill_animation(animation_name):
            var played_chj_skill: bool = await _play_chj_skill_animation()
            if played_chj_skill:
                return
    _apply_sprite_flip_for_animation(animation_name)
    var played_sprite_animation: bool = false
    if _character_sprite.sprite_frames != null and _character_sprite.sprite_frames.has_animation(animation_name):
        _character_sprite.play(animation_name)
        played_sprite_animation = true
    if _animation_player.has_animation(animation_name):
        _animation_player.play(animation_name)
        await _animation_player.animation_finished
        return
    if played_sprite_animation:
        var animation_speed: float = _character_sprite.sprite_frames.get_animation_speed(animation_name)
        var frame_count: int = _character_sprite.sprite_frames.get_frame_count(animation_name)
        if frame_count <= 1 or animation_speed <= 0.0:
            await get_tree().process_frame
            return
        await get_tree().create_timer(float(frame_count) / animation_speed).timeout

func _play_idle_animation() -> void:
    if is_dead:
        return
    if _uses_chj_render and _chj_renderer != null and _skin != null:
        var png_override: String = _skin.get_battle_idle_png_override()
        if not png_override.is_empty():
            _show_battle_png_override(png_override)
            return
        _hide_battle_png_override()
        _chj_renderer.set_world_pose("battle", "down")
        return
    var idle_animation_name: String = "待机"
    if _skin != null and not _skin.default_animation.is_empty():
        idle_animation_name = _skin.default_animation
    _apply_sprite_flip_for_animation(idle_animation_name)
    if _character_sprite.sprite_frames != null and _character_sprite.sprite_frames.has_animation(idle_animation_name):
        _character_sprite.play(idle_animation_name)
    if _animation_player.has_animation(idle_animation_name):
        _animation_player.play(idle_animation_name)
    elif _animation_player.has_animation("待机"):
        _animation_player.play("待机")


## 判断当前动作是否应使用 chj_skill_path 播放（排除 PNG 覆盖与受击/死亡等反馈动画）。
func _should_use_chj_skill_animation(animation_name: String) -> bool:
    if animation_name.is_empty() or _skin == null:
        return false
    if _skin.resolve_chj_skill_path().is_empty():
        return false
    if animation_name in CHJ_REACTION_ANIMATIONS:
        return false
    var idle_png: String = _skin.get_battle_idle_png_override()
    if not idle_png.is_empty() and animation_name == idle_png:
        return false
    if not _skin.get_battle_action_png_override(animation_name).is_empty():
        return false
    if animation_name in CHJ_SKILL_ANIMATIONS:
        return true
    # 自定义 animation_key（如「裂空斩」）在 CHJ 模式下也走技能 CHJ，除非上面已命中 PNG 覆盖。
    return true


## 播放 chj_skill_path 技能动画；先切回 CHJ 渲染，并带超时兜底避免无限 await。
func _play_chj_skill_animation() -> bool:
    var skill_path: String = _skin.resolve_chj_skill_path()
    if skill_path.is_empty():
        return false
    _hide_battle_png_override()
    if not _chj_renderer.start_skill_animation(skill_path, 0):
        return false
    var wait_sec: float = clampf(
        _chj_renderer.estimate_skill_playback_seconds() * 1.25,
        0.2,
        CHJ_SKILL_MAX_WAIT_SEC
    )
    var completed: bool = false
    var on_finished: Callable = func() -> void:
        completed = true
    _chj_renderer.skill_animation_finished.connect(on_finished, CONNECT_ONE_SHOT)
    var elapsed: float = 0.0
    while not completed and elapsed < wait_sec:
        await get_tree().process_frame
        elapsed += get_process_delta_time()
    if _chj_renderer.is_skill_playing():
        _chj_renderer.cancel_skill_animation()
    _play_idle_animation()
    return true

func _ensure_builtin_animations() -> void:
    _ensure_animation("待机", _build_idle_animation())
    _ensure_animation("普攻", _build_attack_animation())
    _ensure_animation("技能", _build_attack_animation())
    _ensure_animation("自动", _build_attack_animation())
    _ensure_animation("物品", _build_support_animation(Color("#78d98e")))
    _ensure_animation("逃跑", _build_escape_animation())
    _ensure_animation("追击", _build_attack_animation())
    _ensure_animation("反击", _build_attack_animation())
    _ensure_animation("持续伤害", _build_support_animation(Color("#ffb36b")))
    _ensure_animation("受击", _build_hit_animation(Color("#ff8d7d"), Vector2(1.06, 0.94)))
    _ensure_animation("治疗", _build_hit_animation(Color("#78d98e"), Vector2(1.04, 1.04)))
    _ensure_animation("死亡", _build_death_animation())

func _ensure_animation(animation_name: String, animation: Animation) -> void:
    if _animation_player.has_animation(animation_name):
        return
    _get_default_animation_library().add_animation(animation_name, animation)

func _get_default_animation_library() -> AnimationLibrary:
    if not _animation_player.has_animation_library(""):
        _animation_player.add_animation_library("", AnimationLibrary.new())
    return _animation_player.get_animation_library("")

func _build_idle_animation() -> Animation:
    var animation: Animation = Animation.new()
    animation.length = 0.8
    animation.loop_mode = Animation.LOOP_LINEAR
    var sprite_scale_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(sprite_scale_track, NodePath("CharacterSprite:scale"))
    animation.track_insert_key(sprite_scale_track, 0.0, DEFAULT_UNIT_SCALE)
    animation.track_insert_key(sprite_scale_track, 0.4, DEFAULT_UNIT_SCALE * 1.03)
    animation.track_insert_key(sprite_scale_track, 0.8, DEFAULT_UNIT_SCALE)
    return animation

func _build_attack_animation() -> Animation:
    var animation: Animation = Animation.new()
    animation.length = 0.24
    var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(scale_track, NodePath(":scale"))
    animation.track_insert_key(scale_track, 0.0, DEFAULT_UNIT_SCALE)
    animation.track_insert_key(scale_track, 0.12, DEFAULT_UNIT_SCALE * 1.06)
    animation.track_insert_key(scale_track, 0.24, DEFAULT_UNIT_SCALE)
    var rotation_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(rotation_track, NodePath("CharacterSprite:rotation"))
    animation.track_insert_key(rotation_track, 0.0, 0.0)
    animation.track_insert_key(rotation_track, 0.12, 0.12)
    animation.track_insert_key(rotation_track, 0.24, 0.0)
    return animation

func _build_support_animation(flash_color: Color) -> Animation:
    var animation: Animation = Animation.new()
    animation.length = 0.26
    var color_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(color_track, NodePath("CharacterSprite:modulate"))
    animation.track_insert_key(color_track, 0.0, Color.WHITE)
    animation.track_insert_key(color_track, 0.13, flash_color)
    animation.track_insert_key(color_track, 0.26, Color.WHITE)
    return animation

func _build_escape_animation() -> Animation:
    var animation: Animation = Animation.new()
    animation.length = 0.24
    var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(scale_track, NodePath(":scale"))
    animation.track_insert_key(scale_track, 0.0, DEFAULT_UNIT_SCALE)
    animation.track_insert_key(
        scale_track,
        0.12,
        Vector2(DEFAULT_UNIT_SCALE.x * 0.94, DEFAULT_UNIT_SCALE.y * 1.08)
    )
    animation.track_insert_key(scale_track, 0.24, DEFAULT_UNIT_SCALE)
    var rotation_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(rotation_track, NodePath("CharacterSprite:rotation"))
    animation.track_insert_key(rotation_track, 0.0, 0.0)
    animation.track_insert_key(rotation_track, 0.12, -0.12)
    animation.track_insert_key(rotation_track, 0.24, 0.0)
    return animation

func _build_hit_animation(flash_color: Color, scale_target: Vector2) -> Animation:
    var animation: Animation = Animation.new()
    animation.length = 0.2
    var color_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(color_track, NodePath("CharacterSprite:modulate"))
    animation.track_insert_key(color_track, 0.0, Color.WHITE)
    animation.track_insert_key(color_track, 0.08, flash_color)
    animation.track_insert_key(color_track, 0.2, Color.WHITE)
    var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(scale_track, NodePath(":scale"))
    animation.track_insert_key(scale_track, 0.0, DEFAULT_UNIT_SCALE)
    animation.track_insert_key(
        scale_track,
        0.08,
        Vector2(DEFAULT_UNIT_SCALE.x * scale_target.x, DEFAULT_UNIT_SCALE.y * scale_target.y)
    )
    animation.track_insert_key(scale_track, 0.2, DEFAULT_UNIT_SCALE)
    return animation

func _build_death_animation() -> Animation:
    var animation: Animation = Animation.new()
    animation.length = 0.35
    var alpha_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(alpha_track, NodePath(":modulate:a"))
    animation.track_insert_key(alpha_track, 0.0, 1.0)
    animation.track_insert_key(alpha_track, 0.35, 0.32)
    var scale_track: int = animation.add_track(Animation.TYPE_VALUE)
    animation.track_set_path(scale_track, NodePath(":scale"))
    animation.track_insert_key(scale_track, 0.0, DEFAULT_UNIT_SCALE)
    animation.track_insert_key(scale_track, 0.35, DEFAULT_UNIT_SCALE * 0.9)
    return animation
