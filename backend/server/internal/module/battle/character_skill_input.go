package battle

import (
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/playerskill"
)

// CharacterBattleSkillInput 描述开战前人物技能可用性判定所需的武器与进度上下文。
type CharacterBattleSkillInput struct {
	WeaponType           string
	EquippedWeaponSkills []equipment.RuntimeWeaponSkill
	ProgressBySkillID    map[uint32]playerskill.Progress
}

// EmptyCharacterBattleSkillInput 返回空的战斗技能上下文，供测试或未注入进度时使用。
func EmptyCharacterBattleSkillInput() CharacterBattleSkillInput {
	return CharacterBattleSkillInput{
		ProgressBySkillID: map[uint32]playerskill.Progress{},
	}
}

// CharacterSkillMergeResult 是 mergePlayerCharacterSkills 的输出：可用技能、等级与学习态标记。
type CharacterSkillMergeResult struct {
	SkillIDs         []uint32
	SkillLevels      map[uint32]uint32
	LearningSkillIDs map[uint32]struct{}
}
