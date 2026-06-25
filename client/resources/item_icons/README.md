# item_icons

客户端本地物品图标资源目录。

背包、装备、奖励弹窗等 UI **只通过 `item_id` 在本地查表显示图标**，不再读取服务端下发的 `icon` 字段。

## 配置方式

在 `res://resources/ui/item_icon_registry.tres` 中为每个物品模板维护映射：

- `item_id`：与服务端 `item_definition.item_id` 一致（推荐）
- `texture`：本地 `AtlasTexture` 或独立贴图

未配置的 `item_id` 会回退到注册表中的 `default_icon`。

## 示例

- 在 `item_icon_registry.tres` 的 `entries` 里新增一条，`item_id = 3003`，绑定对应 `AtlasTexture`
- 或在 `res://resources/item_icons/` 下创建 `.tres` 图集子资源，再挂到注册表条目上
