package battle

import (
	"strings"

	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/playerskill"
	"pocket-pet-remake/server/internal/module/skill"
)

type skillLearningMeta struct {
	Category         string
	WeaponDiscipline string
	LearnExpRequired uint32
	LearnExpPerUse   uint32
}

func getSkillLearningMeta(skillID uint32) (skillLearningMeta, bool) {
	if runtimeSkillResolver != nil {
		if runtimeDef, ok := runtimeSkillResolver(skillID); ok {
			perUse := runtimeDef.LearnExpPerUse
			if perUse == 0 {
				perUse = 1
			}
			return skillLearningMeta{
				Category:         runtimeDef.SkillCategory,
				WeaponDiscipline: runtimeDef.WeaponDiscipline,
				LearnExpRequired: runtimeDef.LearnExpRequired,
				LearnExpPerUse:   perUse,
			}, true
		}
	}
	return skillLearningMeta{}, false
}

func mergePlayerCharacterSkills(profile *player.Profile, input CharacterBattleSkillInput) CharacterSkillMergeResult {
	result := CharacterSkillMergeResult{
		SkillIDs:         playerCharacterSkillIDs(profile),
		SkillLevels:      make(map[uint32]uint32),
		LearningSkillIDs: make(map[uint32]struct{}),
	}
	seen := make(map[uint32]struct{}, len(result.SkillIDs)+len(input.EquippedWeaponSkills))
	for _, skillID := range result.SkillIDs {
		seen[skillID] = struct{}{}
	}

	equippedByID := make(map[uint32]equipment.RuntimeWeaponSkill, len(input.EquippedWeaponSkills))
	for _, weaponSkill := range input.EquippedWeaponSkills {
		if weaponSkill.SkillID == 0 {
			continue
		}
		equippedByID[weaponSkill.SkillID] = weaponSkill
	}
	weaponType := strings.TrimSpace(input.WeaponType)
	if input.ProgressBySkillID == nil {
		input.ProgressBySkillID = map[uint32]playerskill.Progress{}
	}

	appendSkill := func(skillID uint32, level uint32, learning bool) {
		if skillID == 0 {
			return
		}
		if _, exists := seen[skillID]; !exists {
			seen[skillID] = struct{}{}
			result.SkillIDs = append(result.SkillIDs, skillID)
		}
		if level > 0 {
			result.SkillLevels[skillID] = level
		}
		if learning {
			result.LearningSkillIDs[skillID] = struct{}{}
		}
	}

	for skillID, weaponSkill := range equippedByID {
		meta, ok := getSkillLearningMeta(skillID)
		if !ok || meta.Category != skill.CategoryWeapon {
			appendSkill(skillID, weaponSkill.Level, false)
			continue
		}
		progress, hasProgress := input.ProgressBySkillID[skillID]
		if hasProgress && progress.IsLearned {
			if !skill.MatchesWeaponDiscipline(meta.WeaponDiscipline, weaponType) {
				continue
			}
			level := progress.SkillLevel
			if level == 0 {
				level = 1
			}
			appendSkill(skillID, level, false)
			continue
		}
		appendSkill(skillID, weaponSkill.Level, true)
	}

	for skillID, progress := range input.ProgressBySkillID {
		if !progress.IsLearned {
			continue
		}
		if _, exists := seen[skillID]; exists {
			continue
		}
		meta, ok := getSkillLearningMeta(skillID)
		if !ok || meta.Category != skill.CategoryWeapon {
			continue
		}
		if !skill.MatchesWeaponDiscipline(meta.WeaponDiscipline, weaponType) {
			continue
		}
		level := progress.SkillLevel
		if level == 0 {
			level = 1
		}
		appendSkill(skillID, level, false)
	}

	return result
}

func addSkillIDUnique(skillIDs []uint32, seen map[uint32]struct{}, skillID uint32) []uint32 {
	if skillID == 0 {
		return skillIDs
	}
	if _, exists := seen[skillID]; exists {
		return skillIDs
	}
	seen[skillID] = struct{}{}
	return append(skillIDs, skillID)
}
