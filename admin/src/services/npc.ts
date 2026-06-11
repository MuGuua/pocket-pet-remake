import { requestJSON } from './http';
import type {
  AdminCreateNPCEntityPayload,
  AdminCreateNPCMenuEntryPayload,
  AdminNPCEntityDetail,
  AdminNPCEntityFilters,
  AdminNPCEntityListResult,
  AdminNPCMenuEntryDetail,
  AdminNPCMenuEntryFilters,
  AdminNPCMenuEntryListResult,
  AdminUpdateNPCEntityPayload,
  AdminUpdateNPCMenuEntryPayload,
} from '../types/npc';

export async function fetchAdminNPCEntities(params: { filters?: AdminNPCEntityFilters; page?: number; pageSize?: number }): Promise<AdminNPCEntityListResult> {
  const query = new URLSearchParams();
  if (params.filters?.entity_id?.trim()) query.set('entity_id', params.filters.entity_id.trim());
  if (params.filters?.scene_id?.trim()) query.set('scene_id', params.filters.scene_id.trim());
  if (params.filters?.entity_type?.trim()) query.set('entity_type', params.filters.entity_type.trim());
  if (params.filters?.status?.trim()) query.set('status', params.filters.status.trim());
  if (params.filters?.name?.trim()) query.set('name', params.filters.name.trim());
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminNPCEntityListResult>({ url: `/api/admin/npcs/entities?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminNPCEntityDetail(entityID: number): Promise<AdminNPCEntityDetail> {
  return requestJSON<AdminNPCEntityDetail>({ url: `/api/admin/npcs/entities/${entityID}`, method: 'GET' });
}

export async function createAdminNPCEntity(payload: AdminCreateNPCEntityPayload): Promise<AdminNPCEntityDetail> {
  return requestJSON<AdminNPCEntityDetail>({ url: '/api/admin/npcs/entities', method: 'POST', data: payload });
}

export async function updateAdminNPCEntity(entityID: number, payload: AdminUpdateNPCEntityPayload): Promise<AdminNPCEntityDetail> {
  return requestJSON<AdminNPCEntityDetail>({ url: `/api/admin/npcs/entities/${entityID}`, method: 'PUT', data: payload });
}

export async function deleteAdminNPCEntity(entityID: number): Promise<{ entity_id: number; deleted: boolean }> {
  return requestJSON<{ entity_id: number; deleted: boolean }>({ url: `/api/admin/npcs/entities/${entityID}`, method: 'DELETE' });
}

export async function fetchAdminNPCMenuEntries(params: { filters?: AdminNPCMenuEntryFilters; page?: number; pageSize?: number }): Promise<AdminNPCMenuEntryListResult> {
  const query = new URLSearchParams();
  if (params.filters?.entity_id?.trim()) query.set('entity_id', params.filters.entity_id.trim());
  if (params.filters?.entry_id?.trim()) query.set('entry_id', params.filters.entry_id.trim());
  if (params.filters?.status?.trim()) query.set('status', params.filters.status.trim());
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminNPCMenuEntryListResult>({ url: `/api/admin/npcs/menu-entries?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminNPCMenuEntryDetail(entityID: number, entryID: string): Promise<AdminNPCMenuEntryDetail> {
  return requestJSON<AdminNPCMenuEntryDetail>({ url: `/api/admin/npcs/menu-entries/${entityID}/${encodeURIComponent(entryID)}`, method: 'GET' });
}

export async function createAdminNPCMenuEntry(payload: AdminCreateNPCMenuEntryPayload): Promise<AdminNPCMenuEntryDetail> {
  return requestJSON<AdminNPCMenuEntryDetail>({ url: '/api/admin/npcs/menu-entries', method: 'POST', data: payload });
}

export async function updateAdminNPCMenuEntry(entityID: number, entryID: string, payload: AdminUpdateNPCMenuEntryPayload): Promise<AdminNPCMenuEntryDetail> {
  return requestJSON<AdminNPCMenuEntryDetail>({ url: `/api/admin/npcs/menu-entries/${entityID}/${encodeURIComponent(entryID)}`, method: 'PUT', data: payload });
}

export async function deleteAdminNPCMenuEntry(entityID: number, entryID: string): Promise<{ entity_id: number; entry_id: string; deleted: boolean }> {
  return requestJSON<{ entity_id: number; entry_id: string; deleted: boolean }>({ url: `/api/admin/npcs/menu-entries/${entityID}/${encodeURIComponent(entryID)}`, method: 'DELETE' });
}
