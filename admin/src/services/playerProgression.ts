import { requestJSON } from './http';
import type {
  AdminPlayerAttrConvertConfig,
  AdminPlayerAttrConvertListResult,
  AdminPlayerLevelConfig,
  AdminPlayerLevelConfigListResult,
  AdminUpsertPlayerAttrConvertPayload,
  AdminUpsertPlayerLevelConfigPayload,
} from '../types/playerProgression';

export async function fetchAdminPlayerLevelConfigs(): Promise<AdminPlayerLevelConfig[]> {
  const result = await requestJSON<AdminPlayerLevelConfigListResult>({
    url: '/api/admin/player-progression/level-configs',
    method: 'GET',
  });
  return result.items ?? [];
}

export async function updateAdminPlayerLevelConfig(
  level: number,
  payload: AdminUpsertPlayerLevelConfigPayload,
): Promise<AdminPlayerLevelConfig> {
  return requestJSON<AdminPlayerLevelConfig>({
    url: `/api/admin/player-progression/level-configs/${level}`,
    method: 'PUT',
    data: payload,
  });
}

export async function fetchAdminPlayerAttrConvertConfigs(): Promise<AdminPlayerAttrConvertConfig[]> {
  const result = await requestJSON<AdminPlayerAttrConvertListResult>({
    url: '/api/admin/player-progression/attr-convert-configs',
    method: 'GET',
  });
  return result.items ?? [];
}

export async function updateAdminPlayerAttrConvertConfig(
  configID: number,
  payload: AdminUpsertPlayerAttrConvertPayload,
): Promise<AdminPlayerAttrConvertConfig> {
  return requestJSON<AdminPlayerAttrConvertConfig>({
    url: `/api/admin/player-progression/attr-convert-configs/${configID}`,
    method: 'PUT',
    data: payload,
  });
}
