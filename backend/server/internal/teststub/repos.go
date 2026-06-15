package teststub

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/item"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
	"pocket-pet-remake/server/internal/module/wallet"
	"pocket-pet-remake/server/internal/module/world"
)

const (
	DemoAccountName  = "demo"
	DemoPassword     = "demo123"
	DemoAccountID    = 1
	DemoPlayerID     = 10001
	RivalAccountName = "rival"
	RivalPassword    = "rival123"
	RivalAccountID   = 2
	RivalPlayerID    = 10002
)

// NewBattleRepository provides an in-memory reward record set so battle reward
// grant logic can verify the duplicate guard without requiring PostgreSQL.
func NewBattleRepository() *BattleRepository {
	return &BattleRepository{records: map[string]battle.RewardRecord{}}
}

type BattleRepository struct {
	mu      sync.Mutex
	records map[string]battle.RewardRecord
}

func (r *BattleRepository) CreateRewardRecord(_ context.Context, record battle.RewardRecord) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := battleRecordKey(record.BattleID, record.PlayerID)
	if _, exists := r.records[key]; exists {
		return false, nil
	}
	r.records[key] = record
	return true, nil
}

func battleRecordKey(battleID uint64, playerID uint64) string {
	return fmt.Sprintf("%d:%d", battleID, playerID)
}

// NewAccountRepository returns a small in-process auth repository used only by
// tests. It mirrors the seeded demo account so auth tests stay independent from
// a live PostgreSQL/Redis environment.
func NewAccountRepository() *AccountRepository {
	now := time.Now()
	return &AccountRepository{
		accounts: map[string]auth.Account{
			DemoAccountName: {
				AccountID:    DemoAccountID,
				AccountName:  DemoAccountName,
				PasswordHash: auth.HashPassword(DemoPassword),
				PlayerID:     DemoPlayerID,
				PlayerName:   "DemoTrainer",
				PlayerLevel:  1,
			},
			RivalAccountName: {
				AccountID:    RivalAccountID,
				AccountName:  RivalAccountName,
				PasswordHash: auth.HashPassword(RivalPassword),
				PlayerID:     RivalPlayerID,
				PlayerName:   "RivalTrainer",
				PlayerLevel:  1,
			},
		},
		lastLoginAt: map[uint64]time.Time{
			DemoAccountID: now,
		},
		createdAt: map[uint64]time.Time{
			DemoAccountID:  now.Add(-48 * time.Hour),
			RivalAccountID: now,
		},
	}
}

type AccountRepository struct {
	mu            sync.RWMutex
	accounts      map[string]auth.Account
	lastLoginAt   map[uint64]time.Time
	createdAt     map[uint64]time.Time
}

func (r *AccountRepository) FindByAccountName(_ context.Context, accountName string) (*auth.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, ok := r.accounts[accountName]
	if !ok {
		return nil, nil
	}
	copied := account
	return &copied, nil
}

func (r *AccountRepository) TouchLastLoginAt(_ context.Context, accountID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lastLoginAt == nil {
		r.lastLoginAt = make(map[uint64]time.Time)
	}
	r.lastLoginAt[accountID] = time.Now()
	return nil
}

func (r *AccountRepository) GetDashboardAccountMetrics(_ context.Context, dayStart, dayEnd time.Time) (*auth.AccountDashboardMetrics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := &auth.AccountDashboardMetrics{}
	for _, account := range r.accounts {
		metrics.TotalAccounts++
		if lastLogin, ok := r.lastLoginAt[account.AccountID]; ok && !lastLogin.Before(dayStart) && lastLogin.Before(dayEnd) {
			metrics.DailyActiveAccounts++
		}
		if created, ok := r.createdAt[account.AccountID]; ok && !created.Before(dayStart) && created.Before(dayEnd) {
			metrics.NewAccountsToday++
		}
	}
	return metrics, nil
}

// NewWSTokenRepository returns a test-only token store so auth and HTTP tests
// can verify one-time token behavior without bringing up Redis.
func NewWSTokenRepository() *WSTokenRepository {
	return &WSTokenRepository{
		tokens: make(map[string]auth.WSTokenRecord),
		now:    time.Now,
	}
}

type WSTokenRepository struct {
	mu     sync.Mutex
	tokens map[string]auth.WSTokenRecord
	now    func() time.Time
}

func (r *WSTokenRepository) Store(_ context.Context, record auth.WSTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[record.Token] = record
	return nil
}

func (r *WSTokenRepository) Consume(_ context.Context, token string) (*auth.WSTokenRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.tokens[token]
	if !ok {
		return nil, nil
	}
	delete(r.tokens, token)
	if record.ExpiresAt.Before(r.now()) {
		return nil, nil
	}
	copied := record
	return &copied, nil
}

// NewPlayerRepository builds the player state expected by world and battle
// transport tests. The defaults intentionally match the seeded demo account.
func NewPlayerRepository() *PlayerRepository {
	return &PlayerRepository{
		nextID: RivalPlayerID + 1,
		players: map[uint64]player.Profile{
			DemoPlayerID: {
				PlayerID:           DemoPlayerID,
				Name:               "DemoTrainer",
				Level:              1,
				Exp:                0,
				Gold:               100,
				SceneID:            1,
				PosX:               8,
				PosY:               6,
				HP:                 120,
				HPMax:              120,
				Energy:             100,
				EnergyMax:          100,
				ATK:                24,
				DEF:                12,
				SPD:                18,
				MANA:               20,
				HitPct:             10,
				DodgePct:           6,
				CritRatePct:        10,
				CritDmgPct:         155,
				PhysicalResistPct:  6,
				SkillResistPct:     4,
				ConfusionResistPct: 8,
				SleepResistPct:     8,
				ParalysisResistPct: 6,
				SealResistPct:      6,
				CurseResistPct:     5,
				CritResistPct:      4,
				CritDmgResistPct:   10,
				PetResistPct:       4,
				GenericShieldPct:   2,
				SkillIDs:           []uint32{1101, 1001},
				SkinID:             player.DefaultPlayerSkinID,
			},
			RivalPlayerID: {
				PlayerID:           RivalPlayerID,
				Name:               "RivalTrainer",
				Level:              1,
				Exp:                0,
				Gold:               100,
				SceneID:            1,
				PosX:               9,
				PosY:               6,
				HP:                 110,
				HPMax:              110,
				Energy:             100,
				EnergyMax:          100,
				ATK:                23,
				DEF:                11,
				SPD:                17,
				MANA:               18,
				HitPct:             10,
				DodgePct:           6,
				CritRatePct:        10,
				CritDmgPct:         155,
				PhysicalResistPct:  6,
				SkillResistPct:     4,
				ConfusionResistPct: 8,
				SleepResistPct:     8,
				ParalysisResistPct: 6,
				SealResistPct:      6,
				CurseResistPct:     5,
				CritResistPct:      4,
				CritDmgResistPct:   10,
				PetResistPct:       4,
				GenericShieldPct:   2,
				SkillIDs:           []uint32{1101, 1001},
				SkinID:             player.DefaultPlayerSkinID,
			},
		},
	}
}

type PlayerRepository struct {
	mu      sync.RWMutex
	players map[uint64]player.Profile
	nextID  uint64
}

func (r *PlayerRepository) FindByPlayerID(_ context.Context, playerID uint64) (*player.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.players[playerID]
	if !ok {
		return nil, nil
	}
	copied := profile
	return &copied, nil
}

