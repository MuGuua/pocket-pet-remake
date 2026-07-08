import {
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Space,
  Spin,
  Typography,
  message,
} from 'antd';
import { useEffect, useState } from 'react';
import { adjustAdminWallet, fetchAdminWalletDetail } from '../../services/wallet';
import type { AdminAdjustWalletPayload, AdminWalletDetail } from '../../types/wallet';
import { formatDateTime } from '../../utils/formatDateTime';

interface WalletFormValues extends AdminAdjustWalletPayload {}

interface PlayerWalletSectionProps {
  /** 当前查看或编辑的玩家 ID */
  playerId: number;
  /** 玩家昵称，用于区块标题 */
  playerName: string;
  /** 钱包发生变更后的回调，用于通知父级刷新玩家详情。 */
  onDataChanged?: () => void | Promise<void>;
}

// 玩家详情/编辑页内的钱包区块：展示服务端权威钱包快照，并支持按总铜币调账。
export function PlayerWalletSection({ playerId, playerName, onDataChanged }: PlayerWalletSectionProps) {
  const [editorForm] = Form.useForm<WalletFormValues>();
  const [loading, setLoading] = useState<boolean>(false);
  const [detail, setDetail] = useState<AdminWalletDetail | null>(null);
  const [editorOpen, setEditorOpen] = useState<boolean>(false);
  const [saving, setSaving] = useState<boolean>(false);

  useEffect(() => {
    void loadWallet();
  }, [playerId]);

  async function loadWallet(): Promise<void> {
    setLoading(true);
    try {
      setDetail(await fetchAdminWalletDetail(playerId));
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载玩家钱包失败');
      setDetail(null);
    } finally {
      setLoading(false);
    }
  }

  function handleOpenEditor(): void {
    editorForm.setFieldsValue({ change_total_copper: 1000, reason: '运营手动调整' });
    setEditorOpen(true);
  }

  async function handleSubmit(values: WalletFormValues): Promise<void> {
    setSaving(true);
    try {
      await adjustAdminWallet(playerId, values);
      message.success('钱包调整成功');
      setEditorOpen(false);
      await loadWallet();
      if (onDataChanged) {
        await onDataChanged();
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '钱包调整失败');
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <Card
        size="small"
        title="钱包信息"
        extra={(
          <Button type="primary" size="small" onClick={handleOpenEditor}>
            调账
          </Button>
        )}
      >
        {loading && detail == null ? (
          <div style={{ minHeight: 80, display: 'grid', placeItems: 'center' }}>
            <Spin tip="正在加载钱包..." />
          </div>
        ) : detail ? (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="玩家">{`${playerName}（ID ${detail.player_id}）`}</Descriptions.Item>
            <Descriptions.Item label="数据版本">{detail.version}</Descriptions.Item>
            <Descriptions.Item label="总铜币（真值）">{detail.wallet.total_copper}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{formatDateTime(detail.updated_at)}</Descriptions.Item>
            <Descriptions.Item label="金币（展示）">{detail.wallet.gold}</Descriptions.Item>
            <Descriptions.Item label="银币（展示）">{detail.wallet.silver}</Descriptions.Item>
            <Descriptions.Item label="铜币（展示）">{detail.wallet.copper}</Descriptions.Item>
            <Descriptions.Item label="说明" span={2}>
              <Typography.Text type="secondary">
                调账只修改总铜币，金银铜为展示拆分；请勿直接改展示字段。
              </Typography.Text>
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <Typography.Text type="secondary">暂无钱包数据</Typography.Text>
        )}
      </Card>

      <Modal
        title={`调账 · ${playerName}`}
        open={editorOpen}
        onCancel={() => setEditorOpen(false)}
        onOk={() => editorForm.submit()}
        confirmLoading={saving}
        destroyOnClose
        okText="确认调账"
        cancelText="取消"
      >
        <Form form={editorForm} layout="vertical" onFinish={(values) => void handleSubmit(values)}>
          <Form.Item
            label="变更总铜币"
            name="change_total_copper"
            rules={[{ required: true, message: '请输入增减铜币' }]}
            extra="正数为增加，负数为扣减"
          >
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="操作原因" name="reason" rules={[{ required: true, message: '请输入操作原因' }]}>
            <Input.TextArea rows={3} placeholder="请填写本次调账原因，便于后续审计" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
