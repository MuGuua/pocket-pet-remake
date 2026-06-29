import { Button, Card, Form, InputNumber, Modal, Select, Space, Switch, Table, Tag, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo, useState } from 'react';
import { RichTextEditor } from '../../components/RichTextEditor';
import {
  fetchAdminEnhanceSuccessConfigs,
  updateAdminEnhanceSuccessConfig,
} from '../../services/equipmentEnhanceSuccess';
import type {
  AdminEnhanceSuccessConfig,
  AdminUpsertEnhanceSuccessConfigPayload,
} from '../../types/equipmentEnhanceSuccess';
import { ENHANCE_REQUIRED_LEVEL_BAND_OPTIONS } from '../../types/equipmentEnhanceSuccess';

interface EnhanceSuccessEditorFormValues {
  success_rate_pct: number;
  description: string;
  status: boolean;
}

interface EquipmentEnhanceSuccessPageProps {
  embedded?: boolean;
}

/** 全局强化成功率配置页：按穿戴等级段 + 目标强化等级维护基础成功率。 */
export function EquipmentEnhanceSuccessPage({ embedded = false }: EquipmentEnhanceSuccessPageProps) {
  const [selectedBandMin, setSelectedBandMin] = useState<number>(1);
  const [rows, setRows] = useState<AdminEnhanceSuccessConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<AdminEnhanceSuccessConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [editorForm] = Form.useForm<EnhanceSuccessEditorFormValues>();

  useEffect(() => {
    void loadConfigs(selectedBandMin);
  }, [selectedBandMin]);

  async function loadConfigs(requiredLevelMin: number) {
    setLoading(true);
    try {
      const items = await fetchAdminEnhanceSuccessConfigs(requiredLevelMin);
      setRows(items);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载全局强化成功率失败');
      setRows([]);
    } finally {
      setLoading(false);
    }
  }

  function openEditor(record: AdminEnhanceSuccessConfig) {
    setEditingRecord(record);
    editorForm.setFieldsValue({
      success_rate_pct: record.success_rate_pct,
      description: record.description ?? '',
      status: record.status === 1,
    });
    setEditorOpen(true);
  }

  async function handleSubmitEditor(values: EnhanceSuccessEditorFormValues) {
    if (!editingRecord) {
      return;
    }
    setSaving(true);
    const payload: AdminUpsertEnhanceSuccessConfigPayload = {
      success_rate_pct: Math.trunc(Number(values.success_rate_pct ?? 0)),
      description: String(values.description ?? '').trim(),
      status: values.status ? 1 : 0,
    };
    try {
      await updateAdminEnhanceSuccessConfig(
        editingRecord.required_level_min,
        editingRecord.target_level,
        payload,
      );
      message.success(
        `已更新穿戴 ${editingRecord.required_level_band_label || `${editingRecord.required_level_min}级段`} · +${editingRecord.target_level} 的成功率`,
      );
      setEditorOpen(false);
      setEditingRecord(null);
      editorForm.resetFields();
      await loadConfigs(selectedBandMin);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存全局强化成功率失败');
    } finally {
      setSaving(false);
    }
  }

  const columns = useMemo<ColumnsType<AdminEnhanceSuccessConfig>>(
    () => [
      {
        title: '目标强化等级',
        dataIndex: 'target_level',
        key: 'target_level',
        width: 120,
        render: (value: number) => `+${value}`,
      },
      {
        title: '成功率',
        dataIndex: 'success_rate_pct',
        key: 'success_rate_pct',
        width: 120,
        render: (value: number) => `${value}%`,
      },
      {
        title: '说明',
        dataIndex: 'description',
        key: 'description',
        render: (value: string) => value?.trim() || '-',
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: number) => (
          value === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>
        ),
      },
      {
        title: '操作',
        key: 'actions',
        width: 100,
        render: (_value, record) => (
          <Button size="small" onClick={() => openEditor(record)}>编辑</Button>
        ),
      },
    ],
    [rows],
  );

  const selectedBandLabel = ENHANCE_REQUIRED_LEVEL_BAND_OPTIONS.find(
    (option) => option.value === selectedBandMin,
  )?.label ?? `${selectedBandMin}级段`;

  const content = (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {!embedded ? (
        <>
          <Typography.Title level={3} style={{ marginTop: 0 }}>
            全局强化成功率
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            按装备穿戴等级段（每10级）与目标强化等级配置基础成功率；穿戴等级越高可单独调低成功率。
          </Typography.Paragraph>
        </>
      ) : (
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
          先选择穿戴等级段，再维护 +1~+15 各档基础成功率；强化石「沿用全局」会读取当前装备所属段的数据。
        </Typography.Paragraph>
      )}
      <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
        <Space wrap>
          <Typography.Text>穿戴等级段：</Typography.Text>
          <Select
            style={{ minWidth: 160 }}
            value={selectedBandMin}
            options={ENHANCE_REQUIRED_LEVEL_BAND_OPTIONS}
            onChange={(nextValue: number) => setSelectedBandMin(nextValue)}
          />
        </Space>
        <Typography.Text type="secondary">
          当前段：{selectedBandLabel}
        </Typography.Text>
      </Space>
      <Table
        rowKey={(record) => `${record.required_level_min}-${record.target_level}`}
        size="small"
        loading={loading}
        columns={columns}
        dataSource={rows}
        pagination={false}
        scroll={{ x: 760 }}
        locale={{ emptyText: '当前穿戴等级段尚未配置强化成功率，请确认已执行 070 迁移。' }}
      />
      <Modal
        title={editingRecord
          ? `编辑 ${selectedBandLabel} · +${editingRecord.target_level}`
          : '编辑强化成功率'}
        open={editorOpen}
        onCancel={() => {
          setEditorOpen(false);
          setEditingRecord(null);
          editorForm.resetFields();
        }}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        width={520}
      >
        <Form
          form={editorForm}
          layout="vertical"
          onFinish={(values) => void handleSubmitEditor(values)}
        >
          <Form.Item
            label="成功率（%）"
            name="success_rate_pct"
            rules={[{ required: true, message: '请输入成功率' }]}
            extra="该穿戴等级段内所有装备，从上一级强化到 +N 时使用此基础概率。"
          >
            <InputNumber min={0} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="说明" name="description" extra="运营备注，支持 BBCode 富文本。">
            <RichTextEditor rows={2} placeholder="例如：41~50级装备强化至 +10" showPreview={false} />
          </Form.Item>
          <Form.Item label="启用" name="status" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );

  if (embedded) {
    return content;
  }
  return <Card>{content}</Card>;
}
