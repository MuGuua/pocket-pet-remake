package wstransport

import (
	"testing"

	"pocket-pet-remake/server/internal/protocol"
)

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
		PetUIDs: []uint64{20003, 20001},
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
	if len(payload.Lineup) != 2 {
		t.Fatalf("len(payload.Lineup) = %d, want 2", len(payload.Lineup))
	}
	if payload.Lineup[0].PetUID != 20003 {
		t.Fatalf("payload.Lineup[0].PetUID = %d, want 20003", payload.Lineup[0].PetUID)
	}
}
