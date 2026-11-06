# 客户端通用 UI 目录

本目录存放可在多个玩法模块复用的 Godot UI 场景；对应脚本位于 `client/scripts/ui/common/`。

新增功能面板前，请先查阅本清单，优先复用已有通用组件，避免重复造轮子。

## 模态弹窗基类

| 资源 | 说明 |
| --- | --- |
| `modal_popup_layer.gd` | 运行时模态弹窗基类，提供遮罩、输入锁定与 `runtime_modal_popup` 组 |
| `runtime_root_panel.gd` | 运行时根面板基类，提供遮罩点空白关闭与 `menu_closed` 信号 |
| `anchored_popup_base.gd` | 锚点浮层基类，提供 `open_near`、视口 clamp 与 top_level 生命周期 |

## 完整面板

| 场景 | 脚本 | 说明 |
| --- | --- | --- |
| `confirm_prompt_popup.tscn` | `confirm_prompt_popup.gd` | 通用确认提示：标题 + BBCode 正文 + 底部继续按钮，右上角关闭取消 |
| `reward_popup.tscn` | `reward_popup.gd` | 通用奖励结算弹窗 |
| `item_slot_picker.tscn` | `item_slot_picker.gd` | 锚点旁弹出的物品格子选择浮层 |
| `action_menu_popup.tscn` | `action_menu_popup.gd` | 锚点旁竖向动作菜单（背包「更多」等） |
| `option_list_panel.tscn` | `option_list_panel.gd` | `680×1000` 移动端居中选项列表面板（NPC 菜单 / 附近 NPC / 挂机目标 / PVP 目标） |
| `runtime_progress_overlay.tscn` | `runtime_progress_overlay.gd` | 兼容壳：内部委托 `GenericLoadingScene`，保留 `show_waiting` / `play_progress` 旧接口 |
| `runtime_progress_bar_overlay.tscn` | `runtime_progress_bar_overlay.gd` | 固定时长线性进度条；**开礼包**等需要 3 秒进度条演出的流程专用 |
| `generic_loading_scene.tscn` | `generic_loading_scene.gd` | **标准**全屏 Loading：仅滚动动画 +「读取中」图字，不展示说明文案 |
| `dev_server_switcher.tscn` | `dev_server_switcher.gd` | 开发调试切服面板：可切本地 / 远程 / 浏览器同源，并支持手填 HTTP / WS 地址 |

## 组件

| 场景 | 脚本 | 说明 |
| --- | --- | --- |
| `menu_frame.tscn` | `menu_frame.gd` | 九宫格菜单边框装饰容器 |
| `runtime_action_button.tscn` | `runtime_action_button.gd` | 带悬停/按下缩放反馈的通用按钮 |
| `panel_close_button.tscn` | - | 通用 `88×88` 面板关闭按钮：普通/禁用使用半透明图，悬停/按下使用不透明图 |
| `bag_item_hover_name.tscn` | `bag_item_hover_name.gd` | 悬停物品名称浮层 |

## 纯脚本组件

| 脚本 | 说明 |
| --- | --- |
| `item_description_view.gd` | 物品介绍 BBCode 视图，支持 `{item:ID}` 内联 icon 与属性行分色 |

## 使用示例

```gdscript
var popup: ConfirmPromptPopup = preload(ConfirmPromptPopup.SCENE_PATH).instantiate() as ConfirmPromptPopup
popup.show_prompt("丢弃物品", "确定要丢弃 [color=#82d563]新手长剑[/color] 吗？")
```

```gdscript
# 推荐：直接使用 GenericLoadingScene
var loading: GenericLoadingScene = preload(GenericLoadingScene.SCENE_PATH).instantiate() as GenericLoadingScene
add_child(loading)
loading.show_loading()
loading.hide_loading()
```

```gdscript
# 兼容旧代码：RuntimeProgressOverlay 内部同样展示 GenericLoadingScene
var overlay: RuntimeProgressOverlay = preload(RuntimeProgressOverlay.SCENE_PATH).instantiate() as RuntimeProgressOverlay
add_child(overlay)
overlay.show_waiting()
overlay.hide_overlay()
await overlay.play_progress(3.0)
```

```gdscript
# 开礼包：使用独立进度条遮罩，保留 3 秒线性进度演出
var box_progress: RuntimeProgressBarOverlay = preload(RuntimeProgressBarOverlay.SCENE_PATH).instantiate() as RuntimeProgressBarOverlay
add_child(box_progress)
await box_progress.play_progress(3.0, "打开中...")
box_progress.hide_progress()
```

## 手机竖屏参数（2026-11-04）