func (r *PlayerRepository) ListForAdmin(_ context.Context, query player.AdminListQuery) (*player.AdminPlayerList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]player.AdminPlayerSummary, 0, len(r.players))
	for _, profile := range r.players {
		if query.PlayerID > 0 && profile.PlayerID != query.PlayerID {
			continue
		}
		if query.Name != "" && !strings.Contains(strings.ToLower(profile.Name), strings.ToLower(query.Name)) {
			continue
		}
		if query.Status != nil && *query.Status != 1 {
			continue
		}
		items = append(items, player.AdminPlayerSummary{
			PlayerID:    profile.PlayerID,
			AccountName: fmt.Sprintf("player_%d", profile.PlayerID),
			Name:        profile.Name,
			Level:       profile.Level,
			Gold:        uint64(profile.Gold),
			Status:      1,
			StatusText:  player.AdminPlayerStatusText(1),
			SceneID:     profile.SceneID,
			HP:          profile.HP,
			HPMax:       profile.HPMax,
			Energy:      profile.Energy,
			EnergyMax:   profile.EnergyMax,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}
	return &player.AdminPlayerList{
		Items:    items,
		Total:    uint64(len(items)),
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (r *PlayerRepository) FindAdminDetailByPlayerID(_ context.Context, playerID uint64) (*player.AdminPlayerDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.players[playerID]
	if !ok {
		return nil, nil
	}
	return &player.AdminPlayerDetail{
		PlayerID:           profile.PlayerID,
		AccountID:          profile.PlayerID,
		AccountName:        fmt.Sprintf("player_%d", profile.PlayerID),
		Name:               profile.Name,
		Level:              profile.Level,
		Exp:                profile.Exp,
		Gold:               uint64(profile.Gold),
		Status:             1,
		StatusText:         player.AdminPlayerStatusText(1),
		SceneID:            profile.SceneID,
		PosX:               profile.PosX,
		PosY:               profile.PosY,
		HP:                 profile.HP,
		HPMax:              profile.HPMax,
		Energy:             profile.Energy,
		EnergyMax:          profile.EnergyMax,
		ATK:                profile.ATK,
		DEF:                profile.DEF,
		SPD:                profile.SPD,
		MANA:               profile.MANA,
		HitPct:             profile.HitPct,
		DodgePct:           profile.DodgePct,
		CritRatePct:        profile.CritRatePct,
		CritDmgPct:         profile.CritDmgPct,
		PhysicalResistPct:  profile.PhysicalResistPct,
		SkillResistPct:     profile.SkillResistPct,
		ConfusionResistPct: profile.ConfusionResistPct,
		SleepResistPct:     profile.SleepResistPct,
		ParalysisResistPct: profile.ParalysisResistPct,
		SealResistPct:      profile.SealResistPct,
		CurseResistPct:     profile.CurseResistPct,
		CritResistPct:      profile.CritResistPct,
		CritDmgResistPct:   profile.CritDmgResistPct,
		CharacterResistPct: profile.CharacterResistPct,
		PetResistPct:       profile.PetResistPct,
		MercenaryResistPct: profile.MercenaryResistPct,
		GenericShieldPct:   profile.GenericShieldPct,
		SkillIDs:           append([]uint32{}, profile.SkillIDs...),
		SkinID:             profile.SkinID,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}, nil
}

func (r *PlayerRepository) CreateForAdmin(_ context.Context, input player.AdminCreatePlayerInput) (*player.AdminPlayerDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, current := range r.players {
		if strings.EqualFold(current.Name, input.Name) {
			return nil, player.ErrPlayerNameDuplicated
		}
	}

	input = input.Normalize()
	playerID := r.nextID
	r.nextID++
	r.players[playerID] = player.Profile{
		PlayerID:  playerID,
		Name:      input.Name,
		Level:     input.Level,
		Gold:      uint32(input.Gold),
		SceneID:   input.SceneID,
		PosX:      input.PosX,
		PosY:      input.PosY,
		HP:        input.HP,
		HPMax:     input.HPMax,
		Energy:    input.Energy,
		EnergyMax: input.EnergyMax,
		ATK:       input.ATK,
		DEF:       input.DEF,
		SPD:       input.SPD,
		MANA:      input.MANA,
		SkillIDs:  append([]uint32{}, input.SkillIDs...),
		SkinID:    resolveTeststubPlayerSkinID(input.SkinID),
	}
	profile := r.players[playerID]
	return &player.AdminPlayerDetail{
		PlayerID:    profile.PlayerID,
		AccountID:   profile.PlayerID,
		AccountName: input.AccountName,
		Name:        profile.Name,
		Level:       profile.Level,
		Gold:        uint64(profile.Gold),
		Status:      1,
		StatusText:  player.AdminPlayerStatusText(1),
		SceneID:     profile.SceneID,
		PosX:        profile.PosX,
		PosY:        profile.PosY,
		HP:          profile.HP,
		HPMax:       profile.HPMax,
		Energy:      profile.Energy,
		EnergyMax:   profile.EnergyMax,
		ATK:         profile.ATK,
		DEF:         profile.DEF,
		SPD:         profile.SPD,
		MANA:        profile.MANA,
		SkillIDs:    append([]uint32{}, profile.SkillIDs...),
		SkinID:      profile.SkinID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func resolveTeststubPlayerSkinID(skinID string) string {
	skinID = strings.TrimSpace(skinID)
	if skinID == "" {
		return player.DefaultPlayerSkinID
	}
	return skinID
}

func (r *PlayerRepository) UpdateForAdmin(_ context.Context, playerID uint64, input player.AdminUpdatePlayerInput) (*player.AdminPlayerDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.players[playerID]
	if !ok {
		return nil, player.ErrPlayerNotFound
	}
	input = input.Normalize()
	current.Name = input.Name
	current.Level = input.Level
	current.Exp = input.Exp
	current.Gold = uint32(input.Gold)
	current.SceneID = input.SceneID
	current.PosX = input.PosX
	current.PosY = input.PosY
	current.HP = input.HP
	current.HPMax = input.HPMax
	current.Energy = input.Energy
	current.EnergyMax = input.EnergyMax
	current.ATK = input.ATK
	current.DEF = input.DEF
	current.SPD = input.SPD
	current.MANA = input.MANA
	current.SkillIDs = append([]uint32{}, input.SkillIDs...)
	current.SkinID = resolveTeststubPlayerSkinID(input.SkinID)
	r.players[playerID] = current
	return &player.AdminPlayerDetail{
		PlayerID:           current.PlayerID,
		AccountID:          current.PlayerID,
		AccountName:        fmt.Sprintf("player_%d", current.PlayerID),
		Name:               current.Name,
		Level:              current.Level,
		Exp:                current.Exp,
		Gold:               uint64(current.Gold),
		Status:             input.Status,
		StatusText:         player.AdminPlayerStatusText(input.Status),
		SceneID:            current.SceneID,
		PosX:               current.PosX,
		PosY:               current.PosY,
		HP:                 current.HP,
		HPMax:              current.HPMax,
		Energy:             current.Energy,
		EnergyMax:          current.EnergyMax,
		ATK:                current.ATK,
		DEF:                current.DEF,
		SPD:                current.SPD,
		MANA:               current.MANA,
		HitPct:             current.HitPct,
		DodgePct:           current.DodgePct,
		CritRatePct:        current.CritRatePct,
		CritDmgPct:         current.CritDmgPct,
		PhysicalResistPct:  current.PhysicalResistPct,
		SkillResistPct:     current.SkillResistPct,
		ConfusionResistPct: current.ConfusionResistPct,
		SleepResistPct:     current.SleepResistPct,
		ParalysisResistPct: current.ParalysisResistPct,
		SealResistPct:      current.SealResistPct,
		CurseResistPct:     current.CurseResistPct,
		CritResistPct:      current.CritResistPct,
		CritDmgResistPct:   current.CritDmgResistPct,
		CharacterResistPct: current.CharacterResistPct,
		PetResistPct:       current.PetResistPct,
		MercenaryResistPct: current.MercenaryResistPct,
		GenericShieldPct:   current.GenericShieldPct,
		SkillIDs:           append([]uint32{}, current.SkillIDs...),
		SkinID:             current.SkinID,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}, nil
}

func (r *PlayerRepository) DeleteForAdmin(_ context.Context, playerID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[playerID]; !ok {
		return player.ErrPlayerNotFound
	}
	delete(r.players, playerID)
	return nil
}

func (r *PlayerRepository) CountActivePlayers(_ context.Context) (uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return uint64(len(r.players)), nil
}

func (r *PlayerRepository) UpdatePosition(_ context.Context, playerID uint64, sceneID uint32, posX, posY int32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.players[playerID]
	if !ok {
		return player.ErrPlayerNotFound
	}
	current.SceneID = sceneID
	current.PosX = posX
	current.PosY = posY
	r.players[playerID] = current
	return nil
}

func (r *PlayerRepository) AddExp(_ context.Context, playerID uint64, exp uint64) (*player.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.players[playerID]
	if !ok {
		return nil, player.ErrPlayerNotFound
	}
	current.Exp += exp
	r.players[playerID] = current
	copied := current
	return &copied, nil
}

// NewPetRepository returns the fixed starter pets used by battle, pet list,
// and lineup tests. Keeping this local avoids coupling transport tests to a DB.
func NewPetRepository() *PetRepository {
	return &PetRepository{
		nextID: 30000,
		definitions: map[uint32]pet.AdminPetDefinitionDetail{
			101: {
				PetID: 101, PetName: "小火龙", Description: "初始火系宠物", AcquireMethod: "新手赠送",
				IsEnabled: true, StatusText: "启用", SkinID: "嫩叶犬_001",
				BaseStats:       pet.AdminPetDefinitionBaseStats{Level: 1, Quality: 1, HP: 32, HPMax: 32, ATK: 14, DEF: 10, SPD: 12, MANA: 16},
				GrowthAptitudes: pet.AdminPetDefinitionGrowthAptitudes{HPApt: 12, ATKApt: 11, DEFApt: 10, SPDApt: 10, MANAApt: 9},
				SkillIDs:        []uint32{1001, 1002},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			102: {
				PetID: 102, PetName: "小水龟", Description: "初始水系宠物", AcquireMethod: "新手赠送",
				IsEnabled: true, StatusText: "启用", SkinID: "潮汐狐_001",
				BaseStats:       pet.AdminPetDefinitionBaseStats{Level: 1, Quality: 1, HP: 30, HPMax: 30, ATK: 12, DEF: 11, SPD: 9, MANA: 20},
				GrowthAptitudes: pet.AdminPetDefinitionGrowthAptitudes{HPApt: 11, ATKApt: 9, DEFApt: 12, SPDApt: 8, MANAApt: 13},
				SkillIDs:        []uint32{1001, 1003},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
		},
		pets: map[uint64][]pet.Pet{
			DemoPlayerID: {
				{PetUID: 20001, PetID: 101, Level: 5, Exp: 120, Quality: 1, HP: 32, HPMax: 32, ATK: 14, DEF: 10, SPD: 12, MANA: 16, SkillIDs: []uint32{1001, 1002}},
				{PetUID: 20002, PetID: 102, Level: 4, Exp: 80, Quality: 1, HP: 28, HPMax: 30, ATK: 12, DEF: 11, SPD: 9, MANA: 20, SkillIDs: []uint32{1001, 1003}},
				{PetUID: 20003, PetID: 101, Level: 3, Exp: 40, Quality: 1, HP: 24, HPMax: 24, ATK: 10, DEF: 8, SPD: 11, MANA: 12, SkillIDs: []uint32{1001}},
			},
			RivalPlayerID: {
				{PetUID: 21001, PetID: 101, Level: 5, Exp: 110, Quality: 1, HP: 31, HPMax: 31, ATK: 13, DEF: 10, SPD: 11, MANA: 15, SkillIDs: []uint32{1001, 1002}},
				{PetUID: 21002, PetID: 102, Level: 4, Exp: 75, Quality: 1, HP: 29, HPMax: 29, ATK: 11, DEF: 12, SPD: 10, MANA: 18, SkillIDs: []uint32{1001, 1003}},
			},
		},
		lineup: map[uint64][]pet.LineupPet{
			DemoPlayerID: {
				{PetUID: 20001, PetID: 101, Level: 5, HP: 32, HPMax: 32, ATK: 14, DEF: 10, SPD: 12, MANA: 16, SkillIDs: []uint32{1001, 1002}},
				{PetUID: 20002, PetID: 102, Level: 4, HP: 28, HPMax: 30, ATK: 12, DEF: 11, SPD: 9, MANA: 20, SkillIDs: []uint32{1001, 1003}},
			},
			RivalPlayerID: {
				{PetUID: 21001, PetID: 101, Level: 5, HP: 31, HPMax: 31, ATK: 13, DEF: 10, SPD: 11, MANA: 15, SkillIDs: []uint32{1001, 1002}},
				{PetUID: 21002, PetID: 102, Level: 4, HP: 29, HPMax: 29, ATK: 11, DEF: 12, SPD: 10, MANA: 18, SkillIDs: []uint32{1001, 1003}},
			},
		},
	}
}

type PetRepository struct {
	mu          sync.RWMutex
	pets        map[uint64][]pet.Pet
	lineup      map[uint64][]pet.LineupPet
	definitions map[uint32]pet.AdminPetDefinitionDetail
	nextID      uint64
}

func (r *PetRepository) ListPetsByPlayerID(_ context.Context, playerID uint64) ([]pet.Pet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items, ok := r.pets[playerID]
	if !ok {
		return []pet.Pet{}, nil
	}

	copied := make([]pet.Pet, 0, len(items))
	for _, item := range items {
		next := item
		if len(item.SkillIDs) > 0 {
			next.SkillIDs = append([]uint32{}, item.SkillIDs...)
		}
		copied = append(copied, next)
	}
	return copied, nil
}

func (r *PetRepository) ListLineupByPlayerID(_ context.Context, playerID uint64) ([]pet.LineupPet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items, ok := r.lineup[playerID]
	if !ok {
		return []pet.LineupPet{}, nil
	}
	copied := make([]pet.LineupPet, len(items))
	copy(copied, items)
	return copied, nil
}

func (r *PetRepository) SetLineupByPlayerID(_ context.Context, playerID uint64, petUIDs []uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	petsForPlayer, ok := r.pets[playerID]
	if !ok {
		return nil
	}

	byUID := make(map[uint64]pet.Pet, len(petsForPlayer))
	for _, item := range petsForPlayer {
		byUID[item.PetUID] = item
	}

	nextLineup := make([]pet.LineupPet, 0, len(petUIDs))
	for _, petUID := range petUIDs {
		item, exists := byUID[petUID]
		if !exists {
			return pet.ErrPetNotFound
		}
		nextLineup = append(nextLineup, pet.LineupPet{
			PetUID:   item.PetUID,
			PetID:    item.PetID,
			Level:    item.Level,
			HP:       item.HP,
			HPMax:    item.HPMax,
			ATK:      item.ATK,
			DEF:      item.DEF,
			SPD:      item.SPD,
			MANA:     item.MANA,
			SkillIDs: append([]uint32{}, item.SkillIDs...),
		})
	}

	r.lineup[playerID] = nextLineup
	return nil
}

func (r *PetRepository) UpdatePetHPByUID(_ context.Context, playerID uint64, petUID uint64, hp uint32) (pet.Pet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	petsForPlayer, ok := r.pets[playerID]
	if !ok {
		return pet.Pet{}, pet.ErrPetNotFound
	}

	for index := range petsForPlayer {
		if petsForPlayer[index].PetUID != petUID {
			continue
		}
		if hp > petsForPlayer[index].HPMax {
			hp = petsForPlayer[index].HPMax
		}
		petsForPlayer[index].HP = hp
		r.pets[playerID] = petsForPlayer

		lineup := r.lineup[playerID]
		for lineupIndex := range lineup {
			if lineup[lineupIndex].PetUID == petUID {
				lineup[lineupIndex].HP = hp
			}
		}
		r.lineup[playerID] = lineup

		updated := petsForPlayer[index]
		if len(updated.SkillIDs) > 0 {
			updated.SkillIDs = append([]uint32{}, updated.SkillIDs...)
		}
		return updated, nil
	}

	return pet.Pet{}, pet.ErrPetNotFound
}

func (r *PetRepository) UpdatePetHPAndExpByUID(_ context.Context, playerID uint64, petUID uint64, hp uint32, expGain uint64) (pet.Pet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	petsForPlayer, ok := r.pets[playerID]
	if !ok {
		return pet.Pet{}, pet.ErrPetNotFound
	}

	for index := range petsForPlayer {
		if petsForPlayer[index].PetUID != petUID {
			continue
		}
		if hp > petsForPlayer[index].HPMax {
			hp = petsForPlayer[index].HPMax
		}
		petsForPlayer[index].HP = hp
		petsForPlayer[index].Exp += expGain
		r.pets[playerID] = petsForPlayer

		lineup := r.lineup[playerID]
		for lineupIndex := range lineup {
			if lineup[lineupIndex].PetUID == petUID {
				lineup[lineupIndex].HP = hp
			}
		}
		r.lineup[playerID] = lineup

		updated := petsForPlayer[index]
		if len(updated.SkillIDs) > 0 {
			updated.SkillIDs = append([]uint32{}, updated.SkillIDs...)
		}
		return updated, nil
	}

	return pet.Pet{}, pet.ErrPetNotFound
}

func (r *PetRepository) GrantRuntimePet(_ context.Context, playerID uint64, petID uint32, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*pet.RuntimeGrantResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	template := pet.Pet{
		PetID:    petID,
		Level:    1,
		Quality:  1,
		HP:       20,
		HPMax:    20,
		ATK:      8,
		DEF:      7,
		SPD:      6,
		MANA:     10,
		SkillIDs: []uint32{1001},
		InLineup: false,
	}
	switch petID {
	case 101:
		template.Level = 5
		template.HP = 32
		template.HPMax = 32
		template.ATK = 14
		template.DEF = 10
		template.SPD = 12
		template.MANA = 16
		template.SkillIDs = []uint32{1001, 1002}
	case 102:
		template.Level = 4
		template.HP = 28
		template.HPMax = 30
		template.ATK = 12
		template.DEF = 11
		template.SPD = 9
		template.MANA = 20
		template.SkillIDs = []uint32{1001, 1003}
	}
	template.PetUID = r.nextID
	r.nextID++
	r.pets[playerID] = append(r.pets[playerID], template)
	_ = reasonType
	_ = reasonRefID
	_ = operatorType
	_ = operatorID
	copyValue := template
	if len(copyValue.SkillIDs) > 0 {
		copyValue.SkillIDs = append([]uint32{}, copyValue.SkillIDs...)
	}
	return &pet.RuntimeGrantResult{Pet: copyValue}, nil
}

func (r *PetRepository) GrantWildCapturePet(_ context.Context, playerID uint64, petID uint32, captureMonsterID uint32, reasonType string, reasonRefID uint64) (*pet.RuntimeGrantResult, error) {
	result, err := r.GrantRuntimePet(context.Background(), playerID, petID, reasonType, reasonRefID, "", 0)
	if err != nil {
		return nil, err
	}
	result.Pet.GrantSource = pet.GrantSourceWildCapture
	result.Pet.CaptureMonsterID = captureMonsterID
	result.Pet.GrowthAptitudes = pet.RollWildCaptureAptitudes(pet.AptitudeRollRanges{
		HPAptMin: 8, HPAptMax: 14,
		ATKAptMin: 8, ATKAptMax: 13,
		DEFAptMin: 8, DEFAptMax: 12,
		SPDAptMin: 7, SPDAptMax: 12,
		MANAAptMin: 6, MANAAptMax: 11,
	}, nil)
	return result, nil
}

func (r *PetRepository) ListForAdmin(_ context.Context, query pet.AdminListQuery) (*pet.AdminPetList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]pet.AdminPetSummary, 0)
	for playerID, petsForPlayer := range r.pets {
		for _, item := range petsForPlayer {
			if query.PetUID > 0 && item.PetUID != query.PetUID {
				continue
			}
			if query.PlayerID > 0 && playerID != query.PlayerID {
				continue
			}
			if query.PetID > 0 && item.PetID != query.PetID {
				continue
			}
			items = append(items, pet.AdminPetSummary{
				PetUID:     item.PetUID,
				PlayerID:   playerID,
				PlayerName: fmt.Sprintf("Player%d", playerID),
				PetID:      item.PetID,
				Level:      item.Level,
				Quality:    item.Quality,
				HP:         item.HP,
				HPMax:      item.HPMax,
				ATK:        item.ATK,
				DEF:        item.DEF,
				SPD:        item.SPD,
				MANA:       item.MANA,
				InLineup:   item.InLineup,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})
		}
	}
	return &pet.AdminPetList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *PetRepository) FindAdminDetailByPetUID(_ context.Context, petUID uint64) (*pet.AdminPetDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for playerID, petsForPlayer := range r.pets {
		for _, item := range petsForPlayer {
			if item.PetUID != petUID {
				continue
			}
			return &pet.AdminPetDetail{
				PetUID:     item.PetUID,
				PlayerID:   playerID,
				PlayerName: fmt.Sprintf("Player%d", playerID),
				PetID:      item.PetID,
				Level:      item.Level,
				Exp:        item.Exp,
				Quality:    item.Quality,
				HP:         item.HP,
				HPMax:      item.HPMax,
				ATK:        item.ATK,
				DEF:        item.DEF,
				SPD:        item.SPD,
				MANA:       item.MANA,
				SkillIDs:   append([]uint32{}, item.SkillIDs...),
				InLineup:   item.InLineup,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}, nil
		}
	}
	return nil, nil
}

func (r *PetRepository) CreateForAdmin(_ context.Context, input pet.AdminCreatePetInput) (*pet.AdminPetDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	input = input.Normalize()
	item := pet.Pet{
		PetUID:   r.nextID,
		PetID:    input.PetID,
		Level:    input.Level,
		Exp:      input.Exp,
		Quality:  input.Quality,
		HP:       input.HP,
		HPMax:    input.HPMax,
		ATK:      input.ATK,
		DEF:      input.DEF,
		SPD:      input.SPD,
		MANA:     input.MANA,
		SkillIDs: append([]uint32{}, input.SkillIDs...),
	}
	r.nextID++
	r.pets[input.PlayerID] = append(r.pets[input.PlayerID], item)
	return &pet.AdminPetDetail{
		PetUID:     item.PetUID,
		PlayerID:   input.PlayerID,
		PlayerName: fmt.Sprintf("Player%d", input.PlayerID),
		PetID:      item.PetID,
		Level:      item.Level,
		Exp:        item.Exp,
		Quality:    item.Quality,
		HP:         item.HP,
		HPMax:      item.HPMax,
		ATK:        item.ATK,
		DEF:        item.DEF,
		SPD:        item.SPD,
		MANA:       item.MANA,
		SkillIDs:   append([]uint32{}, item.SkillIDs...),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

func (r *PetRepository) UpdateForAdmin(_ context.Context, petUID uint64, input pet.AdminUpdatePetInput) (*pet.AdminPetDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for playerID, petsForPlayer := range r.pets {
		for index := range petsForPlayer {
			if petsForPlayer[index].PetUID != petUID {
				continue
			}
			input = input.Normalize()
			petsForPlayer[index].PetID = input.PetID
			petsForPlayer[index].Level = input.Level
			petsForPlayer[index].Exp = input.Exp
			petsForPlayer[index].Quality = input.Quality
			petsForPlayer[index].HP = input.HP
			petsForPlayer[index].HPMax = input.HPMax
			petsForPlayer[index].ATK = input.ATK
			petsForPlayer[index].DEF = input.DEF
			petsForPlayer[index].SPD = input.SPD
			petsForPlayer[index].MANA = input.MANA
			petsForPlayer[index].SkillIDs = append([]uint32{}, input.SkillIDs...)
			r.pets[playerID] = petsForPlayer
			item := petsForPlayer[index]
			return &pet.AdminPetDetail{
				PetUID:     item.PetUID,
				PlayerID:   playerID,
				PlayerName: fmt.Sprintf("Player%d", playerID),
				PetID:      item.PetID,
				Level:      item.Level,
				Exp:        item.Exp,
				Quality:    item.Quality,
				HP:         item.HP,
				HPMax:      item.HPMax,
				ATK:        item.ATK,
				DEF:        item.DEF,
				SPD:        item.SPD,
				MANA:       item.MANA,
				SkillIDs:   append([]uint32{}, item.SkillIDs...),
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}, nil
		}
	}
	return nil, pet.ErrPetNotFound
}

func (r *PetRepository) DeleteForAdmin(_ context.Context, petUID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for playerID, petsForPlayer := range r.pets {
		for index := range petsForPlayer {
			if petsForPlayer[index].PetUID != petUID {
				continue
			}
			r.pets[playerID] = append(petsForPlayer[:index], petsForPlayer[index+1:]...)
			lineup := r.lineup[playerID]
			nextLineup := make([]pet.LineupPet, 0, len(lineup))
			for _, item := range lineup {
				if item.PetUID != petUID {
					nextLineup = append(nextLineup, item)
				}
			}
			r.lineup[playerID] = nextLineup
			return nil
		}
	}
	return pet.ErrPetNotFound
}

func (r *PetRepository) ListPetDefinitionsForAdmin(_ context.Context, query pet.AdminPetDefinitionListQuery) (*pet.AdminPetDefinitionList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]pet.AdminPetDefinitionSummary, 0, len(r.definitions))
	for _, current := range r.definitions {
		if query.PetID > 0 && current.PetID != query.PetID {
			continue
		}
		if query.Name != "" && !strings.Contains(current.PetName, query.Name) {
			continue
		}
		if query.Enabled != nil && current.IsEnabled != *query.Enabled {
			continue
		}
		items = append(items, pet.AdminPetDefinitionSummary{
			PetID:         current.PetID,
			PetName:       current.PetName,
			Quality:       current.BaseStats.Quality,
			Level:         current.BaseStats.Level,
			AcquireMethod: current.AcquireMethod,
			IsEnabled:     current.IsEnabled,
			StatusText:    current.StatusText,
			SkinID:        current.SkinID,
			CreatedAt:     current.CreatedAt,
			UpdatedAt:     current.UpdatedAt,
		})
	}
	return &pet.AdminPetDefinitionList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *PetRepository) FindPetDefinitionForAdmin(_ context.Context, petID uint32) (*pet.AdminPetDefinitionDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current, ok := r.definitions[petID]
	if !ok {
		return nil, nil
	}
	copied := current
	if len(current.SkillIDs) > 0 {
		copied.SkillIDs = append([]uint32{}, current.SkillIDs...)
	}
	return &copied, nil
}

func (r *PetRepository) CreatePetDefinitionForAdmin(_ context.Context, input pet.AdminUpsertPetDefinitionInput) (*pet.AdminPetDefinitionDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.definitions[input.PetID]; exists {
		return nil, pet.ErrPetDefinitionConflict
	}
	now := time.Now()
	detail := buildStubPetDefinitionDetail(input, now)
	r.definitions[input.PetID] = detail
	return r.findPetDefinitionLocked(input.PetID)
}

func (r *PetRepository) UpdatePetDefinitionForAdmin(_ context.Context, petID uint32, input pet.AdminUpsertPetDefinitionInput) (*pet.AdminPetDefinitionDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.definitions[petID]
	if !ok {
		return nil, nil
	}
	updated := buildStubPetDefinitionDetail(input, current.CreatedAt)
	updated.PetID = petID
	updated.UpdatedAt = time.Now()
	r.definitions[petID] = updated
	return r.findPetDefinitionLocked(petID)
}

func (r *PetRepository) DeletePetDefinitionForAdmin(_ context.Context, petID uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.definitions[petID]; !ok {
		return pet.ErrPetDefinitionNotFound
	}
	delete(r.definitions, petID)
	return nil
}

func (r *PetRepository) MapUsablePetDefinitionIDs(_ context.Context, petIDs []uint32) (map[uint32]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[uint32]bool, len(petIDs))
	for _, petID := range petIDs {
		current, ok := r.definitions[petID]
		if ok && current.IsEnabled {
			result[petID] = true
		}
	}
	return result, nil
}

func (r *PetRepository) FindPetSkinID(_ context.Context, petID uint32) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current, ok := r.definitions[petID]
	if !ok || !current.IsEnabled {
		return "", nil
	}
	return current.SkinID, nil
}

func (r *PetRepository) findPetDefinitionLocked(petID uint32) (*pet.AdminPetDefinitionDetail, error) {
	current, ok := r.definitions[petID]
	if !ok {
		return nil, nil
	}
	copied := current
	if len(current.SkillIDs) > 0 {
		copied.SkillIDs = append([]uint32{}, current.SkillIDs...)
	}
	return &copied, nil
}

func buildStubPetDefinitionDetail(input pet.AdminUpsertPetDefinitionInput, createdAt time.Time) pet.AdminPetDefinitionDetail {
	statusText := "停用"
	if input.IsEnabled {
		statusText = "启用"
	}
	skillIDs := append([]uint32{}, input.SkillIDs...)
	return pet.AdminPetDefinitionDetail{
		PetID:         input.PetID,
		PetName:       input.PetName,
		Description:   input.Description,
		AcquireMethod: input.AcquireMethod,
		IsEnabled:     input.IsEnabled,
		StatusText:    statusText,
		SkinID:        input.SkinID,
		BaseStats: pet.AdminPetDefinitionBaseStats{
			Level: input.Level, Quality: input.Quality, HP: input.HP, HPMax: input.HPMax,
			ATK: input.ATK, DEF: input.DEF, SPD: input.SPD, MANA: input.MANA,
		},
		GrowthAptitudes: pet.AdminPetDefinitionGrowthAptitudes{
			HPApt: input.HPApt, ATKApt: input.ATKApt, DEFApt: input.DEFApt, SPDApt: input.SPDApt, MANAApt: input.MANAApt,
		},
		AptitudeRollRanges: pet.AdminPetDefinitionAptitudeRollRanges{
			HPAptRollMin: input.HPAptRollMin, HPAptRollMax: input.HPAptRollMax,
			ATKAptRollMin: input.ATKAptRollMin, ATKAptRollMax: input.ATKAptRollMax,
			DEFAptRollMin: input.DEFAptRollMin, DEFAptRollMax: input.DEFAptRollMax,
			SPDAptRollMin: input.SPDAptRollMin, SPDAptRollMax: input.SPDAptRollMax,
			MANAAptRollMin: input.MANAAptRollMin, MANAAptRollMax: input.MANAAptRollMax,
		},
		SkillIDs:  skillIDs,
		CreatedAt: createdAt,
		UpdatedAt: time.Now(),
	}
}

// NewBagRepository 提供后台背包与仓库 CRUD 的内存桩，避免 HTTP 测试依赖真实 PostgreSQL。
func NewBagRepository() *BagRepository {
	now := time.Now()
	return &BagRepository{
		nextID: 40000,
		capacities: map[uint64]map[string]uint32{
			DemoPlayerID:  {bag.ContainerTypeBag: 30, bag.ContainerTypeWarehouse: 30},
			RivalPlayerID: {bag.ContainerTypeBag: 30, bag.ContainerTypeWarehouse: 30},
		},
		items: map[uint64]bag.AdminItemDetail{
			30001: {RecordID: 30001, PlayerID: DemoPlayerID, PlayerName: "DemoTrainer", ContainerType: "bag", SlotIndex: 1, ItemID: 3003, ItemName: "宠物治疗药剂", ItemType: "consumable", Quantity: 3, CreatedAt: now, UpdatedAt: now},
			30004: {RecordID: 30004, PlayerID: DemoPlayerID, PlayerName: "DemoTrainer", ContainerType: "bag", SlotIndex: 4, ItemID: 3004, ItemName: "新手补给礼包", ItemType: "box", Quantity: 1, CreatedAt: now, UpdatedAt: now},
			30003: {RecordID: 30003, PlayerID: DemoPlayerID, PlayerName: "DemoTrainer", ContainerType: "bag", SlotIndex: 3, ItemID: 3001, ItemName: "背包扩容券", ItemType: "functional", Quantity: 2, CreatedAt: now, UpdatedAt: now},
			30002: {RecordID: 30002, PlayerID: RivalPlayerID, PlayerName: "RivalTrainer", ContainerType: "warehouse", SlotIndex: 2, ItemID: 2002, ItemName: "训练护腕", ItemType: "equipment", ItemUID: "eq_rival_1", Quantity: 1, IsBound: true, CreatedAt: now, UpdatedAt: now},
		},
	}
}

type BagRepository struct {
	mu         sync.RWMutex
	capacities map[uint64]map[string]uint32
	items      map[uint64]bag.AdminItemDetail
	petRepo    *PetRepository
	nextID     uint64
}

// BindPetRepository 让背包道具使用和宠物查询共用同一份内存宠物状态。
// 这样测试里的宠物治疗药剂在扣道具后，`PET_LIST` 读取到的也是同一份最新数据。
func (r *BagRepository) BindPetRepository(petRepo *PetRepository) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.petRepo = petRepo
}

func (r *BagRepository) ListForAdmin(_ context.Context, query bag.AdminListQuery) (*bag.AdminItemList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]bag.AdminItemSummary, 0, len(r.items))
	for _, current := range r.items {
		if query.RecordID > 0 && current.RecordID != query.RecordID {
			continue
		}
		if query.PlayerID > 0 && current.PlayerID != query.PlayerID {
			continue
		}
		if query.ContainerType != "" && current.ContainerType != query.ContainerType {
			continue
		}
		if query.ItemID > 0 && current.ItemID != query.ItemID {
			continue
		}
		if query.ItemUID != "" && current.ItemUID != query.ItemUID {
			continue
		}
		items = append(items, bag.AdminItemSummary{
			RecordID: current.RecordID, PlayerID: current.PlayerID, PlayerName: current.PlayerName,
			ContainerType: current.ContainerType, SlotIndex: current.SlotIndex, ItemID: current.ItemID, ItemUID: current.ItemUID,
			ItemName: current.ItemName, ItemType: current.ItemType, Quantity: current.Quantity, IsBound: current.IsBound,
			CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt,
		})
	}
	return &bag.AdminItemList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *BagRepository) FindAdminDetailByRecordID(_ context.Context, recordID uint64) (*bag.AdminItemDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.items[recordID]
	if !ok {
		return nil, nil
	}
	copied := current
	return &copied, nil
}

