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
| `confirm_prompt_popup.tscn` | `confirm_prompt_popup.gd` | 通用确认提示：标题 + BBCode 正文 + 左确定 / 右取消 |
| `info_modal_popup.tscn` | `info_modal_popup.gd` | 通用信息模态：可选标题 + 多行纯文本 + 确定（玩家/宠物升级等） |
| `reward_popup.tscn` | `reward_popup.gd` | 通用奖励结算弹窗 |
| `item_slot_picker.tscn` | `item_slot_picker.gd` | 锚点旁弹出的物品格子选择浮层 |
| `action_menu_popup.tscn` | `action_menu_popup.gd` | 锚点旁竖向动作菜单（背包「更多」等） |
| `option_list_panel.tscn` | `option_list_panel.gd` | 居中选项列表面板（NPC 菜单 / 附近 NPC / PVP 目标） |
| `runtime_progress_overlay.tscn` | `runtime_progress_overlay.gd` | 兼容壳：内部委托 `GenericLoadingScene`，保留 `show_waiting` / `play_progress` 旧接口 |
| `runtime_progress_bar_overlay.tscn` | `runtime_progress_bar_overlay.gd` | 固定时长线性进度条；**开礼包**等需要 3 秒进度条演出的流程专用 |
| `generic_loading_scene.tscn` | `generic_loading_scene.gd` | **标准**全屏 Loading：仅滚动动画 +「读取中」图字，不展示说明文案 |

## 组件

| 场景 | 脚本 | 说明 |
| --- | --- | --- |
| `menu_frame.tscn` | `menu_frame.gd` | 九宫格菜单边框装饰容器 |
| `runtime_action_button.tscn` | `runtime_action_button.gd` | 带悬停/按下缩放反馈的通用按钮 |
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
