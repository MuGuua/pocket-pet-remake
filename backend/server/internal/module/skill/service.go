package skill

import (
	"context"
	"fmt"
	"sync"
)

// Service 负责系统技能模板 CRUD、运行时缓存与引用校验。
type Service struct {
	repo  Repository
	mu    sync.RWMutex
	cache map[uint32]RuntimeDefinition
}

// NewService 构造技能模板服务。
func NewService(repo Repository) *Service {
	return &Service{repo: repo, cache: map[uint32]RuntimeDefinition{}}
}

// RefreshRuntimeCache 从数据库重建战斗运行时缓存。
func (s *Service) RefreshRuntimeCache(ctx context.Context) error {
	if s.repo == nil {
		return fmt.Errorf("skill repository is unavailable")
	}
	definitions, err := s.repo.ListEnabledRuntimeDefinitions(ctx)
	if err != nil {
		return err
	}
	nextCache := make(map[uint32]RuntimeDefinition, len(definitions))
	for _, item := range definitions {
		nextCache[item.SkillID] = item
	}
	s.mu.Lock()
	s.cache = nextCache
	s.mu.Unlock()
	return nil
}

// GetRuntimeDefinition 读取启用中的运行时技能模板。
func (s *Service) GetRuntimeDefinition(skillID uint32) (RuntimeDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.cache[skillID]
	return item, ok
}

// BattleResolver 返回供 battle 模块注册的技能解析函数。
func (s *Service) BattleResolver() func(uint32) (RuntimeDefinition, bool) {
	return func(skillID uint32) (RuntimeDefinition, bool) {
		return s.GetRuntimeDefinition(skillID)
	}
}

// ValidateEnabledSkillIDs 校验 skill_id 均存在于启用中的系统技能模板。
func (s *Service) ValidateEnabledSkillIDs(ctx context.Context, skillIDs []uint32) error {
	if len(skillIDs) == 0 {
		return nil
	}
	usableMap, err := s.repo.MapUsableSkillIDs(ctx, skillIDs)
	if err != nil {
		return err
	}
	for _, skillID := range skillIDs {
		if skillID == 0 || !usableMap[skillID] {
			return fmt.Errorf("%w: skill_id=%d", ErrInvalidSkillReference, skillID)
		}
	}
	return nil
}

// ValidateWeaponSkillReferences 校验装备引用的武器技能均存在、启用且分类为 weapon。
func (s *Service) ValidateWeaponSkillReferences(ctx context.Context, skillIDs []uint32) error {
	if len(skillIDs) == 0 {
		return nil
	}
	usableMap, err := s.repo.MapUsableSkillIDs(ctx, skillIDs)
	if err != nil {
		return err
	}
	categoryMap, err := s.repo.MapSkillCategoriesByIDs(ctx, skillIDs)
	if err != nil {
		return err
	}
	for _, skillID := range skillIDs {
		if skillID == 0 || !usableMap[skillID] {
			return fmt.Errorf("%w: skill_id=%d", ErrInvalidSkillReference, skillID)
		}
		if categoryMap[skillID] != CategoryWeapon {
			return fmt.Errorf("%w: skill_id=%d category=%s", ErrInvalidSkillReference, skillID, categoryMap[skillID])
		}
	}
	disciplineMap, err := s.repo.MapSkillWeaponDisciplinesByIDs(ctx, skillIDs)
	if err != nil {
		return err
	}
	for _, skillID := range skillIDs {
		if !IsValidWeaponDiscipline(disciplineMap[skillID]) {
			return fmt.Errorf("%w: skill_id=%d missing weapon_discipline", ErrInvalidSkillReference, skillID)
		}
	}
	return nil
}

// ListAdminSkillDefinitions 返回后台技能模板分页列表。
func (s *Service) ListAdminSkillDefinitions(ctx context.Context, query AdminListQuery) (*AdminList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminList{Items: []AdminSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

// GetAdminSkillDefinitionDetail 返回单个技能模板详情。
func (s *Service) GetAdminSkillDefinitionDetail(ctx context.Context, skillID uint32) (*AdminDetail, error) {
	result, err := s.repo.FindForAdmin(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrSkillDefinitionNotFound
	}
	return result, nil
}

// CreateAdminSkillDefinition 新增系统技能模板并刷新运行时缓存。
func (s *Service) CreateAdminSkillDefinition(ctx context.Context, input AdminUpsertInput) (*AdminDetail, error) {
	input = input.Normalize()
	if err := validateAdminSkillDefinitionInput(input); err != nil {
		return nil, ErrInvalidAdminSkillDefinitionInput
	}
	created, err := s.repo.CreateForAdmin(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.RefreshRuntimeCache(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateAdminSkillDefinition 更新系统技能模板并刷新运行时缓存。
func (s *Service) UpdateAdminSkillDefinition(ctx context.Context, skillID uint32, input AdminUpsertInput) (*AdminDetail, error) {
	input = input.Normalize()
	if skillID == 0 {
		return nil, ErrInvalidAdminSkillDefinitionInput
	}
	input.SkillID = skillID
	if err := validateAdminSkillDefinitionInput(input); err != nil {
		return nil, ErrInvalidAdminSkillDefinitionInput
	}
	updated, err := s.repo.UpdateForAdmin(ctx, skillID, input)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrSkillDefinitionNotFound
	}
	if err := s.RefreshRuntimeCache(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteAdminSkillDefinition 删除系统技能模板并刷新运行时缓存。
func (s *Service) DeleteAdminSkillDefinition(ctx context.Context, skillID uint32) error {
	if err := s.repo.DeleteForAdmin(ctx, skillID); err != nil {
		return err
	}
	return s.RefreshRuntimeCache(ctx)
}

// validateAdminSkillDefinitionInput 收口后台技能模板的最小业务约束，
// 避免把“被动技能也可主动释放”这类非法配置写入权威技能库。
func validateAdminSkillDefinitionInput(input AdminUpsertInput) error {
	if input.SkillID == 0 || input.SkillName == "" {
		return ErrInvalidAdminSkillDefinitionInput
	}
	if !IsValidPassiveAttrKey(input.PassiveAttrKey) || !IsValidPassiveAttrMode(input.PassiveAttrMode) {
		return ErrInvalidAdminSkillDefinitionInput
	}
	if input.PassiveAttrKey == "" {
		if input.PassiveAttrMode != "" || input.PassiveAttrValue != 0 {
			return ErrInvalidAdminSkillDefinitionInput
		}
	} else {
		if input.ActivationMode != ActivationModePassive {
			return ErrInvalidAdminSkillDefinitionInput
		}
		if input.PassiveAttrMode == "" || input.PassiveAttrValue <= 0 {
			return ErrInvalidAdminSkillDefinitionInput
		}
		if input.PassiveAttrMode == PassiveAttrModePercent && !SupportsPassiveAttrPercent(input.PassiveAttrKey) {
			return ErrInvalidAdminSkillDefinitionInput
		}
	}
	if input.ActivationMode == ActivationModePassive {
		if input.IsBasicAttack {
			return ErrInvalidAdminSkillDefinitionInput
		}
		if input.TargetType != "self" {
			return ErrInvalidAdminSkillDefinitionInput
		}
		if input.TargetCount != 0 {
			return ErrInvalidAdminSkillDefinitionInput
		}
		if input.PreferredTargetHP != "" {
			return ErrInvalidAdminSkillDefinitionInput
		}
		if input.EnergyCost != 0 {
			return ErrInvalidAdminSkillDefinitionInput
		}
	}
	return nil
}
