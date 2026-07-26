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
## Web 调试与正式构建都使用 780x1440 设计尺寸；浏览器空间不足时只做等比缩小，不扩大逻辑视野。
func _apply_canvas_constraints() -> void:
    var script: String = """
    (function () {
        const aspect = %f;
        const designWidth = %f;
        const designHeight = %f;
        const root = document.documentElement;
        const body = document.body;
        const canvas = document.getElementById('canvas') || document.querySelector('canvas');
        const status = document.getElementById('status');
        const setImportant = function (element, property, value) {
            if (element) {
                element.style.setProperty(property, value, 'important');
            }
        };
        const viewportWidth = window.innerWidth || 0;
        const viewportHeight = window.innerHeight || 0;
        let width = Math.min(viewportWidth, designWidth);
        let height = Math.min(viewportHeight, designHeight);
        if (width > 0 && height > 0) {
            if (viewportWidth / viewportHeight > aspect) {
                width = height * aspect;
            } else {
                height = width / aspect;
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
            setImportant(canvas, 'display', 'block');
            setImportant(canvas, 'width', width + 'px');
            setImportant(canvas, 'height', height + 'px');
            setImportant(canvas, 'min-width', '0px');
            setImportant(canvas, 'min-height', '0px');
            setImportant(canvas, 'max-width', width + 'px');
            setImportant(canvas, 'max-height', height + 'px');
            setImportant(canvas, 'aspect-ratio', String(aspect));
            setImportant(canvas, 'background', '#000');
        }
        if (status) {
            setImportant(status, 'width', width + 'px');
            setImportant(status, 'height', height + 'px');
            setImportant(status, 'min-width', '0px');
            setImportant(status, 'min-height', '0px');
            setImportant(status, 'max-width', width + 'px');
            setImportant(status, 'max-height', height + 'px');
            setImportant(status, 'aspect-ratio', String(aspect));
        }
        const canvasParent = canvas ? canvas.parentElement : null;
        if (canvasParent && canvasParent !== body && canvasParent !== root) {
            setImportant(canvasParent, 'width', width + 'px');
            setImportant(canvasParent, 'height', height + 'px');
            setImportant(canvasParent, 'min-width', '0px');
            setImportant(canvasParent, 'min-height', '0px');
            setImportant(canvasParent, 'max-width', width + 'px');
            setImportant(canvasParent, 'max-height', height + 'px');
        }
        if (!canvas) {
            return 'canvas-missing';
        }
        return canvas.clientWidth + 'x' + canvas.clientHeight + '|' + viewportWidth + 'x' + viewportHeight;
    })();
    """ % [TARGET_CANVAS_ASPECT, TARGET_CANVAS_WIDTH, TARGET_CANVAS_HEIGHT]
    JavaScriptBridge.eval(script, true)
    if _metrics_logged:
        return
    _metrics_logged = true
