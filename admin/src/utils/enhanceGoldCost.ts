import type { AdminEquipmentEnhanceGoldCost, EnhanceGoldIncrementMode } from '../types/equipmentDefinition';

export interface EnhanceGoldCostPreviewRow {
  target_level: number;
  cost_copper: number;
}

/** 与服务端 CalculateEnhanceGoldCost 保持一致的本地预览计算。 */
export function calculateEnhanceGoldCost(
  targetLevel: number,
  config: Pick<
    AdminEquipmentEnhanceGoldCost,
    'is_enabled' | 'base_copper' | 'increment_mode' | 'increment_fixed' | 'increment_percent'
  >,
): number {
  if (!config.is_enabled || targetLevel <= 0) {
    return 0;
  }
  if (targetLevel === 1) {
    return Math.max(0, Math.trunc(config.base_copper));
  }
  if (config.increment_mode === 'fixed') {
    return Math.max(0, Math.trunc(config.base_copper + (targetLevel - 1) * config.increment_fixed));
  }
  if (config.increment_percent <= 0) {
    return Math.max(0, Math.trunc(config.base_copper));
  }
  let cost = Math.max(0, Math.trunc(config.base_copper));
  const multiplierNumerator = 100 + Math.trunc(config.increment_percent);
  for (let level = 2; level <= targetLevel; level += 1) {
    cost = Math.floor((cost * multiplierNumerator) / 100);
  }
  return cost;
}

export function buildEnhanceGoldCostPreview(
  config: Pick<
    AdminEquipmentEnhanceGoldCost,
    'is_enabled' | 'base_copper' | 'increment_mode' | 'increment_fixed' | 'increment_percent'
  >,
  maxLevel = 15,
): EnhanceGoldCostPreviewRow[] {
  const rows: EnhanceGoldCostPreviewRow[] = [];
  for (let level = 1; level <= maxLevel; level += 1) {
    rows.push({
      target_level: level,
      cost_copper: calculateEnhanceGoldCost(level, config),
    });
  }
  return rows;
}

export function describeEnhanceGoldCostFormula(mode: EnhanceGoldIncrementMode): string {
  if (mode === 'percent') {
    return '消耗 = 基础 × (1 + 每级百分比/100)^(目标等级-1)，逐级复合';
  }
  return '消耗 = 基础 + (目标等级 - 1) × 每级固定增加值';
}
