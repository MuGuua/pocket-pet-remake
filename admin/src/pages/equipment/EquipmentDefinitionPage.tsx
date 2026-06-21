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
import {
  createAdminEquipmentDefinition,
  deleteAdminEquipmentDefinition,
  fetchAdminEquipmentDetail,
  fetchAdminEquipmentDefinitions,
  updateAdminEquipmentDefinition,
} from '../../services/equipmentDefinition';
import {
  ADMIN_PET_COMBAT_STAT_FIELDS,
  defaultAdminPetCombatStats,
} from '../../types/petCombatStats';
import type {
  AdminEquipmentDetail,
  AdminEquipmentListFilters,
  AdminEquipmentSummary,
  AdminUpsertEquipmentPayload,
} from '../../types/equipmentDefinition';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import {
  EQUIPMENT_SLOT_OPTIONS,
  defaultEquipmentValues,
  defaultMedicinePouchExtra,
} from '../../types/equipmentDefinition';

interface EquipmentFormValues extends AdminUpsertEquipmentPayload {
  enhance_atk_per_level: number;
  enhance_def_per_level: number;
  enhance_hp_max_per_level: number;
  allowed_gem_types_text: string;
}

// 系统装备管理页：维护 item_definition + item_equipment_extra 人物装备模板。
export function EquipmentDefinitionPage() {
  const [filterForm] = Form.useForm<AdminEquipmentListFilters>();
  const [editorForm] = Form.useForm<EquipmentFormValues>();
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
  const equipSlot = Form.useWatch('equip_slot', editorForm);

  useEffect(() => {
    void loadRows(filters, page, pageSize);
  }, [filters, page, pageSize]);

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
      editorForm.setFieldsValue(mapPayloadToForm(defaultEquipmentValues()));
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
    const payload = mapFormToPayload(values);
    try {
      if (editingRecord) {
        await updateAdminEquipmentDefinition(editingRecord.item_id, payload);
        message.success('装备模板已更新');
      } else {
        await createAdminEquipmentDefinition(payload);
        message.success('装备模板已创建');
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

  const columns = useMemo<ColumnsType<AdminEquipmentSummary>>(
    () => [
      { title: '物品ID', dataIndex: 'item_id', width: 90 },
      { title: '编码', dataIndex: 'item_code', width: 140 },
      { title: '名称', dataIndex: 'item_name', width: 160 },
      { title: '部位', dataIndex: 'equip_slot_label', width: 100 },
      { title: '佩戴等级', dataIndex: 'required_level', width: 90 },
      { title: '品质', dataIndex: 'quality', width: 70 },
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

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Typography.Title level={4} style={{ marginTop: 0 }}>系统装备管理</Typography.Title>
        <Typography.Text type="secondary">
          维护人物可穿戴装备模板；创建时同时写入 item_definition 与 item_equipment_extra。
        </Typography.Text>
      </Card>

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
              <Descriptions.Item label="佩戴等级">{detail.required_level}</Descriptions.Item>
              <Descriptions.Item label="品质">{detail.quality}</Descriptions.Item>
              <Descriptions.Item label="强化">{detail.can_enhance ? `最高 +${detail.max_enhance_level}` : '不可强化'}</Descriptions.Item>
              <Descriptions.Item label="套装ID">{detail.set_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="五维" span={2}>
                {`生命${detail.base_hp} / 法力${detail.base_mana} / 攻${detail.base_atk} / 防${detail.base_def} / 速${detail.base_spd}`}
              </Descriptions.Item>
              <Descriptions.Item label="介绍" span={2}>{detail.desc || '-'}</Descriptions.Item>
            </Descriptions>
            {!detail.appearance_only ? (
              <Descriptions bordered column={2} size="small" title="次要战斗属性">
                {ADMIN_PET_COMBAT_STAT_FIELDS.map((field) => (
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
              <Form.Item label="物品ID" name="item_id" rules={[{ required: true, message: '请输入物品ID' }]}>
                <InputNumber min={1} style={{ width: '100%' }} disabled={!!editingRecord} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="编码" name="item_code" rules={[{ required: true, message: '请输入编码' }]}>
                <Input />
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
            <Col xs={12} md={4}><Form.Item label="品质" name="quality"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={4}><Form.Item label="稀有度" name="rarity"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}><Form.Item label="介绍" name="desc"><Input.TextArea rows={2} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="图标" name="icon"><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="绑定类型" name="bind_type"><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="套装ID" name="set_id"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="职业限制" name="career_limit"><Input placeholder="留空表示不限" /></Form.Item></Col>
            <Col xs={8} md={4}><Form.Item label="可出售" name="can_sell" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={8} md={4}><Form.Item label="可存仓" name="can_store" valuePropName="checked"><Switch /></Form.Item></Col>
            <Col xs={8} md={4}><Form.Item label="启用" name="is_enabled" valuePropName="checked"><Switch /></Form.Item></Col>
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
              <Divider plain>基础属性</Divider>
              <Row gutter={16}>
                <Col xs={12} md={6}><Form.Item label="生命" name="base_hp"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="法力" name="base_mana"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="攻击" name="base_atk"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="防御" name="base_def"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="速度" name="base_spd"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
              </Row>
              <Divider plain>次要战斗属性</Divider>
              <Row gutter={16}>
                {ADMIN_PET_COMBAT_STAT_FIELDS.map((field) => (
                  <Col xs={12} md={6} key={field.key}>
                    <Form.Item label={field.label} name={['combat_stats', field.key]}>
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                ))}
              </Row>
              <Divider plain>强化与镶嵌</Divider>
              <Row gutter={16}>
                <Col xs={12} md={6}><Form.Item label="可强化" name="can_enhance" valuePropName="checked"><Switch /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="最高强化" name="max_enhance_level"><InputNumber min={0} max={15} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="每级+攻击" name="enhance_atk_per_level"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="每级+防御" name="enhance_def_per_level"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="每级+生命上限" name="enhance_hp_max_per_level"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item label="镶嵌孔数" name="socket_count"><InputNumber min={0} max={8} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={12}>
                  <Form.Item label="允许宝石类型" name="allowed_gem_types_text" extra="英文逗号分隔，如 attack,defense">
                    <Input placeholder="attack,defense" />
                  </Form.Item>
                </Col>
              </Row>
            </>
          ) : null}
        </Form>
      </Modal>
    </Space>
  );
}

function mapDetailToForm(detail: AdminEquipmentDetail): EquipmentFormValues {
  return {
    ...detail,
    combat_stats: detail.combat_stats ?? defaultAdminPetCombatStats(),
    enhance_atk_per_level: detail.enhance_per_level_stats?.atk ?? 0,
    enhance_def_per_level: detail.enhance_per_level_stats?.def ?? 0,
    enhance_hp_max_per_level: detail.enhance_per_level_stats?.hp_max ?? 0,
    allowed_gem_types_text: (detail.allowed_gem_types ?? []).join(','),
    medicine_pouch: detail.medicine_pouch ?? defaultMedicinePouchExtra(),
  };
}

function mapPayloadToForm(payload: AdminUpsertEquipmentPayload): EquipmentFormValues {
  return {
    ...payload,
    enhance_atk_per_level: payload.enhance_per_level_stats?.atk ?? 0,
    enhance_def_per_level: payload.enhance_per_level_stats?.def ?? 0,
    enhance_hp_max_per_level: payload.enhance_per_level_stats?.hp_max ?? 0,
    allowed_gem_types_text: (payload.allowed_gem_types ?? []).join(','),
    medicine_pouch: payload.medicine_pouch ?? defaultMedicinePouchExtra(),
  };
}

function mapFormToPayload(values: EquipmentFormValues): AdminUpsertEquipmentPayload {
  const enhanceStats: Record<string, number> = {};
  if (values.enhance_atk_per_level > 0) {
    enhanceStats.atk = Number(values.enhance_atk_per_level);
  }
  if (values.enhance_def_per_level > 0) {
    enhanceStats.def = Number(values.enhance_def_per_level);
  }
  if (values.enhance_hp_max_per_level > 0) {
    enhanceStats.hp_max = Number(values.enhance_hp_max_per_level);
  }
  const allowedGemTypes = values.allowed_gem_types_text
    ? values.allowed_gem_types_text.split(',').map((item) => item.trim()).filter(Boolean)
    : [];
  return {
    item_id: Number(values.item_id),
    item_code: values.item_code,
    item_name: values.item_name,
    desc: values.desc,
    icon: values.icon,
    quality: Number(values.quality),
    rarity: Number(values.rarity),
    required_level: Number(values.required_level),
    bind_type: values.bind_type,
    can_sell: values.can_sell,
    can_store: values.can_store,
    is_enabled: values.is_enabled,
    equip_slot: values.equip_slot,
    career_limit: values.career_limit,
    can_enhance: values.can_enhance,
    max_enhance_level: Number(values.max_enhance_level),
    set_id: Number(values.set_id),
    appearance_skin_id: values.appearance_skin_id,
    appearance_only: values.equip_slot === 'costume',
    base_hp: Number(values.base_hp),
    base_mana: Number(values.base_mana),
    base_atk: Number(values.base_atk),
    base_def: Number(values.base_def),
    base_spd: Number(values.base_spd),
    combat_stats: values.combat_stats ?? defaultAdminPetCombatStats(),
    enhance_per_level_stats: enhanceStats,
    socket_count: Number(values.socket_count),
    allowed_gem_types: allowedGemTypes,
    medicine_pouch: values.equip_slot === 'medicine_pouch' ? (values.medicine_pouch ?? defaultMedicinePouchExtra()) : undefined,
  };
}
