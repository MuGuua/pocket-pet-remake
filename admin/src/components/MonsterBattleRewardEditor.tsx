import { Button, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useMemo, useState } from 'react';
import type { AdminMonsterBattleRewardEntry, AdminMonsterBattleRewardSaveEntry } from '../types/monsterDefinition';
import { GrantableItemSelect } from './GrantableItemSelect';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../utils/modalLayout';

interface MonsterBattleRewardEditorProps {
  value?: AdminMonsterBattleRewardEntry[];
  onChange?: (nextValue: AdminMonsterBattleRewardEntry[]) => void;
}

interface RewardEditorFormValues {
  reward_type: AdminMonsterBattleRewardEntry['reward_type'];
  exp_target?: AdminMonsterBattleRewardEntry['exp_target'];
  exp_value?: number;
  item_id?: number;
  item_name?: string;
  quantity?: number;
  grant_once?: number;
  sort_order?: number;
  status?: number;
}

const REWARD_TYPE_OPTIONS = [
  { label: '经验', value: 'exp' as const },
  { label: '铜币', value: 'gold' as const },
  { label: '物品', value: 'item' as const },
];

const EXP_TARGET_OPTIONS = [
  { label: '角色', value: 'player' as const },
  { label: '宠物', value: 'pet' as const },
];

const STATUS_OPTIONS = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

const GRANT_ONCE_OPTIONS = [
  { label: '否', value: 0 },
  { label: '是', value: 1 },
];

