import { requestJSON } from './http';
import type {
  AdminEnhanceSuccessConfig,
  AdminEnhanceSuccessConfigListResult,
  AdminUpsertEnhanceSuccessConfigPayload,
} from '../types/equipmentEnhanceSuccess';

export async function fetchAdminEnhanceSuccessConfigs(
  requiredLevelMin?: number,
): Promise<AdminEnhanceSuccessConfig[]> {
  const query = new URLSearchParams();
  if (requiredLevelMin != null && requiredLevelMin > 0) {
    query.set('required_level_min', String(requiredLevelMin));
  }
  const suffix = query.toString() ? `?${query.toString()}` : '';
  const result = await requestJSON<AdminEnhanceSuccessConfigListResult>({
    url: `/api/admin/equipment-enhance-success-configs${suffix}`,
    method: 'GET',
  });
  return result.items ?? [];
}

export async function updateAdminEnhanceSuccessConfig(
  requiredLevelMin: number,
  targetLevel: number,
  payload: AdminUpsertEnhanceSuccessConfigPayload,
): Promise<AdminEnhanceSuccessConfig> {
  return requestJSON<AdminEnhanceSuccessConfig>({
    url: `/api/admin/equipment-enhance-success-configs/${requiredLevelMin}/${targetLevel}`,
    method: 'PUT',
    data: payload,
  });
}
