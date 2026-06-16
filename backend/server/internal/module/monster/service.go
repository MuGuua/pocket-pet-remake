package monster

import (
	"context"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/skill"
)

// CapturePetValidator 校验怪物关联的 capture_pet_id 是否为合法野外捕捉模板。
type CapturePetValidator interface {
	ValidateCapturePetTemplate(ctx context.Context, petID uint32) error
}

// Service 负责怪物模板、遭遇配置与战斗运行时解析。
type Service struct {
	repo                Repository
	skillService        *skill.Service
	capturePetValidator CapturePetValidator
	battleRewardCache   *battleRewardCache
}

func NewService(repo Repository, skillService *skill.Service, capturePetValidator CapturePetValidator) *Service {
	return &Service{
		repo:                repo,
		skillService:        skillService,
		capturePetValidator: capturePetValidator,
		battleRewardCache:   newBattleRewardCache(),
	}
}

func (s *Service) ValidateEnabledSkillIDs(ctx context.Context, skillIDs []uint32) error {
	if s.skillService == nil || len(skillIDs) == 0 {
		return nil
	}
	return s.skillService.ValidateEnabledSkillIDs(ctx, skillIDs)
}

func (s *Service) ValidateEnabledMonsterIDs(ctx context.Context, monsterIDs []uint32) error {
	if len(monsterIDs) == 0 {
		return fmt.Errorf("%w: empty monster list", ErrInvalidMonsterReference)
	}
	usableMap, err := s.repo.MapUsableMonsterIDs(ctx, monsterIDs)
	if err != nil {
		return err
	}
	for _, monsterID := range monsterIDs {
		if monsterID == 0 || !usableMap[monsterID] {
			return fmt.Errorf("%w: monster_id=%d", ErrInvalidMonsterReference, monsterID)
		}
	}
	return nil
}

// ResolveEncounterForEntity 按世界 entity_id 解析启用中的遭遇与怪物模板。
func (s *Service) ResolveEncounterForEntity(ctx context.Context, entityID uint64) (*RuntimeEncounter, error) {
	if s.repo == nil || entityID == 0 {
		return nil, nil
	}
	return s.repo.FindRuntimeEncounter(ctx, entityID)
}

// BuildWildEncounterConfig 读取指定地图的暗雷配置，供进图/切图下发客户端。
func (s *Service) BuildWildEncounterConfig(ctx context.Context, sceneID uint32) (*RuntimeWildEncounterConfig, error) {
	if s.repo == nil || sceneID == 0 {
		return &RuntimeWildEncounterConfig{SceneID: sceneID}, nil
	}
	config, err := s.repo.FindRuntimeWildEncounterConfig(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return &RuntimeWildEncounterConfig{SceneID: sceneID}, nil
	}
	return config, nil
}

// ResolveWildEncounterForScene 解析地图暗雷遭遇，供客户端上报后服务端权威开战。
func (s *Service) ResolveWildEncounterForScene(ctx context.Context, sceneID uint32) (*RuntimeWildEncounter, error) {
	if s.repo == nil || sceneID == 0 {
		return nil, nil
	}
	return s.repo.FindRuntimeWildEncounter(ctx, sceneID)
}

// GetCaptureConfig 返回怪物模板的捕捉配置，供战斗捕捉判定与发放链路使用。
func (s *Service) GetCaptureConfig(ctx context.Context, monsterID uint32) (*CaptureConfig, error) {
	if s.repo == nil || monsterID == 0 {
		return nil, nil
	}
	return s.repo.FindCaptureConfig(ctx, monsterID)
}

