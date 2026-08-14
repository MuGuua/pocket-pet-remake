# 实时协议草案

当前服务端实现以 `server/internal/protocol` 为准。本文档已按当前代码同步：

- WebSocket 路径：`/ws`
- 包头：固定二进制头
- 消息体：`JSON` 编码，不是 protobuf
- 校验：`crc32(cmd|seq|ts_ms|body)`

## 后台世界移动配置

- `GET /api/admin/world/movement-config`：需要 `world_movement:view`，返回数据库配置、更新时间、最近操作原因和管理员 ID。
- `PUT /api/admin/world/movement-config`：需要 `world_movement:edit`；请求包含 `speed_milli_cells_per_second`、`max_elapsed_ms`、`axis_tolerance_milli` 和必填 `reason`。成功响应代表数据库配置与当前服务进程运行时快照均已更新。
- `max_elapsed_ms` 允许 `50..2000`，`axis_tolerance_milli` 允许 `0..1000`，速度必须大于零；全部数值均使用整数，客户端与后台不得自行提供运行时覆盖值。

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
- `2033 NPC_ACTION_REQ`（已实现）
- `2034 NPC_ACTION_RESP`（已实现）
- `2042 NPC_MENU_REQ`（已实现）
- `2043 NPC_MENU_RESP`（已实现）
- `2044 SCENE_TRIGGER_PUSH`（已实现）
- `2045 SCENE_TRIGGER_ACK_REQ`（已实现）
- `2046 SCENE_TRIGGER_ACK_RESP`（已实现）
- `2047 NPC_MENU_BATCH_REQ`（已实现）
- `2048 NPC_MENU_BATCH_RESP`（已实现）
- `2037 NPC_DIALOGUE_NEXT_REQ`（已实现）
- `2038 NPC_DIALOGUE_RESP`（已实现）
- `2039 NPC_DIALOGUE_CHOOSE_REQ`（已实现）
- `2035 WILD_ENCOUNTER_REQ`（已实现）
- `2036 WILD_ENCOUNTER_RESP`（已实现）
- `2041 ENCOUNTER_PUSH`（预留，暗雷改由客户端上报 + `4011 BATTLE_START_PUSH`）

### 2060-2069 玩家 / 宠物成长
- `2061 PLAYER_ALLOCATE_ATTR_REQ`
- `2062 PLAYER_ALLOCATE_ATTR_RESP`
- `2063 PET_ALLOCATE_ATTR_REQ`
- `2064 PET_ALLOCATE_ATTR_RESP`
- `2065 PLAYER_PROFILE_REQ`
- `2066 PLAYER_PROFILE_RESP`

`PLAYER_PROFILE_REQ` 只获取当前会话人物的权威属性，供人物状态面板打开前刷新数据。该请求不加载世界快照、背包物品、钱包、宠物、任务或场景触发器。

```json
{}
```

`PLAYER_PROFILE_RESP`：

```json
{
  "player": {
    "player_id": 10001,
    "name": "DemoTrainer",
    "level": 12,
    "exp": 3456,
    "hp": 120,
    "hp_max": 120,
    "vigor": 100,
    "vigor_max": 100,
    "spirit": 40,
    "spirit_max": 40,
    "skin_id": "初始形象男_001"
  }
}
```

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

当前 `5001/5002` 已支持新版背包 UI 使用的服务端分页参数：
- `page`
- `page_size`
- `category`（`all` / `equipment` / `other`）

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

### 注册
- `POST /api/v1/auth/register`

当前请求体：

```json
{
  "account": "new_trainer",
  "password": "pwd123456",
  "gender": "female"
}
```

当前响应格式：

```json
{
  "code": 200,
  "msg": "success",
  "uuid": "trace-id",
  "data": {
    "player_id": 10003,
    "account": "new_trainer",
    "player_name": "new_trainer",
    "skin_id": "初始形象女_002"
  }
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
    "command_deadline_ms": 0,
    "auto_battle_enabled": false,
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
    "command_deadline_ms": 0,
    "auto_battle_enabled": false,
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
      "command_deadline_ms": 0,
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
    "player_gold": 20,
    "player_exp": 28,
    "pet_rewards": [],
    "drop_texts": [
      "掉落: 野性毛皮 x1"
    ]
  },
  "active_dialogue": {
    "entity_id": 93001,
    "npc_name": "市场理萌",
    "node": {
      "dialogue_id": 1,
      "node_id": "start",
      "node_type": "line",
      "speaker": "市场理萌",
      "is_player_speaker": false,
      "content": "你先稍等一下，我把前面的货箱挪开。",
      "content_format": "plain",
      "portrait_key": "npc_limeng_normal",
      "client_animation_key": "",
      "client_animation_block": false,
      "options": [],
      "is_end": false,
      "effect_notice": ""
    }
  }
}
```

说明：

