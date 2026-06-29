/** 强化材料成功率计算模式 */
export type EnhanceMaterialSuccessMode = 'base' | 'bonus' | 'override';

/** 强化失败惩罚类型 */
export type EnhanceMaterialFailurePenalty = 'damage' | 'none' | 'level_down';

/** 后台物品编辑弹窗中的锻造/强化材料专属配置 */
export interface EnhanceMaterialConfig {
  success_rate_mode: EnhanceMaterialSuccessMode;
  success_rate_bonus_pct: number;
  success_rate_override_pct: number;
  guaranteed_success: boolean;
  failure_penalty: EnhanceMaterialFailurePenalty;
  failure_level_delta: number;
  description: string;
}

export const ENHANCE_MATERIAL_SUCCESS_MODE_OPTIONS: Array<{ value: EnhanceMaterialSuccessMode; label: string }> = [
  { value: 'base', label: '沿用全局成功率' },
  { value: 'bonus', label: '全局成功率 + 加成' },
  { value: 'override', label: '固定成功率' },
];

export const ENHANCE_MATERIAL_FAILURE_PENALTY_OPTIONS: Array<{ value: EnhanceMaterialFailurePenalty; label: string }> = [
  { value: 'damage', label: '失败损坏装备' },
  { value: 'level_down', label: '失败降强化等级（不损坏）' },
  { value: 'none', label: '失败无惩罚' },
];

export function defaultEnhanceMaterialConfig(): EnhanceMaterialConfig {
  return {
    success_rate_mode: 'base',
    success_rate_bonus_pct: 0,
    success_rate_override_pct: 0,
    guaranteed_success: false,
    failure_penalty: 'damage',
    failure_level_delta: 1,
    description: '',
  };
}

export function normalizeEnhanceMaterialConfig(
  value: Partial<EnhanceMaterialConfig> | null | undefined,
): EnhanceMaterialConfig {
  const defaults = defaultEnhanceMaterialConfig();
  const nextValue: EnhanceMaterialConfig = {
    success_rate_mode: value?.success_rate_mode ?? defaults.success_rate_mode,
    success_rate_bonus_pct: Math.max(0, Math.min(100, Math.trunc(Number(value?.success_rate_bonus_pct ?? 0)))),
    success_rate_override_pct: Math.max(0, Math.min(100, Math.trunc(Number(value?.success_rate_override_pct ?? 0)))),
    guaranteed_success: Boolean(value?.guaranteed_success),
    failure_penalty: value?.failure_penalty ?? defaults.failure_penalty,
    failure_level_delta: Math.max(0, Math.min(15, Math.trunc(Number(value?.failure_level_delta ?? 1)))),
    description: String(value?.description ?? '').trim(),
  };
  if (nextValue.guaranteed_success) {
    nextValue.success_rate_mode = 'override';
    nextValue.success_rate_override_pct = 100;
  }
  if (nextValue.failure_penalty !== 'level_down') {
    nextValue.failure_level_delta = 0;
  } else if (nextValue.failure_level_delta <= 0) {
    nextValue.failure_level_delta = 1;
  }
  return nextValue;
}

export function formatEnhanceMaterialConfigSummary(config: EnhanceMaterialConfig): string {
  const normalized = normalizeEnhanceMaterialConfig(config);
  const successParts: string[] = [];
  if (normalized.guaranteed_success) {
    successParts.push('100% 必成');
  } else if (normalized.success_rate_mode === 'bonus') {
    successParts.push(`全局 +${normalized.success_rate_bonus_pct}%`);
  } else if (normalized.success_rate_mode === 'override') {
    successParts.push(`固定 ${normalized.success_rate_override_pct}%`);
  } else {
    successParts.push('沿用全局成功率');
  }
  const penaltyLabel = ENHANCE_MATERIAL_FAILURE_PENALTY_OPTIONS.find(
    (option) => option.value === normalized.failure_penalty,
  )?.label ?? normalized.failure_penalty;
  return `${successParts.join('')}；${penaltyLabel}`;
}
