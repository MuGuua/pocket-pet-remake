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
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { PetSkillSlotListEditor } from '../../components/PetSkillSlotListEditor';
import { RichTextDisplay } from '../../components/RichTextDisplay';
import { RichTextEditor } from '../../components/RichTextEditor';
import { SkillReferenceText } from '../../components/SkillReferenceText';
import { TableActionDropdown } from '../../components/TableActionDropdown';
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
import {
  PET_QUALITY_OPTIONS,
  formatPetQualityLabel,
  getPetQualityTagColor,
  isWildCapturePetTemplate,
} from '../../constants/petQuality';
import { ADMIN_INTEGER_INPUT_PROPS, formatAdminInteger } from '../../utils/adminNumberInput';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

interface PetDefinitionFormValues extends Omit<AdminUpsertPetDefinitionPayload, 'acquire_method'> {
  /** 留空时由天生技 + 普通技自动合并。 */
  legacy_skill_ids?: number[];
}

const APTITUDE_ROWS = [
  { key: 'hp', label: '生命', color: 'red' },
  { key: 'atk', label: '攻击', color: 'orange' },
  { key: 'def', label: '防御', color: 'blue' },
  { key: 'spd', label: '速度', color: 'green' },
  { key: 'mana', label: '法力', color: 'purple' },
] as const;

