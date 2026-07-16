package wstransport

import (
	"context"
	"errors"

	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/runtimeview"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/storyprogress"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/module/world"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type packetSender interface {
	ID() string
	SendPacket(packet *protocol.Packet) error
	Close() error
}

type WorldHandler struct {
	sessionService   *session.Service
	playerService    *player.Service
	petService       *pet.Service
	questService     *quest.Service
	walletService    *wallet.Service
	worldService     *world.Service
	monsterService   *monster.Service
	equipmentService *equipment.Service
	runtimeSnapshots *runtimeview.Service
	storyService     *storyprogress.Service
}

func NewWorldHandler(sessionService *session.Service, playerService *player.Service, petService *pet.Service, questService *quest.Service, walletService *wallet.Service, worldService *world.Service, monsterService *monster.Service, equipmentService *equipment.Service) *WorldHandler {
	return &WorldHandler{
		sessionService:   sessionService,
		playerService:    playerService,
		petService:       petService,
		questService:     questService,
		walletService:    walletService,
		worldService:     worldService,
		monsterService:   monsterService,
		equipmentService: equipmentService,
	}
}

// SetRuntimeSnapshotService 注入统一运行时快照刷新入口。
func (h *WorldHandler) SetRuntimeSnapshotService(service *runtimeview.Service) {
	if h == nil {
		return
	}
	h.runtimeSnapshots = service
}

// SetStoryProgressService 注入玩家剧情进度服务，用于场景进入时触发一次性过场。
func (h *WorldHandler) SetStoryProgressService(service *storyprogress.Service) {
	if h == nil {
		return
	}
	h.storyService = service
}

