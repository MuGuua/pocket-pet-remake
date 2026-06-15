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
  Table,
  Tag,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import {
  createAdminNPCEntity,
  deleteAdminNPCEntity,
  fetchAdminNPCEntities,
  fetchAdminNPCEntityDetail,
  updateAdminNPCEntity,
} from '../../services/npc';
import type {
  AdminCreateNPCEntityPayload,
  AdminNPCEntityDetail,
  AdminNPCEntityFilters,
  AdminNPCEntitySummary,
  AdminUpdateNPCEntityPayload,
} from '../../types/npc';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { NPCMenuEntryDrawer } from './NPCMenuEntryDrawer';

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

  useEffect(() => {
    filterForm.setFieldsValue({ status: '1' });
  }, [filterForm]);

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
    { title: '显示名', dataIndex: 'display_name', key: 'display_name', width: 160 },
    { title: '地图', dataIndex: 'scene_id', key: 'scene_id', width: 90 },
    { title: '坐标', key: 'pos', width: 120, render: (_value, record) => `${record.pos_x}, ${record.pos_y}` },
    { title: '方向/速度', key: 'move', width: 120, render: (_value, record) => `${record.dir}/${record.speed}` },
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
              setFilters(values);
            }}
          >
            <Form.Item name="entity_id" label="实体ID">
              <Input allowClear placeholder="实体ID" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="name" label="显示名">
              <Input allowClear placeholder="显示名" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="scene_id" label="地图ID">
              <Input allowClear placeholder="地图ID" style={{ width: 100 }} />
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
          scroll={{ x: 1400 }}
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
            <Descriptions.Item label="显示名">{detail.display_name}</Descriptions.Item>
            <Descriptions.Item label="地图ID">{detail.scene_id}</Descriptions.Item>
            <Descriptions.Item label="坐标">{`${detail.pos_x}, ${detail.pos_y}`}</Descriptions.Item>
            <Descriptions.Item label="方向 / 速度">{`${detail.dir} / ${detail.speed}`}</Descriptions.Item>
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
        width={720}
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
          <Row gutter={16}>
            {!editingRecord ? (
              <Col xs={24} md={8}>
                <Form.Item label="实体ID" name="entity_id" rules={[{ required: true, message: '请输入实体ID' }]}>
                  <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            ) : null}
            <Col xs={24} md={editingRecord ? 12 : 8}>
              <Form.Item label="实体编码" name="entity_code" rules={[{ required: true, message: '请输入实体编码' }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col xs={24} md={editingRecord ? 12 : 8}>
              <Form.Item label="显示名" name="display_name" rules={[{ required: true, message: '请输入显示名' }]}>
                <Input />
              </Form.Item>
            </Col>
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
    entity_id: 99001,
    entity_code: 'ops_npc',
    display_name: '后台测试 NPC',
    entity_type: 2,
    scene_id: 1,
    pos_x: 8,
    pos_y: 8,
    dir: 2,
    speed: 0,
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
    pos_x: detail.pos_x,
    pos_y: detail.pos_y,
    dir: detail.dir,
    speed: detail.speed,
    status: detail.status,
  };
}

function mapEntityFormToCreatePayload(values: EntityFormValues): AdminCreateNPCEntityPayload {
  return {
    entity_id: values.entity_id ?? 0,
    entity_code: values.entity_code,
    display_name: values.display_name,
    entity_type: values.entity_type,
    scene_id: values.scene_id,
    pos_x: values.pos_x,
    pos_y: values.pos_y,
    dir: values.dir,
    speed: values.speed,
    status: values.status,
  };
}

function mapEntityFormToUpdatePayload(values: EntityFormValues): AdminUpdateNPCEntityPayload {
  const { entity_id: _entityID, ...rest } = mapEntityFormToCreatePayload(values);
  return rest;
}
