import type { AdminPetCombatStats } from './petCombatStats';

export interface AdminPetSummary {
  pet_uid: number;
  player_id: number;
  player_name: string;
  pet_id: number;
  pet_name: string;
  custom_name: string;
  level: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  in_lineup: boolean;
  updated_at: string;
  created_at: string;
}

export interface AdminPetListResult {
  items: AdminPetSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminPetDetail extends AdminPetCombatStats {
  pet_uid: number;
  player_id: number;
  player_name: string;
  pet_id: number;
  pet_name: string;
  custom_name: string;
  level: number;
  exp: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  skill_ids: number[];
  innate_skill_ids: number[];
  normal_skill_ids: number[];
  in_lineup: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminPetListFilters {
  pet_uid?: string;
  player_id?: string;
  pet_id?: string;
}

export interface AdminGrantPetFromTemplatePayload {
  player_id: number;
  pet_id: number;
}

export interface AdminCreatePetPayload extends AdminPetCombatStats {
  player_id: number;
  pet_id: number;
  level: number;
  exp: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  skill_ids: number[];
  innate_skill_ids: number[];
  normal_skill_ids: number[];
}

export interface AdminUpdatePetPayload extends AdminPetCombatStats {
  pet_id: number;
  custom_name: string;
  level: number;
  exp: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  skill_ids: number[];
  innate_skill_ids: number[];
  normal_skill_ids: number[];
}
