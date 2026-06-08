# 任务系统设计方案

## 1. 文档目的

- 本文用于为当前联机版项目补一套可落地的任务系统设计口径，覆盖服务端模块、协议、数据模型、客户端接入方式和实施顺序。
- 本文以当前项目已经存在的双端骨架为前提：
  - 客户端：Godot 4 + `GameState` 全局快照 + `MessageRouter` 协议分发
  - 服务端：Go 单体服 + HTTP 登录 + WebSocket 实时同步 + 服务端权威 world/battle/pet/bag
- 本文优先解决当前版本真正需要的问题：
  - 新手引导主线
  - 地图/NPC/战斗/宠物/背包驱动的任务进度
  - 联机环境下的服务端权威任务推进
  - 日常任务和后续活动任务的扩展基础

本文不在第一版内解决：
- 可视化任务编辑器
- 跨服活动任务
- 复杂多分支剧情树编辑器
- 自动寻路与全自动导航
- 大规模多人协作事件编排器

## 2. 当前项目约束

任务系统必须服从当前项目已经确定的架构边界。

### 2.1 客户端约束

- 客户端不负责判定任务是否真正完成。
- 客户端只负责：
  - 展示任务列表和追踪信息
  - 发起接取/提交/追踪请求
  - 响应服务端推送刷新本地任务快照
- 当前客户端已经具备以下适合直接扩展任务系统的结构：
  - `client/autoload/game_state.gd`
  - `client/autoload/message_router.gd`
  - `client/scripts/bootstrap/main.gd`
  - `client/scripts/common/command_ids.gd`

### 2.2 服务端约束

- 服务端必须拥有任务状态推进和奖励结算的最终权威。
- 任务系统不能要求 world/battle/pet/bag 反向依赖 quest。
- 推荐方式是各领域模块产出标准化领域事件，`quest` 模块消费这些事件推进任务。
- 当前服务端模块已经适合作为任务事件源：
  - `world`
  - `battle`
  - `pet`
  - `bag`
  - `session`

### 2.3 联机游戏约束

- 任务进度不能信任客户端上报结果。
- 所有可影响经济和成长的奖励都必须服务端发放。
- 共享任务进度必须有显式规则，不能默认所有组队成员都共享。
- 断线重连后必须支持任务状态全量恢复，而不能只依赖增量 push。

## 3. 设计目标

任务系统整体需要同时满足下面 7 个目标：

1. 支持主线、支线、日常、活动四类任务
2. 支持地图、NPC、战斗、宠物、背包等多来源任务目标
3. 支持服务端权威推进和奖励发放
4. 支持客户端任务追踪与轻量引导
5. 支持日常刷新和后续周期任务扩展
6. 支持任务配置化，而不是写死在代码里
7. 支持后续增加组队共享和活动任务，而不推翻当前结构

## 4. 核心设计原则

### 4.1 配置与实例分离

- 任务模板是静态配置。
- 玩家任务状态是运行时实例。
- 模板定义“这个任务是什么”。
- 实例记录“这个玩家做到哪一步了”。

### 4.2 事件驱动推进

- world/battle/pet/bag 不直接调用 quest 内部逻辑细节。
- 各领域只发出标准化事件。
- `quest` 模块消费事件并匹配当前玩家已接任务的目标条件。

### 4.3 服务端权威

- 接取条件、完成条件、提交条件、奖励发放全部由服务端判定。
- 客户端只能请求“我要接任务”或“我要提交任务”，不能直接声明“我已经完成了”。

### 4.4 状态快照优先

- 客户端只维护任务快照，不维护独立真相。
- 断线重连后，任务状态以服务端重新下发的列表为准。

### 4.5 逐步演进

- 第一版先做“稳定、清晰、可扩展”。
- 不为了未来所有复杂场景而过早做超重系统。
- 先满足当前项目主线引导和基础日常需求。

## 5. 任务系统总览

推荐新增一个独立 `quest` 领域模块，并形成如下结构：

