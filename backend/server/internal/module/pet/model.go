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
	SkillIDs         []uint32
	GrowthAptitudes  GrowthAptitudes
	GrantSource      string
	CaptureMonsterID uint32
	InLineup         bool
	// IsUsable 表示该实例对应的 pet_id 是否存在于启用中的系统宠物模板列表。
	IsUsable bool
}

// RuntimeGrantResult 描述一次系统侧发放宠物后的结果。
// 当前供任务奖励、活动补偿等正式链路复用，确保奖励落库后可直接给客户端推送最新宠物快照。
type RuntimeGrantResult struct {
	Pet Pet `json:"pet"`
}

type LineupPet struct {
	PetUID   uint64
	PetID    uint32
	Level    uint32
	HP       uint32
	HPMax    uint32
	ATK      uint32
	DEF      uint32
	SPD      uint32
	MANA     uint32
	SkillIDs []uint32
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

type AdminCreatePetInput struct {
	PlayerID uint64   `json:"player_id"`
	PetID    uint32   `json:"pet_id"`
	Level    uint32   `json:"level"`
	Exp      uint64   `json:"exp"`
	Quality  uint32   `json:"quality"`
	HP       uint32   `json:"hp"`
	HPMax    uint32   `json:"hp_max"`
	ATK      uint32   `json:"atk"`
	DEF      uint32   `json:"def"`
	SPD      uint32   `json:"spd"`
	MANA     uint32   `json:"mana"`
	SkillIDs []uint32 `json:"skill_ids"`
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
	PetID    uint32   `json:"pet_id"`
	Level    uint32   `json:"level"`
	Exp      uint64   `json:"exp"`
	Quality  uint32   `json:"quality"`
	HP       uint32   `json:"hp"`
	HPMax    uint32   `json:"hp_max"`
	ATK      uint32   `json:"atk"`
	DEF      uint32   `json:"def"`
	SPD      uint32   `json:"spd"`
	MANA     uint32   `json:"mana"`
	SkillIDs []uint32 `json:"skill_ids"`
}

func (input AdminUpdatePetInput) Normalize() AdminUpdatePetInput {
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
	Level      uint32    `json:"level"`
	Quality    uint32    `json:"quality"`
	HP         uint32    `json:"hp"`
	HPMax      uint32    `json:"hp_max"`
	ATK        uint32    `json:"atk"`
	DEF        uint32    `json:"def"`
	SPD        uint32    `json:"spd"`
	MANA       uint32    `json:"mana"`
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
	PetUID     uint64     `json:"pet_uid"`
	PlayerID   uint64     `json:"player_id"`
	PlayerName string     `json:"player_name"`
	PetID      uint32     `json:"pet_id"`
	Level      uint32     `json:"level"`
	Exp        uint64     `json:"exp"`
	Quality    uint32     `json:"quality"`
	HP         uint32     `json:"hp"`
	HPMax      uint32     `json:"hp_max"`
	ATK        uint32     `json:"atk"`
	DEF        uint32     `json:"def"`
	SPD        uint32     `json:"spd"`
	MANA       uint32     `json:"mana"`
	SkillIDs   []uint32   `json:"skill_ids"`
	InLineup   bool       `json:"in_lineup"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func NormalizeSkillIDs(raw string) []uint32 {
	parts := strings.Split(raw, ",")
	result := make([]uint32, 0, len(parts))
	for _, part := range parts {
		_ = part
	}
	return result
}