- `world` 为断线恢复后的最新世界全量快照，客户端可按 `ENTER_WORLD_RESP/WORLD_RESYNC_PUSH` 的同等结构重建世界态
- `active_dialogue` 仅在玩家断线前仍有未结束的 NPC 结构化剧情时返回；客户端应直接恢复对话面板，而不是重新发起 `NPC_ACTION_REQ`
- `battle_start` / `battle_state` 仅在玩家重连时仍处于活动战斗中时返回；客户端应按正常战斗进入流程恢复界面
- `battle_replay_states` 表示服务端保留的最近若干帧战斗状态；当客户端上报的 `last_frame` 落后但仍在缓存窗口内时，可先按顺序回放这些状态，再与当前状态对齐
- `battle_result` 仅在断线期间战斗已经被服务端结算、但客户端还没收到结算时返回；单人战斗断线会被判定为失败且不发放奖励，客户端应按正常 `BATTLE_RESULT_PUSH` 的处理链路退出战斗界面
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
  "self_precise_pos": {
    "x": 250,
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
    },
    {
      "entity_id": 10002,
      "player_id": 10002,
      "entity_type": 1,
      "pos": {
        "x": 7,
        "y": 7
      },
      "dir": 2,
      "speed": 0,
      "name": "RivalTrainer",
      "level": 1,
      "exp": 0,
      "hp": 110,
      "hp_max": 110,
      "vigor": 100,
      "vigor_max": 100,
      "spirit": 40,
      "spirit_max": 40,
      "skin_id": "初始形象男_001",
      "following_pet": {
        "pet_uid": 21001,
        "pet_id": 101,
        "name": "小火龙",
        "level": 5,
        "exp": 110,
        "hp": 31,
        "hp_max": 31,
        "spirit": 12,
        "spirit_max": 20,
        "skin_id": "嫩叶犬_001"
      }
    }
  ],
  "lineup": [],
  "gold": 100,
  "wild_encounter": {
    "enabled": false,
    "scene_id": 1,
    "encounter_rate": 0,
    "spawn_monster_ids": [],
    "targets": []
  }
}
```

- 玩家实体的 `following_pet` 来自服务端持久化编队首位；无出战宠物或形象资源时省略该字段。
- `self_precise_pos` 仅在断线重连存在 Redis权威移动状态时返回，使用千分之一场景格定点整数；客户端应优先于 `self_pos` 恢复人物位置。首次进入世界或 Redis状态缺失时省略，并使用 PostgreSQL来源的 `self_pos`。
- 玩家实体与跟随宠物只携带世界同屏展示所需的名字、等级、经验、血量、精力和形象等轻量字段；这些字段来自服务端持久化数据，不包含背包内容。
- `ENTER_WORLD_RESP.nearby_entities` 与 `ENTITY_ENTER_PUSH.entity` 使用同一结构；玩家更换编队后服务端复用 `ENTITY_ENTER_PUSH` 刷新同场景远端表现。

`wild_encounter` 字段说明：

- 服务端在 `ENTER_WORLD_RESP` 与 `WORLD_RESYNC_PUSH` 中下发当前地图暗雷配置
- 地图内移动由客户端本地处理；客户端按 `encounter_rate`（万分比，800=8%）逐步判定是否触发
- 触发后客户端发送 `2035 WILD_ENCOUNTER_REQ`，服务端校验通过后按后台配置的怪物编队权重随机选择一个编队，再推送 `4011 BATTLE_START_PUSH`
- `spawn_monster_ids` 仅作为客户端展示/兼容字段；实际开战怪物以服务端随机选中的后台 `formations` 编队为准
- `targets` 由服务端按 `spawn_monster_ids` 查询已启用的数据库怪物模板生成，每项包含 `monster_id`、`monster_name`、`skin_id`，仅用于挂机目标选择面板
- 无暗雷配置的地图返回 `enabled=false`

### 2021 MOVE_INTENT_REQ

`MOVE_INTENT_REQ` 同时承载门区切图、世界地图快速传送和同场景移动同步。普通门切图只使用 `target_scene_id`、`portal_id`，不携带地图出生落点；地图快速传送额外传 `map_teleport=true`，同样不携带本地出生点；只有同场景移动使用整数 `target_pos` 持久化位置，并以限频后的表现字段同步其他玩家。服务端保留 `target_pos` 字段解析只是为了兼容旧客户端，跨场景请求一律忽略该值。

```json
{
  "op_id": 1,
  "move_seq": 1,
  "scene_id": 1,
  "target_scene_id": 2,
  "portal_id": 1001
}
```

世界地图快速传送示例：

```json
{
  "op_id": 3,
  "move_seq": 19,
  "scene_id": 1,
  "target_scene_id": 6,
  "portal_id": 0,
  "map_teleport": true
}
```

同场景移动示例：

```json
{
  "op_id": 2,
  "move_seq": 18,
  "scene_id": 1,
  "target_pos": {"x": 10, "y": 7},
  "precise_pos": {"x": 10250, "y": 7000},
  "facing": {"x": 1, "y": 0},
  "moving": true,
  "input": {"x": 1, "y": 0},
  "client_tick": 18200
}
```

说明：

- `target_scene_id`：目标地图
- `portal_id`：可选，表示通过哪个门/入口触发切图；当前门区切图优先带上该字段
- `map_teleport`：可选，缺省为 `false`；为 `true` 时服务端从 `world_map_teleport_node` 校验开放状态并读取中心出生格，客户端不得提交或覆盖出生坐标
- `target_pos`：仅用于同场景移动时表示当前整数格；字段保留兼容旧客户端，但普通门切图和世界地图快速传送都会忽略该值
- 正式登录、普通门和世界地图快速传送的唯一人物落点均来自服务端权威快照 `self_pos`；客户端地图加载完成后只做场景坐标到 Godot 像素坐标的转换。历史场景导出出生配置不属于协议字段，也不再覆盖服务端快照
- `precise_pos`：千分之一场景格的定点表现坐标；服务端会限制在 `target_pos` 周围半格内，不直接写入数据库
- `facing`：四方向单位向量，只接受上下左右，服务端拒绝斜向表现数据并回退为位移方向
- `moving`：明确人物正在行走或已经停止；旧客户端缺少该字段时由服务端按整数格变化兼容推导
- `input`：可选的四方向移动输入意图；停止时为零向量。字段用于后续服务端权威位移计算，当前服务端仍保留旧坐标同步行为以兼容旧客户端
- `client_tick`：可选的客户端单调时钟毫秒数，只用于延迟诊断，不参与服务端权威计时或速度计算

### 2013 ENTITY_MOVE_PUSH

服务端持久化或确认移动后，仅向同场景其他玩家推送。`to_pos` 继续作为整数权威位置，`precise_pos`、`facing`、`moving` 用于远端人物与宠物的实时表现。

```json
{
  "scene_id": 1,
  "scene_version": 1,
  "entity_id": 10001,
  "move_seq": 18,
  "from_pos": {"x": 10, "y": 7},
  "to_pos": {"x": 10, "y": 7},
  "precise_pos": {"x": 10250, "y": 7000},
  "facing": {"x": 1, "y": 0},
  "moving": true,
  "speed": 0,
  "server_tick": 0
}
```

### 2022 MOVE_INTENT_RESP

```json
{
  "accepted": true,
  "move_seq": 1,
  "scene_id": 2,
  "corrected_pos": {
    "x": 4,
    "y": 1
  },
  "corrected_precise_pos": {
    "x": 4000,
    "y": 1000
  },
  "server_tick": 0,
  "speed": 0,
  "reason": ""
}
```

说明：

- `scene_id`：服务端确认后的目标地图
- `corrected_pos`：服务端最终采用并持久化的整数场景坐标；普通门使用服务端拓扑目标点，快速传送使用数据库中心点。切图后的 `WORLD_RESYNC_PUSH.self_pos` 与该权威位置保持一致，客户端不得覆盖
- `corrected_precise_pos`：服务端确认的千分之一场景格权威表现坐标；协议字段已预留，权威移动服务接入前零值表示尚未提供
- `server_tick`：服务端移动时间基线；协议字段已预留，快照插值接入前零值表示尚未提供
- `speed`：服务端确认的权威移动速度，单位为“千分之一场景格/秒”；例如 `3750` 表示每秒 `3.75` 个场景格。未装配权威移动状态的兼容测试链路可能返回零值
- 如果 `target_scene_id` 为空、为 `0`、或等于当前 `scene_id` 且不是快速传送，服务端按同场景移动同步处理
- 如果带了 `portal_id`，服务端只用于校验来源场景、门和目标地图是否匹配；跨场景请求即使由旧客户端携带 `target_pos` 也会忽略
- 服务端会读取 `world_scene_definition.required_level`，并使用玩家数据库档案中的权威 `level` 校验目标地图准入等级；客户端请求不携带、也不能覆盖玩家等级
- 等级不足时返回 `accepted=false`、原 `scene_id` / 原坐标，以及 `reason="前面的路以后再来探索吧"`；服务端不会更新玩家持久化场景与坐标，客户端只展示该提示

### 2014 WORLD_RESYNC_PUSH

跨地图切换时，服务端优先发送不含在线玩家资料的基础快照，使客户端立即完成地图加载和解除黑屏；在线玩家随后通过 `2011 ENTITY_ENTER_PUSH` 增量补齐。NPC、地图实体和暗雷配置仍包含在基础快照中。

`self_pos` 是登录、重连、普通门和世界地图快速传送完成后摆放本地人物的唯一权威落点；客户端只负责坐标换算和显示。

```json
{
  "scene_id": 2,
  "self_pos": {
    "x": 4,
    "y": 1
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
  ],
  "wild_encounter": {
    "enabled": false,
    "scene_id": 2,
    "encounter_rate": 0,
    "spawn_monster_ids": [],
    "targets": []
  }
}
```

### 2044 SCENE_TRIGGER_PUSH

服务端在 `ENTER_WORLD_RESP` 或 `WORLD_RESYNC_PUSH` 之后，按玩家个人剧情进度判断是否需要播放一次性场景剧情。客户端只负责播放 `client_animation_key` 对应的本地剧情场景，不能自行决定是否已完成。

```json
{
  "trigger_code": "first_enter_east_road_taozi",
  "scene_id": 2,
  "client_animation_key": "初见桃子",
  "block_movement": true
}
```

- `trigger_code`：服务端剧情触发器唯一编码，用于后续 Ack。
- `client_animation_key`：客户端本地剧情场景 Key，按现有 `CinematicRegistry` 规则解析。
- `block_movement`：客户端播放期间应锁住玩家移动和运行时菜单。

### 2045 SCENE_TRIGGER_ACK_REQ

客户端播放完服务端触发的场景剧情后发送。服务端收到后写入 `player_story_flag`，执行 NPC 解锁、任务接取等权威副作用，并随后推送最新世界快照和任务更新。

```json
{
  "trigger_code": "first_enter_east_road_taozi"
}
```

### 2046 SCENE_TRIGGER_ACK_RESP

```json
{
  "accepted": true,
  "reason": "scene trigger completed",
  "trigger_code": "first_enter_east_road_taozi"
}
```

### 2035 WILD_ENCOUNTER_REQ

客户端在本地判定暗雷触发后上报，服务端校验 scene 与冷却后权威开战；如果该地图配置了多个暗雷怪物编队，服务端会按各编队权重随机抽取一个编队：

```json
{
  "scene_id": 4,
  "move_seq": 128,
  "self_pos": { "x": 12, "y": 34 }
}
```

`self_pos` 为客户端当前场景坐标（与 `ENTER_WORLD_RESP.self_pos` 同坐标系）。服务端在开战前会写回玩家档案并作为 `return_pos`，避免场景内移动后战斗结束坐标回跳。

### 2036 WILD_ENCOUNTER_RESP

```json
{
  "accepted": true,
  "reason": "battle started"
}
```

成功后服务端还会推送 `4011 BATTLE_START_PUSH`，后续战斗交互与 NPC 固定战一致。

### 2031 INTERACT_REQ

当前用于**无 NPC 菜单**的附近实体交互，例如直接触发 PvE 固定战：

```json
{
  "entity_id": 90001,
  "self_pos": { "x": 12, "y": 34 }
}
```

`self_pos` 含义同上，用于固定战开战前同步 return_pos。

若目标 NPC 配置了菜单项，服务端会拒绝本次交互并提示改用 `2042 NPC_MENU_REQ`：

```json
{
  "accepted": false,
  "reason": "use npc menu request",
  "entity_id": 93001,
  "npc_name": "市场理萌"
}
```

### 2032 INTERACT_RESP

战斗交互成功时返回：

```json
{
  "accepted": true,
  "reason": "battle started",
  "response_type": "battle",
  "entity_id": 90001,
  "npc_name": "引导NPC"
}
```

### 2042 NPC_MENU_REQ

拉取指定 NPC 的动态菜单列表；当前主要保留给任务状态变化后需要立即刷新单个 NPC 的兼容流程，地图首次菜单准备使用 `2047 NPC_MENU_BATCH_REQ`：

```json
{
  "entity_id": 93001
}
```

### 2043 NPC_MENU_RESP

```json
{
  "accepted": true,
  "reason": "menu loaded",
  "entity_id": 93001,
  "npc_name": "市场理萌",
  "menu_entries": [
    {
      "entry_id": "dialog_market_intro",
      "entry_type": "dialog",
      "title": "让个路",
      "subtitle": "看看市场理萌的轻剧情演出",
      "state": "available",
      "priority": 90
    }
  ]
}
```

### 2047 NPC_MENU_BATCH_REQ

客户端进入地图后异步请求当前场景全部可见 NPC 菜单。该请求不参与地图转场完成判定，也不能锁定玩家移动：

```json
{
  "scene_id": 3
}
```

### 2048 NPC_MENU_BATCH_RESP

服务端校验玩家权威场景后，一次读取场景实体、任务摘要和全部 NPC 静态菜单，再返回按 NPC 分组的结果：

```json
{
  "accepted": true,
  "reason": "scene npc menus loaded",
  "scene_id": 3,
  "menus": [
    {
      "accepted": true,
      "reason": "menu loaded",
      "entity_id": 93001,
      "npc_name": "市场理萌",
      "menu_entries": [
        {
          "entry_id": "dialog_market_intro",
          "entry_type": "dialog",
          "title": "让个路",
          "subtitle": "看看市场理萌的轻剧情演出",
          "state": "available",
          "priority": 90
        }
      ]
    }
  ]
}
```

响应到达时若 `scene_id` 已不是客户端当前场景，客户端必须丢弃该批缓存，避免旧地图迟到数据污染新地图。

### 2033 NPC_ACTION_REQ

执行某个 NPC 菜单项：

```json
{
  "entity_id": 93001,
  "entry_id": "dialog_market_intro",
  "self_pos": { "x": 12, "y": 34 }
}
```

触发 NPC 固定战时，`self_pos` 同样用于开战前同步 `return_pos`。

### 2034 NPC_ACTION_RESP

当菜单项 `action_result_type=dialogue` 时，会在同一条响应里直接返回首个剧情节点：

```json
{
  "accepted": true,
  "reason": "dialogue started",
  "entity_id": 93001,
  "entry_id": "dialog_market_intro",
  "result_type": "dialogue",
  "npc_name": "市场理萌",
  "dialogue": {
    "dialogue_id": 1,
    "node_id": "start",
    "node_type": "line",
    "speaker": "市场理萌",
    "content": "你先稍等一下，我把前面的货箱挪开。",
    "content_format": "plain",
    "portrait_key": "npc_limeng_normal",
    "client_animation_key": "",
    "client_animation_block": false,
    "options": [],
    "mentioned_items": [
      {
        "item_id": 1001,
        "item_name": "新手药水",
        "icon": "res://asset/icons/items/potion.png"
      }
    ],
    "is_end": false,
    "effect_notice": ""
  }
}
```

剧情节点 `content` 支持服务端渲染占位符：

- `{player_name}`：替换为当前玩家名。
- `{player_id}`：替换为当前玩家 ID。
- `{item:物品ID}`：替换为 `item_definition.item_name`，并在 `mentioned_items` 中返回物品 ID、名称与 icon，客户端用于展示物品图标。

剧情节点 `speaker` / `portrait_key` 由服务端在下发前统一解析：

- 后台可配置 `@player`、`$player`、`玩家` 或 `{player_name}` 表示玩家说话。
- 服务端会把上述占位符替换成当前玩家展示名，并在 `is_player_speaker=true` 时默认补齐 `portrait_key=player_default`。
- 客户端应优先读取 `is_player_speaker` 决定角标显示在右上（玩家）还是左上（NPC）。

其它结果类型：

- `result_type=notice`：仅返回 `notice` 文案
- `result_type=shop`：返回 `shop.goods` 商品列表与 `shop.wallet` 钱包快照
- `result_type=battle`：返回战斗开始，并推送 `4011 BATTLE_START_PUSH`

`shop` 载荷示例：

```json
{
  "accepted": true,
  "reason": "shop opened",
  "entity_id": 93002,
  "entry_id": "shop_open_market",
  "result_type": "shop",
  "npc_name": "市场罗格",
  "shop": {
    "goods": [
      { "item_id": 3003, "item_name": "宠物治疗药剂", "price_copper": 800 }
    ],
    "wallet": {
      "total_copper": 1200,
      "gold": 0,
      "silver": 12,
      "copper": 0
    }
  }
}
```

### 2037 NPC_DIALOGUE_NEXT_REQ

客户端点击“继续”或本地剧情动画结束后，请求推进到当前节点的下一节点：

```json
{
  "entity_id": 93001,
  "dialogue_id": 1,
  "node_id": "start"
}
```

### 2038 NPC_DIALOGUE_RESP

```json
{
  "accepted": true,
  "reason": "dialogue advanced",
  "entity_id": 93001,
  "node": {
    "dialogue_id": 1,
    "node_id": "move_aside",
    "node_type": "action",
    "client_animation_key": "market_limeng_step_aside",
    "client_animation_block": true,
    "is_end": false
  }
}
```

### 2039 NPC_DIALOGUE_CHOOSE_REQ

```json
{
  "entity_id": 93001,
  "dialogue_id": 1,
  "node_id": "after_move",
  "option_id": "news"
}
```

说明：

- 剧情节点与会话位置由服务端权威维护，客户端必须原样回传 `dialogue_id/node_id`
- `node_type` 支持 `line/choice/action/end`
- 节点 `effects_json.notice` 会通过 `effect_notice` 字段同步给客户端展示
- `action.client_animation_key` 与客户端 `res://scenes/cinematics/{key}.tscn` 或 `res://剧情动画/{key}.tscn` 文件名对应；客户端会按目录顺序自动解析同名场景，不维护硬编码注册表
- 阻塞动作场景可以复用 `common/world_player_cinematic.tscn`，按统一场景坐标驱动真实玩家沿导航路径移动、设置朝向并播放指定动画帧；演出发出 `finished` 后客户端才发送 `2037`
- 固定过场允许在动画 Key 对应的客户端脚本内写死 Tween、角色动画和多句展示对白；这些本地对白的继续操作只推进客户端演出，不发送 `2037`，整段脚本最终发出 `finished` 后才发送一次 `2037`

