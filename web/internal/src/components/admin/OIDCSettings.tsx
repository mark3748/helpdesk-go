import { useEffect } from 'react';
import { Form, Input, Button, message } from 'antd';
import { useSettings, useTestConnection } from '../../api';
import { apiFetch } from '../../../../shared/api';

export default function OIDCSettings() {
  const [form] = Form.useForm();
  const { data } = useSettings();
  const test = useTestConnection();

  useEffect(() => {
    if ((data as any)?.oidc) {
      form.setFieldsValue((data as any).oidc);
    }
  }, [data, form]);

  const onFinish = async (values: any) => {
    await apiFetch('/settings/oidc', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(values),
    });
    message.success('OIDC settings saved');
  };

  return (
    <>
      <p>Log directory: {(data as any)?.log_path}</p>
      <p>Last test: {(data as any)?.last_test || 'never'}</p>
      <Button
        onClick={() =>
          test.mutate(undefined, {
            onSuccess: () => message.success('Test complete'),
            onError: () => message.error('Test failed'),
          })
        }
        loading={test.isPending}
        style={{ marginBottom: 16 }}
      >
        Test Connection
      </Button>
      <Form form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item label="Issuer" name="issuer">
          <Input />
        </Form.Item>
        <Form.Item label="Client ID" name="client_id">
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
