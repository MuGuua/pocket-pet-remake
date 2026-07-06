package monster

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// WildEncounterEntityIDBase 为暗雷虚拟 entity_id 段起点，避免与世界 NPC entity 冲突。
const WildEncounterEntityIDBase uint64 = 8800000000

// WildEncounterEntityID 根据 scene_id 生成暗雷战斗用的虚拟 entity_id。
func WildEncounterEntityID(sceneID uint32) uint64 {
	return WildEncounterEntityIDBase + uint64(sceneID)
}

var (
	ErrMonsterDefinitionNotFound           = errors.New("monster definition not found")
	ErrInvalidAdminMonsterDefinitionInput  = errors.New("invalid admin monster definition input")
	ErrMonsterDefinitionConflict           = errors.New("monster definition conflict")
	ErrMonsterEncounterNotFound            = errors.New("monster encounter not found")
	ErrInvalidAdminMonsterEncounterInput   = errors.New("invalid admin monster encounter input")
	ErrMonsterEncounterConflict            = errors.New("monster encounter conflict")
	ErrInvalidMonsterReference             = errors.New("invalid monster reference")
	ErrSceneWildEncounterNotFound          = errors.New("scene wild encounter not found")
	ErrInvalidAdminSceneWildEncounterInput = errors.New("invalid admin scene wild encounter input")
	ErrSceneWildEncounterConflict          = errors.New("scene wild encounter conflict")
	ErrMonsterNotCapturable                = errors.New("monster not capturable")
	ErrInvalidCapturePetReference          = errors.New("invalid capture pet reference")
)

type AdminDefinitionListQuery struct {
	MonsterID uint32
	Name      string
	Enabled   *bool
	Page      uint32
	PageSize  uint32
}

func (q AdminDefinitionListQuery) Normalize() AdminDefinitionListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Name = strings.TrimSpace(q.Name)
	return q
}