// BuildWorldSnapshotForPlayer reuses the same authority path as enter-world so
// reconnect can recover the current world view without duplicating scene logic.
func (h *WorldHandler) BuildWorldSnapshotForPlayer(ctx context.Context, playerID uint64) (*protocol.EnterWorldResp, error) {
	if h.runtimeSnapshots != nil {
		if err := h.runtimeSnapshots.RefreshPlayerRuntimeSnapshots(ctx, playerID); err != nil {
			return nil, err
		}
	}
	profile, err := h.playerService.GetBattleReadyProfile(ctx, playerID)
	if err != nil {
		return nil, err
	}
	lineup, err := h.petService.ListLineup(ctx, playerID)
	if err != nil {
		return nil, err
	}
	snapshot, err := h.worldService.GetSceneSnapshot(ctx, playerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
	if err != nil {
		// 当玩家档案中的 scene_id 已失效时，服务端直接把人物修正到默认市场，
		// 避免客户端因为拿不到世界快照而一直停留在 ERROR_PUSH。
		if errors.Is(err, world.ErrSnapshotUnavailable) {
			fallbackPos := world.FallbackSpawnPos()
			if updateErr := h.playerService.UpdatePosition(ctx, playerID, world.FallbackSceneID, fallbackPos.X, fallbackPos.Y); updateErr != nil {
				return nil, updateErr
			}
			profile.SceneID = world.FallbackSceneID
			profile.PosX = fallbackPos.X
			profile.PosY = fallbackPos.Y
			snapshot, err = h.worldService.GetSceneSnapshot(ctx, playerID, profile.SceneID, fallbackPos)
		}
		if err != nil {
			return nil, err
		}
	}
	legacyGold := profile.Gold
	if h.walletService != nil {
		walletSnapshot, err := h.walletService.GetRuntimeWallet(ctx, playerID)
		if err != nil {
			return nil, err
		}
		legacyGold = legacyGoldFromWalletSnapshot(walletSnapshot)
	}
	wildEncounter, err := h.loadWildEncounterConfig(ctx, snapshot.SceneID)
	if err != nil {
		return nil, err
	}
	playerSnapshot := toProtocolPlayerSnapshot(profile)
	playerSnapshot.Gold = legacyGold
	if h.equipmentService != nil {
		equippedItems, err := h.equipmentService.ListEquipped(ctx, playerID)
		if err != nil {
			return nil, err
		}
		playerSnapshot.EquippedItems = toProtocolEquippedItems(equippedItems)
	}
	return &protocol.EnterWorldResp{
		Self: protocol.PlayerBrief{
			PlayerID: profile.PlayerID,
			Name:     profile.Name,
			Level:    profile.Level,
		},
		Player:         playerSnapshot,
		SceneID:        snapshot.SceneID,
		SelfPos:        protocol.Vec2i{X: snapshot.SelfPos.X, Y: snapshot.SelfPos.Y},
		SceneVersion:   snapshot.SceneVersion,
		NearbyEntities: toProtocolEntities(snapshot.NearbyEntities),
		Lineup:         toProtocolLineup(lineup, h.petService.ResolveSkinID),
		Gold:           legacyGold,
		WildEncounter:  wildEncounter,
	}, nil
}

func (h *WorldHandler) HandleEnterWorld(conn packetSender, packet *protocol.Packet) error {
	var request protocol.EnterWorldReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid enter world body")
	}

	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	ctx := context.Background()
	responseBody, err := h.BuildWorldSnapshotForPlayer(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load scene snapshot failed", err)
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdEnterWorldResp, packet.Seq, errcode.WSCodeSuccess, responseBody)
	if err != nil {
		return err
	}
	var questBefore []quest.Summary
	if h.questService != nil {
		questBefore, _ = listQuestSummaries(ctx, h.questService, sess.PlayerID)
		_, _ = h.questService.HandleEvent(ctx, quest.Event{
			PlayerID:  sess.PlayerID,
			EventType: "ENTER_SCENE",
			SceneID:   responseBody.SceneID,
			Count:     1,
		})
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	if h.questService != nil {
		_ = pushQuestDiff(ctx, conn, h.questService, sess.PlayerID, questBefore)
	}
	return h.pushPendingSceneTrigger(ctx, conn, sess.PlayerID, responseBody.SceneID)
}

func (h *WorldHandler) HandleMoveIntent(conn packetSender, packet *protocol.Packet) error {
	var request protocol.MoveIntentReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid move intent body")
	}

	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}

	ctx := context.Background()
	profile, err := h.playerService.GetProfile(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "player not found")
	}

	currentPos := world.Vec2i{X: profile.PosX, Y: profile.PosY}
	if request.SceneID != profile.SceneID {
		return h.sendMoveRejectedWithResync(conn, packet.Seq, request.MoveSeq, profile.SceneID, currentPos, "scene mismatch")
	}

	// In-scene movement is client-authoritative now; the server only tracks scene/map transfers.
	if request.TargetSceneID == 0 || request.TargetSceneID == profile.SceneID {
		responsePacket, err := protocol.NewJSONPacket(protocol.CmdMoveIntentResp, packet.Seq, errcode.WSCodeSuccess, protocol.MoveIntentResp{
			Accepted:     true,
			MoveSeq:      request.MoveSeq,
			SceneID:      profile.SceneID,
			CorrectedPos: protocol.Vec2i{X: currentPos.X, Y: currentPos.Y},
			Reason:       "local movement handled by client",
		})
		if err != nil {
			return err
		}
		return conn.SendPacket(responsePacket)
	}

	decision, err := h.worldService.EvaluateTransfer(ctx, sess.PlayerID, request.SceneID, currentPos, request.TargetSceneID, request.PortalID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldMoveFailed, "evaluate move failed")
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdMoveIntentResp, packet.Seq, errcode.WSCodeSuccess, protocol.MoveIntentResp{
		Accepted:     decision.Accepted,
		MoveSeq:      request.MoveSeq,
		SceneID:      decision.ToSceneID,
		CorrectedPos: protocol.Vec2i{X: decision.SpawnPos.X, Y: decision.SpawnPos.Y},
		Reason:       decision.Reason,
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}

	if !decision.Accepted {
		return h.sendWorldResync(conn, profile.SceneID, currentPos)
	}

	if err := h.playerService.UpdatePosition(ctx, sess.PlayerID, decision.ToSceneID, decision.SpawnPos.X, decision.SpawnPos.Y); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldMoveFailed, "update player position failed")
	}
	if h.questService != nil {
		questBefore, _ := listQuestSummaries(ctx, h.questService, sess.PlayerID)
		_, _ = h.questService.HandleEvent(ctx, quest.Event{
			PlayerID:  sess.PlayerID,
			EventType: "ENTER_SCENE",
			SceneID:   decision.ToSceneID,
			Count:     1,
		})
		if err := h.sendWorldResync(conn, decision.ToSceneID, decision.SpawnPos); err != nil {
			return err
		}
		_ = pushQuestDiff(ctx, conn, h.questService, sess.PlayerID, questBefore)
		return h.pushPendingSceneTrigger(ctx, conn, sess.PlayerID, decision.ToSceneID)
	}
	if err := h.sendWorldResync(conn, decision.ToSceneID, decision.SpawnPos); err != nil {
		return err
	}
	return h.pushPendingSceneTrigger(ctx, conn, sess.PlayerID, decision.ToSceneID)
}

