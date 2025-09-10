import { useState } from 'react';
import { Card, Table, Button, Modal, Form, Input, DatePicker, Select, message, Tag, Space, Tooltip } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../shared/api';
import dayjs from 'dayjs';

const { Option } = Select;
const { TextArea } = Input;

interface AssetCheckout {
  id: string;
  asset_id: string;
  asset_name: string;
  asset_tag: string;
  checked_out_to: string;
  checked_out_by: string;
  checkout_date: string;
  expected_return_date?: string;
  actual_return_date?: string;
  condition_out: 'excellent' | 'good' | 'fair' | 'poor' | 'broken';
  condition_in?: 'excellent' | 'good' | 'fair' | 'poor' | 'broken';
  notes?: string;
  return_notes?: string;
  status: 'checked_out' | 'overdue' | 'returned';
}

interface Asset {
  id: string;
  name: string;
  asset_tag: string;
}

interface CheckoutFormData {
  asset_id: string;
  checked_out_to: string;
  expected_return_date?: string;
  condition_out: string;
  notes?: string;
}

interface CheckinFormData {
  condition_in: string;
  return_notes?: string;
}

export default function AssetCheckout() {
  const [checkoutModalVisible, setCheckoutModalVisible] = useState(false);
  const [checkinModalVisible, setCheckinModalVisible] = useState(false);
  const [selectedCheckout, setSelectedCheckout] = useState<AssetCheckout | null>(null);
  const [checkoutForm] = Form.useForm();
  const [checkinForm] = Form.useForm();
  const queryClient = useQueryClient();

  // Fetch checkouts
  const { data: checkouts, isLoading } = useQuery<AssetCheckout[]>({
    queryKey: ['asset-checkouts'],
    queryFn: () => api.get<AssetCheckout[]>('/asset-checkouts'),
  });

  // Fetch available assets for checkout
  const { data: availableAssets } = useQuery<Asset[]>({
    queryKey: ['assets-available'],
    queryFn: () =>
      api.get<{ assets: Asset[] }>('/assets?status=active').then((res) => res.assets),
  });

  // Checkout asset mutation
  const checkoutMutation = useMutation({
    mutationFn: (data: CheckoutFormData) =>
      api.post<AssetCheckout>('/asset-checkouts', data),
    onSuccess: () => {
      message.success('Asset checked out successfully');
      queryClient.invalidateQueries({ queryKey: ['asset-checkouts'] });
      queryClient.invalidateQueries({ queryKey: ['assets-available'] });
      setCheckoutModalVisible(false);
      checkoutForm.resetFields();
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } }; message: string };
      message.error(`Failed to checkout asset: ${err.response?.data?.error || err.message}`);
    },
  });

  // Checkin asset mutation
  const checkinMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CheckinFormData }) =>
      api.patch<AssetCheckout>(`/asset-checkouts/${id}/checkin`, data),
    onSuccess: () => {
      message.success('Asset checked in successfully');
      queryClient.invalidateQueries({ queryKey: ['asset-checkouts'] });
      queryClient.invalidateQueries({ queryKey: ['assets-available'] });
      setCheckinModalVisible(false);
      setSelectedCheckout(null);
      checkinForm.resetFields();
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } }; message: string };
      message.error(`Failed to checkin asset: ${err.response?.data?.error || err.message}`);
    },
  });

  const handleCheckout = (values: CheckoutFormData) => {
    checkoutMutation.mutate({
      ...values,
      expected_return_date: values.expected_return_date
        ? dayjs(values.expected_return_date).format('YYYY-MM-DD')
        : undefined,
    });
  };

  const handleCheckin = (values: CheckinFormData) => {
    if (!selectedCheckout) return;
    checkinMutation.mutate({
      id: selectedCheckout.id,
      data: values,
    });
  };

  const openCheckinModal = (checkout: AssetCheckout) => {
    setSelectedCheckout(checkout);
    setCheckinModalVisible(true);
  };

  const getStatusTag = (checkout: AssetCheckout) => {
    switch (checkout.status) {
      case 'checked_out':
        return <Tag color="blue" icon={<ClockCircleOutlined />}>Checked Out</Tag>;
      case 'overdue':
        return <Tag color="red" icon={<ExclamationCircleOutlined />}>Overdue</Tag>;
      case 'returned':
        return <Tag color="green" icon={<CheckCircleOutlined />}>Returned</Tag>;
      default:
        return <Tag>{checkout.status}</Tag>;
    }
  };

  const getConditionColor = (condition: string) => {
    switch (condition) {
      case 'excellent': return 'green';
      case 'good': return 'blue';
      case 'fair': return 'orange';
      case 'poor': return 'red';
      case 'broken': return 'volcano';
      default: return 'default';
    }
  };

  const columns = [
    {
      title: 'Asset',
      key: 'asset',
      render: (record: AssetCheckout) => (
        <div>
          <div style={{ fontWeight: 500 }}>{record.asset_name}</div>
          <div style={{ color: '#666', fontSize: '12px' }}>{record.asset_tag}</div>
        </div>
      ),
    },
    {
      title: 'Checked Out To',
      dataIndex: 'checked_out_to',
      key: 'checked_out_to',
    },
    {
      title: 'Checkout Date',
      dataIndex: 'checkout_date',
      key: 'checkout_date',
      render: (date: string) => dayjs(date).format('MMM D, YYYY'),
    },
    {
      title: 'Expected Return',
      dataIndex: 'expected_return_date',
      key: 'expected_return_date',
      render: (date: string | null) => date ? dayjs(date).format('MMM D, YYYY') : '-',
    },
    {
      title: 'Status',
      key: 'status',
      render: (record: AssetCheckout) => getStatusTag(record),
    },
    {
      title: 'Condition',
      key: 'condition',
      render: (record: AssetCheckout) => (
        <Space>
          <Tooltip title="Condition when checked out">
            <Tag color={getConditionColor(record.condition_out)}>
              Out: {record.condition_out}
            </Tag>
          </Tooltip>
          {record.condition_in && (
            <Tooltip title="Condition when returned">
              <Tag color={getConditionColor(record.condition_in)}>
                In: {record.condition_in}
              </Tag>
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (record: AssetCheckout) => (
        <Space>
          {record.status !== 'returned' && (
            <Button
              type="primary"
              size="small"
              onClick={() => openCheckinModal(record)}
            >
              Check In
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>Asset Checkouts</h2>
        <Button type="primary" onClick={() => setCheckoutModalVisible(true)}>
          Checkout Asset
        </Button>
      </div>

      <Card>
        <Table
          dataSource={checkouts}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      {/* Checkout Modal */}
      <Modal
        title="Checkout Asset"
        open={checkoutModalVisible}
        onCancel={() => setCheckoutModalVisible(false)}
        onOk={() => checkoutForm.submit()}
        confirmLoading={checkoutMutation.isPending}
      >
        <Form
          form={checkoutForm}
          layout="vertical"
          onFinish={handleCheckout}
        >
          <Form.Item
            name="asset_id"
            label="Asset"
            rules={[{ required: true, message: 'Please select an asset' }]}
          >
            <Select placeholder="Select an asset to checkout">
              {availableAssets?.map((asset) => (
                <Option key={asset.id} value={asset.id}>
                  {asset.name} ({asset.asset_tag})
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="checked_out_to"
            label="Checked Out To"
            rules={[{ required: true, message: 'Please enter who the asset is checked out to' }]}
          >
            <Input placeholder="Employee name or department" />
          </Form.Item>

          <Form.Item name="expected_return_date" label="Expected Return Date">
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="condition_out"
            label="Condition When Checking Out"
            rules={[{ required: true, message: 'Please select the asset condition' }]}
          >
            <Select placeholder="Select condition">
              <Option value="excellent">Excellent</Option>
              <Option value="good">Good</Option>
              <Option value="fair">Fair</Option>
              <Option value="poor">Poor</Option>
              <Option value="broken">Broken</Option>
            </Select>
          </Form.Item>

          <Form.Item name="notes" label="Checkout Notes">
            <TextArea rows={3} placeholder="Any notes about the checkout..." />
          </Form.Item>
        </Form>
      </Modal>

      {/* Checkin Modal */}
      <Modal
        title={`Check In Asset: ${selectedCheckout?.asset_name}`}
        open={checkinModalVisible}
        onCancel={() => {
          setCheckinModalVisible(false);
          setSelectedCheckout(null);
        }}
        onOk={() => checkinForm.submit()}
        confirmLoading={checkinMutation.isPending}
      >
        <Form
          form={checkinForm}
          layout="vertical"
          onFinish={handleCheckin}
        >
          <div style={{ marginBottom: 16, padding: 12, backgroundColor: '#f5f5f5', borderRadius: 4 }}>
            <div><strong>Asset:</strong> {selectedCheckout?.asset_name}</div>
            <div><strong>Checked out to:</strong> {selectedCheckout?.checked_out_to}</div>
            <div><strong>Checkout date:</strong> {selectedCheckout && dayjs(selectedCheckout.checkout_date).format('MMM D, YYYY')}</div>
            <div><strong>Condition when checked out:</strong> {selectedCheckout?.condition_out}</div>
          </div>

          <Form.Item
            name="condition_in"
            label="Condition When Returning"
            rules={[{ required: true, message: 'Please select the asset condition' }]}
          >
            <Select placeholder="Select condition">
              <Option value="excellent">Excellent</Option>
              <Option value="good">Good</Option>
              <Option value="fair">Fair</Option>
              <Option value="poor">Poor</Option>
              <Option value="broken">Broken</Option>
            </Select>
          </Form.Item>

          <Form.Item name="return_notes" label="Return Notes">
            <TextArea rows={3} placeholder="Any notes about the return or asset condition..." />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}