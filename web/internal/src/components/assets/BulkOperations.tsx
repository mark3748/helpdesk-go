import React, { useState } from 'react';
import { Card, Button, Upload, Select, Form, Input, Modal, Table, Progress, message, Space, Alert } from 'antd';
import { UploadOutlined, DownloadOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../shared/api';

const { Option } = Select;
const { TextArea } = Input;

interface BulkUpdateData {
  asset_ids: string[];
  updates: {
    status?: string;
    condition?: string;
    location?: string;
    category_id?: string;
  };
  notes?: string;
}

interface ImportProgress {
  total: number;
  processed: number;
  errors: Array<{ row: number; error: string }>;
  warnings: Array<{ row: number; warning: string }>;
}

export default function BulkOperations() {
  const [selectedAssets, setSelectedAssets] = useState<string[]>([]);
  const [bulkUpdateModalVisible, setBulkUpdateModalVisible] = useState(false);
  const [importProgress, setImportProgress] = useState<ImportProgress | null>(null);
  const [uploading, setUploading] = useState(false);
  const [bulkForm] = Form.useForm();
  const queryClient = useQueryClient();

  // Fetch assets for bulk operations
  const { data: assets, isLoading } = useQuery({
    queryKey: ['assets-bulk'],
    queryFn: async () => {
      const response = await api.get<{ items: any[] }>('/assets?limit=1000');
      return response.items;
    },
  });

  // Fetch asset categories
  const { data: categories } = useQuery({
    queryKey: ['asset-categories'],
    queryFn: async () => {
      const response = await api.get('/asset-categories');
      return (response as { data: any }).data;
    },
  });

  // Bulk update mutation
  const bulkUpdateMutation = useMutation({
    mutationFn: async (data: BulkUpdateData) => {
      const response = await api.patch<{ data: any }>('/assets/bulk-update', data);
      return (response as any).data;
    },
    onSuccess: (data) => {
      message.success(`Successfully updated ${data.updated_count} assets`);
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      setBulkUpdateModalVisible(false);
      setSelectedAssets([]);
      bulkForm.resetFields();
    },
    onError: (error: any) => {
      message.error(`Failed to update assets: ${error.response?.data?.error || error.message}`);
    },
  });

  // Import assets mutation
  const importAssetsMutation = useMutation({
    mutationFn: async (formData: FormData) => {
      const response = await api.post<ImportProgress>('/assets/import', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      return response;
    },
    onSuccess: (data: ImportProgress) => {
      setImportProgress(data);
      message.success(`Import completed! Processed ${data.processed}/${data.total} assets`);
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      setUploading(false);
    },
    onError: (error: any) => {
      message.error(`Import failed: ${error.response?.data?.error || error.message}`);
      setUploading(false);
      setImportProgress(null);
    },
  });

  const handleBulkUpdate = (values: any) => {
    if (selectedAssets.length === 0) {
      message.warning('Please select at least one asset');
      return;
    }

    const updates: any = {};
    if (values.status) updates.status = values.status;
    if (values.condition) updates.condition = values.condition;
    if (values.location) updates.location = values.location;
    if (values.category_id) updates.category_id = values.category_id;

    if (Object.keys(updates).length === 0) {
      message.warning('Please specify at least one field to update');
      return;
    }

    bulkUpdateMutation.mutate({
      asset_ids: selectedAssets,
      updates,
      notes: values.notes,
    });
  };

  const handleExport = async () => {
    try {
      // Manually fetch as blob since fetch does not support responseType
      const params = new URLSearchParams({ format: 'csv' });
      const res = await fetch(`/assets/export?${params.toString()}`, {
        method: 'GET',
        // Add authentication headers here if needed, e.g.:
        // headers: { Authorization: `Bearer ${yourToken}` },
      });
      if (!res.ok) throw new Error('Failed to export assets');
      const blob = await res.blob();

      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `assets-export-${new Date().toISOString().split('T')[0]}.csv`);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      message.success('Assets exported successfully');
    } catch (error: any) {
      message.error(`Export failed: ${error.response?.data?.error || error.message}`);
    }
  };

  const uploadProps = {
    name: 'file',
    accept: '.csv,.xlsx',
    beforeUpload: (file: File) => {
      const isValidType = file.type === 'text/csv' || 
                         file.type === 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
      if (!isValidType) {
        message.error('Please upload a CSV or Excel file');
        return false;
      }
      return false; // Prevent automatic upload
    },
    customRequest: ({ file, onSuccess }: any) => {
      // Handle custom upload
      const formData = new FormData();
      formData.append('file', file);
      importAssetsMutation.mutate(formData);
      onSuccess?.();
    },
  };

  const rowSelection = {
    selectedRowKeys: selectedAssets,
    onChange: (selectedRowKeys: React.Key[]) => {
      setSelectedAssets(selectedRowKeys as string[]);
    },
  };

  const columns = [
    {
      title: 'Asset Tag',
      dataIndex: 'asset_tag',
      key: 'asset_tag',
    },
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
    },
    {
      title: 'Condition',
      dataIndex: 'condition',
      key: 'condition',
    },
    {
      title: 'Location',
      dataIndex: 'location',
      key: 'location',
    },
  ];

  return (
    <div>
      <h2>Bulk Operations</h2>
      
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* Import/Export Section */}
        <Card title="Import/Export Assets">
          <Space>
            <Upload {...uploadProps}>
              <Button icon={<UploadOutlined />} loading={uploading}>
                Import Assets
              </Button>
            </Upload>
            <Button icon={<DownloadOutlined />} onClick={handleExport}>
              Export Assets
            </Button>
          </Space>

          {importProgress && (
            <div style={{ marginTop: 16 }}>
              <Progress 
                percent={Math.round((importProgress.processed / importProgress.total) * 100)}
                status={importProgress.errors.length > 0 ? 'exception' : 'success'}
              />
              <div style={{ marginTop: 8 }}>
                Processed: {importProgress.processed}/{importProgress.total}
              </div>
              
              {importProgress.errors.length > 0 && (
                <Alert
                  message={`${importProgress.errors.length} errors occurred`}
                  description={
                    <div>
                      {importProgress.errors.slice(0, 5).map((error, index) => (
                        <div key={index}>Row {error.row}: {error.error}</div>
                      ))}
                      {importProgress.errors.length > 5 && <div>... and {importProgress.errors.length - 5} more</div>}
                    </div>
                  }
                  type="error"
                  style={{ marginTop: 8 }}
                />
              )}

              {importProgress.warnings.length > 0 && (
                <Alert
                  message={`${importProgress.warnings.length} warnings`}
                  description={
                    <div>
                      {importProgress.warnings.slice(0, 3).map((warning, index) => (
                        <div key={index}>Row {warning.row}: {warning.warning}</div>
                      ))}
                      {importProgress.warnings.length > 3 && <div>... and {importProgress.warnings.length - 3} more</div>}
                    </div>
                  }
                  type="warning"
                  style={{ marginTop: 8 }}
                />
              )}
            </div>
          )}
        </Card>

        {/* Bulk Update Section */}
        <Card 
          title="Bulk Update Assets"
          extra={
            <Button 
              type="primary" 
              disabled={selectedAssets.length === 0}
              onClick={() => setBulkUpdateModalVisible(true)}
            >
              Update Selected ({selectedAssets.length})
            </Button>
          }
        >
          <Table
            dataSource={assets}
            columns={columns}
            rowKey="id"
            rowSelection={rowSelection}
            loading={isLoading}
            pagination={{ pageSize: 20 }}
            scroll={{ y: 400 }}
          />
        </Card>
      </Space>

      {/* Bulk Update Modal */}
      <Modal
        title={`Update ${selectedAssets.length} Selected Assets`}
        open={bulkUpdateModalVisible}
        onCancel={() => setBulkUpdateModalVisible(false)}
        onOk={() => bulkForm.submit()}
        confirmLoading={bulkUpdateMutation.isPending}
        width={600}
      >
        <Alert
          message="Bulk Update Warning"
          description="This will update all selected assets. Empty fields will be ignored."
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form
          form={bulkForm}
          layout="vertical"
          onFinish={handleBulkUpdate}
        >
          <Form.Item name="status" label="Status">
            <Select placeholder="Select new status (leave empty to skip)" allowClear>
              <Option value="active">Active</Option>
              <Option value="inactive">Inactive</Option>
              <Option value="maintenance">Maintenance</Option>
              <Option value="retired">Retired</Option>
              <Option value="disposed">Disposed</Option>
            </Select>
          </Form.Item>

          <Form.Item name="condition" label="Condition">
            <Select placeholder="Select new condition (leave empty to skip)" allowClear>
              <Option value="excellent">Excellent</Option>
              <Option value="good">Good</Option>
              <Option value="fair">Fair</Option>
              <Option value="poor">Poor</Option>
              <Option value="broken">Broken</Option>
            </Select>
          </Form.Item>

          <Form.Item name="location" label="Location">
            <Input placeholder="New location (leave empty to skip)" />
          </Form.Item>

          <Form.Item name="category_id" label="Category">
            <Select placeholder="Select new category (leave empty to skip)" allowClear>
              {categories?.map((category: any) => (
                <Option key={category.id} value={category.id}>
                  {category.name}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="notes" label="Update Notes">
            <TextArea rows={3} placeholder="Reason for bulk update..." />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}