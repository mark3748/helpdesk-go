import { useState } from 'react';
import { Card, Table, Button, Modal, Form, Input, message, Space, Popconfirm } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../shared/api';

const { TextArea } = Input;

interface AssetCategory {
  id: string;
  name: string;
  description?: string;
  asset_count?: number;
  created_at: string;
  updated_at: string;
}

interface CategoryFormData {
  name: string;
  description?: string;
}

export default function AssetCategories() {
  const [modalVisible, setModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState<AssetCategory | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  // Fetch categories
  const { data: categories, isLoading } = useQuery<AssetCategory[]>({
    queryKey: ['asset-categories'],
    queryFn: async () => {
      const response = await api.get('/asset-categories');
      return (response as any).data;
    },
  });

  // Create/Update category mutation
  const saveCategoryMutation = useMutation({
    mutationFn: async (data: CategoryFormData) => {
      if (editingCategory) {
        const response = await api.patch(`/asset-categories/${editingCategory.id}`, data);
        return (response as any).data;
      } else {
        const response = await api.post('/asset-categories', data);
        return (response as any).data;
      }
    },
    onSuccess: () => {
      message.success(`Category ${editingCategory ? 'updated' : 'created'} successfully`);
      queryClient.invalidateQueries({ queryKey: ['asset-categories'] });
      handleCloseModal();
    },
    onError: (error: any) => {
      message.error(`Failed to ${editingCategory ? 'update' : 'create'} category: ${error.response?.data?.error || error.message}`);
    },
  });

  // Delete category mutation
  const deleteCategoryMutation = useMutation({
    mutationFn: async (categoryId: string) => {
      await api.delete(`/asset-categories/${categoryId}`);
    },
    onSuccess: () => {
      message.success('Category deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['asset-categories'] });
    },
    onError: (error: any) => {
      message.error(`Failed to delete category: ${error.response?.data?.error || error.message}`);
    },
  });

  const handleOpenModal = (category?: AssetCategory) => {
    setEditingCategory(category || null);
    if (category) {
      form.setFieldsValue(category);
    } else {
      form.resetFields();
    }
    setModalVisible(true);
  };

  const handleCloseModal = () => {
    setModalVisible(false);
    setEditingCategory(null);
    form.resetFields();
  };

  const handleSubmit = (values: CategoryFormData) => {
    saveCategoryMutation.mutate(values);
  };

  const handleDelete = (category: AssetCategory) => {
    if (category.asset_count && category.asset_count > 0) {
      message.warning(`Cannot delete category "${category.name}" because it has ${category.asset_count} assets assigned to it.`);
      return;
    }
    deleteCategoryMutation.mutate(category.id);
  };

  const columns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => <strong>{name}</strong>,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (description: string) => description || '-',
    },
    {
      title: 'Assets',
      dataIndex: 'asset_count',
      key: 'asset_count',
      width: 100,
      render: (count: number) => count || 0,
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 120,
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_: any, record: AssetCategory) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleOpenModal(record)}
          />
          <Popconfirm
            title="Delete Category"
            description={`Are you sure you want to delete "${record.name}"?`}
            onConfirm={() => handleDelete(record)}
            okText="Delete"
            okType="danger"
            disabled={Boolean(record.asset_count && record.asset_count > 0)}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              disabled={Boolean(record.asset_count && record.asset_count > 0)}
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>Asset Categories</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => handleOpenModal()}>
          New Category
        </Button>
      </div>

      <Card>
        <Table
          dataSource={categories}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{ pageSize: 20 }}
        />
      </Card>

      <Modal
        title={editingCategory ? 'Edit Category' : 'Create Category'}
        open={modalVisible}
        onCancel={handleCloseModal}
        onOk={() => form.submit()}
        confirmLoading={saveCategoryMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="name"
            label="Category Name"
            rules={[{ required: true, message: 'Category name is required' }]}
          >
            <Input placeholder="e.g., Laptops, Furniture, Vehicles" />
          </Form.Item>

          <Form.Item name="description" label="Description">
            <TextArea rows={3} placeholder="Brief description of this category..." />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}