import {
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import {
  createAdminBagItem,
  deleteAdminBagItem,
  fetchAdminBagDetail,
  fetchAdminBags,
  updateAdminBagItem,
} from '../../services/bag';
import type {
  AdminBagDetail,
  AdminBagSummary,
  AdminCreateBagPayload,
  AdminUpdateBagPayload,
} from '../../types/bag';

interface BagFormValues extends AdminCreateBagPayload {}

interface PlayerBagSectionProps {
  playerId: number;
  playerName: string;
}

const containerTypeOptions = [
  { label: '全部容器', value: '' },
  { label: '背包', value: 'bag' },
  { label: '仓库', value: 'warehouse' },
];

const editableContainerTypeOptions = [
  { label: '背包', value: 'bag' },
  { label: '仓库', value: 'warehouse' },
];

// 玩家详情/编辑页内的背包区块：按 player_id 拉取容器格子，并支持查看、编辑、新增与删除。
export function PlayerBagSection({ playerId, playerName }: PlayerBagSectionProps) {
  const [editorForm] = Form.useForm<BagFormValues>();
  const [containerType, setContainerType] = useState('');
  const [rows, setRows] = useState<AdminBagSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [bagDetailOpen, setBagDetailOpen] = useState(false);
  const [bagDetailLoading, setBagDetailLoading] = useState(false);
  const [bagDetail, setBagDetail] = useState<AdminBagDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminBagDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);

  useEffect(() => {
    void loadBags();
  }, [playerId, containerType]);

  async function loadBags() {
    setLoading(true);
    try {
      const result = await fetchAdminBags({
        filters: {
          player_id: String(playerId),
          container_type: containerType || undefined,
        },
        page: 1,
        pageSize: 200,
      });
      setRows(result.items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家背包失败');
      setRows([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(recordID: number) {
    setBagDetailOpen(true);
    setBagDetailLoading(true);
    setBagDetail(null);
    try {
      setBagDetail(await fetchAdminBagDetail(recordID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载容器详情失败');
      setBagDetailOpen(false);
    } finally {
      setBagDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', recordID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultCreateValues(playerId));
      return;
    }
    if (!recordID) {
      return;
    }
    setBagDetailLoading(true);
    try {
      const result = await fetchAdminBagDetail(recordID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载容器编辑数据失败');
      setEditorOpen(false);
    } finally {
      setBagDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: BagFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminBagItem(editingRecord.record_id, mapFormToUpdatePayload(values));
        message.success('容器记录更新成功');
      } else {
        await createAdminBagItem(mapFormToCreatePayload(values));
        message.success('容器记录创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadBags();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存容器记录失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(recordID: number) {
    setDeletingID(recordID);
    try {
      await deleteAdminBagItem(recordID);
      message.success('容器记录已删除');
      if (bagDetail?.record_id === recordID) {
        setBagDetailOpen(false);
        setBagDetail(null);
      }
      await loadBags();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除容器记录失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminBagSummary>>(
    () => [
      { title: '记录ID', dataIndex: 'record_id', key: 'record_id', width: 90 },
      { title: '容器', dataIndex: 'container_type', key: 'container_type', width: 90 },
      { title: '格子', dataIndex: 'slot_index', key: 'slot_index', width: 70 },
      { title: '物品ID', dataIndex: 'item_id', key: 'item_id', width: 90 },
      { title: '物品名', dataIndex: 'item_name', key: 'item_name', width: 140 },
      { title: '实例ID', dataIndex: 'item_uid', key: 'item_uid', width: 120, render: (value: string) => value || '-' },
      { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 70 },
      { title: '绑定', dataIndex: 'is_bound', key: 'is_bound', width: 70, render: (value: boolean) => (value ? '是' : '否') },
      {
        title: '操作',
        key: 'actions',
        width: 160,
        render: (_value, record) => (
          <Space size="small" onClick={(event) => event.stopPropagation()}>
            <Button type="link" onClick={() => void handleViewDetail(record.record_id)}>查看</Button>
            <Button type="link" onClick={() => void handleOpenEditor('edit', record.record_id)}>编辑</Button>
            <Popconfirm title="确认删除这条容器记录吗？" onConfirm={() => void handleDelete(record.record_id)}>
              <Button type="link" danger loading={deletingID === record.record_id}>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [deletingID],
  );

  return (
    <>
      <Card
        size="small"
        title="背包信息"
        extra={(
          <Space>
            <Select
              value={containerType}
              options={containerTypeOptions}
              style={{ width: 110 }}
              onChange={(value) => setContainerType(value)}
            />
            <Button type="primary" size="small" onClick={() => void handleOpenEditor('create')}>
              新增记录
            </Button>
          </Space>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="record_id"
          loading={loading}
          size="small"
          locale={{ emptyText: <Empty description={`${playerName} 还没有容器物品`} /> }}
          scroll={{ x: 1100 }}
          pagination={false}
          onRow={(record) => ({
            onClick: () => void handleViewDetail(record.record_id),
            style: { cursor: 'pointer' },
          })}
        />
        <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
          点击表格行可快速查看容器详情。
        </Typography.Text>
      </Card>

      <Drawer
        title={bagDetail ? `容器详情 · ${bagDetail.record_id}` : '容器详情'}
        width={520}
        open={bagDetailOpen}
        onClose={() => setBagDetailOpen(false)}
        destroyOnClose
        extra={bagDetail ? (
          <Space>
            <Button onClick={() => void handleOpenEditor('edit', bagDetail.record_id)}>编辑</Button>
            <Popconfirm title="确认删除这条容器记录吗？" onConfirm={() => void handleDelete(bagDetail.record_id)}>
              <Button danger loading={deletingID === bagDetail.record_id}>删除</Button>
            </Popconfirm>
          </Space>
        ) : null}
      >
        {bagDetailLoading ? (
          <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载容器详情..." />
          </div>
        ) : bagDetail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="记录ID">{bagDetail.record_id}</Descriptions.Item>
            <Descriptions.Item label="玩家ID">{bagDetail.player_id}</Descriptions.Item>
            <Descriptions.Item label="玩家名">{bagDetail.player_name}</Descriptions.Item>
            <Descriptions.Item label="容器类型">{bagDetail.container_type}</Descriptions.Item>
            <Descriptions.Item label="格子号">{bagDetail.slot_index}</Descriptions.Item>
            <Descriptions.Item label="物品ID">{bagDetail.item_id}</Descriptions.Item>
            <Descriptions.Item label="物品名">{bagDetail.item_name || '-'}</Descriptions.Item>
            <Descriptions.Item label="实例ID">{bagDetail.item_uid || '-'}</Descriptions.Item>
            <Descriptions.Item label="物品类型">{bagDetail.item_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="数量">{bagDetail.quantity}</Descriptions.Item>
            <Descriptions.Item label="绑定">{bagDetail.is_bound ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{formatDateTime(bagDetail.updated_at)}</Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑容器记录 · ${editingRecord.record_id}` : `新增容器记录 · ${playerName}`}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingRecord(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={600}
        okText={editingRecord ? '保存修改' : '创建记录'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            {!editingRecord ? (
              <Col xs={24} md={12}>
                <Form.Item label="玩家ID" name="player_id">
                  <InputNumber min={1} style={{ width: '100%' }} disabled />
                </Form.Item>
              </Col>
            ) : null}
            <Col xs={24} md={12}>
              <Form.Item label="容器类型" name="container_type" rules={[{ required: true, message: '请选择容器类型' }]}>
                <Select options={editableContainerTypeOptions} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="格子号" name="slot_index" rules={[{ required: true, message: '请输入格子号' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="物品ID" name="item_id" rules={[{ required: true, message: '请输入物品ID' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="实例ID" name="item_uid">
                <Input allowClear placeholder="装备等实例物品再填写" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="数量" name="quantity" rules={[{ required: true, message: '请输入数量' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item label="绑定状态" name="is_bound" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </>
  );
}

function defaultCreateValues(playerId: number): BagFormValues {
  return {
    player_id: playerId,
    container_type: 'bag',
    slot_index: 1,
    item_id: 1001,
    item_uid: '',
    quantity: 1,
    is_bound: false,
  };
}

function mapDetailToForm(detail: AdminBagDetail): BagFormValues {
  return {
    player_id: detail.player_id,
    container_type: detail.container_type,
    slot_index: detail.slot_index,
    item_id: detail.item_id,
    item_uid: detail.item_uid,
    quantity: detail.quantity,
    is_bound: detail.is_bound,
  };
}

function mapFormToCreatePayload(values: BagFormValues): AdminCreateBagPayload {
  return { ...values };
}

function mapFormToUpdatePayload(values: BagFormValues): AdminUpdateBagPayload {
  return { ...values };
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', { hour12: false });
}
