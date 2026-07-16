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
  Spin,
  Switch,
  Table,
  Tabs,
  Tag,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { QuestRewardEditor } from '../../components/QuestRewardEditor';
import { RichTextDisplay } from '../../components/RichTextDisplay';
import { RichTextEditor } from '../../components/RichTextEditor';
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
  AdminQuestRewardInput,
  AdminQuestTemplateDetail,
  AdminQuestTemplateFilters,
  AdminQuestTemplateSummary,
  AdminUpdatePlayerQuestPayload,
  AdminUpdateQuestTemplatePayload,
} from '../../types/quest';
import { buildFilterSelectOptions, buildSelectOptions, formatDisplayLabel, QUEST_EVENT_TYPE_LABELS, QUEST_MODE_LABELS, QUEST_STATE_LABELS, QUEST_TYPE_LABELS } from '../../utils/displayLabels';
import { formatDateTime } from '../../utils/formatDateTime';
import { QuestStageEditor } from './QuestStageEditor';
import { apiObjectivesToStages, createDefaultQuestStages, stagesToApiObjectives, type QuestStageFormItem } from './questStageUtils';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import { fetchAllAdminNPCEntities } from '../../services/npc';
import type { AdminNPCEntitySummary } from '../../types/npc';

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
  client_icon_id: number;
  start_npc_id: number;
  submit_npc_id: number;
  accept_animation_key: string;
  submit_animation_key: string;
  min_player_level: number;
  status: number;
  pre_quest_ids: number[];
  stages: QuestStageFormItem[];
  rewards: AdminQuestRewardInput[];
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

const questTypeOptions = buildSelectOptions(QUEST_TYPE_LABELS);

const questStateOptions = buildFilterSelectOptions(QUEST_STATE_LABELS, '全部状态');

const editableQuestStateOptions = buildSelectOptions(QUEST_STATE_LABELS);

const questModeOptions = buildSelectOptions(QUEST_MODE_LABELS);
const QUEST_TEMPLATE_EDITOR_PANEL_HEIGHT = 'calc(100vh - 260px)';
const QUEST_TEMPLATE_EDITOR_TAB_BAR_STYLE: CSSProperties = { marginBottom: 12 };

