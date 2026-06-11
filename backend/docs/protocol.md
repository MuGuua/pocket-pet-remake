# 实时协议草案

当前服务端实现以 `server/internal/protocol` 为准。本文档已按当前代码同步：

- WebSocket 路径：`/ws`
- 包头：固定二进制头
- 消息体：`JSON` 编码，不是 protobuf
- 校验：`crc32(cmd|seq|ts_ms|body)`

## 包结构

实时消息使用固定二进制包头 + JSON 消息体：

```text
| packet_len:u32 | cmd:u16 | seq:u32 | ts_ms:u64 | code:u32 | checksum:u32 | body:bytes |
```

字段说明：
- `packet_len`：整个包长度
- `cmd`：消息号
- `seq`：请求序号；客户端请求自增，服务端响应回传；服务端主动推送填 `0`
- `ts_ms`：发送时间戳
- `code`：业务码；请求固定 `0`，响应为错误码
- `checksum`：当前实现为 `crc32(cmd|seq|ts_ms|body)`
- `body`：对应 `cmd` 的 JSON 字节串；空请求体可为空字节数组

## 当前已实现接口

### HTTP

- `POST /api/v1/auth/login`
- `GET /healthz`

### WebSocket

- `GET /ws`
- 只接受 `binary message`

## 规则约束

- 不是所有请求都强制带 `op_id`；当前仅 `MOVE_INTENT_REQ` 定义了 `op_id`
- 世界和战斗分别维护 `scene_version`、`battle_version`
- 世界移动只提交“目标点意图”，不提交最终坐标
- 战斗只提交“回合行动意图”，不提交伤害结果
- 断线重连第一版只做全量重同步，不做增量补帧

## cmd 编号

以下编号与 `server/internal/protocol/command.go` 一致。标注“已实现”的命令可以直接联调，其余目前仅保留编号。

### 1000-1099 连接 / 鉴权 / 会话
- `1001 WS_AUTH_REQ`（已实现）
- `1002 WS_AUTH_RESP`（已实现）
- `1003 HEARTBEAT_REQ`（已实现）
- `1004 HEARTBEAT_RESP`（已实现）
- `1011 FORCE_OFFLINE_PUSH`
- `1012 ERROR_PUSH`（已实现）
- `1021 RECONNECT_REQ`
- `1022 RECONNECT_RESP`

### 2000-2099 世界 / 地图 / AOI / 交互
- `2001 ENTER_WORLD_REQ`（已实现）
- `2002 ENTER_WORLD_RESP`（已实现）
- `2011 ENTITY_ENTER_PUSH`
- `2012 ENTITY_LEAVE_PUSH`
- `2013 ENTITY_MOVE_PUSH`（已实现）
- `2014 WORLD_RESYNC_PUSH`（已实现）
- `2021 MOVE_INTENT_REQ`（已实现）
- `2022 MOVE_INTENT_RESP`（已实现）
- `2031 INTERACT_REQ`
- `2032 INTERACT_RESP`
- `2041 ENCOUNTER_PUSH`

### 3000-3099 宠物 / 编队
- `3001 PET_LIST_REQ`
- `3002 PET_LIST_RESP`
- `3011 PET_UPDATE_PUSH`
- `3021 PET_LINEUP_SET_REQ`
- `3022 PET_LINEUP_SET_RESP`

### 4000-4099 战斗
- `4001 BATTLE_ACTION_REQ`
- `4002 BATTLE_ACTION_RESP`
- `4011 BATTLE_START_PUSH`
- `4012 BATTLE_STATE_PUSH`
- `4013 BATTLE_RESULT_PUSH`
- `4021 BATTLE_EXIT_REQ`
- `4022 BATTLE_EXIT_RESP`

### 5000-5099 背包 / 道具（当前已实现）
- `5001 BAG_LIST_REQ`
- `5002 BAG_LIST_RESP`
- `5011 BAG_UPDATE_PUSH`
- `5021 USE_ITEM_REQ`
- `5022 USE_ITEM_RESP`

### 5030-5199 背包 / 仓库 / 钱包 / 商店（规划预留，尚未进入当前代码）

