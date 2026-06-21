package pet

import (
	"context"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/petprogression"
	"pocket-pet-remake/server/internal/module/skill"
)

// CaptureConfigReader 供捕捉发放链路读取怪物模板的 capture_pet_id 映射。
type CaptureConfigReader interface {
	FindCaptureConfig(ctx context.Context, monsterID uint32) (*monster.CaptureConfig, error)
}

type Service struct {
	repo                Repository
	skillService        *skill.Service
	captureConfigReader CaptureConfigReader
	progressionService  *petprogression.Service
}

func NewService(repo Repository, skillService *skill.Service, captureConfigReader CaptureConfigReader, progressionService *petprogression.Service) *Service {
	return &Service{
		repo:                repo,
		skillService:        skillService,
		captureConfigReader: captureConfigReader,
		progressionService:  progressionService,
	}
}

func (s *Service) ListPets(ctx context.Context, playerID uint64) ([]Pet, error) {
	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if pets == nil {
		pets = []Pet{}
	}
	if err := s.applyUsableFlags(ctx, pets); err != nil {
		return nil, err
	}
	s.enrichProgressionFields(pets)

	lineup, err := s.ListLineup(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if len(lineup) == 0 {
		return pets, nil
	}

	lineupSet := make(map[uint64]struct{}, len(lineup))
	for _, lineupPet := range lineup {
		lineupSet[lineupPet.PetUID] = struct{}{}
	}
	for index := range pets {
		_, inLineup := lineupSet[pets[index].PetUID]
		pets[index].InLineup = inLineup
	}
	return pets, nil
}

func (s *Service) ListLineup(ctx context.Context, playerID uint64) ([]LineupPet, error) {
	lineup, err := s.repo.ListLineupByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if lineup == nil {
		return []LineupPet{}, nil
	}
	return lineup, nil
}

// SetAdminPetLineup 复用游戏内编队校验，供运营后台调整玩家出战宠物。
func (s *Service) SetAdminPetLineup(ctx context.Context, playerID uint64, input AdminSetPetLineupInput) (*AdminSetPetLineupResult, error) {
	if playerID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	lineup, err := s.SetLineup(ctx, playerID, input.PetUIDs)
	if err != nil {
		return nil, err
	}
	resultPetUIDs := make([]uint64, 0, len(lineup))
	for _, item := range lineup {
		resultPetUIDs = append(resultPetUIDs, item.PetUID)
	}
	return &AdminSetPetLineupResult{
		PlayerID: playerID,
		PetUIDs:  resultPetUIDs,
	}, nil
}

func (s *Service) SetLineup(ctx context.Context, playerID uint64, petUIDs []uint64) ([]LineupPet, error) {
	// 当前玩法允许空编队，且同时最多只能出战 1 只宠物。
	if len(petUIDs) > 1 {
		return nil, ErrInvalidLineup
	}
	if len(petUIDs) == 0 {
		if err := s.repo.SetLineupByPlayerID(ctx, playerID, nil); err != nil {
			return nil, err
		}
		return []LineupPet{}, nil
	}

	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if len(pets) == 0 {
		return nil, ErrPetNotFound
	}
	if err := s.applyUsableFlags(ctx, pets); err != nil {
		return nil, err
	}

	owned := make(map[uint64]Pet, len(pets))
	for _, item := range pets {
		owned[item.PetUID] = item
	}

	seen := make(map[uint64]struct{}, len(petUIDs))
	for _, petUID := range petUIDs {
		if petUID == 0 {
			return nil, ErrInvalidLineup
		}
		if _, exists := seen[petUID]; exists {
			return nil, ErrDuplicateLineup
		}
		item, exists := owned[petUID]
		if !exists {
			return nil, ErrPetNotFound
		}
		if !item.IsUsable {
			return nil, ErrPetUnusable
		}
		seen[petUID] = struct{}{}
	}

	if err := s.repo.SetLineupByPlayerID(ctx, playerID, petUIDs); err != nil {
		return nil, err
	}
	return s.ListLineup(ctx, playerID)
}

func (s *Service) UpdatePetHP(ctx context.Context, playerID uint64, petUID uint64, hp uint32) (Pet, error) {
	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return Pet{}, err
	}

	var target *Pet
	for index := range pets {
		if pets[index].PetUID == petUID {
			target = &pets[index]
			break
		}
	}
	if target == nil {
		return Pet{}, ErrPetNotFound
	}

	if hp > target.HPMax {
		hp = target.HPMax
	}
	return s.repo.UpdatePetHPByUID(ctx, playerID, petUID, hp)
}

