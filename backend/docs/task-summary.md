# 任务总结

本次输出聚焦在线复刻版的基础骨架，完成了三部分设计落地：
- 协议层：定义固定包头、cmd 编号、关键消息边界
- 路由层：明确 server/client 双端消息分发与职责归属
- 存储层：给出可直接初始化的 PostgreSQL 最小表结构
- 服务端骨架：落地 HTTP 登录、JWT、`ws_token`、WebSocket 会话、心跳与基础路由
- 进入世界链路：落地 `ENTER_WORLD_REQ`，返回角色、场景、附近实体和编队快照
- 世界移动链路：落地 `MOVE_INTENT_REQ`，支持移动校验、位置更新、移动推送与重同步
- 目录重组：根目录拆分为 `backend/` 和 `client/`，当前后端工程整体归档到 `backend/`

设计上坚持以下约束：
- 客户端只提交意图，不提交结果
- 服务端拥有世界与战斗的最终权威
- 模板配置与玩家实例分离
- 世界同步和战斗同步隔离
- 当前服务端骨架使用内存仓储完成登录与会话验证，后续再切到 PostgreSQL/Redis
- 进入世界阶段只返回静态快照，不提前混入 AOI 广播和移动状态机
- 当前移动阶段只向请求方回推 `ENTITY_MOVE_PUSH`，AOI 对其他玩家的广播仍在下一阶段实现
- 此前 `client/` 仅保留空目录占位，当前已补齐可直接打开的 Godot 客户端骨架

建议的下一步实现顺序：
1. 生成 protobuf 代码，并把当前 auth/session JSON 消息体切换到 protobuf
2. 接入 PostgreSQL driver 与 Redis client，打通 `postgres_redis` 模式并替换当前内存版账号仓储与 `ws_token` 仓储
3. 在已完成的移动基础上，继续落 AOI 可见集和对其他玩家的移动广播
4. 落宠物实例、编队、战斗状态机
5. 落断线重连、限流与统一错误码映射

## 2026-05-14 客户端骨架补充

本次补充聚焦 Godot 客户端最小可开发骨架，目标是让 `client/` 可以直接被 Godot 4 打开并继续迭代：
- 初始化 `client/project.godot`、入口场景、图标和基础目录结构
- 按架构草案落地 `autoload` 层：`App`、`HttpClient`、`NetClient`、`MessageRouter`、`GameState`
- 预留世界、宠物、战斗、背包四个客户端控制器，并把消息号路由挂接到对应模块
- 当前 HTTP 登录已接好 `POST /api/v1/auth/login` 的调用封装
- 当前 WebSocket 只完成连接与开发期 JSON 路由骨架，二进制包头、protobuf 编解码和正式鉴权仍是下一步工作
- 增加 `.gitignore`，避免本地 SkillHub 目录和 Godot 生成目录进入版本库
- 当前持久化方案已统一切到 PostgreSQL，初始化 SQL 脚本已同步改写为 PostgreSQL 方言

## 2026-05-14 存储骨架补充

本次补充聚焦服务端真实存储切换前的骨架准备，先把配置、仓储适配器和装配边界补齐：
- 新增 `PP_REPOSITORY_MODE`、PostgreSQL、Redis 相关配置项，并补充示例环境变量
- 新增 PostgreSQL 版账号、玩家、宠物仓储适配器，统一复用现有模块仓储接口
- 新增 Redis 版 `ws_token` 仓储适配器，使用 key 前缀和一次性消费语义预留真实接入点
- 新增 provider 装配层，统一管理 memory 与 `postgres_redis` 两种仓储模式的依赖绑定
- 当前 `postgres_redis` 模式只完成骨架与接口约束，真实数据库连接、Redis 客户端初始化和驱动导入仍是下一步工作
- 新增 `config.env` 自动加载能力，后续只需要改 `backend/server/configs/config.env` 和切换 `PP_REPOSITORY_MODE` 即可接入真实服务

## 2026-05-14 登录页与登录链路补充

