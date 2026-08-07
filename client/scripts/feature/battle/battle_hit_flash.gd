extends Node
class_name BattleHitFlash

## 受击闪烁从全白恢复到原色所需的固定时间，单位为秒。
const FLASH_RECOVER_DURATION_SEC: float = 0.2

## 受击闪烁使用的画布 Shader；只改变精灵颜色，不参与任何战斗数值计算。
const HIT_FLASH_SHADER: Shader = preload("res://scenes/battle/battle_hit_flash.gdshader")

## 受击时沿远离战场中心方向后退的距离，单位为战斗场景世界像素。
const HIT_RECOIL_DISTANCE: float = 12.0

## 从权威站位快速后退到受击位置所需时间，单位为秒。
const HIT_RECOIL_DURATION_SEC: float = 0.08

## 从受击位置平滑返回权威站位所需时间，单位为秒。
const HIT_RETURN_DURATION_SEC: float = 0.12

## Shader 中控制白色混合比例的统一参数名。
const FLASH_AMOUNT_PARAMETER: StringName = &"flash_amount"

## 上一帧记录的服务端权威生命值；小于零表示尚未取得首次战斗快照。
var _last_hp: int = -1

## 当前受击闪烁渐变；连续受击时会终止旧渐变并从全白重新开始。
var _flash_tween: Tween = null

## 当前受击后退渐变；连续受击时会终止旧位移并重新播放，防止偏移累计。
var _recoil_tween: Tween = null

## 普通 PNG/图集动画精灵的独立闪烁材质，避免多个战斗单位共享 Shader 参数。
var _character_flash_material: ShaderMaterial = null

## CHJ 动画精灵的独立闪烁材质，用于覆盖人物使用 CHJ 渲染的情况。
var _chj_flash_material: ShaderMaterial = null


## 每帧比较一次父级战斗单位的权威生命值；生命值下降时触发受击闪白。
## `_delta` 是当前帧耗时，本效果只比较整数生命值，因此不直接使用该参数。
func _process(_delta: float) -> void:
    var battle_unit: BattleUnit = get_parent() as BattleUnit
    if battle_unit == null:
        return

    var current_hp: int = battle_unit.current_hp
    if _last_hp < 0:
        _last_hp = current_hp
        return

    if current_hp < _last_hp:
        _play_hit_flash()
    _last_hp = current_hp


## 播放当前单位的受击视觉反馈：先触发后退归位，再让实际显示的精灵闪白并恢复原色。
func _play_hit_flash() -> void:
    _play_hit_recoil()
    _ensure_flash_materials()
    if _character_flash_material == null and _chj_flash_material == null:
        return

    if _flash_tween != null and _flash_tween.is_valid():
        _flash_tween.kill()

    _set_flash_amount(1.0)
    _flash_tween = create_tween()
    _flash_tween.tween_method(
        Callable(self, "_set_flash_amount"),
        1.0,
        0.0,
        FLASH_RECOVER_DURATION_SEC
    ).set_trans(Tween.TRANS_LINEAR).set_ease(Tween.EASE_IN_OUT)

## 播放远离战场中心的短距离后退，并返回服务端权威站位。
## 己方人物和宠物位于左侧，因此向左后退；敌方位于右侧，因此向右后退。
func _play_hit_recoil() -> void:
    var battle_unit: BattleUnit = get_parent() as BattleUnit
    if battle_unit == null:
        return

    if _recoil_tween != null and _recoil_tween.is_valid():
        _recoil_tween.kill()

    var recoil_direction: float = 1.0
    if battle_unit.unit_type != "enemy":
        recoil_direction = -1.0

    var recoil_position: Vector2 = battle_unit.base_position + Vector2(
        recoil_direction * HIT_RECOIL_DISTANCE,
        0.0
    )
    _recoil_tween = create_tween()
    _recoil_tween.tween_property(
        battle_unit,
        "position",
        recoil_position,
        HIT_RECOIL_DURATION_SEC
    ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_OUT)
    _recoil_tween.tween_property(
        battle_unit,
        "position",
        battle_unit.base_position,
        HIT_RETURN_DURATION_SEC
    ).set_trans(Tween.TRANS_QUAD).set_ease(Tween.EASE_IN)


## 为已经存在的精灵节点创建单位私有材质；CHJ 节点可能在皮肤初始化阶段动态加入，因此在每次受击前补查。
func _ensure_flash_materials() -> void:
    if _character_flash_material == null:
        var character_sprite: AnimatedSprite2D = get_parent().get_node_or_null("CharacterSprite") as AnimatedSprite2D
        if character_sprite != null:
            _character_flash_material = _create_flash_material()
            character_sprite.material = _character_flash_material

    if _chj_flash_material == null:
        var chj_sprite: Sprite2D = get_parent().get_node_or_null("ChjSprite2D") as Sprite2D
        if chj_sprite != null:
            _chj_flash_material = _create_flash_material()
            chj_sprite.material = _chj_flash_material


## 创建仅供一个精灵节点使用的闪烁材质，确保一方受击不会让另一方同步变白。
## 返回值是已绑定受击 Shader 且初始闪烁强度为零的材质。
func _create_flash_material() -> ShaderMaterial:
    var flash_material: ShaderMaterial = ShaderMaterial.new()
    flash_material.shader = HIT_FLASH_SHADER
    flash_material.set_shader_parameter(FLASH_AMOUNT_PARAMETER, 0.0)
    return flash_material


## 同步设置普通精灵和 CHJ 精灵的白色混合比例。
## `amount` 的有效范围为 0.0 到 1.0，方法内部会再次限制范围以避免异常颜色值。
func _set_flash_amount(amount: float) -> void:
    var clamped_amount: float = clampf(amount, 0.0, 1.0)
    if _character_flash_material != null:
        _character_flash_material.set_shader_parameter(FLASH_AMOUNT_PARAMETER, clamped_amount)
    if _chj_flash_material != null:
        _chj_flash_material.set_shader_parameter(FLASH_AMOUNT_PARAMETER, clamped_amount)
