import { requestJSON } from './http';
import type { AdminBagDetail, AdminBagListFilters, AdminBagListResult, AdminCreateBagPayload, AdminUpdateBagPayload } from '../types/bag';

export async function fetchAdminBags(params: {
  filters?: AdminBagListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminBagListResult> {
  const query = new URLSearchParams();
  if (params.filters?.record_id?.trim()) query.set('record_id', params.filters.record_id.trim());
  if (params.filters?.player_id?.trim()) query.set('player_id', params.filters.player_id.trim());
  if (params.filters?.container_type?.trim()) query.set('container_type', params.filters.container_type.trim());
  if (params.filters?.item_id?.trim()) query.set('item_id', params.filters.item_id.trim());
  if (params.filters?.item_uid?.trim()) query.set('item_uid', params.filters.item_uid.trim());
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminBagListResult>({ url: `/api/admin/bags?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminBagDetail(recordID: number): Promise<AdminBagDetail> {
  return requestJSON<AdminBagDetail>({ url: `/api/admin/bags/${recordID}`, method: 'GET' });
}

export async function createAdminBagItem(payload: AdminCreateBagPayload): Promise<AdminBagDetail> {
  return requestJSON<AdminBagDetail>({ url: '/api/admin/bags', method: 'POST', data: payload });
}

export async function updateAdminBagItem(recordID: number, payload: AdminUpdateBagPayload): Promise<AdminBagDetail> {
  return requestJSON<AdminBagDetail>({ url: `/api/admin/bags/${recordID}`, method: 'PUT', data: payload });
}

export async function deleteAdminBagItem(recordID: number): Promise<{ record_id: number; deleted: boolean }> {
  return requestJSON<{ record_id: number; deleted: boolean }>({ url: `/api/admin/bags/${recordID}`, method: 'DELETE' });
}
