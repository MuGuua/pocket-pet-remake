package wstransport

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/runtimeview"
	"pocket-pet-remake/server/internal/teststub"
)

// 使用既有源数据桩提供快照能力，并记录真实世界构建流程中的重复刷新。
type worldPreparedPlayerRepo struct {
	*teststub.PlayerRepository
	refreshes int
	reads     int
}

func (r *worldPreparedPlayerRepo) RefreshPlayerCombatSnapshot(context.Context, uint64) error {
	r.refreshes++
	return nil
}
func (r *worldPreparedPlayerRepo) FindPlayerCombatSnapshot(ctx context.Context, id uint64) (*player.Profile, error) {
	r.reads++
	return r.PlayerRepository.FindByPlayerID(ctx, id)
}

type worldPreparedPetRepo struct {
	*teststub.PetRepository
	refreshes int
	reads     int
}

func (r *worldPreparedPetRepo) RefreshPlayerPetCombatSnapshots(context.Context, uint64) error {
	r.refreshes++
	return nil
}
func (r *worldPreparedPetRepo) ListPlayerPetCombatSnapshotsByPlayerID(context.Context, uint64) ([]pet.Pet, error) {
	return []pet.Pet{}, nil
}
func (r *worldPreparedPetRepo) FindPlayerPetCombatSnapshotByUID(context.Context, uint64, uint64) (*pet.Pet, error) {
	return nil, nil
}
func (r *worldPreparedPetRepo) ListPlayerLineupCombatSnapshotsByPlayerID(ctx context.Context, id uint64) ([]pet.LineupPet, error) {
	r.reads++
	return r.PetRepository.ListLineupByPlayerID(ctx, id)
}

// 世界快照的统一刷新结果必须用于后续人物与宠物编队读取，每个视图在该次流程仅刷新一次。
func TestWorldSnapshotUsesPreparedPlayerAndPetViews(t *testing.T) {
	players := &worldPreparedPlayerRepo{PlayerRepository: teststub.NewPlayerRepository()}
	pets := &worldPreparedPetRepo{PetRepository: teststub.NewPetRepository()}
	playerService := player.NewService(players, nil, nil, nil)
	id, router, _, _ := buildWorldRouterWithPlayerServiceForTest(t, playerService)
	router.worldHandler.petService = pet.NewService(pets, nil, nil, nil)
	router.worldHandler.SetRuntimeSnapshotService(runtimeview.NewService(players, pets, nil, nil))
	snapshot, err := router.worldHandler.BuildWorldSnapshotForPlayer(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot == nil || snapshot.Self.PlayerID != id || players.refreshes != 1 || players.reads != 1 || pets.refreshes != 1 || pets.reads != 1 {
		t.Fatalf("duplicate world snapshot reads: player=%d/%d pet=%d/%d", players.refreshes, players.reads, pets.refreshes, pets.reads)
	}
	if _, err := router.worldHandler.BuildWorldSnapshotForPlayer(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if players.refreshes != 2 || pets.refreshes != 2 {
		t.Fatal("prepared marker escaped current request")
	}
}
