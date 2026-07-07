package pet

import "testing"

func TestBuildBattleSkillIDsMergesEnabledSlots(t *testing.T) {
	loadout := SkillLoadout{
		InnateSkillIDs:        []uint32{1001, 1002},
		NormalSkillIDs:        []uint32{2001, 2002, 0},
		ActiveTalismanSkillID: 3001,
		ActiveTalismanEnabled: true,
		TalismanHeroSkillID:   3002,
		TalismanHeroEnabled:   false,
		TalismanSlot1SkillID:  3003,
		TalismanSlot1Enabled:  true,
		ArtifactSkillIDs:      [MaxArtifactSkillSlots]uint32{4001, 0, 4002},
	}
	got := BuildBattleSkillIDs(loadout)
	want := []uint32{1001, 1002, 2001, 2002, 3001, 3003, 4001, 4002}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d got=%v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("index=%d got=%d want=%d full=%v", index, got[index], want[index], got)
		}
	}
}

func TestBuildBattleSkillIDsDedupes(t *testing.T) {
	loadout := SkillLoadout{
		InnateSkillIDs: []uint32{1001},
		NormalSkillIDs: []uint32{1001, 2001},
	}
	got := BuildBattleSkillIDs(loadout)
	if len(got) != 2 || got[0] != 1001 || got[1] != 2001 {
		t.Fatalf("unexpected dedupe result: %v", got)
	}
}

func TestApplyLegacySkillIDs(t *testing.T) {
	loadout := SkillLoadout{}
	ApplyLegacySkillIDs(&loadout, []uint32{11, 22, 33, 44})
	if len(loadout.NormalSkillIDs) != 3 {
		t.Fatalf("normal slots=%v", loadout.NormalSkillIDs)
	}
	if loadout.NormalSkillIDs[0] != 11 || loadout.NormalSkillIDs[2] != 33 {
		t.Fatalf("unexpected normal=%v", loadout.NormalSkillIDs)
	}
}

func TestBuildSkillSlotViewHidesArtifactWhenDisabled(t *testing.T) {
	loadout := SkillLoadout{
		ArtifactSkillIDs: [MaxArtifactSkillSlots]uint32{9001, 9002, 0},
	}
	view := BuildSkillSlotView(loadout, false)
	for _, entry := range view.Artifact {
		if entry.SkillID != 0 {
			t.Fatalf("artifact should be hidden: %+v", entry)
		}
	}
	viewWithArtifact := BuildSkillSlotView(loadout, true)
	if viewWithArtifact.Artifact[0].SkillID != 9001 || viewWithArtifact.Artifact[1].SkillID != 9002 {
		t.Fatalf("artifact not shown: %+v", viewWithArtifact.Artifact)
	}
}

func TestResolvePetBattleSkillsUsesLegacyFallback(t *testing.T) {
	item := Pet{
		SkillIDs: []uint32{501, 502},
	}
	ResolvePetBattleSkills(&item)
	if len(item.SkillIDs) != 2 || item.SkillIDs[0] != 501 {
		t.Fatalf("legacy fallback failed: %v", item.SkillIDs)
	}
}

func TestResolvePetBattleSkillsMergesStructuredAndLegacySkills(t *testing.T) {
	item := Pet{
		SkillIDs: []uint32{1001, 2999},
		SkillLoadout: SkillLoadout{
			InnateSkillIDs: []uint32{1001},
			NormalSkillIDs: []uint32{2001},
		},
	}
	ResolvePetBattleSkills(&item)
	want := []uint32{1001, 2001, 2999}
	if len(item.SkillIDs) != len(want) {
		t.Fatalf("len=%d want=%d got=%v", len(item.SkillIDs), len(want), item.SkillIDs)
	}
	for index := range want {
		if item.SkillIDs[index] != want[index] {
			t.Fatalf("index=%d got=%d want=%d full=%v", index, item.SkillIDs[index], want[index], item.SkillIDs)
		}
	}
}
