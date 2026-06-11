export interface AdminPlayerSummary {
  player_id: number;
  account_name: string;
  name: string;
  level: number;
  gold: number;
  status: number;
  status_text: string;
  scene_id: number;
  hp: number;
  hp_max: number;
  energy: number;
  energy_max: number;
  last_login_at: string | null;
  updated_at: string;
  created_at: string;
}

export interface AdminPlayerListResult {
  items: AdminPlayerSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminPlayerDetail {
  player_id: number;
  account_id: number;
  account_name: string;
  name: string;
  level: number;
  exp: number;
  gold: number;
  status: number;
  status_text: string;
  scene_id: number;
  pos_x: number;
  pos_y: number;
  hp: number;
  hp_max: number;
  energy: number;
  energy_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  hit_pct: number;
  dodge_pct: number;
  crit_rate_pct: number;
  crit_dmg_pct: number;
  physical_resist_pct: number;
  skill_resist_pct: number;
  confusion_resist_pct: number;
  sleep_resist_pct: number;
  paralysis_resist_pct: number;
  seal_resist_pct: number;
  curse_resist_pct: number;
  crit_resist_pct: number;
  crit_dmg_resist_pct: number;
  character_resist_pct: number;
  pet_resist_pct: number;
  mercenary_resist_pct: number;
  generic_shield_pct: number;
  skill_ids: number[];
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface AdminPlayerListFilters {
  player_id?: string;
  name?: string;
  status?: string;
}

export interface AdminCreatePlayerPayload {
  account_name: string;
  password: string;
  name: string;
  level: number;
  gold: number;
  scene_id: number;
  pos_x: number;
  pos_y: number;
  hp: number;
  hp_max: number;
  energy: number;
  energy_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  status: number;
  skill_ids: number[];
}

export interface AdminUpdatePlayerPayload {
  name: string;
  level: number;
  exp: number;
  gold: number;
  scene_id: number;
  pos_x: number;
  pos_y: number;
  hp: number;
  hp_max: number;
  energy: number;
  energy_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  status: number;
  skill_ids: number[];
}
