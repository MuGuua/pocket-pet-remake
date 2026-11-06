package pet_test

import (
	"context"
	"errors"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/teststub"
	"testing"
)

// 仓储带有旧快照能力；任何详情对其重建/读取都会失败，确保展示从源数据读取。
type detailSnapshotRepository struct {
	*teststub.PetRepository
	snapshotCalls int
}

func (r *detailSnapshotRepository) RefreshPlayerPetCombatSnapshots(context.Context, uint64) error {
	r.snapshotCalls++
	return errors.New("unexpected snapshot rebuild")
}
func (r *detailSnapshotRepository) ListPlayerPetCombatSnapshotsByPlayerID(context.Context, uint64) ([]pet.Pet, error) {
	r.snapshotCalls++
	return nil, errors.New("unexpected snapshot read")
}
func (r *detailSnapshotRepository) ListPlayerLineupCombatSnapshotsByPlayerID(context.Context, uint64) ([]pet.LineupPet, error) {
	r.snapshotCalls++
	return nil, errors.New("unexpected lineup snapshot read")
}
func (r *detailSnapshotRepository) FindPlayerPetCombatSnapshotByUID(context.Context, uint64, uint64) (*pet.Pet, error) {
	r.snapshotCalls++
	return nil, errors.New("unexpected snapshot read")
}

// 重复查看跟随数据库的新值，保留编队与归属检查，不通过旧持久化视图恢复旧属性。
func TestPetDetailReadsFreshSourceWithoutSnapshots(t *testing.T) {
	ctx := context.Background()
	repo := &detailSnapshotRepository{PetRepository: teststub.NewPetRepository()}
	lineup, err := repo.ListLineupByPlayerID(ctx, teststub.DemoPlayerID)
	if err != nil || len(lineup) == 0 {
		t.Fatal("missing lineup fixture")
	}
	uid := lineup[0].PetUID
	svc := pet.NewService(repo, nil, nil, nil)
	first, err := svc.GetPetDetail(ctx, teststub.DemoPlayerID, uid)
	if err != nil || !first.InLineup || first.PetUID != uid {
		t.Fatalf("detail unavailable: %+v %v", first, err)
	}
	if _, err = repo.UpdatePetHPByUID(ctx, teststub.DemoPlayerID, uid, 1); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.GetPetDetail(ctx, teststub.DemoPlayerID, uid)
	if err != nil || fresh.HP != 1 {
		t.Fatalf("source update was lost: %+v %v", fresh, err)
	}
	if _, err = svc.GetPetDetail(ctx, teststub.DemoPlayerID+1, uid); !errors.Is(err, pet.ErrPetNotFound) {
		t.Fatalf("ownership not enforced: %v", err)
	}
	if repo.snapshotCalls != 0 {
		t.Fatal("detail entered snapshot rebuild/read chain")
	}
}
