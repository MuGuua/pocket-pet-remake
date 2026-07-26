package wstransport

import (
	"context"
	"errors"
	"strings"

	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

// PlayerHandler 处理玩家成长相关的 WebSocket 请求。
type PlayerHandler struct {
	sessionService *session.Service
	playerService  *player.Service
}

// NewPlayerHandler 构造玩家成长处理器。
func NewPlayerHandler(sessionService *session.Service, playerService *player.Service) *PlayerHandler {
	return &PlayerHandler{
		sessionService: sessionService,
		playerService:  playerService,
	}
}

// HandleProfile 返回人物状态面板所需的权威属性，不进入完整世界、背包、宠物或任务查询链路。
func (h *PlayerHandler) HandleProfile(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PlayerProfileReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid player profile body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.playerService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "player service unavailable")
	}

	profile, err := h.playerService.GetBattleReadyProfile(context.Background(), sess.PlayerID)
	if err != nil {
		if errors.Is(err, player.ErrPlayerNotFound) {
			return sendError(conn, packet.Seq, errcode.WSCodePlayerNotFound, "player not found", err)
		}
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "load player profile failed", err)
	}
	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPlayerProfileResp, packet.Seq, errcode.WSCodeSuccess, protocol.PlayerProfileResp{
		Player: toProtocolPlayerSnapshot(profile),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

// HandleAllocateAttr 处理玩家主动分配自由属性点。
func (h *PlayerHandler) HandleAllocateAttr(conn packetSender, packet *protocol.Packet) error {
	var request protocol.PlayerAllocateAttrReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid allocate attr body")
	}
	sess, err := h.sessionService.GetByConnID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	if h.playerService == nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "player service unavailable")
	}

	delta := progression.AttrAllocationDelta{
		Strength: request.Strength,
		Vitality: request.Vitality,
		Agility:  request.Agility,
		Mind:     request.Mind,
	}
	profile, err := h.playerService.AllocateAttrPoints(context.Background(), sess.PlayerID, delta)
	if err != nil {
		switch {
		case errors.Is(err, progression.ErrInsufficientAttrPoints):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "insufficient attr points")
		case errors.Is(err, progression.ErrInvalidAllocateInput):
			return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid allocate attr input")
		default:
			return sendError(conn, packet.Seq, errcode.WSCodeInteractFailed, "allocate attr failed")
		}
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdPlayerAllocateAttrResp, packet.Seq, errcode.WSCodeSuccess, protocol.PlayerAllocateAttrResp{
		Player: toProtocolPlayerSnapshot(profile),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

func toProtocolPlayerSnapshot(profile *player.Profile) protocol.PlayerSnapshot {
	if profile == nil {
		return protocol.PlayerSnapshot{}
	}
	return protocol.PlayerSnapshot{
		PlayerID:           profile.PlayerID,
		Name:               profile.Name,
		Level:              profile.Level,
		Exp:                profile.Exp,
		ExpToNext:          profile.ExpToNext,
		FreeAttrPoints:     profile.FreeAttrPoints,
		Strength:           profile.Strength,
		Vitality:           profile.Vitality,
		Agility:            profile.Agility,
		Mind:               profile.Mind,
		Gold:               profile.Gold,
		HP:                 profile.HP,
		HPMax:              profile.HPMax,
		Vigor:              profile.Vigor,
		VigorMax:           profile.VigorMax,
		Spirit:             profile.Spirit,
		SpiritMax:          profile.SpiritMax,
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
		SkinID:             strings.TrimSpace(profile.SkinID),
	}
}
