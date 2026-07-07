package wstransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/npcdialogue"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/playerskill"
	"pocket-pet-remake/server/internal/module/progression"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/runtimeview"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type BattleHandler struct {
	sessionService     *session.Service
	playerService      *player.Service
	petService         *pet.Service
	bagService         *bag.Service
	walletService      *wallet.Service
	worldService       *world.Service
	questService       *quest.Service
	npcService         *npc.Service
	npcDialogueService *npcdialogue.Service
	itemService        *item.Service
	equipmentService   *equipment.Service
	playerSkillService *playerskill.Service
	battleService      *battle.Service
	battleRepo         battle.Repository
	rewardService      *reward.Service
	runtimeSnapshots   *runtimeview.Service
	reconnectMu        sync.Mutex
	reconnectCache     map[uint64]protocol.BattleResultPush
	wildEncounterMu    sync.Mutex
	wildEncounterLast  map[uint64]time.Time
}

// SetRuntimeSnapshotService 注入统一运行时快照刷新入口。
func (h *BattleHandler) SetRuntimeSnapshotService(service *runtimeview.Service) {
	if h == nil {
		return
	}
	h.runtimeSnapshots = service
}

var dialogueItemTokenPattern = regexp.MustCompile(`\{item:(\d+)\}`)
var dialoguePetTokenPattern = regexp.MustCompile(`\{pet:(\d+)\}`)

func NewBattleHandler(sessionService *session.Service, playerService *player.Service, petService *pet.Service, bagService *bag.Service, walletService *wallet.Service, worldService *world.Service, questService *quest.Service, npcService *npc.Service, npcDialogueService *npcdialogue.Service, battleService *battle.Service, battleRepo battle.Repository, equipmentService *equipment.Service, playerSkillService *playerskill.Service, itemServices ...*item.Service) *BattleHandler {
	var itemService *item.Service
	if len(itemServices) > 0 {
		itemService = itemServices[0]
	}
	return &BattleHandler{
		sessionService:     sessionService,
		playerService:      playerService,
		petService:         petService,
		bagService:         bagService,
		walletService:      walletService,
		worldService:       worldService,
		questService:       questService,
		npcService:         npcService,
		npcDialogueService: npcDialogueService,
		itemService:        itemService,
		equipmentService:   equipmentService,
		playerSkillService: playerSkillService,
		battleService:      battleService,
		battleRepo:         battleRepo,
		rewardService:      reward.NewService(bagService, petService, playerService, nil, walletService),
		reconnectCache:     make(map[uint64]protocol.BattleResultPush),
		wildEncounterLast:  make(map[uint64]time.Time),
	}
}

func (h *BattleHandler) HandleInteract(conn packetSender, packet *protocol.Packet) error {
	var request protocol.InteractReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid interact body")
	}
	logBattlePacket("req", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdInteractReq, packet.Seq, request)

	sess, profile, lineup, sceneSnapshot, err := h.loadPlayerBattleContext(conn.ID(), request.SelfPos)
	if err != nil {
		return h.handleContextError(conn, packet.Seq, err)
	}
	target, found := findInteractTarget(sceneSnapshot.NearbyEntities, request.EntityID)
	if !found {
		return h.sendInteractResponse(conn, packet.Seq, protocol.InteractResp{Accepted: false, Reason: "target unavailable"})
	}
	if _, ok := h.buildNPCMenuResponse(context.Background(), sess.PlayerID, target); ok {
		return h.sendInteractResponse(conn, packet.Seq, protocol.InteractResp{
			Accepted: false,
			Reason:   "use npc menu request",
			EntityID: target.EntityID,
			NPCName:  target.Name,
		})
	}

	startSnapshot, err := h.battleService.StartPVE(context.Background(), profile, lineup, target, h.loadCharacterBattleSkillInput(context.Background(), sess.PlayerID))
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
	startPayload := protocol.BattleStartPush{
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
	logBattlePacket("push", conn.ID(), sess.PlayerID, protocol.CmdBattleStartPush, 0, startPayload)
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleStartPush, 0, startPayload))
}

func (h *BattleHandler) loadCharacterBattleSkillInput(ctx context.Context, playerID uint64) battle.CharacterBattleSkillInput {
	input := battle.EmptyCharacterBattleSkillInput()
	if h.runtimeSnapshots != nil && playerID > 0 {
		_ = h.runtimeSnapshots.RefreshPlayerRuntimeSnapshots(ctx, playerID)
	}
	if h.equipmentService != nil && playerID > 0 {
		items, err := h.equipmentService.ListEquipped(ctx, playerID)
		if err == nil {
			input.WeaponType = equipment.ExtractWeaponTypeFromEquipped(items)
			input.EquippedWeaponSkills = equipment.ExtractWeaponSkillsFromEquipped(items)
		}
	}
	if h.playerSkillService != nil && playerID > 0 {
		progressItems, err := h.playerSkillService.ListProgress(ctx, playerID)
		if err == nil {
			input.ProgressBySkillID = playerskill.ProgressMap(progressItems)
		}
	}
	return input
}

func (h *BattleHandler) applyBattleSkillProgressUpdates(ctx context.Context, playerID uint64, result *battle.ResultSnapshot) error {
	if h.playerSkillService == nil || result == nil || len(result.SkillProgressUpdates) == 0 {
		return nil
	}
	return h.playerSkillService.ApplyBattleUpdates(ctx, playerID, skillProgressUpdatesToPlayerSkill(result.SkillProgressUpdates))
}

func skillProgressUpdatesToPlayerSkill(updates []battle.SkillProgressUpdate) []playerskill.BattleUseUpdate {
	if len(updates) == 0 {
		return nil
	}
	result := make([]playerskill.BattleUseUpdate, 0, len(updates))
	for _, update := range updates {
		result = append(result, playerskill.BattleUseUpdate{
			SkillID:          update.SkillID,
			ExpGained:        update.ExpGained,
			FinalExp:         update.FinalExp,
			FinalLevel:       update.FinalLevel,
			NewlyLearned:     update.NewlyLearned,
			LearnExpRequired: update.LearnExpRequired,
		})
	}
	return result
}

func (h *BattleHandler) HandleNPCMenu(conn packetSender, packet *protocol.Packet) error {
	var request protocol.NPCMenuReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid npc menu body")
	}
	logBattlePacket("req", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdNPCMenuReq, packet.Seq, request)

	sess, _, _, sceneSnapshot, err := h.loadPlayerBattleContext(conn.ID(), nil)
	if err != nil {
		return h.handleContextError(conn, packet.Seq, err)
	}
	target, found := findInteractTarget(sceneSnapshot.NearbyEntities, request.EntityID)
	if !found {
		return h.sendNPCMenuResponse(conn, packet.Seq, protocol.NPCMenuResp{Accepted: false, Reason: "target unavailable", EntityID: request.EntityID})
	}
	menuResp, ok := h.buildNPCMenuResponse(context.Background(), sess.PlayerID, target)
	if !ok {
		return h.sendNPCMenuResponse(conn, packet.Seq, protocol.NPCMenuResp{Accepted: false, Reason: "npc menu unavailable", EntityID: request.EntityID, NPCName: target.Name})
	}
	menuResp.Accepted = true
	menuResp.Reason = "menu loaded"
	return h.sendNPCMenuResponse(conn, packet.Seq, menuResp)
}

const wildEncounterCooldown = 2 * time.Second