```text
客户端输入/NPC交互/战斗结算/宠物变化/背包变化
    -> 各领域模块产出标准事件
    -> quest 模块消费事件
    -> quest 模块更新玩家任务状态
    -> quest 模块发放奖励或切换任务状态
    -> quest 模块通过 WS 下发任务更新推送
    -> 客户端刷新任务列表与追踪 UI
```

对应双端职责：

- 服务端：
  - 任务模板加载
  - 玩家任务实例读写
  - 任务事件消费
  - 进度推进
  - 奖励结算
  - 协议响应与推送
- 客户端：
  - 任务列表展示
  - 任务详情展示
  - 任务追踪
  - NPC 头顶标识和提示文案
  - 地图/战斗中轻量进度提示

## 6. 任务分类

建议先把任务按用途分成 6 类。

### 6.1 主线任务

- 用于新手引导和主剧情推进。
- 通常有明确前置关系。
- 一般不可放弃。
- 适合解锁系统功能，例如：
  - 开启宠物编队
  - 开启背包
  - 开启捕捉

### 6.2 支线任务

- 提供地图内容和额外奖励。
- 可由 NPC 发放。
- 可以允许放弃。

### 6.3 日常任务

- 按天刷新。
- 目标短、奖励稳定。
- 适合强化留存行为：
  - 完成若干场战斗
  - 捕捉若干宠物
  - 消耗若干体力或道具

### 6.4 周常任务

- 周期更长，奖励更高。
- 第一版可以先不落协议，但在数据结构中预留周期类型。

### 6.5 活动任务

- 限时开放。
- 配置上支持起止时间。
- 第一版先按普通任务模板处理，不单独做活动编排器。

### 6.6 成就型任务

- 常驻累计目标。
- 不一定要求显式接取。
- 后续可复用 quest 模型，也可以单独拆 achievement 模块。

## 7. 任务状态机

建议统一采用如下状态集：

- `LOCKED`：未解锁
- `AVAILABLE`：可接取
- `ACCEPTED`：已接取，进行中
- `READY_TO_SUBMIT`：目标已完成，待提交
- `COMPLETED`：已提交并领奖
- `EXPIRED`：限时任务过期
- `ABANDONED`：已放弃
- `FAILED`：失败

### 7.1 推荐状态流转

主线/NPC 提交任务的典型流转：

```text
LOCKED -> AVAILABLE -> ACCEPTED -> READY_TO_SUBMIT -> COMPLETED
```

自动完成任务的典型流转：

```text
LOCKED -> AVAILABLE -> ACCEPTED -> COMPLETED
```

限时任务可能的流转：

```text
AVAILABLE -> ACCEPTED -> EXPIRED
AVAILABLE -> ACCEPTED -> FAILED
```

### 7.2 状态流转规则

- `LOCKED -> AVAILABLE`
  - 满足等级、前置任务、时间窗、功能开关等条件
- `AVAILABLE -> ACCEPTED`
  - 自动接取或玩家主动接取成功
- `ACCEPTED -> READY_TO_SUBMIT`
  - 所有目标满足，且任务要求人工提交
- `ACCEPTED -> COMPLETED`
  - 所有目标满足，且任务配置为自动提交
- `READY_TO_SUBMIT -> COMPLETED`
  - 玩家提交成功，奖励发放成功

## 8. 任务模板模型

任务模板建议采用配置化定义，最少包含下面字段。

### 8.1 基础字段

- `quest_id`
- `quest_type`
- `name`
- `title`
- `description`
- `sort_order`
- `chapter`
- `tags`

### 8.2 可见与开放字段

- `visible_conditions`
- `unlock_conditions`
- `accept_conditions`
- `pre_quest_ids`
- `mutually_exclusive_group`
- `min_player_level`
- `time_window`

### 8.3 行为字段

- `accept_mode`
  - `AUTO`
  - `NPC`
  - `SYSTEM`
- `submit_mode`
  - `AUTO`
  - `NPC`
  - `SCENE_POINT`
