package quest

import (
	"context"
	"time"
)

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
	LoadAcceptConditionFacts(ctx context.Context, playerID uint64) (AcceptConditionFacts, error)
	// LoadSceneEventConditionFacts 只读取场景任务事实，不得访问背包物品容器。
	LoadSceneEventConditionFacts(ctx context.Context, playerID uint64) (AcceptConditionFacts, error)
	ListPlayerQuestsForAdmin(ctx context.Context, query AdminPlayerQuestListQuery) (*AdminPlayerQuestList, error)
	FindAdminPlayerQuestDetailByRecordID(ctx context.Context, recordID uint64) (*AdminPlayerQuestDetail, error)
	CreatePlayerQuestForAdmin(ctx context.Context, input AdminCreatePlayerQuestInput) (*AdminPlayerQuestDetail, error)
	UpdatePlayerQuestForAdmin(ctx context.Context, recordID uint64, input AdminUpdatePlayerQuestInput) (*AdminPlayerQuestDetail, error)
	DeletePlayerQuestForAdmin(ctx context.Context, recordID uint64) error
	UpsertPlayerQuest(ctx context.Context, value PlayerQuest) error
	ReplacePlayerObjectives(ctx context.Context, playerID uint64, questID uint64, objectives []PlayerObjective) error
	SetTrackedQuest(ctx context.Context, playerID uint64, questID uint64) error
	RevertCompletedQuestToReady(ctx context.Context, playerID uint64, questID uint64) error
	// ResetPlayerQuest 把任务恢复为 AVAILABLE：清空目标进度、取消追踪与刷新时间，保留 completed_at。
	ResetPlayerQuest(ctx context.Context, playerID uint64, questID uint64) error
	// SetQuestNextRefreshAt 写入可重复任务的下次刷新时间；nil 表示清除。
	SetQuestNextRefreshAt(ctx context.Context, playerID uint64, questID uint64, nextRefreshAt *time.Time) error
}
