export interface AdminDashboardOverview {
  stat_date: string;
  timezone: string;
  generated_at: string;
  online_player_count: number;
  daily_active_accounts: number;
  new_accounts_today: number;
  total_accounts: number;
  total_players: number;
}

// 进程累计计数与最近最多 256 次样本分位；us 表示微秒，不能直接视为公网操作延迟。
export interface RuntimeMetric {
  key: string;
  unit: string;
  count: number;
  failures: number;
  recent_samples: number;
  last: number;
  p50: number;
  p95: number;
  max: number;
}
