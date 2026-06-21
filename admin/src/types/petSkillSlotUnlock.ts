export interface AdminPetSkillSlotUnlockItem {
  slot_key: string;
  item_id: number;
  description: string;
  status: number;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminUpsertPetSkillSlotUnlockPayload {
  slot_key: string;
  item_id: number;
  description: string;
  status: number;
}

export const PET_TALISMAN_SLOT_OPTIONS = [
  { value: 'active_talisman', label: '主动神符技' },
  { value: 'talisman_hero', label: '神符技·英雄' },
  { value: 'talisman_1', label: '神符技【1】' },
  { value: 'talisman_2', label: '神符技【2】' },
  { value: 'talisman_3', label: '神符技【3】' },
];
