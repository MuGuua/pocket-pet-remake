package wstransport

import (
	"context"
	"log"
	"time"

	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type Router struct {
	authHandler      *AuthHandler
	worldHandler     *WorldHandler
	petHandler       *PetHandler
	playerHandler    *PlayerHandler
	equipmentHandler *EquipmentHandler
	battleHandler    *BattleHandler
	bagHandler       *BagHandler
	questHandler     *QuestHandler
	sessionService   *session.Service
}

func NewRouter(authHandler *AuthHandler, worldHandler *WorldHandler, petHandler *PetHandler, playerHandler *PlayerHandler, equipmentHandler *EquipmentHandler, battleHandler *BattleHandler, bagHandler *BagHandler, questHandler *QuestHandler, sessionService *session.Service) *Router {
	return &Router{
		authHandler:      authHandler,
		worldHandler:     worldHandler,
		petHandler:       petHandler,
		playerHandler:    playerHandler,
		equipmentHandler: equipmentHandler,
		battleHandler:    battleHandler,
		bagHandler:       bagHandler,
		questHandler:     questHandler,
		sessionService:   sessionService,
	}
}

func (r *Router) Handle(conn packetSender, raw []byte) error {
	packet, err := protocol.DecodePacket(raw)
	if err != nil {
		return sendError(conn, 0, errcode.WSCodeInvalidPacket, "invalid packet")
	}

	switch packet.Cmd {
	case protocol.CmdWSAuthReq:
		return r.authHandler.Handle(conn, packet)
	case protocol.CmdHeartbeatReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		_, err := r.sessionService.Touch(conn.ID())
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
		}
		responsePacket, err := protocol.NewJSONPacket(protocol.CmdHeartbeatResp, packet.Seq, errcode.WSCodeSuccess, protocol.HeartbeatResp{
			ServerTimeMS: time.Now().UnixMilli(),
		})
		if err != nil {
			return err
		}
		if err := conn.SendPacket(responsePacket); err != nil {
			return err
		}
		if r.battleHandler != nil {
			return r.battleHandler.HandleBattleHeartbeat(conn)
		}
		return nil
	case protocol.CmdEnterWorldReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.worldHandler.HandleEnterWorld(conn, packet)
	case protocol.CmdReconnectReq:
		if r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "connection already authenticated")
		}
		return r.handleReconnect(conn, packet)
	case protocol.CmdMoveIntentReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.worldHandler.HandleMoveIntent(conn, packet)
	case protocol.CmdSceneTriggerAckReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.worldHandler.HandleSceneTriggerAck(conn, packet)
	case protocol.CmdInteractReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleInteract(conn, packet)
	case protocol.CmdNPCActionReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleNPCAction(conn, packet)
	case protocol.CmdNPCMenuReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleNPCMenu(conn, packet)
	case protocol.CmdNPCDialogueNextReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleNPCDialogueNext(conn, packet)
	case protocol.CmdNPCDialogueChooseReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleNPCDialogueChoose(conn, packet)
	case protocol.CmdPetListReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.petHandler.HandlePetList(conn, packet)
	case protocol.CmdPetLineupSetReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.petHandler.HandleLineupSet(conn, packet)
	case protocol.CmdPetAllocateAttrReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.petHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "pet handler unavailable")
		}
		return r.petHandler.HandleAllocateAttr(conn, packet)
	case protocol.CmdPetArtifactEquipReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.petHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "pet handler unavailable")
		}
		return r.petHandler.HandleArtifactEquip(conn, packet)
	case protocol.CmdPetArtifactUnequipReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.petHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "pet handler unavailable")
		}
		return r.petHandler.HandleArtifactUnequip(conn, packet)
	case protocol.CmdPetSkillDetailReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.petHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "pet handler unavailable")
		}
		return r.petHandler.HandleSkillDetail(conn, packet)
	case protocol.CmdBagListReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleBagList(conn, packet)
	case protocol.CmdUseItemReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleUseItem(conn, packet)
	case protocol.CmdDropItemReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleDropItem(conn, packet)
	case protocol.CmdContainerListReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleContainerList(conn, packet)
	case protocol.CmdBagToWarehouseReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleBagToWarehouse(conn, packet)
	case protocol.CmdWarehouseToBagReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleWarehouseToBag(conn, packet)
	case protocol.CmdContainerSortReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleContainerSort(conn, packet)
	case protocol.CmdContainerMoveReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleContainerMove(conn, packet)
	case protocol.CmdWalletQueryReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleWalletQuery(conn, packet)
	case protocol.CmdBuyItemReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.bagHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "bag handler unavailable")
		}
		return r.bagHandler.HandleBuyItem(conn, packet)
	case protocol.CmdWildEncounterReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleWildEncounter(conn, packet)
	case protocol.CmdPlayerAllocateAttrReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.playerHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "player handler unavailable")
		}
		return r.playerHandler.HandleAllocateAttr(conn, packet)
	case protocol.CmdPlayerEquipmentListReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.equipmentHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "equipment handler unavailable")
		}
		return r.equipmentHandler.HandleList(conn, packet)
	case protocol.CmdPlayerEquipReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.equipmentHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "equipment handler unavailable")
		}
		return r.equipmentHandler.HandleEquip(conn, packet)
	case protocol.CmdPlayerUnequipReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.equipmentHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "equipment handler unavailable")
		}
		return r.equipmentHandler.HandleUnequip(conn, packet)
	case protocol.CmdPlayerEquipmentEnhanceReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.equipmentHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "equipment handler unavailable")
		}
		return r.equipmentHandler.HandleEnhance(conn, packet)
	case protocol.CmdPlayerEquipmentRepairReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		if r.equipmentHandler == nil {
			return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "equipment handler unavailable")
		}
		return r.equipmentHandler.HandleRepair(conn, packet)
	case protocol.CmdBattleActionReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandleBattleAction(conn, packet)
	case protocol.CmdPVPChallengeReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandlePVPChallenge(conn, packet)
	case protocol.CmdPVPChallengeReplyReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.battleHandler.HandlePVPChallengeReply(conn, packet)
	case protocol.CmdQuestListReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestList(conn, packet)
	case protocol.CmdQuestAcceptReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestAccept(conn, packet)
	case protocol.CmdQuestSubmitReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestSubmit(conn, packet)
	case protocol.CmdQuestTrackReq:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return r.questHandler.HandleQuestTrack(conn, packet)
	default:
		if !r.sessionService.IsAuthenticated(conn.ID()) {
			return sendError(conn, packet.Seq, errcode.WSCodeUnauthorized, "unauthorized")
		}
		return sendError(conn, packet.Seq, errcode.WSCodeUnsupportedCmd, "unsupported command")
	}
}

