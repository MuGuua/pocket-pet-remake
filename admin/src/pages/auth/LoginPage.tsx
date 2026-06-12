import { Button, Card, Form, Input, Space, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { fetchAdminProfile, loginAdmin, logoutAdmin } from '../../services/auth';
import { getAdminToken } from '../../services/http';

interface LoginFormValues {
  account: string;
  password: string;
}

// 登录页直接接真实管理员接口；只有后台独立管理员账号才能进入运营页面。
export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const [submitting, setSubmitting] = useState(false);
  const [checking, setChecking] = useState(true);
  const redirectTo = (location.state as { from?: string } | null)?.from ?? '/dashboard';

  useEffect(() => {
    let cancelled = false;
    const token = getAdminToken();
    if (!token) {
      setChecking(false);
      return;
    }
    fetchAdminProfile()
      .then(() => {
        if (!cancelled) {
          navigate(redirectTo, { replace: true });
        }
      })
      .catch(() => {
        if (!cancelled) {
          logoutAdmin();
          setChecking(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [navigate, redirectTo]);

  async function handleFinish(values: LoginFormValues) {
    setSubmitting(true);
    try {
      await loginAdmin(values.account, values.password);
      message.success('登录成功');
      navigate(redirectTo, { replace: true });
    } catch (error) {
      const reason = error instanceof Error ? error.message : '登录失败';
      message.error(reason);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'grid',
        placeItems: 'center',
        background: 'linear-gradient(135deg, #f4efe6 0%, #dce7df 100%)',
      }}
    >
      <Card style={{ width: 420 }}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <div>
            <Typography.Title level={3}>后台登录</Typography.Title>
            <Typography.Text type="secondary">
              使用独立管理员账号登录，不复用游戏玩家账号体系。
            </Typography.Text>
          </div>

          <Form<LoginFormValues>
            layout="vertical"
            initialValues={{
              // 登录页默认带出迁移脚本里初始化的管理员账号，方便本地联调时直接进入后台。
              account: 'admin',
              password: 'admin123',
            }}
            onFinish={handleFinish}
          >
            <Form.Item label="管理员账号" name="account" rules={[{ required: true, message: '请输入管理员账号' }]}>
              <Input placeholder="请输入管理员账号" autoComplete="username" disabled={checking || submitting} />
            </Form.Item>
            <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password placeholder="请输入密码" autoComplete="current-password" disabled={checking || submitting} />
            </Form.Item>
            <Button type="primary" htmlType="submit" block loading={submitting || checking}>
              登录后台
            </Button>
          </Form>
        </Space>
      </Card>
    </div>
  );
}
