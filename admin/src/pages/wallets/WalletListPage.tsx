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
  Space,
  Table,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { adjustAdminWallet, fetchAdminWalletDetail, fetchAdminWallets } from '../../services/wallet';
import type { AdminAdjustWalletPayload, AdminWalletDetail, AdminWalletListFilters, AdminWalletSummary } from '../../types/wallet';

interface WalletFormValues extends AdminAdjustWalletPayload {}

// 钱包页统一按总铜币做真值调整，展示层再拆成金银铜，避免运营误改展示态字段。
export function WalletListPage() {
  const [filterForm] = Form.useForm<AdminWalletListFilters>();
  const [editorForm] = Form.useForm<WalletFormValues>();
  const [filters, setFilters] = useState<AdminWalletListFilters>({});
  const [rows, setRows] = useState<AdminWalletSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminWalletDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingPlayerID, setEditingPlayerID] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void loadWallets(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadWallets(nextFilters: AdminWalletListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminWallets({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载钱包列表失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(playerID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminWalletDetail(playerID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载钱包详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  function handleOpenEditor(playerID: number) {
    setEditingPlayerID(playerID);
    editorForm.setFieldsValue({ change_total_copper: 1000, reason: '运营手动调整' });
    setEditorOpen(true);
  }

  async function handleSubmit(values: WalletFormValues) {
    if (!editingPlayerID) return;
    setSaving(true);
    try {
      await adjustAdminWallet(editingPlayerID, values);
      message.success('钱包调整成功');
      setEditorOpen(false);
      if (detail?.player_id === editingPlayerID) {
        setDetail(await fetchAdminWalletDetail(editingPlayerID));
      }
      await loadWallets(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '钱包调整失败');
    } finally {
      setSaving(false);
    }
  }

  const columns = useMemo<ColumnsType<AdminWalletSummary>>(
    () => [
      { title: '玩家ID', dataIndex: 'player_id', key: 'player_id', width: 110 },
      { title: '玩家名', dataIndex: 'player_name', key: 'player_name', width: 150 },
      { title: '总铜币', dataIndex: ['wallet', 'total_copper'], key: 'total_copper', width: 140 },
      { title: '金币', dataIndex: ['wallet', 'gold'], key: 'gold', width: 100 },
      { title: '银币', dataIndex: ['wallet', 'silver'], key: 'silver', width: 100 },
      { title: '铜币', dataIndex: ['wallet', 'copper'], key: 'copper', width: 100 },
      {
        title: '操作', key: 'actions', width: 180, render: (_value, record) => (
          <Space size="small">
            <Button type="link" onClick={() => void handleViewDetail(record.player_id)}>查看</Button>
            <Button type="link" onClick={() => handleOpenEditor(record.player_id)}>调账</Button>
          </Space>
        ),
      },
    ],
    [],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title="钱包列表"
        extra={(
          <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }}>
            <Form.Item name="player_id" label="玩家ID">
              <Input allowClear placeholder="玩家ID" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="keyword" label="玩家名">
              <Input allowClear placeholder="玩家名" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="player_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有钱包数据" /> }}
          scroll={{ x: 1000 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条钱包记录`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </Card>
      <Drawer title={detail ? `钱包详情 · ${detail.player_name}` : '钱包详情'} width={520} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? <Typography.Text type="secondary">正在加载钱包详情...</Typography.Text> : (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="玩家ID">{detail.player_id}</Descriptions.Item>
            <Descriptions.Item label="玩家名">{detail.player_name}</Descriptions.Item>
            <Descriptions.Item label="总铜币">{detail.wallet.total_copper}</Descriptions.Item>
            <Descriptions.Item label="版本">{detail.version}</Descriptions.Item>
            <Descriptions.Item label="金币">{detail.wallet.gold}</Descriptions.Item>
            <Descriptions.Item label="银币">{detail.wallet.silver}</Descriptions.Item>
            <Descriptions.Item label="铜币">{detail.wallet.copper}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{formatDateTime(detail.updated_at)}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
      <Modal title={editingPlayerID ? `调账 · 玩家 ${editingPlayerID}` : '调账'} open={editorOpen} onCancel={() => setEditorOpen(false)} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose okText="确认调账" cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Form.Item label="变更总铜币" name="change_total_copper" rules={[{ required: true, message: '请输入增减铜币' }]}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="操作原因" name="reason" rules={[{ required: true, message: '请输入操作原因' }]}>
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', { hour12: false });
}
