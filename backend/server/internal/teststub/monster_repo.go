package teststub

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/monster"
)

// NewMonsterRepository 提供系统怪物模板与遭遇配置的内存桩。
func NewMonsterRepository() *MonsterRepository {
	now := time.Now()
	repo := &MonsterRepository{
		definitions:   make(map[uint32]monster.AdminDefinitionDetail, 4),
		encounters:    make(map[uint64]monster.AdminEncounterDetail, 8),
		battleRewards: defaultStubBattleRewards(),
	}
	definitionSeeds := []monster.AdminUpsertDefinitionInput{
		{MonsterID: 9001, MonsterName: "野生怪物", Description: "默认野外战斗怪物模板", IsEnabled: true, SkinID: "史莱姆_001", Level: 1, Quality: 1, HP: 22, HPMax: 22, ATK: 12, DEF: 9, SPD: 8, MANA: 9, SkillIDs: []uint32{90001, 90002}},
		{MonsterID: 9002, MonsterName: "野性支援", Description: "带治疗技能的怪物模板", IsEnabled: true, SkinID: "史莱姆_001", Level: 1, Quality: 1, HP: 20, HPMax: 20, ATK: 11, DEF: 10, SPD: 9, MANA: 12, SkillIDs: []uint32{90002, 90003}},
	}
	for _, seed := range definitionSeeds {
		repo.definitions[seed.MonsterID] = buildStubMonsterDefinitionDetail(seed.Normalize(), now)
	}
	encounterSeeds := []monster.AdminUpsertEncounterInput{
		{EntityID: 90001, EncounterName: "GuideNPC 遭遇", Description: "新手引导战斗", SpawnMonsterIDs: []uint32{9001}, IsEnabled: true},
		{EntityID: 90002, EncounterName: "StationKeeper 遭遇", Description: "车站守卫战斗", SpawnMonsterIDs: []uint32{9001}, IsEnabled: true},
		{EntityID: 90004, EncounterName: "NorthFieldScout 遭遇", Description: "北部scout双人战", SpawnMonsterIDs: []uint32{9001, 9001}, IsEnabled: true},
		{EntityID: 90005, EncounterName: "SchoolCaretaker 遭遇", Description: "学校看护双人战", SpawnMonsterIDs: []uint32{9001, 9001}, IsEnabled: true},
		{EntityID: 90006, EncounterName: "BattleGuide 遭遇", Description: "战斗教学：攻击+支援", SpawnMonsterIDs: []uint32{9001, 9002}, IsEnabled: true},
	}
	for _, seed := range encounterSeeds {
		repo.encounters[seed.EntityID] = buildStubMonsterEncounterDetail(seed.Normalize(), now)
	}
	repo.wildEncounterDetails = map[uint32]monster.AdminWildEncounterDetail{
		4: buildStubWildEncounterDetail(monster.AdminUpsertWildEncounterInput{
			SceneID: 4, EncounterName: "北部野外暗雷", Description: "测试暗雷配置",
			EncounterRate: 800, SpawnMonsterIDs: []uint32{9001}, IsEnabled: true,
		}.Normalize(), now),
	}
	return repo
}

type MonsterRepository struct {
	mu                   sync.RWMutex
	definitions          map[uint32]monster.AdminDefinitionDetail
	encounters           map[uint64]monster.AdminEncounterDetail
	wildEncounterDetails map[uint32]monster.AdminWildEncounterDetail
	battleRewards        map[uint32][]monster.BattleRewardEntry
	nextBattleRewardID   uint64
}

