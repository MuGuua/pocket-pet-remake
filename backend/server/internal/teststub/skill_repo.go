package teststub

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/skill"
)

// NewSkillRepository 提供系统技能模板 CRUD 与运行时读取的内存桩。
func NewSkillRepository() *SkillRepository {
	now := time.Now()
	repo := &SkillRepository{
		definitions: make(map[uint32]skill.AdminDetail, 8),
	}
	seeds := []skill.AdminUpsertInput{
		{SkillID: 1101, SkillCode: "character_slash", SkillName: "裂空斩", SkillCategory: "character", SkillType: "attack", ActivationMode: skill.ActivationModeActive, TargetType: "enemy_single", AnimationKey: "character_slash", SkillVisualID: "character_slash", CastColor: "#8FD6FF", ImpactColor: "#BDE9FF", Projectile: true, IsSkillAttack: true, IsEnabled: true, EnergyCost: 16, AttackPct: 135, ManaPct: 55, SpeedPct: 35, AllowCrit: true, ArmorBreakChancePct: 100, ArmorBreakRounds: 2, BleedChancePct: 45, BleedRounds: 2, BleedDamage: 3},
		{SkillID: 1001, SkillCode: "basic_attack", SkillName: "普通攻击", SkillCategory: "common", SkillType: "attack", ActivationMode: skill.ActivationModeActive, TargetType: "enemy_single", AnimationKey: "slash", IsBasicAttack: true, IsEnabled: true, AttackPct: 100, ManaPct: 35, SpeedPct: 35, AllowCrit: true},
		{SkillID: 1002, SkillCode: "pet_spark_burst", SkillName: "火花冲击", SkillCategory: "pet", SkillType: "attack", ActivationMode: skill.ActivationModeActive, TargetType: "enemy_all", AnimationKey: "burst", CastColor: "#FFAA5C", ImpactColor: "#FFD46B", Projectile: true, IsSkillAttack: true, IsEnabled: true, EnergyCost: 18, AttackPct: 120, ManaPct: 85, SpeedPct: 55, AllowCrit: true, BleedChancePct: 70, BleedRounds: 2, BleedDamage: 4, VulnerabilityChancePct: 100, VulnerabilityRounds: 2, VulnerabilityApplyPct: 12, ControlChancePct: 35, ControlRounds: 1, ControlStatusID: 11},
		{SkillID: 1003, SkillCode: "pet_vital_heal", SkillName: "活力治愈", SkillCategory: "pet", SkillType: "heal", ActivationMode: skill.ActivationModeActive, TargetType: "ally_single", PreferredTargetHP: "lowest", AnimationKey: "heal", CastColor: "#73F5A3", ImpactColor: "#B7FFD0", IsSkillAttack: true, IsEnabled: true, EnergyCost: 14, HealPct: 22, CritBoostRounds: 2, CritBoostPct: 20},
		{SkillID: 1004, SkillCode: "pet_arc_volley", SkillName: "弧光连射", SkillCategory: "pet", SkillType: "attack", ActivationMode: skill.ActivationModeActive, TargetType: "enemy_multi", TargetCount: 2, AnimationKey: "volley", CastColor: "#C6D1FF", ImpactColor: "#ECECFF", Projectile: true, IsSkillAttack: true, IsEnabled: true, EnergyCost: 16, AttackPct: 105, ManaPct: 40, SpeedPct: 25, AllowCrit: true, BleedChancePct: 40, BleedRounds: 2, BleedDamage: 2, ControlChancePct: 20, ControlRounds: 1, ControlStatusID: 12},
		{SkillID: 90001, SkillCode: "monster_wild_charge", SkillName: "野性撞击", SkillCategory: "monster", SkillType: "attack", ActivationMode: skill.ActivationModeActive, TargetType: "enemy_single", AnimationKey: "slash", CastColor: "#FFB88F", ImpactColor: "#FFDDD1", IsEnabled: true, AttackPct: 95, ManaPct: 20, FixedDamage: 2, AllowCrit: true, CurseChancePct: 40, CurseRounds: 2, CurseDamage: 3},
		{SkillID: 90002, SkillCode: "monster_claw_strike", SkillName: "利爪突袭", SkillCategory: "monster", SkillType: "attack", ActivationMode: skill.ActivationModeActive, TargetType: "enemy_single", AnimationKey: "volley", CastColor: "#FF9E85", ImpactColor: "#FFC7BA", Projectile: true, IsEnabled: true, EnergyCost: 12, AttackPct: 110, ManaPct: 30, SpeedPct: 20, AllowCrit: true, BleedChancePct: 50, BleedRounds: 2, BleedDamage: 3, SlowChancePct: 100, SlowRounds: 2, SlowMultiplierPct: 70, ControlChancePct: 30, ControlRounds: 1, ControlStatusID: 12},
		{SkillID: 90003, SkillCode: "monster_wild_regen", SkillName: "野性回春", SkillCategory: "monster", SkillType: "heal", ActivationMode: skill.ActivationModeActive, TargetType: "ally_single", PreferredTargetHP: "lowest", AnimationKey: "heal", CastColor: "#84F8B3", ImpactColor: "#C8FFE0", IsSkillAttack: true, IsEnabled: true, EnergyCost: 10, HealPct: 18},
	}
	for _, seed := range seeds {
		repo.definitions[seed.SkillID] = buildStubSkillDetail(seed.Normalize(), now)
	}
	return repo
}

