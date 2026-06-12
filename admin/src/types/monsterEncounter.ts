export interface AdminMonsterEncounterSummary {
  entity_id: number;
  encounter_name: string;
  spawn_count: number;
  is_enabled: boolean;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminMonsterEncounterListResult {
  items: AdminMonsterEncounterSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminMonsterEncounterDetail {
  entity_id: number;
  encounter_name: string;
  description: string;
  spawn_monster_ids: number[];
  is_enabled: boolean;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminMonsterEncounterListFilters {
  entity_id?: string;
  name?: string;
  enabled?: 'true' | 'false';
}

export interface AdminUpsertMonsterEncounterPayload {
  entity_id: number;
  encounter_name: string;
  description: string;
  spawn_monster_ids: number[];
  is_enabled: boolean;
}
