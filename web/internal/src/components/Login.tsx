import { useState } from 'react';
import { Form, Input, Button, Typography, Alert } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';

export default function Login() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const nav = useNavigate();
  const qc = useQueryClient();

  async function onFinish(values: { username: string; password: string }) {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(values),
      });
      if (!res.ok) throw new Error(await res.text());
      await qc.invalidateQueries({ queryKey: ['me'] });
      nav('/tickets', { replace: true });
    } catch (e: any) {
      setError(e?.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ maxWidth: 360, margin: '10vh auto', background: '#fff', padding: 24, borderRadius: 8 }}>
      <Typography.Title level={4} style={{ textAlign: 'center' }}>Sign in</Typography.Title>
      {error && <Alert type="error" message={error} style={{ marginBottom: 12 }} />}
      <Form layout="vertical" onFinish={onFinish} initialValues={(import.meta as any).env?.DEV ? { username: 'admin', password: 'admin' } : {}}>
        <Form.Item label="Username" name="username" rules={[{ required: true }]}>
          <Input autoFocus />
        </Form.Item>
        <Form.Item label="Password" name="password" rules={[{ required: true }]}>
          <Input.Password />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={loading}>
          Log in
        </Button>
        {(import.meta as any).env?.DEV && (
          <Typography.Paragraph type="secondary" style={{ marginTop: 12, textAlign: 'center' }}>
            Dev default: admin / admin
          </Typography.Paragraph>
        )}
      </Form>
    </div>
  );
}

