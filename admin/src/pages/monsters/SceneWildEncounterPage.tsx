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
  createDefaultMonsterBattleRewardEntry,
  mapMonsterBattleRewardPayload,
  MonsterBattleRewardEditor,
} from '../../components/MonsterBattleRewardEditor';
import {
  createAdminSceneWildEncounter,
  deleteAdminSceneWildEncounter,
  fetchAdminSceneWildEncounterDetail,
  fetchAdminSceneWildEncounters,
  updateAdminSceneWildEncounter,
} from '../../services/sceneWildEncounter';
import { fetchAdminMonsterDefinitions } from '../../services/monsterDefinition';
import type {
  AdminSceneWildEncounterDetail,
  AdminSceneWildEncounterListFilters,
  AdminSceneWildEncounterSummary,
  AdminSceneWildEncounterFormation,
  AdminSceneWildEncounterMonsterSlot,
  AdminUpsertSceneWildEncounterPayload,
} from '../../types/sceneWildEncounter';
import { SCENE_ID_OPTIONS, formatEncounterRatePercent } from '../../types/sceneWildEncounter';
import type { AdminMonsterDefinitionSummary } from '../../types/monsterDefinition';
import type { AdminMonsterBattleRewardEntry } from '../../types/monsterDefinition';

interface SceneWildEncounterFormValues extends Omit<AdminUpsertSceneWildEncounterPayload, 'rewards'> {
  rewards: AdminMonsterBattleRewardEntry[];
}

type FormationModalMode = 'view' | 'create' | 'edit';

interface FormationEditorValues extends AdminSceneWildEncounterFormation {}

