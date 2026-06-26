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

// ActiveBattleChecker 用于在战斗进行期间跳开会触发读库重算的路径。
type ActiveBattleChecker interface {
	IsPlayerInActiveBattle(playerID uint64) bool
}

// Service 负责装备模板后台 CRUD 与运行时佩戴的领域入口。
type Service struct {
	repo              Repository
	progression       *progression.Service
	players           player.Repository
	combatCapsLoader  CombatCapsLoader
	battleChecker     ActiveBattleChecker
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

// SetBattleChecker 注入战斗状态检查器，用于模板刷新与重算时跳过战斗中的玩家。
func (s *Service) SetBattleChecker(checker ActiveBattleChecker) {
	if s == nil {
		return
	}
	s.battleChecker = checker
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
	if err := s.refreshPlayersEquippedItemTemplate(ctx, itemID); err != nil {
		return nil, err
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

// RecalcPlayerCombatStats 在成长层写库后，把当前已佩戴装备加成重新合并到 player 战斗字段。
// refillHP 为 true 时会把当前 hp 补满到新 hp_max（用于升级后补满，避免成长层先写入裸装上限）。
func (s *Service) RecalcPlayerCombatStats(ctx context.Context, playerID uint64, refillHP bool) error {
	if s.repo == nil || playerID == 0 {
		return nil
	}
	recalc, profile, err := s.buildRecalcContext(ctx, playerID)
	if err != nil {
		return err
	}
	return s.repo.RecalcEquippedCombatStats(ctx, playerID, recalc, profile, refillHP)
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

// refreshPlayersEquippedItemTemplate 在装备模板数值变更后，为所有正佩戴该模板的玩家
// 复用客户端同款卸装/再穿戴链路，确保属性重算与正式玩法完全一致。
func (s *Service) refreshPlayersEquippedItemTemplate(ctx context.Context, itemID uint64) error {
	if s.repo == nil || itemID == 0 {
		return nil
	}
	entries, err := s.repo.ListEquippedEntriesForItemID(ctx, itemID)
	if err != nil {
		return err
	}
	containerType := bag.ContainerTypeBag
	for _, entry := range entries {
		if entry.PlayerID == 0 || strings.TrimSpace(entry.EquipSlot) == "" || strings.TrimSpace(entry.ItemUID) == "" {
			continue
		}
		if s.battleChecker != nil && s.battleChecker.IsPlayerInActiveBattle(entry.PlayerID) {
			continue
		}
		recalc, profile, err := s.buildRecalcContext(ctx, entry.PlayerID)
		if err != nil {
			return err
		}
		if err := s.repo.RefreshEquippedTemplateEntry(
			ctx,
			entry.PlayerID,
			entry.EquipSlot,
			entry.ItemUID,
			containerType,
			recalc,
			profile,
		); err != nil {
			return err
		}
	}
	return nil
}
