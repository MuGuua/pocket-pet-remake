# 最新变更记录

## 2026-06-15
- 新增玩家成长体系第一版：迁移 `035_player_level_progression.sql`、`036_admin_player_progression_permissions.sql`，引入等级经验表、属性转化率表、玩家自由属性点与 `base_*` 裸装战斗值
- 新增 `module/progression` 领域模块，统一承接经验连升、升级发点、加点校验与战斗属性重算；`player.AddExp`、战斗结算发经验、任务发经验均走该模块
- 新增 WebSocket `PLAYER_ALLOCATE_ATTR_REQ/RESP (2061/2062)`；`PlayerSnapshot` 与 `BattleResultPush` 扩展成长相关字段
- 后台新增 `/player-progression` 页面与 `/api/admin/player-progression/...` 配置接口；玩家详情页展示自由属性点与四维分配值
- 客户端状态面板与加点页已对接服务端权威快照，加点请求带 loading 遮罩
- 设计文档：`backend/docs/player-progression.md`

## 2026-06-11
- 新增迁移 `backend/server/migrations/010_add_player_pet_mana.sql`，为 `player_pet` 表补齐 `mana` 字段，并同步回填演示宠物初始法力，修复 PostgreSQL 模式下进入世界/读取编队时因 `pp.mana` 缺列导致的失败
- 服务端配置加载方式已从 `config.env` 环境变量文件切换为单一 YAML 配置文件：`backend/server/cmd/game-server/main.go` 现在会优先解析 `backend/server/configs/config.yaml`
- `backend/server/internal/config/config.go` 改为从 YAML 结构读取 `http/auth/heartbeat/postgres/redis` 五段配置，并继续复用原有运行时校验逻辑，避免只改加载方式就把启动约束放松
- 新增 `backend/server/internal/config/config_test.go`、`backend/server/internal/config/yamlfile_test.go`，覆盖 YAML 配置解析、默认路径解析与基础校验
- 示例配置文件已从 `backend/server/configs/config.env(.example)` 切换为 `backend/server/configs/config.yaml(.example)`；`PP_CONFIG_FILE` 现仅用于覆盖 YAML 文件路径，不再承载各项运行参数

## 2026-06-10
- 服务端配置已收敛为单一 `PostgreSQL + Redis` 运行路径：`backend/server/internal/config/config.go`、`backend/server/internal/data/provider/` 与示例环境变量已删除 `memory` / `PP_REPOSITORY_MODE` 分支，后续不再维护双仓储模式
- 已新增 `backend/server/migrations/006_seed_postgres_demo_account.sql`，为 `postgres_redis` 模式补齐 `demo / demo123` 演示账号、`DemoTrainer` 玩家、三只起始宠物与默认编队；切到 PostgreSQL 仓储后不再因为数据库缺少演示数据而登录失败
- 服务端已新增数据库驱动的 NPC 配置能力：新增 `backend/server/migrations/004_npc_config.sql`，引入 `world_entity_definition` 与 `npc_menu_entry` 两张表，并预置当前世界里的引导 NPC、市场 NPC 与仓库 NPC 数据
- `backend/server/internal/module/npc/` 已补齐最小 NPC 配置服务边界，`battle_handler.go` 不再通过硬编码 `switch entity_id` 组装菜单与对话，而是统一从 NPC 仓储读取静态菜单项与动作结果
- `backend/server/internal/data/postgres/world_repo.go` 已接管 PostgreSQL 模式下的世界实体查询；后续若要新增一个可交互 NPC，只需在 SQL 迁移或数据库数据中新增实体定义和菜单项，不必再改服务端交互代码
- 内存模式也已同步补上 `NPCRepository` 和 `91001 罗思` 的基础菜单配置，便于当前默认内存模式下继续本地联调

