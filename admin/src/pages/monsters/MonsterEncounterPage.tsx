import {
  Button,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
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
import { MonsterSpawnListEditor } from '../../components/MonsterSpawnListEditor';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import {
  createAdminMonsterEncounter,
  deleteAdminMonsterEncounter,
  fetchAdminMonsterEncounterDetail,
  fetchAdminMonsterEncounters,
  updateAdminMonsterEncounter,
} from '../../services/monsterEncounter';
import type {
  AdminMonsterEncounterDetail,
  AdminMonsterEncounterListFilters,
  AdminMonsterEncounterSummary,
  AdminUpsertMonsterEncounterPayload,
} from '../../types/monsterEncounter';

type MonsterEncounterFormValues = AdminUpsertMonsterEncounterPayload;

// NPC 固定战遭遇按世界 entity_id 绑定刷怪列表；玩家与 NPC 交互后开战。
export function NpcFixedEncounterPanel() {
  const [filterForm] = Form.useForm<AdminMonsterEncounterListFilters>();
  const [editorForm] = Form.useForm<MonsterEncounterFormValues>();
  const [filters, setFilters] = useState<AdminMonsterEncounterListFilters>({});
  const [rows, setRows] = useState<AdminMonsterEncounterSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminMonsterEncounterDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminMonsterEncounterDetail | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void loadEncounters(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadEncounters(nextFilters: AdminMonsterEncounterListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminMonsterEncounters({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载怪物遭遇配置失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(entityID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminMonsterEncounterDetail(entityID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载怪物遭遇详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', entityID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.setFieldsValue(defaultMonsterEncounterValues());
      return;
    }
    if (!entityID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminMonsterEncounterDetail(entityID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToFormValues(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载怪物遭遇编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmit(values: MonsterEncounterFormValues) {
    setSaving(true);
    try {
      const payload = buildPayloadFromForm(values);
      if (editingRecord) {
        await updateAdminMonsterEncounter(editingRecord.entity_id, payload);
        message.success('怪物遭遇配置更新成功');
      } else {
        await createAdminMonsterEncounter(payload);
        message.success('怪物遭遇配置创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      await loadEncounters(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存怪物遭遇配置失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(entityID: number) {
    try {
      await deleteAdminMonsterEncounter(entityID);
      message.success('怪物遭遇配置已删除');
      if (detail?.entity_id === entityID) {
        setDetail(null);
        setDetailOpen(false);
      }
      await loadEncounters(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除怪物遭遇配置失败');
    }
  }

  const columns = useMemo<ColumnsType<AdminMonsterEncounterSummary>>(
    () => [
      { title: '实体ID', dataIndex: 'entity_id', key: 'entity_id', width: 120, fixed: 'left' },
      { title: '遭遇名称', dataIndex: 'encounter_name', key: 'encounter_name', width: 180 },
      { title: '刷怪数量', dataIndex: 'spawn_count', key: 'spawn_count', width: 100 },
      {
        title: '启用',
        dataIndex: 'is_enabled',
        key: 'is_enabled',
        width: 90,
        render: (value: boolean) => (value ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
      },
      {
        title: '操作',
        key: 'actions',
        width: 100,
        fixed: 'right',
        render: (_value, record) => (
          <TableActionDropdown
            actions={[
              { key: 'view', label: '详情', onClick: () => void handleViewDetail(record.entity_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.entity_id) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这条怪物遭遇配置吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => void handleDelete(record.entity_id),
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
      <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }} style={{ marginBottom: 16 }}>
          <Form.Item name="entity_id" label="实体ID">
            <Input allowClear placeholder="实体ID" style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="name" label="名称">
            <Input allowClear placeholder="遭遇名称" style={{ width: 140 }} />
          </Form.Item>
          <Form.Item name="enabled" label="启用">
            <Select allowClear placeholder="状态" style={{ width: 90 }} options={[{ label: '启用', value: 'true' }, { label: '停用', value: 'false' }]} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
              <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
              <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增遭遇</Button>
            </Space>
          </Form.Item>
        </Form>
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="entity_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有怪物遭遇配置" /> }}
          scroll={{ x: 900 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条配置`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      <Drawer title={detail ? `遭遇详情 · ${detail.encounter_name}` : '遭遇详情'} width={720} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? (
          <Typography.Text type="secondary">正在加载遭遇详情...</Typography.Text>
        ) : (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="实体ID">{detail.entity_id}</Descriptions.Item>
            <Descriptions.Item label="遭遇名称">{detail.encounter_name}</Descriptions.Item>
            <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="描述">{detail.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="刷怪 monster_id 列表">{detail.spawn_monster_ids.join(', ') || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑遭遇 · ${editingRecord.encounter_name}` : '新增怪物遭遇配置'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={640}
        okText={editingRecord ? '保存修改' : '创建配置'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Form.Item label="世界实体ID" name="entity_id" rules={[{ required: true, message: '请输入实体 ID' }]}>
            <InputNumber min={1} disabled={Boolean(editingRecord)} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="遭遇名称" name="encounter_name" rules={[{ required: true, message: '请输入遭遇名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item
            label="刷怪 monster_id 列表"
            name="spawn_monster_ids"
            extra="按列表顺序配置刷怪槽位；可重复添加同一怪物，表示多个相同出场位。"
            rules={[{ required: true, message: '请至少添加一个怪物' }]}
          >
            <MonsterSpawnListEditor />
          </Form.Item>
          <Form.Item label="启用" name="is_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultMonsterEncounterValues(): MonsterEncounterFormValues {
  return {
    entity_id: 88001,
    encounter_name: '新遭遇配置',
    description: '',
    spawn_monster_ids: [9001],
    is_enabled: true,
  };
}

function mapDetailToFormValues(detail: AdminMonsterEncounterDetail): MonsterEncounterFormValues {
  return {
    entity_id: detail.entity_id,
    encounter_name: detail.encounter_name,
    description: detail.description,
    spawn_monster_ids: detail.spawn_monster_ids ?? [],
    is_enabled: detail.is_enabled,
  };
}

function buildPayloadFromForm(values: MonsterEncounterFormValues): AdminUpsertMonsterEncounterPayload {
  return {
    entity_id: Number(values.entity_id),
    encounter_name: values.encounter_name.trim(),
    description: values.description?.trim() ?? '',
    spawn_monster_ids: values.spawn_monster_ids ?? [],
    is_enabled: Boolean(values.is_enabled),
  };
}
