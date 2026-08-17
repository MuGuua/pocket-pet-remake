# 多人同屏权威移动与同步优化方案

## 1. 文档目的

本文档用于指导 `pocket-pet-remake` 多人同屏移动链路的持续修复与优化，并作为开发过程中的唯一执行清单。

每完成一个可独立验证的小任务，开发者必须同时完成以下动作：

1. 更新本文档对应清单项状态与验证结果。
2. 更新 `backend/docs/changelog.md`。
3. 更新 `backend/docs/task-summary.md`。
4. 如果协议或架构发生变化，同步更新 `backend/docs/protocol.md` 或 `backend/docs/architecture.md`。

## 2. 当前问题

当前多人同屏已经支持同场景玩家进入、离开、移动广播，高精度表现坐标、四方向朝向、起停状态、远端短时预测以及平滑追赶。

仍需解决以下核心问题：

- 同场景移动直接采用客户端 `target_pos`，缺少服务端速度、边界、阻挡和状态校验。
- 普通移动保持单请求在途，实际发送频率受网络 RTT 和数据库耗时限制。
- 普通移动响应未驱动本机权威位置纠偏。
- `move_seq` 和 `scene_version` 尚未形成完整的防重复、防倒退链路。
- 移动过程中同步查询和写入 PostgreSQL，数据库位于实时广播关键路径。
- 远端角色只追赶最新目标点，没有基于服务端时间的快照插值。
- 当前广播范围是整张场景，不是真正的网格 AOI。

## 3. 目标架构

```text
客户端移动输入
    -> WebSocket 移动意图
    -> world 领域服务校验与计算
    -> Redis 原子更新实时权威状态
    -> 服务端广播权威移动状态
    -> 客户端本机纠偏 / 远端快照插值
    -> 定时或关键节点批量持久化 PostgreSQL
```

职责边界：

- 客户端负责输入、轻预测、动画和纠偏表现，不决定最终位置。
- `ws` handler 只负责协议解析、调用领域服务和发送响应。
- `world` 领域服务负责移动规则、序号、速度、场景版本和通行判定。
- Redis 保存在线玩家的短时权威移动状态和待持久化标记。
- PostgreSQL保存永久位置、地图通行配置和版本数据。

## 4. Redis运行态设计

### 4.1 Key

所有 Key 必须复用现有 Redis `key_prefix`：

```text
{prefix}:world:movement:player:{player_id}
{prefix}:world:movement:dirty
{prefix}:world:scene:{scene_id}:players
```

第一阶段只实现玩家移动状态和 dirty 集合；场景在线索引在 AOI 阶段接入。

### 4.2 玩家移动状态

```json
{
  "player_id": 10001,
  "session_generation": 8,
  "scene_id": 3,
  "scene_version": 12,
  "precise_x": 12310,
  "precise_y": 8000,
  "persisted_x": 12,
  "persisted_y": 8,
  "facing_x": 1,
  "facing_y": 0,
  "moving": true,
  "speed": 90,
  "last_move_seq": 302,
  "last_server_tick": 92001,
  "position_version": 1502,
  "dirty": true
}
```

坐标使用千分之一场景格定点整数。Redis 状态设置过期时间并在活跃期间续期，但 TTL 只负责回收，不代替 PostgreSQL持久化。

状态更新必须使用 Lua 或等价的 CAS 操作，至少比较 `session_generation`、`scene_version` 和 `last_move_seq`，禁止使用非原子的 `GET` 后 `SET` 覆盖。

### 4.3 Redis故障策略

- Redis不可用时，不静默切回每个移动包写 PostgreSQL。
- 已在线玩家应收到明确的暂时失败响应并停止接受新的权威位移。
- 切图、进入战斗、顶号和正常停服前必须尝试把最终状态写回 PostgreSQL。
- PostgreSQL短时失败时保留 dirty 标记并重试，不删除 Redis状态。

## 5. 协议演进

### 5.1 `MOVE_INTENT_REQ`

保留现有字段以兼容旧客户端，新增：

- `input`：四方向移动输入，服务端移动计算的主要输入。
- `client_tick`：客户端单调时间，仅用于延迟诊断，不参与权威计时。

`target_pos` 和 `precise_pos` 降级为客户端候选位置，只用于误差分析和旧客户端兼容，服务端不得直接采用。

