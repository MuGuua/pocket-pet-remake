import { requestJSON } from './http';
import type {
  AdminPetSkillSlotUnlockItem,
  AdminUpsertPetSkillSlotUnlockPayload,
} from '../types/petSkillSlotUnlock';

export async function fetchAdminPetSkillSlotUnlockItems(): Promise<AdminPetSkillSlotUnlockItem[]> {
  return requestJSON<AdminPetSkillSlotUnlockItem[]>({
    url: '/api/admin/pet-skill-slot-unlock-items',
    method: 'GET',
  });
}

export async function createAdminPetSkillSlotUnlockItem(
  payload: AdminUpsertPetSkillSlotUnlockPayload,
): Promise<AdminPetSkillSlotUnlockItem> {
  return requestJSON<AdminPetSkillSlotUnlockItem>({
    url: '/api/admin/pet-skill-slot-unlock-items',
    method: 'POST',
    data: payload,
  });
}

export async function updateAdminPetSkillSlotUnlockItem(
  slotKey: string,
  payload: AdminUpsertPetSkillSlotUnlockPayload,
): Promise<AdminPetSkillSlotUnlockItem> {
  return requestJSON<AdminPetSkillSlotUnlockItem>({
    url: `/api/admin/pet-skill-slot-unlock-items/${encodeURIComponent(slotKey)}`,
    method: 'PUT',
    data: payload,
  });
}

export async function deleteAdminPetSkillSlotUnlockItem(slotKey: string): Promise<void> {
  await requestJSON<null>({
    url: `/api/admin/pet-skill-slot-unlock-items/${encodeURIComponent(slotKey)}`,
    method: 'DELETE',
  });
}
