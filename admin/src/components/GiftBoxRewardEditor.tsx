import { Button, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { formatPetQualityLabel } from '../constants/petQuality';
import { fetchAllEnabledAdminPetDefinitions } from '../services/petDefinition';
import type { AdminPetDefinitionSummary } from '../types/petDefinition';
import { GrantableItemSelect } from './GrantableItemSelect';
import type { GiftBoxRewardEntry } from '../types/giftBoxReward';
import { GIFT_BOX_REWARD_TYPE_OPTIONS } from '../types/giftBoxReward';

interface GiftBoxRewardEditorProps {
  value?: GiftBoxRewardEntry[];
  onChange?: (nextValue: GiftBoxRewardEntry[]) => void;
}

interface RewardEditorFormValues {
  type: GiftBoxRewardEntry['type'];
  item_id?: number;
  item_name?: string;
  count?: number;
  value?: number;
  pet_id?: number;
  pet_name?: string;
}

/** 礼包内容可视化编辑器：维护开启后发放的物品、铜币与宠物奖励。 */
export function GiftBoxRewardEditor({ value = [], onChange }: GiftBoxRewardEditorProps) {
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [preferredItemID, setPreferredItemID] = useState<number | undefined>(undefined);
  const [preferredItemName, setPreferredItemName] = useState('');
  const [petTemplates, setPetTemplates] = useState<AdminPetDefinitionSummary[]>([]);
  const [petTemplatesLoading, setPetTemplatesLoading] = useState(false);
  const [editorForm] = Form.useForm<RewardEditorFormValues>();
  const rewardType = Form.useWatch('type', editorForm);

  useEffect(() => {
    if (!editorOpen) {
      return;
    }
    let cancelled = false;
    setPetTemplatesLoading(true);
    fetchAllEnabledAdminPetDefinitions()
      .then((items) => {
        if (!cancelled) {
          setPetTemplates(items);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setPetTemplatesLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [editorOpen]);

  const petSelectOptions = useMemo(
    () => petTemplates.map((item) => ({
      value: item.pet_id,
      label: `#${item.pet_id} ${item.pet_name} · ${formatPetQualityLabel(item.quality)} · Lv.${item.level}`,
    })),
    [petTemplates],
  );

  const columns = useMemo<ColumnsType<GiftBoxRewardEntry>>(
    () => [
      {
        title: '奖励类型',
        dataIndex: 'type',
        key: 'type',
        width: 120,
        render: (type: GiftBoxRewardEntry['type']) => {
          if (type === 'item') {
            return <Tag color="blue">物品</Tag>;
          }
          if (type === 'pet') {
            return <Tag color="purple">宠物</Tag>;
          }
          return <Tag color="gold">金币</Tag>;
        },
      },
      {
        title: '内容',
        key: 'content',
        render: (_value, record) => formatGiftBoxRewardRowSummary(record),
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
        pet_id: undefined,
        pet_name: '',
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
        pet_id: current.pet_id,
        pet_name: current.pet_name,
      });
    }
    setEditorOpen(true);
  }

  function removeEntry(index: number) {
    emitChange(value.filter((_entry, entryIndex) => entryIndex !== index));
  }

  function handleSubmitEditor(formValues: RewardEditorFormValues) {
    let nextEntry: GiftBoxRewardEntry;
    if (formValues.type === 'gold') {
      nextEntry = { type: 'gold', value: Number(formValues.value ?? 0) };
    } else if (formValues.type === 'pet') {
      nextEntry = {
        type: 'pet',
        pet_id: Number(formValues.pet_id ?? 0),
        pet_name: formValues.pet_name ?? '',
      };
    } else {
      nextEntry = {
        type: 'item',
        item_id: Number(formValues.item_id ?? 0),
        item_name: formValues.item_name ?? '',
        count: Number(formValues.count ?? 1),
      };
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
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Text type="secondary">配置打开礼包后固定获得的奖励内容（物品、铜币或宠物）。</Typography.Text>
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
          ) : null}
          {rewardType === 'gold' ? (
            <Form.Item label="铜币数量" name="value" rules={[{ required: true, message: '请输入铜币数量' }]}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          ) : null}
          {rewardType === 'pet' ? (
            <>
              <Form.Item label="宠物模板" name="pet_id" rules={[{ required: true, message: '请选择宠物模板' }]}>
                <Select
                  showSearch
                  loading={petTemplatesLoading}
                  options={petSelectOptions}
                  optionFilterProp="label"
                  placeholder="搜索并选择系统宠物模板"
                  onChange={(petID: number) => {
                    const selected = petTemplates.find((item) => item.pet_id === petID);
                    editorForm.setFieldValue('pet_name', selected?.pet_name ?? '');
                  }}
                />
              </Form.Item>
              <Form.Item name="pet_name" hidden>
                <Input />
              </Form.Item>
            </>
          ) : null}
        </Form>
      </Modal>
    </Space>
  );
}

/** 表格行内展示单条礼包奖励摘要。 */
export function formatGiftBoxRewardRowSummary(record: GiftBoxRewardEntry): string {
  if (record.type === 'gold') {
    return `铜币 ${record.value ?? 0}`;
  }
  if (record.type === 'pet') {
    const petName = record.pet_name?.trim();
    return petName ? `宠物 ${petName}` : `宠物ID ${record.pet_id ?? 0}`;
  }
  const itemName = record.item_name?.trim();
  const nameText = itemName ? itemName : `物品ID ${record.item_id ?? 0}`;
  return `${nameText} × ${record.count ?? 1}`;
}