### 5.2 `MOVE_INTENT_RESP`

新增：

- `corrected_precise_pos`：服务端最终高精度权威位置。
- `server_tick`：服务端生成权威结果的时间基线。
- `speed`：服务端权威移动速度。
- `correction_ignore_distance`：本机预测误差可直接忽略的最大距离，由数据库移动配置派生。
- `correction_snap_distance`：本机预测必须立即吸附的最小距离，由数据库移动配置派生。

### 5.3 `ENTITY_MOVE_PUSH`

正式启用或新增：

- `scene_version`：玩家本次进入场景的代次。
- `move_seq`：当前场景代次内单调递增的移动序号。
- `server_tick`：生成权威状态的服务端单调时间。
- `speed`：服务端权威移动速度。

客户端只接收同时满足 `scene_id`、`scene_version` 和 `move_seq` 条件的新状态。

## 6. 数据库与地图通行数据

服务端必须具备地图边界和静态通行数据，不能依赖 Godot 客户端碰撞作为权威判定。

建议通过迁移新增场景导航版本表，实际字段在实施该任务时结合现有后台结构复核：

```text
world_scene_navigation
- scene_id
- version
- grid_width
- grid_height
- cell_size
- navigation_data
- status
- created_at
- updated_at
```

约束：

- 数据库是发布后导航数据的事实来源。
- Godot 工具只负责从地图资源导出通行位图并生成待执行的 SQL。
- 服务端启动时读取已发布版本到只读内存缓存。
- 后台需要提供查看、上传、发布和回滚能力。
- 动态 NPC、玩家和临时障碍不写入静态导航位图。
- 所有数据库结构变更只生成迁移 SQL，由用户执行。

## 7. PostgreSQL持久化策略

移动期间由 Redis保存最新权威位置，PostgreSQL采用以下写回策略：

| 事件 | 写回要求 |
| --- | --- |
| 持续移动 | 每 3 到 5 秒批量写回 dirty 玩家 |
| 停止移动 | 尽快异步写回 |
| 场景切换 | 必须写回或以更高版本写入新场景位置 |
| 进入战斗 | 必须保存权威返回位置 |
| 断开或顶号 | 尽力同步写回并使旧会话失效 |
| 服务关闭 | 在优雅停机窗口批量刷回 |

周期写回 worker 已按 YAML 配置串行运行：默认每 5 秒使用 `SPOP` 最多领取 100 名 dirty 玩家，逐个读取 Redis 最新权威状态，并携带 `position_version` 条件写入 PostgreSQL；单玩家失败不阻断同批其他玩家，失败编号在批次末统一重新入队。数据库已有相同或更高版本时，本批状态按 stale 安全跳过，不会覆盖新位置，也不会重新入队形成无效重试。应用退出时先停止 HTTP 接入并由 Hub 关闭、等待全部 WebSocket 处理和断线回调结束，再停止周期 worker，在 5 秒窗口内有限排空 dirty 集合，最后关闭 Redis 与 PostgreSQL。排空期间失败的玩家会暂存到本轮结束后统一重入队，避免持续故障时反复领取同一玩家形成无限循环。

迁移 `123_player_position_version.sql` 定义了非负 `position_version BIGINT NOT NULL DEFAULT 0` 字段。普通档案与战斗快照读取都会恢复该版本；Redis 状态缺失时，首次进入世界以 PostgreSQL 版本作为后续移动的递增基线。周期 worker 与关键节点写回统一通过 `UpdatePositionIfNewer` 保证只有更高版本可以覆盖场景和坐标。停止移动会异步读取 Redis 最新状态写回；切图先以严格更高版本写入 PostgreSQL，再用同一版本重建 Redis；进入战斗前不再信任客户端 `SelfPos`，而是同步保存 Redis 权威返回位置；断线和顶号在后续生命周期处理前尽力同步写回；停服按 WebSocket 收口、worker 停止和 dirty 有限排空的顺序完成最终刷回。迁移已于 2026-08-17 执行；P1-09 的完整故障恢复测试仍未完成。

## 8. 客户端表现策略

### 8.1 本机玩家

