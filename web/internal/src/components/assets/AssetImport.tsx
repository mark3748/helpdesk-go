import { useState } from 'react';
import { Card, Upload, Button, Table, Progress, Alert, Steps, Typography, Space, Divider } from 'antd';
import { UploadOutlined, DownloadOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../shared/api';

const { Title, Paragraph, Text } = Typography;
const { Step } = Steps;

interface ImportResult {
  total_rows: number;
  successful_imports: number;
  failed_imports: number;
  warnings: Array<{
    row: number;
    message: string;
  }>;
  errors: Array<{
    row: number;
    message: string;
  }>;
  imported_assets: Array<{
    asset_tag: string;
    name: string;
    status: string;
  }>;
}

interface ValidationResult {
  valid: boolean;
  errors: Array<{
    row: number;
    column: string;
    message: string;
  }>;
  warnings: Array<{
    row: number;
    column: string;
    message: string;
  }>;
  preview: Array<Record<string, any>>;
}

export default function AssetImport() {
  const [currentStep, setCurrentStep] = useState(0);
  const [uploadedFile, setUploadedFile] = useState<File | null>(null);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [uploading, setUploading] = useState(false);
  const queryClient = useQueryClient();

  // Validate file mutation
  const validateMutation = useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      const response = await api.post<ValidationResult>('/assets/import/validate', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      return response;
    },
    onSuccess: (result) => {
      setValidationResult(result);
      setCurrentStep(1);
      setUploading(false);
    },
    onError: (error: any) => {
      console.error('Validation failed:', error);
      setUploading(false);
    },
  });

  // Import assets mutation
  const importMutation = useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      const response = await api.post<ImportResult>('/assets/import', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      return response;
    },
    onSuccess: (result) => {
      setImportResult(result);
      setCurrentStep(2);
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      setUploading(false);
    },
    onError: (error: any) => {
      console.error('Import failed:', error);
      setUploading(false);
    },
  });

  const handleFileUpload = (file: File) => {
    setUploadedFile(file);
    setUploading(true);
    validateMutation.mutate(file);
    return false; // Prevent automatic upload
  };

  const handleImport = () => {
    if (uploadedFile) {
      setUploading(true);
      importMutation.mutate(uploadedFile);
    }
  };

  const handleReset = () => {
    setCurrentStep(0);
    setUploadedFile(null);
    setValidationResult(null);
    setImportResult(null);
  };

  const downloadTemplate = () => {
    // Create a sample CSV template
    const headers = ['asset_tag', 'name', 'description', 'category', 'status', 'condition', 'manufacturer', 'model', 'serial_number', 'location', 'purchase_price', 'purchase_date'];
    const sampleData = [
      'LP-001,Dell Laptop,Dell Latitude 7420,Laptops,active,good,Dell,Latitude 7420,ABC123,Office Floor 3,1200.00,2023-01-15',
      'CH-001,Office Chair,Ergonomic office chair,Furniture,active,excellent,Herman Miller,Aeron,XYZ789,Office Floor 2,800.00,2023-02-01'
    ];
    
    const csvContent = [headers.join(','), ...sampleData].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', 'asset-import-template.csv');
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  };

  const validationColumns = [
    {
      title: 'Row',
      dataIndex: 'row',
      key: 'row',
      width: 80,
    },
    {
      title: 'Column',
      dataIndex: 'column',
      key: 'column',
      width: 120,
    },
    {
      title: 'Message',
      dataIndex: 'message',
      key: 'message',
    },
  ];

  const previewColumns = validationResult?.preview.length > 0 
    ? Object.keys(validationResult.preview[0]).map(key => ({
        title: key,
        dataIndex: key,
        key,
        ellipsis: true,
      }))
    : [];

  const resultColumns = [
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
  ];

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title level={2}>Asset Import</Title>
        <Paragraph>
          Import multiple assets from a CSV or Excel file. Download the template to see the required format.
        </Paragraph>
      </div>

      <Steps current={currentStep} style={{ marginBottom: 24 }}>
        <Step title="Upload File" description="Select and validate your import file" />
        <Step title="Review" description="Check validation results and preview data" />
        <Step title="Complete" description="Import results and summary" />
      </Steps>

      {currentStep === 0 && (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card title="Step 1: Prepare Your File">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Paragraph>
                Before uploading, make sure your file follows the correct format:
              </Paragraph>
              <ul>
                <li>CSV or Excel (.xlsx) format</li>
                <li>First row should contain column headers</li>
                <li>Required columns: asset_tag, name</li>
                <li>Optional columns: description, category, status, condition, manufacturer, model, serial_number, location, purchase_price, purchase_date</li>
              </ul>
              
              <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>
                Download Template
              </Button>
            </Space>
          </Card>

          <Card title="Step 2: Upload File">
            <Upload.Dragger
              accept=".csv,.xlsx"
              beforeUpload={handleFileUpload}
              showUploadList={false}
              disabled={uploading}
            >
              <p className="ant-upload-drag-icon">
                <UploadOutlined />
              </p>
              <p className="ant-upload-text">Click or drag file to this area to upload</p>
              <p className="ant-upload-hint">
                Support for CSV and Excel files. The file will be validated before import.
              </p>
            </Upload.Dragger>
            
            {uploading && (
              <div style={{ marginTop: 16, textAlign: 'center' }}>
                <Progress type="circle" percent={50} />
                <div style={{ marginTop: 8 }}>Validating file...</div>
              </div>
            )}
          </Card>
        </Space>
      )}

      {currentStep === 1 && validationResult && (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card 
            title="Validation Results"
            extra={
              <Space>
                <Button onClick={handleReset}>Upload Different File</Button>
                <Button 
                  type="primary" 
                  onClick={handleImport}
                  disabled={!validationResult.valid || uploading}
                  loading={uploading}
                >
                  Proceed with Import
                </Button>
              </Space>
            }
          >
            {validationResult.valid ? (
              <Alert
                message="File validation passed"
                description="Your file is ready for import."
                type="success"
                icon={<CheckCircleOutlined />}
                style={{ marginBottom: 16 }}
              />
            ) : (
              <Alert
                message="File validation failed"
                description="Please fix the errors below before proceeding."
                type="error"
                icon={<ExclamationCircleOutlined />}
                style={{ marginBottom: 16 }}
              />
            )}

            {validationResult.errors.length > 0 && (
              <div style={{ marginBottom: 16 }}>
                <Title level={4}>Errors</Title>
                <Table
                  dataSource={validationResult.errors}
                  columns={validationColumns}
                  pagination={false}
                  size="small"
                />
              </div>
            )}

            {validationResult.warnings.length > 0 && (
              <div style={{ marginBottom: 16 }}>
                <Title level={4}>Warnings</Title>
                <Table
                  dataSource={validationResult.warnings}
                  columns={validationColumns}
                  pagination={false}
                  size="small"
                />
              </div>
            )}
          </Card>

          {validationResult.preview.length > 0 && (
            <Card title="Data Preview">
              <Table
                dataSource={validationResult.preview.slice(0, 10)}
                columns={previewColumns}
                pagination={false}
                scroll={{ x: 1000 }}
                size="small"
              />
              {validationResult.preview.length > 10 && (
                <div style={{ marginTop: 8, textAlign: 'center' }}>
                  <Text type="secondary">Showing first 10 rows of {validationResult.preview.length} total rows</Text>
                </div>
              )}
            </Card>
          )}
        </Space>
      )}

      {currentStep === 2 && importResult && (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card 
            title="Import Complete"
            extra={<Button type="primary" onClick={handleReset}>Import Another File</Button>}
          >
            <div style={{ marginBottom: 16 }}>
              <Progress
                percent={Math.round((importResult.successful_imports / importResult.total_rows) * 100)}
                status={importResult.failed_imports > 0 ? 'exception' : 'success'}
              />
            </div>

            <Space size="large">
              <div>
                <div style={{ fontSize: 24, fontWeight: 'bold', color: '#52c41a' }}>
                  {importResult.successful_imports}
                </div>
                <div>Successful</div>
              </div>
              <div>
                <div style={{ fontSize: 24, fontWeight: 'bold', color: '#ff4d4f' }}>
                  {importResult.failed_imports}
                </div>
                <div>Failed</div>
              </div>
              <div>
                <div style={{ fontSize: 24, fontWeight: 'bold' }}>
                  {importResult.total_rows}
                </div>
                <div>Total Rows</div>
              </div>
            </Space>

            {importResult.errors.length > 0 && (
              <div style={{ marginTop: 24 }}>
                <Title level={4}>Import Errors</Title>
                <Table
                  dataSource={importResult.errors}
                  columns={[
                    { title: 'Row', dataIndex: 'row', key: 'row', width: 80 },
                    { title: 'Error', dataIndex: 'message', key: 'message' },
                  ]}
                  pagination={false}
                  size="small"
                />
              </div>
            )}

            {importResult.warnings.length > 0 && (
              <div style={{ marginTop: 24 }}>
                <Title level={4}>Import Warnings</Title>
                <Table
                  dataSource={importResult.warnings}
                  columns={[
                    { title: 'Row', dataIndex: 'row', key: 'row', width: 80 },
                    { title: 'Warning', dataIndex: 'message', key: 'message' },
                  ]}
                  pagination={false}
                  size="small"
                />
              </div>
            )}
          </Card>

          {importResult.imported_assets.length > 0 && (
            <Card title="Successfully Imported Assets">
              <Table
                dataSource={importResult.imported_assets}
                columns={resultColumns}
                pagination={{ pageSize: 10 }}
                size="small"
              />
            </Card>
          )}
        </Space>
      )}
    </div>
  );
}