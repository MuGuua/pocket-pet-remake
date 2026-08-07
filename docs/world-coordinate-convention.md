# 世界坐标统一约定

本文档记录 Godot 客户端与 Go 服务端之间的世界/场景坐标约定，避免后续地图、传送门、NPC 和野外遇敌继续维护多套坐标。

## 坐标来源

- 服务端 `world.Vec2i` 是玩家场景和日常移动的持久化坐标，字段包括 `self_pos`、`corrected_pos`、玩家持久化 `pos_x/pos_y`。断线重连继续使用该坐标；正式登录、世界地图快速传送和普通门切图的本地出生表现则由目标场景导出变量覆盖。
- 同场景实时移动额外使用 `precise_pos` 千分之一格定点坐标；该字段只负责短时表现，服务端会将其限制在当前整数权威格周围半格内，不替代数据库坐标。
- 普通门请求只提交 `target_scene_id + portal_id`。服务端验证当前场景、门拓扑、目标场景、发布状态和等级，但不接收或采用目标地图出生格；目标地图加载后，客户端从当前场景脚本调用 `get_portal_spawn_scene_position(portal_id)` 读取导出落点。未配置专属入口时只回退当前场景脚本的 `login_and_map_teleport_spawn_position`。旧客户端跨场景请求携带的 `target_pos` 会被服务端忽略。
- 每个场景自己的地图左上角统一视为 `(0, 0)`，不再使用每张地图单独维护的 `world_anchor/local_anchor` 偏移。

## 换算规则

- `grid_to_pixels` 由 `WorldSceneRegistry` 按场景配置；闪光镇旧地图主要为 `24`，闪光镇传送区和闪光平原地图为 `16`。
- 服务端场景坐标 `(x, y)` 渲染到客户端像素位置：`Vector2(x, y) * grid_to_pixels + map_origin_pixels`。
- 客户端像素位置换算回场景坐标：`(local_pixels - map_origin_pixels) / grid_to_pixels`。
- 实时表现坐标换算：`scene_position = Vector2(precise_pos.x, precise_pos.y) / 1000`，再使用相同 `grid_to_pixels` 规则转成像素。
- `map_origin_pixels` 由客户端在加载地图时根据 TileMap 可用矩形自动计算，保证地图左上角对应统一坐标 `(0, 0)`。

## 开发要求

- 新增地图时，允许 `.tscn` 内部节点有编辑器偏移；运行时会自动把地图左上角归零。
- 新增或调整普通传送门落点时，只在目标地图脚本中维护来源 `portal_id -> 场景坐标`，并确认来源门提交的 `target_scene_id` 与服务端拓扑一致。场景坐标允许使用负数；该坐标不加入协议、不写入服务端门配置，也不需要与服务端 `worldScenes` 的内部兼容位置保持一致。
- 服务端确认普通门切图并下发目标场景快照后，客户端才读取目标场景导出变量并摆放本地人物。应用导出落点时会同步初始化本地移动上报基线，避免“刚落地”本身被当成普通移动发送；玩家之后真正移动时仍恢复原有持久化和多人同步。缩放居中用的 `get_level_center_position()` 仍由脚本基于 TileMap 自动计算 Godot 像素中心点。
- 右上角 HUD 场景名来自每个场景脚本导出的 `scene_display_name`，可在对应 `.tscn` 中覆盖。
- HUD 展示的“坐标”使用当前客户端人物的场景格坐标，而不是 Godot 像素坐标；通过本地出生点落地后应立即展示场景导出坐标。
