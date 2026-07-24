package wstransport

import (
	"context"
	"errors"
	"strings"
	"sync"

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
	presenceMu       sync.RWMutex
	playerScenes     map[uint64]uint32
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
		playerScenes:     make(map[uint64]uint32),
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
	nearbyEntities := toProtocolEntities(snapshot.NearbyEntities)
	nearbyEntities = h.appendOnlinePlayerEntities(ctx, playerID, snapshot.SceneID, nearbyEntities)
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
		NearbyEntities: nearbyEntities,
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
	h.enterPlayerScene(ctx, sess.PlayerID, responseBody.SceneID)
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

	// 新客户端会持续上报当前格子坐标；服务端先持久化，再把权威结果广播给同场景其他玩家。
	// 旧客户端没有 target_pos 时仍维持原有响应，避免破坏已有切图协议兼容性。
	if request.TargetSceneID == 0 || request.TargetSceneID == profile.SceneID {
		correctedPos := currentPos
		reason := "local movement handled by client"
		if request.TargetPos != nil {
			correctedPos = world.Vec2i{X: request.TargetPos.X, Y: request.TargetPos.Y}
			if correctedPos != currentPos {
				if err := h.playerService.UpdatePosition(ctx, sess.PlayerID, profile.SceneID, correctedPos.X, correctedPos.Y); err != nil {
					return sendError(conn, packet.Seq, errcode.WSCodeWorldMoveFailed, "update player position failed")
				}
			}
			reason = "position synchronized"
		}
		responsePacket, err := protocol.NewJSONPacket(protocol.CmdMoveIntentResp, packet.Seq, errcode.WSCodeSuccess, protocol.MoveIntentResp{
			Accepted:     true,
			MoveSeq:      request.MoveSeq,
			SceneID:      profile.SceneID,
			CorrectedPos: protocol.Vec2i{X: correctedPos.X, Y: correctedPos.Y},
			Reason:       reason,
		})
		if err != nil {
			return err
		}
		if err := conn.SendPacket(responsePacket); err != nil {
			return err
		}
		if request.TargetPos != nil {
			h.enterPlayerScene(ctx, sess.PlayerID, profile.SceneID)
			h.broadcastEntityMove(
				profile.SceneID,
				sess.PlayerID,
				request.MoveSeq,
				currentPos,
				correctedPos,
				normalizePreciseMovementPosition(correctedPos, request.PrecisePos),
				normalizeMovementFacing(currentPos, correctedPos, request.Facing),
				normalizeMovementState(currentPos, correctedPos, request.Moving),
			)
		}
		return nil
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
	h.transferPlayerScene(ctx, sess.PlayerID, profile.SceneID, decision.ToSceneID)
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

// HandleSessionDisconnect 从在线场景索引移除玩家，并通知原场景剩余客户端删除远端角色。
func (h *WorldHandler) HandleSessionDisconnect(playerID uint64) {
	h.presenceMu.Lock()
	sceneID, exists := h.playerScenes[playerID]
	delete(h.playerScenes, playerID)
	h.presenceMu.Unlock()
	if !exists {
		return
	}
	h.broadcastEntityLeave(sceneID, playerID)
}

// appendOnlinePlayerEntities 把进程内已进入世界的同场景玩家合并到数据库场景实体快照。
func (h *WorldHandler) appendOnlinePlayerEntities(ctx context.Context, selfPlayerID uint64, sceneID uint32, entities []protocol.EntityBrief) []protocol.EntityBrief {
	seenEntityIDs := make(map[uint64]struct{}, len(entities))
	for index := range entities {
		entity := entities[index]
		seenEntityIDs[entity.EntityID] = struct{}{}
		if entity.EntityType == 1 && entity.PlayerID != 0 && entity.PlayerID != selfPlayerID {
			entities[index] = h.attachFollowingPet(ctx, entity)
		}
	}
	for _, playerID := range h.playerIDsInScene(sceneID, selfPlayerID) {
		if _, exists := seenEntityIDs[playerID]; exists {
			continue
		}
		profile, err := h.playerService.GetProfile(ctx, playerID)
		if err != nil {
			continue
		}
		entities = append(entities, h.playerEntity(ctx, profile))
		seenEntityIDs[playerID] = struct{}{}
	}
	return entities
}

// enterPlayerScene 记录在线场景，并仅在首次进入或切换场景时向其他玩家广播实体进入。
func (h *WorldHandler) enterPlayerScene(ctx context.Context, playerID uint64, sceneID uint32) {
	h.presenceMu.Lock()
	previousSceneID, existed := h.playerScenes[playerID]
	h.playerScenes[playerID] = sceneID
	h.presenceMu.Unlock()
	if existed && previousSceneID == sceneID {
		return
	}
	profile, err := h.playerService.GetProfile(ctx, playerID)
	if err != nil {
		return
	}
	packet, err := protocol.NewJSONPacket(protocol.CmdEntityEnterPush, 0, errcode.WSCodeSuccess, protocol.EntityEnterPush{
		SceneID: sceneID,
		Entity:  h.playerEntity(ctx, profile),
	})
	if err == nil {
		h.sendPacketToScene(sceneID, playerID, packet)
	}
}

// transferPlayerScene 广播旧场景离开后登记新场景，保证两边客户端实体列表同步。
func (h *WorldHandler) transferPlayerScene(ctx context.Context, playerID uint64, fromSceneID uint32, toSceneID uint32) {
	h.presenceMu.Lock()
	delete(h.playerScenes, playerID)
	h.presenceMu.Unlock()
	h.broadcastEntityLeave(fromSceneID, playerID)
	h.enterPlayerScene(ctx, playerID, toSceneID)
}

// broadcastEntityMove 把持久化后的坐标发给同场景其他玩家，发送失败不反向中断移动者连接。
func (h *WorldHandler) broadcastEntityMove(sceneID uint32, playerID uint64, moveSeq uint32, fromPos world.Vec2i, toPos world.Vec2i, precisePos protocol.Vec2i, facing protocol.Vec2i, moving bool) {
	packet, err := protocol.NewJSONPacket(protocol.CmdEntityMovePush, 0, errcode.WSCodeSuccess, protocol.EntityMovePush{
		SceneID:      sceneID,
		SceneVersion: 1,
		EntityID:     playerID,
		MoveSeq:      moveSeq,
		FromPos:      protocol.Vec2i{X: fromPos.X, Y: fromPos.Y},
		ToPos:        protocol.Vec2i{X: toPos.X, Y: toPos.Y},
		PrecisePos:   precisePos,
		Facing:       facing,
		Moving:       moving,
	})
	if err == nil {
		h.sendPacketToScene(sceneID, playerID, packet)
	}
}

const movementPositionFixedScale int32 = 1000
const movementPositionHalfCell int32 = movementPositionFixedScale / 2

// normalizePreciseMovementPosition 把客户端表现坐标限制在权威整数格周围半格，阻止表现位置脱离持久化落点。
func normalizePreciseMovementPosition(targetPos world.Vec2i, precisePos *protocol.Vec2i) protocol.Vec2i {
	targetX := targetPos.X * movementPositionFixedScale
	targetY := targetPos.Y * movementPositionFixedScale
	if precisePos == nil {
		return protocol.Vec2i{X: targetX, Y: targetY}
	}
	return protocol.Vec2i{
		X: clampMovementCoordinate(precisePos.X, targetX-movementPositionHalfCell, targetX+movementPositionHalfCell),
		Y: clampMovementCoordinate(precisePos.Y, targetY-movementPositionHalfCell, targetY+movementPositionHalfCell),
	}
}

// clampMovementCoordinate 将一个定点坐标限制到服务端允许的闭区间。
func clampMovementCoordinate(value int32, minimum int32, maximum int32) int32 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

// normalizeMovementFacing 只接受四方向单位向量，旧客户端未提供时按本次整数格位移推导。
func normalizeMovementFacing(fromPos world.Vec2i, toPos world.Vec2i, facing *protocol.Vec2i) protocol.Vec2i {
	if facing != nil {
		if (facing.X == -1 || facing.X == 1) && facing.Y == 0 {
			return *facing
		}
		if (facing.Y == -1 || facing.Y == 1) && facing.X == 0 {
			return *facing
		}
	}
	offsetX := toPos.X - fromPos.X
	offsetY := toPos.Y - fromPos.Y
	if offsetX != 0 {
		if offsetX < 0 {
			return protocol.Vec2i{X: -1}
		}
		return protocol.Vec2i{X: 1}
	}
	if offsetY < 0 {
		return protocol.Vec2i{Y: -1}
	}
	return protocol.Vec2i{Y: 1}
}

// normalizeMovementState 保留新客户端明确上报的起停状态，旧客户端则按整数格是否变化兼容推导。
func normalizeMovementState(fromPos world.Vec2i, toPos world.Vec2i, moving *bool) bool {
	if moving != nil {
		return *moving
	}
	return fromPos != toPos
}

// broadcastEntityLeave 通知指定场景中的其他在线玩家移除目标角色。
func (h *WorldHandler) broadcastEntityLeave(sceneID uint32, playerID uint64) {
	packet, err := protocol.NewJSONPacket(protocol.CmdEntityLeavePush, 0, errcode.WSCodeSuccess, protocol.EntityLeavePush{
		SceneID:  sceneID,
		EntityID: playerID,
	})
	if err == nil {
		h.sendPacketToScene(sceneID, playerID, packet)
	}
}

// sendPacketToScene 按在线场景索引筛选接收者，避免把坐标泄漏给其他地图玩家。
func (h *WorldHandler) sendPacketToScene(sceneID uint32, excludedPlayerID uint64, packet *protocol.Packet) {
	for _, playerID := range h.playerIDsInScene(sceneID, excludedPlayerID) {
		sess, err := h.sessionService.GetByPlayerID(playerID)
		if err != nil || sess.Conn == nil {
			continue
		}
		_ = sess.Conn.SendPacket(packet)
	}
}

// playerIDsInScene 返回场景内玩家 ID 快照，避免发送网络包时长期持有在线索引锁。
func (h *WorldHandler) playerIDsInScene(sceneID uint32, excludedPlayerID uint64) []uint64 {
	h.presenceMu.RLock()
	defer h.presenceMu.RUnlock()
	result := make([]uint64, 0)
	for playerID, currentSceneID := range h.playerScenes {
		if currentSceneID == sceneID && playerID != excludedPlayerID {
			result = append(result, playerID)
		}
	}
	return result
}

// playerEntity 把权威玩家档案和首只出战宠物转换成客户端远端世界表现摘要。
func (h *WorldHandler) playerEntity(ctx context.Context, profile *player.Profile) protocol.EntityBrief {
	entity := protocol.EntityBrief{
		EntityID:   profile.PlayerID,
		PlayerID:   profile.PlayerID,
		EntityType: 1,
		Pos:        protocol.Vec2i{X: profile.PosX, Y: profile.PosY},
		Name:       profile.Name,
	}
	return h.attachFollowingPet(ctx, entity)
}

// attachFollowingPet 从持久化编队读取首只宠物，并补充世界展示需要的实例与形象信息。
func (h *WorldHandler) attachFollowingPet(ctx context.Context, entity protocol.EntityBrief) protocol.EntityBrief {
	if h.petService == nil || entity.PlayerID == 0 {
		return entity
	}
	lineup, err := h.petService.ListLineupSummaries(ctx, entity.PlayerID)
	if err != nil || len(lineup) == 0 {
		return entity
	}
	petBrief := toProtocolPetBrief(lineup[0], h.petService.ResolveSkinID)
	if petBrief.PetUID == 0 || petBrief.SkinID == "" {
		return entity
	}
	entity.FollowingPet = &petBrief
	return entity
}

// BroadcastPlayerEntityRefresh 在玩家编队变化后向同场景其他客户端刷新角色与跟随宠物摘要。
func (h *WorldHandler) BroadcastPlayerEntityRefresh(ctx context.Context, playerID uint64) {
	h.presenceMu.RLock()
	sceneID, exists := h.playerScenes[playerID]
	h.presenceMu.RUnlock()
	if !exists {
		return
	}
	profile, err := h.playerService.GetProfile(ctx, playerID)
	if err != nil {
		return
	}
	packet, err := protocol.NewJSONPacket(protocol.CmdEntityEnterPush, 0, errcode.WSCodeSuccess, protocol.EntityEnterPush{
		SceneID: sceneID,
		Entity:  h.playerEntity(ctx, profile),
	})
	if err == nil {
		h.sendPacketToScene(sceneID, playerID, packet)
	}
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
	// 纯地图提示没有剧情动画和服务端副作用，首次进入时立即持久化一次性标记。
	// 这样即使客户端尚未关闭提示便重复同步世界，也不会再次收到同一地图提示。
	if isPromptOnlySceneTrigger(trigger) {
		completedTrigger, completeErr := h.storyService.CompleteSceneTrigger(ctx, playerID, trigger.TriggerCode)
		if completeErr != nil {
			return completeErr
		}
		if completedTrigger == nil {
			return nil
		}
		trigger = completedTrigger
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

// isPromptOnlySceneTrigger 判断触发器是否只是首次进图提示。
// 带动画、任务接取或剧情标记副作用的触发器仍必须等待客户端播放完成后 Ack，避免提前推进剧情。
func isPromptOnlySceneTrigger(trigger *storyprogress.SceneTrigger) bool {
	if trigger == nil || strings.TrimSpace(trigger.PromptText) == "" {
		return false
	}
	return strings.TrimSpace(trigger.ClientAnimationKey) == "" &&
		trigger.EffectAcceptQuestID == 0 &&
		len(trigger.EffectSetFlags) == 0
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
		result = append(result, toProtocolPetBrief(lineupPet, resolveSkinID))
	}
	return result
}

// toProtocolPetBrief 统一转换宠物摘要，保证自身编队和远端跟随宠物使用同一协议口径。
func toProtocolPetBrief(lineupPet pet.LineupPet, resolveSkinID func(petID uint32) string) protocol.PetBrief {
	return protocol.PetBrief{
		PetUID: lineupPet.PetUID,
		PetID:  lineupPet.PetID,
		Level:  lineupPet.Level,
		HP:     lineupPet.HP,
		HPMax:  lineupPet.HPMax,
		SkinID: resolveSkinID(lineupPet.PetID),
	}
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
		NearbyEntities: h.appendOnlinePlayerEntities(context.Background(), sess.PlayerID, snapshot.SceneID, toProtocolEntities(snapshot.NearbyEntities)),
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
