import { ItemMentionPreview } from './ItemMentionPreview';

interface RichTextDisplayProps {
  /** 原始 BBCode 文本；空值时展示占位符。 */
  value?: string | null;
  /** 空内容占位文案。 */
  emptyText?: string;
}

/**
 * 后台详情抽屉/描述列表中的富文本展示。
 * 与客户端 Godot RichTextLabel 使用同一套 BBCode 与 mention 占位符语法。
 */
export function RichTextDisplay({
  value,
  emptyText = '-',
}: RichTextDisplayProps) {
  const normalized = String(value ?? '').trim();
  if (!normalized) {
    return <span style={{ color: '#999' }}>{emptyText}</span>;
  }

  return (
    <ItemMentionPreview
      content={normalized}
      embedded
    />
  );
}
