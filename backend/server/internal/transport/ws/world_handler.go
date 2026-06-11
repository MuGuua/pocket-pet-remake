package wstransport

import (
	"context"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/session"
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
	sessionService *session.Service
	playerService  *player.Service
	petService     *pet.Service
	questService   *quest.Service
	worldService   *world.Service
}

func NewWorldHandler(sessionService *session.Service, playerService *player.Service, petService *pet.Service, questService *quest.Service, worldService *world.Service) *WorldHandler {
	return &WorldHandler{
		sessionService: sessionService,
		playerService:  playerService,
		petService:     petService,
		questService:   questService,
		worldService:   worldService,
	}
}

// BuildWorldSnapshotForPlayer reuses the same authority path as enter-world so
// reconnect can recover the current world view without duplicating scene logic.
func (h *WorldHandler) BuildWorldSnapshotForPlayer(ctx context.Context, playerID uint64) (*protocol.EnterWorldResp, error) {
	profile, err := h.playerService.GetProfile(ctx, playerID)
	if err != nil {
		return nil, err
	}
	lineup, err := h.petService.ListLineup(ctx, playerID)
	if err != nil {
		return nil, err
	}
	snapshot, err := h.worldService.GetSceneSnapshot(ctx, playerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
	if err != nil {
		return nil, err
	}
	return &protocol.EnterWorldResp{
		Self: protocol.PlayerBrief{
			PlayerID: profile.PlayerID,
			Name:     profile.Name,
			Level:    profile.Level,
		},
		Player: protocol.PlayerSnapshot{
			PlayerID:           profile.PlayerID,
			Name:               profile.Name,
			Level:              profile.Level,
			Exp:                profile.Exp,
			Gold:               profile.Gold,
			HP:                 profile.HP,
			HPMax:              profile.HPMax,
			Energy:             profile.Energy,
			EnergyMax:          profile.EnergyMax,
			ATK:                profile.ATK,
			DEF:                profile.DEF,
			SPD:                profile.SPD,
			MANA:               profile.MANA,
			HitPct:             profile.HitPct,
			DodgePct:           profile.DodgePct,
			CritRatePct:        profile.CritRatePct,
			CritDmgPct:         profile.CritDmgPct,
			PhysicalResistPct:  profile.PhysicalResistPct,
			SkillResistPct:     profile.SkillResistPct,
			ConfusionResistPct: profile.ConfusionResistPct,
			SleepResistPct:     profile.SleepResistPct,
			ParalysisResistPct: profile.ParalysisResistPct,
			SealResistPct:      profile.SealResistPct,
			CurseResistPct:     profile.CurseResistPct,
			CritResistPct:      profile.CritResistPct,
			CritDmgResistPct:   profile.CritDmgResistPct,
			CharacterResistPct: profile.CharacterResistPct,
			PetResistPct:       profile.PetResistPct,
			MercenaryResistPct: profile.MercenaryResistPct,
			GenericShieldPct:   profile.GenericShieldPct,
			SkillIDs:           append([]uint32{}, profile.SkillIDs...),
		},
		SceneID:        snapshot.SceneID,
		SelfPos:        protocol.Vec2i{X: snapshot.SelfPos.X, Y: snapshot.SelfPos.Y},
		SceneVersion:   snapshot.SceneVersion,
		NearbyEntities: toProtocolEntities(snapshot.NearbyEntities),
		Lineup:         toProtocolLineup(lineup),
		Gold:           profile.Gold,
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
	return nil
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
		return nil
	}
	return h.sendWorldResync(conn, decision.ToSceneID, decision.SpawnPos)
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

func toProtocolLineup(lineup []pet.LineupPet) []protocol.PetBrief {
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

	packet, err := protocol.NewJSONPacket(protocol.CmdWorldResyncPush, 0, errcode.WSCodeSuccess, protocol.WorldResyncPush{
		SceneID:        snapshot.SceneID,
		SelfPos:        protocol.Vec2i{X: snapshot.SelfPos.X, Y: snapshot.SelfPos.Y},
		SceneVersion:   snapshot.SceneVersion,
		NearbyEntities: toProtocolEntities(snapshot.NearbyEntities),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(packet)
}
