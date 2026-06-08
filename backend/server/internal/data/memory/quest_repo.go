package memory

import (
	"context"
	"sync"

	"pocket-pet-remake/server/internal/module/quest"
)

type QuestRepository struct {
	mu               sync.RWMutex
	templates        map[uint64]quest.Template
	playerQuests     map[uint64]map[uint64]quest.PlayerQuest
	playerObjectives map[uint64]map[uint64][]quest.PlayerObjective
}

func NewQuestRepository() *QuestRepository {
	return &QuestRepository{
		templates: map[uint64]quest.Template{
			1001: {
				QuestID:     1001,
				QuestType:   "MAIN",
				Title:       "初入闪光镇",
				Description: "前往闪光镇东路，熟悉周围环境。",
				AcceptMode:  "AUTO",
				SubmitMode:  "AUTO",
				AutoTrack:   true,
				Objectives: []quest.ObjectiveTemplate{
					{ObjectiveID: 1, EventType: "ENTER_SCENE", Description: "进入闪光镇东路", TargetValue: 1, TargetSelector: map[string]any{"scene_id": uint32(2)}},
				},
			},
			1002: {
				QuestID:     1002,
				QuestType:   "MAIN",
				Title:       "向市场管理员报到",
				Description: "找到市场理萌并和她交谈。",
				AcceptMode:  "NPC",
				SubmitMode:  "NPC",
				StartNPCID:  93001,
				SubmitNPCID: 93001,
				AutoTrack:   true,
				PreQuestIDs: []uint64{1001},
				Objectives: []quest.ObjectiveTemplate{
					{ObjectiveID: 1, EventType: "TALK_TO_NPC", Description: "与市场理萌交谈", TargetValue: 1, TargetSelector: map[string]any{"npc_id": uint64(93001)}},
				},
			},
			1003: {
				QuestID:     1003,
				QuestType:   "MAIN",
				Title:       "完成第一次对战",
				Description: "挑战附近的教学 NPC 并赢得胜利。",
				AcceptMode:  "AUTO",
				SubmitMode:  "AUTO",
				AutoTrack:   true,
				PreQuestIDs: []uint64{1002},
				Objectives: []quest.ObjectiveTemplate{
					{ObjectiveID: 1, EventType: "WIN_BATTLE", Description: "完成 1 场战斗", TargetValue: 1, TargetSelector: map[string]any{"battle_type": "PVE"}},
				},
			},
		},
		playerQuests:     map[uint64]map[uint64]quest.PlayerQuest{},
		playerObjectives: map[uint64]map[uint64][]quest.PlayerObjective{},
	}
}

func (r *QuestRepository) ListTemplates(_ context.Context) ([]quest.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]quest.Template, 0, len(r.templates))
	for _, template := range r.templates {
		result = append(result, template)
	}
	return result, nil
}

func (r *QuestRepository) FindTemplateByQuestID(_ context.Context, questID uint64) (*quest.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.templates[questID]
	if !ok {
		return nil, nil
	}
	copyValue := value
	return &copyValue, nil
}

func (r *QuestRepository) ListPlayerQuestsByPlayerID(_ context.Context, playerID uint64) ([]quest.PlayerQuest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playerMap := r.playerQuests[playerID]
	if len(playerMap) == 0 {
		return []quest.PlayerQuest{}, nil
	}
	result := make([]quest.PlayerQuest, 0, len(playerMap))
	for _, value := range playerMap {
		result = append(result, value)
	}
	return result, nil
}

func (r *QuestRepository) ListPlayerObjectivesByPlayerID(_ context.Context, playerID uint64) ([]quest.PlayerObjective, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	questMap := r.playerObjectives[playerID]
	if len(questMap) == 0 {
		return []quest.PlayerObjective{}, nil
	}
	result := []quest.PlayerObjective{}
	for _, values := range questMap {
		result = append(result, values...)
	}
	return result, nil
}

func (r *QuestRepository) UpsertPlayerQuest(_ context.Context, value quest.PlayerQuest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.playerQuests[value.PlayerID] == nil {
		r.playerQuests[value.PlayerID] = map[uint64]quest.PlayerQuest{}
	}
	r.playerQuests[value.PlayerID][value.QuestID] = value
	return nil
}

func (r *QuestRepository) ReplacePlayerObjectives(_ context.Context, playerID uint64, questID uint64, objectives []quest.PlayerObjective) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.playerObjectives[playerID] == nil {
		r.playerObjectives[playerID] = map[uint64][]quest.PlayerObjective{}
	}
	copied := make([]quest.PlayerObjective, 0, len(objectives))
	copied = append(copied, objectives...)
	r.playerObjectives[playerID][questID] = copied
	return nil
}

func (r *QuestRepository) SetTrackedQuest(_ context.Context, playerID uint64, questID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.playerQuests[playerID] == nil {
		r.playerQuests[playerID] = map[uint64]quest.PlayerQuest{}
	}
	for currentQuestID, current := range r.playerQuests[playerID] {
		current.Tracked = currentQuestID == questID
		r.playerQuests[playerID][currentQuestID] = current
	}
	target := r.playerQuests[playerID][questID]
	target.PlayerID = playerID
	target.QuestID = questID
	target.Tracked = true
	if target.State == "" {
		target.State = quest.StateAvailable
	}
	r.playerQuests[playerID][questID] = target
	return nil
}
