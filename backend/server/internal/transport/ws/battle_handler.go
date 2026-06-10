package wstransport

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type BattleHandler struct {
	sessionService *session.Service
	playerService  *player.Service
	petService     *pet.Service
	worldService   *world.Service
	questService   *quest.Service
	npcService     *npc.Service
	battleService  *battle.Service
	battleRepo     battle.Repository
	reconnectMu    sync.Mutex
	reconnectCache map[uint64]protocol.BattleResultPush
}

func NewBattleHandler(sessionService *session.Service, playerService *player.Service, petService *pet.Service, worldService *world.Service, questService *quest.Service, npcService *npc.Service, battleService *battle.Service, battleRepo battle.Repository) *BattleHandler {
	return &BattleHandler{
		sessionService: sessionService,
		playerService:  playerService,
		petService:     petService,
		worldService:   worldService,
		questService:   questService,
		npcService:     npcService,
		battleService:  battleService,
		battleRepo:     battleRepo,
		reconnectCache: make(map[uint64]protocol.BattleResultPush),
	}
}

func (h *BattleHandler) HandleInteract(conn packetSender, packet *protocol.Packet) error {
	var request protocol.InteractReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid interact body")
	}

	sess, profile, lineup, sceneSnapshot, err := h.loadPlayerBattleContext(conn.ID())
	if err != nil {
		return h.handleContextError(conn, packet.Seq, err)
	}
	target, found := findInteractTarget(sceneSnapshot.NearbyEntities, request.EntityID)
	if !found {
		return h.sendInteractResponse(conn, packet.Seq, protocol.InteractResp{Accepted: false, Reason: "target unavailable"})
	}

	if menuResp, ok := h.buildNPCMenuResponse(context.Background(), sess.PlayerID, target); ok {
		menuResp.Accepted = true
		return h.sendInteractResponse(conn, packet.Seq, menuResp)
	}

	startSnapshot, err := h.battleService.StartPVE(context.Background(), profile, lineup, target)
	if err != nil {
		if errors.Is(err, battle.ErrBattleAlreadyActive) {
			return h.sendInteractResponse(conn, packet.Seq, protocol.InteractResp{Accepted: false, Reason: "battle already active"})
		}
		if errors.Is(err, battle.ErrNoLineupAvailable) {
			return h.sendInteractResponse(conn, packet.Seq, protocol.InteractResp{Accepted: false, Reason: "no lineup available"})
		}
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "battle start failed")
	}

	if err := h.sendInteractResponse(conn, packet.Seq, protocol.InteractResp{Accepted: true, Reason: "battle started", ResponseType: "battle", EntityID: target.EntityID, NPCName: target.Name}); err != nil {
		return err
	}
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleStartPush, 0, protocol.BattleStartPush{
		BattleID:             startSnapshot.BattleID,
		BattleType:           startSnapshot.BattleType,
		BattleVersion:        startSnapshot.BattleVersion,
		Frame:                startSnapshot.Frame,
		ParticipantPlayerIDs: append([]uint64{}, startSnapshot.ParticipantPlayerIDs...),
		Allies:               toProtocolBattleActors(startSnapshot.Allies),
		Enemies:              toProtocolBattleActors(startSnapshot.Enemies),
		Round:                startSnapshot.Round,
		Phase:                startSnapshot.Phase,
		ActiveActorID:        startSnapshot.ActiveActorID,
		ActivePetUID:         startSnapshot.ActivePetUID,
		CommandDeadlineMS:    startSnapshot.CommandDeadlineMS,
		AutoBattleEnabled:    startSnapshot.AutoBattleEnabled,
		PendingActorIDs:      append([]uint64{}, startSnapshot.PendingActorIDs...),
		ControllableActorIDs: append([]uint64{}, startSnapshot.ControllableActorIDs...),
	}))
}

func (h *BattleHandler) HandleBattleAction(conn packetSender, packet *protocol.Packet) error {
	var request protocol.BattleActionReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid battle action body")
	}

	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	outcome, err := h.battleService.SubmitAction(context.Background(), sess.PlayerID, battle.ActionRequest{
		BattleID:          request.BattleID,
		Round:             request.Round,
		ActionType:        request.ActionType,
		ActorID:           request.ActorID,
		SkillID:           request.SkillID,
		TargetID:          request.TargetID,
		AutoBattleEnabled: request.AutoBattleEnabled,
	})
	if err != nil {
		if errors.Is(err, battle.ErrBattleNotFound) {
			return h.sendBattleActionResponse(conn, packet.Seq, false, "battle not found")
		}
		if errors.Is(err, battle.ErrInvalidAction) {
			return h.sendBattleActionResponse(conn, packet.Seq, false, "invalid action")
		}
		return sendError(conn, packet.Seq, errcode.WSCodeBattleActionInvalid, "battle action failed")
	}

	if err := h.sendBattleActionResponse(conn, packet.Seq, outcome.Response.Accepted, outcome.Response.Reason); err != nil {
		return err
	}
	return h.pushBattleOutcome(context.Background(), conn, sess.PlayerID, outcome)
}

