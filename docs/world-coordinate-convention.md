# 世界坐标统一约定

本文档记录 Godot 客户端与 Go 服务端之间的世界/场景坐标约定，避免后续地图、传送门、NPC 和野外遇敌继续维护多套坐标。

## 坐标来源

- 服务端 `world.Vec2i` 是玩家场景整数坐标的权威来源，协议字段包括 `self_pos`、`corrected_pos`，持久化字段包括 `pos_x/pos_y`。正式登录、断线重连、普通门和世界地图快速传送都以服务端返回的坐标为唯一人物落点。
- `WORLD_RESYNC_PUSH.self_pos` 是客户端完成地图加载后摆放本地人物的事实来源；客户端只负责把服务端场景坐标转换为 Godot 像素坐标，不得再用场景导出变量覆盖快照。
- 同场景实时移动额外使用 `precise_pos` 千分之一格定点坐标。该字段用于短时权威移动与表现同步，服务端会按移动配置、场景边界和静态通行位图进行校验，不替代数据库整数坐标。
- 普通门请求只提交 `target_scene_id + portal_id`；服务端验证当前场景、门拓扑、目标场景、发布状态和等级，并从服务端世界拓扑选择目标位置。旧客户端跨场景请求携带的 `target_pos` 会被服务端忽略。
- 世界地图快速传送请求只提交 `target_scene_id`；目标中心来自数据库 `world_map_teleport_node`，由服务端校验、持久化并写入快照。
- 场景脚本中历史遗留的 `login_and_map_teleport_spawn_position`、`portal_spawn_scene_positions` 和 `get_portal_spawn_scene_position()` 可暂时保留以兼容旧资源或编辑器配置，但不再参与运行时权威落点链路。
- 每个场景自己的地图左上角统一视为 `(0, 0)`，不再使用每张地图单独维护的 `world_anchor/local_anchor` 偏移。

## 换算规则

- `grid_to_pixels` 由 `WorldSceneRegistry` 按场景配置；闪光镇旧地图主要为 `24`，闪光镇传送区和闪光平原地图为 `16`。
- 服务端场景坐标 `(x, y)` 渲染到客户端像素位置：`Vector2(x, y) * grid_to_pixels + map_origin_pixels`。
- 客户端像素位置换算回场景坐标：`(local_pixels - map_origin_pixels) / grid_to_pixels`。
- 实时表现坐标换算：`scene_position = Vector2(precise_pos.x, precise_pos.y) / 1000`，再使用相同 `grid_to_pixels` 规则转成像素。
- `map_origin_pixels` 由客户端在加载地图时根据 TileMap 可用矩形自动计算，保证地图左上角对应统一坐标 `(0, 0)`。

## 静态通行约束

- PostgreSQL `world_scene_navigation` 的当前已发布版本是服务端精细通行判定的事实来源；Godot 地图碰撞只通过导出工具生成待发布位图，不直接成为在线权威数据。
- 服务端默认出生点、普通门目标点和快速传送中心必须位于对应场景已发布位图的可通行格；新增或调整落点时必须同时完成导航审计。
- 普通移动由服务端依次执行输入、速度、矩形边界和静态通行路径校验；客户端碰撞只能改善本地操作体验，不能替代服务端判定。

## 开发要求

- 新增地图时，允许 `.tscn` 内部节点有编辑器偏移；运行时会自动把地图左上角归零。
- 新增或调整普通传送门时，只由客户端门区维护请求所需的 `portal_id + target_scene_id`；目标落点必须在服务端世界拓扑中维护并通过发布导航审计，不再在客户端场景脚本中新增权威出生配置。
- 服务端确认切图并下发目标场景快照后，客户端必须等待目标地图加载完成，再应用 `self_pos` 并初始化本地移动上报基线，避免把“刚落地”当成普通移动发送。
- 缩放居中用的 `get_level_center_position()` 仍由场景脚本基于 TileMap 自动计算 Godot 像素中心点，该表现逻辑不影响权威坐标。
- 右上角 HUD 场景名来自每个场景脚本导出的 `scene_display_name`，可在对应 `.tscn` 中覆盖。
- HUD 展示的“坐标”使用服务端权威场景格对应的当前人物位置，不显示 Godot 像素坐标。
