import { Button, Card, Form, InputNumber, Modal, Select, Space, Switch, Table, Tabs, Tag, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import {
  fetchAdminPlayerAttrConvertConfigs,
  fetchAdminPlayerLevelConfigs,
  updateAdminPlayerAttrConvertConfig,
  updateAdminPlayerLevelConfig,
} from '../../services/playerProgression';
import type {
  AdminPlayerAttrConvertConfig,
  AdminPlayerLevelConfig,
  AdminUpsertPlayerAttrConvertPayload,
  AdminUpsertPlayerLevelConfigPayload,
} from '../../types/playerProgression';

const SOURCE_ATTR_LABELS: Record<string, string> = {
  strength: '力量',
  vitality: '体质',
  agility: '敏捷',
  mind: '灵力',
};

const TARGET_ATTR_LABELS: Record<string, string> = {
  atk: '攻击',
  def: '防御',
  spd: '速度',
  mana: '法力',
  hp_max: '生命上限',
  hit_pct: '命中',
  dodge_pct: '闪避',
};

const SOURCE_ATTR_OPTIONS = [
  { label: '力量', value: 'strength' },
  { label: '体质', value: 'vitality' },
  { label: '敏捷', value: 'agility' },
  { label: '灵力', value: 'mind' },
];

const TARGET_ATTR_OPTIONS = [
  { label: '攻击', value: 'atk' },
  { label: '防御', value: 'def' },
  { label: '速度', value: 'spd' },
  { label: '法力', value: 'mana' },
  { label: '生命上限', value: 'hp_max' },
  { label: '命中', value: 'hit_pct' },
  { label: '闪避', value: 'dodge_pct' },
];

function formatSourceAttr(value: string): string {
  return SOURCE_ATTR_LABELS[value] ?? value;
}

function formatTargetAttr(value: string): string {
  return TARGET_ATTR_LABELS[value] ?? value;
}

// 玩家成长配置页：维护 1~100 级经验曲线与属性点转化率，供服务端升级/加点权威计算消费。
export function PlayerProgressionPage() {
  const [levelRows, setLevelRows] = useState<AdminPlayerLevelConfig[]>([]);
  const [convertRows, setConvertRows] = useState<AdminPlayerAttrConvertConfig[]>([]);
  const [levelLoading, setLevelLoading] = useState(false);
  const [convertLoading, setConvertLoading] = useState(false);
  const [levelEditorOpen, setLevelEditorOpen] = useState(false);
  const [convertEditorOpen, setConvertEditorOpen] = useState(false);
  const [editingLevel, setEditingLevel] = useState<AdminPlayerLevelConfig | null>(null);
  const [editingConvert, setEditingConvert] = useState<AdminPlayerAttrConvertConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [levelForm] = Form.useForm<AdminUpsertPlayerLevelConfigPayload>();
  const [convertForm] = Form.useForm<AdminUpsertPlayerAttrConvertPayload>();

  useEffect(() => {
    void loadLevelConfigs();
    void loadConvertConfigs();
  }, []);

  async function loadLevelConfigs() {
    setLevelLoading(true);
    try {
      const items = await fetchAdminPlayerLevelConfigs();
      setLevelRows(items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载等级经验配置失败');
      setLevelRows([]);
    } finally {
      setLevelLoading(false);
    }
  }

  async function loadConvertConfigs() {
    setConvertLoading(true);
    try {
      const items = await fetchAdminPlayerAttrConvertConfigs();
      setConvertRows(items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载属性转化率配置失败');
      setConvertRows([]);
    } finally {
      setConvertLoading(false);
    }
  }

  function openLevelEditor(record: AdminPlayerLevelConfig) {
    setEditingLevel(record);
    levelForm.setFieldsValue({
      exp_required: record.exp_required,
      attr_points: record.attr_points,
      bonus_atk: record.bonus_atk,
      bonus_hp_max: record.bonus_hp_max,
      bonus_spd: record.bonus_spd,
      bonus_mana: record.bonus_mana,
      status: record.status === 1,
    });
    setLevelEditorOpen(true);
  }

  function openConvertEditor(record: AdminPlayerAttrConvertConfig) {
    setEditingConvert(record);
    convertForm.setFieldsValue({
      source_attr: record.source_attr,
      target_attr: record.target_attr,
      convert_rate: record.convert_rate,
      status: record.status === 1,
    });
    setConvertEditorOpen(true);
  }

  async function submitLevelEditor(values: {
    exp_required: number;
    attr_points: number;
    bonus_atk: number;
    bonus_hp_max: number;
    bonus_spd: number;
    bonus_mana: number;
    status: boolean;
  }) {
    if (!editingLevel) return;
    setSaving(true);
    try {
      await updateAdminPlayerLevelConfig(editingLevel.level, {
        exp_required: values.exp_required,
        attr_points: values.attr_points,
        bonus_atk: values.bonus_atk,
        bonus_hp_max: values.bonus_hp_max,
        bonus_spd: values.bonus_spd,
        bonus_mana: values.bonus_mana,
        status: values.status ? 1 : 0,
      });
      message.success(`等级 ${editingLevel.level} 配置已更新`);
      setLevelEditorOpen(false);
      await loadLevelConfigs();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新等级配置失败');
    } finally {
      setSaving(false);
    }
  }

  async function submitConvertEditor(values: AdminUpsertPlayerAttrConvertPayload & { status: boolean }) {
    if (!editingConvert) return;
    setSaving(true);
    try {
      await updateAdminPlayerAttrConvertConfig(editingConvert.id, {
        source_attr: values.source_attr,
        target_attr: values.target_attr,
        convert_rate: values.convert_rate,
        status: Boolean(values.status) ? 1 : 0,
      });
      message.success(`转化率配置 #${editingConvert.id} 已更新`);
      setConvertEditorOpen(false);
      await loadConvertConfigs();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新转化率配置失败');
    } finally {
      setSaving(false);
    }
  }

  const levelColumns: ColumnsType<AdminPlayerLevelConfig> = useMemo(
    () => [
      { title: '等级', dataIndex: 'level', width: 80 },
      { title: '升级所需经验', dataIndex: 'exp_required' },
      { title: '升级奖励属性点', dataIndex: 'attr_points' },
      { title: '攻击加成', dataIndex: 'bonus_atk', width: 100 },
      { title: '生命上限加成', dataIndex: 'bonus_hp_max', width: 120 },
      { title: '速度加成', dataIndex: 'bonus_spd', width: 100 },
      { title: '法力加成', dataIndex: 'bonus_mana', width: 100 },
      {
        title: '状态',
        dataIndex: 'status',
        render: (value: number) => <Tag color={value === 1 ? 'green' : 'default'}>{value === 1 ? '启用' : '停用'}</Tag>,
      },
      {
        title: '操作',
        width: 100,
        render: (_, record) => (
          <Button type="link" onClick={() => openLevelEditor(record)}>
            编辑
          </Button>
        ),
      },
    ],
    [],
  );

  const convertColumns: ColumnsType<AdminPlayerAttrConvertConfig> = useMemo(
    () => [
      { title: 'ID', dataIndex: 'id', width: 80 },
      {
        title: '基础属性',
        dataIndex: 'source_attr',
        render: (value: string) => formatSourceAttr(value),
      },
      {
        title: '目标战斗属性',
        dataIndex: 'target_attr',
        render: (value: string) => formatTargetAttr(value),
      },
      { title: '每点转化率', dataIndex: 'convert_rate' },
      {
        title: '状态',
        dataIndex: 'status',
        render: (value: number) => <Tag color={value === 1 ? 'green' : 'default'}>{value === 1 ? '启用' : '停用'}</Tag>,
      },
      {
        title: '操作',
        width: 100,
        render: (_, record) => (
          <Button type="link" onClick={() => openConvertEditor(record)}>
            编辑
          </Button>
        ),
      },
    ],
    [],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Typography.Title level={4} style={{ margin: 0 }}>
        玩家成长配置
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        服务端会在玩家获得经验、升级和分配属性点时读取这里的配置。修改后即时生效，已分配属性点会按新转化率重算战斗属性。
      </Typography.Paragraph>

      <Tabs
        items={[
          {
            key: 'level',
            label: '等级经验表',
            children: (
              <Card>
                <Table
                  rowKey="level"
                  loading={levelLoading}
                  columns={levelColumns}
                  dataSource={levelRows}
                  pagination={{ pageSize: 20, showSizeChanger: true }}
                />
              </Card>
            ),
          },
          {
            key: 'convert',
            label: '属性转化率',
            children: (
              <Card>
                <Table
                  rowKey="id"
                  loading={convertLoading}
                  columns={convertColumns}
                  dataSource={convertRows}
                  pagination={false}
                />
              </Card>
            ),
          },
        ]}
      />

      <Modal
        title={editingLevel ? `编辑等级 ${editingLevel.level} 配置` : '编辑等级配置'}
        open={levelEditorOpen}
        onCancel={() => setLevelEditorOpen(false)}
        onOk={() => levelForm.submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={levelForm} layout="vertical" onFinish={(values) => void submitLevelEditor(values)}>
          <Form.Item name="exp_required" label="升到下一级所需经验" rules={[{ required: true, message: '请输入经验值' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="attr_points" label="升级奖励属性点" rules={[{ required: true, message: '请输入属性点' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="bonus_atk" label="升级攻击加成" rules={[{ required: true, message: '请输入攻击加成' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="bonus_hp_max" label="升级生命上限加成" rules={[{ required: true, message: '请输入生命上限加成' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="bonus_spd" label="升级速度加成" rules={[{ required: true, message: '请输入速度加成' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="bonus_mana" label="升级法力加成" rules={[{ required: true, message: '请输入法力加成' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingConvert ? `编辑转化率 #${editingConvert.id}` : '编辑转化率'}
        open={convertEditorOpen}
        onCancel={() => setConvertEditorOpen(false)}
        onOk={() => convertForm.submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={convertForm} layout="vertical" onFinish={(values) => void submitConvertEditor(values)}>
          <Form.Item name="source_attr" label="基础属性" rules={[{ required: true, message: '请选择基础属性' }]}>
            <Select options={SOURCE_ATTR_OPTIONS} />
          </Form.Item>
          <Form.Item name="target_attr" label="目标战斗属性" rules={[{ required: true, message: '请选择目标属性' }]}>
            <Select options={TARGET_ATTR_OPTIONS} />
          </Form.Item>
          <Form.Item name="convert_rate" label="每 1 点基础属性转化值" rules={[{ required: true, message: '请输入转化率' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
