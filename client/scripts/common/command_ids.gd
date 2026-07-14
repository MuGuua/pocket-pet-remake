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
# NPC 菜单拉取请求消息号。
const NPC_MENU_REQ: int = 2042
# NPC 菜单拉取响应消息号。
const NPC_MENU_RESP: int = 2043
# 场景剧情触发推送消息号。
const SCENE_TRIGGER_PUSH: int = 2044
# 场景剧情播放完成确认请求消息号。
const SCENE_TRIGGER_ACK_REQ: int = 2045
# 场景剧情播放完成确认响应消息号。
const SCENE_TRIGGER_ACK_RESP: int = 2046
# NPC 剧情继续请求消息号。
const NPC_DIALOGUE_NEXT_REQ: int = 2037
# NPC 剧情节点响应消息号。
const NPC_DIALOGUE_RESP: int = 2038
# NPC 剧情选项选择请求消息号。
const NPC_DIALOGUE_CHOOSE_REQ: int = 2039
# 暗雷遭遇上报请求消息号。
const WILD_ENCOUNTER_REQ: int = 2035
# 暗雷遭遇上报响应消息号。
const WILD_ENCOUNTER_RESP: int = 2036
# 遭遇战触发推送消息号。
const ENCOUNTER_PUSH: int = 2041
# 玩家分配属性点请求消息号。
const PLAYER_ALLOCATE_ATTR_REQ: int = 2061
# 玩家分配属性点响应消息号。
const PLAYER_ALLOCATE_ATTR_RESP: int = 2062
# 宠物分配属性点请求消息号。
const PET_ALLOCATE_ATTR_REQ: int = 2063
# 宠物分配属性点响应消息号。
const PET_ALLOCATE_ATTR_RESP: int = 2064
# 人物已佩戴装备列表请求消息号。
const PLAYER_EQUIPMENT_LIST_REQ: int = 2070
# 人物已佩戴装备列表响应消息号。
const PLAYER_EQUIPMENT_LIST_RESP: int = 2071
# 人物佩戴装备请求消息号。
const PLAYER_EQUIP_REQ: int = 2072
# 人物佩戴装备响应消息号。
const PLAYER_EQUIP_RESP: int = 2073
# 人物卸下装备请求消息号。
const PLAYER_UNEQUIP_REQ: int = 2074
# 人物卸下装备响应消息号。
const PLAYER_UNEQUIP_RESP: int = 2075
# 人物装备强化请求消息号。
const PLAYER_EQUIPMENT_ENHANCE_REQ: int = 2076
# 人物装备强化响应消息号。
const PLAYER_EQUIPMENT_ENHANCE_RESP: int = 2077
# 人物装备修复请求消息号。
const PLAYER_EQUIPMENT_REPAIR_REQ: int = 2078
# 人物装备修复响应消息号。
const PLAYER_EQUIPMENT_REPAIR_RESP: int = 2079