以下命令当前用于文档预留，不代表 `server/internal/protocol/command.go` 已经声明：

- `5031 CONTAINER_LIST_REQ`
- `5032 CONTAINER_LIST_RESP`
- `5041 BAG_TO_WAREHOUSE_REQ`
- `5042 BAG_TO_WAREHOUSE_RESP`
- `5051 WAREHOUSE_TO_BAG_REQ`
- `5052 WAREHOUSE_TO_BAG_RESP`
- `5061 CONTAINER_SORT_REQ`
- `5062 CONTAINER_SORT_RESP`
- `5071 CONTAINER_MOVE_REQ`
- `5072 CONTAINER_MOVE_RESP`
- `5081 WALLET_QUERY_REQ`
- `5082 WALLET_QUERY_RESP`
- `5091 WALLET_UPDATE_PUSH`
- `5101 BUY_ITEM_REQ`
- `5102 BUY_ITEM_RESP`
- `5111 SELL_ITEM_REQ`
- `5112 SELL_ITEM_RESP`

详细字段设计、数据库分层与客户端展示约束请同步参考 `backend/docs/bag-system.md`。

### 9000-9099 系统通知
- `9001 NOTICE_PUSH`
- `9002 KICKOUT_PUSH`

## HTTP 接口约束

### 登录
- `POST /api/v1/auth/login`

当前请求体：

```json
{
  "account": "demo",
  "password": "demo123",
  "device_id": "ios-demo"
}
```

当前响应格式：

```json
{
  "code": 200,
  "msg": "success",
  "uuid": "trace-id",
  "data": {
    "player_id": 10001,
    "player_name": "DemoTrainer",
    "access_jwt": "xxx",
    "ws_token": "xxx",
    "ws_expire_at": 1710000000
  }
}
```

### token 角色分离
- `access_jwt`：HTTP 登录态
- `ws_token`：首次 WebSocket 鉴权令牌
- `reconnect_token`：短时断线重连令牌

## WebSocket 消息体

### 1001 WS_AUTH_REQ

```json
{
  "ws_token": "xxx",
  "client_version": "dev-build",
  "device_id": "ios-demo"
}
```

### 1002 WS_AUTH_RESP

```json
{
  "player_id": 10001,
  "session_id": "xxx",
  "reconnect_token": "xxx",
  "heartbeat_sec": 10,
  "server_time_ms": 1710000000000
}
```

### 1003 HEARTBEAT_REQ

```json
{
  "client_time_ms": 1710000000000
}
```

### 1004 HEARTBEAT_RESP

```json
{
  "server_time_ms": 1710000000000
}
```

### 1021 RECONNECT_REQ

```json
{
  "reconnect_token": "xxx",
  "battle_id": 70001,
  "last_frame": 3
}
```

说明：

- 当前第一版断线重连直接使用 `reconnect_token` 恢复实时会话
- 若客户端断线前仍处于战斗中，可额外携带 `battle_id` 与 `last_frame`
- 当前 `last_frame` 直接复用服务端下发的 `frame` / `battle_version`

### 1022 RECONNECT_RESP

