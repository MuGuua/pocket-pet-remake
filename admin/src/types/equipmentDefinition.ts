import { defaultAdminPetCombatStats, type AdminPetCombatStats } from './petCombatStats';

export const EQUIPMENT_SLOT_OPTIONS = [
  { value: 'weapon', label: '武器' },
  { value: 'hat', label: '帽子' },
  { value: 'clothes', label: '衣服' },
  { value: 'pants', label: '裤子' },
  { value: 'shoes', label: '鞋子' },
  { value: 'necklace', label: '项链' },
  { value: 'ring', label: '戒指' },
  { value: 'hero_ring', label: '英雄之戒' },
  { value: 'medicine_pouch', label: '药囊' },
  { value: 'class_badge', label: '职业徽章' },
  { value: 'class_weapon', label: '职业武器' },
  { value: 'costume', label: '时装' },
  { value: 'element_bracelet', label: '元素手镯' },
] as const;

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
  combat_stats: AdminPetCombatStats;
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
  combat_stats: AdminPetCombatStats;
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

export function defaultEquipmentValues(): AdminUpsertEquipmentPayload {
  return {
    item_id: 4001,
    item_code: 'starter_sword',
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
    combat_stats: defaultAdminPetCombatStats(),
    enhance_per_level_stats: { atk: 100 },
    socket_count: 0,
    allowed_gem_types: [],
  };
}
