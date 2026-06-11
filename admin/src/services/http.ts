import axios from 'axios';
import type { AxiosRequestConfig } from 'axios';
import type { ApiEnvelope } from '../types/http';

const ADMIN_TOKEN_KEY = 'pp_admin_access_token';

export const httpClient = axios.create({
  baseURL: '/',
  timeout: 10000,
});

httpClient.interceptors.request.use((config) => {
  const token = window.localStorage.getItem(ADMIN_TOKEN_KEY);
  if (token) {
    config.headers = config.headers ?? {};
    if (typeof config.headers.set === 'function') {
      config.headers.set('Authorization', `Bearer ${token}`);
    } else {
      (config.headers as Record<string, string>).Authorization = `Bearer ${token}`;
    }
  }
  return config;
});

// 后台接口虽然走统一 envelope，但页面只想消费 data；统一拆包能减少每页样板代码。
export async function requestJSON<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await httpClient.request<ApiEnvelope<T>>(config);
  if (response.data.code !== 200) {
    throw new Error(response.data.msg || 'request failed');
  }
  return response.data.data;
}

export function getAdminToken(): string {
  return window.localStorage.getItem(ADMIN_TOKEN_KEY) ?? '';
}

export function setAdminToken(token: string): void {
  window.localStorage.setItem(ADMIN_TOKEN_KEY, token);
}

export function clearAdminToken(): void {
  window.localStorage.removeItem(ADMIN_TOKEN_KEY);
}
