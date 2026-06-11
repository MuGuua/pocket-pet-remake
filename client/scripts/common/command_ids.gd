class_name CommandIds
extends RefCounted

# WebSocket 鉴权请求消息号。
const WS_AUTH_REQ: int = 1001
# WebSocket 鉴权响应消息号。
const WS_AUTH_RESP: int = 1002
# 心跳请求消息号。
const HEARTBEAT_REQ: int = 1003
# 心跳响应消息号。
const HEARTBEAT_RESP: int = 1004
# 服务端强制下线推送消息号。
const FORCE_OFFLINE_PUSH: int = 1011
# 服务端错误推送消息号。
const ERROR_PUSH: int = 1012
# 断线重连请求消息号。
const RECONNECT_REQ: int = 1021
# 断线重连响应消息号。
const RECONNECT_RESP: int = 1022

# 进入世界请求消息号。
const ENTER_WORLD_REQ: int = 2001
# 进入世界响应消息号。
const ENTER_WORLD_RESP: int = 2002
# 实体进入附近视野推送消息号。
const ENTITY_ENTER_PUSH: int = 2011
# 实体离开附近视野推送消息号。
const ENTITY_LEAVE_PUSH: int = 2012
# 实体移动推送消息号。
const ENTITY_MOVE_PUSH: int = 2013
# 世界重同步推送消息号。
const WORLD_RESYNC_PUSH: int = 2014
# 世界移动或切图意图请求消息号。
const MOVE_INTENT_REQ: int = 2021
# 世界移动或切图意图响应消息号。
const MOVE_INTENT_RESP: int = 2022
# 世界交互请求消息号。
const INTERACT_REQ: int = 2031
# 世界交互响应消息号。
const INTERACT_RESP: int = 2032
# NPC 菜单项执行请求消息号。
const NPC_ACTION_REQ: int = 2033
# NPC 菜单项执行响应消息号。
const NPC_ACTION_RESP: int = 2034
# 遭遇战触发推送消息号。
const ENCOUNTER_PUSH: int = 2041

# 宠物列表请求消息号。
const PET_LIST_REQ: int = 3001
# 宠物列表响应消息号。
const PET_LIST_RESP: int = 3002
# 宠物实例更新推送消息号。
const PET_UPDATE_PUSH: int = 3011
# 编队设置请求消息号。
const PET_LINEUP_SET_REQ: int = 3021
# 编队设置响应消息号。
const PET_LINEUP_SET_RESP: int = 3022

# 战斗动作请求消息号。
const BATTLE_ACTION_REQ: int = 4001
# 战斗动作响应消息号。
const BATTLE_ACTION_RESP: int = 4002
# 战斗开始推送消息号。
const BATTLE_START_PUSH: int = 4011
# 战斗状态推送消息号。
const BATTLE_STATE_PUSH: int = 4012
# 战斗结果推送消息号。
const BATTLE_RESULT_PUSH: int = 4013
# 战斗退出请求消息号。
const BATTLE_EXIT_REQ: int = 4021
# 战斗退出响应消息号。
const BATTLE_EXIT_RESP: int = 4022
# PVP 挑战请求消息号。
const PVP_CHALLENGE_REQ: int = 4031
# PVP 挑战响应消息号。
const PVP_CHALLENGE_RESP: int = 4032
# PVP 挑战邀请推送消息号。
const PVP_CHALLENGE_PUSH: int = 4033
# PVP 挑战应答请求消息号。
const PVP_CHALLENGE_REPLY_REQ: int = 4034
# PVP 挑战应答响应消息号。
const PVP_CHALLENGE_REPLY_RESP: int = 4035

# 背包列表请求消息号。
const BAG_LIST_REQ: int = 5001
# 背包列表响应消息号。
const BAG_LIST_RESP: int = 5002
# 背包物品更新推送消息号。
const BAG_UPDATE_PUSH: int = 5011
# 使用物品请求消息号。
const USE_ITEM_REQ: int = 5021
# 使用物品响应消息号。
const USE_ITEM_RESP: int = 5022

# 任务列表请求消息号。
const QUEST_LIST_REQ: int = 6001
# 任务列表响应消息号。
const QUEST_LIST_RESP: int = 6002
# 任务增量更新推送消息号。
const QUEST_UPDATE_PUSH: int = 6011
# 任务移除推送消息号。
const QUEST_REMOVE_PUSH: int = 6012
# 任务接取请求消息号。
const QUEST_ACCEPT_REQ: int = 6021
# 任务接取响应消息号。
const QUEST_ACCEPT_RESP: int = 6022
# 任务提交请求消息号。
const QUEST_SUBMIT_REQ: int = 6031
# 任务提交响应消息号。
const QUEST_SUBMIT_RESP: int = 6032
# 任务追踪请求消息号。
const QUEST_TRACK_REQ: int = 6041
# 任务追踪响应消息号。
const QUEST_TRACK_RESP: int = 6042

# 服务端公告推送消息号。
const NOTICE_PUSH: int = 9001
# 服务端踢下线推送消息号。
const KICKOUT_PUSH: int = 9002


