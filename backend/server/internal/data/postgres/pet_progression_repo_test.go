package postgres

import (
	"strings"
	"testing"
)

// TestPetProgressionHPClampParametersUseIntegerCasts 防止 PostgreSQL 在 LEAST 表达式中把 HP 参数推导为 TEXT。
// player_pet.hp 与 player_pet.hp_max 都是 INTEGER；若同一占位参数还用于字段赋值，缺少显式转换会在语句解析阶段触发 SQLSTATE 42P08。
func TestPetProgressionHPClampParametersUseIntegerCasts(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "battle experience progression",
			query: savePetExpProgressionQuery,
			want:  "LEAST($11::INTEGER, $6::INTEGER)",
		},
		{
			name:  "manual attribute allocation",
			query: savePetAttrAllocationQuery,
			want:  "LEAST($14::INTEGER, $9::INTEGER)",
		},
		{
			name:  "recalculated combat stats",
			query: saveRecalculatedPetCombatStatsQuery,
			want:  "LEAST($8::INTEGER, $3::INTEGER)",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if !strings.Contains(testCase.query, testCase.want) {
				t.Fatalf("progression query must keep explicit INTEGER casts: want %q", testCase.want)
			}
		})
	}
}