- 输入后立即本地预测，保证移动端操作响应。
- 收到普通移动响应后比较本机预测位置与服务端权威位置。
- 小误差忽略，中误差平滑纠偏，大误差立即设置。
- 切图、复活和传送始终立即使用权威位置。
- 服务端纠偏只改变位置，不重放奖励、任务或其他业务结果。

### 8.2 远端玩家

- 每个远端实体保存最近 3 到 5 个服务端快照。
- 渲染时间落后服务端约 100 到 150 毫秒，在两个历史快照间插值。
- 快照不足时允许最多 180 毫秒外推，之后停在最后权威位置。
- 停止包优先，旧移动包不能覆盖新的停止状态。
- 网络移动不持续创建 Godot Tween，继续在帧循环中更新表现。

## 9. AOI演进

完成权威移动、Redis运行态和快照插值后，再根据压测数据实现网格 AOI：

- 先增加每场景在线人数、移动请求、广播包数、Redis耗时和写回失败指标。
- 按服务端场景格划分 AOI 网格，网格尺寸根据镜头范围和移动速度确定。
- 玩家跨网格时计算新旧邻域差集，分别发送进入、离开和移动事件。
- AOI可视范围至少比移动端镜头多一圈，避免屏幕边缘频繁创建和销毁。

## 10. 开发清单

状态说明：`[ ]` 未开始，`[~]` 进行中，`[x]` 已完成，`[!]` 阻塞。

### P0：权威移动安全

- [x] P0-01 扩展移动协议结构，保留旧字段兼容性。已新增 `input`、`client_tick`、`corrected_precise_pos` 和 `server_tick` 字段；客户端开始发送输入意图，服务端权威判定仍待后续清单项接入。
- [x] P0-02 在 `world` 模块定义移动状态仓储和权威移动结果模型。已定义 Redis运行态所需的玩家、会话、场景代次、定点坐标、序号和位置版本字段，并由 `world.Service` 持有仓储边界。
- [x] P0-03 实现服务端 `move_seq`、会话代次和场景代次校验。普通移动、普通门和地图快速传送统一推进 Redis序号；旧会话、旧场景、重复和倒退请求均在业务执行前拒绝。
- [x] P0-04 实现服务端四方向输入归一化、速度和最大时间跨度校验。数据库配置、启动缓存、四方向校验、轴容差、服务端时间窗和候选位置裁剪均已完成；后台已提供带权限、操作原因审计、二次确认和运行时即时刷新的配置维护闭环。
- [x] P0-05 增加场景边界数据迁移、仓储、后台维护和运行时缓存。已生成 `119_world_scene_boundaries.sql`，为 1~26 号启用场景初始化千分之一格矩形边界；服务启动时加载只读缓存，普通移动在速度裁剪后继续执行边界裁剪；后台复用 `world_movement:view/edit` 权限提供列表、完整矩形编辑、操作原因、二次确认和运行时即时刷新。迁移已于 2026-08-17 执行，墙体与精细通行判定由 P0-06 的已发布导航位图继续负责。
- [x] P0-06 增加场景静态通行数据迁移、Godot 导出工具、后台发布与回滚。已生成 `120_world_scene_navigation.sql` 建立版本化位图表，使用 `export_scene_navigation.gd` 从正式 Godot 地图和玩家碰撞体严格导出 1~26 号场景，并生成 `122_world_scene_navigation_seed.sql` 初始化 26 个已发布版本；服务启动时强制加载只读缓存，普通移动按路径逐格检查并在首个阻挡格前裁剪，缺少发布导航或起点阻挡时失败关闭。后台复用 `world_movement:view/edit` 权限提供草稿、发布和回滚，发布/回滚后当前进程立即刷新运行时缓存。`121_repair_world_authoritative_spawn_positions.sql` 同步修复快速传送权威中心，服务端默认出生点、普通门目标点和快速传送点均已通过发布位图审计；已补领域防穿墙、缓存替换及后台 HTTP 发布/回滚闭环测试。`120`、`121`、`122` 迁移均已于 2026-08-17 执行。
- [x] P0-07 将移动判定从 WebSocket handler 下沉到 `world` 领域服务。普通同场景移动统一调用 `world.Service.MovePlayer`，由领域层完成 Redis 状态加载、玩家/会话/场景权威校验、旧客户端朝向与起停字段归一化、速度/边界/静态通行计算、序号校验和单次 CAS 推进；handler 只保留协议转换、错误映射、响应及同场景广播。未装配 Redis 的旧服务兼容分支继续使用 PostgreSQL 档案位置，换图流程继续复用既有状态推进。
- [x] P0-08 客户端消费 `corrected_precise_pos` 并实现分级权威纠偏。普通移动响应按 `move_seq` 匹配后更新本机网络基线；小误差按数据库轴容差忽略，中误差按服务端权威速度逐帧消化固定偏移，大误差或拒绝响应立即吸附。吸附上限由数据库速度与最大计算时间窗派生，场景不一致继续等待 `WORLD_RESYNC_PUSH`，旧服务未提供有效阈值时不使用客户端常量兜底。
- [x] P0-09 客户端按远端实体保存最近 `scene_version + move_seq`，丢弃旧包。`GameState` 以实体 ID 保存最近接受的场景、场景代次和移动序号；跨场景、旧代次、同代次重复/倒退序号及离场延迟包均被拒绝，新场景代次允许序号重新开始。全量快照、运行态重置和实体离场会清理对应基线，实体摘要刷新不会误清基线；服务端广播改为携带 Redis 权威移动状态或传送决策中的真实 `scene_version`，未装配移动仓储的兼容链路保留零值。
- [x] P0-10 补充正常、瞬移、越界、穿墙、重复、倒退和切图竞态测试。领域层既有用例覆盖正常权威推进、服务端时间窗瞬移裁剪、场景边界裁剪、静态阻挡路径裁剪、重复/倒退序号与权威会话/场景不匹配；WebSocket 层新增成功移动后倒退序号无副作用测试，以及切图完成后旧场景延迟移动包竞态测试，确认拒绝包不会再次推进 Redis、修改 PostgreSQL 永久快照或广播旧坐标，并会向移动者返回新场景权威位置与 `WORLD_RESYNC_PUSH`。

