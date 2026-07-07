package pet

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrPetNotFound          = errors.New("pet not found")
	ErrInvalidLineup        = errors.New("invalid lineup")
	ErrDuplicateLineup      = errors.New("duplicate lineup pet")
	ErrInvalidAdminPetInput = errors.New("invalid admin pet input")
)

type Pet struct {
	PetUID   uint64
	PetID    uint32
	Level    uint32
	Exp      uint64
	Quality  uint32
	HP       uint32
	HPMax    uint32
	ATK      uint32
	DEF      uint32
	SPD      uint32
	MANA     uint32
	SkillIDs []uint32
	// SkillLoadout 分槽技能快照；SkillIDs 为战斗合并结果。
	SkillLoadout     SkillLoadout
	GrowthAptitudes  GrowthAptitudes
	GrantSource      string
	CaptureMonsterID uint32
	InLineup         bool
	// IsUsable 表示该实例对应的 pet_id 是否存在于启用中的系统宠物模板列表。
	IsUsable bool
	// FreeAttrPoints 是尚未分配的自由属性点。
	FreeAttrPoints uint32
	// AllocHPPoints 等五项是玩家已手动分配的自由点累计。
	AllocHPPoints   uint32
	AllocATKPoints  uint32
	AllocSPDPoints  uint32
	AllocMANAPoints uint32
	AllocDEFPoints  uint32
	// BaseHPApt 等五项是基础资质快照。
	BaseHPApt   uint32
	BaseATKApt  uint32
	BaseDEFApt  uint32
	BaseSPDApt  uint32
	BaseMANAApt uint32
	// ExtraHPApt 等五项是红色资质（提资超出基础部分）。
	ExtraHPApt      uint32
	ExtraATKApt     uint32
	ExtraDEFApt     uint32
	ExtraSPDApt     uint32
	ExtraMANAApt    uint32
	EvolutionLevel  uint32
	RebirthLevel    uint32
	AptitudeProfile string
	// ExpToNext 由成长服务按等级配置计算，仅运行态填充。
	ExpToNext uint64
	// LastLevelUpCount 记录最近一次经验结算连升次数，供战斗结算推送使用。
	LastLevelUpCount uint32
	// LastAttrPointsGained 记录最近一次经验结算获得的自由点。
	LastAttrPointsGained uint32
	// Spirit 是当前精力；SpiritMax 是精力上限。
	Spirit    uint32
	SpiritMax uint32
	// HitPct 等字段为宠物次要战斗属性，首期默认 0，由玩法/装备后续写入。
	HitPct                   uint32
	DodgePct                 uint32
	CritRatePct              uint32
	CritDmgPct               uint32
	PhysicalResistPct        uint32
	ReversePhysicalResistPct uint32
	SkillResistPct           uint32
	ReverseSkillResistPct    uint32
	ConfusionResistPct       uint32
	SleepResistPct           uint32
	ParalysisResistPct       uint32
	SealResistPct            uint32
	CurseResistPct           uint32
	CritDmgResistPct         uint32
	CritResistPct            uint32
	CharacterResistPct       uint32
	PetResistPct             uint32
	Guard                    uint32
	TalentDmgPct             uint32
	TalentReducePct          uint32
	ElementAdvPct            uint32
	ElementPenaltyPct        uint32
}

// RuntimeGrantResult 描述一次系统侧发放宠物后的结果。
// 当前供任务奖励、活动补偿等正式链路复用，确保奖励落库后可直接给客户端推送最新宠物快照。
type RuntimeGrantResult struct {
	Pet Pet `json:"pet"`
}

type LineupPet struct {
	PetUID                   uint64
	PetID                    uint32
	Level                    uint32
	HP                       uint32
	HPMax                    uint32
	ATK                      uint32
	DEF                      uint32
	SPD                      uint32
	MANA                     uint32
	Spirit                   uint32
	SpiritMax                uint32
	HitPct                   uint32
	DodgePct                 uint32
	CritRatePct              uint32
	CritDmgPct               uint32
	PhysicalResistPct        uint32
	ReversePhysicalResistPct uint32
	SkillResistPct           uint32
	ReverseSkillResistPct    uint32
	ConfusionResistPct       uint32
	SleepResistPct           uint32
	ParalysisResistPct       uint32
	SealResistPct            uint32
	CurseResistPct           uint32
	CritDmgResistPct         uint32
	CritResistPct            uint32
	CharacterResistPct       uint32
	PetResistPct             uint32
	Guard                    uint32
	TalentDmgPct             uint32
	TalentReducePct          uint32
	ElementAdvPct            uint32
	ElementPenaltyPct        uint32
	SkillIDs                 []uint32
}

