import { Button, Layout, Menu, Space, Typography } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { logoutAdmin } from '../services/auth';
import type { AdminSessionProfile } from '../types/admin';

const { Header, Sider, Content } = Layout;

const menuItems = [
  { key: '/dashboard', label: '控制台' },
  { key: '/players', label: '玩家管理' },
  { key: '/pets', label: '宠物管理' },
  { key: '/skill-definitions', label: '系统技能管理' },
  { key: '/monster-definitions', label: '系统怪物管理' },
  { key: '/monster-encounters', label: '怪物遭遇配置' },
  { key: '/items', label: '物品模板' },
  { key: '/quests', label: '任务管理' },
  { key: '/npcs', label: '地图NPC配置' },
];

interface AdminLayoutProps {
  profile: AdminSessionProfile;
}

// 后台通用布局：左侧菜单 + 顶栏 + 内容区。后续所有页面都应复用这层壳，不再各自实现导航。
export function AdminLayout({ profile }: AdminLayoutProps) {
  const navigate = useNavigate();
  const location = useLocation();

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} theme="light">
        <div style={{ padding: 20, fontWeight: 700 }}>口袋宠物运营后台</div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            background: '#fffaf2',
            borderBottom: '1px solid #e7dcc9',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Typography.Title level={4} style={{ margin: 0 }}>
            运营后台
          </Typography.Title>
          <Space>
            <Typography.Text type="secondary">{profile.display_name}</Typography.Text>
            <Button
              onClick={() => {
                logoutAdmin();
                navigate('/login', { replace: true });
              }}
            >
              退出登录
            </Button>
          </Space>
        </Header>
        <Content style={{ padding: 24 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