```json
{
  "player_id": 10001,
  "session_id": "xxx",
  "reconnect_token": "xxx-new",
  "heartbeat_sec": 10,
  "server_time_ms": 1710000005000,
  "world": {
    "self": {
      "player_id": 10001,
      "name": "DemoTrainer",
      "level": 8
    },
    "player": {
      "player_id": 10001,
      "name": "DemoTrainer",
      "level": 8,
      "hp": 120,
      "hp_max": 120,
      "energy": 100,
      "energy_max": 100,
      "atk": 24,
      "def": 12,
      "spd": 18,
      "mana": 20,
      "skill_ids": [1101, 1001]
    },
    "scene_id": 1,
    "self_pos": {
      "x": 8,
      "y": 6
    },
    "scene_version": 1,
    "nearby_entities": [],
    "lineup": [],
    "gold": 118
  },
  "battle_start": {
    "battle_id": 70001,
    "battle_type": 1,
    "battle_version": 3,
    "allies": [],
    "enemies": [],
    "round": 1,
    "phase": "command",
    "active_actor_id": 20002,
    "active_pet_uid": 20002,
    "command_deadline_ms": 1710000015000,
    "auto_battle_enabled": true,
    "pending_actor_ids": [20002],
    "controllable_actor_ids": [20001, 20002]
  },
  "battle_state": {
    "battle_id": 70001,
    "battle_version": 3,
    "frame": 3,
    "round": 1,
    "phase": "command",
    "events": [],
    "actors": [],
    "active_actor_id": 20002,
    "active_pet_uid": 20002,
    "command_deadline_ms": 1710000015000,
    "auto_battle_enabled": true,
    "pending_actor_ids": [20002],
    "controllable_actor_ids": [20001, 20002]
  },
  "battle_replay_states": [
    {
      "battle_id": 70001,
      "battle_version": 2,
      "frame": 2,
      "round": 1,
      "phase": "command",
      "events": [],
      "actors": [],
      "active_actor_id": 20002,
      "active_pet_uid": 20002,
      "command_deadline_ms": 1710000010000,
      "auto_battle_enabled": false,
      "pending_actor_ids": [20002],
      "controllable_actor_ids": [20001, 20002]
    }
  ],
  "battle_result": {
    "battle_id": 70001,
    "win": true,
    "return_scene_id": 1,
    "return_pos": {
      "x": 8,
      "y": 6
    },
    "reason": "enemy defeated",
    "reward_gold": 18,
    "reward_player_exp": 28,
    "player_gold": 118,
    "player_exp": 28,
    "pet_rewards": [],
    "drop_texts": [
      "掉落: 野性毛皮 x1"
    ]
  }
}
```

说明：

- `world` 为断线恢复后的最新世界全量快照，客户端可按 `ENTER_WORLD_RESP/WORLD_RESYNC_PUSH` 的同等结构重建世界态
- `battle_start` / `battle_state` 仅在玩家重连时仍处于活动战斗中时返回；客户端应按正常战斗进入流程恢复界面
- `battle_replay_states` 表示服务端保留的最近若干帧战斗状态；当客户端上报的 `last_frame` 落后但仍在缓存窗口内时，可先按顺序回放这些状态，再与当前状态对齐
- `battle_result` 仅在断线期间战斗已经由服务端托管结束、但客户端还没收到结算时返回；客户端应按正常 `BATTLE_RESULT_PUSH` 的处理链路展示奖励并退出战斗界面
- 当前仍是“全量重同步优先、最近帧补发增强”的最小版；如果客户端落后帧数超过服务端缓存窗口，仍应回退到当前全量快照
- `reconnect_token` 会在每次重连成功后轮换，旧 token 应立即作废

### 1012 ERROR_PUSH

```json
{
  "code": 10001,
  "msg": "invalid ws token"
}
```

### 2001 ENTER_WORLD_REQ

```json
{}
```

### 2002 ENTER_WORLD_RESP

```json
{
  "self": {
    "player_id": 10001,
    "name": "DemoTrainer",
    "level": 1
  },
  "scene_id": 1,
  "self_pos": {
    "x": 0,
    "y": 0
  },
  "scene_version": 1,
  "nearby_entities": [
    {
      "entity_id": 90001,
      "entity_type": 2,
      "pos": {
        "x": 10,
        "y": 6
      },
      "dir": 2,
      "speed": 0,
      "name": "GuideNPC"
    }
  ],
  "lineup": [],
  "gold": 100
}
```

### 2021 MOVE_INTENT_REQ

当前实现里，`MOVE_INTENT_REQ` 用于“申请切换到目标地图”。地图内逐点移动由客户端本地表现，不需要每步都上报服务端。

```json
{
  "op_id": 1,
  "move_seq": 1,
  "scene_id": 1,
  "target_scene_id": 2,
  "portal_id": 1001
}
```

说明：

- `target_scene_id`：目标地图
- `portal_id`：可选，表示通过哪个门/入口触发切图；当前门区切图优先带上该字段

### 2022 MOVE_INTENT_RESP

```json
{
  "accepted": true,
  "move_seq": 1,
  "scene_id": 2,
  "corrected_pos": {
    "x": -6,
    "y": 4
  },
  "reason": ""
}
```