- `can_abandon`
- `is_repeatable`
- `reset_policy`

### 8.4 目标字段

- `completion_policy`
  - `ALL`
  - `ANY`
- `objectives`

### 8.5 奖励字段

- `rewards`
- `unlocks`
- `post_actions`

### 8.6 引导字段

- `guide_scene_id`
- `guide_npc_id`
- `guide_entity_id`
- `guide_text`
- `toast_text`

## 9. 任务目标模型

建议不要把任务目标硬编码成“杀怪/采集/对话”几个专门结构，而是统一抽象成“事件 + 过滤条件 + 进度规则”。

### 9.1 推荐结构

```text
QuestObjective
- objective_id
- event_type
- required_count
- progress_mode
- target_selector
- extra_conditions
- share_policy
```

### 9.2 常见 event_type

- `TALK_TO_NPC`
- `ENTER_SCENE`
- `INTERACT_ENTITY`
- `WIN_BATTLE`
- `KILL_MONSTER`
- `CAPTURE_PET`
- `USE_ITEM`
- `OWN_ITEM`
- `SUBMIT_ITEM`
- `PET_LEVEL_REACH`
- `PET_IN_LINEUP`
- `COMPLETE_QUEST`
- `DAILY_LOGIN`

### 9.3 常见 progress_mode

- `BOOLEAN`
  - 触发一次即完成
- `ACCUMULATE`
  - 累加次数
- `SNAPSHOT`
  - 读取当前快照是否满足，例如是否拥有某道具
- `DISTINCT_SET`
  - 收集不同目标类型

### 9.4 target_selector 示例

- 场景类：
  - `scene_id = 2`
- NPC 类：
  - `npc_id = 90001`
- 战斗类：
  - `monster_group_id = 3001`
  - `battle_type = npc`
- 道具类：
  - `item_id = 2001`
- 宠物类：
  - `pet_id = 101`
  - `lineup_slot = 1`

## 10. 任务配置示例

下面给出一份适合当前项目第一章主线的配置示意：

```json
{
  "quest_id": 10001,
  "quest_type": "main",
  "name": "first_steps_in_town",
  "title": "初入闪光镇",
  "description": "前往闪光镇东路并与引导员交谈。",
  "accept_mode": "AUTO",
  "submit_mode": "NPC",
  "completion_policy": "ALL",
  "pre_quest_ids": [],
  "objectives": [
    {
      "objective_id": 1,
      "event_type": "ENTER_SCENE",
      "required_count": 1,
      "progress_mode": "BOOLEAN",
      "target_selector": {
        "scene_id": 2
      }
    },
    {
      "objective_id": 2,
      "event_type": "TALK_TO_NPC",
      "required_count": 1,
      "progress_mode": "BOOLEAN",
      "target_selector": {
        "npc_id": 90001
      }
    }
  ],
  "rewards": [
    {
      "type": "gold",
      "value": 100
    },
    {
      "type": "item",
      "item_id": 2001,
      "count": 3
    }
  ],
  "guide_scene_id": 2,
  "guide_npc_id": 90001
}
```

## 11. 服务端领域事件设计

任务推进建议统一依赖领域事件，而不是零散回调。

### 11.1 统一事件结构

建议定义统一事件模型：

```text
QuestEvent
- player_id
- event_type
- scene_id
- battle_id
- source_entity_id
- target_entity_id
- item_id
- pet_uid
- count
- metadata
- occurred_at
- idempotency_key
```

### 11.2 事件来源

#### world 模块事件

- 玩家进入场景
- 与 NPC 交互
- 与传送门交互
- 触发地图脚本点

#### battle 模块事件

- 战斗开始
- 战斗胜利
- 击败指定敌方目标
- 捕捉成功
- 使用指定技能获胜

#### bag 模块事件

- 获得道具
- 使用道具
- 扣除道具
- 提交道具

#### pet 模块事件

- 获得宠物
- 宠物升级
- 宠物进入编队
- 编队设置成功

