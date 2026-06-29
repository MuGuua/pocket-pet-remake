import { requestJSON } from './http';
import type {
  AdminCreatePetPayload,
  AdminGrantPetFromTemplatePayload,
  AdminPetDetail,
  AdminPetListFilters,
  AdminPetListResult,
  AdminUpdatePetPayload,
} from '../types/pet';

export async function fetchAdminPets(params: {
  filters?: AdminPetListFilters;
  page?: number;
  pageSize?: number;
}): Promise<AdminPetListResult> {
  const query = new URLSearchParams();
  if (params.filters?.pet_uid?.trim()) {
    query.set('pet_uid', params.filters.pet_uid.trim());
  }
  if (params.filters?.player_id?.trim()) {
    query.set('player_id', params.filters.player_id.trim());
  }
  if (params.filters?.pet_id?.trim()) {
    query.set('pet_id', params.filters.pet_id.trim());
  }
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminPetListResult>({ url: `/api/admin/pets?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminPetDetail(petUID: number): Promise<AdminPetDetail> {
  return requestJSON<AdminPetDetail>({ url: `/api/admin/pets/${petUID}`, method: 'GET' });
}

export async function grantAdminPetFromTemplate(payload: AdminGrantPetFromTemplatePayload): Promise<AdminPetDetail> {
  return requestJSON<AdminPetDetail>({
    url: '/api/admin/pets/grant-from-template',
    method: 'POST',
    data: payload,
  });
}

export async function createAdminPet(payload: AdminCreatePetPayload): Promise<AdminPetDetail> {
  return requestJSON<AdminPetDetail>({ url: '/api/admin/pets', method: 'POST', data: payload });
}

export async function updateAdminPet(petUID: number, payload: AdminUpdatePetPayload): Promise<AdminPetDetail> {
  return requestJSON<AdminPetDetail>({ url: `/api/admin/pets/${petUID}`, method: 'PUT', data: payload });
}

export async function deleteAdminPet(petUID: number): Promise<{ pet_uid: number; deleted: boolean }> {
  return requestJSON<{ pet_uid: number; deleted: boolean }>({ url: `/api/admin/pets/${petUID}`, method: 'DELETE' });
}
