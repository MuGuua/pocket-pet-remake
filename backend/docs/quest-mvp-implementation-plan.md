# 任务系统 MVP 落地方案

## 1. 文档目的

- 把现有 `backend/docs/quest-system.md` 和 `backend/docs/quest-protocol.md` 再往前推进一步
- 给当前仓库补一份“按现有代码结构怎么开始做”的实施蓝图
- 让客户端 Godot 与服务端 Go 可以按同一批文件边界并行开发

本文默认遵循两份既有设计：
- `backend/docs/quest-system.md`
- `backend/docs/quest-protocol.md`

## 2. 当前项目的任务系统切入点

### 2.1 服务端已经具备的前置能力

当前服务端已经有：
- 会话与鉴权
- 世界进入与切图
- NPC 交互
- 服务端权威 PvE 战斗
- 宠物列表与编队
- 背包列表与道具使用

这意味着任务系统第一版不需要“先造事件源”，而是可以直接复用已有业务动作。

### 2.2 客户端已经具备的前置能力

当前客户端已经有：
- 全局快照容器 `GameState`
- 消息号常量 `CommandIds`
- 网络消息分发 `MessageRouter`
- 登录后主运行态 UI `RuntimeHud`
- 世界/NPC 交互入口

所以第一版任务系统也不需要先大改前端框架，而是在现有入口旁边补任务 UI 与任务推送消费。

## 3. 第一版业务目标

建议把第一版范围收敛为 4 组任务能力。

### 3.1 新手主线链

建议直接围绕当前已有地图和玩法做 3 段任务：

1. `Q1001 初入闪光镇`
   - 自动接取
   - 目标：进入 `scene_id = 2`
2. `Q1002 向市场管理员报到`
   - 前置：`Q1001`
   - 目标：与指定 NPC 对话
3. `Q1003 完成第一次对战`
   - 前置：`Q1002`
   - 目标：击败指定 NPC 或完成 1 场 PvE

### 3.2 功能解锁任务

建议把功能开关与主线绑定：

- 完成 `Q1002` 后解锁任务面板追踪
- 完成 `Q1003` 后解锁背包操作或编队编辑

如果当前功能已经开放，也可以先只做“引导性文案任务”，奖励写成金币/道具，功能解锁字段先保留不用。

### 3.3 基础支线

围绕现有地图与 NPC 再补 1~2 个可选支线：

- 与某个 NPC 对话
- 进入指定地图
- 完成 2 场战斗

### 3.4 基础日常

第一版只做最简单的一条：
- 每日完成 3 场战斗

这样就能把 `周期刷新` 的骨架也一起验证掉。

## 4. 服务端建议目录

建议新增独立 `quest` 模块，目录大致如下：

```text
backend/server/internal/module/quest/
  service.go
  types.go
  template_loader.go
  matcher.go
  progress.go
  reward.go
  repo.go
```

推荐职责：
- `types.go`
  - 任务模板、目标模板、任务实例、奖励定义、领域事件定义
- `template_loader.go`
  - 从内置 JSON 或配置资源加载任务模板
- `matcher.go`
  - 把领域事件与任务目标做匹配
- `progress.go`
  - 进度累加、状态迁移、自动提交判断
- `reward.go`
  - 金币/道具/宠物/功能解锁奖励发放
- `repo.go`
  - 任务实例读写接口
- `service.go`
  - 对外统一入口，供 handler 和其他模块调用

第一版建议先走内存仓储，接口留好后再落 PostgreSQL。

## 5. 服务端数据模型建议

### 5.1 任务模板 `QuestTemplate`

建议字段：

```go
QuestTemplate {
  QuestID       int64
  QuestType     string   // MAIN/SIDE/DAILY/EVENT
  Title         string
  Description   string
  Chapter       int
  SortOrder     int
  AcceptMode    string   // AUTO/NPC/MANUAL
  SubmitMode    string   // AUTO/NPC/MANUAL
  AutoTrack     bool
  CanAbandon    bool
  StartNPCID    int64
  SubmitNPCID   int64
  PreQuestIDs   []int64
  UnlockRules   []Rule
  Objectives    []ObjectiveTemplate
  Rewards       []Reward
  TimeWindow    TimeWindow
}
```

### 5.2 任务目标模板 `ObjectiveTemplate`

```go
ObjectiveTemplate {
  ObjectiveID    int64
  EventType      string   // ENTER_SCENE/TALK_TO_NPC/WIN_BATTLE/USE_ITEM/OBTAIN_PET
  Description    string
  Target         int
  Selector       map[string]any
  AccumulateMode string   // SET/MAX/ADD
  Guide          GuideInfo
}
```

### 5.3 玩家任务实例 `QuestInstance`

