package battle

type battleSkillProgressState struct {
	SkillExp               uint32
	SkillLevel             uint32
	IsLearned              bool
	LearnExpRequired       uint32
	LearnExpPerUse         uint32
	UsedThisBattle         bool
	ExpGainedThisBattle    uint32
	NewlyLearnedThisBattle bool
}

func (b *activeBattle) initSkillProgressTracker(input CharacterBattleSkillInput, merge CharacterSkillMergeResult) {
	b.characterSkillInput = input
	b.skillProgressStates = make(map[uint32]*battleSkillProgressState)
	for skillID := range merge.LearningSkillIDs {
		meta, ok := getSkillLearningMeta(skillID)
		if !ok || meta.LearnExpRequired == 0 {
			continue
		}
		state := &battleSkillProgressState{
			LearnExpRequired: meta.LearnExpRequired,
			LearnExpPerUse:   meta.LearnExpPerUse,
			SkillLevel:       merge.SkillLevels[skillID],
		}
		if progress, exists := input.ProgressBySkillID[skillID]; exists {
			state.SkillExp = progress.SkillExp
			if progress.SkillLevel > 0 {
				state.SkillLevel = progress.SkillLevel
			}
			state.IsLearned = progress.IsLearned
		}
		if state.SkillLevel == 0 {
			state.SkillLevel = 1
		}
		b.skillProgressStates[skillID] = state
	}
}

func (b *activeBattle) recordWeaponSkillUse(actor *actorRuntime, requestedSkillID uint32, resolvedSkillID uint32) {
	if actor == nil || actor.unitClass != ActorUnitClassCharacter {
		return
	}
	if requestedSkillID == 0 || requestedSkillID != resolvedSkillID || resolvedSkillID == DefaultAttackSkillID {
		return
	}
	state, ok := b.skillProgressStates[resolvedSkillID]
	if !ok || state.IsLearned || state.LearnExpRequired == 0 {
		return
	}
	state.UsedThisBattle = true
}

// finalizeSkillProgressOnWin 在战斗胜利时按本场敌人数量结算武器技能学习经验。
func (b *activeBattle) finalizeSkillProgressOnWin() {
	enemyCount := b.initialEnemyCount
	if enemyCount == 0 {
		enemyCount = uint32(len(b.enemies))
	}
	if enemyCount == 0 {
		return
	}
	var characterActor *actorRuntime
	for _, ally := range b.allies {
		if ally != nil && ally.unitClass == ActorUnitClassCharacter {
			characterActor = ally
			break
		}
	}
	for skillID, state := range b.skillProgressStates {
		if !state.UsedThisBattle || state.IsLearned {
			continue
		}
		perEnemy := state.LearnExpPerUse
		if perEnemy == 0 {
			perEnemy = 1
		}
		expGain := enemyCount * perEnemy
		if expGain == 0 {
			continue
		}
		state.SkillExp += expGain
		state.ExpGainedThisBattle = expGain
		if state.SkillExp >= state.LearnExpRequired {
			state.IsLearned = true
			state.NewlyLearnedThisBattle = true
			if characterActor != nil {
				if level := characterActor.skillLevels[skillID]; level > state.SkillLevel {
					state.SkillLevel = level
				}
			}
		}
	}
}

func (b *activeBattle) collectSkillProgressUpdates() []SkillProgressUpdate {
	if len(b.skillProgressStates) == 0 {
		return nil
	}
	updates := make([]SkillProgressUpdate, 0, len(b.skillProgressStates))
	for skillID, state := range b.skillProgressStates {
		if state.ExpGainedThisBattle == 0 {
			continue
		}
		finalLevel := state.SkillLevel
		if finalLevel == 0 {
			finalLevel = 1
		}
		skillName := ""
		if skillDef, ok := getSkillDef(skillID); ok {
			skillName = skillDef.Name
		}
		updates = append(updates, SkillProgressUpdate{
			SkillID:          skillID,
			SkillName:        skillName,
			ExpGained:        state.ExpGainedThisBattle,
			FinalExp:         state.SkillExp,
			FinalLevel:       finalLevel,
			NewlyLearned:     state.NewlyLearnedThisBattle,
			LearnExpRequired: state.LearnExpRequired,
		})
	}
	return updates
}
