import { requestJSON } from './http';
import type {
  AdminSkillDetail,
  AdminSkillListFilters,
  AdminSkillListResult,
  AdminSkillSummary,
  AdminUpsertSkillPayload,
} from '../types/skillDefinition';

const ADMIN_SKILL_LIST_PAGE_SIZE = 100;

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
  if (params.filters?.activation_mode?.trim()) query.set('activation_mode', params.filters.activation_mode.trim());
  if (params.filters?.skill_quality?.trim()) query.set('skill_quality', params.filters.skill_quality.trim());
  if (params.filters?.enabled) query.set('enabled', params.filters.enabled);
  if (params.filters?.order_by?.trim()) query.set('order_by', params.filters.order_by.trim());
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminSkillListResult>({ url: `/api/admin/skill-definitions?${query.toString()}`, method: 'GET' });
}

/** 分页拉取全部技能模板，避免列表接口单页上限导致名称映射缺失。 */
export async function fetchAllAdminSkillDefinitions(filters?: AdminSkillListFilters): Promise<AdminSkillSummary[]> {
  const items: AdminSkillSummary[] = [];
  let page = 1;
  let total = 0;
  do {
    const result = await fetchAdminSkillDefinitions({
      filters,
      page,
      pageSize: ADMIN_SKILL_LIST_PAGE_SIZE,
    });
    items.push(...result.items);
    total = result.total;
    page += 1;
  } while (items.length < total);
  return items;
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
