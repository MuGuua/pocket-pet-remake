import { requestJSON } from './http';
import type {
  AdminSkillDetail,
  AdminSkillListFilters,
  AdminSkillListResult,
  AdminUpsertSkillPayload,
} from '../types/skillDefinition';

export async function fetchAdminSkillDefinitions(params: {
  filters?: AdminSkillListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminSkillListResult> {
  const query = new URLSearchParams();
  if (params.filters?.skill_id?.trim()) query.set('skill_id', params.filters.skill_id.trim());
  if (params.filters?.name?.trim()) query.set('name', params.filters.name.trim());
  if (params.filters?.category?.trim()) query.set('category', params.filters.category.trim());
  if (params.filters?.skill_type?.trim()) query.set('skill_type', params.filters.skill_type.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminSkillListResult>({ url: `/api/admin/skill-definitions?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminSkillDefinitionDetail(skillID: number): Promise<AdminSkillDetail> {
  return requestJSON<AdminSkillDetail>({ url: `/api/admin/skill-definitions/${skillID}`, method: 'GET' });
}

export async function createAdminSkillDefinition(payload: AdminUpsertSkillPayload): Promise<AdminSkillDetail> {
  return requestJSON<AdminSkillDetail>({ url: '/api/admin/skill-definitions', method: 'POST', data: payload });
}

export async function updateAdminSkillDefinition(skillID: number, payload: AdminUpsertSkillPayload): Promise<AdminSkillDetail> {
  return requestJSON<AdminSkillDetail>({ url: `/api/admin/skill-definitions/${skillID}`, method: 'PUT', data: payload });
}

export async function deleteAdminSkillDefinition(skillID: number): Promise<{ skill_id: number; deleted: boolean }> {
  return requestJSON<{ skill_id: number; deleted: boolean }>({ url: `/api/admin/skill-definitions/${skillID}`, method: 'DELETE' });
}
