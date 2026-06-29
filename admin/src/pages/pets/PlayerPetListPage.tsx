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
import type { AdminPetDetail, AdminPetListFilters, AdminPetSummary } from '../../types/pet';
import { formatPetQualityLabel, getPetQualityTagColor, PET_QUALITY_OPTIONS } from '../../constants/petQuality';
import { ADMIN_PET_COMBAT_STAT_FIELDS } from '../../types/petCombatStats';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';
import {
  formatPetDateTime,
  mapPetDetailToForm,
  mapPetFormToUpdatePayload,
  type PetInstanceFormValues,
} from './petInstanceFormUtils';

// 玩家宠物实例独立管理页：跨玩家检索 pet_uid / player_id / pet_id，支持详情、编辑、新增与删除。
export function PlayerPetListPage() {
  const { map: skillReferenceMap } = useSkillReferenceMap();
  const [filterForm] = Form.useForm<AdminPetListFilters>();
  const [editorForm] = Form.useForm<PetInstanceFormValues>();
  const [filters, setFilters] = useState<AdminPetListFilters>({});
  const [rows, setRows] = useState<AdminPetSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminPetDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [grantModalOpen, setGrantModalOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminPetDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [lineupUpdatingUID, setLineupUpdatingUID] = useState<number | null>(null);

  useEffect(() => {
    void loadPets(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadPets(nextFilters: AdminPetListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminPets({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物列表失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(petUID: number) {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      setDetail(await fetchAdminPetDetail(petUID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(petUID: number) {
    setEditorOpen(true);
    setEditingRecord(null);
    editorForm.resetFields();
    if (!petUID) {
      return;
    }
    setDetailLoading(true);
    try {
      const result = await fetchAdminPetDetail(petUID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapPetDetailToForm(result, skillReferenceMap));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: PetInstanceFormValues) {
    if (!editingRecord) {
      return;
    }
    setSaving(true);
    try {
      await updateAdminPet(editingRecord.pet_uid, mapPetFormToUpdatePayload(values, skillReferenceMap));
      message.success('宠物更新成功');
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadPets(filters, page, pageSize);
      if (detail && editingRecord.pet_uid === detail.pet_uid) {
        setDetail(await fetchAdminPetDetail(detail.pet_uid));
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存宠物失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleToggleLineup(record: AdminPetSummary | AdminPetDetail, checked: boolean) {
    const playerId = record.player_id;
    const nextLineup = checked ? [record.pet_uid] : [];
    setLineupUpdatingUID(record.pet_uid);
    try {
      await updateAdminPlayerPetLineup(playerId, { pet_uids: nextLineup });
      message.success(checked ? '已设为出战宠物' : '已取消出战');
      await loadPets(filters, page, pageSize);
      if (detail?.pet_uid === record.pet_uid) {
        setDetail(await fetchAdminPetDetail(record.pet_uid));
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
      if (detail?.pet_uid === petUID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadPets(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除宠物失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminPetSummary>>(
    () => [
      { title: '宠物UID', dataIndex: 'pet_uid', width: 100 },
      { title: '玩家ID', dataIndex: 'player_id', width: 90 },
      { title: '玩家名', dataIndex: 'player_name', width: 120 },
      { title: '宠物ID', dataIndex: 'pet_id', width: 90 },
      { title: '等级', dataIndex: 'level', width: 70 },
      {
        title: '品质',
        dataIndex: 'quality',
        width: 100,
        render: (value: number) => <Tag color={getPetQualityTagColor(value)}>{formatPetQualityLabel(value)}</Tag>,
      },
      { title: '生命', width: 100, render: (_value, record) => `${record.hp}/${record.hp_max}` },
      { title: '攻/防/速', width: 110, render: (_value, record) => `${record.atk}/${record.def}/${record.spd}` },
      {
        title: '出战',
        dataIndex: 'in_lineup',
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
        title: '更新时间',
        dataIndex: 'updated_at',
        width: 170,
        render: (value: string) => formatPetDateTime(value),
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
    [deletingID, lineupUpdatingUID],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Typography.Title level={4} style={{ marginTop: 0 }}>玩家宠物实例</Typography.Title>
        <Typography.Text type="secondary">
          跨玩家检索与维护 player_pet 实例；保存次要战斗属性时服务端会按 pet_combat_stat_cap 封顶表截断。
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
          <Form.Item label="宠物UID" name="pet_uid">
            <Input placeholder="精确匹配" allowClear style={{ width: 140 }} />
          </Form.Item>
          <Form.Item label="玩家ID" name="player_id">
            <Input placeholder="精确匹配" allowClear style={{ width: 140 }} />
          </Form.Item>
          <Form.Item label="宠物ID" name="pet_id">
            <Input placeholder="模板 pet_id" allowClear style={{ width: 140 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">查询</Button>
              <Button
                onClick={() => {
                  filterForm.resetFields();
                  setFilters({});
                  setPage(1);
                }}
              >
                重置
              </Button>
              <Button type="primary" onClick={() => setGrantModalOpen(true)}>新增宠物</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card>
        <Table
          rowKey="pet_uid"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1200 }}
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
            onClick: () => void handleViewDetail(record.pet_uid),
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      <Drawer
        title={detail ? `宠物详情 · ${detail.pet_uid}` : '宠物详情'}
        width={560}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
        extra={detail ? (
          <TableActionDropdown
            buttonType="default"
            loading={deletingID === detail.pet_uid}
            actions={[
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor(detail.pet_uid) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这个宠物吗？' },
                onClick: () => void handleDelete(detail.pet_uid),
              },
            ]}
          />
        ) : null}
      >
        {detailLoading ? (
          <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载宠物详情..." />
          </div>
        ) : detail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="宠物UID">{detail.pet_uid}</Descriptions.Item>
              <Descriptions.Item label="玩家ID">{detail.player_id}</Descriptions.Item>
              <Descriptions.Item label="玩家名">{detail.player_name}</Descriptions.Item>
              <Descriptions.Item label="宠物ID">{detail.pet_id}</Descriptions.Item>
              <Descriptions.Item label="等级">{detail.level}</Descriptions.Item>
              <Descriptions.Item label="经验">{detail.exp}</Descriptions.Item>
              <Descriptions.Item label="品质">
                <Tag color={getPetQualityTagColor(detail.quality)}>{formatPetQualityLabel(detail.quality)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="出战">
                <Switch
                  checked={detail.in_lineup}
                  loading={lineupUpdatingUID === detail.pet_uid}
                  checkedChildren="是"
                  unCheckedChildren="否"
                  onChange={(checked) => void handleToggleLineup(detail, checked)}
                />
              </Descriptions.Item>
              <Descriptions.Item label="生命">{`${detail.hp}/${detail.hp_max}`}</Descriptions.Item>
              <Descriptions.Item label="攻/防/速">{`${detail.atk}/${detail.def}/${detail.spd}`}</Descriptions.Item>
              <Descriptions.Item label="法力">{detail.mana}</Descriptions.Item>
              <Descriptions.Item label="技能" span={2}>
                <SkillReferenceText skillIds={detail.skill_ids} map={skillReferenceMap} emptyText="无" />
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{formatPetDateTime(detail.created_at)}</Descriptions.Item>
              <Descriptions.Item label="更新时间">{formatPetDateTime(detail.updated_at)}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="次要战斗属性">
              {ADMIN_PET_COMBAT_STAT_FIELDS.map((field) => (
                <Descriptions.Item key={field.key} label={field.label}>
                  {detail[field.key]}
                </Descriptions.Item>
              ))}
            </Descriptions>
          </Space>
        ) : null}
      </Drawer>

      <GrantPetFromTemplateModal
        open={grantModalOpen}
        onCancel={() => setGrantModalOpen(false)}
        onSuccess={() => void loadPets(filters, page, pageSize)}
      />

      <Modal
        title={editingRecord ? `编辑宠物 · ${editingRecord.pet_uid}` : '编辑宠物'}
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
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item label="宠物ID" name="pet_id" rules={[{ required: true, message: '请输入宠物ID' }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
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
              <Form.Item label="技能" name="skill_names_text" extra="填写系统技能名称，多个用英文逗号分隔">
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
      </Modal>
    </Space>
  );
}