func (r *MonsterRepository) ListDefinitionsForAdmin(_ context.Context, query monster.AdminDefinitionListQuery) (*monster.AdminDefinitionList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = query.Normalize()
	items := make([]monster.AdminDefinitionSummary, 0, len(r.definitions))
	for _, current := range r.definitions {
		if query.MonsterID > 0 && current.MonsterID != query.MonsterID {
			continue
		}
		if query.Name != "" && !strings.Contains(current.MonsterName, query.Name) {
			continue
		}
		if query.Enabled != nil && current.IsEnabled != *query.Enabled {
			continue
		}
		items = append(items, adminMonsterSummaryFromDetail(current))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].MonsterID < items[j].MonsterID })
	return &monster.AdminDefinitionList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *MonsterRepository) FindDefinitionForAdmin(_ context.Context, monsterID uint32) (*monster.AdminDefinitionDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.definitions[monsterID]
	if !ok {
		return nil, nil
	}
	copied := current
	if len(current.SkillIDs) > 0 {
		copied.SkillIDs = append([]uint32{}, current.SkillIDs...)
	}
	return &copied, nil
}

func (r *MonsterRepository) CreateDefinitionForAdmin(_ context.Context, input monster.AdminUpsertDefinitionInput) (*monster.AdminDefinitionDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.definitions[input.MonsterID]; exists {
		return nil, monster.ErrMonsterDefinitionConflict
	}
	detail := buildStubMonsterDefinitionDetail(input.Normalize(), time.Now())
	r.definitions[input.MonsterID] = detail
	copied := detail
	return &copied, nil
}

func (r *MonsterRepository) UpdateDefinitionForAdmin(_ context.Context, monsterID uint32, input monster.AdminUpsertDefinitionInput) (*monster.AdminDefinitionDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.definitions[monsterID]
	if !ok {
		return nil, nil
	}
	detail := buildStubMonsterDefinitionDetail(input.Normalize(), current.CreatedAt)
	detail.MonsterID = monsterID
	detail.UpdatedAt = time.Now()
	r.definitions[monsterID] = detail
	copied := detail
	return &copied, nil
}

func (r *MonsterRepository) DeleteDefinitionForAdmin(_ context.Context, monsterID uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.definitions[monsterID]; !ok {
		return monster.ErrMonsterDefinitionNotFound
	}
	delete(r.definitions, monsterID)
	return nil
}

func (r *MonsterRepository) MapUsableMonsterIDs(_ context.Context, monsterIDs []uint32) (map[uint32]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[uint32]bool, len(monsterIDs))
	for _, monsterID := range monsterIDs {
		current, ok := r.definitions[monsterID]
		if ok && current.IsEnabled {
			result[monsterID] = true
		}
	}
	return result, nil
}

func (r *MonsterRepository) FindRuntimeDefinition(_ context.Context, monsterID uint32) (*monster.RuntimeDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.definitions[monsterID]
	if !ok || !current.IsEnabled {
		return nil, nil
	}
	return &monster.RuntimeDefinition{
		MonsterID: current.MonsterID, MonsterName: current.MonsterName,
		Level: current.BaseStats.Level, Quality: current.BaseStats.Quality,
		HP: current.BaseStats.HP, HPMax: current.BaseStats.HPMax,
		ATK: current.BaseStats.ATK, DEF: current.BaseStats.DEF, SPD: current.BaseStats.SPD, MANA: current.BaseStats.MANA,
		SkillIDs: append([]uint32{}, current.SkillIDs...),
	}, nil
}

func (r *MonsterRepository) ListEncountersForAdmin(_ context.Context, query monster.AdminEncounterListQuery) (*monster.AdminEncounterList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = query.Normalize()
	items := make([]monster.AdminEncounterSummary, 0, len(r.encounters))
	for _, current := range r.encounters {
		if query.EntityID > 0 && current.EntityID != query.EntityID {
			continue
		}
		if query.Name != "" && !strings.Contains(current.EncounterName, query.Name) {
			continue
		}
		if query.Enabled != nil && current.IsEnabled != *query.Enabled {
			continue
		}
		items = append(items, adminMonsterEncounterSummaryFromDetail(current))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].EntityID < items[j].EntityID })
	return &monster.AdminEncounterList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *MonsterRepository) FindEncounterForAdmin(_ context.Context, entityID uint64) (*monster.AdminEncounterDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.encounters[entityID]
	if !ok {
		return nil, nil
	}
	copied := current
	if len(current.SpawnMonsterIDs) > 0 {
		copied.SpawnMonsterIDs = append([]uint32{}, current.SpawnMonsterIDs...)
	}
	return &copied, nil
}