## 2026-06-09
- 已把 `/Users/wangzhiwei/game/dialogue_demo` 中可直接复用的对话与运行态 UI 骨架迁入当前客户端：新增 `client/addons/dialogue_manager/`、`client/dialogue/`、`client/scenes/ui/`、`client/scripts/ui/`、`client/scripts/data/` 与 `client/data/`
- `client/project.godot` 已接入 `DialogueManager` 与 `SomeGlobal` 自动加载，并新增 `open_main_menu`、`open_player_panel`、`open_scene_npc_list` 输入动作；当前可直接呼出 demo 风格主菜单、角色面板、场景 NPC 列表与 NPC 菜单
- `client/scripts/bootstrap/main.gd` 已把新 UI 接入现有联机主运行态：附近 NPC 现在可通过列表选择并向服务端发起交互，服务端返回的 `menu_entries` 会用 demo 风格 NPC 菜单展示；当菜单项为本地 `talk`/`dialogue` 类型时，会直接走 `DialogueManager` 气泡框
- 状态面板头部称号与玩家名现已通过 `SomeGlobal`/`GameState` 联动填充；宠物状态页会优先读取当前联机宠物列表中的首只宠物，其余数值项缺省时继续回退到 demo 默认数据
- `client/scenes/bootstrap/main.tscn` 已移除登录后主运行态底部“世界操作区 / 战斗操作区”面板、按钮组和数据弹层，`GameplayArea` 现恢复为全屏显示世界/战斗内容
- `client/scripts/bootstrap/runtime_hud.gd` 已收敛为最小顶部状态条，只保留连接、场景、玩家文案和隐藏日志输出，不再承载宠物、编队、任务、背包或 NPC 菜单交互
- `client/scripts/bootstrap/main.gd` 已同步删除对底部操作区信号、自动摘要拉取和 NPC 菜单面板的依赖；当前收到相关交互负载时仅记录日志，不再弹出界面
- 本次未改动服务端协议与世界/战斗主链路，只移除客户端运行态中的世界操作区相关界面与脚本耦合

## 2026-05-20
- `client/autoload/http_client.gd` 已补上非 JSON、空响应和底层 HTTP 失败结果的容错处理，避免后端未启动或返回异常内容时直接触发 `JSON.parse_string()` 解析报错
- `client/scripts/common/command_ids.gd` 已为客户端协议消息号常量补齐说明性注释，当前各请求、响应和推送编号的用途更清晰
- `client/autoload/message_router.gd` 已为消息回调注册表、注册/注销和统一分发逻辑补齐说明性注释
- `client/autoload/http_client.gd` 已为基础地址、HTTP 请求节点、登录接口和通用 JSON 请求封装逻辑补齐说明性注释
- `client/scripts/feature/world/player.gd` 已为四方向移动、状态机切换、动画回退和切图/战斗锁定逻辑补齐说明性注释
- `client/scripts/feature/pet/pet_controller.gd` 已为宠物列表响应、宠物更新推送和编队设置响应的状态写回逻辑补齐说明性注释
- `client/scripts/feature/bag/bag_controller.gd` 已为背包列表响应和单物品更新推送的状态写回逻辑补齐说明性注释
- `client/scripts/feature/battle/battle_controller.gd` 已为交互响应、战斗开始/更新/结算推送的状态写回与事件广播逻辑补齐说明性注释
- `client/scripts/bootstrap/main.gd` 已为主运行态场景挂载、消息路由注册、HUD 刷新、世界/战斗视图切换和返回登录页流程补齐说明性注释
- `client/autoload/app.gd` 已为应用层启动编排、HTTP 登录、WebSocket 鉴权、推送处理和战斗动作上报入口补齐说明性注释
- `client/scripts/auth/login_scene.gd` 已为登录按钮链路、演示账号填充、登录页状态刷新和过渡动画流程补齐说明性注释
- `client/scripts/feature/battle/battle_scene.gd` 已为战斗界面刷新、技能按钮绑定、战斗事件文案生成和单位状态读取逻辑补齐说明性注释
- `client/scripts/bootstrap/runtime_hud.gd` 已为运行态 HUD 的常量、信号、节点引用、面板状态字段和主要渲染函数补齐说明性注释，当前宠物/编队/背包面板的职责与交互入口更清晰
- `client/scripts/feature/world/world_controller.gd` 已为场景配置、地图装载、固定镜头布局、门区切图、坐标换算和序列号生成逻辑补齐说明性注释，不改变现有地图切换与权威同步链路
- `client/autoload/net_client.gd` 已为连接状态、心跳调度、二进制封包解包、CRC32 校验和开发态 JSON 收发逻辑补齐说明性注释，便于后续继续维护网络层
- `client/autoload/game_state.gd` 已为会话状态、世界快照、宠物/编队、背包和战斗状态写入逻辑补齐说明性注释，保持现有状态合并与事件广播行为不变