func (h *BattleHandler) HandleWildEncounter(conn packetSender, packet *protocol.Packet) error {
	var request protocol.WildEncounterReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid wild encounter body")
	}
	logBattlePacket("req", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdWildEncounterReq, packet.Seq, request)

	sess, profile, lineup, _, err := h.loadPlayerBattleContext(conn.ID(), request.SelfPos)
	if err != nil {
		return h.handleContextError(conn, packet.Seq, err)
	}
	if request.SceneID == 0 || request.SceneID != profile.SceneID {
		return h.sendWildEncounterResponse(conn, packet.Seq, protocol.WildEncounterResp{Accepted: false, Reason: "scene mismatch"})
	}
	if !h.allowWildEncounterReport(sess.PlayerID) {
		return h.sendWildEncounterResponse(conn, packet.Seq, protocol.WildEncounterResp{Accepted: false, Reason: "encounter cooldown"})
	}

	startSnapshot, err := h.battleService.StartPVEWildEncounterByScene(context.Background(), profile, lineup, request.SceneID, h.loadCharacterBattleSkillInput(context.Background(), sess.PlayerID))
	if err != nil {
		if errors.Is(err, battle.ErrBattleAlreadyActive) {
			return h.sendWildEncounterResponse(conn, packet.Seq, protocol.WildEncounterResp{Accepted: false, Reason: "battle already active"})
		}
		if errors.Is(err, battle.ErrNoLineupAvailable) {
			return h.sendWildEncounterResponse(conn, packet.Seq, protocol.WildEncounterResp{Accepted: false, Reason: "no lineup available"})
		}
		if errors.Is(err, battle.ErrWildEncounterUnavailable) {
			return h.sendWildEncounterResponse(conn, packet.Seq, protocol.WildEncounterResp{Accepted: false, Reason: "wild encounter unavailable"})
		}
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "wild encounter battle start failed")
	}
	h.markWildEncounterReport(sess.PlayerID)

	if err := h.sendWildEncounterResponse(conn, packet.Seq, protocol.WildEncounterResp{Accepted: true, Reason: "battle started"}); err != nil {
		return err
	}
	startPayload := protocol.BattleStartPush{
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
	logBattlePacket("push", conn.ID(), sess.PlayerID, protocol.CmdBattleStartPush, 0, startPayload)
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleStartPush, 0, startPayload))
}

func (h *BattleHandler) allowWildEncounterReport(playerID uint64) bool {
	h.wildEncounterMu.Lock()
	defer h.wildEncounterMu.Unlock()
	lastReport, exists := h.wildEncounterLast[playerID]
	if exists && time.Since(lastReport) < wildEncounterCooldown {
		return false
	}
	return true
}

func (h *BattleHandler) markWildEncounterReport(playerID uint64) {
	h.wildEncounterMu.Lock()
	defer h.wildEncounterMu.Unlock()
	h.wildEncounterLast[playerID] = time.Now()
}

func (h *BattleHandler) sendWildEncounterResponse(conn packetSender, seq uint32, response protocol.WildEncounterResp) error {
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdWildEncounterResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdWildEncounterResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) HandleBattleAction(conn packetSender, packet *protocol.Packet) error {
	var request protocol.BattleActionReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid battle action body")
	}
	logBattlePacket("req", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdBattleActionReq, packet.Seq, request)

	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	if request.ActionType == battle.ActionTypeCapture {
		if err := h.validateCaptureBagItem(context.Background(), sess.PlayerID, request.ItemID, request.BagSlotIndex); err != nil {
			return h.sendBattleActionResponse(conn, packet.Seq, false, err.Error(), false)
		}
	}

	outcome, err := h.battleService.SubmitAction(context.Background(), sess.PlayerID, battle.ActionRequest{
		BattleID:          request.BattleID,
		Round:             request.Round,
		ActionType:        request.ActionType,
		ActorID:           request.ActorID,
		SkillID:           request.SkillID,
		TargetID:          request.TargetID,
		ItemID:            request.ItemID,
		BagSlotIndex:      request.BagSlotIndex,
		AutoBattleEnabled: request.AutoBattleEnabled,
	})
	if err != nil {
		if errors.Is(err, battle.ErrBattleNotFound) {
			return h.sendBattleActionResponse(conn, packet.Seq, false, "battle not found", false)
		}
		if errors.Is(err, battle.ErrInvalidAction) {
			return h.sendBattleActionResponse(conn, packet.Seq, false, "invalid action", false)
		}
		return sendError(conn, packet.Seq, errcode.WSCodeBattleActionInvalid, "battle action failed")
	}

	var bagSnapshot *bag.RuntimeContainerSnapshot
	if request.ActionType == battle.ActionTypeCapture && outcome != nil && outcome.Response.Accepted {
		if h.bagService == nil {
			return h.sendBattleActionResponse(conn, packet.Seq, false, "bag service unavailable", false)
		}
		consumedBag, consumeErr := h.bagService.ConsumeRuntimeItemStack(
			context.Background(),
			sess.PlayerID,
			bag.ContainerTypeBag,
			request.BagSlotIndex,
			1,
			"battle_capture",
			request.BattleID,
		)
		if consumeErr != nil {
			return h.sendBattleActionResponse(conn, packet.Seq, false, "capture item consume failed", false)
		}
		bagSnapshot = consumedBag
	}

	if err := h.sendBattleActionResponse(conn, packet.Seq, outcome.Response.Accepted, outcome.Response.Reason, outcome.Response.CaptureSuccess); err != nil {
		return err
	}
	return h.pushBattleOutcome(context.Background(), conn, sess.PlayerID, outcome, bagSnapshot)
}

func (h *BattleHandler) HandlePVPChallenge(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PVPChallengeReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pvp challenge body")
	}
	logBattlePacket("req", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdPVPChallengeReq, packet.Seq, request)
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
	pushPayload := protocol.PVPChallengePush{
		ChallengeID: challenge.ChallengeID,
		Challenger: protocol.PlayerBrief{
			PlayerID: challengerProfile.PlayerID,
			Name:     challengerProfile.Name,
			Level:    challengerProfile.Level,
		},
		ExpiresAtMS: challenge.ExpiresAt.UnixMilli(),
	}
	logBattlePacket("push", targetSession.Conn.ID(), request.TargetPlayerID, protocol.CmdPVPChallengePush, 0, pushPayload)
	return targetSession.Conn.SendPacket(mustJSONPacket(protocol.CmdPVPChallengePush, 0, pushPayload))
}

func (h *BattleHandler) HandlePVPChallengeReply(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PVPChallengeReplyReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid pvp challenge reply body")
	}
	logBattlePacket("req", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdPVPChallengeReplyReq, packet.Seq, request)
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

	challengerProfile, err := h.playerService.GetBattleReadyProfile(context.Background(), challenge.ChallengerPlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "challenger not found")
	}
	challengerLineup, err := h.petService.ListLineup(context.Background(), challenge.ChallengerPlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "load challenger lineup failed")
	}
	defenderProfile, err := h.playerService.GetBattleReadyProfile(context.Background(), challenge.DefenderPlayerID)
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
	return h.pushBattleOutcome(context.Background(), conn, sess.PlayerID, outcome, nil)
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

// HandleSessionDisconnect ends single-player battles as failures and keeps the
// legacy PVP loser handling. The result is cached so a reconnecting mobile
// client can leave the battle scene cleanly without receiving rewards.
func (h *BattleHandler) HandleSessionDisconnect(playerID uint64) {
	if h == nil || h.battleService == nil || playerID == 0 {
		return
	}
	if result := h.battleService.ResolveDisconnect(context.Background(), playerID); result != nil {
		ctx := context.Background()
		h.storeReconnectResult(playerID, h.buildBattleResultPayload(ctx, result, &battleSettlement{}))
		_ = h.pushBattleResultToParticipants(ctx, result, &battleSettlement{})
		return
	}
	h.battleService.EnableAutoForPlayer(context.Background(), playerID)
}

// StartCustodySweeper keeps the legacy background hook alive. Auto progression
// is currently disabled because command countdowns live on the client.
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

// ProcessAutoCustodyOnce performs one legacy sweep. The battle service returns
// no synthesized outcomes while client-owned round intents are enabled.
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

func (h *BattleHandler) pushBattleOutcome(ctx context.Context, conn packetSender, playerID uint64, outcome *battle.ActionOutcome, preConsumedBag *bag.RuntimeContainerSnapshot) error {
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
	settlement, grantErr := h.applyBattleResultSideEffects(ctx, conn, playerID, outcome.Result, preConsumedBag)
	return h.pushBattleResultAfterSideEffects(ctx, conn, playerID, outcome.Result, settlement, grantErr)
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
	if outcome.Result == nil {
		return nil
	}
	settlement, grantErr := h.applyBattleResultSideEffects(ctx, conn, playerID, outcome.Result, nil)
	if conn != nil {
		h.clearReconnectResult(playerID)
	}
	return h.pushBattleResultAfterSideEffects(ctx, conn, playerID, outcome.Result, settlement, grantErr)
}

