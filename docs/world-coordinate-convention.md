# 世界坐标统一约定

本文档记录 Godot 客户端与 Go 服务端之间的世界/场景坐标约定，避免后续地图、传送门、NPC 和野外遇敌继续维护多套坐标。

## 坐标来源

- 服务端 `world.Vec2i` 是持久化场景坐标，字段包括 `self_pos`、玩家持久化 `pos_x/pos_y`，仍用于进入世界和无客户端落点配置时兜底。
- 客户端地图显示使用像素坐标；传送门切图后的实际站位由目标场景脚本按 `portal_id` 配置场景坐标，再由 `world_controller` 统一换算为像素。
- 每个场景自己的地图左上角统一视为 `(0, 0)`，不再使用每张地图单独维护的 `world_anchor/local_anchor` 偏移。

## 换算规则

- 当前客户端统一使用 `grid_to_pixels = 24`。
- 服务端场景坐标 `(x, y)` 渲染到客户端像素位置：`Vector2(x, y) * grid_to_pixels + map_origin_pixels`。
- 客户端像素位置换算回场景坐标：`(local_pixels - map_origin_pixels) / grid_to_pixels`。
- `map_origin_pixels` 由客户端在加载地图时根据 TileMap 可用矩形自动计算，保证地图左上角对应统一坐标 `(0, 0)`。

## 开发要求

- 新增地图时，允许 `.tscn` 内部节点有编辑器偏移；运行时会自动把地图左上角归零。
- 新增传送门落点时，在目标场景的 `client/scripts/feature/world/scene_levels/*_level.gd` 中维护 `get_portal_spawn_scene_position(portal_id)`，不要再改服务端落点。
- 玩家出生/传送落点使用导出的场景坐标维护；缩放居中用的 `get_level_center_position()` 仍由脚本基于 TileMap 自动计算 Godot 像素中心点，不使用导出值覆盖。
- 右上角 HUD 场景名来自每个场景脚本导出的 `scene_display_name`，可在对应 `.tscn` 中覆盖。
- HUD 展示的“坐标”应与服务端 `self_pos` 使用同一套场景坐标，而不是 Godot 像素坐标。
