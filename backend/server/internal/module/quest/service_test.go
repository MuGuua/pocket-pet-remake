package quest

import (
	"context"
	"testing"
	"time"
)

// 验证多阶段任务只会推进当前最早未完成阶段，避免跳阶段完成后续目标。
func TestApplyEventToObjectivesSequentialStages(t *testing.T) {
	template := Template{
		QuestID: 9001,
		Objectives: []ObjectiveTemplate{
			{ObjectiveID: 1, EventType: "TALK_TO_NPC", Description: "阶段1", TargetValue: 1, TargetSelector: map[string]any{"npc_id": uint64(93001)}},
			{ObjectiveID: 2, EventType: "TALK_TO_NPC", Description: "阶段2", TargetValue: 1, TargetSelector: map[string]any{"npc_id": uint64(93002)}},
		},
	}
	objectives := []PlayerObjective{
		{ObjectiveID: 1, Description: "阶段1", CurrentValue: 0, TargetValue: 1, Completed: false},
		{ObjectiveID: 2, Description: "阶段2", CurrentValue: 0, TargetValue: 1, Completed: false},
	}

	updated, changed := applyEventToObjectives(template, objectives, Event{
		EventType: "TALK_TO_NPC",
		NPCID:     93002,
		Count:     1,
	})
	if changed {
		t.Fatal("stage 2 event should be ignored before stage 1 completes")
	}
	if updated[0].Completed || updated[0].CurrentValue != 0 {
		t.Fatalf("stage 1 should stay incomplete, got current=%d completed=%v", updated[0].CurrentValue, updated[0].Completed)
	}
	if updated[1].Completed || updated[1].CurrentValue != 0 {
		t.Fatalf("stage 2 should not progress yet, got current=%d completed=%v", updated[1].CurrentValue, updated[1].Completed)
	}

	updated, changed = applyEventToObjectives(template, objectives, Event{
		EventType: "TALK_TO_NPC",
		NPCID:     93001,
		Count:     1,
	})
	if !changed {
		t.Fatal("expected stage 1 to progress")
	}
	if !updated[0].Completed {
		t.Fatal("stage 1 should be completed")
	}

	updated, changed = applyEventToObjectives(template, updated, Event{
		EventType: "TALK_TO_NPC",
		NPCID:     93002,
		Count:     1,
	})
	if !changed {
		t.Fatal("expected stage 2 to progress after stage 1 completes")
	}
	if !updated[1].Completed {
		t.Fatal("stage 2 should be completed")
	}
}

// 验证奖励发放失败后的补偿入口会把任务退回可提交状态，避免玩家任务完成但奖励缺失。
func TestRevertCompletedQuestToReady(t *testing.T) {
	repo := newQuestServiceTestRepo()
	service := NewService(repo)

	if err := service.RevertCompletedQuestToReady(context.Background(), 10001, 9001); err != nil {
		t.Fatalf("revert completed quest: %v", err)
	}

	current := repo.playerQuests[10001][9001]
	if current.State != StateReadyToSubmit {
		t.Fatalf("quest state = %s, want %s", current.State, StateReadyToSubmit)
	}
	if !current.Tracked {
		t.Fatal("tracked flag should be preserved when reverting submit state")
	}
}

// 验证运行时摘要会保留后台配置的目标引导，客户端才能据此显示“前往哪里/找谁”。
func TestBuildSummaryKeepsObjectiveGuide(t *testing.T) {
	guide := &ObjectiveGuideInput{SceneID: 3, NPCID: 93001, NPCName: "生产导师·璃梦", Text: "去市场找生产导师"}
	summary := buildSummary(Template{
		QuestID:   9101,
		QuestType: "MAIN",
		Objectives: []ObjectiveTemplate{
			{ObjectiveID: 1, EventType: "TALK_TO_NPC", Description: "与生产导师对话", TargetValue: 1, Guide: guide},
		},
	}, nil, nil, map[uint64]bool{}, AcceptConditionFacts{})

	if len(summary.Objectives) != 1 {
		t.Fatalf("objective count = %d, want 1", len(summary.Objectives))
	}
	if summary.Objectives[0].Guide == nil {
		t.Fatal("objective guide should be kept in runtime summary")
	}
	if summary.Objectives[0].Guide.SceneID != guide.SceneID || summary.Objectives[0].Guide.NPCID != guide.NPCID {
		t.Fatalf("guide = %#v, want %#v", summary.Objectives[0].Guide, guide)
	}
	if summary.Objectives[0].Guide.NPCName != guide.NPCName {
		t.Fatalf("guide npc name = %q, want %q", summary.Objectives[0].Guide.NPCName, guide.NPCName)
	}
}

