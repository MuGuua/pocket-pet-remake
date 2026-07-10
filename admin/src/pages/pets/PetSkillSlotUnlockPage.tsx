import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
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
  createAdminPetSkillSlotUnlockItem,
  deleteAdminPetSkillSlotUnlockItem,
  fetchAdminPetSkillSlotUnlockItems,
  updateAdminPetSkillSlotUnlockItem,
} from '../../services/petSkillSlotUnlock';
import type {
  AdminPetSkillSlotUnlockItem,
  AdminUpsertPetSkillSlotUnlockPayload,
} from '../../types/petSkillSlotUnlock';
import { PET_TALISMAN_SLOT_OPTIONS } from '../../types/petSkillSlotUnlock';

interface EditorFormValues extends AdminUpsertPetSkillSlotUnlockPayload {
  status_enabled: boolean;
}

function slotLabel(slotKey: string): string {
  return PET_TALISMAN_SLOT_OPTIONS.find((item) => item.value === slotKey)?.label ?? slotKey;
}

// 神符槽解锁道具映射配置页：维护 pet_skill_slot_unlock_item 表。
export function PetSkillSlotUnlockPage() {
  const [rows, setRows] = useState<AdminPetSkillSlotUnlockItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminPetSkillSlotUnlockItem | null>(null);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm<EditorFormValues>();

  useEffect(() => {
    void loadRows();
  }, []);

  async function loadRows() {
    setLoading(true);
    try {
      const items = await fetchAdminPetSkillSlotUnlockItems();
      setRows(items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载神符槽解锁配置失败');
      setRows([]);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditingRecord(null);
    form.setFieldsValue({
      slot_key: 'talisman_1',
      item_id: 0,
      description: '',
      status_enabled: true,
    });
    setEditorOpen(true);
  }

  function openEdit(record: AdminPetSkillSlotUnlockItem) {
    setEditingRecord(record);
    form.setFieldsValue({
      slot_key: record.slot_key,
      item_id: record.item_id,
      description: record.description,
      status_enabled: record.status === 1,
    });
    setEditorOpen(true);
  }

  async function handleSubmit(values: EditorFormValues) {
    setSaving(true);
    const payload: AdminUpsertPetSkillSlotUnlockPayload = {
      slot_key: values.slot_key,
      item_id: Number(values.item_id),
      description: values.description?.trim() ?? '',
      status: values.status_enabled ? 1 : 0,
    };
    try {
      if (editingRecord) {
        await updateAdminPetSkillSlotUnlockItem(editingRecord.slot_key, payload);
        message.success('解锁配置已更新');
      } else {
        await createAdminPetSkillSlotUnlockItem(payload);
        message.success('解锁配置已创建');
      }
      setEditorOpen(false);
      setEditingRecord(null);
      await loadRows();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败');
    } finally {
      setSaving(false);
    }
  }

  function confirmDelete(record: AdminPetSkillSlotUnlockItem) {
    Modal.confirm({
      title: `删除 ${slotLabel(record.slot_key)} 的解锁映射？`,
      content: '删除后对应道具将无法再开启神符槽，请确认无线上玩家依赖。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await deleteAdminPetSkillSlotUnlockItem(record.slot_key);
        message.success('已删除');
        await loadRows();
      },
    });
  }

  const columns: ColumnsType<AdminPetSkillSlotUnlockItem> = [
    { title: '槽位键', dataIndex: 'slot_key', width: 160, render: (value: string) => slotLabel(value) },
    { title: 'slot_key', dataIndex: 'slot_key', width: 140 },
    { title: '道具ID', dataIndex: 'item_id', width: 100 },
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
      width: 160,
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => openEdit(record)}>编辑</Button>
          <Button type="link" danger onClick={() => confirmDelete(record)}>删除</Button>
        </Space>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Typography.Title level={4} style={{ margin: 0 }}>神符槽解锁配置</Typography.Title>
          <Typography.Text type="secondary">
            维护道具与神符槽的映射；物品模板需设置 effect_type=pet_talisman_slot_unlock，使用时带 target_pet_uid。
          </Typography.Text>
          <Button type="primary" onClick={openCreate}>新增映射</Button>
        </Space>
      </Card>
      <Card>
        <Table
          rowKey="slot_key"
          loading={loading}
          columns={columns}
          dataSource={rows}
          pagination={false}
        />
      </Card>
      <Modal
        title={editingRecord ? '编辑解锁映射' : '新增解锁映射'}
        open={editorOpen}
        onCancel={() => { setEditorOpen(false); setEditingRecord(null); }}
        onOk={() => form.submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Form.Item label="神符槽" name="slot_key" rules={[{ required: true, message: '请选择槽位' }]}>
            <Select
              options={PET_TALISMAN_SLOT_OPTIONS}
              disabled={Boolean(editingRecord)}
            />
          </Form.Item>
          <Form.Item label="道具ID" name="item_id" rules={[{ required: true, message: '请输入道具ID' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="说明" name="description" extra="可在下方预览中刷色。">
            <RichTextEditor rows={2} />
          </Form.Item>
          <Form.Item label="启用" name="status_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
