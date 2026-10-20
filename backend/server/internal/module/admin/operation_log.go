package admin

import (
	"context"
	"errors"
	"strings"
)

// ErrOperationLogInvalid 表示审计日志缺少必要字段（操作人、模块、动作或目标）。
var ErrOperationLogInvalid = errors.New("admin operation log invalid")

// OperationLogEntry 描述一条后台高风险操作的审计记录，落库到 admin_operation_log。
// Reason 为运营填写的操作原因；AfterSnapshot / ExtraPayload 为任意可 JSON 序列化的结构，
// 用于事后排查「谁、在什么时候、为什么、改了什么、结果如何」。
type OperationLogEntry struct {
	AdminUserID   uint64
	ModuleKey     string
	ActionKey     string
	TargetType    string
	TargetID      string
	Reason        string
	AfterSnapshot any
	ExtraPayload  any
}

// OperationLogWriter 是审计日志的持久化边界，由 postgres.AdminRepository 实现。
type OperationLogWriter interface {
	InsertOperationLog(ctx context.Context, entry OperationLogEntry) error
}

// SetOperationLogWriter 注入审计日志写入器。未注入时 RecordOperation 静默跳过，
// 保证单元测试与未装配数据库的场景不受影响。
func (s *Service) SetOperationLogWriter(writer OperationLogWriter) {
	if s == nil {
		return
	}
	s.operationLogs = writer
}

// RecordOperation 写入一条后台操作审计日志。字段不完整时返回 ErrOperationLogInvalid。
func (s *Service) RecordOperation(ctx context.Context, entry OperationLogEntry) error {
	if s == nil || s.operationLogs == nil {
		return nil
	}
	entry.ModuleKey = strings.TrimSpace(entry.ModuleKey)
	entry.ActionKey = strings.TrimSpace(entry.ActionKey)
	entry.TargetType = strings.TrimSpace(entry.TargetType)
	entry.TargetID = strings.TrimSpace(entry.TargetID)
	entry.Reason = strings.TrimSpace(entry.Reason)
	if entry.AdminUserID == 0 || entry.ModuleKey == "" || entry.ActionKey == "" || entry.TargetType == "" || entry.TargetID == "" {
		return ErrOperationLogInvalid
	}
	return s.operationLogs.InsertOperationLog(ctx, entry)
}
