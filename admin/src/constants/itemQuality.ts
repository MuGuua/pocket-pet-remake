// ITEM_QUALITY_OPTIONS 统一维护物品/装备后台使用的品质档位。
// 当前仅供物品与装备页面使用，不影响宠物、怪物等其他模块。
export const ITEM_QUALITY_OPTIONS = [
  { value: 1, label: '一品' },
  { value: 2, label: '二品' },
  { value: 3, label: '三品' },
  { value: 4, label: '四品' },
  { value: 5, label: '五品' },
  { value: 6, label: '神品' },
  { value: 7, label: '魂品' },
  { value: 8, label: '圣品' },
  { value: 9, label: '绝世' },
];

// formatItemQualityLabel 把数据库里的数字品质转成运营后台展示文案。
export function formatItemQualityLabel(quality: number): string | number {
  return ITEM_QUALITY_OPTIONS.find((item) => item.value === quality)?.label ?? quality;
}