// HandleSceneTriggerAck 在客户端播放完服务端触发的场景剧情后落库剧情进度，并执行解锁 NPC / 接取任务等副作用。
func (h *WorldHandler) HandleSceneTriggerAck(conn packetSender, packet *protocol.Packet) error {
	var request protocol.SceneTriggerAckReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid scene trigger ack body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.storyService == nil {
		return h.sendSceneTriggerAck(conn, packet.Seq, false, "scene trigger unavailable", request.TriggerCode)
	}

	ctx := context.Background()
	questBefore := []quest.Summary{}
	if h.questService != nil {
		questBefore, _ = listQuestSummaries(ctx, h.questService, sess.PlayerID)
	}
	trigger, err := h.storyService.CompleteSceneTrigger(ctx, sess.PlayerID, request.TriggerCode)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldMoveFailed, "complete scene trigger failed", err)
	}
	if trigger == nil {
		return h.sendSceneTriggerAck(conn, packet.Seq, false, "scene trigger not found", request.TriggerCode)
	}
	if h.questService != nil && trigger.EffectAcceptQuestID > 0 {
		_, err := h.questService.Accept(ctx, sess.PlayerID, trigger.EffectAcceptQuestID, 92001)
		if err != nil && !errors.Is(err, quest.ErrQuestLocked) {
			return sendError(conn, packet.Seq, errcode.WSCodeWorldMoveFailed, "accept scene trigger quest failed", err)
		}
	}
	if err := h.sendSceneTriggerAck(conn, packet.Seq, true, "scene trigger completed", request.TriggerCode); err != nil {
		return err
	}
	profile, err := h.playerService.GetProfile(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "player not found")
	}
	if err := h.sendWorldResync(conn, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY}); err != nil {
		return err
	}
	if h.questService != nil {
		_ = pushQuestDiff(ctx, conn, h.questService, sess.PlayerID, questBefore)
	}
	return nil
}