const BASE_STAT_ROWS = [
  { key: 'hp', label: '当前生命', maxKey: 'hp_max', maxLabel: '生命上限' },
  { key: 'atk', label: '攻击' },
  { key: 'def', label: '防御' },
  { key: 'spd', label: '速度' },
  { key: 'mana', label: '法力' },
] as const;

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
  const innateSkillIDs = Form.useWatch('innate_skill_ids', editorForm) ?? [];
  const normalSkillIDs = Form.useWatch('normal_skill_ids', editorForm) ?? [];
  const legacySkillIDs = Form.useWatch('legacy_skill_ids', editorForm) ?? [];
  const showWildCaptureRollSection = isWildCapturePetTemplate(editingRecord?.acquire_method);
  const mergedSkillPreview = useMemo(() => {
    if (legacySkillIDs.length > 0) {
      return legacySkillIDs;
    }
    return [...innateSkillIDs, ...normalSkillIDs];
  }, [innateSkillIDs, legacySkillIDs, normalSkillIDs]);

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
      editorForm.setFieldsValue(mapDetailToFormValues(result));
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
      const payload = buildPayloadFromForm(values, editingRecord?.acquire_method ?? '');
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
      {
        title: '品质',
        dataIndex: 'quality',
        key: 'quality',
        width: 110,
        render: (value: number) => (
          <Tag color={getPetQualityTagColor(value)}>{formatPetQualityLabel(value)}</Tag>
        ),
      },
      { title: '等级', dataIndex: 'level', key: 'level', width: 90 },
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
          scroll={{ x: 980 }}
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

      <Drawer title={detail ? `系统宠物详情 · ${detail.pet_name}` : '系统宠物详情'} width={760} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? (
          <Typography.Text type="secondary">正在加载系统宠物详情...</Typography.Text>
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small" title="基础信息">
              <Descriptions.Item label="宠物ID">{detail.pet_id}</Descriptions.Item>
              <Descriptions.Item label="名称">{detail.pet_name}</Descriptions.Item>
              <Descriptions.Item label="品质">
                <Tag color={getPetQualityTagColor(detail.base_stats.quality)}>
                  {formatPetQualityLabel(detail.base_stats.quality)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="带出等级">{detail.base_stats.level}</Descriptions.Item>
              <Descriptions.Item label="战斗外观ID">{detail.skin_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>
                <RichTextDisplay value={detail.description} />
              </Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={3} size="small" title="基础战斗数值">
              <Descriptions.Item label="生命">{formatAdminInteger(detail.base_stats.hp)} / {formatAdminInteger(detail.base_stats.hp_max)}</Descriptions.Item>
              <Descriptions.Item label="攻击">{formatAdminInteger(detail.base_stats.atk)}</Descriptions.Item>
              <Descriptions.Item label="防御">{formatAdminInteger(detail.base_stats.def)}</Descriptions.Item>
              <Descriptions.Item label="速度">{formatAdminInteger(detail.base_stats.spd)}</Descriptions.Item>
              <Descriptions.Item label="法力">{formatAdminInteger(detail.base_stats.mana)}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={3} size="small" title="成长资质">
              {APTITUDE_ROWS.map((row) => (
                <Descriptions.Item key={row.key} label={row.label}>
                  <Tag color={row.color}>{formatAdminInteger(detail.growth_aptitudes[`${row.key}_apt` as keyof typeof detail.growth_aptitudes])}</Tag>
                </Descriptions.Item>
              ))}
            </Descriptions>
            {isWildCapturePetTemplate(detail.acquire_method) ? (
              <Descriptions bordered column={1} size="small" title="野外捕捉资质 Roll 范围">
                {APTITUDE_ROWS.map((row) => {
                  const minKey = `${row.key}_apt_roll_min` as keyof typeof detail.aptitude_roll_ranges;
                  const maxKey = `${row.key}_apt_roll_max` as keyof typeof detail.aptitude_roll_ranges;
                  return (
                    <Descriptions.Item key={row.key} label={row.label}>
                      {formatAdminInteger(detail.aptitude_roll_ranges[minKey])} ~ {formatAdminInteger(detail.aptitude_roll_ranges[maxKey])}
                    </Descriptions.Item>
                  );
                })}
              </Descriptions>
            ) : null}
            <Descriptions bordered column={1} size="small" title="技能配置">
              <Descriptions.Item label="天生技">
                <SkillReferenceText skillIds={detail.innate_skill_ids ?? []} map={skillReferenceMap} />
              </Descriptions.Item>
              <Descriptions.Item label="普通技">
                <SkillReferenceText skillIds={detail.normal_skill_ids ?? []} map={skillReferenceMap} />
              </Descriptions.Item>
              <Descriptions.Item label="兼容技能列表">
                <SkillReferenceText skillIds={detail.skill_ids} map={skillReferenceMap} />
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
        width={920}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={{
          body: {
            ...FIXED_FORM_MODAL_STYLES.body,
            height: 'min(760px, calc(100vh - 200px))',
          },
        }}
        okText={editingRecord ? '保存修改' : '创建模板'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Divider plain orientation="left">基础信息</Divider>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item label="宠物ID" name="pet_id" rules={[{ required: true, message: '请输入宠物ID' }]}>
                <InputNumber min={1} disabled={Boolean(editingRecord)} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="宠物名称" name="pet_name" rules={[{ required: true, message: '请输入宠物名称' }]}>
                <Input placeholder="例如：白色幻影" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="启用" name="is_enabled" valuePropName="checked">
                <Switch checkedChildren="启用" unCheckedChildren="停用" />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="品质" name="quality" rules={[{ required: true, message: '请选择品质' }]}>
                <Select options={PET_QUALITY_OPTIONS.map((item) => ({ value: item.value, label: item.label }))} />
              </Form.Item>
            </Col>
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
                <Input placeholder="例如：白色幻影_001" />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item label="描述 / 运营备注" name="description" extra="支持 BBCode 富文本，客户端宠物详情会原样渲染。">
                <RichTextEditor rows={4} placeholder="加点推荐、宝石推荐、不可交易说明等" />
              </Form.Item>
            </Col>
          </Row>

          <Divider plain orientation="left">等级与基础战斗数值</Divider>
          <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
            发放 1 级宠物时，战斗五维会按资质公式重算；此处数值主要用于模板展示与兼容旧发放逻辑。
          </Typography.Paragraph>
          <Row gutter={16}>
            <Col xs={12} md={6}>
              <Form.Item label="带出等级" name="level">
                <InputNumber min={1} max={100} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
              </Form.Item>
            </Col>
          </Row>
          <Card size="small" styles={{ body: { padding: 12 } }}>
            <Row gutter={[12, 12]}>
              <Col xs={12} md={8}>
                <Form.Item label="当前生命" name="hp" style={{ marginBottom: 0 }}>
                  <InputNumber min={1} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
                </Form.Item>
              </Col>
              <Col xs={12} md={8}>
                <Form.Item label="生命上限" name="hp_max" style={{ marginBottom: 0 }}>
                  <InputNumber min={1} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
                </Form.Item>
              </Col>
              {BASE_STAT_ROWS.filter((row) => row.key !== 'hp').map((row) => (
                <Col xs={12} md={8} key={row.key}>
                  <Form.Item label={row.label} name={row.key} style={{ marginBottom: 0 }}>
                    <InputNumber min={row.key === 'mana' ? 0 : 1} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
                  </Form.Item>
                </Col>
              ))}
            </Row>
          </Card>

          <Divider plain orientation="left">成长资质</Divider>
          <Card size="small" styles={{ body: { padding: 12 } }}>
            <Row gutter={[12, 12]}>
              {APTITUDE_ROWS.map((row) => (
                <Col xs={12} md={8} lg={4} key={row.key}>
                  <Form.Item
                    label={<Space size={6}><Tag color={row.color}>{row.label}</Tag><span>资质</span></Space>}
                    name={`${row.key}_apt`}
                    style={{ marginBottom: 0 }}
                  >
                    <InputNumber min={1} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
                  </Form.Item>
                </Col>
              ))}
            </Row>
          </Card>

          {showWildCaptureRollSection ? (
            <>
              <Divider plain orientation="left">野外捕捉 Roll 范围</Divider>
              <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
                仅在野外捕捉发放时随机 roll；任务/运营发放仍使用上方固定资质。
              </Typography.Paragraph>
              <Card size="small" styles={{ body: { padding: 12 } }}>
                <Row gutter={[12, 12]}>
                  {APTITUDE_ROWS.map((row) => (
                    <Col span={24} key={row.key}>
                      <Row gutter={12} align="middle">
                        <Col xs={24} md={4}>
                          <Tag color={row.color} style={{ marginInlineEnd: 0 }}>{row.label}</Tag>
                        </Col>
                        <Col xs={12} md={10}>
                          <Form.Item
                            label="Roll 最小"
                            name={`${row.key}_apt_roll_min`}
                            rules={[{ required: true, message: '必填' }]}
                            style={{ marginBottom: 0 }}
                          >
                            <InputNumber min={1} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
                          </Form.Item>
                        </Col>
                        <Col xs={12} md={10}>
                          <Form.Item
                            label="Roll 最大"
                            name={`${row.key}_apt_roll_max`}
                            rules={[{ required: true, message: '必填' }]}
                            style={{ marginBottom: 0 }}
                          >
                            <InputNumber min={1} style={{ width: '100%' }} {...ADMIN_INTEGER_INPUT_PROPS} />
                          </Form.Item>
                        </Col>
                      </Row>
                    </Col>
                  ))}
                </Row>
              </Card>
            </>
          ) : null}

          <Divider plain orientation="left">技能配置</Divider>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Form.Item
              label="天生技"
              name="innate_skill_ids"
              extra="发放时写入实例天生技槽，最多 5 个。"
            >
              <PetSkillSlotListEditor
                maxCount={5}
                skillReferenceMap={skillReferenceMap}
                description="天生圣技/魂技；顺序影响兼容技能列表默认合并。"
              />
            </Form.Item>
            <Form.Item
              label="普通技（3 槽）"
              name="normal_skill_ids"
              extra="默认开启的普通技槽，最多 3 个。"
            >
              <PetSkillSlotListEditor
                maxCount={3}
                skillReferenceMap={skillReferenceMap}
                description="致命/涅槃/迅捷等普通技能卡。"
              />
            </Form.Item>
            <Form.Item
              label="兼容技能列表（可选）"
              name="legacy_skill_ids"
              extra="旧字段 skill_ids；留空时自动合并「天生技 + 普通技」。"
            >
              <PetSkillSlotListEditor
                maxCount={8}
                skillReferenceMap={skillReferenceMap}
                description="仅当需要与默认合并顺序不同时手动覆盖。"
              />
            </Form.Item>
            <Card size="small" title="保存后将写入的兼容 skill_ids" styles={{ body: { padding: 12 } }}>
              {mergedSkillPreview.length > 0 ? (
                <SkillReferenceText skillIds={mergedSkillPreview} map={skillReferenceMap} />
              ) : (
                <Typography.Text type="secondary">请先配置天生技或普通技。</Typography.Text>
              )}
            </Card>
          </Space>
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
    legacy_skill_ids: [],
  };
}

function mapDetailToFormValues(detail: AdminPetDefinitionDetail): PetDefinitionFormValues {
  const innateSkillIDs = detail.innate_skill_ids ?? [];
  const normalSkillIDs = detail.normal_skill_ids ?? [];
  const defaultMerged = [...innateSkillIDs, ...normalSkillIDs];
  const legacySkillIDs = arraysEqual(detail.skill_ids, defaultMerged) ? [] : detail.skill_ids;
  return {
    pet_id: detail.pet_id,
    pet_name: detail.pet_name,
    description: detail.description,
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
    innate_skill_ids: innateSkillIDs,
    normal_skill_ids: normalSkillIDs,
    legacy_skill_ids: legacySkillIDs,
  };
}

function buildPayloadFromForm(values: PetDefinitionFormValues, preservedAcquireMethod: string): AdminUpsertPetDefinitionPayload {
  const innateSkillIDs = (values.innate_skill_ids ?? []).slice(0, 5);
  const normalSkillIDs = (values.normal_skill_ids ?? []).slice(0, 3);
  const legacySkillIDs = values.legacy_skill_ids ?? [];
  const skillIDs = legacySkillIDs.length > 0 ? legacySkillIDs : [...innateSkillIDs, ...normalSkillIDs];
  return {
    pet_id: Number(values.pet_id),
    pet_name: values.pet_name.trim(),
    description: values.description?.trim() ?? '',
    acquire_method: preservedAcquireMethod.trim(),
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

function arraysEqual(left: number[], right: number[]): boolean {
  if (left.length !== right.length) {
    return false;
  }
  return left.every((value, index) => value === right[index]);
}
