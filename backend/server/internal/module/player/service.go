package player

import (
	"context"

	"pocket-pet-remake/server/internal/module/progression"
	"pocket-pet-remake/server/internal/module/skill"
)

// EquipmentCombatRecalculator 在成长层变更后重算并写回含装备加成的最终战斗属性。
type EquipmentCombatRecalculator interface {
	RecalcPlayerCombatStats(ctx context.Context, playerID uint64, refillHP bool) error
}

// ActiveBattleChecker 用于战斗期间跳过会触发读库重算的属性刷新。
type ActiveBattleChecker interface {
	IsPlayerInActiveBattle(playerID uint64) bool
}

// CombatSnapshotRepository 提供最终战斗属性快照的刷新与读取能力。
// 目前仅 PostgreSQL 仓储实现该能力，测试桩会自然回退到旧读链路。
type CombatSnapshotRepository interface {
	RefreshPlayerCombatSnapshot(ctx context.Context, playerID uint64) error
	FindPlayerCombatSnapshot(ctx context.Context, playerID uint64) (*Profile, error)
}

type Service struct {
	repo               Repository
	skillService       *skill.Service
	progressionService *progression.Service
	equipmentRecalc    EquipmentCombatRecalculator
	battleChecker      ActiveBattleChecker
}

func NewService(repo Repository, skillService *skill.Service, progressionService *progression.Service, equipmentRecalc EquipmentCombatRecalculator) *Service {
	return &Service{
		repo:               repo,
		skillService:       skillService,
		progressionService: progressionService,
		equipmentRecalc:    equipmentRecalc,
	}
}

// SetBattleChecker 注入战斗状态检查器，开战快照生效期间不再触发装备重算读库。
func (s *Service) SetBattleChecker(checker ActiveBattleChecker) {
	if s == nil {
		return
	}
	s.battleChecker = checker
}

func (s *Service) GetProfile(ctx context.Context, playerID uint64) (*Profile, error) {
	profile, err := s.repo.FindByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrPlayerNotFound
	}
	s.fillExpToNext(profile)
	return profile, nil
}

// Register 创建一个可直接登录的新账号与初始玩家角色。
// 当前公开注册阶段复用账号名作为玩家名，并按性别选择初始形象。
func (s *Service) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	input = input.Normalize()
	if input.AccountName == "" || input.Password == "" {
		return nil, ErrInvalidRegisterInput
	}

	detail, err := s.repo.CreateForAdmin(ctx, AdminCreatePlayerInput{
		AccountName: input.AccountName,
		Password:    input.Password,
		Name:        input.AccountName,
		SkinID:      resolveStarterSkinIDByGender(input.Gender),
	})
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrPlayerNotFound
	}

	return &RegisterResult{
		PlayerID:    detail.PlayerID,
		AccountName: detail.AccountName,
		PlayerName:  detail.Name,
		SkinID:      detail.SkinID,
	}, nil
}

// GetBattleReadyProfile 返回开战前权威的战斗属性快照。
// 非战斗状态下会先重算并写回当前已佩戴装备加成；战斗中直接返回当前持久化快照，不再读表重算。
func (s *Service) GetBattleReadyProfile(ctx context.Context, playerID uint64) (*Profile, error) {
	inBattle := s.battleChecker != nil && s.battleChecker.IsPlayerInActiveBattle(playerID)
	if !inBattle {
		if snapshotRepo, ok := s.repo.(CombatSnapshotRepository); ok {
			if err := snapshotRepo.RefreshPlayerCombatSnapshot(ctx, playerID); err != nil {
				return nil, err
			}
			return s.GetProfile(ctx, playerID)
		}
	}
	if !inBattle && s.equipmentRecalc != nil {
		if err := s.equipmentRecalc.RecalcPlayerCombatStats(ctx, playerID, false); err != nil {
			return nil, err
		}
	}
	return s.GetProfile(ctx, playerID)
}

