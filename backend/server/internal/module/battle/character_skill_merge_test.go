package battle

import (
	"testing"

	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/playerskill"
	"pocket-pet-remake/server/internal/module/skill"
)

func TestMergePlayerCharacterSkillsLearningRequiresEquippedWeapon(t *testing.T) {
	SetRuntimeSkillResolver(func(skillID uint32) (skill.RuntimeDefinition, bool) {
		if skillID != 1201 {
			return skill.RuntimeDefinition{}, false
		}
		return skill.RuntimeDefinition{
			SkillID:          1201,
			SkillCategory:    skill.CategoryWeapon,
			WeaponDiscipline: skill.DisciplineSword,
			LearnExpRequired: 10,
			LearnExpPerUse:   1,
			SkillName:        "试炼剑技",
		}, true
	})
	t.Cleanup(func() { SetRuntimeSkillResolver(nil) })

	input := CharacterBattleSkillInput{
		WeaponType: equipment.WeaponTypeSword,
		EquippedWeaponSkills: []equipment.RuntimeWeaponSkill{
			{SkillID: 1201, Level: 2},
		},
		ProgressBySkillID: map[uint32]playerskill.Progress{},
	}
	merged := mergePlayerCharacterSkills(&player.Profile{PlayerID: 1}, input)
	if _, learning := merged.LearningSkillIDs[1201]; !learning {
		t.Fatalf("LearningSkillIDs = %#v, want 1201 learning", merged.LearningSkillIDs)
	}
	if merged.SkillLevels[1201] != 2 {
		t.Fatalf("SkillLevels[1201] = %d, want 2", merged.SkillLevels[1201])
	}
}

func TestMergePlayerCharacterSkillsLearnedRequiresMatchingWeaponType(t *testing.T) {
	SetRuntimeSkillResolver(func(skillID uint32) (skill.RuntimeDefinition, bool) {
		if skillID != 1201 {
			return skill.RuntimeDefinition{}, false
		}
		return skill.RuntimeDefinition{
			SkillID:          1201,
			SkillCategory:    skill.CategoryWeapon,
			WeaponDiscipline: skill.DisciplineSword,
			LearnExpRequired: 10,
			LearnExpPerUse:   1,
			SkillName:        "试炼剑技",
		}, true
	})
	t.Cleanup(func() { SetRuntimeSkillResolver(nil) })

	learnedInput := CharacterBattleSkillInput{
		WeaponType:           equipment.WeaponTypeSword,
		EquippedWeaponSkills: nil,
		ProgressBySkillID: map[uint32]playerskill.Progress{
			1201: {SkillID: 1201, SkillLevel: 4, IsLearned: true},
		},
	}
	learnedMerged := mergePlayerCharacterSkills(&player.Profile{PlayerID: 1}, learnedInput)
	if _, ok := containsSkillID(learnedMerged.SkillIDs, 1201); !ok {
		t.Fatalf("learned SkillIDs = %#v, want 1201 when sword equipped", learnedMerged.SkillIDs)
	}

	mismatchInput := CharacterBattleSkillInput{
		WeaponType:           equipment.WeaponTypeStaff,
		EquippedWeaponSkills: nil,
		ProgressBySkillID: map[uint32]playerskill.Progress{
			1201: {SkillID: 1201, SkillLevel: 4, IsLearned: true},
		},
	}
	mismatchMerged := mergePlayerCharacterSkills(&player.Profile{PlayerID: 1}, mismatchInput)
	if _, ok := containsSkillID(mismatchMerged.SkillIDs, 1201); ok {
		t.Fatalf("mismatch SkillIDs = %#v, want 1201 excluded without sword", mismatchMerged.SkillIDs)
	}
}

func containsSkillID(skillIDs []uint32, skillID uint32) (int, bool) {
	for index, item := range skillIDs {
		if item == skillID {
			return index, true
		}
	}
	return -1, false
}
