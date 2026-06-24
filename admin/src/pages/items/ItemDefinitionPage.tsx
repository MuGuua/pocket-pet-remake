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
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { createAdminItem, deleteAdminItem, fetchAdminItemDetail, fetchAdminItems, updateAdminItem } from '../../services/item';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import type { AdminItemDetail, AdminItemListFilters, AdminItemSummary, AdminUpsertItemPayload } from '../../types/item';
import { formatDisplayLabel, ITEM_TYPE_LABELS } from '../../utils/displayLabels';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

interface ItemFormValues extends AdminUpsertItemPayload {}

// 物品模板页直接面向正式数据库模板，所有字段修改都会影响后续背包、掉落、商店和奖励链路。
export function ItemDefinitionPage() {
  const [filterForm] = Form.useForm<AdminItemListFilters>();
  const [editorForm] = Form.useForm<ItemFormValues>();
  const [filters, setFilters] = useState<AdminItemListFilters>({});
  const [rows, setRows] = useState<AdminItemSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminItemDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminItemDetail | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void loadItems(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadItems(nextFilters: AdminItemListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminItems({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载物品模板失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(itemID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminItemDetail(itemID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载物品详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', itemID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.setFieldsValue(defaultItemValues());
      return;
    }
    if (!itemID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminItemDetail(itemID);
      setEditingRecord(result);
      editorForm.setFieldsValue(result);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载物品编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmit(values: ItemFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminItem(editingRecord.item_id, values);
        message.success('物品模板更新成功');
      } else {
        await createAdminItem(values);
        message.success('物品模板创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      await loadItems(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存物品模板失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(itemID: number) {
    try {
      await deleteAdminItem(itemID);
      message.success('物品模板已删除');
      if (detail?.item_id === itemID) {
        setDetail(null);
        setDetailOpen(false);
      }
      await loadItems(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除物品模板失败');
    }
  }

  const columns = useMemo<ColumnsType<AdminItemSummary>>(
    () => [
      { title: '物品ID', dataIndex: 'item_id', key: 'item_id', width: 110, fixed: 'left' },
      { title: '编码', dataIndex: 'item_code', key: 'item_code', width: 150 },
      { title: '名称', dataIndex: 'item_name', key: 'item_name', width: 160 },
      { title: '分类', dataIndex: 'item_type', key: 'item_type', width: 120, render: (value: string) => <Tag color="blue">{formatDisplayLabel(ITEM_TYPE_LABELS, value)}</Tag> },
      { title: '子类', dataIndex: 'item_sub_type', key: 'item_sub_type', width: 120 },
      { title: '品质', dataIndex: 'quality', key: 'quality', width: 90 },
      { title: '堆叠上限', dataIndex: 'max_stack', key: 'max_stack', width: 110 },
      { title: '买价(铜)', dataIndex: 'buy_price_copper', key: 'buy_price_copper', width: 120 },
      { title: '卖价(铜)', dataIndex: 'sell_price_copper', key: 'sell_price_copper', width: 120 },
      { title: '启用', dataIndex: 'is_enabled', key: 'is_enabled', width: 90, render: (value: boolean) => value ? <Tag color="green">启用</Tag> : <Tag>停用</Tag> },
      {
        title: '操作', key: 'actions', width: 100, fixed: 'right', render: (_value, record) => (
          <TableActionDropdown
            actions={[
              { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.item_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.item_id) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这个物品模板吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => void handleDelete(record.item_id),
              },
            ]}
          />
        ),
      },
    ],
    [detail],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title="模板列表"
        extra={(
          <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }}>
            <Form.Item name="item_id" label="物品ID">
              <Input allowClear placeholder="物品ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="item_type" label="分类">
              <Input allowClear placeholder="分类" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="keyword" label="关键字">
              <Input allowClear placeholder="编码或名称" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用">
              <Select allowClear placeholder="状态" style={{ width: 90 }} options={[{ label: '启用', value: 'true' }, { label: '停用', value: 'false' }]} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增物品</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="item_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有模板数据" /> }}
          scroll={{ x: 1500 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条模板`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </Card>
      <Drawer title={detail ? `物品详情 · ${detail.item_name}` : '物品详情'} width={680} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? <Typography.Text type="secondary">正在加载物品详情...</Typography.Text> : (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="物品ID">{detail.item_id}</Descriptions.Item>
            <Descriptions.Item label="编码">{detail.item_code}</Descriptions.Item>
            <Descriptions.Item label="名称">{detail.item_name}</Descriptions.Item>
            <Descriptions.Item label="分类">{formatDisplayLabel(ITEM_TYPE_LABELS, detail.item_type)}</Descriptions.Item>
            <Descriptions.Item label="子类">{detail.item_sub_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="品质 / 稀有度">{detail.quality} / {detail.rarity}</Descriptions.Item>
            <Descriptions.Item label="堆叠上限">{detail.max_stack}</Descriptions.Item>
            <Descriptions.Item label="占格">{detail.occupy_slots}</Descriptions.Item>
            <Descriptions.Item label="买价(铜)">{detail.buy_price_copper}</Descriptions.Item>
            <Descriptions.Item label="卖价(铜)">{detail.sell_price_copper}</Descriptions.Item>
            <Descriptions.Item label="效果类型">{detail.effect_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="效果值">{detail.effect_value}</Descriptions.Item>
            <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="说明" span={2}>{detail.desc || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
      <Modal title={editingRecord ? `编辑模板 · ${editingRecord.item_name}` : '新增物品模板'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={720} style={{ top: FIXED_FORM_MODAL_TOP }} styles={FIXED_FORM_MODAL_STYLES} okText={editingRecord ? '保存修改' : '创建模板'} cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Row gutter={16}>
            <Col xs={24} md={8}><Form.Item label="物品ID" name="item_id" rules={[{ required: true, message: '请输入物品ID' }]}><InputNumber min={1} disabled={Boolean(editingRecord)} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="物品编码" name="item_code" rules={[{ required: true, message: '请输入物品编码' }]}><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="物品名称" name="item_name" rules={[{ required: true, message: '请输入物品名称' }]}><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="主分类" name="item_type" rules={[{ required: true, message: '请输入主分类' }]}><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="子分类" name="item_sub_type"><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="品质" name="quality"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="稀有度" name="rarity"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={16}>
              <Form.Item
                label="图标资源"
                name="icon"
                extra="直接填写客户端资源路径。单图可填 res://...png；图集中某一格建议先做成 AtlasTexture .tres，再填 res://...tres。"
              >
                <Input placeholder="res://resources/item_icons/red_potion_icon.tres" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}><Form.Item label="堆叠上限" name="max_stack"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="占格" name="occupy_slots"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="买价(铜)" name="buy_price_copper"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="卖价(铜)" name="sell_price_copper"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="回收价(铜)" name="recycle_price_copper"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}><Form.Item label="描述" name="desc"><Input.TextArea rows={3} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item label="效果类型" name="effect_type"><Input /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item label="效果值" name="effect_value"><InputNumber style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}><Form.Item label="效果参数 JSON" name="effect_params_json"><Input.TextArea rows={3} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可使用" name="usable" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可出售" name="can_sell" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可丢弃" name="can_drop" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可存仓" name="can_store" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可交易" name="can_trade" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="自动并堆" name="auto_merge" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="启用" name="is_enabled" valuePropName="checked"><Switch /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultItemValues(): ItemFormValues {
  return {
    item_id: 10001,
    item_code: 'new_item_code',
    item_name: '新物品模板',
    item_type: 'consumable',
    item_sub_type: '',
    quality: 1,
    rarity: 1,
    icon: '',
    desc: '',
    max_stack: 1,
    occupy_slots: 1,
    auto_merge: true,
    sort_weight: 0,
    usable: false,
    use_scope: '',
    target_type: '',
    required_level: 0,
    required_scene_id: 0,
    bind_type: 'none',
    can_sell: false,
    can_drop: false,
    can_store: true,
    can_trade: false,
    expire_at_rule: '',
    effect_type: '',
    effect_value: 0,
    effect_params_json: '{}',
    buy_price_copper: 0,
    sell_price_copper: 0,
    recycle_price_copper: 0,
    price_type: 'base_coin',
    is_enabled: true,
  };
}
