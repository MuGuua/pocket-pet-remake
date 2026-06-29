package bag

import "testing"

func TestParseConfiguredUseEffects(t *testing.T) {
	effects, err := ParseConfiguredUseEffects([]byte(`{
		"version": 1,
		"use_effects": [
			{"category":"pet","field_key":"hp","operation":"add","value":10},
			{"category":"system","field_key":"pet_talisman_slot_unlock","operation":"set","value":true}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseConfiguredUseEffects() error = %v", err)
	}
	if len(effects) != 2 {
		t.Fatalf("len(effects) = %d, want 2", len(effects))
	}
	if effects[0].Category != ConfiguredUseEffectCategoryPet || effects[0].FieldKey != "hp" || effects[0].Value != 10 {
		t.Fatalf("effects[0] = %+v, want pet hp add 10", effects[0])
	}
	if !effects[1].IsBoolean || !effects[1].BoolValue {
		t.Fatalf("effects[1] = %+v, want boolean unlock true", effects[1])
	}

	subtractEffects, err := ParseConfiguredUseEffects([]byte(`{
		"use_effects": [
			{"category":"player","field_key":"level","operation":"subtract","value":1}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseConfiguredUseEffects(subtract) error = %v", err)
	}
	if len(subtractEffects) != 1 || subtractEffects[0].Operation != ConfiguredUseEffectOperationSubtract {
		t.Fatalf("subtractEffects = %+v, want single subtract entry", subtractEffects)
	}
}

func TestRequiresTargetsFromConfiguredUseEffects(t *testing.T) {
	effects := []ConfiguredUseEffect{
		{Category: ConfiguredUseEffectCategoryPlayer, FieldKey: "gold", Operation: ConfiguredUseEffectOperationAdd, Value: 1},
		{Category: ConfiguredUseEffectCategoryEquipment, FieldKey: "enhance_level", Operation: ConfiguredUseEffectOperationSet, Value: 5},
	}
	if RequiresPetTarget(effects) {
		t.Fatal("RequiresPetTarget() = true, want false")
	}
	if !RequiresEquipmentTarget(effects) {
		t.Fatal("RequiresEquipmentTarget() = false, want true")
	}
}
