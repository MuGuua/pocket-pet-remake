export interface AdminPetDefinitionSummary {
  pet_id: number;
  pet_name: string;
  quality: number;
  level: number;
  acquire_method: string;
  is_enabled: boolean;
  status_text: string;
  skin_id: string;
  created_at: string;
  updated_at: string;
}

export interface AdminPetDefinitionListResult {
  items: AdminPetDefinitionSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminPetDefinitionBaseStats {
  level: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
}

export interface AdminPetDefinitionGrowthAptitudes {
  hp_apt: number;
  atk_apt: number;
  def_apt: number;
  spd_apt: number;
  mana_apt: number;
}

export interface AdminPetDefinitionAptitudeRollRanges {
  hp_apt_roll_min: number;
  hp_apt_roll_max: number;
  atk_apt_roll_min: number;
  atk_apt_roll_max: number;
  def_apt_roll_min: number;
  def_apt_roll_max: number;
  spd_apt_roll_min: number;
  spd_apt_roll_max: number;
  mana_apt_roll_min: number;
  mana_apt_roll_max: number;
}

export interface AdminPetDefinitionDetail {
  pet_id: number;
  pet_name: string;
  description: string;
  acquire_method: string;
  is_enabled: boolean;
  status_text: string;
  skin_id: string;
  base_stats: AdminPetDefinitionBaseStats;
  growth_aptitudes: AdminPetDefinitionGrowthAptitudes;
  aptitude_roll_ranges: AdminPetDefinitionAptitudeRollRanges;
  skill_ids: number[];
  created_at: string;
  updated_at: string;
}

export interface AdminPetDefinitionListFilters {
  pet_id?: string;
  name?: string;
  enabled?: 'true' | 'false';
}

export interface AdminUpsertPetDefinitionPayload {
  pet_id: number;
  pet_name: string;
  description: string;
  acquire_method: string;
  is_enabled: boolean;
  skin_id: string;
  level: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  hp_apt: number;
  atk_apt: number;
  def_apt: number;
  spd_apt: number;
  mana_apt: number;
  hp_apt_roll_min: number;
  hp_apt_roll_max: number;
  atk_apt_roll_min: number;
  atk_apt_roll_max: number;
  def_apt_roll_min: number;
  def_apt_roll_max: number;
  spd_apt_roll_min: number;
  spd_apt_roll_max: number;
  mana_apt_roll_min: number;
  mana_apt_roll_max: number;
  skill_ids: number[];
}
