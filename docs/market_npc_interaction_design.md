# 市场 NPC 交互设计方案

## 目标

先为市场场景里的 NPC 落一套适合联机项目的交互方案，满足下面几点：

- NPC 可以放到独立 `npc` 碰撞层。
- 玩家靠近 NPC 时可以触发交互。
- 交互结果仍然由服务端权威决定。
- 后续可以扩展到对话、商店、任务、战斗等多种 NPC。
- 不把交互逻辑写死在单个市场 NPC 场景里，尽量可复用。

---

## 设计结论

不建议只靠 `CharacterBody2D` 的物理碰撞直接做业务交互。

推荐拆成两层：

1. **物理阻挡层**
   - 负责玩家能不能穿过 NPC。
   - 由 NPC 本体碰撞承担。

2. **交互检测层**
   - 负责判断玩家是否进入交互范围。
   - 由 NPC 的 `InteractArea2D` 承担。

这样做的好处：

- 交互和碰撞分离，后续更容易扩展。
- 可以支持“阻挡但不自动交互”。
- 可以支持“可交互但不阻挡”。
- 更适合联机项目把最终交互结果交给服务端校验。

---

## 推荐节点结构

每个市场 NPC 建议统一为下面的结构：

```text
NPCRoot (Node2D)
├── Sprite2D
├── CollisionBody / StaticBody2D 或 CharacterBody2D
│   └── CollisionShape2D
├── InteractArea (Area2D)
│   └── CollisionShape2D
└── npc_xxx.gd
```

也可以简化为：

```text
NPCRoot (Area2D/Node2D)
├── Sprite2D
├── BodyBlocker (StaticBody2D)
│   └── CollisionShape2D
├── InteractArea (Area2D)
│   └── CollisionShape2D
└── Script
```

### 说明

- `CollisionBody`
  - 负责阻挡玩家。
  - 挂在 `npc` 层。
- `InteractArea`
  - 负责检测玩家进入交互范围。
  - 不负责阻挡。
  - 可以比实际碰撞体稍大一些，提升手感。

---

## 碰撞层建议

建议统一约定几类 2D 物理层：

- `world_block`
  - 地图阻挡。
- `player`
  - 玩家本体。
- `npc`
  - NPC 物理本体。
- `interact`
  - NPC / 机关 / 采集点的交互区域。

### 推荐关系

#### 玩家本体

- `collision_layer = player`
- `collision_mask = world_block | npc | interact`

#### NPC 本体

- `collision_layer = npc`
- `collision_mask = player`

#### NPC 交互区 `InteractArea2D`

- `collision_layer = interact`
- `collision_mask = player`

---

## 联机交互主流程

推荐流程：

```text
玩家进入 NPC 交互范围
-> 客户端记录当前可交互 NPC
-> 玩家主动按交互键 / 点击按钮
-> 客户端发送 INTERACT_REQ(entity_id)
-> 服务端校验距离、场景、状态、任务条件
-> 服务端返回 INTERACT_RESP 或后续事件
-> 客户端打开对话 / 商店 / 战斗 / 任务 UI
```

### 为什么要服务端权威

因为当前项目是联机结构，客户端不应该直接决定：

- 这个 NPC 是否可交互
- 当前玩家是否满足条件
- 交互结果是什么

客户端只负责：

- 检测可交互候选目标
- 发起请求
- 展示服务端结果

---

## 不推荐的方案

### 方案：玩家一撞到 NPC 就直接触发业务交互

不推荐原因：

- 容易误触。
- 难扩展成“靠近后按键交互”。
- 后续任务条件、商店、对话分支会变复杂。
- 玩家和 NPC 的阻挡逻辑会和交互逻辑搅在一起。

如果确实需要“接触即交互”，也建议：

- 仍然由 `InteractArea2D` 发信号。
- 不要直接依赖物理碰撞回调承担全部业务语义。

---

## 推荐的脚本职责拆分

### 1. NPC 场景脚本

职责：

- 暴露 `entity_id`。
- 监听 `InteractArea2D`。
- 向上层发出“可交互 / 请求交互”信号。

建议字段：

```gdscript
@export var entity_id: int = 0
@export var npc_kind: String = "dialog"
@export var blocks_player: bool = true
@export var auto_interact: bool = false
```

建议信号：

```gdscript
signal interaction_entered(entity_id: int)
signal interaction_exited(entity_id: int)
signal interaction_requested(entity_id: int)
```

### 2. World Controller

职责：

- 管理“当前可交互 NPC”。
- 监听输入。
- 在适当时机调用 `App.request_interact(entity_id)`。

### 3. 服务端

职责：

- 校验玩家和 NPC 是否真的处于可交互状态。
- 返回具体交互结果。

---

## 市场 NPC 的推荐业务分类

建议先抽象成统一配置：

- `dialog`
- `shop`
- `quest`
- `battle`

