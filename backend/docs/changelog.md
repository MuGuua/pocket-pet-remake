# 最新变更记录

## 2026-08-15 P1-04 普通移动热路径移除 PostgreSQL 同步访问

- 完成多人同屏权威移动优化清单 P1-04。`backend/server/internal/transport/ws/world_handler.go` 在读取玩家永久档案前先识别普通移动与切图请求；Redis 移动状态已启用时，普通移动直接进入 `world.Service.MovePlayer`，由领域服务从 Redis 加载当前场景、位置和序号。
- Redis 普通移动成功后不再调用 `player.Service.UpdatePosition`。权威位置仍通过 Lua CAS 更新 Redis 并进入 dirty 玩家集合，等待 P1-05～P1-09 的集合消费、批量写回、位置版本保护和关键节点最终写回。
- 切图流程继续读取 PostgreSQL 玩家等级与永久档案，并在成功切图时立即持久化目标场景和出生点；未装配 Redis 移动仓储的旧服务兼容分支仍读取档案并同步写入普通移动位置，没有改变旧部署行为。
- `backend/server/internal/transport/ws/world_handler_test.go` 增加玩家仓储热路径访问计数桩，确认进入世界初始化完成后，单个 Redis 普通移动的档案查询和位置写入次数均为零，同时 Redis 权威位置、响应和同场景广播继续推进，PostgreSQL 测试快照保持不变。
- 同步更新 `backend/docs/multiplayer-movement-optimization-plan.md` 与 `backend/docs/architecture.md`；本批没有修改协议字段、客户端、数据库结构、迁移、后台页面或依赖。
- 验证：`GOCACHE=/tmp/pocket-pet-p104-go-cache go test ./server/internal/transport/ws ./server/internal/module/world -count=1`、`go test ./server/... -count=1` 和 `go vet ./server/...` 均通过。

## 2026-08-15 P0-10 权威移动竞态测试矩阵

- 完成多人同屏权威移动优化清单 P0-10；复核确认 `world` 领域层既有测试已经覆盖正常移动、服务端时间窗瞬移裁剪、场景矩形越界裁剪、静态导航穿墙裁剪、重复/倒退序号以及会话和场景权威不匹配。
- `backend/server/internal/transport/ws/world_handler_test.go` 新增成功移动后倒退序号专项测试：倒退请求只返回 Redis 当前权威位置，不再次执行 CAS，不回滚 PostgreSQL 兼容位置，也不向同场景旁观者广播旧坐标。
- 同文件新增切图竞态专项测试：玩家从场景 1 权威切换到场景 2 后，延迟到达的场景 1 普通移动包会收到 `scene mismatch` 拒绝和场景 2 的 `WORLD_RESYNC_PUSH`；Redis 新场景状态、PostgreSQL 新场景位置以及新旧场景旁观者均不受旧包污染。
- 本批只新增测试并更新计划、变更记录和任务总结；没有修改运行代码、协议字段、数据库迁移、后台页面、Godot 客户端或依赖。
- 验证：新增测试定向运行通过；`GOCACHE=/tmp/pocket-pet-p010-go-cache go test ./server/internal/module/world ./server/internal/transport/ws -count=1`、`go test ./server/... -count=1` 与 `go vet ./server/...` 均通过；`git diff --check` 通过。

## 2026-08-15 P0-09 远端移动旧包过滤

- `client/autoload/game_state.gd` 按远端实体保存最近接受的 `scene_id`、`scene_version` 和 `move_seq`；只接受当前场景、非旧场景代次且同代次序号严格递增的移动推送，新场景代次允许序号重新开始。
- 全量世界快照和运行态重置清空全部远端移动基线，实体离场清理单个基线；已存在实体的 `ENTITY_ENTER_PUSH` 摘要刷新保留基线，离场后的延迟移动包不会重新创建幽灵实体。旧服务广播的零场景代次按当前快照代次兼容，但仍要求正数且严格递增的 `move_seq`。
- `client/scripts/feature/world/world_controller.gd` 只在 `GameState.apply_entity_move()` 接受推送后刷新远端节点目标位置，重复、倒退或跨场景包不再触发表现同步。
- `backend/server/internal/transport/ws/world_handler.go` 删除 `ENTITY_MOVE_PUSH.scene_version=1` 的旧固定值：普通移动使用 Redis 权威移动结果的场景代次，同地图快速传送使用传送决策代次；兼容分支保留零值。对应 WebSocket 测试断言广播场景代次与权威状态一致。
- 本批无数据库迁移、后台页面、命令号、新依赖或正式玩法数据变化，未启动 P0-10 或 P1-04。
- 验证：Godot 4.7 Headless 最小项目专项矩阵覆盖正常包、重复/倒退序号、旧代次、跨场景、新代次序号重启、摘要刷新、离场延迟包、重新进入、全量快照和零值兼容；全项目编辑器解析通过。`go test ./server/internal/module/world ./server/internal/transport/ws`、`go test ./server/...`、`go vet ./server/...`、目标 GDScript Tab 检查和 `git diff --check` 均通过。

## 2026-08-15 P0-08 本机权威移动分级纠偏

- `MOVE_INTENT_RESP` 新增向后兼容的 `correction_ignore_distance` 与 `correction_snap_distance`；服务端从现有数据库移动配置快照派生阈值，小误差复用轴容差，大误差使用权威速度乘最大计算时间窗，不新增客户端正式数值常量。
- `client/scripts/feature/world/world_controller.gd` 开始在普通移动 `move_seq` 匹配后消费 `corrected_precise_pos`：小误差保留本地预测，中误差在不取消实时输入和自动寻路的前提下按响应 `speed` 逐帧消化固定偏移，大误差或拒绝响应立即应用权威位置。
- 场景不一致响应不在当前地图换算坐标，继续等待 `WORLD_RESYNC_PUSH`；切图、重同步和权威快照应用时会清理未完成的旧场景纠偏。旧服务未提供有效精确坐标时回退整数权威格，未提供有效阈值时保留预测，不使用客户端硬编码兜底。
- `world.Service.MovementCorrectionPolicySnapshot` 统一输出只读纠偏策略，并对极端异常配置进行有序和 `uint32` 溢出保护；成功和拒绝响应均携带同一策略。
- 补充领域阈值派生、有序和溢出测试，以及 WebSocket 成功/旧会话拒绝响应契约测试。本批无数据库迁移、无后台字段、无新增依赖，未启动 P0-09、P0-10 或 P1-04。
- 验证：后端受影响包与 `GOCACHE=/tmp/pocket-pet-p008-go-cache go test ./server/...` 全量测试通过；`go vet ./server/...`、Godot 4.7 Headless 分级边界专项测试、全项目编辑器解析、目标 GDScript Tab 检查和 `git diff --check` 均通过。Godot 专项退出时仅有既存的 `ObjectDB instance was leaked at exit` 警告。

## 2026-08-15 P0-07 普通移动领域用例下沉

- 新增 `world.MovePlayerInput`、`world.MovePlayerResult` 和 `world.Service.MovePlayer`，普通同场景移动由领域层统一加载 Redis 权威状态，校验玩家、会话与场景，兼容归一化旧客户端朝向/起停字段，并执行既有速度、边界、静态通行和序号规则。
- `MovePlayer` 使用加载到的旧序号直接执行一次仓储 CAS；抽取 `validateMovementStateAdvance` 供普通移动和切图序号推进复用，避免 handler 复制状态推进规则。
- `WorldHandler.HandleMoveIntent` 不再直接调用 `EvaluateMovement` 和 `AdvanceMovementState` 处理普通移动；新增的同场景协议适配方法只负责 DTO 转换、领域错误映射、响应和广播。
- 按 P1-04 边界保留普通移动前的 PostgreSQL 档案查询及整数位置兼容写入，未改动切图流程、协议字段、客户端纠偏逻辑、数据库结构或迁移。
- 补充领域测试覆盖正常 CAS、状态读取失败、会话/场景不一致、重复序号、非法输入、旧客户端无坐标心跳和 CAS 冲突；WebSocket 测试覆盖权威响应、PostgreSQL 兼容写入、同场景广播及旧会话拒绝。
- 验证：`GOCACHE=/tmp/pocket-pet-p007-go-cache go test ./server/internal/module/world ./server/internal/transport/ws` 与 `go test ./server/...` 全量后端测试均通过；`go vet ./server/...`、`git diff --check` 和工作区状态检查均通过。

## 2026-08-14 P0-06 场景静态通行闭环

- 检查最近提交 `85cc41b7` 后确认 P0-06 的版本化导航表、后台发布/回滚和导出工具主体已存在，但正式 26 场景种子、服务端权威落点审计、专项闭环测试及文档同步尚未完成。
- 修复 `client/scenes/maps/闪光平原/准备区.tscn` 主导航层命名，避免导出器误选单格动画占位层；严格导出 1~26 号正式场景成功，准备区导出为 `11×13`、74 个可通行格。
- `client/scripts/feature/world/world_controller.gd` 恢复服务端唯一权威落点：登录、普通门和快速传送统一应用 `WORLD_RESYNC_PUSH.self_pos`，客户端不再用历史场景导出出生配置覆盖快照。
- `backend/server/internal/data/postgres/world_repo.go`、默认异常回退点和测试仓储同步修复原本落在阻挡格的默认出生点与普通门目标点；数据库快速传送 `scene_id=13/26` 分别修复为 `(6,8)`、`(6,6)`。
- 新增 `backend/server/migrations/121_repair_world_authoritative_spawn_positions.sql` 和 `122_world_scene_navigation_seed.sql`；后者在事务内写入 26 个 `version=1/status=1` 的正式导航位图并检查所有启用场景均有发布版本。迁移仅生成，未连接或修改数据库。
- 修复 `backend/server/internal/teststub/repos.go` 导航主键游标，避免初始化 1~26 号导航后创建草稿复用 ID；同步测试拓扑的快速传送权威中心。
- 新增导航领域专项测试，覆盖穿墙路径裁剪、阻挡起点、缺失导航失败关闭、快照深拷贝、发布和回滚即时替换运行时缓存；后台 HTTP 测试覆盖草稿创建、发布归档和历史回滚完整契约。
- 同步更新权威移动计划、架构、协议、世界坐标和地图加载文档；P0-06 标记完成，P0-07 保持未开始。
- 验证通过：后端受影响包与 `go test ./server/...` 全量测试、后台 `npm run build`、Godot Headless 全项目解析、26 场景严格重导出与正式种子逐字节比对、目标 GDScript Tab 检查和 `git diff --check`。后台仅有既存的大包体积提示，Godot 仅有退出时 `ObjectDB instance was leaked at exit` 警告。

## 2026-08-12 P0-05 场景权威矩形边界闭环

- 新增迁移 `backend/server/migrations/119_world_scene_boundaries.sql`，扩展 `world_scene_definition` 的千分之一场景格边界和审计字段，并根据当前正式 Godot 地图资源及服务端传送落点初始化 1~26 号场景。迁移脚本仅生成，未连接或修改数据库。
- `backend/server/internal/module/world/` 新增场景边界模型、仓储接口、启动缓存、后台更新校验和移动结果矩形裁剪；边界缺失或非法时拒绝启动/移动，不回退代码常量。
- `backend/server/internal/data/postgres/world_repo.go` 增加启用场景边界列表与单场景更新仓储，`backend/server/internal/app/bootstrap.go` 在服务启动阶段强制加载缓存。
- 后台新增 `GET /api/admin/world/scene-boundaries` 与 `PUT /api/admin/world/scene-boundaries/{scene_id}`，复用 `world_movement:view/edit` 权限、操作原因审计和统一响应格式；`admin/src/pages/world/WorldMovementConfigPage.tsx` 增加边界列表、单位换算、固定尺寸编辑弹窗和二次确认。
- 补充领域层边界裁剪、缓存刷新、缺失边界以及后台列表/更新/非法输入/不存在场景测试。已通过 `GOCACHE=/tmp/pocket-pet-go-build go test ./server/...` 全量后端测试与后台 `npm run build`；P0-06 墙体和精细静态通行判定仍未实现。

## 2026-08-11 P0-04 后台配置闭环

- 完成多人同屏优化 `P0-04`：新增世界移动配置后台 GET/PUT、查看与编辑权限、操作原因审计和保存后二次确认；数据库写入成功后立即原子刷新服务端运行时配置快照。
- 管理后台新增“世界移动配置”页面，明确展示数据库配置值、单位换算、最终生效状态、更新时间和最近修改来源；不引入新依赖。
- 扩展迁移 `117_world_movement_config.sql` 保存最近更新原因和管理员 ID，新增 `118_admin_world_movement_permissions.sql`；迁移仅生成，未代替用户执行。

## 2026-08-11 多人同屏权威移动优化方案与协议预留

- 新增 `backend/docs/multiplayer-movement-optimization-plan.md`，明确服务端权威移动、Redis实时状态、PostgreSQL批量持久化、客户端纠偏、快照插值和 AOI 的分阶段开发清单。
- `MOVE_INTENT_REQ` 向后兼容新增四方向 `input` 和仅用于诊断的 `client_tick`；客户端同场景移动开始提交这两个字段，旧坐标字段和现有处理流程保持不变。
- `MOVE_INTENT_RESP` 预留 `corrected_precise_pos`、`server_tick`、`speed`，`ENTITY_MOVE_PUSH` 预留 `server_tick`；权威移动服务接入前这些响应字段保持零值，不提前改变现有移动判定。
- 同步更新移动协议文档；验证通过 Go 协议与 WebSocket 测试、Godot 4.7 Headless 全项目解析、GDScript Tab 检查和 `git diff --check`。
- `world` 模块新增权威 `MovementState` 与 `MovementStateRepository` 边界，覆盖会话、场景代次、定点坐标、移动序号和位置版本；`world.Service` 通过注入使用该边界，不直接引用 Redis实现。
- Redis客户端补充移动仓储需要的读取、Lua执行和删除能力；新增移动状态适配器，以 24 小时 TTL 保存状态，并用 Lua CAS 原子校验会话、场景代次和序号，成功后写入 dirty 玩家集合。
- provider 已创建 Redis移动状态仓储并注入世界领域服务；当前尚未切换移动 handler 或移除 PostgreSQL同步写入，避免在仓储验证完成前改变线上行为。
- `world.Service` 新增移动状态初始化与推进入口；推进时先在领域层拒绝旧会话、旧场景代次和非递增序号，再由 Redis Lua CAS 防止并发覆盖，并自动推进位置版本。
- 进入世界现在以服务端快照初始化 Redis移动状态；同会话重复进入保留现有序号，新登录会话替换旧状态，切图持久化成功后按服务端目标场景和出生点重建移动代次。
- 普通同场景移动已接入 Redis状态读取与 CAS推进，重复或倒退序号返回 `accepted=false` 及当前权威坐标；成功响应开始返回服务端确认的高精度坐标和时间基线。
- 重连会保留同一会话已有 Redis状态，缓存缺失时用 PostgreSQL世界快照恢复；本批仍保留 PostgreSQL同步位置写入，待批量持久化 worker 完成后再移除热路径。
- 重连世界快照现已优先使用 Redis最新场景、整数位置和千分之一格位置；协议新增可选 `self_precise_pos`，Godot 世界控制器在字段存在时先转换为场景坐标，再交给统一 `GameState` 快照入口。
- 普通门和地图快速传送接入与普通移动相同的 Redis序号推进，重复传送请求会在地图判定、落库和广播前返回当前权威状态，切图后的旧场景请求继续由权威场景 ID 拒绝。
- 新增迁移 `117_world_movement_config.sql`，配置权威速度、单包最大服务端时间跨度和非主轴容差；迁移只生成未执行，服务启动时从 PostgreSQL加载有效配置，缺失配置会明确阻止启动。
- `world.Service.EvaluateMovement` 使用服务端时间和数据库速度裁剪客户端候选高精度位置，拒绝斜向输入和超容差轴漂移，并把权威速度、高精度位置与服务端时间同步到响应和同场景广播。

## 2026-08-07 背包物品详情新节点树适配

