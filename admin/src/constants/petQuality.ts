/** 宠物品质档位：与 pet_definition.quality / player_pet.quality 数值一一对应。 */
export const PET_QUALITY_OPTIONS = [
  { value: 1, label: '普通宠', color: 'default' },
  { value: 2, label: '神宠', color: 'gold' },
  { value: 3, label: '魂宠', color: 'purple' },
  { value: 4, label: '圣宠', color: 'blue' },
  { value: 5, label: 'S宠', color: 'cyan' },
  { value: 6, label: 'SS宠', color: 'magenta' },
  { value: 7, label: '珍奇宠', color: 'orange' },
  { value: 8, label: '绝世宠', color: 'red' },
] as const;

export type PetQualityValue = (typeof PET_QUALITY_OPTIONS)[number]['value'];

/** 把数据库 quality 转成运营可读文案。 */
export function formatPetQualityLabel(quality: number): string {
  return PET_QUALITY_OPTIONS.find((item) => item.value === quality)?.label ?? `未知(${quality})`;
}

/** 返回品质 Tag 颜色；未知档位回退 default。 */
export function getPetQualityTagColor(quality: number): string {
  return PET_QUALITY_OPTIONS.find((item) => item.value === quality)?.color ?? 'default';
}

/** 判断模板是否为野外捕捉类（服务端 acquire_method 字段，后台不再展示）。 */
export function isWildCapturePetTemplate(acquireMethod: string | undefined): boolean {
  const value = String(acquireMethod ?? '').trim();
  return value === 'wild_capture' || value.includes('野外捕捉');
}
