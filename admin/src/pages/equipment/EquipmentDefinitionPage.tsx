import {
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
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
import { EquipmentEnhanceGoldCostEditor } from '../../components/EquipmentEnhanceGoldCostEditor';
import { RichTextDisplay } from '../../components/RichTextDisplay';
import { RichTextEditor } from '../../components/RichTextEditor';
import { ITEM_QUALITY_OPTIONS, formatItemQualityLabel } from '../../constants/itemQuality';
import {
  createAdminEquipmentDefinition,
  deleteAdminEquipmentDefinition,
  fetchAdminEquipmentDetail,
  fetchAdminEquipmentDefinitions,
  updateAdminEquipmentDefinition,
} from '../../services/equipmentDefinition';
import { fetchAdminItems } from '../../services/item';
import { fetchAdminSkillDefinitions } from '../../services/skillDefinition';
import type {
  AdminEquipmentDetail,
  AdminEquipmentListFilters,
  AdminEquipmentSummary,
  AdminEquipmentWeaponSkill,
  AdminUpsertEquipmentPayload,
} from '../../types/equipmentDefinition';
import { BIND_TYPE_LABELS, WEAPON_TYPE_LABELS, WEAPON_TYPE_OPTIONS, buildSelectOptions, formatDisplayLabel } from '../../utils/displayLabels';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import {
  ADMIN_EQUIPMENT_COMBAT_STAT_FIELDS,
  EQUIPMENT_SLOT_OPTIONS,
  defaultAdminEquipmentCombatStats,
  defaultEquipmentValues,
  defaultEquipmentEnhanceGoldCost,
  defaultMedicinePouchExtra,
} from '../../types/equipmentDefinition';

interface EquipmentFormValues extends AdminUpsertEquipmentPayload {
  allowed_gem_types_form: string[];
  property_entries: EquipmentPropertyEntry[];
  enhance_entries: EquipmentPropertyEntry[];
  weapon_skill_entries: WeaponSkillEntry[];
  weapon_skill_enhance_entries: WeaponSkillEnhanceEntry[];
}

interface WeaponSkillEntry {
  skill_id: number;
  base_level: number;
}

interface WeaponSkillEnhanceEntry {
  skill_id: number;
  level_per_enhance: number;
}

interface EquipmentDefinitionPageProps {
  embedded?: boolean;
}

interface EquipmentPropertyOption {
  key: string;
  label: string;
  group: 'base' | 'combat';
}

interface EquipmentPropertyEntry {
  key: string;
  value: number;
}

interface PropertyEditorFormValues {
  property_key: string;
  property_value: number;
}

const BASE_PROPERTY_OPTIONS: EquipmentPropertyOption[] = [
  { key: 'base_hp', label: '生命', group: 'base' },
  { key: 'base_mana', label: '法力', group: 'base' },
  { key: 'base_atk', label: '攻击', group: 'base' },
  { key: 'base_def', label: '防御', group: 'base' },
  { key: 'base_spd', label: '速度', group: 'base' },
];

const EQUIPMENT_PROPERTY_OPTIONS: EquipmentPropertyOption[] = [
  ...BASE_PROPERTY_OPTIONS,
  ...ADMIN_EQUIPMENT_COMBAT_STAT_FIELDS.map((field) => ({
    key: field.key,
    label: field.label,
    group: 'combat' as const,
  })),
];

const ENHANCE_PROPERTY_OPTIONS: EquipmentPropertyOption[] = [
  { key: 'hp_max', label: '生命', group: 'base' },
  { key: 'mana', label: '法力', group: 'base' },
  { key: 'atk', label: '攻击', group: 'base' },
  { key: 'def', label: '防御', group: 'base' },
  { key: 'spd', label: '速度', group: 'base' },
  ...ADMIN_EQUIPMENT_COMBAT_STAT_FIELDS.map((field) => ({
    key: field.key,
    label: field.label,
    group: 'combat' as const,
  })),
];

const ALLOWED_GEM_TYPE_OPTIONS = [
  { value: 'attack', label: '攻击' },
  { value: 'defense', label: '防御' },
  { value: 'hp', label: '生命' },
  { value: 'mana', label: '法力' },
  { value: 'speed', label: '速度' },
  { value: 'crit', label: '暴击' },
  { value: 'resist', label: '抗性' },
];

// 系统装备管理页：维护 item_definition + item_equipment_extra 人物装备模板。
export function EquipmentDefinitionPage({ embedded = false }: EquipmentDefinitionPageProps) {
  const [filterForm] = Form.useForm<AdminEquipmentListFilters>();
  const [editorForm] = Form.useForm<EquipmentFormValues>();
  const [propertyEditorForm] = Form.useForm<PropertyEditorFormValues>();
  const [filters, setFilters] = useState<AdminEquipmentListFilters>({});
  const [rows, setRows] = useState<AdminEquipmentSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminEquipmentDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminEquipmentDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [propertyEditorOpen, setPropertyEditorOpen] = useState(false);
  const [editingPropertyKey, setEditingPropertyKey] = useState<string | null>(null);
  const [enhanceEditorOpen, setEnhanceEditorOpen] = useState(false);
  const [editingEnhanceKey, setEditingEnhanceKey] = useState<string | null>(null);
  const [weaponSkillEditorOpen, setWeaponSkillEditorOpen] = useState(false);
  const [editingWeaponSkillID, setEditingWeaponSkillID] = useState<number | null>(null);
  const [weaponSkillEnhanceEditorOpen, setWeaponSkillEnhanceEditorOpen] = useState(false);
  const [editingWeaponSkillEnhanceID, setEditingWeaponSkillEnhanceID] = useState<number | null>(null);
  const [weaponSkillOptions, setWeaponSkillOptions] = useState<Array<{ value: number; label: string }>>([]);
  const [weaponSkillEditorForm] = Form.useForm<{ skill_id: number; base_level: number }>();
  const [weaponSkillEnhanceEditorForm] = Form.useForm<{ skill_id: number; level_per_enhance: number }>();
  const equipSlot = Form.useWatch('equip_slot', editorForm);
  const propertyEntries = Form.useWatch('property_entries', editorForm) ?? [];
  const enhanceEntries = Form.useWatch('enhance_entries', editorForm) ?? [];
  const weaponSkillEntries = Form.useWatch('weapon_skill_entries', editorForm) ?? [];
  const weaponSkillEnhanceEntries = Form.useWatch('weapon_skill_enhance_entries', editorForm) ?? [];
  const isWeaponSlot = equipSlot === 'weapon' || equipSlot === 'class_weapon';

  useEffect(() => {
    void loadRows(filters, page, pageSize);
  }, [filters, page, pageSize]);

  useEffect(() => {
    if (!editorOpen || !isWeaponSlot) {
      return;
    }
    void loadWeaponSkillOptions();
  }, [editorOpen, isWeaponSlot]);

  async function loadWeaponSkillOptions() {
    try {
      const result = await fetchAdminSkillDefinitions({ filters: { category: 'weapon', enabled: 'true' }, page: 1, pageSize: 100 });
      setWeaponSkillOptions(result.items.map((item) => ({ value: item.skill_id, label: `${item.skill_name} (#${item.skill_id})` })));
    } catch {
      setWeaponSkillOptions([]);
    }
  }

  async function loadRows(nextFilters: AdminEquipmentListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminEquipmentDefinitions({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载装备模板失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(itemID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminEquipmentDetail(itemID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载装备详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', itemID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      const nextItemID = await loadNextEquipmentItemID();
      editorForm.setFieldsValue(mapPayloadToForm(defaultEquipmentValues(nextItemID)));
      return;
    }
    if (!itemID) {
      return;
    }
    setDetailLoading(true);
    try {
      const result = await fetchAdminEquipmentDetail(itemID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载装备编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmit(values: EquipmentFormValues) {
    setSaving(true);
    // disabled/hidden 字段在 antd 提交值里可能缺失，这里强制取整份表单快照再映射，
    // 避免 item_id、item_code、rarity 等自动字段丢失后被序列化成 null。
    const fullValues = editorForm.getFieldsValue(true) as EquipmentFormValues;
    const payload = mapFormToPayload({ ...fullValues, ...values });
    try {
      if (editingRecord) {
        await updateAdminEquipmentDefinition(editingRecord.item_id, payload);
        message.success(`装备模板已更新：ID ${editingRecord.item_id} / ${payload.item_code}`);
      } else {
        const created = await createAdminEquipmentDefinition(payload);
        message.success(`装备模板已创建：ID ${created.item_id} / ${created.item_code}`);
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadRows(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存装备模板失败');
    } finally {
      setSaving(false);
    }
  }

  function handleOpenPropertyEditor(propertyKey?: string) {
    const currentEntries = editorForm.getFieldValue('property_entries') ?? [];
    const currentEntry = propertyKey ? currentEntries.find((entry: EquipmentPropertyEntry) => entry.key === propertyKey) : null;
    setEditingPropertyKey(propertyKey ?? null);
    propertyEditorForm.setFieldsValue({
      property_key: currentEntry?.key ?? '',
      property_value: currentEntry?.value ?? 0,
    });
    setPropertyEditorOpen(true);
  }

  function handleSubmitPropertyEditor(values: PropertyEditorFormValues) {
    const currentEntries: EquipmentPropertyEntry[] = editorForm.getFieldValue('property_entries') ?? [];
    const nextEntries = currentEntries.filter((entry) => entry.key !== editingPropertyKey && entry.key !== values.property_key);
    nextEntries.push({
      key: values.property_key,
      value: Number(values.property_value),
    });
    editorForm.setFieldValue('property_entries', sortPropertyEntries(nextEntries));
    setPropertyEditorOpen(false);
    setEditingPropertyKey(null);
    propertyEditorForm.resetFields();
  }

  function handleDeleteProperty(propertyKey: string) {
    const currentEntries: EquipmentPropertyEntry[] = editorForm.getFieldValue('property_entries') ?? [];
    editorForm.setFieldValue(
      'property_entries',
      currentEntries.filter((entry) => entry.key !== propertyKey),
    );
  }

  function handleOpenEnhanceEditor(propertyKey?: string) {
    const currentEntries = editorForm.getFieldValue('enhance_entries') ?? [];
    const currentEntry = propertyKey ? currentEntries.find((entry: EquipmentPropertyEntry) => entry.key === propertyKey) : null;
    setEditingEnhanceKey(propertyKey ?? null);
    propertyEditorForm.setFieldsValue({
      property_key: currentEntry?.key ?? '',
      property_value: currentEntry?.value ?? 0,
    });
    setEnhanceEditorOpen(true);
  }

  function handleSubmitEnhanceEditor(values: PropertyEditorFormValues) {
    const currentEntries: EquipmentPropertyEntry[] = editorForm.getFieldValue('enhance_entries') ?? [];
    const nextEntries = currentEntries.filter((entry) => entry.key !== editingEnhanceKey && entry.key !== values.property_key);
    nextEntries.push({
      key: values.property_key,
      value: Number(values.property_value),
    });
    editorForm.setFieldValue('enhance_entries', sortPropertyEntries(nextEntries, ENHANCE_PROPERTY_OPTIONS));
    setEnhanceEditorOpen(false);
    setEditingEnhanceKey(null);
    propertyEditorForm.resetFields();
  }

  function handleDeleteEnhance(propertyKey: string) {
    const currentEntries: EquipmentPropertyEntry[] = editorForm.getFieldValue('enhance_entries') ?? [];
    editorForm.setFieldValue(
      'enhance_entries',
      currentEntries.filter((entry) => entry.key !== propertyKey),
    );
  }

  function handleOpenWeaponSkillEditor(skillID?: number) {
    const currentEntries: WeaponSkillEntry[] = editorForm.getFieldValue('weapon_skill_entries') ?? [];
    const currentEntry = skillID ? currentEntries.find((entry) => entry.skill_id === skillID) : null;
    setEditingWeaponSkillID(skillID ?? null);
    weaponSkillEditorForm.setFieldsValue({
      skill_id: currentEntry?.skill_id ?? 0,
      base_level: currentEntry?.base_level ?? 0,
    });
    setWeaponSkillEditorOpen(true);
  }

  function handleSubmitWeaponSkillEditor(values: { skill_id: number; base_level: number }) {
    const skillID = Number(values.skill_id ?? 0);
    if (skillID <= 0) {
      message.error('请选择武器技能');
      return;
    }
    const currentEntries: WeaponSkillEntry[] = editorForm.getFieldValue('weapon_skill_entries') ?? [];
    const nextEntries = currentEntries.filter((entry) => entry.skill_id !== editingWeaponSkillID && entry.skill_id !== skillID);
    nextEntries.push({ skill_id: skillID, base_level: Number(values.base_level ?? 0) });
    editorForm.setFieldValue('weapon_skill_entries', nextEntries.sort((left, right) => left.skill_id - right.skill_id));
    setWeaponSkillEditorOpen(false);
    setEditingWeaponSkillID(null);
    weaponSkillEditorForm.resetFields();
  }

  function handleDeleteWeaponSkill(skillID: number) {
    const currentEntries: WeaponSkillEntry[] = editorForm.getFieldValue('weapon_skill_entries') ?? [];
    editorForm.setFieldValue('weapon_skill_entries', currentEntries.filter((entry) => entry.skill_id !== skillID));
    const enhanceEntriesValue: WeaponSkillEnhanceEntry[] = editorForm.getFieldValue('weapon_skill_enhance_entries') ?? [];
    editorForm.setFieldValue('weapon_skill_enhance_entries', enhanceEntriesValue.filter((entry) => entry.skill_id !== skillID));
  }

  function handleOpenWeaponSkillEnhanceEditor(skillID?: number) {
    const currentEntries: WeaponSkillEnhanceEntry[] = editorForm.getFieldValue('weapon_skill_enhance_entries') ?? [];
    const currentEntry = skillID ? currentEntries.find((entry) => entry.skill_id === skillID) : null;
    setEditingWeaponSkillEnhanceID(skillID ?? null);
    weaponSkillEnhanceEditorForm.setFieldsValue({
      skill_id: currentEntry?.skill_id ?? 0,
      level_per_enhance: currentEntry?.level_per_enhance ?? 1,
    });
    setWeaponSkillEnhanceEditorOpen(true);
  }

  function handleSubmitWeaponSkillEnhanceEditor(values: { skill_id: number; level_per_enhance: number }) {
    const skillID = Number(values.skill_id ?? 0);
    if (skillID <= 0) {
      message.error('请选择武器技能');
      return;
    }
    const configuredSkills: WeaponSkillEntry[] = editorForm.getFieldValue('weapon_skill_entries') ?? [];
    if (!configuredSkills.some((entry) => entry.skill_id === skillID)) {
      message.error('请先在武器附加技能中配置该技能');
      return;
    }
    const currentEntries: WeaponSkillEnhanceEntry[] = editorForm.getFieldValue('weapon_skill_enhance_entries') ?? [];
    const nextEntries = currentEntries.filter((entry) => entry.skill_id !== editingWeaponSkillEnhanceID && entry.skill_id !== skillID);
    nextEntries.push({ skill_id: skillID, level_per_enhance: Number(values.level_per_enhance ?? 0) });
    editorForm.setFieldValue('weapon_skill_enhance_entries', nextEntries.sort((left, right) => left.skill_id - right.skill_id));
    setWeaponSkillEnhanceEditorOpen(false);
    setEditingWeaponSkillEnhanceID(null);
    weaponSkillEnhanceEditorForm.resetFields();
  }

  function handleDeleteWeaponSkillEnhance(skillID: number) {
    const currentEntries: WeaponSkillEnhanceEntry[] = editorForm.getFieldValue('weapon_skill_enhance_entries') ?? [];
    editorForm.setFieldValue('weapon_skill_enhance_entries', currentEntries.filter((entry) => entry.skill_id !== skillID));
  }

  async function handleDelete(itemID: number) {
    try {
      await deleteAdminEquipmentDefinition(itemID);
      message.success('装备模板已停用');
      if (detail?.item_id === itemID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadRows(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '停用装备模板失败');
    }
  }

  // 新建装备模板时基于 item_definition 全表最大 item_id 分配下一个编号，
  // 避免只统计装备列表时漏掉普通物品/半成品模板占用的 ID 导致 409 冲突。
  async function loadNextEquipmentItemID(): Promise<number> {
    try {
      const result = await fetchAdminItems({ filters: {}, page: 1, pageSize: 1 });
      const currentMaxItemID = result.items[0]?.item_id ?? 4000;
      return currentMaxItemID + 1;
    } catch (error) {
      message.error(error instanceof Error ? error.message : '获取下一个装备物品ID失败');
      return 4001;
    }
  }

  const columns = useMemo<ColumnsType<AdminEquipmentSummary>>(
    () => [
      { title: '物品ID', dataIndex: 'item_id', width: 90 },
      { title: '编码', dataIndex: 'item_code', width: 140 },
      { title: '名称', dataIndex: 'item_name', width: 160 },
      { title: '部位', dataIndex: 'equip_slot_label', width: 100 },
      { title: '佩戴等级', dataIndex: 'required_level', width: 90 },
      {
        title: '品质',
        dataIndex: 'quality',
        width: 90,
        render: (value: number) => formatItemQualityLabel(value),
      },
      {
        title: '强化',
        width: 90,
        render: (_value, record) => (record.can_enhance ? `+${record.max_enhance_level}` : '不可'),
      },
      {
        title: '状态',
        dataIndex: 'is_enabled',
        width: 80,
        render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '停用'}</Tag>,
      },
      {
        title: '操作',
        key: 'actions',
        width: 100,
        render: (_value, record) => (
          <span onClick={(event) => event.stopPropagation()}>
            <TableActionDropdown
              actions={[
                { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.item_id) },
                { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.item_id) },
                {
                  key: 'delete',
                  label: '停用',
                  danger: true,
                  confirm: { title: '确认停用该装备模板吗？' },
                  onClick: () => void handleDelete(record.item_id),
                },
              ]}
            />
          </span>
        ),
      },
    ],
    [detail],
  );

  const isCostume = equipSlot === 'costume';
  const isMedicinePouch = equipSlot === 'medicine_pouch';
  const skipCombatStats = isCostume || isMedicinePouch;
  const selectedPropertyKeys = new Set(propertyEntries.map((entry) => entry.key));
  const propertyOptionMap = useMemo(
    () => new Map(EQUIPMENT_PROPERTY_OPTIONS.map((option) => [option.key, option])),
    [],
  );
  const enhanceOptionMap = useMemo(
    () => new Map(ENHANCE_PROPERTY_OPTIONS.map((option) => [option.key, option])),
    [],
  );
  const selectedEnhanceKeys = new Set(enhanceEntries.map((entry) => entry.key));
  const propertyColumns = useMemo<ColumnsType<EquipmentPropertyEntry>>(
    () => [
      {
        title: '属性',
        dataIndex: 'key',
        key: 'key',
        render: (value: string) => propertyOptionMap.get(value)?.label ?? value,
      },
      {
        title: '分类',
        dataIndex: 'key',
        key: 'group',
        width: 100,
        render: (value: string) => {
          const option = propertyOptionMap.get(value);
          return <Tag color={option?.group === 'base' ? 'blue' : 'gold'}>{option?.group === 'base' ? '基础' : '次要战斗'}</Tag>;
        },
      },
      {
        title: '数值',
        dataIndex: 'value',
        key: 'value',
        width: 120,
      },
      {
        title: '操作',
        key: 'actions',
        width: 140,
        render: (_value, record) => (
          <Space size={8}>
            <Button size="small" onClick={() => handleOpenPropertyEditor(record.key)}>编辑</Button>
            <Button size="small" danger onClick={() => handleDeleteProperty(record.key)}>删除</Button>
          </Space>
        ),
      },
    ],
    [propertyOptionMap],
  );
  const enhanceColumns = useMemo<ColumnsType<EquipmentPropertyEntry>>(
    () => [
      {
        title: '每级强化属性',
        dataIndex: 'key',
        key: 'key',
        render: (value: string) => enhanceOptionMap.get(value)?.label ?? value,
      },
      {
        title: '分类',
        dataIndex: 'key',
        key: 'group',
        width: 100,
        render: (value: string) => {
          const option = enhanceOptionMap.get(value);
          return <Tag color={option?.group === 'base' ? 'cyan' : 'purple'}>{option?.group === 'base' ? '基础' : '次要战斗'}</Tag>;
        },
      },
      {
        title: '每级增加',
        dataIndex: 'value',
        key: 'value',
        width: 120,
      },
      {
        title: '操作',
        key: 'actions',
        width: 140,
        render: (_value, record) => (
          <Space size={8}>
            <Button size="small" onClick={() => handleOpenEnhanceEditor(record.key)}>编辑</Button>
            <Button size="small" danger onClick={() => handleDeleteEnhance(record.key)}>删除</Button>
          </Space>
        ),
      },
    ],
    [enhanceOptionMap],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {!embedded ? (
        <Card>
          <Typography.Title level={4} style={{ marginTop: 0 }}>系统装备管理</Typography.Title>
          <Typography.Text type="secondary">
            维护人物可穿戴装备模板；创建时同时写入 item_definition 与 item_equipment_extra。
          </Typography.Text>
        </Card>
      ) : null}

      <Card>
        <Form
          form={filterForm}
          layout="inline"
          onFinish={(values) => {
            setFilters(values);
            setPage(1);
          }}
        >
          <Form.Item label="物品ID" name="item_id"><Input allowClear style={{ width: 120 }} /></Form.Item>
          <Form.Item label="关键字" name="keyword"><Input placeholder="编码/名称" allowClear style={{ width: 160 }} /></Form.Item>
          <Form.Item label="部位" name="equip_slot">
            <Select allowClear style={{ width: 140 }} options={EQUIPMENT_SLOT_OPTIONS.map((item) => ({ value: item.value, label: item.label }))} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">查询</Button>
              <Button onClick={() => { filterForm.resetFields(); setFilters({}); setPage(1); }}>重置</Button>
              <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增装备</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card>
        <Table
          rowKey="item_id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1100 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
          onRow={(record) => ({
            onClick: () => void handleViewDetail(record.item_id),
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      <Drawer
        title={detail ? `装备详情 · ${detail.item_name}` : '装备详情'}
        width={640}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
      >
        {detailLoading ? null : detail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="物品ID">{detail.item_id}</Descriptions.Item>
              <Descriptions.Item label="编码">{detail.item_code}</Descriptions.Item>
              <Descriptions.Item label="名称">{detail.item_name}</Descriptions.Item>
              <Descriptions.Item label="部位">{detail.equip_slot_label}</Descriptions.Item>
              {(detail.equip_slot === 'weapon' || detail.equip_slot === 'class_weapon') && detail.weapon_type ? (
                <Descriptions.Item label="武器类型">{formatDisplayLabel(WEAPON_TYPE_LABELS, detail.weapon_type)}</Descriptions.Item>
              ) : null}
              <Descriptions.Item label="佩戴等级">{detail.required_level}</Descriptions.Item>
              <Descriptions.Item label="品质">{formatItemQualityLabel(detail.quality)}</Descriptions.Item>
              <Descriptions.Item label="绑定类型">{formatDisplayLabel(BIND_TYPE_LABELS, detail.bind_type)}</Descriptions.Item>
              <Descriptions.Item label="可丢弃">{detail.can_drop ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="强化">{detail.can_enhance ? `最高 +${detail.max_enhance_level}` : '不可强化'}</Descriptions.Item>
              <Descriptions.Item label="套装ID">{detail.set_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="五维" span={2}>
                {`生命${detail.base_hp} / 法力${detail.base_mana} / 攻${detail.base_atk} / 防${detail.base_def} / 速${detail.base_spd}`}
              </Descriptions.Item>
              <Descriptions.Item label="介绍" span={2}>
                <RichTextDisplay value={detail.desc} />
              </Descriptions.Item>
            </Descriptions>
            {!detail.appearance_only ? (
              <Descriptions bordered column={2} size="small" title="次要战斗属性">
                {ADMIN_EQUIPMENT_COMBAT_STAT_FIELDS.map((field) => (
                  <Descriptions.Item key={field.key} label={field.label}>{detail.combat_stats[field.key]}</Descriptions.Item>
                ))}
              </Descriptions>
            ) : null}
            {detail.medicine_pouch ? (
              <Descriptions bordered column={2} size="small" title="药囊战后恢复">
                <Descriptions.Item label="回满玩家生命">{detail.medicine_pouch.restore_player_hp ? '是' : '否'}</Descriptions.Item>
                <Descriptions.Item label="回满玩家精力">{detail.medicine_pouch.restore_player_spirit ? '是' : '否'}</Descriptions.Item>
                <Descriptions.Item label="回满玩家体力">{detail.medicine_pouch.restore_player_vigor ? '是' : '否'}</Descriptions.Item>
                <Descriptions.Item label="回满宠物生命">{detail.medicine_pouch.restore_pet_hp ? '是' : '否'}</Descriptions.Item>
                <Descriptions.Item label="回满宠物精力">{detail.medicine_pouch.restore_pet_spirit ? '是' : '否'}</Descriptions.Item>
              </Descriptions>
            ) : null}
          </Space>
        ) : null}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑装备 · ${editingRecord.item_name}` : '新增装备模板'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={920}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={editingRecord ? '保存修改' : '创建装备'}
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item
                label="物品ID"
                name="item_id"
                rules={[{ required: true, message: '系统未生成物品ID，请关闭弹窗后重试' }]}
                extra="新建时自动取当前最大物品ID + 1，创建后不可修改。"
              >
                <InputNumber min={1} style={{ width: '100%' }} disabled controls={false} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item
                label="编码"
                name="item_code"
                rules={[{ required: true, message: '系统未生成编码，请关闭弹窗后重试' }]}
                extra="系统按 equipment_{item_id} 自动生成并锁定。"
              >
                <Input disabled />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="名称" name="item_name" rules={[{ required: true, message: '请输入名称' }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="部位" name="equip_slot" rules={[{ required: true, message: '请选择部位' }]}>
                <Select options={EQUIPMENT_SLOT_OPTIONS.map((item) => ({ value: item.value, label: item.label }))} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="佩戴等级" name="required_level"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            </Col>
            <Col xs={12} md={8}>
              <Form.Item label="品质" name="quality">
                <Select options={ITEM_QUALITY_OPTIONS} />
              </Form.Item>
            </Col>
            <Form.Item name="rarity" hidden>
              <InputNumber min={1} />
            </Form.Item>
            <Col span={24}>
              <Form.Item
                label="介绍"
                name="desc"
                extra="支持 BBCode 富文本与 {item:物品ID} 占位符，客户端会在占位符处内联展示物品 icon 与名称。"
              >
                <RichTextEditor rows={4} placeholder="例如：攻击 [color=green]+50[/color]" />
              </Form.Item>
            </Col>
            <Form.Item name="icon" hidden>
              <Input />
            </Form.Item>
            <Col xs={24} md={8}>
              <Form.Item label="绑定类型" name="bind_type">
                <Select options={buildSelectOptions(BIND_TYPE_LABELS)} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}><Form.Item label="套装ID" name="set_id"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="职业限制" name="career_limit"><Input placeholder="留空表示不限" /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可出售" name="can_sell" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可丢弃" name="can_drop" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可存仓" name="can_store" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="启用" name="is_enabled" valuePropName="checked"><Switch /></Form.Item></Col>
          </Row>

          {isCostume ? (
            <>
              <Divider plain>时装外观</Divider>
              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item label="外观资源ID" name="appearance_skin_id">
                    <Input placeholder="例如 泳装_001" />
                  </Form.Item>
                </Col>
              </Row>
            </>
          ) : null}

          {isMedicinePouch ? (
            <>
              <Divider plain>药囊战后恢复</Divider>
              <Row gutter={16}>
                <Col xs={12} md={8}><Form.Item label="回满玩家生命" name={['medicine_pouch', 'restore_player_hp']} valuePropName="checked"><Switch /></Form.Item></Col>
                <Col xs={12} md={8}><Form.Item label="回满玩家精力" name={['medicine_pouch', 'restore_player_spirit']} valuePropName="checked"><Switch /></Form.Item></Col>
                <Col xs={12} md={8}><Form.Item label="回满玩家体力" name={['medicine_pouch', 'restore_player_vigor']} valuePropName="checked"><Switch /></Form.Item></Col>
                <Col xs={12} md={8}><Form.Item label="回满宠物生命" name={['medicine_pouch', 'restore_pet_hp']} valuePropName="checked"><Switch /></Form.Item></Col>
                <Col xs={12} md={8}><Form.Item label="回满宠物精力" name={['medicine_pouch', 'restore_pet_spirit']} valuePropName="checked"><Switch /></Form.Item></Col>
              </Row>
            </>
          ) : null}

          {!skipCombatStats ? (
            <>
              <Divider plain>装备属性</Divider>
              <Form.Item name="property_entries" hidden>
                <Input />
              </Form.Item>
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Button type="dashed" onClick={() => handleOpenPropertyEditor()}>
                  添加属性
                </Button>
                <Table<EquipmentPropertyEntry>
                  rowKey="key"
                  size="small"
                  columns={propertyColumns}
                  dataSource={propertyEntries}
                  pagination={false}
                  locale={{ emptyText: '当前还没有配置属性，点击“添加属性”开始设置' }}
                />
              </Space>
              <Divider plain>强化与镶嵌</Divider>
              <Row gutter={16}>
                <Col xs={12} md={6}><Form.Item label="可强化" name="can_enhance" valuePropName="checked"><Switch /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="最高强化" name="max_enhance_level"><InputNumber min={0} max={15} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="镶嵌孔数" name="socket_count"><InputNumber min={0} max={8} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={12}>
                  <Form.Item label="允许宝石类型" name="allowed_gem_types_form" extra="支持多选；保存后会写回 allowed_gem_types 列表。">
                    <Select
                      mode="multiple"
                      allowClear
                      options={buildAllowedGemTypeOptions(editorForm.getFieldValue('allowed_gem_types_form') ?? [])}
                      placeholder="选择允许镶嵌的宝石类型"
                    />
                  </Form.Item>
                </Col>
              </Row>
              <Divider plain>强化铜币消耗</Divider>
              <EquipmentEnhanceGoldCostEditor />
              <Divider plain>强化成长属性</Divider>
              <Form.Item name="enhance_entries" hidden>
                <Input />
              </Form.Item>
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Typography.Text type="secondary">
                  强化属性按“每升 1 级增加多少”配置；只添加这件装备真正需要增长的属性。
                </Typography.Text>
                <Button type="dashed" onClick={() => handleOpenEnhanceEditor()} disabled={!Boolean(editorForm.getFieldValue('can_enhance'))}>
                  添加强化属性
                </Button>
                <Table<EquipmentPropertyEntry>
                  rowKey="key"
                  size="small"
                  columns={enhanceColumns}
                  dataSource={enhanceEntries}
                  pagination={false}
                  locale={{ emptyText: '当前没有强化成长属性，可按需添加' }}
                />
              </Space>
              {isWeaponSlot ? (
                <>
                  <Divider plain>武器类型</Divider>
                  <Row gutter={16}>
                    <Col xs={24} md={8}>
                      <Form.Item
                        label="武器类型"
                        name="weapon_type"
                        extra="已学会的武器技能需在战斗中装备同类型武器才能使用。"
                        rules={weaponSkillEntries.length > 0 ? [{ required: true, message: '配置了武器技能时必须选择武器类型' }] : []}
                      >
                        <Select allowClear options={WEAPON_TYPE_OPTIONS} placeholder="剑 / 枪 / 法杖" />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Divider plain>武器附加技能</Divider>
                  <Form.Item name="weapon_skill_entries" hidden>
                    <Input />
                  </Form.Item>
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Typography.Text type="secondary">
                      仅可引用系统技能分类为「武器技能」的模板；基础等级为 +0 强化时的技能等级。
                    </Typography.Text>
                    <Button type="dashed" onClick={() => handleOpenWeaponSkillEditor()}>
                      添加武器技能
                    </Button>
                    <Table<WeaponSkillEntry>
                      rowKey="skill_id"
                      size="small"
                      columns={[
                        {
                          title: '技能',
                          dataIndex: 'skill_id',
                          render: (value: number) => weaponSkillOptions.find((item) => item.value === value)?.label ?? `#${value}`,
                        },
                        { title: '基础等级', dataIndex: 'base_level', width: 100 },
                        {
                          title: '操作',
                          key: 'actions',
                          width: 140,
                          render: (_value, record) => (
                            <Space size={8}>
                              <Button size="small" onClick={() => handleOpenWeaponSkillEditor(record.skill_id)}>编辑</Button>
                              <Button size="small" danger onClick={() => handleDeleteWeaponSkill(record.skill_id)}>删除</Button>
                            </Space>
                          ),
                        },
                      ]}
                      dataSource={weaponSkillEntries}
                      pagination={false}
                      locale={{ emptyText: '尚未配置武器附加技能' }}
                    />
                  </Space>
                  <Divider plain>武器技能强化成长</Divider>
                  <Form.Item name="weapon_skill_enhance_entries" hidden>
                    <Input />
                  </Form.Item>
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Typography.Text type="secondary">
                      配置每强化 1 级时，对应武器技能等级增加多少；有效等级 = 基础等级 + 强化等级 × 每级成长。
                    </Typography.Text>
                    <Button
                      type="dashed"
                      onClick={() => handleOpenWeaponSkillEnhanceEditor()}
                      disabled={!Boolean(editorForm.getFieldValue('can_enhance')) || weaponSkillEntries.length === 0}
                    >
                      添加强化技能等级成长
                    </Button>
                    <Table<WeaponSkillEnhanceEntry>
                      rowKey="skill_id"
                      size="small"
                      columns={[
                        {
                          title: '武器技能',
                          dataIndex: 'skill_id',
                          render: (value: number) => weaponSkillOptions.find((item) => item.value === value)?.label ?? `#${value}`,
                        },
                        { title: '每级增加等级', dataIndex: 'level_per_enhance', width: 120 },
                        {
                          title: '操作',
                          key: 'actions',
                          width: 140,
                          render: (_value, record) => (
                            <Space size={8}>
                              <Button size="small" onClick={() => handleOpenWeaponSkillEnhanceEditor(record.skill_id)}>编辑</Button>
                              <Button size="small" danger onClick={() => handleDeleteWeaponSkillEnhance(record.skill_id)}>删除</Button>
                            </Space>
                          ),
                        },
                      ]}
                      dataSource={weaponSkillEnhanceEntries}
                      pagination={false}
                      locale={{ emptyText: '尚未配置武器技能强化成长' }}
                    />
                  </Space>
                </>
              ) : null}
            </>
          ) : null}
        </Form>
      </Modal>

      <Modal
        title={editingPropertyKey ? '编辑属性' : '添加属性'}
        open={propertyEditorOpen}
        onCancel={() => {
          setPropertyEditorOpen(false);
          setEditingPropertyKey(null);
          propertyEditorForm.resetFields();
        }}
        onOk={() => propertyEditorForm.submit()}
        destroyOnClose
        okText="确定"
        cancelText="取消"
      >
        <Form form={propertyEditorForm} layout="vertical" onFinish={handleSubmitPropertyEditor}>
          <Form.Item
            label="属性"
            name="property_key"
            rules={[{ required: true, message: '请选择属性' }]}
          >
            <Select
              options={EQUIPMENT_PROPERTY_OPTIONS.map((option) => ({
                value: option.key,
                label: `${option.label} · ${option.group === 'base' ? '基础' : '次要战斗'}`,
                disabled: option.key !== editingPropertyKey && selectedPropertyKeys.has(option.key),
              }))}
              placeholder="请选择属性"
            />
          </Form.Item>
          <Form.Item
            label="数值"
            name="property_value"
            rules={[{ required: true, message: '请输入数值' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingWeaponSkillID ? '编辑武器技能' : '添加武器技能'}
        open={weaponSkillEditorOpen}
        onCancel={() => {
          setWeaponSkillEditorOpen(false);
          setEditingWeaponSkillID(null);
          weaponSkillEditorForm.resetFields();
        }}
        onOk={() => weaponSkillEditorForm.submit()}
        destroyOnClose
        okText="确定"
        cancelText="取消"
      >
        <Form form={weaponSkillEditorForm} layout="vertical" onFinish={handleSubmitWeaponSkillEditor}>
          <Form.Item label="武器技能" name="skill_id" rules={[{ required: true, message: '请选择武器技能' }]}>
            <Select
              showSearch
              optionFilterProp="label"
              options={weaponSkillOptions.map((option) => ({
                ...option,
                disabled: option.value !== editingWeaponSkillID && weaponSkillEntries.some((entry: WeaponSkillEntry) => entry.skill_id === option.value),
              }))}
              placeholder="选择系统武器技能"
            />
          </Form.Item>
          <Form.Item label="基础等级" name="base_level" rules={[{ required: true, message: '请输入基础等级' }]}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingWeaponSkillEnhanceID ? '编辑武器技能强化成长' : '添加武器技能强化成长'}
        open={weaponSkillEnhanceEditorOpen}
        onCancel={() => {
          setWeaponSkillEnhanceEditorOpen(false);
          setEditingWeaponSkillEnhanceID(null);
          weaponSkillEnhanceEditorForm.resetFields();
        }}
        onOk={() => weaponSkillEnhanceEditorForm.submit()}
        destroyOnClose
        okText="确定"
        cancelText="取消"
      >
        <Form form={weaponSkillEnhanceEditorForm} layout="vertical" onFinish={handleSubmitWeaponSkillEnhanceEditor}>
          <Form.Item label="武器技能" name="skill_id" rules={[{ required: true, message: '请选择武器技能' }]}>
            <Select
              options={weaponSkillEntries.map((entry: WeaponSkillEntry) => ({
                value: entry.skill_id,
                label: weaponSkillOptions.find((item) => item.value === entry.skill_id)?.label ?? `#${entry.skill_id}`,
                disabled: entry.skill_id !== editingWeaponSkillEnhanceID && weaponSkillEnhanceEntries.some((item: WeaponSkillEnhanceEntry) => item.skill_id === entry.skill_id),
              }))}
              placeholder="选择已配置的武器技能"
            />
          </Form.Item>
          <Form.Item label="每强化 1 级增加等级" name="level_per_enhance" rules={[{ required: true, message: '请输入每级成长' }]}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingEnhanceKey ? '编辑强化属性' : '添加强化属性'}
        open={enhanceEditorOpen}
        onCancel={() => {
          setEnhanceEditorOpen(false);
          setEditingEnhanceKey(null);
          propertyEditorForm.resetFields();
        }}
        onOk={() => propertyEditorForm.submit()}
        destroyOnClose
        okText="确定"
        cancelText="取消"
      >
        <Form form={propertyEditorForm} layout="vertical" onFinish={handleSubmitEnhanceEditor}>
          <Form.Item
            label="强化属性"
            name="property_key"
            rules={[{ required: true, message: '请选择强化属性' }]}
          >
            <Select
              options={ENHANCE_PROPERTY_OPTIONS.map((option) => ({
                value: option.key,
                label: `${option.label} · ${option.group === 'base' ? '基础' : '次要战斗'}`,
                disabled: option.key !== editingEnhanceKey && selectedEnhanceKeys.has(option.key),
              }))}
              placeholder="请选择强化后每级增加的属性"
            />
          </Form.Item>
          <Form.Item
            label="每级增加值"
            name="property_value"
            rules={[{ required: true, message: '请输入每级增加值' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}

function mapDetailToForm(detail: AdminEquipmentDetail): EquipmentFormValues {
  return {
    ...detail,
    combat_stats: detail.combat_stats ?? defaultAdminEquipmentCombatStats(),
    allowed_gem_types_form: detail.allowed_gem_types ?? [],
    medicine_pouch: detail.medicine_pouch ?? defaultMedicinePouchExtra(),
    property_entries: buildPropertyEntries(detail),
    enhance_entries: buildEnhanceEntries(detail.enhance_per_level_stats ?? {}),
    weapon_skill_entries: buildWeaponSkillEntries(detail.weapon_skills ?? []),
    weapon_skill_enhance_entries: buildWeaponSkillEnhanceEntries(detail.enhance_per_level_weapon_skill_levels ?? {}),
    enhance_gold_cost: detail.enhance_gold_cost ?? defaultEquipmentEnhanceGoldCost(),
  };
}

function mapPayloadToForm(payload: AdminUpsertEquipmentPayload): EquipmentFormValues {
  return {
    ...payload,
    allowed_gem_types_form: payload.allowed_gem_types ?? [],
    medicine_pouch: payload.medicine_pouch ?? defaultMedicinePouchExtra(),
    property_entries: buildPropertyEntries(payload),
    enhance_entries: buildEnhanceEntries(payload.enhance_per_level_stats ?? {}),
    weapon_skill_entries: buildWeaponSkillEntries(payload.weapon_skills ?? []),
    weapon_skill_enhance_entries: buildWeaponSkillEnhanceEntries(payload.enhance_per_level_weapon_skill_levels ?? {}),
    enhance_gold_cost: payload.enhance_gold_cost ?? defaultEquipmentEnhanceGoldCost(),
  };
}

function mapFormToPayload(values: EquipmentFormValues): AdminUpsertEquipmentPayload {
  const enhanceStats: Record<string, number> = buildEnhanceStatsMap(values.enhance_entries ?? []);
  const allowedGemTypes = (values.allowed_gem_types_form ?? []).map((item) => item.trim()).filter(Boolean);
  const propertyValues = splitPropertyEntries(values.property_entries ?? []);
  return {
    item_id: Number(values.item_id ?? 0),
    item_code: values.item_code ?? '',
    item_name: values.item_name ?? '',
    desc: values.desc ?? '',
    icon: values.icon ?? '',
    quality: Number(values.quality ?? 1),
    rarity: Number(values.rarity ?? 1),
    required_level: Number(values.required_level ?? 0),
    bind_type: values.bind_type ?? 'none',
    can_sell: Boolean(values.can_sell),
    can_drop: Boolean(values.can_drop),
    can_store: Boolean(values.can_store),
    is_enabled: Boolean(values.is_enabled),
    equip_slot: values.equip_slot ?? '',
    career_limit: values.career_limit ?? '',
    can_enhance: Boolean(values.can_enhance),
    max_enhance_level: Number(values.max_enhance_level ?? 0),
    set_id: Number(values.set_id ?? 0),
    appearance_skin_id: values.appearance_skin_id ?? '',
    appearance_only: values.equip_slot === 'costume',
    base_hp: propertyValues.base_hp,
    base_mana: propertyValues.base_mana,
    base_atk: propertyValues.base_atk,
    base_def: propertyValues.base_def,
    base_spd: propertyValues.base_spd,
    combat_stats: propertyValues.combat_stats,
    enhance_per_level_stats: enhanceStats,
    enhance_per_level_weapon_skill_levels: buildWeaponSkillEnhanceMap(values.weapon_skill_enhance_entries ?? []),
    weapon_skills: buildWeaponSkillPayload(values.weapon_skill_entries ?? []),
    weapon_type: values.weapon_type ?? '',
    enhance_gold_cost: values.can_enhance
      ? {
          is_enabled: Boolean(values.enhance_gold_cost?.is_enabled),
          base_copper: Number(values.enhance_gold_cost?.base_copper ?? 0),
          increment_mode: values.enhance_gold_cost?.increment_mode === 'percent' ? 'percent' : 'fixed',
          increment_fixed: Number(values.enhance_gold_cost?.increment_fixed ?? 0),
          increment_percent: Number(values.enhance_gold_cost?.increment_percent ?? 0),
        }
      : defaultEquipmentEnhanceGoldCost(),
    socket_count: Number(values.socket_count ?? 0),
    allowed_gem_types: allowedGemTypes,
    medicine_pouch: values.equip_slot === 'medicine_pouch' ? (values.medicine_pouch ?? defaultMedicinePouchExtra()) : undefined,
  };
}

function buildAllowedGemTypeOptions(currentValues: string[]): Array<{ value: string; label: string }> {
  const knownMap = new Map(ALLOWED_GEM_TYPE_OPTIONS.map((item) => [item.value, item]));
  const currentOnlyOptions = currentValues
    .filter((item) => item && !knownMap.has(item))
    .map((item) => ({ value: item, label: `${item} (历史值)` }));
  return [...ALLOWED_GEM_TYPE_OPTIONS, ...currentOnlyOptions];
}

function buildPropertyEntries(source: Pick<
  AdminUpsertEquipmentPayload,
  'base_hp' | 'base_mana' | 'base_atk' | 'base_def' | 'base_spd' | 'combat_stats'
>): EquipmentPropertyEntry[] {
  const nextEntries: EquipmentPropertyEntry[] = [];
  BASE_PROPERTY_OPTIONS.forEach((option) => {
    const value = Number(source[option.key as keyof typeof source] ?? 0);
    if (value > 0) {
      nextEntries.push({ key: option.key, value });
    }
  });
  const combatStats = source.combat_stats ?? defaultAdminEquipmentCombatStats();
  ADMIN_EQUIPMENT_COMBAT_STAT_FIELDS.forEach((field) => {
    const value = Number(combatStats[field.key] ?? 0);
    if (value > 0) {
      nextEntries.push({ key: field.key, value });
    }
  });
  return sortPropertyEntries(nextEntries);
}

function splitPropertyEntries(entries: EquipmentPropertyEntry[]): {
  base_hp: number;
  base_mana: number;
  base_atk: number;
  base_def: number;
  base_spd: number;
  combat_stats: ReturnType<typeof defaultAdminEquipmentCombatStats>;
} {
  const result = {
    base_hp: 0,
    base_mana: 0,
    base_atk: 0,
    base_def: 0,
    base_spd: 0,
    combat_stats: defaultAdminEquipmentCombatStats(),
  };
  entries.forEach((entry) => {
    const value = Number(entry.value ?? 0);
    switch (entry.key) {
      case 'base_hp':
        result.base_hp = value;
        break;
      case 'base_mana':
        result.base_mana = value;
        break;
      case 'base_atk':
        result.base_atk = value;
        break;
      case 'base_def':
        result.base_def = value;
        break;
      case 'base_spd':
        result.base_spd = value;
        break;
      default:
        if (entry.key in result.combat_stats) {
          result.combat_stats[entry.key as keyof ReturnType<typeof defaultAdminEquipmentCombatStats>] = value;
        }
        break;
    }
  });
  return result;
}

function buildEnhanceEntries(enhanceStats: Record<string, number>): EquipmentPropertyEntry[] {
  const entries: EquipmentPropertyEntry[] = [];
  Object.entries(enhanceStats).forEach(([key, value]) => {
    if (Number(value) > 0) {
      entries.push({ key, value: Number(value) });
    }
  });
  return sortPropertyEntries(entries, ENHANCE_PROPERTY_OPTIONS);
}

function buildEnhanceStatsMap(entries: EquipmentPropertyEntry[]): Record<string, number> {
  const result: Record<string, number> = {};
  entries.forEach((entry) => {
    const value = Number(entry.value ?? 0);
    if (value > 0) {
      result[entry.key] = value;
    }
  });
  return result;
}

function buildWeaponSkillEntries(items: AdminEquipmentWeaponSkill[]): WeaponSkillEntry[] {
  return items
    .filter((item) => item.skill_id > 0)
    .map((item) => ({ skill_id: item.skill_id, base_level: Number(item.base_level ?? 0) }))
    .sort((left, right) => left.skill_id - right.skill_id);
}

function buildWeaponSkillPayload(entries: WeaponSkillEntry[]): AdminEquipmentWeaponSkill[] {
  return entries
    .filter((entry) => entry.skill_id > 0)
    .map((entry) => ({ skill_id: entry.skill_id, base_level: Number(entry.base_level ?? 0) }));
}

function buildWeaponSkillEnhanceEntries(levels: Record<string, number>): WeaponSkillEnhanceEntry[] {
  return Object.entries(levels)
    .map(([skillID, value]) => ({ skill_id: Number(skillID), level_per_enhance: Number(value) }))
    .filter((entry) => entry.skill_id > 0 && entry.level_per_enhance > 0)
    .sort((left, right) => left.skill_id - right.skill_id);
}

function buildWeaponSkillEnhanceMap(entries: WeaponSkillEnhanceEntry[]): Record<string, number> {
  const result: Record<string, number> = {};
  entries.forEach((entry) => {
    if (entry.skill_id > 0 && entry.level_per_enhance > 0) {
      result[String(entry.skill_id)] = entry.level_per_enhance;
    }
  });
  return result;
}

function sortPropertyEntries(entries: EquipmentPropertyEntry[], orderSource: EquipmentPropertyOption[] = EQUIPMENT_PROPERTY_OPTIONS): EquipmentPropertyEntry[] {
  const orderMap = new Map(orderSource.map((option, index) => [option.key, index]));
  return [...entries].sort((left, right) => (orderMap.get(left.key) ?? 999) - (orderMap.get(right.key) ?? 999));
}
