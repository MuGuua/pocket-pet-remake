import { requestJSON } from './http';
import type { AdminAdjustWalletPayload, AdminWalletDetail, AdminWalletListFilters, AdminWalletListResult } from '../types/wallet';

export async function fetchAdminWallets(params: { filters?: AdminWalletListFilters; page?: number; pageSize?: number }): Promise<AdminWalletListResult> {
  const query = new URLSearchParams();
  if (params.filters?.player_id?.trim()) query.set('player_id', params.filters.player_id.trim());
  if (params.filters?.keyword?.trim()) query.set('keyword', params.filters.keyword.trim());
  query.set('page', String(params.page ?? 1));
  query.set('page_size', String(params.pageSize ?? 20));
  return requestJSON<AdminWalletListResult>({ url: `/api/admin/wallets?${query.toString()}`, method: 'GET' });
}

export async function fetchAdminWalletDetail(playerID: number): Promise<AdminWalletDetail> {
  return requestJSON<AdminWalletDetail>({ url: `/api/admin/wallets/${playerID}`, method: 'GET' });
}

export async function adjustAdminWallet(playerID: number, payload: AdminAdjustWalletPayload): Promise<AdminWalletDetail> {
  return requestJSON<AdminWalletDetail>({ url: `/api/admin/wallets/${playerID}`, method: 'PUT', data: payload });
}
