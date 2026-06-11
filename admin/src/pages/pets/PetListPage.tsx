import {
  Alert,
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
  Popconfirm,
  Row,
  Space,
  Spin,
  Statistic,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { createAdminPet, deleteAdminPet, fetchAdminPetDetail, fetchAdminPets, updateAdminPet } from '../../services/pet';
import type { AdminCreatePetPayload, AdminPetDetail, AdminPetListFilters, AdminPetSummary, AdminUpdatePetPayload } from '../../types/pet';

interface PetFormValues {
  player_id?: number;
  pet_id: number;
  level: number;
  exp: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  skill_ids_text: string;
}

export function PetListPage() {
  const [filterForm] = Form.useForm<AdminPetListFilters>();
  const [editorForm] = Form.useForm<PetFormValues>();
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
  const [editingRecord, setEditingRecord] = useState<AdminPetDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);

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

  async function handleOpenEditor(mode: 'create' | 'edit', petUID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultCreateValues());
      return;
    }
    if (!petUID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminPetDetail(petUID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: PetFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminPet(editingRecord.pet_uid, mapFormToUpdatePayload(values));
        message.success('宠物更新成功');
      } else {
        await createAdminPet(mapFormToCreatePayload(values));
        message.success('宠物创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadPets(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存宠物失败');
    } finally {
      setSaving(false);
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
      { title: '宠物UID', dataIndex: 'pet_uid', key: 'pet_uid', width: 110, fixed: 'left' },
      { title: '玩家ID', dataIndex: 'player_id', key: 'player_id', width: 110 },
      { title: '玩家名', dataIndex: 'player_name', key: 'player_name', width: 150 },
      { title: '宠物ID', dataIndex: 'pet_id', key: 'pet_id', width: 100 },
      { title: '等级', dataIndex: 'level', key: 'level', width: 90 },
      { title: '品质', dataIndex: 'quality', key: 'quality', width: 90 },
      { title: '生命', key: 'hp', width: 120, render: (_v, record) => `${record.hp}/${record.hp_max}` },
      { title: '攻/防/速', key: 'stats', width: 140, render: (_v, record) => `${record.atk}/${record.def}/${record.spd}` },
      { title: '法力', dataIndex: 'mana', key: 'mana', width: 90 },
      { title: '出战', dataIndex: 'in_lineup', key: 'in_lineup', width: 90, render: (value: boolean) => <Tag color={value ? 'green' : 'default'}>{value ? '是' : '否'}</Tag> },
      {
        title: '操作', key: 'actions', width: 220, fixed: 'right', render: (_v, record) => (
          <Space size="small">
            <Button type="link" onClick={() => void handleViewDetail(record.pet_uid)}>查看</Button>
            <Button type="link" onClick={() => void handleOpenEditor('edit', record.pet_uid)}>编辑</Button>
            <Popconfirm title="确认删除这个宠物吗？" onConfirm={() => void handleDelete(record.pet_uid)} okText="确认删除" cancelText="取消">
              <Button type="link" danger loading={deletingID === record.pet_uid}>删除</Button>
            </Popconfirm>
          </Space>
        )
      }
    ],
    [deletingID],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="info" showIcon message="宠物管理已按 Ant Design CRUD 模式接入真实服务端接口。" />
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}><Card><Statistic title="当前页宠物数" value={rows.length} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="出战宠物" value={rows.filter((item) => item.in_lineup).length} valueStyle={{ color: '#2f7d4a' }} /></Card></Col>
        <Col xs={24} md={8}><Card><Statistic title="总记录数" value={total} /></Card></Col>
      </Row>
      <Card title="宠物筛选" extra={<Button type="primary" onClick={() => void handleOpenEditor('create')}>新增宠物</Button>}>
        <Form form={filterForm} layout="vertical" onFinish={(values) => { setPage(1); setFilters(values); }}>
          <Row gutter={16}>
            <Col xs={24} md={8}><Form.Item label="宠物UID" name="pet_uid"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="玩家ID" name="player_id"><Input allowClear /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="宠物ID" name="pet_id"><Input allowClear /></Form.Item></Col>
          </Row>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
            <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
          </Space>
        </Form>
      </Card>
      <Card title="宠物列表" extra={<Typography.Text type="secondary">支持创建、编辑、删除与详情查看。</Typography.Text>}>
        <Table columns={columns} dataSource={rows} rowKey="pet_uid" loading={loading} locale={{ emptyText: <Empty description="当前筛选条件下没有宠物数据" /> }} scroll={{ x: 1400 }} pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (value) => `共 ${value} 只宠物`, onChange: (nextPage, nextPageSize) => { setPage(nextPage); setPageSize(nextPageSize); } }} />
      </Card>
      <Drawer title={detail ? `宠物详情 · ${detail.pet_uid}` : '宠物详情'} width={560} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading ? <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}><Spin tip="正在加载宠物详情..." /></div> : detail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="宠物UID">{detail.pet_uid}</Descriptions.Item>
            <Descriptions.Item label="玩家ID">{detail.player_id}</Descriptions.Item>
            <Descriptions.Item label="玩家名">{detail.player_name}</Descriptions.Item>
            <Descriptions.Item label="宠物ID">{detail.pet_id}</Descriptions.Item>
            <Descriptions.Item label="等级">{detail.level}</Descriptions.Item>
            <Descriptions.Item label="经验">{detail.exp}</Descriptions.Item>
            <Descriptions.Item label="品质">{detail.quality}</Descriptions.Item>
            <Descriptions.Item label="出战"><Switch checked={detail.in_lineup} disabled /></Descriptions.Item>
            <Descriptions.Item label="生命">{`${detail.hp}/${detail.hp_max}`}</Descriptions.Item>
            <Descriptions.Item label="攻/防/速">{`${detail.atk}/${detail.def}/${detail.spd}`}</Descriptions.Item>
            <Descriptions.Item label="法力">{detail.mana}</Descriptions.Item>
            <Descriptions.Item label="技能ID" span={2}>{detail.skill_ids.length > 0 ? detail.skill_ids.join(', ') : '无'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDateTime(detail.created_at)}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{formatDateTime(detail.updated_at)}</Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>
      <Modal title={editingRecord ? `编辑宠物 · ${editingRecord.pet_uid}` : '新增宠物'} open={editorOpen} onCancel={() => { setEditorOpen(false); setEditingRecord(null); editorForm.resetFields(); }} onOk={() => editorForm.submit()} confirmLoading={saving} destroyOnClose width={700} okText={editingRecord ? '保存修改' : '创建宠物'} cancelText="取消">
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            {!editingRecord ? <Col xs={24} md={12}><Form.Item label="所属玩家ID" name="player_id" rules={[{ required: true, message: '请输入所属玩家ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col> : null}
            <Col xs={24} md={12}><Form.Item label="宠物ID" name="pet_id" rules={[{ required: true, message: '请输入宠物ID' }]}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="等级" name="level"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="经验" name="exp"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="品质" name="quality"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="法力" name="mana"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="生命" name="hp"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="生命上限" name="hp_max"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="攻击" name="atk"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="防御" name="def"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="速度" name="spd"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}><Form.Item label="技能ID 列表" name="skill_ids_text" extra="使用英文逗号分隔，例如 1001,1002"><Input placeholder="1001,1002" /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultCreateValues(): PetFormValues {
  return { player_id: 10001, pet_id: 101, level: 1, exp: 0, quality: 1, hp: 10, hp_max: 10, atk: 5, def: 5, spd: 5, mana: 0, skill_ids_text: '1001' };
}

function mapDetailToForm(detail: AdminPetDetail): PetFormValues {
  return { pet_id: detail.pet_id, level: detail.level, exp: detail.exp, quality: detail.quality, hp: detail.hp, hp_max: detail.hp_max, atk: detail.atk, def: detail.def, spd: detail.spd, mana: detail.mana, skill_ids_text: detail.skill_ids.join(',') };
}

function parseSkillIDs(value: string): number[] {
  return value.split(',').map((item) => Number(item.trim())).filter((item) => Number.isFinite(item) && item > 0);
}

function mapFormToCreatePayload(values: PetFormValues): AdminCreatePetPayload {
  return { player_id: values.player_id ?? 0, pet_id: values.pet_id, level: values.level, exp: values.exp, quality: values.quality, hp: values.hp, hp_max: values.hp_max, atk: values.atk, def: values.def, spd: values.spd, mana: values.mana, skill_ids: parseSkillIDs(values.skill_ids_text) };
}

function mapFormToUpdatePayload(values: PetFormValues): AdminUpdatePetPayload {
  return { pet_id: values.pet_id, level: values.level, exp: values.exp, quality: values.quality, hp: values.hp, hp_max: values.hp_max, atk: values.atk, def: values.def, spd: values.spd, mana: values.mana, skill_ids: parseSkillIDs(values.skill_ids_text) };
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', { hour12: false });
}
