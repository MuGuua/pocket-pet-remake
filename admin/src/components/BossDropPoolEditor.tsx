import { Button, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import type { AdminBossDropEntry } from '../services/sceneWildEncounter';
import { GrantableItemSelect } from './GrantableItemSelect';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../utils/modalLayout';

interface BossDropPoolEditorProps {
  value?: AdminBossDropEntry[];
  onChange?: (nextValue: AdminBossDropEntry[]) => void;
}

interface BossDropFormValues {
  item_id?: number;
  item_name?: string;
  quantity?: number;
  weight?: number;
  draw_group?: number;
  grant_once?: number;
  sort_order?: number;
  status?: number;
}

const GRANT_ONCE_OPTIONS = [
  { label: '否', value: 0 },
  { label: '是', value: 1 },
];

const STATUS_OPTIONS = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

/**
 * Boss 掉落池编辑器：按「抽取组」配置候选掉落，每组胜利时按权重抽 1 个。
 * 与固定奖励分开维护，固定奖励每次都发，这里的条目按概率掉落。
 */
export function BossDropPoolEditor({ value = [], onChange }: BossDropPoolEditorProps) {
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [preferredItemID, setPreferredItemID] = useState<number | undefined>(undefined);
  const [preferredItemName, setPreferredItemName] = useState('');
  const [editorForm] = Form.useForm<BossDropFormValues>();

  const enabledCount = useMemo(
    () => value.filter((entry) => Number(entry.status ?? 1) === 1).length,
    [value],
  );
  const groupCount = useMemo(
    () => new Set(value.filter((entry) => Number(entry.status ?? 1) === 1).map((entry) => Number(entry.draw_group ?? 1))).size,
    [value],
  );

  const columns = useMemo<ColumnsType<AdminBossDropEntry>>(
    () => [
      {
        title: '抽取组',
        dataIndex: 'draw_group',
        key: 'draw_group',
        width: 88,
        render: (group: number | undefined) => <Tag color="blue">组 {Number(group ?? 1)}</Tag>,
      },
      {
        title: '物品',
        dataIndex: 'item_name',
        key: 'item_name',
        render: (itemName: string | undefined, record) => itemName?.trim() || `物品 ${record.item_id}`,
      },
      {
        title: '数量',
        dataIndex: 'quantity',
        key: 'quantity',
        width: 80,
      },
      {
        title: '权重',
        dataIndex: 'weight',
        key: 'weight',
        width: 88,
        render: (weight: number | undefined, record) => {
          const group = Number(record.draw_group ?? 1);
          const groupTotal = value
            .filter((entry) => Number(entry.status ?? 1) === 1 && Number(entry.draw_group ?? 1) === group)
            .reduce((sum, entry) => sum + Math.max(0, Number(entry.weight ?? 0)), 0);
          const current = Math.max(0, Number(weight ?? 0));
          if (groupTotal <= 0) {
            return current;
          }
          const percent = ((current / groupTotal) * 100).toFixed(1);
          return `${current} (${percent}%)`;
        },
      },
      {
        title: '唯一掉落',
        dataIndex: 'grant_once',
        key: 'grant_once',
        width: 92,
        render: (grantOnce: number | undefined) => (Number(grantOnce ?? 0) > 0 ? '是' : '否'),
      },
      {
        title: '排序',
        dataIndex: 'sort_order',
        key: 'sort_order',
        width: 72,
        render: (sortOrder: number, _record, index) => (sortOrder > 0 ? sortOrder : index + 1),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 88,
        render: (status: number | undefined) =>
          Number(status ?? 1) === 1 ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
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

  function emitChange(nextValue: AdminBossDropEntry[]) {
    onChange?.(nextValue);
  }

  function openEditor(index: number | null) {
    setEditingIndex(index);
    if (index === null) {
      setPreferredItemID(undefined);
      setPreferredItemName('');
      editorForm.setFieldsValue({
        quantity: 1,
        weight: 100,
        draw_group: 1,
        grant_once: 0,
        sort_order: value.length + 1,
        status: 1,
      });
    } else {
      const current = value[index];
      setPreferredItemID(current.item_id > 0 ? current.item_id : undefined);
      setPreferredItemName(current.item_name?.trim() ?? '');
      editorForm.setFieldsValue({
        item_id: current.item_id > 0 ? current.item_id : undefined,
        item_name: current.item_name ?? '',
        quantity: current.quantity > 0 ? current.quantity : 1,
        weight: current.weight > 0 ? current.weight : 100,
        draw_group: current.draw_group > 0 ? current.draw_group : 1,
        grant_once: Number(current.grant_once ?? 0) > 0 ? 1 : 0,
        sort_order: current.sort_order > 0 ? current.sort_order : index + 1,
        status: current.status ?? 1,
      });
    }
    setEditorOpen(true);
  }

  function removeEntry(index: number) {
    emitChange(value.filter((_entry, entryIndex) => entryIndex !== index));
  }

  function handleSubmitEditor(formValues: BossDropFormValues) {
    const itemID = Number(formValues.item_id ?? 0);
    if (itemID <= 0) {
      return;
    }
    const nextEntry: AdminBossDropEntry = {
      item_id: itemID,
      item_name: formValues.item_name?.trim() ?? '',
      quantity: Math.max(1, Number(formValues.quantity ?? 1)),
      weight: Math.max(1, Number(formValues.weight ?? 100)),
      draw_group: Math.max(1, Number(formValues.draw_group ?? 1)),
      grant_once: Number(formValues.grant_once ?? 0) > 0 ? 1 : 0,
      sort_order: Number(formValues.sort_order ?? 0) > 0 ? Number(formValues.sort_order) : value.length + 1,
      status: Number(formValues.status ?? 1) === 1 ? 1 : 0,
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
      <Space style={{ width: '100%', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <Typography.Text type="secondary">
          共 {value.length} 条掉落，启用 {enabledCount} 条，{groupCount} 个抽取组。
          同一抽取组内按权重随机抽 1 个，不同组互不影响；固定奖励请在上方奖励列表配置。
        </Typography.Text>
        <Button type="primary" onClick={() => openEditor(null)}>添加掉落</Button>
      </Space>
      <Table
        size="small"
        rowKey={(_record, index) => String(index)}
        columns={columns}
        dataSource={value}
        pagination={false}
        scroll={{ x: 760 }}
        locale={{ emptyText: '尚未配置 Boss 掉落池，该遭遇战只发固定奖励。' }}
      />
      <Modal
        title={editingIndex === null ? '添加 Boss 掉落' : '编辑 Boss 掉落'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingIndex(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        destroyOnClose
        width={560}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText="确定"
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(formValues) => handleSubmitEditor(formValues)}>
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
          <Space size={16} style={{ width: '100%' }}>
            <Form.Item label="数量" name="quantity" style={{ flex: 1 }} rules={[{ required: true, message: '请输入数量' }]}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item
              label="权重"
              name="weight"
              style={{ flex: 1 }}
              tooltip="同组内按权重随机抽取，权重越大越容易命中。"
              rules={[{ required: true, message: '请输入权重' }]}
            >
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
          </Space>
          <Space size={16} style={{ width: '100%' }}>
            <Form.Item
              label="抽取组"
              name="draw_group"
              style={{ flex: 1 }}
              tooltip="同组只抽 1 个；不同组可以同时命中，用于配置「必掉一件武器 + 必掉一件材料」。"
              rules={[{ required: true, message: '请输入抽取组' }]}
            >
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item
              label="唯一掉落"
              name="grant_once"
              style={{ flex: 1 }}
              tooltip="开启后每名玩家仅首次获得该物品。"
            >
              <Select options={GRANT_ONCE_OPTIONS} />
            </Form.Item>
          </Space>
          <Space size={16} style={{ width: '100%' }}>
            <Form.Item label="排序" name="sort_order" style={{ flex: 1 }}>
              <InputNumber min={1} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item label="状态" name="status" style={{ flex: 1 }}>
              <Select options={STATUS_OPTIONS} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </Space>
  );
}
