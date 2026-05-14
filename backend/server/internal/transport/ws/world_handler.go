package wstransport

import (
	"context"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
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
	worldService   *world.Service
}

func NewWorldHandler(sessionService *session.Service, playerService *player.Service, petService *pet.Service, worldService *world.Service) *WorldHandler {
	return &WorldHandler{
		sessionService: sessionService,
		playerService:  playerService,
		petService:     petService,
		worldService:   worldService,
	}
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
	profile, err := h.playerService.GetProfile(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "player not found")
	}

	lineup, err := h.petService.ListLineup(ctx, sess.PlayerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load pet lineup failed")
	}

	snapshot, err := h.worldService.GetSceneSnapshot(ctx, sess.PlayerID, profile.SceneID, world.Vec2i{X: profile.PosX, Y: profile.PosY})
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load scene snapshot failed")
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdEnterWorldResp, packet.Seq, errcode.WSCodeSuccess, protocol.EnterWorldResp{
		Self: protocol.PlayerBrief{
			PlayerID: profile.PlayerID,
			Name:     profile.Name,
			Level:    profile.Level,
		},
		SceneID:        snapshot.SceneID,
		SelfPos:        protocol.Vec2i{X: snapshot.SelfPos.X, Y: snapshot.SelfPos.Y},
		SceneVersion:   snapshot.SceneVersion,
		NearbyEntities: toProtocolEntities(snapshot.NearbyEntities),
		Lineup:         toProtocolLineup(lineup),
		Gold:           profile.Gold,
	})
	if err != nil {
		return err
	}

	return conn.SendPacket(responsePacket)
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

	decision, err := h.worldService.EvaluateMove(ctx, sess.PlayerID, request.SceneID, currentPos, world.Vec2i{
		X: request.TargetPos.X,
		Y: request.TargetPos.Y,
	})
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldMoveFailed, "evaluate move failed")
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdMoveIntentResp, packet.Seq, errcode.WSCodeSuccess, protocol.MoveIntentResp{
		Accepted:     decision.Accepted,
		MoveSeq:      request.MoveSeq,
		CorrectedPos: protocol.Vec2i{X: decision.CorrectedPos.X, Y: decision.CorrectedPos.Y},
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

	if err := h.playerService.UpdatePosition(ctx, sess.PlayerID, profile.SceneID, decision.ToPos.X, decision.ToPos.Y); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldMoveFailed, "update player position failed")
	}

	movePushPacket, err := protocol.NewJSONPacket(protocol.CmdEntityMovePush, 0, errcode.WSCodeSuccess, protocol.EntityMovePush{
		SceneVersion: decision.SceneVersion,
		EntityID:     sess.PlayerID,
		MoveSeq:      request.MoveSeq,
		FromPos:      protocol.Vec2i{X: decision.FromPos.X, Y: decision.FromPos.Y},
		ToPos:        protocol.Vec2i{X: decision.ToPos.X, Y: decision.ToPos.Y},
		Speed:        decision.Speed,
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(movePushPacket)
}

func toProtocolEntities(entities []world.Entity) []protocol.EntityBrief {
	if len(entities) == 0 {
		return []protocol.EntityBrief{}
	}
	result := make([]protocol.EntityBrief, 0, len(entities))
	for _, entity := range entities {
		result = append(result, protocol.EntityBrief{
			EntityID:   entity.EntityID,
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