- 适配用户重组后的 `client/scenes/ui/bag/bag_item_detail.tscn`：保留新增的 `PanelContainer` 内容包裹层、模糊背景实例和原版面板样式，不回退现有节点布局与视觉参数。
- `client/scripts/ui/bag/bag_item_detail.gd` 已通过 `%唯一节点名` 绑定名称、等级、强化、类型、部位、描述和操作按钮；父路径变化不需要新增并行取节点逻辑，详情刷新与“更多”菜单保持兼容。
- 恢复详情场景根节点 `visible = false`，避免运行时弹窗资源在编辑器或其他场景预览中默认常驻；正式打开仍由 `bag_panel.gd` 的详情层入口调用 `show()`。
- 本次仅涉及客户端场景适配与文档记录，无服务端、协议、数据库、依赖或正式玩法数据变化。
- 验证通过：详情场景运行态专项检查、背包面板集成启动、Godot 4.7 Headless 全项目编辑器解析和 `git diff --check`。

## 2026-08-07 主界面宠物 HUD 补充状态数值

- `client/scenes/ui/pet_status_hud.tscn` 在现有生命、法力、经验进度条内增加整数数值文本，并扩大状态条高度以适配移动端阅读；UI 节点全部预置在场景中，没有使用脚本动态创建。
- `client/scripts/ui/pet_status_hud.gd` 按服务端权威宠物快照同步刷新三项文本；经验总值使用 `exp + exp_to_next`，满级显示“经验 满级”。
- `PET_LIST_RESP` 的轻量宠物摘要补充 `exp`、`exp_to_next`、`mana`，继续复用主场景已有 `PET_LIST_REQ`，未新增请求、数据库字段、迁移、依赖或客户端正式玩法硬编码。
- 更新宠物列表协议与主运行态 UI 文档，并新增轻量协议转换测试，防止 HUD 所需字段再次丢失。
- 验证通过：宠物 HUD 场景实例化专项检查、Godot 4.7 Headless 主场景解析、`go test ./server/...` 和 `git diff --check`。

## 2026-08-07 NPC 菜单仅显示任务名

- `client/scripts/ui/common/option_list_panel.gd` 的 NPC 菜单按钮只显示数字序号和服务端返回的 `title`，不再展示 `subtitle` 简单描述或状态文案。
- 服务端返回的副标题和状态字段仍保留在原始选项数据中；`locked`、`completed` 的按钮禁用规则以及点击后的完整数据回传保持不变。
- 验证通过：目标脚本解析、Godot Headless 全项目解析，以及包含副标题和锁定状态的 4 条菜单运行态测试。

## 2026-08-07 NPC 菜单新节点树适配

- 适配用户重组后的 `option_list_panel.tscn`：标题区不再依赖已删除的 `HeaderIcon`，保留 `HeaderPortrait`、NPC 名称和右上角关闭按钮。
- 选项行改为复用独立的 `client/scenes/ui/common/option_row.tscn` 按钮场景；`option_list_panel.gd` 直接按 `ButtonsContainer` 节点顺序收集预置按钮，不再读取旧的 `HBoxContainer/Portrait/OptionButton` 子节点。
- NPC 菜单仍按服务端 `menu_entries` 顺序显示数字序号；附近 NPC、PVP 和挂机目标等头像列表改用 Button 自带 `icon` 展示形象，保持原始选项数据回传与禁用逻辑。
- `option_list_panel.tscn` 预置 30 个 `option_row` 实例并恢复根节点默认隐藏，脚本只控制内容和显隐，不动态创建 UI。
- 验证通过：Godot 4.7 脚本检查、目标场景加载、全项目编辑器解析，以及新节点树下的 3 条动态选项、服务端数据映射、上半身头像裁剪、头像列表模式专项测试。

## 2026-08-07 NPC 菜单列表参考图样式优化

- 优化 `client/scenes/ui/common/option_list_panel.tscn`：通用选项面板继续使用深色半透明圆角卡片、细描边、阴影、标题功能标识、移动端触摸选项卡和底部装饰。
- NPC 菜单标题在 NPC 名称旁显示当前地图实体的上半身形象；头像通过服务端权威 `entity_id` 查找当前场景中的 `InteractiveNPCBase`，复用其 `AnimatedSprite2D` 当前帧，并在面板中裁剪为上半身预览，不新增名称或 ID 图片映射。
- NPC 菜单选项严格按服务端 `menu_entries` 返回顺序动态显示，复用场景内预置选项行隐藏未使用节点，不在脚本中动态创建 UI，也不添加客户端假数据。
- 移除 NPC 菜单选项左侧的类型图标和类型前缀，改为 `1.、2.、3.` 数字序号；服务端返回的标题、副标题、状态、禁用逻辑和点击后的原始选项数据映射保持不变。
- 普通 NPC 列表、PVP 目标和挂机目标继续复用同一场景；这些头像列表模式不会显示 NPC 菜单标题头像。
- 本次只涉及客户端展示与文档记录，无后端、协议、数据库、依赖或正式资源新增。
- 验证通过：Godot 4.7 Headless 项目解析、目标脚本 `--check-only`、通用场景启动、NPC 菜单动态选项和标题上半身头像裁剪测试；`git diff --check` 通过。


## 2026-08-06 底部菜单按钮样式与抽屉动画
- 为主运行态底部“地图、挂机、任务、设置、背包”入口统一增加深色半透明圆角底板、蓝色描边、文字描边以及悬停/按下反馈，移动端触摸区域统一为 `70×52`。
- 右下角新增固定 `60×60` 十字按钮；点击收起时十字顺时针旋转 `90°`，业务按钮按从右到左的顺序滑入并淡出，再次点击时按从左到右的顺序滑出并恢复。
- 抽屉收起后业务按钮隐藏并忽略触摸，展开后恢复原有背包、任务、设置、地图和挂机信号，不改变任何面板打开或服务端请求链路。
- 挂机入口继续由当前地图的服务端权威暗雷配置决定可用性；不可用时其他按钮自动紧凑排列，不保留空槽。
- Godot 4.7 Headless 项目解析、两个脚本独立解析及抽屉收起/展开专项交互测试通过。

## 2026-08-06 右上角地图名称字号调整
- 将主场景 HUD 右上角 `SceneNameLabel` 字号由 `12` 调整为 `18`，提高移动端地图名称的辨识度。
- 保持标签原有右对齐、锚点和占位范围不变，不影响左上角状态 HUD 及其他按钮布局。
- Godot 4.7 Headless 项目解析通过。

## 2026-08-06 普通传送门落点改为纯场景导出变量控制
- 普通门 `MOVE_INTENT_REQ` 现在只提交 `target_scene_id + portal_id`，不再预加载目标地图，也不再携带 `target_pos`。
- 服务端继续校验当前场景、门拓扑、目标场景、发布状态和最低等级，但不接收、不采用普通门落点；旧客户端跨场景请求即使携带 `target_pos` 也会被忽略。
- 目标地图加载成功后，客户端每次从当前场景脚本调用 `get_portal_spawn_scene_position(portal_id)` 读取导出坐标；漏配专属入口时只回退同一场景脚本的统一出生点，不读取服务端坐标。
- 场景导出落点会初始化为本地移动同步基线，刚落地本身不会同步给服务端；玩家随后真正移动时仍沿用原有同场景移动和多人表现同步。
- 协议字段 `target_pos` 保留以兼容旧客户端和同场景移动；无数据库迁移、无新增依赖。
- 验证通过：服务端 `go test ./server/...`、Godot 4.7 Headless 项目解析、东路与传送区双向入口专项实例化检查以及全工作区 `git diff --check`。

## 2026-08-06 闪光镇东路与传送区双向普通门落点修复
- 将闪光镇东路 `portal_id=2003` 进入闪光镇传送区的目标场景格由 `(1,13)` 修正为 `(1,12)`，使人物落在左侧门内侧上方的无碰撞区域。
- 将闪光镇传送区 `portal_id=8001` 返回闪光镇东路的目标场景格由 `(9,5)` 修正为 `(8,5)`，避免人物碰撞范围与东路 `RightPortal` 重叠而落地后再次触发传送。
- 两个落点均在目标地图脚本中保留为导出变量，并在对应 `.tscn` 根节点显式保存；普通门落点现已与服务端内部兼容坐标解耦，因此服务端世界拓扑和测试仓储无需同步这两个 Inspector 值。
- 本次未修改普通门 ID、登录与世界地图快速旅行共用出生点，也不涉及数据库迁移。

## 2026-08-06 开放闪光镇传送区快速旅行
- 将闪光镇地图“通往闪光平原”标点指向 `scene_id=8` 并解除客户端禁用状态，第二次点击会沿用现有快速旅行流程发送请求。
- 新增迁移 `116_enable_shanguang_transfer_area_map_teleport.sql`，开放闪光镇传送区的服务端快速传送节点，服务端中心格为 `(5,10)`；迁移仅生成，未直接执行。
- 客户端目标地图加载后仍从闪光镇传送区脚本读取 `login_and_map_teleport_spawn_position`，该坐标不加入协议，也不从服务端获取。

## 2026-08-06 登录与世界地图快速传送统一出生点
- 为 26 张已注册地图统一提供根节点导出变量 `login_and_map_teleport_spawn_position`，可直接在各地图场景 Inspector 中独立调整坐标。
- 正式登录首次加载地图和世界地图快速传送加载目标地图后，客户端直接从当前地图实例脚本读取该变量并设置本地人物落点；每次使用时重新读取，不建立额外缓存。
- 统一出生点不加入 HTTP 登录响应、`ENTER_WORLD_REQ` 或 `MOVE_INTENT_REQ`，不提交服务端，也不从服务端获取；服务端快速传送的开放状态、目标地图、等级校验和数据库中心格逻辑保持不变。
- 普通传送门继续按 `portal_id` 使用专属入口，断线重连继续恢复服务端持久化位置；本次没有后端代码、协议或数据库迁移。

## 2026-08-06 战斗双方受击闪白与后退归位表现
- 在双方共用的 `battle_unit.tscn` 单位场景中加入独立 `BattleHitFlash` 表现组件；所有己方人物、宠物与敌方怪物都会复用同一套受击反馈。
- 组件只观察 `BattleUnit` 中由服务端权威战斗结果写入的生命值下降，不修改伤害计算、WebSocket 协议、血条动画、战斗动作顺序或数据持久化。
- 新增画布 Shader，使当前实际显示的普通图集精灵或 CHJ 精灵在受击时立即全白，并在 `0.2` 秒内线性恢复原色；每个单位使用独立材质，连续受击会重新从全白开始。
- 受击时己方单位向左、敌方单位向右后退 `12` 个世界像素，并在总计 `0.2` 秒内返回 `BattleUnit.base_position`；连续受击会终止旧位移并按权威站位重新计算，避免累计漂移。

## 2026-08-05 东路进入闪光镇传送区出生格修正
- 将从闪光镇东路 `portal_id=2003` 进入闪光镇传送区的目标场景格统一修正为 `(1,13)`，并在 `闪光镇传送区.tscn` 根节点显式保存该入口配置。
- 同步调整服务端正式世界拓扑、无客户端 `target_pos` 时的兼容入口、测试仓储及普通门坐标契约测试；门关系、协议、其他入口和快速传送逻辑保持不变。

## 2026-08-05 闪光平原快速传送节点修复
- 定位到线上数据库仅存在 `scene_id=26` 的海道快速传送节点，缺少闪光平原 `scene_id=9..25` 节点；服务端因此按权威配置返回 `map teleport unavailable`，客户端保持原场景。
- 新增迁移 `backend/server/migrations/115_repair_shining_plain_map_teleport_nodes.sql`，幂等补齐闪光平原 `scene_id=9..25` 及海道 `scene_id=26` 的中心出生格和开放状态。
- 本次不修改客户端传送协议和服务端传送逻辑；迁移文件仅生成，需由用户手动执行后重启/刷新服务端配置。

## 2026-08-03
- 将东路、闪光镇传送区和闪耀广场的普通门入口落点整理为地图根节点检查器中的“普通门切图落点（场景格）”导出配置，并在对应 `.tscn` 显式保存当前值，便于直接调整而无需修改脚本匹配逻辑。

- 调整普通门切图落点职责：客户端在发出 `MOVE_INTENT_REQ` 前实例化目标地图，并通过 `get_portal_spawn_scene_position(portal_id)` 选择入口 `target_pos`；服务端继续验证当前场景、门拓扑、目标场景和等级。
- 服务端验证通过后采用客户端入口坐标，并统一用于 `MOVE_INTENT_RESP.corrected_pos`、数据库位置、`WORLD_RESYNC_PUSH.self_pos` 和目标场景旁观者 `ENTITY_ENTER_PUSH`；未提交坐标时保留原服务端门点回退。
- 世界地图快速传送继续忽略客户端 `target_pos`，使用 `world_map_teleport_node` 的数据库中心格。
- 补齐闪光镇东路、闪光镇传送区和闪耀广场的双向来源门落点：`2003 -> scene 8 (2,12)`、`8001 -> scene 2 (9,5)`、`8002 -> scene 9 (20,12)`、`9001 -> scene 8 (6,9)`。
- 修复闪光镇东路隐藏“时光小屋”后仍保留 TileMap 物理阻挡：隐藏时同步关闭碰撞，重新显示时自动恢复碰撞，原有联网传送门逻辑保持不变。
- 删除客户端世界控制器周期输出的 `[PlayerPos][Client]` 玩家位置调试日志及其专用组装逻辑；玩家移动上报、服务端权威位置、远端玩家同步与持久化逻辑保持不变。
- 修复闪光平原区域地图缺少节点交互：新增 19 个地图热点、共享四帧选中光标、上下循环选择、人物当前位置图标，以及“首次选择、再次点击传送”的闪光镇同款交互。
- 闪光平原 17 张已注册地图使用 `scene_id=9..25` 发起服务端权威快速传送；“传送门”和“海道”保留可选择状态，但二次点击只提示尚未开放。
- 新增迁移 `113_shining_plain_map_teleport_nodes.sql` 写入闪光平原快速传送中心格，并同步 WebSocket 测试仓储与跨地区快速传送用例；迁移文件仅生成，未直接执行。Godot 4.7 Headless 实例化验证与 `go test ./server/...` 均通过。
- 修正世界地图“闪光平原”地区热点：从中央 Boss 图标移动到原图坐标 `(112, 144)` 的中间绿色节点，与地图原始标识及实际地区入口保持一致；闪光镇、精灵迷宫及服务端传送逻辑不变。

## 2026-08-02
- 为 `client/scenes/maps/闪光平原/` 下除“闪耀广场”外的 16 张新增地图绑定共享地图脚本 `shining_plain_level.gd`，统一提供 HUD 地图名称、移动端缩放、地图中心点、默认出生点和世界切图信号基础能力。
- 新增地图脚本不在客户端伪造尚未注册的 `scene_id` / `portal_id`；各地图现有传送区域保留，待服务端数据库场景定义、权威传送关系和客户端场景注册同步接入后再启用。

