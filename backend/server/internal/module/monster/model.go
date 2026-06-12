package monster

import (
	"errors"
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
	ErrMonsterDefinitionNotFound = errors.New("monster definition not found")
	ErrInvalidAdminMonsterDefinitionInput = errors.New("invalid admin monster definition input")
	ErrMonsterDefinitionConflict = errors.New("monster definition conflict")
	ErrMonsterEncounterNotFound = errors.New("monster encounter not found")
	ErrInvalidAdminMonsterEncounterInput = errors.New("invalid admin monster encounter input")
	ErrMonsterEncounterConflict = errors.New("monster encounter conflict")
	ErrInvalidMonsterReference = errors.New("invalid monster reference")
	ErrSceneWildEncounterNotFound = errors.New("scene wild encounter not found")
	ErrInvalidAdminSceneWildEncounterInput = errors.New("invalid admin scene wild encounter input")
	ErrSceneWildEncounterConflict = errors.New("scene wild encounter conflict")
	ErrMonsterNotCapturable = errors.New("monster not capturable")
	ErrInvalidCapturePetReference = errors.New("invalid capture pet reference")
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
	Level   uint32 `json:"level"`
	Quality uint32 `json:"quality"`
	HP      uint32 `json:"hp"`
	HPMax   uint32 `json:"hp_max"`
	ATK     uint32 `json:"atk"`
	DEF     uint32 `json:"def"`
	SPD     uint32 `json:"spd"`
	MANA    uint32 `json:"mana"`
}

type AdminDefinitionDetail struct {
	MonsterID       uint32                   `json:"monster_id"`
	MonsterName     string                   `json:"monster_name"`
	Description     string                   `json:"description"`
	IsEnabled       bool                     `json:"is_enabled"`
	StatusText      string                   `json:"status_text"`
	BaseStats       AdminDefinitionBaseStats `json:"base_stats"`
	SkillIDs        []uint32                 `json:"skill_ids"`
	IsCapturable    bool                     `json:"is_capturable"`
	CapturePetID    uint32                   `json:"capture_pet_id"`
	CaptureRateBase uint32                   `json:"capture_rate_base"`
	CaptureMinHPPct uint32                   `json:"capture_min_hp_pct"`
	CaptureItemIDs  []uint32                 `json:"capture_item_ids"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

type AdminUpsertDefinitionInput struct {
	MonsterID       uint32   `json:"monster_id"`
	MonsterName     string   `json:"monster_name"`
	Description     string   `json:"description"`
	IsEnabled       bool     `json:"is_enabled"`
	Level           uint32   `json:"level"`
	Quality         uint32   `json:"quality"`
	HP              uint32   `json:"hp"`
	HPMax           uint32   `json:"hp_max"`
	ATK             uint32   `json:"atk"`
	DEF             uint32   `json:"def"`
	SPD             uint32   `json:"spd"`
	MANA            uint32   `json:"mana"`
	SkillIDs        []uint32 `json:"skill_ids"`
	IsCapturable    bool     `json:"is_capturable"`
	CapturePetID    uint32   `json:"capture_pet_id"`
	CaptureRateBase uint32   `json:"capture_rate_base"`
	CaptureMinHPPct uint32   `json:"capture_min_hp_pct"`
	CaptureItemIDs  []uint32 `json:"capture_item_ids"`
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
	EntityID       uint64    `json:"entity_id"`
	EncounterName  string    `json:"encounter_name"`
	SpawnCount     uint32    `json:"spawn_count"`
	IsEnabled      bool      `json:"is_enabled"`
	StatusText     string    `json:"status_text"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
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
	SceneID       uint32    `json:"scene_id"`
	EncounterName string    `json:"encounter_name"`
	EncounterRate uint32    `json:"encounter_rate"`
	SpawnCount    uint32    `json:"spawn_count"`
	IsEnabled     bool      `json:"is_enabled"`
	StatusText    string    `json:"status_text"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type AdminWildEncounterList struct {
	Items    []AdminWildEncounterSummary `json:"items"`
	Total    uint64                      `json:"total"`
	Page     uint32                      `json:"page"`
	PageSize uint32                      `json:"page_size"`
}

type AdminWildEncounterDetail struct {
	SceneID         uint32    `json:"scene_id"`
	EncounterName   string    `json:"encounter_name"`
	Description     string    `json:"description"`
	EncounterRate   uint32    `json:"encounter_rate"`
	SpawnMonsterIDs []uint32  `json:"spawn_monster_ids"`
	IsEnabled       bool      `json:"is_enabled"`
	StatusText      string    `json:"status_text"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AdminUpsertWildEncounterInput struct {
	SceneID         uint32   `json:"scene_id"`
	EncounterName   string   `json:"encounter_name"`
	Description     string   `json:"description"`
	EncounterRate   uint32   `json:"encounter_rate"`
	SpawnMonsterIDs []uint32 `json:"spawn_monster_ids"`
	IsEnabled       bool     `json:"is_enabled"`
}

func (input AdminUpsertWildEncounterInput) Normalize() AdminUpsertWildEncounterInput {
	input.EncounterName = strings.TrimSpace(input.EncounterName)
	input.Description = strings.TrimSpace(input.Description)
	if input.SpawnMonsterIDs == nil {
		input.SpawnMonsterIDs = []uint32{}
	}
	return input
}

// RuntimeDefinition 是战斗运行时读取的怪物模板。
type RuntimeDefinition struct {
	MonsterID   uint32
	MonsterName string
	Level       uint32
	Quality     uint32
	HP          uint32
	HPMax       uint32
	ATK         uint32
	DEF         uint32
	SPD         uint32
	MANA        uint32
	SkillIDs    []uint32
}

// RuntimeEncounterSlot 描述一次遭遇中的一个怪物槽位。
type RuntimeEncounterSlot struct {
	MonsterID   uint32
	MonsterName string
	Level       uint32
	HP          uint32
	HPMax       uint32
	ATK         uint32
	DEF         uint32
	SPD         uint32
	MANA        uint32
	SkillIDs    []uint32
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
	Slots         []RuntimeEncounterSlot
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