```go
QuestInstance {
  PlayerID       int64
  QuestID        int64
  State          string
  Tracked        bool
  AcceptedAt     int64
  CompletedAt    int64
  SubmittedAt    int64
  ExpireAt       int64
  ObjectiveState []ObjectiveProgress
  RewardVersion  int
}
```

### 5.4 目标进度 `ObjectiveProgress`

```go
ObjectiveProgress {
  ObjectiveID int64
  Current     int
  Target      int
  Completed   bool
}
```

第一版即便只做简单任务，也建议一开始就把“任务状态”和“目标状态”分开，否则很快会卡在 `2/3`、`7/10` 这种需求上。

## 6. 领域事件模型

任务模块不要反向依赖其他模块内部细节，统一消费标准事件。

### 6.1 事件结构

```go
QuestEvent {
  PlayerID   int64
  EventType  string
  SceneID    int64
  NPCID      int64
  BattleID   int64
  ItemID     int64
  PetID      int64
  Count      int
  Meta       map[string]any
  OccurredAt int64
}
```

### 6.2 第一版需要的事件类型

- `ENTER_SCENE`
- `TALK_TO_NPC`
- `WIN_BATTLE`
- `FINISH_BATTLE`
- `USE_ITEM`
- `OBTAIN_ITEM`
- `OBTAIN_PET`
- `SET_LINEUP`

### 6.3 事件来源建议

- `world` 模块
  - 进入场景成功后发 `ENTER_SCENE`
  - NPC 菜单交互成功后发 `TALK_TO_NPC`
- `battle` 模块
  - 结算成功后发 `WIN_BATTLE` 或 `FINISH_BATTLE`
- `bag` 模块
  - 使用道具成功后发 `USE_ITEM`
  - 获得关键掉落后发 `OBTAIN_ITEM`
- `pet` 模块
  - 获得宠物后发 `OBTAIN_PET`
  - 编队提交成功后发 `SET_LINEUP`

## 7. 服务端状态流转建议

第一版推荐统一沿用这条主路径：

```text
LOCKED -> AVAILABLE -> ACCEPTED -> READY_TO_SUBMIT -> COMPLETED
```

细化规则：
- 自动接取任务：`AVAILABLE -> ACCEPTED` 立即发生
- 自动提交任务：完成时直接 `ACCEPTED -> COMPLETED`
- NPC 提交任务：完成时先到 `READY_TO_SUBMIT`
- 日常刷新任务：重置时删除旧实例或重置状态为 `AVAILABLE`

## 8. 奖励发放建议

第一版只实现下面 4 类奖励即可：

- `gold`
- `item`
- `pet`
- `feature_unlock`

其中：
- `gold`、`item`、`pet` 可以直接落到现有经济/背包/宠物链路
- `feature_unlock` 第一版可以先只写进玩家状态字段，客户端收到世界/任务快照后决定是否展示入口

为避免重复领奖，奖励发放建议遵守：
- 只有 `READY_TO_SUBMIT -> COMPLETED` 或自动完成瞬间可以发奖
- 状态更新和奖励发放必须放在同一事务语义里
- 对重复提交请求直接返回当前快照，不重复发奖

## 9. 协议接入建议

命令号建议沿用已有草案的 `6000-6099` 区间。

### 9.1 先实现的消息

- `6001 QUEST_LIST_REQ`
- `6002 QUEST_LIST_RESP`
- `6011 QUEST_UPDATE_PUSH`
- `6021 QUEST_ACCEPT_REQ`
- `6022 QUEST_ACCEPT_RESP`
- `6031 QUEST_SUBMIT_REQ`
- `6032 QUEST_SUBMIT_RESP`
- `6041 QUEST_TRACK_REQ`
- `6042 QUEST_TRACK_RESP`

### 9.2 第一版可以先不做的消息

- `6012 QUEST_REMOVE_PUSH`
- `6051 QUEST_GUIDE_PUSH`
- `6061 QUEST_DAILY_REFRESH_PUSH`

这些都可以在主链跑通后补。

## 10. 服务端文件修改建议

建议按下面顺序接入，尽量减少扩散：

### 10.1 协议层

- `backend/server/internal/protocol/command.go`
  - 增加 quest 消息号常量
- `backend/server/internal/protocol/messages.go`
  - 增加 quest 请求/响应结构

### 10.2 路由层

- `backend/server/internal/transport/ws/router.go`
  - 注册 quest handler
- `backend/server/internal/transport/ws/quest_handler.go`
  - 处理列表、接取、提交、追踪请求

### 10.3 模块层

- 新增 `backend/server/internal/module/quest/*`
- 在 `world`、`battle`、`bag`、`pet` 成功动作后投递 `QuestEvent`

