# item_icons

客户端本地物品图标资源目录。每个物品单独一个 `.tres` 文件，运行时按 `item_id` 索引。

背包、装备槽、奖励弹窗等 UI **只通过 `item_id` 查本地资源显示图标**，不读取服务端下发的 `icon` 字段。

**武器、防具与其他道具共用同一套规则**，不再使用已删除的 `item_icon_registry.tres` 集中注册表。

## 新增物品图标

1. 在 `res://resources/item_icons/` 下新建资源（可复制 `atlas_icon_template.tres`）。
2. 资源类型选择 `ItemIconDefinition`（脚本：`scripts/ui/bag/item_icon_definition.gd`）。
3. 填写 `item_id`（与服务端 `item_definition.item_id` 一致）。
4. 配置 `static_texture`（常见为 `AtlasTexture`），或配置 `sprite_frames` 做帧动画。

文件命名建议用物品中文名（如 `飞行之羽.tres`），也支持 `{item_id}.tres` 纯数字命名。

## 静态与动画

- **静态图标**：只填 `static_texture`。
- **帧动画**：填 `sprite_frames` 与 `animation`；静态预览仍走 `ItemIcons.resolve_texture()` 的首帧。
- 需要格子内播放动画时，可调用 `ItemIcons.resolve_definition(item_id)` 读取完整定义。

## 默认图标

未配置的 `item_id` 回退到 `_default.tres`；若默认资源缺失，再回退到内置图集占位。

## 兼容写法

`3003.tres` 若直接保存为 `AtlasTexture` / `Texture2D`（文件名即 item_id），启动时也会自动注册，无需 `ItemIconDefinition` 包装。