说明：

- `scene_id`：服务端确认后的目标地图
- `corrected_pos`：进入目标地图后的权威入口落点；当前最小实现会按“从哪张地图进入”决定，不再统一落在地图中心
- 如果 `target_scene_id` 为空、为 `0`、或等于当前 `scene_id`，服务端只返回成功确认，表示“地图内移动由客户端处理”
- 如果带了 `portal_id`，服务端会优先按门/入口配置决定目标地图与入口落点；若 `portal_id` 非法则拒绝本次切图

### 2014 WORLD_RESYNC_PUSH

```json
{
  "scene_id": 2,
  "self_pos": {
    "x": -6,
    "y": 4
  },
  "scene_version": 1,
  "nearby_entities": [
    {
      "entity_id": 90002,
      "entity_type": 2,
      "pos": {
        "x": 5,
        "y": 4
      },
      "dir": 1,
      "speed": 0,
      "name": "StationKeeper"
    }
  ]
}
```

### 2031 INTERACT_REQ

当前最小战斗入口使用“与附近 NPC 交互”触发：

```json
{
  "entity_id": 90001
}
```

### 2032 INTERACT_RESP

```json
{
  "accepted": true,
  "reason": "battle started"
}
```

### 3001 PET_LIST_REQ

```json
{}
```

### 3002 PET_LIST_RESP

```json
{
  "pets": [
    {
      "pet_uid": 20001,
      "pet_id": 101,
      "level": 5,
      "exp": 120,
      "quality": 1,
      "hp": 32,
      "hp_max": 32,
      "atk": 14,
      "def": 10,
      "spd": 12,
      "skill_ids": [1001, 1002],
      "in_lineup": true
    }
  ],
  "lineup": [
    {
      "pet_uid": 20001,
      "pet_id": 101,
      "level": 5,
      "hp": 32,
      "hp_max": 32
    }
  ]
}
```

说明：

- `pets` 返回玩家拥有的完整宠物实例列表
- `lineup` 返回当前编队摘要和顺序
- `in_lineup` 仅用于客户端展示，不替代 `lineup` 顺序本身

### 3011 PET_UPDATE_PUSH

当服务端结算会改变宠物实例状态的结果时，可直接推送单只宠物最新详情；当前最小实现用于“战斗结束后回写主战宠 HP”：

```json
{
  "pet": {
    "pet_uid": 20001,
    "pet_id": 101,
    "level": 5,
    "exp": 120,
    "quality": 1,
    "hp": 28,
    "hp_max": 32,
    "atk": 14,
    "def": 10,
    "spd": 12,
    "skill_ids": [1001, 1002],
    "in_lineup": true
  }
}
```

说明：

- 当前只推送发生变化的单只宠物详情
- 客户端按 `pet_uid` 合并本地宠物实例
- 宠物列表和编队摘要后续再次查询时，也应与该推送保持一致

### 3021 PET_LINEUP_SET_REQ

```json
{
  "op_id": 1,
  "pet_uids": [20003, 20001]
}
```

说明：

- 客户端提交的是完整编队顺序
- 当前同一只宠物不能重复进入编队

### 3022 PET_LINEUP_SET_RESP

```json
{
  "accepted": true,
  "lineup": [
    {
      "pet_uid": 20003,
      "pet_id": 101,
      "level": 3,
      "hp": 24,
      "hp_max": 24
    },
    {
      "pet_uid": 20001,
      "pet_id": 101,
      "level": 5,
      "hp": 32,
      "hp_max": 32
    }
  ],
  "reason": "lineup updated"
}
```

## 背包扩展协议规划

以下内容描述的是背包、仓库、钱包与购买/出售链路的规划协议口径，当前尚未全部进入服务端代码。实现时应与：

- `backend/docs/bag-system.md`
- `backend/docs/message-routing.md`
- `server/internal/protocol/command.go`

保持同步。

### 通用钱包结构

```json
{
  "total_copper": 2345678,
  "gold": 2,
  "silver": 345,
  "copper": 678
}
```

说明：

