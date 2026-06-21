package pet

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrPetSkillSlotUnlockNotFound 表示神符槽解锁配置不存在。
	ErrPetSkillSlotUnlockNotFound = errors.New("pet skill slot unlock config not found")
	// ErrInvalidPetSkillSlotUnlockInput 表示后台提交的神符槽解锁配置非法。
	ErrInvalidPetSkillSlotUnlockInput = errors.New("invalid pet skill slot unlock input")
	// ErrPetSkillSlotUnlockConflict 表示 slot_key 已存在。
	ErrPetSkillSlotUnlockConflict = errors.New("pet skill slot unlock config conflict")
)

// AdminPetSkillSlotUnlockItem 描述一条道具与神符槽的解锁映射。
type AdminPetSkillSlotUnlockItem struct {
	SlotKey     string    `json:"slot_key"`
	ItemID      uint32    `json:"item_id"`
	Description string    `json:"description"`
	Status      uint32    `json:"status"`
	StatusText  string    `json:"status_text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AdminUpsertPetSkillSlotUnlockInput 供后台新增或编辑解锁映射。
type AdminUpsertPetSkillSlotUnlockInput struct {
	SlotKey     string `json:"slot_key"`
	ItemID      uint32 `json:"item_id"`
	Description string `json:"description"`
	Status      uint32 `json:"status"`
}

// Normalize 校验并规范化后台输入。
func (input AdminUpsertPetSkillSlotUnlockInput) Normalize() (AdminUpsertPetSkillSlotUnlockInput, error) {
	input.SlotKey = strings.TrimSpace(input.SlotKey)
	input.Description = strings.TrimSpace(input.Description)
	if input.SlotKey == "" || input.ItemID == 0 {
		return AdminUpsertPetSkillSlotUnlockInput{}, ErrInvalidPetSkillSlotUnlockInput
	}
	if _, err := ResolveTalismanSlotColumns(input.SlotKey); err != nil {
		return AdminUpsertPetSkillSlotUnlockInput{}, ErrInvalidPetSkillSlotUnlockInput
	}
	if input.Status == 0 {
		input.Status = 1
	}
	return input, nil
}