func (h *BattleHandler) HandlePVPChallenge(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PVPChallengeReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pvp challenge body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	targetSession, err := h.sessionService.GetByPlayerID(request.TargetPlayerID)
	if err != nil || targetSession.Conn == nil {
		return h.sendPVPChallengeResponse(conn, packet.Seq, false, "target player offline", 0, request.TargetPlayerID)
	}
	challenge, err := h.battleService.CreatePVPChallenge(context.Background(), sess.PlayerID, request.TargetPlayerID)
	if err != nil {
		if errors.Is(err, battle.ErrBattleAlreadyActive) {
			return h.sendPVPChallengeResponse(conn, packet.Seq, false, "player already in battle", 0, request.TargetPlayerID)
		}
		if errors.Is(err, battle.ErrChallengeInvalid) {
			return h.sendPVPChallengeResponse(conn, packet.Seq, false, "invalid challenge target", 0, request.TargetPlayerID)
		}
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "create pvp challenge failed")
	}
	challengerProfile, err := h.playerService.GetProfile(context.Background(), sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "challenger not found")
	}
	if err := h.sendPVPChallengeResponse(conn, packet.Seq, true, "challenge sent", challenge.ChallengeID, request.TargetPlayerID); err != nil {
		return err
	}
	if targetSession.Conn == nil {
		return nil
	}
	return targetSession.Conn.SendPacket(mustJSONPacket(protocol.CmdPVPChallengePush, 0, protocol.PVPChallengePush{
		ChallengeID: challenge.ChallengeID,
		Challenger: protocol.PlayerBrief{
			PlayerID: challengerProfile.PlayerID,
			Name:     challengerProfile.Name,
			Level:    challengerProfile.Level,
		},
		ExpiresAtMS: challenge.ExpiresAt.UnixMilli(),
	}))
}

func (h *BattleHandler) HandlePVPChallengeReply(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PVPChallengeReplyReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pvp challenge reply body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	challenge, err := h.battleService.ResolvePVPChallenge(context.Background(), request.ChallengeID, sess.PlayerID, request.Accept)
	if err != nil {
		switch {
		case errors.Is(err, battle.ErrChallengeNotFound):
			return h.sendPVPChallengeReplyResponse(conn, packet.Seq, false, "challenge not found", request.ChallengeID)
		case errors.Is(err, battle.ErrChallengeExpired):
			return h.sendPVPChallengeReplyResponse(conn, packet.Seq, false, "challenge expired", request.ChallengeID)
		case errors.Is(err, battle.ErrChallengeInvalid):
			return h.sendPVPChallengeReplyResponse(conn, packet.Seq, false, "challenge invalid", request.ChallengeID)
		case errors.Is(err, battle.ErrBattleAlreadyActive):
			return h.sendPVPChallengeReplyResponse(conn, packet.Seq, false, "player already in battle", request.ChallengeID)
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "resolve pvp challenge failed")
		}
	}
	if !request.Accept {
		if err := h.sendPVPChallengeReplyResponse(conn, packet.Seq, true, "challenge rejected", request.ChallengeID); err != nil {
			return err
		}
		return h.pushNoticeToPlayer(challenge.ChallengerPlayerID, "对方拒绝了 PVP 邀请。")
	}

	challengerProfile, err := h.playerService.GetProfile(context.Background(), challenge.ChallengerPlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "challenger not found")
	}
	challengerLineup, err := h.petService.ListLineup(context.Background(), challenge.ChallengerPlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "load challenger lineup failed")
	}
	defenderProfile, err := h.playerService.GetProfile(context.Background(), challenge.DefenderPlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "defender not found")
	}
	defenderLineup, err := h.petService.ListLineup(context.Background(), challenge.DefenderPlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "load defender lineup failed")
	}
	startSnapshot, err := h.battleService.StartPVP(context.Background(), challengerProfile, challengerLineup, defenderProfile, defenderLineup)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "start pvp battle failed")
	}
	if err := h.sendPVPChallengeReplyResponse(conn, packet.Seq, true, "challenge accepted", request.ChallengeID); err != nil {
		return err
	}
	return h.pushBattleStartToParticipants(startSnapshot)
}

func (h *BattleHandler) HandleBattleHeartbeat(conn packetSender) error {
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return nil
	}
	outcome, err := h.battleService.ProgressAuto(context.Background(), sess.PlayerID)
	if err != nil || outcome == nil {
		return err
	}
	return h.pushBattleOutcome(context.Background(), conn, sess.PlayerID, outcome)
}