- `total_copper` 是服务端权威结算字段
- `gold / silver / copper` 是返回给客户端直接渲染的拆分结果

### 通用容器快照结构

```json
{
  "container_type": "bag",
  "capacity": 30,
  "max_capacity": 300,
  "used_slots": 12,
  "items": []
}
```

说明：

- `container_type` 当前规划值为 `bag` 或 `warehouse`
- 打开面板前客户端应先请求全量快照，再展示最新数据

### 5031 CONTAINER_LIST_REQ（规划）

```json
{
  "container_type": "warehouse"
}
```

### 5032 CONTAINER_LIST_RESP（规划）

```json
{
  "container": {
    "container_type": "warehouse",
    "capacity": 30,
    "max_capacity": 300,
    "used_slots": 6,
    "items": []
  },
  "wallet": {
    "total_copper": 2345678,
    "gold": 2,
    "silver": 345,
    "copper": 678
  }
}
```

说明：

- 仓库面板可按需带回 `wallet`，方便后续付费扩容或展示资产

### 5041 BAG_TO_WAREHOUSE_REQ（规划）

```json
{
  "from_slot_index": 4,
  "quantity": 10
}
```

### 5042 BAG_TO_WAREHOUSE_RESP（规划）

```json
{
  "moved_item_id": 1002,
  "moved_item_uid": "",
  "moved_quantity": 10,
  "from_container_type": "bag",
  "to_container_type": "warehouse"
}
```

### 5051 WAREHOUSE_TO_BAG_REQ（规划）

```json
{
  "from_slot_index": 2,
  "quantity": 1
}
```

### 5052 WAREHOUSE_TO_BAG_RESP（规划）

```json
{
  "moved_item_id": 2001,
  "moved_item_uid": "eq_10001",
  "moved_quantity": 1,
  "from_container_type": "warehouse",
  "to_container_type": "bag"
}
```

### 5061 CONTAINER_SORT_REQ（规划）

```json
{
  "container_type": "bag"
}
```

### 5062 CONTAINER_SORT_RESP（规划）

```json
{
  "container_type": "bag",
  "sorted": true
}
```

### 5071 CONTAINER_MOVE_REQ（规划）

```json
{
  "container_type": "bag",
  "from_slot_index": 2,
  "to_slot_index": 8,
  "quantity": 1
}
```

### 5072 CONTAINER_MOVE_RESP（规划）

```json
{
  "container_type": "bag",
  "from_slot_index": 2,
  "to_slot_index": 8,
  "moved": true
}
```

### 5081 WALLET_QUERY_REQ（规划）

```json
{}
```

### 5082 WALLET_QUERY_RESP（规划）

```json
{
  "wallet": {
    "total_copper": 2345678,
    "gold": 2,
    "silver": 345,
    "copper": 678
  }
}
```

### 5091 WALLET_UPDATE_PUSH（规划）

```json
{
  "wallet": {
    "total_copper": 2350678,
    "gold": 2,
    "silver": 350,
    "copper": 678
  },
  "reason_type": "quest_reward",
  "reason_ref_id": 10001
}
```

### 5101 BUY_ITEM_REQ（规划）

```json
{
  "shop_id": 1,
  "goods_id": 10001,
  "item_id": 1001,
  "quantity": 5
}
```

### 5102 BUY_ITEM_RESP（规划）

```json
{
  "shop_id": 1,
  "goods_id": 10001,
  "item_id": 1001,
  "quantity": 5,
  "cost": {
    "currency_type": "base_coin",
    "total_copper": 12500,
    "gold": 0,
    "silver": 12,
    "copper": 500
  },
  "wallet": {
    "total_copper": 2333178,
    "gold": 2,
    "silver": 333,
    "copper": 178
  }
}
```

### 5111 SELL_ITEM_REQ（规划）

```json
{
  "container_type": "bag",
  "slot_index": 6,
  "quantity": 3
}
```

### 5112 SELL_ITEM_RESP（规划）