func (r *MonsterRepository) CreateEncounterForAdmin(_ context.Context, input monster.AdminUpsertEncounterInput) (*monster.AdminEncounterDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.encounters[input.EntityID]; exists {
		return nil, monster.ErrMonsterEncounterConflict
	}
	detail := buildStubMonsterEncounterDetail(input.Normalize(), time.Now())
	r.encounters[input.EntityID] = detail
	copied := detail
	return &copied, nil
}

func (r *MonsterRepository) UpdateEncounterForAdmin(_ context.Context, entityID uint64, input monster.AdminUpsertEncounterInput) (*monster.AdminEncounterDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.encounters[entityID]
	if !ok {
		return nil, nil
	}
	detail := buildStubMonsterEncounterDetail(input.Normalize(), current.CreatedAt)
	detail.EntityID = entityID
	detail.UpdatedAt = time.Now()
	r.encounters[entityID] = detail
	copied := detail
	return &copied, nil
}

func (r *MonsterRepository) DeleteEncounterForAdmin(_ context.Context, entityID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.encounters[entityID]; !ok {
		return monster.ErrMonsterEncounterNotFound
	}
	delete(r.encounters, entityID)
	return nil
}

func (r *MonsterRepository) FindRuntimeEncounter(_ context.Context, entityID uint64) (*monster.RuntimeEncounter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.encounters[entityID]
	if !ok || !current.IsEnabled || len(current.SpawnMonsterIDs) == 0 {
		return nil, nil
	}
	slots := make([]monster.RuntimeEncounterSlot, 0, len(current.SpawnMonsterIDs))
	for _, monsterID := range current.SpawnMonsterIDs {
		definition, exists := r.definitions[monsterID]
		if !exists || !definition.IsEnabled {
			continue
		}
		slots = append(slots, monster.RuntimeEncounterSlot{
			MonsterID: definition.MonsterID, MonsterName: definition.MonsterName, SkinID: definition.SkinID,
			Level: definition.BaseStats.Level, HP: definition.BaseStats.HP, HPMax: definition.BaseStats.HPMax,
			ATK: definition.BaseStats.ATK, DEF: definition.BaseStats.DEF, SPD: definition.BaseStats.SPD, MANA: definition.BaseStats.MANA,
			SkillIDs: append([]uint32{}, definition.SkillIDs...), RewardEnabled: true,
		})
	}
	if len(slots) == 0 {
		return nil, nil
	}
	return &monster.RuntimeEncounter{EntityID: entityID, EncounterName: current.EncounterName, Slots: slots}, nil
}

func (r *MonsterRepository) FindRuntimeWildEncounterConfig(_ context.Context, sceneID uint32) (*monster.RuntimeWildEncounterConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.wildEncounterDetails[sceneID]
	if !ok || !current.IsEnabled || len(current.SpawnMonsterIDs) == 0 || current.EncounterRate == 0 {
		return nil, nil
	}
	return &monster.RuntimeWildEncounterConfig{
		Enabled:         true,
		SceneID:         current.SceneID,
		EncounterRate:   current.EncounterRate,
		SpawnMonsterIDs: append([]uint32{}, current.SpawnMonsterIDs...),
	}, nil
}

