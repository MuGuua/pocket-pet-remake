package quest

import "errors"

var (
	ErrQuestTemplateNotFound  = errors.New("quest template not found")
	ErrQuestLocked            = errors.New("quest locked")
	ErrQuestNotAvailable      = errors.New("quest not available")
	ErrQuestNotReady          = errors.New("quest not ready to submit")
	ErrQuestAcceptNPCMismatch = errors.New("quest accept npc mismatch")
	ErrQuestSubmitNPCMismatch = errors.New("quest submit npc mismatch")
)
