# 任务总结

## 2026-06-10 回合战斗公式层第一轮补全

本次补充聚焦把回合战斗 MVP 中最简化的直接伤害计算，收敛成一个可继续扩展的独立公式层：
- 新增 `backend/server/internal/module/battle/formula.go`，把基础伤害合成、防御减伤、最终伤害、治疗量、暴击上下限等规则从 `service.go` 中拆出
- 当前公式层已支持攻击/防御/速度/目标当前生命/固定伤害合成，支持穿甲、易伤入口与 90% 防御减伤上限
- 当前暴击链路已补上 100% 暴击率上限、2000% 暴击伤害上限，并明确“纯固定伤害默认不暴击”
- 新增 `backend/server/internal/module/battle/formula_test.go`，覆盖基础伤害构成、防御修正、暴击边界、治疗量和固定伤害不暴击等关键规则
- 已执行 `cd backend && GOCACHE=/private/tmp/pocket-pet-gocache go test ./server/internal/module/battle ./server/internal/transport/ws` 与 `cd backend && GOCACHE=/private/tmp/pocket-pet-gocache go test ./server/...`，当前通过

本次继续补充第二轮公式能力：
- `backend/server/internal/module/battle/formula.go` 继续接入法力系数、有效属性倍率入口与格挡减伤入口
- `backend/server/internal/module/battle/service.go` 的战斗单位运行态新增 `mana` 与属性倍率/平值修正字段，为后续状态系统和被动系统预留承接点
- 新增 `backend/server/migrations/007_add_pet_mana.sql`，为 `player_pet` 增加 `mana` 列，并回填演示账号宠物的默认法力值
- `backend/server/internal/data/postgres/pet_repo.go` 与 `backend/server/internal/teststub/repos.go` 已同步接入 `mana`，避免 PostgreSQL 仓储和测试桩公式口径漂移

本次继续补充第三轮“状态驱动公式”能力：
- `backend/server/internal/module/battle/model.go` 新增 `易伤 / 破甲 / 减速 / 暴击提升` 状态编号
- `backend/server/internal/module/battle/service.go` 已增加 `refreshStatusDerivedModifiers()`，让状态在施加、覆盖和到期时真实回写到速度倍率、暴击率、破甲与易伤等派生战斗属性
- 当前技能表中，`火花冲击` 会附加易伤，`利爪突袭` 会附加减速，`活力治愈` 会附加暴击提升，便于先验证“状态影响公式”的闭环
- `backend/server/internal/module/battle/formula_test.go` 已新增状态驱动公式测试，覆盖状态生效与到期恢复

本次继续补充第四轮状态能力：
- `backend/server/internal/module/battle/model.go` 新增 `诅咒 / 束缚 / 沉睡 / 麻痹 / 混乱` 状态编号
- `backend/server/internal/module/battle/service.go` 已把诅咒加入持续伤害结算，并把束缚/沉睡/麻痹统一接入“跳过行动”判定
- 当前混乱会在服务端执行阶段强制改写目标，随机命中除自身外的任意存活单位，保持权威结算边界
- `backend/server/internal/module/battle/formula_test.go` 已补充诅咒 tick、控制状态阻断与混乱目标改写测试

本次修正持续伤害结算时机：
- `backend/server/internal/module/battle/service.go` 已将流血、诅咒等被动扣血从全体回合末改为“该单位回合结束”结算，跳过行动也会进入自身回合结束结算
- 持续伤害事件仍使用 `EventTypeStatusTick`，不走 `resolveDamageSkill()`，因此不会触发吸血、反击、连击等命中后被动
- `backend/server/internal/module/battle/formula_test.go` 已补充事件顺序测试，锁定高速度单位行动后立即结算自身流血，再进入下一个单位行动

本次继续补充第五轮被动能力：
- `backend/server/internal/module/battle/service.go` 已新增服务端权威的 `闪避 / 吸血 / 反击` 结算分支，并统一收口到 `resolveDamageSkill()`
- 当前被动仍先使用演示型运行时配置验证链路：101 号宠物默认吸血、102 号宠物默认闪避、敌方默认有一定反击概率
- `backend/server/internal/module/battle/formula_test.go` 已新增被动测试，覆盖闪避拦截命中、吸血自我恢复和反击回打
- `client/scripts/feature/battle/battle_scene.gd` 已补充新事件类型文案，避免移动端战斗面板只显示通用占位文本

本次继续补充第六轮被动能力：
- `backend/server/internal/module/battle/service.go` 已新增服务端权威的 `连击 / 复活 / 控制免疫` 结算分支
- 当前 101 号宠物默认有连击概率，102 号宠物默认具备一次复活与控制免疫，用于验证完整被动主链
- 持续伤害致死现在也会先尝试走复活分支，避免复活只覆盖直伤而漏掉状态结算
- `backend/server/internal/module/battle/formula_test.go` 已补充连击额外攻击、复活打断死亡与控制免疫拦截控制状态测试

## 2026-06-10 回合战斗开发任务清单入库

