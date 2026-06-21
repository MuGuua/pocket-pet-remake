import { Button, Card, Form, InputNumber, Modal, Space, Switch, Table, Tabs, Tag, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import {
  fetchAdminPetAttrConvertConfigs,
  fetchAdminPetLevelConfigs,
  updateAdminPetAttrConvertConfig,
  updateAdminPetLevelConfig,
} from '../../services/petProgression';
import type {
  AdminPetAttrConvertConfig,
  AdminPetLevelConfig,
  AdminUpsertPetAttrConvertPayload,
  AdminUpsertPetLevelConfigPayload,
} from '../../types/petProgression';

const ATTR_TYPE_LABELS: Record<string, string> = {
  hp_max: '生命',
  atk: '攻击',
  spd: '速度',
  mana: '法力',
  def: '防御',
};

interface LevelEditorFormValues {
  exp_required: number;
  attr_points: number;
  status: boolean;
}

interface ConvertEditorFormValues {
  convert_rate: number;
  status: boolean;
}

function formatAttrType(value: string): string {
  return ATTR_TYPE_LABELS[value] ?? value;
}

// 宠物成长配置页：维护 1~100 级经验曲线与资质→战斗属性转化率。
export function PetProgressionPage() {
  const [levelRows, setLevelRows] = useState<AdminPetLevelConfig[]>([]);
  const [convertRows, setConvertRows] = useState<AdminPetAttrConvertConfig[]>([]);
  const [levelLoading, setLevelLoading] = useState(false);
  const [convertLoading, setConvertLoading] = useState(false);
  const [levelEditorOpen, setLevelEditorOpen] = useState(false);
  const [convertEditorOpen, setConvertEditorOpen] = useState(false);
  const [editingLevel, setEditingLevel] = useState<AdminPetLevelConfig | null>(null);
  const [editingConvert, setEditingConvert] = useState<AdminPetAttrConvertConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [levelForm] = Form.useForm<LevelEditorFormValues>();
  const [convertForm] = Form.useForm<ConvertEditorFormValues>();

  useEffect(() => {
    void loadLevelConfigs();
    void loadConvertConfigs();
  }, []);

  async function loadLevelConfigs() {
    setLevelLoading(true);
    try {
      const items = await fetchAdminPetLevelConfigs();
      setLevelRows(items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物等级经验配置失败');
      setLevelRows([]);
    } finally {
      setLevelLoading(false);
    }
  }

  async function loadConvertConfigs() {
    setConvertLoading(true);
    try {
      const items = await fetchAdminPetAttrConvertConfigs();
      setConvertRows(items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载宠物转化率配置失败');
      setConvertRows([]);
    } finally {
      setConvertLoading(false);
    }
  }

  function openLevelEditor(record: AdminPetLevelConfig) {
    setEditingLevel(record);
    levelForm.setFieldsValue({
      exp_required: record.exp_required,
      attr_points: record.attr_points,
      status: record.status === 1,
    });
    setLevelEditorOpen(true);
  }

  function openConvertEditor(record: AdminPetAttrConvertConfig) {
    setEditingConvert(record);
    convertForm.setFieldsValue({
      convert_rate: record.convert_rate,
      status: record.status === 1,
    });
    setConvertEditorOpen(true);
  }

  async function submitLevelEditor(values: LevelEditorFormValues) {
    if (!editingLevel) return;
    setSaving(true);
    try {
      await updateAdminPetLevelConfig(editingLevel.level, {
        exp_required: values.exp_required,
        attr_points: values.attr_points,
        status: values.status ? 1 : 0,
      });
      message.success(`宠物等级 ${editingLevel.level} 配置已更新`);
      setLevelEditorOpen(false);
      await loadLevelConfigs();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新宠物等级配置失败');
    } finally {
      setSaving(false);
    }
  }

  async function submitConvertEditor(values: ConvertEditorFormValues) {
    if (!editingConvert) return;
    setSaving(true);
    try {
      await updateAdminPetAttrConvertConfig(editingConvert.attr_type, {
        convert_rate: values.convert_rate,
        status: values.status ? 1 : 0,
      });
      message.success(`${formatAttrType(editingConvert.attr_type)} 转化率已更新`);
      setConvertEditorOpen(false);
      await loadConvertConfigs();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新宠物转化率配置失败');
    } finally {
      setSaving(false);
    }
  }

  const levelColumns: ColumnsType<AdminPetLevelConfig> = useMemo(
    () => [
      { title: '等级', dataIndex: 'level', width: 80 },
      { title: '升级所需经验', dataIndex: 'exp_required' },
      { title: '升级奖励自由点', dataIndex: 'attr_points' },
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

  const convertColumns: ColumnsType<AdminPetAttrConvertConfig> = useMemo(
    () => [
      {
        title: '战斗属性',
        dataIndex: 'attr_type',
        render: (value: string) => formatAttrType(value),
      },
      { title: '转化率常数', dataIndex: 'convert_rate' },
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
        宠物成长配置
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        服务端会在宠物获得经验、升级和分配属性点时读取这里的配置。修改后刷新运行时缓存立即生效。
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
            label: '资质转化率',
            children: (
              <Card>
                <Table
                  rowKey="attr_type"
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
        title={editingLevel ? `编辑宠物等级 ${editingLevel.level} 配置` : '编辑等级配置'}
        open={levelEditorOpen}
        onCancel={() => setLevelEditorOpen(false)}
        onOk={() => levelForm.submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={levelForm} layout="vertical" onFinish={(values: LevelEditorFormValues) => void submitLevelEditor(values)}>
          <Form.Item name="exp_required" label="升到下一级所需经验" rules={[{ required: true, message: '请输入经验值' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="attr_points" label="升级奖励自由属性点" rules={[{ required: true, message: '请输入属性点' }]}>
            <InputNumber min={0} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingConvert ? `编辑 ${formatAttrType(editingConvert.attr_type)} 转化率` : '编辑转化率'}
        open={convertEditorOpen}
        onCancel={() => setConvertEditorOpen(false)}
        onOk={() => convertForm.submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={convertForm} layout="vertical" onFinish={(values: ConvertEditorFormValues) => void submitConvertEditor(values)}>
          <Form.Item name="convert_rate" label="转化率常数（有效资质 / 转化率 × 分配点）" rules={[{ required: true, message: '请输入转化率' }]}>
            <InputNumber min={0.0001} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