/** 系统怪物战斗奖励可视化编辑器：表格预览 + 弹窗编辑，物品奖励走可搜索下拉。 */
export function MonsterBattleRewardEditor({ value = [], onChange }: MonsterBattleRewardEditorProps) {
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [preferredItemID, setPreferredItemID] = useState<number | undefined>(undefined);
  const [preferredItemName, setPreferredItemName] = useState('');
  const [editorForm] = Form.useForm<RewardEditorFormValues>();
  const rewardType = Form.useWatch('reward_type', editorForm);

  const enabledCount = useMemo(
    () => value.filter((entry) => Number(entry.status ?? 1) === 1).length,
    [value],
  );

  const columns = useMemo<ColumnsType<AdminMonsterBattleRewardEntry>>(
    () => [
      {
        title: '排序',
        dataIndex: 'sort_order',
        key: 'sort_order',
        width: 72,
        render: (sortOrder: number, _record, index) => sortOrder > 0 ? sortOrder : index + 1,
      },
      {
        title: '类型',
        dataIndex: 'reward_type',
        key: 'reward_type',
        width: 88,
        render: (type: AdminMonsterBattleRewardEntry['reward_type']) => {
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
        render: (_value, record) => formatMonsterBattleRewardSummary(record),
      },
      {
        title: '唯一掉落',
        dataIndex: 'grant_once',
        key: 'grant_once',
        width: 92,
        render: (grantOnce: number | undefined, record) =>
          record.reward_type === 'item' ? (Number(grantOnce ?? 0) > 0 ? '是' : '否') : '-',
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

  function emitChange(nextValue: AdminMonsterBattleRewardEntry[]) {
    onChange?.(nextValue);
  }

  function openEditor(index: number | null) {
    setEditingIndex(index);
    if (index === null) {
      setPreferredItemID(undefined);
      setPreferredItemName('');
      editorForm.setFieldsValue(createDefaultMonsterBattleRewardRow(value.length + 1));
    } else {
      const current = value[index];
      setPreferredItemID(current.item_id > 0 ? current.item_id : undefined);
      setPreferredItemName(current.item_name?.trim() ?? '');
      editorForm.setFieldsValue({
        reward_type: current.reward_type,
        exp_target: current.exp_target || 'player',
        exp_value: current.exp_value,
        item_id: current.item_id > 0 ? current.item_id : undefined,
        item_name: current.item_name ?? '',
        quantity: current.quantity > 0 ? current.quantity : 1,
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

  function handleSubmitEditor(formValues: RewardEditorFormValues) {
    const nextEntry = normalizeMonsterBattleRewardEntry(formValues, editingIndex ?? value.length);
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
          共 {value.length} 条奖励，启用 {enabledCount} 条。胜利后按排序依次发放；物品支持唯一掉落。
        </Typography.Text>
        <Button type="primary" onClick={() => openEditor(null)}>添加奖励</Button>
      </Space>
      <Table
        size="small"
        rowKey={(_record, index) => String(index)}
        columns={columns}
        dataSource={value}
        pagination={false}
        scroll={{ x: 760 }}
        locale={{ emptyText: '尚未配置战斗奖励，请点击「添加奖励」。' }}
      />
      <Modal
        title={editingIndex === null ? '添加战斗奖励' : '编辑战斗奖励'}
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
          <Form.Item label="奖励类型" name="reward_type" rules={[{ required: true, message: '请选择奖励类型' }]}>
            <Select options={REWARD_TYPE_OPTIONS} />
          </Form.Item>
          {rewardType === 'exp' ? (
            <>
              <Form.Item label="经验目标" name="exp_target" rules={[{ required: true, message: '请选择经验目标' }]}>
                <Select options={EXP_TARGET_OPTIONS} />
              </Form.Item>
              <Form.Item label="经验值" name="exp_value" rules={[{ required: true, message: '请输入经验值' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </>
          ) : null}
          {rewardType === 'gold' ? (
            <Form.Item label="铜币数量" name="exp_value" rules={[{ required: true, message: '请输入铜币数量' }]}>
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
                  onItemChange={(item) => {
                    editorForm.setFieldValue('item_name', item?.item_name ?? '');
                  }}
                />
              </Form.Item>
              <Form.Item name="item_name" hidden>
                <Input />
              </Form.Item>
              <Form.Item label="数量" name="quantity" rules={[{ required: true, message: '请输入数量' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                label="唯一掉落"
                name="grant_once"
                tooltip="开启后每名玩家仅首次获得该物品，之后战斗不再重复发放。"
              >
                <Select options={GRANT_ONCE_OPTIONS} />
              </Form.Item>
            </>
          ) : null}
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

/** 将表单值整理为服务端可保存的怪物战斗奖励条目。 */
export function normalizeMonsterBattleRewardEntry(
  formValues: RewardEditorFormValues,
  fallbackSortOrder: number,
): AdminMonsterBattleRewardEntry {
  const sortOrder = Number(formValues.sort_order ?? 0) > 0 ? Number(formValues.sort_order) : fallbackSortOrder + 1;
  const status = Number(formValues.status ?? 1);
  if (formValues.reward_type === 'item') {
    return {
      reward_type: 'item',
      exp_target: 'player',
      item_id: Number(formValues.item_id ?? 0),
      item_name: String(formValues.item_name ?? '').trim(),
      quantity: Number(formValues.quantity ?? 0),
      exp_value: 0,
      sort_order: sortOrder,
      status,
      grant_once: Number(formValues.grant_once ?? 0) > 0 ? 1 : 0,
    };
  }
  if (formValues.reward_type === 'gold') {
    return {
      reward_type: 'gold',
      exp_target: 'player',
      item_id: 0,
      quantity: 0,
      exp_value: Number(formValues.exp_value ?? 0),
      sort_order: sortOrder,
      status,
    };
  }
  return {
    reward_type: 'exp',
    exp_target: formValues.exp_target || 'player',
    item_id: 0,
    quantity: 0,
    exp_value: Number(formValues.exp_value ?? 0),
    sort_order: sortOrder,
    status,
  };
}

/** 新建一条默认怪物战斗奖励条目。 */
export function createDefaultMonsterBattleRewardEntry(): AdminMonsterBattleRewardEntry {
  return normalizeMonsterBattleRewardEntry(createDefaultMonsterBattleRewardRow(1), 0);
}

/** 新建一条默认怪物战斗奖励表单值。 */
export function createDefaultMonsterBattleRewardRow(nextSortOrder: number): RewardEditorFormValues {
  return {
    reward_type: 'exp',
    exp_target: 'player',
    exp_value: 10,
    item_id: undefined,
    quantity: 1,
    grant_once: 0,
    sort_order: nextSortOrder,
    status: 1,
  };
}

/** 提交前再次规范化列表，并剔除服务端不接受的展示字段（如 item_name）。 */
export function mapMonsterBattleRewardPayload(
  item: AdminMonsterBattleRewardEntry,
  index: number,
): AdminMonsterBattleRewardSaveEntry {
  const normalized = normalizeMonsterBattleRewardEntry(
    {
      reward_type: item.reward_type,
      exp_target: item.exp_target,
      exp_value: item.exp_value,
      item_id: item.item_id > 0 ? item.item_id : undefined,
      item_name: item.item_name,
      quantity: item.quantity > 0 ? item.quantity : undefined,
      grant_once: item.grant_once,
      sort_order: item.sort_order,
      status: item.status,
    },
    index,
  );
  return {
    reward_type: normalized.reward_type,
    exp_target: normalized.exp_target,
    item_id: normalized.item_id,
    quantity: normalized.quantity,
    exp_value: normalized.exp_value,
    sort_order: normalized.sort_order,
    status: normalized.status,
    grant_once: normalized.grant_once,
  };
}

function formatMonsterBattleRewardSummary(record: AdminMonsterBattleRewardEntry): string {
  if (record.reward_type === 'item') {
    const itemID = Number(record.item_id ?? 0);
    const quantity = Number(record.quantity ?? 0);
    if (itemID <= 0) {
      return '未选择物品';
    }
    const itemName = record.item_name?.trim();
    const nameText = itemName ? itemName : `物品ID ${itemID}`;
    return `${nameText} × ${quantity}`;
  }
  if (record.reward_type === 'gold') {
    return `铜币 ${Number(record.exp_value ?? 0)}`;
  }
  const targetLabel = record.exp_target === 'pet' ? '宠物' : '角色';
  return `${targetLabel}经验 ${Number(record.exp_value ?? 0)}`;
}
