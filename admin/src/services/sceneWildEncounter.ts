import { requestJSON } from './http';
import type {
  AdminSceneWildEncounterDetail,
  AdminSceneWildEncounterListFilters,
  AdminSceneWildEncounterListResult,
  AdminUpsertSceneWildEncounterPayload,
} from '../types/sceneWildEncounter';

export async function fetchAdminSceneWildEncounters(params: {
  filters?: AdminSceneWildEncounterListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminSceneWildEncounterListResult> {
  const query = new URLSearchParams();
  if (params.filters?.scene_id?.trim()) query.set('scene_id', params.filters.scene_id.trim());
  if (params.filters?.name?.trim()) query.set('name', params.filters.name.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminSceneWildEncounterListResult>({ url: `/api/admin/scene-wild-encounters?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminSceneWildEncounterDetail(sceneID: number): Promise<AdminSceneWildEncounterDetail> {
  return requestJSON<AdminSceneWildEncounterDetail>({ url: `/api/admin/scene-wild-encounters/${sceneID}`, method: 'GET' });
}

export async function createAdminSceneWildEncounter(payload: AdminUpsertSceneWildEncounterPayload): Promise<AdminSceneWildEncounterDetail> {
  return requestJSON<AdminSceneWildEncounterDetail>({ url: '/api/admin/scene-wild-encounters', method: 'POST', data: payload });
}

export async function updateAdminSceneWildEncounter(sceneID: number, payload: AdminUpsertSceneWildEncounterPayload): Promise<AdminSceneWildEncounterDetail> {
  return requestJSON<AdminSceneWildEncounterDetail>({ url: `/api/admin/scene-wild-encounters/${sceneID}`, method: 'PUT', data: payload });
}

export async function deleteAdminSceneWildEncounter(sceneID: number): Promise<{ scene_id: number; deleted: boolean }> {
  return requestJSON<{ scene_id: number; deleted: boolean }>({ url: `/api/admin/scene-wild-encounters/${sceneID}`, method: 'DELETE' });
}
