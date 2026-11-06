package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"pocket-pet-remake/server/internal/platform/telemetry"
)

// 指标只保留固定操作类别，不泄露 SQL 字面量或参数；结束错误可在受保护后台看到。
func TestQueryMetricsTracerDoesNotRecordSQLOrArguments(t *testing.T) {
	tracer := queryMetricsTracer{}
	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: " SELECT secret_value FROM private_table WHERE token=$1", Args: []any{"sensitive"}})
	start := ctx.Value(queryMetricsKey{}).(queryMetricsStart)
	if start.operation != "SELECT" {
		t.Fatal("wrong operation label")
	}
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errors.New("failed")})
	found := false
	for _, row := range telemetry.Default.Snapshots() {
		if strings.Contains(row.Key, "secret") || strings.Contains(row.Key, "sensitive") {
			t.Fatal("query data leaked into labels")
		}
		if row.Key == "sql.SELECT" {
			found = row.Failures > 0 && row.Unit == "us"
		}
	}
	if !found {
		t.Fatal("query error metric missing")
	}
}