### P1：Redis实时状态与持久化

- [x] P1-01 扩展 Redis 适配能力并新增移动状态专用仓储。仓储已经通过 provider 注入 `world.Service`，没有把 Redis细节泄漏到 handler 或领域模型。
- [x] P1-02 使用 Lua/CAS 原子初始化和更新玩家移动状态。首次进入、新登录替换、普通移动推进和切图重置均已接入；更新成功后续期并写入 dirty 集合。
- [x] P1-03 进入世界时实现 Redis优先、PostgreSQL回退的恢复流程。重连快照优先使用 Redis最新场景、整数位置和千分之一格位置；缓存缺失时使用 PostgreSQL快照并重新初始化 Redis。
- [x] P1-04 移除普通移动请求中的 PostgreSQL同步档案查询和位置写入。Redis 已启用时，handler 直接调用 `world.Service.MovePlayer` 加载当前场景与位置，成功移动只通过 Lua CAS 推进 Redis 并标记 dirty；切图和未装配 Redis 的旧服务兼容分支继续读取永久档案，旧服务普通移动继续同步写位置。
- [x] P1-05 实现 Redis dirty 玩家集合。移动 CAS 成功后继续通过 `SADD` 生产 dirty 标记；`world.MovementStateRepository` 与领域服务新增批量领取、失败重入队接口，Redis 适配器使用 `SPOP key count` 原子领取玩家编号。领取后产生的新移动会再次 `SADD`，不会被旧批次成功处理误删；PostgreSQL 写回失败的编号可通过单次 Lua 调用重新加入集合，供 P1-06 worker 重试。
- [x] P1-06 实现周期批量写回 worker 和失败重试。应用层 worker 默认每 5 秒最多领取 100 名 dirty 玩家，逐个读取 Redis 最新权威整数位置；单玩家读取或写入失败不会阻断同批其他玩家，失败编号统一重入队。运行参数来自 YAML，批次串行不重叠，应用退出时先等待 worker 停止再关闭数据连接。P1-07 已在该 worker 上补齐版本条件写回。
- [x] P1-07 为 PostgreSQL位置增加版本字段和条件更新迁移。迁移 `123_player_position_version.sql` 已于 2026-08-17 执行；玩家档案恢复 `position_version`，Redis 缺失时沿用数据库版本基线；周期 worker 仅写入更高版本，旧版本按 stale 安全跳过且不重入队。关键节点最终写回已由 P1-08 完成。
- [x] P1-08 在停止、切图、战斗、断线、顶号和停服节点触发最终写回。停止包响应后异步读取 Redis 最新状态；切图使用严格高于 Redis/PostgreSQL 已知版本的新状态先写 PostgreSQL、再以同版本重建 Redis；PVE、野外遭遇、NPC 战斗和 PVP 接受前同步保存服务端权威返回位置，不再采用客户端 `SelfPos`；断线与顶号先写回再执行后续生命周期处理；停服先关闭并等待 WebSocket，再停止 worker、有限排空 dirty 集合并关闭数据连接。
- [ ] P1-09 补充 Redis断开、PostgreSQL短时失败、重启恢复和旧版本覆盖测试。

