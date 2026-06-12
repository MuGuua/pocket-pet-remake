package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/wallet"
)

// WalletRepository 把后台钱包查询与调整映射到 player_wallet 与 currency_change_log。
// 钱包始终以总铜币为真值，后台展示所需的金银铜拆分由仓储统一计算返回。
type WalletRepository struct {
	db DBTX
}

// NewWalletRepository 构造后台钱包仓储。
func NewWalletRepository(db DBTX) *WalletRepository {
	return &WalletRepository{db: db}
}

const adminWalletListBaseQuery = `
SELECT
  w.player_id,
  p.name,
  w.currency_copper_total,
  w.created_at,
  w.updated_at
FROM player_wallet w
JOIN player p ON p.id = w.player_id
`

const adminWalletDetailQuery = `
SELECT
  w.player_id,
  p.name,
  w.currency_copper_total,
  w.version,
  w.created_at,
  w.updated_at
FROM player_wallet w
JOIN player p ON p.id = w.player_id
WHERE w.player_id = $1
LIMIT 1
`

const runtimeWalletDetailQuery = `
SELECT
  currency_copper_total,
  version,
  created_at,
  updated_at
FROM player_wallet
WHERE player_id = $1
LIMIT 1
`

const adjustWalletTotalQuery = `
UPDATE player_wallet
SET currency_copper_total = currency_copper_total + $2,
    version = version + 1
WHERE player_id = $1
  AND currency_copper_total + $2 >= 0
RETURNING currency_copper_total, version, created_at, updated_at
`

const insertCurrencyChangeLogQuery = `
INSERT INTO currency_change_log (
  player_id,
  currency_type,
  before_total_copper,
  change_total_copper,
  after_total_copper,
  reason_type,
  reason_ref_id,
  operator_type,
  operator_id
) VALUES ($1, 'base_coin', $2, $3, $4, $5, $6, $7, $8)
`

const runtimeWalletQuery = `
SELECT currency_copper_total
FROM player_wallet
WHERE player_id = $1
LIMIT 1
`

func (r *WalletRepository) ListForAdmin(ctx context.Context, query wallet.AdminListQuery) (*wallet.AdminWalletList, error) {
	query = query.Normalize()
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.PlayerID > 0 {
		conditions = append(conditions, "w.player_id = "+nextArg(query.PlayerID))
	}
	if query.Keyword != "" {
		placeholder := nextArg("%" + query.Keyword + "%")
		conditions = append(conditions, "p.name ILIKE "+placeholder)
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	countQuery := "SELECT COUNT(1) FROM player_wallet w JOIN player p ON p.id = w.player_id " + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	listQuery := adminWalletListBaseQuery + whereClause + fmt.Sprintf("\nORDER BY w.player_id DESC\nLIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]wallet.AdminWalletSummary, 0)
	for rows.Next() {
		var (
			value       wallet.AdminWalletSummary
			totalCopper uint64
		)
		if err := rows.Scan(&value.PlayerID, &value.PlayerName, &totalCopper, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, err
		}
		value.Wallet = buildWalletSnapshot(totalCopper)
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &wallet.AdminWalletList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *WalletRepository) FindAdminDetailByPlayerID(ctx context.Context, playerID uint64) (*wallet.AdminWalletDetail, error) {
	var (
		value       wallet.AdminWalletDetail
		totalCopper uint64
	)
	if err := r.db.QueryRowContext(ctx, adminWalletDetailQuery, playerID).Scan(
		&value.PlayerID,
		&value.PlayerName,
		&totalCopper,
		&value.Version,
		&value.CreatedAt,
		&value.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	value.Wallet = buildWalletSnapshot(totalCopper)
	return &value, nil
}

func (r *WalletRepository) AdjustForAdmin(ctx context.Context, playerID uint64, input wallet.AdminAdjustInput) (*wallet.AdminWalletDetail, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	detail, err := r.FindAdminDetailByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	var (
		afterTotal uint64
		version    uint64
		createdAt  = detail.CreatedAt
		updatedAt  = detail.UpdatedAt
	)
	if err := tx.QueryRowContext(ctx, adjustWalletTotalQuery, playerID, input.ChangeTotalCopper).Scan(&afterTotal, &version, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, wallet.ErrInvalidAdminWalletInput
		}
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, insertCurrencyChangeLogQuery, playerID, detail.Wallet.TotalCopper, input.ChangeTotalCopper, afterTotal, "admin_adjust", 0, "admin", 0); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &wallet.AdminWalletDetail{
		PlayerID:   playerID,
		PlayerName: detail.PlayerName,
		Wallet:     buildWalletSnapshot(afterTotal),
		Version:    version,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}

func (r *WalletRepository) AdjustRuntime(ctx context.Context, playerID uint64, input wallet.RuntimeAdjustInput) (*wallet.RuntimeAdjustResult, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	beforeTotal, _, _, _, err := r.loadRuntimeWalletRow(ctx, tx, playerID)
	if err != nil {
		return nil, err
	}
	if beforeTotal == nil {
		return nil, nil
	}

	var (
		afterTotal uint64
		version    uint64
		createdAt  sql.NullTime
		updatedAt  sql.NullTime
	)
	if err := tx.QueryRowContext(ctx, adjustWalletTotalQuery, playerID, input.ChangeTotalCopper).Scan(&afterTotal, &version, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, wallet.ErrInvalidRuntimeAdjustInput
		}
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, insertCurrencyChangeLogQuery, playerID, *beforeTotal, input.ChangeTotalCopper, afterTotal, input.ReasonType, input.ReasonRefID, input.OperatorType, input.OperatorID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &wallet.RuntimeAdjustResult{
		Wallet:      buildWalletSnapshot(afterTotal),
		Version:     version,
		ReasonType:  input.ReasonType,
		ReasonRefID: input.ReasonRefID,
	}, nil
}

func buildWalletSnapshot(total uint64) wallet.Snapshot {
	return wallet.Snapshot{
		TotalCopper: total,
		Gold:        total / 1000000,
		Silver:      (total % 1000000) / 1000,
		Copper:      total % 1000,
	}
}

func (r *WalletRepository) GetRuntimeSnapshot(ctx context.Context, playerID uint64) (*wallet.Snapshot, error) {
	totalCopper, _, _, _, err := r.loadRuntimeWalletRow(ctx, r.db, playerID)
	if err != nil {
		return nil, err
	}
	if totalCopper == nil {
		return nil, nil
	}
	snapshot := buildWalletSnapshot(*totalCopper)
	return &snapshot, nil
}

type walletRowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *WalletRepository) loadRuntimeWalletRow(ctx context.Context, querier walletRowQuerier, playerID uint64) (*uint64, *uint64, *sql.NullTime, *sql.NullTime, error) {
	var (
		totalCopper uint64
		version     uint64
		createdAt   sql.NullTime
		updatedAt   sql.NullTime
	)
	if err := querier.QueryRowContext(ctx, runtimeWalletDetailQuery, playerID).Scan(&totalCopper, &version, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, nil, nil
		}
		return nil, nil, nil, nil, err
	}
	return &totalCopper, &version, &createdAt, &updatedAt, nil
}
