package wstransport

import (
	"context"
	"errors"
	"time"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/reward"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/unlock"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

type QuestHandler struct {
	questService   *quest.Service
	sessionService *session.Service
	bagService     *bag.Service
	petService     *pet.Service
	walletService  *wallet.Service
	unlockService  *unlock.Service
	playerService  *player.Service
	rewardService  *reward.Service
}

func NewQuestHandler(questService *quest.Service, sessionService *session.Service, bagService *bag.Service, petService *pet.Service, walletService *wallet.Service, unlockService *unlock.Service, playerService *player.Service) *QuestHandler {
	return &QuestHandler{
		questService:   questService,
		sessionService: sessionService,
		bagService:     bagService,
		petService:     petService,
		walletService:  walletService,
		unlockService:  unlockService,
		playerService:  playerService,
		rewardService:  reward.NewService(bagService, petService, playerService, unlockService, walletService),
	}
}

func (h *QuestHandler) HandleQuestList(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestListReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest list body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	summaries, trackedQuestID, err := h.questService.List(context.Background(), playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest list failed", err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestListResp, packet.Seq, errcode.WSCodeSuccess, protocol.QuestListResp{
		Quests:         toProtocolQuestSummaries(summaries),
		TrackedQuestID: trackedQuestID,
		ServerTimeMS:   time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	return conn.SendPacket(responsePacket)
}

func (h *QuestHandler) HandleQuestAccept(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestAcceptReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest accept body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	ctx := context.Background()
	before, err := listQuestSummaries(ctx, h.questService, playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest state failed", err)
	}
	summary, err := h.questService.Accept(ctx, playerID, request.QuestID, request.NPCID)
	if err != nil {
		return h.sendQuestDomainError(conn, packet.Seq, err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestAcceptResp, packet.Seq, errcode.WSCodeSuccess, protocol.QuestAcceptResp{
		Accepted:           true,
		Reason:             "quest accepted",
		Quest:              toProtocolQuestSummary(summary),
		ClientAnimationKey: summary.AcceptAnimationKey,
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, before)
}

func (h *QuestHandler) HandleQuestSubmit(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestSubmitReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest submit body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	ctx := context.Background()
	before, err := listQuestSummaries(ctx, h.questService, playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest state failed", err)
	}
	result, err := h.questService.Submit(ctx, playerID, request.QuestID, request.NPCID)
	if err != nil {
		return h.sendQuestDomainError(conn, packet.Seq, err)
	}
	grantResult, err := h.grantQuestRewards(ctx, playerID, request.QuestID, result.Rewards)
	if err != nil {
		if revertErr := h.questService.RevertCompletedQuestToReady(ctx, playerID, request.QuestID); revertErr != nil {
			err = errors.Join(err, revertErr)
		}
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "grant quest rewards failed", err)
	}

	submitResp := protocol.QuestSubmitResp{
		Accepted:           true,
		Reason:             "quest submitted",
		Quest:              toProtocolQuestSummary(result.Summary),
		Rewards:            toProtocolPopupRewards(grantResult.Granted),
		ClientAnimationKey: result.Summary.SubmitAnimationKey,
	}
	if grantResult != nil {
		submitResp.LevelUpCount = grantResult.LevelUpCount
		submitResp.AttrPointsGained = grantResult.AttrPointsGained
		submitResp.LevelUpBonus = toProtocolLevelUpBonus(grantResult.LevelUpCount, grantResult.CombatBonusGain)
		if grantResult.PlayerProfile != nil {
			submitResp.PlayerLevel = grantResult.PlayerProfile.Level
		}
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestSubmitResp, packet.Seq, errcode.WSCodeSuccess, submitResp)
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	if grantResult != nil && grantResult.Wallet != nil {
		if err := pushWalletUpdatePacket(conn, *grantResult.Wallet, "quest_reward", request.QuestID); err != nil {
			return err
		}
	}
	if grantResult != nil && grantResult.BagUpdated && h.bagService != nil {
		bagSnapshot, err := h.bagService.ListRuntimeContainer(ctx, playerID, bag.ContainerTypeBag)
		if err != nil {
			return sendError(conn, packet.Seq, errcode.WSCodeBagListFailed, "load bag snapshot after quest reward failed", err)
		}
		if bagSnapshot != nil {
			if err := conn.SendPacket(mustJSONPacket(protocol.CmdBagUpdatePush, 0, buildContainerUpdatePush(*bagSnapshot))); err != nil {
				return err
			}
		}
	}
	if grantResult != nil {
		for _, grantedPet := range grantResult.GrantedPets {
			if err := conn.SendPacket(mustJSONPacket(protocol.CmdPetUpdatePush, 0, protocol.PetUpdatePush{
				Pet: toProtocolPetDetail(grantedPet),
			})); err != nil {
				return err
			}
		}
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, before)
}

func (h *QuestHandler) HandleQuestTrack(conn packetSender, packet *protocol.Packet) error {
	var request protocol.QuestTrackReq
	if err := protocol.UnmarshalBody(packet.Body, &request); err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeInvalidPacket, "invalid quest track body")
	}

	playerID, err := h.requirePlayerID(conn.ID())
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeSessionInvalid, "session invalid")
	}
	ctx := context.Background()
	before, err := listQuestSummaries(ctx, h.questService, playerID)
	if err != nil {
		return sendError(conn, packet.Seq, errcode.WSCodeWorldEnterFailed, "load quest state failed", err)
	}
	if err := h.questService.Track(ctx, playerID, request.QuestID); err != nil {
		return h.sendQuestDomainError(conn, packet.Seq, err)
	}

	responsePacket, err := protocol.NewJSONPacket(protocol.CmdQuestTrackResp, packet.Seq, errcode.WSCodeSuccess, protocol.QuestTrackResp{
		Accepted: true,
		Reason:   "track updated",
		QuestID:  request.QuestID,
	})
	if err != nil {
		return err
	}
	if err := conn.SendPacket(responsePacket); err != nil {
		return err
	}
	return pushQuestDiff(ctx, conn, h.questService, playerID, before)
}