func (r *BagRepository) CreateForAdmin(_ context.Context, input bag.AdminCreateItemInput) (*bag.AdminItemDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, current := range r.items {
		if current.PlayerID == input.PlayerID && current.ContainerType == input.ContainerType && current.SlotIndex == input.SlotIndex {
			return nil, bag.ErrBagItemConflict
		}
	}
	recordID := r.nextID
	r.nextID++
	now := time.Now()
	itemValue := bag.AdminItemDetail{
		RecordID: recordID, PlayerID: input.PlayerID, PlayerName: bagPlayerName(input.PlayerID), ContainerType: input.ContainerType,
		SlotIndex: input.SlotIndex, ItemID: input.ItemID, ItemUID: input.ItemUID, ItemName: fmt.Sprintf("Item%d", input.ItemID),
		ItemType: "consumable", Quantity: input.Quantity, IsBound: input.IsBound, CreatedAt: now, UpdatedAt: now,
	}
	r.items[recordID] = itemValue
	copied := itemValue
	return &copied, nil
}

func (r *BagRepository) UpdateForAdmin(_ context.Context, recordID uint64, input bag.AdminUpdateItemInput) (*bag.AdminItemDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[recordID]
	if !ok {
		return nil, bag.ErrBagItemNotFound
	}
	for otherID, other := range r.items {
		if otherID != recordID && other.PlayerID == input.PlayerID && other.ContainerType == input.ContainerType && other.SlotIndex == input.SlotIndex {
			return nil, bag.ErrBagItemConflict
		}
	}
	current.PlayerID = input.PlayerID
	current.PlayerName = bagPlayerName(input.PlayerID)
	current.ContainerType = input.ContainerType
	current.SlotIndex = input.SlotIndex
	current.ItemID = input.ItemID
	current.ItemUID = input.ItemUID
	current.ItemName = fmt.Sprintf("Item%d", input.ItemID)
	current.Quantity = input.Quantity
	current.IsBound = input.IsBound
	current.UpdatedAt = time.Now()
	r.items[recordID] = current
	copied := current
	return &copied, nil
}