// pushBattleResultAfterSideEffects 无论发奖是否成功都推送 4013，避免客户端只收到 4012 finished 而无法展示奖励弹窗。
func (h *BattleHandler) pushBattleResultAfterSideEffects(
	ctx context.Context,
	conn packetSender,
	playerID uint64,
	result *battle.ResultSnapshot,
	settlement *battleSettlement,
	grantErr error,
) error {
	if result == nil {
		return grantErr
	}
	if settlement == nil {
		settlement = &battleSettlement{}
	}
	var pushErr error
	if len(result.ParticipantPlayerIDs) > 1 {
		pushErr = h.pushBattleResultToParticipants(ctx, result, settlement)
	} else if conn != nil {
		pushErr = h.pushBattleResultPacket(ctx, conn, result, settlement)
	} else if playerID != 0 {
		h.storeReconnectResult(playerID, h.buildBattleResultPayload(ctx, result, settlement))
	}
	if pushErr != nil {
		return pushErr
	}
	if grantErr != nil {
		_ = h.pushNoticeToPlayer(playerID, "战斗奖励发放异常，请检查背包空间后重试")
	}
	followUpErr := h.pushBattleSettlementFollowUps(ctx, conn, playerID, result, settlement)
	if grantErr != nil {
		return followUpErr
	}
	if followUpErr != nil {
		return followUpErr
	}
	return nil
}

func (h *BattleHandler) pushBattleStatePacket(conn packetSender, state *battle.StateSnapshot) error {
	if conn == nil || state == nil {
		return nil
	}
	payload := protocol.BattleStatePush{
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
	}
	logBattlePacket("push", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdBattleStatePush, 0, payload)
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleStatePush, 0, payload))
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
	PlayerProfile    *player.Profile
	LevelUpCount     uint32
	AttrPointsGained uint32
	CombatBonusGain  progression.LevelUpCombatBonus
	BagSnapshot      *bag.RuntimeContainerSnapshot
	Wallet           *wallet.Snapshot
	WalletReason     string
	WalletRefID      uint64
	Pets             []pet.Pet
	CapturedPet      *pet.Pet
	questBefore      []quest.Summary
	GrantedRewards   []reward.Entry
	// rewardsAlreadyGranted 表示本场战斗奖励已在更早的结算链路中落库，当前包只做同步展示。
	rewardsAlreadyGranted bool
}

func (h *BattleHandler) pushBattleResultPacket(ctx context.Context, conn packetSender, result *battle.ResultSnapshot, settlement *battleSettlement) error {
	if conn == nil || result == nil {
		return nil
	}
	payload := h.buildBattleResultPayload(ctx, result, settlement)
	logBattlePacket("push", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdBattleResultPush, 0, payload)
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleResultPush, 0, payload))
}

func (h *BattleHandler) pushBattleResultToParticipants(ctx context.Context, result *battle.ResultSnapshot, settlement *battleSettlement) error {
	if result == nil {
		return nil
	}
	payload := h.buildBattleResultPayload(ctx, result, settlement)
	for _, conn := range h.participantConns(result.ParticipantPlayerIDs) {
		logBattlePacket("push", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdBattleResultPush, 0, payload)
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdBattleResultPush, 0, payload)); err != nil {
			return err
		}
	}
	return nil
}