// 验证任务完成提示会进入运行时摘要，后续协议层才能把后台富文本文案下发给客户端。
func TestBuildSummaryKeepsCompletionPromptText(t *testing.T) {
	summary := buildSummary(Template{
		QuestID:              9102,
		QuestType:            "MAIN",
		CompletionPromptText: "任务完成！获得[color=green]新目标[/color]。",
		Objectives: []ObjectiveTemplate{
			{ObjectiveID: 1, EventType: "TALK_TO_NPC", Description: "与 NPC 对话", TargetValue: 1},
		},
	}, nil, nil, map[uint64]bool{}, AcceptConditionFacts{})

	if summary.CompletionPromptText != "任务完成！获得[color=green]新目标[/color]。" {
		t.Fatalf("completion prompt = %q", summary.CompletionPromptText)
	}
}

// TestAcceptConditionMatchesSupportedTypes 锁定全部后台可配置条件都只读取对应的服务端权威快照。
func TestAcceptConditionMatchesSupportedTypes(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	facts := AcceptConditionFacts{
		Level: 20, SceneID: 7, Stats: map[string]uint64{"atk": 300},
		ItemCounts: map[uint64]uint64{1001: 5}, PetLevels: map[uint64]uint64{2001: 30}, MaxPetLevel: 30,
		StoryFlags: map[string]bool{"main_started": true}, Now: now,
	}
	completed := map[uint64]bool{9001: true}
	conditions := []AcceptCondition{
		{Type: AcceptConditionQuestCompleted, QuestID: 9001},
		{Type: AcceptConditionPlayerLevel, Operator: "gte", Value: 20},
		{Type: AcceptConditionPlayerStat, StatKey: "atk", Operator: "eq", Value: 300},
		{Type: AcceptConditionScene, SceneID: 7},
		{Type: AcceptConditionItemCount, ItemID: 1001, Operator: "gte", Value: 5},
		{Type: AcceptConditionPetLevel, PetID: 2001, Operator: "gte", Value: 30},
		{Type: AcceptConditionStoryFlag, FlagKey: "main_started"},
		{Type: AcceptConditionTimeWindow, StartAt: "2026-07-21T11:00:00Z", EndAt: "2026-07-21T13:00:00Z"},
	}
	for _, condition := range conditions {
		if !acceptConditionMatches(condition, completed, facts) {
			t.Fatalf("condition %#v should match", condition)
		}
	}
}

// TestAcceptConditionsUseANDRelationship 验证任意一条条件失败时任务保持锁定。
func TestAcceptConditionsUseANDRelationship(t *testing.T) {
	template := &Template{AcceptConditions: []AcceptCondition{
		{Type: AcceptConditionPlayerLevel, Operator: "gte", Value: 10},
		{Type: AcceptConditionScene, SceneID: 8},
	}}
	if isUnlocked(template, map[uint64]bool{}, AcceptConditionFacts{Level: 10, SceneID: 7}) {
		t.Fatal("quest should stay locked when one AND condition does not match")
	}
}

// TestHandleProgressEventUsesSceneFacts 验证切图任务事件只读取场景轻量事实，不能退回包含背包物品的完整条件查询。
func TestHandleProgressEventUsesSceneFacts(t *testing.T) {
	repo := newQuestServiceTestRepo()
	service := NewService(repo)

	if _, err := service.HandleProgressEvent(context.Background(), Event{
		PlayerID:  10001,
		EventType: "ENTER_SCENE",
		SceneID:   2,
		Count:     1,
	}); err != nil {
		t.Fatalf("HandleProgressEvent() error = %v", err)
	}
	if repo.sceneFactsCalls != 1 {
		t.Fatalf("scene facts calls = %d, want 1", repo.sceneFactsCalls)
	}
	if repo.fullFactsCalls != 0 {
		t.Fatalf("full facts calls = %d, want 0 because scene transfer must not query bag items", repo.fullFactsCalls)
	}
}

type questServiceTestRepo struct {
	templates        map[uint64]Template
	playerQuests     map[uint64]map[uint64]PlayerQuest
	playerObjectives map[uint64]map[uint64][]PlayerObjective
	fullFactsCalls   int
	sceneFactsCalls  int
}

func newQuestServiceTestRepo() *questServiceTestRepo {
	return &questServiceTestRepo{
		templates: map[uint64]Template{},
		playerQuests: map[uint64]map[uint64]PlayerQuest{
			10001: {
				9001: {PlayerID: 10001, QuestID: 9001, State: StateCompleted, Tracked: true},
			},
		},
		playerObjectives: map[uint64]map[uint64][]PlayerObjective{},
	}
}

func (r *questServiceTestRepo) ListTemplates(context.Context) ([]Template, error) {
	return []Template{}, nil
}