func (r *MonsterRepository) FindRuntimeWildEncounter(_ context.Context, sceneID uint32) (*monster.RuntimeWildEncounter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.wildEncounterDetails[sceneID]
	if !ok || !current.IsEnabled || len(current.SpawnMonsterIDs) == 0 {
		return nil, nil
	}
	slots := make([]monster.RuntimeEncounterSlot, 0, len(current.SpawnMonsterIDs))
	for _, monsterID := range current.SpawnMonsterIDs {
		definition, exists := r.definitions[monsterID]
		if !exists || !definition.IsEnabled {
			continue
		}
		slots = append(slots, monster.RuntimeEncounterSlot{
			MonsterID: definition.MonsterID, MonsterName: definition.MonsterName, SkinID: definition.SkinID,
			Level: definition.BaseStats.Level, HP: definition.BaseStats.HP, HPMax: definition.BaseStats.HPMax,
			ATK: definition.BaseStats.ATK, DEF: definition.BaseStats.DEF, SPD: definition.BaseStats.SPD, MANA: definition.BaseStats.MANA,
			SkillIDs: append([]uint32{}, definition.SkillIDs...), RewardEnabled: true,
		})
	}
	if len(slots) == 0 {
		return nil, nil
	}
	return &monster.RuntimeWildEncounter{
		SceneID: sceneID, EncounterName: current.EncounterName, Slots: slots, Rewards: append([]monster.BattleRewardEntry{}, current.Rewards...),
	}, nil
}

func (r *MonsterRepository) ListWildEncountersForAdmin(_ context.Context, query monster.AdminWildEncounterListQuery) (*monster.AdminWildEncounterList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = query.Normalize()
	items := make([]monster.AdminWildEncounterSummary, 0, len(r.wildEncounterDetails))
	for _, current := range r.wildEncounterDetails {
		if query.SceneID > 0 && current.SceneID != query.SceneID {
			continue
		}
		if query.Name != "" && !strings.Contains(current.EncounterName, query.Name) {
			continue
		}
		if query.Enabled != nil && current.IsEnabled != *query.Enabled {
			continue
		}
		items = append(items, adminWildEncounterSummaryFromDetail(current))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SceneID < items[j].SceneID })
	return &monster.AdminWildEncounterList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *MonsterRepository) FindWildEncounterForAdmin(_ context.Context, sceneID uint32) (*monster.AdminWildEncounterDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.wildEncounterDetails[sceneID]
	if !ok {
		return nil, nil
	}
	copied := current
	if len(current.SpawnMonsterIDs) > 0 {
		copied.SpawnMonsterIDs = append([]uint32{}, current.SpawnMonsterIDs...)
	}
	return &copied, nil
}

func (r *MonsterRepository) CreateWildEncounterForAdmin(_ context.Context, input monster.AdminUpsertWildEncounterInput) (*monster.AdminWildEncounterDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.wildEncounterDetails[input.SceneID]; exists {
		return nil, monster.ErrSceneWildEncounterConflict
	}
	detail := buildStubWildEncounterDetail(input.Normalize(), time.Now())
	r.wildEncounterDetails[input.SceneID] = detail
	copied := detail
	return &copied, nil
}

func (r *MonsterRepository) UpdateWildEncounterForAdmin(_ context.Context, sceneID uint32, input monster.AdminUpsertWildEncounterInput) (*monster.AdminWildEncounterDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.wildEncounterDetails[sceneID]
	if !ok {
		return nil, nil
	}
	detail := buildStubWildEncounterDetail(input.Normalize(), current.CreatedAt)
	detail.SceneID = sceneID
	detail.UpdatedAt = time.Now()
	r.wildEncounterDetails[sceneID] = detail
	copied := detail
	return &copied, nil
}

func (r *MonsterRepository) DeleteWildEncounterForAdmin(_ context.Context, sceneID uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.wildEncounterDetails[sceneID]; !ok {
		return monster.ErrSceneWildEncounterNotFound
	}
	delete(r.wildEncounterDetails, sceneID)
	return nil
}

