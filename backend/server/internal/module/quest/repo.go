package quest

import "context"

type Repository interface {
	ListTemplates(ctx context.Context) ([]Template, error)
	FindTemplateByQuestID(ctx context.Context, questID uint64) (*Template, error)
	ListTemplatesForAdmin(ctx context.Context, query AdminTemplateListQuery) (*AdminTemplateList, error)
	FindAdminTemplateDetailByQuestID(ctx context.Context, questID uint64) (*AdminTemplateDetail, error)
	CreateTemplateForAdmin(ctx context.Context, input AdminCreateTemplateInput) (*AdminTemplateDetail, error)
	UpdateTemplateForAdmin(ctx context.Context, questID uint64, input AdminUpdateTemplateInput) (*AdminTemplateDetail, error)
	DeleteTemplateForAdmin(ctx context.Context, questID uint64) error
	ListPlayerQuestsByPlayerID(ctx context.Context, playerID uint64) ([]PlayerQuest, error)
	ListPlayerObjectivesByPlayerID(ctx context.Context, playerID uint64) ([]PlayerObjective, error)
	ListPlayerQuestsForAdmin(ctx context.Context, query AdminPlayerQuestListQuery) (*AdminPlayerQuestList, error)
	FindAdminPlayerQuestDetailByRecordID(ctx context.Context, recordID uint64) (*AdminPlayerQuestDetail, error)
	CreatePlayerQuestForAdmin(ctx context.Context, input AdminCreatePlayerQuestInput) (*AdminPlayerQuestDetail, error)
	UpdatePlayerQuestForAdmin(ctx context.Context, recordID uint64, input AdminUpdatePlayerQuestInput) (*AdminPlayerQuestDetail, error)
	DeletePlayerQuestForAdmin(ctx context.Context, recordID uint64) error
	UpsertPlayerQuest(ctx context.Context, value PlayerQuest) error
	ReplacePlayerObjectives(ctx context.Context, playerID uint64, questID uint64, objectives []PlayerObjective) error
	SetTrackedQuest(ctx context.Context, playerID uint64, questID uint64) error
}
