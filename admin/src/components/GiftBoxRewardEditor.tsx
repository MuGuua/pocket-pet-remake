import { Button, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import { GrantableItemSelect } from './GrantableItemSelect';
import type { GiftBoxRewardEntry } from '../types/giftBoxReward';
import { GIFT_BOX_REWARD_TYPE_OPTIONS } from '../types/giftBoxReward';

interface GiftBoxRewardEditorProps {
  value?: GiftBoxRewardEntry[];
  onChange?: (nextValue: GiftBoxRewardEntry[]) => void;
}

interface RewardEditorFormValues {
  type: 'item' | 'gold';
  item_id?: number;
  item_name?: string;
  count?: number;
  value?: number;
}

/** 礼包内容可视化编辑器：维护开启后发放的物品与数量。 */
export function GiftBoxRewardEditor({ value = [], onChange }: GiftBoxRewardEditorProps) {
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [preferredItemID, setPreferredItemID] = useState<number | undefined>(undefined);
  const [preferredItemName, setPreferredItemName] = useState('');
  const [editorForm] = Form.useForm<RewardEditorFormValues>();
  const rewardType = Form.useWatch('type', editorForm);

  const columns = useMemo<ColumnsType<GiftBoxRewardEntry>>(
    () => [
      {
        title: '奖励类型',
        dataIndex: 'type',
        key: 'type',
        width: 120,
        render: (type: GiftBoxRewardEntry['type']) => (
          <Tag color={type === 'item' ? 'blue' : 'gold'}>{type === 'item' ? '物品' : '金币'}</Tag>
        ),
      },
      {
        title: '内容',
        key: 'content',
        render: (_value, record) => {
          if (record.type === 'gold') {
            return `铜币 ${record.value ?? 0}`;
          }
          const itemName = record.item_name?.trim();
          const nameText = itemName ? itemName : `物品ID ${record.item_id ?? 0}`;
          return `${nameText} × ${record.count ?? 1}`;
        },
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

  function emitChange(nextValue: GiftBoxRewardEntry[]) {
    onChange?.(nextValue);
  }

  function openEditor(index: number | null) {
    setEditingIndex(index);
    if (index === null) {
      setPreferredItemID(undefined);
      setPreferredItemName('');
      editorForm.setFieldsValue({
        type: 'item',
        item_id: undefined,
        item_name: '',
        count: 1,
        value: undefined,
      });
    } else {
      const current = value[index];
      setPreferredItemID(current.item_id);
      setPreferredItemName(current.item_name ?? '');
      editorForm.setFieldsValue({
        type: current.type,
        item_id: current.item_id,
        item_name: current.item_name,
        count: current.count ?? 1,
        value: current.value,
      });
    }
    setEditorOpen(true);
  }

  function removeEntry(index: number) {
    emitChange(value.filter((_entry, entryIndex) => entryIndex !== index));
  }

  function handleSubmitEditor(formValues: RewardEditorFormValues) {
    const nextEntry: GiftBoxRewardEntry = formValues.type === 'gold'
      ? { type: 'gold', value: Number(formValues.value ?? 0) }
      : {
          type: 'item',
          item_id: Number(formValues.item_id ?? 0),
          item_name: formValues.item_name ?? '',
          count: Number(formValues.count ?? 1),
        };
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
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Text type="secondary">配置打开礼包后固定获得的奖励内容。</Typography.Text>
        <Button type="primary" onClick={() => openEditor(null)}>添加奖励</Button>
      </Space>
      <Table
        size="small"
        rowKey={(_record, index) => String(index)}
        columns={columns}
        dataSource={value}
        pagination={false}
        locale={{ emptyText: '尚未配置礼包内容，请点击“添加奖励”。' }}
      />
      <Modal
        title={editingIndex === null ? '添加礼包奖励' : '编辑礼包奖励'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingIndex(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        destroyOnClose
        width={560}
      >
        <Form
          form={editorForm}
          layout="vertical"
          onFinish={(formValues) => handleSubmitEditor(formValues)}
        >
          <Form.Item label="奖励类型" name="type" rules={[{ required: true, message: '请选择奖励类型' }]}>
            <Select options={GIFT_BOX_REWARD_TYPE_OPTIONS.map((option) => ({ value: option.value, label: option.label }))} />
          </Form.Item>
          {rewardType === 'item' ? (
            <>
              <Form.Item label="物品" name="item_id" rules={[{ required: true, message: '请选择物品' }]}>
                <GrantableItemSelect
                  showCategoryFilter
                  preferredItemID={preferredItemID}
                  preferredItemName={preferredItemName}
                  onItemChange={(item) => {
                    editorForm.setFieldValue('item_name', item?.item_name ?? '');
                  }}
                />
              </Form.Item>
              <Form.Item name="item_name" hidden>
                <Input />
              </Form.Item>
              <Form.Item label="数量" name="count" rules={[{ required: true, message: '请输入数量' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </>
          ) : (
            <Form.Item label="铜币数量" name="value" rules={[{ required: true, message: '请输入铜币数量' }]}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </Space>
  );
}
