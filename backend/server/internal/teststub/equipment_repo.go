package teststub

import (
	"context"
	"sync"
	"time"

	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/player"
)

// NewEquipmentRepository 提供后台装备模板 CRUD 的内存桩。
func NewEquipmentRepository() *EquipmentRepository {
	return &EquipmentRepository{items: map[uint64]equipment.AdminEquipmentDetail{}}
}

// EquipmentRepository 是装备模板 Admin 仓储内存实现。
type EquipmentRepository struct {
	mu    sync.RWMutex
	items map[uint64]equipment.AdminEquipmentDetail
}

func (r *EquipmentRepository) ListForAdmin(_ context.Context, query equipment.AdminListQuery) (*equipment.AdminEquipmentList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query = query.Normalize()
	items := make([]equipment.AdminEquipmentSummary, 0, len(r.items))
	for _, current := range r.items {
		if query.ItemID > 0 && current.ItemID != query.ItemID {
			continue
		}
		if query.EquipSlot != "" && current.EquipSlot != query.EquipSlot {
			continue
		}
		items = append(items, equipment.AdminEquipmentSummary{
			ItemID:          current.ItemID,
			ItemCode:        current.ItemCode,
			ItemName:        current.ItemName,
			EquipSlot:       current.EquipSlot,
			EquipSlotLabel:  current.EquipSlotLabel,
			RequiredLevel:   current.RequiredLevel,
			Quality:         current.Quality,
			CanEnhance:      current.CanEnhance,
			MaxEnhanceLevel: current.MaxEnhanceLevel,
			SetID:           current.SetID,
			IsEnabled:       current.IsEnabled,
			UpdatedAt:       current.UpdatedAt,
			CreatedAt:       current.CreatedAt,
		})
	}
	return &equipment.AdminEquipmentList{Items: items, Total: uint64(len(items)), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *EquipmentRepository) FindForAdminByItemID(_ context.Context, itemID uint64) (*equipment.AdminEquipmentDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.items[itemID]
	if !ok {
		return nil, nil
	}
	copied := current
	return &copied, nil
}

func (r *EquipmentRepository) CreateForAdmin(_ context.Context, input equipment.AdminUpsertEquipmentInput) (*equipment.AdminEquipmentDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[input.ItemID]; ok {
		return nil, equipment.ErrEquipmentDefinitionConflict
	}
	now := time.Now()
	detail := buildStubEquipmentDetail(input, now, now)
	r.items[input.ItemID] = detail
	copied := detail
	return &copied, nil
}

func (r *EquipmentRepository) UpdateForAdmin(_ context.Context, itemID uint64, input equipment.AdminUpsertEquipmentInput) (*equipment.AdminEquipmentDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[itemID]
	if !ok {
		return nil, nil
	}
	updated := buildStubEquipmentDetail(input, current.CreatedAt, time.Now())
	updated.ItemID = itemID
	r.items[itemID] = updated
	copied := updated
	return &copied, nil
}

func (r *EquipmentRepository) DeleteForAdmin(_ context.Context, itemID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.items[itemID]
	if !ok {
		return equipment.ErrEquipmentDefinitionNotFound
	}
	current.IsEnabled = false
	r.items[itemID] = current
	return nil
}

func (r *EquipmentRepository) ListEquipped(_ context.Context, _ uint64) ([]equipment.RuntimeEquippedItem, error) {
	return []equipment.RuntimeEquippedItem{}, nil
}

func (r *EquipmentRepository) EquipFromBagSlot(_ context.Context, _ uint64, _ string, _ uint32, _ equipment.RecalcContext, _ *player.Profile) (*equipment.EquipFromBagResult, error) {
	return nil, equipment.ErrEquipmentBagItemInvalid
}

func (r *EquipmentRepository) UnequipSlot(_ context.Context, _ uint64, _ string, _ string, _ equipment.RecalcContext, _ *player.Profile) (*equipment.UnequipSlotResult, error) {
	return nil, equipment.ErrEquipmentSlotEmpty
}

func (r *EquipmentRepository) EnhanceInstance(_ context.Context, _ uint64, _ string) (*equipment.EnhanceResult, error) {
	return nil, equipment.ErrEquipmentNotFound
}

func buildStubEquipmentDetail(input equipment.AdminUpsertEquipmentInput, createdAt, updatedAt time.Time) equipment.AdminEquipmentDetail {
	input = input.Normalize()
	return equipment.AdminEquipmentDetail{
		ItemID:               input.ItemID,
		ItemCode:             input.ItemCode,
		ItemName:             input.ItemName,
		Desc:                 input.Desc,
		Icon:                 input.Icon,
		Quality:              input.Quality,
		Rarity:               input.Rarity,
		RequiredLevel:        input.RequiredLevel,
		BindType:             input.BindType,
		CanSell:              input.CanSell,
		CanStore:             input.CanStore,
		IsEnabled:            input.IsEnabled,
		EquipSlot:            input.EquipSlot,
		EquipSlotLabel:       equipment.EquipSlotLabel(equipment.EquipSlot(input.EquipSlot)),
		CareerLimit:          input.CareerLimit,
		CanEnhance:           input.CanEnhance,
		MaxEnhanceLevel:      input.MaxEnhanceLevel,
		SetID:                input.SetID,
		AppearanceSkinID:     input.AppearanceSkinID,
		AppearanceOnly:       input.AppearanceOnly,
		BaseHP:               input.BaseHP,
		BaseMana:             input.BaseMana,
		BaseATK:              input.BaseATK,
		BaseDEF:              input.BaseDEF,
		BaseSPD:              input.BaseSPD,
		CombatStats:          input.CombatStats,
		EnhancePerLevelStats: input.EnhancePerLevelStats,
		SocketCount:          input.SocketCount,
		AllowedGemTypes:      input.AllowedGemTypes,
		MedicinePouch:        input.MedicinePouch,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}
}