func (s *Service) UpdatePetBattleProgress(ctx context.Context, playerID uint64, petUID uint64, hp uint32, expGain uint64) (Pet, error) {
	if err := s.ensurePetOwned(ctx, playerID, petUID); err != nil {
		return Pet{}, err
	}
	if s.progressionService != nil && expGain > 0 {
		result, err := s.progressionService.ApplyExp(ctx, playerID, petUID, expGain, hp)
		if err != nil {
			return Pet{}, err
		}
		updated, err := s.loadPetByUID(ctx, playerID, petUID)
		if err != nil {
			return Pet{}, err
		}
		updated.LastLevelUpCount = result.LevelUpCount
		updated.LastAttrPointsGained = result.AttrPointsGained
		return updated, nil
	}
	if hp == 0 {
		return s.loadPetByUID(ctx, playerID, petUID)
	}
	updated, err := s.repo.UpdatePetHPByUID(ctx, playerID, petUID, hp)
	if err != nil {
		return Pet{}, err
	}
	s.enrichProgressionFields([]Pet{updated})
	return updated, nil
}

// AllocateAttrPoints 为单只宠物分配自由属性点并重算战斗属性。
func (s *Service) AllocateAttrPoints(ctx context.Context, playerID uint64, petUID uint64, delta petprogression.ManualAllocatedPoints) (Pet, error) {
	if s.progressionService == nil {
		return Pet{}, petprogression.ErrInvalidAllocateInput
	}
	if err := s.ensurePetOwned(ctx, playerID, petUID); err != nil {
		return Pet{}, err
	}
	if _, err := s.progressionService.AllocateAttrPoints(ctx, playerID, petUID, delta); err != nil {
		return Pet{}, err
	}
	return s.loadPetByUID(ctx, playerID, petUID)
}

func (s *Service) ensurePetOwned(ctx context.Context, playerID uint64, petUID uint64) error {
	pets, err := s.repo.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return err
	}
	for index := range pets {
		if pets[index].PetUID == petUID {
			return nil
		}
	}
	return ErrPetNotFound
}

func (s *Service) loadPetByUID(ctx context.Context, playerID uint64, petUID uint64) (Pet, error) {
	item, err := s.repo.FindPetByUID(ctx, playerID, petUID)
	if err != nil {
		return Pet{}, err
	}
	if err := s.applyUsableFlags(ctx, []Pet{item}); err != nil {
		return Pet{}, err
	}
	s.enrichProgressionFields([]Pet{item})
	lineup, err := s.ListLineup(ctx, playerID)
	if err != nil {
		return Pet{}, err
	}
	for _, lineupPet := range lineup {
		if lineupPet.PetUID == petUID {
			item.InLineup = true
			break
		}
	}
	return item, nil
}

func (s *Service) enrichProgressionFields(pets []Pet) {
	if s.progressionService == nil {
		return
	}
	for index := range pets {
		pets[index].ExpToNext = s.progressionService.ExpToNext(pets[index].Level, pets[index].Exp)
	}
}

// GrantRuntimePet 按服务端数据库里的宠物模板发放一只新宠物给玩家。
// 奖励宠物默认不会自动进出战阵容，客户端收到推送后再决定是否引导玩家配置。
func (s *Service) GrantRuntimePet(ctx context.Context, playerID uint64, petID uint32, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*RuntimeGrantResult, error) {
	if playerID == 0 || petID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	return s.repo.GrantRuntimePet(ctx, playerID, petID, reasonType, reasonRefID, operatorType, operatorID)
}

// GrantWildCapturePet 按野外捕捉模板发放宠物，并在服务端 roll 五项成长资质。
func (s *Service) GrantWildCapturePet(ctx context.Context, playerID uint64, petID uint32, captureMonsterID uint32, reasonType string, reasonRefID uint64) (*RuntimeGrantResult, error) {
	if playerID == 0 || petID == 0 || captureMonsterID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	return s.repo.GrantWildCapturePet(ctx, playerID, petID, captureMonsterID, reasonType, reasonRefID)
}

