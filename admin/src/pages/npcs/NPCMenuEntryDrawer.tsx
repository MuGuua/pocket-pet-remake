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
  Table,
  Tag,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import {
  createAdminNPCMenuEntry,
  deleteAdminNPCMenuEntry,
  fetchAdminNPCMenuEntries,
  fetchAdminNPCMenuEntryDetail,
  updateAdminNPCMenuEntry,
} from '../../services/npc';
import { fetchAdminMonsterEncounters } from '../../services/monsterEncounter';
import type { AdminMonsterEncounterSummary } from '../../types/monsterEncounter';
import type {
  AdminCreateNPCMenuEntryPayload,
  AdminNPCMenuEntryDetail,
  AdminNPCMenuEntryFilters,
  AdminNPCMenuEntrySummary,
  AdminUpdateNPCMenuEntryPayload,
} from '../../types/npc';

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
  battle_encounter_entity_id: number;
  status: number;
}

interface NPCMenuEntryDrawerProps {
  open: boolean;
  entityId: number | null;
  entityName: string;
  onClose: () => void;
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
  { label: '挑战 (battle)', value: 'battle' },
];

const actionResultTypeOptions = [
  { label: 'notice', value: 'notice' },
  { label: 'dialog', value: 'dialog' },
  { label: 'shop', value: 'shop' },
  { label: 'battle（直接开战）', value: 'battle' },
];