本次补充聚焦把当前回合战斗的后续开发计划沉淀进仓库，方便重启会话后继续推进：
- 新增 `docs/回合战斗开发任务清单.md`，整理了“当前已完成能力、未完成能力、推荐实施顺序、下一步任务拆分”
- 文档明确当前实现仍是“PVE 回合战斗 MVP”，不是原始战斗规格文档中的全量实现
- 文档把后续开发拆成规则补全、服务端权威托管、PVE 奖励闭环、客户端站位/表现层、PVP/组队 PVP 五个大阶段
- 后续继续开发时，建议先阅读 `docs/回合战斗开发文档.md` 与 `docs/回合战斗开发任务清单.md` 再进入具体代码实现

## 2026-05-20 登录接口异常响应容错修复

本次补充聚焦修正“后端未启动时客户端登录直接刷 JSON 解析错误”的问题：
- `client/autoload/http_client.gd` 已为底层 HTTP 失败结果、空响应体和非 JSON 响应补上显式容错处理，不再直接对异常内容调用 `JSON.parse_string()`
- 当前 `HTTPRequest.request_completed` 会先检查底层请求结果，再检查响应体是否为空，最后才尝试解析 JSON 字典
- 当后端未启动或返回非 JSON 内容时，客户端现在会返回统一错误字典，而不是在控制台刷出 JSON 解析报错
- 已重新启动 `backend/server/cmd/game-server`，并确认 `POST /api/v1/auth/login` 当前返回 `200` 与标准 JSON 结构

## 2026-05-20 客户端核心脚本注释补齐

本次补充聚焦把客户端核心运行链路中最常看的四个脚本按当前项目规则补上说明性注释：
- `client/scripts/bootstrap/runtime_hud.gd` 已为运行态 HUD 的常量、信号、节点引用、面板状态字段、数据卡渲染和编队编辑流程补齐注释
- `client/scripts/feature/world/world_controller.gd` 已为场景配置、权威快照应用、固定镜头布局、门区切图和本地坐标换算补齐注释
- `client/autoload/net_client.gd` 已为连接状态、心跳调度、正式链路封包解包、CRC32 校验和开发态文本协议分发补齐注释
- `client/autoload/game_state.gd` 已为会话状态、世界快照、附近实体、宠物/编队、背包和战斗状态合并逻辑补齐注释
- 本次没有调整任何协议字段、状态结构、消息流转或游戏表现，仅增加注释说明并通过相关 GDScript 诊断检查

本次继续补充第二批核心脚本的说明性注释：
- `client/scripts/bootstrap/main.gd` 已为主运行态场景挂载、消息路由注册、HUD 刷新、世界/战斗视图切换和返回登录页流程补齐注释
- `client/autoload/app.gd` 已为应用层启动编排、HTTP 登录、WebSocket 鉴权、提示推送和战斗动作上报入口补齐注释
- `client/scripts/auth/login_scene.gd` 已为登录链路、登录页状态刷新、演示账号填充和场景切换过渡补齐注释
- `client/scripts/feature/battle/battle_scene.gd` 已为战斗界面刷新、技能按钮状态、战斗事件文案生成和单位状态读取逻辑补齐注释
- 第二批脚本同样只增加注释说明，不改动现有协议、状态结构和交互行为，并通过相关 GDScript 诊断检查

本次继续补充第三批运行态薄控制脚本的说明性注释：
- `client/scripts/feature/world/player.gd` 已为四方向移动、状态机切换、动画回退和切图/战斗锁定逻辑补齐注释
- `client/scripts/feature/pet/pet_controller.gd` 已为宠物列表响应、宠物更新推送和编队设置响应的状态写回逻辑补齐注释
- `client/scripts/feature/bag/bag_controller.gd` 已为背包列表响应和单物品更新推送的状态写回逻辑补齐注释
- `client/scripts/feature/battle/battle_controller.gd` 已为交互响应、战斗开始/更新/结算推送的状态写回与事件广播逻辑补齐注释
- 第三批脚本继续保持“只补注释、不改行为”的原则，并通过相关 GDScript 诊断检查

本次补充最后一批客户端基础设施脚本的说明性注释：
- `client/scripts/common/command_ids.gd` 已为客户端协议消息号常量补齐注释，明确各请求、响应和推送编号的用途
- `client/autoload/message_router.gd` 已为消息回调注册表、注册/注销和统一分发逻辑补齐注释
- `client/autoload/http_client.gd` 已为基础地址、登录接口和通用 JSON 请求封装逻辑补齐注释
- 最后一批脚本同样只增加注释说明，不改动现有协议、网络链路和状态结构，并通过相关 GDScript 诊断检查

## 2026-05-17 固定镜头地图出生点居中

本次补充聚焦把登录后角色在固定镜头地图中的出生显示点统一收敛到地图场景中心：
- `client/scripts/feature/world/world_controller.gd` 不再把 `scene_id = 1` 的出生显示点写死为单独的 `spawn_local_position`
- 固定镜头地图现在会优先读取显式配置；如果未配置，则自动按当前地图可见内容包围盒中心计算出生显示点
- 因此登录进入世界、收到权威世界重同步、以及后续切回固定镜头地图时，角色都会默认显示在对应地图场景中心
- 非固定镜头地图原有“出生逻辑坐标映射到视口中心”的链路保持不变，没有扩散修改现有服务端世界权威坐标规则

