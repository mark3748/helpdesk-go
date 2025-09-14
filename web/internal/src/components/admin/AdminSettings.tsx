import { Card, Row, Col, Statistic, List, Button, Space, Typography, Tag, Alert } from 'antd';
import { SettingOutlined, DatabaseOutlined, MailOutlined, LockOutlined, CloudOutlined, UserOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api } from '../../shared/api';

const { Title, Text } = Typography;

interface SystemInfo {
  version: string;
  uptime: string;
  database_status: 'connected' | 'disconnected';
  storage_status: 'configured' | 'not_configured';
  mail_status: 'configured' | 'not_configured';
  oidc_status: 'configured' | 'not_configured';
  total_users: number;
  total_tickets: number;
  total_assets: number;
}

export default function AdminSettings() {
  const { data: systemInfo, isLoading } = useQuery<SystemInfo>({
    queryKey: ['system-info'],
    queryFn: async () => {
      // Mock data since this endpoint might not exist yet
      return {
        version: '1.0.0',
        uptime: '5 days, 12 hours',
        database_status: 'connected',
        storage_status: 'configured',
        mail_status: 'configured',
        oidc_status: 'not_configured',
        total_users: 25,
        total_tickets: 150,
        total_assets: 75,
      } as SystemInfo;
    },
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
      case 'configured':
        return 'success';
      case 'disconnected':
      case 'not_configured':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'connected':
        return 'Connected';
      case 'configured':
        return 'Configured';
      case 'disconnected':
        return 'Disconnected';
      case 'not_configured':
        return 'Not Configured';
      default:
        return status;
    }
  };

  const settingsCategories = [
    {
      title: 'Mail Settings',
      description: 'Configure SMTP settings for email notifications',
      icon: <MailOutlined />,
      path: '/settings/mail',
      status: systemInfo?.mail_status,
    },
    {
      title: 'OIDC Settings',
      description: 'Configure OpenID Connect authentication',
      icon: <LockOutlined />,
      path: '/settings/oidc',
      status: systemInfo?.oidc_status,
    },
    {
      title: 'Storage Settings',
      description: 'Configure file storage and attachments',
      icon: <CloudOutlined />,
      path: '/settings/storage',
      status: systemInfo?.storage_status,
    },
    {
      title: 'User Management',
      description: 'Manage users and role assignments',
      icon: <UserOutlined />,
      path: '/settings/users',
      status: 'configured',
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title level={2}>Admin Settings</Title>
        <Text type="secondary">
          Manage system configuration and monitor system health
        </Text>
      </div>

      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* System Overview */}
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={6}>
            <Card loading={isLoading}>
              <Statistic
                title="System Version"
                value={systemInfo?.version || 'Unknown'}
                prefix={<SettingOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card loading={isLoading}>
              <Statistic
                title="Uptime"
                value={systemInfo?.uptime || 'Unknown'}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card loading={isLoading}>
              <Statistic
                title="Total Users"
                value={systemInfo?.total_users || 0}
                prefix={<UserOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card loading={isLoading}>
              <Statistic
                title="Database"
                value={getStatusText(systemInfo?.database_status || 'unknown')}
                prefix={<DatabaseOutlined />}
                valueStyle={{ 
                  color: systemInfo?.database_status === 'connected' ? '#3f8600' : '#cf1322' 
                }}
              />
            </Card>
          </Col>
        </Row>

        {/* System Status */}
        <Card title="System Status">
          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text>Database Connection</Text>
                  <Tag color={getStatusColor(systemInfo?.database_status || '')}>
                    {getStatusText(systemInfo?.database_status || 'unknown')}
                  </Tag>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text>Mail Configuration</Text>
                  <Tag color={getStatusColor(systemInfo?.mail_status || '')}>
                    {getStatusText(systemInfo?.mail_status || 'unknown')}
                  </Tag>
                </div>
              </Space>
            </Col>
            <Col xs={24} md={12}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text>Storage Configuration</Text>
                  <Tag color={getStatusColor(systemInfo?.storage_status || '')}>
                    {getStatusText(systemInfo?.storage_status || 'unknown')}
                  </Tag>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text>OIDC Authentication</Text>
                  <Tag color={getStatusColor(systemInfo?.oidc_status || '')}>
                    {getStatusText(systemInfo?.oidc_status || 'unknown')}
                  </Tag>
                </div>
              </Space>
            </Col>
          </Row>
        </Card>

        {/* Configuration Categories */}
        <Card title="Configuration Categories">
          <List
            grid={{ gutter: 16, xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 3 }}
            dataSource={settingsCategories}
            renderItem={(item) => (
              <List.Item>
                <Card
                  hoverable
                  actions={[
                    <Link to={item.path} key="configure">
                      <Button type="primary" size="small">
                        Configure
                      </Button>
                    </Link>
                  ]}
                >
                  <Card.Meta
                    avatar={item.icon}
                    title={
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        {item.title}
                        <Tag color={getStatusColor(item.status || '')}>
                          {getStatusText(item.status || 'unknown')}
                        </Tag>
                      </div>
                    }
                    description={item.description}
                  />
                </Card>
              </List.Item>
            )}
          />
        </Card>

        {/* System Alerts */}
        {systemInfo?.oidc_status === 'not_configured' && (
          <Alert
            message="OIDC Not Configured"
            description="Single sign-on authentication is not configured. Users will need to use local accounts."
            type="warning"
            showIcon
            action={
              <Link to="/settings/oidc">
                <Button size="small" type="primary">
                  Configure OIDC
                </Button>
              </Link>
            }
          />
        )}
      </Space>
    </div>
  );
}