## 2026-05-17
- `world_controller.gd` 已把固定镜头地图的角色出生显示点收敛为“地图可见内容中心”规则：当场景未显式配置 `spawn_local_position` 时，会自动按当前地图内容包围盒中心计算出生显示点
- `scene_id = 1` 的 `roxus_house` 已移除手写出生显示坐标，登录进入世界或权威重同步后，角色会默认显示在地图场景中心；后续新增固定镜头地图时也可直接复用同一规则
- `main.tscn` 的登录后主运行态上下分区已从 `4:1` 调整为 `3:1`：上部 `GameplayArea` 现占 `75%` 高度，下部 `HudRoot` 现占 `25%` 高度，继续保持游戏区与操作区互不遮挡
- 客户端设计分辨率已从 `1080x1920` 收敛回 `360x640`，并继续通过 `window/stretch` 在移动端按整数倍率自动放大适配；`main.tscn`、`world_scene.tscn`、`battle_scene.tscn`、`login_scene.tscn` 与 `runtime_hud.gd` 也已同步回收到小设计分辨率口径
- 客户端设计分辨率曾短暂收敛为 `240x320`，但由于与当前竖屏目标分辨率比例不一致，运行时整数倍率放大后清晰度下降；现已回退为 `360x640`，并同步恢复主运行态 `SubViewport`、世界层默认渲染尺寸、登录页、战斗卡片和底部 HUD 的对应尺寸口径
- 客户端早期占位地图文件已清理，只保留 `roxus_house` 作为当前地图资源；`world_controller.gd` 中对已删除占位地图的加载引用，以及 `roxus_house` 中通往已删除地图的出口门区也已同步移除
- 客户端地图切换现已重新接通 `scene_id = 1 <-> 2`：`world_controller.gd` 新增 `scene_id = 2 -> east_road_of_shanguang_town.tscn` 的地图映射，`roxus_house.tscn` 中门区现为 `portal_id = 1001 -> scene_id = 2`，`east_road_of_shanguang_town.tscn` 中回程门现为 `portal_id = 2001 -> scene_id = 1`
- 为修正正式地图门区无法稳定触发的问题，`scene_id = 2` 的客户端坐标基准现已改为贴合 `east_road_of_shanguang_town.tscn` 当前门区像素位置；服务端内存世界仓储中 `portal_id = 1001` 与 `portal_id = 2001` 的权威入口落点也已同步重标定，`go test ./server/internal/transport/ws` 通过