#### session 模块事件

- 每日首次登录
- 连续登录
- 账号年龄达到阈值

### 11.3 事件处理原则

- 每次事件到达后，只扫描当前玩家“可能被该事件推进”的任务。
- 不应在事件到达后全量遍历所有任务模板。
- 建议按 `event_type` 维护任务目标索引。

## 12. 服务端模块设计

建议新增如下文件结构：

```text
backend/server/internal/module/quest/
  ├─ model.go
  ├─ service.go
  ├─ evaluator.go
  ├─ event_handler.go
  ├─ reward.go
  ├─ config_loader.go
  └─ repo.go
```

### 12.1 model.go

定义：
- 任务模板结构
- 玩家任务实例结构
- 玩家目标进度结构
- 奖励结构
- 事件结构

### 12.2 service.go

负责：
- 列表查询
- 接取任务
- 提交任务
- 放弃任务
- 周期刷新
- 追踪设置

### 12.3 evaluator.go

负责：
- 根据模板和事件判断目标是否推进
- 根据快照类目标判断任务是否满足
- 计算任务是否从 `ACCEPTED` 进入 `READY_TO_SUBMIT`

### 12.4 event_handler.go

负责：
- 订阅 world/battle/pet/bag/session 事件
- 调用 evaluator 推进任务
- 生成任务更新推送

### 12.5 reward.go

负责：
- 奖励发放
- 幂等处理
- 奖励日志

建议不要让 `quest` 直接散落调用多个仓储，而是统一通过奖励结算器处理。

### 12.6 config_loader.go

负责：
- 从 JSON 或 YAML 读取任务模板
- 启动时校验配置合法性
- 构建目标事件索引

## 13. 数据库存储设计

第一版可以采用“模板文件 + 玩家实例入库”的组合方式。

### 13.1 player_quest

字段建议：

- `player_id`
- `quest_id`
- `state`
- `accepted_at`
- `completed_at`
- `submitted_at`
- `expire_at`
- `tracked`
- `version`
- `updated_at`

用途：
- 存玩家任务实例主状态

### 13.2 player_quest_objective

字段建议：

- `player_id`
- `quest_id`
- `objective_id`
- `progress_current`
- `progress_target`
- `status`
- `extra_json`
- `updated_at`

用途：
- 存单目标进度

### 13.3 player_quest_cycle

字段建议：

- `player_id`
- `cycle_type`
- `cycle_key`
- `refreshed_at`

用途：
- 存每日/每周刷新边界

### 13.4 quest_reward_claim_log

字段建议：

- `player_id`
- `quest_id`
- `claim_key`
- `claimed_at`
- `reward_payload`

用途：
- 保证领奖幂等

### 13.5 quest_event_audit

第一版不是必须表。

建议只在调试期或灰度期保留：
- 事件来源
- 目标匹配结果
- 进度变化

便于排查“为什么任务没涨”。

## 14. 协议设计

当前项目 `cmd` 编号按模块分段，建议任务系统使用 `6000-6099`。

### 14.1 消息号建议

- `6001 QUEST_LIST_REQ`
- `6002 QUEST_LIST_RESP`
- `6011 QUEST_UPDATE_PUSH`
- `6012 QUEST_REMOVE_PUSH`
- `6021 QUEST_ACCEPT_REQ`
- `6022 QUEST_ACCEPT_RESP`
- `6031 QUEST_SUBMIT_REQ`
- `6032 QUEST_SUBMIT_RESP`
- `6041 QUEST_TRACK_REQ`
- `6042 QUEST_TRACK_RESP`
- `6051 QUEST_GUIDE_PUSH`
- `6061 QUEST_DAILY_REFRESH_PUSH`

### 14.2 QUEST_LIST_RESP

建议返回：

- 当前可见任务列表
- 当前追踪任务
- 当前服务器时间
- 每日任务下次刷新时间

### 14.3 QUEST_UPDATE_PUSH

建议返回单任务完整快照：

