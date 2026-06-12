# 任务系统协议草案

## 1. 文档目的

- 本文用于把任务系统协议层固定下来，作为客户端 Godot 和服务端 Go 并行实现的联调依据。
- 本文基于当前项目实时协议约定：
  - WebSocket 二进制包头不变
  - 消息体继续使用 `JSON`
  - `cmd` 按模块分段
- 本文只覆盖任务系统第一版 MVP 需要的协议，不扩展活动编排器、邮件领奖箱、跨服任务等后续能力。

## 2. 设计目标

本协议需要满足以下 6 个目标：

1. 支持任务列表首次全量同步
2. 支持任务增量更新推送
3. 支持接取、提交、追踪三类核心交互
4. 支持任务进度、目标详情和奖励预览展示
5. 支持断线重连后的全量恢复
6. 与当前 `world / battle / pet / bag` 链路风格保持一致

## 3. cmd 编号建议

建议任务系统使用 `6000-6099` 区间。

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

## 4. 统一字段约定

### 4.1 时间字段

- 所有时间统一使用 `unix_ms`
- 字段名统一使用：
  - `server_time_ms`
  - `accepted_at`
  - `completed_at`
  - `submitted_at`
  - `expire_at`
  - `next_refresh_at`

### 4.2 标识字段

- `quest_id`：任务模板唯一标识
- `objective_id`：任务目标唯一标识
- `npc_id`：NPC 标识
- `scene_id`：地图标识
- `item_id`：道具模板标识
- `pet_id`：宠物模板标识
- `pet_uid`：玩家宠物实例唯一标识

### 4.3 状态字段

建议任务状态使用字符串，便于客户端调试和日志阅读。

可选值：
- `LOCKED`
- `AVAILABLE`
- `ACCEPTED`
- `READY_TO_SUBMIT`
- `COMPLETED`
- `EXPIRED`
- `ABANDONED`
- `FAILED`

### 4.4 任务类型字段

建议任务类型也先使用字符串：

- `MAIN`
- `SIDE`
- `DAILY`
- `EVENT`

## 5. 通用数据结构

以下结构不是独立消息号，而是各消息体复用的公共 JSON 结构。

### 5.1 QuestReward

```json
{
  "type": "gold",
  "value": 100,
  "item_id": 0,
  "count": 0,
  "pet_id": 0
}
```

字段说明：
- `type`
  - `gold`
  - `exp`
  - `item`
  - `pet`
  - `feature_unlock`
- `value`
  - 对金币、经验、功能解锁型奖励生效
- `item_id`
  - 道具奖励时使用
- `count`
  - 道具数量
- `pet_id`
  - 宠物奖励时使用

### 5.2 QuestObjectiveState

```json
{
  "objective_id": 1,
  "event_type": "ENTER_SCENE",
  "description": "前往闪光镇东路",
  "current": 1,
  "target": 1,
  "completed": true,
  "target_selector": {
    "scene_id": 2
  },
  "guide": {
    "scene_id": 2,
    "npc_id": 0,
    "text": "前往闪光镇东路"
  }
}
```

字段说明：
- `event_type`：目标推进类型
- `description`：客户端展示文案
- `current` / `target`：进度值
- `completed`：当前目标是否已完成
- `target_selector`：调试和通用展示用
- `guide`：该目标的轻量引导信息

### 5.3 QuestSummary

`QuestSummary` 建议作为任务列表和增量 push 的统一载荷。

```json
{
  "quest_id": 10001,
  "quest_type": "MAIN",
  "state": "ACCEPTED",
  "tracked": true,
  "title": "初入闪光镇",
  "description": "前往闪光镇东路并与引导员交谈。",
  "chapter": 1,
  "sort_order": 10,
  "accept_mode": "AUTO",
  "submit_mode": "NPC",
  "can_abandon": false,
  "npc_id": 90001,
  "scene_id": 2,
  "accepted_at": 1780000000000,
  "completed_at": 0,
  "submitted_at": 0,
  "expire_at": 0,
  "toast_text": "",
  "objectives": [
    {
      "objective_id": 1,
      "event_type": "ENTER_SCENE",
      "description": "前往闪光镇东路",
      "current": 1,
      "target": 1,
      "completed": true,
      "target_selector": {
        "scene_id": 2
      },
      "guide": {
        "scene_id": 2,
        "npc_id": 0,
        "text": "前往闪光镇东路"
      }
    },
    {
      "objective_id": 2,
      "event_type": "TALK_TO_NPC",
      "description": "与引导员交谈",
      "current": 0,
      "target": 1,
      "completed": false,
      "target_selector": {
        "npc_id": 90001
      },
      "guide": {
        "scene_id": 2,
        "npc_id": 90001,
        "text": "找到引导员并交谈"
      }
    }
  ],
  "rewards": [
    {
      "type": "gold",
      "value": 100,
      "item_id": 0,
      "count": 0,
      "pet_id": 0
    },
    {
      "type": "item",
      "value": 0,
      "item_id": 2001,
      "count": 3,
      "pet_id": 0
    }
  ]
}
```

