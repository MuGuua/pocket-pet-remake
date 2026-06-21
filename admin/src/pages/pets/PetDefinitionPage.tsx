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
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { SkillReferenceText } from '../../components/SkillReferenceText';
import { useSkillReferenceMap } from '../../hooks/useSkillReferenceMap';
import {
  createAdminPetDefinition,
  deleteAdminPetDefinition,
  fetchAdminPetDefinitionDetail,
  fetchAdminPetDefinitions,
  updateAdminPetDefinition,
} from '../../services/petDefinition';
import type {
  AdminPetDefinitionDetail,
  AdminPetDefinitionListFilters,
  AdminPetDefinitionSummary,
  AdminUpsertPetDefinitionPayload,
} from '../../types/petDefinition';
import { formatSkillReferenceInput, parseSkillReferenceInput, type SkillReferenceMap } from '../../utils/skillReference';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

interface PetDefinitionFormValues extends AdminUpsertPetDefinitionPayload {
  skill_names_text?: string;
  innate_skill_names_text?: string;
  normal_skill_names_text?: string;
}

// 系统宠物模板页维护可召唤宠物白名单；停用或删除模板后，对应 pet_id 的玩家宠物将不可用。
export function PetDefinitionPage() {
  const { map: skillReferenceMap } = useSkillReferenceMap();
  const [filterForm] = Form.useForm<AdminPetDefinitionListFilters>();
  const [editorForm] = Form.useForm<PetDefinitionFormValues>();
  const [filters, setFilters] = useState<AdminPetDefinitionListFilters>({});
  const [rows, setRows] = useState<AdminPetDefinitionSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminPetDefinitionDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminPetDefinitionDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const acquireMethod = Form.useWatch('acquire_method', editorForm);
  const isWildCaptureTemplate = useMemo(() => {
    const value = String(acquireMethod ?? '').trim();
    return value === 'wild_capture' || value.includes('野外捕捉');
  }, [acquireMethod]);

  useEffect(() => {
    void loadDefinitions(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadDefinitions(nextFilters: AdminPetDefinitionListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminPetDefinitions({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统宠物模板失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(petID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminPetDefinitionDetail(petID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统宠物详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', petID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.setFieldsValue(defaultPetDefinitionValues());
      return;
    }
    if (!petID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminPetDefinitionDetail(petID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToFormValues(result, skillReferenceMap));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统宠物编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmit(values: PetDefinitionFormValues) {
    setSaving(true);
    try {
      const payload = buildPayloadFromForm(values, skillReferenceMap);
      if (editingRecord) {
        await updateAdminPetDefinition(editingRecord.pet_id, payload);
        message.success('系统宠物模板更新成功');
      } else {
        await createAdminPetDefinition(payload);
        message.success('系统宠物模板创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      await loadDefinitions(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存系统宠物模板失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(petID: number) {
    try {
      await deleteAdminPetDefinition(petID);
      message.success('系统宠物模板已删除');
      if (detail?.pet_id === petID) {
        setDetail(null);
        setDetailOpen(false);
      }
      await loadDefinitions(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除系统宠物模板失败');
    }
  }

  const columns = useMemo<ColumnsType<AdminPetDefinitionSummary>>(
    () => [
      { title: '宠物ID', dataIndex: 'pet_id', key: 'pet_id', width: 100, fixed: 'left' },
      { title: '名称', dataIndex: 'pet_name', key: 'pet_name', width: 160 },
      { title: '品质', dataIndex: 'quality', key: 'quality', width: 90 },
      { title: '等级', dataIndex: 'level', key: 'level', width: 90 },
      { title: '获取方式', dataIndex: 'acquire_method', key: 'acquire_method', width: 180, ellipsis: true },
      { title: '战斗外观ID', dataIndex: 'skin_id', key: 'skin_id', width: 140, ellipsis: true },
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
              { key: 'view', label: '详情', onClick: () => void handleViewDetail(record.pet_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.pet_id) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这个系统宠物模板吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => void handleDelete(record.pet_id),
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
      <Card
        title="系统宠物列表"
        extra={(
          <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }}>
            <Form.Item name="pet_id" label="宠物ID">
              <Input allowClear placeholder="宠物ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="name" label="名称">
              <Input allowClear placeholder="宠物名称" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用">
              <Select allowClear placeholder="状态" style={{ width: 90 }} options={[{ label: '启用', value: 'true' }, { label: '停用', value: 'false' }]} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增宠物</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="pet_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有系统宠物模板" /> }}
          scroll={{ x: 1100 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条模板`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </Card>

      <Drawer title={detail ? `系统宠物详情 · ${detail.pet_name}` : '系统宠物详情'} width={720} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? (
          <Typography.Text type="secondary">正在加载系统宠物详情...</Typography.Text>
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small" title="基础信息">
              <Descriptions.Item label="宠物ID">{detail.pet_id}</Descriptions.Item>
              <Descriptions.Item label="名称">{detail.pet_name}</Descriptions.Item>
              <Descriptions.Item label="获取方式">{detail.acquire_method || '-'}</Descriptions.Item>
              <Descriptions.Item label="战斗外观ID">{detail.skin_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>{detail.description || '-'}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="基础数值">
              <Descriptions.Item label="等级">{detail.base_stats.level}</Descriptions.Item>
              <Descriptions.Item label="品质">{detail.base_stats.quality}</Descriptions.Item>
              <Descriptions.Item label="生命">{detail.base_stats.hp} / {detail.base_stats.hp_max}</Descriptions.Item>
              <Descriptions.Item label="攻击">{detail.base_stats.atk}</Descriptions.Item>
              <Descriptions.Item label="防御">{detail.base_stats.def}</Descriptions.Item>
              <Descriptions.Item label="速度">{detail.base_stats.spd}</Descriptions.Item>
              <Descriptions.Item label="法力">{detail.base_stats.mana}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="成长资质">
              <Descriptions.Item label="生命资质">{detail.growth_aptitudes.hp_apt}</Descriptions.Item>
              <Descriptions.Item label="攻击资质">{detail.growth_aptitudes.atk_apt}</Descriptions.Item>
              <Descriptions.Item label="防御资质">{detail.growth_aptitudes.def_apt}</Descriptions.Item>
              <Descriptions.Item label="速度资质">{detail.growth_aptitudes.spd_apt}</Descriptions.Item>
              <Descriptions.Item label="法力资质">{detail.growth_aptitudes.mana_apt}</Descriptions.Item>
            </Descriptions>
            {(detail.acquire_method === 'wild_capture' || detail.acquire_method.includes('野外捕捉')) ? (
              <Descriptions bordered column={2} size="small" title="野外捕捉资质 Roll 范围">
                <Descriptions.Item label="生命">{detail.aptitude_roll_ranges.hp_apt_roll_min} ~ {detail.aptitude_roll_ranges.hp_apt_roll_max}</Descriptions.Item>
                <Descriptions.Item label="攻击">{detail.aptitude_roll_ranges.atk_apt_roll_min} ~ {detail.aptitude_roll_ranges.atk_apt_roll_max}</Descriptions.Item>
                <Descriptions.Item label="防御">{detail.aptitude_roll_ranges.def_apt_roll_min} ~ {detail.aptitude_roll_ranges.def_apt_roll_max}</Descriptions.Item>
                <Descriptions.Item label="速度">{detail.aptitude_roll_ranges.spd_apt_roll_min} ~ {detail.aptitude_roll_ranges.spd_apt_roll_max}</Descriptions.Item>
                <Descriptions.Item label="法力">{detail.aptitude_roll_ranges.mana_apt_roll_min} ~ {detail.aptitude_roll_ranges.mana_apt_roll_max}</Descriptions.Item>
              </Descriptions>
            ) : null}
            <Descriptions bordered column={1} size="small" title="技能">
              <Descriptions.Item label="兼容技能列表">
                <SkillReferenceText skillIds={detail.skill_ids} map={skillReferenceMap} />
              </Descriptions.Item>
              <Descriptions.Item label="天生技（最多5）">
                <SkillReferenceText skillIds={detail.innate_skill_ids ?? []} map={skillReferenceMap} />
              </Descriptions.Item>
              <Descriptions.Item label="普通技（3槽）">
                <SkillReferenceText skillIds={detail.normal_skill_ids ?? []} map={skillReferenceMap} />
              </Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑系统宠物 · ${editingRecord.pet_name}` : '新增系统宠物模板'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={760}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText={editingRecord ? '保存修改' : '创建模板'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Row gutter={16}>
            <Col xs={24} md={8}><Form.Item label="宠物ID" name="pet_id" rules={[{ required: true, message: '请输入宠物ID' }]}><InputNumber min={1} disabled={Boolean(editingRecord)} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="宠物名称" name="pet_name" rules={[{ required: true, message: '请输入宠物名称' }]}><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="获取方式" name="acquire_method" extra="野外捕捉请填写 wild_capture 或包含“野外捕捉”"><Input placeholder="例如：wild_capture、任务奖励" /></Form.Item></Col>
            <Col xs={24} md={8}>
              <Form.Item
                label="战斗外观ID"
                name="skin_id"
                extra="对应客户端 unit_skins/{skin_id}.tres；启用模板时必填"
                rules={[
                  ({ getFieldValue }) => ({
                    validator: async (_rule, value: string | undefined) => {
                      if (!getFieldValue('is_enabled')) {
                        return;
                      }
                      if (!value || !value.trim()) {
                        throw new Error('启用模板时必须填写战斗外观ID');
                      }
                    },
                  }),
                ]}
              >
                <Input placeholder="例如：嫩叶犬_001" />
              </Form.Item>
            </Col>
            <Col span={24}><Form.Item label="描述" name="description"><Input.TextArea rows={2} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="等级" name="level"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="品质" name="quality"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="当前生命" name="hp"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="生命上限" name="hp_max"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="攻击" name="atk"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="防御" name="def"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="速度" name="spd"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="法力" name="mana"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="生命资质" name="hp_apt"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="攻击资质" name="atk_apt"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="防御资质" name="def_apt"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="速度资质" name="spd_apt"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item label="法力资质" name="mana_apt"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            {isWildCaptureTemplate ? (
              <>
                <Col span={24}><Typography.Text type="secondary">以下 Roll 范围仅在野外捕捉发放时生效，任务/运营发放仍使用上方固定资质。</Typography.Text></Col>
                <Col xs={24} md={6}><Form.Item label="生命 Roll 最小" name="hp_apt_roll_min" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="生命 Roll 最大" name="hp_apt_roll_max" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="攻击 Roll 最小" name="atk_apt_roll_min" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="攻击 Roll 最大" name="atk_apt_roll_max" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="防御 Roll 最小" name="def_apt_roll_min" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="防御 Roll 最大" name="def_apt_roll_max" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="速度 Roll 最小" name="spd_apt_roll_min" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="速度 Roll 最大" name="spd_apt_roll_max" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="法力 Roll 最小" name="mana_apt_roll_min" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item label="法力 Roll 最大" name="mana_apt_roll_max" rules={[{ required: true, message: '必填' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
              </>
            ) : null}
            <Col span={24}><Form.Item label="天生技（最多5）" name="innate_skill_names_text" extra="发放时写入实例天生技槽"><Input placeholder="例如：撕咬,利爪" /></Form.Item></Col>
            <Col span={24}><Form.Item label="普通技（3槽）" name="normal_skill_names_text" extra="默认开启的普通技槽"><Input placeholder="例如：普通攻击,火花冲击" /></Form.Item></Col>
            <Col span={24}><Form.Item label="兼容技能列表" name="skill_names_text" extra="旧字段；留空时由天生+普通技自动生成"><Input placeholder="普通攻击,火花冲击" /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="启用" name="is_enabled" valuePropName="checked"><Switch /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultPetDefinitionValues(): PetDefinitionFormValues {
  return {
    pet_id: 103,
    pet_name: '新系统宠物',
    description: '',
    acquire_method: '运营发放',
    is_enabled: true,
    skin_id: '',
    level: 1,
    quality: 1,
    hp: 20,
    hp_max: 20,
    atk: 10,
    def: 8,
    spd: 9,
    mana: 12,
    hp_apt: 10,
    atk_apt: 10,
    def_apt: 10,
    spd_apt: 10,
    mana_apt: 10,
    hp_apt_roll_min: 8,
    hp_apt_roll_max: 14,
    atk_apt_roll_min: 8,
    atk_apt_roll_max: 13,
    def_apt_roll_min: 8,
    def_apt_roll_max: 12,
    spd_apt_roll_min: 7,
    spd_apt_roll_max: 12,
    mana_apt_roll_min: 6,
    mana_apt_roll_max: 11,
    skill_ids: [],
    innate_skill_ids: [],
    normal_skill_ids: [],
    skill_names_text: '',
    innate_skill_names_text: '',
    normal_skill_names_text: '普通攻击',
  };
}

function mapDetailToFormValues(detail: AdminPetDefinitionDetail, skillReferenceMap: SkillReferenceMap): PetDefinitionFormValues {
  return {
    pet_id: detail.pet_id,
    pet_name: detail.pet_name,
    description: detail.description,
    acquire_method: detail.acquire_method,
    is_enabled: detail.is_enabled,
    skin_id: detail.skin_id,
    level: detail.base_stats.level,
    quality: detail.base_stats.quality,
    hp: detail.base_stats.hp,
    hp_max: detail.base_stats.hp_max,
    atk: detail.base_stats.atk,
    def: detail.base_stats.def,
    spd: detail.base_stats.spd,
    mana: detail.base_stats.mana,
    hp_apt: detail.growth_aptitudes.hp_apt,
    atk_apt: detail.growth_aptitudes.atk_apt,
    def_apt: detail.growth_aptitudes.def_apt,
    spd_apt: detail.growth_aptitudes.spd_apt,
    mana_apt: detail.growth_aptitudes.mana_apt,
    hp_apt_roll_min: detail.aptitude_roll_ranges.hp_apt_roll_min,
    hp_apt_roll_max: detail.aptitude_roll_ranges.hp_apt_roll_max,
    atk_apt_roll_min: detail.aptitude_roll_ranges.atk_apt_roll_min,
    atk_apt_roll_max: detail.aptitude_roll_ranges.atk_apt_roll_max,
    def_apt_roll_min: detail.aptitude_roll_ranges.def_apt_roll_min,
    def_apt_roll_max: detail.aptitude_roll_ranges.def_apt_roll_max,
    spd_apt_roll_min: detail.aptitude_roll_ranges.spd_apt_roll_min,
    spd_apt_roll_max: detail.aptitude_roll_ranges.spd_apt_roll_max,
    mana_apt_roll_min: detail.aptitude_roll_ranges.mana_apt_roll_min,
    mana_apt_roll_max: detail.aptitude_roll_ranges.mana_apt_roll_max,
    skill_ids: detail.skill_ids,
    innate_skill_ids: detail.innate_skill_ids ?? [],
    normal_skill_ids: detail.normal_skill_ids ?? [],
    skill_names_text: formatSkillReferenceInput(detail.skill_ids, skillReferenceMap),
    innate_skill_names_text: formatSkillReferenceInput(detail.innate_skill_ids ?? [], skillReferenceMap),
    normal_skill_names_text: formatSkillReferenceInput(detail.normal_skill_ids ?? [], skillReferenceMap),
  };
}

function buildPayloadFromForm(values: PetDefinitionFormValues, skillReferenceMap: SkillReferenceMap): AdminUpsertPetDefinitionPayload {
  const innateSkillIDs = parseSkillReferenceInput(values.innate_skill_names_text ?? '', skillReferenceMap).slice(0, 5);
  const normalSkillIDs = parseSkillReferenceInput(values.normal_skill_names_text ?? '', skillReferenceMap).slice(0, 3);
  const legacySkillIDs = parseSkillReferenceInput(values.skill_names_text ?? '', skillReferenceMap);
  const skillIDs = legacySkillIDs.length > 0 ? legacySkillIDs : [...innateSkillIDs, ...normalSkillIDs];
  return {
    pet_id: Number(values.pet_id),
    pet_name: values.pet_name.trim(),
    description: values.description?.trim() ?? '',
    acquire_method: values.acquire_method?.trim() ?? '',
    is_enabled: Boolean(values.is_enabled),
    skin_id: values.skin_id?.trim() ?? '',
    level: Number(values.level),
    quality: Number(values.quality),
    hp: Number(values.hp),
    hp_max: Number(values.hp_max),
    atk: Number(values.atk),
    def: Number(values.def),
    spd: Number(values.spd),
    mana: Number(values.mana),
    hp_apt: Number(values.hp_apt),
    atk_apt: Number(values.atk_apt),
    def_apt: Number(values.def_apt),
    spd_apt: Number(values.spd_apt),
    mana_apt: Number(values.mana_apt),
    hp_apt_roll_min: Number(values.hp_apt_roll_min || 0),
    hp_apt_roll_max: Number(values.hp_apt_roll_max || 0),
    atk_apt_roll_min: Number(values.atk_apt_roll_min || 0),
    atk_apt_roll_max: Number(values.atk_apt_roll_max || 0),
    def_apt_roll_min: Number(values.def_apt_roll_min || 0),
    def_apt_roll_max: Number(values.def_apt_roll_max || 0),
    spd_apt_roll_min: Number(values.spd_apt_roll_min || 0),
    spd_apt_roll_max: Number(values.spd_apt_roll_max || 0),
    mana_apt_roll_min: Number(values.mana_apt_roll_min || 0),
    mana_apt_roll_max: Number(values.mana_apt_roll_max || 0),
    skill_ids: skillIDs,
    innate_skill_ids: innateSkillIDs,
    normal_skill_ids: normalSkillIDs,
  };
}