func (h *WorldHandler) sendSceneTriggerAck(conn packetSender, seq uint32, accepted bool, reason string, triggerCode string) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdSceneTriggerAckResp, seq, errcode.WSCodeSuccess, protocol.SceneTriggerAckResp{
		Accepted:    accepted,
		Reason:      reason,
		TriggerCode: triggerCode,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *WorldHandler) pushPendingSceneTrigger(ctx context.Context, conn packetSender, playerID uint64, sceneID uint32) error {
	if h.storyService == nil || conn == nil || playerID == 0 || sceneID == 0 {
		return nil
	}
	trigger, err := h.storyService.EvaluateSceneEntry(ctx, playerID, sceneID)
	if err != nil {
		return err
	}
	if trigger == nil {
		return nil
	}
	packet, err := protocol.NewJSONPacket(protocol.CmdSceneTriggerPush, 0, errcode.WSCodeSuccess, protocol.SceneTriggerPush{
		TriggerCode:        trigger.TriggerCode,
		SceneID:            trigger.SceneID,
		ClientAnimationKey: trigger.ClientAnimationKey,
		PromptText:         trigger.PromptText,
		BlockMovement:      trigger.BlockMovement,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func toProtocolEntities(entities []world.Entity) []protocol.EntityBrief {
	if len(entities) == 0 {
		return []protocol.EntityBrief{}
	}
	result := make([]protocol.EntityBrief, 0, len(entities))
	for _, entity := range entities {
		result = append(result, protocol.EntityBrief{
			EntityID:   entity.EntityID,
			PlayerID:   entity.PlayerID,
			EntityType: entity.EntityType,
			Pos:        protocol.Vec2i{X: entity.Pos.X, Y: entity.Pos.Y},
			Dir:        entity.Dir,
			Speed:      entity.Speed,
			Name:       entity.Name,
		})
	}
	return result
}

func toProtocolLineup(lineup []pet.LineupPet, resolveSkinID func(petID uint32) string) []protocol.PetBrief {
	if len(lineup) == 0 {
		return []protocol.PetBrief{}
	}
	result := make([]protocol.PetBrief, 0, len(lineup))
	for _, lineupPet := range lineup {
		result = append(result, protocol.PetBrief{
			PetUID: lineupPet.PetUID,
			PetID:  lineupPet.PetID,
			Level:  lineupPet.Level,
			HP:     lineupPet.HP,
			HPMax:  lineupPet.HPMax,
			SkinID: resolveSkinID(lineupPet.PetID),
		})
	}
	return result
}

func (h *WorldHandler) sendMoveRejectedWithResync(conn packetSender, seq uint32, moveSeq uint32, sceneID uint32, currentPos world.Vec2i, reason string) error {
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdMoveIntentResp, seq, errcode.WSCodeSuccess, protocol.MoveIntentResp{
		Accepted:     false,
		MoveSeq:      moveSeq,
		SceneID:      sceneID,
		CorrectedPos: protocol.Vec2i{X: currentPos.X, Y: currentPos.Y},
		Reason:       reason,
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	return h.sendWorldResync(conn, sceneID, currentPos)
}

func (h *WorldHandler) sendWorldResync(conn packetSender, sceneID uint32, selfPos world.Vec2i) error {
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, 0, errcode.WSCodeSessionInvalid, "session invalid")
	}

	snapshot, err := h.worldService.GetSceneSnapshot(context.Background(), sess.PlayerID, sceneID, selfPos)
	if err != nil {
		return sendError(conn, 0, errcode.WSCodeWorldMoveFailed, "load scene snapshot failed")
	}
	wildEncounter, err := h.loadWildEncounterConfig(context.Background(), snapshot.SceneID)
	if err != nil {
		return sendError(conn, 0, errcode.WSCodeWorldMoveFailed, "load wild encounter config failed")
	}

	packet, err := protocol.NewJSONPacket(protocol.CmdWorldResyncPush, 0, errcode.WSCodeSuccess, protocol.WorldResyncPush{
		SceneID:        snapshot.SceneID,
		SelfPos:        protocol.Vec2i{X: snapshot.SelfPos.X, Y: snapshot.SelfPos.Y},
		SceneVersion:   snapshot.SceneVersion,
		NearbyEntities: toProtocolEntities(snapshot.NearbyEntities),
		WildEncounter:  wildEncounter,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}

func (h *WorldHandler) loadWildEncounterConfig(ctx context.Context, sceneID uint32) (protocol.WildEncounterConfig, error) {
	if h.monsterService == nil || sceneID == 0 {
		return protocol.WildEncounterConfig{SceneID: sceneID}, nil
	}
	config, err := h.monsterService.BuildWildEncounterConfig(ctx, sceneID)
	if err != nil {
		return protocol.WildEncounterConfig{}, err
	}
	return toProtocolWildEncounterConfig(config), nil
}

func toProtocolWildEncounterConfig(config *monster.RuntimeWildEncounterConfig) protocol.WildEncounterConfig {
	if config == nil {
		return protocol.WildEncounterConfig{}
	}
	spawnMonsterIDs := append([]uint32{}, config.SpawnMonsterIDs...)
	return protocol.WildEncounterConfig{
		Enabled:         config.Enabled,
		SceneID:         config.SceneID,
		EncounterRate:   config.EncounterRate,
		SpawnMonsterIDs: spawnMonsterIDs,
	}
}
