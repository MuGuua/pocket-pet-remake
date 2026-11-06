extends RuntimeRootPanel

## 保留菜单数据与信号契约，主场景继续负责服务端请求及业务面板打开。
signal menu_item_selected(item: Dictionary)

## 原始图集单个图标宽度，仅用于裁剪；展示大小由场景按钮管理。
const ICON_SIZE: int = 16
## 菜单沿用已有图集，不创建新的美术资源。
const MENU_ICON_1: Texture2D = preload("res://asset/分类/菜单表情/菜单图标1.png")
## 第二份菜单图集。
const MENU_ICON_2: Texture2D = preload("res://asset/分类/菜单表情/菜单图标2.png")

const MENU_DATA: Array[Dictionary] = [
    {
        "title": "常用",
        "items": [
            {"label": "物品行囊", "icon": "2.8"},
            {"label": "个人状态", "icon": "2.10"},
            {"label": "宠物指令", "icon": "1.9"},
            {"label": "队伍指令", "icon": "1.13"},
            {"label": "神秘商店", "icon": "2.4"},
            {"label": "佣兵指令", "icon": "1.9"},
            {"label": "宠物之魂", "icon": "2.12"},
            {"label": "龙魂系统", "icon": "1.49"},
            {"label": "全服竞技场", "icon": "1.49"},
            {"label": "仙气修炼", "icon": "1.49"},
        ],
    },
    {
        "title": "辅助",
        "items": [
            {"label": "快速购物", "icon": "1.16"},
            {"label": "生产装备", "icon": "2.18"},
            {"label": "快速补血", "icon": "1.14"},
            {"label": "查看成就", "icon": "2.6"},
            {"label": "原地挂机", "icon": "1.47"},
            {"label": "生活技能", "icon": "1.41"},
            {"label": "灵魂粉末", "icon": "1.11"},
        ],
    },
    {
        "title": "聊天",
        "items": [
            {"label": "同屏聊天", "icon": "2.7"},
            {"label": "本服广播", "icon": "2.28"},
            {"label": "本线广播", "icon": "2.28"},
            {"label": "队伍聊天", "icon": "2.7"},
            {"label": "发悄悄话", "icon": "2.7"},
            {"label": "口袋好友", "icon": "1.42"},
            {"label": "家族指令", "icon": "1.42"},
        ],
    },
    {
        "title": "任务",
        "items": [
            {"label": "查看任务", "icon": "2.21"},
            {"label": "区域地图", "icon": "2.11"},
            {"label": "世界地图", "icon": "2.11"},
            {"label": "每日活动", "icon": "2.20"},
            {"label": "口袋攻略", "icon": "1.12"},
        ],
    },
    {
        "title": "系统",
        "items": [
            {"label": "流量设置", "icon": "1.13"},
            {"label": "游戏帮助", "icon": "1.13"},
            {"label": "细节设置", "icon": "1.13"},
            {"label": "退出游戏", "icon": "1.13"},
            {"label": "免费领取", "icon": "1.13"},
            {"label": "账号功能", "icon": "1.13"},
        ],
    },
    {
        "title": "商城",
        "items": [
            {"label": "游戏充值", "icon": "2.4"},
            {"label": "元宝道具", "icon": "2.4"},
            {"label": "金币道具", "icon": "2.4"},
            {"label": "当前等级特惠商品", "icon": "2.4"},
            {"label": "团包", "emoji": "📦"},
        ],
    },
]

## 分类按钮由场景预置，布局和热区可直接在编辑器中调整。
@onready var _tabs: GridContainer = %TabsRow
## 功能按钮由场景预置，长列表放在独立滚动区域内。
@onready var _items: VBoxContainer = %ItemsList
## 分类切换后将列表滚回顶部，防止保留上个分类的滚动位置。
@onready var _items_scroll: ScrollContainer = %ItemsScroll
## 标题栏关闭按钮，位置跟随面板而非屏幕边缘。
@onready var close_button: BaseButton = %CloseButton
## 当前菜单分类。
var _current_tab_index: int = 0
## 当前键盘选择的功能索引。
var _current_item_index: int = 0
## 场景中可复用的分类按钮。
var _tab_buttons: Array[Button] = []
## 场景中可复用的功能按钮。
var _item_buttons: Array[Button] = []


## 绑定场景预置按钮；不在脚本中生成 UI 或修改控件尺寸。
func _ready() -> void:
    super._ready()
    close_button.pressed.connect(close_menu)
    for child: Node in _tabs.get_children():
        var button: Button = child as Button
        if button == null:
            continue
        var index: int = _tab_buttons.size()
        _tab_buttons.append(button)
        button.pressed.connect(_select_tab.bind(index))
    for child: Node in _items.get_children():
        var button: Button = child as Button
        if button == null:
            continue
        var index: int = _item_buttons.size()
        _item_buttons.append(button)
        button.pressed.connect(_on_item_pressed.bind(index))
    _refresh_tabs()
    _refresh_items()


