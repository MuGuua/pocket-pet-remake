import { requestJSON } from './http';
import type {
  AdminCreatePlayerQuestPayload,
  AdminCreateQuestTemplatePayload,
  AdminPlayerQuestDetail,
  AdminPlayerQuestFilters,
  AdminPlayerQuestListResult,
  AdminQuestTemplateDetail,
  AdminQuestTemplateFilters,
  AdminQuestTemplateListResult,
  AdminQuestTemplateSummary,
  AdminUpdatePlayerQuestPayload,
  AdminUpdateQuestTemplatePayload,
} from '../types/quest';

export async function fetchAdminQuestTemplates(params: {
  filters?: AdminQuestTemplateFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminQuestTemplateListResult> {
  const query = new URLSearchParams();
  if (params.filters?.quest_id?.trim()) query.set('quest_id', params.filters.quest_id.trim());
  if (params.filters?.quest_type?.trim()) query.set('quest_type', params.filters.quest_type.trim());
  if (params.filters?.title?.trim()) query.set('title', params.filters.title.trim());
  if (params.filters?.status?.trim()) query.set('status', params.filters.status.trim());
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminQuestTemplateListResult>({ url: `/api/admin/quests/templates?${query.toString()}`, method: 'GET' });
}

/** 分页读取全部任务模板，供前置任务等关联配置下拉框使用。 */
export async function fetchAllAdminQuestTemplates(): Promise<AdminQuestTemplateSummary[]> {
  const pageSize = 100;
  const firstPage = await fetchAdminQuestTemplates({ page: 1, pageSize });
  const templates: AdminQuestTemplateSummary[] = [...firstPage.items];
  const totalPages = Math.ceil(firstPage.total / firstPage.page_size);
  for (let page = 2; page <= totalPages; page += 1) {
    const result = await fetchAdminQuestTemplates({ page, pageSize });
    templates.push(...result.items);
  }
  return templates;
}

export async function fetchAdminQuestTemplateDetail(questID: number): Promise<AdminQuestTemplateDetail> {
  return requestJSON<AdminQuestTemplateDetail>({ url: `/api/admin/quests/templates/${questID}`, method: 'GET' });
}

export async function createAdminQuestTemplate(payload: AdminCreateQuestTemplatePayload): Promise<AdminQuestTemplateDetail> {
  return requestJSON<AdminQuestTemplateDetail>({ url: '/api/admin/quests/templates', method: 'POST', data: payload });
}

export async function updateAdminQuestTemplate(questID: number, payload: AdminUpdateQuestTemplatePayload): Promise<AdminQuestTemplateDetail> {
  return requestJSON<AdminQuestTemplateDetail>({ url: `/api/admin/quests/templates/${questID}`, method: 'PUT', data: payload });
}

export async function deleteAdminQuestTemplate(questID: number): Promise<{ quest_id: number; deleted: boolean }> {
  return requestJSON<{ quest_id: number; deleted: boolean }>({ url: `/api/admin/quests/templates/${questID}`, method: 'DELETE' });
}

export async function fetchAdminPlayerQuests(params: {
  filters?: AdminPlayerQuestFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminPlayerQuestListResult> {
  const query = new URLSearchParams();
  if (params.filters?.record_id?.trim()) query.set('record_id', params.filters.record_id.trim());
  if (params.filters?.player_id?.trim()) query.set('player_id', params.filters.player_id.trim());
  if (params.filters?.quest_id?.trim()) query.set('quest_id', params.filters.quest_id.trim());
  if (params.filters?.state?.trim()) query.set('state', params.filters.state.trim());
  if (params.filters?.tracked?.trim()) query.set('tracked', params.filters.tracked.trim());
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminPlayerQuestListResult>({ url: `/api/admin/quests/player-progress?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminPlayerQuestDetail(recordID: number): Promise<AdminPlayerQuestDetail> {
  return requestJSON<AdminPlayerQuestDetail>({ url: `/api/admin/quests/player-progress/${recordID}`, method: 'GET' });
}

export async function createAdminPlayerQuest(payload: AdminCreatePlayerQuestPayload): Promise<AdminPlayerQuestDetail> {
  return requestJSON<AdminPlayerQuestDetail>({ url: '/api/admin/quests/player-progress', method: 'POST', data: payload });
}

export async function updateAdminPlayerQuest(recordID: number, payload: AdminUpdatePlayerQuestPayload): Promise<AdminPlayerQuestDetail> {
  return requestJSON<AdminPlayerQuestDetail>({ url: `/api/admin/quests/player-progress/${recordID}`, method: 'PUT', data: payload });
}

export async function deleteAdminPlayerQuest(recordID: number): Promise<{ record_id: number; deleted: boolean }> {
  return requestJSON<{ record_id: number; deleted: boolean }>({ url: `/api/admin/quests/player-progress/${recordID}`, method: 'DELETE' });
}
