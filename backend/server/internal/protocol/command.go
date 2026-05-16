package protocol

const (
	CmdWSAuthReq        uint16 = 1001
	CmdWSAuthResp       uint16 = 1002
	CmdHeartbeatReq     uint16 = 1003
	CmdHeartbeatResp    uint16 = 1004
	CmdForceOfflinePush uint16 = 1011
	CmdErrorPush        uint16 = 1012
	CmdReconnectReq     uint16 = 1021
	CmdReconnectResp    uint16 = 1022
	CmdEnterWorldReq    uint16 = 2001
	CmdEnterWorldResp   uint16 = 2002
	CmdEntityMovePush   uint16 = 2013
	CmdWorldResyncPush  uint16 = 2014
	CmdMoveIntentReq    uint16 = 2021
	CmdMoveIntentResp   uint16 = 2022
	CmdInteractReq      uint16 = 2031
	CmdInteractResp     uint16 = 2032
	CmdEncounterPush    uint16 = 2041
	CmdPetListReq       uint16 = 3001
	CmdPetListResp      uint16 = 3002
	CmdPetUpdatePush    uint16 = 3011
	CmdPetLineupSetReq  uint16 = 3021
	CmdPetLineupSetResp uint16 = 3022
	CmdBattleActionReq  uint16 = 4001
	CmdBattleActionResp uint16 = 4002
	CmdBattleStartPush  uint16 = 4011
	CmdBattleStatePush  uint16 = 4012
	CmdBattleResultPush uint16 = 4013
	CmdBattleExitReq    uint16 = 4021
	CmdBattleExitResp   uint16 = 4022
	CmdBagListReq       uint16 = 5001
	CmdBagListResp      uint16 = 5002
	CmdUseItemReq       uint16 = 5021
	CmdUseItemResp      uint16 = 5022
	CmdNoticePush       uint16 = 9001
	CmdKickOutPush      uint16 = 9002
)