# 返回指定消息号对应的稳定名称，方便统一打印服务端请求结果日志。
static func name_of(cmd: int) -> String:
	match cmd:
		WS_AUTH_REQ:
			return "WS_AUTH_REQ"
		WS_AUTH_RESP:
			return "WS_AUTH_RESP"
		HEARTBEAT_REQ:
			return "HEARTBEAT_REQ"
		HEARTBEAT_RESP:
			return "HEARTBEAT_RESP"
		FORCE_OFFLINE_PUSH:
			return "FORCE_OFFLINE_PUSH"
		ERROR_PUSH:
			return "ERROR_PUSH"
		RECONNECT_REQ:
			return "RECONNECT_REQ"
		RECONNECT_RESP:
			return "RECONNECT_RESP"
		ENTER_WORLD_REQ:
			return "ENTER_WORLD_REQ"
		ENTER_WORLD_RESP:
			return "ENTER_WORLD_RESP"
		ENTITY_ENTER_PUSH:
			return "ENTITY_ENTER_PUSH"
		ENTITY_LEAVE_PUSH:
			return "ENTITY_LEAVE_PUSH"
		ENTITY_MOVE_PUSH:
			return "ENTITY_MOVE_PUSH"
		WORLD_RESYNC_PUSH:
			return "WORLD_RESYNC_PUSH"
		MOVE_INTENT_REQ:
			return "MOVE_INTENT_REQ"
		MOVE_INTENT_RESP:
			return "MOVE_INTENT_RESP"
		INTERACT_REQ:
			return "INTERACT_REQ"
		INTERACT_RESP:
			return "INTERACT_RESP"
		NPC_ACTION_REQ:
			return "NPC_ACTION_REQ"
		NPC_ACTION_RESP:
			return "NPC_ACTION_RESP"
		ENCOUNTER_PUSH:
			return "ENCOUNTER_PUSH"
		PET_LIST_REQ:
			return "PET_LIST_REQ"
		PET_LIST_RESP:
			return "PET_LIST_RESP"
		PET_UPDATE_PUSH:
			return "PET_UPDATE_PUSH"
		PET_LINEUP_SET_REQ:
			return "PET_LINEUP_SET_REQ"
		PET_LINEUP_SET_RESP:
			return "PET_LINEUP_SET_RESP"
		BATTLE_ACTION_REQ:
			return "BATTLE_ACTION_REQ"
		BATTLE_ACTION_RESP:
			return "BATTLE_ACTION_RESP"
		BATTLE_START_PUSH:
			return "BATTLE_START_PUSH"
		BATTLE_STATE_PUSH:
			return "BATTLE_STATE_PUSH"
		BATTLE_RESULT_PUSH:
			return "BATTLE_RESULT_PUSH"
		BATTLE_EXIT_REQ:
			return "BATTLE_EXIT_REQ"
		BATTLE_EXIT_RESP:
			return "BATTLE_EXIT_RESP"
		PVP_CHALLENGE_REQ:
			return "PVP_CHALLENGE_REQ"
		PVP_CHALLENGE_RESP:
			return "PVP_CHALLENGE_RESP"
		PVP_CHALLENGE_PUSH:
			return "PVP_CHALLENGE_PUSH"
		PVP_CHALLENGE_REPLY_REQ:
			return "PVP_CHALLENGE_REPLY_REQ"
		PVP_CHALLENGE_REPLY_RESP:
			return "PVP_CHALLENGE_REPLY_RESP"
		BAG_LIST_REQ:
			return "BAG_LIST_REQ"
		BAG_LIST_RESP:
			return "BAG_LIST_RESP"
		BAG_UPDATE_PUSH:
			return "BAG_UPDATE_PUSH"
		USE_ITEM_REQ:
			return "USE_ITEM_REQ"
		USE_ITEM_RESP:
			return "USE_ITEM_RESP"
		QUEST_LIST_REQ:
			return "QUEST_LIST_REQ"
		QUEST_LIST_RESP:
			return "QUEST_LIST_RESP"
		QUEST_UPDATE_PUSH:
			return "QUEST_UPDATE_PUSH"
		QUEST_REMOVE_PUSH:
			return "QUEST_REMOVE_PUSH"
		QUEST_ACCEPT_REQ:
			return "QUEST_ACCEPT_REQ"
		QUEST_ACCEPT_RESP:
			return "QUEST_ACCEPT_RESP"
		QUEST_SUBMIT_REQ:
			return "QUEST_SUBMIT_REQ"
		QUEST_SUBMIT_RESP:
			return "QUEST_SUBMIT_RESP"
		QUEST_TRACK_REQ:
			return "QUEST_TRACK_REQ"
		QUEST_TRACK_RESP:
			return "QUEST_TRACK_RESP"
		NOTICE_PUSH:
			return "NOTICE_PUSH"
		KICKOUT_PUSH:
			return "KICKOUT_PUSH"
		_:
			return "CMD_%d" % cmd


# 判断当前消息是否值得打印到请求结果日志中；默认跳过高频心跳，避免刷屏。
static func should_log_result(cmd: int) -> bool:
	return cmd != HEARTBEAT_RESP