- `quest_id`
- `state`
- `progress`
- `objectives`
- `tracked`
- `toast_text`

单任务完整快照比只推差异字段更简单稳妥，更适合当前项目第一版。

### 14.4 QUEST_ACCEPT_REQ

请求体建议：

- `quest_id`

服务端校验：
- 是否可见
- 是否已解锁
- 是否已接取
- 是否与其他任务互斥

### 14.5 QUEST_SUBMIT_REQ

请求体建议：

- `quest_id`

服务端校验：
- 是否处于 `READY_TO_SUBMIT`
- 是否满足提交条件
- 是否仍满足快照类目标
- 奖励是否可发放

### 14.6 QUEST_TRACK_REQ

请求体建议：

- `quest_id`

作用：
- 切换当前追踪任务
- 客户端地图提示和 HUD 追踪栏据此刷新

## 15. 客户端设计

当前客户端结构适合增量接入，不需要重构主运行态。

### 15.1 GameState 扩展

建议在 `client/autoload/game_state.gd` 新增：

- `signal quest_changed`
- `var quests: Array = []`
- `var tracked_quest_id: int = 0`

并补充方法：
- `set_quests(next_quests: Array, tracked_id: int = 0)`
- `upsert_quest(quest: Dictionary)`
- `remove_quest(quest_id: int)`
- `set_tracked_quest(quest_id: int)`

### 15.2 QuestController

建议新增：

- `client/scripts/feature/quest/quest_controller.gd`

职责：
- 处理 `QUEST_LIST_RESP`
- 处理 `QUEST_UPDATE_PUSH`
- 处理 `QUEST_REMOVE_PUSH`
- 处理 `QUEST_ACCEPT_RESP`
- 处理 `QUEST_SUBMIT_RESP`
- 对外广播 `quests_updated(count)`、`tracked_quest_changed(quest_id)`

### 15.3 App 层请求入口

建议在 `client/autoload/app.gd` 新增：

- `request_quest_list()`
- `request_accept_quest(quest_id: int)`
- `request_submit_quest(quest_id: int)`
- `request_track_quest(quest_id: int)`

### 15.4 Main 场景接入

建议在 `client/scripts/bootstrap/main.gd`：

- 挂载 `QuestController`
- 在 `_register_routes()` 注册 quest 协议
- 在首次进入世界成功后追加 `App.request_quest_list()`
- 在 HUD 中增加任务入口和追踪摘要

### 15.5 HUD 与表现层建议

第一版建议包含：

- 任务入口按钮
- 主线/支线/日常分页
- 单任务追踪栏
- 任务完成 toast
- NPC 头顶 `!` / `?` 提示

第一版先不做：

- 自动寻路
- 复杂路线画线
- 世界地图面板

## 16. 与现有系统的衔接

### 16.1 world 对接点

可直接接入的任务事件：

- 进入地图
- 与 NPC 对话
- 交互传送门
- 触发场景脚本点

当前项目里，`INTERACT_REQ` 很适合作为 NPC 任务接取/提交入口复用。

### 16.2 battle 对接点

可直接接入的任务事件：

- 发起战斗
- 战斗胜利
- 击败指定目标
- 捕捉成功

任务进度应在服务端战斗结算后推进，不应在客户端按钮点击时推进。

### 16.3 pet 对接点

可直接接入的任务事件：

- 宠物获得
- 宠物升级
- 宠物入队

### 16.4 bag 对接点

可直接接入的任务事件：

- 持有指定道具
- 使用指定道具
- 提交指定道具

## 17. 共享进度设计

第一版建议在结构上预留，但只正式支持最简单规则。

### 17.1 share_policy 建议

- `NONE`
- `TEAM_SAME_SCENE`
- `TEAM_SAME_BATTLE`
- `TEAM_NEARBY_RADIUS`

### 17.2 第一版推荐

- 默认全部 `NONE`
- 只对少量日常或协作任务开放 `TEAM_SAME_BATTLE`

