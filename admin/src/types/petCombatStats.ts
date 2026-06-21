export interface AdminPetCombatStats {
  spirit: number;
  spirit_max: number;
  hit_pct: number;
  dodge_pct: number;
  crit_rate_pct: number;
  crit_dmg_pct: number;
  physical_resist_pct: number;
  reverse_physical_resist_pct: number;
  skill_resist_pct: number;
  reverse_skill_resist_pct: number;
  confusion_resist_pct: number;
  sleep_resist_pct: number;
  paralysis_resist_pct: number;
  seal_resist_pct: number;
  curse_resist_pct: number;
  crit_dmg_resist_pct: number;
  crit_resist_pct: number;
  character_resist_pct: number;
  pet_resist_pct: number;
  guard: number;
  talent_dmg_pct: number;
  talent_reduce_pct: number;
  element_adv_pct: number;
  element_penalty_pct: number;
}

export const ADMIN_PET_COMBAT_STAT_FIELDS: Array<{ key: keyof AdminPetCombatStats; label: string }> = [
  { key: 'spirit', label: '精力' },
  { key: 'spirit_max', label: '精力上限' },
  { key: 'hit_pct', label: '命中' },
  { key: 'dodge_pct', label: '闪避' },
  { key: 'crit_rate_pct', label: '致命' },
  { key: 'crit_dmg_pct', label: '爆伤' },
  { key: 'physical_resist_pct', label: '物抗' },
  { key: 'reverse_physical_resist_pct', label: '逆物抗' },
  { key: 'skill_resist_pct', label: '技抗' },
  { key: 'reverse_skill_resist_pct', label: '逆技抗' },
  { key: 'confusion_resist_pct', label: '混乱抗性' },
  { key: 'sleep_resist_pct', label: '昏睡抗性' },
  { key: 'paralysis_resist_pct', label: '麻痹抗性' },
  { key: 'seal_resist_pct', label: '封印抗性' },
  { key: 'curse_resist_pct', label: '诅咒抗性' },
  { key: 'crit_dmg_resist_pct', label: '抗爆伤' },
  { key: 'crit_resist_pct', label: '抗致命' },
  { key: 'character_resist_pct', label: '抗人物' },
  { key: 'pet_resist_pct', label: '抗宠物' },
  { key: 'guard', label: '守护' },
  { key: 'talent_dmg_pct', label: '天赋增伤%' },
  { key: 'talent_reduce_pct', label: '天赋减伤%' },
  { key: 'element_adv_pct', label: '元素克制%' },
  { key: 'element_penalty_pct', label: '元素被克%' },
];

export function defaultAdminPetCombatStats(): AdminPetCombatStats {
  return {
    spirit: 0,
    spirit_max: 0,
    hit_pct: 0,
    dodge_pct: 0,
    crit_rate_pct: 0,
    crit_dmg_pct: 0,
    physical_resist_pct: 0,
    reverse_physical_resist_pct: 0,
    skill_resist_pct: 0,
    reverse_skill_resist_pct: 0,
    confusion_resist_pct: 0,
    sleep_resist_pct: 0,
    paralysis_resist_pct: 0,
    seal_resist_pct: 0,
    curse_resist_pct: 0,
    crit_dmg_resist_pct: 0,
    crit_resist_pct: 0,
    character_resist_pct: 0,
    pet_resist_pct: 0,
    guard: 0,
    talent_dmg_pct: 0,
    talent_reduce_pct: 0,
    element_adv_pct: 0,
    element_penalty_pct: 0,
  };
}
