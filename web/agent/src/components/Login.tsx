import { useState } from 'react';
import { login } from '../api';
import { Button, Card, Form, Input, Typography, message } from 'antd';

interface Props {
  onLoggedIn(): void;
}

export default function Login({ onLoggedIn }: Props) {
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  async function onFinish(values: { username: string; password: string }) {
    setLoading(true);
    const ok = await login(values.username, values.password);
    setLoading(false);
    if (ok) {
      onLoggedIn();
    } else {
      message.error('Login failed');
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Card style={{ width: 360 }}>
        <Typography.Title level={3} style={{ textAlign: 'center', marginBottom: 24 }}>
          Helpdesk Agent
        </Typography.Title>
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item name="username" label="Username" rules={[{ required: true }]}> 
            <Input autoFocus placeholder="admin" />
          </Form.Item>
          <Form.Item name="password" label="Password" rules={[{ required: true }]}> 
            <Input.Password placeholder="admin" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading}>
              Sign in
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
