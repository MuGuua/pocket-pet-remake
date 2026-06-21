import { requestJSON } from './http';
import type {
  AdminPetCombatStatCap,
  AdminUpsertPetCombatStatCapPayload,
} from '../types/petCombatStatCap';

export async function fetchAdminPetCombatStatCaps(): Promise<AdminPetCombatStatCap[]> {
  return requestJSON<AdminPetCombatStatCap[]>({
    url: '/api/admin/pet-combat-stat-caps',
    method: 'GET',
  });
}

export async function updateAdminPetCombatStatCap(
  statKey: string,
  payload: AdminUpsertPetCombatStatCapPayload,
): Promise<AdminPetCombatStatCap> {
  return requestJSON<AdminPetCombatStatCap>({
    url: `/api/admin/pet-combat-stat-caps/${encodeURIComponent(statKey)}`,
    method: 'PUT',
    data: payload,
  });
}
