import { requestJSON } from './http';
import type {
  AdminMonsterDefinitionDetail,
  AdminMonsterDefinitionListFilters,
  AdminMonsterDefinitionListResult,
  AdminMonsterBattleRewardEntry,
  AdminReplaceMonsterBattleRewardsPayload,
  AdminUpsertMonsterDefinitionPayload,
} from '../types/monsterDefinition';

export async function fetchAdminMonsterDefinitions(params: {
  filters?: AdminMonsterDefinitionListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminMonsterDefinitionListResult> {
  const query = new URLSearchParams();
  if (params.filters?.monster_id?.trim()) query.set('monster_id', params.filters.monster_id.trim());
  if (params.filters?.name?.trim()) query.set('name', params.filters.name.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminMonsterDefinitionListResult>({ url: `/api/admin/monster-definitions?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminMonsterDefinitionDetail(monsterID: number): Promise<AdminMonsterDefinitionDetail> {
  return requestJSON<AdminMonsterDefinitionDetail>({ url: `/api/admin/monster-definitions/${monsterID}`, method: 'GET' });
}

export async function createAdminMonsterDefinition(payload: AdminUpsertMonsterDefinitionPayload): Promise<AdminMonsterDefinitionDetail> {
  return requestJSON<AdminMonsterDefinitionDetail>({ url: '/api/admin/monster-definitions', method: 'POST', data: payload });
}

export async function updateAdminMonsterDefinition(monsterID: number, payload: AdminUpsertMonsterDefinitionPayload): Promise<AdminMonsterDefinitionDetail> {
  return requestJSON<AdminMonsterDefinitionDetail>({ url: `/api/admin/monster-definitions/${monsterID}`, method: 'PUT', data: payload });
}

export async function deleteAdminMonsterDefinition(monsterID: number): Promise<{ monster_id: number; deleted: boolean }> {
  return requestJSON<{ monster_id: number; deleted: boolean }>({ url: `/api/admin/monster-definitions/${monsterID}`, method: 'DELETE' });
}

export async function fetchAdminMonsterBattleRewards(monsterID: number): Promise<AdminMonsterBattleRewardEntry[]> {
  const result = await requestJSON<{ items: AdminMonsterBattleRewardEntry[] }>({
    url: `/api/admin/monster-definitions/${monsterID}/battle-rewards`,
    method: 'GET',
  });
  return result.items ?? [];
}

export async function updateAdminMonsterBattleRewards(
  monsterID: number,
  payload: AdminReplaceMonsterBattleRewardsPayload,
): Promise<AdminMonsterBattleRewardEntry[]> {
  const result = await requestJSON<{ items: AdminMonsterBattleRewardEntry[] }>({
    url: `/api/admin/monster-definitions/${monsterID}/battle-rewards`,
    method: 'PUT',
    data: payload,
  });
  return result.items ?? [];
}
