/**
 * 与 Godot RichTextLabel BBCode 对齐的富文本工具。
 * 后台录入、预览与客户端展示共用同一套标签语法。
 */

/** 常用颜色快捷项，供编辑器工具栏使用。 */
export interface RichTextColorPreset {
  label: string;
  value: string;
}

export const RICH_TEXT_COLOR_PRESETS: RichTextColorPreset[] = [
  { label: '常用绿', value: '#2AFF2A' },
  { label: '常用金', value: '#FFFF00' },
  { label: '常用蓝', value: '#00FFFF' },
  { label: '常用橙', value: '#FF7D00' },
  { label: '常用粉', value: '#FF64FF' },
  { label: '常用红', value: '#FF0000' },
  { label: '常用白', value: '#FFFFFF' },
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

interface BbcodeStyleToken {
  name: 'b' | 'i' | 'u' | 'color';
  openingTag: string;
  closingTag: string;
}

interface StyledCharacter {
  character: string;
  styles: BbcodeStyleToken[];
}

const BBCODE_TOKEN_AT_START_PATTERN = /^\[(\/)?(b|i|u|color)(?:=([^\]]+))?\]/i;

/** 把 BBCode 解析为带格式栈的可见字符，供隐藏源码的文本框反向编辑。 */
function parseBbcodeStyledCharacters(source: string): StyledCharacter[] {
  const result: StyledCharacter[] = [];
  const activeStyles: BbcodeStyleToken[] = [];
  let sourceOffset = 0;

  while (sourceOffset < source.length) {
    const remaining = source.slice(sourceOffset);
    const tagMatch = remaining.match(BBCODE_TOKEN_AT_START_PATTERN);
    if (tagMatch) {
      const isClosing = Boolean(tagMatch[1]);
      const tagName = tagMatch[2].toLowerCase() as BbcodeStyleToken['name'];
      if (isClosing) {
        for (let index = activeStyles.length - 1; index >= 0; index -= 1) {
          if (activeStyles[index].name === tagName) {
            activeStyles.splice(index, 1);
            break;
          }
        }
      } else if (tagName !== 'color' || tagMatch[3]) {
        const openingTag = tagName === 'color'
          ? `[color=${tagMatch[3]}]`
          : `[${tagName}]`;
        activeStyles.push({
          name: tagName,
          openingTag,
          closingTag: `[/${tagName}]`,
        });
      }
      sourceOffset += tagMatch[0].length;
      continue;
    }

    const codePoint = source.codePointAt(sourceOffset);
    if (codePoint === undefined) {
      break;
    }
    const character = String.fromCodePoint(codePoint);
    result.push({
      character,
      styles: activeStyles.map((style) => ({ ...style })),
    });
    sourceOffset += character.length;
  }

  return result;
}

/** 比较两个字符格式栈是否一致。 */
function areStyleStacksEqual(left: BbcodeStyleToken[], right: BbcodeStyleToken[]): boolean {
  if (left.length !== right.length) {
    return false;
  }
  return left.every((style, index) => style.openingTag === right[index].openingTag);
}

/** 把带格式栈的字符重新序列化为服务端持久化的 BBCode。 */
function serializeStyledCharacters(characters: StyledCharacter[]): string {
  let result = '';
  let activeStyles: BbcodeStyleToken[] = [];

  for (const entry of characters) {
    if (!areStyleStacksEqual(activeStyles, entry.styles)) {
      for (let index = activeStyles.length - 1; index >= 0; index -= 1) {
        result += activeStyles[index].closingTag;
      }
      for (const style of entry.styles) {
        result += style.openingTag;
      }
      activeStyles = entry.styles.map((style) => ({ ...style }));
    }
    result += entry.character;
  }

  for (let index = activeStyles.length - 1; index >= 0; index -= 1) {
    result += activeStyles[index].closingTag;
  }
  return result;
}

/** 返回不含 BBCode 标签的运营可读文本。 */
export function bbcodeToPlainText(source: string): string {
  return parseBbcodeStyledCharacters(source).map((entry) => entry.character).join('');
}

/**
 * 把上方普通文本框的修改合并回 BBCode。
 * 未变文字保留原格式，新输入文字继承编辑位置附近的格式。
 */
