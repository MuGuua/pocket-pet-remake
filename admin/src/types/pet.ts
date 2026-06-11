export interface AdminPetSummary {
  pet_uid: number;
  player_id: number;
  player_name: string;
  pet_id: number;
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

export interface AdminPetDetail {
  pet_uid: number;
  player_id: number;
  player_name: string;
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
  in_lineup: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminPetListFilters {
  pet_uid?: string;
  player_id?: string;
  pet_id?: string;
}

export interface AdminCreatePetPayload {
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
}

export interface AdminUpdatePetPayload {
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
}
