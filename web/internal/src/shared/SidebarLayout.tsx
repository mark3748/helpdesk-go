import { Layout, Menu, Dropdown, Typography, Avatar, Space } from 'antd';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useMe } from './auth';
import { useQueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';

const { Sider, Content, Header } = Layout;

type Role = 'agent' | 'manager' | 'admin';
type NavChild = { key: string; label: string; path: string };
type NavGroup = { key: string; label: string; role: Role; children: NavChild[] };

const groups: NavGroup[] = [
  {
    key: 'group-agent',
    label: 'Agent',
    role: 'agent',
    children: [
      { key: 'tickets', label: 'Tickets', path: '/tickets' },
      { key: 'metrics', label: 'Metrics', path: '/metrics' },
    ],
  },
  {
    key: 'group-manager',
    label: 'Manager',
    role: 'manager',
    children: [
      { key: 'manager', label: 'Queue Manager', path: '/manager' },
      { key: 'manager-analytics', label: 'Analytics', path: '/manager/analytics' },
    ],
  },
  {
    key: 'group-admin',
    label: 'Admin',
    role: 'admin',
    children: [
      { key: 'settings', label: 'Admin Settings', path: '/settings' },
      { key: 'settings-mail', label: 'Mail Settings', path: '/settings/mail' },
      { key: 'settings-oidc', label: 'OIDC Settings', path: '/settings/oidc' },
      { key: 'settings-storage', label: 'Storage Settings', path: '/settings/storage' },
      { key: 'settings-users', label: 'User Roles', path: '/settings/users' },
    ],
  },
];

export function SidebarLayout({ children }: { children?: ReactNode }) {
  const { data: me } = useMe();
  const loc = useLocation();
  const nav = useNavigate();
  const qc = useQueryClient();
  const roles = me?.roles || [];
  const isSuper = roles.includes('admin');
  const visibleGroups = groups.filter((g) => isSuper || roles.includes(g.role));
  const menuItems = visibleGroups.map((g) => ({
    key: g.key,
    label: g.label,
    children: g.children.map((c) => ({ key: c.key, label: <Link to={c.path}>{c.label}</Link> })),
  }));

  const parts = loc.pathname.split('/').filter(Boolean);
  // Prefer a more specific key when on sub-pages under /settings
  const selected = parts.length > 1 && parts[0] === 'settings'
    ? `settings-${parts[1]}`
    : (parts[0] || 'tickets');
  const defaultOpenKeys = visibleGroups
    .filter((g) => g.children.some((c) => c.key === selected))
    .map((g) => g.key);

  async function doLogout() {
    try {
      await fetch('/api/logout', { method: 'POST', credentials: 'include' });
    } catch {}
    await qc.invalidateQueries({ queryKey: ['me'] });
    nav('/login', { replace: true });
  }

  const userMenuItems = [
    { key: 'settings', label: 'Settings' },
    { key: 'logout', label: 'Logout' },
  ];

  const displayName = me?.display_name || me?.email || me?.id || 'Account';
  const rolesText = me?.roles && me.roles.length > 0 ? `Roles: ${me.roles.join(', ')}` : '';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider>
        <Menu theme="dark" selectedKeys={[selected]} defaultOpenKeys={defaultOpenKeys} items={menuItems} />
      </Sider>
      <Layout>
        <Header style={{ background: '#fff', padding: '0 16px', display: 'flex', alignItems: 'center', borderBottom: '1px solid #f0f0f0' }}>
          {/* Left side could host app name/logo later */}
          <div style={{ fontWeight: 600 }}>Helpdesk</div>

          {/* Right-aligned user area */}
          <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 12 }}>
            {rolesText && <Typography.Text type="secondary">{rolesText}</Typography.Text>}
            <Dropdown
              menu={{
                items: userMenuItems,
                onClick: ({ key }) => {
                  if (key === 'logout') doLogout();
                  if (key === 'settings') nav('/me/settings');
                },
              }}
              trigger={["click"]}
            >
              <Space style={{ cursor: 'pointer' }}>
                <Avatar size={32}>{String(displayName).trim().charAt(0).toUpperCase()}</Avatar>
                <Typography.Text>{displayName}</Typography.Text>
              </Space>
            </Dropdown>
          </div>
        </Header>
        <Content style={{ padding: 16 }}>
          {children || <Outlet />}
        </Content>
      </Layout>
    </Layout>
  );
}