## 2026-05-16
- 新增 `backend/docs/kdjl-client-reference.md`，梳理逆向原版客户端 `/Users/wangzhiwei/study/kdjl` 中对当前 MVP 有参考价值的流程设计
- 文档聚焦登录前状态机、世界与战斗场景切换、宠物实例/编队/出战宠分层、战斗意图上报与服务端权威结算
- 文档同时明确原版中不应直接迁移的部分，包括文本 UI 协议、WAP 代理联网细节、旧资源协议与敏感信息处理方式
- 新增 `backend/docs/pet-lineup-battle-model.md`，把“宠物实例 / 编队 / 当前出战宠 / 战斗快照”四层模型固定为后续实现口径
- 新文档同步梳理了服务端模块边界、客户端 `GameState` 状态建议、协议补强方向和分步实现顺序
- 补齐 `PET_LIST_REQ/RESP` 与 `PET_LINEUP_SET_REQ/RESP` 的最小双端闭环，服务端新增 `pet_handler`、内存/PostgreSQL 仓储能力和相关 WebSocket 路由
- 客户端 `GameState` 的宠物合并主键从 `pet_id` 调整为 `pet_uid`，并在编队变更后自动同步 `in_lineup` 标记
- 同步更新 `backend/docs/protocol.md` 与 `backend/proto/pet/pet.proto`，使宠物列表和编队设置响应结构与当前实现一致
- 新增 `backend/docs/map-scene-loading.md`，把参考原版客户端后的地图切换加载方案落成当前项目的实现文档
- 文档明确了“世界层常驻、地图资源热切换、服务端权威切图、客户端按 `WORLD_RESYNC_PUSH` 装载地图”的实现口径，并给出分阶段实施顺序
- 客户端 `world_scene.tscn` 新增 `MapMount` 和最小地图加载遮罩，`world_controller.gd` 已接入 `scene_id -> scene_path` 地图挂载/卸载逻辑，并按 `WORLD_RESYNC_PUSH` 切换地图资源
- 新增三张最小地图占位骨架，作为当时地图切换链路与门区接入的早期占位资源
- 服务端内存版 `world_repo` 新增按来源地图决定入口落点的切图逻辑，不再把目标地图统一 `spawnPos` 当作落点，解决切图后角色总出现在地图中心的问题
- 同步更新世界切图测试与协议/设计文档，明确当前最小入口模型为“按来源地图选择目标地图入口落点”；`go test ./server/...` 已通过
- 继续补齐地图门区实例：服务端 `MOVE_INTENT_REQ` 已支持 `portal_id`，客户端地图场景已接入 `Area2D` 门区与 `MapPortal` 脚本，门区触发后会按 `portal_id` 发起权威切图
- 同步更新 `backend/proto/world/world.proto`、`backend/docs/protocol.md` 与 `backend/docs/map-scene-loading.md`，并新增无效 `portal_id` 的服务端测试；相关 GDScript/场景诊断与 `go test ./server/...` 已通过
- 客户端已移除边界触发切图链路，`player.gd` 不再检测地图边缘，`world_controller.gd` 只保留门区 `Area2D` 触发的地图切换
- 继续落地宠物战斗模型：`battle` 模块现已在 `BATTLE_START_PUSH` / `BATTLE_STATE_PUSH` 中显式返回 `active_actor_id`、`active_pet_uid`，并为战斗单位补充 `lineup_index`
- 客户端 `GameState` 与 `battle_scene.gd` 已按当前出战宠字段组织战斗展示和动作提交；`backend/proto/battle/battle.proto` 与协议文档已同步更新，`go test ./server/...` 通过
- 继续补齐核心模型闭环：服务端战斗结算后现已把主战宠最终 HP 回写到 `pet` 模块，并通过 `3011 PET_UPDATE_PUSH` 把更新后的宠物实例同步给客户端
- `pet` 模块新增宠物 HP 更新接口，内存仓储与 PostgreSQL 仓储都已补齐最小实现；客户端继续复用现有 `handle_pet_update()`，无需新增一套路由或 UI
- 战斗链路测试现已覆盖 `PET_UPDATE_PUSH` 与回写后的 `PET_LIST_RESP` 一致性校验，`go test ./server/...` 通过
- 客户端世界地图配置已将 `scene_id = 1` 的加载资源切换为 `client/scenes/maps/fashtown/roxus_house.tscn`，从而直接复用新建的 `roxus_house` 地图场景
- `roxus_house.tscn` 已补接最小门区节点，复用现有 `map_portal.gd` 脚本并配置为 `portal_id = 1001 -> scene_2`，同时增加可见出口标记，便于继续迭代地图细化
- `world_controller.gd` 已为 `roxus_house` 接入固定镜头模式：相机固定在当前视口中心，地图按实际可见内容自动居中并在必要时缩放到可完整显示，角色按地图内本地坐标移动，不再带动镜头平移
- 客户端主场景已拆成“上部游戏显示区 + 下部固定 HUD 区”两段布局：世界地图、地图切换、战斗场景都只在上部区域渲染，下部会永久显示 `client/asset/场景原图/闪光镇/时光小屋.png`
- `main.tscn` 已新增底部固定背景与 HUD 容器，并把现有状态面板、挑战按钮和日志区收敛到底部常驻区域；`main.gd` 与 `world_controller.gd` 已同步支持按上部游戏区域尺寸布局固定镜头地图
- 继续补充 `backend/docs/kdjl-client-reference.md`，新增“登录后主运行态分层布局”和“战斗层与常驻 UI 共存关系”两节，明确原客户端采用单主画布分层承载世界、战斗与底部常驻功能区的结构
- 新增 `backend/docs/main-runtime-ui-layout.md`，把当前项目登录后“上部游戏区 + 下部常驻 HUD 区”的主运行态 UI 结构单独沉淀为实现文档，并明确只覆盖当前 MVP 内的世界、战斗、宠物/编队与背包入口挂点
- 客户端新增 `runtime_hud.gd` 组件并接入 `main.tscn`，底部正式操作区现已补出 MVP 骨架：运行状态区、世界交互按钮、宠物/编队/背包入口按钮与日志区
- `main.gd` 现已通过 `RuntimeHud` 统一驱动底部 HUD，并在首次进入世界后自动请求宠物列表与背包列表，使底部入口计数能同步当前摘要数据
- `RuntimeHud` 现已新增最小数据面板：点击 `宠物`、`编队`、`背包` 按钮会打开对应摘要面板，并随 `GameState` 数据更新自动刷新内容；进入战斗时该面板会自动收起
- `RuntimeHud` 的数据面板已进一步升级为滚动卡片列表样式，并在 `编队` 面板中补上最小交互：支持加入/移除、上移/下移调整顺序，以及通过 `main.gd -> App.set_pet_lineup()` 提交完整编队
- 主场景布局继续保持明确的上下分区：上部 `GameplayArea` 占约 `4/5`，下部 `HudRoot` 占约 `1/5`；当前已改为上部天蓝色纯背景、下部淡红色纯背景，游戏画布和操作区不再互相遮挡
- 上部游戏区现已改为 `SubViewportContainer + SubViewport` 独立渲染：世界层与战斗层挂点都迁入子视口，`main.gd` 会同步子视口尺寸，从而修复根视口清屏色在顶部泄露导致的黑条问题
- 客户端现已按 `1080x1920` 新设计分辨率补齐主运行态适配：`main.tscn` 不再依赖旧的 `320/384/96` 固定尺寸，改为保持 `4:1` 上下比例并放大 HUD 字号与按钮尺寸；`world_controller.gd` 的固定视角地图允许在大屏中自动放大，`battle_scene.tscn` 与 `world_scene.tscn` 的旧加载提示/战斗卡片也已同步扩大
- 登录页 `login_scene.tscn` 也已按 `1080x1920` 适配：新增纯色底板、居中登录卡片，并整体放大标题、输入框、主按钮、状态文字与日志区，保持与当前大屏主运行态一致的可读性

