package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/bag"
)

// BagRepository 把后台背包 CRUD 映射到 player_item 持久化表。
// 这样所有后台改动都会直接进入数据库，不会只停留在服务端运行态。
type BagRepository struct {
	db DBTX
}

func NewBagRepository(db DBTX) *BagRepository {
	return &BagRepository{db: db}
}

const adminBagListBaseQuery = `
SELECT
  pi.id,
  pi.player_id,
  p.name,
  pi.item_id,
  pi.count,
  pi.created_at,
  pi.updated_at
FROM player_item pi
JOIN player p ON p.id = pi.player_id
`

const adminBagDetailQuery = `
SELECT
  pi.id,
  pi.player_id,
  p.name,
  pi.item_id,
  pi.count,
  pi.created_at,
  pi.updated_at
FROM player_item pi
JOIN player p ON p.id = pi.player_id
WHERE pi.id = $1
LIMIT 1
`

const adminBagPlayerExistsQuery = `
SELECT COUNT(1)
FROM player
WHERE id = $1 AND status = 1
`

const insertAdminBagItemQuery = `
INSERT INTO player_item (
  player_id,
  item_id,
  count
) VALUES ($1, $2, $3)
RETURNING id
`

const updateAdminBagItemQuery = `
UPDATE player_item
SET player_id = $2,
    item_id = $3,
    count = $4
WHERE id = $1
`

const deleteAdminBagItemQuery = `
DELETE FROM player_item
WHERE id = $1
`

func (r *BagRepository) ListForAdmin(ctx context.Context, query bag.AdminListQuery) (*bag.AdminItemList, error) {
	query = query.Normalize()

	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if query.RecordID > 0 {
		conditions = append(conditions, "pi.id = "+nextArg(query.RecordID))
	}
	if query.PlayerID > 0 {
		conditions = append(conditions, "pi.player_id = "+nextArg(query.PlayerID))
	}
	if query.ItemID > 0 {
		conditions = append(conditions, "pi.item_id = "+nextArg(query.ItemID))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := `
SELECT COUNT(1)
FROM player_item pi
JOIN player p ON p.id = pi.player_id
` + whereClause

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminBagListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY pi.id DESC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]bag.AdminItemSummary, 0)
	for rows.Next() {
		item, err := scanAdminBagSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &bag.AdminItemList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *BagRepository) FindAdminDetailByRecordID(ctx context.Context, recordID uint64) (*bag.AdminItemDetail, error) {
	item, err := scanAdminBagDetailRow(r.db.QueryRowContext(ctx, adminBagDetailQuery, recordID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *BagRepository) CreateForAdmin(ctx context.Context, input bag.AdminCreateItemInput) (*bag.AdminItemDetail, error) {
	if ok, err := r.playerExists(ctx, input.PlayerID); err != nil {
		return nil, err
	} else if !ok {
		return nil, bag.ErrBagItemNotFound
	}

	var recordID int64
	if err := r.db.QueryRowContext(ctx, insertAdminBagItemQuery, input.PlayerID, input.ItemID, input.Count).Scan(&recordID); err != nil {
		if isPlayerItemUniqueViolation(err) {
			return nil, bag.ErrBagItemConflict
		}
		return nil, err
	}
	return r.FindAdminDetailByRecordID(ctx, uint64(recordID))
}

func (r *BagRepository) UpdateForAdmin(ctx context.Context, recordID uint64, input bag.AdminUpdateItemInput) (*bag.AdminItemDetail, error) {
	if ok, err := r.playerExists(ctx, input.PlayerID); err != nil {
		return nil, err
	} else if !ok {
		return nil, bag.ErrBagItemNotFound
	}

	result, err := r.db.ExecContext(ctx, updateAdminBagItemQuery, recordID, input.PlayerID, input.ItemID, input.Count)
	if err != nil {
		if isPlayerItemUniqueViolation(err) {
			return nil, bag.ErrBagItemConflict
		}
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, bag.ErrBagItemNotFound
	}
	return r.FindAdminDetailByRecordID(ctx, recordID)
}

func (r *BagRepository) DeleteForAdmin(ctx context.Context, recordID uint64) error {
	result, err := r.db.ExecContext(ctx, deleteAdminBagItemQuery, recordID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return bag.ErrBagItemNotFound
	}
	return nil
}

func (r *BagRepository) playerExists(ctx context.Context, playerID uint64) (bool, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, adminBagPlayerExistsQuery, playerID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func scanAdminBagSummary(rows *sql.Rows) (bag.AdminItemSummary, error) {
	var (
		item     bag.AdminItemSummary
		recordID int64
		playerID int64
		itemID   int64
		count    int64
	)
	if err := rows.Scan(&recordID, &playerID, &item.PlayerName, &itemID, &count, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return bag.AdminItemSummary{}, err
	}
	item.RecordID = uint64(recordID)
	item.PlayerID = uint64(playerID)
	item.ItemID = uint32(itemID)
	item.Count = uint32(count)
	return item, nil
}

func scanAdminBagDetailRow(row *sql.Row) (*bag.AdminItemDetail, error) {
	var (
		item     bag.AdminItemDetail
		recordID int64
		playerID int64
		itemID   int64
		count    int64
	)
	if err := row.Scan(&recordID, &playerID, &item.PlayerName, &itemID, &count, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.RecordID = uint64(recordID)
	item.PlayerID = uint64(playerID)
	item.ItemID = uint32(itemID)
	item.Count = uint32(count)
	return &item, nil
}

func isPlayerItemUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