## 打开菜单并恢复当前分类；显示数据仍使用原有菜单定义。
func open_menu() -> void:
    super.open_menu()
    _refresh_tabs()
    _refresh_items()


## 更新选中分类的文字和按下状态，颜色与轮廓来自既有按钮资源。
func _refresh_tabs() -> void:
    for index: int in range(_tab_buttons.size()):
        var button: Button = _tab_buttons[index]
        button.visible = index < MENU_DATA.size()
        if not button.visible:
            continue
        var title: String = str(MENU_DATA[index].get("title", ""))
        button.text = "【%s】" % title if index == _current_tab_index else title
        button.set_pressed_no_signal(index == _current_tab_index)


## 刷新场景预置功能按钮；按钮数量覆盖原菜单最大分类，不改变功能入口。
func _refresh_items() -> void:
    var items: Array = MENU_DATA[_current_tab_index].get("items", [])
    for index: int in range(_item_buttons.size()):
        var button: Button = _item_buttons[index]
        button.visible = index < items.size()
        if not button.visible:
            continue
        var item: Dictionary = items[index] as Dictionary
        button.text = str(item.get("label", ""))
        if item.has("emoji"):
            button.icon = null
            button.text = "%s  %s" % [str(item.get("emoji", "")), button.text]
        else:
            button.icon = _build_icon_texture(str(item.get("icon", "")))
    _current_item_index = clampi(_current_item_index, 0, maxi(items.size() - 1, 0))
    _refresh_item_selection()


## 仅标记当前键盘选择项，不影响触屏按下和释放的原生按钮反馈。
func _refresh_item_selection() -> void:
    for index: int in range(_item_buttons.size()):
        _item_buttons[index].set_pressed_no_signal(index == _current_item_index)


## index 为场景分类按钮索引；切换后重新展示该分类并回到列表顶部。
func _select_tab(index: int) -> void:
    if index < 0 or index >= MENU_DATA.size():
        return
    _current_tab_index = index
    _current_item_index = 0
    _items_scroll.scroll_vertical = 0
    _refresh_tabs()
    _refresh_items()


## index 为场景功能按钮索引；释放时才执行业务，拖动列表不会在按下时误触。
func _on_item_pressed(index: int) -> void:
    _current_item_index = index
    _refresh_item_selection()
    _emit_current_item_selected()


## 保留左右切分类、上下选功能的键盘操作；触屏由原生按钮与滚动容器处理。
func _unhandled_input(event: InputEvent) -> void:
    if not visible or event is not InputEventKey:
        return
    var key_event: InputEventKey = event as InputEventKey
    if not key_event.pressed or key_event.echo:
        return
    if event.is_action_pressed("ui_left"):
        _select_tab(wrapi(_current_tab_index - 1, 0, MENU_DATA.size()))
    elif event.is_action_pressed("ui_right"):
        _select_tab(wrapi(_current_tab_index + 1, 0, MENU_DATA.size()))
    elif event.is_action_pressed("ui_up") or event.is_action_pressed("ui_down"):
        var items: Array = MENU_DATA[_current_tab_index].get("items", [])
        if items.is_empty():
            return
        var step: int = -1 if event.is_action_pressed("ui_up") else 1
        _current_item_index = wrapi(_current_item_index + step, 0, items.size())
        _refresh_item_selection()
        _items_scroll.ensure_control_visible(_item_buttons[_current_item_index])
    else:
        return
    get_viewport().set_input_as_handled()


## icon_ref 为原菜单的图集编号与格子编号；返回原图标单帧，非法编号返回空值。
func _build_icon_texture(icon_ref: String) -> Texture2D:
    var parts: PackedStringArray = icon_ref.split(".")
    if parts.size() != 2:
        return null
    var sheet_index: int = int(parts[0])
    var icon_index: int = int(parts[1]) - 1
    if icon_index < 0:
        return null
    var atlas_texture: AtlasTexture = AtlasTexture.new()
    atlas_texture.atlas = MENU_ICON_1 if sheet_index == 1 else MENU_ICON_2
    atlas_texture.region = Rect2(icon_index * ICON_SIZE, 0, ICON_SIZE, ICON_SIZE)
    return atlas_texture


## 将当前项按原有结构广播给主场景，保持分类标题与业务入口兼容。
func _emit_current_item_selected() -> void:
    var items: Array = MENU_DATA[_current_tab_index].get("items", [])
    if _current_item_index < 0 or _current_item_index >= items.size():
        return
    var item: Dictionary = (items[_current_item_index] as Dictionary).duplicate(true)
    item["tab_title"] = str(MENU_DATA[_current_tab_index].get("title", ""))
    menu_item_selected.emit(item)