## 2026-07-29
- 修复多人同屏三类问题：切图后进入者在旁观视角“从别的坐标慢慢走过来”、远端人物移动逐像素卡顿、同场景玩家看不到对方权威人物形象。
- 修复远端玩家出生在地图左上角 (0,0) 的直接原因：`_sync_remote_players` 读取 `precise_pos` 时缺省值空字典也会通过 `is Dictionary` 检查，把没有该字段的 `ENTITY_ENTER_PUSH`/世界快照实体坐标算成 (0,0)；现在仅在字段真实存在且非空时使用高精度表现坐标。
- 出生坐标统一以服务端为唯一事实来源：客户端删除登录出生点与传送门落点本地覆盖，一律使用 `WORLD_RESYNC.self_pos`；服务端 `worldScenes` portals/entries 逐门对齐 Godot 地图当前调好的进门站位，注册默认时光小屋出生格改为 `(6,6)` 并新增迁移 `110_align_time_house_default_spawn.sql` 对齐存量未移动角色。
- 远端玩家插值改为保留连续浮点坐标：目标点与逐帧位置不再 `round()`，像素吸附只发生在项目级渲染层，消除 100ms 包间预测被逐帧取整重新拆成阶梯位移的卡顿。
- 远端玩家节点创建与 `ENTITY_ENTER_PUSH` 刷新时应用实体摘要中的服务端 `skin_id`（新增 `apply_remote_skin_id` 入口），不再回退到本机快照或默认精灵；跟随宠物继续按 `following_pet` 创建独立跟随节点。
- 新增端到端测试：切图后旁观者收到的实体进入推送坐标必须与 `MOVE_INTENT_RESP.corrected_pos` 一致且携带人物与宠物 `skin_id`；`world_repo_test.go` 逐门断言服务端出生坐标与客户端地图站位一致。
- 修复相机放大到 `3.0` 后地图与人物发糊：人物继续保留连续浮点移动，相机中心按当前 zoom 吸附到屏幕整数像素，并关闭 `Camera2D` 物理插值，避免半像素相机变换使整张世界画面错位采样。
- 修复特效移入人物场景后后半传送阵被地图吞掉的问题：外层光柱与后半环不再使用负 `z_index`，改为在同层先于人物精灵绘制；人物继续遮挡身后光效，前半环、核心光柱和粒子仍在人物前方。
- 将正式 `MapTeleportEffect` 从世界场景移入 `player.tscn`，位置与缩放完全由人物场景节点配置；世界控制器只负责播放和停止，不再在运行时覆盖位置，便于直接在 Godot 编辑器中调整对齐。
- 校准正式地图传送阵中心：特效相对人物根节点的偏移由碰撞体中心 `(13,-7)` 调整为人物 `Sprite2D` 图像中心 `(13,-20)`，传送阵、光柱和聚能光点整体上移 `13px` 世界坐标。
- 修正正式地图传送特效相对人物过大的问题：正式 `MapTeleportEffect` 实例缩小为三分之一，抵消世界相机的 3 倍缩放，使光柱、传送阵和粒子与 Demo 中人物的尺寸关系一致；Demo 与可复用特效资源保持不变。
- 将超时空传送特效接入世界地图二次点击传送：正式世界场景预置 `MapTeleportEffect`，点击后先锁定人物并播放聚能演出，达到人物消失点才启动原有黑屏换图。
- 地图传送继续在点击后立即发送 `MOVE_INTENT_REQ(map_teleport=true)`；提前到达的 `WORLD_RESYNC_PUSH` 会被视觉锁缓存，在黑屏中点应用服务端权威场景与中心坐标，普通传送门时序保持不变。
- 补齐拒绝、地图加载失败和转场超时恢复：统一停止特效、恢复由演出隐藏的人物并解除视觉锁；跨地图换图前会把预置特效移出旧地图角色层，避免节点随旧场景释放。
- 缩小传送收尾十字光点：`CrossFlash` 绘制区域由 `160×160px` 调整为 `120×120px`，四条光臂改用高斯衰减取代硬边截断，使长度更短、末端与交叉中心更圆润柔和。
- 补全传送特效后半段：播放进度达到 `0.70` 时通过 `vanish_started` 通知调用方隐藏人物，光柱、传送阵和粒子在 `0.70～0.82` 区间快速消散，随后人物中心出现蓝白十字光点并收束淡出。
- Demo 在每轮开始时恢复人物显示、收到消失事件后隐藏人物；整轮结束后人物和特效均保持消失，下一轮自动或手动重播时再恢复，验证完整“聚能传送 → 人物消失 → 十字光点消失”流程。
- 为底部传送阵增加人物遮挡关系：Shader 将圆环按上下半区平滑拆分，后半环在人物精灵后方绘制、前半环在人物前方绘制，人物形象会自然遮住身后的传送阵光纹。
- 缩小底部传送阵的初始最大尺寸：Shader 环半径由 `1.04 → 0.50` 调整为 `0.625 → 0.50`，使最大直径精确为最小直径的 `1.25` 倍；收束方向、线宽变化和播放时间轴不变。
- 根据 Demo 实际构图将传送光柱高度由 `560px` 先缩短到 `420px`，再按反馈继续降低 `64px`，最终高度为 `356px`；保留原有光柱宽度与透明度，并把中心聚能闪光下移到人物形象中心，地面能量环和播放时间轴保持不变。
- 新增独立可运行的超时空传送特效 Demo：人物周围依次出现蓝色柔光柱、高亮核心、收束地面环、聚能闪光和两层上升粒子，整体按“淡入聚能 → 浓度增强 → 停止发射并消散”的 2.8 秒时间轴播放。
- 光柱、地面环、爆闪和深蓝网格背景统一使用程序化 CanvasItem Shader，不新增位图素材；粒子使用移动端友好的固定 30 FPS 与有限数量发射。
- 可复用特效场景默认隐藏，通过 `play_effect()` / `stop_effect()` 控制；Demo 自动循环并支持点击、触摸、空格或确认键重播，正式流程继续复用既有服务端判定、协议与数据库地图中心配置。

## 2026-07-28
- 背包与地图面板右上角关闭按钮统一实例化 `panel_close_button.tscn`：保留原节点名、唯一节点引用和关闭信号，只移除各自旧图集/文字样式。
- 通用 `panel_close_button.tscn` 替换为 `BtnExitOpacity.png` / `BtnExitNoOpacity.png` 两态样式：普通与禁用为半透明，悬停与按下为不透明，并按素材原生尺寸统一为 `55×55`。
- 从地区地图返回世界地图时自动选中来源地区节点：恢复对应地区名称、键盘焦点和四帧选中动画；再次点击该节点可直接重新进入来源地区地图。
- 地图节点移除悬停视觉样式；世界地图地区节点改为“首次点击仅选中并播放动画、再次点击同一节点才进入地区地图”，点击其他地区只切换当前选中项。
- 地图节点选中态改用 `选中状态.png` 第一行的 4 帧精灵动画：共享 `AnimatedSprite2D` 跟随当前节点中心移动并以 8 FPS 循环播放，旧静态按下框和焦点框改为空样式。
- 根据确认后的世界地图地区标注修正热点映射：闪光镇移动到左侧绿色节点，精灵迷宫移动到右下绿色节点；中央 boss 继续对应闪光平原，左上皇冠不再作为闪光镇入口。
- 世界地图三个地区选中框统一按 `世界地图.png` 原始像素坐标、3 倍显示倍率和地图居中边距进行定位。
- 地图节点选中框统一改为 `52×52` 无圆角正方形；世界地图三个地区热点按原图标中心重新定位，闪光镇节点中心和服务端传送配置保持不变。
- 修正不同地图被强制拉伸到相同正方形尺寸导致节点图标大小不一致的问题：四张地图改为独立场景纹理节点，统一按原始像素 3 倍显示并保持各自宽高比。
- 地图面板中的地图绘制区域整体放大 2 倍：地图纹理由 `256×256` 调整为 `512×512`，外框与节点中心坐标同步等比放大；节点点击区和人物当前位置图标保持上一轮放大后的尺寸，避免相邻节点热区重叠。
- 地区地图节点、世界地图地区节点和人物当前位置图标统一以各自中心点为基准放大到原来的 2 倍，地图纹理与服务端传送目标保持不变。
- 地图面板删除“上一个 / 下一个”屏幕按钮，改为单个“世界地图”入口；点击后在原 `map_teleport_panel.tscn` 内切换到 `世界地图.png`，不创建新的顶层地图场景。
- 世界地图新增闪光镇、闪光平原和精灵迷宫三个地区热点，点击后在同一面板切换到对应地区地图；闪光镇继续显示已接入服务端权威传送的 1–6 号场景节点，其他尚未落地的地区不伪造传送目标。
- 删除旧的通用信息弹窗场景及脚本，服务端错误、人物/宠物升级和装备修复成功提示全部统一使用 `confirm_prompt_popup.tscn`；保留各业务原有的逐项等待、关闭输入阻断与提示内容。
- 放大移动端通用选项弹窗：`OptionListPanel` 统一调整为 `640×760`，头像由 `32×32` 放大为 `72×72`，同步放大标题、选项文字、行间距和关闭按钮；NPC List、挂机目标、PVP 目标和 NPC 菜单共用场景尺寸，调用脚本不再覆盖面板宽度。
- 修复可挂机地图点击“挂机”无反应：挂机目标改为服务端按暗雷 `spawn_monster_ids` 读取数据库怪物名称与形象，通过世界快照 `wild_encounter.targets` 下发；客户端不再从不包含暗雷怪物的 `nearby_entities` 收集目标。
- 修复人物移动时一卡一卡的问题：本地 `CharacterBody2D` 改为保留连续浮点坐标，不再把 `move_speed=90` 在 60Hz 下强制拆成 1px、2px 的交替位移；像素素材仍由项目级像素吸附和 nearest 过滤保持清晰，服务端移动协议不变。
- 闪光镇地图面板接入人物当前位置图标：复用场景中已有图标，根据服务端权威 `GameState.scene_snapshot.scene_id` 匹配地图按钮的 `target_scene_id`，并显示在对应标点左下角。
- 人物图标监听世界快照变化并在地图打开时重新校准；没有对应地图标点时自动隐藏，同时保持鼠标穿透，不影响选中和传送操作。
- 闪光镇地图面板适配新的 `680×1000` 尺寸：地图标点层随地图纹理居中，人物图标继续按标点定位；模糊背景与边框改为同层铺满，消除边框样式内容边距造成的四周间隙。

## 2026-07-27
- 地图 NPC 菜单预加载由逐 NPC 串行 `NPC_MENU_REQ` 改为单次 `NPC_MENU_BATCH_REQ`：服务端复用一次场景快照和任务摘要，并通过一条批量 SQL 读取全部 NPC 菜单配置。
- 客户端收到权威地图快照并挂载场景后立即结束转场；NPC 菜单、首次宠物/背包/任务请求、在线实体、任务更新和剧情推送均允许在进入地图后异步返回，不再锁输入或延长黑屏。
- 修复世界地图传送完成后角色无法移动：NPC 菜单预加载在地图转场期间不再重复叠加运行时输入锁，转场继续使用世界控制器自己的场景切换锁。
- 闪光镇传送地图绘制区由 `512×512` 缩小为 `256×256`，7 个地图标点由 `52×52` 同步缩小为 `26×26`，并按原比例调整全部标点坐标与外框尺寸。
- 客户端进入地图后会按服务端世界快照异步批量加载当前场景全部 NPC 动态菜单；玩家进入 NPC 碰撞区后只读取该地图缓存，不再临时发送菜单请求。
- NPC 菜单缓存严格绑定场景 ID，切图时清空并重建；批量请求不锁世界输入、不显示通用 loading，菜单可以在玩家进入地图并恢复移动后返回。
- 剧情确认后在同一地图新解锁 NPC 时会重新批量刷新当前地图菜单；旧地图迟到响应按场景 ID 丢弃，不会污染新地图缓存。
- 修复人物资料等待函数缺少显式末尾返回导致的 Godot 静态检查错误 `Not all code paths return a value`
- 修复人物状态面板打开失败：新增独立 `2065/2066 PLAYER_PROFILE` 协议，只读取当前人物权威属性，不再复用包含地图、宠物、任务等数据的完整 `ENTER_WORLD_REQ`
- 人物面板打开流程移除 `BAG_LIST_REQ` 和 `WALLET_QUERY_REQ`；背包物品只在玩家点击背包入口时查询，背包或钱包接口失败不再连带阻止人物面板打开
- 人物面板中的背包数量改为“请打开背包查看”，不展示未主动刷新过的旧背包缓存

## 2026-07-26
- 优化跨地图切换关键路径：位置落库后立即发送不含在线玩家资料的基础 `WORLD_RESYNC_PUSH`，客户端解除黑屏后再通过 `ENTITY_ENTER_PUSH` 增量补齐同屏玩家和跟随宠物，避免远程数据库资料查询阻塞转场
- 同屏玩家与跟随宠物改为各一次轻量批量查询，只读取名字、等级、经验、血量、精力、形象及位置等世界展示字段，不刷新完整战斗快照，也不读取背包
- `ENTER_SCENE` 任务推进新增不含背包的场景事实查询，并移动到基础世界快照之后；任务更新和一次性场景触发继续保持服务端权威，但不再延长黑屏时间
- 修复高频普通移动请求在慢数据库链路中串行堆积并阻塞后续切图：客户端现在同时只保留一个普通移动请求在途，收到对应 `MOVE_INTENT_RESP` 后再补发角色最新状态，使传送请求最多只排在一个移动请求之后
- 修复切图请求与高频坐标同步回包竞态：客户端记录本次切图 `move_seq`，延迟到达的旧 `MOVE_INTENT_RESP` 不再清空目标场景和传送门状态；远程数据库世界快照等待窗口由 5 秒调整为 15 秒
- 闪光镇地图节点 UI 接入服务端权威快速传送：再次点击当前选中标点发送兼容的 `MOVE_INTENT_REQ(map_teleport=true)`，迁移 `108_world_map_teleport_nodes.sql` 配置 1–6 号地图中心格；同地图传送只同步中心坐标，不重复触发进出场景事件
- 修复地图切换等待服务端快照超时后仍保留 pending 场景和玩家移动锁的问题；超时现在完整回滚传送门、朝向、出生点与视觉锁状态，玩家可继续移动并重新触发传送
- 普通业务提示从已关闭的历史日志入口拆出，改为复用 HUD 场景节点展示单条 3 秒移动端提示；登录页继续通过现有提示标签展示最新状态或错误
- Web 与 Android 导出预设排除纯编辑器目录 `addons/godot_ai/**`，保留编辑器内 MCP 能力的同时不再把相关脚本编译进游戏发布包
- 恢复闪光镇东路地图脚本的四空格缩进，避免引入 Tab 与后续局部修改混用

## 2026-07-24
- 恢复客户端标准输出并新增 `[SceneTransition][Client]` 定向切图日志，覆盖传送门请求、响应、世界重同步、地图挂载耗时、遮罩中点与超时；其他业务/HUD 日志继续关闭
- 服务端新增 `[SceneTransition][Server]` 权威传送日志，记录请求场景、玩家数据库场景、传送门判定、出生点、位置落库和 `WORLD_RESYNC_PUSH` 发送结果
- 关闭客户端标准输出、错误输出、登录页/HUD 日志和战斗/奖励调试打印；网络请求、错误弹窗与业务提示链路保持不变
- 优化同场景移动性能：远端移动推送不再触发全局世界快照刷新，本地坐标 HUD 与玩家快照只在整数场景格变化时更新，保留 100ms 高精度网络表现同步
- 修复切图异步流程离树后访问空 `SceneTree.process_frame`；地图等待增加 5 秒兜底，失败或超时均恢复黑屏遮罩与输入，不再永久卡在转场
- 地图首次进入提示在服务端下发前立即持久化一次性剧情标记，后续登录、切图或世界重同步不再重复展示；带动画、任务接取或剧情解锁副作用的提示仍等待客户端完成确认后推进
- 关闭高频 `MOVE_INTENT_RESP` 的通用服务端结果日志，避免同场景位置同步持续输出 `position synchronized`；消息仍正常路由并处理
- 将从时光小屋 `portal_id=7001` 返回闪光镇东路的出生坐标统一为 `(9, 5)`，同步更新客户端脚本默认值、地图场景覆盖值、服务端权威传送配置和测试桩

## 2026-07-23
- 修复远端玩家每次快速追上 100ms 网络目标后停顿造成的移动卡顿：观察端现在会在相邻权威表现包之间按人物朝向和速度连续预测，新包到达后平滑纠偏，并用 180ms 上限防止断包后持续移动
- 修复多人同场景的人物、宠物位置和朝向在不同客户端视角不一致：移动同步增加服务端校验的千分之一格表现坐标、明确四方向与起停状态，远端人物按高精度目标插值；宠物按 24px 路径步重新累计并在转向时重置路径轴，避免稀疏整数格采样造成二次偏差
- 优化客户端四方向点击寻路：保留 AStar 障碍可达性判断，在不穿越碰撞格的前提下优先重建长直线或单拐角路径，直线路段遇到障碍后才沿 AStar 绕行节点转弯；剧情动作路径同步复用该低转弯算法

