import { requestJSON } from './http';
import type {
  AdminEquipmentDetail,
  AdminEquipmentListFilters,
  AdminEquipmentListResult,
  AdminUpsertEquipmentPayload,
} from '../types/equipmentDefinition';

export async function fetchAdminEquipmentDefinitions(params: {
  filters?: AdminEquipmentListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminEquipmentListResult> {
  const query = new URLSearchParams();
  if (params.filters?.item_id?.trim()) {
    query.set('item_id', params.filters.item_id.trim());
  }
  if (params.filters?.equip_slot?.trim()) {
    query.set('equip_slot', params.filters.equip_slot.trim());
  }
  if (params.filters?.set_id?.trim()) {
    query.set('set_id', params.filters.set_id.trim());
  }
  if (params.filters?.keyword?.trim()) {
    query.set('keyword', params.filters.keyword.trim());
  }
  if (params.filters?.is_enabled?.trim()) {
    query.set('is_enabled', params.filters.is_enabled.trim());
  }
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminEquipmentListResult>({
    url: `/api/admin/equipment-definitions?${query.toString()}`,
    method: 'GET',
  });
}

export async function fetchAdminEquipmentDetail(itemID: number): Promise<AdminEquipmentDetail> {
  return requestJSON<AdminEquipmentDetail>({
    url: `/api/admin/equipment-definitions/${itemID}`,
    method: 'GET',
  });
}

export async function createAdminEquipmentDefinition(
  payload: AdminUpsertEquipmentPayload,
): Promise<AdminEquipmentDetail> {
  return requestJSON<AdminEquipmentDetail>({
    url: '/api/admin/equipment-definitions',
    method: 'POST',
    data: payload,
  });
}

export async function updateAdminEquipmentDefinition(
  itemID: number,
  payload: AdminUpsertEquipmentPayload,
): Promise<AdminEquipmentDetail> {
  return requestJSON<AdminEquipmentDetail>({
    url: `/api/admin/equipment-definitions/${itemID}`,
    method: 'PUT',
    data: payload,
  });
}

export async function deleteAdminEquipmentDefinition(itemID: number): Promise<{ item_id: number; deleted: boolean }> {
  return requestJSON<{ item_id: number; deleted: boolean }>({
    url: `/api/admin/equipment-definitions/${itemID}`,
    method: 'DELETE',
  });
}
