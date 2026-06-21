package battle

import "testing"

func TestCalculateControlChanceByPower(t *testing.T) {
	cases := []struct {
		name   string
		power  uint32
		resist uint32
		want   uint32
	}{
		{name: "equal immune", power: 350, resist: 350, want: 0},
		{name: "resist higher immune", power: 300, resist: 350, want: 0},
		{name: "gap50 guaranteed", power: 350, resist: 300, want: 100},
		{name: "gap49", power: 350, resist: 301, want: 98},
		{name: "gap1", power: 350, resist: 349, want: 2},
		{name: "zero power", power: 0, resist: 0, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CalculateControlChanceByPower(tc.power, tc.resist); got != tc.want {
				t.Fatalf("CalculateControlChanceByPower(%d, %d) = %d, want %d", tc.power, tc.resist, got, tc.want)
			}
		})
	}
}

func TestResolveControlApplyChance(t *testing.T) {
	if got := resolveControlApplyChance(50, 0, 999); got != 50 {
		t.Fatalf("probability mode = %d, want 50 ignoring resist", got)
	}
	if got := resolveControlApplyChance(0, 350, 300); got != 100 {
		t.Fatalf("power mode = %d, want 100", got)
	}
	if got := resolveControlApplyChance(50, 350, 350); got != 0 {
		t.Fatalf("power mode should win over chance = %d, want 0", got)
	}
}
