import { Button, Card, Col, Row, Space, Spin, Statistic, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { fetchAdminDashboardOverview } from '../../services/dashboard';
import type { AdminDashboardOverview } from '../../types/dashboard';

const REFRESH_INTERVAL_MS = 30000;

// 控制台首页展示服务端权威统计：在线人数、日活与账号规模。
export function DashboardPage() {
  const [loading, setLoading] = useState<boolean>(true);
  const [overview, setOverview] = useState<AdminDashboardOverview | null>(null);

  const loadOverview = useCallback(async (showLoading: boolean) => {
    if (showLoading) {
      setLoading(true);
    }
    try {
      setOverview(await fetchAdminDashboardOverview());
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载控制台数据失败');
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void loadOverview(true);
    const timer = window.setInterval(() => {
      void loadOverview(false);
    }, REFRESH_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [loadOverview]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Space style={{ width: '100%', justifyContent: 'space-between' }} wrap>
          <div>
            <Typography.Title level={4} style={{ margin: 0 }}>业务概览</Typography.Title>
            <Typography.Text type="secondary">
              统计日：{overview?.stat_date ?? '-'}（{overview?.timezone ?? 'Asia/Shanghai'}）
            </Typography.Text>
          </div>
          <Button onClick={() => void loadOverview(true)} loading={loading}>
            刷新
          </Button>
        </Space>
      </Card>

      <Spin spinning={loading && overview == null}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} xl={6}>
            <Card>
              <Statistic title="当前在线玩家" value={overview?.online_player_count ?? 0} suffix="人" />
              <Typography.Text type="secondary">基于有效 WebSocket 会话统计</Typography.Text>
            </Card>
          </Col>
          <Col xs={24} sm={12} xl={6}>
            <Card>
              <Statistic title="今日日活" value={overview?.daily_active_accounts ?? 0} suffix="账号" />
              <Typography.Text type="secondary">按账号最近登录时间统计</Typography.Text>
            </Card>
          </Col>
          <Col xs={24} sm={12} xl={6}>
            <Card>
              <Statistic title="启用账号总数" value={overview?.total_accounts ?? 0} suffix="个" />
              <Typography.Text type="secondary">启用状态的账号数量</Typography.Text>
            </Card>
          </Col>
          <Col xs={24} sm={12} xl={6}>
            <Card>
              <Statistic title="启用玩家总数" value={overview?.total_players ?? 0} suffix="人" />
              <Typography.Text type="secondary">启用状态的角色数量</Typography.Text>
            </Card>
          </Col>
          <Col xs={24} sm={12} xl={6}>
            <Card>
              <Statistic title="今日新增账号" value={overview?.new_accounts_today ?? 0} suffix="个" />
              <Typography.Text type="secondary">按账号 created_at 统计</Typography.Text>
            </Card>
          </Col>
        </Row>
      </Spin>
    </Space>
  );
}
