package equipment

import "testing"

func TestCalculateEnhanceGoldCostFixedIncrement(t *testing.T) {
	config := EnhanceGoldCostConfig{
		IsEnabled:      true,
		BaseCopper:     500,
		IncrementMode:  EnhanceGoldIncrementModeFixed,
		IncrementFixed: 500,
	}
	if got := CalculateEnhanceGoldCost(1, config); got != 500 {
		t.Fatalf("level 1 = %d, want 500", got)
	}
	if got := CalculateEnhanceGoldCost(2, config); got != 1000 {
		t.Fatalf("level 2 = %d, want 1000", got)
	}
	if got := CalculateEnhanceGoldCost(3, config); got != 1500 {
		t.Fatalf("level 3 = %d, want 1500", got)
	}
}

func TestCalculateEnhanceGoldCostPercentIncrement(t *testing.T) {
	config := EnhanceGoldCostConfig{
		IsEnabled:        true,
		BaseCopper:       500,
		IncrementMode:    EnhanceGoldIncrementModePercent,
		IncrementPercent: 10,
	}
	if got := CalculateEnhanceGoldCost(1, config); got != 500 {
		t.Fatalf("level 1 = %d, want 500", got)
	}
	if got := CalculateEnhanceGoldCost(2, config); got != 550 {
		t.Fatalf("level 2 = %d, want 550", got)
	}
	if got := CalculateEnhanceGoldCost(3, config); got != 605 {
		t.Fatalf("level 3 = %d, want 605", got)
	}
}

func TestCalculateEnhanceGoldCostDisabled(t *testing.T) {
	config := EnhanceGoldCostConfig{IsEnabled: false, BaseCopper: 500, IncrementMode: EnhanceGoldIncrementModeFixed, IncrementFixed: 500}
	if got := CalculateEnhanceGoldCost(5, config); got != 0 {
		t.Fatalf("disabled config = %d, want 0", got)
	}
}
