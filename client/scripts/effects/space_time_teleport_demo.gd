extends Node2D

## 自动循环时两轮传送特效之间的停顿秒数。
@export_range(0.2, 5.0, 0.1) var replay_delay_seconds: float = 1.0

## Demo 中围绕人物播放的可复用传送特效实例。
@onready var _teleport_effect: SpaceTimeTeleportEffect = $Stage/SpaceTimeTeleportEffect
## Demo 中用于展示传送前后显隐变化的人物节点。
@onready var _player: CharacterBody2D = $Stage/Player
## 控制自动循环等待时间的场景节点。
@onready var _replay_timer: Timer = $ReplayTimer

## 绑定循环信号并在场景就绪后的下一帧自动播放第一轮特效。
func _ready() -> void:
    _replay_timer.wait_time = replay_delay_seconds
    _teleport_effect.effect_finished.connect(_on_effect_finished)
    _teleport_effect.vanish_started.connect(_on_vanish_started)
    _replay_timer.timeout.connect(_play_demo_effect)
    call_deferred("_play_demo_effect")

## 支持桌面端空格、确认键、鼠标点击以及移动端触摸随时从头重播。
## event 是 Godot 分发的未被其他节点消费的输入事件。
func _unhandled_input(event: InputEvent) -> void:
    var should_replay: bool = false
    if event is InputEventKey:
        var key_event: InputEventKey = event as InputEventKey
        should_replay = key_event.pressed and not key_event.echo and (
            key_event.keycode == KEY_SPACE or key_event.is_action_pressed("ui_accept")
        )
    elif event is InputEventMouseButton:
        var mouse_event: InputEventMouseButton = event as InputEventMouseButton
        should_replay = mouse_event.pressed and mouse_event.button_index == MOUSE_BUTTON_LEFT
    elif event is InputEventScreenTouch:
        var touch_event: InputEventScreenTouch = event as InputEventScreenTouch
        should_replay = touch_event.pressed

    if not should_replay:
        return
    _replay_timer.stop()
    _play_demo_effect()
    get_viewport().set_input_as_handled()

## 播放完成后启动一次性计时器，形成便于观察浓度变化的循环 Demo。
func _on_effect_finished() -> void:
    _replay_timer.start()

## 进入传送后半段时隐藏人物，只保留快速消散的特效和十字光点。
func _on_vanish_started() -> void:
    _player.visible = false

## 停止循环等待并从头播放一轮传送特效。
func _play_demo_effect() -> void:
    _replay_timer.stop()
    _player.visible = true
    _teleport_effect.play_effect()