## 2026-05-17 主运行态改为 3:1 上下布局

本次补充聚焦把登录后主运行态的游戏区与操作区比例从 `4:1` 调整为 `3:1`：
- `client/scenes/bootstrap/main.tscn` 中 `GameplayArea.anchor_bottom` 已从 `0.8` 调整为 `0.75`
- `client/scenes/bootstrap/main.tscn` 中 `HudRoot.anchor_top` 已从 `0.8` 调整为 `0.75`
- 当前上部游戏区占 `75%` 高度，下部操作区占 `25%` 高度，世界、战斗与底部 HUD 的现有链路保持不变
- 本次只调整布局比例，不改动世界渲染、战斗挂载、协议、控制器和底部 HUD 交互逻辑

## 2026-05-17 设计分辨率收敛回 360x640

本次补充聚焦把客户端从大设计分辨率切回更适合像素地图编辑的小设计分辨率，并继续依赖运行时自动拉伸：
- `client/project.godot` 的设计分辨率与窗口覆盖尺寸已从 `1080x1920` 调整为 `360x640`，同时继续保留 `canvas_items + expand + integer` 的移动端适配方式
- `client/scripts/feature/world/world_controller.gd` 与 `client/scenes/world/world_scene.tscn` 已同步把世界层默认渲染尺寸收敛为 `360x480`，与当前 `3:1` 的主运行态上部游戏区一致
- `client/scenes/bootstrap/main.tscn`、`client/scripts/bootstrap/runtime_hud.gd`、`client/scenes/battle/battle_scene.tscn` 与 `client/scenes/auth/login_scene.tscn` 已把此前按大屏放大的字号、面板和按钮尺寸同步收回到小设计分辨率口径
- 当前思路改为“编辑期按小设计分辨率绘制像素地图，运行期由 Godot 按整数倍率自动放大”，不再需要为每张地图单独做统一缩放改造

## 2026-05-17 240x320 方案回退为 360x640

本次补充聚焦修正 `240x320` 设计分辨率导致的运行时发糊问题，并把客户端口径恢复为更适合当前竖屏目标分辨率的 `360x640`：
- `client/project.godot` 已把设计分辨率与窗口覆盖尺寸从 `240x320` 回退为 `360x640`，继续保留 `canvas_items + expand + integer` 的移动端整数倍率拉伸方式
- `client/scripts/feature/world/world_controller.gd` 与 `client/scenes/world/world_scene.tscn` 已同步把世界层默认渲染尺寸恢复为 `360x480`，重新匹配当前主运行态 `3:1` 布局下的上部游戏区
- `client/scenes/bootstrap/main.tscn` 与 `client/scripts/bootstrap/runtime_hud.gd` 已把底部 HUD 的字号、按钮高度、边距和数据面板尺寸恢复到 `360x640` 口径，避免 `240x320` 下过度压缩
- `client/scenes/auth/login_scene.tscn` 与 `client/scenes/battle/battle_scene.tscn` 也已同步恢复卡片、输入框、按钮与文本尺寸，使登录页和战斗界面在当前清晰度优先的方案下保持可读性

## 2026-05-17 清理早期占位地图文件

本次补充聚焦把客户端早期联调用的占位地图文件和对应引用一起清理掉：
- 三张早期占位地图场景已从仓库中删除
- `client/scripts/feature/world/world_controller.gd` 已移除对已删除占位地图的 `SCENE_CONFIGS` 加载路径，只保留当前正式接入的 `roxus_house`
- `client/scenes/maps/fashtown/roxus_house.tscn` 中通往已删除占位地图的出口门区也已同步移除，避免客户端继续发起无效切图
- `backend/docs/changelog.md` 与 `backend/docs/map-scene-loading.md` 已同步清理旧文件路径说明，避免文档继续指向已删除资源

## 2026-05-17 重新接通 roxus_house 与 east_road 双向切图

本次补充聚焦把两张正式地图重新接回当前服务端权威门区切图链路：
- `client/scripts/feature/world/world_controller.gd` 已重新补上 `scene_id = 2` 的地图映射，当前会加载 `client/scenes/maps/fashtown/east_road_of_shanguang_town.tscn`
- `client/scenes/maps/fashtown/roxus_house.tscn` 中新增的 `MapPortal` 现已补齐 `portal_id = 1001` 与 `target_scene_id = 2`，踩中后会沿用现有 `MOVE_INTENT_REQ -> MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH` 链路切到 `east_road`
- `client/scenes/maps/fashtown/east_road_of_shanguang_town.tscn` 中的回程门现已补齐 `portal_id = 2001` 与 `target_scene_id = 1`，踩中后会返回 `roxus_house`
- 本次没有改动客户端门区脚本和协议结构，只是把现有 `MapPortal` 与服务端内存世界仓储中已经存在的 `portal_id 1001/2001` 重新对接起来

## 2026-05-17 正式地图门区入口坐标重标定

