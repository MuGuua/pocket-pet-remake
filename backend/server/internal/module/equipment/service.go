package equipment

import (
	"context"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
)

// CombatCapsLoader 加载人物/宠物共用战斗属性封顶表。
type CombatCapsLoader interface {
	LoadCombatStatCaps(ctx context.Context) (pet.CombatStatCaps, error)
}

// Service 负责装备模板后台 CRUD 与运行时佩戴的领域入口。
type Service struct {
	repo              Repository
	progression       *progression.Service
	players           player.Repository
	combatCapsLoader  CombatCapsLoader
}

// NewService 构造装备服务。
func NewService(repo Repository, progressionService *progression.Service, players player.Repository, combatCapsLoader CombatCapsLoader) *Service {
	return &Service{
		repo:             repo,
		progression:      progressionService,
		players:          players,
		combatCapsLoader: combatCapsLoader,
	}
}

// ListAdminEquipmentDefinitions 返回装备模板分页列表。
func (s *Service) ListAdminEquipmentDefinitions(ctx context.Context, query AdminListQuery) (*AdminEquipmentList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminEquipmentList{Items: []AdminEquipmentSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

// GetAdminEquipmentDetail 返回单条装备模板详情。
func (s *Service) GetAdminEquipmentDetail(ctx context.Context, itemID uint64) (*AdminEquipmentDetail, error) {
	result, err := s.repo.FindForAdminByItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrEquipmentDefinitionNotFound
	}
	return result, nil
}

// CreateAdminEquipmentDefinition 创建装备模板。
func (s *Service) CreateAdminEquipmentDefinition(ctx context.Context, input AdminUpsertEquipmentInput) (*AdminEquipmentDetail, error) {
	input = input.Normalize()
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.repo.CreateForAdmin(ctx, input)
}

// UpdateAdminEquipmentDefinition 更新装备模板。
func (s *Service) UpdateAdminEquipmentDefinition(ctx context.Context, itemID uint64, input AdminUpsertEquipmentInput) (*AdminEquipmentDetail, error) {
	input = input.Normalize()
	if itemID == 0 {
		return nil, ErrInvalidAdminEquipmentInput
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateForAdmin(ctx, itemID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrEquipmentDefinitionNotFound
	}
	return result, nil
}

// DeleteAdminEquipmentDefinition 停用装备模板（软删 is_enabled=false）。
func (s *Service) DeleteAdminEquipmentDefinition(ctx context.Context, itemID uint64) error {
	return s.repo.DeleteForAdmin(ctx, itemID)
}

// ListEquipped 返回玩家当前全身已佩戴装备摘要。
func (s *Service) ListEquipped(ctx context.Context, playerID uint64) ([]RuntimeEquippedItem, error) {
	if s.repo == nil || playerID == 0 {
		return nil, ErrEquipmentNotOwned
	}
	return s.repo.ListEquipped(ctx, playerID)
}

// EquipFromBagSlot 从背包指定格子佩戴装备并重算玩家战斗属性。
func (s *Service) EquipFromBagSlot(ctx context.Context, playerID uint64, containerType string, bagSlotIndex uint32) (*EquipFromBagResult, *player.Profile, error) {
	recalc, profile, err := s.buildRecalcContext(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}
	normalizedContainer := strings.TrimSpace(containerType)
	if normalizedContainer == "" {
		normalizedContainer = bag.ContainerTypeBag
	}
	result, err := s.repo.EquipFromBagSlot(ctx, playerID, normalizedContainer, bagSlotIndex, recalc, profile)
	if err != nil {
		return nil, nil, err
	}
	updatedProfile, err := s.players.FindByPlayerID(ctx, playerID)
	if err != nil {
		return result, nil, err
	}
	return result, updatedProfile, nil
}

// UnequipSlot 卸下指定部位装备并重算玩家战斗属性。
func (s *Service) UnequipSlot(ctx context.Context, playerID uint64, equipSlot string, containerType string) (*UnequipSlotResult, *player.Profile, error) {
	if !IsValidEquipSlot(equipSlot) {
		return nil, nil, ErrEquipmentSlotMismatch
	}
	recalc, profile, err := s.buildRecalcContext(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}
	normalizedContainer := strings.TrimSpace(containerType)
	if normalizedContainer == "" {
		normalizedContainer = bag.ContainerTypeBag
	}
	result, err := s.repo.UnequipSlot(ctx, playerID, equipSlot, normalizedContainer, recalc, profile)
	if err != nil {
		return nil, nil, err
	}
	updatedProfile, err := s.players.FindByPlayerID(ctx, playerID)
	if err != nil {
		return result, nil, err
	}
	return result, updatedProfile, nil
}

// EnhanceInstance 消耗材料并尝试强化指定装备实例；仅允许未佩戴且位于背包中的实例。
func (s *Service) EnhanceInstance(ctx context.Context, playerID uint64, itemUID string) (*EnhanceResult, error) {
	if strings.TrimSpace(itemUID) == "" {
		return nil, ErrEquipmentNotFound
	}
	return s.repo.EnhanceInstance(ctx, playerID, itemUID)
}

func (s *Service) buildRecalcContext(ctx context.Context, playerID uint64) (RecalcContext, *player.Profile, error) {
	if s.progression == nil || s.combatCapsLoader == nil {
		return RecalcContext{}, nil, ErrEquipmentNotFound
	}
	profile, err := s.players.FindByPlayerID(ctx, playerID)
	if err != nil {
		return RecalcContext{}, nil, err
	}
	if profile == nil {
		return RecalcContext{}, nil, player.ErrPlayerNotFound
	}
	progressionState, err := s.progression.LoadProgressionState(ctx, playerID)
	if err != nil {
		return RecalcContext{}, nil, err
	}
	if progressionState == nil {
		return RecalcContext{}, nil, progression.ErrLevelConfigNotFound
	}
	caps, err := s.combatCapsLoader.LoadCombatStatCaps(ctx)
	if err != nil {
		return RecalcContext{}, nil, err
	}
	combatBonus := s.progression.CombatBonusForAllocated(progressionState.Allocated)
	return RecalcContext{
		Progression: *progressionState,
		CombatBonus: combatBonus,
		Caps:        caps,
	}, profile, nil
}
