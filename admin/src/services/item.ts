import { requestJSON } from './http';
import type { AdminItemDetail, AdminItemListFilters, AdminItemListResult, AdminUpsertItemPayload } from '../types/item';

export async function fetchAdminItems(params: { filters?: AdminItemListFilters; page?: number; pageSize?: number }): Promise<AdminItemListResult> {
  const query = new URLSearchParams();
  if (params.filters?.item_id?.trim()) query.set('item_id', params.filters.item_id.trim());
  if (params.filters?.item_type?.trim()) query.set('item_type', params.filters.item_type.trim());
  if (params.filters?.keyword?.trim()) query.set('keyword', params.filters.keyword.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminItemListResult>({ url: `/api/admin/items?${query.toString()}`, method: 'GET' });
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
