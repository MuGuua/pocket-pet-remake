package quest

import "errors"

var (
	ErrQuestTemplateNotFound  = errors.New("quest template not found")
	ErrQuestLocked            = errors.New("quest locked")
	ErrQuestNotAvailable      = errors.New("quest not available")
	ErrQuestNotReady          = errors.New("quest not ready to submit")
	ErrQuestAcceptNPCMismatch = errors.New("quest accept npc mismatch")
	ErrQuestSubmitNPCMismatch = errors.New("quest submit npc mismatch")
	// ErrQuestCannotAbandon 表示任务模板不允许放弃、任务为自动接取，或当前状态不可放弃。
	ErrQuestCannotAbandon = errors.New("quest cannot be abandoned")
)
