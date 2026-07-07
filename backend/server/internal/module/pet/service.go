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

// CombatSnapshotRepository 提供宠物最终属性快照的刷新与读取能力。
// 只有正式 PostgreSQL 仓储实现它；测试桩仍走原始表 + 运行时折算兜底。
type CombatSnapshotRepository interface {
	RefreshPlayerPetCombatSnapshots(ctx context.Context, playerID uint64) error
	ListPlayerPetCombatSnapshotsByPlayerID(ctx context.Context, playerID uint64) ([]Pet, error)
	ListPlayerLineupCombatSnapshotsByPlayerID(ctx context.Context, playerID uint64) ([]LineupPet, error)
	FindPlayerPetCombatSnapshotByUID(ctx context.Context, playerID uint64, petUID uint64) (*Pet, error)
}

// SummaryRepository 提供宠物面板列表需要的轻量摘要读取能力。
// 摘要接口不刷新战斗快照，也不加载技能、法宝和完整数值，避免打开面板时一次性计算所有宠物。
type SummaryRepository interface {
	ListPetSummariesByPlayerID(ctx context.Context, playerID uint64) ([]Pet, error)
	ListLineupSummariesByPlayerID(ctx context.Context, playerID uint64) ([]LineupPet, error)
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

// ListPetSummaries 返回宠物状态面板左侧列表使用的轻量宠物摘要。
// 完整属性、资质、技能和法宝数据应通过 GetPetDetail 按单只宠物拉取。
func (s *Service) ListPetSummaries(ctx context.Context, playerID uint64) ([]Pet, error) {
	if summaryRepo, ok := s.repo.(SummaryRepository); ok {
		pets, err := summaryRepo.ListPetSummariesByPlayerID(ctx, playerID)
		if err != nil {
			return nil, err
		}
		if pets == nil {
			pets = []Pet{}
		}
		if err := s.applyUsableFlags(ctx, pets); err != nil {
			return nil, err
		}
		return pets, nil
	}
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
	lineup, err := s.repo.ListLineupByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	lineupSet := make(map[uint64]struct{}, len(lineup))
	for _, lineupPet := range lineup {
		lineupSet[lineupPet.PetUID] = struct{}{}
	}
	for index := range pets {
		_, pets[index].InLineup = lineupSet[pets[index].PetUID]
	}
	return pets, nil
}

// ListLineupSummaries 返回宠物状态面板需要的轻量编队摘要。
// 它刻意绕过 ListLineup 的战斗快照刷新，避免列表请求触发全量宠物数值重算。
func (s *Service) ListLineupSummaries(ctx context.Context, playerID uint64) ([]LineupPet, error) {
	if summaryRepo, ok := s.repo.(SummaryRepository); ok {
		lineup, err := summaryRepo.ListLineupSummariesByPlayerID(ctx, playerID)
		if err != nil {
			return nil, err
		}
		if lineup == nil {
			return []LineupPet{}, nil
		}
		return lineup, nil
	}
	lineup, err := s.repo.ListLineupByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if lineup == nil {
		return []LineupPet{}, nil
	}
	return lineup, nil
}

func (s *Service) ListPets(ctx context.Context, playerID uint64) ([]Pet, error) {
	if snapshotRepo, ok := s.repo.(CombatSnapshotRepository); ok {
		if err := snapshotRepo.RefreshPlayerPetCombatSnapshots(ctx, playerID); err != nil {
			return nil, err
		}
		pets, err := snapshotRepo.ListPlayerPetCombatSnapshotsByPlayerID(ctx, playerID)
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
		applyPersistentPassiveBonusesToPets(pets, s.resolveRuntimeSkillDefinition)
		lineup, err := s.ListLineup(ctx, playerID)
		if err != nil {
			return nil, err
		}
		lineupSet := make(map[uint64]struct{}, len(lineup))
		for _, lineupPet := range lineup {
			lineupSet[lineupPet.PetUID] = struct{}{}
		}
		for index := range pets {
			_, pets[index].InLineup = lineupSet[pets[index].PetUID]
		}
		return pets, nil
	}
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
	applyPersistentPassiveBonusesToPets(pets, s.resolveRuntimeSkillDefinition)

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
	if snapshotRepo, ok := s.repo.(CombatSnapshotRepository); ok {
		if err := snapshotRepo.RefreshPlayerPetCombatSnapshots(ctx, playerID); err != nil {
			return nil, err
		}
		lineup, err := snapshotRepo.ListPlayerLineupCombatSnapshotsByPlayerID(ctx, playerID)
		if err != nil {
			return nil, err
		}
		if lineup == nil {
			return []LineupPet{}, nil
		}
		applyPersistentPassiveBonusesToLineupPets(lineup, s.resolveRuntimeSkillDefinition)
		return lineup, nil
	}
	lineup, err := s.repo.ListLineupByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if lineup == nil {
		return []LineupPet{}, nil
	}
	applyPersistentPassiveBonusesToLineupPets(lineup, s.resolveRuntimeSkillDefinition)
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
	if _, err := s.repo.UpdatePetHPByUID(ctx, playerID, petUID, hp); err != nil {
		return Pet{}, err
	}
	return s.loadPetByUID(ctx, playerID, petUID)
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
	if _, err := s.repo.UpdatePetHPByUID(ctx, playerID, petUID, hp); err != nil {
		return Pet{}, err
	}
	return s.loadPetByUID(ctx, playerID, petUID)
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
	if snapshotRepo, ok := s.repo.(CombatSnapshotRepository); ok {
		if err := snapshotRepo.RefreshPlayerPetCombatSnapshots(ctx, playerID); err != nil {
			return Pet{}, err
		}
		item, err := snapshotRepo.FindPlayerPetCombatSnapshotByUID(ctx, playerID, petUID)
		if err != nil {
			return Pet{}, err
		}
		if item == nil {
			return Pet{}, ErrPetNotFound
		}
		items := []Pet{*item}
		if err := s.applyUsableFlags(ctx, items); err != nil {
			return Pet{}, err
		}
		s.enrichProgressionFields(items)
		applyPersistentPassiveBonusesToPets(items, s.resolveRuntimeSkillDefinition)
		lineup, err := s.ListLineup(ctx, playerID)
		if err != nil {
			return Pet{}, err
		}
		for _, lineupPet := range lineup {
			if lineupPet.PetUID == petUID {
				items[0].InLineup = true
				break
			}
		}
		return items[0], nil
	}
	item, err := s.repo.FindPetByUID(ctx, playerID, petUID)
	if err != nil {
		return Pet{}, err
	}
	items := []Pet{item}
	if err := s.applyUsableFlags(ctx, items); err != nil {
		return Pet{}, err
	}
	s.enrichProgressionFields(items)
	applyPersistentPassiveBonusesToPets(items, s.resolveRuntimeSkillDefinition)
	item = items[0]
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
	result, err := s.repo.GrantRuntimePet(ctx, playerID, petID, reasonType, reasonRefID, operatorType, operatorID)
	if err != nil {
		return nil, err
	}
	if result != nil {
		applyPersistentPassiveBonusesToPet(&result.Pet, s.resolveRuntimeSkillDefinition)
	}
	return result, nil
}

// GrantWildCapturePet 按野外捕捉模板发放宠物，并在服务端 roll 五项成长资质。
func (s *Service) GrantWildCapturePet(ctx context.Context, playerID uint64, petID uint32, captureMonsterID uint32, reasonType string, reasonRefID uint64) (*RuntimeGrantResult, error) {
	if playerID == 0 || petID == 0 || captureMonsterID == 0 {
		return nil, ErrInvalidAdminPetInput
	}
	result, err := s.repo.GrantWildCapturePet(ctx, playerID, petID, captureMonsterID, reasonType, reasonRefID)
	if err != nil {
		return nil, err
	}
	if result != nil {
		applyPersistentPassiveBonusesToPet(&result.Pet, s.resolveRuntimeSkillDefinition)
	}
	return result, nil
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
	s.applyCombatSnapshotsToAdminSummaries(ctx, result.Items)
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
	s.applyCombatSnapshotToAdminDetail(ctx, result)
	return result, nil
}

// GrantAdminPetFromTemplate 按启用的系统宠物模板发放一只初始宠物实例（资质/技能/等级由服务端模板链路计算）。
func (s *Service) GrantAdminPetFromTemplate(ctx context.Context, input AdminGrantPetFromTemplateInput) (*AdminPetDetail, error) {
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
	granted, err := s.repo.GrantRuntimePet(ctx, input.PlayerID, input.PetID, "admin_grant", 0, "admin", 0)
	if err != nil {
		return nil, err
	}
	if granted == nil {
		return nil, ErrPetNotFound
	}
	return s.GetAdminPetDetail(ctx, granted.Pet.PetUID)
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
	input.SkillIDs = BuildAdminBattleSkillIDs(input.InnateSkillIDs, input.NormalSkillIDs, input.SkillIDs)
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
	existingRawDetail, err := s.repo.FindAdminDetailByPetUID(ctx, petUID)
	if err != nil {
		return nil, err
	}
	if existingRawDetail == nil {
		return nil, ErrPetNotFound
	}
	input.SkillIDs = BuildAdminBattleSkillIDs(input.InnateSkillIDs, input.NormalSkillIDs, input.SkillIDs)
	if err := s.validateSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	reconcileDisplayedAdminStatsToRaw(&input, existingRawDetail, s.applyPersistentPassiveBonusesToAdminDetail)
	result, err := s.repo.UpdateForAdmin(ctx, petUID, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPetNotFound
	}
	s.applyPersistentPassiveBonusesToAdminDetail(result)
	return result, nil
}

// reconcileDisplayedAdminStatsToRaw 把后台详情页里“仅用于展示的被动加成结果”还原回可持久化的基础值。
// 这样当运营只删除技能、不手改生命/攻击等字段时，保存后不会把加成后的展示值误写回 player_pet。
func reconcileDisplayedAdminStatsToRaw(
	input *AdminUpdatePetInput,
	rawDetail *AdminPetDetail,
	applyDisplayed func(*AdminPetDetail),
) {
	if input == nil || rawDetail == nil || applyDisplayed == nil {
		return
	}
	displayedDetail := *rawDetail
	applyDisplayed(&displayedDetail)

	if input.HP == displayedDetail.HP {
		input.HP = rawDetail.HP
	}
	if input.HPMax == displayedDetail.HPMax {
		input.HPMax = rawDetail.HPMax
	}
	if input.ATK == displayedDetail.ATK {
		input.ATK = rawDetail.ATK
	}
	if input.SPD == displayedDetail.SPD {
		input.SPD = rawDetail.SPD
	}
	if input.MANA == displayedDetail.MANA {
		input.MANA = rawDetail.MANA
	}
	if input.CritRatePct == displayedDetail.CritRatePct {
		input.CritRatePct = rawDetail.CritRatePct
	}
	if input.CritDmgPct == displayedDetail.CritDmgPct {
		input.CritDmgPct = rawDetail.CritDmgPct
	}
	if input.PhysicalResistPct == displayedDetail.PhysicalResistPct {
		input.PhysicalResistPct = rawDetail.PhysicalResistPct
	}
	if input.SkillResistPct == displayedDetail.SkillResistPct {
		input.SkillResistPct = rawDetail.SkillResistPct
	}
	if input.ConfusionResistPct == displayedDetail.ConfusionResistPct {
		input.ConfusionResistPct = rawDetail.ConfusionResistPct
	}
	if input.SleepResistPct == displayedDetail.SleepResistPct {
		input.SleepResistPct = rawDetail.SleepResistPct
	}
	if input.ParalysisResistPct == displayedDetail.ParalysisResistPct {
		input.ParalysisResistPct = rawDetail.ParalysisResistPct
	}
	if input.SealResistPct == displayedDetail.SealResistPct {
		input.SealResistPct = rawDetail.SealResistPct
	}
	if input.CurseResistPct == displayedDetail.CurseResistPct {
		input.CurseResistPct = rawDetail.CurseResistPct
	}
}

func (s *Service) DeleteAdminPet(ctx context.Context, petUID uint64) error {
	return s.repo.DeleteForAdmin(ctx, petUID)
}

// applyPersistentPassiveBonusesToAdminSummaries 让后台宠物列表展示的基础属性口径与玩家面板保持一致。
func (s *Service) applyPersistentPassiveBonusesToAdminSummaries(items []AdminPetSummary) {
	for index := range items {
		runtimePet := Pet{
			HP:       items[index].HP,
			HPMax:    items[index].HPMax,
			ATK:      items[index].ATK,
			DEF:      items[index].DEF,
			SPD:      items[index].SPD,
			MANA:     items[index].MANA,
			SkillIDs: append([]uint32{}, items[index].SkillIDs...),
		}
		if !applyPersistentPassiveBonusesToPet(&runtimePet, s.resolveRuntimeSkillDefinition) {
			continue
		}
		items[index].HP = runtimePet.HP
		items[index].HPMax = runtimePet.HPMax
		items[index].ATK = runtimePet.ATK
		items[index].SPD = runtimePet.SPD
		items[index].MANA = runtimePet.MANA
	}
}

// applyPersistentPassiveBonusesToAdminDetail 让后台宠物详情中的基础/次要战斗属性同步展示被动加成后的结果。
func (s *Service) applyPersistentPassiveBonusesToAdminDetail(detail *AdminPetDetail) {
	if detail == nil {
		return
	}
	runtimePet := Pet{
		HP:                 detail.HP,
		HPMax:              detail.HPMax,
		ATK:                detail.ATK,
		DEF:                detail.DEF,
		SPD:                detail.SPD,
		MANA:               detail.MANA,
		SkillIDs:           append([]uint32{}, detail.SkillIDs...),
		CritRatePct:        detail.CritRatePct,
		CritDmgPct:         detail.CritDmgPct,
		PhysicalResistPct:  detail.PhysicalResistPct,
		SkillResistPct:     detail.SkillResistPct,
		ConfusionResistPct: detail.ConfusionResistPct,
		SleepResistPct:     detail.SleepResistPct,
		ParalysisResistPct: detail.ParalysisResistPct,
		SealResistPct:      detail.SealResistPct,
		CurseResistPct:     detail.CurseResistPct,
	}
	if !applyPersistentPassiveBonusesToPet(&runtimePet, s.resolveRuntimeSkillDefinition) {
		return
	}
	detail.HP = runtimePet.HP
	detail.HPMax = runtimePet.HPMax
	detail.ATK = runtimePet.ATK
	detail.SPD = runtimePet.SPD
	detail.MANA = runtimePet.MANA
	detail.CritRatePct = runtimePet.CritRatePct
	detail.CritDmgPct = runtimePet.CritDmgPct
	detail.PhysicalResistPct = runtimePet.PhysicalResistPct
	detail.SkillResistPct = runtimePet.SkillResistPct
	detail.ConfusionResistPct = runtimePet.ConfusionResistPct
	detail.SleepResistPct = runtimePet.SleepResistPct
	detail.ParalysisResistPct = runtimePet.ParalysisResistPct
	detail.SealResistPct = runtimePet.SealResistPct
	detail.CurseResistPct = runtimePet.CurseResistPct
}

func (s *Service) applyCombatSnapshotsToAdminSummaries(ctx context.Context, items []AdminPetSummary) {
	snapshotRepo, ok := s.repo.(CombatSnapshotRepository)
	if !ok {
		s.applyPersistentPassiveBonusesToAdminSummaries(items)
		return
	}
	playerIDs := make(map[uint64]struct{}, len(items))
	for _, item := range items {
		playerIDs[item.PlayerID] = struct{}{}
	}
	for playerID := range playerIDs {
		_ = snapshotRepo.RefreshPlayerPetCombatSnapshots(ctx, playerID)
	}
	for index := range items {
		snapshot, err := snapshotRepo.FindPlayerPetCombatSnapshotByUID(ctx, items[index].PlayerID, items[index].PetUID)
		if err != nil || snapshot == nil {
			continue
		}
		runtimePet := *snapshot
		applyPersistentPassiveBonusesToPet(&runtimePet, s.resolveRuntimeSkillDefinition)
		items[index].HP = runtimePet.HP
		items[index].HPMax = runtimePet.HPMax
		items[index].ATK = runtimePet.ATK
		items[index].DEF = runtimePet.DEF
		items[index].SPD = runtimePet.SPD
		items[index].MANA = runtimePet.MANA
		items[index].SkillIDs = append([]uint32{}, runtimePet.SkillIDs...)
	}
}

func (s *Service) applyCombatSnapshotToAdminDetail(ctx context.Context, detail *AdminPetDetail) {
	if detail == nil {
		return
	}
	snapshotRepo, ok := s.repo.(CombatSnapshotRepository)
	if !ok {
		s.applyPersistentPassiveBonusesToAdminDetail(detail)
		return
	}
	if err := snapshotRepo.RefreshPlayerPetCombatSnapshots(ctx, detail.PlayerID); err != nil {
		return
	}
	snapshot, err := snapshotRepo.FindPlayerPetCombatSnapshotByUID(ctx, detail.PlayerID, detail.PetUID)
	if err != nil || snapshot == nil {
		return
	}
	runtimePet := *snapshot
	applyPersistentPassiveBonusesToPet(&runtimePet, s.resolveRuntimeSkillDefinition)
	detail.HP = runtimePet.HP
	detail.HPMax = runtimePet.HPMax
	detail.ATK = runtimePet.ATK
	detail.DEF = runtimePet.DEF
	detail.SPD = runtimePet.SPD
	detail.MANA = runtimePet.MANA
	detail.SkillIDs = append([]uint32{}, runtimePet.SkillIDs...)
	detail.InnateSkillIDs = append([]uint32{}, runtimePet.SkillLoadout.InnateSkillIDs...)
	detail.NormalSkillIDs = append([]uint32{}, runtimePet.SkillLoadout.NormalSkillIDs...)
	detail.Spirit = runtimePet.Spirit
	detail.SpiritMax = runtimePet.SpiritMax
	detail.HitPct = runtimePet.HitPct
	detail.DodgePct = runtimePet.DodgePct
	detail.CritRatePct = runtimePet.CritRatePct
	detail.CritDmgPct = runtimePet.CritDmgPct
	detail.PhysicalResistPct = runtimePet.PhysicalResistPct
	detail.ReversePhysicalResistPct = runtimePet.ReversePhysicalResistPct
	detail.SkillResistPct = runtimePet.SkillResistPct
	detail.ReverseSkillResistPct = runtimePet.ReverseSkillResistPct
	detail.ConfusionResistPct = runtimePet.ConfusionResistPct
	detail.SleepResistPct = runtimePet.SleepResistPct
	detail.ParalysisResistPct = runtimePet.ParalysisResistPct
	detail.SealResistPct = runtimePet.SealResistPct
	detail.CurseResistPct = runtimePet.CurseResistPct
	detail.CritDmgResistPct = runtimePet.CritDmgResistPct
	detail.CritResistPct = runtimePet.CritResistPct
	detail.CharacterResistPct = runtimePet.CharacterResistPct
	detail.PetResistPct = runtimePet.PetResistPct
	detail.Guard = runtimePet.Guard
	detail.TalentDmgPct = runtimePet.TalentDmgPct
	detail.TalentReducePct = runtimePet.TalentReducePct
	detail.ElementAdvPct = runtimePet.ElementAdvPct
	detail.ElementPenaltyPct = runtimePet.ElementPenaltyPct
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

// resolveRuntimeSkillDefinition 统一从技能运行时缓存读取 skill_id，
// 宠物属性面板与战斗链路共用同一份权威技能模板。
func (s *Service) resolveRuntimeSkillDefinition(skillID uint32) (skill.RuntimeDefinition, bool) {
	if s == nil || s.skillService == nil {
		return skill.RuntimeDefinition{}, false
	}
	return s.skillService.GetRuntimeDefinition(skillID)
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
