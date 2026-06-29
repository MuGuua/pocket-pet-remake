/** 后台大数值展示：千分位分隔，便于运营辨认资质与战斗属性。 */
export function formatAdminInteger(value: number | string | undefined | null): string {
  if (value === undefined || value === null || value === '') {
    return '';
  }
  const normalized = String(value).replace(/,/g, '');
  if (!normalized) {
    return '';
  }
  return normalized.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

/** 解析千分位整数输入为 number。 */
export function parseAdminInteger(value: string | undefined): number {
  const parsed = Number(String(value ?? '').replace(/,/g, ''));
  return Number.isFinite(parsed) ? parsed : 0;
}

/** Ant Design InputNumber 通用整数格式化 props。 */
export const ADMIN_INTEGER_INPUT_PROPS = {
  formatter: (value: number | string | undefined) => formatAdminInteger(value),
  parser: (value: string | undefined) => parseAdminInteger(value),
};