## 2026-07-22
- 补齐同场景远端玩家宠物展示：玩家实体摘要新增数据库编队首只宠物 `following_pet`，进入世界和实体进入时同步下发，编队变化后实时广播刷新；客户端为每个远端玩家创建独立宠物跟随节点并沿权威移动路径更新
- 修复通用确认提示无法点击关闭且 5 键无效：`ConfirmPromptPopup` 现在启用专用输入监听，支持主键盘/小键盘 5、回车、关闭按钮与底部继续按钮，并为移动端按钮补充扩大热区兜底
- 进入地图提示、任务前置与目标引导提示、剧情动画结束提示和任务完成提示统一复用 `confirm_prompt_popup.tscn`；保留服务端 BBCode 文案与原有确认后推进时序，普通系统通知仍只写入 HUD 日志
- 新增地图最低进入等级配置：迁移 `109_world_scene_required_level.sql` 为 `world_scene_definition` 增加 `required_level`（1~100），现有地图统一初始化为 1 级；后台“地图NPC配置”页新增地图进入等级配置表与编辑弹窗
- 地图切换改为服务端读取目标地图数据库配置并使用玩家权威 `level` 校验；等级不足时不更新玩家场景/坐标，`MOVE_INTENT_RESP` 返回 `accepted=false` 与“前面的路以后再来探索吧”，客户端通过全局 notice 展示
- 修复 Godot 编辑器重新生成的 Web 临时调试页仍用 `!important` 保留旧 `1080x1440`（3:4）样式的问题：Web 运行时现在以同优先级覆盖 Canvas、加载层和独立包装节点的宽高、边界与纵横比，强制恢复项目设计尺寸 `780x1440`（13:24），同时继续保留全屏 `body` 负责水平和垂直居中
- 补齐同场景多人实时移动同步：客户端按整数场景格上报 `MOVE_INTENT_REQ.target_pos`，服务端持久化后仅向同地图玩家广播进入、移动与离开事件；客户端复用现有玩家场景创建无输入、无碰撞、无相机的远端角色并平滑追赶权威坐标

## 2026-07-21
- 后台任务模板将“前置任务”升级为结构化“任务开启条件”，服务端接入任务完成、人物等级/最终属性、地图、物品、宠物等级、剧情标记与时间段的权威校验，并保留旧字段兼容

## 2026-07-13
- 新增可复用世界剧情动作场景：支持剧情期间锁定玩家手动输入、按统一场景坐标通过导航网格行走、设置结束朝向、播放指定角色动画或暂停在指定 PNG 帧，并在结束后恢复世界状态和推进服务端剧情节点
- `CinematicPlayer` 现在向动作场景注入当前世界控制器，并通过单活动实例与播放代次避免旧演出误推进新节点；剧情动画 Key 自动解析 `scenes/cinematics/{key}.tscn`，新增剧情场景不再修改客户端注册表
- 移除 `CharacterVisual` 对自身所属场景的无引用预加载，解除角色场景与脚本的循环加载
- 固定客户端过场现在可以在动画 Key 脚本中使用 Tween 编排像素移动、方向动画和多句本地对白；本地继续操作与服务端剧情继续请求完全隔离，整段完成后才统一推进服务端 action
- 新增“初见桃子”固定过场初始化：桃子、七色羽和本地玩家按场景脚本指定位置与朝向展示；动画 Key 自动查找范围兼容 `res://剧情动画/` 目录
- 交互 NPC 基类改为安全读取可选 `Area2D`，固定过场复用 NPC 形象但不配置交互区域时不再产生节点缺失错误
- “初见桃子”过场启动时按世界控制器相同算法归一地图原点、重算相机边界，并同步当前世界相机的位置、中心偏移和缩放系数，保证地图位置与真实客户端构图一致
- “初见桃子”按客户端脚本补齐首段完整演出：六句固定对白、桃子往返与上移、桃子和七色羽并行右移及七色羽终点转身；本地对白支持鼠标左键、回车、主键盘 5 和小键盘 5 推进
- “初见桃子”场景内新增默认隐藏的备用 `NPCDialoguePanel`：正式客户端仍走主场景对话桥接，Godot 单独运行该场景时自动使用备用面板，支持完整预览对白和 Tween 顺序
- NPC 对话容器宽度调整为 `720px` 并水平居中，对应东路地图 `240px × 3.0` 的实际显示宽度；单场景预览和正式客户端调用均与地图左右边界对齐
- NPC 对话正文调整为 `32px`，按 `704px` 可用宽度约显示 22 个汉字；同步放大说话人文字与头像、状态文字和继续按钮，避免移动端出现文字过小或控件裁切
- “初见桃子”中桃子和七色羽并行向右移动完成后统一转为左向待机，补齐桃子的终点转身表现
- “初见桃子”在两名角色转身后播放三层冲击波：从七色羽终点左移 30px、上移 40px 的位置出现，同步向左移动 100px，播放完成后停止并隐藏
- 冲击波演出改为固定持续 2 秒，三个序列帧节点统一以 6 FPS 循环播放，并在同一时长内完成整体左移 100px 后消失
- 冲击波在桃子和七色羽完成转身后先等待 1 秒，再开始显现、循环播放和左移动画

## 2026-07-10
- 技能模板新增独立品质 `skill_quality`：普通、神技、魂技、圣技、绝世；后台支持编辑、筛选和详情展示，宠物技能详情协议同步下发品质。客户端宠物技能按钮分别使用灰、绿、蓝、紫、金边框，品质仅影响展示，不改变战斗逻辑
- 技能表现资源 `SkillVisualConfig` 新增可在 Godot Inspector 配置的 `icon` 字段；宠物技能详情协议同步下发数据库技能模板已有的 `skill_visual_id`，客户端宠物状态面板和战斗技能选择列表按该 ID 读取对应 `.tres.icon`，未注册或未配置时继续使用默认占位图或纯文字，不向服务端保存客户端资源路径
- 修复圣技幻影闪击配置图标后宠物技能栏仍显示默认图标：新增迁移 `096_backfill_signature_skill_visual_id.sql` 回填技能 `20191` 的表现资源 ID；运行时技能快照在历史 `skill_visual_id` 为空时使用数据库 `skill_code` 兼容，显式配置值仍保持最高优先级
- 技能表现注册改为使用 Godot `ResourceLoader.list_directory()` 自动扫描 `resources/battle/skill_visuals/`；新增 `.tres/.res` 并设置唯一 `skill_visual_id` 后即可自动生效，不再维护代码路径清单，同时兼容当前 `all_resources` Web/桌面导出策略
- 修复被动技能迅捷A、致命A已配置本地图标但宠物技能栏仍显示默认图标；两个资源现在由目录扫描自动注册，被动技能仍不会进入战斗可释放技能列表，也不会播放技能特效

## 2026-07-09
- 系统技能新增“技能命中加成”配置：迁移 `backend/server/migrations/094_skill_hit_bonus.sql` 为 `skill_definition` 增加 `skill_hit_bonus`；后端技能模板、运行时缓存与战斗结算已同步读取该字段，命中/闪避判定现在使用“施法者命中 + 技能命中加成”，不会再复用 `skill_crit_add` 造成暴伤字段污染；后台技能效果编辑器新增“命中加成”条目，默认可直接配置 `40` 点本次技能命中

## 2026-07-08
- 后台任务阶段编辑器改为摘要卡片 + 原生拖拽排序：`admin/src/pages/quests/QuestStageEditor.tsx` 不再使用表格平铺阶段，而是按卡片展示“阶段顺序 / 阶段ID / 事件类型 / NPC / 菜单 / 剧情 / 引导”等摘要信息，并支持直接拖拽卡片调整任务推进顺序；保存结构仍沿用原有 `stages` 列表，不改接口契约
- 后台任务模板创建改为服务端自动分配任务 ID：`backend/server/internal/module/quest/service.go`、`backend/server/internal/data/postgres/quest_admin_repo.go` 与 `backend/server/internal/teststub/repos.go` 现支持在创建任务模板时省略 `quest_id`，由服务端按当前最大 `quest_id + 1` 自动生成；`admin/src/pages/quests/QuestAdminPage.tsx` 同步移除新增表单中的手填任务 ID，改为“保存后自动生成”，减少人工编号冲突
- 后台新增/编辑任务模板表单结构优化：`admin/src/pages/quests/QuestAdminPage.tsx` 将原先单列超长表单改为「基础信息 / 任务阶段 / 任务奖励」三标签结构；基础资料与接取提交流程拆成独立卡片，阶段与奖励各自独立编辑区，并统一固定高度内部滚动，减少策划新增任务时长距离滚动和字段混杂带来的误填
- 后台任务模板的前置任务录入改为列表新增样式：`admin/src/pages/quests/QuestAdminPage.tsx` 不再要求手写 `pre_quest_ids` JSON，而是直接按行新增/删除前置任务 ID；保存时仍按原有 `number[]` 契约提交，减少 JSON 格式输错导致的配置问题
- 后台玩家详情页改为分标签结构：`admin/src/pages/players/PlayerListPage.tsx` 将原先基础信息、战斗属性、钱包、宠物、背包的超长纵向内容改为「角色概览 / 钱包 / 宠物 / 背包」分标签展示，并把每个标签内容收敛到固定高度的内部滚动区域，避免运营查看角色信息时需要整页长距离下滑
- 后台玩家详情/编辑页补齐子模块更新后的自动刷新：`admin/src/pages/players/PlayerWalletSection.tsx`、`admin/src/pages/players/PlayerPetSection.tsx` 与 `admin/src/pages/players/PlayerBagSection.tsx` 现支持通过 `onDataChanged` 回调通知父级；当运营在角色页内调整钱包、宠物或背包后，玩家列表、详情抽屉和编辑弹窗会自动重新拉取最新玩家快照，避免修改成功后仍停留在旧展示
- 后台玩家编辑弹窗同步改为分标签结构，并补充手动刷新按钮：`admin/src/pages/players/PlayerListPage.tsx` 的编辑弹窗现已拆为「基础编辑 / 钱包 / 宠物 / 背包」四段，内容区固定高度内部滚动；玩家详情抽屉与编辑弹窗标题区都新增“刷新”按钮，方便运营在跨页面修改后主动重拉服务端最新数据
- 后台玩家详情抽屉与编辑弹窗的工作台样式进一步统一：`admin/src/pages/players/PlayerListPage.tsx` 现已统一标签栏尺寸、内部滚动容器样式与刷新按钮尺寸；编辑弹窗的首个标签更名为“角色编辑”，与详情页“角色概览”形成对应，减少运营在详情/编辑两种视图间切换时的认知跳变

## 2026-07-07
- 修复后台删除宠物被动技能后生命/攻击等展示属性未回退：`backend/server/internal/module/pet/service.go` 现在会在保存玩家宠物前，把“后台详情页因被动技能临时展示出来的加成值”还原回底层基础值；当运营只删除技能、不手改生命/攻击字段时，保存后 `hp/hp_max/atk/spd/mana` 及相关暴击/抗性会按技能删除结果正确回退，不再把加成后的展示值误写回 `player_pet`
- 后台技能编辑页进一步收紧被动技能效果项：`admin/src/components/SkillDefinitionEditor.tsx`、`admin/src/components/SkillEffectConfigEditor.tsx` 与 `admin/src/pages/skills/SkillDefinitionPage.tsx` 现已在被动技能模式下只保留“被动属性加成”效果类型，并在切换为被动或重新打开旧数据时自动过滤攻击系数、战斗表现等主动技能条目，避免运营再次误配无效字段
- 修复被动技能保存后又回填默认攻击系数/战斗表现：`backend/server/internal/module/skill/model.go` 现在只对主动技能补默认 `attack_pct=100`、`animation_key=slash` 和默认颜色；被动技能若在后台删除这些配置，保存后将保持为空，避免刷新再次看到旧的攻击系数和战斗表现条目
- 新增 `backend/server/migrations/088_backfill_player_pet_skill_json_numbers.sql`：把 `player_pet.skill_ids`、`innate_skill_ids`、`normal_skill_ids` 中可安全转换的字符串数字数组批量回填为 JSON 数字数组，便于逐步清掉后台玩家宠物详情 500 对应的历史脏数据
- 后台玩家宠物详情/列表兼容历史技能 JSON 脏数据：`backend/server/internal/data/postgres/pet_repo.go` 读取 `player_pet.skill_ids`、`innate_skill_ids`、`normal_skill_ids` 时已改为复用弹性 `uint32[]` 解析，兼容 `[20001]` 与 `["20001"]` 两种格式，修复 `GET /api/admin/pets/:pet_uid` 因旧字符串数组返回 `500 load admin pet detail failed`
- 玩家详情页内的宠物编辑入口现已与独立宠物页统一为正式技能槽编辑：`admin/src/pages/players/PlayerPetSection.tsx` 改为 `天生技 / 普通技 / 兼容战斗技能预览` 三段式结构，避免不同入口一个编辑结构化技能槽、另一个仍编辑旧 `skill_ids` 文本导致行为不一致
- 后台玩家宠物编辑页改为正式技能槽口径：`backend/server/internal/module/pet/model.go`、`backend/server/internal/data/postgres/pet_repo.go` 与 `admin/src/pages/pets/PlayerPetListPage.tsx` 已接入 `innate_skill_ids / normal_skill_ids`，后台保存玩家宠物技能时会优先写入正式结构化技能槽，再自动回写兼容 `skill_ids`
- 玩家宠物运行时技能集合改为“结构化技能槽 + 兼容 skill_ids 去重合并”：`backend/server/internal/module/pet/skill_slots.go` 新增 `MergeBattleSkillIDs()`，保证后台临时补到 `skill_ids` 的被动技能也能真实进入属性面板与战斗口径，同时不覆盖已存在的正式技能槽数据
- 后台玩家宠物列表/详情现按与游戏内一致的永久被动口径展示基础属性：运营在后台给宠物挂上加生命/加速度等被动后，无需手算即可直接看到折算后的生命、攻击、速度和法力
- 系统技能模板新增显式永久属性加成配置：迁移 `backend/server/migrations/087_skill_passive_attribute_bonus.sql` 为 `skill_definition` 增加 `passive_attr_key / passive_attr_mode / passive_attr_value` 三个字段，后台不再只能依赖技能名前缀推断“加什么属性”
- 后台技能效果编辑器新增“被动属性加成”效果类型：`admin/src/types/skillEffectConfig.ts` 与 `admin/src/components/SkillEffectConfigEditor.tsx` 现支持为被动技能直接选择属性字段、加成方式与数值，并自动与技能编辑接口扁平字段互转
- 宠物永久被动属性优先读取显式配置：`backend/server/internal/module/pet/passive_attributes.go` 会先按 `passive_attr_*` 字段把生命/攻击/速度/法力、暴击与抗性加成折算进 `Pet` / `LineupPet` 最终快照；若旧技能尚未补配置，则继续回退到 `强壮/强力/迅捷/...` 旧前缀规则兼容历史数据
- 服务端新增最小校验：显式永久属性加成只能配置在 `activation_mode=passive` 的被动技能上，且只允许当前受支持的属性字段与加成方式，避免后台把永久属性被动错误配置到主动技能
- 系统技能库显式增加 `activation_mode`：新增迁移 `backend/server/migrations/086_skill_activation_mode.sql`，支持区分 `active` 主动技能与 `passive` 被动技能，并把旧的 `support + 0 消耗` 技能回填为被动
- `skill_type` 继续保留攻击/治疗/辅助等效果分类语义；后台技能模板列表、详情、编辑表单与筛选条件已同步接入 `activation_mode`
- 选择“被动技能”后，后台会把目标与消耗字段自动收口为 `self / 0 消耗`；服务端也新增最小校验，禁止把被动技能配置成普攻或主动目标技能
- 战斗运行时不再把被动技能下发到客户端可选技能列表，也不会在自动托管、默认选技或主动施法入口中释放被动；如果客户端误传被动技能，服务端会权威回退为普通攻击
- 现有 `passive_skills.go` 的被动效果链路保持不变，被动技能仍可继续提供吸血、反伤、属性加成等常驻效果
- 宠物属性面板现已读取“被动加成后的最终属性”：`backend/server/internal/module/pet/passive_attributes.go` 会把 `强壮/强力/迅捷/魔心/致命/暴伤/厚甲/坚韧/结界` 这类永久属性被动折算进 `Pet` / `LineupPet` 快照
- 宠物进入战斗时不再重复叠加这些永久属性型被动，只保留吸血、连击、复活、反伤等战斗期效果，修复“面板已经加过一次、战斗里又加一次”的双算问题
- 修复战斗奖励掉落概率判定时机：怪物战斗奖励缓存现在只保存后台配置，不再在 `RefreshBattleRewardCache` / 后台保存时提前 roll `drop_rate`；每场 PVE 结算调用 `BuildPVERewardBundle` 时按最新缓存配置独立判定，避免 50% 掉落在缓存刷新时未命中后连续多场都不掉落
- 后台宠物模板详情兼容历史技能 JSON 脏数据：`backend/server/internal/data/postgres/pet_definition_admin_repo.go` 读取 `skill_ids`、`innate_skill_ids`、`normal_skill_ids` 时不再只接受 `[]uint32`，新增 `json_uint32_array.go` 兼容 `[101,102]` 与 `["101","102"]` 两种格式，避免旧库里字符串数组导致详情接口 500
- 新增宠物模板技能 JSON 清洗迁移：`backend/server/migrations/085_backfill_pet_definition_skill_json_numbers.sql` 会把 `pet_definition` 中 `skill_ids`、`innate_skill_ids`、`normal_skill_ids` 里可安全转换的字符串数字数组回填为真正的 JSON 数字数组，便于逐步清掉历史脏数据
- 后台宠物模板详情接口补充底层错误日志：`backend/server/internal/transport/http/admin_pet_definition_handlers.go` 在 `load admin pet definition detail failed` 返回 500 前会把 `pet_id` 与真实 `err` 打到控制台，便于直接定位缺字段、JSON 解析失败或 SQL 错误
- 登录页开发切服面板改为极简模式：`client/scenes/ui/common/dev_server_switcher.tscn` 仅保留下拉框，移除手填 HTTP/WS、地址摘要与额外按钮；`client/scripts/ui/common/dev_server_switcher.gd` 改为切换选项即立即应用环境，减少本地调试误操作
- 客户端默认网络环境切回本地：`client/autoload/network_config.gd` 将原生端与 Web 端默认环境都调整为 `local`，并在登录页切服面板初始化时主动应用本地环境，确保登录页默认直接走 `http://127.0.0.1:8080` 与 `ws://127.0.0.1:8080/ws`

