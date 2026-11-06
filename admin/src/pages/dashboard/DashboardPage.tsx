import { Button, Card, Col, Input, Row, Space, Spin, Statistic, Table, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { fetchAdminDashboardOverview, fetchRuntimeMetrics } from '../../services/dashboard';
import type { AdminDashboardOverview, RuntimeMetric } from '../../types/dashboard';

const REFRESH_INTERVAL_MS = 30000;

// 中文名称和稳定指标标识共用映射，检索与表格展示保持一致。
function metricLabel(key: string): string {
  const names: Record<string, string> = {
    'ws.handler': '服务端请求处理', 'ws.queue_wait': '发送队列等待', 'ws.write': '网络写入',
    'ws.queue_depth_sample': '单连接队列深度采样', 'ws.backpressure_close': '背压关闭连接',
    'world.relay_publish': '跨实例事件发布', 'world.relay_failure': '跨实例同步恢复事件',
    'world.move': '移动处理（含拒绝）', 'world.move_rejected': '移动拒绝', 'world.move_authority_lost': '移动权威状态缺失', 'world.aoi_recipients': 'AOI 广播接收人数', 'world.persist': '移动位置写回', 'world.broadcast_recipients': '单次广播接收人数',
    'sql.SELECT': '数据库查询', 'sql.INSERT': '数据库插入', 'sql.UPDATE': '数据库更新', 'sql.DELETE': '数据库删除',
    'sql.BEGIN': '数据库事务开始', 'sql.COMMIT': '数据库事务提交', 'sql.ROLLBACK': '数据库事务回滚', 'sql.OTHER': '数据库其他操作',
    'redis.GET': 'Redis 读取', 'redis.GETDEL': 'Redis 读取并删除', 'redis.SET': 'Redis 写入', 'redis.EVAL': 'Redis 原子脚本', 'redis.DEL': 'Redis 删除',
  };
  if (key.startsWith('client.')) {
    const phase = key.split('.')[1];
    const label = ({ total: '客户端操作总耗时', response: '客户端响应等待', dispatch: '客户端回包分发', timeout: '客户端诊断超时（超过30秒）' } as Record<string, string>)[phase] ?? '客户端上报';
    return `${label}（命令 ${key.slice(key.lastIndexOf('.') + 1)}）`;
  }
  if (key.startsWith('ws.cmd.')) return `命令 ${key.slice(7)} 处理`;
  if (key.startsWith('world.scene_online.')) return `场景 ${key.slice(19)} 索引人数`;
  return names[key] ?? key;
}

// 控制台首页展示服务端权威统计：在线人数、日活与账号规模。
export function DashboardPage() {
  const [loading, setLoading] = useState<boolean>(true);
  const [metricQuery, setMetricQuery] = useState<string>('');
  const [metricsUpdatedAt, setMetricsUpdatedAt] = useState<string>('-');
  const [metrics, setMetrics] = useState<RuntimeMetric[]>([]);
  const [overview, setOverview] = useState<AdminDashboardOverview | null>(null);

  const loadOverview = useCallback(async (showLoading: boolean) => {
    if (showLoading) {
      setLoading(true);
    }
    try {
      const [overviewResult, metricsResult] = await Promise.allSettled([fetchAdminDashboardOverview(), fetchRuntimeMetrics()]);
      if (overviewResult.status === 'fulfilled') setOverview(overviewResult.value);
      else message.error('加载业务概览失败');
      if (metricsResult.status === 'fulfilled') {
        setMetrics(metricsResult.value);
        setMetricsUpdatedAt(new Date().toLocaleString('zh-CN'));
      }
      else message.error('加载运行指标失败');
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
              <Statistic title="当前进程在线玩家" value={overview?.online_player_count ?? 0} suffix="人" />
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
      <Card title="当前服务进程运行指标">
        <Typography.Paragraph type="secondary">
          计数与失败为进程启动以来累计值；P50/P95 来自最近最多 256 次采样。服务端微秒指标不是公网操作延迟；“客户端”指标来自心跳上报的原始操作耗时样本，包含网络与回包处理，不能用于玩法判定。客户端分位仅包含已完成且不超过30秒的请求，超时单列事件，不终止业务请求。队列深度和场景人数为单连接／单场景采样，非全服总数；启用跨实例广播后，场景索引包含远端展示节点。
        </Typography.Paragraph>
        <Space wrap style={{ marginBottom: 12 }}>
          <Input.Search allowClear placeholder="搜索指标名称或标识" value={metricQuery} onChange={(event) => setMetricQuery(event.target.value)} style={{ width: 260 }} />
          <Typography.Text type="secondary">指标更新时间：{metricsUpdatedAt}</Typography.Text>
        </Space>
        <Table<RuntimeMetric> rowKey="key" dataSource={metrics.filter((metric) => `${metric.key} ${metricLabel(metric.key)}`.toLowerCase().includes(metricQuery.trim().toLowerCase()))} size="small" scroll={{ x: 900 }} pagination={{ pageSize: 10 }} columns={[
          { title: '指标', dataIndex: 'key', render: (key: string) => {
            return <Space direction="vertical" size={0}><span>{metricLabel(key)}</span><Typography.Text type="secondary" style={{ fontSize: 12 }}>{key}</Typography.Text></Space>;
          } },
          { title: '单位', dataIndex: 'unit', filters: [{ text: '耗时', value: 'us' }, { text: '帧', value: 'frames' }, { text: '人数', value: 'players' }, { text: '事件', value: 'events' }], onFilter: (value, record) => record.unit === value, render: (unit: string) => ({ us: '微秒', frames: '帧', players: '人', events: '次' }[unit] ?? unit) },
          { title: '累计次数', dataIndex: 'count' },
          { title: '失败／拒绝', dataIndex: 'failures' },
          { title: '近期样本数', dataIndex: 'recent_samples' },
          { title: '最近值', dataIndex: 'last' },
          { title: 'P50', dataIndex: 'p50' },
          { title: 'P95', dataIndex: 'p95' },
          { title: '历史最大', dataIndex: 'max' },
        ]} />
      </Card>
    </Space>
  );
}