type SkillRepository struct {
	mu          sync.RWMutex
	definitions map[uint32]skill.AdminDetail
}

func (r *SkillRepository) ListForAdmin(_ context.Context, query skill.AdminListQuery) (*skill.AdminList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = query.Normalize()
	items := make([]skill.AdminSummary, 0, len(r.definitions))
	for _, current := range r.definitions {
		if query.SkillID > 0 && current.SkillID != query.SkillID {
			continue
		}
		if query.Name != "" && !strings.Contains(current.SkillName, query.Name) && !strings.Contains(current.SkillCode, query.Name) {
			continue
		}
		if query.Category != "" && current.SkillCategory != query.Category {
			continue
		}
		if query.Type != "" && current.SkillType != query.Type {
			continue
		}
		if query.ActivationMode != "" && current.ActivationMode != query.ActivationMode {
			continue
		}
		if query.Enabled != nil && current.IsEnabled != *query.Enabled {
			continue
		}
		items = append(items, adminSummaryFromSkillDetail(current))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SkillID == items[j].SkillID {
			return items[i].SkillName < items[j].SkillName
		}
		return items[i].SkillID < items[j].SkillID
	})
	return &skill.AdminList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *SkillRepository) FindForAdmin(_ context.Context, skillID uint32) (*skill.AdminDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.definitions[skillID]
	if !ok {
		return nil, nil
	}
	copied := current
	return &copied, nil
}

func (r *SkillRepository) CreateForAdmin(_ context.Context, input skill.AdminUpsertInput) (*skill.AdminDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.definitions[input.SkillID]; exists {
		return nil, skill.ErrSkillDefinitionConflict
	}
	detail := buildStubSkillDetail(input.Normalize(), time.Now())
	r.definitions[input.SkillID] = detail
	copied := detail
	return &copied, nil
}

func (r *SkillRepository) UpdateForAdmin(_ context.Context, skillID uint32, input skill.AdminUpsertInput) (*skill.AdminDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.definitions[skillID]
	if !ok {
		return nil, nil
	}
	detail := buildStubSkillDetail(input.Normalize(), current.CreatedAt)
	detail.SkillID = skillID
	detail.UpdatedAt = time.Now()
	r.definitions[skillID] = detail
	copied := detail
	return &copied, nil
}

func (r *SkillRepository) DeleteForAdmin(_ context.Context, skillID uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.definitions[skillID]; !ok {
		return skill.ErrSkillDefinitionNotFound
	}
	delete(r.definitions, skillID)
	return nil
}

func (r *SkillRepository) ListEnabledRuntimeDefinitions(_ context.Context) ([]skill.RuntimeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]skill.RuntimeDefinition, 0, len(r.definitions))
	for _, current := range r.definitions {
		if !current.IsEnabled {
			continue
		}
		items = append(items, runtimeFromSkillDetail(current))
	}
	return items, nil
}

