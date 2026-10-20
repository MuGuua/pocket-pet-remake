import { Alert, Button, Card, Descriptions, Form, Input, InputNumber, Modal, Select, Space, Table, Typography, message } from 'antd';
import { useState } from 'react';
import { GrantableItemSelect } from '../../components/GrantableItemSelect';
import { grantAdminRewards } from '../../services/reward';
import type { AdminRewardEntry, AdminRewardEntryType, AdminRewardGrantResult } from '../../types/reward';

interface PlayerRewardGrantSectionProps {
  playerId: number;
  playerName: string;
  onDataChanged?: () => void;
}

/** 表单内一行补发条目；itemName 仅用于确认弹窗展示，不提交服务端。 */
interface RewardRow {
  key: number;
  type: Exclude<AdminRewardEntryType, 'pet'>;
  itemId?: number;
  itemName?: string;
  amount?: number;
}

/** 补发类型选项；宠物有独立的模板发放入口（宠物页签），这里不重复提供。 */
const REWARD_TYPE_OPTIONS: Array<{ value: RewardRow['type']; label: string }> = [
  { value: 'item', label: '物品' },
  { value: 'gold', label: '货币（总铜币）' },
  { value: 'exp', label: '人物经验' },
];

/** 单次补发条目上限，与服务端 adminRewardMaxEntries 一致。 */
const MAX_ROWS = 20;
/** 单条数量/数值上限，与服务端 adminRewardMaxAmount 一致；前端提前拦截，服务端仍会再次校验。 */
const MAX_AMOUNT = 100_000_000;

let rowKeySeed = 1;
const createRow = (): RewardRow => ({ key: rowKeySeed++, type: 'item', amount: 1 });

/** 把表单行转换为服务端 reward.Entry；不完整的行返回 null。 */
function toEntry(row: RewardRow): AdminRewardEntry | null {
  const amount = Math.trunc(row.amount ?? 0);
  if (amount <= 0) return null;
  if (row.type === 'item') {
    if (!row.itemId) return null;
    return { type: 'item', item_id: row.itemId, item_name: row.itemName ?? '', count: amount };
  }
  return { type: row.type, value: amount };
}

/** 生成确认弹窗里每条奖励的中文描述，所有数值展示为整数。 */
function describeRow(row: RewardRow): string {
  const amount = Math.trunc(row.amount ?? 0);
  if (row.type === 'item') return `物品「${row.itemName || `#${row.itemId}`}」× ${amount}`;
  if (row.type === 'gold') return `货币 ${amount} 铜`;
  return `人物经验 ${amount}`;
}

/**
 * 玩家补发奖励面板：走服务端统一发奖入口 POST /api/admin/rewards。
 * 高风险操作保护：必填操作原因 + 提交前二次确认（列出影响范围）+ 服务端白名单与数值上限 + 审计日志。
 */