func (r *questServiceTestRepo) FindTemplateByQuestID(_ context.Context, questID uint64) (*Template, error) {
	value, ok := r.templates[questID]
	if !ok {
		return nil, nil
	}
	return &value, nil
}

func (r *questServiceTestRepo) ListTemplatesForAdmin(context.Context, AdminTemplateListQuery) (*AdminTemplateList, error) {
	return &AdminTemplateList{}, nil
}

func (r *questServiceTestRepo) FindAdminTemplateDetailByQuestID(context.Context, uint64) (*AdminTemplateDetail, error) {
	return nil, nil
}

func (r *questServiceTestRepo) CreateTemplateForAdmin(context.Context, AdminCreateTemplateInput) (*AdminTemplateDetail, error) {
	return nil, nil
}

func (r *questServiceTestRepo) UpdateTemplateForAdmin(context.Context, uint64, AdminUpdateTemplateInput) (*AdminTemplateDetail, error) {
	return nil, nil
}

func (r *questServiceTestRepo) DeleteTemplateForAdmin(context.Context, uint64) error {
	return nil
}

func (r *questServiceTestRepo) ListPlayerQuestsByPlayerID(_ context.Context, playerID uint64) ([]PlayerQuest, error) {
	values := make([]PlayerQuest, 0, len(r.playerQuests[playerID]))
	for _, value := range r.playerQuests[playerID] {
		values = append(values, value)
	}
	return values, nil
}

func (r *questServiceTestRepo) ListPlayerObjectivesByPlayerID(_ context.Context, playerID uint64) ([]PlayerObjective, error) {
	values := []PlayerObjective{}
	for _, objectiveValues := range r.playerObjectives[playerID] {
		values = append(values, objectiveValues...)
	}
	return values, nil
}

func (r *questServiceTestRepo) LoadAcceptConditionFacts(context.Context, uint64) (AcceptConditionFacts, error) {
	r.fullFactsCalls++
	return AcceptConditionFacts{Level: 100, Stats: map[string]uint64{}, ItemCounts: map[uint64]uint64{}, PetLevels: map[uint64]uint64{}, StoryFlags: map[string]bool{}, Now: time.Now()}, nil
}

// LoadSceneEventConditionFacts 模拟生产仓储的轻量场景事实读取，ItemCounts 始终为空以代表未访问背包。
func (r *questServiceTestRepo) LoadSceneEventConditionFacts(context.Context, uint64) (AcceptConditionFacts, error) {
	r.sceneFactsCalls++
	return AcceptConditionFacts{Level: 100, Stats: map[string]uint64{}, ItemCounts: map[uint64]uint64{}, PetLevels: map[uint64]uint64{}, StoryFlags: map[string]bool{}, Now: time.Now()}, nil
}

func (r *questServiceTestRepo) ListPlayerQuestsForAdmin(context.Context, AdminPlayerQuestListQuery) (*AdminPlayerQuestList, error) {
	return &AdminPlayerQuestList{}, nil
}

func (r *questServiceTestRepo) FindAdminPlayerQuestDetailByRecordID(context.Context, uint64) (*AdminPlayerQuestDetail, error) {
	return nil, nil
}

func (r *questServiceTestRepo) CreatePlayerQuestForAdmin(context.Context, AdminCreatePlayerQuestInput) (*AdminPlayerQuestDetail, error) {
	return nil, nil
}

func (r *questServiceTestRepo) UpdatePlayerQuestForAdmin(context.Context, uint64, AdminUpdatePlayerQuestInput) (*AdminPlayerQuestDetail, error) {
	return nil, nil
}

func (r *questServiceTestRepo) DeletePlayerQuestForAdmin(context.Context, uint64) error {
	return nil
}

func (r *questServiceTestRepo) UpsertPlayerQuest(_ context.Context, value PlayerQuest) error {
	if r.playerQuests[value.PlayerID] == nil {
		r.playerQuests[value.PlayerID] = map[uint64]PlayerQuest{}
	}
	r.playerQuests[value.PlayerID][value.QuestID] = value
	return nil
}

func (r *questServiceTestRepo) ReplacePlayerObjectives(_ context.Context, playerID uint64, questID uint64, objectives []PlayerObjective) error {
	if r.playerObjectives[playerID] == nil {
		r.playerObjectives[playerID] = map[uint64][]PlayerObjective{}
	}
	r.playerObjectives[playerID][questID] = objectives
	return nil
}

func (r *questServiceTestRepo) SetTrackedQuest(context.Context, uint64, uint64) error {
	return nil
}

func (r *questServiceTestRepo) RevertCompletedQuestToReady(_ context.Context, playerID uint64, questID uint64) error {
	current := r.playerQuests[playerID][questID]
	if current.State == StateCompleted {
		current.State = StateReadyToSubmit
		r.playerQuests[playerID][questID] = current
	}
	return nil
}
