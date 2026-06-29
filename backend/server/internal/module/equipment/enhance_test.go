package equipment

import "testing"

// TestRollEnhanceSuccessAlwaysSucceedsAt100 验证 100% 成功率不会再掷骰失败。
func TestRollEnhanceSuccessAlwaysSucceedsAt100(t *testing.T) {
	for index := 0; index < 32; index++ {
		rollPct, success, err := RollEnhanceSuccess(100)
		if err != nil {
			t.Fatalf("RollEnhanceSuccess() error = %v", err)
		}
		if !success {
			t.Fatalf("RollEnhanceSuccess(100) success = false, roll = %d", rollPct)
		}
		if rollPct != 100 {
			t.Fatalf("RollEnhanceSuccess(100) roll = %d, want 100", rollPct)
		}
	}
}
