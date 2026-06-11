export interface AdminBagSummary {
  record_id: number;
  player_id: number;
  player_name: string;
  item_id: number;
  count: number;
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
  item_id: number;
  count: number;
  created_at: string;
  updated_at: string;
}

export interface AdminBagListFilters {
  record_id?: string;
  player_id?: string;
  item_id?: string;
}

export interface AdminCreateBagPayload {
  player_id: number;
  item_id: number;
  count: number;
}

export interface AdminUpdateBagPayload {
  player_id: number;
  item_id: number;
  count: number;
}