func (r *MonsterRepository) FindCaptureConfig(_ context.Context, monsterID uint32) (*monster.CaptureConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.definitions[monsterID]
	if !ok || !current.IsEnabled {
		return nil, nil
	}
	return &monster.CaptureConfig{
		MonsterID:       current.MonsterID,
		IsCapturable:    current.IsCapturable,
		CapturePetID:    current.CapturePetID,
		CaptureRateBase: current.CaptureRateBase,
		CaptureMinHPPct: current.CaptureMinHPPct,
		CaptureItemIDs:  append([]uint32{}, current.CaptureItemIDs...),
	}, nil
}

func buildStubMonsterDefinitionDetail(input monster.AdminUpsertDefinitionInput, createdAt time.Time) monster.AdminDefinitionDetail {
	statusText := "停用"
	if input.IsEnabled {
		statusText = "启用"
	}
	skillIDs := append([]uint32{}, input.SkillIDs...)
	return monster.AdminDefinitionDetail{
		MonsterID: input.MonsterID, MonsterName: input.MonsterName, Description: input.Description,
		IsEnabled: input.IsEnabled, StatusText: statusText, SkinID: input.SkinID,
		BaseStats: monster.AdminDefinitionBaseStats{
			Level: input.Level, Quality: input.Quality, HP: input.HP, HPMax: input.HPMax,
			ATK: input.ATK, DEF: input.DEF, SPD: input.SPD, MANA: input.MANA,
		},
		SkillIDs:        skillIDs,
		IsCapturable:    input.IsCapturable,
		CapturePetID:    input.CapturePetID,
		CaptureRateBase: input.CaptureRateBase,
		CaptureMinHPPct: input.CaptureMinHPPct,
		CaptureItemIDs:  append([]uint32{}, input.CaptureItemIDs...),
		CreatedAt:       createdAt, UpdatedAt: time.Now(),
	}
}

func adminMonsterSummaryFromDetail(detail monster.AdminDefinitionDetail) monster.AdminDefinitionSummary {
	return monster.AdminDefinitionSummary{
		MonsterID: detail.MonsterID, MonsterName: detail.MonsterName,
		Level: detail.BaseStats.Level, Quality: detail.BaseStats.Quality,
		IsEnabled: detail.IsEnabled, StatusText: detail.StatusText, SkinID: detail.SkinID,
		CreatedAt: detail.CreatedAt, UpdatedAt: detail.UpdatedAt,
	}
}

func buildStubMonsterEncounterDetail(input monster.AdminUpsertEncounterInput, createdAt time.Time) monster.AdminEncounterDetail {
	statusText := "停用"
	if input.IsEnabled {
		statusText = "启用"
	}
	spawnIDs := append([]uint32{}, input.SpawnMonsterIDs...)
	return monster.AdminEncounterDetail{
		EntityID: input.EntityID, EncounterName: input.EncounterName, Description: input.Description,
		SpawnMonsterIDs: spawnIDs, IsEnabled: input.IsEnabled, StatusText: statusText,
		CreatedAt: createdAt, UpdatedAt: time.Now(),
	}
}

func adminMonsterEncounterSummaryFromDetail(detail monster.AdminEncounterDetail) monster.AdminEncounterSummary {
	return monster.AdminEncounterSummary{
		EntityID: detail.EntityID, EncounterName: detail.EncounterName,
		SpawnCount: uint32(len(detail.SpawnMonsterIDs)), IsEnabled: detail.IsEnabled, StatusText: detail.StatusText,
		CreatedAt: detail.CreatedAt, UpdatedAt: detail.UpdatedAt,
	}
}

func buildStubWildEncounterDetail(input monster.AdminUpsertWildEncounterInput, createdAt time.Time) monster.AdminWildEncounterDetail {
	statusText := "停用"
	if input.IsEnabled {
		statusText = "启用"
	}
	spawnIDs := append([]uint32{}, input.SpawnMonsterIDs...)
	return monster.AdminWildEncounterDetail{
		SceneID: input.SceneID, EncounterName: input.EncounterName, Description: input.Description,
		EncounterRate: input.EncounterRate, SpawnMonsterIDs: spawnIDs,
		IsEnabled: input.IsEnabled, StatusText: statusText,
		CreatedAt: createdAt, UpdatedAt: time.Now(),
	}
}

