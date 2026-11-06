package wstransport

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/skill"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
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

// 合并详情只加载本次自有列表中的一只宠物，默认摘要请求保持原协议形状。
func TestPetListIncludesSelectedDetailOnDemand(t *testing.T) {
	for _, mode := range []string{"summary", "first", "selected", "missing"} {
		t.Run(mode, func(t *testing.T) {
			_, router, _, conn := buildWorldRouterForTest(t)
			mustHandleJSONPacket(t, router, conn, protocol.CmdPetListReq, 60, protocol.PetListReq{})
			var initial protocol.PetListResp
			if err := protocol.UnmarshalBody(conn.packets[0].Body, &initial); err != nil {
				t.Fatal(err)
			}
			if len(initial.Pets) < 2 {
				t.Fatal("need two owned pets")
			}
			request := protocol.PetListReq{IncludeSelectedDetail: mode != "summary"}
			expectedUID := initial.Pets[0].PetUID
			if mode == "selected" {
				request.SelectedPetUID = initial.Pets[1].PetUID
				expectedUID = request.SelectedPetUID
			}
			if mode == "missing" {
				request.SelectedPetUID = ^uint64(0)
			}
			conn.packets = nil
			mustHandleJSONPacket(t, router, conn, protocol.CmdPetListReq, 61, request)
			if len(conn.packets) != 1 || conn.packets[0].Cmd != protocol.CmdPetListResp || conn.packets[0].Seq != 61 {
				t.Fatalf("expected one correlated pet list response: %+v", conn.packets)
			}
			var actual protocol.PetListResp
			if err := protocol.UnmarshalBody(conn.packets[0].Body, &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual.Pets, initial.Pets) || !reflect.DeepEqual(actual.Lineup, initial.Lineup) {
				t.Fatal("summary list or lineup changed")
			}
			if mode == "summary" {
				if actual.SelectedPet != nil {
					t.Fatal("ordinary list must remain lightweight")
				}
				return
			}
			if actual.SelectedPet == nil || actual.SelectedPet.PetUID != expectedUID {
				t.Fatalf("unexpected selected pet: %+v", actual.SelectedPet)
			}
			mustHandleJSONPacket(t, router, conn, protocol.CmdPetSkillDetailReq, 62, protocol.PetSkillDetailReq{PetUID: expectedUID})
			var standalone protocol.PetSkillDetailResp
			if err := protocol.UnmarshalBody(conn.packets[1].Body, &standalone); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*actual.SelectedPet, standalone.Pet) {
				t.Fatal("merged detail differs from authoritative standalone detail")
			}
		})
	}
}

// 测试仓储保留原有数据实现，仅控制空列表或详情读取错误，并统计详情访问次数。
type petListDetailRepository struct {
	pet.Repository
	empty       bool
	detailErr   error
	detailReads int
}

// 摘要读取不会触发完整宠物详情访问。
func (r *petListDetailRepository) ListPetSummariesByPlayerID(ctx context.Context, playerID uint64) ([]pet.Pet, error) {
	if r.empty {
		return []pet.Pet{}, nil
	}
	return r.Repository.ListPetsByPlayerID(ctx, playerID)
}

// 编队摘要沿用现有仓储，空列表场景同步返回空编队。
func (r *petListDetailRepository) ListLineupSummariesByPlayerID(ctx context.Context, playerID uint64) ([]pet.LineupPet, error) {
	if r.empty {
		return []pet.LineupPet{}, nil
	}
	return r.Repository.ListLineupByPlayerID(ctx, playerID)
}

// 完整详情访问可注入数据库错误，验证合并失败不会被当成成功摘要返回。
func (r *petListDetailRepository) FindPetByUID(ctx context.Context, playerID uint64, petUID uint64) (pet.Pet, error) {
	r.detailReads++
	if r.detailErr != nil {
		return pet.Pet{}, r.detailErr
	}
	return r.Repository.FindPetByUID(ctx, playerID, petUID)
}

// 空列表不读取详情；详情失败只返回错误，不发送缺少完整属性的成功响应。
func TestPetListMergedDetailEmptyAndFailure(t *testing.T) {
	for _, empty := range []bool{true, false} {
		t.Run(map[bool]string{true: "empty", false: "detail failure"}[empty], func(t *testing.T) {
			_, router, _, conn := buildWorldRouterForTest(t)
			repo := &petListDetailRepository{Repository: teststub.NewPetRepository(), empty: empty, detailErr: errors.New("detail database unavailable")}
			router.petHandler.petService = pet.NewService(repo, nil, nil, nil)
			mustHandleJSONPacket(t, router, conn, protocol.CmdPetListReq, 63, protocol.PetListReq{IncludeSelectedDetail: true, SelectedPetUID: 999999})
			if len(conn.packets) != 1 || conn.packets[0].Seq != 63 {
				t.Fatal("expected one correlated result")
			}
			if empty {
				var payload protocol.PetListResp
				if err := protocol.UnmarshalBody(conn.packets[0].Body, &payload); err != nil {
					t.Fatal(err)
				}
				if conn.packets[0].Cmd != protocol.CmdPetListResp || len(payload.Pets) != 0 || payload.SelectedPet != nil || repo.detailReads != 0 {
					t.Fatal("empty list unexpectedly loaded detail")
				}
			} else if conn.packets[0].Cmd != protocol.CmdErrorPush || repo.detailReads != 1 {
				t.Fatal("detail failure must fail the merged request")
			}
		})
	}
}