// BuildReconnectSnapshot exposes the active authoritative battle in the same
// packet shapes the client already consumes during normal live battle flow.
func (h *BattleHandler) BuildReconnectSnapshot(ctx context.Context, playerID uint64, battleID uint64, lastFrame uint32) (*protocol.BattleStartPush, *protocol.BattleStatePush, *protocol.BattleResultPush, []protocol.BattleStatePush, error) {
	if h == nil || h.battleService == nil {
		return nil, nil, h.popReconnectResult(playerID), nil, nil
	}
	start, state, ok := h.battleService.GetActiveSnapshot(ctx, playerID)
	if !ok {
		return nil, nil, h.popReconnectResult(playerID), nil, nil
	}
	replaySnapshots := h.battleService.GetReplaySnapshots(ctx, playerID, battleID, lastFrame)
	replayStates := make([]protocol.BattleStatePush, 0, len(replaySnapshots))
	for _, replay := range replaySnapshots {
		replayStates = append(replayStates, protocol.BattleStatePush{
			BattleID:             replay.BattleID,
			BattleVersion:        replay.BattleVersion,
			Frame:                replay.Frame,
			ParticipantPlayerIDs: append([]uint64{}, replay.ParticipantPlayerIDs...),
			Round:                replay.Round,
			Phase:                replay.Phase,
			Events:               toProtocolBattleEvents(replay.Events),
			Actors:               toProtocolBattleActorStates(replay.Actors),
			ActiveActorID:        replay.ActiveActorID,
			ActivePetUID:         replay.ActivePetUID,
			CommandDeadlineMS:    replay.CommandDeadlineMS,
			AutoBattleEnabled:    replay.AutoBattleEnabled,
			PendingActorIDs:      append([]uint64{}, replay.PendingActorIDs...),
			ControllableActorIDs: append([]uint64{}, replay.ControllableActorIDs...),
		})
	}
	return &protocol.BattleStartPush{
			BattleID:             start.BattleID,
			BattleType:           start.BattleType,
			BattleVersion:        start.BattleVersion,
			Frame:                start.Frame,
			ParticipantPlayerIDs: append([]uint64{}, start.ParticipantPlayerIDs...),
			Allies:               toProtocolBattleActors(start.Allies),
			Enemies:              toProtocolBattleActors(start.Enemies),
			Round:                start.Round,
			Phase:                start.Phase,
			ActiveActorID:        start.ActiveActorID,
			ActivePetUID:         start.ActivePetUID,
			CommandDeadlineMS:    start.CommandDeadlineMS,
			AutoBattleEnabled:    start.AutoBattleEnabled,
			PendingActorIDs:      append([]uint64{}, start.PendingActorIDs...),
			ControllableActorIDs: append([]uint64{}, start.ControllableActorIDs...),
		}, &protocol.BattleStatePush{
			BattleID:             state.BattleID,
			BattleVersion:        state.BattleVersion,
			Frame:                state.Frame,
			ParticipantPlayerIDs: append([]uint64{}, state.ParticipantPlayerIDs...),
			Round:                state.Round,
			Phase:                state.Phase,
			Events:               toProtocolBattleEvents(state.Events),
			Actors:               toProtocolBattleActorStates(state.Actors),
			ActiveActorID:        state.ActiveActorID,
			ActivePetUID:         state.ActivePetUID,
			CommandDeadlineMS:    state.CommandDeadlineMS,
			AutoBattleEnabled:    state.AutoBattleEnabled,
			PendingActorIDs:      append([]uint64{}, state.PendingActorIDs...),
			ControllableActorIDs: append([]uint64{}, state.ControllableActorIDs...),
		}, nil, replayStates, nil
}

// HandleSessionDisconnect switches the player's active battle into server
// custody mode so later sweeps can keep resolving rounds even after the socket
// has gone away.
func (h *BattleHandler) HandleSessionDisconnect(playerID uint64) {
	if h == nil || h.battleService == nil || playerID == 0 {
		return
	}
	if result := h.battleService.ResolveDisconnect(context.Background(), playerID); result != nil {
		_ = h.pushBattleResultToParticipants(result, &battleSettlement{})
		return
	}
	h.battleService.EnableAutoForPlayer(context.Background(), playerID)
}

// StartCustodySweeper periodically progresses battles that have entered server
// custody mode, covering heartbeat loss and fully disconnected sessions.
func (h *BattleHandler) StartCustodySweeper(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = h.ProcessAutoCustodyOnce(ctx)
		}
	}
}