func adminWildEncounterSummaryFromDetail(detail monster.AdminWildEncounterDetail) monster.AdminWildEncounterSummary {
	return monster.AdminWildEncounterSummary{
		SceneID: detail.SceneID, EncounterName: detail.EncounterName,
		EncounterRate: detail.EncounterRate, SpawnCount: uint32(len(detail.SpawnMonsterIDs)),
		IsEnabled: detail.IsEnabled, StatusText: detail.StatusText,
		CreatedAt: detail.CreatedAt, UpdatedAt: detail.UpdatedAt,
	}
}

func defaultStubBattleRewards() map[uint32][]monster.BattleRewardEntry {
	return map[uint32][]monster.BattleRewardEntry{
		9001: {
			{ID: 1, MonsterID: 9001, RewardType: monster.RewardTypeExp, ExpTarget: monster.ExpTargetPlayer, ExpValue: 28, SortOrder: 1, Status: 1},
			{ID: 2, MonsterID: 9001, RewardType: monster.RewardTypeExp, ExpTarget: monster.ExpTargetPet, ExpValue: 28, SortOrder: 2, Status: 1},
			{ID: 3, MonsterID: 9001, RewardType: monster.RewardTypeItem, ItemID: 3101, Quantity: 1, SortOrder: 3, Status: 1},
		},
		9002: {
			{ID: 4, MonsterID: 9002, RewardType: monster.RewardTypeExp, ExpTarget: monster.ExpTargetPlayer, ExpValue: 36, SortOrder: 1, Status: 1},
			{ID: 5, MonsterID: 9002, RewardType: monster.RewardTypeExp, ExpTarget: monster.ExpTargetPet, ExpValue: 36, SortOrder: 2, Status: 1},
			{ID: 6, MonsterID: 9002, RewardType: monster.RewardTypeItem, ItemID: 3102, Quantity: 1, SortOrder: 3, Status: 1},
		},
	}
}

func (r *MonsterRepository) ListBattleRewards(_ context.Context) ([]monster.BattleRewardEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]monster.BattleRewardEntry, 0)
	for _, entries := range r.battleRewards {
		result = append(result, entries...)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].MonsterID == result[j].MonsterID {
			return result[i].SortOrder < result[j].SortOrder
		}
		return result[i].MonsterID < result[j].MonsterID
	})
	return result, nil
}

func (r *MonsterRepository) ListBattleRewardsByMonsterID(_ context.Context, monsterID uint32) ([]monster.BattleRewardEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]monster.BattleRewardEntry(nil), r.battleRewards[monsterID]...), nil
}

func (r *MonsterRepository) ReplaceBattleRewardsForMonster(_ context.Context, monsterID uint32, rewards []monster.AdminBattleRewardInput) ([]monster.BattleRewardEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entries := make([]monster.BattleRewardEntry, 0, len(rewards))
	for index, reward := range rewards {
		r.nextBattleRewardID++
		entries = append(entries, monster.BattleRewardEntry{
			ID:         r.nextBattleRewardID,
			MonsterID:  monsterID,
			RewardType: reward.RewardType,
			ExpTarget:  reward.ExpTarget,
			ItemID:     reward.ItemID,
			Quantity:   reward.Quantity,
			ExpValue:   reward.ExpValue,
			AttrKey:    reward.AttrKey,
			DropRate:   reward.DropRate,
			SortOrder:  reward.SortOrder,
			Status:     reward.Status,
			GrantOnce:  reward.GrantOnce,
		})
		if entries[len(entries)-1].SortOrder == 0 {
			entries[len(entries)-1].SortOrder = uint32(index + 1)
		}
	}
	r.battleRewards[monsterID] = entries
	return append([]monster.BattleRewardEntry(nil), entries...), nil
}
