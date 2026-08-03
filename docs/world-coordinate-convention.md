# 世界坐标统一约定

本文档记录 Godot 客户端与 Go 服务端之间的世界/场景坐标约定，避免后续地图、传送门、NPC 和野外遇敌继续维护多套坐标。

## 坐标来源

- 服务端 `world.Vec2i` 是最终持久化场景坐标，字段包括 `self_pos`、`corrected_pos`、玩家持久化 `pos_x/pos_y`。客户端实际摆放一律消费服务端下发的 `self_pos`，避免本人、旁观者和重连档案出现不同坐标。
- 同场景实时移动额外使用 `precise_pos` 千分之一格定点坐标；该字段只负责短时表现，服务端会将其限制在当前整数权威格周围半格内，不替代数据库坐标。
- 普通门落点由客户端目标场景脚本的 `get_portal_spawn_scene_position(portal_id)` 选择，并随 `MOVE_INTENT_REQ.target_pos` 提交。服务端验证当前场景、门拓扑、目标场景和等级后采用该坐标，使进入者本人、旁观者推送和数据库落库坐标保持一致；未配置入口时回退 `worldScenes` 门点。登录、重连与世界地图快速传送不使用该客户端入口选择，快速传送继续读取数据库中心点。
- 每个场景自己的地图左上角统一视为 `(0, 0)`，不再使用每张地图单独维护的 `world_anchor/local_anchor` 偏移。

## 换算规则

- `grid_to_pixels` 由 `WorldSceneRegistry` 按场景配置；闪光镇旧地图主要为 `24`，闪光镇传送区和闪光平原地图为 `16`。
- 服务端场景坐标 `(x, y)` 渲染到客户端像素位置：`Vector2(x, y) * grid_to_pixels + map_origin_pixels`。
- 客户端像素位置换算回场景坐标：`(local_pixels - map_origin_pixels) / grid_to_pixels`。
- 实时表现坐标换算：`scene_position = Vector2(precise_pos.x, precise_pos.y) / 1000`，再使用相同 `grid_to_pixels` 规则转成像素。
- `map_origin_pixels` 由客户端在加载地图时根据 TileMap 可用矩形自动计算，保证地图左上角对应统一坐标 `(0, 0)`。

## 开发要求

- 新增地图时，允许 `.tscn` 内部节点有编辑器偏移；运行时会自动把地图左上角归零。
- 新增或调整普通传送门落点时，在目标地图脚本中维护来源 `portal_id -> 场景坐标`，并确认来源门提交的 `target_scene_id` 与服务端拓扑一致。服务端 `worldScenes` portals/entries 继续保留可达关系与旧客户端兼容落点，但不再是新客户端普通门出生格的唯一配置来源。
- 玩家最终摆放仍统一以服务端确认后的 `self_pos` 为准；客户端脚本选定的普通门入口必须先由服务端验证、持久化并回传，不能在收到快照后本地静默覆盖。缩放居中用的 `get_level_center_position()` 仍由脚本基于 TileMap 自动计算 Godot 像素中心点。
- 右上角 HUD 场景名来自每个场景脚本导出的 `scene_display_name`，可在对应 `.tscn` 中覆盖。
- HUD 展示的“坐标”应与服务端 `self_pos` 使用同一套场景坐标，而不是 Godot 像素坐标。