### 3001 PET_LIST_REQ

```json
{}
```

### 3002 PET_LIST_RESP

`PET_LIST_RESP` 返回宠物状态面板左侧列表与主界面宠物 HUD 共用的轻量摘要，不触发技能、资质等完整详情加载。客户端打开宠物状态面板时，应先请求本接口拿列表，再对默认选中的第一只宠物发送 `3035 PET_SKILL_DETAIL_REQ` 拉取完整属性；点击切换宠物时同样按目标 `pet_uid` 再请求详情。

```json
{
  "pets": [
    {
      "pet_uid": 20001,
      "pet_id": 101,
      "pet_name": "小火龙",
      "custom_name": "",
      "name": "小火龙",
      "skin_id": "嫩叶犬_001",
      "level": 5,
      "exp": 120,
      "exp_to_next": 380,
      "quality": 1,
      "hp": 32,
      "hp_max": 32,
      "mana": 16,
      "skill_ids": [],
      "in_lineup": true,
      "is_usable": true
    }
  ],
  "lineup": [
    {
      "pet_uid": 20001,
      "pet_id": 101,
      "level": 5,
      "hp": 32,
      "hp_max": 32,
      "skin_id": "嫩叶犬_001"
    }
  ]
}
```

说明：

- `pets` 返回玩家拥有的宠物摘要，供列表、名称、外观、等级、出战标记，以及主界面生命、法力、经验 HUD 展示使用
- `lineup` 返回当前编队摘要和顺序
- `custom_name` 是玩家自定义名；为空时客户端展示 `pet_name`。`name` 为服务端计算后的最终展示名，兼容旧客户端直接读取
- `skin_id` 来自 `pet_definition.skin_id`，客户端用于加载宠物槽“下待机”第一帧，不允许客户端按 `pet_id` 硬编码推断
- `exp` 是当前等级已获得经验，`exp_to_next` 是距离下一级还差的经验；客户端经验条总值按 `exp + exp_to_next` 展示
- 完整战斗属性、资质、技能槽、法宝槽等详情不在列表下发，必须按单只宠物使用 `3035/3036` 获取

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

