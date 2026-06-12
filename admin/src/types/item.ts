export interface AdminItemSummary {
  item_id: number;
  item_code: string;
  item_name: string;
  item_type: string;
  item_sub_type: string;
  quality: number;
  max_stack: number;
  buy_price_copper: number;
  sell_price_copper: number;
  usable: boolean;
  can_sell: boolean;
  can_store: boolean;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminItemListResult {
  items: AdminItemSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminItemDetail {
  item_id: number;
  item_code: string;
  item_name: string;
  item_type: string;
  item_sub_type: string;
  quality: number;
  rarity: number;
  icon: string;
  desc: string;
  max_stack: number;
  occupy_slots: number;
  auto_merge: boolean;
  sort_weight: number;
  usable: boolean;
  use_scope: string;
  target_type: string;
  required_level: number;
  required_scene_id: number;
  bind_type: string;
  can_sell: boolean;
  can_drop: boolean;
  can_store: boolean;
  can_trade: boolean;
  expire_at_rule: string;
  effect_type: string;
  effect_value: number;
  effect_params_json: string;
  buy_price_copper: number;
  sell_price_copper: number;
  recycle_price_copper: number;
  price_type: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminItemListFilters {
  item_id?: string;
  item_type?: string;
  keyword?: string;
  enabled?: 'true' | 'false';
}

export type AdminUpsertItemPayload = Omit<AdminItemDetail, 'created_at' | 'updated_at'>;