## 6. 列表同步协议

### 6.1 `6001 QUEST_LIST_REQ`

用途：
- 登录进入世界后首次拉取任务列表
- 断线重连后重新同步
- 客户端手动刷新任务列表

请求体：

```json
{
  "include_completed": false
}
```

字段说明：
- `include_completed`
  - 是否把已完成任务也一起返回
  - 第一版客户端默认传 `false`

### 6.2 `6002 QUEST_LIST_RESP`

响应体：

```json
{
  "server_time_ms": 1780000000000,
  "tracked_quest_id": 10001,
  "next_refresh_at": 1780041600000,
  "quests": [
    {
      "quest_id": 10001,
      "quest_type": "MAIN",
      "state": "ACCEPTED",
      "tracked": true,
      "title": "初入闪光镇",
      "description": "前往闪光镇东路并与引导员交谈。",
      "chapter": 1,
      "sort_order": 10,
      "accept_mode": "AUTO",
      "submit_mode": "NPC",
      "can_abandon": false,
      "npc_id": 90001,
      "scene_id": 2,
      "accepted_at": 1780000000000,
      "completed_at": 0,
      "submitted_at": 0,
      "expire_at": 0,
      "toast_text": "",
      "objectives": [],
      "rewards": []
    },
    {
      "quest_id": 20001,
      "quest_type": "DAILY",
      "state": "AVAILABLE",
      "tracked": false,
      "title": "今日训练",
      "description": "完成 3 场战斗。",
      "chapter": 0,
      "sort_order": 100,
      "accept_mode": "AUTO",
      "submit_mode": "AUTO",
      "can_abandon": false,
      "npc_id": 0,
      "scene_id": 0,
      "accepted_at": 0,
      "completed_at": 0,
      "submitted_at": 0,
      "expire_at": 1780041600000,
      "toast_text": "",
      "objectives": [
        {
          "objective_id": 1,
          "event_type": "WIN_BATTLE",
          "description": "完成 3 场战斗",
          "current": 0,
          "target": 3,
          "completed": false,
          "target_selector": {},
          "guide": {
            "scene_id": 0,
            "npc_id": 0,
            "text": "任意战斗获胜均可推进"
          }
        }
      ],
      "rewards": [
        {
          "type": "gold",
          "value": 200,
          "item_id": 0,
          "count": 0,
          "pet_id": 0
        }
      ]
    }
  ]
}
```

字段说明：
- `tracked_quest_id`：当前服务端记录的追踪任务
- `next_refresh_at`：日常/周期任务下一次刷新时间
- `quests`：当前对客户端可见的全部任务快照

## 7. 增量更新推送协议

### 7.1 `6011 QUEST_UPDATE_PUSH`

用途：
- 接取成功后推送
- 任务进度变化后推送
- 任务完成后推送
- 任务奖励领取后推送
- 追踪状态变化后推送

响应体：

```json
{
  "tracked_quest_id": 10001,
  "reason": "progress_updated",
  "quest": {
    "quest_id": 10001,
    "quest_type": "MAIN",
    "state": "READY_TO_SUBMIT",
    "tracked": true,
    "title": "初入闪光镇",
    "description": "前往闪光镇东路并与引导员交谈。",
    "chapter": 1,
    "sort_order": 10,
    "accept_mode": "AUTO",
    "submit_mode": "NPC",
    "can_abandon": false,
    "npc_id": 90001,
    "scene_id": 2,
    "accepted_at": 1780000000000,
    "completed_at": 1780000100000,
    "submitted_at": 0,
    "expire_at": 0,
    "toast_text": "任务已完成，返回引导员处提交。",
    "objectives": [
      {
        "objective_id": 1,
        "event_type": "ENTER_SCENE",
        "description": "前往闪光镇东路",
        "current": 1,
        "target": 1,
        "completed": true,
        "target_selector": {
          "scene_id": 2
        },
        "guide": {
          "scene_id": 2,
          "npc_id": 0,
          "text": "前往闪光镇东路"
        }
      },
      {
        "objective_id": 2,
        "event_type": "TALK_TO_NPC",
        "description": "与引导员交谈",
        "current": 1,
        "target": 1,
        "completed": true,
        "target_selector": {
          "npc_id": 90001
        },
        "guide": {
          "scene_id": 2,
          "npc_id": 90001,
          "text": "找到引导员并交谈"
        }
      }
    ],
    "rewards": [
      {
        "type": "gold",
        "value": 100,
        "item_id": 0,
        "count": 0,
        "pet_id": 0
      }
    ]
  }
}
```