## 2026-05-14
- 新增联机复刻版架构草案，明确客户端、服务端、同步和持久化边界
- 新增实时协议文档，固定包头、消息号分段和 HTTP/WS 令牌策略已定稿
- 新增双端消息路由文档，明确 server/client 消息处理职责
- 新增 `proto/` 初版协议草案，覆盖 auth、world、pet、battle、bag 五类消息
- 新增 PostgreSQL 最小表结构迁移脚本，覆盖账号、玩家、宠物、背包、编队、战斗记录
- 新增 Go 服务端骨架，覆盖 HTTP 登录、JWT 签发、`ws_token` 鉴权、WebSocket 会话与应用层心跳
- 新增内存版账号仓储与 `ws_token` 仓储，用于当前阶段的无数据库联调
- 新增协议包头编解码与基础测试，`go test ./server/...` 已通过
- 新增 `ENTER_WORLD_REQ` 链路，打通 `session -> player -> pet -> world` 的场景快照返回
- 新增内存版 `player/pet/world` 仓储，当前可返回演示角色、编队和单场景快照
- 新增 WebSocket 路由测试，已覆盖已鉴权进入世界与未鉴权拦截场景
- 新增 `MOVE_INTENT_REQ` 链路，已支持移动合法性校验、位置更新、移动回执和世界重同步
- 新增玩家位置更新能力，移动成功后再次进入世界会返回最新坐标
- 新增世界移动测试，已覆盖合法移动、非法越界移动与重同步场景
- 调整根目录结构，现已拆分为 `backend/` 服务端目录和 `client/` 客户端目录
- 当前 Go 工程、协议、文档和迁移脚本已全部迁入 `backend/`
- 新增 Godot 4 客户端骨架，补齐 `client/project.godot`、入口场景和可直接打开的最小工程结构
- 新增客户端 `autoload` 单例：`App.gd`、`HttpClient.gd`、`NetClient.gd`、`MessageRouter.gd`、`GameState.gd`
- 新增世界、宠物、战斗、背包控制器占位脚本，先把客户端模块边界与消息路由挂好
- 新增根目录 `.gitignore`，忽略本地 SkillHub 目录和 Godot 生成的 `.godot/` 目录
- 持久化方案从 MySQL 调整为 PostgreSQL，并同步改写初始化迁移脚本方言与字段定义
- 新增 PostgreSQL、Redis 配置骨架与示例环境变量
- 新增 PostgreSQL 账号/玩家/宠物仓储适配器，以及 Redis `ws_token` 仓储适配器骨架
- 新增仓储 provider 装配层，为后续统一切换到 PostgreSQL/Redis 持久化准备依赖注入入口
- 新增 `backend/server/configs/config.env` 实际配置文件，并支持启动时自动加载本地 env 文件
- 客户端主场景改为最小登录页，支持账号密码输入、状态展示与日志输出
- 客户端补齐 WebSocket 二进制包头编码、CRC32 校验、JSON 消息体编解码与基础心跳
- 客户端打通 `HTTP 登录 -> WS 鉴权 -> 进入世界` 主流程，登录后自动建立实时会话
- 客户端全局状态新增 `session_id`、`reconnect_token`、`heartbeat_sec` 与 `is_ws_authenticated` 追踪
- 客户端新增独立 `login_scene` 登录场景，并将项目启动入口切换到登录场景
- 客户端 `bootstrap/main` 收敛为登录后的世界入口，只负责世界挂载、消息路由与运行态状态展示
- 客户端登录场景与主场景新增淡入淡出切场景过渡，优化登录成功和掉线返回体验
- 客户端主场景顶部状态面板与底部日志区进一步压缩，更适配 `320x480` 小窗口
- 客户端角色在进入战斗场景前补齐三态状态机，当前支持待机、行走、战斗中三种状态
- 客户端战斗消息已驱动角色状态切换，战斗中会锁定移动并优先播放战斗态动画回退
- 客户端新增独立 `battle_scene` 战斗视图场景，收到 `BATTLE_START_PUSH` 后会挂载战斗场景
- 客户端收到 `BATTLE_RESULT_PUSH` 后会卸载战斗场景并返回世界视图，保持现有网络链路不重建
- 服务端新增最小 PvE 战斗模块，当前通过与附近 NPC 交互触发战斗，所有伤害、回合推进和胜负结算均在服务端完成
- 客户端主场景新增“挑战附近NPC”入口，战斗场景新增“普通攻击”按钮，客户端仅发送交互/动作意图，不参与数值计算
- 服务端战斗模块新增最小技能表与技能合法性校验，当前支持按 `skill_id` 驱动不同技能伤害
- 客户端战斗场景改为根据服务端下发的 `skill_ids` 动态展示技能按钮，继续保持只发技能意图