// ListAdminPlayers 返回后台玩家列表。
// 这里统一做分页兜底，避免多个后台入口各自复制 page/page_size 默认值。
func (s *Service) ListAdminPlayers(ctx context.Context, query AdminListQuery) (*AdminPlayerList, error) {
	result, err := s.repo.ListForAdmin(ctx, query.Normalize())
	if err != nil {
		return nil, err
	}
	if result == nil {
		query = query.Normalize()
		return &AdminPlayerList{Items: []AdminPlayerSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	s.applyCombatSnapshotsToAdminList(ctx, result.Items)
	return result, nil
}

// GetAdminPlayerDetail 返回后台详情面板需要的人物全量快照。
func (s *Service) GetAdminPlayerDetail(ctx context.Context, playerID uint64) (*AdminPlayerDetail, error) {
	detail, err := s.repo.FindAdminDetailByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrPlayerNotFound
	}
	s.applyCombatSnapshotToAdminDetail(ctx, detail)
	return detail, nil
}

// CreateAdminPlayer 创建完整玩家账号与人物档案，确保后台新增后能直接走正式登录链路。
func (s *Service) CreateAdminPlayer(ctx context.Context, input AdminCreatePlayerInput) (*AdminPlayerDetail, error) {
	input = input.Normalize()
	if input.AccountName == "" || input.Password == "" || input.Name == "" {
		return nil, ErrInvalidAdminInput
	}
	if err := s.validateSkinID(input.SkinID); err != nil {
		return nil, err
	}
	return s.repo.CreateForAdmin(ctx, input)
}

// UpdateAdminPlayer 会覆写后台允许调整的玩家持久化字段。
func (s *Service) UpdateAdminPlayer(ctx context.Context, playerID uint64, input AdminUpdatePlayerInput) (*AdminPlayerDetail, error) {
	input = input.Normalize()
	if input.Name == "" {
		return nil, ErrInvalidAdminInput
	}
	if err := s.validateSkinID(input.SkinID); err != nil {
		return nil, err
	}
	if err := s.validateSkillIDs(ctx, input.SkillIDs); err != nil {
		return nil, err
	}
	detail, err := s.repo.UpdateForAdmin(ctx, playerID, input)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrPlayerNotFound
	}
	return detail, nil
}

// DeleteAdminPlayer 使用软删除禁用 account/player，避免直接物理删除破坏联机关联数据。
func (s *Service) DeleteAdminPlayer(ctx context.Context, playerID uint64) error {
	return s.repo.DeleteForAdmin(ctx, playerID)
}

// PurgeDisabledAdminAccount 永久删除已禁用账号及其全部玩家数据；仓储层会在事务内再次校验禁用状态。
func (s *Service) PurgeDisabledAdminAccount(ctx context.Context, playerID uint64) error {
	return s.repo.PurgeDisabledAccountForAdmin(ctx, playerID)
}

func (s *Service) UpdatePosition(ctx context.Context, playerID uint64, sceneID uint32, posX, posY int32) error {
	return s.repo.UpdatePosition(ctx, playerID, sceneID, posX, posY)
}

