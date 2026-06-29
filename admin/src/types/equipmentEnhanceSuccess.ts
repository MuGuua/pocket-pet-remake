export interface AdminEnhanceSuccessConfig {
  target_level: number;
  required_level_min: number;
  required_level_band_max: number;
  required_level_band_label: string;
  success_rate_pct: number;
  description: string;
  status: number;
  updated_at: string;
}

export interface AdminEnhanceSuccessConfigListResult {
  items: AdminEnhanceSuccessConfig[];
}

export interface AdminUpsertEnhanceSuccessConfigPayload {
  success_rate_pct: number;
  description: string;
  status: number;
}

export const ENHANCE_REQUIRED_LEVEL_BAND_OPTIONS: Array<{ value: number; label: string }> = [
  { value: 1, label: '1~10级' },
  { value: 11, label: '11~20级' },
  { value: 21, label: '21~30级' },
  { value: 31, label: '31~40级' },
  { value: 41, label: '41~50级' },
  { value: 51, label: '51~60级' },
  { value: 61, label: '61~70级' },
  { value: 71, label: '71~80级' },
  { value: 81, label: '81~90级' },
  { value: 91, label: '91~100级' },
  { value: 101, label: '101~110级' },
];

export function formatEnhanceRequiredLevelBandLabel(bandMin: number): string {
  const normalized = Math.max(1, Math.trunc(bandMin));
  return `${normalized}~${normalized + 9}级`;
}