func (r *BagRepository) DeleteForAdmin(_ context.Context, recordID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[recordID]; !ok {
		return bag.ErrBagItemNotFound
	}
	delete(r.items, recordID)
	return nil
}

func (r *BagRepository) ListRuntimeContainer(_ context.Context, playerID uint64, containerType string) (*bag.RuntimeContainerSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedContainerType, err := bag.NormalizeRuntimeContainerType(containerType)
	if err != nil {
		return nil, err
	}
	items := make([]bag.RuntimeItemSnapshot, 0)
	for _, current := range r.items {
		if current.PlayerID != playerID || current.ContainerType != normalizedContainerType {
			continue
		}
		items = append(items, bag.RuntimeItemSnapshot{
			SlotIndex:    current.SlotIndex,
			ItemID:       current.ItemID,
			ItemUID:      current.ItemUID,
			Quantity:     current.Quantity,
			IsBound:      current.IsBound,
			ItemName:     current.ItemName,
			ItemType:     current.ItemType,
			ItemSubType:  "",
			Quality:      1,
			Icon:         "",
			EnhanceLevel: 0,
		})
	}
	return &bag.RuntimeContainerSnapshot{
		ContainerType: normalizedContainerType,
		Capacity:      r.containerCapacity(playerID, normalizedContainerType),
		MaxCapacity:   300,
		UsedSlots:     uint32(len(items)),
		Items:         items,
	}, nil
}

func (r *BagRepository) TransferRuntimeItem(_ context.Context, playerID uint64, fromContainerType string, toContainerType string, fromSlotIndex uint32, quantity uint64) (*bag.RuntimeTransferResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var (
		recordID uint64
		source   bag.AdminItemDetail
		found    bool
	)
	for currentRecordID, current := range r.items {
		if current.PlayerID == playerID && current.ContainerType == fromContainerType && current.SlotIndex == fromSlotIndex {
			recordID = currentRecordID
			source = current
			found = true
			break
		}
	}
	if !found {
		return nil, bag.ErrContainerItemNotFound
	}
	if quantity == 0 || quantity > source.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}
	if toContainerType == bag.ContainerTypeWarehouse && source.ItemID == 0 {
		return nil, bag.ErrItemCannotStore
	}

	targetSlotIndex := uint32(1)
	occupied := map[uint32]bool{}
	for _, current := range r.items {
		if current.PlayerID == playerID && current.ContainerType == toContainerType {
			occupied[current.SlotIndex] = true
		}
	}
	for occupied[targetSlotIndex] {
		targetSlotIndex++
		if targetSlotIndex > 300 {
			return nil, bag.ErrContainerCapacityFull
		}
	}

	now := time.Now()
	targetRecordID := r.nextID
	r.nextID++
	r.items[targetRecordID] = bag.AdminItemDetail{
		RecordID:      targetRecordID,
		PlayerID:      playerID,
		PlayerName:    bagPlayerName(playerID),
		ContainerType: toContainerType,
		SlotIndex:     targetSlotIndex,
		ItemID:        source.ItemID,
		ItemUID:       source.ItemUID,
		ItemName:      source.ItemName,
		ItemType:      source.ItemType,
		Quantity:      quantity,
		IsBound:       source.IsBound,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if quantity == source.Quantity {
		delete(r.items, recordID)
	} else {
		source.Quantity -= quantity
		source.UpdatedAt = now
		r.items[recordID] = source
	}

	return &bag.RuntimeTransferResult{
		MovedItemID:       source.ItemID,
		MovedItemUID:      source.ItemUID,
		MovedQuantity:     quantity,
		FromContainerType: fromContainerType,
		ToContainerType:   toContainerType,
		FromSlotIndex:     fromSlotIndex,
		ToSlotIndex:       targetSlotIndex,
	}, nil
}

func (r *BagRepository) SortRuntimeContainer(_ context.Context, playerID uint64, containerType string) (*bag.RuntimeSortResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]bag.AdminItemDetail, 0)
	for _, current := range r.items {
		if current.PlayerID == playerID && current.ContainerType == containerType {
			items = append(items, current)
		}
	}
	sort.SliceStable(items, func(left int, right int) bool {
		if items[left].ItemID != items[right].ItemID {
			return items[left].ItemID < items[right].ItemID
		}
		return items[left].SlotIndex < items[right].SlotIndex
	})
	for index, current := range items {
		current.SlotIndex = uint32(index + 1)
		current.UpdatedAt = time.Now()
		for recordID, saved := range r.items {
			if saved.RecordID == current.RecordID {
				r.items[recordID] = current
				break
			}
		}
	}
	return &bag.RuntimeSortResult{ContainerType: containerType, Sorted: true}, nil
}