// ProcessAutoCustodyOnce performs one custody sweep so tests and the runtime
// loop can reuse the same auto progression path.
func (h *BattleHandler) ProcessAutoCustodyOnce(ctx context.Context) error {
	if h == nil || h.battleService == nil {
		return nil
	}
	for _, item := range h.battleService.ProgressAutoAll(ctx) {
		if err := h.deliverAutoOutcome(ctx, item.PlayerID, item.Outcome); err != nil {
			return err
		}
	}
	return nil
}

func (h *BattleHandler) pushBattleOutcome(ctx context.Context, conn packetSender, playerID uint64, outcome *battle.ActionOutcome) error {
	if outcome == nil {
		return nil
	}
	if outcome.State != nil && len(outcome.State.ParticipantPlayerIDs) > 1 {
		if err := h.pushBattleStateToParticipants(outcome.State); err != nil {
			return err
		}
	} else if err := h.pushBattleStatePacket(conn, outcome.State); err != nil {
		return err
	}
	if outcome.Result == nil {
		return nil
	}
	settlement, err := h.applyBattleResultSideEffects(ctx, conn, playerID, outcome.Result)
	if err != nil {
		return err
	}
	if len(outcome.Result.ParticipantPlayerIDs) > 1 {
		if err := h.pushBattleResultToParticipants(outcome.Result, settlement); err != nil {
			return err
		}
	} else {
		if err := h.pushBattleResultPacket(conn, outcome.Result, settlement); err != nil {
			return err
		}
	}
	return h.pushBattleSettlementFollowUps(ctx, conn, playerID, outcome.Result, settlement)
}

func (h *BattleHandler) deliverAutoOutcome(ctx context.Context, playerID uint64, outcome *battle.ActionOutcome) error {
	if outcome == nil {
		return nil
	}
	var conn packetSender
	if h.sessionService != nil {
		sess, err := h.sessionService.GetByPlayerID(playerID)
		if err == nil {
			conn = sess.Conn
		}
	}
	if conn != nil {
		if err := h.pushBattleStatePacket(conn, outcome.State); err != nil {
			return err
		}
	}
	settlement, err := h.applyBattleResultSideEffects(ctx, conn, playerID, outcome.Result)
	if err != nil {
		return err
	}
	if conn != nil && outcome.Result != nil {
		h.clearReconnectResult(playerID)
		if err := h.pushBattleResultPacket(conn, outcome.Result, settlement); err != nil {
			return err
		}
	} else if outcome.Result != nil {
		h.storeReconnectResult(playerID, h.buildBattleResultPayload(outcome.Result, settlement))
	}
	return h.pushBattleSettlementFollowUps(ctx, conn, playerID, outcome.Result, settlement)
}

func (h *BattleHandler) pushBattleStatePacket(conn packetSender, state *battle.StateSnapshot) error {
	if conn == nil || state == nil {
		return nil
	}
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleStatePush, 0, protocol.BattleStatePush{
		BattleID:             state.BattleID,
		BattleVersion:        state.BattleVersion,
		Frame:                state.Frame,
		ParticipantPlayerIDs: append([]uint64{}, state.ParticipantPlayerIDs...),
		Round:                state.Round,
		Phase:                state.Phase,
		Events:               toProtocolBattleEvents(state.Events),
		Actors:               toProtocolBattleActorStates(state.Actors),
		ActiveActorID:        state.ActiveActorID,
		ActivePetUID:         state.ActivePetUID,
		CommandDeadlineMS:    state.CommandDeadlineMS,
		AutoBattleEnabled:    state.AutoBattleEnabled,
		PendingActorIDs:      append([]uint64{}, state.PendingActorIDs...),
		ControllableActorIDs: append([]uint64{}, state.ControllableActorIDs...),
	}))
}

func (h *BattleHandler) pushBattleStateToParticipants(state *battle.StateSnapshot) error {
	if state == nil {
		return nil
	}
	for _, conn := range h.participantConns(state.ParticipantPlayerIDs) {
		if err := h.pushBattleStatePacket(conn, state); err != nil {
			return err
		}
	}
	return nil
}

type battleSettlement struct {
	PlayerProfile *player.Profile
	Pets          []pet.Pet
	questBefore   []quest.Summary
}

func (h *BattleHandler) pushBattleResultPacket(conn packetSender, result *battle.ResultSnapshot, settlement *battleSettlement) error {
	if conn == nil || result == nil {
		return nil
	}
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleResultPush, 0, h.buildBattleResultPayload(result, settlement)))
}

func (h *BattleHandler) pushBattleResultToParticipants(result *battle.ResultSnapshot, settlement *battleSettlement) error {
	if result == nil {
		return nil
	}
	payload := h.buildBattleResultPayload(result, settlement)
	for _, conn := range h.participantConns(result.ParticipantPlayerIDs) {
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdBattleResultPush, 0, payload)); err != nil {
			return err
		}
	}
	return nil
}

