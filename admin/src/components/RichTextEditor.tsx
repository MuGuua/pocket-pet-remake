import { Button, Card, Input, Modal, Space, Typography } from 'antd';
import { useRef, useState } from 'react';
import {
  RICH_TEXT_COLOR_PRESETS,
  applyBbcodeColorRange,
  bbcodeToPlainText,
  updateBbcodeFromPlainText,
  visibleOffsetToBbcodeOffset,
} from '../utils/richTextBbcode';
import { ItemMentionPreview } from './ItemMentionPreview';

interface RichTextEditorProps {
  value?: string;
  onChange?: (nextValue: string) => void;
  rows?: number;
  placeholder?: string;
  disabled?: boolean;
  /** 是否在右侧预览中把玩家名占位符渲染为示例名称。 */
  enablePlayerNameMention?: boolean;
}

interface RichTextVisualSelection {
  sourceStart: number;
  sourceEnd: number;
  text: string;
}

/** 查找预览边界所在的原文映射节点。 */
function findSourceElement(node: Node, previewRoot: HTMLElement): HTMLElement | null {
  let current: HTMLElement | null = node.nodeType === Node.ELEMENT_NODE
    ? node as HTMLElement
    : node.parentElement;
  while (current && current !== previewRoot) {
    if (current.dataset.richSourceStart !== undefined && current.dataset.richSourceEnd !== undefined) {
      return current;
    }
    current = current.parentElement;
  }
  return null;
}

/** 将右侧 DOM 选区边界映射回 BBCode 原文偏移。 */
function resolveSourceBoundary(
  source: string,
  previewRoot: HTMLElement,
  boundaryNode: Node,
  boundaryOffset: number,
  isEndBoundary: boolean,
): number | null {
  const sourceElement = findSourceElement(boundaryNode, previewRoot);
  if (!sourceElement) {
    return null;
  }
  const sourceStart = Number(sourceElement.dataset.richSourceStart ?? 0);
  const sourceEnd = Number(sourceElement.dataset.richSourceEnd ?? sourceStart);
  if (sourceElement.dataset.richSourceKind === 'mention') {
    return isEndBoundary ? sourceEnd : sourceStart;
  }

  try {
    const localRange = document.createRange();
    localRange.selectNodeContents(sourceElement);
    localRange.setEnd(boundaryNode, boundaryOffset);
    const visibleOffset = localRange.toString().length;
    const sourceSlice = source.slice(sourceStart, sourceEnd);
    return sourceStart + visibleOffsetToBbcodeOffset(sourceSlice, visibleOffset);
  } catch {
    return isEndBoundary ? sourceEnd : sourceStart;
  }
}

/**
 * 后台统一 BBCode 富文本编辑器。
 * 默认只展示合并后的客户端效果卡片，点击编辑后再打开“原文输入 + 刷色预览”的编辑表单。
 */