// GrantCapturedPet 在战斗捕捉成功后，按怪物 capture_pet_id 发放关联系统宠物。
// 不继承战斗中的 HP/等级等数值，仅使用模板基础属性 + roll 资质。
func (s *Service) GrantCapturedPet(ctx context.Context, playerID uint64, monsterID uint32, reasonType string, reasonRefID uint64) (*RuntimeGrantResult, error) {
	if playerID == 0 || monsterID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	if s.captureConfigReader == nil {
		return nil, monster.ErrMonsterNotCapturable
	}
	config, err := s.captureConfigReader.FindCaptureConfig(ctx, monsterID)
	if err != nil {
		return nil, err
	}
	if config == nil || !config.IsCapturable || config.CapturePetID == 0 {
		return nil, monster.ErrMonsterNotCapturable
	}
	return s.repo.GrantWildCapturePet(ctx, playerID, config.CapturePetID, monsterID, reasonType, reasonRefID)
}

// ValidateCapturePetTemplate 校验怪物关联的捕捉宠物模板是否为启用的野外捕捉类。
func (s *Service) ValidateCapturePetTemplate(ctx context.Context, petID uint32) error {
	if petID == 0 {
		return ErrInvalidWildCapturePetTemplate
	}
	usableMap, err := s.repo.MapUsablePetDefinitionIDs(ctx, []uint32{petID})
	if err != nil {
		return err
	}
	if !usableMap[petID] {
		return ErrPetUnusable
	}
	definition, err := s.repo.FindPetDefinitionForAdmin(ctx, petID)
	if err != nil {
		return err
	}
	if definition == nil {
		return ErrPetDefinitionNotFound
	}
	if !IsWildCaptureAcquireMethod(definition.AcquireMethod) {
		return ErrInvalidWildCapturePetTemplate
	}
	return ValidateAptitudeRollRanges(adminRollRangesFromDefinition(*definition))
}

