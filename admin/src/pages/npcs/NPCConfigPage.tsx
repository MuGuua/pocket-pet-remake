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
  Modal,
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
import { RichTextEditor } from '../../components/RichTextEditor';
import { RichTextDisplay } from '../../components/RichTextDisplay';
import {
  createAdminNPCEntity,
  deleteAdminNPCEntity,
  fetchAdminNPCEntities,
  fetchAdminNPCEntityDetail,
  fetchAdminWorldScenes,
  updateAdminNPCEntity,
} from '../../services/npc';
import type {
  AdminCreateNPCEntityPayload,
  AdminNPCEntityDetail,
  AdminNPCEntityFilters,
  AdminNPCEntitySummary,
  AdminUpdateNPCEntityPayload,
  AdminWorldSceneSummary,
} from '../../types/npc';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { NPCMenuEntryDrawer } from './NPCMenuEntryDrawer';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

interface EntityFormValues {
  entity_id?: number;
  entity_code?: string;
  display_name: string;
  entity_type: number;
  scene_id?: number;
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

// 地图 NPC 配置页：统一管理地图实体，并在每个 NPC 编辑入口中维护其交互菜单。
export function NPCConfigPage() {
  return <MapNPCPanel />;
}

function MapNPCPanel() {
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
  const [menuDrawerOpen, setMenuDrawerOpen] = useState(false);
  const [menuDrawerEntity, setMenuDrawerEntity] = useState<{ entityId: number; entityName: string } | null>(null);
  const [sceneOptions, setSceneOptions] = useState<Array<{ label: string; value: number }>>([]);

  useEffect(() => {
    filterForm.setFieldsValue({ status: '1' });
    void loadSceneOptions();
  }, [filterForm]);

  async function loadSceneOptions() {
    try {
      const result = await fetchAdminWorldScenes();
      setSceneOptions(result.items.map((scene: AdminWorldSceneSummary) => ({
        label: `${scene.scene_name}（Scene ${scene.scene_id}）`,
        value: scene.scene_id,
      })));
    } catch {
      setSceneOptions([]);
    }
  }

  useEffect(() => {
    void loadRows(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadRows(nextFilters: AdminNPCEntityFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminNPCEntities({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载地图 NPC 失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  function handleOpenMenuDrawer(entityID: number, entityName: string) {
    setMenuDrawerEntity({ entityId: entityID, entityName });
    setMenuDrawerOpen(true);
  }

  async function handleViewDetail(entityID: number) {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      setDetail(await fetchAdminNPCEntityDetail(entityID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载地图 NPC 详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', entityID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultEntityValues());
      return;
    }
    if (!entityID) {
      return;
    }
    setDetailLoading(true);
    try {
      const result = await fetchAdminNPCEntityDetail(entityID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapEntityDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载地图 NPC 编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: EntityFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminNPCEntity(editingRecord.entity_id, mapEntityFormToUpdatePayload(values));
        message.success('地图 NPC 更新成功');
      } else {
        const created = await createAdminNPCEntity(mapEntityFormToCreatePayload(values));
        message.success('地图 NPC 创建成功');
        setEditingRecord(created);
        editorForm.setFieldsValue(mapEntityDetailToForm(created));
      }
      await loadRows(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存地图 NPC 失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(entityID: number) {
    setDeletingID(entityID);
    try {
      await deleteAdminNPCEntity(entityID);
      message.success('地图 NPC 已删除');
      if (detail?.entity_id === entityID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadRows(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除地图 NPC 失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminNPCEntitySummary>>(() => [
    { title: '实体ID', dataIndex: 'entity_id', key: 'entity_id', width: 110, fixed: 'left' },
    { title: '实体编码', dataIndex: 'entity_code', key: 'entity_code', width: 160 },
    { title: '显示名', dataIndex: 'display_name', key: 'display_name', width: 160, render: (value: string) => <RichTextDisplay value={value} /> },
    {
      title: '所属场景',
      dataIndex: 'scene_name',
      key: 'scene_name',
      width: 160,
      render: (value: string, record: AdminNPCEntitySummary) => value || `场景 ${record.scene_id}`,
    },
    {
      title: '状态',
      dataIndex: 'status_text',
      key: 'status_text',
      width: 100,
      render: (value: string) => <Tag color={value === '启用' ? 'green' : 'default'}>{value}</Tag>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 100,
      fixed: 'right',
      render: (_value, record) => (
        <TableActionDropdown
          loading={deletingID === record.entity_id}
          actions={[
            { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.entity_id) },
            { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.entity_id) },
            { key: 'menu', label: '菜单配置', onClick: () => handleOpenMenuDrawer(record.entity_id, record.display_name) },
            {
              key: 'delete',
              label: '删除',
              danger: true,
              confirm: {
                title: '确认删除这个地图 NPC 吗？',
                description: '删除后，该 NPC 下的所有菜单配置也会一并删除。',
              },
              onClick: () => void handleDelete(record.entity_id),
            },
          ]}
        />
      ),
    },
  ], [deletingID]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title="地图 NPC 列表"
        extra={(
          <Form
            form={filterForm}
            layout="inline"
            onFinish={(values) => {
              setPage(1);
              setFilters({
                ...values,
                scene_id: values.scene_id !== undefined && values.scene_id !== null && String(values.scene_id) !== ''
                  ? String(values.scene_id)
                  : undefined,
              });
            }}
          >
            <Form.Item name="entity_id" label="实体ID">
              <Input allowClear placeholder="实体ID" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="name" label="显示名">
              <Input allowClear placeholder="显示名" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="scene_id" label="所属场景">
              <Select
                allowClear
                placeholder="全部场景"
                style={{ width: 160 }}
                options={sceneOptions}
              />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select options={statusOptions} style={{ width: 100 }} />
            </Form.Item>
            <Form.Item>
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
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增 NPC</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="entity_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有地图 NPC 数据" /> }}
          scroll={{ x: 1100 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 个地图 NPC`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </Card>

      <Drawer
        title={detail ? `地图 NPC 详情 · ${detail.entity_id}` : '地图 NPC 详情'}
        width={520}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
        extra={detail ? (
          <TableActionDropdown
            buttonType="default"
            actions={[
              { key: 'menu', label: '菜单配置', onClick: () => handleOpenMenuDrawer(detail.entity_id, detail.display_name) },
            ]}
          />
        ) : null}
      >
        {detailLoading ? (
          <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载 NPC 详情..." />
          </div>
        ) : detail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="实体ID">{detail.entity_id}</Descriptions.Item>
            <Descriptions.Item label="编码">{detail.entity_code}</Descriptions.Item>
            <Descriptions.Item label="显示名"><RichTextDisplay value={detail.display_name} /></Descriptions.Item>
            <Descriptions.Item label="所属场景">{detail.scene_name || `场景 ${detail.scene_id}`}</Descriptions.Item>
            <Descriptions.Item label="状态">{detail.status_text}</Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑地图 NPC · ${editingRecord.entity_id}` : '新增地图 NPC'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingRecord(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={880}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={editingRecord ? '保存修改' : '创建 NPC'}
        cancelText="取消"
        footer={(_, { OkBtn, CancelBtn }) => (
          <Space>
            <CancelBtn />
            {editingRecord ? (
              <Button onClick={() => handleOpenMenuDrawer(editingRecord.entity_id, editingRecord.display_name)}>
                菜单配置
              </Button>
            ) : null}
            <OkBtn />
          </Space>
        )}
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Divider plain orientation="left">身份信息</Divider>
          <Row gutter={16}>
            {editingRecord ? (
              <>
                <Col xs={24} md={12}>
                  <Form.Item label="实体 ID" name="entity_id">
                    <Input disabled />
                  </Form.Item>
                </Col>
                <Col xs={24} md={12}>
                  <Form.Item label="实体编码" name="entity_code">
                    <Input disabled />
                  </Form.Item>
                </Col>
              </>
            ) : null}
            <Col span={24}>
              <Form.Item
                label="NPC 显示名"
                name="display_name"
                rules={[{ required: true, message: '请输入 NPC 显示名' }]}
              >
                <RichTextEditor rows={1} />
              </Form.Item>
            </Col>

            <Col xs={24} md={12}>
              <Form.Item label="实体类型" name="entity_type">
                <Select disabled options={[{ label: '地图 NPC（类型 2）', value: 2 }]} />
              </Form.Item>
            </Col>

            <Col span={24}>
              <Divider plain orientation="left">归属与发布</Divider>
            </Col>
            <Col xs={24} md={16}>
              <Form.Item
                label="所属场景"
                name="scene_id"
                rules={[{ required: true, message: '请选择所属场景' }]}
              >
                <Select
                  showSearch
                  allowClear
                  optionFilterProp="label"
                  options={sceneOptions}
                />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="发布状态" name="status">
                <Select options={editableStatusOptions} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      <NPCMenuEntryDrawer
        open={menuDrawerOpen}
        entityId={menuDrawerEntity?.entityId ?? null}
        entityName={menuDrawerEntity?.entityName ?? ''}
        onClose={() => {
          setMenuDrawerOpen(false);
          setMenuDrawerEntity(null);
        }}
      />
    </Space>
  );
}

function defaultEntityValues(): EntityFormValues {
  return {
    entity_id: undefined,
    entity_code: undefined,
    display_name: '',
    entity_type: 2,
    scene_id: undefined,
    status: 1,
  };
}

function mapEntityDetailToForm(detail: AdminNPCEntityDetail): EntityFormValues {
  return {
    entity_id: detail.entity_id,
    entity_code: detail.entity_code,
    display_name: detail.display_name,
    entity_type: detail.entity_type,
    scene_id: detail.scene_id,
    status: detail.status,
  };
}

function mapEntityFormToCreatePayload(values: EntityFormValues): AdminCreateNPCEntityPayload {
  return {
    display_name: values.display_name,
    entity_type: values.entity_type,
    scene_id: values.scene_id ?? 0,
    status: values.status,
  };
}

function mapEntityFormToUpdatePayload(values: EntityFormValues): AdminUpdateNPCEntityPayload {
  return mapEntityFormToCreatePayload(values);
}