// ToLineupPet 把完整宠物快照转换为战斗编队读取用的精简结构。
func ToLineupPet(item Pet) LineupPet {
	ResolvePetBattleSkills(&item)
	return LineupPet{
		PetUID:                   item.PetUID,
		PetID:                    item.PetID,
		Level:                    item.Level,
		HP:                       item.HP,
		HPMax:                    item.HPMax,
		ATK:                      item.ATK,
		DEF:                      item.DEF,
		SPD:                      item.SPD,
		MANA:                     item.MANA,
		Spirit:                   item.Spirit,
		SpiritMax:                item.SpiritMax,
		HitPct:                   item.HitPct,
		DodgePct:                 item.DodgePct,
		CritRatePct:              item.CritRatePct,
		CritDmgPct:               item.CritDmgPct,
		PhysicalResistPct:        item.PhysicalResistPct,
		ReversePhysicalResistPct: item.ReversePhysicalResistPct,
		SkillResistPct:           item.SkillResistPct,
		ReverseSkillResistPct:    item.ReverseSkillResistPct,
		ConfusionResistPct:       item.ConfusionResistPct,
		SleepResistPct:           item.SleepResistPct,
		ParalysisResistPct:       item.ParalysisResistPct,
		SealResistPct:            item.SealResistPct,
		CurseResistPct:           item.CurseResistPct,
		CritDmgResistPct:         item.CritDmgResistPct,
		CritResistPct:            item.CritResistPct,
		CharacterResistPct:       item.CharacterResistPct,
		PetResistPct:             item.PetResistPct,
		Guard:                    item.Guard,
		TalentDmgPct:             item.TalentDmgPct,
		TalentReducePct:          item.TalentReducePct,
		ElementAdvPct:            item.ElementAdvPct,
		ElementPenaltyPct:        item.ElementPenaltyPct,
		SkillIDs:                 append([]uint32{}, item.SkillIDs...),
	}
}

type AdminListQuery struct {
	PetUID   uint64
	PlayerID uint64
	PetID    uint32
	Page     uint32
	PageSize uint32
}

func (q AdminListQuery) Normalize() AdminListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}

// AdminGrantPetFromTemplateInput 描述运营按系统宠物模板给玩家发放初始宠物。
type AdminGrantPetFromTemplateInput struct {
	PlayerID uint64 `json:"player_id"`
	PetID    uint32 `json:"pet_id"`
}

type AdminCreatePetInput struct {
	PlayerID       uint64   `json:"player_id"`
	PetID          uint32   `json:"pet_id"`
	Level          uint32   `json:"level"`
	Exp            uint64   `json:"exp"`
	Quality        uint32   `json:"quality"`
	HP             uint32   `json:"hp"`
	HPMax          uint32   `json:"hp_max"`
	ATK            uint32   `json:"atk"`
	DEF            uint32   `json:"def"`
	SPD            uint32   `json:"spd"`
	MANA           uint32   `json:"mana"`
	SkillIDs       []uint32 `json:"skill_ids"`
	InnateSkillIDs []uint32 `json:"innate_skill_ids"`
	NormalSkillIDs []uint32 `json:"normal_skill_ids"`
	AdminPetCombatStats
}

func (input AdminCreatePetInput) Normalize() AdminCreatePetInput {
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
	return input
}

