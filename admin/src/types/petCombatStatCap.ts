export interface AdminPetCombatStatCap {
  stat_key: string;
  cap_value: number;
  description: string;
  status: number;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminUpsertPetCombatStatCapPayload {
  cap_value: number;
  description: string;
  status: number;
}

export const PET_COMBAT_STAT_CAP_LABELS: Record<string, string> = {
  hp_max: '生命上限',
  spirit: '精力',
  spirit_max: '精力上限',
  atk: '攻击',
  def: '防御',
  spd: '速度',
  mana: '法力',
  hit_pct: '命中',
  dodge_pct: '闪避',
  crit_rate_pct: '致命',
  crit_dmg_pct: '爆伤',
  physical_resist_pct: '物抗',
  reverse_physical_resist_pct: '逆物抗',
  skill_resist_pct: '技抗',
  reverse_skill_resist_pct: '逆技抗',
  confusion_resist_pct: '混乱抗性',
  sleep_resist_pct: '昏睡抗性',
  paralysis_resist_pct: '麻痹抗性',
  seal_resist_pct: '封印抗性',
  curse_resist_pct: '诅咒抗性',
  crit_dmg_resist_pct: '抗爆伤',
  crit_resist_pct: '抗致命',
  character_resist_pct: '抗人物',
  pet_resist_pct: '抗宠物',
};

export function formatPetCombatStatCapLabel(statKey: string): string {
  return PET_COMBAT_STAT_CAP_LABELS[statKey] ?? statKey;
}