本次补充聚焦修正“人物看起来踩进门区，但没有稳定触发切图”的问题：
- `client/scripts/feature/world/world_controller.gd` 中 `scene_id = 2` 的 `spawn` 现已从旧占位地图时代的 `(2,4)` 调整为更贴合 `east_road_of_shanguang_town.tscn` 当前门区像素位置的本地坐标基准
- `backend/server/internal/data/memory/world_repo.go` 中 `portal_id = 1001` 的目标落点已调整到 `east_road` 左侧入口附近，`portal_id = 2001` 的目标落点已调整到 `roxus_house` 底部门区附近
- 同一文件里 `scene 1 <- 2` 与 `scene 2 <- 1` 的兼容入口落点也已同步修正，避免未携带 `portal_id` 的兼容切图仍然落到旧占位地图坐标
- `backend/server/internal/transport/ws/world_handler_test.go` 已把相关断言更新为新的权威落点，并执行 `go test ./server/internal/transport/ws`，当前通过

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
- 新增 PostgreSQL、Redis 相关配置项，并补充示例环境变量；后续已进一步收敛为单一 `PostgreSQL + Redis` 运行路径
- 新增 PostgreSQL 版账号、玩家、宠物仓储适配器，统一复用现有模块仓储接口
- 新增 Redis 版 `ws_token` 仓储适配器，使用 key 前缀和一次性消费语义预留真实接入点
- 新增 provider 装配层，统一管理服务端仓储依赖绑定；后续已删除 memory 运行分支
- 当时的 PostgreSQL/Redis 适配器先完成了骨架与接口约束，后续版本已补齐真实数据库连接、Redis 客户端初始化和驱动导入
- 新增 `config.env` 自动加载能力，后续只需要改 `backend/server/configs/config.env` 即可接入真实服务

## 2026-06-11 服务端 YAML 配置切换

本次补充聚焦把服务端启动配置从环境变量文件收敛到 YAML 配置文件：
- `backend/server/cmd/game-server/main.go` 不再先加载 `config.env`，改为解析 `backend/server/configs/config.yaml`；`PP_CONFIG_FILE` 仍可保留为“覆盖配置文件路径”的入口
- `backend/server/internal/config/config.go` 改为读取分段 YAML 结构：`http`、`auth`、`heartbeat`、`postgres`、`redis`，再转换成现有运行时 `Config` 结构，尽量不影响业务层依赖注入
- 示例文件已更新为 `backend/server/configs/config.yaml` 与 `backend/server/configs/config.yaml.example`，后续本地联调或部署时统一改 YAML，不再维护一长串 `PP_*` 配置键值
- 已新增配置加载测试，覆盖 YAML 解析、默认路径选择与基础校验，降低这次加载方式切换对启动链路的破坏风险

## 2026-06-11 PostgreSQL 宠物 mana 字段补齐

本次补充聚焦修复 PostgreSQL 模式下进入世界时报 `pp.mana does not exist` 的结构不一致问题：
- 新增迁移 `backend/server/migrations/010_add_player_pet_mana.sql`，为 `player_pet` 表补充 `mana` 字段，和当前 `pet_repo` / 战斗构建链路保持一致
- 同一迁移里已回填演示宠物 `20001/20002/20003` 的起始法力，保持 PostgreSQL 模式与内存测试仓储的默认值一致
- 这样 `PET_LIST_RESP`、编队读取、人物带宠进入战斗以及断线重连回放都可以继续复用同一个持久化宠物资源字段，不再因为数据库旧结构直接失败

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
- 早期曾补三张最小地图占位骨架用于联调地图切换链路；后续正式地图资源接入后，这些占位场景已被清理
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

## 2026-05-16 scene_1 地图资源替换

本次补充聚焦把客户端 `scene_id = 1` 对应的地图资源替换为新建的 `roxus_house` 场景：
- `world_controller.gd` 中 `SCENE_CONFIGS[1].scene_path` 已调整为 `res://scenes/maps/fashtown/roxus_house.tscn`
- 本次只替换客户端地图资源映射，不改服务端 `scene_id`、出生点配置和现有地图切换协议
- 当前 `roxus_house.tscn` 本身未接入门区 `Area2D`，因此如果需要保留 `1 -> 2` 的切图出口，还需要后续继续补门区节点与 `portal_id`

## 2026-05-16 roxus_house 门区补齐

本次补充聚焦把刚替换进来的 `roxus_house` 地图接回现有门区切图链路：
- `roxus_house.tscn` 已新增 `ExitPortal` 门区节点，并复用现有 `res://scripts/feature/world/map_portal.gd`
- 当前门区配置为 `portal_id = 1001`、`target_scene_id = 2`，与之前 `scene_1 -> scene_2` 的最小切图链路保持一致
- 同时新增了一个半透明 `ExitMarker` 出口标记，便于在只画了瓦片的阶段快速确认门区位置和触发范围
- 已完成 `roxus_house.tscn` 的 Godot 诊断检查，当前无新增场景错误

## 2026-05-16 roxus_house 固定镜头模式

本次补充聚焦把 `roxus_house` 调整为“相机固定、整图展示、角色在图内移动”的视角模式：
- `world_controller.gd` 为 `scene_id = 1` 新增 `fixed_view` 与 `spawn_local_position` 配置，当前 `roxus_house` 会按固定镜头模式渲染
- 固定镜头模式下，相机会固定在当前视口中心；地图会按场景内可见内容计算包围盒，并自动居中到屏幕可视区域
- 当地图实际尺寸大于当前窗口可视区域时，地图与角色锚点会按同一缩放比例缩小，尽量完整展示当前地图内容
- 角色位置换算从“相对出生点居中”切换为“出生点对应地图内本地落点 + 服务器坐标偏移”，避免角色和地图相对位置错位
- 已完成 `world_controller.gd` 的 Godot 诊断检查，当前无新增脚本错误

