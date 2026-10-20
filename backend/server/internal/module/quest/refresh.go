package quest

import (
	"strings"
	"time"
)

const (
	// RefreshCycleNone 表示不按时间刷新；可重复任务交付后立即可再次接取。
	RefreshCycleNone = "none"
	// RefreshCycleDaily 表示每天 RefreshHour 点刷新。
	RefreshCycleDaily = "daily"
	// RefreshCycleWeekly 表示每周一 RefreshHour 点刷新。
	RefreshCycleWeekly = "weekly"
)

// NormalizeRefreshCycle 把配置值规范化为合法周期；未知值按 none 处理。
func NormalizeRefreshCycle(cycle string) string {
	switch strings.ToLower(strings.TrimSpace(cycle)) {
	case RefreshCycleDaily:
		return RefreshCycleDaily
	case RefreshCycleWeekly:
		return RefreshCycleWeekly
	default:
		return RefreshCycleNone
	}
}

// NextRefreshTime 计算任务在 completedAt 之后的下一次刷新时刻。
// 不可重复任务返回 nil；可重复但周期为 none 时返回 completedAt 本身，表示立即可重做。
// 时间计算统一使用 completedAt 自带的时区（服务端本地时区），保证与运营理解的「每天 5 点」一致。
func NextRefreshTime(template Template, completedAt time.Time) *time.Time {
	if !template.IsRepeatable {
		return nil
	}
	hour := int(template.RefreshHour)
	if hour > 23 {
		hour = 23
	}
	loc := completedAt.Location()
	switch NormalizeRefreshCycle(template.RefreshCycle) {
	case RefreshCycleDaily:
		next := time.Date(completedAt.Year(), completedAt.Month(), completedAt.Day(), hour, 0, 0, 0, loc)
		if !next.After(completedAt) {
			next = next.AddDate(0, 0, 1)
		}
		return &next
	case RefreshCycleWeekly:
		next := time.Date(completedAt.Year(), completedAt.Month(), completedAt.Day(), hour, 0, 0, 0, loc)
		// time.Weekday 中周一为 1；往后推到最近一个「周一 hour 点」且必须晚于完成时间。
		daysUntilMonday := (int(time.Monday) - int(next.Weekday()) + 7) % 7
		next = next.AddDate(0, 0, daysUntilMonday)
		if !next.After(completedAt) {
			next = next.AddDate(0, 0, 7)
		}
		return &next
	default:
		immediate := completedAt
		return &immediate
	}
}

// shouldResetRepeatable 判断已完成的可重复任务是否已经到了重置时刻。
func shouldResetRepeatable(template Template, playerQuest *PlayerQuest, now time.Time) bool {
	if playerQuest == nil || playerQuest.State != StateCompleted || !template.IsRepeatable {
		return false
	}
	if playerQuest.NextRefreshAt == nil {
		return false
	}
	return !now.Before(*playerQuest.NextRefreshAt)
}

// canAbandon 判断玩家当前能否放弃该任务：模板允许、非自动接取、且处于进行中或待交付。
// 自动接取的任务放弃后会在下次加载时被重新自动接取，因此直接禁止，避免玩家看到「放弃无效」。
func canAbandon(template Template, playerQuest *PlayerQuest) bool {
	if !template.CanAbandon || template.AcceptMode == "AUTO" || playerQuest == nil {
		return false
	}
	return playerQuest.State == StateAccepted || playerQuest.State == StateReadyToSubmit
}