func (r *BagRepository) MoveRuntimeItem(_ context.Context, playerID uint64, containerType string, fromSlotIndex uint32, toSlotIndex uint32, quantity uint64) (*bag.RuntimeMoveResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var (
		sourceRecordID uint64
		targetRecordID uint64
		source         bag.AdminItemDetail
		target         bag.AdminItemDetail
		sourceFound    bool
		targetFound    bool
	)
	for recordID, current := range r.items {
		if current.PlayerID != playerID || current.ContainerType != containerType {
			continue
		}
		if current.SlotIndex == fromSlotIndex {
			sourceRecordID = recordID
			source = current
			sourceFound = true
		}
		if current.SlotIndex == toSlotIndex {
			targetRecordID = recordID
			target = current
			targetFound = true
		}
	}
	if !sourceFound || quantity == 0 || quantity > source.Quantity || fromSlotIndex == toSlotIndex {
		return nil, bag.ErrInvalidContainerMove
	}
	now := time.Now()
	if !targetFound {
		if quantity == source.Quantity {
			source.SlotIndex = toSlotIndex
			source.UpdatedAt = now
			r.items[sourceRecordID] = source
		} else {
			source.Quantity -= quantity
			source.UpdatedAt = now
			r.items[sourceRecordID] = source
			recordID := r.nextID
			r.nextID++
			r.items[recordID] = bag.AdminItemDetail{
				RecordID:      recordID,
				PlayerID:      playerID,
				PlayerName:    bagPlayerName(playerID),
				ContainerType: containerType,
				SlotIndex:     toSlotIndex,
				ItemID:        source.ItemID,
				ItemUID:       source.ItemUID,
				ItemName:      source.ItemName,
				ItemType:      source.ItemType,
				Quantity:      quantity,
				IsBound:       source.IsBound,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
		}
	} else {
		if quantity != source.Quantity {
			return nil, bag.ErrInvalidContainerMove
		}
		source.SlotIndex = toSlotIndex
		source.UpdatedAt = now
		target.SlotIndex = fromSlotIndex
		target.UpdatedAt = now
		r.items[sourceRecordID] = source
		r.items[targetRecordID] = target
	}
	return &bag.RuntimeMoveResult{
		ContainerType: containerType,
		FromSlotIndex: fromSlotIndex,
		ToSlotIndex:   toSlotIndex,
		Moved:         true,
	}, nil
}

func (r *BagRepository) GrantRuntimeItem(_ context.Context, playerID uint64, containerType string, itemID uint64, quantity uint64, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*bag.RuntimeGrantResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if quantity == 0 {
		return nil, bag.ErrInvalidTransferQuantity
	}
	occupied := map[uint32]bool{}
	var mergeRecordID uint64
	var mergeItem bag.AdminItemDetail
	for recordID, current := range r.items {
		if current.PlayerID != playerID || current.ContainerType != containerType {
			continue
		}
		occupied[current.SlotIndex] = true
		if current.ItemID == itemID && current.ItemUID == "" {
			mergeRecordID = recordID
			mergeItem = current
		}
	}
	now := time.Now()
	if mergeRecordID != 0 {
		mergeItem.Quantity += quantity
		mergeItem.UpdatedAt = now
		r.items[mergeRecordID] = mergeItem
		return &bag.RuntimeGrantResult{
			ContainerType: containerType,
			ItemID:        itemID,
			ItemName:      mergeItem.ItemName,
			ItemUID:       "",
			GrantedQty:    quantity,
			SlotIndex:     mergeItem.SlotIndex,
		}, nil
	}
	slotIndex := uint32(1)
	for occupied[slotIndex] {
		slotIndex++
		if slotIndex > 300 {
			return nil, bag.ErrContainerCapacityFull
		}
	}
	recordID := r.nextID
	r.nextID++
	itemName := fmt.Sprintf("Item%d", itemID)
	itemType := "consumable"
	if itemID == 2001 {
		itemName = "新手精灵球"
	}
	r.items[recordID] = bag.AdminItemDetail{
		RecordID:      recordID,
		PlayerID:      playerID,
		PlayerName:    bagPlayerName(playerID),
		ContainerType: containerType,
		SlotIndex:     slotIndex,
		ItemID:        itemID,
		ItemUID:       "",
		ItemName:      itemName,
		ItemType:      itemType,
		Quantity:      quantity,
		IsBound:       false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return &bag.RuntimeGrantResult{
		ContainerType: containerType,
		ItemID:        itemID,
		ItemName:      itemName,
		ItemUID:       "",
		GrantedQty:    quantity,
		SlotIndex:     slotIndex,
	}, nil
}

func (r *BagRepository) UseRuntimeItem(_ context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, targetPetUID uint64, targetPlayerID uint64) (*bag.RuntimeUseResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if quantity == 0 {
		return nil, bag.ErrInvalidTransferQuantity
	}
	var (
		recordID uint64
		source   bag.AdminItemDetail
		found    bool
	)
	for currentRecordID, current := range r.items {
		if current.PlayerID == playerID && current.ContainerType == containerType && current.SlotIndex == slotIndex {
			recordID = currentRecordID
			source = current
			found = true
			break
		}
	}
	if !found {
		return nil, bag.ErrContainerItemNotFound
	}
	if quantity > source.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}

	var (
		effectType   string
		expandTarget string
		expandSlots  uint32
	)
	switch source.ItemID {
	case 3001:
		effectType = "bag_expand"
		expandTarget = bag.ContainerTypeBag
		expandSlots = 5
	case 3002:
		effectType = "warehouse_expand"
		expandTarget = bag.ContainerTypeWarehouse
		expandSlots = 5
	case 3003:
		if targetPlayerID != 0 && targetPlayerID != playerID {
			return nil, bag.ErrUseTargetNotFound
		}
		if targetPetUID == 0 {
			return nil, bag.ErrUseTargetRequired
		}
		if r.petRepo == nil {
			return nil, bag.ErrUseTargetNotFound
		}
		r.petRepo.mu.Lock()
		defer r.petRepo.mu.Unlock()
		petsForPlayer, ok := r.petRepo.pets[playerID]
		if !ok {
			return nil, bag.ErrUseTargetNotFound
		}
		var (
			targetPet *pet.Pet
			petIndex  int
		)
		for index := range petsForPlayer {
			if petsForPlayer[index].PetUID == targetPetUID {
				targetPet = &petsForPlayer[index]
				petIndex = index
				break
			}
		}
		if targetPet == nil {
			return nil, bag.ErrUseTargetNotFound
		}
		if targetPet.HP >= targetPet.HPMax {
			return nil, bag.ErrItemUseNoEffect
		}
		restoreAmount := uint32(quantity * 10)
		nextHP := targetPet.HP + restoreAmount
		if nextHP > targetPet.HPMax {
			nextHP = targetPet.HPMax
		}
		restoredHP := nextHP - targetPet.HP
		petsForPlayer[petIndex].HP = nextHP
		r.petRepo.pets[playerID] = petsForPlayer
		lineup := r.petRepo.lineup[playerID]
		inLineup := false
		for lineupIndex := range lineup {
			if lineup[lineupIndex].PetUID == targetPetUID {
				lineup[lineupIndex].HP = nextHP
				inLineup = true
			}
		}
		r.petRepo.lineup[playerID] = lineup
		updatedPet := petsForPlayer[petIndex]
		if len(updatedPet.SkillIDs) > 0 {
			updatedPet.SkillIDs = append([]uint32{}, updatedPet.SkillIDs...)
		}
		now := time.Now()
		if quantity == source.Quantity {
			delete(r.items, recordID)
		} else {
			source.Quantity -= quantity
			source.UpdatedAt = now
			r.items[recordID] = source
		}
		return &bag.RuntimeUseResult{
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ItemID:        source.ItemID,
			UsedQuantity:  quantity,
			Result: bag.RuntimeUseEffect{
				EffectType:   "pet_hp_restore",
				TargetPetUID: targetPetUID,
				RestoredHP:   restoredHP,
				NewPetHP:     nextHP,
				UpdatedPet: &bag.RuntimePetSnapshot{
					PetUID:   updatedPet.PetUID,
					PetID:    updatedPet.PetID,
					Level:    updatedPet.Level,
					Exp:      updatedPet.Exp,
					Quality:  updatedPet.Quality,
					HP:       updatedPet.HP,
					HPMax:    updatedPet.HPMax,
					ATK:      updatedPet.ATK,
					DEF:      updatedPet.DEF,
					SPD:      updatedPet.SPD,
					SkillIDs: updatedPet.SkillIDs,
					InLineup: inLineup,
				},
			},
		}, nil
	case 3004:
		now := time.Now()
		if quantity == source.Quantity {
			delete(r.items, recordID)
		} else {
			source.Quantity -= quantity
			source.UpdatedAt = now
			r.items[recordID] = source
		}
		return &bag.RuntimeUseResult{
			ContainerType: containerType,
			SlotIndex:     slotIndex,
			ItemID:        source.ItemID,
			UsedQuantity:  quantity,
			Result: bag.RuntimeUseEffect{
				EffectType: "reward_box",
				Rewards: []bag.RuntimeRewardItem{
					{
						Type:  "gold",
						Value: 2 * quantity,
					},
					{
						Type:     "item",
						ItemID:   2001,
						ItemName: "新手精灵球",
						Count:    quantity,
					},
				},
			},
		}, nil
	default:
		return nil, bag.ErrItemNotUsable
	}

	currentCapacity := r.containerCapacity(playerID, expandTarget)
	totalExpand := expandSlots * uint32(quantity)
	nextCapacity := currentCapacity + totalExpand
	if nextCapacity > 300 {
		return nil, bag.ErrContainerCapacityLimit
	}
	r.ensureContainerCapacityMap(playerID)
	r.capacities[playerID][expandTarget] = nextCapacity

	now := time.Now()
	if quantity == source.Quantity {
		delete(r.items, recordID)
	} else {
		source.Quantity -= quantity
		source.UpdatedAt = now
		r.items[recordID] = source
	}

	return &bag.RuntimeUseResult{
		ContainerType: containerType,
		SlotIndex:     slotIndex,
		ItemID:        source.ItemID,
		UsedQuantity:  quantity,
		Result: bag.RuntimeUseEffect{
			EffectType:   effectType,
			ExpandTarget: expandTarget,
			ExpandSlots:  totalExpand,
			NewCapacity:  nextCapacity,
		},
	}, nil
}

func (r *BagRepository) ConsumeRuntimeItemStack(_ context.Context, playerID uint64, containerType string, slotIndex uint32, quantity uint64, reasonType string, reasonRefID uint64) (*bag.RuntimeContainerSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if quantity == 0 {
		return nil, bag.ErrInvalidTransferQuantity
	}
	var (
		recordID uint64
		source   bag.AdminItemDetail
		found    bool
	)
	for currentRecordID, current := range r.items {
		if current.PlayerID == playerID && current.ContainerType == containerType && current.SlotIndex == slotIndex {
			recordID = currentRecordID
			source = current
			found = true
			break
		}
	}
	if !found {
		return nil, bag.ErrContainerItemNotFound
	}
	if quantity > source.Quantity {
		return nil, bag.ErrInvalidTransferQuantity
	}
	if quantity == source.Quantity {
		delete(r.items, recordID)
	} else {
		source.Quantity -= quantity
		source.UpdatedAt = time.Now()
		r.items[recordID] = source
	}
	_ = reasonType
	_ = reasonRefID
	return r.ListRuntimeContainer(context.Background(), playerID, containerType)
}

func bagPlayerName(playerID uint64) string {
	switch playerID {
	case DemoPlayerID:
		return "DemoTrainer"
	case RivalPlayerID:
		return "RivalTrainer"
	default:
		return fmt.Sprintf("Player%d", playerID)
	}
}

func (r *BagRepository) ensureContainerCapacityMap(playerID uint64) {
	if r.capacities == nil {
		r.capacities = map[uint64]map[string]uint32{}
	}
	if _, ok := r.capacities[playerID]; !ok {
		r.capacities[playerID] = map[string]uint32{
			bag.ContainerTypeBag:       30,
			bag.ContainerTypeWarehouse: 30,
		}
	}
}

func (r *BagRepository) containerCapacity(playerID uint64, containerType string) uint32 {
	r.ensureContainerCapacityMap(playerID)
	if value, ok := r.capacities[playerID][containerType]; ok && value > 0 {
		return value
	}
	return 30
}

// NewItemRepository 提供后台物品模板 CRUD 的内存桩。
func NewItemRepository() *ItemRepository {
	now := time.Now()
	return &ItemRepository{
		items: map[uint64]item.AdminItemDetail{
			2001: {ItemID: 2001, ItemCode: "starter_capture_ball", ItemName: "新手精灵球", ItemType: "consumable", ItemSubType: "capture", Quality: 1, Rarity: 1, MaxStack: 99, OccupySlots: 1, AutoMerge: true, CanSell: true, CanDrop: true, CanStore: true, PriceType: "base_coin", BuyPriceCopper: 1000, SellPriceCopper: 200, EffectParamsJSON: "{}", BindType: "none", IsEnabled: true, CreatedAt: now, UpdatedAt: now},
			1001: {ItemID: 1001, ItemCode: "hp_potion_small", ItemName: "小型生命药剂", ItemType: "consumable", ItemSubType: "hp_potion", Quality: 1, Rarity: 1, MaxStack: 99, OccupySlots: 1, AutoMerge: true, CanSell: true, CanDrop: true, CanStore: true, PriceType: "base_coin", BuyPriceCopper: 500, SellPriceCopper: 100, EffectParamsJSON: "{}", BindType: "none", IsEnabled: true, CreatedAt: now, UpdatedAt: now},
			2002: {ItemID: 2002, ItemCode: "training_bracer", ItemName: "训练护腕", ItemType: "equipment", ItemSubType: "armor", Quality: 2, Rarity: 2, MaxStack: 1, OccupySlots: 1, AutoMerge: false, CanSell: true, CanDrop: false, CanStore: true, PriceType: "base_coin", SellPriceCopper: 1200, EffectParamsJSON: "{}", BindType: "pickup_bind", IsEnabled: true, CreatedAt: now, UpdatedAt: now},
			3001: {ItemID: 3001, ItemCode: "bag_expand_ticket_small", ItemName: "背包扩容券", ItemType: "functional", ItemSubType: "expand", Quality: 2, Rarity: 2, MaxStack: 99, OccupySlots: 1, AutoMerge: true, Usable: true, CanSell: false, CanDrop: false, CanStore: true, EffectType: "bag_expand", EffectParamsJSON: "{\"expand_target\":\"bag\",\"expand_slots\":5}", BindType: "pickup_bind", IsEnabled: true, CreatedAt: now, UpdatedAt: now},
			3003: {ItemID: 3003, ItemCode: "pet_hp_potion_small", ItemName: "宠物治疗药剂", ItemType: "consumable", ItemSubType: "pet_restore", Quality: 1, Rarity: 1, MaxStack: 99, OccupySlots: 1, AutoMerge: true, Usable: true, UseScope: "world", TargetType: "pet_single", CanSell: true, CanDrop: true, CanStore: true, EffectType: "pet_hp_restore", EffectValue: 10, EffectParamsJSON: "{\"restore_type\":\"flat\"}", BindType: "none", IsEnabled: true, CreatedAt: now, UpdatedAt: now},
			3004: {ItemID: 3004, ItemCode: "starter_reward_box", ItemName: "新手补给礼包", ItemType: "box", ItemSubType: "reward_box", Quality: 2, Rarity: 2, MaxStack: 99, OccupySlots: 1, AutoMerge: true, Usable: true, UseScope: "world", TargetType: "self", CanSell: false, CanDrop: false, CanStore: true, EffectType: "reward_box", EffectParamsJSON: "{\"rewards\":[{\"type\":\"gold\",\"value\":2},{\"type\":\"item\",\"item_id\":2001,\"item_name\":\"新手精灵球\",\"count\":1}]}", BindType: "pickup_bind", IsEnabled: true, CreatedAt: now, UpdatedAt: now},
		},
	}
}

type ItemRepository struct {
	mu    sync.RWMutex
	items map[uint64]item.AdminItemDetail
}

func (r *ItemRepository) ListForAdmin(_ context.Context, query item.AdminListQuery) (*item.AdminItemList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = query.Normalize()
	items := make([]item.AdminItemSummary, 0, len(r.items))
	for _, current := range r.items {
		if query.ItemID > 0 && current.ItemID != query.ItemID {
			continue
		}
		if query.ItemType != "" && current.ItemType != query.ItemType {
			continue
		}
		if query.Keyword != "" && !strings.Contains(strings.ToLower(current.ItemCode+current.ItemName), strings.ToLower(query.Keyword)) {
			continue
		}
		if query.Enabled != nil && current.IsEnabled != *query.Enabled {
			continue
		}
		items = append(items, item.AdminItemSummary{ItemID: current.ItemID, ItemCode: current.ItemCode, ItemName: current.ItemName, ItemType: current.ItemType, ItemSubType: current.ItemSubType, Quality: current.Quality, MaxStack: current.MaxStack, BuyPriceCopper: current.BuyPriceCopper, SellPriceCopper: current.SellPriceCopper, Usable: current.Usable, CanSell: current.CanSell, CanStore: current.CanStore, IsEnabled: current.IsEnabled, UpdatedAt: current.UpdatedAt, CreatedAt: current.CreatedAt})
	}
	return &item.AdminItemList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *ItemRepository) FindAdminDetailByItemID(_ context.Context, itemID uint64) (*item.AdminItemDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.items[itemID]
	if !ok {
		return nil, nil
	}
	copied := current
	return &copied, nil
}

func (r *ItemRepository) CreateForAdmin(_ context.Context, input item.AdminUpsertItemInput) (*item.AdminItemDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[input.ItemID]; ok {
		return nil, item.ErrItemDefinitionConflict
	}
	now := time.Now()
	value := item.AdminItemDetail{ItemID: input.ItemID, ItemCode: input.ItemCode, ItemName: input.ItemName, ItemType: input.ItemType, ItemSubType: input.ItemSubType, Quality: input.Quality, Rarity: input.Rarity, Icon: input.Icon, Desc: input.Desc, MaxStack: input.MaxStack, OccupySlots: input.OccupySlots, AutoMerge: input.AutoMerge, SortWeight: input.SortWeight, Usable: input.Usable, UseScope: input.UseScope, TargetType: input.TargetType, RequiredLevel: input.RequiredLevel, RequiredSceneID: input.RequiredSceneID, BindType: input.BindType, CanSell: input.CanSell, CanDrop: input.CanDrop, CanStore: input.CanStore, CanTrade: input.CanTrade, ExpireAtRule: input.ExpireAtRule, EffectType: input.EffectType, EffectValue: input.EffectValue, EffectParamsJSON: input.EffectParamsJSON, BuyPriceCopper: input.BuyPriceCopper, SellPriceCopper: input.SellPriceCopper, RecyclePriceCopper: input.RecyclePriceCopper, PriceType: input.PriceType, IsEnabled: input.IsEnabled, CreatedAt: now, UpdatedAt: now}
	r.items[input.ItemID] = value
	copied := value
	return &copied, nil
}

func (r *ItemRepository) UpdateForAdmin(_ context.Context, itemID uint64, input item.AdminUpsertItemInput) (*item.AdminItemDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[itemID]
	if !ok {
		return nil, item.ErrItemDefinitionNotFound
	}
	current.ItemCode = input.ItemCode
	current.ItemName = input.ItemName
	current.ItemType = input.ItemType
	current.ItemSubType = input.ItemSubType
	current.Quality = input.Quality
	current.Rarity = input.Rarity
	current.Icon = input.Icon
	current.Desc = input.Desc
	current.MaxStack = input.MaxStack
	current.OccupySlots = input.OccupySlots
	current.AutoMerge = input.AutoMerge
	current.SortWeight = input.SortWeight
	current.Usable = input.Usable
	current.UseScope = input.UseScope
	current.TargetType = input.TargetType
	current.RequiredLevel = input.RequiredLevel
	current.RequiredSceneID = input.RequiredSceneID
	current.BindType = input.BindType
	current.CanSell = input.CanSell
	current.CanDrop = input.CanDrop
	current.CanStore = input.CanStore
	current.CanTrade = input.CanTrade
	current.ExpireAtRule = input.ExpireAtRule
	current.EffectType = input.EffectType
	current.EffectValue = input.EffectValue
	current.EffectParamsJSON = input.EffectParamsJSON
	current.BuyPriceCopper = input.BuyPriceCopper
	current.SellPriceCopper = input.SellPriceCopper
	current.RecyclePriceCopper = input.RecyclePriceCopper
	current.PriceType = input.PriceType
	current.IsEnabled = input.IsEnabled
	current.UpdatedAt = time.Now()
	r.items[itemID] = current
	copied := current
	return &copied, nil
}

func (r *ItemRepository) DeleteForAdmin(_ context.Context, itemID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[itemID]; !ok {
		return item.ErrItemDefinitionNotFound
	}
	delete(r.items, itemID)
	return nil
}

// NewWalletRepository 提供后台钱包列表与调账的内存桩。
func NewWalletRepository() *WalletRepository {
	now := time.Now()
	return &WalletRepository{
		wallets: map[uint64]wallet.AdminWalletDetail{
			DemoPlayerID:  {PlayerID: DemoPlayerID, PlayerName: "DemoTrainer", Wallet: wallet.Snapshot{TotalCopper: 2345678, Gold: 2, Silver: 345, Copper: 678}, Version: 1, CreatedAt: now, UpdatedAt: now},
			RivalPlayerID: {PlayerID: RivalPlayerID, PlayerName: "RivalTrainer", Wallet: wallet.Snapshot{TotalCopper: 9800, Gold: 0, Silver: 9, Copper: 800}, Version: 1, CreatedAt: now, UpdatedAt: now},
		},
	}
}

type WalletRepository struct {
	mu      sync.RWMutex
	wallets map[uint64]wallet.AdminWalletDetail
}

func (r *WalletRepository) ListForAdmin(_ context.Context, query wallet.AdminListQuery) (*wallet.AdminWalletList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = query.Normalize()
	items := make([]wallet.AdminWalletSummary, 0, len(r.wallets))
	for _, current := range r.wallets {
		if query.PlayerID > 0 && current.PlayerID != query.PlayerID {
			continue
		}
		if query.Keyword != "" && !strings.Contains(strings.ToLower(current.PlayerName), strings.ToLower(query.Keyword)) {
			continue
		}
		items = append(items, wallet.AdminWalletSummary{PlayerID: current.PlayerID, PlayerName: current.PlayerName, Wallet: current.Wallet, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt})
	}
	return &wallet.AdminWalletList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *WalletRepository) FindAdminDetailByPlayerID(_ context.Context, playerID uint64) (*wallet.AdminWalletDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.wallets[playerID]
	if !ok {
		return nil, nil
	}
	copied := current
	return &copied, nil
}

func (r *WalletRepository) AdjustForAdmin(_ context.Context, playerID uint64, input wallet.AdminAdjustInput) (*wallet.AdminWalletDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.wallets[playerID]
	if !ok {
		return nil, nil
	}
	next := int64(current.Wallet.TotalCopper) + input.ChangeTotalCopper
	if next < 0 {
		return nil, wallet.ErrInvalidAdminWalletInput
	}
	current.Wallet = wallet.Snapshot{TotalCopper: uint64(next), Gold: uint64(next) / 1000000, Silver: (uint64(next) % 1000000) / 1000, Copper: uint64(next) % 1000}
	current.Version++
	current.UpdatedAt = time.Now()
	r.wallets[playerID] = current
	copied := current
	return &copied, nil
}

func (r *WalletRepository) AdjustRuntime(_ context.Context, playerID uint64, input wallet.RuntimeAdjustInput) (*wallet.RuntimeAdjustResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.wallets[playerID]
	if !ok {
		return nil, nil
	}
	next := int64(current.Wallet.TotalCopper) + input.ChangeTotalCopper
	if next < 0 {
		return nil, wallet.ErrInvalidRuntimeAdjustInput
	}
	current.Wallet = wallet.Snapshot{
		TotalCopper: uint64(next),
		Gold:        uint64(next) / wallet.CopperPerGold,
		Silver:      (uint64(next) % wallet.CopperPerGold) / wallet.CopperPerSilver,
		Copper:      uint64(next) % wallet.CopperPerSilver,
	}
	current.Version++
	current.UpdatedAt = time.Now()
	r.wallets[playerID] = current
	return &wallet.RuntimeAdjustResult{
		Wallet:      current.Wallet,
		Version:     current.Version,
		ReasonType:  input.ReasonType,
		ReasonRefID: input.ReasonRefID,
	}, nil
}

func (r *WalletRepository) GetRuntimeSnapshot(_ context.Context, playerID uint64) (*wallet.Snapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.wallets[playerID]
	if !ok {
		return nil, nil
	}
	snapshot := current.Wallet
	return &snapshot, nil
}

// NewQuestRepository provides deterministic quest templates and per-player
// mutable state for world handler tests.
func NewQuestRepository() *QuestRepository {
	return &QuestRepository{
		nextPlayerQuestRecordID: 50000,
		templates: map[uint64]quest.Template{
			1001: {
				QuestID:     1001,
				QuestType:   "MAIN",
				Title:       "初入闪光镇",
				Description: "前往闪光镇东路，熟悉周围环境。",
				AcceptMode:  "AUTO",
				SubmitMode:  "AUTO",
				AutoTrack:   true,
				Objectives:  []quest.ObjectiveTemplate{{ObjectiveID: 1, EventType: "ENTER_SCENE", Description: "进入闪光镇东路", TargetValue: 1, TargetSelector: map[string]any{"scene_id": uint32(2)}}},
				Rewards:     []quest.Reward{{Type: "gold", Value: 100}},
			},
			1002: {
				QuestID:     1002,
				QuestType:   "MAIN",
				Title:       "向市场管理员报到",
				Description: "找到市场理萌并和她交谈。",
				AcceptMode:  "NPC",
				SubmitMode:  "NPC",
				StartNPCID:  93001,
				SubmitNPCID: 93001,
				AutoTrack:   true,
				PreQuestIDs: []uint64{1001},
				Objectives:  []quest.ObjectiveTemplate{{ObjectiveID: 1, EventType: "TALK_TO_NPC", Description: "与市场理萌交谈", TargetValue: 1, TargetSelector: map[string]any{"npc_id": uint64(93001)}}},
				Rewards: []quest.Reward{
					{Type: "gold", Value: 150},
					{Type: "item", ItemID: 2001, Count: 2},
					{Type: "pet", PetID: 102},
					{Type: "feature_unlock", Value: 1},
				},
			},
			1003: {
				QuestID:     1003,
				QuestType:   "MAIN",
				Title:       "完成第一次对战",
				Description: "挑战附近的教学 NPC 并赢得胜利。",
				AcceptMode:  "AUTO",
				SubmitMode:  "AUTO",
				AutoTrack:   true,
				PreQuestIDs: []uint64{1002},
				Objectives:  []quest.ObjectiveTemplate{{ObjectiveID: 1, EventType: "WIN_BATTLE", Description: "完成 1 场战斗", TargetValue: 1, TargetSelector: map[string]any{"battle_type": "PVE"}}},
				Rewards: []quest.Reward{
					{Type: "gold", Value: 200},
				},
			},
		},
		playerQuests:     map[uint64]map[uint64]quest.PlayerQuest{},
		playerObjectives: map[uint64]map[uint64][]quest.PlayerObjective{},
	}
}

type QuestRepository struct {
	mu                      sync.RWMutex
	nextPlayerQuestRecordID uint64
	templates               map[uint64]quest.Template
	playerQuests            map[uint64]map[uint64]quest.PlayerQuest
	playerObjectives        map[uint64]map[uint64][]quest.PlayerObjective
}

func (r *QuestRepository) ListTemplates(_ context.Context) ([]quest.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]quest.Template, 0, len(r.templates))
	for _, template := range r.templates {
		result = append(result, template)
	}
	return result, nil
}