### 3031 PET_ARTIFACT_EQUIP_REQ / 3032 PET_ARTIFACT_EQUIP_RESP

从背包装备法宝到宠物法宝槽（消耗 1 个物品）。

```json
{
  "pet_uid": 20001,
  "slot_index": 0,
  "container_type": "bag",
  "bag_slot_index": 3
}
```

物品模板要求：`effect_type=pet_artifact`，`effect_params.skill_id` 或 `effect_value` 为镶嵌技能 ID。响应 `pet` 含完整 `skill_slots.artifact`。

### 3033 PET_ARTIFACT_UNEQUIP_REQ / 3034 PET_ARTIFACT_UNEQUIP_RESP

```json
{
  "pet_uid": 20001,
  "slot_index": 0
}
```

### 3035 PET_SKILL_DETAIL_REQ / 3036 PET_SKILL_DETAIL_RESP

拉取单只宠物完整属性、资质、`skill_slots`（含法宝槽真实 `skill_id`）。列表接口 `3002` 只返回摘要，不返回完整详情。

```json
{
  "pet_uid": 20001
}
```

响应 `pet` 结构与 `PetDetail` 一致，包含基础属性、成长资质、状态抗性、技能槽和法宝槽完整技能 ID。每个非空技能槽会尽量附带服务端技能模板中的 `skill_name`、支持 BBCode 的 `description`、`skill_visual_id` 和 `skill_quality`。客户端用前两个字段渲染技能说明，按 `skill_visual_id` 查找本地 `SkillVisualConfig.tres` 图标，并按 `skill_quality` 切换按钮边框；服务端不保存或下发 `res://` 客户端资源路径。历史技能的 `skill_visual_id` 为空时，运行时暂以数据库 `skill_code` 作为兼容标识，显式配置的 `skill_visual_id` 始终优先。