func (s *Service) ListAdminMonsterDefinitions(ctx context.Context, query AdminDefinitionListQuery) (*AdminDefinitionList, error) {
	result, err := s.repo.ListDefinitionsForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminDefinitionList{Items: []AdminDefinitionSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminMonsterDefinitionDetail(ctx context.Context, monsterID uint32) (*AdminDefinitionDetail, error) {
	result, err := s.repo.FindDefinitionForAdmin(ctx, monsterID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrMonsterDefinitionNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminMonsterDefinition(ctx context.Context, input AdminUpsertDefinitionInput) (*AdminDefinitionDetail, error) {
	input = input.Normalize()
	if input.MonsterID == 0 || input.MonsterName == "" {
		return nil, ErrInvalidAdminMonsterDefinitionInput
	}
	if input.IsEnabled && strings.TrimSpace(input.SkinID) == "" {
		return nil, ErrInvalidAdminMonsterDefinitionInput
	}
	if err := s.validateCaptureConfig(ctx, input); err != nil {
		return nil, err
	}
	if err := s.ValidateEnabledSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	return s.repo.CreateDefinitionForAdmin(ctx, input)
}

func (s *Service) UpdateAdminMonsterDefinition(ctx context.Context, monsterID uint32, input AdminUpsertDefinitionInput) (*AdminDefinitionDetail, error) {
	input = input.Normalize()
	if monsterID == 0 || input.MonsterName == "" {
		return nil, ErrInvalidAdminMonsterDefinitionInput
	}
	if input.IsEnabled && strings.TrimSpace(input.SkinID) == "" {
		return nil, ErrInvalidAdminMonsterDefinitionInput
	}
	if err := s.validateCaptureConfig(ctx, input); err != nil {
		return nil, err
	}
	if err := s.ValidateEnabledSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateDefinitionForAdmin(ctx, monsterID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrMonsterDefinitionNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminMonsterDefinition(ctx context.Context, monsterID uint32) error {
	return s.repo.DeleteDefinitionForAdmin(ctx, monsterID)
}

func (s *Service) ListAdminMonsterEncounters(ctx context.Context, query AdminEncounterListQuery) (*AdminEncounterList, error) {
	result, err := s.repo.ListEncountersForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminEncounterList{Items: []AdminEncounterSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminMonsterEncounterDetail(ctx context.Context, entityID uint64) (*AdminEncounterDetail, error) {
	result, err := s.repo.FindEncounterForAdmin(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrMonsterEncounterNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminMonsterEncounter(ctx context.Context, input AdminUpsertEncounterInput) (*AdminEncounterDetail, error) {
	input = input.Normalize()
	if input.EntityID == 0 || input.EncounterName == "" || len(input.SpawnMonsterIDs) == 0 {
		return nil, ErrInvalidAdminMonsterEncounterInput
	}
	if err := s.ValidateEnabledMonsterIDs(ctx, input.SpawnMonsterIDs); err != nil {
		return nil, err
	}
	return s.repo.CreateEncounterForAdmin(ctx, input)
}

func (s *Service) UpdateAdminMonsterEncounter(ctx context.Context, entityID uint64, input AdminUpsertEncounterInput) (*AdminEncounterDetail, error) {
	input = input.Normalize()
	if entityID == 0 || input.EncounterName == "" || len(input.SpawnMonsterIDs) == 0 {
		return nil, ErrInvalidAdminMonsterEncounterInput
	}
	if err := s.ValidateEnabledMonsterIDs(ctx, input.SpawnMonsterIDs); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateEncounterForAdmin(ctx, entityID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrMonsterEncounterNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminMonsterEncounter(ctx context.Context, entityID uint64) error {
	return s.repo.DeleteEncounterForAdmin(ctx, entityID)
}

func (s *Service) ListAdminSceneWildEncounters(ctx context.Context, query AdminWildEncounterListQuery) (*AdminWildEncounterList, error) {
	result, err := s.repo.ListWildEncountersForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminWildEncounterList{Items: []AdminWildEncounterSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminSceneWildEncounterDetail(ctx context.Context, sceneID uint32) (*AdminWildEncounterDetail, error) {
	result, err := s.repo.FindWildEncounterForAdmin(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrSceneWildEncounterNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminSceneWildEncounter(ctx context.Context, input AdminUpsertWildEncounterInput) (*AdminWildEncounterDetail, error) {
	input = input.Normalize()
	if err := s.validateAdminWildEncounterInput(input); err != nil {
		return nil, err
	}
	if err := s.ValidateEnabledMonsterIDs(ctx, input.SpawnMonsterIDs); err != nil {
		return nil, err
	}
	return s.repo.CreateWildEncounterForAdmin(ctx, input)
}

func (s *Service) UpdateAdminSceneWildEncounter(ctx context.Context, sceneID uint32, input AdminUpsertWildEncounterInput) (*AdminWildEncounterDetail, error) {
	input = input.Normalize()
	input.SceneID = sceneID
	if sceneID == 0 {
		return nil, ErrInvalidAdminSceneWildEncounterInput
	}
	if err := s.validateAdminWildEncounterInput(input); err != nil {
		return nil, err
	}
	if err := s.ValidateEnabledMonsterIDs(ctx, input.SpawnMonsterIDs); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateWildEncounterForAdmin(ctx, sceneID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrSceneWildEncounterNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminSceneWildEncounter(ctx context.Context, sceneID uint32) error {
	return s.repo.DeleteWildEncounterForAdmin(ctx, sceneID)
}

func (s *Service) validateAdminWildEncounterInput(input AdminUpsertWildEncounterInput) error {
	if input.SceneID == 0 || input.EncounterName == "" || len(input.SpawnMonsterIDs) == 0 {
		return ErrInvalidAdminSceneWildEncounterInput
	}
	if input.EncounterRate > 10000 {
		return fmt.Errorf("%w: encounter_rate must be <= 10000", ErrInvalidAdminSceneWildEncounterInput)
	}
	return nil
}

func (s *Service) validateCaptureConfig(ctx context.Context, input AdminUpsertDefinitionInput) error {
	if !input.IsCapturable {
		return nil
	}
	if input.CapturePetID == 0 {
		return fmt.Errorf("%w: capture_pet_id required", ErrInvalidAdminMonsterDefinitionInput)
	}
	if s.capturePetValidator == nil {
		return ErrInvalidCapturePetReference
	}
	if err := s.capturePetValidator.ValidateCapturePetTemplate(ctx, input.CapturePetID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCapturePetReference, err)
	}
	return nil
}
