import { Layout, Menu, Dropdown, Typography, Avatar, Space, Input, Badge, Button } from 'antd';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useMe } from './auth';
import { useQueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import {
  AppstoreOutlined,
  UserOutlined,
  ContainerOutlined,
  FileTextOutlined,
  SettingOutlined,
  BellOutlined,
  SearchOutlined,
  GlobalOutlined,
  DownOutlined,
  MenuFoldOutlined
} from '@ant-design/icons';

const { Sider, Content, Header } = Layout;

type Role = 'agent' | 'manager' | 'admin';
type NavChild = { key: string; label: string; path: string };
type NavGroup = { key: string; label: string; role: Role; icon?: ReactNode; children?: NavChild[]; path?: string };

// Flattened structure for the new design while keeping role-based logic
const navItems: NavGroup[] = [
  { key: 'dashboard', label: 'Dashboard', role: 'agent', icon: <AppstoreOutlined />, path: '/tickets' }, // Mapping Dashboard to Tickets for now
  { key: 'profile', label: 'Profile', role: 'agent', icon: <UserOutlined />, path: '/me/settings' },
  {
    key: 'tickets-group',
    label: 'Tickets',
    role: 'agent',
    icon: <ContainerOutlined />,
    children: [
      { key: 'tickets-all', label: 'All Tickets', path: '/tickets' },
      { key: 'tickets-active', label: 'Active Tickets', path: '/tickets?status=active' }, // Placeholder paths
      { key: 'tickets-assigned', label: 'Assigned Tickets', path: '/tickets?assignee=me' },
      { key: 'tickets-closed', label: 'Closed Tickets', path: '/tickets?status=closed' },
    ],
  },
  { key: 'categories', label: 'Categories', role: 'admin', icon: <AppstoreOutlined />, path: '/assets/categories' },
  { key: 'customers', label: 'Customers', role: 'agent', icon: <UserOutlined />, path: '/users' },
  { key: 'notifications', label: 'Notifications', role: 'agent', icon: <BellOutlined />, path: '/notifications' },
  { key: 'settings', label: 'Settings', role: 'admin', icon: <SettingOutlined />, path: '/settings' },
  { key: 'reports', label: 'Reports', role: 'manager', icon: <FileTextOutlined />, path: '/manager/analytics' },
];

export function SidebarLayout({ children }: { children?: ReactNode }) {
  const { data: me } = useMe();
  const loc = useLocation();
  const nav = useNavigate();
  const qc = useQueryClient();
  const roles = me?.roles || [];
  const isSuper = roles.includes('admin');

  // Filter items based on role
  const visibleItems = navItems.filter((g) => isSuper || roles.includes(g.role));

  const menuItems = visibleItems.map((g) => {
    if (g.children) {
      return {
        key: g.key,
        icon: g.icon,
        label: g.label,
        children: g.children.map((c) => ({ key: c.key, label: <Link to={c.path}>{c.label}</Link> })),
      };
    }
    return {
      key: g.key,
      icon: g.icon,
      label: <Link to={g.path || '#'}>{g.label}</Link>,
    };
  });

  const parts = loc.pathname.split('/').filter(Boolean);
  const selected = parts.length > 0 ? parts[0] : 'dashboard'; // Naive selection logic
  // Better selection logic could be implemented

  async function doLogout() {
    try {
      await fetch('/api/logout', { method: 'POST', credentials: 'include' });
    } catch {
      // Logout request failed, continue anyway
    }
    await qc.invalidateQueries({ queryKey: ['me'] });
    nav('/login', { replace: true });
  }

  const userMenuItems = [
    { key: 'settings', label: 'Settings' },
    { key: 'logout', label: 'Logout', danger: true },
  ];

  const displayName = me?.display_name || me?.email || 'User';

  return (
    <Layout style={{ minHeight: '100vh', background: 'var(--bg-color)' }}>
      <Sider
        width={260}
        theme="light"
        style={{
          background: '#fff',
          borderRight: '1px solid #f0f0f0',
          position: 'fixed',
          height: '100vh',
          left: 0,
          top: 0,
          zIndex: 100,
        }}
      >
        <div style={{ height: 64, display: 'flex', alignItems: 'center', padding: '0 24px' }}>
          <Typography.Title level={4} style={{ margin: 0, color: '#6B4EFF', fontWeight: 800 }}>
            HELPDESK
          </Typography.Title>
        </div>
        <Menu
          mode="inline"
          defaultSelectedKeys={[selected]}
          defaultOpenKeys={['tickets-group']}
          style={{ borderRight: 0 }}
          items={menuItems}
        />
      </Sider>
      <Layout style={{ marginLeft: 260, transition: 'all 0.2s' }}>
        <Header
          style={{
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 24,
            height: 80, // Taller header as per design
          }}
        >
          {/* Left: Search */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Button type="text" icon={<MenuFoldOutlined />} style={{ display: 'none' }} /> {/* Mobile toggle placeholder */}
            <Input
              prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
              placeholder="Search"
              variant="borderless"
              style={{
                background: '#F3F4F8',
                borderRadius: 8,
                width: 300,
                padding: '8px 12px',
              }}
            />
          </div>

          {/* Right: Actions & Profile */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
            <Badge dot color="#6B4EFF">
              <BellOutlined style={{ fontSize: 20, color: '#555' }} />
            </Badge>

            <Space style={{ cursor: 'pointer', color: '#555' }}>
              <GlobalOutlined />
              <span>English</span>
              <DownOutlined style={{ fontSize: 10 }} />
            </Space>

            <Dropdown menu={{ items: userMenuItems, onClick: ({ key }) => key === 'logout' && doLogout() }}>
              <Space style={{ cursor: 'pointer', marginLeft: 12 }}>
                <Avatar src="https://i.pravatar.cc/150?img=32" /> {/* Mock avatar */}
                <div style={{ lineHeight: 1.2 }}>
                  <Typography.Text strong style={{ display: 'block' }}>{displayName}</Typography.Text>
                </div>
                <DownOutlined style={{ fontSize: 10, color: '#999' }} />
              </Space>
            </Dropdown>
          </div>
        </Header>
        <Content style={{ padding: '0 24px 24px', minHeight: 280 }}>
          {children || <Outlet />}

          <div style={{ textAlign: 'center', marginTop: 40, color: '#999', fontSize: 12 }}>
            Copyright © (NH-Rashed)
          </div>
        </Content>
      </Layout>
    </Layout>
  );
}