本次补充聚焦 Godot 客户端首个可用登录入口，目标是把现有 HTTP 登录骨架升级为可直接联调的完整登录流程：
- 主场景 UI 从调试面板收敛为最小登录页，保留账号、密码、状态、场景、玩家和日志展示
- 登录按钮触发 `HTTP 登录 -> WebSocket 连接 -> WS_AUTH_REQ -> ENTER_WORLD_REQ` 串行流程
- `NetClient` 补齐固定包头编码、CRC32 校验、二进制包解析与按序号发送能力
- `App` 增加 WebSocket 打开后自动鉴权、鉴权成功后缓存会话并启动心跳的编排逻辑
- `GameState` 补充 `session_id`、`reconnect_token`、`heartbeat_sec`、`is_ws_authenticated` 等会话状态字段
- 现阶段服务端登录接口无需调整，客户端已对齐当前后端的 JSON 消息体和二进制包结构
- 已完成 GDScript 诊断检查、服务端 `go test ./...` 验证以及运行期无报错启动检查

## 2026-05-14 登录场景拆分

本次补充聚焦客户端场景职责收敛，把登录流程从主场景拆成独立入口：
- 新增 `res://scenes/auth/login_scene.tscn` 与对应脚本，专职处理账号密码输入、HTTP 登录、WS 连接与鉴权反馈
- 项目启动入口调整为登录场景，应用启动后先进入登录页，再在鉴权成功后切换到主场景
- `bootstrap/main` 不再承担登录表单职责，当前只负责世界场景挂载、消息路由注册、状态展示与进入世界请求
- 场景切换过程中保留已建立的 WebSocket 会话，避免登录成功后重复认证
- 已完成新旧场景和启动配置的诊断检查，当前无新增 GDScript 或场景报错

## 2026-05-14 登录转场与主场景 HUD 微调

本次补充聚焦登录切换体验与小窗口界面密度：
- 登录场景和主场景均新增全屏遮罩过渡层，当前使用轻量淡入淡出转场，不引入额外资源和依赖
- 登录成功切主场景、主场景掉线返回登录场景时都会经过同一套黑场过渡，减少场景切换突兀感
- 主场景顶部状态面板进一步缩小，保留连接、场景、玩家三类核心信息，尽量不遮挡游戏画面
- 主场景底部日志面板高度同步压缩，继续保留联调可见性但降低运行态占屏
- 已完成场景与脚本诊断、运行态重启检查，当前无新增报错

## 2026-05-15 角色三态状态机补充

本次补充聚焦角色在进入战斗场景前的运行态约束，先把世界内角色状态机补齐：
- `player.gd` 从原先仅依赖输入方向的二态逻辑升级为显式三态：待机、行走、战斗中
- 战斗中状态会锁定角色移动输入，并优先尝试播放 `battle_*` 动画；若资源未补齐，则回退到同朝向待机动画
- `GameState` 新增 `is_in_battle`，用于在世界层和角色层共享当前是否处于战斗中的状态
- `battle_controller.gd` 在战斗开始/进行中时置为战斗态，在战斗结果到达时退出战斗态
- `world_controller.gd` 监听战斗状态变化并同步给本地角色，保证角色表现与战斗入口状态一致
- 已完成相关脚本诊断检查，当前无新增报错

## 2026-05-15 战斗视图场景接入

本次补充聚焦“进入战斗场景”和“战斗结束返回世界”的最小可用链路：
- 新增 `res://scenes/battle/battle_scene.tscn` 作为独立战斗视图场景，并配套 `battle_scene.gd` 做基础信息展示
- `battle_controller.gd` 补充 `battle_started`、`battle_finished` 信号，用于通知主场景进入和退出战斗视图
- 主场景新增 `BattleMount` 容器，在收到 `BATTLE_START_PUSH` 时挂载战斗视图，在收到 `BATTLE_RESULT_PUSH` 时卸载并回到世界视图
- 该实现保留主场景根节点和现有消息路由，不使用整棵树 `change_scene`，从而避免战斗期间网络链路和路由中断
- 战斗进行中会隐藏世界层显示，战斗结束后恢复世界层显示，并继续复用已有世界快照状态
- 已完成相关场景/脚本诊断及运行日志检查，当前无新增报错

## 2026-05-15 服务端权威最小战斗闭环