func (h *BattleHandler) buildBattleResultPayload(ctx context.Context, result *battle.ResultSnapshot, settlement *battleSettlement) protocol.BattleResultPush {
	playerGold := uint32(0)
	playerExp := uint64(0)
	playerLevel := uint32(0)
	levelUpCount := uint32(0)
	attrPointsGained := uint32(0)
	var levelUpBonus *protocol.LevelUpBonus
	freeAttrPoints := uint32(0)
	expToNext := uint64(0)
	petRewards := make([]protocol.BattlePetReward, 0, len(result.PetResults))
	if settlement != nil && settlement.PlayerProfile != nil {
		playerExp = settlement.PlayerProfile.Exp
		playerLevel = settlement.PlayerProfile.Level
		expToNext = settlement.PlayerProfile.ExpToNext
		freeAttrPoints = settlement.PlayerProfile.FreeAttrPoints
		levelUpCount = settlement.LevelUpCount
		attrPointsGained = settlement.AttrPointsGained
		levelUpBonus = toProtocolLevelUpBonus(settlement.LevelUpCount, settlement.CombatBonusGain)
	}
	if settlement != nil && settlement.Wallet != nil {
		playerGold = legacyGoldFromWalletSnapshot(settlement.Wallet)
	}
	for _, petResult := range result.PetResults {
		reward := protocol.BattlePetReward{
			PetUID:    petResult.PetUID,
			Exp:       petResult.ExpGained,
			ExpGained: petResult.ExpGained,
		}
		if settlement != nil {
			for _, updatedPet := range settlement.Pets {
				if updatedPet.PetUID != petResult.PetUID {
					continue
				}
				reward.LevelUpCount = updatedPet.LastLevelUpCount
				reward.AttrPointsGained = updatedPet.LastAttrPointsGained
				reward.FreeAttrPoints = updatedPet.FreeAttrPoints
				reward.ExpToNext = updatedPet.ExpToNext
				reward.Level = updatedPet.Level
				reward.PetID = updatedPet.PetID
				break
			}
		}
		petRewards = append(petRewards, reward)
	}
	skillProgress := make([]protocol.BattleSkillProgressPush, 0, len(result.SkillProgressUpdates))
	for _, update := range result.SkillProgressUpdates {
		if update.SkillID == 0 || update.ExpGained == 0 {
			continue
		}
		skillProgress = append(skillProgress, protocol.BattleSkillProgressPush{
			SkillID:          update.SkillID,
			SkillName:        update.SkillName,
			SkillExp:         update.FinalExp,
			LearnExpRequired: update.LearnExpRequired,
			ExpGained:        update.ExpGained,
			NewlyLearned:     update.NewlyLearned,
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
		Reason:           result.Reason,
		RewardGold:       result.RewardGold,
		RewardPlayerExp:  result.RewardPlayerExp,
		PlayerGold:       playerGold,
		PlayerExp:        playerExp,
		PlayerLevel:      playerLevel,
		LevelUpCount:     levelUpCount,
		AttrPointsGained: attrPointsGained,
		LevelUpBonus:     levelUpBonus,
		FreeAttrPoints:   freeAttrPoints,
		ExpToNext:        expToNext,
		PetRewards:       petRewards,
		Rewards: enrichProtocolPopupRewardItemNames(
			ctx,
			h.itemService,
			battlePopupRewardsFromResult(result, settlement),
		),
		DropTexts:        append([]string{}, result.DropTexts...),
		CaptureSuccess:   result.CaptureSuccess,
		CaptureMonsterID: result.CaptureMonsterID,
		CapturedPetID:    result.CapturedPetID,
		CapturedPetUID:   capturedPetUIDFromSettlement(settlement),
		SkillProgress:    skillProgress,
	}
}

func battlePopupRewardsFromSettlement(settlement *battleSettlement) []protocol.QuestReward {
	if settlement == nil || len(settlement.GrantedRewards) == 0 {
		return nil
	}
	return toProtocolPopupRewards(settlement.GrantedRewards)
}

// battlePopupRewardsFromResult 优先使用发奖服务回写的 rewards。
// 仅当本场奖励已确认落库（首次发奖成功或命中 battle_record 去重后的同步包）时，
// 才允许回退到战斗快照字段，避免“弹窗有奖励但账户未变更”的假象。
func battlePopupRewardsFromResult(result *battle.ResultSnapshot, settlement *battleSettlement) []protocol.QuestReward {
	rewards := battlePopupRewardsFromSettlement(settlement)
	if len(rewards) > 0 {
		return rewards
	}
	if result == nil || !result.Win {
		return nil
	}
	// 重复结算同步包不再弹奖励，避免同一场战斗重复展示。
	if settlement != nil && settlement.rewardsAlreadyGranted {
		return nil
	}
	// 首次结算或发奖异常时，用战斗快照里的奖励摘要驱动客户端弹窗。
	return popupRewardsFromBattleResultSnapshot(result)
}

// popupRewardsFromBattleResultSnapshot 把战斗结果里的奖励摘要转成弹窗展示结构。
func popupRewardsFromBattleResultSnapshot(result *battle.ResultSnapshot) []protocol.QuestReward {
	if result == nil || !result.Win {
		return nil
	}
	fallback := make([]protocol.QuestReward, 0, len(result.DropItems)+len(result.AttrRewards)+2)
	if result.RewardPlayerExp > 0 {
		fallback = append(fallback, protocol.QuestReward{Type: "exp", Value: result.RewardPlayerExp})
	}
	if result.RewardGold > 0 {
		fallback = append(fallback, protocol.QuestReward{Type: "gold", Value: uint64(result.RewardGold)})
	}
	for _, item := range result.DropItems {
		if item.ItemID == 0 || item.Quantity == 0 {
			continue
		}
		fallback = append(fallback, protocol.QuestReward{
			Type:   "item",
			ItemID: item.ItemID,
			Count:  item.Quantity,
		})
	}
	for _, attr := range result.AttrRewards {
		if attr.AttrKey == "" || attr.Value == 0 {
			continue
		}
		fallback = append(fallback, protocol.QuestReward{Type: "attr", AttrKey: attr.AttrKey, Value: attr.Value})
	}
	if len(fallback) == 0 {
		return nil
	}
	return fallback
}

func capturedPetUIDFromSettlement(settlement *battleSettlement) uint64 {
	if settlement == nil || settlement.CapturedPet == nil {
		return 0
	}
	return settlement.CapturedPet.PetUID
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

func (h *BattleHandler) applyBattleResultSideEffects(ctx context.Context, _ packetSender, playerID uint64, result *battle.ResultSnapshot, preConsumedBag *bag.RuntimeContainerSnapshot) (*battleSettlement, error) {
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
		return h.loadBattleSettlementSync(ctx, playerID, result)
	}
	// 发奖占位记录写在奖励真正落库之前；若后续任一步失败，需要删除记录以便重试。
	commitRewardRecord := false
	defer func() {
		if !commitRewardRecord {
			h.rollbackBattleRewardGrantRecord(ctx, result.BattleID, playerID)
		}
	}()
	settlement := &battleSettlement{}
	if preConsumedBag != nil {
		settlement.BagSnapshot = preConsumedBag
	}
	if result.CaptureSuccess {
		if h.petService != nil && result.CaptureMonsterID > 0 {
			grantResult, grantErr := h.petService.GrantCapturedPet(ctx, playerID, result.CaptureMonsterID, "battle_capture", result.BattleID)
			if grantErr != nil {
				return nil, grantErr
			}
			if grantResult != nil {
				captured := grantResult.Pet
				settlement.CapturedPet = &captured
			}
		}
		commitRewardRecord = true
		return settlement, nil
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
	if h.rewardService != nil && result.Win {
		grantEntries, grantEntryErr := h.buildBattleGrantEntries(ctx, playerID, result)
		if grantEntryErr != nil {
			return nil, grantEntryErr
		}
		grantResult, err := h.rewardService.GrantRuntimeRewards(ctx, reward.GrantInput{
			PlayerID:     playerID,
			ReasonType:   "battle_reward",
			ReasonRefID:  result.BattleID,
			OperatorType: "system",
			OperatorID:   playerID,
			Rewards:      grantEntries,
		})
		if err != nil {
			return nil, err
		}
		if recordErr := h.recordGrantedUniqueBattleItems(ctx, playerID, result, grantResult); recordErr != nil {
			return nil, recordErr
		}
		settlement.LevelUpCount = grantResult.LevelUpCount
		settlement.AttrPointsGained = grantResult.AttrPointsGained
		settlement.CombatBonusGain = grantResult.CombatBonusGain
		settlement.Wallet = grantResult.Wallet
		if grantResult.Wallet != nil {
			settlement.WalletReason = "battle_reward"
			settlement.WalletRefID = result.BattleID
		}
		if grantResult.BagUpdated && h.bagService != nil {
			bagSnapshot, err := h.bagService.ListRuntimeContainer(ctx, playerID, bag.ContainerTypeBag)
			if err != nil {
				return nil, err
			}
			settlement.BagSnapshot = bagSnapshot
		}
		if len(grantResult.Granted) > 0 {
			result.DropTexts = battleDropTextsFromGrantedRewards(grantResult.Granted)
			settlement.GrantedRewards = append([]reward.Entry{}, grantResult.Granted...)
		}
	}
	// 战斗胜利后始终以数据库最新档案回填结算包，避免发奖服务未带回 PlayerProfile 时客户端经验停留在 0。
	if result.Win && h.playerService != nil {
		currentProfile, err := h.playerService.GetProfile(ctx, playerID)
		if err != nil {
			return nil, err
		}
		settlement.PlayerProfile = currentProfile
	}
	if (result.RewardGold > 0 || result.RewardPlayerExp > 0) && settlement.Wallet == nil && h.walletService != nil {
		currentWallet, err := h.walletService.GetRuntimeWallet(ctx, playerID)
		if err != nil {
			return nil, err
		}
		settlement.Wallet = currentWallet
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
	if err := h.applyBattleSkillProgressUpdates(ctx, playerID, result); err != nil {
		return nil, err
	}
	settlement.questBefore = questBefore
	commitRewardRecord = true
	return settlement, nil
}

// rollbackBattleRewardGrantRecord 回滚尚未完成发奖的 battle_record 占位，避免后续重试被误判为重复结算。
func (h *BattleHandler) rollbackBattleRewardGrantRecord(ctx context.Context, battleID uint64, playerID uint64) {
	if h == nil || h.battleRepo == nil || battleID == 0 || playerID == 0 {
		return
	}
	_ = h.battleRepo.DeleteRewardRecord(ctx, battleID, playerID)
}

// loadBattleSettlementSync 在 battle_record 已存在时加载当前权威档案用于客户端同步。
// 这里不会重复发奖，但会带回最新玩家/钱包快照，避免重复结算包把客户端状态写回 0。
func (h *BattleHandler) loadBattleSettlementSync(ctx context.Context, playerID uint64, result *battle.ResultSnapshot) (*battleSettlement, error) {
	if result == nil {
		return nil, nil
	}
	settlement := &battleSettlement{rewardsAlreadyGranted: true}
	if h.playerService != nil {
		currentProfile, err := h.playerService.GetProfile(ctx, playerID)
		if err != nil {
			return nil, err
		}
		settlement.PlayerProfile = currentProfile
	}
	if h.walletService != nil {
		currentWallet, err := h.walletService.GetRuntimeWallet(ctx, playerID)
		if err != nil {
			return nil, err
		}
		settlement.Wallet = currentWallet
		settlement.WalletReason = "battle_reward_sync"
		settlement.WalletRefID = result.BattleID
	}
	if h.bagService != nil {
		bagSnapshot, err := h.bagService.ListRuntimeContainer(ctx, playerID, bag.ContainerTypeBag)
		if err != nil {
			return nil, err
		}
		settlement.BagSnapshot = bagSnapshot
	}
	return settlement, nil
}

// toBattleRewardEntries 把当前战斗结算结果转成统一发奖服务可消费的奖励列表。
func toBattleRewardEntries(result *battle.ResultSnapshot) []reward.Entry {
	if result == nil {
		return nil
	}
	entries := make([]reward.Entry, 0, len(result.DropItems)+len(result.AttrRewards)+2)
	if result.RewardGold > 0 {
		entries = append(entries, reward.Entry{Type: "gold", Value: uint64(result.RewardGold)})
	}
	if result.RewardPlayerExp > 0 {
		entries = append(entries, reward.Entry{Type: "exp", Value: result.RewardPlayerExp})
	}
	for _, dropItem := range result.DropItems {
		if dropItem.ItemID == 0 || dropItem.Quantity == 0 {
			continue
		}
		entries = append(entries, reward.Entry{
			Type:   "item",
			ItemID: dropItem.ItemID,
			Count:  dropItem.Quantity,
		})
	}
	for _, attr := range result.AttrRewards {
		if attr.AttrKey == "" || attr.Value == 0 {
			continue
		}
		entries = append(entries, reward.Entry{Type: "attr", AttrKey: attr.AttrKey, Value: attr.Value})
	}
	return entries
}

// buildBattleGrantEntries 在真正发奖前过滤掉玩家已获得的唯一物品，经验与货币始终发放。
func (h *BattleHandler) buildBattleGrantEntries(ctx context.Context, playerID uint64, result *battle.ResultSnapshot) ([]reward.Entry, error) {
	if result == nil {
		return nil, nil
	}
	entries := make([]reward.Entry, 0, len(result.DropItems)+len(result.AttrRewards)+2)
	if result.RewardGold > 0 {
		entries = append(entries, reward.Entry{Type: "gold", Value: uint64(result.RewardGold)})
	}
	if result.RewardPlayerExp > 0 {
		entries = append(entries, reward.Entry{Type: "exp", Value: result.RewardPlayerExp})
	}
	for _, dropItem := range result.DropItems {
		if dropItem.ItemID == 0 || dropItem.Quantity == 0 {
			continue
		}
		if dropItem.GrantOnce && h.bagService != nil {
			owned, err := h.bagService.PlayerHasEverOwnedItem(ctx, playerID, dropItem.ItemID)
			if err != nil {
				return nil, err
			}
			if owned {
				continue
			}
		}
		entries = append(entries, reward.Entry{
			Type:   "item",
			ItemID: dropItem.ItemID,
			Count:  dropItem.Quantity,
		})
	}
	for _, attr := range result.AttrRewards {
		if attr.AttrKey == "" || attr.Value == 0 {
			continue
		}
		entries = append(entries, reward.Entry{Type: "attr", AttrKey: attr.AttrKey, Value: attr.Value})
	}
	return entries, nil
}

// recordGrantedUniqueBattleItems 在唯一物品首次发放成功后写入获得记录，供后续战斗去重。
func (h *BattleHandler) recordGrantedUniqueBattleItems(ctx context.Context, playerID uint64, result *battle.ResultSnapshot, grantResult *reward.GrantResult) error {
	if h == nil || h.bagService == nil || result == nil || grantResult == nil || len(grantResult.Granted) == 0 {
		return nil
	}
	grantOnceItems := make(map[uint64]struct{})
	for _, dropItem := range result.DropItems {
		if dropItem.GrantOnce && dropItem.ItemID > 0 {
			grantOnceItems[dropItem.ItemID] = struct{}{}
		}
	}
	for _, grantedEntry := range grantResult.Granted {
		if grantedEntry.Type != "item" || grantedEntry.ItemID == 0 {
			continue
		}
		if _, ok := grantOnceItems[grantedEntry.ItemID]; !ok {
			continue
		}
		if err := h.bagService.RecordUniqueItemObtained(ctx, playerID, grantedEntry.ItemID, "battle_reward", result.BattleID); err != nil {
			return err
		}
	}
	return nil
}

// battleDropTextsFromGrantedRewards 用统一发奖服务成功发放后的奖励结果重建掉落文本。
// 这里只提取 item 奖励，保证战斗结果面板仍然只展示战利品摘要而不是全部奖励。
func battleDropTextsFromGrantedRewards(values []reward.Entry) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value.Type != "item" || value.ItemID == 0 || value.Count == 0 {
			continue
		}
		itemName := value.ItemName
		if itemName == "" {
			itemName = fmt.Sprintf("物品%d", value.ItemID)
		}
		result = append(result, fmt.Sprintf("掉落: %s x%d", itemName, value.Count))
	}
	return result
}

func (h *BattleHandler) pushBattleSettlementFollowUps(ctx context.Context, conn packetSender, playerID uint64, result *battle.ResultSnapshot, settlement *battleSettlement) error {
	if conn == nil || result == nil || settlement == nil || result.BattleType != battle.BattleTypePVE {
		return nil
	}
	if settlement.CapturedPet != nil {
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
			Pet: toProtocolPetDetail(*settlement.CapturedPet),
		})); err != nil {
			return err
		}
	}
	for _, updatedPet := range settlement.Pets {
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
			Pet: toProtocolPetDetail(updatedPet),
		})); err != nil {
			return err
		}
	}
	if settlement.BagSnapshot != nil {
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*settlement.BagSnapshot))); err != nil {
			return err
		}
	}
	if settlement.Wallet != nil && settlement.WalletReason != "" {
		if err := pushWalletUpdatePacket(conn, *settlement.Wallet, settlement.WalletReason, settlement.WalletRefID); err != nil {
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

func (h *BattleHandler) loadPlayerBattleContext(connID string, clientSelfPos *protocol.Vec2i) (*session.Session, *player.Profile, []pet.LineupPet, *world.SceneSnapshot, error) {
	sess, err := h.sessionService.GetByConnID(connID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ctx := context.Background()
	profile, err := h.playerService.GetBattleReadyProfile(ctx, sess.PlayerID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	h.syncBattleReturnPosition(ctx, sess.PlayerID, profile, clientSelfPos)
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

// syncBattleReturnPosition 在开战前把客户端上报的场景坐标写回玩家档案，确保 return_pos 与战斗结束回世界位置一致。
func (h *BattleHandler) syncBattleReturnPosition(ctx context.Context, playerID uint64, profile *player.Profile, clientSelfPos *protocol.Vec2i) {
	if h == nil || h.playerService == nil || profile == nil || clientSelfPos == nil {
		return
	}
	pos := world.Vec2i{X: clientSelfPos.X, Y: clientSelfPos.Y}
	if err := h.playerService.UpdatePosition(ctx, playerID, profile.SceneID, pos.X, pos.Y); err != nil {
		return
	}
	profile.PosX = pos.X
	profile.PosY = pos.Y
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
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdInteractResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdInteractResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) sendNPCMenuResponse(conn packetSender, seq uint32, response protocol.NPCMenuResp) error {
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdNPCMenuResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdNPCMenuResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) sendPVPChallengeResponse(conn packetSender, seq uint32, accepted bool, reason string, challengeID uint64, targetPlayerID uint64) error {
	response := protocol.PVPChallengeResp{
		Accepted:       accepted,
		Reason:         reason,
		ChallengeID:    challengeID,
		TargetPlayerID: targetPlayerID,
	}
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdPVPChallengeResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdPVPChallengeResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) sendPVPChallengeReplyResponse(conn packetSender, seq uint32, accepted bool, reason string, challengeID uint64) error {
	response := protocol.PVPChallengeReplyResp{
		Accepted:    accepted,
		Reason:      reason,
		ChallengeID: challengeID,
	}
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdPVPChallengeReplyResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdPVPChallengeReplyResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) validateCaptureBagItem(ctx context.Context, playerID uint64, itemID uint32, bagSlotIndex uint32) error {
	if playerID == 0 || itemID == 0 || bagSlotIndex == 0 {
		return fmt.Errorf("capture item required")
	}
	if h.bagService == nil {
		return fmt.Errorf("bag service unavailable")
	}
	container, err := h.bagService.ListRuntimeContainer(ctx, playerID, bag.ContainerTypeBag)
	if err != nil {
		return fmt.Errorf("load bag failed")
	}
	if container == nil {
		return fmt.Errorf("bag item not found")
	}
	for _, current := range container.Items {
		if current.SlotIndex != bagSlotIndex {
			continue
		}
		if uint32(current.ItemID) != itemID {
			return fmt.Errorf("bag item mismatch")
		}
		if current.Quantity == 0 {
			return fmt.Errorf("bag item not found")
		}
		return nil
	}
	return fmt.Errorf("bag item not found")
}

func (h *BattleHandler) sendBattleActionResponse(conn packetSender, seq uint32, accepted bool, reason string, captureSuccess bool) error {
	response := protocol.BattleActionResp{
		Accepted:       accepted,
		Reason:         reason,
		CaptureSuccess: captureSuccess,
	}
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdBattleActionResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdBattleActionResp, seq, errcode.WSCodeSuccess, response)
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
		logBattlePacket("push", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdBattleStartPush, 0, payload)
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
				SkillID:       skill.SkillID,
				Name:          skill.Name,
				TargetType:    skill.TargetType,
				TargetCount:   skill.TargetCount,
				AnimationKey:  skill.AnimationKey,
				SkillVisualID: skill.SkillVisualID,
				CastColor:     skill.CastColor,
				ImpactColor:   skill.ImpactColor,
				Projectile:    skill.Projectile,
				IsBasicAttack: skill.IsBasicAttack,
				Level:         skill.Level,
			})
		}
		result = append(result, protocol.BattleActorSnapshot{
			ActorID:       actor.ActorID,
			ActorType:     actor.ActorType,
			UnitClass:     actor.UnitClass,
			OwnerPlayerID: actor.OwnerPlayerID,
			PetUID:        actor.PetUID,
			PetID:         actor.PetID,
			Name:          actor.Name,
			SkinID:        actor.SkinID,
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

func (h *BattleHandler) HandleNPCAction(conn packetSender, packet *protocol.Packet) error {
	var request protocol.NPCActionReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid npc action body")
	}
	logBattlePacket("req", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdNPCActionReq, packet.Seq, request)

	sess, profile, lineup, sceneSnapshot, err := h.loadPlayerBattleContext(conn.ID(), request.SelfPos)
	if err != nil {
		return h.handleContextError(conn, packet.Seq, err)
	}
	target, found := findInteractTarget(sceneSnapshot.NearbyEntities, request.EntityID)
	if !found {
		return h.sendNPCActionResponse(conn, packet.Seq, protocol.NPCActionResp{Accepted: false, Reason: "target unavailable", EntityID: request.EntityID, EntryID: request.EntryID})
	}

	actionResult, err := h.npcService.FindActionResult(context.Background(), target.EntityID, request.EntryID)
	if err != nil || actionResult == nil {
		return h.sendNPCActionResponse(conn, packet.Seq, protocol.NPCActionResp{Accepted: false, Reason: "unsupported npc action", EntityID: request.EntityID, EntryID: request.EntryID})
	}
	if actionResult.ResultType == "battle" {
		return h.handleNPCBattleAction(conn, packet.Seq, sess.PlayerID, profile, lineup, target, request.EntryID, actionResult)
	}
	if actionResult.ResultType == "quest_accept" {
		return h.handleNPCQuestAcceptAction(conn, packet.Seq, sess.PlayerID, target, request.EntryID, actionResult)
	}
	if actionResult.ResultType == "quest_submit" {
		return h.handleNPCQuestSubmitAction(conn, packet.Seq, sess.PlayerID, target, request.EntryID, actionResult)
	}
	if actionResult.ResultType == "dialogue" {
		return h.handleNPCDialogueStart(conn, packet.Seq, sess.PlayerID, target, request.EntryID)
	}
	if actionResult.ResultType == "shop" {
		return h.handleNPCShopOpen(conn, packet.Seq, sess.PlayerID, target, request.EntryID)
	}

	var questBefore []quest.Summary
	response, ok := h.buildNPCActionResponse(context.Background(), sess.PlayerID, target, request.EntryID, actionResult)
	if !ok {
		return h.sendNPCActionResponse(conn, packet.Seq, protocol.NPCActionResp{Accepted: false, Reason: "unsupported npc action", EntityID: request.EntityID, EntryID: request.EntryID})
	}
	if h.questService != nil {
		questBefore, _ = listQuestSummaries(context.Background(), h.questService, sess.PlayerID)
		_, _ = h.questService.HandleEvent(context.Background(), quest.Event{
			PlayerID:  sess.PlayerID,
			EventType: "TALK_TO_NPC",
			NPCID:     target.EntityID,
			Count:     1,
		})
	}
	if err := h.sendNPCActionResponse(conn, packet.Seq, response); err != nil {
		return err
	}
	if h.questService != nil && sess.PlayerID != 0 {
		_ = pushQuestDiff(context.Background(), conn, h.questService, sess.PlayerID, questBefore)
	}
	return nil
}

// HandleNPCDialogueNext 在客户端点击继续或本地剧情动画播放结束后推进到下一个剧情节点。
func (h *BattleHandler) HandleNPCDialogueNext(conn packetSender, packet *protocol.Packet) error {
	var request protocol.NPCDialogueNextReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid npc dialogue next body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	node, err := h.npcDialogueService.AdvanceDialogue(context.Background(), sess.PlayerID, request.EntityID, request.DialogueID, request.NodeID)
	if err != nil {
		return h.sendNPCDialogueResponse(conn, packet.Seq, protocol.NPCDialogueResp{Accepted: false, Reason: err.Error(), EntityID: request.EntityID})
	}
	_ = h.applyDialogueNodeSideEffects(context.Background(), conn, sess.PlayerID, request.EntityID, node)
	return h.sendNPCDialogueResponse(conn, packet.Seq, protocol.NPCDialogueResp{Accepted: true, Reason: "dialogue advanced", EntityID: request.EntityID, Node: h.toProtocolNPCDialogueNode(context.Background(), sess.PlayerID, node)})
}

// HandleNPCDialogueChoose 在客户端点选分支按钮后按服务端配置推进剧情。
func (h *BattleHandler) HandleNPCDialogueChoose(conn packetSender, packet *protocol.Packet) error {
	var request protocol.NPCDialogueChooseReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid npc dialogue choose body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	node, err := h.npcDialogueService.ChooseOption(context.Background(), sess.PlayerID, request.EntityID, request.DialogueID, request.NodeID, request.OptionID)
	if err != nil {
		return h.sendNPCDialogueResponse(conn, packet.Seq, protocol.NPCDialogueResp{Accepted: false, Reason: err.Error(), EntityID: request.EntityID})
	}
	_ = h.applyDialogueNodeSideEffects(context.Background(), conn, sess.PlayerID, request.EntityID, node)
	return h.sendNPCDialogueResponse(conn, packet.Seq, protocol.NPCDialogueResp{Accepted: true, Reason: "dialogue advanced", EntityID: request.EntityID, Node: h.toProtocolNPCDialogueNode(context.Background(), sess.PlayerID, node)})
}

func (h *BattleHandler) handleNPCBattleAction(conn packetSender, seq uint32, playerID uint64, profile *player.Profile, lineup []pet.LineupPet, target world.Entity, entryID string, actionResult *npc.ActionResult) error {
	encounterEntityID := actionResult.BattleEncounterEntityID
	if encounterEntityID == 0 {
		encounterEntityID = target.EntityID
	}
	enemy := target
	enemy.EntityID = encounterEntityID

	var questBefore []quest.Summary
	if h.questService != nil && playerID != 0 {
		questBefore, _ = listQuestSummaries(context.Background(), h.questService, playerID)
		_, _ = h.questService.HandleEvent(context.Background(), quest.Event{
			PlayerID:  playerID,
			EventType: "TALK_TO_NPC",
			NPCID:     target.EntityID,
			Count:     1,
		})
	}

	startSnapshot, err := h.battleService.StartPVE(context.Background(), profile, lineup, enemy, h.loadCharacterBattleSkillInput(context.Background(), playerID))
	if err != nil {
		reason := "battle start failed"
		switch {
		case errors.Is(err, battle.ErrBattleAlreadyActive):
			reason = "battle already active"
		case errors.Is(err, battle.ErrNoLineupAvailable):
			reason = "no lineup available"
		case errors.Is(err, battle.ErrTargetUnavailable):
			reason = "target unavailable"
		}
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{
			Accepted: false,
			Reason:   reason,
			EntityID: target.EntityID,
			EntryID:  entryID,
		})
	}

	response := protocol.NPCActionResp{
		Accepted:   true,
		Reason:     "battle started",
		EntityID:   target.EntityID,
		EntryID:    entryID,
		ResultType: "battle",
		Notice:     "battle started",
		NPCName:    target.Name,
	}
	if entries, ok := h.npcMenuEntriesByEntityID(context.Background(), playerID, target.EntityID); ok {
		response.MenuEntries = entries
	}
	if err := h.sendNPCActionResponse(conn, seq, response); err != nil {
		return err
	}
	if h.questService != nil && playerID != 0 {
		_ = pushQuestDiff(context.Background(), conn, h.questService, playerID, questBefore)
	}
	return h.pushBattleStartToParticipants(startSnapshot)
}

func (h *BattleHandler) requirePlayerID(connID string) (uint64, error) {
	sess, err := h.sessionService.GetByConnID(connID)
	if err != nil {
		return 0, err
	}
	return sess.PlayerID, nil
}

// playerIDByConn 根据连接标识读取玩家 ID，日志里用于定位是哪一位玩家触发了战斗请求。
func (h *BattleHandler) playerIDByConn(connID string) uint64 {
	if h == nil || h.sessionService == nil || connID == "" {
		return 0
	}
	playerID, err := h.requirePlayerID(connID)
	if err != nil {
		return 0
	}
	return playerID
}

func (h *BattleHandler) sendNPCActionResponse(conn packetSender, seq uint32, response protocol.NPCActionResp) error {
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdNPCActionResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdNPCActionResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

// sendNPCDialogueResponse 把剧情继续/分支选择的结果统一回给客户端对话面板。
func (h *BattleHandler) sendNPCDialogueResponse(conn packetSender, seq uint32, response protocol.NPCDialogueResp) error {
	logBattlePacket("resp", conn.ID(), h.playerIDByConn(conn.ID()), protocol.CmdNPCDialogueResp, seq, response)
	packet, err := protocol.NewJSONPacket(protocol.CmdNPCDialogueResp, seq, errcode.WSCodeSuccess, response)
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *BattleHandler) buildNPCMenuResponse(ctx context.Context, playerID uint64, target world.Entity) (protocol.NPCMenuResp, bool) {
	entries, ok := h.npcMenuEntriesByEntityID(ctx, playerID, target.EntityID)
	if !ok {
		return protocol.NPCMenuResp{}, false
	}
	return protocol.NPCMenuResp{
		EntityID:    target.EntityID,
		NPCName:     target.Name,
		MenuEntries: entries,
	}, true
}

func (h *BattleHandler) buildNPCActionResponse(ctx context.Context, playerID uint64, target world.Entity, entryID string, actionResult *npc.ActionResult) (protocol.NPCActionResp, bool) {
	if actionResult == nil {
		return protocol.NPCActionResp{}, false
	}
	base := protocol.NPCActionResp{
		Accepted:   true,
		EntityID:   target.EntityID,
		EntryID:    entryID,
		ResultType: actionResult.ResultType,
		Notice:     actionResult.Notice,
		NPCName:    target.Name,
	}
	entries, ok := h.npcMenuEntriesByEntityID(ctx, playerID, target.EntityID)
	if ok {
		base.MenuEntries = entries
	}
	return base, true
}

// BuildActiveDialogueReconnect 在断线重连时恢复玩家未结束的 NPC 剧情会话。
func (h *BattleHandler) BuildActiveDialogueReconnect(ctx context.Context, playerID uint64) (*protocol.ActiveDialogue, error) {
	if h == nil || h.npcDialogueService == nil || playerID == 0 {
		return nil, nil
	}
	entityID, node, err := h.npcDialogueService.GetActiveDialogue(ctx, playerID)
	if err != nil || node == nil || entityID == 0 {
		return nil, err
	}
	npcName := "NPC"
	if h.worldService != nil && h.playerService != nil {
		profile, profileErr := h.playerService.GetProfile(ctx, playerID)
		if profileErr == nil && profile != nil {
			sceneSnapshot, snapshotErr := h.worldService.GetSceneSnapshot(ctx, playerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
			if snapshotErr == nil {
				for _, entity := range sceneSnapshot.NearbyEntities {
					if entity.EntityID == entityID {
						npcName = entity.Name
						break
					}
				}
			}
		}
	}
	return &protocol.ActiveDialogue{
		EntityID: entityID,
		NPCName:  npcName,
		Node:     h.toProtocolNPCDialogueNode(ctx, playerID, node),
	}, nil
}

// handleNPCShopOpen 打开商店面板所需的服务端权威商品列表与钱包快照。
func (h *BattleHandler) handleNPCShopOpen(conn packetSender, seq uint32, playerID uint64, target world.Entity, entryID string) error {
	if h.npcService == nil {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: "shop unavailable", EntityID: target.EntityID, EntryID: entryID})
	}
	var questBefore []quest.Summary
	if h.questService != nil && playerID != 0 {
		questBefore, _ = listQuestSummaries(context.Background(), h.questService, playerID)
		_, _ = h.questService.HandleEvent(context.Background(), quest.Event{
			PlayerID:  playerID,
			EventType: "TALK_TO_NPC",
			NPCID:     target.EntityID,
			Count:     1,
		})
	}
	goods, err := h.npcService.ListShopGoodsByEntityID(context.Background(), target.EntityID)
	if err != nil {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: "shop load failed", EntityID: target.EntityID, EntryID: entryID})
	}
	if len(goods) == 0 {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: "shop has no goods", EntityID: target.EntityID, EntryID: entryID})
	}
	walletSnapshot, walletErr := h.walletService.GetRuntimeWallet(context.Background(), playerID)
	if walletErr != nil {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: "wallet unavailable", EntityID: target.EntityID, EntryID: entryID})
	}
	shopGoods := make([]protocol.NPCShopGood, 0, len(goods))
	for _, good := range goods {
		shopGoods = append(shopGoods, protocol.NPCShopGood{
			ItemID:      good.ItemID,
			ItemName:    good.ItemName,
			PriceCopper: good.BuyPriceCopper,
		})
	}
	response := protocol.NPCActionResp{
		Accepted:   true,
		Reason:     "shop opened",
		EntityID:   target.EntityID,
		EntryID:    entryID,
		ResultType: "shop",
		NPCName:    target.Name,
		Shop: &protocol.NPCShopPayload{
			Goods:  shopGoods,
			Wallet: toProtocolWalletSnapshot(*walletSnapshot),
		},
	}
	if entries, ok := h.npcMenuEntriesByEntityID(context.Background(), playerID, target.EntityID); ok {
		response.MenuEntries = entries
	}
	if err := h.sendNPCActionResponse(conn, seq, response); err != nil {
		return err
	}
	if h.questService != nil && playerID != 0 {
		_ = pushQuestDiff(context.Background(), conn, h.questService, playerID, questBefore)
	}
	return nil
}

