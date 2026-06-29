package playerskill

import (
	"errors"
	"time"
)

var (
	// ErrInvalidSkillProgressInput 表示写入玩家技能进度时参数非法。
	ErrInvalidSkillProgressInput = errors.New("invalid player skill progress input")
)

// Progress 描述玩家对单个武器技能的持久化学习进度。
type Progress struct {
	PlayerID   uint64     `json:"player_id"`
	SkillID    uint32     `json:"skill_id"`
	SkillExp   uint32     `json:"skill_exp"`
	SkillLevel uint32     `json:"skill_level"`
	IsLearned  bool       `json:"is_learned"`
	LearnedAt  *time.Time `json:"learned_at,omitempty"`
}

// BattleUseUpdate 描述一场战斗结束后应落库的单条技能进度变更。
type BattleUseUpdate struct {
	SkillID              uint32 `json:"skill_id"`
	ExpGained            uint32 `json:"exp_gained"`
	FinalExp             uint32 `json:"final_exp"`
	FinalLevel           uint32 `json:"final_level"`
	NewlyLearned         bool   `json:"newly_learned"`
	LearnExpRequired     uint32 `json:"learn_exp_required"`
}
