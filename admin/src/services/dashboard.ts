import { requestJSON } from './http';
import type { AdminDashboardOverview } from '../types/dashboard';

export async function fetchAdminDashboardOverview(): Promise<AdminDashboardOverview> {
  return requestJSON<AdminDashboardOverview>({ url: '/api/admin/dashboard/overview', method: 'GET' });
}