本次补充聚焦“多人联机场景下所有战斗计算必须由服务端负责”的约束，完成了第一版可跑闭环：
- 服务端新增 `battle` 模块，以玩家当前主战宠对战附近 NPC 的最小 PvE 模型管理单场战斗状态
- 世界内通过 `INTERACT_REQ` 申请与附近 NPC 交互开战，服务端校验会话、玩家、阵容和附近实体后返回 `BATTLE_START_PUSH`
- 战斗内客户端只会提交 `BATTLE_ACTION_REQ` 动作意图，当前最小实现支持普通攻击和逃跑，其中伤害、回合推进和结算全部在服务端完成
- 服务端每次动作处理后会返回 `BATTLE_ACTION_RESP`，并按结果推送 `BATTLE_STATE_PUSH` 与 `BATTLE_RESULT_PUSH`
- 客户端主场景新增“挑战附近NPC”入口，战斗视图新增“普通攻击”按钮；客户端仅负责展示状态和提交意图，不做本地数值计算
- `GameState` 的战斗状态同步改为增量合并，保证战斗开始快照与后续状态推送可以共同驱动 UI
- 已补充协议文档、WebSocket 路由测试和 `go test ./...` 验证，当前服务端测试通过，客户端脚本/场景诊断无报错

## 2026-05-15 最小技能模型与技能按钮

本次补充聚焦战斗动作从“单一普通攻击”升级到“按技能意图提交”：
- 服务端 `battle` 模块新增最小技能表，当前内置玩家和敌方各两种技能，并由服务端按 `skill_id` 计算不同伤害值
- 服务端会校验提交的 `skill_id` 是否属于当前出战单位可用技能，非法技能请求将直接拒绝
- 敌方行动改为按回合轮换自身技能表，不再固定使用单一伤害模板
- 客户端战斗场景改为根据 `BATTLE_START_PUSH` 下发的 `skill_ids` 动态展示技能按钮，而不是写死一个攻击按钮
- 技能按钮点击后仅发送 `BATTLE_ACTION_REQ`，本地不做伤害、命中或回合推进推导，继续保持服务端权威
- 已补充战斗路由测试以覆盖多技能快照和技能动作联调，`go test ./...` 通过，战斗场景诊断无报错

## 2026-05-16 原版客户端参考逻辑沉淀

本次补充聚焦把逆向出来的原版客户端 `/Users/wangzhiwei/study/kdjl` 中可复用的流程设计沉淀为当前项目文档：
- 新增 `backend/docs/kdjl-client-reference.md`，只保留与当前 MVP 直接相关的参考逻辑，不扩展公会、交易、活动等边界外能力
- 文档确认原版最值得吸收的是登录前状态机、登录上下文本地持久化、世界/战斗场景切换关系、地图入口意图上报、战斗意图提交与服务端结算边界
- 文档明确原版协议和 UI 技术只适合参考思路，不适合直接迁移，包括文本协议、服务端驱动 `<menu>/<input>`、WAP 代理联网和敏感信息缓存
- 文档补齐了逆向类与当前项目模块的映射，便于后续在 `client` 与 `backend/server/internal/module/*` 中按现有架构落地
- 本次任务只新增文档与记录，不改动现有双端功能链路

## 2026-05-16 宠物编队与战斗快照模型设计

本次补充聚焦把上一步的原版参考结论进一步收敛成可直接指导实现的模型文档：
- 新增 `backend/docs/pet-lineup-battle-model.md`，把后续实现必须区分的四层对象固定为 `PetInstance`、`Lineup`、`ActivePet`、`BattleActorSnapshot`
- 文档结合当前仓库现状，明确 `pet`、`player`、`battle` 三个模块各自负责什么，不允许把宠物持久化状态、编队顺序和战斗运行态混在一起
- 文档补充客户端 `GameState` 的建议状态结构，明确 `pets`、`lineup`、`battle_state` 的边界，并指出当前 `upsert_pet()` 以 `pet_id` 合并的风险
- 文档补充了 `PET_LIST_RESP`、`PET_LINEUP_SET_REQ/RESP`、`BATTLE_START_PUSH`、`BATTLE_STATE_PUSH` 的后续补强方向，便于后面按最小代价逐步落实现有骨架
- 文档给出建议实现顺序：先补完整宠物实例，再补编队闭环，再显式化当前出战宠，最后再做战斗结算回写与换宠