func (h *QuestHandler) requirePlayerID(connID string) (uint64, error) {
	sess, err := h.sessionService.GetByConnID(connID)
	if err != nil {
		return 0, err
	}
	return sess.PlayerID, nil
}

func (h *QuestHandler) sendQuestDomainError(conn packetSender, seq uint32, err error) error {
	reason := "quest request failed"
	switch {
	case errors.Is(err, quest.ErrQuestTemplateNotFound):
		reason = "quest not found"
	case errors.Is(err, quest.ErrQuestLocked):
		reason = "quest locked"
	case errors.Is(err, quest.ErrQuestNotAvailable):
		reason = "quest not available"
	case errors.Is(err, quest.ErrQuestNotReady):
		reason = "quest not ready to submit"
	case errors.Is(err, quest.ErrQuestAcceptNPCMismatch):
		reason = "quest accept npc mismatch"
	case errors.Is(err, quest.ErrQuestSubmitNPCMismatch):
		reason = "quest submit npc mismatch"
	}
	packet, packetErr := protocol.NewJSONPacket(protocol.CmdErrorPush, seq, errcode.WSCodeSuccess, protocol.ErrorPush{
		Code: errcode.WSCodeSuccess,
		Msg:  reason,
	})
	if packetErr != nil {
		return packetErr
	}
	return conn.SendPacket(packet)
}

func (h *QuestHandler) grantQuestRewards(ctx context.Context, playerID uint64, questID uint64, rewards []quest.Reward) (*reward.GrantResult, error) {
	if h.rewardService == nil {
		return &reward.GrantResult{}, nil
	}
	return h.rewardService.GrantRuntimeRewards(ctx, reward.GrantInput{
		PlayerID:     playerID,
		ReasonType:   "quest_reward",
		ReasonRefID:  questID,
		OperatorType: "system",
		OperatorID:   playerID,
		Rewards:      toRuntimeRewardEntries(rewards),
	})
}

// toRuntimeRewardEntries 把任务奖励模板转成统一发奖服务可以消费的结构。
func toRuntimeRewardEntries(values []quest.Reward) []reward.Entry {
	result := make([]reward.Entry, 0, len(values))
	for _, value := range values {
		result = append(result, reward.Entry{
			Type:     value.Type,
			Value:    value.Value,
			ItemID:   value.ItemID,
			ItemName: "",
			Count:    value.Count,
			PetID:    value.PetID,
		})
	}
	return result
}

// toQuestRewards 把统一发奖结果转回 quest 协议层沿用的奖励结构。
func toQuestRewards(values []reward.Entry) []quest.Reward {
	result := make([]quest.Reward, 0, len(values))
	for _, value := range values {
		result = append(result, quest.Reward{
			Type:   value.Type,
			Value:  value.Value,
			ItemID: value.ItemID,
			Count:  value.Count,
			PetID:  value.PetID,
		})
	}
	return result
}