即使当前先只做一个市场 NPC，也建议字段先留出来：

```gdscript
@export var npc_kind: String = "dialog"
```

这样后面扩展市场里的：

- 商店老板
- 任务发布者
- 对战 NPC
- 剧情 NPC

就不需要重构整体交互架构。

---

## 当前项目中的建议落地顺序

### 第一步：先让市场 NPC 具备交互能力

目标：

- 玩家走近市场 NPC。
- 客户端能识别“当前可交互 NPC”。
- 玩家触发交互时走现有 `App.request_interact(entity_id)`。

### 第二步：给市场 NPC 区分阻挡和交互半径

目标：

- 阻挡体小一点，避免卡住手感。
- 交互体略大一点，方便靠近交互。

### 第三步：统一可复用基类

目标：

- 抽一个 `interactive_npc_base.gd`。
- 市场、学校、野外 NPC 都走同一套交互框架。

---

## 对当前项目的推荐实现方案

### 推荐方案 A：靠近后按键交互

这是最推荐的落地方案。

流程：

- 玩家进入 NPC `InteractArea2D`。
- world controller 记录 `current_interactable_entity_id`。
- HUD 或输入系统显示“可交互”。
- 玩家按键后发 `App.request_interact(entity_id)`。

优点：

- 最稳。
- 最像正式游戏。
- 最适合联机。

### 备选方案 B：进入范围自动发起交互

适合：

- 靠近就对话
- 靠近就自动进事件

缺点：

- 误触概率高
- 市场里体验一般

因此不作为首选。

---

## 本文结论

对于“市场 NPC 设置为 npc 层，玩家碰撞这个 NPC 发生交互”这个需求，推荐最终设计为：

- **NPC 本体进 `npc` 层，负责阻挡。**
- **NPC 再挂一个 `InteractArea2D`，负责交互检测。**
- **客户端只负责发现交互目标并发起请求。**
- **服务端负责最终裁定交互结果。**

这是当前联机 Godot 项目里最成熟、最稳、最好扩展的方案。

---

## 下一步落地建议

建议下一步直接做：

1. 抽一个通用 `interactive_npc_base.gd`
2. 给 `radiant_market.tscn` 里的两个 NPC 接上 `InteractArea2D`
3. world controller 增加“当前可交互 NPC”管理
4. 先复用现有 `App.request_interact(entity_id)` 跑通一条完整链路

这样可以先把市场 NPC 交互落地，再决定是否需要 UI 提示和按键交互面板。


---

## NPC 显示内容的决定方式

对于“碰撞后显示任务列表、交互列表，或者任务和交互混合列表”这个需求，推荐统一成一套方案：

- **NPC 不直接决定最终显示什么内容。**
- **服务端根据当前玩家状态返回一份 NPC 可用操作列表。**
- **客户端只负责把这份列表渲染成统一菜单。**

这样最适合联机项目，也最方便后续扩展。

### 为什么不建议客户端写死内容

如果直接在 NPC 本地脚本里写死：

- 这个 NPC 只显示任务
- 这个 NPC 只显示商店
- 这个 NPC 总是显示固定对话

后面就很快会遇到问题：

- 不同玩家因为任务进度不同，看到的内容应该不同。
- 同一个 NPC 在不同时间段、活动阶段可能显示不同入口。
- 有些选项需要先完成前置任务才显示。
- 有些选项虽然显示，但当前应为灰态不可用。

这些都更适合交给服务端动态裁定。

---

## 统一抽象：NPC 可用操作列表

推荐把任务、商店、对话、战斗等内容统一抽象成：

- **NPC 菜单项列表**

不再区分“任务页”和“交互页”两套完全不同的机制，而是统一由服务端返回一组菜单项。

### 统一数据模型示例

```json
{
  "entity_id": 10001,
  "npc_name": "市场管理员",
  "menu_style": "list",
  "entries": [
    {
      "entry_id": "quest_accept_1001",
      "entry_type": "quest",
      "title": "领取日常任务",
      "subtitle": "帮助整理市场货物",
      "state": "available",
      "priority": 100
    },
    {
      "entry_id": "shop_open",
      "entry_type": "shop",
      "title": "打开商店",
      "subtitle": "购买基础补给",
      "state": "available",
      "priority": 80
    },
    {
      "entry_id": "dialog_gossip",
      "entry_type": "dialog",
      "title": "随便聊聊",
      "subtitle": "打听市场消息",
      "state": "available",
      "priority": 60
    }
  ]
}
```

这样三种场景都能统一覆盖：

1. **只显示任务列表**
   - `entries` 全是 `quest`
2. **只显示交互列表**
   - `entries` 全是 `dialog / shop / battle / teleport`
3. **任务和交互混合显示**
   - `entries` 混排即可

---

## 推荐的协议与交互流程

推荐把 NPC 交互拆成两段：

### 第一步：请求 NPC 菜单

客户端：

