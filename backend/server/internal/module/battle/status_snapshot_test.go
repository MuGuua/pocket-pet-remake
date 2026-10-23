package battle

import "testing"

// 状态快照必须携带剩余回合，顺序与 statusIDs 一致，且已到期（剩余 0 回合）的状态不下发。
func TestStatusSnapshotsCarryRemainingRounds(t *testing.T) {
	actor := &actorRuntime{hp: 100, hpMax: 100, statuses: map[uint32]*statusRuntime{}}
	actor.applyStatus(StatusSleep, 2, 0)
	actor.applyStatus(StatusArmorBreak, 3, 10)
	actor.statuses[StatusBleed] = &statusRuntime{statusID: StatusBleed, remainingRound: 0}

	snapshots := actor.statusSnapshots()
	ids := actor.statusIDs()
	if len(snapshots) != 2 || len(ids) != 2 {
		t.Fatalf("expected 2 active statuses, got snapshots=%v ids=%v", snapshots, ids)
	}
	for index, snapshot := range snapshots {
		if snapshot.StatusID != ids[index] {
			t.Fatalf("snapshot order mismatch at %d: %v vs %v", index, snapshot.StatusID, ids[index])
		}
	}
	if snapshots[0].StatusID != StatusArmorBreak || snapshots[0].RemainingRounds != 3 {
		t.Fatalf("unexpected first snapshot %+v", snapshots[0])
	}
	if snapshots[1].StatusID != StatusSleep || snapshots[1].RemainingRounds != 2 {
		t.Fatalf("unexpected second snapshot %+v", snapshots[1])
	}
}

// 没有状态时返回空切片而不是 nil，客户端收到 [] 便于统一处理。
func TestStatusSnapshotsEmptyIsNotNil(t *testing.T) {
	actor := &actorRuntime{statuses: map[uint32]*statusRuntime{}}
	if snapshots := actor.statusSnapshots(); snapshots == nil || len(snapshots) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", snapshots)
	}
}
