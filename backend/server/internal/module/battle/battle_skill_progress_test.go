package battle

import (
	"testing"

	"pocket-pet-remake/server/internal/module/skill"
)

func TestFinalizeSkillProgressOnWinUsesEnemyCount(t *testing.T) {
	SetRuntimeSkillResolver(func(skillID uint32) (skill.RuntimeDefinition, bool) {
		if skillID != 1201 {
			return skill.RuntimeDefinition{}, false
		}
		return skill.RuntimeDefinition{
			SkillID:          1201,
			SkillCategory:    skill.CategoryWeapon,
			SkillName:        "试炼剑技",
			LearnExpRequired: 100,
			LearnExpPerUse:   1,
		}, true
	})
	t.Cleanup(func() { SetRuntimeSkillResolver(nil) })

	battle := &activeBattle{
		initialEnemyCount: 3,
		allies: []*actorRuntime{
			{
				actorID:     1,
				unitClass:   ActorUnitClassCharacter,
				skillLevels: map[uint32]uint32{1201: 2},
			},
		},
		skillProgressStates: map[uint32]*battleSkillProgressState{
			1201: {
				LearnExpRequired: 100,
				LearnExpPerUse:   1,
				SkillLevel:       2,
				UsedThisBattle:   true,
			},
		},
	}
	battle.finalizeSkillProgressOnWin()
	state := battle.skillProgressStates[1201]
	if state.ExpGainedThisBattle != 3 {
		t.Fatalf("ExpGainedThisBattle = %d, want 3", state.ExpGainedThisBattle)
	}
	if state.SkillExp != 3 {
		t.Fatalf("SkillExp = %d, want 3", state.SkillExp)
	}
	updates := battle.collectSkillProgressUpdates()
	if len(updates) != 1 || updates[0].SkillName != "试炼剑技" || updates[0].FinalExp != 3 {
		t.Fatalf("collectSkillProgressUpdates() = %#v, want one update with exp 3/100", updates)
	}
}

func TestFinalizeSkillProgressOnWinSkipsUnusedSkill(t *testing.T) {
	battle := &activeBattle{
		initialEnemyCount: 2,
		skillProgressStates: map[uint32]*battleSkillProgressState{
			1201: {
				LearnExpRequired: 100,
				LearnExpPerUse:   1,
				UsedThisBattle:   false,
			},
		},
	}
	battle.finalizeSkillProgressOnWin()
	if battle.skillProgressStates[1201].ExpGainedThisBattle != 0 {
		t.Fatalf("ExpGainedThisBattle = %d, want 0 for unused skill", battle.skillProgressStates[1201].ExpGainedThisBattle)
	}
}