`skill_quality` 支持 `normal`（普通）、`divine`（神技）、`soul`（魂技）、`sacred`（圣技）、`peerless`（绝世）。品质只控制客户端展示，不参与技能公式、释放方式或被动判定。

### 5021 神符槽道具解锁

物品 `effect_type=pet_talisman_slot_unlock`，并在 `pet_skill_slot_unlock_item` 配置 `item_id → slot_key`；请求需带 `target_pet_uid`。成功时 `result.unlocked_talisman_slot` 返回槽位键，并推送 `3011`。

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
  "entity_id": 91001,
  "from_slot_index": 4,
  "quantity": 10
}
```

说明：

- 当前落地实现会校验玩家当前场景附近是否存在可交互的仓库 NPC
- `entity_id` 建议传入本次操作对应的仓库 NPC，便于服务端做更严格的交互校验

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
  "entity_id": 91001,
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

### 5081 WALLET_QUERY_REQ

```json
{}
```

### 5082 WALLET_QUERY_RESP

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

### 5091 WALLET_UPDATE_PUSH

```json
{
  "wallet": {
    "total_copper": 2350678,
    "gold": 2,
    "silver": 350,
    "copper": 678
  },
  "reason_type": "battle_reward",
  "reason_ref_id": 9000001
}
```

### 5101 BUY_ITEM_REQ（最小已实现）

```json
{
  "shop_id": 1,
  "goods_id": 10001,
  "item_id": 1001,
  "quantity": 5
}
```

- 当前最小实现里，`shop_id` 直接表示附近商店 NPC 的 `entity_id`
- 服务端会校验玩家当前场景附近是否存在该 NPC，且 NPC 菜单中至少有一个 `entry_type=shop`
- 当前正式购买仅支持 `price_type=base_coin` 的启用物品模板，`goods_id` 先与 `item_id` 等价消费；若传 `0`，服务端会按 `item_id` 回填响应

### 5102 BUY_ITEM_RESP（最小已实现）

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

- 成功后当前连接还会继续收到：
  - `5011 BAG_UPDATE_PUSH`：刷新背包最新格子状态
  - `5091 WALLET_UPDATE_PUSH`：刷新钱包余额
- 当前最小实现暂未补入独立商店货架表，因此购买价格直接来自 `item_definition.buy_price_copper`
- 若背包加物成功但钱包扣减失败，服务端会回滚本次加到背包里的购买道具，避免出现“白拿道具”

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

### 5121 DROP_ITEM_REQ（已实现）

玩家主动丢弃背包/仓库容器内物品。服务端以当前 WebSocket 会话的 `player_id` 为准，不接受客户端传玩家 ID。

```json
{
  "container_type": "bag",
  "slot_index": 3,
  "quantity": 2,
  "item_uid": "eq_10001"
}
```

- `container_type` 当前支持 `bag` / `warehouse`
- `slot_index` 为服务端权威格子编号；非实例化可堆叠物品按 `slot_index + quantity` 支持部分丢弃
- `item_uid` 可选；实例化物品（装备等）应传该字段，服务端会按 `player_id + container_type + item_uid` 定位唯一条目
- 实例化物品必须整件丢弃，即 `quantity` 必须等于该格数量；已穿戴实例不可丢弃
- 服务端会校验 `item_definition.can_drop=true`，不可丢弃物品返回错误

### 5122 DROP_ITEM_RESP（已实现）

```json
{
  "container_type": "bag",
  "slot_index": 3,
  "item_uid": "eq_10001",
  "item_id": 2002,
  "item_name": "训练护腕",
  "dropped_quantity": 1
}
```

- 成功后服务端会在事务内扣减或删除 `player_container_item`，实例化装备还会同步删除 `equipment_instance`
- 服务端会写入 `item_change_log`，`change_type` 为 `drop_reduce` 或 `drop_remove`，`reason_type=item_drop`
- 成功响应后当前连接还会继续收到 `5011 BAG_UPDATE_PUSH`；分页背包 UI 应等待最新 `5002 BAG_LIST_RESP` 或完整容器快照落地后再关闭 loading，避免先展示旧数据
- 典型错误：空格子返回 `item slot is empty`；数量非法返回 `invalid drop item request`；不可丢弃或已穿戴返回 `item cannot be dropped`

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
- `action_type=5`：兼容旧客户端的自动战斗开关请求；服务端只返回 `accepted=true`，不保存自动状态，也不推进回合
- `action_type=6`：PVE 战斗捕捉，需携带 `actor_id`（玩家侧单位）、`target_id`（敌方 actor_id）、`item_id`（捕捉球道具 ID）、`bag_slot_index`（背包格子）
- `auto_battle_enabled`：仅为兼容字段；当前自动战斗开关由客户端本地维护，服务端不读取该值参与结算

捕捉请求示例：

```json
{
  "op_id": 3,
  "battle_id": 70001,
  "round": 1,
  "action_type": 6,
  "actor_id": 10001,
  "target_id": 900011,
  "item_id": 2001,
  "bag_slot_index": 2
}
```

捕捉规则（服务端权威）：

- 仅 PVE 战斗可用；PVP 返回 `accepted=false`
- 目标怪物必须在 `monster_definition.is_capturable=1` 且配置了 `capture_pet_id`
- 目标当前生命百分比需 `<= capture_min_hp_pct`
- `item_id` 必须出现在怪物模板的 `capture_item_ids` 中
- 捕捉成功：结束战斗，不发放常规击败金币/经验/掉落；按 `capture_pet_id` 发放系统宠物并 roll 野外资质
- 捕捉失败：扣除 1 个捕捉道具，战斗继续，推送 `4012 BATTLE_STATE_PUSH` 内含 `event_type=12` 捕捉事件
- 捕捉成功响应：`4002` 的 `capture_success=true`；随后 `4013 BATTLE_RESULT_PUSH` 携带 `capture_success`、`capture_monster_id`、`captured_pet_id`、`captured_pet_uid`；并推送 `3011 PET_UPDATE_PUSH`

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

当前回合流程：

- 客户端进入 `phase=command` 后，在本地维护操作倒计时与自动战斗开关
- 玩家需要为 `pending_actor_ids` 中本回合可控单位都选择动作；客户端收齐后逐条发送 `4001`
- 服务端收到前 N-1 个动作时只返回 `4002 accepted=true`，不推送中间战斗状态
- 服务端收到当前回合最后一个待行动单位意图后，结算本回合并推送 `4012 BATTLE_STATE_PUSH`；若战斗结束则继续推送 `4013`
- 本地倒计时归零时，客户端自行生成默认自动战斗意图并提交；服务端不再有回合超时补行动逻辑
- 只有一个玩家参与的战斗在玩家掉线后立即按失败结算，`reward_gold` / `reward_player_exp` / `rewards` / `drop_texts` 均为空或 `0`

### 4002 BATTLE_ACTION_RESP

```json
{
  "accepted": true,
  "reason": "capture success",
  "capture_success": true
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
      "skin_id": "决斗者_001",
      "hp": 120,
      "hp_max": 120,
      "atk": 24,
      "def": 12,
      "spd": 18,
      "skills": [
        {"skill_id": 1101, "name": "裂空斩", "target_type": "enemy_single", "target_count": 1, "animation_key": "slash", "skill_visual_id": "", "cast_color": "#8FD6FF", "impact_color": "#BDE9FF", "projectile": true},
        {"skill_id": 1001, "name": "普通攻击", "target_type": "enemy_single", "target_count": 1, "animation_key": "slash", "skill_visual_id": "", "cast_color": "#EBEBF5", "impact_color": "#FFF2F2", "projectile": false}
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
      "skin_id": "嫩叶犬_001",
      "hp": 32,
      "hp_max": 32,
      "atk": 14,
      "def": 10,
      "spd": 12,
      "skills": [
        {"skill_id": 1001, "name": "普通攻击", "target_type": "enemy_single", "target_count": 1, "animation_key": "slash", "skill_visual_id": "", "cast_color": "#EBEBF5", "impact_color": "#FFF2F2", "projectile": false},
        {"skill_id": 1002, "name": "火花冲击", "target_type": "enemy_all", "target_count": 0, "animation_key": "burst", "skill_visual_id": "", "cast_color": "#FFAA5C", "impact_color": "#FFD46B", "projectile": true}
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
      "skin_id": "史莱姆_001",
      "hp": 22,
      "hp_max": 22,
      "atk": 12,
      "def": 9,
      "spd": 8,
      "skills": [
        {"skill_id": 90001, "name": "野性撞击", "target_type": "enemy_single", "target_count": 1, "animation_key": "slash", "skill_visual_id": "", "cast_color": "#FFB88F", "impact_color": "#FFDDD1", "projectile": false},
        {"skill_id": 90002, "name": "利爪突袭", "target_type": "enemy_single", "target_count": 1, "animation_key": "volley", "skill_visual_id": "", "cast_color": "#FF9E85", "impact_color": "#FFC7BA", "projectile": true}
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
  "command_deadline_ms": 0,
  "auto_battle_enabled": false,
  "pending_actor_ids": [20001, 20002],
  "controllable_actor_ids": [20001, 20002]
}
```

说明：

- `skill_ids` 仅表示当前角色可提交的技能意图列表
- `skills` 为 `skill_ids` 的增强版快照，额外携带技能展示名、目标规则与表现参数，客户端应优先使用它来决定按钮文案、友/敌方目标选择和技能动画播放
- `unit_class` 用来区分当前战斗单位是真人角色、宠物还是怪物；当前约定 `1=人物`、`2=宠物`、`4=怪物`
- 当前已使用的 `target_type` 包括：`enemy_single`、`ally_single`、`enemy_all`、`enemy_multi`
- `target_count` 表示技能配置的目标数量；当前单体技能通常为 `1`，`enemy_all` 可忽略该字段，`enemy_multi` 表示客户端先指定一个主目标，剩余目标数量由服务端按 `target_count` 自动补足
- `animation_key` / `cast_color` / `impact_color` / `projectile` 为技能表现字段；当前仅用于客户端战斗表现层，真实命中、伤害、治疗与目标选择仍完全以服务端结算结果为准
- `phase=command` 表示当前轮到客户端继续为己方单位收集动作
- `command_deadline_ms` 当前固定为 `0`；回合操作倒计时由客户端本地维护
- `auto_battle_enabled` 当前固定为 `false`；自动战斗与取消自动战斗由客户端本地维护
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
  "command_deadline_ms": 0,
  "auto_battle_enabled": false,
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
  "player_gold": 20,
  "player_exp": 28,
  "drop_texts": [
    "掉落: 野性毛皮 x1"
  ],
  "pet_rewards": [
    {
      "pet_uid": 20001,
      "pet_id": 1001,
      "level": 6,
      "exp": 28,
      "exp_gained": 28,
      "level_up_count": 1,
      "attr_points_gained": 1,
      "free_attr_points": 4,
      "exp_to_next": 800
    },
    {
      "pet_uid": 20002,
      "exp": 28,
      "exp_gained": 28
    }
  ]
}
```

说明：

- `reward_gold` / `reward_player_exp` 表示本场战斗实际发放的铜币与角色经验；失败或逃跑时通常为 `0`
- `reward_gold` 会按 `1 铜币 = 1` 写入 `player_wallet.currency_copper_total`；钱包推送里的 `gold/silver/copper` 只是总铜币的展示拆分
- 后台遭遇战奖励支持 `gold` / `silver` / `copper` 三种配置单位，服务端结算时统一换算进 `reward_gold` 铜币总量；历史怪物奖励里的 `gold` 会迁移为 `copper`
- PVE 胜利的最终奖励 = 本次暗雷遭遇战固定奖励 + 随机到的编队中每个怪物自身奖励；怪物名和编队名不会影响奖励归属
- 怪物奖励与遭遇战奖励每条都有 `drop_rate` 万分比掉落率，`10000` 表示必掉；暗雷编队槽位可用 `reward_enabled=false` 让该怪物只参战但不贡献怪物自身奖励
- `rewards` 弹窗摘要里 `type="attr"` 时会携带 `attr_key` 与 `value`，表示本场战斗已直接写入玩家档案的属性加成
- `player_gold` / `player_exp` 表示服务端发奖后的玩家当前累计值，客户端可直接用来刷新本地摘要；其中 `player_gold` 是从钱包快照映射出的兼容字段
- 成功发放货币后，当前连接还会额外收到 `5091 WALLET_UPDATE_PUSH`，用于刷新 HUD / 背包面板里的钱包展示
- `drop_texts` 表示本场战斗的文本掉落提示，当前只用于展示，不会写入背包
- `pet_rewards` 表示本场战斗中各参战宠物的经验与升级摘要；`exp` 与 `exp_gained` 同值，保留兼容
- 宠物完整 HP / 等级 / 战斗属性仍以战斗结束后 `request_pet_list()` 或 `3011 PET_UPDATE_PUSH` 为准
- 当前最小 PVE 奖励闭环已覆盖金币、角色经验、宠物经验、文本掉落展示和 `battle_record` 防重记录；真实物品掉落与背包落库仍待后续扩展