`reason` 建议值：
- `accepted`
- `progress_updated`
- `completed`
- `submitted`
- `tracked_changed`
- `daily_refreshed`

### 7.2 `6012 QUEST_REMOVE_PUSH`

用途：
- 客户端从当前列表中移除某任务
- 常见于任务过期、互斥替换、活动关闭

响应体：

```json
{
  "quest_id": 20001,
  "reason": "expired",
  "tracked_quest_id": 0
}
```

`reason` 建议值：
- `expired`
- `mutually_replaced`
- `hidden_after_complete`
- `event_closed`

## 8. 接取任务协议

### 8.1 `6021 QUEST_ACCEPT_REQ`

用途：
- 玩家主动接取可接任务
- 自动接取任务通常不需要客户端发送该请求

请求体：

```json
{
  "quest_id": 20001
}
```

### 8.2 `6022 QUEST_ACCEPT_RESP`

成功示例：

```json
{
  "accepted": true,
  "reason": "quest submitted",
  "quest": {
    "quest_id": 20001,
    "quest_type": "DAILY",
    "state": "ACCEPTED",
    "tracked": false,
    "title": "今日训练",
    "description": "完成 3 场战斗。",
    "chapter": 0,
    "sort_order": 100,
    "accept_mode": "NPC",
    "submit_mode": "AUTO",
    "can_abandon": false,
    "npc_id": 0,
    "scene_id": 0,
    "accepted_at": 1780000000000,
    "completed_at": 0,
    "submitted_at": 0,
    "expire_at": 1780041600000,
    "toast_text": "已接取任务：今日训练",
    "objectives": [],
    "rewards": []
  }
}
```

失败示例：

```json
{
  "accepted": false,
  "reason": "quest not available"
}
```

失败原因建议：
- `quest not found`
- `quest not visible`
- `quest not available`
- `pre quest not completed`
- `player level not enough`
- `quest already accepted`
- `quest already completed`
- `quest expired`

说明：
- 成功时返回完整 `quest` 快照，方便客户端立即写入本地状态
- 同时服务端仍可补发 `QUEST_UPDATE_PUSH`，但客户端不能依赖一定会补发

## 9. 提交任务协议

### 9.1 `6031 QUEST_SUBMIT_REQ`

请求体：

```json
{
  "quest_id": 10001
}
```

### 9.2 `6032 QUEST_SUBMIT_RESP`

成功示例：

```json
{
  "accepted": true,
  "reason": "quest submitted",
  "quest": {
    "quest_id": 10001,
    "quest_type": "MAIN",
    "state": "COMPLETED",
    "tracked": false,
    "title": "初入闪光镇",
    "description": "前往闪光镇东路并与引导员交谈。",
    "chapter": 1,
    "sort_order": 10,
    "accept_mode": "AUTO",
    "submit_mode": "NPC",
    "can_abandon": false,
    "npc_id": 90001,
    "scene_id": 2,
    "accepted_at": 1780000000000,
    "completed_at": 1780000100000,
    "submitted_at": 1780000200000,
    "expire_at": 0,
    "toast_text": "任务完成：获得 100 金币",
    "objectives": []
  },
  "rewards": [
    {
      "type": "gold",
      "value": 100,
      "item_id": 0,
      "count": 0,
      "pet_id": 0
    },
    {
      "type": "item",
      "value": 0,
      "item_id": 2001,
      "count": 2,
      "pet_id": 0
    },
    {
      "type": "pet",
      "value": 0,
      "item_id": 0,
      "count": 0,
      "pet_id": 102
    }
  ]
}
```

失败示例：

```json
{
  "accepted": false,
  "reason": "quest not ready to submit"
}
```

说明：

- 当前运行时已正式接入 `gold` / `exp` / `item` / `pet` 四类任务奖励
- `feature_unlock` 仍保留在任务模板配置中，后续由对应后端模块接入正式发放
- `rewards` 字段表示本次已经实际走过服务端发奖链路的奖励
- 如果本次奖励包含金币，服务端会继续推送 `5091 WALLET_UPDATE_PUSH`
- 如果本次奖励包含道具，服务端会继续推送 `5011 BAG_UPDATE_PUSH`
- 如果本次奖励包含宠物，服务端会继续推送 `3011 PET_UPDATE_PUSH`
- 如果本次任务奖励包含金币，服务端会继续推送 `5091 WALLET_UPDATE_PUSH`
- 如果本次任务奖励包含道具，服务端会继续推送 `5011 BAG_UPDATE_PUSH`

失败原因建议：
- `quest not found`
- `quest not accepted`
- `quest not ready to submit`
- `quest expired`
- `required item not enough`
- `reward grant failed`

说明：
- `granted_rewards` 供客户端直接弹获得提示
- `next_quest_ids` 供客户端做后续文案或轻引导

