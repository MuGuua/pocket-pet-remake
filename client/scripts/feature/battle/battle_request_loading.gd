class_name BattleRequestLoading
extends RuntimeProgressOverlay

## 战斗场景请求 Loading 兼容壳；新代码请直接使用 RuntimeProgressOverlay。


## 开始等待服务端回包。
func show_waiting(tip_text: String = "", skip_delay: bool = false) -> void:
    _ensure_loading()
    if _loading != null:
        _loading.show_waiting(tip_text, skip_delay)


## 关闭等待遮罩。
func hide_waiting() -> void:
    hide_loading()
