class_name SpaceTimeTeleportEffect
extends Node2D

## 一轮传送表现完整播放并恢复隐藏状态后发出。
signal effect_finished
## 聚能完成并进入传送后半段时发出，调用方应在此时隐藏被传送人物。
signal vanish_started

## 人物消失并切换到十字光点收尾阶段的归一化时间点。
const VANISH_PROGRESS: float = 0.70

## 从透明聚能到完全消散的总时长，单位为秒。
@export_range(0.8, 6.0, 0.1) var duration_seconds: float = 2.8

## 当前特效播放进度；Setter 会把同一时间轴同步给光柱、能量环和爆闪 Shader。
var _effect_progress: float = 0.0:
    set(value):
        _effect_progress = clampf(value, 0.0, 1.0)
        if is_node_ready():
            _apply_effect_progress()

## 当前正在驱动特效时间轴的 Tween；重播前会先终止旧时间轴。
var _playback_tween: Tween = null
## 播放代次用于阻止旧 Tween 的完成回调关闭新一轮特效。
var _play_generation: int = 0
## 记录当前播放是否已经发出人物消失事件，避免逐帧重复通知调用方。
var _vanish_emitted: bool = false

## 外层柔光光柱使用的独立 ShaderMaterial 实例。
@onready var _beam_halo_material: ShaderMaterial = $BeamHalo.material as ShaderMaterial
## 内层高亮光柱使用的独立 ShaderMaterial 实例。
@onready var _beam_core_material: ShaderMaterial = $BeamCore.material as ShaderMaterial
## 绘制在人物精灵后方的传送阵后半环材质。
@onready var _ground_ring_back_material: ShaderMaterial = $GroundRingBack.material as ShaderMaterial
## 绘制在人物精灵前方的传送阵前半环材质。
@onready var _ground_ring_front_material: ShaderMaterial = $GroundRingFront.material as ShaderMaterial
## 人物周围瞬时聚能闪光使用的独立 ShaderMaterial 实例。
@onready var _focus_flash_material: ShaderMaterial = $FocusFlash.material as ShaderMaterial
## 人物消失后负责收束并淡出的十字光点材质。
@onready var _cross_flash_material: ShaderMaterial = $CrossFlash.material as ShaderMaterial
## 从人物周围向上升起的短粒子。
@onready var _spark_particles: GPUParticles2D = $SparkParticles
## 贯穿光柱的细长高速粒子。
@onready var _streak_particles: GPUParticles2D = $StreakParticles

## 初始化时保持运行时特效隐藏，必须通过 play_effect() 正式播放。
func _ready() -> void:
    visible = false
    _effect_progress = 0.0
    _apply_effect_progress()

## 从头播放一次蓝色时空传送表现；重复调用会安全替换仍在进行的上一轮播放。
func play_effect() -> void:
    _play_generation += 1
    var generation: int = _play_generation
    if _playback_tween != null and _playback_tween.is_valid():
        _playback_tween.kill()

    visible = true
    _vanish_emitted = false
    _effect_progress = 0.0
    _restart_particles()

    _playback_tween = create_tween()
    _playback_tween.set_trans(Tween.TRANS_SINE)
    _playback_tween.set_ease(Tween.EASE_IN_OUT)
    _playback_tween.tween_property(self, "_effect_progress", 1.0, duration_seconds)
    _playback_tween.finished.connect(_finish_effect.bind(generation))

## 立即停止当前播放并隐藏全部表现，供场景切换中断时清理残留特效。
func stop_effect() -> void:
    _play_generation += 1
    if _playback_tween != null and _playback_tween.is_valid():
        _playback_tween.kill()
    _playback_tween = null
    _spark_particles.emitting = false
    _streak_particles.emitting = false
    _vanish_emitted = false
    _effect_progress = 0.0
    visible = false

## 将当前统一进度写入所有 Shader，并让粒子在消散阶段提前停止发射以自然飞离。
func _apply_effect_progress() -> void:
    _set_material_progress(_beam_halo_material, _effect_progress)
    _set_material_progress(_beam_core_material, _effect_progress)
    _set_material_progress(_ground_ring_back_material, _effect_progress)
    _set_material_progress(_ground_ring_front_material, _effect_progress)
    _set_material_progress(_focus_flash_material, _effect_progress)
    _set_material_progress(_cross_flash_material, _effect_progress)

    if not _vanish_emitted and _effect_progress >= VANISH_PROGRESS:
        _vanish_emitted = true
        vanish_started.emit()

    var particle_alpha: float = activation_envelope(_effect_progress)
    _spark_particles.modulate = Color(0.35, 0.78, 1.0, particle_alpha)
    _streak_particles.modulate = Color(0.50, 0.86, 1.0, particle_alpha)
    if _effect_progress >= VANISH_PROGRESS:
        _spark_particles.emitting = false
        _streak_particles.emitting = false

## 写入一个 ShaderMaterial 的播放进度；材质缺失时安全跳过，避免 Demo 中断。
## target_material 是目标 Shader 材质，value 是 0 到 1 的归一化时间轴进度。
func _set_material_progress(target_material: ShaderMaterial, value: float) -> void:
    if target_material == null:
        return
    target_material.set_shader_parameter("progress", value)

## 重新启动两层粒子，使连续点击重播时不会继承上一轮仍在场景中的粒子状态。
func _restart_particles() -> void:
    _spark_particles.emitting = true
    _streak_particles.emitting = true
    _spark_particles.restart()
    _streak_particles.restart()

## 计算与 Shader 相同的淡入淡出包络，专门用于同步粒子透明度。
## value 是 0 到 1 的播放进度，返回值是 0 到 1 的当前可见强度。
func activation_envelope(value: float) -> float:
    var build: float = smoothstep(0.0, 0.34, value)
    var fade: float = 1.0 - smoothstep(0.70, 0.82, value)
    return build * fade

## 只允许当前播放代次完成收尾，随后隐藏节点并通知 Demo 或正式流程继续。
## generation 是创建 Tween 时记录的播放代次。
func _finish_effect(generation: int) -> void:
    if generation != _play_generation:
        return
    _playback_tween = null
    _spark_particles.emitting = false
    _streak_particles.emitting = false
    visible = false
    effect_finished.emit()
