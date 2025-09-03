import { useEffect, useState } from 'react';
import { Form, Input, Button, Alert, Space, Divider, Typography } from 'antd';

export default function UserSettings() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [profile, setProfile] = useState<{ email?: string; display_name?: string }>({});
  const [form] = Form.useForm();

  async function load() {
    try {
      setError(null);
      const res = await fetch('/api/me/profile', { credentials: 'include' });
      if (!res.ok) throw new Error(await res.text());
      const p = await res.json();
      const next = { email: p.email || '', display_name: p.display_name || '' };
      setProfile(next);
      form.setFieldsValue(next);
    } catch (e: any) {
      setError(e?.message || 'Failed to load profile');
    }
  }

  useEffect(() => { load(); }, []);

  async function saveProfile(values: any) {
    setLoading(true);
    setOk(null); setError(null);
    try {
      const res = await fetch('/api/me/profile', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(values),
      });
      if (!res.ok) throw new Error(await res.text());
      setOk('Profile updated');
    } catch (e: any) {
      setError(e?.message || 'Failed to update profile');
    } finally {
      setLoading(false);
    }
  }

  async function changePassword(values: { old_password: string; new_password: string }) {
    setLoading(true);
    setOk(null); setError(null);
    try {
      const res = await fetch('/api/me/password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(values),
      });
      if (res.status === 409) {
        setError('Password is managed by your identity provider.');
        return;
      }
      if (!res.ok) throw new Error(await res.text());
      setOk('Password changed');
    } catch (e: any) {
      setError(e?.message || 'Failed to change password');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <Typography.Title level={3}>User Settings</Typography.Title>
      {error && <Alert type="error" message={error} style={{ marginBottom: 12 }} />}
      {ok && <Alert type="success" message={ok} style={{ marginBottom: 12 }} />}

      <Space direction="vertical" style={{ width: 420 }}>
        <Form layout="vertical" form={form} initialValues={profile} onFinish={saveProfile} onValuesChange={(_, all) => setProfile(all)}>
          <Form.Item label="Display Name" name="display_name">
            <Input />
          </Form.Item>
          <Form.Item label="Email" name="email">
            <Input type="email" />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={loading}>Save Profile</Button>
        </Form>

        <Divider />

        <Form layout="vertical" onFinish={changePassword}>
          <Form.Item label="Current Password" name="old_password" rules={[{ required: true }]}> <Input.Password /> </Form.Item>
          <Form.Item label="New Password" name="new_password" rules={[{ required: true, min: 8 }]}> <Input.Password /> </Form.Item>
          <Button htmlType="submit" loading={loading}>Change Password</Button>
        </Form>
      </Space>
    </div>
  );
}
