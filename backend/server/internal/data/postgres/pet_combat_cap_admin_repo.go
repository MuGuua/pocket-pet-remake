package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pocket-pet-remake/server/internal/module/pet"
)

const listAdminPetCombatStatCapsQuery = `
SELECT stat_key, cap_value, description, status, created_at, updated_at
FROM pet_combat_stat_cap
ORDER BY stat_key ASC
`

const findAdminPetCombatStatCapQuery = `
SELECT stat_key, cap_value, description, status, created_at, updated_at
FROM pet_combat_stat_cap
WHERE stat_key = $1
LIMIT 1
`

const updateAdminPetCombatStatCapQuery = `
UPDATE pet_combat_stat_cap
SET cap_value = $2,
    description = $3,
    status = $4
WHERE stat_key = $1
`

const listEnabledPetCombatStatCapsQuery = `
SELECT stat_key, cap_value
FROM pet_combat_stat_cap
WHERE status = 1
`

// ListAdminPetCombatStatCaps 返回全部封顶配置，供运营后台列表展示。
func (r *PetRepository) ListAdminPetCombatStatCaps(ctx context.Context) ([]pet.AdminPetCombatStatCap, error) {
	rows, err := r.db.QueryContext(ctx, listAdminPetCombatStatCapsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]pet.AdminPetCombatStatCap, 0)
	for rows.Next() {
		item, err := scanAdminPetCombatStatCapRow(rows)
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

// FindAdminPetCombatStatCap 按 stat_key 读取单条封顶配置。
func (r *PetRepository) FindAdminPetCombatStatCap(ctx context.Context, statKey string) (*pet.AdminPetCombatStatCap, error) {
	row := r.db.QueryRowContext(ctx, findAdminPetCombatStatCapQuery, statKey)
	item, err := scanAdminPetCombatStatCapRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateAdminPetCombatStatCap 更新封顶值、说明与启用状态。
func (r *PetRepository) UpdateAdminPetCombatStatCap(ctx context.Context, statKey string, input pet.AdminUpsertPetCombatStatCapInput) (*pet.AdminPetCombatStatCap, error) {
	result, err := r.db.ExecContext(ctx, updateAdminPetCombatStatCapQuery, statKey, input.CapValue, input.Description, input.Status)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, pet.ErrPetCombatStatCapNotFound
	}
	return r.FindAdminPetCombatStatCap(ctx, statKey)
}

// LoadCombatStatCaps 从数据库读取启用中的封顶表；无数据时回退默认种子。
func (r *PetRepository) LoadCombatStatCaps(ctx context.Context) (pet.CombatStatCaps, error) {
	rows, err := r.db.QueryContext(ctx, listEnabledPetCombatStatCapsQuery)
	if err != nil {
		return pet.DefaultCombatStatCaps(), err
	}
	defer rows.Close()

	values := make(map[pet.CombatStatCapKey]uint32)
	for rows.Next() {
		var statKey string
		var capValue int64
		if err := rows.Scan(&statKey, &capValue); err != nil {
			return pet.DefaultCombatStatCaps(), err
		}
		key := pet.CombatStatCapKey(statKey)
		if capValue < 0 {
			continue
		}
		values[key] = uint32(capValue)
	}
	if err := rows.Err(); err != nil {
		return pet.DefaultCombatStatCaps(), err
	}
	if len(values) == 0 {
		return pet.DefaultCombatStatCaps(), nil
	}
	return pet.MergeCombatStatCaps(values), nil
}

type adminPetCombatStatCapScanner interface {
	Scan(dest ...any) error
}

func scanAdminPetCombatStatCapRow(scanner adminPetCombatStatCapScanner) (pet.AdminPetCombatStatCap, error) {
	var item pet.AdminPetCombatStatCap
	var capValue int64
	var status int16
	if err := scanner.Scan(&item.StatKey, &capValue, &item.Description, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return pet.AdminPetCombatStatCap{}, fmt.Errorf("scan pet combat stat cap: %w", err)
	}
	item.CapValue = uint32(capValue)
	item.Status = status
	item.StatusText = adminPetCombatStatCapStatusText(status)
	return item, nil
}

func adminPetCombatStatCapStatusText(status int16) string {
	if status == 1 {
		return "启用"
	}
	return "停用"
}