## 2026-05-16 主场景上下分区布局

本次补充聚焦把登录后的主运行态调整为“上部跑游戏、下部常驻 HUD”：
- `client/scenes/bootstrap/main.tscn` 已新增 `GameplayArea`，世界地图与战斗场景现在只在上部固定区域内渲染，避免覆盖底部常驻 UI
- 同一主场景新增底部 `HudRoot` 与 `HudBackground`，会永久显示 `client/asset/场景原图/闪光镇/时光小屋.png`，作为运行态底图
- 现有连接状态、场景信息、玩家信息、挑战按钮与日志输出已统一挪到底部 HUD 区，登录成功后会持续保留，不再压在地图上方
- `main.gd` 会把上部游戏区域尺寸同步给 `world_controller.gd`，固定镜头地图改为按游戏显示区大小计算居中与缩放，而不是按整个窗口布局
- 已完成 `main.tscn`、`main.gd` 与 `world_controller.gd` 的 Godot 诊断检查，当前无新增场景或脚本错误

## 2026-05-16 原客户端主运行态分层参考补充

本次补充聚焦把原客户端里和“登录后常驻主界面”最相关的结构继续沉淀到参考文档：
- `backend/docs/kdjl-client-reference.md` 已新增“登录后主运行态的分层布局”小节，明确原客户端采用单主画布承载上部游戏内容、下部常驻功能区和全局弹层
- 文档同时补充了“战斗层与常驻 UI 的共存关系”，说明原客户端世界层切到战斗时会继续复用公共 UI 资源，而不是整棵界面重建
- 当前项目可以继续吸收这条结构原则：`main.tscn` 作为常驻运行态根容器，上部切换世界/战斗显示层，下部保留固定 HUD 和后续操作区
- 本次只更新参考文档与记录，不扩展新的玩法范围，也不改变现有协议和主链路

## 2026-05-16 主运行态 UI 结构文档

本次补充聚焦把当前项目登录后的主运行态 UI 结构进一步沉淀为单独设计文档：
- 新增 `backend/docs/main-runtime-ui-layout.md`，明确主运行态采用“上部游戏显示区 + 下部常驻 HUD 区”的固定分层
- 文档把 `GameplayArea`、`WorldMount`、`BattleMount`、`HudRoot` 等节点职责单独拆开，约束地图切换、战斗切换只影响上部显示层
- 文档同时明确当前 MVP 下底部 HUD 只应承接连接状态、世界交互、战斗摘要以及宠物/编队/背包入口挂点，不直接扩展商城、频道、任务等超范围功能
- 本次没有新增代码逻辑，只补充了后续 UI 实现所需的结构口径与演进顺序

## 2026-05-16 底部正式操作区骨架

本次补充聚焦把主运行态文档中的底部操作区真正落成第一版可运行骨架：
- 新增 `client/scripts/bootstrap/runtime_hud.gd`，把底部常驻 HUD 的状态刷新、按钮事件和日志输出从 `main.gd` 中独立出来
- `main.tscn` 的 `HudRoot` 现已接入 `RuntimeHud`，并补充 `ModeLabel`、`SummaryLabel`、`ChallengeButton`、`PetButton`、`LineupButton`、`BagButton` 等操作区节点
- `main.gd` 现改为通过 `RuntimeHud` 驱动头部状态文本与日志，并接收底部按钮事件后分别复用现有 `App.request_interact()`、`App.request_pet_list()`、`App.request_bag_list()` 链路
- 首次进入世界后会自动同步一次宠物与背包摘要，使底部按钮的宠物数、编队数、背包数能尽快显示当前状态
- 已完成 `runtime_hud.gd`、`main.gd` 与 `main.tscn` 的 Godot 诊断检查，当前无新增场景或脚本错误

## 2026-05-16 底部入口最小弹出面板

