package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"pocket-pet-remake/server/internal/platform/telemetry"
)

// 仅记录 SQL 操作类别，不保存语句、参数或玩家标识；驱动在读取/关闭结果后报告结束。
type queryMetricsTracer struct{}
type queryMetricsKey struct{}
type queryMetricsStart struct {
	at        time.Time
	operation string
}

// 保存单调开始时间和固定操作类别，不传播数据库载荷到指标。
func (queryMetricsTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	text := strings.TrimSpace(data.SQL)
	if index := strings.IndexAny(text, " \r\n\t("); index >= 0 {
		text = text[:index]
	}
	operation := strings.ToUpper(text)
	switch operation {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "BEGIN", "COMMIT", "ROLLBACK":
	default:
		operation = "OTHER"
	}
	return context.WithValue(ctx, queryMetricsKey{}, queryMetricsStart{at: time.Now(), operation: operation})
}

// 驱动报告的错误计入失败；耗时包含结果读取，但不包含 database/sql 连接池排队。
func (queryMetricsTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if start, ok := ctx.Value(queryMetricsKey{}).(queryMetricsStart); ok {
		telemetry.Duration("sql."+start.operation, time.Since(start.at), data.Err != nil)
	}
}
