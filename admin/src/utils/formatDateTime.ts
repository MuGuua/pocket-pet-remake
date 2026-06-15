/**
 * 将 ISO 时间字符串格式化为运营可读的中文本地时间。
 */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return '-';
  }
  const date: Date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', { hour12: false });
}
