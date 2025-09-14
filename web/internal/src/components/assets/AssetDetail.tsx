import { useState } from 'react';
import { Card, Descriptions, Button, Space, Tag, Tabs, Table, Modal, message } from 'antd';
import { EditOutlined, ArrowLeftOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { api } from '../../shared/api';
import dayjs from 'dayjs';

interface Asset {
  id: string;
  asset_tag: string;
  name: string;
  description?: string;
  status: 'active' | 'inactive' | 'maintenance' | 'retired' | 'disposed';
  condition?: 'excellent' | 'good' | 'fair' | 'poor' | 'broken';
  category?: {
    id: string;
    name: string;
  };
  manufacturer?: string;
  model?: string;
  serial_number?: string;
  location?: string;
  purchase_price?: number;
  purchase_date?: string;
  warranty_expiry?: string;
  depreciation_rate?: number;
  assigned_user?: {
    id: string;
    email: string;
    display_name?: string;
  };
  created_at: string;
  updated_at: string;
}

interface AuditEntry {
  id: string;
  user_name: string;
  action: string;
  field_name?: string;
  old_value?: string;
  new_value?: string;
  notes?: string;
  timestamp: string;
}

interface AssetRelationship {
  id: string;
  related_asset: {
    id: string;
    name: string;
    asset_tag: string;
  };
  relationship_type: string;
  direction: 'parent' | 'child';
}

const statusColors = {
  active: 'green',
  inactive: 'default',
  maintenance: 'orange',
  retired: 'red',
  disposed: 'red',
};

const conditionColors = {
  excellent: 'green',
  good: 'blue',
  fair: 'orange',
  poor: 'red',
  broken: 'red',
};

export default function AssetDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);

  // Fetch asset details
  const { data: asset, isLoading } = useQuery<Asset>({
    queryKey: ['asset', id],
    queryFn: async () => {
      const response = await api.get(`/assets/${id}`);
      return (response as any).data;
    },
    enabled: !!id,
  });

  // Fetch audit history
  const { data: auditHistory } = useQuery<AuditEntry[]>({
    queryKey: ['asset-audit', id],
    queryFn: async () => {
      const response = await api.get(`/assets/${id}/audit`);
      return (response as any).data;
    },
    enabled: !!id,
  });

  // Fetch relationships
  const { data: relationships } = useQuery<AssetRelationship[]>({
    queryKey: ['asset-relationships', id],
    queryFn: async () => {
      const response = await api.get(`/assets/${id}/relationships`);
      return (response as any).data || [];
    },
    enabled: !!id,
  });

  // Delete asset mutation
  const deleteAssetMutation = useMutation({
    mutationFn: async () => {
      await api.delete(`/assets/${id}`);
    },
    onSuccess: () => {
      message.success('Asset deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      navigate('/assets');
    },
    onError: (error: any) => {
      message.error(`Failed to delete asset: ${error.response?.data?.error || error.message}`);
    },
  });

  const handleDelete = () => {
    deleteAssetMutation.mutate();
    setDeleteModalVisible(false);
  };

  const formatCurrency = (value?: number) => {
    if (!value) return '-';
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  };

  const formatDate = (date?: string) => {
    if (!date) return '-';
    return dayjs(date).format('MMM D, YYYY');
  };

  const auditColumns = [
    {
      title: 'Date',
      dataIndex: 'timestamp',
      key: 'timestamp',
      render: (timestamp: string) => dayjs(timestamp).format('MMM D, YYYY h:mm A'),
      width: 150,
    },
    {
      title: 'User',
      dataIndex: 'user_name',
      key: 'user_name',
      width: 120,
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
      width: 100,
      render: (action: string) => (
        <Tag color="blue">{action.toUpperCase()}</Tag>
      ),
    },
    {
      title: 'Changes',
      key: 'changes',
      render: (record: AuditEntry) => {
        if (record.field_name && record.old_value && record.new_value) {
          return (
            <div>
              <strong>{record.field_name}:</strong> {record.old_value} → {record.new_value}
            </div>
          );
        }
        return record.notes || '-';
      },
    },
  ];

  const relationshipColumns = [
    {
      title: 'Asset',
      key: 'asset',
      render: (record: AssetRelationship) => (
        <Link to={`/assets/${record.related_asset.id}`}>
          <strong>{record.related_asset.name}</strong>
          <br />
          <span style={{ color: '#666', fontSize: '12px' }}>
            {record.related_asset.asset_tag}
          </span>
        </Link>
      ),
    },
    {
      title: 'Relationship',
      dataIndex: 'relationship_type',
      key: 'relationship_type',
      render: (type: string, record: AssetRelationship) => (
        <Tag color={record.direction === 'parent' ? 'blue' : 'green'}>
          {type.replace('_', ' ').toUpperCase()}
        </Tag>
      ),
    },
    {
      title: 'Direction',
      dataIndex: 'direction',
      key: 'direction',
      render: (direction: string) => (
        <Tag color={direction === 'parent' ? 'orange' : 'purple'}>
          {direction.toUpperCase()}
        </Tag>
      ),
    },
  ];

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        Loading asset details...
      </div>
    );
  }

  if (!asset) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        Asset not found
      </div>
    );
  }

  const tabItems = [
    {
      key: 'details',
      label: 'Details',
      children: (
        <Card>
          <Descriptions bordered column={2}>
            <Descriptions.Item label="Asset Tag" span={1}>
              <strong>{asset.asset_tag}</strong>
            </Descriptions.Item>
            <Descriptions.Item label="Name" span={1}>
              {asset.name}
            </Descriptions.Item>
            <Descriptions.Item label="Description" span={2}>
              {asset.description || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Status">
              <Tag color={statusColors[asset.status]}>
                {asset.status.toUpperCase()}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Condition">
              {asset.condition ? (
                <Tag color={conditionColors[asset.condition]}>
                  {asset.condition.toUpperCase()}
                </Tag>
              ) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Category">
              {asset.category?.name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Location">
              {asset.location || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Manufacturer">
              {asset.manufacturer || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Model">
              {asset.model || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Serial Number">
              {asset.serial_number || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Assigned To">
              {asset.assigned_user ? 
                (asset.assigned_user.display_name || asset.assigned_user.email) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Purchase Price">
              {formatCurrency(asset.purchase_price)}
            </Descriptions.Item>
            <Descriptions.Item label="Purchase Date">
              {formatDate(asset.purchase_date)}
            </Descriptions.Item>
            <Descriptions.Item label="Warranty Expiry">
              {formatDate(asset.warranty_expiry)}
            </Descriptions.Item>
            <Descriptions.Item label="Depreciation Rate">
              {asset.depreciation_rate ? `${asset.depreciation_rate}%` : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Created">
              {formatDate(asset.created_at)}
            </Descriptions.Item>
            <Descriptions.Item label="Last Updated">
              {formatDate(asset.updated_at)}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      ),
    },
    {
      key: 'relationships',
      label: `Relationships (${relationships?.length || 0})`,
      children: (
        <Card>
          <Table
            dataSource={relationships}
            columns={relationshipColumns}
            rowKey="id"
            pagination={false}
            locale={{ emptyText: 'No relationships found' }}
          />
        </Card>
      ),
    },
    {
      key: 'audit',
      label: `Audit History (${auditHistory?.length || 0})`,
      children: (
        <Card>
          <Table
            dataSource={auditHistory}
            columns={auditColumns}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            locale={{ emptyText: 'No audit history found' }}
          />
        </Card>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/assets')}>
            Back to Assets
          </Button>
          <div>
            <h2 style={{ margin: 0 }}>{asset.name}</h2>
            <span style={{ color: '#666' }}>{asset.asset_tag}</span>
          </div>
        </div>
        <Space>
          <Link to={`/assets/${asset.id}/edit`}>
            <Button type="primary" icon={<EditOutlined />}>
              Edit Asset
            </Button>
          </Link>
          <Button 
            danger 
            icon={<DeleteOutlined />}
            onClick={() => setDeleteModalVisible(true)}
          >
            Delete
          </Button>
        </Space>
      </div>

      <Tabs items={tabItems} />

      <Modal
        title="Delete Asset"
        open={deleteModalVisible}
        onCancel={() => setDeleteModalVisible(false)}
        onOk={handleDelete}
        okText="Delete"
        okType="danger"
        confirmLoading={deleteAssetMutation.isPending}
      >
        <p>Are you sure you want to delete "{asset.name}" ({asset.asset_tag})?</p>
        <p>This action cannot be undone.</p>
      </Modal>
    </div>
  );
}