func (r *QuestRepository) FindTemplateByQuestID(_ context.Context, questID uint64) (*quest.Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.templates[questID]
	if !ok {
		return nil, nil
	}
	copied := value
	return &copied, nil
}

func (r *QuestRepository) ListTemplatesForAdmin(_ context.Context, query quest.AdminTemplateListQuery) (*quest.AdminTemplateList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]quest.AdminTemplateSummary, 0, len(r.templates))
	for _, template := range r.templates {
		if query.QuestID > 0 && template.QuestID != query.QuestID {
			continue
		}
		if query.QuestType != "" && !strings.EqualFold(template.QuestType, query.QuestType) {
			continue
		}
		if query.Title != "" && !strings.Contains(strings.ToLower(template.Title), strings.ToLower(query.Title)) {
			continue
		}
		status := uint32(1)
		if query.Status != nil && *query.Status != status {
			continue
		}
		items = append(items, quest.AdminTemplateSummary{
			QuestID:        template.QuestID,
			Name:           fmt.Sprintf("quest_%d", template.QuestID),
			QuestType:      template.QuestType,
			Title:          template.Title,
			Chapter:        1,
			SortOrder:      uint32(template.QuestID),
			AcceptMode:     template.AcceptMode,
			SubmitMode:     template.SubmitMode,
			AutoTrack:      template.AutoTrack,
			MinPlayerLevel: 1,
			Status:         status,
			StatusText:     quest.AdminQuestTemplateStatusText(status),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
	}
	return &quest.AdminTemplateList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *QuestRepository) FindAdminTemplateDetailByQuestID(_ context.Context, questID uint64) (*quest.AdminTemplateDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	template, ok := r.templates[questID]
	if !ok {
		return nil, nil
	}
	objectives := make([]quest.AdminObjectiveInput, 0, len(template.Objectives))
	for _, objective := range template.Objectives {
		objectives = append(objectives, quest.AdminObjectiveInput{
			ObjectiveID:    objective.ObjectiveID,
			EventType:      objective.EventType,
			Description:    objective.Description,
			TargetValue:    objective.TargetValue,
			TargetSelector: objective.TargetSelector,
		})
	}
	return &quest.AdminTemplateDetail{
		QuestID:        template.QuestID,
		Name:           fmt.Sprintf("quest_%d", template.QuestID),
		QuestType:      template.QuestType,
		Title:          template.Title,
		Description:    template.Description,
		Chapter:        1,
		SortOrder:      uint32(template.QuestID),
		AcceptMode:     template.AcceptMode,
		SubmitMode:     template.SubmitMode,
		AutoTrack:      template.AutoTrack,
		StartNPCID:     template.StartNPCID,
		SubmitNPCID:    template.SubmitNPCID,
		MinPlayerLevel: 1,
		Status:         1,
		StatusText:     quest.AdminQuestTemplateStatusText(1),
		PreQuestIDs:    append([]uint64{}, template.PreQuestIDs...),
		Objectives:     objectives,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (r *QuestRepository) CreateTemplateForAdmin(_ context.Context, input quest.AdminCreateTemplateInput) (*quest.AdminTemplateDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.templates[input.QuestID]; exists {
		return nil, quest.ErrAdminQuestConflict
	}
	template := quest.Template{
		QuestID:     input.QuestID,
		QuestType:   input.QuestType,
		Title:       input.Title,
		Description: input.Description,
		AcceptMode:  input.AcceptMode,
		SubmitMode:  input.SubmitMode,
		StartNPCID:  input.StartNPCID,
		SubmitNPCID: input.SubmitNPCID,
		AutoTrack:   input.AutoTrack,
		PreQuestIDs: append([]uint64{}, input.PreQuestIDs...),
	}
	for _, objective := range input.Objectives {
		template.Objectives = append(template.Objectives, quest.ObjectiveTemplate{
			ObjectiveID:    objective.ObjectiveID,
			EventType:      objective.EventType,
			Description:    objective.Description,
			TargetValue:    objective.TargetValue,
			TargetSelector: objective.TargetSelector,
		})
	}
	r.templates[input.QuestID] = template
	objectives := make([]quest.AdminObjectiveInput, 0, len(input.Objectives))
	objectives = append(objectives, input.Objectives...)
	return &quest.AdminTemplateDetail{
		QuestID:        template.QuestID,
		Name:           input.Name,
		QuestType:      template.QuestType,
		Title:          template.Title,
		Description:    template.Description,
		Chapter:        input.Chapter,
		SortOrder:      input.SortOrder,
		AcceptMode:     template.AcceptMode,
		SubmitMode:     template.SubmitMode,
		AutoTrack:      template.AutoTrack,
		StartNPCID:     template.StartNPCID,
		SubmitNPCID:    template.SubmitNPCID,
		MinPlayerLevel: input.MinPlayerLevel,
		Status:         input.Status,
		StatusText:     quest.AdminQuestTemplateStatusText(input.Status),
		PreQuestIDs:    append([]uint64{}, template.PreQuestIDs...),
		Objectives:     objectives,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (r *QuestRepository) UpdateTemplateForAdmin(_ context.Context, questID uint64, input quest.AdminUpdateTemplateInput) (*quest.AdminTemplateDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	template, exists := r.templates[questID]
	if !exists {
		return nil, quest.ErrAdminQuestTemplateNotFound
	}
	template.QuestType = input.QuestType
	template.Title = input.Title
	template.Description = input.Description
	template.AcceptMode = input.AcceptMode
	template.SubmitMode = input.SubmitMode
	template.StartNPCID = input.StartNPCID
	template.SubmitNPCID = input.SubmitNPCID
	template.AutoTrack = input.AutoTrack
	template.PreQuestIDs = append([]uint64{}, input.PreQuestIDs...)
	template.Objectives = []quest.ObjectiveTemplate{}
	for _, objective := range input.Objectives {
		template.Objectives = append(template.Objectives, quest.ObjectiveTemplate{
			ObjectiveID:    objective.ObjectiveID,
			EventType:      objective.EventType,
			Description:    objective.Description,
			TargetValue:    objective.TargetValue,
			TargetSelector: objective.TargetSelector,
		})
	}
	r.templates[questID] = template
	objectives := make([]quest.AdminObjectiveInput, 0, len(input.Objectives))
	objectives = append(objectives, input.Objectives...)
	return &quest.AdminTemplateDetail{
		QuestID:        template.QuestID,
		Name:           input.Name,
		QuestType:      template.QuestType,
		Title:          template.Title,
		Description:    template.Description,
		Chapter:        input.Chapter,
		SortOrder:      input.SortOrder,
		AcceptMode:     template.AcceptMode,
		SubmitMode:     template.SubmitMode,
		AutoTrack:      template.AutoTrack,
		StartNPCID:     template.StartNPCID,
		SubmitNPCID:    template.SubmitNPCID,
		MinPlayerLevel: input.MinPlayerLevel,
		Status:         input.Status,
		StatusText:     quest.AdminQuestTemplateStatusText(input.Status),
		PreQuestIDs:    append([]uint64{}, template.PreQuestIDs...),
		Objectives:     objectives,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (r *QuestRepository) DeleteTemplateForAdmin(_ context.Context, questID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.templates[questID]; !exists {
		return quest.ErrAdminQuestTemplateNotFound
	}
	delete(r.templates, questID)
	return nil
}

func (r *QuestRepository) ListPlayerQuestsByPlayerID(_ context.Context, playerID uint64) ([]quest.PlayerQuest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playerMap := r.playerQuests[playerID]
	if len(playerMap) == 0 {
		return []quest.PlayerQuest{}, nil
	}
	result := make([]quest.PlayerQuest, 0, len(playerMap))
	for _, value := range playerMap {
		result = append(result, value)
	}
	return result, nil
}

func (r *QuestRepository) ListPlayerObjectivesByPlayerID(_ context.Context, playerID uint64) ([]quest.PlayerObjective, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	questMap := r.playerObjectives[playerID]
	if len(questMap) == 0 {
		return []quest.PlayerObjective{}, nil
	}
	result := []quest.PlayerObjective{}
	for _, values := range questMap {
		result = append(result, values...)
	}
	return result, nil
}

func (r *QuestRepository) ListPlayerQuestsForAdmin(_ context.Context, query quest.AdminPlayerQuestListQuery) (*quest.AdminPlayerQuestList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]quest.AdminPlayerQuestSummary, 0)
	for playerID, playerMap := range r.playerQuests {
		for questID, value := range playerMap {
			if query.PlayerID > 0 && playerID != query.PlayerID {
				continue
			}
			if query.QuestID > 0 && questID != query.QuestID {
				continue
			}
			if query.State != "" && value.State != query.State {
				continue
			}
			if query.Tracked != nil && value.Tracked != *query.Tracked {
				continue
			}
			recordID := adminPlayerQuestRecordID(playerID, questID)
			if query.RecordID > 0 && recordID != query.RecordID {
				continue
			}
			template := r.templates[questID]
			items = append(items, quest.AdminPlayerQuestSummary{
				RecordID:      recordID,
				PlayerID:      playerID,
				PlayerName:    bagPlayerName(playerID),
				QuestID:       questID,
				QuestTitle:    template.Title,
				QuestType:     template.QuestType,
				State:         value.State,
				Tracked:       value.Tracked,
				RewardClaimed: value.State == quest.StateCompleted,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			})
		}
	}
	return &quest.AdminPlayerQuestList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *QuestRepository) FindAdminPlayerQuestDetailByRecordID(_ context.Context, recordID uint64) (*quest.AdminPlayerQuestDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	playerID, questID, ok := parseAdminPlayerQuestRecordID(recordID)
	if !ok {
		return nil, nil
	}
	value, exists := r.playerQuests[playerID][questID]
	if !exists {
		return nil, nil
	}
	template := r.templates[questID]
	objectives := make([]quest.AdminPlayerObjectiveInput, 0, len(r.playerObjectives[playerID][questID]))
	for _, objective := range r.playerObjectives[playerID][questID] {
		objectives = append(objectives, quest.AdminPlayerObjectiveInput{
			ObjectiveID:  objective.ObjectiveID,
			Description:  objective.Description,
			CurrentValue: objective.CurrentValue,
			TargetValue:  objective.TargetValue,
			Completed:    objective.Completed,
		})
	}
	return &quest.AdminPlayerQuestDetail{
		RecordID:      recordID,
		PlayerID:      playerID,
		PlayerName:    bagPlayerName(playerID),
		QuestID:       questID,
		QuestTitle:    template.Title,
		QuestType:     template.QuestType,
		State:         value.State,
		Tracked:       value.Tracked,
		RewardClaimed: value.State == quest.StateCompleted,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Objectives:    objectives,
	}, nil
}

func (r *QuestRepository) CreatePlayerQuestForAdmin(_ context.Context, input quest.AdminCreatePlayerQuestInput) (*quest.AdminPlayerQuestDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.templates[input.QuestID]; !exists {
		return nil, quest.ErrAdminQuestTemplateNotFound
	}
	if r.playerQuests[input.PlayerID] == nil {
		r.playerQuests[input.PlayerID] = map[uint64]quest.PlayerQuest{}
	}
	if _, exists := r.playerQuests[input.PlayerID][input.QuestID]; exists {
		return nil, quest.ErrAdminQuestConflict
	}
	r.playerQuests[input.PlayerID][input.QuestID] = quest.PlayerQuest{PlayerID: input.PlayerID, QuestID: input.QuestID, State: input.State, Tracked: input.Tracked}
	if r.playerObjectives[input.PlayerID] == nil {
		r.playerObjectives[input.PlayerID] = map[uint64][]quest.PlayerObjective{}
	}
	objectives := make([]quest.PlayerObjective, 0, len(input.Objectives))
	for _, objective := range input.Objectives {
		objectives = append(objectives, quest.PlayerObjective{
			PlayerID: input.PlayerID, QuestID: input.QuestID, ObjectiveID: objective.ObjectiveID,
			Description: objective.Description, CurrentValue: objective.CurrentValue, TargetValue: objective.TargetValue, Completed: objective.Completed,
		})
	}
	r.playerObjectives[input.PlayerID][input.QuestID] = objectives
	template := r.templates[input.QuestID]
	detailObjectives := make([]quest.AdminPlayerObjectiveInput, 0, len(input.Objectives))
	detailObjectives = append(detailObjectives, input.Objectives...)
	return &quest.AdminPlayerQuestDetail{
		RecordID:      adminPlayerQuestRecordID(input.PlayerID, input.QuestID),
		PlayerID:      input.PlayerID,
		PlayerName:    bagPlayerName(input.PlayerID),
		QuestID:       input.QuestID,
		QuestTitle:    template.Title,
		QuestType:     template.QuestType,
		State:         input.State,
		Tracked:       input.Tracked,
		RewardClaimed: input.RewardClaimed,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Objectives:    detailObjectives,
	}, nil
}

func (r *QuestRepository) UpdatePlayerQuestForAdmin(_ context.Context, recordID uint64, input quest.AdminUpdatePlayerQuestInput) (*quest.AdminPlayerQuestDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldPlayerID, oldQuestID, ok := parseAdminPlayerQuestRecordID(recordID)
	if !ok {
		return nil, quest.ErrAdminPlayerQuestNotFound
	}
	if _, exists := r.playerQuests[oldPlayerID][oldQuestID]; !exists {
		return nil, quest.ErrAdminPlayerQuestNotFound
	}
	if _, exists := r.templates[input.QuestID]; !exists {
		return nil, quest.ErrAdminQuestTemplateNotFound
	}
	if r.playerQuests[input.PlayerID] == nil {
		r.playerQuests[input.PlayerID] = map[uint64]quest.PlayerQuest{}
	}
	if (oldPlayerID != input.PlayerID || oldQuestID != input.QuestID) && r.playerQuests[input.PlayerID][input.QuestID].QuestID != 0 {
		return nil, quest.ErrAdminQuestConflict
	}
	delete(r.playerQuests[oldPlayerID], oldQuestID)
	if r.playerObjectives[oldPlayerID] != nil {
		delete(r.playerObjectives[oldPlayerID], oldQuestID)
	}
	r.playerQuests[input.PlayerID][input.QuestID] = quest.PlayerQuest{PlayerID: input.PlayerID, QuestID: input.QuestID, State: input.State, Tracked: input.Tracked}
	if r.playerObjectives[input.PlayerID] == nil {
		r.playerObjectives[input.PlayerID] = map[uint64][]quest.PlayerObjective{}
	}
	objectives := make([]quest.PlayerObjective, 0, len(input.Objectives))
	for _, objective := range input.Objectives {
		objectives = append(objectives, quest.PlayerObjective{
			PlayerID: input.PlayerID, QuestID: input.QuestID, ObjectiveID: objective.ObjectiveID,
			Description: objective.Description, CurrentValue: objective.CurrentValue, TargetValue: objective.TargetValue, Completed: objective.Completed,
		})
	}
	r.playerObjectives[input.PlayerID][input.QuestID] = objectives
	template := r.templates[input.QuestID]
	detailObjectives := make([]quest.AdminPlayerObjectiveInput, 0, len(input.Objectives))
	detailObjectives = append(detailObjectives, input.Objectives...)
	return &quest.AdminPlayerQuestDetail{
		RecordID:      adminPlayerQuestRecordID(input.PlayerID, input.QuestID),
		PlayerID:      input.PlayerID,
		PlayerName:    bagPlayerName(input.PlayerID),
		QuestID:       input.QuestID,
		QuestTitle:    template.Title,
		QuestType:     template.QuestType,
		State:         input.State,
		Tracked:       input.Tracked,
		RewardClaimed: input.RewardClaimed,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Objectives:    detailObjectives,
	}, nil
}

func (r *QuestRepository) DeletePlayerQuestForAdmin(_ context.Context, recordID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	playerID, questID, ok := parseAdminPlayerQuestRecordID(recordID)
	if !ok {
		return quest.ErrAdminPlayerQuestNotFound
	}
	if _, exists := r.playerQuests[playerID][questID]; !exists {
		return quest.ErrAdminPlayerQuestNotFound
	}
	delete(r.playerQuests[playerID], questID)
	if r.playerObjectives[playerID] != nil {
		delete(r.playerObjectives[playerID], questID)
	}
	return nil
}

func (r *QuestRepository) UpsertPlayerQuest(_ context.Context, value quest.PlayerQuest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.playerQuests[value.PlayerID] == nil {
		r.playerQuests[value.PlayerID] = map[uint64]quest.PlayerQuest{}
	}
	r.playerQuests[value.PlayerID][value.QuestID] = value
	return nil
}

func (r *QuestRepository) ReplacePlayerObjectives(_ context.Context, playerID uint64, questID uint64, objectives []quest.PlayerObjective) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.playerObjectives[playerID] == nil {
		r.playerObjectives[playerID] = map[uint64][]quest.PlayerObjective{}
	}
	copied := append([]quest.PlayerObjective{}, objectives...)
	r.playerObjectives[playerID][questID] = copied
	return nil
}

func (r *QuestRepository) SetTrackedQuest(_ context.Context, playerID uint64, questID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.playerQuests[playerID] == nil {
		r.playerQuests[playerID] = map[uint64]quest.PlayerQuest{}
	}
	for currentQuestID, current := range r.playerQuests[playerID] {
		current.Tracked = currentQuestID == questID
		r.playerQuests[playerID][currentQuestID] = current
	}
	target := r.playerQuests[playerID][questID]
	target.PlayerID = playerID
	target.QuestID = questID
	target.Tracked = true
	if target.State == "" {
		target.State = quest.StateAvailable
	}
	r.playerQuests[playerID][questID] = target
	return nil
}

func adminPlayerQuestRecordID(playerID uint64, questID uint64) uint64 {
	return playerID*100000 + questID
}

func parseAdminPlayerQuestRecordID(recordID uint64) (uint64, uint64, bool) {
	if recordID < 100000 {
		return 0, 0, false
	}
	return recordID / 100000, recordID % 100000, true
}

// NewNPCRepository supplies the static menu/action data expected by NPC tests.
func NewNPCRepository() *NPCRepository {
	return &NPCRepository{
		entities: map[uint64]npc.AdminEntityDetail{
			91001: {EntityID: 91001, EntityCode: "warehouse_luosi", DisplayName: "罗思", EntityType: 2, SceneID: 1, PosX: 6, PosY: 6, Dir: 2, Speed: 0, Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			93001: {EntityID: 93001, EntityCode: "radiant_market_limeng", DisplayName: "市场理萌", EntityType: 2, SceneID: 3, PosX: 13, PosY: 8, Dir: 2, Speed: 0, Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			93002: {EntityID: 93002, EntityCode: "radiant_market_luoge", DisplayName: "市场罗格", EntityType: 2, SceneID: 3, PosX: 14, PosY: 6, Dir: 2, Speed: 0, Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
		menuEntries: map[uint64]map[string]npc.AdminMenuEntryDetail{
			91001: {
				"dialog_warehouse_intro": {EntityID: 91001, EntryID: "dialog_warehouse_intro", EntryType: "dialog", Title: "仓库介绍", Subtitle: "问问仓库平时负责什么", State: "available", Priority: 80, SortOrder: 10, ActionResultType: "notice", ActionNotice: "罗思说：这里负责保管训练家暂时寄存的物资。", Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			93001: {
				"dialog_market_news": {EntityID: 93001, EntryID: "dialog_market_news", EntryType: "dialog", Title: "打听消息", Subtitle: "问问市场最近的新鲜事", State: "available", Priority: 80, SortOrder: 10, ActionResultType: "notice", ActionNotice: "理萌说：最近市场新开了几家铺子。", Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			93002: {
				"shop_open_market": {EntityID: 93002, EntryID: "shop_open_market", EntryType: "shop", Title: "打开商店", Subtitle: "浏览基础商品（占位）", State: "available", Priority: 100, SortOrder: 10, ActionResultType: "notice", ActionNotice: "商店面板待接入，当前先返回占位提示。", Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
				"dialog_trade_tip": {EntityID: 93002, EntryID: "dialog_trade_tip", EntryType: "dialog", Title: "讨价还价", Subtitle: "听听老商贩的经验", State: "available", Priority: 70, SortOrder: 20, ActionResultType: "notice", ActionNotice: "罗格说：买卖讲究货比三家。", Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
				"battle_market_guard": {EntityID: 93002, EntryID: "battle_market_guard", EntryType: "battle", Title: "挑战", Subtitle: "与市场守卫切磋", State: "available", Priority: 90, SortOrder: 15, ActionResultType: "battle", ActionNotice: "", BattleEncounterEntityID: 90002, Status: 1, StatusText: npc.AdminNPCStatusText(1), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
		},
	}
}

type NPCRepository struct {
	mu          sync.RWMutex
	entities    map[uint64]npc.AdminEntityDetail
	menuEntries map[uint64]map[string]npc.AdminMenuEntryDetail
}

func (r *NPCRepository) ListMenuEntriesByEntityID(_ context.Context, entityID uint64) ([]npc.MenuEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, ok := r.menuEntries[entityID]
	if !ok {
		return nil, nil
	}
	result := make([]npc.MenuEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, npc.MenuEntry{
			EntityID: entry.EntityID, EntryID: entry.EntryID, EntryType: entry.EntryType, Title: entry.Title,
			Subtitle: entry.Subtitle, State: entry.State, Priority: entry.Priority, ActionResultType: entry.ActionResultType, ActionNotice: entry.ActionNotice,
			BattleEncounterEntityID: entry.BattleEncounterEntityID,
		})
	}
	return result, nil
}

func (r *NPCRepository) FindActionResult(_ context.Context, entityID uint64, entryID string) (*npc.ActionResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, ok := r.menuEntries[entityID]
	if !ok {
		return nil, nil
	}
	entry, ok := entries[entryID]
	if ok {
		return &npc.ActionResult{EntityID: entityID, EntryID: entryID, ResultType: entry.ActionResultType, Notice: entry.ActionNotice, BattleEncounterEntityID: entry.BattleEncounterEntityID}, nil
	}
	return nil, nil
}

func (r *NPCRepository) ListEntitiesForAdmin(_ context.Context, query npc.AdminEntityListQuery) (*npc.AdminEntityList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]npc.AdminEntitySummary, 0, len(r.entities))
	for _, current := range r.entities {
		if query.EntityID > 0 && current.EntityID != query.EntityID {
			continue
		}
		if query.SceneID > 0 && current.SceneID != query.SceneID {
			continue
		}
		if query.EntityType != nil && current.EntityType != *query.EntityType {
			continue
		}
		if query.Status != nil && current.Status != *query.Status {
			continue
		}
		if query.Name != "" && !strings.Contains(strings.ToLower(current.DisplayName), strings.ToLower(query.Name)) {
			continue
		}
		items = append(items, npc.AdminEntitySummary{
			EntityID: current.EntityID, EntityCode: current.EntityCode, DisplayName: current.DisplayName, EntityType: current.EntityType,
			SceneID: current.SceneID, PosX: current.PosX, PosY: current.PosY, Dir: current.Dir, Speed: current.Speed,
			Status: current.Status, StatusText: current.StatusText, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt,
		})
	}
	return &npc.AdminEntityList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *NPCRepository) FindAdminEntityDetailByEntityID(_ context.Context, entityID uint64) (*npc.AdminEntityDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current, ok := r.entities[entityID]
	if !ok {
		return nil, nil
	}
	copied := current
	return &copied, nil
}

func (r *NPCRepository) CreateEntityForAdmin(_ context.Context, input npc.AdminCreateEntityInput) (*npc.AdminEntityDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entities[input.EntityID]; exists {
		return nil, npc.ErrAdminNPCConflict
	}
	now := time.Now()
	detail := npc.AdminEntityDetail{
		EntityID: input.EntityID, EntityCode: input.EntityCode, DisplayName: input.DisplayName, EntityType: input.EntityType,
		SceneID: input.SceneID, PosX: input.PosX, PosY: input.PosY, Dir: input.Dir, Speed: input.Speed,
		Status: input.Status, StatusText: npc.AdminNPCStatusText(input.Status), CreatedAt: now, UpdatedAt: now,
	}
	r.entities[input.EntityID] = detail
	return &detail, nil
}

func (r *NPCRepository) UpdateEntityForAdmin(_ context.Context, entityID uint64, input npc.AdminUpdateEntityInput) (*npc.AdminEntityDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.entities[entityID]
	if !exists {
		return nil, npc.ErrAdminNPCNotFound
	}
	current.EntityCode = input.EntityCode
	current.DisplayName = input.DisplayName
	current.EntityType = input.EntityType
	current.SceneID = input.SceneID
	current.PosX = input.PosX
	current.PosY = input.PosY
	current.Dir = input.Dir
	current.Speed = input.Speed
	current.Status = input.Status
	current.StatusText = npc.AdminNPCStatusText(input.Status)
	current.UpdatedAt = time.Now()
	r.entities[entityID] = current
	return &current, nil
}

func (r *NPCRepository) DeleteEntityForAdmin(_ context.Context, entityID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entities[entityID]; !exists {
		return npc.ErrAdminNPCNotFound
	}
	delete(r.entities, entityID)
	delete(r.menuEntries, entityID)
	return nil
}

func (r *NPCRepository) ListMenuEntriesForAdmin(_ context.Context, query npc.AdminMenuEntryListQuery) (*npc.AdminMenuEntryList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = query.Normalize()
	items := make([]npc.AdminMenuEntrySummary, 0)
	for entityID, entryMap := range r.menuEntries {
		if query.EntityID > 0 && entityID != query.EntityID {
			continue
		}
		for _, current := range entryMap {
			if query.EntryID != "" && current.EntryID != query.EntryID {
				continue
			}
			if query.Status != nil && current.Status != *query.Status {
				continue
			}
			items = append(items, npc.AdminMenuEntrySummary{
				EntityID: current.EntityID, EntryID: current.EntryID, EntryType: current.EntryType, Title: current.Title,
				Subtitle: current.Subtitle, State: current.State, Priority: current.Priority, SortOrder: current.SortOrder,
				ActionResultType: current.ActionResultType, BattleEncounterEntityID: current.BattleEncounterEntityID,
				Status: current.Status, StatusText: current.StatusText,
				CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt,
			})
		}
	}
	return &npc.AdminMenuEntryList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *NPCRepository) FindAdminMenuEntryDetail(_ context.Context, entityID uint64, entryID string) (*npc.AdminMenuEntryDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entryMap, ok := r.menuEntries[entityID]
	if !ok {
		return nil, nil
	}
	current, ok := entryMap[entryID]
	if !ok {
		return nil, nil
	}
	copied := current
	return &copied, nil
}

func (r *NPCRepository) CreateMenuEntryForAdmin(_ context.Context, input npc.AdminCreateMenuEntryInput) (*npc.AdminMenuEntryDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entities[input.EntityID]; !exists {
		return nil, npc.ErrAdminNPCNotFound
	}
	if r.menuEntries[input.EntityID] == nil {
		r.menuEntries[input.EntityID] = map[string]npc.AdminMenuEntryDetail{}
	}
	if _, exists := r.menuEntries[input.EntityID][input.EntryID]; exists {
		return nil, npc.ErrAdminNPCConflict
	}
	now := time.Now()
	detail := npc.AdminMenuEntryDetail{
		EntityID: input.EntityID, EntryID: input.EntryID, EntryType: input.EntryType, Title: input.Title, Subtitle: input.Subtitle,
		State: input.State, Priority: input.Priority, SortOrder: input.SortOrder, ActionResultType: input.ActionResultType,
		ActionNotice: input.ActionNotice, BattleEncounterEntityID: input.BattleEncounterEntityID, Status: input.Status, StatusText: npc.AdminNPCStatusText(input.Status), CreatedAt: now, UpdatedAt: now,
	}
	r.menuEntries[input.EntityID][input.EntryID] = detail
	return &detail, nil
}

func (r *NPCRepository) UpdateMenuEntryForAdmin(_ context.Context, entityID uint64, entryID string, input npc.AdminUpdateMenuEntryInput) (*npc.AdminMenuEntryDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entryMap, ok := r.menuEntries[entityID]
	if !ok {
		return nil, npc.ErrAdminNPCMenuEntryNotFound
	}
	current, ok := entryMap[entryID]
	if !ok {
		return nil, npc.ErrAdminNPCMenuEntryNotFound
	}
	if _, exists := r.entities[input.EntityID]; !exists {
		return nil, npc.ErrAdminNPCNotFound
	}
	delete(entryMap, entryID)
	if r.menuEntries[input.EntityID] == nil {
		r.menuEntries[input.EntityID] = map[string]npc.AdminMenuEntryDetail{}
	}
	current.EntityID = input.EntityID
	current.EntryType = input.EntryType
	current.Title = input.Title
	current.Subtitle = input.Subtitle
	current.State = input.State
	current.Priority = input.Priority
	current.SortOrder = input.SortOrder
	current.ActionResultType = input.ActionResultType
	current.ActionNotice = input.ActionNotice
	current.BattleEncounterEntityID = input.BattleEncounterEntityID
	current.Status = input.Status
	current.StatusText = npc.AdminNPCStatusText(input.Status)
	current.UpdatedAt = time.Now()
	r.menuEntries[input.EntityID][entryID] = current
	return &current, nil
}

func (r *NPCRepository) DeleteMenuEntryForAdmin(_ context.Context, entityID uint64, entryID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entryMap, ok := r.menuEntries[entityID]
	if !ok {
		return npc.ErrAdminNPCMenuEntryNotFound
	}
	if _, ok := entryMap[entryID]; !ok {
		return npc.ErrAdminNPCMenuEntryNotFound
	}
	delete(entryMap, entryID)
	return nil
}

// NewWorldRepository provides deterministic scene snapshots and portal routing
// so world transport tests can still cover transfer logic without a live DB.
func NewWorldRepository() *WorldRepository {
	return &WorldRepository{}
}

type WorldRepository struct{}

type portalData struct {
	targetSceneID uint32
	targetPos     world.Vec2i
}

type sceneData struct {
	spawnPos world.Vec2i
	entries  map[uint32]world.Vec2i
	nearby   []world.Entity
	exits    map[uint32]struct{}
	portals  map[uint32]portalData
}

var scenes = map[uint32]sceneData{
	1: {spawnPos: world.Vec2i{X: 8, Y: 6}, entries: map[uint32]world.Vec2i{2: {X: 8, Y: 12}}, nearby: []world.Entity{{EntityID: 90001, EntityType: 2, Pos: world.Vec2i{X: 10, Y: 6}, Dir: 2, Speed: 0, Name: "GuideNPC"}, {EntityID: 91001, EntityType: 2, Pos: world.Vec2i{X: 6, Y: 6}, Dir: 2, Speed: 0, Name: "罗思"}, {EntityID: RivalPlayerID, PlayerID: RivalPlayerID, EntityType: 1, Pos: world.Vec2i{X: 7, Y: 7}, Dir: 2, Speed: 0, Name: "RivalTrainer"}}, exits: map[uint32]struct{}{2: {}}, portals: map[uint32]portalData{1001: {targetSceneID: 2, targetPos: world.Vec2i{X: 4, Y: 1}}}},
	2: {spawnPos: world.Vec2i{X: 4, Y: 1}, entries: map[uint32]world.Vec2i{1: {X: 4, Y: 1}, 3: {X: 0, Y: 4}}, nearby: []world.Entity{{EntityID: 90002, EntityType: 2, Pos: world.Vec2i{X: 2, Y: 3}, Dir: 1, Speed: 0, Name: "StationKeeper"}}, exits: map[uint32]struct{}{1: {}, 3: {}}, portals: map[uint32]portalData{2001: {targetSceneID: 1, targetPos: world.Vec2i{X: 8, Y: 12}}, 2002: {targetSceneID: 3, targetPos: world.Vec2i{X: 12, Y: 10}}}},
	3: {spawnPos: world.Vec2i{X: 12, Y: 10}, entries: map[uint32]world.Vec2i{2: {X: 12, Y: 10}, 4: {X: 5, Y: 2}, 5: {X: 4, Y: 13}}, nearby: []world.Entity{{EntityID: 93001, EntityType: 2, Pos: world.Vec2i{X: 13, Y: 8}, Dir: 2, Speed: 0, Name: "市场理萌"}, {EntityID: 93002, EntityType: 2, Pos: world.Vec2i{X: 14, Y: 6}, Dir: 2, Speed: 0, Name: "市场罗格"}}, exits: map[uint32]struct{}{2: {}, 4: {}, 5: {}}, portals: map[uint32]portalData{3001: {targetSceneID: 2, targetPos: world.Vec2i{X: 0, Y: 4}}, 3002: {targetSceneID: 4, targetPos: world.Vec2i{X: 2, Y: 8}}, 3003: {targetSceneID: 5, targetPos: world.Vec2i{X: 11, Y: 2}}}},
	4: {spawnPos: world.Vec2i{X: 2, Y: 8}, entries: map[uint32]world.Vec2i{3: {X: 2, Y: 8}}, nearby: []world.Entity{{EntityID: 90004, EntityType: 2, Pos: world.Vec2i{X: 4, Y: 7}, Dir: 2, Speed: 0, Name: "NorthFieldScout"}}, exits: map[uint32]struct{}{3: {}}, portals: map[uint32]portalData{4001: {targetSceneID: 3, targetPos: world.Vec2i{X: 5, Y: 2}}}},
	5: {spawnPos: world.Vec2i{X: 11, Y: 2}, entries: map[uint32]world.Vec2i{3: {X: 11, Y: 2}, 6: {X: 6, Y: 10}}, nearby: []world.Entity{{EntityID: 90005, EntityType: 2, Pos: world.Vec2i{X: 9, Y: 4}, Dir: 1, Speed: 0, Name: "SchoolCaretaker"}}, exits: map[uint32]struct{}{3: {}, 6: {}}, portals: map[uint32]portalData{5001: {targetSceneID: 3, targetPos: world.Vec2i{X: 4, Y: 13}}, 5002: {targetSceneID: 6, targetPos: world.Vec2i{X: 6, Y: 10}}}},
	6: {spawnPos: world.Vec2i{X: 6, Y: 10}, entries: map[uint32]world.Vec2i{5: {X: 6, Y: 10}}, nearby: []world.Entity{{EntityID: 90006, EntityType: 2, Pos: world.Vec2i{X: 7, Y: 8}, Dir: 0, Speed: 0, Name: "BattleGuide"}}, exits: map[uint32]struct{}{5: {}}, portals: map[uint32]portalData{6001: {targetSceneID: 5, targetPos: world.Vec2i{X: 6, Y: 10}}}},
}

func (r *WorldRepository) GetSceneSnapshot(_ context.Context, _ uint64, sceneID uint32, selfPos world.Vec2i) (*world.SceneSnapshot, error) {
	scene, ok := scenes[sceneID]
	if !ok {
		return nil, world.ErrSnapshotUnavailable
	}
	return &world.SceneSnapshot{SceneID: sceneID, SelfPos: selfPos, SceneVersion: 1, NearbyEntities: scene.nearby}, nil
}

func (r *WorldRepository) EvaluateTransfer(_ context.Context, _ uint64, sceneID uint32, currentPos world.Vec2i, targetSceneID uint32, portalID uint32) (*world.MoveDecision, error) {
	decision := &world.MoveDecision{SceneVersion: 1, ToSceneID: sceneID, SpawnPos: currentPos}

	currentScene, ok := scenes[sceneID]
	if !ok {
		decision.Accepted = false
		decision.Reason = "current scene unavailable"
		return decision, nil
	}

	targetScene, ok := scenes[targetSceneID]
	if !ok {
		decision.Accepted = false
		decision.Reason = "target scene unavailable"
		return decision, nil
	}

	if _, ok := currentScene.exits[targetSceneID]; !ok {
		decision.Accepted = false
		decision.Reason = "target scene unreachable"
		return decision, nil
	}

	if portalID != 0 {
		portal, ok := currentScene.portals[portalID]
		if !ok {
			decision.Accepted = false
			decision.Reason = "portal unavailable"
			return decision, nil
		}
		if portal.targetSceneID != targetSceneID {
			decision.Accepted = false
			decision.Reason = "portal target mismatch"
			return decision, nil
		}
		decision.Accepted = true
		decision.ToSceneID = portal.targetSceneID
		decision.SpawnPos = portal.targetPos
		return decision, nil
	}

	decision.Accepted = true
	decision.ToSceneID = targetSceneID
	decision.SpawnPos = targetScene.spawnPos
	if entryPos, ok := targetScene.entries[sceneID]; ok {
		decision.SpawnPos = entryPos
	}
	return decision, nil
}
