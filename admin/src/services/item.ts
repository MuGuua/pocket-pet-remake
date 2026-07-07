import { requestJSON } from './http';
import type { AdminItemDetail, AdminItemListFilters, AdminItemListResult, AdminItemSummary, AdminUpsertItemPayload } from '../types/item';

const ADMIN_ITEM_LIST_PAGE_SIZE = 100;

export async function fetchAdminItems(params: { filters?: AdminItemListFilters; page?: number; pageSize?: number }): Promise<AdminItemListResult> {
  const query = new URLSearchParams();
  if (params.filters?.item_id?.trim()) query.set('item_id', params.filters.item_id.trim());
  if (params.filters?.item_type?.trim()) query.set('item_type', params.filters.item_type.trim());
  if (params.filters?.exclude_item_type?.trim()) query.set('exclude_item_type', params.filters.exclude_item_type.trim());
  if (params.filters?.keyword?.trim()) query.set('keyword', params.filters.keyword.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminItemListResult>({ url: `/api/admin/items?${query.toString()}`, method: 'GET' });
}

// 分页拉取全部物品，供怪物捕捉道具等引用字段复用。
export async function fetchAllAdminItems(filters?: AdminItemListFilters) {
  const items: AdminItemSummary[] = [];
  let page = 1;
  let total = 0;
  do {
    const result = await fetchAdminItems({
      filters,
      page,
      pageSize: ADMIN_ITEM_LIST_PAGE_SIZE,
    });
    items.push(...result.items);
    total = result.total;
    page += 1;
  } while (items.length < total);
  return items;
}

export async function fetchAdminItemDetail(itemID: number): Promise<AdminItemDetail> {
  return requestJSON<AdminItemDetail>({ url: `/api/admin/items/${itemID}`, method: 'GET' });
}

export async function createAdminItem(payload: AdminUpsertItemPayload): Promise<AdminItemDetail> {
  return requestJSON<AdminItemDetail>({ url: '/api/admin/items', method: 'POST', data: payload });
}

export async function updateAdminItem(itemID: number, payload: AdminUpsertItemPayload): Promise<AdminItemDetail> {
  return requestJSON<AdminItemDetail>({ url: `/api/admin/items/${itemID}`, method: 'PUT', data: payload });
}

export async function deleteAdminItem(itemID: number): Promise<{ item_id: number; deleted: boolean }> {
  return requestJSON<{ item_id: number; deleted: boolean }>({ url: `/api/admin/items/${itemID}`, method: 'DELETE' });
}