func (r *Router) handleReconnect(conn packetSender, packet *protocol.Packet) error {
	var request protocol.ReconnectReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid reconnect body")
	}
	logBattlePacket("req", conn.ID(), 0, protocol.CmdReconnectReq, packet.Seq, request)
	sess, err := r.sessionService.Reconnect(request.ReconnectToken, conn)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "reconnect token invalid")
	}

	var worldSnapshot *protocol.EnterWorldResp
	if r.worldHandler != nil {
		worldSnapshot, err = r.worldHandler.BuildWorldSnapshotForPlayer(context.Background(), sess.PlayerID)
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load reconnect world snapshot failed")
		}
	}
	var battleStart *protocol.BattleStartPush
	var battleState *protocol.BattleStatePush
	var battleResult *protocol.BattleResultPush
	var battleReplayStates []protocol.BattleStatePush
	if r.battleHandler != nil {
		battleStart, battleState, battleResult, battleReplayStates, err = r.battleHandler.BuildReconnectSnapshot(context.Background(), sess.PlayerID, request.BattleID, request.LastFrame)
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "load reconnect battle snapshot failed")
		}
	}
	var activeDialogue *protocol.ActiveDialogue
	if r.battleHandler != nil {
		activeDialogue, err = r.battleHandler.BuildActiveDialogueReconnect(context.Background(), sess.PlayerID)
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load reconnect dialogue snapshot failed")
		}
	}

	reconnectResp := protocol.ReconnectResp{
		PlayerID:           sess.PlayerID,
		SessionID:          sess.ID,
		ReconnectToken:     sess.ReconnectToken,
		HeartbeatSec:       uint32(r.sessionService.HeartbeatInterval() / time.Second),
		ServerTimeMS:       time.Now().UnixMilli(),
		World:              worldSnapshot,
		BattleStart:        battleStart,
		BattleState:        battleState,
		BattleResult:       battleResult,
		BattleReplayStates: battleReplayStates,
		ActiveDialogue:     activeDialogue,
	}
	logBattlePacket("resp", conn.ID(), sess.PlayerID, protocol.CmdReconnectResp, packet.Seq, reconnectResp)
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdReconnectResp, packet.Seq, errcode.WSCodeSuccess, reconnectResp)
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	if r.worldHandler != nil && worldSnapshot != nil {
		r.worldHandler.enterPlayerScene(context.Background(), sess.PlayerID, worldSnapshot.SceneID)
	}
	return nil
}

func sendError(conn packetSender, seq uint32, code uint32, message string, causes ...error) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdErrorPush, seq, code, protocol.ErrorPush{
		Code: code,
		Msg:  message,
	})
	if err != nil {
		log.Printf("[ws-error] conn_id=%s seq=%d code=%d msg=%s packet_err=%v", conn.ID(), seq, code, message, err)
		return err
	}
	if len(causes) > 0 && causes[0] != nil {
		log.Printf("[ws-error] conn_id=%s seq=%d code=%d msg=%s cause=%v", conn.ID(), seq, code, message, causes[0])
	} else {
		log.Printf("[ws-error] conn_id=%s seq=%d code=%d msg=%s", conn.ID(), seq, code, message)
	}
	return conn.SendPacket(packet)
}
