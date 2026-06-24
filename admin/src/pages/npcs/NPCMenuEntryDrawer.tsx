import {
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
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
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { NPCDialogueConfigDrawer } from './NPCDialogueConfigDrawer';
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
  AdminDialogueConditions,
  AdminNPCMenuEntryDetail,
  AdminNPCMenuEntrySummary,
  AdminUpdateNPCMenuEntryPayload,
} from '../../types/npc';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import {
  buildSelectOptions,
  formatDisplayLabel,
  NPC_ACTION_RESULT_LABELS,
  NPC_ENTRY_TYPE_LABELS,
  NPC_MENU_STATE_LABELS,
} from '../../utils/displayLabels';

interface MenuEntryFormValues {
  entity_id: number;
  entry_id?: string;
  entry_type: string;
  title: string;
  subtitle: string;
  state: string;
  action_result_type: string;
  action_notice: string;
  battle_encounter_entity_id: number;
  linked_quest_id: number;
  conditions: AdminDialogueConditions;
  status: number;
}

interface NPCMenuEntryDrawerProps {
  open: boolean;
  entityId: number | null;
  entityName: string;
  onClose: () => void;
}

const editableStatusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

const entryTypeOptions = buildSelectOptions(NPC_ENTRY_TYPE_LABELS);

const actionResultTypeOptions = buildSelectOptions(NPC_ACTION_RESULT_LABELS);

const menuStateOptions = buildSelectOptions(NPC_MENU_STATE_LABELS);

const questVisibilityStateOptions: Array<{ label: string; value: string }> = [
  { label: '不限制', value: '' },
  { label: '可接取 AVAILABLE', value: 'AVAILABLE' },
  { label: '进行中 ACCEPTED', value: 'ACCEPTED' },
  { label: '可提交 READY_TO_SUBMIT', value: 'READY_TO_SUBMIT' },
  { label: '已完成 COMPLETED', value: 'COMPLETED' },
];

const menuObjectiveCompletedOptions: Array<{ label: string; value: string }> = [
  { label: '不限制', value: '' },
  { label: '要求目标已完成', value: 'true' },
  { label: '要求目标未完成', value: 'false' },
];

const NPC_DIALOGUE_EMBEDDED_FORM_ID = 'npc-dialogue-embedded-editor-form';

