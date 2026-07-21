# 任务系统协议草案

## 任务领取与交付动画

- `QUEST_ACCEPT_RESP.client_animation_key`：领取成功后客户端播放的剧情注册键。
- `QUEST_SUBMIT_RESP.client_animation_key`：交付成功并完成奖励持久化后客户端播放的剧情注册键。
- NPC 菜单领取和交付通过 `NPC_ACTION_RESP.client_animation_key` 返回同一配置。
- 空字符串表示不播放；客户端找不到对应场景时直接继续任务刷新或奖励弹窗，不回滚服务端状态。
- 交付展示顺序固定为：服务端结算成功、客户端播放动画、动画结束、升级及奖励弹窗。

## 对话驱动的任务流转

- NPC 对话节点 `effects_json.accept_quest_id` 在玩家选择“接受”并进入后续节点时接取任务。
- NPC 对话节点 `effects_json.quest_event` 先推进任务目标，`effects_json.submit_quest_id` 随后校验并交付任务、持久化奖励。
- 对话副作用失败时返回失败响应，不允许客户端继续显示“任务完成”。
- `SCENE_TRIGGER_PUSH.prompt_text` 用于一次性场景提示；有剧情动画时在动画结束后展示，无动画时进入场景立即展示。

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

任务模板的 `accept_conditions` 使用结构化 JSON 数组保存开启条件，数组内条件全部按 AND 关系判断。当前支持：

- `quest_completed`：指定任务必须完成
- `player_level`：人物等级比较
- `player_stat`：人物最终 `hp_max/atk/def/spd/mana` 属性比较
- `scene`：玩家当前持久化地图必须匹配
- `item_count`：背包与仓库内指定物品总数比较
- `pet_level`：指定宠物模板或任意宠物的最高等级比较
- `story_flag`：指定服务端剧情标记必须存在
- `time_window`：当前服务端时间必须位于 RFC3339 起止时间内

数值条件的 `operator` 支持 `gte`、`eq`、`lte`。任务列表状态计算、自动接取和主动接取均执行同一套服务端校验；客户端不能自行判定或绕过。历史 `pre_quest_ids` 与 `min_player_level` 仍作为兼容条件继续生效。

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
    "completion_prompt_text": "任务完成！继续下一段旅程吧。",
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
  ],
  "completion_prompt_text": "任务完成！继续下一段旅程吧。"
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
- `completion_prompt_text` 表示任务提交成功后客户端先展示的完成提示文案，支持 Godot RichTextLabel BBCode；为空时客户端跳过该提示，直接展示升级/奖励结算
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

## 2026-07-09 实装补充：任务面板闭环

- `QuestSummary` 运行时响应已补充 `rewards` 奖励预览数组，结构沿用 `QuestReward`。
- `QuestObjectiveState` 已补充 `event_type`，客户端可直接显示当前目标类型与进度。
- `QUEST_SUBMIT_REQ` 只允许提交 `READY_TO_SUBMIT` 且所有目标完成的任务；未完成任务提交会通过 `ERROR_PUSH` 返回 `quest not ready to submit`。
- 客户端任务面板按 `quest_type` 分为 `MAIN`、`SIDE`、`DAILY`，主按钮根据 `state` 映射为：`AVAILABLE=领取`、`ACCEPTED=追踪`、`READY_TO_SUBMIT=交付`。

## 2026-07-09 实装补充：滚动任务列表与领取规则

- 客户端任务列表按 `quest_type` 分类后使用滚动容器展示；每个服务端返回的可见任务生成一个任务卡片，`LOCKED` 与 `COMPLETED` 不在面板中展示。
- 任务卡片进度只信任 `QuestObjectiveState.current / target`：进度条 `step=1`，当前值为 `current`，最大值为 `target`，文案显示 `current/target`。
- 任务达到目标后服务端状态为 `READY_TO_SUBMIT`。如果 `submit_npc_id > 0`，客户端只展示“前往”，奖励仍在对应 NPC 处交付领取；如果 `submit_npc_id = 0`，客户端展示“领取”，点击后发送 `QUEST_SUBMIT_REQ`。
- `AVAILABLE` 且 `start_npc_id > 0` 的任务同样只展示“前往”，需要到 NPC 处领取；`start_npc_id = 0` 时才允许任务面板直接发送 `QUEST_ACCEPT_REQ`。

## 2026-07-09 实装补充：任务图标 ID

`QuestSummary` 新增字段：

```json
{
  "client_icon_id": 1
}
```

说明：
- `client_icon_id` 是服务端任务模板中的任务图标编号，客户端只用它查本地任务图标注册表。
- 服务端不得下发 `res://` 路径或贴图文件名，避免把客户端资源组织暴露给服务端配置。
- 客户端当前预置：`1=主线默认`、`2=对话任务`、`3=战斗任务`；未命中时回退默认任务图标。

## 2026-07-09 实装补充：客户端图标 ID 复用规则

- `client_icon_id` 是客户端任务图标注册表的 ID，不是任务 ID，也不是服务端资源路径。
- 多个 `QuestSummary` 可以携带相同的 `client_icon_id`，客户端应渲染为同一张图标。
- 服务端不对 `client_icon_id` 做唯一性限制；如果客户端未注册该 ID，应显示默认任务图标。