func (h *BattleHandler) buildBattleResultPayload(result *battle.ResultSnapshot, settlement *battleSettlement) protocol.BattleResultPush {
	playerGold := uint32(0)
	playerExp := uint64(0)
	petRewards := make([]protocol.BattlePetReward, 0, len(result.PetResults))
	if settlement != nil && settlement.PlayerProfile != nil {
		playerGold = settlement.PlayerProfile.Gold
		playerExp = settlement.PlayerProfile.Exp
	}
	for _, petResult := range result.PetResults {
		petRewards = append(petRewards, protocol.BattlePetReward{
			PetUID: petResult.PetUID,
			Exp:    petResult.ExpGained,
		})
	}
	return protocol.BattleResultPush{
		BattleID:      result.BattleID,
		Win:           result.Win,
		ReturnSceneID: result.ReturnSceneID,
		ReturnPos: protocol.Vec2i{
			X: result.ReturnPos.X,
			Y: result.ReturnPos.Y,
		},
		Reason:          result.Reason,
		RewardGold:      result.RewardGold,
		RewardPlayerExp: result.RewardPlayerExp,
		PlayerGold:      playerGold,
		PlayerExp:       playerExp,
		PetRewards:      petRewards,
		DropTexts:       append([]string{}, result.DropTexts...),
	}
}

func (h *BattleHandler) storeReconnectResult(playerID uint64, payload protocol.BattleResultPush) {
	if h == nil || playerID == 0 {
		return
	}
	h.reconnectMu.Lock()
	defer h.reconnectMu.Unlock()
	h.reconnectCache[playerID] = payload
}

func (h *BattleHandler) popReconnectResult(playerID uint64) *protocol.BattleResultPush {
	if h == nil || playerID == 0 {
		return nil
	}
	h.reconnectMu.Lock()
	defer h.reconnectMu.Unlock()
	payload, ok := h.reconnectCache[playerID]
	if !ok {
		return nil
	}
	delete(h.reconnectCache, playerID)
	copy := payload
	return &copy
}

func (h *BattleHandler) clearReconnectResult(playerID uint64) {
	if h == nil || playerID == 0 {
		return
	}
	h.reconnectMu.Lock()
	defer h.reconnectMu.Unlock()
	delete(h.reconnectCache, playerID)
}

func (h *BattleHandler) participantConns(playerIDs []uint64) []packetSender {
	if h == nil || h.sessionService == nil || len(playerIDs) == 0 {
		return nil
	}
	result := make([]packetSender, 0, len(playerIDs))
	seen := map[string]bool{}
	for _, playerID := range playerIDs {
		sess, err := h.sessionService.GetByPlayerID(playerID)
		if err != nil || sess.Conn == nil || seen[sess.Conn.ID()] {
			continue
		}
		seen[sess.Conn.ID()] = true
		result = append(result, sess.Conn)
	}
	return result
}

func (h *BattleHandler) pushNoticeToPlayer(playerID uint64, message string) error {
	for _, conn := range h.participantConns([]uint64{playerID}) {
		return conn.SendPacket(mustJSONPacket(protocol.CmdNoticePush, 0, map[string]any{
			"message": message,
		}))
	}
	return nil
}

func (h *BattleHandler) applyBattleResultSideEffects(ctx context.Context, _ packetSender, playerID uint64, result *battle.ResultSnapshot) (*battleSettlement, error) {
	if result == nil {
		return nil, nil
	}
	if result.BattleType != battle.BattleTypePVE {
		return &battleSettlement{}, nil
	}
	inserted, err := h.tryBeginBattleRewardGrant(ctx, playerID, result)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return nil, nil
	}
	var questBefore []quest.Summary
	if h.questService != nil && result.Win {
		questBefore, _ = listQuestSummaries(ctx, h.questService, playerID)
		_, _ = h.questService.HandleEvent(ctx, quest.Event{
			PlayerID:  playerID,
			EventType: "WIN_BATTLE",
			Count:     1,
			Meta: map[string]any{
				"battle_type": "PVE",
			},
		})
	}
	settlement := &battleSettlement{}
	if h.playerService != nil && (result.RewardGold > 0 || result.RewardPlayerExp > 0) {
		updatedProfile, err := h.playerService.AddGoldAndExp(ctx, playerID, result.RewardGold, result.RewardPlayerExp)
		if err != nil {
			return nil, err
		}
		settlement.PlayerProfile = updatedProfile
	}
	if h.petService != nil {
		for _, petResult := range result.PetResults {
			updatedPet, err := h.petService.UpdatePetBattleProgress(ctx, playerID, petResult.PetUID, petResult.HP, petResult.ExpGained)
			if err != nil {
				return nil, err
			}
			settlement.Pets = append(settlement.Pets, updatedPet)
		}
	}
	settlement.questBefore = questBefore
	return settlement, nil
}