type AdminDefinitionSummary struct {
	MonsterID   uint32    `json:"monster_id"`
	MonsterName string    `json:"monster_name"`
	Level       uint32    `json:"level"`
	Quality     uint32    `json:"quality"`
	IsEnabled   bool      `json:"is_enabled"`
	StatusText  string    `json:"status_text"`
	SkinID      string    `json:"skin_id"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminDefinitionList struct {
	Items    []AdminDefinitionSummary `json:"items"`
	Total    uint64                   `json:"total"`
	Page     uint32                   `json:"page"`
	PageSize uint32                   `json:"page_size"`
}

type AdminDefinitionBaseStats struct {
	Level             uint32 `json:"level"`
	Quality           uint32 `json:"quality"`
	HP                uint32 `json:"hp"`
	HPMax             uint32 `json:"hp_max"`
	ATK               uint32 `json:"atk"`
	DEF               uint32 `json:"def"`
	SPD               uint32 `json:"spd"`
	MANA              uint32 `json:"mana"`
	Guard             uint32 `json:"guard"`
	TalentDmgPct      uint32 `json:"talent_dmg_pct"`
	TalentReducePct   uint32 `json:"talent_reduce_pct"`
	ElementAdvPct     uint32 `json:"element_adv_pct"`
	ElementPenaltyPct uint32 `json:"element_penalty_pct"`
}

type AdminDefinitionDetail struct {
	MonsterID       uint32                   `json:"monster_id"`
	MonsterName     string                   `json:"monster_name"`
	Description     string                   `json:"description"`
	IsEnabled       bool                     `json:"is_enabled"`
	StatusText      string                   `json:"status_text"`
	BaseStats       AdminDefinitionBaseStats `json:"base_stats"`
	SkillIDs        []uint32                 `json:"skill_ids"`
	SkinID          string                   `json:"skin_id"`
	IsCapturable    bool                     `json:"is_capturable"`
	CapturePetID    uint32                   `json:"capture_pet_id"`
	CaptureRateBase uint32                   `json:"capture_rate_base"`
	CaptureMinHPPct uint32                   `json:"capture_min_hp_pct"`
	CaptureItemIDs  []uint32                 `json:"capture_item_ids"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

type AdminUpsertDefinitionInput struct {
	MonsterID         uint32   `json:"monster_id"`
	MonsterName       string   `json:"monster_name"`
	Description       string   `json:"description"`
	IsEnabled         bool     `json:"is_enabled"`
	Level             uint32   `json:"level"`
	Quality           uint32   `json:"quality"`
	HP                uint32   `json:"hp"`
	HPMax             uint32   `json:"hp_max"`
	ATK               uint32   `json:"atk"`
	DEF               uint32   `json:"def"`
	SPD               uint32   `json:"spd"`
	MANA              uint32   `json:"mana"`
	Guard             uint32   `json:"guard"`
	TalentDmgPct      uint32   `json:"talent_dmg_pct"`
	TalentReducePct   uint32   `json:"talent_reduce_pct"`
	ElementAdvPct     uint32   `json:"element_adv_pct"`
	ElementPenaltyPct uint32   `json:"element_penalty_pct"`
	SkillIDs          []uint32 `json:"skill_ids"`
	SkinID            string   `json:"skin_id"`
	IsCapturable      bool     `json:"is_capturable"`
	CapturePetID      uint32   `json:"capture_pet_id"`
	CaptureRateBase   uint32   `json:"capture_rate_base"`
	CaptureMinHPPct   uint32   `json:"capture_min_hp_pct"`
	CaptureItemIDs    []uint32 `json:"capture_item_ids"`
}

func (input AdminUpsertDefinitionInput) Normalize() AdminUpsertDefinitionInput {
	input.MonsterName = strings.TrimSpace(input.MonsterName)
	input.Description = strings.TrimSpace(input.Description)
	if input.Level == 0 {
		input.Level = 1
	}
	if input.Quality == 0 {
		input.Quality = 1
	}
	if input.HP == 0 {
		input.HP = 1
	}
	if input.HPMax == 0 {
		input.HPMax = input.HP
	}
	if input.ATK == 0 {
		input.ATK = 1
	}
	if input.DEF == 0 {
		input.DEF = 1
	}
	if input.SPD == 0 {
		input.SPD = 1
	}
	if input.SkillIDs == nil {
		input.SkillIDs = []uint32{}
	}
	if input.CaptureItemIDs == nil {
		input.CaptureItemIDs = []uint32{}
	}
	input.SkinID = strings.TrimSpace(input.SkinID)
	if !input.IsCapturable {
		input.CapturePetID = 0
	}
	if input.CaptureRateBase == 0 {
		input.CaptureRateBase = 5000
	}
	if input.CaptureMinHPPct == 0 {
		input.CaptureMinHPPct = 30
	}
	return input
}

type AdminEncounterListQuery struct {
	EntityID uint64
	Name     string
	Enabled  *bool
	Page     uint32
	PageSize uint32
}

func (q AdminEncounterListQuery) Normalize() AdminEncounterListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Name = strings.TrimSpace(q.Name)
	return q
}

type AdminEncounterSummary struct {
	EntityID      uint64    `json:"entity_id"`
	EncounterName string    `json:"encounter_name"`
	SpawnCount    uint32    `json:"spawn_count"`
	IsEnabled     bool      `json:"is_enabled"`
	StatusText    string    `json:"status_text"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type AdminEncounterList struct {
	Items    []AdminEncounterSummary `json:"items"`
	Total    uint64                  `json:"total"`
	Page     uint32                  `json:"page"`
	PageSize uint32                  `json:"page_size"`
}

type AdminEncounterDetail struct {
	EntityID        uint64    `json:"entity_id"`
	EncounterName   string    `json:"encounter_name"`
	Description     string    `json:"description"`
	SpawnMonsterIDs []uint32  `json:"spawn_monster_ids"`
	IsEnabled       bool      `json:"is_enabled"`
	StatusText      string    `json:"status_text"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AdminUpsertEncounterInput struct {
	EntityID        uint64   `json:"entity_id"`
	EncounterName   string   `json:"encounter_name"`
	Description     string   `json:"description"`
	SpawnMonsterIDs []uint32 `json:"spawn_monster_ids"`
	IsEnabled       bool     `json:"is_enabled"`
}

func (input AdminUpsertEncounterInput) Normalize() AdminUpsertEncounterInput {
	input.EncounterName = strings.TrimSpace(input.EncounterName)
	input.Description = strings.TrimSpace(input.Description)
	if input.SpawnMonsterIDs == nil {
		input.SpawnMonsterIDs = []uint32{}
	}
	return input
}

type AdminWildEncounterListQuery struct {
	SceneID  uint32
	Name     string
	Enabled  *bool
	Page     uint32
	PageSize uint32
}

func (q AdminWildEncounterListQuery) Normalize() AdminWildEncounterListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Name = strings.TrimSpace(q.Name)
	return q
}

