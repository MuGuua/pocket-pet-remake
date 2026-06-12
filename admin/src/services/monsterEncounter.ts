import { requestJSON } from './http';
import type {
  AdminMonsterEncounterDetail,
  AdminMonsterEncounterListFilters,
  AdminMonsterEncounterListResult,
  AdminUpsertMonsterEncounterPayload,
} from '../types/monsterEncounter';

export async function fetchAdminMonsterEncounters(params: {
  filters?: AdminMonsterEncounterListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminMonsterEncounterListResult> {
  const query = new URLSearchParams();
  if (params.filters?.entity_id?.trim()) query.set('entity_id', params.filters.entity_id.trim());
  if (params.filters?.name?.trim()) query.set('name', params.filters.name.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminMonsterEncounterListResult>({ url: `/api/admin/monster-encounters?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminMonsterEncounterDetail(entityID: number): Promise<AdminMonsterEncounterDetail> {
  return requestJSON<AdminMonsterEncounterDetail>({ url: `/api/admin/monster-encounters/${entityID}`, method: 'GET' });
}

export async function createAdminMonsterEncounter(payload: AdminUpsertMonsterEncounterPayload): Promise<AdminMonsterEncounterDetail> {
  return requestJSON<AdminMonsterEncounterDetail>({ url: '/api/admin/monster-encounters', method: 'POST', data: payload });
}

export async function updateAdminMonsterEncounter(entityID: number, payload: AdminUpsertMonsterEncounterPayload): Promise<AdminMonsterEncounterDetail> {
  return requestJSON<AdminMonsterEncounterDetail>({ url: `/api/admin/monster-encounters/${entityID}`, method: 'PUT', data: payload });
}

export async function deleteAdminMonsterEncounter(entityID: number): Promise<{ entity_id: number; deleted: boolean }> {
  return requestJSON<{ entity_id: number; deleted: boolean }>({ url: `/api/admin/monster-encounters/${entityID}`, method: 'DELETE' });
}