// 单个地图 NPC 的菜单与剧情统一配置抽屉。
export function NPCMenuEntryDrawer({ open, entityId, entityName, onClose }: NPCMenuEntryDrawerProps) {
  const [editorForm] = Form.useForm<MenuEntryFormValues>();
  const [rows, setRows] = useState<AdminNPCMenuEntrySummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [draggingEntryID, setDraggingEntryID] = useState<string | null>(null);
  const [dragOverEntryID, setDragOverEntryID] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminNPCMenuEntryDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminNPCMenuEntryDetail | null>(null);
  const [insertAfterEntryID, setInsertAfterEntryID] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingKey, setDeletingKey] = useState<string | null>(null);
  const [encounterOptions, setEncounterOptions] = useState<{ label: string; value: number }[]>([]);
  const [editorTab, setEditorTab] = useState<string>('menu');
  const [dialogueEditing, setDialogueEditing] = useState(false);
  const watchedEntryType = Form.useWatch('entry_type', editorForm);
  const watchedActionResultType = Form.useWatch('action_result_type', editorForm);
  const watchedEntryID = Form.useWatch('entry_id', editorForm);
  const isBattleEntry = watchedEntryType === 'battle' || watchedActionResultType === 'battle';
  const isDialogEntry = watchedEntryType === 'dialog' || watchedActionResultType === 'dialogue' || watchedActionResultType === 'dialog';
  const isQuestEntry = watchedEntryType === 'quest' || watchedActionResultType === 'quest_accept' || watchedActionResultType === 'quest_submit';
  const isShopEntry = watchedEntryType === 'shop' || watchedActionResultType === 'shop';
  const dialogueEntryID: string = editingRecord?.entry_id ?? watchedEntryID ?? '';

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
    setPage(1);
  }, [open, entityId]);

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
    void loadRows(entityId, page, pageSize);
  }, [open, entityId, page, pageSize]);

  async function loadRows(
    currentEntityId: number,
    nextPage: number,
    nextPageSize: number,
  ) {
    setLoading(true);
    try {
      const result = await fetchAdminNPCMenuEntries({
        filters: {
          entity_id: String(currentEntityId),
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

  // 顺序由前端列表托管，这里把当前列表顺序重新写回 sort_order，避免依赖人工填写排序字段。
  async function applyMenuEntryOrder(currentEntityId: number, orderedEntryIDs: string[]): Promise<void> {
    for (let index = 0; index < orderedEntryIDs.length; index += 1) {
      const entryID: string = orderedEntryIDs[index];
      const detailPayload: AdminNPCMenuEntryDetail = await fetchAdminNPCMenuEntryDetail(currentEntityId, entryID);
      await updateAdminNPCMenuEntry(
        currentEntityId,
        entryID,
        mapMenuEntryDetailToOrderedPayload(detailPayload, index + 1),
      );
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

  async function handleOpenEditor(mode: 'create' | 'edit', entryID?: string, initialTab: string = 'menu', insertAfterID?: string) {
    if (!entityId) {
      return;
    }
    setEditorOpen(true);
    setEditorTab(initialTab);
    if (mode === 'create') {
      setEditingRecord(null);
      setInsertAfterEntryID(insertAfterID ?? null);
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
      setInsertAfterEntryID(null);
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
        await applyMenuEntryOrder(entityId, rows.map((row: AdminNPCMenuEntrySummary) => row.entry_id));
        setInsertAfterEntryID(null);
        message.success('NPC 菜单更新成功');
      } else {
        const createdEntryID: string = values.entry_id ?? '';
        await createAdminNPCMenuEntry(mapMenuEntryFormToCreatePayload(values));
        await applyMenuEntryOrder(entityId, buildMenuOrderAfterCreate(rows, createdEntryID, insertAfterEntryID));
        setInsertAfterEntryID(null);
        message.success('NPC 菜单创建成功');
        if (createdEntryID && menuEntryUsesDialogue(values)) {
          const createdDetail: AdminNPCMenuEntryDetail = await fetchAdminNPCMenuEntryDetail(entityId, createdEntryID);
          setEditingRecord(createdDetail);
          setInsertAfterEntryID(null);
          editorForm.setFieldsValue(mapMenuEntryDetailToForm(createdDetail));
          setEditorTab('dialogue');
          await loadRows(entityId, page, pageSize);
          return;
        }
      }
      setEditorOpen(false);
      setEditingRecord(null);
      setEditorTab('menu');
      editorForm.resetFields();
      await loadRows(entityId, page, pageSize);
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
      await applyMenuEntryOrder(entityId, rows.filter((row: AdminNPCMenuEntrySummary) => row.entry_id !== entryID).map((row: AdminNPCMenuEntrySummary) => row.entry_id));
      await loadRows(entityId, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除 NPC 菜单失败');
    } finally {
      setDeletingKey(null);
    }
  }

  // 菜单顺序直接由当前列表控制；上移/下移后立刻整体回写 sort_order。
  async function handleMoveMenuEntry(entryID: string, direction: 'up' | 'down'): Promise<void> {
    if (!entityId) {
      return;
    }
    const currentOrder: string[] = rows.map((row: AdminNPCMenuEntrySummary) => row.entry_id);
    const currentIndex: number = currentOrder.findIndex((value: string) => value === entryID);
    if (currentIndex < 0) {
      return;
    }
    const targetIndex: number = direction === 'up' ? currentIndex - 1 : currentIndex + 1;
    if (targetIndex < 0 || targetIndex >= currentOrder.length) {
      return;
    }
    const nextOrder: string[] = [...currentOrder];
    [nextOrder[currentIndex], nextOrder[targetIndex]] = [nextOrder[targetIndex], nextOrder[currentIndex]];
    setLoading(true);
    try {
      await applyMenuEntryOrder(entityId, nextOrder);
      await loadRows(entityId, page, pageSize);
      message.success(direction === 'up' ? '菜单项已上移' : '菜单项已下移');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '调整菜单顺序失败');
      setLoading(false);
    }
  }

  // 表格行拖拽后，直接以拖拽结果作为新的菜单顺序并整体回写。
  async function handleDropMenuEntry(targetEntryID: string): Promise<void> {
    if (!entityId || !draggingEntryID || draggingEntryID === targetEntryID) {
      setDraggingEntryID(null);
      setDragOverEntryID(null);
      return;
    }
    const currentOrder: string[] = rows.map((row: AdminNPCMenuEntrySummary) => row.entry_id);
    const draggingIndex: number = currentOrder.findIndex((entryID: string) => entryID === draggingEntryID);
    const targetIndex: number = currentOrder.findIndex((entryID: string) => entryID === targetEntryID);
    if (draggingIndex < 0 || targetIndex < 0) {
      setDraggingEntryID(null);
      setDragOverEntryID(null);
      return;
    }
    const nextOrder: string[] = [...currentOrder];
    const [draggingValue] = nextOrder.splice(draggingIndex, 1);
    nextOrder.splice(targetIndex, 0, draggingValue);
    setLoading(true);
    try {
      await applyMenuEntryOrder(entityId, nextOrder);
      await loadRows(entityId, page, pageSize);
      message.success('菜单顺序已更新');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '拖拽调整菜单顺序失败');
      setLoading(false);
    } finally {
      setDraggingEntryID(null);
      setDragOverEntryID(null);
    }
  }

  function handleCloseEditor() {
    setEditorOpen(false);
    setEditingRecord(null);
    setInsertAfterEntryID(null);
    setEditorTab('menu');
    setDialogueEditing(false);
    editorForm.resetFields();
  }

  const dialogueEntryTitle: string = editingRecord?.title ?? editorForm.getFieldValue('title') ?? '';

  const columns = useMemo<ColumnsType<AdminNPCMenuEntrySummary>>(() => [
    { title: '入口ID', dataIndex: 'entry_id', key: 'entry_id', width: 160, fixed: 'left' },
    { title: '入口类型', dataIndex: 'entry_type', key: 'entry_type', width: 120, render: (value: string) => formatDisplayLabel(NPC_ENTRY_TYPE_LABELS, value) },
    { title: '标题', dataIndex: 'title', key: 'title', width: 180 },
    {
      title: '状态',
      dataIndex: 'status_text',
      key: 'status_text',
      width: 100,
      render: (value: string) => <Tag color={value === '启用' ? 'green' : 'default'}>{value}</Tag>,
    },
    { title: '当前顺序', key: 'sort', width: 100, render: (_value, _record, index) => index + 1 },
    { title: '动作类型', dataIndex: 'action_result_type', key: 'action_result_type', width: 120, render: (value: string) => formatDisplayLabel(NPC_ACTION_RESULT_LABELS, value) },
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
      width: 100,
      fixed: 'right',
      render: (_value, record, index) => (
        <TableActionDropdown
          loading={deletingKey === `${record.entity_id}:${record.entry_id}`}
          actions={[
            { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.entry_id) },
            { key: 'edit', label: '菜单与剧情', onClick: () => void handleOpenEditor('edit', record.entry_id) },
            { key: 'move_up', label: '上移', disabled: index === 0, onClick: () => void handleMoveMenuEntry(record.entry_id, 'up') },
            { key: 'move_down', label: '下移', disabled: index === rows.length - 1, onClick: () => void handleMoveMenuEntry(record.entry_id, 'down') },
            { key: 'insert_below', label: '下方插入', onClick: () => void handleOpenEditor('create', undefined, 'menu', record.entry_id) },
            ...((record.entry_type === 'dialog' || record.action_result_type === 'dialog' || record.action_result_type === 'dialogue') ? [{
              key: 'dialogue',
              label: '剧情配置',
              onClick: () => void handleOpenEditor('edit', record.entry_id, 'dialogue'),
            }] : []),
            {
              key: 'delete',
              label: '删除',
              danger: true,
              confirm: { title: '确认删除这条菜单配置吗？' },
              onClick: () => void handleDelete(record.entry_id),
            },
          ]}
        />
      ),
    },
  ], [deletingKey, entityId, rows, page, pageSize]);

  return (
    <>
      <Drawer
        title={entityId ? `NPC菜单配置 · ${entityName}（${entityId}）` : 'NPC菜单配置'}
        width={960}
        open={open}
        onClose={onClose}
        destroyOnClose
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Card title="菜单列表" extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增菜单项</Button>}>
            <Table
              columns={columns}
              dataSource={rows}
              rowKey={(record) => `${record.entity_id}:${record.entry_id}`}
              loading={loading}
              onRow={(record) => ({
                draggable: true,
                onDragStart: (event) => {
                  event.dataTransfer.effectAllowed = 'move';
                  setDraggingEntryID(record.entry_id);
                },
                onDragOver: (event) => {
                  event.preventDefault();
                  if (draggingEntryID !== record.entry_id) {
                    setDragOverEntryID(record.entry_id);
                  }
                },
                onDragLeave: () => {
                  if (dragOverEntryID === record.entry_id) {
                    setDragOverEntryID(null);
                  }
                },
                onDrop: (event) => {
                  event.preventDefault();
                  void handleDropMenuEntry(record.entry_id);
                },
                onDragEnd: () => {
                  setDraggingEntryID(null);
                  setDragOverEntryID(null);
                },
                style: dragOverEntryID === record.entry_id
                  ? { background: '#fff7e6', outline: '1px dashed #fa8c16', cursor: 'move' }
                  : { cursor: 'move' },
              })}
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
            <Descriptions.Item label="入口类型">{formatDisplayLabel(NPC_ENTRY_TYPE_LABELS, detail.entry_type)}</Descriptions.Item>
            <Descriptions.Item label="菜单状态">{formatDisplayLabel(NPC_MENU_STATE_LABELS, detail.state)}</Descriptions.Item>
            <Descriptions.Item label="动作类型">{formatDisplayLabel(NPC_ACTION_RESULT_LABELS, detail.action_result_type)}</Descriptions.Item>
            <Descriptions.Item label="固定战实体ID">{detail.battle_encounter_entity_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="提示文案">{detail.action_notice || '-'}</Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>

      <Modal
        title={editingRecord ? `菜单与剧情 · ${editingRecord.entry_id}` : '新增菜单项'}
        open={editorOpen}
        onCancel={handleCloseEditor}
        onOk={() => {
          if (editorTab === 'menu') {
            editorForm.submit();
          }
        }}
        confirmLoading={saving && editorTab === 'menu'}
        destroyOnClose
        width={1080}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={editorTab === 'menu' ? (editingRecord ? '保存菜单' : '创建菜单项') : undefined}
        cancelText="关闭"
        footer={editorTab === 'dialogue' ? [
          <Button
            key="save-dialogue"
            type="primary"
            htmlType="submit"
            form={NPC_DIALOGUE_EMBEDDED_FORM_ID}
            disabled={!dialogueEditing}
          >
            保存剧情
          </Button>,
          <Button key="close" onClick={handleCloseEditor}>关闭</Button>,
        ] : undefined}
      >
        <Tabs
          activeKey={editorTab}
          onChange={(nextTab) => {
            setEditorTab(nextTab);
            if (nextTab !== 'dialogue') {
              setDialogueEditing(false);
            }
          }}
          items={[
            {
              key: 'menu',
              label: '菜单配置',
              children: (
                <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
                  <Typography.Paragraph type="secondary">
                    支持对话、任务、商店、挑战四类入口；可通过可见条件按任务状态或分阶段目标控制菜单显示。
                  </Typography.Paragraph>
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
                          showSearch
                          optionFilterProp="label"
                          options={entryTypeOptions}
                          onChange={(value) => {
                            if (value === 'battle') {
                              editorForm.setFieldsValue({
                                action_result_type: 'battle',
                                title: editorForm.getFieldValue('title') || '挑战',
                              });
                            }
                            if (value === 'dialog') {
                              editorForm.setFieldsValue({ action_result_type: 'dialogue' });
                            }
                            if (value === 'shop') {
                              editorForm.setFieldsValue({
                                action_result_type: 'shop',
                                title: editorForm.getFieldValue('title') || '商店',
                              });
                            }
                            if (value === 'quest') {
                              editorForm.setFieldsValue({
                                action_result_type: 'quest_accept',
                                title: editorForm.getFieldValue('title') || '任务',
                              });
                            }
                            if (value === 'warehouse') {
                              editorForm.setFieldsValue({
                                action_result_type: 'panel',
                                title: editorForm.getFieldValue('title') || '打开仓库',
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
                      <Form.Item
                        label="菜单状态"
                        name="state"
                        tooltip="写入 state 字段；多数菜单使用「可用」。若数据库中有未收录的状态值，下拉会保留原值。"
                      >
                        <Select showSearch optionFilterProp="label" options={menuStateOptions} />
                      </Form.Item>
                    </Col>
                    <Col xs={12} md={6}>
                      <Form.Item label="当前顺序">
                        <Input value={describeMenuInsertPosition(rows, editingRecord?.entry_id ?? null, insertAfterEntryID)} disabled />
                      </Form.Item>
                    </Col>
                    <Col xs={12} md={6}>
                      <Form.Item label="动作类型" name="action_result_type">
                        <Select showSearch optionFilterProp="label" options={actionResultTypeOptions} />
                      </Form.Item>
                    </Col>
                    {isBattleEntry ? (
                      <Col span={24}>
                        <Form.Item
                          label="绑定 NPC 固定战"
                          name="battle_encounter_entity_id"
                          extra="选择怪物固定战遭遇配置；留空时保存后会默认使用当前 NPC 的实体 ID"
                          rules={[{ required: true, message: '请选择要绑定的固定战遭遇' }]}
                        >
                          <Select showSearch optionFilterProp="label" placeholder="选择固定战遭遇" options={encounterOptions} />
                        </Form.Item>
                      </Col>
                    ) : null}
                    {isQuestEntry ? (
                      <Col xs={24} md={12}>
                        <Form.Item label="关联任务ID" name="linked_quest_id" extra="任务类菜单接取/提交时使用">
                          <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示不绑定" />
                        </Form.Item>
                      </Col>
                    ) : null}
                    <Col xs={12} md={6}>
                      <Form.Item label="状态" name="status"><Select options={editableStatusOptions} /></Form.Item>
                    </Col>
                    {!isBattleEntry ? (
                      <Col span={24}>
                        <Form.Item label="提示文案" name="action_notice"><Input.TextArea rows={3} /></Form.Item>
                      </Col>
                    ) : (
                      <Col span={24}>
                        <Form.Item label="提示文案（可选）" name="action_notice"><Input.TextArea rows={2} placeholder="挑战菜单通常无需提示文案" /></Form.Item>
                      </Col>
                    )}
                  </Row>

                  <Divider orientation="left">可见条件（可选）</Divider>
                  <Row gutter={16}>
                    <Col xs={24} md={6}>
                      <Form.Item label="任务ID" name={['conditions', 'quest_id']} tooltip="玩家拥有该任务时才显示此菜单">
                        <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示不限制" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item label="任务状态" name={['conditions', 'quest_state']}>
                        <Select options={questVisibilityStateOptions} placeholder="留空表示不限制" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item label="目标ID" name={['conditions', 'objective_id']} tooltip="分阶段任务填写 objective_id">
                        <InputNumber min={0} style={{ width: '100%' }} placeholder="0 表示不限制" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item
                        label="目标完成状态"
                        name={['conditions', 'objective_completed']}
                        getValueProps={(value: boolean | undefined) => ({
                          value: value === true ? 'true' : value === false ? 'false' : '',
                        })}
                        normalize={(value: string) => {
                          if (value === 'true') {
                            return true;
                          }
                          if (value === 'false') {
                            return false;
                          }
                          return undefined;
                        }}
                      >
                        <Select options={menuObjectiveCompletedOptions} placeholder="留空表示不限制" />
                      </Form.Item>
                    </Col>
                  </Row>
                </Form>
              ),
            },
            {
              key: 'dialogue',
              label: '剧情配置',
              disabled: !dialogueEntryID || !isDialogEntry,
              children: (
                <div style={{ maxHeight: '62vh', overflow: 'auto', paddingRight: 8 }}>
                  <NPCDialogueConfigDrawer
                    embedded
                    open={editorOpen && editorTab === 'dialogue'}
                    entityId={entityId}
                    entryId={dialogueEntryID}
                    entryTitle={dialogueEntryTitle}
                    npcName={entityName}
                    embeddedFormId={NPC_DIALOGUE_EMBEDDED_FORM_ID}
                    onEmbeddedEditingChange={setDialogueEditing}
                    onClose={handleCloseEditor}
                  />
                </div>
              ),
            },
          ]}
        />
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
    action_result_type: 'dialogue',
    action_notice: '这是一条后台新增提示。',
    battle_encounter_entity_id: entityId,
    linked_quest_id: 0,
    conditions: {},
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
    action_result_type: detail.action_result_type,
    action_notice: detail.action_notice,
    battle_encounter_entity_id: detail.battle_encounter_entity_id ?? detail.entity_id,
    linked_quest_id: detail.linked_quest_id ?? 0,
    conditions: detail.conditions ?? {},
    status: detail.status,
  };
}

function menuEntryUsesDialogue(values: MenuEntryFormValues): boolean {
  return values.entry_type === 'dialog' || values.action_result_type === 'dialogue' || values.action_result_type === 'dialog';
}

function normalizeMenuEntryPayload(values: MenuEntryFormValues): MenuEntryFormValues {
  return {
    ...values,
    linked_quest_id: values.linked_quest_id > 0 ? values.linked_quest_id : 0,
    conditions: {
      quest_id: values.conditions?.quest_id ?? 0,
      quest_state: values.conditions?.quest_state?.trim() ?? '',
      objective_id: values.conditions?.objective_id ?? 0,
      objective_completed: values.conditions?.objective_completed,
    },
  };
}

function mapMenuEntryFormToCreatePayload(values: MenuEntryFormValues): AdminCreateNPCMenuEntryPayload {
  const normalizedValues: MenuEntryFormValues = normalizeMenuEntryPayload(values);
  return {
    entity_id: normalizedValues.entity_id,
    entry_id: normalizedValues.entry_id ?? '',
    entry_type: normalizedValues.entry_type,
    title: normalizedValues.title,
    subtitle: normalizedValues.subtitle,
    state: normalizedValues.state,
    priority: 0,
    sort_order: 0,
    action_result_type: normalizedValues.action_result_type,
    action_notice: normalizedValues.action_notice,
    battle_encounter_entity_id: normalizedValues.battle_encounter_entity_id,
    linked_quest_id: normalizedValues.linked_quest_id,
    conditions: normalizedValues.conditions,
    status: normalizedValues.status,
  };
}

function mapMenuEntryFormToUpdatePayload(values: MenuEntryFormValues): AdminUpdateNPCMenuEntryPayload {
  const { entry_id: _entryID, ...rest } = mapMenuEntryFormToCreatePayload(values);
  return rest;
}

// 菜单顺序完全由前端列表顺序决定；创建后按插入位置重排全部菜单项。
function buildMenuOrderAfterCreate(rows: AdminNPCMenuEntrySummary[], createdEntryID: string, insertAfterEntryID: string | null): string[] {
  const currentOrder: string[] = rows.map((row: AdminNPCMenuEntrySummary) => row.entry_id);
  if (createdEntryID.trim() === '') {
    return currentOrder;
  }
  if (insertAfterEntryID) {
    const anchorIndex: number = currentOrder.findIndex((entryID: string) => entryID === insertAfterEntryID);
    if (anchorIndex >= 0) {
      currentOrder.splice(anchorIndex + 1, 0, createdEntryID);
      return currentOrder;
    }
  }
  currentOrder.push(createdEntryID);
  return currentOrder;
}

function describeMenuInsertPosition(rows: AdminNPCMenuEntrySummary[], editingEntryID: string | null, insertAfterEntryID: string | null): string {
  if (editingEntryID) {
    const currentIndex: number = rows.findIndex((row: AdminNPCMenuEntrySummary) => row.entry_id === editingEntryID);
    return currentIndex >= 0 ? `第 ${currentIndex + 1} 个` : '按当前列表顺序';
  }
  if (insertAfterEntryID) {
    const anchorIndex: number = rows.findIndex((row: AdminNPCMenuEntrySummary) => row.entry_id === insertAfterEntryID);
    if (anchorIndex >= 0) {
      return `插入到第 ${anchorIndex + 2} 个`;
    }
  }
  return `追加到第 ${rows.length + 1} 个`;
}

function mapMenuEntryDetailToOrderedPayload(detail: AdminNPCMenuEntryDetail, sortOrder: number): AdminUpdateNPCMenuEntryPayload {
  return {
    entity_id: detail.entity_id,
    entry_type: detail.entry_type,
    title: detail.title,
    subtitle: detail.subtitle,
    state: detail.state,
    priority: 0,
    sort_order: sortOrder,
    action_result_type: detail.action_result_type,
    action_notice: detail.action_notice,
    battle_encounter_entity_id: detail.battle_encounter_entity_id,
    linked_quest_id: detail.linked_quest_id,
    conditions: detail.conditions ?? {},
    status: detail.status,
  };
}