### P2：弱网与插值表现

- [ ] P2-01 解除普通移动单请求在途限制，保留切图请求互斥。
- [ ] P2-02 实现移动包定频发送与起步、停步、转向立即发送。
- [ ] P2-03 服务端广播真实 `server_tick`、`speed` 和场景版本。
- [ ] P2-04 远端玩家增加固定长度快照缓冲。
- [ ] P2-05 实现历史快照插值、短时外推和瞬移阈值。
- [ ] P2-06 为本机与远端移动增加弱网专项测试。
- [ ] P2-07 在移动端设备上验证帧率、内存和触摸移动响应。

### P3：监控与 AOI

- [ ] P3-01 增加在线人数、移动请求、纠偏、拒绝、广播和持久化指标。
- [ ] P3-02 建立 Redis场景在线索引并处理重连、切图和断开清理。
- [ ] P3-03 根据指标确定 AOI 网格尺寸和可视邻域。
- [ ] P3-04 实现网格进入、离开和移动广播。
- [ ] P3-05 完成 20、50、100 人同场景并发压测。

### 文档与发布

- [x] DOC-01 创建多人同屏权威移动与同步优化方案及维护清单。
- [x] DOC-02 更新架构文档中的 Redis实时位置和 PostgreSQL写回职责。
- [ ] DOC-03 更新完整移动协议字段、兼容策略和错误语义。
- [ ] DOC-04 为数据库迁移补充执行、回滚和发布步骤。
- [ ] DOC-05 每批开发完成后更新变更记录和任务总结。

## 11. 验证矩阵

| 类别 | 必测内容 |
| --- | --- |
| 服务端单元测试 | 速度、边界、阻挡、序号、状态锁、CAS、版本写回 |
| WebSocket集成测试 | 同场景广播、跨场景隔离、旧包、重复包、断线重连 |
| Redis测试 | 初始化、续期、CAS冲突、dirty重试、恢复 |
| 客户端静态检查 | GDScript类型、四空格缩进、信号和资源引用 |
| Godot运行测试 | 本机纠偏、远端插值、停止、转向、传送、切图 |
| 弱网测试 | RTT 100/300ms、抖动、5%/10%丢包、2秒断流 |
| 压力测试 | 同场景20/50/100人持续移动和批量切图 |

## 12. 提交拆分

建议保持每个提交只有一个主要职责：

1. `docs(world): add multiplayer movement optimization plan`
2. `feat(protocol): extend authoritative movement messages`
3. `fix(world): validate movement sequences and authoritative position`
4. `feat(redis): add authoritative movement state repository`
5. `perf(world): batch persist dirty movement states`
6. `feat(client): reconcile local authoritative movement`
7. `perf(client): interpolate remote movement snapshots`
8. `perf(world): add grid based AOI broadcasting`

## 13. 完成标准

本方案完成需要同时满足：

- 客户端不能通过坐标提交实现瞬移、越界或穿墙。
- Redis保存在线实时权威位置，PostgreSQL不处于每个移动包的同步关键路径。
- PostgreSQL最终位置不会被旧批次覆盖。
- 高延迟和短时抖动下，本机操作及时且远端移动连续。
- 旧场景、旧连接和旧序号消息不能覆盖新状态。
- 同场景广播规模具有指标依据，并能按需要切换到网格 AOI。
- 所有清单项、测试、协议和变更记录保持同步。
