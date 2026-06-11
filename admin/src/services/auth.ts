import { clearAdminToken, requestJSON, setAdminToken } from './http';
import type { AdminLoginResult, AdminSessionProfile } from '../types/admin';

export async function loginAdmin(account: string, password: string): Promise<AdminLoginResult> {
  const result = await requestJSON<AdminLoginResult>({
    url: '/api/admin/auth/login',
    method: 'POST',
    data: { account, password },
  });
  setAdminToken(result.access_token);
  return result;
}

export async function fetchAdminProfile(): Promise<AdminSessionProfile> {
  return requestJSON<AdminSessionProfile>({
    url: '/api/admin/me',
    method: 'GET',
  });
}

export function logoutAdmin(): void {
  clearAdminToken();
}