## 2026-05-16 宠物列表与编队设置最小闭环

本次补充聚焦把上一条模型设计落成第一批最小代码改动：
- 服务端新增 `pet_handler.go`，正式接入 `PET_LIST_REQ` 与 `PET_LINEUP_SET_REQ` 两条 WebSocket 链路，并接入路由与应用启动装配
- `pet` 模块补齐了宠物实例模型、宠物列表查询、编队设置校验和仓储接口；内存仓储新增演示宠物列表，PostgreSQL 仓储新增宠物列表查询与编队写入能力
- `PET_LIST_RESP` 现已返回 `pets + lineup`，`PET_LINEUP_SET_RESP` 现已返回 `accepted + lineup + reason`，避免客户端收到编队变更后还要二次查详情
- 客户端 `GameState.upsert_pet()` 改为按 `pet_uid` 合并，解决同种宠物多只并存时被错误覆盖的问题；`set_pets()` / `set_lineup()` 现在会自动同步 `in_lineup`
- 客户端 `App.gd` 新增 `set_pet_lineup()` 发送入口，`pet_controller.gd` 仅在服务端确认成功后才更新本地编队，避免失败响应把本地状态误清空
- 协议文档和 `backend/proto/pet/pet.proto` 已同步更新；已执行 `go test ./server/...`，并完成相关 GDScript 诊断检查，当前无新增报错

## 2026-05-16 地图切换加载方案沉淀

本次补充聚焦把“参考原版客户端如何做地图切换加载”的方案落到当前仓库文档：
- 新增 `backend/docs/map-scene-loading.md`，明确世界层与战斗层分离、地图资源热切换、服务端权威切图、客户端按 `MOVE_INTENT_REQ -> MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH` 时序装载地图
- 文档对照当前 `world_controller.gd`、`main.gd` 和服务端 `world_handler.go`，说明现有可复用骨架与当前缺口，避免后续为了切图重写整套世界链路
- 文档给出推荐场景结构：`WorldRoot -> MapMount / RemoteEntities / LocalPlayerAnchor`，要求 `main.tscn` 和 `world_scene.tscn` 常驻，只替换地图节点
- 文档给出地图配置、门区切换、加载遮罩和分阶段实施顺序，便于后续按最小代价推进地图绘制与切图接入
- 本次仅新增设计文档和记录，不改动现有双端运行代码

## 2026-05-16 世界地图资源挂载第一阶段

本次补充聚焦把地图切换加载方案先落成客户端第一阶段的最小实现：
- `client/scenes/world/world_scene.tscn` 新增 `MapMount` 挂载点和最小 `MapLoadingOverlay`，保证世界根场景常驻，只替换地图资源节点
- `client/scripts/feature/world/world_controller.gd` 为 `SCENE_CONFIGS` 增加 `scene_path`，并新增地图资源加载、卸载和切图加载态控制逻辑
- 客户端现在会在收到服务端世界快照时按当前 `scene_id` 装载对应地图资源；地图切换仍然沿用 `MOVE_INTENT_REQ -> MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH`，没有改变服务端权威链路
- `client/scripts/feature/world/player.gd` 继续只负责角色移动和战斗锁定，不承担地图切换判定
- 新增 `client/scenes/maps/scene_1.tscn`、`scene_2.tscn`、`scene_3.tscn` 三张最小地图骨架，当前仅提供视觉占位和出入口提示，便于后续逐张替换成正式地图
- 已对相关 GDScript 和 `.tscn` 文件完成诊断检查，当前无新增报错

## 2026-05-16 地图入口落点修正

本次补充聚焦修正“切图后角色总出现在新地图中心”的问题：
- 根因是服务端内存版 `world_repo` 在场景切换时统一使用目标地图 `spawnPos` 作为落点，导致无论从哪边进入都落在固定中心参考点附近
- 当前最小实现已改为“按来源地图决定目标地图入口落点”：例如 `1 -> 2` 会落在 `2` 号地图左入口，`2 -> 1` 会落在 `1` 号地图右入口，`2 -> 3` 会落在 `3` 号地图左入口
- 这次没有扩协议字段，仍沿用 `target_scene_id`；因为当前每对相邻地图只有一个入口，最小规则足够支撑现阶段地图切换
- 同步更新 `backend/docs/protocol.md` 与 `backend/docs/map-scene-loading.md`，把 `corrected_pos` / `self_pos` 的口径明确为“权威入口落点”，不再写成统一出生点
- 已更新 `world_handler_test.go` 的切图断言，并执行 `go test ./server/...`，当前通过