# 宠物列表摘要请求消息号。
const PET_LIST_REQ: int = 3001
# 宠物列表摘要响应消息号。
const PET_LIST_RESP: int = 3002
# 宠物实例更新推送消息号。
const PET_UPDATE_PUSH: int = 3011
# 编队设置请求消息号。
const PET_LINEUP_SET_REQ: int = 3021
# 编队设置响应消息号。
const PET_LINEUP_SET_RESP: int = 3022
# 宠物法宝装备请求消息号。
const PET_ARTIFACT_EQUIP_REQ: int = 3031
# 宠物法宝装备响应消息号。
const PET_ARTIFACT_EQUIP_RESP: int = 3032
# 宠物法宝卸下请求消息号。
const PET_ARTIFACT_UNEQUIP_REQ: int = 3033
# 宠物法宝卸下响应消息号。
const PET_ARTIFACT_UNEQUIP_RESP: int = 3034
# 单只宠物完整属性和技能详情请求消息号（含法宝槽完整 skill_id）。
const PET_SKILL_DETAIL_REQ: int = 3035
# 单只宠物完整属性和技能详情响应消息号。
const PET_SKILL_DETAIL_RESP: int = 3036

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
# 丢弃物品请求消息号。
const DROP_ITEM_REQ: int = 5121
# 丢弃物品响应消息号。
const DROP_ITEM_RESP: int = 5122
# 容器列表请求消息号，可用于单独查询仓库等容器。
const CONTAINER_LIST_REQ: int = 5031
# 容器列表响应消息号。
const CONTAINER_LIST_RESP: int = 5032
# 背包存入仓库请求消息号。
const BAG_TO_WAREHOUSE_REQ: int = 5041
# 背包存入仓库响应消息号。
const BAG_TO_WAREHOUSE_RESP: int = 5042
# 仓库取回背包请求消息号。
const WAREHOUSE_TO_BAG_REQ: int = 5051
# 仓库取回背包响应消息号。
const WAREHOUSE_TO_BAG_RESP: int = 5052
# 钱包查询请求消息号。
const WALLET_QUERY_REQ: int = 5081
# 钱包查询响应消息号。
const WALLET_QUERY_RESP: int = 5082
# 钱包增量更新推送消息号。
const WALLET_UPDATE_PUSH: int = 5091
# 商店购买物品请求消息号。
const BUY_ITEM_REQ: int = 5101
# 商店购买物品响应消息号。
const BUY_ITEM_RESP: int = 5102

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
        NPC_MENU_REQ:
            return "NPC_MENU_REQ"
        NPC_MENU_RESP:
            return "NPC_MENU_RESP"
        NPC_DIALOGUE_NEXT_REQ:
            return "NPC_DIALOGUE_NEXT_REQ"
        NPC_DIALOGUE_RESP:
            return "NPC_DIALOGUE_RESP"
        NPC_DIALOGUE_CHOOSE_REQ:
            return "NPC_DIALOGUE_CHOOSE_REQ"
        WILD_ENCOUNTER_REQ:
            return "WILD_ENCOUNTER_REQ"
        WILD_ENCOUNTER_RESP:
            return "WILD_ENCOUNTER_RESP"
        ENCOUNTER_PUSH:
            return "ENCOUNTER_PUSH"
        PLAYER_ALLOCATE_ATTR_REQ:
            return "PLAYER_ALLOCATE_ATTR_REQ"
        PLAYER_ALLOCATE_ATTR_RESP:
            return "PLAYER_ALLOCATE_ATTR_RESP"
        PET_ALLOCATE_ATTR_REQ:
            return "PET_ALLOCATE_ATTR_REQ"
        PET_ALLOCATE_ATTR_RESP:
            return "PET_ALLOCATE_ATTR_RESP"
        PLAYER_EQUIPMENT_LIST_REQ:
            return "PLAYER_EQUIPMENT_LIST_REQ"
        PLAYER_EQUIPMENT_LIST_RESP:
            return "PLAYER_EQUIPMENT_LIST_RESP"
        PLAYER_EQUIP_REQ:
            return "PLAYER_EQUIP_REQ"
        PLAYER_EQUIP_RESP:
            return "PLAYER_EQUIP_RESP"
        PLAYER_UNEQUIP_REQ:
            return "PLAYER_UNEQUIP_REQ"
        PLAYER_UNEQUIP_RESP:
            return "PLAYER_UNEQUIP_RESP"
        PLAYER_EQUIPMENT_ENHANCE_REQ:
            return "PLAYER_EQUIPMENT_ENHANCE_REQ"
        PLAYER_EQUIPMENT_ENHANCE_RESP:
            return "PLAYER_EQUIPMENT_ENHANCE_RESP"
        PLAYER_EQUIPMENT_REPAIR_REQ:
            return "PLAYER_EQUIPMENT_REPAIR_REQ"
        PLAYER_EQUIPMENT_REPAIR_RESP:
            return "PLAYER_EQUIPMENT_REPAIR_RESP"
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
        PET_ARTIFACT_EQUIP_REQ:
            return "PET_ARTIFACT_EQUIP_REQ"
        PET_ARTIFACT_EQUIP_RESP:
            return "PET_ARTIFACT_EQUIP_RESP"
        PET_ARTIFACT_UNEQUIP_REQ:
            return "PET_ARTIFACT_UNEQUIP_REQ"
        PET_ARTIFACT_UNEQUIP_RESP:
            return "PET_ARTIFACT_UNEQUIP_RESP"
        PET_SKILL_DETAIL_REQ:
            return "PET_SKILL_DETAIL_REQ"
        PET_SKILL_DETAIL_RESP:
            return "PET_SKILL_DETAIL_RESP"
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
        DROP_ITEM_REQ:
            return "DROP_ITEM_REQ"
        DROP_ITEM_RESP:
            return "DROP_ITEM_RESP"
        CONTAINER_LIST_REQ:
            return "CONTAINER_LIST_REQ"
        CONTAINER_LIST_RESP:
            return "CONTAINER_LIST_RESP"
        BAG_TO_WAREHOUSE_REQ:
            return "BAG_TO_WAREHOUSE_REQ"
        BAG_TO_WAREHOUSE_RESP:
            return "BAG_TO_WAREHOUSE_RESP"
        WAREHOUSE_TO_BAG_REQ:
            return "WAREHOUSE_TO_BAG_REQ"
        WAREHOUSE_TO_BAG_RESP:
            return "WAREHOUSE_TO_BAG_RESP"
        WALLET_QUERY_REQ:
            return "WALLET_QUERY_REQ"
        WALLET_QUERY_RESP:
            return "WALLET_QUERY_RESP"
        WALLET_UPDATE_PUSH:
            return "WALLET_UPDATE_PUSH"
        BUY_ITEM_REQ:
            return "BUY_ITEM_REQ"
        BUY_ITEM_RESP:
            return "BUY_ITEM_RESP"
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


