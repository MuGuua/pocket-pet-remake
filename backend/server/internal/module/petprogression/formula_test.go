package petprogression

import "testing"

// 表11 示例：基础攻资 12848、红色 67500、分配 583、增幅 1.48 → 最终攻击约 495182。
func TestFinalCombatStatsAttackMatchesReferenceTable(t *testing.T) {
	stats := FinalCombatStats(ProgressionInput{
		Level:           100,
		AptitudeProfile: AptitudeProfileNormal,
		Aptitudes: GrowthAptitudes{
			BaseATKApt:  12848,
			ExtraATKApt: 67500,
		},
		ManualPoints: ManualAllocatedPoints{ATK: 583 - 99},
		BoostPct:     1.48,
	})
	const want uint32 = 495182
	if stats.ATK != want {
		t.Fatalf("ATK = %d, want %d", stats.ATK, want)
	}
}

// 表9 反推：1 点攻击属性、无增幅时，有效攻资 277.77 对应 +1 攻击。
func TestRequiredEffectiveAptitudeForOneAttackPoint(t *testing.T) {
	const convertRate = 277.77
	got := RequiredEffectiveAptitudeForStatGain(convertRate, 1, 1, 0)
	const want = 277.77
	if got < want-0.01 || got > want+0.01 {
		t.Fatalf("required effective apt = %v, want %v", got, want)
	}
	stats := FinalCombatStats(ProgressionInput{
		Level:           1,
		AptitudeProfile: AptitudeProfileNormal,
		Aptitudes: GrowthAptitudes{
			BaseATKApt: 253,
		},
		ManualPoints: ManualAllocatedPoints{ATK: 1},
	})
	if stats.ATK != 1 {
		t.Fatalf("ATK = %d, want 1", stats.ATK)
	}
}

func TestAutoAllocatedPointsDefenseWithoutEvolutionBonus(t *testing.T) {
	autoPoints := AutoAllocatedPointsForLevel(100, 100, 0)
	if autoPoints.ATK != 174 {
		t.Fatalf("auto ATK = %d, want 174", autoPoints.ATK)
	}
	if autoPoints.DEF != 99 {
		t.Fatalf("auto DEF = %d, want 99", autoPoints.DEF)
	}
}

func TestEffectiveAptitudeUsesBaseAndExtraWeights(t *testing.T) {
	got := EffectiveAptitude(12000, 285372, 1)
	const want = 355646.4
	if got < want-0.1 || got > want+0.1 {
		t.Fatalf("effective apt = %v, want %v", got, want)
	}
}
