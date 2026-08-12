import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Space,
  Spin,
  Statistic,
  Table,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useState } from 'react';
import {
  fetchSceneBoundaries,
  fetchWorldMovementConfig,
  updateSceneBoundary,
  updateWorldMovementConfig,
} from '../../services/worldMovement';
import type {
  SceneBoundary,
  UpdateSceneBoundaryPayload,
  UpdateWorldMovementConfigPayload,
  WorldMovementConfig,
} from '../../types/worldMovement';

// fixedCoordinateText 同时展示数据库整数和换算后的场景格，避免运营误解坐标单位。
function fixedCoordinateText(value: number): string {
  return `${value}（${(value / 1000).toFixed(3)} 格）`;
}

// 世界移动配置页统一维护移动参数和场景外边界；两类数据保存后都会立即刷新服务端运行时快照。
export function WorldMovementConfigPage() {
  const [config, setConfig] = useState<WorldMovementConfig | null>(null);
  const [boundaries, setBoundaries] = useState<SceneBoundary[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [savingConfig, setSavingConfig] = useState<boolean>(false);
  const [savingBoundary, setSavingBoundary] = useState<boolean>(false);
  const [configEditorOpen, setConfigEditorOpen] = useState<boolean>(false);
  const [boundaryEditorOpen, setBoundaryEditorOpen] = useState<boolean>(false);
  const [editingBoundary, setEditingBoundary] = useState<SceneBoundary | null>(null);
  const [configForm] = Form.useForm<UpdateWorldMovementConfigPayload>();
  const [boundaryForm] = Form.useForm<UpdateSceneBoundaryPayload>();

  // loadPageData 并行读取两个数据库事实来源，页面首次出现时不展示旧缓存值。
  async function loadPageData(): Promise<void> {
    setLoading(true);
    try {
      const [nextConfig, nextBoundaries] = await Promise.all([
        fetchWorldMovementConfig(),
        fetchSceneBoundaries(),
      ]);
      setConfig(nextConfig);
      setBoundaries(nextBoundaries);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '读取世界移动配置失败');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadPageData();
  }, []);

  // openConfigEditor 使用当前数据库返回值填表，操作原因始终要求重新填写。
  function openConfigEditor(): void {
    if (!config) {
      return;
    }
    configForm.setFieldsValue({
      speed_milli_cells_per_second: config.speed_milli_cells_per_second,
      max_elapsed_ms: config.max_elapsed_ms,
      axis_tolerance_milli: config.axis_tolerance_milli,
      reason: '',
    });
    setConfigEditorOpen(true);
  }

  // submitConfigUpdate 在最终写入前增加二次确认，明确提示即时生效范围。
  async function submitConfigUpdate(): Promise<void> {
    const values: UpdateWorldMovementConfigPayload = await configForm.validateFields();
    Modal.confirm({
      title: '确认立即更新权威移动参数？',
      content: '保存后会立即影响当前服务进程中新收到的移动意图，请确认数值和操作原因无误。',
      okText: '确认更新',
      cancelText: '返回检查',
      onOk: async () => {
        setSavingConfig(true);
        try {
          const updated: WorldMovementConfig = await updateWorldMovementConfig(values);
          setConfig(updated);
          setConfigEditorOpen(false);
          message.success('数据库配置与运行时生效值已同步更新');
        } catch (error) {
          message.error(error instanceof Error ? error.message : '更新世界移动配置失败');
          throw error;
        } finally {
          setSavingConfig(false);
        }
      },
    });
  }

  // openBoundaryEditor 只允许编辑服务端列表中存在的场景，并填入完整矩形以便整体校验。
  function openBoundaryEditor(boundary: SceneBoundary): void {
    setEditingBoundary(boundary);
    boundaryForm.setFieldsValue({
      min_x_milli: boundary.min_x_milli,
      min_y_milli: boundary.min_y_milli,
      max_x_milli: boundary.max_x_milli,
      max_y_milli: boundary.max_y_milli,
      reason: '',
    });
    setBoundaryEditorOpen(true);
  }

  // closeBoundaryEditor 清除选中项，防止下次打开短暂显示上一个场景标题。
  function closeBoundaryEditor(): void {
    if (savingBoundary) {
      return;
    }
    setBoundaryEditorOpen(false);
    setEditingBoundary(null);
  }

  // submitBoundaryUpdate 校验完整矩形并要求二次确认，避免错误边界立即影响在线移动。
  async function submitBoundaryUpdate(): Promise<void> {
    if (!editingBoundary) {
      return;
    }
    const values: UpdateSceneBoundaryPayload = await boundaryForm.validateFields();
    if (values.max_x_milli <= values.min_x_milli || values.max_y_milli <= values.min_y_milli) {
      message.error('最大坐标必须分别大于最小坐标');
      return;
    }
    Modal.confirm({
      title: `确认更新“${editingBoundary.scene_name}”移动边界？`,
      content: '保存后，新收到的玩家移动会立即被裁剪到该矩形内。墙体和精细阻挡不在本次配置范围内。',
      okText: '确认更新',
      cancelText: '返回检查',
      onOk: async () => {
        setSavingBoundary(true);
        try {
          const updated: SceneBoundary = await updateSceneBoundary(editingBoundary.scene_id, values);
          setBoundaries((current: SceneBoundary[]) => current.map((item: SceneBoundary) => (
            item.scene_id === updated.scene_id ? updated : item
          )));
          setBoundaryEditorOpen(false);
          setEditingBoundary(null);
          message.success('场景边界已写入数据库并刷新运行时缓存');
        } catch (error) {
          message.error(error instanceof Error ? error.message : '更新场景移动边界失败');
          throw error;
        } finally {
          setSavingBoundary(false);
        }
      },
    });
  }

  const boundaryColumns: ColumnsType<SceneBoundary> = [
    { title: '场景 ID', dataIndex: 'scene_id', width: 86, fixed: 'left' },
    {
      title: '场景',
      key: 'scene',
      width: 180,
      fixed: 'left',
      render: (_value: unknown, record: SceneBoundary) => (
        <Space direction="vertical" size={0}>
          <Typography.Text>{record.scene_name}</Typography.Text>
          <Typography.Text type="secondary" code>{record.scene_code}</Typography.Text>
        </Space>
      ),
    },
    { title: '最小 X', dataIndex: 'min_x_milli', width: 180, render: fixedCoordinateText },
    { title: '最小 Y', dataIndex: 'min_y_milli', width: 180, render: fixedCoordinateText },
    { title: '最大 X', dataIndex: 'max_x_milli', width: 180, render: fixedCoordinateText },
    { title: '最大 Y', dataIndex: 'max_y_milli', width: 180, render: fixedCoordinateText },
    {
      title: '审计信息',
      key: 'audit',
      width: 240,
      render: (_value: unknown, record: SceneBoundary) => (
        <Space direction="vertical" size={0}>
          <Typography.Text>{new Date(record.updated_at).toLocaleString('zh-CN')}</Typography.Text>
          <Typography.Text type="secondary">管理员：{record.updated_by_admin_user_id || '系统初始化'}</Typography.Text>
          <Typography.Text type="secondary" ellipsis={{ tooltip: record.last_update_reason }}>
            原因：{record.last_update_reason || '未记录'}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 90,
      fixed: 'right',
      render: (_value: unknown, record: SceneBoundary) => (
        <Button type="link" onClick={() => openBoundaryEditor(record)}>编辑</Button>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Space direction="vertical" size={20} style={{ width: '100%' }}>
        <Card
          title="世界移动配置"
          extra={<Button type="primary" disabled={!config} onClick={openConfigEditor}>调整参数</Button>}
        >
          <Alert
            showIcon
            type="warning"
            message="服务端权威配置"
            description="数据库配置是唯一事实来源。后台保存成功后，服务端会立即替换运行时快照；客户端只提交输入和展示纠偏结果。"
            style={{ marginBottom: 20 }}
          />
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
        </Card>

        <Card title="场景移动边界">
          <Alert
            showIcon
            type="info"
            message="矩形外边界"
            description="坐标为人物中心可到达的闭区间，数据库存储千分之一场景格。该配置只限制场景外围，墙体和装饰阻挡由后续静态通行数据维护。"
            style={{ marginBottom: 20 }}
          />
          <Table<SceneBoundary>
            rowKey="scene_id"
            columns={boundaryColumns}
            dataSource={boundaries}
            pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total: number) => `共 ${total} 个启用场景` }}
            scroll={{ x: 1510 }}
          />
        </Card>
      </Space>

      <Modal
        title="调整世界移动参数"
        open={configEditorOpen}
        width={620}
        okText="校验并保存"
        cancelText="取消"
        confirmLoading={savingConfig}
        onCancel={() => setConfigEditorOpen(false)}
        onOk={() => void submitConfigUpdate()}
        styles={{ body: { maxHeight: 520, overflowY: 'auto' } }}
      >
        <Form form={configForm} layout="vertical" disabled={savingConfig}>
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

      <Modal
        title={editingBoundary ? `编辑场景边界：${editingBoundary.scene_name}` : '编辑场景边界'}
        open={boundaryEditorOpen}
        width={680}
        okText="校验并保存"
        cancelText="取消"
        confirmLoading={savingBoundary}
        onCancel={closeBoundaryEditor}
        onOk={() => void submitBoundaryUpdate()}
        styles={{ body: { maxHeight: 520, overflowY: 'auto' } }}
      >
        <Alert
          showIcon
          type="warning"
          message="保存后立即影响在线移动"
          description="请使用千分之一场景格整数填写完整矩形，并确认传送出生点仍位于边界内。"
          style={{ marginBottom: 16 }}
        />
        <Form form={boundaryForm} layout="vertical" disabled={savingBoundary}>
          <Space size={16} align="start" style={{ width: '100%' }}>
            <Form.Item label="最小 X" name="min_x_milli" rules={[{ required: true }, { type: 'number', min: -10000000, max: 10000000 }]} style={{ flex: 1 }}>
              <InputNumber min={-10000000} max={10000000} precision={0} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item label="最小 Y" name="min_y_milli" rules={[{ required: true }, { type: 'number', min: -10000000, max: 10000000 }]} style={{ flex: 1 }}>
              <InputNumber min={-10000000} max={10000000} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Space>
          <Space size={16} align="start" style={{ width: '100%' }}>
            <Form.Item label="最大 X" name="max_x_milli" rules={[{ required: true }, { type: 'number', min: -10000000, max: 10000000 }]} style={{ flex: 1 }}>
              <InputNumber min={-10000000} max={10000000} precision={0} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item label="最大 Y" name="max_y_milli" rules={[{ required: true }, { type: 'number', min: -10000000, max: 10000000 }]} style={{ flex: 1 }}>
              <InputNumber min={-10000000} max={10000000} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Space>
          <Form.Item label="操作原因" name="reason" rules={[{ required: true, whitespace: true, message: '请输入本次调整原因' }, { max: 500 }]}>
            <Input.TextArea rows={4} placeholder="请说明地图资源依据、影响范围和回滚值" />
          </Form.Item>
        </Form>
      </Modal>
    </Spin>
  );
}