export function RichTextEditor({
  value = '',
  onChange,
  rows = 3,
  placeholder,
  disabled = false,
  enablePlayerNameMention = false,
}: RichTextEditorProps) {
  const previewRef = useRef<HTMLDivElement>(null);
  const [editing, setEditing] = useState(false);
  const [draftValue, setDraftValue] = useState(value);
  const [visualSelection, setVisualSelection] = useState<RichTextVisualSelection | null>(null);

  function captureVisualSelection() {
    const previewRoot = previewRef.current;
    const selection = window.getSelection();
    if (!previewRoot || !selection || selection.rangeCount === 0 || selection.isCollapsed) {
      setVisualSelection(null);
      return;
    }
    const range = selection.getRangeAt(0);
    if (!previewRoot.contains(range.startContainer) || !previewRoot.contains(range.endContainer)) {
      setVisualSelection(null);
      return;
    }
    const sourceStart = resolveSourceBoundary(draftValue, previewRoot, range.startContainer, range.startOffset, false);
    const sourceEnd = resolveSourceBoundary(draftValue, previewRoot, range.endContainer, range.endOffset, true);
    if (sourceStart === null || sourceEnd === null || sourceEnd <= sourceStart) {
      setVisualSelection(null);
      return;
    }
    setVisualSelection({ sourceStart, sourceEnd, text: selection.toString() });
  }

  function handleVisualColorBrush(colorValue: string) {
    if (disabled || !visualSelection) {
      return;
    }
    setDraftValue(applyBbcodeColorRange(draftValue, visualSelection.sourceStart, visualSelection.sourceEnd, colorValue));
    setVisualSelection(null);
    window.getSelection()?.removeAllRanges();
  }

  function openEditor() {
    setDraftValue(value);
    setVisualSelection(null);
    setEditing(true);
  }

  function closeEditor() {
    setVisualSelection(null);
    setEditing(false);
    window.getSelection()?.removeAllRanges();
  }

  function confirmEditor() {
    onChange?.(draftValue);
    closeEditor();
  }

  const previewTitle = enablePlayerNameMention ? '对话效果' : '客户端效果';
  const editorPreviewTitle = enablePlayerNameMention ? '对话效果与刷色' : '客户端效果与刷色';
  const previewMinHeight = Math.max(120, rows * 24);
  const plainTextValue = bbcodeToPlainText(draftValue);

  return (
    <>
      <Card
        size="small"
        title={previewTitle}
        extra={
          <Button size="small" type="link" disabled={disabled} onClick={openEditor}>
            编辑
          </Button>
        }
        styles={{ body: { padding: 12 } }}
      >
        <div
          style={{
            minHeight: Math.max(48, rows * 20),
            padding: 12,
            overflow: 'auto',
            border: '1px solid #424b57',
            background: '#171b20',
          }}
        >
          <ItemMentionPreview
            content={value}
            embedded
            showPlayerName={enablePlayerNameMention}
          />
        </div>
      </Card>

      <Modal
        title="编辑富文本"
        open={editing}
        width={720}
        okText="保存"
        cancelText="取消"
        destroyOnHidden
        onOk={confirmEditor}
        onCancel={closeEditor}
      >
        <Space direction="vertical" size={10} style={{ width: '100%' }}>
          <Input.TextArea
            value={plainTextValue}
            rows={rows}
            placeholder={placeholder}
            disabled={disabled}
            onChange={(event) => {
              setVisualSelection(null);
              setDraftValue(updateBbcodeFromPlainText(draftValue, event.target.value));
            }}
          />

          <Card size="small" title={editorPreviewTitle} styles={{ body: { padding: 12 } }}>
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              <Space wrap size={[6, 6]}>
                {RICH_TEXT_COLOR_PRESETS.map((preset) => (
                  <Button
                    key={preset.value}
                    size="small"
                    disabled={disabled || !visualSelection}
                    title={`${preset.label} ${preset.value}`}
                    onClick={() => handleVisualColorBrush(preset.value)}
                    style={{ paddingInline: 8 }}
                  >
                    <span
                      aria-hidden="true"
                      style={{
                        display: 'inline-block',
                        width: 14,
                        height: 14,
                        marginRight: 6,
                        border: '1px solid rgba(0, 0, 0, 0.35)',
                        background: preset.value,
                        verticalAlign: -2,
                      }}
                    />
                    {preset.label}
                  </Button>
                ))}
              </Space>

              <Typography.Text type={visualSelection ? 'success' : 'secondary'} style={{ fontSize: 12 }}>
                {visualSelection ? `已选中「${visualSelection.text}」，点击颜色完成刷色。` : '在下方预览中拖选文字，再点击颜色笔刷。'}
              </Typography.Text>

              <div
                ref={previewRef}
                onMouseUp={captureVisualSelection}
                onKeyUp={captureVisualSelection}
                style={{
                  minHeight: previewMinHeight,
                  padding: 12,
                  overflow: 'auto',
                  border: '1px solid #424b57',
                  background: '#171b20',
                  cursor: 'text',
                  userSelect: 'text',
                }}
              >
                <ItemMentionPreview
                  content={draftValue}
                  embedded
                  showPlayerName={enablePlayerNameMention}
                />
              </div>
            </Space>
          </Card>
        </Space>
      </Modal>
    </>
  );
}