// handleNPCDialogueStart 以 NPC 菜单项为入口开启结构化剧情，并把首个节点打包到 NPC_ACTION_RESP 中返回。
func (h *BattleHandler) handleNPCDialogueStart(conn packetSender, seq uint32, playerID uint64, target world.Entity, entryID string) error {
	if h.npcDialogueService == nil {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: "dialogue unavailable", EntityID: target.EntityID, EntryID: entryID})
	}
	var questBefore []quest.Summary
	if h.questService != nil && playerID != 0 {
		questBefore, _ = listQuestSummaries(context.Background(), h.questService, playerID)
		_, _ = h.questService.HandleEvent(context.Background(), quest.Event{
			PlayerID:  playerID,
			EventType: "TALK_TO_NPC",
			NPCID:     target.EntityID,
			Count:     1,
		})
	}
	node, err := h.npcDialogueService.StartDialogue(context.Background(), playerID, target.EntityID, entryID)
	if err != nil {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: err.Error(), EntityID: target.EntityID, EntryID: entryID})
	}
	_ = h.applyDialogueNodeSideEffects(context.Background(), conn, playerID, target.EntityID, node)
	response := protocol.NPCActionResp{
		Accepted:   true,
		Reason:     "dialogue started",
		EntityID:   target.EntityID,
		EntryID:    entryID,
		ResultType: "dialogue",
		NPCName:    target.Name,
		Dialogue:   h.toProtocolNPCDialogueNode(context.Background(), playerID, node),
	}
	if entries, ok := h.npcMenuEntriesByEntityID(context.Background(), playerID, target.EntityID); ok {
		response.MenuEntries = entries
	}
	if err := h.sendNPCActionResponse(conn, seq, response); err != nil {
		return err
	}
	if h.questService != nil && playerID != 0 {
		_ = pushQuestDiff(context.Background(), conn, h.questService, playerID, questBefore)
	}
	return nil
}

