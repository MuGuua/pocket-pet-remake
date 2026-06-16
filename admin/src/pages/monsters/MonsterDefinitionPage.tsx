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
import { SkillReferenceText } from '../../components/SkillReferenceText';
import { useSkillReferenceMap } from '../../hooks/useSkillReferenceMap';
import { TableActionDropdown } from '../../components/TableActionDropdown';
import {
  createAdminMonsterDefinition,
  deleteAdminMonsterDefinition,
  fetchAdminMonsterBattleRewards,
  fetchAdminMonsterDefinitionDetail,
  fetchAdminMonsterDefinitions,
  updateAdminMonsterBattleRewards,
  updateAdminMonsterDefinition,
} from '../../services/monsterDefinition';
import { fetchAdminPetDefinitions } from '../../services/petDefinition';
import type {
  AdminMonsterBattleRewardEntry,
  AdminMonsterDefinitionDetail,
  AdminMonsterDefinitionListFilters,
  AdminMonsterDefinitionSummary,
  AdminUpsertMonsterDefinitionPayload,
} from '../../types/monsterDefinition';
import type { AdminPetDefinitionSummary } from '../../types/petDefinition';
import { formatSkillReferenceInput, parseSkillReferenceInput, type SkillReferenceMap } from '../../utils/skillReference';

interface MonsterDefinitionFormValues extends AdminUpsertMonsterDefinitionPayload {
  skill_names_text?: string;
  capture_item_ids_text?: string;
}

interface MonsterBattleRewardFormValues {
  rewards: AdminMonsterBattleRewardEntry[];
}

