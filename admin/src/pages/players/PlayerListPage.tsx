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
  Spin,
  Table,
  Tag,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { SkillReferenceText } from '../../components/SkillReferenceText';
import { useSkillReferenceMap } from '../../hooks/useSkillReferenceMap';
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
  AdminPlayerEquippedItem,
  AdminPlayerListFilters,
  AdminPlayerSummary,
  AdminUpdatePlayerPayload,
} from '../../types/player';
import { PlayerPetSection } from './PlayerPetSection';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import { PlayerBagSection } from './PlayerBagSection';
import { PlayerWalletSection } from './PlayerWalletSection';
import { formatDisplayLabel, PLAYER_STATUS_LABELS } from '../../utils/displayLabels';
import { formatDateTime } from '../../utils/formatDateTime';
import { formatSkillReferenceInput, parseSkillReferenceInput, type SkillReferenceMap } from '../../utils/skillReference';
import { FIXED_FORM_MODAL_STYLES, FIXED_FORM_MODAL_TOP } from '../../utils/modalLayout';

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
  vigor: number;
  vigor_max: number;
  spirit: number;
  spirit_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  status: number;
  skill_names_text: string;
  skin_id: string;
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

const PLAYER_EDITOR_MODAL_WIDTH = 860;

// 玩家管理页按 ant-design-skill 的 CRUD 模式重写：筛选表格 + 详情抽屉 + 新增/编辑弹窗 + 删除确认。
export function PlayerListPage() {
  const { map: skillReferenceMap } = useSkillReferenceMap();
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
      editorForm.setFieldsValue(mapDetailToForm(result, skillReferenceMap));
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
        await updateAdminPlayer(editingRecord.player_id, mapFormToUpdatePayload(values, skillReferenceMap));
        message.success('玩家更新成功');
      } else {
        const created = await createAdminPlayer(mapFormToCreatePayload(values, skillReferenceMap));
        message.success('玩家创建成功');
        setEditingRecord(created);
        editorForm.setFieldsValue(mapDetailToForm(created, skillReferenceMap));
      }
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
        render: (value: string) => (
          <Tag color={statusColor(value)}>{formatDisplayLabel(PLAYER_STATUS_LABELS, value)}</Tag>
        ),
      },
      { title: '场景', dataIndex: 'scene_id', key: 'scene_id', width: 90 },
      {
        title: '生命 / 活力 / 精力',
        key: 'stats',
        width: 240,
        render: (_value, record) => `${record.hp}/${record.hp_max} · ${record.vigor}/${record.vigor_max} · ${record.spirit}/${record.spirit_max}`,
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
        width: 100,
        fixed: 'right',
        render: (_value, record) => (
          <TableActionDropdown
            loading={deletingID === record.player_id}
            actions={[
              { key: 'view', label: '查看', onClick: () => void handleViewDetail(record.player_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.player_id) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: {
                  title: '确认删除这个玩家吗？',
                  description: '删除会把账号与人物都软删除，不会物理清库。',
                  okText: '确认删除',
                  cancelText: '取消',
                },
                onClick: () => void handleDelete(record.player_id),
              },
            ]}
          />
        ),
      },
    ],
    [deletingID],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title="玩家列表"
        extra={(
          <Form<AdminPlayerListFilters>
            form={filterForm}
            layout="inline"
            onFinish={(values) => {
              setPage(1);
              setFilters(values);
            }}
          >
            <Form.Item name="player_id" label="玩家ID">
              <Input allowClear placeholder="玩家ID" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="name" label="昵称">
              <Input allowClear placeholder="昵称" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select options={statusOptions} style={{ width: 100 }} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
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
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增玩家</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
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
        width={760}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        destroyOnClose
        extra={
          detail ? (
            <TableActionDropdown
              buttonType="default"
              loading={deletingID === detail.player_id}
              actions={[
                { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', detail.player_id) },
                {
                  key: 'delete',
                  label: '删除',
                  danger: true,
                  confirm: {
                    title: '确认删除这个玩家吗？',
                    description: '删除会把账号与人物都软删除。',
                    okText: '确认删除',
                    cancelText: '取消',
                  },
                  onClick: () => void handleDelete(detail.player_id),
                },
              ]}
            />
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
                <Tag color={statusColor(detail.status_text)}>
                  {formatDisplayLabel(PLAYER_STATUS_LABELS, detail.status_text)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="昵称">{detail.name}</Descriptions.Item>
              <Descriptions.Item label="当前形象ID">{detail.skin_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="等级">{detail.level}</Descriptions.Item>
              <Descriptions.Item label="经验">{detail.exp}</Descriptions.Item>
              <Descriptions.Item label="自由属性点">{detail.free_attr_points ?? 0}</Descriptions.Item>
              <Descriptions.Item label="力量">{detail.strength ?? 0}</Descriptions.Item>
              <Descriptions.Item label="体质">{detail.vitality ?? 0}</Descriptions.Item>
              <Descriptions.Item label="敏捷">{detail.agility ?? 0}</Descriptions.Item>
              <Descriptions.Item label="灵力">{detail.mind ?? 0}</Descriptions.Item>
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
              <Descriptions.Item label="活力">{`${detail.vigor}/${detail.vigor_max}`}</Descriptions.Item>
              <Descriptions.Item label="精力">{`${detail.spirit}/${detail.spirit_max}`}</Descriptions.Item>
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
              <Descriptions.Item label="技能" span={2}>
                <SkillReferenceText skillIds={detail.skill_ids} map={skillReferenceMap} emptyText="无" />
              </Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="人物装备">
              {(detail.equipped_items ?? []).map((item: AdminPlayerEquippedItem) => (
                <Descriptions.Item key={item.equip_slot} label={item.equip_slot_label}>
                  {formatPlayerEquippedItem(item)}
                </Descriptions.Item>
              ))}
            </Descriptions>
            <PlayerWalletSection playerId={detail.player_id} playerName={detail.name} />
            <PlayerPetSection playerId={detail.player_id} playerName={detail.name} />
            <PlayerBagSection playerId={detail.player_id} playerName={detail.name} />
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
        width={PLAYER_EDITOR_MODAL_WIDTH}
        style={{ top: FIXED_FORM_MODAL_TOP }}
        styles={FIXED_FORM_MODAL_STYLES}
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
            <Col xs={12} md={6}><Form.Item label="活力" name="vigor"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="活力上限" name="vigor_max"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="精力" name="spirit"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="精力上限" name="spirit_max"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="攻击" name="atk"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="防御" name="def"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="速度" name="spd"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="法力" name="mana"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={12}>
              <Form.Item
                label="当前形象ID"
                name="skin_id"
                extra="对应客户端 unit_skins/{skin_id}.tres，例如 初始形象男_001"
              >
                <Input placeholder="初始形象男_001" />
              </Form.Item>
            </Col>
            <Col span={24}>
              <Form.Item label="技能" name="skill_names_text" extra="填写系统技能名称，多个用英文逗号分隔，例如 裂空斩,普通攻击">
                <Input placeholder="裂空斩,普通攻击" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
        {editingRecord ? (
          <div style={{ marginTop: 24 }}>
            <PlayerWalletSection playerId={editingRecord.player_id} playerName={editingRecord.name} />
            <div style={{ marginTop: 16 }}>
              <PlayerPetSection playerId={editingRecord.player_id} playerName={editingRecord.name} />
            </div>
            <div style={{ marginTop: 16 }}>
              <PlayerBagSection playerId={editingRecord.player_id} playerName={editingRecord.name} />
            </div>
          </div>
        ) : null}
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
    hp: 148,
    hp_max: 148,
    vigor: 100,
    vigor_max: 100,
    spirit: 40,
    spirit_max: 40,
    atk: 42,
    def: 15,
    spd: 11,
    mana: 30,
    status: 1,
    skill_names_text: '裂空斩,普通攻击',
    skin_id: '初始形象男_001',
  };
}

function mapDetailToForm(detail: AdminPlayerDetail, skillReferenceMap: SkillReferenceMap): PlayerFormValues {
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
    vigor: detail.vigor,
    vigor_max: detail.vigor_max,
    spirit: detail.spirit,
    spirit_max: detail.spirit_max,
    atk: detail.atk,
    def: detail.def,
    spd: detail.spd,
    mana: detail.mana,
    status: detail.status,
    skill_names_text: formatSkillReferenceInput(detail.skill_ids, skillReferenceMap),
    skin_id: detail.skin_id,
  };
}

function mapFormToCreatePayload(values: PlayerFormValues, skillReferenceMap: SkillReferenceMap): AdminCreatePlayerPayload {
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
    vigor: values.vigor,
    vigor_max: values.vigor_max,
    spirit: values.spirit,
    spirit_max: values.spirit_max,
    atk: values.atk,
    def: values.def,
    spd: values.spd,
    mana: values.mana,
    status: values.status,
    skill_ids: parseSkillReferenceInput(values.skill_names_text, skillReferenceMap),
    skin_id: values.skin_id?.trim() ?? '',
  };
}

function mapFormToUpdatePayload(values: PlayerFormValues, skillReferenceMap: SkillReferenceMap): AdminUpdatePlayerPayload {
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
    vigor: values.vigor,
    vigor_max: values.vigor_max,
    spirit: values.spirit,
    spirit_max: values.spirit_max,
    atk: values.atk,
    def: values.def,
    spd: values.spd,
    mana: values.mana,
    status: values.status,
    skill_ids: parseSkillReferenceInput(values.skill_names_text, skillReferenceMap),
    skin_id: values.skin_id?.trim() ?? '',
  };
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

function formatPlayerEquippedItem(item: AdminPlayerEquippedItem): string {
  if (item.is_empty || !item.item_name) {
    return '空';
  }
  if (item.enhance_level > 0) {
    return `${item.item_name} +${item.enhance_level}`;
  }
  return item.item_name;
}
