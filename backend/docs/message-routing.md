# 双端消息路由与处理流程

## 服务端路由表

| cmd | 消息名 | 入口层 | 业务模块 | 说明 |
| --- | --- | --- | --- | --- |
| 1001 | WS_AUTH_REQ | ws/auth_handler | auth + session | 校验 `ws_token`，绑定连接 |
| 1003 | HEARTBEAT_REQ | ws/router | session | 更新心跳时间并回包 |
| 1021 | RECONNECT_REQ | ws/router | session + world + battle | 校验重连令牌并返回重同步结果 |
| 2001 | ENTER_WORLD_REQ | ws/world_handler | player + pet + world | 加载玩家快照并进入场景 |
| 2021 | MOVE_INTENT_REQ | ws/world_handler | world | 校验移动并广播 |
| 2031 | INTERACT_REQ | ws/world_handler | world / bag / battle | NPC、传送点、遭遇触发 |
| 3001 | PET_LIST_REQ | ws/pet_handler | pet | 返回宠物列表 |
| 3021 | PET_LINEUP_SET_REQ | ws/pet_handler | pet + player | 设置编队 |
| 4001 | BATTLE_ACTION_REQ | ws/battle_handler | battle + pet + bag | 回合行动受理 |
| 4021 | BATTLE_EXIT_REQ | ws/battle_handler | battle | 非结算场景退出 |
| 5001 | BAG_LIST_REQ | ws/bag_handler | bag | 返回背包列表 |
| 5021 | USE_ITEM_REQ | ws/bag_handler | bag + pet + player | 世界内道具使用 |

## 服务端推送表

| cmd | 消息名 | 来源模块 | 目标 |
| --- | --- | --- | --- |
| 1002 | WS_AUTH_RESP | auth + session | 当前连接 |
| 1004 | HEARTBEAT_RESP | session | 当前连接 |
| 1011 | FORCE_OFFLINE_PUSH | session | 旧连接 |
| 1012 | ERROR_PUSH | 任意模块 | 当前连接 |
| 2002 | ENTER_WORLD_RESP | world | 当前连接 |
| 2011 | ENTITY_ENTER_PUSH | world | AOI 内其他玩家 |
| 2012 | ENTITY_LEAVE_PUSH | world | AOI 内其他玩家 |
| 2013 | ENTITY_MOVE_PUSH | world | AOI 内所有相关玩家 |
| 2014 | WORLD_RESYNC_PUSH | world | 当前位置失真玩家 |
| 2041 | ENCOUNTER_PUSH | world | 当前连接 |
| 3011 | PET_UPDATE_PUSH | pet | 当前连接 |
| 4011 | BATTLE_START_PUSH | battle | 参战连接 |
| 4012 | BATTLE_STATE_PUSH | battle | 参战连接 |
| 4013 | BATTLE_RESULT_PUSH | battle | 参战连接 |
| 5011 | BAG_UPDATE_PUSH | bag | 当前连接 |
| 9001 | NOTICE_PUSH | system | 当前连接或广播 |
| 9002 | KICKOUT_PUSH | session / admin | 当前连接 |

## 客户端路由表

| cmd | 接收模块 | 后续动作 |
| --- | --- | --- |
| 1002 | `auth_service.gd` | 保存 `session_id`、`reconnect_token`、心跳间隔 |
| 1004 | `NetClient.gd` | 更新时间差与连接状态 |
| 1011 | `App.gd` | 提示并回登录界面 |
| 1012 | `MessageRouter.gd` | 分发错误提示 |
| 1022 | `App.gd` | 成功则触发世界/战斗重同步 |
| 2002 | `world_controller.gd` | 初始化地图和附近实体 |
| 2011 | `world_controller.gd` | 创建进入视野实体 |
| 2012 | `world_controller.gd` | 删除离开视野实体 |
| 2013 | `world_controller.gd` | 目标实体插值移动 |
| 2014 | `world_controller.gd` | 强制重置本地位置与 AOI 缓存 |
| 2041 | `world_controller.gd` | 切换战斗加载状态 |
| 3002 | `pet_controller.gd` | 刷新宠物列表 |
| 3011 | `pet_controller.gd` | 刷新变化宠物并通知 UI |
| 3022 | `pet_controller.gd` | 更新编队数据 |
| 4011 | `battle_controller.gd` | 创建战斗场景与初始状态 |
| 4012 | `battle_controller.gd` | 依序播放战斗事件并刷新状态 |
| 4013 | `battle_controller.gd` | 结算展示并返回世界 |
| 5002 | `bag_controller.gd` | 刷新背包列表 |
| 5011 | `bag_controller.gd` | 局部刷新道具数量 |
| 9001 | `App.gd` | 系统通知展示 |
| 9002 | `App.gd` | 弹窗后断开连接 |

## 4 条关键流程

### 1. 登录与 WebSocket 鉴权
1. 客户端 `POST /api/v1/auth/login`
2. `auth` 校验账号并返回 `access_jwt` 与 `ws_token`
3. 客户端建立 WSS 连接
4. 客户端发送 `WS_AUTH_REQ`
5. `auth + session` 校验令牌并绑定连接
6. 服务端返回 `WS_AUTH_RESP`
7. 会话进入心跳态

### 2. 进入世界
1. 客户端发送 `ENTER_WORLD_REQ`
2. `player` 加载主数据
3. `pet` 加载编队
4. `world` 计算场景与附近实体
5. 服务端返回 `ENTER_WORLD_RESP`
6. `world` 向 AOI 内其他玩家广播 `ENTITY_ENTER_PUSH`

### 3. 世界移动
1. 客户端点击目标点并发送 `MOVE_INTENT_REQ`
2. `world` 校验场景、状态、目标点可达性
3. 服务端回 `MOVE_INTENT_RESP`
4. `world` 广播 `ENTITY_MOVE_PUSH`
5. 客户端按服务端速度插值
6. 若偏差过大，服务端下发 `WORLD_RESYNC_PUSH`

### 4. 战斗结算
1. `world` 触发遭遇并创建 `battle_id`
2. 服务端推送 `ENCOUNTER_PUSH` 和 `BATTLE_START_PUSH`
3. 客户端发送 `BATTLE_ACTION_REQ`
4. `battle` 校验并受理行动
5. 服务端回 `BATTLE_ACTION_RESP`
6. `battle` 执行状态机并推送 `BATTLE_STATE_PUSH`
7. 战斗结束后推送 `BATTLE_RESULT_PUSH`
8. `pet`、`bag`、`player` 完成落库与状态刷新
