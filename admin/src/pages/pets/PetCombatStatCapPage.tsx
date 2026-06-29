import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useState } from 'react';
import { RichTextEditor } from '../../components/RichTextEditor';
import {
  fetchAdminPetCombatStatCaps,
  updateAdminPetCombatStatCap,
} from '../../services/petCombatStatCap';
import type {
  AdminPetCombatStatCap,
  AdminUpsertPetCombatStatCapPayload,
} from '../../types/petCombatStatCap';
import { formatPetCombatStatCapLabel } from '../../types/petCombatStatCap';

interface EditorFormValues extends AdminUpsertPetCombatStatCapPayload {
  status_enabled: boolean;
}

// 宠物战斗属性封顶配置页：维护 pet_combat_stat_cap 表，保存后影响运行时截断与后台宠物编辑。
export function PetCombatStatCapPage() {
  const [rows, setRows] = useState<AdminPetCombatStatCap[]>([]);
  const [loading, setLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminPetCombatStatCap | null>(null);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm<EditorFormValues>();

  useEffect(() => {
    void loadRows();
  }, []);

  async function loadRows() {
    setLoading(true);
    try {
      setRows(await fetchAdminPetCombatStatCaps());
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载封顶配置失败');
      setRows([]);
    } finally {
      setLoading(false);
    }
  }

  function openEdit(record: AdminPetCombatStatCap) {
    setEditingRecord(record);
    form.setFieldsValue({
      cap_value: record.cap_value,
      description: record.description,
      status_enabled: record.status === 1,
    });
    setEditorOpen(true);
  }

  async function handleSubmit(values: EditorFormValues) {
    if (!editingRecord) {
      return;
    }
    setSaving(true);
    const payload: AdminUpsertPetCombatStatCapPayload = {
      cap_value: Number(values.cap_value),
      description: values.description?.trim() ?? '',
      status: values.status_enabled ? 1 : 0,
    };
    try {
      await updateAdminPetCombatStatCap(editingRecord.stat_key, payload);
      message.success('封顶配置已更新');
      setEditorOpen(false);
      setEditingRecord(null);
      await loadRows();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败');
    } finally {
      setSaving(false);
    }
  }

  const columns: ColumnsType<AdminPetCombatStatCap> = [
    {
      title: '属性',
      dataIndex: 'stat_key',
      width: 180,
      render: (value: string) => formatPetCombatStatCapLabel(value),
    },
    { title: 'stat_key', dataIndex: 'stat_key', width: 220 },
    { title: '封顶值', dataIndex: 'cap_value', width: 120 },
    { title: '说明', dataIndex: 'description' },
    {
      title: '状态',
      dataIndex: 'status_text',
      width: 90,
      render: (text: string, record) => (
        <Tag color={record.status === 1 ? 'green' : 'default'}>{text}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 100,
      render: (_, record) => (
        <Button type="link" onClick={() => openEdit(record)}>编辑</Button>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Typography.Title level={4} style={{ marginTop: 0 }}>战斗属性封顶（人物/宠物共用）</Typography.Title>
        <Typography.Text type="secondary">
          对应数据库表 pet_combat_stat_cap；更新后立即影响宠物列表读取与后台宠物保存时的截断逻辑。
        </Typography.Text>
      </Card>
      <Card>
        <Table rowKey="stat_key" loading={loading} columns={columns} dataSource={rows} pagination={false} />
      </Card>
      <Modal
        title={editingRecord ? `编辑封顶 · ${formatPetCombatStatCapLabel(editingRecord.stat_key)}` : '编辑封顶'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); }}
        onOk={() => form.submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Form.Item label="封顶值" name="cap_value" rules={[{ required: true, message: '请输入封顶值' }]}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="说明" name="description" extra="支持 BBCode 富文本。">
            <RichTextEditor rows={2} showPreview={false} />
          </Form.Item>
          <Form.Item label="启用" name="status_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
