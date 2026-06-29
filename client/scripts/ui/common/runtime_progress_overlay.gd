class_name RuntimeProgressOverlay
extends Node

## 兼容旧引用路径；实际展示统一委托给 GenericLoadingScene。
const SCENE_PATH: String = "res://scenes/ui/common/runtime_progress_overlay.tscn"

var _loading: GenericLoadingScene = null


func _ready() -> void:
    _ensure_loading()


## 开始等待服务端回包。
func show_waiting(_tip_text: String = "", skip_delay: bool = false) -> void:
    _ensure_loading()
    if _loading != null:
        _loading.show_waiting(_tip_text, skip_delay)


## 立即展示 Loading。
func show_loading(_tip_text: String = "") -> void:
    _ensure_loading()
    if _loading != null:
        _loading.show_loading()


## 播放固定时长 Loading 动画。
func play_progress(duration_sec: float = GenericLoadingScene.DEFAULT_PROGRESS_DURATION_SEC, _status_text: String = "") -> void:
    _ensure_loading()
    if _loading != null:
        await _loading.play_progress(duration_sec)


func hide_overlay() -> void:
    hide_loading()


func hide_progress() -> void:
    hide_loading()


func hide_loading() -> void:
    if _loading != null:
        _loading.hide_loading()


func hide_waiting() -> void:
    hide_loading()


func _ensure_loading() -> void:
    if _loading != null:
        return
    var loading_scene: PackedScene = preload(GenericLoadingScene.SCENE_PATH)
    _loading = loading_scene.instantiate() as GenericLoadingScene
    if _loading == null:
        push_warning("RuntimeProgressOverlay: GenericLoadingScene 实例化失败。")
        return
    add_child(_loading)
