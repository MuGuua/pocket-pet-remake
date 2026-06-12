export interface AdminMonsterDefinitionSummary {
  monster_id: number;
  monster_name: string;
  level: number;
  quality: number;
  is_enabled: boolean;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminMonsterDefinitionListResult {
  items: AdminMonsterDefinitionSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminMonsterDefinitionBaseStats {
  level: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
}

export interface AdminMonsterDefinitionDetail {
  monster_id: number;
  monster_name: string;
  description: string;
  is_enabled: boolean;
  status_text: string;
  base_stats: AdminMonsterDefinitionBaseStats;
  skill_ids: number[];
  is_capturable: boolean;
  capture_pet_id: number;
  capture_rate_base: number;
  capture_min_hp_pct: number;
  capture_item_ids: number[];
  created_at: string;
  updated_at: string;
}

export interface AdminMonsterDefinitionListFilters {
  monster_id?: string;
  name?: string;
  enabled?: 'true' | 'false';
}

export interface AdminUpsertMonsterDefinitionPayload {
  monster_id: number;
  monster_name: string;
  description: string;
  is_enabled: boolean;
  level: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  skill_ids: number[];
  is_capturable: boolean;
  capture_pet_id: number;
  capture_rate_base: number;
  capture_min_hp_pct: number;
  capture_item_ids: number[];
}
