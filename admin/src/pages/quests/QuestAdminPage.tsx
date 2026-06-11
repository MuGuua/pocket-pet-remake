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
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import type { CSSProperties } from 'react';
import {
  createAdminPlayerQuest,
  createAdminQuestTemplate,
  deleteAdminPlayerQuest,
  deleteAdminQuestTemplate,
  fetchAdminPlayerQuestDetail,
  fetchAdminPlayerQuests,
  fetchAdminQuestTemplateDetail,
  fetchAdminQuestTemplates,
  updateAdminPlayerQuest,
  updateAdminQuestTemplate,
} from '../../services/quest';
import type {
  AdminCreatePlayerQuestPayload,
  AdminCreateQuestTemplatePayload,
  AdminPlayerQuestDetail,
  AdminPlayerQuestFilters,
  AdminPlayerQuestObjectiveInput,
  AdminPlayerQuestSummary,
  AdminQuestObjectiveInput,
  AdminQuestTemplateDetail,
  AdminQuestTemplateFilters,
  AdminQuestTemplateSummary,
  AdminUpdatePlayerQuestPayload,
  AdminUpdateQuestTemplatePayload,
} from '../../types/quest';

interface TemplateFormValues {
  quest_id?: number;
  name: string;
  quest_type: string;
  title: string;
  description: string;
  chapter: number;
  sort_order: number;
  accept_mode: string;
  submit_mode: string;
  auto_track: boolean;
  start_npc_id: number;
  submit_npc_id: number;
  min_player_level: number;
  status: number;
  pre_quest_ids_text: string;
  objectives_text: string;
}

interface PlayerQuestFormValues {
  player_id: number;
  quest_id: number;
  state: string;
  tracked: boolean;
  reward_claimed: boolean;
  objectives_text: string;
}

const templateStatusOptions = [
  { label: '全部状态', value: '' },
  { label: '启用', value: '1' },
  { label: '停用', value: '0' },
];

const editableTemplateStatusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

const questTypeOptions = [
  { label: 'MAIN', value: 'MAIN' },
  { label: 'SIDE', value: 'SIDE' },
  { label: 'DAILY', value: 'DAILY' },
];

const questStateOptions = [
  { label: '全部状态', value: '' },
  { label: 'LOCKED', value: 'LOCKED' },
  { label: 'AVAILABLE', value: 'AVAILABLE' },
  { label: 'ACCEPTED', value: 'ACCEPTED' },
  { label: 'READY_TO_SUBMIT', value: 'READY_TO_SUBMIT' },
  { label: 'COMPLETED', value: 'COMPLETED' },
];

const editableQuestStateOptions = questStateOptions.filter((item) => item.value !== '');

// 任务管理页把模板管理和玩家任务修正放到同一个页面，方便策划和运营在一个入口完成模板配置与进度排障。
export function QuestAdminPage() {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="info" showIcon message="任务管理已接入真实服务端接口，支持任务模板 CRUD 与玩家任务进度修正。" />
      <Tabs
        defaultActiveKey="templates"
        items={[
          { key: 'templates', label: '任务模板管理', children: <QuestTemplatePanel /> },
          { key: 'player-progress', label: '玩家任务进度', children: <PlayerQuestPanel /> },
        ]}
      />
    </Space>
  );
}

