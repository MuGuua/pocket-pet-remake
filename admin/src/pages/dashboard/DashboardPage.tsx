import { Card, Col, List, Progress, Row, Space, Tag, Typography } from 'antd';

const statCards = [
  { title: '后台鉴权', value: '已接通', tag: 'DONE', percent: 100 },
  { title: '玩家管理', value: 'CRUD 已就绪', tag: 'PHASE B', percent: 100 },
  { title: '宠物管理', value: '待接下一阶段', tag: 'NEXT', percent: 30 },
  { title: '审计日志', value: '建议尽快补齐', tag: 'TODO', percent: 20 },
];

const todoItems = [
  '按相同 CRUD 模式接入宠物管理：列表、详情、新增、编辑、删除',
  '给高风险操作补后台审计日志入库与操作原因字段',
  '给玩家、宠物、任务模块增加批量导出与权限细分',
  '补移动窄屏下的筛选区折叠与表格卡片视图',
];

export function DashboardPage() {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        {statCards.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.title}>
            <Card>
              <Space direction="vertical" size={10} style={{ width: '100%' }}>
                <Space style={{ justifyContent: 'space-between', width: '100%' }}>
                  <Typography.Text type="secondary">{item.title}</Typography.Text>
                  <Tag color={item.percent >= 100 ? 'green' : 'gold'}>{item.tag}</Tag>
                </Space>
                <Typography.Title level={3} style={{ margin: 0 }}>
                  {item.value}
                </Typography.Title>
                <Progress percent={item.percent} showInfo={false} strokeColor="#2f5d50" />
              </Space>
            </Card>
          </Col>
        ))}
        <Col span={24}>
          <Card title="当前待办" extra={<Typography.Text type="secondary">下一步建议先复制玩家 CRUD 模式到宠物与任务模块</Typography.Text>}>
            <List dataSource={todoItems} renderItem={(item) => <List.Item>{item}</List.Item>} />
          </Card>
        </Col>
      </Row>
    </Space>
  );
}
