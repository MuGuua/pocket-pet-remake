package wstransport

import (
	"encoding/json"
	"log"

	"pocket-pet-remake/server/internal/protocol"
)

// isBattleCmd 判断当前 WebSocket 命令是否属于战斗链路，便于只打印相关请求/响应。
func isBattleCmd(cmd uint16) bool {
	switch cmd {
	case protocol.CmdInteractReq, protocol.CmdInteractResp,
		protocol.CmdNPCActionReq, protocol.CmdNPCActionResp,
		protocol.CmdNPCMenuReq, protocol.CmdNPCMenuResp,
		protocol.CmdNPCDialogueNextReq, protocol.CmdNPCDialogueResp, protocol.CmdNPCDialogueChooseReq,
		protocol.CmdWildEncounterReq, protocol.CmdWildEncounterResp,
		protocol.CmdBattleActionReq, protocol.CmdBattleActionResp,
		protocol.CmdBattleStartPush, protocol.CmdBattleStatePush, protocol.CmdBattleResultPush,
		protocol.CmdBattleExitReq, protocol.CmdBattleExitResp,
		protocol.CmdPVPChallengeReq, protocol.CmdPVPChallengeResp, protocol.CmdPVPChallengePush,
		protocol.CmdPVPChallengeReplyReq, protocol.CmdPVPChallengeReplyResp,
		protocol.CmdReconnectReq, protocol.CmdReconnectResp:
		return true
	default:
		return false
	}
}

// battleCmdName 把命令号转成稳定名称，方便在服务端日志里检索。
func battleCmdName(cmd uint16) string {
	switch cmd {
	case protocol.CmdInteractReq:
		return "INTERACT_REQ"
	case protocol.CmdInteractResp:
		return "INTERACT_RESP"
	case protocol.CmdNPCActionReq:
		return "NPC_ACTION_REQ"
	case protocol.CmdNPCActionResp:
		return "NPC_ACTION_RESP"
	case protocol.CmdNPCMenuReq:
		return "NPC_MENU_REQ"
	case protocol.CmdNPCMenuResp:
		return "NPC_MENU_RESP"
	case protocol.CmdNPCDialogueNextReq:
		return "NPC_DIALOGUE_NEXT_REQ"
	case protocol.CmdNPCDialogueResp:
		return "NPC_DIALOGUE_RESP"
	case protocol.CmdNPCDialogueChooseReq:
		return "NPC_DIALOGUE_CHOOSE_REQ"
	case protocol.CmdWildEncounterReq:
		return "WILD_ENCOUNTER_REQ"
	case protocol.CmdWildEncounterResp:
		return "WILD_ENCOUNTER_RESP"
	case protocol.CmdBattleActionReq:
		return "BATTLE_ACTION_REQ"
	case protocol.CmdBattleActionResp:
		return "BATTLE_ACTION_RESP"
	case protocol.CmdBattleStartPush:
		return "BATTLE_START_PUSH"
	case protocol.CmdBattleStatePush:
		return "BATTLE_STATE_PUSH"
	case protocol.CmdBattleResultPush:
		return "BATTLE_RESULT_PUSH"
	case protocol.CmdBattleExitReq:
		return "BATTLE_EXIT_REQ"
	case protocol.CmdBattleExitResp:
		return "BATTLE_EXIT_RESP"
	case protocol.CmdPVPChallengeReq:
		return "PVP_CHALLENGE_REQ"
	case protocol.CmdPVPChallengeResp:
		return "PVP_CHALLENGE_RESP"
	case protocol.CmdPVPChallengePush:
		return "PVP_CHALLENGE_PUSH"
	case protocol.CmdPVPChallengeReplyReq:
		return "PVP_CHALLENGE_REPLY_REQ"
	case protocol.CmdPVPChallengeReplyResp:
		return "PVP_CHALLENGE_REPLY_RESP"
	case protocol.CmdReconnectReq:
		return "RECONNECT_REQ"
	case protocol.CmdReconnectResp:
		return "RECONNECT_RESP"
	default:
		return "CMD_UNKNOWN"
	}
}

// logBattlePacket 输出战斗链路的完整 JSON 载荷，便于联调时对照双端字段。
func logBattlePacket(direction string, connID string, playerID uint64, cmd uint16, seq uint32, payload any) {
	if !isBattleCmd(cmd) {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf(
			"[battle-%s] conn_id=%s player_id=%d cmd=%s(%d) seq=%d marshal_err=%v",
			direction,
			connID,
			playerID,
			battleCmdName(cmd),
			cmd,
			seq,
			err,
		)
		return
	}
	log.Printf(
		"[battle-%s] conn_id=%s player_id=%d cmd=%s(%d) seq=%d body=%s",
		direction,
		connID,
		playerID,
		battleCmdName(cmd),
		cmd,
		seq,
		string(body),
	)
}
