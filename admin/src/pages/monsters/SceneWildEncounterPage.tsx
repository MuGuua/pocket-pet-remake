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
import { TableActionDropdown } from '../../components/TableActionDropdown';
import {
  createAdminSceneWildEncounter,
  deleteAdminSceneWildEncounter,
  fetchAdminSceneWildEncounterDetail,
  fetchAdminSceneWildEncounters,
  updateAdminSceneWildEncounter,
} from '../../services/sceneWildEncounter';
import type {
  AdminSceneWildEncounterDetail,
  AdminSceneWildEncounterListFilters,
  AdminSceneWildEncounterSummary,
  AdminUpsertSceneWildEncounterPayload,
} from '../../types/sceneWildEncounter';
import { SCENE_ID_OPTIONS, formatEncounterRatePercent } from '../../types/sceneWildEncounter';

interface SceneWildEncounterFormValues extends AdminUpsertSceneWildEncounterPayload {
  spawn_monster_ids_text?: string;
}

// 地图暗雷配置按 scene_id 绑定步进概率与刷怪池；进图后下发客户端本地判定，触发后上报服务端开战。
// 地图暗雷按 scene_id 配置步进概率与刷怪池；客户端本地 roll 后上报开战。
export function SceneWildEncounterPanel() {
  const [filterForm] = Form.useForm<AdminSceneWildEncounterListFilters>();
  const [editorForm] = Form.useForm<SceneWildEncounterFormValues>();
  const [filters, setFilters] = useState<AdminSceneWildEncounterListFilters>({});
  const [rows, setRows] = useState<AdminSceneWildEncounterSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminSceneWildEncounterDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminSceneWildEncounterDetail | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void loadEncounters(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadEncounters(nextFilters: AdminSceneWildEncounterListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminSceneWildEncounters({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载地图暗雷配置失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(sceneID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminSceneWildEncounterDetail(sceneID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载地图暗雷详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', sceneID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.setFieldsValue(defaultSceneWildEncounterValues());
      return;
    }
    if (!sceneID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminSceneWildEncounterDetail(sceneID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToFormValues(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载地图暗雷编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmit(values: SceneWildEncounterFormValues) {
    setSaving(true);
    try {
      const payload = buildPayloadFromForm(values);
      if (editingRecord) {
        await updateAdminSceneWildEncounter(editingRecord.scene_id, payload);
        message.success('地图暗雷配置更新成功');
      } else {
        await createAdminSceneWildEncounter(payload);
        message.success('地图暗雷配置创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      await loadEncounters(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存地图暗雷配置失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(sceneID: number) {
    try {
      await deleteAdminSceneWildEncounter(sceneID);
      message.success('地图暗雷配置已删除');
      if (detail?.scene_id === sceneID) {
        setDetail(null);
        setDetailOpen(false);
      }
      await loadEncounters(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除地图暗雷配置失败');
    }
  }

  const columns = useMemo<ColumnsType<AdminSceneWildEncounterSummary>>(
    () => [
      { title: '地图ID', dataIndex: 'scene_id', key: 'scene_id', width: 90, fixed: 'left' },
      { title: '配置名称', dataIndex: 'encounter_name', key: 'encounter_name', width: 180 },
      {
        title: '步进概率',
        dataIndex: 'encounter_rate',
        key: 'encounter_rate',
        width: 120,
        render: (value: number) => `${value}（${formatEncounterRatePercent(value)}）`,
      },
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
              { key: 'view', label: '详情', onClick: () => void handleViewDetail(record.scene_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.scene_id) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这条地图暗雷配置吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => void handleDelete(record.scene_id),
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
          <Form.Item name="scene_id" label="地图ID">
            <Input allowClear placeholder="scene_id" style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="name" label="名称">
            <Input allowClear placeholder="配置名称" style={{ width: 140 }} />
          </Form.Item>
          <Form.Item name="enabled" label="启用">
            <Select allowClear placeholder="状态" style={{ width: 90 }} options={[{ label: '启用', value: 'true' }, { label: '停用', value: 'false' }]} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
              <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
              <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增配置</Button>
            </Space>
          </Form.Item>
        </Form>

        <Table
          columns={columns}
          dataSource={rows}
          rowKey="scene_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有地图暗雷配置" /> }}
          scroll={{ x: 980 }}
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

      <Drawer title={detail ? `暗雷详情 · ${detail.encounter_name}` : '暗雷详情'} width={720} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? (
          <Typography.Text type="secondary">正在加载暗雷详情...</Typography.Text>
        ) : (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="地图ID">{detail.scene_id}</Descriptions.Item>
            <Descriptions.Item label="配置名称">{detail.encounter_name}</Descriptions.Item>
            <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="步进概率">
              {detail.encounter_rate}（{formatEncounterRatePercent(detail.encounter_rate)}，万分比）
            </Descriptions.Item>
            <Descriptions.Item label="描述">{detail.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="刷怪 monster_id 列表">{detail.spawn_monster_ids.join(', ') || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑暗雷 · scene ${editingRecord.scene_id}` : '新增地图暗雷配置'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={680}
        okText={editingRecord ? '保存修改' : '创建配置'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Form.Item label="地图 scene_id" name="scene_id" rules={[{ required: true, message: '请选择地图 scene_id' }]}>
            <Select disabled={Boolean(editingRecord)} options={SCENE_ID_OPTIONS} placeholder="选择地图" />
          </Form.Item>
          <Form.Item label="配置名称" name="encounter_name" rules={[{ required: true, message: '请输入配置名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item
            label="步进遭遇概率（万分比）"
            name="encounter_rate"
            extra="800 表示 8%；10000 表示 100%"
            rules={[{ required: true, message: '请输入遭遇概率' }]}
          >
            <InputNumber min={0} max={10000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="刷怪 monster_id 列表" name="spawn_monster_ids_text" extra="按槽位顺序填写，多个 ID 用英文逗号分隔，例如 9001,9002" rules={[{ required: true, message: '请填写刷怪列表' }]}>
            <Input placeholder="9001,9002" />
          </Form.Item>
          <Form.Item label="启用" name="is_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultSceneWildEncounterValues(): SceneWildEncounterFormValues {
  return {
    scene_id: 4,
    encounter_name: '新地图暗雷',
    description: '',
    encounter_rate: 800,
    spawn_monster_ids: [],
    spawn_monster_ids_text: '9001',
    is_enabled: true,
  };
}

function mapDetailToFormValues(detail: AdminSceneWildEncounterDetail): SceneWildEncounterFormValues {
  return {
    scene_id: detail.scene_id,
    encounter_name: detail.encounter_name,
    description: detail.description,
    encounter_rate: detail.encounter_rate,
    spawn_monster_ids: detail.spawn_monster_ids,
    spawn_monster_ids_text: detail.spawn_monster_ids.join(','),
    is_enabled: detail.is_enabled,
  };
}

function buildPayloadFromForm(values: SceneWildEncounterFormValues): AdminUpsertSceneWildEncounterPayload {
  const spawnMonsterIDs = parseMonsterIDs(values.spawn_monster_ids_text ?? '');
  return {
    scene_id: Number(values.scene_id),
    encounter_name: values.encounter_name.trim(),
    description: values.description?.trim() ?? '',
    encounter_rate: Number(values.encounter_rate),
    spawn_monster_ids: spawnMonsterIDs,
    is_enabled: Boolean(values.is_enabled),
  };
}

function parseMonsterIDs(raw: string): number[] {
  if (!raw.trim()) return [];
  return raw.split(',').map((item) => Number(item.trim())).filter((item) => Number.isFinite(item) && item > 0);
}