func (r *SkillRepository) MapUsableSkillIDs(_ context.Context, skillIDs []uint32) (map[uint32]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[uint32]bool, len(skillIDs))
	for _, skillID := range skillIDs {
		current, ok := r.definitions[skillID]
		if ok && current.IsEnabled {
			result[skillID] = true
		}
	}
	return result, nil
}

func (r *SkillRepository) MapSkillCategoriesByIDs(_ context.Context, skillIDs []uint32) (map[uint32]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[uint32]string, len(skillIDs))
	for _, skillID := range skillIDs {
		current, ok := r.definitions[skillID]
		if ok {
			result[skillID] = current.SkillCategory
		}
	}
	return result, nil
}

func (r *SkillRepository) MapSkillWeaponDisciplinesByIDs(_ context.Context, skillIDs []uint32) (map[uint32]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[uint32]string, len(skillIDs))
	for _, skillID := range skillIDs {
		current, ok := r.definitions[skillID]
		if ok {
			result[skillID] = current.WeaponDiscipline
		}
	}
	return result, nil
}

func buildStubSkillDetail(input skill.AdminUpsertInput, createdAt time.Time) skill.AdminDetail {
	statusText := "停用"
	if input.IsEnabled {
		statusText = "启用"
	}
	return skill.AdminDetail{
		SkillID: input.SkillID, SkillCode: input.SkillCode, SkillName: input.SkillName,
		SkillCategory: input.SkillCategory, WeaponDiscipline: input.WeaponDiscipline,
		LearnExpRequired: input.LearnExpRequired, LearnExpPerUse: input.LearnExpPerUse,
		SkillType: input.SkillType, ActivationMode: input.ActivationMode,
		Description: input.Description, AcquireMethod: input.AcquireMethod,
		IsBasicAttack: input.IsBasicAttack, IsEnabled: input.IsEnabled, StatusText: statusText, SortWeight: input.SortWeight,
		TargetRule: skill.AdminTargetRule{TargetType: input.TargetType, TargetCount: input.TargetCount, PreferredTargetHP: input.PreferredTargetHP},
		Formula: skill.AdminFormula{
			AttackPct: input.AttackPct, ManaPct: input.ManaPct, DefensePct: input.DefensePct, SpeedPct: input.SpeedPct,
			TargetCurrentHPPct: input.TargetCurrentHPPct, FixedDamage: input.FixedDamage, HealPct: input.HealPct, FixedHeal: input.FixedHeal,
			EnergyCost: input.EnergyCost, IsSkillAttack: input.IsSkillAttack, AllowCrit: input.AllowCrit, IgnoreDefense: input.IgnoreDefense,
		},
		StatusEffects: skill.AdminStatusEffects{
			ArmorBreakPct: input.ArmorBreakPct, VulnerabilityPct: input.VulnerabilityPct,
			BleedChancePct: input.BleedChancePct, BleedRounds: input.BleedRounds, BleedDamage: input.BleedDamage,
			SealChancePct: input.SealChancePct, SealRounds: input.SealRounds,
			VulnerabilityChancePct: input.VulnerabilityChancePct, VulnerabilityRounds: input.VulnerabilityRounds, VulnerabilityApplyPct: input.VulnerabilityApplyPct,
			ArmorBreakChancePct: input.ArmorBreakChancePct, ArmorBreakRounds: input.ArmorBreakRounds,
			SlowChancePct: input.SlowChancePct, SlowRounds: input.SlowRounds, SlowMultiplierPct: input.SlowMultiplierPct,
			CritBoostRounds: input.CritBoostRounds, CritBoostPct: input.CritBoostPct,
			CurseChancePct: input.CurseChancePct, CurseRounds: input.CurseRounds, CurseDamage: input.CurseDamage, CurseManaPct: input.CurseManaPct,
			ControlChancePct: input.ControlChancePct, ControlRounds: input.ControlRounds, ControlStatusID: input.ControlStatusID,
		},
		Presentation: skill.AdminPresentation{AnimationKey: input.AnimationKey, SkillVisualID: input.SkillVisualID, CastColor: input.CastColor, ImpactColor: input.ImpactColor, Projectile: input.Projectile},
		CreatedAt:    createdAt, UpdatedAt: time.Now(),
	}
}

