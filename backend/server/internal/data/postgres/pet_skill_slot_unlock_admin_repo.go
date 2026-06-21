package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/pet"
)

const listAdminPetSkillSlotUnlockItemsQuery = `
SELECT slot_key, item_id, description, status, created_at, updated_at
FROM pet_skill_slot_unlock_item
ORDER BY slot_key ASC
`

const findAdminPetSkillSlotUnlockItemQuery = `
SELECT slot_key, item_id, description, status, created_at, updated_at
FROM pet_skill_slot_unlock_item
WHERE slot_key = $1
LIMIT 1
`

const insertAdminPetSkillSlotUnlockItemQuery = `
INSERT INTO pet_skill_slot_unlock_item (slot_key, item_id, description, status)
VALUES ($1, $2, $3, $4)
`

const updateAdminPetSkillSlotUnlockItemQuery = `
UPDATE pet_skill_slot_unlock_item
SET item_id = $2,
    description = $3,
    status = $4
WHERE slot_key = $1
`

const deleteAdminPetSkillSlotUnlockItemQuery = `
DELETE FROM pet_skill_slot_unlock_item
WHERE slot_key = $1
`

// ListAdminPetSkillSlotUnlockItems 返回全部神符槽解锁道具映射。
func (r *PetRepository) ListAdminPetSkillSlotUnlockItems(ctx context.Context) ([]pet.AdminPetSkillSlotUnlockItem, error) {
	rows, err := r.db.QueryContext(ctx, listAdminPetSkillSlotUnlockItemsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]pet.AdminPetSkillSlotUnlockItem, 0)
	for rows.Next() {
		item, err := scanAdminPetSkillSlotUnlockItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// FindAdminPetSkillSlotUnlockItem 按 slot_key 读取单条配置。
func (r *PetRepository) FindAdminPetSkillSlotUnlockItem(ctx context.Context, slotKey string) (*pet.AdminPetSkillSlotUnlockItem, error) {
	row := r.db.QueryRowContext(ctx, findAdminPetSkillSlotUnlockItemQuery, slotKey)
	item, err := scanAdminPetSkillSlotUnlockItemRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateAdminPetSkillSlotUnlockItem 新增解锁映射。
func (r *PetRepository) CreateAdminPetSkillSlotUnlockItem(ctx context.Context, input pet.AdminUpsertPetSkillSlotUnlockInput) (*pet.AdminPetSkillSlotUnlockItem, error) {
	if _, err := r.db.ExecContext(ctx, insertAdminPetSkillSlotUnlockItemQuery, input.SlotKey, input.ItemID, input.Description, input.Status); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, pet.ErrPetSkillSlotUnlockConflict
		}
		return nil, err
	}
	return r.FindAdminPetSkillSlotUnlockItem(ctx, input.SlotKey)
}

// UpdateAdminPetSkillSlotUnlockItem 更新解锁映射。
func (r *PetRepository) UpdateAdminPetSkillSlotUnlockItem(ctx context.Context, slotKey string, input pet.AdminUpsertPetSkillSlotUnlockInput) (*pet.AdminPetSkillSlotUnlockItem, error) {
	result, err := r.db.ExecContext(ctx, updateAdminPetSkillSlotUnlockItemQuery, slotKey, input.ItemID, input.Description, input.Status)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, pet.ErrPetSkillSlotUnlockNotFound
	}
	return r.FindAdminPetSkillSlotUnlockItem(ctx, slotKey)
}

// DeleteAdminPetSkillSlotUnlockItem 删除解锁映射。
func (r *PetRepository) DeleteAdminPetSkillSlotUnlockItem(ctx context.Context, slotKey string) error {
	result, err := r.db.ExecContext(ctx, deleteAdminPetSkillSlotUnlockItemQuery, slotKey)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return pet.ErrPetSkillSlotUnlockNotFound
	}
	return nil
}

func scanAdminPetSkillSlotUnlockItem(rows *sql.Rows) (pet.AdminPetSkillSlotUnlockItem, error) {
	var (
		item     pet.AdminPetSkillSlotUnlockItem
		itemID   int64
		status   int64
	)
	if err := rows.Scan(&item.SlotKey, &itemID, &item.Description, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return pet.AdminPetSkillSlotUnlockItem{}, err
	}
	item.ItemID = uint32(itemID)
	item.Status = uint32(status)
	item.StatusText = adminStatusText(item.Status)
	return item, nil
}

func scanAdminPetSkillSlotUnlockItemRow(row *sql.Row) (pet.AdminPetSkillSlotUnlockItem, error) {
	var (
		item   pet.AdminPetSkillSlotUnlockItem
		itemID int64
		status int64
	)
	if err := row.Scan(&item.SlotKey, &itemID, &item.Description, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return pet.AdminPetSkillSlotUnlockItem{}, err
	}
	item.ItemID = uint32(itemID)
	item.Status = uint32(status)
	item.StatusText = adminStatusText(item.Status)
	return item, nil
}

func adminStatusText(status uint32) string {
	if status == 1 {
		return "启用"
	}
	return "停用"
}
