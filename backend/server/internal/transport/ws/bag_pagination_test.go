package wstransport

import (
	"context"
	"testing"

	"pocket-pet-remake/server/internal/module/bag"
	"pocket-pet-remake/server/internal/protocol"
	"pocket-pet-remake/server/internal/teststub"
)

// 分页能力仓储记录请求路径；完整读取仍交给原仓库，验证仓库存取快照不会走分页。
type pagedBagRepository struct {
	bag.Repository
	pageCalls int
	fullCalls int
	query     bag.RuntimePageQuery
}

func (r *pagedBagRepository) ListRuntimeBagPage(_ context.Context, _ uint64, q bag.RuntimePageQuery) (*bag.RuntimeContainerPage, error) {
	r.pageCalls++
	r.query = q
	snapshot := bag.RuntimeContainerSnapshot{ContainerType: bag.ContainerTypeBag, Capacity: 80, MaxCapacity: 80, UsedSlots: 60}
	for i := uint32(1); i <= 60; i++ {
		snapshot.Items = append(snapshot.Items, bag.RuntimeItemSnapshot{SlotIndex: 80 - i, ItemID: uint64(i), Quantity: 1})
	}
	return q.Apply(snapshot), nil
}
func (r *pagedBagRepository) ListRuntimeContainer(ctx context.Context, id uint64, kind string) (*bag.RuntimeContainerSnapshot, error) {
	r.fullCalls++
	return r.Repository.ListRuntimeContainer(ctx, id, kind)
}

// 背包列表只走一次分页读取，计数与槽位正确；完整容器查询不截断、不额外分页。
func TestBagListUsesRepositoryPageOnly(t *testing.T) {
	_, router, _, conn := buildWorldRouterForTest(t)
	repo := &pagedBagRepository{Repository: teststub.NewBagRepository()}
	router.bagHandler.bagService = bag.NewService(repo)
	mustHandleJSONPacket(t, router, conn, protocol.CmdBagListReq, 901, protocol.BagListReq{Page: 2, PageSize: 28, Category: "all"})
	if repo.pageCalls != 1 || repo.fullCalls != 0 || len(conn.packets) != 1 {
		t.Fatal("unexpected list query path")
	}
	var result protocol.BagListResp
	if err := protocol.UnmarshalBody(conn.packets[0].Body, &result); err != nil {
		t.Fatal(err)
	}
	c := result.Container
	if c.UsedSlots != 60 || c.TotalItems != 60 || c.Page != 2 || c.PageSize != 28 || len(c.Items) != 28 {
		t.Fatalf("wrong counts or page: %+v", c)
	}
	if c.Items[0].SlotIndex != 51 || c.Items[27].SlotIndex != 24 {
		t.Fatal("physical slot identities changed")
	}
	for _, container := range []string{bag.ContainerTypeBag, bag.ContainerTypeWarehouse} {
		conn.packets = nil
		mustHandleJSONPacket(t, router, conn, protocol.CmdContainerListReq, 902, protocol.ContainerListReq{ContainerType: container})
		if repo.pageCalls != 1 || len(conn.packets) != 1 || conn.packets[0].Cmd != protocol.CmdContainerListResp {
			t.Fatal("complete container query changed")
		}
	}
	if repo.fullCalls != 2 {
		t.Fatal("complete snapshots must still use full repository read")
	}
}