本次补充聚焦让底部 `宠物`、`编队`、`背包` 入口不再只是占位按钮：
- `runtime_hud.gd` 已新增统一 `DataPanel` 逻辑，点击 `宠物`、`编队`、`背包` 按钮会打开对应的最小摘要面板，并支持关闭
- 宠物面板当前展示 `pet_uid`、`pet_id`、等级、HP 与是否在编队中；编队面板展示当前编队顺序和 HP 摘要；背包面板展示物品 ID 与数量
- 面板内容会跟随 `GameState.pets_changed`、`GameState.bag_changed`、`GameState.battle_changed` 自动刷新；进入战斗时会自动收起，避免与战斗态 HUD 冲突
- 本次继续复用已有 `App.request_pet_list()` 与 `App.request_bag_list()` 链路，没有新增额外协议或控制器
- 已完成 `runtime_hud.gd`、`main.tscn` 与 `main.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 编队最小交互与卡片面板

本次补充聚焦把底部右侧数据面板从文本摘要升级成更正式、可操作的列表样式：
- `main.tscn` 的 `DataPanel` 已改为“标题栏 + 提示文案 + 滚动列表 + 底部操作栏”结构，为后续继续细化样式保留稳定骨架
- `runtime_hud.gd` 现已按面板类型动态生成卡片列表：宠物面板显示宠物实例卡片，背包面板显示物品摘要卡片，编队面板显示“当前编队 + 可加入宠物”两段结构
- 编队面板已补最小可操作闭环：支持加入宠物、移除宠物、上移、下移和重置当前待提交编队
- 点击“提交编队”后会通过 `RuntimeHud -> main.gd -> App.set_pet_lineup()` 复用既有请求链路，仍然遵循客户端只提交完整编队顺序、服务端最终校验的口径
- 已完成 `runtime_hud.gd`、`main.gd` 与 `main.tscn` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 主场景 4:1 上下布局

本次补充聚焦把登录后主场景调整成更接近原版参考图的上下分区：
- `main.tscn` 现已将上部 `GameplayArea` 调整为约 `384px` 高、下部 `HudRoot` 调整为约 `96px` 高，对应 `320x480` 小窗口下约 `4/5 : 1/5` 的布局比例
- 当前已取消 `时光小屋.png` 作为下部背景，改为上部天蓝色纯背景、下部淡红色纯背景，并保留轻微遮罩，保证操作区与游戏画布上下分离、互不遮挡
- 底部状态区、按钮区和数据面板已同步压缩到更适合 `1/5` 高度的尺寸，日志面板改为隐藏，避免继续占用操作区可视空间
- `main.gd` 与 `world_controller.gd` 继续沿用现有上部游戏区域尺寸同步链路，因此地图和战斗仍只在上部区域渲染
- 已完成 `main.tscn`、`main.gd` 与 `runtime_hud.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 上部游戏区独立子视口

本次补充聚焦修复上部游戏区顶部出现根视口黑色清屏区域的问题：
- `main.tscn` 的上部游戏区已改为 `GameplayArea -> GameplayViewportContainer -> GameplayViewport` 结构，世界层与战斗层挂点均迁入 `SubViewport`
- `GameplayBackground` 继续作为上部区域的底色，而世界地图与战斗界面改为在透明子视口中绘制，避免根视口默认清屏色继续漏到游戏区
- `main.gd` 的 `_sync_world_render_frame()` 现会同步更新 `GameplayViewport.size`，并继续把同一份尺寸传给 `world_controller.gd`
- 本次修复只涉及主场景渲染边界，不改动现有世界、战斗、宠物、编队和背包链路
- 已完成 `main.tscn`、`main.gd` 与 `world_controller.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 适配 1080x1920 分辨率

本次补充聚焦把此前基于 `320x480` 小窗口假设的主运行态 UI 和固定视角地图，整体迁移到 `1080x1920` 新设计分辨率：
- `main.tscn` 现已改为按锚点保持 `4:1` 的上下比例，上部游戏区会自动占据 `80%` 高度，下部操作区会自动占据 `20%` 高度，不再依赖旧的 `384px/96px` 写死尺寸
- `HudRoot` 内的状态区、操作区、数据面板、按钮和标题字号都已整体放大，使其在 `1080x1920` 下保持可读性和可点击性；`runtime_hud.gd` 动态生成的卡片字体、边距和按钮尺寸也已同步放大
- `world_controller.gd` 的固定视角布局现已允许在大屏上按可见区域自动放大地图，不再强行把缩放结果限制在 `1.0` 以下；同时移除了先前只针对小屏临时加上的固定偏移，改为通过 `view_offset/view_scale` 配置控制
- `world_scene.tscn` 的地图加载蒙层提示与 `battle_scene.tscn` 的战斗卡片尺寸、字体和按钮高度已同步扩展，避免在大分辨率下仍然维持旧小窗比例
- 已完成 `main.tscn`、`runtime_hud.gd`、`world_controller.gd`、`world_scene.tscn`、`battle_scene.tscn` 与 `main.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误
- 登录页 `login_scene.tscn` 也已同步适配：新增浅色纯背景和居中登录卡片，整体放大标题、输入框、登录按钮、状态信息和日志区，使登录前入口在 `1080x1920` 下不再显得过小

## 2026-06-10 服务端权威自动战斗与超时补行动基础

本次补充聚焦把“命令阶段超时后谁来补行动”从客户端兜底改成服务端权威：
- `battle` 模块现已为活动战斗补充 `command_deadline` 与 `autoBattleEnabled` 运行态，`BATTLE_START_PUSH` / `BATTLE_STATE_PUSH` 会同步下发 `command_deadline_ms` 与 `auto_battle_enabled`
- `BATTLE_ACTION_REQ` 新增 `action_type=5` 入口，客户端可以只提交“开启/关闭自动战斗”的意图，具体剩余动作选择仍由服务端决定
- WebSocket 心跳链路现会顺带调用战斗超时推进；当命令阶段超过权威截止时间，服务端会自动为尚未提交的己方单位补默认动作并继续回合结算
- `session` 模块现已补充断线回调，玩家连接关闭或心跳超时后，活动战斗会自动切入服务端托管；后台轮询会继续推进回合并落宠物 HP 等持久化结果，即使客户端已经离线
- 当前客户端战斗场景已改为只展示服务端倒计时与托管状态，不再本地代投默认动作，避免联机时前后端各自推进同一回合
- 已执行 `cd backend && GOCACHE=/private/tmp/pocket-pet-gocache go test ./server/...`，当前服务端测试通过；Godot 侧仅完成静态脚本核对，尚未在引擎内实际联调

