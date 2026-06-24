export interface AdminBagSummary {
  record_id: number;
  player_id: number;
  player_name: string;
  container_type: string;
  slot_index: number;
  item_id: number;
  item_uid: string;
  item_name: string;
  item_type: string;
  quantity: number;
  is_bound: boolean;
  updated_at: string;
  created_at: string;
}

export interface AdminBagListResult {
  items: AdminBagSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminBagDetail {
  record_id: number;
  player_id: number;
  player_name: string;
  container_type: string;
  slot_index: number;
  item_id: number;
  item_uid: string;
  item_name: string;
  item_type: string;
  quantity: number;
  is_bound: boolean;
  expire_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AdminBagListFilters {
  record_id?: string;
  player_id?: string;
  container_type?: string;
  item_id?: string;
  item_uid?: string;
}

export interface AdminCreateBagPayload {
  player_id: number;
  container_type: string;
  item_id: number;
  quantity: number;
  is_bound: boolean;
}

export interface AdminUpdateBagPayload extends AdminCreateBagPayload {
  slot_index: number;
  item_uid: string;
}
