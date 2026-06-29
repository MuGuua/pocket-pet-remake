import { requestJSON } from './http';
import type {
  AdminPetDefinitionDetail,
  AdminPetDefinitionListFilters,
  AdminPetDefinitionListResult,
  AdminUpsertPetDefinitionPayload,
} from '../types/petDefinition';

export async function fetchAdminPetDefinitions(params: {
  filters?: AdminPetDefinitionListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminPetDefinitionListResult> {
  const query = new URLSearchParams();
  if (params.filters?.pet_id?.trim()) query.set('pet_id', params.filters.pet_id.trim());
  if (params.filters?.name?.trim()) query.set('name', params.filters.name.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminPetDefinitionListResult>({ url: `/api/admin/pet-definitions?${query.toString()}`, method: 'GET' });
}

/** 分页拉取全部已启用系统宠物模板，供发放宠物下拉使用。 */
export async function fetchAllEnabledAdminPetDefinitions(): Promise<AdminPetDefinitionListResult['items']> {
  const pageSize = 100;
  const items: AdminPetDefinitionListResult['items'] = [];
  let page = 1;
  let total = 0;
  do {
    const result = await fetchAdminPetDefinitions({
      filters: { enabled: 'true' },
      page,
      pageSize,
    });
    items.push(...result.items);
    total = result.total;
    page += 1;
  } while (items.length < total);
  return items;
}

export async function fetchAdminPetDefinitionDetail(petID: number): Promise<AdminPetDefinitionDetail> {
  return requestJSON<AdminPetDefinitionDetail>({ url: `/api/admin/pet-definitions/${petID}`, method: 'GET' });
}

export async function createAdminPetDefinition(payload: AdminUpsertPetDefinitionPayload): Promise<AdminPetDefinitionDetail> {
  return requestJSON<AdminPetDefinitionDetail>({ url: '/api/admin/pet-definitions', method: 'POST', data: payload });
}

export async function updateAdminPetDefinition(petID: number, payload: AdminUpsertPetDefinitionPayload): Promise<AdminPetDefinitionDetail> {
  return requestJSON<AdminPetDefinitionDetail>({ url: `/api/admin/pet-definitions/${petID}`, method: 'PUT', data: payload });
}

export async function deleteAdminPetDefinition(petID: number): Promise<{ pet_id: number; deleted: boolean }> {
  return requestJSON<{ pet_id: number; deleted: boolean }>({ url: `/api/admin/pet-definitions/${petID}`, method: 'DELETE' });
}
