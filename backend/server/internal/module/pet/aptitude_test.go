package pet

import (
	"math/rand"
	"testing"
)

func TestRollWildCaptureAptitudesWithinRange(t *testing.T) {
	ranges := AptitudeRollRanges{
		HPAptMin: 8, HPAptMax: 14,
		ATKAptMin: 8, ATKAptMax: 13,
		DEFAptMin: 8, DEFAptMax: 12,
		SPDAptMin: 7, SPDAptMax: 12,
		MANAAptMin: 6, MANAAptMax: 11,
	}
	rng := rand.New(rand.NewSource(42))
	for index := 0; index < 100; index++ {
		result := RollWildCaptureAptitudes(ranges, rng)
		assertInRange(t, result.HPApt, ranges.HPAptMin, ranges.HPAptMax)
		assertInRange(t, result.ATKApt, ranges.ATKAptMin, ranges.ATKAptMax)
		assertInRange(t, result.DEFApt, ranges.DEFAptMin, ranges.DEFAptMax)
		assertInRange(t, result.SPDApt, ranges.SPDAptMin, ranges.SPDAptMax)
		assertInRange(t, result.MANAApt, ranges.MANAAptMin, ranges.MANAAptMax)
	}
}

func TestValidateAptitudeRollRanges(t *testing.T) {
	valid := AptitudeRollRanges{
		HPAptMin: 1, HPAptMax: 10,
		ATKAptMin: 1, ATKAptMax: 10,
		DEFAptMin: 1, DEFAptMax: 10,
		SPDAptMin: 1, SPDAptMax: 10,
		MANAAptMin: 1, MANAAptMax: 10,
	}
	if err := ValidateAptitudeRollRanges(valid); err != nil {
		t.Fatalf("expected valid ranges, got %v", err)
	}
	invalid := valid
	invalid.HPAptMax = 0
	if err := ValidateAptitudeRollRanges(invalid); err == nil {
		t.Fatal("expected invalid range error")
	}
}

func assertInRange(t *testing.T, value uint32, minValue uint32, maxValue uint32) {
	t.Helper()
	if value < minValue || value > maxValue {
		t.Fatalf("value %d out of range [%d,%d]", value, minValue, maxValue)
	}
}