# 判断当前消息是否值得打印到请求结果日志中；跳过高频心跳与状态同步推送，避免 HUD 与控制台被刷爆。
static func should_log_result(cmd: int) -> bool:
    match cmd:
        HEARTBEAT_RESP, BATTLE_STATE_PUSH, ENTITY_MOVE_PUSH:
            return false
        _:
            return true


# 判断消息是否属于背包/装备/钱包链路，便于调试构建下输出完整 JSON。
static func is_bag_related(cmd: int) -> bool:
    match cmd:
        BAG_LIST_REQ, BAG_LIST_RESP, BAG_UPDATE_PUSH, \
        USE_ITEM_REQ, USE_ITEM_RESP, \
        DROP_ITEM_REQ, DROP_ITEM_RESP, \
        BAG_TO_WAREHOUSE_REQ, BAG_TO_WAREHOUSE_RESP, \
        WAREHOUSE_TO_BAG_REQ, WAREHOUSE_TO_BAG_RESP, \
        WALLET_QUERY_REQ, WALLET_QUERY_RESP, \
        BUY_ITEM_REQ, BUY_ITEM_RESP, \
        PLAYER_EQUIPMENT_LIST_REQ, PLAYER_EQUIPMENT_LIST_RESP, \
        PLAYER_EQUIP_REQ, PLAYER_EQUIP_RESP, \
        PLAYER_UNEQUIP_REQ, PLAYER_UNEQUIP_RESP, \
        PLAYER_EQUIPMENT_ENHANCE_REQ, PLAYER_EQUIPMENT_ENHANCE_RESP, \
        PLAYER_EQUIPMENT_REPAIR_REQ, PLAYER_EQUIPMENT_REPAIR_RESP:
            return true
        _:
            return false


# 判断消息是否属于战斗链路（开战入口、回合动作、状态推送、PVP 与重连恢复）。
static func is_battle_related(cmd: int) -> bool:
    match cmd:
        INTERACT_REQ, INTERACT_RESP, \
        NPC_ACTION_REQ, NPC_ACTION_RESP, \
        NPC_MENU_REQ, NPC_MENU_RESP, \
        NPC_DIALOGUE_NEXT_REQ, NPC_DIALOGUE_RESP, NPC_DIALOGUE_CHOOSE_REQ, \
        WILD_ENCOUNTER_REQ, WILD_ENCOUNTER_RESP, \
        BATTLE_ACTION_REQ, BATTLE_ACTION_RESP, \
        BATTLE_START_PUSH, BATTLE_STATE_PUSH, BATTLE_RESULT_PUSH, \
        BATTLE_EXIT_REQ, BATTLE_EXIT_RESP, \
        PVP_CHALLENGE_REQ, PVP_CHALLENGE_RESP, PVP_CHALLENGE_PUSH, \
        PVP_CHALLENGE_REPLY_REQ, PVP_CHALLENGE_REPLY_RESP, \
        RECONNECT_REQ, RECONNECT_RESP:
            return true
        _:
            return false
