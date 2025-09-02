import { Layout, Menu, Dropdown, Typography, Avatar, Space } from 'antd';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useMe } from './auth';
import { useQueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';

const { Sider, Content, Header } = Layout;

interface Item {
  key: string;
  label: string;
  path: string;
  roles?: string[];
}

const items: Item[] = [
  { key: 'tickets', label: 'Tickets', path: '/tickets', roles: ['agent', 'admin'] },
  { key: 'manager', label: 'Manager', path: '/manager', roles: ['manager'] },
  // Expose individual settings pages for clearer navigation
  { key: 'settings-oidc', label: 'OIDC Settings', path: '/settings/oidc', roles: ['admin'] },
  { key: 'settings-storage', label: 'Storage Settings', path: '/settings/storage', roles: ['admin'] },
];

export function SidebarLayout({ children }: { children?: ReactNode }) {
  const { data: me } = useMe();
  const loc = useLocation();
  const nav = useNavigate();
  const qc = useQueryClient();
  const menuItems = items
    .filter((i) => !i.roles || i.roles.some((r) => me?.roles?.includes(r)))
    .map((i) => ({ key: i.key, label: <Link to={i.path}>{i.label}</Link> }));

  const parts = loc.pathname.split('/').filter(Boolean);
  // Prefer a more specific key when on sub-pages under /settings
  const selected = parts.length > 1 && parts[0] === 'settings'
    ? `settings-${parts[1]}`
    : (parts[0] || 'tickets');

  async function doLogout() {
    try {
      await fetch('/api/logout', { method: 'POST', credentials: 'include' });
    } catch {}
    await qc.invalidateQueries({ queryKey: ['me'] });
    nav('/login', { replace: true });
  }

  const userMenuItems = [
    { key: 'logout', label: 'Logout' },
  ];

  const displayName = me?.display_name || me?.email || me?.id || 'Account';
  const rolesText = me?.roles && me.roles.length > 0 ? `Roles: ${me.roles.join(', ')}` : '';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider>
        <Menu theme="dark" selectedKeys={[selected]} items={menuItems} />
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
                onClick: ({ key }) => { if (key === 'logout') doLogout(); },
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