## 2026-06-10 PVE 奖励闭环第一版

本次补充聚焦把当前 PVE 从“能打完”推进到“打完会结算成长”：
- `battle` 结果快照现已补充 `reward_gold`、`reward_player_exp` 和按宠物拆分的 `pet_rewards`，服务端会在胜利时按敌方等级和数量生成稳定奖励
- `player` 与 `pet` 仓储已补充战斗结算写回接口，当前会在战斗结束时持久化玩家金币/经验以及各参战宠物的 HP / EXP
- 新增 `battle_record` 仓储接入，当前会按 `battle_id + player_id` 写入唯一奖励记录，作为最小版重复发奖保护
- `BATTLE_RESULT_PUSH` 现已额外返回本场发放的金币、角色经验以及发奖后的玩家累计金币/经验；客户端收到后会同步刷新本地玩家快照并在主日志中提示奖励
- 进一步补入了 `drop_texts` 文本掉落展示：当前会按怪物生成确定性的掉落文案，并在战斗结算日志和战斗详情里展示，但还不会写入背包
- 已补充 WebSocket 联调测试，覆盖战斗结果奖励字段、文本掉落、宠物经验回写和断线托管后的持久化结果；当前仍未扩展真实物品掉落 / 背包落库，也还没有把发奖与 battle_record 写入收敛到数据库事务

## 2026-06-10 技能目标类型扩展

本次补充聚焦把文档中提到的“全体技能 / 指定数量多目标技能”补到当前最小 PVE 闭环里：
- `battle` 技能表现已支持 `enemy_all` 与 `enemy_multi` 两类敌方范围目标规则；其中 `enemy_multi` 会保留客户端主目标选择，再由服务端按 `TargetCount` 自动补齐剩余目标
- 示例技能中，`1002 火花冲击` 已切换为全体敌方技能，`1004 弧光连射` 已作为双目标技能接入技能目录
- 战斗执行链路已补充多目标解析：全体技能不再要求客户端提供 `target_id`，双目标技能则会优先命中主目标并顺序补足其他存活敌方单位
- 客户端战斗界面已按 `skills[].target_type` 展示 `[敌全]` / `[敌二]` 徽标，并对全体技能禁用无意义的切换目标按钮，继续保持移动端简洁交互
- 已补充 `service_test.go` 覆盖全体技能与双目标技能命中多个敌人的服务端结算行为

## 2026-06-10 断线重连恢复第一版

本次补充聚焦把“战斗托管”继续往前推进到“客户端回来后能接上当前战斗态”：
- `session` 模块不再在 socket 断开时立刻销毁会话，而是保留一段短时重连窗口；窗口内可用 `reconnect_token` 把新连接重新绑定到原玩家会话
- `RECONNECT_REQ/RESP (1021/1022)` 已正式接入，服务端会在重连成功后返回新的会话信息，并轮换新的 `reconnect_token`
- `world` 与 `battle` 处理链路已补充重连快照拼装：当前会返回世界全量快照，以及活动战斗的 `battle_start + battle_state` 双快照，方便客户端直接复用已有控制器恢复界面
- 如果断线期间战斗已由服务端托管结束，重连响应还会临时带回一份 `battle_result`，让客户端仍能看到奖励与掉落文本，而不是只看到世界状态突然刷新
- 进一步补入了最近战斗状态缓存与 `last_frame` 协议字段：客户端重连时若仍知道自己停在第几帧，服务端会在缓存窗口内返回 `battle_replay_states`，先补最近几帧，再与当前战斗态对齐
- 客户端 `App` 已补入最小自动重连流程：连接关闭后若本地仍持有 `reconnect_token`，会优先发起 `RECONNECT_REQ`，成功后重建世界快照、恢复战斗界面，并补拉宠物/背包/任务摘要
- 已补充 WebSocket 测试覆盖“断开后重连恢复世界与战斗快照”的服务端闭环；当前版本仍属于全量重同步，不做战斗事件增量补帧与逐帧回放

## 2026-06-10 战斗属性与抗性底座扩展

