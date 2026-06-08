package quest

import "context"

type Repository interface {
	ListTemplates(ctx context.Context) ([]Template, error)
	FindTemplateByQuestID(ctx context.Context, questID uint64) (*Template, error)
	ListPlayerQuestsByPlayerID(ctx context.Context, playerID uint64) ([]PlayerQuest, error)
	ListPlayerObjectivesByPlayerID(ctx context.Context, playerID uint64) ([]PlayerObjective, error)
	UpsertPlayerQuest(ctx context.Context, value PlayerQuest) error
	ReplacePlayerObjectives(ctx context.Context, playerID uint64, questID uint64, objectives []PlayerObjective) error
	SetTrackedQuest(ctx context.Context, playerID uint64, questID uint64) error
}
