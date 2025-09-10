import React, { useState } from 'react';
import { Table, Button, Input, Select, Tag, Space, Card, Typography, Dropdown, Modal, message } from 'antd';
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, MoreOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api } from '../../shared/api';

const { Title } = Typography;
const { Option } = Select;

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
  location?: string;
  assigned_user?: {
    id: string;
    email: string;
    display_name?: string;
  };
  created_at: string;
  updated_at: string;
}

interface AssetListResponse {
  assets: Asset[];
  total: number;
  page: number;
  limit: number;
  pages: number;
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

export default function AssetList() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const queryClient = useQueryClient();

  // Fetch assets
  const { data, isLoading, error } = useQuery<AssetListResponse>({
    queryKey: ['assets', { search, status: statusFilter, page, limit: pageSize }],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (search) params.set('q', search);
      if (statusFilter.length > 0) {
        statusFilter.forEach(status => params.append('status', status));
      }
      params.set('page', page.toString());
      params.set('limit', pageSize.toString());

      const response = await api.get(`/assets?${params.toString()}`);
      return response.data;
    },
  });

  // Delete asset mutation
  const deleteAssetMutation = useMutation({
    mutationFn: async (assetId: string) => {
      await api.delete(`/assets/${assetId}`);
    },
    onSuccess: () => {
      message.success('Asset deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['assets'] });
    },
    onError: (error: any) => {
      message.error(`Failed to delete asset: ${error.response?.data?.error || error.message}`);
    },
  });

  const handleDelete = (asset: Asset) => {
    Modal.confirm({
      title: 'Delete Asset',
      content: `Are you sure you want to delete "${asset.name}" (${asset.asset_tag})?`,
      okText: 'Delete',
      okType: 'danger',
      onOk: () => deleteAssetMutation.mutate(asset.id),
    });
  };

  const columns = [
    {
      title: 'Asset Tag',
      dataIndex: 'asset_tag',
      key: 'asset_tag',
      width: 120,
      render: (tag: string, record: Asset) => (
        <Link to={`/assets/${record.id}`}>
          <strong>{tag}</strong>
        </Link>
      ),
    },
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: 'Category',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (category: Asset['category']) => category?.name || '-',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: Asset['status']) => (
        <Tag color={statusColors[status]}>{status.toUpperCase()}</Tag>
      ),
    },
    {
      title: 'Condition',
      dataIndex: 'condition',
      key: 'condition',
      width: 100,
      render: (condition: Asset['condition']) => 
        condition ? <Tag color={conditionColors[condition]}>{condition.toUpperCase()}</Tag> : '-',
    },
    {
      title: 'Location',
      dataIndex: 'location',
      key: 'location',
      width: 120,
      ellipsis: true,
      render: (location: string) => location || '-',
    },
    {
      title: 'Assigned To',
      dataIndex: 'assigned_user',
      key: 'assigned_user',
      width: 150,
      ellipsis: true,
      render: (user: Asset['assigned_user']) => 
        user ? (user.display_name || user.email) : '-',
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 100,
      render: (_: any, record: Asset) => (
        <Dropdown
          menu={{
            items: [
              {
                key: 'view',
                label: 'View Details',
                icon: <EyeOutlined />,
                onClick: () => window.open(`/assets/${record.id}`, '_blank'),
              },
              {
                key: 'edit',
                label: 'Edit',
                icon: <EditOutlined />,
              },
              {
                type: 'divider',
              },
              {
                key: 'delete',
                label: 'Delete',
                icon: <DeleteOutlined />,
                danger: true,
                onClick: () => handleDelete(record),
              },
            ],
          }}
          trigger={['click']}
        >
          <Button type="text" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
  ];

  if (error) {
    return (
      <Card>
        <div style={{ textAlign: 'center', padding: 20 }}>
          <p>Error loading assets: {(error as any).message}</p>
          <Button onClick={() => queryClient.invalidateQueries({ queryKey: ['assets'] })}>
            Retry
          </Button>
        </div>
      </Card>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2} style={{ margin: 0 }}>Assets</Title>
        <Link to="/assets/new">
          <Button type="primary" icon={<PlusOutlined />}>
            New Asset
          </Button>
        </Link>
      </div>

      <Card>
        <div style={{ marginBottom: 16, display: 'flex', gap: 16, flexWrap: 'wrap' }}>
          <Input
            placeholder="Search assets..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ width: 300 }}
          />
          <Select
            mode="multiple"
            placeholder="Filter by status"
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ minWidth: 200 }}
          >
            <Option value="active">Active</Option>
            <Option value="inactive">Inactive</Option>
            <Option value="maintenance">Maintenance</Option>
            <Option value="retired">Retired</Option>
            <Option value="disposed">Disposed</Option>
          </Select>
        </div>

        <Table
          columns={columns}
          dataSource={data?.assets || []}
          loading={isLoading}
          rowKey="id"
          pagination={{
            current: page,
            pageSize,
            total: data?.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} assets`,
            onChange: (newPage, newPageSize) => {
              setPage(newPage);
              if (newPageSize !== pageSize) {
                setPageSize(newPageSize);
                setPage(1);
              }
            },
          }}
          scroll={{ x: 1000 }}
        />
      </Card>
    </div>
  );
}