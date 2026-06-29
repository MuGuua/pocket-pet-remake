/**
 * 与 Godot RichTextLabel BBCode 对齐的富文本工具。
 * 后台录入、预览与客户端展示共用同一套标签语法。
 */

/** 常用颜色快捷项，供编辑器工具栏使用。 */
export interface RichTextColorPreset {
  label: string;
  value: string;
}

/** 编辑器工具栏可插入的 BBCode 片段。 */
export interface RichTextToolbarAction {
  key: string;
  label: string;
  /** 无选区时在光标处插入的完整片段。 */
  insert: string;
  /** 有选区时包裹选中文本的 opening / closing 标签。 */
  wrap?: [string, string];
}

export const RICH_TEXT_COLOR_PRESETS: RichTextColorPreset[] = [
  { label: '绿', value: 'green' },
  { label: '蓝', value: 'blue' },
  { label: '红', value: 'red' },
  { label: '黄', value: '#e6b422' },
  { label: '灰', value: '#9494b8' },
];

export const RICH_TEXT_TOOLBAR_ACTIONS: RichTextToolbarAction[] = [
  { key: 'bold', label: '加粗', insert: '[b]加粗文字[/b]', wrap: ['[b]', '[/b]'] },
  { key: 'italic', label: '斜体', insert: '[i]斜体文字[/i]', wrap: ['[i]', '[/i]'] },
  { key: 'underline', label: '下划线', insert: '[u]下划线[/u]', wrap: ['[u]', '[/u]'] },
  { key: 'line', label: '换行', insert: '\n' },
];

/** 运营录入时可一键插入的示例模板。 */
export const RICH_TEXT_EXAMPLE_TEMPLATES: Array<{ label: string; value: string }> = [
  {
    label: '消耗品效果',
    value: '一瓶非常普通的药水，但却旅行冒险必需品\n使用后效果 :\n\n恢复HP [color=green]+300[/color]\n恢复MP [color=blue]+300[/color]',
  },
  {
    label: '技能说明',
    value: '对单个敌人造成物理伤害。\n\n伤害 [color=green]+120%[/color]\n消耗精力 [color=blue]15[/color]',
  },
];

const BBCode_COLOR_PATTERN = /\[color=([^\]]+)\]([\s\S]*?)\[\/color\]/gi;
const BBCode_BOLD_PATTERN = /\[b\]([\s\S]*?)\[\/b\]/gi;
const BBCode_ITALIC_PATTERN = /\[i\]([\s\S]*?)\[\/i\]/gi;
const BBCode_UNDERLINE_PATTERN = /\[u\]([\s\S]*?)\[\/u\]/gi;
const BBCode_TAG_HINT_PATTERN = /\[(?:\/)?(?:b|i|u|color(?:=[^\]]+)?)\]/i;

/** 判断文本是否包含 BBCode 标签。 */
export function containsBbcode(text: string): boolean {
  return BBCode_TAG_HINT_PATTERN.test(text);
}

/** 转义 HTML 特殊字符，避免预览区 XSS。 */
export function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

/** 将 Godot BBCode 转为可在后台预览的 HTML 片段（不含外层容器）。 */
export function bbcodeToHtml(text: string): string {
  if (!text.trim()) {
    return '';
  }
  let html = escapeHtml(text);
  html = html.replace(BBCode_BOLD_PATTERN, '<strong>$1</strong>');
  html = html.replace(BBCode_ITALIC_PATTERN, '<em>$1</em>');
  html = html.replace(BBCode_UNDERLINE_PATTERN, '<u>$1</u>');
  html = html.replace(BBCode_COLOR_PATTERN, '<span style="color: $1">$2</span>');
  html = html.replace(/\n/g, '<br />');
  return html;
}

/** 在 textarea 光标处插入或包裹 BBCode 片段。 */
export function applyTextareaBbcodeAction(
  textarea: HTMLTextAreaElement,
  action: RichTextToolbarAction,
  colorValue?: string,
): { nextValue: string; selectionStart: number; selectionEnd: number } {
  const currentValue = textarea.value;
  const selectionStart = textarea.selectionStart ?? currentValue.length;
  const selectionEnd = textarea.selectionEnd ?? currentValue.length;
  const selectedText = currentValue.slice(selectionStart, selectionEnd);

  if (colorValue) {
    const wrapped = selectedText
      ? `[color=${colorValue}]${selectedText}[/color]`
      : `[color=${colorValue}]文字[/color]`;
    const nextValue = currentValue.slice(0, selectionStart) + wrapped + currentValue.slice(selectionEnd);
    const cursor = selectionStart + wrapped.length;
    return { nextValue, selectionStart: cursor, selectionEnd: cursor };
  }

  if (selectedText && action.wrap) {
    const wrapped = `${action.wrap[0]}${selectedText}${action.wrap[1]}`;
    const nextValue = currentValue.slice(0, selectionStart) + wrapped + currentValue.slice(selectionEnd);
    const cursor = selectionStart + wrapped.length;
    return { nextValue, selectionStart: cursor, selectionEnd: cursor };
  }

  const nextValue = currentValue.slice(0, selectionStart) + action.insert + currentValue.slice(selectionEnd);
  const cursor = selectionStart + action.insert.length;
  return { nextValue, selectionStart: cursor, selectionEnd: cursor };
}

/** 在 textarea 光标处插入任意文本片段（如 {item:1001}、{pet:2001}）。 */
export function insertTextAtCursor(
  textarea: HTMLTextAreaElement,
  insertText: string,
): { nextValue: string; selectionStart: number; selectionEnd: number } {
  const currentValue = textarea.value;
  const selectionStart = textarea.selectionStart ?? currentValue.length;
  const selectionEnd = textarea.selectionEnd ?? currentValue.length;
  const needsLeadingSpace = selectionStart > 0 && !/\s/.test(currentValue.charAt(selectionStart - 1));
  const needsTrailingSpace = selectionEnd < currentValue.length && !/\s/.test(currentValue.charAt(selectionEnd));
  const normalizedInsert = `${needsLeadingSpace ? ' ' : ''}${insertText}${needsTrailingSpace ? ' ' : ''}`;
  const nextValue = currentValue.slice(0, selectionStart) + normalizedInsert + currentValue.slice(selectionEnd);
  const cursor = selectionStart + normalizedInsert.length;
  return { nextValue, selectionStart: cursor, selectionEnd: cursor };
}
