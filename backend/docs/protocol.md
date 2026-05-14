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
  "target_scene_id": 2
}
```

### 2022 MOVE_INTENT_RESP

```json
{
  "accepted": true,
  "move_seq": 1,
  "scene_id": 2,
  "corrected_pos": {
    "x": 2,
    "y": 4
  },
  "reason": ""
}
```

说明：

- `scene_id`：服务端确认后的目标地图
- `corrected_pos`：进入目标地图后的出生点/落点
- 如果 `target_scene_id` 为空、为 `0`、或等于当前 `scene_id`，服务端只返回成功确认，表示“地图内移动由客户端处理”

### 2014 WORLD_RESYNC_PUSH

```json
{
  "scene_id": 2,
  "self_pos": {
    "x": 2,
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