### 17.3 共享资格建议

满足全部条件才共享：
- 玩家已接取该任务
- 玩家与贡献者同队
- 玩家在线
- 玩家处于允许共享的地图或战斗
- 玩家不在超距离范围外

## 18. 奖励设计

建议把奖励发放抽成独立能力，而不是只存在于 quest 内部。

### 18.1 奖励类型

- 经验
- 金币
- 道具
- 宠物
- 系统解锁标记
- 活动积分

### 18.2 奖励发放原则

- 奖励发放与任务完成状态切换要放在同一事务语义下
- 奖励必须可幂等
- 奖励日志必须可追踪

### 18.3 奖励失败处理

若发奖失败：
- 不应把任务直接标记为 `COMPLETED`
- 应保留在 `READY_TO_SUBMIT` 或进入可重试状态

## 19. 防作弊与一致性

任务系统需要重点防以下问题：

### 19.1 客户端伪造完成

- 所有完成条件只看服务端事件和服务端当前快照

### 19.2 重复领奖

- 使用 `quest_reward_claim_log` 或等价幂等键防重

### 19.3 道具类任务不同步

- 提交任务前再次读取背包快照验证数量
- “拥有型”与“消耗提交型”要严格区分

### 19.4 日常刷新不一致

- 一律按服务器时间刷新
- 客户端只展示下一次刷新倒计时

### 19.5 断线重连状态错乱

- 重连成功后重新请求任务列表
- push 只做增量优化，不承担唯一真相职责

## 20. 第一版推荐任务内容

结合当前项目已有 world/battle/pet/bag 能力，第一版推荐先做如下主线：

### 20.1 新手主线链路

1. 进入闪光镇东路
2. 与引导 NPC 对话
3. 完成 1 次 NPC 战斗
4. 打开宠物列表
5. 设置 1 只编队宠物
6. 使用 1 次恢复道具
7. 捕捉 1 只宠物

### 20.2 基础日常

- 完成 3 场战斗
- 捕捉 2 只宠物
- 使用 2 次道具
- 登录 1 次

这些目标都能较自然地复用现有双端能力，不需要额外先开发复杂新系统。

## 21. 实施顺序

建议分 5 个阶段逐步落地。

### 阶段 1：服务端任务骨架

- 新增 `quest` 模块
- 增加任务模板加载
- 增加玩家任务实例存储
- 增加 `QUEST_LIST_REQ/RESP`

### 阶段 2：客户端快照与列表 UI

- 扩展 `GameState`
- 增加 `QuestController`
- 增加任务列表入口和追踪栏

### 阶段 3：主线接取与提交

- 接 world 的 NPC 交互链路
- 打通接取/提交协议
- 打通主线第一章

### 阶段 4：战斗/宠物/背包事件推进

- 接 battle、pet、bag 事件
- 完成主线成长链路与基础日常

### 阶段 5：刷新与共享

- 增加日常刷新
- 增加部分共享进度任务
- 增加活动任务起止时间支持

## 22. MVP 范围建议

第一版正式建议只包含：

- 主线任务
- 支线任务
- 日常任务
- 任务列表
- 单任务追踪
- NPC 接取/提交
- 战斗/宠物/背包进度推进
- 奖励发放

第一版先不包含：

- 周常
- 自动寻路
- 大型协作任务
- 多分支剧情树
- 后台可视化编辑器

## 23. 结论

当前项目非常适合把任务系统做成：

- 服务端权威
- 配置驱动
- 事件推进
- 客户端快照展示

这套方案的关键好处有三点：

1. 与当前 world/battle/pet/bag 模块边界一致，不需要推翻现有架构
2. 先能快速支持主线和日常，再平滑扩到活动与共享任务
3. 对联机游戏最关键的防作弊、一致性和断线恢复都有明确落点

如果下一步继续实现，建议先从“协议 + `quest` 模块骨架 + 客户端 `GameState`/`QuestController` 接入”开始，而不是先做复杂 UI。
