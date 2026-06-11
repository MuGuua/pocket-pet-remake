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
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tabs,
  Tag,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import {
  createAdminNPCEntity,
  createAdminNPCMenuEntry,
  deleteAdminNPCEntity,
  deleteAdminNPCMenuEntry,
  fetchAdminNPCEntities,
  fetchAdminNPCEntityDetail,
  fetchAdminNPCMenuEntries,
  fetchAdminNPCMenuEntryDetail,
  updateAdminNPCEntity,
  updateAdminNPCMenuEntry,
} from '../../services/npc';
import type {
  AdminCreateNPCEntityPayload,
  AdminCreateNPCMenuEntryPayload,
  AdminNPCEntityDetail,
  AdminNPCEntityFilters,
  AdminNPCEntitySummary,
  AdminNPCMenuEntryDetail,
  AdminNPCMenuEntryFilters,
  AdminNPCMenuEntrySummary,
  AdminUpdateNPCEntityPayload,
  AdminUpdateNPCMenuEntryPayload,
} from '../../types/npc';

interface EntityFormValues {
  entity_id?: number;
  entity_code: string;
  display_name: string;
  entity_type: number;
  scene_id: number;
  pos_x: number;
  pos_y: number;
  dir: number;
  speed: number;
  status: number;
}

interface MenuEntryFormValues {
  entity_id: number;
  entry_id?: string;
  entry_type: string;
  title: string;
  subtitle: string;
  state: string;
  priority: number;
  sort_order: number;
  action_result_type: string;
  action_notice: string;
  status: number;
}

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '启用', value: '1' },
  { label: '停用', value: '0' },
];

const editableStatusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

const entryTypeOptions = [
  { label: 'dialog', value: 'dialog' },
  { label: 'shop', value: 'shop' },
  { label: 'quest', value: 'quest' },
];

const actionResultTypeOptions = [
  { label: 'notice', value: 'notice' },
  { label: 'dialog', value: 'dialog' },
  { label: 'shop', value: 'shop' },
];

// NPC 配置页把地图中的实体分布和 NPC 菜单动作一起放到后台，方便运营直接改 scene_id、坐标和交互入口。
export function NPCConfigPage() {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="info" showIcon message="NPC 配置管理已接入真实服务端接口，可直接维护世界实体分布与 NPC 菜单动作。" />
      <Tabs defaultActiveKey="entities" items={[
        { key: 'entities', label: '地图实体配置', children: <NPCEntityPanel /> },
        { key: 'menu_entries', label: 'NPC 菜单配置', children: <NPCMenuEntryPanel /> },
      ]} />
    </Space>
  );
}

