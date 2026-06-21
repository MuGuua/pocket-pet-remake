package pet

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrInvalidPetCombatStatCapInput 表示后台提交的封顶配置字段非法。
	ErrInvalidPetCombatStatCapInput = errors.New("invalid pet combat stat cap input")
	// ErrPetCombatStatCapNotFound 表示 stat_key 不存在。
	ErrPetCombatStatCapNotFound = errors.New("pet combat stat cap not found")
)

// AdminPetCombatStatCap 描述一条宠物战斗属性封顶配置。
type AdminPetCombatStatCap struct {
	StatKey     string    `json:"stat_key"`
	CapValue    uint32    `json:"cap_value"`
	Description string    `json:"description"`
	Status      int16     `json:"status"`
	StatusText  string    `json:"status_text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AdminUpsertPetCombatStatCapInput 供运营更新封顶值与说明。
type AdminUpsertPetCombatStatCapInput struct {
	CapValue    uint32 `json:"cap_value"`
	Description string `json:"description"`
	Status      int16  `json:"status"`
}

func (input AdminUpsertPetCombatStatCapInput) Normalize() AdminUpsertPetCombatStatCapInput {
	input.Description = strings.TrimSpace(input.Description)
	if input.Status != 0 && input.Status != 1 {
		input.Status = 1
	}
	return input
}

func validateAdminPetCombatStatCapInput(statKey string, input AdminUpsertPetCombatStatCapInput) error {
	normalizedKey := CombatStatCapKey(strings.TrimSpace(statKey))
	if !isKnownCombatStatCapKey(normalizedKey) {
		return ErrInvalidPetCombatStatCapInput
	}
	return nil
}

func isKnownCombatStatCapKey(key CombatStatCapKey) bool {
	_, ok := DefaultCombatStatCaps().values[key]
	return ok
}

func adminPetCombatStatCapStatusText(status int16) string {
	if status == 1 {
		return "启用"
	}
	return "停用"
}
