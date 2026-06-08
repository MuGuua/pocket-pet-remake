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
			ObjectiveID: objective.ObjectiveID,
			Description: objective.Description,
			Current:     objective.Current,
			Target:      objective.Target,
			Completed:   objective.Completed,
		})
	}
	return protocol.QuestSummary{
		QuestID:     value.QuestID,
		QuestType:   value.QuestType,
		State:       value.State,
		Tracked:     value.Tracked,
		StartNPCID:  value.StartNPCID,
		SubmitNPCID: value.SubmitNPCID,
		Title:       value.Title,
		Description: value.Description,
		Objectives:  objectives,
	}
}
