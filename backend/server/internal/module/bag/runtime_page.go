package bag

import "strings"

// 运行时背包分页参数；分类与默认值在领域层统一，SQL 和兼容仓储共用同一契约。
type RuntimePageQuery struct {
	Page     uint32
	PageSize uint32
	Category string
}

// 补齐旧客户端的空参数，未知分类按全部处理，不改变原页长约定。
func (q RuntimePageQuery) Normalize() RuntimePageQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 28
	}
	q.Category = strings.ToLower(strings.TrimSpace(q.Category))
	switch q.Category {
	case "all", "equipment", "other", "enhance_material":
	default:
		q.Category = "all"
	}
	return q
}

// 数据页保留全容器占用数与分类总数，Items 只含当前页，物品槽位保持数据库原值。
type RuntimeContainerPage struct {
	Snapshot   RuntimeContainerSnapshot
	Query      RuntimePageQuery
	TotalItems uint32
}

// 仅用于不支持数据库分页的兼容仓储，顺序沿用其返回的进入时间/槽位顺序。
func (q RuntimePageQuery) Apply(snapshot RuntimeContainerSnapshot) *RuntimeContainerPage {
	q = q.Normalize()
	filtered := make([]RuntimeItemSnapshot, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		equipment := strings.EqualFold(item.ItemType, "equipment")
		material := strings.EqualFold(item.ItemSubType, ItemSubTypeEquipmentEnhance)
		if q.Category == "all" || (q.Category == "equipment" && equipment) || (q.Category == "enhance_material" && material) || (q.Category == "other" && !equipment && !material) {
			filtered = append(filtered, item)
		}
	}
	total := uint32(len(filtered))
	// 先扩成 64 位再乘，极大页码也不会回绕到第一页。
	offset := (uint64(q.Page) - 1) * uint64(q.PageSize)
	snapshot.Items = []RuntimeItemSnapshot{}
	if offset < uint64(total) {
		end := offset + uint64(q.PageSize)
		if end > uint64(total) {
			end = uint64(total)
		}
		snapshot.Items = append(snapshot.Items, filtered[offset:end]...)
	}
	return &RuntimeContainerPage{Snapshot: snapshot, Query: q, TotalItems: total}
}
