package wstransport

import (
	"context"
	"errors"

	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
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
	battleService  *battle.Service
}

func NewBattleHandler(sessionService *session.Service, playerService *player.Service, petService *pet.Service, worldService *world.Service, battleService *battle.Service) *BattleHandler {
	return &BattleHandler{
		sessionService: sessionService,
		playerService:  playerService,
		petService:     petService,
		worldService:   worldService,
		battleService:  battleService,
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
	_ = sess

	target, found := findInteractTarget(sceneSnapshot.NearbyEntities, request.EntityID)
	if !found {
		return h.sendInteractResponse(conn, packet.Seq, false, "target unavailable")
	}

	startSnapshot, err := h.battleService.StartPVE(context.Background(), profile, lineup, target)
	if err != nil {
		if errors.Is(err, battle.ErrBattleAlreadyActive) {
			return h.sendInteractResponse(conn, packet.Seq, false, "battle already active")
		}
		if errors.Is(err, battle.ErrNoLineupAvailable) {
			return h.sendInteractResponse(conn, packet.Seq, false, "no lineup available")
		}
		return sendError(conn, packet.Seq, errcode.WSCodeBattleStartFailed, "battle start failed")
	}

	if err := h.sendInteractResponse(conn, packet.Seq, true, "battle started"); err != nil {
		return err
	}
	return conn.SendPacket(mustJSONPacket(protocol.CmdBattleStartPush, 0, protocol.BattleStartPush{
		BattleID:      startSnapshot.BattleID,
		BattleType:    startSnapshot.BattleType,
		BattleVersion: startSnapshot.BattleVersion,
		Allies:        toProtocolBattleActors(startSnapshot.Allies),
		Enemies:       toProtocolBattleActors(startSnapshot.Enemies),
		Round:         startSnapshot.Round,
		ActiveActorID: startSnapshot.ActiveActorID,
		ActivePetUID:  startSnapshot.ActivePetUID,
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
		BattleID:   request.BattleID,
		Round:      request.Round,
		ActionType: request.ActionType,
		ActorID:    request.ActorID,
		SkillID:    request.SkillID,
		TargetID:   request.TargetID,
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
	if outcome.State != nil {
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdBattleStatePush, 0, protocol.BattleStatePush{
			BattleID:      outcome.State.BattleID,
			BattleVersion: outcome.State.BattleVersion,
			Round:         outcome.State.Round,
			Events:        toProtocolBattleEvents(outcome.State.Events),
			Actors:        toProtocolBattleActorStates(outcome.State.Actors),
			ActiveActorID: outcome.State.ActiveActorID,
			ActivePetUID:  outcome.State.ActivePetUID,
		})); err != nil {
			return err
		}
	}
	if outcome.Result != nil {
		updatedPet, err := h.petService.UpdatePetHP(context.Background(), sess.PlayerID, outcome.Result.ActivePetUID, outcome.Result.ActivePetHP)
		if err != nil {
			return err
		}
		if err := conn.SendPacket(mustJSONPacket(protocol.CmdBattleResultPush, 0, protocol.BattleResultPush{
			BattleID:      outcome.Result.BattleID,
			Win:           outcome.Result.Win,
			ReturnSceneID: outcome.Result.ReturnSceneID,
			ReturnPos: protocol.Vec2i{
				X: outcome.Result.ReturnPos.X,
				Y: outcome.Result.ReturnPos.Y,
			},
			Reason: outcome.Result.Reason,
		})); err != nil {
			return err
		}
		return conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
			Pet: toProtocolPetDetail(updatedPet),
		}))
	}
	return nil
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

func (h *BattleHandler) sendInteractResponse(conn packetSender, seq uint32, accepted bool, reason string) error {
	packet, err := protocol.NewJSONPacket(protocol.CmdInteractResp, seq, errcode.WSCodeSuccess, protocol.InteractResp{
		Accepted: accepted,
		Reason:   reason,
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
		result = append(result, protocol.BattleActorSnapshot{
			ActorID:     actor.ActorID,
			ActorType:   actor.ActorType,
			PetUID:      actor.PetUID,
			PetID:       actor.PetID,
			Name:        actor.Name,
			HP:          actor.HP,
			HPMax:       actor.HPMax,
			SkillIDs:    skills,
			LineupIndex: actor.LineupIndex,
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
			ActorID: actor.ActorID,
			HP:      actor.HP,
			HPMax:   actor.HPMax,
			Dead:    actor.Dead,
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
