import { useEffect, useMemo, useState, useCallback } from 'react';
import { Table, Input, Button, Space, Tag, Typography, message, Select, Form, Card } from 'antd';
import { apiFetch } from '../../shared/api';

type User = {
  id: string;
  external_id?: string;
  username?: string;
  email?: string;
  display_name?: string;
  roles: string[];
};

export default function AdminUsers() {
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [selected, setSelected] = useState<User | null>(null);
  const [roles, setRoles] = useState<string[]>([]);
  const [newRole, setNewRole] = useState('');
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<User[]>(`/users${q ? `?q=${encodeURIComponent(q)}` : ''}`);
      setUsers(data);
      if (selected) {
        const s = data.find((u) => u.id === selected.id);
        setSelected(s || null);
      }
    } catch (e: any) {
      message.error(e?.message || 'Failed to load users');
    } finally {
      setLoading(false);
    }
  }, [q, selected]);

  useEffect(() => { load(); /* initial */ }, [load]);
  useEffect(() => { (async () => { try { setRoles(await apiFetch<string[]>('/roles')); } catch { /* Error loading roles */ } })(); }, []);

  const columns = useMemo(() => ([
    { title: 'Name', dataIndex: 'display_name', key: 'display_name' },
    { title: 'Email', dataIndex: 'email', key: 'email' },
    { title: 'Username', dataIndex: 'username', key: 'username' },
    { title: 'External ID', dataIndex: 'external_id', key: 'external_id' },
  ]), []);

  async function addRole() {
    if (!selected || !newRole) return;
    try {
      await apiFetch(`/users/${selected.id}/roles`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role: newRole }),
      });
      setNewRole('');
      await load();
      message.success('Role added');
    } catch (e: any) {
      message.error(e?.message || 'Failed to add role');
    }
  }

  async function removeRole(role: string) {
    if (!selected) return;
    try {
      await apiFetch(`/users/${selected.id}/roles/${encodeURIComponent(role)}`, { method: 'DELETE' });
      await load();
      message.success('Role removed');
    } catch (e: any) {
      message.error(e?.message || 'Failed to remove role');
    }
  }

  return (
    <Space align="start" style={{ width: '100%' }}>
      <div style={{ flex: 1 }}>
        <Space style={{ marginBottom: 12 }}>
          <Input
            allowClear
            placeholder="Search users (email, username, name)"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            onPressEnter={load}
            style={{ width: 360 }}
          />
          <Button onClick={load} loading={loading}>Search</Button>
        </Space>
        <Table
          rowKey={(u: User) => u.id}
          columns={columns as any}
          dataSource={users}
          loading={loading}
          onRow={(u) => ({ onClick: () => setSelected(u) })}
          pagination={{ pageSize: 10 }}
        />
      </div>
      <div style={{ width: 420 }}>
        <Card size="small" style={{ marginBottom: 12 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>Create Local User</Typography.Title>
          <Form layout="vertical" onFinish={async (vals: any) => {
            setCreating(true);
            try {
              await apiFetch('/users', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(vals),
              });
              message.success('User created');
              (document.querySelector('#create-user-form') as HTMLFormElement)?.reset();
              load();
            } catch (e: any) {
              message.error(e?.message || 'Failed to create user');
            } finally { setCreating(false); }
          }} id="create-user-form">
            <Form.Item label="Username" name="username" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item label="Email" name="email"><Input type="email" /></Form.Item>
            <Form.Item label="Display Name" name="display_name"><Input /></Form.Item>
            <Form.Item label="Password" name="password" rules={[{ required: true, min: 8 }]}><Input.Password /></Form.Item>
            <Button type="primary" htmlType="submit" loading={creating}>Create</Button>
          </Form>
        </Card>

        <Typography.Title level={4}>Roles</Typography.Title>
        {!selected && <div>Select a user to manage roles.</div>}
        {selected && (
          <>
            <div style={{ marginBottom: 8 }}>
              <div style={{ fontWeight: 600 }}>{selected.display_name || selected.email || selected.username || selected.id}</div>
              <div style={{ color: '#888' }}>{selected.email || selected.username || selected.external_id}</div>
            </div>
            <Space wrap>
              {(selected.roles || []).map((r) => (
                <Tag key={r} closable onClose={(e) => { e.preventDefault(); removeRole(r); }}>{r}</Tag>
              ))}
            </Space>
            <Space style={{ marginTop: 12 }}>
              <Select
                showSearch
                placeholder="Select role"
                value={newRole || undefined}
                style={{ minWidth: 180 }}
                onChange={(v) => setNewRole(v)}
                options={roles
                  .filter((r) => !(selected.roles || []).includes(r))
                  .map((r) => ({ value: r, label: r }))}
              />
              <Button onClick={addRole} disabled={!newRole}>Add</Button>
            </Space>
          </>
        )}
      </div>
    </Space>
  );
}
