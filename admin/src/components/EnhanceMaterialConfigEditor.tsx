import { Form, InputNumber, Select, Space, Switch, Typography } from 'antd';
import { RichTextEditor } from './RichTextEditor';
import type { EnhanceMaterialConfig, EnhanceMaterialFailurePenalty, EnhanceMaterialSuccessMode } from '../types/enhanceMaterialConfig';
import {
  ENHANCE_MATERIAL_FAILURE_PENALTY_OPTIONS,
  ENHANCE_MATERIAL_SUCCESS_MODE_OPTIONS,
  normalizeEnhanceMaterialConfig,
} from '../types/enhanceMaterialConfig';

interface EnhanceMaterialConfigEditorProps {
  value?: EnhanceMaterialConfig;
  onChange?: (nextValue: EnhanceMaterialConfig) => void;
}

/** 强化材料（item_sub_type=equipment_enhance）锻造效果配置编辑器。 */
export function EnhanceMaterialConfigEditor({ value, onChange }: EnhanceMaterialConfigEditorProps) {
  const normalized = normalizeEnhanceMaterialConfig(value);

  function emitChange(partial: Partial<EnhanceMaterialConfig>) {
    onChange?.(normalizeEnhanceMaterialConfig({ ...normalized, ...partial }));
  }

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Typography.Text type="secondary">
        配置该强化石在客户端强化面板中的成功率修正与失败惩罚；「沿用全局」读取「物品模板 → 强化成功率」中当前装备穿戴等级段的基础表。
      </Typography.Text>
      <Form layout="vertical">
        <Form.Item label="100% 强化成功">
          <Switch
            checked={normalized.guaranteed_success}
            checkedChildren="是"
            unCheckedChildren="否"
            onChange={(checked) => emitChange({ guaranteed_success: checked })}
          />
        </Form.Item>
        {!normalized.guaranteed_success ? (
          <>
            <Form.Item label="成功率模式">
              <Select
                value={normalized.success_rate_mode}
                options={ENHANCE_MATERIAL_SUCCESS_MODE_OPTIONS}
                onChange={(nextMode: EnhanceMaterialSuccessMode) => emitChange({ success_rate_mode: nextMode })}
              />
            </Form.Item>
            {normalized.success_rate_mode === 'bonus' ? (
              <Form.Item label="成功率加成（百分点）" extra="最终成功率 = 全局成功率 + 加成，上限 100%。">
                <InputNumber
                  min={0}
                  max={100}
                  style={{ width: '100%' }}
                  value={normalized.success_rate_bonus_pct}
                  onChange={(nextValue) => emitChange({ success_rate_bonus_pct: Number(nextValue ?? 0) })}
                />
              </Form.Item>
            ) : null}
            {normalized.success_rate_mode === 'override' ? (
              <Form.Item label="固定成功率（%）" extra="忽略全局表，直接使用该固定值。">
                <InputNumber
                  min={1}
                  max={100}
                  style={{ width: '100%' }}
                  value={normalized.success_rate_override_pct}
                  onChange={(nextValue) => emitChange({ success_rate_override_pct: Number(nextValue ?? 0) })}
                />
              </Form.Item>
            ) : null}
          </>
        ) : null}
        <Form.Item label="失败惩罚">
          <Select
            value={normalized.failure_penalty}
            options={ENHANCE_MATERIAL_FAILURE_PENALTY_OPTIONS}
            onChange={(nextPenalty: EnhanceMaterialFailurePenalty) => emitChange({ failure_penalty: nextPenalty })}
          />
        </Form.Item>
        {normalized.failure_penalty === 'level_down' ? (
          <Form.Item label="失败降级数" extra="失败时降低的强化等级，装备不会损坏。">
            <InputNumber
              min={1}
              max={15}
              style={{ width: '100%' }}
              value={normalized.failure_level_delta}
              onChange={(nextValue) => emitChange({ failure_level_delta: Number(nextValue ?? 1) })}
            />
          </Form.Item>
        ) : null}
        <Form.Item label="客户端说明文案" extra="可选；会随强化预览材料选项下发，供弹窗展示；可在下方预览中刷色。">
          <RichTextEditor
            rows={3}
            value={normalized.description}
            onChange={(nextValue) => emitChange({ description: nextValue })}
            placeholder="例如：精炼宝石，成功率 +15%，失败仍会损坏装备。"
          />
        </Form.Item>
      </Form>
    </Space>
  );
}
