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
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { GrantPetFromTemplateModal } from '../../components/GrantPetFromTemplateModal';
import { SkillReferenceText } from '../../components/SkillReferenceText';
import { useSkillReferenceMap } from '../../hooks/useSkillReferenceMap';
import {
  deleteAdminPet,
  fetchAdminPetDetail,
  fetchAdminPets,
  updateAdminPet,
} from '../../services/pet';
import { updateAdminPlayerPetLineup } from '../../services/player';
import type {
  AdminPetDetail,
  AdminPetSummary,
  AdminUpdatePetPayload,
} from '../../types/pet';
import { formatPetQualityLabel, getPetQualityTagColor, PET_QUALITY_OPTIONS } from '../../constants/petQuality';
import {
  ADMIN_PET_COMBAT_STAT_FIELDS,
  type AdminPetCombatStats,
} from '../../types/petCombatStats';
import { formatSkillReferenceInput, parseSkillReferenceInput, type SkillReferenceMap } from '../../utils/skillReference';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

interface PetFormValues extends PetFormBaseValues, AdminPetCombatStats {}

interface PetFormBaseValues {
  player_id?: number;
  pet_id: number;
  custom_name: string;
  level: number;
  exp: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  skill_names_text: string;
}

interface PlayerPetSectionProps {
  playerId: number;
  playerName: string;
}

