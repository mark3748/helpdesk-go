import { Layout, Menu, Dropdown, Typography, Avatar, Space, Input, Badge, Button, Segmented } from 'antd';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useMe } from './auth';
import { useQueryClient } from '@tanstack/react-query';
import { type ReactNode, useEffect, useState, useMemo } from 'react';
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
  MenuFoldOutlined,
  TeamOutlined,
  SolutionOutlined,
  DashboardOutlined,
  DatabaseOutlined
} from '@ant-design/icons';

const { Sider, Content, Header } = Layout;

type Role = 'agent' | 'manager' | 'admin';
type NavChild = { key: string; label: string; path: string };
type NavGroup = { key: string; label: string; icon?: ReactNode; children?: NavChild[]; path?: string };

const ROLE_LABELS: Record<Role, string> = {
  agent: 'Agent',
  manager: 'Manager',
  admin: 'Admin',
};

const NAV_CONFIG: Record<Role, NavGroup[]> = {
  agent: [
    { key: 'dashboard', label: 'Dashboard', icon: <DashboardOutlined />, path: '/' },
    { key: 'tickets-dashboard', label: 'Tickets', icon: <AppstoreOutlined />, path: '/tickets' },
    {
      key: 'tickets-group',
      label: 'Ticket Views',
      icon: <ContainerOutlined />,
      children: [
        { key: 'tickets-all', label: 'All Tickets', path: '/tickets' },
        { key: 'tickets-assigned', label: 'My Tickets', path: '/tickets?assignee=me' },
        { key: 'tickets-unassigned', label: 'Grab Tickets', path: '/tickets?assignee=none' },
        { key: 'tickets-closed', label: 'Closed Tickets', path: '/tickets?status=closed' },
      ],
    },
    { key: 'assets', label: 'Assets', icon: <DatabaseOutlined />, path: '/assets' },
    { key: 'customers', label: 'Customers', icon: <TeamOutlined />, path: '/users' },
    { key: 'profile', label: 'Profile', icon: <UserOutlined />, path: '/me/settings' },
  ],
  manager: [
    { key: 'dashboard', label: 'Dashboard', icon: <DashboardOutlined />, path: '/' },
    { key: 'manager-queue', label: 'Queue Manager', icon: <AppstoreOutlined />, path: '/manager' },
    { key: 'manager-analytics', label: 'Analytics', icon: <FileTextOutlined />, path: '/manager/analytics' },
    { key: 'assets-dashboard', label: 'Asset Dashboard', icon: <DatabaseOutlined />, path: '/assets/dashboard' },
  ],
  admin: [
    { key: 'admin-settings', label: 'General Settings', icon: <SettingOutlined />, path: '/settings' },
    { key: 'admin-users', label: 'User Management', icon: <TeamOutlined />, path: '/settings/users' },
    { key: 'admin-assets', label: 'Asset Categories', icon: <DatabaseOutlined />, path: '/assets/categories' },
    { key: 'admin-mail', label: 'Mail Settings', icon: <SolutionOutlined />, path: '/settings/mail' },
    { key: 'admin-oidc', label: 'OIDC Settings', icon: <GlobalOutlined />, path: '/settings/oidc' },
    { key: 'admin-storage', label: 'Storage Settings', icon: <DatabaseOutlined />, path: '/settings/storage' },
  ],
};


export function SidebarLayout({ children }: { children?: ReactNode }) {
  const { data: me } = useMe();
  const loc = useLocation();
  const nav = useNavigate();
  const qc = useQueryClient();
  const userRoles = me?.roles || [];

  // Determine available roles for the switcher
  const availableRoles = useMemo(() => {
    const roles: Role[] = [];
    if (userRoles.includes('agent')) roles.push('agent');
    if (userRoles.includes('manager')) roles.push('manager');
    if (userRoles.includes('admin')) roles.push('admin');
    // Basic fallback if no roles match
    if (roles.length === 0) roles.push('agent');
    return roles;
  }, [userRoles]);

  const [activeRole, setActiveRole] = useState<Role>(availableRoles[0]);

  // Sync active role if available roles change (e.g. on load)
  useEffect(() => {
    if (!availableRoles.includes(activeRole)) {
      setActiveRole(availableRoles[0]);
    }
  }, [availableRoles, activeRole]);

  useEffect(() => {
    const titles: Record<string, string> = {
      '/': 'Dashboard',
      '/tickets': 'Tickets',
      '/metrics': 'Metrics',
      '/assets': 'Assets',
      '/settings': 'Settings',
      '/settings/mail': 'Mail Settings',
      '/settings/oidc': 'OIDC Settings',
      '/settings/storage': 'Storage Settings',
      '/settings/users': 'User Management',
      '/manager': 'Queue Manager',
      '/manager/analytics': 'Manager Analytics',
      '/me/settings': 'My Settings',
    };

    let title = 'Helpdesk';
    const path = loc.pathname;

    if (titles[path]) {
      title = `${titles[path]} - Helpdesk`;
    } else {
      if (path.startsWith('/tickets/')) title = 'Ticket Detail - Helpdesk';
      else if (path.startsWith('/assets/')) title = 'Assets - Helpdesk';
    }

    document.title = title;
  }, [loc]);

  const menuItems = useMemo(() => {
    const items = NAV_CONFIG[activeRole] || [];
    return items.map((g) => {
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
  }, [activeRole]);

  // Derive selected keys by matching current location (path + query) against nav config
  const selectedKeys = useMemo(() => {
    const items = NAV_CONFIG[activeRole] || [];
    const fullPath = loc.pathname + loc.search;
    
    // Find matching item in flat or nested structure
    for (const item of items) {
      // Check parent item path
      if (item.path && fullPath === item.path) {
        return [item.key];
      }
      // Check children paths
      if (item.children) {
        for (const child of item.children) {
          if (child.path && fullPath === child.path) {
            return [child.key];
          }
        }
      }
      // Fallback: if pathname (without query) starts with item path, select it
      if (item.path && loc.pathname.startsWith(item.path) && item.path !== '/') {
        return [item.key];
      }
    }
    
    // Default to dashboard if no match
    return ['dashboard'];
  }, [activeRole, loc.pathname, loc.search]);


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
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <div style={{ height: 64, display: 'flex', alignItems: 'center', padding: '0 24px', flexShrink: 0 }}>
          <Typography.Title level={4} style={{ margin: 0, color: '#6B4EFF', fontWeight: 800 }}>
            HELPDESK
          </Typography.Title>
        </div>

        {availableRoles.length > 1 && (
          <div style={{ padding: '0 16px 16px' }}>
            <Segmented
              block
              options={availableRoles.map(r => ({ label: ROLE_LABELS[r], value: r }))}
              value={activeRole}
              onChange={(v) => setActiveRole(v as Role)}
            />
          </div>
        )}

        <div style={{ flex: 1, overflowY: 'auto' }}>
          <Menu
            mode="inline"
            selectedKeys={selectedKeys}
            defaultOpenKeys={['tickets-group']}
            style={{ borderRight: 0 }}
            items={menuItems}
          />
        </div>
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
            height: 80,
          }}
        >
          {/* Left: Search */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Button type="text" icon={<MenuFoldOutlined />} style={{ display: 'none' }} />
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
                <Avatar style={{ backgroundColor: '#6B4EFF', verticalAlign: 'middle' }}>
                  {(displayName && displayName.charAt(0).toUpperCase()) || '?'}
                </Avatar>
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
            Copyright {new Date().getFullYear()} ©
          </div>
        </Content>
      </Layout>
    </Layout>
  );
}
