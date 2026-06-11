import {
  Alert,
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
  Space,
  Spin,
  Statistic,
  Table,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { createAdminBagItem, deleteAdminBagItem, fetchAdminBagDetail, fetchAdminBags, updateAdminBagItem } from '../../services/bag';
import type { AdminBagDetail, AdminBagListFilters, AdminBagSummary, AdminCreateBagPayload, AdminUpdateBagPayload } from '../../types/bag';

interface BagFormValues {
  player_id: number;
  item_id: number;
  count: number;
}

// 背包管理页保持和玩家/宠物页一致的 CRUD 结构，方便后续继续复制到任务和配置模块。
export function BagListPage() {
  const [filterForm] = Form.useForm<AdminBagListFilters>();
  const [editorForm] = Form.useForm<BagFormValues>();
  const [filters, setFilters] = useState<AdminBagListFilters>({});
  const [rows, setRows] = useState<AdminBagSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminBagDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminBagDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);

  useEffect(() => {
    void loadBags(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadBags(nextFilters: AdminBagListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminBags({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载背包列表失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(recordID: number) {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      setDetail(await fetchAdminBagDetail(recordID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载背包详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', recordID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultCreateValues());
      return;
    }
    if (!recordID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminBagDetail(recordID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载背包编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: BagFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminBagItem(editingRecord.record_id, mapFormToUpdatePayload(values));
        message.success('背包记录更新成功');
      } else {
        await createAdminBagItem(mapFormToCreatePayload(values));
        message.success('背包记录创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadBags(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存背包记录失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(recordID: number) {
    setDeletingID(recordID);
    try {
      await deleteAdminBagItem(recordID);
      message.success('背包记录已删除');
      if (detail?.record_id === recordID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadBags(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除背包记录失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminBagSummary>>(
    () => [
      { title: '记录ID', dataIndex: 'record_id', key: 'record_id', width: 110, fixed: 'left' },
      { title: '玩家ID', dataIndex: 'player_id', key: 'player_id', width: 110 },
      { title: '玩家名', dataIndex: 'player_name', key: 'player_name', width: 150 },
      { title: '道具ID', dataIndex: 'item_id', key: 'item_id', width: 110 },
      { title: '数量', dataIndex: 'count', key: 'count', width: 100 },
      { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 180, render: (value: string) => formatDateTime(value) },
      {
        title: '操作',
        key: 'actions',
        width: 220,
        fixed: 'right',
        render: (_value, record) => (
          <Space size="small">
            <Button type="link" onClick={() => void handleViewDetail(record.record_id)}>查看</Button>
            <Button type="link" onClick={() => void handleOpenEditor('edit', record.record_id)}>编辑</Button>
            <Popconfirm title="确认删除这条背包记录吗？" onConfirm={() => void handleDelete(record.record_id)} okText="确认删除" cancelText="取消">
              <Button type="link" danger loading={deletingID === record.record_id}>删除</Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [deletingID],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="info" showIcon message="背包管理已接入真实服务端 /api/admin/bags CRUD 接口，所有改动直接持久化到 player_item。" />
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}><Card><Statistic title="当前页记录数" value={rows.length} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="当前页道具总数" value={rows.reduce((sum, item) => sum + item.count, 0)} valueStyle={{ color: '#2f7d4a' }} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="总记录数" value={total} /></Card></Col>
      </Row>
      <Card title="背包筛选" extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增记录</Button>}>
        <Form form={filterForm} layout="vertical" onFinish={(values) => { setPage(1); setFilters(values); }}>
          <Row gutter={16}>
            <Col xs={24} md={8}><Form.Item label="记录ID" name="record_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="玩家ID" name="player_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="道具ID" name="item_id"><Input allowClear /></Form.Item></Col>
          </Row>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
            <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
          </Space>
        </Form>
      </Card>
      <Card title="背包列表" extra={<Typography.Text type="secondary">支持按玩家与道具检索，并进行增删改查。</Typography.Text>}>
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="record_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有背包数据" /> }}
          scroll={{ x: 1200 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条记录`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </Card>
      <Drawer title={detail ? `背包详情 · ${detail.record_id}` : '背包详情'} width={520} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading ? <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}><Spin tip="正在加载背包详情..." /></div> : detail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="记录ID">{detail.record_id}</Descriptions.Item>
            <Descriptions.Item label="玩家ID">{detail.player_id}</Descriptions.Item>
            <Descriptions.Item label="玩家名">{detail.player_name}</Descriptions.Item>
            <Descriptions.Item label="道具ID">{detail.item_id}</Descriptions.Item>
            <Descriptions.Item label="数量">{detail.count}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDateTime(detail.created_at)}</Descriptions.Item>
            <Descriptions.Item label="更新时间" span={2}>{formatDateTime(detail.updated_at)}</Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>
      <Modal title={editingRecord ? `编辑背包记录 · ${editingRecord.record_id}` : '新增背包记录'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={560} okText={editingRecord ? '保存修改' : '创建记录'} cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            <Col xs={24} md={12}><Form.Item label="玩家ID" name="player_id" rules={[{ required: true, message: '请输入玩家ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item label="道具ID" name="item_id" rules={[{ required: true, message: '请输入道具ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}><Form.Item label="数量" name="count" rules={[{ required: true, message: '请输入道具数量' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultCreateValues(): BagFormValues {
  return { player_id: 10001, item_id: 2001, count: 1 };
}

function mapDetailToForm(detail: AdminBagDetail): BagFormValues {
  return { player_id: detail.player_id, item_id: detail.item_id, count: detail.count };
}

function mapFormToCreatePayload(values: BagFormValues): AdminCreateBagPayload {
  return { player_id: values.player_id, item_id: values.item_id, count: values.count };
}

function mapFormToUpdatePayload(values: BagFormValues): AdminUpdateBagPayload {
  return { player_id: values.player_id, item_id: values.item_id, count: values.count };
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', { hour12: false });
}
