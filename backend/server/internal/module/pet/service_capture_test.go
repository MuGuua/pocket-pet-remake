package pet_test

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/monster"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/teststub"
)

func TestGrantCapturedPetUsesMonsterMappingAndRollsAptitudes(t *testing.T) {
	monsterRepo := teststub.NewMonsterRepository()
	petRepo := teststub.NewPetRepository()
	service := pet.NewService(petRepo, nil, monsterRepo, nil)

	wildPetInput := pet.AdminUpsertPetDefinitionInput{
		PetID: 103, PetName: "野生幼犬", AcquireMethod: pet.AcquireMethodWildCapture, IsEnabled: true,
		Level: 1, Quality: 1, HP: 22, HPMax: 22, ATK: 12, DEF: 9, SPD: 8, MANA: 9,
		HPApt: 10, ATKApt: 10, DEFApt: 10, SPDApt: 10, MANAApt: 10,
		HPAptRollMin: 8, HPAptRollMax: 14,
		ATKAptRollMin: 8, ATKAptRollMax: 13,
		DEFAptRollMin: 8, DEFAptRollMax: 12,
		SPDAptRollMin: 7, SPDAptRollMax: 12,
		MANAAptRollMin: 6, MANAAptRollMax: 11,
		SkillIDs: []uint32{1001},
	}
	if _, err := petRepo.CreatePetDefinitionForAdmin(context.Background(), wildPetInput.Normalize()); err != nil {
		t.Fatalf("CreatePetDefinitionForAdmin() error = %v", err)
	}

	monsterInput := monster.AdminUpsertDefinitionInput{
		MonsterID: 9101, MonsterName: "测试捕捉怪", IsEnabled: true,
		Level: 1, Quality: 1, HP: 20, HPMax: 20, ATK: 10, DEF: 8, SPD: 8, MANA: 8,
		SkillIDs: []uint32{90001}, IsCapturable: true, CapturePetID: 103,
		CaptureRateBase: 5000, CaptureMinHPPct: 30, CaptureItemIDs: []uint32{2001},
	}
	if _, err := monsterRepo.CreateDefinitionForAdmin(context.Background(), monsterInput.Normalize()); err != nil {
		t.Fatalf("CreateDefinitionForAdmin() error = %v", err)
	}

	result, err := service.GrantCapturedPet(context.Background(), teststub.DemoPlayerID, 9101, "battle_capture", 1)
	if err != nil {
		t.Fatalf("GrantCapturedPet() error = %v", err)
	}
	if result.Pet.PetID != 103 {
		t.Fatalf("expected pet_id=103, got %d", result.Pet.PetID)
	}
	if result.Pet.GrantSource != pet.GrantSourceWildCapture {
		t.Fatalf("expected grant_source=%s, got %s", pet.GrantSourceWildCapture, result.Pet.GrantSource)
	}
	if result.Pet.CaptureMonsterID != 9101 {
		t.Fatalf("expected capture_monster_id=9101, got %d", result.Pet.CaptureMonsterID)
	}
	if result.Pet.GrowthAptitudes.HPApt < 8 || result.Pet.GrowthAptitudes.HPApt > 14 {
		t.Fatalf("hp aptitude out of range: %d", result.Pet.GrowthAptitudes.HPApt)
	}
}

func TestGrantCapturedPetRejectsNonCapturableMonster(t *testing.T) {
	monsterRepo := teststub.NewMonsterRepository()
	service := pet.NewService(teststub.NewPetRepository(), nil, monsterRepo, nil)
	_, err := service.GrantCapturedPet(context.Background(), teststub.DemoPlayerID, 9002, "battle_capture", 1)
	if err == nil {
		t.Fatal("expected error for non-capturable monster")
	}
}
