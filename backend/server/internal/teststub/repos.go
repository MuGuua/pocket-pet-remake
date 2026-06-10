package teststub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/npc"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/quest"
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
	}
}

type AccountRepository struct {
	mu       sync.RWMutex
	accounts map[string]auth.Account
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
		players: map[uint64]player.Profile{
			DemoPlayerID: {
				PlayerID: DemoPlayerID,
				Name:     "DemoTrainer",
				Level:    1,
				Exp:      0,
				Gold:     100,
				SceneID:  1,
				PosX:     8,
				PosY:     6,
			},
			RivalPlayerID: {
				PlayerID: RivalPlayerID,
				Name:     "RivalTrainer",
				Level:    1,
				Exp:      0,
				Gold:     100,
				SceneID:  1,
				PosX:     9,
				PosY:     6,
			},
		},
	}
}

type PlayerRepository struct {
	mu      sync.RWMutex
	players map[uint64]player.Profile
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

func (r *PlayerRepository) AddGoldAndExp(_ context.Context, playerID uint64, gold uint32, exp uint64) (*player.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.players[playerID]
	if !ok {
		return nil, player.ErrPlayerNotFound
	}
	current.Gold += gold
	current.Exp += exp
	r.players[playerID] = current
	copied := current
	return &copied, nil
}

// NewPetRepository returns the fixed starter pets used by battle, pet list,
// and lineup tests. Keeping this local avoids coupling transport tests to a DB.
func NewPetRepository() *PetRepository {
	return &PetRepository{
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
	mu     sync.RWMutex
	pets   map[uint64][]pet.Pet
	lineup map[uint64][]pet.LineupPet
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

// NewQuestRepository provides deterministic quest templates and per-player
// mutable state for world handler tests.
func NewQuestRepository() *QuestRepository {
	return &QuestRepository{
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
			},
		},
		playerQuests:     map[uint64]map[uint64]quest.PlayerQuest{},
		playerObjectives: map[uint64]map[uint64][]quest.PlayerObjective{},
	}
}

type QuestRepository struct {
	mu               sync.RWMutex
	templates        map[uint64]quest.Template
	playerQuests     map[uint64]map[uint64]quest.PlayerQuest
	playerObjectives map[uint64]map[uint64][]quest.PlayerObjective
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

// NewNPCRepository supplies the static menu/action data expected by NPC tests.
func NewNPCRepository() *NPCRepository {
	return &NPCRepository{}
}

type NPCRepository struct{}

var npcMenuEntries = map[uint64][]npc.MenuEntry{
	91001: {{EntityID: 91001, EntryID: "dialog_warehouse_intro", EntryType: "dialog", Title: "仓库介绍", Subtitle: "问问仓库平时负责什么", State: "available", Priority: 80, ActionResultType: "notice", ActionNotice: "罗思说：这里负责保管训练家暂时寄存的物资。"}},
	93001: {{EntityID: 93001, EntryID: "dialog_market_news", EntryType: "dialog", Title: "打听消息", Subtitle: "问问市场最近的新鲜事", State: "available", Priority: 80, ActionResultType: "notice", ActionNotice: "理萌说：最近市场新开了几家铺子。"}},
	93002: {
		{EntityID: 93002, EntryID: "shop_open_market", EntryType: "shop", Title: "打开商店", Subtitle: "浏览基础商品（占位）", State: "available", Priority: 100, ActionResultType: "notice", ActionNotice: "商店面板待接入，当前先返回占位提示。"},
		{EntityID: 93002, EntryID: "dialog_trade_tip", EntryType: "dialog", Title: "讨价还价", Subtitle: "听听老商贩的经验", State: "available", Priority: 70, ActionResultType: "notice", ActionNotice: "罗格说：买卖讲究货比三家。"},
	},
}

func (r *NPCRepository) ListMenuEntriesByEntityID(_ context.Context, entityID uint64) ([]npc.MenuEntry, error) {
	entries, ok := npcMenuEntries[entityID]
	if !ok {
		return nil, nil
	}
	result := make([]npc.MenuEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	return result, nil
}

func (r *NPCRepository) FindActionResult(_ context.Context, entityID uint64, entryID string) (*npc.ActionResult, error) {
	entries, ok := npcMenuEntries[entityID]
	if !ok {
		return nil, nil
	}
	for _, entry := range entries {
		if entry.EntryID != entryID {
			continue
		}
		return &npc.ActionResult{EntityID: entityID, EntryID: entryID, ResultType: entry.ActionResultType, Notice: entry.ActionNotice}, nil
	}
	return nil, nil
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
