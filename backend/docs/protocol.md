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

### 5000-5099 背包 / 道具
- `5001 BAG_LIST_REQ`
- `5002 BAG_LIST_RESP`
- `5011 BAG_UPDATE_PUSH`
- `5021 USE_ITEM_REQ`
- `5022 USE_ITEM_RESP`

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
      "actor_id": 20001,
      "actor_type": 1,
      "pet_uid": 20001,
      "pet_id": 101,
      "name": "DemoTrainer 的主战宠",
      "hp": 32,
      "hp_max": 32,
      "skill_ids": [1001, 1002],
      "lineup_index": 0
    }
  ],
  "enemies": [
    {
      "actor_id": 190001,
      "actor_type": 2,
      "pet_uid": 0,
      "pet_id": 9001,
      "name": "GuideNPC",
      "hp": 22,
      "hp_max": 22,
      "skill_ids": [90001, 90002],
      "lineup_index": 0
    }
  ],
  "round": 1,
  "active_actor_id": 20001,
  "active_pet_uid": 20001
}
```

说明：

- `skill_ids` 仅表示当前角色可提交的技能意图列表
- `active_actor_id` / `active_pet_uid` 明确当前己方出战宠锚点
- 技能名称、伤害、回合推进和胜负判定都由服务端技能表和战斗状态机决定
- 客户端只负责展示按钮和发送 `skill_id`

### 4012 BATTLE_STATE_PUSH

```json
{
  "battle_id": 70001,
  "battle_version": 2,
  "round": 2,
  "active_actor_id": 20001,
  "active_pet_uid": 20001,
  "events": [
    {
      "event_type": 1,
      "source_id": 20001,
      "target_id": 190001,
      "skill_id": 1001,
      "value": 0,
      "state_id": 0
    },
    {
      "event_type": 2,
      "source_id": 20001,
      "target_id": 190001,
      "skill_id": 1001,
      "value": 11,
      "state_id": 0
    }
  ],
  "actors": [
    {
      "actor_id": 20001,
      "hp": 28,
      "hp_max": 32,
      "dead": false
    },
    {
      "actor_id": 190001,
      "hp": 11,
      "hp_max": 22,
      "dead": false
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
  "reason": "enemy defeated"
}
```

说明：

- 当前 `BATTLE_RESULT_PUSH` 仍只负责表达战斗胜负与返回世界信息
- 如果该场战斗使主战宠 HP 发生变化，服务端会在结果后继续推送 `3011 PET_UPDATE_PUSH`
