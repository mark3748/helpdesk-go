import { useEffect } from 'react';
import { Form, Input, Button, message, Checkbox } from 'antd';
import { useSettings } from '../../api';
import { apiFetch } from '../../shared/api';

export default function StorageSettings() {
  const [form] = Form.useForm();
  const { data } = useSettings();

  useEffect(() => {
    if ((data as any)?.storage) {
      const storage = (data as any).storage;
      // Convert string 'true'/'false' to boolean for Checkbox if needed, or keep as is if using Select/Radio.
      // Assuming storage stores strings.
      form.setFieldsValue({
        ...storage,
        use_ssl: storage.use_ssl === 'true',
        path_style: storage.path_style === 'true',
      });
    }
  }, [data, form]);

  const onFinish = async (values: any) => {
    // Convert boolean to string for backend map[string]string
    const payload = { 
      ...values, 
      use_ssl: String(values.use_ssl ?? false),
      path_style: String(values.path_style ?? false) 
    };
    await apiFetch('/settings/storage', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    message.success('Storage settings saved');
  };

  const onTest = async () => {
    try {
      const values = await form.validateFields();
      const payload = { 
        ...values, 
        use_ssl: String(values.use_ssl ?? false),
        path_style: String(values.path_style ?? false)
      };
      const res = await apiFetch<any>('/settings/storage/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        message.success('Connection successful');
      } else {
        message.error(res.error || 'Connection failed');
      }
    } catch (err: any) {
      message.error(err.message || 'Test failed');
    }
  };

  return (
    <>
      <p>Log directory: {(data as any)?.log_path}</p>
      <p>Last test: {(data as any)?.last_test || 'never'}</p>
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item label="Endpoint" name="endpoint" rules={[{ required: true }]}>
          <Input placeholder="s3.amazonaws.com or minio.local:9000" />
        </Form.Item>
        <Form.Item label="Region" name="region">
          <Input placeholder="us-east-1" />
        </Form.Item>
        <Form.Item label="Bucket" name="bucket" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Access Key ID" name="access_key_id" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Secret Access Key" name="secret_access_key" rules={[{ required: true }]}>
          <Input.Password />
        </Form.Item>
        <div style={{ display: 'flex', gap: '24px' }}>
          <Form.Item name="use_ssl" valuePropName="checked">
            <Checkbox>Use SSL</Checkbox>
          </Form.Item>
          <Form.Item name="path_style" valuePropName="checked">
            <Checkbox>Force Path Style (recommended for Garage/Custom S3)</Checkbox>
          </Form.Item>
        </div>
        <Form.Item>
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button type="primary" htmlType="submit">
              Save
            </Button>
            <Button onClick={onTest}>
              Test Connection
            </Button>
          </div>
        </Form.Item>
      </Form>
    </>
  );
}