## 2026-07-06
- Web 本地调试画布改为铺满浏览器窗口：`client/autoload/web_runtime_canvas.gd` 在 Debug Web 运行时不再强制 `780:1440` 竖屏比例，避免桌面浏览器调试时画布宽度过窄、可视视口显示不全；正式 Web 构建仍保留 `780:1440` 移动端比例约束
- Web 画布尺寸策略改为“固定比例、允许同比例缩放”：`client/autoload/web_runtime_canvas.gd` 不再把浏览器 `canvas` 锁死为 `780x1440` 像素，而是统一按 `780:1440` 纵横比在当前可视区域内自适应缩放；登录页与主运行态都会共用同一套全局比例约束，既避免拉伸变形，也兼容不同浏览器窗口大小
- Web 运行时世界渲染改为固定设计尺寸：`client/scripts/feature/world/world_controller.gd` 在 Web 环境下不再把内部 `SubViewport` 跟随 `GameShell` 当前尺寸，而是强制按 `780x1440` 渲染，专门兜底 `tmp_js_export.html` 这类临时调试页把实际内部视口压成 `621x834` 的问题
- Web 导出壳层改为固定比例盒子：`client/export_presets.cfg` 的自定义 CSS 不再强制固定 `#canvas` 像素尺寸，而是统一使用 `780:1440` 纵横比居中显示，确保正式导出页与运行时兜底策略一致
- 客户端新增全局静音背景音保活器：`client/autoload/background_audio_keeper.gd` 通过 `AudioStreamGenerator` 持续播放 0 振幅静音流，并作为自动加载单例贯穿所有场景；主要用于尽量减少切后台时音频上下文被立即回收的概率，但浏览器/移动系统后台冻结仍不保证完全不断线
- 修复主菜单运行态节点绑定报错：`client/scripts/ui/main_menu.gd` 不再只依赖 `%TabsRow` / `%ItemsList`，当 unique-name 绑定因 owner 变化失效时会回退到稳定场景路径，避免 `_ready()` 因 `Node not found` 中断主菜单初始化
- 修复市场地图 TileSet 越界：`client/scenes/maps/fashtown/radiant_market.tscn` 的 `TileSetAtlasSource_rbgk4` 删除超出 `Tilemap_Platform.png` 高度上限的 `0:24`~`0:27` 条目，避免切图到市场时触发 `Cannot create tile` 与 `atlas has no tile` 报错
- 客户端网络入口切换为统一多环境解析：`client/autoload/network_config.gd` 现支持 `local`、`remote`、`browser_origin` 三种环境，并区分原生端默认环境与 Web 默认环境，避免再分别修改 HTTP / WebSocket 脚本
- Web 调试支持灵活切服：浏览器运行时优先读取地址栏参数 `server`、`http_base`、`ws_base`，其次读取 `localStorage` 中的 `pp_server_profile`、`pp_http_base`、`pp_ws_base`；便于本地打开导出页时在本地后端、远程服务和同源部署之间快速切换
- `http_client.gd` 与 `net_client.gd` 不再各自强制改回当前页面同源地址，统一改为读取集中配置解析结果，修复 `http://localhost:8060/tmp_js_export.html` 调试时请求总被锁到 `http://localhost:8060` 的问题
- 登录页新增开发切服面板：`client/scenes/ui/common/dev_server_switcher.tscn` + `client/scripts/ui/common/dev_server_switcher.gd` 支持运行时切换本地后端 / 远程服务 / 浏览器同源，应用后会同步刷新 HTTP / WebSocket 入口并清理旧会话，避免把旧服 token 带到新服

## 2026-07-03
- 奖励弹窗右上角关闭按钮增加专属直连：`reward_popup.gd` 现在对 `%TopCloseButton` 显式绑定 `pressed -> close_popup()`，不再只依赖模态基类的通用关闭按钮链路
- 奖励弹窗模板节点改为可选兜底：`reward_popup.gd` 不再强依赖 `PlainLineTemplate` 等模板节点，场景少某个模板时会自动回退到运行时创建默认行，避免 `_ready()` 因 `Node not found` 直接报错
- 修复奖励弹窗右上角关闭按钮仍无响应：`reward_popup.gd` 改为像确认弹窗一样只走 GUI 点击链路，不再启用基类全局 `_input` 吞事件，并补齐标题面板的鼠标交互
- 修复奖励弹窗标题与正文面板重叠：`reward_popup.tscn` 的正文容器改为在 `VBoxContainer` 中按正常高度参与排版，移除把内容向上顶回标题区域的负偏移，恢复两个边框格子的间距
- 奖励弹窗正文布局改为以场景模板节点为准：`reward_popup.gd` 不再在脚本里硬编码奖励行字号、图标尺寸和行布局，改为复制 `reward_popup.tscn` 中的文本/富文本/物品行模板，后续可直接在场景里调布局
- 战斗进场网格转场放大：`grid_spread.gd` 提高 `spacing`，`grid_spread.tscn` 放大圆点 `QuadMesh` 尺寸，让转场网格更疏、圆圈更大
- 战斗进场网格转场提速：`grid_spread.gd` 下调 `stagger_delay` 与 `scale_duration`，让世界进入战斗时的铺展/揭开动画更利落，减少等待感
- 适配重构后的奖励弹窗场景层级：`reward_popup.gd` 现在按 `reward_popup.tscn` 新结构重新绑定 `DimLayer`、正文 `CenterContainer` 与 `PanelContainer`，恢复遮罩关闭、正文点击拦截和右上角关闭按钮交互
- 修复背包装备预览面板节点路径错误：`equipment_preview_panel.gd` 改为匹配 `equipment_slot_1.tscn` 真实层级，移除不存在的 `MarginContainer` 路径，恢复背包运行态初始化
- 修复战斗结算奖励弹窗右上角关闭按钮无响应：`reward_popup.gd` 现在在打开前显式恢复正文面板与 `TopCloseButton` 的可交互鼠标过滤，并把 `close_popup()` 收敛到 `_dismiss_modal()`，保证点击 `X` 时能正常关闭并广播 `popup_closed`
- 客户端世界场景改为 3 倍放大显示：修正世界相机缩放方向判断后，`player.tscn` 的 `camera_zoom_scale` 调整为 `3.0`，在当前统一 `780x1440` 设计分辨率下，让地图、人物与同层世界内容按旧 `260x480` 视野放大约 3 倍显示，而不影响 UI 分辨率
- 客户端统一设计分辨率切换为 `780x1440`：`project.godot` 的 `viewport` 与 `window override` 同步放大，`world_controller.gd` 的默认渲染基准与 `main.gd` 的战斗弹窗固定尺寸也一并调整到同一口径，保证 UI、地图、角色与战斗继续共用同一套分辨率
- 修复主菜单场景缺少 `TabsRow` / `ItemsList` 内容节点导致 `main_menu.gd` 在 `_ready()` 期间报 `Node not found`；`main_menu.tscn` 已补齐 `MenuFrame/Content` 下的最小容器结构，恢复主菜单初始化

## 2026-07-02
- 新增公开注册接口 `POST /api/v1/auth/register`：当前只要求账号、密码和男女形象选择；服务端复用账号名作为玩家名，男性默认 `初始形象男_001`、女性默认 `初始形象女_002`，并沿用正式玩家初始属性/背包/钱包创建链路
- 客户端登录页新增注册入口与初始形象选择：输入账号密码后可直接注册男女角色；注册成功后自动登录并进入世界；注册表单新增确认密码与基础前端校验，减少误输密码导致的无效注册
- 主世界 HUD 在背包按钮左侧新增“设置”按钮；设置菜单复用通用 `ActionMenuPopup`，提供“返回登录页”“退出游戏”两个入口，其中退出游戏补充二次确认弹窗，分别沿用现有登录页切换清理链路与客户端退出链路
- 客户端网络入口改为集中配置：新增 `client/autoload/network_config.gd`，统一维护 HTTP / WebSocket 地址；默认启用正式服 `117.72.124.51:9002`，并保留注释态本地 `127.0.0.1:8080` 配置，便于开发时通过切换注释在本地与服务器之间快速切换
- `http_client.gd` 与 `net_client.gd` 不再各自硬编码地址；浏览器 Web 导出时也统一从 `network_config.gd` 读取端口，避免只改原生端地址却漏改 Web 入口

## 2026-06-29
- 客户端运行态面板打开链路继续统一 loading：`main.gd` 中「个人状态」快捷键与主菜单入口改为和背包一致，先展示全屏通用 loading，等待 `ENTER_WORLD_REQ + BAG_LIST_REQ + WALLET_QUERY_REQ` 权威数据就绪后再打开 `player_status_panel`；避免旧数据先展示、再闪成新数据
- 客户端面板预加载判定继续收严：`player_status_panel.gd` 与 `bag_panel.gd` 在 opening loading 阶段必须等待各自依赖请求全部成功（人物面板：人物/背包/钱包；背包面板：背包/已穿戴装备）后才允许打开，避免某一条子请求失败时仍带旧快照展示
- 背包已打开后的二次刷新继续收严：普通 `USE_ITEM` 与装备 `REPAIR` 不再在回包到达时立即结束 loading，而是等待后续背包快照真正写入 `GameState` 后再关闭 loading / 提示成功，减少旧数量、旧修复石库存或旧损坏状态闪现

## 2026-06-28
- 强化成功率穿戴等级段：迁移 `070_equipment_enhance_success_required_level_band.sql`（`equipment_enhance_success_config` 复合主键 `target_level + required_level_min`，每10级穿戴段独立配置）；后台「强化成功率」Tab 增加穿戴等级段筛选；强化/预览按装备 `required_level` 解析段并查表；`enhance_preview` 新增 `required_level_band_label`
- 强化材料锻造属性：迁移 `069_equipment_enhance_material_config.sql`（按 `item_id` 配置成功率模式/失败惩罚）；后台物品页在 `item_sub_type=equipment_enhance` 时展示「锻造属性」编辑器；Admin API `GET/PUT /api/admin/equipment-enhance-success-configs` + 物品模板页「强化成功率」Tab 维护 `equipment_enhance_success_config`；强化事务按所选材料计算有效成功率并在失败时分支（损坏 / 降级 / 无惩罚）；`enhance_preview.materials[]` 下发 `effective_success_rate_pct` / `failure_penalty_label`；强化回包新增 `failure_penalty`
- 装备强化失败损坏与修复：迁移 `068_equipment_damaged_repair.sql`（`equipment_instance.is_damaged`、修复宝石 3202、`equipment_repair_cost`）；强化失败时将实例标记为损坏（等级不变、材料仍消耗）；损坏装备不可佩戴/不可强化；WS `2078/2079` 修复接口消耗修复宝石 ×1 清除损坏；背包快照新增 `is_damaged` / `repair_preview`；客户端背包格子红色损坏样式、详情「已损坏」标签、主按钮「修复」+ 确认弹窗

## 2026-06-27
- 消耗品使用效果列表化：后台 `effect_params_json.use_effects[]` 支持人物/宠物/装备/系统多类字段配置；服务端 `use_effects` 通用执行器在同一事务内落地（兼容旧版 `pet_hp_restore` / 扩容 / 神符槽等单效果）；`5021 USE_ITEM_REQ` 新增可选 `target_item_uid`（装备类效果目标）；`5022` 回包新增 `applied_effects` / `needs_wallet_push`；客户端背包新增目标选择弹窗，支持 `pet_single` 选宠与 `equipment_single` 选装备后再 USE_ITEM
- 背包丢弃实例化物品：`DROP_ITEM_REQ` 新增 `item_uid`；服务端按 `player_id + container_type + item_uid` 定位格子并整格丢弃，同步删除 `equipment_instance`；客户端实例化物品（含多件相同装备）丢弃时传 `item_uid`
- 背包丢弃可堆叠物品：确认弹窗新增数量选择（-/+/最大），客户端按所选 `quantity` 部分丢弃；`DROP_ITEM_RESP` 回传 `item_uid` 供提示与刷新
- 战斗/奖励弹窗物品名：4013 结算包 `rewards[]` 在构建时通过 `itemService` 补全缺失的 `item_name`（战斗快照回退路径此前只有 `item_id`）；客户端 `RewardPopup` 在服务端字段缺失时从 `GameState` 背包/已佩戴快照兜底
- 背包丢弃：`5121/5122 DROP_ITEM` 服务端权威丢弃（校验 `can_drop`、写变更日志、推送 `5011`）；客户端「更多 → 丢弃」二次确认后走 loading 请求；背包快照新增 `can_drop` 字段供 UI 控制丢弃按钮
- 系统装备管理「可丢弃」：后台装备模板表单新增 `can_drop` 开关（新建默认开启）；Admin API 与 `item_definition.can_drop` 读写贯通，修复创建时 `can_store`/`is_enabled` 占位错位
- 丢弃确认弹窗：`ConfirmPromptPopup` 关闭基类全局 `_input` 吞事件，修复确定/取消按钮点击无响应
- 丢弃回包刷新：`App._request_cmd_for_response` 补齐 `DROP_ITEM_RESP -> DROP_ITEM_REQ` 映射，修复丢弃成功后 UI 一直等待超时、背包不刷新；`BagController.drop_item_responded` 作为 UI 完成兜底，并修正 `_finish_drop_action` 提前 return
- 装备强化铜币消耗：迁移 `065_equipment_enhance_cost_gold_copper.sql` 为 `equipment_enhance_cost` 增加 `cost_gold_copper`（总铜币真值）；`enhance_preview.cost_gold_copper` 由服务端填充；强化成功/失败均扣铜币并推送 `WALLET_UPDATE_PUSH`；客户端强化页展示消耗铜币与当前金/银/铜余额，余额不足时禁用强化按钮
- 装备强化铜币公式可配置：迁移 `067_item_equipment_enhance_gold_cost.sql` 在 `item_equipment_extra` 增加每件装备独立的强化铜币公式（默认基础 100 铜、每级固定 +200）；运行时按装备模板 `item_id` 计算 `enhance_preview.cost_gold_copper` 与强化扣费；系统装备管理页「编辑装备」弹窗内可配置并预览 +1~+15 消耗