// 任务管理页把模板管理和玩家任务修正放到同一个页面，方便策划和运营在一个入口完成模板配置与进度排障。
export function QuestAdminPage() {
  return (
    <Tabs
        defaultActiveKey="templates"
        items={[
          { key: 'templates', label: '任务模板管理', children: <QuestTemplatePanel /> },
          { key: 'player-progress', label: '玩家任务进度', children: <PlayerQuestPanel /> },
        ]}
    />
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
  const [editorLoading, setEditorLoading] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminQuestTemplateDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [npcEntities, setNPCEntities] = useState<AdminNPCEntitySummary[]>([]);
  const editingQuestID: number = editingRecord?.quest_id ?? 0;
  const pendingEditorValuesRef = useRef<TemplateFormValues | null>(null);
  const npcOptions = useMemo(
    () => [
      { label: '无', value: 0 },
      ...npcEntities.map((entity) => ({
        label: `${entity.display_name || entity.entity_code}（${entity.entity_id}）`,
        value: entity.entity_id,
      })),
    ],
    [npcEntities],
  );

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
    setEditorLoading(true);
    if (mode === 'create') {
      try {
        setNPCEntities(await fetchAllAdminNPCEntities());
        setEditingRecord(null);
        pendingEditorValuesRef.current = defaultTemplateValues();
        setEditorOpen(true);
      } catch (error) {
        message.error(error instanceof Error ? error.message : '加载 NPC 选项失败');
      } finally {
        setEditorLoading(false);
      }
      return;
    }
    if (!questID) {
      setEditorLoading(false);
      return;
    }
    try {
      const [result, entities] = await Promise.all([
        fetchAdminQuestTemplateDetail(questID),
        fetchAllAdminNPCEntities(),
      ]);
      setNPCEntities(entities);
      setEditingRecord(result);
      pendingEditorValuesRef.current = mapTemplateDetailToForm(result);
      setEditorOpen(true);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载任务模板编辑数据失败');
    } finally {
      setEditorLoading(false);
    }
  }

  /** 模板编辑弹窗完全打开后再写入表单，避免 destroyOnClose 导致 setFieldsValue 失效。 */
  function handleEditorModalAfterOpenChange(open: boolean): void {
    if (open) {
      if (pendingEditorValuesRef.current) {
        editorForm.setFieldsValue(pendingEditorValuesRef.current);
        pendingEditorValuesRef.current = null;
      }
      return;
    }
    editorForm.resetFields();
    setEditingRecord(null);
    pendingEditorValuesRef.current = null;
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
      { title: '类型', dataIndex: 'quest_type', key: 'quest_type', width: 100, render: (value: string) => formatDisplayLabel(QUEST_TYPE_LABELS, value) },
      { title: '章节/排序', key: 'sort', width: 120, render: (_v, record) => `${record.chapter}/${record.sort_order}` },
      {
        title: '接/交模式',
        key: 'mode',
        width: 140,
        render: (_v, record) => `${formatDisplayLabel(QUEST_MODE_LABELS, record.accept_mode)}/${formatDisplayLabel(QUEST_MODE_LABELS, record.submit_mode)}`,
      },
      { title: '客户端图标ID', dataIndex: 'client_icon_id', key: 'client_icon_id', width: 120 },
      { title: '自动追踪', dataIndex: 'auto_track', key: 'auto_track', width: 100, render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '是' : '否'}</Tag> },
      { title: '状态', dataIndex: 'status_text', key: 'status_text', width: 100, render: (value: string) => <Tag color={value === '启用' ? 'blue' : 'default'}>{value}</Tag> },
      {
        title: '操作', key: 'actions', width: 100, fixed: 'right', render: (_v, record) => (
          <TableActionDropdown
            loading={deletingID === record.quest_id}
            actions={[
              { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.quest_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.quest_id), disabled: editorLoading },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: {
                  title: '确认停用这个任务模板吗？',
                  description: '这里会把模板状态改成停用，不直接物理删除。',
                  okText: '确认停用',
                  cancelText: '取消',
                },
                onClick: () => void handleDelete(record.quest_id),
              },
            ]}
          />
        ),
      },
    ],
    [deletingID, editorLoading],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title="模板列表"
        extra={(
          <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }}>
            <Form.Item name="quest_id" label="任务ID">
              <Input allowClear placeholder="任务ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="quest_type" label="类型">
              <Select allowClear placeholder="类型" style={{ width: 100 }} options={questTypeOptions} />
            </Form.Item>
            <Form.Item name="title" label="标题">
              <Input allowClear placeholder="标题" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select options={templateStatusOptions} style={{ width: 100 }} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button onClick={() => { filterForm.resetFields(); filterForm.setFieldsValue({ status: '1' }); setPage(1); setFilters({ status: '1' }); }}>重置</Button>
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增模板</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
        <Table columns={columns} dataSource={rows} rowKey="quest_id" loading={loading} locale={{ emptyText: <Empty description="当前筛选条件下没有任务模板" /> }} scroll={{ x: 1400 }} pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (value) => `共 ${value} 个模板`, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} />
      </Card>
      <Drawer title={detail ? `任务模板详情 · ${detail.quest_id}` : '任务模板详情'} width={640} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading ? <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}><Spin tip="正在加载任务模板详情..." /></div> : detail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="任务ID">{detail.quest_id}</Descriptions.Item>
            <Descriptions.Item label="模板名">{detail.name}</Descriptions.Item>
            <Descriptions.Item label="标题" span={2}>{detail.title}</Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              <RichTextDisplay value={detail.description} />
            </Descriptions.Item>
            <Descriptions.Item label="类型">{formatDisplayLabel(QUEST_TYPE_LABELS, detail.quest_type)}</Descriptions.Item>
            <Descriptions.Item label="状态">{detail.status_text}</Descriptions.Item>
            <Descriptions.Item label="接取方式">{formatDisplayLabel(QUEST_MODE_LABELS, detail.accept_mode)}</Descriptions.Item>
            <Descriptions.Item label="提交方式">{formatDisplayLabel(QUEST_MODE_LABELS, detail.submit_mode)}</Descriptions.Item>
            <Descriptions.Item label="客户端图标ID">{detail.client_icon_id}</Descriptions.Item>
            <Descriptions.Item label="起始 NPC">{detail.start_npc_id}</Descriptions.Item>
            <Descriptions.Item label="提交 NPC">{detail.submit_npc_id}</Descriptions.Item>
            <Descriptions.Item label="领取动画键">{detail.accept_animation_key || '不播放'}</Descriptions.Item>
            <Descriptions.Item label="交付动画键">{detail.submit_animation_key || '不播放'}</Descriptions.Item>
            <Descriptions.Item label="前置任务" span={2}>{detail.pre_quest_ids.length > 0 ? detail.pre_quest_ids.join(', ') : '无'}</Descriptions.Item>
            <Descriptions.Item label="任务阶段" span={2}>
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {detail.objectives.map((stage) => (
                  <Card key={stage.objective_id} size="small" title={`阶段 ${stage.objective_id} · ${stage.description}`}>
                    <Descriptions bordered column={2} size="small">
                      <Descriptions.Item label="事件类型">{formatDisplayLabel(QUEST_EVENT_TYPE_LABELS, stage.event_type)}</Descriptions.Item>
                      <Descriptions.Item label="目标次数">{stage.target_value}</Descriptions.Item>
                      <Descriptions.Item label="目标选择器" span={2}><pre style={jsonBlockStyle}>{JSON.stringify(stage.target_selector ?? {}, null, 2)}</pre></Descriptions.Item>
                      <Descriptions.Item label="引导/绑定" span={2}><pre style={jsonBlockStyle}>{JSON.stringify(stage.guide ?? {}, null, 2)}</pre></Descriptions.Item>
                    </Descriptions>
                  </Card>
                ))}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="奖励配置" span={2}><pre style={jsonBlockStyle}>{JSON.stringify(detail.rewards ?? [], null, 2)}</pre></Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>
      <Modal
        title={editingRecord ? `编辑任务模板 · ${editingRecord.quest_id}` : '新增任务模板'}
        open={editorOpen}
        afterOpenChange={handleEditorModalAfterOpenChange}
        onCancel={() => { setEditorOpen(false); }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={820}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={editingRecord ? '保存修改' : '创建模板'}
        cancelText="取消"
      >
        <Spin spinning={editorLoading} tip="正在加载任务模板...">
          <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
            <Tabs
              size="small"
              tabBarStyle={QUEST_TEMPLATE_EDITOR_TAB_BAR_STYLE}
              items={[
                {
                  key: 'basic',
                  label: '基础信息',
                  children: (
                    <div style={buildEditorPanelStyle(QUEST_TEMPLATE_EDITOR_PANEL_HEIGHT)}>
                      <Space direction="vertical" size={16} style={{ width: '100%' }}>
                        <Card size="small" title="任务基础资料">
                          <Row gutter={16}>
                            {!editingRecord ? <Col span={24}><Form.Item label="任务ID" extra="新增时由服务端自动分配，无需手动填写。"><Input value="保存后自动生成" disabled /></Form.Item></Col> : null}
                            <Col xs={24} md={12}><Form.Item label="模板名" name="name" rules={[{ required: true, message: '请输入模板名' }]} extra="建议使用稳定英文 key，便于程序与运营排查。"><Input placeholder="例如：market_intro_quest" /></Form.Item></Col>
                            <Col xs={24} md={12}><Form.Item label="任务类型" name="quest_type" rules={[{ required: true, message: '请选择任务类型' }]}><Select options={questTypeOptions} /></Form.Item></Col>
                            <Col span={24}><Form.Item label="标题" name="title" rules={[{ required: true, message: '请输入标题' }]} extra="展示给玩家的正式任务标题。"><Input placeholder="例如：初识市场理萌" /></Form.Item></Col>
                            <Col span={24}>
                              <Form.Item label="描述" name="description" extra="可在下方预览中刷色，客户端任务详情会保持同样效果。">
                                <RichTextEditor rows={5} />
                              </Form.Item>
                            </Col>
                          </Row>
                        </Card>
                        <Card size="small" title="接取与提交流程">
                          <Row gutter={16}>
                            <Col xs={12} md={6}><Form.Item label="章节" name="chapter" extra="用于章节分组展示。"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="排序" name="sort_order" extra="同章节内排序值。"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="接取方式" name="accept_mode"><Select options={questModeOptions} /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="提交方式" name="submit_mode"><Select options={questModeOptions} /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="客户端图标ID" name="client_icon_id" extra="由客户端 TaskIcons 注册表解释，多个任务可共用。"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="起始 NPC" name="start_npc_id"><Select showSearch optionFilterProp="label" options={npcOptions} /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="目标 NPC" name="submit_npc_id"><Select showSearch optionFilterProp="label" options={npcOptions} /></Form.Item></Col>
                            <Col xs={24} md={12}><Form.Item label="领取动画键" name="accept_animation_key" rules={[{ pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5-]*$/, message: '只能填写动画注册键' }]}><Input allowClear /></Form.Item></Col>
                            <Col xs={24} md={12}><Form.Item label="交付动画键" name="submit_animation_key" rules={[{ pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5-]*$/, message: '只能填写动画注册键' }]}><Input allowClear /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="最低等级" name="min_player_level"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                            <Col xs={12} md={6}><Form.Item label="状态" name="status"><Select options={editableTemplateStatusOptions} /></Form.Item></Col>
                            <Col xs={24} md={8}><Form.Item label="自动追踪" name="auto_track" valuePropName="checked" extra="接取后是否默认开启追踪。"><Switch /></Form.Item></Col>
                            <Col span={24}>
                              <Form.Item label="前置任务" extra="按顺序添加前置任务 ID；不需要前置任务时可留空。">
                                <Form.List name="pre_quest_ids">
                                  {(fields, { add, remove }) => (
                                    <Space direction="vertical" size={8} style={{ width: '100%' }}>
                                      {fields.length > 0 ? fields.map((field, index) => (
                                        <Space key={field.key} align="start" style={{ display: 'flex' }}>
                                          <Form.Item
                                            {...field}
                                            label={index === 0 ? '前置任务 ID' : ' '}
                                            rules={[{ required: true, message: '请输入任务ID' }]}
                                            style={{ marginBottom: 0, minWidth: 220 }}
                                          >
                                            <InputNumber min={1} style={{ width: '100%' }} placeholder="例如：1001" />
                                          </Form.Item>
                                          <Button danger onClick={() => remove(field.name)}>
                                            删除
                                          </Button>
                                        </Space>
                                      )) : (
                                        <span style={{ color: '#8c8c8c', fontSize: 12 }}>当前没有前置任务，任务可独立接取。</span>
                                      )}
                                      <Button type="dashed" onClick={() => add(0)}>
                                        添加前置任务
                                      </Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Form.Item>
                            </Col>
                          </Row>
                        </Card>
                      </Space>
                    </div>
                  ),
                },
                {
                  key: 'stages',
                  label: '任务阶段',
                  children: (
                    <div style={buildEditorPanelStyle(QUEST_TEMPLATE_EDITOR_PANEL_HEIGHT)}>
                      <Card size="small" title="阶段配置" extra={<span style={{ color: '#8c8c8c', fontSize: 12 }}>先补全基础信息，再按阶段推进链路配置 NPC / 菜单 / 剧情</span>}>
                        <Form.Item
                          label="任务阶段"
                          name="stages"
                          rules={[
                            {
                              validator: async (_, stageList: QuestStageFormItem[] | undefined) => {
                                if (!stageList || stageList.length === 0) {
                                  throw new Error('至少配置一个任务阶段');
                                }
                              },
                            },
                          ]}
                          tooltip="同一任务可配置多个阶段；每阶段可绑定不同 NPC、菜单 entry 与剧情。点击「添加阶段」在弹窗中编辑。"
                        >
                          <QuestStageEditor questID={editingQuestID} npcOptions={npcOptions} />
                        </Form.Item>
                      </Card>
                    </div>
                  ),
                },
                {
                  key: 'rewards',
                  label: '任务奖励',
                  children: (
                    <div style={buildEditorPanelStyle(QUEST_TEMPLATE_EDITOR_PANEL_HEIGHT)}>
                      <Card size="small" title="奖励配置" extra={<span style={{ color: '#8c8c8c', fontSize: 12 }}>配置完成任务后发放给玩家的奖励</span>}>
                        <Form.Item label="任务奖励" name="rewards">
                          <QuestRewardEditor />
                        </Form.Item>
                      </Card>
                    </div>
                  ),
                },
              ]}
            />
          </Form>
        </Spin>
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
      {
        title: '状态',
        dataIndex: 'state',
        key: 'state',
        width: 150,
        render: (value: string) => (
          <Tag color={value === 'COMPLETED' ? 'green' : 'blue'}>{formatDisplayLabel(QUEST_STATE_LABELS, value)}</Tag>
        ),
      },
      { title: '追踪', dataIndex: 'tracked', key: 'tracked', width: 90, render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '是' : '否'}</Tag> },
      { title: '已领奖', dataIndex: 'reward_claimed', key: 'reward_claimed', width: 100, render: (value: boolean) => <Tag color={value ? 'gold' : 'default'}>{value ? '是' : '否'}</Tag> },
      {
        title: '操作', key: 'actions', width: 100, fixed: 'right', render: (_v, record) => (
          <TableActionDropdown
            loading={deletingID === record.record_id}
            actions={[
              { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.record_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.record_id) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这条玩家任务记录吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => void handleDelete(record.record_id),
              },
            ]}
          />
        ),
      },
    ],
    [deletingID],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title="玩家任务列表"
        extra={(
          <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }}>
            <Form.Item name="record_id" label="记录ID">
              <Input allowClear placeholder="记录ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="player_id" label="玩家ID">
              <Input allowClear placeholder="玩家ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="quest_id" label="任务ID">
              <Input allowClear placeholder="任务ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="state" label="状态">
              <Select options={questStateOptions} style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="tracked" label="追踪">
              <Select allowClear placeholder="追踪" style={{ width: 90 }} options={[{ label: '全部', value: '' }, { label: '是', value: 'true' }, { label: '否', value: 'false' }]} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增玩家任务</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
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
            <Descriptions.Item label="状态">{formatDisplayLabel(QUEST_STATE_LABELS, detail.state)}</Descriptions.Item>
            <Descriptions.Item label="追踪">{detail.tracked ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="已领奖">{detail.reward_claimed ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="接受时间">{formatDateTime(detail.accepted_at)}</Descriptions.Item>
            <Descriptions.Item label="完成时间">{formatDateTime(detail.completed_at)}</Descriptions.Item>
            <Descriptions.Item label="提交时间">{formatDateTime(detail.submitted_at)}</Descriptions.Item>
            <Descriptions.Item label="目标进度" span={2}><pre style={jsonBlockStyle}>{JSON.stringify(detail.objectives, null, 2)}</pre></Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>
      <Modal title={editingRecord ? `编辑玩家任务 · ${editingRecord.record_id}` : '新增玩家任务'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={760} style={{ top: FIXED_FORM_MODAL_TOP }} styles={FIXED_FORM_MODAL_STYLES} okText={editingRecord ? '保存修改' : '创建记录'} cancelText="取消">
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
    name: 'ops_quest_template',
    quest_type: 'SIDE',
    title: '后台新任务',
    description: '用于后台创建测试任务模板。',
    chapter: 9,
    sort_order: 1,
    accept_mode: 'AUTO',
    submit_mode: 'AUTO',
    auto_track: true,
    client_icon_id: 1,
    start_npc_id: 0,
    submit_npc_id: 0,
    accept_animation_key: '',
    submit_animation_key: '',
    min_player_level: 1,
    status: 1,
    pre_quest_ids: [],
    stages: createDefaultQuestStages(),
    rewards: [{ type: 'exp', value: 50, item_id: 0, count: 0 }],
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
    client_icon_id: detail.client_icon_id || 1,
    start_npc_id: detail.start_npc_id,
    submit_npc_id: detail.submit_npc_id,
    accept_animation_key: detail.accept_animation_key ?? '',
    submit_animation_key: detail.submit_animation_key ?? '',
    min_player_level: detail.min_player_level,
    status: detail.status,
    pre_quest_ids: detail.pre_quest_ids ?? [],
    stages: apiObjectivesToStages(detail.objectives),
    rewards: detail.rewards ?? [],
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
    quest_id: values.quest_id,
    name: values.name,
    quest_type: values.quest_type,
    title: values.title,
    description: values.description,
    chapter: values.chapter,
    sort_order: values.sort_order,
    accept_mode: values.accept_mode,
    submit_mode: values.submit_mode,
    auto_track: values.auto_track,
    client_icon_id: values.client_icon_id || 1,
    start_npc_id: values.start_npc_id,
    submit_npc_id: values.submit_npc_id,
    accept_animation_key: values.accept_animation_key?.trim() ?? '',
    submit_animation_key: values.submit_animation_key?.trim() ?? '',
    min_player_level: values.min_player_level,
    status: values.status,
    pre_quest_ids: normalizePositiveNumberList(values.pre_quest_ids),
    objectives: stagesToApiObjectives(values.stages ?? []),
    rewards: values.rewards ?? [],
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

function normalizePositiveNumberList(values: number[] | undefined): number[] {
  if (!values || values.length === 0) {
    return [];
  }
  return values
    .map((value) => Number(value ?? 0))
    .filter((value) => Number.isFinite(value) && value > 0);
}

const jsonBlockStyle: CSSProperties = {
  margin: 0,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  fontSize: 12,
  lineHeight: 1.5,
};

function buildEditorPanelStyle(maxHeight: string): CSSProperties {
  return {
    maxHeight,
    overflowY: 'auto',
    paddingRight: 4,
  };
}
