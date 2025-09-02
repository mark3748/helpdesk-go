import { Layout, Menu } from 'antd';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { useMe } from './auth';
import type { ReactNode } from 'react';

const { Sider, Content } = Layout;

interface Item {
  key: string;
  label: string;
  path: string;
  roles?: string[];
}

const items: Item[] = [
  { key: 'tickets', label: 'Tickets', path: '/tickets', roles: ['agent', 'admin'] },
  { key: 'settings', label: 'Settings', path: '/settings', roles: ['admin'] },
];

export function SidebarLayout({ children }: { children?: ReactNode }) {
  const { data: me } = useMe();
  const loc = useLocation();
  const menuItems = items
    .filter((i) => !i.roles || i.roles.some((r) => me?.roles?.includes(r)))
    .map((i) => ({ key: i.key, label: <Link to={i.path}>{i.label}</Link> }));

  const selected = loc.pathname.split('/')[1];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider>
        <Menu theme="dark" selectedKeys={[selected]} items={menuItems} />
      </Sider>
      <Layout>
        <Content style={{ padding: 16 }}>
          {children || <Outlet />}
        </Content>
      </Layout>
    </Layout>
  );
}