## 2026-06-23
- 客户端新增新版背包一期：主菜单“物品行囊”打开 `bag_panel.tscn`，通过 `BAG_LIST_REQ/USE_ITEM_REQ` 与服务端交互；新增本地 `ItemIconRegistry` 仅映射 `icon_key -> Texture2D`，物品数量/可用行为仍来自服务端快照。
- 客户端左上角头像入口改为打开新版 `player_status_panel.tscn`；旧人物弹窗及其背包/组队/技能旧页资源已移除，新面板直接读取 `GameState` 服务端权威快照展示战斗属性、状态抗性和社会属性。
- 客户端人物信息新面板三个分页按钮补齐 hover 样式，悬停态复用按下态贴图与字体颜色，保证只有普通/按下两种视觉状态。
- 客户端人物信息新面板 `player_status_panel.tscn` 接入分页脚本：打开默认选中“战斗属性”，三个分页按钮保持单选按下/弹起状态，并同步切换战斗属性、状态抗性、社会属性内容面板。

## 2026-06-18
- 战斗伤害公式切换为《口袋伤害计算新表》链路：分子 `(A×SkillMult)×(爆伤链)/100×(1−技能抗性差/100)−D`、分母 `1+Guard×(0.001|0.01)`、综合乘子含天赋/元素/抗类与全局 0.5；删除旧 `def/(def+K)` 与 block 叠乘逻辑；爆伤链直接进分子（不再独立掷暴击骰）；`skillDef` 新增 `skill_mult`/`skill_crit_add`（缺省 `attack_pct/100`）；`actorRuntime` 新增 `guard`/`talent_dmg_pct`/`talent_reduce_pct`/元素字段
- 口袋伤害 DB/Admin：迁移 `062_skill_pocket_damage_fields.sql`（`skill_mult`/`skill_crit_add`）、`063_combat_pocket_damage_stats.sql`（player/player_pet/monster 的 guard/天赋/元素 + 宠物封顶）；技能页/怪物页/宠物实例页可配置上述字段
- 战斗控制双体系：`seal_chance_pct`/`control_chance_pct` 概率无视抗性；`seal_power`/`control_power` 威力对抗控制抗性（差值≥50 稳控，每缩小 1 点降 2%）；迁移 `061_skill_control_power.sql`；文档 `backend/docs/battle-control-effects.md`；Admin 系统技能页「效果」Tab 可配置封印/控制双体系字段
- 人物装备系统 P2 强化：迁移 `060_player_equipment_enhance_cost.sql`（材料表 + 强化石 3201）；WS `2076/2077`；**仅未佩戴且位于背包**时可强化；扣材料 → 掷骰 → 成功升一级（失败不掉级）；客户端背包「强化」按钮；`enhance_preview` 含强化等级行 + 可强化属性行 + `materials` 列表；请求可传 `cost_item_id` 选择强化材料；背包 category `enhance_material` 按 `item_sub_type=equipment_enhance` 筛选
- 人物装备系统 P1：运行时佩戴/卸下 WS `2070`–`2075`；`player_equipment_slot` 事务写入；`equipment/stats.go` 属性重算 + `pet_combat_stat_cap` 截断；背包装备无 `item_uid` 时自动创建 `equipment_instance`；客户端背包「穿戴」、状态页「已佩戴装备/卸下」；`EnterWorld.player.equipped_items`
- 人物装备系统 P0：迁移 `058_player_equipment_foundation.sql`、`059_admin_equipment_permissions.sql`；`module/equipment` Admin CRUD；后台「系统装备管理」页 `/equipment-definitions`
- 玩家人物装备系统设计文档 `backend/docs/player-equipment-system.md`：13 部位、强化成功率、药囊战后恢复、时装纯外观、镶嵌无损取下、属性全额叠加、与宠物共用 `pet_combat_stat_cap`
- 宠物战斗属性封顶 Admin：迁移 `057_admin_pet_combat_stat_cap_permissions.sql`；`/api/admin/pet-combat-stat-caps` GET 列表 + PUT 按 stat_key 更新；后台「战斗属性封顶」页 `/pet-combat-stat-caps`
- 玩家宠物实例独立管理页 `/player-pets`：跨玩家筛选 pet_uid / player_id / pet_id，分页列表 + 详情/编辑/新增/删除/出战开关
- 宠物技能分槽一期：迁移 `054_pet_skill_slots.sql`（天生/神符/普通/法宝槽 + `pet_artifact_equipment`）；设计文档 `backend/docs/pet-skill-slots.md`
- 神符槽道具解锁：`5021 USE_ITEM` 扩展 `effect_type=pet_talisman_slot_unlock`，读取 `pet_skill_slot_unlock_item` 配置；迁移 `055_admin_pet_skill_slot_unlock_permissions.sql` 后台权限
- 法宝装备/卸下 WS：`3031/3032` 装备、`3033/3034` 卸下；技能详情 `3035/3036`；物品 `effect_type=pet_artifact`
- Admin：神符槽解锁配置页 `/pet-skill-slot-unlock`；宠物模板支持编辑 `innate_skill_ids` / `normal_skill_ids`；玩家宠物编辑支持次要战斗属性（精力/命中/抗性等），保存时服务端封顶截断
- 客户端：宠物状态页「查看技能」面板（3035）、背包 usable 道具使用 + 选宠物（5021 + `target_pet_uid`）；法宝装备双入口（背包「装备」3031、技能面板空槽「装备」）
- 可选测试种子：迁移 `056_seed_pet_skill_slot_items.sql`（神符解锁符 3010/3011、法宝 3020/3021）
- 服务端 `module/pet/skill_slots.go` 合并战斗可用技能；`PetDetail.skill_slots` 协议字段（列表页隐藏法宝技 skill_id）
- 宠物战斗属性封顶：迁移 `053_pet_combat_stat_caps.sql` 扩展 `player_pet` 次要战斗字段与 `pet_combat_stat_cap` 配置表；服务端读写与公式重算后强制截断
- 宠物成长一期落地：迁移 `051_pet_progression.sql`、`052_admin_pet_progression_permissions.sql`、`module/petprogression`（经验升级/加点/公式重算）、WS `2063/2064`、Admin `/api/admin/pet-progression/`、后台「宠物成长配置」页
- 客户端状态面板宠物页支持切换宠物、展示成长字段与 +1 加点；战斗结算 `pet_rewards` 扩展升级摘要；新增宠物升级弹窗
- Admin 宠物成长页支持「重算全部宠物战斗属性」运维入口（`POST /api/admin/pet-progression/recalculate-combat-stats`）
- 新增 `backend/docs/pet-progression.md`：基于 `docs/风车做资参考表（v6.2）.xlsx` 反推的宠物升级、自由加点与资质→战斗属性公式
- `backend/docs/protocol.md` 同步 `PetDetail`、`PET_ALLOCATE_ATTR_*`、`BattlePetReward` 扩展字段

## 2026-06-17
- 新增 `docs/形象动画配置指南.md`：UnitSkin 全参数、CHJ/PNG 局部覆盖规则、动画帧新建流程与配置示例
- 客户端 CHJ 战斗待机改为主 CHJ 末尾最后两个动画组合并循环；技能/普攻通过 `UnitSkin.chj_skill_path` 独立 CHJ 补充；`sprite_frames` 可按动画名局部覆盖 CHJ
- 新增 `ChjSprite`、`ChjWorldRenderer`、`CharacterVisual` 双后端；示例皮肤 `CHJ测试_2057` + `client/asset/chj/2057.chj`

## 2026-06-16
- 任务模板后台支持多阶段结构化编辑：每阶段可配置事件类型、目标 NPC/场景、菜单 entry_id、剧情 entry_id 与引导文案；详情页以阶段卡片展示
- 任务运行时按 `objective_id` 顺序推进阶段，未完成前置阶段时不会跳阶段完成后续目标
- `AdminObjectiveInput` 与 `objectives_json.guide` 扩展 `menu_entry_id`、`dialogue_entry_id` 字段，便于运营绑定 NPC 菜单/剧情
- 新增结构化 NPC 剧情系统：迁移 `044_npc_dialogue.sql`、`045_npc_dialogue_more_entries.sql`，引入 `npc_dialogue`、`npc_dialogue_node`、`npc_dialogue_option`、`player_npc_dialogue_session` 表
- 新增 `module/npcdialogue` 领域模块与后台 `/api/admin/npcs/dialogues` CRUD；WebSocket 新增 `2037/2038/2039` 剧情推进协议
- 客户端新增 `NPCDialoguePanel`、`CinematicPlayer`、`PortraitRegistry`、`RequestLoadingOverlay`；NPC 交互/菜单/剧情请求统一走 loading 遮罩后再开面板
- 示例剧情：`93001/dialog_market_intro` 含 action 节点 + `market_limeng_step_aside` 客户端演出；`dialog_market_news`、`dialog_warehouse_intro`、`dialog_trade_tip` 已迁入结构化对话
- 节点 `effects_json` 第一版支持 `notice` 与 `quest_event` 两类服务端副作用
- 新增迁移 `046_npc_shop_goods.sql`：商店商品表 + 市场罗格 `shop_open_market` 改为 `result_type=shop`；`NPC_ACTION_RESP` 增加 `shop` 载荷，客户端接入 `npc_shop_panel` 与 `5101/5102 BUY_ITEM`
- 剧情 `conditions_json` 第一版支持 `quest_id + quest_state` 过滤节点/选项；断线重连 `1022 RECONNECT_RESP` 增加 `active_dialogue` 恢复未结束剧情
- 后台新增 `/npc-dialogues` 独立剧情列表页，复用 `fetchAdminNPCDialogues` 与剧情编辑抽屉
- 后台剧情编辑抽屉支持节点/选项 `conditions_json`（`quest_id + quest_state`）可视化配置
- 后台剧情编辑抽屉支持节点 `effects_json`（`notice + quest_event`）可视化配置，修复后台保存时会清空副作用的问题
- 新增 WebSocket `2042 NPC_MENU_REQ` / `2043 NPC_MENU_RESP`：NPC 菜单拉取与 `2031 INTERACT_REQ` 拆分；对有菜单 NPC 的 INTERACT 请求返回 `use npc menu request`
- 新增迁移 `047_npc_menu_entry_conditions.sql`：菜单项 `conditions_json` 与 `linked_quest_id`；支持按任务状态/分阶段 `objective_id + objective_completed` 过滤可见菜单
- 剧情节点 `effects_json` 扩展 `grant_items`（服务端发物品）与 `accept_quest_id`（进入节点自动接任务）；节点/菜单条件扩展 `objective_id + objective_completed`
- 运行时菜单按玩家任务进度过滤；剧情推进时应用发奖/接任务副作用；菜单动作支持 `quest_accept` / `quest_submit`
- 后台地图 NPC 菜单编辑改为「菜单配置 | 剧情配置」合并 Tab；移除独立 `/npc-dialogues` 导航（旧路由重定向至 `/npcs`）
- 后台剧情表单支持发物品、接任务、任务阶段可见条件可视化编辑
- 新增迁移 `048_npc_scene_only_placement.sql`：`world_entity_definition` 移除坐标/朝向/速度字段，新增 `world_scene_definition` 供后台展示场景中文名；NPC 摆放改由客户端场景资源维护

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
- 人物/宠物最终属性新增数据库快照第一版：新增迁移 `backend/server/migrations/089_player_pet_combat_snapshots.sql`，引入 `player_combat_snapshot` 与 `player_pet_combat_snapshot` 两张表，专门承接服务端实时计算后的最终战斗属性，不再要求客户端、后台和战斗入口直接信任 `player` / `player_pet` 中的最终值列
- 新增统一主属性公式层 `backend/server/internal/module/combatcalc/formula.go`：五大主属性现统一按 `（基础值 + 加算）× 百分比乘算` 口径计算，首期已接入人物快照重算；人物装备、成长加点后的最终 `hp_max/atk/def/spd/mana` 均由服务端按统一公式刷新后写入快照表
- `backend/server/internal/data/postgres/combat_snapshot_repo.go` 新增 PostgreSQL 快照读写：玩家读取时会先按 `base_* + 加点转换 + 已佩戴装备` 刷新 `player_combat_snapshot`；宠物读取时会先把当前 `player_pet` 运行时数据收敛进 `player_pet_combat_snapshot`，再由服务层统一补上被动技能常驻属性折算
- 人物主查询口径已切到快照优先：`backend/server/internal/data/postgres/player_repo.go` 的 `FindByPlayerID` 现每次优先刷新并读取 `player_combat_snapshot`；`backend/server/internal/module/player/service.go` 的开战快照、后台玩家列表与详情也会覆盖为最新最终属性，确保客户端、后台和战斗入口看到的是同一份结果
- 宠物主查询口径已切到快照优先：`backend/server/internal/module/pet/service.go` 的宠物列表、编队、单宠读取以及后台宠物列表/详情，现会先刷新/读取 `player_pet_combat_snapshot`，再在服务层补永久被动属性折算，避免客户端、后台、战斗入口各自从不同链路拼最终值
- 装备重算不再把最终属性硬写回 `player`：`backend/server/internal/data/postgres/equipment_runtime_repo.go` 的佩戴/卸下/重算链路现在改为刷新人物战斗快照；`player` 表中的 `base_*` 继续作为基础真源，装备改动只影响快照，不再污染基础值
- 玩家奖励与后台编辑已同步维护基础真源：`backend/server/internal/data/postgres/player_repo.go` 的后台更新会同步写回 `base_hp_max/base_atk/base_def/base_spd/base_mana`；`AddRewardAttribute()` 在奖励五大主属性时也会改写对应 `base_*` 字段，避免奖励把最终值直接写死进运行态读取口径
- 运行时快照体系继续补第二层视图快照：新增迁移 `backend/server/migrations/090_runtime_view_snapshots.sql`，引入 `player_equipment_snapshot` 与 `player_skill_progress_snapshot` 两张表；当前装备面板读取、人物武器技能进度读取以及战斗开战前的人物武器技能输入，都会优先走数据库快照表，不再直接依赖原始佩戴表或技能进度表
- 新增统一运行时快照刷新服务 `backend/server/internal/module/runtimeview/service.go`：把人物战斗属性、宠物战斗属性、已佩戴装备视图和人物技能进度视图刷新收口成单一入口；`bootstrap.go` 已在进入世界、装备面板、战斗开战等主链路注入该服务，减少不同入口各自拼刷新逻辑
- `backend/server/internal/data/postgres/runtime_view_snapshot_repo.go` 新增 PostgreSQL 视图快照实现：`EquipmentRepository.ListEquipped()` 现会刷新并读取 `player_equipment_snapshot`；`PlayerSkillProgressRepository.ListByPlayerID()` 现会刷新并读取 `player_skill_progress_snapshot`；战斗处理器读取武器类型、武器附加技能与技能学习进度时，已经统一复用这两张视图快照表
- 装备操作与技能进度写链路已同步双更快照：`backend/server/internal/data/postgres/equipment_runtime_repo.go` 在佩戴、卸下、模板热刷新后会同步刷新装备视图快照；`backend/server/internal/data/postgres/player_skill_progress_repo.go` 在战斗结算落库技能经验后会同步刷新技能进度快照，尽量保证操作后下一次查询直接命中新结果

## 2026-07-09 装备强化弹窗关闭按钮常驻
- 客户端装备强化弹窗在强化进度演出锁定期间不再禁用右上角关闭按钮节点，避免通用关闭按钮的 disabled 空样式导致按钮视觉消失；关闭行为仍由演出锁定状态拦截，强化完成后恢复正常关闭。

