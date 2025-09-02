import { useEffect } from 'react';
import { Form, Input, Button, message } from 'antd';
import { useSettings, useSaveMailSettings, useTestConnection } from '../../api';

export default function MailSettings() {
  const [form] = Form.useForm();
  const { data } = useSettings();
  const save = useSaveMailSettings();
  const test = useTestConnection();

  useEffect(() => {
    if ((data as any)?.mail) {
      form.setFieldsValue((data as any).mail);
    }
  }, [data, form]);

  const onFinish = (values: any) => {
    save.mutate(values, {
      onSuccess: () => message.success('Email settings saved'),
      onError: () => message.error('Failed to save'),
    });
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
      <Form.Item label="SMTP Host" name="smtp_host">
        <Input />
      </Form.Item>
      <Form.Item label="IMAP Host" name="imap_host">
        <Input />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" loading={save.isPending}>
          Save
        </Button>
      </Form.Item>
    </Form>
    </>
  );
}
