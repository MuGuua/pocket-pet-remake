import { requestJSON } from './http';
import type {
  AdminCreatePlayerPayload,
  AdminPlayerDetail,
  AdminPlayerListFilters,
  AdminPlayerListResult,
  AdminUpdatePlayerPayload,
} from '../types/player';

export async function fetchAdminPlayers(params: {
  filters?: AdminPlayerListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminPlayerListResult> {
  const query = new URLSearchParams();
  if (params.filters?.player_id?.trim()) {
    query.set('player_id', params.filters.player_id.trim());
  }
  if (params.filters?.name?.trim()) {
    query.set('name', params.filters.name.trim());
  }
  if (params.filters?.status?.trim()) {
    query.set('status', params.filters.status.trim());
  }
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));

  return requestJSON<AdminPlayerListResult>({
    url: `/api/admin/players?${query.toString()}`,
    method: 'GET',
  });
}

export async function fetchAdminPlayerDetail(playerID: number): Promise<AdminPlayerDetail> {
  return requestJSON<AdminPlayerDetail>({
    url: `/api/admin/players/${playerID}`,
    method: 'GET',
  });
}

export async function createAdminPlayer(payload: AdminCreatePlayerPayload): Promise<AdminPlayerDetail> {
  return requestJSON<AdminPlayerDetail>({
    url: '/api/admin/players',
    method: 'POST',
    data: payload,
  });
}

export async function updateAdminPlayer(playerID: number, payload: AdminUpdatePlayerPayload): Promise<AdminPlayerDetail> {
  return requestJSON<AdminPlayerDetail>({
    url: `/api/admin/players/${playerID}`,
    method: 'PUT',
    data: payload,
  });
}

export async function deleteAdminPlayer(playerID: number): Promise<{ player_id: number; deleted: boolean }> {
  return requestJSON<{ player_id: number; deleted: boolean }>({
    url: `/api/admin/players/${playerID}`,
    method: 'DELETE',
  });
}
