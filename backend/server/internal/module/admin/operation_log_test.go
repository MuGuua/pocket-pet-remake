package admin

import (
	"context"
	"errors"
	"testing"
)

type captureLogWriter struct{ entries []OperationLogEntry }

func (w *captureLogWriter) InsertOperationLog(_ context.Context, entry OperationLogEntry) error {
	w.entries = append(w.entries, entry)
	return nil
}

// TestRecordOperationWritesTrimmedEntry 验证审计日志会去除首尾空白后写入。
func TestRecordOperationWritesTrimmedEntry(t *testing.T) {
	writer := &captureLogWriter{}
	svc := NewService(nil, nil)
	svc.SetOperationLogWriter(writer)
	err := svc.RecordOperation(context.Background(), OperationLogEntry{
		AdminUserID: 1, ModuleKey: " reward ", ActionKey: "grant", TargetType: "player", TargetID: "10001", Reason: "  补偿 ",
	})
	if err != nil {
		t.Fatalf("RecordOperation() err = %v", err)
	}
	if len(writer.entries) != 1 || writer.entries[0].ModuleKey != "reward" || writer.entries[0].Reason != "补偿" {
		t.Fatalf("entries = %#v, want one trimmed entry", writer.entries)
	}
}

// TestRecordOperationRejectsIncompleteEntry 验证缺少操作人或目标时拒绝写入。
func TestRecordOperationRejectsIncompleteEntry(t *testing.T) {
	svc := NewService(nil, nil)
	svc.SetOperationLogWriter(&captureLogWriter{})
	err := svc.RecordOperation(context.Background(), OperationLogEntry{ModuleKey: "reward", ActionKey: "grant", TargetType: "player", TargetID: "1"})
	if !errors.Is(err, ErrOperationLogInvalid) {
		t.Fatalf("err = %v, want ErrOperationLogInvalid", err)
	}
}

// TestRecordOperationWithoutWriterIsNoop 验证未注入写入器时静默跳过。
func TestRecordOperationWithoutWriterIsNoop(t *testing.T) {
	if err := NewService(nil, nil).RecordOperation(context.Background(), OperationLogEntry{}); err != nil {
		t.Fatalf("err = %v, want nil when writer is absent", err)
	}
}
