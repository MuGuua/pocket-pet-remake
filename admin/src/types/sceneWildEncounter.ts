import type { AdminMonsterBattleRewardEntry, AdminMonsterBattleRewardSaveEntry } from './monsterDefinition';

export interface AdminSceneWildEncounterSummary {
  scene_id: number;
  encounter_name: string;
  encounter_rate: number;
  spawn_count: number;
  formation_count: number;
  is_enabled: boolean;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminSceneWildEncounterListResult {
  items: AdminSceneWildEncounterSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminSceneWildEncounterDetail {
  scene_id: number;
  encounter_name: string;
  description: string;
  encounter_rate: number;
  spawn_monster_ids: number[];
  formations: AdminSceneWildEncounterFormation[];
  rewards: AdminMonsterBattleRewardEntry[];
  is_enabled: boolean;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminSceneWildEncounterFormation {
  formation_name: string;
  weight: number;
  spawn_monster_ids: number[];
  monster_slots?: AdminSceneWildEncounterMonsterSlot[];
}

export interface AdminSceneWildEncounterMonsterSlot {
  monster_id: number;
  reward_enabled: boolean;
}

export interface AdminSceneWildEncounterListFilters {
  scene_id?: string;
  name?: string;
  enabled?: 'true' | 'false';
}

export interface AdminUpsertSceneWildEncounterPayload {
  scene_id: number;
  encounter_name: string;
  description: string;
  encounter_rate: number;
  spawn_monster_ids: number[];
  formations: AdminSceneWildEncounterFormation[];
  rewards: AdminMonsterBattleRewardSaveEntry[];
  is_enabled: boolean;
}

export const SCENE_ID_OPTIONS: Array<{ label: string; value: number }> = [
  { label: '1 · 若克思家', value: 1 },
  { label: '2 · 闪光镇东路', value: 2 },
  { label: '3 · 市场', value: 3 },
  { label: '4 · 北路/北部野外', value: 4 },
  { label: '5 · 学校', value: 5 },
  { label: '6 · 打怪区', value: 6 },
];

export function formatEncounterRatePercent(encounterRate: number): string {
  const percentValue = Math.floor(encounterRate / 100);
  const fractionValue = encounterRate % 100;
  if (fractionValue === 0) {
    return `${percentValue}%`;
  }
  return `${percentValue}.${String(fractionValue).padStart(2, '0')}%`;
}