- 玩家进入交互范围后按键，或者自动触发交互。
- 客户端发送 `NPC_MENU_REQ(entity_id)`。

服务端：

- 校验玩家是否真的可以和该 NPC 交互。
- 根据玩家状态、任务进度、NPC 类型生成菜单。
- 返回 `NPC_MENU_RESP`。

### 第二步：执行某个菜单项

客户端：

- 玩家点击某个菜单项。
- 客户端发送 `NPC_ACTION_REQ(entity_id, entry_id)`。

服务端：

- 执行对应逻辑。
- 返回：
  - 对话内容
  - 任务列表/任务详情
  - 商店数据
  - 战斗开始推送
  - 拒绝原因

---

## 客户端与服务端职责边界

### NPC 场景本地脚本负责

- 暴露 `entity_id`
- 检测玩家进入交互范围
- 告知 world controller “这个 NPC 可交互”

### World Controller 负责

- 管理当前可交互 NPC
- 监听玩家输入
- 触发 `App.request_interact(entity_id)` 或后续的菜单请求
- 接收服务端返回的数据并打开统一菜单面板

### 服务端负责

- 决定当前玩家能看到哪些菜单项
- 决定这些菜单项是否可点击
- 决定点击后发生什么结果

---

## 推荐支持的菜单项类型

建议先统一支持以下类型：

- `quest`
- `dialog`
- `shop`
- `battle`
- `teleport`
- `craft`
- `event`

### 推荐支持的状态字段

- `available`
- `locked`
- `completed`
- `in_progress`

这样客户端后面可以做：

- 灰态不可点
- 已完成标记
- 正在进行中的任务强调显示
- 按优先级排序

---

## 推荐的客户端 UI 方案

建议不要为任务、商店、对话分别做完全独立的第一层入口 UI。

推荐统一做一个：

- `NpcMenuPanel`

### 面板结构建议

```text
NpcMenuPanel
├── NpcName
├── OptionalPortrait
├── EntryList
│   ├── EntryButton
│   ├── EntryButton
│   └── ...
└── CloseButton
```

### 每个菜单项建议显示

- 类型图标
- 标题
- 副标题
- 状态标签

例如：

```text
[任务] 领取日常任务
帮助整理市场货物

[商店] 打开商店
购买基础补给

[对话] 随便聊聊
打听市场消息
```

这样任务和普通交互天然可以混排，不需要拆成两套面板。

---

## 推荐的触发策略

推荐区分两件事：

1. **什么时候弹出菜单**
2. **菜单里显示什么**

### 什么时候弹出菜单

可以由客户端/NPC 配置决定：

```gdscript
@export var interact_trigger_mode: String = "press"
# press / auto
```

- `press`
  - 玩家进入交互范围后，按键才打开菜单
- `auto`
  - 玩家进入范围后自动请求菜单

### 菜单里显示什么

统一由服务端返回。

---

## 对当前项目的可落地方案

结合当前项目现状，推荐按下面顺序逐步落地。

### 方案阶段 1：先复用现有交互主链路

当前项目已经有：

- `App.request_interact(entity_id)`
- `INTERACT_RESP`

因此第一阶段可以先不急着扩完整的新协议，而是：

1. 玩家靠近市场 NPC
2. 客户端记录可交互 `entity_id`
3. 玩家按键后调用现有 `App.request_interact(entity_id)`
4. 服务端先固定返回一份混合菜单数据或临时交互结果
5. 客户端弹出统一列表面板

这一步的目标是：

- **先跑通 UI + 客户端世界交互链路 + 服务端返回菜单的整体流程**

### 方案阶段 2：补标准化菜单协议

第二阶段再抽象成更正式的协议：

- `NPC_MENU_REQ`
- `NPC_MENU_RESP`
- `NPC_ACTION_REQ`
- `NPC_ACTION_RESP` 或后续推送

这一步的目标是：

- 把“请求菜单”和“执行菜单项”彻底拆开
- 让商店、任务、对话、战斗都能走统一机制

---

## 推荐的 MVP 落地顺序

建议按下面顺序推进：

1. 抽一个通用 `interactive_npc_base.gd`
2. 给市场 NPC 接上 `InteractArea2D`
3. world controller 增加“当前可交互 NPC”管理
4. 做一个统一 `NpcMenuPanel`
5. 第一阶段先复用 `App.request_interact(entity_id)` 跑通菜单弹出
6. 第二阶段再把服务端菜单协议标准化

---

## 本文补充后的最终结论

对于市场 NPC 的显示内容，推荐最终采用：

- **统一菜单列表模型**
- **服务端动态决定菜单项内容**
- **客户端统一渲染列表 UI**

也就是说，不再单独设计：

- 任务列表页
- 交互列表页
- 混合列表页

而是把它们统一成：

- **NPC 可用操作列表**

这是当前项目里最合理、最可扩展、最适合联机的可落地方案。
