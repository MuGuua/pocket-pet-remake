package skill

import "testing"

func TestValidateAdminSkillDefinitionInputRejectsPassiveBasicAttack(t *testing.T) {
	input := AdminUpsertInput{
		SkillID:           2001,
		SkillName:         "被动普攻",
		ActivationMode:    ActivationModePassive,
		IsBasicAttack:     true,
		TargetType:        "self",
		TargetCount:       0,
		PreferredTargetHP: "",
		EnergyCost:        0,
	}.Normalize()

	if err := validateAdminSkillDefinitionInput(input); err == nil {
		t.Fatal("validateAdminSkillDefinitionInput() = nil, want invalid passive basic attack")
	}
}

func TestValidateAdminSkillDefinitionInputAllowsPassiveSelfBuff(t *testing.T) {
	input := AdminUpsertInput{
		SkillID:        2002,
		SkillName:      "嗜血被动",
		ActivationMode: ActivationModePassive,
		SkillType:      "support",
		TargetType:     "self",
		EnergyCost:     0,
	}.Normalize()

	if err := validateAdminSkillDefinitionInput(input); err != nil {
		t.Fatalf("validateAdminSkillDefinitionInput() error = %v, want nil", err)
	}
}

func TestValidateAdminSkillDefinitionInputRejectsInvalidPassiveAttributeBonusMode(t *testing.T) {
	input := AdminUpsertInput{
		SkillID:           2003,
		SkillName:         "异常属性被动",
		ActivationMode:    ActivationModePassive,
		SkillType:         "support",
		TargetType:        "self",
		PassiveAttrKey:    PassiveAttrKeyCritRatePct,
		PassiveAttrMode:   PassiveAttrModePercent,
		PassiveAttrValue:  20,
		PreferredTargetHP: "",
		EnergyCost:        0,
	}.Normalize()

	if err := validateAdminSkillDefinitionInput(input); err == nil {
		t.Fatal("validateAdminSkillDefinitionInput() = nil, want invalid passive attribute mode")
	}
}

func TestValidateAdminSkillDefinitionInputAllowsPassiveAttributeBonus(t *testing.T) {
	input := AdminUpsertInput{
		SkillID:          2004,
		SkillName:        "迅捷被动",
		ActivationMode:   ActivationModePassive,
		SkillType:        "support",
		TargetType:       "self",
		PassiveAttrKey:   PassiveAttrKeySPD,
		PassiveAttrMode:  PassiveAttrModePercent,
		PassiveAttrValue: 50,
		EnergyCost:       0,
	}.Normalize()

	if err := validateAdminSkillDefinitionInput(input); err != nil {
		t.Fatalf("validateAdminSkillDefinitionInput() error = %v, want nil", err)
	}
}

func TestAdminUpsertInputNormalizeDoesNotReapplyActiveDefaultsToPassiveSkill(t *testing.T) {
	input := AdminUpsertInput{
		SkillID:        2005,
		SkillName:      "已清空表现的被动",
		ActivationMode: ActivationModePassive,
		SkillType:      "support",
		TargetType:     "self",
		AttackPct:      0,
		AnimationKey:   "",
		CastColor:      "",
		ImpactColor:    "",
		Projectile:     false,
	}.Normalize()

	if input.AttackPct != 0 {
		t.Fatalf("Normalize() AttackPct = %d, want 0", input.AttackPct)
	}
	if input.AnimationKey != "" {
		t.Fatalf("Normalize() AnimationKey = %q, want empty", input.AnimationKey)
	}
	if input.CastColor != "" {
		t.Fatalf("Normalize() CastColor = %q, want empty", input.CastColor)
	}
	if input.ImpactColor != "" {
		t.Fatalf("Normalize() ImpactColor = %q, want empty", input.ImpactColor)
	}
}
