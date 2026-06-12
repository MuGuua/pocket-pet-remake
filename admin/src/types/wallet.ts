export interface WalletSnapshot {
  total_copper: number;
  gold: number;
  silver: number;
  copper: number;
}

export interface AdminWalletSummary {
  player_id: number;
  player_name: string;
  wallet: WalletSnapshot;
  created_at: string;
  updated_at: string;
}

export interface AdminWalletListResult {
  items: AdminWalletSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminWalletDetail {
  player_id: number;
  player_name: string;
  wallet: WalletSnapshot;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface AdminWalletListFilters {
  player_id?: string;
  keyword?: string;
}

export interface AdminAdjustWalletPayload {
  change_total_copper: number;
  reason: string;
}