// 系统怪物模板页维护 PVE 战斗怪物白名单；遭遇配置通过 monster_id 引用这里的模板。
export function MonsterDefinitionPage() {
  const { map: skillReferenceMap } = useSkillReferenceMap();
  const [filterForm] = Form.useForm<AdminMonsterDefinitionListFilters>();
  const [editorForm] = Form.useForm<MonsterDefinitionFormValues>();
  const [filters, setFilters] = useState<AdminMonsterDefinitionListFilters>({});
  const [rows, setRows] = useState<AdminMonsterDefinitionSummary[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<AdminMonsterDefinitionDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminMonsterDefinitionDetail | null>(null);
  const [saving, setSaving] = useState(false);
  const [wildCapturePetOptions, setWildCapturePetOptions] = useState<AdminPetDefinitionSummary[]>([]);
  const [rewardEditorOpen, setRewardEditorOpen] = useState(false);
  const [rewardMonsterID, setRewardMonsterID] = useState<number | null>(null);
  const [rewardMonsterName, setRewardMonsterName] = useState('');
  const [rewardSaving, setRewardSaving] = useState(false);
  const [rewardLoading, setRewardLoading] = useState(false);
  const [rewardForm] = Form.useForm<MonsterBattleRewardFormValues>();
  const isCapturable = Form.useWatch('is_capturable', editorForm);

  useEffect(() => {
    void loadDefinitions(filters, page, pageSize);
  }, [filters, page, pageSize]);

  useEffect(() => {
    void loadWildCapturePetOptions();
  }, []);

  async function loadWildCapturePetOptions() {
    try {
      const result = await fetchAdminPetDefinitions({ filters: {}, page: 1, pageSize: 100 });
      const options = result.items.filter((item) => item.acquire_method === 'wild_capture' || item.acquire_method.includes('野外捕捉'));
      setWildCapturePetOptions(options);
    } catch {
      setWildCapturePetOptions([]);
    }
  }

  async function loadDefinitions(nextFilters: AdminMonsterDefinitionListFilters, nextPage: number, nextPageSize: number) {
    setLoading(true);
    try {
      const result = await fetchAdminMonsterDefinitions({ filters: nextFilters, page: nextPage, pageSize: nextPageSize });
      setRows(result.items);
      setTotal(result.total);
      setPage(result.page);
      setPageSize(result.page_size);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统怪物模板失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewDetail(monsterID: number) {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchAdminMonsterDefinitionDetail(monsterID));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统怪物详情失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleOpenEditor(mode: 'create' | 'edit', monsterID?: number) {
    setEditorOpen(true);
    if (mode === 'create') {
      setEditingRecord(null);
      editorForm.setFieldsValue(defaultMonsterDefinitionValues());
      return;
    }
    if (!monsterID) return;
    setDetailLoading(true);
    try {
      const result = await fetchAdminMonsterDefinitionDetail(monsterID);
      setEditingRecord(result);
      editorForm.setFieldsValue(mapDetailToFormValues(result, skillReferenceMap));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载系统怪物编辑数据失败');
      setEditorOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  async function handleSubmit(values: MonsterDefinitionFormValues) {
    setSaving(true);
    try {
      const payload = buildPayloadFromForm(values, skillReferenceMap);
      if (editingRecord) {
        await updateAdminMonsterDefinition(editingRecord.monster_id, payload);
        message.success('系统怪物模板更新成功');
      } else {
        await createAdminMonsterDefinition(payload);
        message.success('系统怪物模板创建成功');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      await loadDefinitions(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存系统怪物模板失败');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(monsterID: number) {
    try {
      await deleteAdminMonsterDefinition(monsterID);
      message.success('系统怪物模板已删除');
      if (detail?.monster_id === monsterID) {
        setDetail(null);
        setDetailOpen(false);
      }
      await loadDefinitions(filters, page, pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除系统怪物模板失败');
    }
  }

  async function handleOpenBattleRewards(record: AdminMonsterDefinitionSummary) {
    setRewardEditorOpen(true);
    setRewardMonsterID(record.monster_id);
    setRewardMonsterName(record.monster_name);
    rewardForm.resetFields();
    setRewardLoading(true);
    try {
      const items = await fetchAdminMonsterBattleRewards(record.monster_id);
      rewardForm.setFieldsValue({ rewards: items.length > 0 ? items : [defaultMonsterBattleRewardRow()] });
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载战斗奖励配置失败');
      setRewardEditorOpen(false);
    } finally {
      setRewardLoading(false);
    }
  }

  async function handleSubmitBattleRewards(values: MonsterBattleRewardFormValues) {
    if (rewardMonsterID == null) {
      return;
    }
    setRewardSaving(true);
    try {
      const rewards = (values.rewards ?? []).map((item, index) => mapMonsterBattleRewardPayload(item, index));
      await updateAdminMonsterBattleRewards(rewardMonsterID, { rewards });
      message.success('战斗奖励配置已保存');
      setRewardEditorOpen(false);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存战斗奖励配置失败');
    } finally {
      setRewardSaving(false);
    }
  }

  const columns = useMemo<ColumnsType<AdminMonsterDefinitionSummary>>(
    () => [
      { title: '怪物ID', dataIndex: 'monster_id', key: 'monster_id', width: 100, fixed: 'left' },
      { title: '名称', dataIndex: 'monster_name', key: 'monster_name', width: 160 },
      { title: '等级', dataIndex: 'level', key: 'level', width: 90 },
      { title: '品质', dataIndex: 'quality', key: 'quality', width: 90 },
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
              { key: 'view', label: '详情', onClick: () => void handleViewDetail(record.monster_id) },
              { key: 'edit', label: '编辑', onClick: () => void handleOpenEditor('edit', record.monster_id) },
              { key: 'rewards', label: '战斗奖励', onClick: () => void handleOpenBattleRewards(record) },
              {
                key: 'delete',
                label: '删除',
                danger: true,
                confirm: { title: '确认删除这个系统怪物模板吗？', okText: '确认删除', cancelText: '取消' },
                onClick: () => void handleDelete(record.monster_id),
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
        title="系统怪物列表"
        extra={(
          <Form form={filterForm} layout="inline" onFinish={(values) => { setPage(1); setFilters(values); }}>
            <Form.Item name="monster_id" label="怪物ID">
              <Input allowClear placeholder="怪物ID" style={{ width: 100 }} />
            </Form.Item>
            <Form.Item name="name" label="名称">
              <Input allowClear placeholder="怪物名称" style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用">
              <Select allowClear placeholder="状态" style={{ width: 90 }} options={[{ label: '启用', value: 'true' }, { label: '停用', value: 'false' }]} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={loading}>查询</Button>
                <Button onClick={() => { filterForm.resetFields(); setPage(1); setFilters({}); }}>重置</Button>
                <Button type="primary" onClick={() => void handleOpenEditor('create')}>新增怪物</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      >
        <Table
          columns={columns}
          dataSource={rows}
          rowKey="monster_id"
          loading={loading}
          locale={{ emptyText: <Empty description="当前筛选条件下没有系统怪物模板" /> }}
          scroll={{ x: 900 }}
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

      <Drawer title={detail ? `系统怪物详情 · ${detail.monster_name}` : '系统怪物详情'} width={720} open={detailOpen} onClose={() => setDetailOpen(false)} destroyOnClose>
        {detailLoading || !detail ? (
          <Typography.Text type="secondary">正在加载系统怪物详情...</Typography.Text>
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small" title="基础信息">
              <Descriptions.Item label="怪物ID">{detail.monster_id}</Descriptions.Item>
              <Descriptions.Item label="名称">{detail.monster_name}</Descriptions.Item>
              <Descriptions.Item label="战斗外观ID">{detail.skin_id || '-'}</Descriptions.Item>
              <Descriptions.Item label="启用">{detail.is_enabled ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>{detail.description || '-'}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="战斗数值">
              <Descriptions.Item label="等级">{detail.base_stats.level}</Descriptions.Item>
              <Descriptions.Item label="品质">{detail.base_stats.quality}</Descriptions.Item>
              <Descriptions.Item label="生命">{detail.base_stats.hp} / {detail.base_stats.hp_max}</Descriptions.Item>
              <Descriptions.Item label="攻击">{detail.base_stats.atk}</Descriptions.Item>
              <Descriptions.Item label="防御">{detail.base_stats.def}</Descriptions.Item>
              <Descriptions.Item label="速度">{detail.base_stats.spd}</Descriptions.Item>
              <Descriptions.Item label="法力">{detail.base_stats.mana}</Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={1} size="small" title="技能">
              <Descriptions.Item label="技能名称">
                <SkillReferenceText skillIds={detail.skill_ids} map={skillReferenceMap} />
              </Descriptions.Item>
            </Descriptions>
            <Descriptions bordered column={2} size="small" title="捕捉配置">
              <Descriptions.Item label="可捕捉">{detail.is_capturable ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="关联宠物ID">{detail.is_capturable ? detail.capture_pet_id : '-'}</Descriptions.Item>
              <Descriptions.Item label="基础成功率">{detail.is_capturable ? `${detail.capture_rate_base} / 10000` : '-'}</Descriptions.Item>
              <Descriptions.Item label="最低生命百分比">{detail.is_capturable ? `${detail.capture_min_hp_pct}%` : '-'}</Descriptions.Item>
              <Descriptions.Item label="允许道具" span={2}>{detail.is_capturable ? detail.capture_item_ids.join(', ') || '-' : '-'}</Descriptions.Item>
            </Descriptions>
            <Button type="primary" onClick={() => void handleOpenBattleRewards({ monster_id: detail.monster_id, monster_name: detail.monster_name } as AdminMonsterDefinitionSummary)}>
              配置战斗奖励
            </Button>
          </Space>
        )}
      </Drawer>

      <Modal
        title={editingRecord ? `编辑系统怪物 · ${editingRecord.monster_name}` : '新增系统怪物模板'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={760}
        okText={editingRecord ? '保存修改' : '创建模板'}
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Row gutter={16}>
            <Col xs={24} md={8}><Form.Item label="怪物ID" name="monster_id" rules={[{ required: true, message: '请输入怪物ID' }]}><InputNumber min={1} disabled={Boolean(editingRecord)} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item label="怪物名称" name="monster_name" rules={[{ required: true, message: '请输入怪物名称' }]}><Input /></Form.Item></Col>
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
                <Input placeholder="例如：史莱姆_001" />
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
            <Col span={24}><Form.Item label="技能" name="skill_names_text" extra="填写已启用的系统技能名称，多个用英文逗号分隔"><Input placeholder="野性撞击,利爪突袭" /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item label="可捕捉" name="is_capturable" valuePropName="checked"><Switch /></Form.Item></Col>
            {isCapturable ? (
              <>
                <Col xs={24} md={8}>
                  <Form.Item label="关联野外宠物" name="capture_pet_id" rules={[{ required: true, message: '请选择捕捉成功后发放的系统宠物' }]}>
                    <Select
                      placeholder="选择 acquire_method=wild_capture 的宠物模板"
                      options={wildCapturePetOptions.map((item) => ({ label: `${item.pet_name} (#${item.pet_id})`, value: item.pet_id }))}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} md={8}><Form.Item label="基础捕捉成功率" name="capture_rate_base" extra="万分比，5000 表示 50%"><InputNumber min={1} max={10000} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item label="最低生命百分比" name="capture_min_hp_pct" extra="敌方生命低于该百分比才可尝试捕捉"><InputNumber min={1} max={100} style={{ width: '100%' }} /></Form.Item></Col>
                <Col span={24}><Form.Item label="允许捕捉道具ID" name="capture_item_ids_text" extra="多个道具用英文逗号分隔，例如 2001"><Input placeholder="2001" /></Form.Item></Col>
              </>
            ) : null}
            <Col xs={12} md={6}><Form.Item label="启用" name="is_enabled" valuePropName="checked"><Switch /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        title={rewardMonsterID ? `战斗奖励 · ${rewardMonsterName} (#${rewardMonsterID})` : '战斗奖励'}
        open={rewardEditorOpen}
        onCancel={() => setRewardEditorOpen(false)}
        onOk={() => rewardForm.submit()}
        confirmLoading={rewardSaving}
        destroyOnClose
        width={900}
        okText="保存奖励"
        cancelText="取消"
      >
        {rewardLoading ? (
          <Typography.Text type="secondary">正在加载战斗奖励配置...</Typography.Text>
        ) : (
          <Form form={rewardForm} layout="vertical" onFinish={(values) => void handleSubmitBattleRewards(values)}>
            <Typography.Paragraph type="secondary">
              支持经验、铜币与物品奖励，保存后将覆盖该怪物全部战斗奖励配置。
            </Typography.Paragraph>
            <Form.List name="rewards">
              {(fields, { add, remove }) => (
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  {fields.map((field, index) => (
                    <Card
                      key={field.key}
                      size="small"
                      title={`奖励 ${index + 1}`}
                      extra={<Button type="link" danger onClick={() => remove(field.name)}>删除</Button>}
                    >
                      <Row gutter={12}>
                        <Col xs={24} md={6}>
                          <Form.Item label="奖励类型" name={[field.name, 'reward_type']} rules={[{ required: true, message: '请选择奖励类型' }]}>
                            <Select options={[{ label: '经验', value: 'exp' }, { label: '铜币', value: 'gold' }, { label: '物品', value: 'item' }]} />
                          </Form.Item>
                        </Col>
                        <Form.Item noStyle shouldUpdate>
                          {() => {
                            const rewardType = rewardForm.getFieldValue(['rewards', field.name, 'reward_type']) as string | undefined;
                            if (rewardType === 'item') {
                              return (
                                <>
                                  <Col xs={24} md={6}>
                                    <Form.Item label="物品ID" name={[field.name, 'item_id']} rules={[{ required: true, message: '请输入物品ID' }]}>
                                      <InputNumber min={1} style={{ width: '100%' }} />
                                    </Form.Item>
                                  </Col>
                                  <Col xs={24} md={6}>
                                    <Form.Item label="数量" name={[field.name, 'quantity']} rules={[{ required: true, message: '请输入数量' }]}>
                                      <InputNumber min={1} style={{ width: '100%' }} />
                                    </Form.Item>
                                  </Col>
                                  <Col xs={24} md={6}>
                                    <Form.Item label="唯一掉落" name={[field.name, 'grant_once']} tooltip="开启后每名玩家仅首次获得该物品，之后战斗不再重复发放。">
                                      <Select options={[{ label: '否', value: 0 }, { label: '是', value: 1 }]} />
                                    </Form.Item>
                                  </Col>
                                </>
                              );
                            }
                            if (rewardType === 'gold') {
                              return (
                                <Col xs={24} md={6}>
                                  <Form.Item label="铜币数量" name={[field.name, 'exp_value']} rules={[{ required: true, message: '请输入铜币数量' }]}>
                                    <InputNumber min={1} style={{ width: '100%' }} />
                                  </Form.Item>
                                </Col>
                              );
                            }
                            return (
                              <>
                                <Col xs={24} md={6}>
                                  <Form.Item label="经验目标" name={[field.name, 'exp_target']} rules={[{ required: true, message: '请选择经验目标' }]}>
                                    <Select options={[{ label: '角色', value: 'player' }, { label: '宠物', value: 'pet' }]} />
                                  </Form.Item>
                                </Col>
                                <Col xs={24} md={6}>
                                  <Form.Item label="经验值" name={[field.name, 'exp_value']} rules={[{ required: true, message: '请输入经验值' }]}>
                                    <InputNumber min={1} style={{ width: '100%' }} />
                                  </Form.Item>
                                </Col>
                              </>
                            );
                          }}
                        </Form.Item>
                        <Col xs={12} md={4}>
                          <Form.Item label="排序" name={[field.name, 'sort_order']}>
                            <InputNumber min={1} style={{ width: '100%' }} />
                          </Form.Item>
                        </Col>
                        <Col xs={12} md={4}>
                          <Form.Item label="状态" name={[field.name, 'status']}>
                            <Select options={[{ label: '启用', value: 1 }, { label: '停用', value: 0 }]} />
                          </Form.Item>
                        </Col>
                      </Row>
                    </Card>
                  ))}
                  <Button type="dashed" block onClick={() => add(defaultMonsterBattleRewardRow())}>
                    新增奖励
                  </Button>
                </Space>
              )}
            </Form.List>
          </Form>
        )}
      </Modal>
    </Space>
  );
}

function mapMonsterBattleRewardPayload(item: AdminMonsterBattleRewardEntry, index: number): AdminMonsterBattleRewardEntry {
  const sortOrder: number = item.sort_order > 0 ? item.sort_order : index + 1;
  const status: number = item.status ?? 1;
  if (item.reward_type === 'item') {
    return {
      reward_type: 'item',
      exp_target: 'player',
      item_id: Number(item.item_id ?? 0),
      quantity: Number(item.quantity ?? 0),
      exp_value: 0,
      sort_order: sortOrder,
      status,
      grant_once: Number(item.grant_once ?? 0) > 0 ? 1 : 0,
    };
  }
  if (item.reward_type === 'gold') {
    return {
      reward_type: 'gold',
      exp_target: 'player',
      item_id: 0,
      quantity: 0,
      exp_value: Number(item.exp_value ?? 0),
      sort_order: sortOrder,
      status,
    };
  }
  return {
    reward_type: 'exp',
    exp_target: item.exp_target || 'player',
    item_id: 0,
    quantity: 0,
    exp_value: Number(item.exp_value ?? 0),
    sort_order: sortOrder,
    status,
  };
}

function defaultMonsterBattleRewardRow(): AdminMonsterBattleRewardEntry {
  return {
    reward_type: 'exp',
    exp_target: 'player',
    item_id: 0,
    quantity: 0,
    exp_value: 10,
    sort_order: 1,
    status: 1,
    grant_once: 0,
  };
}

function defaultMonsterDefinitionValues(): MonsterDefinitionFormValues {
  return {
    monster_id: 9003,
    monster_name: '新系统怪物',
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
    mana: 8,
    skill_ids: [],
    skill_names_text: '野性撞击,利爪突袭',
    is_capturable: false,
    capture_pet_id: 0,
    capture_rate_base: 5000,
    capture_min_hp_pct: 30,
    capture_item_ids: [2001],
    capture_item_ids_text: '2001',
  };
}

function mapDetailToFormValues(detail: AdminMonsterDefinitionDetail, skillReferenceMap: SkillReferenceMap): MonsterDefinitionFormValues {
  return {
    monster_id: detail.monster_id,
    monster_name: detail.monster_name,
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
    skill_ids: detail.skill_ids,
    skill_names_text: formatSkillReferenceInput(detail.skill_ids, skillReferenceMap),
    is_capturable: detail.is_capturable,
    capture_pet_id: detail.capture_pet_id,
    capture_rate_base: detail.capture_rate_base,
    capture_min_hp_pct: detail.capture_min_hp_pct,
    capture_item_ids: detail.capture_item_ids,
    capture_item_ids_text: detail.capture_item_ids.join(','),
  };
}

function buildPayloadFromForm(values: MonsterDefinitionFormValues, skillReferenceMap: SkillReferenceMap): AdminUpsertMonsterDefinitionPayload {
  const skillIDs = parseSkillReferenceInput(values.skill_names_text ?? '', skillReferenceMap);
  const captureItemIDs = parseNumberListInput(values.capture_item_ids_text ?? '');
  return {
    monster_id: Number(values.monster_id),
    monster_name: values.monster_name.trim(),
    description: values.description?.trim() ?? '',
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
    skill_ids: skillIDs,
    is_capturable: Boolean(values.is_capturable),
    capture_pet_id: values.is_capturable ? Number(values.capture_pet_id) : 0,
    capture_rate_base: Number(values.capture_rate_base || 5000),
    capture_min_hp_pct: Number(values.capture_min_hp_pct || 30),
    capture_item_ids: values.is_capturable ? captureItemIDs : [],
  };
}

function parseNumberListInput(raw: string): number[] {
  return raw
    .split(',')
    .map((part) => Number(part.trim()))
    .filter((value) => Number.isFinite(value) && value > 0);
}
