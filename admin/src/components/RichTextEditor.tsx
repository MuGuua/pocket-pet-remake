import { Button, Card, Dropdown, Input, Space, Typography } from 'antd';
import type { TextAreaRef } from 'antd/es/input/TextArea';
import { useMemo, useRef, useState } from 'react';
import {
  RICH_TEXT_COLOR_PRESETS,
  RICH_TEXT_EXAMPLE_TEMPLATES,
  RICH_TEXT_TOOLBAR_ACTIONS,
  applyTextareaBbcodeAction,
  insertTextAtCursor,
  type RichTextToolbarAction,
} from '../utils/richTextBbcode';
import { PLAYER_NAME_TOKEN } from '../utils/itemMentionContent';
import { ItemMentionPickerModal } from './ItemMentionPickerModal';
import { ItemMentionPreview } from './ItemMentionPreview';

interface RichTextEditorProps {
  value?: string;
  onChange?: (nextValue: string) => void;
  rows?: number;
  placeholder?: string;
  disabled?: boolean;
  /** 是否展示实时预览区。 */
  showPreview?: boolean;
  /** 是否展示「插入系统模板」按钮（物品/装备/宠物）。 */
  enableItemMention?: boolean;
  /** 是否展示「@玩家」按钮（剧情对白专用）。 */
  enablePlayerNameMention?: boolean;
}

/**
 * 面向运营的 BBCode 富文本编辑器：工具栏快捷插入 + 系统模板选择 + 客户端效果预览。
 * 可直接作为 Ant Design Form.Item 的受控子组件。
 */
export function RichTextEditor({
  value = '',
  onChange,
  rows = 3,
  placeholder,
  disabled = false,
  showPreview = true,
  enableItemMention = true,
  enablePlayerNameMention = false,
}: RichTextEditorProps) {
  const textareaRef = useRef<TextAreaRef>(null);
  const [itemPickerOpen, setItemPickerOpen] = useState(false);

  const previewTitle = useMemo(
    () => (enablePlayerNameMention ? '对话预览（客户端最终效果）' : '客户端预览'),
    [enablePlayerNameMention],
  );

  function focusTextarea() {
    textareaRef.current?.focus();
  }

  function commitTextareaChange(nextValue: string, selectionStart: number, selectionEnd: number) {
    onChange?.(nextValue);
    requestAnimationFrame(() => {
      const native = textareaRef.current?.resizableTextArea?.textArea;
      if (!native) {
        return;
      }
      native.focus();
      native.setSelectionRange(selectionStart, selectionEnd);
    });
  }

  function insertToken(token: string) {
    const native = textareaRef.current?.resizableTextArea?.textArea;
    if (!native || disabled) {
      onChange?.(`${value}${value && !value.endsWith(' ') ? ' ' : ''}${token}`);
      return;
    }
    focusTextarea();
    const result = insertTextAtCursor(native, token);
    commitTextareaChange(result.nextValue, result.selectionStart, result.selectionEnd);
  }

  function handleToolbarAction(action: RichTextToolbarAction) {
    const native = textareaRef.current?.resizableTextArea?.textArea;
    if (!native || disabled) {
      onChange?.(`${value}${action.insert}`);
      return;
    }
    focusTextarea();
    const result = applyTextareaBbcodeAction(native, action);
    commitTextareaChange(result.nextValue, result.selectionStart, result.selectionEnd);
  }

  function handleColorPreset(colorValue: string) {
    const native = textareaRef.current?.resizableTextArea?.textArea;
    if (!native || disabled) {
      onChange?.(`${value}[color=${colorValue}]文字[/color]`);
      return;
    }
    focusTextarea();
    const result = applyTextareaBbcodeAction(native, RICH_TEXT_TOOLBAR_ACTIONS[0], colorValue);
    commitTextareaChange(result.nextValue, result.selectionStart, result.selectionEnd);
  }

  function handleInsertTemplate(templateValue: string) {
    if (disabled) {
      return;
    }
    if (!value.trim()) {
      onChange?.(templateValue);
      return;
    }
    onChange?.(`${value}\n${templateValue}`);
  }

  return (
    <>
      <Space direction="vertical" size={8} style={{ width: '100%' }}>
        <Space wrap size={[4, 4]}>
          {RICH_TEXT_TOOLBAR_ACTIONS.map((action) => (
            <Button key={action.key} size="small" disabled={disabled} onClick={() => handleToolbarAction(action)}>
              {action.label}
            </Button>
          ))}
          {RICH_TEXT_COLOR_PRESETS.map((preset) => (
            <Button
              key={preset.value}
              size="small"
              disabled={disabled}
              onClick={() => handleColorPreset(preset.value)}
            >
              {preset.label}
            </Button>
          ))}
          {enableItemMention ? (
            <Button size="small" disabled={disabled} onClick={() => setItemPickerOpen(true)}>
              插入系统模板
            </Button>
          ) : null}
          {enablePlayerNameMention ? (
            <Button size="small" disabled={disabled} onClick={() => insertToken(PLAYER_NAME_TOKEN)}>
              @玩家
            </Button>
          ) : null}
          <Dropdown
            menu={{
              items: RICH_TEXT_EXAMPLE_TEMPLATES.map((item) => ({
                key: item.label,
                label: item.label,
                onClick: () => handleInsertTemplate(item.value),
              })),
            }}
            trigger={['click']}
            disabled={disabled}
          >
            <Button size="small">插入示例</Button>
          </Dropdown>
        </Space>

        <Input.TextArea
          ref={textareaRef}
          value={value}
          rows={rows}
          placeholder={placeholder}
          disabled={disabled}
          onChange={(event) => onChange?.(event.target.value)}
        />

        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          支持 Godot BBCode：{' '}
          <Typography.Text code>[b][/b]</Typography.Text>{' '}
          <Typography.Text code>[color=green][/color]</Typography.Text>{' '}
          <Typography.Text code>[i][/i]</Typography.Text>{' '}
          <Typography.Text code>[u][/u]</Typography.Text>
          ；占位符 {'{item:物品ID}'} / {'{pet:宠物ID}'}
          {enablePlayerNameMention ? ' / {player_name}' : ''}。
        </Typography.Text>

        {showPreview ? (
          <ItemMentionPreview
            content={value}
            title={previewTitle}
            showPlayerName={enablePlayerNameMention}
          />
        ) : null}
      </Space>

      {enableItemMention ? (
        <ItemMentionPickerModal
          open={itemPickerOpen}
          onCancel={() => setItemPickerOpen(false)}
          onSelect={(token) => insertToken(token)}
        />
      ) : null}
    </>
  );
}