func (s *Service) ListAdminPets(ctx context.Context, query AdminListQuery) (*AdminPetList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminPetList{Items: []AdminPetSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminPetDetail(ctx context.Context, petUID uint64) (*AdminPetDetail, error) {
	result, err := s.repo.FindAdminDetailByPetUID(ctx, petUID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminPet(ctx context.Context, input AdminCreatePetInput) (*AdminPetDetail, error) {
	caps, err := s.repo.LoadCombatStatCaps(ctx)
	if err != nil {
		return nil, err
	}
	input = input.Normalize().applyCreateAdminCombatCaps(caps)
	if input.PlayerID == 0 || input.PetID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	usableMap, err := s.repo.MapUsablePetDefinitionIDs(ctx, []uint32{input.PetID})
	if err != nil {
		return nil, err
	}
	if !usableMap[input.PetID] {
		return nil, ErrPetUnusable
	}
	if err := s.validateSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	return s.repo.CreateForAdmin(ctx, input)
}

func (s *Service) UpdateAdminPet(ctx context.Context, petUID uint64, input AdminUpdatePetInput) (*AdminPetDetail, error) {
	caps, err := s.repo.LoadCombatStatCaps(ctx)
	if err != nil {
		return nil, err
	}
	input = input.Normalize().applyUpdateAdminCombatCaps(caps)
	if input.PetID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	if err := s.validateSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateForAdmin(ctx, petUID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminPet(ctx context.Context, petUID uint64) error {
	return s.repo.DeleteForAdmin(ctx, petUID)
}

func (s *Service) ListAdminPetDefinitions(ctx context.Context, query AdminPetDefinitionListQuery) (*AdminPetDefinitionList, error) {
	result, err := s.repo.ListPetDefinitionsForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminPetDefinitionList{Items: []AdminPetDefinitionSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return result, nil
}

func (s *Service) GetAdminPetDefinitionDetail(ctx context.Context, petID uint32) (*AdminPetDefinitionDetail, error) {
	result, err := s.repo.FindPetDefinitionForAdmin(ctx, petID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetDefinitionNotFound
	}
	return result, nil
}

func (s *Service) CreateAdminPetDefinition(ctx context.Context, input AdminUpsertPetDefinitionInput) (*AdminPetDefinitionDetail, error) {
	input = input.Normalize()
	if input.PetID == 0 || input.PetName == "" {
		return nil, ErrInvalidAdminPetDefinitionInput
	}
	if err := s.validateAdminPetDefinitionInput(input); err != nil {
		return nil, err
	}
	if err := s.validateSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	return s.repo.CreatePetDefinitionForAdmin(ctx, input)
}

func (s *Service) UpdateAdminPetDefinition(ctx context.Context, petID uint32, input AdminUpsertPetDefinitionInput) (*AdminPetDefinitionDetail, error) {
	input = input.Normalize()
	if petID == 0 || input.PetName == "" {
		return nil, ErrInvalidAdminPetDefinitionInput
	}
	if err := s.validateAdminPetDefinitionInput(input); err != nil {
		return nil, err
	}
	if err := s.validateSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdatePetDefinitionForAdmin(ctx, petID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetDefinitionNotFound
	}
	return result, nil
}

func (s *Service) DeleteAdminPetDefinition(ctx context.Context, petID uint32) error {
	return s.repo.DeletePetDefinitionForAdmin(ctx, petID)
}

// ResolveSkinID 按宠物模板 ID 返回战斗外观资源 ID，供战斗快照推送使用。
func (s *Service) ResolveSkinID(petID uint32) string {
	if petID == 0 {
		return ""
	}
	skinID, err := s.repo.FindPetSkinID(context.Background(), petID)
	if err != nil {
		return ""
	}
	return skinID
}

func (s *Service) applyUsableFlags(ctx context.Context, pets []Pet) error {
	if len(pets) == 0 {
		return nil
	}
	petIDs := make([]uint32, 0, len(pets))
	for _, item := range pets {
		petIDs = append(petIDs, item.PetID)
	}
	usableMap, err := s.repo.MapUsablePetDefinitionIDs(ctx, petIDs)
	if err != nil {
		return err
	}
	for index := range pets {
		pets[index].IsUsable = usableMap[pets[index].PetID]
	}
	return nil
}

func (s *Service) validateSkillIDs(ctx context.Context, skillIDs []uint32) error {
	if s.skillService == nil || len(skillIDs) == 0 {
		return nil
	}
	return s.skillService.ValidateEnabledSkillIDs(ctx, skillIDs)
}

func (s *Service) validateAdminPetDefinitionInput(input AdminUpsertPetDefinitionInput) error {
	if input.IsEnabled && strings.TrimSpace(input.SkinID) == "" {
		return fmt.Errorf("%w: skin_id required", ErrInvalidAdminPetDefinitionInput)
	}
	if !IsWildCaptureAcquireMethod(input.AcquireMethod) {
		return nil
	}
	return ValidateAptitudeRollRanges(AptitudeRollRanges{
		HPAptMin: input.HPAptRollMin, HPAptMax: input.HPAptRollMax,
		ATKAptMin: input.ATKAptRollMin, ATKAptMax: input.ATKAptRollMax,
		DEFAptMin: input.DEFAptRollMin, DEFAptMax: input.DEFAptRollMax,
		SPDAptMin: input.SPDAptRollMin, SPDAptMax: input.SPDAptRollMax,
		MANAAptMin: input.MANAAptRollMin, MANAAptMax: input.MANAAptRollMax,
	})
}

func adminRollRangesFromDefinition(definition AdminPetDefinitionDetail) AptitudeRollRanges {
	return AptitudeRollRanges{
		HPAptMin: definition.AptitudeRollRanges.HPAptRollMin, HPAptMax: definition.AptitudeRollRanges.HPAptRollMax,
		ATKAptMin: definition.AptitudeRollRanges.ATKAptRollMin, ATKAptMax: definition.AptitudeRollRanges.ATKAptRollMax,
		DEFAptMin: definition.AptitudeRollRanges.DEFAptRollMin, DEFAptMax: definition.AptitudeRollRanges.DEFAptRollMax,
		SPDAptMin: definition.AptitudeRollRanges.SPDAptRollMin, SPDAptMax: definition.AptitudeRollRanges.SPDAptRollMax,
		MANAAptMin: definition.AptitudeRollRanges.MANAAptRollMin, MANAAptMax: definition.AptitudeRollRanges.MANAAptRollMax,
	}
}

// EquipArtifactFromBagSlot 从背包装备法宝并刷新宠物战斗技能列表。
func (s *Service) EquipArtifactFromBagSlot(ctx context.Context, playerID uint64, petUID uint64, slotIndex uint32, containerType string, bagSlotIndex uint32) (Pet, error) {
	if playerID == 0 || petUID == 0 {
		return Pet{}, ErrPetNotFound
	}
	normalizedContainerType := strings.TrimSpace(containerType)
	if normalizedContainerType == "" {
		normalizedContainerType = "bag"
	}
	updatedPet, err := s.repo.EquipArtifactFromBagSlot(ctx, playerID, petUID, slotIndex, normalizedContainerType, bagSlotIndex)
	if err != nil {
		return Pet{}, err
	}
	return s.loadPetByUID(ctx, playerID, updatedPet.PetUID)
}

// UnequipArtifact 卸下宠物法宝槽并刷新战斗技能列表。
func (s *Service) UnequipArtifact(ctx context.Context, playerID uint64, petUID uint64, slotIndex uint32) (Pet, error) {
	if playerID == 0 || petUID == 0 {
		return Pet{}, ErrPetNotFound
	}
	updatedPet, err := s.repo.UnequipArtifact(ctx, playerID, petUID, slotIndex)
	if err != nil {
		return Pet{}, err
	}
	return s.loadPetByUID(ctx, playerID, updatedPet.PetUID)
}

// GetPetDetail 返回单只宠物完整快照，供技能详情等需要法宝技分槽的界面使用。
func (s *Service) GetPetDetail(ctx context.Context, playerID uint64, petUID uint64) (Pet, error) {
	return s.loadPetByUID(ctx, playerID, petUID)
}

func (s *Service) ListAdminPetSkillSlotUnlockItems(ctx context.Context) ([]AdminPetSkillSlotUnlockItem, error) {
	items, err := s.repo.ListAdminPetSkillSlotUnlockItems(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []AdminPetSkillSlotUnlockItem{}, nil
	}
	return items, nil
}

func (s *Service) GetAdminPetSkillSlotUnlockItem(ctx context.Context, slotKey string) (*AdminPetSkillSlotUnlockItem, error) {
	return s.repo.FindAdminPetSkillSlotUnlockItem(ctx, slotKey)
}

func (s *Service) CreateAdminPetSkillSlotUnlockItem(ctx context.Context, input AdminUpsertPetSkillSlotUnlockInput) (*AdminPetSkillSlotUnlockItem, error) {
	normalized, err := input.Normalize()
	if err != nil {
		return nil, err
	}
	return s.repo.CreateAdminPetSkillSlotUnlockItem(ctx, normalized)
}

func (s *Service) UpdateAdminPetSkillSlotUnlockItem(ctx context.Context, slotKey string, input AdminUpsertPetSkillSlotUnlockInput) (*AdminPetSkillSlotUnlockItem, error) {
	normalized, err := input.Normalize()
	if err != nil {
		return nil, err
	}
	normalized.SlotKey = slotKey
	return s.repo.UpdateAdminPetSkillSlotUnlockItem(ctx, slotKey, normalized)
}

func (s *Service) DeleteAdminPetSkillSlotUnlockItem(ctx context.Context, slotKey string) error {
	return s.repo.DeleteAdminPetSkillSlotUnlockItem(ctx, slotKey)
}

func (s *Service) ListAdminPetCombatStatCaps(ctx context.Context) ([]AdminPetCombatStatCap, error) {
	items, err := s.repo.ListAdminPetCombatStatCaps(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []AdminPetCombatStatCap{}, nil
	}
	return items, nil
}

func (s *Service) GetAdminPetCombatStatCap(ctx context.Context, statKey string) (*AdminPetCombatStatCap, error) {
	result, err := s.repo.FindAdminPetCombatStatCap(ctx, statKey)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetCombatStatCapNotFound
	}
	return result, nil
}

func (s *Service) UpdateAdminPetCombatStatCap(ctx context.Context, statKey string, input AdminUpsertPetCombatStatCapInput) (*AdminPetCombatStatCap, error) {
	input = input.Normalize()
	if err := validateAdminPetCombatStatCapInput(statKey, input); err != nil {
		return nil, err
	}
	return s.repo.UpdateAdminPetCombatStatCap(ctx, statKey, input)
}
