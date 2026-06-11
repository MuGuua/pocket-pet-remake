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
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import {
  createAdminPlayer,
  deleteAdminPlayer,
  fetchAdminPlayerDetail,
  fetchAdminPlayers,
  updateAdminPlayer,
} from '../../services/player';
import type {
  AdminCreatePlayerPayload,
  AdminPlayerDetail,
  AdminPlayerListFilters,
  AdminPlayerSummary,
  AdminUpdatePlayerPayload,
} from '../../types/player';

interface PlayerFormValues {
  account_name?: string;
  password?: string;
  name: string;
  level: number;
  exp?: number;
  gold: number;
  scene_id: number;
  pos_x: number;
  pos_y: number;
  hp: number;
  hp_max: number;
  energy: number;
  energy_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  status: number;
  skill_ids_text: string;
}

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '正常', value: '1' },
  { label: '封禁', value: '2' },
  { label: '已删除', value: '0' },
];

const editableStatusOptions = [
  { label: '正常', value: 1 },
  { label: '封禁', value: 2 },
  { label: '已删除', value: 0 },
];

// 玩家管理页按 ant-design-skill 的 CRUD 模式重写：筛选表格 + 详情抽屉 + 新增/编辑弹窗 + 删除确认。
export function PlayerListPage() {
  const [filterForm] = Form.useForm<AdminPlayerListFilters>();
  const [editorForm] = Form.useForm<PlayerFormValues>();
  const [filters, setFilters] = useState<AdminPlayerListFilters>({ status: '1' });
  const [rows, setRows] = useState<AdminPlayerSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<AdminPlayerDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminPlayerDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);

  useEffect(() => {
    filterForm.setFieldsValue({ status: '1' });
  }, [filterForm]);

  useEffect(() => {
    void loadPlayers(filters, page, pageSize);
  }, [filters, page, pageSize]);

  async function loadPlayers(nextFilters: AdminPlayerListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminPlayers({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家列表失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(playerID: number) {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      const result = await fetchAdminPlayerDetail(playerID);
      setDetail(result);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', playerID?: number) {
    setEditorOpen(true);
    setSaving(false);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.resetFields();
      editorForm.setFieldsValue(defaultCreateValues());
      return;
    }
    if (!playerID) {
      return;
    }
    setDetailLoading(true);
    try {
      const result = await fetchAdminPlayerDetail(playerID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToForm(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmitEditor(values: PlayerFormValues) {
    setSaving(true);
    try {
      if (editingRecord) {
        await updateAdminPlayer(editingRecord.player_id, mapFormToUpdatePayload(values));
        message.success('玩家更新成功');
      } else {
        await createAdminPlayer(mapFormToCreatePayload(values));
        message.success('玩家创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadPlayers(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存玩家失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(playerID: number) {
    setDeletingID(playerID);
    try {
      await deleteAdminPlayer(playerID);
      message.success('玩家已删除');
      if (detail?.player_id === playerID) {
        setDetailOpen(false);
        setDetail(null);
      }
      await loadPlayers(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除玩家失败');
    } finally {
      setDeletingID(null);
    }
  }

  const columns = useMemo<ColumnsType<AdminPlayerSummary>>(
    () => [
      { title: '玩家ID', dataIndex: 'player_id', key: 'player_id', width: 110, fixed: 'left' },
      { title: '账号', dataIndex: 'account_name', key: 'account_name', width: 160 },
      { title: '昵称', dataIndex: 'name', key: 'name', width: 160 },
      { title: '等级', dataIndex: 'level', key: 'level', width: 90 },
      { title: '金币', dataIndex: 'gold', key: 'gold', width: 120 },
      {
        title: '状态',
        dataIndex: 'status_text',
        key: 'status_text',
        width: 120,
        render: (value: string) => <Tag color={statusColor(value)}>{value}</Tag>,
      },
      { title: '场景', dataIndex: 'scene_id', key: 'scene_id', width: 90 },
      {
        title: '生命 / 体力',
        key: 'stats',
        width: 180,
        render: (_value, record) => `${record.hp}/${record.hp_max} · ${record.energy}/${record.energy_max}`,
      },
      {
        title: '最近登录',
        dataIndex: 'last_login_at',
        key: 'last_login_at',
        width: 180,
        render: (value: string | null) => formatDateTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        width: 220,
        fixed: 'right',
        render: (_value, record) => (
          <Space size="small">
            <Button type="link" onClick={() => void handleViewDetail(record.player_id)}>
              查看
            </Button>
            <Button type="link" onClick={() => void handleOpenEditor('edit', record.player_id)}>
              编辑
            </Button>
            <Popconfirm
              title="确认删除这个玩家吗？"
              description="删除会把账号与人物都软删除，不会物理清库。"
              okText="确认删除"
              cancelText="取消"
              onConfirm={() => void handleDelete(record.player_id)}
            >
              <Button type="link" danger loading={deletingID === record.player_id}>
                删除
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [deletingID],
  );

  const activeCount = rows.filter((item) => item.status === 1).length;
  const bannedCount = rows.filter((item) => item.status === 2).length;
  const deletedCount = rows.filter((item) => item.status === 0).length;

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        message="当前玩家管理已支持完整 CRUD，所有增删改查都直接走服务端 `/api/admin/players` 权威接口。"
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} md={8} xl={6}>
          <Card>
            <Statistic title="当前页玩家数" value={rows.length} />
          </Card>
        </Col>
        <Col xs={24} md={8} xl={6}>
          <Card>
            <Statistic title="正常玩家" value={activeCount} valueStyle={{ color: '#2f7d4a' }} />
          </Card>
        </Col>
        <Col xs={24} md={8} xl={6}>
          <Card>
            <Statistic title="封禁玩家" value={bannedCount} valueStyle={{ color: '#d48806' }} />
          </Card>
        </Col>
        <Col xs={24} md={8} xl={6}>
          <Card>
            <Statistic title="已删除" value={deletedCount} valueStyle={{ color: '#cf1322' }} />
          </Card>
        </Col>
      </Row>

      <Card
        title="玩家筛选"
        extra={
          <Button type="primary" onClick={() => void handleOpenEditor('create')}>
            新增玩家
          </Button>
        }
      >
        <Form<AdminPlayerListFilters>
          form={filterForm}
          layout="vertical"
          onFinish={(values) => {
            setPage(1);
            setFilters(values);
          }}
        >
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item label="玩家ID" name="player_id">
                <Input placeholder="按玩家ID精确查询" allowClear />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="昵称" name="name">
                <Input placeholder="按昵称模糊查询" allowClear />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item label="状态" name="status">
                <Select options={statusOptions} />
              </Form.Item>
            </Col>
          </Row>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading}>
              查询
            </Button>
            <Button
              onClick={() => {
                const nextValues = { status: '1' };
                filterForm.resetFields();
                filterForm.setFieldsValue(nextValues);
                setPage(1);
                setFilters(nextValues);
              }}
            >
              重置
            </Button>
          </Space>
        </Form>
      </Card>

      <Card title="玩家列表" extra={<Typography.Text type="secondary">默认采用服务端分页，避免前端伪造列表状态。</Typography.Text>}>
        <Table<AdminPlayerSummary>
          columns={columns}
          dataSource={rows}
          rowKey="player_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有玩家数据" /> }}
          scroll={{ x: 1460 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 名玩家`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </Card>

      <Drawer
        title={detail ? `玩家详情 · ${detail.name}` : '玩家详情'}
        width={600}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
        extra={
          detail ? (
            <Space>
              <Button onClick={() => void handleOpenEditor('edit', detail.player_id)}>编辑</Button>
              <Popconfirm
                title="确认删除这个玩家吗？"
                description="删除会把账号与人物都软删除。"
                okText="确认删除"
                cancelText="取消"
                onConfirm={() => void handleDelete(detail.player_id)}
              >
                <Button danger loading={deletingID === detail.player_id}>
                  删除
                </Button>
              </Popconfirm>
            </Space>
          ) : null
        }
      >
        {detailLoading ? (
          <div style={{ minHeight: 260, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载玩家详情..." />
          </div>
        ) : detail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small" title="基础信息">
              <Descriptions.Item label="玩家ID">{detail.player_id}</Descriptions.Item>
              <Descriptions.Item label="账号ID">{detail.account_id}</Descriptions.Item>
              <Descriptions.Item label="账号名">{detail.account_name}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColor(detail.status_text)}>{detail.status_text}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="昵称">{detail.name}</Descriptions.Item>
              <Descriptions.Item label="等级">{detail.level}</Descriptions.Item>
              <Descriptions.Item label="经验">{detail.exp}</Descriptions.Item>
              <Descriptions.Item label="金币">{detail.gold}</Descriptions.Item>
              <Descriptions.Item label="场景ID">{detail.scene_id}</Descriptions.Item>
              <Descriptions.Item label="坐标">{`${detail.pos_x}, ${detail.pos_y}`}</Descriptions.Item>
              <Descriptions.Item label="最近登录">{formatDateTime(detail.last_login_at)}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{formatDateTime(detail.created_at)}</Descriptions.Item>
              <Descriptions.Item label="更新时间" span={2}>
                {formatDateTime(detail.updated_at)}
              </Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="战斗属性">
              <Descriptions.Item label="生命">{`${detail.hp}/${detail.hp_max}`}</Descriptions.Item>
              <Descriptions.Item label="体力">{`${detail.energy}/${detail.energy_max}`}</Descriptions.Item>
              <Descriptions.Item label="攻击">{detail.atk}</Descriptions.Item>
              <Descriptions.Item label="防御">{detail.def}</Descriptions.Item>
              <Descriptions.Item label="速度">{detail.spd}</Descriptions.Item>
              <Descriptions.Item label="法力">{detail.mana}</Descriptions.Item>
              <Descriptions.Item label="命中">{detail.hit_pct}</Descriptions.Item>
              <Descriptions.Item label="闪避">{detail.dodge_pct}</Descriptions.Item>
              <Descriptions.Item label="暴击">{detail.crit_rate_pct}</Descriptions.Item>
              <Descriptions.Item label="暴伤">{detail.crit_dmg_pct}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="抗性与技能">
              <Descriptions.Item label="物抗">{detail.physical_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="技抗">{detail.skill_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="混乱抗性">{detail.confusion_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="睡眠抗性">{detail.sleep_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="麻痹抗性">{detail.paralysis_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="封印抗性">{detail.seal_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="诅咒抗性">{detail.curse_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="暴击抗性">{detail.crit_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="暴伤减免">{detail.crit_dmg_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="人物减伤">{detail.character_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="宠物减伤">{detail.pet_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="佣兵减伤">{detail.mercenary_resist_pct}</Descriptions.Item>
              <Descriptions.Item label="护盾减免">{detail.generic_shield_pct}</Descriptions.Item>
              <Descriptions.Item label="技能ID" span={2}>
                {detail.skill_ids.length > 0 ? detail.skill_ids.join(', ') : '无'}
              </Descriptions.Item>
            </Descriptions>
          </Space>
        ) : null}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑玩家 · ${editingRecord.name}` : '新增玩家'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingRecord(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={760}
        okText={editingRecord ? '保存修改' : '创建玩家'}
        cancelText="取消"
      >
        <Form<PlayerFormValues> form={editorForm} layout="vertical" onFinish={(values) => void handleSubmitEditor(values)}>
          <Row gutter={16}>
            {!editingRecord ? (
              <>
                <Col xs={24} md={12}>
                  <Form.Item label="账号名" name="account_name" rules={[{ required: true, message: '请输入账号名' }]}> 
                    <Input placeholder="用于登录的账号名" />
                  </Form.Item>
                </Col>
                <Col xs={24} md={12}>
                  <Form.Item label="初始密码" name="password" rules={[{ required: true, message: '请输入初始密码' }]}> 
                    <Input.Password placeholder="用于登录的初始密码" />
                  </Form.Item>
                </Col>
              </>
            ) : null}
            <Col xs={24} md={12}>
              <Form.Item label="昵称" name="name" rules={[{ required: true, message: '请输入玩家昵称' }]}> 
                <Input placeholder="请输入玩家昵称" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item label="状态" name="status" rules={[{ required: true, message: '请选择状态' }]}> 
                <Select options={editableStatusOptions} />
              </Form.Item>
            </Col>
            <Col xs={12} md={6}><Form.Item label="等级" name="level"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="经验" name="exp"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="金币" name="gold"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="场景ID" name="scene_id"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="X 坐标" name="pos_x"><InputNumber style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="Y 坐标" name="pos_y"><InputNumber style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="生命" name="hp"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="生命上限" name="hp_max"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="体力" name="energy"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="体力上限" name="energy_max"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="攻击" name="atk"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="防御" name="def"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="速度" name="spd"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="法力" name="mana"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={24}>
              <Form.Item label="技能ID 列表" name="skill_ids_text" extra="使用英文逗号分隔，例如 1101,1001">
                <Input placeholder="1101,1001" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </Space>
  );
}

function defaultCreateValues(): PlayerFormValues {
  return {
    account_name: '',
    password: '',
    name: '',
    level: 1,
    exp: 0,
    gold: 0,
    scene_id: 1,
    pos_x: 0,
    pos_y: 0,
    hp: 100,
    hp_max: 100,
    energy: 100,
    energy_max: 100,
    atk: 24,
    def: 12,
    spd: 18,
    mana: 20,
    status: 1,
    skill_ids_text: '1101,1001',
  };
}

function mapDetailToForm(detail: AdminPlayerDetail): PlayerFormValues {
  return {
    name: detail.name,
    level: detail.level,
    exp: detail.exp,
    gold: detail.gold,
    scene_id: detail.scene_id,
    pos_x: detail.pos_x,
    pos_y: detail.pos_y,
    hp: detail.hp,
    hp_max: detail.hp_max,
    energy: detail.energy,
    energy_max: detail.energy_max,
    atk: detail.atk,
    def: detail.def,
    spd: detail.spd,
    mana: detail.mana,
    status: detail.status,
    skill_ids_text: detail.skill_ids.join(','),
  };
}

function mapFormToCreatePayload(values: PlayerFormValues): AdminCreatePlayerPayload {
  return {
    account_name: values.account_name?.trim() ?? '',
    password: values.password?.trim() ?? '',
    name: values.name.trim(),
    level: values.level,
    gold: values.gold,
    scene_id: values.scene_id,
    pos_x: values.pos_x,
    pos_y: values.pos_y,
    hp: values.hp,
    hp_max: values.hp_max,
    energy: values.energy,
    energy_max: values.energy_max,
    atk: values.atk,
    def: values.def,
    spd: values.spd,
    mana: values.mana,
    status: values.status,
    skill_ids: parseSkillIDs(values.skill_ids_text),
  };
}

function mapFormToUpdatePayload(values: PlayerFormValues): AdminUpdatePlayerPayload {
  return {
    name: values.name.trim(),
    level: values.level,
    exp: values.exp ?? 0,
    gold: values.gold,
    scene_id: values.scene_id,
    pos_x: values.pos_x,
    pos_y: values.pos_y,
    hp: values.hp,
    hp_max: values.hp_max,
    energy: values.energy,
    energy_max: values.energy_max,
    atk: values.atk,
    def: values.def,
    spd: values.spd,
    mana: values.mana,
    status: values.status,
    skill_ids: parseSkillIDs(values.skill_ids_text),
  };
}

function parseSkillIDs(value: string): number[] {
  return value
    .split(',')
    .map((item) => Number(item.trim()))
    .filter((item) => Number.isFinite(item) && item > 0);
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

function statusColor(statusText: string): string {
  switch (statusText) {
    case 'NORMAL':
      return 'green';
    case 'BANNED':
      return 'orange';
    case 'DELETED':
      return 'red';
    default:
      return 'default';
  }
}
