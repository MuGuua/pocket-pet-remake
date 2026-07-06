extends Node


## Web 运行时统一使用的设计宽度，用于计算目标纵横比。
const TARGET_CANVAS_WIDTH: float = 780.0
## Web 运行时统一使用的设计高度，用于计算目标纵横比。
const TARGET_CANVAS_HEIGHT: float = 1440.0
## 目标画布纵横比；浏览器内允许同比例缩放，但不允许拉伸变形。
const TARGET_CANVAS_ASPECT: float = TARGET_CANVAS_WIDTH / TARGET_CANVAS_HEIGHT
## Web 本地调试时直接铺满浏览器窗口，避免桌面浏览器按手机竖屏比例缩得过窄。
const USE_FULL_BROWSER_VIEWPORT_IN_DEBUG: bool = true

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
## 本地调试构建铺满浏览器，方便在桌面浏览器看到完整视口；正式 Web 运行仍锁定 13:24 纵横比。
func _apply_canvas_constraints() -> void:
    var script: String = """
    (function () {
        const aspect = %f;
        const useFullBrowserViewport = %s;
        const root = document.documentElement;
        const body = document.body;
        const canvas = document.getElementById('canvas') || document.querySelector('canvas');
        const status = document.getElementById('status');
        const viewportWidth = window.innerWidth || 0;
        const viewportHeight = window.innerHeight || 0;
        let width = viewportWidth;
        let height = viewportHeight;
        if (!useFullBrowserViewport && viewportWidth > 0 && viewportHeight > 0) {
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
            canvas.style.aspectRatio = useFullBrowserViewport ? 'auto' : String(aspect);
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
    """ % [TARGET_CANVAS_ASPECT, _bool_to_js(_should_use_full_browser_viewport())]
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


## 判断当前 Web 运行是否应该铺满浏览器窗口；仅调试构建启用，避免影响正式移动端竖屏比例。
func _should_use_full_browser_viewport() -> bool:
    return USE_FULL_BROWSER_VIEWPORT_IN_DEBUG and OS.is_debug_build()


## 把 GDScript 布尔值转为 JavaScript 字面量，避免字符串插值后出现 Godot 的 True/False 写法。
func _bool_to_js(value: bool) -> String:
    if value:
        return "true"
    return "false"
