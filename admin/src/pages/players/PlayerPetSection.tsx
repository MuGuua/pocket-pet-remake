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
import { PetSkillSlotListEditor } from '../../components/PetSkillSlotListEditor';
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
} from '../../types/pet';
import { formatPetQualityLabel, getPetQualityTagColor, PET_QUALITY_OPTIONS } from '../../constants/petQuality';
import {
  ADMIN_PET_COMBAT_STAT_FIELDS,
} from '../../types/petCombatStats';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import {
  formatPetDateTime,
  mapPetDetailToForm,
  mapPetFormToUpdatePayload,
  mergePlayerPetSkillIDs,
  type PetInstanceFormValues,
} from '../pets/petInstanceFormUtils';

interface PlayerPetSectionProps {
  playerId: number;
  playerName: string;
  onDataChanged?: () => void | Promise<void>;
}

// 玩家详情/编辑页内的宠物区块：按 player_id 拉取列表，并支持查看、编辑、新增与删除。
export function PlayerPetSection({ playerId, playerName, onDataChanged }: PlayerPetSectionProps) {
  const { map: skillReferenceMap } = useSkillReferenceMap();
  const [editorForm] = Form.useForm<PetInstanceFormValues>();
  const innateSkillIDs = Form.useWatch('innate_skill_ids', editorForm) ?? [];
  const normalSkillIDs = Form.useWatch('normal_skill_ids', editorForm) ?? [];
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

  useEffect(() => {
    const mergedSkillIDs = mergePlayerPetSkillIDs(innateSkillIDs, normalSkillIDs, []);
    editorForm.setFieldValue('skill_ids', mergedSkillIDs);
  }, [editorForm, innateSkillIDs, normalSkillIDs]);

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
      editorForm.setFieldsValue(mapPetDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物编辑数据失败');
      setEditorOpen(false);
    } finally {
      setEditorDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: PetInstanceFormValues) {
    if (!editingRecord) {
      return;
    }
    setSaving(true);
    try {
      await updateAdminPet(editingRecord.pet_uid, mapPetFormToUpdatePayload(values));
      message.success('宠物更新成功');
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadPets();
      if (onDataChanged) {
        await onDataChanged();
      }
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
      if (onDataChanged) {
        await onDataChanged();
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
      if (onDataChanged) {
        await onDataChanged();
      }
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
              <Descriptions.Item label="天生技" span={2}>
                <SkillReferenceText skillIds={petDetail.innate_skill_ids ?? []} map={skillReferenceMap} emptyText="无" />
              </Descriptions.Item>
              <Descriptions.Item label="普通技" span={2}>
                <SkillReferenceText skillIds={petDetail.normal_skill_ids ?? []} map={skillReferenceMap} emptyText="无" />
              </Descriptions.Item>
              <Descriptions.Item label="兼容战斗技能" span={2}>
                <SkillReferenceText skillIds={petDetail.skill_ids} map={skillReferenceMap} emptyText="无" />
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{formatPetDateTime(petDetail.created_at)}</Descriptions.Item>
              <Descriptions.Item label="更新时间">{formatPetDateTime(petDetail.updated_at)}</Descriptions.Item>
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
        onSuccess={() => {
          void loadPets();
          if (onDataChanged) {
            void onDataChanged();
          }
        }}
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
              <Form.Item
                label="天生技"
                name="innate_skill_ids"
                extra="会直接进入正式技能槽体系。"
              >
                <PetSkillSlotListEditor
                  categories={['pet', 'common']}
                  skillReferenceMap={skillReferenceMap}
                  description="按列表顺序维护玩家宠物天生技；这里的顺序就是运行时正式技能槽顺序。"
                />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item
                label="普通技"
                name="normal_skill_ids"
                extra="会直接进入正式技能槽体系。"
              >
                <PetSkillSlotListEditor
                  categories={['pet', 'common']}
                  skillReferenceMap={skillReferenceMap}
                  description="按列表顺序维护玩家宠物普通技；后台保存后会自动生成兼容 battle skill_ids。"
                />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item
                label="兼容战斗技能预览"
                name="skill_ids"
                extra="该字段会由「天生技 + 普通技」自动合并生成，主要用于兼容旧口径展示。"
              >
                <PetSkillSlotListEditor
                  categories={['pet', 'common']}
                  skillReferenceMap={skillReferenceMap}
                  disabled
                  description="只读预览：保存时后端会以正式技能槽为主，同时回写兼容 skill_ids。"
                />
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
