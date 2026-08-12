package wstransport

import (
	"testing"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/skill"
	"pocket-pet-remake/server/internal/protocol"
)

func TestToProtocolSkillSlotEntryIncludesSkillVisualID(t *testing.T) {
	entry := pet.SkillSlotEntry{
		SlotIndex: 2,
		SkillID:   20001,
		Enabled:   true,
	}
	metadata := map[uint32]pet.SkillMetadata{
		20001: {
			SkillName:     "圣技幻影闪击",
			Description:   "快速攻击目标。",
			SkillVisualID: "pet_圣技_幻影闪击",
			SkillQuality:  skill.QualitySacred,
		},
	}

	actual := toProtocolSkillSlotEntry(entry, metadata)
	if actual.SkillVisualID != "pet_圣技_幻影闪击" {
		t.Fatalf("SkillVisualID = %q, want %q", actual.SkillVisualID, "pet_圣技_幻影闪击")
	}
	if actual.SkillQuality != skill.QualitySacred {
		t.Fatalf("SkillQuality = %q, want %q", actual.SkillQuality, skill.QualitySacred)
	}
}

// TestToProtocolPetDetailForListIncludesHudStatusFields 验证宠物列表摘要携带主界面 HUD 直接展示所需的权威状态。
func TestToProtocolPetDetailForListIncludesHudStatusFields(t *testing.T) {
	item := pet.Pet{
		PetUID:    20001,
		PetID:     101,
		Level:     5,
		Exp:       120,
		ExpToNext: 380,
		HP:        32,
		HPMax:     40,
		MANA:      16,
	}

	actual := toProtocolPetDetailForList(item)
	if actual.Exp != 120 || actual.ExpToNext != 380 {
		t.Fatalf("experience summary = %d/%d, want exp=120 exp_to_next=380", actual.Exp, actual.ExpToNext)
	}
	if actual.HP != 32 || actual.HPMax != 40 || actual.MANA != 16 {
		t.Fatalf("status summary = hp %d/%d mana %d, want hp 32/40 mana 16", actual.HP, actual.HPMax, actual.MANA)
	}
}

func TestRouterHandlePetList(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)

	packet := protocol.NewPacket(protocol.CmdPetListReq, 21, 0, nil)
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	response := conn.packets[0]
	if response.Cmd != protocol.CmdPetListResp {
		t.Fatalf("response.Cmd = %d, want %d", response.Cmd, protocol.CmdPetListResp)
	}

	var payload protocol.PetListResp
	if err := protocol.UnmarshalBody(response.Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if len(payload.Pets) != 3 {
		t.Fatalf("len(payload.Pets) = %d, want 3", len(payload.Pets))
	}
	if len(payload.Lineup) != 2 {
		t.Fatalf("len(payload.Lineup) = %d, want 2", len(payload.Lineup))
	}
	if !payload.Pets[0].InLineup {
		t.Fatalf("payload.Pets[0].InLineup = false, want true")
	}
	if payload.Pets[2].InLineup {
		t.Fatalf("payload.Pets[2].InLineup = true, want false")
	}
}

func TestRouterHandlePetLineupSet(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)

	body, err := protocol.MarshalBody(protocol.PetLineupSetReq{
		OpID:    1,
		PetUIDs: []uint64{20003},
	})
	if err != nil {
		t.Fatalf("MarshalBody() error = %v", err)
	}
	packet := protocol.NewPacket(protocol.CmdPetLineupSetReq, 22, 0, body)
	raw, err := protocol.EncodePacket(packet)
	if err != nil {
		t.Fatalf("EncodePacket() error = %v", err)
	}

	if err := router.Handle(conn, raw); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(conn.packets) != 1 {
		t.Fatalf("len(conn.packets) = %d, want 1", len(conn.packets))
	}

	response := conn.packets[0]
	if response.Cmd != protocol.CmdPetLineupSetResp {
		t.Fatalf("response.Cmd = %d, want %d", response.Cmd, protocol.CmdPetLineupSetResp)
	}

	var payload protocol.PetLineupSetResp
	if err := protocol.UnmarshalBody(response.Body, &payload); err != nil {
		t.Fatalf("UnmarshalBody() error = %v", err)
	}
	if !payload.Accepted {
		t.Fatalf("payload.Accepted = false, want true")
	}
	if len(payload.Lineup) != 1 {
		t.Fatalf("len(payload.Lineup) = %d, want 1", len(payload.Lineup))
	}
	if payload.Lineup[0].PetUID != 20003 {
		t.Fatalf("payload.Lineup[0].PetUID = %d, want 20003", payload.Lineup[0].PetUID)
	}
}
