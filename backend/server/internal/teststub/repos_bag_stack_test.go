package teststub

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/bag"
)

func TestBagRepositoryCreateForAdminSplitsStackableItemsByMaxStack(t *testing.T) {
	repo := NewBagRepository()
	repo.itemMaxStacks[2001] = 99

	if _, err := repo.CreateForAdmin(context.Background(), bag.AdminCreateItemInput{
		PlayerID:      DemoPlayerID,
		ContainerType: bag.ContainerTypeBag,
		ItemID:        2001,
		Quantity:      101,
	}); err != nil {
		t.Fatalf("CreateForAdmin() error = %v, want nil", err)
	}

	snapshot, err := repo.ListRuntimeContainer(context.Background(), DemoPlayerID, bag.ContainerTypeBag)
	if err != nil {
		t.Fatalf("ListRuntimeContainer() error = %v, want nil", err)
	}

	var stackQuantities []uint64
	for _, item := range snapshot.Items {
		if item.ItemID != 2001 {
			continue
		}
		stackQuantities = append(stackQuantities, item.Quantity)
	}
	if len(stackQuantities) != 2 {
		t.Fatalf("stack count = %d, want 2", len(stackQuantities))
	}
	if stackQuantities[0] != 99 || stackQuantities[1] != 2 {
		t.Fatalf("stack quantities = %v, want [99 2]", stackQuantities)
	}
}

func TestBagRepositoryConsumeRuntimeItemStackRefillsFrontStackBeforeRemovingTailStack(t *testing.T) {
	repo := NewBagRepository()
	repo.itemMaxStacks[2001] = 99

	if _, err := repo.CreateForAdmin(context.Background(), bag.AdminCreateItemInput{
		PlayerID:      DemoPlayerID,
		ContainerType: bag.ContainerTypeBag,
		ItemID:        2001,
		Quantity:      101,
	}); err != nil {
		t.Fatalf("CreateForAdmin() error = %v, want nil", err)
	}

	beforeSnapshot, err := repo.ListRuntimeContainer(context.Background(), DemoPlayerID, bag.ContainerTypeBag)
	if err != nil {
		t.Fatalf("ListRuntimeContainer() before consume error = %v, want nil", err)
	}
	sourceSlotIndex := uint32(0)
	for _, item := range beforeSnapshot.Items {
		if item.ItemID == 2001 && item.Quantity == 99 {
			sourceSlotIndex = item.SlotIndex
			break
		}
	}
	if sourceSlotIndex == 0 {
		t.Fatalf("sourceSlotIndex = 0, want the front 99 stack")
	}

	snapshot, err := repo.ConsumeRuntimeItemStack(context.Background(), DemoPlayerID, bag.ContainerTypeBag, sourceSlotIndex, 2, "test_consume", 1)
	if err != nil {
		t.Fatalf("ConsumeRuntimeItemStack() error = %v, want nil", err)
	}

	var stackQuantities []uint64
	for _, item := range snapshot.Items {
		if item.ItemID != 2001 {
			continue
		}
		stackQuantities = append(stackQuantities, item.Quantity)
	}
	if len(stackQuantities) != 1 {
		t.Fatalf("stack count after consume = %d, want 1", len(stackQuantities))
	}
	if stackQuantities[0] != 99 {
		t.Fatalf("front stack quantity after consume = %d, want 99", stackQuantities[0])
	}
}
