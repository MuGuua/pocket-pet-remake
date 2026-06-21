import { requestJSON } from './http';
import type {
  AdminPetAttrConvertConfig,
  AdminPetAttrConvertListResult,
  AdminPetLevelConfig,
  AdminPetLevelConfigListResult,
  AdminUpsertPetAttrConvertPayload,
  AdminUpsertPetLevelConfigPayload,
} from '../types/petProgression';

export async function fetchAdminPetLevelConfigs(): Promise<AdminPetLevelConfig[]> {
  const result = await requestJSON<AdminPetLevelConfigListResult>({
    url: '/api/admin/pet-progression/level-configs',
    method: 'GET',
  });
  return result.items ?? [];
}

export async function updateAdminPetLevelConfig(
  level: number,
  payload: AdminUpsertPetLevelConfigPayload,
): Promise<AdminPetLevelConfig> {
  return requestJSON<AdminPetLevelConfig>({
    url: `/api/admin/pet-progression/level-configs/${level}`,
    method: 'PUT',
    data: payload,
  });
}

export async function fetchAdminPetAttrConvertConfigs(): Promise<AdminPetAttrConvertConfig[]> {
  const result = await requestJSON<AdminPetAttrConvertListResult>({
    url: '/api/admin/pet-progression/attr-convert-configs',
    method: 'GET',
  });
  return result.items ?? [];
}

export async function updateAdminPetAttrConvertConfig(
  attrType: string,
  payload: AdminUpsertPetAttrConvertPayload,
): Promise<AdminPetAttrConvertConfig> {
  return requestJSON<AdminPetAttrConvertConfig>({
    url: `/api/admin/pet-progression/attr-convert-configs/${attrType}`,
    method: 'PUT',
    data: payload,
  });
}
