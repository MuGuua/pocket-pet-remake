package runtimeview

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// 同一可观察刷新器用于核对执行顺序、失败短路与本次成功标记。
type resultRefresher struct {
	calls  []string
	failAt string
	err    error
}

func (r *resultRefresher) record(key string) error {
	r.calls = append(r.calls, key)
	if key == r.failAt {
		return r.err
	}
	return nil
}
func (r *resultRefresher) RefreshPlayerCombatSnapshot(context.Context, uint64) error {
	return r.record("player")
}
func (r *resultRefresher) RefreshPlayerPetCombatSnapshots(context.Context, uint64) error {
	return r.record("pets")
}
func (r *resultRefresher) RefreshPlayerEquipmentSnapshot(context.Context, uint64) error {
	return r.record("equipment")
}
func (r *resultRefresher) RefreshPlayerSkillProgressSnapshot(context.Context, uint64) error {
	return r.record("skills")
}

// 标记只用于当前流程，重复请求仍执行必要刷新；失败不能被标记为已准备。
func TestRuntimeRefreshResult(t *testing.T) {
	for _, failAt := range []string{"", "player", "pets", "equipment", "skills"} {
		t.Run(failAt, func(t *testing.T) {
			refresher := &resultRefresher{failAt: failAt, err: errors.New("refresh failed")}
			service := NewService(refresher, refresher, refresher, refresher)
			result, err := service.RefreshPlayerRuntimeSnapshotsWithResult(context.Background(), 1)
			expected := RefreshResult{}
			keys := []string{"player", "pets", "equipment", "skills"}
			flags := []*bool{&expected.Player, &expected.Pets, &expected.Equipment, &expected.Skills}
			end := len(keys)
			for index, key := range keys {
				if key == failAt {
					end = index + 1
					break
				}
				*flags[index] = true
			}
			if result != expected || !reflect.DeepEqual(refresher.calls, keys[:end]) || (failAt != "" && !errors.Is(err, refresher.err)) {
				t.Fatalf("result=%+v calls=%v err=%v", result, refresher.calls, err)
			}
			if failAt == "" {
				if err := service.RefreshPlayerRuntimeSnapshots(context.Background(), 1); err != nil || len(refresher.calls) != 8 {
					t.Fatal("refresh result became a cross-request cache")
				}
			}
		})
	}
}