// toProtocolNPCDialogueNode 把服务端剧情模块的运行态节点转换成协议层结构，便于客户端直接消费。
func (h *BattleHandler) toProtocolNPCDialogueNode(ctx context.Context, playerID uint64, node *npcdialogue.RuntimeNode) *protocol.NPCDialogueNode {
	if node == nil {
		return nil
	}
	options := make([]protocol.NPCDialogueOption, 0, len(node.Options))
	for _, option := range node.Options {
		options = append(options, protocol.NPCDialogueOption{OptionID: option.OptionID, Text: option.OptionText, Format: option.OptionFormat})
	}
	playerName := h.resolveDialoguePlayerName(ctx, playerID)
	speaker, portraitKey, isPlayerSpeaker := resolveDialogueSpeaker(node.Speaker, node.PortraitKey, playerName)
	content, mentionedItems := h.renderDialogueContent(ctx, playerID, node.Content)
	return &protocol.NPCDialogueNode{
		DialogueID:           node.DialogueID,
		NodeID:               node.NodeID,
		NodeType:             node.NodeType,
		Speaker:              speaker,
		IsPlayerSpeaker:      isPlayerSpeaker,
		Content:              content,
		ContentFormat:        node.ContentFormat,
		PortraitKey:          portraitKey,
		ClientAnimationKey:   node.ClientAnimationKey,
		ClientAnimationBlock: node.ClientAnimationBlock,
		Options:              options,
		MentionedItems:       mentionedItems,
		IsEnd:                node.IsEnd,
		EffectNotice:         node.EffectNotice,
	}
}