func adminSummaryFromSkillDetail(detail skill.AdminDetail) skill.AdminSummary {
	return skill.AdminSummary{
		SkillID: detail.SkillID, SkillCode: detail.SkillCode, SkillName: detail.SkillName,
		SkillCategory: detail.SkillCategory, SkillType: detail.SkillType, ActivationMode: detail.ActivationMode, TargetType: detail.TargetRule.TargetType,
		EnergyCost: detail.Formula.EnergyCost, IsBasicAttack: detail.IsBasicAttack, IsEnabled: detail.IsEnabled, StatusText: detail.StatusText,
		CreatedAt: detail.CreatedAt, UpdatedAt: detail.UpdatedAt,
	}
}

func runtimeFromSkillDetail(detail skill.AdminDetail) skill.RuntimeDefinition {
	return skill.RuntimeDefinition{
		SkillID: detail.SkillID, SkillType: detail.SkillType, ActivationMode: detail.ActivationMode, SkillCategory: detail.SkillCategory, WeaponDiscipline: detail.WeaponDiscipline,
		LearnExpRequired: detail.LearnExpRequired, LearnExpPerUse: detail.LearnExpPerUse,
		SkillName:  detail.SkillName,
		TargetType: detail.TargetRule.TargetType, TargetCount: detail.TargetRule.TargetCount, PreferredTargetHP: detail.TargetRule.PreferredTargetHP,
		AnimationKey: detail.Presentation.AnimationKey, SkillVisualID: detail.Presentation.SkillVisualID, CastColor: detail.Presentation.CastColor, ImpactColor: detail.Presentation.ImpactColor, Projectile: detail.Presentation.Projectile,
		IsSkillAttack: detail.Formula.IsSkillAttack, EnergyCost: detail.Formula.EnergyCost,
		SkillMult: detail.Formula.SkillMult, SkillCritAdd: detail.Formula.SkillCritAdd,
		AttackPct: detail.Formula.AttackPct, ManaPct: detail.Formula.ManaPct, DefensePct: detail.Formula.DefensePct, SpeedPct: detail.Formula.SpeedPct,
		TargetCurrentHPPct: detail.Formula.TargetCurrentHPPct, FixedDamage: detail.Formula.FixedDamage, HealPct: detail.Formula.HealPct, FixedHeal: detail.Formula.FixedHeal,
		AllowCrit: detail.Formula.AllowCrit, IgnoreDefense: detail.Formula.IgnoreDefense,
		ArmorBreakPct: detail.StatusEffects.ArmorBreakPct, VulnerabilityPct: detail.StatusEffects.VulnerabilityPct,
		BleedChancePct: detail.StatusEffects.BleedChancePct, BleedRounds: detail.StatusEffects.BleedRounds, BleedDamage: detail.StatusEffects.BleedDamage,
		SealChancePct: detail.StatusEffects.SealChancePct, SealRounds: detail.StatusEffects.SealRounds,
		VulnerabilityChancePct: detail.StatusEffects.VulnerabilityChancePct, VulnerabilityRounds: detail.StatusEffects.VulnerabilityRounds, VulnerabilityApplyPct: detail.StatusEffects.VulnerabilityApplyPct,
		ArmorBreakChancePct: detail.StatusEffects.ArmorBreakChancePct, ArmorBreakRounds: detail.StatusEffects.ArmorBreakRounds,
		SlowChancePct: detail.StatusEffects.SlowChancePct, SlowRounds: detail.StatusEffects.SlowRounds, SlowMultiplierPct: detail.StatusEffects.SlowMultiplierPct,
		CritBoostRounds: detail.StatusEffects.CritBoostRounds, CritBoostPct: detail.StatusEffects.CritBoostPct,
		CurseChancePct: detail.StatusEffects.CurseChancePct, CurseRounds: detail.StatusEffects.CurseRounds, CurseDamage: detail.StatusEffects.CurseDamage, CurseManaPct: detail.StatusEffects.CurseManaPct,
		ControlChancePct: detail.StatusEffects.ControlChancePct, ControlRounds: detail.StatusEffects.ControlRounds, ControlStatusID: detail.StatusEffects.ControlStatusID,
	}
}
