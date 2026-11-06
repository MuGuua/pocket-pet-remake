package bag

import (
	"context"
	"errors"
	"io"
	"testing"
)

// 混合分类页保留全容器格数、分类总数与真实槽位；极大页码不能回绕。
func TestRuntimePageContract(t *testing.T) {
	snapshot := RuntimeContainerSnapshot{ContainerType: ContainerTypeBag, Capacity: 40, UsedSlots: 5, Items: []RuntimeItemSnapshot{
		{SlotIndex: 9, ItemType: "equipment"},
		{SlotIndex: 2, ItemType: "consumable", ItemSubType: ItemSubTypeEquipmentEnhance},
		{SlotIndex: 30, ItemType: "EQUIPMENT"},
		{SlotIndex: 8, ItemType: "consumable"},
		{SlotIndex: 17, ItemType: "misc"},
	}}
	for _, tc := range []struct {
		category          string
		page, size, total uint32
		slots             []uint32
	}{
		{"all", 2, 2, 5, []uint32{30, 8}},
		{" equipment ", 1, 2, 2, []uint32{9, 30}},
		{"enhance_material", 1, 2, 1, []uint32{2}},
		{"other", 1, 2, 2, []uint32{8, 17}},
		{"unknown", 0, 0, 5, []uint32{9, 2, 30, 8, 17}},
		{"all", ^uint32(0), ^uint32(0), 5, nil},
	} {
		actual := (RuntimePageQuery{Page: tc.page, PageSize: tc.size, Category: tc.category}).Apply(snapshot)
		if actual.Snapshot.UsedSlots != 5 || actual.TotalItems != tc.total || len(actual.Snapshot.Items) != len(tc.slots) {
			t.Fatalf("wrong page metadata: %+v", actual)
		}
		for i, slot := range tc.slots {
			if actual.Snapshot.Items[i].SlotIndex != slot {
				t.Fatal("slot identity or ordering changed")
			}
		}
		if len(snapshot.Items) != 5 {
			t.Fatal("source snapshot changed")
		}
	}
}

// 最小仓储只提供全量读取，用于验证兼容实现无需增加新接口。
type pageFallbackRepo struct {
	Repository
	snapshot  *RuntimeContainerSnapshot
	fullCalls int
}

func (r *pageFallbackRepo) ListRuntimeContainer(context.Context, uint64, string) (*RuntimeContainerSnapshot, error) {
	r.fullCalls++
	return r.snapshot, nil
}

// 分页仓储记录调用次数和规范化参数，可模拟瞬断或业务失败。
type pageRepoStub struct {
	*pageFallbackRepo
	calls int
	err   error
	query RuntimePageQuery
}

func (r *pageRepoStub) ListRuntimeBagPage(_ context.Context, _ uint64, q RuntimePageQuery) (*RuntimeContainerPage, error) {
	r.calls++
	r.query = q
	if r.err != nil && (r.err != io.EOF || r.calls == 1) {
		return nil, r.err
	}
	return q.Apply(*r.snapshot), nil
}

// 生产分页优先、瞬断仅重试一次，业务错误不能回退成一次额外的全量查询。
func TestRuntimePageRepositorySelection(t *testing.T) {
	source := &RuntimeContainerSnapshot{ContainerType: ContainerTypeBag, UsedSlots: 1, Items: []RuntimeItemSnapshot{{SlotIndex: 7}}}
	for _, failure := range []error{nil, io.EOF, errors.New("database failure")} {
		repo := &pageRepoStub{pageFallbackRepo: &pageFallbackRepo{snapshot: source}, err: failure}
		result, err := NewService(repo).ListRuntimeBagPage(context.Background(), 1, RuntimePageQuery{})
		expected := 1
		if failure == io.EOF {
			expected = 2
		}
		if repo.calls != expected || repo.fullCalls != 0 || repo.query.Page != 1 || repo.query.PageSize != 28 || repo.query.Category != "all" {
			t.Fatal("unexpected repository path or retry")
		}
		if failure != nil && failure != io.EOF {
			if err == nil {
				t.Fatal("error swallowed")
			}
		} else if err != nil || len(result.Snapshot.Items) != 1 {
			t.Fatal("page unavailable")
		}
	}
	fallback := &pageFallbackRepo{snapshot: source}
	if _, err := NewService(fallback).ListRuntimeBagPage(context.Background(), 1, RuntimePageQuery{}); err != nil || fallback.fullCalls != 1 {
		t.Fatal("legacy repository fallback failed")
	}
}
