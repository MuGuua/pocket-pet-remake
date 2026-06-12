package wstransport

import (
	"math"

	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/protocol"
)

// toProtocolWalletSnapshot 把领域层钱包快照转换成协议结构。
// 背包、仓库、商城和战斗都复用同一份转换逻辑，避免不同入口拆分口径不一致。
func toProtocolWalletSnapshot(snapshot wallet.Snapshot) protocol.WalletSnapshot {
	return protocol.WalletSnapshot{
		TotalCopper: snapshot.TotalCopper,
		Gold:        snapshot.Gold,
		Silver:      snapshot.Silver,
		Copper:      snapshot.Copper,
	}
}

// pushWalletUpdatePacket 给当前连接主动推送钱包变更。
// 运行时所有正式货币变更都应该走这条推送，客户端才能在不重开面板的情况下同步展示最新余额。
func pushWalletUpdatePacket(conn packetSender, snapshot wallet.Snapshot, reasonType string, reasonRefID uint64) error {
	if conn == nil {
		return nil
	}
	return conn.SendPacket(mustJSONPacket(protocol.CmdWalletUpdatePush, 0, protocol.WalletUpdatePush{
		Wallet:      toProtocolWalletSnapshot(snapshot),
		ReasonType:  reasonType,
		ReasonRefID: reasonRefID,
	}))
}

// legacyGoldFromWalletSnapshot 把新钱包体系里的金币拆分结果映射回旧协议字段。
// 旧世界快照和部分旧 UI 仍读取 uint32 gold，这里先做兼容转换，后续可以逐步删除旧字段。
func legacyGoldFromWalletSnapshot(snapshot *wallet.Snapshot) uint32 {
	if snapshot == nil {
		return 0
	}
	if snapshot.Gold > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(snapshot.Gold)
}