type AdminWildEncounterSummary struct {
	SceneID        uint32    `json:"scene_id"`
	EncounterName  string    `json:"encounter_name"`
	EncounterRate  uint32    `json:"encounter_rate"`
	SpawnCount     uint32    `json:"spawn_count"`
	FormationCount uint32    `json:"formation_count"`
	IsEnabled      bool      `json:"is_enabled"`
	StatusText     string    `json:"status_text"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type AdminWildEncounterList struct {
	Items    []AdminWildEncounterSummary `json:"items"`
	Total    uint64                      `json:"total"`
	Page     uint32                      `json:"page"`
	PageSize uint32                      `json:"page_size"`
}

type AdminWildEncounterMonsterSlot struct {
	MonsterID     uint32 `json:"monster_id"`
	RewardEnabled bool   `json:"reward_enabled"`
}

type AdminWildEncounterFormation struct {
	FormationName   string                          `json:"formation_name"`
	Weight          uint32                          `json:"weight"`
	SpawnMonsterIDs []uint32                        `json:"spawn_monster_ids"`
	MonsterSlots    []AdminWildEncounterMonsterSlot `json:"monster_slots"`
}

func (formation AdminWildEncounterFormation) Normalize(index int) AdminWildEncounterFormation {
	formation.FormationName = strings.TrimSpace(formation.FormationName)
	if formation.FormationName == "" {
		formation.FormationName = "编队" + strconv.Itoa(index+1)
	}
	if formation.Weight == 0 {
		formation.Weight = 1
	}
	if formation.SpawnMonsterIDs == nil {
		formation.SpawnMonsterIDs = []uint32{}
	}
	if len(formation.MonsterSlots) == 0 {
		formation.MonsterSlots = make([]AdminWildEncounterMonsterSlot, 0, len(formation.SpawnMonsterIDs))
		for _, monsterID := range formation.SpawnMonsterIDs {
			formation.MonsterSlots = append(formation.MonsterSlots, AdminWildEncounterMonsterSlot{MonsterID: monsterID, RewardEnabled: true})
		}
	}
	normalizedSlots := make([]AdminWildEncounterMonsterSlot, 0, len(formation.MonsterSlots))
	formation.SpawnMonsterIDs = make([]uint32, 0, len(formation.MonsterSlots))
	for _, slot := range formation.MonsterSlots {
		if slot.MonsterID == 0 {
			continue
		}
		normalizedSlots = append(normalizedSlots, AdminWildEncounterMonsterSlot{MonsterID: slot.MonsterID, RewardEnabled: slot.RewardEnabled})
		formation.SpawnMonsterIDs = append(formation.SpawnMonsterIDs, slot.MonsterID)
	}
	formation.MonsterSlots = normalizedSlots
	return formation
}

type AdminWildEncounterDetail struct {
	SceneID         uint32                        `json:"scene_id"`
	EncounterName   string                        `json:"encounter_name"`
	Description     string                        `json:"description"`
	EncounterRate   uint32                        `json:"encounter_rate"`
	SpawnMonsterIDs []uint32                      `json:"spawn_monster_ids"`
	Formations      []AdminWildEncounterFormation `json:"formations"`
	Rewards         []BattleRewardEntry           `json:"rewards"`
	IsEnabled       bool                          `json:"is_enabled"`
	StatusText      string                        `json:"status_text"`
	CreatedAt       time.Time                     `json:"created_at"`
	UpdatedAt       time.Time                     `json:"updated_at"`
}

type AdminUpsertWildEncounterInput struct {
	SceneID         uint32                        `json:"scene_id"`
	EncounterName   string                        `json:"encounter_name"`
	Description     string                        `json:"description"`
	EncounterRate   uint32                        `json:"encounter_rate"`
	SpawnMonsterIDs []uint32                      `json:"spawn_monster_ids"`
	Formations      []AdminWildEncounterFormation `json:"formations"`
	Rewards         []AdminBattleRewardInput      `json:"rewards"`
	IsEnabled       bool                          `json:"is_enabled"`
}

func (input AdminUpsertWildEncounterInput) Normalize() AdminUpsertWildEncounterInput {
	input.EncounterName = strings.TrimSpace(input.EncounterName)
	input.Description = strings.TrimSpace(input.Description)
	if input.SpawnMonsterIDs == nil {
		input.SpawnMonsterIDs = []uint32{}
	}
	if len(input.Formations) == 0 && len(input.SpawnMonsterIDs) > 0 {
		input.Formations = []AdminWildEncounterFormation{{FormationName: "默认编队", Weight: 10000, SpawnMonsterIDs: append([]uint32{}, input.SpawnMonsterIDs...)}}
	}
	for index, formation := range input.Formations {
		input.Formations[index] = formation.Normalize(index)
	}
	if input.Rewards == nil {
		input.Rewards = []AdminBattleRewardInput{}
	}
	for index, reward := range input.Rewards {
		input.Rewards[index] = reward.Normalize()
		if input.Rewards[index].SortOrder == 0 {
			input.Rewards[index].SortOrder = uint32(index + 1)
		}
	}
	input.SpawnMonsterIDs = flattenWildEncounterFormationMonsterIDs(input.Formations, input.SpawnMonsterIDs)
	return input
}

func flattenWildEncounterFormationMonsterIDs(formations []AdminWildEncounterFormation, fallback []uint32) []uint32 {
	result := make([]uint32, 0)
	seen := make(map[uint32]struct{})
	appendID := func(monsterID uint32) {
		if monsterID == 0 {
			return
		}
		if _, exists := seen[monsterID]; exists {
			return
		}
		seen[monsterID] = struct{}{}
		result = append(result, monsterID)
	}
	for _, formation := range formations {
		for _, monsterID := range formation.SpawnMonsterIDs {
			appendID(monsterID)
		}
	}
	if len(result) == 0 {
		for _, monsterID := range fallback {
			appendID(monsterID)
		}
	}
	return result
}

// RuntimeDefinition 是战斗运行时读取的怪物模板。
type RuntimeDefinition struct {
	MonsterID         uint32
	MonsterName       string
	Level             uint32
	Quality           uint32
	HP                uint32
	HPMax             uint32
	ATK               uint32
	DEF               uint32
	SPD               uint32
	MANA              uint32
	SkillIDs          []uint32
	SkinID            string
	Guard             uint32
	TalentDmgPct      uint32
	TalentReducePct   uint32
	ElementAdvPct     uint32
	ElementPenaltyPct uint32
}

// RuntimeEncounterSlot 描述一次遭遇中的一个怪物槽位。
type RuntimeEncounterSlot struct {
	MonsterID         uint32
	MonsterName       string
	Level             uint32
	HP                uint32
	HPMax             uint32
	ATK               uint32
	DEF               uint32
	SPD               uint32
	MANA              uint32
	SkillIDs          []uint32
	SkinID            string
	Guard             uint32
	TalentDmgPct      uint32
	TalentReducePct   uint32
	ElementAdvPct     uint32
	ElementPenaltyPct uint32
	RewardEnabled     bool
}

// RuntimeEncounter 是按 entity_id 解析后的完整遭遇。
type RuntimeEncounter struct {
	EntityID      uint64
	EncounterName string
	Slots         []RuntimeEncounterSlot
}

// RuntimeWildEncounterConfig 是下发给客户端的暗雷配置，不含完整怪物数值。
type RuntimeWildEncounterConfig struct {
	Enabled         bool
	SceneID         uint32
	EncounterRate   uint32
	SpawnMonsterIDs []uint32
}

// RuntimeWildEncounter 是服务端开战时解析后的暗雷遭遇。
type RuntimeWildEncounter struct {
	SceneID       uint32
	EncounterName string
	FormationName string
	Slots         []RuntimeEncounterSlot
	Rewards       []BattleRewardEntry
}

// CaptureConfig 描述怪物模板的捕捉发放配置。
type CaptureConfig struct {
	MonsterID       uint32
	IsCapturable    bool
	CapturePetID    uint32
	CaptureRateBase uint32
	CaptureMinHPPct uint32
	CaptureItemIDs  []uint32
}