// resolveDialoguePlayerName 返回当前玩家在剧情渲染中使用的展示名。
func (h *BattleHandler) resolveDialoguePlayerName(ctx context.Context, playerID uint64) string {
	playerName := ""
	if h != nil && h.playerService != nil && playerID > 0 {
		profile, err := h.playerService.GetProfile(ctx, playerID)
		if err == nil && profile != nil {
			playerName = profile.Name
		}
	}
	if playerName == "" && playerID > 0 {
		playerName = fmt.Sprintf("玩家%d", playerID)
	}
	return playerName
}

// renderDialogueContent renders safe server-side placeholders before the node
// reaches the client. Item names and icons always come from item_definition.
func (h *BattleHandler) renderDialogueContent(ctx context.Context, playerID uint64, content string) (string, []protocol.NPCDialogueItem) {
	rendered := content
	playerName := h.resolveDialoguePlayerName(ctx, playerID)
	rendered = strings.ReplaceAll(rendered, "{player_name}", playerName)
	rendered = strings.ReplaceAll(rendered, "{player_id}", strconv.FormatUint(playerID, 10))

	mentionedItems := make([]protocol.NPCDialogueItem, 0)
	seenItems := make(map[uint64]bool)
	rendered = dialogueItemTokenPattern.ReplaceAllStringFunc(rendered, func(token string) string {
		matches := dialogueItemTokenPattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		itemID, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil || itemID == 0 {
			return token
		}
		itemName := fmt.Sprintf("物品%d", itemID)
		if h != nil && h.itemService != nil {
			itemDetail, detailErr := h.itemService.GetRuntimeItemDetail(ctx, itemID)
			if detailErr == nil && itemDetail != nil {
				itemName = itemDetail.ItemName
			}
		}
		if !seenItems[itemID] {
			seenItems[itemID] = true
			mentionedItems = append(mentionedItems, protocol.NPCDialogueItem{
				ItemID:   itemID,
				ItemName: itemName,
			})
		}
		return itemName
	})
	rendered = dialoguePetTokenPattern.ReplaceAllStringFunc(rendered, func(token string) string {
		matches := dialoguePetTokenPattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		petID, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil || petID == 0 {
			return token
		}
		petName := fmt.Sprintf("宠物%d", petID)
		if h != nil && h.petService != nil {
			petDetail, detailErr := h.petService.GetAdminPetDefinitionDetail(ctx, uint32(petID))
			if detailErr == nil && petDetail != nil && strings.TrimSpace(petDetail.PetName) != "" {
				petName = petDetail.PetName
			}
		}
		return petName
	})
	return rendered, mentionedItems
}