```json
{
  "container_type": "bag",
  "slot_index": 6,
  "item_id": 1001,
  "sold_quantity": 3,
  "gain": {
    "currency_type": "base_coin",
    "total_copper": 300,
    "gold": 0,
    "silver": 0,
    "copper": 300
  },
  "wallet": {
    "total_copper": 2345978,
    "gold": 2,
    "silver": 345,
    "copper": 978
  }
}
```

### 4001 BATTLE_ACTION_REQ

战斗动作只提交意图，伤害、回合推进和胜负均由服务端结算：

```json
{
  "op_id": 1,
  "battle_id": 70001,
  "round": 1,
  "action_type": 1,
  "actor_id": 20001,
  "skill_id": 1001,
  "target_id": 190001
}
```

说明：

- `action_type=1`：提交技能/普通攻击动作，需携带 `actor_id`、`skill_id` 与 `target_id`
- `action_type=4`：提交逃跑请求，当前最小 PVE 版本由服务端直接处理逃跑结果
- `action_type=5`：切换服务端自动战斗开关，此时 `actor_id`、`skill_id`、`target_id` 可为 `0`
- `auto_battle_enabled`：仅在 `action_type=5` 时使用，表示当前请求希望服务端开启还是关闭托管

自动战斗开关示例：

```json
{
  "op_id": 2,
  "battle_id": 70001,
  "round": 1,
  "action_type": 5,
  "actor_id": 0,
  "skill_id": 0,
  "target_id": 0,
  "auto_battle_enabled": true
}
```

### 4002 BATTLE_ACTION_RESP

```json
{
  "accepted": true,
  "reason": "action accepted"
}
```

### 4011 BATTLE_START_PUSH

```json
{
  "battle_id": 70001,
  "battle_type": 1,
  "battle_version": 1,
  "allies": [
    {
      "actor_id": 10001,
      "actor_type": 1,
      "unit_class": 1,
      "pet_uid": 0,
      "pet_id": 0,
      "name": "DemoTrainer",
      "hp": 120,
      "hp_max": 120,
      "atk": 24,
      "def": 12,
      "spd": 18,
      "skills": [
        {"skill_id": 1101, "name": "裂空斩", "target_type": "enemy_single"},
        {"skill_id": 1001, "name": "普通攻击", "target_type": "enemy_single"}
      ],
      "skill_ids": [1101, 1001],
      "status_ids": [],
      "lineup_index": 0
    },
    {
      "actor_id": 20001,
      "actor_type": 1,
      "unit_class": 2,
      "pet_uid": 20001,
      "pet_id": 101,
      "name": "DemoTrainer 的1号宠物",
      "hp": 32,
      "hp_max": 32,
      "atk": 14,
      "def": 10,
      "spd": 12,
      "skills": [
        {"skill_id": 1001, "name": "普通攻击", "target_type": "enemy_single"},
        {"skill_id": 1002, "name": "火花冲击", "target_type": "enemy_all"}
      ],
      "skill_ids": [1001, 1002],
      "status_ids": [],
      "lineup_index": 0
    }
  ],
  "enemies": [
    {
      "actor_id": 900011,
      "actor_type": 2,
      "pet_uid": 0,
      "pet_id": 9001,
      "name": "GuideNPC",
      "hp": 22,
      "hp_max": 22,
      "atk": 12,
      "def": 9,
      "spd": 8,
      "skills": [
        {"skill_id": 90001, "name": "野性撞击", "target_type": "enemy_single"},
        {"skill_id": 90002, "name": "利爪突袭", "target_type": "enemy_single"}
      ],
      "skill_ids": [90001, 90002],
      "status_ids": [],
      "lineup_index": 0
    }
  ],
  "round": 1,
  "phase": "command",
  "active_actor_id": 20001,
  "active_pet_uid": 20001,
  "command_deadline_ms": 1710000015000,
  "auto_battle_enabled": false,
  "pending_actor_ids": [20001, 20002],
  "controllable_actor_ids": [20001, 20002]
}
```

说明：

