# 实时协议草案

## 包结构

实时消息使用固定二进制包头 + protobuf 消息体：

```text
| packet_len:u32 | cmd:u16 | seq:u32 | ts_ms:u64 | code:u32 | checksum:u32 | body:bytes |
```

字段说明：
- `packet_len`：整个包长度
- `cmd`：消息号
- `seq`：请求序号；客户端请求自增，服务端响应回传；服务端主动推送填 `0`
- `ts_ms`：发送时间戳
- `code`：业务码；请求固定 `0`，响应为错误码
- `checksum`：建议 `crc32(cmd|seq|ts_ms|body)`
- `body`：对应 `cmd` 的 protobuf 消息体

## 规则约束

- 所有会产生副作用的请求必须带 `op_id`
- 世界和战斗分别维护 `scene_version`、`battle_version`
- 世界移动只提交“目标点意图”，不提交最终坐标
- 战斗只提交“回合行动意图”，不提交伤害结果
- 断线重连第一版只做全量重同步，不做增量补帧

## cmd 编号规划

### 1000-1099 连接 / 鉴权 / 会话
- `1001 WS_AUTH_REQ`
- `1002 WS_AUTH_RESP`
- `1003 HEARTBEAT_REQ`
- `1004 HEARTBEAT_RESP`
- `1011 FORCE_OFFLINE_PUSH`
- `1012 ERROR_PUSH`
- `1021 RECONNECT_REQ`
- `1022 RECONNECT_RESP`

### 2000-2099 世界 / 地图 / AOI / 交互
- `2001 ENTER_WORLD_REQ`
- `2002 ENTER_WORLD_RESP`
- `2011 ENTITY_ENTER_PUSH`
- `2012 ENTITY_LEAVE_PUSH`
- `2013 ENTITY_MOVE_PUSH`
- `2014 WORLD_RESYNC_PUSH`
- `2021 MOVE_INTENT_REQ`
- `2022 MOVE_INTENT_RESP`
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

建议响应格式：

```json
{
  "code": 200,
  "msg": "success",
  "uuid": "trace-id",
  "data": {
    "player_id": 10001,
    "access_jwt": "xxx",
    "ws_token": "xxx",
    "ws_expire_at": 1710000000
  }
}
```

### token 角色分离
- `access_jwt`：HTTP 登录态
- `ws_token`：首次 WSS 鉴权令牌
- `reconnect_token`：短时断线重连令牌
