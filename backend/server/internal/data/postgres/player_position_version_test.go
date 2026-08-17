package postgres

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

// positionVersionResultStub 模拟 database/sql 的更新结果，允许测试实际写入、旧版本跳过和驱动错误。
type positionVersionResultStub struct {
	rowsAffected    int64
	rowsAffectedErr error
}

// LastInsertId 不参与位置更新，仅用于满足 sql.Result 接口。
func (r positionVersionResultStub) LastInsertId() (int64, error) {
	return 0, nil
}

// RowsAffected 返回测试预设的条件更新命中行数或驱动错误。
func (r positionVersionResultStub) RowsAffected() (int64, error) {
	return r.rowsAffected, r.rowsAffectedErr
}

// positionVersionDBStub 记录条件更新 SQL 与参数，不建立真实 PostgreSQL 连接。
type positionVersionDBStub struct {
	query   string
	args    []any
	result  sql.Result
	execErr error
}

// ExecContext 记录仓储发出的 SQL 和参数，并返回测试预设结果。
func (db *positionVersionDBStub) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	db.query = query
	db.args = append([]any(nil), args...)
	return db.result, db.execErr
}

// QueryContext 不参与本组测试，仅用于满足 DBTX 接口。
func (db *positionVersionDBStub) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected QueryContext call")
}

// QueryRowContext 不参与本组测试，仅用于满足 DBTX 接口。
func (db *positionVersionDBStub) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

// TestPlayerRepositoryUpdatePositionIfNewerApplied 验证更高版本会携带完整位置参数并报告实际写入。
func TestPlayerRepositoryUpdatePositionIfNewerApplied(t *testing.T) {
	db := &positionVersionDBStub{result: positionVersionResultStub{rowsAffected: 1}}
	repository := NewPlayerRepository(db)

	applied, err := repository.UpdatePositionIfNewer(context.Background(), 10001, 9, 12, 18, 41)
	if err != nil {
		t.Fatalf("UpdatePositionIfNewer() error = %v", err)
	}
	if !applied {
		t.Fatal("UpdatePositionIfNewer() applied = false, want true")
	}
	if !strings.Contains(db.query, "position_version < $5") || !strings.Contains(db.query, "position_version = $5") {
		t.Fatalf("query = %q, want version assignment and newer-version predicate", db.query)
	}
	wantArgs := []any{uint64(10001), uint32(9), int32(12), int32(18), int64(41)}
	if !reflect.DeepEqual(db.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", db.args, wantArgs)
	}
}

// TestPlayerProfileQueriesReadPositionVersion 验证普通档案和战斗快照读取都包含永久位置版本列。
func TestPlayerProfileQueriesReadPositionVersion(t *testing.T) {
	if !strings.Contains(findPlayerByIDQuery, "position_version") {
		t.Fatalf("findPlayerByIDQuery = %q, want position_version", findPlayerByIDQuery)
	}
	if !strings.Contains(findPlayerCombatSnapshotQuery, "p.position_version") {
		t.Fatalf("findPlayerCombatSnapshotQuery = %q, want p.position_version", findPlayerCombatSnapshotQuery)
	}
}

// TestPlayerRepositoryUpdatePositionIfNewerSkipsStaleVersion 验证数据库未命中时按旧版本安全跳过处理。
func TestPlayerRepositoryUpdatePositionIfNewerSkipsStaleVersion(t *testing.T) {
	db := &positionVersionDBStub{result: positionVersionResultStub{rowsAffected: 0}}
	repository := NewPlayerRepository(db)

	applied, err := repository.UpdatePositionIfNewer(context.Background(), 10001, 9, 12, 18, 40)
	if err != nil {
		t.Fatalf("UpdatePositionIfNewer() error = %v", err)
	}
	if applied {
		t.Fatal("UpdatePositionIfNewer() applied = true, want false")
	}
}

// TestPlayerRepositoryUpdatePositionIfNewerReturnsDatabaseErrors 验证 SQL 执行错误和 RowsAffected 错误都会向上返回。
func TestPlayerRepositoryUpdatePositionIfNewerReturnsDatabaseErrors(t *testing.T) {
	execErr := errors.New("postgres unavailable")
	db := &positionVersionDBStub{execErr: execErr}
	repository := NewPlayerRepository(db)
	if _, err := repository.UpdatePositionIfNewer(context.Background(), 10001, 9, 12, 18, 41); !errors.Is(err, execErr) {
		t.Fatalf("UpdatePositionIfNewer(exec error) = %v, want %v", err, execErr)
	}

	rowsErr := errors.New("rows affected unavailable")
	db = &positionVersionDBStub{result: positionVersionResultStub{rowsAffectedErr: rowsErr}}
	repository = NewPlayerRepository(db)
	if _, err := repository.UpdatePositionIfNewer(context.Background(), 10001, 9, 12, 18, 41); !errors.Is(err, rowsErr) {
		t.Fatalf("UpdatePositionIfNewer(rows error) = %v, want %v", err, rowsErr)
	}
}

// TestPlayerRepositoryUpdatePositionIfNewerRejectsVersionOverflow 验证 uint64 超过 PostgreSQL BIGINT 上限时不会发出无效 SQL。
func TestPlayerRepositoryUpdatePositionIfNewerRejectsVersionOverflow(t *testing.T) {
	db := &positionVersionDBStub{result: positionVersionResultStub{rowsAffected: 1}}
	repository := NewPlayerRepository(db)

	if _, err := repository.UpdatePositionIfNewer(context.Background(), 10001, 9, 12, 18, uint64(math.MaxInt64)+1); err == nil {
		t.Fatal("UpdatePositionIfNewer() error = nil, want BIGINT overflow error")
	}
	if db.query != "" || len(db.args) != 0 {
		t.Fatalf("overflow issued SQL: query=%q args=%#v", db.query, db.args)
	}
}