## 2026-05-16 地图门区与 portal_id 闭环

本次补充聚焦把“入口落点”进一步落成真正的门/入口实例：
- 服务端 `protocol.MoveIntentReq`、`world.Service` 与内存版 `world_repo` 已补充 `portal_id`，当前会优先按门区配置决定目标地图和入口落点；若 `portal_id` 无效则拒绝切图
- `client/scripts/feature/world/map_portal.gd` 新增为最小门区脚本，地图场景中的 `Area2D` 门区进入后会发出 `portal_id + target_scene_id`，再由 `world_controller.gd` 统一走现有权威切图链路
- 三张占位地图场景已接入门区节点：`scene_1` 右门通往 `scene_2`，`scene_2` 左右门分别通往 `scene_1/scene_3`，`scene_3` 左门通往 `scene_2`
- `world_controller.gd` 新增门区绑定与切图冷却，避免玩家刚落在入口附近时立即再次触发反向传送，并彻底移除了边界触发切图逻辑
- 同步更新 `backend/proto/world/world.proto`、`backend/docs/protocol.md` 与 `backend/docs/map-scene-loading.md`，让协议草案、实现文档和当前代码保持一致
- 已新增无效 `portal_id` 的服务端测试，执行 `go test ./server/...` 通过；相关 GDScript 与地图场景诊断无新增报错

## 2026-05-16 当前出战宠显式化

本次补充聚焦把宠物战斗模型文档里“显式化当前出战宠”这一步真正落成代码：
- 服务端 `battle` 模块的运行时快照已补充 `active_actor_id`、`active_pet_uid`，并为 `BattleActorSnapshot` 增加 `lineup_index`，使“当前出战宠”和“战斗单位快照”不再隐含耦合在数组第一位
- `BATTLE_START_PUSH` 与 `BATTLE_STATE_PUSH` 现在都会下发当前出战宠锚点，客户端不需要再默认用 `allies[0]` 猜测当前己方在场宠物
- 客户端 `GameState` 新增 `active_battle_actor()` 辅助方法，`battle_scene.gd` 改为按 `active_actor_id` / `active_pet_uid` 组织我方显示和动作提交，为后续换宠留稳定接口
- 同步更新 `backend/proto/battle/battle.proto` 与 `backend/docs/protocol.md` 的战斗快照结构，确保协议草案、文档说明和当前 JSON 实现一致
- 已补充战斗链路测试，校验 `BATTLE_START_PUSH` 与 `BATTLE_STATE_PUSH` 中的 `active_actor_id`、`active_pet_uid`、`lineup_index`；执行 `go test ./server/...` 通过

## 2026-05-16 战斗结束主战宠 HP 回写

本次补充聚焦把宠物战斗模型文档里“战斗结束回写主战宠 HP”这一步真正落成最小闭环：
- 服务端 `pet` 模块新增宠物 HP 更新接口，`memory` 与 `postgres` 两套仓储均已支持按 `player_id + pet_uid` 回写当前 HP
- 服务端 `battle` 结算结果现已显式带出主战宠 `pet_uid` 与最终 HP，`battle_handler` 会在发送战斗结果时先回写宠物实例，再通过 `3011 PET_UPDATE_PUSH` 推送最新宠物详情
- 客户端继续复用现有 `pet_controller.gd` 的 `handle_pet_update()`，按 `pet_uid` 合并本地宠物实例，不新增额外路由与 UI 逻辑
- 协议文档已补充 `PET_UPDATE_PUSH` 消息体，并明确当前 `BATTLE_RESULT_PUSH` 之后可能继续跟随宠物更新推送
- 已扩展 `world_handler_test.go`，同时校验 `PET_UPDATE_PUSH` 内容与回写后 `PET_LIST_RESP` / `lineup` 的 HP 一致性；执行 `go test ./server/...` 通过
