package quest

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 测试统一使用固定时区，避免运行环境时区影响「每天 5 点」的边界判断。
var refreshTestLoc = time.FixedZone("CST", 8*3600)

// TestNextRefreshTimeDaily 锁定每日刷新：5 点前完成当天 5 点刷新，5 点后完成次日 5 点刷新。
func TestNextRefreshTimeDaily(t *testing.T) {
	template := Template{IsRepeatable: true, RefreshCycle: RefreshCycleDaily, RefreshHour: 5}
	cases := []struct {
		name      string
		completed time.Time
		want      time.Time
	}{
		{"凌晨完成当天刷新", time.Date(2026, 10, 20, 3, 0, 0, 0, refreshTestLoc), time.Date(2026, 10, 20, 5, 0, 0, 0, refreshTestLoc)},
		{"白天完成次日刷新", time.Date(2026, 10, 20, 14, 0, 0, 0, refreshTestLoc), time.Date(2026, 10, 21, 5, 0, 0, 0, refreshTestLoc)},
		{"整点完成次日刷新", time.Date(2026, 10, 20, 5, 0, 0, 0, refreshTestLoc), time.Date(2026, 10, 21, 5, 0, 0, 0, refreshTestLoc)},
	}
	for _, tc := range cases {
		got := NextRefreshTime(template, tc.completed)
		if got == nil || !got.Equal(tc.want) {
			t.Fatalf("%s: next = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestNextRefreshTimeWeekly 锁定每周一刷新。2026-10-20 是周二。
func TestNextRefreshTimeWeekly(t *testing.T) {
	template := Template{IsRepeatable: true, RefreshCycle: RefreshCycleWeekly, RefreshHour: 5}
	completed := time.Date(2026, 10, 20, 14, 0, 0, 0, refreshTestLoc)
	want := time.Date(2026, 10, 26, 5, 0, 0, 0, refreshTestLoc)
	if got := NextRefreshTime(template, completed); got == nil || !got.Equal(want) {
		t.Fatalf("next = %v, want %v", got, want)
	}
	// 周一 3 点完成，当天 5 点即刷新。
	monday := time.Date(2026, 10, 26, 3, 0, 0, 0, refreshTestLoc)
	wantSameDay := time.Date(2026, 10, 26, 5, 0, 0, 0, refreshTestLoc)
	if got := NextRefreshTime(template, monday); got == nil || !got.Equal(wantSameDay) {
		t.Fatalf("monday next = %v, want %v", got, wantSameDay)
	}
}

// TestNextRefreshTimeNonRepeatable 不可重复任务不刷新；可重复但周期 none 立即可重做。
func TestNextRefreshTimeNonRepeatable(t *testing.T) {
	now := time.Date(2026, 10, 20, 14, 0, 0, 0, refreshTestLoc)
	if got := NextRefreshTime(Template{RefreshCycle: RefreshCycleDaily}, now); got != nil {
		t.Fatalf("non repeatable next = %v, want nil", got)
	}
	if got := NextRefreshTime(Template{IsRepeatable: true}, now); got == nil || !got.Equal(now) {
		t.Fatalf("cycle none next = %v, want immediate %v", got, now)
	}
}

// newDailyQuestFixture 准备一个可重复的每日 NPC 任务，玩家已达成全部目标、待交付。
func newDailyQuestFixture() (*questServiceTestRepo, *Service, *time.Time) {
	const playerID uint64 = 10001
	const questID uint64 = 7001
	repo := newQuestServiceTestRepo()
	repo.templates[questID] = Template{
		QuestID: questID, QuestType: "DAILY", Title: "每日击败怪物", AcceptMode: "NPC", SubmitMode: "NPC",
		IsRepeatable: true, RefreshCycle: RefreshCycleDaily, RefreshHour: 5, CanAbandon: true,
		Objectives: []ObjectiveTemplate{{ObjectiveID: 1, EventType: "WIN_BATTLE", Description: "赢得 1 场战斗", TargetValue: 1}},
	}
	repo.playerQuests[playerID][questID] = PlayerQuest{PlayerID: playerID, QuestID: questID, State: StateReadyToSubmit}
	repo.playerObjectives[playerID] = map[uint64][]PlayerObjective{
		questID: {{PlayerID: playerID, QuestID: questID, ObjectiveID: 1, CurrentValue: 1, TargetValue: 1, Completed: true}},
	}
	clock := time.Date(2026, 10, 20, 14, 0, 0, 0, refreshTestLoc)
	service := NewService(repo)
	service.SetClock(func() time.Time { return clock })
	return repo, service, &clock
}

// TestDailyQuestResetsAfterRefreshTime 端到端验证：交付后当天仍为已完成，过次日 5 点自动重置为可接取。
func TestDailyQuestResetsAfterRefreshTime(t *testing.T) {
	repo, service, clock := newDailyQuestFixture()
	ctx := context.Background()

	result, err := service.Submit(ctx, 10001, 7001, 0)
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if result.Summary.State != StateCompleted || result.Summary.NextRefreshAt == nil {
		t.Fatalf("summary = %#v, want completed with next refresh time", result.Summary)
	}

	*clock = time.Date(2026, 10, 21, 4, 59, 0, 0, refreshTestLoc)
	summaries, _, err := service.List(ctx, 10001)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if state := findSummaryState(summaries, 7001); state != StateCompleted {
		t.Fatalf("before refresh state = %s, want COMPLETED", state)
	}

	*clock = time.Date(2026, 10, 21, 5, 0, 0, 0, refreshTestLoc)
	summaries, _, err = service.List(ctx, 10001)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if state := findSummaryState(summaries, 7001); state != StateAvailable {
		t.Fatalf("after refresh state = %s, want AVAILABLE", state)
	}
	if objectives := repo.playerObjectives[10001][7001]; len(objectives) != 0 {
		t.Fatalf("objectives after refresh = %#v, want cleared", objectives)
	}
	// completed_at 保留，后置任务不会被重新锁住。
	if repo.playerQuests[10001][7001].CompletedAt == nil {
		t.Fatal("completed_at should be kept after refresh")
	}

	// 重置后可以再次接取，目标从 0 开始。
	summary, err := service.Accept(ctx, 10001, 7001, 0)
	if err != nil {
		t.Fatalf("Accept() after refresh error = %v", err)
	}
	if summary.State != StateAccepted || summary.Objectives[0].Current != 0 {
		t.Fatalf("re-accepted summary = %#v, want accepted from zero", summary)
	}
}

// TestAbandonQuest 验证放弃：清空进度、恢复可接取；不可放弃的配置被拒绝。
func TestAbandonQuest(t *testing.T) {
	repo, service, _ := newDailyQuestFixture()
	ctx := context.Background()

	summary, err := service.Abandon(ctx, 10001, 7001)
	if err != nil {
		t.Fatalf("Abandon() error = %v", err)
	}
	if summary.State != StateAvailable || summary.CanAbandon {
		t.Fatalf("summary = %#v, want available and no longer abandonable", summary)
	}
	if objectives := repo.playerObjectives[10001][7001]; len(objectives) != 0 {
		t.Fatalf("objectives after abandon = %#v, want cleared", objectives)
	}
	// 已经是可接取状态，再放弃应被拒绝。
	if _, err := service.Abandon(ctx, 10001, 7001); !errors.Is(err, ErrQuestCannotAbandon) {
		t.Fatalf("second abandon err = %v, want ErrQuestCannotAbandon", err)
	}

	cases := map[string]Template{
		"模板不可放弃": {QuestID: 7002, AcceptMode: "NPC", CanAbandon: false},
		"自动接取任务": {QuestID: 7003, AcceptMode: "AUTO", CanAbandon: true},
	}
	for name, template := range cases {
		repo.templates[template.QuestID] = template
		repo.playerQuests[10001][template.QuestID] = PlayerQuest{PlayerID: 10001, QuestID: template.QuestID, State: StateAccepted}
		if _, err := service.Abandon(ctx, 10001, template.QuestID); !errors.Is(err, ErrQuestCannotAbandon) {
			t.Fatalf("%s: err = %v, want ErrQuestCannotAbandon", name, err)
		}
	}
}

func findSummaryState(summaries []Summary, questID uint64) string {
	for _, summary := range summaries {
		if summary.QuestID == questID {
			return summary.State
		}
	}
	return ""
}

// TestWithBeforeLoadsStateOnce 锁定「操作前快照与操作本身共用一次加载」：
// 接取 / 交付 / 追踪 / 放弃各自只读取一次开启条件事实（全量状态加载的代表性开销）。
func TestWithBeforeLoadsStateOnce(t *testing.T) {
	ctx := context.Background()
	cases := map[string]func(*Service) error{
		"accept":  func(s *Service) error { _, _, err := s.AcceptWithBefore(ctx, 10001, 7001, 0); return err },
		"submit":  func(s *Service) error { _, _, err := s.SubmitWithBefore(ctx, 10001, 7001, 0); return err },
		"track":   func(s *Service) error { _, err := s.TrackWithBefore(ctx, 10001, 7001); return err },
		"abandon": func(s *Service) error { _, _, err := s.AbandonWithBefore(ctx, 10001, 7001); return err },
	}
	for name, run := range cases {
		repo, service, _ := newDailyQuestFixture()
		before := repo.fullFactsCalls
		if err := run(service); err != nil {
			t.Fatalf("%s error = %v", name, err)
		}
		if calls := repo.fullFactsCalls - before; calls != 1 {
			t.Fatalf("%s loaded facts %d times, want 1", name, calls)
		}
	}
}
