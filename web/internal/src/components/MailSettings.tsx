import React from 'react';
import { Form, Input, Button, message } from 'antd';
import { useSettings, useSaveMailSettings } from '../api';

export default function MailSettings() {
  const { data } = useSettings();
  const save = useSaveMailSettings();

  const onFinish = (values: any) => {
    save.mutate(values, {
      onSuccess: () => message.success('Email settings saved'),
      onError: () => message.error('Failed to save'),
    });
  };

  return (
    <Form layout="vertical" onFinish={onFinish} initialValues={(data as any)?.mail}>
      <Form.Item label="SMTP Host" name="smtp_host">
        <Input />
      </Form.Item>
      <Form.Item label="IMAP Host" name="imap_host">
        <Input />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" loading={save.isLoading}>
          Save
        </Button>
      </Form.Item>
    </Form>
  );
}
