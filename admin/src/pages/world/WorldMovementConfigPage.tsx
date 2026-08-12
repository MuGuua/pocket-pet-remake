import { Alert, Button, Card, Descriptions, Form, Input, InputNumber, Modal, Space, Spin, Statistic, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { fetchWorldMovementConfig, updateWorldMovementConfig } from '../../services/worldMovement';
import type { UpdateWorldMovementConfigPayload, WorldMovementConfig } from '../../types/worldMovement';

// 世界移动配置页只维护数据库权威参数；保存成功即代表当前服务进程已经刷新运行时快照。
export function WorldMovementConfigPage() {
  const [config, setConfig] = useState<WorldMovementConfig | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [editorOpen, setEditorOpen] = useState<boolean>(false);
  const [form] = Form.useForm<UpdateWorldMovementConfigPayload>();

  async function loadConfig(): Promise<void> {
    setLoading(true);
    try {
      setConfig(await fetchWorldMovementConfig());
    } catch (error) {
      message.error(error instanceof Error ? error.message : '读取世界移动配置失败');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadConfig();
  }, []);

  function openEditor(): void {
    if (!config) {
      return;
    }
    form.setFieldsValue({
      speed_milli_cells_per_second: config.speed_milli_cells_per_second,
      max_elapsed_ms: config.max_elapsed_ms,
      axis_tolerance_milli: config.axis_tolerance_milli,
      reason: '',
    });
    setEditorOpen(true);
  }

  async function submitUpdate(): Promise<void> {
    const values = await form.validateFields();
    Modal.confirm({
      title: '确认立即更新权威移动参数？',
      content: '保存后会立即影响当前服务进程中新收到的移动意图，请确认数值和操作原因无误。',
      okText: '确认更新',
      cancelText: '返回检查',
      onOk: async () => {
        setSaving(true);
        try {
          const updated = await updateWorldMovementConfig(values);
          setConfig(updated);
          setEditorOpen(false);
          message.success('数据库配置与运行时生效值已同步更新');
        } catch (error) {
          message.error(error instanceof Error ? error.message : '更新世界移动配置失败');
          throw error;
        } finally {
          setSaving(false);
        }
      },
    });
  }

  return (
    <Card
      title="世界移动配置"
      extra={<Button type="primary" disabled={!config} onClick={openEditor}>调整参数</Button>}
    >
      <Alert
        showIcon
        type="warning"
        message="服务端权威配置"
        description="数据库配置是唯一事实来源。后台保存成功后，服务端会立即替换运行时快照；客户端只提交输入和展示纠偏结果。"
        style={{ marginBottom: 20 }}
      />
      <Spin spinning={loading}>
        {config ? (
          <>
            <Space size={32} wrap style={{ marginBottom: 24 }}>
              <Statistic title="配置值：移动速度" value={config.speed_milli_cells_per_second} suffix="千分之一格/秒" />
              <Statistic title="展示换算值" value={(config.speed_milli_cells_per_second / 1000).toFixed(3)} suffix="格/秒" />
              <Statistic title="最大计算时间窗" value={config.max_elapsed_ms} suffix="毫秒" />
              <Statistic title="非主轴容差" value={config.axis_tolerance_milli} suffix="千分之一格" />
            </Space>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="最终生效值">与当前数据库配置一致</Descriptions.Item>
              <Descriptions.Item label="更新时间">{new Date(config.updated_at).toLocaleString('zh-CN')}</Descriptions.Item>
              <Descriptions.Item label="更新管理员 ID">{config.updated_by_admin_user_id || '系统初始化'}</Descriptions.Item>
              <Descriptions.Item label="最近操作原因">{config.last_update_reason || '未记录'}</Descriptions.Item>
            </Descriptions>
          </>
        ) : <Typography.Text type="secondary">暂无可展示的移动配置</Typography.Text>}
      </Spin>

      <Modal
        title="调整世界移动参数"
        open={editorOpen}
        width={620}
        okText="校验并保存"
        cancelText="取消"
        confirmLoading={saving}
        onCancel={() => setEditorOpen(false)}
        onOk={() => void submitUpdate()}
        styles={{ body: { maxHeight: 520, overflowY: 'auto' } }}
      >
        <Form form={form} layout="vertical" disabled={saving}>
          <Form.Item label="移动速度（千分之一格/秒）" name="speed_milli_cells_per_second" rules={[{ required: true }, { type: 'number', min: 1 }]}>
            <InputNumber min={1} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="最大计算时间窗（毫秒）" name="max_elapsed_ms" rules={[{ required: true }, { type: 'number', min: 50, max: 2000 }]}>
            <InputNumber min={50} max={2000} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="非主轴容差（千分之一格）" name="axis_tolerance_milli" rules={[{ required: true }, { type: 'number', min: 0, max: 1000 }]}>
            <InputNumber min={0} max={1000} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="操作原因" name="reason" rules={[{ required: true, whitespace: true, message: '请输入本次调整原因' }, { max: 500 }]}>
            <Input.TextArea rows={4} placeholder="请说明调整背景、预期影响和回滚依据" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
