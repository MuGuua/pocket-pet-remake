import { Button, Form, Input, InputNumber, Modal, Select, Space, Switch, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import type { SkillEffectConfigEntry, SkillEffectConfigType } from '../types/skillEffectConfig';
import {
  PASSIVE_ATTRIBUTE_KEY_OPTIONS,
  SKILL_EFFECT_CONFIG_TYPE_LABELS,
  SKILL_EFFECT_CONFIG_TYPE_OPTIONS,
  UNIQUE_SKILL_EFFECT_CONFIG_TYPES,
  createDefaultSkillEffectConfigRow,
  filterSkillEffectEntriesForActivationMode,
  formatSkillEffectConfigSummary,
  getPassiveAttributeModeOptions,
  isSkillEffectTypeAllowedForActivationMode,
  normalizeSkillEffectConfigEntry,
} from '../types/skillEffectConfig';
import { BATTLE_CONTROL_STATUS_OPTIONS } from '../utils/displayLabels';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../utils/modalLayout';

interface SkillEffectConfigEditorProps {
  value?: SkillEffectConfigEntry[];
  onChange?: (nextValue: SkillEffectConfigEntry[]) => void;
  activationMode?: string;
}

/** 技能公式/状态/表现配置：表格预览 + 弹窗新增，交互对齐怪物战斗奖励编辑器。 */
export function SkillEffectConfigEditor({ value = [], onChange, activationMode = 'active' }: SkillEffectConfigEditorProps) {
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editorForm] = Form.useForm<SkillEffectConfigEntry>();
  const entryType = Form.useWatch('entry_type', editorForm);
  const passiveAttrKey = Form.useWatch('passive_attr_key', editorForm);

  const usedTypes = useMemo(
    () => new Set(
      value
        .map((entry, index) => (editingIndex === index ? null : entry.entry_type))
        .filter((item): item is SkillEffectConfigType => item != null),
    ),
    [value, editingIndex],
  );

  const addableTypeOptions = useMemo(
    () => SKILL_EFFECT_CONFIG_TYPE_OPTIONS.filter((option) => {
      if (!isSkillEffectTypeAllowedForActivationMode(option.value, activationMode)) {
        return false;
      }
      if (!UNIQUE_SKILL_EFFECT_CONFIG_TYPES.has(option.value)) {
        return true;
      }
      return !usedTypes.has(option.value);
    }),
    [activationMode, usedTypes],
  );

  const visibleEntries = useMemo(
    () => filterSkillEffectEntriesForActivationMode(value, activationMode),
    [activationMode, value],
  );

  const columns = useMemo<ColumnsType<SkillEffectConfigEntry>>(
    () => [
      {
        title: '排序',
        dataIndex: 'sort_order',
        key: 'sort_order',
        width: 72,
        render: (sortOrder: number, _record, index) => (sortOrder > 0 ? sortOrder : index + 1),
      },
      {
        title: '类型',
        dataIndex: 'entry_type',
        key: 'entry_type',
        width: 140,
        render: (type: SkillEffectConfigType) => (
          <Tag color="blue">{SKILL_EFFECT_CONFIG_TYPE_LABELS[type] ?? type}</Tag>
        ),
      },
      {
        title: '配置摘要',
        key: 'summary',
        render: (_value, record) => formatSkillEffectConfigSummary(record),
      },
      {
        title: '操作',
        key: 'actions',
        width: 140,
        fixed: 'right',
        render: (_value, _record, index) => (
          <Space size={8}>
            <Button size="small" onClick={() => openEditor(index)}>编辑</Button>
            <Button size="small" danger onClick={() => removeEntry(index)}>删除</Button>
          </Space>
        ),
      },
    ],
    [value],
  );

  function emitChange(nextValue: SkillEffectConfigEntry[]) {
    onChange?.(nextValue);
  }

  function openEditor(index: number | null) {
    setEditingIndex(index);
    if (index === null) {
      const defaultType = addableTypeOptions[0]?.value ?? 'damage_attack_pct';
      editorForm.setFieldsValue(createDefaultSkillEffectConfigRow(defaultType, value.length));
    } else {
      editorForm.setFieldsValue(value[index]);
    }
    setEditorOpen(true);
  }

  function removeEntry(index: number) {
    emitChange(value.filter((_entry, entryIndex) => entryIndex !== index));
  }

  function handleSubmitEditor(formValues: SkillEffectConfigEntry) {
    const nextEntry = normalizeSkillEffectConfigEntry(formValues, editingIndex ?? value.length);
    const duplicateIndex = value.findIndex(
      (entry, index) => entry.entry_type === nextEntry.entry_type && index !== editingIndex,
    );
    if (duplicateIndex >= 0 && UNIQUE_SKILL_EFFECT_CONFIG_TYPES.has(nextEntry.entry_type)) {
      Modal.warning({ title: '配置类型重复', content: '每种效果类型最多只能配置一条，请编辑已有条目。' });
      return;
    }
    const nextValue = [...value];
    if (editingIndex === null) {
      nextValue.push(nextEntry);
    } else {
      nextValue[editingIndex] = nextEntry;
    }
    emitChange(nextValue);
    setEditorOpen(false);
    setEditingIndex(null);
    editorForm.resetFields();
  }

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <Typography.Text type="secondary">
          共 {visibleEntries.length} 条效果配置。伤害/治疗/控制/表现均通过「添加效果」录入，保存时自动合并为服务端字段。
        </Typography.Text>
        <Button type="primary" disabled={addableTypeOptions.length === 0} onClick={() => openEditor(null)}>
          添加效果
        </Button>
      </Space>
      <Table
        size="small"
        rowKey={(_record, index) => String(index)}
        columns={columns}
        dataSource={visibleEntries}
        pagination={false}
        scroll={{ x: 760 }}
        locale={{ emptyText: '尚未配置技能效果，请点击「添加效果」。' }}
      />
      <Modal
        title={editingIndex === null ? '添加技能效果' : '编辑技能效果'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingIndex(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        destroyOnClose
        width={680}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText="确定"
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(formValues) => handleSubmitEditor(formValues)}>
          <Form.Item label="效果类型" name="entry_type" rules={[{ required: true, message: '请选择效果类型' }]}>
            <Select
              disabled={editingIndex !== null}
              options={
                editingIndex === null
                  ? addableTypeOptions
                  : SKILL_EFFECT_CONFIG_TYPE_OPTIONS.filter((option) => option.value === entryType)
              }
            />
          </Form.Item>
          <Form.Item label="排序" name="sort_order">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          {entryType === 'damage_skill_mult' ? (
            <Form.Item label="技能倍数" name="skill_mult" rules={[{ required: true, message: '请输入技能倍数' }]}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          ) : null}
          {entryType === 'attribute_bonus' ? (
            <>
              <Form.Item label="属性类型" name="passive_attr_key" rules={[{ required: true, message: '请选择属性类型' }]}>
                <Select options={PASSIVE_ATTRIBUTE_KEY_OPTIONS.map((option) => ({ value: option.value, label: option.label }))} />
              </Form.Item>
              <Form.Item label="加成方式" name="passive_attr_mode" rules={[{ required: true, message: '请选择加成方式' }]}>
                <Select options={getPassiveAttributeModeOptions(passiveAttrKey).map((option) => ({ value: option.value, label: option.label }))} />
              </Form.Item>
              <Form.Item label="加成数值" name="passive_attr_value" rules={[{ required: true, message: '请输入加成数值' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </>
          ) : null}
          {entryType === 'damage_attack_pct' ? (
            <Form.Item label="攻击系数 (%)" name="attack_pct" rules={[{ required: true, message: '请输入攻击系数' }]}>
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>
          ) : null}
          {entryType === 'heal' ? (
            <>
              <Form.Item label="治疗系数 (%)" name="heal_pct" rules={[{ required: true, message: '请输入治疗系数' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item label="固定治疗" name="fixed_heal">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </>
          ) : null}
          {entryType === 'fixed_damage' ? (
            <Form.Item label="固定伤害" name="fixed_damage" rules={[{ required: true, message: '请输入固定伤害' }]}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          ) : null}
          {entryType === 'formula_flags' ? (
            <>
              <Form.Item label="技能攻击" name="is_skill_attack" valuePropName="checked"><Switch /></Form.Item>
              <Form.Item label="允许暴击" name="allow_crit" valuePropName="checked"><Switch /></Form.Item>
              <Form.Item label="忽略防御" name="ignore_defense" valuePropName="checked"><Switch /></Form.Item>
            </>
          ) : null}
          {entryType === 'advanced_coefficients' ? (
            <>
              <Form.Item label="法力系数" name="mana_pct"><InputNumber style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="防御系数" name="defense_pct"><InputNumber style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="速度系数" name="speed_pct"><InputNumber style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="技能附加爆伤" name="skill_crit_add"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="目标当前生命%" name="target_current_hp_pct"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'hit_bonus' ? (
            <Form.Item label="技能命中加成" name="skill_hit_bonus" rules={[{ required: true, message: '请输入技能命中加成' }]}>
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>
          ) : null}
          {entryType === 'seal' ? (
            <>
              <Form.Item label="封印概率 (%)" name="seal_chance_pct"><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="封印威力" name="seal_power"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="封印回合" name="seal_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'control' ? (
            <>
              <Form.Item label="控制概率 (%)" name="control_chance_pct"><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="控制威力" name="control_power"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="控制回合" name="control_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="控制状态" name="control_status_id">
                <Select
                  showSearch
                  optionFilterProp="label"
                  options={[
                    { value: 0, label: '无 / 不使用通用控制' },
                    ...BATTLE_CONTROL_STATUS_OPTIONS,
                  ]}
                />
              </Form.Item>
            </>
          ) : null}
          {entryType === 'bleed' ? (
            <>
              <Form.Item label="流血概率 (%)" name="bleed_chance_pct"><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="流血回合" name="bleed_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="流血伤害" name="bleed_damage"><InputNumber style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'curse' ? (
            <>
              <Form.Item label="诅咒概率 (%)" name="curse_chance_pct"><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="诅咒回合" name="curse_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="诅咒伤害" name="curse_damage"><InputNumber style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="诅咒法力%" name="curse_mana_pct"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'crit_boost' ? (
            <>
              <Form.Item label="暴击增益 (%)" name="crit_boost_pct"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="暴击增益回合" name="crit_boost_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'slow' ? (
            <>
              <Form.Item label="减速概率 (%)" name="slow_chance_pct"><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="减速回合" name="slow_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="减速倍率 (%)" name="slow_multiplier_pct"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'armor_break' ? (
            <>
              <Form.Item label="破甲%" name="armor_break_pct"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="破甲概率%" name="armor_break_chance_pct"><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="破甲回合" name="armor_break_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'vulnerability' ? (
            <>
              <Form.Item label="易伤%" name="vulnerability_pct"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="易伤概率%" name="vulnerability_chance_pct"><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="易伤回合" name="vulnerability_rounds"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
              <Form.Item label="易伤施加%" name="vulnerability_apply_pct"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </>
          ) : null}
          {entryType === 'presentation' ? (
            <>
              <Form.Item label="动画键" name="animation_key"><Input placeholder="slash" /></Form.Item>
              <Form.Item label="视觉资源ID" name="skill_visual_id"><Input placeholder="可选" /></Form.Item>
              <Form.Item label="施法色" name="cast_color"><Input placeholder="#EBEBF5" /></Form.Item>
              <Form.Item label="命中色" name="impact_color"><Input placeholder="#FFF2F2" /></Form.Item>
              <Form.Item label="投射物" name="projectile" valuePropName="checked"><Switch /></Form.Item>
            </>
          ) : null}
        </Form>
      </Modal>
    </Space>
  );
}