function NPCEntityPanel() {
  const [filterForm] = Form.useForm<AdminNPCEntityFilters>();
  const [editorForm] = Form.useForm<EntityFormValues>();
  const [filters, setFilters] = useState<AdminNPCEntityFilters>({ status: '1' });
  const [rows, setRows] = useState<AdminNPCEntitySummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminNPCEntityDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminNPCEntityDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);

  useEffect(() => { filterForm.setFieldsValue({ status: '1' }); }, [filterForm]);
  useEffect(() => { void loadRows(filters, page, pageSize); }, [filters, page, pageSize]);

  async function loadRows(nextFilters: AdminNPCEntityFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminNPCEntities({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items); setTotal(result.total); setPage(result.page); setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 NPC 实体失败');
      setRows([]); setTotal(0);
    } finally { setLoading(false); }
  }

  async function handleViewDetail(entityID: number) {
    setDetailOpen(true); setDetailLoading(true); setDetail(null);
    try { setDetail(await fetchAdminNPCEntityDetail(entityID)); }
    catch (error) { message.error(error instanceof Error ? error.message : '加载 NPC 实体详情失败'); setDetailOpen(false); }
    finally { setDetailLoading(false); }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', entityID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null); editorForm.resetFields(); editorForm.setFieldsValue(defaultEntityValues()); return;
    }
    if (!entityID) return;
    setDetailLoading(true);
    try { const result = await fetchAdminNPCEntityDetail(entityID); setEditingRecord(result); editorForm.setFieldsValue(mapEntityDetailToForm(result)); }
    catch (error) { message.error(error instanceof Error ? error.message : '加载 NPC 实体编辑数据失败'); setEditorOpen(false); }
    finally { setDetailLoading(false); }
  }

  async function handleSubmitEditor(values: EntityFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminNPCEntity(editingRecord.entity_id, mapEntityFormToUpdatePayload(values));
        message.success('NPC 实体更新成功');
      } else {
        await createAdminNPCEntity(mapEntityFormToCreatePayload(values));
        message.success('NPC 实体创建成功');
      }
      setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); await loadRows(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存 NPC 实体失败');
    } finally { setSaving(false); }
  }

  async function handleDelete(entityID: number) {
    setDeletingID(entityID);
    try {
      await deleteAdminNPCEntity(entityID); message.success('NPC 实体已删除');
      if (detail?.entity_id === entityID) { setDetailOpen(false); setDetail(null); }
      await loadRows(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除 NPC 实体失败');
    } finally { setDeletingID(null); }
  }

  const columns = useMemo<ColumnsType<AdminNPCEntitySummary>>(() => [
    { title: '实体ID', dataIndex: 'entity_id', key: 'entity_id', width: 110, fixed: 'left' },
    { title: '实体编码', dataIndex: 'entity_code', key: 'entity_code', width: 160 },
    { title: '显示名', dataIndex: 'display_name', key: 'display_name', width: 160 },
    { title: '地图', dataIndex: 'scene_id', key: 'scene_id', width: 90 },
    { title: '坐标', key: 'pos', width: 120, render: (_v, record) => `${record.pos_x}, ${record.pos_y}` },
    { title: '方向/速度', key: 'move', width: 120, render: (_v, record) => `${record.dir}/${record.speed}` },
    { title: '状态', dataIndex: 'status_text', key: 'status_text', width: 100, render: (value: string) => <Tag color={value === '启用' ? 'green' : 'default'}>{value}</Tag> },
    { title: '操作', key: 'actions', width: 220, fixed: 'right', render: (_v, record) => <Space size="small"><Button type="link" onClick={() => void handleViewDetail(record.entity_id)}>查看</Button><Button type="link" onClick={() => void handleOpenEditor('edit', record.entity_id)}>编辑</Button><Popconfirm title="确认删除这个实体吗？" onConfirm={() => void handleDelete(record.entity_id)}><Button type="link" danger loading={deletingID === record.entity_id}>删除</Button></Popconfirm></Space> },
  ], [deletingID]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}><Card><Statistic title="当前页实体数" value={rows.length} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="涉及地图数" value={new Set(rows.map((item) => item.scene_id)).size} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="总记录数" value={total} /></Card></Col>
      </Row>
      <Card title="实体筛选" extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增实体</Button>}>
        <Form form={filterForm} layout="vertical" onFinish={(values) => { setPage(1); setFilters(values); }}>
          <Row gutter={16}>
            <Col xs={24} md={6}><Form.Item label="实体ID" name="entity_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="显示名" name="name"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="地图ID" name="scene_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="状态" name="status"><Select options={statusOptions} /></Form.Item></Col>
          </Row>
          <Space><Button type="primary" htmlType="submit" loading={loading}>查询</Button><Button onClick={() => { filterForm.resetFields(); filterForm.setFieldsValue({ status: '1' }); setPage(1); setFilters({ status: '1' }); }}>重置</Button></Space>
        </Form>
      </Card>
      <Card title="实体列表">
        <Table columns={columns} dataSource={rows} rowKey="entity_id" loading={loading} locale={{ emptyText: <Empty description="当前筛选条件下没有实体数据" /> }} scroll={{ x: 1300 }} pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (value) => `共 ${value} 个实体`, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} />
      </Card>
      <Drawer title={detail ? `实体详情 · ${detail.entity_id}` : '实体详情'} width={520} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading ? <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}><Spin tip="正在加载实体详情..." /></div> : detail ? <Descriptions bordered column={2} size="small"><Descriptions.Item label="实体ID">{detail.entity_id}</Descriptions.Item><Descriptions.Item label="编码">{detail.entity_code}</Descriptions.Item><Descriptions.Item label="显示名">{detail.display_name}</Descriptions.Item><Descriptions.Item label="地图ID">{detail.scene_id}</Descriptions.Item><Descriptions.Item label="坐标">{`${detail.pos_x}, ${detail.pos_y}`}</Descriptions.Item><Descriptions.Item label="方向 / 速度">{`${detail.dir} / ${detail.speed}`}</Descriptions.Item><Descriptions.Item label="状态">{detail.status_text}</Descriptions.Item></Descriptions> : null}
      </Drawer>
      <Modal title={editingRecord ? `编辑实体 · ${editingRecord.entity_id}` : '新增实体'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={720} okText={editingRecord ? '保存修改' : '创建实体'} cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            {!editingRecord ? <Col xs={24} md={8}><Form.Item label="实体ID" name="entity_id" rules={[{ required: true, message: '请输入实体ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col> : null}
            <Col xs={24} md={editingRecord ? 12 : 8}><Form.Item label="实体编码" name="entity_code" rules={[{ required: true, message: '请输入实体编码' }]}><Input /></Form.Item></Col>
            <Col xs={24} md={editingRecord ? 12 : 8}><Form.Item label="显示名" name="display_name" rules={[{ required: true, message: '请输入显示名' }]}><Input /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="实体类型" name="entity_type"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="地图ID" name="scene_id"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="X" name="pos_x"><InputNumber style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="Y" name="pos_y"><InputNumber style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="方向" name="dir"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="速度" name="speed"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="状态" name="status"><Select options={editableStatusOptions} /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function NPCMenuEntryPanel() {
  const [filterForm] = Form.useForm<AdminNPCMenuEntryFilters>();
  const [editorForm] = Form.useForm<MenuEntryFormValues>();
  const [filters, setFilters] = useState<AdminNPCMenuEntryFilters>({ status: '1' });
  const [rows, setRows] = useState<AdminNPCMenuEntrySummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminNPCMenuEntryDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminNPCMenuEntryDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingKey, setDeletingKey] = useState<string | null>(null);

  useEffect(() => { filterForm.setFieldsValue({ status: '1' }); }, [filterForm]);
  useEffect(() => { void loadRows(filters, page, pageSize); }, [filters, page, pageSize]);

  async function loadRows(nextFilters: AdminNPCMenuEntryFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminNPCMenuEntries({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items); setTotal(result.total); setPage(result.page); setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 NPC 菜单失败');
      setRows([]); setTotal(0);
    } finally { setLoading(false); }
  }

  async function handleViewDetail(entityID: number, entryID: string) {
    setDetailOpen(true); setDetailLoading(true); setDetail(null);
    try { setDetail(await fetchAdminNPCMenuEntryDetail(entityID, entryID)); }
    catch (error) { message.error(error instanceof Error ? error.message : '加载 NPC 菜单详情失败'); setDetailOpen(false); }
    finally { setDetailLoading(false); }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', entityID?: number, entryID?: string) {
    setEditorOpen(true);
    if (mode === 'create') { setEditingRecord(null); editorForm.resetFields(); editorForm.setFieldsValue(defaultMenuEntryValues()); return; }
    if (!entityID || !entryID) return;
    setDetailLoading(true);
    try { const result = await fetchAdminNPCMenuEntryDetail(entityID, entryID); setEditingRecord(result); editorForm.setFieldsValue(mapMenuEntryDetailToForm(result)); }
    catch (error) { message.error(error instanceof Error ? error.message : '加载 NPC 菜单编辑数据失败'); setEditorOpen(false); }
    finally { setDetailLoading(false); }
  }

  async function handleSubmitEditor(values: MenuEntryFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminNPCMenuEntry(editingRecord.entity_id, editingRecord.entry_id, mapMenuEntryFormToUpdatePayload(values));
        message.success('NPC 菜单更新成功');
      } else {
        await createAdminNPCMenuEntry(mapMenuEntryFormToCreatePayload(values));
        message.success('NPC 菜单创建成功');
      }
      setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); await loadRows(filters, page, pageSize);
    } catch (error) { message.error(error instanceof Error ? error.message : '保存 NPC 菜单失败'); }
    finally { setSaving(false); }
  }

  async function handleDelete(entityID: number, entryID: string) {
    const key = `${entityID}:${entryID}`;
    setDeletingKey(key);
    try {
      await deleteAdminNPCMenuEntry(entityID, entryID); message.success('NPC 菜单已删除');
      if (detail?.entity_id === entityID && detail.entry_id === entryID) { setDetailOpen(false); setDetail(null); }
      await loadRows(filters, page, pageSize);
    } catch (error) { message.error(error instanceof Error ? error.message : '删除 NPC 菜单失败'); }
    finally { setDeletingKey(null); }
  }

  const columns = useMemo<ColumnsType<AdminNPCMenuEntrySummary>>(() => [
    { title: '实体ID', dataIndex: 'entity_id', key: 'entity_id', width: 110, fixed: 'left' },
    { title: '入口ID', dataIndex: 'entry_id', key: 'entry_id', width: 160 },
    { title: '入口类型', dataIndex: 'entry_type', key: 'entry_type', width: 120 },
    { title: '标题', dataIndex: 'title', key: 'title', width: 180 },
    { title: '状态', dataIndex: 'status_text', key: 'status_text', width: 100, render: (value: string) => <Tag color={value === '启用' ? 'green' : 'default'}>{value}</Tag> },
    { title: '优先级/排序', key: 'sort', width: 120, render: (_v, record) => `${record.priority}/${record.sort_order}` },
    { title: '动作类型', dataIndex: 'action_result_type', key: 'action_result_type', width: 120 },
    { title: '操作', key: 'actions', width: 220, fixed: 'right', render: (_v, record) => <Space size="small"><Button type="link" onClick={() => void handleViewDetail(record.entity_id, record.entry_id)}>查看</Button><Button type="link" onClick={() => void handleOpenEditor('edit', record.entity_id, record.entry_id)}>编辑</Button><Popconfirm title="确认删除这条菜单配置吗？" onConfirm={() => void handleDelete(record.entity_id, record.entry_id)}><Button type="link" danger loading={deletingKey === `${record.entity_id}:${record.entry_id}`}>删除</Button></Popconfirm></Space> },
  ], [deletingKey]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}><Card><Statistic title="当前页菜单项" value={rows.length} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="覆盖 NPC 数" value={new Set(rows.map((item) => item.entity_id)).size} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="总记录数" value={total} /></Card></Col>
      </Row>
      <Card title="菜单筛选" extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增菜单项</Button>}>
        <Form form={filterForm} layout="vertical" onFinish={(values) => { setPage(1); setFilters(values); }}>
          <Row gutter={16}>
            <Col xs={24} md={8}><Form.Item label="实体ID" name="entity_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="入口ID" name="entry_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="状态" name="status"><Select options={statusOptions} /></Form.Item></Col>
          </Row>
          <Space><Button type="primary" htmlType="submit" loading={loading}>查询</Button><Button onClick={() => { filterForm.resetFields(); filterForm.setFieldsValue({ status: '1' }); setPage(1); setFilters({ status: '1' }); }}>重置</Button></Space>
        </Form>
      </Card>
      <Card title="菜单列表">
        <Table columns={columns} dataSource={rows} rowKey={(record) => `${record.entity_id}:${record.entry_id}`} loading={loading} locale={{ emptyText: <Empty description="当前筛选条件下没有菜单配置" /> }} scroll={{ x: 1400 }} pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (value) => `共 ${value} 条菜单配置`, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} />
      </Card>
      <Drawer title={detail ? `菜单详情 · ${detail.entry_id}` : '菜单详情'} width={560} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading ? <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}><Spin tip="正在加载菜单详情..." /></div> : detail ? <Descriptions bordered column={1} size="small"><Descriptions.Item label="实体ID">{detail.entity_id}</Descriptions.Item><Descriptions.Item label="入口ID">{detail.entry_id}</Descriptions.Item><Descriptions.Item label="标题">{detail.title}</Descriptions.Item><Descriptions.Item label="副标题">{detail.subtitle}</Descriptions.Item><Descriptions.Item label="动作类型">{detail.action_result_type}</Descriptions.Item><Descriptions.Item label="提示文案">{detail.action_notice}</Descriptions.Item></Descriptions> : null}
      </Drawer>
      <Modal title={editingRecord ? `编辑菜单项 · ${editingRecord.entry_id}` : '新增菜单项'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={760} okText={editingRecord ? '保存修改' : '创建菜单项'} cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            <Col xs={24} md={12}><Form.Item label="实体ID" name="entity_id" rules={[{ required: true, message: '请输入实体ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            {!editingRecord ? <Col xs={24} md={12}><Form.Item label="入口ID" name="entry_id" rules={[{ required: true, message: '请输入入口ID' }]}><Input /></Form.Item></Col> : null}
            <Col xs={24} md={12}><Form.Item label="入口类型" name="entry_type"><Select options={entryTypeOptions} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item label="标题" name="title"><Input /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item label="副标题" name="subtitle"><Input /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item label="状态 key" name="state"><Input /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="优先级" name="priority"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="排序" name="sort_order"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="动作类型" name="action_result_type"><Select options={actionResultTypeOptions} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="状态" name="status"><Select options={editableStatusOptions} /></Form.Item></Col>
            <Col span={24}><Form.Item label="提示文案" name="action_notice"><Input.TextArea rows={4} /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultEntityValues(): EntityFormValues {
  return { entity_id: 99001, entity_code: 'ops_npc', display_name: '后台测试 NPC', entity_type: 2, scene_id: 1, pos_x: 8, pos_y: 8, dir: 2, speed: 0, status: 1 };
}

function mapEntityDetailToForm(detail: AdminNPCEntityDetail): EntityFormValues {
  return { entity_id: detail.entity_id, entity_code: detail.entity_code, display_name: detail.display_name, entity_type: detail.entity_type, scene_id: detail.scene_id, pos_x: detail.pos_x, pos_y: detail.pos_y, dir: detail.dir, speed: detail.speed, status: detail.status };
}

function mapEntityFormToCreatePayload(values: EntityFormValues): AdminCreateNPCEntityPayload {
  return { entity_id: values.entity_id ?? 0, entity_code: values.entity_code, display_name: values.display_name, entity_type: values.entity_type, scene_id: values.scene_id, pos_x: values.pos_x, pos_y: values.pos_y, dir: values.dir, speed: values.speed, status: values.status };
}

function mapEntityFormToUpdatePayload(values: EntityFormValues): AdminUpdateNPCEntityPayload {
  const { entity_id: _entityID, ...rest } = mapEntityFormToCreatePayload(values);
  return rest;
}

function defaultMenuEntryValues(): MenuEntryFormValues {
  return { entity_id: 93001, entry_id: 'ops_dialog', entry_type: 'dialog', title: '后台菜单项', subtitle: '用于测试新入口', state: 'available', priority: 100, sort_order: 10, action_result_type: 'notice', action_notice: '这是一条后台新增提示。', status: 1 };
}

function mapMenuEntryDetailToForm(detail: AdminNPCMenuEntryDetail): MenuEntryFormValues {
  return { entity_id: detail.entity_id, entry_id: detail.entry_id, entry_type: detail.entry_type, title: detail.title, subtitle: detail.subtitle, state: detail.state, priority: detail.priority, sort_order: detail.sort_order, action_result_type: detail.action_result_type, action_notice: detail.action_notice, status: detail.status };
}

function mapMenuEntryFormToCreatePayload(values: MenuEntryFormValues): AdminCreateNPCMenuEntryPayload {
  return { entity_id: values.entity_id, entry_id: values.entry_id ?? '', entry_type: values.entry_type, title: values.title, subtitle: values.subtitle, state: values.state, priority: values.priority, sort_order: values.sort_order, action_result_type: values.action_result_type, action_notice: values.action_notice, status: values.status };
}

function mapMenuEntryFormToUpdatePayload(values: MenuEntryFormValues): AdminUpdateNPCMenuEntryPayload {
  const { entry_id: _entryID, ...rest } = mapMenuEntryFormToCreatePayload(values);
  return rest;
}
