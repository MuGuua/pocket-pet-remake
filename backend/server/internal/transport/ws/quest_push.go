package wstransport

import (
	"context"
	"reflect"

	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/platform/errcode"
	"pocket-pet-remake/server/internal/protocol"
)

func snapshotQuestSummaries(values []quest.Summary) map[uint64]quest.Summary {
	result := make(map[uint64]quest.Summary, len(values))
	for _, value := range values {
		result[value.QuestID] = value
	}
	return result
}

func listQuestSummaries(ctx context.Context, service *quest.Service, playerID uint64) ([]quest.Summary, error) {
	if service == nil {
		return []quest.Summary{}, nil
	}
	summaries, _, err := service.List(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

func pushQuestDiff(ctx context.Context, conn packetSender, service *quest.Service, playerID uint64, before []quest.Summary) error {
	after, err := listQuestSummaries(ctx, service, playerID)
	if err != nil {
		return err
	}
	return sendQuestDiff(conn, before, after)
}

// pushQuestUpdates 推送已由领域服务确定发生变化的任务，避免为了生成差异再次读取全部任务和开启条件事实。
func pushQuestUpdates(conn packetSender, summaries []quest.Summary) error {
	for _, summary := range summaries {
		packet, err := protocol.NewJSONPacket(protocol.CmdQuestUpdatePush, 0, errcode.WSCodeSuccess, protocol.QuestUpdatePush{
			Quest: toProtocolQuestSummary(summary),
		})
		if err != nil {
			return err
		}
		if err := conn.SendPacket(packet); err != nil {
			return err
		}
	}
	return nil
}

func sendQuestDiff(conn packetSender, before []quest.Summary, after []quest.Summary) error {
	beforeMap := snapshotQuestSummaries(before)
	afterMap := snapshotQuestSummaries(after)

	for questID, summary := range afterMap {
		previous, ok := beforeMap[questID]
		if ok && reflect.DeepEqual(previous, summary) {
			continue
		}
		packet, err := protocol.NewJSONPacket(protocol.CmdQuestUpdatePush, 0, errcode.WSCodeSuccess, protocol.QuestUpdatePush{
			Quest: toProtocolQuestSummary(summary),
		})
		if err != nil {
			return err
		}
		if err := conn.SendPacket(packet); err != nil {
			return err
		}
	}

	for questID := range beforeMap {
		if _, ok := afterMap[questID]; ok {
			continue
		}
		packet, err := protocol.NewJSONPacket(protocol.CmdQuestRemovePush, 0, errcode.WSCodeSuccess, protocol.QuestRemovePush{
			QuestID: questID,
		})
		if err != nil {
			return err
		}
		if err := conn.SendPacket(packet); err != nil {
			return err
		}
	}
	return nil
}

func toProtocolQuestSummaries(values []quest.Summary) []protocol.QuestSummary {
	result := make([]protocol.QuestSummary, 0, len(values))
	for _, value := range values {
		result = append(result, toProtocolQuestSummary(value))
	}
	return result
}

func toProtocolQuestSummary(value quest.Summary) protocol.QuestSummary {
	objectives := make([]protocol.QuestObjectiveState, 0, len(value.Objectives))
	for _, objective := range value.Objectives {
		objectives = append(objectives, protocol.QuestObjectiveState{
			ObjectiveID:    objective.ObjectiveID,
			EventType:      objective.EventType,
			Description:    objective.Description,
			Current:        objective.Current,
			Target:         objective.Target,
			Completed:      objective.Completed,
			TargetSelector: copyQuestTargetSelector(objective.TargetSelector),
			Guide:          toProtocolQuestObjectiveGuide(objective.Guide),
		})
	}
	return protocol.QuestSummary{
		QuestID:              value.QuestID,
		QuestType:            value.QuestType,
		ClientIconID:         value.ClientIconID,
		State:                value.State,
		Tracked:              value.Tracked,
		StartNPCID:           value.StartNPCID,
		SubmitNPCID:          value.SubmitNPCID,
		Title:                value.Title,
		Description:          value.Description,
		CompletionPromptText: value.CompletionPromptText,
		Objectives:           objectives,
		Rewards:              toProtocolQuestRewards(value.Rewards),
	}
}

func copyQuestTargetSelector(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func toProtocolQuestObjectiveGuide(value *quest.ObjectiveGuideInput) *protocol.QuestObjectiveGuide {
	if value == nil {
		return nil
	}
	if value.SceneID == 0 && value.NPCID == 0 && value.NPCName == "" && value.Text == "" && value.MenuEntryID == 0 && value.DialogueEntryID == 0 {
		return nil
	}
	return &protocol.QuestObjectiveGuide{
		SceneID:         value.SceneID,
		NPCID:           value.NPCID,
		NPCName:         value.NPCName,
		Text:            value.Text,
		MenuEntryID:     value.MenuEntryID,
		DialogueEntryID: value.DialogueEntryID,
	}
}

func toProtocolQuestRewards(values []quest.Reward) []protocol.QuestReward {
	result := make([]protocol.QuestReward, 0, len(values))
	for _, value := range values {
		result = append(result, protocol.QuestReward{
			Type:   value.Type,
			Value:  value.Value,
			ItemID: value.ItemID,
			Count:  value.Count,
			PetID:  value.PetID,
		})
	}
	return result
}
