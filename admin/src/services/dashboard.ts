import { requestJSON } from './http';
import type { AdminDashboardOverview, RuntimeMetric } from '../types/dashboard';

export async function fetchAdminDashboardOverview(): Promise<AdminDashboardOverview> {
  return requestJSON<AdminDashboardOverview>({ url: '/api/admin/dashboard/overview', method: 'GET' });
}

// 复用统一鉴权请求层，只读取当前服务进程的观测结果。
export async function fetchRuntimeMetrics(): Promise<RuntimeMetric[]> {
  return requestJSON<RuntimeMetric[]>({ url: '/api/admin/dashboard/runtime-metrics', method: 'GET' });
}
