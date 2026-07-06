extends Node


## Web 运行时统一使用的设计宽度，用于计算目标纵横比。
const TARGET_CANVAS_WIDTH: float = 780.0
## Web 运行时统一使用的设计高度，用于计算目标纵横比。
const TARGET_CANVAS_HEIGHT: float = 1440.0
## 目标画布纵横比；浏览器内允许同比例缩放，但不允许拉伸变形。
const TARGET_CANVAS_ASPECT: float = TARGET_CANVAS_WIDTH / TARGET_CANVAS_HEIGHT

## 只打印一次尺寸诊断日志，避免浏览器尺寸变化时反复刷屏。
var _metrics_logged: bool = false


## 自动加载后立刻接管 Web 画布尺寸，并监听根视口尺寸变化。
func _ready() -> void:
    if not OS.has_feature("web"):
        return
    _apply_canvas_constraints()
    var root_viewport: Viewport = get_viewport()
    if root_viewport != null and not root_viewport.size_changed.is_connected(_on_root_viewport_size_changed):
        root_viewport.size_changed.connect(_on_root_viewport_size_changed)


## 根视口变化后再次把临时调试页 DOM 尺寸拉回项目设计尺寸。
func _on_root_viewport_size_changed() -> void:
    _apply_canvas_constraints()


## 统一约束浏览器中的 html、body、canvas 与父容器尺寸，覆盖临时调试页默认自适应行为。
## 这里不再锁死 780x1440 像素，只锁定 13:24 纵横比并按浏览器可视区域同比例缩放。
func _apply_canvas_constraints() -> void:
    var script: String = """
    (function () {
        const aspect = %f;
        const root = document.documentElement;
        const body = document.body;
        const canvas = document.getElementById('canvas') || document.querySelector('canvas');
        const status = document.getElementById('status');
        const viewportWidth = window.innerWidth || 0;
        const viewportHeight = window.innerHeight || 0;
        let width = viewportWidth;
        let height = 0;
        if (viewportWidth > 0 && viewportHeight > 0) {
            if (viewportWidth / viewportHeight > aspect) {
                height = viewportHeight;
                width = Math.floor(height * aspect);
            } else {
                width = viewportWidth;
                height = Math.floor(width / aspect);
            }
        }
        if (root) {
            root.style.margin = '0';
            root.style.padding = '0';
            root.style.background = '#000';
            root.style.overflow = 'hidden';
            root.style.width = '100%%';
            root.style.height = '100%%';
        }
        if (body) {
            body.style.margin = '0';
            body.style.padding = '0';
            body.style.width = '100vw';
            body.style.height = '100vh';
            body.style.background = '#000';
            body.style.overflow = 'hidden';
            body.style.display = 'grid';
            body.style.placeItems = 'center';
        }
        if (canvas) {
            canvas.style.display = 'block';
            canvas.style.width = width + 'px';
            canvas.style.height = height + 'px';
            canvas.style.minWidth = '0px';
            canvas.style.minHeight = '0px';
            canvas.style.maxWidth = width + 'px';
            canvas.style.maxHeight = height + 'px';
            canvas.style.aspectRatio = String(aspect);
            canvas.style.background = '#000';
        }
        if (status) {
            status.style.width = width + 'px';
            status.style.minWidth = '0px';
            status.style.maxWidth = width + 'px';
        }
        if (canvas && canvas.parentElement) {
            canvas.parentElement.style.width = width + 'px';
            canvas.parentElement.style.height = height + 'px';
            canvas.parentElement.style.minWidth = '0px';
            canvas.parentElement.style.minHeight = '0px';
            canvas.parentElement.style.maxWidth = width + 'px';
            canvas.parentElement.style.maxHeight = height + 'px';
        }
        if (!canvas) {
            return 'canvas-missing';
        }
        return canvas.clientWidth + 'x' + canvas.clientHeight + '|' + viewportWidth + 'x' + viewportHeight;
    })();
    """ % [TARGET_CANVAS_ASPECT]
    var metrics: String = str(JavaScriptBridge.eval(script, true)).strip_edges()
    if _metrics_logged:
        return
    _metrics_logged = true
    var viewport_size: Vector2 = get_viewport().get_visible_rect().size
    print(
        "[WebRuntimeCanvas] DOM=%s Godot=%dx%d aspect=%.6f" % [
            metrics,
            int(viewport_size.x),
            int(viewport_size.y),
            TARGET_CANVAS_ASPECT
        ]
    )