func (h *BattleHandler) pushBattleSettlementFollowUps(ctx context.Context, conn packetSender, playerID uint64, result *battle.ResultSnapshot, settlement *battleSettlement) error {
	if conn == nil || result == nil || settlement == nil || result.BattleType != battle.BattleTypePVE {
		return nil
	}
	for _, updatedPet := range settlement.Pets {
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
			Pet: toProtocolPetDetail(updatedPet),
		})); err != nil {
			return err
		}
	}
	if h.questService != nil && result.Win {
		_ = pushQuestDiff(ctx, conn, h.questService, playerID, settlement.questBefore)
	}
	return nil
}

func (h *BattleHandler) tryBeginBattleRewardGrant(ctx context.Context, playerID uint64, result *battle.ResultSnapshot) (bool, error) {
	if h == nil || h.battleRepo == nil || result == nil {
		return true, nil
	}
	payloadJSON, err := json.Marshal(result)
	if err != nil {
		return false, err
	}
	recordResult := int16(0)
	if result.Win {
		recordResult = 1
	}
	inserted, err := h.battleRepo.CreateRewardRecord(ctx, battle.RewardRecord{
		BattleID:    result.BattleID,
		PlayerID:    playerID,
		BattleType:  battle.BattleTypePVE,
		Result:      recordResult,
		RewardGold:  result.RewardGold,
		RewardExp:   result.RewardPlayerExp,
		PayloadJSON: payloadJSON,
	})
	return inserted, err
}

