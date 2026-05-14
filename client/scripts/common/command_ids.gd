class_name CommandIds
extends RefCounted

const WS_AUTH_REQ: int = 1001
const WS_AUTH_RESP: int = 1002
const HEARTBEAT_REQ: int = 1003
const HEARTBEAT_RESP: int = 1004
const FORCE_OFFLINE_PUSH: int = 1011
const ERROR_PUSH: int = 1012
const RECONNECT_REQ: int = 1021
const RECONNECT_RESP: int = 1022

const ENTER_WORLD_REQ: int = 2001
const ENTER_WORLD_RESP: int = 2002
const ENTITY_ENTER_PUSH: int = 2011
const ENTITY_LEAVE_PUSH: int = 2012
const ENTITY_MOVE_PUSH: int = 2013
const WORLD_RESYNC_PUSH: int = 2014
const MOVE_INTENT_REQ: int = 2021
const MOVE_INTENT_RESP: int = 2022
const INTERACT_REQ: int = 2031
const INTERACT_RESP: int = 2032
const ENCOUNTER_PUSH: int = 2041

const PET_LIST_REQ: int = 3001
const PET_LIST_RESP: int = 3002
const PET_UPDATE_PUSH: int = 3011
const PET_LINEUP_SET_REQ: int = 3021
const PET_LINEUP_SET_RESP: int = 3022

const BATTLE_ACTION_REQ: int = 4001
const BATTLE_ACTION_RESP: int = 4002
const BATTLE_START_PUSH: int = 4011
const BATTLE_STATE_PUSH: int = 4012
const BATTLE_RESULT_PUSH: int = 4013
const BATTLE_EXIT_REQ: int = 4021
const BATTLE_EXIT_RESP: int = 4022

const BAG_LIST_REQ: int = 5001
const BAG_LIST_RESP: int = 5002
const BAG_UPDATE_PUSH: int = 5011
const USE_ITEM_REQ: int = 5021
const USE_ITEM_RESP: int = 5022

const NOTICE_PUSH: int = 9001
const KICKOUT_PUSH: int = 9002