export function updateBbcodeFromPlainText(source: string, nextPlainText: string): string {
  const previousCharacters = parseBbcodeStyledCharacters(source);
  const previousPlainCharacters = previousCharacters.map((entry) => entry.character);
  const nextPlainCharacters = Array.from(nextPlainText);

  let prefixLength = 0;
  while (
    prefixLength < previousPlainCharacters.length
    && prefixLength < nextPlainCharacters.length
    && previousPlainCharacters[prefixLength] === nextPlainCharacters[prefixLength]
  ) {
    prefixLength += 1;
  }

  let suffixLength = 0;
  while (
    suffixLength < previousPlainCharacters.length - prefixLength
    && suffixLength < nextPlainCharacters.length - prefixLength
    && previousPlainCharacters[previousPlainCharacters.length - 1 - suffixLength]
      === nextPlainCharacters[nextPlainCharacters.length - 1 - suffixLength]
  ) {
    suffixLength += 1;
  }

  const insertedCharacters = nextPlainCharacters.slice(prefixLength, nextPlainCharacters.length - suffixLength);
  const inheritedStyles = previousCharacters[prefixLength]?.styles
    ?? previousCharacters[prefixLength - 1]?.styles
    ?? previousCharacters[previousCharacters.length - suffixLength]?.styles
    ?? [];
  const nextCharacters: StyledCharacter[] = [
    ...previousCharacters.slice(0, prefixLength),
    ...insertedCharacters.map((character) => ({
      character,
      styles: inheritedStyles.map((style) => ({ ...style })),
    })),
    ...previousCharacters.slice(previousCharacters.length - suffixLength),
  ];

  return serializeStyledCharacters(nextCharacters);
}

/**
 * 将右侧预览中的可见字符偏移转换为 BBCode 原文偏移。
 * 标签本身不占可见字符，因此刷色只包裹运营实际选中的文字。
 */
export function visibleOffsetToBbcodeOffset(source: string, visibleOffset: number): number {
  const targetOffset = Math.max(0, visibleOffset);
  let sourceOffset = 0;
  let currentVisibleOffset = 0;

  while (sourceOffset < source.length) {
    if (source[sourceOffset] === '[') {
      const tagEnd = source.indexOf(']', sourceOffset);
      if (tagEnd >= sourceOffset) {
        const tag = source.slice(sourceOffset, tagEnd + 1);
        if (/^\[(?:\/)?(?:b|i|u|color(?:=[^\]]+)?)\]$/i.test(tag)) {
          sourceOffset = tagEnd + 1;
          continue;
        }
      }
    }
    if (currentVisibleOffset >= targetOffset) {
      return sourceOffset;
    }
    sourceOffset += 1;
    currentVisibleOffset += 1;
  }

  return sourceOffset;
}

/** 将 BBCode 原文偏移转换为带格式字符数组的可见下标。 */
function bbcodeOffsetToVisibleIndex(source: string, sourceOffset: number): number {
  const targetOffset = Math.max(0, Math.min(sourceOffset, source.length));
  let currentSourceOffset = 0;
  let visibleIndex = 0;

  while (currentSourceOffset < targetOffset) {
    const remaining = source.slice(currentSourceOffset);
    const tagMatch = remaining.match(BBCODE_TOKEN_AT_START_PATTERN);
    if (tagMatch && currentSourceOffset + tagMatch[0].length <= targetOffset) {
      currentSourceOffset += tagMatch[0].length;
      continue;
    }
    const codePoint = source.codePointAt(currentSourceOffset);
    if (codePoint === undefined) {
      break;
    }
    const character = String.fromCodePoint(codePoint);
    if (currentSourceOffset + character.length > targetOffset) {
      break;
    }
    currentSourceOffset += character.length;
    visibleIndex += 1;
  }

  return visibleIndex;
}

/**
 * 把原文指定范围设置为新颜色。
 * 选中字符原有颜色会被替换，避免嵌套颜色标签导致预览与客户端解析错位。
 */
export function applyBbcodeColorRange(
  source: string,
  selectionStart: number,
  selectionEnd: number,
  colorValue: string,
): string {
  const start = Math.max(0, Math.min(selectionStart, source.length));
  const end = Math.max(start, Math.min(selectionEnd, source.length));
  if (start === end) {
    return source;
  }

  const visibleStart = bbcodeOffsetToVisibleIndex(source, start);
  const visibleEnd = bbcodeOffsetToVisibleIndex(source, end);
  if (visibleStart >= visibleEnd) {
    return source;
  }

  const characters = parseBbcodeStyledCharacters(source);
  const colorStyle: BbcodeStyleToken = {
    name: 'color',
    openingTag: `[color=${colorValue}]`,
    closingTag: '[/color]',
  };
  for (let index = visibleStart; index < visibleEnd && index < characters.length; index += 1) {
    const nonColorStyles = characters[index].styles.filter((style) => style.name !== 'color');
    characters[index].styles = [...nonColorStyles, { ...colorStyle }];
  }

  return serializeStyledCharacters(characters);
}