function QuestTemplatePanel() {
  const [filterForm] = Form.useForm<AdminQuestTemplateFilters>();
  const [editorForm] = Form.useForm<TemplateFormValues>();
  const [filters, setFilters] = useState<AdminQuestTemplateFilters>({ status: '1' });
  const [rows, setRows] = useState<AdminQuestTemplateSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminQuestTemplateDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminQuestTemplateDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);

  useEffect(() => {
    filterForm.setFieldsValue({ status: '1' });
  }, [filterForm]);

  useEffect(() => {
    void loadTemplates(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadTemplates(nextFilters: AdminQuestTemplateFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminQuestTemplates({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载任务模板失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(questID: number) {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      setDetail(await fetchAdminQuestTemplateDetail(questID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载任务模板详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', questID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultTemplateValues());
      return;
    }
    if (!questID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminQuestTemplateDetail(questID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapTemplateDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载任务模板编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: TemplateFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminQuestTemplate(editingRecord.quest_id, mapTemplateFormToUpdatePayload(values));
        message.success('任务模板更新成功');
      } else {
        await createAdminQuestTemplate(mapTemplateFormToCreatePayload(values));
        message.success('任务模板创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadTemplates(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存任务模板失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(questID: number) {
    setDeletingID(questID);
    try {
      await deleteAdminQuestTemplate(questID);
      message.success('任务模板已停用');
      if (detail?.quest_id === questID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadTemplates(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除任务模板失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminQuestTemplateSummary>>(
    () => [
      { title: '任务ID', dataIndex: 'quest_id', key: 'quest_id', width: 110, fixed: 'left' },
      { title: '模板名', dataIndex: 'name', key: 'name', width: 160 },
      { title: '标题', dataIndex: 'title', key: 'title', width: 180 },
      { title: '类型', dataIndex: 'quest_type', key: 'quest_type', width: 100 },
      { title: '章节/排序', key: 'sort', width: 120, render: (_v, record) => `${record.chapter}/${record.sort_order}` },
      { title: '接/交模式', key: 'mode', width: 140, render: (_v, record) => `${record.accept_mode}/${record.submit_mode}` },
      { title: '自动追踪', dataIndex: 'auto_track', key: 'auto_track', width: 100, render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '是' : '否'}</Tag> },
      { title: '状态', dataIndex: 'status_text', key: 'status_text', width: 100, render: (value: string) => <Tag color={value === '启用' ? 'blue' : 'default'}>{value}</Tag> },
      {
        title: '操作', key: 'actions', width: 220, fixed: 'right', render: (_v, record) => (
          <Space size="small">
            <Button type="link" onClick={() => void handleViewDetail(record.quest_id)}>查看</Button>
            <Button type="link" onClick={() => void handleOpenEditor('edit', record.quest_id)}>编辑</Button>
            <Popconfirm title="确认停用这个任务模板吗？" description="这里会把模板状态改成停用，不直接物理删除。" onConfirm={() => void handleDelete(record.quest_id)} okText="确认停用" cancelText="取消">
              <Button type="link" danger loading={deletingID === record.quest_id}>删除</Button>
            </Popconfirm>
          </Space>
        )
      },
    ],
    [deletingID],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}><Card><Statistic title="当前页模板数" value={rows.length} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="启用模板" value={rows.filter((item) => item.status === 1).length} valueStyle={{ color: '#2f7d4a' }} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="总模板数" value={total} /></Card></Col>
      </Row>
      <Card title="模板筛选" extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增模板</Button>}>
        <Form form={filterForm} layout="vertical" onFinish={(values) => { setPage(1); setFilters(values); }}>
          <Row gutter={16}>
            <Col xs={24} md={6}><Form.Item label="任务ID" name="quest_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="类型" name="quest_type"><Select allowClear options={questTypeOptions} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="标题" name="title"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="状态" name="status"><Select options={templateStatusOptions} /></Form.Item></Col>
          </Row>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
            <Button onClick={() => { filterForm.resetFields(); filterForm.setFieldsValue({ status: '1' }); setPage(1); setFilters({ status: '1' }); }}>重置</Button>
          </Space>
        </Form>
      </Card>
      <Card title="模板列表">
        <Table columns={columns} dataSource={rows} rowKey="quest_id" loading={loading} locale={{ emptyText: <Empty description="当前筛选条件下没有任务模板" /> }} scroll={{ x: 1400 }} pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (value) => `共 ${value} 个模板`, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} />
      </Card>
      <Drawer title={detail ? `任务模板详情 · ${detail.quest_id}` : '任务模板详情'} width={640} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading ? <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}><Spin tip="正在加载任务模板详情..." /></div> : detail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="任务ID">{detail.quest_id}</Descriptions.Item>
            <Descriptions.Item label="模板名">{detail.name}</Descriptions.Item>
            <Descriptions.Item label="标题" span={2}>{detail.title}</Descriptions.Item>
            <Descriptions.Item label="类型">{detail.quest_type}</Descriptions.Item>
            <Descriptions.Item label="状态">{detail.status_text}</Descriptions.Item>
            <Descriptions.Item label="接取方式">{detail.accept_mode}</Descriptions.Item>
            <Descriptions.Item label="提交方式">{detail.submit_mode}</Descriptions.Item>
            <Descriptions.Item label="起始 NPC">{detail.start_npc_id}</Descriptions.Item>
            <Descriptions.Item label="提交 NPC">{detail.submit_npc_id}</Descriptions.Item>
            <Descriptions.Item label="前置任务" span={2}>{detail.pre_quest_ids.length > 0 ? detail.pre_quest_ids.join(', ') : '无'}</Descriptions.Item>
            <Descriptions.Item label="目标定义" span={2}><pre style={jsonBlockStyle}>{JSON.stringify(detail.objectives, null, 2)}</pre></Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>
      <Modal title={editingRecord ? `编辑任务模板 · ${editingRecord.quest_id}` : '新增任务模板'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={820} okText={editingRecord ? '保存修改' : '创建模板'} cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            {!editingRecord ? <Col xs={24} md={8}><Form.Item label="任务ID" name="quest_id" rules={[{ required: true, message: '请输入任务ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col> : null}
            <Col xs={24} md={editingRecord ? 12 : 8}><Form.Item label="模板名" name="name" rules={[{ required: true, message: '请输入模板名' }]}><Input /></Form.Item></Col>
            <Col xs={24} md={editingRecord ? 12 : 8}><Form.Item label="任务类型" name="quest_type" rules={[{ required: true, message: '请选择任务类型' }]}><Select options={questTypeOptions} /></Form.Item></Col>
            <Col span={24}><Form.Item label="标题" name="title" rules={[{ required: true, message: '请输入标题' }]}><Input /></Form.Item></Col>
            <Col span={24}><Form.Item label="描述" name="description"><Input.TextArea rows={3} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="章节" name="chapter"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="排序" name="sort_order"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="接取方式" name="accept_mode"><Select options={[{ label: 'AUTO', value: 'AUTO' }, { label: 'NPC', value: 'NPC' }]} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="提交方式" name="submit_mode"><Select options={[{ label: 'AUTO', value: 'AUTO' }, { label: 'NPC', value: 'NPC' }]} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="起始 NPC" name="start_npc_id"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="提交 NPC" name="submit_npc_id"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="最低等级" name="min_player_level"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="状态" name="status"><Select options={editableTemplateStatusOptions} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="自动追踪" name="auto_track" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col span={24}><Form.Item label="前置任务 ID JSON" name="pre_quest_ids_text" extra='示例: [1001,1002]'><Input.TextArea rows={2} /></Form.Item></Col>
            <Col span={24}><Form.Item label="目标定义 JSON" name="objectives_text" extra='示例: [{"objective_id":1,"event_type":"ENTER_SCENE","description":"进入场景","target_value":1,"target_selector":{"scene_id":2}}]'><Input.TextArea rows={8} /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function PlayerQuestPanel() {
  const [filterForm] = Form.useForm<AdminPlayerQuestFilters>();
  const [editorForm] = Form.useForm<PlayerQuestFormValues>();
  const [filters, setFilters] = useState<AdminPlayerQuestFilters>({});
  const [rows, setRows] = useState<AdminPlayerQuestSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminPlayerQuestDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminPlayerQuestDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);

  useEffect(() => {
    void loadPlayerQuests(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadPlayerQuests(nextFilters: AdminPlayerQuestFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminPlayerQuests({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家任务失败');
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
      setDetail(await fetchAdminPlayerQuestDetail(recordID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家任务详情失败');
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
      editorForm.setFieldsValue(defaultPlayerQuestValues());
      return;
    }
    if (!recordID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminPlayerQuestDetail(recordID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapPlayerQuestDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家任务编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: PlayerQuestFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminPlayerQuest(editingRecord.record_id, mapPlayerQuestFormToUpdatePayload(values));
        message.success('玩家任务更新成功');
      } else {
        await createAdminPlayerQuest(mapPlayerQuestFormToCreatePayload(values));
        message.success('玩家任务创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadPlayerQuests(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存玩家任务失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(recordID: number) {
    setDeletingID(recordID);
    try {
      await deleteAdminPlayerQuest(recordID);
      message.success('玩家任务记录已删除');
      if (detail?.record_id === recordID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadPlayerQuests(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除玩家任务失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminPlayerQuestSummary>>(
    () => [
      { title: '记录ID', dataIndex: 'record_id', key: 'record_id', width: 120, fixed: 'left' },
      { title: '玩家ID', dataIndex: 'player_id', key: 'player_id', width: 110 },
      { title: '玩家名', dataIndex: 'player_name', key: 'player_name', width: 140 },
      { title: '任务ID', dataIndex: 'quest_id', key: 'quest_id', width: 110 },
      { title: '任务标题', dataIndex: 'quest_title', key: 'quest_title', width: 180 },
      { title: '状态', dataIndex: 'state', key: 'state', width: 150, render: (value: string) => <Tag color={value === 'COMPLETED' ? 'green' : 'blue'}>{value}</Tag> },
      { title: '追踪', dataIndex: 'tracked', key: 'tracked', width: 90, render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '是' : '否'}</Tag> },
      { title: '已领奖', dataIndex: 'reward_claimed', key: 'reward_claimed', width: 100, render: (value: boolean) => <Tag color={value ? 'gold' : 'default'}>{value ? '是' : '否'}</Tag> },
      {
        title: '操作', key: 'actions', width: 220, fixed: 'right', render: (_v, record) => (
          <Space size="small">
            <Button type="link" onClick={() => void handleViewDetail(record.record_id)}>查看</Button>
            <Button type="link" onClick={() => void handleOpenEditor('edit', record.record_id)}>编辑</Button>
            <Popconfirm title="确认删除这条玩家任务记录吗？" onConfirm={() => void handleDelete(record.record_id)} okText="确认删除" cancelText="取消">
              <Button type="link" danger loading={deletingID === record.record_id}>删除</Button>
            </Popconfirm>
          </Space>
        )
      },
    ],
    [deletingID],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}><Card><Statistic title="当前页任务数" value={rows.length} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="已完成任务" value={rows.filter((item) => item.state === 'COMPLETED').length} valueStyle={{ color: '#2f7d4a' }} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="总记录数" value={total} /></Card></Col>
      </Row>
      <Card title="玩家任务筛选" extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增玩家任务</Button>}>
        <Form form={filterForm} layout="vertical" onFinish={(values) => { setPage(1); setFilters(values); }}>
          <Row gutter={16}>
            <Col xs={24} md={6}><Form.Item label="记录ID" name="record_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="玩家ID" name="player_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="任务ID" name="quest_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="状态" name="state"><Select options={questStateOptions} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="是否追踪" name="tracked"><Select allowClear options={[{ label: '全部', value: '' }, { label: '是', value: 'true' }, { label: '否', value: 'false' }]} /></Form.Item></Col>
          </Row>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
            <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
          </Space>
        </Form>
      </Card>
      <Card title="玩家任务列表">
        <Table columns={columns} dataSource={rows} rowKey="record_id" loading={loading} locale={{ emptyText: <Empty description="当前筛选条件下没有玩家任务" /> }} scroll={{ x: 1500 }} pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (value) => `共 ${value} 条任务记录`, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} />
      </Card>
      <Drawer title={detail ? `玩家任务详情 · ${detail.record_id}` : '玩家任务详情'} width={640} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading ? <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}><Spin tip="正在加载玩家任务详情..." /></div> : detail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="记录ID">{detail.record_id}</Descriptions.Item>
            <Descriptions.Item label="玩家ID">{detail.player_id}</Descriptions.Item>
            <Descriptions.Item label="玩家名">{detail.player_name}</Descriptions.Item>
            <Descriptions.Item label="任务ID">{detail.quest_id}</Descriptions.Item>
            <Descriptions.Item label="任务标题" span={2}>{detail.quest_title}</Descriptions.Item>
            <Descriptions.Item label="状态">{detail.state}</Descriptions.Item>
            <Descriptions.Item label="追踪">{detail.tracked ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="已领奖">{detail.reward_claimed ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="接受时间">{formatDateTime(detail.accepted_at)}</Descriptions.Item>
            <Descriptions.Item label="完成时间">{formatDateTime(detail.completed_at)}</Descriptions.Item>
            <Descriptions.Item label="提交时间">{formatDateTime(detail.submitted_at)}</Descriptions.Item>
            <Descriptions.Item label="目标进度" span={2}><pre style={jsonBlockStyle}>{JSON.stringify(detail.objectives, null, 2)}</pre></Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>
      <Modal title={editingRecord ? `编辑玩家任务 · ${editingRecord.record_id}` : '新增玩家任务'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={760} okText={editingRecord ? '保存修改' : '创建记录'} cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            <Col xs={24} md={12}><Form.Item label="玩家ID" name="player_id" rules={[{ required: true, message: '请输入玩家ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item label="任务ID" name="quest_id" rules={[{ required: true, message: '请输入任务ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="任务状态" name="state"><Select options={editableQuestStateOptions} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="是否追踪" name="tracked" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="是否已领奖" name="reward_claimed" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col span={24}><Form.Item label="目标进度 JSON" name="objectives_text" extra='示例: [{"objective_id":1,"description":"与 NPC 对话","current_value":0,"target_value":1,"completed":false}]'><Input.TextArea rows={8} /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultTemplateValues(): TemplateFormValues {
  return {
    quest_id: 9001,
    name: 'ops_quest_template',
    quest_type: 'SIDE',
    title: '后台新任务',
    description: '用于后台创建测试任务模板。',
    chapter: 9,
    sort_order: 1,
    accept_mode: 'AUTO',
    submit_mode: 'AUTO',
    auto_track: true,
    start_npc_id: 0,
    submit_npc_id: 0,
    min_player_level: 1,
    status: 1,
    pre_quest_ids_text: '[]',
    objectives_text: JSON.stringify([{ objective_id: 1, event_type: 'ENTER_SCENE', description: '进入测试场景', target_value: 1, target_selector: { scene_id: 99 } }], null, 2),
  };
}

function defaultPlayerQuestValues(): PlayerQuestFormValues {
  return {
    player_id: 10001,
    quest_id: 1002,
    state: 'ACCEPTED',
    tracked: true,
    reward_claimed: false,
    objectives_text: JSON.stringify([{ objective_id: 1, description: '与市场理萌交谈', current_value: 0, target_value: 1, completed: false }], null, 2),
  };
}

function mapTemplateDetailToForm(detail: AdminQuestTemplateDetail): TemplateFormValues {
  return {
    quest_id: detail.quest_id,
    name: detail.name,
    quest_type: detail.quest_type,
    title: detail.title,
    description: detail.description,
    chapter: detail.chapter,
    sort_order: detail.sort_order,
    accept_mode: detail.accept_mode,
    submit_mode: detail.submit_mode,
    auto_track: detail.auto_track,
    start_npc_id: detail.start_npc_id,
    submit_npc_id: detail.submit_npc_id,
    min_player_level: detail.min_player_level,
    status: detail.status,
    pre_quest_ids_text: JSON.stringify(detail.pre_quest_ids, null, 2),
    objectives_text: JSON.stringify(detail.objectives, null, 2),
  };
}

function mapPlayerQuestDetailToForm(detail: AdminPlayerQuestDetail): PlayerQuestFormValues {
  return {
    player_id: detail.player_id,
    quest_id: detail.quest_id,
    state: detail.state,
    tracked: detail.tracked,
    reward_claimed: detail.reward_claimed,
    objectives_text: JSON.stringify(detail.objectives, null, 2),
  };
}

function mapTemplateFormToCreatePayload(values: TemplateFormValues): AdminCreateQuestTemplatePayload {
  return {
    quest_id: values.quest_id ?? 0,
    name: values.name,
    quest_type: values.quest_type,
    title: values.title,
    description: values.description,
    chapter: values.chapter,
    sort_order: values.sort_order,
    accept_mode: values.accept_mode,
    submit_mode: values.submit_mode,
    auto_track: values.auto_track,
    start_npc_id: values.start_npc_id,
    submit_npc_id: values.submit_npc_id,
    min_player_level: values.min_player_level,
    status: values.status,
    pre_quest_ids: parseJSONArray<number[]>(values.pre_quest_ids_text, []),
    objectives: parseJSONArray<AdminQuestObjectiveInput[]>(values.objectives_text, []),
  };
}

function mapTemplateFormToUpdatePayload(values: TemplateFormValues): AdminUpdateQuestTemplatePayload {
  const { quest_id: _questID, ...rest } = mapTemplateFormToCreatePayload(values);
  return rest;
}

function mapPlayerQuestFormToCreatePayload(values: PlayerQuestFormValues): AdminCreatePlayerQuestPayload {
  return {
    player_id: values.player_id,
    quest_id: values.quest_id,
    state: values.state,
    tracked: values.tracked,
    reward_claimed: values.reward_claimed,
    objectives: parseJSONArray<AdminPlayerQuestObjectiveInput[]>(values.objectives_text, []),
  };
}

function mapPlayerQuestFormToUpdatePayload(values: PlayerQuestFormValues): AdminUpdatePlayerQuestPayload {
  return mapPlayerQuestFormToCreatePayload(values);
}

function parseJSONArray<T>(text: string, fallback: T): T {
  try {
    return JSON.parse(text) as T;
  } catch {
    return fallback;
  }
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', { hour12: false });
}

const jsonBlockStyle: CSSProperties = {
  margin: 0,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  fontSize: 12,
  lineHeight: 1.5,
};