// 单个地图 NPC 的菜单配置抽屉：从实体编辑页进入，只维护当前 entity_id 下的菜单项。
export function NPCMenuEntryDrawer({ open, entityId, entityName, onClose }: NPCMenuEntryDrawerProps) {
  const [filterForm] = Form.useForm<Pick<AdminNPCMenuEntryFilters, 'entry_id' | 'status'>>();
  const [editorForm] = Form.useForm<MenuEntryFormValues>();
  const [filters, setFilters] = useState<Pick<AdminNPCMenuEntryFilters, 'entry_id' | 'status'>>({ status: '1' });
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
  const [encounterOptions, setEncounterOptions] = useState<{ label: string; value: number }[]>([]);
  const watchedEntryType = Form.useWatch('entry_type', editorForm);
  const watchedActionResultType = Form.useWatch('action_result_type', editorForm);
  const isBattleEntry = watchedEntryType === 'battle' || watchedActionResultType === 'battle';

  useEffect(() => {
    if (!open) {
      return;
    }
    void loadEncounterOptions();
  }, [open]);

  useEffect(() => {
    if (!open || !entityId) {
      return;
    }
    filterForm.setFieldsValue({ status: '1' });
    setFilters({ status: '1' });
    setPage(1);
  }, [open, entityId, filterForm]);

  async function loadEncounterOptions() {
    try {
      const result = await fetchAdminMonsterEncounters({ filters: { enabled: 'true' }, page: 1, pageSize: 100 });
      setEncounterOptions(result.items.map((item: AdminMonsterEncounterSummary) => ({
        label: `${item.entity_id} · ${item.encounter_name}`,
        value: item.entity_id,
      })));
    } catch {
      setEncounterOptions([]);
    }
  }

  useEffect(() => {
    if (!open || !entityId) {
      return;
    }
    void loadRows(entityId, filters, page, pageSize);
  }, [open, entityId, filters, page, pageSize]);

  async function loadRows(
    currentEntityId: number,
    nextFilters: Pick<AdminNPCMenuEntryFilters, 'entry_id' | 'status'>,
    nextPage: number,
    nextPageSize: number,
  ) {
    setLoading(true);
    try {
      const result = await fetchAdminNPCMenuEntries({
        filters: {
          entity_id: String(currentEntityId),
          entry_id: nextFilters.entry_id,
          status: nextFilters.status,
        },
        page: nextPage,
        pageSize: nextPageSize,
      });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 NPC 菜单失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(entryID: string) {
    if (!entityId) {
      return;
    }
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      setDetail(await fetchAdminNPCMenuEntryDetail(entityId, entryID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 NPC 菜单详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', entryID?: string) {
    if (!entityId) {
      return;
    }
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultMenuEntryValues(entityId));
      return;
    }
    if (!entryID) {
      return;
    }
    setDetailLoading(true);
    try {
      const result = await fetchAdminNPCMenuEntryDetail(entityId, entryID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapMenuEntryDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 NPC 菜单编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: MenuEntryFormValues) {
    if (!entityId) {
      return;
    }
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminNPCMenuEntry(
          editingRecord.entity_id,
          editingRecord.entry_id,
          mapMenuEntryFormToUpdatePayload(values),
        );
        message.success('NPC 菜单更新成功');
      } else {
        await createAdminNPCMenuEntry(mapMenuEntryFormToCreatePayload(values));
        message.success('NPC 菜单创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadRows(entityId, filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存 NPC 菜单失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(entryID: string) {
    if (!entityId) {
      return;
    }
    const key = `${entityId}:${entryID}`;
    setDeletingKey(key);
    try {
      await deleteAdminNPCMenuEntry(entityId, entryID);
      message.success('NPC 菜单已删除');
      if (detail?.entity_id === entityId && detail.entry_id === entryID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadRows(entityId, filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除 NPC 菜单失败');
    } finally {
      setDeletingKey(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminNPCMenuEntrySummary>>(() => [
    { title: '入口ID', dataIndex: 'entry_id', key: 'entry_id', width: 160, fixed: 'left' },
    { title: '入口类型', dataIndex: 'entry_type', key: 'entry_type', width: 120 },
    { title: '标题', dataIndex: 'title', key: 'title', width: 180 },
    {
      title: '状态',
      dataIndex: 'status_text',
      key: 'status_text',
      width: 100,
      render: (value: string) => <Tag color={value === '启用' ? 'green' : 'default'}>{value}</Tag>,
    },
    { title: '优先级/排序', key: 'sort', width: 120, render: (_value, record) => `${record.priority}/${record.sort_order}` },
    { title: '动作类型', dataIndex: 'action_result_type', key: 'action_result_type', width: 120 },
    {
      title: '固定战',
      dataIndex: 'battle_encounter_entity_id',
      key: 'battle_encounter_entity_id',
      width: 100,
      render: (value: number | undefined) => (value && value > 0 ? value : '-'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 220,
      fixed: 'right',
      render: (_value, record) => (
        <Space size="small">
          <Button type="link" onClick={() => void handleViewDetail(record.entry_id)}>查看</Button>
          <Button type="link" onClick={() => void handleOpenEditor('edit', record.entry_id)}>编辑</Button>
          <Popconfirm title="确认删除这条菜单配置吗？" onConfirm={() => void handleDelete(record.entry_id)}>
            <Button type="link" danger loading={deletingKey === `${record.entity_id}:${record.entry_id}`}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ], [deletingKey, entityId]);

  return (
    <>
      <Drawer
        title={entityId ? `菜单配置 · ${entityName}（${entityId}）` : '菜单配置'}
        width={960}
        open={open}
        onClose={onClose}
        destroyOnClose
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Card
            title="菜单筛选"
            extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增菜单项</Button>}
          >
            <Form
              form={filterForm}
              layout="vertical"
              onFinish={(values) => {
                setPage(1);
                setFilters(values);
              }}
            >
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <Form.Item label="入口ID" name="entry_id"><Input allowClear /></Form.Item>
                </Col>
                <Col xs={24} md={12}>
                  <Form.Item label="状态" name="status"><Select options={statusOptions} /></Form.Item>
                </Col>
              </Row>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button
                  onClick={() => {
                    filterForm.resetFields();
                    filterForm.setFieldsValue({ status: '1' });
                    setPage(1);
                    setFilters({ status: '1' });
                  }}
                >
                  重置
                </Button>
              </Space>
            </Form>
          </Card>
          <Card title="菜单列表">
            <Table
              columns={columns}
              dataSource={rows}
              rowKey={(record) => `${record.entity_id}:${record.entry_id}`}
              loading={loading}
              locale={{ emptyText: <Empty description="当前 NPC 还没有菜单配置" /> }}
              scroll={{ x: 1100 }}
              pagination={{
                current: page,
                pageSize,
                total,
                showSizeChanger: true,
                showTotal: (value) => `共 ${value} 条菜单配置`,
                onChange: (nextPage, nextPageSize) => {
                  setPage(nextPage);
                  setPageSize(nextPageSize);
                },
              }}
            />
          </Card>
        </Space>
      </Drawer>

      <Drawer
        title={detail ? `菜单详情 · ${detail.entry_id}` : '菜单详情'}
        width={560}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
      >
        {detailLoading ? (
          <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载菜单详情..." />
          </div>
        ) : detail ? (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="实体ID">{detail.entity_id}</Descriptions.Item>
            <Descriptions.Item label="入口ID">{detail.entry_id}</Descriptions.Item>
            <Descriptions.Item label="标题">{detail.title}</Descriptions.Item>
            <Descriptions.Item label="副标题">{detail.subtitle}</Descriptions.Item>
            <Descriptions.Item label="动作类型">{detail.action_result_type}</Descriptions.Item>
            <Descriptions.Item label="固定战 entity_id">{detail.battle_encounter_entity_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="提示文案">{detail.action_notice || '-'}</Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑菜单项 · ${editingRecord.entry_id}` : '新增菜单项'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingRecord(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={760}
        okText={editingRecord ? '保存修改' : '创建菜单项'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item label="实体ID" name="entity_id">
                <InputNumber min={1} style={{ width: '100%' }} disabled />
              </Form.Item>
            </Col>
            {!editingRecord ? (
              <Col xs={24} md={12}>
                <Form.Item label="入口ID" name="entry_id" rules={[{ required: true, message: '请输入入口ID' }]}>
                  <Input />
                </Form.Item>
              </Col>
            ) : null}
            <Col xs={24} md={12}>
              <Form.Item label="入口类型" name="entry_type">
                <Select
                  options={entryTypeOptions}
                  onChange={(value) => {
                    if (value === 'battle') {
                      editorForm.setFieldsValue({
                        action_result_type: 'battle',
                        title: editorForm.getFieldValue('title') || '挑战',
                      });
                    }
                  }}
                />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="标题" name="title"><Input /></Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="副标题" name="subtitle"><Input /></Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="状态 key" name="state"><Input /></Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="优先级" name="priority"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="排序" name="sort_order"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </Col>
            <Col xs={12} md={6}>
              <Form.Item label="动作类型" name="action_result_type"><Select options={actionResultTypeOptions} /></Form.Item>
            </Col>
            {isBattleEntry ? (
              <Col span={24}>
                <Form.Item
                  label="绑定 NPC 固定战"
                  name="battle_encounter_entity_id"
                  extra="选择 monster_encounter 配置；留空时保存后会默认使用当前 NPC 的 entity_id"
                  rules={[{ required: true, message: '请选择要绑定的固定战遭遇' }]}
                >
                  <Select
                    showSearch
                    optionFilterProp="label"
                    placeholder="选择固定战遭遇 entity_id"
                    options={encounterOptions}
                  />
                </Form.Item>
              </Col>
            ) : null}
            <Col xs={12} md={6}>
              <Form.Item label="状态" name="status"><Select options={editableStatusOptions} /></Form.Item>
            </Col>
            {!isBattleEntry ? (
              <Col span={24}>
                <Form.Item label="提示文案" name="action_notice"><Input.TextArea rows={4} /></Form.Item>
              </Col>
            ) : (
              <Col span={24}>
                <Form.Item label="提示文案（可选）" name="action_notice"><Input.TextArea rows={2} placeholder="挑战菜单通常无需提示文案，开战由服务端推送" /></Form.Item>
              </Col>
            )}
          </Row>
        </Form>
      </Modal>
    </>
  );
}

function defaultMenuEntryValues(entityId: number): MenuEntryFormValues {
  return {
    entity_id: entityId,
    entry_id: 'ops_dialog',
    entry_type: 'dialog',
    title: '后台菜单项',
    subtitle: '用于测试新入口',
    state: 'available',
    priority: 100,
    sort_order: 10,
    action_result_type: 'notice',
    action_notice: '这是一条后台新增提示。',
    battle_encounter_entity_id: entityId,
    status: 1,
  };
}

function mapMenuEntryDetailToForm(detail: AdminNPCMenuEntryDetail): MenuEntryFormValues {
  return {
    entity_id: detail.entity_id,
    entry_id: detail.entry_id,
    entry_type: detail.entry_type,
    title: detail.title,
    subtitle: detail.subtitle,
    state: detail.state,
    priority: detail.priority,
    sort_order: detail.sort_order,
    action_result_type: detail.action_result_type,
    action_notice: detail.action_notice,
    battle_encounter_entity_id: detail.battle_encounter_entity_id ?? detail.entity_id,
    status: detail.status,
  };
}

function mapMenuEntryFormToCreatePayload(values: MenuEntryFormValues): AdminCreateNPCMenuEntryPayload {
  return {
    entity_id: values.entity_id,
    entry_id: values.entry_id ?? '',
    entry_type: values.entry_type,
    title: values.title,
    subtitle: values.subtitle,
    state: values.state,
    priority: values.priority,
    sort_order: values.sort_order,
    action_result_type: values.action_result_type,
    action_notice: values.action_notice,
    battle_encounter_entity_id: values.battle_encounter_entity_id,
    status: values.status,
  };
}

function mapMenuEntryFormToUpdatePayload(values: MenuEntryFormValues): AdminUpdateNPCMenuEntryPayload {
  const { entry_id: _entryID, ...rest } = mapMenuEntryFormToCreatePayload(values);
  return rest;
}