## 2026-07-09 背包装备详情背景修正
- 客户端背包装备物品详情移除详情面板内部的屏幕采样模糊背景，改为普通半透明底色，避免打开详情时背景直接采样到世界场景；详情现在作为背包内层弹窗叠在背包面板之上。

## 2026-07-09 任务面板真实数据与交付闭环
- 任务列表协议 `QuestSummary` 补充奖励预览与目标事件类型，客户端任务面板现在直接读取 `GameState.quests` 中的服务端任务快照，按主线/支线/日常分类展示标题、当前目标和进度。
- 任务提交改为必须处于 `READY_TO_SUBMIT` 且目标全部完成，禁止客户端直接提交未完成任务把进度强制补满；未完成提交会返回 `quest not ready to submit`。
- Godot 任务面板复用现有任务卡片节点填充真实数据，支持领取、追踪、交付三类操作；交付成功后沿用现有任务结算与统一发奖服务推送钱包、背包、宠物等奖励更新。

## 2026-07-09 任务列表滚动动态卡片与面板领奖
- 客户端任务面板的主线、支线、日常列表改为 `ScrollContainer` 承载动态卡片容器，列表内容可上下滑动查看；每条服务端 `QuestSummary` 只实例化一个 `task_list.tscn` 卡片，任务图标本期先隐藏不渲染。
- 任务卡片进度条严格使用服务端目标进度：`min_value=0`、`max_value=target`、`value=current`、`step=1`，杀怪、经验、对话等任务都统一按 `current/target` 显示。
- 服务端目标完成后统一进入 `READY_TO_SUBMIT`，不再因 `SubmitMode=AUTO` 直接完成发奖；无交付 NPC 的任务由任务面板显示“领取”并发送 `QUEST_SUBMIT_REQ` 领取奖励，有交付 NPC 的剧情主线/支线仍只提示前往 NPC 交付。

## 2026-07-09 任务图标 ID 下发与客户端本地映射
- 任务模板新增 `client_icon_id` 数据库字段并通过 `QuestSummary.client_icon_id` 下发；服务端只传任务图标 ID，不下发客户端资源路径。
- 客户端新增 `TaskIcons` 任务图标注册表，按服务端 `client_icon_id` 解析 `resources/task_icons/` 下的本地图标资源；未命中时回退到默认任务图标。
- 任务卡片图标改为动态读取 `client_icon_id` 对应贴图，当前预置 `1=主线默认`、`2=对话任务`、`3=战斗任务` 三个占位图标。

## 2026-07-09 任务图标改为客户端图标 ID
- 任务图标字段明确改为 `client_icon_id`，表示该值由客户端 `TaskIcons` 注册表定义；服务端任务模板只保存并下发引用 ID。
- `client_icon_id` 不设置唯一约束，多个任务模板可以配置同一个客户端图标 ID，从而复用同一张任务图标。
- 后台任务模板列表、详情、新增和编辑表单已同步展示与保存 `client_icon_id`。

## 2026-07-10 后台富文本双栏编辑与可视化刷色
- 后台统一 `RichTextEditor` 改为上下排列：上方文本框只显示不含标签的可读文字，下方实时渲染客户端效果。
- 右侧支持拖选文字后使用颜色笔刷，刷色会回写为 `[color=#RRGGBB]...[/color]`，不改变服务端存储协议。
- 固定常用色为：绿 `#2AFF2A`、金 `#FFFF00`、蓝 `#00FFFF`、橙 `#FF7D00`、粉 `#FF64FF`、红 `#FF0000`。
- 物品、装备、宠物、技能、任务、NPC 对话等现有富文本入口全部复用该双栏编辑器；原有关闭预览开关已移除。
- 客户端任务卡片描述节点改为 `RichTextLabel`，直接渲染服务端 BBCode，避免显示原始颜色标签。
- 根据最终交互收口上方输入区：只保留纯文本框，不向运营暴露 BBCode 源码；删除标题、格式工具、系统模板/示例按钮和底部说明，颜色区新增常用白 `#FFFFFF`。
- 组件内部会在纯文本修改后重建 BBCode，未变文字保留原格式，服务端存储和客户端渲染协议不变。
- 修复已有颜色文字重新刷色时产生嵌套 `[color]` 标签的问题；现在会先移除选中字符的旧颜色，再写入新颜色，避免预览出现残留标签或乱码。

- 新增服务端权威场景剧情触发链路：`2044 SCENE_TRIGGER_PUSH` 会在玩家首次进入东路时下发 `初见桃子`，客户端播放完后通过 `2045 SCENE_TRIGGER_ACK_REQ` 确认，服务端再落库剧情 flag、解锁桃子 NPC 并接取第一条主线任务。
- 新增迁移 `101_story_scene_entry_trigger.sql`：落地 `player_story_flag`、`scene_entry_trigger` 和 NPC 个人可见性条件，桃子 NPC 在 `taozi_npc_unlocked` 前不会进入该玩家的世界快照。

## 2026-07-13 固定剧情对话名字框与场景头像修正
- 客户端对话名字框高度调整为 `60px`，动态高度计算同步计入背景纹理上下边距，避免角色名和头像被边框裁切。
- 客户端固定过场对白链路新增可选场景头像纹理参数；服务端 NPC 对话仍沿用 `PortraitRegistry`，保持原有头像解析行为不变。
- “初见桃子”固定剧情中的桃子、七色羽和玩家对白改为读取场景角色当前显示帧，正式客户端与 F6 单独运行均使用一致形象。

## 2026-07-13 初见桃子后续剧情动作与对白
- 冲击波结束后，七色羽向上移动 `18px` 并以镜像左向动画表现右向行走，到达后保持右向待机；随后桃子向左移动 `10px`。
- 玩家新增固定偏移移动逻辑，本段向左移动 `10px` 时播放 `walk_left`，到达后切换为向上待机。
- 追加七色羽、桃子与玩家共六句本地对白，继续使用场景角色当前帧作为名字框头像；冲击波播放完成后恢复隐藏。

## 2026-07-13 重复剧情对白正文不显示修复
- 修复 `RichTextLabel.clear()` 后再次赋相同 `text` 时 Godot 不重建解析缓冲的问题；正文改为按 BBCode/纯文本分别使用 `append_text()`/`add_text()` 强制写入。
- 连续两句内容完全相同的剧情对白现在都会正常启动打字机并显示文字，不改变服务端对话或富文本格式。

## 2026-07-14 时光小屋地图脚本接入
- `时光小屋.tscn` 已绑定独立地图脚本，补齐 HUD 场景名、移动端缩放、默认出生点和 TileMap 合并边界居中能力。
- 场景已有的 `LeftDoor` 使用 `portal_id=7001` 单向进入 `scene_id=2` 的东路东侧落点；东路 `RightPortal` 不绑定返回逻辑。
- 服务端权威地图关系新增 `scene_id=7`，仅允许 `7 -> 2`；迁移 `099_time_house_scene.sql` 注册时光小屋场景元数据。
- 时光小屋原“墙壁”图层统一命名为 `Collision`，接入现有世界控制器的地图原点归一化、相机边界、背景铺放和点击寻路链路。

## 2026-07-14 新增地图 NPC 表单优化
- 后台新增/编辑地图 NPC 弹窗改为“身份信息 + 归属与发布”分区布局，NPC 富文本显示名使用完整宽度编辑和预览。
- 场景选项同时展示场景名称与 `Scene ID`，并补充服务端场景未注册时的提示；地图 NPC 的实体类型固定为类型 `2`。
- 移除新增表单里的测试实体 ID、编码、名称和默认场景，避免运营误提交测试数据；弹窗接入统一固定高度内部滚动布局。
- 地图 NPC 的实体 ID 和编码改为服务端自动生成：数据库 sequence 原子分配 ID，编码固定为 `npc_{entity_id}`；编辑时两者只读且不可修改。
- 新增迁移 `100_npc_entity_auto_identity.sql` 初始化序列，创建接口仅接收显示名、实体类型、所属场景和状态。
- 地图 NPC 新增/编辑表单移除全部说明性提示卡、字段补充说明和说明型占位文案，仅保留字段标签、分区与校验错误。

## 2026-07-15 初见桃子技能黑闪与震屏
- “凤凰神炎”冲击波第一帧出现时增加短促全屏黑闪，黑色遮罩快速退回透明后再触发相机震动。
- 冲击波移动与黑闪/震屏并行播放；演出结束或剧情被中断时恢复相机基础偏移并隐藏遮罩，避免状态残留。
- 修复剧情脚本依赖 `PlotImage` 全局类缓存导致的类型解析失败；改用场景节点真实基类并显式调用图片方法。

## 2026-07-15 背包物品详情显示修复
- 修复背包详情场景默认隐藏后，点击物品只显示遮罩、不显示详情面板的问题。
- 打开和关闭详情弹层时显式同步详情面板可见状态，保留场景资源默认隐藏约束。

## 2026-07-15 闪光镇宠物学校固定剧情
- 完成“闪光镇宠物学校”客户端固定过场：按既定坐标摆放薇安、桃子和玩家，并串联迎接、对白、桃子离场及玩家走向薇安的动作。
- 剧情对白复用现有富文本面板与场景角色上半身头像，正式客户端和单独运行场景均可推进对白。
- 过场地图原点、相机缩放及边界复用世界场景同步逻辑，结束时通过统一剧情完成信号恢复主流程。
- 调整迎接动作时序：玩家先向下移动 `60px`，完成并转向左侧后，桃子才向右移动 `50px`。
- 修正剧情收尾方向：桃子离场后，玩家改为向左移动 `100px`并保持左向待机。
- 桃子迎接玩家向右移动 `50px` 后停止行走，并切换为右向待机状态。
- 宠物学校剧情收尾的玩家左移距离增加到 `120px`，并复用“初见桃子”的顶部闪烁“剧情”图片，演出结束或中断时自动隐藏。

## 2026-07-15 桃子与散文固定剧情
- 完成“桃子与散文”客户端固定过场，按注释实现桃子、七色羽和玩家的站位、串行/并行动作及全部富文本对白。
- 场景复用北路地图相机同步、角色上半身头像、备用对话面板和顶部闪烁“剧情”图片。
- 结尾新增场景节点承载的全屏渐黑演出；黑屏层位于对话面板下方，渐黑后仍可显示桃子的内心独白，并在剧情中断时清理残留状态。
- 桃子黑屏内心独白由玩家点击结束后立即取消黑色遮罩，再发送剧情完成信号恢复世界画面。
- 屏幕完全渐黑后关闭桃子和七色羽的场景显示状态，避免独白结束取消黑屏时两个角色短暂闪回。

## 2026-07-15 对话人物名字字号统一
- 通用 NPC 对话面板中的 NPC 名字和玩家名字由 `24px` 调整为 `32px`，与正文文本字号保持一致。
- 名字框继续使用现有动态宽高测量与角色头像布局，服务端对白和剧情调用方式不变。

## 2026-07-15 剧情场景出战宠物跟随
- 所有继承 `WorldPlayerCinematic` 且包含本地 `Player` 节点的固定剧情，统一读取服务端权威 `GameState.lineup[0]` 显示首只出战宠物；没有出战宠物时不显示。
- 剧情宠物复用世界场景的 `WorldPetFollower` 与 `PathFollowController`，按相同的 24px 路径步长、起步延迟、等速移动和皮肤动画跟随剧情玩家。
- 跟随能力收口在剧情基类，现有“初见桃子”“闪光镇宠物学校”“桃子与散文”以及后续同结构剧情无需各自重复接入。

## 2026-07-15 机械熊剧情泛光颜色修正
- 机械熊剧情场景发光源由绿色 HDR 色改为等通道白色 HDR 色；保留原有 `WorldEnvironment` 泛光强度、范围与混合配置。

## 2026-07-15 机械熊单节点 Shader 泛光
- 移除机械熊剧情发光特效中的全局 `WorldEnvironment`，改为发光节点自身的白色径向加法 Shader，避免影响同一 Viewport 内的地图、角色和 UI。
- 发光矩形以原 `(316, 421)` 中心点扩展为 `64x64px`，保留亮点位置并为 Shader 光晕提供绘制空间。

## 2026-07-15 机械熊剧情 NPC 局部模糊
- 仅在机械熊剧情场景的机械熊、七色羽实例子节点上覆盖纹理模糊 ShaderMaterial，不修改两个 NPC 的原始场景，也不影响当前剧情中的地图和其他画面。
- 两个 NPC 共用九点采样模糊材质，默认模糊半径为 `1px`，可在场景实例材质参数中继续调整。

## 2026-07-15 机械熊固定剧情
- 完成机械熊固定剧情的东路开场、桃子受击、七色羽能量爆发、渐黑转场和罗克萨斯家醒来对白。
- 场景补齐桃子、玩家、仓库地图、备用对话面板、剧情图片、黑白闪烁层和渐黑层；机械熊与发光特效在资源层默认隐藏，运行到对应演出段再显示。
- 最终爆发阶段才开启机械熊和七色羽的局部模糊 Shader、白色发光、黑白闪烁与相机震动，避免影响此前对白和其他画面。
- 移除所有屏幕闪烁中的白色闪屏，只保留短促黑屏闪烁；角色序列帧动画和剧情移动速度均调整为原来的 `0.5` 倍。
- “走吧，去收拾东西”对白后桃子先向右领走 `30px`；第二阶段三人按统一 `50px/s` 各自移动，桃子和七色羽先到先停，玩家按较长距离稍后停止，不再强制同步结束时间。

## 2026-07-15 机械熊分帧攻击演出调整
- 机械熊从黑屏中显示时立即播放一次短促震屏，强化角色突然出现的反馈。
- 攻击前桃子向左、机械熊向右同时后退 `15px`；双方移动完成后，第一次纯黑闪烁切换机械熊为“攻击第一帧”，等待 `0.5s`，第二次纯黑闪烁切换为“攻击第二帧”。
- 第二次闪烁完成后桃子立即播放受惊动画并向左退让，玩家与七色羽继续沿用原有退让动作和各自停止时间。

## 2026-07-16 任务领取与交付动画
- 新增迁移 `102_quest_transition_animation_keys.sql`，任务模板可分别配置领取和交付动画注册键。
- `QUEST_ACCEPT_RESP`、`QUEST_SUBMIT_RESP` 以及 NPC 菜单的 `NPC_ACTION_RESP` 成功载荷统一返回 `client_animation_key`。
- 客户端复用 `CinematicPlayer` 播放任务动画；交付动画结束后才展示升级和奖励弹窗，空键或未知键会直接继续流程。
- 后台任务模板表单支持查看、创建和编辑领取/交付动画键。

## 2026-07-16 桃子 NPC 碰撞与交互补齐
- `client/npc/dong-lu/桃子.tscn` 参照仓库罗思补充 `Area2D` 交互感应区和碰撞形状，继续复用 `InteractiveNPCBase` 的玩家进入/离开信号。
- 桃子新增碰撞层为 `4` 的 `StaticBody2D` 脚底阻挡，玩家不能再直接穿过角色纹理。
- 客户端桃子身份由临时 `entity_id=1` 对齐为服务端已注册的 `92001`，显示名由 `taozi` 修正为“桃子”，稳定编码保持 `taozi`。

## 2026-07-16 镇北兔子 NPC
- 新增可复用的 `client/npc/shan-guang-zhen/兔子.tscn`，预留空的“待机下”序列帧供后续在 Godot 中设置纹理。
- 兔子复用桃子的 `InteractiveNPCBase`、`16x7px` 交互区和 `14x5px` 实体阻挡，并以实体 `92002` 实例化到镇北 `(120,120)`。
- 新增迁移 `103_north_road_rabbit_npc.sql`，在 `scene_id=4` 注册 `rabbit`；位置与朝向仅由客户端地图场景维护。

## 2026-07-16 主线任务数据清理脚本
- 新增手动迁移 `104_delete_main_quest_data.sql`，按 `quest_type='MAIN'` 收集任务 ID，并在事务中删除玩家进度、目标、事件和主线模板。
- 同步删除绑定主线的 NPC 菜单与场景触发器，并清理其他任务 `pre_quest_ids` 中已失效的主线引用。
- 脚本不回收已经发放的钱包、背包、宠物和属性奖励，也不自动执行数据库变更。