本次补充聚焦先把“人物单人 PVE”需要的服务端属性底座补齐，但暂时不接等级成长：
- `battle` 运行态的 `actorRuntime` 已新增完整核心属性：生命、精力、攻击、防御、速度、法力、命中、闪避、致命率、爆伤倍率，并补入物理/技能抗性、混乱/昏睡/麻痹/封印/诅咒抗性、抗致命、抗爆伤、抗人物、抗宠物、抗佣兵与通用护盾抗性
- `formula.go` 的有效属性与减伤链路现已识别新的攻击来源与抗性维度：技能/物理抗性、通用护盾抗性、人物/宠物/佣兵来源抗性都会进入最终减伤计算
- 战斗执行链路已补入精力消耗判定：主动技能若精力不足，会在服务端权威降级为普通攻击，避免未来人物单人战斗再单独补一套资源校验
- 状态命中链路现已支持按目标的混乱、昏睡、麻痹、封印、诅咒抗性扣减命中率；暴击结算也会额外吃目标的抗致命与抗爆伤
- 已新增 `buildPlayerCharacterActor()` 作为未来“人物单人 PVE”入口的服务端人物战斗模板；当前尚未接入 `StartPVE()` 正式参战，但人物和敌人的服务端属性模型已经可复用
- 已执行 `cd backend && go test ./server/internal/module/battle/...`，当前 battle 模块测试通过

## 2026-06-10 人物战斗属性入库第一版

本次补充聚焦把上一节的人物战斗属性从“只存在 battle 运行态”推进到“玩家仓储可持久化读取”：
- 新增迁移 `backend/server/migrations/008_add_player_combat_stats.sql`，为 `player` 表补充人物战斗核心字段与抗性字段，包括精力、攻击、防御、速度、法力、命中、闪避、致命、爆伤，以及物理/技能/状态/来源抗性和通用护盾抗性
- `player.Profile` 与 PostgreSQL `player_repo` 现已同步扩展；`FindByPlayerID()` 会把这些持久化字段全部读回，后续人物参战时不再只能依赖 battle 模块里的硬编码模板
- `teststub` 的内存玩家仓储也已补齐同一套人物战斗字段，保证 battle / ws / auth 等本地测试仓储口径一致
- `buildPlayerCharacterActor()` 现已优先使用 `player.Profile` 中的真实持久化人物数值；若旧测试桩或未迁移数据里某些字段仍为 `0`，才会回落到最小默认模板，降低迁移阶段的联调风险
- 敌方构建逻辑保留了现有按玩家等级生成强度的口径，只是在同一条链路中补上了新的抗性和资源字段，避免这次入库改动意外改变现有 PVE 测试节奏
- 已执行 `cd backend && go test ./server/...`，当前服务端测试通过

## 2026-06-10 单人 PVE 人物 actor 正式接入

本次补充把前两节铺好的“人物战斗属性 + 持久化数据”真正接进单人 PVE 开战链路：
- `battle.Service.StartPVE()` 现在会先构建人物 actor，再按原顺序追加宠物编队；即使当前没有上阵宠物，单人 PVE 也可以由人物独立开战
- 新增人物持久化技能字段迁移 `backend/server/migrations/009_add_player_skill_ids.sql`，并把 `player.Profile` / PostgreSQL 仓储 / teststub 一起扩展为读取 `skill_ids`，避免人物技能继续硬编码在 battle 运行态
- 人物 actor 默认使用数据库中的技能顺序；当前人物起手技能为人物专属主动技 `1101: 裂空斩`，服务端自动托管与超时补行动作也会优先选择人物主动技能，精力不足时再权威回退到普通攻击
- `EnterWorldResp` 新增完整 `player` 快照，客户端 `player_snapshot` 现在可以直接读取人物的生命、精力、攻击、防御、速度、法力、命中、闪避、暴击与技能列表；战斗 actor 快照也新增 `unit_class`，客户端可以区分人物 / 宠物 / 怪物
- 战斗结算已改为只回写真实宠物的 HP / 经验，避免把人物 actor 误当成 `pet_uid=0` 的宠物持久化

## 2026-06-15 玩家成长与属性加点第一版

本次补充聚焦把玩家角色从“有经验字段”推进到“可升级、可加点、可后台配表”的完整闭环（宠物升级留二期）：
- 新增迁移 `035_player_level_progression.sql`：引入 `player_level_config`、`player_attr_convert_config`、`player_attr_allocate_log`，并为 `player` 扩展 `free_attr_points`、四维已分配点数与 `base_*` 裸装战斗值
- 新增迁移 `036_admin_player_progression_permissions.sql`：补充 `player_progression:view/edit` 后台权限
- 新增 `module/progression` 领域模块：服务端权威处理经验连升、溢出结转、升级发点、加点校验与战斗属性重算；配置表变更后自动刷新运行时缓存
- `player.AddExp` 与 `reward` 发经验链路已统一委托 `progression.ApplyExp`；升级只增加 `free_attr_points`，不直接改 `atk/def/...`
- 新增 WebSocket `PLAYER_ALLOCATE_ATTR_REQ/RESP (2061/2062)`；`PlayerSnapshot` 扩展 `exp_to_next`、`free_attr_points` 与四维属性；`BATTLE_RESULT_PUSH` 扩展 `level_up_count`、`attr_points_gained`、`free_attr_points`、`exp_to_next`
- 后台新增 `/api/admin/player-progression/...` 等级经验与转化率配置接口，前端页面 `/player-progression`；玩家详情页展示自由属性点与四维分配值
- 客户端状态面板与加点页已对接真实快照；`points_status_panel.gd` 加点请求走通用 loading 遮罩，响应后通过 `GameState.merge_player_snapshot()` 刷新 UI
- 设计说明见 `backend/docs/player-progression.md`；需本地执行迁移 `035`、`036`、`037` 后重启服务
