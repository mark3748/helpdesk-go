import React, { useEffect } from 'react';
import { Form, Input, Select, DatePicker, InputNumber, Card, Button, Row, Col, message, Spin } from 'antd';
import { SaveOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { api } from '../../shared/api';
import dayjs from 'dayjs';

const { Option } = Select;
const { TextArea } = Input;

interface Asset {
  id?: string;
  asset_tag: string;
  name: string;
  description?: string;
  category_id?: string;
  status: 'active' | 'inactive' | 'maintenance' | 'retired' | 'disposed';
  condition?: 'excellent' | 'good' | 'fair' | 'poor' | 'broken';
  purchase_price?: number;
  purchase_date?: string;
  warranty_expiry?: string;
  depreciation_rate?: number;
  serial_number?: string;
  model?: string;
  manufacturer?: string;
  location?: string;
  custom_fields?: Record<string, any>;
}

interface AssetCategory {
  id: string;
  name: string;
  description?: string;
}

export default function AssetForm() {
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isEditing = id !== 'new' && !!id;
  const queryClient = useQueryClient();

  // Fetch asset categories
  const { data: categories } = useQuery<AssetCategory[]>({
    queryKey: ['asset-categories'],
    queryFn: async () => {
      const response = await api.get('/asset-categories');
      return response.data;
    },
  });

  // Fetch existing asset if editing
  const { data: asset, isLoading } = useQuery<Asset>({
    queryKey: ['asset', id],
    queryFn: async () => {
      const response = await api.get(`/assets/${id}`);
      return response.data;
    },
    enabled: isEditing,
  });

  // Create/Update asset mutation
  const saveAssetMutation = useMutation({
    mutationFn: async (values: Asset) => {
      if (isEditing) {
        const response = await api.patch(`/assets/${id}`, values);
        return response.data;
      } else {
        const response = await api.post('/assets', values);
        return response.data;
      }
    },
    onSuccess: (data) => {
      message.success(`Asset ${isEditing ? 'updated' : 'created'} successfully`);
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      navigate(`/assets/${data.id}`);
    },
    onError: (error: any) => {
      message.error(`Failed to ${isEditing ? 'update' : 'create'} asset: ${error.response?.data?.error || error.message}`);
    },
  });

  // Populate form with existing asset data
  useEffect(() => {
    if (asset) {
      form.setFieldsValue({
        ...asset,
        purchase_date: asset.purchase_date ? dayjs(asset.purchase_date) : null,
        warranty_expiry: asset.warranty_expiry ? dayjs(asset.warranty_expiry) : null,
      });
    }
  }, [asset, form]);

  const onFinish = (values: any) => {
    const formattedValues = {
      ...values,
      purchase_date: values.purchase_date?.format('YYYY-MM-DD') || null,
      warranty_expiry: values.warranty_expiry?.format('YYYY-MM-DD') || null,
    };
    saveAssetMutation.mutate(formattedValues);
  };

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', alignItems: 'center', gap: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/assets')}>
          Back to Assets
        </Button>
        <h2 style={{ margin: 0 }}>
          {isEditing ? `Edit Asset: ${asset?.name}` : 'Create New Asset'}
        </h2>
      </div>

      <Form
        form={form}
        layout="vertical"
        onFinish={onFinish}
        initialValues={{
          status: 'active',
          condition: 'good',
        }}
      >
        <Row gutter={[24, 0]}>
          {/* Basic Information */}
          <Col xs={24} lg={12}>
            <Card title="Basic Information" style={{ marginBottom: 24 }}>
              <Form.Item
                name="asset_tag"
                label="Asset Tag"
                rules={[{ required: true, message: 'Asset tag is required' }]}
              >
                <Input placeholder="e.g., LP-001, DT-042" />
              </Form.Item>

              <Form.Item
                name="name"
                label="Asset Name"
                rules={[{ required: true, message: 'Asset name is required' }]}
              >
                <Input placeholder="e.g., Dell Laptop, Office Chair" />
              </Form.Item>

              <Form.Item name="description" label="Description">
                <TextArea rows={3} placeholder="Brief description of the asset" />
              </Form.Item>

              <Form.Item name="category_id" label="Category">
                <Select placeholder="Select a category" allowClear>
                  {categories?.map(category => (
                    <Option key={category.id} value={category.id}>
                      {category.name}
                    </Option>
                  ))}
                </Select>
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="status" label="Status" rules={[{ required: true }]}>
                    <Select>
                      <Option value="active">Active</Option>
                      <Option value="inactive">Inactive</Option>
                      <Option value="maintenance">Maintenance</Option>
                      <Option value="retired">Retired</Option>
                      <Option value="disposed">Disposed</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="condition" label="Condition">
                    <Select placeholder="Select condition">
                      <Option value="excellent">Excellent</Option>
                      <Option value="good">Good</Option>
                      <Option value="fair">Fair</Option>
                      <Option value="poor">Poor</Option>
                      <Option value="broken">Broken</Option>
                    </Select>
                  </Form.Item>
                </Col>
              </Row>
            </Card>
          </Col>

          {/* Details */}
          <Col xs={24} lg={12}>
            <Card title="Asset Details" style={{ marginBottom: 24 }}>
              <Form.Item name="manufacturer" label="Manufacturer">
                <Input placeholder="e.g., Dell, Apple, Microsoft" />
              </Form.Item>

              <Form.Item name="model" label="Model">
                <Input placeholder="e.g., Latitude 7420, iPhone 15 Pro" />
              </Form.Item>

              <Form.Item name="serial_number" label="Serial Number">
                <Input placeholder="Unique serial number" />
              </Form.Item>

              <Form.Item name="location" label="Location">
                <Input placeholder="e.g., Office Floor 3, Warehouse A" />
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="purchase_price" label="Purchase Price">
                    <InputNumber
                      style={{ width: '100%' }}
                      formatter={value => `$ ${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                      parser={value => value!.replace(/\$\s?|(,*)/g, '')}
                      placeholder="0.00"
                      precision={2}
                    />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="depreciation_rate" label="Depreciation Rate (%)">
                    <InputNumber
                      style={{ width: '100%' }}
                      min={0}
                      max={100}
                      formatter={value => `${value}%`}
                      parser={value => value!.replace('%', '')}
                      placeholder="0"
                      precision={2}
                    />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="purchase_date" label="Purchase Date">
                    <DatePicker style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="warranty_expiry" label="Warranty Expiry">
                    <DatePicker style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
            </Card>
          </Col>
        </Row>

        {/* Actions */}
        <Card>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
            <Button onClick={() => navigate('/assets')}>
              Cancel
            </Button>
            <Button 
              type="primary" 
              htmlType="submit" 
              icon={<SaveOutlined />}
              loading={saveAssetMutation.isPending}
            >
              {isEditing ? 'Update Asset' : 'Create Asset'}
            </Button>
          </div>
        </Card>
      </Form>
    </div>
  );
}