package wstransport

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/battle"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/teststub"
)

// TestMedicinePouchRestoresPetToFullHP 验证佩戴药囊时，战斗结束后出战宠物被回满血。
func TestMedicinePouchRestoresPetToFullHP(t *testing.T) {
	ctx := context.Background()
	playerService := teststub.NewTestPlayerService()
	petRepo := teststub.NewPetRepository()
	petService := pet.NewService(petRepo, nil, nil, nil)
	equipmentRepo := teststub.NewEquipmentRepository()
	equipmentService := equipment.NewService(equipmentRepo, nil, nil, nil, nil)

	handler := NewBattleHandler(nil, playerService, petService, nil, nil, nil, nil, nil, nil,
		battle.NewService(nil), teststub.NewBattleRepository(), equipmentService, nil)

	// 先把宠物血量压到很低，模拟战斗消耗。
	pets, err := petService.ListPets(ctx, teststub.DemoPlayerID)
	if err != nil {
		t.Fatalf("ListPets() error = %v", err)
	}
	target := pets[0]
	if _, err := petService.UpdatePetHP(ctx, teststub.DemoPlayerID, target.PetUID, 1); err != nil {
		t.Fatalf("set low hp error = %v", err)
	}

	// 未佩戴药囊时应保持残血。
	settlement := &battleSettlement{}
	if err := handler.applyMedicinePouchRestore(ctx, teststub.DemoPlayerID, settlement); err != nil {
		t.Fatalf("restore without pouch error = %v", err)
	}
	afterNoPouch, err := petService.GetPetDetail(ctx, teststub.DemoPlayerID, target.PetUID)
	if err != nil {
		t.Fatalf("GetPet() error = %v", err)
	}
	if afterNoPouch.HP != 1 {
		t.Fatalf("HP without pouch = %d, want unchanged 1", afterNoPouch.HP)
	}

	// 佩戴药囊后应回满。
	equipmentRepo.SetMedicinePouch(&equipment.RuntimeMedicinePouch{
		ItemID:       1,
		ItemName:     "回血药囊",
		RestorePetHP: true,
	})
	settlementWithPouch := &battleSettlement{
		Pets: []pet.Pet{afterNoPouch},
	}
	if err := handler.applyMedicinePouchRestore(ctx, teststub.DemoPlayerID, settlementWithPouch); err != nil {
		t.Fatalf("restore with pouch error = %v", err)
	}
	afterPouch, err := petService.GetPetDetail(ctx, teststub.DemoPlayerID, target.PetUID)
	if err != nil {
		t.Fatalf("GetPet() error = %v", err)
	}
	if afterPouch.HP != afterPouch.HPMax {
		t.Fatalf("HP with pouch = %d, want full %d", afterPouch.HP, afterPouch.HPMax)
	}
}