type AdminUpdatePetInput struct {
	PetID          uint32   `json:"pet_id"`
	CustomName     string   `json:"custom_name"`
	Level          uint32   `json:"level"`
	Exp            uint64   `json:"exp"`
	Quality        uint32   `json:"quality"`
	HP             uint32   `json:"hp"`
	HPMax          uint32   `json:"hp_max"`
	ATK            uint32   `json:"atk"`
	DEF            uint32   `json:"def"`
	SPD            uint32   `json:"spd"`
	MANA           uint32   `json:"mana"`
	SkillIDs       []uint32 `json:"skill_ids"`
	InnateSkillIDs []uint32 `json:"innate_skill_ids"`
	NormalSkillIDs []uint32 `json:"normal_skill_ids"`
	AdminPetCombatStats
}

func (input AdminUpdatePetInput) Normalize() AdminUpdatePetInput {
	input.CustomName = strings.TrimSpace(input.CustomName)
	if input.Level == 0 {
		input.Level = 1
	}
	if input.Quality == 0 {
		input.Quality = 1
	}
	if input.HPMax == 0 {
		input.HPMax = input.HP
	}
	return input
}

// AdminSetPetLineupInput 供运营后台提交玩家出战宠物，最多 1 只，也可为空。
type AdminSetPetLineupInput struct {
	PetUIDs []uint64 `json:"pet_uids"`
}

// AdminSetPetLineupResult 返回写入后的玩家 ID 与出战宠物 UID 列表。
type AdminSetPetLineupResult struct {
	PlayerID uint64   `json:"player_id"`
	PetUIDs  []uint64 `json:"pet_uids"`
}

type AdminPetSummary struct {
	PetUID     uint64    `json:"pet_uid"`
	PlayerID   uint64    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	PetID      uint32    `json:"pet_id"`
	PetName    string    `json:"pet_name"`
	CustomName string    `json:"custom_name"`
	Level      uint32    `json:"level"`
	Quality    uint32    `json:"quality"`
	HP         uint32    `json:"hp"`
	HPMax      uint32    `json:"hp_max"`
	ATK        uint32    `json:"atk"`
	DEF        uint32    `json:"def"`
	SPD        uint32    `json:"spd"`
	MANA       uint32    `json:"mana"`
	SkillIDs   []uint32  `json:"skill_ids,omitempty"`
	InLineup   bool      `json:"in_lineup"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type AdminPetList struct {
	Items    []AdminPetSummary `json:"items"`
	Total    uint64            `json:"total"`
	Page     uint32            `json:"page"`
	PageSize uint32            `json:"page_size"`
}

type AdminPetDetail struct {
	PetUID         uint64     `json:"pet_uid"`
	PlayerID       uint64     `json:"player_id"`
	PlayerName     string     `json:"player_name"`
	PetID          uint32     `json:"pet_id"`
	PetName        string     `json:"pet_name"`
	CustomName     string     `json:"custom_name"`
	Level          uint32     `json:"level"`
	Exp            uint64     `json:"exp"`
	Quality        uint32     `json:"quality"`
	HP             uint32     `json:"hp"`
	HPMax          uint32     `json:"hp_max"`
	ATK            uint32     `json:"atk"`
	DEF            uint32     `json:"def"`
	SPD            uint32     `json:"spd"`
	MANA           uint32     `json:"mana"`
	SkillIDs       []uint32   `json:"skill_ids"`
	InnateSkillIDs []uint32   `json:"innate_skill_ids"`
	NormalSkillIDs []uint32   `json:"normal_skill_ids"`
	InLineup       bool       `json:"in_lineup"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	AdminPetCombatStats
}

// BuildAdminBattleSkillIDs 统一把后台宠物编辑页提交的结构化技能槽合并成兼容 skill_ids。
func BuildAdminBattleSkillIDs(innateSkillIDs, normalSkillIDs, legacySkillIDs []uint32) []uint32 {
	loadout := SkillLoadoutFromDefinition(innateSkillIDs, normalSkillIDs)
	if len(innateSkillIDs) == 0 && len(normalSkillIDs) == 0 {
		return append([]uint32{}, legacySkillIDs...)
	}
	return MergeBattleSkillIDs(loadout, legacySkillIDs)
}

func NormalizeSkillIDs(raw string) []uint32 {
	parts := strings.Split(raw, ",")
	result := make([]uint32, 0, len(parts))
	for _, part := range parts {
		_ = part
	}
	return result
}