export function PlayerRewardGrantSection({ playerId, playerName, onDataChanged }: PlayerRewardGrantSectionProps) {
  const [rows, setRows] = useState<RewardRow[]>([createRow()]);
  const [reason, setReason] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [lastResult, setLastResult] = useState<AdminRewardGrantResult | null>(null);

  const updateRow = (key: number, patch: Partial<RewardRow>) => {
    setRows((current) => current.map((row) => (row.key === key ? { ...row, ...patch } : row)));
  };

  const handleSubmit = () => {
    const trimmedReason = reason.trim();
    if (!trimmedReason) {
      message.warning('请填写操作原因，便于后续审计');
      return;
    }
    const validRows = rows.filter((row) => toEntry(row) != null);
    if (validRows.length === 0) {
      message.warning('请至少填写一条完整的奖励');
      return;
    }
    if (validRows.length !== rows.length) {
      message.warning('存在未填写完整的奖励行，请补全或删除');
      return;
    }
    Modal.confirm({
      title: `确认向「${playerName}」（ID ${playerId}）补发以下奖励？`,
      width: 520,
      okText: '确认补发',
      cancelText: '取消',
      content: (
        <Space direction="vertical" style={{ width: '100%' }}>
          <ul style={{ paddingLeft: 20, margin: 0 }}>
            {validRows.map((row) => <li key={row.key}>{describeRow(row)}</li>)}
          </ul>
          <Typography.Text type="secondary">操作原因：{trimmedReason}</Typography.Text>
          <Alert type="warning" showIcon message="补发立即到账且不可自动撤回，请核对玩家与数量。" />
        </Space>
      ),
      onOk: async () => {
        setSubmitting(true);
        try {
          const result = await grantAdminRewards({
            player_id: playerId,
            reason: trimmedReason,
            rewards: validRows.map((row) => toEntry(row) as AdminRewardEntry),
          });
          setLastResult(result);
          setRows([createRow()]);
          setReason('');
          message.success(`补发完成，实际到账 ${result.granted.length} 条`);
          onDataChanged?.();
        } catch (error) {
          message.error(error instanceof Error ? error.message : '补发失败');
        } finally {
          setSubmitting(false);
        }
      },
    });
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Card size="small" title={`补发奖励 · ${playerName}`}>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          通过服务端统一发奖入口发放，物品进入背包、货币写入钱包流水，并记录后台操作审计。宠物请在「宠物」页签按模板发放。
        </Typography.Paragraph>
        <Table<RewardRow>
          size="small"
          rowKey="key"
          pagination={false}
          dataSource={rows}
          columns={[
            {
              title: '类型',
              width: 160,
              render: (_, row) => (
                <Select
                  value={row.type}
                  options={REWARD_TYPE_OPTIONS}
                  style={{ width: '100%' }}
                  onChange={(type) => updateRow(row.key, { type, itemId: undefined, itemName: undefined })}
                />
              ),
            },
            {
              title: '内容',
              render: (_, row) => (row.type === 'item' ? (
                <GrantableItemSelect
                  value={row.itemId}
                  onChange={(itemId) => updateRow(row.key, { itemId })}
                  onItemChange={(item) => updateRow(row.key, { itemName: item?.item_name })}
                  showCategoryFilter
                />
              ) : (
                <Typography.Text type="secondary">{row.type === 'gold' ? '按总铜币真值发放' : '直接累加人物经验，可能触发升级'}</Typography.Text>
              )),
            },
            {
              title: '数量',
              width: 160,
              render: (_, row) => (
                <InputNumber
                  min={1}
                  max={MAX_AMOUNT}
                  precision={0}
                  value={row.amount}
                  style={{ width: '100%' }}
                  onChange={(amount) => updateRow(row.key, { amount: amount ?? undefined })}
                />
              ),
            },
            {
              title: '操作',
              width: 72,
              render: (_, row) => (
                <Button type="link" danger disabled={rows.length <= 1} onClick={() => setRows((current) => current.filter((item) => item.key !== row.key))}>
                  删除
                </Button>
              ),
            },
          ]}
        />
        <Button style={{ marginTop: 8 }} disabled={rows.length >= MAX_ROWS} onClick={() => setRows((current) => [...current, createRow()])}>
          添加一行（最多 {MAX_ROWS} 条）
        </Button>
        <Form layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item label="操作原因" required>
            <Input.TextArea rows={3} value={reason} maxLength={200} showCount placeholder="例如：10月20日停服补偿 / 工单 #1234 道具丢失补发" onChange={(event) => setReason(event.target.value)} />
          </Form.Item>
          <Button type="primary" loading={submitting} onClick={handleSubmit}>
            补发
          </Button>
        </Form>
      </Card>

      {lastResult ? (
        <Card size="small" title="最近一次补发结果（服务端返回值）">
          <Descriptions bordered size="small" column={1}>
            <Descriptions.Item label="玩家ID">{lastResult.player_id}</Descriptions.Item>
            <Descriptions.Item label="操作原因">{lastResult.reason}</Descriptions.Item>
            <Descriptions.Item label="实际到账">
              {lastResult.granted.length === 0 ? '无' : lastResult.granted.map((entry, index) => (
                <div key={index}>
                  {entry.type === 'item' ? `物品「${entry.item_name || `#${entry.item_id}`}」× ${Math.trunc(entry.count ?? 0)}` : `${entry.type === 'gold' ? '货币' : '人物经验'} ${Math.trunc(entry.value ?? 0)}`}
                </div>
              ))}
            </Descriptions.Item>
            {lastResult.wallet ? (
              <Descriptions.Item label="补发后钱包（总铜币）">{Math.trunc(lastResult.wallet.total_copper)}</Descriptions.Item>
            ) : null}
          </Descriptions>
        </Card>
      ) : null}
    </Space>
  );
}
