import { defaultAdminPetCombatStats, type AdminPetCombatStats } from './petCombatStats';

export const EQUIPMENT_SLOT_OPTIONS = [
  { value: 'weapon', label: '武器' },
  { value: 'class_weapon', label: '职业武器' },
  { value: 'hat', label: '帽子' },
  { value: 'clothes', label: '衣服' },
  { value: 'pants', label: '裤子' },
  { value: 'shoes', label: '鞋子' },
  { value: 'necklace', label: '项链' },
  { value: 'ring', label: '戒指' },
  { value: 'hero_ring', label: '英雄之戒' },
  { value: 'badge', label: '徽章' },
  { value: 'medicine_pouch', label: '药囊' },
  { value: 'charm', label: '护符' },
  { value: 'class_badge', label: '职业徽章' },
  { value: 'costume', label: '时装' },
  { value: 'element_bracelet', label: '元素手镯' },
  { value: 'rebirth_stone', label: '转生之石' },
  { value: 'guardian_ring', label: '守护之戒' },
] as const;

export interface AdminEquipmentCombatStats {
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
}

export const ADMIN_EQUIPMENT_COMBAT_STAT_FIELDS: Array<{ key: keyof AdminEquipmentCombatStats; label: string }> = [
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
];

export interface AdminMedicinePouchExtra {
  restore_player_hp: boolean;
  restore_player_spirit: boolean;
  restore_player_vigor: boolean;
  restore_pet_hp: boolean;
  restore_pet_spirit: boolean;
  restore_lineup_pets: boolean;
}

export interface AdminEquipmentSummary {
  item_id: number;
  item_code: string;
  item_name: string;
  equip_slot: string;
  equip_slot_label: string;
  required_level: number;
  quality: number;
  can_enhance: boolean;
  max_enhance_level: number;
  set_id: number;
  is_enabled: boolean;
  updated_at: string;
  created_at: string;
}

export interface AdminEquipmentListResult {
  items: AdminEquipmentSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminEquipmentListFilters {
  item_id?: string;
  equip_slot?: string;
  set_id?: string;
  keyword?: string;
  is_enabled?: string;
}

export interface AdminEquipmentDetail {
  item_id: number;
  item_code: string;
  item_name: string;
  desc: string;
  icon: string;
  quality: number;
  rarity: number;
  required_level: number;
  bind_type: string;
  can_sell: boolean;
  can_store: boolean;
  is_enabled: boolean;
  equip_slot: string;
  equip_slot_label: string;
  career_limit: string;
  can_enhance: boolean;
  max_enhance_level: number;
  set_id: number;
  appearance_skin_id: string;
  appearance_only: boolean;
  base_hp: number;
  base_mana: number;
  base_atk: number;
  base_def: number;
  base_spd: number;
  combat_stats: AdminEquipmentCombatStats;
  enhance_per_level_stats: Record<string, number>;
  socket_count: number;
  allowed_gem_types: string[];
  medicine_pouch?: AdminMedicinePouchExtra;
  created_at: string;
  updated_at: string;
}

export interface AdminUpsertEquipmentPayload {
  item_id: number;
  item_code: string;
  item_name: string;
  desc: string;
  icon: string;
  quality: number;
  rarity: number;
  required_level: number;
  bind_type: string;
  can_sell: boolean;
  can_store: boolean;
  is_enabled: boolean;
  equip_slot: string;
  career_limit: string;
  can_enhance: boolean;
  max_enhance_level: number;
  set_id: number;
  appearance_skin_id: string;
  appearance_only: boolean;
  base_hp: number;
  base_mana: number;
  base_atk: number;
  base_def: number;
  base_spd: number;
  combat_stats: AdminEquipmentCombatStats;
  enhance_per_level_stats: Record<string, number>;
  socket_count: number;
  allowed_gem_types: string[];
  medicine_pouch?: AdminMedicinePouchExtra;
}

export function formatEquipmentSlotLabel(slot: string): string {
  return EQUIPMENT_SLOT_OPTIONS.find((item) => item.value === slot)?.label ?? slot;
}

export function defaultMedicinePouchExtra(): AdminMedicinePouchExtra {
  return {
    restore_player_hp: true,
    restore_player_spirit: true,
    restore_player_vigor: true,
    restore_pet_hp: true,
    restore_pet_spirit: true,
    restore_lineup_pets: false,
  };
}

export function defaultAdminEquipmentCombatStats(): AdminEquipmentCombatStats {
  const petStats: AdminPetCombatStats = defaultAdminPetCombatStats();
  return {
    spirit: petStats.spirit,
    spirit_max: petStats.spirit_max,
    hit_pct: petStats.hit_pct,
    dodge_pct: petStats.dodge_pct,
    crit_rate_pct: petStats.crit_rate_pct,
    crit_dmg_pct: petStats.crit_dmg_pct,
    physical_resist_pct: petStats.physical_resist_pct,
    reverse_physical_resist_pct: petStats.reverse_physical_resist_pct,
    skill_resist_pct: petStats.skill_resist_pct,
    reverse_skill_resist_pct: petStats.reverse_skill_resist_pct,
    confusion_resist_pct: petStats.confusion_resist_pct,
    sleep_resist_pct: petStats.sleep_resist_pct,
    paralysis_resist_pct: petStats.paralysis_resist_pct,
    seal_resist_pct: petStats.seal_resist_pct,
    curse_resist_pct: petStats.curse_resist_pct,
    crit_dmg_resist_pct: petStats.crit_dmg_resist_pct,
    crit_resist_pct: petStats.crit_resist_pct,
    character_resist_pct: petStats.character_resist_pct,
    pet_resist_pct: petStats.pet_resist_pct,
  };
}

// defaultEquipmentValues 为“新增装备模板”提供表单默认值。
// item_id 由页面在打开新建弹窗时按当前最大 ID 自动注入，这里只保留其他字段默认值。
export function defaultEquipmentValues(itemID: number): AdminUpsertEquipmentPayload {
  return {
    item_id: itemID,
    item_code: `equipment_${itemID}`,
    item_name: '新手长剑',
    desc: '',
    icon: '',
    quality: 1,
    rarity: 1,
    required_level: 1,
    bind_type: 'none',
    can_sell: true,
    can_store: true,
    is_enabled: true,
    equip_slot: 'weapon',
    career_limit: '',
    can_enhance: true,
    max_enhance_level: 15,
    set_id: 0,
    appearance_skin_id: '',
    appearance_only: false,
    base_hp: 0,
    base_mana: 0,
    base_atk: 100,
    base_def: 0,
    base_spd: 0,
    combat_stats: defaultAdminEquipmentCombatStats(),
    enhance_per_level_stats: { atk: 100 },
    socket_count: 0,
    allowed_gem_types: [],
  };
}