- 基准视口为 `780×1440`，保留 `canvas_items + expand`；原生手机方向显式设为竖屏。下列尺寸均为设计坐标，不是设备物理像素或固定 dp。
- 默认字号为 `28`，主要正文使用 `28～32`，标题通常为 `36～40`；次要说明可以使用 `24`。不要再依赖桌面尺寸的 `12/14` 作为主要交互文字。
- 通用操作与关闭按钮高度为 `88`，常驻底部入口为 `96×96`，按钮间留出 `12～16` 间距。小图标可以保留原始比例，但热区不随图标缩小。
- 主菜单为 `720×1120`，6 个分类按钮与 10 个功能按钮均预置在场景；功能列表可以滚动。修改尺寸、字号、图标间距请在场景中操作，不在脚本中重建布局。
- 通用确认提示为 `736×560`；奖励弹窗为 `736×880`。标题、关闭和底部操作保持可见，仅正文区域滚动；每次打开将正文滚回顶部，保留服务端 BBCode 内容与确认/取消信号语义。
- NPC 对话卡片固定在屏幕底部上方 `64` 的位置，尺寸为 `736×720`。正文和分支选项分别滚动；拖动正文不推进剧情，未拖动的点击与继续按钮仍使用原有推进流程。
- 按钮输入兜底由通用模态基类复用：在同一热区按下并释放才触发，拖出和系统取消触摸不执行操作；正文输入继续交给滚动容器。不要恢复为全局吞掉所有指针事件。
- 背包与仓库维持每页 28 格，保留服务端快照、仓库存取的既有行为；扩大热区不改变分页、筛选和网络请求。宠物列表使用可滚动单列，数据页签与底部操作保持独立空间。
- 常驻 HUD 顶部预留 `72`、底部按钮预留 `64` 的设计空间；这属于布局留白，不等同于对所有机型刘海和软键盘的动态适配。Android 现有导出保持 `edge_to_edge=false`。
- 运行时弹窗场景根节点显式设置 `visible = false`，通过正式打开入口显示。普通按钮、格子与嵌入式数据页继续默认可见。

验证覆盖：`780×1440` 基准下的菜单、确认/取消、奖励关闭、对话推进与滚动回归；`360×780`、`390×844`、`430×932` 桌面模拟窗口下的 16 个 UI 场景布局与截图。输入法、系统安全区及实际手指操作需在 Android/iOS 真机继续验收。

## 参考图像素边框与通透底色（2026-11-04）

- 使用 `dialogue/原版样式.tres` 的面板共享 `resources/ui/pixel_panel_material.tres`，仅替换底图中心颜色。材质只挂在面板容器，不传递给正文、图标或按钮子节点，因此不会整体降低文字和描边的不透明度。
- `background_color` 默认 `Color(0.012, 0.016, 0.012, 0.68)`：偏中性的近黑底，保留地图轮廓。确认弹窗遮罩为 0.08，叠加后地图颜色权重约 29%；商店、数量和修复弹窗遮罩同为 0.08。不要再用 0.65 全屏遮罩叠加几乎不透明的深蓝底图。
- `border_inset_pixels = 2` 保留旧贴图的像素转角与亮暗线，并向内部扩展两像素，使手机缩放后边框仍可辨识；额外采样仅发生在九宫格边缘。面板使用最近邻过滤，不用整体 scale 放大 UI。
- 提示、奖励及对话正文的额外边距统一为 8，加上原样式 8 的内容边距，总缩进为 16 个设计像素。提示段落间距为 4，标题/正文之间间距为 4；原有 88 高按钮热区、富文本、长文本滚动和确认/取消契约不变。
- 奖励弹窗覆盖原模糊材质为透明遮罩，保留原节点及输入拦截，地图不再被模糊成色块。
- 此材质依赖现有 `menu-fan7.png`：中心 alpha 约为 0.957，像素边框为完全不透明，透明转角保持透明。若替换底图，应同步检查材质的中心识别范围；不要直接给绘制字体的 Label/Button 挂此材质。

- 提示、奖励和对话框宽度为 736，780 基准视口左右各留 22（390 宽显示时约 11），接近参考图的贴边比例；按钮热区仍保持原手机操作尺寸。

## 数据面板透明度统一（2026-11-04）

- 背包、人物、宠物和任务的 UI 图集底板复用 `resources/ui/atlas_panel_material.tres`。它与原版菜单材质共用同一着色器及默认近黑底色（不透明度 0.68），识别图集原深蓝底 `#0d1c3d`，保留原金色边框和转角。只挂在背景节点，禁止对子节点启用材质继承。
- 仓库外框的 `StyleBoxFlat_panel.bg_color` 同为 `Color(0.012, 0.016, 0.012, 0.68)`；人物数据行、宠物列表/数据页和仓库内卡片的填充透明，保留分隔线，避免底板相乘后变成实心。
- 根面板遮罩与确认、奖励、商店及物品操作弹窗统一为黑色 0.08；原模糊背景实例保留输入节点，材质置空、颜色透明，不再重复压暗或模糊地图。锻造外框使用原版菜单材质，内说明底板保持透明。
- 这里只统一面板底板；按钮状态、格子底色、进度和宠物装饰仍使用原资源。打开叠在另一面板上的详情时，底下的面板仍正常显示。

## 背包内部布局修正（2026-11-04）

- `bag/bag_panel.tscn` 在外框内统一留 16，分区间距 12，不用负边距把内容推向边框；容量与货币仍分两行，数值区和加号分别分配空间。
- 88×88 是按钮点击尺寸。加号、翻页符号用字号 36 绘制，避免把 5×5 的图集符号直接拉伸到整个按钮；禁用翻页也保留可辨认的灰色符号。
- 页码输入框填满既有页码容器，分类、页码和前后翻页留在同一行，保留原节点路径及服务端分页行为。