// 地图暗雷配置按 scene_id 绑定步进概率与刷怪池；进图后下发客户端本地判定，触发后上报服务端开战。
// 地图暗雷按 scene_id 配置步进概率与刷怪池；客户端本地 roll 后上报开战。
export function SceneWildEncounterPanel() {
  const [filterForm] = Form.useForm<AdminSceneWildEncounterListFilters>();
  const [editorForm] = Form.useForm<SceneWildEncounterFormValues>();
  const [formationForm] = Form.useForm<FormationEditorValues>();
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
  const [monsterOptions, setMonsterOptions] = useState<Array<{ label: string; value: number }>>([]);
  const [formationModalOpen, setFormationModalOpen] = useState(false);
  const [formationModalMode, setFormationModalMode] = useState<FormationModalMode>('create');
  const [editingFormationIndex, setEditingFormationIndex] = useState<number | null>(null);
  const [formationRows, setFormationRows] = useState<AdminSceneWildEncounterFormation[]>([]);

  useEffect(() => {
    void loadEncounters(filters, page, pageSize);
  }, [filters, page, pageSize]);

  useEffect(() => {
    void loadMonsterOptions();
  }, []);

  async function loadMonsterOptions() {
    try {
      const result = await fetchAdminMonsterDefinitions({ filters: { enabled: 'true' }, page: 1, pageSize: 100 });
      setMonsterOptions(result.items.map((item: AdminMonsterDefinitionSummary) => ({
        label: `${item.monster_id} · ${item.monster_name}（Lv.${item.level}）`,
        value: item.monster_id,
      })));
    } catch {
      setMonsterOptions([]);
    }
  }

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
      const defaults = defaultSceneWildEncounterValues();
      setEditingRecord(null);
      setFormationRows(defaults.formations);
      editorForm.setFieldsValue(defaults);
      return;
    }
    if (!sceneID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminSceneWildEncounterDetail(sceneID);
      const formValues = mapDetailToFormValues(result);
      setEditingRecord(result);
      setFormationRows(formValues.formations);
      editorForm.setFieldsValue(formValues);
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
      if (!normalizeFormations(formationRows).length) {
        message.error('请至少添加一个怪物编队');
        return;
      }
      const payload = buildPayloadFromForm({ ...values, formations: formationRows });
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

  function handleOpenFormationModal(mode: FormationModalMode, index?: number) {
    const formations = getCurrentFormations(formationRows);
    setFormationModalMode(mode);
    setEditingFormationIndex(typeof index === 'number' ? index : null);
    if (mode === 'create') {
      formationForm.setFieldsValue(defaultFormationValues(formations.length));
    } else if (typeof index === 'number' && formations[index]) {
      formationForm.setFieldsValue(cloneFormation(formations[index]));
    }
    setFormationModalOpen(true);
  }

  async function handleSaveFormation() {
    try {
      const values = await formationForm.validateFields();
      const normalized = normalizeFormations([values])[0];
      if (!normalized) {
        message.error('请至少选择一个怪物槽位');
        return;
      }
      const formations = getCurrentFormations(formationRows);
      if (formationModalMode === 'edit' && editingFormationIndex !== null) {
        formations[editingFormationIndex] = normalized;
      } else {
        formations.push(normalized);
      }
      setFormationRows(formations);
      editorForm.setFieldValue('formations', formations);
      setFormationModalOpen(false);
      setEditingFormationIndex(null);
    } catch {
      // antd Form 会展示具体字段错误，这里不额外弹出全局提示。
    }
  }

  function handleDeleteFormation(index: number) {
    const formations = getCurrentFormations(formationRows);
    if (formations.length <= 1) {
      message.warning('至少保留一个怪物编队');
      return;
    }
    formations.splice(index, 1);
    setFormationRows(formations);
    editorForm.setFieldValue('formations', formations);
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
      { title: '编队数', dataIndex: 'formation_count', key: 'formation_count', width: 90 },
      { title: '怪物种类数', dataIndex: 'spawn_count', key: 'spawn_count', width: 110 },
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


  const formationColumns = useMemo<ColumnsType<AdminSceneWildEncounterFormation>>(
    () => [
      { title: '编队名称', dataIndex: 'formation_name', key: 'formation_name', width: 180 },
      { title: '权重', dataIndex: 'weight', key: 'weight', width: 100 },
      {
        title: '怪物槽位',
        dataIndex: 'spawn_monster_ids',
        key: 'spawn_monster_ids',
        render: (value: number[]) => formatMonsterIDs(value),
      },
      {
        title: '操作',
        key: 'actions',
        width: 110,
        render: (_value, _record, index) => (
          <TableActionDropdown
            actions={[
              { key: 'view', label: '查看', onClick: () => handleOpenFormationModal('view', index) },
              { key: 'edit', label: '修改', onClick: () => handleOpenFormationModal('edit', index) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这个怪物编队吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => handleDeleteFormation(index),
              },
            ]}
          />
        ),
      },
    ],
    [formationRows],
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
            <Descriptions.Item label="怪物编队">{formatFormations(detail.formations)}</Descriptions.Item>
            <Descriptions.Item label="遭遇战奖励">{formatRewardCount(detail.rewards)}</Descriptions.Item>
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
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            <Space style={{ width: '100%', justifyContent: 'space-between' }}>
              <Typography.Text strong>怪物编队</Typography.Text>
              <Button type="primary" onClick={() => handleOpenFormationModal('create')}>添加编队</Button>
            </Space>
            <Table
              size="small"
              rowKey={(_record, index) => String(index ?? 0)}
              columns={formationColumns}
              dataSource={formationRows}
              pagination={false}
              locale={{ emptyText: <Empty description="请点击右上角添加怪物编队" /> }}
            />
          </Space>
          <Form.Item
            label="遭遇战固定奖励"
            name="rewards"
            extra="本处奖励会与实际随机到的编队中每个怪物自身奖励相加，作为最终战斗奖励。"
          >
            <MonsterBattleRewardEditor />
          </Form.Item>
          <Form.Item label="启用" name="is_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={formationModalTitle(formationModalMode)}
        open={formationModalOpen}
        onCancel={() => setFormationModalOpen(false)}
        onOk={formationModalMode === 'view' ? () => setFormationModalOpen(false) : () => void handleSaveFormation()}
        okText={formationModalMode === 'view' ? '关闭' : '保存编队'}
        cancelText="取消"
        width={560}
        destroyOnClose
      >
        <Form form={formationForm} layout="vertical" disabled={formationModalMode === 'view'}>
          <Form.Item label="编队名称" name="formation_name" rules={[{ required: true, message: '请输入编队名称' }]}>
            <Input placeholder="例如：螳螂双怪" />
          </Form.Item>
          <Form.Item label="触发权重" name="weight" extra="同一地图内按权重随机；例如 80 和 20 表示约 80%/20%。" rules={[{ required: true, message: '请输入权重' }]}>
            <InputNumber min={1} max={10000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.List name="monster_slots">
            {(slotFields, slotOps) => (
              <Space direction="vertical" style={{ width: '100%' }}>
                <Typography.Text type="secondary">怪物槽位（可重复添加同一种怪物；关闭奖励后该槽位只参战，不结算怪物自身奖励）</Typography.Text>
                {slotFields.map((slotField, slotIndex) => (
                  <Space key={slotField.key} align="baseline" style={{ width: '100%' }}>
                    <Form.Item name={[slotField.name, 'monster_id']} rules={[{ required: true, message: '请选择怪物' }]} style={{ flex: 1, marginBottom: 8 }}>
                      <Select showSearch options={monsterOptions} placeholder={`第 ${slotIndex + 1} 个怪物`} optionFilterProp="label" />
                    </Form.Item>
                    <Form.Item name={[slotField.name, 'reward_enabled']} valuePropName="checked" style={{ marginBottom: 8 }}>
                      <Switch checkedChildren="发奖励" unCheckedChildren="不发" />
                    </Form.Item>
                    {formationModalMode !== 'view' ? (
                      <Button onClick={() => slotOps.add(cloneFormationSlot(getFormationSlotValue(formationForm.getFieldValue('monster_slots'), slotIndex)), slotIndex + 1)}>
                        复制
                      </Button>
                    ) : null}
                    {formationModalMode !== 'view' && slotFields.length > 1 ? <Button danger onClick={() => slotOps.remove(slotField.name)}>删除</Button> : null}
                  </Space>
                ))}
                {formationModalMode !== 'view' ? <Button onClick={() => slotOps.add({ monster_id: undefined, reward_enabled: true })}>添加怪物槽位</Button> : null}
              </Space>
            )}
          </Form.List>
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
    spawn_monster_ids: [9001],
    formations: [defaultFormationValues(0)],
    rewards: [createDefaultMonsterBattleRewardEntry()],
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
    formations: detail.formations?.length ? detail.formations : [defaultFormationValues(0)],
    rewards: detail.rewards?.length ? detail.rewards : [],
    is_enabled: detail.is_enabled,
  };
}

function buildPayloadFromForm(values: SceneWildEncounterFormValues): AdminUpsertSceneWildEncounterPayload {
  const formations = normalizeFormations(values.formations);
  const spawnMonsterIDs = uniqueMonsterIDs(formations.flatMap((formation) => formation.spawn_monster_ids));
  return {
    scene_id: Number(values.scene_id),
    encounter_name: values.encounter_name.trim(),
    description: values.description?.trim() ?? '',
    encounter_rate: Number(values.encounter_rate),
    spawn_monster_ids: spawnMonsterIDs,
    formations,
    rewards: (values.rewards ?? []).map((item, index) => mapMonsterBattleRewardPayload(item, index)),
    is_enabled: Boolean(values.is_enabled),
  };
}

function defaultFormationValues(index: number): AdminSceneWildEncounterFormation {
  return {
    formation_name: index === 0 ? '默认编队' : `编队${index + 1}`,
    weight: 10000,
    spawn_monster_ids: [9001],
    monster_slots: [{ monster_id: 9001, reward_enabled: true }],
  };
}

function getCurrentFormations(value: unknown): AdminSceneWildEncounterFormation[] {
  return Array.isArray(value) ? (value as AdminSceneWildEncounterFormation[]).map(cloneFormation) : [];
}

function cloneFormation(formation: AdminSceneWildEncounterFormation): AdminSceneWildEncounterFormation {
  const monsterSlots = normalizeFormationSlots(formation.monster_slots, formation.spawn_monster_ids);
  return {
    formation_name: formation.formation_name,
    weight: formation.weight,
    spawn_monster_ids: monsterSlots.map((slot) => slot.monster_id),
    monster_slots: monsterSlots,
  };
}

function normalizeFormations(formations?: AdminSceneWildEncounterFormation[]): AdminSceneWildEncounterFormation[] {
  return (formations ?? []).map((formation, index) => {
    const monsterSlots = normalizeFormationSlots(formation.monster_slots, formation.spawn_monster_ids);
    return {
      formation_name: formation.formation_name?.trim() || `编队${index + 1}`,
      weight: Number(formation.weight) || 1,
      spawn_monster_ids: monsterSlots.map((slot) => slot.monster_id),
      monster_slots: monsterSlots,
    };
  }).filter((formation) => formation.spawn_monster_ids.length > 0);
}

function normalizeFormationSlots(
  monsterSlots?: AdminSceneWildEncounterMonsterSlot[],
  fallbackMonsterIDs?: number[],
): AdminSceneWildEncounterMonsterSlot[] {
  const sourceSlots = monsterSlots?.length
    ? monsterSlots
    : (fallbackMonsterIDs ?? []).map((monsterID) => ({ monster_id: monsterID, reward_enabled: true }));
  return sourceSlots
    .map((slot) => ({
      monster_id: Number(slot.monster_id),
      reward_enabled: Boolean(slot.reward_enabled ?? true),
    }))
    .filter((slot) => Number.isFinite(slot.monster_id) && slot.monster_id > 0);
}

function getFormationSlotValue(value: unknown, index: number): AdminSceneWildEncounterMonsterSlot {
  if (!Array.isArray(value)) {
    return { monster_id: 0, reward_enabled: true };
  }
  return cloneFormationSlot(value[index] as AdminSceneWildEncounterMonsterSlot | undefined);
}

function cloneFormationSlot(slot?: AdminSceneWildEncounterMonsterSlot): AdminSceneWildEncounterMonsterSlot {
  return {
    monster_id: Number(slot?.monster_id ?? 0),
    reward_enabled: Boolean(slot?.reward_enabled ?? true),
  };
}

function uniqueMonsterIDs(monsterIDs: number[]): number[] {
  return Array.from(new Set(monsterIDs.filter((item) => Number.isFinite(item) && item > 0)));
}

function formatFormations(formations?: AdminSceneWildEncounterFormation[]): string {
  if (!formations?.length) return '-';
  return formations.map((formation) => {
    const monsterSlots = normalizeFormationSlots(formation.monster_slots, formation.spawn_monster_ids);
    const slotText = monsterSlots.map((slot) => `${slot.monster_id}${slot.reward_enabled ? '' : '（不发奖励）'}`).join(', ');
    return `${formation.formation_name} [权重 ${formation.weight}]: ${slotText || '-'}`;
  }).join('；');
}

function formatMonsterIDs(monsterIDs?: number[]): string {
  return monsterIDs?.length ? monsterIDs.join(', ') : '-';
}

function formatRewardCount(rewards?: unknown[]): string {
  return rewards?.length ? `${rewards.length} 条` : '未配置';
}

function formationModalTitle(mode: FormationModalMode): string {
  if (mode === 'view') return '查看怪物编队';
  if (mode === 'edit') return '修改怪物编队';
  return '添加怪物编队';
}