## 2026-07-16 《主线·旅行的起点》1/5 至 3/5
- 新增迁移 `105_main_journey_start_quests.sql`，配置任务 `1101/1102/1103`、前置关系、金币奖励、场景提示、NPC 菜单及完整结构化对话。
- 桃子、生产导师·璃梦和杂货商人·罗格的主线入口使用对话型菜单；需要接取的阶段均提供“接受/取消”，已流转到璃梦的 1/5 直接进入对话。
- NPC 对话副作用新增 `submit_quest_id`，会在对话结束节点先推进目标，再交付任务、持久化奖励并推送钱包/背包/宠物更新。
- 场景触发器新增 `prompt_text`；初见桃子剧情结束后显示提示再解锁桃子，首次进入市场直接显示寻找璃梦的提示。
- 修复璃梦 NPC 未继承交互基类且缺少实体身份的问题，并将璃梦、罗格的客户端显示名对齐新主线文案。
- 东路正式地图新增桃子实例，位置由客户端地图场景维护；仅当世界快照包含 `92001` 时显示并启用碰撞，剧情实例不受该可见性规则影响。

## 2026-07-16 NPC 注册迁移兼容修复
- 修复迁移 `105_main_journey_start_quests.sql` 仍更新已由迁移 `048_npc_scene_only_placement.sql` 删除的 `pos_x`、`pos_y`、`dir`、`speed` 字段，桃子注册现在只维护场景归属和启用状态。
- 同步修复迁移 `103_north_road_rabbit_npc.sql` 的同类字段引用；桃子与兔子的坐标、朝向继续由各自客户端地图场景资源维护。
- 两份迁移均保持幂等，可在批处理部分失败后直接从头重新执行，无需手工回滚已成功写入的数据。

## 2026-07-16 任务模板编辑表单修复
- 修复任务模板编辑弹窗切换“基础信息 / 任务阶段 / 任务奖励”后字段被卸载并清空的问题，页签切换期间统一保留表单状态。
- 起始 NPC、目标 NPC 以及任务阶段中的目标 NPC 改为可搜索下拉框，选项通过现有后台 NPC 接口分页读取，显示 NPC 名称和实体 ID。
- 保存载荷继续使用原有 `start_npc_id`、`submit_npc_id` 与阶段 `npc_id` 数值字段，服务端协议和数据库结构保持不变。

## 2026-07-16 剧情场景统一黑屏转场
- 所有通过 `CinematicPlayer` 注册播放的独立剧情场景统一使用主场景现有高层黑色遮罩：先渐黑世界画面，在全黑时挂载剧情场景，再渐亮开始演出。
- 剧情发出完成信号后先保持当前剧情场景并渐黑，屏幕全黑时才释放剧情实例，随后渐亮恢复原世界场景。
- 场景剧情、NPC 动作剧情以及任务领取/交付动画共用同一流程；未知动画键仍直接完成，不额外播放空转场。

## 2026-07-16 初见桃子剧情转场修复
- 修复“初见桃子”自定义演出末尾遗漏 `complete_cinematic()`，导致最后一句对白后不发送完成信号、剧情实例无法释放的问题。
- 服务端场景剧情推送在地图黑屏切换期间到达时先进入队列；目标地图加载完成后保持全黑，直接挂载剧情场景并渐亮，不再先展示目标世界画面。
- “时光小屋 → 东路 → 初见桃子”的时序现在固定为地图黑屏切换、黑屏内挂载剧情、剧情渐亮、剧情结束渐黑退出、恢复东路后 Ack 服务端。

## 2026-07-16 后台永久删除禁用账号
- 玩家列表原“删除”操作明确调整为“禁用账号”，继续将账号和玩家状态软删除为 `0`，保留现有数据恢复边界。
- 已禁用账号新增“永久删除”操作，使用不可恢复的二次确认；服务端 `DELETE /api/admin/players/{player_id}/purge` 会再次校验账号及其全部玩家均已禁用。
- PostgreSQL 仓储在单一事务内按依赖顺序删除账号下的宠物、编队、装备、镶嵌、背包、仓库、钱包、流水、任务、剧情、战斗、技能成长、属性分配、玩家和账号数据，任一步失败整体回滚。
- 新增迁移 `106_admin_player_account_purge_permission.sql`，永久删除使用独立 `players:purge` 权限，并默认授予 `super_admin`；普通 `players:edit` 权限不能调用该接口。
- 删除成功后服务端会断开该玩家可能残留的在线会话，避免客户端继续使用已不存在的玩家身份。

## 2026-07-26 闪光镇地图节点 UI

- 使用 Godot AI MCP 新增 `map_teleport_panel.tscn`，以现有闪光镇地图图片展示 7 个可点击标点，并提供清晰的唯一选中态。
- 地图面板支持屏幕“上一个 / 下一个”按钮以及键盘、手柄 `ui_up/ui_down` 循环切换；右上角关闭按钮和 `ui_cancel` 均可退出。
- 主场景 HUD 新增“地图”按钮，并通过独立 `MapPanelController` 管理面板互斥、世界输入锁和战斗态自动关闭。
- 地图标点现在采用“首次点击选中、再次点击传送”；客户端只提交目标 `scene_id`，服务端从 `world_map_teleport_node` 校验开放状态并读取中心格。闪光平原尚无目标场景，因此保持可选但提示未开放。

## 2026-08-02 闪光平原场景注册与权威传送
- 新增迁移 `112_shining_plain_scenes.sql`，将闪光平原宠物学校、梦境区域、办公/商业/报名/准备区、家族会馆、南路、湖泊、沼泽、海岸、尘泥之地和精灵大厅注册为 `scene_id=10..25`。
- 服务端 `worldScenes` 补齐闪耀广场及闪光平原已落地地图的入口、出口、传送门编号和权威出生格；测试桩同步相同拓扑。
- 客户端 `WorldSceneRegistry` 新增 `scene_id=10..25` 映射，通用地图脚本支持由场景资源配置门节点与服务端 `portal_id`。
- 补齐宠物学校、办公区和家族会馆缺少的回程门节点；尚无目标场景资源的海道、战斗区和闪光平原传送区入口保持未启用。
- 校验全部启用传送门的目标出生格，修正准备区和家族会馆落点，避免人物切图后落到地图外或碰撞阻挡格。

## 2026-08-04 闪光平原普通门出生坐标补齐

- `shining_plain_level.gd` 新增场景级 `portal_spawn_scene_positions` 配置读取，目标地图可按来源 `portal_id` 返回独立出生场景格；未配置入口继续返回无效坐标并沿用服务端兼容落点。
- `shining_square_level.gd` 补齐闪光镇传送区、宠物学校、办公区、商业区、报名区和闪光南路返回闪耀广场的六个入口坐标。
- 闪光平原 15 张存在已启用入口的地图场景补齐全部来源门出生坐标；没有正式目标资源的闪光平原传送区、战斗区和海道相关门仍保持未启用。
- 普通门请求仍提交 `target_scene_id + portal_id + target_pos`，服务端继续负责验证门拓扑、持久化权威位置并广播世界快照；本次没有协议、数据库或后端逻辑变更。
- 已通过 Godot headless 检查 33 条入口解析映射及对应出生格，所有出生格均存在地图瓦片且没有瓦片物理阻挡。

## 2026-08-04 海道场景正式接入

- 保留用户新增的 `client/scenes/maps/闪光平原/海道.tscn` 地图内容，只绑定闪光平原通用地图脚本，并注册客户端 `scene_id=26`。
- 闪光海岸、海道和精灵大厅接通双向普通门：`23003` 闪光海岸到海道、`26001` 海道到闪光海岸、`26002` 海道到精灵大厅、`25001` 精灵大厅到海道。
- 四个入口出生格分别为海道 `(6,2)`、闪光海岸 `(8,11)`、精灵大厅 `(2,8)`、海道 `(10,8)`；服务端 `worldScenes` 与 WebSocket 测试仓储同步相同拓扑。
- 新增迁移 `114_shining_plain_seaway.sql`，注册海道 `scene_id=26` 并开放数据库权威快速传送中心格 `(6,7)`；迁移文件仅生成，未直接执行。
- 世界地图海道热点已映射 `target_scene_id=26`，客户端只提交目标场景，中心坐标继续由服务端数据库读取。
- 已通过 Godot 4.7 Headless 的四门信号、四个入口格、场景注册与地图热点专项检查，主场景无解析错误；`go test ./server/internal/data/postgres` 与 `go test ./server/...` 均通过。

## 2026-08-04 闪光平原地图当前场景定位修复

- 修复地图面板打开时沿用上一次地区、导致玩家身处闪光平原仍显示闪光镇默认热点的问题。
- `map_teleport_panel.gd` 现在根据 `GameState.scene_snapshot.scene_id` 在现有地区热点容器中查找归属，自动打开闪光镇或闪光平原地图，不额外维护重复场景区间。
- 闪光平原 `scene_id=9..26` 打开地图时会选中当前地图热点，并同步定位选择动画与人物当前位置图标；再次点击仍通过现有服务端权威快速传送链路提交 `target_scene_id`。
- 已通过 Godot 4.7 Headless 逐项验证 `scene_id=9..26` 的地区切换、选中光标、人物图标和二次点击传送信号。

## 2026-08-05 闪光海岸树木动态遮挡修复

- 修复 `client/scenes/maps/闪光平原/闪光海岸.tscn` 中四棵棕榈树与人物、跟随宠物始终按固定场景层绘制，无法根据脚底与树根位置正确切换前后遮挡的问题。
- 原树木 `TileMapLayer` 继续保留图块物理碰撞，但关闭重复绘制；四棵完整树木精灵改为放入场景既有 `ActorRoot`，并以各自树根作为 Y-Sort 锚点。
- 本次仅调整客户端场景表现层，不修改地图碰撞数据、服务端权威位置、传送门、协议或数据库。

## 2026-08-05 翡翠梦境树木动态遮挡修复

- 修复 `client/scenes/maps/闪光平原/翡翠梦境.tscn` 中右侧两棵棕榈树与人物、跟随宠物始终按固定场景层绘制，无法根据脚底与树根位置正确切换前后遮挡的问题。
- 原树木 `TileMapLayer` 继续保留 17 个图块和两处树根物理碰撞，但关闭重复绘制；两棵树分别以 `(200,144)`、`(184,208)` 为 Y-Sort 树根锚点放入预置 `ActorRoot`。
- 右上边缘原有的单格树叶残片继续保留在固定地图层；本次不修改地图外观、服务端权威位置、传送门、协议或数据库。

## 2026-08-05 报名区进入准备区出生格修正

- 将准备区场景中 `portal_id=17002` 的客户端目标出生格从地图外的 `(-4,1)` 修正为左侧返回门内侧的 `(2,7)`。
- 服务端正式世界拓扑、测试仓储和普通门坐标契约原本已统一使用 `(2,7)`，因此不修改服务端权威传送逻辑、协议或数据库。
- `(2,7)` 已确认存在地图瓦片且没有物理阻挡；另一条 `portal_id=17003` 的比武区入口配置保持不变。

## 2026-08-05 普通门负坐标落点支持

- 修复普通门客户端入口格被服务端限制为非负数的问题，`target_pos=(-4,1)` 等有符号场景坐标现在可在门拓扑和等级验证通过后正常生效。
- 负坐标会与正坐标一样用于权威回包、世界重同步和玩家位置持久化；旧客户端未提交入口时仍回退服务端兼容门点。
- 世界地图快速传送继续使用数据库中心格并忽略客户端落点；本次不修改协议字段、数据库结构或客户端请求格式。

## 2026-08-06 后端启动 Redis 地址修复与错误诊断增强

- 修正 `server/configs/config.yaml` 中已不可达的 Redis 地址，将运行时连接目标从 `38.76.179.44:6379` 更新为当前与 PostgreSQL 同机且可访问的 `117.72.124.51:6379`。
- PostgreSQL 原连接地址、账号、数据库配置保持不变；本次没有数据库结构、迁移、协议或业务逻辑变更。
- 依赖初始化失败现在会附带 `open postgres dependency` 或 `open redis dependency` 上下文，后续连接超时时可直接从启动日志定位具体依赖。
- 使用正式配置完成后端启动回归，服务成功监听 `:8080`，`GET /healthz` 返回 `200` 与 `data.status=ok`。

## 2026-08-10 椭圆宠物轮播 Demo UI

- 新增 `client/scenes/demos/ellipse_carousel_demo.tscn`，使用场景中预置的 12 个宠物图标和移动端左右按钮展示椭圆轮播，不修改正式登录流程或项目启动场景。
- 新增 `client/scripts/ui/ellipse_carousel_demo.gd`：先均分 32 个椭圆基础点，下半圆按 4 点间隔抽样、上半圆按 2 点间隔抽样；抽样间隔、半径、补间时长、缩放和透明度范围均可在检查器调整。
- 左右轮转时，每个图标通过 Tween 移动到相邻对象旋转前占用的旧点位，并同步按点位 Y 深度补间缩放与透明度；图标父节点及全部图标节点均启用 Y 排序。
- 将椭圆纵向半径压缩到 `50`，形成更接近正视角的扁平轨迹并允许前后图标自然遮挡；隐藏仅用于调试的椭圆辅助线，同时将形象边框改为 `96×96` 正方形和 `6` 像素小圆角。宠物图片继续等比缩放、不旋转、不做纵向压缩，确保图片始终正面立着。
- 本次仅新增客户端独立 Demo，不接入正式宠物数据，不修改后端权威逻辑、协议或数据库；已通过 Godot 4.7 脚本解析、场景运行、12 点位/Y 排序/景深与双向旧点位轮转专项测试，并完成 `780×1440` 移动端视口截图检查。

## 2026-08-10 椭圆宠物轮盘抽奖 Demo 运行逻辑

- 将椭圆轮播 Demo 的左右手动轮转改为“开始抽奖”流程：点击按钮后先随机确定结果，再沿当前图标点位逐格轮转，至少完整转动 3 圈并在最后减速停轮。
- 使用 `RandomNumberGenerator` 每次初始化随机种子；两个配置了 `is_rare=true` 的形象分别占 `5%`，其余普通形象平分剩余 `90%` 概率。
- 将“芽芽”和“可可”配置为稀有形象，使用金色边框、加粗描边和金色阴影实现发光效果；其他形象继续使用原蓝色小圆角方框。
- 抽奖过程中禁用按钮，结束后显示“普通/稀有”结果；保留原椭圆点位、前后遮挡、Y 排序、缩放和透明度表现。

## 2026-08-10 椭圆宠物轮盘抽奖减速曲线调整

- 延长轮盘抽奖的整体转动过程：默认至少完整转动 `4` 圈，单步基础时长调整为 `0.045` 秒，最终单步最长时长调整为 `0.32` 秒。
- 新增减速阶段比例配置，默认最后 `32%` 的点位进入减速区间；减速区间使用 smoothstep 曲线，前段保持较快速度，后段连续平滑减速，减少突然停顿感。
- 本次只调整抽奖动画时序，不修改随机结果、稀有概率、点位布局和稀有边框样式。

## 2026-08-10 椭圆宠物轮盘抽奖末段停轮时长调整

- 再次延长轮盘抽奖整体时长：默认至少完整转动 `5` 圈，快速阶段单步时长调整为 `0.06` 秒。
- 将最后减速阶段单步最长时长调整为 `0.58` 秒，并把减速区间扩展到总步数的最后 `48%`，让停轮前有更充足的缓冲时间。
- 末段速度曲线改为三次 ease-out，最后几格逐步拉长移动时间，降低最后一步突然停止的感觉；本次不修改随机概率和轮盘布局。

## 2026-08-10 椭圆宠物轮盘抽奖减速参数适中化

- 将抽奖末段最长单步时长从 `0.58` 秒调整为 `0.42` 秒，缩短最后慢速阶段的停轮时间。
- 将减速区间比例从 `48%` 调整为 `36%`，保留平滑减速效果的同时减少末段拖慢感。
- 整体仍保持 `5` 圈轮转、随机结果、概率配置、点位布局和稀有边框样式不变。
