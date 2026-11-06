package wstransport

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"pocket-pet-remake/server/internal/protocol"
)

// isBagEquipCmd 判断当前 WebSocket 命令是否属于背包/装备/钱包链路。
func isBagEquipCmd(cmd uint16) bool {
	switch cmd {
	case protocol.CmdBagListReq, protocol.CmdBagListResp, protocol.CmdBagUpdatePush,
		protocol.CmdUseItemReq, protocol.CmdUseItemResp,
		protocol.CmdDropItemReq, protocol.CmdDropItemResp,
		protocol.CmdBagToWarehouseReq, protocol.CmdBagToWarehouseResp,
		protocol.CmdWarehouseToBagReq, protocol.CmdWarehouseToBagResp,
		protocol.CmdWalletQueryReq, protocol.CmdWalletQueryResp,
		protocol.CmdBuyItemReq, protocol.CmdBuyItemResp,
		protocol.CmdSellItemReq, protocol.CmdSellItemResp,
		protocol.CmdPlayerEquipmentListReq, protocol.CmdPlayerEquipmentListResp,
		protocol.CmdPlayerEquipReq, protocol.CmdPlayerEquipResp,
		protocol.CmdPlayerUnequipReq, protocol.CmdPlayerUnequipResp,
		protocol.CmdPlayerEquipmentEnhanceReq, protocol.CmdPlayerEquipmentEnhanceResp,
		protocol.CmdPlayerEquipmentRepairReq, protocol.CmdPlayerEquipmentRepairResp:
		return true
	default:
		return false
	}
}

// bagEquipCmdName 把背包/装备命令号转成稳定名称，便于服务端日志检索。
func bagEquipCmdName(cmd uint16) string {
	switch cmd {
	case protocol.CmdBagListReq:
		return "BAG_LIST_REQ"
	case protocol.CmdBagListResp:
		return "BAG_LIST_RESP"
	case protocol.CmdBagUpdatePush:
		return "BAG_UPDATE_PUSH"
	case protocol.CmdUseItemReq:
		return "USE_ITEM_REQ"
	case protocol.CmdUseItemResp:
		return "USE_ITEM_RESP"
	case protocol.CmdDropItemReq:
		return "DROP_ITEM_REQ"
	case protocol.CmdDropItemResp:
		return "DROP_ITEM_RESP"
	case protocol.CmdBagToWarehouseReq:
		return "BAG_TO_WAREHOUSE_REQ"
	case protocol.CmdBagToWarehouseResp:
		return "BAG_TO_WAREHOUSE_RESP"
	case protocol.CmdWarehouseToBagReq:
		return "WAREHOUSE_TO_BAG_REQ"
	case protocol.CmdWarehouseToBagResp:
		return "WAREHOUSE_TO_BAG_RESP"
	case protocol.CmdWalletQueryReq:
		return "WALLET_QUERY_REQ"
	case protocol.CmdWalletQueryResp:
		return "WALLET_QUERY_RESP"
	case protocol.CmdBuyItemReq:
		return "BUY_ITEM_REQ"
	case protocol.CmdBuyItemResp:
		return "BUY_ITEM_RESP"
	case protocol.CmdSellItemReq:
		return "SELL_ITEM_REQ"
	case protocol.CmdSellItemResp:
		return "SELL_ITEM_RESP"
	case protocol.CmdPlayerEquipmentListReq:
		return "PLAYER_EQUIPMENT_LIST_REQ"
	case protocol.CmdPlayerEquipmentListResp:
		return "PLAYER_EQUIPMENT_LIST_RESP"
	case protocol.CmdPlayerEquipReq:
		return "PLAYER_EQUIP_REQ"
	case protocol.CmdPlayerEquipResp:
		return "PLAYER_EQUIP_RESP"
	case protocol.CmdPlayerUnequipReq:
		return "PLAYER_UNEQUIP_REQ"
	case protocol.CmdPlayerUnequipResp:
		return "PLAYER_UNEQUIP_RESP"
	case protocol.CmdPlayerEquipmentEnhanceReq:
		return "PLAYER_EQUIPMENT_ENHANCE_REQ"
	case protocol.CmdPlayerEquipmentEnhanceResp:
		return "PLAYER_EQUIPMENT_ENHANCE_RESP"
	case protocol.CmdPlayerEquipmentRepairReq:
		return "PLAYER_EQUIPMENT_REPAIR_REQ"
	case protocol.CmdPlayerEquipmentRepairResp:
		return "PLAYER_EQUIPMENT_REPAIR_RESP"
	default:
		return fmt.Sprintf("CMD_%d", cmd)
	}
}

// wsCmdName 把常见 WebSocket 命令号转成可读名称，供错误日志使用。
func wsCmdName(cmd uint16) string {
	if isBattleCmd(cmd) {
		return battleCmdName(cmd)
	}
	return bagEquipCmdName(cmd)
}

// logBagEquipPacket 输出背包/装备链路的完整 JSON 载荷，便于联调时对照双端字段。
func logBagEquipPacket(direction string, connID string, playerID uint64, cmd uint16, seq uint32, payload any) {
	if !isBagEquipCmd(cmd) {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf(
			"[bag-equip-%s] conn_id=%s player_id=%d cmd=%s(%d) seq=%d marshal_err=%v",
			direction,
			connID,
			playerID,
			bagEquipCmdName(cmd),
			cmd,
			seq,
			err,
		)
		return
	}
	log.Printf(
		"[bag-equip-%s] conn_id=%s player_id=%d cmd=%s(%d) seq=%d body=%s",
		direction,
		connID,
		playerID,
		bagEquipCmdName(cmd),
		cmd,
		seq,
		string(body),
	)
}

// 只观测主要查询命令，避免高频移动、广播和心跳产生逐包日志开销。
func isTimedWSRequest(cmd uint16) bool {
	switch cmd {
	case protocol.CmdBagListReq, protocol.CmdPetListReq, protocol.CmdPetSkillDetailReq,
		protocol.CmdPlayerProfileReq, protocol.CmdPlayerEquipmentListReq,
		protocol.CmdNPCMenuReq, protocol.CmdEnterWorldReq:
		return true
	default:
		return false
	}
}

// 仅输出超过 200 毫秒的查询；耗时包含同步业务、SQL、编码和发送，不是纯 SQL 耗时。
// 参数只含连接、命令、序号、包大小及单调耗时，不记录令牌或业务载荷。
func logSlowWSRequest(logger *log.Logger, connID string, cmd uint16, seq uint32, requestBytes int, elapsed time.Duration) {
	if elapsed < 200*time.Millisecond || !isTimedWSRequest(cmd) {
		return
	}
	logger.Printf("[ws-timing] conn_id=%s cmd=%d seq=%d request_bytes=%d handler_us=%d", connID, cmd, seq, requestBytes, elapsed.Microseconds())
}
