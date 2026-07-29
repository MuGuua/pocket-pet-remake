# 世界坐标统一约定

本文档记录 Godot 客户端与 Go 服务端之间的世界/场景坐标约定，避免后续地图、传送门、NPC 和野外遇敌继续维护多套坐标。

## 坐标来源

- 服务端 `world.Vec2i` 是持久化场景坐标，字段包括 `self_pos`、玩家持久化 `pos_x/pos_y`。登录、切图和地图快速传送后的实际站位一律使用服务端下发的 `self_pos`。
- 同场景实时移动额外使用 `precise_pos` 千分之一格定点坐标；该字段只负责短时表现，服务端会将其限制在当前整数权威格周围半格内，不替代数据库坐标。
- 传送门落点由服务端 `world_repo.go` 中的 `worldScenes` portals/entries 配置权威决定；进入者本人、同场景旁观者和数据库落库坐标使用同一个值。客户端场景脚本中的 `get_portal_spawn_scene_position` / `get_login_spawn_position` 导出已废弃，不再参与站位计算。
- 每个场景自己的地图左上角统一视为 `(0, 0)`，不再使用每张地图单独维护的 `world_anchor/local_anchor` 偏移。

## 换算规则

- 当前客户端统一使用 `grid_to_pixels = 24`。
- 服务端场景坐标 `(x, y)` 渲染到客户端像素位置：`Vector2(x, y) * grid_to_pixels + map_origin_pixels`。
- 客户端像素位置换算回场景坐标：`(local_pixels - map_origin_pixels) / grid_to_pixels`。
- 实时表现坐标换算：`scene_position = Vector2(precise_pos.x, precise_pos.y) / 1000`，再使用相同 `grid_to_pixels` 规则转成像素。
- `map_origin_pixels` 由客户端在加载地图时根据 TileMap 可用矩形自动计算，保证地图左上角对应统一坐标 `(0, 0)`。

## 开发要求

- 新增地图时，允许 `.tscn` 内部节点有编辑器偏移；运行时会自动把地图左上角归零。
- 新增或调整传送门落点时，修改服务端 `backend/server/internal/data/postgres/world_repo.go` 的 `worldScenes` portals/entries 配置，并同步更新 `world_repo_test.go` 中的逐门坐标断言；不要再在客户端场景脚本里维护本地落点覆盖。
- 玩家出生/传送落点统一以服务端 `self_pos` 为准；缩放居中用的 `get_level_center_position()` 仍由脚本基于 TileMap 自动计算 Godot 像素中心点，不使用导出值覆盖。
- 右上角 HUD 场景名来自每个场景脚本导出的 `scene_display_name`，可在对应 `.tscn` 中覆盖。
- HUD 展示的“坐标”应与服务端 `self_pos` 使用同一套场景坐标，而不是 Godot 像素坐标。
