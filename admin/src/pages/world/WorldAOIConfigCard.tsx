import { Alert, Button, Card, Descriptions, Form, Input, InputNumber, Modal, Space, Spin, Switch, Select, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { fetchWorldAOIConfig, updateWorldAOIConfig, fetchWorldSceneOnline } from '../../services/worldMovement';
import type { WorldAOIState, UpdateWorldAOIPayload, WorldSceneOnline } from '../../types/worldMovement';

// 沿用世界配置页的统一请求/确认机制，明确区分持久化配置与当前进程实际生效值。
export function WorldAOIConfigCard({ scenes }: { scenes: { scene_id: number; scene_name: string }[] }) {
  const [selectedScene, setSelectedScene] = useState<number | null>(null);
  const [online, setOnline] = useState<WorldSceneOnline | null>(null);
  const [onlineLoading, setOnlineLoading] = useState<boolean>(false);
  const [state, setState] = useState<WorldAOIState | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>('');
  const [open, setOpen] = useState<boolean>(false);
  const [saving, setSaving] = useState<boolean>(false);
  const [form] = Form.useForm<UpdateWorldAOIPayload>();

  async function reload(): Promise<void> {
    setLoading(true);
    try { setState(await fetchWorldAOIConfig()); setError(''); }
    catch (failure) { setError(failure instanceof Error ? failure.message : '读取 AOI 配置失败'); }
    finally { setLoading(false); }
  }
  useEffect(() => { void reload(); }, []);

  function edit(): void {
    if (!state) return;
    form.setFieldsValue({ enabled: state.configured.enabled, relay_enabled: state.configured.relay_enabled, cell_size_milli: state.configured.cell_size_milli, radius: state.configured.radius, reason: '' });
    setOpen(true);
  }

  async function save(): Promise<void> {
    const values = await form.validateFields();
    Modal.confirm({
      title: '确认保存下次启动使用的世界同步配置？',
      content: '邻域必须覆盖实际手机镜头及预加载余量。跨实例广播需所有参与实例配置一致。保存只写入数据库，当前连接继续沿用原配置，重启服务后生效。',
      okText: '确认保存', cancelText: '返回检查',
      onOk: async () => {
        setSaving(true);
        try {
          setState(await updateWorldAOIConfig(values));
          setOpen(false);
          message.success('配置已持久化，重启服务后生效');
        } catch (failure) {
          message.error(failure instanceof Error ? failure.message : '保存 AOI 配置失败');
          throw failure;
        } finally { setSaving(false); }
      },
    });
  }

  async function queryOnline(): Promise<void> {
    if (selectedScene == null) return;
    setOnlineLoading(true);
    setOnline(null);
    try { setOnline(await fetchWorldSceneOnline(selectedScene)); }
    catch (failure) { message.error(failure instanceof Error ? failure.message : '读取场景在线索引失败'); }
    finally { setOnlineLoading(false); }
  }

  return (
    <Card title="玩家网格可见范围（AOI）" extra={<Space><Button onClick={() => void reload()} loading={loading}>刷新</Button><Button onClick={edit} disabled={!state}>编辑配置</Button></Space>}>
      <Spin spinning={loading}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <Alert type="info" showIcon message="配置保存后重启服务生效；小地图默认保持全场景可见，开启前请核对镜头覆盖。" />
          <Space wrap>
            <Select placeholder="选择服务端场景" style={{ width: 220 }} value={selectedScene} disabled={onlineLoading} options={scenes.map((scene) => ({ value: scene.scene_id, label: `${scene.scene_name}（${scene.scene_id}）` }))} onChange={(value: number) => { setSelectedScene(value); setOnline(null); }} />
            <Button onClick={() => void queryOnline()} loading={onlineLoading} disabled={selectedScene == null}>查询 Redis 在线人数</Button>
            {online && <Typography.Text>场景 {online.scene_id} 租约在线人数：{online.count}</Typography.Text>}
          </Space>
          <Typography.Text type="secondary">共享索引按心跳租约清理过期成员；跨实例世界广播需所有参与实例启用并重启；战斗和 PVP 仍使用原单实例归属。</Typography.Text>
          {error && <Alert type="error" showIcon message={error} />}
          {state && <Descriptions bordered size="small" column={2}>
            <Descriptions.Item label="数据库跨实例世界广播">{state.configured.relay_enabled ? '已开启' : '已关闭'}</Descriptions.Item>
            <Descriptions.Item label="当前跨实例世界广播">{state.active.relay_enabled ? '已开启' : '已关闭'}</Descriptions.Item>
            <Descriptions.Item label="数据库开关">{state.configured.enabled ? '已开启' : '已关闭'}</Descriptions.Item>
            <Descriptions.Item label="当前进程开关">{state.active.enabled ? '已开启' : '已关闭'}</Descriptions.Item>
            <Descriptions.Item label="数据库网格边长（千分之一格）">{state.configured.cell_size_milli}</Descriptions.Item>
            <Descriptions.Item label="当前网格边长（千分之一格）">{state.active.cell_size_milli}</Descriptions.Item>
            <Descriptions.Item label="数据库邻域半径">{state.configured.radius}</Descriptions.Item>
            <Descriptions.Item label="当前邻域半径">{state.active.radius}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{state.configured.updated_at}</Descriptions.Item>
            <Descriptions.Item label="管理员标识">{state.configured.admin_user_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="最近操作原因" span={2}>{state.configured.reason}</Descriptions.Item>
          </Descriptions>}
        </Space>
      </Spin>
      <Modal title="编辑世界同步数据库配置" open={open} onCancel={() => !saving && setOpen(false)} onOk={() => void save()} confirmLoading={saving} destroyOnClose styles={{ body: { maxHeight: '60vh', overflowY: 'auto' } }}>
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="relay_enabled" label="下次启动启用跨实例世界广播（所有实例配置需一致）" valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="enabled" label="下次启动启用 AOI" valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="cell_size_milli" label="网格边长（千分之一场景格）" rules={[{ required: true }, { type: 'number', min: 1000, max: 1000000 }]}><InputNumber precision={0} min={1000} max={1000000} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="radius" label="邻域半径（网格数）" rules={[{ required: true }, { type: 'number', min: 1, max: 4 }]}><InputNumber precision={0} min={1} max={4} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="reason" label="操作原因" rules={[{ required: true, whitespace: true, max: 500 }]}><Input.TextArea rows={3} maxLength={500} /></Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