func (h *BattleHandler) loadPlayerBattleContext(connID string) (*session.Session, *player.Profile, []pet.LineupPet, *world.SceneSnapshot, error) {
	sess, err := h.sessionService.GetByConnID(connID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ctx := context.Background()
	profile, err := h.playerService.GetProfile(ctx, sess.PlayerID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	lineup, err := h.petService.ListLineup(ctx, sess.PlayerID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sceneSnapshot, err := h.worldService.GetSceneSnapshot(ctx, sess.PlayerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return sess, profile, lineup, sceneSnapshot, nil
}

func (h *BattleHandler) handleContextError(conn packetSender, seq uint32, err error) error {
	if errors.Is(err, session.ErrSessionNotFound) {
		return sendError(conn, seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if errors.Is(err, player.ErrPlayerNotFound) {
		return sendError(conn, seq, errcode.WSCodePlayerNotFound, "player not found")
	}
	return sendError(conn, seq, errcode.WSCodeInteractFailed, "load interact context failed")
}

func (h *BattleHandler) sendInteractResponse(conn packetSender, seq uint32, response protocol.InteractResp) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdInteractResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) sendPVPChallengeResponse(conn packetSender, seq uint32, accepted bool, reason string, challengeID uint64, targetPlayerID uint64) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdPVPChallengeResp, seq, errcode.WSCodeSuccess, protocol.PVPChallengeResp{
		Accepted:       accepted,
		Reason:         reason,
		ChallengeID:    challengeID,
		TargetPlayerID: targetPlayerID,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) sendPVPChallengeReplyResponse(conn packetSender, seq uint32, accepted bool, reason string, challengeID uint64) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdPVPChallengeReplyResp, seq, errcode.WSCodeSuccess, protocol.PVPChallengeReplyResp{
		Accepted:    accepted,
		Reason:      reason,
		ChallengeID: challengeID,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) sendBattleActionResponse(conn packetSender, seq uint32, accepted bool, reason string) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdBattleActionResp, seq, errcode.WSCodeSuccess, protocol.BattleActionResp{
		Accepted: accepted,
		Reason:   reason,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) pushBattleStartToParticipants(startSnapshot *battle.StartSnapshot) error {
	if startSnapshot == nil {
		return nil
	}
	payload := protocol.BattleStartPush{
		BattleID:             startSnapshot.BattleID,
		BattleType:           startSnapshot.BattleType,
		BattleVersion:        startSnapshot.BattleVersion,
		Frame:                startSnapshot.Frame,
		ParticipantPlayerIDs: append([]uint64{}, startSnapshot.ParticipantPlayerIDs...),
		Allies:               toProtocolBattleActors(startSnapshot.Allies),
		Enemies:              toProtocolBattleActors(startSnapshot.Enemies),
		Round:                startSnapshot.Round,
		Phase:                startSnapshot.Phase,
		ActiveActorID:        startSnapshot.ActiveActorID,
		ActivePetUID:         startSnapshot.ActivePetUID,
		CommandDeadlineMS:    startSnapshot.CommandDeadlineMS,
		AutoBattleEnabled:    startSnapshot.AutoBattleEnabled,
		PendingActorIDs:      append([]uint64{}, startSnapshot.PendingActorIDs...),
		ControllableActorIDs: append([]uint64{}, startSnapshot.ControllableActorIDs...),
	}
	for _, conn := range h.participantConns(startSnapshot.ParticipantPlayerIDs) {
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdBattleStartPush, 0, payload)); err != nil {
			return err
		}
	}
	return nil
}

func findInteractTarget(entities []world.Entity, entityID uint64) (world.Entity, bool) {
	if len(entities) == 0 {
		return world.Entity{}, false
	}
	if entityID == 0 {
		return entities[0], true
	}
	for _, entity := range entities {
		if entity.EntityID == entityID {
			return entity, true
		}
	}
	return world.Entity{}, false
}

func toProtocolBattleActors(actors []battle.ActorSnapshot) []protocol.BattleActorSnapshot {
	if len(actors) == 0 {
		return []protocol.BattleActorSnapshot{}
	}
	result := make([]protocol.BattleActorSnapshot, 0, len(actors))
	for _, actor := range actors {
		skills := make([]uint32, 0, len(actor.SkillIDs))
		skills = append(skills, actor.SkillIDs...)
		skillSnapshots := make([]protocol.BattleSkillSnapshot, 0, len(actor.Skills))
		for _, skill := range actor.Skills {
			skillSnapshots = append(skillSnapshots, protocol.BattleSkillSnapshot{
				SkillID:     skill.SkillID,
				Name:        skill.Name,
				TargetType:  skill.TargetType,
				TargetCount: skill.TargetCount,
			})
		}
		result = append(result, protocol.BattleActorSnapshot{
			ActorID:       actor.ActorID,
			ActorType:     actor.ActorType,
			OwnerPlayerID: actor.OwnerPlayerID,
			PetUID:        actor.PetUID,
			PetID:         actor.PetID,
			Name:          actor.Name,
			HP:            actor.HP,
			HPMax:         actor.HPMax,
			ATK:           actor.ATK,
			DEF:           actor.DEF,
			SPD:           actor.SPD,
			Skills:        skillSnapshots,
			SkillIDs:      skills,
			StatusIDs:     append([]uint32{}, actor.StatusIDs...),
			LineupIndex:   actor.LineupIndex,
		})
	}
	return result
}

func toProtocolBattleEvents(events []battle.Event) []protocol.BattleEvent {
	if len(events) == 0 {
		return []protocol.BattleEvent{}
	}
	result := make([]protocol.BattleEvent, 0, len(events))
	for _, event := range events {
		result = append(result, protocol.BattleEvent{
			EventType: event.EventType,
			SourceID:  event.SourceID,
			TargetID:  event.TargetID,
			SkillID:   event.SkillID,
			Value:     event.Value,
			StateID:   event.StateID,
			Label:     event.Label,
		})
	}
	return result
}

func toProtocolBattleActorStates(actors []battle.ActorState) []protocol.BattleActorState {
	if len(actors) == 0 {
		return []protocol.BattleActorState{}
	}
	result := make([]protocol.BattleActorState, 0, len(actors))
	for _, actor := range actors {
		result = append(result, protocol.BattleActorState{
			ActorID:    actor.ActorID,
			HP:         actor.HP,
			HPMax:      actor.HPMax,
			Dead:       actor.Dead,
			CanAct:     actor.CanAct,
			StatusIDs:  append([]uint32{}, actor.StatusIDs...),
			ChargeDone: actor.ChargeDone,
		})
	}
	return result
}

func mustJSONPacket(cmd uint16, seq uint32, payload any) *protocol.Packet {
	packet, err := protocol.NewJSONPacket(cmd, seq, errcode.WSCodeSuccess, payload)
	if err != nil {
		panic(err)
	}
	return packet
}

func toProtocolPetDetail(item pet.Pet) protocol.PetDetail {
	skills := make([]uint32, 0, len(item.SkillIDs))
	skills = append(skills, item.SkillIDs...)
	return protocol.PetDetail{
		PetUID:   item.PetUID,
		PetID:    item.PetID,
		Level:    item.Level,
		Exp:      item.Exp,
		Quality:  item.Quality,
		HP:       item.HP,
		HPMax:    item.HPMax,
		ATK:      item.ATK,
		DEF:      item.DEF,
		SPD:      item.SPD,
		SkillIDs: skills,
		InLineup: item.InLineup,
	}
}

func (h *BattleHandler) HandleNPCAction(conn packetSender, packet *protocol.Packet) error {
	var request protocol.NPCActionReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid npc action body")
	}

	sess, _, _, sceneSnapshot, err := h.loadPlayerBattleContext(conn.ID())
	if err != nil {
		return h.handleContextError(conn, packet.Seq, err)
	}
	target, found := findInteractTarget(sceneSnapshot.NearbyEntities, request.EntityID)
	if !found {
		return h.sendNPCActionResponse(conn, packet.Seq, protocol.NPCActionResp{Accepted: false, Reason: "target unavailable", EntityID: request.EntityID, EntryID: request.EntryID})
	}
	var (
		playerID    uint64 = sess.PlayerID
		questBefore []quest.Summary
	)
	response, ok := h.buildNPCActionResponse(context.Background(), playerID, target, request.EntryID)
	if !ok {
		return h.sendNPCActionResponse(conn, packet.Seq, protocol.NPCActionResp{Accepted: false, Reason: "unsupported npc action", EntityID: request.EntityID, EntryID: request.EntryID})
	}
	if h.questService != nil {
		questBefore, _ = listQuestSummaries(context.Background(), h.questService, playerID)
		_, _ = h.questService.HandleEvent(context.Background(), quest.Event{
			PlayerID:  playerID,
			EventType: "TALK_TO_NPC",
			NPCID:     target.EntityID,
			Count:     1,
		})
	}
	if err := h.sendNPCActionResponse(conn, packet.Seq, response); err != nil {
		return err
	}
	if h.questService != nil && playerID != 0 {
		_ = pushQuestDiff(context.Background(), conn, h.questService, playerID, questBefore)
	}
	return nil
}

func (h *BattleHandler) requirePlayerID(connID string) (uint64, error) {
	sess, err := h.sessionService.GetByConnID(connID)
	if err != nil {
		return 0, err
	}
	return sess.PlayerID, nil
}

func (h *BattleHandler) sendNPCActionResponse(conn packetSender, seq uint32, response protocol.NPCActionResp) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdNPCActionResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) buildNPCMenuResponse(ctx context.Context, playerID uint64, target world.Entity) (protocol.InteractResp, bool) {
	entries, ok := h.npcMenuEntriesByEntityID(ctx, playerID, target.EntityID)
	if !ok {
		return protocol.InteractResp{}, false
	}
	return protocol.InteractResp{
		ResponseType: "menu",
		EntityID:     target.EntityID,
		NPCName:      target.Name,
		MenuEntries:  entries,
	}, true
}

func (h *BattleHandler) buildNPCActionResponse(ctx context.Context, playerID uint64, target world.Entity, entryID string) (protocol.NPCActionResp, bool) {
	base := protocol.NPCActionResp{
		Accepted:   true,
		EntityID:   target.EntityID,
		EntryID:    entryID,
		ResultType: "notice",
		NPCName:    target.Name,
	}
	actionResult, err := h.npcService.FindActionResult(ctx, target.EntityID, entryID)
	if err != nil || actionResult == nil {
		return protocol.NPCActionResp{}, false
	}
	base.ResultType = actionResult.ResultType
	base.Notice = actionResult.Notice
	entries, ok := h.npcMenuEntriesByEntityID(ctx, playerID, target.EntityID)
	if ok {
		base.MenuEntries = entries
	}
	return base, true
}

func (h *BattleHandler) npcMenuEntriesByEntityID(ctx context.Context, playerID uint64, entityID uint64) ([]protocol.NpcMenuEntry, bool) {
	result := []protocol.NpcMenuEntry{}
	if h.questService != nil && playerID != 0 {
		if summaries, err := listQuestSummaries(ctx, h.questService, playerID); err == nil {
			result = append(result, questMenuEntriesForNPC(entityID, summaries)...)
		}
	}

	staticEntries, err := h.npcService.ListMenuEntriesByEntityID(ctx, entityID)
	if err == nil {
		for _, entry := range staticEntries {
			result = append(result, protocol.NpcMenuEntry{
				EntryID:   entry.EntryID,
				EntryType: entry.EntryType,
				Title:     entry.Title,
				Subtitle:  entry.Subtitle,
				State:     entry.State,
				Priority:  entry.Priority,
			})
		}
	}

	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func questMenuEntriesForNPC(npcID uint64, summaries []quest.Summary) []protocol.NpcMenuEntry {
	result := []protocol.NpcMenuEntry{}
	for _, summary := range summaries {
		switch {
		case summary.State == quest.StateAvailable && summary.StartNPCID == npcID:
			result = append(result, protocol.NpcMenuEntry{
				EntryID:    "quest_accept",
				EntryType:  "quest",
				QuestID:    summary.QuestID,
				QuestState: summary.State,
				Title:      "领取任务",
				Subtitle:   summary.Title,
				State:      "available",
				Priority:   100,
			})
		case summary.State == quest.StateReadyToSubmit && summary.SubmitNPCID == npcID:
			result = append(result, protocol.NpcMenuEntry{
				EntryID:    "quest_submit",
				EntryType:  "quest",
				QuestID:    summary.QuestID,
				QuestState: summary.State,
				Title:      "提交任务",
				Subtitle:   summary.Title,
				State:      "available",
				Priority:   100,
			})
		}
	}
	return result
}
