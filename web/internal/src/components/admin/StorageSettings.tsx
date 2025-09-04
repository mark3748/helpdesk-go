import { useEffect } from 'react';
import { Form, Input, Button, message } from 'antd';
import { useSettings } from '../../api';
import { apiFetch } from '../../shared/api';

export default function StorageSettings() {
  const [form] = Form.useForm();
  const { data } = useSettings();

  useEffect(() => {
    if ((data as any)?.storage) {
      form.setFieldsValue((data as any).storage);
    }
  }, [data, form]);

  const onFinish = async (values: any) => {
    await apiFetch('/settings/storage', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(values),
    });
    message.success('Storage settings saved');
  };

  return (
    <>
      <p>Log directory: {(data as any)?.log_path}</p>
      <p>Last test: {(data as any)?.last_test || 'never'}</p>
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item label="Endpoint" name="endpoint">
          <Input />
        </Form.Item>
        <Form.Item label="Bucket" name="bucket">
          <Input />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit">
            Save
          </Button>
        </Form.Item>
      </Form>
    </>
  );
}