// 玩家详情/编辑页内的宠物区块：按 player_id 拉取列表，并支持查看、编辑、新增与删除。
export function PlayerPetSection({ playerId, playerName }: PlayerPetSectionProps) {
  const { map: skillReferenceMap } = useSkillReferenceMap();
  const [editorForm] = Form.useForm<PetFormValues>();
  const [rows, setRows] = useState<AdminPetSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [petDetailOpen, setPetDetailOpen] = useState(false);
  const [petDetailLoading, setPetDetailLoading] = useState(false);
  const [petDetail, setPetDetail] = useState<AdminPetDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [grantModalOpen, setGrantModalOpen] = useState(false);
  const [editorDetailLoading, setEditorDetailLoading] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminPetDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [lineupUpdatingUID, setLineupUpdatingUID] = useState<number | null>(null);

  useEffect(() => {
    void loadPets();
  }, [playerId]);

  async function loadPets() {
    setLoading(true);
    try {
      const result = await fetchAdminPets({
        filters: { player_id: String(playerId) },
        page: 1,
        pageSize: 100,
      });
      setRows(result.items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家宠物失败');
      setRows([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(petUID: number) {
    setPetDetailOpen(true);
    setPetDetailLoading(true);
    setPetDetail(null);
    try {
      setPetDetail(await fetchAdminPetDetail(petUID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物详情失败');
      setPetDetailOpen(false);
    } finally {
      setPetDetailLoading(false);
    }
  }

  async function handleOpenEditor(petUID: number) {
    setEditorOpen(true);
    setEditingRecord(null);
    editorForm.resetFields();
    if (!petUID) {
      return;
    }
    setEditorDetailLoading(true);
    try {
      const result = await fetchAdminPetDetail(petUID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToForm(result, skillReferenceMap));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物编辑数据失败');
      setEditorOpen(false);
    } finally {
      setEditorDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: PetFormValues) {
    if (!editingRecord) {
      return;
    }
    setSaving(true);
    try {
      await updateAdminPet(editingRecord.pet_uid, mapFormToUpdatePayload(values, skillReferenceMap));
      message.success('宠物更新成功');
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadPets();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存宠物失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleToggleLineup(record: AdminPetSummary, checked: boolean) {
    let nextLineup: number[] = [];
    if (checked) {
      if (record.in_lineup) {
        return;
      }
      // 同时只能出战一只，开启新宠物会替换当前出战宠物。
      nextLineup = [record.pet_uid];
    }
    setLineupUpdatingUID(record.pet_uid);
    try {
      await updateAdminPlayerPetLineup(playerId, { pet_uids: nextLineup });
      message.success(checked ? '已设为出战宠物' : '已取消出战');
      await loadPets();
      if (petDetail?.pet_uid === record.pet_uid) {
        setPetDetail(await fetchAdminPetDetail(record.pet_uid));
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新出战状态失败');
    } finally {
      setLineupUpdatingUID(null);
    }
  }

  async function handleDelete(petUID: number) {
    setDeletingID(petUID);
    try {
      await deleteAdminPet(petUID);
      message.success('宠物已删除');
      if (petDetail?.pet_uid === petUID) {
        setPetDetailOpen(false);
        setPetDetail(null);
      }
      await loadPets();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除宠物失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminPetSummary>>(
    () => [
      {
        title: '宠物系统名',
        dataIndex: 'pet_name',
        key: 'pet_name',
        width: 140,
        render: (_value: string, record) => formatPetSystemName(record),
      },
      {
        title: '玩家自定义名',
        dataIndex: 'custom_name',
        key: 'custom_name',
        width: 140,
        render: (_value: string, record) => formatPetCustomName(record),
      },
      { title: '等级', dataIndex: 'level', key: 'level', width: 70 },
      {
        title: '品质',
        dataIndex: 'quality',
        key: 'quality',
        width: 100,
        render: (value: number) => <Tag color={getPetQualityTagColor(value)}>{formatPetQualityLabel(value)}</Tag>,
      },
      { title: '生命', key: 'hp', width: 100, render: (_value, record) => `${record.hp}/${record.hp_max}` },
      { title: '攻/防/速', key: 'stats', width: 110, render: (_value, record) => `${record.atk}/${record.def}/${record.spd}` },
      {
        title: '出战',
        dataIndex: 'in_lineup',
        key: 'in_lineup',
        width: 90,
        render: (value: boolean, record) => (
          <span onClick={(event) => event.stopPropagation()}>
            <Switch
              checked={value}
              loading={lineupUpdatingUID === record.pet_uid}
              checkedChildren="是"
              unCheckedChildren="否"
              onChange={(checked) => void handleToggleLineup(record, checked)}
            />
          </span>
        ),
      },
      {
        title: '操作',
        key: 'actions',
        width: 100,
        render: (_value, record) => (
          <span onClick={(event) => event.stopPropagation()}>
            <TableActionDropdown
              loading={deletingID === record.pet_uid}
              actions={[
                { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.pet_uid) },
                { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor(record.pet_uid) },
                {
                  key: 'delete',
                  label: '删除',
                  danger: true,
                  confirm: { title: '确认删除这个宠物吗？' },
                  onClick: () => void handleDelete(record.pet_uid),
                },
              ]}
            />
          </span>
        ),
      },
    ],
    [deletingID, lineupUpdatingUID, rows],
  );

  return (
    <>
      <Card
        size="small"
        title="宠物信息"
        extra={(
          <Button type="primary" size="small" onClick={() => setGrantModalOpen(true)}>
            新增宠物
          </Button>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="pet_uid"
          loading={loading}
          size="small"
          locale={{ emptyText: <Empty description={`${playerName} 还没有宠物`} /> }}
          scroll={{ x: 900 }}
          pagination={false}
          onRow={(record) => ({
            onClick: () => void handleViewDetail(record.pet_uid),
            style: { cursor: 'pointer' },
          })}
        />
        <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
          点击表格行可查看详情；「出战」开关会立即写入服务端，同时只能出战 1 只，也可全部不出战。
        </Typography.Text>
      </Card>

      <Drawer
        title={petDetail ? `宠物详情 · ${formatPetDisplayTitle(petDetail)}` : '宠物详情'}
        width={560}
        open={petDetailOpen}
        onClose={() => setPetDetailOpen(false)}
        destroyOnClose
        extra={petDetail ? (
          <TableActionDropdown
            buttonType="default"
            loading={deletingID === petDetail.pet_uid}
            actions={[
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor(petDetail.pet_uid) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这个宠物吗？' },
                onClick: () => void handleDelete(petDetail.pet_uid),
              },
            ]}
          />
        ) : null}
      >
        {petDetailLoading ? (
          <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载宠物详情..." />
          </div>
        ) : petDetail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="宠物系统名">{formatPetSystemName(petDetail)}</Descriptions.Item>
              <Descriptions.Item label="玩家自定义名">{formatPetCustomName(petDetail)}</Descriptions.Item>
              <Descriptions.Item label="宠物UID">{petDetail.pet_uid}</Descriptions.Item>
              <Descriptions.Item label="玩家ID">{petDetail.player_id}</Descriptions.Item>
              <Descriptions.Item label="玩家名">{petDetail.player_name}</Descriptions.Item>
              <Descriptions.Item label="模板ID">{petDetail.pet_id}</Descriptions.Item>
              <Descriptions.Item label="等级">{petDetail.level}</Descriptions.Item>
              <Descriptions.Item label="经验">{petDetail.exp}</Descriptions.Item>
              <Descriptions.Item label="品质">
                <Tag color={getPetQualityTagColor(petDetail.quality)}>{formatPetQualityLabel(petDetail.quality)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="出战">
                <Switch
                  checked={petDetail.in_lineup}
                  loading={lineupUpdatingUID === petDetail.pet_uid}
                  checkedChildren="是"
                  unCheckedChildren="否"
                  onChange={(checked) => void handleToggleLineup(petDetail, checked)}
                />
              </Descriptions.Item>
              <Descriptions.Item label="生命">{`${petDetail.hp}/${petDetail.hp_max}`}</Descriptions.Item>
              <Descriptions.Item label="攻/防/速">{`${petDetail.atk}/${petDetail.def}/${petDetail.spd}`}</Descriptions.Item>
              <Descriptions.Item label="法力">{petDetail.mana}</Descriptions.Item>
              <Descriptions.Item label="技能" span={2}>
                <SkillReferenceText skillIds={petDetail.skill_ids} map={skillReferenceMap} emptyText="无" />
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{formatDateTime(petDetail.created_at)}</Descriptions.Item>
              <Descriptions.Item label="更新时间">{formatDateTime(petDetail.updated_at)}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="次要战斗属性">
              {ADMIN_PET_COMBAT_STAT_FIELDS.map((field) => (
                <Descriptions.Item key={field.key} label={field.label}>
                  {petDetail[field.key]}
                </Descriptions.Item>
              ))}
            </Descriptions>
          </Space>
        ) : null}
      </Drawer>

      <GrantPetFromTemplateModal
        open={grantModalOpen}
        fixedPlayerId={playerId}
        fixedPlayerName={playerName}
        onCancel={() => setGrantModalOpen(false)}
        onSuccess={() => void loadPets()}
      />

      <Modal
        title={editingRecord ? `编辑宠物 · ${formatPetDisplayTitle(editingRecord)}` : '编辑宠物'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingRecord(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={860}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
        okText="保存修改"
        cancelText="取消"
      >
        {editorDetailLoading ? (
          <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载宠物编辑数据..." />
          </div>
        ) : (
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item label="宠物模板ID" name="pet_id" rules={[{ required: true, message: '请输入宠物模板ID' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="玩家自定义名" name="custom_name" extra="留空表示尚未设置自定义名，客户端将回退展示系统名。">
                <Input placeholder="例如：小火龙" maxLength={64} />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}><Form.Item label="等级" name="level"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="经验" name="exp"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}>
              <Form.Item label="品质" name="quality" rules={[{ required: true, message: '请选择品质' }]}>
                <Select options={PET_QUALITY_OPTIONS.map((item) => ({ value: item.value, label: item.label }))} />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}><Form.Item label="法力" name="mana"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="生命" name="hp"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="生命上限" name="hp_max"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="攻击" name="atk"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="防御" name="def"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="速度" name="spd"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}>
              <Form.Item label="技能" name="skill_names_text" extra="填写系统技能名称，多个用英文逗号分隔，例如 普通攻击,火花冲击">
                <Input placeholder="普通攻击,火花冲击" />
              </Form.Item>
            </Col>
          </Row>
          <Divider orientation="left" plain>次要战斗属性</Divider>
          <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
            保存时服务端会按 pet_combat_stat_cap 封顶表自动截断超限数值。
          </Typography.Text>
          <Row gutter={16}>
            {ADMIN_PET_COMBAT_STAT_FIELDS.map((field) => (
              <Col xs={12} md={6} key={field.key}>
                <Form.Item label={field.label} name={field.key}>
                  <InputNumber min={0} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            ))}
          </Row>
        </Form>
        )}
      </Modal>
    </>
  );
}

function mapDetailToForm(detail: AdminPetDetail, skillReferenceMap: SkillReferenceMap): PetFormValues {
  return {
    pet_id: detail.pet_id,
    custom_name: detail.custom_name ?? '',
    level: detail.level,
    exp: detail.exp,
    quality: detail.quality,
    hp: detail.hp,
    hp_max: detail.hp_max,
    atk: detail.atk,
    def: detail.def,
    spd: detail.spd,
    mana: detail.mana,
    skill_names_text: formatSkillReferenceInput(detail.skill_ids, skillReferenceMap),
    spirit: detail.spirit,
    spirit_max: detail.spirit_max,
    hit_pct: detail.hit_pct,
    dodge_pct: detail.dodge_pct,
    crit_rate_pct: detail.crit_rate_pct,
    crit_dmg_pct: detail.crit_dmg_pct,
    physical_resist_pct: detail.physical_resist_pct,
    reverse_physical_resist_pct: detail.reverse_physical_resist_pct,
    skill_resist_pct: detail.skill_resist_pct,
    reverse_skill_resist_pct: detail.reverse_skill_resist_pct,
    confusion_resist_pct: detail.confusion_resist_pct,
    sleep_resist_pct: detail.sleep_resist_pct,
    paralysis_resist_pct: detail.paralysis_resist_pct,
    seal_resist_pct: detail.seal_resist_pct,
    curse_resist_pct: detail.curse_resist_pct,
    crit_dmg_resist_pct: detail.crit_dmg_resist_pct,
    crit_resist_pct: detail.crit_resist_pct,
    character_resist_pct: detail.character_resist_pct,
    pet_resist_pct: detail.pet_resist_pct,
    guard: detail.guard,
    talent_dmg_pct: detail.talent_dmg_pct,
    talent_reduce_pct: detail.talent_reduce_pct,
    element_adv_pct: detail.element_adv_pct,
    element_penalty_pct: detail.element_penalty_pct,
  };
}

function mapFormToUpdatePayload(values: PetFormValues, skillReferenceMap: SkillReferenceMap): AdminUpdatePetPayload {
  return {
    pet_id: values.pet_id,
    custom_name: values.custom_name ?? '',
    level: values.level,
    exp: values.exp,
    quality: values.quality,
    hp: values.hp,
    hp_max: values.hp_max,
    atk: values.atk,
    def: values.def,
    spd: values.spd,
    mana: values.mana,
    skill_ids: parseSkillReferenceInput(values.skill_names_text, skillReferenceMap),
    spirit: values.spirit,
    spirit_max: values.spirit_max,
    hit_pct: values.hit_pct,
    dodge_pct: values.dodge_pct,
    crit_rate_pct: values.crit_rate_pct,
    crit_dmg_pct: values.crit_dmg_pct,
    physical_resist_pct: values.physical_resist_pct,
    reverse_physical_resist_pct: values.reverse_physical_resist_pct,
    skill_resist_pct: values.skill_resist_pct,
    reverse_skill_resist_pct: values.reverse_skill_resist_pct,
    confusion_resist_pct: values.confusion_resist_pct,
    sleep_resist_pct: values.sleep_resist_pct,
    paralysis_resist_pct: values.paralysis_resist_pct,
    seal_resist_pct: values.seal_resist_pct,
    curse_resist_pct: values.curse_resist_pct,
    crit_dmg_resist_pct: values.crit_dmg_resist_pct,
    crit_resist_pct: values.crit_resist_pct,
    character_resist_pct: values.character_resist_pct,
    pet_resist_pct: values.pet_resist_pct,
    guard: values.guard,
    talent_dmg_pct: values.talent_dmg_pct,
    talent_reduce_pct: values.talent_reduce_pct,
    element_adv_pct: values.element_adv_pct,
    element_penalty_pct: values.element_penalty_pct,
  };
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', { hour12: false });
}

/** 展示系统宠物模板名称，缺失时回退到模板 ID。 */
function formatPetSystemName(record: Pick<AdminPetSummary, 'pet_name' | 'pet_id'>): string {
  const systemName = record.pet_name?.trim();
  if (systemName) {
    return systemName;
  }
  return record.pet_id > 0 ? `模板#${record.pet_id}` : '-';
}

/** 展示玩家自定义名，未设置时给出占位文案。 */
function formatPetCustomName(record: Pick<AdminPetSummary, 'custom_name'>): string {
  const customName = record.custom_name?.trim();
  return customName || '未设置';
}

/** 详情/编辑标题优先展示自定义名，其次系统名。 */
function formatPetDisplayTitle(record: Pick<AdminPetSummary, 'custom_name' | 'pet_name' | 'pet_uid'>): string {
  const customName = record.custom_name?.trim();
  if (customName) {
    return customName;
  }
  const systemName = record.pet_name?.trim();
  if (systemName) {
    return systemName;
  }
  return record.pet_uid > 0 ? `#${record.pet_uid}` : '宠物';
}