// AddExp 增加角色经验并在服务端处理升级、溢出结转与属性点发放。
func (s *Service) AddExp(ctx context.Context, playerID uint64, exp uint64) (*ExpGrantResult, error) {
	if s.progressionService == nil {
		return nil, ErrPlayerNotFound
	}
	applyResult, err := s.progressionService.ApplyExp(ctx, playerID, exp)
	if err != nil {
		return nil, err
	}
	if s.equipmentRecalc != nil {
		refillHP := applyResult.LevelUpCount > 0
		if err := s.equipmentRecalc.RecalcPlayerCombatStats(ctx, playerID, refillHP); err != nil {
			return nil, err
		}
	}
	profile, err := s.GetProfile(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return &ExpGrantResult{
		Profile:          profile,
		LevelUpCount:     applyResult.LevelUpCount,
		AttrPointsGained: applyResult.AttrPointsGained,
		CombatBonusGain:  applyResult.CombatBonusGain,
	}, nil
}

// AllocateAttrPoints 处理玩家主动分配自由属性点。
func (s *Service) AllocateAttrPoints(ctx context.Context, playerID uint64, delta progression.AttrAllocationDelta) (*Profile, error) {
	if s.progressionService == nil {
		return nil, ErrPlayerNotFound
	}
	if err := s.progressionService.AllocateAttrPoints(ctx, playerID, delta); err != nil {
		return nil, err
	}
	if s.equipmentRecalc != nil {
		if err := s.equipmentRecalc.RecalcPlayerCombatStats(ctx, playerID, false); err != nil {
			return nil, err
		}
	}
	return s.GetProfile(ctx, playerID)
}

// AddRewardAttribute 发放战斗/任务奖励中配置的直接属性值加成，并返回最新玩家档案。
func (s *Service) AddRewardAttribute(ctx context.Context, playerID uint64, attrKey string, value uint32) (*Profile, error) {
	if s.repo == nil || playerID == 0 || value == 0 {
		return nil, ErrPlayerNotFound
	}
	if err := s.repo.AddRewardAttribute(ctx, playerID, attrKey, value); err != nil {
		return nil, err
	}
	return s.GetProfile(ctx, playerID)
}

func (s *Service) fillExpToNext(profile *Profile) {
	if profile == nil || s.progressionService == nil {
		return
	}
	profile.ExpToNext = s.progressionService.ExpToNext(profile.Level, profile.Exp)
}

func (s *Service) validateSkillIDs(ctx context.Context, skillIDs []uint32) error {
	if s.skillService == nil || len(skillIDs) == 0 {
		return nil
	}
	return s.skillService.ValidateEnabledSkillIDs(ctx, skillIDs)
}

func (s *Service) validateSkinID(skinID string) error {
	if len(skinID) > 64 {
		return ErrInvalidAdminInput
	}
	return nil
}

func (s *Service) applyCombatSnapshotsToAdminList(ctx context.Context, items []AdminPlayerSummary) {
	snapshotRepo, ok := s.repo.(CombatSnapshotRepository)
	if !ok {
		return
	}
	for index := range items {
		snapshot, err := snapshotRepo.FindPlayerCombatSnapshot(ctx, items[index].PlayerID)
		if err != nil || snapshot == nil {
			if refreshErr := snapshotRepo.RefreshPlayerCombatSnapshot(ctx, items[index].PlayerID); refreshErr != nil {
				continue
			}
			snapshot, err = snapshotRepo.FindPlayerCombatSnapshot(ctx, items[index].PlayerID)
			if err != nil || snapshot == nil {
				continue
			}
		}
		items[index].HP = snapshot.HP
		items[index].HPMax = snapshot.HPMax
		items[index].Vigor = snapshot.Vigor
		items[index].VigorMax = snapshot.VigorMax
		items[index].Spirit = snapshot.Spirit
		items[index].SpiritMax = snapshot.SpiritMax
	}
}

func (s *Service) applyCombatSnapshotToAdminDetail(ctx context.Context, detail *AdminPlayerDetail) {
	if detail == nil {
		return
	}
	snapshotRepo, ok := s.repo.(CombatSnapshotRepository)
	if !ok {
		return
	}
	snapshot, err := snapshotRepo.FindPlayerCombatSnapshot(ctx, detail.PlayerID)
	if err != nil || snapshot == nil {
		if refreshErr := snapshotRepo.RefreshPlayerCombatSnapshot(ctx, detail.PlayerID); refreshErr != nil {
			return
		}
		snapshot, err = snapshotRepo.FindPlayerCombatSnapshot(ctx, detail.PlayerID)
		if err != nil || snapshot == nil {
			return
		}
	}
	detail.HP = snapshot.HP
	detail.HPMax = snapshot.HPMax
	detail.Vigor = snapshot.Vigor
	detail.VigorMax = snapshot.VigorMax
	detail.Spirit = snapshot.Spirit
	detail.SpiritMax = snapshot.SpiritMax
	detail.ATK = snapshot.ATK
	detail.DEF = snapshot.DEF
	detail.SPD = snapshot.SPD
	detail.MANA = snapshot.MANA
	detail.HitPct = snapshot.HitPct
	detail.DodgePct = snapshot.DodgePct
	detail.CritRatePct = snapshot.CritRatePct
	detail.CritDmgPct = snapshot.CritDmgPct
	detail.PhysicalResistPct = snapshot.PhysicalResistPct
	detail.SkillResistPct = snapshot.SkillResistPct
	detail.ConfusionResistPct = snapshot.ConfusionResistPct
	detail.SleepResistPct = snapshot.SleepResistPct
	detail.ParalysisResistPct = snapshot.ParalysisResistPct
	detail.SealResistPct = snapshot.SealResistPct
	detail.CurseResistPct = snapshot.CurseResistPct
	detail.CritResistPct = snapshot.CritResistPct
	detail.CritDmgResistPct = snapshot.CritDmgResistPct
	detail.CharacterResistPct = snapshot.CharacterResistPct
	detail.PetResistPct = snapshot.PetResistPct
	detail.MercenaryResistPct = snapshot.MercenaryResistPct
	detail.GenericShieldPct = snapshot.GenericShieldPct
	detail.Guard = snapshot.Guard
	detail.TalentDmgPct = snapshot.TalentDmgPct
	detail.TalentReducePct = snapshot.TalentReducePct
	detail.ElementAdvPct = snapshot.ElementAdvPct
	detail.ElementPenaltyPct = snapshot.ElementPenaltyPct
	detail.SkillIDs = append([]uint32{}, snapshot.SkillIDs...)
	detail.SkinID = snapshot.SkinID
}

// CountActivePlayers 统计启用中的玩家角色总数，供后台控制台展示。
func (s *Service) CountActivePlayers(ctx context.Context) (uint64, error) {
	if s.repo == nil {
		return 0, nil
	}
	return s.repo.CountActivePlayers(ctx)
}