- `skill_ids` 仅表示当前角色可提交的技能意图列表
- `skills` 为 `skill_ids` 的增强版快照，额外携带技能展示名和目标类型，客户端应优先使用它来决定按钮文案和友/敌方目标选择
- `unit_class` 用来区分当前战斗单位是真人角色、宠物还是怪物；当前约定 `1=人物`、`2=宠物`、`4=怪物`
- 当前已使用的 `target_type` 包括：`enemy_single`、`ally_single`、`enemy_all`、`enemy_multi`
- `target_count` 表示技能配置的目标数量；当前单体技能通常为 `1`，`enemy_all` 可忽略该字段，`enemy_multi` 表示客户端先指定一个主目标，剩余目标数量由服务端按 `target_count` 自动补足
- `phase=command` 表示当前轮到客户端继续为己方单位收集动作
- `command_deadline_ms` 表示当前命令阶段由服务端给出的权威截止时间；超时补行动由服务端负责
- `auto_battle_enabled` 表示当前战斗是否已经进入服务端自动托管模式
- `pending_actor_ids` 表示这一回合还没提交动作的己方单位
- `active_actor_id` / `active_pet_uid` 明确当前应高亮的己方单位；若当前轮到人物 actor，则 `active_pet_uid=0`
- 技能名称、伤害、回合推进和胜负判定都由服务端技能表和战斗状态机决定
- 客户端只负责展示按钮和发送 `skill_id`

### 4012 BATTLE_STATE_PUSH

```json
{
  "battle_id": 70001,
  "battle_version": 2,
  "round": 2,
  "phase": "command",
  "active_actor_id": 20001,
  "active_pet_uid": 20001,
  "command_deadline_ms": 1710000030000,
  "auto_battle_enabled": true,
  "pending_actor_ids": [20001, 20002],
  "controllable_actor_ids": [20001, 20002],
  "events": [
    {
      "event_type": 1,
      "source_id": 20001,
      "target_id": 900011,
      "skill_id": 1002,
      "value": 0,
      "state_id": 0,
      "label": "DemoTrainer 的1号宠物 使用了 火花冲击。"
    },
    {
      "event_type": 2,
      "source_id": 20001,
      "target_id": 900011,
      "skill_id": 1002,
      "value": 22,
      "state_id": 0,
      "label": "GuideNPC 受到 22 点伤害。"
    }
  ],
  "actors": [
    {
      "actor_id": 20001,
      "hp": 32,
      "hp_max": 32,
      "dead": false,
      "can_act": true,
      "status_ids": [],
      "charge_done": false
    },
    {
      "actor_id": 20002,
      "hp": 28,
      "hp_max": 30,
      "dead": false,
      "can_act": true,
      "status_ids": [],
      "charge_done": false
    },
    {
      "actor_id": 900011,
      "hp": 0,
      "hp_max": 22,
      "dead": true,
      "can_act": false,
      "status_ids": [],
      "charge_done": true
    }
  ]
}
```

### 4013 BATTLE_RESULT_PUSH

```json
{
  "battle_id": 70001,
  "win": true,
  "return_scene_id": 1,
  "return_pos": {
    "x": 8,
    "y": 6
  },
  "reason": "enemy defeated",
  "reward_gold": 18,
  "reward_player_exp": 28,
  "player_gold": 118,
  "player_exp": 28,
  "drop_texts": [
    "掉落: 野性毛皮 x1"
  ],
  "pet_rewards": [
    {
      "pet_uid": 20001,
      "exp": 28
    },
    {
      "pet_uid": 20002,
      "exp": 28
    }
  ]
}
```

说明：

- `reward_gold` / `reward_player_exp` 表示本场战斗实际发放的金币与角色经验；失败或逃跑时通常为 `0`
- `player_gold` / `player_exp` 表示服务端发奖后的玩家当前累计值，客户端可直接用来刷新本地摘要
- `drop_texts` 表示本场战斗的文本掉落提示，当前只用于展示，不会写入背包
- `pet_rewards` 表示本场战斗中各参战宠物获得的经验摘要；宠物最终 HP / EXP 明细仍以随后逐条推送的 `3011 PET_UPDATE_PUSH` 为准
- 当前最小 PVE 奖励闭环已覆盖金币、角色经验、宠物经验、文本掉落展示和 `battle_record` 防重记录；真实物品掉落与背包落库仍待后续扩展