## 10. 追踪任务协议

### 10.1 `6041 QUEST_TRACK_REQ`

请求体：

```json
{
  "quest_id": 10001
}
```

取消追踪示例：

```json
{
  "quest_id": 0
}
```

### 10.2 `6042 QUEST_TRACK_RESP`

响应体：

```json
{
  "accepted": true,
  "reason": "",
  "tracked_quest_id": 10001
}
```

失败示例：

```json
{
  "accepted": false,
  "reason": "quest not visible",
  "tracked_quest_id": 0
}
```

说明：
- 追踪状态变化后，服务端也可补发 `QUEST_UPDATE_PUSH`
- 客户端本地不应只靠按钮状态自行切换最终追踪结果

## 11. 引导推送协议

### 11.1 `6051 QUEST_GUIDE_PUSH`

用途：
- 服务端主动提示客户端切换当前任务引导目标
- 适合以下情况：
  - 自动接取新主线
  - 任务目标推进到下一步
  - 任务已完成需要返回 NPC 提交

响应体：

```json
{
  "quest_id": 10001,
  "objective_id": 2,
  "scene_id": 2,
  "npc_id": 90001,
  "entity_id": 0,
  "text": "返回引导员处提交任务。"
}
```

说明：
- 客户端可以用这个推送刷新右侧追踪栏、高亮 NPC 头顶标识或顶部提示文案
- 第一版不要求自动寻路

## 12. 周期刷新推送协议

### 12.1 `6061 QUEST_DAILY_REFRESH_PUSH`

用途：
- 日常任务刷新后通知客户端整体刷新状态

响应体：

```json
{
  "server_time_ms": 1780041600000,
  "next_refresh_at": 1780128000000,
  "added_quest_ids": [
    20001,
    20002
  ],
  "removed_quest_ids": [
    20000
  ],
  "tracked_quest_id": 0,
  "toast_text": "今日任务已刷新"
}
```

说明：
- 收到此推送后，客户端推荐立即再次调用 `QUEST_LIST_REQ`
- 这样比仅依赖 `added_quest_ids` 和 `removed_quest_ids` 更稳妥

## 13. 客户端处理建议

### 13.1 首次同步

登录并进入世界后：

1. `ENTER_WORLD_RESP`
2. `PET_LIST_REQ`
3. `BAG_LIST_REQ`
4. `QUEST_LIST_REQ`

### 13.2 本地状态结构建议

建议在 `GameState` 中新增：

```text
quests: Array
tracked_quest_id: int
next_quest_refresh_at: int
```

### 13.3 写入策略建议

- `QUEST_LIST_RESP`
  - 全量替换 `quests`
- `QUEST_UPDATE_PUSH`
  - 按 `quest_id` 覆盖或插入
- `QUEST_REMOVE_PUSH`
  - 按 `quest_id` 删除
- `QUEST_TRACK_RESP`
  - 更新 `tracked_quest_id`

## 14. 服务端实现建议

### 14.1 最小服务接口

建议 `quest` 服务至少提供：

- `ListVisibleQuests(playerID)`
- `AcceptQuest(playerID, questID)`
- `SubmitQuest(playerID, questID)`
- `SetTrackedQuest(playerID, questID)`
- `RefreshDailyQuests(playerID)`
- `HandleEvent(event)`

### 14.2 推送策略

第一版建议：

- 关键请求响应时直接带完整任务快照
- 同时在必要时补发 `QUEST_UPDATE_PUSH`
- 断线重连后一律依赖 `QUEST_LIST_REQ` 全量恢复

## 15. 错误处理建议

任务协议错误不建议过度细分业务码的第一版客户端表现。

第一版客户端只需做到：
- 收到 `accepted = false`
- 展示 `reason`
- 不自行猜测状态变化

后续如果需要，再把 `reason` 收敛到错误码表。

## 16. 第一版联调建议

推荐按下面顺序联调：

1. `QUEST_LIST_REQ / RESP`
2. `QUEST_ACCEPT_REQ / RESP`
3. `QUEST_UPDATE_PUSH`
4. `QUEST_SUBMIT_REQ / RESP`
5. `QUEST_TRACK_REQ / RESP`
6. `QUEST_DAILY_REFRESH_PUSH`

这样可以先把“查、接、提、追踪”主闭环跑通，再补刷新与引导。

## 17. 结论

这份协议草案的核心口径是：

- 列表全量同步
- 单任务完整快照增量更新
- 接取/提交/追踪三条核心交互链路
- 引导与刷新作为辅助推送

对于当前项目，这是最稳妥也最容易落代码的方案：

- 客户端实现简单
- 服务端幂等和一致性更容易保证
- 出现断线或异常时，也能通过 `QUEST_LIST_REQ` 快速恢复状态