// applyDialogueNodeSideEffects 在进入某个剧情节点后执行 effects_json 中配置的服务端权威副作用。
func (h *BattleHandler) applyDialogueNodeSideEffects(ctx context.Context, conn packetSender, playerID uint64, entityID uint64, node *npcdialogue.RuntimeNode) error {
	if node == nil {
		return nil
	}
	var questBefore []quest.Summary
	if h.questService != nil && playerID != 0 {
		questBefore, _ = listQuestSummaries(ctx, h.questService, playerID)
	}
	if node.EffectNotice != "" {
		_ = h.pushNoticeToPlayer(playerID, node.EffectNotice)
	}
	if h.questService != nil && playerID != 0 && node.EffectAcceptQuestID > 0 {
		_, err := h.questService.Accept(ctx, playerID, node.EffectAcceptQuestID, entityID)
		if err != nil && !errors.Is(err, quest.ErrQuestLocked) {
			return err
		}
	}
	if h.rewardService != nil && playerID != 0 && len(node.EffectGrantItems) > 0 {
		rewardEntries := make([]reward.Entry, 0, len(node.EffectGrantItems))
		for _, grantItem := range node.EffectGrantItems {
			rewardEntries = append(rewardEntries, reward.Entry{
				Type:   "item",
				ItemID: grantItem.ItemID,
				Count:  grantItem.Quantity,
			})
		}
		_, err := h.rewardService.GrantRuntimeRewards(ctx, reward.GrantInput{
			PlayerID:     playerID,
			ReasonType:   "npc_dialogue",
			ReasonRefID:  uint64(node.DialogueID),
			OperatorType: "system",
			OperatorID:   0,
			Rewards:      rewardEntries,
		})
		if err != nil {
			return err
		}
	}
	if h.questService != nil && playerID != 0 && node.EffectQuestEvent != "" {
		_, err := h.questService.HandleEvent(ctx, quest.Event{
			PlayerID:  playerID,
			EventType: node.EffectQuestEvent,
			NPCID:     entityID,
			Count:     1,
		})
		if err != nil {
			return err
		}
	}
	if h.questService != nil && playerID != 0 && conn != nil {
		_ = pushQuestDiff(ctx, conn, h.questService, playerID, questBefore)
	}
	return nil
}

func (h *BattleHandler) npcMenuEntriesByEntityID(ctx context.Context, playerID uint64, entityID uint64) ([]protocol.NpcMenuEntry, bool) {
	result := []protocol.NpcMenuEntry{}
	questReader := &npcdialogue.QuestServiceAdapter{Service: h.questService}
	if h.questService != nil && playerID != 0 {
		if summaries, err := listQuestSummaries(ctx, h.questService, playerID); err == nil {
			result = append(result, questMenuEntriesForNPC(entityID, summaries)...)
		}
	}

	staticEntries, err := h.npcService.ListMenuEntriesByEntityID(ctx, entityID)
	if err == nil {
		for _, entry := range staticEntries {
			visible, matchErr := npcdialogue.MatchNodeConditions(ctx, questReader, playerID, entry.ConditionsJSON)
			if matchErr != nil || !visible {
				continue
			}
			menuEntry := protocol.NpcMenuEntry{
				EntryID:   entry.EntryID,
				EntryType: entry.EntryType,
				Title:     entry.Title,
				Subtitle:  entry.Subtitle,
				State:     entry.State,
				Priority:  entry.Priority,
			}
			if entry.LinkedQuestID > 0 {
				menuEntry.QuestID = entry.LinkedQuestID
			}
			result = append(result, menuEntry)
		}
	}

	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

// handleNPCQuestAcceptAction 通过菜单项接取绑定的任务模板。
func (h *BattleHandler) handleNPCQuestAcceptAction(conn packetSender, seq uint32, playerID uint64, target world.Entity, entryID string, actionResult *npc.ActionResult) error {
	questID := actionResult.LinkedQuestID
	if questID == 0 || h.questService == nil || playerID == 0 {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: "quest unavailable", EntityID: target.EntityID, EntryID: entryID})
	}
	ctx := context.Background()
	questBefore, _ := listQuestSummaries(ctx, h.questService, playerID)
	summary, err := h.questService.Accept(ctx, playerID, questID, target.EntityID)
	if err != nil {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: err.Error(), EntityID: target.EntityID, EntryID: entryID, NPCName: target.Name})
	}
	response := protocol.NPCActionResp{
		Accepted:   true,
		Reason:     "quest accepted",
		EntityID:   target.EntityID,
		EntryID:    entryID,
		ResultType: "notice",
		Notice:     summary.Title,
		NPCName:    target.Name,
	}
	if entries, ok := h.npcMenuEntriesByEntityID(ctx, playerID, target.EntityID); ok {
		response.MenuEntries = entries
	}
	if err := h.sendNPCActionResponse(conn, seq, response); err != nil {
		return err
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, questBefore)
}

// handleNPCQuestSubmitAction 通过菜单项提交绑定的任务模板。
func (h *BattleHandler) handleNPCQuestSubmitAction(conn packetSender, seq uint32, playerID uint64, target world.Entity, entryID string, actionResult *npc.ActionResult) error {
	questID := actionResult.LinkedQuestID
	if questID == 0 || h.questService == nil || playerID == 0 {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: "quest unavailable", EntityID: target.EntityID, EntryID: entryID})
	}
	ctx := context.Background()
	questBefore, _ := listQuestSummaries(ctx, h.questService, playerID)
	result, err := h.questService.Submit(ctx, playerID, questID, target.EntityID)
	if err != nil {
		return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: err.Error(), EntityID: target.EntityID, EntryID: entryID, NPCName: target.Name})
	}
	if h.rewardService != nil && len(result.Rewards) > 0 {
		if _, grantErr := h.rewardService.GrantRuntimeRewards(ctx, reward.GrantInput{
			PlayerID:     playerID,
			ReasonType:   "quest_reward",
			ReasonRefID:  questID,
			OperatorType: "system",
			OperatorID:   playerID,
			Rewards:      toRuntimeRewardEntries(result.Rewards),
		}); grantErr != nil {
			return h.sendNPCActionResponse(conn, seq, protocol.NPCActionResp{Accepted: false, Reason: grantErr.Error(), EntityID: target.EntityID, EntryID: entryID, NPCName: target.Name})
		}
	}
	response := protocol.NPCActionResp{
		Accepted:   true,
		Reason:     "quest submitted",
		EntityID:   target.EntityID,
		EntryID:    entryID,
		ResultType: "notice",
		Notice:     result.Summary.Title,
		NPCName:    target.Name,
	}
	if entries, ok := h.npcMenuEntriesByEntityID(ctx, playerID, target.EntityID); ok {
		response.MenuEntries = entries
	}
	if err := h.sendNPCActionResponse(conn, seq, response); err != nil {
		return err
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, questBefore)
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
