import { Button, Form, InputNumber, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import type { AdminQuestRewardInput } from '../types/quest';
import { GrantableItemSelect } from './GrantableItemSelect';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../utils/modalLayout';

interface QuestRewardEditorProps {
  value?: AdminQuestRewardInput[];
  onChange?: (nextValue: AdminQuestRewardInput[]) => void;
}

interface RewardEditorFormValues {
  type: AdminQuestRewardInput['type'];
  value?: number;
  item_id?: number;
  count?: number;
}

const REWARD_TYPE_OPTIONS = [
  { label: '经验', value: 'exp' as const },
  { label: '铜币', value: 'gold' as const },
  { label: '物品', value: 'item' as const },
];

/** 任务模板奖励可视化编辑器：表格预览 + 弹窗编辑，物品奖励走可搜索下拉。 */
export function QuestRewardEditor({ value = [], onChange }: QuestRewardEditorProps) {
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [preferredItemID, setPreferredItemID] = useState<number | undefined>(undefined);
  const [preferredItemName, setPreferredItemName] = useState('');
  const [editorForm] = Form.useForm<RewardEditorFormValues>();
  const rewardType = Form.useWatch('type', editorForm);

  const columns = useMemo<ColumnsType<AdminQuestRewardInput>>(
    () => [
      {
        title: '类型',
        dataIndex: 'type',
        key: 'type',
        width: 88,
        render: (type: AdminQuestRewardInput['type']) => {
          if (type === 'item') {
            return <Tag color="blue">物品</Tag>;
          }
          if (type === 'gold') {
            return <Tag color="gold">铜币</Tag>;
          }
          return <Tag color="green">经验</Tag>;
        },
      },
      {
        title: '奖励内容',
        key: 'content',
        render: (_value, record) => formatQuestRewardSummary(record),
      },
      {
        title: '操作',
        key: 'actions',
        width: 140,
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

  function emitChange(nextValue: AdminQuestRewardInput[]) {
    onChange?.(nextValue);
  }

  function openEditor(index: number | null) {
    setEditingIndex(index);
    if (index === null) {
      setPreferredItemID(undefined);
      setPreferredItemName('');
      editorForm.setFieldsValue({
        type: 'exp',
        value: 50,
        item_id: undefined,
        count: 1,
      });
    } else {
      const current = value[index];
      setPreferredItemID(current.type === 'item' && current.item_id > 0 ? current.item_id : undefined);
      setPreferredItemName('');
      editorForm.setFieldsValue({
        type: current.type,
        value: current.type === 'item' ? 0 : current.value,
        item_id: current.item_id > 0 ? current.item_id : undefined,
        count: current.count > 0 ? current.count : 1,
      });
    }
    setEditorOpen(true);
  }

  function removeEntry(index: number) {
    emitChange(value.filter((_entry, entryIndex) => entryIndex !== index));
  }

  function handleSubmitEditor(formValues: RewardEditorFormValues) {
    const nextEntry = normalizeQuestRewardEntry(formValues);
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
          配置任务提交成功后发放的经验、铜币与物品奖励。
        </Typography.Text>
        <Button type="primary" onClick={() => openEditor(null)}>添加奖励</Button>
      </Space>
      <Table
        size="small"
        rowKey={(_record, index) => String(index)}
        columns={columns}
        dataSource={value}
        pagination={false}
        locale={{ emptyText: '尚未配置任务奖励，请点击「添加奖励」。' }}
      />
      <Modal
        title={editingIndex === null ? '添加任务奖励' : '编辑任务奖励'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingIndex(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        destroyOnClose
        width={640}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText="确定"
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(formValues) => handleSubmitEditor(formValues)}>
          <Form.Item label="奖励类型" name="type" rules={[{ required: true, message: '请选择奖励类型' }]}>
            <Select options={REWARD_TYPE_OPTIONS} />
          </Form.Item>
          {rewardType === 'exp' ? (
            <Form.Item label="经验值" name="value" rules={[{ required: true, message: '请输入经验值' }]}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          ) : null}
          {rewardType === 'gold' ? (
            <Form.Item label="铜币数量" name="value" rules={[{ required: true, message: '请输入铜币数量' }]}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          ) : null}
          {rewardType === 'item' ? (
            <>
              <Form.Item label="物品" name="item_id" rules={[{ required: true, message: '请选择物品' }]}>
                <GrantableItemSelect
                  showCategoryFilter
                  preferredItemID={preferredItemID}
                  preferredItemName={preferredItemName}
                />
              </Form.Item>
              <Form.Item label="数量" name="count" rules={[{ required: true, message: '请输入数量' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </>
          ) : null}
        </Form>
      </Modal>
    </Space>
  );
}

/** 将表单值整理为任务奖励条目。 */
export function normalizeQuestRewardEntry(formValues: RewardEditorFormValues): AdminQuestRewardInput {
  if (formValues.type === 'item') {
    return {
      type: 'item',
      value: 0,
      item_id: Number(formValues.item_id ?? 0),
      count: Number(formValues.count ?? 0),
    };
  }
  return {
    type: formValues.type,
    value: Number(formValues.value ?? 0),
    item_id: 0,
    count: 0,
  };
}

function formatQuestRewardSummary(record: AdminQuestRewardInput): string {
  if (record.type === 'item') {
    const itemID = Number(record.item_id ?? 0);
    const count = Number(record.count ?? 0);
    return itemID > 0 ? `物品 #${itemID} × ${count}` : '未选择物品';
  }
  if (record.type === 'gold') {
    return `铜币 ${Number(record.value ?? 0)}`;
  }
  return `角色经验 ${Number(record.value ?? 0)}`;
}