这里建议采用“事件投递函数”而不是直接让其他模块操作 quest 仓储，例如：

```go
questService.HandleEvent(ctx, quest.QuestEvent{...})
```

## 11. 客户端数据结构建议

### 11.1 `GameState` 新增字段

建议在 `client/autoload/game_state.gd` 增加：

```gdscript
signal quests_changed

var quests: Array = []
var tracked_quest_id: int = 0
```

以及最少 4 个方法：
- `set_quests(next_quests: Array) -> void`
- `upsert_quest(quest: Dictionary) -> void`
- `remove_quest(quest_id: int) -> void`
- `set_tracked_quest(quest_id: int) -> void`

客户端继续保持“只存快照，不做权威判定”。

### 11.2 `CommandIds` 新增消息号

在 `client/scripts/common/command_ids.gd` 同步增加 quest 常量，保持双端一致。

### 11.3 `MessageRouter` 与控制器

建议新增：

- `client/scripts/feature/quest/quest_controller.gd`

职责：
- 消费 `QUEST_LIST_RESP`
- 消费 `QUEST_UPDATE_PUSH`
- 消费 `QUEST_ACCEPT_RESP`
- 消费 `QUEST_SUBMIT_RESP`
- 把结果写回 `GameState`

## 12. 客户端 UI 接入建议

### 12.1 `RuntimeHud`

建议在 `client/scripts/bootstrap/runtime_hud.gd` 增加一个 `任务` 入口按钮和一个轻量追踪区：

- 任务按钮：打开任务列表
- 追踪区：显示当前追踪任务的标题和首个未完成目标
- 已完成待提交时：高亮显示“去找谁交”

第一版不要做复杂卷轴面板，沿用当前卡片式数据面板即可。

### 12.2 NPC 头顶标识

建议在现有 NPC 交互脚本上补充 3 态提示：

- `!`：有可接任务
- `?`：有可提交任务
- 空：无任务

即便第一版先不做真正图标，也可以先通过名字颜色、顶部文本或菜单文案验证整条链路。

### 12.3 地图内追踪引导

第一版只做轻量引导：
- 在任务追踪区显示“前往某地图/与某 NPC 对话”
- 不做自动寻路
- 不做地面指引箭头

## 13. MVP 实施顺序

建议按 5 个小阶段推进。

### 阶段 1：只做任务快照读写

目标：
- 模板加载
- 玩家任务实例结构
- `QUEST_LIST_REQ/RESP`
- 客户端任务列表展示

不接入任何自动推进，先把数据流打通。

### 阶段 2：接入世界事件

目标：
- `ENTER_SCENE`
- `TALK_TO_NPC`
- 新手主线 `Q1001/Q1002`

这样可以先验证：
- 解锁
- 自动接取
- NPC 提交
- 任务追踪

### 阶段 3：接入战斗事件

目标：
- `WIN_BATTLE`
- 新手主线 `Q1003`
- 首次奖励发放

这样可以验证服务端权威完成判定与发奖。

### 阶段 4：接入日常任务

目标：
- 每日 3 战
- 每日刷新逻辑

这样可以验证周期任务模型是否足够。

### 阶段 5：补 UI 细节

目标：
- NPC 可接/可提交提示
- HUD 追踪态优化
- 任务完成提示文案

## 14. 最容易踩坑的点

### 14.1 用客户端事件直接改任务状态

不能让客户端在本地把任务改成完成，只能展示服务端下发的结果。

### 14.2 把任务规则写死在 handler 里

`quest_handler.go` 应只做协议收发，不要在里面写解锁/完成逻辑。

### 14.3 只记录任务状态，不记录目标进度

这样很快就无法支持累计型任务，也不利于 HUD 展示进度。

### 14.4 让 world/battle/bag/pet 反向依赖 quest 内部类型过多

最好只依赖一个轻量 `HandleEvent()` 入口，保持模块解耦。

### 14.5 奖励与状态更新分两次提交

这会在断线、重试或并发请求下制造重复领奖漏洞。

## 15. 结论

对于当前 `pocket-pet-remake`，任务系统并不需要从零重新发明。

最稳妥的路径是：
- 用 `backend/docs/quest-system.md` 作为原则文档
- 用 `backend/docs/quest-protocol.md` 作为协议文档
- 用本文档作为“本仓库从哪里开始改”的执行蓝图

如果严格按这个 MVP 顺序落地，第一版就可以较快拿到一条完整闭环：
- 登录进入世界
- 自动接到主线
- 进地图/找 NPC/打一场战斗
- 服务端完成判定与发奖
- 客户端 HUD 和 NPC 提示同步刷新
